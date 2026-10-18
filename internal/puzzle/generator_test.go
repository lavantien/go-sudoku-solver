package puzzle

import (
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestGenerateDeterministic(t *testing.T) {
	a, err := Generate(40, 7, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(40, 7, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	if a.Grid != b.Grid || a.Clues != b.Clues {
		t.Errorf("same seed differs")
	}
}

func TestGenerateUniqueSolution(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3} {
		res, err := Generate(35, seed, solver.CountSolutions)
		if err != nil {
			t.Fatal(err)
		}
		if got := solver.CountSolutions(res.Grid, 2); got != 1 {
			t.Errorf("seed %d: %d solutions", seed, got)
		}
	}
}

func TestGenerateClueTargets(t *testing.T) {
	cases := []struct {
		target int
		lo     int
	}{
		{40, 40},
		{30, 26},
		{17, 20},
	}
	for _, tc := range cases {
		t.Run("target", func(t *testing.T) {
			res, err := Generate(tc.target, 5, solver.CountSolutions)
			if err != nil {
				t.Fatal(err)
			}
			if res.Clues > tc.target || res.Clues < tc.lo {
				t.Errorf("target %d: got %d clues, want between %d and %d", tc.target, res.Clues, tc.lo, tc.target)
			}
			counted := 0
			for i := range res.Grid {
				if res.Grid[i] != 0 {
					counted++
				}
			}
			if counted != res.Clues {
				t.Errorf("clue count mismatch: struct says %d, grid has %d", res.Clues, counted)
			}
		})
	}
}

func TestGenerateRoundTrip(t *testing.T) {
	res, err := Generate(40, 11, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	s := sudoku.Serialize(res.Grid)
	if len(s) != 81 || strings.ContainsAny(s, "0") {
		t.Errorf("serialization invalid: %s", s)
	}
	back, err := sudoku.Parse81(s)
	if err != nil {
		t.Fatal(err)
	}
	if back != res.Grid {
		t.Errorf("round trip changed the grid")
	}
}

func TestGenerateDifferentSeedsDiffer(t *testing.T) {
	a, err := Generate(40, 1, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(40, 2, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	if a.Grid == b.Grid {
		t.Errorf("different seeds produced identical grids")
	}
}

func TestGenerateInvalidClues(t *testing.T) {
	for _, clues := range []int{16, 82, 0, -1} {
		if _, err := Generate(clues, 1, solver.CountSolutions); err == nil {
			t.Errorf("clues %d accepted", clues)
		}
	}
}

func TestGenerateNilCheckerRejected(t *testing.T) {
	if _, err := Generate(40, 1, nil); err == nil {
		t.Errorf("nil checker accepted")
	}
}
