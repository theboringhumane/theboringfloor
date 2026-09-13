//go:build darwin || linux

package app

import (
	"os/exec"
	"syscall"
)

// isolateHandoffProcessGroup gives the detached handoff office its own
// process group so this process's later exec (or death) never signals it —
// mirrors cmd/floorgate/process_unix.go's isolateOfficeProcessGroup and
// internal/backend/process_unix.go's isolateProcessGroup (same Setpgid
// recipe, reimplemented here since neither package can be imported from
// internal/app — floorgate is `package main`, backend is off-limits per
// this task's scope).
func isolateHandoffProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
