// claude_wire_21247_test.go — fixture pins for shapes captured from a REAL
// claude CLI 2.1.247 wire (gated live run, Bedrock account: message/tool ids
// arrive as msg_bdrk_… / toolu_bdrk_…, NOT the msg_01…/toolu_01… shapes the
// older fixtures assume). Every fixture below was observed on that wire;
// each test locks in the mapper's DELIBERATE behavior for it so future
// fixture authors cannot drift the contract by accident:
//
//   - can_use_tool control_request envelopes carry NO session_id field at
//     all (the stub fixtures carrying "session_id":"sess-sh-1" were wrong
//     about the wire) — the mapper fills EvPermission.SessionID from the
//     pinned primary session id (the same source system/init pins).
//   - system/status, system/informational, system/thinking_tokens ride the
//     wire as protocol chatter — the open-ended parser rule ignores them:
//     ZERO events, never an error, never a panic.
//   - stream_event content_block_delta with delta.type "signature_delta"
//     (the Anthropic thinking-signature trailer) is ignored the same way.
//   - the CLI's own control_response ack of the office's initialize write
//     arrives as a frame BEFORE system/init — ignored (our own response
//     echo), and it must not pin anything.
//   - Bedrock id shapes flow through the assistant/stream mapping with
//     prefix-agnostic keying: bubbles key "bossmsg-"+message.id and tool
//     holds key the tool_use id VERBATIM, whatever the provider prefix.
package backend

import (
	"encoding/json"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// TestClaudeWire21247CanUseToolNoSessionIDFallsBackToPrimary is the FIX-A
// regression pin: the REAL 2.1.247 can_use_tool control_request envelope
// carries NO session_id field (fixture below omits the key entirely, as
// captured). After a system/init pin, the mapped EvPermission (and its
// parked hold, which AnswerPermission's resolved follow-up inherits) must
// carry the PINNED primary session id — never "". With no primary pinned
// yet, the field stays empty: the mapper fills, it does not invent.
func TestClaudeWire21247CanUseToolNoSessionIDFallsBackToPrimary(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"system","subtype":"init","cwd":"/x","session_id":"sess-live-1","model":"claude-test","mcp_servers":[],"claude_code_version":"2.1.247","uuid":"init-1"}`, 100)
	if ctx.primaryID != "sess-live-1" {
		t.Fatalf("init must pin the primary session, got %q", ctx.primaryID)
	}
	// the fixture's field set mirrors the captured envelope verbatim —
	// tool_name/display_name/input/description/permission_suggestions/
	// tool_use_id/blocked_path/decision_reason — and NO session_id key.
	evs := claudeFeed(t, ctx, `{"type":"control_request","request_id":"req-live-1","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash","input":{"command":"open hello.html"},"description":"Open the hello page","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"open hello.html"}],"behavior":"allow","destination":"localSettings"}],"tool_use_id":"toolu_bdrk_01PERM1","blocked_path":null,"decision_reason":null}}`, 110)
	if len(evs) != 1 || evs[0].Kind != state.EvPermission || evs[0].ToolState != "pending" {
		t.Fatalf("can_use_tool must surface one EvPermission pending, got %+v", evs)
	}
	pending := evs[0]
	rendered, _ := json.Marshal(pending)
	t.Logf("EvPermission with the filled SessionID: %s", rendered)
	if pending.SessionID != "sess-live-1" {
		t.Fatalf("EvPermission.SessionID = %q, want the pinned primary %q", pending.SessionID, "sess-live-1")
	}
	if pending.PermissionID != "req-live-1" || pending.ToolName != "bash" ||
		pending.ToolSummary != "Open the hello page" || pending.EmployeeName != "boss" {
		t.Fatalf("EvPermission identity drifted: %+v", pending)
	}
	// the parked hold inherits the same fill: AnswerPermission's resolved
	// follow-up event reads hold.SessionID (claude.go), so a "" here would
	// leak the same empty id into the resolved row.
	if hold := ctx.pendingPerms["req-live-1"]; hold.SessionID != "sess-live-1" {
		t.Fatalf("the parked permHold.SessionID = %q, want %q", hold.SessionID, "sess-live-1")
	}

	// no primary pinned yet -> the field stays empty (fill, never invent).
	ctx2 := newClaudeNormCtx(nil)
	evs2 := claudeFeed(t, ctx2, `{"type":"control_request","request_id":"req-live-2","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_bdrk_01PERM2"}}`, 100)
	if len(evs2) != 1 || evs2[0].SessionID != "" {
		t.Fatalf("with no primary pinned the SessionID must stay empty, got %+v", evs2)
	}
}

// TestClaudeWire21247SystemSubtypesSafeIgnored pins the deliberate
// safe-ignore contract for the system/* chatter subtypes the 2.1.247 wire
// emits: status, informational, thinking_tokens. Each must map to ZERO
// events (the open-ended parser rule), never error, never panic, and must
// not disturb the init pin.
func TestClaudeWire21247SystemSubtypesSafeIgnored(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"system","subtype":"init","cwd":"/x","session_id":"sess-live-1","model":"claude-test","mcp_servers":[],"claude_code_version":"2.1.247","uuid":"init-1"}`, 100)
	// Field shapes are the CLI 2.1.247 binary's own emitter literals
	// (strings -a): status -> {status,…compact_result/compact_error,
	// permissionMode}; informational -> {level,content,uuid};
	// thinking_tokens -> {estimated_tokens,estimated_tokens_delta,uuid,
	// session_id}.
	for name, line := range map[string]string{
		// runtime status flips (compacting on/off ride this subtype)
		"system/status": `{"type":"system","subtype":"status","status":"compacting","permissionMode":"default","session_id":"sess-live-1","uuid":"sys-status-1"}`,
		// informational notices (level + content strings)
		"system/informational": `{"type":"system","subtype":"informational","level":"info","content":"Bedrock inference profile resolved","session_id":"sess-live-1","uuid":"sys-info-1"}`,
		// thinking token bookkeeping (interleaved with the stream)
		"system/thinking_tokens": `{"type":"system","subtype":"thinking_tokens","estimated_tokens":512,"estimated_tokens_delta":64,"session_id":"sess-live-1","uuid":"sys-tt-1"}`,
	} {
		if evs := claudeFeed(t, ctx, line, 110); len(evs) != 0 {
			t.Fatalf("%s must map to ZERO events (safe-ignore), got %+v", name, evs)
		}
	}
	if ctx.primaryID != "sess-live-1" {
		t.Fatalf("chatter frames must not disturb the init pin, got %q", ctx.primaryID)
	}
}

// TestClaudeWire21247SignatureDeltaSafeIgnored pins the safe-ignore for
// stream_event content_block_delta with delta.type "signature_delta" — the
// Anthropic thinking-signature trailer the 2.1.247 wire emits after a
// thinking block. The mapper folds only text/thinking/input_json deltas;
// a signature delta must map to ZERO events and never panic.
func TestClaudeWire21247SignatureDeltaSafeIgnored(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_bdrk_01SIG1","model":"claude-test"}},"session_id":"sess-live-1","parent_tool_use_id":null}`, 100)
	claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}},"session_id":"sess-live-1","parent_tool_use_id":null}`, 110)
	if evs := claudeFeed(t, ctx, `{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"EqQBCkYIAxgCKkB3b3JrZXItYmxvY2stc2ln"}},"session_id":"sess-live-1","parent_tool_use_id":null}`, 120); len(evs) != 0 {
		t.Fatalf("a signature_delta must map to ZERO events (safe-ignore), got %+v", evs)
	}
}

// TestClaudeWire21247PreInitControlResponseAckSafeIgnored pins the
// safe-ignore for the CLI's own control_response ack of the office's
// initialize write (request_id "office-init-N", see claudeInitializeLine):
// on the real wire it lands as a frame BEFORE system/init. It must map to
// ZERO events (our own response echoes nowhere) and pin nothing.
func TestClaudeWire21247PreInitControlResponseAckSafeIgnored(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	// the ack of the office's initialize write — BEFORE any system/init.
	if evs := claudeFeed(t, ctx, `{"type":"control_response","response":{"subtype":"success","request_id":"office-init-1","response":{"supportedDialogKinds":["permission_prompt"]}},"uuid":"ack-1"}`, 100); len(evs) != 0 {
		t.Fatalf("the pre-init initialize ack must map to ZERO events, got %+v", evs)
	}
	if ctx.primaryID != "" {
		t.Fatalf("the ack must never pin a primary session, got %q", ctx.primaryID)
	}
}

// TestClaudeWire21247BedrockIDShapesPrefixAgnostic documents the Bedrock id
// shapes (msg_bdrk_… / toolu_bdrk_…) flowing through the stream/assistant
// mapping: bubble keys and tool CallIDs take the wire ids VERBATIM, whatever
// the provider prefix. (The mapper holds no msg_01/toolu_01 prefix
// assumption — this test is the pin that keeps it that way.)
func TestClaudeWire21247BedrockIDShapesPrefixAgnostic(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-live-1"
	frames := []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_bdrk_01ABC","model":"claude-test"}},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Honey never spoils"}},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_bdrk_01XYZ","name":"Bash","input":{}}},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\": \"open hello.html\"}"}},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"content_block_stop","index":1},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"stream_event","event":{"type":"message_stop"},"session_id":"sess-live-1","parent_tool_use_id":null}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_bdrk_01XYZ","content":"opened","is_error":false}]},"session_id":"sess-live-1","parent_tool_use_id":null}`,
	}
	var toolStates []string
	merged := map[string]state.ChatMsg{}
	for i, line := range frames {
		for _, e := range claudeFeed(t, ctx, line, int64(100+i)) {
			switch e.Kind {
			case state.EvChatBoss:
				merged[e.Msg.ID] = e.Msg
			case state.EvTool:
				if e.CallID != "toolu_bdrk_01XYZ" {
					t.Fatalf("the tool CallID must carry the Bedrock id verbatim, got %+v", e)
				}
				toolStates = append(toolStates, e.ToolState)
			}
		}
	}
	final, ok := merged["bossmsg-msg_bdrk_01ABC"]
	if !ok || len(merged) != 1 {
		t.Fatalf("exactly ONE bubble keyed bossmsg-msg_bdrk_01ABC must exist, got %+v", merged)
	}
	if final.Pending || final.Text != "Honey never spoils" {
		t.Fatalf("the Bedrock-keyed bubble must pin the complete text, got %+v", final)
	}
	if len(toolStates) == 0 || toolStates[len(toolStates)-1] != "done" {
		t.Fatalf("the Bedrock tool hold must run to done on the same CallID, got %v", toolStates)
	}
}
