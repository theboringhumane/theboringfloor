package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestBtwPersistHydrateStripsOrphanPinAndKeepsOrder(t *testing.T) {
	scratchHome(t)
	m := New(&pinBackend{}, nil)
	sf := &SessionFile{
		Dir:     t.TempDir(),
		SavedAt: time.Now().UnixMilli(),
		Chat: []state.ChatMsg{
			{ID: "before", From: "user", Text: "before pin"},
			{ID: "pin", From: "office", Meta: "btw-pin", Text: "btw session hidden — click to reopen"},
			{ID: "after", From: "boss", Text: "after pin"},
		},
	}

	m.hydrateSession(sf)
	if len(m.st.Chat) != 3 { // two survivors, then this boot's restore notice
		t.Fatalf("hydrated chat length = %d, want 3: %+v", len(m.st.Chat), m.st.Chat)
	}
	for i, want := range []string{"before", "after"} {
		if got := m.st.Chat[i].ID; got != want {
			t.Fatalf("survivor %d = %q, want %q; chat=%+v", i, got, want, m.st.Chat)
		}
	}
	for _, msg := range m.st.Chat {
		if msg.Meta == "btw-pin" {
			t.Fatalf("orphan btw pin survived hydration: %+v", msg)
		}
	}
}

func TestBtwPersistUsesSavedBossStateWhileSideSessionActive(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	backend := &pinBackend{primary: "side-primary"}
	m := New(backend, nil)
	m.sessDir = dir
	m.st.Chat = []state.ChatMsg{{ID: "side-chat", From: "user", Text: "side conversation"}}
	m.st.Tasks = []state.BoardTask{{ID: "side-task", Title: "side task"}}
	m.st.Mails = []state.MailItem{{ID: "side-mail", Subject: "side mail"}}
	m.btwSaved = &btwSnapshot{
		chat:      []state.ChatMsg{{ID: "boss-chat", From: "user", Text: "boss conversation"}},
		tasks:     []state.BoardTask{{ID: "boss-task", Title: "boss task"}},
		mails:     []state.MailItem{{ID: "boss-mail", Subject: "boss mail"}},
		primaryID: "boss-primary",
	}

	m.persistOfficeSession(true)
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no persisted session")
	}
	if got.PrimaryID != "boss-primary" {
		t.Fatalf("persisted primary = %q, want saved boss primary %q", got.PrimaryID, "boss-primary")
	}
	if got.PrimaryID == backend.primary || len(got.Chat) != 1 || got.Chat[0].ID != "boss-chat" {
		t.Fatalf("persisted side state instead of boss state: primary=%q chat=%+v", got.PrimaryID, got.Chat)
	}
	if got.Tasks[0].ID != "boss-task" || got.Mails[0].ID != "boss-mail" {
		t.Fatalf("persisted boss surfaces = tasks=%+v mails=%+v", got.Tasks, got.Mails)
	}
}

func TestBtwPersistOfficePinUsesSavedBossStateWhileSideSessionActive(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	m := New(&pinBackend{primary: "side-primary"}, nil)
	m.sessDir = dir
	m.st.Chat = []state.ChatMsg{{ID: "side-chat", From: "user", Text: "side conversation"}}
	m.btwSaved = &btwSnapshot{
		chat:      []state.ChatMsg{{ID: "boss-chat", From: "user", Text: "boss conversation"}},
		primaryID: "boss-primary",
	}

	// persistOfficePin bypasses persistOfficeSession for the /session
	// exec-replace path, so it must share the active-/btw persistence rule.
	m.persistOfficePin("selected-side-primary")
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no persisted session")
	}
	if got.PrimaryID != "boss-primary" || len(got.Chat) != 1 || got.Chat[0].ID != "boss-chat" {
		t.Fatalf("persistOfficePin wrote side session: primary=%q chat=%+v", got.PrimaryID, got.Chat)
	}
}

func TestBtwPersistWithoutActiveSessionKeepsLiveState(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	backend := &pinBackend{primary: "live-primary"}
	m := New(backend, nil)
	m.sessDir = dir
	m.st.Chat = []state.ChatMsg{{ID: "live-chat", From: "user", Text: "live conversation"}}
	m.st.Tasks = []state.BoardTask{{ID: "live-task", Title: "live task"}}
	m.st.Mails = []state.MailItem{{ID: "live-mail", Subject: "live mail"}}

	want := Snapshot(dir, backend.primary, m.st)
	mergeBackendPins(&want, dir, m.backendName(), backend.primary)
	m.persistOfficeSession(true)
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no persisted session")
	}
	if got.PrimaryID != want.PrimaryID || !reflect.DeepEqual(got.Chat, want.Chat) ||
		!reflect.DeepEqual(got.Tasks, want.Tasks) || !reflect.DeepEqual(got.Mails, want.Mails) ||
		!reflect.DeepEqual(got.PrimaryIDs, want.PrimaryIDs) {
		t.Fatalf("non-btw persistence drifted: got=%+v want=%+v", got, want)
	}
}

func TestBtwPersistLoadsLegacySessionFileWithoutBtwFields(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	legacy := SessionFile{
		Dir:       dir,
		PrimaryID: "legacy-primary",
		Chat:      []state.ChatMsg{{ID: "legacy-chat", From: "user", Text: "legacy conversation"}},
		SavedAt:   time.Now().UnixMilli(),
	}
	plantSession(t, dir, legacy)

	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession rejected legacy session file")
	}
	if got.PrimaryID != legacy.PrimaryID || !reflect.DeepEqual(got.Chat, legacy.Chat) {
		t.Fatalf("legacy session changed on load: got=%+v want=%+v", got, legacy)
	}
}
