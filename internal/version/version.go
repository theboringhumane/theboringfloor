// Package version — build-time version stamp, one source of truth.
//
// `go build` / `go install` of the main command package
// leaves Version = "dev". Releases stamp real values via ldflags:
//
//	go build -ldflags "\
//	    -X github.com/theboringhumane/theboringfloor/internal/version.Version=0.1.0 \
//	    -X github.com/theboringhumane/theboringfloor/internal/version.Commit=$(git rev-parse --short HEAD) \
//	    -X github.com/theboringhumane/theboringfloor/internal/version.Date=$(date -u +%F)" ./cmd/<main>
//
// -X only rewrites package-level string vars — keep these as `var`, not `const`.
package version

import "strings"

// Stamped via -X at release time (see package doc).
var (
	// Version — semver of the build ("0.1.0" or "v0.1.0"); "dev" = tree build.
	Version = "dev"
	// Commit — short git SHA of the build.
	Commit = "unknown"
	// Date — build date (YYYY-MM-DD).
	Date = "unknown"
)

// String — one-line display for `theboringfloor --version`:
//
//	dev build: "theboringfloor dev"
//	stamped:   "theboringfloor v0.1.0 (abc1234, 2026-08-22)"
func String() string {
	v := strings.TrimPrefix(Version, "v") // milestones stamp both "0.1.0" and "v0.1.0" shapes
	display := "theboringfloor dev"
	if v != "dev" && v != "" {
		display = "theboringfloor v" + v
	}
	// build metadata — append only what was actually stamped
	var meta []string
	if Commit != "unknown" {
		meta = append(meta, Commit)
	}
	if Date != "unknown" {
		meta = append(meta, Date)
	}
	if len(meta) > 0 {
		display += " (" + strings.Join(meta, ", ") + ")"
	}
	return display
}
