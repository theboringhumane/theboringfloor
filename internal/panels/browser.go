// browser.go — BROWSER tab: a real in-TUI page viewer. Web pages render
// as navigable text+link rows inside the sidebar — no external binary, no
// zenbu, works on every terminal.
//
// FETCH POSTURE (FetchPage): what the pane will load is deliberately
// narrow and NEVER silently on the open internet:
//
//	file:// URLs and bare paths (testdata/*.html fixtures, any on-disk
//	file) read straight off disk; http(s):// fetches are limited to a
//	localhost whitelist (localhost / 127.0.0.1 / ::1) — outbound internet
//	is OFF by default and only unlocks when the member exports
//	THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 (read AT USE TIME, no config
//	schema, no brain.json key). Every fetch is context-bounded (10s),
//	byte-capped (4 MiB), and the payload is content-SNIFFED: HTML only —
//	images/pdf/etc land a dim "unsupported content type" row instead of a
//	parse. Non-2xx surfaces as a dim error row: "404: no route → go to
//	<base url>".
//
// PANE CONTRACT (the house Tab + Interactive twin of gitpanel.go):
//
//	bar     — "▸ <url> · <title>" (one row, ansi-truncated, never styled
//	          past the width budget).
//	body    — a viewport over the rendered rows: title line, bold
//	          headings, wrapped paragraphs, bullet rows, "a │ b" table
//	          rows, code rows, "🖼 <alt>" image chips (image BYTES are
//	          never fetched), and link rows carrying markdown-style [n]
//	          indexes with the URLs kept in the page's side map (stable
//	          order, deduped by exact URL).
//
// Keys (the app routes every unclaimed key here while the tab is active):
//
//	pgup/pgdn + wheel — scroll the body.
//	↑/↓ (j/k)         — move the LINK cursor: link rows go dim→bright as
//	                    the cursor lands (auto-scrolled into view).
//	o                 — the focused link: a local file rides links.go's
//	                    OpenInBrowser (the exec seam); http(s) navigates
//	                    in place (re-fetch + history push).
//	[ / ]             — back/forward inside a bounded history ring (100
//	                    pages, scroll offsets restored, wraps at edges).
//	r                 — reload the current page in place (no dup history).
//	q / esc           — leave the tab (the app's q-quit claim yields on
//	                    this tab; the leave rides BrowserLeaveMsg).
//
// Idle (no page loaded) shows the starter card:
// "▸ enter a url · /open <url> · o for file".
//
// The async halves (fetch, exec) ride tea.Cmds and land back as
// BrowserPageMsg / BrowserOpenedMsg — the app forwards BOTH straight to
// this panel (never through the active-tab hop, so a mid-flight tab
// switch can't misdeliver a page).
package panels

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// browserFetchDeadline bounds every page fetch (spec: 10s).
const browserFetchDeadline = 10 * time.Second

// browserMaxBytes caps one page's payload (spec: 4 MiB).
const browserMaxBytes = 4 << 20

// browserHistMax bounds the back/forward ring (spec: last 100 pages).
const browserHistMax = 100

// browserAllowHTTPEnv — the ONLY config surface: set "1" to allow non-
// localhost http(s) fetches (read at use time; no schema, no brain.json).
const browserAllowHTTPEnv = "THEBORINGOFFICE_BROWSER_ALLOW_HTTP"

// browserStarterCard — the idle body (frozen copy, uishot-pinned).
const browserStarterCard = "▸ enter a url · /open <url> · o for file"

// --- async messages (the app forwards ALL THREE straight to the panel) --

// BrowserPageMsg delivers one fetch's verdict (Open / Reload / link-nav).
type BrowserPageMsg struct {
	URL  string
	Page *Page
	Err  error
}

// BrowserOpenedMsg delivers the OS-open verdict for a focused LOCAL link
// (the note row under the bar carries the human-readable outcome).
type BrowserOpenedMsg struct {
	Target LinkTarget
	Err    error
}

// BrowserLeaveMsg asks the app to leave the browser tab (q/esc — the app
// jumps back to the chat tab, thread-focus's esc contract).
type BrowserLeaveMsg struct{}

// histEntry — one stop in the navigation ring: the page itself (kept
// parsed — back/forward never re-fetches) plus the scroll offset to
// restore.
type histEntry struct {
	url  string
	page *Page
	yoff int
}

// Browser is the browser sidebar tab panel.
type Browser struct {
	vp   viewport.Model
	w, h int

	url     string // current (or attempted) location; "" = idle
	page    *Page  // nil while idle or after a failed load
	err     string // last load error, rendered as dim rows
	loading bool   // a fetch is in flight
	note    string // transient dim verdict row (OS-open outcome)
	reload  bool   // the in-flight fetch is an r-reload (replace, don't push)

	cursor    int   // focused link index into page.Links (-1 = none)
	rowOfLink []int // rendered-row index of each link's first row (-1 = none)

	hist    []histEntry // oldest first, bounded at browserHistMax
	histIdx int         // position of the CURRENT page (-1 = nothing yet)

	// test seams: default to the real fetch + the links.go exec seam.
	fetchFn func(string) (*Page, error)
	openFn  func(LinkTarget) error

	lastOuts []rowOut // the renderRows output behind the current body rows
	rev      string   // content cache key (width/url/cursor/note/err/theme)
}

// NewBrowser builds the browser panel (idle — the starter card shows).
func NewBrowser() *Browser {
	vp := viewport.New(viewport.WithWidth(10), viewport.WithHeight(5))
	vp.MouseWheelEnabled = true
	b := &Browser{vp: vp, histIdx: -1, cursor: -1}
	b.fetchFn = FetchPage
	b.openFn = OpenInBrowser
	return b
}

// Title implements Tab.
func (b *Browser) Title() string { return "browser" }

// SetSize implements Tab; the body re-renders at the new width (reflow).
func (b *Browser) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	b.w, b.h = w, h
	b.vp.SetWidth(w)
	bodyH := h - 1 // the location bar owns row 0
	if bodyH < 1 {
		bodyH = 1
	}
	b.vp.SetHeight(bodyH)
	b.refreshBody()
}

// SetState implements Tab. Office state never feeds the pane; the tick IS
// the cheap invalidation clock for theme swaps (chrome vars re-point, the
// rev key carries the theme name so a /theme re-renders once).
func (b *Browser) SetState(_ state.OfficeState) { b.refreshBody() }

// Close clears the navigation history + the loaded page (pane teardown —
// the app calls it from its quit paths; memory-only state regardless).
func (b *Browser) Close() {
	b.hist = nil
	b.histIdx = -1
	b.page = nil
	b.url = ""
	b.cursor = -1
	b.rowOfLink = nil
	b.note = ""
	b.err = ""
	b.rev = ""
	b.vp.SetContent("")
}

// wrapW — the text-wrap budget: the content width minus one cell of pad
// (the other panels' mdWidth-minus discipline), floored at 10.
func (b *Browser) wrapW() int {
	w := b.w - 1
	if w < 10 {
		w = 10
	}
	return w
}

// ---------------------------------------------------------------------------
// commands (fetch + exec — the ONLY async halves, both tea.Cmds)
// ---------------------------------------------------------------------------

// Open loads url into the pane: the bar flips optimistic, the fetch rides
// a tea.Cmd (the UI goroutine never blocks), and the verdict lands back as
// BrowserPageMsg. Test seam: fetchFn.
func (b *Browser) Open(rawurl string) tea.Cmd {
	rawurl = strings.TrimSpace(rawurl)
	if rawurl == "" {
		return nil
	}
	b.loading = true
	b.err = ""
	b.note = ""
	fn := b.fetchFn
	return func() tea.Msg {
		p, err := fn(rawurl)
		return BrowserPageMsg{URL: rawurl, Page: p, Err: err}
	}
}

// Reload re-fetches the current page in place: the history ring keeps ONE
// entry for the location (the in-flight fetch replaces the entry instead
// of pushing a twin). Deterministic — the same bytes re-parse.
func (b *Browser) Reload() tea.Cmd {
	if b.url == "" {
		return nil
	}
	b.reload = true
	return b.Open(b.url)
}

// ---------------------------------------------------------------------------
// message handling
// ---------------------------------------------------------------------------

// Update implements Interactive: scroll, link cursor, open, history,
// reload, leave — plus the two async verdicts the app forwards.
func (b *Browser) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case BrowserPageMsg:
		b.applyPage(msg)
		return nil
	case BrowserOpenedMsg:
		if msg.Err != nil {
			b.note = "could not open: " + msg.Err.Error()
		} else {
			b.note = "→ opened: " + msg.Target.Name
		}
		b.refreshBody()
		return nil
	case tea.KeyPressMsg:
		return b.handleKey(msg)
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		b.vp, cmd = b.vp.Update(msg)
		return cmd
	}
	return nil
}

// handleKey — the pane's key surface (see the package header).
func (b *Browser) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		b.moveCursor(-1)
	case "down", "j":
		b.moveCursor(1)
	case "pgup", "pgdn", "pgdown":
		var cmd tea.Cmd
		b.vp, cmd = b.vp.Update(msg)
		return cmd
	case "r":
		return b.Reload()
	case "[":
		b.historyMove(-1)
	case "]":
		b.historyMove(1)
	case "o":
		return b.openFocused()
	case "q", "esc":
		return func() tea.Msg { return BrowserLeaveMsg{} }
	}
	return nil
}

// applyPage lands one fetch verdict: errors keep the bar on the attempted
// URL with dim body rows; successes install the page, push (or replace, on
// reload) the history entry, focus the first link, and reset the scroll.
func (b *Browser) applyPage(msg BrowserPageMsg) {
	b.loading = false
	b.url = msg.URL
	if msg.Err != nil {
		b.err = msg.Err.Error()
		b.page = nil
		b.cursor = -1
		b.rowOfLink = nil
		b.reload = false
		b.vp.SetYOffset(0)
		b.refreshBody()
		return
	}
	b.err = ""
	// stash the OUTGOING page's scroll offset into its ring entry — the
	// same swap discipline historyMove applies, so `[` restores it.
	if b.histIdx >= 0 && b.histIdx < len(b.hist) {
		b.hist[b.histIdx].yoff = b.vp.YOffset()
	}
	b.page = msg.Page
	if b.reload {
		// r-reload: replace the current entry (never a twin), keep the
		// ring position.
		b.reload = false
		if b.histIdx >= 0 && b.histIdx < len(b.hist) {
			b.hist[b.histIdx] = histEntry{url: msg.URL, page: msg.Page}
		} else {
			b.pushHistory(msg.URL, msg.Page)
		}
	} else {
		b.pushHistory(msg.URL, msg.Page)
	}
	b.cursor = -1
	if msg.Page != nil && len(msg.Page.Links) > 0 {
		b.cursor = 0
	}
	b.vp.SetYOffset(0)
	b.refreshBody()
}

// pushHistory appends one navigation, dropping any forward entries past
// the current position and capping the ring at browserHistMax.
func (b *Browser) pushHistory(rawurl string, p *Page) {
	if b.histIdx >= 0 && b.histIdx < len(b.hist)-1 {
		b.hist = b.hist[:b.histIdx+1]
	}
	b.hist = append(b.hist, histEntry{url: rawurl, page: p})
	if len(b.hist) > browserHistMax {
		b.hist = b.hist[len(b.hist)-browserHistMax:]
	}
	b.histIdx = len(b.hist) - 1
}

// historyMove steps the ring (WRAPS at the edges), stashing the current
// scroll offset into the outgoing entry and restoring the incoming one's.
func (b *Browser) historyMove(d int) {
	n := len(b.hist)
	if n == 0 || b.histIdx < 0 {
		return
	}
	b.hist[b.histIdx].yoff = b.vp.YOffset()
	b.histIdx = (b.histIdx + d + n) % n
	e := b.hist[b.histIdx]
	b.url = e.url
	b.page = e.page
	b.err = ""
	b.note = ""
	b.cursor = -1
	if e.page != nil && len(e.page.Links) > 0 {
		b.cursor = 0
	}
	b.refreshBody()
	yoff := e.yoff
	if yoff < 0 {
		yoff = 0
	}
	b.vp.SetYOffset(yoff)
}

// moveCursor walks the link focus; the landing row auto-scrolls into view.
func (b *Browser) moveCursor(d int) {
	if b.page == nil || len(b.page.Links) == 0 {
		return
	}
	n := len(b.page.Links)
	b.cursor += d
	if b.cursor < 0 {
		b.cursor = 0
	}
	if b.cursor >= n {
		b.cursor = n - 1
	}
	b.refreshBody()
	b.ensureCursorVisible()
}

// ensureCursorVisible scrolls the focused link's first row into the
// viewport window (only when it sits outside).
func (b *Browser) ensureCursorVisible() {
	if b.cursor < 0 || b.cursor >= len(b.rowOfLink) {
		return
	}
	row := b.rowOfLink[b.cursor]
	if row < 0 {
		return
	}
	if row < b.vp.YOffset() {
		b.vp.SetYOffset(row)
		return
	}
	if row >= b.vp.YOffset()+b.vp.Height() {
		b.vp.SetYOffset(row - b.vp.Height() + 1)
	}
}

// openFocused — `o` on the cursor's link: a LOCAL file rides the links.go
// exec seam (async verdict → BrowserOpenedMsg); anything remote (http(s))
// navigates the pane in place (a normal Open: re-fetch + history push).
func (b *Browser) openFocused() tea.Cmd {
	if b.page == nil || b.cursor < 0 || b.cursor >= len(b.page.Links) {
		return nil
	}
	l := b.page.Links[b.cursor]
	if isLocalBrowserURL(l.URL) {
		path := localBrowserPath(l.URL)
		t := LinkTarget{Kind: LinkFile, Value: path, Name: filepath.Base(path)}
		fn := b.openFn
		return func() tea.Msg { return BrowserOpenedMsg{Target: t, Err: fn(t)} }
	}
	return b.Open(l.URL)
}

// ---------------------------------------------------------------------------
// rendering
// ---------------------------------------------------------------------------

// View implements Tab: the location bar over the body viewport, fitted to
// the panel's height.
func (b *Browser) View() string {
	return fit(b.bar()+"\n"+b.vp.View(), b.h)
}

// bar — "▸ <url> · <title>", ansi-aware truncated to the panel width.
func (b *Browser) bar() string {
	loc := b.url
	if loc == "" {
		loc = "browser"
	}
	s := chrome.AccentText.Render("▸ ") + loc
	if b.page != nil && b.page.Title != "" {
		s += chrome.DimText.Render(" · " + b.page.Title)
	}
	return ansi.Truncate(s, b.w, "")
}

// refreshBody re-renders the viewport content when any render input moved
// (rev-key discipline, mail.go's pattern — a hit skips the whole pass).
func (b *Browser) refreshBody() {
	key := fmt.Sprintf("%d|%s|%d|%s|%s|%t|%s",
		b.w, b.url, b.cursor, b.note, b.err, b.loading, chrome.CurrentTheme().Name)
	if key == b.rev {
		return
	}
	b.rev = key
	rows, rowOfLink := b.bodyRows()
	b.rowOfLink = rowOfLink
	b.vp.SetContent(strings.Join(rows, "\n"))
}

// bodyRows composes the styled body: optional verdict note, then the
// load-state rows (loading / error / starter card / the page itself).
// Returns the rows plus the link→first-row index map (prefix rows shift
// the page's own indexes).
func (b *Browser) bodyRows() ([]string, []int) {
	var rows []string
	if b.note != "" {
		rows = append(rows, chrome.DimText.Render(b.note))
	}
	prefix := len(rows)
	var pageOuts []rowOut
	switch {
	case b.page == nil && b.loading:
		rows = append(rows, chrome.DimText.Render("▸ loading…"))
	case b.err != "":
		for _, ln := range strings.Split(wrapPlain(b.err, b.wrapW()), "\n") {
			rows = append(rows, chrome.DimText.Render(ln))
		}
	case b.page == nil:
		rows = append(rows, chrome.DimText.Render(browserStarterCard))
	default:
		var pageRows []string
		pageRows, pageOuts = b.pageRows()
		rows = append(rows, pageRows...)
	}
	b.lastOuts = pageOuts
	rowOfLink := make([]int, b.linkCount())
	for i := range rowOfLink {
		rowOfLink[i] = -1
	}
	for i, ro := range pageOuts {
		for _, li := range ro.links {
			if li >= 0 && li < len(rowOfLink) && rowOfLink[li] == -1 {
				rowOfLink[li] = prefix + i
			}
		}
	}
	return rows, rowOfLink
}

// --- the page's own rows -------------------------------------------------

// pageRows renders the loaded page at the current wrap width, styling by
// row kind: title/headings bold (chrome.Header), link rows dim with the
// FOCUSED one bright (chrome.AccentText), image chips dim, the rest plain.
func (b *Browser) pageRows() ([]string, []rowOut) {
	outs := b.page.renderRows(b.wrapW())
	rows := make([]string, 0, len(outs))
	for _, ro := range outs {
		switch {
		case len(ro.links) > 0:
			if intIn(ro.links, b.cursor) {
				rows = append(rows, chrome.AccentText.Render(ro.text))
			} else {
				rows = append(rows, chrome.DimText.Render(ro.text))
			}
		case ro.kind == blkTitle || ro.kind == blkHeading:
			rows = append(rows, chrome.Header.Render(ro.text))
		case ro.kind == blkImage || ro.kind == blkDim:
			rows = append(rows, chrome.DimText.Render(ro.text))
		default:
			rows = append(rows, ro.text)
		}
	}
	return rows, outs
}

// linkCount — the loaded page's indexed-link count (0 while idle/errored).
func (b *Browser) linkCount() int {
	if b.page == nil {
		return 0
	}
	return len(b.page.Links)
}

// intIn reports whether n rides the set.
func intIn(set []int, n int) bool {
	for _, v := range set {
		if v == n {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// FetchPage — the loader (spec §2)
// ---------------------------------------------------------------------------

// FetchPage loads one URL into a parsed Page. Supported sources:
//
//   - file:// URLs and bare paths (relative paths resolve against the
//     process cwd — testdata/*.html fixtures included);
//   - http(s):// against the localhost whitelist (localhost, 127.0.0.1,
//     ::1) — every other host is REFUSED unless
//     THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 is exported (never network
//     silently).
//
// The fetch is context-bounded (10s) and byte-capped (4 MiB); the payload
// is content-SNIFFED and only text/html parses — anything else returns a
// Page whose Unsupported field carries the sniffed type (the pane renders
// it as one dim row). Non-2xx http answers error as
// "<status>: no route → go to <base url>".
func FetchPage(rawurl string) (*Page, error) {
	rawurl = strings.TrimSpace(rawurl)
	if rawurl == "" {
		return nil, errors.New("empty url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), browserFetchDeadline)
	defer cancel()

	body, finalURL, err := fetchBrowserBytes(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	sniff := http.DetectContentType(body)
	if !strings.HasPrefix(sniff, "text/html") {
		return &Page{URL: finalURL, Unsupported: sniff}, nil
	}
	p, err := parseHTMLPage(strings.NewReader(string(body)), finalURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", finalURL, err)
	}
	return p, nil
}

// fetchBrowserBytes reads one URL's payload (bounded + capped) and returns
// the bytes plus the FINAL url (bare paths normalized to file:// form).
func fetchBrowserBytes(ctx context.Context, rawurl string) ([]byte, string, error) {
	if isLocalBrowserURL(rawurl) {
		path := localBrowserPath(rawurl)
		f, err := os.Open(path)
		if err != nil {
			return nil, rawurl, fmt.Errorf("open %s: %w", path, err)
		}
		defer f.Close()
		body, err := io.ReadAll(io.LimitReader(f, browserMaxBytes))
		if err != nil {
			return nil, rawurl, fmt.Errorf("read %s: %w", path, err)
		}
		return body, "file://" + path, nil
	}

	u, err := url.Parse(rawurl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, rawurl, fmt.Errorf("unsupported url %q (file://, a bare path, or http(s)://localhost only)", rawurl)
	}
	if !browserHostAllowed(u.Hostname()) {
		return nil, rawurl, fmt.Errorf("http fetch blocked: %s is not localhost (export %s=1 to allow outbound pages)", u.Hostname(), browserAllowHTTPEnv)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, rawurl, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, rawurl, fmt.Errorf("fetch %s: %w", rawurl, err)
	}
	defer resp.Body.Close()
	base := u.Scheme + "://" + u.Host
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, rawurl, fmt.Errorf("%d: no route → go to %s", resp.StatusCode, base)
	}
	body, err := io.ReadAll(http.MaxBytesReader(nil, resp.Body, browserMaxBytes))
	if err != nil {
		return nil, rawurl, fmt.Errorf("read %s: %w", rawurl, err)
	}
	return body, rawurl, nil
}

// isLocalBrowserURL — a file:// URL or a bare path (no scheme at all).
func isLocalBrowserURL(rawurl string) bool {
	if strings.HasPrefix(rawurl, "file://") {
		return true
	}
	return !strings.Contains(rawurl, "://")
}

// localBrowserPath normalizes a local target to a filesystem path: the
// file:// prefix strips, bare paths absolutize against the cwd.
func localBrowserPath(rawurl string) string {
	p := strings.TrimPrefix(rawurl, "file://")
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return abs
}

// browserHostAllowed — the fetch gate: localhost always, anything else
// only with THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 (read AT USE TIME).
func browserHostAllowed(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return os.Getenv(browserAllowHTTPEnv) == "1"
}
