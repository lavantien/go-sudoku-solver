package solver

import (
	"context"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func annealFixtureGrid(t *testing.T) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(puzzle.Anneal81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestAnnealSolvesDefaultFixtureWithDefaultSeed(t *testing.T) {
	got := Anneal(context.Background(), annealFixtureGrid(t), config.Defaults())
	if !got.Solved {
		t.Fatalf("default seed does not solve the anneal fixture")
	}
	want := solutionGrid(t)
	if got.Grid != want {
		t.Errorf("grid: got %s want %s", sudoku.Serialize(got.Grid), sudoku.Serialize(want))
	}
	if got.Steps <= 0 {
		t.Errorf("steps: got %d", got.Steps)
	}
}

func TestAnnealSampleBestEffortUnsolved(t *testing.T) {
	in := sampleGrid(t)
	got := Anneal(context.Background(), in, config.Defaults())
	if got.Solved {
		t.Fatalf("adversarial sample reported solved by anneal")
	}
	if !puzzle.GivensPreserved(got.Grid, in) {
		t.Errorf("givens not preserved")
	}
	for u := range 9 {
		if mask := sudoku.RowUsed(got.Grid, u); mask != sudoku.AllDigits {
			t.Errorf("row %d lost a digit: %04x", u, mask)
		}
	}
	again := Anneal(context.Background(), in, config.Defaults())
	if again.Grid != got.Grid || again.Steps != got.Steps {
		t.Errorf("two runs differ")
	}
}

func TestAnnealDeterministic(t *testing.T) {
	a := Anneal(context.Background(), annealFixtureGrid(t), config.Defaults())
	b := Anneal(context.Background(), annealFixtureGrid(t), config.Defaults())
	if a.Grid != b.Grid || a.Steps != b.Steps || a.Solved != b.Solved {
		t.Errorf("two runs differ: steps %d vs %d, solved %v vs %v", a.Steps, b.Steps, a.Solved, b.Solved)
	}
}

func TestAnnealUnsolvableKeepsRowPermutations(t *testing.T) {
	in := unsolvableGrid(t)
	got := Anneal(context.Background(), in, config.Defaults())
	if got.Solved {
		t.Fatalf("unsolvable reported solved")
	}
	if !puzzle.GivensPreserved(got.Grid, in) {
		t.Errorf("givens not preserved: %s", sudoku.Serialize(got.Grid))
	}
	for u := 1; u < 9; u++ {
		mask := sudoku.RowUsed(got.Grid, u)
		if mask != sudoku.AllDigits {
			t.Errorf("row %d lost a digit: %04x", u, mask)
		}
	}
	if got.Grid[0] != 5 || got.Grid[1] != 5 {
		t.Errorf("row 0 givens moved: %s", sudoku.Serialize(got.Grid))
	}
}

func TestAnnealFullyGivenInvalidGridExitsFastUnsolvedDeterministic(t *testing.T) {
	in := solutionGrid(t)
	in[0] = in[6]
	start := time.Now()
	got := Anneal(context.Background(), in, config.Defaults())
	wall := time.Since(start)
	if got.Solved {
		t.Errorf("fully given invalid grid reported solved")
	}
	if wall >= 2*time.Second {
		t.Errorf("wall: got %v want under 2s, unswappable rounds must not burn the budget", wall)
	}
	again := Anneal(context.Background(), in, config.Defaults())
	if again.Grid != got.Grid || again.Steps != got.Steps {
		t.Errorf("two runs differ: steps %d vs %d", got.Steps, again.Steps)
	}
}

func TestAnnealCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := Anneal(ctx, sampleGrid(t), config.Defaults())
	if got.Solved {
		t.Errorf("cancelled context reported solved")
	}
}

func TestAnnealBudgetExhaustion(t *testing.T) {
	cfg := config.Defaults()
	cfg.Anneal.Restarts = 1
	cfg.Anneal.Iterations = 50
	got := Anneal(context.Background(), annealFixtureGrid(t), cfg)
	if got.Solved {
		t.Errorf("tiny budget solved the sample, budget semantics broken")
	}
	if got.Steps > 50 {
		t.Errorf("steps exceed budget: %d", got.Steps)
	}
}

func TestAnnealSolvesEmptyGrid(t *testing.T) {
	g := sudoku.Grid{}
	got := Anneal(context.Background(), g, config.Defaults())
	if !got.Solved {
		t.Fatalf("empty grid not solved by the portfolio")
	}
	if !puzzle.ValidSolution(got.Grid) {
		t.Errorf("invalid solution")
	}
}

func TestRegistryHasAnneal(t *testing.T) {
	all := All()
	if len(all) < 3 {
		t.Fatalf("registry has %d methods", len(all))
	}
	if all[2].ID != "anneal" || all[2].Label != config.LabelAnneal {
		t.Errorf("third method: got %s %s", all[2].ID, all[2].Label)
	}
}
