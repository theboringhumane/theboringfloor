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
	"os"
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

// TestBrowserLaneSuspendResumeClose — the lifecycle: Suspend kills the
// embed (tab switch, silent), Resume re-spawns it (lane still premium,
// url never fell back — NO new history entry), Close seals (office
// shutdown) and is idempotent.
func TestBrowserLaneSuspendResumeClose(t *testing.T) {
	pinKittyEnv(t)
	made := fakeSpawnPins(t)
	c := NewBrowserLaneController(64, 16)
	const u = "https://a.dev/live"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	f1 := (*made)[0]

	c.Suspend() // the pane switched away
	if !f1.closed {
		t.Fatal("Suspend SIGKILLs the embedded child")
	}
	if c.PremiumActive() || c.Session() != nil {
		t.Fatal("Suspend drops the pane back to the text lane")
	}
	if c.Note() != "" {
		t.Fatalf("a tab switch is not a failure — no note: %q", c.Note())
	}

	c.Resume() // the pane is back — the embed re-spawns for the SAME url
	if len(*made) != 2 || !c.PremiumActive() {
		t.Fatalf("Resume re-spawns the premium embed: made=%d active=%v", len(*made), c.PremiumActive())
	}
	if got := strings.Join(c.VisitedURLs(), ","); got != u {
		t.Fatalf("Resume adds NO history entry: got %q", got)
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

// TestBrowserLaneReapReal — the REAL spawn against a fixture-PATH fake
// binary (real PTY, real exec — the --openurl harness precedent): the
// child's bytes paint the embedded grid, Suspend (tab switch) group-kills
// + reaps, and Close (office shutdown) reaps a fresh child — NO process
// ever leaks past the office.
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

	waitGrid := func(t *testing.T, g *term.Grid, want string) {
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
	assertReaped := func(t *testing.T, s *ZenbuSession, pid int) {
		t.Helper()
		if !s.Exited() {
			t.Fatal("the bounded reap observed the child's exit")
		}
		if err := syscall.Kill(pid, 0); err == nil || !errors.Is(err, syscall.ESRCH) {
			t.Fatalf("no child leaks past the kill: kill(%d, 0) = %v", pid, err)
		}
	}

	// tab-switch reap.
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
	waitGrid(t, sess.Grid(), "zenbu-fake open https://x.dev/a")
	pid := sess.Pid()
	c.Suspend()
	assertReaped(t, sess, pid)

	// office-shutdown reap: a fresh open spawns fresh; Close kills it.
	if err := c.OpenURL("https://x.dev/b"); err != nil {
		t.Fatalf("open b: %v", err)
	}
	sess2, ok := c.Session().(*ZenbuSession)
	if !ok {
		t.Fatalf("the fresh embed is a *ZenbuSession, got %T", c.Session())
	}
	waitGrid(t, sess2.Grid(), "zenbu-fake open https://x.dev/b")
	pid2 := sess2.Pid()
	c.Close()
	assertReaped(t, sess2, pid2)
	if pid2 == pid {
		t.Fatal("a fresh open is a fresh process (the old group is gone)")
	}
}
