// swap_safety_test.go — optional /btw capability seams on the two live
// backends. Every OpenCode read uses an httptest serve; Claude is constructed
// only, so this file never starts a real CLI process.
package backend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

var _ interface {
	SwapSafeMidTurn() bool
	ReconcileBoss(string) error
} = (*liveBackend)(nil)

var _ interface {
	SwapSafeMidTurn() bool
	ReconcileBoss(string) error
} = (*liveClaudeBackend)(nil)

func TestSwapSafeMidTurn(t *testing.T) {
	if !newLiveBackend("", t.TempDir(), config.Default()).SwapSafeMidTurn() {
		t.Fatal("opencode swaps must preserve an in-flight server-side turn")
	}
	if newClaudeBackend("", t.TempDir(), config.Default()).SwapSafeMidTurn() {
		t.Fatal("claude swaps tear down the in-flight subprocess turn")
	}
}

func TestReconcileBossEmptySkipsHTTP(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Errorf("ReconcileBoss(\"\") must not make an HTTP request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	if err := b.ReconcileBoss(""); err != nil {
		t.Fatalf("ReconcileBoss empty: %v", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("ReconcileBoss empty made %d HTTP calls, want 0", got)
	}
}

func TestReconcileBossAlreadyEmittedDoesNotDuplicate(t *testing.T) {
	b, log := swapSafetyBackend(t, `[
  {"info":{"id":"u-1","sessionID":"ses-boss","role":"user","time":{"created":100}},"parts":[]},
  {"info":{"id":"m-1","sessionID":"ses-boss","role":"assistant","finish":"stop","time":{"created":110,"completed":120}},"parts":[]}
]`, "already pinned")
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.bossCompleted["m-1"] = true
	b.mu.Unlock()

	if err := b.ReconcileBoss("ses-boss"); err != nil {
		t.Fatalf("ReconcileBoss: %v", err)
	}
	if got := swapSafetyBossPins(log); got != 0 {
		t.Fatalf("an already-emitted completion must not emit a duplicate, got %d events", got)
	}
}

func TestReconcileBossMintsUnseatedCompletionOnce(t *testing.T) {
	b, log := swapSafetyBackend(t, `[
  {"info":{"id":"u-1","sessionID":"ses-boss","role":"user","time":{"created":100}},"parts":[]},
  {"info":{"id":"m-1","sessionID":"ses-boss","role":"assistant","finish":"stop","time":{"created":110,"completed":120}},"parts":[]}
]`, "reply recovered after /btw")
	b.mu.Lock()
	b.primaryID = "ses-btw" // Recovery must not rely on the currently seated session.
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	if err := b.ReconcileBoss("ses-boss"); err != nil {
		t.Fatalf("ReconcileBoss: %v", err)
	}
	if got := swapSafetyBossPins(log); got != 1 {
		t.Fatalf("the missed completion must emit exactly once, got %d events", got)
	}
	log.mu.Lock()
	for _, event := range log.evs {
		if event.Kind == state.EvChatBoss && !event.Msg.Pending {
			if event.Msg.ID != "bossmsg-m-1" || event.Msg.Text != "reply recovered after /btw" {
				log.mu.Unlock()
				t.Fatalf("recovered completion = %+v, want pinned m-1 reply", event.Msg)
			}
		}
	}
	log.mu.Unlock()
	if err := b.ReconcileBoss("ses-boss"); err != nil {
		t.Fatalf("second ReconcileBoss: %v", err)
	}
	if got := swapSafetyBossPins(log); got != 1 {
		t.Fatalf("dedupe must hold across repeated reconcile calls, got %d events", got)
	}
}

func TestReconcileBossReturnsTransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session/ses-boss/message" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"serve unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	b.mu.Lock()
	b.baseURL = srv.URL
	b.mu.Unlock()
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	if err := b.ReconcileBoss("ses-boss"); err == nil {
		t.Fatal("a failed session re-read must be returned")
	}
}

// swapSafetyBackend serves exactly the read endpoints ReconcileBoss follows:
// message listing, message text, and its best-effort diff fetch.
func swapSafetyBackend(t *testing.T, rows, text string) (*liveBackend, *eventLog) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-boss/message":
			_, _ = w.Write([]byte(rows))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-boss/message/m-1":
			_, _ = w.Write([]byte(`{"info":{"id":"m-1","sessionID":"ses-boss","role":"assistant","finish":"stop","time":{"created":110,"completed":120}},"parts":[{"id":"p-1","sessionID":"ses-boss","messageID":"m-1","type":"text","text":"` + text + `"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-boss/diff":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Errorf("unexpected ReconcileBoss request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	b.mu.Lock()
	b.baseURL = srv.URL
	b.mu.Unlock()
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	return b, log
}

func swapSafetyBossPins(log *eventLog) int {
	log.mu.Lock()
	defer log.mu.Unlock()
	count := 0
	for _, event := range log.evs {
		if event.Kind == state.EvChatBoss && !event.Msg.Pending && strings.HasPrefix(event.Msg.ID, "bossmsg-") {
			count++
		}
	}
	return count
}
