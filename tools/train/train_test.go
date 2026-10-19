package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/matmul"
)

func tinyTrainCfg(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Defaults()
	cfg.Train.Seeds = 4
	cfg.Train.TablePath = filepath.Join(t.TempDir(), "table_gen.go")
	return cfg
}

func TestTrainTinyCorpusIsByteIdenticalOnRerun(t *testing.T) {
	cfg := tinyTrainCfg(t)
	logger := logging.New("")
	if err := run(logger, cfg); err != nil {
		t.Fatalf("first train run: %v", err)
	}
	first, err := os.ReadFile(cfg.Train.TablePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := run(logger, cfg); err != nil {
		t.Fatalf("second train run: %v", err)
	}
	second, err := os.ReadFile(cfg.Train.TablePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("rerun rewrote the table:\nfirst:  %q\nsecond: %q", first, second)
	}
	if !strings.HasPrefix(string(first), "package policy\n\n//go:generate go run ../../tools/train\nvar TableWeights = [][]float64{\n") {
		t.Errorf("table header malformed: %q", first[:min(120, len(first))])
	}
	if got := strings.Count(string(first), "\t{"); got != 3 {
		t.Errorf("table rows: got %d want 3 racer rows", got)
	}
	if !strings.HasSuffix(string(first), "}\n") {
		t.Errorf("table trailer malformed: %q", first[len(first)-10:])
	}
}

func TestFitStrassenCutoffReproducesClassicalBytes(t *testing.T) {
	cfg := tinyTrainCfg(t)
	logger := logging.New("")
	samples, err := corpus(logger, cfg)
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	width := len(samples[0].features)
	padded := 1
	for padded < width {
		padded *= 2
	}
	defaultCutoff := config.Defaults().MatmulCutoff
	atPadded, err := fit(logger, samples, padded)
	if err != nil {
		t.Fatalf("fit at the padded cutoff %d: %v", padded, err)
	}
	atDefault, err := fit(logger, samples, defaultCutoff)
	if err != nil {
		t.Fatalf("fit at the default cutoff %d: %v", defaultCutoff, err)
	}
	if !bytes.Equal(render(atPadded), render(atDefault)) {
		t.Errorf("cutoff %d drifted from the classical bytes kept at the default cutoff %d:\npadded:  %q\ndefault: %q",
			padded, defaultCutoff, render(atPadded), render(atDefault))
	}
}

func TestTrainTablePathHonorsOverrideAndDefault(t *testing.T) {
	cfg := config.Defaults()
	cfg.Train.TablePath = filepath.Join("x", "y", "t.go")
	got, err := tablePath(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("x", "y", "t.go") {
		t.Errorf("override: got %q", got)
	}
	def, err := tablePath(config.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(def), "internal/policy/table_gen.go") {
		t.Errorf("default: got %q want the committed policy table", def)
	}
}

func TestTrainEmitFailureFailsTheRun(t *testing.T) {
	cfg := tinyTrainCfg(t)
	cfg.Train.TablePath = filepath.Join(t.TempDir(), "no-dir", "table_gen.go")
	if err := run(logging.New(""), cfg); err == nil {
		t.Error("unwritable table path: got nil want error")
	}
}

func TestTrainCliExitCodes(t *testing.T) {
	cfg := tinyTrainCfg(t)
	if code := cli(cfg); code != 0 {
		t.Errorf("cli on a tiny corpus: got exit %d want 0", code)
	}
	cfg.Train.TablePath = filepath.Join(t.TempDir(), "no-dir", "table_gen.go")
	if code := cli(cfg); code != 1 {
		t.Errorf("cli with an unwritable table: got exit %d want 1", code)
	}
}

func TestInvertRejectsSingularAndSwapsPivots(t *testing.T) {
	singular := matmul.Dense{N: 2, Data: []float64{0, 1, 0, 0}}
	if _, err := invert(singular); err == nil {
		t.Error("singular matrix: got nil want error")
	}
	needsSwap := matmul.Dense{N: 2, Data: []float64{0, 1, 1, 2}}
	inv, err := invert(needsSwap)
	if err != nil {
		t.Fatal(err)
	}
	prod := matmul.MulClassical(needsSwap, inv)
	for i := range 4 {
		want := 0.0
		if i == 0 || i == 3 {
			want = 1.0
		}
		if math.Abs(prod.Data[i]-want) > 1e-9 {
			t.Errorf("inverse at %d: got %v want %v", i, prod.Data[i], want)
		}
	}
}

func TestClueTargetStaysInTheBand(t *testing.T) {
	for seed := uint64(1); seed <= 50; seed++ {
		got := clueTarget(seed)
		if got < clueBase || got >= clueBase+clueSpan {
			t.Fatalf("seed %d: clue target %d outside %d..%d", seed, got, clueBase, clueBase+clueSpan-1)
		}
	}
}
