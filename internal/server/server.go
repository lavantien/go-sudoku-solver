package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/policy"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/store"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
	"github.com/lavantien/go-sudoku-solver/web"
)

type RunStore interface {
	CountRuns() (int64, error)
	InsertRun(input81, source string, methods []store.MethodRow) (int64, error)
	ListRuns(limit, offset int) ([]store.RunSummary, error)
	GetRun(id int64) (*store.RunDetail, error)
	InsertPuzzle(grid81 string, clues int, seed int64) (int64, error)
	ListPuzzles(limit, offset int) ([]store.Puzzle, error)
}

type Server struct {
	cfg       config.Config
	store     RunStore
	log       *slog.Logger
	http      *http.Server
	handler   http.Handler
	templates *templates
}

func New(ctx context.Context, cfg config.Config, st RunStore, logger *slog.Logger) *Server {
	logging.Enter(logger, "server.new", "addr", cfg.Server.Addr)
	s := &Server{
		cfg:       cfg,
		store:     st,
		log:       logger,
		templates: parseTemplates(),
	}
	s.handler = s.routes()
	s.http = &http.Server{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		Handler:      s.handler,
		BaseContext:  func(net.Listener) context.Context { return ctx },
	}
	s.seedIfEmpty(ctx)
	logging.Exit(logger, "server.new")
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) Serve(ln net.Listener) error {
	logging.Enter(s.log, "server.serve", "addr", ln.Addr().String())
	err := s.http.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.Fail(s.log, "server.serve", err, "addr", ln.Addr().String())
		return err
	}
	logging.Exit(s.log, "server.serve", "addr", ln.Addr().String())
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	logging.Enter(s.log, "server.shutdown")
	if err := s.http.Shutdown(ctx); err != nil {
		logging.Fail(s.log, "server.shutdown", err)
		return err
	}
	logging.Exit(s.log, "server.shutdown")
	return nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+config.HomePath+"{$}", s.handleHome)
	mux.HandleFunc("GET "+config.GeneratePagePath, s.handleGeneratePage)
	mux.HandleFunc("GET "+config.HistoryPagePath, s.handleHistoryPage)
	mux.HandleFunc("GET "+config.RunPageRoute, s.handleRunPage)
	mux.HandleFunc("POST "+config.FragmentSolvePath, s.handleSolveFragment)
	mux.HandleFunc("POST "+config.FragmentGenPath, s.handleGenerateFragment)
	mux.HandleFunc("POST "+config.APISolvePath, s.handleAPISolve)
	mux.HandleFunc("GET "+config.APIRunsPath, s.handleAPIRuns)
	mux.HandleFunc("GET "+config.APIRunRoute, s.handleAPIRun)
	mux.HandleFunc("POST "+config.APIGeneratePath, s.handleAPIGenerate)
	mux.HandleFunc("GET "+config.APIPuzzlesPath, s.handleAPIPuzzles)
	mux.HandleFunc("GET "+config.APIHealthPath, s.handleAPIHealth)
	mux.HandleFunc("GET "+config.APIVersionPath, s.handleAPIVersion)
	mux.Handle("GET "+config.StaticMountPath, http.StripPrefix(config.StaticMountPath, http.FileServerFS(s.staticFS())))
	mux.HandleFunc(config.APIMountPath, s.handleAPINotFound)
	return s.middleware(mux)
}

func (s *Server) staticFS() fs.FS {
	sub, err := fs.Sub(web.Static, staticDir)
	if err != nil {
		logging.Fail(s.log, "server.static_fs", err)
		return web.Static
	}
	return sub
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.Enter(s.log, "http.request", "method", r.Method, "path", r.URL.Path)
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status >= http.StatusInternalServerError {
			logging.Fail(s.log, "http.request", fmt.Errorf("status %d", sw.status), "method", r.Method, "path", r.URL.Path)
			return
		}
		logging.Exit(s.log, "http.request", "method", r.Method, "path", r.URL.Path, "status", sw.status)
	})
}

func (s *Server) solveAll(ctx context.Context, g sudoku.Grid) []store.MethodRow {
	methods := solver.All()
	rows := make([]store.MethodRow, 0, len(methods))
	for _, m := range methods {
		res := s.solveMethod(ctx, m, g)
		rows = append(rows, store.MethodRow{
			Method: m.ID,
			Grid81: sudoku.Serialize(res.Grid),
			Steps:  res.Steps,
			TimeNS: res.Dur.Nanoseconds(),
			Solved: res.Solved,
		})
	}
	return rows
}

func (s *Server) solveMethod(ctx context.Context, m solver.Method, g sudoku.Grid) solver.Result {
	if m.ID != config.MethodIDs[3] {
		return m.Fn(ctx, g, s.cfg)
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
	return solver.Race(ctx, g, s.cfg, racers...)
}

func (s *Server) solveAndPersist(ctx context.Context, g sudoku.Grid, source string) (int64, []store.MethodRow, error) {
	logging.Enter(s.log, "server.solve_and_persist", "source", source)
	rows := s.solveAll(ctx, g)
	id, err := s.store.InsertRun(sudoku.Serialize(g), source, rows)
	if err != nil {
		logging.Fail(s.log, "server.solve_and_persist", err, "source", source)
		return 0, nil, fmt.Errorf("server persist run: %w", err)
	}
	logging.Exit(s.log, "server.solve_and_persist", "id", id, "source", source)
	return id, rows, nil
}

func (s *Server) seedIfEmpty(ctx context.Context) {
	logging.Enter(s.log, "server.seed")
	n, err := s.store.CountRuns()
	if err != nil {
		logging.Fail(s.log, "server.seed", err)
		return
	}
	if n > 0 {
		logging.Exit(s.log, "server.seed", "seeded", false, "runs", n)
		return
	}
	g, err := sudoku.Parse81(puzzle.Sample81)
	if err != nil {
		logging.Fail(s.log, "server.seed", err)
		return
	}
	if _, _, err := s.solveAndPersist(ctx, g, config.SourceSeed); err != nil {
		logging.Fail(s.log, "server.seed", err)
		return
	}
	logging.Exit(s.log, "server.seed", "seeded", true)
}
