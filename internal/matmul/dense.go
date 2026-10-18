package matmul

import "math/rand/v2"

type Dense struct {
	N    int
	Data []float64
}

type OpCount struct {
	Muls int
	Adds int
	Subs int
}

func RandomDense(n int, rng *rand.Rand, lo, hi int) Dense {
	return Dense{}
}

func Add(a, b Dense) Dense {
	return Dense{}
}

func Sub(a, b Dense) Dense {
	return Dense{}
}

func MulClassical(a, b Dense) Dense {
	return Dense{}
}

func MulClassicalCount(a, b Dense) (Dense, OpCount) {
	return Dense{}, OpCount{}
}
