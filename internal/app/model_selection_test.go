package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Advance only the reducer so a test can delay execution or delivery of a hop.
func modelStep(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}
func modelResult(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected asynchronous model command")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				return modelResult(t, c)
			}
		}
		t.Fatal("empty command batch")
	}
	return msg
}

func TestModelSelectionNativeRowsAndDefaultMetadata(t *testing.T) {
	rows := buildModelRows([]state.ModelInfo{
		{ID: "friendly", Ref: "opus[1m]", Description: "Long\ncontext", IsDefault: true},
		{ID: "native", Ref: "namespace/model@preview", Disabled: true},
		{ID: "bare-token"},
		{Provider: "wrong", ID: "duplicate", Ref: "opus[1m]"},
	}, "namespace/model@preview")
	if len(rows) != 3 {
		t.Fatalf("native deduplication: %+v", rows)
	}
	if rows[0].Ref != "bare-token" || rows[1].Ref != "opus[1m]" || rows[1].Description != "Long context" || !rows[1].IsDefault || rows[1].Current || !rows[2].Disabled || !rows[2].Current {
		t.Fatalf("native rows: %+v", rows)
	}
}

func TestModelSelectionEffectiveLegacyPrecedenceAndIsolation(t *testing.T) {
	for _, backend := range []string{"opencode", "claudecode", "codex"} {
		t.Run(backend, func(t *testing.T) {
			scratchHome(t)
			cfg := config.Default()
			cfg.Backend.Name = backend
			cfg.Boss.Model = "legacy/boss"
			cfg.Backend.BossModel = "legacy/backend"
			b := &modelsBackend{models: []state.ModelInfo{{Provider: "legacy", ID: "boss"}, {Provider: "legacy", ID: "backend"}}}
			m := sized(t, runMsg(t, New(b, cfg), slashMsg{text: "/model"}))
			frame := ansi.Strip(m.Frame())
			if !strings.Contains(frame, "BOSS MODEL · "+backend) {
				t.Fatalf("title missing:\n%s", frame)
			}
			if strings.Contains(frame, "· current") != (backend == "opencode") {
				t.Fatalf("legacy isolation:\n%s", frame)
			}
			if backend == "opencode" && !strings.Contains(m.modelSetHintNote(), "legacy/backend") {
				t.Fatalf("wrong legacy precedence: %s", m.modelSetHintNote())
			}
			m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
			m = runMsg(t, m, slashMsg{text: "/model native[1m]"})
			if cfg.EffectiveModel(backend, "") != "native[1m]" || cfg.Boss.Model != "legacy/boss" || cfg.Backend.BossModel != "legacy/backend" {
				t.Fatalf("selection overwrote legacy: %+v", cfg)
			}
		})
	}
}

func TestModelSelectionAckBeforeScopedPersistence(t *testing.T) {
	scratchHome(t)
	cfg := config.Default()
	cfg.Backend.Name = "codex"
	b := &modelsBackend{}
	m, cmd := modelStep(t, New(b, cfg), slashMsg{text: "/model namespace/model@preview"})
	if len(b.setCalls) != 0 || cfg.EffectiveModel("codex", "") != "" {
		t.Fatal("setter/config work ran on UI thread before Cmd")
	}
	msg := modelResult(t, cmd)
	if len(b.setCalls) != 1 || cfg.EffectiveModel("codex", "") != "" {
		t.Fatal("config changed before acknowledgment reduction")
	}
	m, pending := modelStep(t, m, msg)
	if cfg.EffectiveModel("codex", "") != "namespace/model@preview" {
		t.Fatal("ack did not commit runtime preference")
	}
	if pending != nil {
		t.Fatal("ack left a delayed config save that could overwrite later settings")
	}
	data, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"codex"`) || !strings.Contains(string(data), `"boss": "namespace/model@preview"`) {
		t.Fatalf("scoped JSON:\n%s", data)
	}
	if lastChat(t, m).Text != "boss model · codex → namespace/model@preview (applied for future requests) · saved to brain.json" {
		t.Fatalf("notice: %s", lastChat(t, m).Text)
	}
	t.Logf("setter target=%q ref=%q; scoped JSON modelPreferences.codex.boss=%q; notice=%q", b.setCalls[0].target.Agent, b.setCalls[0].ref, readBrain(t).EffectiveModel("codex", ""), lastChat(t, m).Text)
}

func TestModelSelectionPersistencePreservesLaterRemoteSettingsAndBackendSwitch(t *testing.T) {
	scratchHome(t)
	installFactory(t, map[string]*swapStubBackend{"claudecode": newSwapStub("")})
	m, apply := modelStep(t, New(&modelsBackend{}, nil), slashMsg{text: "/model native/token"})
	m, pending := modelStep(t, m, modelResult(t, apply))
	if pending != nil {
		t.Fatal("model acknowledgment left a competing config-write command")
	}
	// Remote control produces slashMsg even when keyboard input is frozen.
	// Both queued config writes must survive the model selection's persistence.
	m = runMsg(t, m, state.Event{Kind: state.EvControlSend, ControlText: "/notify off"})
	m = runMsg(t, m, state.Event{Kind: state.EvControlSend, ControlText: "/power saver"})
	saved := readBrain(t)
	if saved.UI.Notifications != "off" || saved.UI.Power != config.PowerSaver || saved.EffectiveModel("opencode", "") != "native/token" {
		t.Fatalf("later remote settings were overwritten: %+v", saved)
	}
	m.st.Mode, m.bootDone, m.sessDir = state.ModeLive, true, t.TempDir()
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	saved = readBrain(t)
	if saved.Backend.Name != "claudecode" || saved.UI.Notifications != "off" || saved.UI.Power != config.PowerSaver || saved.EffectiveModel("opencode", "") != "native/token" {
		t.Fatalf("later backend persistence was overwritten: %+v", saved)
	}
	t.Logf("ack pending command=nil; persisted notifications=%s power=%s backend=%s modelPreferences.opencode.boss=%s", saved.UI.Notifications, saved.UI.Power, saved.Backend.Name, saved.EffectiveModel("opencode", ""))
}

func TestModelSelectionSurvivesDelayedBackendTransitionResults(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{}
	m := New(b, nil)
	// The generation is installed before backendReadyMsg completes setup.
	m, delayed := modelStep(t, m, backendReadyMsg{result: backendBuildMsg{
		backend: b, name: "codex", oldName: "opencode", transition: m.backendTransitionID,
	}})
	if got := readBrain(t).Backend.Name; got != "codex" {
		t.Fatalf("transition did not persist before returning commands: %q", got)
	}
	m = runMsg(t, m, slashMsg{text: "/model gpt-5.4"})
	m = runMsg(t, m, state.Event{Kind: state.EvControlSend, ControlText: "/notify off"})
	before, err := os.Stat(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	// Drain every retained transition command after the newer settings save.
	var drain func(tea.Cmd)
	drain = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				drain(child)
			}
		} else if msg != nil {
			m = runMsg(t, m, msg)
		}
	}
	drain(delayed)
	after, err := os.Stat(config.Path())
	if err != nil {
		t.Fatal(err)
	}
	saved := readBrain(t)
	if !os.SameFile(before, after) {
		t.Fatal("delayed backend result rewrote brain.json")
	}
	if saved.Backend.Name != "codex" || saved.UI.Notifications != "off" || saved.EffectiveModel("codex", "") != "gpt-5.4" {
		t.Fatalf("delayed transition overwrote newer settings: %+v", saved)
	}
	t.Logf("delayed transition commands performed no file replacement; persisted backend=%s notifications=%s modelPreferences.codex.boss=%s", saved.Backend.Name, saved.UI.Notifications, saved.EffectiveModel("codex", ""))
}

func TestModelSelectionPendingRejectsDuplicateAndEscape(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: []state.ModelInfo{{ID: "gpt-5.4"}}}
	m := runMsg(t, New(b, nil), slashMsg{text: "/model"})
	request := m.modelSelection.request
	m, cmd := modelStep(t, m, modelPickMsg{request: request, ref: "gpt-5.4"})
	pending := m.modelSelection.request
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = runMsg(t, m, modelPickMsg{request: request, ref: "other"})
	m = runMsg(t, m, modelPickCancelMsg{request: pending})
	m = runMsg(t, m, slashMsg{text: "/model other"})
	if m.modelSelection == nil || m.modelSelection.request != pending || len(b.setCalls) != 0 {
		t.Fatal("pending selection escaped or duplicated")
	}
	m = runMsg(t, m, modelResult(t, cmd))
	if len(b.setCalls) != 1 || b.setCalls[0].ref != "gpt-5.4" || m.ModelPickerOpen() {
		t.Fatalf("pending apply result: %+v", b.setCalls)
	}
}

func TestModelPickerCancelReopenIgnoresOldSuccessErrorAndAccept(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m, oldCmd := modelStep(t, New(b, nil), slashMsg{text: "/model"})
	old := m.modelSelection.request
	oldCtx := m.modelSelection.ctx
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if oldCtx.Err() == nil {
		t.Fatal("Escape did not cancel list context")
	}
	m = sized(t, runMsg(t, m, slashMsg{text: "/model"}))
	before, frame := len(m.st.Chat), ansi.Strip(m.Frame())
	for _, msg := range []tea.Msg{
		modelResult(t, oldCmd),
		modelsListMsg{request: old, models: []state.ModelInfo{{ID: "stale"}}},
		modelsListMsg{request: old, err: errors.New("old failure")},
		modelPickMsg{request: old, ref: "stale"},
		modelPickCancelMsg{request: old},
	} {
		m = runMsg(t, m, msg)
	}
	if len(m.st.Chat) != before || ansi.Strip(m.Frame()) != frame || len(b.setCalls) != 0 {
		t.Fatal("old request changed new picker or notice")
	}
}

func TestSubmodelCancelReopenIgnoresOldAgentError(t *testing.T) {
	scratchHome(t)
	b := &nativeModelsBackend{agents: []state.ModelAgentInfo{{Name: "worker"}}, modelsBackend: modelsBackend{models: modelsFixture()}}
	m, oldCmd := modelStep(t, New(b, nil), slashMsg{text: "/submodel"})
	old := m.modelSelection.request
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = runMsg(t, m, slashMsg{text: "/submodel worker"})
	current, before := m.modelSelection.request, len(m.st.Chat)
	m = runMsg(t, m, modelResult(t, oldCmd))
	m = runMsg(t, m, modelAgentsListMsg{request: old, err: errors.New("old agent failure")})
	if m.modelSelection.request != current || len(m.st.Chat) != before {
		t.Fatal("old agent result disturbed target picker")
	}
}

func TestModelSelectionReplacementRejectsQueuedApplyAndLateSuccess(t *testing.T) {
	scratchHome(t)
	oldBackend, replacement := &modelsBackend{}, &modelsBackend{}
	m, pending := modelStep(t, New(oldBackend, nil), slashMsg{text: "/model old/token"})
	old := m.modelSelection.request
	oldCtx := m.modelSelection.ctx
	_ = m.finishBackendTransition(backendBuildMsg{backend: replacement, transition: m.backendTransitionID, name: "codex"})
	if m.ModelPickerOpen() || oldCtx.Err() == nil {
		t.Fatal("replacement did not invalidate selection")
	}
	m = runMsg(t, m, modelResult(t, pending))
	m = runMsg(t, m, modelApplyMsg{request: old, ref: "old/token"})
	m = runMsg(t, m, modelApplyMsg{request: old, ref: "old/token", err: errors.New("late rejected")})
	if len(oldBackend.setCalls) != 0 || len(replacement.setCalls) != 0 || len(m.cfg.ModelPreferences) != 0 {
		t.Fatal("stale apply reached a backend or persisted")
	}
}

type blockedModelBackend struct {
	recBackend
	entered chan struct{}
	done    chan struct{}
	stopped chan struct{}
}

func (b *blockedModelBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	close(b.entered)
	<-ctx.Done()
	close(b.done)
	return ctx.Err()
}
func (b *blockedModelBackend) Stop() error { close(b.stopped); return nil }

func TestModelSelectionReplacementCancelsAdmittedLeaseBeforeStop(t *testing.T) {
	scratchHome(t)
	b := &blockedModelBackend{entered: make(chan struct{}), done: make(chan struct{}), stopped: make(chan struct{})}
	m, cmd := modelStep(t, New(b, nil), slashMsg{text: "/model token"})
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	select {
	case <-b.entered:
	case <-time.After(time.Second):
		t.Fatal("setter did not enter")
	}
	cleanup := m.finishBackendTransition(backendBuildMsg{backend: &modelsBackend{}, transition: m.backendTransitionID, name: "codex"})
	select {
	case <-b.done:
	case <-time.After(time.Second):
		t.Fatal("replacement did not cancel setter context")
	}
	m = runMsg(t, m, <-result)
	// Run the returned cleanup tree; Stop must follow the admitted lease.
	var drain func(tea.Cmd)
	drain = func(c tea.Cmd) {
		if c == nil {
			return
		}
		if batch, ok := c().(tea.BatchMsg); ok {
			for _, sub := range batch {
				drain(sub)
			}
		}
	}
	drain(cleanup)
	select {
	case <-b.stopped:
	case <-time.After(time.Second):
		t.Fatal("old backend was not stopped")
	}
	if len(m.cfg.ModelPreferences) != 0 {
		t.Fatal("canceled old apply persisted")
	}
}

func TestModelSelectionRejectedUnsupportedAndSaveFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    state.Backend
		want string
	}{
		{"rejected", &modelsBackend{setErr: errors.New("invalid native ref")}, "invalid native ref"},
		{"unsupported", &recBackend{}, "does not support model changes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scratchHome(t)
			m := runMsg(t, New(tc.b, nil), slashMsg{text: "/model token"})
			if lastChat(t, m).Meta != "error" || !strings.Contains(lastChat(t, m).Text, tc.want) || len(m.cfg.ModelPreferences) != 0 {
				t.Fatalf("failed apply: %+v", lastChat(t, m))
			}
		})
	}
	t.Run("save", func(t *testing.T) {
		scratchHome(t)
		if err := os.MkdirAll(filepath.Dir(config.Path()), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(config.Path(), 0700); err != nil {
			t.Fatal(err)
		}
		m := runMsg(t, New(&modelsBackend{}, nil), slashMsg{text: "/model token"})
		if lastChat(t, m).Meta != "error" || !strings.Contains(lastChat(t, m).Text, "applied for future requests; brain.json save failed:") || m.cfg.EffectiveModel("opencode", "") != "token" {
			t.Fatalf("save failure notice: %+v", lastChat(t, m))
		}
	})
}

func TestModelSelectionUnknownBackendAndManualWithoutCatalog(t *testing.T) {
	scratchHome(t)
	m := New(&modelsBackend{}, nil)
	m.st.BackendName = "future"
	m = runMsg(t, m, slashMsg{text: "/model token"})
	if lastChat(t, m).Meta != "error" || !strings.Contains(lastChat(t, m).Text, `unknown backend "future"`) || len(m.cfg.ModelPreferences) != 0 {
		t.Fatalf("unknown backend: %+v", lastChat(t, m))
	}
}

type setterOnlyBackend struct {
	recBackend
	calls []modelSetCall
}

func (b *setterOnlyBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	b.calls = append(b.calls, modelSetCall{target: target, ref: ref})
	return nil
}

func TestModelSelectionManualWhenListingUnsupported(t *testing.T) {
	scratchHome(t)
	b := &setterOnlyBackend{}
	m := runMsg(t, New(b, nil), slashMsg{text: "/model"})
	if m.ModelPickerOpen() || !strings.Contains(lastChat(t, m).Text, "/model <native-ref>") {
		t.Fatal("missing manual fallback")
	}
	m = runMsg(t, m, slashMsg{text: "/model exact/token/with/slashes"})
	if len(b.calls) != 1 || b.calls[0].ref != "exact/token/with/slashes" || m.cfg.EffectiveModel("opencode", "") != "exact/token/with/slashes" {
		t.Fatalf("manual selection rejected: %+v", b.calls)
	}
}

func TestModelSelectionEmptyCatalogAndDisabledRow(t *testing.T) {
	scratchHome(t)
	cfg := config.Default()
	cfg.Backend.Name = "codex"
	b := &modelsBackend{}
	m := sized(t, runMsg(t, New(b, cfg), slashMsg{text: "/model"}))
	if frame := ansi.Strip(m.Frame()); !strings.Contains(frame, "No models listed by codex") {
		t.Fatalf("empty catalog frame:\n%s", frame)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	b.models = []state.ModelInfo{{ID: "unavailable", Disabled: true}}
	m = runMsg(t, m, slashMsg{text: "/model"})
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(b.setCalls) != 0 || !m.ModelPickerOpen() || len(cfg.ModelPreferences) != 0 {
		t.Fatal("disabled row accepted")
	}
}
