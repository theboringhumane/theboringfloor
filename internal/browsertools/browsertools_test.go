// browsertools_test.go — the browser tool's contracts:
//
//	policy (Decide) — localhost always, https by default, plain http
//	     non-localhost only under THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1,
//	     every other scheme refused, each refusal carrying the exact
//	     member-facing reason (ONE policy for all three marker kinds);
//	protocol (Extract) — whole-line markers of ALL THREE kinds strip
//	     cleanly (no ghost blank lines), mid-line/unterminated markers
//	     are prose, and the per-reply cap spans kinds in order of
//	     appearance;
//	bridge (RequestAll/Scrub) — the fake-emit sink sees one correctly
//	     shaped EvBrowserOpen/EvBrowserScreenshot/EvBrowserSnapshot per
//	     request, and a marker-only reply degrades to the one-line
//	     office note (never a blank pin);
//	preamble — the agent-facing text keeps the open-browser paragraph
//	     byte-identical and carries the new directives verbatim.
package browsertools

import (
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/state"
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
			[]Request{{KindOpen, "https://theboring.name"}},
		},
		{
			"trailing marker leaves prose",
			"Sure — opening the docs now.\n⟦open-browser: https://theboring.name⟧",
			"Sure — opening the docs now.",
			[]Request{{KindOpen, "https://theboring.name"}},
		},
		{
			"mid-text marker leaves the two halves",
			"first half\n⟦open-browser: https://theboring.name⟧\nsecond half",
			"first half\nsecond half",
			[]Request{{KindOpen, "https://theboring.name"}},
		},
		{
			"paragraph marker leaves ONE paragraph break",
			"first half\n\n⟦open-browser: https://theboring.name⟧\n\nsecond half",
			"first half\n\nsecond half",
			[]Request{{KindOpen, "https://theboring.name"}},
		},
		{
			"two markers both land",
			"⟦open-browser: https://theboring.name⟧\n⟦open-browser: http://localhost:3000⟧",
			"",
			[]Request{{KindOpen, "https://theboring.name"}, {KindOpen, "http://localhost:3000"}},
		},
		{
			"screenshot marker strips and kinds",
			"Rendering it now.\n⟦browser-screenshot: https://theboring.name/docs⟧",
			"Rendering it now.",
			[]Request{{KindShot, "https://theboring.name/docs"}},
		},
		{
			"snapshot marker strips and kinds",
			"Reading it for you.\n⟦browser-snapshot: http://localhost:3000/api⟧",
			"Reading it for you.",
			[]Request{{KindSnap, "http://localhost:3000/api"}},
		},
		{
			"all three kinds keep order of appearance",
			"⟦browser-snapshot: https://a.example⟧\n⟦open-browser: https://b.example⟧\n⟦browser-screenshot: https://c.example⟧",
			"",
			[]Request{{KindSnap, "https://a.example"}, {KindOpen, "https://b.example"}, {KindShot, "https://c.example"}},
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
			[]Request{{KindOpen, "https://theboring.name"}},
		},
		{
			"whitespace on a snapshot marker line is fine",
			"prose\n\t⟦browser-snapshot:  https://theboring.name\t⟧\ntail",
			"prose\ntail",
			[]Request{{KindSnap, "https://theboring.name"}},
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
	want := []Request{{KindOpen, "https://a.example"}, {KindShot, "https://b.example"}, {KindSnap, "https://c.example"}}
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
		{KindOpen, "https://theboring.name"},  // allowed (https default)
		{KindOpen, "http://localhost:3000"},   // allowed (loopback)
		{KindOpen, "http://theboring.name"},   // refused (plain http, flag off)
		{KindOpen, "file:///tmp/secret.html"}, // refused (scheme)
		{KindShot, "https://theboring.name"},  // allowed screenshot
		{KindSnap, "http://theboring.name"},   // refused snapshot
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
	if ds := br.RequestAll([]Request{{KindSnap, "https://theboring.name"}}); len(ds) != 1 || !ds[0].Allowed {
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
		"[theboringoffice] open-browser: https://theboring.name" {
		t.Fatalf("marker-only fallback = %q", got)
	}
	// marker-only (refused) → the refusal note.
	if got := Scrub("⟦open-browser: http://theboring.name⟧", br); got !=
		"[theboringoffice] open-browser refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
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
		"[theboringoffice] browser-screenshot: https://theboring.name" {
		t.Fatalf("screenshot marker-only fallback = %q", got)
	}
	// snapshot-only (allowed) → the kind-named note.
	if got := Scrub("⟦browser-snapshot: https://theboring.name⟧", br); got !=
		"[theboringoffice] browser-snapshot: https://theboring.name" {
		t.Fatalf("snapshot marker-only fallback = %q", got)
	}
	// one of each kind, all allowed → kinds join with " · ".
	if got := Scrub("⟦open-browser: https://a.example⟧\n⟦browser-snapshot: https://b.example⟧\n⟦browser-screenshot: https://c.example⟧", br); got !=
		"[theboringoffice] open-browser: https://a.example · browser-screenshot: https://c.example · browser-snapshot: https://b.example" {
		t.Fatalf("mixed-kind marker-only fallback = %q", got)
	}
	// a refused screenshot rides its own kind label.
	if got := Scrub("⟦browser-screenshot: http://theboring.name⟧", br); got !=
		"[theboringoffice] browser-screenshot refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("refused screenshot fallback = %q", got)
	}
	// allowed + refused mix → the " · refused: " tail (open-kind contract).
	if got := Scrub("⟦open-browser: https://a.example⟧\n⟦browser-snapshot: http://theboring.name⟧", br); got !=
		"[theboringoffice] open-browser: https://a.example · refused: plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("mixed allowed/refused fallback = %q", got)
	}
	// every scrub above stripped its markers (the notes are the WHOLE
	// remaining text) and emitted one event per marker (8 total).
	if evs := sink.browserEvents(); len(evs) != 8 {
		t.Fatalf("bridge traffic across the scrubs = %d events, want 8: %+v", len(evs), evs)
	}
}

func TestPromptPreambleTeachesTheContract(t *testing.T) {
	// the agent-facing instruction must carry the marker shapes, the
	// own-line rule, the policy flag, and the strip contract.
	for _, want := range []string{
		MarkerOpen + " URL" + MarkerClose,
		MarkerShot + " URL" + MarkerClose,
		MarkerSnap + " URL" + MarkerClose,
		"ITS OWN line",
		AllowHTTPEnv + "=1",
		"strips the directive",
	} {
		if !strings.Contains(PromptPreamble, want) {
			t.Fatalf("PromptPreamble missing %q:\n%s", want, PromptPreamble)
		}
	}
}

// TestPromptPreambleByteContract — the open-browser paragraph is a
// STABLE CONTRACT (backend tests build expected wire lines from it):
// it must stay byte-identical, and the new capability lines append
// after it verbatim.
func TestPromptPreambleByteContract(t *testing.T) {
	const openParagraph = "[theboringoffice harness — browser tool]\n" +
		"You can ask the office to open a web page in the member's in-app browser tab. " +
		"To open a page, emit this directive on ITS OWN line, at most once per reply:\n" +
		MarkerOpen + " URL" + MarkerClose + "\n" +
		"URL must be absolute: https:// for any host; http:// only for localhost, 127.0.0.1 or ::1 " +
		"(plain http to any other host is refused unless the member exports " + AllowHTTPEnv + "=1). " +
		"The office strips the directive from your visible reply and performs the open, so never quote " +
		"or explain the marker itself — just place it. Open a page only when it genuinely helps the " +
		"member (docs, dashboards, pull requests, a dev server you started)."
	if !strings.HasPrefix(PromptPreamble, openParagraph) {
		t.Fatalf("the open-browser paragraph must stay byte-identical (it is a strict prefix), got:\n%s", PromptPreamble)
	}
	const newLines = "\nTwo read-only siblings (same own-line rule, same URL policy, at most one of each per reply, " +
		"3 browser directives total per reply):\n" +
		MarkerShot + " URL" + MarkerClose + " — render the page in the member's browser tab as an image " +
		"(kitty terminals) and save the PNG (the member sees the path).\n" +
		MarkerSnap + " URL" + MarkerClose + " — fetch the page's text + links back to YOU as a follow-up " +
		"message — use it to READ pages."
	if !strings.HasSuffix(PromptPreamble, newLines) {
		t.Fatalf("the screenshot/snapshot lines must append verbatim, got:\n%s", PromptPreamble)
	}
	if PromptPreamble != openParagraph+newLines {
		t.Fatal("the preamble is EXACTLY the open paragraph + the new lines (nothing between, nothing after)")
	}
}
