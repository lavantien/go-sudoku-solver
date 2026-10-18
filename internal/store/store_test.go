package store

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func grid81(v byte) string {
	b := make([]byte, config.Cells)
	for i := range b {
		b[i] = byte('1') + (v+byte(i))%9
	}
	return string(b)
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "sudoku.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func tableNames(t *testing.T, s *Store) []string {
	t.Helper()
	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func columnNames(t *testing.T, s *Store, table string) []string {
	t.Helper()
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return cols
}

func requireUTCStamp(t *testing.T, stamp string) {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		t.Fatalf("stamp %q is not RFC3339: %v", stamp, err)
	}
	if _, offset := tm.Zone(); offset != 0 {
		t.Fatalf("stamp %q is not UTC: offset %d", stamp, offset)
	}
}

func TestOpenCreatesSchema(t *testing.T) {
	s := openTemp(t)
	names := tableNames(t, s)
	if want := []string{"puzzles", "run_methods", "runs"}; !slices.Equal(names, want) {
		t.Errorf("tables = %v, want %v", names, want)
	}
	for table, want := range map[string][]string{
		"runs":        {"id", "created_at", "input81", "source"},
		"run_methods": {"run_id", "method", "grid81", "steps", "time_ns", "solved"},
		"puzzles":     {"id", "grid81", "clues", "seed", "created_at"},
	} {
		if got := columnNames(t, s, table); !slices.Equal(got, want) {
			t.Errorf("%s columns = %v, want %v", table, got, want)
		}
	}
	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
	var timeout int
	if err := s.db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if timeout != busyTimeoutMS {
		t.Errorf("busy_timeout = %d, want %d", timeout, busyTimeoutMS)
	}
}

func TestInsertGetRunRoundTrip(t *testing.T) {
	s := openTemp(t)
	in := grid81(0)
	methods := []MethodRow{
		{Method: "propagate", Grid81: grid81(1), Steps: 64, TimeNS: 1234567, Solved: true},
		{Method: "cover", Grid81: grid81(2), Steps: 70, TimeNS: 2345678, Solved: true},
		{Method: "anneal", Grid81: grid81(3), Steps: 150000, TimeNS: 3456789, Solved: false},
	}
	id, err := s.InsertRun(in, "web", methods)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Errorf("first run id = %d, want 1", id)
	}
	d, err := s.GetRun(id)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != 1 || d.Input81 != in || d.Source != "web" || d.SolvedAny != true {
		t.Errorf("detail header = %+v", d.RunSummary)
	}
	requireUTCStamp(t, d.CreatedAt)
	if !slices.Equal(d.Methods, methods) {
		t.Errorf("methods = %+v, want %+v in insertion order", d.Methods, methods)
	}
}

func TestListRunsPaginationAndSolvedAny(t *testing.T) {
	s := openTemp(t)
	unsolved := []MethodRow{{Method: "propagate", Grid81: grid81(1), Steps: 10, TimeNS: 100, Solved: false}}
	mixed := []MethodRow{
		{Method: "propagate", Grid81: grid81(2), Steps: 20, TimeNS: 200, Solved: false},
		{Method: "cover", Grid81: grid81(3), Steps: 30, TimeNS: 300, Solved: true},
	}
	for i, tc := range []struct {
		methods []MethodRow
	}{
		{methods: unsolved},
		{methods: mixed},
		{methods: nil},
	} {
		if _, err := s.InsertRun(grid81(byte(i+4)), "api", tc.methods); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListRuns(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Fatalf("page len = %d, want 2", len(page))
	}
	if page[0].ID != 3 || page[0].SolvedAny != false {
		t.Errorf("page[0] = %+v, want id 3 solved_any false", page[0])
	}
	if page[1].ID != 2 || page[1].SolvedAny != true {
		t.Errorf("page[1] = %+v, want id 2 solved_any true", page[1])
	}
	requireUTCStamp(t, page[0].CreatedAt)
	page, err = s.ListRuns(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != 2 || page[1].ID != 1 {
		t.Errorf("offset page ids = %+v, want 2 then 1", page)
	}
	page, err = s.ListRuns(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Errorf("page past end len = %d, want 0", len(page))
	}
	n, err := s.CountRuns()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
}

func TestGetRunNotFound(t *testing.T) {
	s := openTemp(t)
	if _, err := s.GetRun(42); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestPuzzlesRoundTripAndPagination(t *testing.T) {
	s := openTemp(t)
	inserted := []Puzzle{
		{Grid81: grid81(1), Clues: 30, Seed: 101},
		{Grid81: grid81(2), Clues: 28, Seed: 202},
		{Grid81: grid81(3), Clues: 26, Seed: 303},
	}
	for i, p := range inserted {
		id, err := s.InsertPuzzle(p.Grid81, p.Clues, p.Seed)
		if err != nil {
			t.Fatal(err)
		}
		if id != int64(i+1) {
			t.Errorf("puzzle id = %d, want %d", id, i+1)
		}
	}
	page, err := s.ListPuzzles(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Fatalf("page len = %d, want 2", len(page))
	}
	if page[0].ID != 3 || page[0].Grid81 != inserted[2].Grid81 || page[0].Clues != 26 || page[0].Seed != 303 {
		t.Errorf("page[0] = %+v, want newest puzzle fields", page[0])
	}
	if page[1].ID != 2 || page[1].Clues != 28 || page[1].Seed != 202 {
		t.Errorf("page[1] = %+v, want middle puzzle fields", page[1])
	}
	requireUTCStamp(t, page[0].CreatedAt)
	page, err = s.ListPuzzles(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != 2 || page[1].ID != 1 {
		t.Errorf("offset page ids = %+v, want 2 then 1", page)
	}
	page, err = s.ListPuzzles(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Errorf("page past end len = %d, want 0", len(page))
	}
}

func TestCloseIdempotent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "sudoku.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	s := openTemp(t)
	const writes = 25
	finished := make(chan struct{})
	writerErr := make(chan error, 1)
	go func() {
		defer close(finished)
		for i := 0; i < writes; i++ {
			m := MethodRow{Method: "propagate", Grid81: grid81(byte(i)), Steps: i, TimeNS: int64(i), Solved: true}
			if _, err := s.InsertRun(grid81(byte(i)), "api", []MethodRow{m}); err != nil {
				writerErr <- err
				return
			}
		}
		writerErr <- nil
	}()
	readErr := make(chan error, 1)
	go func() {
		for {
			select {
			case <-finished:
				readErr <- nil
				return
			default:
			}
			if _, err := s.ListRuns(5, 0); err != nil {
				readErr <- err
				return
			}
			if _, err := s.CountRuns(); err != nil {
				readErr <- err
				return
			}
		}
	}()
	<-finished
	if err := <-writerErr; err != nil {
		t.Fatalf("concurrent write: %v", err)
	}
	if err := <-readErr; err != nil {
		t.Fatalf("concurrent read: %v", err)
	}
	n, err := s.CountRuns()
	if err != nil {
		t.Fatal(err)
	}
	if n != writes {
		t.Errorf("count after race = %d, want %d", n, writes)
	}
}
