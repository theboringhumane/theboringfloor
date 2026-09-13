package app

import (
	"sort"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Native agent names and model tokens belong to the adapter. Never map them
// onto the office's visual roles or impose OpenCode syntax on other backends.
func (m *Model) applySubmodel(fields []string) tea.Cmd {
	switch len(fields) {
	case 1:
		return m.openModelAgentPicker()
	case 2:
		return m.openTargetModelPicker(state.ModelTarget{Agent: fields[1]})
	case 3:
		return m.applyModelSelection(state.ModelTarget{Agent: fields[1]}, fields[2])
	default:
		m.noticeErr("/submodel: usage /submodel [native-agent] [native-ref]")
		return nil
	}
}

func (m *Model) openModelAgentPicker() tea.Cmd {
	req, ctx, ok := m.newModelRequest(state.ModelTarget{}, true)
	if !ok {
		return nil
	}
	_, _, agents, _ := m.currentBackend.modelGeneration()
	if !agents {
		m.invalidateModelSelection()
		m.notice("/submodel: " + req.backend + " cannot list native agent types — try /submodel <agent> <native-ref> when supported")
		return nil
	}
	m.makeModelPicker(req)
	current := m.currentBackend
	return func() tea.Msg {
		var agents []state.ModelAgentInfo
		err := current.leaseModel(ctx, req.generation, func(b state.Backend) error {
			var err error
			agents, err = b.(state.ModelAgentLister).ListModelAgents(ctx)
			return err
		})
		return modelAgentsListMsg{request: req, agents: agents, err: err}
	}
}

func (m *Model) handleModelAgentsList(msg modelAgentsListMsg) {
	if !m.modelRequestActive(msg.request) {
		return
	}
	if msg.err != nil {
		m.invalidateModelSelection()
		m.noticeErr("/submodel: " + msg.request.backend + " agent listing failed: " + msg.err.Error() + " — try /submodel <agent> <native-ref>")
		return
	}
	agents := append([]state.ModelAgentInfo(nil), msg.agents...)
	sort.SliceStable(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })
	rows := make([]panels.ModelPickRow, 0, len(agents))
	seen := map[string]bool{}
	for _, agent := range agents {
		if agent.Name == "" || seen[agent.Name] {
			continue
		}
		seen[agent.Name] = true
		rows = append(rows, panels.ModelPickRow{ID: agent.Name, Ref: agent.Name, Name: agent.Name, Description: agent.Description})
	}
	m.modelPick.SetRows(rows)
}
