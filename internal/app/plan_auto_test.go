package app

import (
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestSubstantialRequestsPlanBeforeSending(t *testing.T) {
	for _, text := range []string{
		"Build a complete task management system",
		"Add workspaces and separate project teams",
		"Implement the frontend and backend for ticket assignment",
		"Migrate the entire storage layer to SQLite",
	} {
		b := &agentRecBackend{}
		m := New(b, nil)
		m.prepareRequestMode(text)
		if m.agentMode != agentModePlan {
			t.Fatalf("did not plan %q", text)
		}
		currentBackendSend(m.currentBackend, m.plan, text, nil)()
		if len(b.agentCalls) != 1 || b.agentCalls[0].agent != "plan" {
			t.Fatalf("wrong routing: %+v", b)
		}
	}
	for _, text := range []string{"Fix the typo in README", "What is authentication?", "How do I build a complete app?", "/new", "Add a button", "Add a full stop to the sentence", approvePrefix + "Build a complete app"} {
		if needsPlanning(text) {
			t.Fatalf("unnecessary plan for %q", text)
		}
	}
}

func TestPlanToolsRevealFromHiddenViewsAndRejectEmptyPresent(t *testing.T) {
	m := New(&agentRecBackend{}, nil)
	m.zen, m.focusPanel, m.floorNavFocused = true, true, true
	m.mountThreadFocus("developer")
	m.tabs.SetActive(6)
	m.setTermCaptured(true)
	before := m.frameNonce
	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: "# Plan\n\n- Build the feature"})
	if !m.planPaneVisible() || m.zen || m.focusPanel || m.floorNavFocused || m.threadFocus != nil || m.tabs.ActiveIndex() != 0 || m.frameNonce == before {
		t.Fatal("plan was still hidden after presentation")
	}
	want := m.plan.Value()
	m.setAgentMode(agentModeBuild)
	m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: " \n"})
	if m.plan.Value() != want || m.agentMode != agentModeBuild {
		t.Fatal("empty presentation changed mode or lost draft")
	}
}

func TestManualBuildOverrideAppliesToOneRequest(t *testing.T) {
	m := New(&agentRecBackend{}, nil)
	m.setAgentMode(agentModePlan)
	m.togglePlanMode()
	m.prepareRequestMode("Build a complete dashboard")
	if m.agentMode != agentModeBuild {
		t.Fatal("ignored explicit build selection")
	}
	m.prepareRequestMode("Build a complete dashboard")
	if m.agentMode != agentModePlan {
		t.Fatal("planning did not resume for next request")
	}
}

func TestComposerSubmissionPlansBeforeTransport(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, chatSubmitMsg{text: "Implement the frontend and backend for a project workspace"})
	if m.agentMode != agentModePlan || len(b.agentCalls) != 1 || b.agentCalls[0].agent != agentModePlan {
		t.Fatal("composer handoff skipped plan routing")
	}
}

func TestAutomaticPlanCannotReuseOldDraftOrApprovalGesture(t *testing.T) {
	m := New(&agentRecBackend{}, nil)
	old := gatedPlan("Earlier feature", "already reviewed")
	m.plan.SetValue(old)
	m.setApprovedPlanText(old)
	m.prepareRequestMode("Build a complete dashboard")
	if m.plan.Value() != old || m.approveRefusal() == "" {
		t.Fatal("old draft lost or still approvable for a new request")
	}
	m.approveArmAt = time.Now()
	next := gatedPlan("New dashboard", "needs its own approval")
	m.applyPlanTools(state.Event{Kind: state.EvPlanUpdate, PlanToolText: next})
	if !m.approveArmAt.IsZero() || m.approveRefusal() != "" || m.approvedPlanText() != old {
		t.Fatal("new draft inherited approval state")
	}
	m = runMsg(t, m, approveSentMsg{plan: old})
	if !m.planPaneVisible() || m.plan.Value() != next {
		t.Fatal("late approval hid the newer plan")
	}
}

type planAttachmentRecorder struct {
	agentRecBackend
	attachments []state.Attachment
	agent       string
}

func (b *planAttachmentRecorder) SendAgentWith(_ string, atts []state.Attachment, agent string) error {
	b.attachments, b.agent = atts, agent
	return nil
}
func TestPlanningRetainsAttachmentAndAgent(t *testing.T) {
	b := &planAttachmentRecorder{}
	atts := []state.Attachment{{Path: "/project/design.png", Name: "design.png"}}
	if err := sendChatMode(b, "Plan this", atts, "plan"); err != nil {
		t.Fatal(err)
	}
	if b.agent != "plan" || len(b.attachments) != 1 || b.attachments[0].Path != atts[0].Path {
		t.Fatal("lost mode or file")
	}
}
