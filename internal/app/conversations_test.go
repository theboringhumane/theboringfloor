package app

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func TestConversationArchiveSurvivesNewSessionAndRejectsStaleSave(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := t.TempDir()
	w := &sessionWriter{}
	var chat []state.ChatMsg
	for i := 0; i < 350; i++ {
		chat = append(chat, state.ChatMsg{ID: fmt.Sprint(i), From: "user", Text: fmt.Sprintf("message %d", i)})
	}
	sf := Snapshot(dir, "one", state.OfficeState{Chat: chat})
	sf.Backend = "codex"
	stale := w.reserve()
	w.save(w.reserve(), dir, sf, chat, "First", "frontend")
	if w.err != nil {
		t.Fatal(w.err)
	}
	one, ok := loadConversation(dir, "codex", "one")
	if !ok || len(one.Chat) != 350 {
		t.Fatalf("lost full history: %v", one)
	}
	if one.Title != "First" || one.Team != "frontend" {
		t.Fatal("archive lost conversation context")
	}
	current, ok := LoadSession(dir)
	if !ok || len(current.Chat) != 200 {
		t.Fatal("resume snapshot is not bounded")
	}
	second := Snapshot(dir, "two", state.OfficeState{Chat: []state.ChatMsg{{ID: "new", From: "user", Text: "new conversation"}}})
	second.Backend = "claudecode"
	w.save(w.reserve(), dir, second, second.Chat, "Second", "backend")
	w.save(stale, dir, sf, chat, "old stale write", "")
	current, _ = LoadSession(dir)
	if current.PrimaryID != "two" {
		t.Fatalf("late save overwrote new pin: %s", current.PrimaryID)
	}
	one, ok = loadConversation(dir, "codex", "one")
	if !ok || len(one.Chat) != 350 {
		t.Fatal("new conversation removed old history")
	}
	rows, err := workspace.Conversations(dir)
	if err != nil || len(rows) != 2 {
		t.Fatalf("index: %+v %v", rows, err)
	}
	info, _ := os.Stat(SessionPath(dir))
	if info.Mode().Perm() != 0600 {
		t.Fatalf("snapshot permissions %o", info.Mode().Perm())
	}
}

func TestConversationReservationDoesNotWaitForDiskWriter(t *testing.T) {
	w := &sessionWriter{}
	w.mu.Lock()
	defer w.mu.Unlock()
	done := make(chan uint64, 1)
	go func() { done <- w.reserve() }()
	select {
	case seq := <-done:
		if seq != 1 {
			t.Fatal(seq)
		}
	case <-time.After(time.Second):
		t.Fatal("autosave reservation blocked the UI behind disk I/O")
	}
}
func TestBackendPinNeverCrossesTransports(t *testing.T) {
	sf := SessionFile{PrimaryID: "codex-thread", Backend: "codex"}
	if sf.primaryIDFor(config.BackendNameDefault) != "" {
		t.Fatal("Codex pin leaked into OpenCode")
	}
	if sf.primaryIDFor("codex") != "codex-thread" {
		t.Fatal("lost typed legacy slot")
	}
	legacy := SessionFile{PrimaryID: "old-opencode"}
	if legacy.primaryIDFor("opencode") != "old-opencode" {
		t.Fatal("legacy compatibility lost")
	}
}
