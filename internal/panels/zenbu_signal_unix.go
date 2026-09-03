//go:build unix

package panels

import "syscall"

var zenbuGroupSignal = syscall.Kill

var (
	zenbuSigStop = syscall.SIGSTOP
	zenbuSigCont = syscall.SIGCONT
	zenbuSigKill = syscall.SIGKILL
)
