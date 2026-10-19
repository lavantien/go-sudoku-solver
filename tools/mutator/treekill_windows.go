//go:build windows

package main

import (
	"os/exec"
	"strconv"
)

func killTree(pid int) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}

func setNewProcessGroup(*exec.Cmd) {}
