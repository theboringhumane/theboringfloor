package app

import (
	"encoding/json"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func TestTicketLinkRejectsChangedConversation(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := t.TempDir()
	floor, err := workspace.PutTicket(dir, workspace.Ticket{Title: "Review invitations", Status: "backlog", Priority: "P1"})
	if err != nil {
		t.Fatal(err)
	}
	ticket := floor.Tickets[0]
	m := New(&pinBackend{primary: "current-session"}, nil)
	m.sessDir = dir
	for _, session := range []string{"old-session", "current-session"} {
		registry := controlReplies.Load()
		id, reply := registry.NewRequest()
		raw, _ := json.Marshal(control.WorkspaceAction{Action: "ticket-link", TicketID: ticket.ID, Backend: m.backendName(), Session: session, ExpectedUpdated: ticket.Updated})
		m.applyWorkspaceAction(state.Event{Kind: state.EvControlWorkspace, ControlReqID: id, ControlText: string(raw)})
		var response struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		}
		json.Unmarshal(<-reply, &response)
		if (session == "current-session") != response.OK {
			t.Fatalf("session %s: %+v", session, response)
		}
	}
	floor, _ = workspace.Load(dir)
	if floor.Tickets[0].Session != "current-session" || floor.Tickets[0].Status != "backlog" {
		t.Fatal(floor.Tickets[0])
	}
}

func TestMobilePlanRejectsStaleApproval(t *testing.T) {
	m := New(&recBackend{}, nil)
	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: "# Current plan\n\n- Verify"})
	registry := controlReplies.Load()
	id, reply := registry.NewRequest()
	raw, _ := json.Marshal(control.WorkspaceAction{Action: "plan-approve", Expected: "# Old plan"})
	if cmd := m.applyWorkspaceAction(state.Event{Kind: state.EvControlWorkspace, ControlReqID: id, ControlText: string(raw)}); cmd != nil {
		t.Fatal("stale approval started a command")
	}
	var failure control.ErrorResponse
	json.Unmarshal(<-reply, &failure)
	if failure.Error == "" || m.remotePlanPending {
		t.Fatal("stale approval was accepted")
	}
}
func TestMobilePlanUpdateAndCancelledRequest(t *testing.T) {
	m := New(&recBackend{}, nil)
	old := "# Plan\n\n- One"
	updated := "# Plan\n\n- Two"
	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: old})
	registry := controlReplies.Load()
	id, reply := registry.NewRequest()
	raw, _ := json.Marshal(control.WorkspaceAction{Action: "plan-update", Expected: old, Text: updated})
	m.applyWorkspaceAction(state.Event{Kind: state.EvControlWorkspace, ControlReqID: id, ControlText: string(raw)})
	var result control.OKResponse
	json.Unmarshal(<-reply, &result)
	if !result.OK || m.plan.Value() != updated {
		t.Fatalf("update rejected: %+v", result)
	}
	id, _ = registry.NewRequest()
	registry.Cancel(id)
	raw, _ = json.Marshal(control.WorkspaceAction{Action: "plan-update", Expected: updated, Text: old})
	m.applyWorkspaceAction(state.Event{Kind: state.EvControlWorkspace, ControlReqID: id, ControlText: string(raw)})
	if m.plan.Value() != updated {
		t.Fatal("cancelled action changed the plan")
	}
}
func TestMobilePlanApprovalWaitsForHTTPAcknowledgement(t *testing.T) {
	m := New(&recBackend{}, nil)
	draft := "# Plan\n\n- Verify"
	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: draft})
	registry := controlReplies.Load()
	id, reply := registry.NewRequest()
	ack := make(chan struct{})
	raw, _ := json.Marshal(control.WorkspaceAction{Action: "plan-approve", Expected: draft})
	cmd := m.applyWorkspaceAction(state.Event{Kind: state.EvControlWorkspace, ControlReqID: id, ControlText: string(raw), ControlAck: ack})
	var result control.OKResponse
	json.Unmarshal(<-reply, &result)
	if !result.OK || cmd == nil || !m.remotePlanPending {
		t.Fatal("approval not accepted")
	}
	close(ack)
	if cmd() == nil {
		t.Fatal("approval produced no completion")
	}
}
