package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestOpenCodeModelSelectionSnapshotAndPrompt(t *testing.T) {
	cfg := config.Default()
	cfg.Backend.BossModel = "legacy/model"
	cfg.SetModelPreference("opencode", "", "gateway/vendor/model:v1")
	cfg.SetModelPreference("codex", "", "other-backend")
	stub := &modelStub{}
	b := liveStubBackend(stub, stub.serve(t), cfg)
	cfg.SetModelPreference("opencode", "", "mutated/shared")
	check := func(want string) {
		t.Helper()
		if err := b.postPrompt("ses-boss", "proof", nil, ""); err != nil {
			t.Fatal(err)
		}
		body, _ := stub.lastPost("POST /session/ses-boss/prompt_async")
		p, m, _ := payloadModel(t, body, want != "")
		if want != "" && p+"/"+m != want {
			t.Fatalf("model payload: %s", body)
		}
		t.Log(body)
	}
	check("gateway/vendor/model:v1")
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "gateway/next/model"); err != nil {
		t.Fatal(err)
	}
	check("gateway/next/model")
	if cfg.EffectiveModel("opencode", "") != "mutated/shared" || cfg.EffectiveModel("codex", "") != "other-backend" {
		t.Fatal("setter mutated app config")
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{}, ""); err != nil {
		t.Fatal(err)
	}
	check("")
}

func TestOpenCodeModelRejectedDoesNotRetryOrLatch(t *testing.T) {
	stub := &modelStub{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		stub.mu.Lock()
		stub.posts = append(stub.posts, modelPost{r.Method + " " + r.URL.Path, string(body)})
		stub.mu.Unlock()
		if strings.Contains(string(body), `"modelID":"reject"`) {
			http.Error(w, "agent model rejected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	cfg := config.Default()
	cfg.SetModelPreference("opencode", "", "native/reject")
	b := liveStubBackend(stub, srv, cfg)
	if err := b.postPrompt("ses-boss", "fail", nil, "plan"); err == nil || !strings.Contains(err.Error(), "native/reject") {
		t.Fatalf("visible rejection missing: %v", err)
	}
	if got := len(stub.promptPosts()); got != 1 {
		t.Fatalf("rejected model generated %d requests", got)
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "native/accepted"); err != nil {
		t.Fatal(err)
	}
	if err := b.postPrompt("ses-boss", "recover", nil, "plan"); err != nil {
		t.Fatal(err)
	}
	posts := stub.promptPosts()
	if len(posts) != 2 {
		t.Fatalf("requests=%v", posts)
	}
	p, m, _ := payloadModel(t, posts[1], true)
	if p+"/"+m != "native/accepted" || payloadAgent(t, posts[1], true) != "plan" {
		t.Fatal(posts[1])
	}
}

func TestOpenCodeModelAgentFallbackPreservesModel(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference("opencode", "", "gateway/vendor/model")
	stub := &modelStub{}
	b := liveStubBackend(stub, stub.serveAgentRejecting(t), cfg)
	if err := b.postPrompt("ses-boss", "plan", nil, "plan"); err != nil {
		t.Fatal(err)
	}
	posts := stub.promptPosts()
	if len(posts) != 2 {
		t.Fatal(posts)
	}
	for _, body := range posts {
		p, m, _ := payloadModel(t, body, true)
		if p+"/"+m != "gateway/vendor/model" {
			t.Fatal(body)
		}
	}
	payloadAgent(t, posts[1], false)
}

func TestOpenCodeModelNativeAgentsAndConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetModelPreference("opencode", "custom/reviewer", "native/saved")
	cfg.SetModelPreference("codex", "custom/reviewer", "foreign")
	b := newLiveBackend("", dir, cfg)
	if err := b.ApplyAgentModels(b.agentModels); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".opencode", "opencode.json")
	before := `{"plugin":["keep"],"agent":{"custom/reviewer":{"prompt":"Review carefully","tools":{"bash":false},"model":"native/saved"},"other":{"model":"native/other"}}}`
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	var cacheMu sync.Mutex
	cached := "native/saved"
	refreshes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacheMu.Lock()
		defer cacheMu.Unlock()
		switch r.Method + " " + r.URL.Path {
		case "GET /session/status":
			io.WriteString(w, `{}`)
		case "POST /instance/dispose":
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Error(err)
				http.Error(w, "read", 500)
				return
			}
			cached, err = agentModelInConfig(raw, "custom/reviewer")
			if err != nil {
				t.Error(err)
				http.Error(w, "config", 500)
				return
			}
			refreshes++
			io.WriteString(w, `true`)
		case "GET /agent":
			if r.URL.Query().Get("directory") != dir {
				t.Errorf("unexpected directory: %s", r.URL)
			}
			var rows []map[string]any
			json.Unmarshal([]byte(`[{"name":"custom/reviewer","description":"Native review","mode":"subagent"},{"name":"general","mode":"all"},{"name":"plan","mode":"primary"},{"name":"hidden","mode":"subagent","hidden":true},{"name":"disabled","mode":"all","disabled":true}]`), &rows)
			if cached != "" {
				p, m := splitModelRef(cached)
				rows[0]["model"] = map[string]string{"providerID": p, "modelID": m}
			}
			json.NewEncoder(w).Encode(rows)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 404)
		}
	}))
	defer srv.Close()
	b.baseURL = srv.URL
	b.primaryID = "preserved-session"
	rows, err := b.ListModelAgents(context.Background())
	if err != nil || !reflect.DeepEqual(rows, []state.ModelAgentInfo{{Name: "custom/reviewer", Description: "Native review"}, {Name: "general"}}) {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "custom/reviewer"}, "gateway/vendor/model"); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	raw := mustRead(t, path)
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	agent := doc["agent"].(map[string]any)["custom/reviewer"].(map[string]any)
	if agent["model"] != "gateway/vendor/model" || agent["prompt"] != "Review carefully" || agent["tools"].(map[string]any)["bash"] != false || len(doc["plugin"].([]any)) != 1 {
		t.Fatal(raw)
	}
	t.Log(raw)
	cacheMu.Lock()
	if cached != "gateway/vendor/model" || refreshes != 1 {
		t.Errorf("cached model=%s refreshes=%d", cached, refreshes)
	}
	cacheMu.Unlock()
	if b.primaryID != "preserved-session" {
		t.Fatal("refresh replaced session")
	}
	if cfg.EffectiveModel("opencode", "custom/reviewer") != "native/saved" {
		t.Fatal("config mutated")
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "plan"}, "native/no"); err == nil {
		t.Fatal("primary agent was accepted")
	}
	if mustRead(t, path) != raw {
		t.Fatal("failed selection mutated config")
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "custom/reviewer"}, ""); err != nil {
		t.Fatal(err)
	}
	var cleared map[string]any
	json.Unmarshal([]byte(mustRead(t, path)), &cleared)
	if _, exists := cleared["agent"].(map[string]any)["custom/reviewer"].(map[string]any)["model"]; exists {
		t.Fatal("clear retained model")
	}
	if err := os.WriteFile(path, []byte(`{"agent":{"custom/reviewer":false}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "custom/reviewer"}, "native/fail"); err == nil {
		t.Fatal("hand-shaped config accepted")
	}
	if b.agentModels["custom/reviewer"] != "" {
		t.Fatal("failed selection changed runtime")
	}
}

func TestOpenCodeModelCancellationAndLifecycle(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.SetModel(ctx, state.ModelTarget{}, "native/model"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := b.ListModelAgents(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "native/model"); err == nil {
		t.Fatal("unstarted accepted")
	}
	b.fl.stop()
	if _, err := b.ListModels(context.Background()); err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatal(err)
	}
}

func TestOpenCodeModelConcurrentPromptAndSelection(t *testing.T) {
	stub := &modelStub{}
	b := liveStubBackend(stub, stub.serve(t), config.Default())
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := b.SetModel(context.Background(), state.ModelTarget{}, "native/model"); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := b.postPrompt("ses-boss", "race", nil, ""); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestOpenCodeModelRefreshFailureRollsBackOnlyModel(t *testing.T) {
	dir := t.TempDir()
	b := newLiveBackend("", dir, config.Default())
	if err := b.ApplyAgentModels(map[string]string{"explore": "native/original"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".opencode", "opencode.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/agent":
			io.WriteString(w, `[{"name":"explore","mode":"subagent","model":{"providerID":"native","modelID":"original"}}]`)
		case "/session/status":
			io.WriteString(w, `{}`)
		case "/instance/dispose":
			// An unrelated external edit survives the rollback.
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Error(err)
			}
			var doc map[string]any
			json.Unmarshal(raw, &doc)
			doc["theme"] = "external"
			raw, _ = json.Marshal(doc)
			os.WriteFile(path, raw, 0600)
			http.Error(w, "refresh unavailable", http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s", r.URL)
		}
	}))
	defer srv.Close()
	b.baseURL = srv.URL
	err := b.SetModel(context.Background(), state.ModelTarget{Agent: "explore"}, "native/new")
	if err == nil || !strings.Contains(err.Error(), "project model restored") {
		t.Fatalf("failure=%v", err)
	}
	raw := mustRead(t, path)
	model, err := agentModelInConfig([]byte(raw), "explore")
	if err != nil || model != "native/original" || !strings.Contains(raw, `"external"`) {
		t.Fatalf("rollback=%s err=%v", raw, err)
	}
	if _, ok := b.agentModels["explore"]; ok {
		t.Fatal("rejected preference persisted")
	}
}

func TestOpenCodeModelNativeAgentErrorsAndCancellation(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "agents unavailable", status) }))
			defer srv.Close()
			b := newLiveBackend("", t.TempDir(), nil)
			b.baseURL = srv.URL
			if _, err := b.ListModelAgents(context.Background()); err == nil {
				t.Fatal("failed listing accepted")
			}
		})
	}
	entered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer srv.Close()
	b := newLiveBackend("", t.TempDir(), nil)
	b.baseURL = srv.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := b.ListModelAgents(ctx); result <- err }()
	<-entered
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestOpenCodeModelBusyAgentSelectionDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	b := newLiveBackend("", dir, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/agent":
			io.WriteString(w, `[{"name":"explore","mode":"subagent"}]`)
		case "/session/status":
			io.WriteString(w, `{"busy-session":{"type":"busy"}}`)
		default:
			t.Errorf("unexpected request: %s", r.URL)
		}
	}))
	defer srv.Close()
	b.baseURL = srv.URL
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "explore"}, "native/model"); err == nil || !strings.Contains(err.Error(), "idle") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opencode", "opencode.json")); !os.IsNotExist(err) {
		t.Fatalf("busy selection touched config: %v", err)
	}
}

func TestOpenCodeModelSavedAgentsActivateExistingServerAtStart(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.AgentModels = map[string]config.ModelRef{"explore": "legacy/model"}
	cfg.SetModelPreference("opencode", "explore", "gateway/saved/model")
	cfg.SetModelPreference("codex", "explore", "foreign/model")
	path := filepath.Join(dir, ".opencode", "opencode.json")
	var mu sync.Mutex
	cached := "legacy/model"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/agent":
			p, m := splitModelRef(cached)
			json.NewEncoder(w).Encode([]map[string]any{{"name": "explore", "mode": "subagent", "model": map[string]string{"providerID": p, "modelID": m}}})
		case "/session/status":
			io.WriteString(w, `{}`)
		case "/instance/dispose":
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Error(err)
			}
			cached, err = agentModelInConfig(raw, "explore")
			if err != nil {
				t.Error(err)
			}
			io.WriteString(w, `true`)
		case "/session":
			http.Error(w, "stop test boot before watchers", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.Error(w, "unexpected", 404)
		}
	}))
	defer srv.Close()
	b := newLiveBackend(srv.URL, dir, cfg)
	cfg.SetModelPreference("opencode", "explore", "mutated/shared")
	if err := b.Start(func(state.Event) {}); err == nil {
		t.Fatal("fixture must stop before watchers")
	}
	mu.Lock()
	defer mu.Unlock()
	if cached != "gateway/saved/model" {
		t.Fatalf("saved scoped model never activated: %q", cached)
	}
	if ref, err := agentModelInConfig([]byte(mustRead(t, path)), "explore"); err != nil || ref != cached {
		t.Fatalf("config model=%s err=%v", ref, err)
	}
}

func TestOpenCodeModelDeadlineRecovery(t *testing.T) {
	for _, mode := range []string{"restored", "refresh-fails", "verification-fails"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			cfg := config.Default()
			cfg.SetModelPreference("opencode", "explore", "native/original")
			b := newLiveBackend("", dir, cfg)
			b.primaryID = "same-session"
			if err := b.ApplyAgentModels(map[string]string{"explore": "native/original"}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, ".opencode", "opencode.json")
			var mu sync.Mutex
			cached := "native/original"
			refreshes, prompts := 0, 0
			allowRecovery := mode == "restored"
			verifyEntered := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				value, n := cached, refreshes
				allow := allowRecovery
				mu.Unlock()
				switch r.URL.Path {
				case "/agent":
					if n == 1 {
						close(verifyEntered)
						<-r.Context().Done()
						return
					}
					p, m := splitModelRef(value)
					json.NewEncoder(w).Encode([]map[string]any{{"name": "explore", "mode": "subagent", "model": map[string]string{"providerID": p, "modelID": m}}})
				case "/session/status":
					io.WriteString(w, `{}`)
				case "/instance/dispose":
					raw, err := os.ReadFile(path)
					if err != nil {
						t.Error(err)
					}
					// Simulate a concurrent unrelated edit; rollback must preserve it.
					var doc map[string]any
					json.Unmarshal(raw, &doc)
					doc["theme"] = "external"
					raw, _ = json.Marshal(doc)
					if err := os.WriteFile(path, raw, 0600); err != nil {
						t.Error(err)
					}
					disk, err := agentModelInConfig(raw, "explore")
					if err != nil {
						t.Error(err)
					}
					mu.Lock()
					refreshes++
					if n == 0 || allow {
						cached = disk
					}
					mu.Unlock()
					if n > 0 && !allow && mode == "refresh-fails" {
						http.Error(w, "recovery unavailable", 503)
						return
					}
					io.WriteString(w, `true`)
				case "/session/same-session/prompt_async":
					mu.Lock()
					prompts++
					mu.Unlock()
					w.WriteHeader(204)
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.Error(w, "unexpected", 404)
				}
			}))
			defer srv.Close()
			b.baseURL = srv.URL
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- b.SetModel(ctx, state.ModelTarget{Agent: "explore"}, "native/new") }()
			select {
			case <-verifyEntered:
			case err := <-result:
				t.Fatalf("selection ended before activation: %v", err)
			}
			err := <-result
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("selection error=%v", err)
			}
			disk, diskErr := agentModelInConfig([]byte(mustRead(t, path)), "explore")
			if diskErr != nil || disk != "native/original" || !strings.Contains(mustRead(t, path), `"external"`) {
				t.Fatalf("rollback disk=%s err=%v", disk, diskErr)
			}
			mu.Lock()
			native, attempts := cached, refreshes
			mu.Unlock()
			if attempts != 2 {
				t.Fatalf("expected activation plus independent recovery, got %d", attempts)
			}
			if b.agentModels["explore"] != "native/original" || b.primaryID != "same-session" {
				t.Fatal("selection changed preference/session")
			}
			if mode == "restored" {
				if native != "native/original" {
					t.Fatalf("native restore=%s", native)
				}
				if err := b.postPrompt("same-session", "safe after verified restoration", nil, ""); err != nil {
					t.Fatal(err)
				}
			} else {
				if native != "native/new" {
					t.Fatalf("fixture lost failed restore: %s", native)
				}
				if err := b.postPrompt("same-session", "must not send", nil, ""); err == nil {
					t.Fatal("unconfirmed native model allowed inference")
				}
				if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "explore"}, "native/another"); err == nil {
					t.Fatal("uncertain agent change accepted")
				}
				if err := b.SetModel(context.Background(), state.ModelTarget{}, "native/boss"); err != nil {
					t.Fatal(err)
				}
				if err := b.postPrompt("same-session", "boss pick must not release guard", nil, ""); err == nil {
					t.Fatal("boss selection cleared uncertainty")
				}
				mu.Lock()
				if prompts != 0 {
					t.Errorf("blocked prompts reached server: %d", prompts)
				}
				mu.Unlock()
				// Explicit reconciliation must not erase an intervening model edit.
				if _, err := ensureAgentModels(dir, map[string]string{"explore": "native/external"}); err != nil {
					t.Fatal(err)
				}
				before := mustRead(t, path)
				if err := b.ReconcileModelSelection(context.Background()); err == nil {
					t.Fatal("external model edit ignored")
				}
				if mustRead(t, path) != before {
					t.Fatal("reconciliation overwrote external edit")
				}
				if _, err := ensureAgentModels(dir, map[string]string{"explore": "native/original"}); err != nil {
					t.Fatal(err)
				}
				mu.Lock()
				allowRecovery = true
				mu.Unlock()
				if err := b.ReconcileModelSelection(context.Background()); err != nil {
					t.Fatal(err)
				}
				if err := b.postPrompt("same-session", "safe after explicit reconciliation", nil, ""); err != nil {
					t.Fatal(err)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if prompts != 1 || cached != "native/original" {
				t.Fatalf("final native=%s prompts=%d", cached, prompts)
			}
			t.Logf("after deadline: disk=%s native=%s refreshes=%d; after verified recovery: native=%s prompts=%d", disk, native, attempts, cached, prompts)
		})
	}
}
