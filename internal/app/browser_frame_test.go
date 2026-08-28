// browser_frame_test.go — the frame-splice REGISTRY's app-level
// contract (the wave-81 emission redesign; the wrapper's byte-level
// suite lives in internal/panels' zenbu_frame_test.go): a Model at a
// KNOWN size with the premium lane live publishes the pane's ABSOLUTE
// cell origin + the live images; the tea.WithOutput wrapper's emitted
// CUP bytes are asserted for a KNOWN pane-local offset in BOTH the
// desktop and the mobile geometry — and the floor/leave posture clears
// the registry (the wrapper then passes frames through untouched).
package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
)

// frameTestPayload — the deterministic fake frame (content irrelevant —
// the office never image-decodes it; the bytes only round-trip base64 +
// hash).
var frameTestPayload = []byte("\x89PNG\r\n\x1a\nAPPFRAMEKITTY0123456789abcdefghijklmnopqrstuvwxyz")

// plantKittyFrameFake — the kitty-streaming fake terminal-browser: homes
// the cursor, paints a toolbar row, emits ONE chunked kitty frame
// (m=1/m=0, the commit landing with the cursor at pane-local (0,1)),
// paints the marker, parks. The pinned-PATH discipline is
// plantFakeTerminalBrowser's.
func plantKittyFrameFake(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	b64 := base64.StdEncoding.EncodeToString(frameTestPayload)
	bin := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:7] + "\\033\\\\'\n" +
		"printf '\\033_Gm=0;" + b64[7:] + "\\033\\\\'\n" +
		"printf '\\033[3;1H'\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(bin), 0o755); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// frameTestWrapper — ONE wrapper over a buffer on the SHARED registry,
// with the lane's direct-emit seam wired to its DirectEmit — the EXACT
// production shape (cmd/theboringoffice's main): one wrapper for the
// process's lifetime, every lane delete serialized through it.
func frameTestWrapper(t *testing.T) (w *panels.ZenbuFrameWriter, out *strings.Builder) {
	t.Helper()
	out = &strings.Builder{}
	w = panels.NewZenbuFrameWriter(out, panels.ZenbuRegistry())
	restore := panels.SetZenbuEmitForShot(w.DirectEmit)
	t.Cleanup(restore)
	return w, out
}

// frameSpliceOut — drive one renderer flush through the wrapper.
func frameSpliceOut(t *testing.T, w *panels.ZenbuFrameWriter, out *strings.Builder, flush string) string {
	t.Helper()
	out.Reset()
	if _, err := w.Write([]byte(flush)); err != nil {
		t.Fatalf("the wrapper write: %v", err)
	}
	return out.String()
}

// frameTestDeleteFrame — the office-side a=d for the fixture frame's
// STABLE office id (ZenbuOfficeID over child i=1 + placement 0 — the
// wave-82 fix; the content-hash id was the flicker bug).
func frameTestDeleteFrame() string {
	return "\x1b_Ga=d,d=I,i=" + panels.KittyIDHash8(panels.ZenbuOfficeID(1, 0)) + ",q=2;\x1b\\"
}

// assertFrameSplice — the shared geometry gate: after /open + paint, ONE
// flush through the wrapper must carry EXACTLY cursor-save + CUP(5;1)
// (registry origin (0,3) + the pane-local commit (0,1), 1-based) + the
// office-id APC (the STABLE id + the child's payload verbatim + the
// pane's body box c=wantCols,r=wantRows — the wave-82 pane-exact sizing)
// + cursor-restore.
func assertFrameSplice(t *testing.T, tag, got string, wantCols, wantRows int) {
	t.Helper()
	hash8 := panels.KittyIDHash8(panels.ZenbuOfficeID(1, 0))
	wantAPC := "\x1b_Ga=T,t=d,q=2,C=1,i=" + hash8 + ",f=100,c=" + fmt.Sprintf("%d", wantCols) + ",r=" + fmt.Sprintf("%d", wantRows) + ";" +
		base64.StdEncoding.EncodeToString(frameTestPayload) + "\x1b\\"
	want := "FLUSH" + "\x1b7\x1b[5;1H" + wantAPC + "\x1b8"
	if got != want {
		t.Fatalf("%s: the wrapper's emitted bytes for pane-local (0,1) @ origin (0,3):\n got %q\nwant %q", tag, got, want)
	}
}

// TestZenbuFramePublishDesktop — the DESKTOP geometry (140x30): the left
// slot at x=0, the grid's absolute origin (0,3) — topbar 1 + switcher
// strip 1 + the RegionView badge row 1. The leave then FREEZES the lane
// (keep-alive: NO direct emit — the child lives on): the registry clear
// makes the wrapper's emitted-set diff flush the a=d on the next frame;
// the office-quit path (ctrl+c → Close) still flushes the delete
// DIRECTLY through the seam (redundant, q=2-hushed no-ops).
func TestZenbuFramePublishDesktop(t *testing.T) {
	pinBrowserLaneEnv(t)
	plantKittyFrameFake(t)
	panels.ZenbuRegistry().Clear()
	t.Cleanup(panels.ZenbuRegistry().Clear)
	w, out := frameTestWrapper(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	if m.mobile() {
		t.Fatal("140 cols is the desktop layout")
	}
	m = runMsg(t, m, slashMsg{text: "/open " + laneFixtureURL(t)})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	_ = m.Frame() // renders + publishes the registry
	// the lane's body box: resize() gave the browser (floorW, middleH-1);
	// the controller reserves the strip + note rows → bodyH = middleH-3.
	assertFrameSplice(t, "desktop", frameSpliceOut(t, w, out, "FLUSH"), m.floorW, m.middleH-3)

	// the leave: the lane FREEZES (keep-alive) — the direct seam stays
	// SILENT (the child lives on); the registry clear makes the wrapper's
	// emitted-set diff flush exactly one a=d on the next frame.
	out.Reset()
	pid := m.BrowserLanePid()
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if m.BrowserPremiumActive() {
		t.Fatal("q left + suspended the premium session")
	}
	if !m.BrowserLaneSuspended() {
		t.Fatal("q FREEZES the lane (the keep-alive posture)")
	}
	if got := out.String(); got != "" {
		t.Fatalf("a freeze direct-emits NOTHING (the delete rides the wrapper's diff): %q", got)
	}
	_ = m.Frame() // publishes the empty state
	if got := frameSpliceOut(t, w, out, "F2"); got != "F2"+frameTestDeleteFrame() {
		t.Fatalf("the wrapper's diff flushes the emitted id once:\n got %q\nwant %q", got, "F2"+frameTestDeleteFrame())
	}
	if got := frameSpliceOut(t, w, out, "F3"); got != "F3" {
		t.Fatalf("…and the next flush passes through clean: %q", got)
	}

	// the quit path: ctrl+c → closeBrowser → the lane's Close — the
	// frozen child is reaped AND its delete rides the direct seam.
	out.Reset()
	_ = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if got := out.String(); got != frameTestDeleteFrame() {
		t.Fatalf("the quit's lane Close flushed the delete through the direct seam:\n got %q\nwant %q", got, frameTestDeleteFrame())
	}
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("the quit path reaps the frozen child: kill(%d, 0) = %v", pid, err)
	}
}

// TestZenbuFramePublishMobile — the MOBILE geometry (60x30 — under
// mobileMaxCols): the browser rides the top band (plan never covers it),
// and the stack above the grid is the SAME three rows (topbar + switcher
// strip + RegionView badge) — the origin stays (0,3).
func TestZenbuFramePublishMobile(t *testing.T) {
	pinBrowserLaneEnv(t)
	plantKittyFrameFake(t)
	panels.ZenbuRegistry().Clear()
	t.Cleanup(panels.ZenbuRegistry().Clear)
	w, out := frameTestWrapper(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 60, Height: 30})
	if !m.mobile() {
		t.Fatal("60 cols is the mobile layout")
	}
	m = runMsg(t, m, slashMsg{text: "/open " + laneFixtureURL(t)})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	_ = m.Frame()
	// mobile: the browser rides the band — SetSize(width, floorBandH()-1);
	// the controller's strip + note rows → bodyH = floorBandH()-3.
	assertFrameSplice(t, "mobile", frameSpliceOut(t, w, out, "FLUSH"), m.width, m.floorBandH()-3)
}

// TestZenbuFramePublishInactive — the floor posture: no /open (or the
// text lane) publishes NOTHING — the wrapper passes frames through
// byte-identically (the "registry empty ⇒ emit nothing" contract).
func TestZenbuFramePublishInactive(t *testing.T) {
	pinBrowserLaneEnv(t)
	plantKittyFrameFake(t)
	panels.ZenbuRegistry().Clear()
	t.Cleanup(panels.ZenbuRegistry().Clear)
	w, out := frameTestWrapper(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	_ = m.Frame() // the floor shows — no lane
	if got := frameSpliceOut(t, w, out, "FLOOR"); got != "FLOOR" {
		t.Fatalf("the floor frame passes through untouched: %q", got)
	}
}

// TestZenbuFrameKeepAliveCycle — THE keep-alive flip cycle, byte-pinned
// at the wrapper: /open paints (a=T under the stable office id) → ctrl+b
// to the floor (the child FREEZES — PID stable + alive + SIGSTOPped —
// and the wrapper's diff flushes exactly one a=d) → ctrl+b back (the
// SAME pid thaws and the RETAINED frame re-emits BYTE-IDENTICALLY —
// the fake child parked at `exec sleep` emits ZERO new bytes, so any
// a=T is definitionally the store's cached frame, proven BEFORE any new
// child output) → ctrl+c (the quit path's Close reaps the thawed child,
// its delete riding the direct seam). The full trace:
//
//	a=T(same id) … a=d(floor) … a=T(same id, instant) … a=d(quit)
func TestZenbuFrameKeepAliveCycle(t *testing.T) {
	pinBrowserLaneEnv(t)
	plantKittyFrameFake(t)
	panels.ZenbuRegistry().Clear()
	t.Cleanup(panels.ZenbuRegistry().Clear)
	w, out := frameTestWrapper(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/open " + laneFixtureURL(t)})
	m = waitLaneGrid(t, m, "zenbu-fake open file:///")
	pid := m.BrowserLanePid()
	if pid <= 0 {
		t.Fatalf("the premium child's pid reads through the harness seam, got %d", pid)
	}

	// OPEN: one flush publishes + splices the retained frame (a=T).
	_ = m.Frame()
	spliceOpen := frameSpliceOut(t, w, out, "FLUSH")
	assertFrameSplice(t, "keepalive open", spliceOpen, m.floorW, m.middleH-3)
	apcOpen := strings.TrimPrefix(spliceOpen, "FLUSH")
	if !strings.Contains(apcOpen, "\x1b_Ga=T,") {
		t.Fatalf("the open splice carries the a=T: %q", apcOpen)
	}

	// ctrl+b → FLOOR: the child FREEZES (PID stable, alive, SIGSTOPped);
	// the wrapper's emitted-set diff flushes exactly one a=d.
	m = runMsg(t, m, ctrlB())
	if m.BrowserPremiumActive() || !m.BrowserLaneSuspended() {
		t.Fatalf("the floor flip freezes the lane: active=%v suspended=%v", m.BrowserPremiumActive(), m.BrowserLaneSuspended())
	}
	if got := m.BrowserLanePid(); got != pid {
		t.Fatalf("the freeze never respawns: pid %d → %d", pid, got)
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("the frozen child stays ALIVE: kill(%d, 0) = %v", pid, err)
	}
	_ = m.Frame()
	if got := frameSpliceOut(t, w, out, "FLOOR"); got != "FLOOR"+frameTestDeleteFrame() {
		t.Fatalf("the floor flip flushes exactly one a=d:\n got %q\nwant %q", got, "FLOOR"+frameTestDeleteFrame())
	}

	// ctrl+b → BROWSER: the SAME pid thaws; the RETAINED frame re-emits
	// BYTE-IDENTICALLY (the parked fake emits zero new bytes — the a=T
	// is definitionally the store's cached frame, BEFORE any child output).
	m = runMsg(t, m, ctrlB())
	if !m.BrowserPremiumActive() || m.BrowserLaneSuspended() {
		t.Fatalf("the return thaws the lane: active=%v suspended=%v", m.BrowserPremiumActive(), m.BrowserLaneSuspended())
	}
	if got := m.BrowserLanePid(); got != pid {
		t.Fatalf("the flip's PID never changes: %d → %d", pid, got)
	}
	_ = m.Frame()
	spliceResume := frameSpliceOut(t, w, out, "BACK")
	if got, want := strings.TrimPrefix(spliceResume, "BACK"), apcOpen; got != want {
		t.Fatalf("the resume re-emits the RETAINED frame byte-identically:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(spliceResume, "\x1b_Ga=d,") {
		t.Fatalf("no a=d may interleave the thaw (same-id atomic replace): %q", spliceResume)
	}

	// ctrl+c — the quit path: Close reaps the thawed child; its delete
	// rides the direct seam.
	out.Reset()
	_ = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if got := out.String(); got != frameTestDeleteFrame() {
		t.Fatalf("the quit's lane Close flushed the delete through the direct seam:\n got %q\nwant %q", got, frameTestDeleteFrame())
	}
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("the quit path reaps the child: kill(%d, 0) = %v", pid, err)
	}
}
