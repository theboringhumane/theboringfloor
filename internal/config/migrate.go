package config

import (
	"os"
	"path/filepath"
)

// MigrateHome is retained for callers from prior releases. State is read and
// written exclusively beneath the canonical directory.
func MigrateHome() {
}

// MigrateThemeDirs is retained for callers from prior releases. Themes are
// read and written exclusively beneath the canonical directory.
func MigrateThemeDirs() {
}

func themeConfigRoot() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return d
	}
	if HomeOverride() != "" {
		return filepath.Join(homeRoot(), ".config")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config")
}
