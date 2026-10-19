package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const reexecEnv = "MUTATOR_MAIN_REEXEC"

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

func TestMainRejectsMissingPackages(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-packages", "")
	cmd.Env = append(os.Environ(), reexecEnv+"=1")
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run: %v", err)
		}
		code = exitErr.ExitCode()
	}
	if code != 2 {
		t.Errorf("exit code: got %d want 2", code)
	}
}

func TestRunReportsProgressEveryTenMutants(t *testing.T) {
	var src, rows strings.Builder
	src.WriteString("package subject\n\n")
	rows.WriteString("package subject\n\nimport \"testing\"\n\nfunc TestBoundaries(t *testing.T) {\n\tpairs := []struct {\n\t\tf   func(int, int) bool\n\t\ta   int\n\t\tb   int\n\t\tout bool\n\t}{\n")
	for i := range 10 {
		name := "L" + string(rune('0'+i))
		src.WriteString("func " + name + "(a, b int) bool { return a < b }\n")
		rows.WriteString("\t\t{" + name + ", 1, 2, true}, {" + name + ", 2, 1, false},\n")
	}
	rows.WriteString("\t}\n\tfor _, p := range pairs {\n\t\tif p.f(p.a, p.b) != p.out {\n\t\t\tt.Fail()\n\t\t}\n\t}\n}\n")
	root := fixtureModule(t, map[string]string{
		"go.mod":            "module fixture\n\ngo 1.27\n",
		"subject/b.go":      src.String(),
		"subject/b_test.go": rows.String(),
	})
	rep, err := run(root, []string{"subject"}, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if rep.mutants != 10 || rep.killed != 10 || rep.survived != 0 || rep.invalid != 0 {
		t.Errorf("report: got %+v want 10 killed mutants", *rep)
	}
}

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

func TestCliExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, 2},
		{"missing packages flag", []string{"-tags", "x"}, 2},
		{"unknown flag", []string{"-nope"}, 2},
		{"unparsable args", []string{"-parallel", "x", "-packages", "subject"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code := cli(tc.args); code != tc.want {
				t.Errorf("cli(%v): got exit %d want %d", tc.args, code, tc.want)
			}
		})
	}
	root := fixtureModule(t, map[string]string{
		"go.mod":               "module fixture\n\ngo 1.27\n",
		"subject/target.go":    readFixture(t, "target_invalid.go"),
		"subject/weak_test.go": readFixture(t, "weak_test.go"),
	})
	t.Chdir(root)
	if code := cli([]string{"-packages", "subject", "-parallel", "1"}); code != 0 {
		t.Errorf("invalid-only run: got exit %d want 0", code)
	}
	if code := cli([]string{"-packages", "gone"}); code != 1 {
		t.Errorf("discovery failure: got exit %d want 1", code)
	}
	survivor := fixtureModule(t, map[string]string{
		"go.mod":            "module fixture\n\ngo 1.27\n",
		"subject/s.go":      "package subject\n\nvar A = 1 < 2\n",
		"subject/s_test.go": "package subject\n\nimport \"testing\"\n\nfunc TestNothing(t *testing.T) { _ = A }\n",
	})
	t.Chdir(survivor)
	if code := cli([]string{"-packages", "subject", "-parallel", "1"}); code != 1 {
		t.Errorf("survivor run: got exit %d want 1", code)
	}
}

func TestDiscoverSkipsBlankEntriesAndReadFailures(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"go.mod":       "module fixture\n",
		"subject/s.go": "package subject\n\nvar A = 1 < 2\n",
	})
	ms, err := discover(root, []string{"  ", "", "subject"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("mutants after blank entries: got %d want 1", len(ms))
	}
	if _, err := discoverFile(filepath.Join(root, "gone.go"), "gone.go"); err == nil {
		t.Error("missing file: got nil want error")
	}
}

func TestRunMutantRejectsUnwritableFile(t *testing.T) {
	root := fixtureModule(t, map[string]string{"s.go": "package main\n\nvar A = 1 < 2\n"})
	path := filepath.Join(root, "s.go")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	got := runMutant(root, mutant{file: "s.go", offset: 24, oldOp: "<", newOp: ">=", line: 3}, "", "fixture")
	if got != outcomeInvalid {
		t.Errorf("readonly target: got %v want invalid", got)
	}
}

func TestDiscoverReportsNonFlippableOperators(t *testing.T) {
	root := fixtureModule(t, map[string]string{
		"go.mod":       "module fixture\n",
		"subject/s.go": "package subject\n\nvar A = 2 * 3\nvar B = 1 < 2\n",
	})
	ms, err := discover(root, []string{"subject"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("mutants: got %d want only the boundary flip", len(ms))
	}
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
