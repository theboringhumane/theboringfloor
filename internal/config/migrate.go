package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/theboringhumane/theboringfloor/internal/brand"
)

// MigrateHome folds prior product dirs into the canonical names:
//
//	~/.grafeio            → ~/.theboringfloor (fill holes)
//	~/.theboringoffice    → ~/.theboringfloor (overwrite)
//	~/.theboringfloor.bak → ~/.theboringfloor (fill holes)
//	~/.config/grafeio           → ~/.config/theboringfloor (fill holes)
//	~/.config/theboringoffice   → ~/.config/theboringfloor (overwrite)
func MigrateHome() {
	root := homeRoot()
	if root == "" {
		return
	}
	dst := filepath.Join(root, brand.DotDir)
	mergeTree(filepath.Join(root, brand.GrafeioDotDir), dst, false)
	mergeTree(filepath.Join(root, brand.OfficeDotDir), dst, true)
	mergeTree(dst+".bak", dst, false)
	MigrateThemeDirs()
}

// MigrateThemeDirs folds ~/.config/grafeio and ~/.config/theboringoffice
// into ~/.config/theboringfloor. Idempotent. Used from Load and from
// chrome.LoadPersistedTheme (theme tests never call Load).
func MigrateThemeDirs() {
	xdg := themeConfigRoot()
	if xdg == "" {
		return
	}
	dst := filepath.Join(xdg, brand.ThemeDir)
	mergeTree(filepath.Join(xdg, brand.GrafeioThemeDir), dst, false)
	mergeTree(filepath.Join(xdg, brand.OfficeThemeDir), dst, true)
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

// mergeTree copies src into dst then deletes src. If dst is missing, this
// is os.Rename. overwrite=true replaces existing files (office-era data
// beats a first-boot stub); overwrite=false only fills holes (grafeio).
func mergeTree(src, dst string, overwrite bool) {
	if src == "" || dst == "" || src == dst {
		return
	}
	st, err := os.Stat(src)
	if err != nil || !st.IsDir() {
		return
	}
	if _, err := os.Stat(dst); err != nil {
		_ = os.Rename(src, dst)
		return
	}
	_ = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			_ = os.MkdirAll(target, 0o755)
			return nil
		}
		if !overwrite {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil
		}
		_ = copyFile(path, target)
		return nil
	})
	_ = os.RemoveAll(src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
