// opencode_ledger_test.go — the office-memory EMISSION sites on the live
// backend:
//
//	(a) maybeChildReturned: after the task-done + returned pair the
//	    completion lands in BOTH memory lanes (agentmemory observe +
//	    office-ledger.md), and a REPLAYED idle frame is silent — exactly
//	    once per completion at every layer;
//	(b) QueueItemDone file-only: a dead agentmemory no longer means
//	    amnesia — the synth "local-que-" handle round-trips and the queue
//	    completion lands in the ledger (worker "queue", verdict done);
//	(c) QueueItemDone live: the board mirror still marks the action done
//	    AND the observe lane carries the queue completion exactly once.
package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// waitLedgerFile polls until the office ledger satisfies pred (async
// savers write off the emit path).
func waitLedgerFile(t *testing.T, dir string, pred func(string) bool, what string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	path := filepath.Join(dir, ".opencode", "office-ledger.md")
	for {
		raw, _ := os.ReadFile(path)
		if pred(string(raw)) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s; ledger:\n%s", what, string(raw))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// childReturnStubServe answers exactly what maybeChildReturned touches:
// the child's message listing (one assistant text) — everything else 404s
// (diff fetches degrade silently, house contract).
func childReturnStubServe(t *testing.T, text string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/session/ses-child-1/message" {
			var b strings.Builder
			b.WriteString(`[{"info":{"id":"m-1","sessionID":"ses-child-1","role":"assistant","finish":"stop","time":{"created":110,"completed":120}},"parts":[{"id":"p1","type":"text","text":`)
			enc, _ := json.Marshal(text)
			b.Write(enc)
			b.WriteString(`}]}]`)
			w.Write([]byte(b.String()))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fixture: a live backend wired for ONE known child (no Start — the
// emission path is driven directly), agentmemory refused (file-only).
func childLedgerFixture(t *testing.T, serveURL string) (*liveBackend, *eventLog, string) {
	t.Helper()
	dir := t.TempDir()
	b := newLiveBackend(serveURL, dir, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	t.Cleanup(func() { _ = b.Stop() })
	b.mu.Lock()
	b.baseURL = serveURL
	b.primaryID = "ses-primary"
	b.am = probeAgentmemory("http://127.0.0.1:1")
	b.ctx.employees["ses-child-1"] = state.Employee{
		ID: "ses-child-1", Name: "tekton-1", Role: state.RoleDeveloper, Seat: "dev-1",
		Sprite: state.SpriteAtDesk, Task: "Fix the ledger gate",
	}
	b.mu.Unlock()
	return b, log, dir
}

// (a) the return emits exactly once per completion at every layer.
func TestChildReturnWritesLedgerOnce(t *testing.T) {
	text := "DONE\n- fixed the tokenizer edge case\n\nFILES\n- internal/backend/ledger.go — the writer\n\nVERIFY\n- go test ./internal/backend/ — PASS (exit 0)\n\nPROOF\n- the ledger renders newest-first\n\nISSUES\n- none"
	srv := childReturnStubServe(t, text)
	b, log, dir := childLedgerFixture(t, srv.URL)

	b.maybeChildReturned("ses-child-1")
	b.maybeChildReturned("ses-child-1") // replayed idle: silent everywhere

	if n := len(eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvReturned })); n != 1 {
		t.Fatalf("EvReturned must fire exactly once, got %d", n)
	}
	doneTasks := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvTask && e.Task.Status == state.TaskDone
	})
	if len(doneTasks) != 1 {
		t.Fatalf("task-done must fire exactly once, got %d", len(doneTasks))
	}
	waitLedgerFile(t, dir, func(s string) bool {
		return strings.Contains(s, "- ledgerId: ")
	}, "the child's ledger entry")
	ledgerText := ledgerFile(t, dir)
	if n := strings.Count(ledgerText, "### 20"); n != 1 {
		t.Fatalf("exactly one ledger entry per completion, got %d:\n%s", n, ledgerText)
	}
	for _, want := range []string{
		"### ", " · Fix the ledger gate — tekton-1 (developer) · `done`",
		"- files: internal/backend/ledger.go",
		"- verify: go test ./internal/backend/ — PASS (exit 0)",
		"- proof: the ledger renders newest-first",
	} {
		if !strings.Contains(ledgerText, want) {
			t.Fatalf("ledger entry missing %q:\n%s", want, ledgerText)
		}
	}
	latest, ok := NewLedger(dir).Latest()
	if !ok || latest.WorkerSession != "" /* lossy by design */ || latest.Verdict != "done" {
		t.Fatalf("the file is the digest — verdict done, session field out of it: %+v ok=%v", latest, ok)
	}
	// The entry shape as the BACKEND minted it keeps the worker session +
	// untruncated summary (the agentmemory lane carries them) — the file
	// is the digest; the parse-back drop is documented.
	b.saveLedgerAsync("child:ses-child-1", b.ledgerEntryForReturn("ses-child-1", "Fix the ledger gate",
		state.Employee{ID: "ses-child-1", Name: "tekton-1", Role: state.RoleDeveloper}, text, nowMs()))
	time.Sleep(100 * time.Millisecond)
	if n := strings.Count(ledgerFile(t, dir), "### 20"); n != 1 {
		t.Fatalf("the key latch must make a repeated save a no-op, got %d blocks", n)
	}
}

// (a2) a return with REAL issues records verdict "issues" + the bullets.
func TestChildReturnIssuesVerdict(t *testing.T) {
	text := "DONE\n- ported the gate\n\nISSUES\n- the parser still rejects CRLF files"
	srv := childReturnStubServe(t, text)
	b, _, dir := childLedgerFixture(t, srv.URL)
	b.maybeChildReturned("ses-child-1")
	waitLedgerFile(t, dir, func(s string) bool {
		return strings.Contains(s, " · `issues`")
	}, "an issues verdict entry")
	// summary rides untruncated on the ENTRY (the agentmemory half); the
	// verdict flip proof lives in the file header.
}

// (b) the queue, file-only: synth handle round-trips, completion lands.
func TestQueueItemDoneLedgerFileOnly(t *testing.T) {
	dir := t.TempDir()
	b := newLiveBackend("http://127.0.0.1:1", dir, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	t.Cleanup(func() { _ = b.Stop() })
	b.mu.Lock()
	b.am = probeAgentmemory("http://127.0.0.1:1")
	b.primaryID = "ses-primary"
	b.mu.Unlock()

	id := b.QueueItemStart(2, "answer the quiz line")
	if !strings.HasPrefix(id, "local-que-") {
		t.Fatalf("a dead board lane must mint a local handle (not the forgetful \"\"), got %q", id)
	}
	b.QueueItemDone(id)
	b.QueueItemDone(id) // twice-safe

	waitLedgerFile(t, dir, func(s string) bool {
		return strings.Contains(s, "answer the quiz line")
	}, "the queue completion entry")
	ledgerText := ledgerFile(t, dir)
	if n := strings.Count(ledgerText, "### 20"); n != 1 {
		t.Fatalf("the queue completion must record exactly once, got %d:\n%s", n, ledgerText)
	}
	latest, ok := NewLedger(dir).Latest()
	if !ok {
		t.Fatal("the queue entry must read back")
	}
	if latest.WorkerName != "queue" || latest.WorkerRole != "queue" || latest.Verdict != "done" {
		t.Fatalf("queue entry identity: %+v", latest)
	}
	if n := log.textCount("board action mark done failed"); n != 0 {
		t.Fatalf("the synth handle must never reach MarkAction, got %d notes", n)
	}
}

// (c) the queue, live: board row marked done + ONE observation + one
// ledger entry.
func TestQueueItemDoneLiveMirrorAndLedger(t *testing.T) {
	var creates, updates, observes atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agentmemory/actions":
			w.Write([]byte(`{"actions":[]}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/agentmemory/signals"):
			w.Write([]byte(`{"signals":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agentmemory/actions":
			creates.Add(1)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"action":{"id":"act-42"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agentmemory/actions/update":
			updates.Add(1)
			w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/agentmemory/observe":
			observes.Add(1)
			w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(stub.Close)

	dir := t.TempDir()
	b := newLiveBackend("http://127.0.0.1:1", dir, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	t.Cleanup(func() { _ = b.Stop() })
	b.mu.Lock()
	b.am = probeAgentmemory(stub.URL)
	b.primaryID = "ses-primary"
	b.mu.Unlock()

	id := b.QueueItemStart(1, "flush the backlog")
	if id != "act-42" {
		t.Fatalf("live lane must return the board action id, got %q", id)
	}
	if creates.Load() != 1 {
		t.Fatalf("one board create, got %d", creates.Load())
	}
	b.QueueItemDone(id)
	b.QueueItemDone(id) // twice-safe

	waitLedgerFile(t, dir, func(s string) bool {
		return strings.Contains(s, "queued item completed: flush the backlog")
	}, "the queue's ledger entry")
	if got := observes.Load(); got != 1 {
		t.Fatalf("exactly one observe POST per completion, got %d", got)
	}
	if got := updates.Load(); got != 1 {
		t.Fatalf("the board row must be marked done, got %d update POSTs", got)
	}
	if n := strings.Count(ledgerFile(t, dir), "### 20"); n != 1 {
		t.Fatalf("one ledger entry, got %d", n)
	}
}
