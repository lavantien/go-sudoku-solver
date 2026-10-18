package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = old
	return <-done
}

func TestDiscardHandlerMethods(t *testing.T) {
	h := discardHandler{}
	if h.Enabled(context.Background(), slog.LevelError) {
		t.Error("discard handler must report every level disabled")
	}
	if err := h.Handle(context.Background(), slog.Record{}); err != nil {
		t.Errorf("handle: %v", err)
	}
	wa, ok := h.WithAttrs([]slog.Attr{slog.String("k", "v")}).(discardHandler)
	if !ok || wa.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("with attrs must keep discarding: %#v", wa)
	}
	wg, ok := h.WithGroup("g").(discardHandler)
	if !ok || wg.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("with group must keep discarding: %#v", wg)
	}
}

func TestTextHandlerLevelCases(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "nonsense"} {
		l := New(level)
		if l == nil {
			t.Fatalf("level %q: nil logger", level)
		}
	}
	out := captureStderr(t, func() {
		l := New("warn")
		l.Info("hidden")
		l.Error("shown")
	})
	if strings.Contains(out, "hidden") {
		t.Errorf("info leaked at warn level: %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Errorf("error missing at warn level: %q", out)
	}
}

func TestDiscardByDefault(t *testing.T) {
	for _, level := range []string{"", "discard"} {
		out := captureStderr(t, func() {
			l := New(level)
			Enter(l, "unit")
			Exit(l, "unit")
			Fail(l, "unit", errors.New("boom"))
		})
		if out != "" {
			t.Errorf("level %q: expected no output, got %q", level, out)
		}
	}
}

func TestEnterExitWriteAtDebug(t *testing.T) {
	out := captureStderr(t, func() {
		l := New("debug")
		Enter(l, "solve", "puzzle", "sample")
		Exit(l, "solve", "steps", 3)
	})
	for _, want := range []string{"phase=enter", "msg=solve", "puzzle=sample", "phase=exit", "steps=3"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
}

func TestFailWritesAtError(t *testing.T) {
	out := captureStderr(t, func() {
		l := New("error")
		Enter(l, "solve")
		Fail(l, "solve", errors.New("bad grid"))
	})
	if !strings.Contains(out, "phase=fail") || !strings.Contains(out, "bad grid") {
		t.Errorf("fail line missing parts: %q", out)
	}
	if strings.Contains(out, "phase=enter") {
		t.Errorf("enter leaked at error level: %q", out)
	}
}
