//go:build darwin || linux

package backend

import (
	"os"
	"os/exec"
	"syscall"
)

// isolateProcessGroup makes the spawned agent and any descendants one
// teardown unit. The TUI remains in its original process group.
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessGroup(proc *os.Process, sig syscall.Signal) error {
	if proc == nil {
		return nil
	}
	return syscall.Kill(-proc.Pid, sig)
}
