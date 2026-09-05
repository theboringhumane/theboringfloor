// model_image_test.go — the inbound boss-turn image preview's APP seam:
// EvChatBoss completion pins (and the EvChatMedia SSE lane) buffer the
// payload ONCE (probe-once), the lazy rasterize lands rows into the chat
// panel through a real tea.Cmd round-trip, and the ui.images posture
// gates: auto/ascii paint the half-block raster, off renders chips only.
package app

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// imageFixture — the raster lane's gold fixture (the panels package's own
// testdata — THE ONE shared pixel contract, never re-encoded here).
func imageFixture(t *testing.T) (dataURL, hash string) {
	t.Helper()
	raw, err := os.ReadFile("../../internal/panels/testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("raster fixture: %v", err)
	}
	dataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	return dataURL, state.DataURLHash(dataURL)
}

// bossMediaPin — the completed boss turn carrying the image: Meta holds
// the small carrier, Event.Media the payload (opencode.go's pin shape).
func bossMediaPin(dataURL, hash, msgID string) state.Event {
	it := state.MediaItem{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8, Hash: hash, URL: dataURL}
	return state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: msgID, From: "boss", Kind: "boss", Text: "the diagram is a checker.", At: 1,
		Pending: false, Meta: state.MediaMeta([]state.MediaItem{it}),
	}, Media: []state.MediaItem{it}}
}

func imageModel(t *testing.T, cfg *config.Config) Model {
	t.Helper()
	backend := &recBackend{}
	m := New(backend, cfg)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 130, Height: 32})
	return m
}

// TestBossImageRendersChipAndRaster — the happy path (auto posture): the
// completion pin lands, the probe runs through the real cmd tree, and the
// frame shows the 🖼 chip + the half-block raster rows EXACTLY ONCE.
// (pinNeutralImageEnv: auto routes the DETECT chain — on a kitty/ghostty
// host the probe would paint the kitty strip instead; the stub folds the
// lane to the universal ASCII paint this test asserts.)
func TestBossImageRendersChipAndRaster(t *testing.T) {
	pinNeutralImageEnv(t)
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))

	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "🖼 paste-diagram.png · 8×8 · image/png") {
		t.Fatalf("the chip must render with dims+mime:\n%s", frame)
	}
	if n := strings.Count(frame, "▀"); n != 32 {
		t.Fatalf("an 8-col × 4-row checker paints 32 half-blocks, got %d:\n%s", n, frame)
	}
	// the RAW frame carries the checker's truecolor palette.
	raw := m.Frame()
	if !strings.Contains(raw, "38;2;255;0;0") || !strings.Contains(raw, "48;2;0;0;255") {
		t.Fatal("the raster rows carry the red-over-blue truecolor SGR")
	}
}

// TestBossImageOffChipsOnly — ui.images=off: the chip renders from the
// carrier, the probe never fires (no raster rows).
func TestBossImageOffChipsOnly(t *testing.T) {
	dataURL, hash := imageFixture(t)
	cfg := config.Default()
	cfg.UI.Images = "off"
	m := imageModel(t, cfg)
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))

	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "🖼 paste-diagram.png · 8×8 · image/png") {
		t.Fatalf("off still renders the chip:\n%s", frame)
	}
	if strings.Contains(frame, "▀") {
		t.Fatalf("off paints NO raster rows:\n%s", frame)
	}
	if len(m.imgProbed) != 0 {
		t.Fatalf("off never probes: %v", m.imgProbed)
	}
}

// TestBossImageASCIILane — ui.images=ascii renders the raster the same as
// auto (v1's lane), and the probe latch pins ONE rasterize per payload.
func TestBossImageASCIILane(t *testing.T) {
	dataURL, hash := imageFixture(t)
	cfg := config.Default()
	cfg.UI.Images = "ascii"
	m := imageModel(t, cfg)
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1")) // a repeated pin NEVER re-rasterizes
	if len(m.imgProbed) != 1 {
		t.Fatalf("probe-once: one payload probes once, got %v", m.imgProbed)
	}
	frame := ansi.Strip(m.Frame())
	if n := strings.Count(frame, "▀"); n != 32 {
		t.Fatalf("ascii lane paints the same 32 half-blocks, got %d", n)
	}
}

// TestBossImageSSEMediaLane — the EvChatMedia SSE sighting buffers the
// payload even when the completion pin's own Event.Media is empty (an
// older serve's fetch missed it) — the raster still lands, keyed by the
// pin's Meta carrier. (pinNeutralImageEnv: same detect-chain stub as the
// happy-path raster test.)
func TestBossImageSSEMediaLane(t *testing.T) {
	pinNeutralImageEnv(t)
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatMedia,
		Msg:    state.ChatMsg{ID: "bossmsg-m1", From: "boss", Kind: "boss"},
		CallID: "part-1", SessionID: "ses-1",
		Media: []state.MediaItem{{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8, Hash: hash, URL: dataURL}},
	})
	it := state.MediaItem{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8, Hash: hash}
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss", Text: "pinned, no payload.", At: 1,
		Meta: state.MediaMeta([]state.MediaItem{it}),
	}})

	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "🖼 paste-diagram.png") {
		t.Fatalf("the SSE lane must buffer for the pin's carrier:\n%s", frame)
	}
	if n := strings.Count(frame, "▀"); n != 32 {
		t.Fatalf("the SSE lane raster lands 32 half-blocks, got %d", n)
	}
}

// TestBossImageFailedProbe — an undecodable payload lands the failed
// latch: the chip swaps to the dim unsupported copy, no raster rows.
func TestBossImageFailedProbe(t *testing.T) {
	bad := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("this is not a png"))
	hash := state.DataURLHash(bad)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(bad, hash, "bossmsg-m9"))

	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "🖼 paste-diagram.png · unsupported image · click txt link") {
		t.Fatalf("a failed probe swaps the chip to unsupported:\n%s", frame)
	}
	if strings.Contains(frame, "▀") {
		t.Fatalf("a failed probe paints no raster:\n%s", frame)
	}
}

// TestImagesSlashCycler — the /images command: bare reports the posture +
// detected lane, a mode argument flips cfg AND persists to brain.json
// (scratch home — the real one is never touched), a bogus mode errors.
func TestImagesSlashCycler(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	cfg := config.Default()
	m := imageModel(t, cfg)
	m = runMsg(t, m, slashMsg{text: "/images"})
	last := m.st.Chat[len(m.st.Chat)-1]
	if !strings.Contains(last.Text, "images auto") || !strings.Contains(last.Text, "detected lane:") {
		t.Fatalf("bare /images reports posture + lane: %q", last.Text)
	}
	m = runMsg(t, m, slashMsg{text: "/images ascii"})
	last = m.st.Chat[len(m.st.Chat)-1]
	if !strings.Contains(last.Text, "images → ascii") {
		t.Fatalf("the cycler confirms: %q", last.Text)
	}
	if m.cfg.UI.Images != "ascii" {
		t.Fatalf("cfg flips: %q", m.cfg.UI.Images)
	}
	if rc := readBrain(t); rc.UI.Images != "ascii" {
		t.Fatalf("brain.json persists: %+v", rc.UI)
	}
	m = runMsg(t, m, slashMsg{text: "/images bogus"})
	last = m.st.Chat[len(m.st.Chat)-1]
	if !strings.Contains(last.Text, "unknown mode") || m.cfg.UI.Images != "ascii" {
		t.Fatalf("a bogus mode errors + keeps the old posture: %q / %q", last.Text, m.cfg.UI.Images)
	}
}

// TestBossImageRemoteChipOnly — a remote URL degrades silently: chip row
// only, NEVER a fetch (the payload channel is stripped at the wire gate —
// the app passes it through untouched).
func TestBossImageRemoteChipOnly(t *testing.T) {
	m := imageModel(t, config.Default())
	it := state.MediaItem{Mime: "image/png", Filename: "remote.png"} // no dims, no hash, no URL
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m3", From: "boss", Kind: "boss", Text: "linked you the shot.", At: 1,
		Meta: state.MediaMeta([]state.MediaItem{it}),
	}, Media: []state.MediaItem{it}})

	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "🖼 remote.png · unsupported image · click txt link") {
		t.Fatalf("remote URLs read as the dim unsupported chip:\n%s", frame)
	}
	if len(m.imgProbed) != 0 {
		t.Fatalf("a remote URL never probes (no fetch possible): %v", m.imgProbed)
	}
}
