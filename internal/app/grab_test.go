// grab_test.go — the terminal tab's KEY-GRAB contract from the model level
// (fakes only, real Model + a recording TerminalTab double wired through
// the SpawnTerminal seam — never a real PTY):
//
//	(a) while the terminal tab is focused, TAB is not an app tab-switch:
//	    the active sidebar index stays on the terminal and the raw key
//	    event forwards to the terminal verbatim (the panel maps it to
//	    0x09 — the shell's completion);
//	(b) SHIFT+TAB rides the same grab — the shell's reverse completion
//	    ("\x1b[Z" from the panel's keyToBytes), never a Prev;
//	(c) the digit jumps (TabJump 1..7) are also captured while focused —
//	    "3" does NOT jump to the agents tab, "7" does not jump to git;
//	(d) ctrl+o is the ONE kept release key: it switches to chat (index 0)
//	    and is NEVER forwarded to the shell; once released, ordinary app
//	    shortcuts are live again;
//	(e) with the terminal NOT focused the old contract is byte-identical:
//	    tab Next, shift+tab Prev, digit jumps fire — including a digit
//	    jump INTO the terminal tab;
//	(f) ctrl+q double-press-to-quit still works from the terminal (arm on
//	    the first press, quit on the second, tab never moves);
//	(g) the frozen termHint copy drops the now-false "1-6/tab panels"
//	    wording and rides hintLine while the terminal is focused;
//	(h) TabJump covers 1..7 with -1 bounds on the edges.
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// grabFakeTerm — a recording TerminalTab double for the SpawnTerminal seam:
// every forwarded keypress is captured so the grab tests can assert which
// raw key events reached the terminal (and, by exclusion, which the app
// kept).
type grabFakeTerm struct {
	keys   []string // String() of every forwarded tea.KeyPressMsg
	closed bool
}

func (f *grabFakeTerm) Title() string                 { return "term" }
func (f *grabFakeTerm) SetSize(w, h int)              {}
func (f *grabFakeTerm) SetState(st state.OfficeState) {}
func (f *grabFakeTerm) View() string                  { return "" }
func (f *grabFakeTerm) Alive() bool                   { return !f.closed }
func (f *grabFakeTerm) Close() error                  { f.closed = true; return nil }
func (f *grabFakeTerm) Update(msg tea.Msg) tea.Cmd {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		f.keys = append(f.keys, kp.String())
	}
	return nil
}

// lastKey — the most recent keypress the fake terminal saw ("" if none).
func (f *grabFakeTerm) lastKey() string {
	if len(f.keys) == 0 {
		return ""
	}
	return f.keys[len(f.keys)-1]
}

// grabKey constructors — bubbletea v2 Key{Code,Mod} → the exact String()
// spellings handleKey switches on ("tab", "shift+tab", "ctrl+o").
func grabTab() tea.KeyPressMsg      { return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}) }
func grabShiftTab() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}) }
func grabCtrlO() tea.KeyPressMsg    { return tea.KeyPressMsg(tea.Key{Code: 'o', Mod: tea.ModCtrl}) }

// grabSetupTerminal — a scratch-home model with the SpawnTerminal seam
// wired to a recording fake, arrived at the terminal tab the REAL way (one
// tab from chat — the arrival also exercises the terminal-INACTIVE switch
// path). The seam restores itself on test cleanup.
func grabSetupTerminal(t *testing.T) (Model, *grabFakeTerm) {
	t.Helper()
	scratchHome(t)
	fake := &grabFakeTerm{}
	prev := SpawnTerminal
	SpawnTerminal = func(cols, rows int) (TerminalTab, error) { return fake, nil }
	t.Cleanup(func() { SpawnTerminal = prev })
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)
	nm, _ := m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("precondition: tab from chat must land on the terminal tab, got %d", idx)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("precondition: the ARRIVAL tab was the app's own switch — nothing forwarded yet, got %v", fake.keys)
	}
	return m, fake
}

// (a) tab while focused: no switch, the raw "tab" event reaches the shell.
func TestTerminalGrabTabForwardsNoSwitch(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	nm, _ := m.Update(grabTab())
	m = nm.(Model)

	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("tab while the terminal is focused must NOT switch tabs, got index %d", idx)
	}
	if got := fake.lastKey(); got != "tab" {
		t.Fatalf("the raw tab event must forward to the shell (the panel maps it to 0x09), got %q (all: %v)", got, fake.keys)
	}
}

// (b) shift+tab while focused: no Prev, the raw "shift+tab" event reaches
// the shell (keyToBytes translates it to "\x1b[Z").
func TestTerminalGrabShiftTabForwardsNoSwitch(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	nm, _ := m.Update(grabShiftTab())
	m = nm.(Model)

	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("shift+tab while the terminal is focused must NOT switch tabs, got index %d", idx)
	}
	if got := fake.lastKey(); got != "shift+tab" {
		t.Fatalf("the raw shift+tab event must forward to the shell, got %q (all: %v)", got, fake.keys)
	}
}

// (c) digit jumps die at the grab too: "3" does not jump to agents, "7"
// (the Tab7 seam) does not jump to git — both are ordinary shell input.
func TestTerminalGrabDigitsForwardNoJump(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	nm, _ := m.Update(pressKey('3'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("\"3\" while the terminal is focused must NOT jump to agents, got index %d", idx)
	}
	if got := fake.lastKey(); got != "3" {
		t.Fatalf("the digit must reach the shell as ordinary input, got %q", got)
	}

	nm, _ = m.Update(pressKey('7'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("\"7\" while the terminal is focused must NOT jump to git, got index %d", idx)
	}
	if got := fake.lastKey(); got != "7" {
		t.Fatalf("the digit must reach the shell as ordinary input, got %q", got)
	}
}

// (d) ctrl+o releases the keyboard back to the app: it switches to chat,
// is never forwarded, and ordinary shortcuts (tab → Next) work again.
func TestTerminalGrabCtrlOReleasesToChat(t *testing.T) {
	m, fake := grabSetupTerminal(t)
	wantKeys := len(fake.keys)

	nm, _ := m.Update(grabCtrlO())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 0 {
		t.Fatalf("ctrl+o must release the terminal focus back to chat, got index %d", idx)
	}
	if len(fake.keys) != wantKeys {
		t.Fatalf("ctrl+o is APP-KEPT — it must never reach the shell, got %v", fake.keys)
	}

	// released → the office shortcuts are live again: tab cycles Next.
	nm, _ = m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("after the release, tab must cycle panels again (chat → terminal), got index %d", idx)
	}
}

// (e) terminal NOT focused — the pre-grab contract byte for byte: tab
// Next, shift+tab Prev, digit jumps fire (into the terminal included).
func TestTerminalInactiveOldSwitchesUnchanged(t *testing.T) {
	fake := &grabFakeTerm{}
	prev := SpawnTerminal
	SpawnTerminal = func(cols, rows int) (TerminalTab, error) { return fake, nil }
	t.Cleanup(func() { SpawnTerminal = prev })
	scratchHome(t)
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)

	// chat focused: tab → Next (terminal).
	nm, _ := m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 1 {
		t.Fatalf("tab from chat must fire Next, got index %d", idx)
	}

	// a non-chat, non-terminal tab: shift+tab → Prev …
	m.tabs.SetActive(2) // agents
	nm, _ = m.Update(grabShiftTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 1 {
		t.Fatalf("shift+tab from agents must fire Prev, got index %d", idx)
	}

	// … and a digit jump fires — 3 → agents (index 2) …
	m.tabs.SetActive(4) // mail — the digit's origin needs no special tab
	nm, _ = m.Update(pressKey('3'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("\"3\" from a non-terminal tab must jump to agents, got index %d", idx)
	}
	// … and 2 → INTO the terminal (lazy spawn via the seam as usual).
	nm, _ = m.Update(pressKey('2'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("\"2\" from a non-terminal tab must jump INTO the terminal, got index %d", idx)
	}
}

// (f) the safety keep: ctrl+q from the terminal ARMS on the first press
// (tab stays put, no quit) and quits on the second inside the window.
func TestTerminalGrabCtrlQStillArmsAndQuits(t *testing.T) {
	m, _ := grabSetupTerminal(t)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("the first ctrl+q must ARM even with the terminal focused")
	}
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("the arming press must not move the tab, got index %d", idx)
	}
	if hint := m.hintLine(); !strings.Contains(hint, quitArmToast) {
		t.Fatalf("the arm toast must outrank the terminal hint, got %q", hint)
	}

	nm, cmd := m.Update(ctrlQ())
	m = nm.(Model)
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("the second ctrl+q inside the window must quit from the terminal")
	}
}

// (g) the terminal hint drops the panel-switch wording — and hintLine
// surfaces it verbatim while the terminal is focused.
func TestTerminalGrabTermHintCopy(t *testing.T) {
	if termHint != "typing → shell · ctrl+o release · ctrl+q quit" {
		t.Fatalf("the termHint copy is frozen to the grab contract, got %q", termHint)
	}
	if strings.Contains(termHint, "1-6/tab") {
		t.Fatalf("the panel-switch wording must be gone (false while grabbed), got %q", termHint)
	}

	m, _ := grabSetupTerminal(t)
	if hint := m.hintLine(); hint != termHint {
		t.Fatalf("the focused terminal's hint line must be the termHint verbatim, got %q", hint)
	}
}

// (h) TabJump covers 1..7 (the git tab's "7" seam) and misses outside.
func TestTerminalGrabTabJumpOneToSeven(t *testing.T) {
	k := NewKeyMap()
	for want, s := range []string{"1", "2", "3", "4", "5", "6", "7"} {
		if got := k.TabJump(s); got != want {
			t.Fatalf("TabJump(%q) = %d, want %d", s, got, want)
		}
	}
	for _, s := range []string{"0", "8", "9", "tab", ""} {
		if got := k.TabJump(s); got != -1 {
			t.Fatalf("TabJump(%q) = %d, want -1 (never a jump)", s, got)
		}
	}
	keys := k.Tab7.Keys()
	found := false
	for _, v := range keys {
		if v == "7" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the Tab7 binding must carry the \"7\" key, got %v", keys)
	}
}
