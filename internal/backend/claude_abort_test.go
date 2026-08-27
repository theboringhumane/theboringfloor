// claude_abort_test.go — the /stop kill ladder: AbortSessions writes ONE
// interrupt control_request, escalates SIGINT after
// claudeAbortSigIntAfter when the turn stubbornly keeps running, then
// SIGTERM after claudeAbortSigTermAfter — and the signal-killed child
// (exit 143) is a CLEAN kill, not a crash line.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

func TestClaudeAbortInterruptThenSignalLadder(t *testing.T) {
	// shrink the ladder
	oldInt, oldTerm := claudeAbortSigIntAfter, claudeAbortSigTermAfter
	claudeAbortSigIntAfter = 60 * time.Millisecond
	claudeAbortSigTermAfter = 140 * time.Millisecond
	defer func() { claudeAbortSigIntAfter, claudeAbortSigTermAfter = oldInt, oldTerm }()

	capture := filepath.Join(t.TempDir(), "capture.log")
	siglog := filepath.Join(t.TempDir(), "sig.log")
	// the stub holds its turn forever, logs the interrupt request off its
	// stdin, traps SIGINT to the signal log, and dies on the SIGTERM tail.
	stubBody := claudeStubPreambleSh() + `trap 'echo INT >> "` + siglog + `"' INT
trap 'echo TERM >> "` + siglog + `"; exit 143' TERM
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if err := b.Send("turn that never ends"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "the turn staged (user line seen)", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})

	if err := b.AbortSessions(); err != nil {
		t.Fatalf("AbortSessions: %v", err)
	}
	claudeWait(t, "the interrupt control_request on stdin", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	if want := `{"type":"control_request","request_id":"office-interrupt-1","request":{"subtype":"interrupt"}}`; lines[1] != want {
		t.Fatalf("interrupt wire shape drifted:\n got %q\nwant %q", lines[1], want)
	}

	// the FIFO head placeholder closed with the stopped marker
	claudeWait(t, "the stopped placeholder", 2*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvChatBoss && strings.Contains(e.Msg.Text, "stopped (turn aborted)") {
				return true
			}
		}
		return false
	})

	// SIGINT fires after the interrupt won't settle the turn; SIGTERM's
	// exit-143 is then a CLEAN kill: the watch latches died without the
	// scary crash line.
	claudeWait(t, "SIGINT + SIGTERM delivered, exit 143", 4*time.Second, func() bool {
		bits, _ := os.ReadFile(siglog)
		return strings.Contains(string(bits), "INT") && strings.Contains(string(bits), "TERM")
	})
	claudeWait(t, "the clean-kill watch line", 3*time.Second, func() bool {
		b.mu.Lock()
		died := b.died
		b.mu.Unlock()
		return died && log.hasStatusContaining("clean kill")
	})
	if log.hasStatusContaining("claude process died (exited") {
		t.Fatalf("exit 143 must read as a clean kill, never a crash line")
	}

	// the next Send respawns (the died latch consumed): an interrupt is
	// not faked against a dead process
	if err := b.AbortSessions(); err != nil {
		t.Fatalf("AbortSessions on a dead turn must be a silent no-op: %v", err)
	}
	if got := len(claudeCapture(t, capture)); got != 2 {
		t.Fatalf("no second interrupt may be written (capture grew to %d)", got)
	}
}

func TestClaudeAbortNothingRunning(t *testing.T) {
	stub := claudeStubScript(t, claudeStubPreambleSh()+`while IFS= read -r line; do :; done
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if err := b.AbortSessions(); err != nil {
		t.Fatalf("AbortSessions with nothing running must be nil, got %v", err)
	}
	if log.hasStatusContaining("interrupt sent") {
		t.Fatalf("an idle AbortSessions must never write an interrupt")
	}
}
