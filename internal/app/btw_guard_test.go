package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

type btwGuardBackend struct {
	recBackend
	swapSafe     bool
	primaryID    string
	swappedTo    []string
	swapErrs     []error
	reconciledID []string
	reconcileErr error
}

func (*btwGuardBackend) Mode() state.Mode         { return state.ModeLive }
func (b *btwGuardBackend) PrimaryOverride(string) {}
func (b *btwGuardBackend) PrimaryID() string      { return b.primaryID }
func (b *btwGuardBackend) SwapSafeMidTurn() bool  { return b.swapSafe }
func (b *btwGuardBackend) SwapPrimary(id string) error {
	b.swappedTo = append(b.swappedTo, id)
	if len(b.swapErrs) > 0 {
		err := b.swapErrs[0]
		b.swapErrs = b.swapErrs[1:]
		if err != nil {
			return err
		}
	}
	b.primaryID = id
	return nil
}
func (b *btwGuardBackend) ReconcileBoss(id string) error {
	b.reconciledID = append(b.reconciledID, id)
	return b.reconcileErr
}

func newBtwGuardModel(t *testing.T, b *btwGuardBackend) Model {
	t.Helper()
	scratchHome(t)
	m := New(b, nil)
	m.bootDone = true
	m.st.Chat = []state.ChatMsg{{ID: "main", From: "user", Text: "main question"}}
	return m
}

func TestBtwBlockedReasonAllowsWedgedPendingBoss(t *testing.T) {
	m := newBtwTestModel(t)
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "boss", From: "boss", Pending: true})
	m.wedgeNoted = true

	if got := btwBlockedReason(&m); got != "" {
		t.Fatalf("btwBlockedReason() = %q, want wedge allowed", got)
	}
}

func TestBtwBlockedReasonAllowsSwapSafePendingBoss(t *testing.T) {
	b := &btwGuardBackend{swapSafe: true}
	m := newBtwGuardModel(t, b)
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "boss", From: "boss", Pending: true})

	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwSaved == nil {
		t.Fatal("/btw must enter a side session when the backend preserves a mid-turn reply")
	}
}

func TestBtwBlockedReasonRefusesUnsafePendingBoss(t *testing.T) {
	b := &btwGuardBackend{}
	m := newBtwGuardModel(t, b)
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "boss", From: "boss", Pending: true})

	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwSaved != nil {
		t.Fatal("/btw must not move an in-flight reply on an unsafe backend")
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.Text != btwMidTurnBlocked {
		t.Fatalf("/btw refusal = %q, want %q", notice.Text, btwMidTurnBlocked)
	}
}

func TestBtwBlockedReasonAllowsNoPendingBoss(t *testing.T) {
	b := &btwGuardBackend{}
	m := newBtwGuardModel(t, b)

	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwSaved == nil {
		t.Fatal("/btw must enter a side session when no boss reply is pending")
	}
}

func TestBtwExitReconcilesSavedPrimary(t *testing.T) {
	b := &btwGuardBackend{primaryID: "btw-primary"}
	m := newBtwGuardModel(t, b)
	m.btwSaved = &btwSnapshot{chat: []state.ChatMsg{{ID: "main", From: "user", Text: "main"}}, primaryID: "boss-primary"}
	m.st.Chat = []state.ChatMsg{{ID: "btw", From: "user", Text: "side question"}}

	m = runMsg(t, m, slashMsg{text: "/done"})
	if got, want := strings.Join(b.swappedTo, ","), "boss-primary"; got != want {
		t.Fatalf("SwapPrimary calls = %q, want %q", got, want)
	}
	if got, want := strings.Join(b.reconciledID, ","), "boss-primary"; got != want {
		t.Fatalf("ReconcileBoss calls = %q, want %q", got, want)
	}
}

func TestBtwOfficeMsgAfterHideDoesNotPostOrSend(t *testing.T) {
	b := &btwGuardBackend{}
	m := newBtwGuardModel(t, b)
	m.btwSaved = nil // Esc already hid the side session while NewOffice was in flight.
	before := append([]state.ChatMsg(nil), m.st.Chat...)

	m = runMsg(t, m, btwOfficeMsg{trailing: "should not be sent"})
	if len(m.st.Chat) != len(before) {
		t.Fatalf("late btwOfficeMsg changed chat: got %+v want %+v", m.st.Chat, before)
	}
	for _, msg := range m.st.Chat {
		if strings.Contains(msg.Text, "btw session") {
			t.Fatalf("late btwOfficeMsg posted contradictory notice: %+v", msg)
		}
	}
	if len(b.sentTexts) != 0 {
		t.Fatalf("late btwOfficeMsg sent trailing text: %q", b.sentTexts)
	}
}

func TestBtwExitReconcileFailureKeepsReturn(t *testing.T) {
	b := &btwGuardBackend{primaryID: "btw-primary", reconcileErr: errors.New("unavailable")}
	m := newBtwGuardModel(t, b)
	m.btwSaved = &btwSnapshot{chat: []state.ChatMsg{{ID: "main", From: "user", Text: "main"}}, primaryID: "boss-primary"}

	m = runMsg(t, m, slashMsg{text: "/done"})
	if m.btwSaved != nil {
		t.Fatal("reconcile failure must not undo the completed return swap")
	}
	if got := strings.Join(b.reconciledID, ","); got != "boss-primary" {
		t.Fatalf("ReconcileBoss calls = %q, want boss-primary", got)
	}
}
