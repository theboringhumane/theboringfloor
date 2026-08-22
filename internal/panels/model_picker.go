// model_picker.go — the /model picker: a floating, centered,
// accent-bordered card mounted at the FRAME level (unlike the /session
// picker, which splices inside the chat panel — the chat panel is
// frozen for this feature, so the app owns this picker outright and
// overlays the composed frame: ModelPickerFrame below is the app's one
// mount point). It lists the backend's switchable boss models (the
// ListModels seam: live = GET /provider, demo/harness = fixed fixture)
// and switches the boss model LIVE via the existing /model slash path.
//
// Rows: "provider/id" left (the CURRENT pick accented + marked
// "· current"), the serve's display name dim at the meta column. Keys
// while open (the app routes EVERY key here while no permission/question
// float outranks it): ↑/↓ walk the cursor CLAMPED (a model list reads
// top-down once — no wrap-around), enter accepts the highlighted row's
// full "provider/id" ref through onPick, esc cancels through onCancel
// (zero side effects of its own — the app closes the card). Every other
// key is swallowed: the list is the connected providers' full set and
// searching was deliberately left out (a follow-up can grow a filter row
// the /session picker's way without touching the key contract). The card
// is keys-only: the app swallows clicks while it is up.
//
// The picker opens in a LOADING state ("fetching models…") — the app's
// ListModels hop rides a tea.Cmd, never blocking the input loop — and
// SetRows fills it; a listing failure never reaches here (the app closes
// the card and prints the classic bare-/model hint note instead).
package panels

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
)

// ModelPickRow — one switchable boss model for the /model picker. Built
// app-side (sorted, Current marked from the configured boss.model) so
// the panel stays a dumb renderer.
type ModelPickRow struct {
	Provider string // the wire provider half of the pick's "provider/id" ref
	ID       string // the wire model half — what rides prompt_async
	Name     string // the serve's display label; "" renders the ID
	Current  bool   // the configured boss model RIGHT NOW
}

// ModelPicker — the open picker: the row set (app-sorted), the clamped
// cursor, and its live/final loading marker. loading marks the ListModels
// hop still in flight (rows empty).
type ModelPicker struct {
	rows    []ModelPickRow
	sel     int
	loading bool
	// onPick/onCancel ferry over tea.Msgs (the model value copy in Update
	// stays the single writer — the same contract the /session picker's
	// handlers keep).
	onPick   func(ref string) tea.Cmd // ref = "provider/id"
	onCancel func() tea.Cmd
}

// modelVisibleRows — the picker list window (same budget as the @ picker,
// the slash popover and the /session picker).
const modelVisibleRows = 8

// modelPickHint — the picker's dim footer.
const modelPickHint = "↑/↓: move · enter: switch · esc: cancel"

// modelPickHigh — the cursor row's reversed-accent run (sessHigh's twin,
// in the app pickers' accent color).
var modelPickHigh = lipgloss.NewStyle().Foreground(chrome.Accent).Reverse(true)

// NewModelPicker opens the card in its LOADING state; the app's fetch hop
// fills it via SetRows. pick fires enter's full "provider/id" ref, cancel
// fires esc (zero side effects — the app closes the card).
func NewModelPicker(pick func(ref string) tea.Cmd, cancel func() tea.Cmd) *ModelPicker {
	return &ModelPicker{loading: true, onPick: pick, onCancel: cancel}
}

// SetRows fills the card with the app's built rows (sorted, current
// marked) and leaves the loading state; the cursor starts on the top row
// and clamps to the list bounds.
func (p *ModelPicker) SetRows(rows []ModelPickRow) {
	if p == nil {
		return
	}
	p.rows = rows
	p.loading = false
	p.sel = 0
	if n := len(rows); p.sel >= n {
		p.sel = n - 1
	}
	if p.sel < 0 {
		p.sel = 0
	}
}

// Loading reports whether the fetch hop is still in flight (the card
// shows its placeholder row; enter is a no-op then).
func (p *ModelPicker) Loading() bool { return p == nil || p.loading }

// Sel — the cursor row index (tests pin the clamped-navigation contract;
// the app reads it to prove the picker yields under a parked float).
func (p *ModelPicker) Sel() int {
	if p == nil {
		return 0
	}
	return p.sel
}

// RowCount — the set row total (tests pin the count badge + dedupe).
func (p *ModelPicker) RowCount() int {
	if p == nil {
		return 0
	}
	return len(p.rows)
}

// modelMove walks the cursor by d rows CLAMPED to [0, len-1] — no
// wrap-around (the requirement: ↑ on the top row / ↓ on the bottom row
// simply stays put).
func (p *ModelPicker) modelMove(d int) {
	if p == nil || len(p.rows) == 0 {
		return
	}
	p.sel += d
	if p.sel < 0 {
		p.sel = 0
	}
	if p.sel >= len(p.rows) {
		p.sel = len(p.rows) - 1
	}
}

// modelCurrentRef is the highlighted row's full "provider/id" ref,
// ok=false while loading / on an empty listing — enter is a no-op then.
func (p *ModelPicker) modelCurrentRef() (string, bool) {
	if p == nil || p.loading || len(p.rows) == 0 || p.sel < 0 || p.sel >= len(p.rows) {
		return "", false
	}
	r := p.rows[p.sel]
	return r.Provider + "/" + r.ID, true
}

// Key handles ONE key routed by the app while the picker is open (the app
// claims every key for it — the textarea below is DISABLED): ↑/↓ move
// the clamped cursor, enter accepts through onPick, esc cancels through
// onCancel, everything else is swallowed.
func (p *ModelPicker) Key(msg tea.KeyPressMsg) tea.Cmd {
	if p == nil {
		return nil
	}
	switch msg.String() {
	case "esc":
		if p.onCancel != nil {
			return p.onCancel()
		}
		return nil
	case "up":
		p.modelMove(-1)
		return nil
	case "down":
		p.modelMove(1)
		return nil
	case "enter":
		ref, ok := p.modelCurrentRef()
		if !ok || p.onPick == nil {
			return nil
		}
		return p.onPick(ref)
	default:
		// keys-only navigation: every other key belongs to the card and
		// dies here (typing / pgup / tab never reach the office below).
		return nil
	}
}

// modelCard renders the picker's rows, each EXACTLY cardW display cells
// (rails included), so the frame splice replaces whole cells without
// shifting a row — perm_card's exact contract, one level up.
func (p *ModelPicker) modelCard(frameW int) (rows []string, cardW int) {
	cardW = floatCardW(frameW)
	inner := cardW - 2 // the content column between the │ rails
	rail := func(s string) string {
		return chrome.AccentText.Render("│") + s + chrome.AccentText.Render("│")
	}
	blank := strings.Repeat(" ", inner)

	rows = append(rows, chrome.AccentText.Render("╭"+strings.Repeat("─", inner)+"╮"))
	rows = append(rows, rail(blank))

	// Title row: ACCENT-bold header left, dim row-count badge right —
	// the badge only shows once rows have landed.
	title := "BOSS MODEL"
	badge := ""
	if !p.loading && len(p.rows) > 0 {
		badge = itoa(len(p.rows))
	}
	gap := inner - 2 - lipgloss.Width(title) - lipgloss.Width(badge)
	if gap < 1 {
		gap = 1
	}
	rows = append(rows, rail(" "+chrome.AccentText.Bold(true).Render(title)+strings.Repeat(" ", gap)+
		chrome.DimText.Render(badge)+" "))
	rows = append(rows, rail(blank))

	// List window: the fetching placeholder, the empty-listing row, or
	// the models themselves (windowed at modelVisibleRows).
	switch {
	case p.loading:
		rows = append(rows, rail(fitLabel("  "+chrome.DimText.Render("fetching models…"), inner)))
	case len(p.rows) == 0:
		rows = append(rows, rail(fitLabel("  "+chrome.DimText.Render("(no models reported — /model provider/model still works)"), inner)))
	default:
		start := 0
		if p.sel >= modelVisibleRows {
			start = p.sel - modelVisibleRows + 1
		}
		end := start + modelVisibleRows
		if end > len(p.rows) {
			end = len(p.rows)
		}
		for i := start; i < end; i++ {
			rows = append(rows, rail(modelMenuRow(p.rows[i], i == p.sel, inner)))
		}
	}
	rows = append(rows, rail(blank))

	// Footer hint — dim italic, wrapped to the content column.
	for _, ln := range strings.Split(strings.TrimRight(wrapPlain(modelPickHint, inner-2), "\n"), "\n") {
		rows = append(rows, rail(chrome.DimText.Italic(true).Render(" "+fitPlain(ln, inner-2))+" "))
	}
	rows = append(rows, chrome.AccentText.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return rows, cardW
}

// modelMenuRow renders one model row of EXACTLY inner cells: the cursor
// marker + "provider/id" left (the CURRENT model's ref also renders
// accented as its standing mark), the dim meta right (the serve's display
// name plus "· current" on the configured model), ANSI-aware-truncated
// (fitLabel, never rune-sliced). The highlighted row runs reversed-accent
// across the WHOLE content column.
func modelMenuRow(row ModelPickRow, on bool, inner int) string {
	mark := "  "
	if on {
		mark = "› "
	}
	ref := row.Provider + "/" + row.ID
	meta := row.Name
	if row.Current {
		if meta != "" {
			meta += " "
		}
		meta += "· current"
	}
	// the ref takes what's left after the meta column (one-cell gap min)
	refW := inner - lipgloss.Width(mark) - lipgloss.Width(meta) - 1
	if refW < 8 {
		refW = 8
	}
	ref = fitPlain(ref, refW)
	body := mark + ref
	if meta != "" {
		body += " " + meta
	}
	if on {
		return modelPickHigh.Render(fitLabel(body, inner))
	}
	out := mark
	if row.Current {
		out += chrome.AccentText.Render(ref)
	} else {
		out += ref
	}
	if meta != "" {
		out += " " + chrome.DimText.Render(meta)
	}
	return fitLabel(out, inner)
}

// OverlayFrame splices the picker's card over the assembled FRAME lines
// cell-wise (ANSI-aware): centered on both axes over the whole terminal
// (the opencode model-dialog spot — one level up from the chat-embedded
// floats, and layout-neutral: floor/sidebar/zen/mobile all splice the
// same). The background pixels outside the card survive and the row
// count never changes, so the frame never jumps when the picker opens.
func (p *ModelPicker) OverlayFrame(bg []string, w, h int) []string {
	if p == nil {
		return bg
	}
	rows, cardW := p.modelCard(w)
	top := (h - len(rows)) / 2
	if top < 0 {
		top = 0 // a very short terminal pins the card to the top edge
	}
	left := (w - cardW) / 2
	if left < 0 {
		left = 0
	}
	for i, row := range rows {
		y := top + i
		if y < 0 || y >= len(bg) {
			continue // a short frame clips the card instead of growing
		}
		bg[y] = permSplice(bg[y], left, row, lipgloss.Width(row), w)
	}
	return bg
}

// ModelPickerFrame is the app's one mount point: splice the open picker
// over the assembled frame string. A nil picker (closed / seam absent)
// returns the frame untouched.
func ModelPickerFrame(p *ModelPicker, frame string, w, h int) string {
	if p == nil {
		return frame
	}
	return strings.Join(p.OverlayFrame(strings.Split(frame, "\n"), w, h), "\n")
}
