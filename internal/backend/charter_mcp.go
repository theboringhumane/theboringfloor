// charter_mcp.go — the charter pass's step 3 (see charter.go's header):
// the MCP prompt attachment. A spawned serve's boss — and through her
// dispatch briefs every developer — should KNOW which MCP servers the
// serve can call, so the attachment lists them where the prompt can see.
//
// Discovery unions the same config chain opencode itself reads, lowest
// precedence first:
//
//	global:  $XDG_CONFIG_HOME/opencode/opencode.{jsonc,json}
//	         (~/.config/opencode/... when XDG is unset — XDG honored
//	         first on every platform the office runs on)
//	project: <dir>/opencode.{jsonc,json}, then
//	         <dir>/.opencode/opencode.{jsonc,json}
//
// Later layers shadow earlier ones by server NAME (whole-entry replace —
// a project "enabled": false entry therefore deletes an inherited global
// server); within one layer the plain .json spelling wins a same-name tie
// over .jsonc; an "enabled": false entry is dropped wherever it lands.
// Every file parses ALONE: a broken or absent sibling degrades without
// sinking the chain, and .jsonc comment/trailing-comma tolerance comes
// from stripJSONC.
//
// Rendering is no-secrets BY CONSTRUCTION: the attachment carries names,
// types, and provenance ("global"/"project") only — never command, args,
// environment, or URL, because the file lands in a prompt and prompts are
// no place for API keys.
//
// The ensure pass mirrors EnsureCharter's guarantees: it writes
// <dir>/.opencode/mcp-servers.md byte-exact (skip when identical) and
// merges "./.opencode/mcp-servers.md" into .opencode/opencode.json's
// instructions beside the charter's own entry, reusing the same
// field-preserving mergeInstruction. Zero configured servers is a clean
// no-op ("no MCP servers configured") — unless a previous run wired one,
// in which case the stale file+entry are RETIRED (unmergeInstruction):
// a prompt advertising dead servers is a lie. Hand-wired spellings of the
// entry are honored (mcpAttachmentAcceptedPaths), so a member's spelling
// never earns a duplicate.
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

// mcpAttachmentRelPath is the attachment's instructions entry spelling,
// mirroring charterRelPath's verified ".opencode/"-explicit shape: opencode
// resolves relative instructions against the PROJECT directory, not the
// config file's own directory.
const mcpAttachmentRelPath = "./.opencode/mcp-servers.md"

// mcpAttachmentAcceptedPaths are all spellings the pass treats as
// "already wired" — a member hand-writing any of these must not get a
// duplicate (mirrors charterAcceptedPaths).
var mcpAttachmentAcceptedPaths = []string{
	mcpAttachmentRelPath,
	".opencode/mcp-servers.md",
	"./mcp-servers.md",
	"mcp-servers.md",
}

// mcpServerHint is one discovered MCP server, reduced to what a prompt
// may see: its name, its transport type, and which config layer declared
// it. Command/args/environment/URL deliberately have no field here.
type mcpServerHint struct {
	name   string
	typ    string
	source string // "global" | "project"
}

// mcpConfigRef is one link in the config chain: a file that might carry
// an "mcp" object, tagged with the layer it came from for provenance.
type mcpConfigRef struct {
	path   string
	source string // "global" | "project"
}

// mcpConfigChain lists the chain lowest-precedence-first. Within a layer
// .jsonc rides before plain .json so the plain spelling wins a same-name
// tie; global rides before project, and the project root before
// .opencode. Absent files are simply skipped later by the reader.
func mcpConfigChain(dir string) []mcpConfigRef {
	var chain []mcpConfigRef
	spellings := []string{"opencode.jsonc", "opencode.json"}

	globalDir := ""
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		globalDir = filepath.Join(xdg, "opencode")
	} else if home := os.Getenv("HOME"); home != "" {
		globalDir = filepath.Join(home, ".config", "opencode")
	}
	if globalDir != "" {
		for _, sp := range spellings {
			chain = append(chain, mcpConfigRef{filepath.Join(globalDir, sp), "global"})
		}
	}
	for _, sub := range []string{"", ".opencode"} {
		for _, sp := range spellings {
			chain = append(chain, mcpConfigRef{filepath.Join(dir, sub, sp), "project"})
		}
	}
	return chain
}

// discoverMCPServers unions the enabled MCP servers across the chain,
// sorted by name. Each file is read and parsed on its own — a missing or
// unparseable link degrades alone, never sinks the chain. Shadowing is
// whole-entry by name: the LAST layer to declare a name owns it, and an
// "enabled": false declaration lands in the same slot and is dropped.
func discoverMCPServers(dir string) []mcpServerHint {
	byName := map[string]mcpServerHint{}
	disabled := map[string]bool{}
	for _, ref := range mcpConfigChain(dir) {
		raw, err := os.ReadFile(ref.path)
		if err != nil {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(stripJSONC(raw), &doc); err != nil {
			continue // broken json/jsonc: this link degrades alone
		}
		mcp, ok := doc["mcp"].(map[string]any)
		if !ok {
			continue
		}
		for name, rawEntry := range mcp {
			entry, ok := rawEntry.(map[string]any)
			if !ok {
				continue // a hand-shaped value is left alone
			}
			typ, _ := entry["type"].(string)
			if typ == "" {
				// Old-style configs omit "type": a URL spells remote,
				// anything else is local.
				if _, hasURL := entry["url"]; hasURL {
					typ = "remote"
				} else {
					typ = "local"
				}
			}
			byName[name] = mcpServerHint{name: name, typ: typ, source: ref.source}
			disabled[name] = false
			if enabled, ok := entry["enabled"].(bool); ok && !enabled {
				disabled[name] = true
			}
		}
	}
	servers := make([]mcpServerHint, 0, len(byName))
	for name, hint := range byName {
		if disabled[name] {
			continue
		}
		servers = append(servers, hint)
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].name < servers[j].name })
	return servers
}

// stripJSONC makes opencode's JSONC configs parseable by encoding/json:
// // line comments and /* block comments */ vanish (newlines survive so
// error positions stay sane), and then trailing commas before } or ] are
// dropped. String contents are SACRED — a // inside a quoted URL is data,
// not a comment, and escaped quotes (\") never flip the string state.
func stripJSONC(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inStr, esc, inLine, inBlock := false, false, false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
				out = append(out, c)
			}
		case inBlock:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
		case inStr:
			out = append(out, c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
		default:
			switch {
			case c == '"':
				inStr = true
				out = append(out, c)
			case c == '/' && i+1 < len(src) && src[i+1] == '/':
				inLine = true
				i++
			case c == '/' && i+1 < len(src) && src[i+1] == '*':
				inBlock = true
				i++
			default:
				out = append(out, c)
			}
		}
	}
	return stripTrailingCommas(out)
}

// stripTrailingCommas drops a comma whose next real (non-whitespace)
// character closes an object or array — JSONC's trailing comma. Commas
// inside strings are content and are never touched.
func stripTrailingCommas(src []byte) []byte {
	out := make([]byte, 0, len(src))
	inStr, esc := false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inStr {
			out = append(out, c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case ',':
			j := i + 1
			for j < len(src) && (src[j] == ' ' || src[j] == '\t' || src[j] == '\n' || src[j] == '\r') {
				j++
			}
			if j < len(src) && (src[j] == '}' || src[j] == ']') {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// renderMCPAttachment is the prompt-visible artifact: names, types, and
// provenance, LF-only, ending on a newline — same byte discipline as the
// embedded charter. It also teaches the ADDRESSING scheme
// ("<server>_<tool>") so briefs can name tools usefully. Deterministic:
// the same servers always render the same bytes (idempotence depends on
// it).
func renderMCPAttachment(servers []mcpServerHint) []byte {
	var b strings.Builder
	b.WriteString("# Available MCP servers\n")
	b.WriteString("\n")
	b.WriteString("The opencode serve backing this session has the MCP servers\n")
	b.WriteString("below configured. Their tools are callable as `<server>_<tool>`\n")
	b.WriteString("(the serve namespaces every MCP tool with its server name).\n")
	b.WriteString("\n")
	for _, s := range servers {
		fmt.Fprintf(&b, "- `%s` (%s) — configured in %s opencode config\n", s.name, s.typ, s.source)
	}
	b.WriteString("\n")
	b.WriteString("Only names, types, and provenance are listed — never commands,\n")
	b.WriteString("URLs, or environment: this file lands in a prompt, and prompts are\n")
	b.WriteString("no place for API keys. When a dispatch needs one of these servers,\n")
	b.WriteString("name it in the dispatch brief's CONTEXT so the developer knows the\n")
	b.WriteString("tools exist; the developer discovers the exact tool names live.\n")
	b.WriteString("\n")
	b.WriteString("Usage discipline:\n")
	b.WriteString("- Prefer a purpose-built MCP tool over a shell workaround when one\n")
	b.WriteString("  covers the job (memory recall, web fetch, tracing) — but never\n")
	b.WriteString("  guess a tool name: list or probe the server's tools first.\n")
	b.WriteString("- A failing or absent MCP tool is an ISSUE to report, not a reason\n")
	b.WriteString("  to fake its output.\n")
	b.WriteString("- Servers listed here may still be down at call time; treat the\n")
	b.WriteString("  first call as the health check.\n")
	return []byte(b.String())
}

// ensureMCPAttachment is the chartered step-3 pass for dir: with servers
// discovered, wire the attachment file + instructions entry
// (idempotently); with none, retire any stale wiring or no-op. Its notes
// ride AHEAD of the charter's final summary line (probes pattern-match
// the tail), and changed=true tells Start an already-running serve may be
// stale, exactly like the charter's own writes.
func ensureMCPAttachment(dir string) (changed bool, notes []string) {
	servers := discoverMCPServers(dir)
	ocDir := filepath.Join(dir, ".opencode")
	attPath := filepath.Join(ocDir, "mcp-servers.md")
	cfgPath := filepath.Join(ocDir, "opencode.json")

	// Zero configured servers: the attachment must not exist. Retire a
	// stale file + entry when a previous pass wired one; otherwise this
	// is a clean no-op that SAYS so.
	if len(servers) == 0 {
		retired := false
		if _, err := os.Stat(attPath); err == nil {
			if err := os.Remove(attPath); err != nil {
				return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (remove "+attPath+": "+err.Error()+")")
			}
			retired = true
		}
		if cfgRaw, err := os.ReadFile(cfgPath); err == nil {
			unmerged, cut, unmergeErr := unmergeInstruction(cfgRaw, mcpAttachmentAcceptedPaths)
			if unmergeErr != nil {
				return retired, append(notes, "[theboringoffice] mcp prompt attachment: failed (unmerge "+cfgPath+": "+unmergeErr.Error()+")")
			}
			if cut {
				if err := os.WriteFile(cfgPath, unmerged, 0o644); err != nil {
					return retired, append(notes, "[theboringoffice] mcp prompt attachment: failed (write "+cfgPath+": "+err.Error()+")")
				}
				retired = true
			}
		} else if !os.IsNotExist(err) {
			return retired, append(notes, "[theboringoffice] mcp prompt attachment: failed (read "+cfgPath+": "+err.Error()+")")
		}
		if retired {
			return true, append(notes, "[theboringoffice] mcp prompt attachment: removed (no MCP servers configured — retired stale .opencode/mcp-servers.md)")
		}
		return false, append(notes, "[theboringoffice] mcp prompt attachment: no MCP servers configured")
	}

	// 1. The attachment markdown: write byte-exact, skip when identical.
	want := renderMCPAttachment(servers)
	if got, err := os.ReadFile(attPath); err != nil || !bytes.Equal(got, want) {
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (mkdir "+ocDir+": "+err.Error()+")")
		}
		if err := os.WriteFile(attPath, want, 0o644); err != nil {
			return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (write "+attPath+": "+err.Error()+")")
		}
		changed = true
	}

	// 2. The instructions entry rides beside the charter's through the
	//    same field-preserving merge. The charter's step 2 guarantees the
	//    config exists by now; the NotExist branch is defensive only.
	cfgRaw, err := os.ReadFile(cfgPath)
	if err == nil {
		merged, mergeChanged, mergeErr := mergeInstruction(cfgRaw, mcpAttachmentRelPath, mcpAttachmentAcceptedPaths)
		if mergeErr != nil {
			return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (merge "+cfgPath+": "+mergeErr.Error()+")")
		}
		if mergeChanged {
			if err := os.WriteFile(cfgPath, merged, 0o644); err != nil {
				return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (write "+cfgPath+": "+err.Error()+")")
			}
			changed = true
		}
	} else if os.IsNotExist(err) {
		fresh := []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"instructions\": [\n    \"" + mcpAttachmentRelPath + "\"\n  ]\n}\n")
		if err := os.WriteFile(cfgPath, fresh, 0o644); err != nil {
			return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (write "+cfgPath+": "+err.Error()+")")
		}
		changed = true
	} else {
		return changed, append(notes, "[theboringoffice] mcp prompt attachment: failed (read "+cfgPath+": "+err.Error()+")")
	}

	if changed {
		word := "servers"
		if len(servers) == 1 {
			word = "server"
		}
		return true, append(notes, fmt.Sprintf("[theboringoffice] mcp prompt attachment: wired (.opencode/mcp-servers.md — %d %s)", len(servers), word))
	}
	return false, append(notes, "[theboringoffice] mcp prompt attachment: already wired (.opencode/mcp-servers.md)")
}

// unmergeInstruction is mergeInstruction's mirror for the retirement
// path: every instructions entry matching ANY accepted spelling is cut,
// the rest preserved in order, and the doc re-marshaled with the same
// 2-space indent + trailing newline discipline. Fail-closed on the same
// hand-shaped inputs mergeInstruction refuses (unparseable top level,
// null or non-array instructions) — a member's hand-rolled config is
// never clobbered. When no entry matches, the ORIGINAL bytes ride back
// untouched (cut=false) so idempotence checks see bit-identical files.
func unmergeInstruction(cfg []byte, accepted []string) (merged []byte, cut bool, err error) {
	var doc map[string]any
	if err := json.Unmarshal(cfg, &doc); err != nil {
		return nil, false, fmt.Errorf("unparseable json: %w", err)
	}

	rawInstr, has := doc["instructions"]
	if !has {
		return cfg, false, nil
	}
	if rawInstr == nil {
		return nil, false, fmt.Errorf("instructions is null — refusing to rewrite a hand-shaped config")
	}
	arr, ok := rawInstr.([]any)
	if !ok {
		return nil, false, fmt.Errorf("instructions is %T, not an array — refusing to rewrite a hand-shaped config", rawInstr)
	}

	kept := make([]any, 0, len(arr))
	for _, item := range arr {
		drop := false
		if s, ok := item.(string); ok {
			for _, spelling := range accepted {
				if s == spelling {
					drop = true
					break
				}
			}
		}
		if !drop {
			kept = append(kept, item)
		}
	}
	if len(kept) == len(arr) {
		return cfg, false, nil // entry absent: original bytes ride back
	}

	doc["instructions"] = kept
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}
