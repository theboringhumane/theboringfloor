// live_test.go — the REAL engine against the member's REAL Chrome:
// gated behind THEBORINGOFFICE_LIVE_CHROME=1 AND a successful Available
// probe, so CI/dev machines without the flag or a browser never spawn a
// process. The fixture rides a loopback httptest server (127.0.0.1 is
// always policy-allowed) — this also proves the policy gate passes real
// loopback navigation before launch.
package headless

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// liveServer — the fixture over loopback http; skips unless both gates open.
func liveServer(t *testing.T) *httptest.Server {
	t.Helper()
	if os.Getenv("THEBORINGOFFICE_LIVE_CHROME") != "1" {
		t.Skip("set THEBORINGOFFICE_LIVE_CHROME=1 to run the live engine tests")
	}
	path, ok := Available()
	if !ok {
		t.Skip("no Chrome-family browser on this machine")
	}
	t.Logf("discovery: Available() = (%q, %v)", path, ok)
	srv := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	t.Cleanup(srv.Close)
	return srv
}

func TestLiveScreenshot(t *testing.T) {
	srv := liveServer(t)
	rawurl := srv.URL + "/fixture.html"

	res, err := Screenshot(context.Background(), rawurl, 320, 200)
	if err != nil {
		t.Fatalf("Screenshot: %v", err)
	}
	if len(res.PNG) == 0 {
		t.Fatal("empty PNG")
	}
	if res.URL != rawurl {
		t.Errorf("final URL = %q, want %q", res.URL, rawurl)
	}
	if res.Title != "Boring Fixture — Headless Engine" {
		t.Errorf("title = %q", res.Title)
	}
	img, err := png.Decode(bytes.NewReader(res.PNG))
	if err != nil {
		t.Fatalf("PNG undecodable: %v", err)
	}
	// deviceScaleFactor 2: 320x200 CSS px → 640x400 device px.
	if got := img.Bounds(); got.Dx() != 640 || got.Dy() != 400 {
		t.Fatalf("PNG dims = %dx%d, want 640x400 (requested * dpr 2)", got.Dx(), got.Dy())
	}
	t.Logf("screenshot: %d PNG bytes, dims %dx%d, title %q, url %s",
		len(res.PNG), img.Bounds().Dx(), img.Bounds().Dy(), res.Title, res.URL)
}

func TestLiveSnapshot(t *testing.T) {
	srv := liveServer(t)
	rawurl := srv.URL + "/fixture.html"

	res, err := Snapshot(context.Background(), rawurl, 100_000)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if res.URL != rawurl {
		t.Errorf("final URL = %q, want %q", res.URL, rawurl)
	}
	if res.Title != "Boring Fixture — Headless Engine" {
		t.Errorf("title = %q", res.Title)
	}
	if !strings.Contains(res.Text, "Hello boring world") {
		t.Errorf("text missing fixture sentence; head: %q", truncate(res.Text, 120))
	}

	// Links: example.com deduped to one, the relative href resolved
	// absolute against the page URL, javascript:/mailto: gone, <= 50.
	var example, local int
	for _, l := range res.Links {
		switch {
		case l.URL == "https://example.com/":
			example++
		case l.URL == srv.URL+"/local":
			local++
		}
		low := strings.ToLower(l.URL)
		if strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "mailto:") {
			t.Errorf("scheme not filtered: %+v", l)
		}
	}
	if example != 1 {
		t.Errorf("example.com links = %d, want 1 (deduped); all: %+v", example, res.Links)
	}
	if local != 1 {
		t.Errorf("local absolute link = %d, want 1; all: %+v", local, res.Links)
	}
	if len(res.Links) > 50 {
		t.Errorf("links = %d, want <= 50", len(res.Links))
	}
	t.Logf("snapshot: text head %q · links %v", truncate(res.Text, 80), res.Links)
}

func TestLiveSnapshotCap(t *testing.T) {
	srv := liveServer(t)
	res, err := Snapshot(context.Background(), srv.URL+"/fixture.html", 40)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if n := len([]rune(res.Text)); n > 40 {
		t.Fatalf("text = %d runes, want <= 40 (%q)", n, res.Text)
	}
}

// TestLiveNavigationRefused — a loopback port with nothing listening:
// policy ALLOWS it, chrome launches, and the failure comes back as a
// wrapped navigation error (never a timeout, never a PolicyError).
func TestLiveNavigationRefused(t *testing.T) {
	liveServer(t)
	_, err := Screenshot(context.Background(), "http://127.0.0.1:1/", 64, 64)
	if err == nil {
		t.Fatal("expected navigation failure to a dead port")
	}
	var pe *PolicyError
	if errors.As(err, &pe) {
		t.Fatalf("dead port must not be a policy refusal: %v", pe)
	}
	if !strings.Contains(err.Error(), "navigation failed") {
		t.Fatalf("want navigation-failed wrap, got: %v", err)
	}
	t.Logf("dead-port error: %v", err)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
