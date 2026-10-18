package policy

import (
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func sampleForFeatures(t *testing.T) sudoku.Grid {
	t.Helper()
	g, err := sudoku.Parse81(puzzle.Sample81)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func oracleFill(g sudoku.Grid) int {
	work := g
	total := 0
	for {
		progress := false
		for i := range work {
			if work[i] != 0 {
				continue
			}
			m := ^(sudoku.RowUsed(work, sudoku.CellRow(i)) | sudoku.ColUsed(work, sudoku.CellCol(i)) | sudoku.BoxUsed(work, sudoku.CellBox(i))) & sudoku.AllDigits
			if m == 0 {
				return total
			}
			if sudoku.Popcount(m) == 1 {
				work[i] = uint8(sudoku.FirstSetDigit(m))
				total++
				progress = true
			}
		}
		for u := range 27 {
			unit := sudoku.UnitCells(u)
			for d := 1; d <= 9; d++ {
				bit := uint16(1) << (d - 1)
				placed := false
				count := 0
				at := -1
				for _, i := range unit {
					if work[i] != 0 {
						if work[i] == uint8(d) {
							placed = true
						}
						continue
					}
					cand := ^(sudoku.RowUsed(work, sudoku.CellRow(i)) | sudoku.ColUsed(work, sudoku.CellCol(i)) | sudoku.BoxUsed(work, sudoku.CellBox(i))) & sudoku.AllDigits
					if cand&bit != 0 {
						count++
						at = i
					}
				}
				if placed {
					continue
				}
				if count == 0 {
					return total
				}
				if count == 1 {
					work[at] = uint8(d)
					total++
					progress = true
				}
			}
		}
		if !progress {
			return total
		}
	}
}

func TestFeaturesPinSample(t *testing.T) {
	g := sampleForFeatures(t)
	got := Features(g)
	want := []float64{17, float64(oracleFill(g)), 0, 4, 51.0 / 27.0}
	if len(got) != len(want) {
		t.Fatalf("feature count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("feature %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestFixedOrder(t *testing.T) {
	got := FixedOrder()
	want := []string{config.MethodIDs[0], config.MethodIDs[1], config.MethodIDs[2]}
	if len(got) != 3 {
		t.Fatalf("fixed order length %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func TestResolveOrderFallsBackWithoutTable(t *testing.T) {
	got := ResolveOrder(sampleForFeatures(t))
	want := FixedOrder()
	if len(got) != len(want) {
		t.Fatalf("fallback length %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func TestResolveOrderPermutesRacers(t *testing.T) {
	saved := TableWeights
	defer func() { TableWeights = saved }()
	TableWeights = [][]float64{
		{0.1, 0.2, 0.3, 0.4, 0.5},
		{0.9, 0.8, 0.7, 0.6, 0.5},
		{0.5, 0.5, 0.5, 0.5, 0.5},
	}
	got := ResolveOrder(sampleForFeatures(t))
	want := []string{config.MethodIDs[0], config.MethodIDs[2], config.MethodIDs[1]}
	if len(got) != 3 {
		t.Fatalf("order length %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot %d: got %s want %s", i, got[i], want[i])
		}
	}
}
