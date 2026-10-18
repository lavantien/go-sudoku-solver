package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
)

func TestBareSolveMatchesSampleGolden(t *testing.T) {
	run := runSudoku(t, nineLineInput(puzzle.Sample81))
	if run.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", run.ExitCode)
	}
	if run.Stderr != "" {
		t.Errorf("stderr: got %q want empty", run.Stderr)
	}
	assertGolden(t, "sample.golden", normalizeReport(t, run.Stdout))
}

func TestSolveIgnoresTrailingGarbageAfterNineLines(t *testing.T) {
	stdin := nineLineInput(puzzle.Sample81) + "GARBAGE\n0123456789\n"
	run := runSudoku(t, stdin)
	if run.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", run.ExitCode)
	}
	if run.Stderr != "" {
		t.Errorf("stderr: got %q want empty", run.Stderr)
	}
	assertGolden(t, "sample.golden", normalizeReport(t, run.Stdout))
}

func TestSolveAcceptsCrlfLineEndings(t *testing.T) {
	run := runSudoku(t, strings.ReplaceAll(nineLineInput(puzzle.Sample81), "\n", "\r\n"))
	if run.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", run.ExitCode)
	}
	if run.Stderr != "" {
		t.Errorf("stderr: got %q want empty", run.Stderr)
	}
	assertGolden(t, "sample.golden", normalizeReport(t, run.Stdout))
}

func TestSolveUnsolvableMatchesGolden(t *testing.T) {
	run := runSudoku(t, nineLineInput(puzzle.Unsolvable81))
	if run.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", run.ExitCode)
	}
	if run.Stderr != "" {
		t.Errorf("stderr: got %q want empty", run.Stderr)
	}
	assertGolden(t, "unsolvable.golden", normalizeReport(t, run.Stdout))
}

func TestSolveEmptyStdinFails(t *testing.T) {
	run := runSudoku(t, "")
	if run.ExitCode != 1 {
		t.Fatalf("exit code: got %d want 1", run.ExitCode)
	}
	if run.Stdout != "" {
		t.Errorf("stdout: got %q want empty", run.Stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, config.ErrNotEnoughLines)
	if run.Stderr != want {
		t.Errorf("stderr: got %q want %q", run.Stderr, want)
	}
}

func TestSolveEightLinesFails(t *testing.T) {
	run := runSudoku(t, strings.Repeat(".........\n", 8))
	if run.ExitCode != 1 {
		t.Fatalf("exit code: got %d want 1", run.ExitCode)
	}
	if run.Stdout != "" {
		t.Errorf("stdout: got %q want empty", run.Stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, config.ErrNotEnoughLines)
	if run.Stderr != want {
		t.Errorf("stderr: got %q want %q", run.Stderr, want)
	}
}

func TestSolveMidlineCarriageReturnFails(t *testing.T) {
	corrupt := strings.Replace(nineLineInput(puzzle.Sample81), ".", "\r", 1)
	if first := strings.Split(corrupt, "\n")[0]; len(first) != config.Size {
		t.Fatalf("corrupt first line is %d bytes, want %d so the length gate passes", len(first), config.Size)
	}
	run := runSudoku(t, corrupt)
	if run.ExitCode != 1 {
		t.Fatalf("exit code: got %d want 1", run.ExitCode)
	}
	if run.Stdout != "" {
		t.Errorf("stdout: got %q want empty", run.Stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, fmt.Sprintf(config.ErrCharFmt, '\r'))
	if run.Stderr != want {
		t.Errorf("stderr: got %q want %q", run.Stderr, want)
	}
}

func TestSolveNinthEmptyLineFails(t *testing.T) {
	run := runSudoku(t, strings.Repeat(".........\n", 8)+"\n")
	if run.ExitCode != 1 {
		t.Fatalf("exit code: got %d want 1", run.ExitCode)
	}
	if run.Stdout != "" {
		t.Errorf("stdout: got %q want empty", run.Stdout)
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, fmt.Sprintf(config.ErrLineLengthFmt, 0))
	if run.Stderr != want {
		t.Errorf("stderr: got %q want %q", run.Stderr, want)
	}
}

func TestVersionPrintsBanner(t *testing.T) {
	run := runSudoku(t, "", config.CmdVersion)
	if run.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", run.ExitCode)
	}
	want := fmt.Sprintf(config.VersionFmt, config.BinName, config.Version)
	if run.Stdout != want {
		t.Errorf("stdout: got %q want %q", run.Stdout, want)
	}
	if run.Stderr != "" {
		t.Errorf("stderr: got %q want empty", run.Stderr)
	}
}
