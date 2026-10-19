package matmul

import (
	"math"
	"testing"
)

func TestRankProfileFailsOnlyTripling(t *testing.T) {
	res := CheckProfile(RankProfile, 12, 40, 1e-9)
	if len(res.Symmetry) != 0 || len(res.UnitRow) != 0 || len(res.Concavity) != 0 {
		t.Errorf("rank profile: unexpected failures %+v", res)
	}
	if len(res.Tripling) == 0 {
		t.Fatalf("rank profile: expected a tripling failure")
	}
	w := res.Tripling[0]
	if w.A != 2 || w.H != 1 || w.P != 5 || w.Q != 6 {
		t.Errorf("tripling witness: got (a=%d h=%d P=%v 3P=%v) want (2,1,5,6)", w.A, w.H, w.P, w.Q)
	}
}

func TestProductProfilePassesAll(t *testing.T) {
	res := CheckProfile(ProductProfile, 12, 40, 1e-9)
	if len(res.Symmetry) != 0 || len(res.UnitRow) != 0 || len(res.Concavity) != 0 || len(res.Tripling) != 0 {
		t.Errorf("product profile: unexpected failures %+v", res)
	}
}

func TestGeometryProfilePassesAll(t *testing.T) {
	res := CheckProfile(GeometryProfile, 12, 40, 1e-9)
	if len(res.Symmetry) != 0 || len(res.UnitRow) != 0 || len(res.Concavity) != 0 || len(res.Tripling) != 0 {
		t.Errorf("geometry profile: unexpected failures %+v", res)
	}
	if got := GeometryProfile(2, 8); got != 12 {
		t.Errorf("P(2,8): got %v want 12", got)
	}
	for a := 1; a <= 12; a++ {
		want := float64(a) * math.Sqrt(float64(2*a-1))
		if got := GeometryProfile(a, a); math.Abs(got-want) > 1e-9 {
			t.Errorf("diagonal D_%d: got %v want %v", a, got, want)
		}
	}
}

func TestBrokenProfilesFailExpectedLaws(t *testing.T) {
	asym := func(a, b int) float64 {
		if a == 3 && b == 5 {
			return float64(a*b) + 1
		}
		return float64(a * b)
	}
	if res := CheckProfile(asym, 6, 12, 1e-9); len(res.Symmetry) == 0 {
		t.Errorf("asymmetric profile passed symmetry")
	}
	badUnit := func(a, b int) float64 {
		if a == 1 {
			return float64(b + 1)
		}
		return float64(a * b)
	}
	if res := CheckProfile(badUnit, 6, 12, 1e-9); len(res.UnitRow) == 0 {
		t.Errorf("bad unit row passed")
	}
	convex := func(a, b int) float64 { return float64(a) * float64(b*b) }
	if res := CheckProfile(convex, 6, 12, 1e-9); len(res.Concavity) == 0 {
		t.Errorf("convex profile passed concavity")
	}
}

func TestRankProfileTriplingWitnessSetIsExact(t *testing.T) {
	res := CheckProfile(RankProfile, 12, 40, 1e-9)
	if len(res.Tripling) != 121 {
		t.Fatalf("tripling witness count: got %d want 121, every (a,h) with 3h+a-1 <= 40 fires", len(res.Tripling))
	}
	last := res.Tripling[120]
	if last.A != 12 || last.H != 9 || last.B != 38 || last.P != 49 || last.Q != 60 {
		t.Fatalf("last witness: got %+v want a=12 h=9 b=38 P=49 Q=60", last)
	}
}

func TestConcavityWitnessCarriesTheNeighbourSum(t *testing.T) {
	convex := func(a, b int) float64 { return float64(a) * float64(b*b) }
	res := CheckProfile(convex, 6, 12, 1e-9)
	if len(res.Concavity) != 50 {
		t.Fatalf("concavity witness count: got %d want 50 over a in 2..6 and b in 2..11", len(res.Concavity))
	}
	w := res.Concavity[0]
	if w.A != 2 || w.B != 2 || w.P != 16 || w.Q != 20 {
		t.Fatalf("first witness: got %+v want a=2 b=2 P=16 Q=20 with Q = p(2,3)+p(2,1)", w)
	}
}

func TestTriplingToleranceRejectsEquality(t *testing.T) {
	boundary := func(a, b int) float64 { return float64(b) + 0.5 }
	res := CheckProfile(boundary, 2, 4, 1e-9)
	if len(res.Tripling) != 0 {
		t.Fatalf("equality at the tripling boundary produced %d witnesses, want none below epsilon", len(res.Tripling))
	}
}

func TestCapTLadderPinsChapterValues(t *testing.T) {
	cases := []struct {
		a    int
		want float64
	}{
		{2, 1.188721876},
		{4, 1.052758096},
		{16, 0.928911808},
		{256, 0.843485576},
		{65536, 0.796874484},
		{1048576, 0.787499974},
	}
	prev := math.Inf(1)
	for _, tc := range cases {
		got := CapT(tc.a)
		if math.Abs(got-tc.want) > 5e-9 {
			t.Errorf("CapT(%d): got %.9f want %.9f", tc.a, got, tc.want)
		}
		if got >= prev {
			t.Errorf("CapT not decreasing at a=%d: %v >= %v", tc.a, got, prev)
		}
		if got <= 0.75 {
			t.Errorf("CapT(%d) dropped to or below 3/4: %v", tc.a, got)
		}
		prev = got
	}
}
