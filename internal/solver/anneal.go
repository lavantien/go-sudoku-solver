package solver

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const annealScale = 1000

func halvings(t0, tEnd int64) int {
	n := 1
	for t := t0; t > tEnd && n < 62; t >>= 1 {
		n++
	}
	return n
}

func Anneal(ctx context.Context, g sudoku.Grid, c config.Config) Result {
	logger := logging.New(c.LogLevel)
	logging.Enter(logger, "anneal", "clues", clueCount(g))
	start := time.Now()
	free := freeByRow(g)
	steps := 0
	best := g
	bestCost := 1 << 30
	for r := range c.Anneal.Restarts {
		if ctx.Err() != nil {
			break
		}
		grid, cost := annealRestart(ctx, g, c, uint64(r), free, &steps)
		if cost < bestCost {
			best, bestCost = grid, cost
		}
		if cost == 0 {
			logging.Exit(logger, "anneal", "steps", steps, "restart", r)
			return Result{Grid: grid, Steps: steps, Dur: time.Since(start), Solved: true}
		}
	}
	logging.Fail(logger, "anneal", errUnsolved, "best_cost", bestCost)
	return Result{Grid: best, Steps: steps, Dur: time.Since(start), Solved: false}
}

func freeByRow(g sudoku.Grid) [config.Size][]int {
	var free [config.Size][]int
	for i := range config.Cells {
		if g[i] == config.EmptyDigit {
			free[sudoku.CellRow(i)] = append(free[sudoku.CellRow(i)], i)
		}
	}
	return free
}

func annealRestart(ctx context.Context, g sudoku.Grid, c config.Config, restart uint64, free [config.Size][]int, steps *int) (sudoku.Grid, int) {
	rng := rand.New(rand.NewPCG(c.Anneal.Seed, restart))
	cur := seedGrid(g, rng)
	curCost := totalCost(cur)
	if curCost == 0 {
		return cur, 0
	}
	bank := cur
	bankCost := curCost
	t0 := c.Anneal.TempStart
	span := c.Anneal.Iterations / halvings(c.Anneal.TempStart, c.Anneal.TempEnd)
	if span < 1 {
		span = 1
	}
	clock := 0
	stuck := false
	var buf [config.Cells]int
	for range c.Anneal.Iterations {
		if ctx.Err() != nil {
			return bank, bankCost
		}
		temp := tempAt(t0, c.Anneal.TempEnd, clock, span)
		clock++
		conflicted := conflictedCells(cur, buf[:0])
		if !anySwappable(conflicted, free) {
			if stuck {
				break
			}
			stuck = true
			continue
		}
		stuck = false
		move, ok := pickConflictedMove(conflicted, free, rng)
		if !ok {
			continue
		}
		delta := moveDelta(cur, move)
		*steps++
		accept := delta <= 0
		if !accept {
			climb := delta * annealScale
			accept = rng.IntN(int(temp)+climb) < int(temp)
		}
		if accept {
			applyMove(&cur, move)
			curCost += delta
			if curCost < bankCost {
				bank, bankCost = cur, curCost
			}
			if curCost == 0 {
				return cur, 0
			}
		}
	}
	return bank, bankCost
}

func tempAt(t0, tEnd int64, clock, span int) int64 {
	t := t0 >> uint(clock/span)
	if t < tEnd {
		return tEnd
	}
	return t
}

func seedGrid(g sudoku.Grid, rng *rand.Rand) sudoku.Grid {
	cur := g
	for r := range config.Size {
		used := sudoku.RowUsed(g, r)
		var missing []uint8
		for d := 1; d <= config.Size; d++ {
			if used&(1<<(d-1)) == 0 {
				missing = append(missing, uint8(d))
			}
		}
		rng.Shuffle(len(missing), func(i, j int) { missing[i], missing[j] = missing[j], missing[i] })
		k := 0
		for c := range config.Size {
			if cur[r*config.Size+c] == config.EmptyDigit {
				cur[r*config.Size+c] = missing[k]
				k++
			}
		}
	}
	return cur
}

const annealRotateRate = 5

func pickMove(cur sudoku.Grid, free [config.Size][]int, rng *rand.Rand) ([3]int, bool) {
	var buf [config.Cells]int
	return pickConflictedMove(conflictedCells(cur, buf[:0]), free, rng)
}

func anySwappable(conflicted []int, free [config.Size][]int) bool {
	for _, a := range conflicted {
		rowFree := free[sudoku.CellRow(a)]
		if len(rowFree) >= 2 && indexOf(rowFree, a) >= 0 {
			return true
		}
	}
	return false
}

func pickConflictedMove(conflicted []int, free [config.Size][]int, rng *rand.Rand) ([3]int, bool) {
	if len(conflicted) == 0 || !anySwappable(conflicted, free) {
		return [3]int{}, false
	}
	for range 16 {
		a := conflicted[rng.IntN(len(conflicted))]
		rowFree := free[sudoku.CellRow(a)]
		if len(rowFree) < 2 {
			continue
		}
		i := indexOf(rowFree, a)
		if i < 0 {
			continue
		}
		if len(rowFree) >= 3 && rng.IntN(annealRotateRate) == 0 {
			k := rng.IntN(len(rowFree) - 1)
			if k >= i {
				k++
			}
			var third int
			for {
				third = rng.IntN(len(rowFree) - 1)
				if third >= i {
					third++
				}
				if third != k {
					break
				}
			}
			return [3]int{a, rowFree[k], rowFree[third]}, true
		}
		j := rng.IntN(len(rowFree) - 1)
		if j >= i {
			j++
		}
		return [3]int{a, rowFree[j], -1}, true
	}
	return [3]int{}, false
}

func applyMove(cur *sudoku.Grid, move [3]int) {
	a, b, c := move[0], move[1], move[2]
	if c < 0 {
		cur[a], cur[b] = cur[b], cur[a]
		return
	}
	cur[a], cur[b], cur[c] = cur[c], cur[a], cur[b]
}

func moveDelta(cur sudoku.Grid, move [3]int) int {
	a, b, c := move[0], move[1], move[2]
	units := []int{
		config.Size + sudoku.CellCol(a),
		config.Size + sudoku.CellCol(b),
		2*config.Size + sudoku.CellBox(a),
		2*config.Size + sudoku.CellBox(b),
	}
	if c >= 0 {
		units = append(units, config.Size+sudoku.CellCol(c), 2*config.Size+sudoku.CellBox(c))
	}
	before := unitsCost(cur, units)
	applyMove(&cur, move)
	after := unitsCost(cur, units)
	return after - before
}

func unitsCost(g sudoku.Grid, units []int) int {
	total := 0
	seen := map[int]bool{}
	for _, u := range units {
		if seen[u] {
			continue
		}
		seen[u] = true
		total += unitCost(g, u)
	}
	return total
}

func indexOf(cells []int, want int) int {
	for i, c := range cells {
		if c == want {
			return i
		}
	}
	return -1
}

func conflictedCells(cur sudoku.Grid, out []int) []int {
	for u := range 3 * config.Size {
		seen := [config.Size + 1]int{}
		unit := sudoku.UnitCells(u)
		for _, i := range unit {
			seen[cur[i]]++
		}
		for _, i := range unit {
			if cur[i] != config.EmptyDigit && seen[cur[i]] > 1 {
				out = append(out, i)
			}
		}
	}
	return out
}

func unitCost(g sudoku.Grid, u int) int {
	var counts [config.Size + 1]int
	for _, i := range sudoku.UnitCells(u) {
		counts[g[i]]++
	}
	cost := 0
	for d := 1; d <= config.Size; d++ {
		cost += counts[d] * (counts[d] - 1) / 2
	}
	return cost
}

func totalCost(g sudoku.Grid) int {
	cost := 0
	for u := range 3 * config.Size {
		cost += unitCost(g, u)
	}
	return cost
}
