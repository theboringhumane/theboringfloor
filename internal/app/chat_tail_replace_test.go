// chat_tail_replace_test.go — the ordering-invariance window for the
// tail-first replace scans in the chat reducer: stream deltas land on the
// TAIL of the transcript, so the reducer's find-by-ID loops sweep backwards
// first (with the historical head sweep kept as fallback inside the same
// branch). These proofs pin that direction change to byte-for-byte
// semantics: a delta replacing a TAIL entry and one replacing a HEAD entry
// must both land on their exact original indices, and the resulting
// transcript must be byte-identical to the naive head-first replacement.
package app

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// replaceByIDHeadFirst is the reference helper: the pre-optimization
// head-first semantics — first ID match wins, else append.
func replaceByIDHeadFirst(chat []state.ChatMsg, msg state.ChatMsg) []state.ChatMsg {
	out := append([]state.ChatMsg(nil), chat...)
	for i, m := range out {
		if m.ID == msg.ID {
			out[i] = msg
			return out
		}
	}
	return append(out, msg)
}

// TestTailFirstReplacePreservesIndices builds a 5000-entry transcript, then
// drives a replace delta at index 4997 (tail region) AND one at index 3
// (head region) plus a no-ID-match delta, asserting every one lands exactly
// where the head-first semantics would have put it.
func TestTailFirstReplacePreservesIndices(t *testing.T) {
	const pairs = 2500 // 2500 user+boss pairs = a 5000-entry transcript
	const n = 2 * pairs

	st := initialState(state.ModeDemo)
	for i := 1; i <= pairs; i++ {
		st = reducer(st, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
			ID: fmt.Sprintf("u%d", i), From: "user", Text: fmt.Sprintf("question %d", i), At: int64(i * 2)}})
		st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: fmt.Sprintf("r%d", i), From: "boss", Text: fmt.Sprintf("answer %d", i), At: int64(i*2 + 1)}})
	}
	if len(st.Chat) != n {
		t.Fatalf("setup: transcript = %d entries, want %d", len(st.Chat), n)
	}
	t.Logf("assert[0]: transcript of %d entries built (oldest head, newest tail)", len(st.Chat))
	base := st // the pre-delta transcript for the byte-identical reference

	// (1) TAIL-region replace: r2499 lives at index (2499-1)*2+1 = 4997.
	tailDelta := state.ChatMsg{ID: "r2499", From: "boss", Text: "TAIL EDIT delta", At: 2*2499 + 1}
	st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: tailDelta})
	if len(st.Chat) != n {
		t.Fatalf("tail replace inflated the transcript: len = %d, want %d", len(st.Chat), n)
	}
	if got := st.Chat[4997]; got.ID != "r2499" || got.Text != "TAIL EDIT delta" {
		t.Fatalf("tail replace: index 4997 = %q/%q, want r2499/TAIL EDIT delta", got.ID, got.Text)
	}
	t.Logf("assert[1]: tail delta replaced index 4997 in place (id r2499, transcript len still %d)", len(st.Chat))

	// (2) HEAD-region replace: r2 lives at index (2-1)*2+1 = 3.
	headDelta := state.ChatMsg{ID: "r2", From: "boss", Text: "HEAD EDIT delta", At: 2*2 + 1}
	st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: headDelta})
	if got := st.Chat[3]; got.ID != "r2" || got.Text != "HEAD EDIT delta" {
		t.Fatalf("head replace: index 3 = %q/%q, want r2/HEAD EDIT delta", got.ID, got.Text)
	}
	if got := st.Chat[2]; got.ID != "u2" {
		t.Fatalf("head replace disturbed the left neighbor: index 2 = %q, want u2", got.ID)
	}
	if got := st.Chat[4]; got.ID != "u3" {
		t.Fatalf("head replace disturbed the right neighbor: index 4 = %q, want u3", got.ID)
	}
	t.Log("assert[2]: head delta replaced index 3 in place; neighbors (u2, u3) untouched")

	// (3) a delta with an UNKNOWN ID must miss every scan and append.
	missDelta := state.ChatMsg{ID: "bossmsg-fresh", From: "boss", Text: "brand new bubble", At: 90001}
	st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: missDelta})
	if len(st.Chat) != n+1 {
		t.Fatalf("miss delta: len = %d, want %d (append, no inflation)", len(st.Chat), n+1)
	}
	if got := st.Chat[n]; got.ID != "bossmsg-fresh" {
		t.Fatalf("miss delta: tail = %q, want bossmsg-fresh", got.ID)
	}
	t.Log("assert[3]: no-ID-match delta appended at the tail (miss edge preserved)")

	// (4) byte-identical against the head-first reference helper, deltas
	// applied in the same order.
	want := replaceByIDHeadFirst(base.Chat, tailDelta)
	want = replaceByIDHeadFirst(want, headDelta)
	want = replaceByIDHeadFirst(want, missDelta)
	if !reflect.DeepEqual(st.Chat, want) {
		t.Fatalf("resulting chat differs from the head-first reference")
	}
	t.Log("assert[4]: resulting chat is byte-identical to the head-first reference (DeepEqual)")
}

// TestChatStreamDeltaHotPath is the transcript-heavy timing probe for the
// tail-first scans and the appendChat capacity reuse: a live boss bubble
// streams D growth deltas onto a transcript of P user/boss pairs. It is a
// BEHAVIOR test (the final bubble must hold the last delta; nothing else
// may move) — run with -count=N before/after the optimization to time it.
func TestChatStreamDeltaHotPath(t *testing.T) {
	const pairs = 1200
	const deltas = 300

	st := initialState(state.ModeDemo)
	for i := 1; i <= pairs; i++ {
		st = reducer(st, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
			ID: fmt.Sprintf("u%d", i), From: "user", Text: fmt.Sprintf("question %d", i), At: int64(i * 2)}})
		st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: fmt.Sprintf("r%d", i), From: "boss", Text: fmt.Sprintf("answer %d", i), At: int64(i*2 + 1)}})
	}
	// The live stream: every delta re-emits the SAME stable ID and grows
	// the bubble — the exact shape of a streaming reply.
	base := len(st.Chat)
	for d := 1; d <= deltas; d++ {
		st = reducer(st, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: "bossmsg-live", From: "boss", Pending: d != deltas,
			Text: fmt.Sprintf("streamed answer body (delta %d)", d), At: 999000 + int64(d)}})
	}
	if len(st.Chat) != base+1 {
		t.Fatalf("stream of %d deltas must collapse into ONE bubble: len = %d, want %d",
			deltas, len(st.Chat), base+1)
	}
	last := st.Chat[len(st.Chat)-1]
	if last.ID != "bossmsg-live" || last.Text != fmt.Sprintf("streamed answer body (delta %d)", deltas) {
		t.Fatalf("final bubble = %q/%q, want bossmsg-live with the LAST delta's text", last.ID, last.Text)
	}
	if last.Pending {
		t.Fatalf("the pinned final must clear Pending")
	}
	if st.Chat[len(st.Chat)-2].ID != fmt.Sprintf("r%d", pairs) {
		t.Fatalf("the stream bubble must sit at the tail, after r%d", pairs)
	}
}
