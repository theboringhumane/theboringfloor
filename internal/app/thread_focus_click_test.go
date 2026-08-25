// thread_focus_click_test.go — the click-open binding (the thread-row
// click → nested focus pane seam, app side; the pane's render + the
// panel's ThreadRowAt hit-test have their own proofs):
//
//	(a) a press+release on a worker thread's frame row in the main chat
//	    CLOSES the transcript view behind the thread-focus pane and opens
//	    THAT agent's own transcript nested inside it — the CLICKED
//	    thread, not ctrl+f's resolved winner (the live chain would pick
//	    tekton-1's newest activity; skopos-1's header must open
//	    skopos-1);
//	(b) esc leaves the click-opened pane and the office returns
//	    BYTE-IDENTICAL (the press armed + the release cleared the
//	    selection seam underneath, and no inline expansion leaked
//	    through the intercept);
//	(c) a click OFF the thread's frame rows (a plain boss bubble row)
//	    opens NOTHING — the legacy ClickRow seams (fold rows, ↳ diff
//	    sub-rows, floor sprites) are untouched by the intercept.
package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// (a)+(b) — the click opens the clicked thread's nested pane; esc
// returns the parent transcript byte-identical.
func TestThreadClickOpensFocusPane(t *testing.T) {
	m, _ := tfSetup(t)
	pre := m.Frame()
	// skopos-1's collapsed header — ctrl+f's live chain would resolve
	// tekton-1 (newest activity); the CLICK names its own agent.
	row := selRowOf(t, m, "Explore Task — Scan the repo")
	x := selChatX(m, 3)
	m, _ = selUpdate(t, m, selClickAt(x, row)) // the press arms the selection seam
	if m.threadFocus != nil {
		t.Fatal("the press alone must wait for the release verdict — no focus may open yet")
	}
	m, _ = selUpdate(t, m, selUpAt(x, row)) // motionless release replays handleClick
	if m.threadFocus == nil || m.focusThread != "skopos-1" {
		t.Fatalf("clicking skopos-1's thread must open ITS focus pane, got pane=%v name=%q",
			m.threadFocus != nil, m.focusThread)
	}
	if !m.focusDeferredRender {
		t.Fatal("the click-open must arm the main chat's render saver")
	}
	fw := ansi.Strip(m.Frame())
	if !strings.Contains(fw, "Explore Task — Scan the repo · 1 tool call") ||
		!strings.Contains(fw, "Grep SSE, 3 hits") ||
		!strings.Contains(fw, "esc · ctrl+f back to office") {
		t.Fatalf("the pane must carry the CLICKED thread's own transcript + the leave hint:\n%s", fw)
	}
	t.Logf("the click-opened nested pane (%dx%d, ansi-stripped):\n%s", m.width, m.height, fw)
	// (b) esc back to the parent transcript — byte-identical through the
	// click-open path (selection pressed+cleared cleanly, no inline
	// expansion rode along)
	m, _ = selUpdate(t, m, tfEsc())
	if m.threadFocus != nil {
		t.Fatal("esc must close the click-opened focus")
	}
	if m.focusDeferredRender {
		t.Fatal("the close must clear the render saver")
	}
	if post := m.Frame(); post != pre {
		t.Fatalf("esc must return the office BYTE-IDENTICAL through the click-open path\n--- before ---\n%s\n--- after ---\n%s", pre, post)
	}
}

// (c) — clicks off the thread frame rows keep their old fate: no pane,
// no focusThread, transcript untouched.
func TestThreadClickOffThreadOpensNothing(t *testing.T) {
	m, _ := tfSetup(t)
	row := selRowOf(t, m, "tekton-1 is on it.") // a plain boss bubble row
	x := selChatX(m, 3)
	m, _ = selUpdate(t, m, selClickAt(x, row))
	m, _ = selUpdate(t, m, selUpAt(x, row))
	if m.threadFocus != nil || m.focusThread != "" {
		t.Fatalf("a click off the thread's frame rows must open nothing (pane=%v name=%q)",
			m.threadFocus != nil, m.focusThread)
	}
}
