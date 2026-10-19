package solver

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

const multiSolutionGrid81 = "183564792" +
	"497283651" +
	"625719843" +
	"269351487" +
	"548672319" +
	"731498526" +
	"316945278" +
	"852137964" +
	"974826135"

func TestHalvingsCountsShiftsToFloor(t *testing.T) {
	cases := []struct {
		t0, tEnd int64
		want     int
	}{
		{4000, 100, 7},
		{128, 127, 2},
		{1, 1, 1},
		{1 << 62, 1, 62},
	}
	for _, tc := range cases {
		if got := halvings(tc.t0, tc.tEnd); got != tc.want {
			t.Errorf("halvings(%d, %d): got %d want %d", tc.t0, tc.tEnd, got, tc.want)
		}
	}
}

func TestTempAtStartsHighFloorsLow(t *testing.T) {
	cases := []struct {
		t0, tEnd int64
		clock    int
		span     int
		want     int64
	}{
		{4000, 100, 0, 100, 4000},
		{4000, 100, 300, 100, 500},
		{4000, 100, 500, 100, 125},
		{4000, 100, 1000, 100, 100},
		{100, 100, 0, 10, 100},
	}
	for _, tc := range cases {
		if got := tempAt(tc.t0, tc.tEnd, tc.clock, tc.span); got != tc.want {
			t.Errorf("tempAt(%d, %d, %d, %d): got %d want %d", tc.t0, tc.tEnd, tc.clock, tc.span, got, tc.want)
		}
	}
}

func TestAnnealGreedyRunBeatsTheSeedGrid(t *testing.T) {
	cfg := config.Defaults()
	cfg.Anneal.Restarts = 1
	cfg.Anneal.Iterations = 200
	cfg.Anneal.TempStart = 0
	cfg.Anneal.TempEnd = 0
	g := annealFixtureGrid(t)
	seedRng := rand.New(rand.NewPCG(cfg.Anneal.Seed, 0))
	seeded := seedGrid(g, seedRng)
	got := Anneal(context.Background(), g, cfg)
	if got.Solved {
		t.Fatalf("greedy 200-step budget solved the fixture, budget semantics broken")
	}
	if totalCost(got.Grid) >= totalCost(seeded) {
		t.Fatalf("greedy run returned cost %d, want strictly below the seed cost %d", totalCost(got.Grid), totalCost(seeded))
	}
}

func TestPickMoveRotatesThirdOfTheTimeWithDistinctCells(t *testing.T) {
	cur := solutionGrid(t)
	cur[0] = 1
	cur[2] = 1
	var free [config.Size][]int
	free[0] = []int{0, 2, 4}
	rng := rand.New(rand.NewPCG(20261019, 1))
	successes, rotations := 0, 0
	for range 200 {
		move, ok := pickMove(cur, free, rng)
		if !ok {
			continue
		}
		successes++
		if move[0] == move[1] {
			t.Fatalf("swap target equals the conflicted cell: %v", move)
		}
		if move[2] >= 0 {
			rotations++
			if move[0] == move[2] || move[1] == move[2] {
				t.Fatalf("rotation repeats a cell: %v", move)
			}
		}
	}
	if successes < 100 {
		t.Fatalf("qualifying moves: got %d want at least 100 of 200 draws", successes)
	}
	fraction := float64(rotations) / float64(successes)
	if fraction < 0.05 || fraction > 0.35 {
		t.Fatalf("rotation fraction: got %.3f want within 0.05 to 0.35 around the 1/5 coin", fraction)
	}
}

func TestCountSolutionsCapsAtTheLimit(t *testing.T) {
	g, err := sudoku.Parse81(multiSolutionGrid81)
	if err != nil {
		t.Fatal(err)
	}
	if !puzzle.ValidSolution(g) {
		t.Fatal("multi-solution fixture is not a valid full grid")
	}
	blocks := [2][4]int{{18, 19, 27, 28}, {24, 25, 33, 34}}
	for _, b := range blocks {
		if g[b[0]] != g[b[3]] || g[b[1]] != g[b[2]] || g[b[0]] == g[b[1]] {
			t.Fatalf("block %v does not carry the deadly cross pattern %d %d %d %d", b, g[b[0]], g[b[1]], g[b[2]], g[b[3]])
		}
	}
	h := g
	for _, b := range blocks {
		for _, i := range b {
			h[i] = config.EmptyDigit
		}
	}
	oracle := 0
	for _, flips := range [4][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} {
		hh := h
		for k, b := range blocks {
			if flips[k] == 0 {
				hh[b[0]], hh[b[1]], hh[b[2]], hh[b[3]] = g[b[0]], g[b[1]], g[b[2]], g[b[3]]
			} else {
				hh[b[0]], hh[b[1]], hh[b[2]], hh[b[3]] = g[b[1]], g[b[0]], g[b[3]], g[b[2]]
			}
		}
		if Propagate(context.Background(), hh, config.Defaults()).Solved {
			oracle++
		}
	}
	if oracle != 4 {
		t.Fatalf("orientation oracle: got %d solvable fillings, want all 4 deadly combinations", oracle)
	}
	if got := CountSolutions(h, 1000); got != 4 {
		t.Fatalf("CountSolutions uncapped: got %d want 4", got)
	}
	for _, limit := range []int{1, 2, 3} {
		if got := CountSolutions(h, limit); got != limit {
			t.Fatalf("CountSolutions capped at %d: got %d want %d", limit, got, limit)
		}
	}
}
