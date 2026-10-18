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

const (
	cmdCoverage    = "coverage"
	usageCoverage  = "refresh the readme coverage badge: -profile -readme"
	flagProfile    = "profile"
	flagReadme     = "readme"
	flagProfileUse = "Go cover profile path"
	flagReadmeUse  = "readme path to update"
	defaultProfile = "coverage.out"
	defaultReadme  = "readme.md"
)

func init() {
	Register(Command{Name: cmdCoverage, Usage: usageCoverage, Run: runCoverage})
}

func runCoverage(_ context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, cmdCoverage)
	fs := flag.NewFlagSet(cmdCoverage, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profile := fs.String(flagProfile, defaultProfile, flagProfileUse)
	readme := fs.String(flagReadme, defaultReadme, flagReadmeUse)
	if err := fs.Parse(args); err != nil {
		logging.Fail(logger, cmdCoverage, err)
		return err
	}
	if fs.NArg() > 0 {
		err := fmt.Errorf("unexpected argument %q", fs.Arg(0))
		logging.Fail(logger, cmdCoverage, err)
		return err
	}
	pct, err := coverageRun(*profile, *readme)
	if err != nil {
		logging.Fail(logger, cmdCoverage, err)
		return err
	}
	logging.Exit(logger, cmdCoverage, "pct", fmt.Sprintf("%.1f", pct))
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
