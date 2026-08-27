// claude_send_test.go — the backend chat path against a shell stub: Send
// writes EXACTLY ONE stdin {"type":"user",…} line per prompt (byte-pinned
// shape), mid-turn sends queue immediately with no blocking, and the
// local echo + placeholder obligations mirror the opencode backend.
package backend

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// claudeCapture reads the stub's capture file (one stdin line per row,
// the stub mirrors with >> appends).
func claudeCapture(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// TestClaudeSendWritesExactlyOnce pins the EXACT stdin wire shape:
// {"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello there"}]},"parent_tool_use_id":null}
func TestClaudeSendWritesExactlyOnce(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
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

	if err := b.Send("hello there"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	const want = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello there"}]},"parent_tool_use_id":null}`
	claudeWait(t, "the user line to land in the stub capture", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	lines := claudeCapture(t, capture)
	if len(lines) != 1 {
		t.Fatalf("Send wrote %d stdin lines (want exactly 1)", len(lines))
	}
	if lines[0] != want {
		t.Fatalf("stdin wire shape drifted:\n got %q\nwant %q", lines[0], want)
	}
	// the local echo + ONE pending placeholder fired (Send owns them)
	var echo, placeholder bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatUser && e.Msg.Text == "hello there" {
			echo = true
		}
		if e.Kind == state.EvChatBoss && e.Msg.Pending && strings.HasPrefix(e.Msg.ID, "boss-") {
			placeholder = true
		}
	}
	if !echo || !placeholder {
		t.Fatalf("Send must echo chat-user and stage one placeholder (echo=%v placeholder=%v)", echo, placeholder)
	}
}

// TestClaudeSendMidTurnQueuedNoBlocking — a Send landing while the turn
// is still running (the stub holds its reply) must go straight to stdin:
// no idle gate, no blocking on a busy turn (opencode prompt_async parity).
func TestClaudeSendMidTurnQueuedNoBlocking(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	// the stub replies to the FIRST input only after a hold, so turn 2's
	// write has to land BEFORE any frame proves the turn ended.
	stubBody := claudeStubPreambleSh() + `first=1
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$first" in
    1) first=0; sleep 1 ;;
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

	if err := b.Send("first prompt"); err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	claudeWait(t, "turn 1's line in the capture", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	// mid-turn (the stub is asleep holding the reply): the second Send
	// must complete its stdin write without waiting for the turn.
	done := make(chan error, 1)
	go func() { done <- b.Send("second prompt") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Send 2: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Send 2 blocked mid-turn — there is no idle gate in this backend")
	}
	claudeWait(t, "both queued lines in the capture", 4*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	if !strings.Contains(lines[1], "second prompt") {
		t.Fatalf("queued stdin line drifted: %q", lines[1])
	}
}
