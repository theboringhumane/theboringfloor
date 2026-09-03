// thread_birthstamp_test.go — the merge-rewrite birth-stamp contract:
// sequential EvTool/EvThought merges for ONE CallID replace the chat
// entry's text/meta IN PLACE but keep the FIRST-seen At — a mid-stream
// update never re-stamps the entry, so its timeline slot (and its
// worker thread's sort key) stays at its birth. Before the fix every
// merge re-stamped At to time.Now() and long threads swam down the
// conversation. The fixtures back-date the first stamp to a sentinel so
// a re-stamp is deterministically visible (time.Now() is never the
// sentinel).
package app

import (
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// birthStamp — the first-seen slot every merge must preserve.
const birthStamp = int64(4242)

func TestThreadMergeKeepsBirthStamp(t *testing.T) {
	cases := []struct {
		name     string
		open     state.Event // the birth event (stream opens)
		update   state.Event // the merge rewrite (same CallID)
		wantID   string
		wantKind string
	}{
		{
			name:     "boss think",
			open:     state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss", CallID: "tc-1", Text: "hmm"},
			update:   state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss", CallID: "tc-1", Text: "hmm, done", Done: true},
			wantID:   "think-tc-1",
			wantKind: "think",
		},
		{
			name:     "boss tool",
			open:     state.Event{Kind: state.EvTool, EmployeeName: "boss", CallID: "tc-2", ToolName: "read", ToolSummary: "a.go", ToolState: "running"},
			update:   state.Event{Kind: state.EvTool, EmployeeName: "boss", CallID: "tc-2", ToolName: "read", ToolSummary: "a.go", ToolState: "done"},
			wantID:   "tool-tc-2",
			wantKind: "tool",
		},
		{
			name:     "employee wthink",
			open:     state.Event{Kind: state.EvThought, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tc-3", Text: "planning"},
			update:   state.Event{Kind: state.EvThought, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tc-3", Text: "planning, more", Done: true},
			wantID:   "wthink-tekton-1-tc-3",
			wantKind: "wthink",
		},
		{
			name:     "employee wtool",
			open:     state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tc-4", ToolName: "grep", ToolSummary: "needle", ToolState: "running"},
			update:   state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tc-4", ToolName: "grep", ToolSummary: "needle", ToolState: "done"},
			wantID:   "wtool-tekton-1-tc-4",
			wantKind: "wtool",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := reducer(state.OfficeState{}, tc.open)
			if len(st.Chat) != 1 {
				t.Fatalf("the birth event must append exactly one chat entry, got %d", len(st.Chat))
			}
			if st.Chat[0].ID != tc.wantID || st.Chat[0].Kind != tc.wantKind {
				t.Fatalf("birth entry = %s/%s, want %s/%s", st.Chat[0].ID, st.Chat[0].Kind, tc.wantID, tc.wantKind)
			}
			// back-date to the sentinel: any re-stamp lands on time.Now()
			// instead and the assertion below catches it deterministically
			st.Chat[0].At = birthStamp

			st = reducer(st, tc.update)
			if len(st.Chat) != 1 {
				t.Fatalf("the merge must replace in place (no second entry), got %d entries", len(st.Chat))
			}
			got := st.Chat[0]
			if got.At != birthStamp {
				t.Fatalf("a merge rewrite must keep the FIRST stamp %d, got %d", birthStamp, got.At)
			}
			if got.Text == "" {
				t.Fatalf("the merge must still carry the updated text, got %q", got.Text)
			}
		})
	}
}
