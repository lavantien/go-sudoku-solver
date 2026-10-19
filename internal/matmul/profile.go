package matmul

import "math"

type Profile func(a, b int) float64

type Witness struct {
	A int
	B int
	H int
	P float64
	Q float64
}

type CheckResult struct {
	Symmetry  []Witness
	UnitRow   []Witness
	Concavity []Witness
	Tripling  []Witness
}

func RankProfile(a, b int) float64 {
	return float64(a + b - 1)
}

func ProductProfile(a, b int) float64 {
	return float64(a * b)
}

func GeometryProfile(a, b int) float64 {
	return math.Sqrt(float64(a * b * (a + b - 1)))
}

func CapT(a int) float64 {
	return 3 * math.Log(2*float64(a)-1) / (4 * math.Log(float64(a)))
}

func CheckProfile(p Profile, maxA, maxB int, eps float64) CheckResult {
	var res CheckResult
	for a := 1; a <= maxA; a++ {
		for b := a; b <= maxB; b++ {
			if p(a, b) != p(b, a) {
				res.Symmetry = append(res.Symmetry, Witness{A: a, B: b, P: p(a, b), Q: p(b, a)})
			}
		}
	}
	for b := 1; b <= maxB; b++ {
		if p(1, b) != float64(b) {
			res.UnitRow = append(res.UnitRow, Witness{A: 1, B: b, P: p(1, b), Q: float64(b)})
		}
	}
	for a := 2; a <= maxA; a++ {
		for b := 2; b < maxB; b++ {
			if 2*p(a, b) < p(a, b+1)+p(a, b-1)-eps {
				res.Concavity = append(res.Concavity, Witness{A: a, B: b, P: 2 * p(a, b), Q: p(a, b+1) + p(a, b-1)})
			}
		}
	}
	for a := 2; a <= maxA; a++ {
		for h := 1; 3*h+a-1 <= maxB; h++ {
			big := 3*h + a - 1
			if p(a, big) < 3*p(a, h)-eps {
				res.Tripling = append(res.Tripling, Witness{A: a, H: h, B: big, P: p(a, big), Q: 3 * p(a, h)})
			}
		}
	}
	return res
}
