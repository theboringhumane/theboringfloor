package config

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestModelPreferencesPrecedenceAndIsolation(t *testing.T) {
	cfg := Default()
	cfg.Backend.Name = BackendNameClaude
	cfg.Boss.Model = " legacy/boss "
	cfg.AgentModels["explore"] = "legacy/explore"
	if got := cfg.EffectiveModel("", ""); got != " legacy/boss " {
		t.Fatalf("empty backend should resolve OpenCode legacy boss verbatim: %q", got)
	}
	cfg.Backend.BossModel = " \t\n "
	if got := cfg.EffectiveModel("", ""); got != " legacy/boss " {
		t.Fatalf("whitespace-only backend boss should fall back verbatim: %q", got)
	}
	cfg.Backend.BossModel = " legacy/backend \t"
	if got := cfg.EffectiveModel(BackendNameDefault, ""); got != "legacy/backend" {
		t.Fatalf("trimmed backend boss should precede boss.model: %q", got)
	}
	for _, backend := range []string{BackendNameClaude, BackendNameCodex, "future"} {
		if cfg.EffectiveModel(backend, "") != "" || cfg.EffectiveModel(backend, "explore") != "" || len(cfg.AgentModelPreferences(backend)) != 0 {
			t.Errorf("legacy OpenCode refs leaked into %s", backend)
		}
	}
	cfg.SetModelPreference("", "", " scoped/boss ")
	cfg.SetModelPreference(BackendNameClaude, "", "opus[1m]")
	cfg.SetModelPreference(BackendNameCodex, "", "gpt-5.4")
	cfg.SetModelPreference(BackendNameClaude, "explore", "sonnet")
	for _, tc := range []struct{ backend, agent, want string }{
		{"", "", " scoped/boss "}, {BackendNameClaude, "", "opus[1m]"},
		{BackendNameCodex, "", "gpt-5.4"}, {BackendNameClaude, "explore", "sonnet"},
		{BackendNameDefault, "explore", "legacy/explore"}, {BackendNameCodex, "explore", ""},
	} {
		if got := cfg.EffectiveModel(tc.backend, tc.agent); got != tc.want {
			t.Errorf("EffectiveModel(%q, %q) = %q, want %q", tc.backend, tc.agent, got, tc.want)
		}
	}
	if cfg.Boss.Model != " legacy/boss " || cfg.Backend.BossModel != " legacy/backend \t" || cfg.AgentModels["explore"] != "legacy/explore" {
		t.Fatal("scoped writes changed legacy preferences")
	}
}

func TestModelPreferencesEmptyOverridesAndSnapshot(t *testing.T) {
	cfg := Default()
	cfg.Backend.BossModel = "legacy/boss"
	cfg.AgentModels = map[string]ModelRef{"explore": "legacy/explore", "developer": "legacy/developer"}
	cfg.ModelPreferences = map[string]BackendModelPreferences{BackendNameDefault: {}}
	if cfg.EffectiveModel("", "") != "legacy/boss" || cfg.EffectiveModel("", "explore") != "legacy/explore" {
		t.Fatal("absent scoped preferences must fall back")
	}
	cfg.SetModelPreference("", "", "")
	cfg.SetModelPreference("", "explore", "")
	if cfg.EffectiveModel("", "") != "" || cfg.EffectiveModel("", "explore") != "" {
		t.Fatal("explicit empty preferences must mask legacy values")
	}
	refs := cfg.AgentModelPreferences("")
	if !reflect.DeepEqual(refs, map[string]string{"explore": "", "developer": "legacy/developer"}) {
		t.Fatalf("snapshot lost inheritance tombstone or fallback: %#v", refs)
	}
	refs["explore"], refs["developer"] = "mutated", "mutated"
	delete(refs, "developer")
	if cfg.EffectiveModel("", "explore") != "" || cfg.EffectiveModel("", "developer") != "legacy/developer" {
		t.Fatal("snapshot aliases config maps")
	}
	cfg.SetModelPreference("", "explore", "new/explore")
	if refs["explore"] != "mutated" {
		t.Fatal("config mutation changed an existing snapshot")
	}
	var nilConfig *Config
	if nilConfig.EffectiveModel("", "") != "" || len(nilConfig.AgentModelPreferences("")) != 0 {
		t.Fatal("nil config reads must return empty preferences")
	}
}

func TestModelPreferencesLoadSave(t *testing.T) {
	home := useHome(t)
	p := writeBrain(t, home, `{
		"boss":{"model":" legacy/boss "},
		"backend":{"bossModel":"opaque legacy"},
		"agentModels":{"explore":"legacy/explore"},
		"modelPreferences":{
			"opencode":{"boss":"","agents":{"explore":""},"unknownPreference":true},
			"claudecode":{"boss":"opus[1m]","agents":{"research":"sonnet"}},
			"codex":{"agents":{"worker":"namespace/model@preview"}}
		},
		"unknownTopLevel":true
	}`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelPreferences[BackendNameCodex].Boss != nil || cfg.ModelPreferences[BackendNameDefault].Boss == nil {
		t.Fatal("load lost absent versus empty boss distinction")
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("load/save changed preferences: got %#v, want %#v", got, cfg)
	}
	if got.EffectiveModel("", "") != "" || got.EffectiveModel("", "explore") != "" || got.EffectiveModel(BackendNameCodex, "worker") != "namespace/model@preview" {
		t.Fatal("roundtrip changed native refs or empty override behavior")
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "unknown") {
		t.Fatal("unknown keys should retain existing drop-on-save behavior")
	}
	defaults, err := json.Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(defaults), "modelPreferences") {
		t.Fatal("defaults should not eagerly add scoped preferences")
	}
}
