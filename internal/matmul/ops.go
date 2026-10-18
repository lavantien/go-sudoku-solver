package matmul

func OracleStrassenOps(n, cutoff int) (OpCount, bool) {
	if !isPow2(n) || !isPow2(cutoff) || cutoff < 1 || cutoff > n {
		return OpCount{}, false
	}
	levels := ilog2(n) - ilog2(cutoff)
	c := cutoff
	p7 := 1
	for range levels {
		p7 *= 7
	}
	mul := p7 * c * c * c
	add := (p7*c*c*(3*c+8) - 11*n*n) / 3
	sub := 7 * (p7*c*c - n*n) / 3
	return OpCount{Muls: mul, Adds: add, Subs: sub}, true
}

func isPow2(n int) bool {
	return n >= 1 && n&(n-1) == 0
}

func ilog2(n int) int {
	l := 0
	for n > 1 {
		n /= 2
		l++
	}
	return l
}
