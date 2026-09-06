package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeAgentModelPreservesRichForeignFields(t *testing.T) {
	before := []byte(`{"$schema":"https://opencode.ai/config.json","instructions":["./rules.md"],"mcp":{},"plugin":["one"],"providers":{"azure":{}},"small_model":"azure/small","agent":{"explore":{"prompt":"Look carefully","tools":{"bash":false}},"general":{"model":"azure/general"}}}`)
	got, changed, err := mergeAgentModel(before, "explore", "azure/claude-sonnet-5")
	if err != nil || !changed {
		t.Fatalf("merge: changed=%v err=%v", changed, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["$schema"] != "https://opencode.ai/config.json" || doc["small_model"] != "azure/small" {
		t.Fatalf("foreign scalar fields changed: %v", doc)
	}
	if len(doc["instructions"].([]any)) != 1 || len(doc["plugin"].([]any)) != 1 || len(doc["mcp"].(map[string]any)) != 0 || len(doc["providers"].(map[string]any)) != 1 {
		t.Fatalf("foreign collection fields changed: %v", doc)
	}
	agents := doc["agent"].(map[string]any)
	explore := agents["explore"].(map[string]any)
	if explore["prompt"] != "Look carefully" || explore["tools"].(map[string]any)["bash"] != false || explore["model"] != "azure/claude-sonnet-5" {
		t.Fatalf("target agent fields changed: %v", explore)
	}
	if agents["general"].(map[string]any)["model"] != "azure/general" {
		t.Fatalf("other agent changed: %v", agents)
	}
	const want = "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"agent\": {\n    \"explore\": {\n      \"model\": \"azure/claude-sonnet-5\",\n      \"prompt\": \"Look carefully\",\n      \"tools\": {\n        \"bash\": false\n      }\n    },\n    \"general\": {\n      \"model\": \"azure/general\"\n    }\n  },\n  \"instructions\": [\n    \"./rules.md\"\n  ],\n  \"mcp\": {},\n  \"plugin\": [\n    \"one\"\n  ],\n  \"providers\": {\n    \"azure\": {}\n  },\n  \"small_model\": \"azure/small\"\n}\n"
	if string(got) != want {
		t.Fatalf("format differs from mergeInstruction convention:\nwant %s\ngot  %s", want, got)
	}
}

func TestMergeAgentModelIdempotent(t *testing.T) {
	first, changed, err := mergeAgentModel([]byte(`{"agent":{"explore":{"prompt":"x"}}}`), "explore", "azure/claude-sonnet-5")
	if err != nil || !changed {
		t.Fatalf("first run: changed=%v err=%v", changed, err)
	}
	second, changed, err := mergeAgentModel(first, "explore", "azure/claude-sonnet-5")
	if err != nil || changed || string(second) != string(first) {
		t.Fatalf("second run rewrote opencode.json: changed=%v err=%v got=%s", changed, err, second)
	}
}

func TestMergeAgentModelPreservesJSONNumbers(t *testing.T) {
	got, changed, err := mergeAgentModel([]byte(`{"revision":9007199254740993}`), "explore", "azure/claude-sonnet-5")
	if err != nil || !changed {
		t.Fatalf("merge: changed=%v err=%v", changed, err)
	}
	if !strings.Contains(string(got), `"revision": 9007199254740993`) {
		t.Fatalf("large JSON number changed: %s", got)
	}
}

func TestMergeAgentModelCreationPaths(t *testing.T) {
	cases := []struct{ name, body string }{
		{"empty", ""}, {"empty object", `{}`}, {"no agent", `{"theme":"nord"}`},
		{"other agent", `{"agent":{"general":{}}}`}, {"agent other fields", `{"agent":{"explore":{"prompt":"x"}}}`},
		{"same model", `{"agent":{"explore":{"model":"azure/claude-sonnet-5"}}}`}, {"different model", `{"agent":{"explore":{"model":"azure/old"}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := mergeAgentModel([]byte(tc.body), "explore", "azure/claude-sonnet-5")
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "same model" {
				if changed || string(got) != tc.body {
					t.Fatalf("same model must be a byte no-op: changed=%v got=%s", changed, got)
				}
				return
			}
			if !changed {
				t.Fatal("changed=false")
			}
			var doc map[string]any
			if err := json.Unmarshal(got, &doc); err != nil {
				t.Fatal(err)
			}
			if doc["agent"].(map[string]any)["explore"].(map[string]any)["model"] != "azure/claude-sonnet-5" {
				t.Fatalf("model missing: %s", got)
			}
		})
	}
}

func TestMergeAgentModelFailsClosed(t *testing.T) {
	cases := []struct {
		body, want string
	}{
		{`[]`, "cannot unmarshal array"},
		{`{"agent":null}`, "agent is null — refusing to rewrite a hand-shaped config"},
		{`{"agent":"bad"}`, "agent is string, not an object — refusing to rewrite a hand-shaped config"},
		{`{"agent":{"explore":null}}`, "agent.explore is null — refusing to rewrite a hand-shaped config"},
		{`{"agent":{"explore":[]}}`, "agent.explore is []interface {}, not an object — refusing to rewrite a hand-shaped config"},
		{`{"agent":{"explore":{"model":false}}}`, "agent.explore.model is bool, not a string — refusing to rewrite a hand-shaped config"},
		{`{"agent":`, "unparseable json:"},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			got, changed, err := mergeAgentModel([]byte(tc.body), "explore", "azure/claude-sonnet-5")
			if err == nil || changed || got != nil {
				t.Fatalf("must fail closed: changed=%v err=%v got=%q", changed, err, got)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestEnsureAgentModelsWritesProjectConfigAndContinuesAfterInvalidEntry(t *testing.T) {
	dir := t.TempDir()
	changed, err := ensureAgentModels(dir, map[string]string{"bad name": "azure/nope", "explore": "azure/claude-sonnet-5"})
	if !changed || err == nil || !strings.Contains(err.Error(), `invalid agent name "bad name"`) {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	cfgPath := filepath.Join(dir, ".opencode", "opencode.json")
	raw, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["$schema"] != opencodeSchemaURL || doc["agent"].(map[string]any)["explore"].(map[string]any)["model"] != "azure/claude-sonnet-5" {
		t.Fatalf("config: %s", raw)
	}
	before := string(raw)
	changed, err = ensureAgentModels(dir, map[string]string{"explore": "azure/claude-sonnet-5"})
	if changed || err != nil || string(mustRead(t, cfgPath)) != before {
		t.Fatalf("second run rewrote opencode.json: changed=%v err=%v", changed, err)
	}
}

func TestEnsureAgentModelsPreservesSymlink(t *testing.T) {
	dir, targetDir := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "target.json")
	if err := os.WriteFile(target, []byte(`{"theme":"nord"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".opencode", "opencode.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if changed, err := ensureAgentModels(dir, map[string]string{"explore": "azure/claude-sonnet-5"}); !changed || err != nil {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink replaced: %v %v", info, err)
	}
	if !strings.Contains(mustRead(t, target), "azure/claude-sonnet-5") {
		t.Fatal("symlink target was not updated")
	}
}

func TestLiveBackendApplyAgentModelsWritesAgentHarnessConfig(t *testing.T) {
	dir := t.TempDir()
	b := newLiveBackend("", dir, nil)
	if err := b.ApplyAgentModels(map[string]string{"explore": "azure/claude-sonnet-5"}); err != nil {
		t.Fatal(err)
	}
	raw := mustRead(t, filepath.Join(dir, ".opencode", "opencode.json"))
	const want = "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"agent\": {\n    \"explore\": {\n      \"model\": \"azure/claude-sonnet-5\"\n    }\n  }\n}\n"
	if raw != want {
		t.Fatalf("agent harness config = %s, want %s", raw, want)
	}
}
