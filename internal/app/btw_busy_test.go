// btw_busy_test.go characterizes /btw's pending-boss guard and wedge behavior.
package app

import (
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestBtwStartRefusedWhileBossPending(t *testing.T) {
	// A non-swap-safe backend keeps the boss seated until its active reply ends.
	m := newBtwTestModel(t)
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "boss-pending", From: "boss", Pending: true})

	m = runMsg(t, m, slashMsg{text: "/btw"})

	if m.btwSaved != nil {
		t.Fatal("/btw must currently refuse to start while any boss message is pending")
	}
	if len(m.st.Chat) == 0 {
		t.Fatal("/btw refusal must append an error notice")
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.From != "office" || notice.Meta != "error" || notice.Text != btwMidTurnBlocked {
		t.Fatalf("/btw refusal notice = %+v, want office error %q", notice, btwMidTurnBlocked)
	}
}

func TestBtwStaysRefusedAfterBossWedge(t *testing.T) {
	// A watchdog-confirmed wedge leaves the stale placeholder intact but lets /btw proceed.
	m := newBtwTestModel(t)
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "boss-pending", From: "boss", Pending: true})
	m.lastBossActivityAt = time.Now().Add(-bossWedgeAfter - time.Second)

	m.applyEvent(state.Event{Kind: state.EvTick})

	if !m.wedgeNoted {
		t.Fatal("setup: stale pending boss turn must trigger the wedge watchdog")
	}
	if !hasPendingBoss(m.st) {
		t.Fatal("the wedge watchdog must currently leave the stale pending boss message in place")
	}
	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwSaved == nil {
		t.Fatal("/btw must start after the wedge watchdog identifies the pending boss turn as stale")
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.From != "office" || notice.Meta == "error" || !strings.Contains(notice.Text, "btw session") {
		t.Fatalf("post-wedge /btw notice = %+v, want successful btw-session notice", notice)
	}
}
