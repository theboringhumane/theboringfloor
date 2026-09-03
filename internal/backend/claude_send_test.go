// claude_send_test.go — the backend chat path against a shell stub: Send
// writes EXACTLY ONE stdin {"type":"user",…} line per prompt (byte-pinned
// shape), mid-turn sends queue immediately with no blocking, and the
// local echo + placeholder obligations mirror the opencode backend.
package backend

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

type claudeFailWriteCloser struct {
	err    error
	writes int
	last   []byte
}

func (w *claudeFailWriteCloser) Write(p []byte) (int, error) {
	w.writes++
	w.last = append(w.last[:0], p...)
	return 0, w.err
}

func (*claudeFailWriteCloser) Close() error { return nil }

type claudeCaptureWriteCloser struct{ writes [][]byte }

func (w *claudeCaptureWriteCloser) Write(p []byte) (int, error) {
	w.writes = append(w.writes, append([]byte(nil), p...))
	return len(p), nil
}

func (*claudeCaptureWriteCloser) Close() error { return nil }

// claudeWriteReadyBackend supplies a deterministic stdin seam without
// spawning a child. send's ready gate requires a non-nil command and writer;
// the fake command is sufficient because no process lifecycle is exercised.
func claudeWriteReadyBackend(writer *claudeFailWriteCloser) (*liveClaudeBackend, *claudeEventLog) {
	b := newClaudeBackend("true", ".", nil)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)
	b.proc = &exec.Cmd{}
	b.procStdin = writer
	return b, log
}

// claudeCapture reads the stub's capture file (one stdin line per row,
// the stub mirrors with >> appends). The office's initialize
// control_request (the supportedDialogKinds declaration, the FIRST stdin
// line of every process since the dialog-kind wave) is FILTERED OUT: it
// rides every capture and no pre-existing assertion counts it — tests
// that pin the declaration bytes read the raw file via claudeCaptureRaw.
func claudeCapture(t *testing.T, path string) []string {
	t.Helper()
	var out []string
	for _, ln := range claudeCaptureRaw(t, path) {
		if strings.Contains(ln, `"subtype":"initialize"`) {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// claudeCaptureRaw reads the stub's capture file UNFILTERED (initialize
// declaration included).
func claudeCaptureRaw(t *testing.T, path string) []string {
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
// The browser-tool preamble (browsertools.PromptPreamble) rides the
// FIRST user line of a session — the byte-pin below is the SECOND
// send's line (the plain-prompt shape the preamble never touches); the
// first line's contract lives in browser_open_test.go.
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

	if err := b.Send("brief me"); err != nil {
		t.Fatalf("Send 1 (the briefed one): %v", err)
	}
	claudeWait(t, "the first (preamble-carrying) user line in the stub capture", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	if err := b.Send("hello there"); err != nil {
		t.Fatalf("Send 2: %v", err)
	}
	const want = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello there"}]},"parent_tool_use_id":null}`
	claudeWait(t, "the second user line to land in the stub capture", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	if len(lines) != 2 {
		t.Fatalf("two Sends wrote %d stdin lines (want exactly 2)", len(lines))
	}
	if lines[1] != want {
		t.Fatalf("stdin wire shape drifted:\n got %q\nwant %q", lines[1], want)
	}
	// the local echo + ONE pending placeholder fired per Send (Send owns them)
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

func TestClaudeSendWriteFailureReturnsActionableError(t *testing.T) {
	writeErr := errors.New("forced stdin write failure")
	writer := &claudeFailWriteCloser{err: writeErr}
	b, log := claudeWriteReadyBackend(writer)

	err := b.Send("retain this prompt")
	if !errors.Is(err, writeErr) {
		t.Fatalf("Send error = %v, want wrapped %v", err, writeErr)
	}
	if !strings.Contains(err.Error(), "write claude prompt") {
		t.Fatalf("Send error must name the failed operation, got %q", err)
	}
	if writer.writes != 1 {
		t.Fatalf("failed Send wrote %d times, want exactly once", writer.writes)
	}
	if b.briefed {
		t.Fatal("a failed first write must not spend the browser preamble")
	}
	b.mu.Lock()
	pending := append([]string(nil), b.pendingBoss...)
	b.mu.Unlock()
	if len(pending) != 0 {
		t.Fatalf("failed Send left pending placeholders behind: %v", pending)
	}

	var sawFailure bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatBoss && strings.Contains(e.Msg.Text, "prompt failed: forced stdin write failure") && !e.Msg.Pending {
			sawFailure = true
		}
	}
	if !sawFailure {
		t.Fatalf("Send must emit the existing failed-prompt bubble before returning; events: %+v", log.snapshot())
	}
}
