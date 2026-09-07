//go:build !darwin && !linux

package main

import "os/exec"

func prepareExecProcessGroup(command *exec.Cmd) {}

func killExecProcessGroup(command *exec.Cmd) error {
	if command.Process == nil {
		return nil
	}
	return command.Process.Kill()
}
