package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// OpenCode splits at the first slash: model IDs can themselves contain slashes.
func validOpenCodeModelRef(ref string) bool {
	provider, model, ok := strings.Cut(ref, "/")
	return ok && provider != "" && model != "" && !strings.ContainsFunc(ref, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
}

// Native names are JSON keys, not paths. Preserve custom names from /agent.
func validOpenCodeAgentName(name string) bool {
	return strings.TrimSpace(name) != "" && !strings.ContainsFunc(name, unicode.IsControl)
}

func (b *liveBackend) modelSelectionReady(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.fl.isStopped() {
		return errors.New("OpenCode backend stopped")
	}
	b.mu.Lock()
	ready := b.baseURL != ""
	b.mu.Unlock()
	if !ready {
		return errors.New("OpenCode backend not started")
	}
	return nil
}

type ocModelAgent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Hidden      bool   `json:"hidden"`
	Disabled    bool   `json:"disabled"`
	Disable     bool   `json:"disable"`
	Model       *struct {
		Provider string `json:"providerID"`
		ID       string `json:"modelID"`
	} `json:"model"`
}

func (b *liveBackend) nativeModelAgents(ctx context.Context) ([]ocModelAgent, error) {
	if err := b.modelSelectionReady(ctx); err != nil {
		return nil, err
	}
	var agents []ocModelAgent
	if err := b.doJSONCtx(ctx, http.MethodGet, "/agent", nil, &agents); err != nil {
		return nil, fmt.Errorf("OpenCode agent listing: %w", err)
	}
	return agents, nil
}

func (b *liveBackend) ListModelAgents(ctx context.Context) ([]state.ModelAgentInfo, error) {
	agents, err := b.nativeModelAgents(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]state.ModelAgentInfo, 0, len(agents))
	for _, agent := range agents {
		if (agent.Mode == "subagent" || agent.Mode == "all") && !agent.Hidden && !agent.Disabled && !agent.Disable && validOpenCodeAgentName(agent.Name) {
			rows = append(rows, state.ModelAgentInfo{Name: agent.Name, Description: agent.Description})
		}
	}
	return rows, nil
}

// SetModel accepts the boss override for the next prompt or merges a discovered
// native agent's model into project config. The app persists its own preference
// only after this succeeds; no shared config map is mutated here.
func (b *liveBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	if err := b.modelSelectionReady(ctx); err != nil {
		return err
	}
	if ref != "" && !validOpenCodeModelRef(ref) {
		return fmt.Errorf("OpenCode model %q must be a provider/model reference", ref)
	}
	if target.Agent == "" {
		b.mu.Lock()
		defer b.mu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		b.bossModel = ref
		return nil
	}
	b.modelMu.Lock()
	defer b.modelMu.Unlock()
	if err := b.modelRecoveryError(); err != nil {
		return err
	}
	agents, err := b.nativeModelAgents(ctx)
	if err != nil {
		return err
	}
	found := false
	previousNative := ""
	for _, agent := range agents {
		if agent.Name == target.Agent && (agent.Mode == "subagent" || agent.Mode == "all") && !agent.Hidden && !agent.Disabled && !agent.Disable {
			if agent.Model != nil {
				previousNative = agent.Model.Provider + "/" + agent.Model.ID
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("OpenCode agent %q is not a selectable native subagent", target.Agent)
	}
	if err := b.modelSelectionReady(ctx); err != nil {
		return err
	}
	if err := b.applyNativeAgentModel(ctx, target.Agent, ref, previousNative); err != nil {
		return err
	}
	b.mu.Lock()
	b.agentModels[target.Agent] = ref
	b.mu.Unlock()
	return nil
}

// v1.18.29's instance.dispose route clears cached agent/config state without
// deleting session records. Check idle status before refreshing that instance.
func (b *liveBackend) refreshModelConfig(ctx context.Context) error {
	if err := b.modelSelectionReady(ctx); err != nil {
		return err
	}
	var disposed bool
	if err := b.doJSONCtx(ctx, http.MethodPost, "/instance/dispose", nil, &disposed); err != nil {
		return err
	}
	if !disposed {
		return errors.New("OpenCode did not acknowledge instance config refresh")
	}
	return nil
}

func agentModelInConfig(raw []byte, name string) (string, error) {
	// The merge validates every object boundary and the model field's type.
	if _, _, err := mergeAgentModel(raw, name, ""); err != nil {
		return "", err
	}
	// Decode just the target; unrelated hand-shaped agent entries may be valid.
	var root map[string]json.RawMessage
	if len(raw) == 0 {
		return "", nil
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", err
	}
	var agents map[string]json.RawMessage
	if rawAgents := root["agent"]; rawAgents != nil {
		if err := json.Unmarshal(rawAgents, &agents); err != nil {
			return "", err
		}
	}
	if target := agents[name]; target != nil {
		var value struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(target, &value); err != nil {
			return "", err
		}
		return value.Model, nil
	}
	return "", nil
}

func (b *liveBackend) idleForModelRefresh(ctx context.Context) error {
	var statuses map[string]struct {
		Type string `json:"type"`
	}
	if err := b.doJSONCtx(ctx, http.MethodGet, "/session/status", nil, &statuses); err != nil {
		return fmt.Errorf("OpenCode cannot check idle state before config refresh: %w", err)
	}
	for _, status := range statuses {
		if status.Type != "idle" {
			return errors.New("OpenCode agent model changes require idle sessions; retry after active work finishes")
		}
	}
	return nil
}

func (b *liveBackend) applyNativeAgentModel(ctx context.Context, name, ref, previousNative string) error {
	if err := b.idleForModelRefresh(ctx); err != nil {
		return err
	}
	path := filepath.Join(b.directory, ".opencode", "opencode.json")
	before, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	previous, err := agentModelInConfig(before, name)
	if err != nil {
		return err
	}
	if err := b.modelSelectionReady(ctx); err != nil {
		return err
	}
	if _, err := ensureAgentModels(b.directory, map[string]string{name: ref}); err != nil {
		return err
	}
	b.modelRecovery = &ocModelRecovery{name: name, diskRef: previous, nativeRef: previousNative}
	applyErr := b.refreshModelConfig(ctx)
	if applyErr == nil {
		var agents []ocModelAgent
		agents, applyErr = b.nativeModelAgents(ctx)
		if applyErr == nil {
			matched := false
			for _, agent := range agents {
				if agent.Name == name && (ref == "" || (agent.Model != nil && agent.Model.Provider+"/"+agent.Model.ID == ref)) {
					matched = true
					break
				}
			}
			if !matched {
				applyErr = fmt.Errorf("OpenCode did not expose selected model %q for native agent %q after refresh", ref, name)
			}
		}
	}
	if applyErr == nil {
		b.modelRecovery = nil
		return nil
	}
	// Roll back just this model field. Never overwrite an intervening user edit.
	current, readErr := os.ReadFile(path)
	if readErr != nil {
		return fmt.Errorf("%w; selected model remains on disk because rollback read failed: %v", applyErr, readErr)
	}
	currentRef, readErr := agentModelInConfig(current, name)
	if readErr != nil || currentRef != ref {
		return fmt.Errorf("%w; project config changed during refresh, model rollback requires review", applyErr)
	}
	if _, err := ensureAgentModels(b.directory, map[string]string{name: previous}); err != nil {
		return fmt.Errorf("%w; selected model remains on disk because rollback failed: %v", applyErr, err)
	}
	// The failed selection's context may already be expired. Recovery gets one
	// independent bounded attempt, including native verification.
	recoveryCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := b.reconcileModelState(recoveryCtx); err != nil {
		return fmt.Errorf("%w; project model restored, but native restoration is unconfirmed (%v); prompts and agent changes blocked until verified reconciliation; otherwise restart the native OpenCode server and reopen the office", applyErr, err)
	}
	return fmt.Errorf("OpenCode agent model not applied; project and native model restored: %w", applyErr)
}

// modelMu owns this recovery record and holds prompts across refresh + rollback.
type ocModelRecovery struct{ name, diskRef, nativeRef string }

func (b *liveBackend) modelRecoveryError() error {
	if b.modelRecovery == nil {
		return nil
	}
	return errors.New("OpenCode native model state is unconfirmed; prompts and agent changes blocked until verified reconciliation; otherwise restart the native OpenCode server and reopen the office")
}

// ReconcileModelSelection explicitly retries recovery without modifying project
// config. An intervening model edit must be resolved by the user or by restarting the native OpenCode server.
func (b *liveBackend) ReconcileModelSelection(ctx context.Context) error {
	b.modelMu.Lock()
	defer b.modelMu.Unlock()
	return b.reconcileModelState(ctx)
}

func (b *liveBackend) reconcileModelState(ctx context.Context) error {
	recovery := b.modelRecovery
	if recovery == nil {
		return nil
	}
	if err := b.modelSelectionReady(ctx); err != nil {
		return err
	}
	if err := b.idleForModelRefresh(ctx); err != nil {
		return err
	}
	checkDisk := func() error {
		raw, err := os.ReadFile(filepath.Join(b.directory, ".opencode", "opencode.json"))
		if err != nil {
			return err
		}
		ref, err := agentModelInConfig(raw, recovery.name)
		if err != nil {
			return err
		}
		if ref != recovery.diskRef {
			return errors.New("project agent model changed; reconciliation will not overwrite it")
		}
		return nil
	}
	if err := checkDisk(); err != nil {
		return err
	}
	if err := b.refreshModelConfig(ctx); err != nil {
		return err
	}
	agents, err := b.nativeModelAgents(ctx)
	if err != nil {
		return err
	}
	for _, agent := range agents {
		if agent.Name != recovery.name {
			continue
		}
		ref := ""
		if agent.Model != nil {
			ref = agent.Model.Provider + "/" + agent.Model.ID
		}
		if ref != recovery.nativeRef {
			break
		}
		if err := checkDisk(); err != nil {
			return err
		}
		b.modelRecovery = nil
		return nil
	}
	return fmt.Errorf("OpenCode did not restore native model %q for agent %q", recovery.nativeRef, recovery.name)
}
