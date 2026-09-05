// topbar.go — one-line app bar, full width (port of node-legacy topbar.tsx):
//
//	left:  theboringfloor <version> | MODE | agents <n>
//	right: <office clock> | <project> (<branch>)   — with a projinfo.Info
//	       <office clock> | <cwd basename>         — zero-arg fallback (unchanged)
//
// app name bold white, DEMO yellow / LIVE green, agents count cyan, branch
// in the theme accent, clock + project dim, all on the inverted (BarBg) bar.
package chrome

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/projinfo"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/version"
)

// AppVersion is shown in the topbar (theboringfloor v0.2.1, or "dev" for a
// tree build). One source of truth: version.Version — releases stamp it via
// ldflags -X (see .goreleaser.yaml), so the bar can never drift stale against
// the tag the binary was cut from.
var AppVersion = shortVersion()

// shortVersion — the tag-shaped form the bar wants: "v0.2.1" stamped, "dev"
// in-tree. Accepts both "0.2.1" and "v0.2.1" stamps, mirroring
// version.String's normalization.
func shortVersion() string {
	v := strings.TrimPrefix(version.Version, "v")
	if v == "" || v == "dev" {
		return "dev"
	}
	return "v" + v
}

// OfficeClock — port of topbar.tsx officeClock: starts 09:00,
// +1 minute per ~30 ticks. Kept here on purpose: chrome does NOT import
// internal/office (dup noted: office owns the same tick clock for its staff).
func OfficeClock(tick int) string {
	if tick < 0 {
		tick = 0
	}
	minutes := (tick / 30) % (12 * 60)
	return fmt.Sprintf("%02d:%02d", 9+minutes/60, minutes%60)
}

var (
	cwdOnce  sync.Once
	cwdValue string
)

// cwdBase is the zero-Info right segment: process working dir basename,
// resolved once per process.
func cwdBase() string {
	cwdOnce.Do(func() {
		if d, err := os.Getwd(); err == nil {
			cwdValue = filepath.Base(d)
			return
		}
		cwdValue = "theboringfloor"
	})
	return cwdValue
}

// rightSeg builds the bar's right side: clock, then either the caller's
// project (+ branch) or, with no Info, the legacy cwd basename. The branch
// gets the theme Accent — the attention ink, and the one accent that stays
// legible on BarBg across all five themes (noir yellow-on-gray, paper
// amber-on-silver, mono bright-on-dark, dracula yellow-on-#44475a,
// solarized amber-on-base02).
func rightSeg(clock, fallback string, infos []projinfo.Info) string {
	proj := fallback
	var branch string
	if len(infos) > 0 {
		if infos[0].Project != "" {
			proj = infos[0].Project
		}
		branch = infos[0].Branch
	}
	right := OnBar(Dim, clock) +
		OnBar(White, " | ") +
		OnBar(Dim, proj)
	if branch != "" {
		right += OnBar(Dim, " ") + OnBar(Accent, "("+branch+")")
	}
	return right + OnBar(Dim, " ")
}

// TopBar renders the full-width top bar for one frame. Zero infos renders
// exactly the pre-project layout; one Info swaps the cwd segment for the
// caller-resolved project + branch. st.BackendName (the backend's boot
// EvStatus name hint, latched reducer-side) inserts a dim segment between
// mode and agents — the compact bar (TopBarCompact) drops it exactly like
// it drops mode, so a narrow terminal never buys it at the clock's expense.
func TopBar(st state.OfficeState, width int, infos ...projinfo.Info) string {
	mode := strings.ToUpper(string(st.Mode))
	agents := fmt.Sprintf("%d", len(st.Employees))
	clock := OfficeClock(st.Tick)

	left := OnBarBold(White, " theboringfloor "+AppVersion) +
		OnBar(White, " | ") +
		OnBar(ModeColor(st.Mode), mode)
	if st.BackendName != "" {
		left += OnBar(White, " | ") + OnBar(Dim, st.BackendName)
	}
	left += OnBar(White, " | agents ") +
		OnBar(Info, agents)
	right := rightSeg(clock, cwdBase(), infos)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	if lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "")
	}
	return Bar.Width(width).Render(line)
}

// TopBarCompact — the /compact layout's compressed top bar: the segment
// budget drops mode and cwd, keeping the app name, agents count and clock.
// With an Info the project joins after the clock — budget rule: below 60
// cells the branch drops, the project stays.
func TopBarCompact(st state.OfficeState, width int, infos ...projinfo.Info) string {
	agents := fmt.Sprintf("%d", len(st.Employees))
	clock := OfficeClock(st.Tick)

	line := OnBarBold(White, " theboringfloor "+AppVersion) +
		OnBar(White, " | agents ") +
		OnBar(Info, agents) +
		OnBar(White, " | ") +
		OnBar(Dim, clock+" ")

	if len(infos) > 0 {
		proj := infos[0].Project
		if proj == "" {
			proj = cwdBase()
		}
		seg := OnBar(White, "| ") + OnBar(Dim, proj)
		if width >= 60 && infos[0].Branch != "" {
			seg += OnBar(Dim, " ") + OnBar(Accent, "("+infos[0].Branch+")")
		}
		line += seg + OnBar(Dim, " ")
	}

	if lipgloss.Width(line) > width {
		line = ansi.Truncate(line, width, "")
	}
	return Bar.Width(width).Render(line)
}
