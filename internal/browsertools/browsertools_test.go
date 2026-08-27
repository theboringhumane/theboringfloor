// browsertools_test.go — the browser tool's three contracts:
//
//	policy (Decide) — localhost always, https by default, plain http
//	     non-localhost only under THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1,
//	     every other scheme refused, each refusal carrying the exact
//	     member-facing reason;
//	protocol (Extract) — whole-line markers strip cleanly (no ghost
//	     blank lines), mid-line/unterminated markers are prose, the
//	     per-reply cap holds;
//	bridge (RequestAll/Scrub) — the fake-emit sink sees one correctly
//	     shaped EvBrowserOpen per request, and a marker-only reply
//	     degrades to the one-line office note (never a blank pin).
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
		urls []string
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
			[]string{"https://theboring.name"},
		},
		{
			"trailing marker leaves prose",
			"Sure — opening the docs now.\n⟦open-browser: https://theboring.name⟧",
			"Sure — opening the docs now.",
			[]string{"https://theboring.name"},
		},
		{
			"mid-text marker leaves the two halves",
			"first half\n⟦open-browser: https://theboring.name⟧\nsecond half",
			"first half\nsecond half",
			[]string{"https://theboring.name"},
		},
		{
			"paragraph marker leaves ONE paragraph break",
			"first half\n\n⟦open-browser: https://theboring.name⟧\n\nsecond half",
			"first half\n\nsecond half",
			[]string{"https://theboring.name"},
		},
		{
			"two markers both land",
			"⟦open-browser: https://theboring.name⟧\n⟦open-browser: http://localhost:3000⟧",
			"",
			[]string{"https://theboring.name", "http://localhost:3000"},
		},
		{
			"a mid-line marker is prose (the own-line rule)",
			"look at ⟦open-browser: https://theboring.name⟧ here",
			"look at ⟦open-browser: https://theboring.name⟧ here",
			nil,
		},
		{
			"an unterminated marker is prose",
			"⟦open-browser: https://theboring.name\nnext",
			"⟦open-browser: https://theboring.name\nnext",
			nil,
		},
		{
			"an empty-URL marker is prose",
			"⟦open-browser:⟧",
			"⟦open-browser:⟧",
			nil,
		},
		{
			"leading/trailing whitespace on the marker line is fine",
			"prose\n  ⟦open-browser:   https://theboring.name  ⟧  \ntail",
			"prose\ntail",
			[]string{"https://theboring.name"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, urls := Extract(c.in)
			if got != c.want {
				t.Fatalf("Extract text = %q, want %q", got, c.want)
			}
			if len(urls) != len(c.urls) {
				t.Fatalf("Extract urls = %v, want %v", urls, c.urls)
			}
			for i, u := range urls {
				if u != c.urls[i] {
					t.Fatalf("Extract urls[%d] = %q, want %q", i, u, c.urls[i])
				}
			}
		})
	}
}

func TestExtractCapsRequestsPerReply(t *testing.T) {
	in := "⟦open-browser: https://a.example⟧\n" +
		"⟦open-browser: https://b.example⟧\n" +
		"⟦open-browser: https://c.example⟧\n" +
		"⟦open-browser: https://d.example⟧"
	cleaned, urls := Extract(in)
	if cleaned != "" {
		t.Fatalf("all marker lines strip, got %q", cleaned)
	}
	if len(urls) != MaxRequestsPerReply {
		t.Fatalf("urls capped at %d, got %v", MaxRequestsPerReply, urls)
	}
	if urls[0] != "https://a.example" || urls[2] != "https://c.example" {
		t.Fatalf("the cap keeps the FIRST requests in order, got %v", urls)
	}
}

// fakeSink is the bridge's fake backend (the emit half of the eventLog
// harness the backend packages use).
type fakeSink struct{ evs []state.Event }

func (s *fakeSink) emit(e state.Event) { s.evs = append(s.evs, e) }

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
	decisions := br.RequestAll([]string{
		"https://theboring.name",  // allowed (https default)
		"http://localhost:3000",   // allowed (loopback)
		"http://theboring.name",   // refused (plain http, flag off)
		"file:///tmp/secret.html", // refused (scheme)
	})
	if len(decisions) != 4 {
		t.Fatalf("one decision per request, got %d", len(decisions))
	}
	evs := sink.browserOpens()
	if len(evs) != 4 {
		t.Fatalf("one EvBrowserOpen per request, got %d: %+v", len(evs), evs)
	}
	// allowed events: URL on Text, verdict true, no reason.
	for i, want := range []string{"https://theboring.name", "http://localhost:3000"} {
		if evs[i].Text != want || !evs[i].BrowserOpenAllowed || evs[i].BrowserOpenReason != "" {
			t.Fatalf("allowed event %d mis-shaped: %+v", i, evs[i])
		}
	}
	// refused events: verdict false + the agent/member-readable reason.
	if evs[2].BrowserOpenAllowed || evs[2].BrowserOpenReason !=
		"plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages" {
		t.Fatalf("the http refusal must carry the exact reason: %+v", evs[2])
	}
	if evs[3].BrowserOpenAllowed || evs[3].BrowserOpenReason != "not an absolute http(s) URL" {
		t.Fatalf("the scheme refusal must carry its reason: %+v", evs[3])
	}
}

func TestBridgeNilEmitIsSafe(t *testing.T) {
	br := &Bridge{Getenv: flagEnv(true)}
	if ds := br.RequestAll([]string{"https://theboring.name"}); len(ds) != 1 || !ds[0].Allowed {
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

func TestPromptPreambleTeachesTheContract(t *testing.T) {
	// the agent-facing instruction must carry the marker shape, the
	// own-line rule, the policy flag, and the strip contract.
	for _, want := range []string{
		MarkerOpen + " URL" + MarkerClose,
		"ITS OWN line",
		AllowHTTPEnv + "=1",
		"strips the directive",
	} {
		if !strings.Contains(PromptPreamble, want) {
			t.Fatalf("PromptPreamble missing %q:\n%s", want, PromptPreamble)
		}
	}
}
