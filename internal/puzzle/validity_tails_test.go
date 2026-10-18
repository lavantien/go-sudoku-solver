package puzzle

import (
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/sudoku"
)

func TestValidSolutionRejectsBrokenRowsAndColumns(t *testing.T) {
	solved, err := sudoku.Parse81(SampleSolution81)
	if err != nil {
		t.Fatal(err)
	}
	brokenRow := solved
	brokenRow[1] = brokenRow[0]
	if ValidSolution(brokenRow) {
		t.Error("row with a duplicate digit accepted")
	}
	brokenCol := solved
	brokenCol[0], brokenCol[1] = brokenCol[1], brokenCol[0]
	if ValidSolution(brokenCol) {
		t.Error("columns with duplicate digits accepted")
	}
}
