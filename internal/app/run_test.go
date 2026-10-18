package app

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

var (
	stepDigits = regexp.MustCompile(`Step count: [0-9]+`)
	timeDigits = regexp.MustCompile(`Execution time: [0-9]+\.[0-9]+`)
)

func normalizeReport(s string) string {
	s = stepDigits.ReplaceAllString(s, "Step count: 0")
	return timeDigits.ReplaceAllString(s, "Execution time: 0.000000")
}

func input81(t *testing.T, s81 string) string {
	t.Helper()
	if len(s81) != config.Cells {
		t.Fatalf("fixture length: got %d want %d", len(s81), config.Cells)
	}
	var b strings.Builder
	for r := range config.Size {
		b.WriteString(s81[r*config.Size : (r+1)*config.Size])
		b.WriteByte('\n')
	}
	return b.String()
}

func grid81(t *testing.T, s81 string) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(s81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func goldenReport(initial, solved sudoku.Grid) string {
	var b strings.Builder
	b.WriteString(config.InitialHeader + "\n")
	b.WriteString(sudoku.RenderGrid(initial))
	for _, id := range config.MethodIDs {
		b.WriteString(config.Separator + "\n")
		b.WriteString(sudoku.RenderGrid(solved))
		_, _ = fmt.Fprintf(&b, config.LabelFmt, config.MethodLabel(id))
		_, _ = fmt.Fprintf(&b, config.StepFmt, 0)
		_, _ = fmt.Fprintf(&b, config.TimeFmt, 0.0)
		b.WriteString(config.Separator + "\n")
	}
	return b.String()
}

func TestRunReportMatchesContractOnFullySolvableFixture(t *testing.T) {
	in := strings.NewReader(input81(t, puzzle.Anneal81))
	var out, errOut bytes.Buffer
	code := Run(context.Background(), config.Defaults(), in, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr under discard logger: got %q", errOut.String())
	}
	want := goldenReport(grid81(t, puzzle.Anneal81), grid81(t, puzzle.SampleSolution81))
	if got := normalizeReport(out.String()); got != want {
		t.Errorf("report mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunSampleReportExactBlocks(t *testing.T) {
	in := strings.NewReader(input81(t, puzzle.Sample81))
	var out, errOut bytes.Buffer
	code := Run(context.Background(), config.Defaults(), in, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr under discard logger: got %q", errOut.String())
	}
	parts := strings.Split(out.String(), config.Separator)
	if len(parts) != 1+2*len(config.MethodIDs) {
		t.Fatalf("separator parts: got %d want %d", len(parts), 1+2*len(config.MethodIDs))
	}
	wantHeader := config.InitialHeader + "\n" + sudoku.RenderGrid(grid81(t, puzzle.Sample81))
	if parts[0] != wantHeader {
		t.Errorf("header part:\ngot:\n%q\nwant:\n%q", parts[0], wantHeader)
	}
	solved := sudoku.RenderGrid(grid81(t, puzzle.SampleSolution81))
	for i, id := range config.MethodIDs {
		part := parts[1+2*i]
		blank := strings.Index(part, "\n\n")
		if blank < 0 {
			t.Fatalf("method %s: no blank line before label: %q", id, part)
		}
		render := part[1 : blank+1]
		tail := part[blank+2:]
		wantLabel := fmt.Sprintf(config.LabelFmt, config.MethodLabel(id))
		if !strings.HasPrefix(tail, wantLabel[1:]) {
			t.Errorf("method %s: tail %q lacks label line %q", id, tail, wantLabel[1:])
		}
		if id == config.MethodIDs[2] {
			continue
		}
		if render != solved {
			t.Errorf("method %s: render:\ngot:\n%s\nwant:\n%s", id, render, solved)
		}
	}
}

func TestRunParseErrorWritesStderrAndExitsOne(t *testing.T) {
	in := strings.NewReader(strings.Repeat(".........\n", 8))
	var out, errOut bytes.Buffer
	code := Run(context.Background(), config.Defaults(), in, &out, &errOut)
	if code != 1 {
		t.Fatalf("exit code: got %d want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout after parse error: got %q", out.String())
	}
	want := fmt.Sprintf("%s %s\n", config.InputErrorPrefix, config.ErrNotEnoughLines)
	if errOut.String() != want {
		t.Errorf("stderr: got %q want %q", errOut.String(), want)
	}
}
