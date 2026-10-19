//go:build mutatorfixture

package subject

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestHang(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run", "^TestSleeper$")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if p := os.Getenv("MUTATOR_TREEKILL_PIDFILE"); p != "" {
		_ = os.WriteFile(p, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestSleeper(t *testing.T) {
	time.Sleep(time.Hour)
}
