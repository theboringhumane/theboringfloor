// terminal.go — the TERM tab: an OS-level terminal embedded in the right
// sidebar. Backed by internal/term.Session (a real PTY running the user's
// shell). The panel:
//
//	body    — the live screen model (internal/term.Grid): cursor-positioned,
//	         SGR-styled cells painted directly with lipgloss. The prompt
//	         stays put, clear/redraw repaints cleanly, and ?1049 alt-screen
//	         apps paint their alternate screen until exit. Mouse-wheel
//	         scroll (and the dead-shell view) fall back to the sanitized
//	         raw scrollback, which holds the full retained byte stream.
//	footer  — a one-row badge: "[tty] focused · ctrl+space to release" while
//	         the terminal CAPTURED the keyboard; "[tty] inactive" (dim)
//	         while RELEASED — the office keys own the terminal tab again;
//	         a red "terminal exited (code N) — press r to respawn" line
//	         replaces the whole body when the shell dies. A copy note
//	         ("· Copied N chars" / the dim clipboard failure) rides the
//	         badge for termNoteWindow when a drag-release copies.
//
// Keyboard contract (wave-42: capture is OPT-IN — the app flips Focus/Blur
// via its ctrl+space toggle (both ways) and ctrl+o release alias; see
// internal/term/term.go for the full byte-level matrix). Only while Focused
// do chars/enter/backspace/tab/esc/arrows/home/end/pgup/pgdown/delete/
// ctrl+letter forward to the PTY; ctrl+space and ctrl+o are RESERVED
// (release capture back to the app — never reach the shell). Mouse wheel
// scrolls the retained scrollback; a click focuses.
//
// MOUSE CONTRACT — webpage-style text selection (the chat transcript twin,
// internal/app/selection.go + panels/chat_selection.go are the design
// precedent; THIS panel is fully self-contained):
//
//	press   — a LEFT press over a body row ARMS a pending selection in
//	          VIEWPORT space (row = the painted row under the cursor,
//	          col = the painted cell; both clamped at the edges like the
//	          chat panel). The same press ALSO keeps the legacy
//	          click-focuses behavior — capture starts on the press, the
//	          fate of the drag stays undecided.
//	motion  — with the button held, the head follows (clamped into body
//	          space so dragging past an edge pins at the edge).
//	release — NO motion since the press = a plain click: the arm retires
//	          silently (the press already focused). A DRAGGED release
//	          extracts the plain text, copies it SYNCHRONOUSLY through
//	          clipboardCopyText (clipboard.go — pbcopy on darwin, wl-copy/
//	          xclip/xsel on linux, clip on windows; the note gates on the
//	          REAL verdict, never a swallowed escape) and returns
//	          tea.SetClipboard(text) so an OSC52-capable outer terminal
//	          wins too (chat's exact belt+fallback). The highlight
//	          persists until cleared.
//
//	Copy confirmation rides the badge row dim (" · Copied N chars"; a
//	failed copy shows the dim "no clipboard tool" note) for
//	termNoteWindow; SetState (the office tick) is the expiry clock — no
//	timers.
//
// CLEARING (frozen rules): the selection retires on (a) new input to the
// PTY — any keystroke that forwards bytes; (b) respawn ("r" on a dead
// shell) and spawn generally; (c) esc — which OWNS the highlight first
// (focused or blurred it cancels the selection and never reaches the
// shell; a second esc does); (d) a press OUTSIDE the body rows (webpage
// rule); (e) a fresh arm. Releasing capture (ctrl+space / ctrl+o) does
// NOT clear — the highlight survives the hop back to the office keys.
//
// COORDINATE SPACE: mouse messages arrive in sidebar-box space (the app
// forwards panel mouse msgs origin-at-the-tabs-box, the same convention
// gitpanel_test.go:488 builds with Tabs.ContentOffset); cellAt subtracts
// (dx, dy) — badge row and outside never arm. Endpoints live in VIEWPORT
// space of the CURRENT render (grid rows while scroll == 0, the
// scrollback window while scrolled); wheel/pgup/pgdn scrolling with a
// selection up SHIFTS both endpoints by the scroll delta so the highlight
// stays pinned to the words (edge-clamped, again the chat rule). The
// live↔history render seam (scroll 0↔1 switches sources) is
// approximate-by-construction in v1.
//
// PTY MOUSE MODES: the grid parser accepts and IGNORES ?1000/?1006 mouse
// reporting (grid.go privateMode handles only ?1049), and no mouse bytes
// are EVER written to the PTY (Session.Write carries keyToBytes only) —
// a shell app's mouse mode can therefore never fight this host-side
// selection in v1: selection always wins. Tradeoff: mouse-aware TUIs
// (vim mouse=on, htop) get no clicks — use the keyboard there, or the
// outer terminal's own native selection when the panel is blurred.
//
// WIDE CHARS / TABS: selection cells are GRID cells — one rune per cell
// exactly as the grid stores them (tabs are 8-stop-expanded by the grid,
// wide glyphs inherit the grid's own cell accounting; the history path's
// sanitizeLine lead-byte counting matches the same rune=cell space).
package panels

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
	"github.com/theboringhumane/theboringoffice/internal/term"
)

// termSess is the panel's seam onto a live PTY session — *term.Session
// satisfies it in production; tests drive a fake (a real Grid + real
// Scrollback, no PTY, no sleeps).
type termSess interface {
	Alive() bool
	Close() error
	Write([]byte) (int, error)
	Resize(cols, rows int) error
	Size() (cols, rows int)
	ExitCode() int
	Grid() *term.Grid
	Scrollback() *term.Scrollback
}

// spawnTermSession is the spawn seam (the app's SpawnTerminal precedent):
// production wires term.Spawn; respawn tests swap in a fake so no test
// ever owns a real PTY.
var spawnTermSession = term.Spawn

// TermPanel is the terminal sidebar tab. It satisfies Tab + Interactive
// exactly like chat/agents; the app additionally calls Focus()/Blur() when
// the term tab becomes (in)active and Close() at quit.
type TermPanel struct {
	sess  termSess // nil only transiently during respawn
	shell string   // shell path remembered for respawn
	cwd   string
	w, h  int

	focused  bool
	scroll   int // rows up from the bottom (mouse wheel viewing)
	spawnErr error
	rev      uint64 // cheap change detection for View caching
	cached   string

	sel    termSel   // the mouse text selection (header: MOUSE CONTRACT)
	note   string    // copy verdict note on the badge row (dim, brief)
	noteAt time.Time // when the note armed (termNoteWindow expiry)
}

// termNoteWindow — how long the copy verdict rides the badge row (the
// chat transcript's 2s toast window, house-consistent).
const termNoteWindow = 2 * time.Second

// termSel states: idle → armed (press down, fate undecided) → done
// (dragged release copied; highlight persists) → idle (clear rules).
const (
	termSelIdle = iota
	termSelArmed
	termSelDone
)

// termSelPoint is one selection endpoint in VIEWPORT space (row = painted
// body row, col = painted cell).
type termSelPoint struct{ row, col int }

// termSel — ONE live selection: the anchor (press cell) and head (current
// drag cell). dragged mirrors the chat panel's "the press's fate stays
// undecided until release" rule: any motion with the button held makes
// the release a copy, none keeps it a plain click.
type termSel struct {
	state   int
	dragged bool
	a, h    termSelPoint
}

// NewTerminal spawns the user's shell NOW on cols=width rows=height-1
// (one row reserved for the badge) and returns the ready panel. If the
// shell can't spawn the panel still comes up, showing the spawn error in
// the dead-shell body (r retries). The keyboard starts RELEASED — the app
// opts into capture per visit with ctrl+space (Focus), so a fresh panel
// must never assume the member wants the shell to own their keys.
func NewTerminal(width, height int) (*TermPanel, error) {
	p := &TermPanel{shell: term.DefaultShell()}
	p.SetSize(width, height)
	if err := p.spawn(); err != nil {
		p.spawnErr = err
		return p, err
	}
	return p, nil
}

// spawn starts a fresh session at the current panel geometry. Old sessions
// must be Close()d first (respawn does it). A respawn attempt retires the
// selection outright (its viewport rows belong to the dead session's
// frame); a SUCCESSFUL spawn also resets the view offset.
func (p *TermPanel) spawn() error {
	p.sel = termSel{} // respawn clears — even when the spawn fails
	sess, err := spawnTermSession(term.TermConfig{
		Shell: p.shell,
		Cols:  p.w,
		Rows:  p.bodyH(),
		CWD:   p.cwd,
	})
	if err != nil {
		return err
	}
	p.sess = sess
	p.scroll = 0
	p.rev = 0
	p.cached = ""
	return nil
}

// Title implements Tab.
func (p *TermPanel) Title() string { return "term" }

// Focus CAPTURES the keyboard for the terminal (the app's ctrl+space dive);
// the badge flips to the focused hint.
func (p *TermPanel) Focus() { p.focused = true }

// Blur RELEASES the keyboard back to the office (the app's ctrl+space
// toggle-off / ctrl+o release alias / auto-release on tab-leave; ctrl+space
// and ctrl+o also blur internally).
func (p *TermPanel) Blur() { p.focused = false }

// Focused reports whether keystrokes are forwarded to the shell.
func (p *TermPanel) Focused() bool { return p.focused }

// Alive reports whether the shell is running.
func (p *TermPanel) Alive() bool { return p.sess != nil && p.sess.Alive() }

// Close kills the shell's whole process group (zombie-proof at app quit).
// Idempotent.
func (p *TermPanel) Close() error {
	if p.sess == nil {
		return nil
	}
	err := p.sess.Close()
	return err
}

// Session exposes the live PTY session (wiring dev: tests, raw access).
// Nil while a test harness drives a fake sess (termSess seam).
func (p *TermPanel) Session() *term.Session {
	if s, ok := p.sess.(*term.Session); ok {
		return s
	}
	return nil
}

// bodyH is the terminal viewport height (panel height minus the badge row).
func (p *TermPanel) bodyH() int {
	h := p.h - 1
	if h < 1 {
		h = 1
	}
	return h
}

// SetSize implements Tab: sizes the panel you see AND resizes the
// underlying PTY (SIGWINCH reaches the shell).
func (p *TermPanel) SetSize(w, h int) {
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}
	p.w, p.h = w, h
	p.rev = 0 // invalidate render cache
	if p.sess != nil && p.sess.Alive() {
		cols, rows := p.sess.Size()
		if cols != w || rows != p.bodyH() {
			_ = p.sess.Resize(w, p.bodyH())
		}
	}
}

// SetState implements Tab: the office tick is our refresh clock — every
// push invalidates the render cache if the screen model moved (grid rev is
// the live signal while the shell runs; scrollback rev covers the edges).
// A FRESH copy note also forces a rebuild per push: the tick is the note's
// expiry clock (no timers), so the badge flips off at the first push past
// termNoteWindow.
func (p *TermPanel) SetState(st state.OfficeState) {
	if p.sess == nil {
		return
	}
	noteFresh := p.note != "" && time.Since(p.noteAt) < termNoteWindow
	rev := p.sess.Grid().Rev() + p.sess.Scrollback().Rev()
	if rev != p.rev || noteFresh {
		p.rev = rev
		p.cached = ""
	}
}

// Update implements Interactive. While focused every keypress goes to the
// PTY (ctrl+space / ctrl+o release focus); while blurred only viewing keys
// work (pgup/pgdn scroll) plus "r" to respawn a dead shell. Mouse: wheel
// scrolls; left press/drag/release run the text-selection contract in the
// file header (both focus states — the press itself captures).
func (p *TermPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()
		if !p.Alive() {
			if key == "r" {
				_ = p.sess.Close()
				p.spawnErr = p.spawn()
				p.cached = ""
			}
			if (key == "ctrl+space" || key == "ctrl+o") && p.focused {
				p.Blur()
			}
			return nil
		}
		if p.focused {
			if key == "ctrl+space" || key == "ctrl+o" {
				// internal belt: the app gates both keeps before its own
				// forward, but a directly-driven panel releases here too.
				p.Blur()
				p.cached = ""
				return nil
			}
			if key == "esc" && p.sel.state != termSelIdle {
				// esc OWNS the highlight first (webpage rule): while a
				// selection is up it cancels and never reaches the shell —
				// a second esc forwards as the real 0x1b.
				p.selClear()
				return nil
			}
			if b, ok := keyToBytes(msg); ok {
				// new input to the PTY retires the selection (frozen rule)
				p.selClear()
				_, _ = p.sess.Write(b)
				p.cached = "" // the echo repaints on the next frame
			}
			return nil // every key is consumed while focused
		}
		// blurred: viewing-only
		switch key {
		case "pgup":
			p.scrollView(1)
		case "pgdown":
			p.scrollView(-1)
		case "esc":
			p.selClear()
		}
		return nil
	case tea.PasteMsg:
		// Bracketed paste into the shell (the app's router sends it here
		// only while CAPTURED — released, the chat textarea takes it).
		// On the MAIN screen the bytes go to the PTY re-wrapped in
		// bracketed-paste markers (ESC[200~ … ESC[201~): readline/zle/
		// fish negotiated ?2004 at the prompt and treat the whole blob as
		// ONE paste unit — a multi-line clipboard never auto-EXECs
		// line-by-line. Inside an ALT-SCREEN app (vim, htop — ?1049, the
		// one private mode the grid tracks) the application owns the
		// keyboard itself: write the bytes raw, exactly like the key
		// path's forwarding discipline (keyToBytes).
		if p.focused && p.Alive() {
			p.selClear() // new input to the PTY retires the selection (frozen rule)
			_, _ = p.sess.Write(pasteToPTY(msg.Content, p.sess.Grid().AltActive()))
			p.cached = ""
		}
		return nil
	case tea.MouseWheelMsg:
		// wheel keeps SCROLLING even mid-selection: scrollView shifts the
		// armed endpoints by the delta so the span stays pinned to the
		// words (scroll ≠ clear — documented in the header).
		switch msg.Button {
		case tea.MouseWheelUp:
			p.scrollView(1)
		case tea.MouseWheelDown:
			p.scrollView(-1)
		}
		return nil
	case tea.MouseClickMsg:
		p.click(msg)
		return nil
	case tea.MouseMotionMsg:
		p.motion(msg)
		return nil
	case tea.MouseReleaseMsg:
		return p.release(msg)
	}
	return nil
}

// click handles a mouse PRESS: the legacy click-focuses behavior first
// (any button keeps it), then — a LEFT press over a body row arms the
// selection; a press anywhere else (badge row, gutters, dead shell) clears
// a finished highlight (webpage rule).
func (p *TermPanel) click(msg tea.MouseClickMsg) {
	if !p.focused {
		p.Focus()
		p.cached = ""
	}
	if msg.Button != tea.MouseLeft {
		return
	}
	cx, cy := p.cellAt(msg.X, msg.Y)
	if !p.Alive() || cx < 0 || cx >= p.w || cy < 0 || cy >= p.bodyH() {
		p.selClear()
		return
	}
	p.sel = termSel{state: termSelArmed}
	p.sel.a = termSelPoint{row: cy, col: cx}
	p.sel.h = p.sel.a
	p.cached = "" // the one-cell anchor highlight repaints
}

// motion extends the armed selection's drag head (clamped into body space
// so out-of-region drags pin at the edge — the chat panel's rule); cheap
// no-op without an armed drag or when the cell didn't move.
func (p *TermPanel) motion(msg tea.MouseMotionMsg) {
	if p.sel.state != termSelArmed {
		return
	}
	cx, cy := p.cellAt(msg.X, msg.Y)
	p.sel.dragged = true
	row, col := termClampCell(cy, cx, p.bodyH()-1, p.w-1)
	if p.sel.h.row == row && p.sel.h.col == col {
		return
	}
	p.sel.h = termSelPoint{row: row, col: col}
	p.cached = ""
}

// release settles an armed selection: motionless = the plain click the
// press promised (the press ALREADY focused — just retire the arm);
// dragged = extract, copy through the platform seam (note gates on the
// verdict) and return the OSC52 fallback cmd. n == 0 (blank spans) decides
// nothing: clear silently.
func (p *TermPanel) release(msg tea.MouseReleaseMsg) tea.Cmd {
	if p.sel.state != termSelArmed {
		return nil
	}
	if !p.sel.dragged {
		p.sel.state = termSelIdle
		p.cached = ""
		return nil
	}
	cx, cy := p.cellAt(msg.X, msg.Y)
	row, col := termClampCell(cy, cx, p.bodyH()-1, p.w-1)
	p.sel.h = termSelPoint{row: row, col: col}
	p.sel.state = termSelDone
	text, n := p.selText()
	if n == 0 {
		p.selClear()
		return nil
	}
	if err := clipboardCopyText(text); err != nil {
		p.note = " · copy failed (no clipboard tool)"
	} else {
		p.note = " · Copied " + itoa(n) + " chars"
	}
	p.noteAt = time.Now()
	p.cached = ""
	return tea.SetClipboard(text) // OSC52 best-effort rides along
}

// cellAt projects a mouse point from sidebar-box space (the app's mouse
// convention for panels — gitpanel_test.go:488's exact seam: the panel
// subtracts Tabs.ContentOffset itself) into panel-local Body cells: (0,0)
// is the first body row; cy == bodyH() is the badge row (never selectable).
func (p *TermPanel) cellAt(x, y int) (col, row int) {
	dx, dy := (&Tabs{}).ContentOffset()
	return x - dx, y - dy
}

// selClear retires any live selection (armed or done) and repaints —
// the no-op-when-idle rule keeps it free on every off-selection event.
func (p *TermPanel) selClear() {
	if p.sel.state == termSelIdle {
		return
	}
	p.sel = termSel{}
	p.cached = ""
}

// scrollView moves the view offset runes-worth of rows through the
// retained scrollback (clamped). A live selection shifts with the scroll
// delta so the highlight stays pinned to the words, edge-clamped (chat
// rule); rows that scroll out of view pin at the nearest edge.
func (p *TermPanel) scrollView(d int) {
	max := 0
	if p.sess != nil {
		if n := len(p.sess.Scrollback().Lines()) - p.bodyH(); n > 0 {
			max = n
		}
	}
	old := p.scroll
	p.scroll += d
	if p.scroll < 0 {
		p.scroll = 0
	}
	if p.scroll > max {
		p.scroll = max
	}
	if delta := p.scroll - old; delta != 0 && p.sel.state != termSelIdle {
		p.sel.a.row = termClampInt(p.sel.a.row+delta, 0, p.bodyH()-1)
		p.sel.h.row = termClampInt(p.sel.h.row+delta, 0, p.bodyH()-1)
	}
	p.cached = ""
}

// pasteToPTY renders one paste's PTY bytes: alt-screen apps (vim & co —
// they own the keyboard) get the content raw; the main-screen shell gets
// it wrapped in bracketed-paste markers so bracketed-aware line editors
// (readline, zle, fish) treat it as ONE paste unit. The wrap is one pair
// of markers around the WHOLE content, newlines included.
func pasteToPTY(content string, altActive bool) []byte {
	if altActive {
		return []byte(content)
	}
	b := make([]byte, 0, len(content)+6)
	b = append(b, "\x1b[200~"...)
	b = append(b, content...)
	b = append(b, "\x1b[201~"...)
	return b
}

// keyToBytes maps a bubbletea keypress to the byte sequence the shell
// expects. The full matrix lives in internal/term/term.go's header.
func keyToBytes(msg tea.KeyPressMsg) ([]byte, bool) {
	switch msg.String() {
	case "enter":
		return []byte("\r"), true
	case "backspace":
		return []byte{0x7f}, true
	case "tab":
		return []byte{0x09}, true
	case "shift+tab":
		return []byte("\x1b[Z"), true // reverse tab (back-completion)
	case "esc":
		return []byte{0x1b}, true
	case "space":
		return []byte(" "), true
	case "up":
		return []byte("\x1b[A"), true
	case "down":
		return []byte("\x1b[B"), true
	case "right":
		return []byte("\x1b[C"), true
	case "left":
		return []byte("\x1b[D"), true
	case "home":
		return []byte("\x1b[H"), true
	case "end":
		return []byte("\x1b[F"), true
	case "pgup":
		return []byte("\x1b[5~"), true
	case "pgdown":
		return []byte("\x1b[6~"), true
	case "delete":
		return []byte("\x1b[3~"), true
	}
	if k := msg.String(); len(k) == len("ctrl+x") && strings.HasPrefix(k, "ctrl+") {
		c := k[len(k)-1]
		if c >= 'a' && c <= 'z' {
			return []byte{c - 'a' + 1}, true // 0x01..0x1a pass-through
		}
	}
	if msg.Text != "" {
		return []byte(msg.Text), true
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// Text selection core — endpoints, spans, extraction. All pure viewport-
// space geometry over the SAME rows the body paints (no render-path drift):
// the highlight and the copy read one source of truth.
// ---------------------------------------------------------------------------

// termClampInt / termClampCell — clamp helpers (edge-pin rule).
func termClampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func termClampCell(row, col, maxRow, maxCol int) (int, int) {
	return termClampInt(row, 0, maxRow), termClampInt(col, 0, maxCol)
}

// termSelNorm returns the endpoints in READING ORDER (top-left first) — a
// drag up/left selects exactly what its down/right twin does (chat rule).
func termSelNorm(a, h termSelPoint) (lo, hi termSelPoint) {
	if h.row < a.row || (h.row == a.row && h.col < a.col) {
		return h, a
	}
	return a, h
}

// termSpanOf computes one row's cell span [from, to) inside a selection:
// the start row runs from loCol to the line end, interior rows are whole,
// the end row runs from 0 to hiCol+1 (the cell under the release cursor is
// INCLUDED). Spans may exceed a row's text — callers clamp per row.
func termSpanOf(r int, lo, hi termSelPoint, w int) (from, to int) {
	switch {
	case lo.row == hi.row:
		return lo.col, hi.col + 1
	case r == lo.row:
		return lo.col, w
	case r == hi.row:
		return 0, hi.col + 1
	default:
		return 0, w
	}
}

// selSpans maps the live selection to per-VIEWPORT-row cell spans for the
// highlight overlay. Nil without a selection; rows outside the body never
// get a span.
func (p *TermPanel) selSpans() map[int][2]int {
	if p.sel.state == termSelIdle {
		return nil
	}
	lo, hi := termSelNorm(p.sel.a, p.sel.h)
	spans := map[int][2]int{}
	for r := lo.row; r <= hi.row && r < p.bodyH(); r++ {
		if r < 0 {
			continue
		}
		from, to := termSpanOf(r, lo, hi, p.w)
		spans[r] = [2]int{from, to}
	}
	return spans
}

// selViewportRows returns the PLAIN text of the rows the body currently
// paints — live grid rows (one rune per cell, full width) while
// scroll == 0, the ANSI-stripped scrollback window while scrolled. The
// highlight overlay indexes the SAME rows, so copy == what was lit.
func (p *TermPanel) selViewportRows() []string {
	if p.sess == nil {
		return nil
	}
	if p.scroll == 0 {
		rows := p.sess.Grid().Render()
		out := make([]string, len(rows))
		for i, r := range rows {
			out[i] = gridRowPlain(r)
		}
		return out
	}
	rows := p.historyRows()
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = ansi.Strip(r)
	}
	return out
}

// selText extracts the selected PLAIN text from the current viewport rows:
// per row the span is rune-cut (rune == cell in both render paths), right
// trimmed of padding spaces, and rows join with "\n" (blank interior rows
// keep their newline — webpage semantics). Returns ("", 0) when the spans
// carry no text — "the drag decided nothing".
func (p *TermPanel) selText() (string, int) {
	if p.sel.state == termSelIdle {
		return "", 0
	}
	rows := p.selViewportRows()
	if len(rows) == 0 {
		return "", 0
	}
	lo, hi := termSelNorm(p.sel.a, p.sel.h)
	lo.row = termClampInt(lo.row, 0, len(rows)-1)
	hi.row = termClampInt(hi.row, 0, len(rows)-1)
	parts := make([]string, 0, hi.row-lo.row+1)
	for r := lo.row; r <= hi.row; r++ {
		plain := rows[r]
		n := len([]rune(plain))
		from, to := termSpanOf(r, lo, hi, p.w)
		from = termClampInt(from, 0, n)
		to = termClampInt(to, 0, n)
		if from >= to {
			parts = append(parts, "")
			continue
		}
		parts = append(parts, strings.TrimRight(runeSlice(plain, from, to), " "))
	}
	text := strings.Join(parts, "\n")
	if strings.TrimSpace(text) == "" {
		return "", 0
	}
	// the frozen count: runes INCLUDING the joining newlines — "Copied N
	// chars" counts chars, and a char is a rune (chat's counting).
	return text, len([]rune(text))
}

// runeSlice cuts s to runes [from, to) — the selection's cell space (grid
// cells hold one rune each; sanitizeLine's lead-byte counting agrees).
func runeSlice(s string, from, to int) string {
	rs := []rune(s)
	from = termClampInt(from, 0, len(rs))
	to = termClampInt(to, from, len(rs))
	return string(rs[from:to])
}

// gridRowPlain flattens one screen row to its full-width plain text (one
// rune per cell — the highlight/extraction cell space).
func gridRowPlain(row term.Row) string {
	rs := make([]rune, len(row))
	for i, c := range row {
		ch := c.Ch
		if ch == 0 {
			ch = ' '
		}
		rs[i] = ch
	}
	return string(rs)
}

// historyRows — the exact window the HISTORY path paints: the sanitized
// scrollback tail above the current scroll offset, capped to the body.
func (p *TermPanel) historyRows() []string {
	rows := p.sess.Scrollback().Render(p.bodyH()+p.scroll, p.w)
	if len(rows) > p.scroll {
		rows = rows[:len(rows)-p.scroll]
	}
	for len(rows) > p.bodyH() {
		rows = rows[1:]
	}
	return rows
}

// View implements Tab.
func (p *TermPanel) View() string {
	if p.cached != "" {
		return p.cached
	}
	var b strings.Builder

	if !p.Alive() {
		code := ""
		if p.sess != nil {
			code = itoa(p.sess.ExitCode())
		} else if p.spawnErr != nil {
			code = "spawn failed"
		}
		body := "terminal exited"
		if code != "" {
			body += " (code " + code + ")"
		}
		body += " — press r to respawn"
		b.WriteString(chrome.ErrText.Render(fitPlain(body, p.w)))
		for i := 1; i < p.bodyH(); i++ {
			b.WriteString("\n" + fitPlain("", p.w))
		}
	} else if p.scroll == 0 {
		// LIVE PATH — paint the screen model: cursor-positioned rows with
		// real SGR styling and a block caret. No sanitizer, no flatten.
		grid := p.sess.Grid()
		rows := grid.Render()
		cx, cy := grid.Cursor()
		spans := p.selSpans() // nil without a live selection
		for y, row := range rows {
			if y > 0 {
				b.WriteString("\n")
			}
			caret := -1
			if y == cy && p.focused {
				caret = cx
			}
			if span, ok := spans[y]; ok {
				b.WriteString(gridRowStringSel(row, caret, span[0], span[1]))
			} else {
				b.WriteString(gridRowString(row, caret))
			}
		}
		for i := len(rows); i < p.bodyH(); i++ {
			if len(rows) > 0 || i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fitPlain("", p.w))
		}
	} else {
		// HISTORY PATH — mouse-wheel scrollback: the raw byte window,
		// sanitized (legacy behavior). The selection highlight splices in
		// reverse-video exactly like the chat transcript's (selHighlight).
		rows := p.historyRows()
		spans := p.selSpans()
		for i, r := range rows {
			if i > 0 {
				b.WriteString("\n")
			}
			if span, ok := spans[i]; ok {
				r = selHighlight(r, span[0], span[1])
			}
			b.WriteString(r)
		}
		for i := len(rows); i < p.bodyH(); i++ {
			if len(rows) > 0 || i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fitPlain("", p.w))
		}
	}

	// badge row
	b.WriteString("\n")
	var badge string
	switch {
	case p.focused && p.Alive():
		badge = chrome.TabActive.Render(" tty ") + chrome.DimText.Render(" focused · ctrl+space to release")
	case p.Alive():
		badge = chrome.TabInactive.Render(" tty ") + chrome.DimText.Render(" inactive")
	default:
		badge = chrome.TabInactive.Render(" tty ") + chrome.ErrText.Render(" dead")
	}
	if p.scroll > 0 {
		badge += chrome.DimText.Render(" · scrolled up " + itoa(p.scroll))
	}
	// the copy verdict rides the badge for termNoteWindow (dim either way —
	// success reads " · Copied N chars", the degraded platform note stays
	// dim instead of screaming red in a one-row badge).
	if p.note != "" && time.Since(p.noteAt) < termNoteWindow {
		badge += chrome.DimText.Render(p.note)
	}
	b.WriteString(badge)

	p.cached = b.String()
	return p.cached
}

// ---------------------------------------------------------------------------
// Grid painting: Cells → lipgloss runs. Contiguous same-styled cells join
// into one Render call so a full row costs a handful of SGR sequences.
// ---------------------------------------------------------------------------

// cellKey is the comparable style signature of one cell (run breaks on ≠).
type cellKey struct {
	fg, bg                string
	bold, dim, under, rev bool
}

// gridRowString renders one screen row. caretCol >= 0 forces reverse video
// on that cell (the block caret — drawn here, never mutating the grid).
func gridRowString(row term.Row, caretCol int) string {
	return gridRowStringSel(row, caretCol, -1, -1)
}

// gridRowStringSel renders one screen row with a selection span: cells
// [selFrom, selTo) get reverse video flipped — the SAME mechanism as the
// block caret, so the highlight composes per-cell with the row's own SGR
// styling (a styled cell inverts, it never loses its colors to a blanket
// overlay). selFrom < 0 means no selection on this row.
func gridRowStringSel(row term.Row, caretCol, selFrom, selTo int) string {
	var b strings.Builder
	var run strings.Builder
	var key cellKey
	var style lipgloss.Style
	started := false

	flush := func() {
		if run.Len() == 0 {
			return
		}
		if key == (cellKey{}) {
			b.WriteString(run.String()) // plain default run, no escapes
		} else {
			b.WriteString(style.Render(run.String()))
		}
		run.Reset()
	}

	for x, c := range row {
		rev := c.Reverse
		if x == caretCol {
			rev = !rev
		}
		if selFrom >= 0 && x >= selFrom && x < selTo {
			rev = !rev
		}
		k := cellKey{
			fg:    term.LipColor(c.Fg),
			bg:    term.LipColor(c.Bg),
			bold:  c.Bold,
			dim:   c.Dim,
			under: c.Underline,
			rev:   rev,
		}
		if started && k != key {
			flush()
		}
		if !started || k != key {
			key = k
			style = lipgloss.NewStyle()
			if k.fg != "" {
				style = style.Foreground(lipgloss.Color(k.fg))
			}
			if k.bg != "" {
				style = style.Background(lipgloss.Color(k.bg))
			}
			style = style.Bold(k.bold).Faint(k.dim).Underline(k.under).Reverse(k.rev)
			started = true
		}
		ch := c.Ch
		if ch == 0 {
			ch = ' '
		}
		run.WriteRune(ch)
	}
	flush()
	return b.String()
}
