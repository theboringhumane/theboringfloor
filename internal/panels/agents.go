// agents.go — AGENTS roster tab (port of node-legacy panels/agents.tsx):
// boss pinned first (brain.json boss.name, default "boss (oikonomos)")
// yellow bold, then the pinned concierge row — "office (concierge)" with
// its live status word in INFO ("answering" while an EvChatOffice bubble is
// pending, else "on call") — then one row per employee:
//
//	<name (role color)>  <glyph> <sprite word (semantic color)>   <task, dim, right>
//
// blocked (at-mailbox) employees show "blocked" in bold red.
package panels

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// sprite word per SpriteState (port of WORD in agents.tsx).
func spriteWord(s state.SpriteState) string {
	switch s {
	case state.SpriteWorking:
		return "working"
	case state.SpriteToManager, state.SpriteMeeting:
		return "meeting"
	case state.SpriteToCoffee, state.SpriteCoffee:
		return "coffee"
	case state.SpriteAtMailbox:
		return "blocked"
	default:
		return "at desk"
	}
}

// wordStyle — semantic chrome color per sprite word (port of WORD_STYLE).
func wordStyle(word string, s string) string {
	switch word {
	case "working":
		return chrome.OnPanel(chrome.Info, s)
	case "meeting":
		return chrome.OnPanel(chrome.Accent, s)
	case "coffee":
		return lipgloss.NewStyle().Foreground(chrome.Accent).Faint(true).Render(s)
	case "blocked":
		return chrome.OnPanelBold(chrome.Err, s)
	case "at desk":
		return chrome.OnPanel(chrome.Dim, s)
	default:
		return chrome.OnPanel(chrome.White, s)
	}
}

// Agents is the roster tab panel.
type Agents struct {
	vp       viewport.Model
	st       state.OfficeState
	w, h     int
	rev      string // last rendered content (compare-based cache)
	key      string // cheap fingerprint of every render input — hit ⇒ skip render+compare
	bossName string // cfg.Boss.Name — the human label pinned first (default below)
	selected string // floor-click selection: a "▸" marker + bold row (popover.go SetSelected)
}

// defaultBossName is the pinned boss label when no config name is set.
const defaultBossName = "boss (oikonomos)"

// NewAgents builds the roster panel.
func NewAgents() *Agents {
	vp := viewport.New(viewport.WithWidth(10), viewport.WithHeight(5))
	vp.MouseWheelEnabled = true
	return &Agents{vp: vp}
}

// SetBossName personalizes the pinned boss row's label (brain.json
// boss.name). Empty restores the default. Re-renders immediately.
func (a *Agents) SetBossName(name string) {
	if a.bossName == name {
		return
	}
	a.bossName = name
	a.SetState(a.st)
}

// bossLabel — the pinned first row's human name (config or default).
func (a *Agents) bossLabel() string {
	if a.bossName == "" {
		return defaultBossName
	}
	return a.bossName
}

// Title implements Tab.
func (a *Agents) Title() string { return "agents" }

// SetSize implements Tab.
func (a *Agents) SetSize(w, h int) {
	a.w, a.h = w, h
	a.vp.SetWidth(w)
	if h-1 > 0 {
		a.vp.SetHeight(h - 1) // header row
	}
	a.SetState(a.st)
}

// SetState implements Tab. Same rev-key discipline as Board/Mail: the key
// fingerprints every render input (width, theme, boss label, selection,
// concierge-answering flag, the full employee tuple set); a hit skips
// render+compare, a miss falls back to the old render+compare path
// verbatim — output is byte-identical to the un-keyed panel in every case.
func (a *Agents) SetState(st state.OfficeState) {
	a.st = st
	key := a.revKeyOf(st)
	if key == a.key {
		return // provably identical content — zero render churn
	}
	a.key = key
	content := a.render()
	if content != a.rev {
		a.rev = content
		a.vp.SetContent(content)
	}
}

// revKeyOf — every input render() consumes, cheaply: panel width, theme
// name (chrome vars re-point on /theme), the pinned boss label, the
// floor-click selection, the concierge "answering" flag (a pending
// EvChatOffice bubble anywhere in the chat log), and per employee
// id·name·role·sprite·task (a mid-roster task edit with stable ids still
// flips the key).
func (a *Agents) revKeyOf(st state.OfficeState) string {
	answering := false
	for _, m := range st.Chat {
		if m.From == "office" && m.Kind == "office" && m.Pending {
			answering = true
			break
		}
	}
	var sb strings.Builder
	sb.Grow(64 + len(st.Employees)*48)
	sb.WriteString(strconv.Itoa(a.w))
	sb.WriteByte('|')
	sb.WriteString(chrome.CurrentTheme().Name)
	sb.WriteByte('|')
	sb.WriteString(a.bossName)
	sb.WriteByte('|')
	sb.WriteString(a.selected)
	sb.WriteByte('|')
	sb.WriteString(strconv.FormatBool(answering))
	for _, e := range st.Employees {
		sb.WriteByte('|')
		sb.WriteString(e.ID)
		sb.WriteByte(',')
		sb.WriteString(e.Name)
		sb.WriteByte(',')
		sb.WriteString(string(e.Role))
		sb.WriteByte(',')
		sb.WriteString(string(e.Sprite))
		sb.WriteByte(',')
		sb.WriteString(e.Task)
	}
	return sb.String()
}

// Update implements Interactive (scroll).
func (a *Agents) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseWheelMsg:
		var cmd tea.Cmd
		a.vp, cmd = a.vp.Update(msg)
		return cmd
	}
	return nil
}

// View implements Tab.
func (a *Agents) View() string {
	return fit(chrome.PanelHeader.Render("AGENTS")+"\n"+a.vp.View(), a.h)
}

// render — boss first, then the rest; task right-aligned when it fits.
func (a *Agents) render() string {
	if len(a.st.Employees) == 0 {
		return chrome.PanelDim.Render("- empty -")
	}
	ordered := make([]state.Employee, 0, len(a.st.Employees))
	for _, e := range a.st.Employees {
		if e.Role == state.RoleManager {
			ordered = append(ordered, e)
		}
	}
	for _, e := range a.st.Employees {
		if e.Role != state.RoleManager {
			ordered = append(ordered, e)
		}
	}

	var b strings.Builder
	officePinned := false
	for _, e := range ordered {
		if e.Role != state.RoleManager && !officePinned {
			// the concierge row pins directly under the boss block
			officePinned = true
			b.WriteString(a.officeRow() + "\n")
		}
		isBoss := e.Role == state.RoleManager
		label := e.Name
		if isBoss {
			label = a.bossLabel()
		}
		word := spriteWord(e.Sprite)
		c := chrome.RoleColor(label)

		// floor-click selection marker: "▸ " + bold row (2 cells of the
		// width budget go to the marker)
		sel := a.selected != "" && e.Name == a.selected
		marker := "  "
		if sel {
			marker = chrome.OnPanelBold(c, "▸ ")
		}
		leftPlain := label + " " + chrome.RoleGlyph(e.Role) + " " + word
		task := e.Task
		room := a.w - len(leftPlain) - 1 - 2
		if task != "" && room >= 6 && len(task) > room {
			task = task[:room-3] + "..." // ellipsis on TASK TEXT (machine), not NL
		}
		gap := a.w - len(leftPlain) - len(task) - 2
		if gap < 1 {
			gap = 1
		}

		var left string
		if isBoss {
			left = chrome.OnPanelBold(c, label) + " " +
				chrome.OnPanel(c, chrome.RoleGlyph(e.Role)) + " " +
				wordStyle(word, word)
		} else {
			left = chrome.OnPanel(c, label) + " " +
				chrome.OnPanel(c, chrome.RoleGlyph(e.Role)) + " " +
				wordStyle(word, word)
		}
		if sel {
			left = lipgloss.NewStyle().Bold(true).Render(left)
		}
		line := marker + left + strings.Repeat(" ", gap)
		if task != "" && gap > 1 {
			line += chrome.PanelDim.Render(task)
		}
		b.WriteString(line + "\n")
	}
	if !officePinned {
		// boss-less roster (defensive — initialState always seats one)
		b.WriteString(a.officeRow() + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// officeRow — the pinned concierge row under the boss: "office (concierge)"
// with its own live status word in INFO (cyan) — "answering" while any
// concierge (EvChatOffice) bubble is pending, "on call" the rest of the
// time. No glyph, no task cell: the concierge lives in the chat lane, not
// on the floor.
func (a *Agents) officeRow() string {
	answering := false
	for _, m := range a.st.Chat {
		if m.From == "office" && m.Kind == "office" && m.Pending {
			answering = true
			break
		}
	}
	word := "on call"
	if answering {
		word = "answering"
	}
	left := chrome.OnPanel(chrome.Info, "office (concierge)") + " " +
		chrome.OnPanel(chrome.Info, word)
	gap := a.w - len("office (concierge) "+word) - 2
	if gap < 1 {
		gap = 1
	}
	return "  " + left + strings.Repeat(" ", gap)
}
