package solver

import (
	"context"
	"sync"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func Race(ctx context.Context, g sudoku.Grid, c config.Config, racers ...Solver) Result {
	logger := logging.New(c.LogLevel)
	logging.Enter(logger, "race", "clues", clueCount(g))
	start := time.Now()
	if len(racers) == 0 {
		racers = []Solver{Propagate, Cover, Anneal}
	}
	var out Result
	if c.Workers == 1 {
		out = raceSequential(ctx, g, c, racers)
	} else {
		out = raceParallel(ctx, g, c, racers)
	}
	out.Dur = time.Since(start)
	if out.Solved {
		logging.Exit(logger, "race", "steps", out.Steps)
	} else {
		logging.Fail(logger, "race", errUnsolved, "steps", out.Steps)
	}
	return out
}

func raceSequential(ctx context.Context, g sudoku.Grid, c config.Config, racers []Solver) Result {
	steps := 0
	for _, fn := range racers {
		if ctx.Err() != nil {
			break
		}
		res := fn(ctx, g, c)
		steps += res.Steps
		if res.Solved {
			return Result{Grid: res.Grid, Steps: steps, Solved: true}
		}
	}
	return Result{Grid: g, Steps: steps, Solved: false}
}

func raceParallel(ctx context.Context, g sudoku.Grid, c config.Config, racers []Solver) Result {
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan Result, len(racers))
	var wg sync.WaitGroup
	for _, fn := range racers {
		wg.Add(1)
		go func(f Solver) {
			defer wg.Done()
			results <- f(rctx, g, c)
		}(fn)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	var winner Result
	steps := 0
	first := true
	for res := range results {
		steps += res.Steps
		if first {
			winner = res
			first = false
		}
		if res.Solved && !winner.Solved {
			winner = res
			cancel()
		}
	}
	return Result{Grid: winner.Grid, Steps: steps, Solved: winner.Solved}
}
