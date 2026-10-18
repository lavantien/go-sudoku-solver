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
	return []Method{
		{ID: config.MethodIDs[0], Label: config.LabelPropagate, Fn: Propagate},
		{ID: config.MethodIDs[1], Label: config.LabelCover, Fn: Cover},
	}
}

func ByID(id string) (Method, bool) {
	for _, m := range All() {
		if m.ID == id {
			return m, true
		}
	}
	return Method{}, false
}
