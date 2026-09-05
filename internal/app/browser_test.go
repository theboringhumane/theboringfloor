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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/headless"
	"github.com/theboringhumane/theboringfloor/internal/panels"
)

// TestMain pins the panels headless engine seam HERMETIC for the whole
// app suite (no live chrome in unit tests): the package-wide default
// verdict is chrome-missing — deterministic on every host. The shot
// suites swap their own fake per test (panels.SetHeadlessForShot).
func TestMain(m *testing.M) {
	defer panels.SetHeadlessForShot(func() (string, bool) { return "", false },
		func(context.Context, string, int, int) (*headless.Result, error) {
			return nil, headless.ErrChromeNotFound
		})()
	os.Exit(m.Run())
}

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

// pinBrowserTextLane — hermetic text lane on EVERY host: on a
// kitty-capable machine with terminal-browser on PATH the pane's lane
// resolve would otherwise spawn a real child out of these opens. The
// premium lane's LIVE app wiring rides browser_lane_test.go.
func pinBrowserTextLane(t *testing.T) {
	t.Helper()
	t.Setenv(panels.BrowserLaneOffEnv, "1")
}

// ctrlB — the left-pane switcher key, bubbletea-encoded.
func ctrlB() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl}) }

func TestBrowserLeftSlotRegistration(t *testing.T) {
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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
	pinBrowserTextLane(t)
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

// ---------------------------------------------------------------------------
// the in-pane URL gestures through the REAL app routing (`e` edits inline,
// `O` opens the current page in the OS browser — both ride the switcher's
// unclaimed-key hop into the pane)
// ---------------------------------------------------------------------------

// shiftO — the OS-open key, bubbletea-encoded (a typed capital O).
func shiftO() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'O', Text: "O"}) }

// browserShortURLServer — the fixture over localhost http at SHORT paths
// (the 64-cell left pane must render the whole URL + the editor hint
// untruncated for the byte assertions).
func browserShortURLServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, p := range []string{"/f", "/g"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			http.ServeFile(w, r, "../panels/testdata/fixture.html")
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestBrowserEditURLThroughApp — `e` on the active browser slot opens the
// pane's inline editor prefilled with the current URL, typed keys land in
// the EDITOR (never the chat draft), and enter commits the EDITED url
// through the pane's normal Open path (the bar wears the new location).
func TestBrowserEditURLThroughApp(t *testing.T) {
	pinBrowserTextLane(t)
	srv := browserShortURLServer(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + srv.URL + "/f"})
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("setup: browser slot active, got %d", got)
	}

	// `e` opens the editor: the row carries the prefilled URL + the hint…
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "▸ "+srv.URL+"/f") || !strings.Contains(frame, "enter: open · esc: cancel") {
		t.Fatalf("the editor row rides the location bar:\n%s", frame)
	}
	// …typed keys land in the editor — never the chat draft.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'z', Text: "z"}))
	if f := ansi.Strip(m.Frame()); strings.Contains(f, "› z") {
		t.Fatalf("the editor owns typed keys (no draft leak):\n%s", f)
	}
	if f := ansi.Strip(m.Frame()); !strings.Contains(f, srv.URL+"/fz") {
		t.Fatalf("the typed rune spliced into the editor buffer:\n%s", f)
	}
	// edit the location: drop the stray z AND the tail f, type g…
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))
	// …and enter commits the EDITED url through the pane's Open path.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	frame = ansi.Strip(m.Frame())
	for _, want := range []string{"▸ " + srv.URL + "/g", "The Fixture Gazette", "· ctrl+b"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the commit→open frame carries %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "esc: cancel") {
		t.Fatalf("enter closed the editor:\n%s", frame)
	}
}

// TestBrowserEditCancelThroughApp — esc on the editor cancels back to the
// previous frame (never the leave-to-floor esc, never a fetch).
func TestBrowserEditCancelThroughApp(t *testing.T) {
	pinBrowserTextLane(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := browserFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})
	before := ansi.Strip(m.Frame())

	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'e', Text: "e"}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'x', Text: "x"}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("the editor's esc cancels — it NEVER leaves the slot, got %d", got)
	}
	if after := ansi.Strip(m.Frame()); after != before {
		t.Fatalf("the cancel restores the frame byte-for-byte:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
	// …and the NEXT esc is the pane's leave again.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if got := m.LeftTabIndex(); got != leftTabFloor {
		t.Fatalf("esc post-edit leaves to the floor, got %d", got)
	}
}

// TestBrowserOSOpenThroughApp — `O` on the loaded page fires the CURRENT
// URL at the OS-open cascade (the runner seam faked — never a real
// shell-out) and the dim confirmation row lands in the pane.
func TestBrowserOSOpenThroughApp(t *testing.T) {
	pinBrowserTextLane(t)
	srv := browserShortURLServer(t)
	var opened []panels.LinkTarget
	restore := panels.SetOpenRunnerForShot(func(tgt panels.LinkTarget) error {
		opened = append(opened, tgt)
		return nil
	})
	defer restore()

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + srv.URL + "/f"})
	m = runMsg(t, m, shiftO())

	if len(opened) != 1 || opened[0].Kind != panels.LinkURL || opened[0].Value != srv.URL+"/f" {
		t.Fatalf("O opened the CURRENT page through the cascade: %+v", opened)
	}
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "opened in system browser: "+srv.URL+"/f") {
		t.Fatalf("the dim confirmation row lands in the pane:\n%s", frame)
	}
}

// ---------------------------------------------------------------------------
// the headless SHOT's app-level latch + flow (the frame-splice byte-pin
// lives in browser_frame_test.go)
// ---------------------------------------------------------------------------

// TestBrowserSlashOpenShotNoticeLatch — /open with a working headless
// engine: the render verdict rides the SAME BrowserPageMsg hop as the
// fetch, and the /open office notice settles on the FETCH verdict
// whichever msg lands — the shot msg never touches the latch. Proven
// BOTH ways: the full drain (fetch first) and a hand-fed shot-first
// ordering.
func TestBrowserSlashOpenShotNoticeLatch(t *testing.T) {
	pinBrowserLaneEnv(t)                    // the kitty display lane
	t.Setenv(panels.BrowserLaneOffEnv, "1") // the zenbu child stays out
	t.Setenv("PATH", t.TempDir())           // no terminal-browser binary either
	t.Setenv("THEFLOOR_HOME", t.TempDir())  // the save lands scratch
	png, err := os.ReadFile("../panels/testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("read the checker fixture: %v", err)
	}
	restore := panels.SetHeadlessForShot(
		func() (string, bool) { return "/fake/chrome", true },
		func(_ context.Context, rawurl string, w, h int) (*headless.Result, error) {
			return &headless.Result{URL: rawurl, Title: "Fixture Gazette", PNG: png}, nil
		})
	defer restore()

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	raw := browserFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})

	// the notice settled on the FETCH verdict (title · url), NOT the
	// shot's; the shot display path engaged.
	if !lastOfficeNoticeHas(m, "browser: Fixture Gazette · "+raw) {
		t.Fatalf("the /open notice settles on the fetch verdict (never the shot's)")
	}
	if !m.BrowserShotActive() {
		t.Fatal("the render's verdict entered shot mode through the app glue")
	}
	if m.BrowserShotPath() == "" {
		t.Fatal("the PNG saved (the path row for the member's `o` habit)")
	}
}

// TestBrowserShotMsgNeverSettlesLatch — the ORDERING half, hand-fed: a
// Shot-carrying verdict landing while the /open latch is armed leaves the
// latch for the FETCH verdict (which then settles exactly once).
func TestBrowserShotMsgNeverSettlesLatch(t *testing.T) {
	pinBrowserTextLane(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.browserSlashNote = "http://x.test/" // the armed /open latch

	// the render verdict lands FIRST (production races the two cmds).
	m = runMsg(t, m, panels.BrowserPageMsg{Shot: &panels.BrowserShot{Seq: 1, Err: errors.New("boom")}})
	if m.browserSlashNote == "" {
		t.Fatal("the shot verdict must NEVER settle the /open notice latch")
	}
	if lastOfficeNoticeHas(m, "browser:") {
		t.Fatalf("no notice rides the shot verdict")
	}

	// the fetch verdict lands: the latch settles on IT, exactly once.
	m = runMsg(t, m, panels.BrowserPageMsg{URL: "http://x.test/", Page: &panels.Page{URL: "http://x.test/", Title: "T"}})
	if m.browserSlashNote != "" {
		t.Fatal("the fetch verdict settles the latch")
	}
	if !lastOfficeNoticeHas(m, "browser: T · http://x.test/") {
		t.Fatalf("the notice carries the fetch's title · url")
	}
}
