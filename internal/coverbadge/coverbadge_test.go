package coverbadge

import (
	"strings"
	"testing"
)

func TestPct(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		want    float64
	}{
		{"all covered", "mode: set\nfoo.go:1.1,2.2 3 1\nbar.go:5.3,6.4 2 1\n", 100.0},
		{"none covered", "mode: set\nfoo.go:1.1,2.2 3 0\n", 0.0},
		{"half covered", "mode: set\nfoo.go:1.1,2.2 4 1\nbar.go:5.3,6.4 4 0\n", 50.0},
		{"quarter covered", "mode: set\na.go:1.1,2.2 1 1\nb.go:1.1,2.2 1 0\nc.go:1.1,2.2 1 0\nd.go:1.1,2.2 1 0\n", 25.0},
		{"count mode covered when >0", "mode: count\nfoo.go:1.1,2.2 5 3\nfoo.go:3.1,3.9 5 0\n", 50.0},
		{"ignores blank and mode lines", "\nmode: set\n\nfoo.go:1.1,2.2 2 1\n", 100.0},
		{"merged profile collapses duplicate blocks", "mode: set\nfoo.go:1.1,2.2 3 1\nfoo.go:1.1,2.2 3 0\nfoo.go:1.1,2.2 3 0\n", 100.0},
		{"merged profile mixed coverage", "mode: set\nfoo.go:1.1,2.2 4 1\nfoo.go:1.1,2.2 4 0\nbar.go:5.3,6.4 4 0\nbar.go:5.3,6.4 4 0\n", 50.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Pct(tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("Pct = %.4f, want %.4f", got, tc.want)
			}
		})
	}
}

func TestPctEmptyProfileErrors(t *testing.T) {
	if _, err := Pct("mode: set\n"); err == nil {
		t.Error("empty profile should error")
	}
}

func TestPctBadNumStmtErrors(t *testing.T) {
	if _, err := Pct("mode: set\nfoo.go:1.1,2.2 x 1\n"); err == nil {
		t.Error("non-numeric numStmt should error")
	}
}

func TestPctBadCountErrors(t *testing.T) {
	if _, err := Pct("mode: set\nfoo.go:1.1,2.2 3 y\n"); err == nil {
		t.Error("non-numeric count should error")
	}
}

func TestPctSkipsMalformedLine(t *testing.T) {
	got, err := Pct("mode: set\nfoo.go:1.1,2.2\nbar.go:1.1,2.2 3 1\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != 100.0 {
		t.Errorf("Pct = %.4f, want 100 with the malformed line skipped", got)
	}
}

func TestBadgeLine(t *testing.T) {
	cases := []struct {
		pct  float64
		want string
	}{
		{100, "coverage: 100.0% of statements"},
		{25, "coverage: 25.0% of statements"},
		{0, "coverage: 0.0% of statements"},
		{200.0 / 3.0, "coverage: 66.7% of statements"},
	}
	for _, c := range cases {
		if got := BadgeLine(c.pct); got != c.want {
			t.Errorf("BadgeLine(%.4f) = %q, want %q", c.pct, got, c.want)
		}
	}
}

func TestRenderReplacesBetweenMarkers(t *testing.T) {
	in := "# p\n\n<!-- coverage:START -->\nold badge\n<!-- coverage:END -->\ntrailer\n"
	want := "# p\n\n<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\ntrailer\n"
	if got := Render(in, 50); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderCreatesSectionWhenAbsent(t *testing.T) {
	in := "# p\nno markers\n"
	want := "# p\nno markers\n\n<!-- coverage:START -->\ncoverage: 75.0% of statements\n<!-- coverage:END -->\n"
	if got := Render(in, 75); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderCreatesSectionWithoutTrailingNewline(t *testing.T) {
	in := "# p"
	want := "# p\n\n<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\n"
	if got := Render(in, 50); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderEmptyReadme(t *testing.T) {
	want := "<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\n"
	if got := Render("", 50); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderOneSidedMarkerCreatesSection(t *testing.T) {
	in := "<!-- coverage:START -->\nno end marker\n"
	want := in + "\n<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\n"
	if got := Render(in, 50); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderReversedMarkersCreatesSection(t *testing.T) {
	in := EndMarker + "\nstray badge\n" + StartMarker + "\n"
	want := in + "\n<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\n"
	if got := Render(in, 50); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderIdempotent(t *testing.T) {
	in := "# p\n\n" + StartMarker + "\nold\n" + EndMarker + "\n"
	once := Render(in, 50)
	twice := Render(once, 50)
	if once != twice {
		t.Errorf("second render changed the readme:\nfirst:  %q\nsecond: %q", once, twice)
	}
}

func TestRenderPreservesContentOutsideMarkers(t *testing.T) {
	in := "head\n" + StartMarker + "\nold\n" + EndMarker + "\ntail\n"
	got := Render(in, 100)
	if !strings.HasPrefix(got, "head\n") || !strings.HasSuffix(got, "\ntail\n") {
		t.Errorf("content outside markers changed: %q", got)
	}
}
