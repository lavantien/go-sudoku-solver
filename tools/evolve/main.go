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
	os.Exit(cli(config.Load()))
}

func cli(cfg config.Config) int {
	logger := logging.New(cfg.LogLevel)
	if err := run(logger, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "evolve:", err)
		return 1
	}
	return 0
}

func run(logger *slog.Logger, cfg config.Config) error {
	evo := cfg.Evolve
	logging.Enter(logger, "evolve", "seed", evo.Seed, "pop", evo.Population, "gens", evo.Generations)
	start := time.Now()
	rng := rand.New(rand.NewPCG(evo.Seed, evo.Seed))
	pop, err := seedPopulation(logger, evo)
	if err != nil {
		return err
	}
	progress := []int{pop[0].steps}
	gens := 0
	stop := "generation budget"
	c := &counters{}
	for gens < evo.Generations {
		if time.Since(start) >= evo.RunCap {
			stop = "wall clock cap"
			break
		}
		logging.Enter(logger, "generation", "gen", gens+1, "best", pop[0].steps)
		pop = nextGeneration(rng, pop, c, evo)
		gens++
		progress = append(progress, pop[0].steps)
		logging.Exit(logger, "generation", "gen", gens, "best", pop[0].steps)
	}
	report(pop, progress, gens, stop, time.Since(start), c, evo)
	logging.Exit(logger, "evolve", "gens", gens, "best", pop[0].steps)
	return nil
}

func seedPopulation(logger *slog.Logger, evo config.EvolveConfig) ([]individual, error) {
	logging.Enter(logger, "seed-population", "size", evo.Population)
	pop := make([]individual, 0, evo.Population)
	generated := evo.Population - 1
	for i := range generated {
		target := seedTargets[i%len(seedTargets)]
		ind, err := generatedSeed(i, target, evo)
		if err != nil {
			logging.Fail(logger, "seed-population", err, "target", target)
			return nil, fmt.Errorf("seed-population: %w", err)
		}
		logger.Debug("seeded", "i", i, "target", target, "clues", ind.clues, "steps", ind.steps)
		pop = append(pop, ind)
	}
	legacy, err := legacyIndividual(evo)
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

func generatedSeed(i, target int, evo config.EvolveConfig) (individual, error) {
	var lastErr error
	for attempt := 0; attempt < evo.SeedAttempts; attempt++ {
		res, err := puzzle.Generate(target, evo.Seed+uint64(i)*uint64(evo.SeedStride)+uint64(attempt), solver.CountSolutions)
		if err != nil {
			lastErr = fmt.Errorf("seed %d attempt %d: %w", i, attempt, err)
			continue
		}
		if ind, ok := evaluate(res.Grid, evo); ok {
			return ind, nil
		}
		lastErr = fmt.Errorf("seed %d attempt %d: generated grid rejected", i, attempt)
	}
	return individual{}, lastErr
}

func legacyIndividual(evo config.EvolveConfig) (individual, error) {
	g, err := sudoku.Parse81(puzzle.Sample81)
	if err != nil {
		return individual{}, fmt.Errorf("legacy sample: %w", err)
	}
	res := solver.Propagate(context.Background(), g, config.Defaults())
	if !res.Solved {
		return individual{}, fmt.Errorf("legacy sample unsolved by the engine")
	}
	for i := range g {
		if cluesOf(g) >= evo.MinClues {
			break
		}
		if g[i] == config.EmptyDigit {
			g[i] = res.Grid[i]
		}
	}
	ind, ok := evaluate(g, evo)
	if !ok {
		return individual{}, fmt.Errorf("topped-up legacy sample rejected")
	}
	return ind, nil
}

func evaluate(g sudoku.Grid, evo config.EvolveConfig) (individual, bool) {
	if n := cluesOf(g); n < evo.MinClues || n > evo.MaxClues {
		return individual{}, false
	}
	if !isUnique(g) {
		return individual{}, false
	}
	return evaluateSolve(g, evo)
}

func evaluateSolve(g sudoku.Grid, evo config.EvolveConfig) (individual, bool) {
	if n := cluesOf(g); n < evo.MinClues || n > evo.MaxClues {
		return individual{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), evo.EvalCap)
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

func nextGeneration(rng *rand.Rand, pop []individual, c *counters, evo config.EvolveConfig) []individual {
	next := make([]individual, 0, evo.Population)
	next = append(next, pop[:evo.EliteCount]...)
	for len(next) < evo.Population {
		a := tournament(rng, pop, evo.TournamentK)
		if rng.IntN(100) < evo.CrossoverPct {
			b := tournament(rng, pop, evo.TournamentK)
			next = append(next, crossoverChild(rng, a, b, c, evo))
			continue
		}
		next = append(next, mutateChild(rng, a, c, evo))
	}
	sortPopulation(next)
	return next
}

func tournament(rng *rand.Rand, pop []individual, k int) individual {
	best := pop[rng.IntN(len(pop))]
	for range k - 1 {
		challenger := pop[rng.IntN(len(pop))]
		if challenger.steps > best.steps {
			best = challenger
		}
	}
	return best
}

func mutateChild(rng *rand.Rand, parent individual, c *counters, evo config.EvolveConfig) individual {
	draw := rng.IntN(100)
	switch {
	case draw < evo.RelabelPct:
		c.relabelAttempt++
		child, ok := evaluateSolve(relabelGivens(rng, parent.grid), evo)
		if ok {
			c.relabelAccept++
			return child
		}
	case draw < evo.RelabelPct+evo.SwapPct:
		c.swapAttempt++
		child, ok := swapCandidate(rng, parent, &c.swapRepair, evo)
		if ok {
			c.swapAccept++
			return child
		}
	case draw < evo.RelabelPct+evo.SwapPct+evo.RemovePct:
		c.removeAttempt++
		child, ok := removeCandidate(rng, parent, &c.removeRepair, evo)
		if ok {
			c.removeAccept++
			return child
		}
	default:
		c.addAttempt++
		child, ok := evaluateSolve(addClue(rng, parent.grid, parent.sol, evo.MaxClues), evo)
		if ok {
			c.addAccept++
			return child
		}
	}
	return parent
}

func swapCandidate(rng *rand.Rand, parent individual, repair *int, evo config.EvolveConfig) (individual, bool) {
	candidate := swapWithinBand(rng, parent.grid)
	if candidate == parent.grid {
		return parent, false
	}
	if isUnique(candidate) {
		return evaluateSolve(candidate, evo)
	}
	*repair++
	return repairedChild(rng, candidate, parent.clues, evo)
}

func removeCandidate(rng *rand.Rand, parent individual, repair *int, evo config.EvolveConfig) (individual, bool) {
	candidate := removeClue(rng, parent.grid, evo.MinClues)
	if candidate == parent.grid {
		return parent, false
	}
	if isUnique(candidate) {
		return evaluateSolve(candidate, evo)
	}
	*repair++
	return repairedChild(rng, candidate, parent.clues, evo)
}

func repairedChild(rng *rand.Rand, ruined sudoku.Grid, targetClues int, evo config.EvolveConfig) (individual, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), evo.EvalCap)
	defer cancel()
	res := solver.Propagate(ctx, ruined, config.Defaults())
	if !res.Solved {
		return individual{}, false
	}
	out := ruined
	for !isUnique(out) {
		empties := emptyCells(out)
		if len(empties) == 0 || cluesOf(out) >= evo.MaxClues {
			return individual{}, false
		}
		at := empties[rng.IntN(len(empties))]
		out[at] = res.Grid[at]
	}
	for range evo.RedigAttempts {
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
	return evaluateSolve(out, evo)
}

func crossoverChild(rng *rand.Rand, a, b individual, c *counters, evo config.EvolveConfig) individual {
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
	for cluesOf(child) > evo.MaxClues {
		givens := givenCells(child)
		child[givens[rng.IntN(len(givens))]] = config.EmptyDigit
	}
	if child == lead.grid {
		return lead
	}
	if child, ok := evaluate(child, evo); ok {
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

func removeClue(rng *rand.Rand, g sudoku.Grid, minClues int) sudoku.Grid {
	if cluesOf(g) <= minClues {
		return g
	}
	givens := givenCells(g)
	out := g
	out[givens[rng.IntN(len(givens))]] = config.EmptyDigit
	return out
}

func addClue(rng *rand.Rand, g, sol sudoku.Grid, maxClues int) sudoku.Grid {
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

func report(pop []individual, progress []int, gens int, stop string, dur time.Duration, c *counters, evo config.EvolveConfig) {
	best := pop[0]
	fmt.Printf("evolve run: seed %d population %d generations %d elites %d tournament %d\n",
		evo.Seed, evo.Population, evo.Generations, evo.EliteCount, evo.TournamentK)
	fmt.Printf("clue band %d..%d crossover %d%% mutation: relabel %d%% swap %d%% remove %d%% add %d%% redig attempts %d\n",
		evo.MinClues, evo.MaxClues, evo.CrossoverPct, evo.RelabelPct, evo.SwapPct, evo.RemovePct, 100-evo.RelabelPct-evo.SwapPct-evo.RemovePct, evo.RedigAttempts)
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
		if i == 0 || i == len(progress)-1 || i%evo.ProgressStride == 0 {
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
