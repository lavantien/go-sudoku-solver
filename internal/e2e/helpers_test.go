package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

const (
	envBin                 = "SUDOKU_BIN"
	envUI                  = "SUDOKU_E2E_UI"
	envBrowser             = "SUDOKU_BROWSER"
	runTimeout             = 4 * time.Minute
	annealBlockPlaceholder = "<anneal best-effort grid>"
)

var (
	e2eBin     string
	stepDigits = regexp.MustCompile(`Step count: [0-9]+`)
	timeDigits = regexp.MustCompile(`Execution time: [0-9]+\.[0-9]+`)
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	if p := os.Getenv(envBin); p != "" {
		e2eBin = p
		fmt.Fprintf(os.Stderr, "e2e: reusing %s=%s\n", envBin, p)
		return m.Run()
	}
	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: %v\n", err)
		return 1
	}
	dir, err := os.MkdirTemp("", "sudoku-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: temp dir: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	e2eBin = filepath.Join(dir, binName())
	build := exec.Command("go", "build", "-o", e2eBin, "./cmd/sudoku")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=1")
	var out bytes.Buffer
	build.Stdout = &out
	build.Stderr = &out
	fmt.Fprintf(os.Stderr, "e2e: building the binary into %s\n", e2eBin)
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: go build failed: %v\n%s", err, out.String())
		return 1
	}
	fmt.Fprintf(os.Stderr, "e2e: binary ready at %s\n", e2eBin)
	return m.Run()
}

func binName() string {
	if runtime.GOOS == "windows" {
		return config.BinName + ".exe"
	}
	return config.BinName
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

func sudokuBin(t *testing.T) string {
	t.Helper()
	if e2eBin == "" {
		t.Fatalf("e2e binary was not built")
	}
	if _, err := os.Stat(e2eBin); err != nil {
		t.Fatalf("e2e binary missing: %v", err)
	}
	return e2eBin
}

type binaryRun struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func runSudoku(t *testing.T, stdin string, args ...string) binaryRun {
	t.Helper()
	t.Logf("e2e: run args=%v stdin_len=%d", args, len(stdin))
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, sudokuBin(t), args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	cmd.Env = cleanSudokuEnv()
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("run timed out after %v: args=%v", runTimeout, args)
	}
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run failed: %v", err)
		}
		code = exitErr.ExitCode()
	}
	t.Logf("e2e: run done exit=%d out_len=%d err_len=%d", code, out.Len(), errOut.Len())
	return binaryRun{ExitCode: code, Stdout: out.String(), Stderr: errOut.String()}
}

func cleanSudokuEnv() []string {
	env := make([]string, 0, 16)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "SUDOKU_") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

func normalizeReport(t *testing.T, report string) string {
	t.Helper()
	report = stepDigits.ReplaceAllString(report, "Step count: 0")
	report = timeDigits.ReplaceAllString(report, "Execution time: 0.000000")
	parts := strings.Split(report, config.Separator)
	if len(parts) != 1+2*len(config.MethodIDs) {
		t.Fatalf("report sections: got %d want %d", len(parts), 1+2*len(config.MethodIDs))
	}
	idx := 1 + 2*2
	part := parts[idx]
	blank := strings.Index(part, "\n\n")
	if blank < 0 {
		t.Fatalf("anneal block lacks the label blank line: %q", part)
	}
	parts[idx] = "\n" + annealBlockPlaceholder + "\n" + part[blank+1:]
	return strings.Join(parts, config.Separator)
}

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("load golden %s: %v", name, err)
	}
	return string(raw)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want := loadGolden(t, name)
	if got == want {
		return
	}
	t.Fatalf("golden %s mismatch:\n%s", name, firstDiff(want, got))
}

func firstDiff(want, got string) string {
	wl := strings.Split(want, "\n")
	gl := strings.Split(got, "\n")
	for i := range max(len(wl), len(gl)) {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return fmt.Sprintf("first diff at line %d:\n want: %q\n  got: %q", i+1, w, g)
		}
	}
	return "no line diff, the byte streams differ only in line endings or trailing bytes"
}

func nineLineInput(grid81 string) string {
	var b strings.Builder
	for r := range config.Size {
		b.WriteString(grid81[r*config.Size : (r+1)*config.Size])
		b.WriteByte('\n')
	}
	return b.String()
}
