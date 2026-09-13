package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

type nativeModelsBackend struct {
	modelsBackend
	agents   []state.ModelAgentInfo
	agentErr error
	targets  []state.ModelTarget
}

func (b *nativeModelsBackend) ListModelAgents(context.Context) ([]state.ModelAgentInfo, error) {
	return b.agents, b.agentErr
}
func (b *nativeModelsBackend) ListModelsForTarget(ctx context.Context, target state.ModelTarget) ([]state.ModelInfo, error) {
	b.targets = append(b.targets, target)
	return b.ListModels(ctx)
}

func TestSubmodelSlashNativeTypesAndTargetCatalog(t *testing.T) {
	scratchHome(t)
	b := &nativeModelsBackend{modelsBackend: modelsBackend{models: []state.ModelInfo{{ID: "sonnet", Ref: "sonnet", Name: "Sonnet"}}}, agents: []state.ModelAgentInfo{{Name: "Explore", Description: "Native read-only search"}, {Name: "general-purpose"}}}
	cfg := config.Default()
	cfg.Backend.Name = "claudecode"
	m := sized(t, New(b, cfg))
	m = runMsg(t, m, slashMsg{text: "/submodel"})
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{"SUB-AGENT TYPE · claudecode", "Explore", "general-purpose"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("missing %q in frame:\n%s", want, frame)
		}
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	frame = ansi.Strip(m.Frame())
	if !strings.Contains(frame, "SUB-AGENT MODEL · Explore · claudecode") {
		t.Fatalf("target frame:\n%s", frame)
	}
	if len(b.targets) != 1 || b.targets[0].Agent != "Explore" {
		t.Fatalf("native target lost: %+v", b.targets)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(b.setCalls) != 1 || b.setCalls[0] != (modelSetCall{state.ModelTarget{Agent: "Explore"}, "sonnet"}) {
		t.Fatalf("setter calls: %+v", b.setCalls)
	}
	if got := readBrain(t).EffectiveModel("claudecode", "Explore"); got != "sonnet" {
		t.Fatalf("saved selection: %q", got)
	}
	if len(cfg.AgentModels) != 0 || cfg.Boss.Model != "" {
		t.Fatalf("legacy fields changed: %+v", cfg)
	}
	t.Logf("rendered: SUB-AGENT TYPE · claudecode; SUB-AGENT MODEL · Explore · claudecode; setter target=%q ref=%q", b.setCalls[0].target.Agent, b.setCalls[0].ref)
}

func TestSubmodelDirectSlashPreservesNativeTokensAndScopes(t *testing.T) {
	scratchHome(t)
	b := &nativeModelsBackend{}
	cfg := config.Default()
	cfg.Backend.Name = "codex"
	cfg.AgentModels["worker"] = "legacy/worker"
	cfg.Boss.Model = "legacy/boss"
	m := New(b, cfg)
	m = runMsg(t, m, slashMsg{text: "/submodel worker org/model@preview"})
	m = runMsg(t, m, slashMsg{text: "/submodel reviewer gpt-5.4"})
	if len(b.setCalls) != 2 || b.setCalls[0].target.Agent != "worker" || b.setCalls[0].ref != "org/model@preview" {
		t.Fatalf("setter calls: %+v", b.setCalls)
	}
	saved := readBrain(t)
	if saved.EffectiveModel("codex", "worker") != "org/model@preview" || saved.EffectiveModel("codex", "reviewer") != "gpt-5.4" || saved.EffectiveModel("codex", "") != "" || saved.AgentModels["worker"] != "legacy/worker" || saved.Boss.Model != "legacy/boss" {
		t.Fatalf("scopes changed: %+v", saved)
	}
	m = runMsg(t, m, slashMsg{text: "/submodel worker"})
	if len(b.targets) != 1 || b.targets[0].Agent != "worker" {
		t.Fatalf("explicit target picker: %+v", b.targets)
	}
}

func TestSubmodelErrorsDoNotPersist(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		b             state.Backend
		want          string
	}{
		{"unsupported", "/submodel worker gpt-5.4", &recBackend{}, "does not support model changes"},
		{"rejected", "/submodel Unknown sonnet", &modelsBackend{setErr: errors.New("unknown native agent Unknown")}, "unknown native agent Unknown"},
		{"agents", "/submodel", &nativeModelsBackend{agentErr: errors.New("catalog offline")}, "agent listing failed: catalog offline"},
		{"usage", "/submodel worker gpt-5.4 extra", &recBackend{}, "usage /submodel [native-agent] [native-ref]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scratchHome(t)
			m := runMsg(t, New(tc.b, nil), slashMsg{text: tc.command})
			last := lastChat(t, m)
			if last.Meta != "error" || !strings.Contains(last.Text, tc.want) {
				t.Fatalf("notice: %+v", last)
			}
			if len(m.cfg.ModelPreferences) != 0 || len(m.cfg.AgentModels) != 0 {
				t.Fatalf("failure persisted config: %+v", m.cfg)
			}
		})
	}
}

func TestSubmodelUnsupportedListingKeepsManualSelection(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{}
	m := runMsg(t, New(b, nil), slashMsg{text: "/submodel"})
	if m.ModelPickerOpen() || !strings.Contains(lastChat(t, m).Text, "/submodel <agent> <native-ref>") {
		t.Fatal("missing manual fallback")
	}
	m = runMsg(t, m, slashMsg{text: "/submodel custom/native exact[1m]"})
	if len(b.setCalls) != 1 || b.setCalls[0].target.Agent != "custom/native" || b.setCalls[0].ref != "exact[1m]" {
		t.Fatalf("native selection: %+v", b.setCalls)
	}
}
