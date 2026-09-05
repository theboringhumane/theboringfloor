package app

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	planPresentNotice       = "plan: boss presented a plan — review, edit, ctrl+x twice to approve"
	planUpdateNotice        = "plan: boss updated the plan — review, edit, ctrl+x twice to approve"
	planApprovedHeader      = "[theboringfloor] current approved plan"
	planNoApprovedAvailable = "[theboringfloor] no approved plan available"
)

// applyPlanTools adapts agent plan-tool events to the existing plan editor
// and approval workflow. Draft presentation never overwrites the separately
// stored approval; only approveSentMsg records a new approved value.
func (m *Model) applyPlanTools(ev state.Event) tea.Cmd {
	if m.plan == nil {
		return nil
	}
	switch ev.Kind {
	case state.EvPlanPresent:
		m.setAgentMode(agentModePlan)
		m.plan.SetValue(ev.PlanToolText)
		m.plan.SetUserDirty(false)
		m.restoredPlan = false
		m.notice(planPresentNotice)
	case state.EvPlanUpdate:
		if strings.TrimSpace(ev.PlanToolText) == "" {
			m.noticeErr("plan update ignored — empty plan text")
			return nil
		}
		m.setAgentMode(agentModePlan)
		m.plan.SetValue(ev.PlanToolText)
		m.plan.SetUserDirty(false)
		m.restoredPlan = false
		m.notice(planUpdateNotice)
	case state.EvPlanGetApproved:
		followup := m.approvedPlanFollowup()
		// The tool-triggered follow-up must be visible immediately rather
		// than relying on transports to echo synthetic user prompts.
		m.st.Chat = capChat(appendChat(m.st.Chat, state.ChatMsg{
			ID:   "approved-plan-" + strconv.FormatInt(time.Now().UnixNano(), 10),
			From: "user", Kind: "user", Text: followup, At: time.Now().UnixMilli(),
		}))
		m.tabs.SetState(m.st)
		return approvedPlanCmd(m.currentBackend, followup)
	}
	return nil
}

func (m *Model) approvedPlanFollowup() string {
	approved := m.approvedPlanText()
	if approved == "" {
		return planNoApprovedAvailable
	}
	// approvedPlanText is already stored through capApprovedPlanText. Do not
	// apply a second, header-sized clip here: the tool must receive the same
	// durable approved plan, including its truncation marker when present.
	return planApprovedHeader + "\n" + approved
}

type approvedPlanResult struct{ err error }

func approvedPlanCmd(current *currentBackend, followup string) tea.Cmd {
	return func() tea.Msg {
		if err := current.send(followup, nil, ""); err != nil {
			return approvedPlanResult{err: err}
		}
		return approvedPlanResult{}
	}
}
