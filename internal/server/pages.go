package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	view := homeView{
		baseView: baseView{Routes: newRoutesView()},
		Field:    config.FormGrid81,
		Grid:     gridLines(puzzle.Sample81),
		Samples:  puzzle.Samples(),
	}
	s.renderPage(w, "home", view)
}

func (s *Server) handleGeneratePage(w http.ResponseWriter, r *http.Request) {
	view := generateView{
		baseView:   baseView{Routes: newRoutesView()},
		FieldClues: config.FlagClues,
		FieldSeed:  config.FlagSeed,
		Clues:      s.cfg.Gen.Clues,
		Seed:       s.cfg.Gen.Seed,
	}
	s.renderPage(w, "generate", view)
}

func (s *Server) handleHistoryPage(w http.ResponseWriter, r *http.Request) {
	page := 1
	if v := r.URL.Query().Get(config.QueryPage); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			http.Error(w, "invalid page", http.StatusBadRequest)
			return
		}
		page = n
	}
	runs, err := s.store.ListRuns(config.HistoryPageSize, (page-1)*config.HistoryPageSize)
	if err != nil {
		s.storeFailPage(w, err)
		return
	}
	view := historyView{
		baseView: baseView{Routes: newRoutesView()},
		Rows:     runRows(runs),
	}
	if page > 1 {
		view.PrevURL = config.HistoryPagePath + "?" + config.QueryPage + "=" + strconv.Itoa(page-1)
	}
	if len(runs) == config.HistoryPageSize {
		view.NextURL = config.HistoryPagePath + "?" + config.QueryPage + "=" + strconv.Itoa(page+1)
	}
	s.renderPage(w, "history", view)
}

func (s *Server) handleRunPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	detail, err := s.store.GetRun(id)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.storeFailPage(w, err)
		return
	}
	view := runView{
		baseView:  baseView{Routes: newRoutesView()},
		ID:        detail.ID,
		CreatedAt: detail.CreatedAt,
		Input81:   detail.Input81,
		Source:    detail.Source,
		InputRows: gridRows(detail.Input81),
		Cards:     methodCardsFromRows(detail.Methods),
	}
	s.renderPage(w, "run", view)
}

func (s *Server) storeFailPage(w http.ResponseWriter, err error) {
	logging.Fail(s.log, "server.store", err)
	http.Error(w, "storage error", http.StatusInternalServerError)
}
