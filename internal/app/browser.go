// browser.go — the app-side wiring for the BROWSER tab (the pane itself
// lives in internal/panels/browser.go): its strip position, the `/open`
// slash command, the async-verdict message routing, and the quit-path
// teardown.
//
// STRIP POSITION: browser appends AFTER git — chat | terminal | agents |
// board | mail | activity | git | browser. git keeps index 6 (its old
// "last" seat; the real invariant is the floor click's activity pin at
// 5), browser takes 7. In v1 the tab is reachable by tab/shift+tab
// CYCLING ONLY — keys 1..7 are burned into the grab/quit test matrix and
// a digit "8" is wave-out (see keys.go), so TabJump deliberately never
// returns 7.
//
// `/open <url>` (the house slash contract): parse the arg, jump to the
// browser tab, kick Browser.Open (the fetch rides a tea.Cmd), and post
// the dim office notice when the verdict lands — success or error. The
// pending-notice latch (browserSlashNote) belongs to the slash flow ONLY:
// in-pane link follows and history hops stay silent.
//
// ASYNC ROUTING: BrowserPageMsg / BrowserOpenedMsg forward STRAIGHT to
// the browser panel (never through the active-tab hop) — a mid-flight tab
// switch can never misdeliver a page. BrowserLeaveMsg (the pane's q/esc)
// jumps back to the chat tab, thread-focus's esc contract.
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
)

// browserIndex is the browser tab's position in the sidebar strip
// (chat | terminal | agents | board | mail | activity | git | browser).
const browserIndex = 7

// applyOpenSlash — `/open <url>`: jump to the browser tab and start the
// load. The completion notice rides browserSlashNote (consumed by the
// BrowserPageMsg case in Update). Usage errors surface as red notices,
// the every-slash contract.
func (m *Model) applyOpenSlash(fields []string) tea.Cmd {
	if len(fields) < 2 || strings.TrimSpace(fields[1]) == "" {
		m.noticeErr("/open: usage /open <url>")
		return nil
	}
	raw := fields[1]
	if m.browser == nil {
		m.noticeErr("/open: browser tab unavailable")
		return nil
	}
	m.tabs.SetActiveByTitle("browser")
	m.browserSlashNote = raw
	return m.browser.Open(raw)
}

// closeBrowser tears the pane down on every app quit path (the
// closeTerminal twin): the navigation ring + loaded page drop.
func (m *Model) closeBrowser() {
	if m.browser != nil {
		m.browser.Close()
	}
}

// handleBrowserPage — the fetch verdict landed: forward to the pane, then
// settle the /open notice latch (success → dim "browser: <title> · <url>",
// error → the red wording the fetch produced).
func (m *Model) handleBrowserPage(msg panels.BrowserPageMsg) tea.Cmd {
	var cmd tea.Cmd
	if m.browser != nil {
		cmd = m.browser.Update(msg)
	}
	if m.browserSlashNote != "" {
		m.browserSlashNote = ""
		switch {
		case msg.Err != nil:
			m.noticeErr("browser: " + msg.Err.Error())
		case msg.Page != nil && msg.Page.Title != "":
			m.notice("browser: " + msg.Page.Title + " · " + msg.URL)
		default:
			m.notice("browser: " + msg.URL)
		}
	}
	return cmd
}

// handleBrowserOpened — the pane's OS-open verdict: forward to the pane
// (its note row carries the human-readable outcome; the activity tab is
// deliberately NOT logged from the browser — spec §8 keeps the diff
// bounded).
func (m *Model) handleBrowserOpened(msg panels.BrowserOpenedMsg) tea.Cmd {
	if m.browser != nil {
		return m.browser.Update(msg)
	}
	return nil
}

// handleBrowserLeave — the pane's q/esc: back to the chat tab (index 0).
func (m *Model) handleBrowserLeave() {
	m.tabs.SetActive(0)
}
