package app

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/policy"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func Run(ctx context.Context, cfg config.Config, in io.Reader, out io.Writer, errOut io.Writer) int {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, "app.run")
	g, err := sudoku.ParseInput(in)
	if err != nil {
		logging.Fail(logger, "app.run", err)
		_, _ = fmt.Fprintln(errOut, config.InputErrorPrefix, err)
		return 1
	}
	var b strings.Builder
	b.WriteString(config.InitialHeader + "\n")
	b.WriteString(sudoku.RenderGrid(g))
	for _, m := range solver.All() {
		writeBlock(&b, m, solveMethod(ctx, cfg, m, g))
	}
	_, _ = io.WriteString(out, b.String())
	logging.Exit(logger, "app.run")
	return 0
}

func solveMethod(ctx context.Context, cfg config.Config, m solver.Method, g sudoku.Grid) solver.Result {
	if m.ID != config.MethodIDs[3] {
		return m.Fn(ctx, g, cfg)
	}
	order := policy.ResolveOrder(g)
	if len(order) == 0 {
		order = policy.FixedOrder()
	}
	racers := make([]solver.Solver, 0, len(order))
	for _, id := range order {
		if r, ok := solver.ByID(id); ok {
			racers = append(racers, r.Fn)
		}
	}
	return solver.Race(ctx, g, cfg, racers...)
}

func writeBlock(b *strings.Builder, m solver.Method, res solver.Result) {
	b.WriteString(config.Separator + "\n")
	b.WriteString(sudoku.RenderGrid(res.Grid))
	_, _ = fmt.Fprintf(b, config.LabelFmt, m.Label)
	_, _ = fmt.Fprintf(b, config.StepFmt, res.Steps)
	_, _ = fmt.Fprintf(b, config.TimeFmt, res.Dur.Seconds())
	b.WriteString(config.Separator + "\n")
}
