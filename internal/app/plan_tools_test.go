package app

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestPlanToolsPresentAndUpdateKeepApprovalSeparate(t *testing.T) {
	m := New(&agentRecBackend{}, nil)
	approved := "# Approved\n\n- Keep this implementation."
	presented := gatedPlan("Initial tool plan", "presented by the boss tool")
	updated := gatedPlan("Updated tool plan", "changed before re-approval")
	m.setApprovedPlanText(approved)

	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: presented})
	if m.agentMode != agentModePlan || m.plan.Value() != presented || m.plan.UserDirty() || m.restoredPlan {
		t.Fatalf("present state: mode=%q value=%q dirty=%v restored=%v", m.agentMode, m.plan.Value(), m.plan.UserDirty(), m.restoredPlan)
	}
	if !m.planPaneVisible() {
		t.Fatal("a presented plan must show the plan pane")
	}
	if got := m.approvedPlanText(); got != approved {
		t.Fatalf("present must not replace approved plan: got %q want %q", got, approved)
	}

	m.plan.SetUserDirty(true)
	m.applyPlanTools(state.Event{Kind: state.EvPlanUpdate, PlanToolText: updated})
	if m.plan.Value() != updated || m.plan.UserDirty() {
		t.Fatalf("update replaces draft and clears dirty state: value=%q dirty=%v", m.plan.Value(), m.plan.UserDirty())
	}
	if got := m.approvedPlanText(); got != approved {
		t.Fatalf("update before approval must preserve approved plan: got %q want %q", got, approved)
	}

	m.applyPlanTools(state.Event{Kind: state.EvPlanUpdate, PlanToolText: " \n\t"})
	if m.plan.Value() != updated {
		t.Fatalf("empty update must be ignored, got %q want %q", m.plan.Value(), updated)
	}
}

func TestApprovedPlanFollowupUsesLatestApprovalAndCap(t *testing.T) {
	m := New(&agentRecBackend{}, nil)
	if got := m.approvedPlanFollowup(); got != planNoApprovedAvailable {
		t.Fatalf("no approval follow-up = %q", got)
	}
	approved := "# Ship\n\n" + strings.Repeat("界", approvedPlanMaxRunes)
	m.setApprovedPlanText(approved)
	stored := m.approvedPlanText()
	if got := utf8.RuneCountInString(stored); got != approvedPlanMaxRunes {
		t.Fatalf("stored approved plan must cap at %d runes, got %d", approvedPlanMaxRunes, got)
	}
	if got := strings.Count(stored, approvedPlanTruncatedMark); got != 1 {
		t.Fatalf("stored approved plan truncation marker count = %d, want 1", got)
	}
	followup := m.approvedPlanFollowup()
	if !strings.HasPrefix(followup, planApprovedHeader+"\n# Ship") {
		t.Fatalf("approved follow-up header/body = %q", followup[:min(len(followup), 100)])
	}
	if got := strings.TrimPrefix(followup, planApprovedHeader+"\n"); got != stored {
		t.Fatalf("approved follow-up body must equal stored capped approval")
	}
	// Repeated storage must not add a second marker or shorten the plan.
	m.setApprovedPlanText(stored)
	if got := m.approvedPlanText(); got != stored || strings.Count(got, approvedPlanTruncatedMark) != 1 {
		t.Fatalf("repeated approved storage must be idempotent, got marker count %d", strings.Count(got, approvedPlanTruncatedMark))
	}
}

func TestPlanGetApprovedUsesBackendAtCommandExecutionAndAddsMemberRow(t *testing.T) {
	old := &agentRecBackend{}
	latest := &agentRecBackend{}
	m := New(old, nil)
	m.setApprovedPlanText("# Approved\n\n- Send this to the boss.")
	cmd := m.applyPlanTools(state.Event{Kind: state.EvPlanGetApproved})
	if cmd == nil {
		t.Fatal("get-approved must schedule a follow-up")
	}
	if len(m.st.Chat) != 1 || m.st.Chat[0].From != "user" || m.st.Chat[0].Text != m.approvedPlanFollowup() {
		t.Fatalf("get-approved must add its visible member row: %+v", m.st.Chat)
	}
	m.currentBackend.replace(latest) // replacement happens before tea executes cmd
	msg := cmd()
	if result, ok := msg.(approvedPlanResult); !ok || result.err != nil {
		t.Fatalf("get-approved result = %#v", msg)
	}
	if len(old.sentTexts) != 0 || len(latest.sentTexts) != 1 || latest.sentTexts[0] != m.approvedPlanFollowup() {
		t.Fatalf("follow-up must use swapped current backend: old=%q latest=%q", old.sentTexts, latest.sentTexts)
	}
}

func TestFailedApprovalDoesNotChangeStoredApprovedPlan(t *testing.T) {
	b := &errBackend{err: errors.New("wire rejected")}
	m := New(b, nil)
	m.setApprovedPlanText("# Previously approved\n\n- Keep me.")
	m.plan.SetValue(gatedPlan("Draft", "a failing approval must not replace the stored approval"))
	m.setAgentMode(agentModePlan)
	msg := m.approvePlan()()
	if _, ok := msg.(approveErrMsg); !ok {
		t.Fatalf("failed approval result = %#v", msg)
	}
	m = runMsg(t, m, msg)
	if got := m.approvedPlanText(); got != "# Previously approved\n\n- Keep me." {
		t.Fatalf("failed approval must keep previous approved plan, got %q", got)
	}
}
