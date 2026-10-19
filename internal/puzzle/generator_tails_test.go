package puzzle

import (
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestBaseGridIsTheCanonicalShiftPattern(t *testing.T) {
	want := "123456789" +
		"456789123" +
		"789123456" +
		"234567891" +
		"567891234" +
		"891234567" +
		"345678912" +
		"678912345" +
		"912345678"
	if got := sudoku.Serialize(baseGrid()); got != want {
		t.Fatalf("base grid: got %s want %s", got, want)
	}
}

func TestShuffledFullGridPinnedPerSeed(t *testing.T) {
	cases := []struct {
		seed uint64
		file string
	}{
		{11, "golden-shuffle-seed-11.txt"},
		{12, "golden-shuffle-seed-12.txt"},
	}
	for _, tc := range cases {
		raw, err := os.ReadFile("testdata/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(string(raw))
		got := sudoku.Serialize(shuffledFullGrid(rand.New(rand.NewPCG(tc.seed, tc.seed))))
		if got != want {
			t.Fatalf("seed %d: got %s want %s", tc.seed, got, want)
		}
	}
}
