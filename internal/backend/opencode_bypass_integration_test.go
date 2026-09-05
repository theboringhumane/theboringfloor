// opencode_bypass_integration_test.go — a spawned, bypass-enabled OpenCode
// candidate must be usable before the app publishes it as the current backend.
// The serve executable is hermetic, but Start, Send, the HTTP prompt request,
// and the SSE completion/tool flow are the production liveBackend paths.
package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/netwatch"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const bypassAfterReply = "OPEN-CODE-AFTER-BYPASS"

// bypassLiveServe is the smallest OpenCode-shaped server that proves the
// production HTTP+SSE path. It holds SSE connections open and broadcasts a
// completed tool plus an assistant completion after prompt_async.
type bypassLiveServe struct {
	mu          sync.Mutex
	subs        map[chan string]struct{}
	promptPosts []string
	ready       chan struct{}
}

func newBypassLiveServe(t *testing.T) (*bypassLiveServe, *httptest.Server) {
	t.Helper()
	f := &bypassLiveServe{subs: make(map[chan string]struct{}), ready: make(chan struct{}, 4)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			_, _ = w.Write([]byte(`{"id":"ses-primary","title":"theboringfloor office","time":{"created":1,"updated":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-primary/message":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-primary/message/msg-after-bypass":
			_, _ = w.Write([]byte(`{"info":{"id":"msg-after-bypass","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":2,"completed":3}},"parts":[{"id":"text-after-bypass","sessionID":"ses-primary","messageID":"msg-after-bypass","type":"text","text":"` + bypassAfterReply + `"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/session/ses-primary/prompt_async":
			var payload struct {
				Parts []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"parts"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			for _, part := range payload.Parts {
				if part.Type == "text" {
					f.mu.Lock()
					f.promptPosts = append(f.promptPosts, part.Text)
					f.mu.Unlock()
				}
			}
			w.WriteHeader(http.StatusOK)
			f.publish(`{"type":"message.part.updated","properties":{"part":{"id":"tool-after-bypass","sessionID":"ses-primary","messageID":"msg-after-bypass","type":"tool","callID":"call-after-bypass","tool":"bash","state":{"status":"completed","title":"prove bypass replacement","input":{"command":"printf OPEN-CODE-AFTER-BYPASS"},"output":"tool result: OPEN-CODE-AFTER-BYPASS","metadata":{}},"time":{"start":2,"end":3}}}}`)
			f.publish(`{"type":"message.updated","properties":{"info":{"id":"msg-after-bypass","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":2,"completed":3},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}}}}`)
			f.publish(`{"type":"session.idle","properties":{"sessionID":"ses-primary"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/event":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("test server response writer must flush SSE")
			}
			flusher.Flush()
			ch := make(chan string, 8)
			f.mu.Lock()
			f.subs[ch] = struct{}{}
			f.mu.Unlock()
			select {
			case f.ready <- struct{}{}:
			default:
			}
			defer func() {
				f.mu.Lock()
				delete(f.subs, ch)
				f.mu.Unlock()
			}()
			for {
				select {
				case data := <-ch:
					_, _ = w.Write([]byte("data: " + data + "\n\n"))
					flusher.Flush()
				case <-r.Context().Done():
					return
				}
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *bypassLiveServe) publish(data string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for ch := range f.subs {
		select {
		case ch <- data:
		default:
		}
	}
}

func (f *bypassLiveServe) waitedForStreams(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-f.ready:
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for SSE stream %d/%d", i+1, n)
		}
	}
}

func (f *bypassLiveServe) sawPrompt() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, text := range f.promptPosts {
		if strings.Contains(text, bypassAfterReply) {
			return true
		}
	}
	return false
}

func startBypassLiveCandidate(t *testing.T, dir, serveURL string, emit func(state.Event)) *liveBackend {
	t.Helper()
	b := newLiveBackend("", dir, config.Default())
	b.net = netwatch.New((&scriptedProbe{online: true}).probe, 2*time.Millisecond)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(emit); err != nil {
		t.Fatalf("bypass candidate Start: %v", err)
	}
	t.Cleanup(func() { _ = b.Stop() })
	return b
}

// TestOpenCodeBypassReplacementRoundTrip is intentionally backend-owned: it
// does not use app's current-backend holder. That isolates whether a freshly
// started bypass candidate can resolve a primary, keep its own child alive
// after old-backend cleanup, and immediately send/receive through its live
// OpenCode transport before the holder would commit it.
func TestOpenCodeBypassReplacementRoundTrip(t *testing.T) {
	f, srv := newBypassLiveServe(t)
	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "opencode")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf 'opencode server listening on %s\\n' \"$OPENCODE_FIXTURE_URL\"\nwhile :; do sleep 1; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OPENCODE_FIXTURE_URL", srv.URL)
	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")

	old := startBypassLiveCandidate(t, t.TempDir(), srv.URL, (&eventLog{}).emit)
	old.mu.Lock()
	oldProc := old.proc
	old.mu.Unlock()
	if oldProc == nil || oldProc.Process == nil {
		t.Fatal("old backend did not own a serve child")
	}

	log := &eventLog{}
	candidate := startBypassLiveCandidate(t, t.TempDir(), srv.URL, log.emit)
	f.waitedForStreams(t, 2)
	candidate.mu.Lock()
	primaryID, candidateProc := candidate.primaryID, candidate.proc
	candidate.mu.Unlock()
	if primaryID == "" {
		t.Fatal("candidate Start returned without a primary session ID")
	}
	if candidateProc == nil || candidateProc.Process == nil {
		t.Fatal("candidate did not own a serve child")
	}
	if candidateProc.Process.Pid == oldProc.Process.Pid {
		t.Fatalf("replacement reused old child PID %d", candidateProc.Process.Pid)
	}

	// This is the app transition's dangerous ordering: candidate is ready,
	// then the retired backend is cleaned up. Its Stop must touch only its own
	// command, never the candidate's child.
	if err := old.Stop(); err != nil {
		t.Fatalf("old backend cleanup: %v", err)
	}
	if err := candidateProc.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("old cleanup killed replacement child pid %d: %v", candidateProc.Process.Pid, err)
	}

	if err := candidate.Send("reply exactly " + bypassAfterReply); err != nil {
		t.Fatalf("candidate Send: %v", err)
	}
	log.waitFor(t, 3*time.Second, func() bool {
		return f.sawPrompt() && hasBypassReply(log) && hasBypassToolResult(log)
	}, "prompt, tool result, and completed OpenCode reply")
	if err := candidateProc.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("candidate child died after ordinary Send: %v", err)
	}
}

func hasBypassReply(log *eventLog) bool {
	for _, e := range eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvChatBoss && !e.Msg.Pending
	}) {
		if e.Msg.Text == bypassAfterReply {
			return true
		}
	}
	return false
}

func hasBypassToolResult(log *eventLog) bool {
	for _, e := range eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvTool }) {
		if e.CallID == "call-after-bypass" && e.ToolState == "done" && e.ToolOutput == "tool result: "+bypassAfterReply {
			return true
		}
	}
	return false
}
