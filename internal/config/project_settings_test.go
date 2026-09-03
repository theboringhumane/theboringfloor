package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/brand"
)

func TestProjectSettingsPath(t *testing.T) {
	if got := ProjectSettingsPath(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got := ProjectSettingsPath("/tmp/proj")
	want := filepath.Join("/tmp/proj", brand.DotDir, "settings.json")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadProjectSettings_MissingFile(t *testing.T) {
	ps := LoadProjectSettings(t.TempDir())
	if ps.BypassPermissions {
		t.Fatal("expected zero value for missing file")
	}
}

func TestLoadProjectSettings_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, brand.DotDir)
	os.MkdirAll(settingsDir, 0o755)
	os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte("{corrupt"), 0o644)
	ps := LoadProjectSettings(dir)
	if ps.BypassPermissions {
		t.Fatal("expected zero value for corrupt JSON")
	}
}

func TestLoadProjectSettings_EmptyDir(t *testing.T) {
	ps := LoadProjectSettings("")
	if ps.BypassPermissions {
		t.Fatal("expected zero value for empty dir")
	}
}

func TestProjectSettings_SaveLoadRoundTrip_True(t *testing.T) {
	dir := t.TempDir()
	if err := SaveProjectSettings(dir, ProjectSettings{BypassPermissions: true}); err != nil {
		t.Fatal(err)
	}
	ps := LoadProjectSettings(dir)
	if !ps.BypassPermissions {
		t.Fatal("expected BypassPermissions=true after round-trip")
	}
}

func TestProjectSettings_SaveLoadRoundTrip_False(t *testing.T) {
	dir := t.TempDir()
	if err := SaveProjectSettings(dir, ProjectSettings{BypassPermissions: false}); err != nil {
		t.Fatal(err)
	}
	ps := LoadProjectSettings(dir)
	if ps.BypassPermissions {
		t.Fatal("expected BypassPermissions=false after round-trip")
	}
}

func TestProjectSettings_SaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, brand.DotDir)
	if _, err := os.Stat(settingsDir); !os.IsNotExist(err) {
		t.Fatal("directory should not exist before save")
	}
	if err := SaveProjectSettings(dir, ProjectSettings{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(settingsDir)
	if err != nil {
		t.Fatalf("directory should exist after save: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}

func TestProjectSettings_SaveEmptyDir_NoOp(t *testing.T) {
	if err := SaveProjectSettings("", ProjectSettings{BypassPermissions: true}); err != nil {
		t.Fatalf("expected nil error for empty dir, got %v", err)
	}
}

func TestProjectSettings_SaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	// Write initial value.
	if err := SaveProjectSettings(dir, ProjectSettings{BypassPermissions: true}); err != nil {
		t.Fatal(err)
	}
	// Overwrite — if non-atomic, a concurrent reader could see partial content.
	// We verify no temp files are left behind and the file is valid JSON.
	if err := SaveProjectSettings(dir, ProjectSettings{BypassPermissions: false}); err != nil {
		t.Fatal(err)
	}
	settingsDir := filepath.Join(dir, brand.DotDir)
	entries, _ := os.ReadDir(settingsDir)
	for _, e := range entries {
		if e.Name() != "settings.json" {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}
	ps := LoadProjectSettings(dir)
	if ps.BypassPermissions {
		t.Fatal("expected BypassPermissions=false after atomic overwrite")
	}
}
