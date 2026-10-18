package solver

import (
	"context"
	"errors"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

var errUnsolved = errors.New("no solution found")

func clueCount(g sudoku.Grid) int {
	n := 0
	for i := range config.Cells {
		if g[i] != config.EmptyDigit {
			n++
		}
	}
	return n
}

func Propagate(ctx context.Context, g sudoku.Grid, c config.Config) Result {
	logger := logging.New(c.LogLevel)
	logging.Enter(logger, "propagate", "clues", clueCount(g))
	start := time.Now()
	steps := 0
	board := g
	if propagateSingles(&board, &steps) && search(ctx, &board, &steps) {
		logging.Exit(logger, "propagate", "steps", steps)
		return Result{Grid: board, Steps: steps, Dur: time.Since(start), Solved: true}
	}
	logging.Fail(logger, "propagate", errUnsolved)
	return Result{Grid: g, Steps: steps, Dur: time.Since(start), Solved: false}
}

func search(ctx context.Context, board *sudoku.Grid, steps *int) bool {
	if ctx.Err() != nil {
		return false
	}
	i := sudoku.MRVCell(*board)
	if i < 0 {
		return true
	}
	mask := sudoku.Candidates(*board, i)
	for mask != 0 {
		d := sudoku.FirstSetDigit(mask)
		mask &= mask - 1
		if ctx.Err() != nil {
			return false
		}
		snap := *board
		board[i] = uint8(d)
		*steps++
		if propagateSingles(board, steps) && search(ctx, board, steps) {
			return true
		}
		*board = snap
	}
	return false
}

func propagateSingles(board *sudoku.Grid, steps *int) bool {
	for {
		progress := false
		for i := range config.Cells {
			if board[i] != config.EmptyDigit {
				continue
			}
			m := sudoku.Candidates(*board, i)
			n := sudoku.Popcount(m)
			if n == 0 {
				return false
			}
			if n == 1 {
				board[i] = uint8(sudoku.FirstSetDigit(m))
				*steps++
				progress = true
			}
		}
		for u := range 3 * config.Size {
			unit := sudoku.UnitCells(u)
			for d := 1; d <= config.Size; d++ {
				bit := uint16(1) << (d - 1)
				placed := false
				count := 0
				at := -1
				for _, i := range unit {
					if board[i] != config.EmptyDigit {
						if board[i] == uint8(d) {
							placed = true
						}
						continue
					}
					if sudoku.Candidates(*board, i)&bit != 0 {
						count++
						at = i
					}
				}
				if placed {
					continue
				}
				if count == 0 {
					return false
				}
				if count == 1 {
					board[at] = uint8(d)
					*steps++
					progress = true
				}
			}
		}
		if !progress {
			return true
		}
	}
}

func CountSolutions(g sudoku.Grid, limit int) int {
	if limit < 1 {
		limit = 1
	}
	board := g
	steps := 0
	if !propagateSingles(&board, &steps) {
		return 0
	}
	return countSearch(&board, &steps, limit)
}

func countSearch(board *sudoku.Grid, steps *int, limit int) int {
	found := 0
	i := sudoku.MRVCell(*board)
	if i < 0 {
		return 1
	}
	mask := sudoku.Candidates(*board, i)
	for mask != 0 && found < limit {
		d := sudoku.FirstSetDigit(mask)
		mask &= mask - 1
		snap := *board
		board[i] = uint8(d)
		*steps++
		if propagateSingles(board, steps) {
			found += countSearch(board, steps, limit-found)
		}
		*board = snap
	}
	return found
}

func PropagationFill(g sudoku.Grid) int {
	board := g
	steps := 0
	propagateSingles(&board, &steps)
	return steps
}
