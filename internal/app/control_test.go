package app

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestControlPlanProjectionAndNilPlan(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
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
	t.Setenv("THEFLOOR_HOME", t.TempDir())
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
	t.Setenv("THEFLOOR_HOME", t.TempDir())
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

func TestControlTranscriptBackwardPaging(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	for i := 1; i <= 7; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), Text: fmt.Sprintf("message %d", i), At: int64(i)})
	}

	type page struct {
		Messages  []control.TranscriptMessage `json:"messages"`
		Truncated bool                        `json:"truncated"`
		HasMore   *bool                       `json:"hasMore"`
	}
	var all []string
	before := ""
	for wantMore := true; wantMore; {
		query := control.QueryTranscript + "?page=1"
		if before != "" {
			query += "&before=" + before
		}
		var got page
		controlQuery(t, m, query, 3, &got)
		if got.HasMore == nil {
			t.Fatal("paged response missing hasMore")
		}
		for _, message := range got.Messages {
			all = append(all, message.ID)
		}
		wantMore = *got.HasMore
		if wantMore {
			before = got.Messages[0].ID
		}
	}
	want := []string{"m-5", "m-6", "m-7", "m-2", "m-3", "m-4", "m-1"}
	if fmt.Sprintf("%v", all) != fmt.Sprintf("%v", want) {
		t.Fatalf("paged messages = %v, want %v", all, want)
	}
}

func TestControlTranscriptDefaultResponseIsByteCompatibleAndClamp(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{{ID: "one", From: "user", Kind: "user", Text: "first", At: 1}}
	response, _ := m.controlTranscript(0, "", false)
	if got := string(marshalControlResponse(response)); got != `{"messages":[{"id":"one","from":"user","kind":"user","text":"first","at":1}],"truncated":false}` {
		t.Fatalf("default response = %s", got)
	}

	m = New(&agentRecBackend{}, nil)
	for i := 0; i < controlTranscriptMax+1; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), At: int64(i)})
	}
	clamped, _ := m.controlTranscript(controlTranscriptMax+1, "", true)
	if len(clamped.Messages) != controlTranscriptMax || clamped.HasMore == nil || !*clamped.HasMore {
		t.Fatalf("clamped response = %#v", clamped)
	}
}

func TestControlTranscriptUnknownBeforeCursor(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{{ID: "known", At: 1}}
	var failure control.ErrorResponse
	controlQuery(t, m, control.QueryTranscript+"?page=1&before=unknown", 50, &failure)
	if failure.Error != "unknown before cursor" {
		t.Fatalf("unknown cursor response = %#v", failure)
	}
}

func TestControlStatusProjectionAndUnknownQuery(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
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
	t.Setenv("THEFLOOR_HOME", t.TempDir())
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
