package solver

import (
	"context"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const (
	fuzzTimeout        = 2 * time.Second
	fuzzAnnealRestarts = 1
	fuzzAnnealSteps    = 200
	fuzzCoverNodeCap   = 4096
)

func fuzzGridBytes(f *testing.F, s81 string) {
	g, err := sudoku.Parse81(s81)
	if err != nil {
		f.Fatal(err)
	}
	b := make([]byte, config.Cells)
	for i, v := range g {
		b[i] = byte(v)
	}
	f.Add(b)
}

func fuzzCfg() config.Config {
	cfg := config.Defaults()
	cfg.Anneal.Restarts = fuzzAnnealRestarts
	cfg.Anneal.Iterations = fuzzAnnealSteps
	cfg.CoverNodeCap = fuzzCoverNodeCap
	return cfg
}

func FuzzSolverCore(f *testing.F) {
	fuzzGridBytes(f, puzzle.Sample81)
	fuzzGridBytes(f, puzzle.Unsolvable81)
	fuzzGridBytes(f, puzzle.Anneal81)
	fuzzGridBytes(f, puzzle.SampleSolution81)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		var g sudoku.Grid
		for i := range config.Cells {
			if i < len(data) {
				g[i] = data[i] % 10
			}
		}
		cfg := fuzzCfg()
		budget := cfg.Anneal.Restarts * cfg.Anneal.Iterations
		for _, m := range All() {
			ctx, cancel := context.WithTimeout(context.Background(), fuzzTimeout)
			res := m.Fn(ctx, g, cfg)
			cancel()
			if res.Solved {
				if !puzzle.ValidSolution(res.Grid) {
					t.Errorf("%s: invalid solution %s", m.ID, sudoku.Serialize(res.Grid))
				}
				if !puzzle.GivensPreserved(res.Grid, g) {
					t.Errorf("%s: givens not preserved against %s", m.ID, sudoku.Serialize(g))
				}
			}
			if m.ID == config.MethodIDs[2] && res.Steps > budget {
				t.Errorf("%s: steps %d exceed budget %d", m.ID, res.Steps, budget)
			}
		}
	})
}
