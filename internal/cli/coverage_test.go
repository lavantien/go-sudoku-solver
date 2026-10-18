package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func writeCoverageInputs(t *testing.T, profile, readme string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "coverage.out")
	readmePath := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	return profilePath, readmePath
}

func TestRunCoverageRewritesBadgeBetweenMarkers(t *testing.T) {
	profilePath, readmePath := writeCoverageInputs(t,
		"mode: set\nfoo.go:1.1,2.2 4 1\nbar.go:3.1,4.2 4 0\n",
		"# p\n\n<!-- coverage:START -->\nold\n<!-- coverage:END -->\n")
	out, _ := withCapture(t, "", func() {
		if _, err := coverageRun(profilePath, readmePath); err != nil {
			t.Error(err)
		}
	})
	got, _ := os.ReadFile(readmePath)
	want := "# p\n\n<!-- coverage:START -->\ncoverage: 50.0% of statements\n<!-- coverage:END -->\n"
	if string(got) != want {
		t.Errorf("readme =\n%q\nwant\n%q", got, want)
	}
	if !strings.Contains(out, "50.0%") {
		t.Errorf("stdout = %q, want the computed percent", out)
	}
}

func TestRunCoverageCreatesSectionWhenAbsent(t *testing.T) {
	profilePath, readmePath := writeCoverageInputs(t,
		"mode: set\nfoo.go:1.1,2.2 5 1\n",
		"no markers\n")
	_, _ = withCapture(t, "", func() {
		if _, err := coverageRun(profilePath, readmePath); err != nil {
			t.Error(err)
		}
	})
	got, _ := os.ReadFile(readmePath)
	want := "no markers\n\n<!-- coverage:START -->\ncoverage: 100.0% of statements\n<!-- coverage:END -->\n"
	if string(got) != want {
		t.Errorf("readme =\n%q\nwant\n%q", got, want)
	}
}

func TestRunCoverageIdempotentSecondRunKeepsBytes(t *testing.T) {
	profilePath, readmePath := writeCoverageInputs(t,
		"mode: set\nfoo.go:1.1,2.2 4 1\nbar.go:3.1,4.2 4 0\n",
		"<!-- coverage:START -->\nold\n<!-- coverage:END -->\n")
	_, _ = withCapture(t, "", func() {
		if _, err := coverageRun(profilePath, readmePath); err != nil {
			t.Error(err)
		}
	})
	first, _ := os.ReadFile(readmePath)
	_, _ = withCapture(t, "", func() {
		if _, err := coverageRun(profilePath, readmePath); err != nil {
			t.Error(err)
		}
	})
	second, _ := os.ReadFile(readmePath)
	if string(first) != string(second) {
		t.Errorf("second run changed the readme:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestRunCoverageMissingProfileErrors(t *testing.T) {
	if _, err := coverageRun(filepath.Join(t.TempDir(), "nope"), "readme.md"); err == nil {
		t.Error("missing profile should error")
	}
}

func TestRunCoverageBadProfileErrors(t *testing.T) {
	profilePath, readmePath := writeCoverageInputs(t,
		"mode: set\nfoo.go:1.1,2.2 x 1\n",
		"x\n")
	if _, err := coverageRun(profilePath, readmePath); err == nil {
		t.Error("unparsable profile should error")
	}
}

func TestRunCoverageMissingReadmeErrors(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(profilePath, []byte("mode: set\nfoo.go:1.1,2.2 5 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := coverageRun(profilePath, filepath.Join(dir, "gone.md")); err == nil {
		t.Error("missing readme should error")
	}
}

func TestRunCoverageRejectsUnexpectedArg(t *testing.T) {
	if err := runCoverage(context.Background(), config.Defaults(), []string{"extra"}); err == nil {
		t.Error("coverage must reject positional arguments")
	}
}

func TestRunCoverageRejectsUnknownFlag(t *testing.T) {
	if err := runCoverage(context.Background(), config.Defaults(), []string{"-nope"}); err == nil {
		t.Error("coverage must reject unknown flags")
	}
}

func TestRunCoverageFlagsDriveTheRun(t *testing.T) {
	profilePath, readmePath := writeCoverageInputs(t,
		"mode: set\nfoo.go:1.1,2.2 4 1\nbar.go:3.1,4.2 4 0\n",
		"<!-- coverage:START -->\nold\n<!-- coverage:END -->\n")
	_, _ = withCapture(t, "", func() {
		if err := runCoverage(context.Background(), config.Defaults(),
			[]string{"-profile", profilePath, "-readme", readmePath}); err != nil {
			t.Error(err)
		}
	})
	got, _ := os.ReadFile(readmePath)
	if !strings.Contains(string(got), "coverage: 50.0% of statements") {
		t.Errorf("readme badge not refreshed: %s", got)
	}
}

func TestRunCoverageMissingProfileFailsTheCommand(t *testing.T) {
	if err := runCoverage(context.Background(), config.Defaults(),
		[]string{"-profile", filepath.Join(t.TempDir(), "nope.out"), "-readme", "readme.md"}); err == nil {
		t.Error("missing profile must fail the command")
	}
}

func TestRunCoverageUnwritableReadmeErrors(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "coverage.out")
	readmePath := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(profilePath, []byte("mode: set\nfoo.go:1.1,2.2 4 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readmePath, []byte("readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readmePath, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readmePath, 0o644) })
	if _, err := coverageRun(profilePath, readmePath); err == nil {
		t.Error("unwritable readme: got nil want error")
	}
}

func TestRunCoverageRegisteredInRegistry(t *testing.T) {
	c, ok := Lookup(config.CmdCoverage)
	if !ok {
		t.Fatal("coverage command not registered")
	}
	if c.Name != config.CmdCoverage || c.Usage == "" {
		t.Errorf("registered command = %+v", c)
	}
}
