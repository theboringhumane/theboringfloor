// control_answer_test.go — a permission prompt or boss question answered
// remotely (control.RoutePermissionAnswer/RouteQuestionAnswer) rides the
// SAME queue/hold bookkeeping and backend seam as the keyboard's y/a/n and
// wizard answers: a matching id unblocks exactly like a keypress would, a
// stale/wrong/nonexistent id never mutates local state and never reaches
// the backend, and /v1/busy exposes the id+text a remote client needs to
// answer safely.
package app

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// controlAnswerBackend — recBackend plus a recording RejectQuestion seam
// (recBackend's own RejectQuestion is a silent no-op).
type controlAnswerBackend struct {
	recBackend
	rejected []string
}

func (b *controlAnswerBackend) RejectQuestion(id string) error {
	b.rejected = append(b.rejected, id)
	return nil
}

// controlAnswerReply arms a fresh registry request the same way
// controlBusyPayload (control_mutations_test.go) does for reads, then runs
// ev through applyControlMutations and returns the registry's reply body
// alongside the resulting command.
func controlAnswerReply(t *testing.T, m *Model, ev state.Event) (string, tea.Cmd) {
	t.Helper()
	registry := control.NewRegistry()
	SetControlRegistry(registry)
	t.Cleanup(func() { SetControlRegistry(control.NewRegistry()) })
	id, reply := registry.NewRequest()
	ev.ControlReqID = id
	cmd := m.applyControlMutations(ev)
	return string(<-reply), cmd
}

func mustErrorResponse(t *testing.T, body string) control.ErrorResponse {
	t.Helper()
	var failure control.ErrorResponse
	if err := json.Unmarshal([]byte(body), &failure); err != nil || failure.Error == "" {
		t.Fatalf("reply = %s, want a non-empty ErrorResponse", body)
	}
	return failure
}

func TestControlPermissionAnswerCorrectIDUnblocks(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-1",
		EmployeeName: "boss", ToolName: "Bash", ToolSummary: "rm -rf /tmp/x", ToolState: "pending"})
	if m.permQ.front() == nil {
		t.Fatal("setup: permission must be pending")
	}

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlPermissionAnswer, ControlPermissionID: "perm-1", ControlPermissionResponse: "once",
	})
	if got != `{"ok":true}` {
		t.Fatalf("reply = %s, want {\"ok\":true}", got)
	}
	if m.permQ.front() != nil {
		t.Fatalf("the matching answer must pop the front, got %+v", m.permQ.front())
	}
	if cmd == nil {
		t.Fatal("a matched answer must return the backend send command")
	}
	if msg := cmd(); msg != nil {
		if failure, ok := msg.(sendErrMsg); ok {
			t.Fatalf("backend call failed: %v", failure.err)
		}
	}
	if len(b.permAnswers) != 1 || b.permAnswers[0] != [2]string{"perm-1", "once"} {
		t.Fatalf("AnswerPermission call = %+v, want [[perm-1 once]]", b.permAnswers)
	}
}

func TestControlPermissionAnswerWrongIDConflict(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-1",
		EmployeeName: "boss", ToolName: "Bash", ToolSummary: "rm -rf /tmp/x", ToolState: "pending"})

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlPermissionAnswer, ControlPermissionID: "perm-STALE", ControlPermissionResponse: "once",
	})
	if cmd != nil {
		t.Fatal("a stale id must not return a backend command")
	}
	mustErrorResponse(t, got)
	if p := m.permQ.front(); p == nil || p.ID != "perm-1" {
		t.Fatalf("the real pending prompt must be untouched: %+v", p)
	}
	if len(b.permAnswers) != 0 {
		t.Fatalf("a stale id must never reach AnswerPermission, got %+v", b.permAnswers)
	}
}

func TestControlPermissionAnswerNoPendingConflict(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlPermissionAnswer, ControlPermissionID: "perm-1", ControlPermissionResponse: "once",
	})
	if cmd != nil {
		t.Fatal("no pending prompt must not return a command")
	}
	mustErrorResponse(t, got)
	if len(b.permAnswers) != 0 {
		t.Fatal("AnswerPermission must never fire when nothing is pending")
	}
}

func TestControlQuestionAnswerCorrectIDMultiSelect(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-1",
		EmployeeName: "boss", ToolState: "pending",
		Questions: []state.QuestionItem{{
			Question: "which files?", Multiple: true,
			Options: []state.QuestionOption{{Label: "a.go"}, {Label: "b.go"}},
		}}})
	if m.question == nil {
		t.Fatal("setup: question must be open")
	}

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlQuestionAnswer, ControlQuestionID: "que-1",
		ControlQuestionAnswers: [][]string{{"a.go", "b.go"}},
	})
	if got != `{"ok":true}` {
		t.Fatalf("reply = %s, want {\"ok\":true}", got)
	}
	if m.question != nil {
		t.Fatalf("the matching answer must close the hold, got %+v", m.question)
	}
	if cmd == nil {
		t.Fatal("a matched answer must return the backend send command")
	}
	if msg := cmd(); msg != nil {
		if failure, ok := msg.(sendErrMsg); ok {
			t.Fatalf("backend call failed: %v", failure.err)
		}
	}
	if len(b.qAnswers) != 1 || b.qAnswers[0].id != "que-1" {
		t.Fatalf("AnswerQuestion call = %+v, want one call for que-1", b.qAnswers)
	}
	set := b.qAnswers[0].answers
	if len(set) != 1 || len(set[0]) != 2 || set[0][0] != "a.go" || set[0][1] != "b.go" {
		t.Fatalf("AnswerQuestion answers = %+v, want [[a.go b.go]]", set)
	}
}

func TestControlQuestionAnswerReject(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &controlAnswerBackend{}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-2",
		EmployeeName: "boss", ToolState: "pending", Text: "ship it?"})

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlQuestionAnswer, ControlQuestionID: "que-2", ControlQuestionReject: true,
	})
	if got != `{"ok":true}` {
		t.Fatalf("reply = %s, want {\"ok\":true}", got)
	}
	if m.question != nil {
		t.Fatalf("reject must close the hold, got %+v", m.question)
	}
	if cmd == nil {
		t.Fatal("reject must return the backend send command")
	}
	if msg := cmd(); msg != nil {
		if failure, ok := msg.(sendErrMsg); ok {
			t.Fatalf("backend call failed: %v", failure.err)
		}
	}
	if len(b.rejected) != 1 || b.rejected[0] != "que-2" {
		t.Fatalf("RejectQuestion call = %+v, want [que-2]", b.rejected)
	}
	if len(b.qAnswers) != 0 {
		t.Fatalf("reject must never call AnswerQuestion, got %+v", b.qAnswers)
	}
}

func TestControlQuestionAnswerNoPendingConflict(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlQuestionAnswer, ControlQuestionID: "que-1",
		ControlQuestionAnswers: [][]string{{"x"}},
	})
	if cmd != nil {
		t.Fatal("no pending question must not return a command")
	}
	mustErrorResponse(t, got)
	if len(b.qAnswers) != 0 {
		t.Fatal("AnswerQuestion must never fire when nothing is pending")
	}
}

func TestControlQuestionAnswerWrongIDConflict(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-1",
		EmployeeName: "boss", ToolState: "pending", Text: "ship it?"})

	got, cmd := controlAnswerReply(t, &m, state.Event{
		Kind: state.EvControlQuestionAnswer, ControlQuestionID: "que-STALE",
		ControlQuestionAnswers: [][]string{{"x"}},
	})
	if cmd != nil {
		t.Fatal("a stale id must not return a backend command")
	}
	mustErrorResponse(t, got)
	if m.question == nil || m.question.IDs[0] != "que-1" {
		t.Fatalf("the real pending question must be untouched: %+v", m.question)
	}
	if len(b.qAnswers) != 0 {
		t.Fatalf("a stale id must never reach AnswerQuestion, got %+v", b.qAnswers)
	}
}

// TestControlBusyExposesPendingPermission — /v1/busy's projection (reused by
// TestControlBusyProjection's own registry helper) must surface the id/text
// a remote client needs to answer via RoutePermissionAnswer.
func TestControlBusyExposesPendingPermission(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&recBackend{}, nil)

	if got := controlBusyPayload(t, m); strings.Contains(got, "pendingPermissionId") {
		t.Fatalf("idle busy payload must omit pending permission fields: %s", got)
	}

	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-1",
		EmployeeName: "boss", ToolName: "Bash", ToolSummary: "rm -rf /tmp/x", ToolState: "pending"})
	const want = `"pendingPermissionId":"perm-1","pendingPermissionText":"rm -rf /tmp/x"`
	if got := controlBusyPayload(t, m); !strings.Contains(got, want) {
		t.Fatalf("busy payload = %s, want to contain %s", got, want)
	}
}

func TestControlBusyExposesPendingQuestion(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&recBackend{}, nil)

	if got := controlBusyPayload(t, m); strings.Contains(got, "pendingQuestionId") {
		t.Fatalf("idle busy payload must omit pending question fields: %s", got)
	}

	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-1",
		EmployeeName: "boss", ToolState: "pending", Text: "which branch?"})
	const want = `"pendingQuestionId":"que-1","pendingQuestionText":"which branch?"`
	if got := controlBusyPayload(t, m); !strings.Contains(got, want) {
		t.Fatalf("busy payload = %s, want to contain %s", got, want)
	}
}
