package app

import (
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

type controlMutationBackend struct {
	agentRecBackend
	abortCalls int
}

func (b *controlMutationBackend) AbortSessions() error {
	b.abortCalls++
	return nil
}

func TestControlMutationSend(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &agentRecBackend{}
	m := New(b, nil)

	cmd := m.applyControlMutations(state.Event{Kind: state.EvControlSend, ControlText: "  remote message  "})
	if cmd == nil {
		t.Fatal("remote send must return a command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("remote send command must produce a result")
	}
	if len(b.sentTexts) != 1 || b.sentTexts[0] != "remote message" {
		t.Fatalf("remote send texts = %q, want [remote message]", b.sentTexts)
	}
}

func TestControlMutationWhitespaceSendIsNoop(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)

	if cmd := m.applyControlMutations(state.Event{Kind: state.EvControlSend, ControlText: " \n\t "}); cmd != nil {
		t.Fatal("whitespace-only remote send must return nil")
	}
}

func TestControlMutationStopAbortsAndNotices(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &controlMutationBackend{}
	m := New(b, nil)

	cmd := m.applyControlMutations(state.Event{Kind: state.EvControlStop})
	if cmd == nil {
		t.Fatal("remote stop must return the abort command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("remote stop abort command must produce a result")
	}
	if b.abortCalls != 1 {
		t.Fatalf("abort calls = %d, want 1", b.abortCalls)
	}
	if got := lastOfficeMsg(m); got != "remote: stopped current work" {
		t.Fatalf("last office notice = %q", got)
	}
}

func TestControlMutationNewResetsTranscriptAndNotices(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{{ID: "old", From: "user", Text: "old transcript"}}

	if cmd := m.applyControlMutations(state.Event{Kind: state.EvControlNew}); cmd != nil {
		t.Fatal("remote new must return nil")
	}
	if len(m.st.Chat) != 2 {
		t.Fatalf("new transcript rows = %d, want 2 notices", len(m.st.Chat))
	}
	if m.st.Chat[0].Text != NewOfficeNotice || m.st.Chat[1].Text != "remote: started a new session" {
		t.Fatalf("new transcript notices = %#v", m.st.Chat)
	}
}

func TestControlMutationIgnoresUnrelatedEvent(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{{ID: "existing", From: "user", Text: "unchanged"}}

	if cmd := m.applyControlMutations(state.Event{Kind: state.EvStatus}); cmd != nil {
		t.Fatal("unrelated event must return nil")
	}
	if len(m.st.Chat) != 1 || m.st.Chat[0].Text != "unchanged" {
		t.Fatalf("unrelated event mutated transcript: %#v", m.st.Chat)
	}
}

func TestControlBusyProjection(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)

	if got := controlBusyPayload(t, m); got != `{"busy":false,"pendingBoss":false,"thinking":false,"delegating":false,"questionParked":false}` {
		t.Fatalf("idle busy payload = %s", got)
	}

	m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true}}
	m.st.BossThinking = true
	m.st.BossDelegating = true
	m.questionParked = true
	if got := controlBusyPayload(t, m); got != `{"busy":true,"pendingBoss":true,"thinking":true,"delegating":true,"questionParked":true}` {
		t.Fatalf("busy payload = %s", got)
	}
}

func controlBusyPayload(t *testing.T, m Model) string {
	t.Helper()
	registry := control.NewRegistry()
	SetControlRegistry(registry)
	t.Cleanup(func() { SetControlRegistry(control.NewRegistry()) })
	id, reply := registry.NewRequest()
	m.applyControl(state.Event{Kind: state.EvControlQuery, ControlReqID: id, ControlQuery: control.QueryBusy})
	return string(<-reply)
}
