// Package browsertools — the office's agent-facing browser tool: a
// marker protocol, a URL policy, and the event bridge BOTH backends
// (opencode + claude) share.
//
// The protocol is backend-agnostic by design (no backend config
// injection): the office's harness preamble (PromptPreamble) rides the
// FIRST prompt of each boss session — the conciergePreamble house
// pattern — and teaches the agent ONE directive: to open a page in the
// member's in-app browser tab, emit
//
//	⟦open-browser: https://example.com/docs⟧
//
// on its OWN line. The backend's completion-pin path (the ONE place the
// final transcript text is fixed — maybeBossCompleted for opencode,
// emitMapped's pin leg for claude) extracts every marker via Scrub,
// strips it from the pinned text, and hands the requested URLs to the
// Bridge, which runs the policy ONCE for both backends and emits one
// state.EvBrowserOpen per request (allowed → the app opens; refused →
// the app posts the reason). Stream DELTAS are never scrubbed (a marker
// can straddle delta boundaries — the completion pin supersedes the
// growing bubble, so the transcript at rest is always clean).
//
// The tool speaks FOUR directives, each on its own line:
//
//	⟦open-browser: URL⟧        — open the page in the member's tab
//	⟦browser-screenshot: URL⟧  — render it for the member (PNG saved)
//	⟦browser-snapshot: URL⟧    — read text+links BACK to the agent
//	⟦browser-action: URL | op⟧ — MUTATE the page (click/fill/eval) —
//	                              ALWAYS gated by the member's permission
//	                              modal (approve-once; even localhost asks)
//
// The first three are READ-ONLY; the fourth mutates, which is why its
// gate is the member's modal and not just the URL policy.
//
// The policy (Decide): localhost/127.0.0.1/::1 always; https:// to any
// host by default; plain http:// to any other host refused unless
// THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 (read AT USE TIME, never
// latched); every other scheme refused. This gates the AGENT's request
// only — the browser PANE's own fetch guard (internal/panels) is a
// second, stricter, member-owned network layer reading the same flag;
// the two never mix.
package browsertools

import (
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// MarkerOpen/MarkerShot/MarkerSnap/MarkerAct open the four one-line
// directives; MarkerClose delimits them all. The brackets are
// U+27E6/U+27E7 (mathematical white square brackets) — distinctive
// enough that ordinary prose or code never carries them by accident.
const (
	MarkerOpen  = "⟦open-browser:"
	MarkerShot  = "⟦browser-screenshot:"
	MarkerSnap  = "⟦browser-snapshot:"
	MarkerAct   = "⟦browser-action:"
	MarkerClose = "⟧"
)

// Kind — which browser-tool directive a parsed marker carries. The
// string values double as the scrub-fallback's bubble labels.
type Kind string

const (
	KindOpen   Kind = "open-browser"
	KindShot   Kind = "browser-screenshot"
	KindSnap   Kind = "browser-snapshot"
	KindAction Kind = "browser-action"
)

// Action ops (Request.Op when Kind == KindAction) — the STRICT grammar
// the actionLineRe matchers encode:
//
//	⟦browser-action: URL | click: CSS-SELECTOR⟧
//	⟦browser-action: URL | fill: CSS-SELECTOR = VALUE⟧
//	⟦browser-action: URL | eval: JS-EXPRESSION⟧
const (
	ActionOpClick = "click"
	ActionOpFill  = "fill"
	ActionOpEval  = "eval"
)

// Request — ONE parsed marker: its directive kind and the (raw,
// policy-pending) URL it carries. Op/Sel/Arg are the KindAction payload
// (zero on every other kind): Op is ActionOpClick/Fill/Eval, Sel the
// CSS selector (click/fill), Arg the fill value or the eval JS
// expression.
type Request struct {
	Kind Kind
	URL  string
	Op   string
	Sel  string
	Arg  string
}

// MaxRequestsPerReply caps the URLs ONE reply may request, SPANNING all
// four marker kinds (the preamble contracts one of each; extras are
// tolerated up to this cap and dropped past it — the marker lines are
// still stripped).
const MaxRequestsPerReply = 3

// AllowHTTPEnv — the member's outbound-http flag (THE name the browser
// pane's own fetch guard reads too: one flag, two independent layers).
const AllowHTTPEnv = "THEBORINGOFFICE_BROWSER_ALLOW_HTTP"

// markerLineRe builds ONE whole-line directive matcher (multiline):
// optional surrounding whitespace, the URL as the first run of
// non-space non-⟧ runes, and the line's trailing newline when present
// (the strip never leaves a ghost blank line). A marker NOT alone on
// its line is prose, not a directive — it stays in the transcript
// untouched (the preamble's "its own line" rule).
func markerLineRe(marker string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(marker) + `[ \t]*([^\s⟧]+)[ \t]*⟧[ \t]*\r?$\n?`)
}

// actionLineRe builds ONE STRICT whole-line browser-action matcher for a
// single op's spec grammar: the URL as the first non-space non-⟧ run, a
// " | " separator (surrounding whitespace optional), then the op's own
// capture shape (a non-empty selector for click; a non-empty
// no-⟧-no-= selector, the first " = " split, and a non-empty value —
// which MAY carry spaces and further '=' — for fill; any non-empty
// non-⟧ expression for eval). A line that starts like a browser-action
// marker but fits NO op's grammar matches NOTHING here: it stays in the
// transcript as visible prose and NEVER acts (malformed = visible,
// never silently executed).
func actionLineRe(spec string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(MarkerAct) +
		`[ \t]*([^\s⟧]+)[ \t]*\|[ \t]*` + spec + `[ \t]*⟧[ \t]*\r?$\n?`)
}

// The STRICT per-op spec grammars (selector starts non-space; lazy
// bodies let the trailing [ \t]*⟧ own the boundary whitespace).
const (
	actionClickSpec = `click:[ \t]*(\S[^⟧]*?)`
	actionFillSpec  = `fill:[ \t]*(\S[^⟧=]*?)[ \t]*=[ \t]*(\S[^⟧]*?)`
	actionEvalSpec  = `eval:[ \t]*(\S[^⟧]*?)`
)

// markerKinds — the four directive matchers (three action grammars, one
// per op), in kind order (extraction itself re-sorts hits by position,
// so cross-kind order of appearance is preserved). build assembles the
// Request from the match's submatch indexes; a nil build is the plain
// URL-only shape (group 1).
var markerKinds = []struct {
	kind  Kind
	re    *regexp.Regexp
	build func(text string, m []int) Request
}{
	{KindOpen, markerLineRe(MarkerOpen), nil},
	{KindShot, markerLineRe(MarkerShot), nil},
	{KindSnap, markerLineRe(MarkerSnap), nil},
	{KindAction, actionLineRe(actionClickSpec), func(text string, m []int) Request {
		return Request{Kind: KindAction, URL: text[m[2]:m[3]], Op: ActionOpClick, Sel: text[m[4]:m[5]]}
	}},
	{KindAction, actionLineRe(actionFillSpec), func(text string, m []int) Request {
		return Request{Kind: KindAction, URL: text[m[2]:m[3]], Op: ActionOpFill, Sel: text[m[4]:m[5]], Arg: text[m[6]:m[7]]}
	}},
	{KindAction, actionLineRe(actionEvalSpec), func(text string, m []int) Request {
		return Request{Kind: KindAction, URL: text[m[2]:m[3]], Op: ActionOpEval, Arg: text[m[4]:m[5]]}
	}},
}

// blankCollapseRe merges the paragraph breaks a stripped marker line
// used to sit between.
var blankCollapseRe = regexp.MustCompile(`\n{3,}`)

// PromptPreamble — the harness instruction riding the FIRST prompt of
// every boss session (both backends): teaches the markers, the
// member-controlled URL policy, and the strip contract (the agent must
// never quote a marker as prose). The browser-policy paragraph is
// intentionally first, before directive syntax, and the full byte contract
// is pinned by browsertools and both backend first-prompt tests.
const PromptPreamble = "[theboringoffice harness — browser tool]\n" +
	"Browser policy: prefer the office's built-in browser directives for every URL, including localhost and external pages. " +
	"Use open-browser to show a page, browser-screenshot to capture it for the member, and browser-snapshot to read text/links yourself. " +
	"Do not launch Chrome, Chromium, Playwright, Puppeteer, terminal-browser, or another browser process unless the member explicitly asks for an external browser or the built-in directive fails; if it fails, explain why before falling back. " +
	"The member-facing slash command is /open <url>; agents use the directives below.\n" +
	"You can ask the office to open a web page in the member's in-app browser tab. " +
	"To open a page, emit this directive on ITS OWN line, at most once per reply:\n" +
	MarkerOpen + " URL" + MarkerClose + "\n" +
	"URL must be absolute: https:// for any host; http:// only for localhost, 127.0.0.1 or ::1 " +
	"(plain http to any other host is refused unless the member exports " + AllowHTTPEnv + "=1). " +
	"The office strips the directive from your visible reply and performs the open, so never quote " +
	"or explain the marker itself — just place it. Open a page only when it genuinely helps the " +
	"member (docs, dashboards, pull requests, a dev server you started)." +
	"\nTwo read-only siblings (same own-line rule, same URL policy, at most one of each per reply, " +
	"3 browser directives total per reply):\n" +
	MarkerShot + " URL" + MarkerClose + " — render the page in the member's browser tab as an image " +
	"(kitty terminals) and save the PNG (the member sees the path).\n" +
	MarkerSnap + " URL" + MarkerClose + " — fetch the page's text + links back to YOU as a follow-up " +
	"message — use it to READ pages." +
	"\nOne MUTATING sibling (same own-line rule, same URL policy, counts toward the 3-directive cap) — " +
	"it CHANGES the page, so the member's permission prompt ALWAYS asks first (approve-once only; " +
	"there is no standing grant, not even for localhost):\n" +
	MarkerAct + " URL | click: CSS-SELECTOR" + MarkerClose + " — click an element.\n" +
	MarkerAct + " URL | fill: CSS-SELECTOR = VALUE" + MarkerClose + " — set an input's value " +
	"(VALUE may contain spaces and '=').\n" +
	MarkerAct + " URL | eval: JS-EXPRESSION" + MarkerClose + " — evaluate JavaScript on the page; " +
	"the JSON result comes back to YOU.\n" +
	"Each action drives a FRESH page load (no session reuse). The outcome — the action's result, " +
	"the error, or the member's rejection — arrives as a follow-up message. " +
	"open-browser/browser-screenshot/browser-snapshot are READ-ONLY; browser-action MUTATES — " +
	"prefer the read-only directives whenever reading is enough."

// Decision — the policy verdict for one requested URL. Kind is the
// directive the request rode (RequestAll stamps it; Decide alone leaves
// it zero — the policy itself is kind-agnostic).
type Decision struct {
	Kind    Kind
	URL     string // the normalized (trimmed) request
	Allowed bool
	Reason  string // member-facing refusal when !Allowed ("" when Allowed)
}

// localhostHosts — the always-allowed loopback names (the pane's fetch
// guard whitelists exactly these three).
var localhostHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

// Decide runs the URL policy for ONE agent request. getenv is the env
// seam (os.Getenv in prod — read AT USE TIME, never latched).
func Decide(rawurl string, getenv func(string) string) Decision {
	u := strings.TrimSpace(rawurl)
	if u == "" {
		return Decision{URL: rawurl, Reason: "empty URL"}
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Decision{URL: u, Reason: "not an absolute http(s) URL"}
	}
	host := strings.ToLower(parsed.Hostname())
	if localhostHosts[host] {
		return Decision{URL: u, Allowed: true}
	}
	if parsed.Scheme == "https" {
		return Decision{URL: u, Allowed: true}
	}
	if getenv != nil && getenv(AllowHTTPEnv) == "1" {
		return Decision{URL: u, Allowed: true}
	}
	return Decision{URL: u, Reason: "plain http to " + host + " refused — export " + AllowHTTPEnv + "=1 to allow outbound http pages"}
}

// Extract pulls every whole-line marker (all four kinds) out of a
// COMPLETED assistant text: it returns the text with the marker lines
// stripped (no ghost blank lines) plus the requests in order of
// appearance ACROSS kinds, capped at MaxRequestsPerReply (past-cap
// marker lines still strip — the preamble contracts one of each kind
// per reply).
func Extract(text string) (string, []Request) {
	type hit struct {
		at  int
		req Request
	}
	var hits []hit
	for _, mk := range markerKinds {
		for _, m := range mk.re.FindAllStringSubmatchIndex(text, -1) {
			req := Request{Kind: mk.kind, URL: text[m[2]:m[3]]}
			if mk.build != nil {
				req = mk.build(text, m)
			}
			hits = append(hits, hit{at: m[0], req: req})
		}
	}
	if len(hits) == 0 {
		return text, nil
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
	reqs := make([]Request, 0, len(hits))
	for _, h := range hits {
		if len(reqs) < MaxRequestsPerReply {
			reqs = append(reqs, h.req)
		}
	}
	cleaned := text
	for _, mk := range markerKinds {
		cleaned = mk.re.ReplaceAllString(cleaned, "")
	}
	cleaned = blankCollapseRe.ReplaceAllString(cleaned, "\n\n")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned, reqs
}

// Bridge turns requested URLs into state events: the ONE policy
// decision point both backends share. Emit is the backend's event sink
// (flow.emit via a closure — read AT CALL TIME, so a pre-Start bridge
// is a safe no-op); Getenv is the env seam (os.Getenv in prod).
type Bridge struct {
	Emit   func(state.Event)
	Getenv func(string) string
}

// NewBridge wires the prod bridge (os.Getenv).
func NewBridge(emit func(state.Event)) *Bridge {
	return &Bridge{Emit: emit, Getenv: os.Getenv}
}

// eventKind maps a directive kind to its state event kind (the app's
// reaction table keys off these).
func eventKind(k Kind) state.EventKind {
	switch k {
	case KindShot:
		return state.EvBrowserScreenshot
	case KindSnap:
		return state.EvBrowserSnapshot
	case KindAction:
		return state.EvBrowserAction
	default:
		return state.EvBrowserOpen
	}
}

// RequestAll decides every request (already Extract-capped) and emits
// ONE event per request — EvBrowserOpen / EvBrowserScreenshot /
// EvBrowserSnapshot / EvBrowserAction by directive kind. The URL rides
// Text, the verdict rides BrowserOpenAllowed, and a refusal carries
// BrowserOpenReason for the app's red notice (ONE field contract across
// all four kinds — the app never branches on kind to read the verdict);
// a browser-action event ALSO carries the parsed action on
// BrowserActionOp/Sel/Arg (the app parks it behind the member's
// permission modal — the verdict here is the URL policy only, never the
// member's grant). The decisions return so the caller's scrub can
// phrase a marker-only reply's fallback bubble.
func (br *Bridge) RequestAll(reqs []Request) []Decision {
	out := make([]Decision, 0, len(reqs))
	for _, r := range reqs {
		d := Decide(r.URL, br.Getenv)
		d.Kind = r.Kind
		out = append(out, d)
		if br.Emit == nil {
			continue
		}
		ev := state.Event{Kind: eventKind(r.Kind), Text: d.URL, BrowserOpenAllowed: d.Allowed}
		if !d.Allowed {
			ev.BrowserOpenReason = d.Reason
		}
		if r.Kind == KindAction {
			ev.BrowserActionOp, ev.BrowserActionSel, ev.BrowserActionArg = r.Op, r.Sel, r.Arg
		}
		br.Emit(ev)
	}
	return out
}

// Scrub is the backend pin-path helper: extract markers from a
// COMPLETED assistant text, fire the bridge for the requests, and
// return the transcript text (markers gone). A marker-ONLY reply
// degrades to a one-line office note — the pinned bubble never goes
// blank (the send-side typing placeholder needs real text to settle).
// The note names each directive kind ("[theboringfloor] open-browser:
// u · browser-snapshot: u"); a refusal-only note names the kind when
// every refusal rode one ("… browser-screenshot refused: r"), else the
// generic "browser refused". A MALFORMED browser-action marker matches
// no strict grammar, so it never extracts — it stays as visible prose
// and never acts.
func Scrub(text string, br *Bridge) string {
	cleaned, reqs := Extract(text)
	if len(reqs) == 0 {
		return text
	}
	decisions := br.RequestAll(reqs)
	if cleaned != "" {
		return cleaned
	}
	byKind := map[Kind][]string{}
	var refused []string
	refusedKinds := map[Kind]bool{}
	for _, d := range decisions {
		if d.Allowed {
			byKind[d.Kind] = append(byKind[d.Kind], d.URL)
		} else {
			refused = append(refused, d.Reason)
			refusedKinds[d.Kind] = true
		}
	}
	var allowed []string
	for _, k := range []Kind{KindOpen, KindShot, KindSnap, KindAction} {
		if urls := byKind[k]; len(urls) > 0 {
			allowed = append(allowed, string(k)+": "+strings.Join(urls, ", "))
		}
	}
	switch {
	case len(refused) == 0:
		return "[theboringfloor] " + strings.Join(allowed, " · ")
	case len(allowed) == 0:
		label := "browser"
		if len(refusedKinds) == 1 {
			for k := range refusedKinds {
				label = string(k)
			}
		}
		return "[theboringfloor] " + label + " refused: " + strings.Join(refused, "; ")
	default:
		return "[theboringfloor] " + strings.Join(allowed, " · ") +
			" · refused: " + strings.Join(refused, "; ")
	}
}
