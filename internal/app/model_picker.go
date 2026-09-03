// model_picker.go — the interactive /model picker (the app's half; the
// panel lives in internal/panels/model_picker.go):
//
//   - bare /model opens the picker over the whole frame (loading state)
//     while the backend's ListModels hop rides a tea.Cmd — the input
//     never stalls. Rows = the backend's switchable boss models (live:
//     GET /provider through the live backend's seam; demo/harness: the
//     fixed fixture), sorted by provider then id, the configured
//     boss.model marked "· current".
//   - ENTER switches the boss model LIVE: the picker's ref drives the
//     EXISTING /model-set slash path verbatim ("boss model → …" notice +
//     brain.json persist) — no second model-setting implementation may
//     grow here. Esc cancels with zero side effects.
//   - GRACEFUL DEGRADATION: no ListModels seam or a failed listing keeps
//     today's free-form behavior — the classic "boss model: … set with
//     /model provider/model" hint note lands, with a dim
//     picker-unavailable tail. Never a crash, never a hung input.
//
// The backend seam is ADDITIVE (the primarySeamBackend pattern: the app
// type-asserts it, harness stubs that never grew it stay valid).
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

// --- backend seam (additive; type-asserted, never added to state.Backend)

// modelListBackend — the /model picker's listing seam: every boss-switchable
// "provider/model" the backend knows. The live backend answers from
// GET /provider (connected providers + their models — models_live.go);
// the demo backend and harness stubs answer the fixed five-model fixture;
// a backend without it (an older harness stub) keeps the classic
// free-form /model hint verbatim. Exactly ONE hop per picker open — the
// result rides the office state, nothing polls behind it.
type modelListBackend interface {
	ListModels(ctx context.Context) ([]state.ModelInfo, error)
}

// --- tea messages ------------------------------------------------------------

// modelsListMsg — the ListModels hop's landing (a failed listing carries
// err; the picker falls back to the classic hint note then).
type modelsListMsg struct {
	models []state.ModelInfo
	err    error
}

// modelPickMsg — the picker accepted a row (its full "provider/id" ref).
type modelPickMsg struct{ ref string }

// modelPickCancelMsg — esc cancelled the picker (zero side effects).
type modelPickCancelMsg struct{}

// modelListTimeout bounds the ListModels hop — the picker must never hang
// the input while the server (or network) drags.
const modelListTimeout = 10 * time.Second

// openModelPicker — the BARE /model slash handler: a backend with the
// listing seam opens the card in its loading state and kicks the async
// ListModels hop; one without it lands the classic hint note with the
// dim picker-unavailable tail (graceful degradation, never a dead card).
func (m *Model) openModelPicker() tea.Cmd {
	lb, ok := m.backend.(modelListBackend)
	if !ok {
		m.notice(modelUnavailableNote(m.modelSetHintNote(), ""))
		return nil
	}
	m.modelPick = panels.NewModelPicker(
		func(ref string) tea.Cmd { return func() tea.Msg { return modelPickMsg{ref: ref} } },
		func() tea.Cmd { return func() tea.Msg { return modelPickCancelMsg{} } },
	)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), modelListTimeout)
		defer cancel()
		models, err := lb.ListModels(ctx)
		return modelsListMsg{models: models, err: err}
	}
}

// modelSetHintNote — today's bare-/model hint body, verbatim: the whole
// story in free-form mode, the graceful-degradation base when the picker
// cannot open (no seam / failed listing).
func (m *Model) modelSetHintNote() string {
	cur := string(m.cfg.Boss.Model)
	if cur == "" {
		cur = "server default"
	}
	return fmt.Sprintf("boss model: %s — set with /model provider/model (the backend honors it on the next send)", cur)
}

// modelUnavailableNote — the hint body plus the dim note that the picker
// is unavailable (listErr "" covers the no-seam backends; non-empty
// carries the listing failure). Mirrors the /session picker's fallback
// wording.
func modelUnavailableNote(hint, listErr string) string {
	if listErr != "" {
		return hint + "\n  (model picker unavailable: " + listErr + " — the note above is all there is)"
	}
	return hint + "\n  (model picker unavailable on this backend — the note above is all there is)"
}

// handleModelsList — the listing hop landed: a failure closes the card
// and prints the classic hint (+ the dim unavailable tail carrying the
// error); rows fill the still-open picker (an esc-cancel while the hop
// flew wins — the late landing is dropped) and ride the office state
// (st.Models — fetch-on-demand, no event, nothing polls). The current
// marking re-reads cfg HERE so it can never go stale between open and fill.
func (m *Model) handleModelsList(msg modelsListMsg) {
	if msg.err != nil {
		m.modelPick = nil
		m.notice(modelUnavailableNote(m.modelSetHintNote(), msg.err.Error()))
		return
	}
	m.st.Models = msg.models
	if m.modelPick == nil {
		return // esc'd while the hop was in flight — the listing still rides the state
	}
	m.modelPick.SetRows(buildModelRows(msg.models, string(m.cfg.Boss.Model)))
}

// acceptModelPick — the picker accepted a "provider/id" ref: the card
// closes HERE (every accept path ends it) and the ref drives the EXISTING
// /model-set slash path — same notice, same persist, same
// next-send-honors-it behavior as typing /model provider/model by hand
// (no duplicate model-setting logic).
func (m *Model) acceptModelPick(ref string) {
	m.modelPick = nil
	m.applySlash("/model " + ref)
}

// closeModelPicker — esc cancels with zero side effects: only the card
// closes (the classic hint note already told the story when it matters).
func (m *Model) closeModelPicker() {
	m.modelPick = nil
}

// ModelPickerOpen reports whether the /model picker card is open
// (loading or filled) — tests + the frame mount read it.
func (m Model) ModelPickerOpen() bool { return m.modelPick != nil }

// --- row building (pure — unit tests pin it) ---------------------------------

// buildModelRows turns the backend's listing into the picker's rows:
// sorted by provider then id (stable, professional menu order — the wire
// order is a map walk on the serve), the configured model marked Current
// (an exact "provider/id" match on cfg.Boss.Model — a server default,
// "", marks NOTHING). Duplicate refs collapse (the serve hands one map
// per provider, but belt-and-braces).
func buildModelRows(models []state.ModelInfo, current string) []panels.ModelPickRow {
	sorted := make([]state.ModelInfo, 0, len(models))
	sorted = append(sorted, models...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Provider != sorted[j].Provider {
			return sorted[i].Provider < sorted[j].Provider
		}
		return sorted[i].ID < sorted[j].ID
	})
	out := make([]panels.ModelPickRow, 0, len(sorted))
	seen := map[string]bool{}
	for _, mi := range sorted {
		ref := mi.Provider + "/" + mi.ID
		if seen[ref] {
			continue
		}
		seen[ref] = true
		name := strings.Join(strings.Fields(mi.Name), " ") // flatten newlines/gaps
		if name == "" {
			name = mi.ID
		}
		out = append(out, panels.ModelPickRow{
			Provider: mi.Provider,
			ID:       mi.ID,
			Name:     name,
			Current:  ref == current && current != "",
		})
	}
	return out
}
