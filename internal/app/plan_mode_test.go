// plan_mode_test.go — the plan/build agent-mode wiring end to end:
// ctrl+p toggling (with the terminal-focus and float-open exclusions),
// the ctrl+x approve→build flow (compose + agent seam + mode flip +
// notice + buffer reset), the unedited-template refusal, plan-mode
// composer sends routing through SendAgent(…, "plan"), the session.json
// planText round-trip (hydrate / /new / approve clearing), the statusbar
// [plan] badge, and the desktop/mobile layout swaps.
package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// agentRecBackend — a recording state.Backend WITH the plan/build agent
// seam: the app's type-assert must pick SendAgent over plain Send for
// plan-mode and approved-plan prompts.
type agentRecBackend struct {
	agentCalls []agentCall
	sentTexts  []string // SendWith path (build mode + seam degrade)
}

type agentCall struct{ text, agent string }

func (r *agentRecBackend) Mode() state.Mode                        { return state.ModeDemo }
func (r *agentRecBackend) Start(func(state.Event)) error           { return nil }
func (r *agentRecBackend) Stop() error                             { return nil }
func (r *agentRecBackend) Send(text string) error                  { return nil }
func (r *agentRecBackend) AnswerPermission(string, string) error   { return nil }
func (r *agentRecBackend) AnswerQuestion(string, [][]string) error { return nil }
func (r *agentRecBackend) RejectQuestion(string) error             { return nil }
func (r *agentRecBackend) MCPServers() ([]state.MCPServer, error)  { return nil, nil }
func (r *agentRecBackend) ReconnectMCP(string) error               { return nil }
func (r *agentRecBackend) SendWith(text string, atts []state.Attachment) error {
	r.sentTexts = append(r.sentTexts, text)
	return nil
}
func (r *agentRecBackend) SendAgent(text, agent string) error {
	r.agentCalls = append(r.agentCalls, agentCall{text: text, agent: agent})
	return nil
}

func ctrlP() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}) }
func ctrlX() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'x', Mod: tea.ModCtrl}) }
func escKey() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
}
func enterKey() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
}

// typeText feeds one run of plain characters through Update (the chat
// textarea inserts them) — mirroring how the queue tests drive keys.
func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = runMsg(t, m, pressKey(r))
	}
	return m
}

// lastOfficeMsg returns the most recent From-"office" transcript line.
func lastOfficeMsg(m Model) string {
	for i := len(m.st.Chat) - 1; i >= 0; i-- {
		if m.st.Chat[i].From == "office" {
			return m.st.Chat[i].Text
		}
	}
	return ""
}

// TestPlanModeToggleWithExclusions pins (a): ctrl+p flips build→plan→
// build and drives the editor's focus; a focused terminal tab keeps
// ctrl+p for the shell, and an open permission float keeps its keys.
func TestPlanModeToggleWithExclusions(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	if m.agentMode != agentModeBuild {
		t.Fatalf("a fresh office boots in build mode, got %q", m.agentMode)
	}

	// build → plan: mode flips, the editor takes focus + the badge mode
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModePlan {
		t.Fatalf("ctrl+p must enter plan mode, got %q", m.agentMode)
	}
	if !m.plan.Focused() {
		t.Fatal("entering plan mode must focus the plan editor")
	}
	if m.plan.Mode() != agentModePlan {
		t.Fatalf("the pane mirrors the mode badge, got %q", m.plan.Mode())
	}

	// esc (editor-focused) blurs back to the chat input — mode unchanged
	m = runMsg(t, m, escKey())
	if m.plan.Focused() {
		t.Fatal("esc must blur the editor (done editing for now)")
	}
	if m.agentMode != agentModePlan {
		t.Fatalf("esc blurs the editor, it must NOT exit plan mode: %q", m.agentMode)
	}

	// plan → build
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModeBuild {
		t.Fatalf("second ctrl+p must return to build, got %q", m.agentMode)
	}
	if m.plan.Focused() {
		t.Fatal("leaving plan mode must blur the editor")
	}

	// EXCLUSION 1 — the focused terminal tab keeps ctrl+p for the shell
	m.tabs.SetActive(terminalIndex)
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModeBuild {
		t.Fatalf("ctrl+p belongs to the shell while the terminal tab is focused: %q", m.agentMode)
	}
	m.tabs.SetActive(0)

	// EXCLUSION 2 — an open permission float keeps its keys
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-pin-1",
		EmployeeID: "tekton-1", EmployeeName: "tekton", ToolName: "Write",
		ToolSummary: "/tmp/x", ToolState: "pending"})
	if m.permQ.front() == nil {
		t.Fatal("the permission event must open the float")
	}
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModeBuild {
		t.Fatalf("ctrl+p must not toggle while a perm float is open: %q", m.agentMode)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-pin-1", ToolState: "resolved"})
	if m.permQ.front() != nil {
		t.Fatal("the float must resolve before the toggle works again")
	}
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModePlan {
		t.Fatalf("ctrl+p toggles again once the float resolves: %q", m.agentMode)
	}
}

// TestPlanApproveHappyPath pins (b): an edited plan + ctrl+x sends the
// composed prompt through the agent seam with agent="build", flips the
// office back to build mode, consumes the buffer, and lands the notice.
func TestPlanApproveHappyPath(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	body := "Plan: plan/build modes for the office\n1. wire the agent seam\n2. swap the floor slot"
	m.plan.SetValue(body)
	m = runMsg(t, m, ctrlX())

	if len(b.agentCalls) != 1 {
		t.Fatalf("approve must send exactly once through the agent seam, got %d calls", len(b.agentCalls))
	}
	call := b.agentCalls[0]
	if call.agent != agentModeBuild {
		t.Fatalf("approve sends with agent=%q, want %q", call.agent, agentModeBuild)
	}
	want := approvePrefix + body
	if call.text != want {
		t.Fatalf("composed prompt mismatch:\n got %q\nwant %q", call.text, want)
	}
	// Wire trace for the report: the seam, then the first lines of the prompt.
	lines := strings.Split(call.text, "\n")
	head := lines
	if len(head) > 3 {
		head = head[:3]
	}
	t.Logf("approve wire: SendAgent(text, agent=%q)", call.agent)
	for i, ln := range head {
		t.Logf("  composed[%d] = %q", i, ln)
	}

	if m.agentMode != agentModeBuild || m.plan.Mode() != agentModeBuild {
		t.Fatalf("approve flips the office to build: agentMode=%q pane=%q", m.agentMode, m.plan.Mode())
	}
	if m.plan.Focused() {
		t.Fatal("approve blurs the editor with the mode flip")
	}
	if m.plan.Value() != m.planTemplate {
		t.Fatal("an approved plan is consumed — the editor resets to the starter template")
	}
	notice := lastOfficeMsg(m)
	wantNotice := fmt.Sprintf("[office] plan approved — sent to build (%d chars)", len(body))
	if notice != wantNotice {
		t.Fatalf("approve notice = %q, want %q", notice, wantNotice)
	}
	// Transcript carries the approval — and the plan is no longer pending
	if m.planText() != "" {
		t.Fatalf("an approved plan drops out of persistence, got %q", m.planText())
	}
}

// TestPlanApproveRefusesUnedited pins (c): ctrl+x on the untouched
// starter template (or an emptied buffer) refuses — a dim notice, no
// send, and the office stays in plan mode.
func TestPlanApproveRefusesUnedited(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	m = runMsg(t, m, ctrlX()) // the template, untouched
	if len(b.agentCalls) != 0 {
		t.Fatal("an untouched template must NOT send")
	}
	if m.agentMode != agentModePlan || !m.plan.Focused() {
		t.Fatalf("a refused approve stays in plan mode with the editor focused: mode=%q focused=%t", m.agentMode, m.plan.Focused())
	}
	if got := lastOfficeMsg(m); !strings.Contains(got, "nothing to approve") {
		t.Fatalf("refusal notice = %q", got)
	}

	m.plan.SetValue("   \n  ") // whitespace-only is also nothing
	m = runMsg(t, m, ctrlX())
	if len(b.agentCalls) != 0 {
		t.Fatal("a whitespace-only buffer must NOT send")
	}
}

// TestPlanModeComposerSendsRouteAgent pins (d): in plan mode the office's
// NORMAL boss sends (the idle chat Enter AND the queue flush) ride
// SendAgent(text, "plan"); build-mode sends never touch the agent seam.
func TestPlanModeComposerSendsRouteAgent(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})

	// build mode first: a plain chat Enter rides the attachment/plain seam
	m = typeText(t, m, "build A")
	m = runMsg(t, m, enterKey())
	if len(b.sentTexts) != 1 || b.sentTexts[0] != "build A" {
		t.Fatalf("build-mode send must ride plain SendWith: %+v", b.sentTexts)
	}
	if len(b.agentCalls) != 0 {
		t.Fatalf("build mode never touches the agent seam: %+v", b.agentCalls)
	}

	// plan mode, editor blurred (esc) — the chat input types again, and
	// the send carries agent="plan"
	m = runMsg(t, m, ctrlP())
	m = runMsg(t, m, escKey())
	m = typeText(t, m, "plan B")
	m = runMsg(t, m, enterKey())
	if len(b.agentCalls) != 1 {
		t.Fatalf("plan-mode chat send must route the agent seam once, got %d", len(b.agentCalls))
	}
	if b.agentCalls[0] != (agentCall{text: "plan B", agent: "plan"}) {
		t.Fatalf("plan-mode send = %+v, want text+agent plan", b.agentCalls[0])
	}
	if len(b.sentTexts) != 1 {
		t.Fatalf("no extra plain send for the plan prompt: %+v", b.sentTexts)
	}

	// the QUEUE flush path routes the same way
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-9", From: "boss", Pending: true}})
	m = runMsg(t, m, enqueueMsg{text: "queued while busy", atts: nil})
	if len(m.queue) != 1 {
		t.Fatalf("the busy boss must queue the prompt, got %d", len(m.queue))
	}
	m = runMsg(t, m, queueFlushMsg{})
	if len(b.agentCalls) != 2 {
		t.Fatalf("the queue flush must ride the agent seam too, got %d agent calls", len(b.agentCalls))
	}
	if b.agentCalls[1].agent != "plan" || !strings.Contains(b.agentCalls[1].text, "queued while busy") {
		t.Fatalf("flushed plan send = %+v", b.agentCalls[1])
	}
}

// TestPlanPersistenceRoundTrip pins (e): session.json's planText survives
// a quit/relaunch into the editor buffer, the starter template never
// writes, and /new clears the buffer.
func TestPlanPersistenceRoundTrip(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m.st.Mode = state.ModeLive // persistOfficeSession is live-only
	m.sessDir = dir

	// pristine editor → the file carries NO planText
	m.persistOfficeSession(true)
	sf, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no session after persist")
	}
	if sf.PlanText != "" {
		t.Fatalf("a pristine editor must not write planText, got %q", sf.PlanText)
	}

	// a drafted plan persists and hydrates into the next boot's editor
	plan := "Plan: relaunch me\n- still drafting"
	m.plan.SetValue(plan)
	m.persistOfficeSession(true)
	sf, ok = LoadSession(dir)
	if !ok || sf.PlanText != plan {
		t.Fatalf("planText round trip: ok=%v planText=%q", ok, sf.PlanText)
	}
	m2 := New(b, nil)
	m2.hydrateSession(sf)
	if m2.plan.Value() != plan {
		t.Fatalf("hydrate restores the plan buffer, got %q", m2.plan.Value())
	}
	if m2.planText() != plan {
		t.Fatalf("a hydrated plan keeps persisting, got %q", m2.planText())
	}

	// /new clears the buffer (and with it, persistence)
	m2.newOffice()
	if m2.plan.Value() != m2.planTemplate || m2.planText() != "" {
		t.Fatalf("/new resets the plan canvas: value=%q", m2.plan.Value())
	}

	// a successful approve clears persistence too (the plan is consumed)
	m3 := New(b, nil)
	m3.plan.SetValue(plan)
	m3.setAgentMode(agentModePlan)
	m3.approvePlan()
	if m3.planText() != "" {
		t.Fatalf("an approved plan drops out of persistence, got %q", m3.planText())
	}
}

// TestStatusBarAgentBadge pins (7): the badge rides the mode segment in
// plan mode and the build default stays byte-identical to plain StatusBar.
func TestStatusBarAgentBadge(t *testing.T) {
	st := state.OfficeState{Mode: state.ModeLive, StatusLine: "live", Tick: 7}
	plain := chrome.StatusBar(st, "hint", 0, 80)
	if got := chrome.StatusBarAgent(st, "hint", 0, "", 80); got != plain {
		t.Fatal("an empty badge must be byte-identical to plain StatusBar")
	}
	badged := chrome.StatusBarAgent(st, "hint", 0, "[plan]", 80)
	if !strings.Contains(badged, "[plan]") {
		t.Fatalf("the plan badge renders: %q", badged)
	}
	if strings.Contains(plain, "[plan]") || strings.Contains(plain, "[build]") {
		t.Fatal("the default bar carries no plan/build marker")
	}
	t.Logf("statusbar plan badge segment: %q", "[plan]")
}

// TestPlanModeLayoutDesktopAndMobile pins (g): on desktop the pane owns
// the floor slot (its header line appears, the [plan] badge rides the
// statusbar); on mobile the pane takes the panel slot while the floor
// band stays on top; zen wins over plan mode entirely.
func TestPlanModeLayoutDesktopAndMobile(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	// Frame BEFORE the marker read: it is Frame that Syncs the pane's
	// size (SetSize on the concrete field) — View() measures against it.
	frame := ansi.Strip(m.Frame())
	paneMarker := firstNonEmptyLine(ansi.Strip(m.plan.View()))
	if paneMarker == "" {
		t.Fatal("the pane renders an empty view — cannot pin the layout swap")
	}
	if !strings.Contains(frame, paneMarker) {
		t.Fatalf("desktop plan mode must render the pane in the floor slot; marker %q missing", paneMarker)
	}
	if !strings.Contains(frame, "[plan]") {
		t.Fatal("desktop plan mode must show the [plan] statusbar badge")
	}
	if !strings.Contains(frame, planHint) {
		t.Fatal("plan hint must ride the statusbar in plan mode")
	}

	// mobile: the floor band stays on top, the pane sits in the panel slot
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 70, Height: 40})
	frame = ansi.Strip(m.Frame())
	paneMarker = firstNonEmptyLine(ansi.Strip(m.plan.View()))
	idx := strings.Index(frame, paneMarker)
	if idx < 0 {
		t.Fatalf("mobile plan mode must render the pane; marker %q missing", paneMarker)
	}
	paneRow := strings.Count(frame[:idx], "\n")
	if floorRow := 1 + m.floorBandH(); paneRow < floorRow {
		t.Fatalf("the pane must sit BELOW the floor band: pane row %d < band end %d", paneRow, floorRow)
	}

	// zen wins: no pane, no badge
	m.zen = true
	frame = ansi.Strip(m.Frame())
	if strings.Contains(frame, paneMarker) {
		t.Fatal("zen must hide the plan pane entirely")
	}
}

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			return ln
		}
	}
	return ""
}
