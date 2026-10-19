package sudoku

import "testing"

func gridFromString(t *testing.T, s string) Grid {
	t.Helper()
	if len(s) != 81 {
		t.Fatalf("fixture length %d", len(s))
	}
	var g Grid
	for i := range 81 {
		if s[i] == '.' {
			g[i] = 0
		} else {
			g[i] = s[i] - '0'
		}
	}
	return g
}

const solvedFixture = "693784512" +
	"487512936" +
	"125963874" +
	"932651487" +
	"568247391" +
	"741398625" +
	"319475268" +
	"856129743" +
	"274836159"

func TestCellCoordinates(t *testing.T) {
	cases := []struct {
		i             int
		row, col, box int
	}{
		{0, 0, 0, 0},
		{4, 0, 4, 1},
		{8, 0, 8, 2},
		{29, 3, 2, 3},
		{40, 4, 4, 4},
		{60, 6, 6, 8},
		{80, 8, 8, 8},
	}
	for _, tc := range cases {
		if got := CellRow(tc.i); got != tc.row {
			t.Errorf("CellRow(%d): got %d want %d", tc.i, got, tc.row)
		}
		if got := CellCol(tc.i); got != tc.col {
			t.Errorf("CellCol(%d): got %d want %d", tc.i, got, tc.col)
		}
		if got := CellBox(tc.i); got != tc.box {
			t.Errorf("CellBox(%d): got %d want %d", tc.i, got, tc.box)
		}
	}
}

func TestRowUsed(t *testing.T) {
	g := gridFromString(t, "123456789"+
		"........."+
		"........."+
		"........."+
		"........."+
		"........."+
		"........."+
		"........."+
		".........")
	if got := RowUsed(g, 0); got != 0x1FF {
		t.Errorf("full row: got %#04x want 0x1ff", got)
	}
	if got := RowUsed(g, 5); got != 0 {
		t.Errorf("empty row: got %#04x want 0", got)
	}
}

func TestColAndBoxUsed(t *testing.T) {
	g := gridFromString(t, "1........"+
		"..1....6."+
		"........."+
		"........."+
		"........."+
		"........."+
		"........."+
		"........."+
		".....5...")
	if got := ColUsed(g, 0); got != 1<<0 {
		t.Errorf("col 0: got %#04x want 1", got)
	}
	if got := ColUsed(g, 2); got != 1<<0 {
		t.Errorf("col 2: got %#04x want 1", got)
	}
	if got := ColUsed(g, 7); got != 1<<5 {
		t.Errorf("col 7: got %#04x want digit 6 bit", got)
	}
	if got := BoxUsed(g, 0); got != 1<<0 {
		t.Errorf("box 0: got %#04x want 1", got)
	}
	if got := BoxUsed(g, 2); got != 1<<5 {
		t.Errorf("box 2: got %#04x want digit 6 bit", got)
	}
	if got := BoxUsed(g, 7); got != 1<<4 {
		t.Errorf("box 7: got %#04x want digit 5 bit", got)
	}
	if got := BoxUsed(g, 8); got != 0 {
		t.Errorf("empty box 8: got %#04x want 0", got)
	}
}

func TestCandidatesAndPopcount(t *testing.T) {
	g := gridFromString(t, "1234567.."+
		"9........"+
		"........."+
		"........9"+
		"........."+
		"........."+
		"........."+
		"........."+
		".........")
	got := Candidates(g, 7)
	wantTwo := uint16(1<<7 | 1<<8)
	if got != wantTwo {
		t.Errorf("cell 7 candidates: got %#04x want %#04x", got, wantTwo)
	}
	if Popcount(got) != 2 {
		t.Errorf("popcount: got %d want 2", Popcount(got))
	}
	if got := Candidates(g, 8); got != 1<<7 {
		t.Errorf("cell 8 candidates: got %#04x want digit 8 bit", got)
	}
	if Popcount(0x1FF) != 9 {
		t.Errorf("popcount full: got %d want 9", Popcount(0x1FF))
	}
}

func TestFirstSetDigit(t *testing.T) {
	if got := FirstSetDigit(1 << 7); got != 8 {
		t.Errorf("single bit: got %d want 8", got)
	}
	if got := FirstSetDigit(1<<2 | 1<<5); got != 3 {
		t.Errorf("lowest bit: got %d want 3", got)
	}
}

func TestUnitCells(t *testing.T) {
	cases := []struct {
		u    int
		want [9]int
	}{
		{0, [9]int{0, 1, 2, 3, 4, 5, 6, 7, 8}},
		{4, [9]int{36, 37, 38, 39, 40, 41, 42, 43, 44}},
		{9, [9]int{0, 9, 18, 27, 36, 45, 54, 63, 72}},
		{17, [9]int{8, 17, 26, 35, 44, 53, 62, 71, 80}},
		{18, [9]int{0, 1, 2, 9, 10, 11, 18, 19, 20}},
		{22, [9]int{30, 31, 32, 39, 40, 41, 48, 49, 50}},
		{26, [9]int{60, 61, 62, 69, 70, 71, 78, 79, 80}},
	}
	for _, tc := range cases {
		if got := UnitCells(tc.u); got != tc.want {
			t.Errorf("unit %d: got %v want %v", tc.u, got, tc.want)
		}
	}
}

func TestMRVCell(t *testing.T) {
	full := gridFromString(t, solvedFixture)
	if got := MRVCell(full); got != -1 {
		t.Errorf("full grid: got %d want -1", got)
	}
	twoEmpty := full
	twoEmpty[0] = 0
	twoEmpty[80] = 0
	if got := MRVCell(twoEmpty); got != 0 {
		t.Errorf("tie of two singletons: got %d want 0", got)
	}
	g := gridFromString(t, "1234567.."+
		"9........"+
		"........."+
		"........9"+
		"........."+
		"........."+
		"........."+
		"........."+
		".........")
	if got := MRVCell(g); got != 8 {
		t.Errorf("singleton beats earlier pair: got %d want 8", got)
	}
	g[8] = 8
	if got := MRVCell(g); got != 7 {
		t.Errorf("after fill: got %d want 7", got)
	}
	var allEmpty Grid
	if got := MRVCell(allEmpty); got != 0 {
		t.Errorf("all-empty grid: got %d want 0, every cell offers the full candidate count", got)
	}
}
