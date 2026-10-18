package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const (
	bootTimeout  = 60 * time.Second
	killTimeout  = 30 * time.Second
	clientPolicy = 120 * time.Second
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

type serveProcess struct {
	addr string
	stop func(t *testing.T)
}

func bootServeProcess(t *testing.T) *serveProcess {
	t.Helper()
	db := filepath.Join(t.TempDir(), "serve.db")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, sudokuBin(t), config.CmdServe,
		"--"+config.FlagAddr, "127.0.0.1:0",
		"--"+config.FlagDB, db)
	cmd.Env = cleanSudokuEnv()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Logf("e2e: starting serve on an ephemeral port with db %s", db)
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start serve: %v", err)
	}
	addrCh := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			if line, ok := strings.CutPrefix(sc.Text(), config.ServeAddrPrefix); ok {
				addrCh <- strings.TrimSpace(line)
				return
			}
		}
		close(addrCh)
	}()
	go func() { _, _ = io.Copy(io.Discard, stdout) }()
	var addr string
	select {
	case v, ok := <-addrCh:
		if !ok {
			_ = cmd.Wait()
			t.Fatalf("serve stderr closed without the addr line")
		}
		addr = v
	case <-time.After(bootTimeout):
		cancel()
		_ = cmd.Wait()
		t.Fatalf("timeout waiting for the serve addr line")
	}
	t.Logf("e2e: serve listening on %s", addr)
	if _, _, err := net.SplitHostPort(addr); err != nil {
		t.Fatalf("addr line is not host:port: %q", addr)
	}
	stop := func(t *testing.T) {
		t.Helper()
		t.Logf("e2e: killing the serve process tree")
		cancel()
		killProcessTree(t, cmd)
		waitErr := cmd.Wait()
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			t.Fatalf("serve did not die from the kill: wait returned %v", waitErr)
		}
		waitForRefused(t, addr)
	}
	return &serveProcess{addr: addr, stop: stop}
}

func killProcessTree(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
	} else {
		err = cmd.Process.Kill()
	}
	if err != nil {
		t.Logf("e2e: kill returned %v", err)
	}
}

func waitForRefused(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(killTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			t.Logf("e2e: port %s closed after the kill", addr)
			return
		}
		_ = conn.Close()
	}
	t.Fatalf("port %s still accepts connections %v after the kill", addr, killTimeout)
}

func doRequest(t *testing.T, c *http.Client, method, rawurl, body string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, rawurl, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", config.JSONContentType)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, rawurl, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("e2e: %s %s -> %d len=%d", method, rawurl, resp.StatusCode, len(raw))
	return resp.StatusCode, string(raw)
}

func decodeInto(t *testing.T, raw string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
}

func TestAPIOverLiveServeProcess(t *testing.T) {
	sp := bootServeProcess(t)
	base := "http://" + sp.addr
	client := &http.Client{Timeout: clientPolicy}
	t.Cleanup(client.CloseIdleConnections)

	t.Run("health", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+config.APIHealthPath, "")
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200", code)
		}
		var body healthJSON
		decodeInto(t, raw, &body)
		if body.Status != config.HealthStatus {
			t.Errorf("body: got %q want status %q", raw, config.HealthStatus)
		}
	})

	t.Run("version", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+config.APIVersionPath, "")
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200", code)
		}
		var body versionJSON
		decodeInto(t, raw, &body)
		if body.Name != config.BinName || body.Version != config.Version {
			t.Errorf("body: got %q want %s %s", raw, config.BinName, config.Version)
		}
	})

	t.Run("runs_start_with_the_seeded_sample", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+config.APIRunsPath, "")
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200", code)
		}
		var body runsJSON
		decodeInto(t, raw, &body)
		if len(body.Runs) != 1 {
			t.Fatalf("seeded runs: got %d want 1", len(body.Runs))
		}
		seed := body.Runs[0]
		if seed.ID != 1 || seed.Input81 != puzzle.Sample81 || seed.Source != config.SourceSeed || !seed.SolvedAny {
			t.Errorf("seed run: got id %d input %s source %s solved_any %v", seed.ID, seed.Input81, seed.Source, seed.SolvedAny)
		}
		if _, err := time.Parse(time.RFC3339, seed.CreatedAt); err != nil {
			t.Errorf("seed created_at %q: %v", seed.CreatedAt, err)
		}
	})

	var solved solveJSON
	t.Run("solve_persists_and_reports_every_method", func(t *testing.T) {
		req := fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, puzzle.Sample81)
		code, raw := doRequest(t, client, http.MethodPost, base+config.APISolvePath, req)
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200: %s", code, raw)
		}
		decodeInto(t, raw, &solved)
		if solved.ID != 2 {
			t.Errorf("run id: got %d want 2 after the seed run", solved.ID)
		}
		if solved.Input81 != puzzle.Sample81 {
			t.Errorf("input: got %q want the classic sample", solved.Input81)
		}
		if solved.Source != config.SourceAPI {
			t.Errorf("source: got %q want %q", solved.Source, config.SourceAPI)
		}
		if len(solved.Methods) != len(config.MethodIDs) {
			t.Fatalf("methods: got %d want %d", len(solved.Methods), len(config.MethodIDs))
		}
		for i, m := range solved.Methods {
			if m.Method != config.MethodIDs[i] {
				t.Errorf("method %d: got %s want %s", i, m.Method, config.MethodIDs[i])
			}
			if m.TimeNS < 0 || m.DurationMS != float64(m.TimeNS)/float64(time.Millisecond) {
				t.Errorf("method %s timing: time_ns %d duration_ms %f", m.Method, m.TimeNS, m.DurationMS)
			}
			if m.Method == config.MethodIDs[2] {
				if m.Solved {
					t.Errorf("anneal reported solved on the classic sample")
				}
				continue
			}
			if m.Grid81 != puzzle.SampleSolution81 {
				t.Errorf("method %s grid: got %s want the sample solution", m.Method, m.Grid81)
			}
			if !m.Solved {
				t.Errorf("method %s reported unsolved", m.Method)
			}
		}
	})

	t.Run("run_by_id_returns_full_detail", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+fmt.Sprintf(config.APIRunPathFmt, solved.ID), "")
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200", code)
		}
		var body runDetailJSON
		decodeInto(t, raw, &body)
		if body.ID != solved.ID || body.Input81 != puzzle.Sample81 || body.Source != config.SourceAPI {
			t.Errorf("detail summary: got id %d input %s source %s", body.ID, body.Input81, body.Source)
		}
		if len(body.Methods) != len(config.MethodIDs) {
			t.Errorf("detail methods: got %d want %d", len(body.Methods), len(config.MethodIDs))
		}
	})

	t.Run("runs_list_newest_first_after_solve", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+config.APIRunsPath, "")
		if code != http.StatusOK {
			t.Fatalf("status: got %d want 200", code)
		}
		var body runsJSON
		decodeInto(t, raw, &body)
		if len(body.Runs) != 2 {
			t.Fatalf("runs: got %d want 2", len(body.Runs))
		}
		if body.Runs[0].ID != 2 || body.Runs[1].ID != 1 {
			t.Errorf("run order: got [%d %d] want [2 1]", body.Runs[0].ID, body.Runs[1].ID)
		}
	})

	t.Run("run_not_found_is_json_404", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+fmt.Sprintf(config.APIRunPathFmt, 999999), "")
		if code != http.StatusNotFound {
			t.Fatalf("status: got %d want 404", code)
		}
		var body errorJSON
		decodeInto(t, raw, &body)
		if body.Error != "run not found" {
			t.Errorf("body: got %q want run not found", raw)
		}
	})

	t.Run("solve_rejects_short_grid", func(t *testing.T) {
		req := fmt.Sprintf(`{"%s": "12"}`, config.FormGrid81)
		code, raw := doRequest(t, client, http.MethodPost, base+config.APISolvePath, req)
		if code != http.StatusBadRequest {
			t.Fatalf("status: got %d want 400", code)
		}
		var body errorJSON
		decodeInto(t, raw, &body)
		want := fmt.Sprintf(config.ErrGridLengthFmt, 2)
		if body.Error != want {
			t.Errorf("body: got %q want %q", body.Error, want)
		}
	})

	t.Run("generate_returns_unique_puzzle", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodPost, base+config.APIGeneratePath, `{"clues": 40, "seed": 7}`)
		if code != http.StatusCreated {
			t.Fatalf("status: got %d want 201: %s", code, raw)
		}
		var body puzzleJSON
		decodeInto(t, raw, &body)
		if body.ID == 0 {
			t.Errorf("puzzle id: got %d want nonzero", body.ID)
		}
		if body.Seed != 7 {
			t.Errorf("seed: got %d want 7", body.Seed)
		}
		if body.Clues < 40 {
			t.Errorf("clues: got %d want at least the requested 40", body.Clues)
		}
		g, err := sudoku.Parse81(body.Grid81)
		if err != nil {
			t.Fatalf("generated grid does not parse: %v", err)
		}
		if n := solver.CountSolutions(g, 2); n != 1 {
			t.Errorf("generated grid solutions: got %d want 1", n)
		}
	})

	t.Run("api_mount_404_is_json", func(t *testing.T) {
		code, raw := doRequest(t, client, http.MethodGet, base+config.APIMountPath+"bogus", "")
		if code != http.StatusNotFound {
			t.Fatalf("status: got %d want 404", code)
		}
		var body errorJSON
		decodeInto(t, raw, &body)
		if body.Error != "not found" {
			t.Errorf("body: got %q want not found", raw)
		}
	})

	sp.stop(t)
}
