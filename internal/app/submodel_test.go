package app

import (
	"reflect"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

const submodelUsage = "/submodel: usage /submodel <agent> provider/model (agents: explore, developer, general)"

// agentModelsBackend records the optional persistence seam without reaching a
// live opencode server.
type agentModelsBackend struct {
	recBackend
	calls []map[string]string
}

func (b *agentModelsBackend) ApplyAgentModels(models map[string]string) error {
	copy := make(map[string]string, len(models))
	for name, model := range models {
		copy[name] = model
	}
	b.calls = append(b.calls, copy)
	return nil
}

func applySubmodel(t *testing.T, m *Model, fields ...string) {
	t.Helper()
	if cmd := m.applySubmodel(fields); cmd != nil {
		_ = cmd()
	}
}

func TestSubmodelUsageErrors(t *testing.T) {
	for _, fields := range [][]string{{"/submodel"}, {"/submodel", "explore"}} {
		t.Run("fields", func(t *testing.T) {
			scratchHome(t)
			m := New(&recBackend{}, nil)
			applySubmodel(t, &m, fields...)
			last := lastChat(t, m)
			if last.Meta != "error" || last.Text != submodelUsage {
				t.Fatalf("usage notice = (%q, %q), want error %q", last.Meta, last.Text, submodelUsage)
			}
		})
	}
}

func TestSubmodelRejectsInvalidAgentWithoutMutation(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, config.Default())
	applySubmodel(t, &m, "/submodel", "scout!", "azure/claude-sonnet-5")
	last := lastChat(t, m)
	want := "/submodel: unknown agent scout! — try explore, developer, or general"
	if last.Meta != "error" || last.Text != want {
		t.Fatalf("invalid-agent notice = (%q, %q), want error %q", last.Meta, last.Text, want)
	}
	if len(m.cfg.AgentModels) != 0 {
		t.Fatalf("invalid agent must not mutate AgentModels: %#v", m.cfg.AgentModels)
	}
}

func TestSubmodelRejectsInvalidRefWithoutMutation(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, config.Default())
	applySubmodel(t, &m, "/submodel", "explore", "claude-sonnet-5")
	last := lastChat(t, m)
	want := "/submodel: model must look like provider/model (e.g. azure/claude-sonnet-5)"
	if last.Meta != "error" || last.Text != want {
		t.Fatalf("invalid-ref notice = (%q, %q), want error %q", last.Meta, last.Text, want)
	}
	if len(m.cfg.AgentModels) != 0 {
		t.Fatalf("invalid ref must not mutate AgentModels: %#v", m.cfg.AgentModels)
	}
}

func TestSubmodelStoresAndAppliesFullMap(t *testing.T) {
	scratchHome(t)
	b := &agentModelsBackend{}
	cfg := config.Default()
	cfg.AgentModels["developer"] = "anthropic/claude-haiku-4-5"
	m := New(b, cfg)
	applySubmodel(t, &m, "/submodel", "explore", "azure/claude-sonnet-5")

	if got := m.cfg.AgentModels["explore"]; got != "azure/claude-sonnet-5" {
		t.Fatalf("AgentModels[explore] = %q", got)
	}
	last := lastChat(t, m)
	want := "sub-agent model → explore = azure/claude-sonnet-5 · saved to brain.json"
	if last.Meta == "error" || last.Text != want {
		t.Fatalf("success notice = (%q, %q), want notice %q", last.Meta, last.Text, want)
	}
	if wantMap := map[string]string{
		"developer": "anthropic/claude-haiku-4-5",
		"explore":   "azure/claude-sonnet-5",
	}; !reflect.DeepEqual(b.calls, []map[string]string{wantMap}) {
		t.Fatalf("ApplyAgentModels calls = %#v, want %#v", b.calls, []map[string]string{wantMap})
	}
}

func TestSubmodelFallbackAndPreservesOtherAgents(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, config.Default())
	applySubmodel(t, &m, "/submodel", "explore", "azure/claude-sonnet-5")
	if last := lastChat(t, m); last.Text != "/submodel: recorded, but this backend cannot apply sub-agent models (opencode only)" {
		t.Fatalf("fallback notice = %q", last.Text)
	}
	applySubmodel(t, &m, "/submodel", "developer", "anthropic/claude-haiku-4-5")
	want := map[string]config.ModelRef{
		"explore":   "azure/claude-sonnet-5",
		"developer": "anthropic/claude-haiku-4-5",
	}
	if !reflect.DeepEqual(m.cfg.AgentModels, want) {
		t.Fatalf("second override must preserve the first: got %#v, want %#v", m.cfg.AgentModels, want)
	}
}
