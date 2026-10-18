package sudoku

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func ParseInput(r io.Reader) (Grid, error) {
	var g Grid
	sc := bufio.NewScanner(r)
	for i := range config.Size {
		if !sc.Scan() {
			return g, errors.New(config.ErrNotEnoughLines)
		}
		line := sc.Text()
		if len(line) != config.Size {
			return g, fmt.Errorf(config.ErrLineLengthFmt, len(line))
		}
		for j, char := range line {
			if char == '.' {
				g[i*config.Size+j] = config.EmptyDigit
			} else if char >= '1' && char <= '9' {
				g[i*config.Size+j] = uint8(char - '0')
			} else {
				return g, fmt.Errorf(config.ErrCharFmt, char)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return g, err
	}
	return g, nil
}

func Parse81(s string) (Grid, error) {
	var g Grid
	if len(s) != config.Cells {
		return g, fmt.Errorf(config.ErrGridLengthFmt, len(s))
	}
	for j, char := range s {
		if char == '.' {
			g[j] = config.EmptyDigit
		} else if char >= '1' && char <= '9' {
			g[j] = uint8(char - '0')
		} else {
			return g, fmt.Errorf(config.ErrCharFmt, char)
		}
	}
	return g, nil
}

func Serialize(g Grid) string {
	var b strings.Builder
	for i := range config.Cells {
		if g[i] == config.EmptyDigit {
			b.WriteByte('.')
		} else {
			b.WriteByte('0' + g[i])
		}
	}
	return b.String()
}
