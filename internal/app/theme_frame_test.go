package app

import (
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// TestThemeFrameAutoInvalidatesCache exercises the terminal-driven path: it
// must invalidate Frame through the chrome theme identity, not a one-off
// frameNonce bump. This test intentionally runs before explicit-theme tests;
// SetThemeAuto is correctly suppressed after an explicit theme is pinned.
func TestThemeFrameAutoInvalidatesCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	noir := m.Frame()
	if hits, misses := m.FrameCacheStats(); hits != 0 || misses != 1 {
		t.Fatalf("first Frame cache stats = hits=%d misses=%d, want 0/1", hits, misses)
	}
	if again := m.Frame(); again != noir {
		t.Fatal("unchanged Frame must reuse the cached bytes")
	}
	if hits, misses := m.FrameCacheStats(); hits != 1 || misses != 1 {
		t.Fatalf("unchanged Frame cache stats = hits=%d misses=%d, want 1/1", hits, misses)
	}

	m = runMsg(t, m, tea.BackgroundColorMsg{Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}})
	paper := m.Frame()
	if chrome.CurrentTheme().Name != chrome.DefaultLightTheme {
		t.Fatalf("light BackgroundColorMsg theme = %q, want %q", chrome.CurrentTheme().Name, chrome.DefaultLightTheme)
	}
	if paper == noir {
		t.Fatal("light BackgroundColorMsg reused noir frame bytes")
	}
	if _, misses := m.FrameCacheStats(); misses != 2 {
		t.Fatalf("theme identity change must miss exactly once, misses=%d want 2", misses)
	}
	if paperAgain := m.Frame(); paperAgain != paper {
		t.Fatal("unchanged paper Frame must reuse the cached bytes")
	}
	if hits, misses := m.FrameCacheStats(); hits != 2 || misses != 2 {
		t.Fatalf("paper repeat cache stats = hits=%d misses=%d, want 2/2", hits, misses)
	}

	// The explicit slash path carries the ordinary slash nonce for panel
	// state, but the new digest identity is what makes this package-global
	// palette repaint. One subsequent Frame is one cache miss, then stable.
	m = runMsg(t, m, slashMsg{text: "/theme noir"})
	noirAgain := m.Frame()
	if chrome.CurrentTheme().Name != "noir" || noirAgain == paper {
		t.Fatal("/theme noir must select a distinct noir frame")
	}
	if _, misses := m.FrameCacheStats(); misses != 3 {
		t.Fatalf("explicit theme change must miss exactly once, misses=%d want 3", misses)
	}
	if cached := m.Frame(); cached != noirAgain {
		t.Fatal("unchanged explicit-theme Frame must reuse the cached bytes")
	}
}

func TestThemeFramePaperAndNoirFillDesktopAndMobilePanels(t *testing.T) {
	for _, tc := range []struct {
		name        string
		width       int
		panelBGCode string
	}{
		{name: "paper desktop", width: 140, panelBGCode: "48;2;240;241;244"},
		{name: "paper mobile", width: 70, panelBGCode: "48;2;240;241;244"},
		{name: "noir desktop", width: 140, panelBGCode: "48;2;22;22;25"},
		{name: "noir mobile", width: 70, panelBGCode: "48;2;22;22;25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			theme := strings.Fields(tc.name)[0]
			if !chrome.SetTheme(theme) {
				t.Fatalf("SetTheme(%q) failed", theme)
			}
			m := New(&recBackend{}, nil)
			m = runMsg(t, m, tea.WindowSizeMsg{Width: tc.width, Height: 32})
			m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
				ID: "theme-user", From: "user", Kind: "user", Text: "show the themed panel"}})
			m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeName: "boss", CallID: "theme-tool",
				ToolName: "read", ToolSummary: "theme.go", ToolState: "done", ToolOutput: "panel paint verified"})
			frame := m.Frame()
			panelH := m.middleH
			if m.mobile() {
				panelH -= m.floorBandH()
			}
			assertFrameWidthAndPanelFill(t, frame, tc.width, panelH, tc.panelBGCode)
			if !strings.Contains(frame, "show the themed panel") {
				t.Fatal("populated chat text missing from themed panel frame")
			}
			if !strings.Contains(frame, "[tool]") {
				t.Fatal("populated tool row missing from themed panel frame")
			}

			// A modal overlays the same populated panel; it must not disturb the
			// full-width mobile or sidebar-width desktop background wrapper.
			m = runMsg(t, m, state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "theme-perm",
				ToolName: "write", ToolSummary: "theme.go"})
			modal := m.Frame()
			assertFrameWidthAndPanelFill(t, modal, tc.width, panelH, tc.panelBGCode)
			if !strings.Contains(modal, "PERMISSION") {
				t.Fatal("permission modal missing from themed frame")
			}
		})
	}
}

func assertFrameWidthAndPanelFill(t *testing.T, frame string, width, panelH int, panelBGCode string) {
	t.Helper()
	lines := strings.Split(frame, "\n")
	if len(lines) < 4 {
		t.Fatalf("short themed frame: %d rows", len(lines))
	}
	paintedRows := 0
	for i, line := range lines {
		if got := lipgloss.Width(line); got != width {
			t.Fatalf("line %d width = %d, want %d: %q", i, got, width, line)
		}
		if strings.Contains(line, panelBGCode) {
			paintedRows++
		}
	}
	if paintedRows < panelH {
		t.Fatalf("PanelBg %q paints only %d/%d panel rows; panel wrapper is not continuous", panelBGCode, paintedRows, panelH)
	}
}
