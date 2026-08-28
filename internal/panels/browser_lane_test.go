// browser_lane_test.go — the premium browser lane's contract suite:
// the resolve matrix (pure, shell-out-free), the per-boot memoization,
// the failure-fallback state machine (fake session seam — no test owns a
// real child), the URL-state persistence across fallback, the suspend/
// resume/close lifecycle, the RegionView chrome, and ONE real-spawn reap
// test against a fixture-PATH fake binary (the --openurl precedent: real
// PTY, real exec, hermetic binary) proving no child leaks past a tab
// switch or office shutdown.
package panels

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/term"
)

// laneEnv builds the injected env read for the pure resolve matrix.
func laneEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func lookFound(string) (string, error)   { return "/fixture/terminal-browser", nil }
func lookMissing(string) (string, error) { return "", errors.New("exec: no command") }

// TestBrowserLaneResolveMatrix — the pure lane table: kitty-found,
// kitty-missing (PATH-resolution-failure), terminal-mismatch, kill-switch
// (both spellings). Every miss is the universal text lane.
func TestBrowserLaneResolveMatrix(t *testing.T) {
	ghostty := map[string]string{"TERM_PROGRAM": "ghostty", "TERM": "xterm-256color"}
	cases := []struct {
		name string
		env  map[string]string
		look func(string) (string, error)
		want BrowserLane
	}{
		{"kitty-found (ghostty + binary)", ghostty, lookFound, BrowserLaneZenbu},
		{"kitty-found (TERM_PROGRAM=kitty)", map[string]string{"TERM_PROGRAM": "kitty"}, lookFound, BrowserLaneZenbu},
		{"kitty-found (KITTY_WINDOW_ID beats TERM_PROGRAM)", map[string]string{"KITTY_WINDOW_ID": "7", "TERM_PROGRAM": "iTerm.app"}, lookFound, BrowserLaneZenbu},
		{"kitty-missing binary (PATH-resolution-failure)", ghostty, lookMissing, BrowserLaneText},
		{"terminal-mismatch: iTerm2 is NOT this lane", map[string]string{"TERM_PROGRAM": "iTerm.app"}, lookFound, BrowserLaneText},
		{"terminal-mismatch: WezTerm resolves the iterm image lane, never zenbu", map[string]string{"TERM_PROGRAM": "WezTerm"}, lookFound, BrowserLaneText},
		{"terminal-mismatch: plain xterm", map[string]string{"TERM": "xterm-256color"}, lookFound, BrowserLaneText},
		{"terminal-mismatch: tmux folds out conservatively", map[string]string{"TERM_PROGRAM": "ghostty", "TMUX": "/tmp/tmux-1000/default,1,0"}, lookFound, BrowserLaneText},
		{"kill-switch: THEBORINGOFFICE_TERMINAL_BROWSER_OFF=1", map[string]string{"TERM_PROGRAM": "ghostty", BrowserLaneOffEnv: "1"}, lookFound, BrowserLaneText},
		{"kill-switch: THEBORINGOFFICE_NO_TERMINAL_BROWSER=1 (wave-70 spelling)", map[string]string{"TERM_PROGRAM": "ghostty", TerminalBrowserOffEnv: "1"}, lookFound, BrowserLaneText},
		{"kill-switch: '0' is NOT armed", map[string]string{"TERM_PROGRAM": "ghostty", BrowserLaneOffEnv: "0"}, lookFound, BrowserLaneZenbu},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveBrowserLaneFrom(laneEnv(tc.env), tc.look); got != tc.want {
				t.Fatalf("ResolveBrowserLaneFrom = %s, want %s", got, tc.want)
			}
		})
	}
}

// pinKittyEnv — the hermetic kitty-capable host stub for the live-read
// tests (uishot's stubTermEnv checklist: every detect-layer input owned).
func pinKittyEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM_VERSION", "")
	t.Setenv("WEZTERM_UNIX_SOCKET", "")
	t.Setenv("VSCODE_PID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv(BrowserLaneOffEnv, "")
	t.Setenv(TerminalBrowserOffEnv, "")
	old := zenbuLookPath
	zenbuLookPath = lookFound
	t.Cleanup(func() { zenbuLookPath = old })
}

// TestBrowserLaneResolverMemo — the per-boot memoization: ONE honest read,
// pinned until Reset (the images-lane memo idiom).
func TestBrowserLaneResolverMemo(t *testing.T) {
	pinKittyEnv(t)
	r := NewBrowserLaneResolver()
	if got := r.Lane(); got != BrowserLaneZenbu {
		t.Fatalf("first read resolves zenbu on the kitty stub, got %s", got)
	}
	t.Setenv("TERM_PROGRAM", "iTerm.app") // the host flips AFTER the read…
	if got := r.Lane(); got != BrowserLaneZenbu {
		t.Fatalf("…but the memoized lane stays pinned, got %s", got)
	}
	r.Reset()
	if got := r.Lane(); got != BrowserLaneText {
		t.Fatalf("a Reset resolver re-reads honestly: got %s, want text", got)
	}
}

// -------------------------------------------------------------------
// the fake session (the controller's zenbuSess seam — no real child)
// -------------------------------------------------------------------

type fakeZenbuSess struct {
	grid   *term.Grid
	sb     *term.Scrollback
	url    string
	cols   int
	rows   int
	alive  bool
	exited bool
	code   int
	life   time.Duration
	closed bool
	writes int
	frozen bool // the keep-alive posture (Freeze/Unfreeze flip it)
}

func newFakeZenbuSess(url string, cols, rows int) *fakeZenbuSess {
	return &fakeZenbuSess{
		grid: term.NewGrid(cols, rows), sb: term.NewScrollback(0),
		url: url, cols: cols, rows: rows, alive: true, code: -1,
	}
}

func (f *fakeZenbuSess) Alive() bool                  { return f.alive }
func (f *fakeZenbuSess) Exited() bool                 { return f.exited }
func (f *fakeZenbuSess) ExitCode() int                { return f.code }
func (f *fakeZenbuSess) Lifetime() time.Duration      { return f.life }
func (f *fakeZenbuSess) URL() string                  { return f.url }
func (f *fakeZenbuSess) Grid() *term.Grid             { return f.grid }
func (f *fakeZenbuSess) Scrollback() *term.Scrollback { return f.sb }
func (f *fakeZenbuSess) Size() (int, int)             { return f.cols, f.rows }
func (f *fakeZenbuSess) Close() error                 { f.closed, f.alive = true, false; return nil }
func (f *fakeZenbuSess) Write(p []byte) (int, error)  { f.writes++; return len(p), nil }
func (f *fakeZenbuSess) Freeze() error                { f.frozen = true; return nil }
func (f *fakeZenbuSess) Unfreeze() error              { f.frozen = false; return nil }
func (f *fakeZenbuSess) Frozen() bool                 { return f.frozen }
func (f *fakeZenbuSess) Resize(c, r int) error {
	f.cols, f.rows = c, r
	f.grid.SetSize(c, r)
	return nil
}

// fakeSpawnPins swaps the spawn seam for a factory capturing every fake
// it mints (the spawn count IS the no-flap latch's evidence).
func fakeSpawnPins(t *testing.T) *[]*fakeZenbuSess {
	t.Helper()
	var made []*fakeZenbuSess
	old := spawnZenbuSession
	spawnZenbuSession = func(url string, cols, rows int) (zenbuSess, error) {
		f := newFakeZenbuSess(url, cols, rows)
		made = append(made, f)
		return f, nil
	}
	t.Cleanup(func() { spawnZenbuSession = old })
	return &made
}

// fakeSpawnPinsCount swaps the spawn seam for the REAL factory wrapped in
// a counter (a test needing a REAL child AND the spawn-count evidence —
// the no-respawn latch over the genuine PTY seam).
func fakeSpawnPinsCount(t *testing.T) *int {
	t.Helper()
	count := 0
	old := spawnZenbuSession
	spawnZenbuSession = func(url string, cols, rows int) (zenbuSess, error) {
		count++
		return newZenbuSession(url, cols, rows)
	}
	t.Cleanup(func() { spawnZenbuSession = old })
	return &count
}

// TestBrowserLaneFallbackPreservesURL — the exit contract: non-zero exit
// AND early-exit (<300ms, even code 0) both latch the text lane for THAT
// url with the exact dim note; the URL state (current + history) survives
// untouched; a NEW url retries premium; a clean long-run exit drops to
// the text bar silently (no note, no latch).
func TestBrowserLaneFallbackPreservesURL(t *testing.T) {
	pinKittyEnv(t)
	made := fakeSpawnPins(t)
	c := NewBrowserLaneController(64, 16)

	const u1, u2, u3 = "https://a.dev/1", "https://a.dev/2", "https://a.dev/3"
	if err := c.OpenURL(u1); err != nil {
		t.Fatalf("open u1: %v", err)
	}
	if !c.PremiumActive() || len(*made) != 1 {
		t.Fatalf("u1 spawns the premium embed: active=%v made=%d", c.PremiumActive(), len(*made))
	}

	// leg 1 — NON-ZERO exit (long-lived): fallback + note + URL state kept.
	f1 := (*made)[0]
	f1.exited, f1.code, f1.life = true, 1, 2*time.Second
	if !c.Poll() {
		t.Fatal("Poll observes the dead child")
	}
	if got, want := c.Note(), "zenbu exited (1) — falling back to text mode"; got != want {
		t.Fatalf("the non-zero note is exact: got %q want %q", got, want)
	}
	if c.Session() != nil || c.PremiumActive() {
		t.Fatal("the fallback drops the embed (text lane paints)")
	}
	if c.CurrentURL() != u1 {
		t.Fatalf("the current url survives the fallback: got %q", c.CurrentURL())
	}
	if got := strings.Join(c.VisitedURLs(), ","); got != u1 {
		t.Fatalf("the visited history survives the fallback: got %q", got)
	}

	// leg 2 — EARLY exit with code 0 (<300ms): the same fallback class.
	if err := c.OpenURL(u2); err != nil {
		t.Fatalf("open u2: %v", err)
	}
	if len(*made) != 2 {
		t.Fatalf("a NEW url retries the premium lane: made=%d", len(*made))
	}
	f2 := (*made)[1]
	f2.exited, f2.code, f2.life = true, 0, 100*time.Millisecond
	if !c.Poll() {
		t.Fatal("Poll observes the early death")
	}
	if got, want := c.Note(), "zenbu exited (0) — falling back to text mode"; got != want {
		t.Fatalf("the early-exit note is exact: got %q want %q", got, want)
	}

	// the no-flap latch: re-opening a fell-back url never re-spawns.
	if err := c.OpenURL(u2); err != nil {
		t.Fatalf("re-open u2: %v", err)
	}
	if len(*made) != 2 || c.PremiumActive() {
		t.Fatalf("a fell-back url stays text (no flap): made=%d active=%v", len(*made), c.PremiumActive())
	}
	if c.CurrentURL() != u2 {
		t.Fatalf("the text lane keeps the url: got %q", c.CurrentURL())
	}

	// leg 3 — CLEAN long-run exit: text bar silently, NO latch.
	if err := c.OpenURL(u3); err != nil {
		t.Fatalf("open u3: %v", err)
	}
	f3 := (*made)[2]
	f3.exited, f3.code, f3.life = true, 0, 5*time.Second
	if !c.Poll() {
		t.Fatal("Poll observes the clean exit")
	}
	if c.Note() != "" {
		t.Fatalf("a clean exit is silent, got note %q", c.Note())
	}
	if err := c.OpenURL(u3); err != nil {
		t.Fatalf("re-open u3: %v", err)
	}
	if len(*made) != 4 {
		t.Fatalf("a cleanly-exited url is NOT latched — premium retries: made=%d", len(*made))
	}
	if got := strings.Join(c.VisitedURLs(), ","); got != u1+","+u2+","+u2+","+u3+","+u3 {
		t.Fatalf("the ring history appends every open in order: got %q", got)
	}
}

// TestBrowserLaneSpawnFailure — a vanished binary between probe and exec
// falls back immediately wearing 127 (the POSIX not-found code).
func TestBrowserLaneSpawnFailure(t *testing.T) {
	pinKittyEnv(t)
	old := spawnZenbuSession
	spawnZenbuSession = func(string, int, int) (zenbuSess, error) {
		return nil, errors.New("zenbu: spawn terminal-browser: executable file not found in $PATH")
	}
	t.Cleanup(func() { spawnZenbuSession = old })
	c := NewBrowserLaneController(64, 16)
	if err := c.OpenURL("https://a.dev/x"); err != nil {
		t.Fatalf("a spawn failure is a fallback, never a fatal: %v", err)
	}
	if c.PremiumActive() {
		t.Fatal("no embed lives past a spawn failure")
	}
	if got, want := c.Note(), "zenbu exited (127) — falling back to text mode"; got != want {
		t.Fatalf("the spawn-failure note wears 127: got %q want %q", got, want)
	}
	if c.CurrentURL() != "https://a.dev/x" {
		t.Fatalf("the url survives: %q", c.CurrentURL())
	}
}

// TestBrowserLaneSuspendResumeClose — the KEEP-ALIVE lifecycle: Suspend
// FREEZES the embed (tab switch, silent — the same fake stays live, no
// Close, no respawn), Resume THAWS the SAME session (made stays 1 — the
// PID-stability contract's fake-level shape), PremiumActive hides while
// frozen (the floor never shows zenbu chrome) and returns on thaw,
// Close seals (office shutdown) and is idempotent.
func TestBrowserLaneSuspendResumeClose(t *testing.T) {
	pinKittyEnv(t)
	made := fakeSpawnPins(t)
	c := NewBrowserLaneController(64, 16)
	const u = "https://a.dev/live"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	f1 := (*made)[0]

	c.Suspend() // the pane switched away — the child FREEZES (keep-alive)
	if !f1.frozen {
		t.Fatal("Suspend freezes the embedded child (SIGSTOP, keep-alive)")
	}
	if f1.closed {
		t.Fatal("Suspend must NOT kill the child (keep-alive — the page is always shown)")
	}
	if c.PremiumActive() {
		t.Fatal("a frozen child is not premium-ACTIVE (the floor shows no zenbu chrome)")
	}
	if !c.Suspended() {
		t.Fatal("the keep-alive posture reads Suspended")
	}
	if c.Note() != "" {
		t.Fatalf("a tab switch is not a failure — no note: %q", c.Note())
	}
	c.Suspend() // idempotent freeze
	if !f1.frozen || f1.closed {
		t.Fatal("a second Suspend keeps the freeze (no kill, no error)")
	}

	c.Resume() // the pane is back — the SAME child thaws (NO respawn)
	if len(*made) != 1 {
		t.Fatalf("Resume thaws the SAME child — no respawn: made=%d, want 1", len(*made))
	}
	if f1.frozen || !c.PremiumActive() || c.Suspended() {
		t.Fatalf("Resume thaws the embed: frozen=%v active=%v suspended=%v", f1.frozen, c.PremiumActive(), c.Suspended())
	}
	if got := strings.Join(c.VisitedURLs(), ","); got != u {
		t.Fatalf("Resume adds NO history entry: got %q", got)
	}
	c.Resume() // idempotent thaw (no respawn on a live session)
	if len(*made) != 1 {
		t.Fatalf("a second Resume never respawns: made=%d", len(*made))
	}

	// the respawn path survives for a MISSING session (the frozen child
	// died and Poll dropped it — the old Resume contract's leg).
	f1.exited, f1.code, f1.life = true, 0, 5*time.Second // a clean long-run exit
	if !c.Poll() {
		t.Fatal("Poll observes the dropped child")
	}
	c.Resume()
	if len(*made) != 2 || !c.PremiumActive() {
		t.Fatalf("a cleanly-dropped session respawns on Resume: made=%d active=%v", len(*made), c.PremiumActive())
	}

	c.Close() // office shutdown
	if !(*made)[1].closed {
		t.Fatal("Close reaps the live embed")
	}
	c.Close() // idempotent
	if err := c.OpenURL("https://a.dev/late"); err == nil {
		t.Fatal("a closed controller refuses new opens")
	}
}

// TestBrowserLaneRegionView — the pane chrome: premium paints the badge +
// the exact zenbu strip + the embedded grid's rows; the fallback frame
// paints the " text " badge, the text-mode body, and the dim note — and
// never the zenbu strip.
func TestBrowserLaneRegionView(t *testing.T) {
	pinKittyEnv(t)
	made := fakeSpawnPins(t)
	c := NewBrowserLaneController(64, 8)
	const u = "https://a.dev/page"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	f := (*made)[0]
	if _, err := f.grid.Write([]byte("FAKE KITTY FRAME ROW")); err != nil {
		t.Fatalf("paint the fake grid: %v", err)
	}
	premium := ansi.Strip(c.RegionView([]string{"text row must NOT paint"}))
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · " + u, "FAKE KITTY FRAME ROW"} {
		if !strings.Contains(premium, want) {
			t.Fatalf("the premium frame carries %q:\n%s", want, premium)
		}
	}
	if strings.Contains(premium, "text row must NOT paint") {
		t.Fatal("the text body never paints while premium runs")
	}

	f.exited, f.code, f.life = true, 1, 2*time.Second
	c.Poll()
	fellBack := ansi.Strip(c.RegionView([]string{"# fixture — text-mode body", "[1] " + u}))
	for _, want := range []string{" text ", "# fixture — text-mode body", "zenbu exited (1) — falling back to text mode"} {
		if !strings.Contains(fellBack, want) {
			t.Fatalf("the fallback frame carries %q:\n%s", want, fellBack)
		}
	}
	if strings.Contains(fellBack, "zenbu terminal-browser ·") {
		t.Fatal("the fallback frame drops the premium strip (the text location bar returns)")
	}
}

// laneWaitGrid — bounded wait for the child's bytes to paint the embedded
// grid (the reader loop is async; the assert itself carries no timing).
func laneWaitGrid(t *testing.T, g *term.Grid, want string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		for y := 0; y < g.Rows(); y++ {
			if strings.Contains(g.LineText(y), want) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the child's bytes never painted the embedded grid (want %q)", want)
}

// laneAssertReaped — the bounded-reap verdict: the session observed the
// exit AND the pid is gone (no child ever leaks past the office).
func laneAssertReaped(t *testing.T, s *ZenbuSession, pid int) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !s.Exited() && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if !s.Exited() {
		t.Fatal("the bounded reap observed the child's exit")
	}
	if err := syscall.Kill(pid, 0); err == nil || !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("no child leaks past the kill: kill(%d, 0) = %v", pid, err)
	}
}

// laneProcessState — the macOS-safe (no /proc) process state letter(s)
// ("Ts" — a stopped session leader; "Ss" — a live one): the SIGSTOP
// verdict's second rider after the heartbeat stall.
func laneProcessState(t *testing.T, pid int) string {
	t.Helper()
	out, err := exec.Command("ps", "-o", "stat=", "-p", fmt.Sprint(pid)).Output()
	if err != nil {
		t.Fatalf("ps -o stat= -p %d: %v", pid, err)
	}
	return strings.TrimSpace(string(out))
}

// TestBrowserLaneReapReal — the REAL spawn against a fixture-PATH fake
// binary (real PTY, real exec — the --openurl harness precedent): the
// child's bytes paint the embedded grid, Suspend (tab switch) FREEZES
// the SAME child (SIGSTOP — the pid stays ALIVE, never a respawn),
// Resume THAWS it (the pid is unchanged), and Close (office shutdown)
// group-kills + reaps — NO process ever leaks past the office.
func TestBrowserLaneReapReal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t) // lane env; zenbuLookPath stays REAL (PATH pins the fake)
	root := t.TempDir()
	fake := "#!/bin/sh\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := NewBrowserLaneController(64, 8)
	if c.Lane() != BrowserLaneZenbu {
		t.Fatal("the fixture PATH + kitty stub resolve the premium lane")
	}
	if err := c.OpenURL("https://x.dev/a"); err != nil {
		t.Fatalf("open a: %v", err)
	}
	sess, ok := c.Session().(*ZenbuSession)
	if !ok {
		t.Fatalf("the real seam embeds a *ZenbuSession, got %T", c.Session())
	}
	laneWaitGrid(t, sess.Grid(), "zenbu-fake open https://x.dev/a")
	pid := sess.Pid()

	// the keep-alive suspend: FREEZE, not kill — the pid stays ALIVE.
	c.Suspend()
	if !sess.Frozen() || !c.Suspended() {
		t.Fatal("Suspend freezes the child (the keep-alive posture)")
	}
	if sess.Exited() {
		t.Fatal("Suspend must NOT reap the child (keep-alive)")
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("the frozen child stays ALIVE: kill(%d, 0) = %v", pid, err)
	}
	if st := laneProcessState(t, pid); !strings.Contains(st, "T") {
		t.Fatalf("the frozen child is SIGSTOPped (ps state %q, want T…)", st)
	}
	if sess.Pid() != pid {
		t.Fatalf("the freeze never respawns: pid %d → %d", pid, sess.Pid())
	}

	// the thaw: the SAME child continues (the PID is stable across the flip).
	c.Resume()
	if sess.Frozen() || c.Suspended() {
		t.Fatal("Resume thaws the SAME child")
	}
	if sess.Pid() != pid {
		t.Fatalf("the flip's PID never changes: %d → %d", pid, sess.Pid())
	}
	if st := laneProcessState(t, pid); strings.Contains(st, "T") {
		t.Fatalf("the thawed child runs again (ps state %q, no T)", st)
	}

	// office-shutdown reap: Close group-kills + reaps the (live) child.
	c.Close()
	laneAssertReaped(t, sess, pid)
	// (the fresh-open-new-pid leg lives in
	// TestBrowserLaneOpenOtherURLWhileSuspended; the frozen-quit reap in
	// TestBrowserLaneFreezeHeartbeatReal.)
}

// TestBrowserLaneFreezeHeartbeatReal — the SIGSTOP verdict's PROCESS-LEVEL
// proof (no /proc, macOS-safe): the fake heartbeats into a LOG FILE
// (outside the office — the lane's own parking can never fake this);
// Suspend must STALL the log (the frozen child writes nothing), keep the
// pid alive + stopped (ps state T…), and Resume must resume the SAME
// pid's heartbeat. The tail then proves close-after-suspend still reaps
// (no zombie past a frozen quit).
func TestBrowserLaneFreezeHeartbeatReal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	hbLog := filepath.Join(root, "hb.log")
	fake := "#!/bin/sh\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"while true; do printf 'hb\\n' >> \"" + hbLog + "\"; sleep 0.05; done\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	hbLen := func() int64 {
		fi, err := os.Stat(hbLog)
		if err != nil {
			return 0
		}
		return fi.Size()
	}

	c := NewBrowserLaneController(64, 8)
	if err := c.OpenURL("https://x.dev/hb"); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	laneWaitGrid(t, sess.Grid(), "zenbu-fake open https://x.dev/hb")
	pid := sess.Pid()
	for deadline := time.Now().Add(2 * time.Second); hbLen() == 0 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if hbLen() == 0 {
		t.Fatal("setup: the heartbeat never started")
	}

	// FREEZE: the heartbeat STALLS (the child is SIGSTOPped, not killed).
	c.Suspend()
	if !sess.Frozen() {
		t.Fatal("Suspend freezes the session")
	}
	time.Sleep(150 * time.Millisecond) // let in-flight beats land
	frozenAt := hbLen()
	time.Sleep(500 * time.Millisecond) // ~10 beats would have landed unfrozen
	if got := hbLen(); got != frozenAt {
		t.Fatalf("the heartbeat must STALL while frozen: grew %d → %d", frozenAt, got)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("the frozen child stays ALIVE: kill(%d, 0) = %v", pid, err)
	}
	if st := laneProcessState(t, pid); !strings.Contains(st, "T") {
		t.Fatalf("the frozen child is SIGSTOPped (ps state %q)", st)
	}
	if sess.Pid() != pid {
		t.Fatalf("the freeze never respawns: pid %d → %d", pid, sess.Pid())
	}

	// THAW: the SAME pid's heartbeat resumes.
	c.Resume()
	if sess.Frozen() {
		t.Fatal("Resume thaws the session")
	}
	resumed := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if hbLen() > frozenAt {
			resumed = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !resumed {
		t.Fatal("the thawed child's heartbeat never resumed")
	}
	if sess.Pid() != pid {
		t.Fatalf("the flip's PID never changes: %d → %d", pid, sess.Pid())
	}

	// close-AFTER-SUSPEND still reaps (no zombie past a frozen quit).
	c.Suspend()
	if !sess.Frozen() {
		t.Fatal("the second suspend freezes again")
	}
	c.Close()
	laneAssertReaped(t, sess, pid)
}

// TestBrowserLaneSuspendedZeroStoreMutations — the resource-discipline
// pin (requirement: the SIGSTOP covers the CPU AND the lane's own render
// state is a frozen snapshot): with the fake streaming kitty frames
// continuously, Suspend must freeze the image store EXACTLY — no new
// applies (seq), no deletes queued, no drops logged, NOTHING through the
// direct-emit seam — for the whole suspended window; Resume unfreezes
// the stream (the seq advances again).
func TestBrowserLaneSuspendedZeroStoreMutations(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"while true; do printf '\\033_Ga=T,t=d,f=100,i=1,q=2;WkVST1BTU0VEMDEyMw==\\033\\\\'; sleep 0.02; done\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	var emitted []string
	restore := SetZenbuEmitForShot(func(s string) { emitted = append(emitted, s) })
	defer restore()
	ZenbuRegistry().Clear()
	defer ZenbuRegistry().Clear()

	c := NewBrowserLaneController(64, 8)
	if err := c.OpenURL("https://x.dev/stream"); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	// wait for the first frame to commit (the store has a live placement).
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if len(sess.images.placements()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(sess.images.placements()) == 0 {
		t.Fatal("setup: the streaming fake never committed a frame")
	}

	c.Suspend()
	// THE SNAPSHOT: every store-observable counter + the placements' seqs.
	seqOf := func() []uint64 {
		var out []uint64
		for _, p := range sess.images.placements() {
			out = append(out, p.seq)
		}
		return out
	}
	seqs := fmt.Sprint(seqOf())
	drops, note := sess.images.dropStats()
	sess.images.mu.Lock()
	pendingN := len(sess.images.pending)
	sess.images.mu.Unlock()
	if got := len(emitted); got != 0 {
		t.Fatalf("the freeze itself direct-emits NOTHING (the deletes ride the registry clear): got %d writes", got)
	}
	// the registry is CLEAR while frozen (the wrapper's next flush a=d's).
	{
		active, _, _, _, _ := ZenbuRegistry().snapshot()
		if active {
			t.Fatal("Freeze clears the frame-splice registry (the floor never shows the page)")
		}
	}

	time.Sleep(500 * time.Millisecond) // ~25 frames would have applied unfrozen

	if got := fmt.Sprint(seqOf()); got != seqs {
		t.Fatalf("ZERO store mutations while suspended — placement seqs moved: %s → %s", seqs, got)
	}
	if d, n := sess.images.dropStats(); d != drops || n != note {
		t.Fatalf("ZERO store mutations while suspended — drop stats moved: (%d,%q) → (%d,%q)", drops, note, d, n)
	}
	sess.images.mu.Lock()
	if len(sess.images.pending) != pendingN {
		sess.images.mu.Unlock()
		t.Fatal("ZERO store mutations while suspended — the pending delete queue moved")
	}
	sess.images.mu.Unlock()
	if got := len(emitted); got != 0 {
		t.Fatalf("NOTHING rides the emit seam while suspended: %d writes", got)
	}

	// the thaw unfreezes the stream (the seq advances again).
	c.Resume()
	advanced := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if fmt.Sprint(seqOf()) != seqs {
			advanced = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !advanced {
		t.Fatal("the thawed stream never resumed (the seq stayed frozen past Resume)")
	}
	c.Close()
	laneAssertReaped(t, sess, sess.Pid())
}

// TestBrowserLaneOpenOtherURLWhileSuspended — a fresh OpenURL from a
// SUSPENDED state keeps the kill+respawn semantics (requirement: the
// KILL PATHS are unchanged): the frozen child is group-killed + reaped
// (pid gone), the new url spawns a NEW child (a different pid).
func TestBrowserLaneOpenOtherURLWhileSuspended(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	fake := "#!/bin/sh\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := NewBrowserLaneController(64, 8)
	if err := c.OpenURL("https://x.dev/frozen"); err != nil {
		t.Fatalf("open A: %v", err)
	}
	sessA := c.Session().(*ZenbuSession)
	laneWaitGrid(t, sessA.Grid(), "zenbu-fake open https://x.dev/frozen")
	pidA := sessA.Pid()
	c.Suspend()
	if !sessA.Frozen() {
		t.Fatal("setup: A is frozen")
	}

	if err := c.OpenURL("https://x.dev/fresh"); err != nil {
		t.Fatalf("open B from the suspended state: %v", err)
	}
	// the frozen A was group-killed + reaped (the kill path is unchanged).
	laneAssertReaped(t, sessA, pidA)
	sessB, ok := c.Session().(*ZenbuSession)
	if !ok {
		t.Fatalf("B embeds a *ZenbuSession, got %T", c.Session())
	}
	laneWaitGrid(t, sessB.Grid(), "zenbu-fake open https://x.dev/fresh")
	if sessB.Pid() == pidA {
		t.Fatal("a fresh open is a fresh process (the frozen group is gone)")
	}
	if sessB.Frozen() {
		t.Fatal("the fresh child is NOT frozen (the freeze never leaks across opens)")
	}
	c.Close()
	laneAssertReaped(t, sessB, sessB.Pid())
}

// TestBrowserLaneFrozenDeathFallback — a child that DIES WHILE FROZEN
// (an external SIGKILL — the frozen process can never exit on its own)
// is detected by Poll and rides the EXISTING fallback: the session
// drops, the exact note latches (a signal death wears -1), the no-flap
// latch keeps Resume text.
func TestBrowserLaneFrozenDeathFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	made := fakeSpawnPinsCount(t)
	root := t.TempDir()
	fake := "#!/bin/sh\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := NewBrowserLaneController(64, 8)
	const u = "https://x.dev/doomed"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	laneWaitGrid(t, sess.Grid(), "zenbu-fake open "+u)
	pid := sess.Pid()
	c.Suspend()
	if !sess.Frozen() {
		t.Fatal("setup: the child is frozen")
	}

	// the frozen child is murdered externally (SIGKILL reaches stopped
	// processes); the waiter reaps it.
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill the frozen child: %v", err)
	}
	for deadline := time.Now().Add(3 * time.Second); !sess.Exited() && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if !sess.Exited() {
		t.Fatal("the murdered frozen child never reaped")
	}

	// Poll lands the existing fallback contract.
	if !c.Poll() {
		t.Fatal("Poll observes the frozen death")
	}
	if c.Session() != nil || c.PremiumActive() || c.Suspended() {
		t.Fatal("the fallback drops the session (no ghost freeze)")
	}
	if got, want := c.Note(), "zenbu exited (-1) — falling back to text mode"; got != want {
		t.Fatalf("the signal-death note wears -1: got %q want %q", got, want)
	}
	// the no-flap latch: Resume never respawns a fell-back url.
	c.Resume()
	if *made != 1 || c.PremiumActive() {
		t.Fatalf("a frozen death latches the text lane: spawns=%d active=%v", *made, c.PremiumActive())
	}
	c.Close()
}

// -------------------------------------------------------------------
// the freeze-preserve discipline (the production freeze-leak fix):
// park PRESERVES the pending tail + open chain — the thawed child's
// tail COMPLETES the in-flight chain into a valid frame; resetting at
// the freeze dropped the chain's HEAD and the resumed tail painted the
// grid with raw base64 (the member's ~7 dense rows on re-open)
// -------------------------------------------------------------------

// kittyStreamPayload — the streaming fake's frame content (512B → 684
// base64 chars): BIG enough that a leaked mid-body tail trips the ≥40
// b64Runs signature several times over (kittyTestB64's 76-char fragments
// could hide under the threshold; the real child's ~4KB chunks never
// could). Content is irrelevant — the store never image-decodes.
var kittyStreamPayload = func() []byte {
	p := make([]byte, 0, 512)
	p = append(p, "\x89PNG\r\n\x1a\n"...)
	for len(p) < 512 {
		p = append(p, byte('A'+len(p)%26))
	}
	return p
}()

func kittyStreamB64() string { return base64.StdEncoding.EncodeToString(kittyStreamPayload) }

// browserLaneStreamFake — requirement 4's REAL streaming child (the
// production Electron's fixture-scale twin): ~2fps of THREE-chunk kitty
// frames (a=T m=1 / m=1 / m=0) under the child's every-repaint id i=1,
// EVERY chunk's APC body split across a 60ms sleep (the mid-body freeze
// windows — a freeze landing there catches the splitter holding a
// PARTIAL APC body, the exact leak shape), 300ms between generations.
func browserLaneStreamFake(root, b64 string) error {
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-STREAM\\r\\n'\n" +
		"while true; do\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:114] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[114:228] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[228:342] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[342:456] + "\\033\\\\'\n" +
		"printf '\\033_Gm=0;" + b64[456:570] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[570:] + "\\033\\\\'\n" +
		"sleep 0.3\n" +
		"done\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// zenbuSplitState — the splitter's freeze-point posture (read under the
// split mutex): the pending byte count, the scanner mode flags, whether
// a chunk chain is open, whether the buffer holds a DANGLING opener
// (unscanned ESC_G + partial body — the park scans nothing), and the
// pending head bytes (the freeze-point dump's evidence).
type zenbuSplitState struct {
	pending  int
	inAPC    bool
	discard  bool
	chain    bool
	parked   bool
	dangling bool
	pendHead string
}

func laneSplitState(s *ZenbuSession) zenbuSplitState {
	s.split.mu.Lock()
	defer s.split.mu.Unlock()
	head := s.split.pending
	if len(head) > 48 {
		head = head[:48]
	}
	return zenbuSplitState{
		pending:  len(s.split.pending),
		inAPC:    s.split.inAPC,
		discard:  s.split.discard,
		chain:    s.split.chain != nil,
		parked:   s.split.parked,
		dangling: danglingKittyOpener(s.split.pending),
		pendHead: string(head),
	}
}

// lanePlacementSeq — the live placement's store seq (the streaming
// fake's i=1 latest-wins placement is the commit clock: +1 per landed
// generation).
func lanePlacementSeq(t *testing.T, sess *ZenbuSession) uint64 {
	t.Helper()
	ps := sess.images.placements()
	if len(ps) != 1 {
		t.Fatalf("the streaming lane holds exactly one placement, got %d", len(ps))
	}
	return ps[0].seq
}

// laneWaitPlacementSeq — bounded wait for the store seq to reach want
// (the reader loop is async; the assert itself carries no timing).
func laneWaitPlacementSeq(t *testing.T, sess *ZenbuSession, want uint64, what string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if lanePlacementSeq(t, sess) >= want {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("%s never landed (placement seq stuck at %d, want >= %d)", what, lanePlacementSeq(t, sess), want)
}

// laneFreezeMidStream — freeze the streaming child at a random point,
// looping until the park catches bytes IN FLIGHT (a non-empty pending
// and/or an open chain — the production shape: the Electron child
// repaints ~1.4MB chains at ~2fps, so it is virtually ALWAYS mid-chain
// when frozen). A between-frames freeze (nothing in flight) is the
// no-op leg: thaw and drift the phase.
func laneFreezeMidStream(t *testing.T, c *BrowserLaneController, sess *ZenbuSession) zenbuSplitState {
	t.Helper()
	for attempt := 0; attempt < 40; attempt++ {
		c.Suspend()
		time.Sleep(60 * time.Millisecond) // the kernel PTY buffer drains into the parked splitter
		st := laneSplitState(sess)
		if st.pending > 0 || st.chain {
			return st
		}
		c.Resume()
		time.Sleep(time.Duration(53+17*(attempt%6)) * time.Millisecond) // the phase drift
	}
	t.Fatal("40 flips never froze the stream mid-chain (the production case)")
	return zenbuSplitState{}
}

// TestBrowserLaneParkEmptyPendingNoop — edge discipline 3(a): a freeze
// with NOTHING in flight is a pure no-op — no drop note, no reset, and
// the lane parses cleanly across the boundary.
func TestBrowserLaneParkEmptyPendingNoop(t *testing.T) {
	ks, store, g, sb := newKittyRig(40, 8)
	ks.park()
	ks.unpark()
	if drops, note := store.dropStats(); drops != 0 || note != "" {
		t.Fatalf("an empty-pending park/unpark logs NOTHING: drops=%d note=%q", drops, note)
	}
	stream, b64 := kittyScriptedStream()
	if _, err := ks.Write([]byte(stream)); err != nil {
		t.Fatalf("splitter write: %v", err)
	}
	store.mu.Lock()
	im := store.images[1]
	store.mu.Unlock()
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the frame parses cleanly across the no-op boundary: %+v", im)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("zero base64 downstream (%d runs)", n)
	}
}

// TestBrowserLaneParkPreservesChain — the fix's DETERMINISTIC core (no
// process): one frame commits, the next generation's chain opens, the
// PARK lands, and the in-flight tail arrives WHILE PARKED (the SIGSTOP's
// effect lag) — the pending tail + open chain are PRESERVED (zero
// store/grid/scrollback mutations), the unpark drains the buffered tail
// (a partial APC holds), and the thawed child's remaining bytes COMPLETE
// the chain into a valid frame — ZERO base64 downstream. (The old
// reset-at-park dropped the chain's HEAD here; the tail leaked.)
func TestBrowserLaneParkPreservesChain(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, g, sb := newKittyRig(40, 8)
	// generation 1 commits (the retained frame), generation 2's chain opens.
	ks.Write([]byte("\x1b[2J\x1b[H" + "TB-TOOLBAR\r\n" + kittyAPC("a=T,t=d,f=100,i=1,q=2", b64)))
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20])))
	seq0 := store.placements()[0].seq

	ks.park()
	gridAtPark := g.ScreenText()
	sbLenAtPark := len(sb.Raw())
	// the in-flight tail lands WHILE PARKED: an unterminated APC body.
	if _, err := ks.Write([]byte("\x1b_Gm=1;" + b64[20:40])); err != nil {
		t.Fatalf("parked write: %v", err)
	}
	// ZERO mutations while parked: the store, the grid, the scrollback
	// are a frozen snapshot; the bytes sit in the pending buffer.
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("ZERO store mutations while parked: seq moved %d → %d", seq0, got)
	}
	if got := g.ScreenText(); got != gridAtPark {
		t.Fatalf("ZERO grid mutations while parked:\n--- at park ---\n%s\n--- now ---\n%s", gridAtPark, got)
	}
	if got := len(sb.Raw()); got != sbLenAtPark {
		t.Fatalf("ZERO scrollback mutations while parked: %d → %d bytes", sbLenAtPark, got)
	}
	ks.mu.Lock()
	held := len(ks.pending) > 0 && ks.chain != nil
	ks.mu.Unlock()
	if !held {
		t.Fatal("the park PRESERVES the pending tail + the open chain")
	}

	ks.unpark() // the buffered tail drains: no terminator yet → it holds
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("the unpark's drain commits nothing without the terminator: seq %d → %d", seq0, got)
	}
	// the thawed child's tail completes chunk 2, then chunk 3 lands.
	ks.Write([]byte("\x1b\\"))
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("chunk 2 (m=1) continues the chain without committing: seq %d → %d", seq0, got)
	}
	ks.Write([]byte(kittyAPC("m=0", b64[40:])))
	// THE chain completed into a valid frame: full payload, placed, +1 seq.
	store.mu.Lock()
	im := store.images[1]
	store.mu.Unlock()
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the preserved chain completed into the valid frame: %+v", im)
	}
	if got := store.placements()[0].seq; got != seq0+1 {
		t.Fatalf("exactly ONE new apply across the thaw: seq %d → %d, want %d", seq0, got, seq0+1)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("ZERO base64 across the freeze/thaw (%d runs):\n%s", n, g.ScreenText())
	}
	if got := g.LineText(0); got != "TB-TOOLBAR" {
		t.Fatalf("the chrome survives: %q", got)
	}
}

// TestBrowserLaneParkedChainCap — edge discipline 3(b): the 8MiB chain
// cap (shrunk here) still bounds a pathological pending chain — the
// WHOLE over-cap chain arrives while PARKED (buffered unscanned), the
// unpark's drain hits the cap mid-join and drops the chain log-once —
// NEVER a flush to the grid or the scrollback — and the lane keeps
// working.
func TestBrowserLaneParkedChainCap(t *testing.T) {
	old := maxKittyChainB64
	maxKittyChainB64 = 32 // shrunk for the test
	t.Cleanup(func() { maxKittyChainB64 = old })
	b64 := kittyTestB64() // 76 chars > the 32-char cap once joined
	ks, store, g, sb := newKittyRig(40, 8)
	ks.park()
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]) +
		kittyAPC("m=1", b64[20:60]) + // 20+40 = 60 joined > 32 → the cap fires in the drain
		kittyAPC("m=0", b64[60:])))
	if drops, _ := store.dropStats(); drops != 0 {
		t.Fatalf("ZERO store mutations while parked: drops=%d", drops)
	}
	ks.unpark() // the drain scans the buffered chain: the cap drops it
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "cap") {
		t.Fatalf("the cap drop + the chain-less remainder log-once: drops=%d note=%q", drops, note)
	}
	store.mu.Lock()
	if len(store.images) != 0 {
		t.Fatalf("the over-cap chain never stores: %d", len(store.images))
	}
	store.mu.Unlock()
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the over-cap chain never leaks downstream (%d runs)", n)
	}
	ks.Write([]byte("alive"))
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("the lane survives the parked cap drop: %q", got)
	}
}

// TestBrowserLaneParkedBufferCap — the parked path's OWN overflow (the
// APC body cap, shrunk here): an UNSCANNED ESC_G opener + a runaway
// unterminated body arrive while parked and bust the cap — the buffer +
// the open chain drop log-once AND the DISCARD mode engages (the
// dangling opener means the thawed child's next bytes are that APC's
// tail — they must discard to their terminator, NEVER flush to grid).
func TestBrowserLaneParkedBufferCap(t *testing.T) {
	old := maxKittyAPCBody
	maxKittyAPCBody = 64 // shrunk for the test
	t.Cleanup(func() { maxKittyAPCBody = old })
	b64 := kittyTestB64()
	ks, store, g, sb := newKittyRig(40, 8)
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]))) // the chain opens
	ks.park()
	// the runaway: an UNSCANNED opener + unterminated body over the cap.
	ks.Write([]byte("\x1b_Gm=1;" + strings.Repeat("A", 100))) // 109 bytes > 64
	ks.mu.Lock()
	parked, inAPC, discard, pend, chain := ks.parked, ks.inAPC, ks.discard, len(ks.pending), ks.chain != nil
	ks.mu.Unlock()
	if !parked || inAPC || !discard || pend != 0 || chain {
		t.Fatalf("the parked overflow drops the buffer + chain and engages the discard: parked=%v inAPC=%v discard=%v pending=%d chain=%v",
			parked, inAPC, discard, pend, chain)
	}
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "parked buffer over the cap") {
		t.Fatalf("the chain drop + the cap drop log-once: drops=%d note=%q", drops, note)
	}
	// the thaw: the runaway body's REST + its terminator discard; the
	// stream resyncs AFTER it.
	ks.unpark()
	ks.Write([]byte(strings.Repeat("B", 50) + "\x1b\\" + "alive"))
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("the discard eats the runaway tail; the resync paints: %q", got)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the runaway tail NEVER flushes downstream (%d runs):\n%s", n, g.ScreenText())
	}
}

// TestBrowserLaneFreezeSignalOrdering — edge discipline 3(d), PROVEN at
// the signal's own instant (the zenbuGroupSignal seam): Freeze's SIGSTOP
// must fire with the park ALREADY engaged (and a byte written AT that
// instant — the racing child write the kernel accepted before the stop
// took effect — lands in the pending buffer, NEVER the grid); Unfreeze's
// SIGCONT must fire with the unpark ALREADY lifted (the preserved
// pending drains before the child emits).
func TestBrowserLaneFreezeSignalOrdering(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	fake := "#!/bin/sh\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	ZenbuRegistry().Clear()
	defer ZenbuRegistry().Clear()

	c := NewBrowserLaneController(64, 8)
	if err := c.OpenURL("https://x.dev/order"); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	laneWaitGrid(t, sess.Grid(), "zenbu-fake open https://x.dev/order")
	pid := sess.Pid()

	type sigObs struct {
		target    int
		sig       syscall.Signal
		parked    bool // the splitter's park state AT the signal instant
		pendAfter int  // the probe byte's landing spot (SIGSTOP leg)
	}
	var obs []sigObs
	old := zenbuGroupSignal
	zenbuGroupSignal = func(p int, sig syscall.Signal) error {
		o := sigObs{target: p, sig: sig}
		sess.split.mu.Lock()
		o.parked = sess.split.parked
		sess.split.mu.Unlock()
		if sig == syscall.SIGSTOP {
			// the leak window's probe: the racing child byte written AT
			// the signal instant (the park is already up) must buffer.
			_, _ = sess.split.Write([]byte("\x1b_Ga=T,t=d,f=100,i=1,q=2,m=1;" + kittyTestB64()[:20]))
			sess.split.mu.Lock()
			o.pendAfter = len(sess.split.pending)
			sess.split.mu.Unlock()
		}
		obs = append(obs, o)
		return nil // the REAL signal never fires (the fake keeps running)
	}
	t.Cleanup(func() { zenbuGroupSignal = old })

	if err := sess.Freeze(); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if len(obs) != 1 || obs[0].target != -pid || obs[0].sig != syscall.SIGSTOP {
		t.Fatalf("Freeze signals exactly one SIGSTOP to the process group: %+v", obs)
	}
	if !obs[0].parked {
		t.Fatal("the SIGSTOP fired with the park NOT engaged — the leak window is OPEN")
	}
	if obs[0].pendAfter == 0 {
		t.Fatal("a byte written AT the SIGSTOP instant lands in the pending buffer")
	}
	if n := gridB64Runs(sess.Grid()); n != 0 {
		t.Fatalf("the signal-instant byte NEVER touches the grid (%d runs)", n)
	}
	t.Logf("ORDER: park engaged BEFORE SIGSTOP(-%d) — the signal-instant probe byte buffered (%d pending), grid untouched", pid, obs[0].pendAfter)

	if err := sess.Unfreeze(); err != nil {
		t.Fatalf("Unfreeze: %v", err)
	}
	if len(obs) != 2 || obs[1].target != -pid || obs[1].sig != syscall.SIGCONT {
		t.Fatalf("Unfreeze signals exactly one SIGCONT to the process group: %+v", obs)
	}
	if obs[1].parked {
		t.Fatal("the SIGCONT fired with the park STILL engaged — the thawed child's bytes would stall")
	}
	if n := gridB64Runs(sess.Grid()); n != 0 {
		t.Fatalf("the probe byte stays out of the grid across the thaw (%d runs)", n)
	}
	t.Logf("ORDER: unpark lifted BEFORE SIGCONT(-%d) — the preserved pending drained ahead of the child", pid)

	zenbuGroupSignal = old // the teardown's SIGKILL must REALLY fire
	c.Close()
	laneAssertReaped(t, sess, pid)
}

// TestBrowserLaneFreezeMidChainReal — requirement 4's headline
// regression: a REAL streaming fake (chunked kitty frames CONTINUOUSLY
// at ~2fps, every chunk's body split across a sleep — the production
// Electron's shape), frozen at a random point until the park catches
// bytes IN FLIGHT (asserted), then thawed: the preserved chain's tail
// COMPLETES it into a valid store frame, a SUBSEQUENT complete frame
// parses behind it, and the grid + scrollback carry ZERO base64. (The
// old reset-at-freeze dropped the chain's HEAD — the thawed tail painted
// ~7 dense base64 rows on the member's re-open.)
func TestBrowserLaneFreezeMidChainReal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	b64 := kittyStreamB64()
	if err := browserLaneStreamFake(root, b64); err != nil {
		t.Fatalf("plant the streaming fake: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	ZenbuRegistry().Clear()
	defer ZenbuRegistry().Clear()
	restore := SetZenbuEmitForShot(func(string) {}) // Close's direct deletes are noise here
	t.Cleanup(restore)

	c := NewBrowserLaneController(64, 16)
	if err := c.OpenURL("https://x.dev/stream"); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	// the stream is healthy: the first generation committed.
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if len(sess.images.placements()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(sess.images.placements()) == 0 {
		t.Fatal("setup: the streaming fake never committed a frame")
	}
	pid := sess.Pid()

	held := laneFreezeMidStream(t, c, sess)
	seqAtFreeze := lanePlacementSeq(t, sess)
	t.Logf("FREEZE POINT: pending=%d bytes inAPC=%v chainOpen=%v danglingOpener=%v parked=%v seq=%d",
		held.pending, held.inAPC, held.chain, held.dangling, held.parked, seqAtFreeze)
	t.Logf("FREEZE POINT pending head: %q", held.pendHead)

	// the frozen window: the parked splitter mutates NOTHING.
	time.Sleep(150 * time.Millisecond)
	if got := lanePlacementSeq(t, sess); got != seqAtFreeze {
		t.Fatalf("ZERO store mutations while frozen: seq moved %d → %d", seqAtFreeze, got)
	}

	// THAW: the preserved chain's tail COMPLETES it into a valid frame…
	c.Resume()
	laneWaitPlacementSeq(t, sess, seqAtFreeze+1, "the in-flight chain's completion")
	sess.images.mu.Lock()
	im := sess.images.images[1]
	sess.images.mu.Unlock()
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the completed chain is the valid frame (placed, the full %d-char payload): %+v", len(b64), im)
	}
	if _, err := base64.StdEncoding.DecodeString(im.b64); err != nil {
		t.Fatalf("the completed chain's payload decodes: %v", err)
	}
	t.Logf("THAWED: the in-flight chain COMPLETED — i=1 seq %d → %d, the full %d-char payload, decodable", seqAtFreeze, seqAtFreeze+1, len(im.b64))

	// …and a SUBSEQUENT complete frame parses behind it.
	laneWaitPlacementSeq(t, sess, seqAtFreeze+2, "the next generation's frame")
	t.Logf("SUBSEQUENT frame parsed: seq → %d", lanePlacementSeq(t, sess))

	// THE regression pin: ZERO base64 in the grid AND the scrollback.
	if n := gridB64Runs(sess.Grid()); n != 0 {
		t.Fatalf("base64 leaked into the grid across the freeze/thaw (%d runs):\n%s", n, sess.Grid().ScreenText())
	}
	if n := b64Runs(string(sess.Scrollback().Raw())); n != 0 {
		t.Fatalf("base64 leaked into the scrollback across the freeze/thaw (%d runs)", n)
	}
	t.Logf("ZERO base64: grid runs=0 scrollback runs=0 — the freeze preserved the chain's HEAD; the thawed tail completed it")

	c.Close()
	laneAssertReaped(t, sess, pid)
}

// TestBrowserLaneFrozenDeathMidChainReset — edge discipline 3(c): the
// streaming child is frozen with bytes IN FLIGHT, then MURDERED (an
// external SIGKILL reaches the stopped process) — the chain can never
// complete, so Unfreeze RESETS the splitter (the held tail + open chain
// drop log-once) and KEEPS THE GATE PARKED (the reader's final drained
// bytes never scan their way to the grid), ahead of Poll's existing
// fallback latch.
func TestBrowserLaneFrozenDeathMidChainReset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	b64 := kittyStreamB64()
	if err := browserLaneStreamFake(root, b64); err != nil {
		t.Fatalf("plant the streaming fake: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	ZenbuRegistry().Clear()
	defer ZenbuRegistry().Clear()
	restore := SetZenbuEmitForShot(func(string) {}) // Close's direct deletes are noise here
	t.Cleanup(restore)

	c := NewBrowserLaneController(64, 16)
	const u = "https://x.dev/doomed-stream"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if len(sess.images.placements()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(sess.images.placements()) == 0 {
		t.Fatal("setup: the streaming fake never committed a frame")
	}
	pid := sess.Pid()

	held := laneFreezeMidStream(t, c, sess)
	t.Logf("FREEZE POINT (doomed): pending=%d bytes inAPC=%v chainOpen=%v danglingOpener=%v head=%q",
		held.pending, held.inAPC, held.chain, held.dangling, held.pendHead)

	// MURDER the frozen child (SIGKILL reaches stopped processes); the
	// waiter reaps it.
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill the frozen child: %v", err)
	}
	for deadline := time.Now().Add(3 * time.Second); !sess.Exited() && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if !sess.Exited() {
		t.Fatal("the murdered frozen child never reaped")
	}

	// THE 3(c) PATH: the thaw finds the child DEAD — reset first, the
	// gate STAYS parked, no signal rides.
	c.Resume()
	st := laneSplitState(sess)
	if st.pending != 0 || st.chain || st.inAPC || !st.parked {
		t.Fatalf("the dead child's thaw reset the splitter and kept the gate parked: pending=%d chain=%v inAPC=%v parked=%v",
			st.pending, st.chain, st.inAPC, st.parked)
	}
	drops, note := sess.images.dropStats()
	if drops == 0 || !strings.Contains(note, "session reset dropped") {
		t.Fatalf("the reset dropped the held tail + chain log-once: drops=%d note=%q", drops, note)
	}
	t.Logf("DEAD-AT-THAW: the held %d pending bytes + chain dropped log-once (%q) — the chainless tail can never flush", held.pending, note)
	if n := gridB64Runs(sess.Grid()) + b64Runs(string(sess.Scrollback().Raw())); n != 0 {
		t.Fatalf("ZERO base64 across the frozen murder (%d runs)", n)
	}

	// …and the EXISTING fallback latches behind it.
	if !c.Poll() {
		t.Fatal("Poll observes the frozen death")
	}
	if got, want := c.Note(), "zenbu exited (-1) — falling back to text mode"; got != want {
		t.Fatalf("the signal-death note wears -1: got %q want %q", got, want)
	}
	c.Close()
}
