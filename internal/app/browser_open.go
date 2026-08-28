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
//     the member gets a one-line dim note (never the full text).
//
// EVENT CONTRACT (the read model for the manager's glue): Kind is one
// of the three above; Text == the policy-decided URL;
// BrowserOpenAllowed == the verdict; BrowserOpenReason == the
// member-facing refusal when !Allowed. The async RESULT leg re-uses the
// same event with BrowserToolDone=true: success carries
// BrowserOpenAllowed=true + BrowserShotPath (screenshot) or
// BrowserSnapTitle/BrowserSnapLinks (snapshot); failure carries the
// member-facing error in BrowserOpenReason.
//
// The VIEW SWITCH for EvBrowserScreenshot lives HERE (applyBrowserShot
// sets m.leftTab = leftTabBrowser on the allowed request leg —
// EvBrowserOpen's switch stays in applyEventCore, see model.go).
package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

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
)

// Engine seams — the ONE swap point per call: prod wires the real
// internal/headless engine (another wave owns it); tests pin fakes (NO
// live chrome in unit tests).
var (
	browserShotFn = headless.Screenshot
	browserSnapFn = headless.Snapshot
)

// applyBrowserOpen — the browser-tool leg of applyEvent: nil for every
// other kind (safe as an unconditional batch leg), so the model.go
// hookup never branches.
func (m *Model) applyBrowserOpen(ev state.Event) tea.Cmd {
	switch ev.Kind {
	case state.EvBrowserScreenshot:
		return m.applyBrowserShot(ev)
	case state.EvBrowserSnapshot:
		return m.applyBrowserSnap(ev)
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
// file per ts while identical bytes share the hash tail).
func saveBrowserShot(png []byte) (string, error) {
	dir := browserShotsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum(png)
	name := fmt.Sprintf("%d-%s.png", time.Now().UnixMilli(), hex.EncodeToString(sum[:4]))
	path := filepath.Join(dir, name)
	return path, os.WriteFile(path, png, 0o644)
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
