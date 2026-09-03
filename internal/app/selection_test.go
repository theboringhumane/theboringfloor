// selection_test.go — the app-side mouse-selection STATE MACHINE from the
// model level (fakes only, real Model + real Chat panel, scripted mouse
// events — the panel half's own geometry tests live in internal/panels):
//
//	(a) a left-PRESS over transcript text ARMS (sel idle→armed, the press
//	    pinned for the release verdict) and fires NO legacy click effect —
//	    the fold hint row under the press stays folded;
//	(b) the muscle-memory path press → motion → release: the dragged span
//	    copies to the clipboard (tea.SetClipboard batch leaf, real text
//	    asserted rune-for-rune), the status bar toasts the frozen
//	    "Copied N chars" note with N matching the dragged span — on darwin
//	    GATED on pbcopy's verdict (clipboardResultMsg), arming only when
//	    the pasteboard round-trip really happened — and the selection
//	    settles into SELECTED (highlight persists);
//	(c) a MOTIONLESS press+release returns to idle, clears the armed
//	    selection, and REPLAYS the original press through handleClick —
//	    the fold hint row under it toggles exactly like a plain click;
//	(d) a zero-cell drag (motion out and back to the origin column over the
//	    pad gutter, a span carrying no text) clears + no-ops: no clipboard
//	    batch, no toast, and no replayed click effect either;
//	(e) esc with an ACTIVE (finalized) selection clears the selection
//	    FIRST — the key never reaches the chat's double-esc stop seam (no
//	    stopWorkMsg leaf, the aborter seam never called, the status line
//	    never swings), and it does NOT stamp the chat's dbl-esc opener
//	    (a following esc is a lone opener, still no stop);
//	(f) press gating: /zen, the topbar/statusbar chrome rows, an open
//	    /model picker, and a non-chat active tab all reject the press —
//	    nothing arms, nothing selects;
//	(g) the copy VERDICT gates the toast: a pbcopy success verdict arms
//	    the frozen "Copied N chars" (OK class), a failure verdict arms a
//	    warn toast naming the error on the same seam (no real clipboard is
//	    touched — clipboardResultMsg is dispatched synthetically).
package app

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// selFoldMarker / selCollapseMarker — the rendered fold-row texts the
// replay assertions key on (panels' frozen copy, minus the count/em-dots).
const (
	selFoldMarker     = "more lines · click to expand"
	selCollapseMarker = "… collapse"
)

// selBackend — the live recording backend (sessBackend seams) PLUS the
// /stop aborter seam with a call counter: the esc test's "the stop path
// was not invoked" flag.
type selBackend struct {
	sessBackend
	aborts int
}

func (b *selBackend) AbortSessions() error { b.aborts++; return nil }

// selSetupModel — a sized desktop model (140x40 → the floor|sidebar split,
// width ≥ mobileMaxCols) on a scratch home; the transcript starts empty
// (fresh THEBORINGOFFICE_HOME → no session.json, no hydration notices).
func selSetupModel(t *testing.T, b state.Backend) Model {
	t.Helper()
	scratchHome(t)
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	return m
}

// selFeedFoldTurn — one 6-line user turn: beyond userFoldVisible, so it
// renders the folded shape (3 head rows + the clickable hint row) with
// lines 4..6 hidden until a click expands it.
func selFeedFoldTurn(t *testing.T, m Model) Model {
	t.Helper()
	return runMsg(t, m, state.Event{Kind: state.EvChatUser,
		Msg: state.ChatMsg{ID: "u-sel", From: "user", Kind: "user", At: 1,
			Text: "sel-line-1\nsel-line-2\nsel-line-3\nsel-line-4\nsel-line-5\nsel-line-6"}})
}

// selRowOf — the SCREEN row of the frame line containing needle (frame row
// 0 is the topbar: exactly the coordinate space handlePress consumes).
func selRowOf(t *testing.T, m Model, needle string) int {
	t.Helper()
	frame := ansi.Strip(m.Frame())
	for i, ln := range strings.Split(frame, "\n") {
		if strings.Contains(ln, needle) {
			return i
		}
	}
	t.Fatalf("the frame lost the transcript row %q:\n%s", needle, frame)
	return -1
}

// selChatX — the screen column over CHAT-CONTENT cell cx (the app's own
// translation seam, inverted: x = floorW + dx + cx).
func selChatX(m Model, cx int) int {
	dx, _ := m.tabs.ContentOffset()
	return m.floorW + dx + cx
}

// selClickAt / selDragAt / selUpAt — the scripted mouse events.
func selClickAt(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}
func selDragAt(x, y int) tea.MouseMotionMsg {
	return tea.MouseMotionMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}
func selUpAt(x, y int) tea.MouseReleaseMsg {
	return tea.MouseReleaseMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

// selUpdate drives ONE message through the real Update and hands back the
// cmd UNEXECUTED (the mouse paths return the copy batch's 2s expiry tick —
// the quitarm idiom: a sleeping tick is never executed, its stale landing
// only matters as a no-op).
func selUpdate(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	return nm.(Model), cmd
}

// selClipLeaf extracts the copied text from the release's returned cmd
// tree: descend BatchMsg wrappers executing ONLY each batch's FIRST child.
// tea.Batch preserves declaration order and Update wraps a non-empty cmds
// slice directly (compactCmds): the release batch declares SetClipboard
// first and the 2s expiry tick second — the tick is never executed.
func selClipLeaf(t *testing.T, cmd tea.Cmd) string {
	t.Helper()
	if cmd == nil {
		t.Fatalf("a dragged release must return the SetClipboard + expiry batch")
	}
	msg := cmd()
	for depth := 0; depth < 8; depth++ {
		batch, ok := msg.(tea.BatchMsg)
		if !ok {
			return fmt.Sprint(msg) // the SetClipboard leaf's text
		}
		if len(batch) == 0 || batch[0] == nil {
			t.Fatalf("the copy batch must carry the SetClipboard leaf first, got %#v", batch)
		}
		msg = batch[0]()
	}
	t.Fatalf("batch nesting deeper than the copy batch warrants")
	return ""
}

// selHasStopLeaf — did any leaf of a key cmd tree carry the double-esc
// stop seam's stopWorkMsg? (leafMsgs is safe on key trees: cursor blinks +
// spinner ticks are dropped, no sleeping tea.Tick hides here.)
func selHasStopLeaf(cmd tea.Cmd) bool {
	for _, leaf := range leafMsgs(cmd) {
		if _, ok := leaf.(stopWorkMsg); ok {
			return true
		}
	}
	return false
}

// selAssertFolded — the folded shape: head rows + hint marker, tail rows
// and the collapse trailer hidden.
func selAssertFolded(t *testing.T, m Model) {
	t.Helper()
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "sel-line-1") || !strings.Contains(frame, selFoldMarker) {
		t.Fatalf("the folded shape must show head rows + the hint:\n%s", frame)
	}
	if strings.Contains(frame, "sel-line-5") || strings.Contains(frame, selCollapseMarker) {
		t.Fatalf("a folded turn must hide the tail + the collapse trailer:\n%s", frame)
	}
}

// (a) press arms: no legacy click effect fires on the press alone.
func TestSelectionPressArmsWithoutClick(t *testing.T) {
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
	m = selFeedFoldTurn(t, m)
	selAssertFolded(t, m) // precondition

	y := selRowOf(t, m, selFoldMarker)
	x := selChatX(m, 8) // over the hint row's text
	t.Logf("pressing the fold-hint row at screen (%d, %d)", x, y)

	m, cmd := selUpdate(t, m, selClickAt(x, y))

	if m.sel != mselArmed {
		t.Fatalf("a press over transcript text must ARM the selection, sel=%d", m.sel)
	}
	if m.selDragged {
		t.Fatalf("the press alone must not mark the drag")
	}
	if m.selPress.X != x || m.selPress.Y != y {
		t.Fatalf("the original press must be pinned for the release verdict, got (%d, %d)",
			m.selPress.X, m.selPress.Y)
	}
	if !m.chat.SelectionActive() {
		t.Fatalf("the armed selection is visible panel-side (esc-visibility contract)")
	}
	if cmd != nil {
		t.Fatalf("an arming press fires NOTHING (the click's fate waits for release)")
	}
	// NO legacy click effect: the fold hint row under the press must NOT
	// have toggled (a plain click would have — proven in (c)).
	selAssertFolded(t, m)
}

// (b) muscle memory: press + motion + release copies the dragged span.
func TestSelectionDragCopies(t *testing.T) {
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser,
		Msg: state.ChatMsg{ID: "u-drag", From: "user", Kind: "user", Text: "sel-drag line target", At: 1}})

	y := selRowOf(t, m, "sel-drag line target")
	x0 := selChatX(m, 0) // the chatPadL gutter: spans clamp to [pad, line end)
	t.Logf("dragging across the user row at screen row %d (press x=%d)", y, x0)

	m, cmd := selUpdate(t, m, selClickAt(x0, y))
	if m.sel != mselArmed || cmd != nil {
		t.Fatalf("precondition: the press arms (sel=%d) and fires nothing (cmd=%v)", m.sel, cmd)
	}
	// motion carries the head across the row — the drag is live
	m, _ = selUpdate(t, m, selDragAt(selChatX(m, 6), y))
	m, _ = selUpdate(t, m, selDragAt(selChatX(m, 16), y))
	if m.sel != mselArmed || !m.selDragged {
		t.Fatalf("motion must keep the arm and mark the drag (sel=%d dragged=%v)", m.sel, m.selDragged)
	}
	t.Logf("press → sel=%d (armed); motion ×2 → selDragged=%v", mselArmed, m.selDragged)

	// release at the far right of the same row → SelectionFinish path
	m, relCmd := selUpdate(t, m, selUpAt(m.width-2, y))

	// the whole posted row minus the pad: "you › " + the turn text
	expected := "you › sel-drag line target"
	wantN := len([]rune(expected))
	if m.sel != mselSelected {
		t.Fatalf("a dragged release settles SELECTED, sel=%d", m.sel)
	}
	if !m.chat.SelectionActive() {
		t.Fatalf("the finished highlight persists until esc / a plain click / a fresh arm")
	}
	wantNote := fmt.Sprintf("Copied %d chars", wantN)
	if relCmd == nil {
		t.Fatalf("the release returns the clipboard + verdict batch (non-nil cmd)")
	}
	if runtime.GOOS == "darwin" {
		// darwin GATES the toast on pbcopy's verdict: the release alone
		// arms NOTHING (a swallowed OSC52 escape used to lie here).
		if m.copyNote != "" {
			t.Fatalf("the toast must wait for pbcopy's verdict on darwin, got %q", m.copyNote)
		}
		t.Logf("release → copyNote empty: the toast gates on the pbcopy round-trip")
		m, _ = selUpdate(t, m, clipboardResultMsg{n: wantN}) // the verdict lands
	}
	if m.copyNote != wantNote {
		t.Fatalf("the frozen toast must count the dragged span: want %q, got %q", wantNote, m.copyNote)
	}
	if strings.Contains(m.copyNote, "Copied 0 chars") {
		t.Fatalf("the drag carried real text — n must be > 0")
	}
	if hint := m.hintLine(); !strings.Contains(hint, wantNote) {
		t.Fatalf("the copy note rides the status-bar seam, hint=%q", hint)
	}
	// the clipboard leaf carries EXACTLY the dragged span's text
	if got := selClipLeaf(t, relCmd); got != expected {
		t.Fatalf("the copied text must be the dragged span verbatim: want %q, got %q", expected, got)
	}
	t.Logf("release → sel=%d (selected); copyNote=%q; clipboard leaf=%q (%d runes)",
		m.sel, m.copyNote, expected, wantN)
}

// (c) motionless release: the selection clears and the ORIGINAL press
// replays through handleClick (fold hint row toggles like a plain click).
func TestSelectionMotionlessReleaseReplays(t *testing.T) {
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
	m = selFeedFoldTurn(t, m)
	selAssertFolded(t, m) // precondition

	y := selRowOf(t, m, selFoldMarker)
	x := selChatX(m, 8)

	m, _ = selUpdate(t, m, selClickAt(x, y))
	if m.sel != mselArmed {
		t.Fatalf("precondition: the press armed")
	}
	m, _ = selUpdate(t, m, selUpAt(x, y)) // same cell, zero motion

	if m.sel != mselIdle {
		t.Fatalf("a motionless release settles back to idle, sel=%d", m.sel)
	}
	if m.chat.SelectionActive() {
		t.Fatalf("the replay path clears the armed selection first")
	}
	// the replay's observable legacy effect: the fold hint row under the
	// replayed press toggled — tail rows + the "… collapse" trailer show.
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "sel-line-5") || !strings.Contains(frame, selCollapseMarker) {
		t.Fatalf("the replayed press must expand the folded turn (tail + trailer):\n%s", frame)
	}
	if strings.Contains(frame, selFoldMarker) {
		t.Fatalf("the hint row is gone once expanded:\n%s", frame)
	}
	t.Logf("motionless press+release on the hint row → sel=%d, fold expanded (replay fired)", m.sel)
}

// (d) zero-cell drag (dragged but back to the origin pad cell, a span
// carrying no text): clear + no-op — nothing copies, nothing replays.
func TestSelectionZeroCellDragNoops(t *testing.T) {
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
	m = selFeedFoldTurn(t, m)
	selAssertFolded(t, m) // precondition

	y := selRowOf(t, m, selFoldMarker)
	x0 := selChatX(m, 0) // origin: the pad gutter (extraction clamps it empty)

	m, _ = selUpdate(t, m, selClickAt(x0, y))
	if m.sel != mselArmed {
		t.Fatalf("precondition: the press armed")
	}
	m, _ = selUpdate(t, m, selDragAt(selChatX(m, 8), y)) // out into the text…
	if !m.selDragged {
		t.Fatalf("precondition: the motion marked the drag")
	}
	m, relCmd := selUpdate(t, m, selUpAt(x0, y)) // …and back to the origin cell

	if m.sel != mselIdle {
		t.Fatalf("a textless span decides nothing: sel must return to idle, got %d", m.sel)
	}
	if m.chat.SelectionActive() {
		t.Fatalf("the zero-cell drag clears the highlight")
	}
	if m.copyNote != "" {
		t.Fatalf("no span, no toast: copyNote must stay empty, got %q", m.copyNote)
	}
	if relCmd != nil {
		t.Fatalf("no clipboard batch, no expiry tick — the release is a pure no-op (cmd=%v)", relCmd)
	}
	// no replayed side-effects either: the fold never toggled.
	selAssertFolded(t, m)
}

// (e) esc with an ACTIVE selection clears the selection FIRST; the
// double-esc stop seam never fires — not on the clearing press, and not on
// the next one either (the first never stamped the chat's dbl-esc opener).
func TestSelectionEscClearsSelectionFirst(t *testing.T) {
	b := &selBackend{sessBackend: sessBackend{primary: "ses-sel"}}
	m := selSetupModel(t, b)
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser,
		Msg: state.ChatMsg{ID: "u-esc", From: "user", Kind: "user", Text: "sel-drag line target", At: 1}})

	y := selRowOf(t, m, "sel-drag line target")
	m, _ = selUpdate(t, m, selClickAt(selChatX(m, 0), y))
	m, _ = selUpdate(t, m, selDragAt(selChatX(m, 12), y))
	m, relCmd := selUpdate(t, m, selUpAt(m.width-2, y))
	if runtime.GOOS == "darwin" {
		m, _ = selUpdate(t, m, clipboardResultMsg{n: 1}) // the verdict the toast gates on
	}
	if m.sel != mselSelected || !m.chat.SelectionActive() || m.copyNote == "" || relCmd == nil {
		t.Fatalf("precondition: a finished selection is up (sel=%d active=%v note=%q)",
			m.sel, m.chat.SelectionActive(), m.copyNote)
	}
	esc := tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})

	// esc #1: the key belongs to the selection — clear, nothing else.
	m, escCmd := selUpdate(t, m, esc)
	if m.sel != mselIdle {
		t.Fatalf("esc must return the selection state to idle, sel=%d", m.sel)
	}
	if m.chat.SelectionActive() {
		t.Fatalf("esc must clear the highlight")
	}
	if selHasStopLeaf(escCmd) {
		t.Fatalf("the selection's esc must never reach the double-esc stop seam")
	}
	if b.aborts != 0 {
		t.Fatalf("the aborter seam must stay untouched, aborts=%d", b.aborts)
	}
	if strings.Contains(m.st.StatusLine, "stopped current work") {
		t.Fatalf("the stop unwind must not run, status=%q", m.st.StatusLine)
	}

	// esc #2: with the highlight gone the key flows to the chat — but as a
	// LONE opener (esc #1 never stamped the dbl-esc clock), still no stop.
	m, escCmd2 := selUpdate(t, m, esc)
	if selHasStopLeaf(escCmd2) {
		t.Fatalf("the esc AFTER the clearing esc pairs with nothing — still no stop")
	}
	if b.aborts != 0 || strings.Contains(m.st.StatusLine, "stopped current work") {
		t.Fatalf("the stop path must not fire on a lone opener (aborts=%d status=%q)",
			b.aborts, m.st.StatusLine)
	}
	t.Logf("esc → sel=%d, highlight off; two esc presses, stop seam fired %d time(s)", m.sel, b.aborts)
}

// (f) press gating: /zen, the chrome rows, an open /model picker, and a
// non-chat active tab all reject the press.
func TestSelectionPressGating(t *testing.T) {

	// (f-1) /zen: the transient fullscreen floor owns every press.
	t.Run("zen", func(t *testing.T) {
		m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
		m = selFeedFoldTurn(t, m)
		y := selRowOf(t, m, selFoldMarker)
		x := selChatX(m, 8) // mined pre-/zen: the zen frame hides the sidebar
		m = runMsg(t, m, slashMsg{text: "/zen"})
		if !m.zen {
			t.Fatalf("precondition: /zen is active")
		}
		m, cmd := selUpdate(t, m, selClickAt(x, y))
		if m.sel != mselIdle || m.chat.SelectionActive() || cmd != nil {
			t.Fatalf("zen owns every press: sel=%d active=%v cmd=%v", m.sel, m.chat.SelectionActive(), cmd)
		}
	})

	// (f-2) the 2-cell chrome (topbar row 0, statusbar row height-1).
	t.Run("chrome rows", func(t *testing.T) {
		m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
		m = selFeedFoldTurn(t, m)
		x := selChatX(m, 8)
		for _, y := range []int{0, m.height - 1} {
			m, cmd := selUpdate(t, m, selClickAt(x, y))
			if m.sel != mselIdle || m.chat.SelectionActive() || cmd != nil {
				t.Fatalf("the chrome row y=%d never arms: sel=%d active=%v cmd=%v",
					y, m.sel, m.chat.SelectionActive(), cmd)
			}
		}
	})

	// (f-3) the keys-only /model picker: a click lands on NOTHING
	// underneath — no selection arm either.
	t.Run("model picker open", func(t *testing.T) {
		b := &modelsBackend{models: modelsFixture()}
		m := selSetupModel(t, b)
		m = selFeedFoldTurn(t, m)
		y := selRowOf(t, m, selFoldMarker) // mine pre-open: the card splices over the frame
		x := selChatX(m, 8)
		m = runMsg(t, m, slashMsg{text: "/model"})
		if !m.ModelPickerOpen() {
			t.Fatalf("precondition: the /model picker is open")
		}
		m, cmd := selUpdate(t, m, selClickAt(x, y))
		if m.sel != mselIdle || m.chat.SelectionActive() || cmd != nil {
			t.Fatalf("the picker swallows the press: sel=%d active=%v cmd=%v",
				m.sel, m.chat.SelectionActive(), cmd)
		}
	})

	// (f-4) a non-chat active tab: the transcript coords belong nowhere.
	t.Run("non-chat tab", func(t *testing.T) {
		m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})
		m = selFeedFoldTurn(t, m)
		y := selRowOf(t, m, selFoldMarker)
		x := selChatX(m, 8)
		m.tabs.SetActive(5) // the activity tab
		if m.ActiveTabIndex() == 0 {
			t.Fatalf("precondition: the chat tab is NOT active")
		}
		m, cmd := selUpdate(t, m, selClickAt(x, y))
		if m.sel != mselIdle || m.chat.SelectionActive() || cmd != nil {
			t.Fatalf("only the chat tab arms selections: sel=%d active=%v cmd=%v",
				m.sel, m.chat.SelectionActive(), cmd)
		}
	})
}

// (g) the copy verdict gates the toast: clipboardResultMsg is the ONLY darwin
// toast trigger — success arms "Copied N chars" (OK), failure arms a warn
// toast naming the error on the same seam. Synthetic msgs: no real clipboard.
func TestSelectionCopyVerdictGatesToast(t *testing.T) {
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-sel"}})

	// success verdict → the frozen note, OK class.
	m, okCmd := selUpdate(t, m, clipboardResultMsg{n: 42})
	if m.copyNote != "Copied 42 chars" {
		t.Fatalf("a success verdict arms the frozen toast, got %q", m.copyNote)
	}
	if m.copyNoteBad {
		t.Fatalf("a success verdict never rides the warn class")
	}
	if okCmd == nil {
		t.Fatalf("the verdict arms the note's own expiry tick")
	}
	if hint := m.hintLine(); !strings.Contains(hint, "Copied 42 chars") {
		t.Fatalf("the toasted verdict rides the status-bar seam, hint=%q", hint)
	}

	// failure verdict → the error toast, warn class, same seam.
	m, errCmd := selUpdate(t, m, clipboardResultMsg{err: errors.New("pbcopy: exit status 1")})
	if m.copyNote != "Copy failed: pbcopy: exit status 1" {
		t.Fatalf("a failure verdict must NAME the error, got %q", m.copyNote)
	}
	if !m.copyNoteBad {
		t.Fatalf("a failure verdict rides the warn class")
	}
	if errCmd == nil {
		t.Fatalf("a failed copy's note still arms its expiry tick")
	}
	if hint := m.hintLine(); !strings.Contains(hint, "Copy failed: pbcopy: exit status 1") {
		t.Fatalf("the failure toast rides the same status-bar seam, hint=%q", hint)
	}
	if strings.Contains(m.copyNote, "Copied ") {
		t.Fatalf("a failed copy must never toast a success, note=%q", m.copyNote)
	}
	t.Logf("verdicts → success %q (OK) / failure %q (warn)", "Copied 42 chars", m.copyNote)
}
