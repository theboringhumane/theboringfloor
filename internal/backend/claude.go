// claude.go — the live Claude Code backend for theboringoffice: one
// `claude -p --input-format stream-json --output-format stream-json
// --verbose --include-partial-messages` process per office session (a
// per-session process, NOT a daemon), with the stdout JSONL stream
// normalized into state.Events via claude_events.go (pure, mirroring how
// opencode.go + events.go share work).
//
// Session identity: the boss conversation's claude session_id pins at the
// FIRST system/init (never from a user Send) and becomes the ONE resume
// pin: a dead process is never auto-respawned, but the NEXT user Send
// respawns with `--resume <pinned uuid>` (mid-turn kill recovery).
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
// (once->allow / always->allow_always / reject->deny) and emit the local
// resolved event the modal needs (claude never echoes responses).
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

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// liveClaudeBackend is the claude CLI twin of liveBackend (opencode.go).
type liveClaudeBackend struct {
	directory string
	cfg       *config.Config
	bin       string
	fl        *flow

	mu           sync.Mutex
	proc         *exec.Cmd
	procStdin    io.WriteCloser
	procExit     <-chan error
	procWait     chan struct{} // closed when the current proc's watch drains
	initDone     bool
	stopping     bool // Stop() engaged: the watch stays silent
	died         bool // the watch latched a death; next Send respawns
	resumeID     string
	primaryID    string
	busyTurns    int // outstanding turns (user writes without a result)
	chatSeq      int
	pendingBoss  []string
	interruptSeq int
	interruptArm bool // an interrupt is already in flight for the live turn(s)

	writeMu sync.Mutex // single-writer stdin guard

	ctx          *claudeNormCtx
	thoughtSlots map[string]*thoughtSlot
	chatSlots    map[string]*thoughtSlot

	lastUserText string
	lastUserAt   int64
}

var _ state.Backend = (*liveClaudeBackend)(nil)

// newClaudeBackend wires the backend; bin "" means "resolve at Start"
// (THEBORINGOFFICE_CLAUDE_BIN, then PATH's `claude`).
func newClaudeBackend(bin, directory string, cfg *config.Config) *liveClaudeBackend {
	return &liveClaudeBackend{
		directory:    directory,
		cfg:          cfgOrDefault(cfg),
		bin:          bin,
		fl:           newFlow(),
		ctx:          newClaudeNormCtx(cfgOrDefault(cfg)),
		thoughtSlots: make(map[string]*thoughtSlot),
		chatSlots:    make(map[string]*thoughtSlot),
	}
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

// ---------------------------------------------------------------- start

// claudeStartTimeout bounds the wait for the process's first system/init.
// A var so deadline tests can shrink it (sseBackoffSteps idiom).
var claudeStartTimeout = 10 * time.Second

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

// claudeConfigDir is the office-owned CLAUDE_CONFIG_DIR: the harness pin
// (THEBORINGOFFICE_CLAUDE_CONFIG) wins outright; otherwise
// <home>/.theboringoffice/claude/<project-hash> — project-keyed, so every
// office keeps its own claude profile sandbox. Settings are NEVER passed
// via --settings (the office owns no settings.json).
func claudeConfigDir(directory string) string {
	if p := strings.TrimSpace(os.Getenv("THEBORINGOFFICE_CLAUDE_CONFIG")); p != "" {
		return p
	}
	home := config.HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".theboringoffice", "claude", claudeDirKey(directory))
}

// claudeDirKey is the stable per-project key (FNV-1a of the cleaned path,
// base36 — short, collision-tolerant, and stable across boots).
func claudeDirKey(directory string) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	h := uint64(14695981039346656037)
	for i := 0; i < len(directory); i++ {
		h ^= uint64(directory[i])
		h *= 1099511628211
	}
	if h == 0 {
		return "root"
	}
	var out [16]byte
	i := len(out)
	for h > 0 {
		i--
		out[i] = digits[h%36]
		h /= 36
	}
	return string(out[i:])
}

// claudeChildEnv builds the child's environment EXPLICITLY (an allowlist):
// a partial inheritance invites the settings.json `env` blocks' half-
// applied overrides — the office controls the whole process env instead.
// ANTHROPIC_API_KEY / CLAUDE_CODE_OAUTH_TOKEN pass through when present
// (never into argv). THEBORINGOFFICE_CLAUDE_* harness vars pass through so
// the uishot/test stubs can be scripted.
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
	env = append(env, "CLAUDE_CONFIG_DIR="+claudeConfigDir(directory))
	_ = os.MkdirAll(claudeConfigDir(directory), 0o755) // profile sandbox must exist
	return env
}

// spawnClaude starts the claude process for one session. resumeID != ""
// rides `--resume <uuid>` (the next-Send respawn path). The returned
// exitCh carries the process's eventual Wait result (the scan-era reaper
// owns cmd.Wait — callers never re-Wait).
func spawnClaude(bin, directory, resumeID string) (*exec.Cmd, io.WriteCloser, io.Reader, <-chan error, *cappedErrBuf, error) {
	argv := []string{"-p", "--input-format", "stream-json",
		"--output-format", "stream-json", "--verbose", "--include-partial-messages"}
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

// Start spawns the claude process and resolves on the FIRST system/init
// (boss-session pin), a process exit (error), or claudeStartTimeout.
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
	b.mu.Unlock()

	proc, stdin, stdout, exitCh, errBuf, err := spawnClaude(bin, b.directory, "")
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.proc, b.procStdin, b.procExit = proc, stdin, exitCh
	wait := make(chan struct{})
	b.procWait = wait
	b.mu.Unlock()

	// The reader owns stdout from here; initCh hands the FIRST system/init
	// to this Start (processed or not, the resolver decides synchronously).
	initCh := make(chan claudeEvent, 1)
	go b.readLoop(stdout, initCh)

	select {
	case ev := <-initCh:
		// system/init landed: the mapper already pinned primaryID/model/mcp
		// (readLoop maps in order). Seat the floor, THEN take the death
		// watch (the watch must never race this resolution on exitCh).
		b.mu.Lock()
		b.initDone = true
		primaryID := b.primaryID
		b.mu.Unlock()
		b.fl.emit(state.Event{Kind: state.EvHire, Employee: state.Employee{
			ID: primaryID, Name: "manager", Role: state.RoleManager, Seat: "manager", Sprite: state.SpriteAtDesk,
		}})
		b.fl.emit(state.Event{Kind: state.EvHire, Employee: state.Employee{
			ID: "hr", Name: "hr", Role: state.RoleHR, Seat: "hr", Sprite: state.SpriteAtDesk,
		}})
		go b.watchProc(proc, exitCh, wait)
		b.mu.Lock()
		bin2 := b.bin
		b.mu.Unlock()
		// Backend-name hint FIRST (the topbar/reducer latch reads this
		// marker "[theboringoffice] backend: <name>" — same contract as
		// opencode.go's boot and the /backend swap line): it precedes the
		// capability line below so every later status can own the line
		// without losing the name.
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend: claudecode"})
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] live (claude) — " + bin2 + " | board: in-memory"})
		_ = ev
		return nil
	case err := <-exitCh:
		b.failSpawn(proc, wait)
		return fmt.Errorf("claude -p exited before init: %v: %s", err, trimTo(errBuf.String(), 200))
	case <-time.After(claudeStartTimeout):
		b.failSpawn(proc, wait)
		return errors.New("claude -p: no system/init within " + claudeStartTimeout.String())
	}
}

// failSpawn tears down a Start that never resolved (exit-before-init /
// init timeout): the child is killed best-effort, the wait channel is
// closed (watchProc was never armed on it, so its closure is ours to own
// — a Stop() after a failed Start must never park the claudeStopDrain on
// a channel nobody drains), and the backend's live refs are released.
func (b *liveClaudeBackend) failSpawn(proc *exec.Cmd, wait chan struct{}) {
	if proc != nil && proc.Process != nil {
		_ = proc.Process.Kill()
	}
	b.mu.Lock()
	if b.proc == proc {
		b.proc = nil
		b.procStdin = nil
	}
	b.mu.Unlock()
	close(wait)
}

// ---------------------------------------------------------------- stdout reader

// readLoop scans stdout JSONL with a 1MB frame cap, hands the first
// system/init to initCh, maps every frame via claude_events.go and emits.
// Unknown frames never log-spam; a malformed line earns ONE dim status.
func (b *liveClaudeBackend) readLoop(stdout io.Reader, initCh chan<- claudeEvent) {
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	initSent := false
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
		b.mu.Lock()
		if raw.Type == "system" && raw.Subtype == "init" && b.resumeID == "" {
			b.resumeID = raw.SessionID
			b.primaryID = raw.SessionID
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
		if raw.Type == "system" && raw.Subtype == "init" && !initSent {
			initSent = true
			select {
			case initCh <- raw:
			default:
			}
		}
		for _, e := range evs {
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
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: fmt.Sprintf(
		"[theboringoffice] claude process died (exited: %v) — your next send respawns it with --resume %s", err, shortTitle(b.resumeIDOrEmpty(), 24))})
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
	b.mu.Unlock()
	proc, stdin, stdout, exitCh, _, err := spawnClaude(bin, b.directory, resume)
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
	go b.watchProc(proc, exitCh, wait)
	go b.readLoop(stdout, make(chan claudeEvent, 1)) // second init lands as a mapped note
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

	if err := b.writeLine(claudeUserLineFor(trimmed)); err != nil {
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
	b.mu.Unlock()
	return nil
}

// ---------------------------------------------------------------- control writers

// claudeControlResponse is the EXACT stdin control_response shape:
// {"type":"control_response","response":{"request_id":"…","response":{"behavior":"…",…}}}
type claudeControlResponse struct {
	Type     string `json:"type"`
	Response struct {
		RequestID string `json:"request_id"`
		Response  struct {
			Behavior string `json:"behavior"`
			Answer   string `json:"answer,omitempty"`
		} `json:"response"`
	} `json:"response"`
}

func claudeControlResponseFor(requestID, behavior, answer string) ([]byte, error) {
	if requestID == "" {
		return nil, errors.New("control_response needs a request_id")
	}
	var v claudeControlResponse
	v.Type = "control_response"
	v.Response.RequestID = requestID
	v.Response.Response.Behavior = behavior
	v.Response.Response.Answer = answer
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

// AnswerPermission replies to a pending control_request can_use_tool:
// once->allow, always->allow (+always — behavior always rides through),
// reject->deny. ONE stdin line; the local resolved event closes the modal
// (claude never echoes control_responses).
func (b *liveClaudeBackend) AnswerPermission(permissionID, response string) error {
	var behavior string
	switch response {
	case "once":
		behavior = "allow"
	case "always":
		behavior = "allow_always" // the office's always grants the standing allow
	case "reject":
		behavior = "deny"
	default:
		return fmt.Errorf("invalid permission response %q (want once|always|reject)", response)
	}
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	body, err := claudeControlResponseFor(permissionID, behavior, "")
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

// AnswerQuestion replies to a pending control_request request_user_dialog:
// the answer text rides response.answer inside the dialog-shaped
// control_response (behavior allow). answers is one string array per asked
// question; the dialog's text is the pages joined "; " (selections ", ").
func (b *liveClaudeBackend) AnswerQuestion(requestID string, answers [][]string) error {
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	pages := make([]string, len(answers))
	for i, a := range answers {
		pages[i] = strings.Join(a, ", ")
	}
	body, err := claudeControlResponseFor(requestID, "allow", strings.Join(pages, "; "))
	if err != nil {
		return err
	}
	if err := b.writeLine(body); err != nil {
		return err
	}
	b.mu.Lock()
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

// RejectQuestion declines a pending dialog outright (behavior deny),
// freeing the parked turn.
func (b *liveClaudeBackend) RejectQuestion(requestID string) error {
	if b.fl.isStopped() {
		return errors.New("backend stopped")
	}
	body, err := claudeControlResponseFor(requestID, "deny", "")
	if err != nil {
		return err
	}
	if err := b.writeLine(body); err != nil {
		return err
	}
	b.mu.Lock()
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
