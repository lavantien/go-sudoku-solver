package solver

import (
	"context"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func Propagate(ctx context.Context, g sudoku.Grid, c config.Config) Result {
	return Result{}
}

func CountSolutions(g sudoku.Grid, limit int) int {
	return 0
}

func PropagationFill(g sudoku.Grid) int {
	return 0
}
