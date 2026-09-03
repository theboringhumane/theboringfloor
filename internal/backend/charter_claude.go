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
// (TestClaudeLiveCharterReaches, gated THEBORINGOFFICE_LIVE_CLAUDE=1).
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
//     never runs (it is wired from opencode.go's Start only).
//  2. The memory file: create <dir>/CLAUDE.md when absent (office-
//     branded comment line + one line of intent + the import line);
//     no-op when it already references oikonomos.md in ANY form (a
//     hand-written reference is never duplicated); APPEND a marked
//     block (<!-- theboringoffice charter --> ... <!-- /theboringoffice
//     charter --> containing the import) when it exists without the
//     reference — member content above stays byte-identical and the
//     reference check makes the block impossible to append twice.
//
// Hard guarantees mirror the opencode charter: a missing served
// directory is a failure note, never an implicit mkdir of the member's
// project root; a failure never blocks the backend (notes only, no
// error return — charter is best-effort); a second run is byte-
// identical (changed=false, "already wired");
// THEBORINGOFFICE_NO_AUTOCHARTER=1 (legacy: GRAFEIO_NO_AUTOCHARTER=1)
// skips the whole pass.
package backend

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/theboringhumane/theboringoffice/internal/charter"
)

// claudeCharterBeginMarker / claudeCharterEndMarker bracket the appended
// import block so a human can see exactly which bytes the office owns.
// HTML comments are stripped from the model's context (see the header),
// so the markers are free. The begin marker alone also heads the
// created-file form.
const claudeCharterBeginMarker = "<!-- theboringoffice charter -->"
const claudeCharterEndMarker = "<!-- /theboringoffice charter -->"

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
func renderClaudeCharterFile() []byte {
	var b strings.Builder
	b.WriteString(claudeCharterBeginMarker + "\n")
	b.WriteString("This project is served by theboringoffice: the oikonomos manager protocol imported below is the operating charter.\n")
	b.WriteString(claudeCharterImportLine + "\n")
	return []byte(b.String())
}

// renderClaudeCharterBlock is the marked append block for an existing
// CLAUDE.md: begin marker, import, end marker, trailing newline.
func renderClaudeCharterBlock() []byte {
	var b strings.Builder
	b.WriteString(claudeCharterBeginMarker + "\n")
	b.WriteString(claudeCharterImportLine + "\n")
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
// THEBORINGOFFICE_NO_AUTOCHARTER=1 opts out entirely (changed=false,
// note explaining); the pre-rename GRAFEIO_NO_AUTOCHARTER=1 is honored
// as a fallback.
func EnsureClaudeCharter(dir string) (changed bool, notes []string) {
	if envOrLegacy("THEBORINGOFFICE_NO_AUTOCHARTER", "GRAFEIO_NO_AUTOCHARTER") == "1" {
		return false, []string{"[theboringfloor] claude charter: disabled (THEBORINGOFFICE_NO_AUTOCHARTER)"}
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

	// 1. The import target: <dir>/.opencode/oikonomos.md is OFFICE-OWNED
	//    (same discipline as the opencode charter pass) — refresh it to
	//    the embedded charter bytes whenever it drifts: absent, stale
	//    (office upgrade), or hand-edited. Members edit CLAUDE.md, never
	//    the payload; byte-exact freshness keeps claude-only offices
	//    current on upgrades. A clean file costs zero writes and zero
	//    notes (no boot noise).
	ocDir := filepath.Join(dir, ".opencode")
	payloadPath := filepath.Join(ocDir, "oikonomos.md")
	existing, statErr := os.ReadFile(payloadPath)
	switch {
	case statErr == nil && string(existing) == charter.Text:
		// fresh — nothing to do
	case statErr != nil && !os.IsNotExist(statErr):
		return changed, append(notes, "[theboringfloor] claude charter: failed (read "+payloadPath+": "+statErr.Error()+")")
	default:
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (mkdir "+ocDir+": "+err.Error()+")")
		}
		if err := os.WriteFile(payloadPath, []byte(charter.Text), 0o644); err != nil {
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
		if err := os.WriteFile(mdPath, renderClaudeCharterFile(), 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
		}
		changed = true
		notes = append(notes, "[theboringfloor] claude charter: wired (CLAUDE.md created, "+claudeCharterImportLine+")")
	case err != nil:
		return changed, append(notes, "[theboringfloor] claude charter: failed (read "+mdPath+": "+err.Error()+")")
	default:
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
		out = append(out, renderClaudeCharterBlock()...)
		if err := os.WriteFile(mdPath, out, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] claude charter: failed (write "+mdPath+": "+err.Error()+")")
		}
		changed = true
		notes = append(notes, "[theboringfloor] claude charter: wired (import block appended to CLAUDE.md)")
	}

	return changed, notes
}
