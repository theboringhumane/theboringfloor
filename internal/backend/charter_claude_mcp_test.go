package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeClaudeConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverClaudeMCPServersUserScopeOnly(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{"mcpServers":{"thefloor_mcp":{"type":"stdio","command":"/private/thefloor_mcp"}}}`)

	servers := discoverClaudeMCPServers(t.TempDir())
	if len(servers) != 1 || servers[0].name != "thefloor_mcp" || servers[0].typ != "stdio" || servers[0].source != "user config" {
		t.Fatalf("user scope discovery = %+v, want thefloor_mcp from user config", servers)
	}
}

func TestDiscoverClaudeMCPServersProjectMergePrecedenceAndDisabled(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{
  "mcpServers": {
    "shared": {"type":"stdio"},
    "user-only": {"type":"sse"},
    "disabled": {"type":"stdio"}
  },
  "projects": {
    "`+dir+`": {
      "mcpServers": {"shared": {"type":"http"}},
      "disabledMcpServers": ["disabled"]
    }
  }
}`)
	writeClaudeConfig(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"project-only":{"type":"http"}}}`)

	servers := discoverClaudeMCPServers(dir)
	got := make([]string, 0, len(servers))
	byName := map[string]claudeMCPServerHint{}
	for _, server := range servers {
		got = append(got, server.name)
		byName[server.name] = server
	}
	if strings.Join(got, ",") != "project-only,shared,user-only" {
		t.Fatalf("merged servers = %v", got)
	}
	if shared := byName["shared"]; shared.typ != "http" || shared.source != "project config" {
		t.Fatalf("matching projects entry must override user scope: %+v", shared)
	}
	if project := byName["project-only"]; project.source != "project .mcp.json" {
		t.Fatalf(".mcp.json provenance = %+v", project)
	}
}

func TestEnsureClaudeMCPAttachmentMalformedConfigDegradesAndRetires(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{broken`)
	path := filepath.Join(dir, ".claude", claudeMCPAttachmentName)
	writeClaudeConfig(t, path, "stale\n")

	changed, hasAttachment, notes := ensureClaudeMCPAttachment(dir)
	if !changed || hasAttachment || !containsNote(notes, "removed") {
		t.Fatalf("malformed config must retire cleanly: changed=%v has=%v notes=%v", changed, hasAttachment, notes)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stale attachment remains: %v", err)
	}
}

func TestEnsureClaudeMCPAttachmentIdempotentAndPrivate(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{
  "mcpServers": {
    "thefloor_mcp": {
      "type": "stdio",
      "command": "/Users/member/.local/bin/thefloor_mcp",
      "args": ["--private-argv"],
      "env": {"PRIVATE_ENV": "private-value"}
    },
    "disabled": {"type":"sse", "url":"https://private.example/mcp"}
  },
  "projects": {"`+dir+`": {"disabledMcpServers":["disabled"]}}
}`)
	writeClaudeConfig(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"project-http":{"type":"http","url":"https://private.example/http"}}}`)

	changed, hasAttachment, notes := ensureClaudeMCPAttachment(dir)
	if !changed || !hasAttachment || !containsNote(notes, "wired") {
		t.Fatalf("first attachment write = changed=%v has=%v notes=%v", changed, hasAttachment, notes)
	}
	attachment, err := os.ReadFile(filepath.Join(dir, ".claude", claudeMCPAttachmentName))
	if err != nil {
		t.Fatal(err)
	}
	out := string(attachment)
	for _, want := range []string{
		"This Claude Code session has the MCP servers below configured.",
		"- `thefloor_mcp` (stdio) — configured in user config",
		"- `project-http` (http) — configured in project .mcp.json",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("attachment missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"disabled", "/Users/member", "--private-argv", "private-value", "https://private.example"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("prompt attachment leaked %q:\n%s", forbidden, out)
		}
	}

	changed, hasAttachment, notes = ensureClaudeMCPAttachment(dir)
	if changed || !hasAttachment || !containsNote(notes, "already wired") {
		t.Fatalf("second attachment write must be a byte no-op: changed=%v has=%v notes=%v", changed, hasAttachment, notes)
	}
}

func TestEnsureClaudeCharterImportsMCPAttachment(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{"mcpServers":{"thefloor_mcp":{"type":"stdio"}}}`)
	if changed, notes := EnsureClaudeCharter(dir); !changed {
		t.Fatalf("charter with MCP server must write: %v", notes)
	}
	payload, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), claudeMCPAttachmentImport+"\n") {
		t.Fatalf("CLAUDE.md must import MCP attachment:\n%s", payload)
	}
}

func TestEnsureClaudeMCPAttachmentLeavesOpenCodeAttachmentUntouched(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeClaudeConfig(t, filepath.Join(configDir, ".claude.json"), `{"mcpServers":{"thefloor_mcp":{"type":"stdio"}}}`)
	openCodePath := filepath.Join(dir, ".opencode", claudeMCPAttachmentName)
	sentinel := []byte("# Available MCP servers\n\nThis OpenCode sentinel belongs here.\n")
	writeClaudeConfig(t, openCodePath, string(sentinel))

	if changed, notes := EnsureClaudeCharter(dir); !changed {
		t.Fatalf("Claude charter pass did not write its attachment: %v", notes)
	}
	got, err := os.ReadFile(openCodePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("OpenCode-owned sentinel changed:\n--- got ---\n%q\n--- want ---\n%q", got, sentinel)
	}
}
