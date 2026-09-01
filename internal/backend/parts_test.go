// parts_test.go — unit proof for OpenCode attachment transport policy.
package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

func writeAttachment(t *testing.T, dir, name, body string) string {
	return writeAttachmentBytes(t, dir, name, []byte(body))
}

func writeAttachmentBytes(t *testing.T, dir, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func textPart(t *testing.T, parts []map[string]any) string {
	t.Helper()
	if len(parts) == 0 || parts[0]["type"] != "text" {
		t.Fatalf("want leading text part, got %#v", parts)
	}
	text, ok := parts[0]["text"].(string)
	if !ok {
		t.Fatalf("text part has non-string text: %#v", parts[0])
	}
	return text
}

func TestPayloadTextOnly(t *testing.T) {
	parts, skipped := payloadParts("ship it", nil)
	if len(skipped) != 0 || len(parts) != 1 || textPart(t, parts) != "ship it" {
		t.Fatalf("text-only payload = %#v, skipped %v", parts, skipped)
	}
}

// Text, source, markdown, and archive inputs become quoted absolute path
// references, while PNG/JPEG/PDF remain real OpenCode file parts.
func TestPayloadPartsTransportPolicyAndMixedOrdering(t *testing.T) {
	dir := t.TempDir()
	txt := writeAttachment(t, dir, "notes with spaces.txt", "notes")
	goFile := writeAttachment(t, dir, "model.go", "package app")
	md := writeAttachment(t, dir, "README.md", "# title")
	archive := writeAttachment(t, dir, "bundle.zip", "zip bytes")
	png := writeAttachmentBytes(t, dir, "paste.png", []byte("\x89PNG\r\n\x1a\n"))
	jpeg := writeAttachmentBytes(t, dir, "photo.jpg", []byte("\xff\xd8\xff\xe0"))
	pdf := writeAttachment(t, dir, "paper.pdf", "%PDF-1.7\n")
	atts := []state.Attachment{
		{Name: "notes with spaces.txt", Mime: "text/plain", Path: txt},
		{Name: "paste.png", Mime: "image/png", Path: png},
		{Name: "model.go", Mime: "text/x-go", Path: goFile},
		{Name: "paper.pdf", Mime: "application/pdf", Path: pdf},
		{Name: "README.md", Mime: "text/markdown", Path: md},
		{Name: "photo.jpg", Mime: "image/jpeg", Path: jpeg},
		{Name: "bundle.zip", Mime: "application/zip", Path: archive},
	}
	parts, skipped := payloadParts("review these", atts)
	if len(skipped) != 0 {
		t.Fatalf("readable attachments skipped: %v", skipped)
	}
	wantText := "review these\n\n" + strings.Join([]string{
		"[attached file: " + strconv.Quote(txt) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(goFile) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(md) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(archive) + "] Read it with your file tools.",
	}, "\n")
	if got := textPart(t, parts); got != wantText {
		t.Fatalf("text prefix/path references mismatch:\n got %q\nwant %q", got, wantText)
	}
	if len(parts) != 4 {
		t.Fatalf("want text + PNG/PDF/JPEG, got %#v", parts)
	}
	for i, want := range []string{"paste.png", "paper.pdf", "photo.jpg"} {
		if got := parts[i+1]["filename"]; got != want {
			t.Fatalf("upload %d filename = %q, want %q", i, got, want)
		}
	}
}

func TestPayloadAttachOnlyTextAndQuoteSafePath(t *testing.T) {
	dir := t.TempDir()
	path := writeAttachment(t, dir, "line\nbell\a.txt", "plain words")
	parts, skipped := payloadParts("", []state.Attachment{{Name: "odd", Mime: "text/plain", Path: path}})
	if len(skipped) != 0 || len(parts) != 1 {
		t.Fatalf("attach-only text = %#v, skipped %v", parts, skipped)
	}
	want := "[attached file: " + strconv.Quote(path) + "] Read it with your file tools."
	if got := textPart(t, parts); got != want || strings.Contains(got, path) {
		t.Fatalf("unsafe attach-only path reference: %q (want %q)", got, want)
	}
}

func TestPayloadMissingAndNonRegularAreSkipped(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "gone.txt")
	parts, skipped := payloadParts("keep this", []state.Attachment{
		{Name: "gone.txt", Path: missing},
		{Name: "directory", Path: dir},
	})
	if got := textPart(t, parts); got != "keep this" {
		t.Fatalf("missing/nonregular path leaked into prompt: %q", got)
	}
	if strings.Join(skipped, ",") != "gone.txt,directory" {
		t.Fatalf("skipped = %v", skipped)
	}
}

func TestPayloadMimeFallback(t *testing.T) {
	dir := t.TempDir()
	pdf := writeAttachment(t, dir, "unknown", "%PDF-1.7\n")
	parts, skipped := payloadParts("", []state.Attachment{{Name: "unknown", Path: pdf}})
	if len(skipped) != 0 || len(parts) != 1 || parts[0]["type"] != "file" {
		t.Fatalf("sniffed PDF payload = %#v, skipped %v", parts, skipped)
	}
}

func TestPayloadRejectsSpoofedOrMismatchedDeclaredMedia(t *testing.T) {
	dir := t.TempDir()
	plainPNG := writeAttachment(t, dir, "plain-as-png.txt", "plain text")
	plainPDF := writeAttachment(t, dir, "plain-as-pdf.txt", "plain text")
	jpegAsPNG := writeAttachmentBytes(t, dir, "jpeg-as-png", []byte("\xff\xd8\xff\xe0"))
	binaryGo := writeAttachmentBytes(t, dir, "binary.go", []byte("\x89PNG\r\n\x1a\n"))
	atts := []state.Attachment{
		{Name: "plain-as-png.txt", Mime: "image/png", Path: plainPNG},
		{Name: "plain-as-pdf.txt", Mime: "application/pdf", Path: plainPDF},
		{Name: "jpeg-as-png", Mime: "image/png", Path: jpegAsPNG},
		{Name: "binary.go", Mime: "text/x-go", Path: binaryGo},
	}
	parts, skipped := payloadParts("inspect", atts)
	if len(skipped) != 0 || len(parts) != 1 {
		t.Fatalf("spoofed media payload = %#v, skipped %v", parts, skipped)
	}
	want := "inspect\n\n" + strings.Join([]string{
		"[attached file: " + strconv.Quote(plainPNG) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(plainPDF) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(jpegAsPNG) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(binaryGo) + "] Read it with your file tools.",
	}, "\n")
	if got := textPart(t, parts); got != want {
		t.Fatalf("spoofed media path references = %q, want %q", got, want)
	}
}

func TestPayloadUploadsValidatedMediaSignatures(t *testing.T) {
	dir := t.TempDir()
	atts := []state.Attachment{
		{Name: "image.png", Mime: "image/png", Path: writeAttachmentBytes(t, dir, "image.png", []byte("\x89PNG\r\n\x1a\n"))},
		{Name: "photo.jpg", Mime: "image/jpeg", Path: writeAttachmentBytes(t, dir, "photo.jpg", []byte("\xff\xd8\xff\xe0"))},
		{Name: "animation.gif", Mime: "image/gif", Path: writeAttachment(t, dir, "animation.gif", "GIF89a")},
		{Name: "image.webp", Mime: "image/webp", Path: writeAttachmentBytes(t, dir, "image.webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "))},
		{Name: "paper.pdf", Mime: "application/pdf", Path: writeAttachment(t, dir, "paper.pdf", "%PDF-1.7\n")},
	}
	parts, skipped := payloadParts("", atts)
	if len(skipped) != 0 || len(parts) != len(atts) {
		t.Fatalf("validated media payload = %#v, skipped %v", parts, skipped)
	}
	for i, want := range []string{"image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf"} {
		if got := parts[i]["mime"]; got != want {
			t.Fatalf("file %d MIME = %q, want %q", i, got, want)
		}
	}
}

func TestPayloadJSONContainsNoUnsupportedFilePart(t *testing.T) {
	dir := t.TempDir()
	txt := writeAttachment(t, dir, "notes.txt", "notes")
	parts, _ := payloadParts("read", []state.Attachment{{Name: "notes.txt", Mime: "text/plain", Path: txt}})
	body, err := json.Marshal(map[string]any{"parts": parts})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), `"type":"file"`) {
		t.Fatalf("text attachment leaked as file part: %s", body)
	}
}
