package panels

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"strings"
)

func (c *Chat) StageDraft(text string) bool {
	if strings.TrimSpace(c.ta.Value()) != "" {
		return false
	}
	c.ta.SetValue(text)
	c.afterDraftEdit()
	return true
}
func (c *Chat) StageAttachment(a state.Attachment) tea.Cmd {
	return c.addAttachment(chatAttachment{name: a.Name, mime: a.Mime, path: a.Path, temp: a.Temp})
}
func (c *Chat) SetWorkspaceContext(project, backend, team string) {
	c.workspaceProject = project
	c.workspaceBackend = backend
	c.workspaceTeam = team
}
func (c *Chat) Searching() bool { return c.searching }
func (c *Chat) workspaceDivider() string {
	if c.searching {
		return chrome.PanelAccent.Render(ansi.Truncate(fmt.Sprintf(" Find: %s  ·  %d matches · Enter next · Esc close", c.searchText, len(c.searchMatches)), c.w, "…"))
	}
	if c.workspaceBackend == "" {
		return chrome.PanelDim.Render(fitPlain(strings.Repeat("─", c.w), c.w))
	}
	label := " " + c.workspaceBackend
	if c.workspaceTeam != "" {
		label += " / " + c.workspaceTeam
	}
	if !c.follow {
		label += " · reading history · End latest"
	} else {
		label += " · Ctrl+R find · Ctrl+W expand"
	}
	return chrome.PanelDim.Render(ansi.Truncate(label+" "+strings.Repeat("─", c.w), c.w, ""))
}
func (c *Chat) searchKey(k tea.KeyPressMsg) {
	c.refreshSearch()
	changed := false
	switch k.String() {
	case "esc":
		c.searching = false
		return
	case "enter", "down":
		if len(c.searchMatches) > 0 {
			c.searchIndex = (c.searchIndex + 1) % len(c.searchMatches)
		}
	case "shift+enter", "up":
		if len(c.searchMatches) > 0 {
			c.searchIndex = (c.searchIndex + len(c.searchMatches) - 1) % len(c.searchMatches)
		}
	case "backspace":
		r := []rune(c.searchText)
		if len(r) > 0 {
			c.searchText = string(r[:len(r)-1])
			changed = true
		}
	default:
		if k.Text != "" {
			c.searchText += k.Text
			changed = true
		}
	}
	if changed {
		c.searchIndex = 0
		c.refreshSearch()
	}
	if len(c.searchMatches) > 0 {
		c.follow = false
		c.vp.SetYOffset(c.searchMatches[c.searchIndex])
		c.syncWindow()
	}
}

func (c *Chat) refreshSearch() {
	c.searchMatches = nil
	if c.searchText != "" {
		q := strings.ToLower(c.searchText)
		for i, row := range c.selLines {
			if strings.Contains(strings.ToLower(ansi.Strip(row)), q) {
				c.searchMatches = append(c.searchMatches, i)
			}
		}
	}
	c.searchIndex = min(c.searchIndex, max(0, len(c.searchMatches)-1))
}
