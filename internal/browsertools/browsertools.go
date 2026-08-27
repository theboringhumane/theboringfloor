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
	"strings"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// MarkerOpen/MarkerClose delimit the one-line directive. The brackets
// are U+27E6/U+27E7 (mathematical white square brackets) — distinctive
// enough that ordinary prose or code never carries them by accident.
const (
	MarkerOpen  = "⟦open-browser:"
	MarkerClose = "⟧"
)

// MaxRequestsPerReply caps the URLs ONE reply may request (the preamble
// contracts a single marker; extras are tolerated up to this cap and
// dropped past it — the marker lines are still stripped).
const MaxRequestsPerReply = 3

// AllowHTTPEnv — the member's outbound-http flag (THE name the browser
// pane's own fetch guard reads too: one flag, two independent layers).
const AllowHTTPEnv = "THEBORINGOFFICE_BROWSER_ALLOW_HTTP"

// markerLineRe matches ONE whole-line directive (multiline): optional
// surrounding whitespace, the URL as the first run of non-space non-⟧
// runes, and the line's trailing newline when present (the strip never
// leaves a ghost blank line). A marker NOT alone on its line is prose,
// not a directive — it stays in the transcript untouched (the
// preamble's "its own line" rule).
var markerLineRe = regexp.MustCompile(`(?m)^[ \t]*⟦open-browser:[ \t]*([^\s⟧]+)[ \t]*⟧[ \t]*\r?$\n?`)

// blankCollapseRe merges the paragraph breaks a stripped marker line
// used to sit between.
var blankCollapseRe = regexp.MustCompile(`\n{3,}`)

// PromptPreamble — the harness instruction riding the FIRST prompt of
// every boss session (both backends): teaches the marker, the
// member-controlled URL policy, and the strip contract (the agent must
// never quote the marker as prose).
const PromptPreamble = "[theboringoffice harness — browser tool]\n" +
	"You can ask the office to open a web page in the member's in-app browser tab. " +
	"To open a page, emit this directive on ITS OWN line, at most once per reply:\n" +
	MarkerOpen + " URL" + MarkerClose + "\n" +
	"URL must be absolute: https:// for any host; http:// only for localhost, 127.0.0.1 or ::1 " +
	"(plain http to any other host is refused unless the member exports " + AllowHTTPEnv + "=1). " +
	"The office strips the directive from your visible reply and performs the open, so never quote " +
	"or explain the marker itself — just place it. Open a page only when it genuinely helps the " +
	"member (docs, dashboards, pull requests, a dev server you started)."

// Decision — the policy verdict for one requested URL.
type Decision struct {
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

// Extract pulls every whole-line marker out of a COMPLETED assistant
// text: it returns the text with the marker lines stripped (no ghost
// blank lines) plus the requested URLs in order of appearance, capped
// at MaxRequestsPerReply (past-cap marker lines still strip — the
// preamble contracts ONE per reply).
func Extract(text string) (string, []string) {
	matches := markerLineRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return text, nil
	}
	urls := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(urls) < MaxRequestsPerReply {
			urls = append(urls, m[1])
		}
	}
	cleaned := markerLineRe.ReplaceAllString(text, "")
	cleaned = blankCollapseRe.ReplaceAllString(cleaned, "\n\n")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned, urls
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

// RequestAll decides every requested URL (already Extract-capped) and
// emits ONE EvBrowserOpen per request — the URL rides Text, the verdict
// rides BrowserOpenAllowed, and a refusal carries BrowserOpenReason for
// the app's red notice. The decisions return so the caller's scrub can
// phrase a marker-only reply's fallback bubble.
func (br *Bridge) RequestAll(urls []string) []Decision {
	out := make([]Decision, 0, len(urls))
	for _, u := range urls {
		d := Decide(u, br.Getenv)
		out = append(out, d)
		if br.Emit == nil {
			continue
		}
		ev := state.Event{Kind: state.EvBrowserOpen, Text: d.URL, BrowserOpenAllowed: d.Allowed}
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
func Scrub(text string, br *Bridge) string {
	cleaned, urls := Extract(text)
	if len(urls) == 0 {
		return text
	}
	decisions := br.RequestAll(urls)
	if cleaned != "" {
		return cleaned
	}
	var opened, refused []string
	for _, d := range decisions {
		if d.Allowed {
			opened = append(opened, d.URL)
		} else {
			refused = append(refused, d.Reason)
		}
	}
	switch {
	case len(refused) == 0:
		return "[theboringoffice] open-browser: " + strings.Join(opened, ", ")
	case len(opened) == 0:
		return "[theboringoffice] open-browser refused: " + strings.Join(refused, "; ")
	default:
		return "[theboringoffice] open-browser: " + strings.Join(opened, ", ") +
			" · refused: " + strings.Join(refused, "; ")
	}
}
