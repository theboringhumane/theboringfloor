// models_live.go — ListModels, the /model picker's listing seam on the
// LIVE backend: GET /provider through doJSONCtx (the directory header +
// ?directory= query ride exactly like ListSessions, one level up in
// opencode.go). The serve answers {all: Provider[], connected: string[]}
// (shape verified against GET /doc 2026-08-22: Provider{ id, name, …,
// models: {modelID: Model{id,name,…}} }); we list the CONNECTED
// providers' models — those are the refs a prompt_async model override
// can actually run (postPrompt's {"providerID","modelID"} pair). When
// the wire connects NOTHING (empty connected — e.g. an unauthenticated
// serve that still knows providers) the mapping DEGRADES OPEN to every
// provider's models: the picker stays usable, a bad pick simply fails on
// the next send like a typo'd free-form ref.
//
// Calls are strictly on demand (the app fetches once per picker open and
// rides the result in the office state): exactly one HTTP request per
// invocation, no caching across calls, no background polling. Failure is
// degrade-open by contract: the error returns, the app closes the card
// and lands the classic free-form /model hint — never fatal.
package backend

import (
	"context"
	"net/http"
	"sort"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// ocProviderModel — one model entry inside a Provider's models map (only
// the fields the picker renders are decoded; capabilities/pricing stay on
// the wire untouched).
type ocProviderModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ocProvider — one provider row of GET /provider's "all" array.
type ocProvider struct {
	ID     string                     `json:"id"`
	Name   string                     `json:"name"`
	Models map[string]ocProviderModel `json:"models"`
}

// ocProviderList — the documented GET /provider response shape
// ({all, default, connected}; "default" decodes nowhere — the picker
// marks the CONFIGURED model app-side instead).
type ocProviderList struct {
	All       []ocProvider `json:"all"`
	Connected []string     `json:"connected"`
}

// ListModels — the /model picker's listing seam (the app type-asserts it;
// NOT part of state.Backend — the same additive pattern as ListSessions
// one file up). One GET /provider call, the connected providers' models
// flattened to state.ModelInfo rows (id falls back to the map key, name
// may stay empty — the picker renders the id then).
func (b *liveBackend) ListModels(ctx context.Context) ([]state.ModelInfo, error) {
	var wrap ocProviderList
	if err := b.doJSONCtx(ctx, http.MethodGet, "/provider", nil, &wrap); err != nil {
		return nil, err
	}
	return providerModels(wrap), nil
}

// providerModels maps the wire shape to picker rows: CONNECTED providers
// only when the connected list is non-empty, ALL providers' models
// otherwise (degrade-open); within a provider the models sort by id so
// the output is deterministic (the wire map walks randomly).
func providerModels(wrap ocProviderList) []state.ModelInfo {
	connected := make(map[string]bool, len(wrap.Connected))
	for _, id := range wrap.Connected {
		connected[id] = true
	}
	var out []state.ModelInfo
	for _, p := range wrap.All {
		if len(connected) > 0 && !connected[p.ID] {
			continue // not authenticated — its models can't run anyway
		}
		ids := make([]string, 0, len(p.Models))
		for id := range p.Models {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			m := p.Models[id]
			mid := m.ID
			if mid == "" {
				mid = id // the map key IS the model id when the body omits it
			}
			out = append(out, state.ModelInfo{Provider: p.ID, ID: mid, Name: m.Name})
		}
	}
	return out
}
