package solver

import (
	"context"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func sampleGrid(t *testing.T) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(puzzle.Sample81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func solutionGrid(t *testing.T) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func unsolvableGrid(t *testing.T) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(puzzle.Unsolvable81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestPropagateSolvesSampleToReferenceSolution(t *testing.T) {
	got := Propagate(context.Background(), sampleGrid(t), config.Defaults())
	if !got.Solved {
		t.Fatalf("not solved")
	}
	want := solutionGrid(t)
	if got.Grid != want {
		t.Errorf("grid: got %s want %s", sudoku.Serialize(got.Grid), sudoku.Serialize(want))
	}
	if got.Steps < 49 {
		t.Errorf("steps: got %d want at least the 49 empties", got.Steps)
	}
	if got.Dur <= 0 {
		t.Errorf("duration: got %v", got.Dur)
	}
}

func TestPropagateDeterministic(t *testing.T) {
	a := Propagate(context.Background(), sampleGrid(t), config.Defaults())
	b := Propagate(context.Background(), sampleGrid(t), config.Defaults())
	if a.Grid != b.Grid || a.Steps != b.Steps || a.Solved != b.Solved {
		t.Errorf("two runs differ: %+v vs %+v", a, b)
	}
}

func TestPropagateEmptyGrid(t *testing.T) {
	g := sudoku.Grid{}
	got := Propagate(context.Background(), g, config.Defaults())
	if !got.Solved {
		t.Fatalf("empty grid not solved")
	}
	if !puzzle.ValidSolution(got.Grid) {
		t.Errorf("empty grid solution invalid: %s", sudoku.Serialize(got.Grid))
	}
	if !puzzle.GivensPreserved(got.Grid, g) {
		t.Errorf("givens changed")
	}
}

func TestPropagateUnsolvable(t *testing.T) {
	in := unsolvableGrid(t)
	got := Propagate(context.Background(), in, config.Defaults())
	if got.Solved {
		t.Fatalf("unsolvable reported solved")
	}
	if got.Grid != in {
		t.Errorf("unsolvable grid changed: got %s", sudoku.Serialize(got.Grid))
	}
}

func TestPropagateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := Propagate(ctx, sampleGrid(t), config.Defaults())
	if got.Solved {
		t.Errorf("cancelled context reported solved")
	}
}

func TestCountSolutions(t *testing.T) {
	cases := []struct {
		name  string
		grid  func(t *testing.T) sudoku.Grid
		limit int
		want  int
	}{
		{"sample unique", sampleGrid, 2, 1},
		{"solved unique", solutionGrid, 2, 1},
		{"unsolvable zero", unsolvableGrid, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountSolutions(tc.grid(t), tc.limit); got != tc.want {
				t.Errorf("got %d want %d", got, tc.want)
			}
		})
	}
	t.Run("empty grid capped", func(t *testing.T) {
		if got := CountSolutions(sudoku.Grid{}, 2); got != 2 {
			t.Errorf("got %d want 2 at the cap", got)
		}
	})
}

func TestPropagationFill(t *testing.T) {
	if got := PropagationFill(solutionGrid(t)); got != 0 {
		t.Errorf("full grid fill: got %d want 0", got)
	}
	got := PropagationFill(sampleGrid(t))
	if got < 0 || got > 49 {
		t.Errorf("sample fill out of range: %d", got)
	}
}

func TestRegistry(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatalf("registry empty")
	}
	if all[0].ID != "propagate" {
		t.Errorf("first method: got %s", all[0].ID)
	}
	if all[0].Label != config.LabelPropagate {
		t.Errorf("label: got %s", all[0].Label)
	}
	m, ok := ByID("propagate")
	if !ok || m.ID != "propagate" {
		t.Errorf("ByID propagate: got %+v ok %v", m, ok)
	}
	if _, ok := ByID("nope"); ok {
		t.Errorf("ByID nope found")
	}
}
