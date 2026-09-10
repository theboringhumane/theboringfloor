package app

import (
	"github.com/theboringhumane/theboringfloor/internal/control"
	"testing"
)

func TestExecutionStatusPreservesDeferredDecisions(t *testing.T) {
	m := New(&recBackend{}, nil)
	if got := m.executionStatus(false).State; got != "ready" {
		t.Fatal(got)
	}
	m.st.BossThinking = true
	if got := m.executionStatus(false).State; got != "working" {
		t.Fatal(got)
	}
	m.agentMode = agentModePlan
	if got := m.executionStatus(false).State; got != "planning" {
		t.Fatal(got)
	}
	m.questionEscd = &questionHold{}
	if got := m.executionStatus(true); got.State != "question" || got.Action != "desktop" {
		t.Fatal(got)
	}
	m.permQ.escd = []*permPrompt{{}}
	if got := m.executionStatus(true).State; got != "permission" {
		t.Fatal(got)
	}
	m.st.Offline = true
	if got := m.executionStatus(true).State; got != "offline" {
		t.Fatal(got)
	}
}

func TestUnreviewedPlanRemainsDiscoverableWithPaneHidden(t *testing.T) {
	m := New(&recBackend{}, nil)
	m.plan.SetValue("# Proposed plan\n\n1. Build it.")
	m.agentMode = agentModeBuild
	var status control.StatusResponse
	controlQuery(t, m, control.QueryStatus, 0, &status)
	if !status.PlanPending || status.Execution.State != "plan" {
		t.Fatalf("hidden plan lost: %+v", status)
	}
	m.setApprovedPlanText(m.plan.Value())
	controlQuery(t, m, control.QueryStatus, 0, &status)
	if status.PlanPending || status.Execution.State != "ready" {
		t.Fatal(status)
	}
}
