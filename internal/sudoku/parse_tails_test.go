package sudoku

import (
	"errors"
	"strings"
	"testing"
)

type errOnceReader struct {
	data string
	err  error
}

func (r *errOnceReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

func TestParseInputReportsReaderErrors(t *testing.T) {
	grid := strings.Repeat(".........\n", 9)
	boom := errors.New("medium exploded")
	if _, err := ParseInput(&errOnceReader{data: grid, err: boom}); !errors.Is(err, boom) {
		t.Errorf("err: got %v want the reader error", err)
	}
}
