//go:build windows

package panels

import (
	"os"
	"syscall"
)

var (
	zenbuSigStop syscall.Signal
	zenbuSigCont syscall.Signal
	zenbuSigKill = syscall.SIGKILL
)

var zenbuGroupSignal = windowsGroupSignal

func windowsGroupSignal(pid int, sig syscall.Signal) error {
	if sig != zenbuSigKill && sig != syscall.SIGKILL {
		return nil
	}
	if pid < 0 {
		pid = -pid
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
