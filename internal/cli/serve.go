package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/server"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

func init() {
	Register(Command{Name: config.CmdServe, Usage: config.UsageServe, Run: runServe})
}

func serveContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
}

func runServe(ctx context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, "serve")
	fs := flag.NewFlagSet(config.CmdServe, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String(config.FlagAddr, cfg.Server.Addr, config.FlagAddrUsage)
	db := fs.String(config.FlagDB, cfg.DBPath, config.FlagDBUsage)
	if err := fs.Parse(args); err != nil {
		logging.Fail(logger, "serve", err)
		return err
	}
	cfg.Server.Addr = *addr
	cfg.DBPath = *db
	if err := cfg.Validate(); err != nil {
		logging.Fail(logger, "serve", err)
		return err
	}
	sctx, stopSignals := serveContext(ctx)
	defer stopSignals()
	st, err := store.Open(cfg.DBPath, config.DBBusyTimeout)
	if err != nil {
		logging.Fail(logger, "serve", err, "db", cfg.DBPath)
		return err
	}
	srv := server.New(sctx, cfg, st, logger)
	ln, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		logging.Fail(logger, "serve", err, "addr", cfg.Server.Addr)
		_ = st.Close()
		return err
	}
	_, _ = fmt.Fprintf(os.Stderr, config.ServeAddrLineFmt, config.BinName, ln.Addr().String())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	select {
	case err := <-errCh:
		if err != nil {
			logging.Fail(logger, "serve", err)
		} else {
			logging.Exit(logger, "serve")
		}
		_ = st.Close()
		return err
	case <-sctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logging.Fail(logger, "serve", err)
		}
		if serveErr := <-errCh; serveErr != nil {
			logging.Fail(logger, "serve", serveErr)
		}
		_ = st.Close()
		logging.Exit(logger, "serve", "reason", "context canceled")
		return nil
	}
}
