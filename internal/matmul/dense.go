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
	d := Dense{N: n, Data: make([]float64, n*n)}
	for i := range d.Data {
		d.Data[i] = float64(rng.IntN(hi-lo+1) + lo)
	}
	return d
}

func mustSame(a, b Dense) {
	if a.N != b.N || len(a.Data) != len(b.Data) || a.N*a.N != len(a.Data) {
		panic("matmul: dimension mismatch")
	}
}

func Add(a, b Dense) Dense {
	mustSame(a, b)
	out := Dense{N: a.N, Data: make([]float64, len(a.Data))}
	for i := range out.Data {
		out.Data[i] = a.Data[i] + b.Data[i]
	}
	return out
}

func Sub(a, b Dense) Dense {
	mustSame(a, b)
	out := Dense{N: a.N, Data: make([]float64, len(a.Data))}
	for i := range out.Data {
		out.Data[i] = a.Data[i] - b.Data[i]
	}
	return out
}

func MulClassical(a, b Dense) Dense {
	c, _ := MulClassicalCount(a, b)
	return c
}

func MulClassicalCount(a, b Dense) (Dense, OpCount) {
	mustSame(a, b)
	n := a.N
	out := Dense{N: n, Data: make([]float64, n*n)}
	var ops OpCount
	for i := range n {
		for j := range n {
			sum := 0.0
			for k := range n {
				sum += a.Data[i*n+k] * b.Data[k*n+j]
			}
			ops.Muls += n
			ops.Adds += n - 1
			out.Data[i*n+j] = sum
		}
	}
	return out, ops
}
