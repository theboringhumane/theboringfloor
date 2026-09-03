// claude_attachment_test.go — attachment SendWith must stay on Claude's
// verified text-only JSONL wire until a media schema is independently proven.
package backend

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/browsertools"
	"github.com/theboringhumane/theboringfloor/internal/chatcontext"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestClaudeSendWithWritesQuotedPathReferences(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "capture.log")
	stub := claudeStubScript(t, claudeStubPreambleSh()+`while IFS= read -r line; do
  printf '%s\n' "$line" >> "`+capture+`"
done
`)
	b := newClaudeBackend(stub, dir, nil)
	log := &claudeEventLog{}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	txt := writeAttachment(t, dir, "notes with spaces.txt", "notes")
	png := writeAttachmentBytes(t, dir, "image.png", []byte("\x89PNG\r\n\x1a\n"))
	pdf := writeAttachment(t, dir, "paper.pdf", "%PDF-1.7\n")
	missing := filepath.Join(dir, "gone.txt")
	atts := []state.Attachment{
		{Name: "notes with spaces.txt", Mime: "text/plain", Path: txt},
		{Name: "image.png", Mime: "image/png", Path: png},
		{Name: "paper.pdf", Mime: "application/pdf", Path: pdf},
		{Name: "gone.txt", Mime: "text/plain", Path: missing},
		{Name: "directory", Mime: "text/plain", Path: dir},
	}
	if err := b.SendWith("please inspect", atts); err != nil {
		t.Fatalf("SendWith: %v", err)
	}
	claudeWait(t, "SendWith user line", 2*time.Second, func() bool { return len(claudeCapture(t, capture)) == 1 })
	got := claudeCapture(t, capture)[0]
	wireText := browsertools.PromptPreamble + "\n\n" + chatcontext.PromptPreamble + "\n\nplease inspect\n\n" + strings.Join([]string{
		"[attached file: " + strconv.Quote(txt) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(png) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(pdf) + "] Read it with your file tools.",
	}, "\n")
	want := string(claudeUserLineFor(wireText))
	if got != want {
		t.Fatalf("Claude SendWith stdin mismatch:\n got  %s\n want %s", got, want)
	}

	var sawOriginalEcho, sawStatus bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatUser && e.Msg.Text == "please inspect" && e.Msg.Meta == state.AttachMeta([]string{"notes with spaces.txt", "image.png", "paper.pdf"}) {
			sawOriginalEcho = true
		}
		if e.Kind == state.EvStatus && strings.Contains(e.Text, "could not attach gone.txt, directory") {
			sawStatus = true
		}
	}
	if !sawOriginalEcho || !sawStatus {
		t.Fatalf("SendWith events: originalEcho=%v skippedStatus=%v", sawOriginalEcho, sawStatus)
	}
}

func TestClaudeAttachmentSendWithWriteFailureReturnsErrorAndCanRetry(t *testing.T) {
	dir := t.TempDir()
	attachment := writeAttachment(t, dir, "retry.txt", "retain me")
	writeErr := errors.New("forced stdin write failure")
	failing := &claudeFailWriteCloser{err: writeErr}
	b, log := claudeWriteReadyBackend(failing)

	err := b.SendWith("retry this attachment", []state.Attachment{{
		Name: "retry.txt", Mime: "text/plain", Path: attachment,
	}})
	if !errors.Is(err, writeErr) {
		t.Fatalf("SendWith error = %v, want wrapped %v", err, writeErr)
	}
	if failing.writes != 1 {
		t.Fatalf("failed SendWith wrote %d times, want exactly once", failing.writes)
	}
	if got := string(failing.last); !strings.Contains(got, attachment) {
		t.Fatalf("failed SendWith wire omitted attachment reference: %q", got)
	}
	if b.briefed {
		t.Fatal("failed SendWith must leave the first-send preamble available for retry")
	}

	var sawUser, sawFailure bool
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatUser && e.Msg.Text == "retry this attachment" && e.Msg.Meta == state.AttachMeta([]string{"retry.txt"}) {
			sawUser = true
		}
		if e.Kind == state.EvChatBoss && strings.Contains(e.Msg.Text, "prompt failed: forced stdin write failure") && !e.Msg.Pending {
			sawFailure = true
		}
	}
	if !sawUser || !sawFailure {
		t.Fatalf("failed SendWith must preserve the attachment echo and emit failure (user=%v failure=%v): %+v", sawUser, sawFailure, log.snapshot())
	}

	// The non-nil error lets the queue retain the attachment and retry it.
	// Replacing only stdin models that retry without adding another prompt.
	capture := &claudeCaptureWriteCloser{}
	b.mu.Lock()
	b.procStdin = capture
	b.mu.Unlock()
	if err := b.SendWith("retry this attachment", []state.Attachment{{
		Name: "retry.txt", Mime: "text/plain", Path: attachment,
	}}); err != nil {
		t.Fatalf("retry SendWith: %v", err)
	}
	if len(capture.writes) != 1 {
		t.Fatalf("retry wrote %d stdin lines, want exactly one", len(capture.writes))
	}
	if got := string(capture.writes[0]); !strings.Contains(got, attachment) {
		t.Fatalf("retry wire omitted retained attachment reference: %q", got)
	}
}

func TestClaudeAttachmentSendWithStoppedAndUnstartedKeepExistingNoWriteBehavior(t *testing.T) {
	dir := t.TempDir()
	attachment := writeAttachment(t, dir, "kept.txt", "kept")

	stoppedWriter := &claudeFailWriteCloser{err: errors.New("must not write")}
	stopped, stoppedLog := claudeWriteReadyBackend(stoppedWriter)
	stopped.fl.stop()
	if err := stopped.SendWith("ignored after stop", []state.Attachment{{Name: "kept.txt", Mime: "text/plain", Path: attachment}}); err != nil {
		t.Fatalf("stopped SendWith error = %v, want nil no-op", err)
	}
	if stoppedWriter.writes != 0 || len(stoppedLog.snapshot()) != 0 {
		t.Fatalf("stopped SendWith must not write or emit, writes=%d events=%+v", stoppedWriter.writes, stoppedLog.snapshot())
	}

	unstarted := newClaudeBackend("true", dir, nil)
	unstartedLog := &claudeEventLog{}
	unstarted.fl.setEmit(unstartedLog.emit)
	if err := unstarted.SendWith("backend is absent", []state.Attachment{{Name: "kept.txt", Mime: "text/plain", Path: attachment}}); err != nil {
		t.Fatalf("unstarted SendWith error = %v, want existing dead-backend signal behavior", err)
	}
	var sawDeadBackend bool
	for _, e := range unstartedLog.snapshot() {
		if e.Kind == state.EvChatBoss && strings.Contains(e.Msg.Text, "backend not started") {
			sawDeadBackend = true
		}
	}
	if !sawDeadBackend {
		t.Fatalf("unstarted SendWith must retain its backend-not-started signal: %+v", unstartedLog.snapshot())
	}
}
