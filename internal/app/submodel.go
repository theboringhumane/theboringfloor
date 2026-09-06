// submodel.go — /submodel records opencode child-agent model overrides.
package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

// agentModelBackend is the optional seam for backends that can persist
// per-sub-agent model overrides into their own config.
type agentModelBackend interface {
	ApplyAgentModels(models map[string]string) error
}

// applySubmodel handles the /submodel slash command. fields is the
// whitespace-split command line, with fields[0] == "/submodel".
func (m *Model) applySubmodel(fields []string) tea.Cmd {
	const usage = "/submodel: usage /submodel <agent> provider/model (agents: explore, developer, general)"
	if len(fields) != 3 {
		m.noticeErr(usage)
		return nil
	}

	agent, ref := fields[1], fields[2]
	if !config.ValidAgentName(agent) {
		m.noticeErr(fmt.Sprintf("/submodel: unknown agent %s — try explore, developer, or general", agent))
		return nil
	}
	if !config.ValidModelRef(ref) {
		m.noticeErr("/submodel: model must look like provider/model (e.g. azure/claude-sonnet-5)")
		return nil
	}

	if m.cfg.AgentModels == nil {
		m.cfg.AgentModels = map[string]config.ModelRef{}
	}
	m.cfg.AgentModels[agent] = config.ModelRef(ref)
	m.notice(fmt.Sprintf("sub-agent model → %s = %s · %s", agent, ref, m.persistCfg()))

	ab, ok := m.backend.(agentModelBackend)
	if !ok {
		m.notice("/submodel: recorded, but this backend cannot apply sub-agent models (opencode only)")
		return nil
	}
	models := make(map[string]string, len(m.cfg.AgentModels))
	for name, model := range m.cfg.AgentModels {
		models[name] = string(model)
	}
	return func() tea.Msg {
		if err := ab.ApplyAgentModels(models); err != nil {
			return chatNoticeMsg{text: "/submodel: " + err.Error()}
		}
		return nil
	}
}
