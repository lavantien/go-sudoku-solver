package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/coverbadge"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
)

func init() {
	Register(Command{Name: config.CmdCoverage, Usage: config.UsageCoverage, Run: runCoverage})
}

func runCoverage(_ context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, config.CmdCoverage)
	fs := flag.NewFlagSet(config.CmdCoverage, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profile := fs.String(config.FlagProfile, config.DefaultProfile, config.FlagProfileUsage)
	readme := fs.String(config.FlagReadme, config.DefaultReadme, config.FlagReadmeUsage)
	if err := fs.Parse(args); err != nil {
		logging.Fail(logger, config.CmdCoverage, err)
		return err
	}
	if fs.NArg() > 0 {
		err := fmt.Errorf("unexpected argument %q", fs.Arg(0))
		logging.Fail(logger, config.CmdCoverage, err)
		return err
	}
	pct, err := coverageRun(*profile, *readme)
	if err != nil {
		logging.Fail(logger, config.CmdCoverage, err)
		return err
	}
	logging.Exit(logger, config.CmdCoverage, "pct", fmt.Sprintf("%.1f", pct))
	return nil
}

func coverageRun(profilePath, readmePath string) (float64, error) {
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		return 0, err
	}
	pct, err := coverbadge.Pct(string(profile))
	if err != nil {
		return 0, err
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		return 0, err
	}
	updated := coverbadge.Render(string(readme), pct)
	if updated != string(readme) {
		if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
			return pct, err
		}
		fmt.Printf("coverage %.1f%%: readme badge updated\n", pct)
	} else {
		fmt.Printf("coverage %.1f%%: readme badge already current\n", pct)
	}
	return pct, nil
}
