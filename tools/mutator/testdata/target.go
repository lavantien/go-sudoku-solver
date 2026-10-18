//go:build mutatorfixture

package subject

func InRange(lo, hi, v int) bool {
	return lo <= v && v <= hi
}
