package main

import (
	"io"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/solver"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	return <-done
}

func tinyEvolveCfg() config.Config {
	cfg := config.Defaults()
	cfg.Evolve.Seed = 20261019
	cfg.Evolve.Population = 6
	cfg.Evolve.Generations = 3
	return cfg
}

func defaultEvo() config.EvolveConfig {
	return config.Defaults().Evolve
}

var durationRe = regexp.MustCompile(`(stopped by .+ after [0-9]+ generations in) .*`)

func TestEvolveTinyBudgetRunsDeterministically(t *testing.T) {
	cfg := tinyEvolveCfg()
	logger := logging.New("")
	first := captureStdout(t, func() {
		if err := run(logger, cfg); err != nil {
			t.Fatalf("first evolve run: %v", err)
		}
	})
	second := captureStdout(t, func() {
		if err := run(logger, cfg); err != nil {
			t.Fatalf("second evolve run: %v", err)
		}
	})
	normalize := func(s string) string {
		return durationRe.ReplaceAllString(s, "$1 DURATION")
	}
	if normalize(first) != normalize(second) {
		t.Errorf("rerun diverged:\nfirst:  %q\nsecond: %q", normalize(first), normalize(second))
	}
	for _, want := range []string{
		"stopped by generation budget after 3 generations",
		"evolve run: seed 20261019 population 6 generations 3",
		"operators accepted/attempted:",
		"best steps by generation:",
		"final population:",
		"winner: clues ",
		"puzzle81 ",
		"solution81 ",
	} {
		if !strings.Contains(first, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	for _, ln := range strings.Split(first, "\n") {
		if line, ok := strings.CutPrefix(ln, "puzzle81 "); ok {
			if len(line) != config.Cells {
				t.Errorf("puzzle81 length: got %d want %d", len(line), config.Cells)
			}
			if _, err := sudoku.Parse81(line); err != nil {
				t.Errorf("puzzle81 does not re-parse: %v", err)
			}
		}
	}
}

func TestEvolveZeroGenerationsReportsSeedPopulation(t *testing.T) {
	cfg := tinyEvolveCfg()
	cfg.Evolve.Generations = 0
	out := captureStdout(t, func() {
		if err := run(logging.New(""), cfg); err != nil {
			t.Fatalf("zero-generation run: %v", err)
		}
	})
	if !strings.Contains(out, "after 0 generations") {
		t.Errorf("report lacks the zero-generation stop: %q", out[:min(200, len(out))])
	}
}

func sampleIndividual(t *testing.T) individual {
	t.Helper()
	res, err := puzzle.Generate(30, 1, solver.CountSolutions)
	if err != nil {
		t.Fatal(err)
	}
	ind, ok := evaluate(res.Grid, defaultEvo())
	if !ok {
		t.Fatalf("the generated puzzle must pass evaluation: %d clues", cluesOf(res.Grid))
	}
	return ind
}

func TestEvaluateEnforcesTheClueBand(t *testing.T) {
	evo := defaultEvo()
	solved, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := evaluate(solved, evo); ok {
		t.Error("full grid outside the band must be rejected")
	}
	if _, ok := evaluate(sudoku.Grid{}, evo); ok {
		t.Error("empty grid outside the band must be rejected")
	}
	if _, ok := evaluate(sampleIndividual(t).grid, evo); !ok {
		t.Error("an in-band unique puzzle must be accepted")
	}
}

func TestClueHelpers(t *testing.T) {
	g := sampleIndividual(t).grid
	if cluesOf(g) != len(givenCells(g)) {
		t.Errorf("clues %d vs given cells %d", cluesOf(g), len(givenCells(g)))
	}
	if got := len(givenCells(g)) + len(emptyCells(g)); got != config.Cells {
		t.Errorf("given plus empty cells: got %d want %d", got, config.Cells)
	}
}

func TestSortPopulationDescendingBySteps(t *testing.T) {
	pop := []individual{{steps: 3}, {steps: 9}, {steps: 5}}
	sortPopulation(pop)
	if pop[0].steps != 9 || pop[1].steps != 5 || pop[2].steps != 3 {
		t.Errorf("sort order: got %d %d %d", pop[0].steps, pop[1].steps, pop[2].steps)
	}
}

func TestRelabelGivensKeepsCluesAndBijection(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 5))
	g := sampleIndividual(t).grid
	out := relabelGivens(rng, g)
	if cluesOf(out) != cluesOf(g) {
		t.Errorf("clue count changed: %d want %d", cluesOf(out), cluesOf(g))
	}
	mapping := map[uint8]uint8{}
	for i := range g {
		if g[i] == config.EmptyDigit {
			continue
		}
		if prev, ok := mapping[g[i]]; ok && prev != out[i] {
			t.Fatalf("digit %d relabeled to both %d and %d", g[i], prev, out[i])
		}
		mapping[g[i]] = out[i]
	}
	if len(mapping) != 9 {
		t.Errorf("relabel mapping covers %d digits want 9", len(mapping))
	}
}

func TestSwapWithinBandPreservesClueCount(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 9))
	g := sampleIndividual(t).grid
	for range 20 {
		out := swapWithinBand(rng, g)
		if cluesOf(out) != cluesOf(g) {
			t.Fatalf("swap changed the clue count: %d want %d", cluesOf(out), cluesOf(g))
		}
		if out != g {
			return
		}
	}
	t.Fatal("20 draws never moved a given")
}

func TestPlaceableExceptDetectsConflicts(t *testing.T) {
	g := sampleIndividual(t).grid
	at := -1
	for i := range g {
		if g[i] != config.EmptyDigit {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("no given cell to probe")
	}
	d := g[at]
	if !placeableExcept(g, at, -1, d) {
		t.Error("digit with no peer in its units must be placeable")
	}
	sameRow := -1
	for cand := range g {
		if cand != at && sudoku.CellRow(cand) == sudoku.CellRow(at) {
			sameRow = cand
			break
		}
	}
	dup := g
	dup[sameRow] = d
	if placeableExcept(dup, at, -1, d) {
		t.Error("duplicate digit in the same row must block placement")
	}
}

func TestRemoveAndAddClueGuards(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 3))
	evo := defaultEvo()
	g := sampleIndividual(t).grid
	if out := removeClue(rng, g, evo.MinClues); cluesOf(out) != cluesOf(g)-1 {
		t.Errorf("remove: got %d clues want %d", cluesOf(out), cluesOf(g)-1)
	}
	solved, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	if out := removeClue(rng, solved, evo.MinClues); cluesOf(out) < cluesOf(solved)-1 {
		t.Errorf("remove below the floor dropped more than one clue")
	}
	if out := addClue(rng, solved, solved, evo.MaxClues); out != solved {
		t.Error("add at max clues must be a no-op")
	}
}

func TestEvolveCliExitCodes(t *testing.T) {
	cfg := tinyEvolveCfg()
	if code := cli(cfg); code != 0 {
		t.Errorf("cli on a tiny budget: got exit %d want 0", code)
	}
}

func TestCrossoverChildWithIdenticalParentsReturnsParent(t *testing.T) {
	rng := rand.New(rand.NewPCG(2, 2))
	parent := sampleIndividual(t)
	c := &counters{}
	if got := crossoverChild(rng, parent, parent, c, defaultEvo()); got.grid != parent.grid {
		t.Error("identical parents must reproduce the parent grid")
	}
	if c.crossoverAttempt != 1 {
		t.Errorf("attempt counter: got %d want 1", c.crossoverAttempt)
	}
}

func TestCrossoverChildTrimsAndAcceptsSameSolutionUnion(t *testing.T) {
	solved, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	dense := individual{grid: solved, sol: solved, steps: 9, clues: 81}
	sparse := dense
	for i := range config.Cells {
		if i%2 == 0 {
			sparse.grid[i] = config.EmptyDigit
		}
	}
	sparse.clues = cluesOf(sparse.grid)
	c := &counters{}
	evo := defaultEvo()
	for seed := uint64(1); seed <= 20; seed++ {
		got := crossoverChild(rand.New(rand.NewPCG(seed, seed)), dense, sparse, c, evo)
		if got.clues < evo.MinClues || got.clues > evo.MaxClues {
			t.Fatalf("seed %d: child clues %d outside the band", seed, got.clues)
		}
		if got.grid != dense.grid && got.grid != sparse.grid {
			return
		}
	}
	if c.crossoverAccept == 0 {
		t.Error("twenty crossovers never accepted a union child")
	}
}

func TestCrossoverChildMixesDistinctSolutions(t *testing.T) {
	solved, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	other := relabelGivens(rand.New(rand.NewPCG(21, 21)), solved)
	a := individual{grid: solved, sol: solved, steps: 10}
	b := individual{grid: other, sol: other, steps: 5}
	for i := range config.Cells {
		if i%3 == 0 {
			a.grid[i] = config.EmptyDigit
		}
		if i%3 == 1 {
			b.grid[i] = config.EmptyDigit
		}
	}
	a.clues, b.clues = cluesOf(a.grid), cluesOf(b.grid)
	c := &counters{}
	evo := defaultEvo()
	for seed := uint64(1); seed <= 10; seed++ {
		got := crossoverChild(rand.New(rand.NewPCG(seed, seed)), a, b, c, evo)
		if got.grid == a.grid || got.grid == b.grid {
			continue
		}
		if got.clues < evo.MinClues || got.clues > evo.MaxClues {
			t.Fatalf("seed %d: child clues %d outside the band", seed, got.clues)
		}
	}
	if c.crossoverAttempt != 10 {
		t.Errorf("attempt counter: got %d want 10", c.crossoverAttempt)
	}
}

func TestEvaluateSolveRejectsUnsolvableInBandGrids(t *testing.T) {
	base := sampleIndividual(t).grid
	rows := map[int][]int{}
	for i := range base {
		if base[i] != config.EmptyDigit {
			rows[sudoku.CellRow(i)] = append(rows[sudoku.CellRow(i)], i)
		}
	}
	at, twin := -1, -1
	for _, cells := range rows {
		if len(cells) >= 2 {
			at, twin = cells[0], cells[1]
			break
		}
	}
	if at < 0 {
		t.Fatal("no row carries two givens")
	}
	broken := base
	broken[twin] = base[at]
	if broken == base {
		t.Fatal("the clash fixture must alter the grid")
	}
	if cluesOf(broken) != cluesOf(base) {
		t.Fatalf("clue count moved: got %d want %d", cluesOf(broken), cluesOf(base))
	}
	if _, ok := evaluateSolve(broken, defaultEvo()); ok {
		t.Error("grid with a duplicated digit in one row must fail evaluation while staying in the band")
	}
}

func TestCrossoverChildPrefersTheHarderLead(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 4))
	hard := sampleIndividual(t)
	easy := hard
	easy.steps = hard.steps - 1
	c := &counters{}
	got := crossoverChild(rng, easy, hard, c, defaultEvo())
	if got.grid != hard.grid {
		t.Errorf("crossover of identical grids must return the harder lead")
	}
}

func TestSwapWithinBandReturnsEmptyGridUnchanged(t *testing.T) {
	rng := rand.New(rand.NewPCG(12, 12))
	var empty sudoku.Grid
	if got := swapWithinBand(rng, empty); got != empty {
		t.Error("grid without givens must come back unchanged")
	}
}

func TestEvaluateRejectsNonUniqueInBandGrids(t *testing.T) {
	evo := defaultEvo()
	ruin := sampleIndividual(t).grid
	for cluesOf(ruin) > evo.MinClues && isUnique(ruin) {
		ruin[givenCells(ruin)[0]] = config.EmptyDigit
	}
	if cluesOf(ruin) < evo.MinClues || isUnique(ruin) {
		t.Fatalf("could not build an in-band non-unique grid: %d clues, unique %v", cluesOf(ruin), isUnique(ruin))
	}
	if _, ok := evaluate(ruin, evo); ok {
		t.Error("grid that lost uniqueness must be rejected while staying in the band")
	}
}

func TestMutateChildHitsEveryArm(t *testing.T) {
	parent := sampleIndividual(t)
	c := &counters{}
	evo := defaultEvo()
	for seed := uint64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed))
		for range 40 {
			_ = mutateChild(rng, parent, c, evo)
		}
	}
	for name, n := range map[string]int{
		"relabel": c.relabelAttempt,
		"swap":    c.swapAttempt,
		"remove":  c.removeAttempt,
		"add":     c.addAttempt,
	} {
		if n == 0 {
			t.Errorf("mutation arm %s never fired", name)
		}
	}
}

func TestSwapAndRemoveCandidatesGuardNoOpGrids(t *testing.T) {
	rng := rand.New(rand.NewPCG(6, 6))
	repairs := 0
	solved, err := sudoku.Parse81(puzzle.SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := swapCandidate(rng, individual{grid: solved, sol: solved, clues: 81}, &repairs, defaultEvo()); ok {
		t.Error("full grid swap must not produce an individual")
	}
	if _, ok := removeCandidate(rng, individual{grid: solved, sol: solved, clues: 81}, &repairs, defaultEvo()); ok {
		t.Error("full grid remove must not produce an individual")
	}
}

func TestRepairedChildIsBestEffort(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 11))
	g := sampleIndividual(t).grid
	ruined := g
	emptied := 0
	for i := range ruined {
		if emptied >= 3 {
			break
		}
		if ruined[i] != config.EmptyDigit {
			ruined[i] = config.EmptyDigit
			emptied++
		}
	}
	if isUnique(ruined) {
		t.Fatal("the ruin must lose uniqueness for this fixture")
	}
	if _, ok := repairedChild(rng, ruined, cluesOf(g), defaultEvo()); ok {
		t.Error("repair past the clue ceiling must refuse to return an individual")
	}
}

func TestNextGenerationKeepsSizeOrderAndElites(t *testing.T) {
	cfg := tinyEvolveCfg()
	logger := logging.New("")
	pop, err := seedPopulation(logger, cfg.Evolve)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(42, 42))
	next := nextGeneration(rng, pop, &counters{}, cfg.Evolve)
	if len(next) != cfg.Evolve.Population {
		t.Fatalf("size: got %d want %d", len(next), cfg.Evolve.Population)
	}
	for i := 1; i < len(next); i++ {
		if next[i-1].steps < next[i].steps {
			t.Fatalf("order broken at %d: %d after %d", i, next[i].steps, next[i-1].steps)
		}
	}
	if next[0].steps < pop[0].steps || next[1].steps < pop[1].steps {
		t.Errorf("best steps regressed: got %d,%d want at least %d,%d",
			next[0].steps, next[1].steps, pop[0].steps, pop[1].steps)
	}
}

func TestTournamentPicksFromThePopulation(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 7))
	pop := []individual{{steps: 1}, {steps: 2}, {steps: 3}, {steps: 4}}
	k := defaultEvo().TournamentK
	for range 20 {
		got := tournament(rng, pop, k)
		found := false
		for _, p := range pop {
			if p.steps == got.steps {
				found = true
			}
		}
		if !found {
			t.Fatalf("tournament returned steps %d not in the population", got.steps)
		}
	}
}
