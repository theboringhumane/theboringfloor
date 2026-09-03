// claude_perm_test.go — the control_request perm/question round-trip
// against the REAL claude CLI wire shape (verified against the installed
// CLI 2.1.247 binary): the can_use_tool request carries subtype/tool_name/
// input/description/permission_suggestions/tool_use_id (there is NO
// tool_input / input_preview), and the office's permission enum maps onto
// the CLI's permission-result vocabulary — once → {"behavior":"allow"},
// always → {"behavior":"allow","updatedPermissions":<the request's
// permission_suggestions verbatim>} (there is NO "allow_always" on the
// wire; plain allow when suggestions are absent), reject →
// {"behavior":"deny","message":…}. The request_user_dialog REQUEST rides
// dialog_kind + payload.questions (the AskUserQuestion kind is
// "permission_ask_user_question"; there are NO flat question/options
// fields), and its reply rides the dialog vocabulary instead — answer →
// {"behavior":"completed","result":…}, dismiss → {"behavior":"cancelled"}
// (the CLI's own cancelDialogByMachine emits exactly that). EVERY
// control_response envelope carries subtype:"success" (the CLI's own
// writers always do; a schema-parse failure rejects the parked promise
// and the CLI converts it to a deny).
package backend

import (
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// TestClaudeControlRequestRoundTrip drives one shell stub that asks for
// tool permissions and a dialog answer, and proves every writer's exact
// stdin bytes against the CLI's real control_response schema.
func TestClaudeControlRequestRoundTrip(t *testing.T) {
	capture := tempCaptureLog(t)
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      # a perm WITH the CLI's own description + permission_suggestions …
      printf '%s\n' '{"type":"control_request","request_id":"req-0001","request":{"subtype":"can_use_tool","tool_name":"Bash","display_name":"Bash","input":{"command":"open hello.html"},"description":"Open the hello page","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"open hello.html"}],"behavior":"allow","destination":"localSettings"}],"tool_use_id":"toolu_01AAA","blocked_path":null,"decision_reason":null},"session_id":"sess-sh-1"}'
      # … a perm with NO description (the input-summary fallback) …
      printf '%s\n' '{"type":"control_request","request_id":"req-0003","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"open hello.html"},"tool_use_id":"toolu_01BBB"},"session_id":"sess-sh-1"}'
      # … a perm to REJECT (Write, description + suggestions) …
      printf '%s\n' '{"type":"control_request","request_id":"req-0004","request":{"subtype":"can_use_tool","tool_name":"Write","display_name":"Write","input":{"file_path":"/tmp/wall.txt","content":"matte"},"description":"Write the wall file","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Write","ruleContent":"/tmp/wall.txt"}],"behavior":"allow","destination":"localSettings"}],"tool_use_id":"toolu_01CCC"},"session_id":"sess-sh-1"}'
      # … a perm with NO suggestions ("always" must fall back to plain allow) …
      printf '%s\n' '{"type":"control_request","request_id":"req-0005","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls -la"},"description":"List the directory","tool_use_id":"toolu_01DDD"},"session_id":"sess-sh-1"}'
      # … and a dialog in the same turn (the REAL wire shape: dialog_kind
      # + payload.questions, NEVER flat question/options)
      printf '%s\n' '{"type":"control_request","request_id":"req-0002","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01EEE","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":false}]},"tool_use_id":"toolu_01EEE"},"session_id":"sess-sh-1"}'
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
	claudeWait(t, "all five control requests mapped", 3*time.Second, func() bool {
		var p1, p3, p4, p5, question bool
		for _, e := range log.snapshot() {
			if e.Kind == state.EvPermission && e.ToolState == "pending" {
				switch e.PermissionID {
				case "req-0001":
					p1 = true
				case "req-0003":
					p3 = true
				case "req-0004":
					p4 = true
				case "req-0005":
					p5 = true
				}
			}
			if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "pending" {
				question = true
			}
		}
		return p1 && p3 && p4 && p5 && question
	})

	// the pending perms carry the REAL wire identity: the modal summary is
	// the CLI's description line, falling back to a summary of `input`.
	for _, e := range log.snapshot() {
		if e.Kind != state.EvPermission || e.ToolState != "pending" {
			continue
		}
		switch e.PermissionID {
		case "req-0001":
			if e.EmployeeName != "boss" || e.ToolName != "bash" || e.ToolSummary != "Open the hello page" {
				t.Fatalf("EvPermission (description) identity drifted: %+v", e)
			}
		case "req-0003":
			if e.EmployeeName != "boss" || e.ToolName != "bash" || e.ToolSummary != "open hello.html" {
				t.Fatalf("EvPermission (input fallback) identity drifted: %+v", e)
			}
		case "req-0004":
			if e.ToolName != "write" || e.ToolSummary != "Write the wall file" {
				t.Fatalf("EvPermission (reject target) identity drifted: %+v", e)
			}
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" {
			if e.Text != "Which finish for the east wall?" || !strings.Contains(e.ToolSummary, "matte") ||
				len(e.Questions) != 1 || len(e.Questions[0].Options) != 3 {
				t.Fatalf("EvQuestion identity drifted: %+v", e)
			}
		}
	}

	// always → allow + updatedPermissions (the suggestions verbatim)
	if err := b.AnswerPermission("req-0001", "always"); err != nil {
		t.Fatalf("AnswerPermission(always): %v", err)
	}
	// once → plain allow
	if err := b.AnswerPermission("req-0003", "once"); err != nil {
		t.Fatalf("AnswerPermission(once): %v", err)
	}
	// reject → deny + message
	if err := b.AnswerPermission("req-0004", "reject"); err != nil {
		t.Fatalf("AnswerPermission(reject): %v", err)
	}
	// always with NO suggestions → plain allow (the sane fallback)
	if err := b.AnswerPermission("req-0005", "always"); err != nil {
		t.Fatalf("AnswerPermission(always, no suggestions): %v", err)
	}
	// invalid enum never writes
	if err := b.AnswerPermission("req-0001", "sometimes"); err == nil {
		t.Fatalf("an invalid response must be refused")
	}
	// question: the answer text rides the dialog response (behavior completed)
	if err := b.AnswerQuestion("req-0002", [][]string{{"matte"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	if err := b.RejectQuestion("req-0002"); err != nil {
		t.Fatalf("RejectQuestion: %v", err)
	}

	claudeWait(t, "all six control_response lines written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 7 // 1 user + 6 responses
	})
	lines := claudeCapture(t, capture)
	wants := []string{
		// always → allow + the fixture's permission_suggestions verbatim
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0001","response":{"behavior":"allow","updatedPermissions":[{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"open hello.html"}],"behavior":"allow","destination":"localSettings"}]}}}`,
		// once → plain allow
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0003","response":{"behavior":"allow"}}}`,
		// reject → deny + message
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0004","response":{"behavior":"deny","message":"Denied by the boss in theboringoffice"}}}`,
		// always without suggestions → plain allow (fallback)
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0005","response":{"behavior":"allow"}}}`,
		// dialog answer / dismiss (the request_user_dialog vocabulary:
		// completed + result, cancelled — NEVER allow/deny/answer at the
		// envelope level). RE-KEYED (2.1.247 binary): the AskUserQuestion
		// dialog's result is the CLI-native OBJECT {behavior:"allow",
		// updatedInput:{questions,answers}} — the kind's result validator
		// requires an object with a "behavior" key; the old bare string
		// "matte" would fail safeParse and settle the dialog cancelled.
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0002","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":false}],"answers":{"Which finish for the east wall?":"matte"}}}}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"req-0002","response":{"behavior":"cancelled"}}}`,
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
	// every response line is schema-conformant: subtype:"success" present,
	// behavior only ever allow|deny (perms) or completed|cancelled (dialogs),
	// never the phantom "allow_always" nor a dialog riding perm vocabulary.
	for _, got := range lines {
		if !strings.Contains(got, `"type":"control_response"`) {
			continue
		}
		if !strings.Contains(got, `"subtype":"success"`) {
			t.Fatalf("control_response lacks subtype:\"success\": %q", got)
		}
		if strings.Contains(got, "allow_always") {
			t.Fatalf("allow_always does not exist on the claude wire: %q", got)
		}
		if strings.Contains(got, `"answer"`) {
			t.Fatalf("dialog answers ride result, there is no answer key on the claude wire: %q", got)
		}
	}
	// every answer pair echoed a local resolved event (modal closer)
	var permResolved, qAnswered, qRejected int
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission && e.ToolState == "resolved" {
			permResolved++
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "resolved" && e.ToolSummary == "answered" {
			qAnswered++
		}
		if e.Kind == state.EvQuestion && e.QuestionID == "req-0002" && e.ToolState == "resolved" && e.ToolSummary == "rejected" {
			qRejected++
		}
	}
	if permResolved != 4 || qAnswered != 1 || qRejected != 1 {
		t.Fatalf("resolved events drifted: perm=%d/4 answered=%d/1 rejected=%d/1", permResolved, qAnswered, qRejected)
	}
}

// tempCaptureLog reserves a fresh capture path (created lazily by the stub
// append when the first stdin line lands).
func tempCaptureLog(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(t.TempDir(), " ", "_") + "/capture.log"
}
