// charter_claude.go — the Claude Code counterpart of charter.go. The
// opencode pass wires the oikonomos manager charter through
// .opencode/opencode.json's instructions array, which the Claude Code
// CLI reads NONE of: `claude -p` takes its project memory from
// CLAUDE.md (the served directory + every ancestor), loaded at launch.
// VERIFIED against the current Anthropic docs
// (https://docs.anthropic.com/en/docs/claude-code/memory):
//
//   - "Claude Code loads CLAUDE.md and CLAUDE.local.md from your current
//     working directory and every directory above it." — and project
//     memory lives at "./CLAUDE.md or ./.claude/CLAUDE.md".
//   - "CLAUDE.md files can import additional files using @path/to/import
//     syntax... Relative paths resolve relative to the file containing
//     the import, not the working directory." Hence the import line
//     "@.opencode/oikonomos.md": written into <dir>/CLAUDE.md it
//     resolves to <dir>/.opencode/oikonomos.md, INSIDE the working
//     directory, so it is not an "external import" and never trips the
//     CLI's external-import approval dialog.
//   - "Block-level HTML comments (<!-- ... -->) in CLAUDE.md files are
//     stripped before the content is injected into Claude's context." —
//     the office's begin/end markers below cost zero context tokens.
//
// …and against the installed CLI 2.1.247 live: a scratch dir carrying
// only the generated CLAUDE.md + a trivial marker payload answers the
// marker back through the office's own spawn path
// (TestClaudeLiveCharterReaches, gated THEFLOOR_LIVE_CLAUDE=1).
//
// The pass has two halves:
//
//  1. The payload: <dir>/.opencode/oikonomos.md is OFFICE-OWNED (the
//     same discipline as the opencode charter pass) — refreshed to the
//     embedded charter (internal/charter.Text) whenever it drifts:
//     absent, stale after an office upgrade, or hand-edited. Members
//     edit CLAUDE.md, never the payload; byte-exact freshness keeps
//     claude-only offices current on upgrades, and a drift-free file
//     costs zero writes and zero boot notes. Without this half the
//     import dangles on claude-backed offices, where EnsureCharter
//     never runs (it is wired from opencode.go's Start only). When Claude
//     has MCP servers, CLAUDE.md imports the prompt-safe server list directly.
//  2. The memory file: create <dir>/CLAUDE.md when absent (office-
//     branded comment line + one line of intent + the import line);
//     no-op when it already references oikonomos.md in ANY form (a
//     hand-written reference is never duplicated); APPEND a marked
//     block (<!-- theboringfloor charter --> ... <!-- /theboringfloor
//     charter --> containing the import) when it exists without the
//     reference — member content above stays byte-identical and the
//     reference check makes the block impossible to append twice.
//
// Hard guarantees mirror the opencode charter: a missing served
// directory is a failure note, never an implicit mkdir of the member's
// project root; a failure never blocks the backend (notes only, no
// error return — charter is best-effort); a second run is byte-
// identical (changed=false, "already wired");
// THEFLOOR_NO_AUTOCHARTER=1 skips the whole pass.
package backend

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/theboringhumane/theboringfloor/internal/charter"
	"github.com/theboringhumane/theboringfloor/internal/config"
)

// claudeCharterBeginMarker / claudeCharterEndMarker bracket the appended
// import block so a human can see exactly which bytes the office owns.
// HTML comments are stripped from the model's context (see the header),
// so the markers are free. The begin marker alone also heads the
// created-file form.
const claudeCharterBeginMarker = "<!-- theboringfloor charter -->"
const claudeCharterEndMarker = "<!-- /theboringfloor charter -->"

// claudeCharterImportLine is the memory import, spelled per the
// Anthropic docs: "@path/to/import", relative to the file containing it
// — <dir>/CLAUDE.md importing <dir>/.opencode/oikonomos.md.
const claudeCharterImportLine = "@.opencode/oikonomos.md"

// claudeCharterRefNeedle is the "already references oikonomos.md (any
// form)" probe: any mention of the payload filename — an import, a
// prose reference, a differently-spelled path — counts as wired. A
// member who referenced the charter by hand never gets a duplicate
// block (charterAcceptedPaths parity, substring-simple).
const claudeCharterRefNeedle = "oikonomos.md"

// renderClaudeCharterFile is the fresh CLAUDE.md, byte-exact: the office
// comment line, one line of intent, the import line. LF-only, ASCII,
// ends on a newline — the same determinism discipline as every
// prompt-visible artifact here.
func renderClaudeCharterFile(hasMCPAttachment bool) []byte {
	var b strings.Builder
	b.WriteString(claudeCharterBeginMarker + "\n")
	b.WriteString("This project is served by theboringfloor: the oikonomos manager protocol imported below is the operating charter.\n")
	b.WriteString(claudeCharterImportLine + "\n")
	if hasMCPAttachment {
		b.WriteString(claudeMCPAttachmentImport + "\n")
	}
	return []byte(b.String())
}

// renderClaudeCharterBlock is the marked append block for an existing
// CLAUDE.md: begin marker, import, end marker, trailing newline.
func renderClaudeCharterBlock(hasMCPAttachment bool) []byte {
	var b strings.Builder
	b.WriteString(claudeCharterBeginMarker + "\n")
	b.WriteString(claudeCharterImportLine + "\n")
	if hasMCPAttachment {
		b.WriteString(claudeMCPAttachmentImport + "\n")
	}
	b.WriteString(claudeCharterEndMarker + "\n")
	return []byte(b.String())
}

// EnsureClaudeCharter wires the bundled oikonomos manager charter into
// dir so a spawned `claude -p` reads it — the liveClaudeBackend.Start
// counterpart of opencode's EnsureCharter. changed is true when any byte
// was written; notes are status-line text describing what happened (a
// failure degrades to a note, never an error — the backend boots
// regardless). Idempotent: a follow-up run with everything in place
// returns changed=false and writes nothing.
// THEFLOOR_NO_AUTOCHARTER=1 opts out entirely (changed=false, note explaining).
func EnsureClaudeCharter(dir string) (changed bool, notes []string) {
	if config.EnvBool("NO_AUTOCHARTER") {
		return false, []string{"[theboringfloor] claude charter: disabled (THEFLOOR_NO_AUTOCHARTER)"}
	}

	// The served project root must already exist — the office never
	// creates a member's project directory implicitly.
	st, err := os.Stat(dir)
	if err != nil {
		return false, []string{"[theboringfloor] claude charter: failed (dir missing: " + dir + ": " + err.Error() + ")"}
	}
	if !st.IsDir() {
		return false, []string{"[theboringfloor] claude charter: failed (not a directory: " + dir + ")"}
	}

	// 1. Discover and refresh the prompt-safe MCP attachment before writing
	//    the office-owned payload and its CLAUDE.md import.
	mcpChanged, hasMCPAttachment, mcpNotes := ensureClaudeMCPAttachment(dir)
	changed = changed || mcpChanged
	notes = append(notes, mcpNotes...)

	// 2. The import target: <dir>/.opencode/oikonomos.md is OFFICE-OWNED
	//    (same discipline as the opencode charter pass) — refresh it to
	//    the embedded charter bytes whenever it drifts: absent, stale
	//    (office upgrade), or hand-edited. Members edit CLAUDE.md, never
	//    the payload; byte-exact freshness keeps claude-only offices
	//    current on upgrades. A clean file costs zero writes and zero
	//    notes (no boot noise).
	ocDir := filepath.Join(dir, ".opencode")
	payloadPath := filepath.Join(ocDir, "oikonomos.md")
	wantPayload := renderClaudeCharterPayload()
	existing, statErr := os.ReadFile(payloadPath)
	switch {
	case statErr == nil && bytes.Equal(existing, wantPayload):
		// fresh — nothing to do
	case statErr != nil && !os.IsNotExist(statErr):
		return changed, append(notes, "[theboringfloor] claude charter: failed (read "+payloadPath+": "+statErr.Error()+")")
	default:
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (mkdir "+ocDir+": "+err.Error()+")")
		}
		if err := os.WriteFile(payloadPath, wantPayload, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (write "+payloadPath+": "+err.Error()+")")
		}
		changed = true
		if os.IsNotExist(statErr) {
			notes = append(notes, "[theboringfloor] claude charter: seeded .opencode/oikonomos.md (the import target)")
		} else {
			notes = append(notes, "[theboringfloor] claude charter: refreshed .opencode/oikonomos.md (drifted from the embedded charter)")
		}
	}

	// 2. The memory file: create / no-op / append-marked-block.
	mdPath := filepath.Join(dir, "CLAUDE.md")
	raw, err := os.ReadFile(mdPath)
	switch {
	case os.IsNotExist(err):
		if err := os.WriteFile(mdPath, renderClaudeCharterFile(hasMCPAttachment), 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
		}
		changed = true
		notes = append(notes, "[theboringfloor] claude charter: wired (CLAUDE.md created, "+claudeCharterImportLine+")")
	case err != nil:
		return changed, append(notes, "[theboringfloor] claude charter: failed (read "+mdPath+": "+err.Error()+")")
	default:
		if isClaudeCharterStarter(raw) && !bytes.Equal(raw, renderClaudeCharterFile(hasMCPAttachment)) {
			if err := os.WriteFile(mdPath, renderClaudeCharterFile(hasMCPAttachment), 0o644); err != nil {
				return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
			}
			changed = true
			notes = append(notes, "[theboringfloor] claude charter: refreshed (CLAUDE.md drifted from the office starter)")
			break
		}
		if migrated, ok := migrateClaudeCharterBlock(raw, hasMCPAttachment); ok {
			if err := os.WriteFile(mdPath, migrated, 0o644); err != nil {
				return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
			}
			changed = true
			notes = append(notes, "[theboringfloor] claude charter: refreshed (CLAUDE.md office import block migrated)")
			break
		}
		if strings.Contains(string(raw), claudeCharterRefNeedle) {
			// Any-form reference — hand-written or office-written — is
			// wired. No bytes move.
			notes = append(notes, "[theboringfloor] claude charter: already wired (CLAUDE.md)")
			break
		}
		// Member content stays byte-identical: append only. Normalize
		// to exactly one blank line between the member's tail and the
		// office block (an empty file gets the block bare).
		out := raw
		if len(out) > 0 {
			if !bytes.HasSuffix(out, []byte("\n")) {
				out = append(out, '\n')
			}
			out = append(out, '\n')
		}
		out = append(out, renderClaudeCharterBlock(hasMCPAttachment)...)
		if err := os.WriteFile(mdPath, out, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
		}
		changed = true
		notes = append(notes, "[theboringfloor] claude charter: wired (import block appended to CLAUDE.md)")
	}

	return changed, notes
}

func renderClaudeCharterPayload() []byte {
	return []byte(charter.Text)
}

// isClaudeCharterStarter identifies the three-line, office-owned starter
// generated by renderClaudeCharterFile. The marker text itself may differ
// after a product rename, so retain the structural test to refresh a stale
// generated file without touching a member-authored CLAUDE.md.
func isClaudeCharterStarter(raw []byte) bool {
	lines := bytes.Split(raw, []byte("\n"))
	if len(lines) != 4 && len(lines) != 5 {
		return false
	}
	if len(lines) == 5 && string(lines[3]) != claudeMCPAttachmentImport {
		return false
	}
	return bytes.HasPrefix(lines[0], []byte("<!-- ")) &&
		bytes.HasSuffix(lines[0], []byte(" charter -->")) &&
		bytes.HasPrefix(lines[1], []byte("This project is served by ")) &&
		string(lines[2]) == claudeCharterImportLine &&
		len(lines[len(lines)-1]) == 0
}

// migrateClaudeCharterBlock replaces only the exact, trailing office-owned
// append block. It leaves every member byte above the block untouched while
// moving any former MCP attachment import to Claude's private directory.
func migrateClaudeCharterBlock(raw []byte, hasMCPAttachment bool) ([]byte, bool) {
	want := renderClaudeCharterBlock(hasMCPAttachment)
	if bytes.HasSuffix(raw, want) {
		return nil, false
	}
	for _, old := range [][]byte{
		[]byte(claudeCharterBeginMarker + "\n" + claudeCharterImportLine + "\n" + claudeCharterEndMarker + "\n"),
		[]byte(claudeCharterBeginMarker + "\n" + claudeCharterImportLine + "\n@mcp-servers.md\n" + claudeCharterEndMarker + "\n"),
	} {
		if bytes.HasSuffix(raw, old) {
			return append(raw[:len(raw)-len(old):len(raw)-len(old)], want...), true
		}
	}
	return nil, false
}
