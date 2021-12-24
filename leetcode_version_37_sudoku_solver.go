// @algorithm @lc id=37 lang=golang
// @title sudoku-solver

package main

const (
	BOARD_SIZE    = 9
	SUBGRID_COUNT = 3
)

// @test([["5","3",".",".","7",".",".",".","."],["6",".",".","1","9","5",".",".","."],[".","9","8",".",".",".",".","6","."],["8",".",".",".","6",".",".",".","3"],["4",".",".","8",".","3",".",".","1"],["7",".",".",".","2",".",".",".","6"],[".","6",".",".",".",".","2","8","."],[".",".",".","4","1","9",".",".","5"],[".",".",".",".","8",".",".","7","9"]])=[["5","3","4","6","7","8","9","1","2"],["6","7","2","1","9","5","3","4","8"],["1","9","8","3","4","2","5","6","7"],["8","5","9","7","6","1","4","2","3"],["4","2","6","8","5","3","7","9","1"],["7","1","3","9","2","4","8","5","6"],["9","6","1","5","3","7","2","8","4"],["2","8","7","4","1","9","6","3","5"],["3","4","5","2","8","6","1","7","9"]]
// @test([[".",".",".","2",".",".",".","6","3"],["3",".",".",".",".","5","4",".","1"],[".",".","1",".",".","3","9","8","."],[".",".",".",".",".",".",".","9","."],[".",".",".","5","3","8",".",".","."],[".","3",".",".",".",".",".",".","."],[".","2","6","3",".",".","5",".","."],["5",".","3","7",".",".",".",".","8"],["4","7",".",".",".","1",".",".","."]])=[["8","5","4","2","1","9","7","6","3"],["3","9","7","8","6","5","4","2","1"],["2","6","1","4","7","3","9","8","5"],["7","8","5","1","2","6","3","9","4"],["6","4","9","5","3","8","1","7","2"],["1","3","2","9","4","7","8","5","6"],["9","2","6","3","8","4","5","1","7"],["5","1","3","7","9","2","6","4","8"],["4","7","8","6","5","1","2","3","9"]]
func solveSudoku(board [][]byte) {
	var (
		grid               [BOARD_SIZE][BOARD_SIZE]int
		isUnsolvedCell     [BOARD_SIZE][BOARD_SIZE]bool
		possibility        [BOARD_SIZE][BOARD_SIZE][BOARD_SIZE]int
		possibilitySize    [BOARD_SIZE][BOARD_SIZE]int
		maxPossibilitySize int
	)

	for i := 0; i < BOARD_SIZE; i++ {
		for j := 0; j < BOARD_SIZE; j++ {
			if board[i][j] == '.' {
				grid[i][j] = 0
			} else {
				grid[i][j] = int(board[i][j] - '0')
			}
		}
	}

	if !firstLevelPresolver(&grid, &isUnsolvedCell, &possibility, &possibilitySize, maxPossibilitySize) {
		basicBacktracking(&grid, &isUnsolvedCell, &possibility, &possibilitySize, 0, 0)
	}

	for i := 0; i < BOARD_SIZE; i++ {
		for j := 0; j < BOARD_SIZE; j++ {
			board[i][j] = byte(grid[i][j] + '0')
		}
	}
}

func firstLevelPresolver(grid *[BOARD_SIZE][BOARD_SIZE]int, isUnsolvedCell *[BOARD_SIZE][BOARD_SIZE]bool, possibility *[BOARD_SIZE][BOARD_SIZE][BOARD_SIZE]int, possibilitySize *[BOARD_SIZE][BOARD_SIZE]int, maxPossibilitySize int) bool {
	for {
		countEmptyCell := 0
		countSolvedCell := 0
		for i := 0; i < BOARD_SIZE; i++ {
			for j := 0; j < BOARD_SIZE; j++ {
				if grid[i][j] == 0 {
					countEmptyCell++
					isUnsolvedCell[i][j] = true
					possibilitySize[i][j] = 0
					for k := 1; k <= BOARD_SIZE; k++ {
						if isFillable(grid, i, j, k) {
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

func basicBacktracking(grid *[BOARD_SIZE][BOARD_SIZE]int, isUnsolvedCell *[BOARD_SIZE][BOARD_SIZE]bool, possibility *[BOARD_SIZE][BOARD_SIZE][BOARD_SIZE]int, possibilitySize *[BOARD_SIZE][BOARD_SIZE]int, x int, y int) bool {
	if y == BOARD_SIZE {
		x++
		y = 0
		if x == BOARD_SIZE {
			return true
		}
	}
	if !isUnsolvedCell[x][y] {
		return basicBacktracking(grid, isUnsolvedCell, possibility, possibilitySize, x, y+1)
	} else {
		for k := 0; k < possibilitySize[x][y]; k++ {
			if isFillable(grid, x, y, possibility[x][y][k]) {
				grid[x][y] = possibility[x][y][k]
				if basicBacktracking(grid, isUnsolvedCell, possibility, possibilitySize, x, y+1) {
					return true
				}
			}
		}
		grid[x][y] = 0
		return false
	}
}

func isFillable(grid *[BOARD_SIZE][BOARD_SIZE]int, x int, y int, k int) bool {
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
