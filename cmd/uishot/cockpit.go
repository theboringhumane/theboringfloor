package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// cockpitShot uses the real app renderer with an explicitly simulated
// mission. It starts neither a backend nor the terminal PTY.
func cockpitShot(theme string) string {
	chrome.SetTheme(theme)
	office.SetTheme(theme)
	d := newFocusDriver()
	d.send(tea.WindowSizeMsg{Width: 168, Height: 46})
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] backend: codex"})
	for _, e := range []state.Employee{
		{ID: "dev", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking},
		{ID: "scout", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteWorking},
		{ID: "review", Name: "kritikos-1", Role: state.RoleReviewer, Sprite: state.SpriteAtMailbox},
		{ID: "run", Name: "runner-1", Role: state.RoleRunner, Sprite: state.SpriteAtDesk},
	} {
		d.send(state.Event{Kind: state.EvHire, Employee: e})
	}
	for i, title := range []string{"Audit project architecture", "Map command surface", "Define instrument palette", "Wire agent telemetry", "Verify terminal geometry", "Review release notes"} {
		status := state.TaskDone
		if i == 3 || i == 4 {
			status = state.TaskInProgress
		}
		if i == 5 {
			status = state.TaskPending
		}
		d.send(state.Event{Kind: state.EvTask, Task: state.BoardTask{ID: fmt.Sprintf("c-%d", i), Title: title, Status: status}})
	}
	for i, msg := range []state.ChatMsg{
		{From: "user", Kind: "user", Text: "Bring the office online. I want a clear view of every agent, every mission, and anything that needs my attention."},
		{From: "boss", Kind: "boss", Text: "Command deck is ready. Six agents on the floor; two tasks are in flight.\n\n### Current mission\nTurn the workspace into a cockpit that makes the team's work visible at a glance.\n\n- **Tactical floor** tracks the crew's positions.\n- **Operations** shows real task counts and agent states.\n- **Command console** keeps your conversation and tools in reach."},
		{From: "boss", Kind: "boss", Text: "The architecture audit is complete. Tekton is wiring telemetry while Skopos checks the layout at smaller terminal sizes."},
		{From: "user", Kind: "user", Text: "Keep the instruments sharp. Make sure the command surface stays readable."},
		{From: "boss", Kind: "boss", Text: "Understood. The instruments follow your selected palette. Active work, completion, and waiting tasks keep distinct labels. The instruments collapse when space is tight.\n\nUse **Tab** to switch tools, **Ctrl+W** to expand this console, or **/** to open the command palette."},
	} {
		msg.ID = fmt.Sprintf("cockpit-%d", i)
		kind := state.EvChatBoss
		if msg.From == "user" {
			kind = state.EvChatUser
		}
		d.send(state.Event{Kind: kind, Msg: msg})
	}
	d.send(state.Event{Kind: state.EvStatus, Text: "Demo mission · awaiting command"})
	d.m.Frame() // compute the floor plan before seating the simulated crew
	d.pump(18)
	return d.m.Frame()
}
