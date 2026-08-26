// plan_editor_selection.go — SELECTION + CUT/COPY/PASTE for the plan
// editor (the chat_selection.go contract, one surface down, in BUFFER space
// so it survives resizes): shift+arrows grow a marked range from the caret,
// ctrl+a marks the whole buffer (the textarea's own ctrl+a=LineStart faction
// is rebound to home-only at construction so the two never double-claim),
// shift+home/end jump by line anchor, ctrl+c copies (ctrl+x cuts — both via
// the app's claim sites), ctrl+v/super+v paste from the OS clipboard and
// cmd+v arrives as tea.PasteMsg — every paste REPLACES the marked range.
// The mouse drag mirrors the transcript's webpage rule: press pins the
// anchor (caret placement), motion extends the head, a dragged release
// finalizes + copies; a motionless press is caret placement ONLY (no copy
// note, no highlight residue).
//
// WHY buffer (row, col) endpoints and not screen cells: the pane resizes
// with every layout pass (mobile ⇄ desktop, window drags); rune-space
// endpoints keep the SAME words marked across a wrap reshaping, and the
// highlight is recomputed from the endpoints at render time. Visual (wrap)
// geometry is derived ONLY for the two seams that need it: mouse mapping
// (screen cell → buffer position) and the SGR-7 highlight splice (buffer
// range → per-visual-row cell spans), both through planWrapSegments — a
// verbatim replica of bubbles v2.1.1 textarea's wrap() (the SAME algorithm,
// with uniseg/mattn width calls swapped for x/ansi's identical-width
// StringWidth) so segment boundaries ALWAYS match the textarea's own
// render. uniseg/mattn stay out of the import list (no go.mod churn).
//
// Clipboard plumbing: COPY writes through clipboardCopyText (clipboard.go's
// stubbed seam — pbcopy/wl-copy/xclip/xsel) and the app rides tea.SetClipboard
// + the statusbar "Copied N chars" toast alongside. PASTE reads through
// clipboardReadText (the read-side twin table, same lazy-tool resolution:
// pbpaste / Get-Clipboard / wl-paste / xclip -o / xsel -o) so the pane never
// leans on bubbles' atotto-backed textarea.Paste — that cmd replies with
// UNEXPORTED msg types the app's routing cannot steer back to the pane
// (today a plan-pane ctrl+v lands in the CHAT draft; this file fixes it by
// owning the whole paste path: key → async read → PlanPasteMsg → pane).
package panels

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// PlanPasteMsg delivers the pane's own async clipboard read back to the
// focused plan editor (ctrl+v / super+v). Exported so the app can route it:
// the app's Update forwards it to the pane ONLY (never the chat draft).
type PlanPasteMsg struct {
	Content string
	Err     error
}

// planPos is a BUFFER-space caret endpoint: a 0-based (line, rune-column)
// pair into the Value() line split. Resize-safe by construction.
type planPos struct {
	row int
	col int
}

// planSelState — ONE live selection on the pane: anchor (press/caret start)
// and head (the moving end). active while ARMED (shift-key/mouse growing)
// and stays active after a dragged release (the highlight persists until
// esc / an unshifted key / Blur, the webpage rule); dragging marks the
// press→release flight (a motionless release finalizes as caret-only).
type planSelState struct {
	active   bool
	dragging bool
	anchor   planPos
	head     planPos
}

// norm returns the selection's endpoints in READING order (top-left first):
// a drag up/left marks exactly what its down/right twin does.
func (s planSelState) norm() (lo, hi planPos) {
	if (s.head.row < s.anchor.row) || (s.head.row == s.anchor.row && s.head.col < s.anchor.col) {
		return s.head, s.anchor
	}
	return s.anchor, s.head
}

// SelectionActive reports whether the pane currently owns a marked range —
// the app's precedence guard for ctrl+x (cut) / ctrl+c (copy) / esc
// (clear-first) while the pane is focused.
func (e *PlanEditor) SelectionActive() bool { return e.sel.active }

// ClearSelection wipes the mark. Blur() and every unshifted key ride this;
// a no-op when nothing is marked (the app's plain-click clear calls it
// unconditionally, same contract as the chat's ClearSelection).
func (e *PlanEditor) ClearSelection() { e.sel = planSelState{} }

// SelectAll marks the WHOLE buffer (ctrl+a): anchor (0,0), head at the
// buffer's end, caret synced to the head so a follow-up shift+left/right
// resumes from there.
func (e *PlanEditor) SelectAll() {
	lines := strings.Split(e.ta.Value(), "\n")
	last := len(lines) - 1
	e.sel = planSelState{
		active: true,
		anchor: planPos{0, 0},
		head:   planPos{row: last, col: len([]rune(lines[last]))},
	}
	e.ta.MoveToEnd()
}

// selEnsureAnchor arms a fresh selection at the CURRENT caret when a
// shift-motion arrives with no mark up yet (classic editors: the caret is
// the anchor, the first motion opens the range).
func (e *PlanEditor) selEnsureAnchor() {
	if e.sel.active {
		return
	}
	p := planPos{row: e.ta.Line(), col: e.ta.Column()}
	e.sel.active = true
	e.sel.anchor = p
	e.sel.head = p
}

// selSyncCursor pins the textarea's real caret onto a buffer position:
// MoveToBegin + a CursorDown walk (VISUAL-row aware, so a wrapped target
// line lands exactly) capped defensively, then the rune column.
func (e *PlanEditor) selSyncCursor(p planPos) {
	e.ta.MoveToBegin()
	for i := 0; e.ta.Line() < p.row && i < 10000; i++ {
		e.ta.CursorDown()
	}
	e.ta.SetCursorColumn(p.col)
}

// selKey handles the pane-owned key family BEFORE the textarea sees them:
// the shift-motions grow the mark, ctrl+a marks all, esc drops a live mark,
// ctrl+v/super+v run the pane's own async clipboard read. Returns
// handled=true when the key was consumed.
func (e *PlanEditor) selKey(kp tea.KeyPressMsg) (tea.Cmd, bool) {
	switch kp.String() {
	case "shift+left":
		e.selEnsureAnchor()
		e.selStepRune(-1)
		return nil, true
	case "shift+right":
		e.selEnsureAnchor()
		e.selStepRune(+1)
		return nil, true
	case "shift+up":
		e.selEnsureAnchor()
		e.ta.CursorUp() // visual row (wrap-aware)
		e.sel.head = planPos{row: e.ta.Line(), col: e.ta.Column()}
		return nil, true
	case "shift+down":
		e.selEnsureAnchor()
		e.ta.CursorDown()
		e.sel.head = planPos{row: e.ta.Line(), col: e.ta.Column()}
		return nil, true
	case "shift+home":
		e.selEnsureAnchor()
		e.sel.head = planPos{row: e.ta.Line(), col: 0}
		e.selSyncCursor(e.sel.head)
		return nil, true
	case "shift+end":
		e.selEnsureAnchor()
		lines := strings.Split(e.ta.Value(), "\n")
		e.sel.head = planPos{row: e.ta.Line(), col: len([]rune(lines[e.ta.Line()]))}
		e.selSyncCursor(e.sel.head)
		return nil, true
	case "ctrl+a":
		e.SelectAll()
		return nil, true
	case "esc":
		// esc owns the selection first: clear the mark, keep the keys and
		// the focus; the NEXT esc blurs back to chat (the app's gate).
		if e.sel.active {
			e.ClearSelection()
			return nil, true
		}
		return nil, false
	case "ctrl+v", "super+v":
		// the pane OWNS its paste (see the file header: bubbles'
		// textarea.Paste would land in the chat draft). The read is async:
		// the answer returns as PlanPasteMsg and replaces the live mark.
		return planReadClipboardCmd(), true
	}
	return nil, false
}

// selStepRune moves the head one rune left/right in BUFFER space (across
// line breaks at the edges) and syncs the real caret onto it.
func (e *PlanEditor) selStepRune(dir int) {
	lines := strings.Split(e.ta.Value(), "\n")
	h := e.sel.head
	if h.row >= len(lines) {
		h.row = len(lines) - 1
	}
	lineLen := len([]rune(lines[h.row]))
	h.col += dir
	if h.col < 0 {
		if h.row > 0 {
			h.row--
			h.col = len([]rune(lines[h.row]))
		} else {
			h.col = 0
		}
	} else if h.col > lineLen {
		if h.row < len(lines)-1 {
			h.row++
			h.col = 0
		} else {
			h.col = lineLen
		}
	}
	e.sel.head = h
	e.selSyncCursor(h)
}

// selText extracts the marked bytes AS-IS: exact runes between the
// normalized endpoints over the Value() line split, joined with "\n" (the
// buffer's own separators — no trimming, indentation preserved verbatim).
func (e *PlanEditor) selText() string {
	if !e.sel.active {
		return ""
	}
	lo, hi := e.sel.norm()
	lines := strings.Split(e.ta.Value(), "\n")
	if lo.row >= len(lines) {
		return ""
	}
	if hi.row >= len(lines) {
		hi.row = len(lines) - 1
		hi.col = len([]rune(lines[hi.row]))
	}
	clampCol := func(row, col int) int {
		if col < 0 {
			return 0
		}
		if n := len([]rune(lines[row])); col > n {
			return n
		}
		return col
	}
	lo.col, hi.col = clampCol(lo.row, lo.col), clampCol(hi.row, hi.col)
	if lo.row == hi.row {
		r := []rune(lines[lo.row])
		return string(r[lo.col:hi.col])
	}
	parts := make([]string, 0, hi.row-lo.row+1)
	first := []rune(lines[lo.row])
	parts = append(parts, string(first[lo.col:]))
	for i := lo.row + 1; i < hi.row; i++ {
		parts = append(parts, lines[i])
	}
	last := []rune(lines[hi.row])
	parts = append(parts, string(last[:hi.col]))
	return strings.Join(parts, "\n")
}

// CopySelection writes the marked text to the OS pasteboard through the
// stubbed clipboard.go seam and returns (text, runeCount, verdict): the app
// rides tea.SetClipboard + the "Copied N chars" toast on success.
func (e *PlanEditor) CopySelection() (string, int, error) {
	text := e.selText()
	if text == "" {
		return "", 0, nil
	}
	if err := clipboardCopyText(text); err != nil {
		return text, len([]rune(text)), err
	}
	return text, len([]rune(text)), nil
}

// CutSelection is CopySelection + the marked bytes deleted, with the caret
// restored to the range START (MoveToBegin → CursorDown walk → column) and
// userDirty latched via the same before/after compare the keystroke path
// uses. The mark clears (the range no longer exists).
func (e *PlanEditor) CutSelection() (string, int, error) {
	if !e.sel.active {
		return "", 0, nil
	}
	text, n, err := e.CopySelection()
	before := e.ta.Value()
	lo, _ := e.sel.norm()
	e.deleteRange()
	e.ClearSelection()
	e.selSyncCursor(lo)
	if e.ta.Value() != before {
		e.userDirty = true
	}
	return text, n, err
}

// deleteRange splices the marked bytes out of the buffer: prefix of the lo
// line + suffix of the hi line merged into one line (buffer split lines,
// exact bytes — no trimming).
func (e *PlanEditor) deleteRange() {
	lo, hi := e.sel.norm()
	lines := strings.Split(e.ta.Value(), "\n")
	if lo.row >= len(lines) || hi.row >= len(lines) {
		return
	}
	clampCol := func(row, col int) int {
		if col < 0 {
			return 0
		}
		if n := len([]rune(lines[row])); col > n {
			return n
		}
		return col
	}
	lo.col, hi.col = clampCol(lo.row, lo.col), clampCol(hi.row, hi.col)
	merged := string([]rune(lines[lo.row])[:lo.col]) + string([]rune(lines[hi.row])[hi.col:])
	out := make([]string, 0, len(lines)-(hi.row-lo.row))
	out = append(out, lines[:lo.row]...)
	out = append(out, merged)
	out = append(out, lines[hi.row+1:]...)
	e.ta.SetValue(strings.Join(out, "\n"))
}

// pasteContent inserts paste bytes at the caret — REPLACING the live mark
// first (the classic paste-over-selection), then the textarea's own
// bracketed-paste arm does the insert (its sanitizer's tab→4sp rides along,
// markdown columns fine). Used by BOTH the direct tea.PasteMsg (cmd+v
// bracketed) and the pane's own PlanPasteMsg (ctrl+v / super+v readback).
func (e *PlanEditor) pasteContent(content string) tea.Cmd {
	if content == "" {
		return nil
	}
	before := e.ta.Value()
	if e.sel.active {
		lo, _ := e.sel.norm()
		e.deleteRange()
		e.ClearSelection()
		e.selSyncCursor(lo)
	}
	var cmd tea.Cmd
	e.ta, cmd = e.ta.Update(tea.PasteMsg{Content: content})
	if e.ta.Value() != before {
		e.userDirty = true
	}
	return cmd
}

// --- mouse drag (app-routed pane-body coords) ------------------------------

// SelectionBeginAt arms the drag at a pane-body-space point (col 0 at the
// pane's left edge, vrow 0 at the body's first row — the app subtracts the
// pane chrome) and places the caret there: the press decides nothing yet.
// The MARK materializes only on the first head movement (dragging arms,
// active stays false) — a motionless press is caret placement ONLY: no
// highlight, SelectionActive()==false (ctrl+x keeps its approve claim, esc
// keeps its blur claim), no copy on release.
func (e *PlanEditor) SelectionBeginAt(col, vrow int) {
	p := e.mapPoint(col, vrow)
	e.sel = planSelState{dragging: true, anchor: p, head: p}
	e.selSyncCursor(p)
}

// SelectionDragTo moves the armed drag's head to the pane-body point
// (clamped at the buffer's edges) and returns whether the head's BUFFER
// position actually moved — the app latches its "dragged" verdict off this
// so motion storms over one cell stay a caret click. The first real move is
// what makes the mark VISIBLE (active).
func (e *PlanEditor) SelectionDragTo(col, vrow int) bool {
	if !e.sel.dragging {
		return false
	}
	p := e.mapPoint(col, vrow)
	if p == e.sel.head {
		return false
	}
	e.sel.head = p
	e.sel.active = true // the first real move materializes the mark
	e.selSyncCursor(p)
	return true
}

// SelectionFinish settles a dragged release: snap the head to the release
// point, finalize the drag (the highlight persists — webpage rule), and
// copy like ctrl+c does. A zero-width (motionless) finish decides NOTHING:
// ("", 0, nil) — the app clears + toasts nothing.
func (e *PlanEditor) SelectionFinish(col, vrow int) (string, int, error) {
	if !e.sel.active {
		return "", 0, nil
	}
	if e.sel.dragging {
		e.sel.head = e.mapPoint(col, vrow)
		e.sel.dragging = false
	}
	return e.CopySelection()
}

// mapPoint resolves a pane-body-space (col, vrow) into a BUFFER position:
// vrow walks the soft-wrap segments PLUS the textarea's scroll offset; the
// cell column maps through the prompt gutter (cells of the wrapped segment,
// double-width runes accounted) and clamps to the line's real runes (the
// synthetic trailing-space padding the textarea renders never lands a caret
// past the line's own end). Out-of-buffer landings snap to the nearest edge.
func (e *PlanEditor) mapPoint(col, vrow int) planPos {
	lines := strings.Split(e.ta.Value(), "\n")
	if vrow < 0 {
		vrow = 0
	}
	vr := vrow + e.ta.ScrollYOffset()
	tw := e.wrapWidth()
	pw := lipgloss.Width(e.ta.Prompt)
	for i, ln := range lines {
		segs := planWrapSegments(ln, tw)
		if vr < len(segs) {
			seg := segs[vr]
			cell := col - pw
			if cell < 0 {
				cell = 0
			}
			pos := seg.start + seg.real // default: end of the real content
			acc := 0
			for j := 0; j < seg.real && j < len(seg.text); j++ {
				w := ansi.StringWidth(string(seg.text[j]))
				if acc+w > cell { // the click landed inside rune j: caret before it
					pos = seg.start + j
					break
				}
				acc += w
			}
			if n := len([]rune(ln)); pos > n {
				pos = n
			}
			return planPos{row: i, col: pos}
		}
		vr -= len(segs)
	}
	last := len(lines) - 1
	return planPos{row: last, col: len([]rune(lines[last]))}
}

// --- the SGR-7 highlight splice --------------------------------------------

// overlaySelection splices the reverse-video highlight into the FOCUSED
// textarea render (one pass over the body rows, ANSI-aware via the shared
// selHighlight): buffer endpoints → soft-wrap segments → per-visual-row
// cell spans [2+gutter …], rows past the scroll offset skipped. Pure render;
// the textarea is untouched.
func (e *PlanEditor) overlaySelection(view string) string {
	if !e.sel.active {
		return view
	}
	rows := strings.Split(view, "\n")
	spans := e.selAbsSpans()
	off := e.ta.ScrollYOffset()
	for abs, span := range spans {
		vi := abs - off
		if vi < 0 || vi >= len(rows) {
			continue
		}
		rows[vi] = selHighlight(rows[vi], span[0], span[1])
	}
	return strings.Join(rows, "\n")
}

// selAbsSpans computes the highlight's per-ABSOLUTE-visual-row cell spans
// (subtract ScrollYOffset for render rows). A marked range running to a
// line's end paints that row's tail bar to the body width (the webpage
// full-row look); the hi row stops at the head's own cell.
func (e *PlanEditor) selAbsSpans() map[int][2]int {
	spans := map[int][2]int{}
	if !e.sel.active {
		return spans
	}
	lo, hi := e.sel.norm()
	lines := strings.Split(e.ta.Value(), "\n")
	tw := e.wrapWidth()
	bw, _ := e.bodyDims()
	pw := lipgloss.Width(e.ta.Prompt)
	abs := 0
	for i, ln := range lines {
		segs := planWrapSegments(ln, tw)
		if i < lo.row || i > hi.row {
			abs += len(segs)
			continue
		}
		for s, seg := range segs {
			segEnd := seg.start + seg.real
			a, b := seg.start, segEnd
			if i == lo.row && lo.col > a {
				a = lo.col
			}
			if i == hi.row && hi.col < b {
				b = hi.col
			}
			if b < a {
				continue // wrapped segment before the caret's own (single line)
			}
			from := pw + segCells(seg, a-seg.start)
			to := pw + segCells(seg, b-seg.start)
			wholeRowBar := i < hi.row && b >= segEnd // the range continues past this line
			if wholeRowBar && to < bw {
				to = bw // the row's tail bar to the body edge
			}
			if to <= from {
				if wholeRowBar { // empty segment mid-range: the bar still paints
					spans[abs+s] = [2]int{from, bw}
				}
				continue
			}
			spans[abs+s] = [2]int{from, to}
		}
		abs += len(segs)
	}
	return spans
}

// segCells measures the CELLS spanned by the first idx runes of a segment
// (idx clamps into the segment's real text; double-width runes count 2).
func segCells(seg planWrapSeg, idx int) int {
	if idx < 0 {
		idx = 0
	}
	if idx > len(seg.text) {
		idx = len(seg.text)
	}
	w := 0
	for _, r := range seg.text[:idx] {
		w += ansi.StringWidth(string(r))
	}
	return w
}

// wrapWidth is the textarea's CONTENT width (what its wrap() breaks at):
// the pane body's SetWidth min-clamped to 3, minus the prompt gutter.
func (e *PlanEditor) wrapWidth() int {
	w := e.w
	if w < 3 {
		w = 3 // textarea.SetWidth's minWidth (prompt 2 + 1)
	}
	return w - 2
}

// --- the soft-wrap replica (bubbles v2.1.1 textarea wrap(), byte-faithful) --

// planWrapSeg is one VISUAL row of a soft-wrapped buffer line, exactly as
// the textarea renders it: text carries the synthetic trailing space the
// textarea pads at segment ends, real counts only the ORIGINAL line runes
// (so buffer columns never land on padding), start is the segment's first
// original rune's offset in the line.
type planWrapSeg struct {
	start int
	real  int
	text  []rune
}

// planWrapSegments replicates bubbles v2.1.1 textarea's unexported wrap()
// (same word-wrap + trailing-space rules, every break condition verbatim)
// so mouse mapping and the highlight splice see THE SAME segment
// boundaries the textarea renders. The only deliberate swap: uniseg/mattn
// width calls become x/ansi's StringWidth — width-identical for the
// markdown/ASCII plans this pane edits, and it keeps go.mod untouched.
func planWrapSegments(line string, width int) []planWrapSeg {
	if width < 1 {
		width = 1
	}
	segs := []planWrapSeg{{text: []rune{}}}
	row := 0
	realStart := 0
	realCount := 0
	word := []rune{}
	spaces := 0
	pushBreak := func() {
		segs[row].real = realCount
		realStart += realCount
		realCount = 0
		segs = append(segs, planWrapSeg{start: realStart, text: []rune{}})
		row++
	}
	for _, r := range line {
		if unicode.IsSpace(r) {
			spaces++
		} else {
			word = append(word, r)
		}
		if spaces > 0 {
			if ansi.StringWidth(string(segs[row].text))+ansi.StringWidth(string(word))+spaces > width {
				pushBreak()
				segs[row].text = append(segs[row].text, word...)
				segs[row].text = append(segs[row].text, repeatSpacesRunes(spaces)...)
				realCount += len(word) + spaces
				spaces = 0
				word = nil
			} else {
				segs[row].text = append(segs[row].text, word...)
				segs[row].text = append(segs[row].text, repeatSpacesRunes(spaces)...)
				realCount += len(word) + spaces
				spaces = 0
				word = nil
			}
		} else {
			lastCharLen := ansi.StringWidth(string(word[len(word)-1]))
			if ansi.StringWidth(string(word))+lastCharLen > width {
				if len(segs[row].text) > 0 {
					pushBreak()
				}
				segs[row].text = append(segs[row].text, word...)
				realCount += len(word)
				word = nil
			}
		}
	}
	if ansi.StringWidth(string(segs[row].text))+ansi.StringWidth(string(word))+spaces >= width {
		pushBreak()
		segs[row].text = append(segs[row].text, word...)
		realCount += len(word) + spaces
		segs[row].text = append(segs[row].text, repeatSpacesRunes(spaces+1)...)
		spaces = 0
		word = nil
	} else {
		segs[row].text = append(segs[row].text, word...)
		realCount += len(word) + spaces
		segs[row].text = append(segs[row].text, repeatSpacesRunes(spaces+1)...)
		spaces = 0
		word = nil
	}
	segs[row].real = realCount
	return segs
}

// repeatSpacesRunes mirrors the textarea's repeatSpaces: runes worth n spaces.
func repeatSpacesRunes(n int) []rune {
	if n <= 0 {
		return nil
	}
	return []rune(strings.Repeat(" ", n))
}

// --- the async clipboard-read seam (ctrl+v / super+v) ------------------------

// clipboardReadText is THE paste-side read seam: the pane's ctrl+v/super+v
// runs it inside a tea.Cmd (the shell-out never parks the input loop).
// Tests swap in a fixture (t.Cleanup restores), mirroring clipboardCopyText.
var clipboardReadText = systemClipboardRead

var (
	clipReadOnce sync.Once
	clipReadRun  func() (string, error) // nil when no tool exists on this host
)

// errNoClipboardReadTool — the read side's degraded-platform verdict (the
// app surfaces it as the dim "paste failed" note, never silently).
var errNoClipboardReadTool = fmt.Errorf("no clipboard reader (pbpaste / wl-paste / xclip / xsel)")

// systemClipboardRead resolves the host's clipboard READ tool lazily
// (clipboard.go's copy-table twin) and drains it: darwin pbpaste, windows
// Get-Clipboard, elsewhere wayland first then X11.
func systemClipboardRead() (string, error) {
	clipReadOnce.Do(func() {
		type tool struct {
			name string
			args []string
		}
		var tools []tool
		switch runtime.GOOS {
		case "darwin":
			tools = []tool{{"pbpaste", nil}}
		case "windows":
			tools = []tool{{"powershell", []string{"-NoProfile", "-Command", "Get-Clipboard"}}}
		default: // linux + the BSDs: wayland first, then X11
			tools = []tool{
				{"wl-paste", []string{"--no-newline"}},
				{"xclip", []string{"-selection", "clipboard", "-o"}},
				{"xsel", []string{"--clipboard", "--output"}},
			}
		}
		for _, t := range tools {
			if _, err := exec.LookPath(t.name); err == nil {
				t := t
				clipReadRun = func() (string, error) {
					out, err := exec.Command(t.name, t.args...).Output()
					if err != nil {
						return "", fmt.Errorf("%s: %w", t.name, err)
					}
					return string(out), nil
				}
				return
			}
		}
	})
	if clipReadRun == nil {
		return "", errNoClipboardReadTool
	}
	return clipReadRun()
}

// planReadClipboardCmd runs the read seam inside a tea.Cmd so the shell-out
// never parks the input loop; the answer comes back as PlanPasteMsg and the
// app routes it to the focused pane only.
func planReadClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := clipboardReadText()
		if err != nil {
			return PlanPasteMsg{Err: err}
		}
		return PlanPasteMsg{Content: s}
	}
}
