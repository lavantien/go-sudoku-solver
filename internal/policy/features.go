package policy

import (
	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func Features(g sudoku.Grid) []float64 {
	clues := 0
	for i := range config.Cells {
		if g[i] != config.EmptyDigit {
			clues++
		}
	}
	fill := solver.PropagationFill(g)
	unitMin := config.Size
	unitMax := 0
	unitSum := 0
	for u := range 3 * config.Size {
		n := 0
		for _, i := range sudoku.UnitCells(u) {
			if g[i] != config.EmptyDigit {
				n++
			}
		}
		unitMin = min(unitMin, n)
		unitMax = max(unitMax, n)
		unitSum += n
	}
	return []float64{
		float64(clues),
		float64(fill),
		float64(unitMin),
		float64(unitMax),
		float64(unitSum) / float64(3*config.Size),
	}
}
