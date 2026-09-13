package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// modelsBackend — a recording backend WITH the /model picker's listing
// seam: models/listErr script the hop, listCalls proves the fetch runs
// exactly once per open (never polled).
type modelsBackend struct {
	recBackend
	models    []state.ModelInfo
	listErr   error
	listCalls int
	setCalls  []modelSetCall
	setErr    error
}

func (b *modelsBackend) ListModels(ctx context.Context) ([]state.ModelInfo, error) {
	b.listCalls++
	return b.models, b.listErr
}

type modelSetCall struct {
	target state.ModelTarget
	ref    string
}

func (b *modelsBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	b.setCalls = append(b.setCalls, modelSetCall{target, ref})
	return b.setErr
}

// readBrain reloads brain.json as persistCfg wrote it (scratchHome
// redirects the home dir; the ACCEPT proof reads the file back).
func readBrain(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

// sized feeds a terminal size so m.Frame() renders (office tests that only
// read chat text never need it; the card-overlay proofs do).
func sized(t *testing.T, m Model) Model {
	t.Helper()
	return runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 36})
}

// modelsFixture — deliberately UNSORTED with a duplicate, so the row
// builder's sort + dedupe have work to do.
func modelsFixture() []state.ModelInfo {
	return []state.ModelInfo{
		{Provider: "openai", ID: "gpt-5", Name: "GPT-5"},
		{Provider: "anthropic", ID: "claude-sonnet-4-5", Name: "Claude\n   Sonnet 4.5"},
		{Provider: "anthropic", ID: "claude-opus-4"},
		{Provider: "google", ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro"},
		{Provider: "openai", ID: "gpt-5", Name: "GPT-5"}, // the dup must collapse
	}
}

// (b) rows: sorted, deduped, marked, flattened.
func TestBuildModelRows(t *testing.T) {
	out := buildModelRows(modelsFixture(), "anthropic/claude-opus-4")
	if len(out) != 4 {
		t.Fatalf("the duplicate ref must collapse — want 4 rows, got %d: %+v", len(out), out)
	}
	want := []string{
		"anthropic/claude-opus-4",
		"anthropic/claude-sonnet-4-5",
		"google/gemini-2.5-pro",
		"openai/gpt-5",
	}
	for i, w := range want {
		if got := out[i].Provider + "/" + out[i].ID; got != w {
			t.Fatalf("row %d: want %q, got %q (sort provider→id)", i, w, got)
		}
	}
	if !out[0].Current || out[1].Current || out[2].Current || out[3].Current {
		t.Fatalf("only the configured model is marked Current: %+v", out)
	}
	if out[0].Name != "claude-opus-4" {
		t.Fatalf("a blank name must fall back to the id, got %q", out[0].Name)
	}
	if out[1].Name != "Claude Sonnet 4.5" {
		t.Fatalf("whitespace/newlines must flatten, got %q", out[1].Name)
	}
	// no marking when nothing is configured (server default).
	for _, r := range buildModelRows(modelsFixture(), "") {
		if r.Current {
			t.Fatalf("an empty current ref must mark NOTHING: %+v", r)
		}
	}
}

// (c) bare /model: picker opens loading, the hop fills rows, and the
// listing rides the office state across later events (state carry).
func TestModelPickerOpenAndStateCarry(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m := New(b, nil)

	m = runMsg(t, m, slashMsg{text: "/model"})
	if b.listCalls != 1 {
		t.Fatalf("one ListModels hop per bare /model, got %d", b.listCalls)
	}
	m = sized(t, m)
	if !m.ModelPickerOpen() {
		t.Fatalf("the picker must be open after the listing lands")
	}
	// the listing rides the office state…
	if len(m.st.Models) != 5 { // the fixture's raw length (dedupe is panel-side)
		t.Fatalf("st.Models must carry the raw listing, got %d rows", len(m.st.Models))
	}
	// …and a later backend event never disturbs it (reducer carry).
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "later"})
	if len(m.st.Models) != 5 || m.st.Models[0].Provider != "openai" {
		t.Fatalf("the listing must survive untouched across events: %+v", m.st.Models)
	}
	// the card renders over the frame with the sorted rows + cursor on 1.
	frame := ansi.Strip(m.Frame())
	for _, want := range []string{"BOSS MODEL", "› anthropic/claude-opus-4", "openai/gpt-5"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the open card must show %q:\n%s", want, frame)
		}
	}
}

// ACCEPT shares the asynchronous setter used by the manual slash command.
// Only acknowledgment commits and persists the scoped preference.
func TestModelPickerAcceptRoutesExistingPath(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model"})
	m = sized(t, m)

	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))  // to row 2 (anthropic/claude-sonnet-4-5)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // accept

	if m.ModelPickerOpen() {
		t.Fatalf("every accept path closes the picker")
	}
	if got := m.cfg.EffectiveModel(m.backendName(), ""); got != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("the pick must set the scoped preference after backend acknowledgment, got %q", got)
	}
	last := lastChat(t, m)
	if last.From != "office" || last.Meta == "error" {
		t.Fatalf("the switch notice must be a clean dim office notice: from=%q meta=%q", last.From, last.Meta)
	}
	if !strings.HasPrefix(last.Text, "boss model · opencode → anthropic/claude-sonnet-4-5 (applied for future requests) · ") {
		t.Fatalf("the EXACT free-form notice must land (consistency), got %q", last.Text)
	}
	// brain.json persists NOW — the picker-set model survives a restart.
	cfg := readBrain(t)
	if cfg.EffectiveModel("opencode", "") != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("brain.json must carry the pick immediately, got %q", cfg.Boss.Model)
	}
	// and a RE-OPEN marks the freshly set model Current in the card.
	m = runMsg(t, m, slashMsg{text: "/model"})
	frame := ansi.Strip(m.Frame())
	if !strings.Contains(frame, "anthropic/claude-sonnet-4-5") || !strings.Contains(frame, "· current") {
		t.Fatalf("the just-set model must render · current on re-open:\n%s", frame)
	}
}

// Manual selection bypasses listing and preserves native reference syntax.
func TestModelSlashFreeFormUnchanged(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()} // the seam EXISTS — the arg path must not consult it
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model openai/gpt-5"})

	if b.listCalls != 0 {
		t.Fatalf("the free-form path must never fetch, got %d ListModels calls", b.listCalls)
	}
	if m.ModelPickerOpen() {
		t.Fatalf("/model x/y must NEVER open the picker")
	}
	if got := m.cfg.EffectiveModel(m.backendName(), ""); got != "openai/gpt-5" {
		t.Fatalf("free-form must set cfg.Boss.Model verbatim, got %q", got)
	}
	last := lastChat(t, m)
	if !strings.HasPrefix(last.Text, "boss model · opencode → openai/gpt-5 (applied for future requests) · ") {
		t.Fatalf("the acknowledgment notice must name the backend: %q", last.Text)
	}
	// Native slash-less references are adapter-owned and pass through exactly.
	m = runMsg(t, m, slashMsg{text: "/model gpt-5"})
	if got := b.setCalls[len(b.setCalls)-1]; got.ref != "gpt-5" || got.target.Agent != "" {
		t.Fatalf("native ref must reach the boss setter unchanged: %+v", got)
	}
}

// A backend without listing explains the manual selection path.
func TestModelSlashFallbackNoSeam(t *testing.T) {
	scratchHome(t)
	m := New(&pinBackend{primary: "ses-live-9"}, nil)
	m = runMsg(t, m, slashMsg{text: "/model"})

	if m.ModelPickerOpen() {
		t.Fatalf("a seam-less backend must never open the picker")
	}
	last := lastChat(t, m)
	for _, want := range []string{
		"boss model · opencode: no saved override — set with /model <native-ref>",
		"(model picker unavailable: listing is not supported by opencode",
	} {
		if !strings.Contains(last.Text, want) {
			t.Fatalf("the fallback must contain %q:\n%s", want, last.Text)
		}
	}
	if last.Meta == "error" {
		t.Fatalf("the fallback is a dim note, never an error: meta=%q", last.Meta)
	}
}

// (f-2) FALLBACK #2: a listing FAILURE closes the opened card and prints
// the classic hint carrying the real error.
func TestModelSlashFallbackListingError(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{listErr: errors.New("opencode serve unreachable")}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model"})
	if b.listCalls != 1 {
		t.Fatalf("the listing hop must run (and fail) exactly once, got %d", b.listCalls)
	}
	if m.ModelPickerOpen() {
		t.Fatalf("a failed listing must close the picker")
	}
	last := lastChat(t, m)
	for _, want := range []string{
		"boss model · opencode: no saved override — set with /model <native-ref>",
		"model picker unavailable: opencode serve unreachable",
	} {
		if !strings.Contains(last.Text, want) {
			t.Fatalf("the error fallback must contain %q:\n%s", want, last.Text)
		}
	}
	if len(m.st.Models) != 0 {
		t.Fatalf("a failed listing must leave the state listing empty, got %v", m.st.Models)
	}
}

// (g) ESC: zero side effects — picker closed, no notice, no cfg change.
func TestModelPickerEscZeroEffects(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model"})
	if !m.ModelPickerOpen() {
		t.Fatalf("precondition: the picker is open")
	}
	before := len(m.st.Chat)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if m.ModelPickerOpen() {
		t.Fatalf("esc must close the picker")
	}
	if len(m.st.Chat) != before {
		t.Fatalf("esc appends NOTHING (zero side effects), chat %d -> %d", before, len(m.st.Chat))
	}
	if got := m.cfg.EffectiveModel(m.backendName(), ""); got != "" {
		t.Fatalf("esc must never change the configured model, got %q", got)
	}
}

// Escape rejects a late listing; the previously accepted catalog is retained.
func TestModelPickerEscBeatsLateListing(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model"}) // hop lands synchronously in tests
	if !m.ModelPickerOpen() {
		t.Fatalf("precondition: the picker is open")
	}
	// simulate the race: esc first, THEN the hop's landing msg arrives.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m = runMsg(t, m, modelsListMsg{models: modelsFixture()})
	if m.ModelPickerOpen() {
		t.Fatalf("a late landing must never re-open an esc'd picker")
	}
	if len(m.st.Models) != 5 {
		t.Fatalf("the previously accepted listing stays in office state, got %d rows", len(m.st.Models))
	}
}

// (h) YIELD: with the picker open, an incoming permission float hides the
// card and owns the arrows; once answered the picker is back and owns.
func TestModelPickerYieldsToPermission(t *testing.T) {
	scratchHome(t)
	b := &modelsBackend{models: modelsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/model"})
	m = sized(t, m)
	if !m.ModelPickerOpen() {
		t.Fatalf("precondition: the picker is open")
	}
	// a permission ask arrives: the float outranks browsing — the card
	// hides and the picker yields its keys.
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, EmployeeName: "boss",
		PermissionID: "per-1", ToolName: "bash", ToolSummary: "go build ./..."})
	frame := ansi.Strip(m.Frame())
	if strings.Contains(frame, "BOSS MODEL") {
		t.Fatalf("the picker must hide under the permission float:\n%s", frame)
	}
	if !strings.Contains(frame, "PERMISSION REQUIRED") {
		t.Fatalf("the permission card must render instead:\n%s", frame)
	}
	// while the float owns the slot the PICKER's cursor never moves — the
	// down key went to the permission menu instead (the chat panel's own
	// tests pin that half of the contract).
	if got := m.modelPick.Sel(); got != 0 {
		t.Fatalf("precondition: the picker cursor sits on row 1, sel=%d", got)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if got := m.modelPick.Sel(); got != 0 {
		t.Fatalf("the picker must YIELD its keys under a permission float, sel=%d", got)
	}
	// answering clears the float → the picker renders and owns again.
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	frame = ansi.Strip(m.Frame())
	if !strings.Contains(frame, "BOSS MODEL") {
		t.Fatalf("with the float gone the picker must render again:\n%s", frame)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if got := m.modelPick.Sel(); got != 1 {
		t.Fatalf("after the float clears the picker must own the arrows again, sel=%d", got)
	}
	if !strings.Contains(ansi.Strip(m.Frame()), "› anthropic/claude-sonnet-4-5") {
		t.Fatalf("the moved cursor must show on row 2")
	}
	// and a click while the picker is up leaks to NOTHING underneath.
	before := len(m.st.Chat)
	m = runMsg(t, m, tea.MouseClickMsg(tea.Mouse{X: 10, Y: 10, Button: tea.MouseLeft}))
	if len(m.st.Chat) != before {
		t.Fatalf("a click under the open card must leak nowhere, chat %d -> %d", before, len(m.st.Chat))
	}
}
