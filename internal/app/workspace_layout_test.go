package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestWorkspaceNavigatorKeepsToolTabAndDraft(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, nil)
	m.bootDone = true
	m.resize(150, 42)
	m.chat.StageDraft("keep my draft")
	m.SelectTab("board")
	m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'e', Mod: tea.ModCtrl}))
	if !m.floorNavFocused || m.tabs.ActiveIndex() != 3 {
		t.Fatal("navigator replaced the active tool tab")
	}
	m.handleKey(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))
	m.routePaste(tea.PasteMsg{Content: "Authentication update"})
	if !m.floors.Editing() || !strings.Contains(ansi.Strip(m.Frame()), "Authentication update") {
		t.Fatal("conversation form did not receive paste")
	}
	m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if m.floorNavFocused || m.tabs.ActiveIndex() != 3 {
		t.Fatal("cancel did not restore tool focus")
	}
	m.SelectTab("chat")
	if !strings.Contains(ansi.Strip(m.Frame()), "keep my draft") {
		t.Fatal("floor navigation lost the composer draft")
	}
}

func TestWorkspaceThreePaneGeometryAndNarrowDrawer(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, nil)
	m.bootDone = true
	for _, width := range []int{40, 70, 100, 120, 150, 200} {
		m.resize(width, 30)
		m.floorNavFocused = false
		for _, expanded := range []bool{false, true} {
			m.focusPanel = expanded
			frame := m.Frame()
			rows := strings.Split(frame, "\n")
			if len(rows) != 30 {
				t.Fatalf("%d expanded=%t: %d rows", width, expanded, len(rows))
			}
			for _, row := range rows {
				if ansi.StringWidth(row) > width {
					t.Fatalf("%d expanded=%t: overflowing row %q", width, expanded, ansi.Strip(row))
				}
			}
			if width >= 100 && !strings.Contains(ansi.Strip(frame), "FLOORS") {
				t.Fatal("desktop lost persistent navigator")
			}
			if width >= 100 && m.panelX() != width-(func() int {
				if expanded {
					return width - m.navigatorWidth()
				}
				return m.sidebar
			})() {
				t.Fatal("tool origin differs from rendered width")
			}
		}
		m.focusPanel = false
		m.floorNavFocused = true
		m.frameNonce++
		if !strings.Contains(ansi.Strip(m.Frame()), "FLOORS") {
			t.Fatalf("%d: navigator drawer missing", width)
		}
	}
}
