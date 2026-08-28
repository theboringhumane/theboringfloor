// kitty_lanes_test.go — the NATIVE image lanes' pins: the kitty
// placeholder strip and the OSC 1337 frame are byte-exact over the
// shared checker gold, the lane-preference chain (posture × detection)
// is pinned corner by corner over hermetic injected env, corrupt bytes
// fall back to the ASCII paint without panicking, and the chat panel
// routes frame vs rows through renderMediaRows.
package panels

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// checkerB64 — both native lanes' payload over the gold fixture: the
// kitty strip re-encodes via the stdlib png encoder (the fixture IS that
// encode — TestCheckerFixtureIsSelfVerifying), OSC 1337 inlines the
// source bytes verbatim. Same b64 either way.
func checkerB64(t *testing.T) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(loadCheckerFixture(t))
}

// checkerHash8 — the strip's i=<hash8>: sha1(source bytes)[:8] hex of
// the FULL fixture (never a truncated prefix).
func checkerHash8(t *testing.T) string {
	t.Helper()
	sum := sha1.Sum(loadCheckerFixture(t))
	return hex.EncodeToString(sum[:4])
}

// TestKittyPlaceholderStripExactPin — the kitty frame over the 8×8
// checker: EXACTLY `\x1b_G a=T,t=d,f=100,i=<hash8>,q=2;` + the base64
// payload head…tail + `ESC\` (the full frame is pinned — the fixture's
// self-verifying encode makes the payload byte-known).
func TestKittyPlaceholderStripExactPin(t *testing.T) {
	raw := loadCheckerFixture(t)
	img, w, h, err := decodeMediaImage(raw)
	if err != nil {
		t.Fatalf("fixture decodes: %v", err)
	}
	if w != 8 || h != 8 {
		t.Fatalf("the checker is 8×8, got %d×%d", w, h)
	}
	id := KittyImageID(raw)
	if got := KittyIDHash8(id); got != checkerHash8(t) {
		t.Fatalf("KittyImageID must render sha1[:8] hex: %q vs %q", got, checkerHash8(t))
	}
	strip := PlaceholderStrip(img, 8, 4, id)
	want := "\x1b_Ga=T,t=d,f=100,i=" + checkerHash8(t) + ",q=2;" + checkerB64(t) + "\x1b\\"
	if strip != want {
		t.Fatalf("the kitty placeholder strip is byte-pinned:\n got %q\nwant %q", strip, want)
	}
	if !strings.HasPrefix(strip, "\x1b_Ga=T,t=d,f=100,i="+checkerHash8(t)+",q=2;") {
		t.Fatalf("the pinned frame START (a=T,t=d,f=100,i=<sha1>,q=2;)")
	}
	if !strings.HasSuffix(strip, "\x1b\\") {
		t.Fatalf("the payload's trailing bytes end in ESC\\ (octet sequence end)")
	}
	// the strip is pure escape: zero display cells, strips to nothing
	// (a transcript row carrying it can never mis-measure).
	if w := ansi.StringWidth(strip); w != 0 {
		t.Fatalf("an escape frame has 0 display cells, got %d", w)
	}
	if s := ansi.Strip(strip); s != "" {
		t.Fatalf("an escape frame strips to nothing, got %q", s)
	}
	// the id is deterministic per payload and differs across payloads.
	if KittyImageID(raw) != KittyImageID(raw) {
		t.Fatal("KittyImageID must be deterministic")
	}
	if KittyImageID(append(append([]byte{}, raw...), 0x00)) == KittyImageID(raw) {
		t.Fatal("different source bytes must hash to a different id")
	}
}

// TestOSCarInlineFrameExactPin — the OSC 1337 frame over the checker:
// `ESC]1337;File=inline=1;width=8:height=4;base64,<b64> BEL` — the
// resize attributes detectable in the marker, the source payload
// verbatim, BEL-terminated.
func TestOSCarInlineFrameExactPin(t *testing.T) {
	raw := loadCheckerFixture(t)
	frame := ITermInlineFrame(raw, 8, 4)
	want := "\x1b]1337;File=inline=1;width=8:height=4;base64," + checkerB64(t) + "\x07"
	if frame != want {
		t.Fatalf("the OSC 1337 frame is byte-pinned:\n got %q\nwant %q", frame, want)
	}
	if !strings.HasPrefix(frame, "\x1b]1337;File=inline=1;width=8:height=4;base64,") {
		t.Fatal("the marker structure: File=inline=1;width=<cols>:height=<rows>;base64,")
	}
	if !strings.HasSuffix(frame, "\x07") {
		t.Fatal("the frame is ^G (BEL) terminated")
	}
	if w := ansi.StringWidth(frame); w != 0 {
		t.Fatalf("an escape frame has 0 display cells, got %d", w)
	}
}

// TestLaneRendersReserveASCIIBox — the native lanes reserve the SAME
// cell box the ASCII paint occupies: checker 8×8 → 8 cols × 4 cell rows
// on ALL THREE lanes; the 128×16 gradient at the 64-col cap → 64×4.
func TestLaneRendersReserveASCIIBox(t *testing.T) {
	raw := loadCheckerFixture(t)
	klr, err := KittyLaneRenderer{}.Render(raw, 64, 20)
	if err != nil || klr.Cols != 8 || klr.CellRows != 4 || klr.Frame == "" || klr.Rows != nil {
		t.Fatalf("kitty render: %+v err=%v", klr, err)
	}
	ilr, err := ITermLaneRenderer{}.Render(raw, 64, 20)
	if err != nil || ilr.Cols != 8 || ilr.CellRows != 4 || ilr.Frame == "" || ilr.Rows != nil {
		t.Fatalf("iterm render: %+v err=%v", ilr, err)
	}
	alr, err := RenderImageForLane(ASCIILane, raw, 64, 20)
	if err != nil || alr.Cols != 8 || alr.CellRows != 4 || len(alr.Rows) != 4 || alr.Frame != "" {
		t.Fatalf("ascii render: cols=%d cellRows=%d rows=%d err=%v", alr.Cols, alr.CellRows, len(alr.Rows), err)
	}
	// and RenderImageForLane's ASCII path is RasterFromBytes-identical.
	rows, err := RasterFromBytes(raw, 64, 20)
	if err != nil || len(rows) != len(alr.Rows) {
		t.Fatalf("RasterFromBytes baseline: %v", err)
	}
	for i := range rows {
		if rows[i] != alr.Rows[i] {
			t.Fatalf("ascii row %d must match RasterFromBytes exactly", i)
		}
	}
	// the gradient: 128×16 JPEG at the cap → 64 cols × 4 cell rows
	// everywhere (a non-PNG format: kitty re-encodes png, iterm inlines
	// the source jpeg bytes).
	big := gradientJPEG(t)
	k2, err2 := KittyLaneRenderer{}.Render(big, 64, 20)
	a2, err3 := RenderImageForLane(ASCIILane, big, 64, 20)
	if err2 != nil || err3 != nil || k2.Cols != 64 || k2.CellRows != 4 || a2.Cols != 64 || a2.CellRows != 4 {
		t.Fatalf("gradient box: kitty=%+v/%v ascii=%+v/%v", k2, err2, a2, err3)
	}
	// jpeg on the kitty lane: the strip payload re-encodes PNG (f=100)
	// while the i= id still hashes the SOURCE bytes.
	if !strings.HasPrefix(k2.Frame, "\x1b_Ga=T,t=d,f=100,i=") {
		t.Fatalf("jpeg rides the same pinned strip shape: %q", k2.Frame[:40])
	}
}

// gradientJPEG — a 128×16 horizontal grayscale gradient JPEG (the mirror
// of TestRasterGradientJPEG's fixture).
func gradientJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 128, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 128; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 2), G: uint8(x * 2), B: uint8(x * 2), A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestResolveImageLaneChain — the lane-preference logic: posture × the
// env-crashed detection matrix, strict kitty → iterm → ascii ordering,
// the explicit ascii pin beating a kitty env, and every unmapped/tmux/
// dumb/sixel combo folding ASCII (zero v1 regression).
func TestResolveImageLaneChain(t *testing.T) {
	cases := []struct {
		name string
		env  func(string) string
		mode string
		want ImageLane
	}{
		{"ghostty auto rides kitty", envOf("TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), "auto", KittyLane},
		{"kitty TERM_PROGRAM auto", envOf("TERM_PROGRAM", "kitty", "TERM", "xterm-kitty"), "auto", KittyLane},
		{"KITTY_WINDOW_ID auto", envOf("KITTY_WINDOW_ID", "3", "TERM", "xterm-kitty"), "auto", KittyLane},
		{"iTerm.app auto rides OSC1337", envOf("TERM_PROGRAM", "iTerm.app", "TERM", "xterm-256color", "ITERM_SESSION_ID", "w0t0p0:stub"), "auto", ITermLane},
		{"wezterm socket auto", envOf("WEZTERM_UNIX_SOCKET", "/tmp/wez", "TERM", "xterm-256color"), "auto", ITermLane},
		{"vscode PID auto", envOf("VSCODE_PID", "4242", "TERM_PROGRAM", "vscode", "TERM", "xterm-256color"), "auto", ITermLane},
		{"tmux folds ascii even inside ghostty", envOf("TMUX", "/tmp/tmux-1000/default,1,0", "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), "auto", ASCIILane},
		{"dumb TERM folds ascii", envOf("TERM", "dumb"), "auto", ASCIILane},
		{"unset TERM folds ascii", envOf(), "auto", ASCIILane},
		{"sixel has no renderer yet — ascii", envOf("TERM", "xterm-sixel"), "auto", ASCIILane},
		{"unmapped Apple_Terminal is ascii", envOf("TERM_PROGRAM", "Apple_Terminal", "TERM", "xterm-256color"), "auto", ASCIILane},
		{"txt-mode empty TERM is ascii", envOf("TERM_PROGRAM", "", "TERM", ""), "auto", ASCIILane},
		// the explicit pin beats ANY detected native lane:
		{"ascii pin beats ghostty", envOf("TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), "ascii", ASCIILane},
		{"ascii pin beats iTerm", envOf("TERM_PROGRAM", "iTerm.app", "TERM", "xterm-256color"), "ascii", ASCIILane},
		// unknown/off postures fold conservative (off is gated upstream):
		{"off folds ascii", envOf("TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), "off", ASCIILane},
		{"bogus posture folds ascii", envOf("TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), "bogus", ASCIILane},
	}
	for _, tc := range cases {
		if got := ResolveImageLane(tc.mode, DetectImageSupportFrom(tc.env)); got != tc.want {
			t.Errorf("%s: want %s, got %s", tc.name, tc.want, got)
		}
	}
	// the chain is strictly ordered: kitty first, then iterm, then ascii.
	if !(KittyLane > ASCIILane && ITermLane > ASCIILane) {
		t.Fatal("lane constants: the native lanes outrank ascii in declaration order")
	}
	if ResolveImageLane("auto", KittyLane) != KittyLane ||
		ResolveImageLane("auto", ITermLane) != ITermLane ||
		ResolveImageLane("auto", SixelLane) != ASCIILane ||
		ResolveImageLane("auto", NoneLane) != ASCIILane ||
		ResolveImageLane("auto", ASCIILane) != ASCIILane {
		t.Fatal("the auto chain: kitty → iterm → ascii, everything else ascii")
	}
}

// TestNativeLaneCorruptFallsBack — rows/eject errors on the native lanes:
// garbage bytes NEVER panic, the dispatcher attempts the ASCII paint
// (which also fails — same decode gate), and the caller's error lands
// for the failed-chip latch. A valid payload on a native lane paints the
// frame (never rows).
func TestNativeLaneCorruptFallsBack(t *testing.T) {
	corrupt := []byte("this is not a png — the fixture duplicated as garbage")
	for _, lane := range []ImageLane{KittyLane, ITermLane, ASCIILane} {
		lr, err := RenderImageForLane(lane, corrupt, 0, 0)
		if err == nil {
			t.Fatalf("%s: corrupt bytes must error (the failed latch), got %+v", lane, lr)
		}
		if lr.Frame != "" || lr.Rows != nil {
			t.Fatalf("%s: a failed render carries no paint", lane)
		}
	}
	// empty payload too — every lane, no panic, clean error.
	for _, lane := range []ImageLane{KittyLane, ITermLane} {
		if _, err := RenderImageForLane(lane, nil, 0, 0); err == nil {
			t.Fatalf("%s: empty payload must error", lane)
		}
	}
	// and the header-only giant-dims claim is refused on the native lanes
	// exactly as on the ASCII one (the shared decode gate).
	if _, err := (KittyLaneRenderer{}).Render(pngHeaderWithDims(20000, 20000), 8, 8); err == nil ||
		!strings.Contains(err.Error(), "20000x20000") {
		t.Fatalf("kitty honors the dimension budget: %v", err)
	}
	if _, err := (ITermLaneRenderer{}).Render(make([]byte, state.MediaMaxPayloadBytes+1), 8, 8); err == nil {
		t.Fatal("iterm honors the 8 MiB payload cap")
	}
	// valid payload on the native lanes: frame paint, zero rows.
	for lane, prefix := range map[ImageLane]string{
		KittyLane: "\x1b_Ga=T,t=d,f=100,i=", ITermLane: "\x1b]1337;File=inline=1;",
	} {
		lr, err := RenderImageForLane(lane, loadCheckerFixture(t), 0, 0)
		if err != nil || !strings.HasPrefix(lr.Frame, prefix) || lr.Rows != nil {
			t.Fatalf("%s: valid payload paints the frame, got %+v err=%v", lane, lr, err)
		}
	}
}

// TestChatBubbleLaneNativeFrameSlot — the renderMediaRows routing hook
// (the wave-86 splice routing): a KITTY frame never rides the View
// string — the media rows are PURE reservation rows (zero APC bytes,
// the renderer would drop them anyway) and the paint slot rides the
// block for the registry publish — while the OSC 1337 (iterm) frame
// keeps the OLD embedded-row behavior (ONE verbatim row + cellRows-1
// reservations; no id, no delete escape — the splice could never target
// it). The pending twin never previews.
func TestChatBubbleLaneNativeFrameSlot(t *testing.T) {
	raw := loadCheckerFixture(t)
	lr, err := KittyLaneRenderer{}.Render(raw, 64, 20)
	if err != nil {
		t.Fatalf("kitty render: %v", err)
	}
	itermFrame := ITermInlineFrame(raw, 8, 4)
	hash := state.DataURLHash("data:image/png;base64,AAAA")
	meta := state.MediaMeta([]state.MediaItem{
		{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8, Hash: hash},
	})
	mkChat := func() *Chat {
		c := NewChat(nil)
		c.SetSize(80, 30)
		c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "what does the diagram say?", At: 10},
			{ID: "bossmsg-k", From: "boss", Kind: "boss", Text: "kitty paints it.", At: 11, Meta: meta},
			{ID: "bossmsg-i", From: "boss", Kind: "boss", Text: "iterm paints it.", At: 12, Meta: meta},
			{ID: "bossmsg-p", From: "boss", Kind: "boss", Text: "streaming…", At: 13, Pending: true, Meta: meta},
		}})
		c.SetImageFrame("bossmsg-k", hash, lr.Frame, lr.CellRows)
		c.SetImageFrame("bossmsg-i", hash, itermFrame, 4)
		return c
	}
	convo := mkChat().renderConversation()
	plain := ansi.Strip(convo)
	// the kitty strip NEVER rides the transcript (the splice paints it);
	// the OSC 1337 frame still rides VERBATIM (the embedded contract).
	if strings.Contains(convo, lr.Frame) {
		t.Fatal("the kitty placeholder strip left the View string (the splice owns it)")
	}
	if strings.Contains(convo, "\x1b_G") {
		t.Fatal("ZERO kitty APC bytes ride the transcript")
	}
	if !strings.Contains(convo, itermFrame) {
		t.Fatal("the OSC 1337 frame rides verbatim (the embedded lane)")
	}
	// chip → body ordering on the kitty bubble (the stripped plain: the
	// reservation rows are blank, chip/body land in order).
	chipIdx := strings.Index(plain, "🖼 paste-diagram.png · 8×8 · image/png")
	bodyIdx := strings.Index(plain, "kitty paints it.")
	if chipIdx < 0 || bodyIdx < 0 || chipIdx >= bodyIdx {
		t.Fatalf("chip above body:\n%s", plain)
	}
	// the iterm raw ordering: chip < frame < body (the chip's own text
	// anchors — the iterm bubble is the SECOND chip).
	rawChip2 := strings.LastIndex(convo, "🖼 paste-diagram.png")
	rawIterm := strings.Index(convo, itermFrame)
	rawBody2 := strings.Index(convo, "iterm paints")
	if !(rawChip2 >= 0 && rawIterm > rawChip2 && rawBody2 > rawIterm) {
		t.Fatalf("iterm order must be chip → frame → body (%d %d %d)", rawChip2, rawIterm, rawBody2)
	}
	// a frame paints zero half-blocks and the chips render exactly once
	// per completed bubble (the pending twin previews nothing).
	if strings.Contains(plain, "▀") {
		t.Fatalf("the native lanes paint no half-blocks:\n%s", plain)
	}
	if n := strings.Count(plain, "🖼 paste-diagram.png"); n != 2 {
		t.Fatalf("chips: exactly the two completed bubbles, got %d", n)
	}
	// the kitty cell-box reservation: between the chip row's end and the
	// body there are cellRows (4) BLANK reservation rows — the bubble
	// spends the SAME vertical budget a 4-row ASCII paint would.
	rawChip := strings.Index(convo, "🖼 paste-diagram.png")
	rawBody := strings.Index(convo, "kitty paints")
	if rawChip < 0 || rawBody < 0 {
		t.Fatalf("the kitty chip + body render: %d %d", rawChip, rawBody)
	}
	seg := convo[rawChip:rawBody]
	if n := strings.Count(seg, "\n"); n != lr.CellRows+1 {
		t.Fatalf("the kitty preview reserves cellRows blank rows (chip break + %d reservation breaks), got %d", lr.CellRows, n)
	}
	if !strings.Contains(plain, "streaming…") {
		t.Fatal("the pending bubble's own text still renders")
	}
	// the paint slot rides the block: ONE kitty slot at the first
	// reservation row (block-local row 1 — the chip is row 0), the
	// strip's i= id, the cached verbatim APC; the iterm bubble registers
	// NOTHING (the embedded lane).
	c := mkChat()
	_ = c.renderConversation()
	var kSlots, iSlots int
	for _, blk := range c.blocks {
		switch blk.id {
		case "bossmsg-k":
			kSlots = len(blk.media)
			if kSlots == 1 {
				if blk.media[0].row != 1 || blk.media[0].id != KittyImageID(raw) || blk.media[0].frame != lr.Frame {
					t.Fatalf("the kitty slot: row=%d id=%08x frame-match=%v", blk.media[0].row, blk.media[0].id, blk.media[0].frame == lr.Frame)
				}
			}
		case "bossmsg-i":
			iSlots = len(blk.media)
		}
	}
	if kSlots != 1 || iSlots != 0 {
		t.Fatalf("slots: kitty bubble registers exactly one slot, iterm none — got %d/%d", kSlots, iSlots)
	}
}

// TestKittyStripSurvivesFold — a frame row passes foldStyledRows
// VERBATIM (0 display cells fit any budget): the frame can never burst
// mid-escape even if a caller folds it.
func TestKittyStripSurvivesFold(t *testing.T) {
	lr, err := KittyLaneRenderer{}.Render(loadCheckerFixture(t), 64, 20)
	if err != nil {
		t.Fatal(err)
	}
	rows := foldStyledRows(lr.Frame, 10, 10)
	if len(rows) != 1 || rows[0] != lr.Frame {
		t.Fatalf("a frame folds atomically (never bursts): got %d rows", len(rows))
	}
}
