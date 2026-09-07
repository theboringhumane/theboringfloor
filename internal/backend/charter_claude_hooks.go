// charter_claude_hooks.go wires the Claude Code SubagentStop hook into the
// office's completed-dispatch ledger path. The hook emits a stable marker on
// stdout; the Claude stream reader owns turning that completion signal and its
// matching Task result into the ledger entry.
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const claudeProjectSettingsName = "settings.json"
const claudeLedgerHookEvent = "SubagentStop"
const claudeLedgerHookMarker = "theboringfloor-office-ledger-subagent-stop"
const claudeLedgerHookCommand = "printf '%s\\n' '{\"systemMessage\":\"[theboringfloor] office ledger: sub-agent completed\"}' # " + claudeLedgerHookMarker

// claudeProjectSettingsPath is deliberately project-scoped. The office never
// alters ~/.claude/settings.json: a project may safely travel with its ledger
// completion hook without changing the member's global Claude configuration.
func claudeProjectSettingsPath(dir string) string {
	return filepath.Join(dir, ".claude", claudeProjectSettingsName)
}

func claudePathWithinProject(dir, path string) bool {
	project, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(project, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func claudeLedgerHook() map[string]any {
	return map[string]any{"type": "command", "command": claudeLedgerHookCommand}
}

// ensureClaudeLedgerHook merges the office-owned SubagentStop command into a
// project's .claude/settings.json. It refuses an unparseable or hand-shaped
// hooks value rather than risking a member's configuration. The marker lives
// in the command string because Claude hook entries have no stable ID field.
func ensureClaudeLedgerHook(dir string) (changed bool, notes []string) {
	path := claudeProjectSettingsPath(dir)
	if !claudePathWithinProject(dir, path) {
		return false, []string{"[theboringfloor] claude ledger hook: failed (settings path escaped project: " + path + ")"}
	}

	doc := map[string]any{}
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(raw, &doc); err != nil {
			return false, []string{"[theboringfloor] claude ledger hook: failed (parse " + path + ": " + err.Error() + ")"}
		}
		if doc == nil {
			return false, []string{"[theboringfloor] claude ledger hook: failed (settings is not an object — refusing to rewrite hand-shaped settings)"}
		}
	case os.IsNotExist(err):
		// Start with the smallest valid project settings document.
	default:
		return false, []string{"[theboringfloor] claude ledger hook: failed (read " + path + ": " + err.Error() + ")"}
	}

	hooks, hasHooks := doc["hooks"]
	if !hasHooks {
		hooks = map[string]any{}
		doc["hooks"] = hooks
		changed = true
	}
	hookEvents, ok := hooks.(map[string]any)
	if !ok {
		return false, []string{"[theboringfloor] claude ledger hook: failed (hooks is not an object — refusing to rewrite hand-shaped settings)"}
	}

	rawGroups, hasEvent := hookEvents[claudeLedgerHookEvent]
	if !hasEvent {
		rawGroups = []any{}
	}
	groups, ok := rawGroups.([]any)
	if !ok {
		return false, []string{"[theboringfloor] claude ledger hook: failed (hooks." + claudeLedgerHookEvent + " is not an array — refusing to rewrite hand-shaped settings)"}
	}

	found := false
	for _, rawGroup := range groups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}
		rawEntries, ok := group["hooks"]
		if !ok {
			continue
		}
		entries, ok := rawEntries.([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(entries))
		entriesChanged := false
		for _, rawEntry := range entries {
			entry, ok := rawEntry.(map[string]any)
			command, isCommand := entry["command"].(string)
			if !ok || !isCommand || !strings.Contains(command, claudeLedgerHookMarker) {
				filtered = append(filtered, rawEntry)
				continue
			}
			if found {
				changed = true // Remove duplicate office-owned entries only.
				entriesChanged = true
				continue
			}
			found = true
			want := claudeLedgerHook()
			if !claudeJSONEqual(entry, want) {
				filtered = append(filtered, want)
				changed = true
				entriesChanged = true
				continue
			}
			filtered = append(filtered, rawEntry)
		}
		if entriesChanged {
			group["hooks"] = filtered
		}
	}

	if !found {
		groups = append(groups, map[string]any{"hooks": []any{claudeLedgerHook()}})
		hookEvents[claudeLedgerHookEvent] = groups
		changed = true
	}
	if !hasEvent && !changed {
		// Unreachable today, retained to make the ownership invariant obvious.
		hookEvents[claudeLedgerHookEvent] = groups
		changed = true
	}
	if !changed {
		return false, []string{"[theboringfloor] claude ledger hook: already wired (.claude/settings.json)"}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, []string{"[theboringfloor] claude ledger hook: failed (marshal " + path + ": " + err.Error() + ")"}
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, []string{"[theboringfloor] claude ledger hook: failed (mkdir " + filepath.Dir(path) + ": " + err.Error() + ")"}
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return false, []string{"[theboringfloor] claude ledger hook: failed (write " + path + ": " + err.Error() + ")"}
	}
	return true, []string{fmt.Sprintf("[theboringfloor] claude ledger hook: wired (.claude/settings.json, %s)", claudeLedgerHookEvent)}
}

func claudeJSONEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	return err == nil && bytes.Equal(left, right)
}
