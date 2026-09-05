// charter_claude_test.go — the disk-level contract of EnsureClaudeCharter
// (charter_claude.go), the Claude Code counterpart of charter_test.go's
// opencode-pass pins: create-if-absent (exact bytes), no-op-if-referenced
// (any spelling), append-if-missing (member prefix byte-identical),
// idempotent double-run, dir-missing error path, the env opt-out, and
// the payload seed discipline (seeded when absent, never rewritten).
// The end-to-end proof that the wired file actually reaches the model is
// the gated TestClaudeLiveCharterReaches in claude_live_test.go.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/charter"
)

// wantClaudeCharterFresh is the pinned byte-exact expectation for the
// created CLAUDE.md (kept as a literal so a drift in the renderer fails
// loud here, not silently in members' projects).
const wantClaudeCharterFresh = "<!-- theboringfloor charter -->\n" +
	"This project is served by theboringfloor: the oikonomos manager protocol imported below is the operating charter.\n" +
	"@.opencode/oikonomos.md\n"

func TestEnsureClaudeCharterCreatesFresh(t *testing.T) {
	dir := t.TempDir()
	changed, notes := EnsureClaudeCharter(dir)
	if !changed {
		t.Fatalf("fresh dir: changed=false, want true (notes %v)", notes)
	}
	if !containsNote(notes, "claude charter: wired (CLAUDE.md created") {
		t.Fatalf("notes missing the created line: %v", notes)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md: %v", err)
	}
	if string(raw) != wantClaudeCharterFresh {
		t.Fatalf("created CLAUDE.md is not the pinned bytes:\n--- got ---\n%q\n--- want ---\n%q", raw, wantClaudeCharterFresh)
	}
	if strings.Contains(string(raw), "\r") {
		t.Fatal("created CLAUDE.md contains CR bytes — expected LF-only")
	}

	// The import target must exist or the import dangles: seeded
	// byte-exact from the embedded charter.
	payload, err := os.ReadFile(filepath.Join(dir, ".opencode", "oikonomos.md"))
	if err != nil {
		t.Fatalf("payload .opencode/oikonomos.md: %v", err)
	}
	if string(payload) != charter.Text {
		t.Fatal("seeded payload is not byte-exact the embedded charter")
	}
	for _, want := range []string{"built-in office browser for every URL", "open-browser", "browser-screenshot", "browser-snapshot", "browser-action"} {
		if !strings.Contains(string(payload), want) {
			t.Fatalf("Claude-delivered charter missing %q", want)
		}
	}
	if !containsNote(notes, "claude charter: seeded .opencode/oikonomos.md") {
		t.Fatalf("notes missing the payload-seed line: %v", notes)
	}
}

func TestEnsureClaudeCharterNoopWhenReferenced(t *testing.T) {
	// "Any form" — an import, a differently-spelled import, a prose
	// mention: each counts as wired, no bytes move.
	for _, body := range []string{
		"# Team rules\n\n@.opencode/oikonomos.md\n",
		"# Team rules\n\n@./.opencode/oikonomos.md\n",
		"# Team rules\n\nSee oikonomos.md for the office protocol.\n",
	} {
		t.Run(strings.TrimSpace(strings.SplitN(body, "\n", 3)[2]), func(t *testing.T) {
			dir := t.TempDir()
			// Pre-seed the payload FRESH (the embedded bytes) so the ONLY
			// variable under test is the CLAUDE.md no-op — a drifted
			// payload legitimately refreshes under office-ownership.
			if err := os.MkdirAll(filepath.Join(dir, ".opencode"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".opencode", "oikonomos.md"), []byte(charter.Text), 0o644); err != nil {
				t.Fatal(err)
			}
			mdPath := filepath.Join(dir, "CLAUDE.md")
			if err := os.WriteFile(mdPath, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			changed, notes := EnsureClaudeCharter(dir)
			if changed {
				t.Fatalf("referenced CLAUDE.md: changed=true, want false (notes %v)", notes)
			}
			if !containsNote(notes, "claude charter: already wired (CLAUDE.md)") {
				t.Fatalf("notes missing the already-wired line: %v", notes)
			}
			got, err := os.ReadFile(mdPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != body {
				t.Fatalf("referenced CLAUDE.md was rewritten:\n--- got ---\n%q\n--- want (untouched) ---\n%q", got, body)
			}
		})
	}
}

func TestEnsureClaudeCharterAppendsWhenMissing(t *testing.T) {
	// Two member shapes: a normal LF-terminated file and one whose tail
	// lacks the newline — the append must normalize to exactly one blank
	// line of separation in both.
	for _, tc := range []struct {
		name string
		body string
	}{
		{"lf-terminated", "# Team rules\n\nRun tests before pushing.\n"},
		{"no-trailing-newline", "# Team rules\n\nRun tests before pushing."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mdPath := filepath.Join(dir, "CLAUDE.md")
			if err := os.WriteFile(mdPath, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			changed, notes := EnsureClaudeCharter(dir)
			if !changed {
				t.Fatalf("unreferenced CLAUDE.md: changed=false, want true (notes %v)", notes)
			}
			if !containsNote(notes, "claude charter: wired (import block appended to CLAUDE.md)") {
				t.Fatalf("notes missing the appended line: %v", notes)
			}
			got, err := os.ReadFile(mdPath)
			if err != nil {
				t.Fatal(err)
			}
			// The member prefix is byte-identical…
			if !strings.HasPrefix(string(got), tc.body) {
				t.Fatalf("member content clobbered:\n--- got ---\n%q\n--- want prefix ---\n%q", got, tc.body)
			}
			// …and the marked block rides below, exactly once.
			wantTail := "\n\n<!-- theboringfloor charter -->\n@.opencode/oikonomos.md\n<!-- /theboringfloor charter -->\n"
			if !strings.HasSuffix(string(got), wantTail) {
				t.Fatalf("appended file does not end on the marked block:\n--- got ---\n%q\n--- want tail ---\n%q", got, wantTail)
			}
			if n := strings.Count(string(got), "<!-- theboringfloor charter -->"); n != 1 {
				t.Fatalf("begin marker appears %d times, want 1", n)
			}
			if n := strings.Count(string(got), claudeCharterImportLine); n != 1 {
				t.Fatalf("import line appears %d times, want 1", n)
			}
		})
	}
}

func TestEnsureClaudeCharterIdempotent(t *testing.T) {
	// Create path: run twice, second run writes nothing and every byte
	// (memory file + payload) is identical.
	dir := t.TempDir()
	if changed, _ := EnsureClaudeCharter(dir); !changed {
		t.Fatal("first run: changed=false, want true")
	}
	mdPath := filepath.Join(dir, "CLAUDE.md")
	payloadPath := filepath.Join(dir, ".opencode", "oikonomos.md")
	md1, _ := os.ReadFile(mdPath)
	payload1, _ := os.ReadFile(payloadPath)
	changed, notes := EnsureClaudeCharter(dir)
	if changed {
		t.Fatalf("second run: changed=true, want false (notes %v)", notes)
	}
	if !containsNote(notes, "claude charter: already wired (CLAUDE.md)") {
		t.Fatalf("second run notes missing already-wired: %v", notes)
	}
	md2, _ := os.ReadFile(mdPath)
	payload2, _ := os.ReadFile(payloadPath)
	if string(md1) != string(md2) || string(payload1) != string(payload2) {
		t.Fatal("second run moved bytes — the pass must be byte-identical")
	}

	// Append path: the appended block must never land twice.
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, "CLAUDE.md"), []byte("# Mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := EnsureClaudeCharter(dir2); !changed {
		t.Fatal("append run 1: changed=false, want true")
	}
	got1, _ := os.ReadFile(filepath.Join(dir2, "CLAUDE.md"))
	changed2, _ := EnsureClaudeCharter(dir2)
	if changed2 {
		t.Fatal("append run 2: changed=true — the block was appended twice")
	}
	got2, _ := os.ReadFile(filepath.Join(dir2, "CLAUDE.md"))
	if string(got1) != string(got2) {
		t.Fatal("append run 2 moved bytes")
	}
	if n := strings.Count(string(got2), "<!-- theboringfloor charter -->"); n != 1 {
		t.Fatalf("begin marker appears %d times after two runs, want 1", n)
	}
}

func TestEnsureClaudeCharterMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-project")
	changed, notes := EnsureClaudeCharter(missing)
	if changed {
		t.Fatalf("missing dir: changed=true, want false (notes %v)", notes)
	}
	if !containsNote(notes, "claude charter: failed") || !containsNote(notes, "dir missing") {
		t.Fatalf("notes missing the dir-missing failure: %v", notes)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("the pass must never create the member's project root: stat err=%v", err)
	}
}

func TestEnsureClaudeCharterEnvOptOut(t *testing.T) {
	t.Setenv("THEFLOOR_NO_AUTOCHARTER", "1")
	dir := t.TempDir()
	changed, notes := EnsureClaudeCharter(dir)
	if changed {
		t.Fatalf("opted-out: changed=true, want false (notes %v)", notes)
	}
	if !containsNote(notes, "claude charter: disabled (THEFLOOR_NO_AUTOCHARTER)") {
		t.Fatalf("notes missing the disabled line: %v", notes)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatal("opted-out pass wrote CLAUDE.md")
	}
}

func TestEnsureClaudeCharterPayloadRefreshed(t *testing.T) {
	// The payload is office-owned: a stale or hand-edited file refreshes
	// to the embedded charter bytes (byte-exact freshness for claude-only
	// offices), while a fresh file costs zero writes and zero notes.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	payloadPath := filepath.Join(dir, ".opencode", "oikonomos.md")
	stale := "# my own charter notes\n"
	if err := os.WriteFile(payloadPath, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, notes := EnsureClaudeCharter(dir)
	if !changed {
		t.Fatal("a drifted payload must refresh")
	}
	if !containsNote(notes, "refreshed .opencode/oikonomos.md") {
		t.Fatalf("the refresh note must name the drift: %v", notes)
	}
	got, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != charter.Text {
		t.Fatalf("the payload must refresh to the embedded charter (%d bytes), got %d", len(charter.Text), len(got))
	}

	// Second pass on the fresh file: no write, no note.
	changed2, notes2 := EnsureClaudeCharter(dir)
	for _, n := range notes2 {
		if strings.Contains(n, "refreshed .opencode/oikonomos.md") || strings.Contains(n, "seeded .opencode/oikonomos.md") {
			t.Fatalf("a fresh payload must not re-report: %v", notes2)
		}
	}
	_ = changed2
}

func TestEnsureClaudeCharterPayloadFreshIsSilent(t *testing.T) {
	// A fresh payload: no write, no seed/refresh note.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	payloadPath := filepath.Join(dir, ".opencode", "oikonomos.md")
	if err := os.WriteFile(payloadPath, []byte(charter.Text), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, notes := EnsureClaudeCharter(dir); containsNote(notes, "seeded .opencode/oikonomos.md") || containsNote(notes, "refreshed .opencode/oikonomos.md") {
		t.Fatalf("a fresh payload must not be (re)seeded or refreshed: %v", notes)
	}
}

func TestEnsureClaudeCharterRefreshesRenamedStarter(t *testing.T) {
	dir := t.TempDir()
	legacyName := "theboring" + "office"
	legacy := "<!-- " + legacyName + " charter -->\n" +
		"This project is served by " + legacyName + ": the oikonomos manager protocol imported below is the operating charter.\n" +
		claudeCharterImportLine + "\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, notes := EnsureClaudeCharter(dir)
	if !changed || !containsNote(notes, "refreshed (CLAUDE.md drifted from the office starter)") {
		t.Fatalf("a renamed office starter must refresh, changed=%v notes=%v", changed, notes)
	}
	got, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil || string(got) != wantClaudeCharterFresh {
		t.Fatalf("refreshed starter = %q, %v; want %q", got, err, wantClaudeCharterFresh)
	}
}
