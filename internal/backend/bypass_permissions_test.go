package backend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/netwatch"
)

// The Claude coverage remains here because the shared pre-Start bypass
// contract is provider-neutral; OpenCode's persistence regression is covered
// below with its own spawned-serve harness.
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
	got := b.bypassPermissions
	b.mu.Unlock()
	if got {
		t.Fatal("last pre-Start setting must win")
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if err := b.SetBypassPermissions(true); err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start SetBypassPermissions = %v, want respawn-required error", err)
	}
}

func TestLiveBackendSetBypassPermissionsContract(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.SetBypassPermissions(false); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(false): %v", err)
	}
	b.mu.Lock()
	got := b.bypassPermissions
	b.mu.Unlock()
	if got {
		t.Fatal("last pre-Start setting must win")
	}
	b.started = true
	if err := b.SetBypassPermissions(true); err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start SetBypassPermissions = %v, want respawn-required error", err)
	}
}

func TestOpenCodeBypassIsEphemeral(t *testing.T) {
	t.Run("on injects only the child environment and leaves project bytes unchanged", func(t *testing.T) {
		b, log, cfgPath, envLog := startEphemeralBypassHarness(t, true, false)
		before, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		waitBypassEnv(t, envLog, 1)
		gotEnv, _ := os.ReadFile(envLog)
		if string(gotEnv) != bypassConfigContent+"\n" {
			t.Fatalf("bypass child env = %q, want %q", gotEnv, bypassConfigContent+"\n")
		}
		if log.textCount("bypass permissions: on (ephemeral OPENCODE_CONFIG_CONTENT override)") == 0 {
			t.Fatalf("missing ephemeral bypass status: %v", log.kinds())
		}
		if err := b.Stop(); err != nil {
			t.Fatal(err)
		}
		after, _ := os.ReadFile(cfgPath)
		if string(after) != string(before) {
			t.Fatalf("bypass changed project config:\nbefore: %s\nafter: %s", before, after)
		}
	})

	t.Run("off has no override and never removes an unmarked wildcard", func(t *testing.T) {
		b, _, cfgPath, envLog := startEphemeralBypassHarness(t, false, true)
		before, _ := os.ReadFile(cfgPath)
		if !strings.Contains(string(before), `"*": "allow"`) {
			t.Fatalf("fixture must contain an unmarked wildcard before off boot: %s", before)
		}
		waitBypassEnv(t, envLog, 1)
		gotEnv, _ := os.ReadFile(envLog)
		if string(gotEnv) != "\n" {
			t.Fatalf("off child unexpectedly received an override: %q", gotEnv)
		}
		_ = b.Stop()
		after, _ := os.ReadFile(cfgPath)
		if string(after) != string(before) {
			t.Fatalf("off boot changed project config:\nbefore: %s\nafter: %s", before, after)
		}
	})

	t.Run("an owned-serve death respawn retains the override", func(t *testing.T) {
		b, _, _, envLog := startEphemeralBypassHarness(t, true, false)
		b.mu.Lock()
		proc := b.proc
		b.mu.Unlock()
		if proc == nil || proc.Process == nil {
			t.Fatal("expected owned OpenCode serve process")
		}
		if err := proc.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(3 * time.Second)
		died := false
		for time.Now().Before(deadline) {
			b.mu.Lock()
			died = b.serveDied
			b.mu.Unlock()
			if died {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !died {
			t.Fatal("timed out waiting for owned serve death")
		}
		b.respawnServeForSend()
		waitBypassEnv(t, envLog, 2)
		got, _ := os.ReadFile(envLog)
		want := bypassConfigContent + "\n" + bypassConfigContent + "\n"
		if string(got) != want {
			t.Fatalf("respawn env = %q, want %q", got, want)
		}
	})
}

func TestOpenCodeBypassChildEnvDoesNotInheritParentOverride(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{"permission":{"*":"allow"},"poison":true}`)

	t.Run("off strips the poisoned parent override", func(t *testing.T) {
		b, _, _, envLog := startEphemeralBypassHarness(t, false, false)
		waitBypassEnv(t, envLog, 1)
		got, err := os.ReadFile(envLog)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "\n" {
			t.Fatalf("off child inherited parent override: %q", got)
		}
		if err := b.Stop(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on replaces the poisoned parent override with the office override", func(t *testing.T) {
		b, _, _, envLog := startEphemeralBypassHarness(t, true, false)
		waitBypassEnv(t, envLog, 1)
		got, err := os.ReadFile(envLog)
		if err != nil {
			t.Fatal(err)
		}
		want := bypassConfigContent + "\n"
		if string(got) != want {
			t.Fatalf("on child env = %q, want exact office override %q", got, want)
		}
		if err := b.Stop(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestOpenCodeStopAwaitsSoleServeReaperAfterRespawn(t *testing.T) {
	b, _, _, _ := startEphemeralBypassHarness(t, true, false)
	b.mu.Lock()
	first := b.proc
	b.mu.Unlock()
	if first == nil || first.Process == nil {
		t.Fatal("expected initial owned serve")
	}
	if err := first.Process.Kill(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b.mu.Lock()
		died := b.serveDied
		b.mu.Unlock()
		if died {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	b.mu.Lock()
	died := b.serveDied
	b.mu.Unlock()
	if !died {
		t.Fatal("timed out waiting for initial serve reaper to report death")
	}

	b.respawnServeForSend()
	b.mu.Lock()
	second := b.proc
	exit := b.procExit
	b.mu.Unlock()
	if second == nil || second.Process == nil || exit == nil {
		t.Fatal("expected respawned serve and its reaper completion")
	}
	if err := b.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exit.done:
	case <-time.After(time.Second):
		t.Fatal("Stop did not await spawnServe's sole reaper")
	}
}

func startEphemeralBypassHarness(t *testing.T, on, legacyWildcard bool) (*liveBackend, *eventLog, string, string) {
	t.Helper()
	dir := t.TempDir()
	if legacyWildcard {
		ocDir := filepath.Join(dir, ".opencode")
		if err := os.MkdirAll(ocDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ocDir, "opencode.json"), []byte("{\n  \"permission\": {\n    \"*\": \"allow\"\n  }\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, notes := EnsureCharter(dir); len(notes) == 0 {
		t.Fatal("EnsureCharter produced no status")
	}
	cfgPath := filepath.Join(dir, ".opencode", "opencode.json")
	envLog := filepath.Join(t.TempDir(), "opencode-config-content.log")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			_, _ = w.Write([]byte(`{"id":"ses-primary","title":"theboringoffice office","time":{"created":1,"updated":1}}`))
		case strings.HasPrefix(r.URL.Path, "/event"):
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/session/"):
			_, _ = w.Write([]byte(`true`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "opencode")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\n' \"${OPENCODE_CONFIG_CONTENT-}\" >> \"$OPENCODE_ENV_LOG\"\nprintf 'opencode server listening on %s\\n' \"$OPENCODE_FIXTURE_URL\"\nwhile :; do sleep 1; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OPENCODE_ENV_LOG", envLog)
	t.Setenv("OPENCODE_FIXTURE_URL", srv.URL)
	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")
	probe := &scriptedProbe{online: true}
	b := newLiveBackend("", dir, config.Default())
	b.net = netwatch.New(probe.probe, 2*time.Millisecond)
	if on {
		if err := b.SetBypassPermissions(true); err != nil {
			t.Fatal(err)
		}
	}
	log := &eventLog{}
	if err := b.Start(log.emit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Stop() })
	return b, log, cfgPath, envLog
}

func waitBypassEnv(t *testing.T, path string, lines int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		bits, err := os.ReadFile(path)
		if err == nil && strings.Count(string(bits), "\n") >= lines {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	bits, _ := os.ReadFile(path)
	t.Fatalf("timed out waiting for %d spawn env records at %s; got %q", lines, path, bits)
}
