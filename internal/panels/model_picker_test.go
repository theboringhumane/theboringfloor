// model_picker_test.go — proofs for the /model picker card (the app-level
// float mounted via ModelPickerFrame):
//
//	(a) the LOADING state renders "fetching models…" and enter is a
//	    no-op while no rows have landed;
//	(b) rows render "provider" + name meta, the CURRENT model marked
//	    "· current", the title's row-count badge and the footer hint
//	    present, and the card splices as a frame overlay (row count never
//	    grows, background outside the card survives);
//	(c) ↑/↓ CLAMP at both ends (no wrap-around) and the list window follows;
//	(d) enter ACCEPTS the highlighted row's full "provider/id" ref through
//	    onPick; on an empty listing enter is a no-op;
//	(e) esc cancels through onCancel with zero pick side effects;
//	(f) every other key is swallowed — nothing mutates, nothing fires.
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// modelHarness — the picker with its seams recorded (mirroring the app's
// wiring: the handlers close over tea.Msg-bearing cmds, not bare nils).
func modelHarness() (p *ModelPicker, picks *[]string, cancels *int) {
	picks = &[]string{}
	cancels = new(int)
	p = NewModelPicker(func(ref string) tea.Cmd {
		*picks = append(*picks, ref)
		return func() tea.Msg { return nil }
	}, func() tea.Cmd {
		(*cancels)++
		return func() tea.Msg { return nil }
	})
	return p, picks, cancels
}

func modelRowsFixture() []ModelPickRow {
	return []ModelPickRow{
		{Provider: "anthropic", ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5"},
		{Provider: "anthropic", ID: "claude-opus-4", Name: "Claude Opus 4", Current: true},
		{Provider: "anthropic", ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5"},
		{Provider: "google", ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro"},
		{Provider: "openai", ID: "gpt-5", Name: "GPT-5"},
	}
}

// modelFrame renders the card over a plain-character background frame
// (80x24), the way the app's ModelPickerFrame mount does.
func modelFrame(t *testing.T, p *ModelPicker, w, h int) string {
	t.Helper()
	bg := make([]string, h)
	for i := range bg {
		bg[i] = strings.Repeat(".", w)
	}
	out := p.OverlayFrame(bg, w, h)
	return ansi.Strip(strings.Join(out, "\n"))
}

// (a) LOADING: the placeholder row renders, no accept can fire.
func TestModelPickerLoading(t *testing.T) {
	p, picks, _ := modelHarness()
	if !p.Loading() {
		t.Fatalf("a fresh picker must be loading")
	}
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "BOSS MODEL") || !strings.Contains(frame, "fetching models…") {
		t.Fatalf("the loading state must render header + placeholder:\n%s", frame)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 0 {
		t.Fatalf("enter must be a no-op while loading, got picks %v", *picks)
	}
	// the row-count badge only lands with rows
	if strings.Contains(frame, " 5 ") {
		t.Fatalf("no badge before rows land:\n%s", frame)
	}
}

// (b) rows render sorted furniture; the card splices as a frame overlay.
func TestModelPickerRendersRows(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())
	if p.Loading() {
		t.Fatalf("SetRows must leave the loading state")
	}
	frame := modelFrame(t, p, 80, 24)
	for _, want := range []string{
		"BOSS MODEL",
		"anthropic/claude-haiku-4-5", "Claude Haiku 4.5",
		"anthropic/claude-opus-4", "Claude Opus 4 · current",
		"anthropic/claude-sonnet-4-5", "Claude Sonnet 4.5",
		"google/gemini-2.5-pro", "Gemini 2.5 Pro",
		"openai/gpt-5", "GPT-5",
		"5", modelPickHint,
		"╭", "╰",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("picker furniture missing %q:\n%s", want, frame)
		}
	}
	// the cursor sits on the top row (clamped start) — "›" leads row 1.
	if !strings.Contains(frame, "› anthropic/claude-haiku-4-5") {
		t.Fatalf("row 1 must carry the › cursor:\n%s", frame)
	}
	// the card is an overlay: the frame row budget NEVER grows…
	bg := make([]string, 24)
	for i := range bg {
		bg[i] = strings.Repeat(".", 80)
	}
	out := p.OverlayFrame(bg, 80, 24)
	if n := len(out); n != 24 {
		t.Fatalf("the overlay must never grow the frame past its 24 rows, got %d", n)
	}
	// …and the background outside the card survives untouched (corners).
	first := ansi.Strip(out[0])
	first = strings.TrimRight(first, " ")
	if first != strings.Repeat(".", 80) {
		t.Fatalf("a row above the card must keep its background dots, got %q", first)
	}
	// picking fires the highlighted row's FULL "provider/id" ref.
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "anthropic/claude-haiku-4-5" {
		t.Fatalf("enter must fire the highlighted ref, got %v", *picks)
	}
	// a nil picker leaves the frame alone (ModelPickerFrame's closed arm).
	if got := ModelPickerFrame(nil, "left-alone", 10, 2); got != "left-alone" {
		t.Fatalf("a nil picker must not touch the frame, got %q", got)
	}
}

// (c) movement CLAMPS at both ends — no wrap-around: ↑ on the top row
// and ↓ on the bottom row simply stay put.
func TestModelPickerMoveClamps(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())

	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // already at the top: stays
	if got := p.sel; got != 0 {
		t.Fatalf("up on the first row must CLAMP at 0, sel=%d", got)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})) // to row 2
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})) // to row 3
	if got := p.sel; got != 2 {
		t.Fatalf("down-down must land on index 2, sel=%d", got)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("enter must accept the moved-to row, got %v", *picks)
	}
	// walk past the bottom: clamps at the last row, never wraps to the top.
	for i := 0; i < 12; i++ {
		p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	if got := p.sel; got != len(modelRowsFixture())-1 {
		t.Fatalf("down past the end must CLAMP at the last row, sel=%d", got)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 2 || (*picks)[1] != "openai/gpt-5" {
		t.Fatalf("the clamped bottom accept must fire the LAST row, got %v", *picks)
	}
	frame := modelFrame(t, p, 80, 24)
	if strings.Count(frame, "›") != 1 || !strings.Contains(frame, "› openai/gpt-5") {
		t.Fatalf("exactly one › cursor, on the clamped last row:\n%s", frame)
	}
}

// (c-2) the list window follows the cursor past modelVisibleRows.
func TestModelPickerWindowScrolls(t *testing.T) {
	p, _, _ := modelHarness()
	var rows []ModelPickRow
	for i := 0; i < 12; i++ {
		rows = append(rows, ModelPickRow{Provider: "anthropic", ID: itoa(i + 1)})
	}
	p.SetRows(rows)
	for i := 0; i < 9; i++ {
		p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	if got := p.sel; got != 9 {
		t.Fatalf("9 downs must land on index 9, sel=%d", got)
	}
	// row window: the top rows scrolled out, the cursor row visible.
	rendered, _ := p.modelCard(80)
	joined := ansi.Strip(strings.Join(rendered, "\n"))
	if !strings.Contains(joined, "› anthropic/10") {
		t.Fatalf("the cursor row must stay visible after scrolling:\n%s", joined)
	}
	if strings.Contains(joined, "anthropic/1 ") {
		t.Fatalf("row 1 must scroll out of the window past modelVisibleRows:\n%s", joined)
	}
}

// (d-2) an empty listing renders the honest row and enter is a no-op.
func TestModelPickerEmptyListing(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(nil)
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "(no models reported") {
		t.Fatalf("an empty listing must say so:\n%s", frame)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 0 {
		t.Fatalf("enter must be a no-op on an empty listing, got %v", *picks)
	}
}

// (e) esc cancels through onCancel with zero pick side effects — the
// panel fires the seam and STAYS open; the model's answer (setting
// modelPick nil) is what ends it (single-writer rule).
func TestModelPickerEscCancels(t *testing.T) {
	p, picks, cancels := modelHarness()
	p.SetRows(modelRowsFixture())
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 1 {
		t.Fatalf("esc must fire onCancel exactly once, got %d", *cancels)
	}
	if len(*picks) != 0 {
		t.Fatalf("esc cancels with zero pick side effects, got %v", *picks)
	}
	if p == nil {
		t.Fatalf("the picker itself must stay open until the MODEL closes it (single writer)")
	}
	if got := ModelPickerFrame(nil, "closed", 10, 2); got != "closed" {
		t.Fatalf("after the model closes it the card must not render")
	}
}

// (f) swallowed keys: typing (rune text), tab, pgup/pgdown — none of them
// move the cursor, fire a pick, or fire a cancel.
func TestModelPickerSwallowsOtherKeys(t *testing.T) {
	p, picks, cancels := modelHarness()
	p.SetRows(modelRowsFixture())
	for _, k := range []tea.KeyPressMsg{
		{Code: 'x', Text: "x"},
		{Code: tea.KeyTab},
		{Code: tea.KeyPgUp},
		{Code: tea.KeyPgDown},
		{Code: tea.KeyHome},
		{Code: tea.KeyEnd},
	} {
		p.Key(k)
	}
	if p.sel != 0 {
		t.Fatalf("swallowed keys must never move the cursor, sel=%d", p.sel)
	}
	if len(*picks) != 0 || *cancels != 0 {
		t.Fatalf("swallowed keys must never fire a seam (picks %v, cancels %d)", *picks, *cancels)
	}
}
