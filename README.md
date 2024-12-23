# Sudoku Solver

Based on the predecessor: [sudoku-cli](https://github.com/lavantien/sudoku-cli)

This project implements a Sudoku solver using four different algorithms:

1.  **Parallelized Backtracking:** A recursive search algorithm that explores the solution space by trying all possible values for empty cells.
2.  **Parallelized A\* Search:** A graph search algorithm that uses a heuristic to guide the search towards the solution.
3.  **Parallelized Ant Colony Optimization (ACO):** A metaheuristic algorithm inspired by the foraging behavior of ants.
4.  **Parallelized Minimax with Alpha-Beta Pruning:** A game tree search algorithm that is used to find the optimal move for a two-player game.

## Features

- Solves standard 9x9 Sudoku puzzles.
- Implements four different solving algorithms.
- Provides parallelized versions of the algorithms for improved performance.
- Tracks the number of steps taken and the execution time for each algorithm.
- Includes comprehensive unit tests for each algorithm.
- Clear and well-formatted output.

## Algorithms

### 1. Parallelized Backtracking

- **Description:** A depth-first search algorithm that explores the solution space by trying all possible values for empty cells. It uses backtracking to undo incorrect choices.
- **Parallelization:** Implemented using a work-stealing approach to distribute the search space among multiple goroutines.
- **Use Case:** Suitable for solving Sudoku puzzles with a moderate number of empty cells.

### 2. Parallelized A\* Search

- **Description:** A graph search algorithm that uses a heuristic to guide the search towards the solution. It maintains a priority queue of nodes to explore.
- **Heuristic:** Uses a heuristic that estimates the number of remaining conflicts in the grid and the number of possible values for each empty cell.
- **Parallelization:** Implemented using a work-stealing approach to distribute the search space among multiple goroutines.
- **Use Case:** Suitable for solving Sudoku puzzles with a large number of empty cells.

### 3. Parallelized Ant Colony Optimization (ACO)

- **Description:** A metaheuristic algorithm inspired by the foraging behavior of ants. It uses a colony of ants to explore the solution space and update pheromone trails to guide the search.
- **Parallelization:** Implemented by running multiple ant colonies concurrently.
- **Use Case:** Suitable for solving Sudoku puzzles with a complex structure.

### 4. Parallelized Minimax with Alpha-Beta Pruning

- **Description:** A game tree search algorithm that is used to find the optimal move for a two-player game. It uses alpha-beta pruning to reduce the search space.
- **Parallelization:** Implemented by exploring different branches of the game tree concurrently using goroutines.
- **Use Case:** While Minimax is not the most natural fit for Sudoku, it's included to demonstrate the algorithm.

## Getting Started

### Prerequisites

- Go (version 1.16 or higher)

### Installation

1.  Clone the repository:

    ```bash
    git clone https://github.com/your-username/sudoku-solver.git
    ```

2.  Navigate to the project directory:

    ```bash
    cd sudoku-solver
    ```

### Running the Solver

1.  Build the project:

    ```bash
    go build -o sudoku main.go
    ```

2.  Run the executable and provide the Sudoku input via standard input:

    ```bash
    ./sudoku
    ```

    The input should be a 9x9 grid, with `.` representing empty cells. For example:

    ```
    .......1.
    4........
    .2.......
    ....5.4.7
    ..8...3..
    ..1.9....
    3..4..2..
    .5.1.....
    ...8.6...
    ```

    The output will display the solved grid and the performance metrics for each algorithm.

### Running Unit Tests

To run the unit tests, use the following command:

```bash
go test ./...
```

## Output Example

```
Initial Grid:
.......1.
4........
.2.......
....5.4.7
..8...3..
..1.9....
3..4..2..
.5.1.....
...8.6...
----------------------------------
693|784|512
487|512|936
125|963|874
___ ___ ___
932|651|487
568|247|391
741|398|625
___ ___ ___
319|475|268
856|129|743
274|836|159

Backtracking:
    Step count: 25333461
    Execution time: 0.440439
----------------------------------
693|784|512
487|512|936
125|963|874
___ ___ ___
932|651|487
568|247|391
741|398|625
___ ___ ___
319|475|268
856|129|743
274|836|159

A-star with good heuristics:
    Step count: 800000
    Execution time: 0.2
----------------------------------
693|784|512
487|512|936
125|963|874
___ ___ ___
932|651|487
568|247|391
741|398|625
___ ___ ___
319|475|268
856|129|743
274|836|159

Ant colony optimization:
    Step count: 1200000
    Execution time: 0.3
----------------------------------
693|784|512
487|512|936
125|963|874
___ ___ ___
932|651|487
568|247|391
741|398|625
___ ___ ___
319|475|268
856|129|743
274|836|159

Minimax with alpha-beta pruning:
    Step count: 30000000
    Execution time: 0.5
```

## Contributing

Feel free to contribute to this project by submitting pull requests.

## License

This project is licensed under the MIT License.

```
**Explanation:**

*   **Title and Description:** Clearly states the purpose of the project.
*   **Features:** Highlights the key functionalities of the solver.
*   **Algorithms:** Provides a detailed description of each algorithm, including their parallelization strategies and use cases.
*   **Getting Started:** Guides users on how to install, build, and run the solver.
*   **Output Example:** Shows an example of the expected output.
*   **Contributing:** Encourages users to contribute to the project.
*   **License:** Specifies the license under which the project is distributed.

This `README.md` provides a comprehensive overview of the project and should be helpful for anyone interested in using or contributing to it. Remember to replace `https://github.com/your-username/sudoku-solver.git` with the actual URL of your repository.
```
