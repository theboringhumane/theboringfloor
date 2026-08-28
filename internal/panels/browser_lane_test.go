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
