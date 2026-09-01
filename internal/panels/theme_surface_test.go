package panels

import (
	"image/color"
	"testing"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
)

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestTextareaOwnsContinuousPanelSurface(t *testing.T) {
	before := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(before) })

	for _, name := range []string{"paper", "noir"} {
		t.Run(name, func(t *testing.T) {
			if !chrome.SetTheme(name) {
				t.Fatalf("SetTheme(%q) returned false", name)
			}
			ta := textarea.New()
			applyTextareaStyles(&ta)
			styles := ta.Styles()
			for label, style := range map[string]lipgloss.Style{
				"focused base":        styles.Focused.Base,
				"focused prompt":      styles.Focused.Prompt,
				"focused placeholder": styles.Focused.Placeholder,
				"focused cursor":      styles.Focused.CursorLine,
				"focused text":        styles.Focused.Text,
				"focused tail":        styles.Focused.EndOfBuffer,
				"blurred base":        styles.Blurred.Base,
				"blurred prompt":      styles.Blurred.Prompt,
				"blurred placeholder": styles.Blurred.Placeholder,
				"blurred cursor":      styles.Blurred.CursorLine,
				"blurred text":        styles.Blurred.Text,
				"blurred tail":        styles.Blurred.EndOfBuffer,
			} {
				if !sameColor(style.GetBackground(), chrome.PanelBgColor) {
					t.Errorf("%s background = %#v, want PanelBg %#v", label, style.GetBackground(), chrome.PanelBgColor)
				}
			}
		})
	}
}

func TestThemeSwitchRebuildsPickerHighlights(t *testing.T) {
	before := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(before) })

	if !chrome.SetTheme("paper") {
		t.Fatal("SetTheme(paper) returned false")
	}
	paperAccent, paperQuestion := chrome.Accent, chrome.Question
	if !chrome.SetTheme("noir") {
		t.Fatal("SetTheme(noir) returned false")
	}
	if sameColor(paperAccent, chrome.Accent) || sameColor(paperQuestion, chrome.Question) {
		t.Fatal("test themes need distinct highlight colors")
	}
	for label, check := range map[string]struct {
		style lipgloss.Style
		want  color.Color
	}{
		"question": {questHigh(), chrome.Question},
		"link":     {linkHigh(), chrome.Accent},
		"model":    {modelPickHigh(), chrome.Accent},
		"session":  {sessHigh(), chrome.Accent},
	} {
		if !sameColor(check.style.GetForeground(), check.want) {
			t.Errorf("%s highlight retained the old theme foreground", label)
		}
	}
}
