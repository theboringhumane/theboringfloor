// tabs_test.go — behavior proofs for the sidebar tab strip with SEVEN tabs
// (chat, terminal, agents, board, mail, activity, git — git ships LAST at
// index 6 because the activity index 5 is hardcoded app-side):
//
//   - Next/Prev wrap around the full ring (6→0 and 0→6), ActiveIndex
//     defaults to 0, SetActiveByTitle("git") lands on index 6 and a miss
//     never moves the selection.
//   - The tab bar degrades density tiers so the DEFAULT 44-col sidebar
//     shows all seven tabs as single letters — the padLetters tier
//     " c t a b m x g" (14 cells) — instead of truncating "git" to "gi"
//     (full-title tiers cost 72/58/45 cells there, all over budget),
//     while a wide sidebar keeps the full padNumbered labels; and no
//     rendered line ever exceeds the strip width.
package panels

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	state "github.com/theboringhumane/theboringoffice/internal/state"
)

// stubTab — the smallest Tab implementation: a fixed canonical title, no
// size/state bookkeeping, and one line of content so the bordered box has
// body text.
type stubTab struct{ title string }

func (s stubTab) Title() string                { return s.title }
func (s stubTab) SetSize(w, h int)             {}
func (s stubTab) SetState(_ state.OfficeState) {}
func (s stubTab) View() string                 { return s.title + " pane" }

// sevenTabs wires the strip exactly as the app does: git LAST (index 6).
func sevenTabs() *Tabs {
	return NewTabs(
		stubTab{"chat"}, stubTab{"terminal"}, stubTab{"agents"},
		stubTab{"board"}, stubTab{"mail"}, stubTab{"activity"}, stubTab{"git"},
	)
}

func TestTabsSevenWrapAround(t *testing.T) {
	tb := sevenTabs()
	if got := tb.ActiveIndex(); got != 0 {
		t.Fatalf("default ActiveIndex = %d, want 0", got)
	}
	tb.Prev()
	if got := tb.ActiveIndex(); got != 6 {
		t.Fatalf("Prev from 0 → %d, want 6 (wrap)", got)
	}
	tb.Next()
	if got := tb.ActiveIndex(); got != 0 {
		t.Fatalf("Next from 6 → %d, want 0 (wrap)", got)
	}
	// walk the whole ring forward: 0 1 2 3 4 5 6 and back to 0
	tb2 := sevenTabs()
	for want := 1; want <= 6; want++ {
		tb2.Next()
		if got := tb2.ActiveIndex(); got != want {
			t.Fatalf("Next #%d → %d, want %d", want, got, want)
		}
	}
	tb2.Next()
	if got := tb2.ActiveIndex(); got != 0 {
		t.Fatalf("8th Next → %d, want 0 (full ring)", got)
	}
}

func TestTabsSetActiveByTitleGit(t *testing.T) {
	tb := sevenTabs()
	if !tb.SetActiveByTitle("git") {
		t.Fatal(`SetActiveByTitle("git") = false, want true`)
	}
	if got := tb.ActiveIndex(); got != 6 {
		t.Fatalf(`SetActiveByTitle("git") → index %d, want 6`, got)
	}
	if tb.SetActiveByTitle("nope") {
		t.Fatal(`SetActiveByTitle("nope") = true, want false`)
	}
	if got := tb.ActiveIndex(); got != 6 {
		t.Fatalf("a missed SetActiveByTitle moved active to %d, want still 6", got)
	}
}

// barLine renders the strip at w×h, fails if any rendered line overflows w
// printable cells, and returns the tab-bar row (first line), ANSI-stripped
// and right-trimmed of join padding.
func barLine(t *testing.T, tb *Tabs, w, h int) string {
	t.Helper()
	tb.SetSize(w, h)
	lines := strings.Split(tb.View(), "\n")
	if len(lines) < 2 {
		t.Fatalf("View() must render a bar row plus the bordered box, got %d lines", len(lines))
	}
	for i, ln := range lines {
		if got := lipgloss.Width(ln); got > w {
			t.Fatalf("line %d is %d cells wide, budget %d: %q", i, got, w, ansi.Strip(ln))
		}
	}
	return strings.TrimRight(ansi.Strip(lines[0]), " ")
}

// TestTabsViewWidth44PadLettersTier pins the 44-col DEFAULT sidebar: the
// full-title tiers (72/58/45 cells) all overflow, so the bar must resolve
// to padLetters — " c t a b m x g" — keeping git alive as "g".
func TestTabsViewWidth44PadLettersTier(t *testing.T) {
	bar := barLine(t, sevenTabs(), 44, 12)
	if want := " c t a b m x g"; bar != want {
		t.Fatalf("44-col bar = %q, want padLetters tier %q", bar, want)
	}
}

// TestTabsViewWidePadNumberedTier pins the wide-sidebar bar: at exactly the
// numbered width (72 cells) the strip keeps full " 1 chat … 7 git " labels.
func TestTabsViewWidePadNumberedTier(t *testing.T) {
	bar := barLine(t, sevenTabs(), 72, 12)
	// " 1 chat " + " " join + " 2 terminal " … → three cells between words.
	want := " 1 chat   2 terminal   3 agents   4 board   5 mail   6 activity   7 git"
	if bar != want {
		t.Fatalf("72-col bar = %q, want padNumbered tier %q", bar, want)
	}
}
