package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
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
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	logger := logging.New(config.Load().LogLevel)
	logging.Enter(logger, "sloc", "root", root)
	v, err := walkTree(root)
	if err != nil {
		logging.Fail(logger, "sloc", err, "root", root)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, f := range v {
		fmt.Printf("SLOC FAIL %s: %d > %d\n", f.path, f.lines, maxSLOC)
	}
	if len(v) > 0 {
		logging.Exit(logger, "sloc", "violations", len(v))
		return 1
	}
	fmt.Println("sloc ok")
	logging.Exit(logger, "sloc", "violations", 0)
	return 0
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
