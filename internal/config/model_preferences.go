package config

import "strings"

// BackendModelPreferences preserves unset versus explicitly empty selections.
// A nil Boss is unset; an empty Boss clears the office override. Default and
// resumed-session behavior belong to the backend; this does not promise a reset.
// An empty agent reference explicitly inherits, masking legacy preferences.
type BackendModelPreferences struct {
	Boss   *ModelRef           `json:"boss,omitempty"`
	Agents map[string]ModelRef `json:"agents,omitempty"`
}

// EffectiveModel resolves a selection for a backend; an empty agent means boss.
// Only OpenCode consults the legacy, unscoped preferences.
func (c *Config) EffectiveModel(backendName, agent string) string {
	if c == nil {
		return ""
	}
	backendName = (BackendConfig{Name: backendName}).ResolvedName()
	prefs := c.ModelPreferences[backendName]
	if agent == "" {
		if prefs.Boss != nil {
			return string(*prefs.Boss)
		}
		if backendName == BackendNameDefault {
			if ref := strings.TrimSpace(c.Backend.BossModel); ref != "" {
				return ref
			}
			return string(c.Boss.Model)
		}
		return ""
	}
	if ref, ok := prefs.Agents[agent]; ok {
		return string(ref)
	}
	if backendName == BackendNameDefault {
		return string(c.AgentModels[agent])
	}
	return ""
}

// AgentModelPreferences returns an independent snapshot, including empty
// inheritance tombstones. Backends must synchronize their own snapshots.
func (c *Config) AgentModelPreferences(backendName string) map[string]string {
	refs := make(map[string]string)
	if c == nil {
		return refs
	}
	backendName = (BackendConfig{Name: backendName}).ResolvedName()
	if backendName == BackendNameDefault {
		for agent, ref := range c.AgentModels {
			refs[agent] = string(ref)
		}
	}
	for agent, ref := range c.ModelPreferences[backendName].Agents {
		refs[agent] = string(ref)
	}
	return refs
}

// SetModelPreference updates only the scoped preference. Validation belongs
// to the backend adapter; callers in the app own config writes.
func (c *Config) SetModelPreference(backendName, agent, ref string) {
	if c == nil {
		return
	}
	backendName = (BackendConfig{Name: backendName}).ResolvedName()
	if c.ModelPreferences == nil {
		c.ModelPreferences = make(map[string]BackendModelPreferences)
	}
	prefs := c.ModelPreferences[backendName]
	if agent == "" {
		model := ModelRef(ref)
		prefs.Boss = &model
	} else {
		if prefs.Agents == nil {
			prefs.Agents = make(map[string]ModelRef)
		}
		prefs.Agents[agent] = ModelRef(ref)
	}
	c.ModelPreferences[backendName] = prefs
}
