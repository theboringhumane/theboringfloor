package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const claudeModelFixtureCatalog = `{"models":[{"value":"default","resolvedModel":"claude-default-test","displayName":"Default","description":"Native default"},{"value":"sonnet","resolvedModel":"claude-sonnet-test","displayName":"Sonnet","description":"Native sonnet"},{"value":"namespace/custom@preview","displayName":"Custom"},{"value":"opus","displayName":"Opus","disabled":true,"description":"Disabled by account"}],"unavailable_models":[{"value":"unavailable","displayName":"Unavailable"}],"agents":[{"name":"Explore","description":"Inspect files"},{"name":"researcher","description":"Project researcher"},{"name":"fork","description":"Inherits parent"}],"hooks_applied":true}`

// This child speaks protocol only; no installed CLI or inference is involved.
func TestClaudeModelFixtureProcess(t *testing.T) {
	if !strings.HasPrefix(os.Getenv("THEFLOOR_CLAUDE_STUB_SCENARIO"), "models-") {
		return
	}
	log := func(path string, v any) {
		f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			os.Exit(10)
		}
		_ = json.NewEncoder(f).Encode(v)
		_ = f.Close()
	}
	log(os.Getenv("THEFLOOR_CLAUDE_STUB_ARGV"), os.Args)
	mode := strings.TrimPrefix(os.Getenv("THEFLOOR_CLAUDE_STUB_SCENARIO"), "models-")
	ack := func(id, sub string, result any) {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"type": "control_response", "response": map[string]any{"subtype": sub, "request_id": id, "response": result}})
	}
	reject := func(id, msg string) {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"type": "control_response", "response": map[string]any{"subtype": "error", "request_id": id, "error": msg}})
	}
	sendHook := func(id string, req map[string]any) {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"type": "control_request", "request_id": id, "request": req})
	}
	var catalog any
	_ = json.Unmarshal([]byte(claudeModelFixtureCatalog), &catalog)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var frame map[string]any
		_ = json.Unmarshal(scanner.Bytes(), &frame)
		log(os.Getenv("THEFLOOR_CLAUDE_STUB_CAPTURE"), frame)
		if frame["type"] == "user" {
			fmt.Println(`{"type":"system","subtype":"init","session_id":"model-test-session","model":"claude-default-test"}`)
			fmt.Println(`{"type":"result","subtype":"success","session_id":"model-test-session"}`)
			continue
		}
		req, _ := frame["request"].(map[string]any)
		id, _ := frame["request_id"].(string)
		switch req["subtype"] {
		case "initialize":
			if mode == "no-init" {
				continue
			}
			if mode == "reject-init" {
				reject(id, "initialization denied")
				continue
			}
			if mode == "no-hooks" {
				catalog.(map[string]any)["hooks_applied"] = false
			}
			ack(id, "success", catalog)
			if mode == "hook" {
				fmt.Println(`{"type":"control_request","request_id":"fixture-hook","request":{"subtype":"hook_callback","callback_id":"office-agent-model-default","input":{"hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"subagent_type":"Explore","description":"Inspect","prompt":"Keep this exact prompt","tools":["Read"],"run_in_background":true}}}}`)
			}
			// hook-drift: simulates the real CLI's hook_callback payload
			// diverging from every field this code assumes (wrong
			// callback_id, wrong hook_event_name, missing input, a
			// non-map tool_input, and a fully renamed shape) plus one
			// recognized agent with no saved model. Every one of these
			// MUST still receive a control_response — that is the exact
			// regression the claude.go:649 fix closes.
			if mode == "hook-drift" {
				sendHook("drift-happy", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "Explore", "prompt": "drift happy path"}}})
				sendHook("drift-bad-callback", map[string]any{"subtype": "hook_callback", "callback_id": "unexpected-callback-id", "input": map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "Explore", "prompt": "drift bad callback"}}})
				sendHook("drift-bad-event", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"hook_event_name": "PostToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "Explore", "prompt": "drift bad event"}}})
				sendHook("drift-missing-input", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent"}})
				sendHook("drift-input-not-map", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": "not-an-object"}})
				sendHook("drift-unknown-shape", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"event": "PreToolUse", "tool": "Agent", "payload": map[string]any{"agent_type": "Explore"}}})
				sendHook("drift-no-saved-model", map[string]any{"subtype": "hook_callback", "callback_id": "office-agent-model-default", "input": map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "researcher", "prompt": "drift no saved model"}}})
			}
		case "list_models":
			if mode == "hold-catalog" {
				time.Sleep(100 * time.Millisecond)
			}
			if mode == "unsupported" {
				reject(id, "Unknown control request: list_models")
				continue
			}
			if mode == "exit" {
				os.Exit(3)
			}
			if mode == "catalog-error" {
				reject(id, "catalog access denied")
				continue
			}
			ack(id, "success", catalog)
		case "set_model":
			if req["model"] == "reject" {
				reject(id, "unknown model reject")
				continue
			}
			if req["model"] == "slow" {
				time.Sleep(90 * time.Millisecond)
			}
			ack("unrelated-id", "success", map[string]any{})
			ack(id, "success", map[string]any{})
		}
	}
	os.Exit(0)
}

func claudeModelFixture(t *testing.T, mode string, cfg *config.Config, bypass ...bool) (*liveClaudeBackend, *claudeEventLog, string, string) {
	t.Helper()
	dir := t.TempDir()
	capture := filepath.Join(dir, "stdin.jsonl")
	argv := filepath.Join(dir, "argv.jsonl")
	t.Setenv("THEFLOOR_CLAUDE_STUB_SCENARIO", "models-"+mode)
	t.Setenv("THEFLOOR_CLAUDE_STUB_CAPTURE", capture)
	t.Setenv("THEFLOOR_CLAUDE_STUB_ARGV", argv)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	stub := claudeStubScript(t, "exec '"+strings.ReplaceAll(exe, "'", "'\\''")+"' -test.run=^TestClaudeModelFixtureProcess$ -- \"$@\"\n")
	b := newClaudeBackend(stub, dir, cfg)
	if len(bypass) > 0 {
		if err := b.SetBypassPermissions(bypass[0]); err != nil {
			t.Fatal(err)
		}
	}
	log := &claudeEventLog{}
	t.Cleanup(func() { _ = b.Stop() })
	if err = b.Start(log.emit); err != nil {
		t.Fatal(err)
	}
	return b, log, capture, argv
}

func claudeModelFrames(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, _ := os.ReadFile(path)
	var frames []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var f map[string]any
		if json.Unmarshal([]byte(line), &f) == nil {
			frames = append(frames, f)
		}
	}
	return frames
}

func TestClaudeModelCatalogNativeRefsAndAgents(t *testing.T) {
	b, log, _, _ := claudeModelFixture(t, "", nil)
	models, err := b.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 5 || models[0].SelectionRef() != "default" || models[0].ID != "claude-default-test" || !models[0].IsDefault || models[2].SelectionRef() != "namespace/custom@preview" || !models[3].Disabled || !models[4].Disabled {
		t.Fatalf("catalog: %+v", models)
	}
	agents, err := b.ListModelAgents(context.Background())
	if err != nil || len(agents) != 3 || agents[0].Name != "Explore" {
		t.Fatalf("agents=%+v err=%v", agents, err)
	}
	agents[0].Name = "mutated"
	again, _ := b.ListModelAgents(context.Background())
	if again[0].Name != "Explore" {
		t.Fatal("catalog slice was shared")
	}
	filtered, err := b.ListModelsForTarget(context.Background(), state.ModelTarget{Agent: "Explore"})
	if err != nil || len(filtered) != 2 || filtered[0].Ref != "sonnet" || filtered[1].Ref != "opus" {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatBoss || e.Kind == state.EvChatUser || e.Kind == state.EvHire && e.Employee.Role != state.RoleManager && e.Employee.Role != state.RoleHR {
			t.Fatalf("control became conversation event: %+v", e)
		}
	}
}

func TestClaudeModelCatalogFallbackAndErrors(t *testing.T) {
	for _, mode := range []string{"unsupported", "catalog-error", "reject-init", "exit", "no-init"} {
		t.Run(mode, func(t *testing.T) {
			b, _, _, _ := claudeModelFixture(t, mode, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			models, err := b.ListModels(ctx)
			if mode == "unsupported" {
				if err != nil || len(models) != 5 {
					t.Fatalf("fallback: %v %+v", err, models)
				}
			} else if err == nil {
				t.Fatal("expected surfaced error")
			}
		})
	}
}

func TestClaudeModelSetAcknowledgmentRejectionAndCancellation(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "", "sonnet")
	b, _, capture, _ := claudeModelFixture(t, "", cfg)
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "namespace/custom@preview"); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"reject", "opus", "unavailable"} {
		if err := b.SetModel(context.Background(), state.ModelTarget{}, ref); err == nil {
			t.Fatalf("accepted %s", ref)
		}
	}
	b.mu.Lock()
	ref := b.modelRef
	b.mu.Unlock()
	if ref != "namespace/custom@preview" {
		t.Fatalf("rejected model persisted: %q", ref)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	if err := b.SetModel(ctx, state.ModelTarget{}, "slow"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout=%v", err)
	}
	b.mu.Lock()
	ref = b.modelRef
	uncertain := b.modelUncertain
	b.mu.Unlock()
	if ref != "namespace/custom@preview" || !uncertain {
		t.Fatal("timeout committed or failed to mark uncertain native state")
	}
	if err := b.Send("after timed out switch"); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "user write after model reconciliation", time.Second, func() bool {
		for _, f := range claudeModelFrames(t, capture) {
			if f["type"] == "user" {
				return true
			}
		}
		return false
	})
	var lastModel string
	for _, f := range claudeModelFrames(t, capture) {
		r, _ := f["request"].(map[string]any)
		if r["subtype"] == "set_model" {
			lastModel, _ = r["model"].(string)
		}
		if f["type"] == "user" && lastModel != "namespace/custom@preview" {
			t.Fatalf("send overtook reconciliation: %q", lastModel)
		}
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{}, ""); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	ref = b.modelRef
	b.mu.Unlock()
	if ref != "" {
		t.Fatal("native default was not retained")
	}
}

func TestClaudeModelControlProcessCorrelation(t *testing.T) {
	b := newClaudeBackend("", "", nil)
	oldProc := &exec.Cmd{}
	newProc := &exec.Cmd{}
	pending := &claudeControlPending{proc: newProc, done: make(chan claudeControlReply, 1)}
	b.controls["same-id"] = pending
	ack := claudeControlAck{Subtype: "success", RequestID: "same-id", Response: json.RawMessage(`{}`)}
	b.mu.Lock()
	b.handleControlResponseLocked(oldProc, ack)
	b.mu.Unlock()
	select {
	case <-pending.done:
		t.Fatal("old process acknowledged new request")
	default:
	}
	b.mu.Lock()
	b.failControlsLocked(oldProc, errors.New("old exit"))
	b.mu.Unlock()
	select {
	case <-pending.done:
		t.Fatal("old exit failed new request")
	default:
	}
	b.mu.Lock()
	b.failControlsLocked(newProc, errors.New("replacement"))
	b.mu.Unlock()
	if reply := <-pending.done; reply.err == nil {
		t.Fatal("replacement did not fail pending control")
	}
}

func TestClaudeModelStartupResumeAndOfficePreserveSelection(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "", "namespace/custom@preview")
	b, _, _, argv := claudeModelFixture(t, "", cfg)
	cfg.SetModelPreference(config.BackendNameClaude, "", "external-mutation")
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "sonnet"); err != nil {
		t.Fatal(err)
	}
	if err := b.SwapPrimary("resume-model-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewOffice(); err != nil {
		t.Fatal(err)
	}
	if err := b.Send("establish session"); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "session pin", time.Second, func() bool { return b.PrimaryID() == "model-test-session" })
	b.mu.Lock()
	proc := b.proc
	b.mu.Unlock()
	if err := proc.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "death watch", time.Second, func() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.died })
	if err := b.Send("recover session"); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "four launches", time.Second, func() bool {
		data, _ := os.ReadFile(argv)
		return len(strings.Split(strings.TrimSpace(string(data)), "\n")) == 4
	})
	data, _ := os.ReadFile(argv)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i, line := range lines {
		var args []string
		_ = json.Unmarshal([]byte(line), &args)
		joined := strings.Join(args, " ")
		want := "--model sonnet"
		if i == 0 {
			want = "--model namespace/custom@preview"
		}
		if !strings.Contains(joined, want) {
			t.Fatalf("launch %d lacks %q: %s", i, want, joined)
		}
		if i == 1 && !strings.Contains(joined, "--resume resume-model-test") {
			t.Fatal("explicit resume lost")
		}
		if i == 3 && !strings.Contains(joined, "--resume model-test-session") {
			t.Fatal("death resume lost")
		}
	}
}

func TestClaudeAgentModelDefaultsPreserveDispatchAndDefinitions(t *testing.T) {
	b, _, _, _ := claudeModelFixture(t, "", nil)
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, "sonnet"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ agent, ref string }{{"missing", "sonnet"}, {"fork", "sonnet"}, {"Explore", "namespace/custom@preview"}, {"Explore", "opus"}} {
		if err := b.SetModel(context.Background(), state.ModelTarget{Agent: tc.agent}, tc.ref); err == nil {
			t.Fatalf("accepted unsupported %+v", tc)
		}
	}
	for _, tool := range []string{"Agent", "Task"} {
		for _, explicit := range []bool{false, true} {
			input := map[string]any{"subagent_type": "Explore", "prompt": "Original prompt", "description": "Original description", "tools": []any{"Read", "Grep"}, "run_in_background": true, "custom": map[string]any{"retained": true}}
			if explicit {
				input["model"] = "haiku"
			}
			before, _ := json.Marshal(input)
			req := claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{"hook_event_name": "PreToolUse", "tool_name": tool, "tool_input": input}}
			b.mu.Lock()
			output := b.agentModelHookResultLocked(req)
			b.mu.Unlock()
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("original input mutated")
			}
			if explicit {
				if len(output) != 0 {
					t.Fatal("explicit dispatch overridden")
				}
				continue
			}
			hook := output["hookSpecificOutput"].(map[string]any)
			updated := hook["updatedInput"].(map[string]any)
			if updated["model"] != "sonnet" {
				t.Fatal("default missing")
			}
			delete(updated, "model")
			if !reflect.DeepEqual(updated, input) {
				t.Fatalf("dispatch fields changed: %+v", updated)
			}
			if _, ok := hook["permissionDecision"]; ok {
				t.Fatal("model hook granted permission")
			}
		}
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, ""); err != nil {
		t.Fatal(err)
	}
	req := claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "Explore"}}}
	b.mu.Lock()
	output := b.agentModelHookResultLocked(req)
	b.mu.Unlock()
	if len(output) != 0 {
		t.Fatal("clearing should restore original native definition")
	}
}

func TestClaudeAgentModelHookWireAndBypass(t *testing.T) {
	for _, bypass := range []bool{false, true} {
		t.Run(fmt.Sprint(bypass), func(t *testing.T) {
			// The same initialize hook is independent of can_use_tool and permission mode.
			cfg := config.Default()
			cfg.SetModelPreference(config.BackendNameClaude, "Explore", "sonnet")
			b, log, capture, _ := claudeModelFixture(t, "hook", cfg, bypass)
			// Wait for native initialization separately from callback latency:
			// subprocess startup under the race detector is not hook handling.
			if _, _, err := b.initializedCatalog(context.Background()); err != nil {
				t.Fatal(err)
			}
			// The callback is tested without ANY permission request on the wire.
			claudeWait(t, "hook response", time.Second, func() bool {
				for _, f := range claudeModelFrames(t, capture) {
					r, _ := f["response"].(map[string]any)
					if r["request_id"] == "fixture-hook" {
						return true
					}
				}
				return false
			})
			for _, f := range claudeModelFrames(t, capture) {
				r, _ := f["response"].(map[string]any)
				if r["request_id"] != "fixture-hook" {
					continue
				}
				out := r["response"].(map[string]any)
				updated := out["hookSpecificOutput"].(map[string]any)["updatedInput"].(map[string]any)
				if updated["model"] != "sonnet" || updated["prompt"] != "Keep this exact prompt" {
					t.Fatalf("hook payload=%+v", updated)
				}
			}
			for _, e := range log.snapshot() {
				if e.Kind == state.EvPermission || e.Kind == state.EvQuestion {
					t.Fatal("hook became permission/dialog")
				}
			}
			_ = b
		})
	}
}

// TestClaudeAgentModelHookDriftedPayloadsStillAck drives real hook_callback
// control_request frames through the ACTUAL readLoop routing in claude.go —
// not just the isolated agentModelHookResultLocked function — with every
// drifted shape the reviewer's finding named: a wrong callback_id, a wrong
// hook_event_name, a missing tool_input, a non-map tool_input, a fully
// renamed shape, and a recognized agent with no saved model. Before the
// claude.go:649 fix, "drift-bad-callback" (the exact scenario the reviewer
// described — CallbackID drift) received NO control_response at all: the
// wedge this test exists to catch. Every frame here MUST receive a
// well-formed control_response; only "drift-happy" may carry a model
// injection, and its payload must be byte-for-byte what today's code
// already produces (requirement 4a: the happy path is unchanged).
func TestClaudeAgentModelHookDriftedPayloadsStillAck(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "Explore", "sonnet")
	b, _, capture, _ := claudeModelFixture(t, "hook-drift", cfg)
	if _, _, err := b.initializedCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	ids := []string{
		"drift-happy", "drift-bad-callback", "drift-bad-event",
		"drift-missing-input", "drift-input-not-map", "drift-unknown-shape",
		"drift-no-saved-model",
	}
	responses := map[string]map[string]any{}
	claudeWait(t, "every drifted hook_callback frame acknowledged", time.Second, func() bool {
		for _, f := range claudeModelFrames(t, capture) {
			r, _ := f["response"].(map[string]any)
			id, _ := r["request_id"].(string)
			for _, want := range ids {
				if id == want {
					responses[id] = r
				}
			}
		}
		return len(responses) == len(ids)
	})
	for _, id := range ids {
		resp, ok := responses[id]
		if !ok {
			t.Fatalf("%q: no control_response was ever written — this is the wedge the fix prevents", id)
		}
		if resp["subtype"] != "success" {
			t.Fatalf("%q: not a well-formed success ack: %+v", id, resp)
		}
		body, _ := resp["response"].(map[string]any)
		if id == "drift-happy" {
			hook, ok := body["hookSpecificOutput"].(map[string]any)
			if !ok {
				t.Fatalf("happy path: no hookSpecificOutput: %+v", body)
			}
			updated, _ := hook["updatedInput"].(map[string]any)
			if updated["model"] != "sonnet" || updated["prompt"] != "drift happy path" || hook["hookEventName"] != "PreToolUse" {
				t.Fatalf("happy path payload changed: %+v", body)
			}
			continue
		}
		if len(body) != 0 {
			t.Fatalf("%q: expected a no-op ack ({}), got %+v", id, body)
		}
	}
}

func TestClaudeAgentModelRequiresAcknowledgedHook(t *testing.T) {
	b, _, _, _ := claudeModelFixture(t, "no-hooks", nil)
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, "sonnet"); err == nil {
		t.Fatal("accepted default without native hook acknowledgment")
	}
}

func TestClaudeAgentModelSavedUnsupportedDiagnostic(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "Explore", "namespace/unsupported")
	_, log, _, _ := claudeModelFixture(t, "", cfg)
	claudeWait(t, "unsupported saved selection diagnostic", time.Second, func() bool { return log.hasStatusContaining("was not applied") })
}

func TestClaudeModelSwitchOrdersSendAndHonorsWaitingCancellation(t *testing.T) {
	b, _, capture, _ := claudeModelFixture(t, "", nil)
	selected := make(chan error, 1)
	go func() { selected <- b.SetModel(context.Background(), state.ModelTarget{}, "slow") }()
	claudeWait(t, "pending set_model", time.Second, func() bool {
		for _, f := range claudeModelFrames(t, capture) {
			r, _ := f["request"].(map[string]any)
			if r["subtype"] == "set_model" {
				return true
			}
		}
		return false
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := b.SetModel(ctx, state.ModelTarget{}, "sonnet"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked switch ignored cancellation: %v", err)
	}
	sent := make(chan error, 1)
	go func() { sent <- b.Send("after acknowledged switch") }()
	select {
	case err := <-sent:
		t.Fatalf("Send overtook model acknowledgment: %v", err)
	case <-time.After(15 * time.Millisecond):
	}
	if err := <-selected; err != nil {
		t.Fatal(err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
}

func TestClaudeAgentModelCatalogContextVariantAliases(t *testing.T) {
	input := []state.ModelInfo{{Ref: "default", IsDefault: true}, {Ref: "opus[1m]", ID: "claude-opus-native[1m]", Name: "Opus (1M context)", Description: "Native catalog row"}, {Ref: "sonnet", Name: "Sonnet"}, {Ref: "haiku", Name: "Haiku"}, {Ref: "claude-full-id", Name: "Full ID"}}
	got := claudeAgentModelInfos(input)
	if len(got) != 3 || got[0].Ref != "opus" || got[1].Ref != "sonnet" || got[2].Ref != "haiku" || !strings.Contains(got[0].Name, "agent alias") || !strings.Contains(got[0].Description, "modifier is not used") {
		t.Fatalf("agent catalog: %+v", got)
	}
	if input[1].Ref != "opus[1m]" {
		t.Fatal("main catalog token mutated")
	}
	input = append(input, state.ModelInfo{Ref: "opus", Disabled: true, Description: "Account restriction"})
	got = claudeAgentModelInfos(input)
	if len(got) != 3 || !got[0].Disabled || !strings.Contains(got[0].Description, "Account restriction") {
		t.Fatalf("duplicate discarded restriction: %+v", got)
	}
	b, _, _, _ := claudeModelFixture(t, "", nil)
	if _, _, err := b.initializedCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	b.initialization.catalog.Models[3].Value = "opus[1m]"
	b.initialization.catalog.Models[3].Disabled = false
	b.mu.Unlock()
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, "opus"); err != nil {
		t.Fatal(err)
	}
	request := claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": map[string]any{"subagent_type": "Explore"}}}
	b.mu.Lock()
	result := b.agentModelHookResultLocked(request)
	b.mu.Unlock()
	if result["hookSpecificOutput"].(map[string]any)["updatedInput"].(map[string]any)["model"] != "opus" {
		t.Fatalf("dispatch lost native alias: %+v", result)
	}
	b.mu.Lock()
	b.initialization.catalog.Models[3].Disabled = true
	b.mu.Unlock()
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, "opus"); err == nil {
		t.Fatal("context row restriction was bypassed")
	}
}

func TestClaudeModelCatalogReplacementFailsPendingRequest(t *testing.T) {
	b, _, capture, _ := claudeModelFixture(t, "hold-catalog", nil)
	finished := make(chan error, 1)
	go func() { _, err := b.ListModels(context.Background()); finished <- err }()
	claudeWait(t, "catalog request", time.Second, func() bool {
		for _, f := range claudeModelFrames(t, capture) {
			r, _ := f["request"].(map[string]any)
			if r["subtype"] == "list_models" {
				return true
			}
		}
		return false
	})
	if _, err := b.NewOffice(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("stale catalog request succeeded after replacement")
		}
	case <-time.After(time.Second):
		t.Fatal("replacement left control waiting")
	}
}

func TestClaudeAgentModelHookUnmatchedDispatchUntouched(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "Explore", "sonnet")
	b, _, _, _ := claudeModelFixture(t, "", cfg)
	if _, _, err := b.initializedCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, input := range []map[string]any{{"subagent_type": "researcher", "prompt": "keep"}, {"prompt": "native default agent"}, {"subagent_type": "Explore", "model": nil}, {"subagent_type": "Explore", "model": ""}} {
		req := claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": input}}
		b.mu.Lock()
		result := b.agentModelHookResultLocked(req)
		b.mu.Unlock()
		if len(result) != 0 {
			t.Fatalf("unmatched or explicit dispatch changed: %+v", result)
		}
	}
	if cfg.EffectiveModel(config.BackendNameClaude, "Explore") != "sonnet" {
		t.Fatal("backend changed caller configuration")
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{Agent: "Explore"}, "haiku"); err != nil {
		t.Fatal(err)
	}
	if cfg.EffectiveModel(config.BackendNameClaude, "Explore") != "sonnet" {
		t.Fatal("runtime selection mutated caller configuration")
	}
}

// TestClaudeAgentModelHookResultLockedSurvivesDriftedInput exercises
// agentModelHookResultLocked directly (no subprocess) against every shape
// requirement 4 names as a possible real-CLI drift: a wrong hook_event_name,
// a wholly absent Input map, an Input map with tool_input missing or of the
// wrong type, and a completely renamed inner shape. Each case must return a
// well-formed (non-nil, injection-free) result without panicking — the
// caller (claude.go:649) always has something safe to hand to
// claudeHookResponseLine.
func TestClaudeAgentModelHookResultLockedSurvivesDriftedInput(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "Explore", "sonnet")
	b, _, _, _ := claudeModelFixture(t, "", cfg)
	if _, _, err := b.initializedCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		req  claudeControlRequest
	}{
		{
			name: "unexpected hook_event_name",
			req: claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{
				"hook_event_name": "PostToolUse", "tool_name": "Agent",
				"tool_input": map[string]any{"subagent_type": "Explore"},
			}},
		},
		{
			name: "input entirely absent",
			req:  claudeControlRequest{CallbackID: claudeAgentModelHook, Input: nil},
		},
		{
			name: "tool_input key missing",
			req: claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{
				"hook_event_name": "PreToolUse", "tool_name": "Agent",
			}},
		},
		{
			name: "tool_input not a map (string)",
			req: claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{
				"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": "not-an-object",
			}},
		},
		{
			name: "tool_input not a map (array)",
			req: claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{
				"hook_event_name": "PreToolUse", "tool_name": "Agent", "tool_input": []any{"Explore"},
			}},
		},
		{
			name: "fully renamed/unrecognized inner shape",
			req: claudeControlRequest{CallbackID: claudeAgentModelHook, Input: map[string]any{
				"event": "PreToolUse", "tool": "Agent", "payload": map[string]any{"agent_type": "Explore"},
			}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("agentModelHookResultLocked panicked on %q: %v", tc.name, r)
				}
			}()
			b.mu.Lock()
			result := b.agentModelHookResultLocked(tc.req)
			b.mu.Unlock()
			if result == nil {
				t.Fatal("nil result is not well-formed: claudeHookResponseLine must always get a non-nil map")
			}
			if len(result) != 0 {
				t.Fatalf("drifted/unrecognized input must not inject a model: %+v", result)
			}
		})
	}
}

func TestClaudeModelControlQueuedCancellationDoesNotWrite(t *testing.T) {
	for _, barrier := range []string{"model", "write"} {
		t.Run(barrier, func(t *testing.T) {
			b := newClaudeBackend("", "", nil)
			proc := &exec.Cmd{}
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			defer writer.Close()
			b.proc, b.procStdin = proc, writer
			b.initialization = &claudeInitialization{proc: proc, done: make(chan struct{}), complete: true}
			close(b.initialization.done)
			lock := &b.writeMu
			if barrier == "model" {
				lock = &b.modelMu
			}
			lock.Lock()
			defer lock.Unlock()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			result := make(chan error, 1)
			start := time.Now()
			go func() { result <- b.SetModel(ctx, state.ModelTarget{}, "sonnet") }()
			select {
			case err := <-result:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("error=%v", err)
				}
			case <-time.After(250 * time.Millisecond):
				t.Fatal("selection remained blocked behind cancelled barrier")
			}
			b.mu.Lock()
			seq, pending, uncertain := b.controlSeq, len(b.controls), b.modelUncertain
			b.mu.Unlock()
			if seq != 0 || pending != 0 || uncertain {
				t.Fatalf("expired operation enqueued: seq=%d pending=%d uncertain=%v", seq, pending, uncertain)
			}
			if err := reader.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			var buf [1]byte
			if n, err := reader.Read(buf[:]); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("queued operation emitted bytes: n=%d err=%v", n, err)
			}
			t.Logf("%s barrier: returned in %s; zero control requests and zero bytes", barrier, time.Since(start))
		})
	}
}

type claudeControlObservedPipe struct {
	*os.File
	entered chan struct{}
}

func (p *claudeControlObservedPipe) Write(data []byte) (int, error) {
	select {
	case p.entered <- struct{}{}:
	default:
	}
	return p.File.Write(data)
}

func TestClaudeModelControlBlockedPipeCancellationRestoresDeadline(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprint(manual), func(t *testing.T) {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			defer writer.Close()
			if err := writer.SetWriteDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			filled, err := writer.Write([]byte(strings.Repeat("x", 1<<20)))
			if filled == 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("pipe did not fill: n=%d err=%v", filled, err)
			}
			if err := writer.SetWriteDeadline(time.Time{}); err != nil {
				t.Fatal(err)
			}
			observed := &claudeControlObservedPipe{File: writer, entered: make(chan struct{}, 1)}
			b := newClaudeBackend("", "", nil)
			proc := &exec.Cmd{}
			b.proc, b.procStdin = proc, observed
			b.initialization = &claudeInitialization{proc: proc, done: make(chan struct{}), complete: true}
			close(b.initialization.done)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			if manual {
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			result := make(chan error, 1)
			start := time.Now()
			go func() { result <- b.SetModel(ctx, state.ModelTarget{}, "sonnet") }()
			select {
			case <-observed.entered:
			case <-time.After(time.Second):
				t.Fatal("control never attempted blocked pipe write")
			}
			if manual {
				cancel()
			}
			select {
			case err := <-result:
				if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
			case <-time.After(250 * time.Millisecond):
				t.Fatal("pipe write ignored cancellation")
			}
			b.mu.Lock()
			uncertain, pending := b.modelUncertain, len(b.controls)
			b.mu.Unlock()
			if !uncertain || pending != 0 {
				t.Fatalf("uncertainty=%v pending=%d", uncertain, pending)
			}
			// Drain the original fill, then prove later ordinary permission/hook writes
			// are not poisoned by this control's deadline or cancellation callback.
			fill := make([]byte, filled)
			if _, err := io.ReadFull(reader, fill); err != nil {
				t.Fatal(err)
			}
			reply := []byte(`{"type":"control_response","response":{"request_id":"permission-after-timeout"}}`)
			if err := b.writeLine(reply); err != nil {
				t.Fatalf("later response inherited expired deadline: %v", err)
			}
			actual := make([]byte, len(reply)+1)
			if _, err := io.ReadFull(reader, actual); err != nil {
				t.Fatal(err)
			}
			if string(actual) != string(reply)+"\n" {
				t.Fatalf("unexpected control bytes before permission response: %q", actual)
			}
			t.Logf("manual cancellation=%v: blocked write returned in %s; subsequent permission response intact", manual, time.Since(start))
		})
	}
}

type claudePartialFrameWriter struct{ io.WriteCloser }

func (p claudePartialFrameWriter) SetWriteDeadline(at time.Time) error {
	return p.WriteCloser.(interface{ SetWriteDeadline(time.Time) error }).SetWriteDeadline(at)
}
func (p claudePartialFrameWriter) Write(data []byte) (int, error) {
	n, err := p.WriteCloser.Write(data[:len(data)/2])
	if err != nil {
		return n, err
	}
	return n, os.ErrDeadlineExceeded
}

func TestClaudeModelControlPartialFrameRecoversSession(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference(config.BackendNameClaude, "", "sonnet")
	b, log, _, argv := claudeModelFixture(t, "", cfg)
	if err := b.Send("establish resume pin"); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "resume pin", time.Second, func() bool { return b.PrimaryID() == "model-test-session" })
	b.writeMu.Lock()
	b.mu.Lock()
	b.procStdin = claudePartialFrameWriter{b.procStdin}
	b.mu.Unlock()
	b.writeMu.Unlock()
	err := b.SetModel(context.Background(), state.ModelTarget{}, "namespace/custom@preview")
	var partial *claudePartialControlWrite
	if !errors.As(err, &partial) {
		t.Fatalf("partial frame error=%v", err)
	}
	b.mu.Lock()
	stdin, broken, pending := b.procStdin, b.procWriteErr, len(b.controls)
	b.mu.Unlock()
	if stdin != nil || broken == nil || pending != 0 {
		t.Fatalf("corrupted stdin reusable: stdin=%v broken=%v pending=%d", stdin, broken, pending)
	}
	if err := b.Send("recover after partial frame"); err != nil {
		t.Fatal(err)
	}
	claudeWait(t, "replacement launch", time.Second, func() bool {
		data, _ := os.ReadFile(argv)
		return len(strings.Split(strings.TrimSpace(string(data)), "\n")) == 2
	})
	data, _ := os.ReadFile(argv)
	var args []string
	_ = json.Unmarshal([]byte(strings.Split(strings.TrimSpace(string(data)), "\n")[1]), &args)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--model sonnet") || !strings.Contains(joined, "--resume model-test-session") {
		t.Fatalf("recovery lost model/session: %s", joined)
	}
	if !log.hasStatusContaining("partial claude control frame") {
		t.Fatal("broken transport was not surfaced")
	}
	t.Log("partial frame detached stdin before return; next send resumed model-test-session with last acknowledged sonnet")
}
