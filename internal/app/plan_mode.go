// plan_mode.go — the plan/build agent mode ("agentMode"): the member's
// switch between steering the boss's PLANNING pass (prompts ride
// SendAgent(text,"plan")) and the normal build pipeline (plain Send — a
// build-mode prompt never carries an "agent" key on the wire).
//
// State lives MODEL-side (never OfficeState — state.Mode already means
// live/demo): m.agentMode is "plan"|"build", mirrored into the plan pane
// via setAgentMode so the pane renders its own mode badge AND the chat
// send closure (built once in New, before any Model copy exists) reads
// the CURRENT mode at send time through the pointer-shared pane.
//
// The pane is the colleague's contract-frozen panels.PlanEditor: the app
// guards the shape with the compile assert below and drives it through a
// typed field. SetSize stays OUT of the asserted interface — its fluent
// *panels.PlanEditor return cannot ride an interface method, so the model
// sizes the pane on the concrete field at Frame time instead.
package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// Agent-mode ids. state.Mode (live/demo) is untouched — these ids ride
// MODEL state and the wire's "agent" field only.
const (
	agentModePlan  = "plan"
	agentModeBuild = "build"
)

// planEditorPane — the plan editor's contract as the app drives it
// (panels.PlanEditor implements EXACTLY this). SetSize is deliberately
// absent: the colleague's SetSize(w,h) returns *panels.PlanEditor
// (fluent), which an interface method cannot express — the model calls it
// on the concrete field at layout time instead.
type planEditorPane interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Focus() tea.Cmd
	Blur()
	Focused() bool
	Value() string
	SetValue(string)
	Mode() string
	SetMode(m string)
}

// The colleague's pane must satisfy the app-side shape — drift fails the
// build HERE, not at an assert deep in the layout code.
var _ planEditorPane = (*panels.PlanEditor)(nil)

// agentBackend — the plan/build routing seam live and demo backends expose
// beyond state.Backend (the same additive type-assert pattern as
// attachmentBackend in model.go; harness stubs without it degrade to the
// plain-text send). SendAgent sends one prompt with the agent tag riding
// the payload ("plan"|"build"); backends keep the key out of the payload
// entirely when a serve rejects it (live backend's degrade latch).
type agentBackend interface {
	SendAgent(text, agent string) error
}

// approvePrefix — the fixed compose head for an approved plan: a plan the
// member signs off leaves the editor and becomes a BUILD-agent prompt.
// Frozen copy — pinned by plan_mode_test.go.
const approvePrefix = "Approved plan — implement it exactly as specified:\n\n"

// setAgentMode is the ONE mutation point for m.agentMode — the pane's own
// mode badge (pane.SetMode) never drifts from the model's.
func (m *Model) setAgentMode(mode string) {
	m.agentMode = mode
	if m.plan != nil {
		m.plan.SetMode(mode)
	}
}

// planAgent is the agent tag the composer send paths attach in plan mode
// (build returns "" — plain Send, no agent field on the wire ever).
func (m *Model) planAgent() string {
	if m.agentMode == agentModePlan {
		return agentModePlan
	}
	return ""
}

// paneAgent is planAgent's twin for the ONE closure that cannot reach the
// model (the chat send callback built in New, captured before any Model
// copy exists): the pane pointer is shared across every copy and
// setAgentMode keeps its Mode() in lockstep, so a prompt typed at T
// routes by the mode AT T, not at app build time.
func paneAgent(plan *panels.PlanEditor) string {
	if plan != nil && plan.Mode() == agentModePlan {
		return agentModePlan
	}
	return ""
}

// sendChatMode is sendChat + the agent seam: in plan mode (agent ==
// "plan") a text-only prompt rides SendAgent(text, "plan"); everything
// else — build mode, a harness stub without the seam, a file-carrying
// prompt (attachments win over the tag: full fidelity beats metadata) —
// takes the existing attachment/plain path untouched.
func sendChatMode(b state.Backend, text string, atts []state.Attachment, agent string) error {
	if agent != "" && len(atts) == 0 {
		if ab, ok := b.(agentBackend); ok {
			return ab.SendAgent(text, agent)
		}
	}
	return sendChat(b, text, atts)
}

// togglePlanMode is ctrl+p: enter plan mode (the editor focuses; key
// routing in handleKey gives it every key but the reserved set) or leave
// back to build. The key's exclusions — focused terminal tab, open
// perm/question/model floats — live at the claim site (handleKey), not
// here.
func (m *Model) togglePlanMode() tea.Cmd {
	if m.plan == nil {
		return nil
	}
	if m.agentMode == agentModePlan {
		m.setAgentMode(agentModeBuild)
		m.plan.Blur()
		m.notice("[office] build mode — prompts go straight to the boss")
		return nil
	}
	m.setAgentMode(agentModePlan)
	m.notice("[office] plan mode — draft the plan below · ctrl+x approve → build · esc back to chat · ctrl+p exits")
	return m.plan.Focus()
}

// approvePlan is ctrl+x with the plan editor focused: the composed prompt
// (approvePrefix + the plan body) leaves through the agent seam with
// agent="build", the office flips BACK to build mode, and the editor
// resets to its starter template (an approved plan is consumed — the
// persistence projection clears with it). An empty buffer or the
// never-touched template refuses: approving boilerplate would spend a
// whole build turn on nothing.
func (m *Model) approvePlan() tea.Cmd {
	if m.plan == nil {
		return nil
	}
	v := m.plan.Value()
	if strings.TrimSpace(v) == "" || v == m.planTemplate {
		m.notice("[office] nothing to approve — edit the plan, then ctrl+x (ctrl+p exits plan mode)")
		return nil
	}
	text := approvePrefix + v
	b := m.backend
	send := func() tea.Msg {
		if b != nil {
			var err error
			if ab, ok := b.(agentBackend); ok {
				err = ab.SendAgent(text, agentModeBuild)
			} else {
				// seam-less harness stub — degrade open to the plain send
				err = sendChat(b, text, nil)
			}
			if err != nil {
				return sendErrMsg{err: err}
			}
		}
		return chatSentMsg{text: text}
	}
	// Flip first, send second — the same optimistic ordering every chat
	// send uses (the member's own echo lands synchronously, the wire POST
	// rides async; a failure surfaces through the normal sendErrMsg path).
	m.setAgentMode(agentModeBuild)
	m.plan.SetValue(m.planTemplate)
	m.plan.Blur()
	m.notice(fmt.Sprintf("[office] plan approved — sent to build (%d chars)", len(v)))
	return send
}

// planText is the persistence projection (session.json's planText field):
// the buffer content worth keeping across boots. An empty or never-edited
// starter template is NOTHING ("", and the omitempty tag drops it from
// the file) — a pristine editor must not fake a saved plan.
func (m *Model) planText() string {
	if m.plan == nil {
		return ""
	}
	v := m.plan.Value()
	if strings.TrimSpace(v) == "" || v == m.planTemplate {
		return ""
	}
	return v
}

// agentBadge is the statusbar's plan/build marker segment: "[plan]" while
// plan mode is active, "" in build — the default office's statusbar stays
// byte-identical to before (the badge only ever appears during a plan
// session).
func (m *Model) agentBadge() string {
	if m.agentMode == agentModePlan {
		return "[plan]"
	}
	return ""
}
