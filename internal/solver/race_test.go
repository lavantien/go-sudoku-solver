package solver

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
	"go.uber.org/goleak"
)

func TestRaceSolvesSample(t *testing.T) {
	defer goleak.VerifyNone(t)
	got := Race(context.Background(), sampleGrid(t), config.Defaults())
	if !got.Solved {
		t.Fatalf("race did not solve the sample")
	}
	want := solutionGrid(t)
	if got.Grid != want {
		t.Errorf("grid: got %s want %s", sudoku.Serialize(got.Grid), sudoku.Serialize(want))
	}
	if got.Steps <= 0 {
		t.Errorf("steps: got %d", got.Steps)
	}
}

func TestRaceUnsolvable(t *testing.T) {
	defer goleak.VerifyNone(t)
	got := Race(context.Background(), unsolvableGrid(t), config.Defaults())
	if got.Solved {
		t.Fatalf("unsolvable reported solved")
	}
	if got.Steps <= 0 {
		t.Errorf("steps: got %d want at least the expended racers", got.Steps)
	}
}

func TestRaceCancelledContext(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := Race(ctx, sampleGrid(t), config.Defaults())
	if got.Solved {
		t.Errorf("cancelled context reported solved")
	}
}

func TestRaceSynctestFirstFinisherWins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fast := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
			time.Sleep(10 * time.Millisecond)
			out := g
			out[0] = 1
			return Result{Grid: out, Steps: 5, Solved: true}
		}
		mid := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
			time.Sleep(30 * time.Millisecond)
			out := g
			out[0] = 2
			return Result{Grid: out, Steps: 7, Solved: true}
		}
		slow := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
			time.Sleep(50 * time.Millisecond)
			out := g
			out[0] = 3
			return Result{Grid: out, Steps: 9, Solved: false}
		}
		start := time.Now()
		got := Race(context.Background(), sudoku.Grid{}, config.Defaults(), fast, mid, slow)
		if !got.Solved || got.Grid[0] != 1 {
			t.Errorf("winner: got solved %v grid0 %d want the 10ms racer", got.Solved, got.Grid[0])
		}
		if got.Steps != 21 {
			t.Errorf("total steps: got %d want 21 summed across racers", got.Steps)
		}
		if got.Dur != 50*time.Millisecond {
			t.Errorf("duration: got %v want 50ms of drained wall time", got.Dur)
		}
		if time.Since(start) != 50*time.Millisecond {
			t.Errorf("wall: got %v", time.Since(start))
		}
	})
}

func TestRaceUnwindsStragglersOnWinnerCancel(t *testing.T) {
	defer goleak.VerifyNone(t)
	g, err := sudoku.Parse81(puzzle.SampleExtreme81)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	start := time.Now()
	got := Race(context.Background(), g, cfg)
	wall := time.Since(start)
	if !got.Solved {
		t.Fatalf("race did not solve the extreme sample")
	}
	if wall >= 300*time.Millisecond {
		t.Errorf("wall: got %v want under 300ms, stragglers must unwind on the winner's cancel", wall)
	}
	if budget := cfg.Anneal.Restarts * cfg.Anneal.Iterations; got.Steps >= budget {
		t.Errorf("steps: got %d want under the anneal budget %d, stragglers must contribute partial steps", got.Steps, budget)
	}
}

func TestRaceSequentialWorkersOne(t *testing.T) {
	cfg := config.Defaults()
	cfg.Workers = 1
	calls := 0
	loser := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
		calls++
		return Result{Grid: g, Steps: 3, Solved: false}
	}
	winner := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
		calls++
		return Result{Grid: g, Steps: 4, Solved: true}
	}
	never := func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
		calls++
		return Result{Grid: g, Steps: 100, Solved: true}
	}
	got := Race(context.Background(), sudoku.Grid{}, cfg, loser, winner, never)
	if !got.Solved || got.Steps != 7 {
		t.Errorf("sequential: got solved %v steps %d", got.Solved, got.Steps)
	}
	if calls != 2 {
		t.Errorf("calls: got %d want 2, the loser then the winner", calls)
	}
}

func TestRegistryHasRace(t *testing.T) {
	all := All()
	if len(all) != 4 {
		t.Fatalf("registry has %d methods want 4", len(all))
	}
	if all[3].ID != "race" || all[3].Label != config.LabelRace {
		t.Errorf("fourth method: got %s %s", all[3].ID, all[3].Label)
	}
	res := all[3].Fn(context.Background(), sampleGrid(t), config.Defaults())
	if !res.Solved {
		t.Errorf("registry race fn did not solve the sample")
	}
}
