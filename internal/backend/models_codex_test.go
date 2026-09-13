package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func codexMetadataFixture(t *testing.T, script string) (*codexBackend, string) {
	t.Helper()
	dir := t.TempDir()
	bin, log := filepath.Join(dir, "codex-fake"), filepath.Join(dir, "requests")
	t.Setenv("CODEX_MODEL_TEST_LOG", log)
	prelude := `#!/bin/sh
printf '%s\n' "$@" > "$CODEX_MODEL_TEST_LOG.args"
request() { IFS= read -r msg || exit 0; printf '%s\n' "$msg" >> "$CODEX_MODEL_TEST_LOG"; }
`
	if err := os.WriteFile(bin, []byte(prelude+script), 0700); err != nil {
		t.Fatal(err)
	}
	return NewCodex(bin, dir, nil).(*codexBackend), log
}

const codexFixtureInitialize = `request
printf '%s\n' '{"method":"notification","params":{}}' '{"id":90,"result":{}}' '{"id":1,"result":{"userAgent":"test"}}'
request
request
`

func TestCodexCatalogProtocolPaginationAndNativeRefs(t *testing.T) {
	b, log := codexMetadataFixture(t, codexFixtureInitialize+`
printf '%s\n' '{"id":2,"result":{"data":[{"id":"catalog-row","model":"native/token","displayName":"Native model","description":"Account model","isDefault":true},{"id":"secret","model":"hidden-token","hidden":true},{"id":"disabled","model":"disabled-token","disabled":true}],"nextCursor":"opaque cursor / 2"}}'
request
printf '%s\n' '{"method":"status","params":{}}' '{"id":3,"result":{"data":[{"id":"another-row","model":"second-native","displayName":"Second"},{"id":"duplicate","model":"native/token"}],"nextCursor":null}}'
exec sleep 60
`)
	start := time.Now()
	models, err := b.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("metadata server was not stopped after success")
	}
	if len(models) != 3 || models[0].SelectionRef() != "native/token" || models[0].ID != "catalog-row" || !models[0].IsDefault || models[0].Description != "Account model" || !models[1].Disabled || models[2].Ref != "second-native" {
		t.Fatalf("wrong catalog: %#v", models)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 4 {
		t.Fatalf("unexpected requests: %s", raw)
	}
	for i, line := range lines {
		var req struct {
			ID     int            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			t.Fatal(err)
		}
		want := []string{"initialize", "initialized", "model/list", "model/list"}[i]
		if req.Method != want {
			t.Fatalf("request %d: %s", i, line)
		}
		if i == 0 && (req.Params["clientInfo"] == nil || req.Params["capabilities"] == nil) {
			t.Fatalf("incomplete handshake: %s", line)
		}
		if i >= 2 && (req.Params["limit"] != float64(100) || req.Params["includeHidden"] != false || req.ID != i) {
			t.Fatalf("bad catalog request: %s", line)
		}
		if i == 2 && req.Params["cursor"] != nil || i == 3 && req.Params["cursor"] != "opaque cursor / 2" {
			t.Fatalf("cursor not preserved: %s", line)
		}
	}
	args, _ := os.ReadFile(log + ".args")
	if string(args) != "app-server\n--stdio\n" {
		t.Fatalf("metadata invoked inference: %s", args)
	}
	t.Logf("RPC capture: %s", raw)
	t.Logf("Native rows: %+v", models)
}

func TestCodexCatalogFailuresDoNotPublishPartialResults(t *testing.T) {
	cases := []struct{ name, script, want string }{
		{"old-cli", "printf 'unknown command app-server' >&2\nexit 2\n", "unknown command"},
		{"unsupported-method", codexFixtureInitialize + `printf '%s\n' '{"id":2,"error":{"code":-32601,"message":"Method not found"}}'`, "update Codex CLI"},
		{"missing-data", codexFixtureInitialize + `printf '%s\n' '{"id":2,"result":{}}'`, "no data array"},
		{"invalid-model", codexFixtureInitialize + `printf '%s\n' '{"id":2,"result":{"data":[{"model":"bad token"}]}}'`, "invalid model token"},
		{"exit", codexFixtureInitialize + "exit 7\n", "exited before responding"},
		{"malformed", codexFixtureInitialize + "printf 'not json\\n'\n", "invalid JSON"},
		{"page-failure", codexFixtureInitialize + `printf '%s\n' '{"id":2,"result":{"data":[{"model":"first"}],"nextCursor":"next"}}'
request
printf '%s\n' '{"id":3,"error":{"code":123,"message":"page unavailable"}}'
`, "page 2"},
		{"repeat-cursor", codexFixtureInitialize + `printf '%s\n' '{"id":2,"result":{"data":[],"nextCursor":"next"}}'
request
printf '%s\n' '{"id":3,"result":{"data":[],"nextCursor":"next"}}'
`, "repeated"},
		{"huge-line", codexFixtureInitialize + "head -c 1200000 /dev/zero | tr '\\000' 'x'\n", "token too long"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := codexMetadataFixture(t, tc.script)
			models, err := b.ListModels(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.want) || models != nil {
				t.Fatalf("models=%v error=%v; wanted %q", models, err, tc.want)
			}
		})
	}
}

func TestCodexCatalogCancellation(t *testing.T) {
	b, _ := codexMetadataFixture(t, codexFixtureInitialize+"sleep 60 &\nwait\n")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := b.ListModels(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("cancellation leaked inherited stdout or subprocess")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := b.ListModels(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancel: %v", err)
	}
}

func TestCodexMainModelFreshResumeAndConstructionSnapshot(t *testing.T) {
	b, log := codexMetadataFixture(t, `cat >/dev/null
printf '%s\n' '{"type":"thread.started","thread_id":"saved-thread"}' '{"type":"turn.completed"}'
`)
	cfg := &config.Config{}
	cfg.SetModelPreference("codex", "", "native/start")
	cfg.SetModelPreference("opencode", "", "wrong/provider")
	b = NewCodex(b.bin, b.dir, cfg).(*codexBackend)
	cfg.SetModelPreference("codex", "", "changed-outside-adapter")
	if err := b.Start(func(state.Event) {}); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	for i, ref := range []string{"native/start", "next-native"} {
		if i == 1 {
			if err := b.SetModel(context.Background(), state.ModelTarget{}, ref); err != nil {
				t.Fatal(err)
			}
		}
		if err := b.Send("test"); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(log + ".args")
		if !strings.Contains(string(raw), "--model\n"+ref+"\n") {
			t.Fatalf("model absent from actual argv: %s", raw)
		}
		if i == 1 && (!strings.Contains(string(raw), "resume\n") || !strings.Contains(string(raw), "saved-thread\n-\n")) {
			t.Fatalf("resume selection lost: %s", raw)
		}
		t.Logf("argv: %s", raw)
	}
	if cfg.EffectiveModel("codex", "") != "changed-outside-adapter" {
		t.Fatal("SetModel mutated app config")
	}
}

func TestCodexRejectInvalidModelAndUnsafeReset(t *testing.T) {
	b := NewCodex("unused", t.TempDir(), nil).(*codexBackend)
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "valid:model/alias"); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"", " ", "two models", "line\nmodel", "model\x00", "-option", strings.Repeat("x", 513)} {
		if err := b.SetModel(context.Background(), state.ModelTarget{}, ref); err == nil {
			t.Errorf("accepted %q", ref)
		}
		if b.model != "valid:model/alias" {
			t.Fatal("rejected choice changed runtime")
		}
	}
}

// An invalid boss model (e.g. reaching disk with whitespace some way other
// than SetModel's own validateCodexModel gate) must still fail the send —
// self-healing a stale per-agent entry must not be mistaken for "Codex
// stopped validating models". Only the boss model can fail a Codex send for
// model reasons now.
func TestCodexInvalidBossModelStillFailsSend(t *testing.T) {
	fake, log := codexMetadataFixture(t, "exit 77\n")
	cfg := &config.Config{}
	cfg.SetModelPreference("codex", "", "bad model with spaces")
	b := NewCodex(fake.bin, fake.dir, cfg).(*codexBackend)
	if err := b.Start(func(state.Event) {}); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.Send("must fail on invalid boss model"); err == nil {
		t.Fatal("invalid boss model must still fail the send")
	}
	if _, err := os.Stat(log + ".args"); !os.IsNotExist(err) {
		t.Fatalf("started inference with an invalid boss model: %v", err)
	}
}

func TestCodexLegacyOpenCodeModelDoesNotLeak(t *testing.T) {
	cfg := &config.Config{}
	cfg.Boss.Model = "openai/legacy"
	cfg.Backend.BossModel = "openai/backend"
	cfg.AgentModels = map[string]config.ModelRef{"worker": "openai/legacy-worker"}
	b := NewCodex("unused", t.TempDir(), cfg).(*codexBackend)
	if b.model != "" || len(b.agentModels) != 0 {
		t.Fatalf("legacy models leaked: %q %v", b.model, b.agentModels)
	}
}
