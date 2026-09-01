// claude_attachment_test.go — attachment SendWith must stay on Claude's
// verified text-only JSONL wire until a media schema is independently proven.
package backend

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
	"github.com/theboringhumane/theboringoffice/internal/state"
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
	wireText := browsertools.PromptPreamble + "\n\nplease inspect\n\n" + strings.Join([]string{
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
