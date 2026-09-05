// open_url.go — the `o` hotkey's OPTIONAL terminal-browser lane, app
// side: RESOLUTION ONLY.
//
// Topology (enforced by the package direction — app imports panels,
// never the reverse): panels owns the exec chain (links.go —
// defaultOpenRunner's candidate cascade: terminal-browser FIRST on
// kitty-capable hosts with the zenbu binary on PATH, systemOpen the
// unconditional next leg; the whole chain swapped away wholesale by
// SetOpenRunnerForShot, so stubbed-runner suites never probe a real
// candidate). The app's ONLY surface is the lane RESOLVER — which tool
// the NEXT `o` press prefers, this instant — consumed by proofs (uishot
// --openurl) and tests; the open itself still travels model.go's
// openTargetCmd tea.Cmd (the UI goroutine never shell-outs mid-frame).
//
// The verdict seam (browserOpenMsg) is lane-agnostic by design: a
// successful candidate open logs the activity tab's "→ opened: <name>"
// exactly like a system open, and a cascade fall-through only ever
// surfaces the SYSTEM opener's failure in the dim "could not open:"
// row — an in-terminal browser hiccup is invisible unless the classic
// opener also fails.
package app

import (
	"github.com/theboringhumane/theboringfloor/internal/panels"
)

// BrowserTool — the panels enum ALIASED app-side (one source of truth,
// one detection table: panels.ResolveOpenToolFrom). A distinct type here
// would fork the truth — the alias keeps every caller on one decision.
type BrowserTool = panels.OpenTool

const (
	// BrowserToolSystemOpen — the classic fallback leg (`open -g` /
	// `xdg-open`).
	BrowserToolSystemOpen = panels.OpenToolSystemOpen
	// BrowserToolTerminalBrowser — the OPTIONAL candidate leg: zenbu's
	// terminal-browser (`terminal-browser open <target>`, a full
	// Chromium app painting pages over the kitty graphics protocol).
	BrowserToolTerminalBrowser = panels.OpenToolTerminalBrowser
)

// ResolveBrowserTool — the next `o` open's preferred lane, resolved
// FRESH per call (panels.ResolveOpenTool — env + PATH probed at use
// time, so THEFLOOR_NO_TERMINAL_BROWSER=1 and an install or
// uninstall land immediately). terminal-browser requires ALL of: no
// kill-switch env, a kitty-capable host (ghostty/kitty/wezterm or
// KITTY_WINDOW_ID; tmux is a default miss, iTerm2/Apple Terminal a
// dead-end) AND the binary on PATH — every miss prefers the system
// opener, and the runner still CASCADES there on a candidate exec
// failure.
func ResolveBrowserTool() BrowserTool { return panels.ResolveOpenTool() }
