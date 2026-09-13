// Backend-aware model catalogs share the same searchable card for boss models,
// native agent types, and native agent models. Transport I/O is asynchronous.
package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Some adapters expose a narrower catalog for native sub-agents.
type targetModelListBackend interface {
	ListModelsForTarget(context.Context, state.ModelTarget) ([]state.ModelInfo, error)
}

type modelsListMsg struct {
	request modelRequest
	models  []state.ModelInfo
	err     error
}
type modelAgentsListMsg struct {
	request modelRequest
	agents  []state.ModelAgentInfo
	err     error
}
type modelPickMsg struct {
	request modelRequest
	ref     string
}
type modelPickCancelMsg struct{ request modelRequest }

const modelListTimeout = 10 * time.Second

func modelPickerTitle(req modelRequest) string {
	if req.agents {
		return "SUB-AGENT TYPE · " + req.backend
	}
	if req.target.Agent != "" {
		return "SUB-AGENT MODEL · " + req.target.Agent + " · " + req.backend
	}
	return "BOSS MODEL · " + req.backend
}

func (m *Model) makeModelPicker(req modelRequest) {
	m.modelPick = panels.NewModelPicker(
		func(ref string) tea.Cmd { return func() tea.Msg { return modelPickMsg{request: req, ref: ref} } },
		func() tea.Cmd { return func() tea.Msg { return modelPickCancelMsg{request: req} } },
	)
	m.modelPick.SetTitle(modelPickerTitle(req))
	hint := "No models listed by " + req.backend + ". Try " + modelCommand(req.target) + " <native-ref>."
	if req.agents {
		hint = "No native agent types listed by " + req.backend + ". Try /submodel <agent> <native-ref>."
	}
	m.modelPick.SetEmptyHint(hint)
}

func (m *Model) openModelPicker() tea.Cmd { return m.openTargetModelPicker(state.ModelTarget{}) }

func (m *Model) openTargetModelPicker(target state.ModelTarget) tea.Cmd {
	req, ctx, ok := m.newModelRequest(target, false)
	if !ok {
		return nil
	}
	_, list, _, _ := m.currentBackend.modelGeneration()
	if !list {
		m.invalidateModelSelection()
		m.notice(modelUnavailableNote(m.modelTargetHint(req), "listing is not supported by "+req.backend))
		return nil
	}
	m.makeModelPicker(req)
	current := m.currentBackend
	return func() tea.Msg {
		var models []state.ModelInfo
		err := current.leaseModel(ctx, req.generation, func(b state.Backend) error {
			var err error
			if targetLister, ok := b.(targetModelListBackend); ok {
				models, err = targetLister.ListModelsForTarget(ctx, req.target)
			} else {
				models, err = b.(state.ModelLister).ListModels(ctx)
			}
			return err
		})
		return modelsListMsg{request: req, models: models, err: err}
	}
}

func (m *Model) modelSetHintNote() string {
	return m.modelTargetHint(modelRequest{backend: m.backendName()})
}

func (m *Model) modelTargetHint(req modelRequest) string {
	cur := m.cfg.EffectiveModel(req.backend, req.target.Agent)
	if cur == "" {
		cur = "no saved override"
	}
	target := "boss model"
	if req.target.Agent != "" {
		target = "sub-agent model " + req.target.Agent
	}
	return fmt.Sprintf("%s · %s: %s — set with %s <native-ref>", target, req.backend, cur, modelCommand(req.target))
}

func modelUnavailableNote(hint, listErr string) string {
	return hint + "\n  (model picker unavailable: " + listErr + "; manual selection remains available when supported)"
}

func (m *Model) handleModelsList(msg modelsListMsg) {
	if !m.modelRequestActive(msg.request) {
		return
	}
	if msg.err != nil {
		m.invalidateModelSelection()
		m.noticeErr(modelUnavailableNote(m.modelTargetHint(msg.request), msg.err.Error()))
		return
	}
	m.st.Models = msg.models
	m.modelPick.SetRows(buildModelRows(msg.models, m.cfg.EffectiveModel(msg.request.backend, msg.request.target.Agent)))
}

func (m *Model) acceptModelPick(msg modelPickMsg) tea.Cmd {
	if !m.modelRequestActive(msg.request) || m.modelSelection.applying {
		return nil
	}
	if msg.request.agents {
		return m.openTargetModelPicker(state.ModelTarget{Agent: msg.ref})
	}
	// Give apply its own identity, making any duplicate accept/list callback
	// stale. Retain the card until acknowledgment; input waits for the backend because
	// an accepted native change cannot universally be rolled back.
	m.modelSelection.cancel()
	m.modelRequestID++
	req := msg.request
	req.id = m.modelRequestID
	ctx, cancel := context.WithTimeout(context.Background(), modelListTimeout)
	m.modelSelection = &modelSelection{request: req, cancel: cancel, ctx: ctx}
	m.makeModelPicker(req)
	m.modelPick.SetRows(buildModelRows(m.st.Models, m.cfg.EffectiveModel(req.backend, req.target.Agent)))
	return m.startModelApply(req, ctx, msg.ref)
}

func (m *Model) closeModelPicker()    { m.invalidateModelSelection() }
func (m Model) ModelPickerOpen() bool { return m.modelPick != nil }

func buildModelRows(models []state.ModelInfo, current string) []panels.ModelPickRow {
	sorted := append([]state.ModelInfo(nil), models...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Provider != sorted[j].Provider {
			return sorted[i].Provider < sorted[j].Provider
		}
		return sorted[i].ID < sorted[j].ID
	})
	out := make([]panels.ModelPickRow, 0, len(sorted))
	seen := map[string]bool{}
	for _, mi := range sorted {
		ref := mi.SelectionRef()
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		name := strings.Join(strings.Fields(mi.Name), " ")
		if name == "" {
			name = mi.ID
		}
		out = append(out, panels.ModelPickRow{Provider: mi.Provider, ID: mi.ID, Name: name,
			Ref: ref, Description: strings.Join(strings.Fields(mi.Description), " "), Disabled: mi.Disabled,
			IsDefault: mi.IsDefault, Current: ref == current && current != ""})
	}
	return out
}
