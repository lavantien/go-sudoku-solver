package solver

import (
	"context"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

type Result struct {
	Grid   sudoku.Grid
	Steps  int
	Dur    time.Duration
	Solved bool
}

type Solver func(ctx context.Context, g sudoku.Grid, c config.Config) Result

type Method struct {
	ID    string
	Label string
	Fn    Solver
}

func All() []Method {
	return nil
}

func ByID(id string) (Method, bool) {
	return Method{}, false
}
