package sudoku

import "testing"

const initialGridGolden = "000|000|010\n" +
	"400|000|000\n" +
	"020|000|000\n" +
	"___ ___ ___\n" +
	"000|050|407\n" +
	"008|000|300\n" +
	"001|090|000\n" +
	"___ ___ ___\n" +
	"300|400|200\n" +
	"050|100|000\n" +
	"000|806|000\n"

const solvedGridGolden = "693|784|512\n" +
	"487|512|936\n" +
	"125|963|874\n" +
	"___ ___ ___\n" +
	"932|651|487\n" +
	"568|247|391\n" +
	"741|398|625\n" +
	"___ ___ ___\n" +
	"319|475|268\n" +
	"856|129|743\n" +
	"274|836|159\n"

func TestRenderGridInitial(t *testing.T) {
	g := gridFromString(t, ".......1."+
		"4........"+
		".2......."+
		"....5.4.7"+
		"..8...3.."+
		"..1.9...."+
		"3..4..2.."+
		".5.1....."+
		"...8.6...")
	if got := RenderGrid(g); got != initialGridGolden {
		t.Errorf("initial render:\ngot\n%s\nwant\n%s", got, initialGridGolden)
	}
}

func TestRenderGridSolved(t *testing.T) {
	g := gridFromString(t, solution81)
	if got := RenderGrid(g); got != solvedGridGolden {
		t.Errorf("solved render:\ngot\n%s\nwant\n%s", got, solvedGridGolden)
	}
}
