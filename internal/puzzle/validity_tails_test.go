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
	boxBroken := solved
	var rowZero, rowThree [9]uint8
	copy(rowZero[:], boxBroken[0:9])
	copy(rowThree[:], boxBroken[27:36])
	copy(boxBroken[0:9], rowThree[:])
	copy(boxBroken[27:36], rowZero[:])
	if ValidSolution(boxBroken) {
		t.Error("boxes with mixed-band rows accepted")
	}
}
