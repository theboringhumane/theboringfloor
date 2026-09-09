package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectFileBoundaryAndPreview(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.WriteFile(filepath.Join(root, "hello.go"), []byte("package hello\n"), 0600)
	os.WriteFile(filepath.Join(outside, "private"), []byte("private"), 0600)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../private", filepath.Join(outside, "private"), "escape/private"} {
		if _, err := ReadFiles(root, name); err == nil {
			t.Fatalf("allowed %q", name)
		}
	}
	page, err := ReadFiles(root, "hello.go")
	if err != nil || page.Content != "package hello\n" || page.Directory {
		t.Fatalf("preview: %+v %v", page, err)
	}
	dir, err := ReadFiles(root, "")
	if err != nil || len(dir.Entries) != 1 || dir.Entries[0].Name != "hello.go" {
		t.Fatalf("listing: %+v %v", dir, err)
	}
	os.WriteFile(filepath.Join(root, "binary"), []byte{0, 1, 2}, 0600)
	page, err = ReadFiles(root, "binary")
	if err != nil || !page.Binary || page.Content != "" {
		t.Fatalf("binary: %+v %v", page, err)
	}
	os.WriteFile(filepath.Join(root, "large"), []byte(strings.Repeat("a", 70000)), 0600)
	page, err = ReadFiles(root, "large")
	if err != nil || !page.Truncated || len(page.Content) != 65536 {
		t.Fatalf("large length %d: %v", len(page.Content), err)
	}
}
