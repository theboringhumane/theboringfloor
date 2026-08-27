// browser.go — the app-side wiring for the BROWSER surface: the LEFT
// pane's floor|browser switcher, the `/open` slash command, the
// async-verdict message routing, and the quit-path teardown. (The pane
// itself lives in internal/panels/browser.go.)
//
// THE LEFT SWITCHER: the left pane is a two-tab slot — "floor" (the
// office floor, the default, rendered through the SAME
// office.CachedStyled seam as before, one row shorter for the strip) and
// "browser" (the in-TUI page viewer). A one-row strip on top carries the
// two segments (active = accent bg, inactive = gray — the right strip's
// chrome classes) plus the dim "ctrl+b" toggle hint. ctrl+b flips the
// slot from every surface except a CAPTURED terminal (there the shell
// owns 0x02 like every other byte — the tab/shift+tab claim tier). The
// RIGHT sidebar strip is untouched by the switcher: chat | terminal |
// agents | board | mail | activity | git keep their indexes AND their
// 1..7 digit jumps; tab/shift+tab still cycle THAT strip only.
//
// `/open <url>` (the house slash contract): parse the arg, flip the left
// slot to the browser, kick Browser.Open (the fetch rides a tea.Cmd), and
// post the dim office notice when the verdict lands — success or error.
// The pending-notice latch (browserSlashNote) belongs to the slash flow
// ONLY: in-pane link follows and history hops stay silent.
//
// ASYNC ROUTING: BrowserPageMsg / BrowserOpenedMsg forward STRAIGHT to
// the browser panel (never through the active-tab hop) — a mid-flight
// switch can never misdeliver a page. BrowserLeaveMsg (the pane's q/esc)
// flips the slot back to the floor.
//
// PREMIUM LANE LIFECYCLE: the pane consults its lane controller on every
// open (kitty-capable host + `terminal-browser` on PATH + no kill-switch
// → the embedded zenbu child paints the slot); THIS file owns the flips —
// leaving the slot (ctrl+b to the floor, the pane's q/esc) SUSPENDS the
// lane (the child is group-killed + reaped, never leaked behind the
// floor), returning RESUMES it (a fresh spawn for the current url unless
// it fell back before), and the quit paths seal it through Close.
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/office"
	"github.com/theboringhumane/theboringoffice/internal/panels"
)

// The left pane's two-tab slot: floor (the default) | browser.
const (
	leftTabFloor = iota
	leftTabBrowser
)

// browserActive reports whether the browser owns the floor slot's keys
// right now: the switcher sits on browser AND the plan pane isn't
// covering the slot (a presented/edited plan takes the whole left pane —
// the browser waits behind it, switcher position kept).
func (m *Model) browserActive() bool {
	return m.leftTab == leftTabBrowser && !m.planPaneVisible()
}

// toggleLeftTab — ctrl+b: flip the floor slot floor ↔ browser. Leaving
// the browser SUSPENDS its premium lane (the embedded child dies with the
// flip — never a leak behind the floor); returning RESUMES it (a fresh
// spawn for the current url unless that url fell back before — the
// controller's no-flap latch).
func (m *Model) toggleLeftTab() {
	if m.leftTab == leftTabBrowser {
		m.leftTab = leftTabFloor
		if m.browser != nil {
			m.browser.SuspendLane()
		}
	} else {
		m.leftTab = leftTabBrowser
		if m.browser != nil {
			m.browser.ResumeLane()
		}
	}
}

// leftPaneView — the floor slot for Frame: the switcher strip row over
// the active left surface. The floor renders through the SAME
// office.CachedStyled seam as before (one row shorter — the strip row);
// the browser pane paints its own bar + body into the rest, sized by
// resize() to exactly this content area.
func (m Model) leftPaneView(w, h int) string {
	contentH := h - 1
	if contentH < 1 {
		contentH = 1
	}
	var content string
	if m.leftTab == leftTabBrowser && m.browser != nil {
		content = m.browser.View()
	} else {
		content = office.CachedStyled(m.st, w, contentH)
	}
	return lipgloss.NewStyle().Width(w).Height(h).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.leftStripView(w), content))
}

// leftStripView — the one-row floor|browser tab strip: the active segment
// in the accent class, the other gray (the right strip's chrome classes),
// the ctrl+b hint dim. Never wider than w.
func (m Model) leftStripView(w int) string {
	floorSeg := chrome.TabInactive.Render(" floor ")
	browserSeg := chrome.TabInactive.Render(" browser ")
	if m.leftTab == leftTabBrowser {
		browserSeg = chrome.TabActive.Render(" browser ")
	} else {
		floorSeg = chrome.TabActive.Render(" floor ")
	}
	bar := floorSeg + " " + browserSeg + chrome.DimText.Render("  · ctrl+b")
	return ansi.Truncate(bar, w, "")
}

// applyOpenSlash — `/open <url>`: flip the left slot to the browser and
// start the load (opening a URL auto-switches the left pane). The
// completion notice rides browserSlashNote (consumed by the
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
	m.leftTab = leftTabBrowser
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

// ---------------------------------------------------------------------------
// harness seams (the uishot live-lane proof reads the lane posture through
// these — it never instantiates the controller directly)
// ---------------------------------------------------------------------------

// BrowserPremiumActive — the left-pane browser's premium embed (the
// embedded zenbu terminal-browser child) is live RIGHT NOW.
func (m Model) BrowserPremiumActive() bool {
	return m.browser != nil && m.browser.PremiumActive()
}

// BrowserLaneGridHas — needle appears in the live premium child's screen
// model (the proof's paint-convergence read: poll this until the child's
// bytes land, then snapshot the frame). False while the text lane paints.
func (m Model) BrowserLaneGridHas(needle string) bool {
	return m.browser != nil && m.browser.LaneGridHas(needle)
}

// BrowserLanePoll — the harness's poll ride: runs the pane's per-frame
// lane check (a dropped child lands the fallback contract) WITHOUT
// waiting on a frame render or a state event (the frame cache would
// otherwise gate the observation).
func (m Model) BrowserLanePoll() {
	if m.browser != nil {
		m.browser.PollLane()
	}
}

// handleBrowserLeave — the pane's q/esc: back to the floor tab (the right
// strip never moves) AND the lane session closes (the premium child never
// outlives the slot).
func (m *Model) handleBrowserLeave() {
	if m.browser != nil {
		m.browser.SuspendLane()
	}
	m.leftTab = leftTabFloor
}

// ---------------------------------------------------------------------------
// the premium lane's frame-splice registry publish (panels/zenbu_frame.go)
// ---------------------------------------------------------------------------

// zenbuGridOriginY — the ABSOLUTE 0-based row of the embedded grid's row 0,
// identical in the desktop and mobile layouts (Frame's branch structure is
// the source of truth; the stack ABOVE the grid is the same in both):
// topbar 1 row (Frame's `top`) + the left pane's switcher strip 1 row
// (leftPaneView) + the RegionView's badge/strip row 1 row (the controller's
// body starts on RegionView row 1). The grid's column origin is always 0
// (the left pane/band paints at x=0 in both layouts).
const zenbuGridOriginY = 3

// publishZenbuFrame — Frame()'s registry write (called once per RENDERED
// frame, after the frame composed): the premium lane's absolute origin +
// live images + drained deletes, or the empty state whenever the lane
// paints nothing this frame (the wrapper then emits nothing and
// diff-deletes whatever it emitted before). A cache-HIT Frame never
// reaches here — and needs to: every field the origin depends on is
// digest-covered, so an unchanged digest means an unchanged entry.
func (m Model) publishZenbuFrame() {
	reg := panels.ZenbuRegistry()
	ox, oy, ok := m.browserGridOrigin()
	if !ok {
		reg.Clear()
		return
	}
	imgs, deletes := m.browser.LaneFrameState()
	reg.Publish(true, ox, oy, imgs, deletes)
}

// browserGridOrigin — the ABSOLUTE cell origin of the premium lane's body
// grid THIS frame, mirroring Frame()'s branch structure exactly: ok=false
// (the registry clears) whenever the browser's RegionView is not painted —
// zen owns the whole middle, thread focus owns the middle, the DESKTOP plan
// pane owns the left slot (mobile's plan swaps only the panel below the
// band — the browser band still paints), the floor tab is showing, or the
// lane is not premium-active (text lane / fell back / closed).
func (m Model) browserGridOrigin() (originX, originY int, ok bool) {
	if m.browser == nil || m.leftTab != leftTabBrowser || !m.browser.PremiumActive() {
		return 0, 0, false
	}
	if m.zen || m.threadFocus != nil {
		return 0, 0, false
	}
	if !m.mobile() && m.planPaneVisible() {
		return 0, 0, false // desktop: the plan owns the floor slot
	}
	return 0, zenbuGridOriginY, true
}
