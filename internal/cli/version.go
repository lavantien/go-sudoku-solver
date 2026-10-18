package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
)

func init() {
	Register(Command{Name: config.CmdVersion, Usage: config.UsageVersion, Run: runVersion})
}

func runVersion(ctx context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, "version")
	_, _ = fmt.Fprintf(os.Stdout, config.VersionFmt, config.BinName, config.Version)
	logging.Exit(logger, "version")
	return nil
}
