// browser.go — BROWSER tab: a real in-TUI page viewer. Web pages render
// as navigable text+link rows inside the sidebar — no external binary, no
// zenbu, works on every terminal.
//
// SHOT MODE (the headless screenshot lane — the section below FetchPage's
// seam): every landed fetch arms ONE headless render of the same URL at
// the pane box's pixel dims (15s bounded, tea.Cmd); on a kitty-capable
// host a success swaps the pane chrome for the " shot " badge + the
// "▸ headless chromium · <url>" strip and the PNG paints the body
// through the wave-81 frame-splice wrapper (f=100, NO c=/r=), while the
// text model rides underneath (history, [n] nav, reload stay on it).
// Everywhere else the shot still saves to shots/<ts>-<hash8>.png and the
// text lane carries the dim "screenshot: <path>" row; a failure (chrome
// absent / refused / timeout) stays text with ONE dim classified reason.
//
// PREMIUM LANE (browser_lane.go's controller, CONSULTED ON EVERY OPEN):
// on a kitty-capable host (kitty/ghostty) with the `terminal-browser`
// binary on PATH and no kill-switch armed, Open ALSO spawns the embedded
// zenbu child — the pane's View swaps the text body for the live PTY
// screen model (the " zenbu " badge + the "▸ zenbu terminal-browser ·
// <url>" strip mark the lane), unclaimed keys forward to the child
// through the controller's Write path, and the text fetch STILL rides
// underneath so an early/non-zero exit lands the fallback on a warm page
// (the dim "zenbu exited (<code>) — falling back to text mode" note
// surfaces through the pane's own note row). Everywhere else the lane
// resolves text and this pane is byte-identical to the universal viewer.
// The app owns the lifecycle flips: SuspendLane on the ctrl+b/q/esc
// switch-away, ResumeLane on return, CloseLane via Close at quit.
//
// FETCH POSTURE (FetchPage): what the pane will load is deliberately
// narrow and NEVER silently on the open internet:
//
//	file:// URLs and bare paths (testdata/*.html fixtures, any on-disk
//	file) read straight off disk; http(s):// fetches allow the localhost
//	whitelist (localhost / 127.0.0.1 / ::1) on either scheme and https to
//	ANY host by default — plain http to a non-localhost host is refused
//	unless the member exports THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 (read
//	AT USE TIME, no config schema, no brain.json key). Every fetch is context-bounded (10s),
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
// THE TEXT LANE EXPLAINS ITSELF: when the lane resolve missed premium
// (the memoized verdict's reason class — binary missing, terminal
// unsupported, or a kill-switch armed), ONE dim hint row rides under the
// location bar — on the starter card AND after every text-lane open — so
// the member sees WHY (and, for the missing binary, where to get the full
// renderer) at the moment of disappointment. Premium hosts paint no hint
// row anywhere.
//
// The async halves (fetch, exec) ride tea.Cmds and land back as
// BrowserPageMsg / BrowserOpenedMsg — the app forwards BOTH straight to
// this panel (never through the active-tab hop, so a mid-flight tab
// switch can't misdeliver a page).
package panels

import (
	"context"
	"encoding/base64"
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
	"github.com/theboringhumane/theboringoffice/internal/headless"
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
// A Shot-carrying msg is the headless screenshot's async verdict riding
// the SAME app→pane forwarding hop (Shot != nil ⇒ URL/Page/Err unset).
type BrowserPageMsg struct {
	URL  string
	Page *Page
	Err  error
	Shot *BrowserShot
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

// BrowserShot — ONE headless-screenshot verdict, riding BrowserPageMsg's
// Shot field (the app's existing forwarding hop needs no new routing
// entry — model.go stays untouched). The Seq is the open's generation
// (a superseded open's render drops silently); resizeTick marks the
// resize debounce's re-shot timer (panels-internal).
type BrowserShot struct {
	Seq  int              // the open generation this verdict belongs to
	Res  *headless.Result // the engine's render (nil on failure)
	Path string           // the saved PNG ("" when the save failed)
	Err  error            // the classified failure (nil on success)

	resizeTick bool      // the debounce tick, not a render verdict
	armedAt    time.Time // the resize stamp the tick armed at
}

// histEntry — one stop in the navigation ring: the page itself (kept
// parsed — back/forward never re-fetches) plus the scroll offset to
// restore.
type histEntry struct {
	url  string
	page *Page
	yoff int
}

// browserShot — the displayed/saved headless screenshot: the redirect-
// final url + title, the PNG bytes (kept for the floor-flip re-publish —
// a suspend/resume NEVER re-renders), the content-addressed office image
// id, the cached verbatim f=100 APC (NO c=/r= — the wave-81 emission
// ruling), and the on-disk path for the member's `o`-to-open habit.
type browserShot struct {
	url      string
	title    string
	png      []byte
	officeID uint32
	frame    string
	path     string
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

	// lane — the premium render lane's state machine (browser_lane.go):
	// the memoized resolve, the embedded zenbu child, the fallback latch.
	// Consulted on EVERY open; on a text-lane host it is the cheap URL
	// recorder and the pane never paints its chrome.
	lane *BrowserLaneController

	// the headless screenshot lane (SHOT MODE — see the section below):
	// every landed fetch arms one render at the pane box's pixel dims; a
	// success on a kitty-capable host swaps the pane chrome for the shot
	// region and the PNG paints through the frame-splice wrapper. ALL of
	// this is UI-goroutine state (the render itself rides a tea.Cmd).
	shotSeq         int    // the open generation (stale drops; Close bumps)
	shotLoading     bool   // a render is in flight (the "rendering" row)
	shotLoadingURL  string // the url the in-flight render targets
	shot            *browserShot
	shotErr         string    // the classified failure's dim row
	shotCapable     bool      // kitty display lane, pinned at pane creation
	shotResizeAt    time.Time // the last resize's stamp (the debounce input)
	shotResizeDue   bool      // a resize awaits its 300ms quiet window
	shotResizeArmed bool      // a debounce tick is already in flight

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
	b.lane = NewBrowserLaneController(10, 5)
	b.fetchFn = FetchPage
	b.openFn = OpenInBrowser
	// the shot display lane pins ONCE at pane creation (the lane memo
	// idiom: one honest env read per pane, never per frame).
	b.shotCapable = DetectImageSupport() == KittyLane
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
	if b.laneHint() != "" {
		bodyH-- // the text lane's dim hint row owns the row under the bar
	}
	if bodyH < 1 {
		bodyH = 1
	}
	b.vp.SetHeight(bodyH)
	// the live premium child takes the SIGWINCH (the strip + note rows
	// stay reserved inside the controller's own math).
	if b.lane != nil {
		b.lane.SetSize(w, h)
	}
	// a live/in-flight screenshot's pixel dims are now stale: stamp the
	// debounce (the NEXT routed msg arms the 300ms quiet-window tick —
	// SetSize itself can never emit a cmd).
	if b.shot != nil || b.shotLoading {
		b.shotResizeAt = shotNow()
		b.shotResizeDue = true
	}
	b.refreshBody()
}

// SetState implements Tab. Office state never feeds the pane; the tick IS
// the cheap invalidation clock for theme swaps (chrome vars re-point, the
// rev key carries the theme name so a /theme re-renders once).
func (b *Browser) SetState(_ state.OfficeState) { b.refreshBody() }

// Close clears the navigation history + the loaded page (pane teardown —
// the app calls it from its quit paths; memory-only state regardless) and
// seals the lane controller (the premium child is group-killed + reaped —
// NEVER leaked past the office exit). The shot state drops with the pane:
// the generation bump drops every in-flight render's verdict, and the
// registry clears on the spot — the QUIT path renders no further Frame
// to publish the empty state, so the renderer's final flush (the
// alt-screen exit) must already find nothing to re-emit (the zenbu
// session Close's exact contract).
func (b *Browser) Close() {
	if b.lane != nil {
		b.lane.Close()
	}
	b.shotSeq++ // invalidate every in-flight render's verdict
	b.shot = nil
	b.shotLoading = false
	b.shotErr = ""
	b.shotResizeDue, b.shotResizeArmed = false, false
	ZenbuRegistry().Clear()
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
// the headless screenshot lane (SHOT MODE)
// ---------------------------------------------------------------------------
//
// Every landed fetch (Open / Reload / link-nav / ring hop) arms ONE
// headless render of the same URL at the pane body box's PIXEL dims
// (widthPx = paneBodyCols*9, heightPx = paneBodyRows*18 — the default
// cell metric; THEBORINGOFFICE_CELL_PX=W:H overrides). The render rides a
// tea.Cmd (15s bounded) and lands back as BrowserPageMsg{Shot:…} through
// the app's existing forwarding hop. On a kitty-capable host
// (DetectImageSupportFrom == KittyLane — the zenbu lane's own gate,
// pinned at pane creation) a success enters SHOT MODE: the pane chrome
// swaps to the " shot " badge + the "▸ headless chromium · <url>" strip
// and the PNG paints the body through the wave-81 frame-splice (the
// per-Frame ZenbuFrameRegistry publish + the tea.WithOutput wrapper —
// f=100, NO c=/r= keys, the wave-81/82 production emission ruling: the
// c=/r= variant did not visibly paint on the member's ghostty, the bare
// a=T+t=d+f=100+i=<id>+q=2 did). Everywhere else a success still SAVES
// the PNG (<$THEBORINGOFFICE_HOME or os.TempDir>/shots/<ts>-<hash8>.png,
// the agent-tool flow's naming convention) and the text lane carries one
// dim "screenshot: <path>" row (kitty prints the path too — the member's
// `o`-to-open habit). A failure (chrome absent / navigation refused /
// timeout) stays text with ONE dim classified reason row.
//
// The text model ALWAYS rides underneath in parallel (the fetch is the
// same open): the history ring records the open, [n] link navigation
// stays on the text lane's model, and a shot-mode leave/failure lands
// back on a warm page.
//
// KEEP-ALIVE: a floor flip needs NOTHING from the pane — the app's
// publish seam (publishZenbuFrame) clears the registry when the slot
// hides (the wrapper's diff flushes one a=d) and re-publishes the CACHED
// bytes on return — no re-render (a resize's debounce re-render is the
// only re-render trigger, and it rides the Update cadence's tea.Tick).

// Timing + metric discipline (vars, not consts — the house deadline-test
// idiom: suites shrink the debounce, never the reverse).
var (
	// browserShotDeadline bounds ONE headless render (spec: 15s; the
	// engine imposes its own 15s budget on top of this caller context).
	browserShotDeadline = 15 * time.Second
	// browserShotResizeDebounce — the resize re-render's quiet window
	// (spec: 300ms; a burst of resizes converges to ONE re-render).
	browserShotResizeDebounce = 300 * time.Millisecond
	// shotNow — the clock seam (the saved filename's <ts> + the debounce
	// stamps; the harness pins it for byte-identical paths).
	shotNow = time.Now
)

// The headless engine seam: prod wires the REAL internal/headless
// package (the wave-85 contract); suites/shots swap ONE point each
// (SetHeadlessForShot) — no live chrome in unit tests.
var (
	headlessAvailable  = headless.Available
	headlessScreenshot = headless.Screenshot
)

// SetHeadlessForShot swaps the headless engine seam for a shot/test
// harness (SetZenbuEmitForShot's exact precedent) and returns the
// restore. A nil shot is safe while avail reports false (runShot probes
// first).
func SetHeadlessForShot(avail func() (string, bool), shot func(context.Context, string, int, int) (*headless.Result, error)) (restore func()) {
	oldA, oldS := headlessAvailable, headlessScreenshot
	if avail != nil {
		headlessAvailable = avail
	}
	if shot != nil {
		headlessScreenshot = shot
	}
	return func() { headlessAvailable, headlessScreenshot = oldA, oldS }
}

// SetShotNowForShot pins the shot clock (SetHeadlessForShot's twin) and
// returns the restore.
func SetShotNowForShot(fn func() time.Time) (restore func()) {
	old := shotNow
	shotNow = fn
	return func() { shotNow = old }
}

// shotFailChromeCopy — the chrome-missing class's EXACT dim row (frozen;
// names the fix).
const shotFailChromeCopy = "text lane — headless chrome not found · install Chrome or export THEBORINGOFFICE_CHROME"

// shotFailCopy — the failure classifier: ONE dim reason row per class
// (chrome-missing names the fix; a policy refusal carries the decision's
// verbatim reason; the timeout row names the bound).
func shotFailCopy(err error) string {
	var pol *headless.PolicyError
	switch {
	case errors.Is(err, headless.ErrChromeNotFound):
		return shotFailChromeCopy
	case errors.As(err, &pol):
		if pol.Reason != "" {
			return "text lane — headless render refused: " + pol.Reason
		}
		return "text lane — headless render refused: " + err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("text lane — headless render timed out (%ds)", int(browserShotDeadline.Seconds()))
	default:
		return "text lane — headless render failed: " + err.Error()
	}
}

// browserShotCellPx — the cell metric for the pixel-dims math:
// THEBORINGOFFICE_CELL_PX=W:H read AT USE TIME, else the 9x18 default.
func browserShotCellPx() (cellW, cellH int) {
	raw := strings.TrimSpace(os.Getenv("THEBORINGOFFICE_CELL_PX"))
	if raw != "" {
		var w, h int
		if n, err := fmt.Sscanf(raw, "%d:%d", &w, &h); err == nil && n == 2 && w > 0 && h > 0 {
			return w, h
		}
	}
	return 9, 18
}

// shotBodyRows — the shot region's body height (the strip + note rows
// stay reserved, the lane controller's bodyH discipline).
func (b *Browser) shotBodyRows() int {
	rows := b.h - 2
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ShotBoxPx — the pane body box in CSS pixels for the engine: cols*cellW
// × bodyRows*cellH (the render's EXACT viewport; the engine captures at
// deviceScaleFactor 2 for retina). The harness asserts the recorded
// engine dims against this.
func (b *Browser) ShotBoxPx() (widthPx, heightPx int) {
	cw, ch := browserShotCellPx()
	cols := b.w
	if cols < 1 {
		cols = 1
	}
	return cols * cw, b.shotBodyRows() * ch
}

// shotKittyFrame — the cached verbatim transmit+display APC for one
// screenshot: a=T + t=d + f=100 + i=<content hash8> + q=2 (+C=1, the
// splice's cursor-stay contract) and NO c=/r= keys — the wave-81/82
// production emission ruling (the c=/r= variant did NOT visibly paint on
// the member's ghostty; this exact key set did). The payload rides the
// PNG's base64 verbatim; the wrapper anchors it at pane-local (0,0).
func shotKittyFrame(officeID uint32, png []byte) string {
	return "\x1b_Ga=T,t=d,q=2,C=1,i=" + KittyIDHash8(officeID) + ",f=100;" +
		base64.StdEncoding.EncodeToString(png) + "\x1b\\"
}

// saveShotPNG — every successful shot lands on disk (the member's
// `o`-to-open habit): <THEBORINGOFFICE_HOME or os.TempDir>/shots/
// <unixMillis>-<hash8>.png (the agent-tool flow's naming convention —
// hash8 = sha1(png)[:4] hex, KittyIDHash8's exact shape). A save failure
// never kills the shot mode ("" path back).
func saveShotPNG(png []byte) (string, error) {
	base := strings.TrimSpace(os.Getenv("THEBORINGOFFICE_HOME"))
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "shots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d-%s.png", shotNow().UnixMilli(), KittyIDHash8(KittyImageID(png)))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// runShot — the render's synchronous core inside the tea.Cmd: the probe
// gates chrome-missing before the engine runs, the render is 15s
// bounded, a success saves the PNG before the verdict lands. Never
// touches pane state (safe on the cmd's goroutine).
func (b *Browser) runShot(rawurl string, seq, widthPx, heightPx int) BrowserPageMsg {
	if _, ok := headlessAvailable(); !ok {
		return BrowserPageMsg{Shot: &BrowserShot{Seq: seq, Err: headless.ErrChromeNotFound}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), browserShotDeadline)
	defer cancel()
	res, err := headlessScreenshot(ctx, rawurl, widthPx, heightPx)
	if err != nil {
		// normalize the timeout class: OUR budget firing wins over the
		// engine's error wording (errors.Is covers the engine's own wrap).
		if ctx.Err() == context.DeadlineExceeded {
			err = context.DeadlineExceeded
		}
		return BrowserPageMsg{Shot: &BrowserShot{Seq: seq, Err: err}}
	}
	if res == nil || len(res.PNG) == 0 {
		return BrowserPageMsg{Shot: &BrowserShot{Seq: seq, Err: errors.New("headless: empty screenshot")}}
	}
	path, _ := saveShotPNG(res.PNG) // a save failure never kills the shot
	return BrowserPageMsg{Shot: &BrowserShot{Seq: seq, Res: res, Path: path}}
}

// armShot — the screenshot's kickoff (every landed fetch + every ring
// hop): the pane shows the dim "rendering <url>…" row until the verdict
// lands. The render cmd's msg shape keeps Open's returned-cmd contract
// untouched (the navigation suite's type pins): the shot chains off the
// FETCH's landing, never batches into Open's own cmd.
func (b *Browser) armShot(rawurl string) tea.Cmd {
	if strings.TrimSpace(rawurl) == "" {
		return nil
	}
	seq := b.shotSeq
	widthPx, heightPx := b.ShotBoxPx()
	b.shotLoading = true
	b.shotLoadingURL = rawurl
	b.shotErr = ""
	b.refreshBody()
	return func() tea.Msg {
		return b.runShot(rawurl, seq, widthPx, heightPx)
	}
}

// applyShot — the render verdict (or the resize debounce's tick): a
// success enters SHOT MODE (the kitty display lane) or records the saved
// path (every host); a failure stays text with the classified dim row.
// Stale generations drop silently. Returns the follow-up cmd when one is
// owed (the resize re-render, a re-armed debounce).
func (b *Browser) applyShot(s *BrowserShot) tea.Cmd {
	if s.resizeTick {
		b.shotResizeArmed = false
		if s.Seq != b.shotSeq { // a superseded open's timer
			b.shotResizeDue = false
			return nil
		}
		if !s.armedAt.Equal(b.shotResizeAt) {
			return b.shotResizeTickCmd() // a newer resize landed — wait out the burst
		}
		if b.shotLoading {
			return b.shotResizeTickCmd() // the open's render is still flying — re-arm
		}
		b.shotResizeDue = false
		if b.shot == nil { // nothing displayed to re-render
			return nil
		}
		// the debounce passed: re-render the CURRENT url at the CURRENT
		// box (the PNG must match the pane).
		b.shotLoading = true
		b.shotLoadingURL = b.url
		seq, rawurl := b.shotSeq, b.url
		widthPx, heightPx := b.ShotBoxPx()
		b.refreshBody()
		return func() tea.Msg { return b.runShot(rawurl, seq, widthPx, heightPx) }
	}
	if s.Seq != b.shotSeq {
		return nil // a superseded open's render — drop
	}
	b.shotLoading = false
	if s.Err != nil {
		b.shot = nil
		b.shotErr = shotFailCopy(s.Err)
		b.refreshBody()
	} else {
		id := KittyImageID(s.Res.PNG)
		finalURL := s.Res.URL
		if finalURL == "" {
			finalURL = b.shotLoadingURL
		}
		b.shot = &browserShot{
			url:      finalURL,
			title:    s.Res.Title,
			png:      s.Res.PNG,
			officeID: id,
			frame:    shotKittyFrame(id, s.Res.PNG),
			path:     s.Path,
		}
		b.shotErr = ""
		b.refreshBody()
	}
	// a resize that landed mid-render re-arms the debounce on the spot.
	return b.shotResizeTickCmd()
}

// shotResizeTickCmd — the debounce's arming seam: a resize stamped due +
// no tick in flight ⇒ ONE tea.Tick whose landing re-checks the quiet
// window (a burst converges to one re-render). Arms ONLY on the Update
// cadence (the pane's cmd return is the runtime's only cmd channel —
// SetSize/View can stamp and observe, never emit).
func (b *Browser) shotResizeTickCmd() tea.Cmd {
	if !b.shotResizeDue || b.shotResizeArmed {
		return nil
	}
	if b.shot == nil && !b.shotLoading {
		b.shotResizeDue = false
		return nil
	}
	b.shotResizeArmed = true
	at, seq := b.shotResizeAt, b.shotSeq
	return tea.Tick(browserShotResizeDebounce, func(time.Time) tea.Msg {
		return BrowserPageMsg{Shot: &BrowserShot{Seq: seq, resizeTick: true, armedAt: at}}
	})
}

// ShotActive — SHOT MODE paints RIGHT NOW (a successful render + the
// kitty display lane): the pane's chrome swaps and the app's publish
// seam carries the PNG to the frame-splice wrapper.
func (b *Browser) ShotActive() bool { return b.shot != nil && b.shotCapable }

// ShotFrameState — the registry contribution (LaneFrameState's twin):
// the cached f=100 APC anchored at pane-local (0,0) under the PNG's
// content-addressed office id (content-addressed is CORRECT here —
// static shots, not a live child's reused ids). nil while no shot shows.
func (b *Browser) ShotFrameState() []ZenbuFrameImage {
	if !b.ShotActive() {
		return nil
	}
	return []ZenbuFrameImage{{OfficeID: b.shot.officeID, OX: 0, OY: 0, Frame: b.shot.frame}}
}

// ShotPath — the saved PNG's on-disk path ("" while none) — the
// harness's read for the save convention.
func (b *Browser) ShotPath() string {
	if b.shot == nil {
		return ""
	}
	return b.shot.path
}

// shotStateRow — the TEXT lane's shot status, ONE dim row under the bar:
// the in-flight render, the classified failure, or (non-kitty) the saved
// PNG's path. "" while nothing shot-related is live (and never on the
// kitty display lane's own chrome — the shot region carries its own).
func (b *Browser) shotStateRow() string {
	switch {
	case b.shotLoading:
		return "rendering " + b.shotLoadingURL + "…"
	case b.shotErr != "":
		return b.shotErr
	case b.shot != nil && !b.shotCapable && b.shot.path != "":
		return "screenshot: " + b.shot.path
	}
	return ""
}

// shotRegionView — the SHOT MODE pane region, RegionView's exact shape:
// the " shot " badge + the "▸ headless chromium · <url>" strip on row 0,
// a BLANK body (kitty z-order paints text OVER images — the screenshot
// owns the box through the frame splice), and the saved path on the dim
// note row at the bottom (the member's `o`-to-open habit).
func (b *Browser) shotRegionView() string {
	var sb strings.Builder
	sb.WriteString(chrome.TabActive.Render(" shot ") + " " + fitPlain("▸ headless chromium · "+b.shot.url, b.w-7))
	for y := 0; y < b.shotBodyRows(); y++ {
		sb.WriteString("\n" + fitPlain("", b.w))
	}
	note := ""
	if b.shot.path != "" {
		note = "screenshot: " + b.shot.path
	}
	sb.WriteString("\n" + chrome.DimText.Render(fitPlain(note, b.w)))
	return sb.String()
}

// ---------------------------------------------------------------------------
// commands (fetch + exec — the ONLY async halves, both tea.Cmds)
// ---------------------------------------------------------------------------

// Open loads url into the pane: the bar flips optimistic, the fetch rides
// a tea.Cmd (the UI goroutine never blocks), and the verdict lands back as
// BrowserPageMsg. Test seam: fetchFn.
//
// EVERY open also drives the lane controller: on a kitty-capable host
// with terminal-browser on PATH the embedded zenbu child spawns here (the
// text fetch below still rides — the fallback lands on a warm page, and
// the history ring stays meaningful in text mode); everywhere else the
// call is the text lane's cheap URL record. A lane miss is never fatal
// (a spawn failure wears the 127 note).
func (b *Browser) Open(rawurl string) tea.Cmd {
	rawurl = strings.TrimSpace(rawurl)
	if rawurl == "" {
		return nil
	}
	b.loading = true
	b.err = ""
	// leave SHOT MODE on every fresh open: the old page's PNG deletes on
	// the next frame publish (the wrapper's emitted-set diff), the text
	// lane shows through (never blank), and the fetch's landing arms the
	// new render (applyPage). The generation bump drops any in-flight
	// render for the superseded open.
	b.shotSeq++
	b.shot = nil
	b.shotErr = ""
	b.shotLoading = false
	b.shotResizeDue, b.shotResizeArmed = false, false
	if b.lane != nil {
		_ = b.lane.OpenURL(rawurl)
		b.note = b.lane.Note() // "" while premium is healthy; the 127 wording on a spawn failure
	} else {
		b.note = ""
	}
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
// reload, leave — plus the async verdicts the app forwards. Every return
// gets the resize debounce's arming sweep appended (the pane's cmd return
// is the ONLY cmd channel — SetSize stamps, Update arms).
func (b *Browser) Update(msg tea.Msg) tea.Cmd {
	b.pollLane()
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case BrowserPageMsg:
		cmd = b.applyPage(msg)
	case BrowserOpenedMsg:
		if msg.Err != nil {
			b.note = "could not open: " + msg.Err.Error()
		} else {
			b.note = "→ opened: " + msg.Target.Name
		}
		b.refreshBody()
	case tea.KeyPressMsg:
		cmd = b.handleKey(msg)
	case tea.MouseWheelMsg:
		if b.lane != nil && b.lane.PremiumActive() {
			// the embed owns its surface — no mouse bytes ever ride the
			// PTY seam (the termSess contract), and the hidden text
			// viewport must not scroll behind the child's back.
			cmd = nil
		} else {
			b.vp, cmd = b.vp.Update(msg)
		}
	}
	if tick := b.shotResizeTickCmd(); tick != nil {
		cmd = tea.Batch(cmd, tick) // compactCmds drops a nil half
	}
	return cmd
}

// handleKey — the pane's key surface (see the package header). While the
// premium embed is live the pane's own keys yield: q/esc still leave (the
// app suspends the lane with the slot flip — the child dies with the
// pane, never a leak), EVERY other key forwards to the child through
// term.go's keyToBytes matrix (the office's own claims — ctrl+b, ctrl+q,
// tab, the digit jumps — are intercepted above and never reach here).
func (b *Browser) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if b.lane != nil && b.lane.PremiumActive() {
		switch msg.String() {
		case "q", "esc":
			return func() tea.Msg { return BrowserLeaveMsg{} }
		}
		if sess := b.lane.Session(); sess != nil {
			if bs, ok := keyToBytes(msg); ok {
				_, _ = sess.Write(bs)
			}
		}
		return nil
	}
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
		return b.historyMove(-1)
	case "]":
		return b.historyMove(1)
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
// A Shot-carrying msg is the render verdict's lane (applyShot). EITHER
// way the headless render arms on the fetch's landing — the screenshot
// rides behind EVERY page state, errors included (chrome renders a 404
// exactly like the text lane prints one).
func (b *Browser) applyPage(msg BrowserPageMsg) tea.Cmd {
	if msg.Shot != nil {
		return b.applyShot(msg.Shot)
	}
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
		return b.armShot(msg.URL)
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
	return b.armShot(msg.URL)
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
// The restored page re-renders its screenshot too (the ring is the TEXT
// lane's model; the shot follows it) — the render cmd rides the return.
func (b *Browser) historyMove(d int) tea.Cmd {
	n := len(b.hist)
	if n == 0 || b.histIdx < 0 {
		return nil
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
	// the ring hop is a navigation: leave the old page's shot mode, arm
	// the restored page's render under a fresh generation.
	b.shotSeq++
	b.shot = nil
	b.shotErr = ""
	b.shotLoading = false
	b.shotResizeDue, b.shotResizeArmed = false, false
	b.refreshBody()
	yoff := e.yoff
	if yoff < 0 {
		yoff = 0
	}
	b.vp.SetYOffset(yoff)
	// the lane follows the ring: a never-fell-back url re-embeds the
	// premium child on a resolving host; a latched one stays text (the
	// controller's no-flap rule).
	if b.lane != nil {
		_ = b.lane.OpenURL(e.url)
	}
	return b.armShot(e.url)
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
// the panel's height. While the premium embed is live the pane paints the
// controller's region instead: the " zenbu " badge + the "▸ zenbu
// terminal-browser · <url>" strip on row 0, the embedded PTY's screen
// model as the body, the (blank-while-healthy) note row at the bottom.
// SHOT MODE is the next posture (a landed headless render on the kitty
// display lane): the " shot " badge + the "▸ headless chromium · <url>"
// strip, a blank body the frame-spliced PNG owns, the saved path on the
// note row. The text lane wears ONE dim hint row under the bar (the lane
// resolve's reason class — never an error style, ansi-truncated to the
// pane width).
func (b *Browser) View() string {
	b.pollLane()
	if b.lane != nil && b.lane.PremiumActive() {
		return fit(b.lane.RegionView(nil), b.h)
	}
	if b.ShotActive() {
		return fit(b.shotRegionView(), b.h)
	}
	view := b.bar()
	if hint := b.laneHint(); hint != "" {
		view += "\n" + chrome.DimText.Render(ansi.Truncate(hint, b.w, ""))
	}
	return fit(view+"\n"+b.vp.View(), b.h)
}

// laneHint — the text lane's "why", from the lane controller's
// pane-creation-memoized verdict: the frozen per-class copy ("" while the
// lane resolved premium — no hint row anywhere, and the resolve never
// re-reads env/PATH past pane creation).
func (b *Browser) laneHint() string {
	if b.lane == nil {
		return ""
	}
	_, reason, killVar := b.lane.Verdict()
	return browserLaneHintText(reason, killVar)
}

// ---------------------------------------------------------------------------
// the lane's pane surface (the app drives the lifecycle through these)
// ---------------------------------------------------------------------------

// pollLane rides the pane's per-frame (View) + per-message (Update)
// cadence — the controller's Poll contract: observes a dropped premium
// child and lands the exit contract. A fallback's dim note surfaces
// through the pane's own note row; a clean long-run exit drops back to
// the text location bar silently.
func (b *Browser) pollLane() {
	if b.lane == nil {
		return
	}
	if b.lane.Poll() {
		if n := b.lane.Note(); n != "" {
			b.note = n
		}
		b.refreshBody()
	}
}

// PollLane — the explicit poll ride (the shots harness + any caller that
// renders outside the pane's own View/Update cadence): observes a dropped
// premium child and lands the exit contract NOW.
func (b *Browser) PollLane() { b.pollLane() }

// SuspendLane — the app flipped the left slot away from the browser
// (ctrl+b to the floor, the pane's q/esc leave): the premium child is
// group-killed + reaped, SILENTLY (a pane switch is never a failure
// note); the URL state keeps for ResumeLane.
func (b *Browser) SuspendLane() {
	if b.lane != nil {
		b.lane.Suspend()
	}
}

// ResumeLane — the app flipped back to the browser: re-spawn the premium
// embed for the current url when the lane still resolves premium and the
// url never fell back (a latched url stays text — no flap). A text-lane
// host no-ops.
func (b *Browser) ResumeLane() {
	if b.lane != nil {
		b.lane.Resume()
	}
}

// PremiumActive — the premium embed is live RIGHT NOW (the app + the
// shots harness read the lane posture through the pane).
func (b *Browser) PremiumActive() bool {
	return b.lane != nil && b.lane.PremiumActive()
}

// LaneNote — the lane's dim fallback note ("" while premium is healthy or
// the last exit was clean).
func (b *Browser) LaneNote() string {
	if b.lane == nil {
		return ""
	}
	return b.lane.Note()
}

// LaneGridHas reports whether needle appears in the live premium child's
// screen model (the harness seam: shots/tests converge on the child's
// paint WITHOUT owning a real terminal). False while the text lane
// paints.
func (b *Browser) LaneGridHas(needle string) bool {
	if b.lane == nil {
		return false
	}
	sess := b.lane.Session()
	if sess == nil {
		return false
	}
	g := sess.Grid()
	for y := 0; y < g.Rows(); y++ {
		if strings.Contains(g.LineText(y), needle) {
			return true
		}
	}
	return false
}

// LaneSessionSize — the live premium child's grid geometry (the
// resize-propagation tests' read seam; ok=false while the text lane
// paints).
func (b *Browser) LaneSessionSize() (cols, rows int, ok bool) {
	if b.lane == nil || b.lane.Session() == nil {
		return 0, 0, false
	}
	cols, rows = b.lane.Session().Size()
	return cols, rows, true
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
	key := fmt.Sprintf("%d|%s|%d|%s|%s|%t|%s|%s|%t",
		b.w, b.url, b.cursor, b.note, b.err, b.loading, chrome.CurrentTheme().Name,
		b.shotStateRow(), b.ShotActive())
	if key == b.rev {
		return
	}
	b.rev = key
	rows, rowOfLink := b.bodyRows()
	b.rowOfLink = rowOfLink
	b.vp.SetContent(strings.Join(rows, "\n"))
}

// bodyRows composes the styled body: optional verdict note, the shot
// lane's status row (rendering / failure / the saved path), then the
// load-state rows (loading / error / starter card / the page itself).
// Returns the rows plus the link→first-row index map (prefix rows shift
// the page's own indexes).
func (b *Browser) bodyRows() ([]string, []int) {
	var rows []string
	if b.note != "" {
		rows = append(rows, chrome.DimText.Render(b.note))
	}
	if shotRow := b.shotStateRow(); shotRow != "" {
		rows = append(rows, chrome.DimText.Render(ansi.Truncate(shotRow, b.wrapW(), "")))
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
//     ::1) on either scheme, and https:// to any host by default — plain
//     http to a non-localhost host is REFUSED unless
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
		return nil, rawurl, fmt.Errorf("unsupported url %q (file://, a bare path, or http(s)://…)", rawurl)
	}
	if !browserFetchAllowed(u.Scheme, u.Hostname()) {
		return nil, rawurl, fmt.Errorf("http fetch blocked: plain http to %s refused (export %s=1 to allow outbound http pages)", u.Hostname(), browserAllowHTTPEnv)
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

// browserFetchAllowed — the fetch gate: localhost always (either scheme),
// https to any host by default, plain http to non-localhost only with
// THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 (read AT USE TIME).
func browserFetchAllowed(scheme, host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if scheme == "https" {
		return true
	}
	return os.Getenv(browserAllowHTTPEnv) == "1"
}
