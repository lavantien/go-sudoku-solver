package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type ACOResult struct {
	Grid      SudokuGrid
	Steps     int
	TimeTaken time.Duration
	Solved    bool
}

type Ant struct {
	path []struct{ row, col, num int }
}

func solveACO(grid SudokuGrid) ACOResult {
	start := time.Now()
	var steps int
	solvedGrid, solved := solveACOSearch(grid.Copy(), &steps)
	return ACOResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}

func solveACOSearch(initialGrid SudokuGrid, steps *int) (SudokuGrid, bool) {
	const (
		numAnts       = 20   // Increased number of ants
		iterations    = 1500 // Increased iterations
		alpha         = 1.0
		beta          = 3.0 // Increased beta
		evaporation   = 0.3 // Reduced evaporation
		pheromoneInit = 0.1
	)

	pheromone := make([][][]float64, 9)
	for i := 0; i < 9; i++ {
		pheromone[i] = make([][]float64, 9)
		for j := 0; j < 9; j++ {
			pheromone[i][j] = make([]float64, 10)
			for k := 1; k <= 9; k++ {
				pheromone[i][j][k] = pheromoneInit
			}
		}
	}

	bestGrid := initialGrid.Copy()
	bestHeuristic := calculateHeuristic(initialGrid)

	for iter := 0; iter < iterations; iter++ {
		var wg sync.WaitGroup
		antPaths := make(chan Ant, numAnts)

		for ant := 0; ant < numAnts; ant++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				antPath := constructSolution(initialGrid.Copy(), pheromone, alpha, beta, steps)
				antPaths <- antPath
			}()
		}
		wg.Wait()
		close(antPaths)

		for antPath := range antPaths {
			antGrid := initialGrid.Copy()
			for _, move := range antPath.path {
				antGrid[move.row][move.col] = move.num
			}
			antHeuristic := calculateHeuristic(antGrid)
			if antHeuristic < bestHeuristic {
				bestHeuristic = antHeuristic
				bestGrid = antGrid.Copy()
			}
			updatePheromones(pheromone, antPath, evaporation)
		}
		if bestHeuristic == 0 {
			return bestGrid, true
		}
	}
	return bestGrid, false
}

func constructSolution(grid SudokuGrid, pheromone [][][]float64, alpha, beta float64, steps *int) Ant {
	ant := Ant{}
	for {
		*steps++
		row, col, found := findEmptyCell(grid)
		if !found {
			break
		}

		probabilities := make([]float64, 10)
		totalProbability := 0.0
		for num := 1; num <= 9; num++ {
			if isValidMove(grid, row, col, num) {
				probabilities[num] = pow(pheromone[row][col][num], alpha) * pow(1.0/float64(calculateConflicts(grid, row, col, num)), beta)
				totalProbability += probabilities[num]
			}
		}

		if totalProbability == 0 {
			break // No valid moves, ant failed
		}

		randNum := rand.Float64() * totalProbability
		var chosenNum int
		cumulativeProbability := 0.0
		for num := 1; num <= 9; num++ {
			if probabilities[num] > 0 {
				cumulativeProbability += probabilities[num]
				if randNum <= cumulativeProbability {
					chosenNum = num
					break
				}
			}
		}

		grid[row][col] = chosenNum
		ant.path = append(ant.path, struct{ row, col, num int }{row, col, chosenNum})
	}
	return ant
}

func calculateConflicts(grid SudokuGrid, row, col, num int) int {
	conflicts := 0
	for i := 0; i < 9; i++ {
		if grid[row][i] == num {
			conflicts++
		}
		if grid[i][col] == num {
			conflicts++
		}
	}
	startRow := row - row%3
	startCol := col - col%3
	for i := startRow; i < startRow+3; i++ {
		for j := startCol; j < startCol+3; j++ {
			if grid[i][j] == num {
				conflicts++
			}
		}
	}
	return conflicts - 1 // Subtract the current cell
}

func updatePheromones(pheromone [][][]float64, ant Ant, evaporation float64) {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			for k := 1; k <= 9; k++ {
				pheromone[i][j][k] *= (1 - evaporation)
			}
		}
	}
	for _, move := range ant.path {
		pheromone[move.row][move.col][move.num] += 1.0
	}
}

func pow(x, y float64) float64 {
	res := 1.0
	for i := 0; i < int(y); i++ {
		res *= x
	}
	return res
}

// Parallelized ACO (using a simple approach)
func solveACOParallel(grid SudokuGrid) ACOResult {
	start := time.Now()
	var steps int
	var solvedGrid SudokuGrid
	var solved bool
	const (
		numAnts       = 20   // Increased number of ants
		iterations    = 1500 // Increased iterations
		alpha         = 1.0
		beta          = 3.0 // Increased beta
		evaporation   = 0.3 // Reduced evaporation
		pheromoneInit = 0.1
	)

	pheromone := make([][][]float64, 9)
	for i := 0; i < 9; i++ {
		pheromone[i] = make([][]float64, 9)
		for j := 0; j < 9; j++ {
			pheromone[i][j] = make([]float64, 10)
			for k := 1; k <= 9; k++ {
				pheromone[i][j][k] = pheromoneInit
			}
		}
	}

	bestGrid := grid.Copy()
	bestHeuristic := calculateHeuristic(grid)

	for iter := 0; iter < iterations; iter++ {
		var wg sync.WaitGroup
		antPaths := make(chan Ant, numAnts)

		for ant := 0; ant < numAnts; ant++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				antPath := constructSolution(grid.Copy(), pheromone, alpha, beta, &steps)
				antPaths <- antPath
			}()
		}
		wg.Wait()
		close(antPaths)

		for antPath := range antPaths {
			antGrid := grid.Copy()
			for _, move := range antPath.path {
				antGrid[move.row][move.col] = move.num
			}
			antHeuristic := calculateHeuristic(antGrid)
			if antHeuristic < bestHeuristic {
				bestHeuristic = antHeuristic
				bestGrid = antGrid.Copy()
			}
			updatePheromones(pheromone, antPath, evaporation)
		}
		if bestHeuristic == 0 {
			solvedGrid = bestGrid
			solved = true
			break
		}
	}

	return ACOResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}
