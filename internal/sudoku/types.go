package sudoku

import "github.com/lavantien/go-sudoku-solver/internal/config"

type Grid [config.Cells]uint8

func CellRow(i int) int {
	return i / config.Size
}

func CellCol(i int) int {
	return i % config.Size
}

func CellBox(i int) int {
	return CellRow(i)/config.Box*config.Box + CellCol(i)/config.Box
}
