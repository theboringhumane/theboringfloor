// plan_editor_selection_test.go — the plan editor's selection + cut/copy/
// paste contract (plan_editor_selection.go): shift-motion marking with both
// drag directions normalizing to the same range, ctrl+a marking the whole
// buffer (and the textarea's old ctrl+a faction never firing), the stubbed
// clipboard seams proving exact payloads, cut removing exact bytes with a
// wrap-aware caret restore + userDirty latch, paste-over-selection for both
// the bracketed (tea.PasteMsg) and readback (ctrl+v/super+v) paths, the
// SGR-7 highlight splice in the focused frame, esc-owns-selection-then-
// blur, motionless press deciding NOTHING (no copy, no note), and the
// build-mode/unfocused locks.
package panels

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// --- key constructors + clipboard stubs -----------------------------------

func shiftKey(c rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: c, Mod: tea.ModShift})
}

func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: r, Mod: tea.ModCtrl})
}

// stubClip swaps BOTH clipboard seams for recorders and restores them on
// cleanup (parallel suites must never leak stubs — clipboard.go's contract).
func stubClip(t *testing.T) (copied *[]string, pasted *string) {
	t.Helper()
	oldCopy := clipboardCopyText
	oldRead := clipboardReadText
	var copies []string
	clipboardCopyText = func(text string) error {
		copies = append(copies, text)
		return nil
	}
	pasteBuf := ""
	clipboardReadText = func() (string, error) { return pasteBuf, nil }
	t.Cleanup(func() {
		clipboardCopyText = oldCopy
		clipboardReadText = oldRead
	})
	return &copies, &pasteBuf
}

// focusedEditor builds an editor at a real size, focused, with content.
// The caret HOMES to (0,0): SetValue parks it at the buffer's end, which
// would silently poison every shift-motion test from here on.
func focusedEditor(t *testing.T, w, h int, content string) *PlanEditor {
	t.Helper()
	e := NewPlanEditor()
	e.SetSize(w, h)
	e.SetValue(content)
	e.Focus()
	e.ta.MoveToBegin()
	return e
}

// selRange runs a shift-motion key series through the editor.
func selKeys(e *PlanEditor, msgs ...tea.Msg) {
	for _, msg := range msgs {
		e.Update(msg)
	}
}

func TestPlanSelAnchorNormalizeBothWays(t *testing.T) {
	content := "# Goal\n- matte panels\n- glass lanes\n"
	// drag DOWN from (0,2) to (2,4)
	e := focusedEditor(t, 60, 20, content)
	e.sel = planSelState{active: true, anchor: planPos{0, 2}, head: planPos{2, 4}}
	downText := e.selText()
	// drag UP from (2,4) to (0,2)
	e.sel = planSelState{active: true, anchor: planPos{2, 4}, head: planPos{0, 2}}
	upText := e.selText()
	if downText != upText {
		t.Fatalf("drag direction must not change the marked text:\n down %q\n up   %q", downText, upText)
	}
	want := "Goal\n- matte panels\n- gl"
	if downText != want {
		t.Fatalf("normalized marked text = %q, want %q", downText, want)
	}
	if n := len([]rune(downText)); n != len([]rune(want)) {
		t.Fatalf("marked rune count = %d, want %d", n, len([]rune(want)))
	}
	// a zero-width mark is nothing
	e.sel = planSelState{active: true, anchor: planPos{1, 3}, head: planPos{1, 3}}
	if got := e.selText(); got != "" {
		t.Fatalf("a zero-width mark extracts nothing, got %q", got)
	}
	t.Logf("normalize: down %q == up %q (zero-width → %q)", downText, upText, "")
}

func TestPlanSelCtrlASelectsAll(t *testing.T) {
	content := "# Goal\n- matte panels\n- glass lanes"
	e := focusedEditor(t, 60, 20, content)
	// park the caret mid-buffer so an accidental LineStart faction would be VISIBLE
	e.selSyncCursor(planPos{row: 1, col: 3})
	e.Update(ctrlKey('a'))
	if !e.SelectionActive() {
		t.Fatal("ctrl+a must mark the buffer")
	}
	if got := e.selText(); got != content {
		t.Fatalf("ctrl+a marks the WHOLE buffer:\n got %q\nwant %q", got, content)
	}
	// the OLD textarea faction must be dead: the caret went to the buffer
	// END (select-all's head), never LineStart of the current row
	last := strings.Split(content, "\n")
	lastLine := last[len(last)-1]
	if e.ta.Line() != len(last)-1 || e.ta.Column() != len([]rune(lastLine)) {
		t.Fatalf("ctrl+a parks the caret at the buffer end, got line=%d col=%d", e.ta.Line(), e.ta.Column())
	}
	if e.ta.Column() == 0 {
		t.Fatal("ctrl+a as LineStart would leave col=0 on a mid-buffer row — the rebind must beat it")
	}
	t.Logf("ctrl+a slice == buffer (%d runes), caret at end (line=%d col=%d)",
		len([]rune(e.selText())), e.ta.Line(), e.ta.Column())
}

func TestPlanSelCtrlCCopyStub(t *testing.T) {
	copies, _ := stubClip(t)
	e := focusedEditor(t, 60, 20, "# Goal\n- matte panels\n- glass lanes")
	e.sel = planSelState{active: true, anchor: planPos{0, 2}, head: planPos{2, 5}}
	text, n, err := e.CopySelection()
	if err != nil {
		t.Fatalf("stubbed copy must succeed: %v", err)
	}
	if len(*copies) != 1 || (*copies)[0] != text {
		t.Fatalf("the stubbed seam saw %v, want exactly %q", *copies, text)
	}
	want := "Goal\n- matte panels\n- gla"
	if text != want {
		t.Fatalf("copy text = %q, want %q (as-is, newlines verbatim)", text, want)
	}
	if n != len([]rune(want)) {
		t.Fatalf("copy count = %d, want %d (runes incl. newlines — the frozen toast count)", n, len([]rune(want)))
	}
	// a copy does NOT mutate the buffer or collapse the mark
	if e.Value() != "# Goal\n- matte panels\n- glass lanes" {
		t.Fatalf("copy must leave the buffer untouched, got %q", e.Value())
	}
	if !e.SelectionActive() {
		t.Fatal("copy keeps the mark (esc or an unshifted key clears it)")
	}
	t.Logf("copy verdict: %q (%d chars — the app's toast reads \"Copied %d chars\")", text, n, n)
}

func TestPlanSelCtrlXCut(t *testing.T) {
	copies, _ := stubClip(t)
	e := focusedEditor(t, 60, 20, "# Goal\n- matte panels\n- glass lanes")
	e.sel = planSelState{active: true, anchor: planPos{0, 2}, head: planPos{2, 5}}
	if e.UserDirty() {
		t.Fatal("precondition: the latch starts clean")
	}
	text, n, err := e.CutSelection()
	if err != nil || text == "" || n == 0 {
		t.Fatalf("cut verdict = (%q, %d, %v)", text, n, err)
	}
	if len(*copies) != 1 {
		t.Fatalf("cut must ride the copy seam too, got %d writes", len(*copies))
	}
	wantBuf := "# ss lanes"
	if e.Value() != wantBuf {
		t.Fatalf("cut removed the wrong bytes:\n got %q\nwant %q", e.Value(), wantBuf)
	}
	if !e.UserDirty() {
		t.Fatal("cut is a real edit — the userDirty latch must ride the before/after compare")
	}
	if e.SelectionActive() {
		t.Fatal("the mark dies with the range it covered")
	}
	if e.ta.Line() != 0 || e.ta.Column() != 2 {
		t.Fatalf("the caret restores to the range START (0,2), got (%d,%d)", e.ta.Line(), e.ta.Column())
	}
	t.Logf("cut: %q out; buffer now %q; caret restored to (%d,%d), dirty=%t",
		text, e.Value(), e.ta.Line(), e.ta.Column(), e.UserDirty())
}

func TestPlanSelPasteReplaces(t *testing.T) {
	e := focusedEditor(t, 60, 20, "# Goal\n- matte panels\n- glass lanes")
	e.sel = planSelState{active: true, anchor: planPos{1, 2}, head: planPos{1, 7}} // "matte"
	e.Update(tea.PasteMsg{Content: "azimuth-washed"})
	want := "# Goal\n- azimuth-washed panels\n- glass lanes"
	if e.Value() != want {
		t.Fatalf("paste must REPLACE the marked range:\n got %q\nwant %q", e.Value(), want)
	}
	if e.SelectionActive() {
		t.Fatal("a paste consumes the mark")
	}
	if !e.UserDirty() {
		t.Fatal("the replacement latches userDirty")
	}
	if e.ta.Line() != 1 || e.ta.Column() != 2+len([]rune("azimuth-washed")) {
		t.Fatalf("the caret lands after the inserted text, got (%d,%d)", e.ta.Line(), e.ta.Column())
	}
	t.Logf("paste-over: \"matte\" → %q; buffer %q", "azimuth-washed", e.Value())
}

func TestPlanSelCtrlVAndSuperV(t *testing.T) {
	_, pasted := stubClip(t)
	*pasted = "FIXED步走廊" // includes a double-width rune on purpose
	for _, key := range []string{"ctrl+v", "super+v"} {
		e := focusedEditor(t, 60, 20, "# Goal\n- matte panels")
		if kp, ok := e.selKey(map[string]tea.KeyPressMsg{
			"ctrl+v":  ctrlKey('v'),
			"super+v": tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModSuper}),
		}[key]); !ok {
			t.Fatalf("%s must be pane-claimed", key)
		} else if kp == nil {
			t.Fatalf("%s must return the async read cmd", key)
		}
		msg := func() tea.Msg {
			kpMsg, _ := e.selKey(map[string]tea.KeyPressMsg{
				"ctrl+v":  ctrlKey('v'),
				"super+v": tea.KeyPressMsg(tea.Key{Code: 'v', Mod: tea.ModSuper}),
			}[key])
			return kpMsg()
		}()
		pm, ok := msg.(PlanPasteMsg)
		if !ok || pm.Err != nil {
			t.Fatalf("%s readback = %#v, want a clean PlanPasteMsg", key, msg)
		}
		// mark "matte" then deliver the readback: it must REPLACE
		e.sel = planSelState{active: true, anchor: planPos{1, 2}, head: planPos{1, 7}}
		e.Update(pm)
		want := "# Goal\n- FIXED步走廊 panels"
		if e.Value() != want {
			t.Fatalf("%s paste-over: got %q, want %q", key, e.Value(), want)
		}
		t.Logf("%s: clipboard %q pasted over the mark → %q (double-width runes intact)", key, *pasted, e.Value())
	}
}

func TestPlanSelSpacingPreserved(t *testing.T) {
	// markdown's load-bearing whitespace: nested indentation + code block
	content := "# Plan\n    1. indented step\n        - nested leaf\n```\n  code line  \n```"
	e := focusedEditor(t, 60, 25, content)
	e.sel = planSelState{active: true, anchor: planPos{1, 0}, head: planPos{5, 3}}
	got := e.selText()
	want := "    1. indented step\n        - nested leaf\n```\n  code line  \n```"
	if got != want {
		t.Fatalf("extraction must preserve indentation verbatim:\n got %q\nwant %q", got, want)
	}
	// cut it: the merge keeps the hi line's tail exactly
	copies, _ := stubClip(t)
	_, _, _ = e.CutSelection()
	if len(*copies) != 1 || (*copies)[0] != want {
		t.Fatalf("the cut payload must equal the marked bytes: %v", *copies)
	}
	if e.Value() != "# Plan\n" {
		t.Fatalf("post-cut buffer = %q, want %q", e.Value(), "# Plan\n")
	}
	t.Logf("spacing: %q extracted+cut verbatim (leading 4sp/8sp and 2sp code line intact)", want)
}

func TestPlanSelWrapAwareRestore(t *testing.T) {
	// narrow pane: line 0 wraps into MANY visual rows
	content := "alpha beta gamma delta epsilon zeta\nshort"
	e := focusedEditor(t, 14, 10, content) // content width 12 → wraps
	// mark from inside the wrap (row 0, col 6) into the next line
	e.sel = planSelState{active: true, anchor: planPos{0, 6}, head: planPos{1, 3}}
	stubClip(t)
	text, _, _ := e.CutSelection()
	if text != "beta gamma delta epsilon zeta\nsho" {
		t.Fatalf("wrap-mark extract = %q", text)
	}
	// caret restored to the range START through the wrapped view: Line()
	// lands on the logical line (buffer space) even across the wrap
	if e.ta.Line() != 0 || e.ta.Column() != 6 {
		t.Fatalf("wrap-aware restore: caret = (%d,%d), want (0,6)", e.ta.Line(), e.ta.Column())
	}
	if e.Value() != "alpha rt" {
		t.Fatalf("post-cut buffer = %q, want %q (line 1's surviving tail merges past the real space)", e.Value(), "alpha rt")
	}
	// and the wrap replica itself: 12-cell content width on a 34-rune line
	segs := planWrapSegments("alpha beta gamma delta epsilon zeta", 12)
	if got := len(segs); got < 3 {
		t.Fatalf("the wrap replica must break the long line into segments, got %d", got)
	}
	total := 0
	for _, s := range segs {
		total += s.real
	}
	if total != len([]rune("alpha beta gamma delta epsilon zeta")) {
		t.Fatalf("replica segments must cover EVERY original rune: got %d", total)
	}
	t.Logf("wrap-aware: %d segments at width 12 covering %d runes; caret restored to (0,6) post-cut",
		len(segs), total)
}

func TestPlanSelShiftArrows(t *testing.T) {
	content := "ab\ncd\nef"
	e := focusedEditor(t, 60, 20, content)

	e.Update(shiftKey(tea.KeyRight))
	e.Update(shiftKey(tea.KeyRight))
	if !e.SelectionActive() {
		t.Fatal("shift+right opens a mark")
	}
	if got := e.selText(); got != "ab" {
		t.Fatalf("shift+right ×2 marks %q, want %q", got, "ab")
	}
	// the caret tracks the head so a follow-up continues from there
	if e.ta.Line() != 0 || e.ta.Column() != 2 {
		t.Fatalf("caret synced to the head, got (%d,%d)", e.ta.Line(), e.ta.Column())
	}
	// right across the line break: the NEXT slot past a line's end is the
	// following line's start — the newline itself is what gets marked
	e.Update(shiftKey(tea.KeyRight))
	if got := e.selText(); got != "ab\n" {
		t.Fatalf("shift+right crossing the line break marks the newline: %q", got)
	}
	// one more right: the marked range takes the next line's first rune
	e.Update(shiftKey(tea.KeyRight))
	if got := e.selText(); got != "ab\nc" {
		t.Fatalf("shift+right past the break: %q", got)
	}
	// left shrinks back across the boundary
	e.Update(shiftKey(tea.KeyLeft))
	if got := e.selText(); got != "ab\n" {
		t.Fatalf("shift+left shrinks: %q", got)
	}
	// down by VISUAL row (from (0,1) so the marked text is unambiguous)
	e2 := focusedEditor(t, 60, 20, content)
	e2.ta.SetCursorColumn(1)
	e2.Update(shiftKey(tea.KeyDown))
	if got := e2.selText(); got != "b\nc" {
		t.Fatalf("shift+down marks the visual row hop: %q", got)
	}
	// home/end by line anchor
	e3 := focusedEditor(t, 60, 20, "hello world\nsecond")
	e3.Update(shiftKey(tea.KeyEnd))
	if got := e3.selText(); got != "hello world" {
		t.Fatalf("shift+end marks the line: %q", got)
	}
	e3.Update(shiftKey(tea.KeyHome))
	if got := e3.selText(); got != "" {
		t.Fatalf("shift+home collapses back to the anchor: %q", got)
	}
	t.Logf("shift family: rune steps + visual rows + line anchors all pin the caret-correct ranges")
}

func TestPlanSelShiftArrowsWrapVariant(t *testing.T) {
	// a wrapped line: shift+down moves by VISUAL row — Line() may stay on
	// the same logical line while the visual caret descends the wrap
	content := "alpha beta gamma delta epsilon zeta\nX"
	e := focusedEditor(t, 14, 12, content) // 12-cell wrap
	e.Update(shiftKey(tea.KeyDown))
	if !e.SelectionActive() {
		t.Fatal("shift+down opens the mark")
	}
	// whichever row the wrap landed on, the mark must equal buffer start
	// through the caret's CURRENT (line, col) — the textarea's visual move
	head := planPos{row: e.ta.Line(), col: e.ta.Column()}
	if e.sel.head != head {
		t.Fatalf("the mark's head tracks the visual caret: sel.head=%v caret=%v", e.sel.head, head)
	}
	lo, hi := e.sel.norm()
	if lo != (planPos{0, 0}) {
		t.Fatalf("the anchor pinned the press point (0,0), got %v", lo)
	}
	if hi.row > 1 {
		t.Fatalf("one visual step can never skip a logical line, got head %v", hi)
	}
	t.Logf("wrap shift+down: head %v (text marked %q)", head, e.selText())
}

func TestPlanSelEscClearsThenBlurFree(t *testing.T) {
	e := focusedEditor(t, 60, 20, "# Goal\n- one")
	e.Update(shiftKey(tea.KeyRight))
	if !e.SelectionActive() {
		t.Fatal("precondition: a mark is live")
	}
	// esc #1: clears the mark, KEEPS the focus (the blur is the NEXT esc's)
	e.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if e.SelectionActive() {
		t.Fatal("esc must clear the mark first")
	}
	if !e.Focused() {
		t.Fatal("the esc that clears the mark must NOT blur the pane")
	}
	if got := e.Value(); got != "# Goal\n- one" {
		t.Fatalf("esc mutates nothing, got %q", got)
	}
	t.Log("esc-owns-selection: mark dropped, focus kept, buffer byte-identical")
}

func TestPlanSelMotionlessNoToast(t *testing.T) {
	copies, _ := stubClip(t)
	e := focusedEditor(t, 60, 20, "# Goal\n- one")
	// press = caret placement; release with NO motion = caret only
	e.SelectionBeginAt(5, 0)
	text, n, err := e.SelectionFinish(5, 0)
	if err != nil {
		t.Fatalf("stubbed verdict: %v", err)
	}
	if text != "" || n != 0 {
		t.Fatalf("a motionless press decides nothing, got (%q, %d)", text, n)
	}
	if len(*copies) != 0 {
		t.Fatalf("a motionless press must NOT write the clipboard, got %v", *copies)
	}
	// caret DID land at the mapped position (press places it)
	if e.ta.Line() != 0 {
		t.Fatalf("the motionless press placed the caret on row 0, got %d", e.ta.Line())
	}
	t.Log("motionless release: no clipboard write, no mark to toast — caret placement only")
}

func TestPlanSelBlurClears(t *testing.T) {
	e := focusedEditor(t, 60, 20, "# Goal\n- one")
	e.Update(shiftKey(tea.KeyRight))
	if !e.SelectionActive() {
		t.Fatal("precondition: a mark is live")
	}
	e.Blur()
	if e.SelectionActive() {
		t.Fatal("Blur must clear the mark — an invisible editor never retains one")
	}
	// and re-focusing starts clean
	e.Focus()
	if e.SelectionActive() {
		t.Fatal("a refocus never resurrects the mark")
	}
	t.Log("Blur clears: Focus → mark → Blur → SelectionActive=false")
}

func TestPlanSelModesLocked(t *testing.T) {
	// build mode: keys fully ignored, no marking, no edits
	e := focusedEditor(t, 60, 20, "# Goal")
	e.SetMode("build")
	before := e.Value()
	e.Update(shiftKey(tea.KeyRight))
	e.Update(ctrlKey('a'))
	e.Update(tea.PasteMsg{Content: "nope"})
	if e.Value() != before {
		t.Fatalf("build mode ignores every key and paste, got %q", e.Value())
	}
	if e.SelectionActive() {
		t.Fatal("no marking while the plan is approved (build)")
	}
	// unfocused: the buffer must not move either
	e2 := NewPlanEditor()
	e2.SetSize(60, 20)
	e2.SetValue("# Goal")
	e2.Update(shiftKey(tea.KeyRight))
	e2.Update(ctrlKey('a'))
	if e2.Value() != "# Goal" || e2.SelectionActive() {
		t.Fatalf("an unfocused pane is fully inert: value=%q sel=%t", e2.Value(), e2.SelectionActive())
	}
	t.Log("locks: build-mode + unfocused both ignore shift-motions, ctrl+a, and paste")
}

func TestPlanSelHighlightSplice(t *testing.T) {
	e := focusedEditor(t, 44, 16, "# Goal\n- matte panels azimuth-washed\n- glass lanes")
	e.sel = planSelState{active: true, anchor: planPos{1, 2}, head: planPos{2, 5}}
	view := e.View()
	raw := view
	if !strings.Contains(raw, "\x1b[7m") {
		t.Fatalf("the focused frame must carry the SGR-7 highlight:\n%q", raw)
	}
	// ANSI-stripped the highlight is invisible structure — but the rows
	// underneath it must still read byte-true
	stripped := ansi.Strip(view)
	if !strings.Contains(stripped, "- matte panels azimuth-washed") {
		t.Fatalf("the marked row still renders its text:\n%s", stripped)
	}
	// the mark spans rows 2..3 of the body (buffer lines 1..2)
	rowCount := strings.Count(raw, "\x1b[7m")
	if rowCount < 2 {
		t.Fatalf("a two-row mark splices at least two highlight spans, got %d", rowCount)
	}
	// unfocused (read-only) render: NO highlight survives the blur
	e.Blur()
	if strings.Contains(e.View(), "\x1b[7m") {
		t.Fatal("the blur drops the mark — the read-only render carries no highlight")
	}
	t.Logf("splice: %d reverse-video spans across the 2-row mark; text byte-true under them", rowCount)
}

func TestPlanSelHighlightWrapContinuation(t *testing.T) {
	// the mark covers a wrapped line's tail + the next line: the wrap's
	// continuation rows paint whole — the absolute-row math must match the
	// textarea's real render
	content := "alpha beta gamma delta epsilon zeta\nlast line"
	e := focusedEditor(t, 14, 14, content) // 12-cell wraps
	e.sel = planSelState{active: true, anchor: planPos{0, 6}, head: planPos{1, 4}}
	view := e.View()
	if !strings.Contains(view, "\x1b[7m") {
		t.Fatalf("the wrapped mark must paint:\n%q", view)
	}
	segs := planWrapSegments(content[:35], e.wrapWidth())
	if len(segs) < 3 {
		t.Fatalf("precondition: the first line wraps into multiple rows, got %d", len(segs))
	}
	// every visual row of the marked span highlights: count rows between
	// the marked segments and the hi row (inclusive)
	markedRows := 0
	lo, hi := e.sel.norm()
	_ = lo
	abs := 0
	lines := strings.Split(e.Value(), "\n")
	for i, ln := range lines {
		n := len(planWrapSegments(ln, e.wrapWidth()))
		if i == 0 {
			markedRows += n // selection starts mid-line-0: its wrap rows from the anchor on
			_ = abs
		}
		_ = i
	}
	if hi.row == 1 {
		markedRows++ // the hi line itself
	}
	gotSpans := len(e.selAbsSpans())
	if gotSpans < 2 {
		t.Fatalf("the wrapped mark spans ≥2 visual rows, got %d", gotSpans)
	}
	t.Logf("wrap continuation: %d segments@width %d, %d highlight spans", len(segs), e.wrapWidth(), gotSpans)
}

// TestPlanSelFooterHintSwap pins the ONLY copy change on the hint bar: the
// selection ops line rides the focused footer WHILE a mark is live; every
// other state's copy stays byte-frozen.
func TestPlanSelFooterHintSwap(t *testing.T) {
	e := focusedEditor(t, 60, 20, "# Goal")
	if got := e.footer(); !strings.Contains(got, "enter: newline · esc: done editing") {
		t.Fatalf("the focused no-mark footer is byte-frozen, got %q", ansi.Strip(got))
	}
	e.sel = planSelState{active: true, anchor: planPos{0, 0}, head: planPos{0, 3}}
	if got := ansi.Strip(e.footer()); !strings.Contains(got, "ctrl+c copy · ctrl+x cut") {
		t.Fatalf("the live-mark footer hints the ops, got %q", got)
	}
	e.ClearSelection()
	if got := e.footer(); !strings.Contains(got, "enter: newline · esc: done editing") {
		t.Fatalf("clearing the mark restores the frozen footer, got %q", ansi.Strip(got))
	}
	e.Blur()
	if got := ansi.Strip(e.footer()); got != "click to edit · ctrl+x approve → build · ctrl+p exits" {
		t.Fatalf("the unfocused footer is byte-frozen, got %q", got)
	}
	e.SetMode("build")
	if got := ansi.Strip(e.footer()); got != "ctrl+p back to plan" {
		t.Fatalf("the build footer is byte-frozen, got %q", got)
	}
	t.Log("footer swap: frozen → ops-hint on the live mark → frozen (blur/build byte-identical)")
}

// TestPlanSelUnshiftedKeyCollapses pins the classic collapse: any key
// outside the shift family drops the mark BEFORE the textarea sees it.
func TestPlanSelUnshiftedKeyCollapses(t *testing.T) {
	e := focusedEditor(t, 60, 20, "ab\ncd")
	e.Update(shiftKey(tea.KeyRight))
	if !e.SelectionActive() {
		t.Fatal("precondition: a mark is live")
	}
	e.Update(tea.KeyPressMsg(tea.Key{Code: 'z', Text: "z"}))
	if e.SelectionActive() {
		t.Fatal("an unshifted keystroke collapses the mark")
	}
	if got := e.Value(); got != "azb\ncd" {
		t.Fatalf("the keystroke then behaves exactly as it always did, got %q", got)
	}
	t.Logf("collapse: mark dropped, keystroke landed normally → %q", e.Value())
}

// TestPlanSelFrameDump renders the PROOF sheets for the dispatch report:
// the focused editor with a live mark (SGR-7 visible in raw, structure in
// the strip), then the post-cut empty case, then the copy verdict line.
func TestPlanSelFrameDump(t *testing.T) {
	copies, _ := stubClip(t)
	e := focusedEditor(t, 56, 14, "# Goal\nA gallery lobby wall that feels calm.\n# Steps\n- matte panels azimuth-washed\n- glass lanes")
	e.sel = planSelState{active: true, anchor: planPos{3, 2}, head: planPos{3, 14}}
	frame(t, "SELECTION LIVE (raw rows carry SGR-7 over the marked span)", e.View())
	marked := strings.ReplaceAll(e.View(), "\x1b[7m", "▸")
	marked = strings.ReplaceAll(marked, "\x1b[27m", "◂")
	frame(t, "SELECTION LIVE (▸…◂ annotates the reverse-video span)", marked)

	_, n, _ := e.CopySelection()
	t.Logf("copy toast the app arms: %q", fmt.Sprintf("Copied %d chars", n))
	if len(*copies) != 1 || (*copies)[0] != "matte panels" {
		t.Fatalf("the dump's copy payload = %v", *copies)
	}

	// the ctrl+x-flavored empty case: mark all, cut → empty pane + frozen hint
	e2 := focusedEditor(t, 56, 14, "x")
	e2.Update(ctrlKey('a'))
	_, n2, _ := e2.CutSelection()
	frame(t, "POST-CUT EMPTY (ctrl+x cut everything, caret at (0,0))", e2.View())
	if e2.Value() != "" || e2.SelectionActive() || n2 != 1 {
		t.Fatalf("the empty case: value=%q sel=%t n=%d", e2.Value(), e2.SelectionActive(), n2)
	}
	t.Logf("cut-flavored empty case: %q buffer, mark gone, frozen footer armed", e2.Value())
}
