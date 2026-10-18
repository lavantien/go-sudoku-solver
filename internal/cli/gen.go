package cli

import (
	"context"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func init() {
	Register(Command{Name: config.CmdGen, Usage: config.UsageGen, Run: runGen})
}

func runGen(ctx context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, "gen")
	fs := flag.NewFlagSet(config.CmdGen, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	clues := fs.Int(config.FlagClues, cfg.Gen.Clues, config.FlagCluesUsage)
	seed := fs.Uint64(config.FlagSeed, cfg.Gen.Seed, config.FlagSeedUsage)
	if err := fs.Parse(args); err != nil {
		logging.Fail(logger, "gen", err)
		return err
	}
	res, err := puzzle.Generate(*clues, *seed, solver.CountSolutions)
	if err != nil {
		logging.Fail(logger, "gen", err)
		return err
	}
	_, _ = io.WriteString(os.Stdout, inputLines(res.Grid))
	logging.Exit(logger, "gen", "clues", res.Clues)
	return nil
}

func inputLines(g sudoku.Grid) string {
	var b strings.Builder
	for r := range config.Size {
		for c := range config.Size {
			v := g[r*config.Size+c]
			if v == config.EmptyDigit {
				b.WriteByte(config.EmptyChar)
			} else {
				b.WriteString(strconv.Itoa(int(v)))
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
