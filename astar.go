package main

import (
	"container/heap"
	"runtime"
	"sync"
	"time"
)

type AStarResult struct {
	Grid      SudokuGrid
	Steps     int
	TimeTaken time.Duration
	Solved    bool
}

type Node struct {
	grid      SudokuGrid
	cost      int
	heuristic int
	priority  int
	index     int
}

type PriorityQueue []*Node

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	node := x.(*Node)
	node.index = n
	*pq = append(*pq, node)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil  // avoid memory leak
	node.index = -1 // for safety
	*pq = old[0 : n-1]
	return node
}

func (pq *PriorityQueue) update(node *Node, grid SudokuGrid, cost int, heuristic int) {
	node.grid = grid
	node.cost = cost
	node.heuristic = heuristic
	node.priority = cost + heuristic
	heap.Fix(pq, node.index)
}

func solveAStar(grid SudokuGrid) AStarResult {
	start := time.Now()
	var steps int
	solvedGrid, solved := solveAStarSearch(grid.Copy(), &steps)
	return AStarResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}

func solveAStarSearch(initialGrid SudokuGrid, steps *int) (SudokuGrid, bool) {
	pq := make(PriorityQueue, 0)
	initialHeuristic := calculateHeuristic(initialGrid)
	heap.Push(&pq, &Node{grid: initialGrid, cost: 0, heuristic: initialHeuristic, priority: initialHeuristic})

	visited := make(map[string]bool)

	for pq.Len() > 0 {
		currentNode := heap.Pop(&pq).(*Node)
		*steps++

		if isSolved(currentNode.grid) {
			return currentNode.grid, true
		}

		gridString := gridToString(currentNode.grid)
		if visited[gridString] {
			continue
		}
		visited[gridString] = true

		row, col, found := findEmptyCell(currentNode.grid)
		if !found {
			continue // Should not happen if isSolved is working correctly
		}

		for num := 1; num <= 9; num++ {
			if isValidMove(currentNode.grid, row, col, num) {
				newGrid := currentNode.grid.Copy()
				newGrid[row][col] = num
				newHeuristic := calculateHeuristic(newGrid)
				heap.Push(&pq, &Node{grid: newGrid, cost: currentNode.cost + 1, heuristic: newHeuristic, priority: currentNode.cost + 1 + newHeuristic})
			}
		}
	}
	return initialGrid, false
}

func calculateHeuristic(grid SudokuGrid) int {
	conflicts := 0
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if grid[i][j] == 0 {
				continue
			}
			for k := j + 1; k < 9; k++ {
				if grid[i][j] == grid[i][k] {
					conflicts++
				}
			}
			for k := i + 1; k < 9; k++ {
				if grid[i][j] == grid[k][j] {
					conflicts++
				}
			}
			startRow := i - i%3
			startCol := j - j%3
			for row := startRow; row < startRow+3; row++ {
				for col := startCol; col < startCol+3; col++ {
					if (row != i || col != j) && grid[i][j] == grid[row][col] {
						conflicts++
					}
				}
			}
		}
	}
	// Add a heuristic based on the number of possible values for each empty cell
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if grid[i][j] == 0 {
				possibleValues := 0
				for num := 1; num <= 9; num++ {
					if isValidMove(grid, i, j, num) {
						possibleValues++
					}
				}
				conflicts += (9 - possibleValues) // Penalize cells with fewer options
			}
		}
	}
	return conflicts
}

func isSolved(grid SudokuGrid) bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if grid[i][j] == 0 {
				return false
			}
		}
	}
	return calculateHeuristic(grid) == 0
}

func gridToString(grid SudokuGrid) string {
	var sb strings.Builder
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			sb.WriteString(fmt.Sprintf("%d", grid[i][j]))
		}
	}
	return sb.String()
}

// Parallelized A* with Work Stealing
func solveAStarParallel(grid SudokuGrid) AStarResult {
	start := time.Now()
	var steps int
	var solvedGrid SudokuGrid
	var solved bool

	numWorkers := runtime.NumCPU()
	workChan := make(chan *Node, numWorkers)
	resultChan := make(chan struct {
		grid   SudokuGrid
		solved bool
	}, numWorkers)
	var wg sync.WaitGroup

	// Initialize priority queue
	pq := make(PriorityQueue, 0)
	initialHeuristic := calculateHeuristic(grid)
	heap.Push(&pq, &Node{grid: grid, cost: 0, heuristic: initialHeuristic, priority: initialHeuristic})

	visited := make(map[string]bool)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for node := range workChan {
				*steps++
				if isSolved(node.grid) {
					resultChan <- struct {
						grid   SudokuGrid
						solved bool
					}{grid: node.grid, solved: true}
					return
				}

				gridString := gridToString(node.grid)
				if visited[gridString] {
					continue
				}
				visited[gridString] = true

				row, col, found := findEmptyCell(node.grid)
				if !found {
					continue
				}

				for num := 1; num <= 9; num++ {
					if isValidMove(node.grid, row, col, num) {
						newGrid := node.grid.Copy()
						newGrid[row][col] = num
						newHeuristic := calculateHeuristic(newGrid)
						heap.Push(&pq, &Node{grid: newGrid, cost: node.cost + 1, heuristic: newHeuristic, priority: node.cost + 1 + newHeuristic})
					}
				}
			}
		}()
	}

	// Initial work
	for pq.Len() > 0 {
		currentNode := heap.Pop(&pq).(*Node)
		workChan <- currentNode
		if solved {
			break
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

	return AStarResult{
		Grid:      solvedGrid,
		Steps:     steps,
		TimeTaken: time.Since(start),
		Solved:    solved,
	}
}
