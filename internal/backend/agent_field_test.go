// agent_field_test.go — the plan/build agent tag on the LIVE wire: a
// SendAgent prompt carries {"agent":"plan"} on prompt_async, a plain Send
// ships NO "agent" key (build mode never pays for the field), and a serve
// that 400s the agent field flips the degrade latch — one status note,
// one bare retry, then every future prompt ships bare without another
// word (the promptModelRejected contract, mirrored).
package backend

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

// payloadAgent decodes the prompt body's "agent" string, "" when absent.
func payloadAgent(t *testing.T, body string, wantKey bool) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("prompt body is not JSON: %v — %s", err, body)
	}
	a, present := m["agent"]
	if present != wantKey {
		t.Fatalf("payload agent key: want present=%v, got %v — body %s", wantKey, present, body)
	}
	if !present {
		return ""
	}
	s, _ := a.(string)
	return s
}

// promptPosts returns every recorded prompt_async POST body, oldest first.
func (s *modelStub) promptPosts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, p := range s.posts {
		if strings.HasSuffix(p.path, "/prompt_async") {
			out = append(out, p.body)
		}
	}
	return out
}

// serveAgentRejecting is modelStub.serve with one behavioral edit: any
// prompt_async POST whose body carries an "agent" field gets a 400 ("an
// older serve without the /doc agent field"); everything else 204s.
func (s *modelStub) serveAgentRejecting(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("stub read body: %v", err)
			}
			s.mu.Lock()
			s.posts = append(s.posts, modelPost{r.Method + " " + r.URL.Path, string(b)})
			s.mu.Unlock()
			if r.URL.Path == "/session" {
				w.Write([]byte(`{"id":"ses-made","title":""}`))
				return
			}
			if strings.Contains(string(b), `"agent"`) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":{"message":"invalid request: unknown agent field"}}`))
				return
			}
			w.WriteHeader(http.StatusNoContent) // prompt_async contract
			return
		}
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestAgentFieldRidesPromptAsync: SendAgent(text,"plan") puts
// {"agent":"plan"} on the wire; the plain Send ships no "agent" key at
// all (additive only — the default office payload is yesterday's).
func TestAgentFieldRidesPromptAsync(t *testing.T) {
	stub := &modelStub{}
	srv := stub.serve(t)
	b := liveStubBackend(stub, srv, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)

	if err := b.SendAgent("draft the rollout plan", "plan"); err != nil {
		t.Fatalf("SendAgent: %v", err)
	}
	posts := stub.promptPosts()
	if len(posts) != 1 {
		t.Fatalf("want 1 prompt POST, got %d", len(posts))
	}
	if got := payloadAgent(t, posts[0], true); got != "plan" {
		t.Fatalf("agent field = %q, want %q", got, "plan")
	}

	if err := b.Send("a normal build prompt"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	posts = stub.promptPosts()
	if len(posts) != 2 {
		t.Fatalf("want 2 prompt POSTs, got %d", len(posts))
	}
	payloadAgent(t, posts[1], false) // plain Send ships no "agent" key
}

// TestAgentFieldDegradeLatchesAndRetriesOnce: a 400 on the agent field
// (a) retries the SAME prompt once without it, (b) emits the status note
// exactly once (with the "[theboringfloor] agent-field:" marker the app
// escalades into the transcript), (c) never sends the field again —
// degrade open, and both prompts still land (SendAgent returns nil), and
// (d) AgentDegraded() (the app's badge/warning seam) follows the latch.
func TestAgentFieldDegradeLatchesAndRetriesOnce(t *testing.T) {
	stub := &modelStub{}
	srv := stub.serveAgentRejecting(t)
	b := liveStubBackend(stub, srv, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)

	if b.AgentDegraded() {
		t.Fatal("a fresh backend is not degraded")
	}
	if err := b.SendAgent("step one of the plan", "plan"); err != nil {
		t.Fatalf("SendAgent must degrade open (nil error), got %v", err)
	}
	posts := stub.promptPosts()
	if len(posts) != 2 {
		t.Fatalf("want the rejected POST + ONE bare retry (2 POSTs), got %d", len(posts))
	}
	if got := payloadAgent(t, posts[0], true); got != "plan" {
		t.Fatalf("first POST carries the tag: agent = %q", got)
	}
	payloadAgent(t, posts[1], false) // the retry is bare
	if n := log.textCount("agent field"); n != 1 {
		t.Fatalf("status note must fire exactly once, got %d", n)
	}
	if n := log.textCount("[theboringfloor] agent-field:"); n != 1 {
		t.Fatalf("the marker the app escalades to the transcript rides the note, got %d", n)
	}
	if !b.AgentDegraded() {
		t.Fatal("the latched 400 must expose AgentDegraded() (the app's badge/warning seam)")
	}

	// Latch held: the next SendAgent ships bare immediately — no retry,
	// no second note.
	if err := b.SendAgent("step two of the plan", "plan"); err != nil {
		t.Fatalf("SendAgent after latch: %v", err)
	}
	posts = stub.promptPosts()
	if len(posts) != 3 {
		t.Fatalf("latched send must POST once (3 total), got %d", len(posts))
	}
	payloadAgent(t, posts[2], false)
	if n := log.textCount("agent field"); n != 1 {
		t.Fatalf("note stays once after latched send, got %d", n)
	}
}
