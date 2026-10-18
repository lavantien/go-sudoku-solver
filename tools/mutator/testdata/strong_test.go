//go:build mutatorfixture

package subject

import "testing"

func TestInRangeBounds(t *testing.T) {
	cases := []struct {
		lo, hi, v int
		want      bool
	}{
		{1, 10, 5, true},
		{1, 10, 1, true},
		{1, 10, 10, true},
		{1, 10, 0, false},
		{1, 10, 11, false},
	}
	for _, c := range cases {
		if got := InRange(c.lo, c.hi, c.v); got != c.want {
			t.Fatalf("InRange(%d, %d, %d) = %v, want %v", c.lo, c.hi, c.v, got, c.want)
		}
	}
}
