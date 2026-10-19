package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var flips = map[string]string{
	"==": "!=", "!=": "==",
	"<": ">=", ">=": "<",
	">": "<=", "<=": ">",
	"&&": "||", "||": "&&",
	"+": "-", "-": "+",
}

const (
	maxParallelSlots = 8
	testTimeout      = 3 * time.Minute
	killTreeGrace    = 10 * time.Second
)

type mutant struct {
	file   string
	offset int
	oldOp  string
	newOp  string
	line   int
}

type report struct {
	slots     int
	mutants   int
	killed    int
	invalid   int
	survived  int
	survivors []mutant
}

type testOutcome int

const (
	outcomeKilled testOutcome = iota
	outcomeSurvived
	outcomeInvalid
)

func main() {
	os.Exit(cli(os.Args[1:]))
}

func cli(args []string) int {
	fs := flag.NewFlagSet("mutator", flag.ContinueOnError)
	pkgs := fs.String("packages", "", "comma-separated package dirs, e.g. internal/store")
	tags := fs.String("tags", "", "build tags forwarded to go test")
	parallel := fs.Int("parallel", 0, "concurrent mutant slots; 0 = min(8, GOMAXPROCS), capped at 8")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pkgs == "" {
		fmt.Fprintln(os.Stderr, "-packages required")
		return 2
	}
	fmt.Fprintf(os.Stderr, "mutate: discovering mutants in %s\n", *pkgs)
	rep, err := run(".", strings.Split(*pkgs, ","), *tags, *parallel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, m := range rep.survivors {
		fmt.Printf("SURVIVED %s:%d %s -> %s\n", m.file, m.line, m.oldOp, m.newOp)
	}
	fmt.Printf("slots=%d mutants=%d killed=%d invalid=%d survived=%d\n",
		rep.slots, rep.mutants, rep.killed, rep.invalid, rep.survived)
	if rep.survived > 0 {
		return 1
	}
	return 0
}

func run(root string, pkgDirs []string, tags string, parallel int) (*report, error) {
	modulePath, err := modulePathOf(root)
	if err != nil {
		return nil, err
	}
	mutants, err := discover(root, pkgDirs)
	if err != nil {
		return nil, err
	}
	if len(mutants) == 0 {
		return nil, fmt.Errorf("no mutation points found in %v", pkgDirs)
	}
	workers := clampParallel(parallel, len(mutants), runtime.GOMAXPROCS(0))

	shadows := make([]string, workers)
	for i := range shadows {
		shadows[i], err = copyModule(root)
		if err != nil {
			cleanupShadows(shadows[:i])
			return nil, err
		}
	}
	defer cleanupShadows(shadows)

	results := make([]testOutcome, len(mutants))
	jobs := make(chan int, len(mutants))
	for i := range mutants {
		jobs <- i
	}
	close(jobs)

	var done atomic.Int64
	var wg sync.WaitGroup
	for _, shadow := range shadows {
		wg.Add(1)
		go func(shadow string) {
			defer wg.Done()
			for i := range jobs {
				results[i] = runMutant(shadow, mutants[i], tags, modulePath)
				if n := done.Add(1); n%10 == 0 {
					fmt.Fprintf(os.Stderr, "progress %d/%d\n", n, len(mutants))
				}
			}
		}(shadow)
	}
	wg.Wait()

	rep := &report{slots: workers, mutants: len(mutants)}
	for i, m := range mutants {
		switch results[i] {
		case outcomeKilled:
			rep.killed++
		case outcomeInvalid:
			rep.invalid++
		default:
			rep.survived++
			rep.survivors = append(rep.survivors, m)
		}
	}
	return rep, nil
}

func runMutant(shadow string, m mutant, tags, modulePath string) testOutcome {
	dst := filepath.Join(shadow, filepath.FromSlash(m.file))
	orig, err := os.ReadFile(dst)
	if err != nil {
		return outcomeInvalid
	}
	mutated := replaceAt(orig, m.offset, m.oldOp, m.newOp)
	if mutated == nil {
		return outcomeInvalid
	}
	if err := os.WriteFile(dst, mutated, 0o644); err != nil {
		return outcomeInvalid
	}
	outcome := testPkg(shadow, pkgFor(modulePath, m.file), tags, testTimeout)
	_ = os.WriteFile(dst, orig, 0o644)
	return outcome
}

func cleanupShadows(shadows []string) {
	for _, s := range shadows {
		if s != "" {
			_ = os.RemoveAll(s)
		}
	}
}

func discover(root string, pkgDirs []string) ([]mutant, error) {
	var out []mutant
	for _, dir := range pkgDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			ms, err := discoverFile(path, filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			out = append(out, ms...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func discoverFile(path, rel string) ([]mutant, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}
	var out []mutant
	ast.Inspect(f, func(n ast.Node) bool {
		be, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		newOp, ok := flips[be.Op.String()]
		if !ok {
			return true
		}
		pos := fset.Position(be.OpPos)
		out = append(out, mutant{
			file:   rel,
			offset: fset.File(be.OpPos).Offset(be.OpPos),
			oldOp:  be.Op.String(),
			newOp:  newOp,
			line:   pos.Line,
		})
		return true
	})
	return out, nil
}

func replaceAt(b []byte, offset int, oldOp, newOp string) []byte {
	if offset < 0 || offset+len(oldOp) > len(b) || string(b[offset:offset+len(oldOp)]) != oldOp {
		return nil
	}
	out := make([]byte, 0, len(b)+len(newOp)-len(oldOp))
	out = append(out, b[:offset]...)
	out = append(out, newOp...)
	out = append(out, b[offset+len(oldOp):]...)
	return out
}

func testPkg(moduleRoot, pkg, tags string, timeout time.Duration) testOutcome {
	ctx, cancelCtx := context.WithTimeout(context.Background(), timeout)
	defer cancelCtx()
	cmd := goTestCmd(ctx, moduleRoot, pkg, tags)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil || strings.Contains(buf.String(), "[build failed]") {
			return outcomeInvalid
		}
		return outcomeKilled
	}
	return outcomeSurvived
}

func goTestCmd(ctx context.Context, moduleRoot, pkg, tags string) *exec.Cmd {
	args := []string{"test", "-run", "^Test"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, pkg)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = moduleRoot
	cmd.Env = os.Environ()
	cmd.Cancel = func() error { return killTree(cmd.Process.Pid) }
	cmd.WaitDelay = killTreeGrace
	return cmd
}

func killTree(pid int) error {
	p := strconv.Itoa(pid)
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/T", "/F", "/PID", p).Run()
	}
	if err := exec.Command("pkill", "-9", "-P", p).Run(); err != nil {
		return err
	}
	return exec.Command("kill", "-9", p).Run()
}

func pkgFor(modulePath, relFile string) string {
	dir := filepath.ToSlash(filepath.Dir(relFile))
	if dir == "." {
		return modulePath
	}
	return modulePath + "/" + dir
}

func modulePathOf(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, ln := range strings.Split(string(b), "\n") {
		f := strings.Fields(ln)
		if len(f) >= 2 && f[0] == "module" {
			return f[1], nil
		}
	}
	return "", fmt.Errorf("go.mod: module directive not found")
}

func clampParallel(requested, mutantCount, procs int) int {
	w := requested
	if w <= 0 {
		w = procs
	}
	if w > maxParallelSlots {
		w = maxParallelSlots
	}
	if w < 1 {
		w = 1
	}
	if mutantCount > 0 && w > mutantCount {
		w = mutantCount
	}
	return w
}

func copyModule(root string) (string, error) {
	tmp, err := os.MkdirTemp("", "mutate-")
	if err != nil {
		return "", err
	}
	skip := map[string]bool{
		".git": true, ".tools": true, "database": true, "dist": true,
		"docs": true, "playground": true, "vendor": true, "web": true,
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (skip[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(tmp, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return copyFile(path, dst)
	})
	if err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	return tmp, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}
