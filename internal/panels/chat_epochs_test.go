// chat_epochs_test.go — the same-name worker EPOCH segmentation proofs:
// desk names are recycled across restarts and re-tasks, so the chat
// panel must not weld every wave of "tekton-9" lines into ONE thread
// wearing today's task title. A same-name line arriving more than
// workforceGap (10 min) after the name's previous line opens a NEW
// segment — its own timeline birth slot, its own captured dispatch
// title; inside the gap the lines merge as one work session. Timestamps
// are fixed literals — no time.Now(), the tests are deterministic.
package panels

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// gapMs — workforceGap in the ChatMsg.At unix-millis unit.
func gapMs() int64 { return workforceGap.Milliseconds() }

// TestEpochSplitKeepsPerSegmentTitles drives the live-session welding
// bug: tekton-9 runs one wave, returns, and is re-tasked MORE than
// workforceGap later. The second render must show TWO thread segments
// for the same name, each wearing ITS OWN dispatch title — the settled
// epoch must NOT inherit the current sticky task (the pre-fix behavior:
// both headers read "Backfill session caps").
func TestEpochSplitKeepsPerSegmentTitles(t *testing.T) {
	const t0 = int64(1_000_000)
	c := NewChat(nil)
	c.SetSize(100, 40)

	wave1 := []state.ChatMsg{
		{ID: "wa1", From: "tekton-9", Kind: wtoolKind, Text: "read · sse.go", Meta: "done\x1f8", At: t0},
		{ID: "wa2", From: "tekton-9", Kind: wtoolKind, Text: "edit · sse.go", Meta: "done\x1f9", At: t0 + 60_000},
	}
	c.SetState(state.OfficeState{
		Tick: 10,
		Employees: []state.Employee{
			{ID: "d9", Name: "tekton-9", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "Wire SSE stream"},
		},
		Chat: append([]state.ChatMsg{{ID: "u1", From: "user", Kind: "user", Text: "ship it", At: t0 - 500}}, wave1...),
	})
	first := ansi.Strip(c.renderConversation())
	if n := strings.Count(first, "Developer Task — Wire SSE stream"); n != 1 {
		t.Fatalf("wave 1 must render exactly one thread header, got %d:\n%s", n, first)
	}

	// the agent returns and is re-tasked a full epoch later — same desk
	// name, new wave of wtool lines past workforceGap
	wave2 := []state.ChatMsg{
		{ID: "wb1", From: "tekton-9", Kind: wtoolKind, Text: "grep · sessionChatCap", Meta: "done\x1f9000", At: t0 + 60_000 + gapMs() + 60_000},
		{ID: "wb2", From: "tekton-9", Kind: wtoolKind, Text: "edit · sessions.go", Meta: "running\x1f9001", At: t0 + 60_000 + gapMs() + 120_000},
	}
	c.SetState(state.OfficeState{
		Tick: 9002,
		Employees: []state.Employee{
			{ID: "d9", Name: "tekton-9", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "Backfill session caps"},
		},
		Chat: append([]state.ChatMsg{{ID: "u1", From: "user", Kind: "user", Text: "ship it", At: t0 - 500}},
			append(append([]state.ChatMsg{}, wave1...), wave2...)...),
	})
	convo := ansi.Strip(c.renderConversation())

	// TWO segments: the settled epoch keeps its own birth-captured title
	// (with its · 2 tool calls rollup), the live epoch wears the current
	// task — never both wearing the current one.
	if n := strings.Count(convo, "Developer Task — Wire SSE stream (· 2 tool calls ✓ done)"); n != 1 {
		t.Fatalf("the settled epoch must keep its own captured title + rollup, got %d:\n%s", n, convo)
	}
	if n := strings.Count(convo, "Developer Task — Backfill session caps"); n != 1 {
		t.Fatalf("the live epoch must wear the current task exactly once, got %d:\n%s", n, convo)
	}
	// each segment anchors at ITS OWN birth slot: the older epoch sorts
	// above the newer one (first-seen order within a segment is kept).
	iw, ib := strings.Index(convo, "Developer Task — Wire SSE stream"), strings.Index(convo, "Developer Task — Backfill session caps")
	if iw < 0 || ib < 0 || iw > ib {
		t.Fatalf("epochs must interleave by their own birth stamps (wire=%d backfill=%d):\n%s", iw, ib, convo)
	}
}

// TestEpochMergeWithinGap guards the continuous-session rule: same-name
// lines INSIDE the gap stay ONE thread — a quickly re-tasked agent is
// one work session, rolled up under a single header.
func TestEpochMergeWithinGap(t *testing.T) {
	const t0 = int64(1_000_000)
	c := NewChat(nil)
	c.SetSize(100, 40)
	c.SetState(state.OfficeState{
		Tick: 1000, // meta ticks 1/2 → both settled (idle sprite too)
		Employees: []state.Employee{
			{ID: "d9", Name: "tekton-9", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk, Task: "Sweep lint"},
		},
		Chat: []state.ChatMsg{
			{ID: "wa1", From: "tekton-9", Kind: wtoolKind, Text: "grep · revive", Meta: "done\x1f1", At: t0},
			{ID: "wa2", From: "tekton-9", Kind: wtoolKind, Text: "edit · chat.go", Meta: "done\x1f2", At: t0 + gapMs()/2}, // 5 min < the gap
		},
	})
	convo := ansi.Strip(c.renderConversation())
	if n := strings.Count(convo, "Developer Task — Sweep lint (· 2 tool calls ✓ done)"); n != 1 {
		t.Fatalf("inside the gap the waves merge into ONE thread with ONE rollup, got %d:\n%s", n, convo)
	}
}

// TestEpochSplitOnRestoredTranscript proves the split survives a cold
// memo (the first render after a boot restore — segment titles have no
// live-birth captures, so an older epoch falls back to the sticky map,
// per the design ruling). The SEGMENTATION itself must still land: two
// epochs of one name render as two threads at their own birth slots.
func TestEpochSplitOnRestoredTranscript(t *testing.T) {
	const t0 = int64(1_000_000)
	c := NewChat(nil)
	c.SetSize(100, 40)
	c.SetState(state.OfficeState{
		Tick: 1000,
		Employees: []state.Employee{
			{ID: "d9", Name: "tekton-9", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "Backfill session caps"},
		},
		Chat: []state.ChatMsg{
			{ID: "wa1", From: "tekton-9", Kind: wtoolKind, Text: "read · sse.go", Meta: "done\x1f1", At: t0},
			{ID: "wa2", From: "tekton-9", Kind: wtoolKind, Text: "edit · sse.go", Meta: "done\x1f2", At: t0 + 60_000},
			{ID: "wb1", From: "tekton-9", Kind: wtoolKind, Text: "grep · sessionChatCap", Meta: "done\x1f999", At: t0 + 60_000 + gapMs() + 60_000},
		},
	})
	convo := ansi.Strip(c.renderConversation())
	// two epochs → two headers; the settled one rolls its · 2 tool calls
	// up, the fresh one stays live (sprite working, tick 1 back).
	if n := strings.Count(convo, "Developer Task — Backfill session caps"); n != 2 {
		t.Fatalf("a restored transcript must still split the epochs, got %d headers:\n%s", n, convo)
	}
	if n := strings.Count(convo, "Developer Task — Backfill session caps (· 2 tool calls ✓ done)"); n != 1 {
		t.Fatalf("the older epoch renders settled with its own rollup, got %d:\n%s", n, convo)
	}
}

// TestEpochStampLessLinesNeverSplit pins the At==0 guard: stamp-less
// legacy lines (or zero-stamped stream placeholders) must NOT force an
// epoch boundary — with no two positive stamps a gap can't be measured,
// so the lines merge into one thread.
func TestEpochStampLessLinesNeverSplit(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(100, 40)
	c.SetState(state.OfficeState{
		Tick: 1000,
		Employees: []state.Employee{
			{ID: "d9", Name: "tekton-9", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk, Task: "Sweep lint"},
		},
		Chat: []state.ChatMsg{
			{ID: "wa1", From: "tekton-9", Kind: wtoolKind, Text: "grep · revive", Meta: "done\x1f1", At: 0},
			{ID: "wa2", From: "tekton-9", Kind: wtoolKind, Text: "edit · chat.go", Meta: "done\x1f2", At: 0},
		},
	})
	convo := ansi.Strip(c.renderConversation())
	if n := strings.Count(convo, "Developer Task — Sweep lint"); n != 1 {
		t.Fatalf("stamp-less lines must stay ONE thread, got %d headers:\n%s", n, convo)
	}
}
