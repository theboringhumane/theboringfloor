// browser_open_test.go — the APP leg of the agent browser tool: an
// EvBrowserOpen reaches applyBrowserOpen (the handler the manager's
// one-line applyEvent hookup batches in), an ALLOWED verdict drives the
// browser pane's existing open path (a real localhost fetch through the
// pane's own guard) + posts the dim confirmation notice, a REFUSED
// verdict opens nothing and posts the red reason row, and every other
// event kind no-ops (the batch-leg contract). The screenshot/snapshot
// legs: a fake headless engine (NO live chrome in unit tests) proves
// the PNG save + notice + slot flip (shot) and the synthetic follow-up
// prompt + member note (snapshot).
package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/browsertools/action"
	"github.com/theboringhumane/theboringoffice/internal/headless"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// drainBrowserCmd — runMsg's BFS for a handler-returned cmd tree (the
// fetch rides a tea.Cmd whose BrowserPageMsg must flow back through
// Update; heartbeats dropped exactly like runMsg).
func drainBrowserCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		out := c()
		if out == nil {
			continue
		}
		switch out := out.(type) {
		case tea.BatchMsg:
			queue = append(queue, out...)
		case spinner.TickMsg:
		case cursor.BlinkMsg:
		default:
			nm, next := m.Update(out)
			m = nm.(Model)
			if next != nil {
				queue = append(queue, next)
			}
		}
	}
	return m
}

// officeRows — every chat row the office posted (the notice feed).
func officeRows(m Model) []state.ChatMsg {
	var out []state.ChatMsg
	for _, c := range m.st.Chat {
		if c.From == "office" {
			out = append(out, c)
		}
	}
	return out
}

func TestBrowserOpenRequestReachesHandler(t *testing.T) {
	pinBrowserTextLane(t) // hermetic: the lane resolve must never spawn a real child here
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>The Boring Gazette</title></head><body><h1>agent opened me</h1></body></html>`))
	}))
	t.Cleanup(srv.Close)

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	// the allowed request (http://127.0.0.1:* passes BOTH the bridge
	// policy's loopback rule and the pane's own fetch whitelist).
	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserOpen, Text: srv.URL, BrowserOpenAllowed: true,
	})
	if cmd == nil {
		t.Fatal("an allowed request must return the pane's open cmd")
	}
	m = drainBrowserCmd(t, m, cmd)

	// the pane actually loaded the page (read the pane's own render — no
	// tab-switch assumption, the pane's home is mid-move).
	frame := ansi.Strip(m.browser.View())
	for _, want := range []string{"The Boring Gazette", "agent opened me", srv.URL} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the browser pane must render the fetched page (%q missing):\n%s", want, frame)
		}
	}
	// the dim confirmation notice posted.
	if !lastOfficeNoticeHas(m, "browser: opening "+srv.URL+" (asked by the boss)") {
		t.Fatalf("the confirmation notice must post, office rows: %+v", officeRows(m))
	}
	// and NO red error row landed.
	for _, c := range officeRows(m) {
		if c.Meta == "error" {
			t.Fatalf("an allowed open must never post a red row, got %q", c.Text)
		}
	}
}

func TestBrowserOpenRefusedPostsReason(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserOpen, Text: "http://theboring.name",
		BrowserOpenAllowed: false, BrowserOpenReason: reason,
	})
	if cmd != nil {
		t.Fatal("a refused request must never kick an open cmd")
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("exactly ONE notice row lands, got %d new", len(m.st.Chat)-before)
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" {
		t.Fatalf("the refusal is a RED office row, got %+v", row)
	}
	if row.Text != "browser: http://theboring.name — "+reason {
		t.Fatalf("the refusal row carries the bridge's exact reason, got %q", row.Text)
	}
	// the pane stayed idle (the starter card, never the refused URL).
	frame := ansi.Strip(m.browser.View())
	if strings.Contains(frame, "theboring.name") {
		t.Fatalf("a refused URL must never reach the pane, got:\n%s", frame)
	}
}

func TestBrowserOpenHandlerNoopsOnOtherKinds(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	for _, ev := range []state.Event{
		{Kind: state.EvStatus, Text: "noise"},
		{Kind: state.EvChatBoss, Msg: state.ChatMsg{ID: "bossmsg-x", Text: "hi"}},
		{Kind: state.EvBrowserOpen, Text: "", BrowserOpenAllowed: true},       // shapeless: no URL, silent
		{Kind: state.EvBrowserScreenshot, Text: "", BrowserOpenAllowed: true}, // shapeless shot, silent
		{Kind: state.EvBrowserSnapshot, Text: "", BrowserOpenAllowed: true},   // shapeless snapshot, silent
	} {
		if cmd := m.applyBrowserOpen(ev); cmd != nil {
			t.Fatalf("kind %q must no-op, got a cmd", ev.Kind)
		}
	}
	if len(m.st.Chat) != before {
		t.Fatalf("no-op kinds must never post notices, got %d new rows", len(m.st.Chat)-before)
	}
}

// ---------------------------------------------------------------- screenshot

// pinFakeBrowserEngines swaps the headless engine seams for fakes (NO
// live chrome in unit tests) and restores them at cleanup.
func pinFakeBrowserEngines(t *testing.T,
	shot func(ctx context.Context, rawurl string, w, h int) (*headless.Result, error),
	snap func(ctx context.Context, rawurl string, maxText int) (*headless.SnapResult, error)) {
	t.Helper()
	oldShot, oldSnap := browserShotFn, browserSnapFn
	if shot != nil {
		browserShotFn = shot
	}
	if snap != nil {
		browserSnapFn = snap
	}
	t.Cleanup(func() { browserShotFn, browserSnapFn = oldShot, oldSnap })
}

func TestBrowserShotFlowSavesPNGAndFlipsSlot(t *testing.T) {
	pinBrowserTextLane(t) // hermetic: the lane resolve must never spawn a real child here
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>The Boring Gazette</title></head><body><h1>agent shot me</h1></body></html>`))
	}))
	t.Cleanup(srv.Close)
	home := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", home)

	fakePNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 1, 2, 3, 4}
	var gotW, gotH int
	var gotDeadline bool
	pinFakeBrowserEngines(t, func(ctx context.Context, rawurl string, w, h int) (*headless.Result, error) {
		gotW, gotH = w, h
		_, gotDeadline = ctx.Deadline()
		return &headless.Result{URL: rawurl, Title: "The Boring Gazette", PNG: fakePNG}, nil
	}, nil)

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserScreenshot, Text: srv.URL, BrowserOpenAllowed: true,
	})
	if cmd == nil {
		t.Fatal("an allowed screenshot request must return the engine + pane-open batch")
	}
	// the REQUEST leg flips the left slot to the browser tab immediately.
	if got := m.LeftTabIndex(); got != leftTabBrowser {
		t.Fatalf("an allowed screenshot flips the left slot to the browser (%d), got %d", leftTabBrowser, got)
	}
	m = drainBrowserCmd(t, m, cmd)

	// the engine got the default 990x540 box under a bounded context.
	if gotW != 990 || gotH != 540 {
		t.Fatalf("the shot box is 990x540, got %dx%d", gotW, gotH)
	}
	if !gotDeadline {
		t.Fatal("the engine ctx must carry the 15s bound")
	}
	// the PNG landed under <THEBORINGOFFICE_HOME>/shots/<ts>-<hash8>.png
	// (the tab's own display path may save ITS shot here too — find MINE
	// by the deterministic hash8 tail of the fake PNG bytes).
	entries, err := os.ReadDir(filepath.Join(home, "shots"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("the shot PNG lands in the shots dir, got %v (err %v)", entries, err)
	}
	sum := sha1.Sum(fakePNG)
	hash8 := hex.EncodeToString(sum[:4])
	var name string
	for _, e := range entries {
		if regexp.MustCompile(`^[0-9]+-[0-9a-f]{8}\.png$`).MatchString(e.Name()) && strings.Contains(e.Name(), hash8) {
			name = e.Name()
		}
	}
	if name == "" {
		t.Fatalf("no <ts>-%s.png among %v — my shot never saved", hash8, entries)
	}
	path := filepath.Join(home, "shots", name)
	if raw, err := os.ReadFile(path); err != nil || string(raw) != string(fakePNG) {
		t.Fatalf("the saved PNG carries the engine's exact bytes (err %v)", err)
	}
	// the transcript notice carries the path, verbatim contract.
	if !lastOfficeNoticeHas(m, "browser: shot of "+srv.URL+" → "+path+" (asked by the boss)") {
		t.Fatalf("the shot notice must post with the PNG path, office rows: %+v", officeRows(m))
	}
	// the pane's normal open rode along (the tab's display path picks up there).
	frame := ansi.Strip(m.browser.View())
	if !strings.Contains(frame, "The Boring Gazette") {
		t.Fatalf("the pane's normal open must drive the page, got:\n%s", frame)
	}
	// and NO red error row landed.
	for _, c := range officeRows(m) {
		if c.Meta == "error" {
			t.Fatalf("an allowed shot must never post a red row, got %q", c.Text)
		}
	}
}

func TestBrowserShotEngineFailurePostsReason(t *testing.T) {
	pinBrowserTextLane(t)
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	called := false
	pinFakeBrowserEngines(t, func(ctx context.Context, rawurl string, w, h int) (*headless.Result, error) {
		called = true
		return nil, errors.New("headless: no Chrome-family browser found")
	}, nil)

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = drainBrowserCmd(t, m, m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserScreenshot, Text: "https://theboring.name", BrowserOpenAllowed: true,
	}))

	if !called {
		t.Fatal("the allowed request must reach the engine")
	}
	if !lastOfficeNoticeHas(m, "browser: shot of https://theboring.name — headless: no Chrome-family browser found") {
		t.Fatalf("the engine failure lands as the red reason row, office rows: %+v", officeRows(m))
	}
	rows := officeRows(m)
	if rows[len(rows)-1].Meta != "error" {
		t.Fatalf("the failure row is RED, got %+v", rows[len(rows)-1])
	}
}

func TestBrowserShotRefusedPostsReason(t *testing.T) {
	pinBrowserTextLane(t)
	called := false
	pinFakeBrowserEngines(t, func(ctx context.Context, rawurl string, w, h int) (*headless.Result, error) {
		called = true
		return nil, nil
	}, nil)

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserScreenshot, Text: "http://theboring.name",
		BrowserOpenAllowed: false, BrowserOpenReason: reason,
	}); cmd != nil {
		t.Fatal("a refused screenshot must never kick an engine cmd")
	}
	if called {
		t.Fatal("a refused screenshot must never reach the engine")
	}
	if m.LeftTabIndex() == leftTabBrowser {
		t.Fatal("a refused screenshot must never flip the left slot")
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("exactly ONE notice row lands, got %d new", len(m.st.Chat)-before)
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" || row.Text != "browser: http://theboring.name — "+reason {
		t.Fatalf("the refusal is the red reason row, got %+v", row)
	}
}

// ----------------------------------------------------------------- snapshot

func TestBrowserSnapFlowSendsFollowup(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	var gotMaxText int
	pinFakeBrowserEngines(t, nil, func(ctx context.Context, rawurl string, maxText int) (*headless.SnapResult, error) {
		gotMaxText = maxText
		return &headless.SnapResult{
			URL: rawurl, Title: "The Boring Gazette", Text: "hello world",
			Links: []headless.Link{{Text: "a", URL: "https://a.example/x"}, {Text: "b", URL: "https://b.example/y"}},
		}, nil
	})

	rb := &recBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserSnapshot, Text: "https://theboring.name/docs", BrowserOpenAllowed: true,
	})
	if cmd == nil {
		t.Fatal("an allowed snapshot request must return the engine cmd")
	}
	m = drainBrowserCmd(t, m, cmd)

	// the engine read with the 6000-rune text cap.
	if gotMaxText != 6000 {
		t.Fatalf("the snapshot text cap is 6000, got %d", gotMaxText)
	}
	// the synthetic follow-up posted BACK to the agent on the SAME
	// backend session (the attachment seam's plain-text call), verbatim:
	if len(rb.sentTexts) != 1 {
		t.Fatalf("exactly ONE follow-up prompt posts back to the agent, got %v", rb.sentTexts)
	}
	want := "[theboringfloor] snapshot of https://theboring.name/docs (title: The Boring Gazette)\n" +
		"hello world\n" +
		"links: [1] https://a.example/x [2] https://b.example/y"
	if rb.sentTexts[0] != want {
		t.Fatalf("the follow-up the agent receives:\n got %q\nwant %q", rb.sentTexts[0], want)
	}
	// the member sees a one-line dim note — never the full text.
	if !lastOfficeNoticeHas(m, `browser: snapshot of https://theboring.name/docs ("The Boring Gazette") → text + 2 links sent to the boss (asked by the boss)`) {
		t.Fatalf("the member note must post, office rows: %+v", officeRows(m))
	}
	for _, c := range officeRows(m) {
		if c.Meta == "error" {
			t.Fatalf("an allowed snapshot must never post a red row, got %q", c.Text)
		}
		if strings.Contains(c.Text, "hello world") {
			t.Fatalf("the member note must never carry the full text, got %q", c.Text)
		}
	}
	// the read never flips the left slot (nothing renders for the member).
	if m.LeftTabIndex() == leftTabBrowser {
		t.Fatal("a snapshot must never flip the left slot to the browser tab")
	}
}

func TestBrowserSnapRefusedPostsReason(t *testing.T) {
	called := false
	pinFakeBrowserEngines(t, nil, func(ctx context.Context, rawurl string, maxText int) (*headless.SnapResult, error) {
		called = true
		return nil, nil
	})

	rb := &recBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserSnapshot, Text: "http://theboring.name",
		BrowserOpenAllowed: false, BrowserOpenReason: reason,
	}); cmd != nil {
		t.Fatal("a refused snapshot must never kick an engine cmd")
	}
	if called {
		t.Fatal("a refused snapshot must never reach the engine")
	}
	if len(rb.sentTexts) != 0 {
		t.Fatalf("a refused snapshot must never post back to the agent, got %v", rb.sentTexts)
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("exactly ONE notice row lands, got %d new", len(m.st.Chat)-before)
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" || row.Text != "browser: http://theboring.name — "+reason {
		t.Fatalf("the refusal is the red reason row, got %+v", row)
	}
}

func TestBrowserSnapEngineFailurePostsReason(t *testing.T) {
	pinFakeBrowserEngines(t, nil, func(ctx context.Context, rawurl string, maxText int) (*headless.SnapResult, error) {
		return nil, errors.New("headless: timed out")
	})

	rb := &recBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = drainBrowserCmd(t, m, m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserSnapshot, Text: "https://theboring.name", BrowserOpenAllowed: true,
	}))

	if len(rb.sentTexts) != 0 {
		t.Fatalf("a failed snapshot must never post back to the agent, got %v", rb.sentTexts)
	}
	if !lastOfficeNoticeHas(m, "browser: snapshot of https://theboring.name — headless: timed out") {
		t.Fatalf("the engine failure lands as the red reason row, office rows: %+v", officeRows(m))
	}
}

// TestSnapshotFollowupShapes — the follow-up builder's unit contract:
// the no-links shape, the 8KB total bound (links tail trims first, then
// the text cuts rune-safe), and the truncation marker.
func TestSnapshotFollowupShapes(t *testing.T) {
	// no links → "links: (none)".
	got := buildSnapshotFollowup("https://x.example", &headless.SnapResult{
		URL: "https://x.example", Title: "T", Text: "body text",
	})
	want := "[theboringfloor] snapshot of https://x.example (title: T)\nbody text\nlinks: (none)"
	if got != want {
		t.Fatalf("no-links followup = %q, want %q", got, want)
	}
	// the 8KB bound: a 6000-rune text + 50 long links overruns — the
	// links TAIL trims first (the text survives whole), the truncated
	// list ends with " …".
	long := &headless.SnapResult{URL: "https://x.example", Title: "T", Text: strings.Repeat("x", 6000)}
	for i := 0; i < 50; i++ {
		long.Links = append(long.Links, headless.Link{Text: "l", URL: "https://example.com/" + strings.Repeat("a", 60)})
	}
	got = buildSnapshotFollowup("https://x.example", long)
	if len(got) > browserSnapMaxFollowup {
		t.Fatalf("the follow-up is bounded at %d bytes, got %d", browserSnapMaxFollowup, len(got))
	}
	if !strings.Contains(got, strings.Repeat("x", 6000)) {
		t.Fatal("the links tail trims FIRST — the full text must survive")
	}
	if !strings.HasSuffix(got, " …") {
		t.Fatalf("a truncated links list ends with the ellipsis marker, got tail %q", got[len(got)-40:])
	}
	if !strings.HasPrefix(got, "[theboringfloor] snapshot of https://x.example (title: T)\n") {
		t.Fatal("the header always survives the bound")
	}
	// no links + an over-long text → the TEXT cuts on a rune boundary.
	huge := &headless.SnapResult{URL: "https://x.example", Title: "T", Text: strings.Repeat("é", 10000)}
	got = buildSnapshotFollowup("https://x.example", huge)
	if len(got) > browserSnapMaxFollowup {
		t.Fatalf("the text cut respects the %d-byte bound, got %d", browserSnapMaxFollowup, len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("the text cut must never split a multi-byte rune")
	}
}

// the shots dir falls back to <os.TempDir>/shots without the home override.
func TestBrowserShotsDirFallback(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", "")
	if got := browserShotsDir(); got != filepath.Join(os.TempDir(), "shots") {
		t.Fatalf("no home override → os.TempDir()/shots, got %q", got)
	}
	t.Setenv("THEBORINGOFFICE_HOME", "/tmp/member-home")
	if got := browserShotsDir(); got != "/tmp/member-home/shots" {
		t.Fatalf("the home override wins, got %q", got)
	}
}

// ------------------------------------------------------------------ action
//
// The MUTATING sibling: an allowed request PARKS behind the member's
// permission modal (the existing popover: toolName "browser", the
// action summary as toolSummary), the answer drives a fake-executor run
// (approve-once — "always" maps to "once") or the rejection path, and
// the outcome posts BACK to the agent as a synthetic follow-up (the
// snapshot precedent) while the member gets a transcript row. NO live
// chrome anywhere here.

// pinFakeBrowserAction swaps the action engine seam for a fake (NO live
// chrome in unit tests) and restores it at cleanup.
func pinFakeBrowserAction(t *testing.T, fn func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error)) {
	t.Helper()
	old := browserActionFn
	browserActionFn = fn
	t.Cleanup(func() { browserActionFn = old })
}

// TestBrowserActionRefusedNoModal — a policy-refused action posts the
// red reason row and NEVER opens the permission modal, never parks a
// hold, never reaches the engine.
func TestBrowserActionRefusedNoModal(t *testing.T) {
	called := false
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		called = true
		return nil, nil
	})

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "http://theboring.name",
		BrowserOpenAllowed: false, BrowserOpenReason: reason,
		BrowserActionOp: "click", BrowserActionSel: "#buy",
	}); cmd != nil {
		t.Fatal("a refused action must never kick a cmd")
	}
	if called {
		t.Fatal("a refused action must never reach the engine")
	}
	if m.permQ.view() != nil || len(m.permQ.pending) != 0 || len(m.browserActionHolds) != 0 {
		t.Fatalf("a refused action must never open the modal or park a hold: view %+v holds %v", m.permQ.view(), m.browserActionHolds)
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("exactly ONE notice row lands, got %d new", len(m.st.Chat)-before)
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" || row.Text != "browser: http://theboring.name — "+reason {
		t.Fatalf("the refusal is the red reason row, got %+v", row)
	}
}

// TestBrowserActionParksModal — an allowed action opens the member's
// EXISTING permission modal: toolName "browser", the action summary as
// toolSummary (selector clipped 40 cols), Agent "boss" — and NOTHING
// executes before the answer.
func TestBrowserActionParksModal(t *testing.T) {
	called := false
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		called = true
		return nil, nil
	})

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	if cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	}); cmd != nil {
		t.Fatal("the request leg parks — it never returns a cmd")
	}
	if called {
		t.Fatal("nothing executes before the member answers")
	}
	v := m.permQ.view()
	if v == nil {
		t.Fatal("the permission modal must be open")
	}
	if v.ToolName != "browser" || v.Summary != "click '#buy' on theboring.name" || v.Agent != "boss" {
		t.Fatalf("the modal shows toolName/summary/agent: %+v", v)
	}
	if len(m.browserActionHolds) != 1 || m.browserActionHolds[v.ID].sel != "#buy" {
		t.Fatalf("the hold parks keyed by the synthetic id: %+v", m.browserActionHolds)
	}
	// the selector clips at 40 cols in the modal summary.
	m2 := New(&recBackend{}, nil)
	m2 = runMsg(t, m2, tea.WindowSizeMsg{Width: 140, Height: 30})
	longSel := "#" + strings.Repeat("a", 60)
	m2.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: longSel,
	})
	v2 := m2.permQ.view()
	wantClip := "click '" + clipRunes(longSel, 40) + "' on theboring.name"
	if v2 == nil || v2.Summary != wantClip {
		t.Fatalf("the selector clips at 40 cols: got %q, want %q", v2.Summary, wantClip)
	}
}

// TestBrowserActionApproveOnceFlow — the full approve path: the member's
// "once" answer runs the engine (bounded ctx), the agent receives the
// EXACT pinned follow-up, the member gets the dim result row, and the
// hold's id NEVER rides the backend's AnswerPermission wire.
func TestBrowserActionApproveOnceFlow(t *testing.T) {
	pinBrowserTextLane(t)
	var gotURL string
	var gotAction action.Action
	var gotDeadline bool
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		gotURL, gotAction = rawurl, a
		_, gotDeadline = ctx.Deadline()
		return &action.Result{URL: "https://theboring.name/after", Text: "clicked #buy"}, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})

	// the member answers "Allow once" on the modal.
	m = runMsg(t, m, permAnswerMsg{response: "once"})

	// the engine ran with the marker's payload under a bounded ctx.
	if gotURL != "https://theboring.name" || gotAction != (action.Action{Op: "click", Sel: "#buy"}) {
		t.Fatalf("the engine got the parked payload, got %q %+v", gotURL, gotAction)
	}
	if !gotDeadline {
		t.Fatal("the engine ctx must carry the app-side bound")
	}
	// the office-minted pid NEVER rides the backend's permission wire.
	if len(rb.answered) != 0 {
		t.Fatalf("the hold resolves LOCALLY — AnswerPermission must never fire, got %+v", rb.answered)
	}
	// the agent's follow-up, verbatim.
	if len(rb.sentTexts) != 1 {
		t.Fatalf("exactly ONE follow-up posts back to the agent, got %v", rb.sentTexts)
	}
	want := "[theboringfloor] browser-action ok on https://theboring.name/after: click '#buy'"
	if rb.sentTexts[0] != want {
		t.Fatalf("the approve follow-up:\n got %q\nwant %q", rb.sentTexts[0], want)
	}
	// the member's dim result row (the engine's own result string).
	if !lastOfficeNoticeHas(m, "browser: action ok on https://theboring.name/after: clicked #buy (approved by the member)") {
		t.Fatalf("the member result row must post, office rows: %+v", officeRows(m))
	}
	// the modal closed and the hold drained.
	if m.permQ.view() != nil || len(m.browserActionHolds) != 0 {
		t.Fatalf("the answer closes the modal + drains the hold: view %+v holds %v", m.permQ.view(), m.browserActionHolds)
	}
}

// TestBrowserActionAlwaysMapsToOnce — the modal's "Allow always" answer
// executes the action exactly ONCE (approve-once only: a mutating
// action never earns a standing grant) and still never rides the wire.
func TestBrowserActionAlwaysMapsToOnce(t *testing.T) {
	runs := 0
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		runs++
		return &action.Result{URL: rawurl, Text: "clicked #buy"}, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})
	m = runMsg(t, m, permAnswerMsg{response: "always"})
	if runs != 1 {
		t.Fatalf("always → once: the engine runs EXACTLY once, got %d", runs)
	}
	if len(rb.answered) != 0 {
		t.Fatalf("the always answer still never rides the wire, got %+v", rb.answered)
	}
	if len(rb.sentTexts) != 1 || rb.sentTexts[0] != "[theboringfloor] browser-action ok on https://theboring.name: click '#buy'" {
		t.Fatalf("the once-mapped follow-up posts, got %v", rb.sentTexts)
	}
}

// TestBrowserActionRejectFlow — the reject path: the engine NEVER runs,
// the agent receives the pinned REJECTED follow-up, and the member gets
// the dim rejection row.
func TestBrowserActionRejectFlow(t *testing.T) {
	called := false
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		called = true
		return nil, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})
	m = runMsg(t, m, permAnswerMsg{response: "reject"})

	if called {
		t.Fatal("a rejected action must never reach the engine")
	}
	if len(rb.answered) != 0 {
		t.Fatalf("the rejection is local too — AnswerPermission must never fire, got %+v", rb.answered)
	}
	if len(rb.sentTexts) != 1 {
		t.Fatalf("exactly ONE rejection follow-up posts, got %v", rb.sentTexts)
	}
	want := "[theboringfloor] browser-action REJECTED by the member: click '#buy' on https://theboring.name"
	if rb.sentTexts[0] != want {
		t.Fatalf("the reject follow-up:\n got %q\nwant %q", rb.sentTexts[0], want)
	}
	if !lastOfficeNoticeHas(m, "browser: action REJECTED by the member: click '#buy' on https://theboring.name") {
		t.Fatalf("the dim member row must post, office rows: %+v", officeRows(m))
	}
	for _, c := range officeRows(m) {
		if c.Meta == "error" {
			t.Fatalf("the rejection row is DIM, never red: %+v", c)
		}
	}
	if len(m.browserActionHolds) != 0 {
		t.Fatalf("the rejection drains the hold, got %v", m.browserActionHolds)
	}
}

// TestBrowserActionEngineFailureFlow — an engine failure (navigation /
// selector / timeout) rides the follow-up to the agent AND lands as the
// member's red row.
func TestBrowserActionEngineFailureFlow(t *testing.T) {
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		return nil, errors.New(`browser-action: https://theboring.name: selector "#buy" did not match a visible node within the 20s budget: context deadline exceeded`)
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})
	m = runMsg(t, m, permAnswerMsg{response: "once"})

	if len(rb.sentTexts) != 1 {
		t.Fatalf("the failure follow-up posts back to the agent, got %v", rb.sentTexts)
	}
	want := `[theboringfloor] browser-action failed on https://theboring.name: click '#buy' — browser-action: https://theboring.name: selector "#buy" did not match a visible node within the 20s budget: context deadline exceeded`
	if rb.sentTexts[0] != want {
		t.Fatalf("the failure follow-up:\n got %q\nwant %q", rb.sentTexts[0], want)
	}
	if !lastOfficeNoticeHas(m, `browser: action on https://theboring.name — browser-action: https://theboring.name: selector "#buy" did not match a visible node within the 20s budget`) {
		t.Fatalf("the member's red row must post, office rows: %+v", officeRows(m))
	}
	rows := officeRows(m)
	if rows[len(rows)-1].Meta != "error" {
		t.Fatalf("the failure row is RED, got %+v", rows[len(rows)-1])
	}
}

// TestBrowserActionEvalFollowup — the eval ok follow-up reports the
// JSON-stringified result; the member's row clips the JSON (the agent
// carries the full engine-capped payload).
func TestBrowserActionEvalFollowup(t *testing.T) {
	big := `"` + strings.Repeat("é", 200) + `"`
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		return &action.Result{URL: rawurl, Text: big}, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "eval", BrowserActionArg: "document.title",
	})
	if v := m.permQ.view(); v == nil || v.Summary != "eval document.title on theboring.name" {
		t.Fatalf("the eval modal summary: %+v", v)
	}
	m = runMsg(t, m, permAnswerMsg{response: "once"})

	want := "[theboringfloor] browser-action ok on https://theboring.name: eval → " + big
	if len(rb.sentTexts) != 1 || rb.sentTexts[0] != want {
		t.Fatalf("the eval follow-up carries the FULL JSON:\n got %q\nwant %q", rb.sentTexts, want)
	}
	// the member row clips the payload at 120 cols.
	if !lastOfficeNoticeHas(m, "browser: action ok on https://theboring.name: eval → "+clipRunes(big, 120)+" (approved by the member)") {
		t.Fatalf("the member row clips the payload, office rows: %+v", officeRows(m))
	}
}

// TestBrowserActionShapelessNoops — a shapeless action event (no URL /
// no op) degrades silent: no modal, no hold, no notice.
func TestBrowserActionShapelessNoops(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)
	for _, ev := range []state.Event{
		{Kind: state.EvBrowserAction, Text: "", BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#x"},
		{Kind: state.EvBrowserAction, Text: "https://x", BrowserOpenAllowed: true},
	} {
		if cmd := m.applyBrowserOpen(ev); cmd != nil {
			t.Fatalf("a shapeless action must no-op, got a cmd for %+v", ev)
		}
	}
	if m.permQ.view() != nil || len(m.browserActionHolds) != 0 || len(m.st.Chat) != before {
		t.Fatalf("shapeless events stay silent: view %+v holds %v new-rows %d",
			m.permQ.view(), m.browserActionHolds, len(m.st.Chat)-before)
	}
}

// TestBrowserActionEscThenAnswer — an esc'd browser-action modal parks
// (the hold survives the defer) and a later answer (via /perm's
// re-open) still resolves it — the modal's own lifecycle, no invented
// timeout.
func TestBrowserActionEscThenAnswer(t *testing.T) {
	runs := 0
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		runs++
		return &action.Result{URL: rawurl, Text: "clicked #buy"}, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})
	// esc defers the modal; the hold survives, nothing runs.
	m = runMsg(t, m, permLaterMsg{})
	if runs != 0 || m.permQ.view() != nil || len(m.browserActionHolds) != 1 {
		t.Fatalf("esc defers without executing: runs %d view %+v holds %v", runs, m.permQ.view(), m.browserActionHolds)
	}
	// /perm re-opens the esc'd ask; the answer resolves it.
	m = runMsg(t, m, slashMsg{text: "/perm"})
	if v := m.permQ.view(); v == nil || v.ToolName != "browser" {
		t.Fatalf("/perm re-opens the browser-action ask: %+v", m.permQ.view())
	}
	m = runMsg(t, m, permAnswerMsg{response: "once"})
	if runs != 1 {
		t.Fatalf("the deferred answer executes exactly once, got %d", runs)
	}
	if len(rb.sentTexts) != 1 || !strings.HasPrefix(rb.sentTexts[0], "[theboringfloor] browser-action ok on ") {
		t.Fatalf("the deferred approve posts the follow-up, got %v", rb.sentTexts)
	}
}

// TestBrowserActionStacksWithBackendAsk — a browser-action modal stacks
// in the SAME queue as a real backend ask: answering pops only the
// front; each id routes to its OWN resolution path (the backend wire
// for the real ask, the local hold for the action).
func TestBrowserActionStacksWithBackendAsk(t *testing.T) {
	runs := 0
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		runs++
		return &action.Result{URL: rawurl, Text: "clicked #buy"}, nil
	})

	rb := &permRecBackend{}
	m := New(rb, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	// a REAL backend ask first…
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-wire-1",
		EmployeeName: "boss", ToolName: "Bash", ToolSummary: "go build ./...", ToolState: "pending"})
	// …then the browser-action ask stacks behind it.
	m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy",
	})
	if v := m.permQ.view(); v == nil || v.ID != "perm-wire-1" || v.Total != 2 {
		t.Fatalf("the wire ask stays front at 1 of 2: %+v", m.permQ.view())
	}
	// answer the front (the wire ask) → rides AnswerPermission, no run.
	m = runMsg(t, m, permAnswerMsg{response: "once"})
	if len(rb.answered) != 1 || rb.answered[0] != [2]string{"perm-wire-1", "once"} {
		t.Fatalf("the wire ask rides the wire: %+v", rb.answered)
	}
	if runs != 0 {
		t.Fatal("the wire answer must never touch the action hold")
	}
	// the action ask advanced to the front; answering it resolves locally.
	if v := m.permQ.view(); v == nil || v.ToolName != "browser" || v.Total != 1 {
		t.Fatalf("the action ask advances: %+v", m.permQ.view())
	}
	m = runMsg(t, m, permAnswerMsg{response: "once"})
	if runs != 1 || len(rb.answered) != 1 {
		t.Fatalf("the action resolves locally: runs %d answered %+v", runs, rb.answered)
	}
}
