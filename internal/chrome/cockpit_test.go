package chrome

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestCockpitTelemetryTracksState(t *testing.T) {
	defer restoreTheme()
	SetTheme("cockpit")
	st := state.OfficeState{Mode: state.ModeLive,
		Employees: []state.Employee{{Name: "pilot", Role: state.RoleManager}, {Name: "worker", Sprite: state.SpriteWorking}},
		Tasks: []state.BoardTask{
			{Title: "Completed work", Status: state.TaskDone},
			{Title: "Ship command deck", Status: state.TaskInProgress},
			{Title: "Review mission", Status: state.TaskPending},
			{Title: "Recover worker", Status: state.TaskStalled},
		},
		BossThinking: true,
	}
	view := ansi.Strip(CockpitTelemetry(st, 48, 24))
	for _, want := range []string{"RUN 01", "WAIT 01", "DONE 01", "25%", "1 stalled", "THINKING", "WORKING", "Ship command deck", "Recover worker"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q in telemetry:\n%s", want, view)
		}
	}
	st.Offline = true
	view = ansi.Strip(CockpitTelemetry(st, 48, 24))
	if strings.Contains(view, "WORKING") || strings.Contains(view, "THINKING") || !strings.Contains(view, "OFFLINE") {
		t.Fatalf("offline telemetry advertised running agents:\n%s", view)
	}
	empty := ansi.Strip(CockpitTelemetry(state.OfficeState{}, 48, 24))
	if strings.Contains(empty, "%") || !strings.Contains(empty, "Awaiting first dispatch") {
		t.Fatalf("empty state invented task progress:\n%s", empty)
	}
}

func TestCockpitTelemetryGeometry(t *testing.T) {
	defer restoreTheme()
	SetTheme("cockpit")
	st := state.OfficeState{Mode: state.ModeDemo, Employees: []state.Employee{{Name: strings.Repeat("界", 60)}}}
	for _, width := range []int{28, 37, 48, 72, 120} {
		for _, height := range []int{16, 20, 24} {
			rows := strings.Split(CockpitTelemetry(st, width, height), "\n")
			if len(rows) != height {
				t.Fatalf("%dx%d drew %d rows", width, height, len(rows))
			}
			for _, row := range rows {
				if got := ansi.StringWidth(row); got != width {
					t.Fatalf("%dx%d row width %d: %q", width, height, got, ansi.Strip(row))
				}
			}
		}
	}
	for _, size := range [][2]int{{70, 7}, {27, 40}, {60, 27}} {
		if CockpitTelemetryHeight(size[0], size[1]) != 0 {
			t.Fatalf("instruments crowded compact floor %v", size)
		}
	}
}
