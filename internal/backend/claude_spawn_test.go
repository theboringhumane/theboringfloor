// claude_spawn_test.go — the claude backend's boot contract: Start seats
// the floor IMMEDIATELY (there is NO system/init wait — `claude -p` emits
// init only after the first stdin user message), init pins primaryID
// WHENEVER it arrives, a child exit before init is the death watch's
// report (never a Start error), and CLAUDE_CONFIG_DIR defaults to the
// user's real ~/.claude unless THEBORINGOFFICE_CLAUDE_CONFIG explicitly
// opts into a sandbox.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
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

// claudeStubHookLines — the REAL `claude -p` stream prefix: startup hook
// frames (hook_started/hook_response pairs) land BEFORE system/init
// (verified against a live 2.x session: init is line 5 of the stream).
// The mapper ignores them silently (the open-ended parser rule), so every
// stub emits them — the whole suite proves init need not be first.
const claudeStubHookLines = `{"type":"system","subtype":"hook_started","hook_id":"hk-0","hook_name":"SessionStart","cwd":"/tmp"}` + "\n" +
	`{"type":"system","subtype":"hook_response","hook_id":"hk-0","hook_name":"SessionStart","exit_code":0,"stdout":"","stderr":""}` + "\n" +
	`{"type":"system","subtype":"hook_started","hook_id":"hk-1","hook_name":"UserPromptSubmit","cwd":"/tmp"}` + "\n" +
	`{"type":"system","subtype":"hook_response","hook_id":"hk-1","hook_name":"UserPromptSubmit","exit_code":0,"stdout":"","stderr":""}` + "\n"

const claudeStubInitLine = `{"type":"system","subtype":"init","cwd":"/tmp","session_id":"sess-sh-1","model":"claude-test-1","mcp_servers":[{"name":"memo","status":"connected"},{"name":"sqlite","status":"failed"}],"claude_code_version":"2.1.246","uuid":"00000000-0000-4000-8000-000000000001"}` + "\n"

// claudeStubPreamble — the full pre-conversation stream prefix every stub
// emits: the four hook frames, THEN system/init at line 5 (real shape).
const claudeStubPreamble = claudeStubHookLines + claudeStubInitLine

// claudeStubSh renders raw stream lines as shell printf statements for a
// stub script body (single-quote safe: the frames carry no ' bytes).
func claudeStubSh(stream string) string {
	var b strings.Builder
	for _, ln := range strings.Split(strings.TrimSuffix(stream, "\n"), "\n") {
		b.WriteString("printf '%s\\n' '")
		b.WriteString(ln)
		b.WriteString("'\n")
	}
	return b.String()
}

// claudeStubPreambleSh — the preamble as shell (hooks, then init).
func claudeStubPreambleSh() string { return claudeStubSh(claudeStubPreamble) }

// TestClaudeSpawnArgvPermissionPromptTool — the permission-modal lifeline:
// headless `claude -p` only wires canUseTool to the stdio control channel
// when `--permission-prompt-tool stdio` rides the spawn argv (the CLI's
// own SDK spawn builder pushes exactly that). Without it EVERY
// permission-requiring tool is auto-denied before the office modal can
// appear, and the denial lands as a tool_result error row. A
// --permission-mode flag would SILENCE prompts instead — never allowed.
func TestClaudeSpawnArgvPermissionPromptTool(t *testing.T) {
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

	claudeWait(t, "the spawn argv record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(argvlog)
		return err == nil && strings.Contains(string(bits), "--permission-prompt-tool stdio")
	})
	bits, err := os.ReadFile(argvlog)
	if err != nil {
		t.Fatalf("read argv log: %v", err)
	}
	argv := string(bits)
	if !strings.Contains(argv, "--permission-prompt-tool stdio") {
		t.Fatalf("spawn argv missing `--permission-prompt-tool stdio`: %s", argv)
	}
	if strings.Contains(argv, "--permission-mode") {
		t.Fatalf("a --permission-mode would silence the modal prompts — argv must never carry one: %s", argv)
	}
}

// TestClaudeStartSeatsFloorBeforeInit — the no-init-wait boot contract:
// Start returns nil with the hires + hint lines ALREADY emitted in a
// deterministic order (the reader starts after them), and the init pins
// land when the reader maps the frame a beat later.
func TestClaudeStartSeatsFloorBeforeInit(t *testing.T) {
	stub := claudeStubScript(t, claudeStubPreambleSh()+`while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	// the boot prefix is deterministic: manager + hr hires, then the two
	// hint lines — all emitted synchronously BEFORE the reader could map
	// anything (the reader goroutine starts after them in Start).
	evs := log.snapshot()
	if len(evs) < 4 {
		t.Fatalf("Start must seat the floor immediately, got %d events: %v", len(evs), evs)
	}
	if evs[0].Kind != state.EvHire || evs[0].Employee.Name != "manager" {
		t.Fatalf("event 0 must be the manager hire, got %+v", evs[0])
	}
	if evs[1].Kind != state.EvHire || evs[1].Employee.Name != "hr" {
		t.Fatalf("event 1 must be the hr hire, got %+v", evs[1])
	}
	if evs[2].Kind != state.EvStatus || evs[2].Text != "[theboringoffice] backend: claudecode" {
		t.Fatalf("event 2 must be the backend-name hint, got %+v", evs[2])
	}
	if evs[3].Kind != state.EvStatus || !strings.Contains(evs[3].Text, "live (claude)") {
		t.Fatalf("event 3 must be the live transport line, got %+v", evs[3])
	}

	// init pins when the reader maps it (the stub printed it at spawn).
	claudeWait(t, "the init pin (primaryID)", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
	if !log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-1") {
		t.Fatalf("missing the [claude] init status line; events: %v", log.snapshot())
	}
	// the leading hook frames were ignored silently (open-ended parser).
	if log.hasStatusContaining("hook") {
		t.Fatalf("hook frames must never surface as statuses; events: %v", log.snapshot())
	}
	servers, err := b.MCPServers()
	if err != nil {
		t.Fatalf("MCPServers: %v", err)
	}
	if len(servers) != 2 || servers[0].Name != "memo" || servers[1].Status != "failed" {
		t.Fatalf("init.mcp_servers pin drifted: %+v", servers)
	}
}

// TestClaudeStartNeverWaitsForInit — a claude that stays silent forever
// (stdin open but unfed — the verified real `claude -p` behavior) must
// NOT block or fail Start: the init-timeout error contract is gone.
func TestClaudeStartNeverWaitsForInit(t *testing.T) {
	stub := claudeStubScript(t, `while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	done := make(chan error, 1)
	go func() { done <- b.Start(log.emit) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start must return nil without init, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Start blocked waiting for system/init — the no-init-wait contract regressed")
	}
	defer func() { _ = b.Stop() }()

	if got := b.PrimaryID(); got != "" {
		t.Fatalf("PrimaryID = %q, want empty until init arrives", got)
	}
	if !log.hasStatusContaining("[theboringoffice] backend: claudecode") {
		t.Fatalf("the backend-name hint must fire without init; events: %v", log.snapshot())
	}
	if log.hasStatusContaining("no system/init") {
		t.Fatalf("the init-timeout error is gone — it must never be emitted")
	}
	var manager, hr bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvHire && e.Employee.Name == "manager" {
			manager = true
		}
		if e.Kind == state.EvHire && e.Employee.Name == "hr" {
			hr = true
		}
	}
	if !manager || !hr {
		t.Fatalf("the floor must seat pre-init (manager=%v hr=%v)", manager, hr)
	}
}

// TestClaudeStartSlowInitPinsLate — the binding timing proof: init lands
// 2s AFTER Start returned; Start neither waited nor errored, and the late
// init still pins primaryID (the reader contract).
func TestClaudeStartSlowInitPinsLate(t *testing.T) {
	stub := claudeStubScript(t, `sleep 2
`+claudeStubPreambleSh()+`while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	start := time.Now()
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Start took %v — it must return immediately, not wait out the 2s init delay", d)
	}
	defer func() { _ = b.Stop() }()

	if got := b.PrimaryID(); got != "" {
		t.Fatalf("PrimaryID = %q before the slow init — want empty", got)
	}
	claudeWait(t, "the late init pin", 5*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
	if !log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-1") {
		t.Fatalf("the late init must still map its status line; events: %v", log.snapshot())
	}
}

// TestClaudeStartChildExitPreInit — a child that exits before any init
// (the fresh-sandbox silent-exit shape): Start still returns nil, the
// DEATH WATCH (which parks on the exit channel only, never initDone)
// reports it, and a Send degrades to the dead-backend bubble instead of a
// stdin write.
func TestClaudeStartChildExitPreInit(t *testing.T) {
	stub := claudeStubScript(t, `echo "claudestub: auth missing for project" >&2
exit 2
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start must never fail on a pre-init exit, got %v", err)
	}
	defer func() { _ = b.Stop() }()

	claudeWait(t, "the death watch latch (died=true)", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})
	claudeWait(t, "the pre-init death status line", 2*time.Second, func() bool {
		return log.hasStatusContaining("claude process died before system/init")
	})
	if !log.hasStatusContaining("[theboringoffice] backend: claudecode") {
		t.Fatalf("the floor must seat even when the child is already gone")
	}
	if err := b.Send("hello?"); err != nil {
		t.Fatalf("Send on the dead child: %v", err)
	}
	claudeWait(t, "the dead-backend bubble", 2*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvChatBoss && strings.Contains(e.Msg.Text, "backend not started") {
				return true
			}
		}
		return false
	})
}

// TestClaudeSendTriggersLateInit — THE user flow: spawn (hook frames
// lead, NO init), Start returns nil immediately, one Send writes the user
// line, and ONLY THEN does claude emit system/init (line 5 after the
// hooks) — the reader maps it on arrival, primaryID pins, and the turn
// proceeds (assistant reply + result).
func TestClaudeSendTriggersLateInit(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubSh(claudeStubHookLines) + `n=0
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      n=$((n+1))
      if [ "$n" -eq 1 ]; then
` + claudeStubSh(claudeStubInitLine) + `      fi
      printf '%s\n' '{"type":"assistant","message":{"id":"msg-late-'$n'","role":"assistant","content":[{"type":"text","text":"Roger that."}]},"session_id":"sess-sh-1","uuid":"msg-late-'$n'","parent_tool_use_id":null}'
      printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"sess-sh-1","uuid":"res-late-'$n'","total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":1}}'
      ;;
  esac
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	// init has NOT been emitted yet (it follows the first stdin line) —
	// primaryID is honestly empty, and the hook prefix mapped to nothing.
	if got := b.PrimaryID(); got != "" {
		t.Fatalf("PrimaryID = %q before the first Send — want empty (init trails the first prompt)", got)
	}
	if log.hasStatusContaining("hook") {
		t.Fatalf("hook frames must map to nothing; events: %v", log.snapshot())
	}

	if err := b.Send("hello there"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// the FIRST user line carries the browser-tool preamble ahead of the
	// member text (browser_open_test.go owns that contract) — assert the
	// exact placement via the production encoder plus the literal marker
	// intro, so a wire-shape drift still fails here.
	wantUser := string(claudeUserLineFor(browsertools.PromptPreamble + "\n\nhello there"))
	claudeWait(t, "the user line in the stub capture", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	lines := claudeCapture(t, capture)
	if len(lines) != 1 || lines[0] != wantUser || !strings.Contains(lines[0], "⟦open-browser:") {
		t.Fatalf("first stdin line must be preamble + prompt:\n got %q\nwant %q", lines, wantUser)
	}

	// the Send-triggered init maps ON ARRIVAL: primaryID pins, the init
	// status lands, and the turn's reply completes the placeholder.
	claudeWait(t, "the late init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
	claudeWait(t, "the init status line", 2*time.Second, func() bool {
		return log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-1")
	})
	claudeWait(t, "the turn's Roger reply", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvChatBoss && e.Msg.Text == "Roger that." && !e.Msg.Pending {
				return true
			}
		}
		return false
	})
}

// TestClaudePrimaryOverrideWinsOverInit — the restore pin (the app's
// primarySeamBackend call, pre-Start) reports immediately and WINS over
// the init frame's own session_id; the RESUME pin still follows the live
// wire session (the respawn target must be the process's real session).
func TestClaudePrimaryOverrideWinsOverInit(t *testing.T) {
	stub := claudeStubScript(t, claudeStubPreambleSh()+`while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	b.PrimaryOverride("sess-pinned-9")
	if got := b.PrimaryID(); got != "sess-pinned-9" {
		t.Fatalf("the pin must report pre-Start, got %q", got)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	claudeWait(t, "the init frame mapped", 3*time.Second, func() bool {
		return log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-1")
	})
	if got := b.PrimaryID(); got != "sess-pinned-9" {
		t.Fatalf("the override must win over the wire id, got %q", got)
	}
	b.mu.Lock()
	resume := b.resumeID
	b.mu.Unlock()
	if resume != "sess-sh-1" {
		t.Fatalf("the resume pin follows the LIVE session, got %q", resume)
	}
}

// TestClaudeEmptyOverrideLetsInitPin — "" semantics: no pin, init's own
// session_id wins on arrival.
func TestClaudeEmptyOverrideLetsInitPin(t *testing.T) {
	stub := claudeStubScript(t, claudeStubPreambleSh()+`while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	b.PrimaryOverride("") // explicit empty = no pin
	if got := b.PrimaryID(); got != "" {
		t.Fatalf("an empty override must not invent an id, got %q", got)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	claudeWait(t, "the init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
}

// TestClaudeConfigDirDefaultsToRealHome — no explicit opt-in: the child
// gets the user's REAL ~/.claude so an existing `claude` login carries
// into office sessions. THEBORINGOFFICE_HOME still roots the default for
// harness hermeticity; GRAFEIO_HOME is the pre-rename fallback.
func TestClaudeConfigDirDefaultsToRealHome(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", "")
	t.Setenv("THEBORINGOFFICE_HOME", "")
	t.Setenv("GRAFEIO_HOME", "")
	want := filepath.Join(os.Getenv("HOME"), ".claude")
	if got := claudeConfigDir(t.TempDir()); got != want {
		t.Fatalf("claudeConfigDir = %q, want the user's real config %q", got, want)
	}
}

func TestClaudeConfigDirHonorsHomeOverride(t *testing.T) {
	scratch := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", scratch)
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", "")
	want := filepath.Join(scratch, ".claude")
	if got := claudeConfigDir(t.TempDir()); got != want {
		t.Fatalf("claudeConfigDir = %q, want the override-rooted %q", got, want)
	}
}

// TestClaudeConfigDirExplicitSandboxOptIn — THEBORINGOFFICE_CLAUDE_CONFIG
// wins outright (the only sandbox path left); whitespace-only is NOT set.
func TestClaudeConfigDirExplicitSandboxOptIn(t *testing.T) {
	sandbox := filepath.Join(t.TempDir(), "sandbox-test")
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", sandbox)
	if got := claudeConfigDir(t.TempDir()); got != sandbox {
		t.Fatalf("claudeConfigDir = %q, want the explicit sandbox %q", got, sandbox)
	}
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", "   ")
	t.Setenv("THEBORINGOFFICE_HOME", "")
	t.Setenv("GRAFEIO_HOME", "")
	if got := claudeConfigDir(t.TempDir()); got == "   " {
		t.Fatalf("a whitespace-only pin must be treated as unset")
	}
}

// TestClaudeChildEnvConfigDir — the env carries CLAUDE_CONFIG_DIR and the
// MkdirAll is best-effort: a creatable sandbox appears; an uncreatable
// path (under a regular FILE) never fails the env build.
func TestClaudeChildEnvConfigDir(t *testing.T) {
	sandbox := filepath.Join(t.TempDir(), "sand")
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", sandbox)
	env := claudeChildEnv(t.TempDir())
	found := ""
	for _, kv := range env {
		if strings.HasPrefix(kv, "CLAUDE_CONFIG_DIR=") {
			found = strings.TrimPrefix(kv, "CLAUDE_CONFIG_DIR=")
		}
	}
	if found != sandbox {
		t.Fatalf("child env CLAUDE_CONFIG_DIR = %q, want %q", found, sandbox)
	}
	if st, err := os.Stat(sandbox); err != nil || !st.IsDir() {
		t.Fatalf("best-effort MkdirAll must create a creatable sandbox: %v", err)
	}

	f, err := os.Create(filepath.Join(t.TempDir(), "afile"))
	if err != nil {
		t.Fatalf("fixture file: %v", err)
	}
	_ = f.Close()
	impossible := filepath.Join(f.Name(), "child")
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", impossible)
	env = claudeChildEnv(t.TempDir()) // must not panic or fail
	found = ""
	for _, kv := range env {
		if strings.HasPrefix(kv, "CLAUDE_CONFIG_DIR=") {
			found = strings.TrimPrefix(kv, "CLAUDE_CONFIG_DIR=")
		}
	}
	if found != impossible {
		t.Fatalf("an uncreatable config dir must still ride the env (best-effort mkdir), got %q", found)
	}
}

// TestClaudeStartExplicitSandboxConfig — the opt-in sandbox still boots:
// THEBORINGOFFICE_CLAUDE_CONFIG passes straight through to the child's
// env and Start returns nil whether or not init ever arrives.
func TestClaudeStartExplicitSandboxConfig(t *testing.T) {
	sandbox := filepath.Join(t.TempDir(), "sandbox-test")
	t.Setenv("THEBORINGOFFICE_CLAUDE_CONFIG", sandbox)
	envlog := filepath.Join(t.TempDir(), "env.log")
	stub := claudeStubScript(t, `printf '%s\n' "$CLAUDE_CONFIG_DIR" >> "`+envlog+`"
while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start with an explicit sandbox config must return nil (init optional), got %v", err)
	}
	defer func() { _ = b.Stop() }()
	claudeWait(t, "the child's CLAUDE_CONFIG_DIR record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(envlog)
		return err == nil && strings.TrimSpace(string(bits)) == sandbox
	})
}
