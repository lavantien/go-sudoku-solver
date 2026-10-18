package puzzle

import (
	"fmt"
	"math/rand/v2"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

type GenResult struct {
	Grid  sudoku.Grid
	Clues int
}

func Generate(clues int, seed uint64, unique func(sudoku.Grid, int) int) (GenResult, error) {
	if clues < 17 || clues > config.Cells {
		return GenResult{}, fmt.Errorf("generator clues must be between 17 and %d", config.Cells)
	}
	if unique == nil {
		return GenResult{}, fmt.Errorf("uniqueness checker required")
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	g := shuffledFullGrid(rng)
	cells := make([]int, config.Cells)
	for i := range cells {
		cells[i] = i
	}
	rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	remaining := config.Cells
	for _, i := range cells {
		if remaining <= clues {
			break
		}
		keep := g[i]
		g[i] = config.EmptyDigit
		if unique(g, 2) != 1 {
			g[i] = keep
			continue
		}
		remaining--
	}
	return GenResult{Grid: g, Clues: remaining}, nil
}

func baseGrid() sudoku.Grid {
	var g sudoku.Grid
	for r := range config.Size {
		for c := range config.Size {
			g[r*config.Size+c] = uint8((r*config.Box+r/config.Box+c)%config.Size + 1)
		}
	}
	return g
}

func shuffledFullGrid(rng *rand.Rand) sudoku.Grid {
	g := baseGrid()
	bands := shuffledTriple(rng)
	stacks := shuffledTriple(rng)
	var rowPerm [config.Box][config.Box]int
	var colPerm [config.Box][config.Box]int
	for b := range config.Box {
		rowPerm[b] = shuffledTriple(rng)
		colPerm[b] = shuffledTriple(rng)
	}
	relabel := shuffledDigits(rng)
	var out sudoku.Grid
	for r := range config.Size {
		newR := bands[r/config.Box]*config.Box + rowPerm[r/config.Box][r%config.Box]
		for c := range config.Size {
			newC := stacks[c/config.Box]*config.Box + colPerm[c/config.Box][c%config.Box]
			out[newR*config.Size+newC] = relabel[g[r*config.Size+c]]
		}
	}
	if rng.IntN(2) == 0 {
		out = transpose(out)
	}
	return out
}

func shuffledTriple(rng *rand.Rand) [config.Box]int {
	p := [config.Box]int{0, 1, 2}
	rng.Shuffle(config.Box, func(i, j int) { p[i], p[j] = p[j], p[i] })
	return p
}

func shuffledDigits(rng *rand.Rand) [config.Size + 1]uint8 {
	digits := []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	rng.Shuffle(config.Size, func(i, j int) { digits[i+1], digits[j+1] = digits[j+1], digits[i+1] })
	var out [config.Size + 1]uint8
	copy(out[:], digits)
	return out
}

func transpose(g sudoku.Grid) sudoku.Grid {
	var out sudoku.Grid
	for r := range config.Size {
		for c := range config.Size {
			out[c*config.Size+r] = g[r*config.Size+c]
		}
	}
	return out
}
