// Package headless — the office's headless browser engine: a pure-Go CDP
// (chromedp) wrapper that renders URLs in the member's OWN installed
// Chrome-family browser and returns viewport PNG screenshots and text
// snapshots. It is the foundation of the browser tab's screenshot lane
// and the agent's read-only browse tools; NOTHING in here is tied to a
// terminal, a PTY, or an embedded browser process.
//
// Discovery (Available): THEBORINGOFFICE_CHROME=<path> wins first; on
// macOS the three /Applications candidates (Chrome, Chromium, Edge); on
// linux the four PATH names (google-chrome, chromium, chromium-browser,
// microsoft-edge). The probe is memoized — a binary appearing mid-
// session is picked up only on restart.
//
// Policy: EVERY navigation passes browsertools.Decide first (localhost
// always / https default / plain http non-localhost behind
// THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1) — the exact same gate the agent
// marker protocol uses; a refused URL never launches a browser process
// and the decision's reason comes back as the error verbatim.
//
// Lifecycle: each call allocates its own exec allocator + browser
// context (temp profile auto-created and removed by chromedp), imposes
// its own 15s overall budget on top of the caller's context, and waits
// on chromedp's built-in load/ready waits only — no fixed sleeps. A
// hung page can never outlive the timeout; a refusal never spawns.
//
// Caching: the public Screenshot / Snapshot fronts ride the engine's
// render cache (cache.go) — a 30s LRU memo per (url, box) with
// singleflight fan-out, a 5s negative cache for timeouts/navigation
// failures, and copy-on-return values — so the agent-tool path and the
// pane display path never re-render the same page twice in a short
// window. The cache holds values only; every render still rides the
// lifecycle above.
package headless

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
)

// Result — one viewport screenshot: the final (post-redirect) URL, the
// page title, and the PNG bytes at deviceScaleFactor 2 (pixel dims are
// widthPx*2 x heightPx*2).
type Result struct {
	URL   string
	Title string
	PNG   []byte
}

// SnapResult — one text snapshot: the final URL, the page title, the
// body's visible innerText trimmed and capped at maxText runes, and the
// page's anchors as absolute URLs (deduped by exact URL, capped at 50,
// javascript:/mailto: skipped).
type SnapResult struct {
	URL   string
	Title string
	Text  string
	Links []Link
}

// Link — one extracted anchor: trimmed visible text + absolute URL.
type Link struct {
	Text string
	URL  string
}

// PolicyError — a browsertools.Decide refusal. Error() is the decision
// reason VERBATIM (member-facing copy owned by browsertools); errors.As
// lets callers tell "the member's policy said no" apart from engine
// failures without string matching.
type PolicyError struct {
	URL    string
	Reason string
}

// Error — the refusal reason, verbatim.
func (e *PolicyError) Error() string { return e.Reason }

// ErrChromeNotFound — Available found no Chrome-family binary: the fix
// is installing Chrome/Chromium/Edge or exporting THEBORINGOFFICE_CHROME.
var ErrChromeNotFound = errors.New("headless: no Chrome-family browser found (install Chrome, Chromium or Edge, or set THEBORINGOFFICE_CHROME)")

const (
	// chromeEnv — the member's explicit binary-path override.
	chromeEnv = "THEBORINGOFFICE_CHROME"

	// navTimeout — the engine-owned overall budget per call. Applied on
	// top of the caller's context (a caller deadline still wins when
	// tighter); the point is the budget exists even when the caller
	// passes a deadline-free context.
	navTimeout = 15 * time.Second

	// deviceScale — deviceScaleFactor for every screenshot: 2 keeps
	// text retina-crisp when the pane scales the PNG down.
	deviceScale = 2.0

	// maxLinks — the contract's anchor cap.
	maxLinks = 50
)

// darwinCandidates — the macOS install paths, in preference order.
var darwinCandidates = []string{
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
}

// pathCandidates — the PATH names probed everywhere else (linux), in
// preference order.
var pathCandidates = []string{
	"google-chrome",
	"chromium",
	"chromium-browser",
	"microsoft-edge",
}

// Probe seams — discover takes every side effect as a parameter so the
// discovery table tests need no real filesystem; Available memoizes the
// one prod probe.
var (
	probeOnce sync.Once
	probePath string
	probeOK   bool
)

// Available returns the resolved Chrome-family binary and true, or
// ("", false). Memoized: the probe runs once per process.
func Available() (path string, ok bool) {
	probeOnce.Do(func() {
		probePath, probeOK = discover(runtime.GOOS, os.Getenv, fileExists, exec.LookPath)
	})
	return probePath, probeOK
}

// discover — the pure probe: env override first, then the platform
// candidate list. stat answers "is this exact path a file", lookPath
// answers "where is this name on PATH".
func discover(goos string, getenv func(string) string, stat func(string) bool, lookPath func(string) (string, error)) (string, bool) {
	if p := strings.TrimSpace(getenv(chromeEnv)); p != "" && stat(p) {
		return p, true
	}
	if goos == "darwin" {
		for _, p := range darwinCandidates {
			if stat(p) {
				return p, true
			}
		}
		return "", false
	}
	for _, name := range pathCandidates {
		if p, err := lookPath(name); err == nil {
			return p, true
		}
	}
	return "", false
}

// fileExists — the prod stat seam: regular file, not a directory.
func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

// Screenshot renders rawurl at EXACTLY widthPx x heightPx CSS pixels
// (deviceScaleFactor 2) and returns the viewport PNG plus the final URL
// and title. The page load event + body readiness are the only waits —
// no fixed sleeps; full-page is deliberately false (the pane shows what
// fits the viewport). Results ride the render cache (cache.go): a repeat
// call for the same (url, box) within 30s — or a concurrent one — never
// spawns a second Chrome run.
func Screenshot(ctx context.Context, rawurl string, widthPx, heightPx int) (*Result, error) {
	if widthPx <= 0 || heightPx <= 0 {
		return nil, fmt.Errorf("headless: invalid viewport %dx%d (both dimensions must be > 0)", widthPx, heightPx)
	}
	return shotCache.do(shotKey(rawurl, widthPx, heightPx), func() (*Result, error) {
		return execScreenshot(ctx, rawurl, widthPx, heightPx)
	})
}

// renderScreenshot — the uncached engine body (the prod executor behind
// the execScreenshot seam). The viewport check lives in the public front
// so a caller bug never lands in the cache.
func renderScreenshot(ctx context.Context, rawurl string, widthPx, heightPx int) (*Result, error) {
	var res Result
	err := run(ctx, rawurl, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.EmulateViewport(int64(widthPx), int64(heightPx), chromedp.EmulateScale(deviceScale)),
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Location(&res.URL),
			chromedp.Title(&res.Title),
			chromedp.CaptureScreenshot(&res.PNG),
		}
	})
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Snapshot renders rawurl and returns its title, visible body text
// (trimmed, capped at maxText runes), and absolute links (deduped,
// capped at 50). maxText <= 0 caps the text to empty. Results ride the
// render cache (cache.go), keyed by (url, maxText) — same 30s / share /
// negative-cache discipline as Screenshot.
func Snapshot(ctx context.Context, rawurl string, maxText int) (*SnapResult, error) {
	return snapCache.do(snapKey(rawurl, maxText), func() (*SnapResult, error) {
		return execSnapshot(ctx, rawurl, maxText)
	})
}

// renderSnapshot — the uncached engine body (the prod executor behind
// the execSnapshot seam).
func renderSnapshot(ctx context.Context, rawurl string, maxText int) (*SnapResult, error) {
	var res SnapResult
	var page snapshotPage
	err := run(ctx, rawurl, func(url string) chromedp.Tasks {
		return chromedp.Tasks{
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Location(&res.URL),
			chromedp.Title(&res.Title),
			chromedp.Evaluate(snapshotJS, &page),
		}
	})
	if err != nil {
		return nil, err
	}
	res.Text = capRunes(page.Text, maxText)
	res.Links = normalizeLinks(page.Links)
	return &res, nil
}

// snapshotPage — the JS evaluation payload (see snapshotJS).
type snapshotPage struct {
	Text  string `json:"text"`
	Links []Link `json:"links"`
}

// snapshotJS — one round trip: the body's visible innerText plus every
// anchor's trimmed text and RESOLVED (absolute) href. Scheme filtering,
// dedup and the cap happen in Go (normalizeLinks) where they're
// unit-testable without a browser.
const snapshotJS = `(() => {
	const links = [];
	for (const a of document.querySelectorAll("a[href]")) {
		links.push({text: (a.textContent || "").trim(), url: a.href});
	}
	return {text: document.body ? document.body.innerText : "", links: links};
})()`

// run — the shared engine path: policy FIRST (a refusal never launches
// a process), then discovery, then one allocator + one browser context
// under the engine-owned 15s budget. makeTasks receives the normalized
// (policy-trimmed) URL so callers navigate exactly what Decide allowed.
func run(ctx context.Context, rawurl string, makeTasks func(url string) chromedp.Tasks) error {
	d := browsertools.Decide(rawurl, os.Getenv)
	if !d.Allowed {
		return &PolicyError{URL: d.URL, Reason: d.Reason}
	}
	binPath, ok := Available()
	if !ok {
		return ErrChromeNotFound
	}

	runCtx, cancel := context.WithTimeout(ctx, navTimeout)
	defer cancel()
	allocCtx, allocCancel := chromedp.NewExecAllocator(runCtx, allocatorOptions(binPath)...)
	defer allocCancel() // kills chrome + removes the temp profile
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	if err := chromedp.Run(browserCtx, makeTasks(d.URL)); err != nil {
		return classify(d.URL, err)
	}
	return nil
}

// NavError — a navigation/render failure (DNS, dead port, renderer
// crash…): the page never came up. Typed (Unwrap keeps the cause) so the
// cache's negative-entry discipline — and any caller — can tell it apart
// from policy refusals and timeouts without string matching. Error()
// keeps the wave-85 "headless: <url>: navigation failed: <cause>"
// wording byte-identical.
type NavError struct {
	URL string
	Err error
}

// Error — "headless: <url>: navigation failed: <cause>".
func (e *NavError) Error() string {
	return fmt.Sprintf("headless: %s: navigation failed: %v", e.URL, e.Err)
}

// Unwrap — the underlying chromedp/ctx cause.
func (e *NavError) Unwrap() error { return e.Err }

// classify turns chromedp/ctx failures into actionable wrapped errors:
// timeout vs canceled vs navigation/render failure stay distinguishable
// (and errors.Is(err, context.DeadlineExceeded) holds for the first).
func classify(url string, err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("headless: %s: timed out after %s: %w", url, navTimeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("headless: %s: canceled: %w", url, err)
	default:
		return &NavError{URL: url, Err: err}
	}
}

// allocatorOptions — the default chromedp set (which includes headless,
// hide-scrollbars and a temp user-data-dir chromedp removes on cancel)
// plus the contract's explicit pins: the discovered binary, the NEW
// headless mode, no GPU, no scrollbars. --no-sandbox joins only when
// running as root on linux (chrome refuses to start there without it).
// The default slice is copied first — appending to chromedp's global
// array would corrupt every later caller.
func allocatorOptions(binPath string) []chromedp.ExecAllocatorOption {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.ExecPath(binPath),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("hide-scrollbars", true),
	)
	if runtime.GOOS == "linux" && runningAsRoot() {
		opts = append(opts, chromedp.Flag("no-sandbox", true))
	}
	return opts
}

// capRunes — trim, then cap at max runes (rune-safe: multi-byte text
// never splits mid-codepoint). max <= 0 yields the empty string.
func capRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

// normalizeLinks — dedup by exact URL, drop empty and
// javascript:/mailto: entries (scheme match is case-insensitive), cap
// at maxLinks, preserving page order.
func normalizeLinks(raw []Link) []Link {
	seen := make(map[string]bool, len(raw))
	out := make([]Link, 0, maxLinks)
	for _, l := range raw {
		u := strings.TrimSpace(l.URL)
		if u == "" || seen[u] {
			continue
		}
		lower := strings.ToLower(u)
		if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "mailto:") {
			continue
		}
		seen[u] = true
		out = append(out, Link{Text: strings.TrimSpace(l.Text), URL: u})
		if len(out) >= maxLinks {
			break
		}
	}
	return out
}
