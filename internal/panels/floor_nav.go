package panels

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
)

type floorNavRow struct {
	text                string
	floor, conversation int
	action              string
}

// Shared row geometry keeps the persistent navigator and its mouse targets
// aligned, including scrolling a long list of projects or conversations.
func (f *Floors) navRows(w, h int, focused bool) []floorNavRow {
	var rows []floorNavRow
	add := func(text string) { rows = append(rows, floorNavRow{text: text, floor: -1, conversation: -1}) }
	heading := " FLOORS"
	if focused {
		heading += " ◂"
	}
	add(chrome.PanelHeader.Render(heading))
	add(chrome.PanelDim.Render(" Ctrl+E · projects"))
	add("")
	count := max(1, min(len(f.rows), (h-14)/2))
	start := max(0, f.selected-count+1)
	for i := start; i < min(len(f.rows), start+count); i++ {
		r := f.rows[i]
		mark := "  "
		if r.Dir == f.dir {
			mark = "● "
		}
		label := ansi.Truncate(mark+r.Name, max(1, w-3), "…")
		if i == f.selected {
			label = chrome.TabActive.Render(label)
		} else {
			label = chrome.PanelHeader.Render(label)
		}
		rows = append(rows, floorNavRow{text: " " + label, floor: i, conversation: -1})
		add(chrome.PanelDim.Render(fmt.Sprintf("   %d teams · %d tickets", len(r.Teams), len(r.Tickets))))
	}
	rows = append(rows, floorNavRow{text: chrome.PanelAccent.Render(" + Add project"), floor: -1, conversation: -1, action: "a"})
	add("")
	row := f.Current()
	add(chrome.PanelHeader.Render(" TEAMS"))
	var names []string
	for _, team := range row.Teams {
		names = append(names, team.Name)
	}
	for _, line := range strings.Split(ansi.Wrap(strings.Join(names, " · "), max(1, w-3), ""), "\n") {
		add(chrome.PanelDim.Render(" " + line))
	}
	add("")
	add(chrome.PanelHeader.Render(" CONVERSATIONS"))
	rows = append(rows, floorNavRow{text: chrome.PanelAccent.Render(" + New conversation"), floor: -1, conversation: -1, action: "n"})
	cs := f.conversations[row.Dir]
	available := max(0, (h-len(rows)-3)/2)
	start = max(0, f.session-available+1)
	for i := start; i < min(len(cs), start+available); i++ {
		c := cs[i]
		label := " " + ansi.Truncate(c.Title, max(1, w-4), "…")
		if i == f.session {
			label = chrome.PanelAccent.Render("›" + label)
		} else {
			label = " " + label
		}
		rows = append(rows, floorNavRow{text: label, floor: -1, conversation: i})
		add(chrome.PanelDim.Render("   " + c.Backend))
	}
	if len(cs) == 0 {
		add(chrome.PanelDim.Render(" No conversations yet"))
	}
	return rows
}

func (f *Floors) NavView(w, h int, focused bool) string {
	rows := f.navRows(w, h, focused)
	var lines []string
	for _, row := range rows {
		lines = append(lines, row.text)
	}
	body := workFit(strings.Join(lines, "\n"), w-1, max(1, h-2))
	foot := " n new · t team"
	if focused {
		foot = " Enter open · Esc back"
	}
	if f.err != "" {
		foot = f.err
	}
	body += "\n" + chrome.PanelDim.Render(ansi.Truncate(foot, w-2, "…")) + "\n"
	return lipgloss.NewStyle().Width(w).Height(h).BorderStyle(lipgloss.NormalBorder()).BorderRight(true).
		BorderForeground(chrome.Dim).Background(chrome.PanelBgColor).Render(body)
}

func (f *Floors) NavClick(y, w, h int) tea.Cmd {
	rows := f.navRows(w, h, true)
	if y < 0 || y >= min(len(rows), h-2) {
		return nil
	}
	r := rows[y]
	if r.floor >= 0 {
		f.selected = r.floor
		f.session = 0
		return nil
	}
	if r.conversation >= 0 {
		f.session = r.conversation
		return f.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	if r.action != "" {
		return f.Update(tea.KeyPressMsg(tea.Key{Code: rune(r.action[0]), Text: r.action}))
	}
	return nil
}

// New-floor and conversation forms float above all three panes. On narrow
// terminals Ctrl+E opens the project navigator as a dismissible drawer.
func (f *Floors) OverlayFrame(frame string, w, h int) string {
	cw, ch := min(76, w-4), min(21, h-2)
	var content string
	if f.form != nil {
		content = f.form.view(cw-2, ch-2)
	} else {
		cw, ch = min(36, w-2), h-2
		content = f.NavView(cw-2, ch-2, true)
	}
	card := chrome.PanelBox.Width(cw).Height(ch).Render(content)
	lines := strings.Split(frame, "\n")
	left, top := (w-cw)/2, (h-ch)/2
	if f.form == nil {
		left = 0
	}
	for i, row := range strings.Split(card, "\n") {
		y := top + i
		if y >= 0 && y < len(lines) {
			lines[y] = permSplice(lines[y], left, row, lipgloss.Width(row), w)
		}
	}
	return strings.Join(lines, "\n")
}
