package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
)

func hangFixtureModule(t *testing.T) (string, string) {
	t.Helper()
	root := fixtureModule(t, map[string]string{
		"go.mod":               "module fixture\n\ngo 1.27\n",
		"subject/s.go":         "package subject\n\nvar A = 1 < 2\n",
		"subject/hang_test.go": readFixture(t, "hang_test.go"),
	})
	pidFile := filepath.Join(root, "sleeper.pid")
	t.Setenv("MUTATOR_TREEKILL_PIDFILE", pidFile)
	pre := exec.Command("go", "test", "-c", "-tags", "mutatorfixture", "-o", filepath.Join(root, "prebuilt.test"), "fixture/subject")
	pre.Dir = root
	if out, err := pre.CombinedOutput(); err != nil {
		t.Fatalf("fixture prebuild: %v: %s", err, out)
	}
	return root, pidFile
}

func TestCancelKillsTheWholeTestTree(t *testing.T) {
	root, pidFile := hangFixtureModule(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := goTestCmd(ctx, root, "fixture/subject", "mutatorfixture")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	up := time.Now().Add(time.Minute)
	for {
		if _, err := os.ReadFile(pidFile); err == nil {
			break
		}
		if time.Now().After(up) {
			_ = killTree(cmd.Process.Pid)
			t.Fatal("sleeper never reported in within a minute")
		}
		time.Sleep(100 * time.Millisecond)
	}
	cancel()
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(config.MutatorKillTreeGrace + 30*time.Second):
		t.Fatal("cmd.Wait never returned after the tree kill")
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("sleeper pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("sleeper pid %q: %v", raw, err)
	}
	deadline := time.Now().Add(config.MutatorKillTreeGrace + 20*time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("sleeper process %d survived the tree kill", pid)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestTestPkgClassifiesTimeoutAsInvalid(t *testing.T) {
	root, _ := hangFixtureModule(t)
	if got := testPkg(logging.New(""), root, "fixture/subject", "mutatorfixture", 3*time.Second); got != outcomeInvalid {
		t.Fatalf("outcome: got %v want invalid", got)
	}
}

func processAlive(pid int) bool {
	p := strconv.Itoa(pid)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "PID eq "+p, "/FO", "CSV", "/NH").Output()
		return err == nil && strings.Contains(string(out), `"`+p+`"`)
	}
	return exec.Command("kill", "-0", p).Run() == nil
}
