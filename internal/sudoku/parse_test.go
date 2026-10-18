package sudoku

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

const sampleInput = ".......1.\n" +
	"4........\n" +
	".2.......\n" +
	"....5.4.7\n" +
	"..8...3..\n" +
	"..1.9....\n" +
	"3..4..2..\n" +
	".5.1.....\n" +
	"...8.6...\n"

const sampleCrLf = ".......1.\r\n" +
	"4........\r\n" +
	".2.......\r\n" +
	"....5.4.7\r\n" +
	"..8...3..\r\n" +
	"..1.9....\r\n" +
	"3..4..2..\r\n" +
	".5.1.....\r\n" +
	"...8.6...\r\n"

const sample81 = ".......1." +
	"4........" +
	".2......." +
	"....5.4.7" +
	"..8...3.." +
	"..1.9...." +
	"3..4..2.." +
	".5.1....." +
	"...8.6..."

const solution81 = "693784512" +
	"487512936" +
	"125963874" +
	"932651487" +
	"568247391" +
	"741398625" +
	"319475268" +
	"856129743" +
	"274836159"

func TestParseInputSample(t *testing.T) {
	g, err := ParseInput(strings.NewReader(sampleInput))
	if err != nil {
		t.Fatal(err)
	}
	if got := Serialize(g); got != sample81 {
		t.Errorf("serialize: got %s want %s", got, sample81)
	}
}

func TestParseInputTrailingLinesIgnored(t *testing.T) {
	r := strings.NewReader(sampleInput + "garbage\nmore\n")
	if _, err := ParseInput(r); err != nil {
		t.Errorf("trailing garbage: %v", err)
	}
}

func TestParseInputErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty input", "", config.ErrNotEnoughLines},
		{"eight lines", "1........\n2........\n3........\n4........\n5........\n6........\n7........\n8........\n", config.ErrNotEnoughLines},
		{"short line", "12345678\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrLineLengthFmt, 8)},
		{"long line", "1234567890\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrLineLengthFmt, 10)},
		{"crlf line", "12345678\r\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrLineLengthFmt, 8)},
		{"crlf valid lines parse", sampleCrLf, ""},
		{"midline carriage return", "1234\r6789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, '\r')},
		{"zero digit", "0........\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, '0')},
		{"letter", "a........\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, 'a')},
		{"pipe", "|........\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, '|')},
		{"fullwidth digit", "１２３\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, '１')},
		{"invalid utf8", "\xc3........\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n123456789\n", fmt.Sprintf(config.ErrCharFmt, '�')},
		{"huge line", strings.Repeat("x", 70*1024) + "\n" + sampleInput, config.ErrNotEnoughLines},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseInput(strings.NewReader(tc.in))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q", tc.want)
			}
			if err.Error() != tc.want {
				t.Errorf("got %q want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestParseInputShortLineLengthReported(t *testing.T) {
	lines := []string{"..5", "123456789", "123456789", "123456789", "123456789", "123456789", "123456789", "123456789", "123456789"}
	_, err := ParseInput(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	want := fmt.Sprintf(config.ErrLineLengthFmt, 3)
	if err == nil || err.Error() != want {
		t.Errorf("got %v want %q", err, want)
	}
}

func TestParse81(t *testing.T) {
	g, err := Parse81(sample81)
	if err != nil {
		t.Fatal(err)
	}
	if Serialize(g) != sample81 {
		t.Errorf("round trip mismatch")
	}
	if _, err := Parse81(sample81[:80]); err == nil || err.Error() != fmt.Sprintf(config.ErrGridLengthFmt, 80) {
		t.Errorf("length error: got %v", err)
	}
	if _, err := Parse81(strings.Replace(sample81, ".", "x", 1)); err == nil || err.Error() != fmt.Sprintf(config.ErrCharFmt, 'x') {
		t.Errorf("char error: got %v", err)
	}
}

func TestSerializeEmptyAndFull(t *testing.T) {
	var g Grid
	if got := Serialize(g); got != strings.Repeat(".", 81) {
		t.Errorf("empty: got %s", got)
	}
	full := gridFromString(t, solution81)
	if got := Serialize(full); got != solution81 {
		t.Errorf("full: got %s", got)
	}
}
