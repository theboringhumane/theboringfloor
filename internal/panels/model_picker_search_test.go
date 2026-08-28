// model_picker_search_test.go — proofs for the /model picker's
// type-to-filter search (the /session picker's exact contract, ported to
// the app-level float):
//
//	(a) typing NARROWS (case-insensitive substring over the
//	    "provider/id" ref + display name + provider), the card renders
//	    the "filter: <q>▏" row + the "N/M" badge, the cursor clamps onto
//	    the narrowed rows, and enter accepts the FILTERED highlight;
//	(b) backspace edits rune-wise, ctrl+u clears, and a dead filter
//	    renders the "(no matches)" row with enter a no-op;
//	(c) esc is TWO-STAGE: the first press clears a live filter (the
//	    list restores, onCancel silent), the second cancels;
//	(d) Paste (the app paste router's duck-typed seam) flattens
//	    newlines/CRs to single spaces, appends, refilters, returns nil;
//	(e) with no query the card keeps its pre-search face: the dim
//	    "type to narrow" invite and the plain total badge.
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// modelType feeds printable text one rune at a time (sessType's twin).
func modelType(p *ModelPicker, s string) {
	for _, r := range s {
		p.Key(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

// (a) narrowing: case-insensitive over ref + name + provider; the filter
// row, the N/M badge, the clamped cursor, and the filtered accept.
func TestModelPickerSearchNarrows(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())

	// move off the top first: the refilter must CLAMP the cursor back.
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if got := p.Sel(); got != 2 {
		t.Fatalf("precondition: cursor on row 3, sel=%d", got)
	}

	modelType(p, "sonnet")
	if n := len(p.filtered); n != 1 {
		t.Fatalf("an id fragment must narrow to one row, got %d", n)
	}
	if got := p.Sel(); got != 0 {
		t.Fatalf("the cursor must clamp onto the narrowed list, sel=%d", got)
	}
	frame := modelFrame(t, p, 80, 24)
	for _, want := range []string{"filter: sonnet", "1/5", "› anthropic/claude-sonnet-4-5"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the mid-filter card must show %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "gemini-2.5-pro") {
		t.Fatalf("non-matching rows must leave the list:\n%s", frame)
	}
	// enter accepts the FILTERED highlight.
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("enter must accept the narrowed row's ref, got %v", *picks)
	}
}

// (a-2) the match surface: UPPERCASE, the display name, the bare
// provider, and the contiguous "provider/id" ref form all narrow.
func TestModelPickerSearchMatchSurface(t *testing.T) {
	legs := []struct {
		query string
		wantN int
		first string // "" when wantN == 0
	}{
		{"GPT", 1, "openai/gpt-5"},                     // case-insensitive (id)
		{"gemini", 1, "google/gemini-2.5-pro"},         // the display name
		{"anthropic", 3, "anthropic/claude-haiku-4-5"}, // the bare provider
		{"openai/gpt", 1, "openai/gpt-5"},              // the contiguous ref form
		{"Opus 4", 1, "anthropic/claude-opus-4"},       // a multi-word name fragment
	}
	for _, l := range legs {
		p, _, _ := modelHarness()
		p.SetRows(modelRowsFixture())
		modelType(p, l.query)
		if n := len(p.filtered); n != l.wantN {
			t.Fatalf("%q: want %d rows, got %d", l.query, l.wantN, n)
		}
		if l.wantN > 0 {
			if got := p.filtered[0].Provider + "/" + p.filtered[0].ID; got != l.first {
				t.Fatalf("%q: the top narrowed row must be %q, got %q", l.query, l.first, got)
			}
		}
	}
}

// (a-3) movement stays CLAMPED over the narrowed list — no wrap-around.
func TestModelPickerSearchMoveClamps(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())
	modelType(p, "anthropic") // 3 narrowed rows
	for i := 0; i < 9; i++ {
		p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	if got := p.Sel(); got != 2 {
		t.Fatalf("down past the narrowed end must CLAMP at the last match, sel=%d", got)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "anthropic/claude-sonnet-4-5" {
		t.Fatalf("the clamped accept must fire the LAST narrowed row, got %v", *picks)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // one past the top: stays
	if got := p.Sel(); got != 0 {
		t.Fatalf("up past the narrowed top must CLAMP at 0, sel=%d", got)
	}
}

// (b) backspace edits rune-wise; ctrl+u clears and restores the full set.
func TestModelPickerSearchBackspaceCtrlU(t *testing.T) {
	p, _, _ := modelHarness()
	p.SetRows(modelRowsFixture())

	modelType(p, "gptx") // a dead end: nothing matches
	if n := len(p.filtered); n != 0 {
		t.Fatalf("gptx must dead-end, got %d rows", n)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace})) // "gpt"
	if n := len(p.filtered); n != 1 {
		t.Fatalf("backspace must revive the gpt match, got %d rows", n)
	}
	if got := p.filter; got != "gpt" {
		t.Fatalf("backspace must edit rune-wise, filter=%q", got)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: 'u', Mod: tea.ModCtrl}))
	if got := p.filter; got != "" {
		t.Fatalf("ctrl+u must clear the filter, got %q", got)
	}
	if n := len(p.filtered); n != len(modelRowsFixture()) {
		t.Fatalf("ctrl+u must restore every row, got %d", n)
	}
	// a backspace on an empty filter is spent, never a panic.
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if got := p.filter; got != "" {
		t.Fatalf("backspace on an empty filter must stay empty, got %q", got)
	}
}

// (b-2) a dead filter renders "(no matches)", the badge runs 0/M, and
// enter stays a no-op.
func TestModelPickerSearchNoMatches(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())
	modelType(p, "zzzzz")
	frame := modelFrame(t, p, 80, 24)
	for _, want := range []string{"(no matches)", "0/5", "filter: zzzzz"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the dead-filter card must show %q:\n%s", want, frame)
		}
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 0 {
		t.Fatalf("enter must be a no-op on a dead filter, got %v", *picks)
	}
}

// (c) esc is TWO-STAGE: the first press clears the live filter (the list
// restores, onCancel silent), the second cancels through onCancel.
func TestModelPickerSearchEscTwoStage(t *testing.T) {
	p, picks, cancels := modelHarness()
	p.SetRows(modelRowsFixture())
	modelType(p, "gpt")
	if n := len(p.filtered); n != 1 {
		t.Fatalf("precondition: the filter narrows to one row, got %d", n)
	}

	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 0 {
		t.Fatalf("the first esc must clear the filter, NOT cancel (cancels=%d)", *cancels)
	}
	if got := p.filter; got != "" {
		t.Fatalf("the first esc must clear the filter, got %q", got)
	}
	if n := len(p.filtered); n != len(modelRowsFixture()) {
		t.Fatalf("the first esc must restore the full list, got %d rows", n)
	}
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "openai/gpt-5") || !strings.Contains(frame, "type to narrow") {
		t.Fatalf("the restored card must show every row + the empty invite:\n%s", frame)
	}

	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 1 {
		t.Fatalf("the second esc (empty filter) must cancel exactly once, got %d", *cancels)
	}
	if len(*picks) != 0 {
		t.Fatalf("esc never picks, got %v", *picks)
	}
}

// (d) Paste — the app paste router's duck-typed seam: newline/CR runs
// flatten to single spaces, the text appends to the filter, the list
// refilters, and the cmd is nil (the paste is fully consumed).
func TestModelPickerSearchPaste(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows(modelRowsFixture())

	if cmd := p.Paste("claude\ropus"); cmd != nil {
		t.Fatalf("Paste must consume the paste (nil cmd), got %v", cmd)
	}
	if got := p.filter; got != "claude opus" {
		t.Fatalf("CR must flatten to a single space, filter=%q", got)
	}
	if n := len(p.filtered); n != 1 {
		t.Fatalf("the pasted filter must narrow to claude-opus-4, got %d rows", n)
	}
	// a second paste APPENDS; a trailing newline run drops away.
	if cmd := p.Paste("\n"); cmd != nil {
		t.Fatalf("Paste must consume the paste (nil cmd), got %v", cmd)
	}
	if got := p.filter; got != "claude opus" {
		t.Fatalf("a bare-newline paste must add nothing, filter=%q", got)
	}
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "filter: claude opus") || !strings.Contains(frame, "1/5") {
		t.Fatalf("the pasted filter must render like a typed one:\n%s", frame)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "anthropic/claude-opus-4" {
		t.Fatalf("enter must accept the paste-narrowed row, got %v", *picks)
	}
	// the nil picker (the closed seam) swallows a paste safely.
	var nilPick *ModelPicker
	if cmd := nilPick.Paste("anything"); cmd != nil {
		t.Fatalf("a nil picker must swallow a paste, got %v", cmd)
	}
}

// (e) no query: the card keeps its pre-search face — the dim invite row
// and the plain total badge (never "N/M").
func TestModelPickerSearchEmptyInvite(t *testing.T) {
	p, _, _ := modelHarness()
	p.SetRows(modelRowsFixture())
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "type to narrow") {
		t.Fatalf("an empty filter must render the dim invite:\n%s", frame)
	}
	if strings.Contains(frame, "5/5") {
		t.Fatalf("the badge stays the plain total without a query:\n%s", frame)
	}
}
