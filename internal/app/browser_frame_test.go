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
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/headless"
	"github.com/theboringhumane/theboringfloor/internal/panels"
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
// production shape (the runtime's main): one wrapper for the
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

// ---------------------------------------------------------------------------
// the headless SHOT's frame-splice contract (the zenbu lane's twin seam)
// ---------------------------------------------------------------------------

// shotFrameEngine — the fake headless engine for the app-level shot
// suite: a call recorder behind the panels seam swap (NO live chrome).
type shotFrameEngine struct {
	calls        int
	lastW, lastH int
}

// pinShotFrameEngine — avail=true + the checker PNG render, swapped and
// restored (panels.SetHeadlessForShot's precedent).
func pinShotFrameEngine(t *testing.T, png []byte) *shotFrameEngine {
	t.Helper()
	e := &shotFrameEngine{}
	restore := panels.SetHeadlessForShot(
		func() (string, bool) { return "/fake/chrome", true },
		func(_ context.Context, rawurl string, w, h int) (*headless.Result, error) {
			e.calls++
			e.lastW, e.lastH = w, h
			return &headless.Result{URL: rawurl, Title: "Fixture Gazette", PNG: png}, nil
		})
	t.Cleanup(restore)
	return e
}

// shotFrameAPC — the EXPECTED emitted APC for the checker PNG: a=T +
// t=d + q=2 + C=1 + i=<content hash8> + f=100, the payload verbatim, NO
// c=/r= keys (the wave-81/82 emission ruling).
func shotFrameAPC(png []byte) string {
	return "\x1b_Ga=T,t=d,q=2,C=1,i=" + panels.KittyIDHash8(panels.KittyImageID(png)) +
		",f=100;" + base64.StdEncoding.EncodeToString(png) + "\x1b\\"
}

// shotFrameDelete — the office-side a=d for the checker PNG's content id.
func shotFrameDelete(png []byte) string {
	return "\x1b_Ga=d,d=I,i=" + panels.KittyIDHash8(panels.KittyImageID(png)) + ",q=2;\x1b\\"
}

// TestShotFramePublish — the headless SHOT through the LIVE app glue
// (DESKTOP 140x30): /open lands the fetch AND the render (the fake
// engine), the pane's shot region paints, the registry publishes the PNG
// at the ABSOLUTE origin (0,3) + pane-local (0,0) — the wrapper's flush
// is byte-pinned (cursor-save + CUP(4;1) + the f=100 APC with NO c=/r=
// keys + cursor-restore) — and the keep-alive flip cycle replays the
// zenbu lane's exact semantics: q hides the slot (the wrapper's diff
// flushes ONE a=d), ctrl+b back re-publishes the CACHED bytes
// byte-identically with ZERO new engine calls, ctrl+c clears the
// registry through the pane's Close.
func TestShotFramePublish(t *testing.T) {
	pinBrowserLaneEnv(t)
	t.Setenv("PATH", t.TempDir()) // no terminal-browser: the zenbu lane misses by construction
	png, err := os.ReadFile("../panels/testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("read the checker fixture: %v", err)
	}
	e := pinShotFrameEngine(t, png)
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	panels.ZenbuRegistry().Clear()
	t.Cleanup(panels.ZenbuRegistry().Clear)
	w, out := frameTestWrapper(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	if m.mobile() {
		t.Fatal("140 cols is the desktop layout")
	}
	raw := laneFixtureURL(t)
	m = runMsg(t, m, slashMsg{text: "/open " + raw})

	// the shot landed: SHOT MODE live, the render fired ONCE at the pane
	// box's exact pixel dims (floorW cols × 9, (middleH-1-2) body rows × 18).
	if !m.BrowserShotActive() {
		t.Fatal("the /open render enters shot mode on the kitty lane")
	}
	wantW, wantH := m.floorW*9, (m.middleH-3)*18
	if e.calls != 1 || e.lastW != wantW || e.lastH != wantH {
		t.Fatalf("the render fired %d× (%dx%d), want 1× (%dx%d)", e.calls, e.lastW, e.lastH, wantW, wantH)
	}
	if bw, bh := m.BrowserShotBoxPx(); bw != wantW || bh != wantH {
		t.Fatalf("the pane's box seam = %dx%d, want %dx%d", bw, bh, wantW, wantH)
	}
	// the /open notice settled on the FETCH verdict (the shot msg never
	// touches the latch).
	if !lastOfficeNoticeHas(m, "browser: Fixture Gazette · "+raw) {
		t.Fatalf("the /open notice settles on the fetch verdict")
	}
	// the pane's frame: badge + strip (LEFT slot, the right strip unmoved).
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{" shot ", "▸ headless chromium · file:///", "screenshot: /"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the shot-mode frame carries %q:\n%s", want, frame)
		}
	}
	if got := m.ActiveTabIndex(); got != 0 {
		t.Fatalf("the right strip never moves, got %d", got)
	}

	// the wrapper byte-pin: FLUSH + cursor-save + CUP(4;1) (the absolute
	// origin (0,3) + pane-local (0,0), 1-based) + the f=100 APC + restore.
	wantSplice := "FLUSH" + fmt.Sprintf("\x1b7\x1b[4;%dH", m.navigatorWidth()+1) + shotFrameAPC(png) + "\x1b8"
	if got := frameSpliceOut(t, w, out, "FLUSH"); got != wantSplice {
		t.Fatalf("the wrapper's emitted bytes for the shot:\n got %q\nwant %q", got, wantSplice)
	}
	if strings.Contains(wantSplice, ",c=") || strings.Contains(wantSplice, ",r=") {
		t.Fatalf("the wave-81 ruling: NO c=/r= keys in the emitted frame")
	}

	// q → the floor: the registry clears through the publish (the
	// wrapper's emitted-set diff flushes exactly ONE a=d; NO direct
	// emit — the shot's keep-alive rides the publish seam only).
	out.Reset()
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'q', Text: "q"}))
	if m.BrowserShotActive() {
		t.Fatal("q hides the shot with the slot flip")
	}
	_ = m.Frame() // publishes the empty state
	if got := frameSpliceOut(t, w, out, "F2"); got != "F2"+shotFrameDelete(png) {
		t.Fatalf("the floor flip flushes exactly one a=d:\n got %q\nwant %q", got, "F2"+shotFrameDelete(png))
	}
	if got := frameSpliceOut(t, w, out, "F3"); got != "F3" {
		t.Fatalf("…and the next flush passes through clean: %q", got)
	}

	// ctrl+b → BACK: the CACHED bytes re-publish byte-identically, ZERO
	// new engine calls (no re-render on the flip cycle).
	m = runMsg(t, m, ctrlB())
	if !m.BrowserShotActive() {
		t.Fatal("the return re-shows the cached shot")
	}
	_ = m.Frame()
	if got := frameSpliceOut(t, w, out, "BACK"); got != "BACK"+fmt.Sprintf("\x1b7\x1b[4;%dH", m.navigatorWidth()+1)+shotFrameAPC(png)+"\x1b8" {
		t.Fatalf("the return re-publishes the cached bytes byte-identically:\n got %q", got)
	}
	if e.calls != 1 {
		t.Fatalf("the flip cycle NEVER re-renders: engine calls %d, want 1", e.calls)
	}

	// ctrl+c — the quit path: the pane's Close clears the registry (the
	// renderer's final flush finds nothing).
	_ = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	active, _, _, imgs, _ := panels.ZenbuRegistry().SnapshotForTest()
	if active || len(imgs) != 0 {
		t.Fatalf("the quit path's Close clears the registry: active=%v imgs=%d", active, len(imgs))
	}
}
