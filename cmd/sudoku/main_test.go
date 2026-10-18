package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

const reexecEnv = "SUDOKU_MAIN_REEXEC"

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

func TestMainDelegatesToTheCli(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"version", []string{config.CmdVersion}, 0, "sudoku v1.0\n"},
		{"solve with empty stdin", []string{config.CmdSolve}, 1, ""},
		{"unknown command", []string{"bogus"}, 2, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], tc.args...)
			cmd.Env = append(os.Environ(), reexecEnv+"=1")
			cmd.Stdin = bytes.NewReader(nil)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			code := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("run %v: %v", tc.args, err)
				}
				code = exitErr.ExitCode()
			}
			if code != tc.wantCode {
				t.Errorf("exit code: got %d want %d", code, tc.wantCode)
			}
			if out.String() != tc.wantOut {
				t.Errorf("stdout: got %q want %q", out.String(), tc.wantOut)
			}
		})
	}
}
