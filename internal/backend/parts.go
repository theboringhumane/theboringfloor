// parts.go — building the prompt_async parts array: one text part plus file
// parts only for OpenCode-supported media; other chat-input attachments are
// represented by safe path references. Split out of opencode.go so the wire
// shape is unit-testable without a server (parts_test.go).
//
// Wire contract (verified two ways, 2026-08-21):
//   - GET /doc on a spawned `opencode serve` 1.18.19: POST
//     /session/{sessionID}/prompt_async (operationId session.prompt_async)
//     takes parts[] of TextPartInput | FilePartInput | …; FilePartInput =
//     {"type":"file","mime":string,"filename"?:string,"url":string}
//     (required: type, mime, url; additionalProperties false).
//   - live POST against the same server: a file part with a base64 data
//     URL is accepted (HTTP 204) — see postPrompt.
package backend

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// preparedAttachment is an attachment proven readable at send time. Paths
// are absolute so a model can use them even when its working directory moves.
type preparedAttachment struct {
	attachment state.Attachment
	data       []byte
	mime       string
	path       string
}

// prepareAttachments is the shared send-time attachment gate. It accepts
// regular, readable files only; missing, directories, and unreadable paths
// are reported by display name and never produce a stale path reference.
func prepareAttachments(atts []state.Attachment) (prepared []preparedAttachment, skipped []string) {
	for _, att := range atts {
		info, err := os.Stat(att.Path)
		if err != nil || !info.Mode().IsRegular() {
			skipped = append(skipped, att.Name)
			continue
		}
		data, err := os.ReadFile(att.Path)
		if err != nil {
			skipped = append(skipped, att.Name)
			continue
		}
		path, err := filepath.Abs(att.Path)
		if err != nil {
			skipped = append(skipped, att.Name)
			continue
		}
		mime := validatedUploadMime(att.Mime, data)
		prepared = append(prepared, preparedAttachment{attachment: att, data: data, mime: mime, path: path})
	}
	return prepared, skipped
}

// attachmentPrompt appends safe, tool-readable path references for prepared
// attachments the transport cannot upload. The user text remains an exact
// prefix; references follow after one clean paragraph break (or stand alone
// for an attach-only send).
func attachmentPrompt(text string, prepared []preparedAttachment, upload func(string) bool) string {
	var refs []string
	for _, att := range prepared {
		if upload(att.mime) {
			continue
		}
		refs = append(refs, "[attached file: "+strconv.Quote(att.path)+"] Read it with your file tools.")
	}
	if len(refs) == 0 {
		return text
	}
	if text == "" {
		return strings.Join(refs, "\n")
	}
	return text + "\n\n" + strings.Join(refs, "\n")
}

func openCodeUploadMime(mime string) bool {
	switch strings.ToLower(mime) {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf":
		return true
	default:
		return false
	}
}

// validatedUploadMime returns an upload MIME only when the bytes identify as
// one of the supported media types and any caller-supplied MIME agrees exactly.
// Attachment metadata comes from the picker, so it is advisory: it must never
// turn arbitrary bytes into an uploadable file part. A declared non-media type
// is likewise intentionally kept on the safe path-reference route.
func validatedUploadMime(declared string, data []byte) string {
	sniffed := strings.ToLower(http.DetectContentType(headBytes(data)))
	if sniffed == "application/pdf" && !bytes.HasPrefix(data, []byte("%PDF-")) {
		return ""
	}
	if !openCodeUploadMime(sniffed) {
		return ""
	}
	declared = strings.ToLower(strings.TrimSpace(declared))
	if declared != "" && declared != sniffed {
		return ""
	}
	return sniffed
}

// payloadParts builds the parts array for prompt_async. OpenCode receives
// real file parts only for images and PDFs; every other readable regular file
// becomes a quoted absolute path reference in the text prompt. Unreadable
// attachments are skipped so callers can surface one status note.
func payloadParts(text string, atts []state.Attachment) (parts []map[string]any, skipped []string) {
	prepared, skipped := prepareAttachments(atts)
	text = attachmentPrompt(text, prepared, openCodeUploadMime)
	if text != "" {
		parts = append(parts, map[string]any{"type": "text", "text": text})
	}
	for _, att := range prepared {
		if !openCodeUploadMime(att.mime) {
			continue
		}
		parts = append(parts, map[string]any{
			"type":     "file",
			"mime":     att.mime,
			"filename": att.attachment.Name,
			"url":      "data:" + att.mime + ";base64," + base64.StdEncoding.EncodeToString(att.data),
		})
	}
	return parts, skipped
}

// headBytes returns the first ≤512 bytes — DetectContentType's sniff
// window — of an already-read file body.
func headBytes(data []byte) []byte {
	if len(data) > 512 {
		return data[:512]
	}
	return data
}

// attachmentNames projects attachments to their display names (the user
// bubble's Meta carrier and demo ack both speak in names, never paths).
func attachmentNames(atts []state.Attachment) []string {
	if len(atts) == 0 {
		return nil
	}
	names := make([]string, len(atts))
	for i, a := range atts {
		names[i] = a.Name
	}
	return names
}

// preparedAttachmentNames returns only attachments that survived the
// send-time readability gate, retaining their original ordering for the user
// bubble metadata.
func preparedAttachmentNames(prepared []preparedAttachment) []string {
	if len(prepared) == 0 {
		return nil
	}
	names := make([]string, len(prepared))
	for i, att := range prepared {
		names[i] = att.attachment.Name
	}
	return names
}
