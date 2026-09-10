package app

import "github.com/theboringhumane/theboringfloor/internal/control"

func (m *Model) executionStatus(planPending bool) *control.ExecutionStatus {
	s := &control.ExecutionStatus{State: "ready", Summary: "Ready for your next request", PlanSafety: "prompt-contract"}
	switch m.backendName() {
	case "codex":
		s.PlanSafety = "read-only-sandbox"
	case "opencode":
		s.PlanSafety = "plan-agent"
	}
	switch {
	case m.st.Offline:
		s.State, s.Summary, s.Action = "offline", "Office connection interrupted", "desktop"
	case len(m.permQ.pending)+len(m.permQ.escd) > 0:
		s.State, s.Summary, s.Action = "permission", "A tool needs your permission in the desktop office", "desktop"
	case m.question != nil || m.questionEscd != nil || m.questionParked:
		s.State, s.Summary, s.Action = "question", "The assistant is waiting for your answer in the desktop office", "desktop"
	case m.remotePlanPending:
		s.State, s.Summary = "working", "Starting the approved plan"
	case m.controlWorking():
		s.State, s.Summary = "working", "The assistant is working"
		if m.agentMode == agentModePlan {
			s.State, s.Summary = "planning", "The assistant is preparing a plan"
		} else if m.st.BossDelegating {
			s.Summary = "The team is working on delegated tasks"
		}
	case planPending:
		s.State, s.Summary, s.Action = "plan", "A plan is ready for your review", "plan"
	}
	return s
}
