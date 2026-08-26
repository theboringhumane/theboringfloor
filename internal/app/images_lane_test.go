// images_lane_test.go — the NATIVE image lanes at the app seam: under a
// hermetically-stubbed terminal env (the host this suite runs on IS a
// ghostty terminal — without stubs every "auto"-posture probe would
// route by the host's real markers), a kitty env paints the kitty
// placeholder strip into the boss bubble, the iterm family paints the
// OSC 1337 frame, the explicit ascii pin + tmux folds keep the v1
// half-block paint, posture off stays chips-only everywhere, and a
// corrupt payload on a native lane degrades to the failed chip instead
// of panicking.
package app

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// pinImageEnv — HERMETIC terminal-env stubbing for the image lanes:
// EVERY detect-layer input is owned by the test (host ghostty markers —
// or a headless CI's empty env — never leak in), then the pair list
// overrides. t.Setenv restores each key at test end.
func pinImageEnv(t *testing.T, pairs ...string) {
	t.Helper()
	for _, k := range []string{
		"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM", "TERM_PROGRAM_VERSION",
		"WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID", "TERM", "COLORTERM",
	} {
		t.Setenv(k, "")
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		t.Setenv(pairs[i], pairs[i+1])
	}
}

// pinNeutralImageEnv — the plain-xterm stub: "auto" resolves the
// universal ASCII lane (what the pre-native-lanes tests always painted).
func pinNeutralImageEnv(t *testing.T) {
	t.Helper()
	pinImageEnv(t, "TERM_PROGRAM", "Apple_Terminal", "TERM", "xterm-256color")
}

// kittyCheckerPin — the kitty strip's pinned fragments over the shared
// gold fixture: the exact frame START (`ESC_G a=T,t=d,f=100,i=<sha1>,
// q=2;` with sha1(source bytes)[:8] hex) and the payload head.
func kittyCheckerPin(t *testing.T) (frameStart, payloadHead, payloadTail string) {
	t.Helper()
	raw, err := os.ReadFile("../../internal/panels/testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("raster fixture: %v", err)
	}
	frameStart = "\x1b_Ga=T,t=d,f=100,i=" + panels.KittyIDHash8(panels.KittyImageID(raw)) + ",q=2;"
	b64 := base64.StdEncoding.EncodeToString(raw)
	return frameStart, b64[:40], b64[len(b64)-16:]
}

// TestBossImageKittyLaneFrame — ghostty env + auto posture: the
// completed boss turn's file part paints the kitty placeholder strip
// (exact pinned start + base64 head/tail + ESC\ terminator), ZERO
// half-blocks, the 🖼 chip intact, ONE probe for a repeated pin.
func TestBossImageKittyLaneFrame(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	if l := panels.DetectImageSupport(); l != panels.KittyLane {
		t.Fatalf("hermetic stub: ghostty env detects kitty, got %s", l)
	}
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1")) // probe-once

	frameStart, head, tail := kittyCheckerPin(t)
	raw := m.Frame()
	if !strings.Contains(raw, frameStart) {
		t.Fatalf("the kitty strip's exact pinned start must ride the frame:\n%q", raw[:400])
	}
	if !strings.Contains(raw, head) || !strings.Contains(raw, tail+"\x1b\\") {
		t.Fatal("the base64 payload (head…tail) + the ESC\\ terminator must ride the frame")
	}
	plain := ansi.Strip(raw)
	if !strings.Contains(plain, "🖼 paste-diagram.png · 8×8 · image/png") {
		t.Fatalf("the chip renders with dims+mime:\n%s", plain)
	}
	if strings.Contains(plain, "▀") {
		t.Fatalf("the kitty lane paints ZERO half-blocks:\n%s", plain)
	}
	if len(m.imgProbed) != 1 {
		t.Fatalf("probe-once on the kitty lane: %v", m.imgProbed)
	}
}

// TestBossImageITermLaneFrame — iTerm env + auto posture: the OSC 1337
// frame paints (`ESC]1337;File=inline=1;width=8:height=4;base64,<b64>
// BEL` — resize attributes + the payload verbatim), zero half-blocks.
func TestBossImageITermLaneFrame(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "iTerm.app", "TERM", "xterm-256color", "ITERM_SESSION_ID", "w0t0p0:stub")
	if l := panels.DetectImageSupport(); l != panels.ITermLane {
		t.Fatalf("hermetic stub: iTerm env detects iterm, got %s", l)
	}
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-i1"))

	rawBytes, err := os.ReadFile("../../internal/panels/testdata/checker-8x8.png")
	if err != nil {
		t.Fatalf("raster fixture: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(rawBytes)
	pin := "\x1b]1337;File=inline=1;width=8:height=4;base64,"
	raw := m.Frame()
	if !strings.Contains(raw, pin) {
		t.Fatalf("the OSC 1337 marker with the resize values must ride the frame")
	}
	if !strings.Contains(raw, b64+"\x07") {
		t.Fatal("the inline payload + the BEL terminator must ride the frame")
	}
	plain := ansi.Strip(raw)
	if !strings.Contains(plain, "🖼 paste-diagram.png · 8×8 · image/png") || strings.Contains(plain, "▀") {
		t.Fatalf("chip intact, zero half-blocks on the iterm lane:\n%s", plain)
	}
}

// TestBossImageWezTermRidesITerm — the WezTerm socket marker routes the
// same OSC 1337 lane (the detect matrix's iterm family).
func TestBossImageWezTermRidesITerm(t *testing.T) {
	pinImageEnv(t, "WEZTERM_UNIX_SOCKET", "/tmp/wez", "TERM", "xterm-256color")
	dataURL, hash := imageFixture(t)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-w1"))
	if !strings.Contains(m.Frame(), "\x1b]1337;File=inline=1;width=8:height=4;base64,") {
		t.Fatal("wezterm rides the OSC 1337 lane")
	}
}

// TestBossImageAsciiPinOnKittyHost — the explicit /images ascii pin
// beats a kitty-detecting env: the SAME 32 half-blocks v1 painted (zero
// native-lane regression).
func TestBossImageAsciiPinOnKittyHost(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	dataURL, hash := imageFixture(t)
	cfg := config.Default()
	cfg.UI.Images = "ascii"
	m := imageModel(t, cfg)
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))

	frame := ansi.Strip(m.Frame())
	if n := strings.Count(frame, "▀"); n != 32 {
		t.Fatalf("the ascii pin paints 32 half-blocks on a kitty host, got %d", n)
	}
	if strings.Contains(m.Frame(), "\x1b_Ga=T") {
		t.Fatal("the ascii pin never emits the kitty strip")
	}
}

// TestBossImageKittyCorruptDegrades — a corrupt payload on the kitty
// lane: no panic, no frame, no half-blocks — the chip swaps to the dim
// unsupported copy (the native error's ASCII fallback also fails, the
// failed latch lands, exactly like v1).
func TestBossImageKittyCorruptDegrades(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	bad := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("this is not a png"))
	hash := state.DataURLHash(bad)
	m := imageModel(t, config.Default())
	m = runMsg(t, m, bossMediaPin(bad, hash, "bossmsg-bad"))

	raw := m.Frame()
	plain := ansi.Strip(raw)
	if !strings.Contains(plain, "🖼 paste-diagram.png · unsupported image · click txt link") {
		t.Fatalf("corrupt swaps the chip to unsupported:\n%s", plain)
	}
	if strings.Contains(raw, "\x1b_Ga=T") || strings.Contains(plain, "▀") {
		t.Fatal("a corrupt payload paints no frame and no half-blocks")
	}
}

// TestBossImageOffOnKittyHost — ui.images=off parks the probe on ANY
// lane (chips only, no kitty strip, no half-blocks).
func TestBossImageOffOnKittyHost(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	dataURL, hash := imageFixture(t)
	cfg := config.Default()
	cfg.UI.Images = "off"
	m := imageModel(t, cfg)
	m = runMsg(t, m, bossMediaPin(dataURL, hash, "bossmsg-m1"))

	raw := m.Frame()
	if strings.Contains(raw, "\x1b_Ga=T") || strings.Contains(ansi.Strip(raw), "▀") {
		t.Fatal("off paints NO paint")
	}
	if !strings.Contains(ansi.Strip(raw), "🖼 paste-diagram.png · 8×8 · image/png") {
		t.Fatal("off still renders the chip")
	}
	if len(m.imgProbed) != 0 {
		t.Fatalf("off never probes: %v", m.imgProbed)
	}
}

// TestBossImageLaneDetectMemoPerBoot — the detect read is memoized per
// Model (probe-time env traffic: one read), and imageLane() composes
// posture × detection through ResolveImageLane's strict chain.
func TestBossImageLaneDetectMemoPerBoot(t *testing.T) {
	pinImageEnv(t, "TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty")
	m := imageModel(t, config.Default())
	if l := m.detectImageLane(); l != panels.KittyLane {
		t.Fatalf("ghostty detects kitty: %s", l)
	}
	if !m.imgLaneDetOK {
		t.Fatal("the read latches (per-boot memo)")
	}
	// a second read is the memo (env mutations in between can't move it;
	// the memo is per Model — a fresh Model re-reads honestly).
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if l := m.detectImageLane(); l != panels.KittyLane {
		t.Fatalf("the memo pins the boot read: %s", l)
	}
	m2 := imageModel(t, config.Default())
	if l := m2.detectImageLane(); l != panels.ITermLane {
		t.Fatalf("a fresh Model re-reads the env: %s", l)
	}
	// posture × detection composes through the strict resolve chain
	// (the full matrix is pinned in panels.TestResolveImageLaneChain).
	if got := panels.ResolveImageLane("auto", m.imgLaneDet); got != panels.KittyLane {
		t.Fatalf("auto × kitty: %s", got)
	}
}
