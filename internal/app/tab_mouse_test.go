package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRightPanelTabLabelsActivateTabs(t *testing.T) {
	prev := SpawnTerminal
	SpawnTerminal = func(cols, rows int) (TerminalTab, error) { return &mouseFakeTerm{}, nil }
	t.Cleanup(func() { SpawnTerminal = prev })

	m := New(&recBackend{}, nil)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = nm.(Model)
	if m.mobile() {
		t.Fatal("precondition: 120x40 must render the desktop right panel")
	}
	clicks := make(map[int]int)
	for x := 0; x < 120-m.floorW; x++ {
		if idx, ok := m.tabs.TabAt(x, 0); ok {
			if _, seen := clicks[idx]; !seen {
				clicks[idx] = x
			}
		}
	}
	if len(clicks) != 7 {
		t.Fatalf("visible tab click targets = %d, want 7", len(clicks))
	}
	for idx := 0; idx < 7; idx++ {
		localX, ok := clicks[idx]
		if !ok {
			t.Errorf("tab %d has no visible click target", idx)
			continue
		}
		x := m.floorW + localX
		if got, hit := m.tabs.TabAt(localX, 0); !hit || got != idx {
			t.Fatalf("precondition: local x=%d hits (%d, %t), want (%d, true)", localX, got, hit, idx)
		}
		nm, _ = m.Update(tea.MouseClickMsg(tea.Mouse{X: x, Y: 1, Button: tea.MouseLeft}))
		m = nm.(Model)
		if got := m.tabs.ActiveIndex(); got != idx {
			t.Errorf("click on tab %d at screen x=%d activated %d", idx, x, got)
		}
	}
}
