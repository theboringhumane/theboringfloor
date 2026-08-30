// claude.go — the live Claude Code backend for theboringoffice: one
// `claude -p --input-format stream-json --output-format stream-json
// --verbose --include-partial-messages --permission-prompt-tool stdio`
// process per office session (a
// per-session process, NOT a daemon), with the stdout JSONL stream
// normalized into state.Events via claude_events.go (pure, mirroring how
// opencode.go + events.go share work).
//
// Session identity: the boss conversation's claude session_id pins at the
// FIRST system/init (never from a user Send) and becomes the ONE resume
// pin: a dead process is never auto-respawned, but the NEXT user Send
// respawns with `--resume <pinned uuid>` (mid-turn kill recovery).
// `claude -p` emits system/init only AFTER the first stdin user message
// (startup hook frames lead, init lands at line 5+ of the stream), so
// Start NEVER waits on it: the floor seats immediately and init pins
// primaryID/resumeID whenever it arrives (readLoop).
//
// Chat path (opencode parity): Send echoes EvChatUser locally (send-owned,
// never from the wire), stages ONE pending boss placeholder and writes
// EXACTLY ONE stdin user line under the single-writer mutex. Main-
// conversation stream_event text deltas grow "bossmsg-<assistant uuid>"
// (coalesced 150ms like the opencode thought/chat gate); the assistant
// snapshot pins the final text on the same id (Pending:false).
//
// Control path: control_request can_use_tool -> EvPermission;
// request_user_dialog -> EvQuestion. Answers write control_response ONCE
// (once->allow / always->allow+updatedPermissions / reject->deny+message)
// and emit the local resolved event the modal needs (claude never
// echoes responses).
//
// Kill ladder (/stop): interrupt control_request -> SIGINT after
// claudeAbortSigIntAfter -> SIGTERM after claudeAbortSigTermAfter; a
// signal-killed child (exit 130/143) is a CLEAN kill, not a crash.
package backend

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/gitx"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// liveClaudeBackend is the claude CLI twin of liveBackend (opencode.go).
type liveClaudeBackend struct {
	directory string
	cfg       *config.Config
	bin       string
	fl        *flow

	mu        sync.Mutex
	proc      *exec.Cmd
	procStdin io.WriteCloser
	procExit  <-chan error
	procWait  chan struct{} // closed when the current proc's watch drains
	initDone  bool          // informational: system/init has landed (set by readLoop, never by Start)
	stopping  bool          // Stop() engaged: the watch stays silent
	died      bool          // the watch latched a death; next Send respawns
	resumeID  string
	primaryID string
	// primaryOverride — the office-session restore pin (internal/app
	// PrimaryOverride seam). Set pre-Start, it WINS over the wire's own
	// init session_id; "" means no pin (init's id pins on arrival).
	primaryOverride string
	// bypassPermissions — the office's bypass-permissions toggle
	// (SetBypassPermissions, pre-Start only). When on, every spawn (boot
	// AND death-respawn) rides `--dangerously-skip-permissions` and OMITS
	// `--permission-prompt-tool stdio`: the CLI then never raises
	// can_use_tool, so zero EvPermission events arrive (nothing is
	// stripped office-side). started latches at the top of Start — the
	// flag is frozen from then on (the app's toggle respawns a FRESH
	// instance rather than mutating a running one).
	bypassPermissions bool
	started           bool
	busyTurns         int // outstanding turns (user writes without a result)
	chatSeq           int
	pendingBoss       []string
	interruptSeq      int
	interruptArm      bool // an interrupt is already in flight for the live turn(s)

	writeMu sync.Mutex // single-writer stdin guard

	ctx          *claudeNormCtx
	thoughtSlots map[string]*thoughtSlot
	chatSlots    map[string]*thoughtSlot

	lastUserText string
	lastUserAt   int64

	// questionStash holds the decoded question pages of each PENDING
	// request_user_dialog (keyed by control request_id), lifted from the
	// mapped EvQuestion in readLoop. AnswerQuestion needs them to rebuild
	// the CLI-native dialog result (for the AskUserQuestion kind:
	// updatedInput.questions + the answers map keyed by question text);
	// AnswerQuestion/RejectQuestion pop it. The per-kind decode the
	// answer needs (family, F1 input, label->result bytes) lives in
	// ctx.dialogMeta — the permMeta analog.
	questionStash map[string][]state.QuestionItem

	// initSeq numbers the initialize control_requests (one per process:
	// the FIRST stdin line of every spawn and every --resume respawn).
	initSeq int

	// briefed latches once the browser-tool preamble
	// (browsertools.PromptPreamble) has ridden the FIRST stdin user line
	// (set only on a clean write — a failed line re-briefs next Send);
	// a --resume respawn keeps the latch (the resumed session's context
	// retains the contract). browserBridge is the shared marker-policy
	// bridge (its emit closure reads fl AT CALL TIME).
	briefed       bool
	browserBridge *browsertools.Bridge

	// rawFrameHook — TEST-ONLY seam (claude_live_test.go): when non-nil,
	// readLoop hands it every parsed stdout frame BEFORE mapping. Nil in
	// production (one nil-check branch per frame, zero behavior change).
	// The live-CLI evidence tests need the REAL wire's frame types and the
	// assistant frame's uuid-vs-message.id divergence — the mapped
	// state.Events deliberately scrub the frame uuid (that scrub IS the
	// doubled-bubble fix), so no events-only observation can produce that
	// mandated evidence. Set BEFORE Start, never mutated after (readLoop's
	// goroutine starts in Start, giving the write a happens-before edge).
	rawFrameHook func(claudeEvent)
}

var _ state.Backend = (*liveClaudeBackend)(nil)

// newClaudeBackend wires the backend; bin "" means "resolve at Start"
// (THEBORINGOFFICE_CLAUDE_BIN, then PATH's `claude`).
func newClaudeBackend(bin, directory string, cfg *config.Config) *liveClaudeBackend {
	b := &liveClaudeBackend{
		directory:     directory,
		cfg:           cfgOrDefault(cfg),
		bin:           bin,
		fl:            newFlow(),
		ctx:           newClaudeNormCtx(cfgOrDefault(cfg)),
		thoughtSlots:  make(map[string]*thoughtSlot),
		chatSlots:     make(map[string]*thoughtSlot),
		questionStash: make(map[string][]state.QuestionItem),
	}
	b.browserBridge = browsertools.NewBridge(func(e state.Event) { b.fl.emit(e) })
	return b
}

// NewClaude creates the live claude-CLI backend: directory is the office's
// project dir, binOverride overrides the `claude` binary explicitly ("" =
// THEBORINGOFFICE_CLAUDE_BIN env, then PATH's `claude`); cfg may be nil —
// config.Default(). Mirrors NewLive's (baseURL, directory, cfg) shape so
// the app's transport resolver (model.go backendFor) can construct both
// backends through one seam.
func NewClaude(binOverride, directory string, cfg *config.Config) state.Backend {
	return newClaudeBackend(binOverride, directory, cfg)
}

func (b *liveClaudeBackend) Mode() state.Mode { return state.ModeLive }

// PrimaryID returns the boss conversation's claude session id ("" until
// system/init pins it) — the same additive seam liveBackend exposes.
func (b *liveClaudeBackend) PrimaryID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.primaryID
}

// SessionID is PrimaryID by another name (harness probes key on it).
func (b *liveClaudeBackend) SessionID() string { return b.PrimaryID() }

// PrimaryOverride pins the boss-session id the office restored from
// session.json (the additive primarySeamBackend seam, called BEFORE
// Start). The pin reports immediately via PrimaryID and WINS over the
// init frame's own session_id when init eventually lands; an empty id
// leaves the wire pin in charge ("" = no override).
func (b *liveClaudeBackend) PrimaryOverride(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.primaryOverride = id
	if id != "" {
		b.primaryID = id
	}
}

// SetBypassPermissions — the office's bypass-permissions toggle (an
// ADDITIVE seam the app type-asserts, same convention as
// ConciergeCapable/SessionAborter in internal/state: never folded into
// state.Backend, harness stubs stay untouched). Pre-Start it latches the
// flag (nil): every later spawn rides `--dangerously-skip-permissions`
// and omits `--permission-prompt-tool stdio`. Once Start was called the
// instance's argv is fixed, so the call fails with "respawn required" —
// the app's toggle always builds a FRESH backend instead of mutating a
// running (or spent) one.
func (b *liveClaudeBackend) SetBypassPermissions(on bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return errors.New("bypass permissions: respawn required — SetBypassPermissions must be called before Start (the toggle respawns a fresh backend)")
	}
	b.bypassPermissions = on
	return nil
}

// ---------------------------------------------------------------- start

// claudeAbortSigIntAfter / claudeAbortSigTermAfter — the kill ladder
// behind /stop's interrupt control_request: escalate to SIGINT, then
// SIGTERM. Vars so tests can shrink them.
var (
	claudeAbortSigIntAfter  = 3 * time.Second
	claudeAbortSigTermAfter = 6 * time.Second
)

// claudeStopDrain caps Stop()'s wait after closing stdin before the
// SIGTERM -> SIGKILL tail engages (contract: 30s drain).
var claudeStopDrain = 30 * time.Second

// claudeBin resolves the `claude` executable: the harness pin wins
// (THEBORINGOFFICE_CLAUDE_BIN), then PATH.
func claudeBin() (string, error) {
	if p := strings.TrimSpace(os.Getenv("THEBORINGOFFICE_CLAUDE_BIN")); p != "" {
		return p, nil
	}
	p, err := exec.LookPath("claude")
	if err != nil {
		return "", errors.New("claude CLI not found on PATH (install: https://claude.ai/code) and THEBORINGOFFICE_CLAUDE_BIN is unset")
	}
	return p, nil
}

// claudeConfigDir resolves the child's CLAUDE_CONFIG_DIR: the harness pin
// (THEBORINGOFFICE_CLAUDE_CONFIG) wins outright — an explicit sandbox
// opt-in for tests and isolation-minded members; otherwise the user's REAL
// ~/.claude, so an existing `claude` login (auth, settings, MCP roster)
// carries straight into office sessions (a fresh sandbox makes `claude -p`
// exit silently without one). Settings are NEVER passed via --settings
// (the office owns no settings.json). directory is retained for seam
// stability — the user's claude config is global, not per-project.
func claudeConfigDir(directory string) string {
	if p := strings.TrimSpace(os.Getenv("THEBORINGOFFICE_CLAUDE_CONFIG")); p != "" {
		return p
	}
	home := config.HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".claude")
}

// claudeChildEnv builds the child's environment EXPLICITLY (an allowlist):
// a partial inheritance invites the settings.json `env` blocks' half-
// applied overrides — the office controls the whole process env instead.
// ANTHROPIC_API_KEY / CLAUDE_CODE_OAUTH_TOKEN pass through when present
// (never into argv). THEBORINGOFFICE_CLAUDE_* harness vars pass through so
// the uishot/test stubs can be scripted. When the office's auto-commit
// flag is on, the four majdoor GIT_* vars are appended so the agent's own
// `git commit`s are authored by the majdoor.
func claudeChildEnv(directory string) []string {
	allow := []string{
		"HOME", "PATH", "LANG", "LC_ALL", "LC_CTYPE", "TERM", "USER", "SHELL",
		"TMPDIR", "TMP", "TEMP", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME",
		"SSH_AUTH_SOCK", "GIT_ASKPASS", "TZ",
		"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN",
	}
	var env []string
	for _, k := range allow {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "THEBORINGOFFICE_CLAUDE_") {
			env = append(env, kv)
		}
	}
	configDir := claudeConfigDir(directory)
	env = append(env, "CLAUDE_CONFIG_DIR="+configDir)
	env = gitx.WithMajdoorAuthorEnv(env) // majdoor-authored commits when the office flag is on; no-op otherwise
	// Best-effort ONLY: the dir usually exists already (~/.claude) and an
	// unreadable/un creatable path must never fail the spawn.
	_ = os.MkdirAll(configDir, 0o755)
	return env
}

// spawnClaude starts the claude process for one session. resumeID != ""
// rides `--resume <uuid>` (the next-Send respawn path). bypass selects
// the permission wiring (see below). The returned
// exitCh carries the process's eventual Wait result (the scan-era reaper
// owns cmd.Wait — callers never re-Wait).
func spawnClaude(bin, directory, resumeID string, bypass bool) (*exec.Cmd, io.WriteCloser, io.Reader, <-chan error, *cappedErrBuf, error) {
	// --permission-prompt-tool stdio is the permission-modal lifeline:
	// headless `claude -p` only wires canUseTool to the stdio control
	// channel when the flag rides the spawn (the CLI's own SDK spawn
	// builder pushes exactly this). Without it EVERY permission-requiring
	// tool is auto-denied and the denial lands as a tool_result error —
	// the office modal never appears. (--permission-mode acceptEdits /
	// bypassPermissions would SILENCE prompts instead — never used here.)
	//
	// bypass=true is the office's bypass-permissions toggle: the spawn
	// rides `--dangerously-skip-permissions` (verified present on the
	// installed CLI 2.1.247 — `claude --help`: "Bypass all permission
	// checks") and OMITS --permission-prompt-tool stdio: with every
	// permission check bypassed, canUseTool never fires, so the stdio
	// prompt tool would be dead weight (the real CLI tolerates both
	// flags together — verified 2.1.247 exits 0 with the pair — but the
	// contradictory pair is never spawned). Zero EvPermission events
	// arrive from a bypassed child; nothing is filtered office-side.
	argv := []string{"-p", "--input-format", "stream-json",
		"--output-format", "stream-json", "--verbose", "--include-partial-messages"}
	if bypass {
		argv = append(argv, "--dangerously-skip-permissions")
	} else {
		argv = append(argv, "--permission-prompt-tool", "stdio")
	}
	if resumeID != "" {
		argv = append(argv, "--resume", resumeID)
	}
	cmd := exec.Command(bin, argv...)
	if directory != "" {
		cmd.Dir = directory
	}
	cmd.Env = claudeChildEnv(directory)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("claude spawn failed: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("claude spawn failed: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("claude spawn failed: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("claude spawn failed: %w", err)
	}
	errBuf := newCappedErrBuf()
	go io.Copy(errBuf, stderr) // stderr only feeds the error buffer
	exitCh := make(chan error, 1)
	go func() { exitCh <- cmd.Wait() }()
	return cmd, stdin, stdout, exitCh, errBuf, nil
}

// cappedErrBuf bounds the stderr snapshot (first 4KB — plenty for an
// init-failure explanation).
type cappedErrBuf struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func newCappedErrBuf() *cappedErrBuf { return &cappedErrBuf{buf: &bytes.Buffer{}} }

func (c *cappedErrBuf) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.buf.Len() < 4096 {
		keep := 4096 - c.buf.Len()
		if keep > len(p) {
			keep = len(p)
		}
		c.buf.Write(p[:keep])
	}
	return len(p), nil
}

func (c *cappedErrBuf) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// Start spawns the claude process and seats the floor IMMEDIATELY — there
// is NO system/init wait: `claude -p` only emits init after the first
// stdin user message, so blocking on it parks every boot on a frame the
// protocol sends mid-conversation (and an init that never comes — e.g. a
// fresh sandbox config — must never fail the boot). Start returns nil the
// moment the child is wired: readLoop pins primaryID/resumeID whenever
// init actually arrives, and the death watch (never initDone) reports a
// child that exits early. A pre-Start PrimaryOverride pin wins over the
// wire id (see the seam).
func (b *liveClaudeBackend) Start(emit func(state.Event)) error {
	b.fl.setEmit(emit)
	bin := b.bin
	if bin == "" {
		var err error
		bin, err = claudeBin()
		if err != nil {
			return err
		}
	}
	b.mu.Lock()
	b.bin = bin
	b.started = true // the argv freezes here — SetBypassPermissions errors from now on
	bypass := b.bypassPermissions
	b.mu.Unlock()

	// Manager charter (oikonomos) for Claude Code runs FIRST, before the
	// spawn: `claude -p` reads <dir>/CLAUDE.md (cwd + ancestors) at launch,
	// so the office's import block must be on disk before the child boots —
	// the same ordering as opencode.go's charter pass running before server
	// resolution. Best-effort like the opencode charter: a failure surfaces
	// on the status line and never blocks the boot. There is NO spoilage
	// restart condition here (an opencode serve spoils its config at boot;
	// this child is spawned below and reads the fresh bytes at this very
	// launch), so the changed flag rides nothing. The NOTES are buffered
	// and emitted behind the fixed boot prefix below (hires + hint lines) —
	// TestClaudeStartSeatsFloorBeforeInit pins that prefix's order.
	_, claudeCharterNotes := EnsureClaudeCharter(b.directory)

	proc, stdin, stdout, exitCh, _, err := spawnClaude(bin, b.directory, "", bypass)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.proc, b.procStdin, b.procExit = proc, stdin, exitCh
	wait := make(chan struct{})
	b.procWait = wait
	primaryID := b.primaryID // a pre-Start override pin; "" until init lands
	bin2 := b.bin
	b.mu.Unlock()

	// Declare the rendered dialog kinds FIRST — the initialize control
	// request must precede the first user line (the stream-json contract:
	// "initialize is optional and normally the first line"), and no Send
	// can have happened yet (Start owns the only pre-Send window).
	b.writeInitialize()

	// Fixed seats FIRST (opencode parity): the boss and Mnemosyne (hr) are
	// always on the floor. The manager's ID is the pinned session id — ""
	// pre-init is fine: every roster lookup keys on the manager ROLE, and
	// the id stays metadata-only once init pins it.
	b.fl.emit(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: primaryID, Name: "manager", Role: state.RoleManager, Seat: "manager", Sprite: state.SpriteAtDesk,
	}})
	b.fl.emit(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "hr", Name: "hr", Role: state.RoleHR, Seat: "hr", Sprite: state.SpriteAtDesk,
	}})
	// Backend-name hint FIRST (the topbar/reducer latch reads this marker
	// "[theboringoffice] backend: <name>" — same contract as opencode.go's
	// boot and the /backend swap line): it precedes the capability line so
	// every later status can own the line without losing the name. Both
	// land BEFORE the reader starts, so the boot event order is
	// deterministic (hires, hints, then whatever the wire delivers).
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend: claudecode"})
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] live (claude) — " + bin2 + " | board: in-memory"})
	if bypass {
		// Same transparency convention as the majdoor/concierge-off boot
		// lines: a mode that silences every permission prompt is named
		// on the record, once, at boot.
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] bypass permissions: on (--dangerously-skip-permissions) — claude never asks"})
	}
	// The charter pass's notes ride right behind the fixed boot prefix,
	// still synchronously BEFORE the reader starts — deterministic boot
	// event order: hires, hints, charter notes, then whatever the wire
	// delivers.
	for _, n := range claudeCharterNotes {
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: n})
	}

	// The reader owns stdout from here; system/init pins the boss session
	// WHENEVER it arrives (typically after the first Send) — Start never
	// waits on it. The death watch parks on the child exit channel only.
	go b.readLoop(stdout)
	go b.watchProc(proc, exitCh, wait)
	return nil
}

// ---------------------------------------------------------------- stdout reader

// readLoop scans stdout JSONL with a 1MB frame cap, maps every frame via
// claude_events.go and emits. The FIRST system/init pins resumeID +
// primaryID and latches initDone (the respawn gate) WHENEVER it arrives —
// claude -p emits init only after the first stdin user line, so nothing
// upstream may block on it. A PrimaryOverride pin wins over the wire id.
// Unknown frames never log-spam; a malformed line earns ONE dim status.
func (b *liveClaudeBackend) readLoop(stdout io.Reader) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if b.fl.isStopped() {
			return
		}
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var raw claudeEvent
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			b.fl.emit(state.Event{Kind: state.EvStatus,
				Text: "[claude] malformed line skipped: " + trimTo(line, 60)})
			continue
		}
		if b.rawFrameHook != nil {
			b.rawFrameHook(raw)
		}
		b.mu.Lock()
		if raw.Type == "system" && raw.Subtype == "init" {
			b.initDone = true
			if b.resumeID == "" {
				b.resumeID = raw.SessionID
				b.primaryID = raw.SessionID
			}
			if b.primaryOverride != "" {
				b.primaryID = b.primaryOverride // the pre-init restore pin wins over the wire id
			}
		}
		evs := mapClaudeEvent(raw, b.ctx, nowMs())
		// Turn bookkeeping: every main-conversation user write expects a
		// result; each result settles one FIFO placeholder + one turn.
		if raw.Type == "result" && raw.ParentToolUseID == "" {
			if b.busyTurns > 0 {
				b.busyTurns--
			}
			if len(b.pendingBoss) > 0 {
				b.pendingBoss = b.pendingBoss[1:]
			}
			if b.busyTurns == 0 {
				b.interruptArm = false
			}
		}
		b.mu.Unlock()
		for _, e := range evs {
			// A pending question dialog carries its decoded pages on the
			// event — stash them so AnswerQuestion can key the CLI-native
			// result by question text (the hold in ctx is metadata-only).
			if e.Kind == state.EvQuestion && e.ToolState == "pending" && len(e.Questions) > 0 {
				b.mu.Lock()
				b.questionStash[e.QuestionID] = e.Questions
				b.mu.Unlock()
			}
			b.emitMapped(e)
		}
	}
}

// emitMapped routes one mapped event through the coalescing gates
// (identical spirit to the opencode backend's emitThought/emitChatStream).
// A COMPLETION pin (Pending:false, or the same id carried by a stream)
// deletes the stream's slot first — no stale trailing flush may land
// after the pinned text (the opencode backend's slot-delete at pin time).
func (b *liveClaudeBackend) emitMapped(e state.Event) {
	if e.Kind == state.EvThought {
		b.emitThought(e)
		return
	}
	if e.Kind == state.EvChatBoss && strings.HasPrefix(e.Msg.ID, "bossmsg-") {
		if !e.Msg.Pending {
			b.mu.Lock()
			delete(b.chatSlots, e.Msg.ID)
			b.mu.Unlock()
		}
		if e.Msg.Pending {
			b.emitChatStream(e)
			return
		}
	}
	// Browser-tool markers (⟦open-browser: URL⟧, the
	// ⟦browser-screenshot: URL⟧/⟦browser-snapshot: URL⟧ read-only
	// siblings, and the ⟦browser-action: URL | op⟧ MUTATING sibling)
	// never reach the transcript: scrub the PINNED text
	// (deltas stream raw — the pin supersedes the growing bubble, so the
	// transcript at rest is clean) and fire the requests (opencode
	// parity: maybeBossCompleted; one event kind per directive kind;
	// browser-action's reaction is the member's permission modal).
	if e.Kind == state.EvChatBoss && !e.Msg.Pending && e.Msg.Text != "" {
		e.Msg.Text = browsertools.Scrub(e.Msg.Text, b.browserBridge)
	}
	b.fl.emit(e)
}

// emitThought / emitChatStream — the per-id 150ms coalescing gate, the
// claude twin of the opencode backend's thought/chat slots.
func (b *liveClaudeBackend) emitThought(e state.Event) {
	if e.CallID == "" {
		b.fl.emit(e)
		return
	}
	now := nowMs()
	b.mu.Lock()
	slot := b.thoughtSlots[e.CallID]
	if slot == nil {
		slot = &thoughtSlot{}
		b.thoughtSlots[e.CallID] = slot
	}
	if e.Done {
		slot.pending = nil
		slot.lastAt = now
		delete(b.thoughtSlots, e.CallID)
		b.mu.Unlock()
		b.fl.emit(e)
		return
	}
	if now-slot.lastAt >= thoughtMinGapMs {
		slot.lastAt = now
		b.mu.Unlock()
		b.fl.emit(e)
		return
	}
	pending := e
	slot.pending = &pending
	if slot.ticking {
		b.mu.Unlock()
		return
	}
	slot.ticking = true
	wait := time.Duration(thoughtMinGapMs-(now-slot.lastAt)) * time.Millisecond
	b.mu.Unlock()
	b.fl.at(wait, func() { b.flushThought(e.CallID) })
}

func (b *liveClaudeBackend) flushThought(callID string) {
	b.mu.Lock()
	slot := b.thoughtSlots[callID]
	if slot == nil {
		b.mu.Unlock()
		return
	}
	slot.ticking = false
	pending := slot.pending
	slot.pending = nil
	if pending != nil {
		slot.lastAt = nowMs()
	}
	b.mu.Unlock()
	if pending != nil {
		b.fl.emit(*pending)
	}
}

func (b *liveClaudeBackend) emitChatStream(e state.Event) {
	if e.Msg.ID == "" {
		b.fl.emit(e)
		return
	}
	now := nowMs()
	b.mu.Lock()
	slot := b.chatSlots[e.Msg.ID]
	if slot == nil {
		slot = &thoughtSlot{}
		b.chatSlots[e.Msg.ID] = slot
	}
	if now-slot.lastAt >= thoughtMinGapMs {
		slot.lastAt = now
		b.mu.Unlock()
		b.fl.emit(e)
		return
	}
	pending := e
	slot.pending = &pending
	if slot.ticking {
		b.mu.Unlock()
		return
	}
	slot.ticking = true
	wait := time.Duration(thoughtMinGapMs-(now-slot.lastAt)) * time.Millisecond
	b.mu.Unlock()
	b.fl.at(wait, func() { b.flushChatStream(e.Msg.ID) })
}

func (b *liveClaudeBackend) flushChatStream(id string) {
	b.mu.Lock()
	slot := b.chatSlots[id]
	if slot == nil {
		b.mu.Unlock()
		return
	}
	slot.ticking = false
	pending := slot.pending
	slot.pending = nil
	if pending != nil {
		slot.lastAt = nowMs()
	}
	b.mu.Unlock()
	if pending != nil {
		b.fl.emit(*pending)
	}
}

// ---------------------------------------------------------------- process watch + respawn

// watchProc is the claude twin of watchServe: ONE goroutine per spawned
// process, parked on the exit channel (the reaper owns cmd.Wait). An
// UNEXPECTED death latches died (the next Send respawns with --resume) and
// prints ONE status row; signal-terminated children (our own kill ladder,
// exit 130/143) are clean kills. Stop()-initiated exits stay silent.
func (b *liveClaudeBackend) watchProc(proc *exec.Cmd, exitCh <-chan error, wait chan struct{}) {
	err := <-exitCh
	b.mu.Lock()
	current := b.proc == proc
	if current {
		b.proc = nil
		b.procStdin = nil
	}
	stopping := b.stopping || b.fl.isStopped()
	if current && !stopping {
		b.died = true
	}
	b.mu.Unlock()
	close(wait)
	if !current || stopping {
		return
	}
	if es := exitStatus(err); es == 130 || es == 143 {
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
			"[claude] process terminated by signal (exit %d) — clean kill; your next send respawns the session", es)})
		return
	}
	if resume := b.resumeIDOrEmpty(); resume != "" {
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
			"[theboringoffice] claude process died (exited: %v) — your next send respawns it with --resume %s", err, shortTitle(resume, 24))})
		return
	}
	// No init pin yet (init arrives after the first Send) — there is no
	// session to resume; the death is reported plainly instead.
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
		"[theboringoffice] claude process died before system/init (exited: %v) — check `claude auth status` (CLAUDE_CONFIG_DIR defaults to ~/.claude so your login carries over)", err)})
}

func (b *liveClaudeBackend) resumeIDOrEmpty() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.resumeID
}

// exitStatus lifts the numeric exit code out of an *exec.ExitError (0 on
// nil or anything else).
func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 0
}

// respawnForSend is the send-side half of the death watch: the process
// died, this Send needs one — respawn NOW with `--resume <pinned uuid>`
// (the system/init pin, NEVER anything derived from a user Send), swap
// the plumbing, take the new watch, and note the respawn once.
func (b *liveClaudeBackend) respawnForSend() error {
	b.mu.Lock()
	resume := b.resumeID
	bin := b.bin
	bypass := b.bypassPermissions // a death-respawn keeps the boot's mode
	b.mu.Unlock()
	proc, stdin, stdout, exitCh, _, err := spawnClaude(bin, b.directory, resume, bypass)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.proc, b.procStdin, b.procExit = proc, stdin, exitCh
	b.died = false
	b.stopping = false
	wait := make(chan struct{})
	b.procWait = wait
	b.mu.Unlock()
	// The respawned process is a NEW client attach: re-declare the
	// rendered dialog kinds (the CLI also restores the prior epoch's
	// declaration — "[print.ts] restored N declared dialog kind(s)" — but
	// the re-initialize is the documented contract and costs one line).
	b.writeInitialize()
	go b.watchProc(proc, exitCh, wait)
	go b.readLoop(stdout) // the resume's own init lands as a mapped note
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
		"[claude] respawned with --resume %s", shortTitle(resume, 24))})
	return nil
}

// ---------------------------------------------------------------- send

// claudeUserLine is the EXACT stdin frame for one user chat message:
// {"type":"user","message":{"role":"user","content":[{"type":"text","text":...}]},"parent_tool_use_id":null}
// (field order pinned by struct layout — byte-stable on the wire).
type claudeUserLine struct {
	Type    string `json:"type"`
	Message struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	ParentToolUseID any `json:"parent_tool_use_id"`
}

func claudeUserLineFor(text string) []byte {
	var v claudeUserLine
	v.Type = "user"
	v.Message.Role = "user"
	v.Message.Content = []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{{Type: "text", Text: text}}
	body, _ := json.Marshal(v)
	return body
}

// writeLine writes ONE JSON line to the process stdin (single writer).
func (b *liveClaudeBackend) writeLine(body []byte) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	b.mu.Lock()
	stdin := b.procStdin
	b.mu.Unlock()
	if stdin == nil {
		return errors.New("claude process not running")
	}
	_, err := stdin.Write(append(body, '\n'))
	return err
}

// Send pushes user chat to the boss. NO IDLE GATE (opencode parity): the
// stdin line is written immediately, every time — claude queues a message
// that lands mid-turn behind the running turn. The local chat-user echo
// fires exactly once per prompt (2s same-text belt-and-braces); ONE
// pending boss placeholder stages per send; a dead process triggers the
// --resume respawn FIRST (never auto-respawned earlier).
func (b *liveClaudeBackend) Send(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || b.fl.isStopped() {
		return nil
	}
	b.mu.Lock()
	now := nowMs()
	duplicate := trimmed == b.lastUserText && b.lastUserText != "" && now-b.lastUserAt < 2000
	if !duplicate {
		b.lastUserText = trimmed
		b.lastUserAt = now
	}
	b.chatSeq++
	userID := "user-" + itoa(b.chatSeq)
	b.mu.Unlock()
	if !duplicate {
		b.fl.emit(state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
			ID: userID, From: "user", Text: trimmed, At: now, Kind: "user",
		}})
	}

	b.mu.Lock()
	respawn := b.died && b.initDone
	b.mu.Unlock()
	if respawn {
		if err := b.respawnForSend(); err != nil {
			b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[claude] respawn failed: " + shortTitle(err.Error(), 100)})
		}
	}

	b.mu.Lock()
	ready := b.proc != nil && b.procStdin != nil && !b.fl.isStopped()
	b.mu.Unlock()
	if !ready {
		b.mu.Lock()
		b.chatSeq++
		deadID := "boss-" + itoa(b.chatSeq)
		b.mu.Unlock()
		b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: deadID, From: "boss", Text: "[theboringoffice] backend not started", At: nowMs(), Pending: false,
		}})
		return nil
	}
	b.mu.Lock()
	b.chatSeq++
	pendingID := "boss-" + itoa(b.chatSeq)
	b.pendingBoss = append(b.pendingBoss, pendingID)
	b.mu.Unlock()

	b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: pendingID, From: "boss", Text: "", At: nowMs(), Pending: true,
	}})

	// The browser-tool preamble (browsertools.PromptPreamble) rides the
	// FIRST stdin user line exactly once — the conciergePreamble house
	// pattern; the member's chat-user echo above carries `trimmed` only.
	// The latch sets ONLY on a clean write (a failed line re-briefs next
	// Send), and a --resume respawn keeps it (the resumed session's
	// context retains the contract).
	b.mu.Lock()
	briefed := b.briefed
	b.mu.Unlock()
	line := claudeUserLineFor(trimmed)
	if !briefed {
		line = claudeUserLineFor(browsertools.PromptPreamble + "\n\n" + trimmed)
	}
	if err := b.writeLine(line); err != nil {
		b.mu.Lock()
		for i, id := range b.pendingBoss {
			if id == pendingID {
				b.pendingBoss = append(b.pendingBoss[:i], b.pendingBoss[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: pendingID, From: "boss",
			Text: "[theboringoffice] prompt failed: " + shortTitle(err.Error(), 120),
			At:   nowMs(), Pending: false,
		}})
		return nil
	}
	b.mu.Lock()
	b.busyTurns++
	if !briefed {
		b.briefed = true // the preamble's line landed — never brief this session twice
	}
	b.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------- control writers

// claudeControlResult is the inner response blob of a control_response.
// The CLI's vocabulary is PER-SUBTYPE (verified against the installed CLI
// 2.1.247 binary's zod schemas):
//   - can_use_tool: behavior ∈ {"allow","deny"} ONLY — "allow_always" does
//     not exist anywhere in the CLI binary, and a schema-parse failure on
//     the reply rejects the parked promise (the CLI then converts it to a
//     deny). message rides the deny reason; updatedPermissions rides the
//     standing grant an "allow always" gives (the request's
//     permission_suggestions re-emitted verbatim, the CLI-native
//     PermissionUpdate[] shape).
//   - request_user_dialog: behavior ∈ {"completed","cancelled"} ONLY — the
//     SDK schema is {behavior: enum("completed","cancelled"),
//     result?: unknown}, and the CLI's own cancel writer emits exactly
//     {behavior:"cancelled"}. result carries the answer payload (opaque per
//     dialog_kind; the CLI safeParses it against the kind's own result
//     schema). "allow"/"deny" here are schema violations the CLI buckets as
//     behavior "other" — the dialog never settles as answered. The result
//     SHAPE is per-kind (claudeDialogResultJSON): an OBJECT carrying
//     "behavior" for the permission_ask_user_question + F1 gates, a bare
//     enum STRING for the consent kinds, a fixed object for the
//     structured kinds; Result is raw JSON so every shape (and its
//     omission on cancelled) serializes byte-exactly.
type claudeControlResult struct {
	Behavior           string          `json:"behavior"`
	Result             json.RawMessage `json:"result,omitempty"`
	Message            string          `json:"message,omitempty"`
	UpdatedPermissions json.RawMessage `json:"updatedPermissions,omitempty"`
}

// claudeControlResponse is the EXACT stdin control_response shape:
// {"type":"control_response","response":{"subtype":"success","request_id":"…","response":{"behavior":"…",…}}}
// The envelope's subtype:"success" is REQUIRED — the CLI's own response
// writers always include it; an envelope without it fails the same
// schema parse and the parked permission converts to a deny.
type claudeControlResponse struct {
	Type     string `json:"type"`
	Response struct {
		Subtype   string              `json:"subtype"`
		RequestID string              `json:"request_id"`
		Response  claudeControlResult `json:"response"`
	} `json:"response"`
}

func claudeControlResponseFor(requestID string, res claudeControlResult) ([]byte, error) {
	if requestID == "" {
		return nil, errors.New("control_response needs a request_id")
	}
	var v claudeControlResponse
	v.Type = "control_response"
	v.Response.Subtype = "success"
	v.Response.RequestID = requestID
	v.Response.Response = res
	return json.Marshal(v)
}

// claudeInterruptLine is the abort control_request shape (field order
// pinned by struct layout — byte-stable on the wire):
// {"type":"control_request","request_id":"office-interrupt-N","request":{"subtype":"interrupt"}}
func claudeInterruptLine(seq int) []byte {
	var v struct {
		Type      string `json:"type"`
		RequestID string `json:"request_id"`
		Request   struct {
			Subtype string `json:"subtype"`
		} `json:"request"`
	}
	v.Type = "control_request"
	v.RequestID = fmt.Sprintf("office-interrupt-%d", seq)
	v.Request.Subtype = "interrupt"
	body, _ := json.Marshal(v)
	return body
}

// claudeInitializeLine is the initialize control_request the office writes
// as the FIRST stdin line of every claude process (and every --resume
// respawn) — field order pinned by struct layout, byte-stable on the wire:
//
//	{"type":"control_request","request_id":"office-init-N",
//	 "request":{"subtype":"initialize","supportedDialogKinds":[...]}}
//
// Binary evidence (CLI 2.1.247): the stream-json StdinMessage doc says
// "initialize is optional and normally the first line; the first user
// message initializes with defaults"; the SDK's own consumer builds
// {subtype:"initialize",…,supportedDialogKinds} and wraps it as
// {request_id, type:"control_request", request} (the same envelope the
// office's interrupt line uses); the CLI's initialize handler records
// initialize.supportedDialogKinds into declared_dialog_kinds, and the
// dialog host fails closed for undeclared kinds ("A kind is only sent in
// sessions where some attached client declared it in
// initialize.supportedDialogKinds"). The office declares EXACTLY the
// kinds claudeRenderedDialogKinds renders — never more (a kind declared
// but unrenderable would park a dialog the boss can never answer).
func claudeInitializeLine(seq int, kinds []string) []byte {
	var v struct {
		Type      string `json:"type"`
		RequestID string `json:"request_id"`
		Request   struct {
			Subtype              string   `json:"subtype"`
			SupportedDialogKinds []string `json:"supportedDialogKinds"`
		} `json:"request"`
	}
	v.Type = "control_request"
	v.RequestID = fmt.Sprintf("office-init-%d", seq)
	v.Request.Subtype = "initialize"
	v.Request.SupportedDialogKinds = kinds
	body, _ := json.Marshal(v)
	return body
}

// writeInitialize declares the rendered dialog kinds to a freshly spawned
// process (Start and the --resume respawn path). Best-effort: a failed
// write earns ONE status line and the session continues undeclared (the
// CLI then never sends the gated dialog kinds — fail-closed, never a
// stuck dialog). The CLI's control_response answer is ignored by the
// read loop's open-ended mapping ("control_response" -> nil).
func (b *liveClaudeBackend) writeInitialize() {
	b.mu.Lock()
	b.initSeq++
	seq := b.initSeq
	b.mu.Unlock()
	if err := b.writeLine(claudeInitializeLine(seq, claudeRenderedDialogKinds)); err != nil {
		b.fl.emit(state.Event{Kind: state.EvStatus,
			Text: "[claude] initialize (supportedDialogKinds) write failed: " + shortTitle(err.Error(), 100)})
	}
}

// AnswerPermission replies to a pending control_request can_use_tool:
// once->allow, always->allow + updatedPermissions (the CLI-native
// standing grant: the request's stashed permission_suggestions re-emitted
// verbatim; plain allow when the request carried none), reject->deny + a
// message. ONE stdin line; the local resolved event closes the modal
// (claude never echoes control_responses).
func (b *liveClaudeBackend) AnswerPermission(permissionID, response string) error {
	var res claudeControlResult
	switch response {
	case "once":
		res.Behavior = "allow"
	case "always":
		res.Behavior = "allow"
		b.mu.Lock()
		meta, hasMeta := b.ctx.permMeta[permissionID]
		b.mu.Unlock()
		if hasMeta && len(meta.Suggestions) > 0 && string(meta.Suggestions) != "null" {
			res.UpdatedPermissions = meta.Suggestions
		}
	case "reject":
		res.Behavior = "deny"
		res.Message = "Denied by the boss in theboringoffice"
	default:
		return fmt.Errorf("invalid permission response %q (want once|always|reject)", response)
	}
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	body, err := claudeControlResponseFor(permissionID, res)
	if err != nil {
		return err
	}
	if err := b.writeLine(body); err != nil {
		return err
	}
	b.mu.Lock()
	hold, ok := b.ctx.pendingPerms[permissionID]
	if ok {
		delete(b.ctx.pendingPerms, permissionID)
	}
	delete(b.ctx.permMeta, permissionID)
	b.mu.Unlock()
	if ok {
		b.fl.emit(state.Event{Kind: state.EvPermission, PermissionID: permissionID,
			SessionID: hold.SessionID, EmployeeID: hold.EmployeeID, EmployeeName: hold.EmployeeName,
			ToolName: hold.Title, ToolSummary: response, ToolState: "resolved"})
	} else {
		b.fl.emit(state.Event{Kind: state.EvPermission, PermissionID: permissionID, ToolState: "resolved"})
	}
	return nil
}

// claudeAskUserDialogQuestion is one questions[] entry as re-emitted in
// the dialog result's updatedInput (the wire field order of the request
// payload: question/header/options/multiSelect).
type claudeAskUserDialogQuestion struct {
	Question string `json:"question"`
	Header   string `json:"header"`
	Options  []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
		// Preview — the option's markdown preview, re-emitted verbatim.
		// omitempty: an absent preview keeps the re-emitted option bytes
		// identical to the pre-preview protocol.
		Preview string `json:"preview,omitempty"`
	} `json:"options"`
	MultiSelect bool `json:"multiSelect"`
}

// claudeAskUserDialogResult is the EXACT object the CLI's own permission
// component produces as the dialog result for dialog_kind
// permission_ask_user_question (CLI 2.1.247 binary evidence):
//   - the kind's registration validates result as an object carrying a
//     "behavior" key: zor=el({kind:"permission_ask_user_question",…,
//     result:ae(()=>ml((e)=>typeof e==="object"&&e!==null&&("behavior"in e))),
//     default:{behavior:"cancelled"}});
//   - the interactive submit builder returns
//     {behavior:"allow",updatedInput:{...input,answers:o,annotations:s}}
//     and the quick-answer path
//     {behavior:"allow",updatedInput:{...input,answers:{[question]:label}}};
//   - the dialog host wraps that as {behavior:"completed",result:r};
//   - the permission flow applies it REPLACE-wise: a.updatedInput??t —
//     so updatedInput must carry the original questions array or the
//     AskUserQuestion tool input schema (questions min 1) would fail;
//   - answers is keyed by QUESTION TEXT (the UI maps key:t.question; the
//     tool's output schema describes "question text -> answer string;
//     multi-select answers are comma-separated").
type claudeAskUserDialogResult struct {
	Behavior     string `json:"behavior"`
	UpdatedInput struct {
		Questions []claudeAskUserDialogQuestion `json:"questions"`
		Answers   map[string]string             `json:"answers"`
		// MetadataSource/Metadata — the request payload's analytics
		// context, re-emitted byte-verbatim when the wire carried them
		// (the CLI's own submit builder spreads the original tool input,
		// metadata included). omitempty: absence keeps the bytes
		// identical to the pre-metadata protocol.
		MetadataSource json.RawMessage `json:"metadataSource,omitempty"`
		Metadata       json.RawMessage `json:"metadata,omitempty"`
	} `json:"updatedInput"`
}

// claudeAskUserResultJSON builds the dialog result bytes for one
// permission_ask_user_question answer: behavior "allow" + updatedInput
// re-emitting the stashed questions and the boss's selections as an
// answers map keyed by question text (multi-select selections join ", ",
// exactly the CLI's own join — hss: n=r===0?o:n+", "+o). answers[i] pairs
// with items[i] (the office's page order IS the wire question order).
func claudeAskUserResultJSON(items []state.QuestionItem, answers [][]string) (json.RawMessage, error) {
	var res claudeAskUserDialogResult
	res.Behavior = "allow"
	res.UpdatedInput.Answers = make(map[string]string, len(items))
	for i, item := range items {
		var q claudeAskUserDialogQuestion
		q.Question = item.Question
		q.Header = item.Header
		q.MultiSelect = item.Multiple
		for _, opt := range item.Options {
			q.Options = append(q.Options, struct {
				Label       string `json:"label"`
				Description string `json:"description"`
				Preview     string `json:"preview,omitempty"`
			}{Label: opt.Label, Description: opt.Description, Preview: opt.Preview})
		}
		res.UpdatedInput.Questions = append(res.UpdatedInput.Questions, q)
		if i < len(answers) && len(answers[i]) > 0 {
			res.UpdatedInput.Answers[item.Question] = strings.Join(answers[i], ", ")
		}
		// The payload-level analytics context rides EVERY page of the
		// dialog (mapClaudeControlRequest copies it onto each item) —
		// the first carrier wins.
		if res.UpdatedInput.Metadata == nil && len(item.Meta) > 0 {
			res.UpdatedInput.Metadata = item.Meta
		}
		if res.UpdatedInput.MetadataSource == nil && len(item.MetaSource) > 0 {
			res.UpdatedInput.MetadataSource = item.MetaSource
		}
	}
	return json.Marshal(res)
}

// claudeDialogResultJSON builds the result bytes for one answered dialog
// from its stashed per-kind decode (claude_events.go writes the stash at
// map time). The CLI safeParses result against the KIND's own schema and
// settles the dialog as the kind's default on ANY mismatch — so each
// family builds exactly its kind's shape:
//
//   - dialogFamilyAUQ: the CLI-native permission decision (see
//     claudeAskUserResultJSON).
//   - dialogFamilyPermission (F1): Allow once/Allow always ->
//     {behavior:"allow"} (+updatedInput re-emitting the payload's tool
//     input when the wire carried one — the CLI's own "yes" builder is
//     {behavior:"allow", updatedInput:t.input}); Reject ->
//     {behavior:"deny", message:"Denied by the boss in theboringoffice"}.
//   - dialogFamilyLabelResult (F2 + F3 label kinds): the picked option's
//     prebuilt bytes (a bare enum STRING for F2, a fixed object for F3).
//   - dialogFamilyFlaggedAllow: {toRemove:[...picked rules...]} ("Remove
//     them all" expands to every flagged rule; no picks -> []).
//
// A picked label the map does not know (the modal's free-text row on a
// label-select kind) fails CLOSED: an error, no write — the dialog stays
// parked for the CLI's deadline rather than settling on garbage.
func claudeDialogResultJSON(meta claudeDialogMeta, items []state.QuestionItem, answers [][]string) (json.RawMessage, error) {
	// firstSelection flattens the modal's [][]string answer into the one
	// picked label of a single-page dialog ("" when nothing was picked).
	firstSelection := func() string {
		for _, page := range answers {
			if len(page) > 0 {
				return page[0]
			}
		}
		return ""
	}
	switch meta.family {
	case dialogFamilyAUQ:
		return claudeAskUserResultJSON(items, answers)
	case dialogFamilyPermission:
		switch firstSelection() {
		case claudeDialogAllowOnce, claudeDialogAllowAlways:
			// The always leg attaches NO permissionUpdates: the F1 payload
			// carries no permission_suggestions to re-emit (documented on
			// claudeRenderPermissionDialog) — a plain allow is the only
			// faithful settle.
			if len(meta.input) > 0 && string(meta.input) != "null" {
				return json.RawMessage(`{"behavior":"allow","updatedInput":` + string(meta.input) + `}`), nil
			}
			return json.RawMessage(`{"behavior":"allow"}`), nil
		case claudeDialogReject:
			return json.RawMessage(`{"behavior":"deny","message":"Denied by the boss in theboringoffice"}`), nil
		}
		return nil, fmt.Errorf("claude dialog %q: unknown permission answer %q", meta.kind, shortTitle(firstSelection(), 40))
	case dialogFamilyLabelResult:
		sel := firstSelection()
		raw, ok := meta.resultByLabel[sel]
		if !ok {
			return nil, fmt.Errorf("claude dialog %q: answer %q is not one of the rendered options", meta.kind, shortTitle(sel, 40))
		}
		return raw, nil
	case dialogFamilyFlaggedAllow:
		removeAll := false
		var picked []string
		seen := map[string]bool{}
		for _, page := range answers {
			for _, sel := range page {
				if sel == claudeFlaggedAllowRemoveAll {
					removeAll = true
					continue
				}
				if !seen[sel] {
					seen[sel] = true
					picked = append(picked, sel)
				}
			}
		}
		if removeAll {
			picked = append([]string(nil), meta.flagged...)
		}
		if picked == nil {
			picked = []string{}
		}
		var b bytes.Buffer
		b.WriteString(`{"toRemove":[`)
		for i, rule := range picked {
			if i > 0 {
				b.WriteByte(',')
			}
			raw, _ := json.Marshal(rule)
			b.Write(raw)
		}
		b.WriteString(`]}`)
		return b.Bytes(), nil
	}
	return nil, fmt.Errorf("claude dialog %q: unknown family %d", meta.kind, meta.family)
}

// AnswerQuestion replies to a pending control_request request_user_dialog:
// the dialog's response vocabulary is behavior "completed" with the answer
// payload riding response.result (the CLI safeParses result against the
// dialog kind's OWN schema — for the AskUserQuestion kind that is the
// permission decision {behavior:"allow",updatedInput:{questions,answers}};
// for the enum kinds a bare STRING; a shape the kind's schema rejects
// settles the dialog as its default — the parked turn never gets its
// answers). answers is one string array per asked question, in page order.
// The per-kind builder dispatches on the mapper's stashed claudeDialogMeta.
func (b *liveClaudeBackend) AnswerQuestion(requestID string, answers [][]string) error {
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	b.mu.Lock()
	items, stashed := b.questionStash[requestID]
	meta, hasMeta := b.ctx.dialogMeta[requestID]
	b.mu.Unlock()
	if !stashed {
		return fmt.Errorf("claude dialog %q has no stashed questions (already settled or never rendered)", shortTitle(requestID, 32))
	}
	if !hasMeta {
		// a stash without its meta can only come from a pre-table mapper
		// build — treat it as the AskUserQuestion kind (the only kind that
		// mapper ever rendered) so the CLI-native result still builds.
		meta = claudeDialogMeta{kind: claudeAskUserDialogKind, family: dialogFamilyAUQ}
	}
	raw, err := claudeDialogResultJSON(meta, items, answers)
	if err != nil {
		return err
	}
	body, err := claudeControlResponseFor(requestID, claudeControlResult{
		Behavior: "completed", Result: raw,
	})
	if err != nil {
		return err
	}
	if err := b.writeLine(body); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.questionStash, requestID)
	delete(b.ctx.dialogMeta, requestID)
	hold, ok := b.ctx.pendingQuestions[requestID]
	if ok {
		delete(b.ctx.pendingQuestions, requestID)
	}
	b.mu.Unlock()
	if ok {
		b.fl.emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
			SessionID: hold.SessionID, EmployeeID: hold.EmployeeID, EmployeeName: hold.EmployeeName,
			ToolSummary: "answered", ToolState: "resolved"})
	} else {
		b.fl.emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
			ToolSummary: "answered", ToolState: "resolved"})
	}
	return nil
}

// RejectQuestion declines a pending dialog outright (behavior "cancelled" —
// the CLI's own cancelDialogByMachine writer emits exactly
// {behavior:"cancelled"}; a dismissal settles the dialog and frees the
// parked turn). Envelope-cancelled is the correct dismissal for EVERY
// dialog kind: it carries NO result, and the CLI substitutes the kind's
// registered default (for the enum kinds their dismissal value — e.g.
// "not_now"/"deferred_no_consent_surface" — never a fabricated answer).
func (b *liveClaudeBackend) RejectQuestion(requestID string) error {
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	body, err := claudeControlResponseFor(requestID, claudeControlResult{Behavior: "cancelled"})
	if err != nil {
		return err
	}
	if err := b.writeLine(body); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.questionStash, requestID)
	delete(b.ctx.dialogMeta, requestID)
	hold, ok := b.ctx.pendingQuestions[requestID]
	if ok {
		delete(b.ctx.pendingQuestions, requestID)
	}
	b.mu.Unlock()
	if ok {
		b.fl.emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
			SessionID: hold.SessionID, EmployeeID: hold.EmployeeID, EmployeeName: hold.EmployeeName,
			ToolSummary: "rejected", ToolState: "resolved"})
	} else {
		b.fl.emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
			ToolSummary: "rejected", ToolState: "resolved"})
	}
	return nil
}

// ---------------------------------------------------------------- abort (/stop)

var _ state.SessionAborter = (*liveClaudeBackend)(nil)

// AbortSessions is the live /stop contract for the claude backend:
//  1. send ONE interrupt control_request (first-class teardown — the turn
//     stops cleanly with a result recorded);
//  2. flush open boss streams as "[theboringoffice] stream interrupted" and
//     close the FIFO-head placeholder with the stopped marker;
//  3. if the turn is STILL live after claudeAbortSigIntAfter, SIGINT the
//     process; if still live after claudeAbortSigTermAfter, SIGTERM —
//     exit 130/143 is a CLEAN kill (the watch says so, never "died").
//
// Nothing running is not an error; a second AbortSessions while an
// interrupt is already in flight re-sends nothing.
func (b *liveClaudeBackend) AbortSessions() error {
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	b.mu.Lock()
	proc := b.proc
	busy := b.busyTurns
	armed := b.interruptArm
	b.mu.Unlock()
	if proc == nil || busy == 0 {
		return nil // nothing running; not an error
	}
	if !armed {
		b.mu.Lock()
		b.interruptSeq++
		seq := b.interruptSeq
		b.interruptArm = true
		b.mu.Unlock()
		if err := b.writeLine(claudeInterruptLine(seq)); err != nil {
			b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[claude] interrupt write failed: " + shortTitle(err.Error(), 100)})
		} else {
			b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] turn abort: interrupt sent to claude"})
		}
	}

	// Post-abort tidy (opencode parity): flush open streams, close the
	// FIFO head placeholder with the stopped marker.
	b.mu.Lock()
	streamEvs := claudeInterruptedStreamEvents(b.ctx, "[theboringoffice] stream interrupted", nowMs())
	for id := range b.chatSlots {
		delete(b.chatSlots, id)
	}
	var headID string
	if len(b.pendingBoss) > 0 {
		headID = b.pendingBoss[0]
		b.pendingBoss = b.pendingBoss[1:]
	}
	b.mu.Unlock()
	for _, e := range streamEvs {
		b.fl.emit(e)
	}
	if headID != "" {
		b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID:      headID,
			From:    "boss",
			Text:    "[theboringoffice] stopped (turn aborted)",
			At:      nowMs(),
			Pending: false,
		}})
	}

	// Kill ladder: interrupt didn't settle the turn — escalate.
	procRef := proc
	b.fl.at(claudeAbortSigIntAfter, func() {
		b.mu.Lock()
		alive := b.proc == procRef && b.proc != nil && b.busyTurns > 0
		b.mu.Unlock()
		if !alive {
			return
		}
		_ = procRef.Process.Signal(syscall.SIGINT)
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[claude] turn abort: SIGINT escalate"})
		b.fl.at(claudeAbortSigTermAfter-claudeAbortSigIntAfter, func() {
			b.mu.Lock()
			alive := b.proc == procRef && b.proc != nil && b.busyTurns > 0
			b.mu.Unlock()
			if !alive {
				return
			}
			_ = procRef.Process.Signal(syscall.SIGTERM)
			b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[claude] turn abort: SIGTERM escalate"})
		})
	})
	return nil
}

// ---------------------------------------------------------------- MCP

// MCPServers lists the MCP servers the LAST system/init reported
// (init.mcp_servers[] only — a running claude never re-lists).
func (b *liveClaudeBackend) MCPServers() ([]state.MCPServer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]state.MCPServer, len(b.ctx.mcp))
	copy(out, b.ctx.mcp)
	return out, nil
}

// ReconnectMCP — the claude CLI exposes NO live MCP reconnect: the office
// respawns the session instead (the next Send's respawn path does the
// actual work; nothing is faked here).
func (b *liveClaudeBackend) ReconnectMCP(name string) error {
	b.mu.Lock()
	proc := b.proc
	b.mu.Unlock()
	if proc == nil {
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
			"[claude] MCP %s reconnect is respawn-only — the process is already down; the next send respawns it", name)})
		return nil
	}
	_ = syscall.Kill(proc.Process.Pid, syscall.SIGTERM)
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
		"[claude] reconnecting MCP %s: process respawn requested — the next send respawns with --resume", name)})
	return nil
}

// ---------------------------------------------------------------- stop

// Stop: close stdin (claude exits), wait up to claudeStopDrain, then
// SIGTERM, then SIGKILL. Open streams flush interrupted FIRST (before
// fl.stop seals emit) — opencode parity.
func (b *liveClaudeBackend) Stop() error {
	if b.fl.isStopped() {
		return nil
	}
	b.mu.Lock()
	streamEvs := claudeInterruptedStreamEvents(b.ctx, "[theboringoffice] stream interrupted", nowMs())
	for id := range b.chatSlots {
		delete(b.chatSlots, id)
	}
	b.stopping = true
	b.mu.Unlock()
	for _, e := range streamEvs {
		b.fl.emit(e)
	}

	b.fl.stop()

	b.mu.Lock()
	proc := b.proc
	stdin := b.procStdin
	wait := b.procWait
	b.proc = nil
	b.procStdin = nil
	b.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if proc == nil || proc.Process == nil {
		return nil
	}
	if wait != nil {
		select {
		case <-wait:
			return nil
		case <-time.After(claudeStopDrain):
		}
	}
	_ = proc.Process.Signal(syscall.SIGTERM)
	if wait != nil {
		select {
		case <-wait:
			return nil
		case <-time.After(stopKillGrace):
		}
	}
	_ = proc.Process.Kill()
	reaped := make(chan struct{})
	go func() { _ = proc.Wait(); close(reaped) }()
	select {
	case <-reaped:
	case <-time.After(stopKillGrace):
	}
	return nil
}
