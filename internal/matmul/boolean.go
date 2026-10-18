package matmul

type BitMatrix struct {
	Rows  int
	Cols  int
	Words []uint64
}

func NewBitMatrix(rows, cols int) BitMatrix {
	return BitMatrix{Rows: rows, Cols: cols, Words: make([]uint64, rows*wordsPerRow(cols))}
}

func wordsPerRow(cols int) int {
	return (cols + 63) / 64
}

func (m BitMatrix) Set(r, c int) {
	m.Words[r*wordsPerRow(m.Cols)+c/64] |= 1 << (c % 64)
}

func (m BitMatrix) Get(r, c int) bool {
	return m.Words[r*wordsPerRow(m.Cols)+c/64]&(1<<(c%64)) != 0
}

func BooleanMul(a, b BitMatrix) BitMatrix {
	if a.Cols != b.Rows {
		panic("matmul: boolean dimension mismatch")
	}
	out := NewBitMatrix(a.Rows, b.Cols)
	bw := wordsPerRow(b.Cols)
	ow := wordsPerRow(out.Cols)
	for i := range a.Rows {
		for k := range a.Cols {
			if !a.Get(i, k) {
				continue
			}
			base := k * bw
			obase := i * ow
			for w := range bw {
				out.Words[obase+w] |= b.Words[base+w]
			}
		}
	}
	return out
}
