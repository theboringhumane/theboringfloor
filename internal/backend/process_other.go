//go:build !darwin && !linux

package backend

import (
	"os"
	"os/exec"
	"syscall"
)

func isolateProcessGroup(cmd *exec.Cmd) {}

func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(sig)
}
