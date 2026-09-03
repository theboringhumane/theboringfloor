package backend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/chatcontext"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func assertRecentMessagesPin(t *testing.T, events []state.Event, wantCount int) {
	t.Helper()
	var recent, pin int = -1, -1
	for i, e := range events {
		if e.Kind == state.EvRecentMessages {
			if recent >= 0 || e.RecentMessagesCount != wantCount {
				t.Fatalf("recent-message event drifted at %d: %+v", i, e)
			}
			recent = i
		}
		if e.Kind == state.EvChatBoss && !e.Msg.Pending && strings.HasPrefix(e.Msg.ID, "bossmsg-") {
			pin = i
			if strings.Contains(e.Msg.Text, "⟦recent-messages") {
				t.Fatalf("completion pin leaked marker: %+v", e.Msg)
			}
		}
	}
	if recent < 0 || pin < 0 || recent >= pin {
		t.Fatalf("recent-message event must precede clean final pin (recent=%d pin=%d): %+v", recent, pin, events)
	}
}

func TestOpenCodeRecentMessagesMarkerEmitsBeforeCleanPin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/session/ses-boss/message/msg-1" {
			w.Write([]byte(`{"info":{"id":"msg-1","sessionID":"ses-boss","role":"assistant","finish":"stop","time":{"completed":1}},"parts":[{"type":"text","text":"I need context.\n⟦recent-messages: 99⟧"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	b.mu.Lock()
	b.baseURL, b.primaryID = srv.URL, "ses-boss"
	b.mu.Unlock()
	info := ocMessage{ID: "msg-1", SessionID: "ses-boss", Role: "assistant"}
	info.Time.Completed = 1
	b.maybeBossCompleted(info)
	assertRecentMessagesPin(t, eventsMatching(log, func(state.Event) bool { return true }), chatcontext.MaxCount)
}

func TestClaudeRecentMessagesMarkerEmitsBeforeCleanPin(t *testing.T) {
	b := newClaudeBackend("true", ".", nil)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-msg-1", From: "boss", Kind: "boss", Text: "⟦recent-messages⟧", Pending: false,
	}})
	assertRecentMessagesPin(t, log.snapshot(), chatcontext.DefaultCount)
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatBoss && e.Msg.ID == "bossmsg-msg-1" && e.Msg.Text != "[theboringfloor] recent messages requested: 20" {
			t.Fatalf("marker-only pin must settle to a useful fallback: %+v", e.Msg)
		}
	}
}
