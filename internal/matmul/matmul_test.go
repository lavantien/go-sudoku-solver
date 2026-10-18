package matmul

import (
	"math"
	"math/rand/v2"
	"testing"
)

func randDense(t *testing.T, n int, seed uint64) Dense {
	t.Helper()
	rng := rand.New(rand.NewPCG(seed, seed))
	return RandomDense(n, rng, -9, 9)
}

func eqDense(t *testing.T, got, want Dense) {
	t.Helper()
	if got.N != want.N {
		t.Fatalf("shape: got n=%d want n=%d", got.N, want.N)
	}
	for i := range want.Data {
		if got.Data[i] != want.Data[i] {
			t.Fatalf("cell %d: got %v want %v", i, got.Data[i], want.Data[i])
		}
	}
}

func TestMulClassicalSmall(t *testing.T) {
	a := Dense{N: 2, Data: []float64{1, 2, 3, 4}}
	b := Dense{N: 2, Data: []float64{5, 6, 7, 8}}
	eqDense(t, MulClassical(a, b), Dense{N: 2, Data: []float64{19, 22, 43, 50}})
	a3 := Dense{N: 3, Data: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	b3 := Dense{N: 3, Data: []float64{9, 8, 7, 6, 5, 4, 3, 2, 1}}
	eqDense(t, MulClassical(a3, b3), Dense{N: 3, Data: []float64{30, 24, 18, 84, 69, 54, 138, 114, 90}})
}

func TestMulClassicalCountFormula(t *testing.T) {
	for _, n := range []int{1, 2, 8, 64} {
		a := randDense(t, n, uint64(n))
		b := randDense(t, n, uint64(n)+1)
		_, ops := MulClassicalCount(a, b)
		if ops.Muls != n*n*n || ops.Adds != n*n*(n-1) || ops.Subs != 0 {
			t.Errorf("n=%d classical ops: got %+v", n, ops)
		}
	}
}

func TestStrassenEqualsClassical(t *testing.T) {
	for n := 1; n <= 65; n++ {
		a := randDense(t, n, 101)
		b := randDense(t, n, 202)
		want := MulClassical(a, b)
		for _, cutoff := range []int{1, 2, 4, 8, 16, 32, 64} {
			if cutoff > n {
				continue
			}
			got := MulStrassen(a, b, cutoff)
			if got.N != n || len(got.Data) != n*n {
				t.Fatalf("n=%d cutoff=%d shape: got n=%d len=%d", n, cutoff, got.N, len(got.Data))
			}
			for i := range want.Data {
				if got.Data[i] != want.Data[i] {
					t.Fatalf("n=%d cutoff=%d mismatch at %d: got %v want %v", n, cutoff, i, got.Data[i], want.Data[i])
				}
			}
		}
	}
}

func TestStrassenCountMatchesOracle(t *testing.T) {
	for _, n := range []int{8, 16, 32, 64} {
		for _, cutoff := range []int{1, 2, 4, 8, 16, 32} {
			if cutoff > n {
				continue
			}
			a := randDense(t, n, 7)
			b := randDense(t, n, 11)
			_, ops := MulStrassenCount(a, b, cutoff)
			want, ok := OracleStrassenOps(n, cutoff)
			if !ok {
				t.Fatalf("n=%d cutoff=%d oracle rejected pow2 input", n, cutoff)
			}
			if ops != want {
				t.Errorf("n=%d cutoff=%d ops: got %+v want %+v", n, cutoff, ops, want)
			}
		}
	}
}

func TestStrassenPinnedChapterFixture(t *testing.T) {
	a := randDense(t, 64, 4303)
	b := randDense(t, 64, 4304)
	_, classOps := MulClassicalCount(a, b)
	if classOps.Muls != 262144 || classOps.Adds != 258048 {
		t.Errorf("classical 64: got %+v", classOps)
	}
	_, ops := MulStrassenCount(a, b, 8)
	if ops.Muls != 175616 || ops.Adds != 219136 || ops.Subs != 41664 {
		t.Errorf("strassen 64 cutoff 8: got %+v", ops)
	}
	want := MulClassical(a, b)
	got := MulStrassen(a, b, 8)
	for i := range want.Data {
		if got.Data[i] != want.Data[i] {
			t.Fatalf("value mismatch at %d", i)
		}
	}
}

func TestStrassenCutoffAtSizeIsClassical(t *testing.T) {
	a := randDense(t, 16, 5)
	b := randDense(t, 16, 6)
	_, ops := MulStrassenCount(a, b, 16)
	if ops.Muls != 4096 || ops.Adds != 16*16*15 || ops.Subs != 0 {
		t.Errorf("cutoff at n: got %+v", ops)
	}
}

func TestRandomDenseBoundsAndDeterminism(t *testing.T) {
	r1 := rand.New(rand.NewPCG(9, 9))
	r2 := rand.New(rand.NewPCG(9, 9))
	d1 := RandomDense(10, r1, -9, 9)
	d2 := RandomDense(10, r2, -9, 9)
	for i := range d1.Data {
		if d1.Data[i] != d2.Data[i] {
			t.Fatalf("same seed differs at %d", i)
		}
	}
	for _, v := range d1.Data {
		if v < -9 || v > 9 || v != math.Trunc(v) {
			t.Errorf("value out of bound: %v", v)
		}
	}
}

func TestAddSub(t *testing.T) {
	a := Dense{N: 2, Data: []float64{1, 2, 3, 4}}
	b := Dense{N: 2, Data: []float64{10, 20, 30, 40}}
	if got := Add(a, b); got.Data[0] != 11 || got.Data[3] != 44 {
		t.Errorf("add: %v", got.Data)
	}
	if got := Sub(a, b); got.Data[0] != -9 || got.Data[3] != -36 {
		t.Errorf("sub: %v", got.Data)
	}
	defer func() {
		if recover() == nil {
			t.Errorf("mismatched dims did not panic")
		}
	}()
	Add(a, Dense{N: 3})
}

func TestBooleanMul(t *testing.T) {
	a := NewBitMatrix(2, 2)
	a.Set(0, 0)
	a.Set(1, 0)
	a.Set(1, 1)
	b := NewBitMatrix(2, 2)
	b.Set(0, 0)
	b.Set(0, 1)
	b.Set(1, 1)
	got := BooleanMul(a, b)
	want := [4]bool{true, true, true, true}
	for i, w := range want {
		if got.Get(i/2, i%2) != w {
			t.Errorf("cell %d: got %v want %v", i, got.Get(i/2, i%2), w)
		}
	}
	id := NewBitMatrix(3, 3)
	for i := range 3 {
		id.Set(i, i)
	}
	m := NewBitMatrix(3, 3)
	m.Set(0, 2)
	m.Set(1, 0)
	m.Set(2, 1)
	prod := BooleanMul(m, id)
	if !prod.Get(0, 2) || !prod.Get(1, 0) || !prod.Get(2, 1) {
		t.Errorf("identity product wrong")
	}
	if prod.Get(0, 0) || prod.Get(1, 1) || prod.Get(2, 2) {
		t.Errorf("identity product has extra bits")
	}
	zero := NewBitMatrix(2, 2)
	if prod2 := BooleanMul(zero, a); prod2.Get(0, 0) || prod2.Get(1, 1) {
		t.Errorf("zero product not zero")
	}
}

func TestOracleRejectsNonPow2(t *testing.T) {
	if _, ok := OracleStrassenOps(12, 4); ok {
		t.Errorf("n=12 accepted")
	}
	if _, ok := OracleStrassenOps(16, 3); ok {
		t.Errorf("cutoff=3 accepted")
	}
	if _, ok := OracleStrassenOps(4, 8); ok {
		t.Errorf("cutoff above n accepted")
	}
	if _, ok := OracleStrassenOps(64, 8); !ok {
		t.Errorf("valid input rejected")
	}
}
