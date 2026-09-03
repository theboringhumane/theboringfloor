// attachment_backend_integration_test.go proves the app's CURRENT backend
// holder reaches each live transport's real attachment wire after a swap.
package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// attachmentOldBackend is deliberately capable of recording a send: a
// replacement test must prove the old holder did not receive the prompt.
type attachmentOldBackend struct {
	mu        sync.Mutex
	sendCalls int
}

func (b *attachmentOldBackend) Mode() state.Mode                      { return state.ModeLive }
func (b *attachmentOldBackend) Start(func(state.Event)) error         { return nil }
func (b *attachmentOldBackend) Stop() error                           { return nil }
func (b *attachmentOldBackend) AnswerPermission(string, string) error { return nil }
func (b *attachmentOldBackend) AnswerQuestion(string, [][]string) error {
	return nil
}
func (b *attachmentOldBackend) RejectQuestion(string) error            { return nil }
func (b *attachmentOldBackend) MCPServers() ([]state.MCPServer, error) { return nil, nil }
func (b *attachmentOldBackend) ReconnectMCP(string) error              { return nil }
func (b *attachmentOldBackend) Send(string) error {
	b.mu.Lock()
	b.sendCalls++
	b.mu.Unlock()
	return nil
}
func (b *attachmentOldBackend) calls() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sendCalls
}

type attachmentPromptCapture struct {
	mu      sync.Mutex
	prompts []map[string]any
	started chan struct{}
}

func newAttachmentPromptServer(t *testing.T) (*attachmentPromptCapture, *httptest.Server) {
	t.Helper()
	c := &attachmentPromptCapture{started: make(chan struct{}, 1)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" && r.Method == http.MethodGet {
			select {
			case c.started <- struct{}{}:
			default:
			}
			_, _ = w.Write([]byte(`[]`))
			return
		}
		if r.URL.Path == "/session" && r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"ses-attachment","title":""}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/prompt_async") && r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read prompt body: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var prompt map[string]any
			if err := json.Unmarshal(body, &prompt); err != nil {
				t.Errorf("decode prompt body: %v — %s", err, body)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			c.mu.Lock()
			c.prompts = append(c.prompts, prompt)
			c.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	return c, srv
}

func (c *attachmentPromptCapture) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-c.started:
	case <-time.After(3 * time.Second):
		t.Fatal("replacement OpenCode backend did not resolve its session")
	}
}

func (c *attachmentPromptCapture) snapshot() []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]map[string]any(nil), c.prompts...)
}

func attachmentFiles(t *testing.T) (string, string, []state.Attachment) {
	t.Helper()
	dir := t.TempDir()
	md := filepath.Join(dir, "README with spaces.md")
	goFile := filepath.Join(dir, "main file.go")
	if err := os.WriteFile(md, []byte("# attachment proof\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goFile, []byte("package proof\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return md, goFile, []state.Attachment{
		{Name: filepath.Base(md), Mime: "text/markdown", Path: md},
		{Name: filepath.Base(goFile), Mime: "text/x-go", Path: goFile},
	}
}

func assertReferenceOnlyPrompt(t *testing.T, prompt map[string]any, wantTextSuffix string) {
	t.Helper()
	rawParts, ok := prompt["parts"].([]any)
	if !ok {
		t.Fatalf("prompt has no parts array: %#v", prompt)
	}
	for i, raw := range rawParts {
		part, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("part %d is not an object: %#v", i, raw)
		}
		if part["type"] == "file" {
			mime, _ := part["mime"].(string)
			if strings.HasPrefix(mime, "text/") || mime == "text/x-go" {
				t.Fatalf("text attachment leaked as a file/media part at %d (mime=%q): %#v", i, mime, prompt)
			}
			t.Fatalf("unexpected file/media part at %d: %#v", i, prompt)
		}
	}
	if len(rawParts) != 1 {
		t.Fatalf("want exactly one text reference part, got %d: %#v", len(rawParts), prompt)
	}
	part, ok := rawParts[0].(map[string]any)
	text, _ := part["text"].(string)
	if !ok || part["type"] != "text" || !strings.HasSuffix(text, wantTextSuffix) {
		t.Fatalf("reference text part mismatch:\n got %#v\nwant text suffix %q", part, wantTextSuffix)
	}
}

func attachmentRefPrompt(text, md, goFile string) string {
	return text + "\n\n" + strings.Join([]string{
		"[attached file: " + strconv.Quote(md) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(goFile) + "] Read it with your file tools.",
	}, "\n")
}

func attachmentRunCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected app send command, got nil")
	}
	return runMsg(t, m, cmd())
}

func attachmentSwappedOpenCodeModel(t *testing.T) (Model, *attachmentOldBackend, *attachmentPromptCapture) {
	t.Helper()
	capture, srv := newAttachmentPromptServer(t)
	old := &attachmentOldBackend{}
	cfg := config.Default()
	cfg.Backend.Name = config.BackendNameClaude
	m := New(old, cfg)
	m.bootDone = true
	m.sessDir = t.TempDir()
	m.SetEventSink(func(state.Event) {})

	previous := BackendFactory
	BackendFactory = func(name, baseURL, dir string, gotCfg *config.Config) state.Backend {
		if name != config.BackendNameDefault {
			t.Fatalf("factory name = %q, want opencode", name)
		}
		return backend.NewLive(srv.URL, dir, gotCfg)
	}
	t.Cleanup(func() { BackendFactory = previous })
	m = runMsg(t, m, slashMsg{text: "/backend opencode"})
	capture.waitStarted(t)
	return m, old, capture
}

func TestAppCurrentOpenCodeAttachmentDispatchesPathReferences(t *testing.T) {
	m, old, capture := attachmentSwappedOpenCodeModel(t)
	md, goFile, atts := attachmentFiles(t)
	// This is the exact tea.Cmd held by the chat panel. It resolves the
	// current backend at command execution time, after /backend replaced it.
	m = attachmentRunCmd(t, m, currentBackendSend(m.currentBackend, m.plan, "review these source files", atts))

	prompts := capture.snapshot()
	if len(prompts) != 1 {
		t.Fatalf("want one immediate prompt_async post, got %d: %#v", len(prompts), prompts)
	}
	assertReferenceOnlyPrompt(t, prompts[0], attachmentRefPrompt("review these source files", md, goFile))
	if old.calls() != 0 {
		t.Fatalf("replaced backend received %d sends; current OpenCode backend must own dispatch", old.calls())
	}
}

func TestAppCurrentOpenCodeBatchResetRetryKeepsPathReferences(t *testing.T) {
	m, old, capture := attachmentSwappedOpenCodeModel(t)
	md, goFile, atts := attachmentFiles(t)
	m = runMsg(t, m, enqueueMsg{text: "first queued request", atts: atts})
	m = runMsg(t, m, enqueueMsg{text: "second queued request", atts: atts})
	m = runMsg(t, m, queueFlushMsg{})
	if len(m.batchItems) != 2 {
		t.Fatalf("queued batch did not preserve its two retry items: %+v", m.batchItems)
	}
	// Exercise the app's retry command itself: it calls the current live
	// backend's ResetPrimary(true), then re-sends this same composed batch.
	// liveBackend intentionally reports prompt failures as events, so a real
	// HTTP 500 cannot reach queueSendErrMsg; this is the deterministic app
	// reset/retry seam that queueSendErrMsg invokes after a retryable failure.
	m = attachmentRunCmd(t, m, m.resendBatchCmd(m.batchItems))

	prompts := capture.snapshot()
	if len(prompts) != 2 {
		t.Fatalf("want initial batch post plus one reset/retry post, got %d: %#v", len(prompts), prompts)
	}
	batchText := composeBatch([]queueEntry{
		{text: "first queued request", atts: atts},
		{text: "second queued request", atts: atts},
	})
	wantText := batchText + "\n\n" + strings.Join([]string{
		"[attached file: " + strconv.Quote(md) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(goFile) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(md) + "] Read it with your file tools.",
		"[attached file: " + strconv.Quote(goFile) + "] Read it with your file tools.",
	}, "\n")
	for i, prompt := range prompts {
		assertReferenceOnlyPrompt(t, prompt, wantText)
		if i == 1 && m.batchItems[0].text != "first queued request" {
			t.Fatalf("reset retry did not retain the queued batch items: %+v", m.batchItems)
		}
	}
	if old.calls() != 0 {
		t.Fatalf("replaced backend received %d sends during batch retry", old.calls())
	}
}

func TestAppReplacedClaudeAttachmentDispatchesTextOnlyUserContent(t *testing.T) {
	dir := t.TempDir()
	capturePath := filepath.Join(dir, "claude stdin.jsonl")
	stubPath := filepath.Join(dir, "claude-stub.sh")
	stub := fmt.Sprintf("#!/bin/sh\nwhile IFS= read -r line; do\n  printf '%%s\\n' \"$line\" >> %s\ndone\n", strconv.Quote(capturePath))
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}

	old := &attachmentOldBackend{}
	cfg := config.Default()
	cfg.Backend.Name = config.BackendNameDefault
	m := New(old, cfg)
	m.bootDone = true
	m.sessDir = dir
	m.SetEventSink(func(state.Event) {})
	previous := BackendFactory
	BackendFactory = func(name, baseURL, gotDir string, gotCfg *config.Config) state.Backend {
		if name != config.BackendNameClaude {
			t.Fatalf("factory name = %q, want claudecode", name)
		}
		return backend.NewClaude(stubPath, gotDir, gotCfg)
	}
	t.Cleanup(func() { BackendFactory = previous })
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	t.Cleanup(func() { _ = m.backend.Stop() })
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(capturePath)
		if err == nil && strings.Contains(string(raw), `"type":"control_request"`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	md, goFile, atts := attachmentFiles(t)
	// Exercise ordinary Enter's current-backend closure, not a direct backend
	// call: the replacement must own the command when it actually executes.
	m = attachmentRunCmd(t, m, currentBackendSend(m.currentBackend, m.plan, "inspect these paths", atts))

	var user map[string]any
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(capturePath)
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
				var frame map[string]any
				if json.Unmarshal([]byte(line), &frame) == nil && frame["type"] == "user" {
					user = frame
					break
				}
			}
		}
		if user != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if user == nil {
		raw, _ := os.ReadFile(capturePath)
		t.Fatalf("did not capture Claude user frame; stdin was:\n%s", raw)
	}
	message, _ := user["message"].(map[string]any)
	content, _ := message["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("Claude user frame must have exactly one text content part, got %#v", user)
	}
	part, _ := content[0].(map[string]any)
	text, _ := part["text"].(string)
	if part["type"] != "text" || !strings.Contains(text, "[attached file: "+strconv.Quote(md)+"]") || !strings.Contains(text, "[attached file: "+strconv.Quote(goFile)+"]") {
		t.Fatalf("Claude content must be one text part with both quoted references, got %#v", user)
	}
	encoded, _ := json.Marshal(user)
	if strings.Contains(string(encoded), `"type":"file"`) || strings.Contains(string(encoded), `"mime":"text/`) {
		t.Fatalf("Claude text/Go attachment leaked as media content: %s", encoded)
	}
	if old.calls() != 0 {
		t.Fatalf("old backend received %d sends after Claude replacement", old.calls())
	}
}
