package solver

import (
	"context"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func Race(ctx context.Context, g sudoku.Grid, c config.Config, racers ...Solver) Result {
	return Result{}
}
