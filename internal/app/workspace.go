package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

type FloorLaunch = panels.FloorLaunchMsg

func WithConversation(fresh bool, title, team string) Option {
	return func(m *Model) { m.freshConversation = fresh; m.conversationTitle = title; m.conversationTeam = team }
}
func (m Model) FloorExecRequest() *FloorLaunch { return m.execFloor }
func (m Model) widePanel() bool {
	return !m.planPaneVisible() && m.focusPanel
}

// The project navigator is separate from the office and its tool tabs.
func (m Model) navigatorWidth() int {
	if m.mobile() {
		return 0
	}
	return min(28, max(22, m.width/6))
}
func (m Model) panelX() int {
	if m.mobile() {
		return 0
	}
	if m.widePanel() {
		return m.navigatorWidth()
	}
	return m.navigatorWidth() + m.floorW
}
func (m Model) floorOverlay() bool {
	return m.floors != nil && (m.floors.Editing() || m.mobile() && m.floorNavFocused)
}
func (m *Model) focusFloors() tea.Cmd {
	m.floorNavFocused = true
	m.setTermCaptured(false)
	if m.plan != nil {
		m.plan.Blur()
	}
	return m.floors.Refresh()
}
func (m *Model) workspaceEditing() bool {
	switch m.tabs.ActiveIndex() {
	case 3:
		return m.tickets.Editing()
	case 7:
		return m.files.Editing()
	}
	return false
}
func (m *Model) launchFloor(req FloorLaunch) tea.Cmd {
	m.floorNavFocused = false
	if why := m.backendSwapBlockers(); len(why) > 0 {
		m.noticeErr("Finish or stop the current work before switching conversations: " + strings.Join(why, "; "))
		m.tabs.SetActive(0)
		return nil
	}
	dir, err := workspace.Canonical(req.Dir)
	if err != nil {
		m.noticeErr(err.Error())
		m.tabs.SetActive(0)
		return nil
	}
	req.Dir = dir
	if req.Backend == "" {
		req.Backend = m.backendName()
		if sf, ok := LoadSession(dir); ok && config.ValidBackendName(sf.Backend) {
			req.Backend = sf.Backend
		}
	}
	if !config.ValidBackendName(req.Backend) {
		m.noticeErr("Unknown backend: " + req.Backend)
		return nil
	}
	if m.st.Mode != state.ModeLive {
		m.notice("Start a live office to open project conversations. Demo tickets can be explored on the board.")
		m.tabs.SetActive(0)
		return nil
	}
	// Catch a missing local CLI before quitting the current office. OpenCode
	// may use a remote serve URL, so its transport validates its own endpoint.
	if req.Backend == "codex" || req.Backend == "claudecode" {
		bin, env := "codex", "CODEX_BIN"
		if req.Backend == "claudecode" {
			bin, env = "claude", "CLAUDE_BIN"
		}
		if override := config.Env(env); override != "" {
			bin = override
		}
		if _, err := exec.LookPath(bin); err != nil {
			m.noticeErr("Cannot open conversation: " + bin + " is not installed or executable")
			m.tabs.SetActive(0)
			return nil
		}
	}
	if !req.Fresh && req.Dir == m.sessDir && (req.Session == "" || req.Session == m.PrimarySessionID()) && req.Backend == m.backendName() {
		m.tabs.SetActive(0)
		return nil
	}
	m.persistOfficeSession(true)
	if m.sessionWriter != nil {
		m.sessionWriter.mu.Lock()
		err = m.sessionWriter.err
		m.sessionWriter.mu.Unlock()
		if err != nil {
			m.noticeErr("Could not save the current conversation: " + err.Error())
			m.tabs.SetActive(0)
			return nil
		}
	}
	m.execFloor = &req
	m.closeTerminal()
	m.closeBrowser()
	return tea.Quit
}
func (m *Model) sendTicketToDraft(t workspace.Ticket) tea.Cmd {
	prompt := fmt.Sprintf("Work on ticket %s: %s\n\n%s", t.ID, t.Title, t.Description)
	if t.Team != "" {
		prompt += "\nTeam: " + t.Team
	}
	if t.Owner != "" {
		prompt += "\nOwner: " + t.Owner
	}
	for _, c := range t.Checklist {
		mark := "[ ]"
		if c.Done {
			mark = "[x]"
		}
		prompt += "\n" + mark + " " + c.Text
	}
	if !m.chat.StageDraft(prompt) {
		m.notice("Your draft is still in the composer. Send or clear it before staging this ticket.")
		m.tabs.SetActive(0)
		return nil
	}
	m.tabs.SetActive(0)
	m.notice("Ticket staged in the composer. Review it, then Enter to send.")
	return nil
}
func (m *Model) workspaceInit() {
	dir := m.sessDir
	if dir == "" {
		dir, _ = os.Getwd()
	}
	demo := m.st.Mode != state.ModeLive
	if !demo {
		// Migrate the previous single snapshot before any fresh conversation can
		// overwrite it, including snapshots older than the auto-resume window.
		if sf, ok := LoadSession(dir); ok && sf.PrimaryID != "" {
			if sf.Backend == "" {
				sf.Backend = config.BackendNameDefault
			}
			if _, exists := loadConversation(dir, sf.Backend, sf.PrimaryID); !exists {
				w := &sessionWriter{}
				w.save(w.reserve(), dir, *sf, mergeArchiveChat(nil, sf.Chat), sf.Title, sf.Team)
			}
		}
	}
	m.cfg.ConversationTeam = m.conversationTeam
	if floor, err := workspace.Load(dir); err == nil {
		for _, team := range floor.Teams {
			if team.ID == m.conversationTeam {
				m.cfg.ConversationTeam = team.Name
			}
		}
	}
	m.floors = panels.NewFloors(dir, m.backendName(), demo)
	m.files = panels.NewFiles(dir)
	m.tickets = panels.NewTickets(dir, demo)
	m.tabs.Replace(3, m.tickets)
	m.tabs.Append(m.files)
	m.tabs.SetState(m.st)
	m.chat.SetWorkspaceContext(m.projInfo().Project, m.backendName(), m.conversationTeam)
}
