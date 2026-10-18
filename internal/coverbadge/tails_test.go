package coverbadge

import (
	"strings"
	"testing"
)

func TestPctRejectsOversizedLine(t *testing.T) {
	huge := "mode: set\n" + strings.Repeat("x", 2<<20) + " 1 1\n"
	if _, err := Pct(huge); err == nil {
		t.Error("oversized profile line must error")
	}
}

func TestRenderAppendCases(t *testing.T) {
	cases := []struct {
		name   string
		readme string
	}{
		{"blank trailer", "body\n\n"},
		{"single trailer", "body\n"},
		{"no trailer", "body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Render(tc.readme, 50)
			if !strings.Contains(got, BadgeLine(50)) {
				t.Errorf("badge missing: %q", got)
			}
			if !strings.HasPrefix(got, tc.readme) {
				t.Errorf("original content lost: %q", got)
			}
			if !strings.HasSuffix(got, EndMarker+"\n") {
				t.Errorf("section unterminated: %q", got)
			}
		})
	}
}
