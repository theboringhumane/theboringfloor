// chat_paste.go — the chat textarea's PASTE surface: fast batched insert +
// large-paste collapse + chip expand-on-send.
//
// SPEED: a terminal paste arrives as ONE tea.PasteMsg and the textarea's
// own arm inserts the whole run in a single op (insertRunesFromUserInput)
// — the ~530ms/key synchronous drain only ever applies to TYPED keys,
// never to a paste. Small pastes therefore insert LITERALLY, cursor-
// correctly, in one shot.
//
// COLLAPSE: a paste spanning MORE than pasteChipMaxLines lines OR more
// than pasteChipMaxChars chars would bury the draft (and the member's
// in-progress sentence) under a wall of quoted text. It collapses to a
// single visible chip row — "[pasted N lines · M chars]" — which:
//
//   - renders as ONE line in the textarea (the draft stays scannable);
//   - deletes as ONE UNIT on backspace (never per-rune into the middle);
//   - EXPANDS back to the full original text on send — the member sees
//     the chip, the agent receives everything.
//
// Chips are tracked in insertion order (c.pasteChips); a token the member
// edited away by hand simply never matches — its chip record is dropped
// silently at expand time and the edited text sends literally.
package panels

import (
	"strings"
)

// pasteChip — one collapsed large paste: the draft holds the one-line
// token, full keeps the original content for expand-on-send.
type pasteChip struct {
	token string // the visible "[pasted N lines · M chars]" run in the draft
	full  string // the original paste content, restored verbatim on send
}

// The collapse thresholds: a paste collapses when it spans MORE than
// pasteChipMaxLines lines (a 21st line tips it) or its rune count exceeds
// pasteChipMaxChars. Frozen — pinned by chat_paste_test.go.
const (
	pasteChipMaxLines = 20
	pasteChipMaxChars = 2000
)

// pasteLines counts the content's display lines (1-based: "a\nb" is 2).
func pasteLines(s string) int { return strings.Count(s, "\n") + 1 }

// pasteChipThreshold reports whether a paste collapses into a chip.
func pasteChipThreshold(content string) bool {
	return pasteLines(content) > pasteChipMaxLines || len([]rune(content)) > pasteChipMaxChars
}

// pasteChipToken renders the chip's visible one-line form. Singulars
// collapse correctly ("1 line", "1 char") — the chip never reads "1 lines".
func pasteChipToken(content string) string {
	lines, chars := pasteLines(content), len([]rune(content))
	ls, cs := "lines", "chars"
	if lines == 1 {
		ls = "line"
	}
	if chars == 1 {
		cs = "char"
	}
	return "[pasted " + itoa(lines) + " " + ls + " · " + itoa(chars) + " " + cs + "]"
}

// insertPasteChip collapses one large paste into the draft: the chip
// token inserts at the cursor (the textarea's own batched insert — one
// op, no drain), the full text rides the chip record for expand-on-send.
func (c *Chat) insertPasteChip(content string) {
	token := pasteChipToken(content)
	c.pasteChips = append(c.pasteChips, pasteChip{token: token, full: content})
	c.ta.InsertString(token)
}

// expandPasteChips swaps every live chip token in the SUBMITTED text for
// its original paste content — left to right, one occurrence per chip in
// insertion order (two identical tokens expand in the order they were
// pasted). A token the member edited away just drops its chip record.
func (c *Chat) expandPasteChips(text string) string {
	if len(c.pasteChips) == 0 {
		return text
	}
	kept := c.pasteChips[:0]
	for _, chip := range c.pasteChips {
		if i := strings.Index(text, chip.token); i >= 0 {
			text = text[:i] + chip.full + text[i+len(chip.token):]
			continue
		}
		kept = append(kept, chip) // token edited away — keep the record honest
	}
	c.pasteChips = kept
	return text
}

// popPasteChipBeforeCursor — the chip's one-unit backspace: when the
// draft text immediately LEFT of the cursor ends with a live chip token,
// the whole token goes in ONE op (SetValue + exact cursor restore), the
// chip record drops, and the key is spent. Every other backspace shape
// (mid-text, after ordinary text, empty draft) returns false so the key
// falls through to its normal arm.
func (c *Chat) popPasteChipBeforeCursor() bool {
	if len(c.pasteChips) == 0 {
		return false
	}
	val := c.ta.Value()
	row, col := c.ta.Line(), c.ta.Column()
	lines := strings.Split(val, "\n")
	if row >= len(lines) {
		return false
	}
	// the cursor's rune offset in the joined draft
	off := 0
	for i := 0; i < row; i++ {
		off += len([]rune(lines[i])) + 1 // +1: the joining newline
	}
	off += col
	r := []rune(val)
	if off > len(r) {
		return false
	}
	before := string(r[:off])
	// newest chip first: with two identical tokens the one just pasted is
	// the one under the cursor.
	for i := len(c.pasteChips) - 1; i >= 0; i-- {
		token := c.pasteChips[i].token
		if !strings.HasSuffix(before, token) {
			continue
		}
		newBefore := string(r[:off-len([]rune(token))])
		c.pasteChips = append(c.pasteChips[:i], c.pasteChips[i+1:]...)
		c.ta.SetValue(newBefore + string(r[off:]))
		// SetValue parks the cursor at the end of the text — walk it back
		// to the deletion point (row = newlines in newBefore, col = runes
		// after the last one; the token never spans rows itself).
		targetRow := strings.Count(newBefore, "\n")
		for up := c.ta.LineCount() - 1 - targetRow; up > 0; up-- {
			c.ta.CursorUp()
		}
		lastNl := strings.LastIndex(newBefore, "\n")
		c.ta.SetCursorColumn(len([]rune(newBefore[lastNl+1:])))
		return true
	}
	return false
}

// flattenPasteLines folds a multi-line paste for a ONE-LINE input (the
// question popover's custom-answer row, the /session picker's filter):
// every newline run becomes a single space. The TEXT question page does
// NOT use this — it preserves the paste's newlines verbatim.
func flattenPasteLines(content string) string {
	return strings.Join(strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == '\r'
	}), " ")
}
