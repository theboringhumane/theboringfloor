// claude_dialog_test.go — the request_user_dialog control_response shape,
// byte-pinned against the installed CLI 2.1.247 binary's SDK schema
// (extracted from the binary's zod literals):
//
//	response := {behavior: enum("completed","cancelled"),
//	             result?: unknown  — opaque per dialog_kind}
//
// corroborated by the CLI's own writers/consumers in the same binary:
//   - cancelDialogByMachine injects exactly
//     {"type":"control_response","response":{"subtype":"success",
//     "request_id":…,"response":{"behavior":"cancelled"}}};
//   - the late-answer telemetry buckets response.behavior as
//     "completed"|"cancelled"|"absent"|"other" ("allow"/"deny" → "other");
//   - every dialog consumer reads g.behavior + g.result — there is NO
//     "answer" key anywhere in the dialog protocol;
//   - for dialog_kind permission_ask_user_question the result is an
//     OBJECT carrying a "behavior" key (the kind registration's result
//     validator: typeof e==="object" && e!==null && ("behavior" in e)) —
//     concretely the permission decision the CLI's own AskUserQuestion
//     submit builder produces:
//     {behavior:"allow",updatedInput:{…input,answers:{question:text}}}.
//
// The REQUEST side rides dialog_kind + payload (the CLI's emitter:
// request:{subtype:"request_user_dialog",dialog_kind:r,payload:s}) — the
// office renders ONLY dialog_kind "permission_ask_user_question" (the
// AskUserQuestion tool's dialog; payload.questions =
// [{question,header,options:[{label,description,preview?}],multiSelect}]
// plus payload-level metadataSource?/metadata? — all re-emitted in the
// answer's updatedInput) and parks every other kind WITHOUT settling it.
package backend

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// mapClaudeLineForTest runs one raw stdout line through the SAME entry
// the live read loop uses (json.Unmarshal -> mapClaudeEvent).
func mapClaudeLineForTest(t *testing.T, ctx *claudeNormCtx, line string) []state.Event {
	t.Helper()
	var raw claudeEvent
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return mapClaudeEvent(raw, ctx, 1000)
}

// TestClaudeDialogRequestMapping: a REAL-shaped request of the known kind
// decodes into the structured EvQuestion pages (question/header/options/
// multiple), the flattened Text ("a b") + ToolSummary ("x | y | z"), and
// the pending hold the AnswerQuestion writer resolves.
func TestClaudeDialogRequestMapping(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	evs := mapClaudeLineForTest(t, ctx,
		`{"type":"control_request","request_id":"dlg-100","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01AUQ9","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep"}],"multiSelect":false}]},"tool_use_id":"toolu_01AUQ9"},"session_id":"sess-1"}`)
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d: %+v", len(evs), evs)
	}
	e := evs[0]
	if e.Kind != state.EvQuestion || e.QuestionID != "dlg-100" || e.ToolState != "pending" ||
		e.SessionID != "sess-1" || e.EmployeeName != "boss" {
		t.Fatalf("EvQuestion identity drifted: %+v", e)
	}
	if e.Text != "Which finish for the east wall? Which spray pattern?" {
		t.Fatalf("flattened text drifted: %q", e.Text)
	}
	if e.ToolSummary != "matte | gloss | both | cross-hatch | single-pass" {
		t.Fatalf("summary drifted: %q", e.ToolSummary)
	}
	if len(e.Questions) != 2 {
		t.Fatalf("want 2 pages, got %+v", e.Questions)
	}
	q1 := e.Questions[0]
	if q1.Question != "Which finish for the east wall?" || q1.Header != "Finish" || !q1.Multiple || len(q1.Options) != 3 {
		t.Fatalf("page 1 drifted: %+v", q1)
	}
	if q1.Options[0].Label != "matte" || q1.Options[0].Description != "Flat, no sheen" {
		t.Fatalf("page 1 option drifted: %+v", q1.Options[0])
	}
	q2 := e.Questions[1]
	if q2.Question != "Which spray pattern?" || q2.Header != "Pattern" || q2.Multiple || len(q2.Options) != 2 {
		t.Fatalf("page 2 drifted: %+v", q2)
	}
	if _, ok := ctx.pendingQuestions["dlg-100"]; !ok {
		t.Fatal("the pending hold for AnswerQuestion was not stashed")
	}
}

// TestClaudeDialogUnknownKindParked: the office must NEVER surface (and
// therefore never settle) a dialog it cannot render — unknown kinds, the
// three deliberatedly-UNDECLARED kinds (computer_use_approval's opaque
// payload, local_jsx's process-local nodeId + null result,
// mcp_url_elicitation's free-text URL input), undecodable payloads, and
// known kinds with no renderable page all park silently: NO EvQuestion,
// NO pending hold, NO dialog meta.
func TestClaudeDialogUnknownKindParked(t *testing.T) {
	lines := map[string]string{
		"unknown kind":      `{"type":"control_request","request_id":"dlg-200","request":{"subtype":"request_user_dialog","dialog_kind":"some_future_kind","payload":{"title":"Sync?","body":"…"},"tool_use_id":"toolu_01X"},"session_id":"sess-1"}`,
		"garbage payload":   `{"type":"control_request","request_id":"dlg-201","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":"not-an-object"},"session_id":"sess-1"}`,
		"no questions":      `{"type":"control_request","request_id":"dlg-202","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01Z","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[]}},"session_id":"sess-1"}`,
		"blank questions":   `{"type":"control_request","request_id":"dlg-203","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"questions":[{"question":"  ","options":[{"label":"a","description":"b"}]}]}},"session_id":"sess-1"}`,
		"legacy flat shape": `{"type":"control_request","request_id":"dlg-204","request":{"subtype":"request_user_dialog","question":"Which finish?","options":["matte","gloss"]},"session_id":"sess-1"}`,
		// pins the KIND GUARD itself (not the empty-questions park): a
		// kind the office did not declare parks even when its payload
		// happens to carry a decodable questions array.
		"unknown kind with decodable questions": `{"type":"control_request","request_id":"dlg-205","request":{"subtype":"request_user_dialog","dialog_kind":"some_future_kind","payload":{"questions":[{"question":"Sneaky?","options":[{"label":"a","description":"b"}]}]}},"session_id":"sess-1"}`,
		// the three fail-closed kinds of claudeRenderedDialogKinds' table:
		// known to the CLI, deliberately NEVER declared by the office.
		"computer_use_approval parked": `{"type":"control_request","request_id":"dlg-206","request":{"subtype":"request_user_dialog","dialog_kind":"computer_use_approval","payload":{"apps":["Safari"]}},"session_id":"sess-1"}`,
		"local_jsx parked":             `{"type":"control_request","request_id":"dlg-207","request":{"subtype":"request_user_dialog","dialog_kind":"local_jsx","payload":{"nodeId":"n-1","commandName":"x","immediate":true,"hidesPrompt":false}},"session_id":"sess-1"}`,
		"mcp_url_elicitation parked":   `{"type":"control_request","request_id":"dlg-208","request":{"subtype":"request_user_dialog","dialog_kind":"mcp_url_elicitation","payload":{"serverName":"memo","params":{"message":"open","mode":"url"}}},"session_id":"sess-1"}`,
		// a declared kind whose payload is undecodable for ITS shape
		// (an array where the object belongs) parks the same way.
		"declared kind, garbage payload": `{"type":"control_request","request_id":"dlg-209","request":{"subtype":"request_user_dialog","dialog_kind":"permission_bash","payload":[1,2,3]},"session_id":"sess-1"}`,
		// auto_mode_flagged_allow with NO flagged rules has no page to
		// render — parks (an empty popover would be a settle path for
		// nothing rendered).
		"flagged_allow with no rules": `{"type":"control_request","request_id":"dlg-210","request":{"subtype":"request_user_dialog","dialog_kind":"auto_mode_flagged_allow","payload":{"flagged":[],"runId":"run-1"}},"session_id":"sess-1"}`,
	}
	for name, line := range lines {
		ctx := newClaudeNormCtx(nil)
		if evs := mapClaudeLineForTest(t, ctx, line); len(evs) != 0 {
			t.Fatalf("%s: want NO events (parked), got %+v", name, evs)
		}
		if len(ctx.pendingQuestions) != 0 {
			t.Fatalf("%s: no hold may be stashed for a parked dialog", name)
		}
		if len(ctx.dialogMeta) != 0 {
			t.Fatalf("%s: no dialog meta may be stashed for a parked dialog", name)
		}
	}
}

// TestClaudeDialogResponseShape: the inner blob, byte-exact, straight out
// of the envelope writer — completed carries result as an OBJECT (the
// permission_ask_user_question result is the CLI's permission decision
// {behavior:"allow",updatedInput:{questions,answers}}), cancelled stands
// alone (NO result key, NO answer key, NO message). The object shape is
// the mutation guard: a bare-string result breaks these bytes (and would
// fail the CLI's ("behavior" in result) check, settling the dialog as
// cancelled — the bug this pins against).
func TestClaudeDialogResponseShape(t *testing.T) {
	raw, err := claudeAskUserResultJSON(
		[]state.QuestionItem{{
			Question: "Which finish?", Header: "Finish",
			Options: []state.QuestionOption{
				{Label: "matte", Description: "Flat, no sheen"},
				{Label: "gloss", Description: "High sheen"},
			},
		}},
		[][]string{{"matte"}},
	)
	if err != nil {
		t.Fatalf("claudeAskUserResultJSON: %v", err)
	}
	got, err := claudeControlResponseFor("dlg-1", claudeControlResult{
		Behavior: "completed", Result: raw,
	})
	if err != nil {
		t.Fatalf("claudeControlResponseFor(completed): %v", err)
	}
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-1","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finish?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"}],"multiSelect":false}],"answers":{"Which finish?":"matte"}}}}}}`
	if string(got) != want {
		t.Fatalf("completed bytes drifted:\n got: %s\nwant: %s", got, want)
	}

	got, err = claudeControlResponseFor("dlg-1", claudeControlResult{Behavior: "cancelled"})
	if err != nil {
		t.Fatalf("claudeControlResponseFor(cancelled): %v", err)
	}
	want = `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-1","response":{"behavior":"cancelled"}}}`
	if string(got) != want {
		t.Fatalf("cancelled bytes drifted:\n got: %s\nwant: %s", got, want)
	}

	// an empty request_id never writes anything
	if _, err := claudeControlResponseFor("", claudeControlResult{Behavior: "completed", Result: json.RawMessage(`"x"`)}); err == nil {
		t.Fatal("an empty request_id must be refused")
	}
}

// TestClaudeAskUserResultShape: the result builder in isolation — the
// CLI's own shape (binary evidence on claudeAskUserDialogResult): behavior
// "allow", updatedInput re-emitting the asked questions, and the answers
// map keyed by QUESTION TEXT with multi-select selections joined ", ".
// A string result (the old joined-text bug) can never satisfy this.
func TestClaudeAskUserResultShape(t *testing.T) {
	raw, err := claudeAskUserResultJSON(
		[]state.QuestionItem{
			{
				Question: "Which finishes for the east wall?", Header: "Finish", Multiple: true,
				Options: []state.QuestionOption{
					{Label: "matte", Description: "Flat, no sheen"},
					{Label: "gloss", Description: "High sheen"},
					{Label: "both", Description: "Matte base, gloss trim"},
				},
			},
			{
				Question: "Which spray pattern?", Header: "Pattern",
				Options: []state.QuestionOption{
					{Label: "cross-hatch", Description: "Two perpendicular passes"},
					{Label: "single-pass", Description: "One steady sweep"},
				},
			},
		},
		[][]string{{"matte", "gloss"}, {"sprayed"}},
	)
	if err != nil {
		t.Fatalf("claudeAskUserResultJSON: %v", err)
	}
	want := `{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finishes for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep"}],"multiSelect":false}],"answers":{"Which finishes for the east wall?":"matte, gloss","Which spray pattern?":"sprayed"}}}`
	if string(raw) != want {
		t.Fatalf("result bytes drifted:\n got: %s\nwant: %s", raw, want)
	}
	// the object carries the kind-required "behavior" key and the
	// answers keyed by question text — a decode-side check so a future
	// shape drift fails on SEMANTICS, not just bytes.
	var decoded struct {
		Behavior     string `json:"behavior"`
		UpdatedInput struct {
			Questions []struct {
				Question string `json:"question"`
			} `json:"questions"`
			Answers map[string]string `json:"answers"`
		} `json:"updatedInput"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("result must be a JSON object: %v", err)
	}
	if decoded.Behavior != "allow" {
		t.Fatalf("result.behavior drifted: %q", decoded.Behavior)
	}
	if len(decoded.UpdatedInput.Questions) != 2 ||
		decoded.UpdatedInput.Questions[0].Question != "Which finishes for the east wall?" {
		t.Fatalf("updatedInput.questions drifted: %+v", decoded.UpdatedInput.Questions)
	}
	if decoded.UpdatedInput.Answers["Which finishes for the east wall?"] != "matte, gloss" {
		t.Fatalf("answers map drifted: %+v", decoded.UpdatedInput.Answers)
	}
}

// TestClaudeDialogAnswerRoundTrip drives a shell stub that parks ONE
// request_user_dialog (the REAL wire shape: dialog_kind
// permission_ask_user_question + payload.questions), then answers it
// with a MULTI-PAGE multi-select reply and dismisses a second dialog —
// asserting the exact stdin bytes the CLI parses (behavior completed +
// the CLI-native result object; behaviour cancelled bare).
func TestClaudeDialogAnswerRoundTrip(t *testing.T) {
	capture := tempCaptureLog(t)
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"dlg-0001","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01AUQ1","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Which finishes for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep"}],"multiSelect":false}]},"tool_use_id":"toolu_01AUQ1"},"session_id":"sess-sh-1"}'
      printf '%s\n' '{"type":"control_request","request_id":"dlg-0002","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01AUQ2","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Ship it?","header":"Ship","options":[{"label":"yes","description":"Send it"},{"label":"no","description":"Hold"}],"multiSelect":false}]},"tool_use_id":"toolu_01AUQ2"},"session_id":"sess-sh-1"}'
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

	if err := b.Send("wall time"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "both dialogs mapped", 3*time.Second, func() bool {
		var d1, d2 bool
		for _, e := range log.snapshot() {
			if e.Kind == state.EvQuestion && e.ToolState == "pending" {
				switch e.QuestionID {
				case "dlg-0001":
					d1 = true
				case "dlg-0002":
					d2 = true
				}
			}
		}
		return d1 && d2
	})

	// page 1 multi-selects two finishes, page 2 picks one pattern: the
	// result is the CLI-native OBJECT — behavior "allow" + updatedInput
	// (the asked questions re-emitted, the answers map keyed by question
	// text, multi-select joined ", ") — riding behavior "completed".
	if err := b.AnswerQuestion("dlg-0001", [][]string{{"matte", "gloss"}, {"sprayed"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	// the second dialog is dismissed: behaviour cancelled, NO result key.
	if err := b.RejectQuestion("dlg-0002"); err != nil {
		t.Fatalf("RejectQuestion: %v", err)
	}

	claudeWait(t, "both control_response lines written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 3 // 1 user + 2 responses
	})
	lines := claudeCapture(t, capture)
	wants := []string{
		`{"type":"control_response","response":{"subtype":"success","request_id":"dlg-0001","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finishes for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen"},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim"}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep"}],"multiSelect":false}],"answers":{"Which finishes for the east wall?":"matte, gloss","Which spray pattern?":"sprayed"}}}}}}`,
		`{"type":"control_response","response":{"subtype":"success","request_id":"dlg-0002","response":{"behavior":"cancelled"}}}`,
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
	// the cancelled line carries no answer payload keys at all
	for _, got := range lines {
		if strings.Contains(got, "dlg-0002") && (strings.Contains(got, `"result"`) || strings.Contains(got, `"answer"`) || strings.Contains(got, `"message"`)) {
			t.Fatalf("a cancelled dialog carries NO payload keys: %q", got)
		}
	}
	// both dialogs resolved locally (the modal closer)
	var answered, rejected int
	for _, e := range log.snapshot() {
		if e.Kind == state.EvQuestion && e.ToolState == "resolved" {
			switch e.ToolSummary {
			case "answered":
				answered++
			case "rejected":
				rejected++
			}
		}
	}
	if answered != 1 || rejected != 1 {
		t.Fatalf("resolved events drifted: answered=%d/1 rejected=%d/1", answered, rejected)
	}
}

// TestClaudeAskUserResultPreviewMetadata: the result builder with the
// FULL wire context — option previews re-emitted verbatim (omitempty per
// option: the preview-less "gloss"/"cross-hatch" options keep their old
// bytes) and the payload-level metadataSource/metadata re-emitted raw at
// the updatedInput level, exactly as the CLI's own submit builder spreads
// the original tool input. Byte-pinned; the semantic decode guards the
// same fields against a byte-lucky shape drift.
func TestClaudeAskUserResultPreviewMetadata(t *testing.T) {
	items := []state.QuestionItem{
		{
			Question: "Which finish for the east wall?", Header: "Finish", Multiple: true,
			Options: []state.QuestionOption{
				{Label: "matte", Description: "Flat, no sheen", Preview: "# Matte\nA flat, non-reflective coat."},
				{Label: "gloss", Description: "High sheen"},
				{Label: "both", Description: "Matte base, gloss trim", Preview: "# Both\nMatte base, gloss trim."},
			},
			Meta:       json.RawMessage(`{"source":"paint-survey","traceId":"tr-42"}`),
			MetaSource: json.RawMessage(`"paint-survey"`),
		},
		{
			Question: "Which spray pattern?", Header: "Pattern",
			Options: []state.QuestionOption{
				{Label: "cross-hatch", Description: "Two perpendicular passes"},
				{Label: "single-pass", Description: "One steady sweep", Preview: "# Single pass\nOne steady sweep, no overlap."},
			},
			Meta:       json.RawMessage(`{"source":"paint-survey","traceId":"tr-42"}`),
			MetaSource: json.RawMessage(`"paint-survey"`),
		},
	}
	raw, err := claudeAskUserResultJSON(items, [][]string{{"matte", "gloss"}, {"single-pass"}})
	if err != nil {
		t.Fatalf("claudeAskUserResultJSON: %v", err)
	}
	want := `{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen","preview":"# Matte\nA flat, non-reflective coat."},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim","preview":"# Both\nMatte base, gloss trim."}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep","preview":"# Single pass\nOne steady sweep, no overlap."}],"multiSelect":false}],"answers":{"Which finish for the east wall?":"matte, gloss","Which spray pattern?":"single-pass"},"metadataSource":"paint-survey","metadata":{"source":"paint-survey","traceId":"tr-42"}}}`
	if string(raw) != want {
		t.Fatalf("result bytes drifted:\n got: %s\nwant: %s", raw, want)
	}
	// semantic guard: preview + both metadata keys survive a decode.
	var decoded struct {
		UpdatedInput struct {
			Questions []struct {
				Options []struct {
					Label   string `json:"label"`
					Preview string `json:"preview"`
				} `json:"options"`
			} `json:"questions"`
			MetadataSource string          `json:"metadataSource"`
			Metadata       json.RawMessage `json:"metadata"`
		} `json:"updatedInput"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("result must be a JSON object: %v", err)
	}
	opts := decoded.UpdatedInput.Questions[0].Options
	if opts[0].Preview != "# Matte\nA flat, non-reflective coat." || opts[1].Preview != "" {
		t.Fatalf("preview leg drifted: %+v", opts)
	}
	if decoded.UpdatedInput.MetadataSource != "paint-survey" ||
		string(decoded.UpdatedInput.Metadata) != `{"source":"paint-survey","traceId":"tr-42"}` {
		t.Fatalf("metadata leg drifted: %+v", decoded.UpdatedInput)
	}

	// metadataSource alone (the payload builder can lift the source
	// without a metadata object): only that key appears — and a
	// metadata-free dialog emits NEITHER key.
	raw, err = claudeAskUserResultJSON([]state.QuestionItem{{
		Question: "Ship it?", Header: "Ship",
		Options:    []state.QuestionOption{{Label: "yes", Description: "Send it"}},
		MetaSource: json.RawMessage(`"deploy-check"`),
	}}, [][]string{{"yes"}})
	if err != nil {
		t.Fatalf("claudeAskUserResultJSON(source only): %v", err)
	}
	want = `{"behavior":"allow","updatedInput":{"questions":[{"question":"Ship it?","header":"Ship","options":[{"label":"yes","description":"Send it"}],"multiSelect":false}],"answers":{"Ship it?":"yes"},"metadataSource":"deploy-check"}}`
	if string(raw) != want {
		t.Fatalf("source-only bytes drifted:\n got: %s\nwant: %s", raw, want)
	}
}

// TestClaudeDialogPreviewMetadataRoundTrip — the FULL leg, byte-tested:
// a wire request whose payload carries option previews + metadataSource +
// metadata decodes into EvQuestion pages (preview on the options, the raw
// analytics bytes riding EVERY page), stashes through the backend, and
// the answer's stdin control_response re-emits ALL of it inside
// updatedInput — the granted AskUserQuestion tool call keeps its full
// original context.
func TestClaudeDialogPreviewMetadataRoundTrip(t *testing.T) {
	capture := tempCaptureLog(t)
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"control_request","request_id":"dlg-0101","request":{"subtype":"request_user_dialog","dialog_kind":"permission_ask_user_question","payload":{"requestId":"toolu_01PM1","toolName":"AskUserQuestion","permissionResult":{"behavior":"ask"},"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen","preview":"# Matte\nA flat, non-reflective coat."},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim","preview":"# Both\nMatte base, gloss trim."}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep","preview":"# Single pass\nOne steady sweep, no overlap."}],"multiSelect":false}],"metadataSource":"paint-survey","metadata":{"source":"paint-survey","traceId":"tr-42"}},"tool_use_id":"toolu_01PM1"},"session_id":"sess-sh-1"}'
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

	if err := b.Send("wall time"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "dialog mapped", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvQuestion && e.QuestionID == "dlg-0101" && e.ToolState == "pending" {
				return true
			}
		}
		return false
	})

	// the decode -> event leg: previews on the options (omitempty for the
	// preview-less one), the raw analytics bytes on EVERY page.
	var pending *state.Event
	for i, e := range log.snapshot() {
		if e.Kind == state.EvQuestion && e.QuestionID == "dlg-0101" && e.ToolState == "pending" {
			pending = &log.snapshot()[i]
		}
	}
	if pending == nil || len(pending.Questions) != 2 {
		t.Fatalf("pending dialog drifted: %+v", pending)
	}
	q1, q2 := pending.Questions[0], pending.Questions[1]
	if q1.Options[0].Preview != "# Matte\nA flat, non-reflective coat." ||
		q1.Options[2].Preview != "# Both\nMatte base, gloss trim." {
		t.Fatalf("page 1 previews drifted: %+v", q1.Options)
	}
	if q1.Options[1].Preview != "" {
		t.Fatalf("a preview-less option must decode empty: %+v", q1.Options[1])
	}
	if q2.Options[1].Preview != "# Single pass\nOne steady sweep, no overlap." {
		t.Fatalf("page 2 preview drifted: %+v", q2.Options)
	}
	for i, q := range pending.Questions {
		if string(q.MetaSource) != `"paint-survey"` ||
			string(q.Meta) != `{"source":"paint-survey","traceId":"tr-42"}` {
			t.Fatalf("page %d metadata carrier drifted: %+v", i, q)
		}
	}

	if err := b.AnswerQuestion("dlg-0101", [][]string{{"matte", "gloss"}, {"single-pass"}}); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	claudeWait(t, "control_response written", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2 // 1 user + 1 response
	})
	want := `{"type":"control_response","response":{"subtype":"success","request_id":"dlg-0101","response":{"behavior":"completed","result":{"behavior":"allow","updatedInput":{"questions":[{"question":"Which finish for the east wall?","header":"Finish","options":[{"label":"matte","description":"Flat, no sheen","preview":"# Matte\nA flat, non-reflective coat."},{"label":"gloss","description":"High sheen"},{"label":"both","description":"Matte base, gloss trim","preview":"# Both\nMatte base, gloss trim."}],"multiSelect":true},{"question":"Which spray pattern?","header":"Pattern","options":[{"label":"cross-hatch","description":"Two perpendicular passes"},{"label":"single-pass","description":"One steady sweep","preview":"# Single pass\nOne steady sweep, no overlap."}],"multiSelect":false}],"answers":{"Which finish for the east wall?":"matte, gloss","Which spray pattern?":"single-pass"},"metadataSource":"paint-survey","metadata":{"source":"paint-survey","traceId":"tr-42"}}}}}}`
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
