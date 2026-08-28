// model_image_splice_test.go — the wave-86 chat-preview frame SPLICE at
// the app seam: the kitty lane's preview never rides the View string
// (the renderer would drop the zero-width APC — the wave-81 forensics);
// the chat-media region of the frame registry carries {office id,
// ABSOLUTE cell, cached APC} per rendered Frame (origin math pinned
// through a REAL Model frame at two widths against the frame's own chip
// row), a scroll moves the splice's CUP byte-for-byte, scrolled-off
// media earns the diff's a=d, and non-kitty lanes publish NOTHING.
package app

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// spliceCleanup — the process-singleton registry's per-test hygiene
// (both regions cleared before AND after).
func spliceCleanup(t *testing.T) {
	t.Helper()
	panels.ZenbuRegistry().Clear()
	panels.ZenbuRegistry().PublishChatMedia(nil)
	t.Cleanup(func() { panels.ZenbuRegistry().Clear(); panels.ZenbuRegistry().PublishChatMedia(nil) })
}

// chipRowInFrame — the ABSOLUTE 0-based screen row of the media chip in
// the rendered frame (the independent geometry read: the image's paint
// cell must sit exactly ONE row below its chip).
func chipRowInFrame(t *testing.T, frame string) int {
	t.Helper()
	for i, ln := range strings.Split(frame, "\n") {
		if strings.Contains(ansi.Strip(ln), "🖼 paste-diagram.png") {
			return i
		}
	}
	t.Fatalf("the chip renders:\n%s", frame)
	return -1
}

// TestBossImageSpliceOriginDesktopMobile — the chat-media publish's
// ABSOLUTE origin math through REAL Model frames at a desktop width
// (130: the sidebar at x=floorW) and a mobile width (60: the panel under
// the floor band): the preview's absolute cell sits ONE row below the
// frame's own chip row, at the transcript's bubble-body column (the
// sidebar/band origin + the box chrome + the chat gutter + the boss
// hanging indent).
func TestBossImageSpliceOriginDesktopMobile(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	for _, width := range []int{130, 60} {
		spliceCleanup(t)
		dataURL, hash := imageFixture(t)
		backend := &recBackend{}
		m := New(backend, config.Default())
		m = runMsg(t, m, tea.WindowSizeMsg{Width: width, Height: 32})
		m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))

		frame := m.Frame()
		if strings.Contains(frame, "\x1b_G") {
			t.Fatalf("width %d: ZERO APC bytes ride the View frame", width)
		}
		chat := panels.ZenbuRegistry().ChatSnapshotForTest()
		if len(chat) != 1 {
			t.Fatalf("width %d: the chat region holds exactly the one preview, got %d", width, len(chat))
		}
		chipRow := chipRowInFrame(t, frame)
		if chat[0].OY != chipRow+1 {
			t.Fatalf("width %d: the preview paints ONE row below the chip: OY=%d, chip row=%d", width, chat[0].OY, chipRow)
		}
		// the column: the content origin (desktop: floorW+1, mobile: 1)
		// + the chat gutter (2) + the boss hanging indent (7).
		wantX := 1 + 2 + 7
		if !m.mobile() {
			wantX += m.floorW
		} else if width != 60 || !m.mobile() {
			t.Fatalf("width %d must be the mobile layout", width)
		}
		if chat[0].OX != wantX {
			t.Fatalf("width %d: the preview's column = content origin + the bubble indent: OX=%d, want %d (floorW=%d)", width, chat[0].OX, wantX, m.floorW)
		}
		if width == 130 && m.mobile() {
			t.Fatal("130 cols is the desktop layout")
		}
	}
}

// bossMediaPinAt — bossMediaPin's twin with an explicit timestamp (the
// timeline interleaves by At — a pinned image must OWN the tail slot to
// sit at the transcript's bottom).
func bossMediaPinAt(dataURL, hash, msgID string, at int64) state.Event {
	it := state.MediaItem{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8, Hash: hash, URL: dataURL}
	return state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: msgID, From: "boss", Kind: "boss", Text: "the diagram is a checker.", At: at,
		Pending: false, Meta: state.MediaMeta([]state.MediaItem{it}),
	}, Media: []state.MediaItem{it}}
}

// bossPad pins ONE plain boss message (the transcript-height filler).
func bossPad(id string, at int64) state.Event {
	return state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: id, From: "boss", Kind: "boss",
		Text: "padding the transcript so the viewport scrolls — a taller tail.",
		At:   at, Pending: false,
	}}
}

// TestBossImageSpliceScrollCUP — the scroll contract end to end: new
// tail content with follow pinned SCROLLS the preview up (the published
// cell moves row-for-row, the wrapper's splice CUP differs
// byte-for-byte), an "up" keypress moves it back down by one, and
// flooding the tail past the window empties the publish (the wrapper's
// emitted-set diff flushes its a=d).
func TestBossImageSpliceScrollCUP(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	spliceCleanup(t)
	dataURL, hash := imageFixture(t)
	backend := &recBackend{}
	m := New(backend, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 130, Height: 32})
	// pad ABOVE the image so the transcript EXCEEDS the viewport (follow
	// pins the bottom — the image bubble owns the tail slot, At 1000).
	for i := 0; i < 12; i++ {
		m = runMsg(t, m, bossPad("pad"+strconv.Itoa(i), int64(1+i)))
	}
	m = runMsg(t, m, bossMediaPinAt(dataURL, hash, "bossmsg-m1", 1000))

	var buf strings.Builder
	w := panels.NewZenbuFrameWriter(&buf, panels.ZenbuRegistry())

	_ = m.Frame()
	chat0 := panels.ZenbuRegistry().ChatSnapshotForTest()
	if len(chat0) != 1 {
		t.Fatalf("the preview publishes at the bottom pin: %d", len(chat0))
	}
	buf.Reset()
	_, _ = w.Write([]byte("F1"))
	splice0 := buf.String()

	// grow the tail: follow pins the bottom, so the preview scrolls UP
	// by the new rows (the viewport was already past-full).
	n0 := m.chat.TranscriptRows()
	m = runMsg(t, m, bossPad("tail-a", 1001))
	m = runMsg(t, m, bossPad("tail-b", 1002))
	m = runMsg(t, m, bossPad("tail-c", 1003))
	_ = m.Frame()
	delta := m.chat.TranscriptRows() - n0
	if delta < 1 {
		t.Fatalf("the tail must grow: %d → %d", n0, m.chat.TranscriptRows())
	}
	chat1 := panels.ZenbuRegistry().ChatSnapshotForTest()
	if len(chat1) != 1 {
		t.Fatalf("the preview still publishes while visible: %d", len(chat1))
	}
	if chat1[0].OY != chat0[0].OY-delta || chat1[0].OX != chat0[0].OX || chat1[0].Frame != chat0[0].Frame {
		t.Fatalf("the tail's growth scrolls the preview up %d rows: (%d,%d) → (%d,%d)",
			delta, chat0[0].OX, chat0[0].OY, chat1[0].OX, chat1[0].OY)
	}
	buf.Reset()
	_, _ = w.Write([]byte("F2"))
	splice1 := buf.String()
	cup0 := "\x1b[" + strconv.Itoa(chat0[0].OY+1) + ";" + strconv.Itoa(chat0[0].OX+1) + "H"
	cup1 := "\x1b[" + strconv.Itoa(chat1[0].OY+1) + ";" + strconv.Itoa(chat1[0].OX+1) + "H"
	if !strings.Contains(splice0, "\x1b7"+cup0) || !strings.Contains(splice1, "\x1b7"+cup1) {
		t.Fatalf("the splices carry DECSC + the frames' own CUPs:\n F1 %q\n F2 %q", splice0[:80], splice1[:80])
	}
	if splice0 == splice1 || !strings.Contains(splice1, chat0[0].Frame) {
		t.Fatal("the scroll re-emits the SAME APC at the moved CUP (byte-for-byte move)")
	}

	// an "up" keypress (the chat tab owns keys) scrolls the viewport up
	// one row: the SAME image re-publishes one row LOWER on screen.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	_ = m.Frame()
	chat2 := panels.ZenbuRegistry().ChatSnapshotForTest()
	if len(chat2) != 1 || chat2[0].OY != chat1[0].OY+1 {
		t.Fatalf("an up-scroll moves the cell back down one row: %+v (was %d)", chat2, chat1[0].OY)
	}

	// flood the tail past the window, then page to the BOTTOM (the "up"
	// above unlatched follow — the frozen scrollback KEEPS the preview
	// visible until the member returns to the tail): the publish empties
	// and the wrapper's diff flushes the a=d.
	for i := 0; i < 30; i++ {
		m = runMsg(t, m, bossPad("flood"+strconv.Itoa(i), int64(1100+i)))
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	_ = m.Frame()
	if chat := panels.ZenbuRegistry().ChatSnapshotForTest(); len(chat) != 0 {
		t.Fatalf("the scrolled-off preview leaves the publish: %d entries", len(chat))
	}
	buf.Reset()
	_, _ = w.Write([]byte("F3"))
	id8 := panels.KittyIDHash8(chat0[0].OfficeID)
	if !strings.Contains(buf.String(), "\x1b_Ga=d,d=I,i="+id8+",q=2;\x1b\\") {
		t.Fatalf("the scrolled-off preview earns the diff's a=d (i=%s):\n%q", id8, buf.String())
	}
	if strings.Contains(buf.String(), chat0[0].Frame) {
		t.Fatal("the scrolled-off preview never re-places")
	}
}

// TestBossImageSpliceWrapperBytes — the wrapper-level byte pin: after a
// REAL Model frame, ONE renderer flush through the wrapper lands the
// chat-media splice — DECSC + CUP(the published absolute cell, 1-based)
// + the cached f=100 APC (head…tail intact, NO c=/r= — the wave-81
// emission ruling) + DECRC — and a second identical flush splices
// NOTHING (the bandwidth skip).
func TestBossImageSpliceWrapperBytes(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	spliceCleanup(t)
	dataURL, hash := imageFixture(t)
	backend := &recBackend{}
	m := New(backend, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 130, Height: 32})
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))
	_ = m.Frame()

	chat := panels.ZenbuRegistry().ChatSnapshotForTest()
	if len(chat) != 1 {
		t.Fatalf("the chat region holds exactly the one preview, got %d", len(chat))
	}
	if strings.Contains(chat[0].Frame, "c=") || strings.Contains(chat[0].Frame, ",r=") {
		t.Fatal("the chat preview's APC carries NO c=/r= (the wave-81 emission ruling)")
	}
	var buf strings.Builder
	w := panels.NewZenbuFrameWriter(&buf, panels.ZenbuRegistry())
	_, _ = w.Write([]byte("FLUSH"))
	want := "FLUSH" + "\x1b7\x1b[" + strconv.Itoa(chat[0].OY+1) + ";" + strconv.Itoa(chat[0].OX+1) + "H" + chat[0].Frame + "\x1b8"
	if buf.String() != want {
		t.Fatalf("the splice lands after the flush at the pinned absolute cell:\n got %q\nwant %q", buf.String()[:120], want[:120])
	}
	buf.Reset()
	_, _ = w.Write([]byte("FLUSH2"))
	if buf.String() != "FLUSH2" {
		t.Fatalf("the identical second flush splices NOTHING (the bandwidth skip): %q", buf.String()[:120])
	}
	// an ED flush (the resize shape) re-emits the preview.
	buf.Reset()
	_, _ = w.Write([]byte("\x1b[2J"))
	if !strings.Contains(buf.String(), chat[0].Frame) {
		t.Fatal("the ED flush re-emits the preview (terminal-state repair)")
	}
}

// TestBossImageSpliceNonKittyPublishesNothing — the ASCII lane (neutral
// env) and posture off publish ZERO chat-media entries: the universal
// half-block paint lives in the View string as REAL glyphs (the
// renderer-safe v1 paint — exactly today's degradation), and chips-only
// needs no splice either.
func TestBossImageSpliceNonKittyPublishesNothing(t *testing.T) {
	pinNeutralImageEnv(t)
	spliceCleanup(t)
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))
	frame := m.Frame()
	if chat := panels.ZenbuRegistry().ChatSnapshotForTest(); len(chat) != 0 {
		t.Fatalf("the ASCII lane publishes nothing: %d entries", len(chat))
	}
	if n := strings.Count(ansi.Strip(frame), "▀"); n != 32 {
		t.Fatalf("the ASCII lane's 32 half-blocks still paint (today's degradation): %d", n)
	}

	// posture off on a kitty host: chips only, no publish.
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	spliceCleanup(t)
	cfg := config.Default()
	cfg.UI.Images = "off"
	m2 := imageModel(t, cfg)
	m2 = runMsg(t, m2, bossMediaPin(dataURL, hash, "bossmsg-m2"))
	_ = m2.Frame()
	if chat := panels.ZenbuRegistry().ChatSnapshotForTest(); len(chat) != 0 {
		t.Fatalf("posture off publishes nothing: %d entries", len(chat))
	}
}

// TestBossImageSpliceHiddenChatPublishesNothing — the chat region clears
// when the transcript is not painted: a NON-chat sidebar tab (the
// registry empties — the wrapper's diff would a=d anything live).
func TestBossImageSpliceHiddenChatPublishesNothing(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	spliceCleanup(t)
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))
	_ = m.Frame()
	if chat := panels.ZenbuRegistry().ChatSnapshotForTest(); len(chat) != 1 {
		t.Fatalf("the chat tab publishes the preview: %d", len(chat))
	}
	if !m.SelectTab("agents") { // NOT "terminal": its first visit spawns a shell (the failure fallback would flip back)
		t.Fatal("the agents tab exists")
	}
	_ = m.Frame()
	if chat := panels.ZenbuRegistry().ChatSnapshotForTest(); len(chat) != 0 {
		t.Fatalf("a non-chat sidebar tab clears the chat region: %d", len(chat))
	}
}
