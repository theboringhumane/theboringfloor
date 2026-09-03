// session_picker.go — the /session picker: a floating, centered,
// accent-bordered card spliced OVER the assembled chat view (the exact
// float mechanics of the permission + question cards — perm_modal.go's
// floatCardW/permSplice and question_modal.go's owns-every-key contract).
// It lists the opencode server's ROOT sessions for this directory and
// re-anchors the office to the accepted one LIVE.
//
// Rows: title (the app pre-falls-back to a short id when the wire title
// is blank), relative age, message count, short id — the session the
// office is currently attached to is marked " · current" and rendered
// accented. Keys while open (question-modal style — the textarea below
// is DISABLED): typing NARROWS the list (case-insensitive substring over
// the title + session id, the @ picker's forgiving match), ↑/↓/tab walk
// the cursor, enter accepts the highlighted row (fires onSessionPick),
// esc is TWO-STAGE — the first press clears a live filter (the list
// restores), only an empty-filter esc cancels (fires onSessionCancel —
// zero side effects of its own; the app closes the card). pgup/pgdown
// still scroll the transcript. A bracketed paste lands in the filter too
// (Paste — the office paste router's duck-typed seam, newlines flattened
// to single spaces, the /model picker's exact contract). While a
// permission/question float owns the slot the picker yields its keys and
// simply waits underneath (a parked turn outranks browsing).
//
// With a live filter every non-cursor row re-inks: the matched spans
// render ACCENTED, the rest DIM (accentMatches — the house search
// highlight; the cursor row stays the whole-row reversed accent, and an
// empty filter keeps the pre-search face: the current session's title
// accented, every meta dim).
//
// The picker opens in a LOADING state ("listing sessions…") — the app's
// ListSessions hop rides a tea.Cmd — and SetSessionPickerRows fills it;
// a listing failure never reaches here (the app closes the card and
// prints the static fallback instead). Clicks inside the card frame are
// swallowed (ClickRow) so nothing leaks to the worker-thread hit-map —
// the picker is keys-only.
package panels

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
)

// SessionPickRow — one pickable root session for the /session picker.
// Built app-side (sorted by Updated desc, Current marked, age rendered)
// so the panel stays a dumb renderer.
type SessionPickRow struct {
	ID       string // full opencode session id (the pick's payload)
	Title    string // display title (app fell back to ShortID when blank)
	Age      string // machine relative age ("45s", "3m", "2h", "5d", "now")
	Messages int    // GET /session/{id}/message count; -1 = unknown
	ShortID  string // short display form of ID
	Current  bool   // the office is attached to THIS session now
}

// sessionPickState — the open picker: the full row set (app-sorted), the
// narrowed view over it, the cursor, and the live filter buffer. loading
// marks the listing hop still in flight (rows empty).
type sessionPickState struct {
	rows     []SessionPickRow
	filtered []SessionPickRow
	sel      int
	filter   string
	loading  bool
}

// sessVisibleRows — the picker list window (same budget as the @ picker
// and the slash popover).
const sessVisibleRows = 8

// sessHint — the picker's dim footer.
const sessHint = "type: narrow · ↑/↓: move · enter: resume · esc: cancel"

// sessHigh builds the cursor row's reversed-accent run from the current
// palette. A package-level style would retain the boot theme after /theme.
func sessHigh() lipgloss.Style { return lipgloss.NewStyle().Foreground(chrome.Accent).Reverse(true) }

// SetSessionPickerHandlers wires the app's pick/cancel callbacks: enter
// on a row submits its full session id (→ the office re-anchors live),
// esc cancels (zero side effects — the card simply closes app-side).
func (c *Chat) SetSessionPickerHandlers(pick func(id string) tea.Cmd, cancel func() tea.Cmd) {
	c.onSessionPick, c.onSessionCancel = pick, cancel
}

// OpenSessionPicker opens the card in its LOADING state. The @ picker +
// slash popover close first (their nav/typing keys would fight the
// picker's); the cursor/filter start fresh on every open.
func (c *Chat) OpenSessionPicker() {
	c.closeAttachPicker()
	c.closeSlashPicker(true)
	c.sessPick = &sessionPickState{loading: true}
}

// SetSessionPickerRows fills the card with the app's built rows (sorted
// by Updated desc, the current session marked) and leaves the loading
// state. Refiltering preserves the app's order.
func (c *Chat) SetSessionPickerRows(rows []SessionPickRow) {
	if c.sessPick == nil {
		return
	}
	c.sessPick.rows = rows
	c.sessPick.loading = false
	c.sessRefilter()
}

// CloseSessionPicker closes the card (accept, esc-cancel, or the app's
// fallback path after a failed listing). Idempotent.
func (c *Chat) CloseSessionPicker() { c.sessPick = nil }

// SessionPickerOpen reports whether the picker card is open (loading or
// filled) — the app drops a late landing row set after an esc-cancel.
func (c *Chat) SessionPickerOpen() bool { return c.sessPick != nil }

// sessRefilter — the Chat-level wrapper every picker mutation funnels
// through; the work lives on the state (refilter) so the paste seam can
// reach it without a Chat receiver.
func (c *Chat) sessRefilter() { c.sessPick.refilter() }

// refilter recomputes the narrowed slice (case-insensitive substring
// over title + full session id) and clamps the cursor.
func (p *sessionPickState) refilter() {
	frag := strings.ToLower(strings.TrimSpace(p.filter))
	p.filtered = p.filtered[:0]
	for _, row := range p.rows {
		if frag == "" || strings.Contains(strings.ToLower(row.Title+" "+row.ID), frag) {
			p.filtered = append(p.filtered, row)
		}
	}
	if n := len(p.filtered); p.sel >= n {
		p.sel = n - 1
	}
	if p.sel < 0 {
		p.sel = 0
	}
}

// Paste — the office paste router's duck-typed seam (Paste(string)
// tea.Cmd — the /model picker's exact contract): a bracketed paste
// (cmd+v) lands in the FILTER, newline/CR runs flattened to single
// spaces (flattenPasteLines — the one-line-input paste rule), the list
// refilters, and the paste is fully consumed (nil cmd — it never sinks
// into the disabled textarea underneath). The chat panel's PasteMsg arm
// routes here while the card is open.
func (p *sessionPickState) Paste(content string) tea.Cmd {
	if p == nil {
		return nil
	}
	p.filter += flattenPasteLines(content)
	p.refilter()
	return nil
}

// accentMatches — the house filtered-row match highlight: every
// case-insensitive occurrence of frag in s renders ACCENTED, the rest
// DIM (a live query's rows read match-bright / context-dim). An empty
// (or whitespace) frag returns s untouched — the pre-search face stays
// plain. The /session picker's rows and the @ attach picker's paths both
// re-ink through this.
func accentMatches(s, frag string) string {
	f := strings.ToLower(strings.TrimSpace(frag))
	if f == "" || s == "" {
		return s
	}
	fr := []rune(f)
	runes := []rune(s)
	var out strings.Builder
	seg := 0
	for i := 0; i < len(runes); {
		// prefix-probe the lowered tail — the same semantics the
		// refilter's strings.Contains applies, so a span highlights IFF
		// it matched.
		if strings.HasPrefix(strings.ToLower(string(runes[i:])), f) {
			m := len(fr)
			if i+m > len(runes) {
				m = len(runes) - i
			}
			if seg < i {
				out.WriteString(chrome.DimText.Render(string(runes[seg:i])))
			}
			out.WriteString(chrome.AccentText.Render(string(runes[i : i+m])))
			i += m
			seg = i
			continue
		}
		i++
	}
	if seg < len(runes) {
		out.WriteString(chrome.DimText.Render(string(runes[seg:])))
	}
	return out.String()
}

// sessMove wraps the cursor by d rows through the narrowed list (the
// permission popover's exact wrap).
func (c *Chat) sessMove(d int) {
	if n := len(c.sessPick.filtered); n > 0 {
		c.sessPick.sel = (c.sessPick.sel + d + n) % n
	}
}

// sessCurrent is the highlighted row, ok=false when the list is empty
// (loading / no matches) — enter is a no-op then.
func (c *Chat) sessCurrent() (SessionPickRow, bool) {
	p := c.sessPick
	if p == nil || len(p.filtered) == 0 || p.sel < 0 || p.sel >= len(p.filtered) {
		return SessionPickRow{}, false
	}
	return p.filtered[p.sel], true
}

// sessKey handles EVERY key while the picker is open (claimed in
// Chat.Update after the question/permission floats — a parked turn's
// modal outranks browsing; with neither open the picker owns the input):
// typing narrows, ↑/↓/tab move, enter accepts, esc is TWO-STAGE (the
// first press clears a live filter, the second cancels — the /model
// picker's exact contract), pgup/pgdown scroll the transcript like the
// question modal's.
func (c *Chat) sessKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if c.sessPick.filter != "" {
			// stage one: clear the live filter and restore the full
			// list — the card stays open, onCancel stays silent.
			c.sessPick.filter = ""
			c.sessRefilter()
			return nil
		}
		if c.onSessionCancel != nil {
			return c.onSessionCancel()
		}
		return nil
	case "up":
		c.sessMove(-1)
		return nil
	case "down", "tab":
		c.sessMove(1)
		return nil
	case "enter":
		row, ok := c.sessCurrent()
		if !ok || c.onSessionPick == nil {
			return nil
		}
		return c.onSessionPick(row.ID)
	case "backspace":
		if r := []rune(c.sessPick.filter); len(r) > 0 {
			c.sessPick.filter = string(r[:len(r)-1])
			c.sessRefilter()
		}
		return nil
	case "ctrl+u":
		c.sessPick.filter = ""
		c.sessRefilter()
		return nil
	case "pgup", "pgdown":
		var cmd tea.Cmd
		c.vp, cmd = c.vp.Update(msg)
		if msg.String() == "pgdown" && c.vp.AtBottom() {
			c.follow = true
		} else {
			c.follow = false
		}
		return cmd
	default:
		if msg.Text != "" {
			c.sessPick.filter += msg.Text
			c.sessRefilter()
		}
		return nil
	}
}

// sessCard renders the picker's rows, each EXACTLY cardW display cells
// (rails included), so the View splice replaces whole cells without
// shifting a row. Geometry feeding the click swallow in ClickRow comes
// from the same builder, so a click can never disagree with the pixels.
func (c *Chat) sessCard() (rows []string, cardW int) {
	cardW = floatCardW(c.w)
	inner := cardW - 2 // the content column between the │ rails
	rail := func(s string) string {
		return chrome.AccentText.Render("│") + s + chrome.AccentText.Render("│")
	}
	blank := strings.Repeat(" ", inner)
	p := c.sessPick

	rows = append(rows, chrome.AccentText.Render("╭"+strings.Repeat("─", inner)+"╮"))
	rows = append(rows, rail(blank))

	// Title row: ACCENT-bold header left, dim "N/M" (narrowed/total)
	// badge right — the badge only shows once rows have landed.
	title := "SESSIONS"
	badge := ""
	if !p.loading && len(p.rows) > 0 {
		badge = itoa(len(p.filtered)) + "/" + itoa(len(p.rows))
	}
	gap := inner - 2 - lipgloss.Width(title) - lipgloss.Width(badge)
	if gap < 1 {
		gap = 1
	}
	rows = append(rows, rail(" "+chrome.AccentText.Bold(true).Render(title)+strings.Repeat(" ", gap)+
		chrome.DimText.Render(badge)+" "))

	// The live filter row: the typed narrowing echo (the textarea under
	// the card is disabled, so the card itself carries the buffer). Empty
	// filter renders as the dim invite; a non-empty one shows its caret.
	switch {
	case p.filter != "":
		rows = append(rows, rail(fitLabel(" filter: "+p.filter+questCaret, inner)))
	default:
		rows = append(rows, rail(fitLabel(chrome.DimText.Italic(true).Render(" type to narrow"), inner)))
	}
	rows = append(rows, rail(blank))

	// List window: the loading placeholder, the empty-filter/empty-server
	// row, or the narrowed rows themselves (windowed at sessVisibleRows).
	switch {
	case p.loading:
		rows = append(rows, rail(fitLabel(chrome.DimText.Render("  listing sessions…"), inner)))
	case len(p.filtered) == 0:
		rows = append(rows, rail(fitLabel(chrome.DimText.Render("  (no matches)"), inner)))
	default:
		start := 0
		if p.sel >= sessVisibleRows {
			start = p.sel - sessVisibleRows + 1
		}
		end := start + sessVisibleRows
		if end > len(p.filtered) {
			end = len(p.filtered)
		}
		for i := start; i < end; i++ {
			rows = append(rows, rail(sessMenuRow(p.filtered[i], i == p.sel, inner, p.filter)))
		}
	}
	rows = append(rows, rail(blank))

	// Footer hint — dim italic, wrapped to the content column.
	for _, ln := range strings.Split(strings.TrimRight(wrapPlain(sessHint, inner-2), "\n"), "\n") {
		rows = append(rows, rail(chrome.DimText.Italic(true).Render(" "+fitPlain(ln, inner-2))+" "))
	}
	rows = append(rows, chrome.AccentText.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return rows, cardW
}

// sessMenuRow renders one session row of EXACTLY inner cells: the cursor
// marker + title left (the CURRENT session's title also renders accented
// as its standing mark), the dim meta right ("<age> · <N msgs> · <short>"
// plus the " · current" suffix on the attached session), ANSI-aware-
// truncated (fitLabel, never rune-sliced). The highlighted row runs
// reversed-accent across the WHOLE content column. With a LIVE filter
// every non-cursor row re-inks instead: the matched spans of the fitted
// title + meta render accented, the rest dim (accentMatches) — accent
// then means "the match", so the current session's standing accent only
// shows on the unfiltered face.
func sessMenuRow(row SessionPickRow, on bool, inner int, frag string) string {
	mark := "  "
	if on {
		mark = "› "
	}
	msgs := itoa(row.Messages) + " msgs"
	if row.Messages < 0 {
		msgs = "? msgs"
	}
	meta := row.Age + " · " + msgs + " · " + row.ShortID
	if row.Current {
		meta += " · current"
	}
	// the title takes what's left after the meta column (one-cell gap min)
	titleW := inner - lipgloss.Width(mark) - lipgloss.Width(meta) - 1
	if titleW < 8 {
		titleW = 8
	}
	title := fitPlain(row.Title, titleW)
	body := mark + title + " " + meta
	if on {
		return sessHigh().Render(fitLabel(body, inner))
	}
	if strings.TrimSpace(frag) != "" {
		return fitLabel(mark+accentMatches(title, frag)+" "+accentMatches(meta, frag), inner)
	}
	out := mark
	if row.Current {
		out += chrome.AccentText.Render(title)
	} else {
		out += title
	}
	out += " " + chrome.DimText.Render(meta)
	return fitLabel(out, inner)
}

// sessCardGeom — where the picker sits, in CHAT CONTENT COORDS (row 0 =
// the viewport's first rendered row): centered over the WHOLE panel on
// both axes, fixed regardless of how the transcript scrolls (perm's and
// quest's identical contract).
func (c *Chat) sessCardGeom() (top, left, cardW int, rows []string) {
	rows, cardW = c.sessCard()
	top = (c.h - len(rows)) / 2
	if top < 0 {
		top = 0 // a very short panel pins the card to the top edge
	}
	left = (c.w - cardW) / 2
	if left < 0 {
		left = 0
	}
	return top, left, cardW, rows
}

// sessOverlay splices the picker's rows over the assembled background
// lines cell-wise (ANSI-aware): the background pixels outside the card
// survive, and the row count never changes, so the layout never jumps
// when the picker opens. Runs FIRST of the three floats — a permission
// or question card popping over it splices on top (the parked turn's
// modal outranks browsing).
func (c *Chat) sessOverlay(bg []string) []string {
	top, left, _, rows := c.sessCardGeom()
	for i, row := range rows {
		y := top + i
		if y < 0 || y >= len(bg) {
			continue // a short panel clips the card instead of growing
		}
		bg[y] = permSplice(bg[y], left, row, lipgloss.Width(row), c.w)
	}
	return bg
}
