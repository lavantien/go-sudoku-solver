package policy

import (
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func TestGeneratedTableMatchesFeatureWidth(t *testing.T) {
	fixed := FixedOrder()
	if len(TableWeights) != len(fixed) {
		t.Fatalf("generated table has %d rows want %d", len(TableWeights), len(fixed))
	}
	width := len(Features(sampleForFeatures(t)))
	for i, row := range TableWeights {
		if len(row) != width {
			t.Fatalf("row %d has %d weights want %d", i, len(row), width)
		}
	}
}

func TestOrderWithGeneratedTableReturnsPermutation(t *testing.T) {
	got := Order(Features(sampleForFeatures(t)))
	fixed := FixedOrder()
	if len(got) != len(fixed) {
		t.Fatalf("order length %d want %d", len(got), len(fixed))
	}
	seen := make(map[string]bool, len(fixed))
	for _, id := range got {
		if seen[id] {
			t.Errorf("duplicate method %s", id)
		}
		seen[id] = true
		known := false
		for _, want := range config.MethodIDs[:3] {
			if id == want {
				known = true
			}
		}
		if !known {
			t.Errorf("unknown method %s", id)
		}
	}
}
