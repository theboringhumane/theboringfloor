//go:build darwin || linux

package main

import (
	"os/exec"
	"syscall"
)

func isolateOfficeProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
