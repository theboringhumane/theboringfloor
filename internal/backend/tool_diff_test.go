// tool_diff_test.go — the per-CALL thread-diff wire contract: a completed
// edit ToolPart whose state.metadata.filediff rides the SSE frame lifts
// ONE EvFileDiff attributed to that call (CallID set) RIGHT AFTER its
// EvTool; a write/create part synthesizes the new-file pseudo-diff from
// state.input.content (never git); repeats of one call dedupe; the
// per-file completion fetch (diffSeen) is superseded for that path only;
// older metadata-less serves and NON-completed parts stay byte-silent,
// and the boss keeps today's per-file flow untouched.
package backend

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// partUpdated wraps one ToolPart body in the message.part.updated envelope.
func partUpdated(t *testing.T, body string) ocSSEEvent {
	t.Helper()
	return ocSSEEvent{Type: "message.part.updated", Properties: json.RawMessage(body)}
}

// hireDiffKid seats one child session so actorFor attributes its parts.
func hireDiffKid(t *testing.T, ctx *normCtx, primary, kid, title string) {
	t.Helper()
	hired := mapOCEvent(ocSSEEvent{Type: "session.created", Properties: json.RawMessage(
		`{"info":{"id":"` + kid + `","parentID":"` + primary + `","title":"` + title + `","time":{"created":1,"updated":1}}}`)},
		ctx, primary, 100)
	if len(hired) != 2 {
		t.Fatalf("child must hire+dispatch first, got %v", hired)
	}
}

// kinds tallies the event kinds of one frame.
func kinds(evs []state.Event) map[state.EventKind]int {
	out := map[state.EventKind]int{}
	for _, e := range evs {
		out[e.Kind]++
	}
	return out
}

// TestToolPartCompletedEditEmitsPerCallDiff: the wire's completed Edit
// frame carries metadata.filediff{file,patch,additions,deletions} — the
// mapping emits the classic EvTool(done) PLUS one EvFileDiff keyed by the
// part's callID, with the path/counts verbatim and the compacted body.
func TestToolPartCompletedEditEmitsPerCallDiff(t *testing.T) {
	ctx := newNormCtx(nil)
	primary := "ses-primary"
	hireDiffKid(t, ctx, primary, "ses-kid", "patch the lexer (developer)")

	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-1","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-1","tool":"edit",`+
			`"state":{"status":"completed","title":"internal/panels/chat.go","input":{"filePath":"internal/panels/chat.go"},`+
			`"metadata":{"filediff":{"file":"internal/panels/chat.go","patch":"--- a/internal/panels/chat.go\n+++ b/internal/panels/chat.go\n@@ -10,2 +10,3 @@\n ctx\n-old\n+new one\n+new two\n","additions":2,"deletions":1}}}}}`),
		ctx, primary, 1000)
	k := kinds(evs)
	if k[state.EvTool] != 1 || k[state.EvFileDiff] != 1 {
		t.Fatalf("a completed edit with metadata must emit EvTool + EvFileDiff, got %v", kinds(evs))
	}
	var d state.Event
	for _, e := range evs {
		if e.Kind == state.EvFileDiff {
			d = e
		}
	}
	empName := "tekton-1"
	for _, e := range evs {
		if e.Kind == state.EvTool {
			empName = e.EmployeeName
		}
	}
	if d.CallID != "call-1" {
		t.Fatalf("the per-call diff must carry the part's callID, got %q", d.CallID)
	}
	if d.SessionID != "ses-kid" || d.EmployeeName != empName {
		t.Fatalf("the per-call diff must inherit the tool's session + employee, got %+v", d)
	}
	if d.DiffPath != "internal/panels/chat.go" || d.DiffAdd != 2 || d.DiffDel != 1 {
		t.Fatalf("diff path/counts mismatch: %+v", d)
	}
	if !strings.Contains(d.DiffBody, "@@ -10,2 +10,3 @@") || !strings.Contains(d.DiffBody, "+new one") || !strings.Contains(d.DiffBody, "-old") {
		t.Fatalf("diff body must carry the (compacted) patch, got %q", d.DiffBody)
	}
	// the EvTool itself rides unchanged
	for _, e := range evs {
		if e.Kind == state.EvTool && (e.ToolState != "done" || e.ToolName != "edit" || e.CallID != "call-1") {
			t.Fatalf("the EvTool must stay exactly today's shape, got %+v", e)
		}
	}
}

// TestToolPartWriteSynthesizesNewFileDiff: Write parts carry no patch
// (metadata is diagnostics/filepath/exists) — the new file's body in
// state.input.content becomes a presentation-only "--- /dev/null"
// pseudo-diff with add-line counts.
func TestToolPartWriteSynthesizesNewFileDiff(t *testing.T) {
	ctx := newNormCtx(nil)
	primary := "ses-primary"
	hireDiffKid(t, ctx, primary, "ses-kid", "write the file (developer)")

	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-2","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-w","tool":"write",`+
			`"state":{"status":"completed","title":"facts/new.txt","input":{"filePath":"facts/new.txt","content":"alpha\nbeta\n"},`+
			`"metadata":{"diagnostics":{},"filepath":"facts/new.txt","exists":false}}}}`),
		ctx, primary, 1000)
	k := kinds(evs)
	if k[state.EvFileDiff] != 1 {
		t.Fatalf("a completed write must synthesize ONE per-call diff, got %v", k)
	}
	var d state.Event
	for _, e := range evs {
		if e.Kind == state.EvFileDiff {
			d = e
		}
	}
	if d.CallID != "call-w" || d.DiffPath != "facts/new.txt" || d.DiffAdd != 2 || d.DiffDel != 0 {
		t.Fatalf("write pseudo-diff mismatch: %+v", d)
	}
	for _, want := range []string{"--- /dev/null", "+++ b/facts/new.txt", "@@ -0,0 +1,2 @@", "+alpha", "+beta"} {
		if !strings.Contains(d.DiffBody, want) {
			t.Fatalf("write pseudo-diff body missing %q:\n%s", want, d.DiffBody)
		}
	}
}

// TestToolCallDiffDedupesPerCall: repeated completed frames of the SAME
// callID emit the per-call diff exactly once; a SECOND call on the same
// path still emits its own (dedupe keys sessions+calls, not paths).
func TestToolCallDiffDedupesPerCall(t *testing.T) {
	ctx := newNormCtx(nil)
	primary := "ses-primary"
	hireDiffKid(t, ctx, primary, "ses-kid", "patch twice (developer)")
	frame := func(call, patch string) ocSSEEvent {
		return partUpdated(t,
			`{"part":{"id":"part-`+call+`","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"`+call+`","tool":"edit",`+
				`"state":{"status":"completed","title":"a.go","input":{"filePath":"a.go"},`+
				`"metadata":{"filediff":{"file":"a.go","patch":"`+patch+`"}}}}}`)
	}
	// the patch rides as JSON-escaped text (\n escapes, not raw newlines)
	const patch = `--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-x\n+y\n`
	if got := kinds(mapOCEvent(frame("call-1", patch), ctx, primary, 1000))[state.EvFileDiff]; got != 1 {
		t.Fatalf("first frame must emit the per-call diff, got %d", got)
	}
	if got := kinds(mapOCEvent(frame("call-1", patch), ctx, primary, 1001))[state.EvFileDiff]; got != 0 {
		t.Fatalf("a repeat of the same completed frame must NOT re-emit, got %d", got)
	}
	if got := kinds(mapOCEvent(frame("call-2", patch), ctx, primary, 1002))[state.EvFileDiff]; got != 1 {
		t.Fatalf("a SECOND call on the same path gets its own per-call diff, got %d", got)
	}
}

// TestToolCallDiffSuppressesPerFileFetch: the path a per-call diff
// covered is marked in ctx.diffSeen — the completion-time fetchDiffAndEmit
// (via diffEvent) for that same file is superseded; OTHER files still
// surface their per-file summary.
func TestToolCallDiffSuppressesPerFileFetch(t *testing.T) {
	ctx := newNormCtx(nil)
	primary := "ses-primary"
	hireDiffKid(t, ctx, primary, "ses-kid", "patch the lexer (developer)")
	mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-9","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-9","tool":"edit",`+
			`"state":{"status":"completed","title":"chat.go","input":{"filePath":"internal/panels/chat.go"},`+
			`"metadata":{"filediff":{"file":"internal/panels/chat.go","patch":"--- a/internal/panels/chat.go\n+++ b/internal/panels/chat.go\n@@ -1 +1 @@\n-a\n+b\n","additions":1,"deletions":1}}}}}`),
		ctx, primary, 1000)
	empID, empName, _ := actorFor("ses-kid", ctx, primary)
	if _, ok := diffEvent("ses-kid", empID, empName,
		ocSnapshotFileDiff{File: "internal/panels/chat.go", Patch: "x", Additions: 1}, ctx); ok {
		t.Fatalf("the per-file fetch for a per-call-covered path must be SUPPRESSED")
	}
	if _, ok := diffEvent("ses-kid", empID, empName,
		ocSnapshotFileDiff{File: "internal/other.go", Patch: "x", Additions: 1}, ctx); !ok {
		t.Fatalf("a different path's per-file summary must still surface")
	}
}

// TestToolCallDiffDegradesSilently: everything the new mapping must NOT
// touch — a running tool part (metadata ignored), a completed part with
// NO metadata at all (older serve), a completed metadata-less part whose
// REPEAT frame arrives with metadata later (the late frame still lifts),
// a non-edit tool, and the BOSS's own completed edit (per-file flow stays
// the boss's only diff lane, diffSeen unmarked).
func TestToolCallDiffDegradesSilently(t *testing.T) {
	ctx := newNormCtx(nil)
	primary := "ses-primary"
	hireDiffKid(t, ctx, primary, "ses-kid", "patch the lexer (developer)")

	// running frame: nothing extra, and it must NOT consume the call's slot
	evs := mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-r","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-r","tool":"edit",`+
			`"state":{"status":"running","title":"a.go","input":{"filePath":"a.go"},`+
			`"metadata":{"filediff":{"file":"a.go","patch":"+x\n"}}}}}`),
		ctx, primary, 1000)
	if kinds(evs)[state.EvFileDiff] != 0 {
		t.Fatalf("a RUNNING part must never emit the per-call diff, got %v", kinds(evs))
	}

	// completed, metadata-less (older serve): today's exact behavior
	evs = mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-n","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-n","tool":"edit",`+
			`"state":{"status":"completed","title":"a.go","input":{"filePath":"a.go"}}}}`),
		ctx, primary, 1001)
	if kinds(evs)[state.EvFileDiff] != 0 {
		t.Fatalf("a metadata-less completed frame must stay silent, got %v", kinds(evs))
	}
	// …and when a LATER frame of the SAME call arrives WITH metadata, the
	// diff still lifts (dedupe marks only on emission)
	evs = mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-n","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-n","tool":"edit",`+
			`"state":{"status":"completed","title":"a.go","input":{"filePath":"a.go"},`+
			`"metadata":{"filediff":{"file":"a.go","patch":"--- a/a.go\n+++ b/a.go\n@@ -1 +1 @@\n-a\n+b\n"}}}}}`),
		ctx, primary, 1002)
	if kinds(evs)[state.EvFileDiff] != 1 {
		t.Fatalf("a later completed frame WITH metadata must lift the diff, got %v", kinds(evs))
	}

	// a completed non-edit tool (bash) never emits
	evs = mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-b","sessionID":"ses-kid","messageID":"msg-1","type":"tool","callID":"call-b","tool":"bash",`+
			`"state":{"status":"completed","title":"go test","input":{"command":"go test"},`+
			`"metadata":{"filediff":{"file":"x","patch":"+x\n"}}}}}`),
		ctx, primary, 1003)
	if kinds(evs)[state.EvFileDiff] != 0 {
		t.Fatalf("a completed non-edit tool must stay silent, got %v", kinds(evs))
	}

	// the BOSS's completed edit with metadata: NO per-call event, diffSeen
	// untouched — the completion-time per-file fetch stays his only lane
	evs = mapOCEvent(partUpdated(t,
		`{"part":{"id":"part-z","sessionID":"ses-primary","messageID":"msg-1","type":"tool","callID":"call-z","tool":"edit",`+
			`"state":{"status":"completed","title":"z.go","input":{"filePath":"z.go"},`+
			`"metadata":{"filediff":{"file":"z.go","patch":"--- a/z.go\n+++ b/z.go\n@@ -1 +1 @@\n-z\n+w\n"}}}}}`),
		ctx, primary, 1004)
	if kinds(evs)[state.EvFileDiff] != 0 {
		t.Fatalf("the boss keeps the per-file flow only — no per-call event, got %v", kinds(evs))
	}
	if _, ok := diffEvent("ses-primary", "boss", "boss",
		ocSnapshotFileDiff{File: "z.go", Patch: "x", Additions: 1}, ctx); !ok {
		t.Fatalf("the boss's per-file fetch must NOT be suppressed by the untouched metadata")
	}
}
