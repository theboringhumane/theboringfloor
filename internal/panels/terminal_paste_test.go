// terminal_paste_test.go — the terminal tab's bracketed-paste contract
// (terminal.go's tea.PasteMsg arm + pasteToPTY):
//
//	(a) a paste on the MAIN screen writes to the PTY WRAPPED in
//	    ESC[200~ … ESC[201~ — ONE marker pair around the WHOLE content,
//	    newlines included, so readline/zle/fish (?2004 negotiated at
//	    the prompt) treat it as one paste unit (a multi-line clipboard
//	    never auto-EXECs line-by-line);
//	(b) inside an ALT-SCREEN app (vim, htop — ?1049, the one private
//	    mode the grid tracks) the bytes go RAW, exactly like the key
//	    path's forwarding discipline;
//	(c) the paste fires ONLY while focused + alive (the app's router
//	    already gates this — a directly-driven panel stays honest);
//	(d) new PTY input retires a live selection (the frozen rule).
package panels

import (
	"bytes"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTermPasteBracketedWrap(t *testing.T) {
	p, f := newSelTestPanel(80, 24)
	p.Focus()
	content := "echo one\necho two\necho three"
	p.Update(tea.PasteMsg{Content: content})
	if len(f.writes) != 1 {
		t.Fatalf("one paste = ONE write, got %d", len(f.writes))
	}
	want := []byte("\x1b[200~" + content + "\x1b[201~")
	if !bytes.Equal(f.writes[0], want) {
		t.Fatalf("main-screen paste = ESC[200~…ESC[201~ wrapped, got %q", f.writes[0])
	}
}

func TestTermPasteAltScreenRaw(t *testing.T) {
	p, f := newSelTestPanel(80, 24)
	p.Focus()
	f.feed("\x1b[?1049h") // vim & co: the app owns the keyboard
	if !f.grid.AltActive() {
		t.Fatal("precondition: ?1049h activates the alt screen")
	}
	p.Update(tea.PasteMsg{Content: ":wq\r"})
	if len(f.writes) != 1 || !bytes.Equal(f.writes[0], []byte(":wq\r")) {
		t.Fatalf("alt-screen paste = RAW bytes (no markers), got %q", f.writes)
	}
}

func TestTermPasteGates(t *testing.T) {
	p, f := newSelTestPanel(80, 24)
	// blurred: no write
	p.Update(tea.PasteMsg{Content: "nope"})
	if len(f.writes) != 0 {
		t.Fatalf("a blurred panel never pastes, got %q", f.writes)
	}
	// dead shell: no write
	p.Focus()
	f.alive = false
	p.Update(tea.PasteMsg{Content: "nope"})
	if len(f.writes) != 0 {
		t.Fatalf("a dead shell never receives paste bytes, got %q", f.writes)
	}
}

func TestTermPasteClearsSelection(t *testing.T) {
	stubClipboard(t, nil)
	p, f := newSelTestPanel(80, 24)
	p.Focus()
	f.feed("$ some shell output\r\n")
	p.Update(selClick(2, 0))
	p.Update(selMotion(10, 0))
	p.Update(selRelease(10, 0))
	if p.sel.state != termSelDone {
		t.Fatalf("precondition: the dragged release leaves a highlight, state=%d", p.sel.state)
	}
	p.Update(tea.PasteMsg{Content: "ls"})
	if p.sel.state != termSelIdle {
		t.Fatalf("new PTY input retires the selection (frozen rule), state=%d", p.sel.state)
	}
}
