package puzzle

import (
	"context"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestExtremeSampleIsLegit(t *testing.T) {
	g, err := sudoku.Parse81(SampleExtreme81)
	if err != nil {
		t.Fatalf("extreme sample does not parse: %v", err)
	}
	clues := 0
	for i := range g {
		if g[i] != config.EmptyDigit {
			clues++
		}
	}
	if clues < 20 || clues > 40 {
		t.Errorf("extreme clues: got %d, want within 20..40", clues)
	}
	if got := solver.CountSolutions(g, 2); got != 1 {
		t.Errorf("extreme solutions: got %d, want 1", got)
	}
	sol, err := sudoku.Parse81(SampleExtremeSolution81)
	if err != nil {
		t.Fatalf("extreme solution does not parse: %v", err)
	}
	if !ValidSolution(sol) {
		t.Errorf("extreme solution is not a valid solved grid")
	}
	if !GivensPreserved(sol, g) {
		t.Errorf("extreme solution loses givens")
	}
}

func TestExtremeSampleExceedsClassicSteps(t *testing.T) {
	classic, err := sudoku.Parse81(Sample81)
	if err != nil {
		t.Fatal(err)
	}
	extreme, err := sudoku.Parse81(SampleExtreme81)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	classicRes := solver.Propagate(context.Background(), classic, cfg)
	if classicRes.Steps != 64 {
		t.Errorf("classic steps: got %d, want 64", classicRes.Steps)
	}
	extremeRes := solver.Propagate(context.Background(), extreme, cfg)
	if extremeRes.Steps <= classicRes.Steps {
		t.Errorf("extreme steps: got %d, want above the classic %d", extremeRes.Steps, classicRes.Steps)
	}
}

func TestSamplesList(t *testing.T) {
	got := Samples()
	if len(got) != 2 {
		t.Fatalf("samples: got %d entries, want 2", len(got))
	}
	if got[0] != (Sample{Label: "classic", Grid81: Sample81}) {
		t.Errorf("first sample: got %+v", got[0])
	}
	if got[1] != (Sample{Label: "extreme complexity", Grid81: SampleExtreme81}) {
		t.Errorf("second sample: got %+v", got[1])
	}
}
