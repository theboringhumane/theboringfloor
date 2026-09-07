package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// applyControlMutations performs remote-control writes on the Bubble Tea
// goroutine, keeping the model's ordinary input, stop, and session seams as
// the sole owners of their respective state transitions.
func (m *Model) applyControlMutations(ev state.Event) tea.Cmd {
	switch ev.Kind {
	case state.EvControlSend:
		text := strings.TrimSpace(ev.ControlText)
		if text == "" {
			return nil
		}
		return currentBackendSend(m.currentBackend, m.plan, text, nil)
	case state.EvControlStop:
		cmd := m.stopWork()
		m.notice("remote: stopped current work")
		return cmd
	case state.EvControlNew:
		m.newOffice()
		m.notice("remote: started a new session")
		return nil
	default:
		return nil
	}
}
