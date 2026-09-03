package backend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/plantools"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func assertPlanToolBeforeCleanPin(t *testing.T, events []state.Event, wantKind state.EventKind, wantText string) {
	t.Helper()
	plan, pin := -1, -1
	for i, e := range events {
		if e.Kind == wantKind {
			if plan >= 0 || e.PlanToolText != wantText {
				t.Fatalf("plan event drifted at %d: %+v", i, e)
			}
			plan = i
		}
		if e.Kind == state.EvChatBoss && !e.Msg.Pending && strings.HasPrefix(e.Msg.ID, "bossmsg-") {
			pin = i
			if strings.Contains(e.Msg.Text, "⟦plan-") {
				t.Fatalf("completion pin leaked plan marker: %+v", e.Msg)
			}
		}
	}
	if plan < 0 || pin < 0 || plan >= pin {
		t.Fatalf("plan event must precede clean final pin (plan=%d pin=%d): %+v", plan, pin, events)
	}
}

func TestOpenCodePlanPresentEmitsBeforeCleanPin(t *testing.T) {
	const plan = "# Ship it\n\n- run focused tests"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/session/ses-boss/message/msg-plan" {
			_, _ = w.Write([]byte(`{"info":{"id":"msg-plan","sessionID":"ses-boss","role":"assistant","finish":"stop","time":{"completed":1}},"parts":[{"type":"text","text":"Here is the plan.\n⟦plan-present⟧\n# Ship it\n\n- run focused tests\n⟦/plan-present⟧"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	b.mu.Lock()
	b.baseURL, b.primaryID = srv.URL, "ses-boss"
	b.mu.Unlock()
	info := ocMessage{ID: "msg-plan", SessionID: "ses-boss", Role: "assistant"}
	info.Time.Completed = 1
	b.maybeBossCompleted(info)
	assertPlanToolBeforeCleanPin(t, eventsMatching(log, func(state.Event) bool { return true }), state.EvPlanPresent, plan)
}

func TestClaudePlanUpdateEmitsBeforeCleanPin(t *testing.T) {
	const plan = "# Revised\n\n- keep scope tight"
	b := newClaudeBackend("true", ".", nil)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-msg-plan", From: "boss", Kind: "boss", Text: "⟦plan-update⟧\n# Revised\n\n- keep scope tight\n⟦/plan-update⟧", Pending: false,
	}})
	assertPlanToolBeforeCleanPin(t, log.snapshot(), state.EvPlanUpdate, plan)
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatBoss && e.Msg.ID == "bossmsg-msg-plan" && e.Msg.Text != "" {
			t.Fatalf("plan-only pin must scrub to an empty bubble, got %+v", e.Msg)
		}
	}
}

func TestClaudePlanGetApprovedHasReadOnlyFallback(t *testing.T) {
	b := newClaudeBackend("true", ".", nil)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-msg-get", From: "boss", Kind: "boss", Text: "⟦plan-get-approved⟧", Pending: false,
	}})
	assertPlanToolBeforeCleanPin(t, log.snapshot(), state.EvPlanGetApproved, "")
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatBoss && e.Msg.ID == "bossmsg-msg-get" && e.Msg.Text != plantools.PlanApprovalStatusRequested {
			t.Fatalf("get-only pin must settle to a useful fallback: %+v", e.Msg)
		}
	}
}
