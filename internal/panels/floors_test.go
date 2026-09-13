package panels

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

// newTestFloors builds a Floors panel with explicit rows, bypassing the
// async Refresh()/workspace.List() path so tests can pin exact floor
// directories and names.
func newTestFloors(dir string, rows []workspace.Floor) *Floors {
	f := NewFloors(dir, "opencode", true)
	f.rows = rows
	f.conversations = map[string][]workspace.Conversation{}
	f.SetSize(100, 30)
	return f
}

func TestFloorsViewShowsCurrentLiveBusyLiveIdleAndNotRunning(t *testing.T) {
	current, busy, idle, dead := "/work/current", "/work/busy", "/work/idle", "/work/not-running"
	f := newTestFloors(current, []workspace.Floor{
		{Dir: current, Name: "current"},
		{Dir: busy, Name: "busy"},
		{Dir: idle, Name: "idle"},
		{Dir: dead, Name: "not-running"},
	})
	f.status = FloorStatus{
		control.DirHash(busy): {Live: true, Busy: true},
		control.DirHash(idle): {Live: true, Busy: false},
		// dead is deliberately absent -- an unprobed/unreachable floor.
	}
	frame := ansi.Strip(f.View())

	if !strings.Contains(frame, "● current") {
		t.Fatalf("current floor missing its plain marker:\n%s", frame)
	}
	if !strings.Contains(frame, "◆ busy") {
		t.Fatalf("live+busy floor missing its working marker:\n%s", frame)
	}
	if !strings.Contains(frame, "working") {
		t.Fatalf("live+busy floor missing its short label:\n%s", frame)
	}
	if !strings.Contains(frame, "○ idle") {
		t.Fatalf("live+idle floor missing its quiet marker:\n%s", frame)
	}
	if strings.Contains(frame, "◆ not-running") || strings.Contains(frame, "○ not-running") {
		t.Fatalf("not-running floor rendered a live marker:\n%s", frame)
	}
	// The current floor must never carry a remote badge, even though it is
	// the selected row here (index 0) and rendered through the TabActive
	// branch rather than the plain PanelHeader branch.
	if strings.Contains(frame, "◆ current") || strings.Contains(frame, "○ current") {
		t.Fatalf("current floor carried a remote badge:\n%s", frame)
	}
}

func TestFloorsViewCurrentFloorNeverBadgedEvenIfProbed(t *testing.T) {
	// Defensive: even if a stray status entry existed for the current
	// floor's dir (it never should, since StatusSweepCmd excludes it), the
	// renderer must still force the plain "●" and no badge -- including the
	// new needs-you state, which is the one a member must not miss.
	current := "/work/current"
	f := newTestFloors(current, []workspace.Floor{{Dir: current, Name: "current"}})
	f.status = FloorStatus{control.DirHash(current): {Live: true, Busy: true, NeedsYou: true}}
	frame := ansi.Strip(f.View())
	if !strings.Contains(frame, "● current") {
		t.Fatalf("current floor missing its plain marker:\n%s", frame)
	}
	if strings.Contains(frame, "◆") || strings.Contains(frame, "▲") || strings.Contains(frame, "working") || strings.Contains(frame, "needs you") {
		t.Fatalf("current floor showed a remote badge despite a stray status entry:\n%s", frame)
	}
}

// TestFloorsViewNeedsYouOutranksWorking is the precedence case that matters
// most: a floor that is both busy AND parked on a permission prompt or boss
// question must render as needs-you, never as merely working, because
// needs-you is the state a member must act on.
func TestFloorsViewNeedsYouOutranksWorking(t *testing.T) {
	current, parked, busyParked, idle, dead := "/work/current", "/work/parked", "/work/busy-parked", "/work/idle", "/work/not-running"
	f := newTestFloors(current, []workspace.Floor{
		{Dir: current, Name: "current"},
		{Dir: parked, Name: "parked"},
		{Dir: busyParked, Name: "busy-parked"},
		{Dir: idle, Name: "idle"},
		{Dir: dead, Name: "not-running"},
	})
	f.status = FloorStatus{
		control.DirHash(parked):     {Live: true, Busy: false, NeedsYou: true},
		control.DirHash(busyParked): {Live: true, Busy: true, NeedsYou: true},
		control.DirHash(idle):       {Live: true, Busy: false, NeedsYou: false},
		// dead is stale: it claims NeedsYou but is NOT live -- it must
		// never render as needs-you, exactly like it must never render as
		// working or idle.
	}
	f.status[control.DirHash(dead)] = FloorState{Live: false, Busy: false, NeedsYou: true}
	frame := ansi.Strip(f.View())

	if !strings.Contains(frame, "▲ parked") || !strings.Contains(frame, "needs you") {
		t.Fatalf("live+parked floor missing its needs-you marker/label:\n%s", frame)
	}
	if !strings.Contains(frame, "▲ busy-parked") {
		t.Fatalf("busy+parked floor missing its needs-you marker (precedence):\n%s", frame)
	}
	// The precedence case: busy-parked must show needs-you, and must NOT
	// also show the working marker/label anywhere on its own row.
	busyParkedIdx := strings.Index(frame, "busy-parked")
	if busyParkedIdx < 0 {
		t.Fatalf("busy-parked row not found:\n%s", frame)
	}
	statsLineEnd := strings.Index(frame[busyParkedIdx:], "\n\n")
	if statsLineEnd < 0 {
		statsLineEnd = len(frame) - busyParkedIdx
	}
	busyParkedBlock := frame[busyParkedIdx : busyParkedIdx+statsLineEnd]
	if strings.Contains(busyParkedBlock, "◆") || strings.Contains(busyParkedBlock, "· working") {
		t.Fatalf("busy+parked floor rendered the working marker/label instead of (or alongside) needs-you:\n%s", busyParkedBlock)
	}
	if !strings.Contains(busyParkedBlock, "needs you") {
		t.Fatalf("busy+parked floor block missing needs-you label:\n%s", busyParkedBlock)
	}
	if strings.Contains(frame, "▲ not-running") {
		t.Fatalf("stale needs-you entry on an offline floor rendered a needs-you marker:\n%s", frame)
	}
	if strings.Contains(frame, "▲ current") {
		t.Fatalf("current floor rendered a needs-you marker:\n%s", frame)
	}
	if !strings.Contains(frame, "○ idle") {
		t.Fatalf("live+idle floor missing its quiet marker:\n%s", frame)
	}
}

func TestFloorsViewRendersSafelyWithNilStatus(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View() panicked with a nil status map: %v", r)
		}
	}()
	f := newTestFloors("/work/only", []workspace.Floor{{Dir: "/work/only", Name: "only"}, {Dir: "/work/other", Name: "other"}})
	// f.status is nil here -- the state before the first sweep has
	// returned (first frame), or while a sweep is still in flight.
	frame := ansi.Strip(f.View())
	if strings.Contains(frame, "◆") || strings.Contains(frame, "○") {
		t.Fatalf("nil status map produced a live marker:\n%s", frame)
	}
}
