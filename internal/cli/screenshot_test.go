package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
)

var screenshotWantPictures = []string{
	"01-home.png",
	"02-solve-results.png",
	"03-generator.png",
	"04-history.png",
	"05-run-detail.png",
	"06-api-json.png",
	"07-cli-session.png",
	"08-curl-session.png",
}

func TestScreenshotWritesEightNumberedPictures(t *testing.T) {
	dir := t.TempDir()
	_, _ = withCapture(t, "", func() {
		done := make(chan error, 1)
		go func() {
			done <- runScreenshot(context.Background(), config.Defaults(), []string{"--" + config.FlagOut, dir})
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("screenshot run: %v", err)
			}
		case <-time.After(config.ShotBudget + time.Minute):
			t.Fatalf("screenshot run exceeded %s", config.ShotBudget)
		}
	})
	for _, name := range screenshotWantPictures {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("picture %s: %v", name, err)
		}
		if info.Size() <= 5000 {
			t.Errorf("picture %s: got %d bytes, want more than 5000", name, info.Size())
		}
	}
}

func TestScreenshotRejectsUnexpectedArg(t *testing.T) {
	if err := runScreenshot(context.Background(), config.Defaults(), []string{"extra"}); err == nil {
		t.Error("screenshot must reject positional arguments")
	}
}

func TestScreenshotRejectsUnknownFlag(t *testing.T) {
	if err := runScreenshot(context.Background(), config.Defaults(), []string{"-nope"}); err == nil {
		t.Error("screenshot must reject unknown flags")
	}
}

func TestScreenshotRejectsBrowserEnvDirectory(t *testing.T) {
	t.Setenv(config.BrowserEnv, t.TempDir())
	if err := runScreenshot(context.Background(), config.Defaults(), nil); err == nil {
		t.Error("browser env pointing at a directory must fail")
	}
}

func TestScreenshotRejectsUnusableOutDir(t *testing.T) {
	t.Setenv(config.BrowserEnv, os.Args[0])
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runScreenshot(context.Background(), config.Defaults(), []string{"--" + config.FlagOut, filepath.Join(blocker, "shots")}); err == nil {
		t.Error("out dir under a plain file must fail")
	}
}

func TestPostSolveReportsTransportStatusAndIndentErrors(t *testing.T) {
	if _, err := postSolve("http://127.0.0.1:1"); err == nil {
		t.Error("dead base url: got nil want error")
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(ts.Close)
	if _, err := postSolve(ts.URL); err == nil {
		t.Error("non-200 api solve: got nil want error")
	}
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(ts2.Close)
	if _, err := postSolve(ts2.URL); err == nil {
		t.Error("malformed api solve body: got nil want error")
	}
}

func TestShotterStepsFailOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &shotter{ctx: ctx, outDir: t.TempDir(), logger: logging.New("")}
	steps := map[string]func() error{
		"home and solve": func() error { return s.homeAndSolve("http://127.0.0.1:1") },
		"generator":      func() error { return s.generator("http://127.0.0.1:1") },
		"history":        func() error { return s.history("http://127.0.0.1:1") },
		"run detail":     func() error { return s.runDetail("http://127.0.0.1:1") },
		"api json":       func() error { return s.apiJSON("{}") },
		"cli session":    func() error { return s.cliSession("$ solve") },
		"curl session":   func() error { return s.curlSession("{}") },
		"full capture":   func() error { return s.full(config.PictureHome) },
	}
	for name, step := range steps {
		if err := step(); err == nil {
			t.Errorf("%s must fail on a canceled context", name)
		}
	}
}

func TestShootFailsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := shoot(ctx, os.Args[0], "http://127.0.0.1:1", "{}", "report", t.TempDir(), logging.New("")); err == nil {
		t.Error("shoot must fail on a canceled context")
	}
}

func TestDetectBrowserErrorsWithoutAnyCandidate(t *testing.T) {
	t.Setenv(config.BrowserEnv, "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("ProgramFiles(x86)", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "empty"))
	_, err := detectBrowser()
	if err == nil {
		t.Error("no browser anywhere must produce a setup error")
	}
}

func TestPostSolveReportsTruncatedBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(ts.Close)
	if _, err := postSolve(ts.URL); err == nil {
		t.Error("truncated api solve body: got nil want error")
	}
}

func TestScreenshotRegisteredInRegistry(t *testing.T) {
	c, ok := Lookup(config.CmdScreenshot)
	if !ok {
		t.Fatal("screenshot command not registered")
	}
	if c.Name != config.CmdScreenshot || c.Usage == "" {
		t.Errorf("registered command = %+v", c)
	}
}
