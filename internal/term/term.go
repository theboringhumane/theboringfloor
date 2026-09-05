// Package term embeds an OS-level terminal: spawn the user's shell on a
// real PTY (github.com/creack/pty — darwin/linux), stream its output into
// a scrollback, forward keystrokes, and resize on layout change.
//
// term.go — the Session: process lifecycle + a non-blocking reader loop.
//
// KEYBOARD CONTRACT (what panels.TermPanel forwards per keystroke):
//
//	printable chars      → msg.Text bytes verbatim (utf-8)
//	enter                → "\r"
//	backspace            → 0x7f  (DEL — what real terminals send)
//	tab                  → 0x09  (completion)
//	shift+tab            → "\x1b[Z"  (reverse completion)
//	esc                  → 0x1b
//	up/down/right/left   → "\x1b[A" … "\x1b[D"  (vt100 cursor keys)
//	home/end             → "\x1b[H" / "\x1b[F"
//	pgup/pgdown          → "\x1b[5~" / "\x1b[6~"
//	delete               → "\x1b[3~"
//	space                → " "
//	ctrl+c/d/x (all ctrl+<letter>) → 0x01..0x1a pass-through
//	ctrl+space / ctrl+o  → RESERVED by the app (the capture pair: ctrl+space
//	                       — 0x00 — TOGGLES shell capture BOTH ways, opt-in,
//	                       released is the default; ctrl+o releases OUT as
//	                       an alias); neither ever reaches the shell.
//	                       ctrl+i can never be a binding: it IS tab (0x09).
//	bracketed paste / F-keys / kitty modifiers → NOT forwarded in v1
//
// Process-group discipline: creack/pty's StartWithSize configures
// Setsid+Setctty, so the shell is a session leader and its own pgroup
// leader — syscall.Kill(-pid, SIGKILL) kills the whole group (shell +
// any pipeline children), and there is never a Quit path that hangs on
// a live child. Caller-callable multiple times; Kill is idempotent.
package term

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/gitx"
)

// ptyResult is the platform-neutral contract returned by platformStart.
// Each platform file (pty_unix.go, pty_windows.go) provides a
// platformStart that fills this in.
type ptyResult struct {
	master   io.ReadWriteCloser
	pid      int
	waitFn   func() (int, error)
	resizeFn func(cols, rows int) error
	closeFn  func()
}

// DefaultShell picks the user's shell: $SHELL, else /bin/zsh (darwin
// default) if present, else /bin/bash.
func DefaultShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	if os.PathSeparator == '\\' {
		if s := os.Getenv("COMSPEC"); s != "" {
			return s
		}
		return `C:\Windows\System32\cmd.exe`
	}
	for _, cand := range []string{"/bin/zsh", "/bin/bash"} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return "/bin/sh"
}

// TermConfig is the spawn-time shape of a Session.
type TermConfig struct {
	Shell string   // default: DefaultShell()
	Cols  int      // default: 80
	Rows  int      // default: 24
	CWD   string   // default: caller's working directory
	Env   []string // default: os.Environ() + TERM + COLORTERM; either way the majdoor GIT_* vars merge in when THEFLOOR_AUTO_COMMIT=true
}

// Session is one live PTY-bound shell process.
type Session struct {
	cfg      TermConfig
	mf       io.ReadWriteCloser // pty master (*os.File on Unix, conptyMaster on Windows)
	sb       *Scrollback
	grid     *Grid  // live screen model (same bytes as sb, parsed)
	name     string // shell basename (ps/pgrep proofs)
	pid      int
	waitFn   func() (int, error)
	resizeFn func(cols, rows int) error
	closeFn  func()

	mu     sync.Mutex
	alive  bool
	exited bool
	code   int // exit code, -1 while alive / kilt unknown
	closed bool
	err    error
}

// Spawn starts the shell under a new PTY of cfg.Cols x cfg.Rows and
// immediately starts the reader loop; the returned Session is already
// streaming. Spawn never blocks the caller on shell startup.
func Spawn(cfg TermConfig) (*Session, error) {
	if cfg.Shell == "" {
		cfg.Shell = DefaultShell()
	}
	if cfg.Cols < 2 {
		cfg.Cols = 80
	}
	if cfg.Rows < 1 {
		cfg.Rows = 24
	}
	env := cfg.Env
	if env == nil {
		env = os.Environ()
		env = append(env, "TERM=xterm-256color", "COLORTERM=truecolor")
	}
	// Majdoor attribution: when the office's auto-commit flag is on, any
	// `git commit` run inside this shell is authored by the majdoor (the
	// four GIT_* vars win over any inherited ones). No-op otherwise.
	env = gitx.WithMajdoorAuthorEnv(env)

	result, err := platformStart(cfg, env)
	if err != nil {
		return nil, fmt.Errorf("term: spawn %s: %w", cfg.Shell, err)
	}

	s := &Session{
		cfg:      cfg,
		mf:       result.master,
		sb:       NewScrollback(defaultScrollback),
		grid:     NewGrid(cfg.Cols, cfg.Rows),
		name:     filepath.Base(cfg.Shell),
		pid:      result.pid,
		waitFn:   result.waitFn,
		resizeFn: result.resizeFn,
		closeFn:  result.closeFn,
		alive:    true,
		code:     -1,
	}
	s.startReader()
	go s.waiter()
	return s, nil
}

// startReader launches the goroutine draining the PTY master into the
// scrollback. It exits when the PTY goes away (shell exit or Kill) and
// MUST never block the TUI: all output lands behind the scrollback mutex.
func (s *Session) startReader() {
	go func() {
		// io.Copy on the master: EOF when every slave closes. Every byte
		// lands in BOTH the raw scrollback (durable history) and the grid
		// (live screen model) — one parse per delta, O(bytes), and the
		// grid nudges the app repaint channel on the way through.
		dst := io.MultiWriter(s.sb, s.grid)
		if _, err := io.Copy(dst, s.mf); err != nil {
			s.mu.Lock()
			s.err = err
			s.mu.Unlock()
		}
		// reader finishing means the PTY is gone → mark dead so Alive()
		// flips even if cmd.Wait is still reaping.
		s.mu.Lock()
		s.alive = false
		s.mu.Unlock()
	}()
}

// waiter reaps the child so it never zombifies and records the exit code.
func (s *Session) waiter() {
	code, _ := s.waitFn()
	s.mu.Lock()
	s.alive, s.exited, s.code = false, true, code
	s.mu.Unlock()
}

// Read drains newly-arrived output accumulated since the last call.
// Non-blocking: returns nil when nothing new. (The scrollback is the
// durable store; Read is for consumers that want deltas.)
func (s *Session) Read() ([]byte, error) {
	b := s.sb.DrainNew()
	if werr := s.errSnapshot(); werr != nil && !s.Alive() {
		return b, werr
	}
	return b, nil
}

func (s *Session) errSnapshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if errors.Is(s.err, os.ErrClosed) || s.err == nil {
		return nil
	}
	return s.err
}

// Write forwards raw key bytes to the shell's PTY master.
func (s *Session) Write(p []byte) (int, error) {
	if !s.Alive() {
		return 0, fmt.Errorf("term: write to dead session")
	}
	return s.mf.Write(p)
}

// Resize hands the PTY a new window size (SIGWINCH to the foreground
// process group happens implicitly on darwin/linux).
func (s *Session) Resize(cols, rows int) error {
	if cols < 2 {
		cols = 2
	}
	if rows < 1 {
		rows = 1
	}
	s.mu.Lock()
	s.cfg.Cols, s.cfg.Rows = cols, rows
	s.mu.Unlock()
	// reshape the screen model first so the SIGWINCH reprint paints into
	// the new geometry (top-left content kept by the grid).
	s.grid.SetSize(cols, rows)
	return s.resizeFn(cols, rows)
}

// Size reports the current PTY geometry.
func (s *Session) Size() (cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Cols, s.cfg.Rows
}

// Kill terminates the whole process group (the shell is a session leader,
// so -pid addresses shell + pipeline children), closes the master, and
// waits briefly for the waiter to reap. Idempotent — safe at app quit.
func (s *Session) Kill() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	pid := s.pid
	s.mu.Unlock()

	// Group kill first (best effort), then close platform resources before
	// the master so the reader unblocks even on the error path.
	killProcessGroup(pid)
	s.closeFn()
	_ = s.mf.Close()

	for i := 0; i < 50; i++ {
		if s.Exited() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// Close is the io.Closer spelling of Kill (app quit path).
func (s *Session) Close() error { return s.Kill() }

// Alive reports whether the shell is believed running (false once the PTY
// output stream ends or the process exits).
func (s *Session) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

// Exited reports whether cmd.Wait has returned (process fully reaped).
func (s *Session) Exited() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exited
}

// ExitCode returns the shell's exit code; -1 while alive or unknown.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.code
}

// Pid is the shell's process id (also its pgroup id — kill -Pid works).
func (s *Session) Pid() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pid
}

// ShellName is the shell binary basename ("zsh", "bash", …) — handy for
// zombie checks via pgrep.
func (s *Session) ShellName() string { return s.name }

// Scrollback exposes the output buffer (last-N rendering, search, etc).
func (s *Session) Scrollback() *Scrollback { return s.sb }

// Grid exposes the live screen model (cursor-positioned, SGR-styled cells)
// panels paint from — the alt screen's content while ?1049 is set.
func (s *Session) Grid() *Grid { return s.grid }
