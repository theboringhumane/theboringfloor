// diff_thread_test.go — the per-call worker-thread diff reducer contract:
// an EvFileDiff that carries a CallID and a NON-boss employee rides Kind
// "wdiff" pinned INSIDE the conversation right AFTER its tool call's own
// entry (the "wtool-<agent>-<callid>" row), MergesByID with the birth
// stamp kept (a repeat completed frame replaces in place), while boss-/
// fetch-level diffs (no CallID, or the boss's own) keep the flat Kind
// "diff" flow byte-identical.
package app

import (
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func workerEditTool(callID string) state.Event {
	return state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		CallID: callID, ToolName: "edit", ToolSummary: "lex.go", ToolState: "done"}
}

func workerDiff(callID, body string) state.Event {
	return state.Event{Kind: state.EvFileDiff, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		SessionID: "ses-kid", CallID: callID,
		DiffPath: "internal/panels/lex.go", DiffBody: body, DiffAdd: 2, DiffDel: 1}
}

// TestWorkerCallDiffPinsUnderItsToolAndMerges: the wdiff entry inserts
// ADJACENT to its tool row (not at the transcript end), carries the
// path␟+A␟-D Meta carrier + body Text, and a repeat frame replaces by ID
// keeping the FIRST At stamp.
func TestWorkerCallDiffPinsUnderItsToolAndMerges(t *testing.T) {
	st := reducer(state.OfficeState{}, workerEditTool("tc-1"))
	// a second, unrelated worker line after the tool: the diff must land
	// BETWEEN them (adjacent to its call), not at the very end
	st = reducer(st, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		CallID: "tc-2", ToolName: "read", ToolSummary: "wire.go", ToolState: "done"})
	st = reducer(st, workerDiff("tc-1", "--- a/lex.go\n+++ b/lex.go\n@@ -1 +1,2 @@\n-a\n+b\n+c\n"))

	if len(st.Chat) != 3 {
		t.Fatalf("tool + tool + one merged diff = 3 entries, got %d", len(st.Chat))
	}
	diff := st.Chat[1]
	if diff.ID != "wdiff-tekton-1-tc-1" || diff.Kind != "wdiff" || diff.From != "tekton-1" {
		t.Fatalf("wdiff identity mismatch: %+v", diff)
	}
	if st.Chat[0].ID != "wtool-tekton-1-tc-1" || st.Chat[2].ID != "wtool-tekton-1-tc-2" {
		t.Fatalf("the diff must pin ADJACENT to its tool call, got %q | %q | %q",
			st.Chat[0].ID, st.Chat[1].ID, st.Chat[2].ID)
	}
	if diff.Meta != "internal/panels/lex.go\x1f+2\x1f-1" {
		t.Fatalf("wdiff Meta carrier mismatch: %q", diff.Meta)
	}
	if diff.Text == "" {
		t.Fatal("wdiff Text must carry the diff body")
	}

	// back-date then REPEAT the frame: merge-by-ID keeps the birth stamp
	st.Chat[1].At = 4242
	st = reducer(st, workerDiff("tc-1", "--- a/lex.go\n+++ b/lex.go\n@@ -1 +1,3 @@\n-a\n+b\n+c\n+d\n"))
	if len(st.Chat) != 3 {
		t.Fatalf("a repeat frame must replace in place (no 4th entry), got %d", len(st.Chat))
	}
	if st.Chat[1].At != 4242 {
		t.Fatalf("the merge must keep the FIRST stamp 4242, got %d", st.Chat[1].At)
	}
	if st.Chat[1].Text == "--- a/lex.go\n+++ b/lex.go\n@@ -1 +1,2 @@\n-a\n+b\n+c\n" {
		t.Fatal("the merge must still carry the UPDATED body")
	}
	// a second call's diff gets its OWN entry, after ITS tool row
	st = reducer(st, workerDiff("tc-2", "--- a/wire.go\n+++ b/wire.go\n@@ -1 +1 @@\n-x\n+y\n"))
	if len(st.Chat) != 4 {
		t.Fatalf("a second call's diff is its own entry, got %d", len(st.Chat))
	}
	if st.Chat[3].ID != "wdiff-tekton-1-tc-2" {
		t.Fatalf("the second diff must pin under call tc-2, got %q", st.Chat[3].ID)
	}
}

// TestWorkerCallDiffOrphanFallsBackToAppend: a diff whose tool row is
// nowhere in chat (the wtool entry was capped away) simply appends — the
// thread still hosts it under that agent.
func TestWorkerCallDiffOrphanFallsBackToAppend(t *testing.T) {
	st := reducer(state.OfficeState{
		Chat: []state.ChatMsg{{ID: "u1", From: "user", Kind: "user", Text: "go", At: 1}},
	}, workerDiff("tc-9", "--- a/lex.go\n+++ b/lex.go\n@@ -1 +1 @@\n-a\n+b\n"))
	if len(st.Chat) != 2 || st.Chat[1].ID != "wdiff-tekton-1-tc-9" {
		t.Fatalf("an orphan wdiff must append after existing chat, got %+v", st.Chat)
	}
}

// TestBossAndFetchLevelDiffsStayFlat: a CallID diff attributed to the
// BOSS and a fetch-level diff with NO CallID at all both keep the flat
// Kind "diff" entries — identical to today ("diff-"+seq IDs, same Meta
// carrier, appended).
func TestBossAndFetchLevelDiffsStayFlat(t *testing.T) {
	st := reducer(state.OfficeState{}, state.Event{Kind: state.EvFileDiff,
		EmployeeID: "boss", EmployeeName: "boss", SessionID: "ses-primary", CallID: "bc-1",
		DiffPath: "z.go", DiffBody: "body", DiffAdd: 1, DiffDel: 0})
	if len(st.Chat) != 1 || st.Chat[0].Kind != "diff" || st.Chat[0].From != "boss" {
		t.Fatalf("the boss's diff must stay a flat Kind=diff entry, got %+v", st.Chat)
	}
	st = reducer(st, state.Event{Kind: state.EvFileDiff,
		EmployeeID: "dev-1", EmployeeName: "tekton-1", SessionID: "ses-kid",
		DiffPath: "wire.go", DiffBody: "body2", DiffAdd: 3, DiffDel: 2})
	if len(st.Chat) != 2 || st.Chat[1].Kind != "diff" || st.Chat[1].From != "tekton-1" {
		t.Fatalf("a fetch-level (no CallID) worker diff must stay a flat Kind=diff entry, got %+v", st.Chat)
	}
	if st.Chat[1].Meta != "wire.go\x1f+3\x1f-2" {
		t.Fatalf("the flat diff's Meta carrier is unchanged, got %q", st.Chat[1].Meta)
	}
}
