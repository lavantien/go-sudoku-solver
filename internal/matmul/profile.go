package matmul

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
	return 0
}

func ProductProfile(a, b int) float64 {
	return 0
}

func GeometryProfile(a, b int) float64 {
	return 0
}

func CapT(a int) float64 {
	return 0
}

func CheckProfile(p Profile, maxA, maxB int) CheckResult {
	return CheckResult{}
}
