package main

import (
	"path/filepath"
	"testing"
)

func TestRunPassesCleanTree(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, filepath.Join(root, "ok.go"), codeLines(10))
	if code := run([]string{root}); code != 0 {
		t.Errorf("clean tree: got exit %d want 0", code)
	}
}

func TestRunDefaultsToWorkingDirAndPasses(t *testing.T) {
	if code := run(nil); code != 0 {
		t.Errorf("repo root: got exit %d want 0", code)
	}
}

func TestCountReportsUnreadableFiles(t *testing.T) {
	if _, err := count(filepath.Join(t.TempDir(), "gone.go")); err == nil {
		t.Error("missing file: got nil want error")
	}
}

func TestRunFailsOnViolation(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, filepath.Join(root, "over.go"), codeLines(1001))
	if code := run([]string{root}); code != 1 {
		t.Errorf("over-limit tree: got exit %d want 1", code)
	}
}

func TestRunFailsOnMissingRoot(t *testing.T) {
	if code := run([]string{filepath.Join(t.TempDir(), "gone")}); code != 1 {
		t.Errorf("missing root: got exit %d want 1", code)
	}
}
