package cli

import (
	"context"
	"os"

	"github.com/lavantien/go-sudoku-solver/internal/app"
	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func init() {
	Register(Command{Name: config.CmdSolve, Usage: config.UsageSolve, Run: runSolve})
}

func runSolve(ctx context.Context, cfg config.Config, args []string) error {
	if code := app.Run(ctx, cfg, os.Stdin, os.Stdout, os.Stderr); code != 0 {
		return errSilent
	}
	return nil
}
