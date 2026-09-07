//go:build darwin || linux

package main

import (
	"os/exec"
	"testing"
)

func TestIsolateOfficeProcessGroup(t *testing.T) {
	command := exec.Command("true")
	isolateOfficeProcessGroup(command)
	if command.SysProcAttr == nil || !command.SysProcAttr.Setpgid {
		t.Fatal("office command is not isolated in its own process group")
	}
}
