package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGoFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func codeLines(n int) string {
	return strings.Repeat("x := 1\n", n)
}

func TestWalkTreePassesAtTheLimit(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, filepath.Join(root, "at-limit.go"), codeLines(1000))
	v, err := walkTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Fatalf("violations at exactly 1000 lines: %+v", v)
	}
}

func TestWalkTreeFailsOverTheLimit(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, filepath.Join(root, "over.go"), codeLines(1001))
	v, err := walkTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 {
		t.Fatalf("violations: got %d want 1", len(v))
	}
	if v[0].path != filepath.Join(root, "over.go") {
		t.Errorf("violation path = %q, want %q", v[0].path, filepath.Join(root, "over.go"))
	}
	if v[0].lines != 1001 {
		t.Errorf("violation lines = %d, want 1001", v[0].lines)
	}
}

func TestWalkTreeSkipsVendoredAndBuildDirs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"vendor", ".tools", "dist", "playground", filepath.Join("pkg", "vendor")} {
		writeGoFile(t, filepath.Join(root, dir, "over.go"), codeLines(1001))
	}
	writeGoFile(t, filepath.Join(root, "over.txt"), codeLines(1001))
	v, err := walkTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 0 {
		t.Fatalf("skipped dirs must not be counted: %+v", v)
	}
}

func TestCountIgnoresBlanksAndComments(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "c.go")
	body := "a := 1\n\n// line comment\n/* block start\nblock end */\n/* inline */\nb := 2\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := count(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2 code lines", n)
	}
}

func TestCountEmptyFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "empty.go")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := count(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count = %d, want 0", n)
	}
}
