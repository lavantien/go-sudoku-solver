package solver

import (
	"context"
	"sync"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/matmul"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const coverRowcount = config.Cells * config.Size

const coverColcount = 4 * config.Cells

const coverLiveWords = (coverRowcount + 63) / 64

const coverCoveredWords = (coverColcount + 63) / 64

type coverStructure struct {
	conflict matmul.BitMatrix
	colRows  [coverColcount][]int32
	rowCols  [coverRowcount][4]int
}

var coverOnce = sync.OnceValue(buildCoverStructure)

func buildCoverStructure() *coverStructure {
	inc := matmul.NewBitMatrix(coverRowcount, coverColcount)
	incT := matmul.NewBitMatrix(coverColcount, coverRowcount)
	for cell := range config.Cells {
		r := sudoku.CellRow(cell)
		c := sudoku.CellCol(cell)
		b := sudoku.CellBox(cell)
		for d := 1; d <= config.Size; d++ {
			row := cell*config.Size + d - 1
			cols := [4]int{
				cell,
				config.Cells + r*config.Size + d - 1,
				2*config.Cells + c*config.Size + d - 1,
				3*config.Cells + b*config.Size + d - 1,
			}
			for _, col := range cols {
				inc.Set(row, col)
				incT.Set(col, row)
			}
		}
	}
	s := &coverStructure{conflict: matmul.BooleanMul(inc, incT)}
	for col := range coverColcount {
		for row := range coverRowcount {
			if incT.Get(col, row) {
				s.colRows[col] = append(s.colRows[col], int32(row))
			}
		}
	}
	for row := range coverRowcount {
		k := 0
		for col := range coverColcount {
			if inc.Get(row, col) {
				s.rowCols[row][k] = col
				k++
			}
		}
	}
	return s
}

type coverSnapshot struct {
	live     [coverLiveWords]uint64
	covered  [coverCoveredWords]uint64
	selected int
}

type coverSearch struct {
	st       *coverStructure
	live     [coverLiveWords]uint64
	covered  [coverCoveredWords]uint64
	selected []int32
	nodes    int
	cap      int
	steps    *int
}

func (s *coverSearch) liveRow(row int32) bool {
	return s.live[row/64]&(1<<(row%64)) != 0
}

func (s *coverSearch) coveredCol(col int) bool {
	return s.covered[col/64]&(1<<(col%64)) != 0
}

func (s *coverSearch) selectRow(row int32) {
	for _, col := range s.st.rowCols[row] {
		s.covered[col/64] |= 1 << (col % 64)
	}
	base := int(row) * coverLiveWords
	for w := range coverLiveWords {
		s.live[w] &^= s.st.conflict.Words[base+w]
	}
	s.selected = append(s.selected, row)
	*s.steps++
}

func (s *coverSearch) snap() coverSnapshot {
	snap := coverSnapshot{selected: len(s.selected)}
	copy(snap.live[:], s.live[:])
	copy(snap.covered[:], s.covered[:])
	return snap
}

func (s *coverSearch) restore(snap coverSnapshot) {
	copy(s.live[:], snap.live[:])
	copy(s.covered[:], snap.covered[:])
	s.selected = s.selected[:snap.selected]
}

func (s *coverSearch) chooseColumn() int {
	best := -1
	bestCount := config.Size + 1
	for col := range coverColcount {
		if s.coveredCol(col) {
			continue
		}
		count := 0
		for _, row := range s.st.colRows[col] {
			if s.liveRow(row) {
				count++
			}
		}
		if count < bestCount {
			bestCount = count
			best = col
			if bestCount <= 1 {
				return best
			}
		}
	}
	return best
}

func (s *coverSearch) search(ctx context.Context) bool {
	s.nodes++
	if s.nodes > s.cap || ctx.Err() != nil {
		return false
	}
	col := s.chooseColumn()
	if col < 0 {
		return true
	}
	for _, row := range s.st.colRows[col] {
		if !s.liveRow(row) {
			continue
		}
		snap := s.snap()
		s.selectRow(row)
		if s.search(ctx) {
			return true
		}
		s.restore(snap)
	}
	return false
}

func Cover(ctx context.Context, g sudoku.Grid, c config.Config) Result {
	logger := logging.New(c.LogLevel)
	logging.Enter(logger, "cover", "clues", clueCount(g))
	start := time.Now()
	st := &coverSearch{
		st:    coverOnce(),
		cap:   c.CoverNodeCap,
		steps: new(int),
	}
	for w := range coverLiveWords {
		st.live[w] = ^uint64(0)
	}
	for cell := range config.Cells {
		if g[cell] == config.EmptyDigit {
			continue
		}
		row := int32(cell*config.Size + int(g[cell]) - 1)
		if !st.liveRow(row) {
			logging.Fail(logger, "cover", errUnsolved, "at", "givens")
			return Result{Grid: g, Steps: *st.steps, Dur: time.Since(start), Solved: false}
		}
		st.selectRow(row)
	}
	if st.search(ctx) {
		var out sudoku.Grid
		for _, row := range st.selected {
			cell := int(row) / config.Size
			out[cell] = uint8(int(row)%config.Size + 1)
		}
		logging.Exit(logger, "cover", "steps", *st.steps)
		return Result{Grid: out, Steps: *st.steps, Dur: time.Since(start), Solved: true}
	}
	logging.Fail(logger, "cover", errUnsolved)
	return Result{Grid: g, Steps: *st.steps, Dur: time.Since(start), Solved: false}
}
