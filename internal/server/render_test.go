package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/store"
	"github.com/lavantien/go-sudoku-solver/web"
)

type failWriter struct{ http.ResponseWriter }

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write refused") }

func newRenderServer() *Server {
	return &Server{log: logging.New(""), templates: parseTemplates()}
}

func TestRenderPageUnknownTemplateFails(t *testing.T) {
	s := newRenderServer()
	rec := httptest.NewRecorder()
	s.renderPage(rec, "bogus", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want 500", rec.Code)
	}
	if rec.Body.String() != "template not found\n" {
		t.Errorf("body: got %q want the not-found cue", rec.Body.String())
	}
}

func TestRenderFragmentUnknownTemplateFails(t *testing.T) {
	s := newRenderServer()
	rec := httptest.NewRecorder()
	s.renderFragment(rec, http.StatusOK, "bogus", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want 500", rec.Code)
	}
	if rec.Body.String() != "template not found\n" {
		t.Errorf("body: got %q want the not-found cue", rec.Body.String())
	}
}

func TestRenderWriteFailuresAreLoggedNotPanicked(t *testing.T) {
	s := newRenderServer()
	fw := failWriter{httptest.NewRecorder()}
	s.renderPage(fw, "home", homeView{
		baseView: baseView{Routes: newRoutesView()},
		Field:    config.FormGrid81,
		Grid:     puzzle.Sample81,
		Samples:  puzzle.Samples(),
	})
	s.renderFragment(failWriter{httptest.NewRecorder()}, http.StatusOK, "solve", solveResultsView{Cards: methodCardsFromRows(nil)})
	s.renderSolveError(failWriter{httptest.NewRecorder()}, http.StatusBadRequest, "boom")
	s.writeJSON(failWriter{httptest.NewRecorder()}, http.StatusOK, map[string]string{"k": "v"})
}

func TestTemplatesParseFromEmbeddedFS(t *testing.T) {
	if _, err := web.Templates.ReadFile("templates/layout.html"); err != nil {
		t.Fatalf("embedded layout missing: %v", err)
	}
	pages := []string{"home", "generate", "history", "run"}
	for _, name := range pages {
		if _, ok := parseTemplates().pages[name]; !ok {
			t.Errorf("page template %q missing", name)
		}
	}
	for _, name := range []string{"solve", "puzzle"} {
		if _, ok := parseTemplates().fragments[name]; !ok {
			t.Errorf("fragment template %q missing", name)
		}
	}
}

func openServerStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "render.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestServeOnClosedListenerReturnsError(t *testing.T) {
	st := openServerStore(t)
	cfg := config.Defaults()
	srv := New(context.Background(), cfg, st, logging.New(""))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if err := srv.Serve(ln); err == nil {
		t.Fatal("serve on a closed listener: want error")
	}
}

func TestShutdownWithStalledClientReturnsContextError(t *testing.T) {
	st := openServerStore(t)
	cfg := config.Defaults()
	srv := New(context.Background(), cfg, st, logging.New(""))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	active := make(chan struct{})
	srv.http.ConnState = func(_ net.Conn, st http.ConnState) {
		if st == http.StateActive {
			select {
			case <-active:
			default:
				close(active)
			}
		}
	}
	go func() { serveErr <- srv.Serve(ln) }()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	head := "POST " + config.APISolvePath + " HTTP/1.1\r\nHost: stall\r\nContent-Length: 100\r\n\r\n1"
	if _, err := conn.Write([]byte(head)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-active:
	case <-time.After(10 * time.Second):
		t.Fatal("stalled request never became active")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := srv.Shutdown(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("shutdown with a stalled client: got %v want context.Canceled", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, done := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer done()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned: %v", err)
		}
	case <-time.After(cfg.Server.ShutdownTimeout + 5*time.Second):
		t.Fatal("serve did not return after shutdown")
	}
}
