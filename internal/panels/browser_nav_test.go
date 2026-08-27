// browser_nav_test.go — the browser pane's NAVIGATION half: the bounded
// back/forward ring (wraps at the edges, restores scroll offsets, drops
// forward entries on a fresh navigation, caps at 100), `o` on the focused
// link (a local file rides the OS-open seam; http(s) re-fetches and
// navigates in place), `r` reloads the same page deterministically (no
// twin history entry), and Close clears the ring.
package panels

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// navPage builds a tiny parsed page in-memory (one heading + n links).
func navPage(rawurl, title string, links ...BrowserLink) *Page {
	p := &Page{URL: rawurl, Title: title, Links: links}
	p.blocks = []block{{kind: blkTitle, text: title}}
	for _, l := range links {
		p.blocks = append(p.blocks, block{kind: blkPara, text: l.Text + " [1]", links: []int{l.N - 1}})
	}
	for i := 0; i < 30; i++ { // body taller than any test viewport (scroll room)
		p.blocks = append(p.blocks, block{kind: blkPara, text: fmt.Sprintf("%s filler %02d", title, i)})
	}
	return p
}

// navRig — a Browser with a stub fetch map + a recording open seam.
type navRig struct {
	b       *Browser
	pages   map[string]*Page
	fetches []string
	opened  []LinkTarget
	openErr error
}

func newNavRig(t *testing.T, pages map[string]*Page) *navRig {
	t.Helper()
	// the text-lane suites are hermetic on EVERY host: on a kitty-capable
	// machine with terminal-browser on PATH the lane resolve would
	// otherwise spawn a real child out of Open. The premium lane's own
	// matrix lives in browser_panel_lane_test.go.
	t.Setenv(BrowserLaneOffEnv, "1")
	r := &navRig{pages: pages}
	b := NewBrowser()
	b.SetSize(60, 8) // 1 bar + 7 body rows (scrollable fixture bodies)
	b.fetchFn = func(rawurl string) (*Page, error) {
		r.fetches = append(r.fetches, rawurl)
		p, ok := r.pages[rawurl]
		if !ok {
			return nil, errors.New("404: no route → go to " + rawurl)
		}
		return p, nil
	}
	b.openFn = func(tgt LinkTarget) error {
		r.opened = append(r.opened, tgt)
		return r.openErr
	}
	r.b = b
	return r
}

// driveOpen runs Open + lands the produced BrowserPageMsg (the app's
// forwarding hop, one level down).
func (r *navRig) driveOpen(t *testing.T, rawurl string) {
	t.Helper()
	cmd := r.b.Open(rawurl)
	if cmd == nil {
		t.Fatalf("Open(%q) returned no cmd", rawurl)
	}
	msg := cmd()
	pm, ok := msg.(BrowserPageMsg)
	if !ok {
		t.Fatalf("Open(%q) produced %T, want BrowserPageMsg", rawurl, msg)
	}
	r.b.Update(pm)
}

func TestBrowserNavRingWraps(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
		"file:///b.html": navPage("file:///b.html", "Beta"),
	})
	r.driveOpen(t, "file:///a.html")
	r.driveOpen(t, "file:///b.html")
	if len(r.b.hist) != 2 || r.b.histIdx != 1 {
		t.Fatalf("ring = %d @%d, want 2 @1", len(r.b.hist), r.b.histIdx)
	}
	// back → Alpha
	r.b.Update(browserKey("["))
	if r.b.page.Title != "Alpha" || r.b.url != "file:///a.html" {
		t.Fatalf("[ → page %q url %q, want Alpha a.html", r.b.page.Title, r.b.url)
	}
	// at the OLDEST entry: [ wraps to the NEWEST (Beta)
	r.b.Update(browserKey("["))
	if r.b.page.Title != "Beta" {
		t.Fatalf("[ at the oldest must wrap to the newest, got %q", r.b.page.Title)
	}
	// at the NEWEST: ] wraps to the OLDEST
	r.b.Update(browserKey("]"))
	if r.b.page.Title != "Alpha" {
		t.Fatalf("] at the newest must wrap to the oldest, got %q", r.b.page.Title)
	}
	// forward → Beta again
	r.b.Update(browserKey("]"))
	if r.b.page.Title != "Beta" {
		t.Fatalf("] → %q, want Beta", r.b.page.Title)
	}
}

func TestBrowserNavRestoresScroll(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
		"file:///b.html": navPage("file:///b.html", "Beta"),
	})
	r.driveOpen(t, "file:///a.html")
	r.b.Update(browserKey("pgdown"))
	scrolled := r.b.vp.YOffset()
	if scrolled == 0 {
		t.Fatalf("pgdown must scroll Alpha before the ring move")
	}
	r.driveOpen(t, "file:///b.html")
	if got := r.b.vp.YOffset(); got != 0 {
		t.Fatalf("a fresh navigation lands at the top, got offset %d", got)
	}
	r.b.Update(browserKey("[")) // back to Alpha
	if got := r.b.vp.YOffset(); got != scrolled {
		t.Fatalf("[ must restore Alpha's offset %d, got %d", scrolled, got)
	}
	r.b.Update(browserKey("]")) // forward to Beta — ITS stash was 0
	if got := r.b.vp.YOffset(); got != 0 {
		t.Fatalf("] must restore Beta's offset 0, got %d", got)
	}
}

func TestBrowserNavDropsForwardOnFreshNav(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
		"file:///b.html": navPage("file:///b.html", "Beta"),
		"file:///c.html": navPage("file:///c.html", "Gamma"),
	})
	r.driveOpen(t, "file:///a.html")
	r.driveOpen(t, "file:///b.html")
	r.b.Update(browserKey("[")) // back at Alpha — Beta rides the forward slot
	r.driveOpen(t, "file:///c.html")
	if len(r.b.hist) != 2 || r.b.hist[1].url != "file:///c.html" {
		t.Fatalf("a fresh nav must drop forward entries: ring %+v", r.b.hist)
	}
}

func TestBrowserNavRingCap(t *testing.T) {
	pages := map[string]*Page{}
	for i := 0; i < browserHistMax+20; i++ {
		u := fmt.Sprintf("file:///p%03d.html", i)
		pages[u] = navPage(u, fmt.Sprintf("P%03d", i))
	}
	r := newNavRig(t, pages)
	for i := 0; i < browserHistMax+20; i++ {
		r.driveOpen(t, fmt.Sprintf("file:///p%03d.html", i))
	}
	if len(r.b.hist) != browserHistMax {
		t.Fatalf("ring must cap at %d, got %d", browserHistMax, len(r.b.hist))
	}
	if r.b.hist[0].url != "file:///p020.html" {
		t.Fatalf("the oldest 20 must evict: ring[0] = %q", r.b.hist[0].url)
	}
}

func TestBrowserOpenFocusedLinkFileVsURL(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///index.html": navPage("file:///index.html", "Index",
			BrowserLink{N: 1, URL: "file:///tmp/notes.txt", Text: "local note"},
			BrowserLink{N: 2, URL: "http://localhost:9/docs", Text: "remote docs"}),
		"http://localhost:9/docs": navPage("http://localhost:9/docs", "Docs"),
	})
	r.driveOpen(t, "file:///index.html")

	// cursor starts on link [1] — the LOCAL file: `o` rides the exec seam.
	cmd := r.b.Update(browserKey("o"))
	if cmd == nil {
		t.Fatalf("o on a focused link must produce a cmd")
	}
	msg := cmd()
	om, ok := msg.(BrowserOpenedMsg)
	if !ok {
		t.Fatalf("local `o` produced %T, want BrowserOpenedMsg", msg)
	}
	if len(r.opened) != 1 || r.opened[0].Kind != LinkFile || r.opened[0].Value != "/tmp/notes.txt" {
		t.Fatalf("exec seam got %+v", r.opened)
	}
	r.b.Update(om) // the app forwards the verdict: the note row lands
	if !strings.Contains(ansi.Strip(r.b.View()), "→ opened: notes.txt") {
		t.Fatalf("the note row must carry the verdict, got:\n%s", ansi.Strip(r.b.View()))
	}

	// move to link [2] — REMOTE: `o` navigates in place (re-fetch, no exec).
	r.b.Update(browserKey("down"))
	if r.b.cursor != 1 {
		t.Fatalf("down → cursor %d, want 1", r.b.cursor)
	}
	cmd = r.b.Update(browserKey("o"))
	if cmd == nil {
		t.Fatalf("o on the remote link must navigate")
	}
	msg = cmd()
	pm, ok := msg.(BrowserPageMsg)
	if !ok {
		t.Fatalf("remote `o` produced %T, want BrowserPageMsg (navigate in place)", msg)
	}
	r.b.Update(pm)
	if r.b.page.Title != "Docs" || r.b.url != "http://localhost:9/docs" {
		t.Fatalf("remote `o` must navigate, got %q @ %q", r.b.page.Title, r.b.url)
	}
	if len(r.opened) != 1 {
		t.Fatalf("a remote link must never hit the exec seam: %+v", r.opened)
	}
	if len(r.b.hist) != 2 {
		t.Fatalf("the in-place nav pushes history: ring %d", len(r.b.hist))
	}
}

func TestBrowserOpenVerdictFailure(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///index.html": navPage("file:///index.html", "Index",
			BrowserLink{N: 1, URL: "file:///tmp/gone.txt", Text: "gone"}),
	})
	r.openErr = errors.New("xdg-open: boom")
	r.driveOpen(t, "file:///index.html")
	msg := r.b.Update(browserKey("o"))()
	r.b.Update(msg)
	if !strings.Contains(ansi.Strip(r.b.View()), "could not open: xdg-open: boom") {
		t.Fatalf("a failed exec must post the dim failure note, got:\n%s", ansi.Strip(r.b.View()))
	}
}

func TestBrowserReloadDeterministic(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
	})
	r.driveOpen(t, "file:///a.html")
	before := ansi.Strip(r.b.View())
	cmd := r.b.Update(browserKey("r"))
	if cmd == nil {
		t.Fatalf("r on a loaded page must reload")
	}
	r.b.Update(cmd())
	after := ansi.Strip(r.b.View())
	if before != after {
		t.Fatalf("reload must be byte-deterministic:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
	if len(r.b.hist) != 1 {
		t.Fatalf("reload must not push a twin entry: ring %d", len(r.b.hist))
	}
	if len(r.fetches) != 2 || r.fetches[1] != "file:///a.html" {
		t.Fatalf("reload re-fetches the same URL: %v", r.fetches)
	}
}

func TestBrowserCloseClearsHistory(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///a.html": navPage("file:///a.html", "Alpha"),
	})
	r.driveOpen(t, "file:///a.html")
	r.b.Close()
	if len(r.b.hist) != 0 || r.b.histIdx != -1 || r.b.page != nil || r.b.url != "" {
		t.Fatalf("Close must clear the ring + the page: %+v", r.b)
	}
}

func TestBrowserLeaveMsg(t *testing.T) {
	r := newNavRig(t, nil)
	for _, k := range []string{"q", "esc"} {
		cmd := r.b.Update(browserKey(k))
		if cmd == nil {
			t.Fatalf("%s must produce the leave cmd", k)
		}
		if _, ok := cmd().(BrowserLeaveMsg); !ok {
			t.Fatalf("%s must produce BrowserLeaveMsg, got %T", k, cmd())
		}
	}
}

func TestBrowserLinkCursorBrightensRow(t *testing.T) {
	r := newNavRig(t, map[string]*Page{
		"file:///index.html": navPage("file:///index.html", "Index",
			BrowserLink{N: 1, URL: "file:///tmp/a.txt", Text: "row one"},
			BrowserLink{N: 2, URL: "file:///tmp/b.txt", Text: "row two"}),
	})
	r.driveOpen(t, "file:///index.html")
	find := func(needle string) string {
		for _, ln := range strings.Split(r.b.View(), "\n") {
			if strings.Contains(ln, needle) {
				return ln
			}
		}
		return ""
	}
	rowOne, rowTwo := find("row one"), find("row two")
	if rowOne == "" || rowTwo == "" {
		t.Fatalf("both link rows must render")
	}
	// cursor 0: the first link row paints ACCENT (focused), the second DIM
	// — the raw SGR bytes must differ (dim→bright when focused).
	if rowOne == rowTwo {
		t.Fatalf("focused vs unfocused link rows must style differently")
	}
	r.b.Update(browserKey("down"))
	if got := find("row one"); got == rowOne {
		t.Fatalf("the cursor leaving row one must dim it")
	}
	if got := find("row two"); got == rowTwo {
		t.Fatalf("the cursor landing on row two must brighten it")
	}
}
