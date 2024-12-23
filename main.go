package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	grid, err := readSudokuInput()
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	fmt.Println("Initial Grid:")
	printGrid(grid)

	// Backtracking
	backtrackingResult := solveBacktrackingParallel(grid)
	fmt.Println("----------------------------------")
	printGrid(backtrackingResult.Grid)
	fmt.Println("\nBacktracking:")
	fmt.Printf("    Step count: %d\n", backtrackingResult.Steps)
	fmt.Printf("    Execution time: %f\n", backtrackingResult.TimeTaken.Seconds())
	fmt.Println("----------------------------------")

	// A*
	aStarResult := solveAStarParallel(grid)
	fmt.Println("----------------------------------")
	printGrid(aStarResult.Grid)
	fmt.Println("\nA-star with good heuristics:")
	fmt.Printf("    Step count: %d\n", aStarResult.Steps)
	fmt.Printf("    Execution time: %f\n", aStarResult.TimeTaken.Seconds())
	fmt.Println("----------------------------------")

	// ACO
	acoResult := solveACOParallel(grid)
	fmt.Println("----------------------------------")
	printGrid(acoResult.Grid)
	fmt.Println("\nAnt colony optimization:")
	fmt.Printf("    Step count: %d\n", acoResult.Steps)
	fmt.Printf("    Execution time: %f\n", acoResult.TimeTaken.Seconds())
	fmt.Println("----------------------------------")

	// Minimax
	minimaxResult := solveMinimaxParallel(grid)
	fmt.Println("----------------------------------")
	printGrid(minimaxResult.Grid)
	fmt.Println("\nMinimax with alpha-beta pruning:")
	fmt.Printf("    Step count: %d\n", minimaxResult.Steps)
	fmt.Printf("    Execution time: %f\n", minimaxResult.TimeTaken.Seconds())
	fmt.Println("----------------------------------")
}

func readSudokuInput() ([][]int, error) {
	grid := make([][]int, 9)
	scanner := bufio.NewScanner(os.Stdin)
	for i := 0; i < 9; i++ {
		if !scanner.Scan() {
			return nil, fmt.Errorf("not enough lines in input")
		}
		line := scanner.Text()
		if len(line) != 9 {
			return nil, fmt.Errorf("invalid line length: %d, expected 9", len(line))
		}
		grid[i] = make([]int, 9)
		for j, char := range line {
			if char == '.' {
				grid[i][j] = 0
			} else if char >= '1' && char <= '9' {
				grid[i][j] = int(char - '0')
			} else {
				return nil, fmt.Errorf("invalid character: %c", char)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return grid, nil
}

func printGrid(grid [][]int) {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			fmt.Print(grid[i][j])
			if (j+1)%3 == 0 && j != 8 {
				fmt.Print("|")
			}
		}
		fmt.Println()
		if (i+1)%3 == 0 && i != 8 {
			fmt.Println("___ ___ ___")
		}
	}
}
