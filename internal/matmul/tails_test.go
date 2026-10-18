package matmul

import "testing"

func TestBooleanMulRejectsDimensionMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("mismatched dimensions must panic")
		}
	}()
	a := NewBitMatrix(2, 3)
	b := NewBitMatrix(2, 3)
	BooleanMul(a, b)
}

func TestStrassenCountCutoffFloorIsOne(t *testing.T) {
	a := Dense{N: 2, Data: []float64{1, 2, 3, 4}}
	b := Dense{N: 2, Data: []float64{5, 6, 7, 8}}
	got, _ := MulStrassenCount(a, b, 0)
	eqDense(t, got, Dense{N: 2, Data: []float64{19, 22, 43, 50}})
}
