package control

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAttachmentsPersistsAllowedImages(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	paths, err := SaveAttachments("/workspace/project", []Attachment{
		{Name: "photo.png", MimeType: "image/png", Data: base64.StdEncoding.EncodeToString([]byte("png"))},
		{Name: "photo.jpeg", MimeType: "image/jpeg", Data: base64.StdEncoding.EncodeToString([]byte("jpeg"))},
		{Name: "animation.gif", MimeType: "image/gif", Data: base64.StdEncoding.EncodeToString([]byte("gif"))},
		{Name: "art.webp", MimeType: "image/webp", Data: base64.StdEncoding.EncodeToString([]byte("webp"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("paths = %d, want 4", len(paths))
	}
	for index, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("file[%d] mode = %o, want 600", index, got)
		}
	}
	uploads, err := AttachmentUploadsDir("/workspace/project")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(uploads)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("uploads mode = %o, want 700", got)
	}
}

func TestSaveAttachmentsRejectsInvalidInputsAndLeavesNoFiles(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	data := base64.StdEncoding.EncodeToString([]byte("image"))
	cases := []struct {
		name string
		atts []Attachment
		want error
	}{
		{"five", []Attachment{{MimeType: "image/png", Data: data}, {MimeType: "image/png", Data: data}, {MimeType: "image/png", Data: data}, {MimeType: "image/png", Data: data}, {MimeType: "image/png", Data: data}}, ErrTooManyAttachments},
		{"bad base64", []Attachment{{MimeType: "image/png", Data: "%%%"}}, ErrInvalidAttachmentBase64},
		{"text MIME", []Attachment{{MimeType: "text/plain", Data: data}}, ErrUnsupportedAttachmentMIME},
		{"PDF MIME", []Attachment{{MimeType: "application/pdf", Data: data}}, ErrUnsupportedAttachmentMIME},
		{"oversize", []Attachment{{MimeType: "image/png", Data: base64.StdEncoding.EncodeToString(make([]byte, MaxAttachmentBytes+1))}}, ErrAttachmentTooLarge},
		{"total oversize", []Attachment{{MimeType: "image/png", Data: base64.StdEncoding.EncodeToString(make([]byte, 6<<20))}, {MimeType: "image/png", Data: base64.StdEncoding.EncodeToString(make([]byte, 6<<20))}, {MimeType: "image/png", Data: base64.StdEncoding.EncodeToString(make([]byte, 6<<20))}}, ErrAttachmentsTooLarge},
		{"later failure", []Attachment{{MimeType: "image/png", Data: data}, {MimeType: "image/png", Data: "%%%"}}, ErrInvalidAttachmentBase64},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := SaveAttachments("/workspace/project", test.atts)
			if !errors.Is(err, test.want) {
				t.Fatalf("SaveAttachments() error = %v, want %v", err, test.want)
			}
			uploads, err := AttachmentUploadsDir("/workspace/project")
			if err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(uploads)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("failed SaveAttachments left files: %v", entries)
			}
		})
	}
}

func TestSaveAttachmentsSanitizesTraversalNames(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := "/workspace/project"
	uploads, err := AttachmentUploadsDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	data := base64.StdEncoding.EncodeToString([]byte("png"))
	for _, name := range []string{"../../etc/passwd", "..", "/abs/path", "a/b/c.png", ""} {
		t.Run(name, func(t *testing.T) {
			paths, err := SaveAttachments(dir, []Attachment{{Name: name, MimeType: "image/png", Data: data}})
			if err != nil {
				t.Fatal(err)
			}
			if len(paths) != 1 {
				t.Fatalf("paths = %v", paths)
			}
			rel, err := filepath.Rel(filepath.Clean(uploads), filepath.Clean(paths[0]))
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				t.Fatalf("saved path %q escapes uploads directory %q", paths[0], uploads)
			}
		})
	}
}

func TestFormatMessageWithAttachments(t *testing.T) {
	if got, want := FormatMessageWithAttachments("", []string{"/tmp/image.png"}), "[[theboringfloor-attachments]]\n/tmp/image.png\n[[/theboringfloor-attachments]]"; got != want {
		t.Fatalf("FormatMessageWithAttachments() = %q, want %q", got, want)
	}
}
