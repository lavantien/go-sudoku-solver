package puzzle

import "github.com/lavantien/go-sudoku-solver/internal/sudoku"

type GenResult struct {
	Grid  sudoku.Grid
	Clues int
}

func Generate(clues int, seed uint64, unique func(sudoku.Grid, int) int) (GenResult, error) {
	return GenResult{}, nil
}
