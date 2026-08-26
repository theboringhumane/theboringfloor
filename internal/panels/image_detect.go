// image_detect.go — the terminal image-protocol determine layer, v1.
//
// Pure env reads, zero shell-outs, zero network, NEVER panics: every
// missing/unknown terminal folds to a lane that still renders (ASCII
// half-blocks paint everywhere; NoneLane is reserved for a byte-dumb
// TERM). The lanes are the upgrade path: v1 renders the ASCII lane only,
// but the detection truth table is settled NOW (kitty → iterm2 → sixel →
// ascii) so the richer transports slot in behind one switch later.
//
// Credence map (terminal-reported env, strongest claim first):
//
//	TMUX != ""                    → ASCII (conservative fold: passthrough
//	  to the outer terminal May work on some tmux builds, but the
//	  graphics escape getting through is never guaranteed — v1 keeps the
//	  lane that renders everywhere rather than trust it)
//	KITTY_WINDOW_ID != ""         → Kitty (kitty's own marker beats TERM_PROGRAM)
//	TERM_PROGRAM ghostty|kitty    → Kitty (both speak the kitty graphics protocol)
//	TERM_PROGRAM iTerm.app|iTerm  → ITerm (OSC 1337 inline images)
//	WEZTERM_UNIX_SOCKET != ""     → ITerm (WezTerm implements the iterm2 image API)
//	TERM_PROGRAM WezTerm          → ITerm (same, noted by name only)
//	VSCODE_PID != ""              → ITerm (the VS Code integrated terminal
//	  implements the iterm2/OSC1337 lane for inline images)
//	TERM_PROGRAM vscode           → ITerm (same, the documented env mark)
//	TERM contains "sixel"         → Sixel (rare but honest when it shows)
//	TERM "" or "dumb"             → None (no credible color capability at all)
//	anything else                 → ASCII (xterm/alacritty/mintty/unknowns —
//	  half-block truecolor renders regardless)
//
// The result is advisory: the auto chain (kitty → iterm → ascii) and the
// explicit /images ascii lane both paint through the ASCII rasterizer in
// v1, so a wrong pick never blanks an image — it picks which transport
// code would take over first when the native lanes land.
package panels

import (
	"os"
	"strings"
)

// ImageLane — the terminal's image-protocol transport.
type ImageLane int

const (
	// ASCIILane — half-block truecolor cells (the v1 renderer; paints in
	// every color terminal, the universal fallback).
	ASCIILane ImageLane = iota
	// KittyLane — kitty's graphics protocol (ghostty speaks it too).
	KittyLane
	// ITermLane — iTerm2's OSC 1337 inline images (WezTerm + VS Code
	// integrated terminal implement the same API).
	ITermLane
	// SixelLane — DEC Sixel (TERM-sensed only in v1).
	SixelLane
	// NoneLane — no credible color terminal (TERM unset/dumb): chips
	// only, raster rows would be noise.
	NoneLane
)

// String — the lane word for notices/tests.
func (l ImageLane) String() string {
	switch l {
	case KittyLane:
		return "kitty"
	case ITermLane:
		return "iterm"
	case SixelLane:
		return "sixel"
	case NoneLane:
		return "none"
	default:
		return "ascii"
	}
}

// DetectImageSupport reads the live process env.
func DetectImageSupport() ImageLane { return DetectImageSupportFrom(os.Getenv) }

// DetectImageSupportFrom is the pure core (env injected so the matrix is
// a shell-out-free table test). Reads ONLY; never panics.
func DetectImageSupportFrom(env func(string) string) ImageLane {
	get := func(k string) string { return strings.TrimSpace(env(k)) }
	// tmux first: the escapes would have to pass through; v1 folds
	// conservative rather than trust passthrough.
	if get("TMUX") != "" {
		return ASCIILane
	}
	// kitty's own marker beats every outer-PROGRAM name.
	if get("KITTY_WINDOW_ID") != "" {
		return KittyLane
	}
	switch strings.ToLower(get("TERM_PROGRAM")) {
	case "ghostty", "kitty":
		return KittyLane
	case "iterm.app", "iterm":
		return ITermLane
	case "wezterm", "vscode":
		return ITermLane
	}
	if get("WEZTERM_UNIX_SOCKET") != "" {
		return ITermLane
	}
	if get("VSCODE_PID") != "" {
		return ITermLane
	}
	term := strings.ToLower(get("TERM"))
	if strings.Contains(term, "sixel") {
		return SixelLane
	}
	if term == "" || term == "dumb" {
		return NoneLane
	}
	return ASCIILane
}
