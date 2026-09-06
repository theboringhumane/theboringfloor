// btw_pin_sticky_test.go — hidden-BTW's transcript marker remains an
// ordinary chronological bubble, while a fixed footer keeps its resume
// affordance available after history has scrolled away.
package panels

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	state "github.com/theboringhumane/theboringfloor/internal/state"
)

const btwPinText = "btw session hidden — click to reopen"

func stickyBtwState(withPin bool) state.OfficeState {
	chat := make([]state.ChatMsg, 0, 81)
	for i := 0; i < 80; i++ {
		chat = append(chat, state.ChatMsg{
			ID:   "u-" + itoa(i),
			From: "user",
			Kind: "user",
			Text: "history row " + itoa(i),
		})
	}
	if withPin {
		chat = append(chat, state.ChatMsg{ID: "btw-pin", From: officeFrom, Meta: "btw-pin", Text: btwPinText})
	}
	return state.OfficeState{Tick: 1, Chat: chat}
}

// TestBtwPinStickyFooterSurvivesTopScroll proves the fixed marker is outside
// the transcript viewport: even at absolute transcript row zero, it paints
// directly below the viewport and claims that content-coordinate row.
func TestBtwPinStickyFooterSurvivesTopScroll(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 14)
	c.SetState(stickyBtwState(true))
	if got, want := len(c.selLines) > 3*c.vp.Height(), true; got != want {
		t.Fatalf("fixture must exceed three viewport heights: transcript=%d viewport=%d", len(c.selLines), c.vp.Height())
	}
	c.vp.SetYOffset(0)
	view := ansi.Strip(c.View())
	rows := strings.Split(view, "\n")
	stickyY := c.vp.Height()
	if stickyY >= len(rows) || !strings.Contains(rows[stickyY], "↩ "+btwPinText) {
		t.Fatalf("sticky row missing at fixed footer y=%d:\n%s", stickyY, view)
	}
	if !c.BtwPinRowAt(0, stickyY) {
		t.Fatalf("sticky footer y=%d must resume hidden BTW", stickyY)
	}
	if !c.ClickRow(0, stickyY) {
		t.Fatalf("sticky footer y=%d must be swallowed by ClickRow", stickyY)
	}
	if c.BtwPinRowAt(0, stickyY+1) {
		t.Fatalf("row below sticky footer must not claim BTW pin")
	}
	if strings.Contains(strings.Join(rows[:c.vp.Height()], "\n"), btwPinText) {
		t.Fatalf("top-scrolled transcript unexpectedly contains tail pin bubble:\n%s", view)
	}
}

// TestBtwPinStickyKeepsTranscriptAndOtherHitMaps verifies the old absolute
// pin map still works when the chronological bubble is visible, and shrinking
// the viewport did not shift unrelated transcript hit maps.
func TestBtwPinStickyKeepsTranscriptAndOtherHitMaps(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 14)
	c.SetState(stickyBtwState(true))
	pinRow := -1
	for row, id := range c.btwPinRows {
		if id == "btw-pin" {
			pinRow = row
			break
		}
	}
	if pinRow < 0 {
		t.Fatalf("in-transcript BTW pin missing from absolute hit-map")
	}
	c.vp.SetYOffset(pinRow)
	c.View() // materialize the moved viewport window
	pinY := pinRow - c.vp.YOffset()
	if !c.BtwPinRowAt(0, pinY) {
		t.Fatalf("visible in-transcript pin row %d must remain clickable", pinRow)
	}

	thread := NewChat(nil)
	thread.SetSize(60, 14)
	thread.SetState(state.OfficeState{Tick: 1, Employees: []state.Employee{{
		ID: "dev", Name: "tekton-pin", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk, Task: "check hit map",
	}}, Chat: []state.ChatMsg{
		{ID: "w", From: "tekton-pin", Kind: wtoolKind, Text: "Read panels", Meta: "done\x1f1"},
		{ID: "btw-pin", From: officeFrom, Meta: "btw-pin", Text: btwPinText},
	}})
	var absolute int
	for row := range thread.threadRows {
		absolute = row
		break
	}
	thread.vp.SetYOffset(absolute)
	thread.View()
	threadY := absolute - thread.vp.YOffset()
	if name, ok := thread.ThreadRowAt(0, threadY); !ok || name != "tekton-pin" {
		t.Fatalf("sticky reservation shifted thread hit-map: got (%q, %t), want tekton-pin", name, ok)
	}
}

// TestBtwPinStickyAbsentKeepsViewportAndHitBehavior proves no-pin layout
// remains the historical viewport budget and claims no sticky or transcript
// row.
func TestBtwPinStickyAbsentKeepsViewportAndHitBehavior(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 14)
	baseline := c.vp.Height()
	c.SetState(stickyBtwState(false))
	if got := c.vp.Height(); got != baseline {
		t.Fatalf("no pin changed viewport height: got %d, want %d", got, baseline)
	}
	view := ansi.Strip(c.View())
	if strings.Contains(view, btwPinText) {
		t.Fatalf("no-pin state rendered sticky copy:\n%s", view)
	}
	for y := -1; y <= c.vp.Height()+1; y++ {
		if c.BtwPinRowAt(0, y) {
			t.Fatalf("no-pin state claimed y=%d", y)
		}
	}
}

func TestBtwPinStickyTinyHeights(t *testing.T) {
	for _, h := range []int{1, 2, 3} {
		t.Run("height-"+itoa(h), func(t *testing.T) {
			c := NewChat(nil)
			c.SetSize(20, h)
			c.SetState(stickyBtwState(true))
			if got := c.vp.Height(); got < 0 {
				t.Fatalf("negative viewport height at panel height %d: %d", h, got)
			}
			_ = c.View() // must not panic while the footer guard is active
		})
	}
}
