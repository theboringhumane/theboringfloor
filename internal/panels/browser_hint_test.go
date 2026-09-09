// browser_hint_test.go — the text lane's "why" row: the reasoned lane
// resolve's class matrix (pure, shell-out-free), the starter card wearing
// the exact dim hint per class, premium painting NO hint anywhere, the
// hint persisting under the location bar after a text-lane open (never
// scrolling away with the body), the pane-creation memoization contract
// (a later env/PATH flip never rewrites a live pane's lane story), and
// the ansi-truncation budget at narrow widths.
package panels

import (
	"testing"
)

// pinLaneDetectEnv — the hermetic detect-layer checklist for the hint
// tests (pinKittyEnv's shape; kitty=false stubs the iTerm2 family — a
// terminal that can NEVER host the embedded browser). Kill-switches
// cleared AND the wave-85 opt-in flag cleared (the default-off world —
// cases wanting premium pin it themselves). The binary probe rides
// pinLaneLook.
func pinLaneDetectEnv(t *testing.T, kitty bool) {
	t.Helper()
	if kitty {
		t.Setenv("TERM_PROGRAM", "ghostty")
	} else {
		t.Setenv("TERM_PROGRAM", "iTerm.app")
	}
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM_VERSION", "")
	t.Setenv("WEZTERM_UNIX_SOCKET", "")
	t.Setenv("VSCODE_PID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv(BrowserLaneOffEnv, "")
	t.Setenv(TerminalBrowserOffEnv, "")
	t.Setenv(BrowserLaneOptInEnv, "")
}

// pinLaneLook swaps the binary probe (found = the fixture PATH hit).
func pinLaneLook(t *testing.T, found bool) {
	t.Helper()
	old := zenbuLookPath
	if found {
		zenbuLookPath = lookFound
	} else {
		zenbuLookPath = lookMissing
	}
	t.Cleanup(func() { zenbuLookPath = old })
}

// the frozen hint copies (the pane renders these VERBATIM, dimmed +
// ansi-truncated to the pane width).
const (
	hintNoBinary   = "text lane — terminal-browser not on PATH · full rendering: github.com/zenbu-labs/terminal-browser (or re-run the office installer)"
	hintNoTerminal = "text lane — this terminal can't host the embedded browser (kitty/ghostty only)"
	hintKillOwn    = "text lane — " + BrowserLaneOffEnv + "=1 set; unset it for the embedded browser"
	hintKillWave70 = "text lane — " + TerminalBrowserOffEnv + "=1 set; unset it for the embedded browser"
	hintOptInOff   = "text lane — the embedded browser is opt-in: " + BrowserLaneOptInEnv + "=1 to enable it"
)

// TestBrowserLaneReasonMatrix — the reasoned resolve's pure table: binary
// present/absent × kitty/non-kitty × each kill-switch spelling × the
// wave-85 opt-in flag (+ the resolve's own precedence: kill-switch →
// terminal → binary → opt-in). The lane half must match
// ResolveBrowserLaneFrom's historic table exactly.
