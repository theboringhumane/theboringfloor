// session_picker_search_test.go — proofs for the /session picker's
// harmonized house-search contract (the /model picker's wave-87 shape,
// closing the last picker-UX follow-up):
//
//	(a) a live filter re-inks every NON-CURSOR row: the matched spans
//	    render ACCENTED, the rest DIM (accentMatches) — case-insensitive,
//	    over the fitted title AND the meta; the cursor row keeps its
//	    whole-row reversed accent and an empty filter keeps the
//	    pre-search face (current session's title accented, meta dim);
//	(b) esc is TWO-STAGE: the first press clears a live filter (the
//	    list restores, onSessionCancel silent), the second cancels;
//	(c) Paste — the office paste router's duck-typed seam — flattens
//	    newline/CR runs to single spaces, appends to the filter,
//	    refilters, returns nil, and the chat panel's REAL PasteMsg arm
//	    routes through it (never the disabled textarea).
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
)

// (a-unit) the row renderer: match bright/accent, rest dim — over title
// AND meta, case-insensitive; cursor row and empty frag unchanged.
func TestSessPickerSearchHighlightRow(t *testing.T) {
	row := SessionPickRow{ID: "ses-alpha-1", Title: "alpha brief", Age: "3m", Messages: 12, ShortID: "ses-alph"}

	// mid-title match: leading context dims, the span pops accent.
	got := sessMenuRow(row, false, 60, "brief")
	for _, want := range []string{
		chrome.DimText.Render("alpha "),
		chrome.AccentText.Render("brief"),
		chrome.DimText.Render("3m · 12 msgs · ses-alph"), // no meta match: all dim
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("a filtered row must render match accent / rest dim — missing %q:\n%q", want, got)
		}
	}
	// the stripped text is the same row as before — only the ink changed.
	if stripped := ansi.Strip(got); !strings.Contains(stripped, "alpha brief") || !strings.Contains(stripped, "3m · 12 msgs · ses-alph") {
		t.Fatalf("highlighting must never change the row's text:\n%q", stripped)
	}

	// case-insensitive: an UPPERCASE query accents the lowercase span.
	got = sessMenuRow(row, false, 60, "ALPHA")
	if !strings.Contains(got, chrome.AccentText.Render("alpha")) {
		t.Fatalf("the match highlight must be case-insensitive:\n%q", got)
	}

	// a meta-span match highlights too (the short id lives in the meta).
	got = sessMenuRow(row, false, 60, "ses-alph")
	if !strings.Contains(got, chrome.AccentText.Render("ses-alph")) {
		t.Fatalf("a meta match must highlight in the meta column:\n%q", got)
	}

	// the cursor row under a live filter is UNCHANGED: the whole-row
	// reversed accent IS its highlight.
	body := "› " + fitPlain("alpha brief", 34) + " 3m · 12 msgs · ses-alph"
	if got, want := sessMenuRow(row, true, 60, "brief"), sessHigh().Render(fitLabel(body, 60)); got != want {
		t.Fatalf("the cursor row must keep its whole-row reversed accent:\ngot  %q\nwant %q", got, want)
	}

	// empty filter: the pre-search face — plain title, dim meta, NO
	// match spans (accent means the current session's standing mark only).
	got = sessMenuRow(row, false, 60, "")
	if strings.Contains(got, chrome.AccentText.Render("brief")) || strings.Contains(got, chrome.DimText.Render("alpha ")) {
		t.Fatalf("an empty filter must not re-ink the row:\n%q", got)
	}
	if !strings.Contains(got, "alpha brief") || !strings.Contains(got, chrome.DimText.Render("3m · 12 msgs · ses-alph")) {
		t.Fatalf("the unfiltered row must keep its classic face:\n%q", got)
	}
	// the CURRENT session's standing accent still shows unfiltered (the
	// accent wraps the FITTED title, padding included).
	cur := SessionPickRow{ID: "ses-beta-22", Title: "beta review", Age: "2h", Messages: 47, ShortID: "ses-beta", Current: true}
	if got := sessMenuRow(cur, false, 60, ""); !strings.Contains(got, chrome.AccentText.Render(fitPlain("beta review", 24))) {
		t.Fatalf("the current session's title keeps its standing accent unfiltered:\n%q", got)
	}
}

// (a-card) the same spans ride the rendered card mid-filter.
func TestSessPickerSearchHighlightCard(t *testing.T) {
	c, _, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())
	sessType(c, "a") // every title matches; cursor stays on row 0

	rows, _ := c.sessCard()
	joined := strings.Join(rows, "\n")
	// the SECOND row (beta review, non-cursor) re-inks: Dim("bet") +
	// Acc("a") adjacent (the trailing dim span absorbs the fitted
	// padding, so assert the contiguous match pair); the THIRD row
	// (gamma notes) re-inks Dim("g")+Acc("a")+Dim("mm")+Acc("a").
	for _, want := range []string{
		chrome.DimText.Render("bet") + chrome.AccentText.Render("a"),
		chrome.DimText.Render("g") + chrome.AccentText.Render("a") + chrome.DimText.Render("mm"),
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the card's filtered rows must carry the accent/dim match spans — missing %q:\n%q", want, joined)
		}
	}
	// the filter row + badge keep their house copy.
	stripped := ansi.Strip(joined)
	for _, w := range []string{"filter: a", "3/3", "beta review"} {
		if !strings.Contains(stripped, w) {
			t.Fatalf("the mid-filter card must show %q:\n%s", w, stripped)
		}
	}
}

// (b) esc is TWO-STAGE: the first press clears the live filter (the list
// restores, onSessionCancel silent), the second cancels.
func TestSessPickerSearchEscTwoStage(t *testing.T) {
	c, picks, cancels := sessHarness()
	c.SetSessionPickerRows(sessRows())
	sessType(c, "alpha")
	if n := len(c.sessPick.filtered); n != 1 {
		t.Fatalf("precondition: the filter narrows to one row, got %d", n)
	}

	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 0 {
		t.Fatalf("the first esc must clear the filter, NOT cancel (cancels=%d)", *cancels)
	}
	if got := c.sessPick.filter; got != "" {
		t.Fatalf("the first esc must clear the filter, got %q", got)
	}
	if n := len(c.sessPick.filtered); n != 3 {
		t.Fatalf("the first esc must restore the full list, got %d rows", n)
	}
	if view := ansi.Strip(c.View()); !strings.Contains(view, "gamma notes") || !strings.Contains(view, "type to narrow") {
		t.Fatalf("the restored card must show every row + the empty invite:\n%s", view)
	}

	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if *cancels != 1 {
		t.Fatalf("the second esc (empty filter) must cancel exactly once, got %d", *cancels)
	}
	if len(*picks) != 0 {
		t.Fatalf("esc never picks, got %v", *picks)
	}
}

// (c) Paste — the office paste router's duck-typed seam: newline/CR runs
// flatten to single spaces, the text appends to the filter, the list
// refilters, the cmd is nil — and the chat panel's REAL PasteMsg arm
// routes through it, never the disabled textarea.
func TestSessPickerSearchPaste(t *testing.T) {
	c, picks, _ := sessHarness()
	c.SetSessionPickerRows(sessRows())

	// the REAL router: the panel's PasteMsg arm calls the seam (cmd+v
	// mid-filter — newlines flatten to single spaces).
	c.Update(tea.PasteMsg{Content: "beta\r\nreview"})
	if got := c.sessPick.filter; got != "beta review" {
		t.Fatalf("the paste arm must flatten newline runs into the filter, got %q", got)
	}
	if n := len(c.sessPick.filtered); n != 1 {
		t.Fatalf("the pasted filter must narrow to beta review, got %d rows", n)
	}
	if got := c.ta.Value(); got != "" {
		t.Fatalf("a picker paste must never sink into the disabled textarea, got %q", got)
	}
	// enter accepts the paste-narrowed row.
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "ses-beta-22" {
		t.Fatalf("enter must accept the paste-narrowed row, got %v", *picks)
	}

	// the seam itself: appends, flattens, refilters, consumes (nil cmd).
	if cmd := c.sessPick.Paste(" x"); cmd != nil {
		t.Fatalf("Paste must consume the paste (nil cmd), got %v", cmd)
	}
	if got := c.sessPick.filter; got != "beta review x" {
		t.Fatalf("a second paste must APPEND, got %q", got)
	}
	if n := len(c.sessPick.filtered); n != 0 {
		t.Fatalf("'beta review x' must dead-end, got %d rows", n)
	}
	if view := ansi.Strip(c.View()); !strings.Contains(view, "(no matches)") {
		t.Fatalf("the dead pasted filter must render (no matches):\n%s", view)
	}

	// a nil picker state (the closed seam) swallows a paste safely.
	var nilPick *sessionPickState
	if cmd := nilPick.Paste("anything"); cmd != nil {
		t.Fatalf("a nil picker state must swallow a paste, got %v", cmd)
	}
}
