// session_picker_exec_test.go — the /session picker's EXEC-REPLACE accept
// surface from the model level (fakes only):
//
//	(a) ACCEPT on a different session captures the exec intent on the
//	    model (ExecRequest == the accepted id), persists the pin into
//	    session.json SYNCHRONOUSLY and stamped with the ACCEPTED id,
//	    lands the frozen closing row, and returns a quit cmd — while
//	    ResumeOffice is NEVER called and the live primary stays put;
//	(b) the BUSY refusal captures NOTHING (no exec intent, no quit cmd);
//	(c) the same-id NO-OP captures nothing;
//	(d) the seam-absent FALLBACK (can list, cannot re-anchor) captures
//	    nothing and lands the static-summary note.
package app

import (
	"context"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// listOnlyBackend — a LIVE backend that can LIST sessions but has no
// resume seam: the accept's graceful-degradation leg (static summary +
// the dim "cannot re-anchor" note, never an exec intent).
type listOnlyBackend struct {
	recBackend
	primary string
	rows    []state.SessionRow
}

func (b *listOnlyBackend) Mode() state.Mode          { return state.ModeLive }
func (b *listOnlyBackend) PrimaryOverride(id string) {}
func (b *listOnlyBackend) PrimaryID() string         { return b.primary }
func (b *listOnlyBackend) ListSessions(ctx context.Context) ([]state.SessionRow, error) {
	return b.rows, nil
}

// driveAccept presses enter on the picker's CURRENT row through the real
// key path, feeds the sessionPickMsg ferry back by hand, and returns the
// model plus EVERY leaf msg (enter tree + accept tree) so quit-cmd
// presence is assertable both ways. (leafMsgs is safe here: the accept
// tree carries ferry + tea.Quit msgs only — no sleeping ticks.)
func driveAccept(t *testing.T, m Model) (Model, []tea.Msg) {
	t.Helper()
	nm, enterCmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = nm.(Model)
	var leaves []tea.Msg
	for _, leaf := range leafMsgs(enterCmd) {
		if pick, ok := leaf.(sessionPickMsg); ok {
			nm, acceptCmd := m.Update(pick)
			m = nm.(Model)
			leaves = append(leaves, leafMsgs(acceptCmd)...)
			continue
		}
		leaves = append(leaves, leaf)
	}
	return m, leaves
}

// (a) ACCEPT captures the exec intent, persists the ACCEPTED pin sync,
// returns the quit cmd, and leaves the resume seam + live primary alone.
func TestSessionPickerAcceptCapturesExecIntent(t *testing.T) {
	scratchHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})) // row 1 = ses-beta-older

	m, leaves := driveAccept(t, m)

	if got := m.ExecRequest(); got != "ses-beta-older" {
		t.Fatalf("accept must capture the exec intent (id + nothing else), got %q", got)
	}
	if !hasQuitLeaf(leaves) {
		t.Fatalf("accept must return a quit cmd (main exec-replaces post-Run), leaves=%v", leaves)
	}
	if len(b.resumed) != 0 {
		t.Fatalf("ResumeOffice is NEVER called under the exec-replace contract: %v", b.resumed)
	}
	if b.primary != "ses-alpha-new" {
		t.Fatalf("the live primary must NOT move in-app (the relaunch swaps it), got %q", b.primary)
	}
	if !chatHas(m, "closing — relaunching as `theboringfloor -s ses-beta-older`") {
		t.Fatalf("the frozen closing row must land as transcript history: %v", chatTexts(m))
	}
	// the pin lands in session.json SYNCHRONOUSLY (a quit right after
	// must never outrun it), stamped with the ACCEPTED id.
	sf, ok := LoadSession(cwd)
	if !ok {
		t.Fatalf("the accept must persist session.json synchronously")
	}
	if sf.PrimaryID != "ses-beta-older" {
		t.Fatalf("session.json must be stamped with the ACCEPTED pin, got %q", sf.PrimaryID)
	}
}

// (b) BUSY refusal: a boss turn in flight captures NOTHING — no exec
// intent, no quit cmd, no seam call; the frozen refusal notice lands.
func TestSessionPickerBusyCapturesNothing(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})
	m = runMsg(t, m, slashMsg{text: "/session"})
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))

	m, leaves := driveAccept(t, m)

	if got := m.ExecRequest(); got != "" {
		t.Fatalf("a busy refusal must capture NOTHING, got exec intent %q", got)
	}
	if hasQuitLeaf(leaves) {
		t.Fatalf("a busy refusal must never return a quit cmd, leaves=%v", leaves)
	}
	if len(b.resumed) != 0 {
		t.Fatalf("the seam must stay untouched on a refusal: %v", b.resumed)
	}
	if !chatHas(m, "boss is busy — /stop or wait, then /session again") {
		t.Fatalf("the frozen busy-block notice must land: %v", chatTexts(m))
	}
}

// (c) same-id NO-OP: captures nothing, closes the picker, dims the
// already-on notice.
func TestSessionPickerAcceptCurrentCapturesNothing(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})
	// sel 0 IS the current session — enter straight away.

	m, leaves := driveAccept(t, m)

	if got := m.ExecRequest(); got != "" {
		t.Fatalf("a no-op accept must capture NOTHING, got exec intent %q", got)
	}
	if hasQuitLeaf(leaves) {
		t.Fatalf("a no-op accept must never return a quit cmd, leaves=%v", leaves)
	}
	if !chatHas(m, "already on session ses-alpha-new") {
		t.Fatalf("the already-on notice must land: %v", chatTexts(m))
	}
}

// (d) seam-absent FALLBACK: can list, cannot re-anchor → the static
// summary + the dim note; nothing captured, no quit cmd.
func TestSessionPickerSeamAbsentCapturesNothing(t *testing.T) {
	scratchHome(t)
	b := &listOnlyBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})
	if m.chat == nil || !m.chat.SessionPickerOpen() {
		t.Fatalf("precondition: the listing seam opens the picker")
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))

	m, leaves := driveAccept(t, m)

	if got := m.ExecRequest(); got != "" {
		t.Fatalf("the fallback must capture NOTHING, got exec intent %q", got)
	}
	if hasQuitLeaf(leaves) {
		t.Fatalf("the fallback must never return a quit cmd, leaves=%v", leaves)
	}
	if !chatHas(m, "session picker unavailable: this backend cannot re-anchor live") {
		t.Fatalf("the static-summary fallback note must land: %v", chatTexts(m))
	}
	if !strings.Contains(chatTexts(m)[len(chatTexts(m))-1], "session: ses-alpha-new (primary)") {
		t.Fatalf("the static /session summary rides the fallback: %v", chatTexts(m))
	}
}
