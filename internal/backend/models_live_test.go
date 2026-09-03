// models_live_test.go — ListModels on the LIVE backend (the /model
// picker's listing seam): the GET /provider mapping over a stub serve
// (httptest, the exact doJSON call shape — never a real server) and the
// demo backend's fixed fixture:
//
//	(a) CONNECTED providers only: rows carry provider + model id + display
//	    name, sorted within a provider by model id, and the map key fills
//	    a missing model id;
//	(b) DEGRADE-OPEN: an empty connected list maps ALL providers' models
//	    (the picker stays usable — a bad pick fails on the send, never the
//	    mapping);
//	(c) FAILURES: an HTTP 500 and a malformed body both return the error
//	    (the app closes the card and lands the classic hint — never fatal,
//	    never panics);
//	(d) the demo fixture answers the fixed five-model gallery, non-empty
//	    and provider/id-complete.
package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// The app's modelListBackend type-assert binds against *liveBackend —
// pinned here so a signature drift breaks the build, not the picker.
var _ interface {
	ListModels(context.Context) ([]state.ModelInfo, error)
} = (*liveBackend)(nil)

var _ interface {
	ListModels(context.Context) ([]state.ModelInfo, error)
} = (*demoBackend)(nil)

// providerStub — a stub opencode serve answering GET /provider with a
// scripted body (statusCode lets the failure legs break the wire).
type providerStub struct {
	body   string
	status int
}

func (s *providerStub) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/provider" {
			t.Errorf("unexpected call %s %s — ListModels must only fetch /provider", r.Method, r.URL.Path)
		}
		// the directory header + ?directory= query ride every control call
		if r.Header.Get("x-opencode-directory") == "" {
			t.Errorf("the x-opencode-directory header must ride the call")
		}
		if !strings.HasPrefix(r.URL.RawQuery, "directory=") {
			t.Errorf("the ?directory= query must ride the GET, got %q", r.URL.RawQuery)
		}
		if s.status != 0 && s.status != 200 {
			w.WriteHeader(s.status)
		}
		w.Write([]byte(s.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// liveProviderBackend pins a live backend on the stub serve — same
// no-Start shape as models_test.go's liveStubBackend.
func liveProviderBackend(srv *httptest.Server) *liveBackend {
	b := newLiveBackend("", "/tmp", config.Default())
	b.mu.Lock()
	b.baseURL = srv.URL
	b.mu.Unlock()
	return b
}

// (a) the connected mapping: connected-only rows, sorted ids, display
// names, map-key id fallback.
func TestListModelsConnectedMapping(t *testing.T) {
	stub := &providerStub{status: 200, body: `{
		"all": [
			{"id":"anthropic","name":"Anthropic","models":{
				"claude-sonnet-4-5":{"id":"claude-sonnet-4-5","name":"Claude Sonnet 4.5"},
				"claude-haiku-4-5":{"name":"Claude Haiku 4.5"},
				"claude-opus-4":{"id":"claude-opus-4","name":"Claude Opus 4"}
			}},
			{"id":"openai","name":"OpenAI","models":{
				"gpt-5":{"id":"gpt-5","name":"GPT-5"}
			}},
			{"id":"google","name":"Google","models":{
				"gemini-2.5-pro":{"id":"gemini-2.5-pro","name":"Gemini 2.5 Pro"}
			}}
		],
		"connected": ["anthropic","openai"],
		"default": {"anthropic":"claude-sonnet-4-5"}
	}`}
	srv := stub.serve(t)
	b := liveProviderBackend(srv)

	rows, err := b.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("connected providers only — want 4 rows (google drops out), got %d: %+v", len(rows), rows)
	}
	want := []struct{ provider, id, name string }{
		{"anthropic", "claude-haiku-4-5", "Claude Haiku 4.5"}, // the ""-id row took its map key AND sorted first
		{"anthropic", "claude-opus-4", "Claude Opus 4"},
		{"anthropic", "claude-sonnet-4-5", "Claude Sonnet 4.5"},
		{"openai", "gpt-5", "GPT-5"},
	}
	for i, w := range want {
		if rows[i].Provider != w.provider || rows[i].ID != w.id || rows[i].Name != w.name {
			t.Errorf("row %d = %+v, want %s/%s %q", i, rows[i], w.provider, w.id, w.name)
		}
	}
}

// (b) DEGRADE-OPEN: no connected providers → ALL providers' models map
// (the picker stays usable on an unauthenticated serve).
func TestListModelsDegradeOpenWhenNoneConnected(t *testing.T) {
	stub := &providerStub{status: 200, body: `{
		"all": [
			{"id":"anthropic","models":{"claude-opus-4":{"id":"claude-opus-4","name":"Claude Opus 4"}}},
			{"id":"google","models":{"gemini-2.5-pro":{"id":"gemini-2.5-pro"}}}
		],
		"connected": []
	}`}
	srv := stub.serve(t)
	b := liveProviderBackend(srv)

	rows, err := b.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("degrade-open must map ALL providers, got %d: %+v", len(rows), rows)
	}
	if rows[1].Name != "" {
		t.Fatalf("a missing display name rides empty (the picker renders the id), got %q", rows[1].Name)
	}
}

// (c) FAILURES: 500 and garbage both return the error — degrade-open via
// the app's fallback, never a panic.
func TestListModelsFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"http-500", 500, `{"error":{"message":"boom"}}`},
		{"garbage-body", 200, `not json at all`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &providerStub{status: tc.status, body: tc.body}
			srv := stub.serve(t)
			b := liveProviderBackend(srv)
			rows, err := b.ListModels(context.Background())
			if err == nil {
				t.Fatalf("a broken /provider must return an error, got rows %+v", rows)
			}
			if rows != nil {
				t.Fatalf("a failed listing returns NO rows, got %+v", rows)
			}
		})
	}
	// the 500's message text is the serve's own (httpErrorText).
	stub := &providerStub{status: 500, body: `{"error":{"message":"boom"}}`}
	srv := stub.serve(t)
	b := liveProviderBackend(srv)
	if _, err := b.ListModels(context.Background()); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("the wire error message must ride through, got %v", err)
	}
}

// a context deadline is honored through doJSONCtx (the app bounds the hop).
func TestListModelsRespectsContext(t *testing.T) {
	stub := &providerStub{status: 200, body: `{"all":[],"connected":[]}`}
	srv := stub.serve(t)
	b := liveProviderBackend(srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // dead before the call
	if _, err := b.ListModels(ctx); err == nil {
		t.Fatalf("a cancelled context must fail the hop")
	}
}

// (d) the demo fixture: the fixed five, provider/id-complete, same list
// the picker demo + the uishot stub serve.
func TestDemoListModelsFixture(t *testing.T) {
	rows, err := newDemoBackend(config.Default()).ListModels(context.Background())
	if err != nil {
		t.Fatalf("demo ListModels: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("the demo gallery has exactly 5 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Provider == "" || r.ID == "" {
			t.Fatalf("every fixture row needs provider + id: %+v", r)
		}
	}
	// the exact five — marketing's gallery is frozen.
	got, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, ref := range []string{
		"anthropic/claude-sonnet-4-5", "anthropic/claude-opus-4", "anthropic/claude-haiku-4-5",
		"openai/gpt-5", "google/gemini-2.5-pro",
	} {
		parts := strings.Split(ref, "/")
		if !strings.Contains(string(got), parts[0]) || !strings.Contains(string(got), parts[1]) {
			t.Fatalf("the fixture must contain %q: %s", ref, got)
		}
	}
	if rows != nil && &DemoModels()[0] != nil { // sanity: a fresh slice each call (no shared mutation)
		a, bb := DemoModels(), DemoModels()
		a[0].Provider = "mutated"
		if bb[0].Provider == "mutated" {
			t.Fatalf("DemoModels must hand a fresh slice per call")
		}
	}
}
