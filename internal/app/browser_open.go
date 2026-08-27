// browser_open.go — the app-side half of the agent browser tool
// (internal/browsertools): a state.EvBrowserOpen event means the boss
// agent emitted the ⟦open-browser: URL⟧ marker, the backend already
// stripped it from the pinned transcript AND ran the URL policy — THIS
// file only turns the verdict into UI:
//
//   - refused (BrowserOpenAllowed=false): NOTHING opens; the bridge's
//     reason lands as a red office notice (the every-local-outcome
//     noticeErr pattern) so the refusal is never silent;
//   - allowed: the dim confirmation notice posts and the browser pane's
//     EXISTING open path drives the load (Browser.Open — the fetch rides
//     a tea.Cmd, the verdict lands back as panels.BrowserPageMsg, and
//     the /open plumbing in browser.go forwards it to the pane; the
//     browserSlashNote latch stays /open-only, so no double notice).
//
// EVENT CONTRACT (the read model for the manager's glue): Kind ==
// state.EvBrowserOpen; Text == the policy-decided URL;
// BrowserOpenAllowed == the verdict; BrowserOpenReason == the
// member-facing refusal when !Allowed.
//
// The VIEW SWITCH is deliberately NOT here — the manager owns it (the
// browser pane's home is the LEFT pane's floor|browser slot; the
// switch is one line: m.leftTab = leftTabBrowser on an allowed
// EvBrowserOpen). THE WHOLE WIRING (model.go, two lines):
//
//	func (m *Model) applyEvent(ev state.Event) tea.Cmd {
//		return tea.Batch(m.pagerKick(ev), m.applyMedia(ev), m.applyEventCore(ev), m.applyBrowserOpen(ev))
//	}
//
// and, inside applyEventCore (or wherever the manager likes the
// switch), the one-line view flip:
//
//	if ev.Kind == state.EvBrowserOpen && ev.BrowserOpenAllowed {
//		m.leftTab = leftTabBrowser
//	}
//
// (optional third line: one describeEvent case so the activity entry
// reads "browser — agent open: <url>" instead of the bare-kind
// default.)
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// applyBrowserOpen — the EvBrowserOpen leg of applyEvent: nil for every
// other kind (safe as an unconditional batch leg), so the model.go
// hookup never branches.
func (m *Model) applyBrowserOpen(ev state.Event) tea.Cmd {
	if ev.Kind != state.EvBrowserOpen {
		return nil
	}
	url := strings.TrimSpace(ev.Text)
	if !ev.BrowserOpenAllowed {
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "refused by office policy"
		}
		m.noticeErr("browser: " + url + " — " + reason)
		return nil
	}
	if url == "" {
		return nil // a shapeless event is never an open (degrade silent)
	}
	if m.browser == nil {
		m.noticeErr("browser: tab unavailable — could not open " + url)
		return nil
	}
	m.notice("browser: opening " + url + " (asked by the boss)")
	return m.browser.Open(url)
}
