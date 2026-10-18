package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

func errTemplateNotFound(name string) error {
	return fmt.Errorf("template not found: %s", name)
}

type routesView struct {
	Home          string
	Generate      string
	History       string
	SolveFragment string
	GenFragment   string
	Static        string
}

func newRoutesView() routesView {
	return routesView{
		Home:          config.HomePath,
		Generate:      config.GeneratePagePath,
		History:       config.HistoryPagePath,
		SolveFragment: config.FragmentSolvePath,
		GenFragment:   config.FragmentGenPath,
		Static:        config.StaticMountPath,
	}
}

type baseView struct {
	Routes routesView
}

type homeView struct {
	baseView
	Field   string
	Grid    string
	Samples []puzzle.Sample
}

type generateView struct {
	baseView
	FieldClues string
	FieldSeed  string
	Clues      int
	Seed       uint64
}

type runRowView struct {
	ID        int64
	URL       string
	CreatedAt string
	Input81   string
	Source    string
	SolvedAny bool
}

type historyView struct {
	baseView
	Rows    []runRowView
	PrevURL string
	NextURL string
}

type runView struct {
	baseView
	ID        int64
	CreatedAt string
	Input81   string
	Source    string
	InputRows [][]string
	Cards     []methodCardView
}

type methodCardView struct {
	Label    string
	Method   string
	Rows     [][]string
	Steps    int
	Duration string
	Solved   bool
}

type solveResultsView struct {
	RunID  int64
	RunURL string
	Cards  []methodCardView
}

type puzzleView struct {
	ID    int64
	Rows  [][]string
	Clues int
	Seed  int64
}

type errorView struct {
	Message string
}

func gridRows(grid81 string) [][]string {
	rows := make([][]string, config.Size)
	for r := range config.Size {
		row := make([]string, config.Size)
		for c := range config.Size {
			row[c] = string(grid81[r*config.Size+c])
		}
		rows[r] = row
	}
	return rows
}

func gridLines(grid81 string) string {
	rows := make([]string, 0, config.Size)
	for r := range config.Size {
		rows = append(rows, grid81[r*config.Size:(r+1)*config.Size])
	}
	return strings.Join(rows, "\n")
}

func methodCardsFromRows(rows []store.MethodRow) []methodCardView {
	cards := make([]methodCardView, 0, len(rows))
	for _, m := range rows {
		cards = append(cards, methodCardView{
			Label:    config.MethodLabel(m.Method),
			Method:   m.Method,
			Rows:     gridRows(m.Grid81),
			Steps:    m.Steps,
			Duration: time.Duration(m.TimeNS).String(),
			Solved:   m.Solved,
		})
	}
	return cards
}

func runRows(summaries []store.RunSummary) []runRowView {
	rows := make([]runRowView, 0, len(summaries))
	for _, r := range summaries {
		rows = append(rows, runRowView{
			ID:        r.ID,
			URL:       fmt.Sprintf(config.RunPagePathFmt, r.ID),
			CreatedAt: r.CreatedAt,
			Input81:   r.Input81,
			Source:    r.Source,
			SolvedAny: r.SolvedAny,
		})
	}
	return rows
}
