package solver

import (
	"context"
	"math/rand/v2"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
	"go.uber.org/goleak"
)

func TestRaceSequentialSkipsRacersAfterCancellation(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := config.Defaults()
	cfg.Workers = 1
	got := Race(ctx, sampleGrid(t), cfg)
	if got.Solved {
		t.Errorf("cancelled sequential race reported solved")
	}
	if got.Steps != 0 {
		t.Errorf("steps: got %d want 0, no racer may run after cancellation", got.Steps)
	}
}

func TestRaceSequentialReportsFailureWhenNoRacerSolves(t *testing.T) {
	defer goleak.VerifyNone(t)
	cfg := config.Defaults()
	cfg.Workers = 1
	got := Race(context.Background(), unsolvableGrid(t), cfg)
	if got.Solved {
		t.Errorf("unsolvable grid reported solved")
	}
	if got.Steps <= 0 {
		t.Errorf("steps: got %d want the expended racers", got.Steps)
	}
}

func TestRaceParallelLaterSolvedWinnerOverridesUnsolvedFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		quickFail := func(_ context.Context, g sudoku.Grid, _ config.Config) Result {
			time.Sleep(10 * time.Millisecond)
			return Result{Grid: g, Steps: 1, Solved: false}
		}
		slowerSolve := func(_ context.Context, g sudoku.Grid, _ config.Config) Result {
			time.Sleep(30 * time.Millisecond)
			return Result{Grid: g, Steps: 2, Solved: true}
		}
		got := Race(context.Background(), sudoku.Grid{}, config.Defaults(), quickFail, slowerSolve)
		if !got.Solved {
			t.Fatalf("later solved racer must win the race")
		}
		if got.Steps != 3 {
			t.Errorf("steps: got %d want 3 summed across racers", got.Steps)
		}
	})
}

func TestAnnealMidLoopCancellationStopsTheWalk(t *testing.T) {
	cfg := config.Defaults()
	cfg.Anneal.Restarts = 1
	cfg.Anneal.Iterations = 50000000
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)
	got := Anneal(ctx, unsolvableGrid(t), cfg)
	if got.Solved {
		t.Errorf("unsolvable grid reported solved")
	}
}

func TestPickMoveReturnsFalseOnSolvedGrid(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	if move, ok := pickMove(solutionGrid(t), [config.Size][]int{}, rng); ok {
		t.Errorf("solved grid: got move %v want none", move)
	}
}

func TestPickMoveSkipsRowsWithoutUsableCells(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	conflicted := solutionGrid(t)
	conflicted[0] = conflicted[1]
	if move, ok := pickMove(conflicted, [config.Size][]int{}, rng); ok {
		t.Errorf("empty free rows: got move %v want none", move)
	}
	var free [config.Size][]int
	free[0] = []int{2, 3}
	if move, ok := pickMove(conflicted, free, rng); ok {
		t.Errorf("free row without the conflicted cell: got move %v want none", move)
	}
}

func TestCountSolutionsEdges(t *testing.T) {
	if got := CountSolutions(sampleGrid(t), 0); got != 1 {
		t.Errorf("limit 0: got %d want 1 clamped to a single count", got)
	}
	dup := solutionGrid(t)
	dup[0] = dup[1]
	if got := CountSolutions(dup, 2); got != 0 {
		t.Errorf("duplicate digit in a full row: got %d want 0", got)
	}
}
