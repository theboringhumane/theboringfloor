package state

import "context"

// SelectionRef returns the exact token the backend accepts for selection.
func (m ModelInfo) SelectionRef() string {
	if m.Ref != "" {
		return m.Ref
	}
	if m.Provider != "" {
		return m.Provider + "/" + m.ID
	}
	return m.ID
}

// ModelTarget selects the boss when Agent is empty, or a named native agent.
type ModelTarget struct {
	Agent string `json:"agent,omitempty"`
}

type ModelAgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ModelLister, ModelAgentLister, and ModelSetter are optional capabilities;
// supporting model selection does not change the required Backend interface.
type ModelLister interface {
	ListModels(context.Context) ([]ModelInfo, error)
}

type ModelAgentLister interface {
	ListModelAgents(context.Context) ([]ModelAgentInfo, error)
}

type ModelSetter interface {
	SetModel(context.Context, ModelTarget, string) error
}
