// claude_dialog_kinds_test.go — the 28-kind request_user_dialog wave,
// byte-pinned against the installed CLI 2.1.247 binary's registrations
// (extracted from the binary's zod literals by the scout):
//
//	F1 permission gates (12 kinds incl. permission_ask_user_question):
//	  {kind:"permission_bash",payload:…("requestId"in e)&&("toolName"in e)&&
//	    ("permissionResult"in e)&&("command"in e)&&("classifierState"in e)…,
//	    result:…("behavior"in e)…, default:{behavior:"cancelled"}}
//	F2 enum consent kinds (12): result is a BARE enum string —
//	  {kind:"resume_return",payload:…{sessionAgeMinutes,estimatedTokens}…,
//	    result:…Ji(["compact","continue","dismiss","never","cancelled"]),
//	    default:"cancelled"}
//	F3 structured kinds (4): e.g.
//	  {kind:"goal_proposal",payload:…{condition:I()}…,
//	    result:…{approved:Qe(),explicit:Qe().optional()}…,
//	    default:{approved:!1}}
//
// and the declare-gate (the binary's schema .describe, verbatim):
//
//	"A kind is only sent in sessions where some attached client declared
//	 it in initialize.supportedDialogKinds (declare exactly the kinds you
//	 can render); … A host that receives a kind it did not declare must
//	 not answer it (an error-subtype response is discarded and the dialog
//	 stays pending) — never with {behavior: "cancelled"}, which is a real
//	 settlement treated as the user dismissing the dialog. An unanswered
//	 dialog is cancelled by the CLI after its dialog deadline."
package backend

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// claudeDialogKindsWant is the EXACT declaration order the initialize
// writer pins (claudeRenderedDialogKinds) — 28 kinds: the AskUserQuestion
// dialog, 11 F1 permission gates, 12 F2 enum consent kinds, 4 F3
// structured kinds. The three fail-closed kinds (computer_use_approval,
// local_jsx, mcp_url_elicitation) must NEVER appear.
var claudeDialogKindsWant = []string{
	"permission_ask_user_question",
	"permission_prompt", "permission_bash", "permission_browser",
	"permission_enter_plan_mode", "permission_exit_plan_mode_v2",
	"permission_file", "permission_monitor", "permission_powershell",
	"permission_skill", "permission_webfetch", "permission_workflow",
	"cloud_sync_consent", "fable_overage_consent_prompt",
	"refusal_fallback_prompt", "chrome_install_upsell", "chrome_install_setup",
	"auto_mode_setup_review", "resume_return", "managed_settings_security",
	"auto_default_nudge", "cost_threshold", "ide_onboarding", "it2_setup",
	"goal_proposal", "auto_mode_flagged_allow", "sandbox_network_access",
	"peer_inbound_approval",
}

// TestClaudeDialogKindTable pins the declaration table: exactly the 28
// rendered kinds in order, the park-set mirrors them, and the three
// fail-closed kinds stay undeclared.
func TestClaudeDialogKindTable(t *testing.T) {
	if len(claudeRenderedDialogKinds) != len(claudeDialogKindsWant) {
		t.Fatalf("kind count drifted: got %d, want %d", len(claudeRenderedDialogKinds), len(claudeDialogKindsWant))
	}
	for i, want := range claudeDialogKindsWant {
		if claudeRenderedDialogKinds[i] != want {
			t.Fatalf("kind[%d] drifted: got %q, want %q", i, claudeRenderedDialogKinds[i], want)
		}
		if !claudeRenderedDialogKindSet[want] {
			t.Fatalf("park set is missing declared kind %q", want)
		}
	}
	for _, closed := range []string{"computer_use_approval", "local_jsx", "mcp_url_elicitation"} {
		if claudeRenderedDialogKindSet[closed] {
			t.Fatalf("fail-closed kind %q must NEVER be declared", closed)
		}
	}
}

// TestClaudeInitializeLineBytes pins the initialize control_request byte
// shape — the SDK's own envelope ({request_id, type:"control_request",
// request:{subtype:"initialize",supportedDialogKinds:[...]}}), with the
// office's field order (type, request_id, request — the
// claudeInterruptLine layout).
func TestClaudeInitializeLineBytes(t *testing.T) {
	got := claudeInitializeLine(1, claudeRenderedDialogKinds)
	var want strings.Builder
	want.WriteString(`{"type":"control_request","request_id":"office-init-1","request":{"subtype":"initialize","supportedDialogKinds":[`)
	for i, k := range claudeDialogKindsWant {
		if i > 0 {
			want.WriteByte(',')
		}
		want.WriteString(`"` + k + `"`)
	}
	want.WriteString(`]}}`)
	if string(got) != want.String() {
		t.Fatalf("initialize bytes drifted:\n got: %s\nwant: %s", got, want.String())
	}
}

// TestClaudeInitializeWrittenAtStart drives a stub and asserts the FIRST
// stdin line the process sees is the initialize declaration (and that a
// Send lands AFTER it), reading the RAW capture (claudeCapture filters
// the declaration out for the pre-existing count assertions).
func TestClaudeInitializeWrittenAtStart(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if err := b.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "initialize + user lines written", 3*time.Second, func() bool {
		return len(claudeCaptureRaw(t, capture)) == 2
	})
	lines := claudeCaptureRaw(t, capture)
	if !strings.Contains(lines[0], `"subtype":"initialize"`) {
		t.Fatalf("the FIRST stdin line must be the initialize declaration, got %q", lines[0])
	}
	if !strings.Contains(lines[0], `"request_id":"office-init-1"`) {
		t.Fatalf("initialize request_id drifted: %q", lines[0])
	}
	for _, k := range claudeDialogKindsWant {
		if !strings.Contains(lines[0], `"`+k+`"`) {
			t.Fatalf("initialize declaration is missing kind %q: %s", k, lines[0])
		}
	}
	for _, closed := range []string{"computer_use_approval", "local_jsx", "mcp_url_elicitation"} {
		if strings.Contains(lines[0], closed) {
			t.Fatalf("fail-closed kind %q must never ride the declaration: %s", closed, lines[0])
		}
	}
	if !strings.Contains(lines[1], `"type":"user"`) {
		t.Fatalf("the user line must follow the declaration, got %q", lines[1])
	}
}

// TestClaudeInitializeRewrittenOnRespawn: a died process's next Send
// respawns with --resume — and the NEW process gets its OWN initialize
// declaration (office-init-2) before the user line.
func TestClaudeInitializeRewrittenOnRespawn(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if err := b.Send("first"); err != nil {
		t.Fatalf("Send first: %v", err)
	}
	claudeWait(t, "first process captured", 3*time.Second, func() bool {
		return len(claudeCaptureRaw(t, capture)) == 2
	})
	// kill the child: the next Send respawns with --resume and re-declares.
	b.mu.Lock()
	proc := b.proc
	b.mu.Unlock()
	if proc == nil {
		t.Fatal("no live process to kill")
	}
	_ = proc.Process.Kill()
	claudeWait(t, "the death-watch latch (died=true)", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})
	if err := b.Send("second"); err != nil {
		t.Fatalf("Send second: %v", err)
	}
	claudeWait(t, "respawn captured its own initialize + user line", 3*time.Second, func() bool {
		return len(claudeCaptureRaw(t, capture)) == 4
	})
	lines := claudeCaptureRaw(t, capture)
	if !strings.Contains(lines[2], `"subtype":"initialize"`) || !strings.Contains(lines[2], `"request_id":"office-init-2"`) {
		t.Fatalf("the respawn must re-declare supportedDialogKinds (office-init-2), got %q", lines[2])
	}
	if !strings.Contains(lines[3], `"type":"user"`) {
		t.Fatalf("the respawn's user line must follow its declaration, got %q", lines[3])
	}
}

// ---------------- F1 round-trips ----------------

// TestClaudeF1PermissionBashRoundTrip — the F1 dialog WITH a command
// payload: request JSON in -> EvQuestion page (title + subject body, the
// tool chip, three rows when showAlwaysAllow) -> the allow-once and
// allow-always legs' exact stdin bytes (updatedInput re-emitting the
// payload's tool input, the CLI's own "yes" builder), the reject leg's
// deny+message, and the dismiss leg's bare cancelled.
func TestClaudeF1PermissionBashRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-b1","request":{"subtype":"request_user_dialog","dialog_kind":"permission_bash","payload":{"requestId":"toolu_B1","toolName":"Bash","input":{"command":"rm -rf /tmp/x"},"description":"Delete temp files","permissionResult":{"behavior":"ask"},"command":"rm -rf /tmp/x","classifierState":"destructive","showAlwaysAllow":true},"tool_use_id":"toolu_B1"},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d: %+v", len(evs), evs)
	}
	e := evs[0]
	if e.Kind != state.EvQuestion || e.QuestionID != "dlg-b1" || e.ToolState != "pending" || e.SessionID != "sess-1" {
		t.Fatalf("EvQuestion identity drifted: %+v", e)
	}
	if len(e.Questions) != 1 {
		t.Fatalf("want 1 page, got %+v", e.Questions)
	}
	q := e.Questions[0]
	if q.Question != "Claude needs your permission\n\nrm -rf /tmp/x\nclassifierState: destructive" {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if q.Header != "Bash" {
		t.Fatalf("page header drifted: %q", q.Header)
	}
	if len(q.Options) != 3 || q.Options[0].Label != "Allow once" || q.Options[1].Label != "Allow always" || q.Options[2].Label != "Reject" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	meta, ok := ctx.dialogMeta["dlg-b1"]
	if !ok || meta.family != dialogFamilyPermission || meta.kind != "permission_bash" {
		t.Fatalf("dialog meta drifted: %+v (ok=%v)", meta, ok)
	}
	if string(meta.input) != `{"command":"rm -rf /tmp/x"}` {
		t.Fatalf("stashed input drifted: %s", meta.input)
	}

	// allow once -> {behavior:"allow", updatedInput:<the payload input>}
	raw, err := claudeDialogResultJSON(meta, e.Questions, [][]string{{"Allow once"}})
	if err != nil {
		t.Fatalf("allow once: %v", err)
	}
	got, err := claudeControlResponseFor("dlg-b1", claudeControlResult{Behavior: "completed", Result: raw})
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-b1","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"command":"rm -rf /tmp/x"}}}}}`
	if string(got) != want {
		t.Fatalf("allow-once bytes drifted:\n got: %s\nwant: %s", got, want)
	}

	// allow always -> the SAME plain allow (no permissionUpdates are
	// derivable from an F1 payload — documented on the renderer).
	raw, err = claudeDialogResultJSON(meta, e.Questions, [][]string{{"Allow always"}})
	if err != nil {
		t.Fatalf("allow always: %v", err)
	}
	got, _ = claudeControlResponseFor("dlg-b1", claudeControlResult{Behavior: "completed", Result: raw})
	if string(got) != want {
		t.Fatalf("allow-always bytes drifted:\n got: %s\nwant: %s", got, want)
	}

	// reject -> {behavior:"deny", message:"Denied by the boss in theboringoffice"}
	raw, err = claudeDialogResultJSON(meta, e.Questions, [][]string{{"Reject"}})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	got, _ = claudeControlResponseFor("dlg-b1", claudeControlResult{Behavior: "completed", Result: raw})
	want = `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-b1","response":{"behavior":"completed","result":{"behavior":"deny","message":"Denied by the boss in theboringoffice"}}}}`
	if string(got) != want {
		t.Fatalf("reject bytes drifted:\n got: %s\nwant: %s", got, want)
	}

	// a label the map does not know fails CLOSED (no write).
	if _, err := claudeDialogResultJSON(meta, e.Questions, [][]string{{"rm -rf /"}}); err == nil {
		t.Fatal("a free-text answer on a permission dialog must error (fail closed)")
	}
}

// TestClaudeF1PermissionWebfetchRoundTrip — the MINIMAL F1 payload (no
// input, no showAlwaysAllow): the always-row drops (the CLI's own
// showAlwaysAllow gate) and the allow leg is the bare {"behavior":"allow"}.
func TestClaudeF1PermissionWebfetchRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-w1","request":{"subtype":"request_user_dialog","dialog_kind":"permission_webfetch","payload":{"requestId":"toolu_W1","toolName":"WebFetch","permissionResult":{"behavior":"ask"},"hostname":"example.com"}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if q.Question != "Claude needs your permission\n\nexample.com" {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if len(q.Options) != 2 || q.Options[0].Label != "Allow once" || q.Options[1].Label != "Reject" {
		t.Fatalf("a payload without showAlwaysAllow must drop the always row: %+v", q.Options)
	}
	meta := ctx.dialogMeta["dlg-w1"]
	raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Allow once"}})
	if err != nil {
		t.Fatalf("allow once: %v", err)
	}
	if string(raw) != `{"behavior":"allow"}` {
		t.Fatalf("an input-less payload yields the BARE allow: %s", raw)
	}
	got, _ := claudeControlResponseFor("dlg-w1", claudeControlResult{Behavior: "completed", Result: raw})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-w1","response":{"behavior":"completed","result":{"behavior":"allow"}}}}`
	if string(got) != want {
		t.Fatalf("bytes drifted:\n got: %s\nwant: %s", got, want)
	}
}

// TestClaudeF1DialogStubRoundTrip drives a stub through the FULL backend
// for the dismiss leg: request in -> AnswerQuestion allow-once on one
// dialog, RejectQuestion on the second — exact stdin bytes both ways.
func TestClaudeF1DialogStubRoundTrip(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"dlg-f1","request":{"subtype":"request_user_dialog","dialog_kind":"permission_file","payload":{"requestId":"toolu_F1","toolName":"Write","input":{"file_path":"/tmp/note.txt","content":"hi"},"permissionResult":{"behavior":"ask"},"filePath":"/tmp/note.txt","operationType":"write","showAlwaysAllow":true},"tool_use_id":"toolu_F1"},"session_id":"sess-sh-1"}'
      printf '%s\n' '{"type":"control_request","request_id":"dlg-f2","request":{"subtype":"request_user_dialog","dialog_kind":"permission_skill","payload":{"requestId":"toolu_F2","toolName":"Skill","permissionResult":{"behavior":"ask"},"skill":"review-pr"},"tool_use_id":"toolu_F2"},"session_id":"sess-sh-1"}'
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
	if err := b.Send("do things"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "both F1 dialogs mapped", 3*time.Second, func() bool {
		var d1, d2 bool
		for _, e := range log.snapshot() {
			if e.Kind == state.EvQuestion && e.ToolState == "pending" {
				switch e.QuestionID {
				case "dlg-f1":
					d1 = true
				case "dlg-f2":
					d2 = true
				}
			}
		}
		return d1 && d2
	})
	// the decode legs through the live backend: subjects + rows.
	for _, e := range log.snapshot() {
		if e.Kind != state.EvQuestion || e.ToolState != "pending" {
			continue
		}
		switch e.QuestionID {
		case "dlg-f1":
			if e.Questions[0].Question != "Claude needs your permission\n\nwrite /tmp/note.txt" || e.Questions[0].Header != "Write" {
				t.Fatalf("file dialog page drifted: %+v", e.Questions[0])
			}
		case "dlg-f2":
			if e.Questions[0].Question != "Claude needs your permission\n\nreview-pr" {
				t.Fatalf("skill dialog page drifted: %+v", e.Questions[0])
			}
			if len(e.Questions[0].Options) != 2 { // no showAlwaysAllow -> no always row
				t.Fatalf("skill dialog rows drifted: %+v", e.Questions[0].Options)
			}
		}
	}
	if err := b.AnswerQuestion("dlg-f1", [][]string{{"Allow always"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	if err := b.RejectQuestion("dlg-f2"); err != nil {
		t.Fatalf("RejectQuestion: %v", err)
	}
	claudeWait(t, "both control_response lines written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 3 // 1 user + 2 responses
	})
	wants := []string{
		`{"type":"control_response","response":{"subtype":"success","request_id":"dlg-f1","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"file_path":"/tmp/note.txt","content":"hi"}}}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"dlg-f2","response":{"behavior":"cancelled"}}}`,
	}
	for _, want := range wants {
		found := false
		for _, got := range claudeCapture(t, capture) {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("control_response missing %q in capture:\n%s", want, strings.Join(claudeCapture(t, capture), "\n"))
		}
	}
}

// ---------------- F2 round-trips ----------------

// TestClaudeF2CloudSyncConsentRoundTrip — the F2 kind WITH title/body
// payload: the CLI's own copy renders; the enum answer is a BARE JSON
// STRING ("sync") — never an object.
func TestClaudeF2CloudSyncConsentRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-c1","request":{"subtype":"request_user_dialog","dialog_kind":"cloud_sync_consent","payload":{"folder":"~/notes","title":"Sync your notes?","body":"Claude wants to keep this folder in sync.","detail":"Keeps the folder mirrored.","fileCount":3,"totalBytes":2048}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	wantQ := "Sync your notes?\n\nClaude wants to keep this folder in sync.\n\nKeeps the folder mirrored.\n\n3 files, 2048 bytes in ~/notes"
	if q.Question != wantQ {
		t.Fatalf("page question drifted:\n got: %q\nwant: %q", q.Question, wantQ)
	}
	if len(q.Options) != 3 || q.Options[0].Label != "Sync this folder" || q.Options[1].Label != "Use device tools" || q.Options[2].Label != "Not now" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	meta := ctx.dialogMeta["dlg-c1"]
	if meta.family != dialogFamilyLabelResult {
		t.Fatalf("family drifted: %v", meta.family)
	}
	raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Sync this folder"}})
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if string(raw) != `"sync"` {
		t.Fatalf("the F2 result must be the BARE enum string: %s", raw)
	}
	got, _ := claudeControlResponseFor("dlg-c1", claudeControlResult{Behavior: "completed", Result: raw})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-c1","response":{"behavior":"completed","result":"sync"}}}`
	if string(got) != want {
		t.Fatalf("bytes drifted:\n got: %s\nwant: %s", got, want)
	}
	// the "not_now" leg — the enum's dismissal VALUE rides completed (the
	// office's own dismiss path stays envelope-cancelled).
	raw, _ = claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Not now"}})
	if string(raw) != `"not_now"` {
		t.Fatalf("not_now leg drifted: %s", raw)
	}
}

// TestClaudeF2CostThresholdRoundTrip — the EMPTY-payload F2 kind: the
// CLI's own title/body copy renders, one option, the answer is the bare
// enum string "acknowledged" (the CLI's wrapper maps its "ok" value to
// exactly that).
func TestClaudeF2CostThresholdRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-c2","request":{"subtype":"request_user_dialog","dialog_kind":"cost_threshold","payload":{}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if q.Question != "You've spent $5 on the Anthropic API this session.\n\nLearn more about how to monitor your spending: https://code.claude.com/docs/en/costs" {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if len(q.Options) != 1 || q.Options[0].Label != "Got it, thanks!" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	raw, err := claudeDialogResultJSON(ctx.dialogMeta["dlg-c2"], evs[0].Questions, [][]string{{"Got it, thanks!"}})
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if string(raw) != `"acknowledged"` {
		t.Fatalf("the acknowledged leg must be the bare enum string: %s", raw)
	}
}

// TestClaudeF2ResumeReturnStubRoundTrip — the FULL backend leg for an F2
// kind: request in, "Resume from summary (recommended)" picked, the bare
// enum string "compact" lands on stdin.
func TestClaudeF2ResumeReturnStubRoundTrip(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"dlg-r1","request":{"subtype":"request_user_dialog","dialog_kind":"resume_return","payload":{"sessionAgeMinutes":47,"estimatedTokens":183000}},"session_id":"sess-sh-1"}'
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
	if err := b.Send("resume me"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "dialog mapped", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvQuestion && e.QuestionID == "dlg-r1" && e.ToolState == "pending" {
				return true
			}
		}
		return false
	})
	var pending *state.Event
	for i, e := range log.snapshot() {
		if e.Kind == state.EvQuestion && e.QuestionID == "dlg-r1" && e.ToolState == "pending" {
			pending = &log.snapshot()[i]
		}
	}
	if pending == nil || len(pending.Questions) != 1 {
		t.Fatalf("pending dialog drifted: %+v", pending)
	}
	q := pending.Questions[0]
	if q.Question != "This session is 47 minutes old and 183000 tokens.\n\nResuming the full session will consume a substantial portion of your usage limits. We recommend resuming from a summary." {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if len(q.Options) != 3 || q.Options[0].Label != "Resume from summary (recommended)" || q.Options[1].Label != "Resume full session as-is" || q.Options[2].Label != "Don't ask me again" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	if err := b.AnswerQuestion("dlg-r1", [][]string{{"Resume from summary (recommended)"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	claudeWait(t, "control_response written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2 // 1 user + 1 response
	})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-r1","response":{"behavior":"completed","result":"compact"}}}`
	found := false
	for _, got := range claudeCapture(t, capture) {
		if got == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("control_response missing %q in capture:\n%s", want, strings.Join(claudeCapture(t, capture), "\n"))
	}
}

// TestClaudeF2ChromeInstallSetupStreaming — the STREAMING kind: payload
// updates re-arrive on the SAME request_id as the phase transitions.
// Every update re-emits the EvQuestion; the stash/meta OVERWRITE, so an
// answer maps against the LATEST phase (a connected-phase "Continue with
// browser tools" settles "continue" even though the first phase only
// offered the skip row).
func TestClaudeF2ChromeInstallSetupStreaming(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs1 := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-cs","request":{"subtype":"request_user_dialog","dialog_kind":"chrome_install_setup","payload":{"phase":"waiting_install","installPageOpened":true}},"session_id":"sess-1"}`)
	if len(evs1) != 1 {
		t.Fatalf("phase 1: want 1 event, got %+v", evs1)
	}
	q1 := evs1[0].Questions[0]
	if len(q1.Options) != 1 || q1.Options[0].Label != "Continue without browser tools" {
		t.Fatalf("waiting_install offers only the skip row: %+v", q1.Options)
	}
	if !strings.HasPrefix(q1.Question, "Setting up Claude in Chrome\n\nphase: waiting_install") {
		t.Fatalf("phase 1 question drifted: %q", q1.Question)
	}
	evs2 := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-cs","request":{"subtype":"request_user_dialog","dialog_kind":"chrome_install_setup","payload":{"phase":"connected","installPageOpened":true}},"session_id":"sess-1"}`)
	if len(evs2) != 1 {
		t.Fatalf("phase 2: want 1 event (re-emitted), got %+v", evs2)
	}
	q2 := evs2[0].Questions[0]
	if len(q2.Options) != 2 || q2.Options[0].Label != "Continue with browser tools" || q2.Options[1].Label != "Continue without browser tools" {
		t.Fatalf("connected offers continue + skip: %+v", q2.Options)
	}
	// the stash/meta overwrite: answering against the LATEST phase works.
	raw, err := claudeDialogResultJSON(ctx.dialogMeta["dlg-cs"], evs2[0].Questions, [][]string{{"Continue with browser tools"}})
	if err != nil {
		t.Fatalf("the latest phase's option must map: %v", err)
	}
	if string(raw) != `"continue"` {
		t.Fatalf("continue leg drifted: %s", raw)
	}
	// and a STALE phase's option (the first frame's skip is still valid —
	// "skip" is in-vocab at every phase; "keep_waiting" from a stalled
	// frame would fail closed once the phase moved on).
	if _, err := claudeDialogResultJSON(ctx.dialogMeta["dlg-cs"], evs2[0].Questions, [][]string{{"Keep waiting"}}); err == nil {
		t.Fatal("a stale-phase label must fail closed after the phase moved on")
	}
}

// ---------------- F3 round-trips ----------------

// TestClaudeF3GoalProposalRoundTrip — both legs byte-exact: approve ->
// {"approved":true,"explicit":true}, deny -> {"approved":false,
// "explicit":true} (the CLI's own onConfirm/onCancel builders).
func TestClaudeF3GoalProposalRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-g1","request":{"subtype":"request_user_dialog","dialog_kind":"goal_proposal","payload":{"condition":"all tests pass"}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if !strings.HasPrefix(q.Question, "Claude proposed a session goal\n\nall tests pass") {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if len(q.Options) != 2 || q.Options[0].Label != "Set this goal" || q.Options[1].Label != "Not now" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	meta := ctx.dialogMeta["dlg-g1"]
	raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Set this goal"}})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if string(raw) != `{"approved":true,"explicit":true}` {
		t.Fatalf("approve leg drifted: %s", raw)
	}
	got, _ := claudeControlResponseFor("dlg-g1", claudeControlResult{Behavior: "completed", Result: raw})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-g1","response":{"behavior":"completed","result":{"approved":true,"explicit":true}}}}`
	if string(got) != want {
		t.Fatalf("approve bytes drifted:\n got: %s\nwant: %s", got, want)
	}
	raw, _ = claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Not now"}})
	if string(raw) != `{"approved":false,"explicit":true}` {
		t.Fatalf("deny leg drifted: %s", raw)
	}
}

// TestClaudeF3SandboxNetworkAccessRoundTrip — all three rows, byte-exact
// (persistRow deliberately omitted on the don't-ask-again leg: it is a
// process-local RENDERED node in the CLI, and the result schema marks it
// optional).
func TestClaudeF3SandboxNetworkAccessRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-s1","request":{"subtype":"request_user_dialog","dialog_kind":"sandbox_network_access","payload":{"host":"api.example.com","port":443}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if q.Question != "Claude wants network access\n\napi.example.com:443" {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	if len(q.Options) != 3 || q.Options[0].Label != "Yes" || q.Options[1].Label != "Yes, don't ask again" || q.Options[2].Label != "No" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	meta := ctx.dialogMeta["dlg-s1"]
	legs := map[string]string{
		"Yes":                  `{"allow":true,"persistToSettings":false}`,
		"Yes, don't ask again": `{"allow":true,"persistToSettings":true}`,
		"No":                   `{"allow":false,"persistToSettings":false}`,
	}
	for label, want := range legs {
		raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{label}})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if string(raw) != want {
			t.Fatalf("%s leg drifted:\n got: %s\nwant: %s", label, raw, want)
		}
	}
}

// TestClaudeF3PeerInboundApprovalRoundTrip — the approve/deny legs of the
// peer message gate, byte-exact.
func TestClaudeF3PeerInboundApprovalRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-p1","request":{"subtype":"request_user_dialog","dialog_kind":"peer_inbound_approval","payload":{"holdCause":"rate-limit","preview":"ship it?"}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if q.Question != "A message from another session needs your approval\n\nship it?\n\nholdCause: rate-limit" {
		t.Fatalf("page question drifted: %q", q.Question)
	}
	meta := ctx.dialogMeta["dlg-p1"]
	raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Deliver this message to Claude"}})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if string(raw) != `{"behavior":"approve"}` {
		t.Fatalf("approve leg drifted: %s", raw)
	}
	raw, _ = claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Deny — drop it and tell the sender it was declined"}})
	if string(raw) != `{"behavior":"deny"}` {
		t.Fatalf("deny leg drifted: %s", raw)
	}
	got, _ := claudeControlResponseFor("dlg-p1", claudeControlResult{Behavior: "completed", Result: raw})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-p1","response":{"behavior":"completed","result":{"behavior":"deny"}}}}`
	if string(got) != want {
		t.Fatalf("deny bytes drifted:\n got: %s\nwant: %s", got, want)
	}
}

// TestClaudeF3FlaggedAllowRoundTrip — the multi-select: picked rules
// become {toRemove:[...]}; "Remove them all" expands to every flagged
// rule; an empty pick leaves them all ({toRemove:[]}).
func TestClaudeF3FlaggedAllowRoundTrip(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-a1","request":{"subtype":"request_user_dialog","dialog_kind":"auto_mode_flagged_allow","payload":{"flagged":["Bash(rm *)","WebFetch","Bash(git push *)"],"runId":"run-9"}},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %+v", evs)
	}
	q := evs[0].Questions[0]
	if !q.Multiple {
		t.Fatalf("the flagged-allow page must be multi-select: %+v", q)
	}
	if len(q.Options) != 4 || q.Options[3].Label != "Remove them all" {
		t.Fatalf("page options drifted: %+v", q.Options)
	}
	meta := ctx.dialogMeta["dlg-a1"]
	raw, err := claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Bash(rm *)", "WebFetch"}})
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	if string(raw) != `{"toRemove":["Bash(rm *)","WebFetch"]}` {
		t.Fatalf("pick leg drifted: %s", raw)
	}
	raw, _ = claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{"Remove them all"}})
	if string(raw) != `{"toRemove":["Bash(rm *)","WebFetch","Bash(git push *)"]}` {
		t.Fatalf("remove-all leg drifted: %s", raw)
	}
	raw, _ = claudeDialogResultJSON(meta, evs[0].Questions, [][]string{{}})
	if string(raw) != `{"toRemove":[]}` {
		t.Fatalf("empty pick = leave them all: %s", raw)
	}
	got, _ := claudeControlResponseFor("dlg-a1", claudeControlResult{Behavior: "completed", Result: raw})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-a1","response":{"behavior":"completed","result":{"toRemove":[]}}}}`
	if string(got) != want {
		t.Fatalf("bytes drifted:\n got: %s\nwant: %s", got, want)
	}
}

// TestClaudeF2KindsRenderCoverage — every declared kind renders ONE page
// through the dispatcher with a REAL-shaped payload (the per-kind zod
// required fields), and every rendered option label maps to a result.
// This is the table's drift guard: a kind added to the declaration
// without a renderer fails here.
func TestClaudeF2KindsRenderCoverage(t *testing.T) {
	payloads := map[string]string{
		"permission_ask_user_question": `{"requestId":"t1","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Ship?","header":"Ship","options":[{"label":"yes","description":"y"},{"label":"no","description":"n"}],"multiSelect":false}]}`,
		"permission_prompt":            `{"requestId":"t2","toolName":"CronCreate","input":{"prompt":"nightly"},"permissionResult":{"behavior":"ask"},"description":"Create a scheduled job","showAlwaysAllow":true}`,
		"permission_bash":              `{"requestId":"t3","toolName":"Bash","input":{"command":"ls"},"permissionResult":{"behavior":"ask"},"command":"ls","classifierState":"read","showAlwaysAllow":true}`,
		"permission_browser":           `{"requestId":"t4","toolName":"WebFetch","input":{"url":"https://x.dev"},"permissionResult":{"behavior":"ask"},"verbPhrase":"browse this page","chrome":{"host":"x.dev","url":"https://x.dev"}}`,
		"permission_enter_plan_mode":   `{"requestId":"t5","toolName":"EnterPlanMode","input":{},"permissionResult":{"behavior":"ask"}}`,
		"permission_exit_plan_mode_v2": `{"requestId":"t6","toolName":"ExitPlanMode","input":{},"permissionResult":{"behavior":"ask"},"plan":"# The plan\n1. do it"}`,
		"permission_file":              `{"requestId":"t7","toolName":"Edit","input":{"file_path":"/a.go"},"permissionResult":{"behavior":"ask"},"filePath":"/a.go","operationType":"edit"}`,
		"permission_monitor":           `{"requestId":"t8","toolName":"Monitor","input":{},"permissionResult":{"behavior":"ask"},"intervalMs":30000}`,
		"permission_powershell":        `{"requestId":"t9","toolName":"PowerShell","input":{"command":"Get-ChildItem"},"permissionResult":{"behavior":"ask"},"command":"Get-ChildItem"}`,
		"permission_skill":             `{"requestId":"t10","toolName":"Skill","permissionResult":{"behavior":"ask"},"skill":"review-pr"}`,
		"permission_webfetch":          `{"requestId":"t11","toolName":"WebFetch","permissionResult":{"behavior":"ask"},"hostname":"x.dev"}`,
		"permission_workflow":          `{"requestId":"t12","toolName":"Workflow","input":{"script":"run"},"permissionResult":{"behavior":"ask"},"script":"run the thing"}`,
		"cloud_sync_consent":           `{"folder":"~/n","title":"Sync?","body":"b","fileCount":1,"totalBytes":2}`,
		"fable_overage_consent_prompt": `{"overagesEnabled":true,"balanceCents":500,"currency":"USD"}`,
		"refusal_fallback_prompt":      `{"originalModel":"claude-opus-4-8","fallbackModel":"claude-sonnet-4-6","guidanceText":"try again"}`,
		"chrome_install_upsell":        `{}`,
		"chrome_install_setup":         `{"phase":"stalled","installPageOpened":true}`,
		"auto_mode_setup_review":       `{"environment":["darwin"],"allow":["Bash(ls *)"],"soft_deny":[],"hard_deny":[],"remove_from_permissions_allow":[],"notes":["looks fine"],"mode":"append"}`,
		"resume_return":                `{"sessionAgeMinutes":5,"estimatedTokens":1000}`,
		"managed_settings_security":    `{"settings":{"permissions":{"defaultMode":"plan"}}}`,
		"auto_default_nudge":           `{"currentMode":"default"}`,
		"cost_threshold":               `{}`,
		"ide_onboarding":               `{"installationStatus":{"ideType":"vscode"}}`,
		"it2_setup":                    `{"tmuxAvailable":true}`,
		"goal_proposal":                `{"condition":"tests pass"}`,
		"auto_mode_flagged_allow":      `{"flagged":["Bash(rm *)"],"runId":"r1"}`,
		"sandbox_network_access":       `{"host":"x.dev","port":443}`,
		"peer_inbound_approval":        `{"holdCause":"rate-limit","preview":"hi"}`,
	}
	for _, kind := range claudeDialogKindsWant {
		payload, ok := payloads[kind]
		if !ok {
			t.Fatalf("no fixture payload for declared kind %q", kind)
		}
		render, ok := claudeRenderDialog(kind, json.RawMessage(payload))
		if !ok {
			t.Fatalf("declared kind %q did not render", kind)
		}
		if len(render.items) != 1 {
			t.Fatalf("kind %q: want ONE page, got %d", kind, len(render.items))
		}
		page := render.items[0]
		if strings.TrimSpace(page.Question) == "" {
			t.Fatalf("kind %q: empty question text", kind)
		}
		if len(page.Options) == 0 {
			t.Fatalf("kind %q: every rendered page needs options (a free-text row could never map to a result)", kind)
		}
		// every rendered label maps to a result (except the AUQ kind,
		// whose answers are free-form by design, and the flagged-allow
		// multi page whose result is pick-built).
		switch render.meta.family {
		case dialogFamilyLabelResult:
			for _, opt := range page.Options {
				raw, ok := render.meta.resultByLabel[opt.Label]
				if !ok || len(raw) == 0 {
					t.Fatalf("kind %q: option %q has no result bytes", kind, opt.Label)
				}
			}
		case dialogFamilyFlaggedAllow:
			if len(render.meta.flagged) == 0 {
				t.Fatalf("kind %q: no flagged rules stashed", kind)
			}
		case dialogFamilyPermission, dialogFamilyAUQ:
			// covered by their own round-trip tests
		default:
			t.Fatalf("kind %q: unknown family %d", kind, render.meta.family)
		}
	}
}

// TestClaudeAnswerQuestionUnknownLabelFailsClosed — through the LIVE
// backend: a picked label outside the rendered set (the modal's free-text
// row) returns an error and writes NOTHING to stdin; the dialog stays
// parked (the CLI's deadline settles it as the kind's default).
func TestClaudeAnswerQuestionUnknownLabelFailsClosed(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"dlg-x1","request":{"subtype":"request_user_dialog","dialog_kind":"cost_threshold","payload":{}},"session_id":"sess-sh-1"}'
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
	if err := b.Send("tick"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "dialog mapped", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvQuestion && e.QuestionID == "dlg-x1" && e.ToolState == "pending" {
				return true
			}
		}
		return false
	})
	if err := b.AnswerQuestion("dlg-x1", [][]string{{"free-typed nonsense"}}); err == nil {
		t.Fatal("an unrendered label must error (fail closed)")
	}
	// nothing written beyond the user line (the filtered capture hides
	// the initialize declaration): the dialog is still parked.
	claudeWait(t, "the user line landed", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	for _, got := range claudeCapture(t, capture) {
		if strings.Contains(got, "dlg-x1") {
			t.Fatalf("no control_response may be written for a failed answer: %q", got)
		}
	}
	// and the dialog still answers correctly afterwards (the error
	// settled nothing).
	if err := b.AnswerQuestion("dlg-x1", [][]string{{"Got it, thanks!"}}); err != nil {
		t.Fatalf("the parked dialog must still answer: %v", err)
	}
	claudeWait(t, "the real answer landed", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-x1","response":{"behavior":"completed","result":"acknowledged"}}}`
	found := false
	for _, got := range claudeCapture(t, capture) {
		if got == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("control_response missing %q in capture:\n%s", want, strings.Join(claudeCapture(t, capture), "\n"))
	}
}
