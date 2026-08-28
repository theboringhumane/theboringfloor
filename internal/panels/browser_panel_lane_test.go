// browser_panel_lane_test.go — the LIVE WIRING half of the premium lane:
// the pane itself consults the lane controller (no app glue, one level
// down): Open drives the resolve + spawn, View swaps the text body for
// the embedded PTY's screen model (badge + strip), keys forward through
// the controller's Write path while the pane's own q/esc still leave,
// SetSize SIGWINCHes the child, the fallback note surfaces through the
// pane's own note row with the text page warm underneath, and the
// kill-switched text lane stays byte-identical to the pre-lane viewer.
// The controller's fake spawn seam (browser_lane_test.go's
// fakeSpawnPins/pinKittyEnv) owns every child — no test owns a real PTY.
//
// The SHOT MODE half lives here too: the open → async-render → shot-mode
// flow against a fake headless engine (the seam swap), the failure-class
// rows, the kitty gate, the resize debounce, the floor-flip re-publish,
// and Close's registry clear.
package panels

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/headless"
)

// laneRig — a Browser wired for the lane matrix: stub fetch, the fake
// spawn seam capturing every minted child, the kitty host stub pinned.
type laneRig struct {
	b     *Browser
	made  *[]*fakeZenbuSess
	pages map[string]*Page
}

func newLaneRig(t *testing.T, pages map[string]*Page) *laneRig {
	t.Helper()
	pinKittyEnv(t)
	made := fakeSpawnPins(t)
	b := NewBrowser()
	b.SetSize(64, 16)
	b.fetchFn = func(rawurl string) (*Page, error) {
		p, ok := pages[rawurl]
		if !ok {
			return nil, errors.New("404: no route → go to " + rawurl)
		}
		return p, nil
	}
	return &laneRig{b: b, made: made, pages: pages}
}

// driveOpen runs Open + lands the produced BrowserPageMsg (navRig's
// idiom — the app's forwarding hop, one level down).
func (r *laneRig) driveOpen(t *testing.T, rawurl string) {
	t.Helper()
	cmd := r.b.Open(rawurl)
	if cmd == nil {
		t.Fatalf("Open(%q) returned no cmd", rawurl)
	}
	r.b.Update(cmd())
}

// TestBrowserPanelLaneEmbed — Open on a resolving host spawns the premium
// embed: the pane's View wears the " zenbu " badge + the exact strip +
// the child's painted rows, AND the text fetch still rode underneath (the
// page is stored — the fallback lands on a warm page, the ring stays
// meaningful).
func TestBrowserPanelLaneEmbed(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray"),
	})
	r.driveOpen(t, "https://a.dev/x")

	if !r.b.PremiumActive() || len(*r.made) != 1 {
		t.Fatalf("Open must spawn the premium embed: active=%v made=%d", r.b.PremiumActive(), len(*r.made))
	}
	if r.b.page == nil || r.b.page.Title != "Xray" {
		t.Fatalf("the text fetch rode under the embed (warm fallback): page %+v", r.b.page)
	}
	f := (*r.made)[0]
	if _, err := f.grid.Write([]byte("FAKE KITTY FRAME ROW")); err != nil {
		t.Fatalf("paint the fake grid: %v", err)
	}
	view := ansi.Strip(r.b.View())
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · https://a.dev/x", "FAKE KITTY FRAME ROW"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the premium pane frame carries %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Xray filler") {
		t.Fatalf("the text body never paints while premium runs:\n%s", view)
	}
}

// TestBrowserPanelLaneFallbackNote — an early death (<300ms) drops the
// embed THROUGH THE PANE's poll ride (View): the text lane returns with
// the fetched page warm (no re-fetch) and the dim fallback note on the
// pane's own note row; re-opening the latched url never re-spawns.
func TestBrowserPanelLaneFallbackNote(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray"),
	})
	r.driveOpen(t, "https://a.dev/x")
	f := (*r.made)[0]
	f.exited, f.code, f.life = true, 0, 100*time.Millisecond // early exit, code 0

	view := ansi.Strip(r.b.View()) // the View poll lands the fallback
	if r.b.PremiumActive() {
		t.Fatal("the early death must drop the embed")
	}
	for _, want := range []string{"zenbu exited (0) — falling back to text mode", "Xray filler 00", "▸ https://a.dev/x"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the fallback pane frame carries %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "zenbu terminal-browser ·") {
		t.Fatalf("the fallback drops the premium strip:\n%s", view)
	}
	// the no-flap latch: a re-open of the fell-back url stays text.
	r.driveOpen(t, "https://a.dev/x")
	if len(*r.made) != 1 || r.b.PremiumActive() {
		t.Fatalf("a fell-back url never re-spawns: made=%d active=%v", len(*r.made), r.b.PremiumActive())
	}
}

// TestBrowserPanelLaneKeys — while premium runs: unclaimed keys forward
// to the child through the controller's Write path (keyToBytes), the text
// pane's own surface (link cursor, scroll) is bypassed, the wheel is
// swallowed, and q/esc still leave (the app's suspend hook rides the
// leave — the pane emits BrowserLeaveMsg exactly like the text lane).
func TestBrowserPanelLaneKeys(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray",
			BrowserLink{N: 1, URL: "https://a.dev/y", Text: "why"}),
	})
	r.driveOpen(t, "https://a.dev/x")
	f := (*r.made)[0]
	if r.b.cursor != 0 {
		t.Fatalf("setup: the text lane focused link [1], cursor %d", r.b.cursor)
	}

	// "j" forwards as the raw byte — the text link cursor never moves.
	if cmd := r.b.Update(browserKey("j")); cmd != nil {
		t.Fatalf("a forwarded key returns no cmd")
	}
	if f.writes != 1 {
		t.Fatalf("j must reach the child once, writes=%d", f.writes)
	}
	if r.b.cursor != 0 {
		t.Fatalf("the text-lane cursor is bypassed while premium runs, cursor %d", r.b.cursor)
	}
	// pgup rides the keyToBytes matrix too; the wheel is swallowed.
	r.b.Update(browserKey("pgup"))
	if f.writes != 2 {
		t.Fatalf("pgup must reach the child, writes=%d", f.writes)
	}
	r.b.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	if f.writes != 2 {
		t.Fatalf("the wheel never reaches the child, writes=%d", f.writes)
	}
	// q and esc still leave (the app's SuspendLane hook rides the leave).
	for _, k := range []string{"q", "esc"} {
		cmd := r.b.Update(browserKey(k))
		if cmd == nil {
			t.Fatalf("%s while premium must produce the leave cmd", k)
		}
		if _, ok := cmd().(BrowserLeaveMsg); !ok {
			t.Fatalf("%s while premium must produce BrowserLeaveMsg, got %T", k, cmd())
		}
	}
}

// TestBrowserPanelLaneResize — a pane SetSize SIGWINCHes the live child:
// the strip + note rows stay reserved, the PTY takes the rest.
func TestBrowserPanelLaneResize(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray"),
	})
	r.driveOpen(t, "https://a.dev/x")
	f := (*r.made)[0]
	if f.cols != 64 || f.rows != 14 {
		t.Fatalf("the spawn sized the PTY to the pane body: %dx%d, want 64x14", f.cols, f.rows)
	}
	r.b.SetSize(40, 10)
	if f.cols != 40 || f.rows != 8 {
		t.Fatalf("SetSize must resize the live PTY: %dx%d, want 40x8", f.cols, f.rows)
	}
	cols, rows, ok := r.b.LaneSessionSize()
	if !ok || cols != 40 || rows != 8 {
		t.Fatalf("LaneSessionSize = %dx%d ok=%v, want 40x8 true", cols, rows, ok)
	}
}

// TestBrowserPanelLaneSuspendResume — the app's slot-flip hooks: Suspend
// FREEZES the embed silently (keep-alive — no kill, no note, the pane's
// premium chrome hides behind the floor), Resume THAWS the SAME session
// (no respawn — made stays 1, no new lane-history entry), Close seals
// the controller.
func TestBrowserPanelLaneSuspendResume(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray"),
	})
	r.driveOpen(t, "https://a.dev/x")

	r.b.SuspendLane()
	if !(*r.made)[0].frozen {
		t.Fatal("SuspendLane freezes the embedded child (SIGSTOP, keep-alive)")
	}
	if (*r.made)[0].closed {
		t.Fatal("SuspendLane must NOT kill the child (keep-alive)")
	}
	if r.b.PremiumActive() {
		t.Fatal("a frozen child hides the premium chrome behind the floor")
	}
	if !r.b.LaneSuspended() {
		t.Fatal("the pane reads the keep-alive posture")
	}
	if n := r.b.LaneNote(); n != "" {
		t.Fatalf("a pane switch is not a failure — no note: %q", n)
	}

	r.b.ResumeLane()
	if len(*r.made) != 1 || !r.b.PremiumActive() || r.b.LaneSuspended() {
		t.Fatalf("ResumeLane thaws the SAME embed (no respawn): made=%d active=%v suspended=%v",
			len(*r.made), r.b.PremiumActive(), r.b.LaneSuspended())
	}

	r.b.Close()
	if !(*r.made)[0].closed {
		t.Fatal("Close reaps the live embed at pane teardown")
	}
}

// ---------------------------------------------------------------------------
// SHOT MODE (the headless screenshot lane)
// ---------------------------------------------------------------------------

// shotEngine — the fake headless engine for the pane suite: a call
// recorder (count + the last render's url/dims) behind the seam swap.
// The cmds drive synchronously on the test goroutine (the house cmd-drain
// idiom), so the recorder needs no locking.
type shotEngine struct {
	calls        int
	lastURL      string
	lastW, lastH int
	res          *headless.Result
	err          error
}

// pinShotEngine swaps the headless seam (available = the probe's verdict)
// and returns the recorder. NO live chrome — ever.
func pinShotEngine(t *testing.T, available bool) *shotEngine {
	t.Helper()
	e := &shotEngine{}
	restore := SetHeadlessForShot(
		func() (string, bool) {
			if available {
				return "/fake/chrome", true
			}
			return "", false
		},
		func(_ context.Context, rawurl string, w, h int) (*headless.Result, error) {
			e.calls++
			e.lastURL, e.lastW, e.lastH = rawurl, w, h
			return e.res, e.err
		})
	t.Cleanup(restore)
	return e
}

// shotRig — a Browser wired for the shot matrix: the kitty display lane
// (or its non-kitty twin) pinned, the lane controller's kill-switch armed
// (the zenbu child never interferes — shot mode owns the lane story
// here), a stub fetch, the fake engine, and a scratch shots dir.
type shotRig struct {
	b *Browser
	e *shotEngine
}

func newShotRig(t *testing.T, kitty bool, available bool) *shotRig {
	t.Helper()
	pinLaneDetectEnv(t, kitty) // BEFORE the pane builds (the creation pin)
	t.Setenv(BrowserLaneOffEnv, "1")
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	e := pinShotEngine(t, available)
	b := NewBrowser()
	b.SetSize(64, 16)
	b.fetchFn = func(rawurl string) (*Page, error) { return navPage(rawurl, "Xray"), nil }
	return &shotRig{b: b, e: e}
}

// driveShotOpen — Open + land the fetch + land the render verdict (the
// app's runMsg drain, one level down: Open's cmd carries the fetch; the
// fetch's landing arms the render cmd; the render's msg lands the shot).
func (r *shotRig) driveShotOpen(t *testing.T, rawurl string) {
	t.Helper()
	cmd := r.b.Open(rawurl)
	if cmd == nil {
		t.Fatalf("Open(%q) returned no cmd", rawurl)
	}
	shotCmd := r.b.Update(cmd())
	if shotCmd == nil {
		t.Fatalf("the fetch's landing must arm the headless render")
	}
	r.b.Update(shotCmd())
}

// TestBrowserShotOpenFlow — the happy path END TO END through the pane:
// Open shows the text lane immediately (never blank), the fetch's landing
// arms the render at the pane box's EXACT pixel dims, the dim
// "rendering <url>…" row rides while the engine works, and the landed
// verdict enters SHOT MODE — the " shot " badge + the "▸ headless
// chromium · <redirect-final url>" strip, the saved PNG's path on the
// note row, the text page warm underneath (the ring recorded the open),
// and the registry contribution pinned (content id, (0,0), f=100, no
// c=/r=).
func TestBrowserShotOpenFlow(t *testing.T) {
	r := newShotRig(t, true, true)
	png := shotTestPNG(t)
	r.e.res = &headless.Result{URL: "https://a.dev/x-final", Title: "Xray Final", PNG: png}

	cmd := r.b.Open("https://a.dev/x")
	if cmd == nil {
		t.Fatal("Open must produce the fetch cmd")
	}
	// IMMEDIATE: the text lane shows through (never blank — the starter
	// card until the fetch lands), no shot chrome yet.
	if r.b.ShotActive() {
		t.Fatal("no shot mode before the render lands")
	}
	if view := ansi.Strip(r.b.View()); !strings.Contains(view, browserStarterCard) || strings.Contains(view, " shot ") {
		t.Fatalf("the immediate frame is the text lane's (never blank, no shot chrome):\n%s", view)
	}
	shotCmd := r.b.Update(cmd())
	if shotCmd == nil {
		t.Fatal("the fetch's landing arms the headless render")
	}
	// the render is in flight: the dim "rendering <url>…" row rides…
	if view := ansi.Strip(r.b.View()); !strings.Contains(view, "rendering https://a.dev/x…") {
		t.Fatalf("the in-flight render wears the dim row:\n%s", view)
	}
	if r.e.calls != 0 {
		t.Fatalf("the render rides the cmd (never the UI goroutine): calls %d", r.e.calls)
	}
	// …and it fires at the pane box's EXACT pixel dims: 64 cols × 9 =
	// 576, (16-2) body rows × 18 = 252.
	r.b.Update(shotCmd())
	if r.e.calls != 1 || r.e.lastURL != "https://a.dev/x" || r.e.lastW != 576 || r.e.lastH != 252 {
		t.Fatalf("the render fired %d× (%s @ %dx%d), want 1× (https://a.dev/x @ 576x252)",
			r.e.calls, r.e.lastURL, r.e.lastW, r.e.lastH)
	}

	// SHOT MODE: badge + strip (the REDIRECT-FINAL url) + the path row.
	if !r.b.ShotActive() {
		t.Fatal("the landed render enters shot mode on the kitty lane")
	}
	view := ansi.Strip(r.b.View())
	for _, want := range []string{" shot ", "▸ headless chromium · https://a.dev/x-final", "screenshot: "} {
		if !strings.Contains(view, want) {
			t.Fatalf("the shot-mode frame carries %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Xray filler") || strings.Contains(view, "rendering ") {
		t.Fatalf("the shot region's body is the PNG's (no text rows):\n%s", view)
	}
	// the strip is the pane's row 0 (the badge + the headless strip).
	if lines := strings.Split(view, "\n"); len(lines) == 0 || !strings.Contains(lines[0], "headless chromium") {
		t.Fatalf("the shot strip rides the pane's row 0:\n%s", view)
	}

	// the text model rode underneath: the page is warm, the ring
	// recorded the open, [n] nav stays on the text lane's model.
	if r.b.page == nil || r.b.page.Title != "Xray" {
		t.Fatalf("the text fetch rode underneath (warm nav/history): page %+v", r.b.page)
	}
	if len(r.b.hist) != 1 || r.b.hist[0].url != "https://a.dev/x" {
		t.Fatalf("the history ring recorded the open: %+v", r.b.hist)
	}

	// the saved PNG: the convention's path, the bytes on disk.
	path := r.b.ShotPath()
	if path == "" || !strings.Contains(path, string(filepath.Join("shots"))+string(os.PathSeparator)) || !strings.HasSuffix(path, ".png") {
		t.Fatalf("the saved PNG rides the convention's path: %q", path)
	}
	if back, err := os.ReadFile(path); err != nil || len(back) != len(png) {
		t.Fatalf("the saved PNG's bytes round-trip: %v (%d)", err, len(back))
	}
	if !strings.Contains(view, "screenshot: /") {
		t.Fatalf("the note row carries the saved path (ansi-truncated at the pane width):\n%s", view)
	}

	// the registry contribution: the content-addressed id at pane-local
	// (0,0), f=100, NO c=/r=.
	imgs := r.b.ShotFrameState()
	if len(imgs) != 1 {
		t.Fatalf("the shot publishes exactly one image: %+v", imgs)
	}
	im := imgs[0]
	if im.OX != 0 || im.OY != 0 {
		t.Fatalf("the shot anchors at pane-local (0,0): (%d,%d)", im.OX, im.OY)
	}
	if im.OfficeID != KittyImageID(png) {
		t.Fatalf("the office id is content-addressed: %08x, want %08x", im.OfficeID, KittyImageID(png))
	}
	if !strings.Contains(im.Frame, ",f=100;") || strings.Contains(im.Frame, ",c=") || strings.Contains(im.Frame, ",r=") {
		t.Fatalf("the published frame is f=100 with NO c=/r= keys: %q…", im.Frame[:80])
	}
}

// TestBrowserShotFailureClasses — chrome-missing / refused / timeout /
// generic: each lands its EXACT dim row, the pane stays text, the fetched
// page is warm underneath, and the shot chrome never paints.
func TestBrowserShotFailureClasses(t *testing.T) {
	cases := []struct {
		name  string
		avail bool
		err   error
		want  string
	}{
		{"chrome missing", false, nil, shotFailChromeCopy},
		{"navigation refused", true, &headless.PolicyError{URL: "http://x.test/", Reason: "plain http to x.test refused"},
			"text lane — headless render refused: plain http to x.test refused"},
		{"timeout", true, context.DeadlineExceeded, "text lane — headless render timed out (15s)"},
		{"generic engine failure", true, errors.New("kaboom"), "text lane — headless render failed: kaboom"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newShotRig(t, true, tc.avail)
			r.b.SetSize(150, 16) // wide enough for the full untruncated copy
			r.e.err = tc.err
			r.driveShotOpen(t, "https://a.dev/x")
			if r.b.ShotActive() {
				t.Fatal("a failed render never enters shot mode")
			}
			view := ansi.Strip(r.b.View())
			if !strings.Contains(view, tc.want) {
				t.Fatalf("the failure lands its exact dim row:\nwant %q\nview:\n%s", tc.want, view)
			}
			if !strings.Contains(view, "Xray filler 00") {
				t.Fatalf("the text page stays warm under the failure:\n%s", view)
			}
			for _, never := range []string{" shot ", "▸ headless chromium"} {
				if strings.Contains(view, never) {
					t.Fatalf("the failure never wears shot chrome %q:\n%s", never, view)
				}
			}
			if r.b.LaneNote() != "" {
				t.Fatalf("a shot failure is not a lane note: %q", r.b.LaneNote())
			}
		})
	}
}

// TestBrowserShotNonKitty — the display gate: a successful render on a
// NON-kitty host NEVER enters shot mode — the text lane paints with the
// dim "screenshot: <path>" row (the save still lands for the member's
// `o`-to-open habit).
func TestBrowserShotNonKitty(t *testing.T) {
	r := newShotRig(t, false, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")

	if r.b.ShotActive() {
		t.Fatal("non-kitty hosts never enter shot mode")
	}
	path := r.b.ShotPath()
	if path == "" {
		t.Fatal("the save still lands on a non-kitty host")
	}
	view := ansi.Strip(r.b.View())
	// the dim row carries the path (ansi-truncated at the pane width —
	// the FULL path reads through the pane's seam + exists on disk).
	if !strings.Contains(view, "screenshot: /") {
		t.Fatalf("the non-kitty frame carries the screenshot row (path prefix):\n%s", view)
	}
	for _, want := range []string{"Xray filler 00", "▸ https://a.dev/x"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the non-kitty frame carries %q:\n%s", want, view)
		}
	}
	for _, never := range []string{" shot ", "▸ headless chromium"} {
		if strings.Contains(view, never) {
			t.Fatalf("the non-kitty lane never wears shot chrome %q:\n%s", never, view)
		}
	}
	if got := r.b.ShotFrameState(); got != nil {
		t.Fatalf("a non-kitty host publishes NOTHING to the registry: %+v", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the saved PNG exists for the `o` habit: %v", err)
	}
}

// TestBrowserShotStaleDropped — two racing opens: the superseded open's
// render verdict drops on the generation guard (the CURRENT open's shot
// survives).
func TestBrowserShotStaleDropped(t *testing.T) {
	r := newShotRig(t, true, true)
	pngA := shotTestPNG(t)
	pngB := append(append([]byte{}, pngA...), 0x42) // a DIFFERENT png → a different id

	// open A: the fetch lands, the render arms — but A's cmd is HELD.
	cmdA := r.b.Open("https://a.dev/a")
	shotCmdA := r.b.Update(cmdA())
	// open B supersedes before A's render lands.
	r.e.res = &headless.Result{URL: "https://a.dev/b", Title: "B", PNG: pngB}
	cmdB := r.b.Open("https://a.dev/b")
	shotCmdB := r.b.Update(cmdB())
	r.b.Update(shotCmdB()) // B's render lands → B's shot paints
	if got := r.b.shot.url; got != "https://a.dev/b" {
		t.Fatalf("B's shot paints: %q", got)
	}
	// A's verdict lands LATE: the generation guard drops it.
	r.e.res = &headless.Result{URL: "https://a.dev/a", Title: "A", PNG: pngA}
	r.b.Update(shotCmdA())
	if got := r.b.shot.url; got != "https://a.dev/b" {
		t.Fatalf("the stale render dropped (still B): %q", got)
	}
	if r.b.shot.officeID != KittyImageID(pngB) {
		t.Fatalf("the displayed PNG is still B's: %08x", r.b.shot.officeID)
	}
}

// TestBrowserShotResizeDebounce — the resize re-render: SetSize stamps,
// the NEXT routed msg arms the 300ms quiet-window tick, and the tick's
// landing fires at the NEW box's exact dims. A resize landing BETWEEN the
// arm and the tick's landing re-arms instead of firing (the burst's
// quiet window — the stamp re-check), and the settled tick fires once.
func TestBrowserShotResizeDebounce(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	if r.e.calls != 1 {
		t.Fatalf("setup: one render, got %d", r.e.calls)
	}

	// the SETTLED leg: one resize, one msg arms, the tick fires once at
	// the NEW box (80 cols × 9 = 720, (20-2) body rows × 18 = 324).
	r.b.SetSize(80, 20)
	if r.e.calls != 1 {
		t.Fatalf("a resize never re-renders synchronously: calls %d", r.e.calls)
	}
	armed := r.b.Update(browserKey("j")) // the sweep arms the debounce tick
	if armed == nil {
		t.Fatal("the routed msg arms the debounce tick")
	}
	fire := r.b.Update(armed()) // the tick lands (300ms) → FIRES
	if fire == nil {
		t.Fatal("the settled debounce fires the re-render")
	}
	r.b.Update(fire())
	if r.e.calls != 2 || r.e.lastW != 720 || r.e.lastH != 324 {
		t.Fatalf("the settled re-render fired %d× (%dx%d), want 2× (720x324)", r.e.calls, r.e.lastW, r.e.lastH)
	}
	if !r.b.ShotActive() {
		t.Fatal("the re-render's verdict keeps shot mode")
	}

	// the BURST leg: a resize lands BETWEEN the arm and the tick's
	// landing — the stale-armed tick RE-ARMS instead of firing (no
	// render), and the settled re-arm fires once at the final box
	// (100 cols × 9 = 900, (28-2) body rows × 18 = 468).
	r.b.SetSize(90, 24)                   // stamp T1
	armed2 := r.b.Update(browserKey("j")) // tick armed AT T1
	r.b.SetSize(100, 28)                  // stamp T2 — mid-flight
	reArm := r.b.Update(armed2())         // tick lands: armedAt(T1) != T2 → RE-ARM
	if reArm == nil {
		t.Fatal("the stale-armed tick re-arms (the burst's quiet window)")
	}
	if r.e.calls != 2 {
		t.Fatalf("the stale tick never fires: calls %d, want 2", r.e.calls)
	}
	fire2 := r.b.Update(reArm()) // the re-armed tick settles → FIRES
	if fire2 == nil {
		t.Fatal("the re-armed debounce fires")
	}
	r.b.Update(fire2())
	if r.e.calls != 3 || r.e.lastW != 900 || r.e.lastH != 468 {
		t.Fatalf("the burst converged to ONE re-render %d× (%dx%d), want 3× (900x468)", r.e.calls, r.e.lastW, r.e.lastH)
	}
}

// TestBrowserShotSuspendResumeRepublish — the keep-alive flip: Suspend +
// Resume need NOTHING from the pane — the registry contribution re-reads
// the CACHED bytes byte-identically and the engine NEVER re-fires (no
// re-render on the flip cycle).
func TestBrowserShotSuspendResumeRepublish(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	before := r.b.ShotFrameState()
	if len(before) != 1 {
		t.Fatalf("setup: one published image, got %d", len(before))
	}

	r.b.SuspendLane()
	r.b.ResumeLane()

	after := r.b.ShotFrameState()
	if len(after) != 1 || after[0].OfficeID != before[0].OfficeID || after[0].Frame != before[0].Frame {
		t.Fatalf("the flip re-publishes the CACHED bytes byte-identically:\n before %+v\n after  %+v", before, after)
	}
	if r.e.calls != 1 {
		t.Fatalf("the flip NEVER re-renders: calls %d, want 1", r.e.calls)
	}
	if !r.b.ShotActive() {
		t.Fatal("shot mode survives the floor flip")
	}
}

// TestBrowserShotCloseClears — pane teardown: the shot state drops and
// the registry clears on the spot (the quit path's final flush must find
// nothing to re-emit — the zenbu session Close's exact contract).
func TestBrowserShotCloseClears(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	// the app's publish seam's pane half: the registry holds the shot.
	ZenbuRegistry().Publish(true, 0, 3, r.b.ShotFrameState(), nil)
	t.Cleanup(ZenbuRegistry().Clear)

	r.b.Close()
	if got := r.b.ShotFrameState(); got != nil {
		t.Fatalf("Close drops the shot state: %+v", got)
	}
	active, _, _, imgs, _ := ZenbuRegistry().snapshot()
	if active || len(imgs) != 0 {
		t.Fatalf("Close clears the registry on the spot: active=%v imgs=%d", active, len(imgs))
	}
}

// TestBrowserShotNewOpenDeletes — a fresh open LEAVES shot mode at once
// (the text lane shows through, never a stale PNG under a new bar) and
// the registry contribution empties until the new render lands.
func TestBrowserShotNewOpenDeletes(t *testing.T) {
	r := newShotRig(t, true, true)
	r.e.res = &headless.Result{URL: "https://a.dev/x", Title: "Xray", PNG: shotTestPNG(t)}
	r.driveShotOpen(t, "https://a.dev/x")
	if !r.b.ShotActive() {
		t.Fatal("setup: shot mode live")
	}

	cmd := r.b.Open("https://a.dev/y")
	if r.b.ShotActive() {
		t.Fatal("a fresh open leaves shot mode immediately (never a stale PNG)")
	}
	if got := r.b.ShotFrameState(); got != nil {
		t.Fatalf("the registry contribution empties on the new open: %+v", got)
	}
	if view := ansi.Strip(r.b.View()); strings.Contains(view, "headless chromium") {
		t.Fatalf("the old shot's chrome drops with the open:\n%s", view)
	}
	r.e.res = &headless.Result{URL: "https://a.dev/y", Title: "Why", PNG: shotTestPNG(t)}
	shotCmd := r.b.Update(cmd()) // the fetch lands → the render arms
	r.b.Update(shotCmd())        // the render lands
	if got := r.b.shot.url; got != "https://a.dev/y" {
		t.Fatalf("the new page's shot paints: %q", got)
	}
}

// TestBrowserPanelTextLaneByteIdentical — the universal default: with the
// kill-switch armed the pane NEVER spawns and the render is the
// pre-lane text viewer verbatim (no badge, no strip — the starter card,
// the location bar, the page rows).
func TestBrowserPanelTextLaneByteIdentical(t *testing.T) {
	pinKittyEnv(t)
	t.Setenv(BrowserLaneOffEnv, "1") // the lane is OFF — the text viewer owns
	made := fakeSpawnPins(t)

	b := NewBrowser()
	b.SetSize(64, 16)
	if idle := ansi.Strip(b.View()); !strings.Contains(idle, browserStarterCard) || strings.Contains(idle, " zenbu ") {
		t.Fatalf("the idle pane is the text starter card, no lane chrome:\n%s", idle)
	}
	b.fetchFn = func(rawurl string) (*Page, error) {
		return navPage(rawurl, "Xray"), nil
	}
	cmd := b.Open("https://a.dev/x")
	b.Update(cmd())
	if len(*made) != 0 {
		t.Fatalf("the kill-switched lane never spawns, made=%d", len(*made))
	}
	if b.PremiumActive() {
		t.Fatal("the kill-switched lane never goes premium")
	}
	view := ansi.Strip(b.View())
	for _, want := range []string{"▸ https://a.dev/x", "Xray filler 00"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the text lane renders %q:\n%s", want, view)
		}
	}
	for _, never := range []string{" zenbu ", "zenbu terminal-browser"} {
		if strings.Contains(view, never) {
			t.Fatalf("the text lane never wears lane chrome %q:\n%s", never, view)
		}
	}
	// keys are the pane's OWN surface (the link cursor moves on j).
	b2page := navPage("https://a.dev/y", "Why",
		BrowserLink{N: 1, URL: "https://a.dev/x", Text: "ex"},
		BrowserLink{N: 2, URL: "https://a.dev/z", Text: "zed"})
	b.fetchFn = func(rawurl string) (*Page, error) { return b2page, nil }
	b.Update(b.Open("https://a.dev/y")())
	if b.cursor != 0 {
		t.Fatalf("setup: link [1] focused, cursor %d", b.cursor)
	}
	b.Update(browserKey("j"))
	if b.cursor != 1 {
		t.Fatalf("text-lane j moves the pane's own link cursor, got %d", b.cursor)
	}
	if got := len(*made); got != 0 {
		t.Fatalf("text-lane keys never spawn: made=%d", got)
	}
}
