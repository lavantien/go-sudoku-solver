package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/store"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

type solveRequest struct {
	Grid81 string `json:"grid81"`
}

type methodOut struct {
	Method     string  `json:"method"`
	Grid81     string  `json:"grid81"`
	Steps      int     `json:"steps"`
	TimeNS     int64   `json:"time_ns"`
	DurationMS float64 `json:"duration_ms"`
	Solved     bool    `json:"solved"`
}

type solveOut struct {
	ID      int64       `json:"id"`
	Input81 string      `json:"input81"`
	Source  string      `json:"source"`
	Methods []methodOut `json:"methods"`
}

type runSummaryOut struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
	Input81   string `json:"input81"`
	Source    string `json:"source"`
	SolvedAny bool   `json:"solved_any"`
}

type runDetailOut struct {
	runSummaryOut
	Methods []methodOut `json:"methods"`
}

type runsOut struct {
	Runs []runSummaryOut `json:"runs"`
}

type puzzleOut struct {
	ID        int64  `json:"id"`
	Grid81    string `json:"grid81"`
	Clues     int    `json:"clues"`
	Seed      int64  `json:"seed"`
	CreatedAt string `json:"created_at"`
}

type puzzlesOut struct {
	Puzzles []puzzleOut `json:"puzzles"`
}

type generateRequest struct {
	Clues int    `json:"clues"`
	Seed  uint64 `json:"seed"`
}

type healthOut struct {
	Status string `json:"status"`
}

type versionOut struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func methodsOutFromRows(rows []store.MethodRow) []methodOut {
	out := make([]methodOut, 0, len(rows))
	for _, m := range rows {
		out = append(out, methodOut{
			Method:     m.Method,
			Grid81:     m.Grid81,
			Steps:      m.Steps,
			TimeNS:     m.TimeNS,
			DurationMS: float64(m.TimeNS) / float64(time.Millisecond),
			Solved:     m.Solved,
		})
	}
	return out
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", config.JSONContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logging.Fail(s.log, "server.write_json", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{config.ErrorKey: msg})
}

func (s *Server) storeFailJSON(w http.ResponseWriter, err error) {
	logging.Fail(s.log, "server.store", err)
	s.writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *Server) handleAPISolve(w http.ResponseWriter, r *http.Request) {
	var req solveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	g, err := sudoku.Parse81(req.Grid81)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, rows, err := s.solveAndPersist(r.Context(), g, config.SourceAPI)
	if err != nil {
		s.storeFailJSON(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, solveOut{
		ID:      id,
		Input81: sudoku.Serialize(g),
		Source:  config.SourceAPI,
		Methods: methodsOutFromRows(rows),
	})
}

func (s *Server) handleAPIRuns(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := s.parsePageParams(w, r)
	if !ok {
		return
	}
	runs, err := s.store.ListRuns(limit, offset)
	if err != nil {
		s.storeFailJSON(w, err)
		return
	}
	out := runsOut{Runs: make([]runSummaryOut, 0, len(runs))}
	for _, run := range runs {
		out.Runs = append(out.Runs, runSummaryOut{
			ID:        run.ID,
			CreatedAt: run.CreatedAt,
			Input81:   run.Input81,
			Source:    run.Source,
			SolvedAny: run.SolvedAny,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAPIRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "run not found")
		return
	}
	detail, err := s.store.GetRun(id)
	if errors.Is(err, store.ErrNotFound) {
		s.writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		s.storeFailJSON(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, runDetailOut{
		runSummaryOut: runSummaryOut{
			ID:        detail.ID,
			CreatedAt: detail.CreatedAt,
			Input81:   detail.Input81,
			Source:    detail.Source,
			SolvedAny: detail.SolvedAny,
		},
		Methods: methodsOutFromRows(detail.Methods),
	})
}

func (s *Server) handleAPIGenerate(w http.ResponseWriter, r *http.Request) {
	req := generateRequest{Clues: s.cfg.Gen.Clues, Seed: s.cfg.Gen.Seed}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Seed > uint64(config.MaxSeed) {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf(config.ErrSeedMaxFmt, config.MaxSeed))
		return
	}
	res, err := puzzle.Generate(req.Clues, req.Seed, solver.CountSolutions)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	grid81 := sudoku.Serialize(res.Grid)
	id, err := s.store.InsertPuzzle(grid81, res.Clues, int64(req.Seed))
	if err != nil {
		s.storeFailJSON(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, puzzleOut{
		ID:     id,
		Grid81: grid81,
		Clues:  res.Clues,
		Seed:   int64(req.Seed),
	})
}

func (s *Server) handleAPIPuzzles(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := s.parsePageParams(w, r)
	if !ok {
		return
	}
	puzzles, err := s.store.ListPuzzles(limit, offset)
	if err != nil {
		s.storeFailJSON(w, err)
		return
	}
	out := puzzlesOut{Puzzles: make([]puzzleOut, 0, len(puzzles))}
	for _, p := range puzzles {
		out.Puzzles = append(out.Puzzles, puzzleOut{
			ID:        p.ID,
			Grid81:    p.Grid81,
			Clues:     p.Clues,
			Seed:      p.Seed,
			CreatedAt: p.CreatedAt,
		})
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, healthOut{Status: config.HealthStatus})
}

func (s *Server) handleAPIVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, versionOut{Name: config.BinName, Version: config.Version})
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) parsePageParams(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit := config.APIPageLimit
	if v := r.URL.Query().Get(config.QueryLimit); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			s.writeError(w, http.StatusBadRequest, "invalid limit")
			return 0, 0, false
		}
		if n > config.APIPageLimitMax {
			n = config.APIPageLimitMax
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get(config.QueryOffset); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			s.writeError(w, http.StatusBadRequest, "invalid offset")
			return 0, 0, false
		}
		offset = n
	}
	return limit, offset, true
}
