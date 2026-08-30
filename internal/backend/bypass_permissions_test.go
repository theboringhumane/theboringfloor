// bypass_permissions_test.go — the office's bypass-permissions toggle,
// backend leg:
//
//  1. CLAUDE argv — bypass ON spawns `--dangerously-skip-permissions`
//     and OMITS `--permission-prompt-tool stdio` (the contradictory
//     pair never rides); bypass OFF keeps today's argv byte-exact; a
//     death-respawn keeps the boot's mode (and still rides --resume).
//  2. OPENCODE config — the pure merge (mergePermissionWildcardAllow)
//     preserves every foreign field, no-ops on an already-bypassed
//     config, replaces a contradictory hand-shaped string loudly, and
//     fails closed on a non-object/non-string permission; the IO pass
//     (ensureBypassPermissions) creates a fresh config when absent and
//     is idempotent.
//  3. INTERFACE — SetBypassPermissions pre-Start latches (nil), after
//     Start (and after Stop) it fails naming "respawn required" — the
//     app's toggle always respawns a fresh backend.
//  4. START WIRING (opencode) — with bypass ON the serve's project
//     config gains the verified permission block; with bypass OFF the
//     config is byte-identical to an unmanaged (charter-only) boot.
package backend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/netwatch"
)

// ---------------------------------------------------------------- 1. claude argv

// TestClaudeSpawnArgvBypassPermissions — the bypass argv, pinned
// byte-exact: `--dangerously-skip-permissions` present (the real CLI
// 2.1.247's flag — `claude --help`: "Bypass all permission checks"),
// `--permission-prompt-tool stdio` ABSENT (with every check bypassed
// canUseTool never fires — the stdio prompt tool would be dead weight).
func TestClaudeSpawnArgvBypassPermissions(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	want := "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions"
	claudeWait(t, "the bypass spawn argv record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(argvlog)
		return err == nil && strings.TrimSpace(string(bits)) == want
	})
	bits, _ := os.ReadFile(argvlog)
	argv := strings.TrimSpace(string(bits))
	if argv != want {
		t.Fatalf("bypass argv drifted:\n got %q\nwant %q", argv, want)
	}
	if strings.Contains(argv, "--permission-prompt-tool") {
		t.Fatalf("the contradictory pair must never ride — argv carried --permission-prompt-tool under bypass: %s", argv)
	}
	// The bypass boot names the mode on the record (same transparency
	// convention as the majdoor/concierge-off lines).
	if !log.hasStatusContaining("[theboringoffice] bypass permissions: on (--dangerously-skip-permissions)") {
		t.Fatalf("the bypass boot line is missing; events: %v", log.snapshot())
	}
}

// TestClaudeSpawnArgvWithoutBypassUnchanged — the default argv is
// byte-identical to the pre-toggle shape: the permission-modal lifeline
// `--permission-prompt-tool stdio` rides and no bypass flag appears.
func TestClaudeSpawnArgvWithoutBypassUnchanged(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	want := "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --permission-prompt-tool stdio"
	claudeWait(t, "the default spawn argv record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(argvlog)
		return err == nil && strings.TrimSpace(string(bits)) == want
	})
	bits, _ := os.ReadFile(argvlog)
	if got := strings.TrimSpace(string(bits)); got != want {
		t.Fatalf("default argv drifted:\n got %q\nwant %q", got, want)
	}
	if log.hasStatusContaining("bypass permissions") {
		t.Fatalf("a bypass note must never print on a default boot; events: %v", log.snapshot())
	}
}

// TestClaudeRespawnKeepsBypass — a death-respawn (process dies, next
// Send respawns with --resume) keeps the boot's bypass mode: the second
// spawn's argv carries BOTH --dangerously-skip-permissions and the
// --resume pin, and still no --permission-prompt-tool. The stub stays
// ALIVE until the init pin lands and dies only on command: an
// instant-exit child loses its buffered stdout to the reaper's cmd.Wait
// closing the pipe (a pre-existing race this test deliberately avoids —
// it pins the RESPAWN argv, not the drain race).
func TestClaudeRespawnKeepsBypass(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	// Log argv, emit the preamble (init pins sess-sh-1), then serve stdin
	// until the magic line — the death the watch turns into a respawn.
	stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
`+claudeStubPreambleSh()+`while IFS= read -r line; do
  case "$line" in
    *please-die-now*) exit 0 ;;
  esac
done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	claudeWait(t, "the init pin before the death", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
	if err := b.Send("please-die-now"); err != nil {
		t.Fatalf("Send (death trigger): %v", err)
	}
	claudeWait(t, "the death watch latch (died=true)", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})
	if err := b.Send("are you still there?"); err != nil {
		t.Fatalf("Send (respawn trigger): %v", err)
	}

	// Line 2 of the argv log is the respawn: bypass flag AND --resume.
	claudeWait(t, "the respawn argv record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(argvlog)
		if err != nil {
			return false
		}
		return len(strings.Split(strings.TrimSpace(string(bits)), "\n")) >= 2
	})
	bits, _ := os.ReadFile(argvlog)
	lines := strings.Split(strings.TrimSpace(string(bits)), "\n")
	respawn := lines[1]
	wantSuffix := "--dangerously-skip-permissions --resume sess-sh-1"
	if !strings.HasSuffix(respawn, wantSuffix) {
		t.Fatalf("the respawn must keep bypass AND ride --resume sess-sh-1:\n got %q\nwant suffix %q", respawn, wantSuffix)
	}
	if strings.Contains(respawn, "--permission-prompt-tool") {
		t.Fatalf("the respawn must never re-add --permission-prompt-tool under bypass: %s", respawn)
	}
}

// ---------------------------------------------------------------- 2. opencode merge (pure + IO)

// TestMergePermissionWildcardAllow — the pure merge matrix (the verified
// schema shape: opencode 1.18.21's config.json, PermissionConfig with the
// "*" wildcard key taking "allow"/"ask"/"deny").
func TestMergePermissionWildcardAllow(t *testing.T) {
	t.Run("fresh field on an existing config preserves everything", func(t *testing.T) {
		in := "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"instructions\": [\n    \"./.opencode/oikonomos.md\"\n  ],\n  \"model\": \"anthropic/claude-opus-4-1\"\n}\n"
		merged, changed, note, err := mergePermissionWildcardAllow([]byte(in))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if !changed {
			t.Fatalf("a missing permission block must merge (changed=true)")
		}
		if note != "" {
			t.Fatalf("a clean merge carries no replacement note, got %q", note)
		}
		got := string(merged)
		for _, keep := range []string{`"$schema"`, `"./.opencode/oikonomos.md"`, `"model": "anthropic/claude-opus-4-1"`, `"permission"`, `"*": "allow"`} {
			if !strings.Contains(got, keep) {
				t.Fatalf("merged config lost %q:\n%s", keep, got)
			}
		}
	})

	t.Run("object form gains the wildcard and keeps sibling rules", func(t *testing.T) {
		in := `{"permission": {"edit": "deny", "bash": {"rm *": "deny"}}}` + "\n"
		merged, changed, _, err := mergePermissionWildcardAllow([]byte(in))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if !changed {
			t.Fatalf("an object without the wildcard must merge (changed=true)")
		}
		got := string(merged)
		// The member's explicit rules survive: opencode's documented
		// precedence lets a specific "deny" still beat the "*" wildcard.
		for _, keep := range []string{`"edit": "deny"`, `"rm *": "deny"`, `"*": "allow"`} {
			if !strings.Contains(got, keep) {
				t.Fatalf("merged config lost %q:\n%s", keep, got)
			}
		}
	})

	t.Run("already bypassed object is a byte-identical no-op", func(t *testing.T) {
		in := "{\n  \"permission\": {\n    \"*\": \"allow\"\n  }\n}\n"
		merged, changed, _, err := mergePermissionWildcardAllow([]byte(in))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if changed || string(merged) != in {
			t.Fatalf("an already-bypassed config must be byte-identical (changed=%v)", changed)
		}
	})

	t.Run("all-at-once allow string is a no-op", func(t *testing.T) {
		in := `{"permission": "allow"}` + "\n"
		_, changed, _, err := mergePermissionWildcardAllow([]byte(in))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if changed {
			t.Fatalf("\"permission\": \"allow\" already bypasses everything — no rewrite wanted")
		}
	})

	t.Run("contradictory string is replaced loudly", func(t *testing.T) {
		in := `{"permission": "ask"}` + "\n"
		merged, changed, note, err := mergePermissionWildcardAllow([]byte(in))
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if !changed {
			t.Fatalf("a contradictory string must be replaced (changed=true)")
		}
		if !strings.Contains(note, `"ask"`) || !strings.Contains(note, "replac") {
			t.Fatalf("the replacement must ride a loud note, got %q", note)
		}
		if !strings.Contains(string(merged), `"*": "allow"`) {
			t.Fatalf("the wildcard block must land:\n%s", merged)
		}
	})

	t.Run("hand-shaped permission fails closed", func(t *testing.T) {
		for _, in := range []string{
			`{"permission": null}`,
			`{"permission": 42}`,
			`{"permission": ["allow"]}`,
		} {
			if _, _, _, err := mergePermissionWildcardAllow([]byte(in)); err == nil {
				t.Fatalf("%s must fail closed (never clobber a hand-shaped config)", in)
			}
		}
	})

	t.Run("unparseable json fails closed", func(t *testing.T) {
		if _, _, _, err := mergePermissionWildcardAllow([]byte("{nope")); err == nil {
			t.Fatalf("unparseable json must fail closed")
		}
	})
}

// TestEnsureBypassPermissionsIO — the disk pass: creates a fresh config
// when absent (the verified block beside the $schema), merges into an
// existing one, and is idempotent (second run: changed=false, same bytes).
func TestEnsureBypassPermissionsIO(t *testing.T) {
	t.Run("creates a fresh config when absent", func(t *testing.T) {
		dir := t.TempDir()
		changed, notes := ensureBypassPermissions(dir)
		if !changed {
			t.Fatalf("an absent config must be created (changed=true); notes: %v", notes)
		}
		bits, err := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
		if err != nil {
			t.Fatalf("read fresh config: %v", err)
		}
		want := "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"permission\": {\n    \"*\": \"allow\"\n  }\n}\n"
		if string(bits) != want {
			t.Fatalf("fresh config drifted:\n got %q\nwant %q", string(bits), want)
		}
		// Idempotent second run.
		changed2, _ := ensureBypassPermissions(dir)
		if changed2 {
			t.Fatalf("the second run must be a no-op (changed=false)")
		}
		bits2, _ := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
		if string(bits2) != want {
			t.Fatalf("the second run rewrote the file")
		}
	})

	t.Run("merges beside the charter's instructions", func(t *testing.T) {
		dir := t.TempDir()
		if _, notes := EnsureCharter(dir); !strings.Contains(strings.Join(notes, "\n"), "manager charter") {
			t.Fatalf("charter notes missing: %v", notes)
		}
		before, _ := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
		changed, _ := ensureBypassPermissions(dir)
		if !changed {
			t.Fatalf("the charter config lacks the permission block — the merge must change it")
		}
		after, _ := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
		if !strings.Contains(string(after), `"*": "allow"`) || !strings.Contains(string(after), "./.opencode/oikonomos.md") {
			t.Fatalf("the merge must add the block AND keep the charter entry:\nbefore: %s\nafter: %s", before, after)
		}
	})
}

// ---------------------------------------------------------------- 3. the SetBypassPermissions contract

// TestClaudeSetBypassPermissionsContract — pre-Start latches (nil, both
// directions); once Start was called the argv is frozen and the call
// fails naming "respawn required" — before AND after Stop (a spent
// instance never takes a new mode; the toggle respawns a fresh one).
func TestClaudeSetBypassPermissionsContract(t *testing.T) {
	stub := claudeStubScript(t, `while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.SetBypassPermissions(false); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(false): %v", err)
	}
	b.mu.Lock()
	if b.bypassPermissions {
		t.Fatalf("the latch must follow the LAST pre-Start call (false)")
	}
	b.mu.Unlock()

	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	err := b.SetBypassPermissions(true)
	if err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start must fail naming \"respawn required\", got %v", err)
	}
	_ = b.Stop()
	err = b.SetBypassPermissions(true)
	if err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Stop must fail naming \"respawn required\" (a spent instance never takes a new mode), got %v", err)
	}
}

// TestLiveBackendSetBypassPermissionsContract — the opencode leg of the
// same contract.
func TestLiveBackendSetBypassPermissionsContract(t *testing.T) {
	b, _ := startLiveForTestBypass(t, t.TempDir(), false)
	err := b.SetBypassPermissions(true)
	if err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start must fail naming \"respawn required\", got %v", err)
	}
	_ = b.Stop()
	err = b.SetBypassPermissions(true)
	if err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Stop must fail naming \"respawn required\", got %v", err)
	}

	// And the pre-Start latch on a fresh instance.
	b2 := newLiveBackend("", t.TempDir(), config.Default())
	if err := b2.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	b2.mu.Lock()
	if !b2.bypassPermissions {
		t.Fatalf("the pre-Start latch must hold")
	}
	b2.mu.Unlock()
}

// ---------------------------------------------------------------- 4. opencode Start wiring

// startLiveForTestBypass boots a REAL liveBackend against the same
// minimal serve double as startLiveForTest, but rooted at dir (the test
// inspects dir/.opencode/opencode.json afterwards) and with the bypass
// toggle set pre-Start when on.
func startLiveForTestBypass(t *testing.T, dir string, on bool) (*liveBackend, *eventLog) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			w.Write([]byte(`{"id":"ses-primary","title":"theboringoffice office","time":{"created":1,"updated":1}}`))
		case strings.HasPrefix(r.URL.Path, "/event"):
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK) // empty body: streamOnce EOFs at once
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/session/"):
			w.Write([]byte(`true`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1") // refuse the lane probe instantly
	probe := &scriptedProbe{online: true}
	b := newLiveBackend(srv.URL, dir, config.Default())
	b.net = netwatch.New(probe.probe, 2*time.Millisecond)
	if on {
		if err := b.SetBypassPermissions(true); err != nil {
			t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
		}
	}
	log := &eventLog{}
	if err := b.Start(log.emit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Stop() }) // registered after srv.Close: runs first (LIFO)
	return b, log
}

// TestLiveBackendStartBypassWiresPermission — with the toggle ON the
// serve's project config gains the verified permission block ahead of
// the boot, and the note rides the status line; with the toggle OFF the
// config stays byte-identical to an unmanaged boot (no permission key).
func TestLiveBackendStartBypassWiresPermission(t *testing.T) {
	t.Run("bypass on merges the block", func(t *testing.T) {
		dir := t.TempDir()
		_, log := startLiveForTestBypass(t, dir, true)
		bits, err := os.ReadFile(filepath.Join(dir, ".opencode", "opencode.json"))
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(bits)
		if !strings.Contains(got, `"permission"`) || !strings.Contains(got, `"*": "allow"`) {
			t.Fatalf("the bypass merge must land in the project config:\n%s", got)
		}
		// The charter's own wiring survives beside it.
		if !strings.Contains(got, "./.opencode/oikonomos.md") {
			t.Fatalf("the charter entry must survive the bypass merge:\n%s", got)
		}
		if log.textCount("[theboringoffice] bypass permissions: wired") == 0 {
			t.Fatalf("the bypass note must ride the boot's status lines; events: %v", log.kinds())
		}
	})

	t.Run("bypass off leaves the config byte-identical", func(t *testing.T) {
		dirOff := t.TempDir()
		_, logOff := startLiveForTestBypass(t, dirOff, false)
		off, err := os.ReadFile(filepath.Join(dirOff, ".opencode", "opencode.json"))
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		if strings.Contains(string(off), `"permission"`) {
			t.Fatalf("an unmanaged boot must never grow a permission block:\n%s", off)
		}
		// Byte-identical to today: the same bytes a bare EnsureCharter
		// pass (the pre-toggle boot) produces.
		dirBare := t.TempDir()
		EnsureCharter(dirBare)
		bare, _ := os.ReadFile(filepath.Join(dirBare, ".opencode", "opencode.json"))
		if string(off) != string(bare) {
			t.Fatalf("the no-toggle config must be byte-identical to a bare charter pass:\nboot:  %s\nbare:  %s", off, bare)
		}
		if logOff.textCount("bypass permissions") != 0 {
			t.Fatalf("no bypass note without the toggle; events: %v", logOff.kinds())
		}
	})
}
