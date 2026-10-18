package solver

import (
	"context"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func benchGrid(b *testing.B, grid81 string) sudoku.Grid {
	b.Helper()
	g, err := sudoku.Parse81(grid81)
	if err != nil {
		b.Fatalf("sample does not parse: %v", err)
	}
	return g
}

func benchSolver(b *testing.B, fn Solver, grid81 string) {
	b.Helper()
	g := benchGrid(b, grid81)
	cfg := config.Defaults()
	b.ResetTimer()
	for range b.N {
		fn(context.Background(), g, cfg)
	}
}

func BenchmarkPropagateExtreme(b *testing.B) {
	benchSolver(b, Propagate, puzzle.SampleExtreme81)
}

func BenchmarkCoverExtreme(b *testing.B) {
	benchSolver(b, Cover, puzzle.SampleExtreme81)
}

func BenchmarkAnnealExtreme(b *testing.B) {
	benchSolver(b, Anneal, puzzle.SampleExtreme81)
}

func BenchmarkRaceExtreme(b *testing.B) {
	benchSolver(b, func(ctx context.Context, g sudoku.Grid, c config.Config) Result {
		return Race(ctx, g, c)
	}, puzzle.SampleExtreme81)
}

func BenchmarkPropagateClassic(b *testing.B) {
	benchSolver(b, Propagate, puzzle.Sample81)
}
