// charter_claude_mcp.go discovers the MCP servers available to a Claude Code
// session and renders the deliberately small prompt attachment that describes
// them. The attachment contains names, transport types, and scope only:
// Claude configuration routinely contains command paths, arguments, URLs, and
// environment values, none of which belongs in a model prompt.
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const claudeMCPAttachmentName = "mcp-servers.md"
const claudeMCPAttachmentImport = "@.claude/mcp-servers.md"

type claudeMCPServerHint struct {
	name   string
	typ    string
	source string
}

// claudeUserConfigPath honors Claude Code's config-directory override.
func claudeUserConfigPath() string {
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, ".claude.json")
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".claude.json")
	}
	return ""
}

// discoverClaudeMCPServers reads each configuration source independently so a
// missing, unreadable, or malformed one never prevents Claude from starting.
// Project declarations replace same-named user declarations. A disabled name
// in the matching project configuration is removed after all declarations.
func discoverClaudeMCPServers(dir string) []claudeMCPServerHint {
	byName := map[string]claudeMCPServerHint{}
	disabled := map[string]bool{}

	var user map[string]any
	if path := claudeUserConfigPath(); path != "" {
		user = readClaudeConfig(path)
	}
	if user != nil {
		addClaudeMCPServers(byName, user["mcpServers"], "user config")
		if project := matchingClaudeProject(user["projects"], dir); project != nil {
			addClaudeMCPServers(byName, project["mcpServers"], "project config")
			addDisabledClaudeMCPServers(disabled, project["disabledMcpServers"])
		}
		addDisabledClaudeMCPServers(disabled, user["disabledMcpServers"])
	}

	if project := readClaudeConfig(filepath.Join(dir, ".mcp.json")); project != nil {
		addClaudeMCPServers(byName, project["mcpServers"], "project .mcp.json")
		addDisabledClaudeMCPServers(disabled, project["disabledMcpServers"])
	}

	servers := make([]claudeMCPServerHint, 0, len(byName))
	for name, server := range byName {
		if !disabled[name] {
			servers = append(servers, server)
		}
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].name < servers[j].name })
	return servers
}

func readClaudeConfig(path string) map[string]any {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	return doc
}

func matchingClaudeProject(raw any, dir string) map[string]any {
	projects, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	clean := filepath.Clean(dir)
	for path, rawProject := range projects {
		if filepath.Clean(path) == clean {
			project, _ := rawProject.(map[string]any)
			return project
		}
	}
	return nil
}

func addClaudeMCPServers(byName map[string]claudeMCPServerHint, raw any, source string) {
	servers, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for name, rawServer := range servers {
		server, ok := rawServer.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := server["type"].(string)
		if typ == "" {
			continue
		}
		byName[name] = claudeMCPServerHint{name: name, typ: typ, source: source}
	}
}

func addDisabledClaudeMCPServers(disabled map[string]bool, raw any) {
	entries, ok := raw.([]any)
	if !ok {
		return
	}
	for _, rawName := range entries {
		if name, ok := rawName.(string); ok {
			disabled[name] = true
		}
	}
}

func renderClaudeMCPAttachment(servers []claudeMCPServerHint) []byte {
	var b strings.Builder
	b.WriteString("# Available MCP servers\n\n")
	b.WriteString("This Claude Code session has the MCP servers below configured.\n")
	b.WriteString("Use only the listed server names when selecting an MCP tool.\n\n")
	for _, server := range servers {
		fmt.Fprintf(&b, "- `%s` (%s) — configured in %s\n", server.name, server.typ, server.source)
	}
	b.WriteString("\nOnly names, types, and provenance are listed — never commands,\n")
	b.WriteString("arguments, URLs, headers, or environment values: this file lands in\n")
	b.WriteString("a prompt, and prompts are no place for API keys.\n")
	return []byte(b.String())
}

// ensureClaudeMCPAttachment writes the attachment byte-exactly, retiring it
// when no server remains. It returns whether the payload should import it.
func ensureClaudeMCPAttachment(dir string) (changed bool, hasAttachment bool, notes []string) {
	servers := discoverClaudeMCPServers(dir)
	path := filepath.Join(dir, ".claude", claudeMCPAttachmentName)
	if len(servers) == 0 {
		if err := os.Remove(path); err == nil {
			return true, false, []string{"[theboringfloor] claude mcp prompt attachment: removed (no MCP servers configured)"}
		} else if !os.IsNotExist(err) {
			return false, false, []string{"[theboringfloor] claude mcp prompt attachment: failed (remove " + path + ": " + err.Error() + ")"}
		}
		return false, false, []string{"[theboringfloor] claude mcp prompt attachment: no MCP servers configured"}
	}

	want := renderClaudeMCPAttachment(servers)
	if got, err := os.ReadFile(path); err == nil && bytes.Equal(got, want) {
		return false, true, []string{"[theboringfloor] claude mcp prompt attachment: already wired (.claude/mcp-servers.md)"}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, false, []string{"[theboringfloor] claude mcp prompt attachment: failed (mkdir " + filepath.Dir(path) + ": " + err.Error() + ")"}
	}
	if err := os.WriteFile(path, want, 0o644); err != nil {
		return false, false, []string{"[theboringfloor] claude mcp prompt attachment: failed (write " + path + ": " + err.Error() + ")"}
	}
	return true, true, []string{fmt.Sprintf("[theboringfloor] claude mcp prompt attachment: wired (.claude/mcp-servers.md — %d servers)", len(servers))}
}
