// selection.go — webpage-style mouse text selection over the chat
// transcript: a left-press over selectable transcript text ARMS a pending
// selection (the click's fate — legacy click action vs copy-drag — stays
// undecided until release, so single-click semantics survive untouched),
// drag-motion extends the head (the chat panel highlights the span live,
// chat_selection.go), and a dragged release copies the plain text via the
// terminal's OSC52 clipboard (tea.SetClipboard) and toasts the status bar
// with the frozen "Copied N chars" note. A motionless release replays the
// ORIGINAL press through handleClick, so thread toggles, fold rows, perm
// cards and floor picks behave exactly like a plain click.
//
// The copy note rides the same status-bar seam as the ctrl+q quit arm:
// hintLine swaps to it while fresh and its own tea.Tick retires it
// (copyNoteClearMsg); a fresher note's tick owns its expiry.
package app

import (
	"fmt"
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
	if cx, cy, ok := m.chatContentCoords(msg.X, msg.Y); ok && m.chat.SelectionBegin(cx, cy) {
		m.sel = mselArmed
		m.selPress = tea.Mouse(msg)
		m.selDragged = false
		return nil
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
// copy-drag (extract, OSC52-copy, toast). The highlight stays up until
// cleared by esc / a plain click / a fresh drag.
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
	return tea.Batch(tea.SetClipboard(text), m.armCopyNote(n))
}

// armCopyNote toasts "Copied N chars" (the README's frozen copy) on the
// status bar and schedules its expiry. A later copy re-arms: the fresher
// note's tick owns its own expiry, a stale landing just no-ops.
func (m *Model) armCopyNote(n int) tea.Cmd {
	m.copyNote = fmt.Sprintf("Copied %d chars", n)
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
