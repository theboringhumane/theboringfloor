//go:build !darwin && !linux

package main

import "os/exec"

func isolateOfficeProcessGroup(cmd *exec.Cmd) {}
