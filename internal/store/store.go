package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/lavantien/go-sudoku-solver/internal/logging"
)

const (
	driverName = "sqlite3"

	createRuns       = `CREATE TABLE IF NOT EXISTS runs(id INTEGER PRIMARY KEY AUTOINCREMENT, created_at TEXT NOT NULL, input81 TEXT NOT NULL, source TEXT NOT NULL)`
	createRunMethods = `CREATE TABLE IF NOT EXISTS run_methods(run_id INTEGER NOT NULL, method TEXT NOT NULL, grid81 TEXT NOT NULL, steps INTEGER NOT NULL, time_ns INTEGER NOT NULL, solved INTEGER NOT NULL)`
	createPuzzles    = `CREATE TABLE IF NOT EXISTS puzzles(id INTEGER PRIMARY KEY AUTOINCREMENT, grid81 TEXT NOT NULL, clues INTEGER NOT NULL, seed INTEGER NOT NULL, created_at TEXT NOT NULL)`

	insertRunStmt          = `INSERT INTO runs (created_at, input81, source) VALUES (?, ?, ?)`
	insertRunMethodStmt    = `INSERT INTO run_methods (run_id, method, grid81, steps, time_ns, solved) VALUES (?, ?, ?, ?, ?, ?)`
	selectRunSummariesStmt = `SELECT id, created_at, input81, source, EXISTS (SELECT 1 FROM run_methods WHERE run_id = runs.id AND solved = 1) FROM runs ORDER BY id DESC LIMIT ? OFFSET ?`
	selectRunSummaryStmt   = `SELECT id, created_at, input81, source, EXISTS (SELECT 1 FROM run_methods WHERE run_id = runs.id AND solved = 1) FROM runs WHERE id = ?`
	selectRunMethodsStmt   = `SELECT method, grid81, steps, time_ns, solved FROM run_methods WHERE run_id = ? ORDER BY rowid`
	insertPuzzleStmt       = `INSERT INTO puzzles (grid81, clues, seed, created_at) VALUES (?, ?, ?, ?)`
	selectPuzzlesStmt      = `SELECT id, grid81, clues, seed, created_at FROM puzzles ORDER BY id DESC LIMIT ? OFFSET ?`
	countRunsStmt          = `SELECT COUNT(*) FROM runs`
)

var (
	ErrNotFound = errors.New("store: not found")
	ErrClosed   = errors.New("store: closed")
)

type MethodRow struct {
	Method string
	Grid81 string
	Steps  int
	TimeNS int64
	Solved bool
}

type RunSummary struct {
	ID        int64
	CreatedAt string
	Input81   string
	Source    string
	SolvedAny bool
}

type RunDetail struct {
	RunSummary
	Methods []MethodRow
}

type Puzzle struct {
	ID        int64
	Grid81    string
	Clues     int
	Seed      int64
	CreatedAt string
}

type Store struct {
	mu  sync.Mutex
	db  *sql.DB
	log *slog.Logger
}

func Open(path string, busyTimeout time.Duration) (*Store, error) {
	log := logging.New("")
	logging.Enter(log, "store.open", "path", path, "busy_timeout", busyTimeout)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logging.Fail(log, "store.open", err, "path", path)
		return nil, fmt.Errorf("store open: %w", err)
	}
	dsn := fmt.Sprintf("%s?_busy_timeout=%d", path, busyTimeout.Milliseconds())
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		logging.Fail(log, "store.open", err, "path", path)
		return nil, fmt.Errorf("store open: %w", err)
	}
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
		_ = db.Close()
		logging.Fail(log, "store.open", err, "path", path)
		return nil, fmt.Errorf("store open wal: %w", err)
	}
	for _, stmt := range []string{createRuns, createRunMethods, createPuzzles} {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			logging.Fail(log, "store.open", err, "path", path)
			return nil, fmt.Errorf("store open schema: %w", err)
		}
	}
	logging.Exit(log, "store.open", "journal_mode", mode)
	return &Store{db: db, log: log}, nil
}

func (s *Store) handle() (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, ErrClosed
	}
	return s.db, nil
}

func (s *Store) Close() error {
	logging.Enter(s.log, "store.close")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		logging.Exit(s.log, "store.close", "already_closed", true)
		return nil
	}
	db := s.db
	s.db = nil
	if err := db.Close(); err != nil {
		logging.Fail(s.log, "store.close", err)
		return fmt.Errorf("store close: %w", err)
	}
	logging.Exit(s.log, "store.close")
	return nil
}

func (s *Store) InsertRun(input81, source string, methods []MethodRow) (int64, error) {
	logging.Enter(s.log, "store.insert_run", "source", source, "methods", len(methods))
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.insert_run", err)
		return 0, fmt.Errorf("store insert run: %w", err)
	}
	tx, err := db.Begin()
	if err != nil {
		logging.Fail(s.log, "store.insert_run", err)
		return 0, fmt.Errorf("store insert run: %w", err)
	}
	created := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(insertRunStmt, created, input81, source)
	if err != nil {
		_ = tx.Rollback()
		logging.Fail(s.log, "store.insert_run", err)
		return 0, fmt.Errorf("store insert run: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		logging.Fail(s.log, "store.insert_run", err)
		return 0, fmt.Errorf("store insert run id: %w", err)
	}
	for _, m := range methods {
		if _, err := tx.Exec(insertRunMethodStmt, id, m.Method, m.Grid81, m.Steps, m.TimeNS, m.Solved); err != nil {
			_ = tx.Rollback()
			logging.Fail(s.log, "store.insert_run", err, "id", id)
			return 0, fmt.Errorf("store insert run method %s: %w", m.Method, err)
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Fail(s.log, "store.insert_run", err, "id", id)
		return 0, fmt.Errorf("store insert run commit: %w", err)
	}
	logging.Exit(s.log, "store.insert_run", "id", id)
	return id, nil
}

func (s *Store) ListRuns(limit, offset int) ([]RunSummary, error) {
	logging.Enter(s.log, "store.list_runs", "limit", limit, "offset", offset)
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.list_runs", err)
		return nil, fmt.Errorf("store list runs: %w", err)
	}
	rows, err := db.Query(selectRunSummariesStmt, limit, offset)
	if err != nil {
		logging.Fail(s.log, "store.list_runs", err)
		return nil, fmt.Errorf("store list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	summaries := []RunSummary{}
	for rows.Next() {
		var r RunSummary
		if err := rows.Scan(&r.ID, &r.CreatedAt, &r.Input81, &r.Source, &r.SolvedAny); err != nil {
			logging.Fail(s.log, "store.list_runs", err)
			return nil, fmt.Errorf("store list runs scan: %w", err)
		}
		summaries = append(summaries, r)
	}
	if err := rows.Err(); err != nil {
		logging.Fail(s.log, "store.list_runs", err)
		return nil, fmt.Errorf("store list runs rows: %w", err)
	}
	logging.Exit(s.log, "store.list_runs", "count", len(summaries))
	return summaries, nil
}

func (s *Store) GetRun(id int64) (*RunDetail, error) {
	logging.Enter(s.log, "store.get_run", "id", id)
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.get_run", err, "id", id)
		return nil, fmt.Errorf("store get run %d: %w", id, err)
	}
	d := &RunDetail{}
	err = db.QueryRow(selectRunSummaryStmt, id).Scan(&d.ID, &d.CreatedAt, &d.Input81, &d.Source, &d.SolvedAny)
	if errors.Is(err, sql.ErrNoRows) {
		logging.Fail(s.log, "store.get_run", ErrNotFound, "id", id)
		return nil, fmt.Errorf("store get run %d: %w", id, ErrNotFound)
	}
	if err != nil {
		logging.Fail(s.log, "store.get_run", err, "id", id)
		return nil, fmt.Errorf("store get run %d: %w", id, err)
	}
	rows, err := db.Query(selectRunMethodsStmt, id)
	if err != nil {
		logging.Fail(s.log, "store.get_run", err, "id", id)
		return nil, fmt.Errorf("store get run %d methods: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m MethodRow
		if err := rows.Scan(&m.Method, &m.Grid81, &m.Steps, &m.TimeNS, &m.Solved); err != nil {
			logging.Fail(s.log, "store.get_run", err, "id", id)
			return nil, fmt.Errorf("store get run %d method scan: %w", id, err)
		}
		d.Methods = append(d.Methods, m)
	}
	if err := rows.Err(); err != nil {
		logging.Fail(s.log, "store.get_run", err, "id", id)
		return nil, fmt.Errorf("store get run %d methods rows: %w", id, err)
	}
	logging.Exit(s.log, "store.get_run", "id", id, "methods", len(d.Methods))
	return d, nil
}

func (s *Store) InsertPuzzle(grid81 string, clues int, seed int64) (int64, error) {
	logging.Enter(s.log, "store.insert_puzzle", "clues", clues, "seed", seed)
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.insert_puzzle", err)
		return 0, fmt.Errorf("store insert puzzle: %w", err)
	}
	created := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(insertPuzzleStmt, grid81, clues, seed, created)
	if err != nil {
		logging.Fail(s.log, "store.insert_puzzle", err)
		return 0, fmt.Errorf("store insert puzzle: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		logging.Fail(s.log, "store.insert_puzzle", err)
		return 0, fmt.Errorf("store insert puzzle id: %w", err)
	}
	logging.Exit(s.log, "store.insert_puzzle", "id", id)
	return id, nil
}

func (s *Store) ListPuzzles(limit, offset int) ([]Puzzle, error) {
	logging.Enter(s.log, "store.list_puzzles", "limit", limit, "offset", offset)
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.list_puzzles", err)
		return nil, fmt.Errorf("store list puzzles: %w", err)
	}
	rows, err := db.Query(selectPuzzlesStmt, limit, offset)
	if err != nil {
		logging.Fail(s.log, "store.list_puzzles", err)
		return nil, fmt.Errorf("store list puzzles: %w", err)
	}
	defer func() { _ = rows.Close() }()
	puzzles := []Puzzle{}
	for rows.Next() {
		var p Puzzle
		if err := rows.Scan(&p.ID, &p.Grid81, &p.Clues, &p.Seed, &p.CreatedAt); err != nil {
			logging.Fail(s.log, "store.list_puzzles", err)
			return nil, fmt.Errorf("store list puzzles scan: %w", err)
		}
		puzzles = append(puzzles, p)
	}
	if err := rows.Err(); err != nil {
		logging.Fail(s.log, "store.list_puzzles", err)
		return nil, fmt.Errorf("store list puzzles rows: %w", err)
	}
	logging.Exit(s.log, "store.list_puzzles", "count", len(puzzles))
	return puzzles, nil
}

func (s *Store) CountRuns() (int64, error) {
	logging.Enter(s.log, "store.count_runs")
	db, err := s.handle()
	if err != nil {
		logging.Fail(s.log, "store.count_runs", err)
		return 0, fmt.Errorf("store count runs: %w", err)
	}
	var n int64
	if err := db.QueryRow(countRunsStmt).Scan(&n); err != nil {
		logging.Fail(s.log, "store.count_runs", err)
		return 0, fmt.Errorf("store count runs: %w", err)
	}
	logging.Exit(s.log, "store.count_runs", "count", n)
	return n, nil
}
