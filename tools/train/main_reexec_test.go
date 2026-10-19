package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const reexecEnv = "TRAIN_MAIN_REEXEC"

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

func TestMainRunsTheTrainer(t *testing.T) {
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		reexecEnv+"=1",
		"SUDOKU_TRAIN_SEEDS=2",
		"SUDOKU_TRAIN_TABLE="+filepath.Join(t.TempDir(), "table_gen.go"),
	)
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run: %v", err)
		}
		code = exitErr.ExitCode()
	}
	if code != 0 {
		t.Errorf("exit code: got %d want 0", code)
	}
}
