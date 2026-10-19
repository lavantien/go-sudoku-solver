package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/server"
	"github.com/lavantien/go-sudoku-solver/internal/store"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
	"go.uber.org/goleak"
)

type methodJSON struct {
	Method     string  `json:"method"`
	Grid81     string  `json:"grid81"`
	Steps      int     `json:"steps"`
	TimeNS     int64   `json:"time_ns"`
	DurationMS float64 `json:"duration_ms"`
	Solved     bool    `json:"solved"`
}

type runSummaryJSON struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
	Input81   string `json:"input81"`
	Source    string `json:"source"`
	SolvedAny bool   `json:"solved_any"`
}

type solveJSON struct {
	ID      int64        `json:"id"`
	Input81 string       `json:"input81"`
	Source  string       `json:"source"`
	Methods []methodJSON `json:"methods"`
}

type runDetailJSON struct {
	runSummaryJSON
	Methods []methodJSON `json:"methods"`
}

type runsJSON struct {
	Runs []runSummaryJSON `json:"runs"`
}

type puzzleJSON struct {
	ID        int64  `json:"id"`
	Grid81    string `json:"grid81"`
	Clues     int    `json:"clues"`
	Seed      int64  `json:"seed"`
	CreatedAt string `json:"created_at"`
}

type puzzlesJSON struct {
	Puzzles []puzzleJSON `json:"puzzles"`
}

type healthJSON struct {
	Status string `json:"status"`
}

type versionJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type errorJSON struct {
	Error string `json:"error"`
}

func openStore(t *testing.T, path string) *store.Store {
	t.Helper()
	st, err := store.Open(path, config.DBBusyTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func newServer(t *testing.T) (*server.Server, *store.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	st := openStore(t, path)
	cfg := config.Defaults()
	return server.New(context.Background(), cfg, st, logging.New("")), st
}

func startSite(t *testing.T, h http.Handler) (*httptest.Server, *http.Client) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	client := &http.Client{Timeout: 60 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	return ts, client
}

func getBody(t *testing.T, c *http.Client, rawurl string) (int, string) {
	t.Helper()
	resp, err := c.Get(rawurl)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	if _, err := io.Copy(&b, resp.Body); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b.String()
}

func decodeJSON(t *testing.T, c *http.Client, rawurl string, into any) (int, string) {
	t.Helper()
	resp, err := c.Get(rawurl)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var raw strings.Builder
	if _, err := io.Copy(&raw, resp.Body); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw.String()), into); err != nil {
		t.Fatalf("decode %s: %v", rawurl, err)
	}
	return resp.StatusCode, raw.String()
}

func postJSON(t *testing.T, c *http.Client, rawurl string, body string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Post(rawurl, config.JSONContentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	if _, err := io.Copy(&b, resp.Body); err != nil {
		t.Fatal(err)
	}
	return resp, b.String()
}

func postForm(t *testing.T, c *http.Client, rawurl string, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(rawurl, form)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	if _, err := io.Copy(&b, resp.Body); err != nil {
		t.Fatal(err)
	}
	return resp, b.String()
}

func decodeInto(t *testing.T, raw string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
}

func nineLine(grid81 string) string {
	rows := make([]string, 0, config.Size)
	for r := range config.Size {
		rows = append(rows, grid81[r*config.Size:(r+1)*config.Size])
	}
	return strings.Join(rows, "\r\n")
}

func pipedRows(grid81 string) string {
	var b strings.Builder
	for r := range config.Size {
		row := grid81[r*config.Size : (r+1)*config.Size]
		b.WriteString(row[0:config.Box] + "|" + row[config.Box:2*config.Box] + "|" + row[2*config.Box:])
	}
	return b.String()
}

var (
	tableRe = regexp.MustCompile(`(?s)<table class="grid">(.*?)</table>`)
	cellRe  = regexp.MustCompile(`<td[^>]*>([^<]*)</td>`)
)

func gridTableTexts(body string) []string {
	texts := []string{}
	for _, m := range tableRe.FindAllStringSubmatch(body, -1) {
		var b strings.Builder
		for _, c := range cellRe.FindAllStringSubmatch(m[1], -1) {
			b.WriteString(c[1])
		}
		texts = append(texts, b.String())
	}
	return texts
}

func TestSeedRunPersistsOncePerDatabase(t *testing.T) {
	defer goleak.VerifyNone(t)
	path := filepath.Join(t.TempDir(), "seed.db")
	cfg := config.Defaults()
	st := openStore(t, path)
	server.New(context.Background(), cfg, st, logging.New(""))
	n, err := st.CountRuns()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("seed runs: got %d want 1", n)
	}
	detail, err := st.GetRun(1)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Input81 != puzzle.Sample81 {
		t.Errorf("seed input: got %q want the classic sample", detail.Input81)
	}
	if detail.Source != config.SourceSeed {
		t.Errorf("seed source: got %q want %q", detail.Source, config.SourceSeed)
	}
	if len(detail.Methods) != len(config.MethodIDs) {
		t.Errorf("seed methods: got %d want %d", len(detail.Methods), len(config.MethodIDs))
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st2 := openStore(t, path)
	server.New(context.Background(), cfg, st2, logging.New(""))
	n, err = st2.CountRuns()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("runs after restart: got %d want 1", n)
	}
	if err := st2.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHomePageShowsSamplesAndForm(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	code, body := getBody(t, c, ts.URL+config.HomePath)
	if code != http.StatusOK {
		t.Fatalf("home status: got %d want 200", code)
	}
	if !strings.HasPrefix(body, "<!DOCTYPE html>") {
		t.Errorf("home lacks doctype")
	}
	for _, s := range puzzle.Samples() {
		if !strings.Contains(body, s.Label) {
			t.Errorf("home lacks sample label %q", s.Label)
		}
		if !strings.Contains(body, s.Grid81[:config.Size]) {
			t.Errorf("home lacks first row of sample %q", s.Label)
		}
	}
	if !strings.Contains(body, config.FragmentSolvePath) {
		t.Errorf("home lacks fragment solve target")
	}
	if !strings.Contains(body, `id="results"`) {
		t.Errorf("home lacks results target div")
	}
	if !strings.Contains(body, config.StaticMountPath+"htmx.min.js") {
		t.Errorf("home lacks vendored htmx include")
	}
}

func TestStaticAssetsServed(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	code, css := getBody(t, c, ts.URL+config.StaticMountPath+"app.css")
	if code != http.StatusOK || !strings.Contains(css, "system-ui") {
		t.Errorf("app.css: status %d, system-ui stack present %v", code, strings.Contains(css, "system-ui"))
	}
	code, js := getBody(t, c, ts.URL+config.StaticMountPath+"app.js")
	if code != http.StatusOK || !strings.Contains(js, "grid-input") {
		t.Errorf("app.js: status %d, grid-input hook present %v", code, strings.Contains(js, "grid-input"))
	}
	code, htmx := getBody(t, c, ts.URL+config.StaticMountPath+"htmx.min.js")
	if code != http.StatusOK || !strings.Contains(htmx, "htmx") {
		t.Errorf("htmx.min.js: status %d, payload present %v", code, strings.Contains(htmx, "htmx"))
	}
}

func solvedTableCount(body string) int {
	want := pipedRows(puzzle.SampleSolution81)
	n := 0
	for _, txt := range gridTableTexts(body) {
		if txt == want {
			n++
		}
	}
	return n
}

func TestSolveFragmentAcceptsNineLineTextarea(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FormGrid81: {nineLine(puzzle.Sample81)}}
	resp, body := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fragment solve status: got %d want 200", resp.StatusCode)
	}
	for _, label := range []string{config.LabelPropagate, config.LabelCover, config.LabelAnneal, config.LabelRace} {
		if !strings.Contains(body, label) {
			t.Errorf("fragment lacks method label %q", label)
		}
	}
	if got := len(gridTableTexts(body)); got != len(config.MethodIDs) {
		t.Errorf("fragment grid tables: got %d want %d", got, len(config.MethodIDs))
	}
	if n := solvedTableCount(body); n != 3 {
		t.Errorf("fragment solved tables: got %d want 3", n)
	}
	if !strings.Contains(body, "solved") {
		t.Errorf("fragment lacks solved badge")
	}
}

func TestSolveFragmentAcceptsStrictGrid81(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FormGrid81: {puzzle.Sample81}}
	resp, body := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fragment solve status: got %d want 200", resp.StatusCode)
	}
	if n := solvedTableCount(body); n != 3 {
		t.Errorf("fragment solved tables: got %d want 3", n)
	}
}

func TestSolveFragmentRejectsInvalidGrid(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FormGrid81: {"not a grid"}}
	resp, body := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid grid status: got %d want 400", resp.StatusCode)
	}
	if !strings.Contains(body, config.ErrorKey) && !strings.Contains(body, "invalid") {
		t.Errorf("invalid grid body lacks error cue: %q", body)
	}
}

func TestGenerateFragmentProducesPuzzle(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FlagClues: {"40"}, config.FlagSeed: {"7"}}
	resp, body := postForm(t, c, ts.URL+config.FragmentGenPath, form)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fragment generate status: got %d want 200", resp.StatusCode)
	}
	if len(gridTableTexts(body)) != 1 {
		t.Fatalf("fragment generate grid tables: got %d want 1", len(gridTableTexts(body)))
	}
	puzzles, err := st.ListPuzzles(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(puzzles) != 1 || puzzles[0].Seed != 7 {
		t.Fatalf("persisted puzzles: got %+v want one entry with seed 7", puzzles)
	}
	if _, err := sudoku.Parse81(puzzles[0].Grid81); err != nil {
		t.Fatalf("persisted puzzle does not re-parse: %v", err)
	}
}

func TestAPISolveReturnsRunAndPersists(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	req := fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, puzzle.Sample81)
	resp, raw := postJSON(t, c, ts.URL+config.APISolvePath, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api solve status: got %d want 200: %s", resp.StatusCode, raw)
	}
	var out solveJSON
	decodeInto(t, raw, &out)
	if out.ID == 0 {
		t.Fatalf("api solve run id: got %d want nonzero", out.ID)
	}
	if out.Input81 != puzzle.Sample81 {
		t.Errorf("api solve input: got %q want the classic sample", out.Input81)
	}
	if out.Source != config.SourceAPI {
		t.Errorf("api solve source: got %q want %q", out.Source, config.SourceAPI)
	}
	if len(out.Methods) != len(config.MethodIDs) {
		t.Fatalf("api solve methods: got %d want %d", len(out.Methods), len(config.MethodIDs))
	}
	exact := map[string]bool{config.MethodIDs[0]: false, config.MethodIDs[1]: false, config.MethodIDs[3]: false}
	for _, m := range out.Methods {
		if _, ok := exact[m.Method]; ok {
			if m.Grid81 != puzzle.SampleSolution81 {
				t.Errorf("api solve %s grid: got %q want the sample solution", m.Method, m.Grid81)
			}
			if !m.Solved {
				t.Errorf("api solve %s solved: got false want true", m.Method)
			}
			exact[m.Method] = true
		}
	}
	for m, seen := range exact {
		if !seen {
			t.Errorf("api solve methods lack %s", m)
		}
	}
	detail, err := st.GetRun(out.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Source != config.SourceAPI || len(detail.Methods) != len(config.MethodIDs) {
		t.Errorf("persisted run: source %q methods %d", detail.Source, len(detail.Methods))
	}
}

func TestAPISolveRejectsInvalidInput(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	cases := []struct {
		name string
		body string
	}{
		{"bad grid", fmt.Sprintf(`{"%s": "12"}`, config.FormGrid81)},
		{"bad char", fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, strings.Repeat("a", config.Cells))},
		{"missing field", `{}`},
		{"malformed json", "{not json"},
	}
	for _, tc := range cases {
		resp, raw := postJSON(t, c, ts.URL+config.APISolvePath, tc.body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s status: got %d want 400: %s", tc.name, resp.StatusCode, raw)
		}
		var e errorJSON
		decodeInto(t, raw, &e)
		if e.Error == "" {
			t.Errorf("%s error envelope empty: %q", tc.name, raw)
		}
	}
}

func TestAPIRunsPagination(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	for i := range 3 {
		if _, err := st.InsertRun(puzzle.Sample81, "test", []store.MethodRow{{Method: config.MethodIDs[0], Grid81: puzzle.SampleSolution81, Steps: i + 1, TimeNS: 1, Solved: true}}); err != nil {
			t.Fatal(err)
		}
	}
	var first runsJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIRunsPath+"?"+config.QueryLimit+"=2&"+config.QueryOffset+"=0", &first)
	if code != http.StatusOK {
		t.Fatalf("runs page 1 status: got %d want 200", code)
	}
	if len(first.Runs) != 2 {
		t.Fatalf("runs page 1 size: got %d want 2", len(first.Runs))
	}
	if first.Runs[0].ID != 4 || first.Runs[1].ID != 3 {
		t.Errorf("runs page 1 ids: got %d,%d want 4,3", first.Runs[0].ID, first.Runs[1].ID)
	}
	var second runsJSON
	code, _ = decodeJSON(t, c, ts.URL+config.APIRunsPath+"?"+config.QueryLimit+"=2&"+config.QueryOffset+"=2", &second)
	if code != http.StatusOK {
		t.Fatalf("runs page 2 status: got %d want 200", code)
	}
	if len(second.Runs) != 2 {
		t.Fatalf("runs page 2 size: got %d want 2", len(second.Runs))
	}
	if second.Runs[1].Source != config.SourceSeed {
		t.Errorf("runs page 2 oldest source: got %q want %q", second.Runs[1].Source, config.SourceSeed)
	}
	code, raw := getBody(t, c, ts.URL+config.APIRunsPath+"?"+config.QueryLimit+"=x")
	if code != http.StatusBadRequest {
		t.Errorf("runs malformed limit status: got %d want 400: %s", code, raw)
	}
}

func TestAPIRunDetailAndNotFound(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	var seedRun runsJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIRunsPath, &seedRun)
	if code != http.StatusOK || len(seedRun.Runs) != 1 {
		t.Fatalf("runs list: status %d len %d", code, len(seedRun.Runs))
	}
	id := seedRun.Runs[0].ID
	var detail runDetailJSON
	code, _ = decodeJSON(t, c, ts.URL+fmt.Sprintf(config.APIRunPathFmt, id), &detail)
	if code != http.StatusOK {
		t.Fatalf("run detail status: got %d want 200", code)
	}
	if detail.Input81 != puzzle.Sample81 || detail.Source != config.SourceSeed {
		t.Errorf("run detail: input/source mismatch: %q %q", detail.Input81, detail.Source)
	}
	if len(detail.Methods) != len(config.MethodIDs) {
		t.Errorf("run detail methods: got %d want %d", len(detail.Methods), len(config.MethodIDs))
	}
	var e errorJSON
	code, _ = decodeJSON(t, c, ts.URL+fmt.Sprintf(config.APIRunPathFmt, 999999), &e)
	if code != http.StatusNotFound || e.Error == "" {
		t.Errorf("unknown run: status %d envelope %q", code, e.Error)
	}
	code, _ = decodeJSON(t, c, ts.URL+config.APIRunsPrefix+"abc", &e)
	if code != http.StatusNotFound {
		t.Errorf("non numeric run: status %d want 404", code)
	}
}

func TestAPIGenerateRoundTrip(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	body := `{"clues": 40, "seed": 7}`
	resp, raw := postJSON(t, c, ts.URL+config.APIGeneratePath, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("api generate status: got %d want 201: %s", resp.StatusCode, raw)
	}
	var gen puzzleJSON
	decodeInto(t, raw, &gen)
	if gen.ID == 0 {
		t.Fatalf("api generate id: got %d want nonzero", gen.ID)
	}
	if gen.Seed != 7 {
		t.Errorf("api generate seed: got %d want 7", gen.Seed)
	}
	if len(gen.Grid81) != config.Cells {
		t.Fatalf("api generate grid length: got %d want %d", len(gen.Grid81), config.Cells)
	}
	if _, err := sudoku.Parse81(gen.Grid81); err != nil {
		t.Fatalf("api generate grid does not parse: %v", err)
	}
	if gen.Clues < 17 || gen.Clues > config.Cells {
		t.Errorf("api generate clues out of range: got %d", gen.Clues)
	}
	var list puzzlesJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIPuzzlesPath, &list)
	if code != http.StatusOK {
		t.Fatalf("api puzzles status: got %d want 200", code)
	}
	if len(list.Puzzles) != 1 || list.Puzzles[0].ID != gen.ID || list.Puzzles[0].Seed != 7 {
		t.Fatalf("api puzzles list: got %+v want one entry with id %d seed 7", list.Puzzles, gen.ID)
	}
	puzzles, err := st.ListPuzzles(10, 0)
	if err != nil || len(puzzles) != 1 {
		t.Fatalf("persisted puzzles: %v len %d", err, len(puzzles))
	}
	resp, raw = postJSON(t, c, ts.URL+config.APIGeneratePath, `{"clues": 5}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("api generate bad clues status: got %d want 400: %s", resp.StatusCode, raw)
	}
	var e errorJSON
	decodeInto(t, raw, &e)
	if e.Error == "" {
		t.Errorf("api generate bad clues envelope empty: %q", raw)
	}
}

func TestAPIGenerateRejectsSeedAboveInt64(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	resp, raw := postJSON(t, c, ts.URL+config.APIGeneratePath, `{"clues": 40, "seed": 9223372036854775808}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("seed above max int64 status: got %d want 400: %s", resp.StatusCode, raw)
	}
	var e errorJSON
	decodeInto(t, raw, &e)
	if !strings.Contains(e.Error, "seed") {
		t.Errorf("error envelope: got %q want the seed ceiling message", e.Error)
	}
	puzzles, err := st.ListPuzzles(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(puzzles) != 0 {
		t.Errorf("rejected seed must not persist: got %d puzzles", len(puzzles))
	}
}

func TestGenerateFragmentRejectsSeedAboveInt64(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FlagClues: {"40"}, config.FlagSeed: {"18446744073709551615"}}
	resp, body := postForm(t, c, ts.URL+config.FragmentGenPath, form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("fragment seed above max int64 status: got %d want 400: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "seed") {
		t.Errorf("fragment body lacks the seed cue: %q", body)
	}
	puzzles, err := st.ListPuzzles(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(puzzles) != 0 {
		t.Errorf("rejected fragment seed must not persist: got %d puzzles", len(puzzles))
	}
}

func TestAPIHealthAndVersion(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	var h healthJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIHealthPath, &h)
	if code != http.StatusOK || h.Status != config.HealthStatus {
		t.Errorf("health: status %d body %+v", code, h)
	}
	var v versionJSON
	code, _ = decodeJSON(t, c, ts.URL+config.APIVersionPath, &v)
	if code != http.StatusOK || v.Name != config.BinName || v.Version != config.Version {
		t.Errorf("version: status %d body %+v", code, v)
	}
}

func TestAPINotFoundEnvelope(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	var e errorJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIMountPath+"v1/bogus", &e)
	if code != http.StatusNotFound || e.Error == "" {
		t.Fatalf("api catch-all: status %d envelope %q", code, e.Error)
	}
	resp, raw := postJSON(t, c, ts.URL+config.APIMountPath+"v1/also-bogus", "{}")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("api catch-all post: status %d want 404: %s", resp.StatusCode, raw)
	}
}

func TestRunPageShowsMethodCards(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	req := fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, puzzle.Sample81)
	resp, raw := postJSON(t, c, ts.URL+config.APISolvePath, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api solve status: got %d want 200: %s", resp.StatusCode, raw)
	}
	var out solveJSON
	decodeInto(t, raw, &out)
	code, body := getBody(t, c, ts.URL+fmt.Sprintf(config.RunPagePathFmt, out.ID))
	if code != http.StatusOK {
		t.Fatalf("run page status: got %d want 200", code)
	}
	for _, label := range []string{config.LabelPropagate, config.LabelCover, config.LabelAnneal, config.LabelRace} {
		if !strings.Contains(body, label) {
			t.Errorf("run page lacks method label %q", label)
		}
	}
	if n := solvedTableCount(body); n != 3 {
		t.Errorf("run page solved tables: got %d want 3", n)
	}
	code, _ = getBody(t, c, ts.URL+fmt.Sprintf(config.RunPagePathFmt, 424242))
	if code != http.StatusNotFound {
		t.Errorf("unknown run page status: got %d want 404", code)
	}
}

func TestGenerateAndHistoryPagesRender(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	code, gen := getBody(t, c, ts.URL+config.GeneratePagePath)
	if code != http.StatusOK {
		t.Fatalf("generate page status: got %d want 200", code)
	}
	if !strings.Contains(gen, config.FlagClues) || !strings.Contains(gen, config.FlagSeed) || !strings.Contains(gen, config.FragmentGenPath) {
		t.Errorf("generate page lacks clue/seed inputs or fragment target")
	}
	code, hist := getBody(t, c, ts.URL+config.HistoryPagePath)
	if code != http.StatusOK {
		t.Fatalf("history page status: got %d want 200", code)
	}
	if !strings.Contains(hist, config.SourceSeed) {
		t.Errorf("history page lacks the seed run row")
	}
	if !strings.Contains(hist, fmt.Sprintf(config.RunPagePathFmt, 1)) {
		t.Errorf("history page lacks run detail link")
	}
}

func TestHistoryPagePaginationAndInvalidPage(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	for range config.HistoryPageSize {
		if _, err := st.InsertRun(puzzle.Sample81, "test", nil); err != nil {
			t.Fatal(err)
		}
	}
	code, page1 := getBody(t, c, ts.URL+config.HistoryPagePath)
	if code != http.StatusOK {
		t.Fatalf("page 1 status: got %d want 200", code)
	}
	if next := fmt.Sprintf("%s?%s=2", config.HistoryPagePath, config.QueryPage); !strings.Contains(page1, next) {
		t.Errorf("page 1 lacks next link %q", next)
	}
	if strings.Contains(page1, config.QueryPage+"=0") {
		t.Errorf("page 1 must not link to page 0: %q", page1)
	}
	if n := strings.Count(page1, `class="mono"`); n != config.HistoryPageSize {
		t.Errorf("page 1 rows: got %d want %d", n, config.HistoryPageSize)
	}
	code, page2 := getBody(t, c, ts.URL+config.HistoryPagePath+"?"+config.QueryPage+"=2")
	if code != http.StatusOK {
		t.Fatalf("page 2 status: got %d want 200", code)
	}
	if prev := fmt.Sprintf("%s?%s=1", config.HistoryPagePath, config.QueryPage); !strings.Contains(page2, prev) {
		t.Errorf("page 2 lacks prev link %q", prev)
	}
	if next := fmt.Sprintf("%s?%s=3", config.HistoryPagePath, config.QueryPage); strings.Contains(page2, next) {
		t.Errorf("partial page 2 must not link to page 3: %q", page2)
	}
	for _, q := range []string{"0", "-1", "abc", "1x"} {
		code, body := getBody(t, c, ts.URL+config.HistoryPagePath+"?"+config.QueryPage+"="+q)
		if code != http.StatusBadRequest || !strings.Contains(body, "invalid page") {
			t.Errorf("page %q: status %d body %q, want 400 invalid page", q, code, body)
		}
	}
}

func TestRunPageRejectsNonNumericID(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	code, _ := getBody(t, c, ts.URL+"/runs/abc")
	if code != http.StatusNotFound {
		t.Errorf("non numeric run page id: status %d want 404", code)
	}
}

func TestGenerateFragmentRejectsBadParams(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	cases := []struct {
		name string
		form url.Values
	}{
		{"bad clues", url.Values{config.FlagClues: {"abc"}, config.FlagSeed: {"1"}}},
		{"bad seed", url.Values{config.FlagClues: {"40"}, config.FlagSeed: {"0x10"}}},
		{"clues out of range", url.Values{config.FlagClues: {"5"}, config.FlagSeed: {"1"}}},
	}
	for _, tc := range cases {
		resp, body := postForm(t, c, ts.URL+config.FragmentGenPath, tc.form)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status: got %d want 400: %s", tc.name, resp.StatusCode, body)
		}
		if !strings.Contains(body, "badge") && !strings.Contains(body, "error") {
			t.Errorf("%s: body lacks the error cue: %q", tc.name, body)
		}
	}
}

func TestSolveFragmentRejectsInvalidCharacter(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	grid := []byte(puzzle.Sample81)
	grid[40] = 'x'
	form := url.Values{config.FormGrid81: {string(grid)}}
	resp, body := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid char status: got %d want 400: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "invalid character") {
		t.Errorf("body lacks the invalid character cue: %q", body)
	}
}

func TestAPIGenerateRejectsMalformedBody(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	resp, raw := postJSON(t, c, ts.URL+config.APIGeneratePath, "{not json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed generate status: got %d want 400: %s", resp.StatusCode, raw)
	}
	var e errorJSON
	decodeInto(t, raw, &e)
	if e.Error == "" {
		t.Errorf("error envelope empty: %q", raw)
	}
}

func TestAPIPageParamsClampAndReject(t *testing.T) {
	srv, st := newServer(t)
	ts, c := startSite(t, srv.Handler())
	for range config.APIPageLimitMax {
		if _, err := st.InsertRun(puzzle.Sample81, "test", nil); err != nil {
			t.Fatal(err)
		}
	}
	var runs runsJSON
	code, _ := decodeJSON(t, c, ts.URL+config.APIRunsPath+"?"+config.QueryLimit+"=500", &runs)
	if code != http.StatusOK {
		t.Fatalf("oversized limit status: got %d want 200", code)
	}
	if len(runs.Runs) != config.APIPageLimitMax {
		t.Errorf("oversized limit clamped: got %d runs want %d", len(runs.Runs), config.APIPageLimitMax)
	}
	for _, q := range []string{
		config.APIRunsPath + "?" + config.QueryOffset + "=-1",
		config.APIPuzzlesPath + "?" + config.QueryOffset + "=x",
		config.APIPuzzlesPath + "?" + config.QueryLimit + "=0",
	} {
		code, raw := getBody(t, c, ts.URL+q)
		if code != http.StatusBadRequest {
			t.Errorf("%s: status %d want 400: %s", q, code, raw)
		}
	}
}

func TestSolveFragmentRejectsWrongLength(t *testing.T) {
	srv, _ := newServer(t)
	ts, c := startSite(t, srv.Handler())
	form := url.Values{config.FormGrid81: {strings.Repeat("5", config.Cells-1)}}
	resp, body := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("short grid status: got %d want 400: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "expected 81") {
		t.Errorf("body lacks the length cue: %q", body)
	}
}

func TestServerGracefulShutdownNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t)
	path := filepath.Join(t.TempDir(), "shutdown.db")
	st := openStore(t, path)
	cfg := config.Defaults()
	srv := server.New(context.Background(), cfg, st, logging.New(""))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	client := &http.Client{Timeout: 60 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Get("http://" + ln.Addr().String() + config.APIHealthPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	err = <-serveErr
	if err != nil && err != http.ErrServerClosed {
		t.Fatalf("serve returned: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
