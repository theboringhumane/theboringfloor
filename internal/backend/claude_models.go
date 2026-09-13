package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

const claudeControlTimeout = 15 * time.Second
const claudeAgentModelHook = "office-agent-model-default"

type claudeControlAck struct {
	Subtype   string          `json:"subtype"`
	RequestID string          `json:"request_id"`
	Response  json.RawMessage `json:"response"`
	Error     string          `json:"error"`
}
type claudeControlReply struct {
	body json.RawMessage
	err  error
}
type claudeControlPending struct {
	proc *exec.Cmd
	done chan claudeControlReply
}
type claudeNativeModel struct {
	Value         string `json:"value"`
	ResolvedModel string `json:"resolvedModel"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Disabled      bool   `json:"disabled"`
}
type claudeCatalog struct {
	Models            []claudeNativeModel    `json:"models"`
	UnavailableModels []claudeNativeModel    `json:"unavailable_models"`
	Agents            []state.ModelAgentInfo `json:"agents"`
	HooksApplied      *bool                  `json:"hooks_applied"`
}
type claudeInitialization struct {
	proc      *exec.Cmd
	requestID string
	done      chan struct{}
	complete  bool
	catalog   claudeCatalog
	err       error
}

func (b *liveClaudeBackend) finishInitializationLocked(init *claudeInitialization, body json.RawMessage, err error) {
	if init == nil || init.complete {
		return
	}
	if err == nil {
		err = json.Unmarshal(body, &init.catalog)
	}
	init.err, init.complete = err, true
	close(init.done)
}

func (b *liveClaudeBackend) handleControlResponseLocked(proc *exec.Cmd, ack claudeControlAck) {
	var err error
	if ack.Subtype != "success" {
		err = fmt.Errorf("claude control rejected: %s", ack.Error)
	}
	if init := b.initialization; init != nil && init.proc == proc && init.requestID == ack.RequestID {
		b.finishInitializationLocked(init, ack.Response, err)
		return
	}
	if pending := b.controls[ack.RequestID]; pending != nil && pending.proc == proc {
		delete(b.controls, ack.RequestID)
		pending.done <- claudeControlReply{body: ack.Response, err: err}
	}
}

func (b *liveClaudeBackend) failControlsLocked(proc *exec.Cmd, err error) {
	if init := b.initialization; init != nil && init.proc == proc {
		b.finishInitializationLocked(init, nil, err)
	}
	for id, pending := range b.controls {
		if pending.proc == proc {
			delete(b.controls, id)
			pending.done <- claudeControlReply{err: err}
		}
	}
}

// A cancelled caller must not acquire a queued barrier and send afterward.
func claudeLockContext(ctx context.Context, mu *sync.Mutex) error {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if mu.TryLock() {
			if err := ctx.Err(); err != nil {
				mu.Unlock()
				return err
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type claudeControlNotSent struct{ error }
type claudePartialControlWrite struct{ error }

func (e *claudePartialControlWrite) Unwrap() error { return e.error }

func (e *claudeControlNotSent) Unwrap() error { return e.error }

// exec.StdinPipe returns a deadline-capable *os.File. Never start an
// uncancellable fallback Write. Join cancellation before clearing the deadline
// and releasing writeMu, so subsequent permission/hook writes remain usable.
func claudeWriteControl(ctx context.Context, stdin io.Writer, line []byte) (attempted bool, err error) {
	pipe, ok := stdin.(interface {
		io.Writer
		SetWriteDeadline(time.Time) error
	})
	if !ok {
		return false, errors.New("claude stdin does not support bounded control writes")
	}
	deadline, _ := ctx.Deadline()
	if err = pipe.SetWriteDeadline(deadline); err != nil {
		return false, err
	}
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = pipe.SetWriteDeadline(time.Now()); close(finished) })
	defer func() {
		if !stop() {
			<-finished
		}
		_ = pipe.SetWriteDeadline(time.Time{})
	}()
	if err = ctx.Err(); err != nil {
		return false, err
	}
	n, writeErr := pipe.Write(line)
	err = writeErr
	if n < len(line) && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		} else if errors.Is(err, os.ErrDeadlineExceeded) {
			err = context.DeadlineExceeded
		}
	}
	if n > 0 && n < len(line) {
		err = &claudePartialControlWrite{fmt.Errorf("partial claude control frame: transport closed; next send resumes the session: %w", err)}
	}
	return true, err
}

// Correlate responses with both request and process, and release the write
// barrier while awaiting acknowledgment so permission and hook replies flow.
func (b *liveClaudeBackend) control(ctx context.Context, request map[string]any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, claudeControlTimeout)
	defer cancel()
	if err := claudeLockContext(ctx, &b.writeMu); err != nil {
		return nil, &claudeControlNotSent{err}
	}
	b.mu.Lock()
	proc, stdin := b.proc, b.procStdin
	err := ctx.Err()
	if err == nil && (proc == nil || stdin == nil || b.fl.isStopped()) {
		err = errors.New("claude process not running")
	}
	if err != nil {
		b.mu.Unlock()
		b.writeMu.Unlock()
		return nil, &claudeControlNotSent{err}
	}
	b.controlSeq++
	id := fmt.Sprintf("office-model-%d", b.controlSeq)
	pending := &claudeControlPending{proc: proc, done: make(chan claudeControlReply, 1)}
	b.controls[id] = pending
	b.mu.Unlock()
	defer func() { b.mu.Lock(); delete(b.controls, id); b.mu.Unlock() }()
	line, err := json.Marshal(map[string]any{"type": "control_request", "request_id": id, "request": request})
	attempted := false
	if err == nil {
		attempted, err = claudeWriteControl(ctx, stdin, append(line, '\n'))
	}
	var partial *claudePartialControlWrite
	if errors.As(err, &partial) {
		b.mu.Lock()
		if b.proc == proc {
			b.procStdin = nil
			b.procWriteErr = err
		}
		b.failControlsLocked(proc, err)
		b.mu.Unlock()
		_ = stdin.Close()
		if proc.Process != nil {
			_ = signalProcessGroup(proc.Process, syscall.SIGKILL)
		}
		b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[claude] " + err.Error()})
	}
	b.writeMu.Unlock()
	if err != nil {
		if !attempted {
			err = &claudeControlNotSent{err}
		}
		return nil, err
	}
	select {
	case reply := <-pending.done:
		return reply.body, reply.err
	case <-ctx.Done():
		return nil, fmt.Errorf("claude %v: %w", request["subtype"], ctx.Err())
	}
}

func (b *liveClaudeBackend) initializedCatalog(ctx context.Context) (claudeCatalog, *exec.Cmd, error) {
	ctx, cancel := context.WithTimeout(ctx, claudeControlTimeout)
	defer cancel()
	b.mu.Lock()
	init := b.initialization
	b.mu.Unlock()
	if init == nil {
		return claudeCatalog{}, nil, errors.New("claude model catalog unavailable before initialization")
	}
	select {
	case <-ctx.Done():
		return claudeCatalog{}, nil, fmt.Errorf("claude initialize: %w", ctx.Err())
	case <-init.done:
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.initialization != init || b.proc != init.proc {
		return claudeCatalog{}, nil, errors.New("claude process replaced or exited during initialization")
	}
	return init.catalog, init.proc, init.err
}

func claudeModelInfos(c claudeCatalog) []state.ModelInfo {
	models := make([]state.ModelInfo, 0, len(c.Models)+len(c.UnavailableModels))
	seen := make(map[string]bool)
	appendModel := func(row claudeNativeModel, unavailable bool) {
		if row.Value == "" || seen[row.Value] {
			return
		}
		seen[row.Value] = true
		id := row.ResolvedModel
		if id == "" {
			id = row.Value
		}
		models = append(models, state.ModelInfo{Provider: "claudecode", ID: id, Ref: row.Value, Name: row.DisplayName, Description: row.Description, Disabled: row.Disabled || unavailable, IsDefault: row.Value == "default"})
	}
	for _, row := range c.Models {
		appendModel(row, false)
	}
	for _, row := range c.UnavailableModels {
		appendModel(row, true)
	}
	return models
}

func (b *liveClaudeBackend) ListModels(ctx context.Context) ([]state.ModelInfo, error) {
	catalog, proc, err := b.initializedCatalog(ctx)
	if err != nil {
		return nil, err
	}
	body, err := b.control(ctx, map[string]any{"subtype": "list_models"})
	if err != nil {
		// Older CLIs expose the initialization catalog but no refresh RPC.
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown") || strings.Contains(msg, "unsupported") || strings.Contains(msg, "unrecognized") || strings.Contains(msg, "not supported") {
			return claudeModelInfos(catalog), nil
		}
		return nil, err
	}
	var refreshed claudeCatalog
	if err = json.Unmarshal(body, &refreshed); err != nil {
		return nil, fmt.Errorf("decode claude model catalog: %w", err)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.proc != proc {
		return nil, errors.New("claude process replaced during model catalog refresh")
	}
	b.initialization.catalog.Models = refreshed.Models
	b.initialization.catalog.UnavailableModels = refreshed.UnavailableModels
	return claudeModelInfos(refreshed), nil
}

func (b *liveClaudeBackend) ListModelAgents(ctx context.Context) ([]state.ModelAgentInfo, error) {
	catalog, _, err := b.initializedCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return append([]state.ModelAgentInfo(nil), catalog.Agents...), nil
}

func claudeAgentModelSupported(ref string) bool {
	switch ref {
	case "", "inherit", "sonnet", "opus", "haiku", "fable":
		return true
	}
	return false
}

func (b *liveClaudeBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Respect cancellation even while another switch owns the send barrier.
	ctx, cancel := context.WithTimeout(ctx, claudeControlTimeout)
	defer cancel()
	if err := claudeLockContext(ctx, &b.modelMu); err != nil {
		return err
	}
	defer b.modelMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	catalog, proc, err := b.initializedCatalog(ctx)
	if err != nil {
		return err
	}
	for _, model := range claudeModelInfos(catalog) {
		if model.Ref == ref && model.Disabled {
			return fmt.Errorf("claude model %q is unavailable: %s", ref, model.Description)
		}
	}
	if target.Agent != "" {
		if target.Agent == "fork" {
			return errors.New("claude fork agents always inherit the parent model")
		}
		if !claudeAgentModelSupported(ref) {
			return fmt.Errorf("claude agent model %q cannot be applied safely: native Agent accepts sonnet, opus, haiku, or fable; clear the choice to restore its native default", ref)
		}
		found := false
		for _, agent := range catalog.Agents {
			if agent.Name == target.Agent {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("claude native agent %q is unavailable", target.Agent)
		}
		if catalog.HooksApplied == nil || !*catalog.HooksApplied {
			return errors.New("claude did not acknowledge native agent model hooks; upgrade Claude Code")
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.proc != proc {
			return errors.New("claude process replaced during model selection")
		}
		if err := b.agentModelPreferenceErrorLocked(target.Agent, ref); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		b.agentModelRefs[target.Agent] = ref
		return nil
	}
	token := ref
	if token == "" {
		token = "default"
	}
	if _, err = b.control(ctx, map[string]any{"subtype": "set_model", "model": token}); err != nil {
		b.mu.Lock()
		// Cancellation does not prove the queued native operation was cancelled.
		// Reassert our last acknowledged choice before the next user message.
		var notSent *claudeControlNotSent
		if !errors.As(err, &notSent) && !strings.Contains(err.Error(), "control rejected:") {
			b.modelUncertain = true
		}
		b.mu.Unlock()
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.proc != proc {
		return errors.New("claude process replaced during model selection")
	}
	b.modelRef = ref
	b.modelUncertain = false
	return nil
}

// A native PreToolUse callback changes only the missing dispatch model. It
// neither replaces agent definitions nor grants permissions, and runs for
// auto-approved/bypass tool calls as well as calls which later ask permission.
func (b *liveClaudeBackend) agentModelHookResultLocked(request claudeControlRequest) map[string]any {
	output := map[string]any{}
	if request.CallbackID != claudeAgentModelHook || request.Input["hook_event_name"] != "PreToolUse" {
		return output
	}
	tool, _ := request.Input["tool_name"].(string)
	if tool != "Agent" && tool != "Task" {
		return output
	}
	input, ok := request.Input["tool_input"].(map[string]any)
	if !ok {
		return output
	}
	if _, explicit := input["model"]; explicit {
		return output
	}
	agent, _ := input["subagent_type"].(string)
	ref := b.agentModelRefs[agent]
	if ref == "" || ref == "inherit" || !claudeAgentModelSupported(ref) || agent == "fork" {
		return output
	}
	if b.agentModelPreferenceErrorLocked(agent, ref) != nil {
		return output
	}
	updated := make(map[string]any, len(input)+1)
	for key, value := range input {
		updated[key] = value
	}
	updated["model"] = ref
	output["hookSpecificOutput"] = map[string]any{"hookEventName": "PreToolUse", "updatedInput": updated}
	return output
}

func claudeHookResponseLine(id string, result map[string]any) []byte {
	body, _ := json.Marshal(map[string]any{"type": "control_response", "response": map[string]any{"subtype": "success", "request_id": id, "response": result}})
	return body
}

func (b *liveClaudeBackend) reconcileModelBeforeSend() error {
	b.mu.Lock()
	uncertain, ref := b.modelUncertain, b.modelRef
	b.mu.Unlock()
	if !uncertain {
		return nil
	}
	if ref == "" {
		ref = "default"
	}
	if _, err := b.control(context.Background(), map[string]any{"subtype": "set_model", "model": ref}); err != nil {
		return fmt.Errorf("restore last acknowledged claude model before sending: %w", err)
	}
	b.mu.Lock()
	b.modelUncertain = false
	b.mu.Unlock()
	return nil
}

// The Agent tool exposes its own alias catalog. A main-session context
// variant may advertise the family, but the Agent selector labels the plain
// alias explicitly and never promises that main-session context modifier.
func claudeAgentModelInfos(models []state.ModelInfo) []state.ModelInfo {
	filtered := make([]state.ModelInfo, 0, len(models))
	seen := make(map[string]int)
	for _, model := range models {
		alias := strings.TrimSuffix(model.Ref, "[1m]")
		if alias == "" || alias == "inherit" || !claudeAgentModelSupported(alias) {
			continue
		}
		if alias != model.Ref {
			model.Ref, model.ID = alias, alias
			model.Name = strings.ToUpper(alias[:1]) + alias[1:] + " · agent alias"
			model.Description += " Native Agent alias; the main-session [1m] context modifier is not used."
		}
		model.IsDefault = false
		if index, ok := seen[alias]; ok {
			if model.Disabled && !filtered[index].Disabled {
				filtered[index].Disabled = true
				filtered[index].Description += " " + model.Description
			}
			continue
		}
		seen[alias] = len(filtered)
		filtered = append(filtered, model)
	}
	return filtered
}

func (b *liveClaudeBackend) ListModelsForTarget(ctx context.Context, target state.ModelTarget) ([]state.ModelInfo, error) {
	models, err := b.ListModels(ctx)
	if err != nil || target.Agent == "" {
		return models, err
	}
	if target.Agent == "fork" {
		return nil, errors.New("claude fork agents always inherit the parent model")
	}
	return claudeAgentModelInfos(models), nil
}

// Initialization is the first authoritative inventory. Surface stale saved
// selections here, and never inject an unavailable/unsupported saved choice.
func (b *liveClaudeBackend) agentModelPreferenceErrorLocked(agent, ref string) error {
	if !claudeAgentModelSupported(ref) {
		return fmt.Errorf("native Agent accepts sonnet, opus, haiku, or fable")
	}
	if agent == "fork" {
		return errors.New("fork always inherits its parent model")
	}
	init := b.initialization
	if init == nil || !init.complete || init.err != nil {
		return errors.New("native initialization was not acknowledged")
	}
	if init.catalog.HooksApplied == nil || !*init.catalog.HooksApplied {
		return errors.New("native model hook was not acknowledged; upgrade Claude Code")
	}
	found := false
	for _, item := range init.catalog.Agents {
		if item.Name == agent {
			found = true
			break
		}
	}
	if !found {
		return errors.New("native agent type is unavailable")
	}
	for _, model := range claudeAgentModelInfos(claudeModelInfos(init.catalog)) {
		if model.Ref == ref && model.Disabled {
			return errors.New("model is unavailable: " + model.Description)
		}
	}
	return nil
}

func (b *liveClaudeBackend) agentModelWarningsLocked() []string {
	var warnings []string
	for agent, ref := range b.agentModelRefs {
		if ref == "" || ref == "inherit" {
			continue
		}
		if err := b.agentModelPreferenceErrorLocked(agent, ref); err != nil {
			warnings = append(warnings, fmt.Sprintf("saved model %q for agent %q was not applied: %s", ref, agent, err))
		}
	}
	sort.Strings(warnings)
	return warnings
}
