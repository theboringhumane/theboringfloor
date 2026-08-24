// charter_mcp_test.go — the MCP prompt-attachment contract: discovery
// unions the config chain opencode itself reads (global + project, .json +
// .jsonc, project shadows global by name; enabled:false drops), the
// attachment renders names/types/provenance only (never command/env —
// prompts are no place for API keys), the ensure pass wires AND retires
// the file+entry idempotently, and zero servers stay a clean no-op.
package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMCPConfig is the scratch-config scribe: body lands at
// <root>/<sub>/opencode<ext> (sub "" = the root itself).
func writeMCPConfig(t *testing.T, root, sub, ext, body string) string {
	t.Helper()
	dir := filepath.Join(root, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "opencode"+ext)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStripJSONC(t *testing.T) {
	src := `{
  // a line comment
  "mcp": {
    "web": {"type": "remote", "url": "https://example.com/mcp"}, // tail comment
    /* a block
       comment */
    "local": {"type": "local", "command": ["npx", "x"]},
  },
}
`
	var doc map[string]any
	if err := json.Unmarshal(stripJSONC([]byte(src)), &doc); err != nil {
		t.Fatalf("comments + trailing commas must become parseable: %v", err)
	}
	mcp := doc["mcp"].(map[string]any)
	if len(mcp) != 2 || mcp["web"].(map[string]any)["url"] != "https://example.com/mcp" {
		t.Fatalf("the URL string must survive the stripper intact, got %v", mcp)
	}
}

func TestStripJSONCKeepsStringSlashes(t *testing.T) {
	src := []byte(`{"note": "http://host/a // not a comment", "x": 1}`)
	got := stripJSONC(src)
	if !strings.Contains(string(got), "http://host/a // not a comment") {
		t.Fatalf("// inside a string is CONTENT, got %s", got)
	}
}

func TestDiscoverMCPServersUnionAndPrecedence(t *testing.T) {
	global := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", global)
	// global: two spellings, one shared name (plain .json wins the type).
	writeMCPConfig(t, global, "opencode", ".json", `{"mcp":{
		"shared": {"type": "local"},
		"off-global": {"type": "local"},
		"global-only": {"type": "remote"}
	}}`)
	writeMCPConfig(t, global, "opencode", ".jsonc", `{"mcp":{
		"shared": {"type": "remote"},
		"jsonc-only": {"type": "local"}
	}}`)
	dir := t.TempDir()
	// project root + .opencode: shadow "shared", disable an inherited one.
	writeMCPConfig(t, dir, "", ".json", `{"mcp":{"root": {"type": "local"}}}`)
	writeMCPConfig(t, dir, ".opencode", ".json", `{"mcp":{
		"shared": {"type": "remote"},
		"off-global": {"enabled": false},
		"off-local": {"type": "local", "enabled": false}
	}}`)

	servers := discoverMCPServers(dir)
	names := make([]string, 0, len(servers))
	byName := map[string]mcpServerHint{}
	for _, s := range servers {
		names = append(names, s.name)
		byName[s.name] = s
	}
	want := []string{"global-only", "jsonc-only", "root", "shared"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("union must be enabled servers sorted by name: want %v, got %v", want, names)
	}
	if s := byName["shared"]; s.typ != "remote" || s.source != "project" {
		t.Fatalf("the project layer shadows global by name, got %+v", s)
	}
	if s := byName["global-only"]; s.source != "global" {
		t.Fatalf("provenance tags the layer, got %+v", s)
	}
}

func TestDiscoverMCPServersDegradesAlone(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	// Broken json, broken jsonc (unterminated), and one good config all
	// sit in the chain: the good one alone answers.
	writeMCPConfig(t, dir, "", ".json", `{"mcp": {`)
	writeMCPConfig(t, dir, "", ".jsonc", `{"mcp": {"a": `)
	writeMCPConfig(t, dir, ".opencode", ".json", `{"mcp":{"ok": {"type": "local"}}}`)
	servers := discoverMCPServers(dir)
	if len(servers) != 1 || servers[0].name != "ok" {
		t.Fatalf("unparseable files degrade alone, got %+v", servers)
	}
}

func TestDiscoverMCPServersNone(t *testing.T) {
	pinGlobalConfig(t)
	if servers := discoverMCPServers(t.TempDir()); len(servers) != 0 {
		t.Fatalf("a project + empty global must discover nothing, got %+v", servers)
	}
}

func TestRenderMCPAttachmentNarratesNoSecrets(t *testing.T) {
	servers := []mcpServerHint{
		{name: "agentmemory", typ: "local", source: "global"},
		{name: "db", typ: "remote", source: "project"},
	}
	out := string(renderMCPAttachment(servers))
	for _, want := range []string{
		"# Available MCP servers",
		"`<server>_<tool>`",
		"- `agentmemory` (local) — configured in global opencode config",
		"- `db` (remote) — configured in project opencode config",
		"dispatch brief's CONTEXT",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("attachment missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\r") {
		t.Fatal("generated attachment must be LF-only like the charter")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatal("attachment must end on a newline")
	}
}

func TestEnsureMCPAttachmentWiresAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("HOME", cfgDir)
	writeMCPConfig(t, cfgDir, "opencode", ".json", `{"mcp":{
		"mem": {"type": "local", "command": ["npx", "-y", "@x/mcp"], "environment": {"API_KEY": "shhh-secret"}},
		"web": {"type": "remote", "url": "https://mcp.example.com"}
	}}`)

	// The charter's step 2 guarantees the config exists by step 3's turn.
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "opencode.json"),
		[]byte(`{"instructions":["./.opencode/oikonomos.md"],"theme":"nord"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, notes := EnsureCharter(dir)
	if !changed {
		t.Fatal("first run with MCP servers: changed=false, want true")
	}
	if !containsNote(notes, "mcp prompt attachment: wired") || !containsNote(notes, "2 servers") {
		t.Fatalf("notes must announce the wiring + count: %v", notes)
	}
	attachment := mustRead(t, filepath.Join(ocDir, "mcp-servers.md"))
	if !strings.Contains(attachment, "- `mem` (local)") || !strings.Contains(attachment, "- `web` (remote)") {
		t.Fatalf("attachment lists the two servers:\n%s", attachment)
	}
	if strings.Contains(attachment, "shhh-secret") || strings.Contains(attachment, "npx") || strings.Contains(attachment, "mcp.example.com") {
		t.Fatalf("commands/URLs/env never enter a prompt-visible file:\n%s", attachment)
	}
	cfg1 := mustRead(t, filepath.Join(ocDir, "opencode.json"))
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfg1), &cfg); err != nil {
		t.Fatal(err)
	}
	instr := cfg["instructions"].([]any)
	if len(instr) != 2 || instr[0] != charterRelPath || instr[1] != mcpAttachmentRelPath {
		t.Fatalf("charter + attachment ride instructions together: %v", instr)
	}
	if cfg["theme"] != "nord" {
		t.Fatalf("foreign fields survive the second merge: %v", cfg)
	}

	// Second run: no bytes move, changed=false.
	changed, notes = EnsureCharter(dir)
	if changed {
		t.Fatalf("second run: changed=true, want false (notes %v)", notes)
	}
	if !containsNote(notes, "mcp prompt attachment: already wired") {
		t.Fatalf("second run notes: %v", notes)
	}
	if got := mustRead(t, filepath.Join(ocDir, "opencode.json")); got != cfg1 {
		t.Fatal("second run rewrote opencode.json")
	}
}

func TestEnsureMCPAttachmentAcceptsHandWiredSpellings(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMCPConfig(t, dir, "", ".json", `{"mcp":{"one": {"type": "local"}}}`)

	// Run 0: the pass itself wires the attachment file + entry.
	if changed, _ := EnsureCharter(dir); !changed {
		t.Fatal("run 0: want the attachment wired")
	}

	// A member hand-writing ANY accepted spelling gets neither a
	// duplicate entry nor a rewritten config on the next pass.
	for _, spelling := range mcpAttachmentAcceptedPaths {
		body := `{"instructions":["` + spelling + `","./.opencode/oikonomos.md"]}`
		cfgPath := filepath.Join(ocDir, "opencode.json")
		if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if changed, _ := EnsureCharter(dir); changed {
			t.Fatalf("spelling %q: changed=true, want false (accepted as wired)", spelling)
		}
		if got := mustRead(t, cfgPath); got != body {
			t.Fatalf("spelling %q: config rewritten: %s", spelling, got)
		}
	}
}

func TestEnsureMCPAttachmentNoServersIsCleanNoop(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	changed, notes := EnsureCharter(dir) // wires the charter itself
	if !changed {
		t.Fatal("fresh dir: changed=false, want true")
	}
	if !containsNote(notes, "no MCP servers configured") {
		t.Fatalf("the no-op must SAY none configured: %v", notes)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opencode", "mcp-servers.md")); !os.IsNotExist(err) {
		t.Fatal("no attachment file when nothing is configured")
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(mustRead(t, filepath.Join(dir, ".opencode", "opencode.json"))), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg["instructions"].([]any)) != 1 {
		t.Fatalf("no attachment entry when nothing is configured: %v", cfg["instructions"])
	}
}

func TestEnsureMCPAttachmentRetiresStale(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	// Run 1 with a configured server: attachment wired.
	writeMCPConfig(t, dir, "", ".json", `{"mcp":{"gone": {"type": "local"}}}`)
	if changed, _ := EnsureCharter(dir); !changed {
		t.Fatal("run 1: want changed=true")
	}
	attPath := filepath.Join(dir, ".opencode", "mcp-servers.md")
	if _, err := os.Stat(attPath); err != nil {
		t.Fatalf("attachment must exist after run 1: %v", err)
	}
	// The server vanishes from the config: the attachment must RETIRE —
	// a prompt advertising dead servers is a lie.
	if err := os.Remove(filepath.Join(dir, "opencode.json")); err != nil {
		t.Fatal(err)
	}
	changed, notes := EnsureCharter(dir)
	if !changed {
		t.Fatalf("retiring a stale attachment is a change: %v", notes)
	}
	if !containsNote(notes, "mcp prompt attachment: removed") {
		t.Fatalf("retirement must be announced: %v", notes)
	}
	if _, err := os.Stat(attPath); !os.IsNotExist(err) {
		t.Fatal("stale attachment file must be removed")
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(mustRead(t, filepath.Join(dir, ".opencode", "opencode.json"))), &cfg); err != nil {
		t.Fatal(err)
	}
	for _, e := range cfg["instructions"].([]any) {
		if e == mcpAttachmentRelPath {
			t.Fatalf("stale attachment entry must be unmerged: %v", cfg["instructions"])
		}
	}
	// And the steady state settles right back to idempotent.
	if changed, _ := EnsureCharter(dir); changed {
		t.Fatal("after retirement the pass must settle to changed=false")
	}
}

func TestUnmergeInstructionFailClosed(t *testing.T) {
	for _, body := range []string{
		`{"instructions":null}`,
		`{"instructions":"docs/*.md"}`,
		`{"instructions": [`,
	} {
		if _, _, err := unmergeInstruction([]byte(body), mcpAttachmentAcceptedPaths); err == nil {
			t.Fatalf("%s must fail closed, not rewrite a hand-shaped config", body)
		}
	}
	// Absent entry: the original bytes ride back untouched.
	body := `{"instructions":["docs/style.md"],"x":1}`
	got, cut, err := unmergeInstruction([]byte(body), mcpAttachmentAcceptedPaths)
	if err != nil || cut || string(got) != body {
		t.Fatalf("entry absent must be a no-op: cut=%v err=%v got=%s", cut, err, got)
	}
}
