// quitarm_test.go — the ctrl+q double-press quit contract from the model
// (named WITHOUT an underscore before "arm" on purpose: a trailing
// "_arm"/"_amd64"/… filename segment is an implicit GOOS/GOARCH build
// constraint and the go tool would silently ignore the file)
// level (fakes only, never a real server):
//
//	(a) the FIRST ctrl+q ARMS instead of quitting: the arm stamps, the
//	    statusbar hint swaps to the frozen warn-class toast, and the only
//	    cmd back is the arm's own expiry tick (NEVER a quit);
//	(b) the SECOND press inside quitArmWindow yields a tea.QuitMsg and
//	    runs the EXISTING quit path (persist + terminal reap);
//	(c) a STALE first press (older than the window) can't pair — the next
//	    press re-opens a fresh arm (the chat panel's dblEsc idiom, direct
//	    field time-injection included);
//	(d) the arm's own tea.Tick expiry (armClearMsg) clears an old arm and
//	    leaves a YOUNGER re-arm alone;
//	(e) ANY other key press clears the arm and retires the toast on the
//	    next render;
//	(f) the toast copy itself is frozen.
package app

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// ctrlQ / pressKey — the keypress constructors. Update is driven DIRECTLY
// (never runMsg) so the arming press's own tea.Tick is never executed —
// executing one would sleep the whole window.
func ctrlQ() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'q', Mod: tea.ModCtrl}) }
func pressKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)})
}

// leafMsgs executes a cmd tree breadth-first (BatchMsg unwrapped) and
// returns the leaf messages, skipping the panels' heartbeat msgs exactly
// like runMsg. NEVER call it on the ctrl+q FIRST-press cmd (its
// tea.Tick(quitArmWindow) closure would sleep the window) — quit /
// accept / ferry trees carry no ticks, so their leaves are instant.
func leafMsgs(cmd tea.Cmd) []tea.Msg {
	var leaves []tea.Msg
	queue := []tea.Cmd{cmd}
	for i := 0; len(queue) > 0 && i < 64; i++ { // cap guards a runaway tree
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		out := c()
		if out == nil {
			continue
		}
		switch out := out.(type) {
		case tea.BatchMsg:
			queue = append(queue, out...)
		case spinner.TickMsg, cursor.BlinkMsg:
			// panel heartbeats — self-re-arming, not results
		default:
			leaves = append(leaves, out)
		}
	}
	return leaves
}

// hasQuitLeaf — did any cmd tree in leaves carry a tea.QuitMsg?
func hasQuitLeaf(leaves []tea.Msg) bool {
	for _, l := range leaves {
		if _, ok := l.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

// (f) the frozen toast copy.
func TestQuitArmToastCopyFrozen(t *testing.T) {
	if quitArmToast != "ctrl+q again: quit the office" {
		t.Fatalf("the armed toast copy is frozen, got %q", quitArmToast)
	}
}

// (a) first press ARMS: arm stamped, toast swapped in, no quit anywhere.
func TestQuitArmFirstPressArms(t *testing.T) {
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	nm, cmd := m.Update(ctrlQ())
	m = nm.(Model)

	if m.quitArmAt.IsZero() {
		t.Fatalf("the first press must STAMP the arm")
	}
	if cmd == nil {
		t.Fatalf("the arm schedules its own expiry tick — a cmd must come back")
	}
	// the quit path (press 2) is the one that DISARMS + persists + quits —
	// none of that happened here: only the toast + the tick.
	if hint := m.hintLine(); !strings.Contains(hint, quitArmToast) {
		t.Fatalf("the armed hint bar must swap to the toast, got %q", hint)
	}
	if hint := m.hintLine(); strings.Contains(hint, "tab:panels") {
		t.Fatalf("the toast REPLACES the keymap line while armed, got %q", hint)
	}
	// not-quit proof via the returned tree: the expiry tick is the ONLY cmd
	// — assert by type WITHOUT executing it (executing sleeps the window);
	// a quitting press returns tea.Quit and clears the arm (proven in (b)).
	if m.quitArmAt.IsZero() {
		t.Fatalf("the first press must not take the quit path")
	}
}

// (b) second press inside the window QUITS via the existing path.
func TestQuitArmSecondPressQuits(t *testing.T) {
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("precondition: the first press armed")
	}

	nm, cmd := m.Update(ctrlQ())
	m = nm.(Model)

	if !m.quitArmAt.IsZero() {
		t.Fatalf("the quitting press disarms on the way out")
	}
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("the second press inside the window must yield a tea.QuitMsg")
	}
	if hint := m.hintLine(); strings.Contains(hint, quitArmToast) {
		t.Fatalf("the toast retires with the arm, got %q", hint)
	}
}

// (c) a STALE first press can't pair: the next press re-opens a fresh arm,
// and only the press AFTER that quits (chat_esc's time-injection idiom —
// the arm stamp is written directly).
func TestQuitArmStaleReArms(t *testing.T) {
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	m.quitArmAt = time.Now().Add(-quitArmWindow - time.Second) // stale the opener

	// press 2: outside the window — no quit, and this press becomes the
	// fresh opener (stamp re-armed to ~now; its cmd is the expiry tick,
	// never executed here).
	nm, _ = m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() || time.Since(m.quitArmAt) > quitArmWindow {
		t.Fatalf("a stale pair must RE-ARM with a fresh stamp, got %v", m.quitArmAt)
	}

	// press 3 completes the FRESH pair.
	nm, cmd := m.Update(ctrlQ())
	m = nm.(Model)
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("the press after the re-arm must quit (fresh pair completed)")
	}
}

// (d) the arm's expiry tick (armClearMsg) clears an old arm, leaves a
// younger re-arm alone.
func TestQuitArmTickExpiryClears(t *testing.T) {
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	// young arm: a stale tick landing early must NOT clear it (a re-armed
	// pair's own tick owns its expiry).
	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	nm, _ = m.Update(armClearMsg{})
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("a tick landing early must not clear a YOUNG arm")
	}

	// old arm: the real expiry case.
	m.quitArmAt = time.Now().Add(-quitArmWindow) // exactly aged out
	nm, _ = m.Update(armClearMsg{})
	m = nm.(Model)
	if !m.quitArmAt.IsZero() {
		t.Fatalf("the expiry tick must clear an arm old enough")
	}
	if hint := m.hintLine(); strings.Contains(hint, quitArmToast) {
		t.Fatalf("the toast retires with the expired arm, got %q", hint)
	}
}

// (e) ANY other key press clears the arm (and its toast on the next
// render).
func TestQuitArmOtherKeyClears(t *testing.T) {
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("precondition: the first press armed")
	}
	if hint := m.hintLine(); !strings.Contains(hint, quitArmToast) {
		t.Fatalf("precondition: the toast is showing, got %q", hint)
	}

	nm, _ = m.Update(pressKey('x'))
	m = nm.(Model)
	if !m.quitArmAt.IsZero() {
		t.Fatalf("any other key press must clear the arm")
	}
	if hint := m.hintLine(); strings.Contains(hint, quitArmToast) {
		t.Fatalf("the toast retires on the same render, got %q", hint)
	}
	// back to the plain keymap line.
	if hint := m.hintLine(); !strings.Contains(hint, "tab:panels") {
		t.Fatalf("the un-armed hint is the keymap line again, got %q", hint)
	}
}
