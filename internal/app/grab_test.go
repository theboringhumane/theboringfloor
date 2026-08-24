// grab_test.go — the terminal tab's OPT-IN keyboard-capture contract from
// the model level (wave-42; fakes only, real Model + a recording
// TerminalTab double wired through the SpawnTerminal seam — never a real
// PTY):
//
//	(a) RELEASED is the default: on the terminal tab the office keys behave
//	    EXACTLY like any other tab — tab/shift+tab cycle out and back,
//	    typed letters and enter are consumed (never the shell, never the
//	    chat), the released hint rides the status bar;
//	(b) the digit jump fires from a released terminal too ("3" → agents);
//	(c) ctrl+space is THE capture toggle BOTH ways: the toggle press is
//	    swallowed (never the shell), letters start reaching the PTY and
//	    the hint swaps — and a SECOND ctrl+space toggles back out in
//	    place;
//	(d) wave-41 while captured: tab forwards without a switch, shift+tab
//	    forwards without a Prev, "3"/"7" forward without a jump;
//	(e) ctrl+o releases OUT (the documented alias): swallowed, the app
//	    stays ON the terminal tab (the office keys are live in place —
//	    the old "release → chat" hop is gone), letters go quiet again and
//	    ctrl+space re-enters; a released ctrl+o is inert (never a dive);
//	(e2) REGRESSION PIN — a real TAB key event (tea.KeyPressMsg{Code:
//	    tea.KeyTab}) NEVER toggles capture, and released-mode tab reaches
//	    the panel-switch path (the old ctrl+i dive was byte-identical to
//	    tab, 0x09; the ctrl+space toggle can never collide);
//	(f) capture can never escape its tab: leaving while captured
//	    auto-releases and every (re-)entry starts RELEASED (explicit opt-in
//	    per visit — no memory of a prior capture);
//	(g) ctrl+q double-press-to-quit works from the RELEASED terminal (arm
//	    on the first press, quit on the second, tab never moves);
//	(h) same from the CAPTURED terminal (the wave-41 safety keep);
//	(i) with the terminal NOT focused the old contract is byte-identical:
//	    tab Next, shift+tab Prev, digit jumps fire — including a digit jump
//	    INTO the terminal tab;
//	(j) q and ctrl+c never lost their quit teeth: released on the terminal
//	    tab they quit; captured they forward to the REAL shell (q as
//	    ordinary input, ctrl+c as the 0x03 SIGINT byte), never a quit;
//	(k) the two frozen hint copies are pinned verbatim and hintLine swaps
//	    between them per capture state;
//	(l) TabJump covers 1..7 with -1 bounds on the edges.
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
// kept). Anything NOT recorded proves consumption — the wrap gates released
// forwarding itself (belt + hanger).
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
// spellings handleKey switches on ("tab", "shift+tab", "ctrl+space",
// "ctrl+o"). (ctrl+i is GONE: it was byte-identical to tab — 0x09 on
// non-kitty terminals — so the toggle key is ctrl+space, 0x00.)
func grabTab() tea.KeyPressMsg       { return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}) }
func grabShiftTab() tea.KeyPressMsg  { return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}) }
func grabCtrlSpace() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl}) }
func grabCtrlO() tea.KeyPressMsg     { return tea.KeyPressMsg(tea.Key{Code: 'o', Mod: tea.ModCtrl}) }
func pressEnter() tea.KeyPressMsg    { return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}) }

// grabSetupTerminal — a scratch-home model with the SpawnTerminal seam
// wired to a recording fake, arrived at the terminal tab the REAL way (one
// tab from chat — the arrival also exercises the terminal-INACTIVE switch
// path). The model arrives RELEASED: the default is the office keys, the
// opt-in capture is explicit per visit. The seam restores itself on test
// cleanup.
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
	if m.termCapturedNow() {
		t.Fatalf("precondition: the terminal tab must arrive RELEASED (opt-in), got captured")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("precondition: the ARRIVAL tab was the app's own switch — nothing forwarded yet, got %v", fake.keys)
	}
	return m, fake
}

// grabSetupCaptured — the released default PLUS one ctrl+space dive (the
// toggle itself is swallowed — never the shell).
func grabSetupCaptured(t *testing.T) (Model, *grabFakeTerm) {
	t.Helper()
	m, fake := grabSetupTerminal(t)
	nm, _ := m.Update(grabCtrlSpace())
	m = nm.(Model)
	if !m.termCapturedNow() {
		t.Fatalf("ctrl+space must dive INTO capture while the terminal is released")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("ctrl+space is APP-KEPT — it must never reach the shell, got %v", fake.keys)
	}
	if hint := m.hintLine(); hint != termHintCaptured {
		t.Fatalf("the captured terminal's hint line must be termHintCaptured verbatim, got %q", hint)
	}
	return m, fake
}

// (a) RELEASED is the default: office keys behave normally on the terminal
// tab — letters/enter consumed (nothing reaches the shell), tab cycles out,
// shift+tab cycles back, released hint throughout.
func TestTerminalReleasedDefaultOfficeKeys(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	// typed letters and enter are CONSUMED — never the shell.
	for _, kp := range []tea.KeyPressMsg{pressKey('h'), pressKey('i'), pressEnter()} {
		nm, _ := m.Update(kp)
		m = nm.(Model)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released terminal: typed letters must NOT reach the PTY, got %v", fake.keys)
	}
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("released terminal must ride the released hint, got %q", hint)
	}

	// tab cycles OFF the terminal (terminal → agents) like any other tab…
	nm, _ := m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("released terminal: tab must cycle to agents, got index %d", idx)
	}
	// …shift+tab cycles back (agents → terminal)…
	nm, _ = m.Update(grabShiftTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("released round-trip: shift+tab must return to the terminal, got index %d", idx)
	}
	// …and the round trip neither captured nor forwarded anything.
	if m.termCapturedNow() {
		t.Fatalf("the tab cycle must not toggle capture (opt-in only)")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released terminal: navigation keys must never reach the shell, got %v", fake.keys)
	}
}

// (b) the digit jump fires from a released terminal (1..7 are office keys
// again): "3" jumps to agents, "2" jumps back INTO the terminal.
func TestTerminalReleasedDigitsJump(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	nm, _ := m.Update(pressKey('3'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("released terminal: \"3\" must jump to agents, got index %d", idx)
	}
	nm, _ = m.Update(pressKey('2'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("released round-trip: \"2\" must jump back INTO the terminal, got index %d", idx)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released terminal: digit jumps must never reach the shell, got %v", fake.keys)
	}
}

// (c) ctrl+space is THE capture toggle BOTH ways: it dives INTO capture
// (letters start reaching the shell) and a second press toggles back OUT
// in place (office keys live again, no tab hop).
func TestTerminalCtrlSpaceTogglesCaptureBothWays(t *testing.T) {
	m, fake := grabSetupCaptured(t)

	// dive in (via the setup helper): letters reach the shell.
	nm, _ := m.Update(pressKey('p'))
	m = nm.(Model)
	if got := fake.lastKey(); got != "p" {
		t.Fatalf("captured terminal: the typed letter must reach the shell, got %q (all: %v)", got, fake.keys)
	}
	if hint := m.hintLine(); hint != termHintCaptured {
		t.Fatalf("captured terminal must ride the captured hint, got %q", hint)
	}

	// the SAME key toggles back OUT — capture flips captured → released.
	wantKeys := len(fake.keys)
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if m.termCapturedNow() {
		t.Fatalf("a second ctrl+space must toggle the capture back OFF")
	}
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("ctrl+space releases IN PLACE — no tab hop, got index %d", idx)
	}
	if len(fake.keys) != wantKeys {
		t.Fatalf("the toggle-out ctrl+space is APP-KEPT — never the shell, got %v", fake.keys)
	}
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("post-toggle hint must be the released copy, got %q", hint)
	}

	// released again: the office keys are live right here.
	nm, _ = m.Update(pressKey('z'))
	m = nm.(Model)
	if len(fake.keys) != wantKeys {
		t.Fatalf("released again: the letter must not reach the shell, got %v", fake.keys)
	}
	nm, _ = m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("released again: a real tab event must cycle to agents, got index %d", idx)
	}
	if m.termCapturedNow() {
		t.Fatalf("the tab cycle must not re-capture (toggle is ctrl+space only)")
	}
}

// (d) wave-41 while captured: TAB/SHIFT+TAB are the shell's completion
// pair (no app switch), the digit keys ordinary input (no jump).
func TestTerminalCapturedWave41Keys(t *testing.T) {
	m, fake := grabSetupCaptured(t)

	nm, _ := m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("captured: tab must NOT switch tabs, got index %d", idx)
	}
	if got := fake.lastKey(); got != "tab" {
		t.Fatalf("captured: the raw tab event must forward to the shell (0x09 completion), got %q (all: %v)", got, fake.keys)
	}

	nm, _ = m.Update(grabShiftTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("captured: shift+tab must NOT Prev tabs, got index %d", idx)
	}
	if got := fake.lastKey(); got != "shift+tab" {
		t.Fatalf("captured: the raw shift+tab event must forward to the shell (\\x1b[Z), got %q", got)
	}

	for _, key := range []string{"3", "7"} {
		nm, _ = m.Update(pressKey(rune(key[0])))
		m = nm.(Model)
		if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
			t.Fatalf("captured: %q must NOT jump tabs, got index %d", key, idx)
		}
		if got := fake.lastKey(); got != key {
			t.Fatalf("captured: the digit %q must reach the shell as ordinary input, got %q", key, got)
		}
	}
}

// (e) ctrl+o releases OUT of capture (the documented alias): swallowed,
// the app stays ON the terminal tab, office keys live again; ctrl+o while
// released is inert (NEVER a dive — release-only); ctrl+space re-enters.
func TestTerminalCtrlOReleasesCapture(t *testing.T) {
	m, fake := grabSetupCaptured(t)
	wantKeys := len(fake.keys)

	nm, _ := m.Update(grabCtrlO())
	m = nm.(Model)
	if m.termCapturedNow() {
		t.Fatalf("ctrl+o must release the shell capture")
	}
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("ctrl+o releases IN PLACE — no hop back to chat (released office keys live here), got index %d", idx)
	}
	if len(fake.keys) != wantKeys {
		t.Fatalf("ctrl+o is APP-KEPT — it must never reach the shell, got %v", fake.keys)
	}
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("post-release hint must be the released copy, got %q", hint)
	}

	// office keys live right here: letters consumed, tab cycles out.
	nm, _ = m.Update(pressKey('z'))
	m = nm.(Model)
	if len(fake.keys) != wantKeys {
		t.Fatalf("released again: the letter must not reach the shell, got %v", fake.keys)
	}
	nm, _ = m.Update(grabTab())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("released again: tab must cycle to agents, got index %d", idx)
	}
	nm, _ = m.Update(grabShiftTab())
	m = nm.(Model)

	// a released ctrl+o is inert — RELEASE-ONLY, it must NEVER dive.
	nm, _ = m.Update(grabCtrlO())
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("ctrl+o fires only while captured — released it must be a no-op, got index %d", idx)
	}
	if m.termCapturedNow() {
		t.Fatalf("ctrl+o is a release alias, never a dive: a released ctrl+o must not capture")
	}
	if len(fake.keys) != wantKeys {
		t.Fatalf("a released ctrl+o must never reach the shell, got %v", fake.keys)
	}

	// …and ctrl+space dives back in (the toggle flips indefinitely).
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if !m.termCapturedNow() {
		t.Fatalf("ctrl+space must re-enter capture from the released state")
	}
	nm, _ = m.Update(pressKey('k'))
	m = nm.(Model)
	if got := fake.lastKey(); got != "k" {
		t.Fatalf("re-captured: the typed letter must reach the shell again, got %q", got)
	}
}

// (e2) REGRESSION PIN — the tab-vs-toggle collision can never come back.
// A REAL tab key event (tea.KeyPressMsg{Code: tea.KeyTab} — an event
// literally named "tab", exactly what a hardware TAB press emits) must
// NEVER toggle the capture, and released-mode tab must reach the
// panel-switch path. ctrl+i is gone because it WAS this tab byte (0x09);
// the toggle is ctrl+space (0x00), so no key shares tab's encoding.
func TestTerminalTabKeyNeverTogglesCapture(t *testing.T) {
	m, fake := grabSetupTerminal(t)

	// released: the synthetic TAB event is an office panel switch, full stop.
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = nm.(Model)
	if m.termCapturedNow() {
		t.Fatalf("a tab key event must NEVER toggle capture into life")
	}
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("released-mode tab must reach the panel-switch path (agents), got index %d", idx)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released-mode tab must never reach the shell, got %v", fake.keys)
	}

	// back onto the terminal (shift+tab event — still released)…
	nm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("shift+tab must return to the terminal, got index %d", idx)
	}
	if m.termCapturedNow() {
		t.Fatalf("shift+tab must never toggle capture either")
	}

	// …dive via ctrl+space, toggle back out via ctrl+space, and the very
	// same tab event leaves the tab again — no stale capture can make tab
	// stick to the shell.
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if !m.termCapturedNow() {
		t.Fatalf("precondition: ctrl+space dives into capture")
	}
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if m.termCapturedNow() {
		t.Fatalf("precondition: ctrl+space releases back out")
	}
	nm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("post-toggle released tab must leave the tab (the old ctrl+i/tab conflict), got index %d", idx)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("post-toggle released tab must not reach the shell, got %v", fake.keys)
	}
}

// (f) capture never escapes its tab: leaving while captured auto-releases;
// re-entering starts RELEASED (no memory of the prior capture).
func TestTerminalCaptureNeverEscapesItsTab(t *testing.T) {
	m, fake := grabSetupCaptured(t)

	// leave while captured through a NON-keyboard path (the click/event
	// seam — an index switch) then route one more key: capture must be
	// released before it is handled.
	m.tabs.SetActive(2) // agents
	nm, _ := m.Update(pressKey('x'))
	m = nm.(Model)
	if m.termCaptured {
		t.Fatalf("leaving the terminal tab while captured must auto-release the capture")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("the auto-release key must never forward into the shell, got %v", fake.keys)
	}

	// re-enter: RELEASED again — the opt-in is explicit per visit.
	nm, _ = m.Update(pressKey('2'))
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("re-enter must land on the terminal tab, got index %d", idx)
	}
	if m.termCapturedNow() {
		t.Fatalf("re-entering the terminal must start RELEASED (no memory of the prior capture)")
	}
	nm, _ = m.Update(pressKey('w'))
	m = nm.(Model)
	if len(fake.keys) != 0 {
		t.Fatalf("released re-entry: the letter must not reach the shell, got %v", fake.keys)
	}
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("re-entry must ride the released hint, got %q", hint)
	}
}

// (g) the safety keep: ctrl+q from a RELEASED terminal ARMS on the first
// press (tab stays put, no quit) and quits on the second inside the window.
func TestTerminalReleasedCtrlQStillArmsAndQuits(t *testing.T) {
	m, _ := grabSetupTerminal(t)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("the first ctrl+q must ARM on the released terminal")
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
		t.Fatalf("the second ctrl+q inside the window must quit from the released terminal")
	}
}

// (h) same from a CAPTURED terminal — the wave-41 safety keep.
func TestTerminalCapturedCtrlQStillArmsAndQuits(t *testing.T) {
	m, fake := grabSetupCaptured(t)

	nm, _ := m.Update(ctrlQ())
	m = nm.(Model)
	if m.quitArmAt.IsZero() {
		t.Fatalf("the first ctrl+q must ARM even with the shell captured")
	}
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("the arming press must not move the tab, got index %d", idx)
	}
	if len(fake.keys) != 0 {
		t.Fatalf("ctrl+q is APP-KEPT — it must never reach the shell, got %v", fake.keys)
	}

	nm, cmd := m.Update(ctrlQ())
	m = nm.(Model)
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("the second ctrl+q inside the window must quit from the captured terminal")
	}
}

// (i) terminal NOT focused — the old contract byte for byte: tab Next,
// shift+tab Prev, digit jumps fire (into the terminal included).
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
	if len(fake.keys) != 0 {
		t.Fatalf("the arriving digit jump must not forward to the shell, got %v", fake.keys)
	}
}

// (j) q and ctrl+c keep their quit teeth while released on the terminal
// tab; captured they belong to the shell (q ordinary input, ctrl+c 0x03
// SIGINT) — never an app quit.
func TestTerminalQuitKeysReleasedVsCaptured(t *testing.T) {
	// released: q quits.
	m, fake := grabSetupTerminal(t)
	nm, cmd := m.Update(pressKey('q'))
	m = nm.(Model)
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("released terminal: q must quit (like any other non-chat tab)")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released terminal: the quitting q must not reach the shell, got %v", fake.keys)
	}

	// released: ctrl+c quits.
	m, fake = grabSetupTerminal(t)
	nm, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	m = nm.(Model)
	if !hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("released terminal: ctrl+c must quit (like any other tab)")
	}
	if len(fake.keys) != 0 {
		t.Fatalf("released terminal: the quitting ctrl+c must not reach the shell, got %v", fake.keys)
	}

	// captured: q is the shell's ordinary input.
	m, fake = grabSetupCaptured(t)
	nm, cmd = m.Update(pressKey('q'))
	m = nm.(Model)
	if hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("captured terminal: q must belong to the shell, never an app quit")
	}
	if got := fake.lastKey(); got != "q" {
		t.Fatalf("captured terminal: q must reach the shell as ordinary input, got %q", got)
	}

	// captured: ctrl+c is the shell's SIGINT byte, not an app quit.
	nm, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	m = nm.(Model)
	if hasQuitLeaf(leafMsgs(cmd)) {
		t.Fatalf("captured terminal: ctrl+c must be the shell's 0x03, never an app quit")
	}
	if got := fake.lastKey(); got != "ctrl+c" {
		t.Fatalf("captured terminal: ctrl+c must forward to the shell (0x03 SIGINT), got %q", got)
	}
}

// (k) the two frozen hint copies — hintLine swaps verbatim per state.
func TestTerminalHintConstPinned(t *testing.T) {
	if termHintReleased != "office keys · ctrl+space → shell · ctrl+q quit" {
		t.Fatalf("termHintReleased copy is frozen to the toggle contract, got %q", termHintReleased)
	}
	if termHintCaptured != "typing → shell · ctrl+space release · ctrl+q quit" {
		t.Fatalf("termHintCaptured copy is frozen to the toggle contract, got %q", termHintCaptured)
	}
	if strings.Contains(termHintCaptured, "1-6/tab") || strings.Contains(termHintReleased, "1-6/tab") {
		t.Fatalf("the hint copies must not mix the panel-switch wording")
	}

	m, _ := grabSetupTerminal(t)
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("the released terminal's hint line must be termHintReleased verbatim, got %q", hint)
	}
	nm, _ := m.Update(grabCtrlSpace())
	m = nm.(Model)
	if hint := m.hintLine(); hint != termHintCaptured {
		t.Fatalf("the captured terminal's hint line must be termHintCaptured verbatim, got %q", hint)
	}
	// back via the SAME toggle key…
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("post-toggle-release the hint must swap back verbatim, got %q", hint)
	}
	// …and via the ctrl+o alias.
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	if hint := m.hintLine(); hint != termHintCaptured {
		t.Fatalf("the re-captured terminal's hint must be termHintCaptured verbatim, got %q", hint)
	}
	nm, _ = m.Update(grabCtrlO())
	m = nm.(Model)
	if hint := m.hintLine(); hint != termHintReleased {
		t.Fatalf("post-ctrl+o-release the hint must swap back verbatim, got %q", hint)
	}
}

// (l) TabJump covers 1..7 (the git tab's "7" seam) and misses outside.
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
