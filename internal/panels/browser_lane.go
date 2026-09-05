// browser_lane.go — the browser tab's PREMIUM render lane: zenbu's
// terminal-browser (https://github.com/zenbu-labs/terminal-browser — a
// full Chromium app painting pages over the kitty graphics protocol; a
// distributed BINARY on PATH, never a Go dependency) embedded INSIDE the
// browser tab's pane, with the universal text-mode HTML viewer as the
// fallback everywhere the premium gates miss.
//
// LANE RESOLVE (memoized per boot, the images-lane memoization idiom —
// one honest env+PATH read per BrowserLaneController, never per frame):
//
//	kitty-capable host (DetectImageSupportFrom == KittyLane: kitty's own
//	  KITTY_WINDOW_ID, or TERM_PROGRAM ghostty|kitty; tmux folds out
//	  conservatively, iTerm2/WezTerm/VSCode/xterm are NOT this lane)
//	AND exec.LookPath("terminal-browser") finds the binary
//	AND neither kill-switch is armed: THEFLOOR_TERMINAL_BROWSER_OFF
//	  (this lane's own gate) or THEFLOOR_NO_TERMINAL_BROWSER
//	  (wave 70's documented `o`-lane gate — one zenbu off-switch contract,
//	  both spellings honored so an armed member is never surprised)
//	AND the lane is explicitly OPTED IN: THEFLOOR_ZENBU_LANE=1
//	  (the wave-85 default-off pivot — the embedded lane is RETAINED but
//	  opt-in; headless screenshots are the default premium path now)
//	→ BrowserLaneZenbu; every miss → BrowserLaneText.
//
// THE REASON RESOLVE (ResolveBrowserLaneReasonFrom — the same gates in
// the same order, PLUS the WHY): a text-lane resolve never stays silent
// anymore — the pane reads the memoized verdict's reason class (premium /
// opt-in-off / binary-missing / terminal-unsupported / kill-switch, the
// kill-switch AND opt-in-off classes naming WHICH var is in play) and
// paints ONE dim hint row
// under the location bar (the starter card wears it too), so the member
// sees the lane's reason at the moment of disappointment instead of
// guessing terminal-browser was never integrated.
//
// THE EMBED — the EXACT terminal.go seam reused, no third tty layer: the
// child spawns on a creack/pty PTY (pty.StartWithSize — Setsid+Setctty,
// the child is its own pgroup leader) with stdout/stderr painting INTO
// the pane's embedded terminal model (term.Grid + term.Scrollback, the
// same pair TermPanel paints from), and ZenbuSession satisfies the
// panel's termSess contract so the browser tab drives it through
// TermPanel's paint/key path verbatim (ctrl+space opt-in capture forwards
// key bytes through Write — the tab owns the toggle, this file owns the
// pipe). term.Spawn itself is shell-locked (`<shell> -i`), so the command
// spawn lives here against the SAME primitives; a term-level
// SpawnCommand can absorb it later without touching the contract.
//
// KEYS / CHROME while premium runs: the location bar collapses to the
// top-only strip "▸ zenbu terminal-browser · <url>" and the lane badge
// flips to " zenbu "; on ANY exit the pane is back to the text-mode
// location bar and the " text " badge.
//
// FALLBACK (exact): a non-zero exit OR an early exit (< zenbuEarlyExit,
// 300ms) latches the text lane for THAT url with the dim note
// "zenbu exited (<code>) — falling back to text mode" (a spawn failure
// wears 127, the POSIX not-found code). Text-mode URL state persists —
// the current url + the visited history survive the drop untouched (no
// re-fetch: this layer never fetches at all). A clean long-run exit
// (the member quit the browser) returns to text mode silently.
//
// LIFECYCLE (the member's keep-alive ruling — the page is "always
// shown"): leaving the browser tab (ctrl+b to the floor, the pane's
// q/esc) FREEZES the premium child instead of killing it — the splitter
// parks FIRST (a CONSUMPTION PAUSE, never a reset: the in-flight APC
// tail + open chunk chain are PRESERVED in the splitter's buffers —
// zero store/grid/scrollback mutations while suspended — and any byte
// the still-running child writes before the stop takes effect lands in
// the pending buffer, never the grid), the frame-splice registry clears
// (the wrapper's a=d — the floor never shows the page), and ONLY THEN
// the process group is SIGSTOPped (CPU off; the PTY's kernel buffer
// backpressures the child naturally). The image store RETAINS the
// latest joined frame. Returning THAWS the SAME child (SIGCONT — the
// PID never changes across a flip, one backgrounded Electron's RAM is
// the accepted cost): the preserved pending chain's tail COMPLETES it
// into a valid frame (a mid-chain SIGSTOP is the common case at ~2fps —
// resetting at the freeze dropped the chain's HEAD and the resumed tail
// painted the grid with raw base64: the production freeze-leak), and
// the retained frame re-emits through the frame-splice wrapper on the
// very next flush: the member sees the last painted page INSTANTLY,
// before the child emits a byte. The KILL paths are unchanged: office
// quit / lane Close, a fresh OpenURL (kill + spawn fresh), and the
// early-exit fallback still group-SIGKILL + bounded-reap
// (zenbuKillGrace, opencode.go's stopKillGrace discipline) + delete the
// images + RESET the splitter exactly as before (a dead child's chain
// never completes); a child found DEAD at the thaw (murdered while
// frozen) resets the splitter first, then is observed by Poll and rides
// the existing fallback. Close is idempotent.
package panels

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/term"
)

// BrowserLane — which render lane the browser tab paints with.
type BrowserLane int

const (
	// BrowserLaneText — the universal text-mode HTML viewer (renders
	// everywhere; the fallback for every premium-lane miss).
	BrowserLaneText BrowserLane = iota
	// BrowserLaneZenbu — zenbu's terminal-browser embedded in the pane
	// (kitty-capable host + binary on PATH + no kill-switch + the
	// THEFLOOR_ZENBU_LANE=1 opt-in — default-off since wave 85).
	BrowserLaneZenbu
)

// String — the lane word for notices/tests/proofs.
func (l BrowserLane) String() string {
	if l == BrowserLaneZenbu {
		return "zenbu"
	}
	return "text"
}

// BrowserLaneOffEnv — the premium lane's own kill-switch, read AT USE
// TIME with no config schema field (the THEFLOOR_MUTE house style;
// TerminalBrowserOffEnv's wave-70 contract). wave 70's
// TerminalBrowserOffEnv is honored too: ONE off-switch contract, both
// spellings — the lane is off when either reads "1".
const BrowserLaneOffEnv = "THEFLOOR_TERMINAL_BROWSER_OFF"

// BrowserLaneOptInEnv — the premium lane's OPT-IN flag (the wave-85
// default-off pivot): the embedded zenbu lane resolves premium ONLY when
// this reads "1" at pane-creation time — unset, "0", or anything else
// keeps the universal text lane even on a fully-qualified host (kitty
// terminal + terminal-browser on PATH + no kill-switch). Read AT USE
// TIME with no config schema field (BrowserLaneOffEnv's house style):
// the lane is retained for members who want it, but headless screenshots
// are the default premium path now, so the embedded Chromium never boots
// unless the member explicitly asks.
const BrowserLaneOptInEnv = "THEFLOOR_ZENBU_LANE"

// zenbuLookPath — the binary probe (links.go's openLookPath precedent):
// exec.LookPath by default, swapped by tests to prove the
// PATH-resolution-failure leg without depending on the host PATH.
var zenbuLookPath = exec.LookPath

// ResolveBrowserLane — the browser tab's lane, live-read (fresh env +
// PATH probe; callers wanting the per-boot memo use a
// BrowserLaneResolver).
func ResolveBrowserLane() BrowserLane { return ResolveBrowserLaneFrom(browserLaneEnv, zenbuLookPath) }

// BrowserLaneReason — WHY the lane resolved the way it did (the pane's
// hint-row class; the gate that missed, in the resolve's own precedence:
// kill-switch → opt-in-off → terminal → binary — the opt-in-off class is
// mutually exclusive with the terminal/binary classes: it only fires when
// BOTH qualify).
type BrowserLaneReason int

const (
	// BrowserLanePremium — every gate passed: the premium embed is
	// available (the pane paints NO hint row).
	BrowserLanePremium BrowserLaneReason = iota
	// BrowserLaneNoBinary — kitty-capable host, no kill-switch, but the
	// `terminal-browser` PATH probe missed (the actionable class: install
	// the binary / re-run the office installer).
	BrowserLaneNoBinary
	// BrowserLaneNoTerminal — the host terminal can't host the embedded
	// browser (not kitty-capable: the tmux fold, the iTerm2 family,
	// xterm/unknowns).
	BrowserLaneNoTerminal
	// BrowserLaneKillSwitch — a kill-switch env is armed; the verdict's
	// third return names WHICH spelling.
	BrowserLaneKillSwitch
	// BrowserLaneOptInOff — terminal AND binary both qualify but the
	// opt-in flag is unset (the wave-85 default-off class: the embedded
	// lane is retained but opt-in — headless screenshots are the default
	// premium path). The verdict's third return names the flag
	// (BrowserLaneOptInEnv), the killSwitchVar-style passthrough the hint
	// renderer consumes.
	BrowserLaneOptInOff
)

// String — the reason word for tests/notices.
func (r BrowserLaneReason) String() string {
	switch r {
	case BrowserLanePremium:
		return "premium"
	case BrowserLaneNoBinary:
		return "no-binary"
	case BrowserLaneNoTerminal:
		return "no-terminal"
	case BrowserLaneKillSwitch:
		return "kill-switch"
	case BrowserLaneOptInOff:
		return "opt-in-off"
	}
	return "unknown"
}

// ResolveBrowserLaneReason — the live-read reasoned resolve (fresh env +
// PATH probe; callers wanting the per-pane memo use a BrowserLaneResolver).
func ResolveBrowserLaneReason() (BrowserLane, BrowserLaneReason, string) {
	return ResolveBrowserLaneReasonFrom(browserLaneEnv, zenbuLookPath)
}

// browserLaneEnv routes this package's product-scoped settings through the
// canonical accessor while leaving terminal-detection inputs untouched.
func browserLaneEnv(name string) string {
	switch name {
	case BrowserLaneOffEnv:
		return config.Env("TERMINAL_BROWSER_OFF")
	case TerminalBrowserOffEnv:
		return config.Env("NO_TERMINAL_BROWSER")
	case BrowserLaneOptInEnv:
		return config.Env("ZENBU_LANE")
	default:
		return os.Getenv(name)
	}
}

// ResolveBrowserLaneReasonFrom — the pure reasoned core
// (DetectImageSupportFrom's shape: env + probe injected, the matrix a
// shell-out-free table). The SAME gates as ResolveBrowserLaneFrom (which
// now delegates here) in the SAME precedence, plus the reason class and —
// for the kill-switch and opt-in-off classes — WHICH var is in play:
//
//  1. a kill-switch reads "1" (BrowserLaneOffEnv first, then
//     TerminalBrowserOffEnv) → text + BrowserLaneKillSwitch + the var name;
//  2. the host is NOT kitty-capable per the detect layer's OWN truth
//     table (DetectImageSupportFrom != KittyLane — ghostty/kitty are the
//     lane; tmux folds out, the iterm family is a different protocol and
//     a dead-end here) → text + BrowserLaneNoTerminal (the probe never
//     fires on a non-kitty host — the historic discipline);
//  3. the probe finds no `terminal-browser` binary → text +
//     BrowserLaneNoBinary;
//  4. terminal AND binary both qualify but BrowserLaneOptInEnv does not
//     read "1" (unset, "0", anything else — the wave-85 default-off
//     pivot) → text + BrowserLaneOptInOff + BrowserLaneOptInEnv;
//
// every gate passed → BrowserLaneZenbu + BrowserLanePremium. (The class
// precedence is kill-switch > opt-in-off > terminal > binary > premium:
// opt-in-off can only fire when terminal+binary both qualify, so its
// position against the two earlier classes is never observable — only
// the kill-switch's precedence over it is.)
func ResolveBrowserLaneReasonFrom(env func(string) string, lookPath func(string) (string, error)) (BrowserLane, BrowserLaneReason, string) {
	if strings.TrimSpace(env(BrowserLaneOffEnv)) == "1" {
		return BrowserLaneText, BrowserLaneKillSwitch, BrowserLaneOffEnv
	}
	if strings.TrimSpace(env(TerminalBrowserOffEnv)) == "1" {
		return BrowserLaneText, BrowserLaneKillSwitch, TerminalBrowserOffEnv
	}
	if DetectImageSupportFrom(env) != KittyLane {
		return BrowserLaneText, BrowserLaneNoTerminal, ""
	}
	if _, err := lookPath("terminal-browser"); err != nil {
		return BrowserLaneText, BrowserLaneNoBinary, ""
	}
	if strings.TrimSpace(env(BrowserLaneOptInEnv)) != "1" {
		return BrowserLaneText, BrowserLaneOptInOff, BrowserLaneOptInEnv
	}
	return BrowserLaneZenbu, BrowserLanePremium, ""
}

// ResolveBrowserLaneFrom — the pure lane-only core, kept for the existing
// callers/tests: the reasoned resolve's lane half.
func ResolveBrowserLaneFrom(env func(string) string, lookPath func(string) (string, error)) BrowserLane {
	lane, _, _ := ResolveBrowserLaneReasonFrom(env, lookPath)
	return lane
}

// browserLaneHintText — the EXACT dim hint copy per reason class (frozen,
// uishot-pinned): ONE row telling the member WHY the text lane is showing
// and, when actionable, how to get the full renderer. "" for the premium
// class (no hint row anywhere while premium is available).
func browserLaneHintText(reason BrowserLaneReason, killVar string) string {
	switch reason {
	case BrowserLaneNoBinary:
		return "text lane — terminal-browser not on PATH · full rendering: github.com/zenbu-labs/terminal-browser (or re-run the office installer)"
	case BrowserLaneNoTerminal:
		return "text lane — this terminal can't host the embedded browser (kitty/ghostty only)"
	case BrowserLaneKillSwitch:
		return "text lane — " + killVar + "=1 set; unset it for the embedded browser"
	case BrowserLaneOptInOff:
		return "text lane — the embedded browser is opt-in: " + killVar + "=1 to enable it"
	}
	return ""
}

// BrowserLaneResolver — the per-pane memo (app/images.go's
// detectImageLane idiom: one honest read, then zero env/PATH traffic per
// frame). One per browser tab; the pane's controller pins it AT CREATION
// (the env+PATH state at pane-creation time is the contract — a later
// install or env flip never changes a live pane's lane story). Harnesses
// that stub the terminal env per drive build a fresh resolver per drive
// (or Reset), exactly like the per-Model image-lane memo.
type BrowserLaneResolver struct {
	lane    BrowserLane
	reason  BrowserLaneReason
	killVar string
	ok      bool
}

// NewBrowserLaneResolver returns a cold resolver (first Lane() reads).
func NewBrowserLaneResolver() *BrowserLaneResolver { return &BrowserLaneResolver{} }

// resolve — the memoized read: the FIRST call reads env+PATH live, later
// calls return the pin.
func (r *BrowserLaneResolver) resolve() {
	if !r.ok {
		r.lane, r.reason, r.killVar = ResolveBrowserLaneReason()
		r.ok = true
	}
}

// Lane — the memoized lane.
func (r *BrowserLaneResolver) Lane() BrowserLane {
	r.resolve()
	return r.lane
}

// Verdict — the memoized resolve in full: the lane, the reason class, and
// the armed kill-switch's spelling ("" unless the class is kill-switch) —
// the pane's hint-row input.
func (r *BrowserLaneResolver) Verdict() (BrowserLane, BrowserLaneReason, string) {
	r.resolve()
	return r.lane, r.reason, r.killVar
}

// Reset drops the pin (the next Lane() re-reads) — shots/tests only.
func (r *BrowserLaneResolver) Reset() { r.ok = false }

// -------------------------------------------------------------------
// the embedded zenbu child (terminal.go's PTY seam, command-flavored)
// -------------------------------------------------------------------

// zenbuSess — the controller's session seam: termSess (terminal.go's
// EXACT panel contract — Alive/Close/Write/Resize/Size/ExitCode/Grid/
// Scrollback) plus the exit introspection the fallback rules need and
// the keep-alive freeze/thaw pair (Suspend/Resume). *ZenbuSession
// satisfies it in production; tests drive a fake.
type zenbuSess interface {
	termSess
	// Exited — cmd.Wait returned (the process is fully reaped).
	Exited() bool
	// Lifetime — start→exit while dead, start→now while alive (the
	// early-exit rule's clock).
	Lifetime() time.Duration
	// URL — the page this process was opened with.
	URL() string
	// Freeze — the keep-alive suspend: park the splitter (a consumption
	// pause — the pending APC tail + open chunk chain are PRESERVED),
	// clear the frame registry, and SIGSTOP the process group (the park
	// engages BEFORE the signal, so a racing write lands in the pending
	// buffer, never the grid); the store RETAINS the latest joined
	// frame. Idempotent; an already-dead child is a silent no-op
	// (Poll's fallback owns it).
	Freeze() error
	// Unfreeze — the keep-alive resume: unpark the splitter (the
	// preserved chain's tail completes it into a valid frame) and
	// SIGCONT the SAME process group (never a respawn); a child found
	// DEAD resets the splitter first (its chain never completes).
	// Idempotent.
	Unfreeze() error
	// Frozen — the keep-alive posture (parked + SIGSTOPped, store
	// retained, PID unchanged).
	Frozen() bool
}

// Timing discipline (vars, not consts — opencode.go's stopKillGrace
// idiom: deadline tests shrink them).
var (
	// zenbuEarlyExit — an exit FASTER than this means the binary never
	// really started painting: fall back to text mode with the dim note.
	zenbuEarlyExit = 300 * time.Millisecond
	// zenbuKillGrace caps SIGKILL → reap for the embedded child
	// (stopKillGrace's twin).
	zenbuKillGrace = 1 * time.Second
)

// ZenbuSession — ONE live `terminal-browser open <url>` child on its own
// PTY, mirroring term.Session's discipline byte-for-byte: creack/pty's
// StartWithSize (Setsid+Setctty → the child leads its own process group),
// the reader loop multiwrites into the Grid + Scrollback pair the pane
// paints from, the waiter reaps so nothing zombifies, and Close
// group-kills (-pid SIGKILL) then waits bounded by zenbuKillGrace.
type ZenbuSession struct {
	url     string
	cmd     *exec.Cmd
	mf      *os.File // pty master
	sb      *term.Scrollback
	grid    *term.Grid
	images  *zenbuImageStore // the kitty passthrough's live state (browser_lane_kitty.go)
	split   *kittyStream     // the stream splitter (the kill paths reset it; the freeze PRESERVES its pending tail/chain)
	started time.Time

	mu        sync.Mutex
	alive     bool
	exited    bool
	code      int           // exit code, -1 while alive / unknown
	life      time.Duration // frozen at reap
	closed    bool
	suspended bool // the keep-alive freeze (SIGSTOPped, store retained)
}

// newZenbuSession spawns `terminal-browser open <url>` on a cols×rows PTY
// and starts streaming immediately (term.Spawn's exact shape: env passes
// through with TERM/COLORTERM pinned for the PTY — zenbu probes the
// OUTER markers for its kitty lane, which os.Environ carries; the exec
// package's last-wins dedup puts the PTY's TERM on top, same as
// term.Spawn).
func newZenbuSession(url string, cols, rows int) (*ZenbuSession, error) {
	if cols < 2 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	cmd := exec.Command("terminal-browser", "open", url)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	mf, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, fmt.Errorf("zenbu: spawn terminal-browser: %w", err)
	}
	s := &ZenbuSession{
		url:     url,
		cmd:     cmd,
		mf:      mf,
		sb:      term.NewScrollback(0), // 0 → term's defaultScrollback
		grid:    term.NewGrid(cols, rows),
		images:  newZenbuImageStore(),
		started: time.Now(),
		alive:   true,
		code:    -1,
	}
	// FIX B: the image store knows the pane's body box from the PTY's
	// first byte — every re-emitted frame carries c=cols,r=rows so the
	// outer terminal SCALES the image to the pane (the child sends no
	// c=/r=; kitty's native-pixel default painted the wrong size).
	s.images.setBodyBox(cols, rows)
	// the reader loop (term.Session.startReader's twin + the kitty
	// passthrough): the stream SPLITTER (browser_lane_kitty.go) extracts
	// the child's kitty graphics APCs into the image store, and every
	// other byte lands in BOTH the raw scrollback and the live screen
	// model (the base64 payloads NEVER reach the grid — they re-emit to
	// the outer terminal at RegionView time instead); the PTY going away
	// flips Alive even if Wait is still reaping.
	split := newKittyStream(io.MultiWriter(s.sb, s.grid), s.grid, s.images)
	s.split = split
	go func() {
		_, _ = io.Copy(split, s.mf)
		s.mu.Lock()
		s.alive = false
		s.mu.Unlock()
	}()
	// the waiter (term.Session.waiter's twin): reap + pin the exit code
	// and the frozen lifetime (the early-exit rule's input).
	go func() {
		err := s.cmd.Wait()
		code := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = ee.ExitCode()
			} else {
				code = -1
			}
		}
		s.mu.Lock()
		s.alive, s.exited, s.code = false, true, code
		s.life = time.Since(s.started)
		s.mu.Unlock()
	}()
	return s, nil
}

// spawnZenbuSession — the spawn seam (terminal.go's spawnTermSession
// precedent): newZenbuSession by default (a REAL PTY + REAL binary),
// swapped by tests so no test ever owns a real child. The uishot harness
// keeps the REAL spawn against a fixture-PATH fake (the --openurl
// precedent: real exec, hermetic binary).
var spawnZenbuSession = func(url string, cols, rows int) (zenbuSess, error) {
	return newZenbuSession(url, cols, rows)
}

// Alive reports whether the child is believed running.
func (s *ZenbuSession) Alive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

// Exited reports whether cmd.Wait has returned (fully reaped).
func (s *ZenbuSession) Exited() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exited
}

// ExitCode returns the child's exit code; -1 while alive or unknown.
func (s *ZenbuSession) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.code
}

// URL is the page the child was opened with.
func (s *ZenbuSession) URL() string { return s.url }

// Lifetime — frozen at reap (the early-exit rule's clock), start→now
// while alive.
func (s *ZenbuSession) Lifetime() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exited {
		return s.life
	}
	return time.Since(s.started)
}

// Pid is the child's process id (also its pgroup id — kill -Pid works).
func (s *ZenbuSession) Pid() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return -1
	}
	return s.cmd.Process.Pid
}

// Write forwards raw key bytes to the child's PTY master (the capture
// toggle's pipe — the tab forwards term.go's keyToBytes matrix while
// ctrl+space capture is opted in).
func (s *ZenbuSession) Write(p []byte) (int, error) {
	if !s.Alive() {
		return 0, fmt.Errorf("zenbu: write to dead session")
	}
	return s.mf.Write(p)
}

// Resize hands the PTY a new window size (SIGWINCH to the foreground
// process group is implicit on darwin/linux) and reshapes the screen
// model first (term.Session.Resize's exact order). Every live kitty
// placement's geometry is now stale: the deletes queue (the next
// RegionView flushes them in-frame) and the child repaints fresh frames
// into the new box, re-placing from scratch — and the store's body box
// moves WITH the resize, so those fresh frames emit the NEW c=/r= dims
// immediately (FIX B). While FROZEN the placements are deliberately
// KEPT (they ARE Resume's instant repaint — retiring them would blank
// the thawed pane): the PTY size + body box still move (kernel state;
// SIGWINCH pends and delivers at SIGCONT) and the waking child repaints
// the new box fresh under the same office ids.
func (s *ZenbuSession) Resize(cols, rows int) error {
	if cols < 2 {
		cols = 2
	}
	if rows < 1 {
		rows = 1
	}
	s.grid.SetSize(cols, rows)
	s.images.setBodyBox(cols, rows)
	if !s.Frozen() {
		s.images.retirePlacements()
	}
	return pty.Setsize(s.mf, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

// Size reports the current PTY geometry.
func (s *ZenbuSession) Size() (cols, rows int) {
	return s.grid.Cols(), s.grid.Rows()
}

// Grid exposes the live screen model (the pane's paint source).
func (s *ZenbuSession) Grid() *term.Grid { return s.grid }

// Scrollback exposes the retained byte stream.
func (s *ZenbuSession) Scrollback() *term.Scrollback { return s.sb }

// Freeze — the keep-alive suspend (the pane flipped to the floor, the
// q/esc leave): the splitter PARKS FIRST (a consumption pause — the
// pending APC tail + the open chunk chain are PRESERVED in the
// splitter's buffers; any byte the still-running child writes before
// the stop takes effect lands in the pending buffer, NEVER the grid),
// the frame-splice registry CLEARS so the wrapper's next flush deletes
// the terminal-side image (the floor never shows the page), and ONLY
// THEN the child process group is SIGSTOPped (the CPU freezes; the
// PTY's kernel buffer backpressures a mid-write child naturally). The
// park-before-signal order closes the leak window: a mid-chain freeze
// keeps the chain's HEAD, the kernel PTY buffer + the parked pending
// hold the in-flight bytes, and Unfreeze's thawed tail COMPLETES the
// chain into a valid frame — zero base64 can reach the grid (resetting
// at this boundary was the production freeze-leak). The image store
// RETAINS the latest joined frame — it IS Resume's instant repaint. The
// PID never changes. Idempotent; an already-exited child skips the
// signal (Poll owns the fallback); a closed session no-ops.
func (s *ZenbuSession) Freeze() error {
	s.mu.Lock()
	if s.closed || s.suspended {
		s.mu.Unlock()
		return nil
	}
	s.suspended = true
	pid := -1
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	exited := s.exited
	s.mu.Unlock()

	// park BEFORE the signal: the consumption pause must be up first so
	// a byte written while the SIGSTOP is still taking effect lands in
	// the pending buffer (an unparked splitter would scan it, and a
	// post-signal park would reopen exactly the window the ordering
	// test pins shut).
	if s.split != nil {
		s.split.park()
	}
	ZenbuRegistry().Clear()
	if !exited && pid > 0 {
		return zenbuGroupSignal(-pid, zenbuSigStop)
	}
	return nil
}

// Unfreeze — the keep-alive resume (the pane flipped back): the
// splitter unparks (its PRESERVED pending drains on the spot — a
// fully-arrived in-flight chain commits before the child emits a byte;
// a partial APC holds) and the SAME process group is SIGCONTed — the
// thawed child's tail COMPLETES the pending chain into a valid frame
// (NO respawn, NO reload, the PID is the one Freeze pinned). A child
// found DEAD (murdered while frozen) can never complete its chain: the
// splitter is RESET first (the pending tail + open chain drop log-once
// — the kill-path discipline, ahead of Poll's fallback latch), the gate
// STAYS PARKED (the reader's last kernel-buffer bytes racing the reap
// buffer unscanned until Close's reset drops them — a dead child's
// chainless tail can NEVER flush), and no signal rides. The store's retained frame never rides this call: the
// app's next Frame() republishes it to the frame-splice registry and
// the wrapper re-emits it on the very next flush (the member sees the
// last painted page BEFORE the thawed child emits anything new).
// Idempotent; a closed session no-ops.
func (s *ZenbuSession) Unfreeze() error {
	s.mu.Lock()
	if s.closed || !s.suspended {
		s.mu.Unlock()
		return nil
	}
	s.suspended = false
	pid := -1
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	alive := s.alive && !s.exited
	s.mu.Unlock()

	if s.split != nil {
		if !alive {
			// the frozen child was MURDERED: the in-flight chain's tail
			// never comes — reset the splitter (the kill-path discipline)
			// and KEEP THE GATE PARKED: the reader's final drained bytes
			// (the kernel buffer racing the reap) buffer UNSCANNED until
			// Close's reset drops them — a dead child's chainless tail
			// can never scan its way to the grid. (Poll drops the session
			// behind this; nothing ever thaws a dead lane.)
			s.split.reset()
			return nil
		}
		s.split.unpark()
	}
	if alive && pid > 0 {
		return zenbuGroupSignal(-pid, zenbuSigCont)
	}
	return nil
}

// Frozen — the keep-alive posture: the child is SIGSTOPped (alive, PID
// unchanged), the splitter parked, the image store retained.
func (s *ZenbuSession) Frozen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.suspended && !s.closed
}

// Close group-kills the child (the pty-spawned session leader: -pid takes
// the WHOLE group — chromium helpers included), closes the master so the
// reader unblocks, and waits BOUNDED by zenbuKillGrace for the reap
// (opencode.go's stopKillGrace contract: the teardown path never commutes
// with a wedged child; the killing process exit reaps the rest).
// Idempotent — safe at office quit. The terminal's live kitty placements
// die WITH the session: the office-side deletes flush DIRECTLY through
// the zenbuEmit seam (no frame will paint this pane again — the images
// must not linger over the floor or past the office exit), and the
// frame-splice registry CLEARS on the spot — the QUIT path renders no
// further Frame to publish the empty state, so the renderer's final
// flush (the alt-screen exit) must already find the registry empty. A
// FROZEN child is SIGCONTed first (insurance: SIGKILL already terminates
// a stopped process on darwin/linux, but no kernel queues the KILL
// behind a CONT this way) — the freeze never becomes a reap wedge.
func (s *ZenbuSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	wasFrozen := s.suspended
	s.suspended = false
	pid := -1
	if s.cmd != nil && s.cmd.Process != nil {
		pid = s.cmd.Process.Pid
	}
	s.mu.Unlock()

	// FIX C: the splitter's pending APC tail + open chunk chain are
	// DISCARDED on the spot — a child dying mid-frame (killSess /
	// respawn / frozen-quit churn) leaves both behind, and they must
	// never flush to the grid or the scrollback. (The FREEZE path is
	// the opposite posture — park PRESERVES them; only the kill paths
	// reset.)
	if s.split != nil {
		s.split.reset()
	}
	if wasFrozen && pid > 0 {
		_ = zenbuGroupSignal(-pid, zenbuSigCont) // thaw into the KILL (see the doc)
	}
	if s.images != nil {
		if frames := s.images.dropAll(); frames != "" {
			zenbuEmit(frames)
		}
	}
	ZenbuRegistry().Clear()
	if pid > 0 {
		_ = zenbuGroupSignal(-pid, zenbuSigKill)
	}
	if s.mf != nil {
		_ = s.mf.Close()
	}
	for deadline := time.Now().Add(zenbuKillGrace); time.Now().Before(deadline); {
		if s.Exited() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// -------------------------------------------------------------------
// the frame-splice read seam (browser_lane_kitty.go + zenbu_frame.go)
// -------------------------------------------------------------------

// zenbuImageSurface — the premium session's kitty-passthrough read seam:
// the REAL *ZenbuSession implements it; the controller suite's fakes
// deliberately don't (their RegionView paint stays the text-only path —
// the existing chrome/fallback tests are untouched by the splice, and the
// controller's FrameState reads (nil, nil) for them).
type zenbuImageSurface interface {
	imagePlacements() []zenbuPlacement
	drainImageDeleteIDs() []uint32
}

// imagePlacements — the live kitty placements for the frame-splice
// registry (zenbu_frame.go's FrameState).
func (s *ZenbuSession) imagePlacements() []zenbuPlacement { return s.images.placements() }

// drainImageDeleteIDs — the queued office-side delete ids (child a=d,
// id-replacement, resize-retire) for the registry publish's per-render
// drain (the queue's OTHER drain is dropAll at Close).
func (s *ZenbuSession) drainImageDeleteIDs() []uint32 { return s.images.drainPendingIDs() }

// -------------------------------------------------------------------
// the lane controller (the browser tab's drive surface)
// -------------------------------------------------------------------

// ZenbuFallbackNoteFmt — the EXACT dim note the text lane wears when the
// premium child fails (non-zero exit, early exit, or a spawn failure
// wearing 127 — the POSIX not-found code).
const ZenbuFallbackNoteFmt = "zenbu exited (%d) — falling back to text mode"

// BrowserLaneController — the browser tab's lane state machine. The tab
// owns the text-mode viewer and the keys; this owns the lane resolve
// (memoized), the embedded zenbu child (spawn/kill/reap), the failure
// fallback (note + text latch per url), and the URL state that must
// survive the fallback untouched (current + visited history — the
// navigation ring's source; this layer NEVER fetches).
type BrowserLaneController struct {
	resolver *BrowserLaneResolver
	sess     zenbuSess // nil while the text lane paints
	url      string    // the CURRENT url (never cleared by a fallback)
	history  []string  // every successfully-opened url, in open order
	failed   map[string]bool
	note     string // the dim fallback note ("" while premium is healthy)
	cols     int
	rows     int
	closed   bool
}

// NewBrowserLaneController — cols×rows is the PANE's full box (the strip
// row and the note row are reserved; the embedded PTY gets the rest). The
// lane resolve is pinned HERE — env+PATH are read ONCE at pane-creation
// time (never per frame, never per open), so the pane's lane story (badge
// AND the text-lane hint row) is stable for the pane's whole life.
func NewBrowserLaneController(cols, rows int) *BrowserLaneController {
	c := &BrowserLaneController{
		resolver: NewBrowserLaneResolver(),
		cols:     cols,
		rows:     rows,
	}
	c.resolver.Lane() // the pane-creation pin (the memo's one honest read)
	return c
}

// bodyH — the embedded PTY's row count (strip + note reserved).
func (c *BrowserLaneController) bodyH() int {
	if c.rows-2 < 1 {
		return 1
	}
	return c.rows - 2
}

// Lane — the boot-memoized resolve (the tab shows the badge from
// PremiumActive; the resolver's pick alone does not paint chrome).
func (c *BrowserLaneController) Lane() BrowserLane { return c.resolver.Lane() }

// Verdict — the pane-creation-memoized resolve in full (lane + reason
// class + armed kill-switch spelling): the text lane's "why" — the pane
// paints the dim hint row from it (browser.go's laneHintRow).
func (c *BrowserLaneController) Verdict() (BrowserLane, BrowserLaneReason, string) {
	return c.resolver.Verdict()
}

// OpenURL drives the browser tab to url: any live premium child is
// SIGKILLed + reaped FIRST (a fresh /open = kill + spawn fresh), the url
// pins current + history, and the premium lane spawns the embed UNLESS
// the resolve missed (text lane) or THIS url already fell back once (the
// no-flap latch). A spawn failure falls back immediately with the 127
// note. The text lane itself renders nothing here — the tab's viewer
// paints CurrentURL; this call never fetches.
func (c *BrowserLaneController) OpenURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return errors.New("browser lane: empty url")
	}
	if c.closed {
		return errors.New("browser lane: controller closed")
	}
	c.killSess()
	c.note = ""
	c.url = url
	c.history = append(c.history, url)
	if c.resolver.Lane() != BrowserLaneZenbu || c.failed[url] {
		return nil // the universal text lane renders CurrentURL
	}
	sess, err := spawnZenbuSession(url, c.cols, c.bodyH())
	if err != nil {
		if c.failed == nil {
			c.failed = map[string]bool{}
		}
		c.failed[url] = true
		c.note = fmt.Sprintf(ZenbuFallbackNoteFmt, 127)
		return nil // a lane miss is a fallback, never a fatal
	}
	c.sess = sess
	return nil
}

// Poll — the tab's per-frame/per-msg check: observes a DEAD premium child
// and lands the exit contract. A non-zero exit or an early exit
// (< zenbuEarlyExit) latches the text lane for this url with the dim
// note; a clean long-run exit (the member quit the embedded browser)
// returns to the text location bar silently. Either way the URL state
// persists. changed=true means the pane must repaint (the session dropped
// THIS call).
func (c *BrowserLaneController) Poll() (changed bool) {
	s := c.sess
	if s == nil || !s.Exited() {
		return false
	}
	code := s.ExitCode()
	early := s.Lifetime() < zenbuEarlyExit
	c.sess = nil
	_ = s.Close() // reaped already; seals the idempotent latch
	if code != 0 || early {
		if c.failed == nil {
			c.failed = map[string]bool{}
		}
		c.failed[c.url] = true
		c.note = fmt.Sprintf(ZenbuFallbackNoteFmt, code)
	}
	return true
}

// Suspend — the pane switched away (ctrl+b to the floor, the pane's
// q/esc): the premium child FREEZES in place (SIGSTOP, keep-alive — the
// member's "always shown" ruling): the terminal-side image deletes ride
// the registry clear → the wrapper's a=d, the splitter parks (a
// consumption pause — the pending APC tail + open chunk chain are
// PRESERVED for the thaw; zero store mutations while suspended), the
// image store RETAINS the latest joined frame for Resume's instant
// repaint, the PID never changes, and the URL state keeps. Silent (not
// a failure — no note). Idempotent.
func (c *BrowserLaneController) Suspend() {
	if c.sess == nil {
		return
	}
	// an already-dead child's ESRCH is swallowed: the next Poll owns
	// the fallback contract for it.
	_ = c.sess.Freeze()
}

// Resume — the pane became active again: the SAME child THAWS (SIGCONT
// — NO respawn, NO reload, NO blank beyond one frame flush; the store's
// retained frame re-emits through the frame-splice wrapper on the next
// flush, before the child emits anything new). A missing session (the
// child died while frozen and Poll already dropped it) takes the old
// respawn path: re-spawn the premium embed for the CURRENT url when the
// lane still resolves premium and the url never fell back (a fell-back
// url stays text — no flap). No new history entry.
func (c *BrowserLaneController) Resume() {
	if c.closed || c.url == "" {
		return
	}
	if c.sess != nil {
		_ = c.sess.Unfreeze() // keep-alive: thaw the SAME child
		return
	}
	if c.resolver.Lane() != BrowserLaneZenbu || c.failed[c.url] {
		return
	}
	sess, err := spawnZenbuSession(c.url, c.cols, c.bodyH())
	if err != nil {
		if c.failed == nil {
			c.failed = map[string]bool{}
		}
		c.failed[c.url] = true
		c.note = fmt.Sprintf(ZenbuFallbackNoteFmt, 127)
		return
	}
	c.sess = sess
}

// Close — browser.Close / office shutdown: kill + reap, seal the
// controller. Idempotent; NEVER leaks a child past the office exit.
func (c *BrowserLaneController) Close() {
	if c.closed {
		return
	}
	c.closed = true
	c.killSess()
}

// killSess — the shared kill leg: group-SIGKILL + bounded reap (the
// session's own Close contract), drop the handle.
func (c *BrowserLaneController) killSess() {
	if c.sess == nil {
		return
	}
	_ = c.sess.Close()
	c.sess = nil
}

// Session — the live embedded session for the pane's paint path (nil
// while the text lane paints). The tab drives it through TermPanel's
// termSess contract verbatim.
func (c *BrowserLaneController) Session() termSess { return c.sess }

// PremiumActive — the premium embed is live AND painting RIGHT NOW (the
// strip + " zenbu " badge paint; otherwise the text location bar +
// " text "). A FROZEN child is NOT active: the floor owns the slot, the
// registry stays clear, and no zenbu chrome paints until Resume thaws
// the same session (Suspended reads the keep-alive posture).
func (c *BrowserLaneController) PremiumActive() bool { return c.sess != nil && !c.sess.Frozen() }

// Suspended — the keep-alive posture: the premium child is FROZEN
// (SIGSTOPped, alive, the PID unchanged, the image store retained)
// behind the floor. The harness proves the flip cycle through this.
func (c *BrowserLaneController) Suspended() bool { return c.sess != nil && c.sess.Frozen() }

// Note — the dim fallback note ("" while premium is healthy or the last
// exit was clean). Rendered dim by the tab / RegionView.
func (c *BrowserLaneController) Note() string { return c.note }

// CurrentURL — the pane's current page (survives every fallback; the
// text lane renders THIS, never a re-fetch).
func (c *BrowserLaneController) CurrentURL() string { return c.url }

// VisitedURLs — the opened history in open order (the navigation ring's
// source; a fallback never truncates it).
func (c *BrowserLaneController) VisitedURLs() []string {
	out := make([]string, len(c.history))
	copy(out, c.history)
	return out
}

// SetSize — the pane resized: the strip/note rows stay reserved, the
// embedded PTY takes the SIGWINCH.
func (c *BrowserLaneController) SetSize(cols, rows int) {
	c.cols, c.rows = cols, rows
	if c.sess != nil {
		_ = c.sess.Resize(cols, c.bodyH())
	}
}

// -------------------------------------------------------------------
// the pane chrome (the tab's strip/badge; the proof region's painter)
// -------------------------------------------------------------------

// ZenbuStripLine — the top-only location strip while the premium child
// runs (the EXACT contract row).
func ZenbuStripLine(url string) string { return "▸ zenbu terminal-browser · " + url }

// BrowserLaneBadge — the pane's lane indicator: " zenbu " (the active-tab
// accent) while a premium session runs, " text " (inactive gray) in the
// universal text lane.
func BrowserLaneBadge(premiumActive bool) string {
	if premiumActive {
		return chrome.TabActive.Render(" zenbu ")
	}
	return chrome.TabInactive.Render(" text ")
}

// -------------------------------------------------------------------
// the pane's keep-alive seams (the LaneFrameState precedent: *Browser
// methods placed with the lane code — the app drives the flip cycle
// through these, the harness proves it)
// -------------------------------------------------------------------

// LaneSuspended — the pane's premium child is FROZEN behind the floor
// (the keep-alive posture: SIGSTOPped, alive, PID unchanged, the image
// store retained).
func (b *Browser) LaneSuspended() bool {
	return b.lane != nil && b.lane.Suspended()
}

// LaneSessionPid — the live OR frozen premium child's process id (-1
// while the text lane paints or the session is a test fake): the
// keep-alive proof's PID-stability read (the flip must never respawn).
func (b *Browser) LaneSessionPid() int {
	if b.lane == nil || b.lane.Session() == nil {
		return -1
	}
	zs, ok := b.lane.Session().(*ZenbuSession)
	if !ok {
		return -1
	}
	return zs.Pid()
}

// RegionView — the browser tab's pane region, lane-aware, exactly cols×
// rows cells: row 0 is the lane badge + the top strip (premium: the
// zenbu strip; text: the location bar stand-in "▸ <url>" — Dev A's
// viewer swaps its real location bar in behind this same seam), the body
// is the embedded PTY's screen model while premium runs (gridRowString's
// SGR run paint — TermPanel's exact body path, caret parked) or the
// caller's text-mode rows otherwise, and the last row carries the dim
// fallback note (blank while healthy). textRows are the text lane's OWN
// render (the viewer's wrapped body); they are clipped/padded into the
// body window verbatim.
func (c *BrowserLaneController) RegionView(textRows []string) string {
	var b strings.Builder
	if c.PremiumActive() {
		b.WriteString(BrowserLaneBadge(true) + " " + fitPlain(ZenbuStripLine(c.url), c.cols-8))
	} else {
		b.WriteString(BrowserLaneBadge(false) + " " + fitPlain("▸ "+c.url, c.cols-8))
	}
	body := c.bodyH()
	if c.PremiumActive() {
		// PURE TEXT: the embedded PTY's screen model paints exactly like
		// TermPanel's body path. The kitty images NEVER ride this string —
		// bubbletea's cell renderer eats zero-width sequences (the wave-80
		// in-View splice never painted; it only bloated the differ) — they
		// reach the terminal through the frame-splice wrapper instead
		// (zenbu_frame.go: the Model's per-Frame registry publish + the
		// tea.WithOutput seam's post-flush emission).
		for y, row := range c.sess.Grid().Render() {
			if y >= body {
				break
			}
			b.WriteString("\n" + gridRowString(row, -1))
		}
	} else {
		for y := 0; y < body; y++ {
			row := ""
			if y < len(textRows) {
				row = textRows[y]
			}
			b.WriteString("\n" + fitPlain(ansi.Strip(row), c.cols))
		}
	}
	b.WriteString("\n" + chrome.DimText.Render(fitPlain(c.note, c.cols)))
	return b.String()
}
