package backend

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureClaudeLedgerHookMergeSafe(t *testing.T) {
	for _, tc := range []struct {
		name     string
		original string
	}{
		{
			name: "unrelated settings and hooks survive",
			original: `{
  "env": {"KEEP": "unchanged"},
  "permissions": {"allow": ["Read"]},
  "hooks": {
    "Stop": [{"hooks": [{"type": "command", "command": "printf stop"}]}],
    "SubagentStop": [{"matcher": "Explore", "hooks": [{"type": "command", "command": "printf member-hook"}]}]
  }

	}
	`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := claudeProjectSettingsPath(dir)
			writeClaudeConfig(t, path, tc.original)

			changed, notes := ensureClaudeLedgerHook(dir)
			if !changed || !containsNote(notes, "wired") {
				t.Fatalf("merge = changed=%v notes=%v", changed, notes)
			}
			var got map[string]any
			raw, err := os.ReadFile(path)
			if err != nil || json.Unmarshal(raw, &got) != nil {
				t.Fatalf("read merged settings: %v; %s", err, raw)
			}
			if got["env"].(map[string]any)["KEEP"] != "unchanged" || got["permissions"].(map[string]any)["allow"].([]any)[0] != "Read" {
				t.Fatalf("unrelated settings changed: %#v", got)
			}
			hooks := got["hooks"].(map[string]any)
			if hooks["Stop"].([]any)[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"] != "printf stop" {
				t.Fatalf("unrelated Stop hook changed: %#v", hooks["Stop"])
			}
			if hooks["SubagentStop"].([]any)[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"] != "printf member-hook" {
				t.Fatalf("unrelated SubagentStop hook changed: %#v", hooks["SubagentStop"])
			}
			if n := strings.Count(string(raw), claudeLedgerHookMarker); n != 1 {
				t.Fatalf("office hook marker count = %d, want 1:\n%s", n, raw)
			}
		})
	}
}

func TestEnsureClaudeLedgerHookIdempotentAndUpdatesOwnMarker(t *testing.T) {
	dir := t.TempDir()
	path := claudeProjectSettingsPath(dir)
	if changed, notes := ensureClaudeLedgerHook(dir); !changed || !containsNote(notes, "wired") {
		t.Fatalf("first write = changed=%v notes=%v", changed, notes)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed, notes := ensureClaudeLedgerHook(dir); changed || !containsNote(notes, "already wired") {
		t.Fatalf("second write = changed=%v notes=%v", changed, notes)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("second write moved bytes:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}

	stale := `{"hooks":{"SubagentStop":[{"hooks":[{"type":"command","command":"printf stale # ` + claudeLedgerHookMarker + `"}]}]}}`
	writeClaudeConfig(t, path, stale)
	if changed, notes := ensureClaudeLedgerHook(dir); !changed || !containsNote(notes, "wired") {
		t.Fatalf("stale office hook = changed=%v notes=%v", changed, notes)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(updated, &doc); err != nil {
		t.Fatal(err)
	}
	entry := doc["hooks"].(map[string]any)[claudeLedgerHookEvent].([]any)[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)
	if n := strings.Count(string(updated), claudeLedgerHookMarker); n != 1 || entry["command"] != claudeLedgerHookCommand {
		t.Fatalf("stale office hook was not updated in place:\n%s", updated)
	}
}

func TestClaudeLedgerHookSettingsPathStaysInProject(t *testing.T) {
	dir := t.TempDir()
	path := claudeProjectSettingsPath(dir)
	if !claudePathWithinProject(dir, path) {
		t.Fatalf("project settings path escaped: project=%q path=%q", dir, path)
	}
	if claudePathWithinProject(dir, filepath.Join(filepath.Dir(dir), "settings.json")) {
		t.Fatal("sibling path incorrectly considered inside project")
	}
}
