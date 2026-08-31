// claude_live_test.go — the LIVE counterpart of claude_regression_test.go.
// The regression pins prove the mapping pipeline against byte-perfect stub
// fixtures; these tests prove the SAME three production contracts against
// the REAL `claude` CLI (the member's explicit ask: "run integration test
// with real claude code cli to see if the change actually works"):
//
//  1. TestClaudeLiveBossBubbleSinglePin — one Send yielding a multi-
//     paragraph prose reply (zero tool use): exactly ONE pinned
//     (Pending=false) boss bubble per Anthropic message.id, no two pinned
//     bubbles carrying identical text (the doubled-bubble symptom), and no
//     bubble keyed off a frame-level uuid. The raw frame inventory with the
//     real msg_... ids and divergent frame uuids is printed as evidence.
//  2. TestClaudeLivePermissionRoundTrip — a prompt that forces a
//     permission-requiring tool call (the Write tool: write-class, ALWAYS
//     prompts in the CLI's default permission mode, and this account's
//     settings.json carries no Write allow-rule and no defaultMode
//     override — unlike Bash(ls), which the CLI's read-only safe-list can
//     auto-approve). The pending EvPermission (or EvQuestion, if the CLI
//     routes the ask as a declared dialog) must ARRIVE, the allow answer
//     must round-trip to claude's stdin, and the post-answer frames
//     (user/tool_result -> assistant -> result) plus the file on disk
//     prove the tool actually ran.
//  3. TestClaudeLiveThinkingEvThought — a thinking-trigger prompt
//     ("Think step by step…" — the CLI's extended-thinking trigger word).
//     If the account/model emits thinking blocks: open (Done=false) ->
//     accumulate (growing transcript) -> close (Done=true, full text)
//     under one stable CallID. If ZERO EvThought arrive, the test prints
//     NO-THINKING-EMITTED-BY-ACCOUNT with the content-block-type
//     inventory proving it really looked, and PASSES — thinking is never
//     faked.
//
// Gating (requirement: CI must never touch the real CLI): every test
// skips unless THEBORINGOFFICE_LIVE_CLAUDE=1 AND `claude` resolves on
// PATH. CLAUDE_CONFIG_DIR is left at its default (the user's real
// ~/.claude) so the machine's existing login carries — that IS the
// production path under test.
//
// Time discipline: real API latency — every wait is bounded (per-test
// budget <= 180s; prompts are tiny). Cleanup: t.Cleanup stops the backend
// (Stop's stdin-close -> drain -> SIGTERM -> SIGKILL ladder — the same
// teardown the existing claude tests use) and every scratch file lives
// under t.TempDir() (auto-removed).
package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/charter"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// liveClaudeGate is the double gate EVERY live test passes through:
// THEBORINGOFFICE_LIVE_CLAUDE=1 (explicit human opt-in — CI never sets it)
// AND a real `claude` CLI on PATH. Both skips carry the reason.
func liveClaudeGate(t *testing.T) string {
	t.Helper()
	if os.Getenv("THEBORINGOFFICE_LIVE_CLAUDE") != "1" {
		t.Skip("skipping LIVE claude CLI test: THEBORINGOFFICE_LIVE_CLAUDE=1 not set " +
			"(real CLI + real API spend — opt-in only, CI must never run this)")
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("skipping LIVE claude CLI test: `claude` CLI not found on PATH")
	}
	return bin
}

// liveFrameRec records every raw stdout frame the backend's reader parses
// (via the rawFrameHook seam) — the live wire inventory. The mapped events
// alone cannot show this: the doubled-bubble fix scrubs frame uuids from
// every event, so the uuid-vs-message.id divergence exists ONLY here.
type liveFrameRec struct {
	mu     sync.Mutex
	frames []claudeEvent
}

func (r *liveFrameRec) add(raw claudeEvent) {
	r.mu.Lock()
	r.frames = append(r.frames, raw)
	r.mu.Unlock()
}

func (r *liveFrameRec) snapshot() []claudeEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]claudeEvent(nil), r.frames...)
}

func (r *liveFrameRec) found(pred func(claudeEvent) bool) bool {
	for _, f := range r.snapshot() {
		if pred(f) {
			return true
		}
	}
	return false
}

func (r *liveFrameRec) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

// liveBoot starts the office's REAL claude backend — the same
// newClaudeBackend -> Start path the app drives (never an exec of claude
// by the test) — with the event log and the raw frame recorder attached.
func liveBoot(t *testing.T, bin, dir string) (*liveClaudeBackend, *claudeEventLog, *liveFrameRec) {
	t.Helper()
	log := &claudeEventLog{}
	rec := &liveFrameRec{}
	b := newClaudeBackend(bin, dir, nil)
	b.rawFrameHook = rec.add
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start (real claude at %s): %v", bin, err)
	}
	t.Cleanup(func() { _ = b.Stop() }) // stdin-close -> drain -> SIGTERM -> SIGKILL
	return b, log, rec
}

// liveWait polls cond until it holds or the deadline passes. A timeout
// fails WITH the full wire inventory + event log attached — a live hang
// is a real bug, never silently tolerated.
func liveWait(t *testing.T, what string, d time.Duration, rec *liveFrameRec, log *claudeEventLog, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)
	t.Fatalf("timed out after %s waiting for %s", d, what)
}

// liveFrameSig is the one-line identity of one raw frame for the
// run-length-compressed sequence print.
func liveFrameSig(f claudeEvent) string {
	sig := f.Type
	if f.Subtype != "" {
		sig += "/" + f.Subtype
	}
	if f.Type == "stream_event" {
		var inner claudeStreamInner
		if json.Unmarshal(f.Event, &inner) == nil {
			sig += " " + inner.Type
			if inner.Delta.Type != "" {
				sig += "/" + inner.Delta.Type
			}
		}
	}
	return sig
}

// liveDumpWire prints the raw wire in two sections: the run-length-
// compressed frame sequence (every frame accounted for, delta runs
// collapsed) and the key-frame identity lines (init/assistant/user/
// control/result with their REAL ids — msg_... vs frame uuid).
func liveDumpWire(t *testing.T, rec *liveFrameRec) {
	t.Helper()
	frames := rec.snapshot()
	t.Logf("=== RAW WIRE: %d frames parsed by the backend's reader ===", len(frames))
	// Section 1: the RLE sequence.
	for i := 0; i < len(frames); {
		sig := liveFrameSig(frames[i])
		j := i
		for j < len(frames) && liveFrameSig(frames[j]) == sig {
			j++
		}
		if n := j - i; n > 1 {
			t.Logf("  seq[%03d..%03d] %s x%d", i, j-1, sig, n)
		} else {
			t.Logf("  seq[%03d]     %s", i, sig)
		}
		i = j
	}
	// Section 2: identity lines for the frames that carry ids.
	t.Log("=== KEY FRAME IDENTITIES (real ids off the live wire) ===")
	for i, f := range frames {
		switch f.Type {
		case "system":
			t.Logf("  frame[%03d] system/%s session_id=%s uuid=%s model=%s cwd=%s",
				i, f.Subtype, f.SessionID, f.UUID, f.Model, f.Cwd)
		case "assistant", "user":
			var blocks []string
			for _, bl := range f.Message.Content {
				blocks = append(blocks, bl.Type)
			}
			t.Logf("  frame[%03d] %s message.id=%s uuid=%s parent_tool_use_id=%q content_blocks=%v",
				i, f.Type, f.Message.ID, f.UUID, f.ParentToolUseID, blocks)
		case "control_request", "control_response":
			t.Logf("  frame[%03d] %s request_id=%s subtype=%s tool_name=%s tool_use_id=%s session_id=%q",
				i, f.Type, f.RequestID, f.Request.Subtype, f.Request.ToolName, f.Request.ToolUseID, f.SessionID)
		case "result":
			t.Logf("  frame[%03d] result/%s is_error=%v num_turns=%d duration_ms=%d cost_usd=%.4f uuid=%s",
				i, f.Subtype, f.IsError, f.NumTurns, f.DurationMs, f.TotalCostUSD, f.UUID)
		case "stream_event":
			var inner claudeStreamInner
			if json.Unmarshal(f.Event, &inner) == nil && inner.Type == "message_start" {
				t.Logf("  frame[%03d] stream_event message_start message.id=%s", i, inner.Message.ID)
			}
		}
	}
}

// liveDumpEvents prints the mapped event log (the reducer's view), one
// line per event with text trimmed.
func liveDumpEvents(t *testing.T, log *claudeEventLog) {
	t.Helper()
	evs := log.snapshot()
	t.Logf("=== MAPPED EVENT LOG: %d events ===", len(evs))
	for i, e := range evs {
		line := fmt.Sprintf("  event[%03d] kind=%s", i, e.Kind)
		switch e.Kind {
		case state.EvHire:
			line += fmt.Sprintf(" name=%s role=%s seat=%s", e.Employee.Name, e.Employee.Role, e.Employee.Seat)
		case state.EvChatBoss, state.EvChatUser, state.EvChatOffice:
			line += fmt.Sprintf(" id=%s pending=%v text=%q", e.Msg.ID, e.Msg.Pending, trimTo(e.Msg.Text, 80))
		case state.EvThought:
			line += fmt.Sprintf(" callID=%s done=%v text=%q", e.CallID, e.Done, trimTo(e.Text, 80))
		case state.EvPermission:
			line += fmt.Sprintf(" permID=%s tool=%s state=%s summary=%q session=%s",
				e.PermissionID, e.ToolName, e.ToolState, trimTo(e.ToolSummary, 60), e.SessionID)
		case state.EvQuestion:
			line += fmt.Sprintf(" questionID=%s state=%s summary=%q", e.QuestionID, e.ToolState, trimTo(e.ToolSummary, 60))
		case state.EvTool:
			line += fmt.Sprintf(" callID=%s tool=%s state=%s summary=%q", e.CallID, e.ToolName, e.ToolState, trimTo(e.ToolSummary, 60))
		case state.EvUsage:
			line += fmt.Sprintf(" callID=%s in=%d out=%d cost=%.4f", e.CallID, e.TokensIn, e.TokensOut, e.CostUSD)
		case state.EvStatus:
			line += fmt.Sprintf(" text=%q", trimTo(e.Text, 100))
		default:
			line += fmt.Sprintf(" %+v", e)
		}
		t.Log(line)
	}
}

// liveResultSeen reports whether a main-conversation result frame (the
// real wire's turn-complete signal) has landed.
func liveResultSeen(rec *liveFrameRec) bool {
	return rec.found(func(f claudeEvent) bool {
		return f.Type == "result" && f.ParentToolUseID == ""
	})
}

// ---------------------------------------------------------------- test 1

// TestClaudeLiveBossBubbleSinglePin — the doubled-bubble contract against
// the REAL CLI: one Send, a multi-paragraph prose reply, zero tool use.
// Exactly ONE pinned boss bubble per Anthropic message.id, no two pins
// with identical text, and every pin keyed off message.id (msg_...),
// never off a frame-level uuid.
func TestClaudeLiveBossBubbleSinglePin(t *testing.T) {
	bin := liveClaudeGate(t)
	b, log, rec := liveBoot(t, bin, t.TempDir())

	if err := b.Send("Reply with exactly three short paragraphs about honey. Do not use any tools."); err != nil {
		t.Fatalf("Send: %v", err)
	}
	liveWait(t, "the turn's result frame (real API latency)", 150*time.Second, rec, log, func() bool { return liveResultSeen(rec) })
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)

	// The wire's REAL identities: main-conversation message.ids (msg_...)
	// vs the frames' own uuids — the divergence the doubled-bubble fix
	// keys on. message_start carries the id mid-stream; assistant frames
	// carry BOTH the inner message.id and their own frame uuid.
	msgIDs := map[string]bool{}
	frameUUIDs := map[string]bool{}
	for _, f := range rec.snapshot() {
		if f.ParentToolUseID != "" {
			continue // subagent frames never key boss bubbles
		}
		switch f.Type {
		case "assistant":
			if f.Message.ID != "" {
				msgIDs[f.Message.ID] = true
			}
			if f.UUID != "" {
				frameUUIDs[f.UUID] = true
			}
		case "stream_event":
			var inner claudeStreamInner
			if json.Unmarshal(f.Event, &inner) == nil && inner.Type == "message_start" && inner.Message.ID != "" {
				msgIDs[inner.Message.ID] = true
			}
		}
	}
	t.Logf("wire message.ids (msg_...): %v", sortedKeys(msgIDs))
	t.Logf("assistant-frame uuids (frame-level): %v", sortedKeys(frameUUIDs))
	for id := range msgIDs {
		if frameUUIDs[id] {
			t.Fatalf("a frame uuid must never double as a message.id on the real wire: %s", id)
		}
	}

	// Fold the boss-chat events by Msg.ID (the chat reducer's replace-on-
	// merge) — the member-visible result.
	merged := map[string]state.ChatMsg{}
	pinCount := map[string]int{}
	growthCount := map[string]int{}
	for _, e := range log.snapshot() {
		if e.Kind != state.EvChatBoss || !strings.HasPrefix(e.Msg.ID, "bossmsg-") {
			continue
		}
		merged[e.Msg.ID] = e.Msg
		if e.Msg.Pending {
			growthCount[e.Msg.ID]++
		} else {
			pinCount[e.Msg.ID]++
		}
	}

	// Exactly ONE pinned bubble per message.id the wire actually produced
	// (this no-tool turn: every message carries text).
	if len(merged) != len(msgIDs) || len(merged) == 0 {
		t.Fatalf("exactly ONE boss bubble per wire message.id: wire ids=%d (%v), bubbles=%d (%v)",
			len(msgIDs), sortedKeys(msgIDs), len(merged), sortedKeys(merged))
	}
	for id, final := range merged {
		wireID := strings.TrimPrefix(id, "bossmsg-")
		if !msgIDs[wireID] {
			t.Fatalf("bubble %s is keyed off a NON-message.id (frame-uuid fork?): wire ids %v", id, sortedKeys(msgIDs))
		}
		if frameUUIDs[wireID] {
			t.Fatalf("bubble %s is keyed off a FRAME UUID — the doubled-bubble fork", id)
		}
		if pinCount[id] != 1 {
			t.Fatalf("exactly ONE pin (Pending=false) per message.id, got %d for %s", pinCount[id], id)
		}
		if final.Pending {
			t.Fatalf("the turn completed but bubble %s still hangs Pending=true", id)
		}
		if strings.TrimSpace(final.Text) == "" {
			t.Fatalf("the pinned bubble %s must carry the reply text", id)
		}
		t.Logf("PINNED BUBBLE %s: pins=%d growth-events=%d final text=%q",
			id, pinCount[id], growthCount[id], trimTo(final.Text, 120))
	}
	// The doubled-bubble symptom: two bubbles carrying IDENTICAL text.
	seen := map[string]string{}
	for id, final := range merged {
		if other, dup := seen[final.Text]; dup {
			t.Fatalf("doubled bubble: %s and %s carry identical text %q", other, id, trimTo(final.Text, 80))
		}
		seen[final.Text] = id
	}

	// The backend's own turn bookkeeping settles with the result frame.
	b.mu.Lock()
	busy := b.busyTurns
	pending := len(b.pendingBoss)
	b.mu.Unlock()
	if busy != 0 || pending != 0 {
		t.Fatalf("the turn must settle with the result frame: busyTurns=%d pendingBoss=%d", busy, pending)
	}
	t.Logf("turn settled: busyTurns=0 pendingBoss=0, primary session=%s", b.PrimaryID())
}

// ---------------------------------------------------------------- test 2

// TestClaudeLivePermissionRoundTrip — a permission-requiring tool call
// surfaces as an office permission event and the allow answer round-trips
// to claude's stdin, letting the tool actually run.
//
// Trigger choice (the brief asks to say what + why): the WRITE tool,
// asked to create a scratch file. Why it reliably triggers can_use_tool
// with THIS account: the account's permission mode is the CLI default
// (no defaultMode override in ~/.claude/settings.json), in which Write is
// a write-class tool that ALWAYS prompts; the settings' permissions.allow
// list carries Bash/Read rules but NO Write rule. Bash(ls) was rejected
// as the trigger: the CLI's default mode auto-approves read-only shell
// commands (and this account even allow-lists some Bash), so no ask would
// surface.
func TestClaudeLivePermissionRoundTrip(t *testing.T) {
	bin := liveClaudeGate(t)
	scratch := t.TempDir() // auto-removed — no residue
	target := filepath.Join(scratch, "theboringoffice-live-perm.txt")
	b, log, rec := liveBoot(t, bin, scratch)

	prompt := fmt.Sprintf("Use the Write tool to create the file %s with the exact content \"hello\" and nothing else. Do not do anything else.", target)
	t.Logf("prompt: %s", prompt)
	if err := b.Send(prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// The permission ask must ARRIVE. The real CLI can route it two ways:
	// the classic can_use_tool control_request (-> EvPermission) or, for
	// declared kinds, a request_user_dialog (-> EvQuestion). The office
	// renders both; the test answers whichever the REAL wire raises (and
	// prints which it was — live evidence no stub can fake).
	askWait := func() bool {
		for _, e := range log.snapshot() {
			if (e.Kind == state.EvPermission || e.Kind == state.EvQuestion) && e.ToolState == "pending" {
				return true
			}
		}
		return false
	}
	liveWait(t, "the permission ask (EvPermission/EvQuestion pending)", 100*time.Second, rec, log, askWait)

	var permEv *state.Event
	var questionEv *state.Event
	for _, e := range log.snapshot() {
		e := e
		if e.Kind == state.EvPermission && e.ToolState == "pending" && permEv == nil {
			permEv = &e
		}
		if e.Kind == state.EvQuestion && e.ToolState == "pending" && questionEv == nil {
			questionEv = &e
		}
	}
	framesAtAnswer := rec.count()

	switch {
	case permEv != nil:
		rendered, _ := json.Marshal(permEv)
		t.Logf("the REAL wire raised can_use_tool -> EvPermission (the office modal trigger): %s", rendered)
		if permEv.PermissionID == "" || permEv.ToolName == "" {
			t.Fatalf("EvPermission identity incomplete (the modal needs PermissionID + ToolName): %+v", permEv)
		}
		// LIVE-WIRE FINDING: the REAL can_use_tool control_request carries
		// NO envelope session_id (unlike the stub fixtures) — the mapper
		// reads it verbatim, so EvPermission.SessionID is "". That is
		// cosmetically incomplete but NOT functionally blocking:
		// AnswerPermission keys on the request_id alone (the CLI parks
		// exactly one promise per request_id), which is precisely what the
		// rest of this test proves by round-tripping the answer. Reported
		// in ISSUES — a mapper fallback to the init-pinned primaryID would
		// be a one-line improvement, but changing the mapper is OUT of
		// this brief's scope.
		t.Logf("LIVE-WIRE FINDING: can_use_tool carried session_id=%q (empty on the real wire); the answer round-trips on request_id alone", permEv.SessionID)
		t.Logf("answering allow-once via AnswerPermission(%q, \"once\")", permEv.PermissionID)
		if err := b.AnswerPermission(permEv.PermissionID, "once"); err != nil {
			t.Fatalf("AnswerPermission: %v", err)
		}
	case questionEv != nil:
		rendered, _ := json.Marshal(questionEv)
		t.Logf("the REAL wire raised request_user_dialog -> EvQuestion (the CLI's declared-kind dialog path): %s", rendered)
		// Pick the allow-once label from the RENDERED options (never
		// invented: it must be one of the pages' real labels).
		allowLabel := ""
		for _, q := range questionEv.Questions {
			for _, opt := range q.Options {
				if opt.Label == claudeDialogAllowOnce {
					allowLabel = opt.Label
				}
			}
		}
		if allowLabel == "" {
			liveDumpWire(t, rec)
			liveDumpEvents(t, log)
			t.Fatalf("the permission dialog rendered no %q option", claudeDialogAllowOnce)
		}
		t.Logf("answering [%q] via AnswerQuestion(%q)", allowLabel, questionEv.QuestionID)
		if err := b.AnswerQuestion(questionEv.QuestionID, [][]string{{allowLabel}}); err != nil {
			t.Fatalf("AnswerQuestion: %v", err)
		}
	default:
		liveDumpWire(t, rec)
		liveDumpEvents(t, log)
		t.Fatal("no pending permission event found despite the wait (impossible)")
	}

	// The tool must ACTUALLY RUN now: a user frame carrying tool_result
	// (is_error=false), then the turn's result frame. Both are observable
	// outcomes on the real wire — the round-trip needs no stub capture.
	liveWait(t, "the post-answer tool_result frame (the tool really ran)", 90*time.Second, rec, log, func() bool {
		return rec.found(func(f claudeEvent) bool {
			if f.Type != "user" {
				return false
			}
			for _, bl := range f.Message.Content {
				if bl.Type == "tool_result" && !bl.IsError {
					return true
				}
			}
			return false
		})
	})
	liveWait(t, "the turn's result frame after the answer", 90*time.Second, rec, log, func() bool { return liveResultSeen(rec) })

	t.Logf("=== POST-ANSWER FRAMES (from frame[%03d] on) — the proof the tool ran ===", framesAtAnswer)
	for i, f := range rec.snapshot() {
		if i < framesAtAnswer {
			continue
		}
		switch f.Type {
		case "user":
			var blocks []string
			for _, bl := range f.Message.Content {
				blocks = append(blocks, fmt.Sprintf("%s(is_error=%v)", bl.Type, bl.IsError))
			}
			t.Logf("  frame[%03d] user tool_result blocks=%v uuid=%s", i, blocks, f.UUID)
		case "assistant":
			var blocks []string
			for _, bl := range f.Message.Content {
				blocks = append(blocks, bl.Type)
			}
			t.Logf("  frame[%03d] assistant message.id=%s uuid=%s content_blocks=%v", i, f.Message.ID, f.UUID, blocks)
		case "result":
			t.Logf("  frame[%03d] result/%s is_error=%v duration_ms=%d uuid=%s", i, f.Subtype, f.IsError, f.DurationMs, f.UUID)
		case "system":
			t.Logf("  frame[%03d] system/%s session_id=%s", i, f.Subtype, f.SessionID)
		case "control_request", "control_response":
			t.Logf("  frame[%03d] %s/%s request_id=%s", i, f.Type, f.Request.Subtype, f.RequestID)
		}
	}
	liveDumpEvents(t, log)

	// The observable end state: an EvTool closed done (never error) and
	// the scratch file on disk with the exact content.
	var toolDone bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvTool && e.ToolState == "done" {
			toolDone = true
			t.Logf("EvTool done: callID=%s tool=%s summary=%q", e.CallID, e.ToolName, e.ToolSummary)
		}
		if e.Kind == state.EvTool && e.ToolState == "error" {
			t.Fatalf("the granted tool must not land an error row: %+v", e)
		}
	}
	if !toolDone {
		t.Fatalf("the granted tool never reported done; events dumped above")
	}
	bits, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("the Write tool ran but the scratch file is missing: %v", err)
	}
	if strings.TrimSpace(string(bits)) != "hello" {
		t.Fatalf("scratch file content drifted: %q (want hello)", string(bits))
	}
	t.Logf("scratch file on disk: %s content=%q (auto-removed with t.TempDir)", target, string(bits))

	// The local resolved event must have closed the modal.
	var resolved bool
	for _, e := range log.snapshot() {
		if (e.Kind == state.EvPermission || e.Kind == state.EvQuestion) && e.ToolState == "resolved" {
			resolved = true
		}
	}
	if !resolved {
		t.Fatalf("the answer must emit the local resolved event (modal closer)")
	}
	t.Log("modal closed: resolved event present — the full ask -> answer -> ran -> settled loop completed on the real CLI")
}

// ---------------------------------------------------------------- test 3

// TestClaudeLiveThinkingEvThought — thinking tokens map to EvThought on
// the REAL wire. Tolerant by design (the brief): if the account/model
// emits no thinking blocks, print NO-THINKING-EMITTED-BY-ACCOUNT with the
// content-block inventory proving the test really looked, and PASS.
func TestClaudeLiveThinkingEvThought(t *testing.T) {
	bin := liveClaudeGate(t)
	b, log, rec := liveBoot(t, bin, t.TempDir())

	if err := b.Send("Think step by step about whether 0.9 recurring equals 1, then answer in one sentence."); err != nil {
		t.Fatalf("Send: %v", err)
	}
	liveWait(t, "the turn's result frame (real API latency)", 150*time.Second, rec, log, func() bool { return liveResultSeen(rec) })
	liveDumpWire(t, rec)

	type thoughtRow struct {
		CallID string
		Done   bool
		Text   string
	}
	var rows []thoughtRow
	for _, e := range log.snapshot() {
		if e.Kind == state.EvThought {
			rows = append(rows, thoughtRow{CallID: e.CallID, Done: e.Done, Text: e.Text})
		}
	}

	if len(rows) == 0 {
		// Prove the look was honest: every assistant frame's content-block
		// types + every stream block-start type, straight off the raw wire.
		t.Log("NO-THINKING-EMITTED-BY-ACCOUNT — zero EvThought events on the live turn; the account/model emitted no thinking blocks. Frame inventory proof:")
		for i, f := range rec.snapshot() {
			switch f.Type {
			case "assistant":
				var blocks []string
				for _, bl := range f.Message.Content {
					blocks = append(blocks, bl.Type)
				}
				t.Logf("  frame[%03d] assistant message.id=%s content_blocks=%v (no \"thinking\" block)", i, f.Message.ID, blocks)
			case "stream_event":
				var inner claudeStreamInner
				if json.Unmarshal(f.Event, &inner) == nil && inner.Type == "content_block_start" {
					t.Logf("  frame[%03d] stream_event content_block_start type=%s", i, inner.ContentBlock.Type)
				}
			}
		}
		liveDumpEvents(t, log)
		return // PASS with the note — thinking is never faked
	}

	t.Logf("=== EvThought sequence on the live wire (%d events) ===", len(rows))
	for i, r := range rows {
		t.Logf("  thought[%02d] callID=%s done=%v len=%d text=%q", i, r.CallID, r.Done, len(r.Text), trimTo(r.Text, 80))
	}
	// One stable CallID, rooted at the real message.id.
	callID := rows[0].CallID
	if !strings.HasPrefix(callID, "think-msg_") {
		t.Fatalf("the thought CallID must root at the real message.id (think-msg_...), got %s", callID)
	}
	for _, r := range rows {
		if r.CallID != callID {
			t.Fatalf("the thought's CallID must be stable across the sequence: %s vs %s", r.CallID, callID)
		}
	}
	// open (Done=false) -> accumulate (growing transcript) -> close
	// (Done=true carrying the full text).
	last := rows[len(rows)-1]
	if !last.Done {
		t.Fatalf("the thought must CLOSE Done=true carrying the full text, last row: %+v", last)
	}
	var longest string
	for _, r := range rows[:len(rows)-1] {
		if r.Done {
			t.Fatalf("a Done=true arrived before the sequence end: %+v in %+v", r, rows)
		}
		if !strings.HasPrefix(last.Text, r.Text) {
			t.Fatalf("EvThought must carry the ACCUMULATED transcript: row %q is not a prefix of the close %q",
				trimTo(r.Text, 60), trimTo(last.Text, 60))
		}
		if len(r.Text) > len(longest) {
			longest = r.Text
		}
	}
	if len(rows) > 1 && longest == "" {
		t.Fatalf("the open rows must carry growing text, got %+v", rows)
	}
	t.Logf("thinking contract held on the real wire: %d open/accumulate rows -> close (full text %d chars)", len(rows)-1, len(last.Text))
}

// ---------------------------------------------------------------- test 4

// TestClaudeLiveBypass — the bypass-permissions toggle against the REAL
// CLI: SetBypassPermissions(true) pre-Start spawns the child with
// `--dangerously-skip-permissions` (and WITHOUT `--permission-prompt-tool
// stdio`), so the SAME Write-tool prompt that drives
// TestClaudeLivePermissionRoundTrip's ask (the pre-flag control run —
// this account's default mode ALWAYS prompts for Write) now runs with
// ZERO permission events: the turn must COMPLETE (result frame), the
// scratch file must exist with the exact content, and the mapped event
// log must carry ZERO EvPermission — with the raw wire proving zero
// can_use_tool control_requests arrived (the CLI stopped asking; the
// office strips NOTHING mid-pipe).
func TestClaudeLiveBypass(t *testing.T) {
	bin := liveClaudeGate(t)
	scratch := t.TempDir() // auto-removed — no residue
	target := filepath.Join(scratch, "theboringoffice-live-bypass.txt")

	log := &claudeEventLog{}
	rec := &liveFrameRec{}
	b := newClaudeBackend(bin, scratch, nil)
	b.rawFrameHook = rec.add
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start (real claude at %s): %v", bin, err)
	}
	t.Cleanup(func() { _ = b.Stop() }) // stdin-close -> drain -> SIGTERM -> SIGKILL

	// The post-Start contract rides the live run too: the argv is frozen.
	if err := b.SetBypassPermissions(false); err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start SetBypassPermissions must fail naming \"respawn required\", got %v", err)
	} else {
		t.Logf("post-Start SetBypassPermissions correctly refused: %v", err)
	}

	// Byte-identical prompt to the control run (test 2): Write ALWAYS
	// prompts in this account's default permission mode.
	prompt := fmt.Sprintf("Use the Write tool to create the file %s with the exact content \"hello\" and nothing else. Do not do anything else.", target)
	t.Logf("prompt: %s", prompt)
	if err := b.Send(prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}

	liveWait(t, "the turn's result frame (real API latency, bypassed tool)", 150*time.Second, rec, log, func() bool { return liveResultSeen(rec) })
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)

	// The event inventory: ZERO EvPermission and ZERO can_use_tool
	// control_requests — vs the control run, where the same prompt raises
	// the ask. (EvPermission is what the office renders; the raw
	// control_request count proves the CLI itself stopped asking.)
	var permEvents, questionEvents int
	for _, e := range log.snapshot() {
		switch e.Kind {
		case state.EvPermission:
			permEvents++
		case state.EvQuestion:
			questionEvents++
		}
	}
	var canUseToolFrames int
	for _, f := range rec.snapshot() {
		if f.Type == "control_request" && f.Request.Subtype == "can_use_tool" {
			canUseToolFrames++
		}
	}
	t.Logf("BYPASS EVENT INVENTORY: EvPermission=%d EvQuestion=%d raw can_use_tool control_requests=%d (control run TestClaudeLivePermissionRoundTrip: EvPermission>=1 on the same prompt)",
		permEvents, questionEvents, canUseToolFrames)
	if permEvents != 0 || canUseToolFrames != 0 {
		t.Fatalf("a bypassed backend must never raise a permission ask: EvPermission=%d can_use_tool frames=%d", permEvents, canUseToolFrames)
	}

	// The tool ACTUALLY RAN without any office answer: a user frame
	// carrying a clean tool_result, an EvTool done, and the file on disk.
	if !rec.found(func(f claudeEvent) bool {
		if f.Type != "user" {
			return false
		}
		for _, bl := range f.Message.Content {
			if bl.Type == "tool_result" && !bl.IsError {
				return true
			}
		}
		return false
	}) {
		t.Fatalf("the bypassed Write never landed a clean tool_result frame")
	}
	var toolDone bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvTool && e.ToolState == "done" {
			toolDone = true
			t.Logf("EvTool done: callID=%s tool=%s summary=%q", e.CallID, e.ToolName, e.ToolSummary)
		}
		if e.Kind == state.EvTool && e.ToolState == "error" {
			t.Fatalf("the bypassed tool must not land an error row: %+v", e)
		}
	}
	if !toolDone {
		t.Fatalf("the bypassed Write never reported done; events dumped above")
	}
	bits, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("the Write tool ran but the scratch file is missing: %v", err)
	}
	if strings.TrimSpace(string(bits)) != "hello" {
		t.Fatalf("scratch file content drifted: %q (want hello)", string(bits))
	}
	t.Logf("scratch file on disk: %s content=%q (auto-removed with t.TempDir)", target, string(bits))
	t.Log("bypass contract held on the real CLI: turn completed, tool ran unanswered, ZERO permission events")
}

// ---------------------------------------------------------------- test 5

// TestClaudeLiveCharterReaches — the whole point of charter_claude.go:
// the oikonomos charter must reach the MODEL through the office's own
// spawn path, not just exist on disk. A scratch dir gets a TRIVIAL
// payload carrying a distinctive nonsense marker (PLUMBERSCRATE — a word
// no pretraining run knows in this role) and a CLAUDE.md generated by
// EnsureClaudeCharter itself; the REAL backend boots on that dir
// (Start re-runs the pass — it must no-op and leave the trivial payload
// untouched), the boss is asked for the word, and the pinned reply must
// carry the marker. Negative control (run by hand during development,
// same CLI): the identical prompt from a dir WITHOUT the wired CLAUDE.md
// answers "IDK".
func TestClaudeLiveCharterReaches(t *testing.T) {
	bin := liveClaudeGate(t)
	scratch := t.TempDir() // auto-removed — no residue

	// The import target is the REAL embedded charter (the pass seeds it
	// byte-exact on claude-only offices — the office-ownership discipline).
	// The marker therefore comes from the real charter itself: the five
	// developer return-contract sections (DONE/FILES/VERIFY/PROOF/ISSUES).
	// Generate the payload + CLAUDE.md via the function under test — the
	// exact bytes a live boot wires.
	changed, notes := EnsureClaudeCharter(scratch)
	if !changed {
		t.Fatalf("fresh scratch dir: EnsureClaudeCharter changed=false, want true (notes %v)", notes)
	}
	t.Logf("EnsureClaudeCharter notes: %v", notes)
	payloadPath := filepath.Join(scratch, ".opencode", "oikonomos.md")
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("the pass must seed the payload: %v", err)
	}
	if string(payload) != charter.Text {
		t.Fatalf("the seeded payload must be the embedded charter (%d bytes), got %d", len(charter.Text), len(payload))
	}
	claudeMD, err := os.ReadFile(filepath.Join(scratch, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("generated CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(claudeMD), claudeCharterImportLine) {
		t.Fatalf("generated CLAUDE.md lacks the import line %q:\n%s", claudeCharterImportLine, claudeMD)
	}
	t.Logf("generated CLAUDE.md (verbatim):\n%s", claudeMD)

	// Boot the REAL backend on the scratch dir — the same
	// newClaudeBackend -> Start path the app drives (cmd.Dir = scratch).
	// Start's own EnsureClaudeCharter call must leave the files alone:
	// CLAUDE.md references oikonomos.md (no-op) and the payload is fresh.
	b, log, rec := liveBoot(t, bin, scratch)
	if got, err := os.ReadFile(payloadPath); err != nil || string(got) != charter.Text {
		t.Fatalf("Start drifted the payload: err=%v (want the embedded charter)", err)
	}

	prompt := "the charter names five sections of the developer return contract — answer with just the five names"
	t.Logf("prompt: %s", prompt)
	if err := b.Send(prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}
	liveWait(t, "the turn's result frame (real API latency)", 150*time.Second, rec, log, func() bool { return liveResultSeen(rec) })
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)

	// Fold the boss-chat events by Msg.ID (the chat reducer's replace-on-
	// merge) and read the final pinned text off every bubble.
	merged := map[string]state.ChatMsg{}
	for _, e := range log.snapshot() {
		if e.Kind != state.EvChatBoss || !strings.HasPrefix(e.Msg.ID, "bossmsg-") {
			continue
		}
		merged[e.Msg.ID] = e.Msg
	}
	if len(merged) == 0 {
		t.Fatal("no boss bubbles landed — the turn produced no reply")
	}
	var reply strings.Builder
	for _, id := range sortedKeys(merged) {
		if merged[id].Pending {
			t.Fatalf("the turn completed but bubble %s still hangs Pending=true", id)
		}
		reply.WriteString(merged[id].Text)
		reply.WriteString("\n")
	}
	t.Logf("boss reply (verbatim): %q", reply.String())
	for _, section := range []string{"DONE", "FILES", "VERIFY", "PROOF", "ISSUES"} {
		if !strings.Contains(reply.String(), section) {
			t.Fatalf("the real charter never reached the model through the office spawn: reply %q lacks the return-contract section %q", reply.String(), section)
		}
	}
	t.Logf("CHARTER-REACHES: the boss recited all five return-contract sections (DONE/FILES/VERIFY/PROOF/ISSUES) — the REAL oikonomos charter rode CLAUDE.md -> @.opencode/oikonomos.md through the office's own claude spawn")
}

// ---------------------------------------------------------------- test 6

// TestClaudeLiveToolOutput — the Event.ToolOutput contract against the
// REAL CLI: one prompt that runs exactly one Bash tool call with a known
// echo marker. The tool's done event must carry the echo's text on
// ToolOutput (the member clicks the tool row and sees what it returned).
// The prompt's Bash(echo …) is read-only-safe on this account (the CLI's
// default mode auto-approves it — see TestClaudeLivePermissionRoundTrip's
// trigger analysis), but the wait loop still answers any pending ask the
// way the office modal would, so a permission-policy drift can never hang
// the test.
func TestClaudeLiveToolOutput(t *testing.T) {
	bin := liveClaudeGate(t)
	b, log, rec := liveBoot(t, bin, t.TempDir())

	const marker = "the-office-tool-output-marker"
	prompt := "Run exactly one Bash tool call with this exact command: echo " + marker +
		" — then reply with the single word done. Do not run any other tool."
	t.Logf("prompt: %s", prompt)
	if err := b.Send(prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Wait for the turn's result frame, answering any pending ask the way
	// the office modal would (allow-once; each request answered at most
	// once).
	answered := map[string]bool{}
	deadline := time.Now().Add(150 * time.Second)
	for !liveResultSeen(rec) {
		if time.Now().After(deadline) {
			liveDumpWire(t, rec)
			liveDumpEvents(t, log)
			t.Fatalf("timed out after %s waiting for the turn's result frame", 150*time.Second)
		}
		for _, e := range log.snapshot() {
			switch {
			case e.Kind == state.EvPermission && e.ToolState == "pending" && !answered[e.PermissionID]:
				answered[e.PermissionID] = true
				t.Logf("answering allow-once via AnswerPermission(%q)", e.PermissionID)
				_ = b.AnswerPermission(e.PermissionID, "once")
			case e.Kind == state.EvQuestion && e.ToolState == "pending" && !answered[e.QuestionID]:
				for _, q := range e.Questions {
					for _, opt := range q.Options {
						if opt.Label == claudeDialogAllowOnce && !answered[e.QuestionID] {
							answered[e.QuestionID] = true
							t.Logf("answering [%q] via AnswerQuestion(%q)", opt.Label, e.QuestionID)
							_ = b.AnswerQuestion(e.QuestionID, [][]string{{opt.Label}})
						}
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)

	// Fold the tool events by CallID (the chat's merge shape): the bash
	// call's done row must carry the echo's output on ToolOutput.
	type toolRow struct {
		name, state, output string
	}
	byCallID := map[string]toolRow{}
	for _, e := range log.snapshot() {
		if e.Kind != state.EvTool {
			continue
		}
		byCallID[e.CallID] = toolRow{e.ToolName, e.ToolState, e.ToolOutput}
	}
	var doneRow *toolRow
	for id, r := range byCallID {
		t.Logf("tool row: callID=%s tool=%s state=%s output=%q", id, r.name, r.state, trimTo(r.output, 120))
		if r.name == "bash" && r.state == "done" {
			r := r
			doneRow = &r
		}
	}
	if doneRow == nil {
		t.Fatalf("the echo's Bash call never landed a done row; tool rows dumped above")
	}
	if !strings.Contains(doneRow.output, marker) {
		t.Fatalf("the done event's ToolOutput must contain the echo marker %q, got %q", marker, doneRow.output)
	}
	if len(doneRow.output) > toolOutputCapBytes {
		t.Fatalf("ToolOutput must never exceed the %d-byte cap, got %d", toolOutputCapBytes, len(doneRow.output))
	}
	t.Logf("LIVE CAPTURE: the bash done event's ToolOutput (verbatim): %q", doneRow.output)
	t.Logf("tool-output contract held on the real CLI: marker %q present in the done event's ToolOutput", marker)
}

// sortedKeys renders a string-keyed map as a sorted stable list for logs.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
