// chat_fold_test.go — behavior proofs for the USER-BUBBLE FOLD
// (chat.go's userFoldVisible/userFoldRows/ToggleUserFold) and the
// transcript inset (chatPadL/chatPadR/contentW/setConversation):
//
//	(a) a user turn of userFoldVisible rows or less renders EXACTLY the
//	    pre-feature shape — no hint row, no userFoldRows registration;
//	(b) a longer turn FOLDS to its first userFoldVisible body rows +
//	    the one-row "… +N more lines · click to expand" hint at the
//	    bubble's hanging indent, registered in userFoldRows at its exact
//	    content row — alone, AND mid-stream after another item;
//	(c) a ClickRow on the hint row EXPANDS (all body rows + the
//	    "… collapse" trailer, registered too), a click on the trailer
//	    folds back — and ToggleUserFold's forceRender changes the
//	    rendered pixels WITHOUT any SetState call;
//	(d) the " · 📎 N" suffix MOVES onto the hint row while folded and
//	    returns to its last body row while expanded;
//	(e) the row math survives the top-edge TrimLeft: a thread TOPPING
//	    the timeline followed by a long user turn registers the hint at
//	    its TRUE visual row (renderWorkerGroup's top-edge lead covers
//	    only the block itself);
//	(f) body rows never toggle — the hit-map carries hint/collapse
//	    rows only;
//	(g) the transcript inset: View rows carry the chatPadL left pad and
//	    stay inside the panel budget, folded and expanded alike.
//
// No clocks: every timestamp is a literal; toggles ride ClickRow /
// ToggleUserFold directly (the same seams the app's mouse handler runs).
package panels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	state "github.com/theboringhumane/theboringoffice/internal/state"
)

// foldChat builds the bare chat holding ONE 5-row user turn
// ("l1"…"l5" — every row short, so wrap/fold passes at c.mdWidth+1 keep
// the five lines verbatim and the fold gate does the cutting). SetSize is
// the Chat's CONTENT width (tabs.go hands each tab its content area with
// the border already accounted for), so 44 here means a 44-cell View
// whose transcript TEXT runs contentW() = 44 − chatPadL − chatPadR = 40
// cells wide.
func foldChat(t *testing.T) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "l1\nl2\nl3\nl4\nl5", At: 10},
	}})
	return c
}

// TestUserFoldShortTurnUntouched is proof (a): a ≤userFoldVisible-row
// user turn renders byte-for-byte the pre-feature shape after
// ansi.Strip — no hint/collapse row, an empty fold hit-map — and every
// row (prefix indent included) lands exactly where it always did.
func TestUserFoldShortTurnUntouched(t *testing.T) {
	for _, text := range []string{"one", "l1\nl2", "l1\nl2\nl3"} {
		c := NewChat(nil)
		c.SetSize(44, 24)
		c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: text, At: 10},
		}})
		convo := ansi.Strip(c.renderConversation())
		var want string
		switch len(strings.Split(text, "\n")) {
		case 1:
			want = "you › one"
		case 2:
			want = "you › l1\n      l2"
		default:
			want = "you › l1\n      l2\n      l3"
		}
		if convo != want {
			t.Fatalf("a ≤%d-row user turn must not change shape:\n got %q\nwant %q", userFoldVisible, convo, want)
		}
		if len(c.userFoldRows) != 0 {
			t.Fatalf("a short user turn must register NO fold row, got %v", c.userFoldRows)
		}
	}
}

// TestUserFoldCollapsedShapeAndRowMath is proof (b): the 5-row turn
// folds to 3 body rows + the exact hint text, hanging under the bubble
// indent, with userFoldRows registering precisely the hint's content
// row — alone (row 3) and mid-stream after a boss turn (row 5).
func TestUserFoldCollapsedShapeAndRowMath(t *testing.T) {
	c := foldChat(t)
	convo := ansi.Strip(c.renderConversation())
	t.Logf("---- FOLDED 5-row user turn (44 cols, ansi-stripped) ----\n%s\n----", convo)
	want := "you › l1\n" +
		"      l2\n" +
		"      l3\n" +
		"      … +2 more lines · click to expand"
	if convo != want {
		t.Fatalf("folded user turn shape:\n got %q\nwant %q", convo, want)
	}
	if len(c.userFoldRows) != 1 || c.userFoldRows[3] != "u1" {
		t.Fatalf("the hint row must register at content row 3 → u1, got %v", c.userFoldRows)
	}

	// mid-stream: after a one-row user turn the SAME fold lands its hint
	// at content row 5 (first turn row 0, blank row 1, body 2-4, hint 5)
	c2 := NewChat(nil)
	c2.SetSize(44, 24)
	c2.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "hi", At: 5},
		{ID: "u2", From: "user", Kind: "user", Text: "l1\nl2\nl3\nl4\nl5", At: 10},
	}})
	convo2 := ansi.Strip(c2.renderConversation())
	if i := strings.Index(convo2, "… +2 more lines · click to expand"); i < 0 {
		t.Fatalf("mid-stream folded turn lost its hint row:\n%s", convo2)
	}
	if len(c2.userFoldRows) != 1 || c2.userFoldRows[5] != "u2" {
		t.Fatalf("mid-stream hint row must register at content row 5 → u2, got %v", c2.userFoldRows)
	}
}

// TestUserFoldClickToggles is proofs (c) + (f): a click on the fold hint
// row expands the bubble to all five rows + the "… collapse" trailer
// (registered, body rows NOT), a click on the trailer folds back to the
// exact pre-click shape, and clicks on body rows fall through unclaimed.
func TestUserFoldClickToggles(t *testing.T) {
	c := foldChat(t)
	// body rows are dead to clicks: row 1 ("l2") is not in the hit-map
	if c.ClickRow(3, 1) {
		t.Fatal("a click on a folded bubble's BODY row must not be claimed")
	}
	if !c.ClickRow(3, 3) {
		t.Fatal("a click on the fold hint row was not claimed")
	}
	convo := ansi.Strip(c.renderConversation())
	t.Logf("---- EXPANDED 5-row user turn (44 cols, ansi-stripped) ----\n%s\n----", convo)
	want := "you › l1\n" +
		"      l2\n" +
		"      l3\n" +
		"      l4\n" +
		"      l5\n" +
		"      … collapse"
	if convo != want {
		t.Fatalf("expanded user turn shape:\n got %q\nwant %q", convo, want)
	}
	if len(c.userFoldRows) != 1 || c.userFoldRows[5] != "u1" {
		t.Fatalf("the collapse trailer must register at content row 5 → u1, got %v", c.userFoldRows)
	}
	// body rows still dead while expanded
	if c.ClickRow(3, 4) {
		t.Fatal("a click on an expanded bubble's body row must not be claimed")
	}
	// the trailer folds back: the render returns to the folded shape
	if !c.ClickRow(3, 5) {
		t.Fatal("a click on the collapse trailer row was not claimed")
	}
	if convo := ansi.Strip(c.renderConversation()); !strings.Contains(convo, "… +2 more lines · click to expand") ||
		strings.Contains(convo, "l4") || strings.Contains(convo, "… collapse") {
		t.Fatalf("re-folding must restore the folded shape:\n%s", convo)
	}
	if len(c.userFoldRows) != 1 || c.userFoldRows[3] != "u1" {
		t.Fatalf("re-folded hint row must register at content row 3 → u1, got %v", c.userFoldRows)
	}
}

// TestUserFoldForceRenderPixels is proof (c, second half): the toggle
// flips the RENDERED viewport content with NO SetState call in between —
// forceRender, not the state revision gate, carries the pixel change.
func TestUserFoldForceRenderPixels(t *testing.T) {
	c := foldChat(t)
	before := ansi.Strip(c.View())
	if strings.Contains(before, "l4") || strings.Contains(before, "… collapse") {
		t.Fatalf("precondition: the fold hides l4/l5 before any toggle:\n%s", before)
	}
	c.ToggleUserFold("u1") // no SetState anywhere
	after := ansi.Strip(c.View())
	if after == before {
		t.Fatal("ToggleUserFold must change the rendered View without a SetState (forceRender)")
	}
	if !strings.Contains(after, "      l5") || !strings.Contains(after, "… collapse") {
		t.Fatalf("the expanded View must show the full body + collapse trailer:\n%s", after)
	}
	// (g) the transcript inset: the expanded body rows and the trailer
	// ride the chatPadL left gutter (6-cell hanging indent + 2-cell pad)
	if !strings.Contains(after, "\n        … collapse") {
		t.Fatalf("the collapse trailer must ride the chatPadL inset (2+6 cells in):\n%s", after)
	}
}

// TestUserFoldAttachSuffixRelocation is proof (d): the dim " · 📎 N"
// suffix rides the FOLD HINT row while collapsed (body rows are bare)
// and returns to the LAST BODY ROW while expanded (the pre-feature
// position), never duplicated, never dropped.
func TestUserFoldAttachSuffixRelocation(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(52, 24) // wide enough for the hint text AND the suffix on one row
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "l1\nl2\nl3\nl4\nl5",
			Meta: state.AttachMeta([]string{"a.png", "b.png"}), At: 10},
	}})
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "      … +2 more lines · click to expand · 📎 2") {
		t.Fatalf("folded: the 📎 count must ride the fold hint row:\n%s", convo)
	}
	if strings.Contains(convo, "l3 · 📎") || strings.Contains(convo, "l5") {
		t.Fatalf("folded: no body row may carry the suffix (or the tail):\n%s", convo)
	}
	c.ToggleUserFold("u1")
	convo = ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "      l5 · 📎 2\n") {
		t.Fatalf("expanded: the suffix returns to the last body row:\n%s", convo)
	}
	if strings.Contains(convo, "… collapse · 📎") {
		t.Fatalf("expanded: the collapse trailer must stay bare:\n%s", convo)
	}
}

// TestUserFoldTopEdgeThreadBefore is proof (e): with the thread block
// TOPPING the timeline (its "\n\n" lead dies in renderConversation's
// TrimLeft) and the long user turn second, the hint registers at its
// TRUE visual row — the topShift correction, matching
// renderWorkerGroup's top-edge lead for the block itself.
func TestUserFoldTopEdgeThreadBefore(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 24)
	c.SetState(state.OfficeState{
		Tick: 50,
		Chat: []state.ChatMsg{
			// the thread is born BEFORE the user's first word — it
			// tops the merged timeline (same seam as
			// TestThreadFirstGroupTopEdgeHitMap)
			{ID: "x1", From: "tekton-1", Kind: wtoolKind, Text: "Read internal/panels/chat.go", Meta: "done\x1f5", At: 10},
			{ID: "u1", From: "user", Kind: "user", Text: "l1\nl2\nl3\nl4\nl5", At: 20},
		},
	})
	convo := ansi.Strip(c.renderConversation())
	hintRow := -1
	for i, ln := range strings.Split(convo, "\n") {
		if strings.Contains(ln, "… +2 more lines · click to expand") {
			hintRow = i
		}
	}
	if hintRow < 0 {
		t.Fatalf("folded hint row missing:\n%s", convo)
	}
	t.Logf("top-edge fold: hint visual row %d, registered rows %v", hintRow, c.userFoldRows)
	if len(c.userFoldRows) != 1 || c.userFoldRows[hintRow] != "u1" {
		t.Fatalf("the hint must register at its TRUE visual row %d (top-shifted), got %v", hintRow, c.userFoldRows)
	}
	if !c.ClickRow(1, hintRow) {
		t.Fatalf("click on the top-edge-corrected hint row %d was not claimed", hintRow)
	}
	if convo := ansi.Strip(c.renderConversation()); !strings.Contains(convo, "      l5\n      … collapse") {
		t.Fatalf("the click must expand the user turn past the top-edge thread:\n%s", convo)
	}
}

// TestUserFoldFrame prints the canonical fold gallery — one long user
// turn in both states beside a normal thread frame — for eyeball review
// and pins the inset: every transcript row rides the chatPadL gutter and
// stays inside the panel budget in BOTH states.
func TestUserFoldFrame(t *testing.T) {
	c := foldChat(t)
	folded := ansi.Strip(c.View())
	fmt.Println("---- USER FOLD (44 cols: folded, ansi-stripped View) ----")
	fmt.Print(folded)
	c.ToggleUserFold("u1")
	expanded := ansi.Strip(c.View())
	fmt.Println("---- USER FOLD (44 cols: expanded, ansi-stripped View) ----")
	fmt.Print(expanded)
	// Budget accounting: SetSize hands the Chat its CONTENT width (see
	// foldChat), so every View row — the viewport pads transcript rows,
	// the divider/textarea chrome is full-width by design — is c.w cells
	// wide. The 40-cell budget is the transcript TEXT budget (contentW),
	// NOT a View-row budget: text runs c.w − chatPadL − chatPadR cells
	// and then rides the chatPadL gutter, so a transcript row's text
	// cells end at c.w − chatPadR at the latest. Two invariants, both
	// states: NO row may exceed the View's own c.w width (the sibling
	// chat_render_test.go convention — the panel never grows past its
	// SetSize width), and every transcript row inside the viewport
	// region keeps the chatPadL pad AND the chatPadR right gutter.
	for _, view := range []struct {
		tag   string
		frame string
	}{{"folded", folded}, {"expanded", expanded}} {
		for i, r := range strings.Split(view.frame, "\n") {
			if w := len([]rune(r)); w > c.w {
				t.Fatalf("%s row %d overflows the View's own %d-col width (%d cells): %q", view.tag, i, c.w, w, r)
			}
			if i >= c.vp.Height() {
				continue // below the transcript: divider/textarea chrome is full-width
			}
			trimmed := strings.TrimRight(r, " ")
			if trimmed == "" {
				continue // the viewport's blank fill rows carry the pad only
			}
			if !strings.HasPrefix(r, strings.Repeat(" ", chatPadL)) {
				t.Fatalf("%s transcript row %d lost its chatPadL left pad: %q", view.tag, i, r)
			}
			if w := len([]rune(trimmed)); w > c.w-chatPadR {
				t.Fatalf("%s transcript row %d crosses the chatPadR right gutter (%d text cells > %d): %q", view.tag, i, w, c.w-chatPadR, r)
			}
		}
	}
	// the empty-key fallback: two ID-less user turns fold INDEPENDENTLY
	// (the "at-<At>" key, not a shared "")
	c2 := NewChat(nil)
	c2.SetSize(44, 24)
	c2.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{From: "user", Kind: "user", Text: "l1\nl2\nl3\nl4", At: 10},
		{From: "user", Kind: "user", Text: "m1\nm2\nm3\nm4", At: 20},
	}})
	c2.ToggleUserFold("at-20")
	convo := ansi.Strip(c2.renderConversation())
	if !strings.Contains(convo, "… +1 more lines · click to expand") ||
		!strings.Contains(convo, "      m4\n      … collapse") {
		t.Fatalf("the ID-less fallback key must fold/expand per-message:\n%s", convo)
	}
}
