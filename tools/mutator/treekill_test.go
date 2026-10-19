package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTestPkgKillsTheTreeOnTimeout(t *testing.T) {
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
	done := make(chan testOutcome, 1)
	go func() { done <- testPkg(root, "fixture/subject", "mutatorfixture", 5*time.Second) }()
	var got testOutcome
	select {
	case got = <-done:
	case <-time.After(5*time.Second + killTreeGrace + time.Minute):
		t.Fatal("testPkg never returned after the mutant timeout")
	}
	if got != outcomeInvalid {
		t.Fatalf("outcome: got %v want invalid", got)
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("sleeper pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("sleeper pid %q: %v", raw, err)
	}
	deadline := time.Now().Add(killTreeGrace + 20*time.Second)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			t.Fatalf("sleeper process %d survived the tree kill", pid)
		}
		time.Sleep(200 * time.Millisecond)
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
