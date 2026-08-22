// session_picker_test.go — proofs for the /session picker card:
//
//	(a) the LOADING state renders "listing sessions…" and enter is a
//	    no-op while no rows have landed;
//	(b) rows render title + "age · N msgs · short" meta, the CURRENT
//	    session marked " · current", the footer hint present, and the
//	    card splices as an overlay (row count never exceeds SetSize, the
//	    disabled textarea still renders under it);
//	(c) typing NARROWS (case-insensitive substring over title + id),
//	    backspace edits, "(no matches)" shows on a dead filter;
//	(d) ↑/↓ move (wrapping), enter ACCEPTS the highlighted row's full
//	    id through onSessionPick;
//	(e) esc cancels through onSessionCancel with zero pick side effects;
//	(f) while a permission popover owns the slot the picker yields its
//	    keys (a parked turn outranks browsing) and the picker renders
//	    UNDER it.
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// sessHarness — a chat with the picker seams recorded plus the picker
// itself opened (mirroring the app's wiring: the handlers close over
// tea.Msg-bearing cmds, not bare nils).
func sessHarness() (c *Chat, picks *[]string, cancels *int) {
	c = NewChat(nil)
	c.SetSize(80, 30)
	picks = &[]string{}
	cancels = new(int)
	c.SetSessionPickerHandlers(func(id string) tea.Cmd {
		*picks = append(*picks, id)
		return func() tea.Msg { return nil }
	}, func() tea.Cmd {
		(*cancels)++
		return func() tea.Msg { return nil }
	})
	c.OpenSessionPicker()
	return c, picks, cancels
}

// sessType feeds printable text one rune at a time.
func sessType(c *Chat, s string) {
	for _, r := range s {
		c.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

func sessRows() []SessionPickRow {
	return []SessionPickRow{
		{ID: "ses-alpha-1", Title: "alpha brief", Age: "3m", Messages: 12, ShortID: "ses-alph", Current: false},
		{ID: "ses-beta-22", Title: "beta review", Age: "2h", Messages: 47, ShortID: "ses-beta", Current: true},
		{ID: "ses-gamma-3", Title: "gamma notes", Age: "5d", Messages: -1, ShortID: "ses-gamm", Current: false},
	}
}

// (a) LOADING: the placeholder row renders, no accept can fire.
func TestSessPickerLoading(t *testing.T) {
	c, picks, _ := sessHarness()
	view := ansi.Strip(c.View())
	if !strings.Contains(view, "SESSIONS") || !strings.Contains(view, "listing sessions…") {
		t.Fatalf("the loading state must render header + placeholder:\n%s", view)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 0 {
		t.Fatalf("enter must be a no-op while loading, got picks %v", *picks)
	}
}

// (b) rows render sorted + marked; the card splices as an overlay.
func TestSessPickerRendersRows(t *testing.T) {
	c, _, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())
	view := ansi.Strip(c.View())
	for _, want := range []string{
		"SESSIONS",
		"alpha brief", "3m · 12 msgs · ses-alph",
		"beta review", "2h · 47 msgs · ses-beta · current",
		"gamma notes", "5d · ? msgs · ses-gamm", // unknown count renders honestly
		"3/3", sessHint,
		"╭", "╰",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("picker row furniture missing %q:\n%s", want, view)
		}
	}
	// the card is an overlay: the panel row budget NEVER grows.
	if n := len(strings.Split(c.View(), "\n")); n > 30 {
		t.Fatalf("the overlay must never grow the panel past its 30 rows, got %d", n)
	}
	// picking while rows are present fires the highlighted row's FULL id.
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if line := ansi.Strip(c.View()); !strings.Contains(line, "SESSIONS") {
		t.Fatalf("a panel-side enter must only FIRE the seam — the model closes the card (still open here)")
	}
}

// (c) narrowing: substring over title + full id, backspace edits, the
// "(no matches)" row shows on a dead filter.
func TestSessPickerNarrows(t *testing.T) {
	c, picks, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())

	sessType(c, "ALPHA")
	if n := len(c.sessPick.filtered); n != 1 {
		t.Fatalf("an uppercase fragment must still match (case-insensitive), got %d rows", n)
	}
	if got := ansi.Strip(c.View()); !strings.Contains(got, "filter: ALPHA") || !strings.Contains(got, "1/3") {
		t.Fatalf("the filter row + narrowed badge must render:\n%s", got)
	}
	// enter accepts the only narrowed row
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "ses-alpha-1" {
		t.Fatalf("enter must accept the narrowed row's full id, got %v", *picks)
	}
	// backspace edits the filter; more typing can dead-end it
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if n := len(c.sessPick.filtered); n != 3 {
		t.Fatalf("clearing the filter must restore every row, got %d", n)
	}
	sessType(c, "zzzzz")
	if got := ansi.Strip(c.View()); !strings.Contains(got, "(no matches)") {
		t.Fatalf("a dead filter must show (no matches):\n%s", got)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 {
		t.Fatalf("enter must stay a no-op on an empty filter result, got %v", *picks)
	}
	// and not one key ever reached the main textarea
	if got := c.ta.Value(); got != "" {
		t.Fatalf("the picker owns every key — the textarea must stay empty, got %q", got)
	}
}

// (d) movement wraps; enter accepts the highlighted row.
func TestSessPickerMoveAccept(t *testing.T) {
	c, picks, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())

	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // wraps to the LAST row
	if got := c.sessPick.sel; got != 2 {
		t.Fatalf("up on the first row must wrap to the last, sel=%d", got)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "ses-gamma-3" {
		t.Fatalf("enter must accept the wrapped-to row, got %v", *picks)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})) // two wrap-around steps back to sel=1
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 2 || (*picks)[1] != "ses-beta-22" {
		t.Fatalf("down-down must land on the middle row, got %v", *picks)
	}
}

// (e) esc cancels through onSessionCancel — the panel fires the seam and
// STAYS open; the model's answer (CloseSessionPicker) is what ends it
// (single-writer rule), so a cancel never picks anything.
func TestSessPickerEscCancels(t *testing.T) {
	c, picks, cancels := sessHarness()
	c.SetSessionPickerRows(sessRows())
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 1 {
		t.Fatalf("esc must fire onSessionCancel exactly once, got %d", *cancels)
	}
	if len(*picks) != 0 {
		t.Fatalf("esc cancels with zero pick side effects, got %v", *picks)
	}
	if !c.SessionPickerOpen() {
		t.Fatalf("the panel must stay open until the MODEL closes it (single writer)")
	}
	c.CloseSessionPicker()
	if c.SessionPickerOpen() {
		t.Fatalf("CloseSessionPicker must end the card")
	}
	if got := ansi.Strip(c.View()); strings.Contains(got, "SESSIONS") {
		t.Fatalf("a closed picker must not render:\n%s", got)
	}
}

// (f) a permission popover popping over the picker owns its keys (the
// picker's filter/cursor must not move) and its pixels (the centered
// cards overlap — the alert float splices on top where they meet); the
// layout never jumps, and once the float clears the picker is fully
// visible and owns again.
func TestSessPickerYieldsToPermission(t *testing.T) {
	c, picks, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())

	c.SetPermission(&PermissionView{ID: "per-1", ToolName: "bash", Summary: "go build ./..."})
	// with the float up, "down" walks the PERM menu — never the picker.
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if c.permSel != 1 {
		t.Fatalf("down must walk the permission menu while it owns the slot, permSel=%d", c.permSel)
	}
	if c.sessPick.sel != 0 {
		t.Fatalf("the picker must YIELD its keys under a permission float, sel=%d", c.sessPick.sel)
	}
	sessType(c, "x") // not a perm choice key — must NOT reach the picker
	if c.sessPick.filter != "" {
		t.Fatalf("the picker's filter must not move while a permission owns the slot: %q", c.sessPick.filter)
	}
	// the permission owns the pixels where the centered cards meet; the
	// overlay still never grows the panel row budget.
	view := ansi.Strip(c.View())
	if !strings.Contains(view, "PERMISSION REQUIRED") {
		t.Fatalf("the permission card must render over the picker:\n%s", view)
	}
	if n := len(strings.Split(c.View(), "\n")); n > 30 {
		t.Fatalf("two floats must still never grow the panel past its 30 rows, got %d", n)
	}
	// the float clears → the picker is fully visible and owns again.
	c.SetPermission(nil)
	if got := ansi.Strip(c.View()); !strings.Contains(got, "SESSIONS") {
		t.Fatalf("with the float gone the picker must render again:\n%s", got)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "ses-alpha-1" {
		t.Fatalf("after the float clears the picker must own enter again, got %v", *picks)
	}
}
