package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"sort"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const (
	evolveSeed     = uint64(20261019)
	popSize        = 40
	genBudget      = 400
	eliteCount     = 2
	tournamentK    = 3
	minClues       = 20
	maxClues       = 40
	crossoverPct   = 10
	relabelPct     = 25
	swapPct        = 30
	removePct      = 35
	redigAttempts  = 12
	evalCap        = 20 * time.Second
	runCap         = 10 * time.Minute
	progressStride = 40
	seedStride     = 104729
	seedAttempts   = 8
)

var seedTargets = [9]int{20, 21, 22, 23, 24, 25, 26, 28, 32}

type individual struct {
	grid  sudoku.Grid
	sol   sudoku.Grid
	steps int
	clues int
}

type counters struct {
	crossoverAttempt int
	crossoverAccept  int
	relabelAttempt   int
	relabelAccept    int
	swapAttempt      int
	swapAccept       int
	swapRepair       int
	removeAttempt    int
	removeAccept     int
	removeRepair     int
	addAttempt       int
	addAccept        int
}

func main() {
	logger := logging.New(config.Load().LogLevel)
	if err := run(logger); err != nil {
		fmt.Fprintln(os.Stderr, "evolve:", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	logging.Enter(logger, "evolve", "seed", evolveSeed, "pop", popSize, "gens", genBudget)
	start := time.Now()
	rng := rand.New(rand.NewPCG(evolveSeed, evolveSeed))
	pop, err := seedPopulation(logger)
	if err != nil {
		return err
	}
	progress := []int{pop[0].steps}
	gens := 0
	stop := "generation budget"
	c := &counters{}
	for gens < genBudget {
		if time.Since(start) >= runCap {
			stop = "wall clock cap"
			break
		}
		logging.Enter(logger, "generation", "gen", gens+1, "best", pop[0].steps)
		pop = nextGeneration(rng, pop, c)
		gens++
		progress = append(progress, pop[0].steps)
		logging.Exit(logger, "generation", "gen", gens, "best", pop[0].steps)
	}
	report(pop, progress, gens, stop, time.Since(start), c)
	logging.Exit(logger, "evolve", "gens", gens, "best", pop[0].steps)
	return nil
}

func seedPopulation(logger *slog.Logger) ([]individual, error) {
	logging.Enter(logger, "seed-population", "size", popSize)
	pop := make([]individual, 0, popSize)
	generated := popSize - 1
	for i := range generated {
		target := seedTargets[i%len(seedTargets)]
		ind, err := generatedSeed(i, target)
		if err != nil {
			logging.Fail(logger, "seed-population", err, "target", target)
			return nil, fmt.Errorf("seed-population: %w", err)
		}
		logger.Debug("seeded", "i", i, "target", target, "clues", ind.clues, "steps", ind.steps)
		pop = append(pop, ind)
	}
	legacy, err := legacyIndividual()
	if err != nil {
		logging.Fail(logger, "seed-population", err)
		return nil, fmt.Errorf("seed-population: %w", err)
	}
	logger.Debug("seeded", "i", generated, "target", "legacy", "clues", legacy.clues, "steps", legacy.steps)
	pop = append(pop, legacy)
	sortPopulation(pop)
	logging.Exit(logger, "seed-population", "best", pop[0].steps, "worst", pop[len(pop)-1].steps)
	return pop, nil
}

func generatedSeed(i, target int) (individual, error) {
	var lastErr error
	for attempt := 0; attempt < seedAttempts; attempt++ {
		res, err := puzzle.Generate(target, evolveSeed+uint64(i)*seedStride+uint64(attempt), solver.CountSolutions)
		if err != nil {
			lastErr = fmt.Errorf("seed %d attempt %d: %w", i, attempt, err)
			continue
		}
		if ind, ok := evaluate(res.Grid); ok {
			return ind, nil
		}
		lastErr = fmt.Errorf("seed %d attempt %d: generated grid rejected", i, attempt)
	}
	return individual{}, lastErr
}

func legacyIndividual() (individual, error) {
	g, err := sudoku.Parse81(puzzle.Sample81)
	if err != nil {
		return individual{}, fmt.Errorf("legacy sample: %w", err)
	}
	res := solver.Propagate(context.Background(), g, config.Defaults())
	if !res.Solved {
		return individual{}, fmt.Errorf("legacy sample unsolved by the engine")
	}
	for i := range g {
		if cluesOf(g) >= minClues {
			break
		}
		if g[i] == config.EmptyDigit {
			g[i] = res.Grid[i]
		}
	}
	ind, ok := evaluate(g)
	if !ok {
		return individual{}, fmt.Errorf("topped-up legacy sample rejected")
	}
	return ind, nil
}

func evaluate(g sudoku.Grid) (individual, bool) {
	if n := cluesOf(g); n < minClues || n > maxClues {
		return individual{}, false
	}
	if !isUnique(g) {
		return individual{}, false
	}
	return evaluateSolve(g)
}

func evaluateSolve(g sudoku.Grid) (individual, bool) {
	if n := cluesOf(g); n < minClues || n > maxClues {
		return individual{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), evalCap)
	defer cancel()
	res := solver.Propagate(ctx, g, config.Defaults())
	if !res.Solved {
		return individual{}, false
	}
	return individual{grid: g, sol: res.Grid, steps: res.Steps, clues: cluesOf(g)}, true
}

func isUnique(g sudoku.Grid) bool {
	return solver.CountSolutions(g, 2) == 1
}

func nextGeneration(rng *rand.Rand, pop []individual, c *counters) []individual {
	next := make([]individual, 0, popSize)
	next = append(next, pop[:eliteCount]...)
	for len(next) < popSize {
		a := tournament(rng, pop)
		if rng.IntN(100) < crossoverPct {
			b := tournament(rng, pop)
			next = append(next, crossoverChild(rng, a, b, c))
			continue
		}
		next = append(next, mutateChild(rng, a, c))
	}
	sortPopulation(next)
	return next
}

func tournament(rng *rand.Rand, pop []individual) individual {
	best := pop[rng.IntN(len(pop))]
	for range tournamentK - 1 {
		challenger := pop[rng.IntN(len(pop))]
		if challenger.steps > best.steps {
			best = challenger
		}
	}
	return best
}

func mutateChild(rng *rand.Rand, parent individual, c *counters) individual {
	draw := rng.IntN(100)
	switch {
	case draw < relabelPct:
		c.relabelAttempt++
		child, ok := evaluateSolve(relabelGivens(rng, parent.grid))
		if ok {
			c.relabelAccept++
			return child
		}
	case draw < relabelPct+swapPct:
		c.swapAttempt++
		child, ok := swapCandidate(rng, parent, &c.swapRepair)
		if ok {
			c.swapAccept++
			return child
		}
	case draw < relabelPct+swapPct+removePct:
		c.removeAttempt++
		child, ok := removeCandidate(rng, parent, &c.removeRepair)
		if ok {
			c.removeAccept++
			return child
		}
	default:
		c.addAttempt++
		child, ok := evaluateSolve(addClue(rng, parent.grid, parent.sol))
		if ok {
			c.addAccept++
			return child
		}
	}
	return parent
}

func swapCandidate(rng *rand.Rand, parent individual, repair *int) (individual, bool) {
	candidate := swapWithinBand(rng, parent.grid)
	if candidate == parent.grid {
		return parent, false
	}
	if isUnique(candidate) {
		return evaluateSolve(candidate)
	}
	*repair++
	return repairedChild(rng, candidate, parent.clues)
}

func removeCandidate(rng *rand.Rand, parent individual, repair *int) (individual, bool) {
	candidate := removeClue(rng, parent.grid)
	if candidate == parent.grid {
		return parent, false
	}
	if isUnique(candidate) {
		return evaluateSolve(candidate)
	}
	*repair++
	return repairedChild(rng, candidate, parent.clues)
}

func repairedChild(rng *rand.Rand, ruined sudoku.Grid, targetClues int) (individual, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), evalCap)
	defer cancel()
	res := solver.Propagate(ctx, ruined, config.Defaults())
	if !res.Solved {
		return individual{}, false
	}
	out := ruined
	for !isUnique(out) {
		empties := emptyCells(out)
		if len(empties) == 0 || cluesOf(out) >= maxClues {
			return individual{}, false
		}
		at := empties[rng.IntN(len(empties))]
		out[at] = res.Grid[at]
	}
	for range redigAttempts {
		if cluesOf(out) <= targetClues {
			break
		}
		givens := givenCells(out)
		i := givens[rng.IntN(len(givens))]
		keep := out[i]
		out[i] = config.EmptyDigit
		if !isUnique(out) {
			out[i] = keep
		}
	}
	return evaluateSolve(out)
}

func crossoverChild(rng *rand.Rand, a, b individual, c *counters) individual {
	c.crossoverAttempt++
	lead, other := a, b
	if b.steps > a.steps {
		lead, other = b, a
	}
	var child sudoku.Grid
	for i := range child {
		lv, ov := lead.grid[i], other.grid[i]
		switch {
		case lv == config.EmptyDigit && ov == config.EmptyDigit:
		case ov == config.EmptyDigit || lv == ov:
			child[i] = lv
		case lv == config.EmptyDigit:
			child[i] = ov
		default:
			child[i] = lv
		}
	}
	for cluesOf(child) > maxClues {
		givens := givenCells(child)
		child[givens[rng.IntN(len(givens))]] = config.EmptyDigit
	}
	if child == lead.grid {
		return lead
	}
	if child, ok := evaluate(child); ok {
		c.crossoverAccept++
		return child
	}
	return lead
}

func relabelGivens(rng *rand.Rand, g sudoku.Grid) sudoku.Grid {
	perm := []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9}
	rng.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })
	var out sudoku.Grid
	for i := range g {
		if g[i] != config.EmptyDigit {
			out[i] = perm[g[i]-1]
		}
	}
	return out
}

func swapWithinBand(rng *rand.Rand, g sudoku.Grid) sudoku.Grid {
	givens := givenCells(g)
	if len(givens) == 0 {
		return g
	}
	from := givens[rng.IntN(len(givens))]
	band := from / (3 * config.Size) * (3 * config.Size)
	var targets []int
	for i := band; i < band+3*config.Size; i++ {
		if g[i] == config.EmptyDigit && placeableExcept(g, i, from, g[from]) {
			targets = append(targets, i)
		}
	}
	if len(targets) == 0 {
		return g
	}
	to := targets[rng.IntN(len(targets))]
	out := g
	out[to] = out[from]
	out[from] = config.EmptyDigit
	return out
}

func placeableExcept(g sudoku.Grid, at, except int, d uint8) bool {
	r, c, b := sudoku.CellRow(at), sudoku.CellCol(at), sudoku.CellBox(at)
	for i := range g {
		if i == at || i == except || g[i] != d {
			continue
		}
		if sudoku.CellRow(i) == r || sudoku.CellCol(i) == c || sudoku.CellBox(i) == b {
			return false
		}
	}
	return true
}

func removeClue(rng *rand.Rand, g sudoku.Grid) sudoku.Grid {
	if cluesOf(g) <= minClues {
		return g
	}
	givens := givenCells(g)
	out := g
	out[givens[rng.IntN(len(givens))]] = config.EmptyDigit
	return out
}

func addClue(rng *rand.Rand, g, sol sudoku.Grid) sudoku.Grid {
	if cluesOf(g) >= maxClues {
		return g
	}
	empties := emptyCells(g)
	out := g
	at := empties[rng.IntN(len(empties))]
	out[at] = sol[at]
	return out
}

func cluesOf(g sudoku.Grid) int {
	n := 0
	for i := range g {
		if g[i] != config.EmptyDigit {
			n++
		}
	}
	return n
}

func givenCells(g sudoku.Grid) []int {
	var out []int
	for i := range g {
		if g[i] != config.EmptyDigit {
			out = append(out, i)
		}
	}
	return out
}

func emptyCells(g sudoku.Grid) []int {
	var out []int
	for i := range g {
		if g[i] == config.EmptyDigit {
			out = append(out, i)
		}
	}
	return out
}

func sortPopulation(pop []individual) {
	sort.SliceStable(pop, func(i, j int) bool { return pop[i].steps > pop[j].steps })
}

func report(pop []individual, progress []int, gens int, stop string, dur time.Duration, c *counters) {
	best := pop[0]
	fmt.Printf("evolve run: seed %d population %d generations %d elites %d tournament %d\n",
		evolveSeed, popSize, genBudget, eliteCount, tournamentK)
	fmt.Printf("clue band %d..%d crossover %d%% mutation: relabel %d%% swap %d%% remove %d%% add %d%% redig attempts %d\n",
		minClues, maxClues, crossoverPct, relabelPct, swapPct, removePct, 100-relabelPct-swapPct-removePct, redigAttempts)
	fmt.Printf("stopped by %s after %d generations in %s\n", stop, gens, dur.Round(time.Millisecond))
	fmt.Printf("operators accepted/attempted: crossover %d/%d relabel %d/%d swap %d/%d remove %d/%d add %d/%d repairs swap %d remove %d\n",
		c.crossoverAccept, c.crossoverAttempt,
		c.relabelAccept, c.relabelAttempt,
		c.swapAccept, c.swapAttempt,
		c.removeAccept, c.removeAttempt,
		c.addAccept, c.addAttempt,
		c.swapRepair, c.removeRepair)
	fmt.Println("best steps by generation:")
	for i, s := range progress {
		if i == 0 || i == len(progress)-1 || i%progressStride == 0 {
			fmt.Printf("  gen %d: %d\n", i, s)
		}
	}
	stepsSum, cluesSum, minSteps := 0, best.clues, best.steps
	maxCluesSeen := 0
	for _, ind := range pop {
		stepsSum += ind.steps
		cluesSum += ind.clues
		if ind.steps < minSteps {
			minSteps = ind.steps
		}
		if ind.clues > maxCluesSeen {
			maxCluesSeen = ind.clues
		}
	}
	fmt.Printf("final population: mean steps %d min steps %d mean clues %d clue range %d..%d\n",
		stepsSum/len(pop), minSteps, cluesSum/len(pop), best.clues, maxCluesSeen)
	fmt.Printf("winner: clues %d steps %d\n", best.clues, best.steps)
	fmt.Println("puzzle:")
	printGrid(best.grid)
	fmt.Println("solution:")
	printGrid(best.sol)
	fmt.Printf("puzzle81 %s\n", sudoku.Serialize(best.grid))
	fmt.Printf("solution81 %s\n", sudoku.Serialize(best.sol))
}

func printGrid(g sudoku.Grid) {
	for r := range config.Size {
		for c := range config.Size {
			if g[r*config.Size+c] == config.EmptyDigit {
				fmt.Print(".")
			} else {
				fmt.Print(g[r*config.Size+c])
			}
		}
		fmt.Println()
	}
}
