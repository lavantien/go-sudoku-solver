package solver

import (
	"context"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestCoverSolvesSample(t *testing.T) {
	got := Cover(context.Background(), sampleGrid(t), config.Defaults())
	if !got.Solved {
		t.Fatalf("not solved")
	}
	want := solutionGrid(t)
	if got.Grid != want {
		t.Errorf("grid: got %s want %s", sudoku.Serialize(got.Grid), sudoku.Serialize(want))
	}
	if got.Steps != 81 {
		t.Errorf("steps: got %d want 81 selected rows", got.Steps)
	}
}

func TestCoverEmptyGrid(t *testing.T) {
	g := sudoku.Grid{}
	got := Cover(context.Background(), g, config.Defaults())
	if !got.Solved {
		t.Fatalf("empty grid not solved")
	}
	if !puzzle.ValidSolution(got.Grid) {
		t.Errorf("invalid solution")
	}
	if !puzzle.GivensPreserved(got.Grid, g) {
		t.Errorf("givens changed")
	}
	if got.Steps != 81 {
		t.Errorf("steps: got %d want 81", got.Steps)
	}
}

func TestCoverUnsolvable(t *testing.T) {
	in := unsolvableGrid(t)
	got := Cover(context.Background(), in, config.Defaults())
	if got.Solved {
		t.Fatalf("unsolvable reported solved")
	}
	if got.Grid != in {
		t.Errorf("grid changed")
	}
	if got.Steps != 1 {
		t.Errorf("steps: got %d want 1, the first given selected and the second rejected", got.Steps)
	}
}

func TestCoverDeterministic(t *testing.T) {
	a := Cover(context.Background(), sampleGrid(t), config.Defaults())
	b := Cover(context.Background(), sampleGrid(t), config.Defaults())
	if a.Grid != b.Grid || a.Steps != b.Steps {
		t.Errorf("two runs differ: %+v vs %+v", a, b)
	}
}

func TestCoverCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := Cover(ctx, sampleGrid(t), config.Defaults())
	if got.Solved {
		t.Errorf("cancelled context reported solved")
	}
}

func TestCoverNodeCapExhaustion(t *testing.T) {
	cfg := config.Defaults()
	cfg.CoverNodeCap = 1
	got := Cover(context.Background(), sampleGrid(t), cfg)
	if got.Solved {
		t.Errorf("cap of one node reported solved")
	}
}

func TestRegistryHasCover(t *testing.T) {
	all := All()
	if len(all) < 2 {
		t.Fatalf("registry has %d methods", len(all))
	}
	if all[1].ID != "cover" || all[1].Label != config.LabelCover {
		t.Errorf("second method: got %s %s", all[1].ID, all[1].Label)
	}
	if m, ok := ByID("cover"); !ok || m.Fn == nil {
		t.Errorf("ByID cover broken")
	}
}
