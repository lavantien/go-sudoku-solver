//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func killTree(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

func setNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
