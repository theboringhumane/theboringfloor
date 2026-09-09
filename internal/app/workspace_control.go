package app

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func (m *Model) applyWorkspaceAction(ev state.Event) tea.Cmd {
	registry := controlReplies.Load()
	if !registry.Pending(ev.ControlReqID) {
		return nil
	}
	var action control.WorkspaceAction
	var cmd tea.Cmd
	failure := ""
	if json.Unmarshal([]byte(ev.ControlText), &action) != nil {
		failure = "Invalid workspace action"
	} else {
		switch action.Action {
		case "team-add":
			_, err := workspace.AddTeam(m.sessDir, action.Text)
			if err != nil {
				failure = err.Error()
			} else {
				cmd = m.floors.Refresh()
			}
		case "ticket-save":
			var ticket workspace.Ticket
			if json.Unmarshal(action.Ticket, &ticket) != nil {
				failure = "Invalid ticket"
				break
			}
			_, err := workspace.PutTicket(m.sessDir, ticket)
			if err != nil {
				failure = err.Error()
			} else {
				cmd = m.tickets.Refresh()
			}
		case "conversation":
			if m.execFloor != nil {
				failure = "A conversation is already opening"
				break
			}
			if !config.ValidBackendName(action.Backend) {
				failure = "Choose Claude Code, OpenCode, or Codex"
				break
			}
			if !action.Fresh {
				if _, ok := loadConversation(m.sessDir, action.Backend, action.Session); !ok {
					failure = "Conversation not found"
					break
				}
			}
			floor, err := workspace.Load(m.sessDir)
			if err != nil {
				failure = "Could not load floor"
				break
			}
			valid := action.Team == ""
			for _, team := range floor.Teams {
				valid = valid || team.ID == action.Team
			}
			if !valid {
				failure = "Choose a team on this floor"
				break
			}
			if why := m.backendSwapBlockers(); len(why) > 0 {
				failure = "Finish or stop current work first: " + strings.Join(why, "; ")
				break
			}
			cmd = m.launchFloor(FloorLaunch{Dir: m.sessDir, Backend: action.Backend, Session: action.Session, Title: strings.TrimSpace(action.Title), Team: action.Team, Fresh: action.Fresh})
			if cmd == nil && m.execFloor == nil && (action.Fresh || action.Session != m.PrimarySessionID() || action.Backend != m.backendName()) {
				failure = "Could not open conversation"
				if n := len(m.st.Chat); n > 0 {
					failure = m.st.Chat[n-1].Text
				}
			}
		case "plan-update", "plan-approve":
			if m.plan == nil || m.plan.Value() != action.Expected {
				failure = "The plan changed. Refresh and review the latest draft."
				break
			}
			if m.controlWorking() || m.remotePlanPending || m.permQ.front() != nil || m.question != nil {
				failure = "Wait for current work or answer the pending request first"
				break
			}
			if action.Action == "plan-update" {
				if strings.TrimSpace(action.Text) == "" {
					failure = "Plan cannot be empty"
					break
				}
				m.applyPlanTools(state.Event{Kind: state.EvPlanUpdate, PlanToolText: action.Text})
				m.plan.SetUserDirty(true)
			} else {
				// The phone explicitly confirmed this exact draft, including restored text.
				m.plan.SetUserDirty(true)
				if why := m.approveRefusal(); why != "" {
					failure = why
					break
				}
				m.remotePlanPending = true
				m.remotePlanError = ""
				cmd = m.approvePlan()
			}
		default:
			failure = "Unknown workspace action"
		}
	}
	var payload any = control.OKResponse{OK: true}
	if failure != "" {
		payload = control.ErrorResponse{Error: failure}
		cmd = nil
	}
	if !registry.Fulfill(ev.ControlReqID, marshalControlResponse(payload)) {
		if action.Action == "plan-approve" {
			m.remotePlanPending = false
		}
		if action.Action == "conversation" {
			m.execFloor = nil
		}
		return nil
	}
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		if ev.ControlAck != nil {
			<-ev.ControlAck
		}
		return cmd()
	}
}
