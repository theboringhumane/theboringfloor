package app

import (
	"encoding/json"
	"fmt"
	"reflect"
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
		{ID: "one", From: "user", Kind: "user", Text: "first", At: 1},
		{ID: "two", From: "boss", Kind: "boss", Text: "second", At: 3},
		{ID: "three", From: "office", Kind: "office", Text: "third", At: 4},
	}, Truncated: false, Working: true}
	if fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", want) {
		t.Fatalf("tail response = %#v, want %#v", got, want)
	}
}

func TestControlTranscriptDefaultLimit(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	for i := 0; i < 51; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), From: "user", Kind: "user", Text: fmt.Sprintf("message %d", i), At: int64(i)})
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
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), From: "user", Kind: "user", Text: fmt.Sprintf("message %d", i), At: int64(i)})
	}

	type page struct {
		control.TranscriptResponse
		HasMore *bool `json:"hasMore"`
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
	if got := string(marshalControlResponse(response)); got != `{"messages":[{"id":"one","from":"user","kind":"user","text":"first","at":1}],"truncated":false,"working":false}` {
		t.Fatalf("default response = %s", got)
	}

	m = New(&agentRecBackend{}, nil)
	for i := 0; i < controlTranscriptMax+1; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: fmt.Sprintf("m-%d", i), From: "user", Kind: "user", At: int64(i)})
	}
	clamped, _ := m.controlTranscript(controlTranscriptMax+1, "", true)
	if len(clamped.Messages) != controlTranscriptMax || clamped.HasMore == nil || !*clamped.HasMore {
		t.Fatalf("clamped response = %#v", clamped)
	}
}

func TestControlTranscriptWorking(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())

	tests := []struct {
		name string
		set  func(*Model)
		want bool
	}{
		{
			name: "no session started",
			want: false,
		},
		{
			name: "session running with output in flight",
			set: func(m *Model) {
				m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true}}
			},
			want: true,
		},
		{
			name: "session finished idle",
			set: func(m *Model) {
				m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: false}}
			},
			want: false,
		},
		{
			name: "session stopped by user",
			set: func(m *Model) {
				m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true}}
				m.st.BossThinking = true
				m.stopWork()
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(&agentRecBackend{}, nil)
			if tt.set != nil {
				tt.set(&m)
			}
			got, found := m.controlTranscript(0, "", false)
			if !found || got.Working != tt.want {
				t.Fatalf("working = %v, found = %v; want %v", got.Working, found, tt.want)
			}
		})
	}
}

func TestControlTranscriptPagesCountOnlyUserMessages(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())

	tests := []struct {
		name     string
		chat     []state.ChatMsg
		limit    int
		before   string
		wantIDs  []string
		wantMore bool
	}{
		{
			name:     "activity does not consume user limit",
			chat:     transcriptRows("u1", "t1", "k1", "a1", "u2", "t2", "k2", "a2", "u3", "a3"),
			limit:    2,
			wantIDs:  []string{"u2", "t2", "k2", "a2", "u3", "a3"},
			wantMore: true,
		},
		{
			name:     "zero user messages returns all activity",
			chat:     transcriptRows("t1", "k1", "a1"),
			limit:    2,
			wantIDs:  []string{"t1", "k1", "a1"},
			wantMore: false,
		},
		{
			name:     "exact user boundary has no more history",
			chat:     transcriptRows("u1", "a1", "u2", "t2", "a2"),
			limit:    2,
			wantIDs:  []string{"u1", "a1", "u2", "t2", "a2"},
			wantMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(&agentRecBackend{}, nil)
			m.st.Chat = tt.chat
			got, found := m.controlTranscript(tt.limit, tt.before, true)
			if !found || got.HasMore == nil {
				t.Fatalf("page = %#v, found = %v", got, found)
			}
			if ids := transcriptIDs(got.Messages); fmt.Sprint(ids) != fmt.Sprint(tt.wantIDs) || *got.HasMore != tt.wantMore {
				t.Fatalf("page IDs = %v, hasMore = %v; want IDs = %v, hasMore = %v", ids, *got.HasMore, tt.wantIDs, tt.wantMore)
			}
		})
	}
}

func TestControlTranscriptTwoPageWalkHasNoGapsOrDuplicates(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = transcriptRows("u1", "t1", "a1", "u2", "k2", "a2", "u3", "t3", "a3")

	var gotIDs []string
	before := ""
	for {
		page, found := m.controlTranscript(2, before, true)
		if !found || page.HasMore == nil || len(page.Messages) == 0 {
			t.Fatalf("page = %#v, found = %v", page, found)
		}
		gotIDs = append(gotIDs, transcriptIDs(page.Messages)...)
		if !*page.HasMore {
			break
		}
		before = page.Messages[0].ID
	}
	wantIDs := []string{"u2", "k2", "a2", "u3", "t3", "a3", "u1", "t1", "a1"}
	if fmt.Sprint(gotIDs) != fmt.Sprint(wantIDs) {
		t.Fatalf("walk IDs = %v, want %v", gotIDs, wantIDs)
	}
}

func transcriptRows(ids ...string) []state.ChatMsg {
	rows := make([]state.ChatMsg, 0, len(ids))
	for i, id := range ids {
		from, kind := "boss", "assistant"
		switch id[0] {
		case 'u':
			from, kind = "user", "user"
		case 't':
			kind = "wtool"
		case 'k':
			kind = "wthink"
		}
		rows = append(rows, state.ChatMsg{ID: id, From: from, Kind: kind, At: int64(i)})
	}
	return rows
}

func transcriptIDs(messages []control.TranscriptMessage) []string {
	ids := make([]string, len(messages))
	for i, message := range messages {
		ids[i] = message.ID
	}
	return ids
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
	want := control.StatusResponse{Execution: m.executionStatus(true), PlanPending: true, PlanRevision: "094c729c5555efa040e55820e4ce0482f0c088930a67558ab5b33983657e5c9d", Dir: "/project", Backend: "opencode", PlanDraftLen: 2, PlanApprovedLen: 2, ChatCount: 2}
	if !reflect.DeepEqual(status, want) {
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
