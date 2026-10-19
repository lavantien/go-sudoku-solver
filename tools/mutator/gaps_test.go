package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureModule(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
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

func TestRunErrorsWithoutGoMod(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"subject/s.go": "package subject\n\nfunc Less(a, b int) bool { return a < b }\n",
	})
	if _, err := run(root, []string{"subject"}, "", 1); err == nil {
		t.Error("missing go.mod: got nil want error")
	}
}

func TestRunErrorsWithoutMutationPoints(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"go.mod":       "module fixture\n\ngo 1.27\n",
		"subject/s.go": "package subject\n\nvar Answer = 42\n",
	})
	if _, err := run(root, []string{"subject"}, "", 1); err == nil || !strings.Contains(err.Error(), "no mutation points") {
		t.Errorf("no mutation points: got %v want the discovery error", err)
	}
}

func TestDiscoverErrors(t *testing.T) {
	root := t.TempDir()
	if _, err := discover(root, []string{"gone"}); err == nil {
		t.Error("missing package dir: got nil want error")
	}
	broken := fixtureModule(t, map[string]string{"subject/broken.go": "package subject\nfunc {"})
	if _, err := discover(broken, []string{"subject"}); err == nil {
		t.Error("unparsable source: got nil want error")
	}
}

func TestModulePathOfErrorsWithoutGoMod(t *testing.T) {
	if _, err := modulePathOf(t.TempDir()); err == nil {
		t.Error("missing go.mod: got nil want error")
	}
}

func TestRunMutantRejectsUnusableInputs(t *testing.T) {
	root := fixtureModule(t, map[string]string{"s.go": "package main\n\nvar A = 1 < 2\n"})
	if got := runMutant(root, mutant{file: "gone.go", oldOp: "<", newOp: ">="}, "", "fixture"); got != outcomeInvalid {
		t.Errorf("missing file: got %v want invalid", got)
	}
	if got := runMutant(root, mutant{file: "s.go", offset: 25, oldOp: ">", newOp: "<=", line: 3}, "", "fixture"); got != outcomeInvalid {
		t.Errorf("wrong old op: got %v want invalid", got)
	}
}

func TestCopyModuleSkipsIgnoredTrees(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"go.mod":            "module fixture\n",
		"keep.go":           "package main\n",
		".hidden":           "x",
		".git/HEAD":         "ref",
		"vendor/v.go":       "package vendor\n",
		"internal/pkg/a.go": "package pkg\n",
	})
	tmp, err := copyModule(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	for _, gone := range []string{".git", "vendor", ".hidden"} {
		if _, err := os.Stat(filepath.Join(tmp, gone)); !os.IsNotExist(err) {
			t.Errorf("%s must not be copied", gone)
		}
	}
	for _, keep := range []string{"go.mod", "keep.go", filepath.Join("internal", "pkg", "a.go")} {
		if _, err := os.Stat(filepath.Join(tmp, keep)); err != nil {
			t.Errorf("%s must be copied: %v", keep, err)
		}
	}
}

func TestCopyModuleAndCopyFileErrors(t *testing.T) {
	if _, err := copyModule(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Error("missing root: got nil want error")
	}
	if err := copyFile(filepath.Join(t.TempDir(), "gone"), filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Error("missing source: got nil want error")
	}
	src := filepath.Join(t.TempDir(), "src")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, filepath.Join(t.TempDir(), "no-dir", "dst")); err == nil {
		t.Error("unwritable destination: got nil want error")
	}
}

func TestMutateInvalidMutantCountsInvalid(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"go.mod":               "module fixture\n\ngo 1.27\n",
		"subject/target.go":    readFixture(t, "target_invalid.go"),
		"subject/weak_test.go": readFixture(t, "weak_test.go"),
	})
	rep, err := run(root, []string{"subject"}, "mutatorfixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.mutants != 1 {
		t.Fatalf("mutants: got %d want the single concat flip", rep.mutants)
	}
	if rep.invalid != 1 {
		t.Fatalf("invalid: got %d want 1, the string concat flip must not build", rep.invalid)
	}
	if rep.survived != 0 || rep.killed != 0 {
		t.Fatalf("outcomes: killed %d survived %d want 0/0", rep.killed, rep.survived)
	}
}
