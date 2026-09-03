// term_mouse_test.go — the terminal drag-select wave's APP-SIDE routing.
// The panel owns selection/clipboard internally (internal/panels/terminal.go
// + terminal_sel_test.go pin that); the model's remaining jobs are pinned
// here with a recording TerminalTab double (never a real PTY):
//
//	(a) press/motion/release reach the terminal tab's panel ADJUSTED into
//	    sidebar-box space (X minus the floor cols, Y minus the topbar row —
//	    the git-click precedent; the panel subtracts Tabs.ContentOffset
//	    itself) in BOTH capture states: mouse is capture-independent;
//	(b) a press OUTSIDE the panel box (the floor's cols) never reaches the
//	    panel — the normal floor/chat path resumes unchanged;
//	(c) the key gate is untouched: released swallows keys, captured sends
//	    them — the mouse pass-through must never leak keystrokes;
//	(d) wheel now reaches the panel released too (terminal scrollback
//	    viewing needs no capture dive).
package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// mouseFakeTerm — a recording TerminalTab double: every msg forwarded to
// the panel lands in msgs with its (already-adjusted) coordinates captured
// for the mouse kinds. What is NOT recorded proves consumption.
type mouseFakeTerm struct {
	msgs     []tea.Msg
	clicks   []tea.Mouse // MouseClickMsg coords as delivered
	motions  []tea.Mouse
	releases []tea.Mouse
	wheels   int
	keys     int
	blurs    int
	focuses  int
}

func (f *mouseFakeTerm) Title() string                 { return "term" }
func (f *mouseFakeTerm) SetSize(w, h int)              {}
func (f *mouseFakeTerm) SetState(st state.OfficeState) {}
func (f *mouseFakeTerm) View() string                  { return "" }
func (f *mouseFakeTerm) Alive() bool                   { return true }
func (f *mouseFakeTerm) Close() error                  { return nil }
func (f *mouseFakeTerm) Focus()                        { f.focuses++ }
func (f *mouseFakeTerm) Blur()                         { f.blurs++ }
func (f *mouseFakeTerm) Update(msg tea.Msg) tea.Cmd {
	f.msgs = append(f.msgs, msg)
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		f.clicks = append(f.clicks, tea.Mouse(msg))
	case tea.MouseMotionMsg:
		f.motions = append(f.motions, tea.Mouse(msg))
	case tea.MouseReleaseMsg:
		f.releases = append(f.releases, tea.Mouse(msg))
	case tea.MouseWheelMsg:
		f.wheels++
	case tea.KeyPressMsg:
		f.keys++
	}
	return nil
}

// mouseSetupTerminal — grab_test's released-terminal model shape, sized at
// a roomy desktop frame (120x40 → desktop branch, floorW = width-sidebar).
func mouseSetupTerminal(t *testing.T, captured bool) (Model, *mouseFakeTerm) {
	t.Helper()
	scratchHome(t)
	fake := &mouseFakeTerm{}
	prev := SpawnTerminal
	SpawnTerminal = func(cols, rows int) (TerminalTab, error) { return fake, nil }
	t.Cleanup(func() { SpawnTerminal = prev })
	m := New(&sessBackend{primary: "ses-alpha-new"}, nil)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = nm.(Model)
	nm, _ = m.Update(grabTab()) // chat → terminal (arrival also spawns the fake)
	m = nm.(Model)
	if idx := m.tabs.ActiveIndex(); idx != terminalIndex {
		t.Fatalf("precondition: tab must land on the terminal tab, got %d", idx)
	}
	if m.termCapturedNow() != captured {
		if captured {
			nm, _ = m.Update(grabCtrlSpace())
			m = nm.(Model)
			if !m.termCapturedNow() {
				t.Fatalf("precondition: ctrl+space must dive into capture")
			}
		} else {
			t.Fatalf("precondition: the terminal tab must arrive RELEASED")
		}
	}
	if m.floorW <= 0 {
		t.Fatalf("precondition: 120x40 must produce the desktop sidebar (floorW>0), got %d", m.floorW)
	}
	fake.msgs, fake.wheels, fake.blurs, fake.focuses = nil, 0, 0, 0 // arrival bookkeeping is not part of the pin
	return m, fake
}

func TestTermMouseRoutesReleased(t *testing.T) {
	m, fake := mouseSetupTerminal(t, false)

	pressY := 15
	pressX := m.floorW + 30
	nm, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: pressX, Y: pressY, Button: tea.MouseLeft}))
	m = nm.(Model)
	if len(fake.clicks) != 1 {
		t.Fatalf("a sidebar-box press on the terminal tab must reach the panel RELEASED, got %d clicks", len(fake.clicks))
	}
	if got, want := fake.clicks[0].X, pressX-m.floorW; got != want {
		t.Errorf("press X must lose the floor cols (box space), got %d want %d", got, want)
	}
	if got, want := fake.clicks[0].Y, pressY-1; got != want {
		t.Errorf("press Y must lose the topbar row (box space), got %d want %d", got, want)
	}

	nm, _ = m.Update(tea.MouseMotionMsg(tea.Mouse{X: pressX + 5, Y: pressY + 2, Button: tea.MouseLeft}))
	m = nm.(Model)
	if len(fake.motions) != 1 {
		t.Fatalf("motion while the chat selection is NOT armed must still reach the terminal panel, got %d", len(fake.motions))
	}
	nm, _ = m.Update(tea.MouseReleaseMsg(tea.Mouse{X: pressX + 5, Y: pressY + 2, Button: tea.MouseLeft}))
	m = nm.(Model)
	if len(fake.releases) != 1 {
		t.Fatalf("the dragged release must reach the terminal panel (copy lives there), got %d", len(fake.releases))
	}
	if got, want := fake.releases[0].X, pressX+5-m.floorW; got != want {
		t.Errorf("release X must ride the same box-space adjust, got %d want %d", got, want)
	}
	if m.sel != mselIdle {
		t.Fatalf("the terminal drag must never arm the CHAT selection machine, got m.sel=%d", m.sel)
	}
}

func TestTermViewportClickCapturesButOutsideDoesNot(t *testing.T) {
	m, fake := mouseSetupTerminal(t, false)
	dx, dy := m.tabs.ContentOffset()
	insideX := m.floorW + dx + 3
	insideY := 1 + dy + 3

	nm, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: insideX, Y: insideY, Button: tea.MouseLeft}))
	m = nm.(Model)
	if !m.termCapturedNow() || fake.focuses == 0 {
		t.Fatalf("a left click in a blank terminal viewport cell must capture, captured=%v focuses=%d", m.termCapturedNow(), fake.focuses)
	}

	// The tab strip is outside the viewport. Its click may be routed by the
	// terminal mouse seam, but must not alter the released capture state.
	m.setTermCaptured(false)
	nm, _ = m.Update(tea.MouseClickMsg(tea.Mouse{X: m.floorW + dx + 3, Y: 1, Button: tea.MouseLeft}))
	m = nm.(Model)
	if m.termCapturedNow() {
		t.Fatal("a click outside the terminal viewport must not capture")
	}
}

func TestTermMouseRoutesCaptured(t *testing.T) {
	m, fake := mouseSetupTerminal(t, true)
	nm, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: m.floorW + 12, Y: 8, Button: tea.MouseLeft}))
	m = nm.(Model)
	if len(fake.clicks) != 1 {
		t.Fatalf("mouse stays forwarded while CAPTURED (selection over a running shell), got %d clicks", len(fake.clicks))
	}
}

func TestTermMousePressOnFloorIgnored(t *testing.T) {
	m, fake := mouseSetupTerminal(t, false)
	if m.floorW < 10 {
		t.Fatalf("precondition: need floor cols to click on, floorW=%d", m.floorW)
	}
	nm, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X: m.floorW - 5, Y: 12, Button: tea.MouseLeft}))
	m = nm.(Model)
	if len(fake.msgs) != 0 {
		t.Fatalf("a floor-col press must NOT reach the terminal panel region, got %d msgs", len(fake.msgs))
	}
}

func TestTermMouseKeysStayGated(t *testing.T) {
	m, fake := mouseSetupTerminal(t, false)
	nm, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'a'}))
	m = nm.(Model)
	if fake.keys != 0 {
		t.Fatalf("released swallows typed keys STILL (the mouse pass-through must not leak them), got %d", fake.keys)
	}
	// wheel in released mode NOW forwards (terminal scrollback viewing does
	// not need a dive) — previously dropped with the keys.
	nm, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{X: m.floorW + 10, Y: 10, Button: tea.MouseWheelUp}))
	m = nm.(Model)
	if fake.wheels != 1 {
		t.Fatalf("wheel must reach the terminal panel released, got %d", fake.wheels)
	}
	nm, _ = m.Update(grabCtrlSpace())
	m = nm.(Model)
	nm, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'a'}))
	m = nm.(Model)
	if fake.keys != 1 {
		t.Fatalf("captured forwards keys as ever, got %d", fake.keys)
	}
}
