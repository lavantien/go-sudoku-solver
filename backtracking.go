package main

import (
	"runtime"
	"sync"
	"time"
)

type BacktrackingResult struct {
	Grid      SudokuGrid
	Steps     int
	TimeTaken time.Duration
	Solved    bool
}

func solveBacktracking(grid SudokuGrid) BacktrackingResult {
	start := time.Now()
	var steps int
	solvedGrid, solved := solveBacktrackingRecursive(grid.Copy(), &steps)
	return BacktrackingResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}

func solveBacktrackingRecursive(grid SudokuGrid, steps *int) (SudokuGrid, bool) {
	*steps++
	row, col, found := findEmptyCell(grid)
	if !found {
		return grid, true // Puzzle is solved
	}

	for num := 1; num <= 9; num++ {
		if isValidMove(grid, row, col, num) {
			grid[row][col] = num
			solvedGrid, solved := solveBacktrackingRecursive(grid, steps)
			if solved {
				return solvedGrid, true
			}
			grid[row][col] = 0 // Backtrack
		}
	}
	return grid, false // No solution found in this branch
}

func findEmptyCell(grid SudokuGrid) (int, int, bool) {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if grid[i][j] == 0 {
				return i, j, true
			}
		}
	}
	return -1, -1, false
}

func isValidMove(grid SudokuGrid, row, col, num int) bool {
	// Check row
	for i := 0; i < 9; i++ {
		if grid[row][i] == num {
			return false
		}
	}

	// Check column
	for i := 0; i < 9; i++ {
		if grid[i][col] == num {
			return false
		}
	}

	// Check 3x3 subgrid
	startRow := row - row%3
	startCol := col - col%3
	for i := startRow; i < startRow+3; i++ {
		for j := startCol; j < startCol+3; j++ {
			if grid[i][j] == num {
				return false
			}
		}
	}

	return true
}

// Parallelized Backtracking with Work Stealing
func solveBacktrackingParallel(grid SudokuGrid) BacktrackingResult {
	start := time.Now()
	var steps int
	var solvedGrid SudokuGrid
	var solved bool

	numWorkers := runtime.NumCPU()
	workChan := make(chan SudokuGrid, numWorkers)
	resultChan := make(chan struct {
		grid   SudokuGrid
		solved bool
	}, numWorkers)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				solvedGridLocal, solvedLocal := solveBacktrackingRecursive(work, &steps)
				if solvedLocal {
					resultChan <- struct {
						grid   SudokuGrid
						solved bool
					}{grid: solvedGridLocal, solved: true}
					return
				}
			}
		}()
	}

	// Initial work
	row, col, found := findEmptyCell(grid)
	if !found {
		return BacktrackingResult{Grid: grid, Steps: 0, TimeTaken: time.Since(start), Solved: true}
	}
	for num := 1; num <= 9; num++ {
		if isValidMove(grid, row, col, num) {
			newGrid := grid.Copy()
			newGrid[row][col] = num
			workChan <- newGrid
		}
	}
	close(workChan)

	// Collect results
	for result := range resultChan {
		solvedGrid = result.grid
		solved = result.solved
		if solved {
			break
		}
	}
	close(resultChan)
	wg.Wait()

	return BacktrackingResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}
