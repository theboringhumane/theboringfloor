// browser_lane_test.go — the premium lane's LIVE APP wiring (the pane
// half lives in internal/panels' browser_panel_lane_test.go; the
// controller's own matrix in browser_lane_test.go there): a REAL fake
// `terminal-browser` binary planted on a pinned PATH under the hermetic
// ghostty stub (the TestBrowserLaneReapReal/--openurl precedent — real
// PTY, real exec, hermetic binary) driven through the REAL app glue:
//
//	(a) /open through the slash path spawns the embedded child — the
//	    LEFT slot's frame wears the " zenbu " badge + the "▸ zenbu
//	    terminal-browser · <url>" strip + the child's painted marker,
//	    the RIGHT strip unmoved, and the text fetch rode underneath;
//	(b) unclaimed keys reach the child (a typed letter echoes through
//	    the real PTY) while the office's own claims (ctrl+b, q/esc)
//	    still win — q FREEZES the session (keep-alive) AND returns to
//	    the floor;
//	(c) ctrl+b SUSPENDS the lane (the child FREEZES — SIGSTOPped,
//	    alive, the spawn log stops at ONE) and returning RESUMES it
//	    (the SAME child thaws — the PID is unchanged, never a respawn);
//	(d) an early exit (<300ms) lands the text fallback THROUGH THE APP:
//	    the pane's real viewer (warm from the fetch) + the dim
//	    "zenbu exited (0) — falling back to text mode" note, and the
//	    no-flap latch keeps a re-open text;
//	(e) a WindowSizeMsg resize SIGWINCHes the live child to the slot's
//	    exact cols/rows (the slot is narrower than the old right-pane
//	    tab — the math is asserted at REAL sizes).
package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// pinBrowserLaneEnv — the hermetic kitty-capable host stub for the
// app-level lane tests (panels' pinKittyEnv checklist, one package up;
// zenbuLookPath stays REAL — the planted PATH pins the fake). The wave-85
// opt-in flag is SET here: this is the "premium resolves" stub (panels'
// pinKittyEnv's exact discipline — tests wanting the default-off world
// clear the flag after pinning).
func pinBrowserLaneEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM_VERSION", "")
	t.Setenv("WEZTERM_UNIX_SOCKET", "")
	t.Setenv("VSCODE_PID", "")
	t.Setenv("ITERM_SESSION_ID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv(panels.BrowserLaneOffEnv, "")
	t.Setenv(panels.TerminalBrowserOffEnv, "")
	t.Setenv(panels.BrowserLaneOptInEnv, "1")
}

// plantFakeTerminalBrowser — the hermetic fake: logs every invocation's
// args to calls.log (the spawn count IS the no-flap latch's evidence),
// prints its marker row, then runs the flavor's tail ("cat" echoes the
// keys it is written; "die" exits 0 immediately — an early death through
// the PTY seam; anything else parks ~11 days).
func plantFakeTerminalBrowser(t *testing.T, flavor string) (logPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	root := t.TempDir()
	log := filepath.Join(root, "calls.log")
	tail := "exec sleep 1000000"
	switch flavor {
	case "cat":
		tail = "exec cat"
	case "die":
		tail = "exit 1"
	}
	bin := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + log + "\"\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		tail + "\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(bin), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// laneSpawnCount — lines in the fake's call log (0 while absent).
func laneSpawnCount(t *testing.T, logPath string) int {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if err != nil {
		return 0
	}
	n := strings.Count(strings.TrimSpace(string(b)), "\n") + 1
	if strings.TrimSpace(string(b)) == "" {
		return 0
	}
	return n
}

// waitLaneGrid — bounded wait for the child's bytes to paint the embedded
// grid (the reader loop is async; the assert itself carries no timing).
func waitLaneGrid(t *testing.T, m Model, needle string) Model {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if m.BrowserLaneGridHas(needle) {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the child's %q never painted the embedded grid", needle)
	return m
}

// waitLaneDropped — bounded wait for the pane's poll ride to observe the
// dead child (View runs the poll; the frame cache never gates a direct
// panel View).
func waitLaneDropped(t *testing.T, m Model) Model {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		_ = m.browser.View() // the pane's poll ride
		if !m.browser.PremiumActive() {
			return m
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the dead child never dropped (poll ride)")
	return m
}

// laneFixtureURL — the shared panels fixture as a file:// URL (the fetch
// half rides the REAL file source — deterministic, no network).
func laneFixtureURL(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../panels/testdata/fixture.html")
	if err != nil {
		t.Fatal(err)
	}
	return "file://" + abs
}

// TestBrowserLaneLiveEmbed — (a) + the key/ownership legs of (b): /open
// through the REAL slash path spawns the embedded child (badge + strip +
// painted marker in the LEFT slot, right strip unmoved); a typed letter
// echoes through the real PTY; q leaves AND closes the session.
func TestBrowserLaneLiveEmbed(t *testing.T) {
	pinBrowserLaneEnv(t)
	logPath := plantFakeTerminalBrowser(t, "cat")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := laneFixtureURL(t)

	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("/open flips the left slot to the browser, got %d", got)
	}
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("the right strip never moves, got %d", got)
	}
	if !m.BrowserPremiumActive() {
		t.Fatal("the lane resolved + spawned the premium embed through the app glue")
	}
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "lane live"})
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · file:///", "zenbu-fake open file:///", "· ctrl+b"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the live premium frame carries %q:\n%s", want, frame)
		}
	}
	// the strip paints INSIDE the left slot (left of the sidebar's chat).
	for _, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "zenbu terminal-browser") {
			zi, ci := strings.Index(line, "zenbu terminal-browser"), strings.Index(line, "chat")
			if ci >= 0 && zi > ci {
				t.Fatalf("the zenbu strip must paint LEFT of the sidebar's chat tab: %q", line)
			}
			break
		}
	}

	// (b) a typed letter forwards through the REAL PTY (the fake's `cat`
	// echoes it back onto the grid) — the office draft never sees it.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'Z', Text: "Z"}))
	m = waitLaneGrid(t, m, "Z") // the PTY echo paints it — the text lane's j/k surface is bypassed
	if f := ansi.Strip(m.Frame()); strings.Contains(f, "› Z") {
		t.Fatalf("the premium lane owns typed keys (no draft leak):\n%s", f)
	}

	// q leaves AND FREEZES the session (SuspendLane rides the leave —
	// keep-alive: the child stays alive behind the floor, spawn log 1).
	pid := m.BrowserLanePid()
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("q on the premium slot must land on the floor, got %d", got)
	}
	if m.BrowserPremiumActive() {
		t.Fatal("q hides the premium session with the leave")
	}
	if !m.BrowserLaneSuspended() {
		t.Fatal("q FREEZES the premium session (the keep-alive posture)")
	}
	if m.BrowserLanePid() != pid {
		t.Fatalf("the leave never respawns: pid %d → %d", pid, m.BrowserLanePid())
	}
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("the leave never re-spawns: spawn log = %d, want 1", got)
	}
	// the frozen child never leaks past the test (the quit path's reap).
	m.browser.Close()
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("the quit path reaps the frozen child: kill(%d, 0) = %v", pid, err)
	}
}

// TestBrowserLaneLiveSuspendResume — (c): ctrl+b to the floor SUSPENDS
// the lane (the child FREEZES — SIGSTOPped, ALIVE, the spawn log stops
// at ONE); ctrl+b back RESUMES it (the SAME child thaws — the PID is
// unchanged, NO respawn, no new fetch, no flap).
func TestBrowserLaneLiveSuspendResume(t *testing.T) {
	pinBrowserLaneEnv(t)
	logPath := plantFakeTerminalBrowser(t, "sleep")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := laneFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("setup: one spawn, got %d", got)
	}
	pid := m.BrowserLanePid()
	if pid <= 0 {
		t.Fatalf("setup: the premium child's pid reads through the harness seam, got %d", pid)
	}

	m = runMsg(t, m, ctrlB()) // → floor: SUSPEND (FREEZE — keep-alive)
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("ctrl+b flips to the floor, got %d", got)
	}
	if m.BrowserPremiumActive() {
		t.Fatal("ctrl+b to the floor suspends the premium lane")
	}
	if !m.BrowserLaneSuspended() {
		t.Fatal("the frozen child keeps the keep-alive posture")
	}
	if got := m.BrowserLanePid(); got != pid {
		t.Fatalf("the freeze never respawns: pid %d → %d", pid, got)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("the frozen child stays ALIVE behind the floor: kill(%d, 0) = %v", pid, err)
	}
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("a suspend never spawns: log %d, want 1", got)
	}

	m = runMsg(t, m, ctrlB()) // → browser: RESUME (the SAME child thaws)
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("ctrl+b flips back to the browser, got %d", got)
	}
	if !m.BrowserPremiumActive() {
		t.Fatal("returning to the slot resumes the premium lane")
	}
	if m.BrowserLaneSuspended() {
		t.Fatal("the thaw clears the keep-alive posture")
	}
	if got := m.BrowserLanePid(); got != pid {
		t.Fatalf("the flip's PID never changes: %d → %d", pid, got)
	}
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("the resume THAWS — never a respawn: log %d, want 1", got)
	}

	// the frozen/thawed child never leaks past the test (the quit path).
	m.browser.Close()
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("Close reaps the child: kill(%d, 0) = %v", pid, err)
	}
}

// TestBrowserLaneLiveFallback — (d): the fake dies inside the 300ms
// early-exit window — the pane's poll ride lands the text fallback
// THROUGH THE APP: the fetched page (warm — never re-fetched) + the exact
// dim note; the no-flap latch keeps a re-open on the text lane.
func TestBrowserLaneLiveFallback(t *testing.T) {
	pinBrowserLaneEnv(t)
	logPath := plantFakeTerminalBrowser(t, "die")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := laneFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})

	m = waitLaneDropped(t, m) // the View poll observes the early death
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "lane fell back"})
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{
		"zenbu exited (1) — falling back to text mode", // the EXACT dim note
		"▸ file:///",          // the text location bar is back (ansi-clipped at the slot width)
		"The Fixture Gazette", // the fetch rode under the embed — the page is warm
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the app-level fallback frame carries %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "zenbu terminal-browser ·") {
		t.Fatalf("the fallback drops the premium strip:\n%s", frame)
	}

	// the no-flap latch: re-opening the fell-back url never re-spawns.
	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("a fell-back url stays text: spawn log %d, want 1", got)
	}
	if m.BrowserPremiumActive() {
		t.Fatal("a fell-back url never goes premium again")
	}
}

// TestBrowserLaneLiveResize — (e): a WindowSizeMsg resize SIGWINCHes the
// live child to the slot's exact geometry (the LEFT slot is narrower than
// the old right-pane tab — asserted at REAL slot sizes).
func TestBrowserLaneLiveResize(t *testing.T) {
	pinBrowserLaneEnv(t)
	plantFakeTerminalBrowser(t, "sleep")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := laneFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")

	cols, rows, ok := m.browser.LaneSessionSize()
	if !ok {
		t.Fatal("the premium session is live")
	}
	if wantC, wantR := m.floorW, m.middleH-1-2; cols != wantC || rows != wantR {
		t.Fatalf("the spawn sized the PTY to the slot body: %dx%d, want %dx%d (floorW-2 rows)", cols, rows, wantC, wantR)
	}

	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	cols, rows, ok = m.browser.LaneSessionSize()
	if !ok {
		t.Fatal("the premium session survives the resize")
	}
	if wantC, wantR := m.floorW, m.middleH-1-2; cols != wantC || rows != wantR {
		t.Fatalf("the resize SIGWINCHes the child to the slot: %dx%d, want %dx%d", cols, rows, wantC, wantR)
	}
	m.browser.Close() // the live child never leaks past the test
}

// TestBrowserLaneLiveKillSwitch — the universal default THROUGH THE APP:
// with the lane off, /open never spawns (the log stays empty) and the
// frame is the pure pre-lane text viewer.
func TestBrowserLaneLiveKillSwitch(t *testing.T) {
	pinBrowserLaneEnv(t)
	t.Setenv(panels.BrowserLaneOffEnv, "1")
	logPath := plantFakeTerminalBrowser(t, "sleep")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + laneFixtureURL(t)})

	if got := laneSpawnCount(t, logPath); got != 0 {
		t.Fatalf("the kill-switched lane never spawns: log %d", got)
	}
	if m.BrowserPremiumActive() {
		t.Fatal("the kill-switched lane never goes premium")
	}
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{"▸ file:///", "The Fixture Gazette", "· ctrl+b"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the text-lane frame carries %q:\n%s", want, frame)
		}
	}
	for _, never := range []string{" zenbu ", "zenbu terminal-browser"} {
		if strings.Contains(frame, never) {
			t.Fatalf("the text lane never wears lane chrome %q:\n%s", never, frame)
		}
	}
}

// TestBrowserLaneLiveNoLeak — the office's own quit path seals the lane:
// ctrl+c runs closeBrowser → the lane's group-kill + bounded reap — the
// premium child never leaks past the office exit, and a sealed lane never
// re-spawns. (The process-level no-leak proof against the same real PTY
// seam lives in panels' TestBrowserLaneReapReal.)
func TestBrowserLaneLiveNoLeak(t *testing.T) {
	pinBrowserLaneEnv(t)
	logPath := plantFakeTerminalBrowser(t, "sleep")
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := laneFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("setup: one live spawn, got %d", got)
	}

	// the quit path: ctrl+c → closeBrowser → the lane's bounded reap.
	_ = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if m.BrowserPremiumActive() {
		t.Fatal("the quit path seals the lane")
	}
	// the controller is sealed: a second Close is a no-op and a fresh Open
	// never re-spawns (the text-lane fetch may still ride — the lane stays
	// closed).
	m.browser.Close()
	if cmd := m.browser.Open(raw); cmd != nil {
		cmd()
	}
	if m.BrowserPremiumActive() {
		t.Fatal("a sealed lane never re-spawns")
	}
	if got := laneSpawnCount(t, logPath); got != 1 {
		t.Fatalf("no spawn happens past the seal: log %d, want 1", got)
	}
}
