// claude_bypass_candidate_integration_test.go proves the exact handoff shape
// used by an office /bypass transition: a fresh, bypassed Claude candidate is
// started with the prior primary pin, the old generation is stopped only after
// the candidate starts, and subsequent ordinary and attachment sends remain on
// that candidate's live JSONL pipe.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

func TestClaudeBypassCandidateSurvivesOldGenerationCleanupAndReplies(t *testing.T) {
	dir := t.TempDir()
	argvLog := filepath.Join(dir, "candidate-argv.log")
	capture := filepath.Join(dir, "candidate-stdin.log")
	candidateStub := claudeStubScript(t, `printf '%s\n' "$*" >> "`+argvLog+`"
n=0
while IFS= read -r line; do
  printf '%s\n' "$line" >> "`+capture+`"
  case "$line" in
    *'"type":"user"'*)
      n=$((n+1))
      if [ "$n" -eq 1 ]; then
`+claudeStubPreambleSh()+`      fi
      printf '%s\n' '{"type":"assistant","message":{"id":"msg-candidate-'$n'","role":"assistant","content":[{"type":"text","text":"candidate boss reply '$n'"}]},"session_id":"fresh-candidate-session","uuid":"frame-candidate-'$n'","parent_tool_use_id":null}'
      printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"fresh-candidate-session","uuid":"result-candidate-'$n'","total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":1}}'
      ;;
  esac
done
`)
	oldStub := claudeStubScript(t, `while IFS= read -r line; do :; done
`)

	old := newClaudeBackend(oldStub, dir, nil)
	if err := old.Start((&claudeEventLog{}).emit); err != nil {
		t.Fatalf("old Start: %v", err)
	}
	defer func() { _ = old.Stop() }()

	log := &claudeEventLog{}
	candidate := newClaudeBackend(candidateStub, dir, nil)
	// This is the app handoff's restored primary seam. It deliberately does
	// not become --resume on Start: a bypass replacement is a fresh child.
	candidate.PrimaryOverride("stale-saved-session")
	if err := candidate.SetBypassPermissions(true); err != nil {
		t.Fatalf("candidate SetBypassPermissions: %v", err)
	}
	if err := candidate.Start(log.emit); err != nil {
		t.Fatalf("candidate Start: %v", err)
	}
	defer func() { _ = candidate.Stop() }()

	// Mirror the app's old-generation cleanup only after the candidate Start
	// completed. These backends must have independent stop/flow/stdin state.
	if err := old.Stop(); err != nil {
		t.Fatalf("old cleanup Stop: %v", err)
	}
	candidate.mu.Lock()
	stopped := candidate.fl.isStopped()
	proc := candidate.proc
	stdin := candidate.procStdin
	stopping := candidate.stopping
	candidate.mu.Unlock()
	if stopped || stopping || proc == nil || stdin == nil {
		t.Fatalf("old cleanup touched the live candidate: stopped=%v stopping=%v proc=%v stdin=%v", stopped, stopping, proc != nil, stdin != nil)
	}

	if err := candidate.Send("ordinary handoff message"); err != nil {
		t.Fatalf("candidate Send: %v", err)
	}
	attachmentPath := filepath.Join(dir, "handoff note.txt")
	if err := os.WriteFile(attachmentPath, []byte("attached through the candidate"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	claudeWait(t, "the ordinary candidate reply", 3*time.Second, func() bool {
		return hasClaudeBossReply(log.snapshot(), "candidate boss reply 1")
	})
	if err := candidate.SendWith("attachment handoff message", []state.Attachment{{
		Name: "handoff note.txt", Mime: "text/plain", Path: attachmentPath,
	}}); err != nil {
		t.Fatalf("candidate SendWith: %v", err)
	}
	claudeWait(t, "both candidate JSONL user writes", 3*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	claudeWait(t, "the attachment candidate reply", 3*time.Second, func() bool {
		return hasClaudeBossReply(log.snapshot(), "candidate boss reply 2")
	})

	argv, err := os.ReadFile(argvLog)
	if err != nil {
		t.Fatalf("read candidate argv: %v", err)
	}
	if got := strings.TrimSpace(string(argv)); !strings.Contains(got, "--dangerously-skip-permissions") || strings.Contains(got, "--permission-prompt-tool") || strings.Contains(got, "--resume") {
		t.Fatalf("candidate argv = %q, want bypass flag only (no stdio permission tool or stale --resume)", got)
	}
	lines := claudeCapture(t, capture)
	if !strings.Contains(lines[0], "ordinary handoff message") || !strings.Contains(lines[1], attachmentPath) {
		t.Fatalf("candidate stdin did not receive ordinary then path-ref attachment sends: %q", lines)
	}
	if got := candidate.PrimaryID(); got != "stale-saved-session" {
		t.Fatalf("PrimaryOverride must remain the app-restored pin after fresh wire init, got %q", got)
	}
	candidate.mu.Lock()
	busy, pending, died := candidate.busyTurns, len(candidate.pendingBoss), candidate.died
	writerReplaced := candidate.procStdin != stdin
	candidate.mu.Unlock()
	if busy != 0 || pending != 0 || died || writerReplaced || candidate.fl.isStopped() {
		t.Fatalf("candidate did not settle on its original live pipe after both replies: busyTurns=%d pendingBoss=%d died=%v writerReplaced=%v stopped=%v", busy, pending, died, writerReplaced, candidate.fl.isStopped())
	}
	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission || e.Kind == state.EvQuestion {
			t.Fatalf("bypass candidate surfaced a permission ask: %+v", e)
		}
	}
	t.Logf("BYPASS CANDIDATE PROOF: old cleanup left candidate live; argv=%q; sends=%d; replies=%q, %q", strings.TrimSpace(string(argv)), len(lines), "candidate boss reply 1", "candidate boss reply 2")
}

func hasClaudeBossReply(events []state.Event, text string) bool {
	for _, e := range events {
		if e.Kind == state.EvChatBoss && !e.Msg.Pending && e.Msg.Text == text {
			return true
		}
	}
	return false
}

// TestClaudeLiveBypassCandidateRoundTrip is intentionally gated because it
// spends a real Claude API turn. It follows the exact backend Start -> Send
// -> SendWith path (not a direct CLI exec), requires a result after each send,
// and proves bypass emits no office permission/question event.
func TestClaudeLiveBypassCandidateRoundTrip(t *testing.T) {
	bin := liveClaudeGate(t)
	dir := t.TempDir()
	log := &claudeEventLog{}
	rec := &liveFrameRec{}
	b := newClaudeBackend(bin, dir, nil)
	b.rawFrameHook = rec.add
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start bypass candidate: %v", err)
	}
	t.Cleanup(func() { _ = b.Stop() })

	if err := b.Send("Reply with exactly: bypass ordinary reply. Do not use tools."); err != nil {
		t.Fatalf("ordinary Send: %v", err)
	}
	liveWait(t, "the bypass ordinary result", 150*time.Second, rec, log, func() bool { return liveResultCount(rec) >= 1 })
	attachmentPath := filepath.Join(dir, "live handoff note.txt")
	if err := os.WriteFile(attachmentPath, []byte("live path reference"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	if err := b.SendWith("Reply with exactly: bypass attachment reply. Do not use tools.", []state.Attachment{{
		Name: "live handoff note.txt", Mime: "text/plain", Path: attachmentPath,
	}}); err != nil {
		t.Fatalf("attachment SendWith: %v", err)
	}
	liveWait(t, "the bypass attachment result", 150*time.Second, rec, log, func() bool { return liveResultCount(rec) >= 2 })
	if got := livePinnedBossReplyCount(log.snapshot()); got < 2 {
		liveDumpWire(t, rec)
		liveDumpEvents(t, log)
		t.Fatalf("two completed bypass sends must produce at least two pinned boss replies, got %d", got)
	}

	for _, e := range log.snapshot() {
		if e.Kind == state.EvPermission || e.Kind == state.EvQuestion {
			t.Fatalf("live bypass emitted a permission ask: %+v", e)
		}
	}
	liveDumpWire(t, rec)
	liveDumpEvents(t, log)
	t.Logf("LIVE BYPASS CANDIDATE PROOF: results=%d, permission asks=0, attachment path=%q", liveResultCount(rec), attachmentPath)
}

func liveResultCount(rec *liveFrameRec) int {
	count := 0
	for _, f := range rec.snapshot() {
		if f.Type == "result" && f.ParentToolUseID == "" {
			count++
		}
	}
	return count
}

func livePinnedBossReplyCount(events []state.Event) int {
	count := 0
	for _, e := range events {
		if e.Kind == state.EvChatBoss && !e.Msg.Pending && strings.HasPrefix(e.Msg.ID, "bossmsg-") && strings.TrimSpace(e.Msg.Text) != "" {
			count++
		}
	}
	return count
}
