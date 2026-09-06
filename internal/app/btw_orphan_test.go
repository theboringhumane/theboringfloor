// btw_orphan_test.go characterizes persisted /btw pin bubbles without their in-memory side snapshot.
package app

import (
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestBtwPinSurvivesRestartButResumeFails(t *testing.T) {
	// Persisted pins have no durable side-session snapshot, so hydration drops
	// them rather than presenting a return action that cannot succeed.
	m := newBtwTestModel(t)
	saved := &SessionFile{
		Dir:     t.TempDir(),
		SavedAt: time.Now().UnixMilli(),
		Chat: []state.ChatMsg{{
			ID:   "orphaned-btw-pin",
			From: "office",
			Meta: "btw-pin",
			Text: "btw session hidden — click to reopen",
		}},
	}

	m.hydrateSession(saved)

	if hasBtwPin(m.st.Chat, "orphaned-btw-pin") {
		t.Fatalf("hydrated chat must discard the orphaned btw pin: %+v", m.st.Chat)
	}
	if m.btwHiddenSnap != nil {
		t.Fatalf("restart simulation must currently have no in-memory hidden snapshot, got %+v", m.btwHiddenSnap)
	}
	if cmd := m.resumeBtw(); cmd != nil {
		t.Fatal("resume without restored hidden state must not schedule backend work")
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.From != "office" || notice.Meta != "error" || !strings.Contains(notice.Text, "no hidden btw session") {
		t.Fatalf("orphaned-pin resume notice = %+v, want office error containing %q", notice, "no hidden btw session")
	}
}

func TestBtwWithOrphanPinStartsNewSession(t *testing.T) {
	// After hydration removes an orphaned pin, /btw starts a fresh side session.
	m := newBtwTestModel(t)
	saved := &SessionFile{
		Dir:     t.TempDir(),
		SavedAt: time.Now().UnixMilli(),
		Chat: []state.ChatMsg{{
			ID:   "orphaned-btw-pin",
			From: "office",
			Meta: "btw-pin",
			Text: "btw session hidden — click to reopen",
		}},
	}

	m.hydrateSession(saved)
	if m.btwHiddenSnap != nil {
		t.Fatalf("restart simulation must begin without a hidden side snapshot, got %+v", m.btwHiddenSnap)
	}
	if hasBtwPin(m.st.Chat, "orphaned-btw-pin") {
		t.Fatalf("setup: hydrated chat must discard orphaned pin: %+v", m.st.Chat)
	}

	m = runMsg(t, m, slashMsg{text: "/btw"})

	if m.btwSaved == nil {
		t.Fatal("/btw must currently start a brand-new side session beside an orphaned pin")
	}
	if hasBtwPin(m.st.Chat, "orphaned-btw-pin") {
		t.Fatalf("new side session must currently clear the visible main chat, got %+v", m.st.Chat)
	}
}
