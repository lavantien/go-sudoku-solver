package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const reexecEnv = "HELP_MAIN_REEXEC"

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

func writeMakefile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Makefile")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRenderPinsTheAwkContract(t *testing.T) {
	fixture := strings.Join([]string{
		"CGO_ENV := CGO_ENABLED=1",
		"ADDR ?= 127.0.0.1:8080",
		"",
		".PHONY: help build test",
		"",
		"help: ## show this help",
		"\t@echo hi",
		"",
		"build: ## build the binary into dist/",
		"\tgo build -o dist/app .",
		"",
		"test: ## run tests twice ## and again",
		"\tgo test ./...",
		"",
		"run server: ## run the web server",
		"\tgo run .",
		"",
		"verylongtargetname: ## long target name",
		"\ttrue",
		"",
		"weird:##no space after hash",
		"\ttrue",
		"",
		"empty:##",
		"\ttrue",
		"",
		"\trecipe: ## tab indented, not a target",
	}, "\n")
	want := strings.Join([]string{
		"Usage: make <target>",
		"",
		"Targets:",
		"  help             show this help",
		"  build            build the binary into dist/",
		"  test             and again",
		"  verylongtargetname  long target name",
		"  weird           no space after hash",
		"  empty           ",
	}, "\n") + "\n"
	got, err := render(writeMakefile(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("render mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRenderKeepsSpaceNamedTargetsHidden(t *testing.T) {
	got, err := render(writeMakefile(t, "run server: ## run the web server\nbuild: ## build it\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "run server") {
		t.Errorf("space-named target must stay hidden: %q", got)
	}
	if !strings.Contains(got, "build") {
		t.Errorf("documented targets missing: %q", got)
	}
}

func TestRenderNormalizesCarriageReturns(t *testing.T) {
	dir := t.TempDir()
	lfPath := filepath.Join(dir, "lf")
	crlfPath := filepath.Join(dir, "crlf")
	if err := os.WriteFile(lfPath, []byte("build: ## build it\nrun server: ## hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(crlfPath, []byte("build: ## build it\r\nrun server: ## hidden\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lf, err := render(lfPath)
	if err != nil {
		t.Fatal(err)
	}
	crlf, err := render(crlfPath)
	if err != nil {
		t.Fatal(err)
	}
	if lf != crlf {
		t.Errorf("crlf checkout must render identically:\nlf:   %q\ncrlf: %q", lf, crlf)
	}
}

func TestRunFailsOnMissingMakefile(t *testing.T) {
	if code := run([]string{filepath.Join(t.TempDir(), "gone")}); code != 1 {
		t.Errorf("missing Makefile: got exit %d want 1", code)
	}
}

func TestMainDelegatesToRun(t *testing.T) {
	cases := []struct {
		name     string
		makefile func(t *testing.T) string
		wantCode int
	}{
		{"documented makefile passes", func(t *testing.T) string {
			return writeMakefile(t, "build: ## build it\n")
		}, 0},
		{"missing makefile fails", func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "gone")
		}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], tc.makefile(t))
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
