package main

import (
	"math"
	"sync"
	"time"
)

type MinimaxResult struct {
	Grid      SudokuGrid
	Steps     int
	TimeTaken time.Duration
	Solved    bool
}

func solveMinimax(grid SudokuGrid) MinimaxResult {
	start := time.Now()
	var steps int
	solvedGrid, solved := solveMinimaxRecursive(grid.Copy(), &steps, true, math.MinInt, math.MaxInt)
	return MinimaxResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}

func solveMinimaxRecursive(grid SudokuGrid, steps *int, maximizingPlayer bool, alpha, beta int) (SudokuGrid, bool) {
	*steps++
	if isSolved(grid) {
		return grid, true
	}

	row, col, found := findEmptyCell(grid)
	if !found {
		return grid, false // No empty cells, but not solved
	}

	if maximizingPlayer {
		bestValue := math.MinInt
		var bestGrid SudokuGrid
		for num := 1; num <= 9; num++ {
			if isValidMove(grid, row, col, num) {
				newGrid := grid.Copy()
				newGrid[row][col] = num
				solvedGrid, solved := solveMinimaxRecursive(newGrid, steps, false, alpha, beta)
				if solved {
					return solvedGrid, true
				}
				value := calculateMinimaxScore(solvedGrid)
				if value > bestValue {
					bestValue = value
					bestGrid = newGrid.Copy()
				}
				alpha = max(alpha, bestValue)
				if beta <= alpha {
					break // Beta cutoff
				}
			}
		}
		if bestValue == math.MinInt {
			return grid, false
		}
		return bestGrid, false
	} else {
		bestValue := math.MaxInt
		var bestGrid SudokuGrid
		for num := 1; num <= 9; num++ {
			if isValidMove(grid, row, col, num) {
				newGrid := grid.Copy()
				newGrid[row][col] = num
				solvedGrid, solved := solveMinimaxRecursive(newGrid, steps, true, alpha, beta)
				if solved {
					return solvedGrid, true
				}
				value := calculateMinimaxScore(solvedGrid)
				if value < bestValue {
					bestValue = value
					bestGrid = newGrid.Copy()
				}
				beta = min(beta, bestValue)
				if beta <= alpha {
					break // Alpha cutoff
				}
			}
		}
		if bestValue == math.MaxInt {
			return grid, false
		}
		return bestGrid, false
	}
}

func calculateMinimaxScore(grid SudokuGrid) int {
	return -calculateHeuristic(grid)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Parallelized Minimax with Alpha-Beta Pruning (using a simple approach)
func solveMinimaxParallel(grid SudokuGrid) MinimaxResult {
	start := time.Now()
	var steps int
	var solvedGrid SudokuGrid
	var solved bool
	var wg sync.WaitGroup

	solvedGrid, solved = solveMinimaxRecursiveParallel(grid.Copy(), &steps, true, math.MinInt, math.MaxInt, &wg)
	wg.Wait()

	return MinimaxResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}

func solveMinimaxRecursiveParallel(grid SudokuGrid, steps *int, maximizingPlayer bool, alpha, beta int, wg *sync.WaitGroup) (SudokuGrid, bool) {
	*steps++
	if isSolved(grid) {
		return grid, true
	}

	row, col, found := findEmptyCell(grid)
	if !found {
		return grid, false
	}

	if maximizingPlayer {
		bestValue := math.MinInt
		var bestGrid SudokuGrid
		for num := 1; num <= 9; num++ {
			if isValidMove(grid, row, col, num) {
				wg.Add(1)
				go func(num int, grid SudokuGrid, steps *int, alpha int, beta int, wg *sync.WaitGroup) {
					defer wg.Done()
					newGrid := grid.Copy()
					newGrid[row][col] = num
					solvedGridLocal, solvedLocal := solveMinimaxRecursiveParallel(newGrid, steps, false, alpha, beta, wg)
					if solvedLocal {
						bestGrid = solvedGridLocal
					}
					value := calculateMinimaxScore(solvedGridLocal)
					if value > bestValue {
						bestValue = value
						bestGrid = newGrid.Copy()
					}
					alpha = max(alpha, bestValue)
				}(num, grid, steps, alpha, beta, wg)
				if beta <= alpha {
					break
				}
			}
		}
		wg.Wait()
		if bestValue == math.MinInt {
			return grid, false
		}
		return bestGrid, false
	} else {
		bestValue := math.MaxInt
		var bestGrid SudokuGrid
		for num := 1; num <= 9; num++ {
			if isValidMove(grid, row, col, num) {
				wg.Add(1)
				go func(num int, grid SudokuGrid, steps *int, alpha int, beta int, wg *sync.WaitGroup) {
					defer wg.Done()
					newGrid := grid.Copy()
					newGrid[row][col] = num
					solvedGridLocal, solvedLocal := solveMinimaxRecursiveParallel(newGrid, steps, true, alpha, beta, wg)
					if solvedLocal {
						bestGrid = solvedGridLocal
					}
					value := calculateMinimaxScore(solvedGridLocal)
					if value < bestValue {
						bestValue = value
						bestGrid = newGrid.Copy()
					}
					beta = min(beta, bestValue)
				}(num, grid, steps, alpha, beta, wg)
				if beta <= alpha {
					break
				}
			}
		}
		wg.Wait()
		if bestValue == math.MaxInt {
			return grid, false
		}
		return bestGrid, false
	}
}
