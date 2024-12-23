package main

type SudokuGrid [][]int

func (grid SudokuGrid) Copy() SudokuGrid {
	newGrid := make(SudokuGrid, len(grid))
	for i := range grid {
		newGrid[i] = make([]int, len(grid[i]))
		copy(newGrid[i], grid[i])
	}
	return newGrid
}
