// agentmemory_ledger_test.go — the memory-lane OBSERVE path: SaveWork
// (the completed-dispatch knowledge the office never used to write) and
// the probe surfacing (MemoryLane / the Start boot note):
//
//	(a) offline: a refused probe degrades to "none" and SaveWork is a
//	    no-op success — NEVER throws;
//	(b) live: SaveWork lands ONE /agentmemory/observe POST carrying
//	    hookType "office_dispatch_done" + the intact FROZEN ledger entry;
//	(c) failure class: a 5xx observe surfaces the helper-style error,
//	    the lane keeps working;
//	(d) a live probe that DIES post-boot fails soft (error, no panic,
//	    bounded) — no-throw means exactly that;
//	(e) the verdict surfacing: liveBackend.MemoryLane + the Start boot
//	    note ("memory: agentmemory OK" vs "memory lane file-only").
package backend

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// observeStub is the agentmemory double for the ledger lane: answers the
// probe candidates (board + mail GETs) and records every POST body by
// path.
type observeStub struct {
	mu      sync.Mutex
	posts   map[string][]string
	observe int // observe status override (0 -> 200)
}

func newObserveStub(t *testing.T, observeStatus int) (*observeStub, *httptest.Server) {
	t.Helper()
	s := &observeStub{posts: map[string][]string{}, observe: observeStatus}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agentmemory/actions":
			w.Write([]byte(`{"actions":[]}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/agentmemory/signals"):
			w.Write([]byte(`{"signals":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/agentmemory/mail":
			w.Write([]byte(`{"mail":[]}`))
		case r.Method == http.MethodPost:
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Errorf("stub read body: %v", readErr)
			}
			s.mu.Lock()
			s.posts[r.URL.Path] = append(s.posts[r.URL.Path], string(body))
			s.mu.Unlock()
			if r.URL.Path == "/agentmemory/observe" && s.observe != 200 && s.observe != 0 {
				w.WriteHeader(s.observe)
				w.Write([]byte(`{"error":"boom"}`))
				return
			}
			w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return s, srv
}

func (s *observeStub) observePayloads(t *testing.T) []map[string]any {
	t.Helper()
	s.mu.Lock()
	raw := append([]string(nil), s.posts["/agentmemory/observe"]...)
	s.mu.Unlock()
	var out []map[string]any
	for _, body := range raw {
		var m map[string]any
		if err := json.Unmarshal([]byte(body), &m); err != nil {
			t.Fatalf("observe payload is not JSON: %v — %s", err, body)
		}
		out = append(out, m)
	}
	return out
}

func ledgerEntryForMemoryTest() LedgerEntry {
	e := ledgerEntryFixture()
	e.Verdict = "issues"
	e.Issues = []string{"the parser still rejects CRLF files"}
	return e
}

// (a) offline: the refused probe degrades and SaveWork no-ops clean.
func TestSaveWorkOfflineNoThrow(t *testing.T) {
	h := probeAgentmemory("http://127.0.0.1:1") // instant refusal
	if h.kind != "none" {
		t.Fatalf("a refused probe must degrade to none, got %q", h.kind)
	}
	if err := h.SaveWork(ledgerEntryForMemoryTest()); err != nil {
		t.Fatalf("none-mode SaveWork must be a no-op success, got %v", err)
	}
	if lane := h.memoryLaneText(); lane != "file-only" {
		t.Fatalf("offline probe's lane text: %q", lane)
	}
}

// (b) live: the observation envelope (hookType/sessionId/project/
// timestamp) and the intact FROZEN entry under data.ledger.
func TestSaveWorkObservePayload(t *testing.T) {
	stub, srv := newObserveStub(t, 200)
	h := probeAgentmemory(srv.URL)
	if h.kind != "actions" {
		t.Fatalf("stub probe must arm the actions lane, got %q", h.kind)
	}
	entry := ledgerEntryForMemoryTest()
	if err := h.SaveWork(entry); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
	payloads := stub.observePayloads(t)
	if len(payloads) != 1 {
		t.Fatalf("want exactly one /observe POST, got %d", len(payloads))
	}
	p := payloads[0]
	if p["hookType"] != "office_dispatch_done" {
		t.Fatalf("hookType: %v", p["hookType"])
	}
	if p["project"] != "proj" || p["sessionId"] != "ses-1" {
		t.Fatalf("envelope fields off: %v", p)
	}
	if p["timestamp"] != "2026-08-25T12:00:00Z" {
		t.Fatalf("completedAt must render UTC RFC3339, got %v", p["timestamp"])
	}
	data, ok := p["data"].(map[string]any)
	if !ok {
		t.Fatalf("payload must carry data: %v", p)
	}
	if data["ledgerId"] != entry.LedgerID || data["dispatchTitle"] != entry.DispatchTitle ||
		data["workerName"] != "tekton-3" || data["verdict"] != "issues" {
		t.Fatalf("data digest fields off: %v", data)
	}
	ledger, ok := data["ledger"].(map[string]any)
	if !ok {
		t.Fatalf("the FROZEN entry must ride data.ledger intact: %v", data)
	}
	if ledger["proofOneLiner"] != entry.ProofOneLiner || ledger["verifyDigest"] != entry.VerifyDigest {
		t.Fatalf("the intact entry must keep its digest fields: %v", ledger)
	}
	if issues, ok := data["issues"].([]any); !ok || len(issues) != 1 || issues[0] != "the parser still rejects CRLF files" {
		t.Fatalf("issues must ride as an array: %v", data["issues"])
	}
}

// (c) the failure class: a 500 observe surfaces the helper-style error.
func TestSaveWorkServerFailure(t *testing.T) {
	_, srv := newObserveStub(t, 500)
	h := probeAgentmemory(srv.URL)
	if h.kind != "actions" {
		t.Fatalf("stub probe must arm the actions lane")
	}
	err := h.SaveWork(ledgerEntryForMemoryTest())
	if err == nil || !strings.Contains(err.Error(), "POST /agentmemory/observe failed") {
		t.Fatalf("want the helper-style error, got %v", err)
	}
}

// (d) the lane dies post-probe: SaveWork fails soft (error + no panic +
// bounded) — no-throw means exactly that.
func TestSaveWorkLaneDiesAfterProbe(t *testing.T) {
	_, srv := newObserveStub(t, 200)
	h := probeAgentmemory(srv.URL)
	if h.kind != "actions" {
		t.Fatalf("stub probe must arm the actions lane")
	}
	srv.Close() // the server dies NOW
	done := make(chan error, 1)
	go func() { done <- h.SaveWork(ledgerEntryForMemoryTest()) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a dead lane must fail soft with an error, not success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SaveWork must stay bounded (the 2s client), never hang")
	}
}

// (e) the verdict surfacing: MemoryLane seam + the Start boot note.
func TestMemoryLaneSurfaced(t *testing.T) {
	// Live lane: MemoryLane reads OK.
	stub, srv := newObserveStub(t, 200)
	_ = stub
	b := newLiveBackend(srv.URL, t.TempDir(), nil)
	b.mu.Lock()
	b.am = probeAgentmemory(srv.URL)
	b.mu.Unlock()
	if got := b.MemoryLane(); got != "OK" {
		t.Fatalf("live lane must read OK, got %q", got)
	}
	// No probe at all (pre-Start): file-only.
	b2 := newLiveBackend("http://127.0.0.1:1", t.TempDir(), nil)
	if got := b2.MemoryLane(); got != "file-only" {
		t.Fatalf("pre-Start backend must read file-only, got %q", got)
	}
}

// (e2) the Start boot note: the silent degrade is over — the status line
// carries the verdict exactly once.
func TestStartBootNoteMemoryLane(t *testing.T) {
	b, log := startLiveForTest(t) // agentmemory pointed at a refused port
	_ = b
	log.waitFor(t, 2*time.Second, func() bool {
		return log.textCount("[theboringoffice] memory: memory lane file-only") == 1
	}, "the file-only memory boot note")
	if n := log.textCount("[theboringoffice] memory:"); n != 1 {
		t.Fatalf("exactly one memory boot note, got %d", n)
	}
}
