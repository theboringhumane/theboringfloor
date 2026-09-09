// Browser-opening compatibility aliases. External links use the OS browser.
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

// ResolveBrowserTool selects the system browser.
func ResolveBrowserTool() BrowserTool { return panels.ResolveOpenTool() }
