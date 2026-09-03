// tool_output_test.go — the Event.ToolOutput contract on BOTH backends:
// the tool's RESULT text rides the done/error event (human-readable,
// plain; 8000 bytes tail-kept with a leading "…" when trimmed) so the
// chat can show what a tool returned.
//
//	claude   — user-frame tool_result content: string or block list
//	           (text joined with newlines; structured/non-text blocks
//	           render compactly, e.g. "[image data]"); is_error results
//	           carry the same content (the error styling is ToolState's
//	           business).
//	opencode — the completed tool part's state.output (REQUIRED on the
//	           wire, serve 1.18.21 GET /doc); an errored part carries
//	           its state.error text.
//
// Empty/absent output stays "" everywhere (the UI renders its own empty
// state), and running events never carry output.
package backend

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// toolEvents filters EvTool rows out of an event slice (in order).
func toolEvents(evs []state.Event) []state.Event {
	var out []state.Event
	for _, e := range evs {
		if e.Kind == state.EvTool {
			out = append(out, e)
		}
	}
	return out
}

// ---------------- claude: tool_result -> ToolOutput ----------------

// TestClaudeToolOutputTextResult: a plain string tool_result closes the
// CallID done with the result text on ToolOutput, byte-exact.
func TestClaudeToolOutputTextResult(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	run := claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o1","role":"assistant","content":[{"type":"tool_use","id":"toolu_o1","name":"Bash","input":{"command":"ls -la"}}]},"session_id":"sess-1","uuid":"msg-o1","parent_tool_use_id":null}`, 100)
	running := toolEvents(run)
	if len(running) != 1 || running[0].ToolState != "running" || running[0].ToolOutput != "" {
		t.Fatalf("the running event must carry NO output, got %+v", running)
	}
	evs := toolEvents(claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o1","content":"total 42","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 110))
	if len(evs) != 1 || evs[0].ToolState != "done" || evs[0].CallID != "toolu_o1" {
		t.Fatalf("tool_result must close the CallID done, got %+v", evs)
	}
	if evs[0].ToolOutput != "total 42" {
		t.Fatalf("ToolOutput must carry the result text verbatim, got %q", evs[0].ToolOutput)
	}
	raw, _ := json.Marshal(evs[0])
	t.Logf("claude done-event JSON (text result): %s", raw)
}

// TestClaudeToolOutputMultiBlockJoined: multi-block text content joins
// with NEWLINES (never spaces, never the summary's flattening).
func TestClaudeToolOutputMultiBlockJoined(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o2","role":"assistant","content":[{"type":"tool_use","id":"toolu_o2","name":"Read","input":{"file_path":"a.txt"}}]},"session_id":"sess-1","uuid":"msg-o2","parent_tool_use_id":null}`, 100)
	evs := toolEvents(claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o2","content":[{"type":"text","text":"alpha"},{"type":"text","text":"beta"}],"is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 110))
	if len(evs) != 1 || evs[0].ToolState != "done" {
		t.Fatalf("tool_result must close done, got %+v", evs)
	}
	if evs[0].ToolOutput != "alpha\nbeta" {
		t.Fatalf("multi-block text must join with newlines, got %q", evs[0].ToolOutput)
	}
}

// TestClaudeToolOutputErrorCarriesContent: an is_error tool_result carries
// the SAME content on ToolOutput (the UI shows it the same way; the error
// styling is ToolState's business).
func TestClaudeToolOutputErrorCarriesContent(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o3","role":"assistant","content":[{"type":"tool_use","id":"toolu_o3","name":"Bash","input":{"command":"rm -rf /"}}]},"session_id":"sess-1","uuid":"msg-o3","parent_tool_use_id":null}`, 100)
	evs := toolEvents(claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o3","content":[{"type":"text","text":"boom: permission denied"}],"is_error":true}]},"session_id":"sess-1","parent_tool_use_id":null}`, 110))
	if len(evs) != 1 || evs[0].ToolState != "error" {
		t.Fatalf("is_error must close the CallID error, got %+v", evs)
	}
	if evs[0].ToolOutput != "boom: permission denied" {
		t.Fatalf("an error row must carry the same content, got %q", evs[0].ToolOutput)
	}
}

// TestClaudeToolOutputCapsTailKept: a 10KB result caps at 8000 bytes,
// tail-kept (errors and exits live at the tail), "…"-marked at the seam.
func TestClaudeToolOutputCapsTailKept(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	big := "OFFICE-HEAD-MARKER" + strings.Repeat("x", 10000) + "OFFICE-TAIL-MARKER"
	want := "…" + big[len(big)-(toolOutputCapBytes-len("…")):]
	claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o4","role":"assistant","content":[{"type":"tool_use","id":"toolu_o4","name":"Bash","input":{"command":"flood"}}]},"session_id":"sess-1","uuid":"msg-o4","parent_tool_use_id":null}`, 100)
	evs := toolEvents(claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o4","content":"`+big+`","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 110))
	if len(evs) != 1 || evs[0].ToolState != "done" {
		t.Fatalf("tool_result must close done, got %+v", evs)
	}
	got := evs[0].ToolOutput
	if got != want {
		t.Fatalf("the cap must be byte-exact tail-kept:\n got len=%d head=%q tail=%q\nwant len=%d head=%q tail=%q",
			len(got), got[:24], got[len(got)-24:], len(want), want[:24], want[len(want)-24:])
	}
	if len(got) != toolOutputCapBytes || !strings.HasPrefix(got, "…") ||
		!strings.HasSuffix(got, "OFFICE-TAIL-MARKER") || strings.Contains(got, "OFFICE-HEAD-MARKER") {
		t.Fatalf("tail-kept cap invariants broken: len=%d head=%q tail=%q", len(got), got[:24], got[len(got)-24:])
	}
	t.Logf("cap trimmed string: len=%d head=%q tail=%q", len(got), got[:32], got[len(got)-32:])
}

// TestClaudeToolOutputNonTextPlaceholder: structured/non-text content
// renders compactly — an image block becomes "[image data]", never the
// raw payload — and keeps its newline seat among text blocks.
func TestClaudeToolOutputNonTextPlaceholder(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o5","role":"assistant","content":[{"type":"tool_use","id":"toolu_o5","name":"Read","input":{"file_path":"p.png"}}]},"session_id":"sess-1","uuid":"msg-o5","parent_tool_use_id":null}`, 100)
	evs := toolEvents(claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o5","content":[{"type":"text","text":"done"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}],"is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 110))
	if len(evs) != 1 || evs[0].ToolState != "done" {
		t.Fatalf("tool_result must close done, got %+v", evs)
	}
	if evs[0].ToolOutput != "done\n[image data]" {
		t.Fatalf("non-text content must render compactly, got %q", evs[0].ToolOutput)
	}
	if strings.Contains(evs[0].ToolOutput, "iVBORw0KGgo=") {
		t.Fatalf("the raw image payload must NEVER leak into ToolOutput")
	}
}

// TestClaudeToolOutputTaskResultCarriesOutput: the contract says ANY
// tool — a Task tool's return also carries its result text on the done
// row (alongside the return mail, which keeps its own 240-cap body).
func TestClaudeToolOutputTaskResultCarriesOutput(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	claudeFeed(t, ctx, `{"type":"assistant","message":{"id":"msg-o6","role":"assistant","content":[{"type":"tool_use","id":"toolu_o6","name":"Task","input":{"description":"scan the repo","subagent_type":"Explore"}}]},"session_id":"sess-1","uuid":"msg-o6","parent_tool_use_id":null}`, 100)
	evs := claudeFeed(t, ctx, `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_o6","content":"repo scanned: 42 files","is_error":false}]},"session_id":"sess-1","parent_tool_use_id":null}`, 200)
	var done *state.Event
	for i, e := range evs {
		if e.Kind == state.EvTool && e.ToolState == "done" {
			done = &evs[i]
		}
	}
	if done == nil {
		t.Fatalf("a Task return must emit the tool done row, got %v", claudeKinds(evs))
	}
	if done.ToolOutput != "repo scanned: 42 files" {
		t.Fatalf("the Task tool's done row must carry its result text, got %q", done.ToolOutput)
	}
}

// ---------------- opencode: tool part state.output -> ToolOutput ----------------

// TestOCToolOutputCompleted: a completed bash part's state.output (the
// REQUIRED ToolStateCompleted field, serve 1.18.21 /doc) rides the done
// event verbatim.
func TestOCToolOutputCompleted(t *testing.T) {
	ctx := newNormCtx(nil)
	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-o1","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-o1","tool":"bash",`+
			`"state":{"status":"completed","title":"ls","input":{"command":"ls"},"output":"total 42\n","metadata":{},"time":{"start":1,"end":2}}}}`),
		ctx, "ses-primary", 1000)
	tools := toolEvents(evs)
	if len(tools) != 1 || tools[0].ToolState != "done" || tools[0].CallID != "call-o1" {
		t.Fatalf("a completed part must emit exactly one done EvTool, got %+v", evs)
	}
	if tools[0].ToolOutput != "total 42\n" {
		t.Fatalf("ToolOutput must carry state.output verbatim, got %q", tools[0].ToolOutput)
	}
	raw, _ := json.Marshal(tools[0])
	t.Logf("opencode done-event JSON (completed): %s", raw)
}

// TestOCToolOutputErrorCarriesError: an errored part carries its
// state.error text on ToolOutput (same show-me contract; the error
// styling is ToolState's business).
func TestOCToolOutputErrorCarriesError(t *testing.T) {
	ctx := newNormCtx(nil)
	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-o2","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-o2","tool":"bash",`+
			`"state":{"status":"error","input":{"command":"rm -rf /"},"error":"boom: permission denied","time":{"start":1,"end":2}}}}`),
		ctx, "ses-primary", 1000)
	tools := toolEvents(evs)
	if len(tools) != 1 || tools[0].ToolState != "error" {
		t.Fatalf("an errored part must emit exactly one error EvTool, got %+v", evs)
	}
	if tools[0].ToolOutput != "boom: permission denied" {
		t.Fatalf("an error row must carry the state's error text, got %q", tools[0].ToolOutput)
	}
}

// TestOCToolOutputCapsTailKept: a 10KB state.output caps at 8000 bytes,
// tail-kept, "…"-marked — multi-megabyte outputs never grow memory.
func TestOCToolOutputCapsTailKept(t *testing.T) {
	ctx := newNormCtx(nil)
	big := "OFFICE-HEAD-MARKER" + strings.Repeat("y", 10000) + "OFFICE-TAIL-MARKER"
	want := "…" + big[len(big)-(toolOutputCapBytes-len("…")):]
	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-o3","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-o3","tool":"bash",`+
			`"state":{"status":"completed","title":"flood","input":{"command":"flood"},"output":"`+big+`","metadata":{},"time":{"start":1,"end":2}}}}`),
		ctx, "ses-primary", 1000)
	tools := toolEvents(evs)
	if len(tools) != 1 {
		t.Fatalf("a completed part must emit exactly one EvTool, got %+v", evs)
	}
	got := tools[0].ToolOutput
	if got != want {
		t.Fatalf("the cap must be byte-exact tail-kept:\n got len=%d head=%q tail=%q\nwant len=%d head=%q tail=%q",
			len(got), got[:24], got[len(got)-24:], len(want), want[:24], want[len(want)-24:])
	}
	if len(got) != toolOutputCapBytes || !strings.HasPrefix(got, "…") ||
		!strings.HasSuffix(got, "OFFICE-TAIL-MARKER") || strings.Contains(got, "OFFICE-HEAD-MARKER") {
		t.Fatalf("tail-kept cap invariants broken: len=%d", len(got))
	}
	t.Logf("cap trimmed string: len=%d head=%q tail=%q", len(got), got[:32], got[len(got)-32:])
}

// TestOCToolOutputEmptyAndRunningStayEmpty: the empty-state contract — a
// completed part with an EMPTY output (or the key absent entirely, an
// older serve) and a RUNNING part all keep ToolOutput "" (the UI renders
// its own empty state).
func TestOCToolOutputEmptyAndRunningStayEmpty(t *testing.T) {
	ctx := newNormCtx(nil)
	for name, frame := range map[string]string{
		"empty-output": `{"part":{"id":"part-e1","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-e1","tool":"read",` +
			`"state":{"status":"completed","title":"a.txt","input":{"filePath":"a.txt"},"output":"","metadata":{},"time":{"start":1,"end":2}}}}`,
		"absent-output": `{"part":{"id":"part-e2","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-e2","tool":"read",` +
			`"state":{"status":"completed","title":"a.txt","input":{"filePath":"a.txt"},"metadata":{},"time":{"start":1,"end":2}}}}`,
		"running": `{"part":{"id":"part-e3","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-e3","tool":"bash",` +
			`"state":{"status":"running","input":{"command":"sleep 1"},"time":{"start":1}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			evs := mapOCEvent(partUpdated(t, frame), ctx, "ses-primary", 1000)
			tools := toolEvents(evs)
			if len(tools) != 1 || tools[0].ToolOutput != "" {
				t.Fatalf("%s must keep ToolOutput empty, got %+v", name, tools)
			}
		})
	}
}

// TestCapToolOutputRuneSafe: the tail seam never splits a multi-byte
// rune — a flood of 3-byte runes capped mid-rune still yields valid
// UTF-8 within budget.
func TestCapToolOutputRuneSafe(t *testing.T) {
	flood := strings.Repeat("a", 10) + strings.Repeat("€", 4000) // 12010 bytes
	got := capToolOutput(flood)
	if !utf8.ValidString(got) {
		t.Fatalf("a mid-rune seam must walk to a boundary, got invalid UTF-8")
	}
	if len(got) > toolOutputCapBytes || !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "€") {
		t.Fatalf("cap invariants broken: len=%d", len(got))
	}
	if exact := strings.Repeat("z", toolOutputCapBytes); capToolOutput(exact) != exact {
		t.Fatalf("exactly-at-cap must pass through untouched")
	}
}
