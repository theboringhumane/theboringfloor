// selection.go — webpage-style mouse text selection over the chat
// transcript: a left-press over selectable transcript text ARMS a pending
// selection (the click's fate — legacy click action vs copy-drag — stays
// undecided until release, so single-click semantics survive untouched),
// drag-motion extends the head (the chat panel highlights the span live,
// chat_selection.go), and a dragged release copies the plain text and
// toasts the status bar with the frozen "Copied N chars" note. The copy
// itself: on darwin a REAL pbcopy round-trip (payload on stdin, error
// checked) with the toast GATED on its verdict (clipboardResultMsg) and
// the terminal's OSC52 escape (tea.SetClipboard) riding along as the
// fallback for ssh'd terminals; elsewhere OSC52 is the only channel and
// the note arms at release as before (no OS feedback exists there). A
// motionless release replays the ORIGINAL press through handleClick, so
// thread toggles, fold rows, perm cards and floor picks behave exactly
// like a plain click.
//
// The copy note rides the same status-bar seam as the ctrl+q quit arm:
// hintLine swaps to it while fresh and its own tea.Tick retires it
// (copyNoteClearMsg); a fresher note's tick owns its expiry.
package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Selection state machine: idle → armed (press down, fate undecided) →
// selected (drag released, highlight persists) → idle (esc / plain click /
// a fresh arm).
const (
	mselIdle = iota
	mselArmed
	mselSelected
)

// copyNoteClearMsg — the copy toast's expiry tick landing (armClearMsg's
// twin for the selection note).
type copyNoteClearMsg struct{}

// clipboardResultMsg — darwin's pbcopy round-trip landing: err == nil means
// the pasteboard really holds the text and the frozen "Copied N chars"
// toast may arm; err != nil rides the same status-bar seam as a warn.
type clipboardResultMsg struct {
	n   int
	err error
}

// copyNoteWindow — how long "Copied N chars" rides the status bar.
const copyNoteWindow = 2 * time.Second

// handlePress routes a left mouse PRESS: over selectable transcript text it
// arms the selection and waits for the release verdict; anywhere else it
// clears any finished selection (webpage rule) and runs the legacy click
// path immediately (floor picks, perm cards, thread rows — those are
// press-actionable today and stay so).
func (m *Model) handlePress(msg tea.MouseClickMsg) tea.Cmd {
	if m.height == 0 || m.zen || msg.Button != tea.MouseLeft {
		return nil
	}
	// the 2-cell chrome (topbar + statusbar) never reacts
	if msg.Y <= 0 || msg.Y >= m.height-1 {
		return nil
	}
	// the /model picker is keys-only: while its card is up a click lands on
	// NOTHING underneath — no floor selection, no thread toggle, no popover
	// answer, no selection arm.
	if m.modelPick != nil {
		return nil
	}
	// an open thread-focus owns the whole middle region: ONLY its ↳ diff
	// sub-rows answer (the pane's own toolDiffRows hit-map) — every other
	// row is inert, and NO selection arms on the hidden main transcript
	// underneath (the release would replay straight into handleClick,
	// whose focus gate below swallows it regardless).
	if m.threadFocus != nil {
		m.threadFocus.Click(msg.X, msg.Y-1) // pane coords: row 0 = its header
		return nil
	}
	if cx, cy, ok := m.chatContentCoords(msg.X, msg.Y); ok && m.chat.SelectionBegin(cx, cy) {
		m.sel = mselArmed
		m.selPress = tea.Mouse(msg)
		m.selDragged = false
		return nil
	}
	// The terminal tab claims presses inside ITS panel box first-wave: a
	// left press over body rows arms the terminal's own drag-select and
	// must never reach the legacy click path (or a hidden-chat arm — the
	// chat gate above already refuses ActiveIndex!=0). A press OUTSIDE the
	// box falls through unchanged (floor picks, popover, thread rows).
	if cmd, ok := m.sendTermMouse(msg); ok {
		return cmd
	}
	// a press anywhere else clears a finished selection (webpage rule)
	if m.chat != nil {
		m.chat.ClearSelection()
	}
	m.sel = mselIdle
	return m.handleClick(msg)
}

// handleMotion extends the armed selection's drag. Cheap: the Update arm
// only calls here while a drag is live, so an idle motion flood costs a
// single compare (battery rule).
func (m *Model) handleMotion(msg tea.MouseMotionMsg) {
	if m.sel != mselArmed || m.chat == nil {
		return
	}
	m.selDragged = true
	cx, cy := m.chatCoordsTranslate(msg.X, msg.Y)
	m.chat.SelectionDrag(cx, cy)
}

// handleRelease settles an armed selection: NO motion since the press = a
// plain click (replay it through the legacy path verbatim), motion = a
// copy-drag (extract, copy, verdict-gated toast — copySelectionCmd). The
// highlight stays up until cleared by esc / a plain click / a fresh drag.
func (m *Model) handleRelease(msg tea.MouseReleaseMsg) tea.Cmd {
	if m.sel != mselArmed || m.chat == nil {
		return nil
	}
	if !m.selDragged {
		m.sel = mselIdle
		m.chat.ClearSelection()
		return m.handleClick(tea.MouseClickMsg(m.selPress))
	}
	cx, cy := m.chatCoordsTranslate(msg.X, msg.Y)
	text, n := m.chat.SelectionFinish(cx, cy)
	if n == 0 {
		m.sel = mselIdle
		m.chat.ClearSelection()
		return nil // a zero-cell drag decides nothing (not even a clear-to-thread)
	}
	m.sel = mselSelected
	return m.copySelectionCmd(text, n)
}

// copySelectionCmd — the dragged release's clipboard effect. On darwin the
// copy is a REAL pbcopy round-trip (synchronous, error-checked — the
// verdict lands as clipboardResultMsg and the toast gates on it) with the
// OSC52 escape kept INSIDE the same batch as the leading fallback leaf:
// tea.SetClipboard is fire-and-forget, and the old OSC52-only path toasted
// unconditionally, so a swallowed escape (tmux without allow-passthrough,
// an unsupported terminal, a piped stdout) showed the confirmation while
// nothing reached the pasteboard. pbcopy is unaffected by terminal
// passthrough — it works inside tmux and over ssh. Elsewhere OSC52 remains
// the only channel (no OS feedback exists to gate on) and the note arms
// at release, as before.
func (m *Model) copySelectionCmd(text string, n int) tea.Cmd {
	if runtime.GOOS != "darwin" {
		return tea.Batch(tea.SetClipboard(text), m.armCopyNote(n))
	}
	return tea.Batch(
		tea.SetClipboard(text), // OSC52 fallback: ssh'd terminals that honor it still win
		func() tea.Msg {
			return clipboardResultMsg{n: n, err: CopyTextPBCopy(text)}
		},
	)
}

// CopyTextPBCopy runs darwin's pbcopy with the payload on stdin and checks
// the result: nil means the pasteboard round-trip really happened. Exported
// for the out-of-tree round-trip harness (/tmp proof programs built against
// the module path) — internal/app is still module-internal API.
func CopyTextPBCopy(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// armCopyNote toasts "Copied N chars" (the README's frozen copy) on the
// status bar and schedules its expiry. A later copy re-arms: the fresher
// note's tick owns its own expiry, a stale landing just no-ops.
func (m *Model) armCopyNote(n int) tea.Cmd {
	m.copyNote = fmt.Sprintf("Copied %d chars", n)
	m.copyNoteBad = false
	m.copyNoteAt = time.Now()
	return tea.Tick(copyNoteWindow, func(time.Time) tea.Msg { return copyNoteClearMsg{} })
}

// armCopyNoteErr toasts a FAILED copy on the same status-bar seam (warn
// class, same 2s window — hintLine picks the class off copyNoteBad): the
// copy verdict is now always REAL, never the old unconditional note.
func (m *Model) armCopyNoteErr(err error) tea.Cmd {
	m.copyNote = "Copy failed: " + err.Error()
	m.copyNoteBad = true
	m.copyNoteAt = time.Now()
	return tea.Tick(copyNoteWindow, func(time.Time) tea.Msg { return copyNoteClearMsg{} })
}

// chatContentCoords translates screen (x, y) into CHAT CONTENT coords when
// the point lands inside the chat tab's panel region (desktop sidebar or
// the full-width panel below the mobile floor band), ok=false otherwise.
func (m *Model) chatContentCoords(x, y int) (cx, cy int, ok bool) {
	if m.tabs.ActiveIndex() != 0 || m.chat == nil {
		return 0, 0, false
	}
	if m.mobile() {
		if y < 1+m.floorBandH() {
			return 0, 0, false
		}
		cx, cy = m.chatCoordsTranslate(x, y)
		return cx, cy, true
	}
	if x < m.floorW {
		return 0, 0, false
	}
	cx, cy = m.chatCoordsTranslate(x, y)
	return cx, cy, true
}

// chatCoordsTranslate — the pure layout translation (region-agnostic): the
// selection's drag/release clamps out-of-region coords down to the nearest
// transcript edge, so it needs the formula even outside the panel.
func (m *Model) chatCoordsTranslate(x, y int) (cx, cy int) {
	dx, dy := m.tabs.ContentOffset()
	if m.mobile() {
		return x - dx, y - (1 + m.floorBandH() + dy)
	}
	return x - (m.floorW + dx), y - (1 + dy)
}
