// thread_focus_test.go — the thread focus VIEW contract (the pane half;
// the app-level binding claims live in internal/app/thread_focus_test.go):
//
//	(a) one focused group renders end-to-end — header glyph + title +
//	    counters, a [tool] row, the FULL think body, the ↳ diff sub-row;
//	(b) the ToggleThreadDiff path THROUGH THE CLONE: a Click on the ↳
//	    sub-row opens the parsed wdiff body, a second Click folds it, and
//	    every other row stays inert;
//	(c) scroll keys move the clone's own viewport offset;
//	(d) the /tools-off empty agent renders the frozen empty-state row
//	    (and zero-counter header), with scroll/click no-ops;
//	(e) the header glyph: the office-tick braille frame while LIVE, the
//	    dim ✓ once settled.
package panels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func tfChat(id, from, kind, text, meta string) state.ChatMsg {
	return state.ChatMsg{ID: id, From: from, Kind: kind, Text: text, Meta: meta, At: 1000}
}

func tfEmployees() []state.Employee {
	return []state.Employee{
		{ID: "boss-1", Name: "boss", Role: state.RoleManager, Sprite: state.SpriteAtDesk},
		{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "Wire the SSE stream"},
		{ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk, Task: "Scan the repo"},
	}
}

// tfFixture — the focused agent (tekton-1) carries one done read, one
// think, one done edit WITH its per-call wdiff, while the UNFOCUSED agent
// (skopos-1) owns a line the view must never show. st.Tick=100, freshest
// tags 99/100 → tekton-1's thread is LIVE.
func tfFixture() state.OfficeState {
	return state.OfficeState{
		Tick:      100,
		Mode:      state.ModeLive,
		Employees: tfEmployees(),
		Chat: []state.ChatMsg{
			tfChat("wtool-tekton-1-call-t1", "tekton-1", "wtool", "read · internal/room/manager.go", "done\x1f99"),
			tfChat("wthink-tekton-1-th1", "tekton-1", "wthink", "handler first, stream second", "done\x1f99"),
			tfChat("wtool-tekton-1-call-t2", "tekton-1", "wtool", "edit · internal/room/handler.go", "done\x1f100"),
			tfChat("wdiff-tekton-1-call-t2", "tekton-1", "wdiff",
				"--- a/internal/room/handler.go\n+++ b/internal/room/handler.go\n@@ -1 +1,2 @@\n-a\n+focusBodyMarker\n",
				"internal/room/handler.go\x1f+3\x1f-1"),
			tfChat("wtool-skopos-1-call-s1", "skopos-1", "wtool", "grep · SSE, 3 hits", "done\x1f100"),
		},
	}
}

// (a) end-to-end: header (live glyph + title + counters), merged tool
// rows, the FULL think body, the ↳ diff sub-row — and strict isolation
// from the sibling thread.
func TestThreadFocusRendersGroupEndToEnd(t *testing.T) {
	tf := NewThreadFocus("tekton-1", 100, 24)
	tf.SetState(tfFixture())
	out := ansi.Strip(tf.View())

	// header: office-tick braille for LIVE (tick 100 → frame 100%7 = "⠯")
	if !strings.Contains(out, "⠯ Developer Task — Wire the SSE stream · 2 tool calls · 1 think") {
		t.Fatalf("header row missing live glyph + title + counters:\n%s", out)
	}
	// body: the merged rows, FULL expansion
	for _, want := range []string{
		"  [tool] ▸ Read internal/room/manager.go ✓",
		"  [tool] ▸ Edit internal/room/handler.go ✓ · +3 -1",
		"    handler first, stream second", // the FULL think body, not the "· N lines" rollup
		"  ↳ diff · internal/room/handler.go +3 -1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("focus body missing %q:\n%s", want, out)
		}
	}
	// the wdiff body starts CLOSED
	if strings.Contains(out, "focusBodyMarker") {
		t.Fatalf("the parsed diff body must wait for the ↳ click:\n%s", out)
	}
	// strict isolation: the other thread's rows never enter the view
	if strings.Contains(out, "skopos-1") || strings.Contains(out, "SSE, 3 hits") {
		t.Fatalf("the focus must show ONLY the focused thread:\n%s", out)
	}
}

// (b) the wdiff body path through the clone: Click claims the ↳ sub-row,
// the parsed line-numbered body opens INSIDE the pane, a second click
// folds it, and the rest of the surface stays click-inert.
func TestThreadFocusDiffClickTogglesBody(t *testing.T) {
	tf := NewThreadFocus("tekton-1", 100, 24)
	tf.SetState(tfFixture())
	out := ansi.Strip(tf.View())

	diffY := -1
	for i, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "↳ diff · internal/room/handler.go") {
			diffY = i
			break
		}
	}
	if diffY < 0 {
		t.Fatalf("setup: the ↳ diff sub-row never rendered:\n%s", out)
	}
	if !tf.Click(6, diffY) {
		t.Fatalf("the ↳ diff row must claim its click")
	}
	opened := ansi.Strip(tf.View())
	if !strings.Contains(opened, "focusBodyMarker") {
		t.Fatalf("a ↳ click must open the parsed body INSIDE the pane:\n%s", opened)
	}
	// in v1 the ↳ row keeps its spot (body inserted below it) — a second
	// click at the same row folds the body back
	if !tf.Click(6, diffY) {
		t.Fatalf("the ↳ row must still claim the closing click")
	}
	if folded := ansi.Strip(tf.View()); strings.Contains(folded, "focusBodyMarker") {
		t.Fatalf("the second ↳ click must close the body:\n%s", folded)
	}
	// inert rows: the header and a lone tool row claim nothing
	if tf.Click(6, 0) {
		t.Fatal("the header row must be click-inert in v1")
	}
	toolY := -1
	for i, ln := range strings.Split(ansi.Strip(tf.View()), "\n") {
		if strings.Contains(ln, "[tool] ▸ Read internal/room/manager.go") {
			toolY = i
			break
		}
	}
	if toolY < 0 {
		t.Fatalf("setup: lost the sibling tool row")
	}
	if tf.Click(6, toolY) {
		t.Fatal("a tool row (the thread's own frame) must be click-inert in v1")
	}
}

// (c) scrolling moves the pane's OWN viewport (not the office's).
func TestThreadFocusScrollMovesViewport(t *testing.T) {
	tf := NewThreadFocus("tekton-1", 100, 8) // 7 body rows, 30 tool rows deep
	st := tfFixture()
	st.Chat = nil
	for i := 0; i < 30; i++ {
		st.Chat = append(st.Chat, tfChat(
			fmt.Sprintf("wtool-tekton-1-call-%02d", i), "tekton-1", "wtool",
			fmt.Sprintf("read · file-%02d.go", i), "done\x1f99"))
	}
	tf.SetState(st)
	followEnd := tf.scrollOffset()
	if followEnd == 0 {
		t.Fatalf("a 30-row thread in a 7-row body must start pinned to the bottom (offset 0 means nothing rendered below the fold)")
	}
	tf.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}))
	if off := tf.scrollOffset(); off >= followEnd {
		t.Fatalf("pgup must scroll the pane's own viewport up (was %d, now %d)", followEnd, off)
	}
	off := tf.scrollOffset()
	tf.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	if back := tf.scrollOffset(); back <= off {
		t.Fatalf("pgdn must scroll back down (was %d, now %d)", off, back)
	}
}

// (d) the /tools-off empty agent: frozen empty-state row, zero counters,
// scroll/click no-ops.
func TestThreadFocusEmptyAgent(t *testing.T) {
	tf := NewThreadFocus("skopos-1", 100, 10)
	st := tfFixture()
	st.Chat = st.Chat[:len(st.Chat)-1] // drop skopos-1's ONLY line
	tf.SetState(st)
	out := ansi.Strip(tf.View())

	if !strings.Contains(out, "✓ Explore Task — Scan the repo · 0 tool calls") {
		t.Fatalf("the header reads title + zero counters (done glyph — no lines, not live):\n%s", out)
	}
	if !strings.Contains(out, "  no recorded tool calls for skopos-1 (tools hidden?)") {
		t.Fatalf("the frozen /tools-off empty-state row must render:\n%s", out)
	}
	tf.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}))
	if out2 := ansi.Strip(tf.View()); !strings.Contains(out2, "no recorded tool calls for skopos-1") {
		t.Fatalf("scroll keys must no-op on the empty state:\n%s", out2)
	}
	if tf.Click(4, 1) {
		t.Fatal("an empty body claims no click")
	}
}

// (e) the glyph flip: a settled roster + stale tags → the dim ✓.
func TestThreadFocusGlyphDone(t *testing.T) {
	tf := NewThreadFocus("tekton-1", 100, 24)
	st := tfFixture()
	st.Mode = state.ModeLive
	st.Tick = 50
	for i := range st.Employees {
		st.Employees[i].Sprite = state.SpriteAtDesk
	}
	tf.SetState(st)
	out := ansi.Strip(tf.View())
	if !strings.Contains(out, "✓ Developer Task — Wire the SSE stream · 2 tool calls · 1 think") {
		t.Fatalf("a settled thread takes the dim ✓ + counters:\n%s", out)
	}
	for _, g := range []string{"⠿", "⠷", "⠯", "⠟", "⠻", "⠽", "⠾"} {
		if strings.Contains(strings.Split(out, "\n")[0], g) {
			t.Fatalf("the header row must not animate once settled:\n%s", out)
		}
	}
}
