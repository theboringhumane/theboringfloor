// claude_spawn_test.go — the claude backend's boot contract: Start
// resolves on the FIRST system/init, times out when none arrives, and the
// exit-before-init error carries the child's stderr.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// claudeStubScript writes a stream-json fake `claude` (POSIX sh) into a
// fresh temp dir and returns its path. The stub signals strictly through
// the THEBORINGOFFICE_CLAUDE_STUB_* env passthroughs (capture, argv log,
// signal log) — the child env allowlist carries them by construction.
func claudeStubScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "claudestub")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return p
}

// claudeEventLog collects emitted state.Events (backend emits from many
// goroutines — every read locks).
type claudeEventLog struct {
	mu  sync.Mutex
	evs []state.Event
}

func (l *claudeEventLog) emit(e state.Event) {
	l.mu.Lock()
	l.evs = append(l.evs, e)
	l.mu.Unlock()
}

func (l *claudeEventLog) snapshot() []state.Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]state.Event(nil), l.evs...)
}

// hasStatusContaining reports whether any EvStatus carries sub.
func (l *claudeEventLog) hasStatusContaining(sub string) bool {
	for _, e := range l.snapshot() {
		if e.Kind == state.EvStatus && strings.Contains(e.Text, sub) {
			return true
		}
	}
	return false
}

// claudeWait polls cond every 2ms until it holds or the deadline passes —
// the deterministic alternative to real sleeps (the shell stubs answer
// in ms; the poll only bounds the delivery).
func claudeWait(t *testing.T, what string, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

const claudeStubInitLine = `{"type":"system","subtype":"init","cwd":"/tmp","session_id":"sess-sh-1","model":"claude-test-1","mcp_servers":[{"name":"memo","status":"connected"},{"name":"sqlite","status":"failed"}],"claude_code_version":"2.1.246","uuid":"00000000-0000-4000-8000-000000000001"}` + "\n"

func TestClaudeStartResolvesOnInit(t *testing.T) {
	stub := claudeStubScript(t, `printf '%s\n' '`+claudeStubInitLine[:len(claudeStubInitLine)-1]+`'`+`
while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if got := b.PrimaryID(); got != "sess-sh-1" {
		t.Fatalf("PrimaryID = %q, want the init-pinned sess-sh-1", got)
	}
	claudeWait(t, "the [claude] init status line", 2*time.Second, func() bool {
		return log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-1")
	})
	if !log.hasStatusContaining("live (claude)") {
		t.Fatalf("missing the live transport status line; events: %v", log.snapshot())
	}
	servers, err := b.MCPServers()
	if err != nil {
		t.Fatalf("MCPServers: %v", err)
	}
	if len(servers) != 2 || servers[0].Name != "memo" || servers[1].Status != "failed" {
		t.Fatalf("init.mcp_servers pin drifted: %+v", servers)
	}
}

func TestClaudeStartTimeoutWithoutInit(t *testing.T) {
	stub := claudeStubScript(t, `while IFS= read -r line; do :; done
`)
	old := claudeStartTimeout
	claudeStartTimeout = 200 * time.Millisecond
	defer func() { claudeStartTimeout = old }()

	b := newClaudeBackend(stub, t.TempDir(), nil)
	err := b.Start(func(state.Event) {})
	if err == nil {
		t.Fatalf("Start must fail when no init arrives within the timeout")
	}
	if !strings.Contains(err.Error(), "no system/init") {
		t.Fatalf("timeout error text drifted: %v", err)
	}
	_ = b.Stop()
}

func TestClaudeStartExitBeforeInit(t *testing.T) {
	stub := claudeStubScript(t, `echo "claudestub: auth missing for project" >&2
exit 2
`)
	b := newClaudeBackend(stub, t.TempDir(), nil)
	err := b.Start(func(state.Event) {})
	if err == nil {
		t.Fatalf("Start must fail when the child exits before init")
	}
	for _, want := range []string{"exited before init", "auth missing for project"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("exit error missing %q: %v", want, err)
		}
	}
	_ = b.Stop()
}
