// browser_hint_test.go — the text lane's "why" row: the reasoned lane
// resolve's class matrix (pure, shell-out-free), the starter card wearing
// the exact dim hint per class, premium painting NO hint anywhere, the
// hint persisting under the location bar after a text-lane open (never
// scrolling away with the body), the pane-creation memoization contract
// (a later env/PATH flip never rewrites a live pane's lane story), and
// the ansi-truncation budget at narrow widths.
package panels

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// pinLaneDetectEnv — the hermetic detect-layer checklist for the hint
// tests (pinKittyEnv's shape; kitty=false stubs the iTerm2 family — a
// terminal that can NEVER host the embedded browser). Kill-switches
// cleared; the binary probe rides pinLaneLook.
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
)

// TestBrowserLaneReasonMatrix — the reasoned resolve's pure table: binary
// present/absent × kitty/non-kitty × each kill-switch spelling (+ the
// resolve's own precedence: kill-switch → terminal → binary). The lane
// half must match ResolveBrowserLaneFrom's historic table exactly.
func TestBrowserLaneReasonMatrix(t *testing.T) {
	ghostty := map[string]string{"TERM_PROGRAM": "ghostty", "TERM": "xterm-256color"}
	iterm := map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM": "xterm-256color"}
	cases := []struct {
		name       string
		env        map[string]string
		look       func(string) (string, error)
		wantLane   BrowserLane
		wantReason BrowserLaneReason
		wantVar    string
	}{
		{"premium: kitty host + binary + no kill-switch", ghostty, lookFound, BrowserLaneZenbu, BrowserLanePremium, ""},
		{"binary missing on a kitty host", ghostty, lookMissing, BrowserLaneText, BrowserLaneNoBinary, ""},
		{"terminal unsupported (iTerm2), binary present", iterm, lookFound, BrowserLaneText, BrowserLaneNoTerminal, ""},
		{"terminal unsupported (iTerm2), binary absent — the terminal gate wins precedence", iterm, lookMissing, BrowserLaneText, BrowserLaneNoTerminal, ""},
		{"terminal unsupported: tmux folds out conservatively", map[string]string{"TERM_PROGRAM": "ghostty", "TMUX": "/tmp/tmux-1000/default,1,0"}, lookFound, BrowserLaneText, BrowserLaneNoTerminal, ""},
		{"kill-switch: own spelling, kitty + binary", map[string]string{"TERM_PROGRAM": "ghostty", BrowserLaneOffEnv: "1"}, lookFound, BrowserLaneText, BrowserLaneKillSwitch, BrowserLaneOffEnv},
		{"kill-switch: wave-70 spelling, kitty + binary", map[string]string{"TERM_PROGRAM": "ghostty", TerminalBrowserOffEnv: "1"}, lookFound, BrowserLaneText, BrowserLaneKillSwitch, TerminalBrowserOffEnv},
		{"kill-switch beats a non-kitty terminal (the resolve's precedence)", map[string]string{"TERM_PROGRAM": "iTerm.app", BrowserLaneOffEnv: "1"}, lookMissing, BrowserLaneText, BrowserLaneKillSwitch, BrowserLaneOffEnv},
		{"kill-switch '0' is NOT armed", map[string]string{"TERM_PROGRAM": "ghostty", BrowserLaneOffEnv: "0"}, lookFound, BrowserLaneZenbu, BrowserLanePremium, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lane, reason, killVar := ResolveBrowserLaneReasonFrom(laneEnv(tc.env), tc.look)
			if lane != tc.wantLane || reason != tc.wantReason || killVar != tc.wantVar {
				t.Fatalf("ResolveBrowserLaneReasonFrom = (%s, %s, %q), want (%s, %s, %q)",
					lane, reason, killVar, tc.wantLane, tc.wantReason, tc.wantVar)
			}
			// the lane-only API is the same table's lane half (the
			// pre-hint callers' contract never moved).
			if got := ResolveBrowserLaneFrom(laneEnv(tc.env), tc.look); got != tc.wantLane {
				t.Fatalf("ResolveBrowserLaneFrom = %s, want %s", got, tc.wantLane)
			}
		})
	}
}

// TestBrowserHintStarterCardPerClass — the idle pane wears the EXACT dim
// hint for its resolve class, right under the starter card's bar.
func TestBrowserHintStarterCardPerClass(t *testing.T) {
	cases := []struct {
		name string
		pin  func(t *testing.T)
		want string
	}{
		{"binary missing", func(t *testing.T) {
			pinLaneDetectEnv(t, true)
			pinLaneLook(t, false)
		}, hintNoBinary},
		{"terminal unsupported", func(t *testing.T) {
			pinLaneDetectEnv(t, false)
			pinLaneLook(t, true)
		}, hintNoTerminal},
		{"kill-switch: own spelling", func(t *testing.T) {
			pinLaneDetectEnv(t, true)
			pinLaneLook(t, true)
			t.Setenv(BrowserLaneOffEnv, "1")
		}, hintKillOwn},
		{"kill-switch: wave-70 spelling", func(t *testing.T) {
			pinLaneDetectEnv(t, true)
			pinLaneLook(t, true)
			t.Setenv(TerminalBrowserOffEnv, "1")
		}, hintKillWave70},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.pin(t) // BEFORE the pane builds: creation-time is the resolve contract
			b := NewBrowser()
			b.SetSize(150, 12) // wide enough for the full untruncated copy
			view := ansi.Strip(b.View())
			if !strings.Contains(view, browserStarterCard) {
				t.Fatalf("the idle pane still shows the starter card:\n%s", view)
			}
			if !strings.Contains(view, tc.want) {
				t.Fatalf("the starter card wears the exact hint:\nwant %q\nview:\n%s", tc.want, view)
			}
			// the hint rides UNDER the bar (row 1), dim — never an error.
			lines := strings.Split(view, "\n")
			if len(lines) < 2 || !strings.Contains(lines[1], "text lane —") {
				t.Fatalf("the hint row rides under the location bar (row 1):\n%s", view)
			}
		})
	}
}

// TestBrowserHintPremiumAbsent — a resolving host paints NO hint row:
// not on the idle card, not under the live premium strip.
func TestBrowserHintPremiumAbsent(t *testing.T) {
	pinKittyEnv(t) // kitty stub + lookFound + both kill-switches cleared
	made := fakeSpawnPins(t)
	b := NewBrowser()
	b.SetSize(64, 16)
	if idle := ansi.Strip(b.View()); strings.Contains(idle, "text lane —") {
		t.Fatalf("premium hosts paint no hint row on the starter card:\n%s", idle)
	}
	b.fetchFn = func(string) (*Page, error) { return navPage("https://a.dev/x", "Xray"), nil }
	cmd := b.Open("https://a.dev/x")
	if cmd == nil {
		t.Fatal("Open must produce the fetch cmd")
	}
	b.Update(cmd())
	if len(*made) != 1 || !b.PremiumActive() {
		t.Fatalf("the resolving host spawned the premium embed: made=%d active=%v", len(*made), b.PremiumActive())
	}
	if view := ansi.Strip(b.View()); strings.Contains(view, "text lane —") {
		t.Fatalf("the premium frame paints no hint row:\n%s", view)
	}
}

// TestBrowserHintPersistsAfterOpen — the text-lane page keeps the dim
// hint pinned under the location bar: the body scrolls, the hint never
// does.
func TestBrowserHintPersistsAfterOpen(t *testing.T) {
	pinLaneDetectEnv(t, true) // kitty-capable…
	pinLaneLook(t, false)     // …but the binary misses → the actionable class
	b := NewBrowser()
	b.SetSize(150, 14)
	b.fetchFn = func(string) (*Page, error) { return navPage("https://a.dev/x", "Xray"), nil }
	cmd := b.Open("https://a.dev/x")
	if cmd == nil {
		t.Fatal("Open must produce the fetch cmd")
	}
	b.Update(cmd())

	view := ansi.Strip(b.View())
	for _, want := range []string{hintNoBinary, "▸ https://a.dev/x", "Xray filler 00"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the opened text-lane frame carries %q:\n%s", want, view)
		}
	}
	// the hint is the row UNDER the bar…
	lines := strings.Split(view, "\n")
	if len(lines) < 2 || !strings.Contains(lines[1], "text lane — terminal-browser not on PATH") {
		t.Fatalf("the hint persists under the location bar after the open:\n%s", view)
	}
	// …and it never scrolls away with the body.
	b.Update(browserKey("pgdown"))
	view = ansi.Strip(b.View())
	lines = strings.Split(view, "\n")
	if len(lines) < 2 || !strings.Contains(lines[1], "text lane — terminal-browser not on PATH") {
		t.Fatalf("the hint row stays pinned past a body scroll:\n%s", view)
	}
	if !strings.Contains(view, "Xray filler") {
		t.Fatalf("the text page keeps painting under the hint:\n%s", view)
	}
}

// TestBrowserHintPinnedAtPaneCreation — the resolve reads env+PATH ONCE
// at pane creation: a binary appearing (or a kill-switch lifting) AFTER
// the pane built never rewrites the live pane's lane story.
func TestBrowserHintPinnedAtPaneCreation(t *testing.T) {
	pinLaneDetectEnv(t, true)
	pinLaneLook(t, false) // the binary misses AT CREATION
	b := NewBrowser()
	pinLaneLook(t, true) // …it "appears" later — the pane's pin must hold
	b.SetSize(150, 12)
	if got := ansi.Strip(b.View()); !strings.Contains(got, "text lane — terminal-browser not on PATH") {
		t.Fatalf("the pane-creation verdict is memoized for the pane's life:\n%s", got)
	}
	if got := b.lane.Lane(); got != BrowserLaneText {
		t.Fatalf("the lane pin holds past the later PATH flip: got %s", got)
	}
}

// TestBrowserHintTruncatesToPaneWidth — the hint is ONE row,
// ansi-truncated to the pane width (never wrapped, never an overflow).
func TestBrowserHintTruncatesToPaneWidth(t *testing.T) {
	pinLaneDetectEnv(t, true)
	pinLaneLook(t, false)
	b := NewBrowser()
	b.SetSize(48, 10)
	view := ansi.Strip(b.View())
	for i, ln := range strings.Split(view, "\n") {
		if w := lipgloss.Width(ln); w > 48 {
			t.Fatalf("row %d is %d cells, budget 48: %q", i, w, ln)
		}
	}
	if !strings.Contains(view, "text lane — terminal-browser not on") {
		t.Fatalf("the truncated hint still names the class:\n%s", view)
	}
	if strings.Contains(view, "office installer") {
		t.Fatalf("the 48-cell pane must truncate the copy's tail:\n%s", view)
	}
}
