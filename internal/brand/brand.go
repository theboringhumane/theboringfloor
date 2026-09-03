// Package brand is the public product name. GitHub module path stays
// github.com/theboringhumane/theboringoffice; this package is what humans
// see: CLI, dirs, env, OS banners, wire prefix.
package brand

import "os"

const (
	CLI     = "theboringfloor"
	Display = "theboringfloor"
	Wire    = "[theboringfloor]"
	DotDir  = ".theboringfloor"

	// Prior product dirs — read fallbacks only; writes land on DotDir.
	OfficeDotDir  = ".theboringoffice"
	GrafeioDotDir = ".grafeio"

	ThemeDir        = "theboringfloor"
	OfficeThemeDir  = "theboringoffice"
	GrafeioThemeDir = "grafeio"
)

var envPrefixes = []string{"THEBORINGFLOOR_", "BORINGFLOOR_", "THEBORINGOFFICE_", "GRAFEIO_"}

// Get reads THEBORINGFLOOR_<suffix>, then prior prefixes.
func Get(suffix string) string {
	for _, p := range envPrefixes {
		if v := os.Getenv(p + suffix); v != "" {
			return v
		}
	}
	return ""
}
