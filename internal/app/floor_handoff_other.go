//go:build !darwin && !linux

package app

import "os/exec"

// isolateHandoffProcessGroup is a no-op on platforms without POSIX process
// groups — mirrors cmd/floorgate/process_other.go.
func isolateHandoffProcessGroup(cmd *exec.Cmd) {}
