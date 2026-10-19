package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
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

func TestClueTargetStaysInTheBand(t *testing.T) {
	for seed := uint64(1); seed <= 50; seed++ {
		got := clueTarget(seed)
		if got < clueBase || got >= clueBase+clueSpan {
			t.Fatalf("seed %d: clue target %d outside %d..%d", seed, got, clueBase, clueBase+clueSpan-1)
		}
	}
}
