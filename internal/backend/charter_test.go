package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/charter"
)

// charterProbePhrases is the live-probe subset contract: the charter must
// name the manager role and the 3-dispatch minimum.
func charterProbePhrases() []string {
	return []string{"manager", "oikonomos", "MINIMUM 3", "Proof-of-work"}
}

func TestCharterAssetSelfCheck(t *testing.T) {
	if !charter.ContainsPhrases(charterProbePhrases()) {
		t.Fatalf("embedded charter failed its own subset probe: missing one of %v", charterProbePhrases())
	}
	for _, want := range []string{
		"MINIMUM 3 sub-agents",
		"built-in office browser for every URL",
		"open-browser",
		"browser-screenshot",
		"browser-snapshot",
		"browser-action",
		"only with member permission",
		"Never launch Chrome, Chromium, Playwright, Puppeteer",
	} {
		if !strings.Contains(charter.String(), want) {
			t.Fatalf("embedded charter does not contain %q", want)
		}
	}
	if strings.Contains(charter.String(), "\r") {
		t.Fatal("embedded charter contains CR bytes — expected LF-only")
	}
}

// pinGlobalConfig exiles the opencode global config dir so the charter
// pass's MCP discovery (charter_mcp.go) sees NO global servers: without
// it, tests on a machine with real MCP servers configured (any
// developer's ~/.config/opencode) would wire the MCP attachment into the
// scratch dir and flip the changed/notes/file expectations under them.
// XDG is honored first on every platform the office runs on; HOME is the
// fallback root. Hermetic > realistic here.
func pinGlobalConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

func TestEnsureCharterCreatesFresh(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	changed, notes := EnsureCharter(dir)
	if !changed {
		t.Fatal("fresh dir: changed=false, want true")
	}
	if !containsNote(notes, "manager charter: wired") {
		t.Fatalf("notes missing the wired line: %v", notes)
	}
	chartRaw, err := os.ReadFile(filepath.Join(dir, ".opencode", "oikonomos.md"))
	if err != nil {
		t.Fatalf("chart file: %v", err)
	}
	if string(chartRaw) != charter.Text {
		t.Fatal("chart file is not byte-exact the embedded charter")
	}
	for _, want := range []string{"built-in office browser for every URL", "open-browser", "browser-screenshot", "browser-snapshot", "browser-action"} {
		if !strings.Contains(string(chartRaw), want) {
			t.Fatalf("OpenCode-delivered charter missing %q", want)
		}
	}
	cfgRaw, err := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
	if err != nil {
		t.Fatalf("config file: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		t.Fatalf("config is not json: %v", err)
	}
	if cfg["$schema"] != "https://opencode.ai/config.json" {
		t.Fatalf("fresh config's $schema: %v", cfg["$schema"])
	}
	assertHasCharterInstructions(t, cfg)
}

func TestEnsureCharterIdempotentByteIdentical(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	if _, notes := EnsureCharter(dir); !containsNote(notes, "manager charter: wired") {
		t.Fatalf("first run: want wired note, got %v", notes)
	}
	chart1 := mustRead(t, filepath.Join(dir, ".opencode", "oikonomos.md"))
	cfg1 := mustRead(t, filepath.Join(dir, ".opencode", "opencode.json"))
	ledger1 := mustRead(t, filepath.Join(dir, ".opencode", "office-ledger.md"))

	changed, notes := EnsureCharter(dir)
	if changed {
		t.Fatal("second run: changed=true, want false (byte-idempotency)")
	}
	if !containsNote(notes, "already") {
		t.Fatalf("second run notes missing an already-wired line: %v", notes)
	}
	if got := mustRead(t, filepath.Join(dir, ".opencode", "oikonomos.md")); got != chart1 {
		t.Fatal("second run rewrote oikonomos.md")
	}
	if got := mustRead(t, filepath.Join(dir, ".opencode", "opencode.json")); got != cfg1 {
		t.Fatal("second run rewrote opencode.json")
	}
	if got := mustRead(t, filepath.Join(dir, ".opencode", "office-ledger.md")); got != ledger1 {
		t.Fatal("second run rewrote office-ledger.md")
	}
}

func TestEnsureCharterMergePreservesForeignFields(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prior := `{
    "theme": "nord",
    "model": "anthropic/claude-haiku-4-5",
    "instructions": ["docs/CODE_STYLE.md"],
    "nested": {"deep": {"list": [1, 2, 3]}, "flag": true}
}
`
	cfgPath := filepath.Join(ocDir, "opencode.json")
	if err := os.WriteFile(cfgPath, []byte(prior), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, notes := EnsureCharter(dir)
	if !changed {
		t.Fatal("existing config without the entry: changed=false, want true")
	}
	if !containsNote(notes, "manager charter: wired") {
		t.Fatalf("notes: %v", notes)
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(mustRead(t, cfgPath)), &cfg); err != nil {
		t.Fatalf("merged config is not json: %v", err)
	}
	// Foreign fields survive, values intact.
	if cfg["theme"] != "nord" {
		t.Fatalf("theme dropped/changed: %v", cfg["theme"])
	}
	if cfg["model"] != "anthropic/claude-haiku-4-5" {
		t.Fatalf("model dropped/changed: %v", cfg["model"])
	}
	nested, ok := cfg["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested dropped: %v", cfg["nested"])
	}
	if nested["deep"].(map[string]any)["list"].([]any)[2].(float64) != 3 {
		t.Fatalf("nested.deep.list mutated: %v", nested)
	}
	if nested["flag"] != true {
		t.Fatalf("nested.flag mutated: %v", nested)
	}
	// The pre-existing instruction entry survives AND the charter lands once.
	instr := cfg["instructions"].([]any)
	if len(instr) != 3 || instr[0] != "docs/CODE_STYLE.md" || instr[1] != charterRelPath || instr[2] != ledgerRelPath {
		t.Fatalf("instructions: %v", instr)
	}

	// A second run must not duplicate the entries.
	if changed, _ := EnsureCharter(dir); changed {
		t.Fatal("second run re-wrote the merged config")
	}
	var cfg2 map[string]any
	if err := json.Unmarshal([]byte(mustRead(t, cfgPath)), &cfg2); err != nil {
		t.Fatal(err)
	}
	if len(cfg2["instructions"].([]any)) != 3 {
		t.Fatalf("second run duplicated a charter entry: %v", cfg2["instructions"])
	}
}

func TestEnsureCharterAcceptedSpellings(t *testing.T) {
	pinGlobalConfig(t)
	for _, spelling := range charterAcceptedPaths {
		dir := t.TempDir()
		ocDir := filepath.Join(dir, ".opencode")
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(ocDir, "opencode.json")
		// The ledger entry rides pre-wired so the spelling check stays
		// isolated to the CHARTER entry's accepted spellings.
		body := `{"instructions":["` + spelling + `","` + ledgerRelPath + `"],"x":1}`
		if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		// First run may rewrite the chart markdown (EnsureCharter creates it
		// if missing); pre-create it so the spelling check is isolated to
		// the config merge. Same for the ledger file: a charter pass seeds
		// it when absent, so pre-create it here.
		if err := os.WriteFile(filepath.Join(ocDir, "oikonomos.md"), []byte(charter.Text), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ocDir, "office-ledger.md"), renderLedgerSeed(), 0o644); err != nil {
			t.Fatal(err)
		}
		changed, _ := EnsureCharter(dir)
		if changed {
			t.Fatalf("spelling %q: changed=true, want false (accepted as wired)", spelling)
		}
		if got := mustRead(t, cfgPath); got != body {
			t.Fatalf("spelling %q: config rewritten: %s", spelling, got)
		}
	}
}

func TestEnsureCharterEnvOptOut(t *testing.T) {
	pinGlobalConfig(t)
	t.Setenv("THEBORINGOFFICE_NO_AUTOCHARTER", "1")
	dir := t.TempDir()
	changed, notes := EnsureCharter(dir)
	if changed {
		t.Fatal("opt-out: changed=true, want false")
	}
	if !containsNote(notes, "disabled (THEBORINGOFFICE_NO_AUTOCHARTER)") {
		t.Fatalf("opt-out notes: %v", notes)
	}
	// Nothing written at all.
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("opt-out wrote files: %v %v", entries, err)
	}
}

func TestEnsureCharterRefusesNullOrNonArrayInstructions(t *testing.T) {
	pinGlobalConfig(t)
	for _, body := range []string{
		`{"instructions":null,"x":1}`,
		`{"instructions":"docs/*.md","x":1}`,
	} {
		dir := t.TempDir()
		ocDir := filepath.Join(dir, ".opencode")
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(ocDir, "opencode.json")
		if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, notes := EnsureCharter(dir)
		if !containsNote(notes, "manager charter: failed") {
			t.Fatalf("%s: want a failed note, got %v", body, notes)
		}
		if got := mustRead(t, cfgPath); got != body {
			t.Fatalf("%s: config clobbered: %s", body, got)
		}
	}
}

func TestEnsureCharterRefusesBrokenJson(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(ocDir, "opencode.json")
	broken := `{"instructions": [`
	if err := os.WriteFile(cfgPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	_, notes := EnsureCharter(dir)
	if !containsNote(notes, "manager charter: failed") {
		t.Fatalf("broken json: want a failed note, got %v", notes)
	}
	if got := mustRead(t, cfgPath); got != broken {
		t.Fatalf("broken json clobbered: %s", got)
	}
}

func TestEnsureCharterNeverTouchesAgentsMd(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	agents := "# member repo rules\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, notes := EnsureCharter(dir); !containsNote(notes, "manager charter: wired") {
		t.Fatalf("notes: %v", notes)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md")); err != nil || string(got) != agents {
		t.Fatalf("AGENTS.md touched: %q %v", got, err)
	}
}

// ledgerMemoryParagraph pins the FROZEN memory copy the charter body and the
// seeded ledger file both carry — sense edits land here deliberately, never
// by drift.
const ledgerMemoryParagraph = "The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done."

// TestEnsureCharterWiresLedgerByteStable is the ledger-step contract in one
// pass over a scratch dir, mirroring the MCP attachment's wire+idempotent
// fixture: run EnsureCharter twice, first run wires (charter + ledger —
// global config pinned so MCP is a pinned no-op), second run changes
// NOTHING (changed=false ⇒ Start's respawn-on-change never fires a second
// time; every file byte-identical; notes only report "already"/no-op).
func TestEnsureCharterWiresLedgerByteStable(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")

	changed, notes := EnsureCharter(dir)
	if !changed {
		t.Fatal("fresh dir: changed=false, want true")
	}
	// Builder-response notes mention the ledger merge (existing note-style).
	if !containsNote(notes, "office ledger: wired (.opencode/office-ledger.md)") {
		t.Fatalf("notes missing the ledger wired line: %v", notes)
	}
	// Step order is stable: MCP step notes ride ahead of the ledger step's,
	// which ride ahead of the charter's final summary line.
	mcpIdx, ledgerIdx, summaryIdx := -1, -1, -1
	for i, n := range notes {
		switch {
		case strings.Contains(n, "mcp prompt attachment:"):
			mcpIdx = i
		case strings.Contains(n, "office ledger:"):
			ledgerIdx = i
		case strings.Contains(n, "manager charter: wired"):
			summaryIdx = i
		}
	}
	if !(mcpIdx >= 0 && ledgerIdx > mcpIdx && summaryIdx > ledgerIdx) {
		t.Fatalf("note order must stay mcp < ledger < charter summary: %v", notes)
	}

	// The ledger FILE exists, seeded with the frozen memory paragraph and
	// the entries marker, LF-only, newline-terminated.
	ledgerPath := filepath.Join(ocDir, "office-ledger.md")
	ledger1 := mustRead(t, ledgerPath)
	if !strings.Contains(ledger1, ledgerMemoryParagraph) {
		t.Fatalf("seeded ledger missing the memory paragraph:\n%s", ledger1)
	}
	if !strings.Contains(ledger1, ledgerEntriesMarker) {
		t.Fatalf("seeded ledger missing the entries marker:\n%s", ledger1)
	}
	if strings.Contains(ledger1, "\r") || !strings.HasSuffix(ledger1, "\n") {
		t.Fatal("seeded ledger must be LF-only and end on a newline")
	}

	// oikonomos.md (byte-exact the embedded charter) teaches ledger
	// consultation AFTER the closing hook block of the pre-existing charter.
	chart1 := mustRead(t, filepath.Join(ocDir, "oikonomos.md"))
	if !strings.Contains(chart1, ledgerMemoryParagraph) {
		t.Fatalf("oikonomos.md missing the memory paragraph")
	}
	hookIdx := strings.Index(chart1, "## After the turn")
	memIdx := strings.Index(chart1, ledgerMemoryParagraph)
	if hookIdx < 0 || memIdx < hookIdx {
		t.Fatal("the memory paragraph must land after the existing charter body")
	}
	if strings.Contains(chart1, "\r") {
		t.Fatal("oikonomos.md must stay LF-only")
	}

	// instructions carries the ledger path exactly once, after the charter's.
	cfgPath := filepath.Join(ocDir, "opencode.json")
	cfg1 := mustRead(t, cfgPath)
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfg1), &cfg); err != nil {
		t.Fatal(err)
	}
	instr := cfg["instructions"].([]any)
	if len(instr) != 2 || instr[0] != charterRelPath || instr[1] != ledgerRelPath {
		t.Fatalf("instructions: %v", instr)
	}

	// Second run: changed=false (=> no serve respawn — Start respawns ONLY on
	// changed=true) and zero bytes anywhere; notes say "already" only.
	changed2, notes2 := EnsureCharter(dir)
	if changed2 {
		t.Fatalf("second run: changed=true, want false — no respawn allowed (notes %v)", notes2)
	}
	for _, fresh := range []string{"office ledger: wired", "manager charter: wired", "mcp prompt attachment: wired"} {
		if containsNote(notes2, fresh) {
			t.Fatalf("second run must announce nothing freshly wired, got %q in %v", fresh, notes2)
		}
	}
	if !containsNote(notes2, "office ledger: already wired") {
		t.Fatalf("second run notes missing the ledger already-wired line: %v", notes2)
	}
	if got := mustRead(t, ledgerPath); got != ledger1 {
		t.Fatal("second run rewrote office-ledger.md")
	}
	if got := mustRead(t, cfgPath); got != cfg1 {
		t.Fatal("second run rewrote opencode.json")
	}
	if got := mustRead(t, filepath.Join(ocDir, "oikonomos.md")); got != chart1 {
		t.Fatal("second run rewrote oikonomos.md")
	}
}

// TestEnsureLedgerAttachmentAcceptsHandWiredSpellings mirrors the MCP
// attachment's spelling fixture: a member hand-writing ANY accepted spelling
// of the ledger entry earns neither a duplicate nor a rewritten config.
func TestEnsureLedgerAttachmentAcceptsHandWiredSpellings(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()

	// Run 0: the pass itself wires charter + ledger (MCP no-op: pinned
	// global config has zero servers).
	if changed, _ := EnsureCharter(dir); !changed {
		t.Fatal("run 0: want the ledger wired")
	}
	for _, spelling := range ledgerAcceptedPaths {
		ocDir := filepath.Join(dir, ".opencode")
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

// TestEnsureLedgerAttachmentNeverRewritesExisting pins the seam contract the
// sibling ledger writer relies on: the charter pass seeds office-ledger.md
// when ABSENT and treats an existing file as sacred (member edits and
// app-recorded entries survive every pass).
func TestEnsureLedgerAttachmentNeverRewritesExisting(t *testing.T) {
	pinGlobalConfig(t)
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mine := "# Office Ledger — completed work\n\n### 2026-08-26 · shipped the thing — dev#7 · done\n"
	ledgerPath := filepath.Join(ocDir, "office-ledger.md")
	if err := os.WriteFile(ledgerPath, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := EnsureCharter(dir); !changed {
		t.Fatal("first run: the instructions merge alone is still a change")
	}
	if got := mustRead(t, ledgerPath); got != mine {
		t.Fatalf("existing ledger bytes rewritten:\n%s", got)
	}
	if changed, _ := EnsureCharter(dir); changed {
		t.Fatal("second run: changed=true, want false")
	}
	if got := mustRead(t, ledgerPath); got != mine {
		t.Fatal("second pass rewrote the existing ledger")
	}
}

// TestEnsureLedgerAttachmentFailClosed: the merge step refuses hand-shaped
// configs exactly like the charter's own merge — a null or non-array
// instructions never gets clobbered, and a broken config is left alone.
func TestEnsureLedgerAttachmentFailClosed(t *testing.T) {
	pinGlobalConfig(t)
	for _, body := range []string{
		`{"instructions":null,"x":1}`,
		`{"instructions":"docs/*.md","x":1}`,
		`{"instructions": [`,
	} {
		dir := t.TempDir()
		ocDir := filepath.Join(dir, ".opencode")
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			t.Fatal(err)
		}
		// Pre-existing ledger file: isolates the failure to the merge arm.
		if err := os.WriteFile(filepath.Join(ocDir, "office-ledger.md"), renderLedgerSeed(), 0o644); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(ocDir, "opencode.json")
		if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		changed, notes := ensureLedgerAttachment(dir)
		if changed {
			t.Fatalf("%s: changed=true on a refused merge", body)
		}
		if !containsNote(notes, "office ledger: failed") {
			t.Fatalf("%s: want a failed note, got %v", body, notes)
		}
		if got := mustRead(t, cfgPath); got != body {
			t.Fatalf("%s: config clobbered: %s", body, got)
		}
	}
}

func containsNote(notes []string, sub string) bool {
	for _, n := range notes {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}

func assertHasCharterInstructions(t *testing.T, cfg map[string]any) {
	t.Helper()
	instr, ok := cfg["instructions"].([]any)
	if !ok {
		t.Fatalf("instructions missing or not an array: %v", cfg["instructions"])
	}
	found := 0
	for _, x := range instr {
		if x == charterRelPath {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("instructions %v: want exactly one %q, got %d", instr, charterRelPath, found)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
