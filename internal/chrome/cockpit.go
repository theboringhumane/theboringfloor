package chrome

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// InstrumentRule labels an existing row, so adding chrome never displaces
// a panel's content or its mouse hit regions.
func InstrumentRule(label string, width int) string {
	if width <= 0 {
		return ""
	}
	text := "─ " + label + " "
	return AccentText.Render(ansi.Truncate(text, width, "")) +
		Fg(CurrentTheme().Border, strings.Repeat("─", max(0, width-ansi.StringWidth(text))))
}

// InstrumentFrame annotates the existing top and bottom border rows.
func InstrumentFrame(frame, title, footer string, width int) string {
	if width < 8 {
		return frame
	}
	rows := strings.Split(frame, "\n")
	if len(rows) < 2 {
		return frame
	}
	border := func(left, text, right string) string {
		return Fg(CurrentTheme().Border, left) + InstrumentRule(text, width-2) + Fg(CurrentTheme().Border, right)
	}
	rows[0] = border("┌", title, "┐")
	rows[len(rows)-1] = border("└", footer, "┘")
	return strings.Join(rows, "\n")
}

// CockpitTelemetryHeight reserves instruments only when a useful floor
// and command surface fit. Small windows retain the compact office band.
func CockpitTelemetryHeight(width, height int) int {
	if width < 28 || height < 28 {
		return 0
	}
	return min(24, height-12)
}

// CockpitTelemetry is a projection of OfficeState. Meters show task counts,
// never invented CPU, network latency, activity histories, or progress.
func CockpitTelemetry(st state.OfficeState, width, height int) string {
	if width < 4 || height < 2 {
		return ""
	}
	w := width - 4
	var rows []string
	add := func(s string) { rows = append(rows, " "+ansi.Truncate(s, w, "…")) }
	section := func(s string) { add(InstrumentRule(s, w)) }
	var pending, running, done, stalled int
	for _, task := range st.Tasks {
		switch task.Status {
		case state.TaskPending:
			pending++
		case state.TaskInProgress:
			running++
		case state.TaskDone:
			done++
		case state.TaskStalled:
			stalled++
		}
	}
	metric := func(label string, count int, c color.Color) string {
		return DimText.Render(label+" ") + Fg(c, fmt.Sprintf("%02d", count))
	}
	add("")
	add(metric("RUN", running, Accent) + "  " + metric("WAIT", pending, Warn) + "  " + metric("DONE", done, OK))
	total := len(st.Tasks)
	filled := 0
	if total > 0 {
		filled = done * max(1, w-7) / total
	}
	barWidth := max(1, w-7)
	percent := "  --"
	if total > 0 {
		percent = fmt.Sprintf(" %3d%%", done*100/total)
	}
	add(OKText.Render(strings.Repeat("━", filled)) + Fg(CurrentTheme().Border, strings.Repeat("━", barWidth-filled)) + DimText.Render(percent))
	if stalled > 0 {
		add(ErrText.Render(fmt.Sprintf("! %d stalled · review the board", stalled)))
	} else if total == 0 {
		add(DimText.Render("Awaiting first dispatch"))
	} else {
		add(DimText.Render(fmt.Sprintf("%d of %d tasks complete", done, total)))
	}
	add("")
	section("AGENT NETWORK")
	limit := min(len(st.Employees), max(1, height-15))
	for _, e := range st.Employees[:limit] {
		status, ink := "STANDBY", Dim
		switch e.Sprite {
		case state.SpriteWorking:
			status, ink = "WORKING", Accent
		case state.SpriteToManager, state.SpriteMeeting:
			status, ink = "SYNCING", Info
		case state.SpriteAtMailbox:
			status, ink = "BLOCKED", Err
		case state.SpriteCoffee, state.SpriteToCoffee:
			status, ink = "BREAK", Warn
		case state.SpriteToDesk:
			status, ink = "ROUTING", Info
		}
		if e.Role == state.RoleManager {
			for _, msg := range st.Chat {
				if msg.From == "boss" && msg.Pending {
					status, ink = "WORKING", Accent
					break
				}
			}
			if st.BossThinking {
				status, ink = "THINKING", Accent
			}
			if st.BossDelegating {
				status, ink = "DELEGATE", Info
			}
		}
		if st.Offline {
			status, ink = "OFFLINE", Err
		}
		nameW := max(1, w-len(status)-4)
		name := ansi.Truncate(e.Name, nameW, "…")
		add(Fg(ink, "● ") + Fg(White, name) + strings.Repeat(" ", max(1, w-2-ansi.StringWidth(name)-len(status))) + Fg(ink, status))
	}
	if len(st.Employees) > limit {
		add(DimText.Render(fmt.Sprintf("+ %d more · 3 agents", len(st.Employees)-limit)))
	}
	add("")
	section("DISPATCH")
	shown := 0
	for _, task := range st.Tasks {
		if task.Status != state.TaskInProgress && task.Status != state.TaskStalled {
			continue
		}
		if shown >= max(1, height-4-len(rows)) {
			break
		}
		glyph, ink := "› ", Info
		if task.Status == state.TaskStalled {
			glyph, ink = "! ", Err
		}
		add(Fg(ink, glyph) + Fg(White, task.Title))
		shown++
	}
	if shown == 0 {
		add(DimText.Render("No tasks in flight"))
	}
	for len(rows) < height-2 {
		add("")
	}
	rows = rows[:min(len(rows), height-2)]
	footer := "4 board · 3 agents"
	if st.Offline {
		footer = "! OFFLINE · reconnecting"
	} else if st.Mode == state.ModeDemo {
		footer = "DEMO · simulated telemetry"
	}
	box := PanelBox.Width(width).Height(height).Render(strings.Join(rows, "\n"))
	return InstrumentFrame(box, "02 / OPERATIONS", footer, width)
}
