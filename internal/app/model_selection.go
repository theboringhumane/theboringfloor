package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Every asynchronous hop and picker callback belongs to one request, target,
// and exact transport generation. Canceled catalogs and replaced backends cannot land late results.
type modelRequest struct {
	id         uint64
	generation *backendGeneration
	backend    string
	target     state.ModelTarget
	agents     bool
}

type modelSelection struct {
	request  modelRequest
	cancel   context.CancelFunc
	applying bool
	ctx      context.Context
}

type modelApplyMsg struct {
	request modelRequest
	ref     string
	err     error
}

// modelGeneration only inspects capabilities; all transport I/O takes a lease.
func (c *currentBackend) modelGeneration() (g *backendGeneration, list, agents, set bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	g = c.current
	if g == nil || g.state != backendAccepting || g.backend == nil {
		return nil, false, false, false
	}
	_, list = g.backend.(state.ModelLister)
	if _, targeted := g.backend.(targetModelListBackend); targeted {
		list = true
	}
	_, agents = g.backend.(state.ModelAgentLister)
	_, set = g.backend.(state.ModelSetter)
	return
}

func (c *currentBackend) modelGenerationActive(g *backendGeneration) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return g != nil && c.current == g && g.state == backendAccepting
}

// Unlike lease, this cannot silently redirect a queued model operation onto
// a replacement backend, whose accepted tokens and native agents may differ.
func (c *currentBackend) leaseModel(ctx context.Context, g *backendGeneration, fn func(state.Backend) error) error {
	if c == nil {
		return errBackendUnavailable
	}
	c.mu.Lock()
	if g == nil || c.current != g || g.state != backendAccepting || g.backend == nil {
		c.mu.Unlock()
		return errBackendUnavailable
	}
	if err := ctx.Err(); err != nil {
		c.mu.Unlock()
		return err
	}
	g.inFlight++
	c.mu.Unlock()
	defer c.release(g)
	return fn(g.backend)
}

func (m *Model) modelRequestActive(request modelRequest) bool {
	return m.modelSelection != nil && m.modelSelection.request == request && m.currentBackend.modelGenerationActive(request.generation)
}

func (m *Model) invalidateModelSelection() {
	if m.modelSelection != nil {
		m.modelSelection.cancel()
	}
	m.modelSelection = nil
	m.modelPick = nil
}

func (m *Model) newModelRequest(target state.ModelTarget, agents bool) (modelRequest, context.Context, bool) {
	if m.modelSelection != nil && m.modelSelection.applying {
		m.noticeErr("model selection already applying — wait for it to finish")
		return modelRequest{}, nil, false
	}
	m.invalidateModelSelection()
	name := m.backendName()
	if !config.ValidBackendName(name) {
		m.noticeErr(fmt.Sprintf("model selection unavailable: unknown backend %q — use /backend opencode|claudecode|codex", name))
		return modelRequest{}, nil, false
	}
	if m.backendTransitioning {
		m.noticeErr("model selection unavailable while the backend is changing — retry after it finishes")
		return modelRequest{}, nil, false
	}
	generation, _, _, _ := m.currentBackend.modelGeneration()
	if generation == nil {
		m.noticeErr("model selection unavailable: active backend unavailable — check /backend")
		return modelRequest{}, nil, false
	}
	m.modelRequestID++
	req := modelRequest{id: m.modelRequestID, generation: generation, backend: name, target: target, agents: agents}
	ctx, cancel := context.WithTimeout(context.Background(), modelListTimeout)
	m.modelSelection = &modelSelection{request: req, cancel: cancel, ctx: ctx}
	return req, ctx, true
}

func modelCommand(target state.ModelTarget) string {
	if target.Agent != "" {
		return "/submodel " + target.Agent
	}
	return "/model"
}

func modelLabel(req modelRequest) string {
	if req.target.Agent != "" {
		return "sub-agent model · " + req.backend + " → " + req.target.Agent + " = "
	}
	return "boss model · " + req.backend + " → "
}

func (m *Model) applyModelSelection(target state.ModelTarget, ref string) tea.Cmd {
	req, ctx, ok := m.newModelRequest(target, false)
	if !ok {
		return nil
	}
	return m.startModelApply(req, ctx, ref)
}

func (m *Model) startModelApply(req modelRequest, ctx context.Context, ref string) tea.Cmd {
	_, _, _, set := m.currentBackend.modelGeneration()
	if !set {
		m.invalidateModelSelection()
		m.noticeErr(fmt.Sprintf("%s: %s does not support model changes — check /backend or configure its native client", modelCommand(req.target), req.backend))
		return nil
	}
	m.modelSelection.applying = true
	if m.modelPick == nil {
		m.makeModelPicker(req)
	}
	if m.modelPick != nil {
		m.modelPick.SetTitle(modelPickerTitle(req) + " · applying…")
		m.modelPick.SetRows(buildModelRows([]state.ModelInfo{{ID: ref, Ref: ref, Name: "applying…"}}, ""))
		m.modelPick.SetPending(true)
	}
	current := m.currentBackend
	return func() tea.Msg {
		err := current.leaseModel(ctx, req.generation, func(b state.Backend) error {
			return b.(state.ModelSetter).SetModel(ctx, req.target, ref)
		})
		return modelApplyMsg{request: req, ref: ref, err: err}
	}
}

func (m *Model) handleModelApply(msg modelApplyMsg) tea.Cmd {
	if !m.modelRequestActive(msg.request) {
		return nil
	}
	if msg.err != nil {
		m.invalidateModelSelection()
		m.noticeErr(fmt.Sprintf("%s: %s rejected %q: %v", modelCommand(msg.request.target), msg.request.backend, msg.ref, msg.err))
		return nil
	}
	// The backend has acknowledged its runtime update. Only now may the app
	// commit its scoped preference; legacy fields and other backends stay intact.
	m.cfg.SetModelPreference(msg.request.backend, msg.request.target.Agent, msg.ref)
	// Use the same serialized config-write path as other slash settings.
	// A delayed whole-config snapshot could overwrite a later setting or
	// backend switch; only the backend operation belongs in a tea.Cmd.
	err := config.Save(m.cfg)
	m.invalidateModelSelection()
	if err != nil {
		m.noticeErr(modelLabel(msg.request) + msg.ref + " applied for future requests; brain.json save failed: " + err.Error())
	} else {
		m.notice(modelLabel(msg.request) + msg.ref + " (applied for future requests) · saved to brain.json")
	}
	return nil
}
