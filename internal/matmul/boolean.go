package matmul

type BitMatrix struct {
	Rows  int
	Cols  int
	Words []uint64
}

func NewBitMatrix(rows, cols int) BitMatrix {
	return BitMatrix{}
}

func (m BitMatrix) Set(r, c int) {
}

func (m BitMatrix) Get(r, c int) bool {
	return false
}

func BooleanMul(a, b BitMatrix) BitMatrix {
	return BitMatrix{}
}
