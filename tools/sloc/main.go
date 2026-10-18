package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxSLOC = 1000

var skipDirs = map[string]bool{
	".git": true, ".tools": true, "dist": true, "playground": true, "vendor": true,
}

type violation struct {
	path  string
	lines int
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	v, err := walkTree(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, f := range v {
		fmt.Printf("SLOC FAIL %s: %d > %d\n", f.path, f.lines, maxSLOC)
	}
	if len(v) > 0 {
		os.Exit(1)
	}
	fmt.Println("sloc ok")
}

func walkTree(root string) ([]violation, error) {
	var out []violation
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		n, err := count(path)
		if err != nil {
			return err
		}
		if n > maxSLOC {
			out = append(out, violation{path: path, lines: n})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func count(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(b), "\n")
	n := 0
	inBlock := false
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		switch {
		case t == "":
		case inBlock:
			if strings.Contains(t, "*/") {
				inBlock = false
			}
		case strings.HasPrefix(t, "//"):
		case strings.HasPrefix(t, "/*"):
			if !strings.Contains(t, "*/") {
				inBlock = true
			}
		default:
			n++
		}
	}
	return n, nil
}
