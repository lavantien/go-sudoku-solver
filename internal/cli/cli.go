package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

type Command struct {
	Name  string
	Usage string
	Run   func(ctx context.Context, cfg config.Config, args []string) error
}

var registry = map[string]Command{}

var errSilent = errors.New(config.ErrSilentExit)

func Register(c Command) { registry[c.Name] = c }

func Lookup(name string) (Command, bool) {
	c, ok := registry[name]
	return c, ok
}

func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func Main(args []string) int {
	if len(args) == 0 {
		args = []string{config.CmdSolve}
	}
	name := args[0]
	cmd, ok := Lookup(name)
	if !ok {
		printUsage(os.Stderr)
		return 2
	}
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, config.RunErrFmt, config.BinName, name, err)
		return 1
	}
	if err := cmd.Run(context.Background(), cfg, args[1:]); err != nil {
		if errors.Is(err, errSilent) {
			return 1
		}
		fmt.Fprintf(os.Stderr, config.RunErrFmt, config.BinName, name, err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, config.UsageHeaderFmt, config.BinName)
	for _, n := range Names() {
		c, _ := Lookup(n)
		_, _ = fmt.Fprintf(w, config.UsageCmdFmt, c.Name, c.Usage)
	}
}
