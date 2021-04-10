package main

import (
	"fmt"
	"time"
)

const BOARD_SIZE = 9
const SUBGRID_COUNT = BOARD_SIZE / 3

var grid [BOARD_SIZE][BOARD_SIZE]int
var isUnsolvedCell [BOARD_SIZE][BOARD_SIZE]bool
var possibility [BOARD_SIZE][BOARD_SIZE][BOARD_SIZE]int
var possibilitySize [BOARD_SIZE][BOARD_SIZE]int
var maxPossibilitySize int
var recursiveCount int
var presolveCount int

func main() {
	for i := 0; i < BOARD_SIZE; i++ {
		var tempString string
		fmt.Scanln(&tempString)
		for j, value := range tempString {
			if value != '.' {
				grid[i][j] = int(value - '0')
			} else {
				grid[i][j] = 0
			}
		}
	}
	tick := time.Now()
	basicSolver()
	fmt.Println()
	for i := 0; i < BOARD_SIZE; i++ {
		for j := 0; j < BOARD_SIZE; j++ {
			fmt.Print(grid[i][j])
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("Presolver step counter:", presolveCount)
	fmt.Println("Recursive step counter:", recursiveCount)
	fmt.Println("Execute time:", time.Since(tick))
}

func basicSolver() {
	if !firstLevelPresolver() {
		basicBacktracking(0, 0)
	}
}

func firstLevelPresolver() bool {
	for {
		presolveCount++
		countEmptyCell := 0
		countSolvedCell := 0
		for i := 0; i < BOARD_SIZE; i++ {
			for j := 0; j < BOARD_SIZE; j++ {
				if grid[i][j] == 0 {
					countEmptyCell++
					isUnsolvedCell[i][j] = true
					possibilitySize[i][j] = 0
					for k := 1; k <= BOARD_SIZE; k++ {
						if isFillable(i, j, k) {
							possibility[i][j][possibilitySize[i][j]] = k
							possibilitySize[i][j]++
						}
					}
					if possibilitySize[i][j] > maxPossibilitySize {
						maxPossibilitySize = possibilitySize[i][j]
					}
					if possibilitySize[i][j] == 1 {
						countSolvedCell++
						isUnsolvedCell[i][j] = false
						grid[i][j] = possibility[i][j][0]
					}
				}
			}
		}
		if countEmptyCell == 0 {
			return true
		} else if countEmptyCell > countSolvedCell && countSolvedCell == 0 {
			return false
		}
	}
}

func basicBacktracking(x int, y int) bool {
	recursiveCount++
	if y == BOARD_SIZE {
		x++
		y = 0
		if x == BOARD_SIZE {
			return true
		}
	}
	if !isUnsolvedCell[x][y] {
		return basicBacktracking(x, y+1)
	} else {
		for k := 0; k < possibilitySize[x][y]; k++ {
			if isFillable(x, y, possibility[x][y][k]) {
				grid[x][y] = possibility[x][y][k]
				if basicBacktracking(x, y+1) {
					return true
				}
			}
		}
		grid[x][y] = 0
		return false
	}
}

func isFillable(x int, y int, k int) bool {
	for i := 0; i < BOARD_SIZE; i++ {
		subgridX := x/SUBGRID_COUNT*SUBGRID_COUNT + i/SUBGRID_COUNT
		subgridY := y/SUBGRID_COUNT*SUBGRID_COUNT + i%SUBGRID_COUNT
		if (i != y && k == grid[x][i]) ||
			(i != x && k == grid[i][y]) ||
			(x != subgridX && y != subgridY && k == grid[subgridX][subgridY]) {
			return false
		}
	}
	return true
}
