package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/theboringhumane/theboringfloor/internal/brand"
)

// ProjectSettings holds per-project configuration persisted in
// <project-root>/.theboringfloor/settings.json.
type ProjectSettings struct {
	BypassPermissions bool `json:"bypassPermissions"`
}

// ProjectSettingsPath returns the settings.json path for a project root.
// Returns "" when dir is "".
func ProjectSettingsPath(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, brand.DotDir, "settings.json")
}

// LoadProjectSettings reads and unmarshals the project settings file.
// Returns zero-value ProjectSettings on any error (missing, corrupt, empty dir).
func LoadProjectSettings(dir string) ProjectSettings {
	p := ProjectSettingsPath(dir)
	if p == "" {
		return ProjectSettings{}
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ProjectSettings{}
	}
	var ps ProjectSettings
	if json.Unmarshal(b, &ps) != nil {
		return ProjectSettings{}
	}
	return ps
}

// SaveProjectSettings writes project settings atomically (tmpfile + rename).
// Creates the .theboringfloor/ directory if needed. No-op when dir is "".
func SaveProjectSettings(dir string, ps ProjectSettings) error {
	if dir == "" {
		return nil
	}
	p := ProjectSettingsPath(dir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(p), ".settings-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, p)
}
