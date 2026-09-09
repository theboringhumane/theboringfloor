package panels

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"strings"
)

type workField struct {
	label   string
	input   textinput.Model
	choices []string
	pick    int
}
type workForm struct {
	title  string
	fields []workField
	focus  int
	err    string
}

func field(label, value string) workField {
	i := textinput.New()
	i.SetValue(value)
	i.CharLimit = 2000
	return workField{label: label, input: i}
}
func choice(label string, values []string, pick int) workField {
	f := field(label, "")
	f.choices = values
	f.pick = pick
	return f
}
func form(title string, fields ...workField) *workForm {
	f := &workForm{title: title, fields: fields}
	f.fields[0].input.Focus()
	return f
}
func (f *workForm) value(i int) string {
	v := f.fields[i]
	if len(v.choices) > 0 {
		return v.choices[v.pick]
	}
	return strings.TrimSpace(v.input.Value())
}
func (f *workForm) update(msg tea.Msg) (save, cancel bool) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc":
			return false, true
		case "ctrl+s":
			return true, false
		case "tab", "down":
			f.fields[f.focus].input.Blur()
			f.focus = (f.focus + 1) % len(f.fields)
			f.fields[f.focus].input.Focus()
			return
		case "shift+tab", "up":
			f.fields[f.focus].input.Blur()
			f.focus = (f.focus + len(f.fields) - 1) % len(f.fields)
			f.fields[f.focus].input.Focus()
			return
		case "enter":
			if f.focus == len(f.fields)-1 {
				return true, false
			}
			f.fields[f.focus].input.Blur()
			f.focus++
			f.fields[f.focus].input.Focus()
			return
		case "left", "right", " ":
			v := &f.fields[f.focus]
			if len(v.choices) > 0 {
				delta := 1
				if k.String() == "left" {
					delta = -1
				}
				v.pick = (v.pick + len(v.choices) + delta) % len(v.choices)
				return
			}
		}
	}
	v := &f.fields[f.focus]
	if len(v.choices) == 0 {
		v.input, _ = v.input.Update(msg)
	}
	return
}
func (f *workForm) view(w, h int) string {
	var rows []string
	rows = append(rows, "", chrome.PanelHeader.Render(f.title), chrome.PanelDim.Render("Tab next field · ← → choose · Ctrl+S save · Esc cancel"), "")
	if f.err != "" {
		rows = append(rows, lipgloss.NewStyle().Foreground(chrome.Warn).Render(f.err))
	}
	first := max(0, f.focus-max(0, (h-len(rows))/3-1))
	for i := first; i < len(f.fields); i++ {
		v := f.fields[i]
		prefix := "  "
		if i == f.focus {
			prefix = "› "
		}
		rows = append(rows, chrome.PanelDim.Render(prefix+v.label))
		var val string
		if len(v.choices) > 0 {
			var parts []string
			for j, c := range v.choices {
				if j == v.pick {
					parts = append(parts, chrome.TabActive.Render(" "+c+" "))
				} else {
					parts = append(parts, chrome.PanelDim.Render(" "+c+" "))
				}
			}
			val = strings.Join(parts, " ")
			if ansi.StringWidth(val) > w-4 {
				val = "← " + chrome.TabActive.Render(" "+v.choices[v.pick]+" ") + " →"
			}
		} else {
			v.input.SetWidth(max(10, w-8))
			val = v.input.View()
		}
		rows = append(rows, "  "+val, "")
	}
	return workFit(strings.Join(rows, "\n"), w, h)
}
func workFit(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, max(1, w), "…")
	}
	return fit(strings.Join(lines, "\n"), max(1, h))
}
func workBox(s string, w, h int) string {
	return lipgloss.NewStyle().Width(max(1, w)).Height(max(1, h)).Render(workFit(s, w, h))
}

func IsWorkspaceResult(msg tea.Msg) bool {
	switch msg.(type) {
	case floorsLoaded, filesLoaded, filePreview, ticketsLoaded:
		return true
	}
	return false
}

func workRule(h int) string {
	return " " + strings.TrimSuffix(strings.Repeat("│ \n ", max(1, h)), "\n ")
}
