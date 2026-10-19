package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func (s *Server) handleSolveFragment(w http.ResponseWriter, r *http.Request) {
	g, err := parseFormGrid(r.PostFormValue(config.FormGrid81))
	if err != nil {
		s.renderSolveError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, rows, err := s.solveAndPersist(r.Context(), g, config.SourceWeb)
	if err != nil {
		s.storeFailPage(w, err)
		return
	}
	view := solveResultsView{
		RunID:  id,
		RunURL: fmt.Sprintf(config.RunPagePathFmt, id),
		Cards:  methodCardsFromRows(rows),
	}
	s.renderFragment(w, http.StatusOK, "solve", view)
}

func (s *Server) handleGenerateFragment(w http.ResponseWriter, r *http.Request) {
	clues := s.cfg.Gen.Clues
	if v := r.PostFormValue(config.FlagClues); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			s.renderSolveError(w, http.StatusBadRequest, err.Error())
			return
		}
		clues = n
	}
	seed := s.cfg.Gen.Seed
	if v := r.PostFormValue(config.FlagSeed); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			s.renderSolveError(w, http.StatusBadRequest, err.Error())
			return
		}
		seed = n
	}
	if seed > uint64(config.MaxSeed) {
		s.renderSolveError(w, http.StatusBadRequest, fmt.Sprintf(config.ErrSeedMaxFmt, config.MaxSeed))
		return
	}
	res, err := puzzle.Generate(clues, seed, solver.CountSolutions)
	if err != nil {
		s.renderSolveError(w, http.StatusBadRequest, err.Error())
		return
	}
	grid81 := sudoku.Serialize(res.Grid)
	id, err := s.store.InsertPuzzle(grid81, res.Clues, int64(seed))
	if err != nil {
		s.storeFailPage(w, err)
		return
	}
	view := puzzleView{ID: id, Rows: gridRows(grid81), Clues: res.Clues, Seed: int64(seed)}
	s.renderFragment(w, http.StatusOK, "puzzle", view)
}

func parseFormGrid(v string) (sudoku.Grid, error) {
	var b strings.Builder
	for _, ch := range v {
		switch {
		case ch >= '1' && ch <= '9', ch == '.':
			b.WriteRune(ch)
		case ch == '\r', ch == '\n', ch == ' ', ch == '\t':
		default:
			return sudoku.Grid{}, fmt.Errorf(config.ErrCharFmt, ch)
		}
	}
	cleaned := b.String()
	if len(cleaned) != config.Cells {
		return sudoku.Grid{}, fmt.Errorf(config.ErrGridLengthFmt, len(cleaned))
	}
	return sudoku.Parse81(cleaned)
}
