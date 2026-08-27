// browser_test.go — the BROWSER tab's APP seam (the pane's own contracts
// live in internal/panels' browser_*_test.go):
//
//	(a) registration — the strip's factory order is chat|terminal|agents|
//	    board|mail|activity|git|browser: browser at index 7, git UNMOVED
//	    at 6 (the floor click's activity pin at 5 never shifted), and the
//	    tab cycle wraps through it while TabJump's digit map stays 1..7
//	    (the burned-keys ruling — no "8" in v1);
//	(b) `/open <url>` happy path — the slash jumps to the tab, the fetch
//	    (a REAL file:// read of the shared fixture) lands, the location
//	    bar + title render, and the dim office notice posts;
//	(c) `/open` error path — a stub-server 404 surfaces the frozen
//	    "404: no route → go to <base>" wording as dim pane rows AND a red
//	    office notice (never fatal);
//	(d) `/open` with no arg posts the usage notice and never moves tabs.
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

func TestBrowserTabRegistration(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	// factory order: git UNMOVED at 6, browser appended at 7.
	if !m.SelectTab("git") || m.ActiveTabIndex() != 6 {
		t.Fatalf("git must stay at index 6, got %d", m.ActiveTabIndex())
	}
	if !m.SelectTab("browser") || m.ActiveTabIndex() != 7 {
		t.Fatalf("browser must register at index 7, got %d", m.ActiveTabIndex())
	}
	// the cycle wraps THROUGH browser: git → browser → chat.
	m.tabs.SetActive(6)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if got := m.ActiveTabIndex(); got != 7 {
		t.Fatalf("tab from git (6) → %d, want 7 (browser)", got)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("tab from browser (7) → %d, want 0 (wrap to chat)", got)
	}
	// and shift+tab from chat lands ON browser.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}))
	if got := m.ActiveTabIndex(); got != 7 {
		t.Fatalf("shift+tab from chat → %d, want 7 (browser)", got)
	}
	// digits stay 1..7: no "8" jump in v1.
	if got := m.keys.TabJump("8"); got != -1 {
		t.Fatalf(`TabJump("8") = %d, want -1 (cycle-only in v1)`, got)
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

	// the slash jumped to the browser tab…
	if got := m.ActiveTabIndex(); got != browserIndex {
		t.Fatalf("/open must jump to the browser tab (%d), got %d", browserIndex, got)
	}
	// …the pane loaded the fixture (the runMsg drain executed the fetch)…
	if m.browser == nil {
		t.Fatalf("the browser pane must be wired")
	}
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{
		"▸ file:///", // the location bar rides the fetched URL (78-col sidebar truncates the tail)
		"The Fixture Gazette",
		"Open link alpha [1] for the first story.",
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
	// (State-side: the chat tab is NOT in the frame while the browser tab
	// is active — read the transcript, not the pixels.)
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

	if got := m.ActiveTabIndex(); got != browserIndex {
		t.Fatalf("/open must still jump to the browser tab, got %d", got)
	}
	frame := ansi.Strip(m.Frame())
	// the pane body carries the frozen 404 wording as dim rows…
	if !strings.Contains(frame, "404: no route → go to "+srv.URL) {
		t.Fatalf("the pane must show the 404 dim rows, got:\n%s", frame)
	}
	// …and the office notice is the red error variant (state-side read —
	// the chat tab isn't in the frame while the browser tab is active).
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
}

// TestBrowserLeaveReturnsToChat — the pane's q/esc ride BrowserLeaveMsg
// back to the chat tab (and q on the browser tab does NOT quit the app).
func TestBrowserLeaveReturnsToChat(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + browserFixtureURL(t)})
	if got := m.ActiveTabIndex(); got != browserIndex {
		t.Fatalf("setup: browser tab active, got %d", got)
	}
	// esc leaves…
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("esc on the browser tab must land on chat, got %d", got)
	}
	// …and so does q (the global q-quit yields on THIS tab).
	m.tabs.SetActive(browserIndex)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("q on the browser tab must leave to chat (not quit), got %d", got)
	}
}

// TestBrowserPageMsgRoutedOffTab — a fetch verdict landing while ANOTHER
// tab is active still reaches the pane (never misdelivered through the
// active-tab hop).
func TestBrowserPageMsgRoutedOffTab(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	page := &panels.Page{URL: "file:///x.html", Title: "Offtab"}
	m = runMsg(t, m, panels.BrowserPageMsg{URL: "file:///x.html", Page: page})
	if m.browser == nil {
		t.Fatalf("the browser pane must be wired")
	}
	// the pane applied the page while the chat tab stayed active.
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("an off-tab verdict must not move tabs, got %d", got)
	}
	m.SelectTab("browser")
	if frame := ansi.Strip(m.Frame()); !strings.Contains(frame, "· Offtab") {
		t.Fatalf("the off-tab page must render on entry, got:\n%s", frame)
	}
}
