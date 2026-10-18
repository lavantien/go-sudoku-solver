package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsBoundaryOperators(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sample")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package sample\n\nfunc Less(a, b int) bool { return a < b }\nfunc Eq(a, b int) bool { return a == b }\nfunc Both(a, b bool) bool { return a && b }\n"
	if err := os.WriteFile(filepath.Join(dir, "s.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ms, err := discover(root, []string{"sample"})
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]bool{}
	for _, m := range ms {
		ops[m.oldOp+"->"+m.newOp] = true
		if m.file != "sample/s.go" {
			t.Errorf("mutant file = %q, want sample/s.go", m.file)
		}
	}
	for _, want := range []string{"<->>=", "==->!=", "&&->||"} {
		if !ops[want] {
			t.Errorf("missing flip %s in %v", want, ops)
		}
	}
}

func TestDiscoverSkipsTestFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sample"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package sample\n\nfunc Less(a, b int) bool { return a < b }\n"
	if err := os.WriteFile(filepath.Join(root, "sample", "s.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample", "s_test.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ms, err := discover(root, []string{"sample"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("mutants: got %d want 1 from the non-test file only", len(ms))
	}
}

func TestReplaceAt(t *testing.T) {
	b := []byte("a == b")
	got := replaceAt(b, 2, "==", "!=")
	if string(got) != "a != b" {
		t.Fatalf("replace failed: %q", got)
	}
	if replaceAt(b, 2, "!=", "==") != nil {
		t.Error("mismatched old op must be rejected")
	}
	if replaceAt(b, 100, "==", "!=") != nil {
		t.Error("out of range must be rejected")
	}
}

func TestPkgFor(t *testing.T) {
	cases := []struct {
		modulePath, file, want string
	}{
		{"github.com/lavantien/go-sudoku-solver", "internal/store/store.go", "github.com/lavantien/go-sudoku-solver/internal/store"},
		{"fixture", "subject/target.go", "fixture/subject"},
		{"fixture", "main.go", "fixture"},
	}
	for _, c := range cases {
		if got := pkgFor(c.modulePath, c.file); got != c.want {
			t.Errorf("pkgFor(%q, %q) = %q, want %q", c.modulePath, c.file, got, c.want)
		}
	}
}

func TestModulePathOf(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := modulePathOf(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "fixture" {
		t.Errorf("modulePathOf = %q, want fixture", got)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture // local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err = modulePathOf(root); err != nil || got != "fixture" {
		t.Errorf("modulePathOf with trailing comment = %q, %v, want fixture", got, err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := modulePathOf(root); err == nil {
		t.Error("missing module directive should error")
	}
}

func TestClampParallel(t *testing.T) {
	cases := []struct {
		name        string
		requested   int
		mutantCount int
		procs       int
		want        int
	}{
		{"zero uses procs", 0, 100, 8, 8},
		{"negative uses procs", -3, 100, 4, 4},
		{"capped at slots", 32, 100, 32, 8},
		{"procs capped too", 0, 100, 64, 8},
		{"floored at one", 0, 100, 0, 1},
		{"trimmed to mutants", 8, 3, 8, 3},
		{"explicit under cap", 4, 100, 8, 4},
	}
	for _, c := range cases {
		if got := clampParallel(c.requested, c.mutantCount, c.procs); got != c.want {
			t.Errorf("%s: clampParallel(%d, %d, %d) = %d, want %d",
				c.name, c.requested, c.mutantCount, c.procs, got, c.want)
		}
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeFixtureModule(t *testing.T, testFile string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":              "module fixture\n\ngo 1.27\n",
		"subject/target.go":   readFixture(t, "target.go"),
		"subject/" + testFile: readFixture(t, testFile),
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestMutateWeakFixtureReportsSurvivor(t *testing.T) {
	root := writeFixtureModule(t, "weak_test.go")
	rep, err := run(root, []string{"subject"}, "mutatorfixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.mutants != 3 {
		t.Fatalf("mutants: got %d want 3", rep.mutants)
	}
	if rep.invalid != 0 {
		t.Fatalf("invalid: got %d want 0, the fixture must build in every shadow", rep.invalid)
	}
	if rep.survived != 1 {
		t.Fatalf("survived: got %d want 1, the weak mid-only test must let the && -> || mutant live: %+v",
			rep.survived, rep.survivors)
	}
	s := rep.survivors[0]
	if s.oldOp != "&&" || s.newOp != "||" {
		t.Fatalf("survivor = %s -> %s, want && -> ||", s.oldOp, s.newOp)
	}
}

func TestMutateStrongFixtureKillsEveryMutant(t *testing.T) {
	root := writeFixtureModule(t, "strong_test.go")
	rep, err := run(root, []string{"subject"}, "mutatorfixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.mutants != 3 {
		t.Fatalf("mutants: got %d want 3", rep.mutants)
	}
	if rep.invalid != 0 {
		t.Fatalf("invalid: got %d want 0, the fixture must build in every shadow", rep.invalid)
	}
	if rep.survived != 0 {
		t.Fatalf("survivors against the strong table: %+v", rep.survivors)
	}
	if rep.killed != 3 {
		t.Fatalf("killed: got %d want 3", rep.killed)
	}
}
