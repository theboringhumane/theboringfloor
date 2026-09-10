package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestCockpitLayoutSurvivesEveryTheme(t *testing.T) {
	m := cockpitModel(t)
	m.chat.StageDraft("Keep this command while switching palettes")
	m.st.Tasks = []state.BoardTask{{ID: "mission", Title: "Verify every palette", Status: state.TaskInProgress}}
	for _, size := range [][2]int{{40, 18}, {70, 24}, {100, 30}, {140, 40}, {168, 46}, {220, 60}} {
		m.resize(size[0], size[1])
		for _, name := range chrome.ThemeNames() {
			chrome.SetTheme(name)
			office.SetTheme(name)
			frame := m.Frame()
			plain := ansi.Strip(frame)
			rows := strings.Split(frame, "\n")
			if len(rows) != size[1] {
				t.Fatalf("%s %v drew %d rows", name, size, len(rows))
			}
			for _, row := range rows {
				if ansi.StringWidth(row) != size[0] {
					t.Fatalf("%s %v: overflowing row %q", name, size, ansi.Strip(row))
				}
			}
			for _, label := range []string{"TACTICAL FLOOR", "03 / CHAT", "COMMAND INPUT", "COMMAND CONSOLE", "Keep this command"} {
				if !strings.Contains(plain, label) {
					t.Fatalf("%s %v lost %q", name, size, label)
				}
			}
			if size[0] >= 140 && size[1] >= 40 {
				for _, label := range []string{"02 / OPERATIONS", "AGENT NETWORK", "Verify every palette"} {
					if !strings.Contains(plain, label) {
						t.Fatalf("%s lost %q", name, label)
					}
				}
			}
			r, g, b, _ := chrome.PanelBgColor.RGBA()
			if !strings.Contains(frame, fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8)) {
				t.Fatalf("%s did not apply its own background", name)
			}
			if m.tabs.ActiveIndex() != 0 {
				t.Fatalf("%s changed the selected tool", name)
			}
		}
	}
}

func cockpitModel(t *testing.T) Model {
	t.Helper()
	scratchHome(t)
	before := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(before); office.SetTheme(before) })
	chrome.SetTheme("cockpit")
	office.SetTheme("cockpit")
	m := New(&recBackend{}, nil)
	m.bootDone = true
	return m
}

func TestCockpitFrameAndStateChanges(t *testing.T) {
	m := cockpitModel(t)
	for _, size := range [][2]int{{40, 18}, {70, 24}, {100, 30}, {140, 40}, {168, 46}, {220, 60}} {
		m.resize(size[0], size[1])
		for _, expanded := range []bool{false, true} {
			m.focusPanel = expanded
			for _, tab := range []string{"chat", "agents", "board", "files"} {
				m.SelectTab(tab)
				rows := strings.Split(m.Frame(), "\n")
				if len(rows) != size[1] {
					t.Fatalf("%v %s expanded=%t drew %d rows", size, tab, expanded, len(rows))
				}
				for _, row := range rows {
					if ansi.StringWidth(row) != size[0] {
						t.Fatalf("%v %s expanded=%t overflow: %q", size, tab, expanded, ansi.Strip(row))
					}
				}
			}
		}
	}
	m.focusPanel = false
	m.resize(168, 46)
	m.SelectTab("chat")
	m = runMsg(t, m, state.Event{Kind: state.EvTask, Task: state.BoardTask{ID: "mission", Title: "Verify live instruments", Status: state.TaskInProgress}})
	before := m.Frame()
	if !strings.Contains(ansi.Strip(before), "Verify live instruments") {
		t.Fatal("dispatch did not reach cockpit")
	}
	m = runMsg(t, m, state.Event{Kind: state.EvTask, Task: state.BoardTask{ID: "mission", Title: "Verify live instruments", Status: state.TaskDone}})
	after := m.Frame()
	if before == after || !strings.Contains(ansi.Strip(after), "100%") {
		t.Fatal("task completion did not refresh the cached meter")
	}
}

func TestCockpitFloorClickTracksRenderedPosition(t *testing.T) {
	m := cockpitModel(t)
	m.resize(168, 46)
	m.Frame()
	next, _ := m.Update(state.Event{Kind: state.EvTick})
	m = next.(Model) // leave the application's recurring tick command unexecuted
	m.Frame()
	position, ok := office.SpritePosition("manager")
	if !ok {
		t.Fatal("manager not seated")
	}
	m.handleClick(tea.MouseClickMsg(tea.Mouse{X: m.navigatorWidth() + position.X, Y: position.Y + 2, Button: tea.MouseLeft}))
	if m.tabs.ActiveIndex() != 5 {
		t.Fatal("click on visible manager did not open activity")
	}
	m.SelectTab("chat")
	y := m.middleH - chrome.CockpitTelemetryHeight(m.floorW, m.middleH-1) + 2
	m.handleClick(tea.MouseClickMsg(tea.Mouse{X: m.navigatorWidth() + position.X, Y: y, Button: tea.MouseLeft}))
	if m.tabs.ActiveIndex() != 0 {
		t.Fatal("telemetry click selected an invisible floor sprite")
	}
}
