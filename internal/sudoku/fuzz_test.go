package sudoku

import (
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func FuzzParseInput(f *testing.F) {
	f.Add([]byte(sampleInput))
	f.Add([]byte(sampleInput + "junk\n"))
	f.Add([]byte("12345678\r\n123456789\n"))
	f.Add([]byte("１２３456789\n"))
	f.Add([]byte(strings.Repeat("x", 70*1024) + "\n"))
	f.Add([]byte("\xc3\x28......\n123456789\n"))
	f.Add([]byte(""))
	f.Add([]byte("9........\n8........\n7........\n6........\n5........\n4........\n3........\n2........\n1........\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		g, err := ParseInput(strings.NewReader(string(data)))
		if err != nil {
			msg := err.Error()
			if msg != config.ErrNotEnoughLines &&
				!strings.HasPrefix(msg, "invalid line length:") &&
				!strings.HasPrefix(msg, "invalid character:") {
				t.Errorf("error outside vocabulary: %q", msg)
			}
			return
		}
		if got := Serialize(g); got != firstNineLines(string(data)) {
			t.Errorf("serialize mismatch: got %s want %s", got, firstNineLines(string(data)))
		}
	})
}

func firstNineLines(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) < 9 {
		return ""
	}
	joined := lines[0] + lines[1] + lines[2] + lines[3] + lines[4] + lines[5] + lines[6] + lines[7] + lines[8]
	var b strings.Builder
	for _, ch := range joined {
		if (ch < '1' || ch > '9') && ch != '.' {
			return ""
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func TestParse81RoundTripProperty(t *testing.T) {
	g, err := Parse81(sample81)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse81(Serialize(g))
	if err != nil {
		t.Fatal(err)
	}
	if again != g {
		t.Errorf("round trip changed grid")
	}
}
