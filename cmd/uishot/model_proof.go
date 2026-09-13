package main

// This is a real-app UI+routing proof with recording native-catalog fixtures.
// It does not claim native transport coverage: backend adapter tests own that.
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

type modelProofBackend struct {
	*stubBackend
	name, initial, chosen, agent, otherAgent string
	models                                   []state.ModelInfo
	selected                                 map[string]string
	capture                                  []string
}

func newModelProofBackend(name string) *modelProofBackend {
	b := &modelProofBackend{stubBackend: &stubBackend{done: make(chan struct{})}, name: name, selected: map[string]string{}}
	switch name {
	case config.BackendNameDefault:
		b.initial, b.chosen, b.agent = "anthropic/claude-sonnet-4", "openai/gpt-5.4", "explore"
		b.otherAgent = "general"
		b.models = []state.ModelInfo{
			{Provider: "anthropic", ID: "claude-opus-4", Name: "Opus", IsDefault: true},
			{Provider: "anthropic", ID: "claude-sonnet-4", Name: "Sonnet"},
			{Provider: "openai", ID: "gpt-5.4", Name: "GPT"},
			{Provider: "openai", ID: "retired", Name: "Retired", Disabled: true},
		}
	case config.BackendNameCodex:
		b.initial, b.chosen = "gpt-5.3-codex", "gpt-5.4-mini"
		b.models = []state.ModelInfo{
			{ID: "catalog-1", Ref: "gpt-5.4", Name: "GPT", IsDefault: true},
			{ID: "catalog-2", Ref: "gpt-5.3-codex", Name: "Codex"},
			{ID: "catalog-3", Ref: "gpt-5.4-mini", Name: "Mini"},
			{ID: "catalog-4", Ref: "retired", Name: "Retired", Disabled: true},
		}
	case config.BackendNameClaude:
		b.initial, b.chosen, b.agent = "opus", "haiku", "Explore"
		b.otherAgent = "general-purpose"
		b.models = []state.ModelInfo{
			{ID: "sonnet", Ref: "sonnet", Name: "Sonnet", IsDefault: true},
			{ID: "opus", Ref: "opus", Name: "Opus"},
			{ID: "haiku", Ref: "haiku", Name: "Haiku"},
			{ID: "retired", Ref: "retired", Name: "Retired", Disabled: true},
		}
	}
	b.selected[""] = b.initial
	return b
}

func (b *modelProofBackend) ListModels(context.Context) ([]state.ModelInfo, error) {
	return append([]state.ModelInfo(nil), b.models...), nil
}

func (b *modelProofBackend) ListModelsForTarget(ctx context.Context, target state.ModelTarget) ([]state.ModelInfo, error) {
	if target.Agent != "" && target.Agent != b.agent && target.Agent != b.otherAgent {
		return nil, fmt.Errorf("unsupported native agent %q", target.Agent)
	}
	return b.ListModels(ctx)
}

func (b *modelProofBackend) ListModelAgents(context.Context) ([]state.ModelAgentInfo, error) {
	if b.agent == "" {
		return nil, fmt.Errorf("Codex per-agent model selection is unsupported")
	}
	return []state.ModelAgentInfo{{Name: b.agent, Description: "native exploration agent"}, {Name: b.otherAgent, Description: "native general agent"}}, nil
}

func (b *modelProofBackend) SetModel(_ context.Context, target state.ModelTarget, ref string) error {
	if target.Agent != "" && target.Agent != b.agent && target.Agent != b.otherAgent {
		return fmt.Errorf("unsupported native agent %q", target.Agent)
	}
	for _, model := range b.models {
		if model.SelectionRef() == ref && !model.Disabled {
			b.selected[target.Agent] = ref
			b.capture = append(b.capture, fmt.Sprintf("SetModel target=%q ref=%q", target.Agent, ref))
			return nil
		}
	}
	return fmt.Errorf("unavailable model %q", ref)
}

func (b *modelProofBackend) Send(prompt string) error {
	refs, _ := json.Marshal(b.selected)
	b.capture = append(b.capture, fmt.Sprintf("Send prompt=%q selected=%s", prompt, refs))
	return nil
}

// Drain all app commands, including Bubble Tea's private []Cmd sequence type.
// Timer commands may be discarded after 100ms; functional completion is checked
// at every checkpoint. No backend/event script, sleeps, or app.Init timer loop.
func modelProofKey(d *focusDriver, key tea.Key) error {
	next, cmd := d.m.Update(tea.KeyPressMsg(key))
	d.m = next.(app.Model)
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 64 {
			return fmt.Errorf("model command queue did not settle")
		}
		cmd, queue = queue[0], queue[1:]
		if cmd == nil {
			continue
		}
		result := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { result <- c() }(cmd)
		var msg tea.Msg
		select {
		case msg = <-result:
		case <-time.After(100 * time.Millisecond):
			continue
		}
		if msg == nil {
			continue
		}
		value, batchType := reflect.ValueOf(msg), reflect.TypeOf(tea.BatchMsg{})
		if value.Type().ConvertibleTo(batchType) {
			queue = append(value.Convert(batchType).Interface().(tea.BatchMsg), queue...)
			continue
		}
		next, more := d.m.Update(msg)
		d.m = next.(app.Model)
		queue = append(queue, more)
	}
	return nil
}

func modelProofType(d *focusDriver, value string) {
	for _, r := range value {
		d.send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

func modelProofSlash(d *focusDriver, command string) error {
	modelProofType(d, command)
	// Bare commands first apply the popover row; commands with arguments
	// dispatch directly, so a second Enter would select the first model.
	presses := 1
	if !strings.Contains(command, " ") {
		presses = 2
	}
	for range presses {
		if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
			return err
		}
	}
	return nil
}

func modelProofFrame(out *strings.Builder, d *focusDriver, label string, width int, wants ...string) error {
	frame := ansi.Strip(d.m.Frame())
	for _, want := range wants {
		if !strings.Contains(frame, want) {
			return fmt.Errorf("%s missing %q\n%s", label, want, frame)
		}
	}
	for _, row := range strings.Split(frame, "\n") {
		if ansi.StringWidth(row) > width {
			return fmt.Errorf("%s overflow: %d > %d", label, ansi.StringWidth(row), width)
		}
	}
	fmt.Fprintf(out, "===== MODEL PROOF %s =====\n%s\n", label, frame)
	return nil
}

func driveModelProof(name string) (string, error) {
	b := newModelProofBackend(name)
	cfg := config.Default()
	cfg.Backend.Name = name
	cfg.SetModelPreference(name, "", b.initial)
	cfg.UI.AmbientChatter = false
	d := &focusDriver{m: app.New(b, cfg)}
	d.m.SelectTab("chat")
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	d.send(state.Event{Kind: state.EvStatus, Text: "UI+routing fixture · " + name})
	var out strings.Builder
	frame := func(label string, wants ...string) error {
		return modelProofFrame(&out, d, name+" / "+label, shotCols, wants...)
	}
	if err := modelProofSlash(d, "/model"); err != nil {
		return "", err
	}
	if err := frame("boss catalog", "BOSS MODEL · "+name, b.initial, "current", "backend default", "unavailable"); err != nil {
		return "", err
	}
	modelProofType(d, "retired")
	if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
		return "", err
	}
	if !d.m.ModelPickerOpen() || len(b.capture) != 0 {
		return "", fmt.Errorf("%s unavailable row accepted", name)
	}
	if err := frame("unavailable rejected", "1/4", "unavailable"); err != nil {
		return "", err
	}
	if err := modelProofKey(d, tea.Key{Code: 'u', Mod: tea.ModCtrl}); err != nil {
		return "", err
	}
	modelProofType(d, b.chosen)
	if err := frame("filtered native reference", "1/4", b.chosen); err != nil {
		return "", err
	}
	if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
		return "", err
	}
	if d.m.ModelPickerOpen() || b.selected[""] != b.chosen {
		return "", fmt.Errorf("%s boss selection did not finish", name)
	}
	if err := frame("boss accepted", "applied for future", "saved to brain.json"); err != nil {
		return "", err
	}
	if err := modelProofSlash(d, "/model"); err != nil {
		return "", err
	}
	modelProofType(d, b.chosen)
	if err := frame("boss reopened current", b.chosen, "current"); err != nil {
		return "", err
	}
	// Esc clears the filter, then closes; neither call may change a model.
	for range 2 {
		if err := modelProofKey(d, tea.Key{Code: tea.KeyEscape}); err != nil {
			return "", err
		}
	}
	if err := modelProofSlash(d, "/submodel"); err != nil {
		return "", err
	}
	if b.agent == "" {
		if err := frame("agent unsupported", "unsupported"); err != nil {
			return "", err
		}
		if len(b.capture) != 1 {
			return "", fmt.Errorf("Codex agent path changed a model")
		}
	} else {
		if err := frame("native agent types", "SUB-AGENT TYPE · "+name, b.agent, b.otherAgent); err != nil {
			return "", err
		}
		modelProofType(d, b.agent)
		if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
			return "", err
		}
		if err := frame("agent model catalog", "SUB-AGENT MODEL · "+b.agent+" · "+name, "backend default"); err != nil {
			return "", err
		}
		modelProofType(d, b.chosen)
		if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
			return "", err
		}
		if err := frame("agent accepted", "sub-agent model", "saved to brain.json"); err != nil {
			return "", err
		}
		if err := modelProofSlash(d, "/submodel "+b.agent); err != nil {
			return "", err
		}
		modelProofType(d, b.chosen)
		if err := frame("agent reopened current", b.chosen, "current"); err != nil {
			return "", err
		}
		// At 80 columns the entire target/backend title must remain visible.
		d.send(tea.WindowSizeMsg{Width: 80, Height: shotRows})
		if err := modelProofFrame(&out, d, name+" / narrow agent card", 80, "SUB-AGENT MODEL · "+b.agent+" · "+name, "current"); err != nil {
			return "", err
		}
		d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
		for range 2 {
			if err := modelProofKey(d, tea.Key{Code: tea.KeyEscape}); err != nil {
				return "", err
			}
		}
	}
	modelProofType(d, "model proof next send")
	if err := modelProofKey(d, tea.Key{Code: tea.KeyEnter}); err != nil {
		return "", err
	}
	wantCalls := 2
	if b.agent != "" {
		wantCalls++
	}
	if len(b.capture) != wantCalls || !strings.HasPrefix(b.capture[len(b.capture)-1], "Send ") {
		return "", fmt.Errorf("%s unexpected captures: %v", name, b.capture)
	}
	data, err := os.ReadFile(config.Path())
	if err != nil {
		return "", err
	}
	var saved config.Config
	if err := json.Unmarshal(data, &saved); err != nil {
		return "", err
	}
	if saved.EffectiveModel(name, "") != b.chosen || (b.agent != "" && (saved.EffectiveModel(name, b.agent) != b.chosen || b.selected[b.agent] != b.chosen)) {
		return "", fmt.Errorf("%s scoped persistence/target mismatch", name)
	}
	prefs, _ := json.Marshal(saved.ModelPreferences)
	fmt.Fprintf(&out, "--- %s recording fixture captures ---\n%s\nPersisted modelPreferences=%s\n", name, strings.Join(b.capture, "\n"), prefs)
	return out.String(), nil
}

func runModelProof(w io.Writer) error {
	root, err := os.MkdirTemp("", "theboringfloor-model-proof-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	oldHome, hadHome := os.LookupEnv("THEFLOOR_HOME")
	defer func() {
		_ = os.Chdir(oldDir)
		if hadHome {
			_ = os.Setenv("THEFLOOR_HOME", oldHome)
		} else {
			_ = os.Unsetenv("THEFLOOR_HOME")
		}
	}()
	project := filepath.Join(root, "model-proof")
	if err := os.Mkdir(project, 0755); err != nil {
		return err
	}
	if err := os.Chdir(project); err != nil {
		return err
	}
	chrome.SetTheme("noir")
	office.SetTheme("noir")
	var first string
	for run := range 2 {
		if err := os.Setenv("THEFLOOR_HOME", filepath.Join(root, fmt.Sprint(run))); err != nil {
			return err
		}
		var rendered strings.Builder
		for _, name := range []string{config.BackendNameDefault, config.BackendNameCodex, config.BackendNameClaude} {
			frames, err := driveModelProof(name)
			if err != nil {
				return err
			}
			rendered.WriteString(frames)
		}
		if run == 0 {
			first = rendered.String()
		} else if first != rendered.String() {
			return fmt.Errorf("model proof runs differ")
		}
	}
	_, err = fmt.Fprintf(w, "UI+routing proof: real app; recording native-catalog fixtures; no native CLI inference.\nScratch home and project; generated IDs and paths are not printed.\n%sPASS: two byte-identical runs; catalogs, disabled row, exact selections, persistence, next-send routing, narrow cards.\n", first)
	return err
}
