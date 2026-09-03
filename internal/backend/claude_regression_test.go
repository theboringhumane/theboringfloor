// claude_regression_test.go — the three production-bug regression pins for
// the claude event-mapping pipeline, each with the exact identifiers and
// end-to-end evidence the member's diagnosis called out:
//
//  1. DOUBLED BOSS BUBBLE: stream deltas key the pending bubble off the
//     message_start's inner.Message.ID ("msg_01REAL"); the assistant-frame
//     pin MUST key off raw.Message.ID too. Production Claude Code frames
//     carry a frame-level uuid ("sess-uuid-9") that DIVERGES from
//     message.id — a UUID-first pin mints a SECOND bubble with the same
//     text (every paragraph doubled). The merged fold below mirrors the
//     chat reducer's replace-on-merge, so it is the member-visible result:
//     exactly ONE bubble, complete text.
//  2. PERMISSION ASKS NEVER SURFACING: a can_use_tool control_request must
//     map to a state.EvPermission the office modal consumes (printed
//     below), and the member's answer must land on claude's stdin as the
//     exact control_response bytes (stub-captured, byte-asserted).
//  3. THINKING NEVER VISIBLE: thinking_delta frames must emit EvThought
//     Done=false rows carrying the ACCUMULATED transcript, then a Done=true
//     close carrying the full text — shape-asserted event by event.
package backend

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// TestClaudeRegressBossBubbleMessageIDBeatsFrameUUID is bug 1's regression:
// the assistant snapshot's frame uuid ("sess-uuid-9") diverges from its
// message.id ("msg_01REAL") — the production Claude Code wire shape the old
// UUID-first pin forked on. The pin must REPLACE the streamed bubble:
// exactly ONE bubble under "bossmsg-msg_01REAL", full text, Pending:false,
// and "sess-uuid-9" must never key a bubble nor a pin.
func TestClaudeRegressBossBubbleMessageIDBeatsFrameUUID(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	frames := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_01REAL","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Honey never"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" spoils"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		// the mid-stream PARTIAL snapshot: frame uuid DIVERGES from message.id
		`{"type":"assistant","message":{"id":"msg_01REAL","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"sess-uuid-9","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-1","parent_tool_use_id":null}`,
	}
	merged := map[string]state.ChatMsg{}
	for i, line := range frames {
		for _, e := range claudeFeed(t, ctx, line, int64(100+i)) {
			if e.Kind != state.EvChatBoss {
				continue
			}
			t.Logf("frame %d -> EvChatBoss id=%s pending=%v text=%q", i, e.Msg.ID, e.Msg.Pending, e.Msg.Text)
			merged[e.Msg.ID] = e.Msg // the chat reducer's replace-merge fold
		}
	}
	if len(merged) != 1 {
		t.Fatalf("exactly ONE boss bubble must exist (the doubled-bubble bug forks the fold), got %d: %+v", len(merged), merged)
	}
	final, ok := merged["bossmsg-msg_01REAL"]
	if !ok {
		t.Fatalf("the ONE bubble must key off message.id (msg_01REAL), got fold keys: %+v", merged)
	}
	if final.Pending || final.Text != "Honey never spoils" {
		t.Fatalf("the pin must REPLACE the streamed bubble with the complete text, got %+v", final)
	}
	for id := range merged {
		if strings.Contains(id, "sess-uuid-9") {
			t.Fatalf("the frame uuid must never key a bubble: %+v", merged)
		}
	}
	if !ctx.pinned["msg_01REAL"] || ctx.pinned["sess-uuid-9"] {
		t.Fatalf("pin keys drifted (message.id must pin, frame uuid must not): %+v", ctx.pinned)
	}
	// a LATE complete snapshot, still carrying the divergent frame uuid,
	// must be absorbed silently — never mint a second bubble post-pin.
	if evs := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg_01REAL","role":"assistant","content":[{"type":"text","text":"Honey never spoils"}]},"session_id":"sess-1","uuid":"sess-uuid-9","parent_tool_use_id":null}`, 200); len(evs) != 0 {
		t.Fatalf("a post-pin snapshot with a divergent frame uuid must stay silent, got %+v", evs)
	}
}

// TestClaudeRegressPermAskSurfacesEventAndAnswersStdin is bug 2's regression:
// a can_use_tool control_request surfaces as a state.EvPermission pending
// (the event the office modal consumes — printed below), and the boss's
// answer lands on claude stdin as the exact control_response line.
func TestClaudeRegressPermAskSurfacesEventAndAnswersStdin(t *testing.T) {
	capture := tempCaptureLog(t)
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"req-perm-9","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash","input":{"command":"open hello.html"},"description":"Open the hello page","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"open hello.html"}],"behavior":"allow","destination":"localSettings"}],"tool_use_id":"toolu_01PERM9","blocked_path":null,"decision_reason":null},"session_id":"sess-sh-1"}'
      ;;
  esac
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if err := b.Send("open the hello page"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// the pending EvPermission surfaces (the member-visible modal trigger).
	claudeWait(t, "the can_use_tool EvPermission pending event", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvPermission && e.PermissionID == "req-perm-9" && e.ToolState == "pending" {
				return true
			}
		}
		return false
	})
	var pending *state.Event
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission && e.PermissionID == "req-perm-9" && e.ToolState == "pending" {
			ev := e
			pending = &ev
		}
	}
	if pending == nil {
		t.Fatalf("can_use_tool must surface an EvPermission pending (the modal trigger); events: %+v", log.snapshot())
	}
	rendered, _ := json.Marshal(pending)
	t.Logf("EvPermission the office consumes: %s", rendered)
	if pending.EmployeeName != "boss" || pending.ToolName != "bash" ||
		pending.ToolSummary != "Open the hello page" || pending.SessionID != "sess-sh-1" {
		t.Fatalf("EvPermission identity drifted: %+v", pending)
	}

	// the boss answers ONCE -> the exact control_response bytes hit stdin.
	if err := b.AnswerPermission("req-perm-9", "once"); err != nil {
		t.Fatalf("AnswerPermission(once): %v", err)
	}
	claudeWait(t, "the control_response stdin line", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2 // 1 user + 1 response
	})
	lines := claudeCapture(t, capture)
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"req-perm-9","response":{"behavior":"allow"}}}`
	if lines[len(lines)-1] != want {
		t.Fatalf("stdin control_response drifted:\n got %q\nwant %q", lines[len(lines)-1], want)
	}
	t.Logf("stdin control_response bytes: %s", lines[len(lines)-1])
	// the local resolved event closes the modal (claude never echoes).
	var resolved bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission && e.PermissionID == "req-perm-9" && e.ToolState == "resolved" {
			resolved = true
		}
	}
	if !resolved {
		t.Fatalf("the answer must emit the local resolved event (modal closer)")
	}
}

// TestClaudeRegressThinkingSequenceOpenAccumulateClose is bug 3's
// regression: a thinking_delta sequence must emit EvThought Done=false rows
// carrying the ACCUMULATED transcript (the chat reducer replaces-on-merge
// by CallID — a bare chunk would shrink the rendered thought), then a
// Done=true close carrying the full text, all under a stable CallID.
func TestClaudeRegressThinkingSequenceOpenAccumulateClose(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-1"
	frames := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_01THINK","model":"claude-test"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"goals first"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":": weekly"}},"session_id":"sess-1","parent_tool_use_id":null}`,
		// the real wire trails an EMPTY thinking_delta: a keep-alive, never a wipe
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":""}},"session_id":"sess-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-1","parent_tool_use_id":null}`,
	}
	type thoughtRow struct {
		CallID string
		Done   bool
		Text   string
	}
	var rows []thoughtRow
	for i, line := range frames {
		for _, e := range claudeFeed(t, ctx, line, int64(100+i)) {
			if e.Kind != state.EvThought {
				continue
			}
			t.Logf("frame %d -> EvThought callID=%s done=%v text=%q", i, e.CallID, e.Done, e.Text)
			rows = append(rows, thoughtRow{CallID: e.CallID, Done: e.Done, Text: e.Text})
			if e.EmployeeID != "boss" || e.EmployeeName != "boss" {
				t.Fatalf("the boss's thought must be owned by the boss: %+v", e)
			}
		}
	}
	want := []thoughtRow{
		{CallID: "think-msg_01THINK", Done: false, Text: "goals first"},
		{CallID: "think-msg_01THINK", Done: false, Text: "goals first: weekly"},
		{CallID: "think-msg_01THINK", Done: true, Text: "goals first: weekly"},
	}
	if len(rows) != len(want) {
		t.Fatalf("thinking sequence drifted — want %d EvThought events, got %d: %+v", len(want), len(rows), rows)
	}
	for i, w := range want {
		if rows[i] != w {
			t.Fatalf("EvThought[%d] drifted:\n got %+v\nwant %+v\nfull sequence: %+v", i, rows[i], w, rows)
		}
	}
}
