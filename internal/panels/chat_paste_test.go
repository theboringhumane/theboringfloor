// chat_paste_test.go — the chat textarea's paste surface (chat_paste.go)
// plus the question popover's paste field (question_modal.go's
// questPaste). Pinned rules:
//
//	(a) SPEED: a paste is ONE tea.PasteMsg → ONE batched insert — small
//	    pastes land literally (never a per-rune drain);
//	(b) COLLAPSE: >20 lines OR >2000 chars folds to the one-line chip
//	    "[pasted N lines · M chars]" — the full text never touches the
//	    draft;
//	(c) EXPAND-ON-SEND: Enter restores the full original text (the
//	    member sees the chip, the agent gets everything); chips clear
//	    with the draft;
//	(d) backspace deletes a chip as ONE UNIT — whole token, cursor
//	    restored, ordinary backspace shapes untouched;
//	(e) shift+enter AND ctrl+j both insert a newline while enter still
//	    sends (shift+enter needs a kitty-protocol terminal — ghostty/
//	    kitty; ctrl+j is the universal fallback);
//	(f) the QUESTION popover's answer field takes the paste batched —
//	    the TEXT page verbatim (newlines kept), the 1-line custom-
//	    answer row flattened, confirm pages ignored — and the paste
//	    NEVER falls through to the disabled main textarea.
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// bigPaste builds a paste of n numbered lines ("lorem NN" — short, so
// nothing soft-wraps) and returns it plus its line/char counts.
func bigPaste(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "lorem " + itoa(i+1)
	}
	return strings.Join(lines, "\n")
}

// sendCapture builds a chat panel whose onSend records every text.
func sendCapture() (*Chat, *[]string) {
	sent := &[]string{}
	c := NewChat(func(text string, atts []state.Attachment) tea.Cmd {
		*sent = append(*sent, text)
		return nil
	})
	c.SetSize(60, 30)
	return c, sent
}

func typeKeys(c *Chat, s string) {
	for _, r := range s {
		c.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

// --- (a)+(b) thresholds & literal-vs-chip -----------------------------------

func TestPasteChipThreshold(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    bool
	}{
		{"one line", "hello paste", false},
		{"20 lines", bigPaste(20), false},
		{"21 lines", bigPaste(21), true},
		{"2000 chars", strings.Repeat("x", 2000), false},
		{"2001 chars", strings.Repeat("x", 2001), true},
		{"few lines but huge", strings.Repeat("ab ", 1000), true},
	} {
		if got := pasteChipThreshold(tc.content); got != tc.want {
			t.Fatalf("%s: threshold = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestChatPasteSmallLiteral(t *testing.T) {
	c, sent := sendCapture()
	typeKeys(c, "pre: ")
	c.Update(tea.PasteMsg{Content: "small paste\nwith three\nlines"})
	if got := c.ta.Value(); got != "pre: small paste\nwith three\nlines" {
		t.Fatalf("a small paste inserts LITERALLY (newlines kept), got %q", got)
	}
	if len(c.pasteChips) != 0 {
		t.Fatalf("a small paste stages no chip, got %+v", c.pasteChips)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*sent) != 1 || (*sent)[0] != "pre: small paste\nwith three\nlines" {
		t.Fatalf("the small paste sends verbatim, got %+v", *sent)
	}
}

func TestChatPasteChipCollapse(t *testing.T) {
	c, _ := sendCapture()
	full := bigPaste(25)
	c.Update(tea.PasteMsg{Content: full})
	token := pasteChipToken(full)
	if token != "[pasted 25 lines · 215 chars]" {
		t.Fatalf("chip token = %q, want %q", token, "[pasted 25 lines · 215 chars]")
	}
	if got := c.ta.Value(); got != token {
		t.Fatalf("the draft holds ONLY the chip row, got %q", got)
	}
	if strings.Contains(c.View(), "lorem 13") {
		t.Fatal("the collapsed paste's body must not paint in the panel")
	}
	if len(c.pasteChips) != 1 || c.pasteChips[0].full != full {
		t.Fatalf("the chip retains the FULL original text, got %+v", c.pasteChips)
	}
}

// --- (c) expand-on-send ------------------------------------------------------

func TestChatPasteChipExpandOnSend(t *testing.T) {
	c, sent := sendCapture()
	typeKeys(c, "see ")
	full := bigPaste(25)
	c.Update(tea.PasteMsg{Content: full})
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	want := "see " + full
	if len(*sent) != 1 || (*sent)[0] != want {
		t.Fatalf("expand-on-send: the agent gets the FULL paste, got %q", (*sent))
	}
	if c.ta.Value() != "" || len(c.pasteChips) != 0 {
		t.Fatalf("send resets draft AND chips, draft=%q chips=%+v", c.ta.Value(), c.pasteChips)
	}
}

func TestChatPasteChipExpandTwoIdenticalTokens(t *testing.T) {
	c, sent := sendCapture()
	a := bigPaste(25)
	b := strings.ReplaceAll(bigPaste(25), "lorem", "ipsum") // same counts, different words
	c.Update(tea.PasteMsg{Content: a})
	c.Update(tea.PasteMsg{Content: b})
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*sent) != 1 || (*sent)[0] != a+b {
		t.Fatalf("two identical tokens expand in insertion order, got %q", (*sent))
	}
}

// --- (d) backspace one-unit --------------------------------------------------

func TestChatPasteChipBackspaceUnit(t *testing.T) {
	c, _ := sendCapture()
	c.Update(tea.PasteMsg{Content: bigPaste(25)})
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if got := c.ta.Value(); got != "" {
		t.Fatalf("ONE backspace eats the WHOLE chip, draft %q", got)
	}
	if len(c.pasteChips) != 0 {
		t.Fatalf("the chip record drops with the token, %+v", c.pasteChips)
	}
}

func TestChatPasteChipBackspaceKeepsPrefix(t *testing.T) {
	c, _ := sendCapture()
	typeKeys(c, "hi ")
	c.Update(tea.PasteMsg{Content: bigPaste(25)})
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if got := c.ta.Value(); got != "hi " {
		t.Fatalf("backspace eats only the chip — the typed prefix survives, draft %q", got)
	}
}

func TestChatPasteChipBackspaceAfterSuffixTypes(t *testing.T) {
	c, _ := sendCapture()
	full := bigPaste(25)
	c.Update(tea.PasteMsg{Content: full})
	typeKeys(c, "x")
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	// the trailing "x" eats the backspace — the chip stays intact.
	if got := c.ta.Value(); got != pasteChipToken(full) {
		t.Fatalf("backspace after ordinary text never bites the chip, draft %q", got)
	}
	if len(c.pasteChips) != 1 {
		t.Fatalf("the chip record survives, %+v", c.pasteChips)
	}
}

// --- (e) newline keys --------------------------------------------------------

func TestChatShiftEnterCtrlJNewline(t *testing.T) {
	c, sent := sendCapture()
	typeKeys(c, "a")
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModShift})) // ghostty/kitty deliver this
	typeKeys(c, "b")
	c.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl})) // the universal fallback
	typeKeys(c, "c")
	if got := c.ta.Value(); got != "a\nb\nc" {
		t.Fatalf("shift+enter AND ctrl+j both insert a newline, draft %q", got)
	}
	if len(*sent) != 0 {
		t.Fatalf("newline keys never SEND, got %+v", *sent)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*sent) != 1 || (*sent)[0] != "a\nb\nc" {
		t.Fatalf("enter still SENDS the multi-line draft, got %+v", *sent)
	}
}

// --- (f) the question popover's paste field ----------------------------------

func TestQuestTextPagePaste(t *testing.T) {
	c, answers, _ := questHarness(&QuestionView{
		ID: "que-p", Kind: QuestionKindText, Index: 1, Total: 1,
		Question: "paste the stack trace?",
	})
	c.Update(tea.PasteMsg{Content: "goroutine 1 [running]:\nmain.main()\n\t/app/main.go:42"})
	if c.qText != "goroutine 1 [running]:\nmain.main()\n\t/app/main.go:42" {
		t.Fatalf("the TEXT page takes the paste VERBATIM (batched), got %q", c.qText)
	}
	if got := c.ta.Value(); got != "" {
		t.Fatalf("the paste must NEVER fall into the disabled main textarea, got %q", got)
	}
	// the echo box paints the pasted tail (multi-line preserved)
	if view := c.View(); !strings.Contains(view, "main.main()") {
		t.Fatalf("the echo box shows the pasted text:\n%s", view)
	}
	// submit/cancel keys unchanged: ctrl+enter ships the multi-line text
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}))
	if len(*answers) != 1 || (*answers)[0].Text != c.qText {
		t.Fatalf("ctrl+enter submits the pasted text, got %+v", *answers)
	}
}

func TestQuestCustomRowPaste(t *testing.T) {
	c, answers, _ := questHarness(&QuestionView{
		ID: "que-r", Kind: QuestionKindRadio, Index: 1, Total: 1,
		Question: "which branch?",
		Options:  []state.QuestionOption{{Label: "main"}, {Label: "release"}},
	})
	// cursor on an OPTION row: the paste types nowhere (typing's rule).
	c.Update(tea.PasteMsg{Content: "nope"})
	if c.qText != "" {
		t.Fatalf("option rows take no text — paste included, got %q", c.qText)
	}
	// walk to the custom-answer row: a multi-line paste flattens to one.
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	c.Update(tea.PasteMsg{Content: "hotfix/one\ntwo\nthree"})
	if c.qText != "hotfix/one two three" {
		t.Fatalf("the 1-line custom row flattens the paste, got %q", c.qText)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*answers) != 1 || (*answers)[0].Text != "hotfix/one two three" {
		t.Fatalf("enter submits the pasted custom answer, got %+v", *answers)
	}
}

func TestQuestConfirmPasteIgnored(t *testing.T) {
	c, answers, _ := questHarness(&QuestionView{
		ID: "que-c", Kind: QuestionKindConfirm, Index: 1, Total: 1,
		Question: "ship it?",
		Options:  []state.QuestionOption{{Label: "yes"}, {Label: "no"}},
	})
	c.Update(tea.PasteMsg{Content: "why not"})
	if c.qText != "" {
		t.Fatalf("the confirm page has no text surface, got %q", c.qText)
	}
	if len(*answers) != 0 {
		t.Fatalf("a paste never answers, got %+v", *answers)
	}
}
