package sudoku

import (
	"strconv"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func RenderGrid(g Grid) string {
	var b strings.Builder
	for r := range config.Size {
		for c := range config.Size {
			b.WriteString(strconv.Itoa(int(g[r*config.Size+c])))
			if (c+1)%config.Box == 0 && c != config.Size-1 {
				b.WriteByte(config.GridColSep)
			}
		}
		b.WriteByte('\n')
		if (r+1)%config.Box == 0 && r != config.Size-1 {
			b.WriteString(config.GridUnitRow)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
