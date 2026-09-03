// browsertools_test.go — the browser tool's contracts:
//
//	policy (Decide) — localhost always, https by default, plain http
//	     non-localhost only under THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1,
//	     every other scheme refused, each refusal carrying the exact
//	     member-facing reason (ONE policy for all four marker kinds);
//	protocol (Extract) — whole-line markers of ALL FOUR kinds strip
//	     cleanly (no ghost blank lines), mid-line/unterminated markers
//	     are prose, the STRICT browser-action grammar (click/fill/eval)
//	     parses its payload or stays VISIBLE prose (malformed never
//	     silently acts), and the per-reply cap spans kinds in order of
//	     appearance;
//	bridge (RequestAll/Scrub) — the fake-emit sink sees one correctly
//	     shaped EvBrowserOpen/EvBrowserScreenshot/EvBrowserSnapshot/
//	     EvBrowserAction per request (the action event carries the
//	     parsed op/sel/arg payload), and a marker-only reply degrades
//	     to the one-line office note (never a blank pin);
//	preamble — the agent-facing text keeps the browser-policy paragraph
//	     before directive syntax and pins the entire byte contract.
package browsertools

import (
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// flagEnv pins the allow-http flag for one Decide/Bridge call.
func flagEnv(on bool) func(string) string {
	return func(k string) string {
		if on && k == AllowHTTPEnv {
			return "1"
		}
		return ""
	}
}

func TestDecidePolicyTable(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		flagOn  bool
		allowed bool
		reason  string // "" → only assert Allowed
	}{
		{"https any host, flag off", "https://theboring.name", false, true, ""},
		{"https deep path + query", "https://theboring.name/docs/keys?x=1#y", false, true, ""},
		{"http localhost, flag off", "http://localhost:3000/dev", false, true, ""},
		{"http 127.0.0.1, flag off", "http://127.0.0.1:8080", false, true, ""},
		{"http ::1, flag off", "http://[::1]:9000", false, true, ""},
		{"https localhost", "https://localhost:8443", false, true, ""},
		{"http non-localhost, flag off", "http://theboring.name", false, false,
			"plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"},
		{"http non-localhost, flag ON", "http://theboring.name", true, true, ""},
		{"http non-localhost ip, flag off", "http://203.0.113.7", false, false,
			"plain http to 203.0.113.7 refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"},
		{"file scheme refused", "file:///tmp/x.html", false, false, "not an absolute http(s) URL"},
		{"bare path refused", "/tmp/x.html", false, false, "not an absolute http(s) URL"},
		{"ftp refused", "ftp://example.com/pub", false, false, "not an absolute http(s) URL"},
		{"javascript refused", "javascript:alert(1)", false, false, "not an absolute http(s) URL"},
		{"scheme-only refused", "https://", false, false, "not an absolute http(s) URL"},
		{"whitespace trimmed", "  https://theboring.name  ", false, true, ""},
		{"empty refused", "   ", false, false, "empty URL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := Decide(c.url, flagEnv(c.flagOn))
			if d.Allowed != c.allowed {
				t.Fatalf("Decide(%q, flag=%v).Allowed = %v, want %v (reason %q)",
					c.url, c.flagOn, d.Allowed, c.allowed, d.Reason)
			}
			if c.reason != "" && d.Reason != c.reason {
				t.Fatalf("Decide(%q).Reason = %q, want %q", c.url, d.Reason, c.reason)
			}
			if c.allowed && d.Reason != "" {
				t.Fatalf("an allowed decision carries no reason, got %q", d.Reason)
			}
		})
	}
}

func TestDecideReadsEnvAtUseTime(t *testing.T) {
	// the flag is NEVER latched: the same process flips the env between
	// two requests and the verdict follows.
	t.Setenv(AllowHTTPEnv, "")
	if d := Decide("http://theboring.name", flagEnv(false)); d.Allowed {
		t.Fatal("flag unset must refuse plain http")
	}
	if d := Decide("http://theboring.name", flagEnv(true)); !d.Allowed {
		t.Fatal("a getenv reading 1 must allow plain http")
	}
}

func TestExtractTable(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		reqs []Request
	}{
		{
			"no markers is identity",
			"plain prose\n\nwith paragraphs",
			"plain prose\n\nwith paragraphs",
			nil,
		},
		{
			"marker-only reply scrubs to empty",
			"⟦open-browser: https://theboring.name⟧",
			"",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}},
		},
		{
			"trailing marker leaves prose",
			"Sure — opening the docs now.\n⟦open-browser: https://theboring.name⟧",
			"Sure — opening the docs now.",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}},
		},
		{
			"mid-text marker leaves the two halves",
			"first half\n⟦open-browser: https://theboring.name⟧\nsecond half",
			"first half\nsecond half",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}},
		},
		{
			"paragraph marker leaves ONE paragraph break",
			"first half\n\n⟦open-browser: https://theboring.name⟧\n\nsecond half",
			"first half\n\nsecond half",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}},
		},
		{
			"two markers both land",
			"⟦open-browser: https://theboring.name⟧\n⟦open-browser: http://localhost:3000⟧",
			"",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}, {Kind: KindOpen, URL: "http://localhost:3000"}},
		},
		{
			"screenshot marker strips and kinds",
			"Rendering it now.\n⟦browser-screenshot: https://theboring.name/docs⟧",
			"Rendering it now.",
			[]Request{{Kind: KindShot, URL: "https://theboring.name/docs"}},
		},
		{
			"snapshot marker strips and kinds",
			"Reading it for you.\n⟦browser-snapshot: http://localhost:3000/api⟧",
			"Reading it for you.",
			[]Request{{Kind: KindSnap, URL: "http://localhost:3000/api"}},
		},
		{
			"all three kinds keep order of appearance",
			"⟦browser-snapshot: https://a.example⟧\n⟦open-browser: https://b.example⟧\n⟦browser-screenshot: https://c.example⟧",
			"",
			[]Request{{Kind: KindSnap, URL: "https://a.example"}, {Kind: KindOpen, URL: "https://b.example"}, {Kind: KindShot, URL: "https://c.example"}},
		},
		{
			"a mid-line marker is prose (the own-line rule)",
			"look at ⟦open-browser: https://theboring.name⟧ here",
			"look at ⟦open-browser: https://theboring.name⟧ here",
			nil,
		},
		{
			"a mid-line screenshot marker is prose too",
			"see ⟦browser-screenshot: https://theboring.name⟧ inline",
			"see ⟦browser-screenshot: https://theboring.name⟧ inline",
			nil,
		},
		{
			"an unterminated marker is prose",
			"⟦open-browser: https://theboring.name\nnext",
			"⟦open-browser: https://theboring.name\nnext",
			nil,
		},
		{
			"an unterminated snapshot marker is prose",
			"⟦browser-snapshot: https://theboring.name\nnext",
			"⟦browser-snapshot: https://theboring.name\nnext",
			nil,
		},
		{
			"an empty-URL marker is prose",
			"⟦open-browser:⟧",
			"⟦open-browser:⟧",
			nil,
		},
		{
			"an empty-URL screenshot marker is prose",
			"⟦browser-screenshot:⟧",
			"⟦browser-screenshot:⟧",
			nil,
		},
		{
			"leading/trailing whitespace on the marker line is fine",
			"prose\n  ⟦open-browser:   https://theboring.name  ⟧  \ntail",
			"prose\ntail",
			[]Request{{Kind: KindOpen, URL: "https://theboring.name"}},
		},
		{
			"whitespace on a snapshot marker line is fine",
			"prose\n\t⟦browser-snapshot:  https://theboring.name\t⟧\ntail",
			"prose\ntail",
			[]Request{{Kind: KindSnap, URL: "https://theboring.name"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reqs := Extract(c.in)
			if got != c.want {
				t.Fatalf("Extract text = %q, want %q", got, c.want)
			}
			if len(reqs) != len(c.reqs) {
				t.Fatalf("Extract reqs = %v, want %v", reqs, c.reqs)
			}
			for i, r := range reqs {
				if r != c.reqs[i] {
					t.Fatalf("Extract reqs[%d] = %+v, want %+v", i, r, c.reqs[i])
				}
			}
		})
	}
}

func TestExtractCapsRequestsPerReply(t *testing.T) {
	// the cap SPANS kinds: 4 mixed markers → the FIRST 3 in order of
	// appearance survive (the past-cap line still strips).
	in := "⟦open-browser: https://a.example⟧\n" +
		"⟦browser-screenshot: https://b.example⟧\n" +
		"⟦browser-snapshot: https://c.example⟧\n" +
		"⟦open-browser: https://d.example⟧"
	cleaned, reqs := Extract(in)
	if cleaned != "" {
		t.Fatalf("all marker lines strip, got %q", cleaned)
	}
	if len(reqs) != MaxRequestsPerReply {
		t.Fatalf("reqs capped at %d, got %v", MaxRequestsPerReply, reqs)
	}
	want := []Request{{Kind: KindOpen, URL: "https://a.example"}, {Kind: KindShot, URL: "https://b.example"}, {Kind: KindSnap, URL: "https://c.example"}}
	for i, r := range reqs {
		if r != want[i] {
			t.Fatalf("the cap keeps the FIRST requests in order: reqs[%d] = %+v, want %+v (all %v)", i, r, want[i], reqs)
		}
	}
}

// fakeSink is the bridge's fake backend (the emit half of the eventLog
// harness the backend packages use).
type fakeSink struct{ evs []state.Event }

func (s *fakeSink) emit(e state.Event) { s.evs = append(s.evs, e) }

func (s *fakeSink) browserEvents() []state.Event {
	var out []state.Event
	for _, e := range s.evs {
		switch e.Kind {
		case state.EvBrowserOpen, state.EvBrowserScreenshot, state.EvBrowserSnapshot:
			out = append(out, e)
		}
	}
	return out
}

func (s *fakeSink) browserOpens() []state.Event {
	var out []state.Event
	for _, e := range s.evs {
		if e.Kind == state.EvBrowserOpen {
			out = append(out, e)
		}
	}
	return out
}

func TestBridgeEmitsOneEventPerRequest(t *testing.T) {
	sink := &fakeSink{}
	br := &Bridge{Emit: sink.emit, Getenv: flagEnv(false)}
	decisions := br.RequestAll([]Request{
		{Kind: KindOpen, URL: "https://theboring.name"},  // allowed (https default)
		{Kind: KindOpen, URL: "http://localhost:3000"},   // allowed (loopback)
		{Kind: KindOpen, URL: "http://theboring.name"},   // refused (plain http, flag off)
		{Kind: KindOpen, URL: "file:///tmp/secret.html"}, // refused (scheme)
		{Kind: KindShot, URL: "https://theboring.name"},  // allowed screenshot
		{Kind: KindSnap, URL: "http://theboring.name"},   // refused snapshot
	})
	if len(decisions) != 6 {
		t.Fatalf("one decision per request, got %d", len(decisions))
	}
	for i, k := range []Kind{KindOpen, KindOpen, KindOpen, KindOpen, KindShot, KindSnap} {
		if decisions[i].Kind != k {
			t.Fatalf("decision %d keeps the request's kind: got %q, want %q", i, decisions[i].Kind, k)
		}
	}
	evs := sink.browserEvents()
	if len(evs) != 6 {
		t.Fatalf("one event per request, got %d: %+v", len(evs), evs)
	}
	// each directive kind rides its OWN event kind, in request order.
	for i, want := range []state.EventKind{
		state.EvBrowserOpen, state.EvBrowserOpen, state.EvBrowserOpen, state.EvBrowserOpen,
		state.EvBrowserScreenshot, state.EvBrowserSnapshot,
	} {
		if evs[i].Kind != want {
			t.Fatalf("event %d kind = %q, want %q (%+v)", i, evs[i].Kind, want, evs[i])
		}
	}
	// allowed events: URL on Text, verdict true, no reason.
	for i, want := range []string{"https://theboring.name", "http://localhost:3000"} {
		if evs[i].Text != want || !evs[i].BrowserOpenAllowed || evs[i].BrowserOpenReason != "" {
			t.Fatalf("allowed event %d mis-shaped: %+v", i, evs[i])
		}
	}
	if evs[4].Text != "https://theboring.name" || !evs[4].BrowserOpenAllowed || evs[4].BrowserOpenReason != "" {
		t.Fatalf("the screenshot event carries the allowed verdict: %+v", evs[4])
	}
	// refused events: verdict false + the agent/member-readable reason
	// (the SAME policy + reason text for every marker kind).
	const httpRefusal = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if evs[2].BrowserOpenAllowed || evs[2].BrowserOpenReason != httpRefusal {
		t.Fatalf("the http refusal must carry the exact reason: %+v", evs[2])
	}
	if evs[3].BrowserOpenAllowed || evs[3].BrowserOpenReason != "not an absolute http(s) URL" {
		t.Fatalf("the scheme refusal must carry its reason: %+v", evs[3])
	}
	if evs[5].BrowserOpenAllowed || evs[5].Text != "http://theboring.name" || evs[5].BrowserOpenReason != httpRefusal {
		t.Fatalf("the snapshot refusal carries the exact reason on its own kind: %+v", evs[5])
	}
}

func TestBridgeNilEmitIsSafe(t *testing.T) {
	br := &Bridge{Getenv: flagEnv(true)}
	if ds := br.RequestAll([]Request{{Kind: KindSnap, URL: "https://theboring.name"}}); len(ds) != 1 || !ds[0].Allowed {
		t.Fatalf("a nil sink still decides: %+v", ds)
	}
}

func TestScrubFallbackBubbles(t *testing.T) {
	sink := &fakeSink{}
	br := &Bridge{Emit: sink.emit, Getenv: flagEnv(false)}

	// prose + marker → the prose stays, no fallback note.
	if got := Scrub("opening it now.\n⟦open-browser: https://theboring.name⟧", br); got != "opening it now." {
		t.Fatalf("scrub with prose = %q", got)
	}
	// marker-only (allowed) → the open note.
	if got := Scrub("⟦open-browser: https://theboring.name⟧", br); got !=
		"[theboringfloor] open-browser: https://theboring.name" {
		t.Fatalf("marker-only fallback = %q", got)
	}
	// marker-only (refused) → the refusal note.
	if got := Scrub("⟦open-browser: http://theboring.name⟧", br); got !=
		"[theboringfloor] open-browser refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("refused marker-only fallback = %q", got)
	}
	// no markers → identity, and NO bridge traffic.
	before := len(sink.browserOpens())
	if got := Scrub("just prose", br); got != "just prose" {
		t.Fatalf("no-marker scrub = %q", got)
	}
	if len(sink.browserOpens()) != before {
		t.Fatal("a marker-less text must never emit")
	}
	// and the scrubs above emitted exactly 3 events (one per marker).
	if len(sink.browserOpens()) != 3 {
		t.Fatalf("bridge traffic across the scrubs = %d, want 3", len(sink.browserOpens()))
	}
}

func TestScrubFallbackNamesNewKinds(t *testing.T) {
	sink := &fakeSink{}
	br := &Bridge{Emit: sink.emit, Getenv: flagEnv(false)}

	// screenshot-only (allowed) → the kind-named note.
	if got := Scrub("⟦browser-screenshot: https://theboring.name⟧", br); got !=
		"[theboringfloor] browser-screenshot: https://theboring.name" {
		t.Fatalf("screenshot marker-only fallback = %q", got)
	}
	// snapshot-only (allowed) → the kind-named note.
	if got := Scrub("⟦browser-snapshot: https://theboring.name⟧", br); got !=
		"[theboringfloor] browser-snapshot: https://theboring.name" {
		t.Fatalf("snapshot marker-only fallback = %q", got)
	}
	// one of each kind, all allowed → kinds join with " · ".
	if got := Scrub("⟦open-browser: https://a.example⟧\n⟦browser-snapshot: https://b.example⟧\n⟦browser-screenshot: https://c.example⟧", br); got !=
		"[theboringfloor] open-browser: https://a.example · browser-screenshot: https://c.example · browser-snapshot: https://b.example" {
		t.Fatalf("mixed-kind marker-only fallback = %q", got)
	}
	// a refused screenshot rides its own kind label.
	if got := Scrub("⟦browser-screenshot: http://theboring.name⟧", br); got !=
		"[theboringfloor] browser-screenshot refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("refused screenshot fallback = %q", got)
	}
	// allowed + refused mix → the " · refused: " tail (open-kind contract).
	if got := Scrub("⟦open-browser: https://a.example⟧\n⟦browser-snapshot: http://theboring.name⟧", br); got !=
		"[theboringfloor] open-browser: https://a.example · refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("mixed allowed/refused fallback = %q", got)
	}
	// every scrub above stripped its markers (the notes are the WHOLE
	// remaining text) and emitted one event per marker (8 total).
	if evs := sink.browserEvents(); len(evs) != 8 {
		t.Fatalf("bridge traffic across the scrubs = %d events, want 8: %+v", len(evs), evs)
	}
}

func TestPromptPreambleTeachesTheContract(t *testing.T) {
	// the agent-facing instruction must carry the browser preference and
	// fallback rules, marker shapes, own-line rule, URL-policy flag, strip
	// contract, and mutating sibling's grammar + always-ask permission.
	for _, want := range []string{
		"Browser policy: prefer the office's built-in browser directives for every URL, including localhost and external pages.",
		"Use open-browser to show a page, browser-screenshot to capture it for the member, and browser-snapshot to read text/links yourself.",
		"Do not launch Chrome, Chromium, Playwright, Puppeteer, terminal-browser, or another browser process unless the member explicitly asks for an external browser or the built-in directive fails; if it fails, explain why before falling back.",
		"The member-facing slash command is /open <url>; agents use the directives below.",
		"localhost",
		"external pages",
		"Chrome",
		"Chromium",
		"Playwright",
		"Puppeteer",
		"terminal-browser",
		MarkerOpen + " URL" + MarkerClose,
		MarkerShot + " URL" + MarkerClose,
		MarkerSnap + " URL" + MarkerClose,
		MarkerAct + " URL | click: CSS-SELECTOR" + MarkerClose,
		MarkerAct + " URL | fill: CSS-SELECTOR = VALUE" + MarkerClose,
		MarkerAct + " URL | eval: JS-EXPRESSION" + MarkerClose,
		"ITS OWN line",
		AllowHTTPEnv + "=1",
		"strips the directive",
		"permission prompt ALWAYS asks first",
		"READ-ONLY",
		"MUTATES",
	} {
		if !strings.Contains(PromptPreamble, want) {
			t.Fatalf("PromptPreamble missing %q:\n%s", want, PromptPreamble)
		}
	}
}

// TestPromptPreambleByteContract — the browser-policy paragraph appears
// before directive syntax, and the entire first-turn preamble is a stable
// byte contract (backend tests build expected wire lines from it).
func TestPromptPreambleByteContract(t *testing.T) {
	const policyParagraph = "[theboringoffice harness — browser tool]\n" +
		"Browser policy: prefer the office's built-in browser directives for every URL, including localhost and external pages. " +
		"Use open-browser to show a page, browser-screenshot to capture it for the member, and browser-snapshot to read text/links yourself. " +
		"Do not launch Chrome, Chromium, Playwright, Puppeteer, terminal-browser, or another browser process unless the member explicitly asks for an external browser or the built-in directive fails; if it fails, explain why before falling back. " +
		"The member-facing slash command is /open <url>; agents use the directives below.\n"
	const openParagraph = "You can ask the office to open a web page in the member's in-app browser tab. " +
		"To open a page, emit this directive on ITS OWN line, at most once per reply:\n" +
		MarkerOpen + " URL" + MarkerClose + "\n" +
		"URL must be absolute: https:// for any host; http:// only for localhost, 127.0.0.1 or ::1 " +
		"(plain http to any other host is refused unless the member exports " + AllowHTTPEnv + "=1). " +
		"The office strips the directive from your visible reply and performs the open, so never quote " +
		"or explain the marker itself — just place it. Open a page only when it genuinely helps the " +
		"member (docs, dashboards, pull requests, a dev server you started)."
	if !strings.HasPrefix(PromptPreamble, policyParagraph) {
		t.Fatalf("the browser-policy paragraph must be the strict pre-directive prefix, got:\n%s", PromptPreamble)
	}
	if !strings.HasPrefix(PromptPreamble, policyParagraph+openParagraph) {
		t.Fatalf("the open-browser paragraph must follow the browser-policy paragraph byte-identically, got:\n%s", PromptPreamble)
	}
	const readOnlyBlock = "\nTwo read-only siblings (same own-line rule, same URL policy, at most one of each per reply, " +
		"3 browser directives total per reply):\n" +
		MarkerShot + " URL" + MarkerClose + " — render the page in the member's browser tab as an image " +
		"(kitty terminals) and save the PNG (the member sees the path).\n" +
		MarkerSnap + " URL" + MarkerClose + " — fetch the page's text + links back to YOU as a follow-up " +
		"message — use it to READ pages."
	if !strings.HasPrefix(PromptPreamble, policyParagraph+openParagraph+readOnlyBlock) {
		t.Fatalf("the policy + open paragraph + read-only block must stay byte-identical (a strict prefix), got:\n%s", PromptPreamble)
	}
	const actionBlock = "\nOne MUTATING sibling (same own-line rule, same URL policy, counts toward the 3-directive cap) — " +
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
	if !strings.HasSuffix(PromptPreamble, actionBlock) {
		t.Fatalf("the browser-action block must append verbatim, got:\n%s", PromptPreamble)
	}
	if PromptPreamble != policyParagraph+openParagraph+readOnlyBlock+actionBlock {
		t.Fatal("the preamble is EXACTLY the policy paragraph + open paragraph + read-only block + action block (nothing between, nothing after)")
	}
}

// TestExtractBrowserActionGrammar — the STRICT mutating-marker grammar:
// well-formed click/fill/eval markers parse their payload (selectors
// may carry spaces, fill values may carry spaces and '=', eval
// expressions any non-⟧ text), and EVERY malformed shape stays VISIBLE
// prose — never extracted, never acted on.
func TestExtractBrowserActionGrammar(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // the scrubbed text (== in when the marker is prose)
		reqs []Request
	}{
		{
			"click parses",
			"⟦browser-action: https://theboring.name | click: #buy⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://theboring.name", Op: "click", Sel: "#buy"}},
		},
		{
			"click selector with spaces (descendant combinator)",
			"⟦browser-action: https://theboring.name | click: .form-row .submit-btn⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://theboring.name", Op: "click", Sel: ".form-row .submit-btn"}},
		},
		{
			"click boundary whitespace trims",
			"prose\n  ⟦browser-action:   https://x.example   |   click:   #buy  ⟧  \ntail",
			"prose\ntail",
			[]Request{{Kind: KindAction, URL: "https://x.example", Op: "click", Sel: "#buy"}},
		},
		{
			"fill parses",
			"⟦browser-action: https://theboring.name | fill: #q = hello⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://theboring.name", Op: "fill", Sel: "#q", Arg: "hello"}},
		},
		{
			"fill value carries spaces and '='",
			"⟦browser-action: http://localhost:3000 | fill: #q = a = b c⟧",
			"",
			[]Request{{Kind: KindAction, URL: "http://localhost:3000", Op: "fill", Sel: "#q", Arg: "a = b c"}},
		},
		{
			"fill splits on the FIRST '='",
			"⟦browser-action: https://x | fill: #q=a=b⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://x", Op: "fill", Sel: "#q", Arg: "a=b"}},
		},
		{
			"eval parses any non-⟧ expression",
			"⟦browser-action: https://x | eval: document.querySelectorAll('a').length⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://x", Op: "eval", Arg: "document.querySelectorAll('a').length"}},
		},
		{
			"eval expression with spaces and pipes-in-strings",
			"⟦browser-action: https://x | eval: ({a: 1, b: 'x|y'})⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://x", Op: "eval", Arg: "({a: 1, b: 'x|y'})"}},
		},
		{
			"https deep path + query URL",
			"⟦browser-action: https://theboring.name/docs/keys?x=1#y | click: #a⟧",
			"",
			[]Request{{Kind: KindAction, URL: "https://theboring.name/docs/keys?x=1#y", Op: "click", Sel: "#a"}},
		},
		// -------- malformed: the marker NEVER extracts, stays visible prose
		{
			"unknown op is prose",
			"⟦browser-action: https://x | frobnicate: #a⟧",
			"⟦browser-action: https://x | frobnicate: #a⟧",
			nil,
		},
		{
			"click with an empty selector is prose",
			"⟦browser-action: https://x | click:⟧",
			"⟦browser-action: https://x | click:⟧",
			nil,
		},
		{
			"click with a whitespace-only selector is prose",
			"⟦browser-action: https://x | click:   ⟧",
			"⟦browser-action: https://x | click:   ⟧",
			nil,
		},
		{
			"fill with an empty value is prose",
			"⟦browser-action: https://x | fill: #q =⟧",
			"⟦browser-action: https://x | fill: #q =⟧",
			nil,
		},
		{
			"fill with an empty selector is prose",
			"⟦browser-action: https://x | fill: = val⟧",
			"⟦browser-action: https://x | fill: = val⟧",
			nil,
		},
		{
			"fill without '=' is prose",
			"⟦browser-action: https://x | fill: #q⟧",
			"⟦browser-action: https://x | fill: #q⟧",
			nil,
		},
		{
			"eval with an empty expression is prose",
			"⟦browser-action: https://x | eval:⟧",
			"⟦browser-action: https://x | eval:⟧",
			nil,
		},
		{
			"missing the pipe is prose",
			"⟦browser-action: https://x click: #a⟧",
			"⟦browser-action: https://x click: #a⟧",
			nil,
		},
		{
			"missing the URL is prose",
			"⟦browser-action: | click: #a⟧",
			"⟦browser-action: | click: #a⟧",
			nil,
		},
		{
			"a mid-line action marker is prose (the own-line rule)",
			"try ⟦browser-action: https://x | click: #a⟧ here",
			"try ⟦browser-action: https://x | click: #a⟧ here",
			nil,
		},
		{
			"trailing text after the close is prose",
			"⟦browser-action: https://x | click: #a⟧ extra",
			"⟦browser-action: https://x | click: #a⟧ extra",
			nil,
		},
		{
			"an unterminated action marker is prose",
			"⟦browser-action: https://x | click: #a\nnext",
			"⟦browser-action: https://x | click: #a\nnext",
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reqs := Extract(c.in)
			if got != c.want {
				t.Fatalf("Extract text = %q, want %q", got, c.want)
			}
			if len(reqs) != len(c.reqs) {
				t.Fatalf("Extract reqs = %+v, want %+v", reqs, c.reqs)
			}
			for i, r := range reqs {
				if r != c.reqs[i] {
					t.Fatalf("Extract reqs[%d] = %+v, want %+v", i, r, c.reqs[i])
				}
			}
		})
	}
}

// TestExtractCapsRequestsSpanningAction — the 3-directive cap SPANS the
// action kind too: 4 mixed markers → the FIRST 3 in order of
// appearance survive (the past-cap action line still strips).
func TestExtractCapsRequestsSpanningAction(t *testing.T) {
	in := "⟦browser-action: https://a.example | click: #one⟧\n" +
		"⟦open-browser: https://b.example⟧\n" +
		"⟦browser-snapshot: https://c.example⟧\n" +
		"⟦browser-action: https://d.example | eval: 1+1⟧"
	cleaned, reqs := Extract(in)
	if cleaned != "" {
		t.Fatalf("all marker lines strip, got %q", cleaned)
	}
	if len(reqs) != MaxRequestsPerReply {
		t.Fatalf("reqs capped at %d, got %+v", MaxRequestsPerReply, reqs)
	}
	want := []Request{
		{Kind: KindAction, URL: "https://a.example", Op: "click", Sel: "#one"},
		{Kind: KindOpen, URL: "https://b.example"},
		{Kind: KindSnap, URL: "https://c.example"},
	}
	for i, r := range reqs {
		if r != want[i] {
			t.Fatalf("the cap keeps the FIRST requests in order: reqs[%d] = %+v, want %+v (all %+v)", i, r, want[i], reqs)
		}
	}
}

// TestBridgeEmitsActionEvent — a browser-action request lands as ONE
// EvBrowserAction carrying the policy verdict AND the parsed action
// payload (op/sel/arg); a refused action carries the exact reason on
// the same kind (the app posts the reason row, NO modal).
func TestBridgeEmitsActionEvent(t *testing.T) {
	sink := &fakeSink{}
	br := &Bridge{Emit: sink.emit, Getenv: flagEnv(false)}
	decisions := br.RequestAll([]Request{
		{Kind: KindAction, URL: "http://localhost:3000", Op: "click", Sel: "#buy"},           // allowed (loopback) — the MODAL still gates it app-side
		{Kind: KindAction, URL: "https://theboring.name", Op: "fill", Sel: "#q", Arg: "x=1"}, // allowed (https)
		{Kind: KindAction, URL: "http://theboring.name", Op: "eval", Arg: "1+1"},             // refused (plain http, flag off)
	})
	if len(decisions) != 3 {
		t.Fatalf("one decision per request, got %d", len(decisions))
	}
	var acts []state.Event
	for _, e := range sink.evs {
		if e.Kind == state.EvBrowserAction {
			acts = append(acts, e)
		}
	}
	if len(acts) != 3 {
		t.Fatalf("one EvBrowserAction per request, got %+v", sink.evs)
	}
	// allowed click: the verdict true, no reason, the payload rides.
	if acts[0].Text != "http://localhost:3000" || !acts[0].BrowserOpenAllowed || acts[0].BrowserOpenReason != "" ||
		acts[0].BrowserActionOp != "click" || acts[0].BrowserActionSel != "#buy" || acts[0].BrowserActionArg != "" {
		t.Fatalf("the allowed click event is mis-shaped: %+v", acts[0])
	}
	// allowed fill: the value rides Arg.
	if acts[1].BrowserActionOp != "fill" || acts[1].BrowserActionSel != "#q" || acts[1].BrowserActionArg != "x=1" {
		t.Fatalf("the allowed fill event must carry sel+arg: %+v", acts[1])
	}
	// refused eval: the exact policy reason, the payload STILL rides
	// (the app's red row names what was refused).
	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if acts[2].BrowserOpenAllowed || acts[2].BrowserOpenReason != reason ||
		acts[2].BrowserActionOp != "eval" || acts[2].BrowserActionArg != "1+1" {
		t.Fatalf("the refused eval event must carry the exact reason + payload: %+v", acts[2])
	}
	for i, d := range decisions {
		if d.Kind != KindAction {
			t.Fatalf("decision %d keeps the action kind: %+v", i, d)
		}
	}
}

// TestScrubFallbackNamesAction — an action-ONLY reply degrades to the
// kind-named office note (allowed) / the kind-named refusal note
// (refused), exactly like its read-only siblings.
func TestScrubFallbackNamesAction(t *testing.T) {
	sink := &fakeSink{}
	br := &Bridge{Emit: sink.emit, Getenv: flagEnv(false)}

	if got := Scrub("⟦browser-action: https://theboring.name | click: #buy⟧", br); got !=
		"[theboringfloor] browser-action: https://theboring.name" {
		t.Fatalf("action-only fallback = %q", got)
	}
	if got := Scrub("⟦browser-action: http://theboring.name | click: #buy⟧", br); got !=
		"[theboringfloor] browser-action refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("refused action-only fallback = %q", got)
	}
	// prose + action marker → the prose stays, no fallback note.
	if got := Scrub("clicking it now.\n⟦browser-action: https://theboring.name | click: #buy⟧", br); got != "clicking it now." {
		t.Fatalf("scrub with prose = %q", got)
	}
	// a MALFORMED action marker never extracts: it stays visible prose,
	// untouched, and fires NO bridge traffic.
	before := len(sink.evs)
	in := "⟦browser-action: https://theboring.name | click:⟧"
	if got := Scrub(in, br); got != in {
		t.Fatalf("a malformed action marker stays visible verbatim, got %q", got)
	}
	if len(sink.evs) != before {
		t.Fatal("a malformed action marker must never emit")
	}
}
