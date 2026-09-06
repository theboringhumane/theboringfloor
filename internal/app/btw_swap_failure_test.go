package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

type btwSwapFailureBackend struct {
	recBackend
	primaryID      string
	newOfficeErr   error
	newOfficeID    string
	newOfficeCalls int
	swapErrs       []error
	swappedTo      []string
}

func (*btwSwapFailureBackend) Mode() state.Mode         { return state.ModeLive }
func (b *btwSwapFailureBackend) PrimaryOverride(string) {}
func (b *btwSwapFailureBackend) PrimaryID() string      { return b.primaryID }
func (b *btwSwapFailureBackend) NewOffice() (string, error) {
	b.newOfficeCalls++
	if b.newOfficeErr != nil {
		return "", b.newOfficeErr
	}
	b.primaryID = b.newOfficeID
	return b.primaryID, nil
}
func (b *btwSwapFailureBackend) SwapPrimary(id string) error {
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

func newBtwSwapFailureModel(t *testing.T, b *btwSwapFailureBackend) Model {
	t.Helper()
	scratchHome(t)
	m := New(b, nil)
	m.bootDone = true
	m.st.Chat = []state.ChatMsg{{ID: "main", From: "user", Text: "main question"}}
	return m
}

func TestBtwStartNewOfficeFailureKeepsOriginalPrimaryAndMainChat(t *testing.T) {
	b := &btwSwapFailureBackend{
		primaryID:    "main-primary",
		newOfficeErr: errors.New("side session unavailable"),
	}
	m := newBtwSwapFailureModel(t, b)
	mainChat := append([]state.ChatMsg(nil), m.st.Chat...)

	m = runMsg(t, m, slashMsg{text: "/btw"})

	if got, want := b.newOfficeCalls, 1; got != want {
		t.Fatalf("NewOffice calls = %d, want %d", got, want)
	}
	if got, want := b.primaryID, "main-primary"; got != want {
		t.Fatalf("primary after failed /btw = %q, want original %q", got, want)
	}
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("failed /btw left side state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if len(m.st.Chat) != len(mainChat)+1 || !reflect.DeepEqual(m.st.Chat[:len(mainChat)], mainChat) {
		t.Fatalf("failed /btw transcript = %+v, want main transcript intact", m.st.Chat)
	}
	if hasBtwPin(m.st.Chat, "") {
		t.Fatalf("failed /btw left an orphaned pin: %+v", m.st.Chat)
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.Meta != "error" || notice.Text != "/btw: side session unavailable" {
		t.Fatalf("failed /btw notice = %+v", notice)
	}
}

func TestBtwDoneSwapFailureKeepsSideSessionForRetry(t *testing.T) {
	b := &btwSwapFailureBackend{
		primaryID:   "main-primary",
		newOfficeID: "side-primary",
		swapErrs:    []error{errors.New("main session unavailable"), nil},
	}
	m := newBtwSwapFailureModel(t, b)
	m = runMsg(t, m, slashMsg{text: "/btw"})
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "side", From: "user", Text: "side question"})
	sideChat := append([]state.ChatMsg(nil), m.st.Chat...)
	m.btwHiddenSnap = &btwSnapshot{chat: []state.ChatMsg{{ID: "hidden", From: "user", Text: "hidden side"}}}
	m.btwPinMsgID = "btw-pin-before-return"

	m = runMsg(t, m, slashMsg{text: "/done"})

	if got, want := strings.Join(b.swappedTo, ","), "main-primary"; got != want {
		t.Fatalf("SwapPrimary calls after failed /done = %q, want %q", got, want)
	}
	if got, want := b.primaryID, "side-primary"; got != want {
		t.Fatalf("primary after failed /done = %q, want side session %q", got, want)
	}
	if m.btwSaved == nil || m.btwHiddenSnap == nil || m.btwPinMsgID != "btw-pin-before-return" {
		t.Fatalf("failed /done discarded retry state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if len(m.st.Chat) != len(sideChat)+2 || !reflect.DeepEqual(m.st.Chat[:len(sideChat)], sideChat) {
		t.Fatalf("failed /done transcript = %+v, want side transcript intact", m.st.Chat)
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.Meta != "error" || notice.Text != "/done: main session unavailable" {
		t.Fatalf("failed /done notice = %+v", notice)
	}

	m = runMsg(t, m, slashMsg{text: "/done"})
	if got, want := strings.Join(b.swappedTo, ","), "main-primary,main-primary"; got != want {
		t.Fatalf("SwapPrimary calls after retry = %q, want %q", got, want)
	}
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("successful retry left btw state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if got, want := b.primaryID, "main-primary"; got != want {
		t.Fatalf("primary after /done retry = %q, want %q", got, want)
	}
	if len(m.st.Chat) == 0 || m.st.Chat[0].Text != "main question" {
		t.Fatalf("successful retry did not restore main transcript: %+v", m.st.Chat)
	}
}
