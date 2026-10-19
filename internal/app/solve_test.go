package app

import (
	"context"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/policy"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
)

func methodByID(t *testing.T, id string) solver.Method {
	t.Helper()
	for _, m := range solver.All() {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("method %s not registered", id)
	return solver.Method{}
}

func racersFor(t *testing.T, order []string) []solver.Solver {
	t.Helper()
	racers := make([]solver.Solver, 0, len(order))
	for _, id := range order {
		if r, ok := solver.ByID(id); ok {
			racers = append(racers, r.Fn)
		}
	}
	return racers
}

func TestSolveMethodRunsOwnSolverOutsideTheRaceSlot(t *testing.T) {
	g := grid81(t, puzzle.Sample81)
	cfg := config.Defaults()
	cfg.Workers = 1
	anneal := methodByID(t, config.MethodIDs[2])
	direct := anneal.Fn(context.Background(), g, cfg)
	got := solveMethod(context.Background(), cfg, anneal, g)
	if got.Solved {
		t.Fatalf("anneal slot reported solved on the adversarial sample")
	}
	if got.Solved != direct.Solved || got.Steps != direct.Steps || got.Grid != direct.Grid {
		t.Fatalf("anneal dispatch: got solved=%v steps=%d, want the direct result solved=%v steps=%d", got.Solved, got.Steps, direct.Solved, direct.Steps)
	}
}

func TestSolveMethodRacesThePolicyOrder(t *testing.T) {
	g := grid81(t, puzzle.Anneal81)
	cfg := config.Defaults()
	cfg.Workers = 1
	race := methodByID(t, config.MethodIDs[3])
	want := solver.Race(context.Background(), g, cfg, racersFor(t, policy.ResolveOrder(g))...)
	got := solveMethod(context.Background(), cfg, race, g)
	if got.Solved != want.Solved || got.Steps != want.Steps || got.Grid != want.Grid {
		t.Fatalf("race dispatch: got solved=%v steps=%d, want the policy-ordered race solved=%v steps=%d", got.Solved, got.Steps, want.Solved, want.Steps)
	}
}
