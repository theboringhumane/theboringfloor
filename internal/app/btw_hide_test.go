// btw_hide_test.go pins Esc's hide/resume lifecycle for /btw side sessions.
package app

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func newBtwTestModel(t *testing.T) Model {
	t.Helper()
	scratchHome(t)
	m := New(&recBackend{}, nil)
	m.bootDone = true
	m.st.Chat = []state.ChatMsg{
		{ID: "main-user", From: "user", Text: "main question"},
		{ID: "main-boss", From: "boss", Text: "main answer"},
	}
	return m
}

func startBtw(t *testing.T, m Model) Model {
	t.Helper()
	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwSaved == nil {
		t.Fatal("/btw must enter an active side session")
	}
	return m
}

func hasBtwPin(chat []state.ChatMsg, id string) bool {
	for _, msg := range chat {
		if msg.ID == id && msg.From == "office" && msg.Meta == "btw-pin" {
			return true
		}
	}
	return false
}

func TestBtwHidePreservesSessionAndPinsMainChat(t *testing.T) {
	m := newBtwTestModel(t)
	mainChat := append([]state.ChatMsg(nil), m.st.Chat...)
	m = startBtw(t, m)
	btwChat := append([]state.ChatMsg(nil), m.st.Chat...)

	if cmd := m.hideBtw(); cmd != nil {
		t.Fatal("demo hide must not schedule a backend swap")
	}
	if m.btwSaved != nil {
		t.Fatal("hide must leave no active btw save slot")
	}
	if m.btwHiddenSnap == nil {
		t.Fatal("hide must preserve the side session in the hidden slot")
	}
	if m.btwPinMsgID == "" {
		t.Fatal("hide must mint a pinned-bubble ID")
	}
	if !reflect.DeepEqual(m.btwHiddenSnap.chat, btwChat) {
		t.Fatalf("hidden side transcript = %+v, want %+v", m.btwHiddenSnap.chat, btwChat)
	}
	if len(m.st.Chat) < len(mainChat)+1 || !reflect.DeepEqual(m.st.Chat[:len(mainChat)], mainChat) {
		t.Fatalf("hide must restore main chat before the pin and return notice: got %+v want prefix %+v", m.st.Chat, mainChat)
	}
	if !hasBtwPin(m.st.Chat, m.btwPinMsgID) {
		t.Fatalf("restored main chat must contain office btw-pin %q: %+v", m.btwPinMsgID, m.st.Chat)
	}
}

func TestBtwHideWithoutActiveSessionAddsErrorNotice(t *testing.T) {
	m := newBtwTestModel(t)
	before := append([]state.ChatMsg(nil), m.st.Chat...)

	if cmd := m.hideBtw(); cmd != nil {
		t.Fatal("hide outside a side session must not schedule work")
	}
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("hide outside a side session changed lifecycle state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if len(m.st.Chat) != len(before)+1 || !reflect.DeepEqual(m.st.Chat[:len(before)], before) {
		t.Fatalf("hide outside a side session changed transcript beyond its notice: %+v", m.st.Chat)
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.From != "office" || notice.Meta != "error" {
		t.Fatalf("hide outside a side session must append noticeErr, got %+v", notice)
	}
}

func TestBtwResumeRestoresHiddenSessionAndRemovesPin(t *testing.T) {
	m := startBtw(t, newBtwTestModel(t))
	btwChat := append([]state.ChatMsg(nil), m.st.Chat...)
	if cmd := m.hideBtw(); cmd != nil {
		t.Fatal("setup hide must not schedule work in demo")
	}
	pinnedID := m.btwPinMsgID

	if cmd := m.resumeBtw(); cmd != nil {
		t.Fatal("demo resume must not schedule a backend swap")
	}
	if m.btwHiddenSnap != nil {
		t.Fatal("resume must consume the hidden side snapshot")
	}
	if m.btwSaved == nil {
		t.Fatal("resume must return to an active side session")
	}
	if m.btwPinMsgID != "" {
		t.Fatalf("resume must clear the pinned-bubble ID, got %q", m.btwPinMsgID)
	}
	if len(m.st.Chat) < len(btwChat) || !reflect.DeepEqual(m.st.Chat[:len(btwChat)], btwChat) {
		t.Fatalf("resume transcript must restore hidden side chat before its notice: got %+v want prefix %+v", m.st.Chat, btwChat)
	}
	if hasBtwPin(m.btwSaved.chat, pinnedID) {
		t.Fatalf("saved main chat must not retain resumed pin %q: %+v", pinnedID, m.btwSaved.chat)
	}
}

func TestBtwResumeWithoutHiddenSessionAddsErrorNotice(t *testing.T) {
	m := newBtwTestModel(t)
	before := append([]state.ChatMsg(nil), m.st.Chat...)

	if cmd := m.resumeBtw(); cmd != nil {
		t.Fatal("resume without a hidden side session must not schedule work")
	}
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("resume without hidden state changed lifecycle state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if len(m.st.Chat) != len(before)+1 || !reflect.DeepEqual(m.st.Chat[:len(before)], before) {
		t.Fatalf("resume without a hidden side session changed transcript beyond its notice: %+v", m.st.Chat)
	}
	notice := m.st.Chat[len(m.st.Chat)-1]
	if notice.From != "office" || notice.Meta != "error" {
		t.Fatalf("resume without a hidden side session must append noticeErr, got %+v", notice)
	}
}

func TestBtwExitClearsHiddenStateAndPin(t *testing.T) {
	m := startBtw(t, newBtwTestModel(t))
	m.btwHiddenSnap = &btwSnapshot{chat: []state.ChatMsg{{ID: "hidden", From: "user", Text: "keep this hidden"}}}
	m.btwPinMsgID = "btw-pin-stale"

	if cmd := m.exitBtw(); cmd != nil {
		t.Fatal("demo /done must not schedule a backend swap")
	}
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("/done must permanently clear btw lifecycle state: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
}

func TestBtwNewClearsHiddenStateAndPin(t *testing.T) {
	m := newBtwTestModel(t)
	m.btwHiddenSnap = &btwSnapshot{chat: []state.ChatMsg{{ID: "hidden", From: "user", Text: "keep this hidden"}}}
	m.btwPinMsgID = "btw-pin-stale"

	m = runMsg(t, m, slashMsg{text: "/new"})
	if m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("/new must clear hidden btw state and pin: hidden=%+v pin=%q", m.btwHiddenSnap, m.btwPinMsgID)
	}
}

func TestBtwHideResumeDoneCycle(t *testing.T) {
	m := newBtwTestModel(t)
	mainChat := append([]state.ChatMsg(nil), m.st.Chat...)

	m = startBtw(t, m) // /btw
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if !m.btwHidden() || m.btwSaved != nil || m.btwPinMsgID == "" {
		t.Fatalf("Esc must hide the side session: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}

	m = runMsg(t, m, slashMsg{text: "/btw"})
	if m.btwHidden() || m.btwSaved == nil || m.btwPinMsgID != "" {
		t.Fatalf("/btw must resume the hidden session: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}

	m = runMsg(t, m, slashMsg{text: "/done"})
	if m.btwSaved != nil || m.btwHiddenSnap != nil || m.btwPinMsgID != "" {
		t.Fatalf("/done must permanently exit after resume: saved=%+v hidden=%+v pin=%q", m.btwSaved, m.btwHiddenSnap, m.btwPinMsgID)
	}
	if len(m.st.Chat) < len(mainChat) || !reflect.DeepEqual(m.st.Chat[:len(mainChat)], mainChat) {
		t.Fatalf("full cycle must restore the original main transcript before its return notices: got %+v want prefix %+v", m.st.Chat, mainChat)
	}
}
