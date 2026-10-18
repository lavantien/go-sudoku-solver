//go:build mutatorfixture

package subject

import "testing"

func TestInRangeMid(t *testing.T) {
	if !InRange(1, 10, 5) {
		t.Fatal("mid value must be in range")
	}
}
