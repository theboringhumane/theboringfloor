// stuck_test.go — the boss-stuck-busy edge cases, backend leg (httptest
// doubles only, never a real serve):
//
//	W3 (G2) — SSE reattach reconcile: sseRecovered with pendingBoss
//	     outstanding runs ONE bounded truth pass over GET
//	     /session/{id}/message — a completed assistant reply newer than
//	     the placeholder's user-prompt window mints the SAME completion
//	     the live pump would (identity, dedupe, abort window, FIFO pop);
//	     a server with nothing newer mints NOTHING (no fake success);
//	W4 (G3-lite) — serve-death detector: an abnormal proc exit posts the
//	     serveDied martker row EXACTLY ONCE + latches serveDied (the next
//	     send respawns); the Stop()-initiated kill stays silent;
//	W5 — session.error on the primary pops the pendingBoss FIFO head ONCE
//	     (the turn's death certificate) and never double-pops vs the
//	     AbortSessions quiet-window swallow.
package backend

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/netwatch"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// reconcileFixture boots a live backend against a serve whose message
// listing says rows (json array of {info,parts}) and whose per-message
// text endpoint resolves textFor (messageID → text) — everything else is
// startLiveForTest's shape (empty /session list, POST-creates ses-primary,
// empty-200 /event so the pump pass EOFs instantly, /diff 404).
func reconcileFixture(t *testing.T, rows string, textFor map[string]string) (*liveBackend, *eventLog) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			w.Write([]byte(`{"id":"ses-primary","title":"theboringoffice office","time":{"created":1,"updated":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-primary/message":
			w.Write([]byte(rows))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/session/ses-primary/message/"):
			id := strings.TrimPrefix(r.URL.Path, "/session/ses-primary/message/")
			text, ok := textFor[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			body := strings.ReplaceAll(text, `"`, `\"`)
			w.Write([]byte(`{"info":{"id":"` + id + `","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[{"id":"p1","sessionID":"ses-primary","messageID":"` + id + `","type":"text","text":"` + body + `"}]}`))
		case strings.HasPrefix(r.URL.Path, "/event"):
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")
	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	b.net = netwatch.New((&scriptedProbe{online: true}).probe, 2*time.Millisecond)
	log := &eventLog{}
	if err := b.Start(log.emit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Stop() })
	return b, log
}

// bossPins counts completed boss bubbles in the log (the reconcile mint
// rides the same EvChatBoss identity as the live pump).
func bossPins(log *eventLog) []state.Event {
	return eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvChatBoss && !e.Msg.Pending
	})
}

// W3(a) — reattach with a completion the stream missed mints it now:
// identity "bossmsg-"+id, pinned fetch text, FIFO drained, dedupe map
// stamped; the SAME window's OLD history (completed before the outstanding
// user prompt) stays unpinned.
func TestReconcileMintsMissedCompletion(t *testing.T) {
	rows := `[
	  {"info":{"id":"u-old","sessionID":"ses-primary","role":"user","time":{"created":40}},"parts":[]},
	  {"info":{"id":"m-old","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":50,"completed":60},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]},
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-re","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, map[string]string{"m-re": "reconciled reply — landed while the stream was down"})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	pins := bossPins(log)
	if len(pins) != 1 {
		t.Fatalf("the missed completion must pin exactly once, got %d", len(pins))
	}
	pin := pins[0]
	if pin.Msg.ID != "bossmsg-m-re" {
		t.Fatalf("reconcile identity must match the live pump's, got %q", pin.Msg.ID)
	}
	if pin.Msg.Text != "reconciled reply — landed while the stream was down" {
		t.Fatalf("the pinned text comes from the per-message fetch, got %q", pin.Msg.Text)
	}
	b.mu.Lock()
	if len(b.pendingBoss) != 0 {
		b.mu.Unlock()
		t.Fatalf("the placeholder's FIFO head drained with its completion, got %v", b.pendingBoss)
	}
	if !b.bossCompleted["m-re"] {
		b.mu.Unlock()
		t.Fatal("the dedupe map must hold the reconciled completion")
	}
	b.mu.Unlock()
	for _, p := range pins {
		if p.Msg.ID == "bossmsg-m-old" {
			t.Fatal("history older than the placeholder's user prompt must stay unpinned")
		}
	}

	// bounded: a second reattach reconciles again but the dedupe map wins.
	b.sseRecovered()
	if n := len(bossPins(log)); n != 1 {
		t.Fatalf("the same completion must never pin twice (dedupe), got %d", n)
	}
}

// W3(b) — no completion server-side → NOTHING is minted (no fake success);
// the placeholder rides on for the next reattach.
func TestReconcileNoCompletionNoFake(t *testing.T) {
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, nil)
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	if n := len(bossPins(log)); n != 0 {
		t.Fatalf("nothing completed server-side — no bubble may be faked, got %d", n)
	}
	b.mu.Lock()
	if len(b.pendingBoss) != 1 || b.pendingBoss[0] != "boss-1" {
		t.Fatalf("the placeholder must ride on untouched, got %v", b.pendingBoss)
	}
	b.mu.Unlock()
}

// W3(c) — the abort-quiet window rides inside maybeBossCompleted: an EMPTY
// completion right after AbortSessions is the aborted turn's death rattle
// — swallowed by the reconcile exactly like by the live pump (FIFO pops,
// no bubble).
func TestReconcileRespectsAbortWindow(t *testing.T) {
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-ab","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, map[string]string{"m-ab": ""}) // empty text death rattle
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.lastAbortAt = nowMs()
	b.mu.Unlock()

	b.sseRecovered()

	if n := len(bossPins(log)); n != 0 {
		t.Fatalf("the aborted turn's empty death rattle must be swallowed, got %d bubbles", n)
	}
	b.mu.Lock()
	if len(b.pendingBoss) != 0 {
		t.Fatalf("the rattle still drains the FIFO (abort semantics), got %v", b.pendingBoss)
	}
	b.mu.Unlock()
}

// W4(a) — abnormal exit posts EXACTLY ONCE + latches serveDied; a replay
// for the same dead proc (watch racing a second exit read) stays silent
// because the proc is no longer current.
func TestServeDeathNoticeOnce(t *testing.T) {
	b := newLiveBackend("http://127.0.0.1:1", t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	cmd := exec.Command("sleep", "60") // identity token only — never started
	b.mu.Lock()
	b.proc = cmd
	b.mu.Unlock()

	exit := make(chan error, 1)
	done := make(chan struct{})
	go func() { b.watchServe(cmd, exit); close(done) }()
	exit <- errors.New("exit status 1")
	<-done

	if n := log.textCount("opencode serve died (exited"); n != 1 {
		t.Fatalf("the death row must post exactly once, got %d", n)
	}
	b.mu.Lock()
	died := b.serveDied
	cur := b.proc
	b.mu.Unlock()
	if !died {
		t.Fatal("an abnormal exit must latch serveDied (the next send respawns)")
	}
	if cur != nil {
		t.Fatal("the dead proc must be released so the next send can replace it")
	}

	// replay: the same proc's "exit" landing again must NOT re-post.
	exit2 := make(chan error, 1)
	done2 := make(chan struct{})
	go func() { b.watchServe(cmd, exit2); close(done2) }()
	exit2 <- errors.New("exit status 1")
	<-done2
	if n := log.textCount("opencode serve died (exited"); n != 1 {
		t.Fatalf("a stale exit read must stay silent, got %d rows", n)
	}
}

// W4(b) — the Stop()-initiated kill is NOT a death: the flow is sealed
// before Stop kills the proc, so the watch stays silent and the latch
// stays clear (no respawn churn for a deliberate shutdown).
func TestServeDeathStopQuiet(t *testing.T) {
	b := newLiveBackend("http://127.0.0.1:1", t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	cmd := exec.Command("sleep", "60")
	b.mu.Lock()
	b.proc = cmd
	b.mu.Unlock()
	b.fl.stop() // what Stop() does BEFORE killing the proc
	b.mu.Lock()
	b.proc = nil
	b.mu.Unlock()

	exit := make(chan error, 1)
	done := make(chan struct{})
	go func() { b.watchServe(cmd, exit); close(done) }()
	exit <- errors.New("signal: killed")
	<-done

	if n := log.textCount("opencode serve died"); n != 0 {
		t.Fatalf("a Stop()-initiated kill must never post the death row, got %d", n)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.serveDied {
		t.Fatal("a deliberate Stop must not latch serveDied")
	}
}

// sessionErr builds a session.error SSE frame for sid carrying msg.
func sessionErr(sid, msg string) ocSSEEvent {
	return ocSSEEvent{Type: "session.error", Properties: json.RawMessage(
		`{"sessionID":"` + sid + `","error":{"name":"API Error","data":{"message":"` + msg + `"}}}`)}
}

// W5(a) — a PRIMARY session.error pops exactly ONE FIFO head: the turn's
// death certificate frees the placeholder the completion pop never sees.
func TestSessionErrorPopsPendingBossOnce(t *testing.T) {
	b := newLiveBackend("http://127.0.0.1:1", t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	b.mu.Lock()
	b.primaryID = "ses-primary"
	b.pendingBoss = []string{"boss-1", "boss-2"}
	b.mu.Unlock()

	if err := b.onEvent(sessionErr("ses-primary", "provider exploded")); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	if len(b.pendingBoss) != 1 || b.pendingBoss[0] != "boss-2" {
		t.Fatalf("one error pops ONE head (the dead turn), got %v", b.pendingBoss)
	}
	b.mu.Unlock()
	if n := log.count(state.EvChatBoss); n != 1 {
		t.Fatalf("the boss-error bubble must mint once, got %d", n)
	}

	// an EMPTY fifo pops nothing (head-guard).
	if err := b.onEvent(sessionErr("ses-primary", "provider exploded")); err != nil {
		t.Fatal(err)
	}
	if err := b.onEvent(sessionErr("ses-primary", "provider exploded")); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	if len(b.pendingBoss) != 0 {
		t.Fatalf("the fifo floors at empty, got %v", b.pendingBoss)
	}
	b.mu.Unlock()

	// a FOREIGN session's error never touches the boss lane's fifo.
	b.mu.Lock()
	b.pendingBoss = []string{"boss-9"}
	b.mu.Unlock()
	if err := b.onEvent(sessionErr("ses-other", "child exploded")); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.pendingBoss) != 1 || b.pendingBoss[0] != "boss-9" {
		t.Fatalf("a non-primary session.error must not pop the boss fifo, got %v", b.pendingBoss)
	}
}

// W5(b) — no double-pop vs the abort path: inside the quiet window the
// "Aborted" session.error is swallowed BEFORE mapping (AbortSessions
// already popped that head), so the fifo moves zero entries.
func TestSessionErrorAbortWindowNoDoublePop(t *testing.T) {
	b := newLiveBackend("http://127.0.0.1:1", t.TempDir(), config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	b.mu.Lock()
	b.primaryID = "ses-primary"
	b.pendingBoss = []string{"boss-4"}
	b.lastAbortAt = nowMs()
	b.mu.Unlock()

	if err := b.onEvent(sessionErr("ses-primary", "Aborted")); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.pendingBoss) != 1 || b.pendingBoss[0] != "boss-4" {
		t.Fatalf("the abort-path pop already happened — the rattle swallows, got %v", b.pendingBoss)
	}
}
