package app

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func assertSolidFrame(t *testing.T, frame string, w, h int) {
	t.Helper()
	bootFrameShape(t, frame, w, h)
	screen := uv.NewScreenBuffer(w, h)
	uv.NewStyledString(frame).Draw(&screen, screen.Bounds())
	for y, row := range screen.Buffer.Lines {
		for x, cell := range row {
			if cell.Width > 0 && (cell.Style.Bg == nil || cell.Style.Fg == nil) {
				t.Fatalf("cell %d,%d (%q) uses the terminal default color", x, y, cell.Content)
			}
		}
	}
}

func TestSolidBackgroundAcrossLayoutsAndResizes(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	previous := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(previous) })
	for _, theme := range []string{"noir", "paper"} {
		chrome.SetTheme(theme)
		for _, layout := range []string{"office", "zen", "expanded", "plan", "dialog", "boot"} {
			t.Run(theme+"/"+layout, func(t *testing.T) {
				m := New(&recBackend{}, nil)
				m.bootDone = layout != "boot"
				for i, size := range [][2]int{{150, 42}, {70, 30}, {120, 35}} {
					m = runMsg(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
					if i == 0 {
						switch layout {
						case "zen":
							m.zen = true
						case "expanded":
							m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl}))
						case "plan":
							m.applyPlanTools(state.Event{Kind: state.EvPlanPresent, PlanToolText: "# Feature\n\n1. Implement\n2. Verify"})
						case "dialog":
							m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'n', Mod: tea.ModCtrl}))
						}
					}
					assertSolidFrame(t, m.View().Content, size[0], size[1])
				}
			})
		}
	}
}

func TestSolidBackgroundThemeSwitchInvalidatesCache(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	previous := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(previous) })
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 150, Height: 42})
	chrome.SetTheme("noir")
	dark := m.Frame()
	if cached := m.Frame(); cached != dark {
		t.Fatal("unchanged frame should reuse the painted cache")
	}
	chrome.SetTheme("paper")
	light := m.Frame()
	if light == dark {
		t.Fatal("theme switch reused the old canvas")
	}
	assertSolidFrame(t, light, 150, 42)
	// A blank cell in the office must adopt the new canvas immediately.
	screen := uv.NewScreenBuffer(150, 42)
	uv.NewStyledString(light).Draw(&screen, screen.Bounds())
	for _, row := range screen.Buffer.Lines {
		for _, cell := range row {
			if cell.Content == " " && cell.Style.Bg != nil && sameSurfaceColor(cell.Style.Bg, chrome.PanelBgColor) {
				return
			}
		}
	}
	t.Fatal("light frame has no cells with the new canvas color")
}

func sameSurfaceColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
