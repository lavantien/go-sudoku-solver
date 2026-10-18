package puzzle

import (
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestFixturesPinLegacyReadmeSample(t *testing.T) {
	wantLines := ".......1." +
		"4........" +
		".2......." +
		"....5.4.7" +
		"..8...3.." +
		"..1.9...." +
		"3..4..2.." +
		".5.1....." +
		"...8.6..."
	if Sample81 != wantLines {
		t.Errorf("sample: got %s", Sample81)
	}
	wantSolution := "693784512" +
		"487512936" +
		"125963874" +
		"932651487" +
		"568247391" +
		"741398625" +
		"319475268" +
		"856129743" +
		"274836159"
	if SampleSolution81 != wantSolution {
		t.Errorf("solution: got %s", SampleSolution81)
	}
	if Unsolvable81 != "55......."+strings.Repeat(".", 72) {
		t.Errorf("unsolvable: got %s", Unsolvable81)
	}
	if _, err := sudoku.Parse81(Sample81); err != nil {
		t.Errorf("sample does not parse: %v", err)
	}
	if _, err := sudoku.Parse81(SampleSolution81); err != nil {
		t.Errorf("solution does not parse: %v", err)
	}
}

func TestValidSolution(t *testing.T) {
	solved, err := sudoku.Parse81(SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	if !ValidSolution(solved) {
		t.Errorf("reference solution rejected")
	}
	broken := solved
	broken[0] = 5
	if ValidSolution(broken) {
		t.Errorf("mutated solution accepted")
	}
	partial := solved
	partial[0] = 0
	if ValidSolution(partial) {
		t.Errorf("partial grid accepted")
	}
}

func TestGivensPreserved(t *testing.T) {
	orig, err := sudoku.Parse81(Sample81)
	if err != nil {
		t.Fatal(err)
	}
	solved, err := sudoku.Parse81(SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	if !GivensPreserved(solved, orig) {
		t.Errorf("reference solution loses givens")
	}
	broken := solved
	for i := range orig {
		if orig[i] != 0 {
			broken[i] = orig[i]%9 + 1
			break
		}
	}
	if GivensPreserved(broken, orig) {
		t.Errorf("mutated grid preserves givens")
	}
}
