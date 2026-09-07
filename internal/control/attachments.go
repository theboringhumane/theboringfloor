package control

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/brand"
)

var attachmentSequence atomic.Uint64

const (
	// MaxAttachmentsPerMessage bounds the number of images accepted by one
	// control message.
	MaxAttachmentsPerMessage = 4
	// MaxAttachmentBytes bounds one decoded image at 8 MiB.
	MaxAttachmentBytes = 8 << 20
	// MaxAttachmentsBytes bounds all decoded images in one message at 16 MiB.
	MaxAttachmentsBytes = 16 << 20
)

var (
	// ErrTooManyAttachments identifies a message with more than four images.
	ErrTooManyAttachments = errors.New("control: too many attachments")
	// ErrAttachmentTooLarge identifies one decoded image larger than 8 MiB.
	ErrAttachmentTooLarge = errors.New("control: attachment too large")
	// ErrAttachmentsTooLarge identifies decoded images larger than 16 MiB in total.
	ErrAttachmentsTooLarge = errors.New("control: attachments too large")
	// ErrInvalidAttachmentBase64 identifies an attachment whose data is not standard base64.
	ErrInvalidAttachmentBase64 = errors.New("control: invalid attachment base64")
	// ErrUnsupportedAttachmentMIME identifies an attachment MIME type outside the image allowlist.
	ErrUnsupportedAttachmentMIME = errors.New("control: unsupported attachment MIME type")
)

// AttachmentUploadsDir returns the absolute upload directory for dir. It uses
// the same canonical dir hash and THEFLOOR_HOME-aware project root as the
// discovery record.
func AttachmentUploadsDir(dir string) (string, error) {
	path, err := filepath.Abs(filepath.Join(home(), brand.DotDir, "projects", DirHash(dir), "uploads"))
	if err != nil {
		return "", err
	}
	return path, nil
}

// SavedAttachment identifies one validated, persisted attachment.
type SavedAttachment struct {
	Name string
	Mime string
	Path string
}

// SaveAttachments validates and persists image attachments for project dir.
// Files are written under the THEFLOOR_HOME-aware per-project uploads directory
// with 0700 directory and 0600 file modes. It returns original names, validated
// MIME types, and absolute paths in request order, and removes every file written
// by this call if any validation or write fails.
func SaveAttachments(dir string, atts []Attachment) ([]SavedAttachment, error) {
	if len(atts) > MaxAttachmentsPerMessage {
		return nil, ErrTooManyAttachments
	}

	type decodedAttachment struct {
		attachment Attachment
		data       []byte
		ext        string
	}
	decoded := make([]decodedAttachment, 0, len(atts))
	total := 0
	for _, att := range atts {
		ext, ok := attachmentExtension(att.MimeType)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedAttachmentMIME, att.MimeType)
		}
		data, err := base64.StdEncoding.Strict().DecodeString(att.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAttachmentBase64, err)
		}
		if len(data) > MaxAttachmentBytes {
			return nil, ErrAttachmentTooLarge
		}
		total += len(data)
		if total > MaxAttachmentsBytes {
			return nil, ErrAttachmentsTooLarge
		}
		decoded = append(decoded, decodedAttachment{attachment: att, data: data, ext: ext})
	}
	if len(decoded) == 0 {
		return nil, nil
	}

	uploadsDir, err := AttachmentUploadsDir(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(uploadsDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(uploadsDir, 0o700); err != nil {
		return nil, err
	}

	saved := make([]SavedAttachment, 0, len(decoded))
	cleanup := func() {
		for _, attachment := range saved {
			_ = os.Remove(attachment.Path)
		}
	}
	for _, att := range decoded {
		name := fmt.Sprintf("%d-%d-%s%s", time.Now().UnixMilli(), attachmentSequence.Add(1), sanitizeAttachmentBase(att.attachment.Name), att.ext)
		path := filepath.Join(uploadsDir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			cleanup()
			return nil, err
		}
		if err := file.Chmod(0o600); err == nil {
			_, err = file.Write(att.data)
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
			cleanup()
			return nil, err
		}
		saved = append(saved, SavedAttachment{
			Name: att.attachment.Name,
			Mime: att.attachment.MimeType,
			Path: path,
		})
	}
	return saved, nil
}

func attachmentExtension(mimeType string) (string, bool) {
	switch mimeType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func sanitizeAttachmentBase(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}
	var builder strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	base := strings.TrimSuffix(builder.String(), filepath.Ext(builder.String()))
	if base == "" || base == "." || base == ".." {
		return "attachment"
	}
	return base
}
