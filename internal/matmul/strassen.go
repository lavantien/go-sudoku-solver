package matmul

func MulStrassen(a, b Dense, cutoff int) Dense {
	c, _ := MulStrassenCount(a, b, cutoff)
	return c
}

func MulStrassenCount(a, b Dense, cutoff int) (Dense, OpCount) {
	mustSame(a, b)
	if cutoff < 1 {
		cutoff = 1
	}
	p := nextPow2(a.N)
	ap := pad(a, p)
	bp := pad(b, p)
	var ops OpCount
	cp := strassenRec(ap, bp, cutoff, &ops)
	out := Dense{N: a.N, Data: make([]float64, a.N*a.N)}
	for i := range a.N {
		for j := range a.N {
			out.Data[i*a.N+j] = cp.Data[i*p+j]
		}
	}
	return out, ops
}

func strassenRec(a, b Dense, cutoff int, ops *OpCount) Dense {
	n := a.N
	if n <= cutoff {
		d, o := MulClassicalCount(a, b)
		ops.Muls += o.Muls
		ops.Adds += o.Adds
		ops.Subs += o.Subs
		return d
	}
	h := n / 2
	a11, a12, a21, a22 := quadrants(a, h)
	b11, b12, b21, b22 := quadrants(b, h)
	m1 := strassenRec(addC(a11, a22, ops), addC(b11, b22, ops), cutoff, ops)
	m2 := strassenRec(addC(a21, a22, ops), b11, cutoff, ops)
	m3 := strassenRec(a11, subC(b12, b22, ops), cutoff, ops)
	m4 := strassenRec(a22, subC(b21, b11, ops), cutoff, ops)
	m5 := strassenRec(addC(a11, a12, ops), b22, cutoff, ops)
	m6 := strassenRec(subC(a21, a11, ops), addC(b11, b12, ops), cutoff, ops)
	m7 := strassenRec(subC(a12, a22, ops), addC(b21, b22, ops), cutoff, ops)
	out := Dense{N: n, Data: make([]float64, n*n)}
	for i := range h {
		for j := range h {
			c11 := m1.Data[i*h+j] + m4.Data[i*h+j] - m5.Data[i*h+j] + m7.Data[i*h+j]
			c12 := m3.Data[i*h+j] + m5.Data[i*h+j]
			c21 := m2.Data[i*h+j] + m4.Data[i*h+j]
			c22 := (m1.Data[i*h+j] + m6.Data[i*h+j]) - (m2.Data[i*h+j] - m3.Data[i*h+j])
			out.Data[i*n+j] = c11
			out.Data[i*n+j+h] = c12
			out.Data[(i+h)*n+j] = c21
			out.Data[(i+h)*n+j+h] = c22
		}
	}
	ops.Adds += 5 * h * h
	ops.Subs += 3 * h * h
	return out
}

func addC(a, b Dense, ops *OpCount) Dense {
	ops.Adds += a.N * a.N
	return Add(a, b)
}

func subC(a, b Dense, ops *OpCount) Dense {
	ops.Subs += a.N * a.N
	return Sub(a, b)
}

func quadrants(a Dense, h int) (Dense, Dense, Dense, Dense) {
	n := a.N
	qs := [4]Dense{}
	for idx := range 4 {
		qs[idx] = Dense{N: h, Data: make([]float64, h*h)}
	}
	for i := range h {
		for j := range h {
			qs[0].Data[i*h+j] = a.Data[i*n+j]
			qs[1].Data[i*h+j] = a.Data[i*n+j+h]
			qs[2].Data[i*h+j] = a.Data[(i+h)*n+j]
			qs[3].Data[i*h+j] = a.Data[(i+h)*n+j+h]
		}
	}
	return qs[0], qs[1], qs[2], qs[3]
}

func pad(a Dense, p int) Dense {
	if a.N == p {
		return a
	}
	out := Dense{N: p, Data: make([]float64, p*p)}
	for i := range a.N {
		for j := range a.N {
			out.Data[i*p+j] = a.Data[i*a.N+j]
		}
	}
	return out
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p *= 2
	}
	return p
}
