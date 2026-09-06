// charter_agentmodel.go — merges theboringfloor's per-agent model choices
// into a served project's local OpenCode config. Like charter.go, it never
// touches AGENTS.md, CLAUDE.md, or unrelated opencode.json fields: the
// member's hand-rolled shape is never clobbered.
package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

const opencodeSchemaURL = "https://opencode.ai/config.json"

// mergeAgentModel returns cfg with agent.<name>.model set to ref. It is the
// pure half of the agent-model config merge: every foreign field survives a
// map[string]any round-trip, and a hand-shaped value at any object boundary
// fails closed rather than being replaced.
func mergeAgentModel(cfg []byte, name, ref string) (merged []byte, changed bool, err error) {
	doc := map[string]any{}
	if len(cfg) != 0 {
		decoder := json.NewDecoder(bytes.NewReader(cfg))
		decoder.UseNumber()
		if err := decoder.Decode(&doc); err != nil {
			return nil, false, fmt.Errorf("unparseable json: %w", err)
		}
		if doc == nil {
			return nil, false, fmt.Errorf("top level is null — refusing to rewrite a hand-shaped config")
		}
	}

	rawAgent, hasAgent := doc["agent"]
	var agents map[string]any
	if hasAgent {
		if rawAgent == nil {
			return nil, false, fmt.Errorf("agent is null — refusing to rewrite a hand-shaped config")
		}
		var ok bool
		agents, ok = rawAgent.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("agent is %T, not an object — refusing to rewrite a hand-shaped config", rawAgent)
		}
	} else {
		agents = map[string]any{}
	}

	rawAgentConfig, hasAgentConfig := agents[name]
	var agentConfig map[string]any
	if hasAgentConfig {
		if rawAgentConfig == nil {
			return nil, false, fmt.Errorf("agent.%s is null — refusing to rewrite a hand-shaped config", name)
		}
		var ok bool
		agentConfig, ok = rawAgentConfig.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("agent.%s is %T, not an object — refusing to rewrite a hand-shaped config", name, rawAgentConfig)
		}
	} else {
		agentConfig = map[string]any{}
	}

	if rawModel, hasModel := agentConfig["model"]; hasModel {
		model, ok := rawModel.(string)
		if !ok {
			return nil, false, fmt.Errorf("agent.%s.model is %T, not a string — refusing to rewrite a hand-shaped config", name, rawModel)
		}
		if model == ref {
			return cfg, false, nil
		}
	}

	agentConfig["model"] = ref
	agents[name] = agentConfig
	doc["agent"] = agents
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(out, '\n'), true, nil
}

// ensureAgentModels merges every entry of models into
// <dir>/.opencode/opencode.json. Invalid entries are reported together but do
// not prevent valid entries from being applied. It deliberately operates only
// on the project's local config, never the user's global OpenCode config.
func ensureAgentModels(dir string, models map[string]string) (changed bool, err error) {
	ocDir := filepath.Join(dir, ".opencode")
	cfgPath := filepath.Join(ocDir, "opencode.json")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		return false, fmt.Errorf("agent models mkdir %s: %w", ocDir, err)
	}

	cfg, readErr := os.ReadFile(cfgPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("agent models read %s: %w", cfgPath, readErr)
	}
	if os.IsNotExist(readErr) {
		cfg = []byte("{\n  \"$schema\": \"" + opencodeSchemaURL + "\"\n}\n")
		changed = true
	}

	names := make([]string, 0, len(models))
	for name := range models {
		names = append(names, name)
	}
	sort.Strings(names)

	var errs []error
	for _, name := range names {
		ref := models[name]
		if !config.ValidAgentName(name) {
			errs = append(errs, fmt.Errorf("invalid agent name %q", name))
			continue
		}
		if !config.ValidModelRef(ref) {
			errs = append(errs, fmt.Errorf("invalid model ref %q for agent %q", ref, name))
			continue
		}
		merged, mergeChanged, mergeErr := mergeAgentModel(cfg, name, ref)
		if mergeErr != nil {
			errs = append(errs, fmt.Errorf("merge agent %q: %w", name, mergeErr))
			continue
		}
		if mergeChanged {
			cfg = merged
			changed = true
		}
	}

	if changed {
		if err := writeAgentModelConfigAtomic(cfgPath, cfg); err != nil {
			return false, err
		}
	}
	return changed, errors.Join(errs...)
}

// writeAgentModelConfigAtomic follows the ledger writer's temp-file plus
// rename discipline. Resolving a symlink first means replacing its target,
// not the member's symlink itself.
func writeAgentModelConfigAtomic(path string, data []byte) error {
	target := path
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("agent models resolve symlink %s: %w", path, err)
		}
		target = resolved
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("agent models stat %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "opencode-agent-models-*.tmp")
	if err != nil {
		return fmt.Errorf("agent models temp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("agent models chmod: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("agent models write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("agent models close: %w", err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("agent models rename: %w", err)
	}
	return nil
}
