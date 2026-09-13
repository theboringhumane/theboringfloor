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
	// The floor being LEFT decides the policy, not the target. EVERY
	// backend is handed off when busy now, via the same
	// handoffCurrentFloor path: opencode is proven to survive detaching
	// fully — an `opencode serve` child outlives its spawner and the
	// detached office can attach to that same serve (state.ServerAttachable),
	// so its in-flight turn survives the switch too. claude and codex
	// cannot keep an in-flight turn alive across the switch (claude's turn
	// rides stdin/stdout pipes that die with this process; codex's send()
	// blocks synchronously inside the dying process), but BOTH persist
	// their conversation outside this process (Claude Code's own session
	// store, resumed with --resume; codex's own thread store, resumed with
	// `codex exec resume`) and the detached child is already pinned to
	// that same session id — so the floor and its conversation still
	// survive, only the turn in flight does not. That is strictly better
	// than the old outright refusal, so the busy-refusal branch is gone:
	// the only thing that can still stop a busy switch is a handoff that
	// never becomes healthy (handoffCurrentFloor's own abort path, which
	// is backend-agnostic).
	leavingBackend := m.backendName()
	// Computed once: this predicate decides whether the departing floor
	// (of ANY backend) is worth handing off. An IDLE floor has nothing in
	// flight to preserve, so it takes the cheap path (plain exec-replace)
	// rather than paying for a detached office nobody asked for —
	// spawning a background process per switch would leak an office for
	// every floor a member merely browses past.
	leavingBusy := len(m.backendSwapBlockers()) > 0
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
	// Hand the DEPARTING floor off to a detached background office before
	// quitting this process — synchronously, so an aborted handoff (the
	// detached office never becomes healthy within handoffReadyTimeout)
	// truly leaves everything as it was: the session is already durably
	// persisted above, but nothing has quit or exec'd yet, so returning nil
	// here just keeps this same TUI running on this same floor.
	// Only a BUSY floor of ANY backend is worth a detached office. An idle
	// one has no in-flight turn to rescue, and its conversation is already
	// resumable from the session just persisted above, so it takes the
	// cheap path.
	if leavingBusy {
		if !m.handoffCurrentFloor() {
			return nil
		}
		// claude and codex cannot keep their in-flight turn alive across
		// the switch (see the gate comment above) — the member must be
		// told plainly what did and did not survive so nobody assumes the
		// turn kept running in the background. Opencode's handoff carries
		// the turn too (state.ServerAttachable + --server), so it has
		// nothing to disclose and stays silent.
		if leavingBackend != config.BackendNameDefault {
			m.notice(fmt.Sprintf(
				"%s floor moved to the background — its conversation keeps running there, but the turn in progress (%s) did not survive the switch and will need to be re-run.",
				leavingBackend,
				strings.Join(m.backendSwapBlockers(), "; "),
			))
		}
	}
	m.execFloor = &req
	m.closeTerminal()
	m.closeBrowser()
	return tea.Quit
}
func (m *Model) sendTicketToDraft(t workspace.Ticket) tea.Cmd {
	prompt := workspace.TicketPrompt(t)
	if !m.chat.StageDraft(prompt) {
		m.notice("Your draft is still in the composer. Send or clear it before staging this ticket.")
		m.tabs.SetActive(0)
		return nil
	}
	if m.st.Mode != state.ModeDemo && t.ID != "" && !strings.HasPrefix(t.ID, "agent:") && m.PrimarySessionID() != "" {
		if _, err := workspace.LinkTicket(m.sessDir, t.ID, m.backendName(), m.PrimarySessionID(), t.Updated); err != nil {
			m.notice("Ticket staged, but conversation link was not saved: " + err.Error())
			m.tabs.SetActive(0)
			return nil
		}
	}
	m.tabs.SetActive(0)
	m.notice("Ticket staged in the composer. Review it, then Enter to send.")
	return m.tickets.Refresh()
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
