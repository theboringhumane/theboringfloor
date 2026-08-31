// tool_output_test.go — the app-level EvTool.ToolOutput contract: a
// tool event carrying the captured result text feeds the chat panel's
// click-to-expand body through the REAL Update seam (applyEventCore's
// SetToolOutput, keyed by the reducer's merge id — toolEntryID), for
// boss inline rows and employee wtool rows alike; empty outputs never
// reach the panel (the row expands to the pinned empty state there).
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// TestToolOutputFeedsThePanel: running (no output) → done WITH output
// flows through Update into the panel — the expanded boss row swaps its
// empty state for the captured body IN PLACE (the entry id the reducer
// merged under, "tool-<callID>").
func TestToolOutputFeedsThePanel(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeName: "boss",
		CallID: "c1", ToolName: "bash", ToolSummary: "go build ./...", ToolState: "running"})
	m.chat.ToggleToolOutput("tool-c1")
	if view := ansi.Strip(m.chat.View()); !strings.Contains(view, "no output as such") {
		t.Fatalf("the running tool's expanded body must be the pinned empty state:\n%s", view)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeName: "boss",
		CallID: "c1", ToolName: "bash", ToolSummary: "go build ./...", ToolState: "done",
		ToolOutput: "build ok"})
	view := ansi.Strip(m.chat.View())
	if !strings.Contains(view, "build ok") {
		t.Fatalf("the done event's ToolOutput must paint the expanded body through the Update seam:\n%s", view)
	}
	if strings.Contains(view, "no output as such") {
		t.Fatalf("the landed output replaces the empty state in place:\n%s", view)
	}
	// an output-LESS follow-up event never wipes the capture
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeName: "boss",
		CallID: "c1", ToolName: "bash", ToolSummary: "go build ./...", ToolState: "done"})
	if view := ansi.Strip(m.chat.View()); !strings.Contains(view, "build ok") {
		t.Fatalf("an empty ToolOutput must never overwrite the captured body:\n%s", view)
	}
}

// TestToolOutputWorkerEntryID: an EMPLOYEE tool event feeds the
// "wtool-<agent>-<callID>" entry — the same id the reducer's merge
// builds (toolEntryID is the single source of both).
func TestToolOutputWorkerEntryID(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		CallID: "c9", ToolName: "read", ToolSummary: "x.go", ToolState: "done", ToolOutput: "package main"})
	m.chat.ToggleThread("tekton-1") // the wtool row renders inside the expanded thread
	m.chat.ToggleToolOutput("wtool-tekton-1-c9")
	view := ansi.Strip(m.chat.View())
	if !strings.Contains(view, "package main") {
		t.Fatalf("the employee tool's output must feed its wtool entry's body:\n%s", view)
	}
	// toolEntryID mirrors the reducer exactly (the merge id IS the feed key)
	st := reducer(state.OfficeState{}, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		CallID: "c9", ToolName: "read", ToolSummary: "x.go", ToolState: "done"})
	if len(st.Chat) != 1 || st.Chat[0].ID != toolEntryID("tekton-1", "c9") {
		t.Fatalf("toolEntryID must mirror the reducer's merge id, got %+v", st.Chat)
	}
	if got := toolEntryID("", "c9"); got != "tool-c9" {
		t.Fatalf("boss/unnamed events merge under tool-<callID>, got %q", got)
	}
}
