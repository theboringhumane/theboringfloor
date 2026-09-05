// Package action — the office's MUTATING browser engine: the chromedp
// executor behind the ⟦browser-action: URL | op⟧ marker (click / fill /
// eval). It is deliberately SEPARATE from internal/headless (the
// read-only screenshot/snapshot engine, another wave's ownership): same
// discovery (headless.Available), same policy gate (browsertools.Decide
// re-runs at execution time — a refused URL never launches a process),
// same lifecycle shape (one exec allocator + one browser context per
// call, a temp profile chromedp removes on cancel), but an
// action-oriented task set and its own 20s budget.
//
// Every call navigates FRESH (no session reuse — the marker carries the
// full URL per action), waits on chromedp's built-in load/ready waits
// only (no fixed sleeps), and classifies every failure into ONE
// actionable error string the agent's follow-up rides verbatim:
// navigation failure vs selector-not-found vs timeout vs JS exception
// stay distinguishable (errors.Is(err, context.DeadlineExceeded) holds
// for the timeout classes).
package action

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chromedp/chromedp"

	"github.com/theboringhumane/theboringfloor/internal/browsertools"
	"github.com/theboringhumane/theboringfloor/internal/headless"
)

// Op names (the marker grammar's three verbs — kept as plain strings so
// the app's hold/summary code never imports the marker package for
// them).
const (
	OpClick = "click"
	OpFill  = "fill"
	OpEval  = "eval"
)

const (
	// navTimeout — the engine-owned overall budget per call (navigation
	// AND the action's selector wait share it): a hung page or a
	// never-appearing element can never outlive it. Applied on top of
	// the caller's context (a caller deadline still wins when tighter).
	navTimeout = 20 * time.Second

	// EvalCapBytes — the eval result's JSON byte cap (rune-safe: the
	// cut never splits a multi-byte rune). The agent's context window
	// never eats an unbounded page object.
	EvalCapBytes = 4 * 1024
)

// Action — ONE parsed browser-action request: Op is OpClick/OpFill/
// OpEval; Sel the CSS selector (click/fill); Arg the fill value or the
// eval JS expression.
type Action struct {
	Op  string
	Sel string
	Arg string
}

// Result — one executed action: URL is the FINAL (post-action)
// location, Text the op's result string ("clicked <sel>" / "filled
// <sel>" / the eval's JSON-stringified value, EvalCapBytes rune-safe
// capped, "…"-suffixed when truncated).
type Result struct {
	URL  string
	Text string
}

// NavigateAndAct navigates rawurl FRESH (no session reuse), waits for
// the body, performs the ONE action, and reports the final URL plus the
// op result. The URL policy re-runs FIRST (a refusal never launches a
// process) and chrome discovery rides headless.Available (the memoized
// probe the read-only engine uses).
func NavigateAndAct(ctx context.Context, rawurl string, a Action) (*Result, error) {
	switch a.Op {
	case OpClick, OpFill:
		if strings.TrimSpace(a.Sel) == "" {
			return nil, fmt.Errorf("browser-action: %s needs a non-empty selector", a.Op)
		}
	case OpEval:
		if strings.TrimSpace(a.Arg) == "" {
			return nil, errors.New("browser-action: eval needs a non-empty expression")
		}
	default:
		return nil, fmt.Errorf("browser-action: unknown op %q (want click|fill|eval)", a.Op)
	}

	d := browsertools.Decide(rawurl, nil)
	if !d.Allowed {
		return nil, &headless.PolicyError{URL: d.URL, Reason: d.Reason}
	}
	binPath, ok := headless.Available()
	if !ok {
		return nil, headless.ErrChromeNotFound
	}

	runCtx, cancel := context.WithTimeout(ctx, navTimeout)
	defer cancel()
	allocCtx, allocCancel := chromedp.NewExecAllocator(runCtx, allocatorOptions(binPath)...)
	defer allocCancel() // kills chrome + removes the temp profile
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Phase 1 — navigation (load + body ready). A deadline here is the
	// PAGE's fault (timeout class), any other failure is navigation.
	if err := chromedp.Run(browserCtx, chromedp.Tasks{
		chromedp.Navigate(d.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	}); err != nil {
		return nil, classifyNav(d.URL, err)
	}

	// Phase 2 — the action itself. Click/fill WAIT on the selector
	// (chromedp retries until it matches or the context dies), so a
	// deadline HERE means the selector never matched a usable node —
	// the selector class, never confused with a slow page.
	var res Result
	switch a.Op {
	case OpClick:
		if err := chromedp.Run(browserCtx, chromedp.Click(a.Sel, chromedp.ByQuery)); err != nil {
			return nil, classifySel(d.URL, a.Op, a.Sel, err)
		}
		res.Text = "clicked " + a.Sel
	case OpFill:
		if err := chromedp.Run(browserCtx, chromedp.SetValue(a.Sel, a.Arg, chromedp.ByQuery)); err != nil {
			return nil, classifySel(d.URL, a.Op, a.Sel, err)
		}
		res.Text = "filled " + a.Sel
	case OpEval:
		var raw []byte
		if err := chromedp.Run(browserCtx, chromedp.Evaluate(a.Arg, &raw)); err != nil {
			return nil, classifyEval(d.URL, err)
		}
		res.Text = CapEvalJSON(raw)
	}

	// The FINAL location (a click may have navigated): read it on the
	// same still-alive context, best-effort — a post-action navigation
	// racing the read degrades to the requested URL, never a failure.
	res.URL = d.URL
	_ = chromedp.Run(browserCtx, chromedp.Location(&res.URL))
	return &res, nil
}

// classifyNav — phase-1 failures: timeout vs canceled vs navigation.
func classifyNav(url string, err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("browser-action: %s: timed out loading after %s: %w", url, navTimeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("browser-action: %s: canceled: %w", url, err)
	default:
		return fmt.Errorf("browser-action: %s: navigation failed: %w", url, err)
	}
}

// classifySel — phase-2 click/fill failures: a deadline means the
// selector never matched (click additionally waits on VISIBILITY), so
// it names the selector, not the page.
func classifySel(url, op, sel string, err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		if op == OpClick {
			return fmt.Errorf("browser-action: %s: selector %q did not match a visible node within the %s budget: %w", url, sel, navTimeout, err)
		}
		return fmt.Errorf("browser-action: %s: selector %q did not match a node within the %s budget: %w", url, sel, navTimeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("browser-action: %s: canceled: %w", url, err)
	default:
		return fmt.Errorf("browser-action: %s: %s on selector %q failed: %w", url, op, sel, err)
	}
}

// classifyEval — phase-2 eval failures: a JS exception rides back as
// its own error text (never confused with the selector/timeout
// classes).
func classifyEval(url string, err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("browser-action: %s: eval timed out after %s: %w", url, navTimeout, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("browser-action: %s: canceled: %w", url, err)
	default:
		return fmt.Errorf("browser-action: %s: eval failed: %w", url, err)
	}
}

// CapEvalJSON normalizes + caps one eval payload: an empty/undefined
// result reads "null", and the JSON-stringified bytes cut at
// EvalCapBytes on a rune boundary with a "…" tail when truncated.
func CapEvalJSON(raw []byte) string {
	s := string(raw)
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	if len(s) <= EvalCapBytes {
		return s
	}
	max := EvalCapBytes
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max] + "…"
}

// allocatorOptions — the default chromedp set (which includes headless,
// hide-scrollbars and a temp user-data-dir chromedp removes on cancel)
// plus the contract's explicit pins: the discovered binary, the NEW
// headless mode, no GPU, no scrollbars. --no-sandbox joins only when
// running as root on linux (chrome refuses to start there without it).
// The default slice is copied first — appending to chromedp's global
// array would corrupt every later caller. (Mirror of the read-only
// engine's options — headless's own are unexported.)
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
