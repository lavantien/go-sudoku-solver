package puzzle

import (
	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func ValidSolution(g sudoku.Grid) bool {
	for i := range config.Size {
		if sudoku.RowUsed(g, i) != sudoku.AllDigits {
			return false
		}
		if sudoku.ColUsed(g, i) != sudoku.AllDigits {
			return false
		}
		if sudoku.BoxUsed(g, i) != sudoku.AllDigits {
			return false
		}
	}
	return true
}

func GivensPreserved(g, orig sudoku.Grid) bool {
	for i := range orig {
		if orig[i] != config.EmptyDigit && g[i] != orig[i] {
			return false
		}
	}
	return true
}
