// terminal_sel_test.go — the terminal panel's MOUSE TEXT SELECTION
// contract (terminal.go's header is the design doc). Driven at the panel's
// own Update seam with synthetic tea.Mouse messages — the same convention
// gitpanel_test.go:488 pins (sidebar-box space: the panel subtracts
// Tabs.ContentOffset itself).
//
// Pinned rules:
//
//	(a) a left press arms, drag motion extends, a dragged release EXTRACTS
//	    viewport text and copies it ONCE through the clipboard seam —
//	    note = " · Copied N chars" (runes, joining newlines included);
//	(b) the copy works from BOTH focus states: the press itself captures
//	    (legacy click-focuses survives untouched);
//	(c) the highlight renders as reverse video on the LIVE grid and as the
//	    chat panel's selHighlight splice on the scrollback window;
//	(d) the selection reaches GRID + SCROLLBACK: a scrolled view maps
//	    press cells through the scroll offset, and wheel-scroll mid-
//	    selection shifts the endpoints with the scroll delta (pinned to
//	    the words, edge-clamped);
//	(e) a motionless release is the plain click — no copy, no selection;
//	(f) clearing: new PTY input retires the selection; esc owns it first
//	    (never reaches the shell — a second esc does); respawn clears;
//	(g) a press outside the body rows clears (webpage rule); a second
//	    release copies NOTHING again (the copy fires exactly once);
//	(h) a failed copy (no clipboard tool) swaps to the dim failure note —
//	    never silent, never a panic; the OSC52 cmd still returns;
//	(i) the copy note rides the badge for termNoteWindow and expires on
//	    the SetState refresh clock (no timers).
//
// DISCIPLINE: no real PTY (fake termSess over a REAL Grid + REAL
// Scrollback), no sleeps, and the clipboard seam is STUBBED before any
// Update runs (tests never exec pbcopy — clipboard.go's contract).
package panels

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
	"github.com/theboringhumane/theboringoffice/internal/term"
)

// errFakeSpawn — the spawn seam's canned failure (respawn tests).
var errFakeSpawn = errors.New("fake spawn (test)")

// stZero — an empty office state for SetState pushes.
var stZero = state.OfficeState{}

// fakeTermSess — a termSess double: real Grid + real Scrollback (the panel
// exercises the TRUE cell/line models), deterministic lifecycle flags, and
// a write ledger. No PTY, no goroutines, no sleeps.
type fakeTermSess struct {
	grid   *term.Grid
	sb     *term.Scrollback
	alive  bool
	code   int
	writes [][]byte
	sizes  [][2]int
	closed bool
}

func newFakeTermSess(cols, rows int) *fakeTermSess {
	return &fakeTermSess{
		grid:  term.NewGrid(cols, rows),
		sb:    term.NewScrollback(1 << 20),
		alive: true,
		code:  -1,
	}
}

func (f *fakeTermSess) Alive() bool                  { return f.alive }
func (f *fakeTermSess) Close() error                 { f.closed = true; f.alive = false; return nil }
func (f *fakeTermSess) ExitCode() int                { return f.code }
func (f *fakeTermSess) Grid() *term.Grid             { return f.grid }
func (f *fakeTermSess) Scrollback() *term.Scrollback { return f.sb }
func (f *fakeTermSess) Write(b []byte) (int, error) {
	cp := make([]byte, len(b))
	copy(cp, b)
	f.writes = append(f.writes, cp)
	return len(b), nil
}
func (f *fakeTermSess) Resize(cols, rows int) error {
	f.sizes = append(f.sizes, [2]int{cols, rows})
	f.grid.SetSize(cols, rows)
	return nil
}
func (f *fakeTermSess) Size() (int, int) { return f.grid.Cols(), f.grid.Rows() }

// feed pushes bytes into the fake's grid AND scrollback — exactly what the
// real session reader loop's MultiWriter does.
func (f *fakeTermSess) feed(s string) {
	_, _ = f.grid.Write([]byte(s))
	_, _ = f.sb.Write([]byte(s))
}

// newSelTestPanel builds a TermPanel over a fake session (never spawns):
// SetSize geos the panel AND the fake PTY.
func newSelTestPanel(w, h int) (*TermPanel, *fakeTermSess) {
	f := newFakeTermSess(w, h-1)
	p := &TermPanel{sess: f}
	p.SetSize(w, h)
	return p, f
}

// stubClipboard installs a recording clipboard seam for the test's
// duration and returns a pointer to the ledger (calls[i] = copied text).
// errNote forces a failure when non-nil. NEVER a real exec.
func stubClipboard(t *testing.T, errNote error) *[]string {
	t.Helper()
	calls := &[]string{}
	prev := clipboardCopyText
	clipboardCopyText = func(text string) error {
		*calls = append(*calls, text)
		return errNote
	}
	t.Cleanup(func() { clipboardCopyText = prev })
	return calls
}

// selPt builds a mouse message at PANEL coordinates (col/row of the body)
// shifted into sidebar-box space — the gitpanel_test.go:488 convention:
// the app hands msgs in box space and the panel subtracts ContentOffset.
func selPt(col, row int) (x, y int) {
	dx, dy := (&Tabs{}).ContentOffset()
	return dx + col, dy + row
}

func selClick(col, row int) tea.MouseClickMsg {
	x, y := selPt(col, row)
	return tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

func selMotion(col, row int) tea.MouseMotionMsg {
	x, y := selPt(col, row)
	return tea.MouseMotionMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

func selRelease(col, row int) tea.MouseReleaseMsg {
	x, y := selPt(col, row)
	return tea.MouseReleaseMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft})
}

func selKey(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "r":
		return tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"})
	case "a":
		return tea.KeyPressMsg(tea.Key{Code: 'a', Text: "a"})
	}
	panic("selKey: unmapped " + s)
}

// selDrag drives a full press → motion → release drag and returns the
// release's cmd.
func selDrag(p *TermPanel, fromCol, fromRow, toCol, toRow int) tea.Cmd {
	p.Update(selClick(fromCol, fromRow))
	p.Update(selMotion(toCol, toRow))
	return p.Update(selRelease(toCol, toRow))
}

// ---------------------------------------------------------------------------
// drag → copy → note (the headline contract, live grid)
// ---------------------------------------------------------------------------

func TestTermSelectDragCopiesOnRelease(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8) // body rows = 7
	p.Focus()
	f.feed("alpha\r\nbeta gamma\r\n$ ")

	cmd := selDrag(p, 2, 0, 9, 1)

	if cmd == nil {
		t.Fatalf("a dragged release must return the OSC52 fallback cmd")
	}
	want := "pha\nbeta gamma" // row0 cols 2.., row1 cols 0..9 (release cell included)
	if len(*calls) != 1 {
		t.Fatalf("copy must fire exactly once per dragged release, got %d", len(*calls))
	}
	if got := (*calls)[0]; got != want {
		t.Fatalf("copied text = %q, want %q", got, want)
	}
	if p.sel.state != termSelDone {
		t.Fatalf("a dragged release must finalize the selection, state = %d", p.sel.state)
	}
	if p.note != " · Copied 14 chars" {
		t.Fatalf("copy note = %q, want the frozen %q", p.note, " · Copied 14 chars")
	}
	out := p.View()
	// start row inverts to the LINE END (the chat panel's full-row-bar
	// rule); the pad highlights but never leaks into the copy
	if !strings.Contains(out, "\x1b[7mpha               \x1b[m") {
		t.Fatalf("the live-grid highlight must reverse the start row to its end, got:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[7mbeta gamma\x1b[m") {
		t.Fatalf("the end row's span must reverse exactly, got:\n%q", out)
	}
	if !strings.Contains(out, " · Copied 14 chars") {
		t.Fatalf("the note must ride the badge row, got:\n%q", out)
	}
}

func TestTermSelectBlurredPressCapturesAndArms(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	if p.Focused() {
		t.Fatalf("precondition: a fresh panel starts RELEASED")
	}
	f.feed("one two\r\nthree four\r\n")

	_ = selDrag(p, 0, 0, 9, 0)

	if !p.Focused() {
		t.Fatalf("the press must keep the legacy click-focuses behavior (the arm rides along)")
	}
	if len(*calls) != 1 || (*calls)[0] != "one two" {
		t.Fatalf("blurred-start drag copy = %v, want [\"one two\"]", *calls)
	}
}

func TestTermSelectPlainClickIsLegacyFocus(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, _ := newSelTestPanel(20, 8)
	if p.Focused() {
		t.Fatalf("precondition: fresh panel released")
	}
	p.Update(selClick(3, 2))
	cmd := p.Update(selRelease(3, 2)) // NO motion — the plain click

	if cmd != nil {
		t.Fatalf("a motionless release must not return a copy cmd")
	}
	if !p.Focused() {
		t.Fatalf("a plain click must still focus (capture) the panel")
	}
	if p.sel.state != termSelIdle {
		t.Fatalf("a motionless release retires the arm silently, state = %d", p.sel.state)
	}
	if len(*calls) != 0 {
		t.Fatalf("a plain click must never copy, got %v", *calls)
	}
	if p.note != "" {
		t.Fatalf("a plain click must not arm a copy note, got %q", p.note)
	}
}

// ---------------------------------------------------------------------------
// scrollback mapping + scroll-shift pinning
// ---------------------------------------------------------------------------

func TestTermSelectCoversScrollbackWindow(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	var hist strings.Builder
	for i := 0; i < 30; i++ {
		hist.WriteString("L" + itoa(100+i) + " line\r\n")
	}
	f.feed(hist.String())
	// wheels climb into history: the window ends `scroll` rows above the tail
	p.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	p.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	if p.scroll != 2 {
		t.Fatalf("precondition: two wheels up ⇒ scroll = 2, got %d", p.scroll)
	}

	_ = selDrag(p, 0, 0, 4, 1)

	// Render(bodyH+scroll=9, w) takes the LAST 9 retained lines (the feed's
	// trailing newline's empty row is dropped): L121..L129 — the history
	// view drops the last `scroll` of those ⇒ shown: L121..L127.
	want := "L121 line\nL122" // row0 full line; row1 cols 0..4 → "L122 "→trim
	if len(*calls) != 1 {
		t.Fatalf("scrolled drag must copy once, got %d", len(*calls))
	}
	if got := (*calls)[0]; got != want {
		t.Fatalf("scrolled drag copy = %q, want %q", got, want)
	}
	out := p.View()
	if !strings.Contains(out, "\x1b[7m") {
		t.Fatalf("the scrollback highlight must splice reverse video, got:\n%q", out)
	}
	if !strings.Contains(out, selRevOff) {
		t.Fatalf("the history splice uses the chat panel's selHighlight, got:\n%q", out)
	}
	if !strings.Contains(out, " · scrolled up 2") {
		t.Fatalf("the scroll badge must survive beside the note, got:\n%q", out)
	}
}

func TestTermSelectScrollShiftsEndpointsPinned(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	var hist strings.Builder
	for i := 0; i < 20; i++ {
		hist.WriteString("row" + itoa(200+i) + "\r\n")
	}
	f.feed(hist.String())

	// arm a one-row selection on viewport row 5 (live grid tail line)
	p.Update(selClick(0, 5))
	p.Update(selMotion(4, 5))
	if p.sel.a.row != 5 {
		t.Fatalf("precondition: anchor row = 5, got %d", p.sel.a.row)
	}
	// wheel UP (into history): scroll grows by 1, content shifts DOWN —
	// text-pinned endpoints move +1 viewport row (header rule)
	p.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	if p.sel.a.row != 6 || p.sel.h.row != 6 {
		t.Fatalf("scroll-shift must move endpoints +1 (pinned to words), a=%d h=%d", p.sel.a.row, p.sel.h.row)
	}
	if p.sel.state != termSelArmed {
		t.Fatalf("wheel scroll must NOT clear an armed selection, state = %d", p.sel.state)
	}
	cmd := p.Update(selRelease(6, 6))
	if cmd == nil {
		t.Fatalf("release after a scroll-shift still copies the drag")
	}
	if len(*calls) != 1 {
		t.Fatalf("copy fired %d times, want 1", len(*calls))
	}
}

// ---------------------------------------------------------------------------
// copy-once gate + webpage clear rule + dead-shell no-arm
// ---------------------------------------------------------------------------

func TestTermSelectCopiesExactlyOnce(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\nbeta\r\n")

	_ = selDrag(p, 0, 0, 3, 0)
	p.Update(selRelease(3, 0)) // duplicate release — no arm, no copy
	p.Update(selRelease(0, 0)) // another stray release
	if len(*calls) != 1 {
		t.Fatalf("stray releases must never re-copy, got %d copies", len(*calls))
	}
}

func TestTermSelectOffBodyPressClears(t *testing.T) {
	_ = stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\nbeta\r\n")
	_ = selDrag(p, 0, 0, 3, 0)
	if p.sel.state != termSelDone {
		t.Fatalf("precondition: selection finalized")
	}
	// a press on the BADGE row (cy == bodyH) is outside the body: webpage rule
	bx, by := selPt(2, p.bodyH())
	p.Update(tea.MouseClickMsg(tea.Mouse{X: bx, Y: by, Button: tea.MouseLeft}))
	if p.sel.state != termSelIdle {
		t.Fatalf("an off-body press must clear the highlight, state = %d", p.sel.state)
	}
}

func TestTermSelectDeadShellDoesNotArm(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\n")
	f.alive = false // died with content on screen

	p.Update(selClick(0, 0))
	p.Update(selMotion(3, 0))
	p.Update(selRelease(3, 0))
	if p.sel.state != termSelIdle {
		t.Fatalf("a dead shell must not arm a selection, state = %d", p.sel.state)
	}
	if len(*calls) != 0 {
		t.Fatalf("a dead shell must never copy, got %v", *calls)
	}
}

// ---------------------------------------------------------------------------
// clearing rules: PTY input / esc-first / respawn / clipboard failure
// ---------------------------------------------------------------------------

func TestTermSelectClearsOnPTYInput(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\nbeta\r\n")
	_ = selDrag(p, 0, 0, 3, 0)
	if p.sel.state != termSelDone {
		t.Fatalf("precondition: selection finalized")
	}

	p.Update(selKey("a")) // focused keystroke → bytes to the PTY
	if len(f.writes) == 0 {
		t.Fatalf("precondition: the keystroke must reach the PTY")
	}
	if p.sel.state != termSelIdle {
		t.Fatalf("new input to the PTY must retire the selection, state = %d", p.sel.state)
	}
	p.Blur() // blurred so the CARET's own reverse flip can't muddy the pin
	if out := p.View(); strings.Contains(out, "\x1b[7m") {
		t.Fatalf("the highlight must vanish after PTY input, got:\n%q", out)
	}
	if len(*calls) != 1 {
		t.Fatalf("the earlier copy ledger must stand untouched at 1, got %d", len(*calls))
	}
}

func TestTermSelectEscOwnsTheHighlight(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\nbeta\r\n")

	// armed: a single esc cancels (no copy even if a release follows)
	p.Update(selClick(0, 0))
	p.Update(selMotion(3, 0))
	p.Update(selKey("esc"))
	if p.sel.state != termSelIdle {
		t.Fatalf("esc must cancel an armed selection, state = %d", p.sel.state)
	}
	p.Update(selRelease(3, 0))
	if len(*calls) != 0 {
		t.Fatalf("an esc-cancelled drag must never copy, got %v", *calls)
	}

	// finalized: esc clears the highlight; the NEXT esc reaches the PTY
	_ = selDrag(p, 0, 0, 3, 0)
	writesBefore := len(f.writes)
	p.Update(selKey("esc"))
	if p.sel.state != termSelIdle {
		t.Fatalf("esc must clear a finalized selection, state = %d", p.sel.state)
	}
	if len(f.writes) != writesBefore {
		t.Fatalf("the selection's esc must never reach the PTY")
	}
	p.Update(selKey("esc"))
	if len(f.writes) != writesBefore+1 {
		t.Fatalf("a bare esc must reach the PTY once the highlight is gone")
	}
	if got := f.writes[len(f.writes)-1]; len(got) != 1 || got[0] != 0x1b {
		t.Fatalf("the PTY esc byte must be 0x1b, got %v", got)
	}
}

func TestTermSelectRespawnClears(t *testing.T) {
	_ = stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\n")
	_ = selDrag(p, 0, 0, 3, 0)
	if p.sel.state != termSelDone {
		t.Fatalf("precondition: selection finalized")
	}

	// route respawn through the spawn seam — never a real PTY in tests
	f.alive = false
	var spawned bool
	prev := spawnTermSession
	spawnTermSession = func(cfg term.TermConfig) (*term.Session, error) {
		spawned = true
		return nil, errFakeSpawn
	}
	t.Cleanup(func() { spawnTermSession = prev })

	p.Update(selKey("r"))
	if !spawned {
		t.Fatalf("r on a dead shell must route through the spawn seam")
	}
	if p.sel.state != termSelIdle {
		t.Fatalf("respawn must clear the selection, state = %d", p.sel.state)
	}
	if p.spawnErr == nil {
		t.Fatalf("the fake spawn error must land on spawnErr")
	}
}

func TestTermSelectNoClipboardToolDimsNote(t *testing.T) {
	calls := stubClipboard(t, errNoClipboardTool)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\nbeta\r\n")

	cmd := selDrag(p, 0, 0, 3, 1)
	if cmd == nil {
		t.Fatalf("the OSC52 fallback cmd rides along even for a failed copy")
	}
	if len(*calls) != 1 {
		t.Fatalf("the copy seam must be invoked once, got %d", len(*calls))
	}
	if p.note != " · copy failed (no clipboard tool)" {
		t.Fatalf("degraded note = %q, want the frozen dim copy", p.note)
	}
	out := p.View()
	if !strings.Contains(out, " · copy failed (no clipboard tool)") {
		t.Fatalf("the failure note must ride the badge row, got:\n%q", out)
	}
	if p.sel.state != termSelDone {
		t.Fatalf("a failed copy keeps the highlight up (retry lives), state = %d", p.sel.state)
	}
}

// ---------------------------------------------------------------------------
// note expiry + cache discipline via the SetState refresh clock
// ---------------------------------------------------------------------------

func TestTermSelectCopyNoteExpiresOnTheTick(t *testing.T) {
	_ = stubClipboard(t, nil)
	p, f := newSelTestPanel(20, 8)
	p.Focus()
	f.feed("alpha\r\n")
	_ = selDrag(p, 0, 0, 3, 0) // [0,4) on "alpha" → "alph", N=4
	if !strings.Contains(p.View(), " · Copied 4 chars") {
		t.Fatalf("precondition: the fresh note renders, got:\n%q", p.View())
	}

	// while fresh, SetState invalidates the frame cache so expiry lands
	p.cached = "STALE"
	p.SetState(stZero)
	if p.cached != "" {
		t.Fatalf("a fresh note must force a rebuild on the refresh clock")
	}

	// age the note past the window (no sleeps — direct timestamp travel)
	p.noteAt = time.Now().Add(-3 * time.Second)
	out := p.View()
	if strings.Contains(out, " · Copied 4 chars") {
		t.Fatalf("the note must vanish at expiry, got:\n%q", out)
	}
	// an expired note must stop forcing rebuilds (battery rule): re-sync
	// rev so the push below has no OTHER reason to invalidate
	p.rev = f.grid.Rev() + f.sb.Rev()
	p.cached = "STALE"
	p.SetState(stZero)
	if p.cached == "" {
		t.Fatalf("an expired note must stop forcing rebuilds (battery rule)")
	}
}

// ---------------------------------------------------------------------------
// coordinate-space + renderer pins
// ---------------------------------------------------------------------------

func TestTermSelectContentOffsetConvention(t *testing.T) {
	_ = stubClipboard(t, nil)
	p, _ := newSelTestPanel(20, 8)
	dx, dy := (&Tabs{}).ContentOffset()
	click := tea.MouseClickMsg(tea.Mouse{X: dx + 3, Y: dy + 2, Button: tea.MouseLeft})
	p.Update(click)
	if p.sel.a.col != 3 || p.sel.a.row != 2 {
		t.Fatalf("box-space press must land on body cell (3,2), got (%d,%d)", p.sel.a.col, p.sel.a.row)
	}
}

func TestGridRowStringSelFlipsExactCells(t *testing.T) {
	row := term.Row{
		{Ch: 'a', Fg: term.ColorDefault, Bg: term.ColorDefault},
		{Ch: 'b', Fg: term.ColorDefault, Bg: term.ColorDefault},
		{Ch: 'c', Fg: term.ColorDefault, Bg: term.ColorDefault},
		{Ch: 'd', Fg: term.ColorDefault, Bg: term.ColorDefault},
		{Ch: 'e', Fg: term.ColorDefault, Bg: term.ColorDefault},
	}
	out := gridRowStringSel(row, -1, 1, 4)
	if !strings.Contains(out, "\x1b[7mbcd\x1b[m") {
		t.Fatalf("the span must reverse exactly bcd, got %q", out)
	}
	if !strings.HasPrefix(out, "a") || !strings.HasSuffix(out, "e") {
		t.Fatalf("cells outside the span stay plain, got %q", out)
	}
	// caret + selection compose per-cell (the caret on a selected cell
	// un-flips it, the rest of the span stays reversed)
	out2 := gridRowStringSel(row, 2, 1, 4)
	if !strings.Contains(out2, "\x1b[7mb\x1b[m") || !strings.Contains(out2, "\x1b[7md\x1b[m") {
		t.Fatalf("caret-in-span must split the reversed runs, got %q", out2)
	}
	if strings.Contains(out2, "\x1b[7mbcd\x1b[m") {
		t.Fatalf("the caret cell must NOT render selected, got %q", out2)
	}
}

// ---------------------------------------------------------------------------
// multi-row shape: blank interior rows keep their newline; soft-wrap joins
// ---------------------------------------------------------------------------

func TestTermSelectMultiRowShapeAndSoftWrap(t *testing.T) {
	calls := stubClipboard(t, nil)
	p, f := newSelTestPanel(10, 8)
	p.Focus()
	// soft-wrap: a line longer than the grid wraps into CONTINUATION rows —
	// a selection across them joins with "\n" (documented v1 shape), and a
	// blank row between stays a newline segment (webpage semantics).
	f.feed("abcdefghijKL\r\n\r\nmid\r\n")
	_ = selDrag(p, 2, 0, 4, 3)
	want := "cdefghij\nKL\n\nmid"
	if len(*calls) != 1 {
		t.Fatalf("copy must fire once, got %d", len(*calls))
	}
	if got := (*calls)[0]; got != want {
		t.Fatalf("multi-row soft-wrap copy = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// PROOF — a full simulated frame: a multi-line selection over a live PTY
// region + the copy-confirmed badge note, printed verbatim for the manager.
// ---------------------------------------------------------------------------

func TestTermSelectFrameProof(t *testing.T) {
	stubClipboard(t, nil)
	p, f := newSelTestPanel(24, 9)
	p.Focus()
	f.feed("$ git status\r\nOn branch main\r\nmodified: internal/panels/terminal.go\r\nnothing else\r\n$ ")
	cmd := selDrag(p, 3, 1, 24, 2)
	if cmd == nil {
		t.Fatalf("release must return the OSC52 cmd")
	}
	t.Logf("\n--- simulated frame (drag (3,1)→(23,2) over a live PTY region) ---\n%s\n--- end frame ---", p.View())

	// and the scrollback-twin frame: scroll up and reselect from history
	p2, f2 := newSelTestPanel(24, 9)
	p2.Focus()
	var hist strings.Builder
	for i := 0; i < 14; i++ {
		hist.WriteString("build step " + itoa(i+1) + " ✓\r\n")
	}
	f2.feed(hist.String())
	p2.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	p2.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	p2.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	_ = selDrag(p2, 0, 1, 8, 2)
	t.Logf("\n--- simulated frame (history path, selection spliced with selHighlight) ---\n%s\n--- end frame ---", p2.View())
}
