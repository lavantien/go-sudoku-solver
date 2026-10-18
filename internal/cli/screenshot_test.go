package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
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

func TestScreenshotRegisteredInRegistry(t *testing.T) {
	c, ok := Lookup(config.CmdScreenshot)
	if !ok {
		t.Fatal("screenshot command not registered")
	}
	if c.Name != config.CmdScreenshot || c.Usage == "" {
		t.Errorf("registered command = %+v", c)
	}
}
