package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const reexecEnv = "SLOC_MAIN_REEXEC"

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

func TestMainDelegatesToRun(t *testing.T) {
	cases := []struct {
		name     string
		arg      func(t *testing.T) string
		wantCode int
	}{
		{"clean tree passes", func(t *testing.T) string {
			dir := t.TempDir()
			writeGoFile(t, filepath.Join(dir, "ok.go"), codeLines(10))
			return dir
		}, 0},
		{"missing root fails", func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "gone")
		}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], tc.arg(t))
			cmd.Env = append(os.Environ(), reexecEnv+"=1")
			err := cmd.Run()
			code := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("run: %v", err)
				}
				code = exitErr.ExitCode()
			}
			if code != tc.wantCode {
				t.Errorf("exit code: got %d want %d", code, tc.wantCode)
			}
		})
	}
}
