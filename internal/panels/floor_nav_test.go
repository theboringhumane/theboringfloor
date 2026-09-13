package panels

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func TestFloorNavShowsCurrentLiveBusyLiveIdleAndNotRunning(t *testing.T) {
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
	}
	frame := ansi.Strip(f.NavView(36, 40, false))

	if !strings.Contains(frame, "● current") {
		t.Fatalf("nav current floor missing its plain marker:\n%s", frame)
	}
	if !strings.Contains(frame, "◆ busy") || !strings.Contains(frame, "working") {
		t.Fatalf("nav live+busy floor missing its badge/label:\n%s", frame)
	}
	if !strings.Contains(frame, "○ idle") {
		t.Fatalf("nav live+idle floor missing its quiet marker:\n%s", frame)
	}
	if strings.Contains(frame, "◆ not-running") || strings.Contains(frame, "○ not-running") {
		t.Fatalf("nav not-running floor rendered a live marker:\n%s", frame)
	}
}

// TestFloorNavNeedsYouOutranksWorking mirrors floors_test.go's precedence
// case in the persistent navigator rail: a floor that is both busy AND
// parked on a prompt must render as needs-you, never as merely working,
// and a stale needs-you entry on an offline floor must never surface.
func TestFloorNavNeedsYouOutranksWorking(t *testing.T) {
	current, parked, busyParked, dead := "/work/current", "/work/parked", "/work/busy-parked", "/work/not-running"
	f := newTestFloors(current, []workspace.Floor{
		{Dir: current, Name: "current"},
		{Dir: parked, Name: "parked"},
		{Dir: busyParked, Name: "busy-parked"},
		{Dir: dead, Name: "not-running"},
	})
	f.status = FloorStatus{
		control.DirHash(parked):     {Live: true, Busy: false, NeedsYou: true},
		control.DirHash(busyParked): {Live: true, Busy: true, NeedsYou: true},
		control.DirHash(dead):       {Live: false, Busy: false, NeedsYou: true}, // stale
	}
	frame := ansi.Strip(f.NavView(36, 40, false))

	if !strings.Contains(frame, "▲ parked") || !strings.Contains(frame, "needs you") {
		t.Fatalf("nav live+parked floor missing its needs-you marker/label:\n%s", frame)
	}
	if !strings.Contains(frame, "▲ busy-parked") {
		t.Fatalf("nav busy+parked floor missing its needs-you marker (precedence):\n%s", frame)
	}
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
		t.Fatalf("busy+parked floor rendered the working marker/label:\n%s", busyParkedBlock)
	}
	if strings.Contains(frame, "▲ not-running") {
		t.Fatalf("stale needs-you entry on an offline floor rendered a needs-you marker:\n%s", frame)
	}
	if strings.Contains(frame, "▲ current") {
		t.Fatalf("current floor rendered a needs-you marker:\n%s", frame)
	}
}

// TestFloorNavBadgeNeverOverflowsRowWidth proves the badge cannot push a
// nav row past NavView's own width budget: every rendered line (marker,
// name, and the "N teams · M tickets [· working]" stats line alike) is
// truncated to fit the narrow rail, exactly as it was before this change.
func TestFloorNavBadgeNeverOverflowsRowWidth(t *testing.T) {
	current, busy := "/work/current", "/work/a-fairly-long-project-directory-name"
	f := newTestFloors(current, []workspace.Floor{
		{Dir: current, Name: "current"},
		{Dir: busy, Name: "a-fairly-long-project-directory-name"},
	})
	f.status = FloorStatus{control.DirHash(busy): {Live: true, Busy: true}}
	const w = 24
	frame := f.NavView(w, 40, false)
	for i, line := range strings.Split(frame, "\n") {
		if width := ansi.StringWidth(line); width > w {
			t.Fatalf("nav line %d exceeds width budget %d: %q (width %d)", i, w, ansi.Strip(line), width)
		}
	}
	// At this narrow width the "· working" suffix is legitimately truncated
	// away (same as any other long stats-line text would be) -- what must
	// survive is the marker glyph itself, which sits at the front of the
	// name line and is truncated last.
	if stripped := ansi.Strip(frame); !strings.Contains(stripped, "◆") {
		t.Fatalf("width-constrained frame lost the busy marker entirely:\n%s", stripped)
	}
}

// TestFloorNavNeedsYouBadgeNeverOverflowsRowWidth is the same overflow proof
// as TestFloorNavBadgeNeverOverflowsRowWidth, but for " · needs you" -- the
// longest of the two remote-status labels -- against a long project name,
// at the narrowest width the navigator supports.
func TestFloorNavNeedsYouBadgeNeverOverflowsRowWidth(t *testing.T) {
	current, parked := "/work/current", "/work/a-fairly-long-project-directory-name"
	f := newTestFloors(current, []workspace.Floor{
		{Dir: current, Name: "current"},
		{Dir: parked, Name: "a-fairly-long-project-directory-name"},
	})
	f.status = FloorStatus{control.DirHash(parked): {Live: true, Busy: true, NeedsYou: true}}
	const w = 24
	frame := f.NavView(w, 40, false)
	for i, line := range strings.Split(frame, "\n") {
		if width := ansi.StringWidth(line); width > w {
			t.Fatalf("nav line %d exceeds width budget %d: %q (width %d)", i, w, ansi.Strip(line), width)
		}
	}
	// The "· needs you" suffix is legitimately truncated away at this
	// width, same as "· working" is above -- what must survive is the
	// needs-you marker glyph itself, at the front of the name line.
	if stripped := ansi.Strip(frame); !strings.Contains(stripped, "▲") {
		t.Fatalf("width-constrained frame lost the needs-you marker entirely:\n%s", stripped)
	}
}

func TestFloorNavRendersSafelyWithNilStatus(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NavView() panicked with a nil status map: %v", r)
		}
	}()
	f := newTestFloors("/work/only", []workspace.Floor{{Dir: "/work/only", Name: "only"}, {Dir: "/work/other", Name: "other"}})
	frame := ansi.Strip(f.NavView(30, 30, true))
	if strings.Contains(frame, "◆") || strings.Contains(frame, "○") {
		t.Fatalf("nil status map produced a live marker:\n%s", frame)
	}
}
