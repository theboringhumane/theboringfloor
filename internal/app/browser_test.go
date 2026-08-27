// browser_test.go — the BROWSER surface's APP seam (the pane's own
// contracts live in internal/panels' browser_*_test.go):
//
//	(a) the LEFT slot — the right strip keeps its seven tabs (git UNMOVED
//	    at 6, the cycle wrapping git → chat with NO browser stop) while
//	    the browser rides the left pane's floor|browser switcher: floor
//	    by default, ctrl+b flipping BOTH ways, TabJump's digit map still
//	    1..7 (the burned-keys ruling);
//	(b) `/open <url>` happy path — the slash flips the left slot to the
//	    browser (the RIGHT strip never moves), the fetch (a REAL file://
//	    read of the shared fixture) lands, the location bar + title
//	    render in the LEFT pane, and the dim office notice posts;
//	(c) `/open` error path — a stub-server 404 surfaces the frozen
//	    "404: no route → go to <base>" wording as dim pane rows AND a red
//	    office notice (never fatal);
//	(d) `/open` with no arg posts the usage notice and never moves
//	    EITHER switcher.
package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/panels"
)

// browserFixtureURL — the shared panels fixture as a file:// URL (the
// REAL FetchPage source, no network).
func browserFixtureURL(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../panels/testdata/fixture.html")
	if err != nil {
		t.Fatal(err)
	}
	return "file://" + abs
}

// ctrlB — the left-pane switcher key, bubbletea-encoded.
func ctrlB() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl}) }

func TestBrowserLeftSlotRegistration(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	// the right strip keeps its seven tabs; the browser is NOT one.
	if !m.SelectTab("git") || m.ActiveTabIndex() != 6 {
		t.Fatalf("git must stay at index 6, got %d", m.ActiveTabIndex())
	}
	if m.SelectTab("browser") {
		t.Fatalf("the browser must NOT ride the right strip anymore")
	}
	// the cycle wraps git → chat (no browser stop in the strip).
	m.tabs.SetActive(6)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("tab from git (6) → %d, want 0 (wrap to chat)", got)
	}
	// and shift+tab from chat lands ON git.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}))
	if got := m.ActiveTabIndex(); got != 6 {
		t.Fatalf("shift+tab from chat → %d, want 6 (git)", got)
	}
	// the left slot defaults to the floor; ctrl+b flips it BOTH ways.
	if m.LeftTabIndex() != leftTabFloor {
		t.Fatalf("the left slot must default to floor, got %d", m.LeftTabIndex())
	}
	m = runMsg(t, m, ctrlB())
	if m.LeftTabIndex() != leftTabBrowser {
		t.Fatalf("ctrl+b must flip the left slot to browser, got %d", m.LeftTabIndex())
	}
	m = runMsg(t, m, ctrlB())
	if m.LeftTabIndex() != leftTabFloor {
		t.Fatalf("ctrl+b must flip the left slot back to floor, got %d", m.LeftTabIndex())
	}
	// digits stay 1..7 for the right strip — the browser never jumps.
	if got := m.keys.TabJump("8"); got != -1 {
		t.Fatalf(`TabJump("8") = %d, want -1`, got)
	}
	if got := m.keys.TabJump("7"); got != 6 {
		t.Fatalf(`TabJump("7") = %d, want 6 (git)`, got)
	}
}

func TestBrowserSlashOpenHappyPath(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := browserFixtureURL(t)

	m = runMsg(t, m, slashMsg{text: "/open " + raw})

	// the slash flipped the LEFT slot to the browser…
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("/open must flip the left slot to the browser (%d), got %d", leftTabBrowser, got)
	}
	// …while the RIGHT strip never moved (chat stays active)…
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("/open must never move the right strip, got %d", got)
	}
	// …the pane loaded the fixture (the runMsg drain executed the fetch)…
	if m.browser == nil {
		t.Fatalf("the browser pane must be wired")
	}
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{
		"▸ file:///", // the location bar rides the fetched URL
		"The Fixture Gazette",
		"Open link alpha [1] for the first story.",
		"· ctrl+b", // the left slot's switcher strip
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("browser frame missing %q:\n%s", want, frame)
		}
	}
	// pgdn scrolls the body app-side (the chip + tail row start below the fold).
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	frame = ansi.Strip(m.Frame())
	for _, want := range []string{"🖼 fixture diagram", "tail-marker"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("scrolled browser frame missing %q:\n%s", want, frame)
		}
	}
	// …and the dim office notice posted (success carries title · url).
	if !lastOfficeNoticeHas(m, "browser: Fixture Gazette · "+raw) {
		t.Fatalf("the /open success notice must post, chat tail: %+v", m.st.Chat[len(m.st.Chat)-1])
	}
}

// lastOfficeNoticeHas — the newest "office" transcript row carries want.
func lastOfficeNoticeHas(m Model, want string) bool {
	for i := len(m.st.Chat) - 1; i >= 0; i-- {
		c := m.st.Chat[i]
		if c.From == "office" {
			return strings.Contains(c.Text, want)
		}
	}
	return false
}

func TestBrowserSlashOpenErrorPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + srv.URL + "/missing"})

	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("/open must still flip the left slot to the browser, got %d", got)
	}
	frame := ansi.Strip(m.Frame())
	// the pane body carries the frozen 404 wording as dim rows…
	if !strings.Contains(frame, "404: no route → go to "+srv.URL) {
		t.Fatalf("the pane must show the 404 dim rows, got:\n%s", frame)
	}
	// …and the office notice is the red error variant (state-side read —
	// the chat panel renders on the right strip under the browser slot).
	if !lastOfficeNoticeHas(m, "browser: 404: no route → go to "+srv.URL) {
		t.Fatalf("the /open error notice must post")
	}
}

func TestBrowserSlashOpenUsageError(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open"})
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "/open: usage /open <url>") {
		t.Fatalf("bare /open must post the usage notice, got:\n%s", frame)
	}
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("a usage error must never move tabs, got %d", got)
	}
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("a usage error must never move the left slot, got %d", got)
	}
}

// TestBrowserLeaveReturnsToFloor — the pane's q/esc ride BrowserLeaveMsg
// back to the FLOOR tab (and q on the browser slot does NOT quit the app;
// the right strip never moves either).
func TestBrowserLeaveReturnsToFloor(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + browserFixtureURL(t)})
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("setup: browser slot active, got %d", got)
	}
	// esc leaves…
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("esc on the browser slot must land on the floor, got %d", got)
	}
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("esc on the browser slot must never move the right strip, got %d", got)
	}
	// …and so does q (the global q-quit yields on THIS surface).
	m = runMsg(t, m, ctrlB())
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("setup: ctrl+b flips back to the browser, got %d", got)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("q on the browser slot must leave to the floor (not quit), got %d", got)
	}
}

// TestBrowserPageMsgRoutedOffTab — a fetch verdict landing while the
// FLOOR is up still reaches the pane (never misdelivered through the
// active-tab hop); the switcher position is the app's, not the pane's.
func TestBrowserPageMsgRoutedOffTab(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	page := &panels.Page{URL: "file:///x.html", Title: "Offtab"}
	m = runMsg(t, m, panels.BrowserPageMsg{URL: "file:///x.html", Page: page})
	if m.browser == nil {
		t.Fatalf("the browser pane must be wired")
	}
	// the pane applied the page while the floor slot stayed put.
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("an off-slot verdict must not move the switcher, got %d", got)
	}
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("an off-slot verdict must not move the right strip, got %d", got)
	}
	m = runMsg(t, m, ctrlB())
	if frame := ansi.Strip(m.Frame()); !strings.Contains(frame, "· Offtab") {
		t.Fatalf("the off-slot page must render on entry, got:\n%s", frame)
	}
}

// TestBrowserSlotOwnsKeys — while the switcher sits on browser, typed
// letters belong to the PANE (the link cursor's j/k), never to the chat
// textarea on the right strip; flipping back to the floor restores the
// draft keys.
func TestBrowserSlotOwnsKeys(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + browserFixtureURL(t)})
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("setup: browser slot active, got %d", got)
	}
	// "j" rides the pane's link cursor — it must NOT land in the draft.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if f := ansi.Strip(m.Frame()); strings.Contains(f, "› j") {
		t.Fatalf("the browser slot must own typed keys (no draft leak):\n%s", f)
	}
	// back on the floor the SAME key types into the chat draft again.
	m = runMsg(t, m, ctrlB())
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	if f := ansi.Strip(m.Frame()); !strings.Contains(f, "› j") {
		t.Fatalf("the floor slot hands keys back to the chat draft:\n%s", f)
	}
}
