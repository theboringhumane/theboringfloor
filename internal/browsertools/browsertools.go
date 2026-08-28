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
// The tool speaks THREE directives, each on its own line:
//
//	⟦open-browser: URL⟧        — open the page in the member's tab
//	⟦browser-screenshot: URL⟧  — render it for the member (PNG saved)
//	⟦browser-snapshot: URL⟧    — read text+links BACK to the agent
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

// MarkerOpen/MarkerShot/MarkerSnap open the three one-line directives;
// MarkerClose delimits them all. The brackets are U+27E6/U+27E7
// (mathematical white square brackets) — distinctive enough that
// ordinary prose or code never carries them by accident.
const (
	MarkerOpen  = "⟦open-browser:"
	MarkerShot  = "⟦browser-screenshot:"
	MarkerSnap  = "⟦browser-snapshot:"
	MarkerClose = "⟧"
)

// Kind — which browser-tool directive a parsed marker carries. The
// string values double as the scrub-fallback's bubble labels.
type Kind string

const (
	KindOpen Kind = "open-browser"
	KindShot Kind = "browser-screenshot"
	KindSnap Kind = "browser-snapshot"
)

// Request — ONE parsed marker: its directive kind and the (raw,
// policy-pending) URL it carries.
type Request struct {
	Kind Kind
	URL  string
}

// MaxRequestsPerReply caps the URLs ONE reply may request, SPANNING all
// three marker kinds (the preamble contracts one of each; extras are
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

// markerKinds — the three directive matchers, in kind order (extraction
// itself re-sorts hits by position, so cross-kind order of appearance
// is preserved).
var markerKinds = []struct {
	kind Kind
	re   *regexp.Regexp
}{
	{KindOpen, markerLineRe(MarkerOpen)},
	{KindShot, markerLineRe(MarkerShot)},
	{KindSnap, markerLineRe(MarkerSnap)},
}

// blankCollapseRe merges the paragraph breaks a stripped marker line
// used to sit between.
var blankCollapseRe = regexp.MustCompile(`\n{3,}`)

// PromptPreamble — the harness instruction riding the FIRST prompt of
// every boss session (both backends): teaches the markers, the
// member-controlled URL policy, and the strip contract (the agent must
// never quote a marker as prose). The open-browser paragraph is
// byte-stable (backend tests pin it); new capabilities APPEND lines.
const PromptPreamble = "[theboringoffice harness — browser tool]\n" +
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
	"message — use it to READ pages."

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

// Extract pulls every whole-line marker (all three kinds) out of a
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
			hits = append(hits, hit{at: m[0], req: Request{Kind: mk.kind, URL: text[m[2]:m[3]]}})
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
	default:
		return state.EvBrowserOpen
	}
}

// RequestAll decides every request (already Extract-capped) and emits
// ONE event per request — EvBrowserOpen / EvBrowserScreenshot /
// EvBrowserSnapshot by directive kind. The URL rides Text, the verdict
// rides BrowserOpenAllowed, and a refusal carries BrowserOpenReason for
// the app's red notice (ONE field contract across all three kinds — the
// app never branches on kind to read the verdict). The decisions return
// so the caller's scrub can phrase a marker-only reply's fallback
// bubble.
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
		br.Emit(ev)
	}
	return out
}

// Scrub is the backend pin-path helper: extract markers from a
// COMPLETED assistant text, fire the bridge for the requests, and
// return the transcript text (markers gone). A marker-ONLY reply
// degrades to a one-line office note — the pinned bubble never goes
// blank (the send-side typing placeholder needs real text to settle).
// The note names each directive kind ("[theboringoffice] open-browser:
// u · browser-snapshot: u"); a refusal-only note names the kind when
// every refusal rode one ("… browser-screenshot refused: r"), else the
// generic "browser refused".
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
	for _, k := range []Kind{KindOpen, KindShot, KindSnap} {
		if urls := byKind[k]; len(urls) > 0 {
			allowed = append(allowed, string(k)+": "+strings.Join(urls, ", "))
		}
	}
	switch {
	case len(refused) == 0:
		return "[theboringoffice] " + strings.Join(allowed, " · ")
	case len(allowed) == 0:
		label := "browser"
		if len(refusedKinds) == 1 {
			for k := range refusedKinds {
				label = string(k)
			}
		}
		return "[theboringoffice] " + label + " refused: " + strings.Join(refused, "; ")
	default:
		return "[theboringoffice] " + strings.Join(allowed, " · ") +
			" · refused: " + strings.Join(refused, "; ")
	}
}
