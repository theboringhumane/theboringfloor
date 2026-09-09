package panels

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

type FloorLaunchMsg struct {
	Dir, Backend, Session, Title, Team string
	Fresh                              bool
}
type floorsLoaded struct {
	floors        []workspace.Floor
	conversations map[string][]workspace.Conversation
	err           error
}
type Floors struct {
	dir               string
	demo              bool
	w, h              int
	rows              []workspace.Floor
	selected, session int
	conversations     map[string][]workspace.Conversation
	form              *workForm
	action            string
	err               string
	loading           bool
	backend           string
}

func NewFloors(dir, backend string, demo bool) *Floors {
	return &Floors{dir: dir, demo: demo, backend: backend, rows: []workspace.Floor{workspace.Default(dir)}, conversations: map[string][]workspace.Conversation{}}
}
func (f *Floors) Title() string              { return "floors" }
func (f *Floors) SetSize(w, h int)           { f.w, f.h = w, h }
func (f *Floors) SetState(state.OfficeState) {}
func (f *Floors) Refresh() tea.Cmd {
	f.loading = true
	dir, demo := f.dir, f.demo
	return func() tea.Msg {
		if demo {
			return floorsLoaded{floors: []workspace.Floor{workspace.Default(dir)}, conversations: map[string][]workspace.Conversation{}}
		}
		_, err := workspace.Register(dir)
		if err != nil {
			return floorsLoaded{err: err}
		}
		rows, err := workspace.List()
		convs := map[string][]workspace.Conversation{}
		for _, r := range rows {
			cs, e := workspace.Conversations(r.Dir)
			if e != nil {
				err = e
			}
			convs[r.Dir] = cs
		}
		return floorsLoaded{rows, convs, err}
	}
}
func (f *Floors) Current() workspace.Floor {
	if len(f.rows) == 0 {
		return workspace.Default(f.dir)
	}
	return f.rows[min(f.selected, len(f.rows)-1)]
}
func (f *Floors) Editing() bool { return f.form != nil }
func (f *Floors) NewConversation() {
	row := f.Current()
	teams := []string{"All teams"}
	for _, t := range row.Teams {
		teams = append(teams, t.Name)
	}
	backends := []string{"opencode", "claudecode", "codex"}
	selected := 0
	for i, b := range backends {
		if b == f.backend {
			selected = i
		}
	}
	f.action = "conversation"
	f.form = form("NEW CONVERSATION · "+row.Name, field("Title", ""), choice("Backend", backends, selected), choice("Team", teams, 0))
}
func (f *Floors) Update(msg tea.Msg) tea.Cmd {
	if m, ok := msg.(ticketsLoaded); ok && m.err == nil {
		for i, floor := range f.rows {
			if floor.Dir == m.floor.Dir {
				f.rows[i] = m.floor
			}
		}
		return nil
	}
	if m, ok := msg.(floorsLoaded); ok {
		f.loading = false
		if m.err != nil {
			f.err = m.err.Error()
			return nil
		}
		previous := f.Current().Dir
		f.err = ""
		f.rows = m.floors
		for i, row := range f.rows {
			if row.Dir == previous {
				f.selected = i
				break
			}
		}
		f.conversations = m.conversations
		f.selected = min(f.selected, max(0, len(f.rows)-1))
		f.session = 0
		return nil
	}
	if f.form != nil {
		save, cancel := f.form.update(msg)
		if cancel {
			f.form = nil
			return nil
		}
		if !save {
			return nil
		}
		row := f.Current()
		switch f.action {
		case "conversation":
			team := ""
			if i := f.form.fields[2].pick; i > 0 {
				team = row.Teams[i-1].ID
			}
			launch := FloorLaunchMsg{Dir: row.Dir, Backend: f.form.value(1), Title: f.form.value(0), Team: team, Fresh: true}
			f.form = nil
			return func() tea.Msg { return launch }
		case "floor":
			if f.demo {
				f.form.err = "Floor registration is available in a live office"
				return nil
			}
			if _, err := workspace.Register(f.form.value(0)); err != nil {
				f.form.err = err.Error()
				return nil
			}
		case "team":
			if f.demo {
				f.form.err = "Team changes are available in a live office"
				return nil
			}
			if _, err := workspace.AddTeam(row.Dir, f.form.value(0)); err != nil {
				f.form.err = err.Error()
				return nil
			}
		}
		f.form = nil
		return f.Refresh()
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up", "k":
			f.selected = max(0, f.selected-1)
			f.session = 0
		case "down", "j":
			f.selected = min(max(0, len(f.rows)-1), f.selected+1)
			f.session = 0
		case "left", "h":
			f.session = max(0, f.session-1)
		case "right", "l":
			f.session = min(max(0, len(f.conversations[f.Current().Dir])-1), f.session+1)
		case "n":
			f.NewConversation()
		case "a":
			f.action = "floor"
			f.form = form("ADD PROJECT FLOOR", field("Project directory (existing folder)", ""))
		case "t":
			f.action = "team"
			f.form = form("ADD TEAM · "+f.Current().Name, field("Team name", ""))
		case "r":
			return f.Refresh()
		case "enter":
			row := f.Current()
			launch := FloorLaunchMsg{Dir: row.Dir}
			if cs := f.conversations[row.Dir]; len(cs) > 0 {
				c := cs[min(f.session, len(cs)-1)]
				launch.Backend = c.Backend
				launch.Session = c.ID
				launch.Title = c.Title
				launch.Team = c.Team
			}
			return func() tea.Msg { return launch }
		}
	}
	return nil
}
func (f *Floors) View() string {
	if f.form != nil {
		return f.form.view(f.w, f.h)
	}
	header := chrome.PanelHeader.Render("PROJECT FLOORS") + chrome.PanelDim.Render(fmt.Sprintf("  %d projects · one floor per repository", len(f.rows)))
	leftW := max(20, min(34, f.w/3))
	rightW := max(10, f.w-leftW-3)
	left := []string{chrome.PanelDim.Render("YOUR WORKSPACES"), ""}
	start := max(0, f.selected-max(0, (f.h-9)/3))
	for i := start; i < len(f.rows); i++ {
		r := f.rows[i]
		prefix := "  "
		if r.Dir == f.dir {
			prefix = "● "
		}
		name := prefix + r.Name
		if i == f.selected {
			name = chrome.TabActive.Render(" " + name + " ")
		} else {
			name = chrome.PanelHeader.Render(name)
		}
		left = append(left, name, chrome.PanelDim.Render(fmt.Sprintf("  %d teams · %d tickets", len(r.Teams), len(r.Tickets))), "")
	}
	row := f.Current()
	right := []string{chrome.PanelHeader.Render(row.Name), chrome.PanelDim.Render(row.Dir), "", chrome.PanelHeader.Render("TEAMS")}
	var teams []string
	for _, t := range row.Teams {
		teams = append(teams, t.Name)
	}
	right = append(right, strings.Join(teams, "  ·  "), "", chrome.PanelHeader.Render("CONVERSATIONS"), chrome.PanelDim.Render("← → select conversation · Enter resume · n new"), "")
	cs := f.conversations[row.Dir]
	if len(cs) == 0 {
		right = append(right, "Start the first conversation on this floor.", "Choose OpenCode, Claude Code, or Codex.")
	}
	start = max(0, f.session-max(0, (f.h-17)/3))
	for i := start; i < len(cs); i++ {
		c := cs[i]
		title := c.Title
		if i == f.session {
			title = chrome.TabActive.Render("› " + title + " ")
		}
		right = append(right, title, chrome.PanelDim.Render(fmt.Sprintf("  %s · %d messages · %s", c.Backend, c.Messages, time.UnixMilli(c.Updated).Format("02 Jan 15:04"))), "")
	}
	if f.loading {
		right = append(right, chrome.PanelDim.Render("Loading floors…"))
	}
	if f.err != "" {
		right = append(right, f.err)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, workBox(strings.Join(left, "\n"), leftW, f.h-5), workRule(f.h-5), workBox(strings.Join(right, "\n"), rightW, f.h-5))
	if f.w < 65 {
		body = workFit(strings.Join(append(left[:min(len(left), 5)], right...), "\n"), f.w, f.h-5)
	}
	return workFit(header+"\n\n"+body+"\n"+chrome.PanelDim.Render("↑↓ floor   a add project   n new conversation   t add team   r refresh"), f.w, f.h)
}

func (f *Floors) SelectCurrent() {
	for i, row := range f.rows {
		if row.Dir == f.dir {
			f.selected = i
			f.session = 0
			return
		}
	}
}
