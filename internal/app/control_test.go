package app

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestControlPlanProjectionAndNilPlan(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.plan.SetValue("draft α")
	m.setApprovedPlanText("approved β")

	var got control.PlanResponse
	controlQuery(t, m, control.QueryPlan, 0, &got)
	if got != (control.PlanResponse{Draft: "draft α", Approved: "approved β", HasApproved: true}) {
		t.Fatalf("plan response = %#v", got)
	}

	m.plan = nil
	controlQuery(t, m, control.QueryPlan, 0, &got)
	if got != (control.PlanResponse{}) {
		t.Fatalf("nil plan response = %#v", got)
	}
}

func TestControlTranscriptTailAndPendingExclusion(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{
		{ID: "one", From: "user", Kind: "user", Text: "first", At: 1},
		{ID: "pending", From: "boss", Kind: "boss", Text: "partial", At: 2, Pending: true},
		{ID: "two", From: "boss", Kind: "boss", Text: "second", At: 3},
		{ID: "three", From: "office", Kind: "office", Text: "third", At: 4},
	}

	var got control.TranscriptResponse
	controlQuery(t, m, control.QueryTranscript, 2, &got)
	want := control.TranscriptResponse{Messages: []control.TranscriptMessage{
		{ID: "two", From: "boss", Kind: "boss", Text: "second", At: 3},
		{ID: "three", From: "office", Kind: "office", Text: "third", At: 4},
	}, Truncated: true}
	if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", want) {
		t.Fatalf("tail response = %#v, want %#v", got, want)
	}
}

func TestControlTranscriptDefaultLimit(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	for i := 0; i < 51; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), Text: fmt.Sprintf("message %d", i), At: int64(i)})
	}

	var got control.TranscriptResponse
	controlQuery(t, m, control.QueryTranscript, 0, &got)
	if !got.Truncated || len(got.Messages) != 50 || got.Messages[0].ID != "m-1" || got.Messages[49].ID != "m-50" {
		t.Fatalf("default transcript = %#v", got)
	}
}

func TestControlStatusProjectionAndUnknownQuery(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.sessDir = "/project"
	m.plan.SetValue("a界")
	m.setApprovedPlanText("β界")
	m.st.Chat = []state.ChatMsg{{ID: "one"}, {ID: "two", Pending: true}}

	var status control.StatusResponse
	controlQuery(t, m, control.QueryStatus, 0, &status)
	want := control.StatusResponse{Dir: "/project", Backend: "opencode", PlanDraftLen: 2, PlanApprovedLen: 2, ChatCount: 2}
	if status != want {
		t.Fatalf("status response = %#v, want %#v", status, want)
	}

	var failure control.ErrorResponse
	controlQuery(t, m, "unknown", 0, &failure)
	if failure.Error != `unknown control query "unknown"` {
		t.Fatalf("unknown query response = %#v", failure)
	}
}

func TestControlIgnoresNonControlEvents(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	if got := m.applyControl(state.Event{Kind: state.EvStatus}); got != nil {
		t.Fatal("non-control event must return nil")
	}
}

func controlQuery(t *testing.T, m Model, query string, limit int, target any) {
	t.Helper()
	registry := control.NewRegistry()
	SetControlRegistry(registry)
	t.Cleanup(func() { SetControlRegistry(control.NewRegistry()) })
	id, reply := registry.NewRequest()
	m.applyControl(state.Event{Kind: state.EvControlQuery, ControlReqID: id, ControlQuery: query, ControlLimit: limit})
	payload := <-reply
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("unmarshal fulfilled payload %s: %v", payload, err)
	}
}
