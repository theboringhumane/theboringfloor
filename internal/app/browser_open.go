// browser_open.go — the app-side half of the agent browser tool
// (internal/browsertools): a state.EvBrowserOpen / EvBrowserScreenshot /
// EvBrowserSnapshot event means the boss agent emitted one of the
// browser markers, the backend already stripped it from the pinned
// transcript AND ran the URL policy — THIS file only turns the verdict
// into UI + engine work:
//
//   - refused (BrowserOpenAllowed=false): NOTHING runs; the bridge's
//     reason lands as a red office notice (the every-local-outcome
//     noticeErr pattern) so the refusal is never silent;
//   - open allowed: the dim confirmation notice posts and the browser
//     pane's EXISTING open path drives the load (Browser.Open — the
//     fetch rides a tea.Cmd, the verdict lands back as
//     panels.BrowserPageMsg; the browserSlashNote latch stays
//     /open-only, so no double notice);
//   - screenshot allowed: the headless engine renders the page (15s
//     bound, 990x540) OFF the UI goroutine, the PNG saves under
//     <THEBORINGOFFICE_HOME or os.TempDir>/shots/<ts>-<hash8>.png, the
//     left slot flips to the browser tab, and the pane's normal open
//     rides along (the tab's own display path — NOT this file — picks
//     the shot up there); the result lands back as a state.Event
//     (BrowserToolDone=true) through Update's state.Event case and the
//     transcript notice posts: "browser: shot of <url> → <path> (asked
//     by the boss)";
//   - snapshot allowed: the headless engine reads the page (maxText
//     6000), the text+links post BACK TO THE AGENT as a synthetic
//     follow-up prompt on the SAME backend session (sendChat — the
//     harness-authored user-message precedent; bounded 8KB total), and
//     the member gets a one-line dim note (never the full text);
//   - browser-action allowed (the MUTATING sibling — click/fill/eval):
//     the request PARKS as a synthetic permission hold (the existing
//     permQueue + popover modal, toolName "browser") — actions mutate,
//     so the member ALWAYS decides, approve-once only (the modal's
//     "always" answer maps to "once"). The permAnswerMsg HOOKUP in
//     model.go routes the answer back here (consumeBrowserActionPerm):
//     approve runs the action engine (internal/browsertools/action,
//     chromedp, 20s budget, FRESH navigation per action) and posts the
//     outcome BACK TO THE AGENT as a synthetic follow-up (the
//     snapshot's send-in-closure precedent); reject posts the REJECTED
//     follow-up + a dim member row. A policy refusal posts the red
//     reason row and NEVER opens the modal.
//
// EVENT CONTRACT (the read model for the manager's glue): Kind is one
// of the four above; Text == the policy-decided URL;
// BrowserOpenAllowed == the verdict; BrowserOpenReason == the
// member-facing refusal when !Allowed. The async RESULT leg re-uses the
// same event with BrowserToolDone=true: success carries
// BrowserOpenAllowed=true + BrowserShotPath (screenshot) or
// BrowserSnapTitle/BrowserSnapLinks (snapshot) or
// BrowserActionFinalURL/BrowserActionResult (action); failure carries
// the member-facing error in BrowserOpenReason. The browser-action
// REQUEST leg additionally carries BrowserActionOp/Sel/Arg (the parsed
// action payload the hold parks).
//
// The VIEW SWITCH for EvBrowserScreenshot lives HERE (applyBrowserShot
// sets m.leftTab = leftTabBrowser on the allowed request leg —
// EvBrowserOpen's switch stays in applyEventCore, see model.go).
package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/browsertools/action"
	"github.com/theboringhumane/theboringoffice/internal/headless"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

const (
	// browserToolTimeout — the per-call bound the app imposes ON TOP of
	// the engine's own budget (a hung page never outlives either).
	browserToolTimeout = 15 * time.Second
	// browserShotWidth/browserShotHeight — the screenshot viewport box
	// (CSS pixels; the engine renders at deviceScaleFactor 2).
	browserShotWidth  = 990
	browserShotHeight = 540
	// browserSnapMaxText — the engine's text cap for one snapshot.
	browserSnapMaxText = 6000
	// browserSnapMaxFollowup — the synthetic follow-up prompt's total
	// byte bound (header + text + links): the agent's context window
	// never eats an unbounded page.
	browserSnapMaxFollowup = 8 * 1024
	// browserActionTimeout — the app-side bound ON TOP of the action
	// engine's own 20s budget (wider than the engine's, so the engine's
	// OWN classification — navigation vs selector vs eval — wins the
	// race; this only backstops a wedged chrome).
	browserActionTimeout = 25 * time.Second
	// browserActionClip — the selector/value clip width for the
	// permission modal's toolSummary (manager-pinned: 40 cols).
	browserActionClip = 40
	// browserActionMemberClip — the eval JSON's clip width in the
	// member's result row (the AGENT gets the full 4KB-capped payload;
	// the member's transcript gets the shape, not the wall).
	browserActionMemberClip = 120
)

// Engine seams — the ONE swap point per call: prod wires the real
// internal/headless engine (another wave owns it) for the read-only
// tools and internal/browsertools/action for the mutating one; tests
// pin fakes (NO live chrome in unit tests).
var (
	browserShotFn   = headless.Screenshot
	browserSnapFn   = headless.Snapshot
	browserActionFn = action.NavigateAndAct
)

// browserActionHold — one PARKED mutating request: the marker payload
// sitting behind the member's permission modal (keyed on the synthetic
// permission id) until the answer lands.
type browserActionHold struct {
	url string
	op  string
	sel string
	arg string
}

// applyBrowserOpen — the browser-tool leg of applyEvent: nil for every
// other kind (safe as an unconditional batch leg), so the model.go
// hookup never branches.
func (m *Model) applyBrowserOpen(ev state.Event) tea.Cmd {
	switch ev.Kind {
	case state.EvBrowserScreenshot:
		return m.applyBrowserShot(ev)
	case state.EvBrowserSnapshot:
		return m.applyBrowserSnap(ev)
	case state.EvBrowserAction:
		return m.applyBrowserAction(ev)
	}
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

// applyBrowserShot — the EvBrowserScreenshot leg. REQUEST: flip the
// left slot to the browser tab, drive the pane's normal open (the tab's
// own display path picks the shot up there), and fire the engine cmd.
// RESULT (BrowserToolDone): the transcript row — the path notice on
// success, the red reason row on failure.
func (m *Model) applyBrowserShot(ev state.Event) tea.Cmd {
	url := strings.TrimSpace(ev.Text)
	if ev.BrowserToolDone {
		if ev.BrowserOpenAllowed && ev.BrowserShotPath != "" {
			m.notice("browser: shot of " + url + " → " + ev.BrowserShotPath + " (asked by the boss)")
			return nil
		}
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "the headless engine failed"
		}
		m.noticeErr("browser: shot of " + url + " — " + reason)
		return nil
	}
	if !ev.BrowserOpenAllowed {
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "refused by office policy"
		}
		m.noticeErr("browser: " + url + " — " + reason)
		return nil
	}
	if url == "" {
		return nil // shapeless: never a shot (degrade silent)
	}
	if m.browser == nil {
		m.noticeErr("browser: tab unavailable — could not open " + url)
		return nil
	}
	m.leftTab = leftTabBrowser
	return tea.Batch(m.browser.Open(url), browserShotCmd(url))
}

// browserShotCmd — the engine half of the screenshot flow: render
// (bounded), save the PNG, and land the result back as a state.Event
// (Update's state.Event case routes it through applyEvent →
// applyBrowserShot's result leg — model.go stays untouched).
func browserShotCmd(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), browserToolTimeout)
		defer cancel()
		res, err := browserShotFn(ctx, url, browserShotWidth, browserShotHeight)
		if err != nil {
			return state.Event{Kind: state.EvBrowserScreenshot, Text: url,
				BrowserToolDone: true, BrowserOpenReason: err.Error()}
		}
		path, err := saveBrowserShot(res.PNG)
		if err != nil {
			return state.Event{Kind: state.EvBrowserScreenshot, Text: url,
				BrowserToolDone: true, BrowserOpenReason: "could not save the PNG: " + err.Error()}
		}
		return state.Event{Kind: state.EvBrowserScreenshot, Text: url,
			BrowserToolDone: true, BrowserOpenAllowed: true, BrowserShotPath: path}
	}
}

// browserShotsDir — the PNG landing zone: <THEBORINGOFFICE_HOME>/shots
// when the member/harness overrides home, else <os.TempDir>/shots.
func browserShotsDir() string {
	if home := os.Getenv("THEBORINGOFFICE_HOME"); home != "" {
		return filepath.Join(home, "shots")
	}
	return filepath.Join(os.TempDir(), "shots")
}

// saveBrowserShot writes one PNG as <ts>-<hash8>.png (ts = unix millis,
// hash8 = sha1(png)[:8] hex — a re-shot of the same page lands a new
// file per ts while identical bytes share the hash tail). Since wave 86
// the convention lives in the engine: this delegates to headless.SaveShot
// (same dir selection, same perms).
func saveBrowserShot(png []byte) (string, error) {
	return headless.SaveShot(png)
}

// applyBrowserSnap — the EvBrowserSnapshot leg. REQUEST: fire the
// engine cmd (the read goes BACK to the agent — no slot flip, no pane
// open). RESULT (BrowserToolDone): the member's one-line dim note, or
// the red reason row on failure.
func (m *Model) applyBrowserSnap(ev state.Event) tea.Cmd {
	url := strings.TrimSpace(ev.Text)
	if ev.BrowserToolDone {
		if ev.BrowserOpenAllowed {
			title := ""
			if ev.BrowserSnapTitle != "" {
				title = " (\"" + ev.BrowserSnapTitle + "\")"
			}
			m.notice("browser: snapshot of " + url + title + " → text + " +
				strconv.Itoa(ev.BrowserSnapLinks) + " links sent to the boss (asked by the boss)")
			return nil
		}
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "the headless engine failed"
		}
		m.noticeErr("browser: snapshot of " + url + " — " + reason)
		return nil
	}
	if !ev.BrowserOpenAllowed {
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "refused by office policy"
		}
		m.noticeErr("browser: " + url + " — " + reason)
		return nil
	}
	if url == "" {
		return nil // shapeless: never a snapshot (degrade silent)
	}
	b := m.backend
	if b == nil {
		m.noticeErr("browser: snapshot of " + url + " — no backend to read it back")
		return nil
	}
	return browserSnapCmd(b, url)
}

// browserSnapCmd — the engine half of the snapshot flow: read the page
// (bounded), build the synthetic follow-up, and post it BACK to the
// agent via the backend's normal prompt path (sendChat from inside the
// cmd — the queue flush's send-in-closure precedent; the UI goroutine
// never blocks on the engine or the wire).
func browserSnapCmd(b state.Backend, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), browserToolTimeout)
		defer cancel()
		res, err := browserSnapFn(ctx, url, browserSnapMaxText)
		if err != nil {
			return state.Event{Kind: state.EvBrowserSnapshot, Text: url,
				BrowserToolDone: true, BrowserOpenReason: err.Error()}
		}
		if err := sendChat(b, buildSnapshotFollowup(url, res), nil); err != nil {
			return state.Event{Kind: state.EvBrowserSnapshot, Text: url,
				BrowserToolDone: true, BrowserOpenReason: "could not post the snapshot back: " + err.Error()}
		}
		return state.Event{Kind: state.EvBrowserSnapshot, Text: url,
			BrowserToolDone: true, BrowserOpenAllowed: true,
			BrowserSnapTitle: res.Title, BrowserSnapLinks: len(res.Links)}
	}
}

// buildSnapshotFollowup — the EXACT synthetic follow-up the agent
// receives after a snapshot:
//
//	[theboringoffice] snapshot of <url> (title: <title>)
//	<text>
//	links: [1] <url> [2] <url> …
//
// ("links: (none)" when the page carries no anchors.) The total is
// bounded at browserSnapMaxFollowup bytes: the links tail trims first
// (a truncated list ends with " …"), then the text cuts on a rune
// boundary.
func buildSnapshotFollowup(url string, res *headless.SnapResult) string {
	head := "[theboringoffice] snapshot of " + url + " (title: " + res.Title + ")\n"
	linksLine := func(n int) string {
		if n == 0 {
			return "links: (none)"
		}
		var b strings.Builder
		b.WriteString("links:")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, " [%d] %s", i+1, res.Links[i].URL)
		}
		return b.String()
	}
	text := res.Text
	n := len(res.Links)
	// trim the links tail first (reserve 2 bytes for the " …" marker a
	// truncated list carries), then cut the text into whatever room is
	// left — the header + the links line always survive.
	for n > 0 && len(head)+len(text)+1+len(linksLine(n))+2 > browserSnapMaxFollowup {
		n--
	}
	links := linksLine(n)
	if n < len(res.Links) {
		links += " …"
	}
	if room := browserSnapMaxFollowup - len(head) - 1 - len(links); len(text) > room {
		text = cutUTF8(text, room)
	}
	return head + text + "\n" + links
}

// cutUTF8 truncates s to at most max BYTES without splitting a
// multi-byte rune (max <= 0 yields the empty string).
func cutUTF8(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

// ------------------------------------------------------------------ action
//
// The MUTATING sibling. REQUEST leg: a policy refusal posts the red
// reason row (NO modal — the URL policy speaks first); an allowed
// request PARKS as a synthetic permission hold and the member's EXISTING
// popover modal decides (handlePermissionEvent does the enqueue, the
// notify cohort ping, and the sound — one code path with the boss's own
// tool asks). The permAnswerMsg HOOKUP in model.go routes the answer to
// consumeBrowserActionPerm: approve-once (the modal's "always" maps to
// "once" — a mutating action never earns a standing grant) runs the
// engine; reject posts the REJECTED follow-up + a dim member row. The
// modal's own lifecycle is the only timeout (esc defers, /perm
// re-opens — exactly like a backend permission ask; an unanswered hold
// simply never executes).

// applyBrowserAction — the EvBrowserAction leg.
func (m *Model) applyBrowserAction(ev state.Event) tea.Cmd {
	url := strings.TrimSpace(ev.Text)
	if ev.BrowserToolDone {
		// RESULT leg: the agent follow-up already rode the wire inside
		// the cmd; only the member's transcript row lands here.
		if ev.BrowserOpenAllowed {
			m.notice("browser: action ok on " + ev.BrowserActionFinalURL + ": " +
				browserActionResultDisplay(ev) + " (approved by the member)")
			return nil
		}
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "the headless engine failed"
		}
		m.noticeErr("browser: action on " + url + " — " + reason)
		return nil
	}
	if !ev.BrowserOpenAllowed {
		reason := ev.BrowserOpenReason
		if reason == "" {
			reason = "refused by office policy"
		}
		m.noticeErr("browser: " + url + " — " + reason)
		return nil
	}
	if url == "" || ev.BrowserActionOp == "" {
		return nil // shapeless: never an action (degrade silent)
	}
	// park the hold + open the member's permission modal (the SAME
	// enqueue path a backend EvPermission takes — one queue, one
	// popover, one notify cohort).
	if m.browserActionHolds == nil {
		m.browserActionHolds = map[string]browserActionHold{}
	}
	m.browserActionSeq++
	pid := "browser-action-" + strconv.Itoa(m.browserActionSeq)
	m.browserActionHolds[pid] = browserActionHold{
		url: url, op: ev.BrowserActionOp, sel: ev.BrowserActionSel, arg: ev.BrowserActionArg,
	}
	m.handlePermissionEvent(state.Event{
		Kind:         state.EvPermission,
		PermissionID: pid,
		EmployeeName: "boss",
		ToolName:     "browser",
		ToolSummary:  browserActionSummary(ev.BrowserActionOp, ev.BrowserActionSel, ev.BrowserActionArg) + " on " + browserActionHost(url),
		ToolState:    "pending",
	})
	return nil
}

// consumeBrowserActionPerm — the permAnswerMsg HOOKUP (model.go): when
// the answered modal front is a PARKED browser action, resolve it
// LOCALLY and report handled=true (the hold's pid never rides the
// backend's AnswerPermission wire — the backend knows nothing about
// office-minted ids). "always" maps to "once": approve-once ONLY.
func (m *Model) consumeBrowserActionPerm(pid, response string) (bool, tea.Cmd) {
	hold, ok := m.browserActionHolds[pid]
	if !ok {
		return false, nil
	}
	delete(m.browserActionHolds, pid)
	b := m.backend
	summary := browserActionSummary(hold.op, hold.sel, hold.arg)
	if response == "reject" {
		m.notice("browser: action REJECTED by the member: " + summary + " on " + hold.url)
		return true, func() tea.Msg {
			if b != nil {
				if err := sendChat(b, "[theboringoffice] browser-action REJECTED by the member: "+summary+" on "+hold.url, nil); err != nil {
					return sendErrMsg{err: err}
				}
			}
			return nil
		}
	}
	// "once" AND "always" land here (always → once).
	return true, browserActionCmd(b, hold)
}

// browserActionCmd — the execution half: run the engine (bounded, OFF
// the UI goroutine), post the outcome BACK to the agent as a synthetic
// follow-up on the SAME backend session (the snapshot's send-in-closure
// precedent), and land the member's result row via the state.Event leg
// (Update's state.Event case routes it back through applyBrowserAction
// with BrowserToolDone=true).
func browserActionCmd(b state.Backend, hold browserActionHold) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), browserActionTimeout)
		defer cancel()
		summary := browserActionSummary(hold.op, hold.sel, hold.arg)
		res, err := browserActionFn(ctx, hold.url, action.Action{Op: hold.op, Sel: hold.sel, Arg: hold.arg})
		if err != nil {
			reason := err.Error()
			if b != nil {
				if serr := sendChat(b, "[theboringoffice] browser-action failed on "+hold.url+": "+summary+" — "+reason, nil); serr != nil {
					reason += " (and the follow-up post failed: " + serr.Error() + ")"
				}
			}
			return state.Event{Kind: state.EvBrowserAction, Text: hold.url,
				BrowserToolDone: true, BrowserActionOp: hold.op, BrowserOpenReason: reason}
		}
		// the follow-up phrasing: click/fill name the REQUEST; eval
		// reports the JSON-stringified result (already 4KB rune-safe
		// capped by the engine).
		tail := summary
		if hold.op == action.OpEval {
			tail = "eval → " + res.Text
		}
		if b != nil {
			if serr := sendChat(b, "[theboringoffice] browser-action ok on "+res.URL+": "+tail, nil); serr != nil {
				return state.Event{Kind: state.EvBrowserAction, Text: hold.url,
					BrowserToolDone: true, BrowserActionOp: hold.op,
					BrowserOpenReason: "the action ran but the follow-up post failed: " + serr.Error()}
			}
		}
		return state.Event{Kind: state.EvBrowserAction, Text: hold.url,
			BrowserToolDone: true, BrowserOpenAllowed: true, BrowserActionOp: hold.op,
			BrowserActionFinalURL: res.URL, BrowserActionResult: res.Text}
	}
}

// browserActionSummary — the request's ONE-LINE phrasing, shared by the
// modal's toolSummary and every follow-up/member row: `click '#buy'` /
// `fill '#q' = 'a = b'` / `eval document.title` (selector + value
// clipped at browserActionClip cols, newlines flattened by clipRunes).
func browserActionSummary(op, sel, arg string) string {
	switch op {
	case action.OpClick:
		return "click '" + clipRunes(sel, browserActionClip) + "'"
	case action.OpFill:
		return "fill '" + clipRunes(sel, browserActionClip) + "' = '" + clipRunes(arg, browserActionClip) + "'"
	case action.OpEval:
		return "eval " + clipRunes(arg, browserActionClip)
	default:
		return clipRunes(op+" '"+sel+"'", browserActionClip)
	}
}

// browserActionHost — the modal summary's "on <host>" tail (host:port
// when the URL parses, the clipped raw URL otherwise).
func browserActionHost(rawurl string) string {
	if u, err := url.Parse(rawurl); err == nil && u.Host != "" {
		return u.Host
	}
	return clipRunes(rawurl, browserActionClip)
}

// browserActionResultDisplay — the member result row's tail: the
// engine's own result string for click/fill ("clicked #buy"), the
// clipped JSON for eval (the agent's follow-up carries the full 4KB
// payload; the member's row carries the shape).
func browserActionResultDisplay(ev state.Event) string {
	if ev.BrowserActionOp == action.OpEval {
		return "eval → " + clipRunes(ev.BrowserActionResult, browserActionMemberClip)
	}
	return clipRunes(ev.BrowserActionResult, browserActionMemberClip)
}
