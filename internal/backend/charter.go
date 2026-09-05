// charter.go — bundles the oikonomos manager charter into whatever
// directory theboringfloor's live backend serves. On Start (live only) it:
//
//  1. writes the embedded charter text (internal/charter) byte-exact to
//     <dir>/.opencode/oikonomos.md,
//  2. merges "./.opencode/oikonomos.md" into
//     <dir>/.opencode/opencode.json's "instructions" array. VERIFIED
//     against serve 1.18.19: opencode resolves relative instructions
//     entries against the PROJECT directory (globUp from the served dir
//     up to the worktree), NOT the config file's own directory — so the
//     entry must name .opencode/ explicitly. Creates the config with
//     {"instructions":["./.opencode/oikonomos.md"]} when absent and
//     surgically adds the entry when present, and
//  3. wires the MCP prompt attachment (charter_mcp.go): generates
//     <dir>/.opencode/mcp-servers.md listing the available MCP servers
//     (discovered from the same config chain the serve reads) and merges
//     it into instructions beside the charter, so the boss — and through
//     her briefs the developers — know which MCP tools exist, and
//  4. wires the office ledger (the charter-merge seam for the sibling
//     ledger writer): seeds <dir>/.opencode/office-ledger.md with its
//     header when ABSENT — never rewriting an existing file, because the
//     ledger is append-only state the app records completed dispatches
//     into — and merges "./.opencode/office-ledger.md" into instructions
//     beside the charter + MCP attachment, so the boss consults the
//     office's completed-work memory BEFORE re-dispatching.
//
// Hard guarantees: AGENTS.md / CLAUDE.md and every other opencode.json
// field are never touched; a second run is byte-identical (changed=false,
// "already wired"); THEFLOOR_NO_AUTOCHARTER=1 skips the whole pass. A fresh
// charter or a newly-added instructions entry reports changed=true so a
// serve started earlier can be found stale and respawned (opencode spoils
// its config at start — a running serve may not pick up a config it
// ignored on boot).
package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theboringhumane/theboringfloor/internal/charter"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// charterRelPath is the instructions entry as it appears in
// .opencode/opencode.json. Verified against opencode serve 1.18.19: the
// consumer resolves relative entries against the PROJECT directory
// (globUp from the served dir toward the worktree root), not the config
// file's directory — hence the explicit ".opencode/" prefix.
const charterRelPath = "./.opencode/oikonomos.md"

// charterAcceptedPaths are all spellings EnsureCharter treats as "already
// wired" — a member hand-writing any of these must not get a duplicate.
var charterAcceptedPaths = []string{
	charterRelPath,
	".opencode/oikonomos.md",
	"./oikonomos.md",
	"oikonomos.md",
}

// EnsureCharter wires the bundled oikonomos manager charter into dir so a
// spawned opencode serve reads it. changed is true when any byte was
// written (callers that already spawned a serve should restart it; see the
// header comment). notes are status-line text describing what happened.
// The pass is idempotent: a follow-up run with everything already in place
// returns changed=false and writes nothing. THEFLOOR_NO_AUTOCHARTER=1 opts
// out entirely (changed=false, note explaining).
func EnsureCharter(dir string) (changed bool, notes []string) {
	if config.EnvBool("NO_AUTOCHARTER") {
		return false, []string{"[theboringfloor] manager charter: disabled (THEFLOOR_NO_AUTOCHARTER)"}
	}

	ocDir := filepath.Join(dir, ".opencode")
	chartPath := filepath.Join(ocDir, "oikonomos.md")
	cfgPath := filepath.Join(ocDir, "opencode.json")

	// 1. The charter markdown: write byte-exact, skip when identical.
	want := []byte(charter.Text)
	if got, err := os.ReadFile(chartPath); err != nil || !bytes.Equal(got, want) {
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			return changed, append(notes, "[theboringfloor] manager charter: failed (mkdir "+ocDir+": "+err.Error()+")")
		}
		if err := os.WriteFile(chartPath, want, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] manager charter: failed (write "+chartPath+": "+err.Error()+")")
		}
		changed = true
	}

	// 2. The config merge: read opencode.json when present, ensure the
	//    instructions array contains the charter entry exactly once,
	//    preserving every other key verbatim (map[string]any round-trip,
	//    2-space indent; see mergeInstruction for field preservation).
	cfgRaw, err := os.ReadFile(cfgPath)
	if err == nil {
		merged, mergeChanged, mergeErr := mergeInstruction(cfgRaw, charterRelPath, charterAcceptedPaths)
		if mergeErr != nil {
			return changed, append(notes, "[theboringfloor] manager charter: failed (merge "+cfgPath+": "+mergeErr.Error()+")")
		}
		if mergeChanged {
			if err := os.WriteFile(cfgPath, merged, 0o644); err != nil {
				return changed, append(notes, "[theboringfloor] manager charter: failed (write "+cfgPath+": "+err.Error()+")")
			}
			changed = true
		}
	} else if os.IsNotExist(err) {
		fresh := []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"instructions\": [\n    \"" + charterRelPath + "\"\n  ]\n}\n")
		if err := os.WriteFile(cfgPath, fresh, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] manager charter: failed (write "+cfgPath+": "+err.Error()+")")
		}
		changed = true
	} else {
		return changed, append(notes, "[theboringfloor] manager charter: failed (read "+cfgPath+": "+err.Error()+")")
	}

	// 	3. The MCP prompt attachment (charter_mcp.go): list the serve's
	// 	   configured MCP servers so the boss — and through her briefs the
	// 	   developers — know which MCP tools exist. Its notes ride AHEAD of
	// 	   the charter's final summary line (probes pattern-match the tail).
	mcpChanged, mcpNotes := ensureMCPAttachment(dir)
	notes = append(notes, mcpNotes...)
	if mcpChanged {
		changed = true
	}

	// 4. The office ledger: every served project learns about the office's
	//    completed-work memory. The step seeds .opencode/office-ledger.md
	//    when absent and merges its instructions entry beside the charter +
	//    MCP attachment, reusing the same field-preserving mergeInstruction.
	//    Notes ride AHEAD of the final summary line exactly like step 3's.
	ledgerChanged, ledgerNotes := ensureLedgerAttachment(dir)
	notes = append(notes, ledgerNotes...)
	if ledgerChanged {
		changed = true
	}

	if changed {
		return true, append(notes, "[theboringfloor] manager charter: wired (.opencode/oikonomos.md)")
	}
	return false, append(notes, "[theboringfloor] manager charter: already wired (.opencode/oikonomos.md)")
}

// charterOutcome is the two things Start needs out of a charter pass.
type charterOutcome struct {
	changed bool
	notes   []string
}

// emitCharterNotes runs EnsureCharter for dir and surfaces every note as a
// status event on emit. Called exactly once from liveBackend.Start, BEFORE
// server resolution: a spawned serve then reads a config that already has
// the charter entry. The demo backend never calls this.
func emitCharterNotes(emit func(state.Event), dir string) charterOutcome {
	changed, notes := EnsureCharter(dir)
	out := charterOutcome{changed: changed, notes: notes}
	for _, n := range notes {
		emit(state.Event{Kind: state.EvStatus, Text: n})
	}
	return out
}

// mergeInstruction is the pure half of the config merge for ONE
// instructions entry: cfg is an existing opencode.json. It returns cfg
// unchanged (changed=false) when the instructions array already names the
// entry by any accepted spelling. Otherwise it appends the canonical
// relPath entry and re-marshals with 2-space indent. Every foreign field
// survives (map[string]any round-trip); a non-object instruction entry (a
// number, say) is left alone and the entry is appended alongside it. A
// non-object top level, an explicit JSON null instructions, or an
// instructions value that is not an array all fail closed: the member's
// hand-rolled shape is never clobbered.
func mergeInstruction(cfg []byte, relPath string, accepted []string) (merged []byte, changed bool, err error) {
	var doc map[string]any
	if err := json.Unmarshal(cfg, &doc); err != nil {
		return nil, false, fmt.Errorf("unparseable json: %w", err)
	}

	rawInstr, has := doc["instructions"]
	if has {
		if rawInstr == nil {
			return nil, false, fmt.Errorf("instructions is null — refusing to rewrite a hand-shaped config")
		}
		arr, ok := rawInstr.([]any)
		if !ok {
			return nil, false, fmt.Errorf("instructions is %T, not an array — refusing to rewrite a hand-shaped config", rawInstr)
		}
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				continue
			}
			for _, spelling := range accepted {
				if s == spelling {
					return cfg, false, nil // already wired
				}
			}
		}
		arr = append(arr, relPath)
		doc["instructions"] = arr
	} else {
		doc["instructions"] = []any{relPath}
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}

// ---------------------------------------------------------------- office ledger

// ledgerRelPath is the office ledger's instructions entry spelling,
// mirroring charterRelPath's verified ".opencode/"-explicit shape: opencode
// resolves relative instructions against the PROJECT directory, not the
// config file's own directory.
const ledgerRelPath = "./.opencode/office-ledger.md"

// ledgerAcceptedPaths are all spellings the pass treats as "already wired"
// — a member hand-writing any of these must not get a duplicate (mirrors
// charterAcceptedPaths / mcpAttachmentAcceptedPaths).
var ledgerAcceptedPaths = []string{
	ledgerRelPath,
	".opencode/office-ledger.md",
	"./office-ledger.md",
	"office-ledger.md",
}

// ledgerEntriesMarker is the byte anchor beneath which the sibling ledger
// writer (internal/backend/ledger.go's territory) inserts each completed
// dispatch, NEWEST-FIRST, as "### YYYY-MM-DD · <title> — <worker> ·
// <verdict>" entries. The charter pass owns seeding + instructions wiring;
// the app owns appending. The marker is invisible markdown (ASCII comment)
// and rides the seed so the writer never has to guess where the header ends.
const ledgerEntriesMarker = "<!-- ledger:entries -->"

// renderLedgerSeed is the ledger's empty-state markdown: the header that
// teaches the boss WHAT the file is (the office's completed-work memory —
// the exact memory paragraph, frozen copy) and WHERE entries land. Like
// every prompt-visible artifact here it is LF-only, ASCII, deterministic,
// and ends on a newline.
func renderLedgerSeed() []byte {
	var b strings.Builder
	b.WriteString("# Office Ledger — completed work\n")
	b.WriteString("\n")
	b.WriteString("The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.\n")
	b.WriteString("\n")
	b.WriteString("Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,\n")
	b.WriteString("recorded by the office app itself after each verified return. This file is\n")
	b.WriteString("append-only state: the charter pass seeds it when absent and never rewrites it.\n")
	b.WriteString("\n")
	b.WriteString(ledgerEntriesMarker + "\n")
	return []byte(b.String())
}

// ensureLedgerAttachment is the chartered step-4 pass for dir (see
// charter.go's header): the charter-merge seam for the office ledger. It
// seeds <dir>/.opencode/office-ledger.md with renderLedgerSeed when the
// file is ABSENT — and ONLY then: unlike the MCP attachment (a
// deterministic render refreshed byte-exact), the ledger is append-only
// state accumulated by the app, so an existing file is never rewritten.
// It then merges "./.opencode/office-ledger.md" into instructions beside
// the charter + MCP attachment, reusing the same field-preserving
// mergeInstruction, so a served project boots ledger-indoctrinated: the
// boss consults completed work BEFORE re-dispatching. Idempotent like the
// sibling steps (second run: changed=false, no bytes written); notes ride
// AHEAD of the charter's final summary line.
func ensureLedgerAttachment(dir string) (changed bool, notes []string) {
	ocDir := filepath.Join(dir, ".opencode")
	ledgerPath := filepath.Join(ocDir, "office-ledger.md")
	cfgPath := filepath.Join(ocDir, "opencode.json")

	// 1. The ledger markdown: seed when absent, NEVER rewrite an existing
	//    file (member edits and app-recorded entries are sacred).
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			return changed, append(notes, "[theboringfloor] office ledger: failed (mkdir "+ocDir+": "+err.Error()+")")
		}
		if err := os.WriteFile(ledgerPath, renderLedgerSeed(), 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] office ledger: failed (write "+ledgerPath+": "+err.Error()+")")
		}
		changed = true
	} else if err != nil {
		return changed, append(notes, "[theboringfloor] office ledger: failed (stat "+ledgerPath+": "+err.Error()+")")
	}

	// 2. The instructions entry rides beside the charter's and the MCP
	//    attachment's through the same field-preserving merge. The
	//    charter's step 2 guarantees the config exists by now; the
	//    NotExist branch is defensive only.
	cfgRaw, err := os.ReadFile(cfgPath)
	if err == nil {
		merged, mergeChanged, mergeErr := mergeInstruction(cfgRaw, ledgerRelPath, ledgerAcceptedPaths)
		if mergeErr != nil {
			return changed, append(notes, "[theboringfloor] office ledger: failed (merge "+cfgPath+": "+mergeErr.Error()+")")
		}
		if mergeChanged {
			if err := os.WriteFile(cfgPath, merged, 0o644); err != nil {
				return changed, append(notes, "[theboringfloor] office ledger: failed (write "+cfgPath+": "+err.Error()+")")
			}
			changed = true
		}
	} else if os.IsNotExist(err) {
		fresh := []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"instructions\": [\n    \"" + ledgerRelPath + "\"\n  ]\n}\n")
		if err := os.WriteFile(cfgPath, fresh, 0o644); err != nil {
			return changed, append(notes, "[theboringfloor] office ledger: failed (write "+cfgPath+": "+err.Error()+")")
		}
		changed = true
	} else {
		return changed, append(notes, "[theboringfloor] office ledger: failed (read "+cfgPath+": "+err.Error()+")")
	}

	if changed {
		return true, append(notes, "[theboringfloor] office ledger: wired (.opencode/office-ledger.md)")
	}
	return false, append(notes, "[theboringfloor] office ledger: already wired (.opencode/office-ledger.md)")
}

// bypassConfigContent is passed to an owned OpenCode process only. OpenCode's
// SDK uses OPENCODE_CONFIG_CONTENT for exactly this config-injection seam.
// The wildcard form is accepted by OpenCode 1.18.21's PermissionConfig schema
// and has the documented global permission-rule meaning.
const bypassConfigContent = `{"permission":{"*":"allow"}}`
