// live_test.go — the REAL executor against the member's REAL Chrome:
// gated behind THEBORINGOFFICE_LIVE_CHROME=1 AND a successful
// headless.Available probe, so CI/dev machines without the flag or a
// browser never spawn a process. The fixture rides a loopback httptest
// server (127.0.0.1 is always policy-allowed) — this also proves the
// policy gate passes real loopback navigation before launch.
package action

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/headless"
)

// liveServer — the fixture over loopback http; skips unless both gates open.
func liveServer(t *testing.T) *httptest.Server {
	t.Helper()
	if os.Getenv("THEBORINGOFFICE_LIVE_CHROME") != "1" {
		t.Skip("set THEBORINGOFFICE_LIVE_CHROME=1 to run the live action tests")
	}
	path, ok := headless.Available()
	if !ok {
		t.Skip("no Chrome-family browser on this machine")
	}
	t.Logf("discovery: headless.Available() = (%q, %v)", path, ok)
	srv := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(srv.Close)
	return srv
}

// TestLiveClick — the click LANDS: #buy's handler navigates (hash), so
// the final URL itself proves the click; the result text is the
// contract's "clicked <sel>".
func TestLiveClick(t *testing.T) {
	srv := liveServer(t)
	res, err := NavigateAndAct(context.Background(), srv.URL+"/fixture.html", Action{Op: OpClick, Sel: "#buy"})
	if err != nil {
		t.Fatalf("NavigateAndAct click: %v", err)
	}
	if res.Text != "clicked #buy" {
		t.Fatalf("click result = %q, want %q", res.Text, "clicked #buy")
	}
	if !strings.HasSuffix(res.URL, "/fixture.html#clicked") {
		t.Fatalf("the click must navigate (final URL carries #clicked), got %q", res.URL)
	}
	t.Logf("click: %q → final %s", res.Text, res.URL)
}

// TestLiveFill — SetValue finds the input and sets its value: no error,
// the contract's "filled <sel>" result, and (no navigation) the final
// URL stays the requested one.
func TestLiveFill(t *testing.T) {
	srv := liveServer(t)
	rawurl := srv.URL + "/fixture.html"
	res, err := NavigateAndAct(context.Background(), rawurl, Action{Op: OpFill, Sel: "#name", Arg: "Ada Lovelace"})
	if err != nil {
		t.Fatalf("NavigateAndAct fill: %v", err)
	}
	if res.Text != "filled #name" {
		t.Fatalf("fill result = %q, want %q", res.Text, "filled #name")
	}
	if res.URL != rawurl {
		t.Fatalf("a fill never navigates: final URL = %q, want %q", res.URL, rawurl)
	}
	t.Logf("fill: %q → final %s", res.Text, res.URL)
}

// TestLiveEval — the JS result comes back JSON-stringified: a number,
// a (quoted) string, and an object; undefined reads "null"; a throwing
// expression is the eval failure class.
func TestLiveEval(t *testing.T) {
	srv := liveServer(t)
	rawurl := srv.URL + "/fixture.html"

	res, err := NavigateAndAct(context.Background(), rawurl, Action{Op: OpEval, Arg: "window.answer"})
	if err != nil {
		t.Fatalf("eval number: %v", err)
	}
	if res.Text != "42" {
		t.Fatalf("eval window.answer = %q, want %q", res.Text, "42")
	}

	res, err = NavigateAndAct(context.Background(), rawurl, Action{Op: OpEval, Arg: "document.title"})
	if err != nil {
		t.Fatalf("eval string: %v", err)
	}
	// V8's JSON serialization escapes non-ASCII (the em-dash arrives as
	// the SIX literal runes \u2014) — the engine carries it VERBATIM
	// (valid, parseable JSON).
	if res.Text != `"Boring Fixture \u2014 Browser Action"` {
		t.Fatalf("eval document.title = %q (JSON-quoted string)", res.Text)
	}

	res, err = NavigateAndAct(context.Background(), rawurl, Action{Op: OpEval,
		Arg: "({href: location.pathname, n: document.querySelectorAll('input').length})"})
	if err != nil {
		t.Fatalf("eval object: %v", err)
	}
	if !strings.Contains(res.Text, `"href":"/fixture.html"`) || !strings.Contains(res.Text, `"n":1`) {
		t.Fatalf("eval object = %q", res.Text)
	}

	res, err = NavigateAndAct(context.Background(), rawurl, Action{Op: OpEval, Arg: "undefined"})
	if err != nil {
		t.Fatalf("eval undefined: %v", err)
	}
	if res.Text != "null" {
		t.Fatalf("eval undefined = %q, want \"null\"", res.Text)
	}

	_, err = NavigateAndAct(context.Background(), rawurl, Action{Op: OpEval, Arg: "(() => { throw new Error('bang') })()"})
	if err == nil || !strings.Contains(err.Error(), "eval failed") || !strings.Contains(err.Error(), "bang") {
		t.Fatalf("a throwing expression is the eval failure class carrying the JS error, got %v", err)
	}
	t.Logf("eval: 42 · quoted title · object · undefined→null · throw→%v", err)
}

// TestLiveSelectorNotFound — a click on a selector that never appears
// burns the 20s budget and comes back as the SELECTOR class (never the
// navigation class, never a bare timeout), with errors.Is(DeadlineExceeded).
// Skipped in -short mode: it spends the full budget by design.
func TestLiveSelectorNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("spends the 20s budget by design")
	}
	srv := liveServer(t)
	_, err := NavigateAndAct(context.Background(), srv.URL+"/fixture.html", Action{Op: OpClick, Sel: "#never-there"})
	if err == nil {
		t.Fatal("a never-matching selector must fail")
	}
	if !strings.Contains(err.Error(), `selector "#never-there" did not match a visible node`) {
		t.Fatalf("want the selector class, got: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the selector class still unwraps to DeadlineExceeded: %v", err)
	}
	t.Logf("selector-not-found: %v", err)
}

// TestLiveNavigationRefused — a loopback port with nothing listening:
// policy ALLOWS it, chrome launches, and the failure comes back as the
// navigation class (never the selector class, never a PolicyError).
func TestLiveNavigationRefused(t *testing.T) {
	liveServer(t)
	_, err := NavigateAndAct(context.Background(), "http://127.0.0.1:1/", Action{Op: OpClick, Sel: "#buy"})
	if err == nil {
		t.Fatal("expected navigation failure to a dead port")
	}
	if !strings.Contains(err.Error(), "navigation failed") {
		t.Fatalf("want the navigation class, got: %v", err)
	}
	t.Logf("dead-port error: %v", err)
}
