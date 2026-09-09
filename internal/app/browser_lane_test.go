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

	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/panels"
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
