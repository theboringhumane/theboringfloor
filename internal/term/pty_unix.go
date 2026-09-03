//go:build !windows

package term

import (
	"os/exec"

	"github.com/creack/pty"
)

func platformStart(cfg TermConfig, env []string) (*ptyResult, error) {
	cmd := exec.Command(cfg.Shell, "-i")
	if cfg.CWD != "" {
		cmd.Dir = cfg.CWD
	}
	cmd.Env = env
	mf, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(cfg.Rows),
		Cols: uint16(cfg.Cols),
	})
	if err != nil {
		return nil, err
	}
	return &ptyResult{
		master: mf,
		pid:    cmd.Process.Pid,
		waitFn: func() (int, error) {
			err := cmd.Wait()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					return ee.ExitCode(), err
				}
				return -1, err
			}
			return 0, nil
		},
		resizeFn: func(cols, rows int) error {
			return pty.Setsize(mf, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		},
		closeFn: func() {},
	}, nil
}
