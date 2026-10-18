package sudoku

import (
	"math/bits"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

const allDigits = 0x1FF

func digitBit(d int) uint16 {
	return 1 << (d - 1)
}

func RowUsed(g Grid, r int) uint16 {
	var m uint16
	for c := range config.Size {
		if d := g[r*config.Size+c]; d != config.EmptyDigit {
			m |= digitBit(int(d))
		}
	}
	return m
}

func ColUsed(g Grid, c int) uint16 {
	var m uint16
	for r := range config.Size {
		if d := g[r*config.Size+c]; d != config.EmptyDigit {
			m |= digitBit(int(d))
		}
	}
	return m
}

func BoxUsed(g Grid, b int) uint16 {
	var m uint16
	base := (b/config.Box)*config.Box*config.Size + (b%config.Box)*config.Box
	for dr := range config.Box {
		for dc := range config.Box {
			if d := g[base+dr*config.Size+dc]; d != config.EmptyDigit {
				m |= digitBit(int(d))
			}
		}
	}
	return m
}

func Candidates(g Grid, i int) uint16 {
	used := RowUsed(g, CellRow(i)) | ColUsed(g, CellCol(i)) | BoxUsed(g, CellBox(i))
	return ^used & allDigits
}

func Popcount(m uint16) int {
	return bits.OnesCount16(m)
}

func FirstSetDigit(m uint16) int {
	return bits.TrailingZeros16(m) + 1
}

func MRVCell(g Grid) int {
	best := -1
	bestCount := config.Size + 1
	for i := range config.Cells {
		if g[i] != config.EmptyDigit {
			continue
		}
		n := Popcount(Candidates(g, i))
		if n < bestCount {
			bestCount = n
			best = i
			if n <= 1 {
				break
			}
		}
	}
	return best
}
