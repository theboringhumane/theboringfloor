// chat_selection.go — WEBPAGE-STYLE MOUSE TEXT SELECTION over the chat
// transcript (the app-side twin is internal/app/selection.go — its header
// comment is the design doc; press arms, drag extends, a dragged release
// claims the text for OSC52 copy, a motionless release replays the
// original press through the legacy click path).
//
// THIS file owns the panel half:
//
//	SelectionBegin(cx, cy) bool    — press at a CHAT-PANEL-LOCAL coord
//	SelectionDrag(cx, cy)          — armed drag motion (head follows)
//	SelectionFinish(cx, cy) (s, n) — release: extract + finalize
//	SelectionActive() bool         — esc-visibility contract for the app
//	ClearSelection()               — wipe state + repaint
//
// COORDINATES are the chat panel's own: (0,0) is the FIRST VISIBLE ROW of
// the transcript viewport (the app translates screen → panel exactly like
// ClickRow — chat.go:~634-657). Rows resolve through the viewport's scroll
// offset into CONTENT-LINE space (row = cy + vp.YOffset()), the same seam
// ClickRow and the click hit-maps (threadRows / userFoldRows) already use;
// because endpoints live in content-line space, scrolling MID-DRAG keeps
// the selection pinned to the words, not to the screen. Columns are cells
// of the POSTED viewport lines — i.e. INCLUDING the chatPadL left gutter
// every transcript row rides (setConversation): cx 0/1 land in the gutter
// pad, text starts at cell chatPadL. Extraction strips the pad (clamps
// each row's span into [chatPadL, line end)) so copied text never carries
// the gutter; the RENDER highlight still paints it (a full-row drag reads
// as a full-row bar, like a webpage).
//
// The selection is RENDER-LEVEL only: it indexes lines as POSTED to the
// viewport (folded user bubbles select the VISIBLE hint row's text,
// thread rows select their rendered titles — the fold/expansion is a
// render shape, and so is the selection). It owns NO state beyond the
// endpoints + the SEEN line cache (selLines captures the padded lines at
// every setConversation call — the overlay's row space; extraction reads
// the same cache and ansi.Strip's each row, so the live overlay never
// leaks into the copy).
//
// The highlight is a POST-PROCESS overlay on the viewport content inside
// setConversation — the ONE seam every render path (SetState, toggles,
// scroll-agnostic content, fold/thread shapes) flows through — spliced
// ANSI-aware per line (mirroring permSplice): reverse-video over the
// selected cell ranges. forceRender() rides every mutation because the
// endpoints change pixels, not state (the SetState revision gate can't
// see them).
//
// Extraction: the selected PLAIN text, ansi.Strip'd per row, rows joined
// with "\n" (blank interior rows survive — webpage semantics), edge
// spans clamped to [chatPadL, line len), direction NORMALIZED (dragging
// up/left yields the same text as down/right), each row's span trimmed
// right of trailing whitespace. n = len([]rune(text)) — the RUNE count
// INCLUDING the joining newlines, literally what "Copied N chars" says
// (chars, not cells — the same counting the toast's frozen copy implies).
// A selection whose rows contribute NO text (pad-only, blanks only)
// returns n == 0 (the app treats it as "the drag decided nothing").
package panels

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// selFullReset matches BOTH SGR full-reset spellings — "\x1b[0m" (explicit
// zero) and lipgloss's "\x1b[m" (empty param): the terminal reads them
// identically ("reset ALL attributes", the reverse bit included), so a
// selection span must re-arm after either.
var selFullReset = regexp.MustCompile("\x1b\\[0?m")

// selState — ONE live selection: the anchor (press cell) and head (current
// drag cell) in CONTENT-LINE space (0-based into the padded lines cache;
// cells include the chatPadL gutter). active while the button is held
// (fate undecided), finalized after a dragged release (the highlight
// persists until esc / a plain click / a fresh arm — webpage rule).
type selState struct {
	active    bool
	finalized bool
	aRow      int
	aCol      int
	hRow      int
	hCol      int
}

// selRevOn / selRevOff — the reverse-video highlight. Plain SGR 7, re-armed
// after every internal full-reset so a styled transcript chunk's "\x1b[0m"
// can't kill the highlight mid-span (the splicer re-emits it, mirroring how
// permSplice keeps ANSI content intact across a splice).
const (
	selRevOn  = "\x1b[7m"
	selRevOff = "\x1b[27m"
)

// selRow maps a panel-local y into a CONTENT-LINE row — the same geometry
// ClickRow applies (viewport row 0 at the panel top, scroll offset added),
// clamped to the posted lines so out-of-region drags snap to the nearest
// transcript edge.
func (c *Chat) selRow(cy int) int {
	row := cy + c.vp.YOffset()
	if row < 0 {
		return 0
	}
	if row >= len(c.selLines) {
		return len(c.selLines) - 1 // n>0 guaranteed: renderConversation always emits ≥1 line
	}
	return row
}

// selCol maps a panel-local x into a cell column of the padded lines,
// clamped to the panel so out-of-panel drags snap to the edge.
func (c *Chat) selCol(cx int) int {
	if cx < 0 {
		return 0
	}
	if cx >= c.w {
		return c.w - 1
	}
	return cx
}

// selNorm returns the selection's endpoints in READING ORDER (top-left
// first) — a drag up/left selects exactly what its down/right twin does.
func (c *Chat) selNorm() (loRow, loCol, hiRow, hiCol int) {
	a, h := c.sel, c.sel
	if h.hRow < a.aRow || (h.hRow == a.aRow && h.hCol < a.aCol) {
		return h.hRow, h.hCol, a.aRow, a.aCol
	}
	return a.aRow, a.aCol, h.hRow, h.hCol
}

// selSpans computes the PER-LINE cell spans [from, to) of the selection
// over the padded lines: the start row runs from loCol to the line end,
// interior rows are whole, the end row runs from 0 to hiCol+1 (the cell
// under the release cursor is INCLUDED). A single-row selection is the
// plain [loCol, hiCol+1) window. out spans may exceed line lengths —
// callers clamp per line (lines end short; only the pad fills).
func (c *Chat) selSpans() map[int][2]int {
	loRow, loCol, hiRow, hiCol := c.selNorm()
	spans := map[int][2]int{}
	switch {
	case loRow == hiRow:
		spans[loRow] = [2]int{loCol, hiCol + 1}
	case loRow+1 == hiRow:
		spans[loRow] = [2]int{loCol, c.w}
		spans[hiRow] = [2]int{0, hiCol + 1}
	default:
		for r := loRow; r <= hiRow; r++ {
			switch r {
			case loRow:
				spans[r] = [2]int{loCol, c.w}
			case hiRow:
				spans[r] = [2]int{0, hiCol + 1}
			default:
				spans[r] = [2]int{0, c.w}
			}
		}
	}
	return spans
}

// SelectionBegin arms a pending selection at the panel-local cell IF the
// press is over the transcript VIEWPORT region (never the divider / typing
// row / chips / pickers / textarea — those live past c.vp.Height() in
// panel rows) AND no floating card claims the point (an open permission /
// question / session card owns its frame's clicks, same contract as
// ClickRow). On acceptance the anchor+head pin at the resolved content
// cell and forceRender shows the first (one-cell) highlight — the press's
// fate stays undecided until the release.
func (c *Chat) SelectionBegin(cx, cy int) bool {
	if c.cardClaims(cx, cy) {
		return false // the floating card owns this point (ClickRow's twin rule)
	}
	if cy < 0 || cy >= c.vp.Height() {
		return false // divider / typing row / chips / pickers / textarea
	}
	if len(c.selLines) == 0 {
		return false
	}
	row, col := c.selRow(cy), c.selCol(cx)
	c.sel = selState{active: true, aRow: row, aCol: col, hRow: row, hCol: col}
	c.forceRender()
	return true
}

// SelectionDrag extends the armed selection's head to the panel-local cell
// (clamped into content bounds; a no-op while no drag is armed — the app
// only calls it armed, but the panel keeps its own contract airtight) and
// repaints ONLY when the cell actually moved (motion storms are cheap:
// one compare per event).
func (c *Chat) SelectionDrag(cx, cy int) {
	if !c.sel.active {
		return
	}
	row, col := c.selRow(cy), c.selCol(cx)
	if row == c.sel.hRow && col == c.sel.hCol {
		return
	}
	c.sel.hRow, c.sel.hCol = row, col
	c.forceRender()
}

// SelectionFinish settles the drag: extends the head to the release cell,
// extracts the selected PLAIN text across the posted lines (see the file
// header for the pad/clamp/join/counting rules), marks the selection
// FINALIZED (highlight persists) and repaints. Returns ("", 0) for an
// unset armed flag or a selection whose spans carry no text — the app
// treats n == 0 as "the drag decided nothing" (clear, no toast).
func (c *Chat) SelectionFinish(cx, cy int) (text string, n int) {
	if !c.sel.active {
		return "", 0
	}
	c.sel.hRow, c.sel.hCol = c.selRow(cy), c.selCol(cx)
	c.sel.active = false
	c.sel.finalized = true
	text = c.selText()
	c.forceRender()
	if text == "" {
		return "", 0
	}
	// the frozen count: runes INCLUDING the joining newlines — the toast
	// reads "Copied N chars", so N counts chars, and a char is a rune.
	return text, len([]rune(text))
}

// selText extracts the selected text from the CACHED padded lines
// (pre-highlight — the overlay never pollutes the copy): per line the span
// is clamped into [chatPadL, line cells) (the gutter pad never leaks into
// the copy), ansi.Strip'd, trimmed right, and rows are joined with "\n"
// (interior blank rows survive as empty segments — webpage semantics).
func (c *Chat) selText() string {
	spans := c.selSpans()
	loRow, _, hiRow, _ := c.selNorm()
	parts := make([]string, 0, hiRow-loRow+1)
	for r := loRow; r <= hiRow && r < len(c.selLines); r++ {
		span, ok := spans[r]
		if !ok || r < 0 {
			continue
		}
		plain := ansi.Strip(c.selLines[r])
		lineW := ansi.StringWidth(plain)
		from, to := span[0], span[1]
		if from < chatPadL {
			from = chatPadL
		}
		if to > lineW {
			to = lineW
		}
		if from >= to {
			parts = append(parts, "") // blank/pad-only row: keeps its newline
			continue
		}
		parts = append(parts, strings.TrimRight(ansi.Cut(plain, from, to), " "))
	}
	return strings.Join(parts, "\n")
}

// selOverlay splices the reverse-video highlight over the POSTED padded
// lines IN PLACE (render-level: content rows shift freely, the overlay is
// recomputed on every setConversation — folds, threads, scroll all survive
// because the endpoints live in the same content-line space). Mirrors
// permSplice: cut the line ANSI-aware at cell edges, wrap the mid span in
// the reverse attribute, keep head and tail byte-identical.
func (c *Chat) selOverlay(lines []string) {
	for r, span := range c.selSpans() {
		if r < 0 || r >= len(lines) {
			continue
		}
		lines[r] = selHighlight(lines[r], span[0], span[1])
	}
}

// selHighlight returns line with cells [from, to) reverse-video-spliced
// (clamped to the line's own width out of the caller's viewport-wide
// spans): head | reversed mid | tail, cut ANSI-aware like permSplice.
func selHighlight(line string, from, to int) string {
	w := ansi.StringWidth(line)
	if from < 0 {
		from = 0
	}
	if to > w {
		to = w
	}
	if from >= to {
		return line
	}
	head := ansi.Cut(line, 0, from)
	mid := ansi.Cut(line, from, to)
	tail := ansi.Cut(line, to, w)
	// re-arm reverse after every internal full reset — styled transcript
	// chunks (glamour runs, tinted prefs) terminate in \x1b[0m, and lipgloss
	// chunks terminate in \x1b[m (SGR's empty-param twin of 0); BOTH reset
	// ALL attributes at the terminal, the reverse bit included.
	mid = selRevOn + selFullReset.ReplaceAllString(mid, "$0"+selRevOn) + selRevOff
	return head + mid + tail
}

// SelectionActive reports whether a selection highlight is VISIBLE (armed
// or finalized) — the app's esc-visibility contract: while true, esc
// belongs to the selection (clear first), never to the double-esc stop.
func (c *Chat) SelectionActive() bool { return c.sel.active || c.sel.finalized }

// ClearSelection wipes the selection state and repaints. A no-op when no
// selection is up (the app's press fall-through calls it on EVERY
// off-selection click — the repaint must not ride those for free).
func (c *Chat) ClearSelection() {
	if !c.SelectionActive() {
		return
	}
	c.sel = selState{}
	c.forceRender()
}
