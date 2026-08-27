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
package panels

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
// kills the embed silently (no note), Resume re-spawns for the SAME url
// (no new lane-history entry), Close seals the controller.
func TestBrowserPanelLaneSuspendResume(t *testing.T) {
	r := newLaneRig(t, map[string]*Page{
		"https://a.dev/x": navPage("https://a.dev/x", "Xray"),
	})
	r.driveOpen(t, "https://a.dev/x")

	r.b.SuspendLane()
	if !(*r.made)[0].closed {
		t.Fatal("SuspendLane group-kills the embedded child")
	}
	if r.b.PremiumActive() {
		t.Fatal("SuspendLane drops the pane back to the text lane")
	}
	if n := r.b.LaneNote(); n != "" {
		t.Fatalf("a pane switch is not a failure — no note: %q", n)
	}

	r.b.ResumeLane()
	if len(*r.made) != 2 || !r.b.PremiumActive() {
		t.Fatalf("ResumeLane re-spawns the embed for the same url: made=%d active=%v", len(*r.made), r.b.PremiumActive())
	}

	r.b.Close()
	if !(*r.made)[1].closed {
		t.Fatal("Close reaps the live embed at pane teardown")
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
