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
	// the snapshot pins the FINAL text on the same id, once, Pending=false
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-7","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"msg-7","parent_tool_use_id":null}`, 140)
	if len(evs) != 1 || evs[0].Msg.ID != "bossmsg-msg-7" || evs[0].Msg.Pending || evs[0].Msg.Text != "Honey never spoils" {
		t.Fatalf("snapshot must pin the same bubble id, got %+v", evs)
	}
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-7","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"msg-7","parent_tool_use_id":null}`, 141); len(evs) != 0 {
		t.Fatalf("a duplicate snapshot re-emitted %v", claudeKinds(evs))
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
	evs = claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-9","role":"assistant","content":[{"type":"thinking","thinking":"do it the simple way"},{"type":"text","text":"ok"}]},"session_id":"sess-1","uuid":"msg-9","parent_tool_use_id":null}`, 130)
	var thought *state.Event
	for i := range evs {
		if evs[i].Kind == state.EvThought {
			thought = &evs[i]
		}
	}
	if thought == nil || !thought.Done || thought.Text != "do it the simple way" {
		t.Fatalf("assistant thinking block must pin a done thought: %+v", evs)
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
	if len(evs) != 2 || kinds[0] != state.EvTask || kinds[1] != state.EvReturned {
		t.Fatalf("a Task result must flip the board + return, got %v", kinds)
	}
	if evs[0].Task.Status != state.TaskDone || evs[1].Mail.Subject != "return: scan the repo" ||
		evs[1].Mail.From != emp.Name || !strings.Contains(evs[1].Mail.Body, "repo scanned") {
		t.Fatalf("return rows drifted: %+v %+v", evs[0].Task, evs[1].Mail)
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
	evs := claudeInterruptedStreamEvents(ctx, "[theboringoffice] stream interrupted", 400)
	if len(evs) != 1 || evs[0].Msg.Pending || !strings.Contains(evs[0].Msg.Text, "half an answer") ||
		!strings.Contains(evs[0].Msg.Text, "stream interrupted") || evs[0].Msg.ID != "bossmsg-msg-1" {
		t.Fatalf("interrupted flush drifted: %+v", evs)
	}
}
