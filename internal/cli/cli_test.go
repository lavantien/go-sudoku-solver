package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func withCapture(t *testing.T, stdin string, fn func()) (string, string) {
	t.Helper()
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	ir, iw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(iw, stdin); err != nil {
		t.Fatal(err)
	}
	_ = iw.Close()
	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = ir, ow, ew
	defer func() {
		os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
		_ = ow.Close()
		_ = ew.Close()
		_ = or.Close()
		_ = er.Close()
	}()
	var outBuf, errBuf bytes.Buffer
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(&outBuf, or); done <- struct{}{} }()
	go func() { _, _ = io.Copy(&errBuf, er); done <- struct{}{} }()
	fn()
	_ = ow.Close()
	_ = ew.Close()
	<-done
	<-done
	return outBuf.String(), errBuf.String()
}

func TestRegistryHoldsCommands(t *testing.T) {
	want := []string{config.CmdCoverage, config.CmdGen, config.CmdScreenshot, config.CmdServe, config.CmdSolve, config.CmdVersion}
	if got := Names(); !slices.Equal(got, want) {
		t.Errorf("names: got %v want %v", got, want)
	}
	for _, n := range want {
		if _, ok := Lookup(n); !ok {
			t.Errorf("lookup %s failed", n)
		}
	}
}

func TestMainUnknownCommandPrintsUsageExitsTwo(t *testing.T) {
	stdout, stderr := withCapture(t, "", func() {
		if code := Main([]string{"bogus"}); code != 2 {
			t.Errorf("exit code: got %d want 2", code)
		}
	})
	if stdout != "" {
		t.Errorf("stdout: got %q", stdout)
	}
	if !strings.HasPrefix(stderr, fmt.Sprintf(config.UsageHeaderFmt, config.BinName)) {
		t.Errorf("stderr lacks usage header: %q", stderr)
	}
	for _, n := range Names() {
		if !strings.Contains(stderr, n) {
			t.Errorf("stderr lacks command %s: %q", n, stderr)
		}
	}
}

func TestMainVersionPrintsBannerExitsZero(t *testing.T) {
	stdout, stderr := withCapture(t, "", func() {
		if code := Main([]string{config.CmdVersion}); code != 0 {
			t.Errorf("exit code: got %d want 0", code)
		}
	})
	want := fmt.Sprintf(config.VersionFmt, config.BinName, config.Version)
	if stdout != want {
		t.Errorf("stdout: got %q want %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr: got %q", stderr)
	}
}

func TestMainBareInvocationRunsSolve(t *testing.T) {
	stdin := strings.Repeat(".........\n", 8)
	stdout, stderr := withCapture(t, stdin, func() {
		if code := Main(nil); code != 1 {
			t.Errorf("exit code: got %d want 1", code)
		}
	})
	if stdout != "" {
		t.Errorf("stdout: got %q", stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, config.ErrNotEnoughLines)
	if stderr != want {
		t.Errorf("stderr: got %q want %q", stderr, want)
	}
}

func TestMainSolveExplicitMatchesBare(t *testing.T) {
	stdin := strings.Repeat(".........\n", 8)
	stdout, stderr := withCapture(t, stdin, func() {
		if code := Main([]string{config.CmdSolve}); code != 1 {
			t.Errorf("exit code: got %d want 1", code)
		}
	})
	if stdout != "" {
		t.Errorf("stdout: got %q", stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, config.ErrNotEnoughLines)
	if stderr != want {
		t.Errorf("stderr: got %q want %q", stderr, want)
	}
}

func TestMainSolvePrintsFullReport(t *testing.T) {
	var in strings.Builder
	for r := range config.Size {
		in.WriteString(puzzle.Sample81[r*config.Size : (r+1)*config.Size])
		in.WriteString("\n")
	}
	stdout, stderr := withCapture(t, in.String(), func() {
		if code := Main([]string{config.CmdSolve}); code != 0 {
			t.Errorf("exit code: got %d want 0", code)
		}
	})
	if stderr != "" {
		t.Errorf("stderr: got %q want empty", stderr)
	}
	if !strings.HasPrefix(stdout, config.InitialHeader+"\n") {
		t.Errorf("report lacks the initial header: %q", stdout[:min(80, len(stdout))])
	}
	for _, label := range []string{config.LabelPropagate, config.LabelCover, config.LabelAnneal, config.LabelRace} {
		if !strings.Contains(stdout, label) {
			t.Errorf("report lacks method label %q", label)
		}
	}
	if got := strings.Count(stdout, config.Separator); got != 2*len(config.MethodIDs) {
		t.Errorf("separator count: got %d want %d", got, 2*len(config.MethodIDs))
	}
	if got := strings.Count(stdout, "Step count: "); got != len(config.MethodIDs) {
		t.Errorf("step lines: got %d want %d", got, len(config.MethodIDs))
	}
}

func TestMainGenRejectsUnknownFlag(t *testing.T) {
	args := []string{config.CmdGen, "-nope"}
	stdout, _ := withCapture(t, "", func() {
		if code := Main(args); code != 1 {
			t.Errorf("exit code: got %d want 1", code)
		}
	})
	if stdout != "" {
		t.Errorf("stdout: got %q want empty", stdout)
	}
}

func TestMainGenRejectsBadClueRange(t *testing.T) {
	args := []string{config.CmdGen, "--" + config.FlagClues, "5"}
	stdout, stderr := withCapture(t, "", func() {
		if code := Main(args); code != 1 {
			t.Errorf("exit code: got %d want 1", code)
		}
	})
	if stdout != "" {
		t.Errorf("stdout: got %q", stdout)
	}
	want := fmt.Sprintf(config.RunErrFmt, config.BinName, config.CmdGen, "generator clues must be between 17 and 81")
	if stderr != want {
		t.Errorf("stderr: got %q want %q", stderr, want)
	}
}

func TestMainGenPrintsUniquePuzzle(t *testing.T) {
	args := []string{config.CmdGen, "--" + config.FlagClues, "40", "--" + config.FlagSeed, "1"}
	stdout, stderr := withCapture(t, "", func() {
		if code := Main(args); code != 0 {
			t.Errorf("exit code: got %d want 0", code)
		}
	})
	if stderr != "" {
		t.Errorf("stderr: got %q", stderr)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != config.Size {
		t.Fatalf("line count: got %d want %d", len(lines), config.Size)
	}
	for i, ln := range lines {
		if len(ln) != config.Size {
			t.Fatalf("line %d length: got %d want %d", i, len(ln), config.Size)
		}
		for _, ch := range ln {
			if ch != rune(config.EmptyChar) && (ch < '1' || ch > '9') {
				t.Errorf("line %d carries invalid char %q", i, ch)
			}
		}
	}
	g, err := sudoku.ParseInput(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("generated puzzle does not re-parse: %v", err)
	}
	if n := solver.CountSolutions(g, 2); n != 1 {
		t.Errorf("generated puzzle solutions: got %d want 1", n)
	}
}
