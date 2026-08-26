// claude_perm_test.go — the control_request perm/question round-trip:
// the office's permission enum maps onto the claude control namespace
// (once → "allow", always → "allow" +always semantics "allow_always",
// reject → "deny"), the response carries the request_id back, and the
// local resolved event lands when the write completes.
package backend

import (
	"github.com/theboringhumane/theboringoffice/internal/state"
	"strings"
	"testing"
	"time"
)

// TestClaudeControlRequestRoundTrip drives one shell stub that asks for a
// tool permission and a dialog answer, and proves every writer's exact
// stdin bytes.
func TestClaudeControlRequestRoundTrip(t *testing.T) {
	capture := tempCaptureLog(t)
	stubBody := `printf '%s\n' '` + claudeStubInitLine[:len(claudeStubInitLine)-1] + `'` + `
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      # the turn asks for a tool …
      printf '%s\n' '{"type":"control_request","request_id":"req-0001","request":{"subtype":"can_use_tool","tool_name":"Bash","tool_input":{"command":"open hello.html"},"input_preview":"open hello.html"},"session_id":"sess-sh-1"}'
      # … and a dialog in the same turn
      printf '%s\n' '{"type":"control_request","request_id":"req-0002","request":{"subtype":"request_user_dialog","question":"Which finish for the east wall?","options":["matte","gloss","both"]},"session_id":"sess-sh-1"}'
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

	if err := b.Send("ship the wall"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "both control requests mapped", 3*time.Second, func() bool {
		var perm, question bool
		for _, e := range log.snapshot() {
			if e.Kind == state.EvPermission && e.PermissionID == "req-0001" && e.ToolState == "pending" {
				perm = true
			}
			if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "pending" {
				question = true
			}
		}
		return perm && question
	})

	// the pending perm carries the wire identity
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission && e.PermissionID == "req-0001" {
			if e.EmployeeName != "boss" || e.ToolName != "bash" || e.ToolSummary != "open hello.html" {
				t.Fatalf("EvPermission identity drifted: %+v", e)
			}
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" {
			if e.Text != "Which finish for the east wall?" || !strings.Contains(e.ToolSummary, "matte") ||
				len(e.Questions) != 1 || len(e.Questions[0].Options) != 3 {
				t.Fatalf("EvQuestion identity drifted: %+v", e)
			}
		}
	}

	// once → {"behavior":"allow"}
	if err := b.AnswerPermission("req-0001", "once"); err != nil {
		t.Fatalf("AnswerPermission(once): %v", err)
	}
	// always → {"behavior":"allow_always"}
	if err := b.AnswerPermission("req-0001", "always"); err != nil {
		t.Fatalf("AnswerPermission(always): %v", err)
	}
	// reject → {"behavior":"deny"}
	if err := b.AnswerPermission("req-0001", "reject"); err != nil {
		t.Fatalf("AnswerPermission(reject): %v", err)
	}
	// invalid enum never writes
	if err := b.AnswerPermission("req-0001", "sometimes"); err == nil {
		t.Fatalf("an invalid response must be refused")
	}
	// question: the answer text rides the dialog response (behavior allow)
	if err := b.AnswerQuestion("req-0002", [][]string{{"matte"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	if err := b.RejectQuestion("req-0002"); err != nil {
		t.Fatalf("RejectQuestion: %v", err)
	}

	claudeWait(t, "all five control_response lines written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 6 // 1 user + 5 responses
	})
	lines := claudeCapture(t, capture)
	wants := []string{
		`{"type":"control_response","response":{"request_id":"req-0002","response":{"behavior":"allow","answer":"matte"}}}`,
		`{"type":"control_response","response":{"request_id":"req-0002","response":{"behavior":"deny"}}}`,
		`{"type":"control_response","response":{"request_id":"req-0001","response":{"behavior":"allow"}}}`,
		`{"type":"control_response","response":{"request_id":"req-0001","response":{"behavior":"allow_always"}}}`,
		`{"type":"control_response","response":{"request_id":"req-0001","response":{"behavior":"deny"}}}`,
	}
	for _, want := range wants {
		found := false
		for _, got := range lines {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("control_response missing %q in capture:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	// every answer pair echoed a local resolved event (modal closer)
	var permResolved, qAnswered, qRejected int
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission && e.PermissionID == "req-0001" && e.ToolState == "resolved" {
			permResolved++
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "resolved" && e.ToolSummary == "answered" {
			qAnswered++
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "resolved" && e.ToolSummary == "rejected" {
			qRejected++
		}
	}
	if permResolved != 3 || qAnswered != 1 || qRejected != 1 {
		t.Fatalf("resolved events drifted: perm=%d/3 answered=%d/1 rejected=%d/1", permResolved, qAnswered, qRejected)
	}
}

// tempCaptureLog reserves a fresh capture path (created lazily by the stub
// append when the first stdin line lands).
func tempCaptureLog(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(t.TempDir(), " ", "_") + "/capture.log"
}
