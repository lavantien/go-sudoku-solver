package solver

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestSeedGridFillsRowsWithMissingDigits(t *testing.T) {
	g, err := sudoku.Parse81(".......1." +
		"4........" +
		".2......." +
		"....5.4.7" +
		"..8...3.." +
		"..1.9...." +
		"3..4..2.." +
		".5.1....." +
		"...8.6...")
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []uint64{1, 2, 7} {
		rng := rand.New(rand.NewPCG(seed, 0))
		cur := seedGrid(g, rng)
		for r := range 9 {
			if got := sudoku.RowUsed(cur, r); got != sudoku.AllDigits {
				t.Errorf("seed %d row %d mask %04x", seed, r, got)
			}
		}
		if !sameGivens(cur, g) {
			t.Errorf("seed %d moved givens", seed)
		}
	}
}

func sameGivens(g, orig sudoku.Grid) bool {
	for i := range orig {
		if orig[i] != 0 && g[i] != orig[i] {
			return false
		}
	}
	return true
}

func TestSwapDeltaMatchesFullRecompute(t *testing.T) {
	rng := rand.New(rand.NewPCG(1234, 99))
	for range 500 {
		g := seedGrid(randomSparseGrid(rng), rng)
		free := freeByRow(g)
		row := rng.IntN(9)
		if len(free[row]) < 2 {
			continue
		}
		a := free[row][rng.IntN(len(free[row]))]
		b := free[row][rng.IntN(len(free[row]))]
		if a == b {
			continue
		}
		want := g
		want[a], want[b] = want[b], want[a]
		delta := totalCost(want) - totalCost(g)
		if got := moveDelta(g, [3]int{a, b, -1}); got != delta {
			t.Fatalf("swap %d %d: got %d want %d", a, b, got, delta)
		}
	}
}

func TestRotateDeltaMatchesFullRecompute(t *testing.T) {
	rng := rand.New(rand.NewPCG(77, 5))
	for range 500 {
		g := seedGrid(randomSparseGrid(rng), rng)
		free := freeByRow(g)
		row := rng.IntN(9)
		if len(free[row]) < 3 {
			continue
		}
		pick := func() int { return free[row][rng.IntN(len(free[row]))] }
		a, b, c := pick(), pick(), pick()
		if a == b || b == c || a == c {
			continue
		}
		want := g
		applyMove(&want, [3]int{a, b, c})
		delta := totalCost(want) - totalCost(g)
		if got := moveDelta(g, [3]int{a, b, c}); got != delta {
			t.Fatalf("rotate %d %d %d: got %d want %d", a, b, c, got, delta)
		}
	}
}

func TestAcceptUphillCapsHugeTemperatureWithoutPanic(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	for _, temp := range []int64{config.AnnealTempMax, math.MaxInt64} {
		if !acceptUphill(rng, temp, 1) {
			t.Errorf("temp %d: the capped threshold must accept a unit climb", temp)
		}
	}
}

func TestAcceptUphillMatchesUncappedForLegalTemps(t *testing.T) {
	capped := rand.New(rand.NewPCG(9, 9))
	uncapped := rand.New(rand.NewPCG(9, 9))
	for _, temp := range []int64{1, 100, 4000, 1 << 40, config.AnnealTempMax} {
		for delta := 1; delta <= 8; delta++ {
			want := uncapped.IntN(int(temp)+delta*config.AnnealScale) < int(temp)
			if got := acceptUphill(capped, temp, delta); got != want {
				t.Fatalf("temp %d delta %d: got %v want the uncapped draw %v", temp, delta, got, want)
			}
		}
	}
}

func randomSparseGrid(rng *rand.Rand) sudoku.Grid {
	var g sudoku.Grid
	for i := range 81 {
		if rng.IntN(3) == 0 {
			g[i] = uint8(1 + rng.IntN(9))
		}
	}
	return g
}
