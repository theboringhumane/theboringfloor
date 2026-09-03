// browser_edit_test.go — the pane's two URL gestures:
//
//	e     — the inline URL editor: open/prefill (current URL; empty on
//	        the starter card), the full editing surface (typed insert,
//	        left/right/home/end, backspace/delete, ctrl+w word-erase,
//	        bracketed paste), enter committing through the pane's normal
//	        Open path, esc cancelling back to the previous state, and the
//	        editor's TOTAL key ownership while open (no link cursor, no
//	        history, no reload, no q-to-leave — esc/enter resolve first).
//	O     — open the CURRENT page in the OS browser: the text lane's URL,
//	        the shot's redirect-final URL in shot mode, a file:// page as
//	        a LinkFile target, the failure wording, and the no-page dim
//	        no-op (the cascade seam stays faked — never a real shell-out).
package panels

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/headless"
)

// editKey — the editor surface's special keys (browserKey covers the
// runes + esc/arrows already).
func editKey(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "left":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft})
	case "right":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyRight})
	case "home":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyHome})
	case "end":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnd})
	case "backspace":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace})
	case "delete":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDelete})
	case "ctrl+w":
		return tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl})
	default:
		return browserKey(s)
	}
}

// editRig — a navRig with the exec seam recording (shared with the nav
// suite's map-driven fetches).
func newEditRig(t *testing.T, pages map[string]*Page) *navRig {
	t.Helper()
	return newNavRig(t, pages)
}

// typeRunes feeds printable text into the editor one key at a time.
func typeRunes(b *Browser, s string) {
	for _, r := range s {
		b.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

func TestBrowserEditOpensPrefilled(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
	})
	r.driveOpen(t, "file:///a.html")

	r.b.Update(browserKey("e"))
	if !r.b.editing {
		t.Fatal("`e` must open the inline editor")
	}
	if got := string(r.b.editBuf); got != "file:///a.html" {
		t.Fatalf("the editor prefills the current URL, got %q", got)
	}
	if r.b.editCur != len(r.b.editBuf) {
		t.Fatalf("the caret pins to the end: %d of %d", r.b.editCur, len(r.b.editBuf))
	}
	// the bar row paints the editor: the buffer + the dim hint (the caret
	// is a reversed cell, invisible in the stripped view).
	bar := ansi.Strip(r.b.View())
	if !strings.Contains(bar, "▸ file:///a.html") || !strings.Contains(bar, "enter: open · esc: cancel") {
		t.Fatalf("the editor row rides the location bar:\n%s", bar)
	}
}

func TestBrowserEditOpensEmptyOnStarterCard(t *testing.T) {
	r := newEditRig(t, nil)
	r.b.Update(browserKey("e"))
	if !r.b.editing {
		t.Fatal("`e` on the starter card must open the editor")
	}
	if got := string(r.b.editBuf); got != "" {
		t.Fatalf("the starter card's editor starts EMPTY, got %q", got)
	}
}

func TestBrowserEditSurface(t *testing.T) {
	r := newEditRig(t, nil)
	r.b.Update(browserKey("e"))
	typeRunes(r.b, "htp://x.dev/pp")
	buf := func() string { return string(r.b.editBuf) }

	// left + backspace eats the second p: htp://x.dev/pp → htp://x.dev/p
	r.b.Update(editKey("left"))
	r.b.Update(editKey("backspace"))
	if buf() != "htp://x.dev/p" {
		t.Fatalf("backspace mid-buffer: %q", buf())
	}
	r.b.Update(editKey("home"))
	r.b.Update(editKey("right"))
	r.b.Update(editKey("right"))
	typeRunes(r.b, "t") // htp → http
	if buf() != "http://x.dev/p" {
		t.Fatalf("home/right/insert: %q", buf())
	}
	r.b.Update(editKey("end"))
	typeRunes(r.b, "age")
	if buf() != "http://x.dev/page" {
		t.Fatalf("end + append: %q", buf())
	}
	// delete (forward) at home eats the h.
	r.b.Update(editKey("home"))
	r.b.Update(editKey("delete"))
	if buf() != "ttp://x.dev/page" {
		t.Fatalf("delete forward: %q", buf())
	}
	// ctrl+w (readline's rubout) kills back to WHITESPACE: a tail word
	// goes first…
	r.b.Update(editKey("end"))
	typeRunes(r.b, " extra")
	r.b.Update(editKey("ctrl+w"))
	if buf() != "ttp://x.dev/page " {
		t.Fatalf("ctrl+w kills the word left: %q", buf())
	}
	// …and a whitespace-free URL is ONE word — the rubout kills the whole
	// region through to the caret (readline's exact semantics).
	r.b.Update(editKey("ctrl+w"))
	if buf() != "" {
		t.Fatalf("ctrl+w kills the word AND the whitespace through to the caret: %q", buf())
	}
	// home/end pin the caret's extremes; left/right clamp at the edges.
	r.b.Update(editKey("home"))
	r.b.Update(editKey("left"))
	if r.b.editCur != 0 {
		t.Fatalf("left clamps at 0: %d", r.b.editCur)
	}
	r.b.Update(editKey("end"))
	r.b.Update(editKey("right"))
	if r.b.editCur != len(r.b.editBuf) {
		t.Fatalf("right clamps at the end: %d of %d", r.b.editCur, len(r.b.editBuf))
	}
}

func TestBrowserEditCommitDrivesOpen(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///a.html":    navPage("file:///a.html", "Alpha"),
		"file:///edit.html": navPage("file:///edit.html", "Edited"),
	})
	r.driveOpen(t, "file:///a.html")
	fetchesBefore := len(r.fetches)

	r.b.Update(browserKey("e"))
	typeRunes(r.b, " BAD") // the editor's buffer is free text…
	cmd := r.b.Update(editKey("enter"))
	if cmd == nil {
		t.Fatal("enter on a non-empty buffer must drive the Open path")
	}
	r.b.Update(cmd()) // land the fetch verdict exactly like the app's hop
	// …but Open trims it: the fetch recorded the EDITED url (never the
	// old location, never the stray pad).
	if len(r.fetches) != fetchesBefore+1 || r.fetches[len(r.fetches)-1] != "file:///a.html BAD" {
		t.Fatalf("enter commits the edited buffer through Open: %v", r.fetches)
	}
	if r.b.editing {
		t.Fatal("the commit closes the editor")
	}
	// the failed fetch lands on the attempted URL with the dim error rows
	// (the normal Open path, exactly like /open).
	if r.b.url != "file:///a.html BAD" || !strings.Contains(ansi.Strip(r.b.View()), "404: no route") {
		t.Fatalf("the commit's fetch landed through the normal Open path:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserEditCommitLoadsEditedURL(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///a.html":    navPage("file:///a.html", "Alpha"),
		"file:///edit.html": navPage("file:///edit.html", "Edited"),
	})
	r.driveOpen(t, "file:///a.html")

	r.b.Update(browserKey("e"))
	if got := string(r.b.editBuf); got != "file:///a.html" {
		t.Fatalf("prefill: %q", got)
	}
	for i := 0; i < len("a.html"); i++ { // erase "a.html" from the tail
		r.b.Update(editKey("backspace"))
	}
	typeRunes(r.b, "edit.html")
	if got := string(r.b.editBuf); got != "file:///edit.html" {
		t.Fatalf("the edited buffer: %q", got)
	}
	cmd := r.b.Update(editKey("enter"))
	if cmd == nil {
		t.Fatal("the commit must drive Open")
	}
	r.b.Update(cmd())
	if r.b.page == nil || r.b.page.Title != "Edited" || r.b.url != "file:///edit.html" {
		t.Fatalf("the edited URL loaded: url %q page %+v", r.b.url, r.b.page)
	}
	if got := r.fetches[len(r.fetches)-1]; got != "file:///edit.html" {
		t.Fatalf("Open was invoked with the edited URL: %v", r.fetches)
	}
	// the new page pushed a ring entry (the normal Open path's history).
	if len(r.b.hist) != 2 || r.b.hist[1].url != "file:///edit.html" {
		t.Fatalf("the commit's navigation pushed history: %+v", r.b.hist)
	}
}

func TestBrowserEditCancelRestores(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
	})
	r.driveOpen(t, "file:///a.html")
	fetchesBefore := len(r.fetches)
	before := ansi.Strip(r.b.View())

	r.b.Update(browserKey("e"))
	typeRunes(r.b, "junk")
	r.b.Update(editKey("ctrl+w"))
	if cmd := r.b.Update(browserKey("esc")); cmd != nil {
		t.Fatalf("esc cancels silently (no cmd), got %v", cmd)
	}
	if r.b.editing {
		t.Fatal("esc must close the editor")
	}
	// the previous state is EXACTLY restored: same url, same page, the
	// same rendered frame — and no fetch fired.
	if r.b.url != "file:///a.html" || r.b.page == nil || r.b.page.Title != "Alpha" {
		t.Fatalf("esc restores the pre-edit state: url %q page %+v", r.b.url, r.b.page)
	}
	if len(r.fetches) != fetchesBefore {
		t.Fatalf("a cancel never fetches: %v", r.fetches)
	}
	if after := ansi.Strip(r.b.View()); after != before {
		t.Fatalf("the cancel restores the frame byte-for-byte:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
}

func TestBrowserEditOwnsKeys(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///index.html": navPage("file:///index.html", "Index",
			BrowserLink{N: 1, URL: "file:///tmp/a.txt", Text: "row one"},
			BrowserLink{N: 2, URL: "file:///tmp/b.txt", Text: "row two"}),
	})
	r.driveOpen(t, "file:///index.html")
	r.b.Update(browserKey("e"))
	cursor, yoff, histLen := r.b.cursor, r.b.vp.YOffset(), len(r.b.hist)

	// the pane's own verbs become TEXT (j/o/q/r/[/]/e/O insert — the
	// editor owns them), the scroll keys swallow (pgup/pgdn move nothing),
	// and esc resolves the EDITOR — never the leave msg.
	for _, k := range []string{"j", "o", "q", "r", "[", "]", "e", "O"} {
		if cmd := r.b.Update(browserKey(k)); cmd != nil {
			t.Fatalf("%q while editing must produce no cmd", k)
		}
	}
	if got := string(r.b.editBuf); got != "file:///index.htmljoqr[]eO" {
		t.Fatalf("the pane's verbs type into the buffer: %q", got)
	}
	if r.b.cursor != cursor || len(r.b.hist) != histLen || len(r.opened) != 0 {
		t.Fatalf("no link nav / history hop / OS-open while editing: cursor %d hist %d opened %v",
			r.b.cursor, len(r.b.hist), r.opened)
	}
	for _, k := range []string{"pgup", "pgdown", "up", "down"} {
		if cmd := r.b.Update(browserKey(k)); cmd != nil {
			t.Fatalf("%q while editing must produce no cmd", k)
		}
	}
	if r.b.vp.YOffset() != yoff {
		t.Fatalf("the scroll keys swallow while editing: yoff %d → %d", yoff, r.b.vp.YOffset())
	}
	// esc is the EDITOR's cancel, not the pane's leave.
	if cmd := r.b.Update(browserKey("esc")); cmd != nil {
		if _, isLeave := cmd().(BrowserLeaveMsg); isLeave {
			t.Fatal("esc while editing must cancel the editor, never leave the tab")
		}
	}
	if r.b.editing {
		t.Fatal("the esc resolved the editor")
	}
	// …and afterwards the pane's q/esc leave works again.
	cmd := r.b.Update(browserKey("esc"))
	if cmd == nil {
		t.Fatal("esc post-edit leaves the tab again")
	}
	if _, ok := cmd().(BrowserLeaveMsg); !ok {
		t.Fatalf("esc post-edit must ride BrowserLeaveMsg, got %T", cmd())
	}
}

func TestBrowserEditPaste(t *testing.T) {
	r := newEditRig(t, nil)
	r.b.Update(browserKey("e"))
	r.b.Update(tea.PasteMsg{Content: "https://a.dev/x\r\n"})
	if got := string(r.b.editBuf); got != "https://a.dev/x" {
		t.Fatalf("the paste splices at the caret (newlines stripped): %q", got)
	}
	// a paste while NOT editing never reaches the buffer.
	r.b.Update(browserKey("esc"))
	r.b.Update(tea.PasteMsg{Content: "ignored"})
	if r.b.editing || len(r.b.editBuf) != 0 {
		t.Fatalf("a stray paste outside the editor is dropped")
	}
}

func TestBrowserEditEnterEmptyIsANoop(t *testing.T) {
	r := newEditRig(t, nil)
	fetchesBefore := len(r.fetches)
	r.b.Update(browserKey("e"))
	if cmd := r.b.Update(editKey("enter")); cmd != nil {
		t.Fatal("enter on an empty buffer is a silent no-op (never an Open)")
	}
	if r.b.editing {
		t.Fatal("the empty enter still closes the editor")
	}
	if len(r.fetches) != fetchesBefore {
		t.Fatalf("no fetch fired: %v", r.fetches)
	}
}

// ---------------------------------------------------------------------------
// `O` — open the current page in the OS browser
// ---------------------------------------------------------------------------

func TestBrowserOSOpenTextLane(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"http://localhost:9/docs": navPage("http://localhost:9/docs", "Docs"),
	})
	r.driveOpen(t, "http://localhost:9/docs")

	cmd := r.b.Update(browserKey("O"))
	if cmd == nil {
		t.Fatal("O on a loaded page must fire the exec seam")
	}
	r.b.Update(cmd()) // the app forwards the verdict
	if len(r.opened) != 1 {
		t.Fatalf("the cascade saw %d targets, want 1", len(r.opened))
	}
	got := r.opened[0]
	if got.Kind != LinkURL || got.Value != "http://localhost:9/docs" {
		t.Fatalf("O opens the CURRENT page: %+v", got)
	}
	if !strings.Contains(ansi.Strip(r.b.View()), "opened in system browser: http://localhost:9/docs") {
		t.Fatalf("the dim confirmation row lands:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserOSOpenFilePageIsAFileTarget(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"file:///tmp/notes.html": navPage("file:///tmp/notes.html", "Notes"),
	})
	r.driveOpen(t, "file:///tmp/notes.html")
	cmd := r.b.Update(browserKey("O"))
	if cmd == nil {
		t.Fatal("O on a file:// page must fire")
	}
	r.b.Update(cmd())
	if len(r.opened) != 1 || r.opened[0].Kind != LinkFile || r.opened[0].Value != "/tmp/notes.html" {
		t.Fatalf("a file:// page rides as a LinkFile target: %+v", r.opened)
	}
	if !strings.Contains(ansi.Strip(r.b.View()), "opened in system browser: /tmp/notes.html") {
		t.Fatalf("the confirmation names the path:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserOSOpenShotModeUsesShotURL(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x-final", Title: "Xray Final", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	if !r.b.ShotActive() {
		t.Fatal("setup: shot mode paints")
	}
	var opened []LinkTarget
	r.b.openFn = func(tgt LinkTarget) error { opened = append(opened, tgt); return nil }

	cmd := r.b.Update(browserKey("O"))
	if cmd == nil {
		t.Fatal("O in shot mode must fire")
	}
	r.b.Update(cmd())
	if len(opened) != 1 || opened[0].Value != "https://a.dev/x-final" {
		t.Fatalf("O opens the shot's redirect-final URL: %+v", opened)
	}
	if !strings.Contains(ansi.Strip(r.b.View()), "opened in system browser: https://a.dev/x-final") {
		t.Fatalf("the confirmation rides the shot's note row:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserOSOpenNoPage(t *testing.T) {
	r := newEditRig(t, nil)
	if cmd := r.b.Update(browserKey("O")); cmd != nil {
		t.Fatalf("O with no page loaded is a no-op (no exec), got %v", cmd)
	}
	if len(r.opened) != 0 {
		t.Fatalf("the seam never fired: %+v", r.opened)
	}
	if !strings.Contains(ansi.Strip(r.b.View()), "nothing to open — load a page first") {
		t.Fatalf("the dim no-op note lands:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserOSOpenFailureNote(t *testing.T) {
	r := newEditRig(t, map[string]*Page{
		"http://localhost:9/docs": navPage("http://localhost:9/docs", "Docs"),
	})
	r.openErr = errors.New("no open/xdg-open tool on this host")
	r.driveOpen(t, "http://localhost:9/docs")
	cmd := r.b.Update(browserKey("O"))
	if cmd == nil {
		t.Fatal("O must fire even when the cascade will fail")
	}
	r.b.Update(cmd())
	if !strings.Contains(ansi.Strip(r.b.View()), "could not open: no open/xdg-open tool on this host") {
		t.Fatalf("the cascade's error surfaces as the dim row:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserEditInShotMode(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x-final", Title: "Xray Final", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	r.b.Update(browserKey("e"))
	if !r.b.editing {
		t.Fatal("`e` opens the editor in shot mode too")
	}
	if got := string(r.b.editBuf); got != "https://a.dev/x-final" {
		t.Fatalf("the shot's redirect-final URL prefills: %q", got)
	}
	// the strip row wears the editor while the shot region keeps painting.
	view := ansi.Strip(r.b.View())
	if !strings.Contains(view, "▸ https://a.dev/x-final") || !strings.Contains(view, "enter: open · esc: cancel") {
		t.Fatalf("the editor takes the shot strip row:\n%s", view)
	}
	if strings.Contains(view, "headless chromium") {
		t.Fatalf("the editor replaces the strip (one location bar only):\n%s", view)
	}
	if !r.b.ShotActive() {
		t.Fatal("the PNG stays published under the editor")
	}
}

func TestBrowserEditNeverOpensWhilePremium(t *testing.T) {
	// the premium lane owns its key surface: `e` forwards to the child
	// (the controller's Write path), never opening the text-lane editor.
	pinKittyEnv(t)
	fakeSpawnPins(t)
	b := NewBrowser()
	b.SetSize(64, 16)
	b.fetchFn = func(string) (*Page, error) { return navPage("https://a.dev/x", "Xray"), nil }
	cmd := b.Open("https://a.dev/x")
	if cmd == nil {
		t.Fatal("Open must produce the fetch cmd")
	}
	b.Update(cmd())
	if !b.PremiumActive() {
		t.Fatal("setup: the premium embed is live")
	}
	b.Update(browserKey("e"))
	if b.editing {
		t.Fatal("`e` on the premium lane forwards to the child — the text editor stays closed")
	}
}
