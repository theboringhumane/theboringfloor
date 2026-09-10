package panels

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

type ticketsLoaded struct {
	floor workspace.Floor
	err   error
}
type ticketSaved struct {
	floor workspace.Floor
	err   error
}
type TicketRunMsg struct{ Ticket workspace.Ticket }
type TicketOpenMsg struct{ Ticket workspace.Ticket }
type Tickets struct {
	dir            string
	demo           bool
	w, h           int
	floor          workspace.Floor
	st             state.OfficeState
	lane, selected int
	detail         bool
	form           *workForm
	editing        workspace.Ticket
	err, query     string
	searching      bool
	team           int
	check          int
	saving         bool
}

func NewTickets(dir string, demo bool) *Tickets {
	return &Tickets{dir: dir, demo: demo, floor: workspace.Default(dir)}
}
func (b *Tickets) Title() string                 { return "board" }
func (b *Tickets) SetSize(w, h int)              { b.w, b.h = w, h }
func (b *Tickets) SetState(st state.OfficeState) { b.st = st }
func (b *Tickets) Editing() bool                 { return b.form != nil || b.searching }
func (b *Tickets) Refresh() tea.Cmd {
	if b.demo {
		return nil
	}
	dir := b.dir
	return func() tea.Msg { f, err := workspace.Load(dir); return ticketsLoaded{f, err} }
}
func (b *Tickets) all() []workspace.Ticket {
	out := append([]workspace.Ticket(nil), b.floor.Tickets...)
	// Automatic agent work stays visible beside persistent tickets. These
	// rows are read-only; changing them locally would lie about backend state.
	for _, t := range b.st.Tasks {
		status := "backlog"
		switch t.Status {
		case state.TaskInProgress:
			status = "in-progress"
		case state.TaskStalled:
			status = "blocked"
		case state.TaskDone:
			status = "done"
		}
		out = append(out, workspace.Ticket{ID: "agent:" + t.ID, Title: t.Title, Owner: t.Owner, Status: status, Priority: "P2"})
	}
	return out
}
func (b *Tickets) rows(lane int) []workspace.Ticket {
	var out []workspace.Ticket
	for _, t := range b.all() {
		if t.Status != workspace.Statuses[lane] {
			continue
		}
		if b.team > 0 && t.Team != b.floor.Teams[b.team-1].ID {
			continue
		}
		if b.query != "" && !strings.Contains(strings.ToLower(t.ID+" "+t.Title+" "+t.Owner+" "+t.Description), strings.ToLower(b.query)) {
			continue
		}
		out = append(out, t)
	}
	return out
}
func (b *Tickets) current() (workspace.Ticket, bool) {
	rows := b.rows(b.lane)
	if len(rows) == 0 {
		return workspace.Ticket{}, false
	}
	return rows[min(b.selected, len(rows)-1)], true
}
func (b *Tickets) edit(t workspace.Ticket) {
	teams := []string{"Unassigned"}
	team := 0
	for i, v := range b.floor.Teams {
		teams = append(teams, v.Name)
		if v.ID == t.Team {
			team = i + 1
		}
	}
	p, s := 2, 0
	for i, v := range workspace.Priorities {
		if v == t.Priority {
			p = i
		}
	}
	for i, v := range workspace.Statuses {
		if v == t.Status {
			s = i
		}
	}
	title := "NEW TICKET"
	if t.ID != "" {
		title = "EDIT " + t.ID
	}
	checks := []string{}
	for _, c := range t.Checklist {
		checks = append(checks, c.Text)
	}
	b.editing = t
	b.form = form(title, field("Title", t.Title), field("Description", t.Description), choice("Status", workspace.Statuses, s), choice("Priority", workspace.Priorities, p), choice("Team", teams, team), field("Owner", t.Owner), field("Checklist (separate items with ;)", strings.Join(checks, "; ")), field("Result summary", t.Result), field("Verification notes", t.Verification))
}
func (b *Tickets) save(t workspace.Ticket) tea.Cmd {
	if b.demo {
		if t.ID == "" {
			t.ID = workspace.ID("TKT")
		}
		found := false
		for i, v := range b.floor.Tickets {
			if v.ID == t.ID {
				b.floor.Tickets[i] = t
				found = true
			}
		}
		if !found {
			b.floor.Tickets = append(b.floor.Tickets, t)
		}
		b.form = nil
		return nil
	}
	dir := b.dir
	b.saving = true
	return func() tea.Msg { f, err := workspace.PutTicket(dir, t); return ticketSaved{f, err} }
}
func (b *Tickets) Update(msg tea.Msg) tea.Cmd {
	if m, ok := msg.(ticketSaved); ok {
		b.saving = false
		if m.err != nil {
			b.err = m.err.Error()
			if b.form != nil {
				b.form.err = b.err
			}
		} else {
			b.floor, b.form, b.err = m.floor, nil, ""
		}
		return nil
	}
	if m, ok := msg.(floorsLoaded); ok && m.err == nil {
		for _, floor := range m.floors {
			if floor.Dir == b.dir {
				b.floor = floor
			}
		}
		return nil
	}
	if m, ok := msg.(ticketsLoaded); ok {
		if m.err != nil {
			b.err = m.err.Error()
		} else {
			b.err = ""
			b.floor = m.floor
			b.team = min(b.team, len(b.floor.Teams))
		}
		return nil
	}
	if b.saving {
		return nil
	}
	if b.form != nil {
		save, cancel := b.form.update(msg)
		if cancel {
			b.form = nil
			return nil
		}
		if !save {
			return nil
		}
		t := b.editing
		t.Title = b.form.value(0)
		if t.Title == "" {
			b.form.err = "Give this ticket a title"
			return nil
		}
		t.Description = b.form.value(1)
		t.Status = b.form.value(2)
		t.Priority = b.form.value(3)
		t.Team = ""
		if i := b.form.fields[4].pick; i > 0 {
			t.Team = b.floor.Teams[i-1].ID
		}
		t.Owner = b.form.value(5)
		t.Result = b.form.value(7)
		t.Verification = b.form.value(8)
		old := map[string]bool{}
		for _, c := range t.Checklist {
			old[c.Text] = c.Done
		}
		t.Checklist = nil
		for _, s := range strings.Split(b.form.value(6), ";") {
			if s = strings.TrimSpace(s); s != "" {
				t.Checklist = append(t.Checklist, workspace.Check{Text: s, Done: old[s]})
			}
		}
		b.lane = b.form.fields[2].pick
		b.selected = 0
		return b.save(t)
	}
	if p, ok := msg.(tea.PasteMsg); ok && b.searching {
		b.query += strings.Join(strings.Fields(p.Content), " ")
		return nil
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	key := k.String()
	if b.searching {
		switch key {
		case "esc":
			b.query = ""
			b.searching = false
		case "enter":
			b.searching = false
		case "backspace":
			r := []rune(b.query)
			if len(r) > 0 {
				b.query = string(r[:len(r)-1])
			}
		default:
			if k.Text != "" {
				b.query += k.Text
			}
		}
		b.selected = 0
		return nil
	}
	t, has := b.current()
	local := has && !strings.HasPrefix(t.ID, "agent:")
	switch key {
	case "left", "h":
		b.lane = (b.lane + 4) % 5
		b.selected = 0
	case "right", "l":
		b.lane = (b.lane + 1) % 5
		b.selected = 0
	case "up", "k":
		if b.detail {
			b.check = max(0, b.check-1)
		} else {
			b.selected = max(0, b.selected-1)
		}
	case "down", "j":
		if b.detail {
			b.check = min(max(0, len(t.Checklist)-1), b.check+1)
		} else {
			b.selected = min(max(0, len(b.rows(b.lane))-1), b.selected+1)
		}
	case "n":
		b.edit(workspace.Ticket{Status: workspace.Statuses[b.lane], Priority: "P2"})
	case "enter":
		if has {
			b.detail = !b.detail
			b.check = 0
		}
	case "esc":
		b.detail = false
		b.query = ""
	case "e":
		if local {
			b.edit(t)
		}
	case "m":
		if local {
			b.lane = (b.lane + 1) % 5
			t.Status = workspace.Statuses[b.lane]
			b.selected = 0
			return b.save(t)
		}
	case "p":
		if local {
			for i, p := range workspace.Priorities {
				if p == t.Priority {
					t.Priority = workspace.Priorities[(i+1)%4]
					break
				}
			}
			return b.save(t)
		}
	case "t":
		b.team = (b.team + 1) % (len(b.floor.Teams) + 1)
		b.selected = 0
	case "x", " ":
		if local && b.detail && len(t.Checklist) > 0 {
			b.check = min(b.check, len(t.Checklist)-1)
			t.Checklist[b.check].Done = !t.Checklist[b.check].Done
			return b.save(t)
		}
	case "s":
		if local {
			return func() tea.Msg { return TicketRunMsg{t} }
		}
	case "o":
		if local && t.Session != "" && t.Backend != "" {
			return func() tea.Msg { return TicketOpenMsg{t} }
		}
	case "/":
		b.searching = true
	case "r":
		return b.Refresh()
	}
	return nil
}
func (b *Tickets) teamName(id string) string {
	for _, t := range b.floor.Teams {
		if t.ID == id {
			return t.Name
		}
	}
	return "Unassigned"
}

var laneLabels = []string{"BACKLOG", "IN PROGRESS", "BLOCKED", "REVIEW", "DONE"}

func (b *Tickets) View() string {
	if b.form != nil {
		return b.form.view(b.w, b.h)
	}
	team := "All teams"
	if b.team > 0 {
		team = b.floor.Teams[b.team-1].Name
	}
	head := chrome.PanelHeader.Render("TASK BOARD") + chrome.PanelDim.Render("  "+b.floor.Name+" / "+team+fmt.Sprintf(" · %d tickets", len(b.all())))
	if b.query != "" || b.searching {
		head += "  / " + b.query
	}
	if b.detail {
		t, ok := b.current()
		if ok {
			rows := []string{head, "", chrome.PanelHeader.Render(t.ID + "  ·  " + t.Priority + "  ·  " + t.Status), "", chrome.PanelHeader.Render(t.Title), "", t.Description, "", chrome.PanelDim.Render("Team: " + b.teamName(t.Team) + "    Owner: " + t.Owner)}
			if t.Session != "" {
				rows = append(rows, chrome.PanelDim.Render("Conversation: "+t.Backend+" / "+t.Session))
			}
			if t.Result != "" {
				rows = append(rows, "Result: "+t.Result)
			}
			if t.Verification != "" {
				rows = append(rows, "Recorded verification: "+t.Verification)
			}
			rows = append(rows, "", chrome.PanelHeader.Render("CHECKLIST"))
			if strings.HasPrefix(t.ID, "agent:") {
				rows = append(rows, "Live agent task · status is managed by the backend.")
			} else if len(t.Checklist) == 0 {
				rows = append(rows, "No checklist yet. Press e to add acceptance criteria.")
			}
			start := max(0, b.check-max(0, b.h-len(rows)-4))
			for i := start; i < len(t.Checklist); i++ {
				c := t.Checklist[i]
				mark := "[ ]"
				if c.Done {
					mark = "[x]"
				}
				prefix := "  "
				if i == b.check {
					prefix = "› "
				}
				rows = append(rows, prefix+mark+" "+c.Text)
			}
			rows = append(rows, "", chrome.PanelDim.Render("e edit   m move   p priority   ↑↓ checklist   x toggle   s prepare   o conversation   Esc back"))
			return workFit(strings.Join(rows, "\n"), b.w, b.h)
		}
	}
	visible := min(5, max(1, (b.w+2)/24))
	first := min(max(0, b.lane-visible+1), 5-visible)
	last := first + visible
	colW := max(1, (b.w-2*(visible-1))/visible)
	var cols []string
	for lane := first; lane < last; lane++ {
		rows := b.rows(lane)
		label := fmt.Sprintf("%s %d", laneLabels[lane], len(rows))
		style := chrome.PanelHeader
		if lane == b.lane {
			style = chrome.TabActive
		}
		lines := []string{style.Render(ansi.Truncate(" "+label+" ", colW, "…")), chrome.PanelDim.Render(strings.Repeat("─", colW)), ""}
		start := 0
		if lane == b.lane {
			start = max(0, b.selected-max(0, (b.h-11)/5))
		}
		for i := start; i < len(rows); i++ {
			t := rows[i]
			title := t.Title
			meta := t.Priority + " · " + b.teamName(t.Team)
			prefix := " "
			if lane == b.lane && i == min(b.selected, len(rows)-1) {
				prefix = "›"
				title = chrome.PanelHeader.Render(title)
			}
			lines = append(lines, chrome.PanelDim.Render(prefix+" "+strings.TrimPrefix(t.ID, "agent:")), prefix+" "+ansi.Truncate(title, max(1, colW-2), "…"), chrome.PanelDim.Render("  "+meta))
			if t.Owner != "" {
				lines = append(lines, chrome.PanelDim.Render("  @"+t.Owner))
			} else {
				lines = append(lines, "")
			}
			lines = append(lines, "")
		}
		if len(rows) == 0 {
			lines = append(lines, chrome.PanelDim.Render("  No tickets"))
			if lane == b.lane {
				lines = append(lines, chrome.PanelDim.Render("  n create ticket"))
			}
		}
		cols = append(cols, workBox(strings.Join(lines, "\n"), colW, b.h-5))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, paddedJoin(cols, 2)...)
	foot := "←→ lane   ↑↓ ticket   Enter details   n new   e edit   m move   t team   / search"
	if b.err != "" {
		foot = b.err
	}
	return workFit(head+"\n\n"+body+"\n"+chrome.PanelDim.Render(foot), b.w, b.h)
}
