// chat_selection_test.go — behavior proofs for the WEBPAGE-STYLE MOUSE
// TEXT SELECTION over the chat transcript (chat_selection.go):
//
//	(a) begin region: a press inside the transcript viewport arms (anchor
//	    pins at the resolved content cell); presses on the divider/typing/
//	    textarea chrome rows, negative y, an empty transcript, or a point
//	    an open FLOATING CARD claims are all rejected (active stays false);
//	(b) a downward 3-row drag extracts the EXACT plain text — rows joined
//	    "\n", ansi.Strip'd, pad-clamped, right-trimmed, the 6-cell hanging
//	    indent of continuation rows SURVIVING the copy;
//	(c) an upward drag over the same span yields the identical text+n
//	    (selNorm direction normalization);
//	(d) the char-count: n = len([]rune(text)) — RUNES INCLUDING the
//	    joining "\n"s (the toast's "Copied N chars" counts chars);
//	(e) the highlight RENDERING: selRevOn (SGR 7) / selRevOff (SGR 27)
//	    reverse-video spans in View while armed/selected, gone after
//	    ClearSelection; render-level only — ansi.Strip(View) is identical
//	    with or without a selection;
//	(f) the re-arm rule: internal full resets ("\x1b[0m") inside a span
//	    re-arm SGR 7; the lipgloss "\x1b[m" flavor is NOT re-armed
//	    (asserted as IMPLEMENTED — see the divergence note);
//	(g) highlight persistence across a SetState rebuild (the overlay
//	    re-splices on the same content-line coords; extraction stays clean);
//	(h) scrolled selection: row = cy + vp.YOffset(); scrolling MID-DRAG
//	    keeps the selection pinned to the words, not the screen;
//	(i) SelectionActive's truth table idle → armed → dragging →
//	    finalized-highlight → cleared, plus the idle no-ops;
//	(j) the chatPadL gutter: cx in the 2-col pad ARMS (the region gate is
//	    the viewport y-range, columns clamp) but the pad never enters the
//	    COPY (spans clamp into [chatPadL, line end)); the RENDER paints it;
//	(k) zero-text finish: begin then finish covering only blank/pad cells
//	    ⇒ ("", 0) — the app's "the drag decided nothing" verdict.
//
// No clocks, no fetches: every transcript is seeded via SetState with
// literal user turns (plain-wrapped, deterministic), the same constructors
// as chat_fold_test.go. SetSize(44, 24) → c.w=44, vpH = 24−textareaH−1 =
// 20, contentW=40, mdWidth=32 — far from every wrap/fold boundary (the 3
// body rows of the seeded turn stay verbatim and ≤userFoldVisible).
package panels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	state "github.com/theboringhumane/theboringoffice/internal/state"
)

// selChat builds the canonical selection fixture: ONE user turn of the
// three body rows "alpha" / "bravo" / "charlie". After the chatPadL pad
// (setConversation) and ansi.Strip the posted transcript is EXACTLY:
//
//	selLines[0] = "  you › alpha   "   (styled "you › " prefix, 13 cells)
//	selLines[1] = "        bravo    "   (2 pad + 6 hanging indent, 13 cells)
//	selLines[2] = "        charlie  "   (15 cells)
func selChat(t *testing.T) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "alpha\nbravo\ncharlie", At: 10},
	}})
	// the fixture's own shape must hold before any selection math trusts it
	if want := []string{"  you › alpha", "        bravo", "        charlie"}; len(c.selLines) != len(want) {
		t.Fatalf("fixture transcript must post %d padded lines, got %d", len(want), len(c.selLines))
	} else {
		for i, w := range want {
			if got := ansi.Strip(c.selLines[i]); got != w {
				t.Fatalf("fixture selLines[%d] = %q, want %q", i, got, w)
			}
		}
	}
	return c
}

// selRowsSplit splits View into its raw (unstripped) rows.
func selRowsSplit(view string) []string { return strings.Split(view, "\n") }

// TestChatSelectionBeginRegion is proof (a): the region gate is the
// transcript VIEWPORT's y-range (no floating card claiming the point) —
// columns are free (they clamp into [0, w-1]), y is not.
func TestChatSelectionBeginRegion(t *testing.T) {
	// (a-1) a press inside the transcript viewport arms and pins the cell.
	c := selChat(t)
	if !c.SelectionBegin(2, 0) {
		t.Fatal("a press inside the transcript viewport must arm the selection")
	}
	if c.sel != (selState{active: true, aRow: 0, aCol: 2, hRow: 0, hCol: 2}) {
		t.Fatalf("begin must pin anchor+head at the resolved content cell, got %+v", c.sel)
	}
	if !c.SelectionActive() {
		t.Fatal("an armed selection must report SelectionActive")
	}

	// (a-2) chrome rows past the viewport bottom — divider (cy == vpH),
	// typing row, textarea rows — can never arm, nor can a negative y.
	for _, cy := range []int{c.vp.Height(), c.vp.Height() + 1, c.h - 1, -1} {
		cy := cy
		c2 := selChat(t)
		if c2.SelectionBegin(2, cy) {
			t.Fatalf("a press on chrome row cy=%d (vpH=%d) must NOT arm", cy, c.vp.Height())
		}
		if c2.SelectionActive() {
			t.Fatalf("a rejected begin (cy=%d) must leave SelectionActive false", cy)
		}
	}

	// (a-3) a press BEFORE any SetState posts no lines and is rejected.
	empty := NewChat(nil)
	empty.SetSize(44, 24)
	if empty.SelectionBegin(2, 0) || empty.SelectionActive() {
		t.Fatal("an empty transcript (len(selLines)==0) must reject the arm")
	}

	// (a-4) x clamps, never rejects: a negative cx lands on the gutter's
	// first pad cell (selCol(−1) → 0) — the y-range alone gates the press.
	c4 := selChat(t)
	if !c4.SelectionBegin(-1, 0) || c4.sel.aCol != 0 {
		t.Fatalf("cx out of range must clamp (to 0), not reject: begin=%v anchor=%+v",
			c4.SelectionBegin(-1, 0), c4.sel)
	}

	// (a-5) a press on a blank viewport row BELOW the posted transcript
	// still arms (it lives inside the viewport): selRow snaps to the last
	// content row — an out-of-range press clamps to the transcript edge.
	c5 := selChat(t)
	if !c5.SelectionBegin(5, 10) || c5.sel.aRow != 2 || c5.sel.hRow != 2 {
		t.Fatalf("a press below content (3 rows in a 20-row vp) must arm at the clamped last row: begin=%v sel=%+v",
			false, c5.sel)
	}
}

// TestChatSelectionBeginRespectsClaimedCard is proof (a, card half): an
// open floating card owns its frame's points even though they sit inside
// the viewport y-range — ClickRow's twin rule (cardClaims).
func TestChatSelectionBeginRespectsClaimedCard(t *testing.T) {
	c := selChat(t)
	c.perm = &PermissionView{ID: "p1", ToolName: "Write", Summary: "/tmp/x", Agent: "boss", Index: 1, Total: 1}
	top, left, cardW, rows, _ := c.permCardGeom()
	if !c.cardClaims(left+2, top+1) {
		t.Fatal("fixture: the point must sit inside the permission card's frame")
	}
	if c.SelectionBegin(left+2, top+1) {
		t.Fatal("a press inside an open permission card must NOT arm the selection (the card claims it)")
	}
	if c.SelectionActive() {
		t.Fatal("a card-claimed press must leave SelectionActive false")
	}
	// the same panel, a point OUTSIDE the frame — row 0 sits ABOVE the
	// card's top edge yet squarely inside its x-range and inside the
	// viewport: region gate passes, no claim — arms as before.
	if top <= 0 || cardW < 4 || len(rows) == 0 || left+2 >= left+cardW {
		t.Fatalf("fixture geometry unexpected: top=%d left=%d cardW=%d rows=%d", top, left, cardW, len(rows))
	}
	if !c.SelectionBegin(left+2, 0) {
		t.Fatal("a press outside the card frame (inside the viewport) must arm")
	}
}

// TestChatSelectionDownwardDragExtracts is proof (b): the exact plain text
// of a downward 3-row drag — "\n" joins, hanging-indents preserved, zero
// ANSI bytes — plus the evidence frame for the span + its extraction.
func TestChatSelectionDownwardDragExtracts(t *testing.T) {
	c := selChat(t)
	if !c.SelectionBegin(chatPadL, 0) { // anchor at row0's first text cell
		t.Fatal("begin on row 0 must arm")
	}
	c.SelectionDrag(6, 1) // head walks down through the middle row
	text, n := c.SelectionFinish(14, 2)
	// the spans: row0 [2, 44) → clamp to [2, 13) → "you › alpha"
	//            row1 [0, 44) → clamp to [2, 13) → "      bravo"  (indent kept)
	//            row2 [0, 15) → clamp to [2, 15) → "      charlie"
	want := "you › alpha\n      bravo\n      charlie"
	if text != want {
		t.Fatalf("extracted text:\n got %q\nwant %q", text, want)
	}
	if strings.Contains(text, "\x1b") {
		t.Fatalf("the copy must carry no ANSI bytes: %q", text)
	}
	if n != 37 { // (d) 11 + 11 + 13 runes + 2 joining "\n"
		t.Fatalf("n must freeze the rune count incl. newlines (11+11+13+2=37), got %d", n)
	}
	if n != len([]rune(text)) {
		t.Fatalf("n (%d) must equal len([]rune(text)) (%d)", n, len([]rune(text)))
	}
	// PROOF frame: the finalized highlight + its extraction, eyeballed.
	t.Logf("---- FINALIZED 3-row selection (44 cols, raw View rows 0-2 — SGR7…SGR27 spans) ----")
	for i, row := range selRowsSplit(c.View())[:3] {
		t.Logf("V[%d]=%q", i, row)
	}
	t.Logf("---- EXTRACTED COPY (plain text, n=%d chars incl. 2 newlines) ----\n%s\n----", n, text)
}

// TestChatSelectionUpwardDragNormalized is proof (c): dragging UP over the
// exact same span extracts byte-identical text and n (selNorm reads the
// endpoints top-left-first, whichever way the mouse ran).
func TestChatSelectionUpwardDragNormalized(t *testing.T) {
	c := selChat(t)
	if !c.SelectionBegin(14, 2) { // the downward test's RELEASE cell becomes the press
		t.Fatal("begin on row 2 must arm")
	}
	c.SelectionDrag(6, 1)
	text, n := c.SelectionFinish(chatPadL, 0) // …and its press cell becomes the release
	if want := "you › alpha\n      bravo\n      charlie"; text != want {
		t.Fatalf("an upward drag must extract its downward twin's text:\n got %q\nwant %q", text, want)
	}
	if n != 37 {
		t.Fatalf("upward n must equal downward n (37), got %d", n)
	}
}

// TestChatSelectionRuneCount is proof (d) stripped to the counting rule
// itself: n is the RUNE length of the extracted text INCLUDING the "\n"
// joins (literally what "Copied N chars" toasts), so a single-row span has
// no newline boost and a partial-cell span counts partial cells. The "›"
// glyph counts as ONE rune (chars, not bytes).
func TestChatSelectionRuneCount(t *testing.T) {
	c := selChat(t)
	c.SelectionBegin(chatPadL, 0)
	text, n := c.SelectionFinish(43, 0) // c.w−1: span [2, 44) clamps to the line end
	if text != "you › alpha" || n != 11 { // 3+1+1+1+5 runes, NO newline
		t.Fatalf("single full row must extract 11 runes newline-free: %q n=%d", text, n)
	}
	c.ClearSelection()
	c.SelectionBegin(chatPadL, 0)
	text, n = c.SelectionFinish(10, 0) // span [2, 11): 9 cells — the last-grapheme edge
	if text != "you › alp" || n != 9 {
		t.Fatalf("a partial single-row span must count exactly its cells: %q n=%d", text, n)
	}
	c.ClearSelection()
	// the multi-row rule one more time, terse: 11 + 11 + 13 + 2 joins.
	c.SelectionBegin(chatPadL, 0)
	_, n = c.SelectionFinish(14, 2)
	if n != 37 {
		t.Fatalf("the 3-row span must count runes INCLUDING both newlines (11+11+13+2=37), got %d", n)
	}
}

// TestChatSelectionHighlightPixels is proof (e): while armed and while
// finalized the View carries the reverse-video markers (selRevOn → SGR 7,
// selRevOff → SGR 27) around the spanned cells — with an EXACT byte pin on
// the unstyled middle row — and after ClearSelection no View row carries
// SGR 7 again. Render-LEVEL only: ansi.Strip(View) is selection-invariant.
func TestChatSelectionHighlightPixels(t *testing.T) {
	c := selChat(t)
	strippedIdle := ansi.Strip(c.View())
	if strings.Contains(c.View(), selRevOn) {
		t.Fatal("precondition: no reverse-video anywhere without a selection")
	}

	// armed, one cell: the 1-cell span on row0 (cell 8 = 'a' of "alpha")
	// renders EXACTLY wrapped — 'a' sits flush against SGR 27.
	c.SelectionBegin(8, 0)
	row0 := selRowsSplit(c.View())[0]
	if !strings.Contains(row0, selRevOn) || !strings.Contains(row0, "a"+selRevOff) {
		t.Fatalf("the armed 1-cell span must render SGR7-wrapped on the arm cell: %q", row0)
	}
	if ansi.Strip(c.View()) != strippedIdle {
		t.Fatal("the highlight must never change the transcript's readable text (ansi.Strip invariant)")
	}

	// finalized over all three rows: the interior (unstyled) row1 pins
	// byte-exact — the full content line inside one SGR7…SGR27 pair.
	c.SelectionDrag(6, 1)
	c.SelectionFinish(14, 2)
	rows := selRowsSplit(c.View())
	if want := selRevOn + "        bravo" + selRevOff; !strings.Contains(rows[1], want) {
		t.Fatalf("the interior row of the span must render the exact reverse wrap %q: %q", want, rows[1])
	}
	if !strings.Contains(rows[0], selRevOn) || !strings.Contains(rows[2], selRevOn) {
		t.Fatalf("the edge rows of the span must carry the highlight too:\nV0=%q\nV2=%q", rows[0], rows[2])
	}
	if ansi.Strip(c.View()) != strippedIdle {
		t.Fatal("a finalized highlight must still leave ansi.Strip(View) untouched")
	}

	// cleared: the markers leave every row; the text never moved.
	c.ClearSelection()
	view := c.View()
	for i, row := range selRowsSplit(view) {
		if strings.Contains(row, selRevOn) || strings.Contains(row, selRevOff) {
			t.Fatalf("ClearSelection must strip the highlight from EVERY row (row %d): %q", i, row)
		}
	}
	if ansi.Strip(view) != strippedIdle {
		t.Fatal("ClearSelection must return the View to its pre-selection text exactly")
	}
}

// TestChatSelectionHighlightResetReArm is proof (f): the splicer re-arms
// SGR 7 after every internal "\x1b[0m" full reset inside a span (the
// documented intent — glamour chunks terminate this way) but NOT after the
// bare "\x1b[m" lipgloss emits (chrome.Fg & friends), so a span crossing
// the styled "you › " prefix renders the cells PAST the reset without
// reverse-video. The test pins the behavior AS IMPLEMENTED.
func TestChatSelectionHighlightResetReArm(t *testing.T) {
	// the unit surface — plain text wraps head|reversed mid|tail:
	if got, want := selHighlight("  abcdef", 2, 5), "  \x1b[7mabc\x1b[27mdef"; got != want {
		t.Fatalf("plain splice:\n got %q\nwant %q", got, want)
	}
	// an internal "\x1b[0m" is re-armed (the documented rule):
	if got, want := selHighlight("ab\x1b[0mcd", 0, 4), "\x1b[7mab\x1b[0m\x1b[7mcd\x1b[27m"; got != want {
		t.Fatalf("the internal full reset must be re-armed with SGR7:\n got %q\nwant %q", got, want)
	}
	// out-of-line spans clamp to the line; empty spans pass through:
	if got, want := selHighlight("ab", 0, 99), "\x1b[7mab\x1b[27m"; got != want {
		t.Fatalf("to>width must clamp to the line's own width:\n got %q\nwant %q", got, want)
	}
	if got := selHighlight("ab", 1, 1); got != "ab" {
		t.Fatalf("an empty span must pass the line through unchanged, got %q", got)
	}

	// the integration surface — over the real transcript the styled user
	// prefix terminates in lipgloss's "\x1b[m" (SGR default param, the
	// terminal's exact equivalent of "\x1b[0m"): the re-arm matcher
	// (selFullReset) covers BOTH spellings, so a span crossing the prefix
	// resets and re-arms — row0 carries TWO SGR7s (the wrap + the re-arm)
	// and the whole span paints reversed.
	c := selChat(t)
	c.SelectionBegin(chatPadL, 0)
	text, n := c.SelectionFinish(43, 0) // the full row0: prefix (styled) + "alpha" (plain)
	row0 := selRowsSplit(c.View())[0]
	if count := strings.Count(row0, selRevOn); count != 2 {
		t.Fatalf("a span crossing lipgloss's \x1b[m reset must re-arm once (wrap + re-arm = 2 SGR7s): %q", row0)
	}
	// the extraction is unaffected — the copy carries the whole row either way.
	if text != "you › alpha" || n != 11 {
		t.Fatalf("the reset gap must never leak into the copy: %q n=%d", text, n)
	}
}

// TestChatSelectionSetStateRefresh is proof (g): a FINALIZED highlight
// survives a SetState rebuild on its exact cells (the overlay re-splices
// over the freshly posted lines — endpoints live in content-line space),
// and the extraction source stays clean: a fresh drag over the same cells
// yields the identical text afterwards.
func TestChatSelectionSetStateRefresh(t *testing.T) {
	c := selChat(t)
	c.SelectionBegin(chatPadL, 0)
	c.SelectionDrag(6, 1)
	text, n := c.SelectionFinish(14, 2)
	if text == "" || n != 37 {
		t.Fatalf("fixture selection must land (n=37), got %q n=%d", text, n)
	}
	row0Before := selRowsSplit(c.View())[0]
	if !c.SelectionActive() {
		t.Fatal("a dragged release must leave the finalized highlight active")
	}

	// a REAL rebuild: an appended turn below bumps the revision — the
	// conversation re-renders, setConversation re-posts the lines, the
	// overlay re-splices; rows 0-2 (above the append) must not move.
	c.SetState(state.OfficeState{Tick: 2, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "alpha\nbravo\ncharlie", At: 10},
		{ID: "u2", From: "user", Kind: "user", Text: "delta", At: 20},
	}})
	row0After := selRowsSplit(c.View())[0]
	if row0After != row0Before {
		t.Fatalf("the finalized highlight must land on the SAME cells after a rebuild:\nbefore %q\nafter  %q", row0Before, row0After)
	}
	if !strings.Contains(c.View(), selRevOn) {
		t.Fatal("the rebuild must re-arm the highlight (setConversation re-splices the overlay)")
	}
	if !c.SelectionActive() {
		t.Fatal("a SetState must not wake the selection state")
	}
	// the extraction source stays clean across the rebuild: an identical
	// fresh drag pulls the identical bytes (no overlay pollution).
	c.ClearSelection()
	c.SelectionBegin(chatPadL, 0)
	c.SelectionDrag(6, 1)
	text2, n2 := c.SelectionFinish(14, 2)
	if text2 != text || n2 != n {
		t.Fatalf("post-rebuild extraction must equal the pre-rebuild copy:\n got %q n=%d\nwant %q n=%d", text2, n2, text, n)
	}
}

// TestChatSelectionScrolled is proof (h): rows resolve through
// cy + vp.YOffset() into content-line space, and scrolling MID-DRAG keeps
// the selection pinned to the words — the endpoints never re-anchor to
// screen rows.
func TestChatSelectionScrolled(t *testing.T) {
	// 12 one-row turns → 23 padded lines (turns on even rows, blank
	// separators between) in a 10-row viewport: genuinely scrollable.
	c := NewChat(nil)
	c.SetSize(50, 14) // vpH = 14 − textareaH(3) − divider(1) = 10
	msgs := make([]state.ChatMsg, 0, 12)
	for i := 1; i <= 12; i++ {
		msgs = append(msgs, state.ChatMsg{
			ID: fmt.Sprintf("u%d", i), From: "user", Kind: "user",
			Text: fmt.Sprintf("word%02d", i), At: int64(i)})
	}
	c.SetState(state.OfficeState{Tick: 1, Chat: msgs})
	if len(c.selLines) != 23 || c.vp.YOffset() != 13 || !c.follow {
		t.Fatalf("fixture must post 23 lines and follow the bottom (yoff=13): lines=%d yoff=%d follow=%v",
			len(c.selLines), c.vp.YOffset(), c.follow)
	}
	// a scrolled-back reader: the follow releases and the offset holds
	// through the selection's repaints (forceRender honors follow=false).
	c.follow = false
	c.vp.SetYOffset(4)
	if !c.SelectionBegin(chatPadL, 0) { // panel row0 ↔ content row 4 = "you › word03"
		t.Fatal("begin at a scrolled offset must arm (the viewport region is scroll-independent)")
	}
	if c.sel.aRow != 4 {
		t.Fatalf("row must resolve through YOffset (0+4=4 -> word03's row), got %d", c.sel.aRow)
	}
	// scroll again MID-DRAG: the anchor stays pinned to word03's row 4.
	c.vp.SetYOffset(6)
	c.SelectionDrag(chatPadL, 0) // panel row0 ↔ content row 6 = "you › word04"
	text, n := c.SelectionFinish(15, 0)
	// span rows 4-6: row4 [2, w) → "you › word03"; row5 (blank separator)
	// contributes "" with its newline (webpage semantics); row6 [0, 16) →
	// "you › word04".
	if want := "you › word03\n\nyou › word04"; text != want {
		t.Fatalf("a scrolled drag must extract the PINNED content rows:\n got %q\nwant %q", text, want)
	}
	if n != 26 { // 12 + 0 + 12 runes + 2 joining "\n"
		t.Fatalf("scrolled n must count runes incl. the 2 newlines (12+12+2=26), got %d", n)
	}
	if c.vp.YOffset() != 6 {
		t.Fatalf("selection repaints must not yank a scrolled-back reader (yoff 6, got %d)", c.vp.YOffset())
	}
}

// TestChatSelectionActiveTruthTable is proof (i): the visible-state
// machine idle → armed → dragging → finalized → cleared, and every
// out-of-phase call is a true no-op.
func TestChatSelectionActiveTruthTable(t *testing.T) {
	c := selChat(t)
	idleView := c.View()

	// idle: no active, no finalized; drag/finish/clear are all no-ops
	// (byte-identical View — ClearSelection must not repaint for free).
	if c.SelectionActive() {
		t.Fatal("a fresh panel must not report an active selection")
	}
	c.SelectionDrag(5, 1)
	if text, n := c.SelectionFinish(5, 1); text != "" || n != 0 || c.SelectionActive() {
		t.Fatalf("drag+finish unarmed must no-op, got %q n=%d active=%v", text, n, c.SelectionActive())
	}
	c.ClearSelection()
	if c.View() != idleView {
		t.Fatal("ClearSelection while idle must not touch the pixels at all")
	}

	// armed.
	c.SelectionBegin(chatPadL, 0)
	if !c.SelectionActive() || !c.sel.active || c.sel.finalized {
		t.Fatalf("armed must read active-only (visible): %+v", c.sel)
	}
	// dragging: still armed, head walked.
	c.SelectionDrag(8, 1)
	if !c.SelectionActive() || c.sel.hRow != 1 || c.sel.hCol != 8 {
		t.Fatalf("drag must only move the head: active=%v sel=%+v", c.SelectionActive(), c.sel)
	}
	// a dragged release: finalized — the HIGHLIGHT stays visible.
	if text, n := c.SelectionFinish(10, 1); text == "" || n == 0 {
		t.Fatalf("a non-trivial drag must extract text, got %q n=%d", text, n)
	}
	if !c.SelectionActive() || c.sel.active || !c.sel.finalized {
		t.Fatalf("a dragged release must read finalized-only (highlight persists): %+v", c.sel)
	}
	// cleared back to idle.
	c.ClearSelection()
	if c.SelectionActive() || c.View() != idleView {
		t.Fatal("ClearSelection after finalize must restore the idle state AND pixels")
	}
	// a fresh arm REPLACES a finalized selection (webpage rule): the next
	// press re-arms, wiping the finalized flag.
	c.SelectionBegin(chatPadL, 0)
	c.SelectionFinish(10, 0) // finalize once
	c.SelectionBegin(3, 1)
	if c.sel != (selState{active: true, aRow: 1, aCol: 3, hRow: 1, hCol: 3}) {
		t.Fatalf("a fresh arm must reset to exactly the new press cell: %+v", c.sel)
	}
}

// TestChatSelectionPadGutter is proof (j): the chatPadL 2-cell gutter sits
// INSIDE the viewport region, so a press there ARMS (columns clamp into
// [0, w) — pad cells are legal anchors), but extraction clamps every row's
// span to [chatPadL, line end): the pad never leaks into the copy — while
// the RENDER still paints it (a drag started in the pad reads as a
// full-row bar, like a webpage).
func TestChatSelectionPadGutter(t *testing.T) {
	c := selChat(t)
	if !c.SelectionBegin(1, 0) { // smack in the gutter
		t.Fatal("a press over the pad gutter must arm (the region gate is the y-range; columns clamp)")
	}
	if c.sel.aCol != 1 {
		t.Fatalf("the gutter press anchors at its clamped cell (col 1), got %d", c.sel.aCol)
	}
	// a span from the pad into the text copies TEXT ONLY from chatPadL on:
	text, n := c.SelectionFinish(3, 0) // span [1, 4) → extraction clamps to [2, 4)
	if text != "yo" || n != 2 {
		t.Fatalf("the pad must never leak into the copy (span [1,4) → 'yo', n=2): %q n=%d", text, n)
	}
	// …but the RENDER's span includes the pad cell (paint starts at col 1).
	row0 := selRowsSplit(c.View())[0]
	if !strings.Contains(row0, selRevOn) {
		t.Fatalf("the pad-anchored span must still paint the highlight: %q", row0)
	}
	c.ClearSelection()

	// a pad-ONLY span contributes no text at all.
	c.SelectionBegin(1, 0)
	text, n = c.SelectionFinish(0, 0) // span [0, 2) → clamp [2, 2) → ""
	if text != "" || n != 0 {
		t.Fatalf("a pad-only span must copy nothing: %q n=%d", text, n)
	}
}

// TestChatSelectionZeroTextFinish is proof (k): a release over only
// blank/pad cells extracts ("", 0) — the app's "the drag decided nothing"
// verdict (its handleRelease then calls ClearSelection itself). The panel
// settles FINALIZED-but-empty in between: active=false, finalized=true, a
// no-op overlay — mirrored here to keep the app's contract honest.
func TestChatSelectionZeroTextFinish(t *testing.T) {
	// two turns → a blank separator between them (row3 posts "  " — the pad alone)
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "alpha\nbravo\ncharlie", At: 10},
		{ID: "u2", From: "user", Kind: "user", Text: "delta", At: 20},
	}})
	if got := ansi.Strip(c.selLines[3]); got != "  " {
		t.Fatalf("fixture: row 3 must be the pad-only separator, got %q", got)
	}

	// begin+finish at the SAME cell of the blank row: zero-area, zero text.
	if !c.SelectionBegin(5, 3) {
		t.Fatal("the blank transcript row sits inside the viewport: begin must arm")
	}
	text, n := c.SelectionFinish(5, 3)
	if text != "" || n != 0 {
		t.Fatalf("a zero-cell finish must return (\"\", 0): %q n=%d", text, n)
	}
	// per the implementation: the release settled finalized (the app owns
	// the clear for n==0 — internal/app/selection.go), the overlay's spans
	// clamp to nothing, and NO highlight paints.
	if !c.SelectionActive() || c.sel.active || !c.sel.finalized {
		t.Fatalf("a zero-text finish settles finalized-but-empty (the APP clears it): %+v", c.sel)
	}
	if strings.Contains(c.View(), selRevOn) {
		t.Fatal("a no-text span must paint no highlight (every span clamps empty)")
	}
	// …and the app's follow-through, replayed panel-side: a clear after
	// the verdict unlatches the state completely.
	c.ClearSelection()
	if c.SelectionActive() {
		t.Fatal("after the app's clear, no selection state may linger")
	}
}
