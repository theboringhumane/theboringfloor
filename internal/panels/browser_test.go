// browser_test.go — the browser pane's RENDER half: the fixture parses
// into the exact row set (title line, bold headings, [n]-indexed link
// rows, bullets, " │ " table rows, the 🖼 chip, verbatim code rows, and
// stripped script/style/meta/comment noise), SetSize clips + reflows, and
// pgup/pgdn move the scroll offset. FetchPage's source matrix rides here
// too: file:// + bare paths, the content sniff's dim fallback, the
// localhost whitelist, the outbound-HTTP env gate, and the 404 wording.
package panels

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/cellmetrics"
	"github.com/theboringhumane/theboringfloor/internal/headless"
)

// TestMain pins the headless engine seam HERMETIC for the whole panels
// suite (no live chrome in unit tests): the package-wide default verdict
// is chrome-missing — deterministic on every host, and cheap (the probe
// gates before any render). The shot suites swap their own fake per test
// (SetHeadlessForShot's restore).
func TestMain(m *testing.M) {
	defer SetHeadlessForShot(func() (string, bool) { return "", false },
		func(context.Context, string, int, int) (*headless.Result, error) {
			return nil, headless.ErrChromeNotFound
		})()
	os.Exit(m.Run())
}

// browserKey mirrors gitpanel_test's gitKey: string → tea.KeyPressMsg.
func browserKey(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "up":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	case "down":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "pgup":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp})
	case "pgdown":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown})
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg(tea.Key{Code: r, Text: s})
	}
}

// fixturePage loads the shared testdata fixture through the REAL FetchPage
// (bare-path source — cwd is internal/panels under `go test`).
func fixturePage(t *testing.T) *Page {
	t.Helper()
	p, err := FetchPage("testdata/fixture.html")
	if err != nil {
		t.Fatalf("FetchPage(testdata/fixture.html): %v", err)
	}
	return p
}

// strippedRows renders the fixture page at width w and returns the raw
// (style-free) row texts.
func strippedRows(t *testing.T, p *Page, w int) []string {
	t.Helper()
	var out []string
	for _, ro := range p.renderRows(w) {
		out = append(out, ansi.Strip(ro.text))
	}
	return out
}

func TestBrowserFixtureRows(t *testing.T) {
	p := fixturePage(t)
	if p.Unsupported != "" {
		t.Fatalf("fixture must parse as HTML, got unsupported %q", p.Unsupported)
	}
	if p.Title != "Fixture Gazette" {
		t.Fatalf("title = %q, want %q", p.Title, "Fixture Gazette")
	}
	rows := strippedRows(t, p, 78)
	joined := "\n" + strings.Join(rows, "\n") + "\n"

	// the ordered contract: title line, h1, intro, h2, the three [n] link
	// rows, the bullet run, the table, the chip, the code rows, the tail.
	mustContain := []string{
		"\nFixture Gazette\n",
		"\nThe Fixture Gazette\n",
		"\nNews desk\n",
		"link alpha [1]",
		"link beta [2]",
		"link gamma [3]",
		"\n• shelf item one\n",
		"\n• shelf item twelve\n",
		"agent │ role",
		"tekton-1 │ developer",
		"🖼 fixture diagram",
		"go build ./...",
		"go test ./internal/panels/ -run Browser",
		"tail-marker — the pgdn proof scrolls this exact row into view.",
	}
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Fatalf("rendered rows missing %q:\n%s", want, joined)
		}
	}
	// stripped noise: the script body, the style body, the comment, the
	// meta attr — none may surface.
	for _, noise := range []string{"window.tracker", "color: red", "renderer must strip", "noindex"} {
		if strings.Contains(joined, noise) {
			t.Fatalf("stripped noise %q leaked into the rows", noise)
		}
	}
	// link side map: stable order, exactly the three anchors.
	if len(p.Links) != 3 {
		t.Fatalf("links = %d, want 3", len(p.Links))
	}
	for i, tail := range []string{"/alpha", "/beta", "/gamma"} {
		if !strings.HasSuffix(p.Links[i].URL, tail) {
			t.Fatalf("link %d URL = %q, want suffix %q", i+1, p.Links[i].URL, tail)
		}
		if p.Links[i].N != i+1 {
			t.Fatalf("link %d N = %d", i, p.Links[i].N)
		}
	}
	// order: the h1 row precedes the first link row, which precedes the
	// bullet run, which precedes the table, which precedes the tail.
	idx := func(needle string) int {
		for i, ln := range rows {
			if strings.Contains(ln, needle) {
				return i
			}
		}
		t.Fatalf("row %q not found", needle)
		return -1
	}
	if !(idx("The Fixture Gazette") < idx("link alpha") &&
		idx("link alpha") < idx("• shelf item one") &&
		idx("• shelf item one") < idx("agent │ role") &&
		idx("agent │ role") < idx("tail-marker")) {
		t.Fatalf("row order drifted:\n%s", joined)
	}
}

func TestBrowserLinkDedupe(t *testing.T) {
	doc := `<html><head><title>t</title></head><body>` +
		`<p>see <a href="/x">first</a> and <a href="/x">second</a> and <a href="/y">third</a></p>` +
		`</body></html>`
	p, err := parseHTMLPage(strings.NewReader(doc), "http://localhost:9/page")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Links) != 2 {
		t.Fatalf("dedupe by exact URL: links = %d, want 2", len(p.Links))
	}
	rows := p.renderRows(60)
	var para string
	for _, ro := range rows {
		if strings.Contains(ro.text, "first") {
			para = ro.text
		}
	}
	// the dup href reuses [1]; the fresh href takes [2].
	if !strings.Contains(para, "first [1]") || !strings.Contains(para, "second [1]") || !strings.Contains(para, "third [2]") {
		t.Fatalf("dedupe indexing wrong: %q", para)
	}
}

func TestBrowserSetSizeClipReflow(t *testing.T) {
	b := NewBrowser()
	b.SetSize(40, 12)
	b.page = fixturePage(t)
	b.url = b.page.URL
	b.refreshBody()
	for i, ln := range strings.Split(ansi.Strip(b.View()), "\n") {
		if w := lipgloss.Width(ln); w > 40 {
			t.Fatalf("row %d is %d cells, budget 40: %q", i, w, ln)
		}
	}
	// a narrow SetSize reflows: a long paragraph wraps onto more rows.
	narrow := b.page.renderRows(24)
	wide := b.page.renderRows(100)
	if len(narrow) <= len(wide) {
		t.Fatalf("reflow: narrow rows %d must exceed wide rows %d", len(narrow), len(wide))
	}
}

func TestBrowserScrollOffsets(t *testing.T) {
	b := NewBrowser()
	b.SetSize(60, 10) // 1 bar row + 9 body rows
	b.page = fixturePage(t)
	b.url = b.page.URL
	b.cursor = 0
	b.refreshBody()
	if got := b.vp.YOffset(); got != 0 {
		t.Fatalf("initial YOffset = %d, want 0", got)
	}
	b.Update(browserKey("pgdown"))
	if got := b.vp.YOffset(); got == 0 {
		t.Fatalf("pgdown must move the scroll offset")
	}
	b.Update(browserKey("pgup"))
	if got := b.vp.YOffset(); got != 0 {
		t.Fatalf("pgup back to top: YOffset = %d, want 0", got)
	}
}

func TestBrowserFetchSources(t *testing.T) {
	// file:// — the same fixture through the schemed form.
	abs, err := filepath.Abs("testdata/fixture.html")
	if err != nil {
		t.Fatal(err)
	}
	p, err := FetchPage("file://" + abs)
	if err != nil || p.Title != "Fixture Gazette" {
		t.Fatalf("file:// fetch: title %q err %v", p.Title, err)
	}
	if p.URL != "file://"+abs {
		t.Fatalf("file:// final URL = %q", p.URL)
	}
	// unsupported content: the 8x8 checker PNG sniffs image/png → the dim
	// fallback row, never a parse.
	p2, err := FetchPage("testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("png fetch: %v", err)
	}
	rows := p2.renderRows(60)
	if len(rows) != 1 || rows[0].kind != blkDim || !strings.Contains(rows[0].text, "unsupported content type") {
		t.Fatalf("unsupported content must render one dim row, got %+v", rows)
	}
	// a missing file errors (never a blank page).
	if _, err := FetchPage("testdata/does-not-exist.html"); err == nil {
		t.Fatalf("missing file must error")
	}
	// http: non-localhost is blocked WITHOUT the env flag (read at use time).
	t.Setenv(browserAllowHTTPEnv, "")
	if _, err := FetchPage("http://example.com/"); err == nil || !strings.Contains(err.Error(), browserAllowHTTPEnv) {
		t.Fatalf("outbound http must be blocked by default, got %v", err)
	}
}

func TestBrowserFetchHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/fixture.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "testdata/fixture.html")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p, err := FetchPage(srv.URL + "/fixture.html")
	if err != nil {
		t.Fatalf("localhost fetch: %v", err)
	}
	if p.Title != "Fixture Gazette" || len(p.Links) != 3 {
		t.Fatalf("localhost fixture: title %q links %d", p.Title, len(p.Links))
	}
	// relative hrefs resolve against the page URL.
	if got := p.Links[0].URL; got != srv.URL+"/alpha" {
		t.Fatalf("link [1] resolved = %q, want %q", got, srv.URL+"/alpha")
	}
	// the 404 leg errors with the frozen "no route → go to <base>" wording.
	_, err = FetchPage(srv.URL + "/missing")
	if err == nil || !strings.Contains(err.Error(), "404: no route → go to "+srv.URL) {
		t.Fatalf("404 wording = %v", err)
	}
}

func TestBrowserStarterCard(t *testing.T) {
	b := NewBrowser()
	b.SetSize(80, 10)
	if got := ansi.Strip(b.View()); !strings.Contains(got, "▸ enter a url · /open <url> · e to edit · o for file") {
		t.Fatalf("idle pane must show the starter card, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// the headless screenshot lane (SHOT MODE) — unit half
// ---------------------------------------------------------------------------

// shotTestPNG — the shared checker fixture's bytes (a REAL PNG the save
// convention + the content-addressed id hash).
func shotTestPNG(t *testing.T) []byte {
	t.Helper()
	png, err := os.ReadFile("testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("read the checker fixture: %v", err)
	}
	return png
}

// TestShotKittyFrameShape — the emitted APC, byte-pinned: a=T + t=d +
// q=2 + C=1 + i=<content hash8> + f=100, the PNG's base64 verbatim, and
// NO c=/r= keys ANYWHERE (the wave-81/82 production emission ruling —
// the c=/r= variant did not visibly paint on the member's ghostty).
func TestShotKittyFrameShape(t *testing.T) {
	png := shotTestPNG(t)
	id := KittyImageID(png)
	got := shotKittyFrame(id, png)
	want := "\x1b_Ga=T,t=d,q=2,C=1,i=" + KittyIDHash8(id) + ",f=100;" +
		base64.StdEncoding.EncodeToString(png) + "\x1b\\"
	if got != want {
		t.Fatalf("the shot frame's bytes:\n got %q\nwant %q", got, want)
	}
	for _, banned := range []string{",c=", ",r="} {
		if strings.Contains(got, banned) {
			t.Fatalf("the wave-81 ruling bans %s from the emitted frame: %q", banned, got)
		}
	}
}

// TestBrowserShotCellPx — the THEBORINGOFFICE_CELL_PX=W:H override matrix
// (the 9x18 default stands on any malformed value).
func TestBrowserShotCellPx(t *testing.T) {
	cases := []struct {
		name string
		env  string
		w, h int
	}{
		{"unset", "", 9, 18},
		{"the override", "10:20", 10, 20},
		{"malformed (no colon)", "10", 9, 18},
		{"malformed (words)", "wide:tall", 9, 18},
		{"zero is not a metric", "0:0", 9, 18},
		{"negative is not a metric", "-4:18", 9, 18},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("THEBORINGOFFICE_CELL_PX", tc.env)
			w, h := browserShotCellPx()
			if w != tc.w || h != tc.h {
				t.Fatalf("browserShotCellPx(%q) = %dx%d, want %dx%d", tc.env, w, h, tc.w, tc.h)
			}
		})
	}
}

// TestBrowserShotCellPxRealMetrics — the metric win order end to end: the
// terminal's LEARNED cell size (cellmetrics' registry) beats the 9x18
// default, and the THEBORINGOFFICE_CELL_PX pin beats BOTH.
func TestBrowserShotCellPxRealMetrics(t *testing.T) {
	restore := cellmetrics.ResetForShot()
	t.Cleanup(restore)
	t.Setenv("THEBORINGOFFICE_CELL_PX", "")

	// the zero-state registry: the 9x18 fallback stands (the old tests'
	// exact expectation — zero behavioral change without an answer).
	if w, h := browserShotCellPx(); w != 9 || h != 18 {
		t.Fatalf("no metric learned → the 9x18 fallback: got %dx%d", w, h)
	}

	// the terminal answered (16w × 32h — ghostty's default at 2x DPR):
	// the REAL metric wins.
	cellmetrics.SetForShot(16, 32)
	if w, h := browserShotCellPx(); w != 16 || h != 32 {
		t.Fatalf("the learned metric beats the fallback: got %dx%d, want 16x32", w, h)
	}

	// the member's pin still wins outright.
	t.Setenv("THEBORINGOFFICE_CELL_PX", "10:20")
	if w, h := browserShotCellPx(); w != 10 || h != 20 {
		t.Fatalf("the override beats the learned metric: got %dx%d, want 10x20", w, h)
	}
}

// TestShotBoxPxRealMetrics — the viewport math with the pinned metric:
// an 80×45 pane (43 body rows) at cell 16w×32h renders EXACTLY
// 1280×1376 CSS px (the engine doubles to 2560×2752 device px).
func TestShotBoxPxRealMetrics(t *testing.T) {
	restore := cellmetrics.ResetForShot()
	t.Cleanup(restore)
	cellmetrics.SetForShot(16, 32)
	b := NewBrowser()
	b.SetSize(80, 45)
	w, h := b.ShotBoxPx()
	if w != 1280 || h != 1376 {
		t.Fatalf("ShotBoxPx = %dx%d, want 1280x1376 (80 cols × 16, 43 body rows × 32)", w, h)
	}
}

// TestBrowserShotOpenRealMetrics — the render's dims ride the learned
// metric END TO END: the fake engine records 1280×1376 for the 80×45
// pane, never the 9x18 guess's 720×774.
func TestBrowserShotOpenRealMetrics(t *testing.T) {
	restore := cellmetrics.ResetForShot()
	t.Cleanup(restore)
	cellmetrics.SetForShot(16, 32)
	r := newShotRig(t, true, true)
	r.b.SetSize(80, 45)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	if r.e.calls != 1 || r.e.lastW != 1280 || r.e.lastH != 1376 {
		t.Fatalf("the render fired %d× (%dx%d), want 1× (1280x1376)", r.e.calls, r.e.lastW, r.e.lastH)
	}
}

// TestBrowserShotResizeRequeries — the resize re-shot debounce re-arms
// the cell-size probe (a font zoom changes the cell's px under the same
// pane box): the settled tick's re-render fires the emitter AND rides the
// registry's CURRENT metric (100 cols × 16, (30-2) body rows × 32 =
// 1600×896).
func TestBrowserShotResizeRequeries(t *testing.T) {
	restore := cellmetrics.ResetForShot()
	t.Cleanup(restore)
	queries := 0
	cellmetrics.SetQueryFunc(func() { queries++ })
	cellmetrics.SetForShot(16, 32)
	r := newShotRig(t, true, true)
	r.b.SetSize(80, 45)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	if queries != 0 {
		t.Fatalf("the open path never probes (the boot + resize arms own that): %d", queries)
	}
	r.b.SetSize(100, 30)                 // stamp the resize
	armed := r.b.Update(browserKey("j")) // the sweep arms the debounce tick
	if armed == nil {
		t.Fatal("the routed msg arms the debounce tick")
	}
	fire := r.b.Update(armed()) // the tick lands → re-query + re-render
	if fire == nil {
		t.Fatal("the settled debounce fires the re-render")
	}
	if queries != 1 {
		t.Fatalf("the settled debounce re-armed the probe %d×, want 1", queries)
	}
	r.b.Update(fire())
	if r.e.calls != 2 || r.e.lastW != 1600 || r.e.lastH != 896 {
		t.Fatalf("the re-render fired %d× (%dx%d), want 2× (1600x896)", r.e.calls, r.e.lastW, r.e.lastH)
	}
}

// TestShotFailCopyClasses — the failure classifier's frozen rows: chrome-
// missing names the fix, a policy refusal carries the decision's verbatim
// reason, the timeout names the bound, anything else degrades honest.
func TestShotFailCopyClasses(t *testing.T) {
	if got := shotFailCopy(headless.ErrChromeNotFound); got != shotFailChromeCopy {
		t.Fatalf("chrome-missing row = %q, want %q", got, shotFailChromeCopy)
	}
	if got := shotFailCopy(fmt.Errorf("wrap: %w", headless.ErrChromeNotFound)); got != shotFailChromeCopy {
		t.Fatalf("a WRAPPED chrome-missing still classifies: %q", got)
	}
	pol := &headless.PolicyError{URL: "http://x.test/", Reason: "plain http to x.test refused"}
	if got, want := shotFailCopy(pol), "text lane — headless render refused: plain http to x.test refused"; got != want {
		t.Fatalf("the policy row carries the verbatim reason:\n got %q\nwant %q", got, want)
	}
	wantTimeout := fmt.Sprintf("text lane — headless render timed out (%ds)", int(browserShotDeadline.Seconds()))
	if got := shotFailCopy(context.DeadlineExceeded); got != wantTimeout {
		t.Fatalf("the timeout row = %q, want %q", got, wantTimeout)
	}
	if got := shotFailCopy(errors.New("kaboom")); got != "text lane — headless render failed: kaboom" {
		t.Fatalf("the generic row degrades honest: %q", got)
	}
	// the chrome copy names the fix (the requirement's own words).
	if !strings.Contains(shotFailChromeCopy, "install Chrome or export THEBORINGOFFICE_CHROME") {
		t.Fatalf("the chrome-missing row names the fix: %q", shotFailChromeCopy)
	}
}

// TestSaveShotPNG — the save convention: <$THEBORINGOFFICE_HOME>/shots/
// <unixMillis>-<hash8>.png with the bytes round-tripping, and the
// os.TempDir fallback when HOME is unset.
func TestSaveShotPNG(t *testing.T) {
	png := shotTestPNG(t)
	pin := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	defer SetShotNowForShot(func() time.Time { return pin })()

	home := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", home)
	got, err := saveShotPNG(png)
	if err != nil {
		t.Fatalf("saveShotPNG: %v", err)
	}
	want := filepath.Join(home, "shots", "1787918400000-"+KittyIDHash8(KittyImageID(png))+".png")
	if got != want {
		t.Fatalf("the saved path = %q, want %q", got, want)
	}
	back, err := os.ReadFile(got)
	if err != nil || !errors.Is(err, nil) || len(back) != len(png) {
		t.Fatalf("the saved PNG round-trips: %v (%d bytes)", err, len(back))
	}
	for i := range png {
		if back[i] != png[i] {
			t.Fatalf("the saved PNG's bytes drifted at %d", i)
		}
	}

	// the TempDir fallback: HOME unset → <os.TempDir>/shots.
	t.Setenv("THEBORINGOFFICE_HOME", "")
	got2, err := saveShotPNG(png)
	if err != nil {
		t.Fatalf("saveShotPNG (TempDir leg): %v", err)
	}
	if want2 := filepath.Join(os.TempDir(), "shots", filepath.Base(want)); got2 != want2 {
		t.Fatalf("the fallback path = %q, want %q", got2, want2)
	}
}

// TestBrowserGoldenRows pins the fixture's raw row stream VERBATIM — the
// regression net for renderer edits (any drift fails with the full diff).
func TestBrowserGoldenRows(t *testing.T) {
	p := fixturePage(t)
	got := strings.Join(strippedRows(t, p, 78), "\n")
	want := `Fixture Gazette

The Fixture Gazette

Morning edition — the office browser renders this stub page as plain text
rows.

News desk

Open link alpha [1] for the first story.

Then link beta [2] for the follow-up.

Finally link gamma [3] closes the set.

Bullet shelf

• shelf item one
• shelf item two
• shelf item three
• shelf item four
• shelf item five
• shelf item six
• shelf item seven
• shelf item eight
• shelf item nine
• shelf item ten
• shelf item eleven
• shelf item twelve

Data corner

agent │ role
tekton-1 │ developer
skopos-1 │ scout

The diagram below renders as a chip — the pane never fetches image bytes.

🖼 fixture diagram

go build ./...
go test ./internal/panels/ -run Browser

Filler row one — padding so the page outgrows the sidebar viewport.

Filler row two — padding so the page outgrows the sidebar viewport.

Filler row three — padding so the page outgrows the sidebar viewport.

Filler row four — padding so the page outgrows the sidebar viewport.

Filler row five — padding so the page outgrows the sidebar viewport.

Filler row six — padding so the page outgrows the sidebar viewport.

tail-marker — the pgdn proof scrolls this exact row into view.`
	if got != want {
		t.Fatalf("fixture rows drifted:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
