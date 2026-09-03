// claude_events_test.go — fixture-driven mapping coverage for the claude
// stream-json -> state.Event normalizer: init / user / assistant /
// tool_use / result / api_retry / control_request / system, including the
// tolerance rules (unknown types, partial input_json folding, no panics on
// weird payloads).
package backend

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// claudeFeed decodes one JSONL fixture line and runs it through
// mapClaudeEvent against ctx, returning the mapped events.
func claudeFeed(t *testing.T, ctx *claudeNormCtx, line string, now int64) []state.Event {
	t.Helper()
	var raw claudeEvent
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatalf("fixture doesn't decode (the STUB would be broken): %v", err)
	}
	return mapClaudeEvent(raw, ctx, now)
}

func claudeKinds(evs []state.Event) []state.EventKind {
	out := make([]state.EventKind, len(evs))
	for i, e := range evs {
		out[i] = e.Kind
	}
	return out
}

func TestClaudeInitMapsStatusAndMCP(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := claudeFeed(t, ctx, `{"type":"system","subtype":"init","cwd":"/x","session_id":"sess-1","model":"claude-test","mcp_servers":[{"name":"memo","status":"connected"}],"claude_code_version":"2.1.246","uuid":"init-1"}`, 100)
	if len(evs) != 1 || evs[0].Kind != state.EvStatus {
		t.Fatalf("init must emit exactly one EvStatus, got %v", claudeKinds(evs))
	}
	if want := "[claude] init model=claude-test session=sess-1"; evs[0].Text != want {
		t.Fatalf("init line = %q, want %q", evs[0].Text, want)
	}
	if ctx.primaryID != "sess-1" || ctx.model != "claude-test" {
		t.Fatalf("init didn't pin the boss session: %+v", ctx)
	}
	if len(ctx.mcp) != 1 || ctx.mcp[0].Name != "memo" || ctx.mcp[0].Status != "connected" {
		t.Fatalf("mcp pin drifted: %+v", ctx.mcp)
	}
}

func TestClaudeUnknownFramesAreSilent(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	for _, line := range []string{
		`{"type":"something_new","novel":true}`,
		`{"type":"system","subtype":"compact_boundary","compact_metadata":{}}`,
		`{"type":"system","subtype":"hook_started"}`,
		`{"type":"assistant"}`,                      // no message at all
		`{"type":"result","subtype":"queued"}`,      // unknown result subtype stays mapping-only
		`{"type":"control_response","response":{}}`, // our own answer echoes nowhere
	} {
		if evs := claudeFeed(t, ctx, line, 100); len(evs) != 0 {
			t.Fatalf("unknown frame %s mapped to %v — must stay silent", line, claudeKinds(evs))
		}
	}
}

func TestClaudeAssistantSnapshotPinsBossBubble(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-1","role":"assistant","model":"claude-test","content":[{"type":"text","text":"Roger that."}]},"session_id":"sess-1","uuid":"msg-1","parent_tool_use_id":null}`, 200)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss {
		t.Fatalf("assistant text must pin one boss bubble, got %v", claudeKinds(evs))
	}
	msg := evs[0].Msg
	if msg.ID != "bossmsg-msg-1" || msg.Pending || msg.Text != "Roger that." || msg.From != "boss" || msg.Kind != "boss" {
		t.Fatalf("pinned bubble drifted: %+v", msg)
	}
	// a replayed snapshot never re-pins (bossCompleted parity)
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-1","role":"assistant","content":[{"type":"text","text":"Roger that."}]},"session_id":"sess-1","uuid":"msg-1","parent_tool_use_id":null}`, 300); len(evs) != 0 {
		t.Fatalf("replayed snapshot emitted %v", claudeKinds(evs))
	}
}

func TestClaudeStreamThenSnapshotSingleBubbleNoDoubleGrowth(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-7","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Honey never"}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss || !evs[0].Msg.Pending {
		t.Fatalf("delta must grow a pending boss bubble, got %v", claudeKinds(evs))
	}
	if evs[0].Msg.ID != "bossmsg-msg-7" || evs[0].Msg.Text != "Honey never" {
		t.Fatalf("stream bubble drifted: %+v", evs[0].Msg)
	}
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" spoils"}},"session_id":"sess-1","parent_tool_use_id":null}`, 130)
	if evs[0].Msg.Text != "Honey never spoils" {
		t.Fatalf("stream accumulates: got %q", evs[0].Msg.Text)
	}
	// the mid-stream PARTIAL snapshot (real wire: lands BEFORE the stops,
	// sharing the message id) REFRESHES the same bubble — it must NOT pin.
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-7","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"frame-partial-7","parent_tool_use_id":null}`, 135)
	if len(evs) != 1 || evs[0].Msg.ID != "bossmsg-msg-7" || !evs[0].Msg.Pending || evs[0].Msg.Text != "Honey never spoils" {
		t.Fatalf("a mid-stream partial snapshot must refresh pending, not pin: %+v", evs)
	}
	if ctx.pinned["msg-7"] {
		t.Fatalf("a mid-stream partial must never set the one-shot pin")
	}
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 136)
	// the real wire sends NO final assistant snapshot after message_stop —
	// the close itself pins the FINAL text on the same id, once, Pending=false.
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`, 137)
	if len(evs) != 1 || evs[0].Msg.ID != "bossmsg-msg-7" || evs[0].Msg.Pending || evs[0].Msg.Text != "Honey never spoils" {
		t.Fatalf("message_stop must pin the same bubble id, got %+v", evs)
	}
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-7","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"msg-7","parent_tool_use_id":null}`, 141); len(evs) != 0 {
		t.Fatalf("a duplicate snapshot re-emitted %v", claudeKinds(evs))
	}
}

func TestClaudeAssistantSnapshotPrefersMessageIDOverFrameUUID(t *testing.T) {
	// the stream pins under message.id; a snapshot whose frame uuid differs
	// must join THAT bubble (and thought), never mint "bossmsg-"+<frame uuid>.
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-real"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"half"}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	if len(evs) != 1 || evs[0].Msg.ID != "bossmsg-msg-real" {
		t.Fatalf("the stream must grow bossmsg-msg-real, got %+v", evs)
	}
	// the snapshot's frame uuid differs from message.id — the refresh MUST
	// land on message.id or the streamed bubble forks. Mid-stream it stays
	// pending (never pins), but its FULLER text ("half an answer" vs the
	// streamed "half") becomes the bubble's best-known text.
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-real","role":"assistant","content":[{"type":"thinking","thinking":"hmm"},{"type":"text","text":"half an answer"}]},"session_id":"sess-1","uuid":"frame-uuid-9","parent_tool_use_id":null}`, 130)
	var refresh *state.ChatMsg
	var thought *state.Event
	for i := range evs {
		switch evs[i].Kind {
		case state.EvChatBoss:
			refresh = &evs[i].Msg
		case state.EvThought:
			thought = &evs[i]
		}
	}
	if refresh == nil || refresh.ID != "bossmsg-msg-real" || !refresh.Pending || refresh.Text != "half an answer" {
		t.Fatalf("snapshot must refresh the streamed bubble (message.id), got %+v", evs)
	}
	if thought == nil || thought.CallID != "think-msg-real" || !thought.Done {
		t.Fatalf("snapshot thought must close on think-msg-real, got %+v", evs)
	}
	if ctx.pinned["frame-uuid-9"] || ctx.pinned["msg-real"] {
		t.Fatalf("a mid-stream snapshot must never pin: %+v", ctx.pinned)
	}
	// the stream close pins DONE on the SAME message.id bubble — carrying
	// the snapshot's fuller text, never the frame uuid's id.
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 140)
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`, 150)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss || evs[0].Msg.ID != "bossmsg-msg-real" ||
		evs[0].Msg.Pending || evs[0].Msg.Text != "half an answer" {
		t.Fatalf("message_stop must pin the streamed bubble (message.id), got %+v", evs)
	}
	if !ctx.pinned["msg-real"] || ctx.pinned["frame-uuid-9"] {
		t.Fatalf("the frame uuid must never be a pin key: %+v", ctx.pinned)
	}
}

func TestClaudeAssistantSnapshotFallsBackToFrameUUID(t *testing.T) {
	// no message.id on the wire: the frame uuid is the only key left.
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"idless answer"}]},"session_id":"sess-1","uuid":"frame-only-1","parent_tool_use_id":null}`, 100)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss || evs[0].Msg.ID != "bossmsg-frame-only-1" || evs[0].Msg.Pending {
		t.Fatalf("an idless snapshot must pin under the frame uuid, got %+v", evs)
	}
}

// TestClaudePartialSnapshotIncompleteTextNeverFreezesBubble is THE freeze
// regression: a mid-stream PARTIAL snapshot carrying INCOMPLETE text must
// REFRESH (never pin) — the one-shot pin used to freeze the truncated
// partial forever and ignore the complete text. The final rendered state
// (the chat reducer's replace-merge fold) must be ONE bubble,
// Pending:false, complete text.
func TestClaudePartialSnapshotIncompleteTextNeverFreezesBubble(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-p1","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 101)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Honey nev"}},"session_id":"sess-1","parent_tool_use_id":null}`, 102)
	// the TRUNCATED partial snapshot: same message.id, mid-stream — it
	// refreshes the pending bubble, it must NEVER pin.
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-p1","role":"assistant","content":[{"type":"text","text":"Honey nev"}]},"session_id":"sess-1","uuid":"frame-partial-p1","parent_tool_use_id":null}`, 103)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss || evs[0].Msg.ID != "bossmsg-msg-p1" ||
		!evs[0].Msg.Pending || evs[0].Msg.Text != "Honey nev" {
		t.Fatalf("the truncated partial must refresh pending, got %+v", evs)
	}
	if ctx.pinned["msg-p1"] {
		t.Fatalf("the truncated partial set the one-shot pin — the freeze bug")
	}
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"er spoils"}},"session_id":"sess-1","parent_tool_use_id":null}`, 104)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 105)
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`, 106)
	if len(evs) != 1 || evs[0].Msg.ID != "bossmsg-msg-p1" || evs[0].Msg.Pending || evs[0].Msg.Text != "Honey never spoils" {
		t.Fatalf("the stream close must pin the COMPLETE text, got %+v", evs)
	}
	// a late complete snapshot (a wire that DOES send one) is absorbed
	// idempotently — single bubble, no double growth.
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-p1","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"msg-p1","parent_tool_use_id":null}`, 107); len(evs) != 0 {
		t.Fatalf("the post-pin snapshot must stay silent, got %v", claudeKinds(evs))
	}
}

// TestClaudeBubblePinsAtMessageStopRealWire drives the EXACT frame order
// captured off a live `claude -p --include-partial-messages` run (CLI
// 2.1.247): the wire sends NO final assistant snapshot after
// message_stop — the last assistant frame is the mid-stream PARTIAL. The
// bubble must still end pinned Pending:false with the complete text (the
// message_stop close lands it), never hang pending. The merged fold
// mirrors the chat reducer's replace-merge, so this is the member-visible
// result.
func TestClaudeBubblePinsAtMessageStopRealWire(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	frames := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_bdrk_01","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"short and sweet"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"id":"msg_bdrk_01","role":"assistant","content":[{"type":"thinking","thinking":"short and sweet"}]},"session_id":"sess-1","uuid":"frame-partial-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Honey never spoils"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"id":"msg_bdrk_01","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"frame-partial-2","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":1},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`,
		// the capture ends here: result frame — NO final assistant snapshot.
	}
	merged := map[string]state.ChatMsg{}
	for i, line := range frames {
		for _, e := range claudeFeed(t, ctx, line, int64(100+i)) {
			if e.Kind == state.EvChatBoss {
				merged[e.Msg.ID] = e.Msg // the chat reducer's replace-merge
			}
		}
	}
	if len(merged) != 1 {
		t.Fatalf("exactly ONE boss bubble must exist, got %+v", merged)
	}
	final := merged["bossmsg-msg_bdrk_01"]
	if final.Pending || final.Text != "Honey never spoils" {
		t.Fatalf("message_stop must pin the complete text (no final snapshot on the wire): %+v", final)
	}
}

// TestClaudePartialSnapshotStaleNeverShrinksBubble: a partial snapshot
// carrying LESS text than the deltas already grew (a stale one) must not
// shrink the rendered bubble — the refresh keeps the longest known text
// and later deltas append on top of it.
func TestClaudePartialSnapshotStaleNeverShrinksBubble(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-s1"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"the full answer"}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-s1","role":"assistant","content":[{"type":"text","text":"the full"}]},"session_id":"sess-1","uuid":"frame-stale","parent_tool_use_id":null}`, 130)
	if len(evs) != 1 || !evs[0].Msg.Pending || evs[0].Msg.Text != "the full answer" {
		t.Fatalf("a stale partial must not shrink the bubble: %+v", evs)
	}
	// deltas keep growing the SAME bubble on the longest known text
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" so far"}},"session_id":"sess-1","parent_tool_use_id":null}`, 140)
	if len(evs) != 1 || evs[0].Msg.Text != "the full answer so far" || evs[0].Msg.ID != "bossmsg-msg-s1" {
		t.Fatalf("deltas must grow the refreshed bubble: %+v", evs)
	}
}

// TestClaudePartialSnapshotMintsPendingBubbleMidStream: a mid-stream
// partial that lands before ANY text delta mints the pending bubble
// (textStart seeded at the snapshot's stamp), still unpinned; the stream's
// own deltas then grow the SAME bubble on its original stamp.
func TestClaudePartialSnapshotMintsPendingBubbleMidStream(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-m1"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-m1","role":"assistant","content":[{"type":"text","text":"first words"}]},"session_id":"sess-1","uuid":"frame-pm1","parent_tool_use_id":null}`, 120)
	if len(evs) != 1 || !evs[0].Msg.Pending || evs[0].Msg.Text != "first words" ||
		evs[0].Msg.ID != "bossmsg-msg-m1" || evs[0].Msg.At != 120 {
		t.Fatalf("a mid-stream partial must mint the pending bubble: %+v", evs)
	}
	if ctx.pinned["msg-m1"] {
		t.Fatalf("a mid-stream partial must never pin: %+v", ctx.pinned)
	}
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 130)
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":" and more"}},"session_id":"sess-1","parent_tool_use_id":null}`, 131)
	if len(evs) != 1 || evs[0].Msg.Text != "first words and more" || evs[0].Msg.At != 120 {
		t.Fatalf("deltas must grow the minted bubble on its original stamp: %+v", evs)
	}
}

// TestClaudeSnapshotOnlyPinsDoneOnFirstSight: no stream events at all
// (resumed/history frame) — the first snapshot pins done outright.
func TestClaudeSnapshotOnlyPinsDoneOnFirstSight(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-h1","role":"assistant","content":[{"type":"text","text":"from history"}]},"session_id":"sess-1","uuid":"msg-h1","parent_tool_use_id":null}`, 100)
	if len(evs) != 1 || evs[0].Kind != state.EvChatBoss || evs[0].Msg.Pending ||
		evs[0].Msg.Text != "from history" || evs[0].Msg.ID != "bossmsg-msg-h1" {
		t.Fatalf("a snapshot-only frame must pin done on first sight: %+v", evs)
	}
	if !ctx.pinned["msg-h1"] {
		t.Fatalf("snapshot-only must set the pin: %+v", ctx.pinned)
	}
}

// TestClaudeSnapshotAfterStreamClosePins: on a wire where the complete
// snapshot arrives only AFTER the stream's close (and no deltas ever grew
// text), the first post-close snapshot pins done — and stays idempotent.
func TestClaudeSnapshotAfterStreamClosePins(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-l1"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	if evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`, 130); len(evs) != 0 {
		t.Fatalf("no text grew — the close must stay silent, got %v", claudeKinds(evs))
	}
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-l1","role":"assistant","content":[{"type":"text","text":"late answer"}]},"session_id":"sess-1","uuid":"msg-l1","parent_tool_use_id":null}`, 140)
	if len(evs) != 1 || evs[0].Msg.Pending || evs[0].Msg.Text != "late answer" || evs[0].Msg.ID != "bossmsg-msg-l1" {
		t.Fatalf("the post-close snapshot must pin done: %+v", evs)
	}
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-l1","role":"assistant","content":[{"type":"text","text":"late answer"}]},"session_id":"sess-1","uuid":"msg-l1","parent_tool_use_id":null}`, 141); len(evs) != 0 {
		t.Fatalf("the pin must stay idempotent, got %v", claudeKinds(evs))
	}
}

// TestClaudeSubagentStreamNeverTouchesBossOpenMarker: a subagent's
// message_start/message_stop (parent_tool_use_id set) must not open or
// close the BOSS stream marker — a boss partial snapshot mid-subagent
// still refreshes, never pins.
func TestClaudeSubagentStreamNeverTouchesBossOpenMarker(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-boss"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	// a subagent's whole stream lifecycle interleaves
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-sub"}},"session_id":"sess-1","parent_tool_use_id":"toolu_task1"}`, 110)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":"toolu_task1"}`, 120)
	// the boss's own stream is still open: its partial snapshot refreshes
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-boss","role":"assistant","content":[{"type":"text","text":"still streaming"}]},"session_id":"sess-1","uuid":"frame-b1","parent_tool_use_id":null}`, 130)
	if len(evs) != 1 || !evs[0].Msg.Pending || evs[0].Msg.ID != "bossmsg-msg-boss" {
		t.Fatalf("the boss marker must survive subagent frames: %+v", evs)
	}
	if ctx.pinned["msg-boss"] {
		t.Fatalf("subagent frames must never close the boss stream: %+v", ctx.pinned)
	}
}

func TestClaudeThinkingStreamAndBlock(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-9"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"goals first: weekly leaderboard."}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	if len(evs) != 1 || evs[0].Kind != state.EvThought || evs[0].EmployeeName != "boss" || evs[0].Done {
		t.Fatalf("thinking delta must be an open boss thought, got %+v", evs)
	}
	// the REAL wire order (--include-partial-messages): the partial snapshot
	// lands MID-stream, BEFORE content_block_stop, sharing the message id —
	// it refreshes the transcript but must NOT close the still-open thought.
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-9","role":"assistant","content":[{"type":"thinking","thinking":"goals first: weekly leaderboard."},{"type":"text","text":"ok"}]},"session_id":"sess-1","uuid":"msg-9","parent_tool_use_id":null}`, 130)
	var thought *state.Event
	for i := range evs {
		if evs[i].Kind == state.EvThought {
			thought = &evs[i]
		}
	}
	if thought == nil || thought.Done || thought.Text != "goals first: weekly leaderboard." {
		t.Fatalf("a mid-stream partial snapshot must refresh, never close: %+v", evs)
	}
	// the stream close pins the thought done WITH the full transcript.
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 140)
	if len(evs) != 1 || evs[0].Kind != state.EvThought || !evs[0].Done ||
		evs[0].Text != "goals first: weekly leaderboard." || evs[0].CallID != "think-msg-9" {
		t.Fatalf("content_block_stop must pin the thought done with its transcript, got %+v", evs)
	}
	// a snapshot that arrives with NO open stream (older/quiet wire) still
	// pins Done=true outright.
	ctx2 := newClaudeNormCtx(nil)
	ctx2.primaryID = "sess-1"
	evs = claudeFeed(t, ctx2, `{"type":"assistant","message":{"id":"msg-10","role":"assistant","content":[{"type":"thinking","thinking":"do it the simple way"},{"type":"text","text":"ok"}]},"session_id":"sess-1","uuid":"msg-10","parent_tool_use_id":null}`, 100)
	thought = nil
	for i := range evs {
		if evs[i].Kind == state.EvThought {
			thought = &evs[i]
		}
	}
	if thought == nil || !thought.Done || thought.Text != "do it the simple way" {
		t.Fatalf("a snapshot-only thought must pin done: %+v", evs)
	}
}

// TestClaudeThinkingDeltasAccumulateGrowingTranscript pins the EvThought
// contract the chat reducer relies on (replace-on-merge by CallID): every
// delta carries the ACCUMULATED transcript, an empty keep-alive delta emits
// nothing, and the stream close carries the transcript (never a wipe).
func TestClaudeThinkingDeltasAccumulateGrowingTranscript(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-t1"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"goals first"}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	if len(evs) != 1 || evs[0].Text != "goals first" || evs[0].Done || evs[0].CallID != "think-msg-t1" {
		t.Fatalf("first delta must open the thought with its text, got %+v", evs)
	}
	// the second delta must carry the GROWING transcript, not the bare chunk
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":": weekly"}},"session_id":"sess-1","parent_tool_use_id":null}`, 130)
	if len(evs) != 1 || evs[0].Text != "goals first: weekly" || evs[0].Done {
		t.Fatalf("delta must emit the accumulated transcript, got %+v", evs)
	}
	// the real wire trails an EMPTY thinking_delta: a keep-alive, never a wipe
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`, 140)
	if len(evs) != 0 {
		t.Fatalf("an empty thinking delta must stay silent, got %+v", evs)
	}
	// the close pins Done=true WITH the transcript — a bare close would
	// replace-merge the chat entry to empty.
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`, 150)
	if len(evs) != 1 || !evs[0].Done || evs[0].Text != "goals first: weekly" {
		t.Fatalf("the close must carry the full transcript, got %+v", evs)
	}
}

// TestClaudeThinkingRealWirePartialSnapshotSequence drives the EXACT frame
// order captured off a live `claude -p --include-partial-messages` run
// (haiku, thinking on): deltas -> empty trailing delta -> PARTIAL assistant
// snapshot ([thinking] only, mid-stream) -> content_block_stop -> text block
// -> partial snapshot ([text] only) -> stops. The final merged chat state
// must show the full thought, done — the reducer fold below mirrors the
// model.go replace-on-merge rule, so this is the member-visible result.
func TestClaudeThinkingRealWirePartialSnapshotSequence(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	frames := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_bdrk_01","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"The user just"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":" sent a"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":" reminder"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		// the mid-stream PARTIAL snapshot: same message.id, thinking block only
		`{"type":"assistant","message":{"id":"msg_bdrk_01","role":"assistant","content":[{"type":"thinking","thinking":"The user just sent a reminder"}]},"session_id":"sess-1","uuid":"frame-uuid-partial-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"OK"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"assistant","message":{"id":"msg_bdrk_01","role":"assistant","content":[{"type":"text","text":"OK"}]},"session_id":"sess-1","uuid":"frame-uuid-partial-2","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":1},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`,
	}
	type thoughtState struct {
		text string
		done bool
	}
	merged := map[string]*thoughtState{}
	var allThoughtEvs []state.Event
	stopFrame := 7 // frames[7] is the thinking content_block_stop
	firstDoneFrame := -1
	for i, line := range frames {
		for _, e := range claudeFeed(t, ctx, line, int64(100+i)) {
			if e.Kind != state.EvThought {
				continue
			}
			allThoughtEvs = append(allThoughtEvs, e)
			// the chat reducer's merge: one entry per CallID, each event
			// REPLACES text/done (model.go's boss-thought arm).
			merged[e.CallID] = &thoughtState{text: e.Text, done: e.Done}
			if e.Done && firstDoneFrame < 0 {
				firstDoneFrame = i
			}
		}
	}
	final := merged["think-msg_bdrk_01"]
	if final == nil {
		t.Fatalf("no thought events at all: %v", allThoughtEvs)
	}
	if final.text != "The user just sent a reminder" || !final.done {
		t.Fatalf("final rendered thought must be the full transcript, done — got text=%q done=%v (all events: %+v)", final.text, final.done, allThoughtEvs)
	}
	// no event in the sequence may ever CARRY an empty transcript wipe
	for _, e := range allThoughtEvs {
		if e.Text == "" {
			t.Fatalf("an EvThought with empty text wipes the rendered entry: %+v", allThoughtEvs)
		}
	}
	// the premature-close guard: the partial snapshot (frame 6, BEFORE the
	// content_block_stop) must not mark the thought done mid-stream.
	if firstDoneFrame < stopFrame {
		t.Fatalf("thought closed at frame %d, before its content_block_stop (frame %d): %+v", firstDoneFrame, stopFrame, allThoughtEvs)
	}
}

func TestClaudeToolUseLifecycleWithJSONFold(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-11"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"Write","input":{}}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	if len(evs) != 1 || evs[0].Kind != state.EvTool || evs[0].ToolState != "running" || evs[0].ToolName != "write" {
		t.Fatalf("tool_use start must be a running EvTool, got %+v", evs)
	}
	if evs[0].CallID != "toolu_1" || evs[0].EmployeeName != "boss" {
		t.Fatalf("tool identity drifted: %+v", evs[0])
	}
	// partial JSON folds: the summary updates when file_path completes
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"file"}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"_path\": \"hello.html\", \"content\": \"<html>\""}},"session_id":"sess-1","parent_tool_use_id":null}`, 130)
	if len(evs) != 1 || evs[0].ToolSummary != "hello.html" {
		t.Fatalf("folded summary must pin hello.html once the field completes, got %+v", evs)
	}
	// a malformed/truncated chunk never panics and never regresses the summary
	evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"garbage\""}},"session_id":"sess-1","parent_tool_use_id":null}`, 131)
	_ = evs // any result but no panic
	// the snapshot refreshes the input (full write payload), tool still running
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-11","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Write","input":{"file_path":"hello.html","content":"<html>"}}]},"session_id":"sess-1","uuid":"msg-11","parent_tool_use_id":null}`, 140)
	for _, e := range evs {
		if e.Kind == state.EvTool && e.ToolState == "running" && e.ToolSummary != "hello.html" {
			t.Fatalf("snapshot summary drifted: %+v", e)
		}
	}
	// the result arrives in the next USER frame: done on the same CallID
	evs = claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"wrote hello.html","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 150)
	if len(evs) != 1 || evs[0].Kind != state.EvTool || evs[0].ToolState != "done" || evs[0].CallID != "toolu_1" {
		t.Fatalf("tool_result must close the CallID done, got %+v", evs)
	}
	// an errored result renders error
	ctx2 := newClaudeNormCtx(nil)
	ctx2.primaryID = "s"
	claudeFeed(t, ctx2, `{"type":"assistant","message":{"id":"m","role":"assistant","content":[{"type":"tool_use","id":"toolu_9","name":"Bash","input":{"command":"rm -rf /"}}]},"session_id":"s","uuid":"m","parent_tool_use_id":null}`, 1)
	evs = claudeFeed(t, ctx2, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_9","content":"permission denied","is_error":true}]},"session_id":"s","parent_tool_use_id":null}`, 2)
	if len(evs) != 1 || evs[0].ToolState != "error" {
		t.Fatalf("errored tool_result must close error, got %+v", evs)
	}
}

func TestClaudeSubagentLifecycle(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	// the boss runs a Task tool_use: hire + dispatch, NO plain EvTool row
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-20","role":"assistant","content":[{"type":"tool_use","id":"toolu_task1","name":"Task","input":{"description":"scan the repo","subagent_type":"Explore"}}]},"session_id":"sess-1","uuid":"msg-20","parent_tool_use_id":null}`, 100)
	kinds := claudeKinds(evs)
	if len(evs) != 2 || kinds[0] != state.EvHire || kinds[1] != state.EvDispatch {
		t.Fatalf("a Task call must hire + dispatch, got %v", kinds)
	}
	emp := evs[0].Employee
	if !strings.HasPrefix(emp.Name, "skopos-") || emp.Role != state.RoleScout {
		t.Fatalf("Explore subagent must seat a scout row, got %+v", emp)
	}
	if emp.ID != "task-toolu_task1" || evs[1].Task.Title != "scan the repo" {
		t.Fatalf("task wiring drifted: %+v %+v", emp, evs[1].Task)
	}
	// the subagent's own beats ride parent_tool_use_id -> working, never chat
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-21","role":"assistant","content":[{"type":"text","text":"reading files…"}]},"session_id":"sess-1","uuid":"msg-21","parent_tool_use_id":"toolu_task1"}`, 200)
	for _, e := range evs {
		if e.Kind == state.EvChatBoss {
			t.Fatalf("a subagent beat must never open a boss bubble: %+v", e)
		}
	}
	// its tools attribute to the employee
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-22","role":"assistant","content":[{"type":"tool_use","id":"toolu_s1","name":"Read","input":{"file_path":"src/main.go"}}]},"session_id":"sess-1","uuid":"msg-22","parent_tool_use_id":"toolu_task1"}`, 210)
	if len(evs) < 1 || evs[len(evs)-1].EmployeeName != emp.Name || evs[len(evs)-1].CallID != "toolu_s1" {
		t.Fatalf("subagent tool must be owned by its employee, got %+v", evs)
	}
	// the Task's tool_result in a main user frame RETURNS the subagent:
	// task done + returned + mail
	evs = claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_task1","content":"repo scanned: 42 files","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 300)
	kinds = claudeKinds(evs)
	if len(evs) != 4 || kinds[0] != state.EvTask || kinds[1] != state.EvReturned || kinds[2] != state.EvTool || kinds[3] != state.EvFire {
		t.Fatalf("a Task result must flip board + return + tool(done) + fire, got %v", kinds)
	}
	if evs[0].Task.Status != state.TaskDone || evs[1].Mail.Subject != "return: scan the repo" ||
		evs[1].Mail.From != emp.Name || !strings.Contains(evs[1].Mail.Body, "repo scanned") {
		t.Fatalf("return rows drifted: %+v %+v", evs[0].Task, evs[1].Mail)
	}
	if evs[2].ToolState != "done" || evs[3].EmployeeID != emp.ID {
		t.Fatalf("tool-done or fire drifted: %+v %+v", evs[2], evs[3])
	}
}

func TestClaudeAgentToolSubagentLifecycle(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	// newer Claude Code CLI emits "Agent" instead of "Task" for subagents
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-30","role":"assistant","content":[{"type":"tool_use","id":"toolu_agent1","name":"Agent","input":{"description":"research auth flow","subagent_type":"general-purpose"}}]},"session_id":"sess-1","uuid":"msg-30","parent_tool_use_id":null}`, 100)
	kinds := claudeKinds(evs)
	if len(evs) != 2 || kinds[0] != state.EvHire || kinds[1] != state.EvDispatch {
		t.Fatalf("an Agent call must hire + dispatch, got %v", kinds)
	}
	emp := evs[0].Employee
	if emp.ID != "task-toolu_agent1" || evs[1].Task.Title != "research auth flow" {
		t.Fatalf("Agent tool wiring drifted: %+v %+v", emp, evs[1].Task)
	}
	// the Agent's tool_result returns the subagent
	evs = claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_agent1","content":"auth flow documented","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 200)
	kinds = claudeKinds(evs)
	if len(evs) != 4 || kinds[0] != state.EvTask || kinds[1] != state.EvReturned || kinds[2] != state.EvTool || kinds[3] != state.EvFire {
		t.Fatalf("Agent result must flip board + return + tool(done) + fire, got %v", kinds)
	}
}

func TestClaudeCancelRequestClearsPerm(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	// simulate a permission prompt arriving via control_request can_use_tool
	evs := claudeFeed(t, ctx, `{"type":"control_request","request_id":"req-42","request":{"subtype":"can_use_tool","tool_name":"Bash","description":"run ls","input":{"command":"ls"}},"session_id":"sess-1"}`, 100)
	kinds := claudeKinds(evs)
	if len(evs) < 1 || kinds[0] != state.EvPermission {
		t.Fatalf("control_request must emit EvPermission, got %v", kinds)
	}
	// now cancel it
	evs = claudeFeed(t, ctx, `{"type":"control_cancel_request","request_id":"req-42","session_id":"sess-1"}`, 200)
	kinds = claudeKinds(evs)
	if len(evs) != 1 || kinds[0] != state.EvPermission {
		t.Fatalf("cancel must resolve the permission, got %v", kinds)
	}
	if evs[0].ToolState != "resolved" || evs[0].PermissionID != "req-42" {
		t.Fatalf("cancel resolution drifted: %+v", evs[0])
	}
	// a second cancel is a no-op
	evs = claudeFeed(t, ctx, `{"type":"control_cancel_request","request_id":"req-42","session_id":"sess-1"}`, 300)
	if len(evs) != 0 {
		t.Fatalf("double cancel must be no-op, got %v", claudeKinds(evs))
	}
}

func TestClaudeAPIRetryStatus(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := claudeFeed(t, ctx, `{"type":"system","subtype":"api_retry","attempt":2,"max_attempts":5,"retry_delay_ms":4000}`, 100)
	if len(evs) != 1 || evs[0].Kind != state.EvStatus || evs[0].Text != "[claude] retry attempt 2/5 after 4000ms" {
		t.Fatalf("api_retry line drifted: %+v", evs)
	}
}

func TestClaudeUserEchoNeverSurfacesChatUser(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	// the CLI replays our own stdin user message back — Send owns the echo,
	// so the wire user surfaces NOTHING (opencode parity).
	evs := claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello there"}]},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	if len(evs) != 0 {
		t.Fatalf("the stdin echo must stay silent, got %v", claudeKinds(evs))
	}
}

func TestClaudeInterruptedStreamFlush(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "s"
	ctx.claudeTextDelta("msg-1", "half an answer", 100)
	evs := claudeInterruptedStreamEvents(ctx, "[theboringfloor] stream interrupted", 400)
	if len(evs) != 1 || evs[0].Msg.Pending || !strings.Contains(evs[0].Msg.Text, "half an answer") ||
		!strings.Contains(evs[0].Msg.Text, "stream interrupted") || evs[0].Msg.ID != "bossmsg-msg-1" {
		t.Fatalf("interrupted flush drifted: %+v", evs)
	}
}

func TestIsSubagentTool(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"Task", true}, {"Agent", true},
		{"task", true}, {"agent", true},
		{"AGENT", true}, {"TASK", true},
		{"Read", false}, {"Bash", false}, {"Edit", false}, {"Workflow", false}, {"", false},
	} {
		if got := isSubagentTool(tc.name); got != tc.want {
			t.Errorf("isSubagentTool(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestClaudeStreamSubagentHireOnContentBlockStart(t *testing.T) {
	// A content_block_start with a subagent tool name on the boss stream
	// (parent_tool_use_id="") must emit EvHire early, before the snapshot.
	for _, toolName := range []string{"Agent", "Task"} {
		t.Run(toolName, func(t *testing.T) {
			ctx := newClaudeNormCtx(nil)
			ctx.primaryID = "sess-1"
			claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-sub"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
			evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_x1","name":"`+toolName+`","input":{}}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
			if len(evs) == 0 {
				t.Fatalf("content_block_start for %s must emit hire events", toolName)
			}
			if evs[0].Kind != state.EvHire {
				t.Fatalf("first event must be EvHire, got %v", evs[0].Kind)
			}
			if evs[0].Employee.ID != "task-toolu_x1" {
				t.Fatalf("employee ID drifted: %s", evs[0].Employee.ID)
			}
			// a duplicate content_block_start for the same tool_use is a no-op
			evs = claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_x1","name":"`+toolName+`","input":{}}},"session_id":"sess-1","parent_tool_use_id":null}`, 120)
			if len(evs) != 0 {
				t.Fatalf("duplicate hire must be silent, got %v", claudeKinds(evs))
			}
		})
	}
}

func TestClaudeStreamSubagentHireNotOnWorkerStream(t *testing.T) {
	// A subagent tool on a WORKER stream (parent_tool_use_id set) must NOT
	// trigger an early hire — it falls through to the normal tool_use path.
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-w"}},"session_id":"sess-1","parent_tool_use_id":"toolu_parent"}`, 100)
	evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_nested","name":"Agent","input":{}}},"session_id":"sess-1","parent_tool_use_id":"toolu_parent"}`, 110)
	for _, e := range evs {
		if e.Kind == state.EvHire {
			t.Fatalf("a subagent tool on a worker stream must not early-hire: %+v", e)
		}
	}
}

func TestClaudeStreamSubagentSnapshotNoDoubleHire(t *testing.T) {
	// After stream-path hire, the snapshot must NOT re-hire (no duplicate
	// EvHire/EvDispatch). The stream hire already seated the employee with
	// the "untitled brief" default; the snapshot recognises the existing task.
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg-e1"}},"session_id":"sess-1","parent_tool_use_id":null}`, 100)
	hires := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_e1","name":"Agent","input":{}}},"session_id":"sess-1","parent_tool_use_id":null}`, 110)
	if len(hires) == 0 || hires[0].Kind != state.EvHire {
		t.Fatalf("stream must hire, got %v", claudeKinds(hires))
	}
	// the snapshot carries the description — must not re-hire but MUST enrich title
	evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-e1","role":"assistant","content":[{"type":"tool_use","id":"toolu_e1","name":"Agent","input":{"description":"check migrations","subagent_type":"Explore"}}]},"session_id":"sess-1","uuid":"msg-e1","parent_tool_use_id":null}`, 120)
	for _, e := range evs {
		if e.Kind == state.EvHire {
			t.Fatalf("snapshot must not re-hire an already-seated subagent: %+v", e)
		}
	}
	// verify enrichment: the task title should now reflect the description
	var enriched bool
	for _, e := range evs {
		if e.Kind == state.EvTask && e.Task.Title == "check migrations" {
			enriched = true
		}
	}
	if !enriched {
		t.Fatalf("snapshot must enrich the task title, got %v", claudeKinds(evs))
	}
}
