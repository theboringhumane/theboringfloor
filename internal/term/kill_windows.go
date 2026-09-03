//go:build windows

package term

import "os"

func killProcessGroup(pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = p.Kill()
}
