// keys.go — the app's keymap, in one place so the statusbar's hint segment
// and the actual key handling can never drift apart. ctrl+q QUIT-ARMS from
// EVERYWHERE — the first press toasts the hint bar, the second press inside
// quitArmWindow quits (q/ctrl+c stay single-press, and only outside the
// chat textarea and a CAPTURED terminal — inside a captured terminal they
// go to the shell; ctrl+c is a real SIGINT there). tab/shift+tab cycle
// panels, 1..7 jump, ↑↓/pgup/pgdn/wheel scroll, enter sends chat.
package app

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// termHintReleased is the statusline hint while the terminal tab is active
// and the keyboard is RELEASED (the DEFAULT): the office keys behave
// exactly like on any other tab — tab/shift+tab cycle, 1..7 jump, q quits —
// and ctrl+space is the ONE toggle INTO shell capture (and back out).
// Frozen copy, pinned by grab_test.go.
const termHintReleased = "office keys · ctrl+space → shell · ctrl+q quit"

// termHintCaptured is the statusline hint while the terminal tab CAPTURED
// the keyboard (opt-in via ctrl+space): typing — tab/shift+tab and the
// digit keys INCLUDED — goes to the REAL shell; the only app-kept keys are
// ctrl+space (the toggle back out), ctrl+o (release alias) and ctrl+q.
// Frozen copy, pinned by grab_test.go.
const termHintCaptured = "typing → shell · ctrl+space release · ctrl+q quit"

// quitArmToast — the high-visibility statusbar line swapped in while a
// ctrl+q arm is live (the FIRST press arms instead of quitting; the
// second press inside quitArmWindow quits). Frozen copy — pinned by
// quit_arm_test.go.
const quitArmToast = "ctrl+q again: quit the office"

// KeyMap — global app bindings.
type KeyMap struct {
	Quit   key.Binding
	Next   key.Binding
	Prev   key.Binding
	Scroll key.Binding
	Send   key.Binding
	Tab1   key.Binding
	Tab2   key.Binding
	Tab3   key.Binding
	Tab4   key.Binding
	Tab5   key.Binding
	Tab6   key.Binding
	Tab7   key.Binding
}

// NewKeyMap returns the default bindings.
func NewKeyMap() KeyMap {
	return KeyMap{
		Quit:   key.NewBinding(key.WithKeys("ctrl+q", "q", "ctrl+c"), key.WithHelp("ctrl+q", "quit")),
		Next:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "panels")),
		Prev:   key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "panels")),
		Scroll: key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown"), key.WithHelp("↑↓", "scroll")),
		Send:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		Tab1:   key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "chat")),
		Tab2:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "terminal")),
		Tab3:   key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "agents")),
		Tab4:   key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "board")),
		Tab5:   key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "mail")),
		Tab6:   key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "activity")),
		Tab7:   key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "git")),
	}
}

// ShortHelp is the statusbar segment, in display order.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Scroll, k.Send, k.Quit}
}

// HintLine renders the static statusbar hint, keymap-driven:
// "tab:panels · ↑↓:scroll · enter:send · ctrl+q:quit"
func (k KeyMap) HintLine() string {
	parts := make([]string, 0, len(k.ShortHelp()))
	for _, b := range k.ShortHelp() {
		h := b.Help()
		parts = append(parts, h.Key+":"+h.Desc)
	}
	return strings.Join(parts, " · ")
}

// ShortHelpView renders the same ShortHelp set through bubbles/help (the
// "… for non-devs" surface); used anywhere a taller help strip is wanted.
func (k KeyMap) ShortHelpView() string {
	h := help.New()
	h.ShortSeparator = "  "
	return h.ShortHelpView(k.ShortHelp())
}

// TabJump maps a 1..7 keypress to a tab index, or -1.
func (k KeyMap) TabJump(s string) int {
	switch s {
	case "1":
		return 0
	case "2":
		return 1
	case "3":
		return 2
	case "4":
		return 3
	case "5":
		return 4
	case "6":
		return 5
	case "7":
		return 6
	}
	return -1
}
