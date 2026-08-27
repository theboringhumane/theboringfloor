// browser_test.go — the browser pane's RENDER half: the fixture parses
// into the exact row set (title line, bold headings, [n]-indexed link
// rows, bullets, " │ " table rows, the 🖼 chip, verbatim code rows, and
// stripped script/style/meta/comment noise), SetSize clips + reflows, and
// pgup/pgdn move the scroll offset. FetchPage's source matrix rides here
// too: file:// + bare paths, the content sniff's dim fallback, the
// localhost whitelist, the outbound-HTTP env gate, and the 404 wording.
package panels

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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
	b.SetSize(60, 10)
	if got := ansi.Strip(b.View()); !strings.Contains(got, "▸ enter a url · /open <url> · o for file") {
		t.Fatalf("idle pane must show the starter card, got:\n%s", got)
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
