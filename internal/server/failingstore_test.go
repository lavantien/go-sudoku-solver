package server_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/server"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

type failStore struct{ err error }

func (f failStore) CountRuns() (int64, error) { return 0, f.err }
func (f failStore) InsertRun(string, string, []store.MethodRow) (int64, error) {
	return 0, f.err
}
func (f failStore) ListRuns(int, int) ([]store.RunSummary, error) { return nil, f.err }
func (f failStore) GetRun(int64) (*store.RunDetail, error)        { return nil, f.err }
func (f failStore) InsertPuzzle(string, int, int64) (int64, error) {
	return 0, f.err
}
func (f failStore) ListPuzzles(int, int) ([]store.Puzzle, error) { return nil, f.err }

type seedFailStore struct{ failStore }

func (seedFailStore) CountRuns() (int64, error) { return 0, nil }

func newFailingServer(t *testing.T, st server.RunStore) *server.Server {
	t.Helper()
	return server.New(context.Background(), config.Defaults(), st, logging.New(""))
}

func TestNewToleratesFailingStoreDuringSeed(t *testing.T) {
	srv := newFailingServer(t, failStore{err: errors.New("count refused")})
	if srv == nil || srv.Handler() == nil {
		t.Fatal("server must survive a failing store at seed time")
	}
	srv = newFailingServer(t, seedFailStore{failStore{err: errors.New("insert refused")}})
	if srv == nil || srv.Handler() == nil {
		t.Fatal("server must survive a seed insert failure")
	}
}

func TestStoreFailuresReturnServerError(t *testing.T) {
	boom := errors.New("storage offline")
	srv := newFailingServer(t, failStore{err: boom})
	ts, c := startSite(t, srv.Handler())

	cases := []struct {
		name string
		call func(t *testing.T) (int, string)
		json bool
		want string
	}{
		{"runs list", func(t *testing.T) (int, string) {
			var e errorJSON
			code, raw := decodeJSON(t, c, ts.URL+config.APIRunsPath, &e)
			return code, raw
		}, true, boom.Error()},
		{"run detail", func(t *testing.T) (int, string) {
			var e errorJSON
			code, raw := decodeJSON(t, c, ts.URL+fmt.Sprintf(config.APIRunPathFmt, 1), &e)
			return code, raw
		}, true, boom.Error()},
		{"puzzles list", func(t *testing.T) (int, string) {
			var e errorJSON
			code, raw := decodeJSON(t, c, ts.URL+config.APIPuzzlesPath, &e)
			return code, raw
		}, true, boom.Error()},
		{"generate", func(t *testing.T) (int, string) {
			resp, raw := postJSON(t, c, ts.URL+config.APIGeneratePath, `{"clues": 40, "seed": 7}`)
			return resp.StatusCode, raw
		}, true, boom.Error()},
		{"solve", func(t *testing.T) (int, string) {
			body := fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, puzzle.Sample81)
			resp, raw := postJSON(t, c, ts.URL+config.APISolvePath, body)
			return resp.StatusCode, raw
		}, true, "server persist run: " + boom.Error()},
		{"solve fragment", func(t *testing.T) (int, string) {
			form := url.Values{config.FormGrid81: {puzzle.Sample81}}
			resp, raw := postForm(t, c, ts.URL+config.FragmentSolvePath, form)
			return resp.StatusCode, raw
		}, false, ""},
		{"history page", func(t *testing.T) (int, string) {
			return getBody(t, c, ts.URL+config.HistoryPagePath)
		}, false, ""},
		{"run page", func(t *testing.T) (int, string) {
			return getBody(t, c, ts.URL+fmt.Sprintf(config.RunPagePathFmt, 1))
		}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := tc.call(t)
			if code != http.StatusInternalServerError {
				t.Fatalf("status: got %d want 500: %s", code, raw)
			}
			if tc.json {
				var e errorJSON
				decodeInto(t, raw, &e)
				if e.Error != tc.want {
					t.Errorf("error envelope: got %q want %q", e.Error, tc.want)
				}
			} else if !strings.Contains(raw, "storage error") {
				t.Errorf("page body lacks the storage error cue: %q", raw)
			}
		})
	}
}
