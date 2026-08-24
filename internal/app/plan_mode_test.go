// plan_mode_test.go — the plan/build agent-mode wiring end to end, the
// CONVERSATION-FIRST flow: ctrl+p toggles the mode only (chat keeps focus,
// the pane stays hidden while empty), a completed boss reply is mirrored
// passively into the pane (with the userDirty anti-clobber latch), ctrl+x
// approves from BOTH focuses (compose + agent seam + mode flip + notice,
// buffer retained for restore), the empty-pane and unedited-template
// refusals, manual-open starter arming via click, plan-mode composer sends
// routing through SendAgent(…, "plan"), the session.json planText
// round-trip (hydrate / /new / approve retention), the statusbar [plan]
// badge, the hint swaps, and the desktop/mobile layout swaps.
package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

// paneClick returns a left click inside the desktop floor slot (the plan
// pane's region) — the manual-open / focus gesture for the pane.
func paneClick() tea.MouseClickMsg {
	return tea.MouseClickMsg(tea.Mouse{X: 5, Y: 5, Button: tea.MouseLeft})
}

// bossReply drives one completed boss message through Update (the
// bossCompleted signal — plan_mode.go's presentation hook listens).
func bossReply(t *testing.T, m Model, id, text string) Model {
	t.Helper()
	return runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: id, From: "boss", Text: text}})
}

// approveDoublePress drives the F1 ctrl+x arm+fire pair end to end: the
// ARMING press goes through Update DIRECTLY (never runMsg — its own
// tea.Tick would sleep the window, then the expiry would clear the young
// arm before the second press lands); the FIRING press rides runMsg so
// the async send closure + approveSentMsg/approveErrMsg resolution
// resolve synchronously.
func approveDoublePress(t *testing.T, m Model) Model {
	t.Helper()
	nm, cmd := m.Update(ctrlX())
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("an approvable pane's first ctrl+x must arm (expiry tick returned)")
	}
	if m.approveArmAt.IsZero() {
		t.Fatal("the first ctrl+x must STAMP the approve arm")
	}
	return runMsg(t, m, ctrlX())
}

// countOfficeRows tallies transcript rows (From "office") containing sub.
func countOfficeRows(m Model, sub string) int {
	n := 0
	for _, c := range m.st.Chat {
		if c.From == "office" && strings.Contains(c.Text, sub) {
			n++
		}
	}
	return n
}

// errBackend — agentRecBackend whose agent seam can be made to fail: the
// F3 rollback playground (a rejected approve must leave plan mode ON).
type errBackend struct {
	agentRecBackend
	err error
}

func (e *errBackend) SendAgent(text, agent string) error {
	if e.err != nil {
		return e.err
	}
	e.agentCalls = append(e.agentCalls, agentCall{text: text, agent: agent})
	return nil
}

// failSendBackend — agentRecBackend whose plain attachment seam fails:
// the F6 playground (an ordinary failed send must land a red transcript
// row, not just a statusline blip).
type failSendBackend struct {
	agentRecBackend
	err error
}

func (f *failSendBackend) SendWith(text string, atts []state.Attachment) error {
	return f.err
}

// degradedBackend — agentRecBackend + the additive agentDegradeSeam, hard
// latched: the F5 badge/warning playground.
type degradedBackend struct {
	agentRecBackend
}

func (d *degradedBackend) AgentDegraded() bool { return true }

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

// TestPlanModeToggleWithExclusions pins (a): ctrl+p flips build→plan→build
// and NOTHING ELSE — chat keeps focus, the pane does not open (empty+
// hidden is the plan-mode default), sends route per-mode; a focused
// terminal tab keeps ctrl+p for the shell, and an open permission float
// keeps its keys.
func TestPlanModeToggleWithExclusions(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	if m.agentMode != agentModeBuild {
		t.Fatalf("a fresh office boots in build mode, got %q", m.agentMode)
	}

	// build → plan: mode flips ONLY — chat keeps focus, pane stays hidden
	m = runMsg(t, m, ctrlP())
	if m.agentMode != agentModePlan {
		t.Fatalf("ctrl+p must enter plan mode, got %q", m.agentMode)
	}
	if m.plan.Focused() {
		t.Fatal("conversation-first: ctrl+p must NOT steal chat focus for the pane")
	}
	if m.plan.Mode() != agentModePlan {
		t.Fatalf("the pane mirrors the mode badge, got %q", m.plan.Mode())
	}
	if m.planPaneVisible() {
		t.Fatal("an EMPTY plan-mode pane must not own the floor slot")
	}
	if got := m.hintLine(); got != planHintIdle {
		t.Fatalf("boss-idle-empty plan mode hint = %q, want %q", got, planHintIdle)
	}
	if got := lastOfficeMsg(m); !strings.Contains(got, "plan mode") {
		t.Fatalf("the toggle lands an office notice, got %q", got)
	}

	// chat typing still lands in the chat composer — the pane is untouched
	m = typeText(t, m, "plan the lobby wall")
	if m.plan.UserDirty() {
		t.Fatal("chat typing must never dirty the plan pane")
	}
	if m.plan.Value() != "" {
		t.Fatalf("chat typing must not touch the pane buffer, got %q", m.plan.Value())
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

// TestPlanBossReplyPresents pins (b): while plan mode is active a
// COMPLETED boss message mirrors into the pane (passive — chat keeps
// focus, the pane opens with the markdown); typing placeholders and
// boss-error bubbles never present; the hint swaps to the pane variant.
func TestPlanBossReplyPresents(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	// a typing placeholder opens nothing
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})
	if m.planPaneVisible() {
		t.Fatal("a typing placeholder must NOT open the pane")
	}

	bossPlan := "# Lobby plan\n1. matte panels azimuth-washed\n2. kanban lanes"
	m = bossReply(t, m, "b1", bossPlan)

	if got := m.plan.Value(); got != bossPlan {
		t.Fatalf("the boss's completed reply must mirror into the pane:\n got %q\nwant %q", got, bossPlan)
	}
	if !m.planPaneVisible() {
		t.Fatal("a presented plan owns the floor slot")
	}
	if m.plan.Focused() {
		t.Fatal("presentation is PASSIVE — chat keeps focus, the pane must not focus itself")
	}
	if m.plan.UserDirty() {
		t.Fatal("a boss-set refresh is not a user edit (the latch stays clean)")
	}
	if got := m.hintLine(); got != planHintPane {
		t.Fatalf("presented-plan hint = %q, want %q", got, planHintPane)
	}
	t.Logf("boss reply mirrored → pane buffer %q (focused=%t dirty=%t)",
		firstNonEmptyLine(m.plan.Value()), m.plan.Focused(), m.plan.UserDirty())

	// chat input still owns the keys — typing lands in the composer
	m = typeText(t, m, " tighter")
	if m.plan.Value() != bossPlan {
		t.Fatal("the pane stays untouched while chat types")
	}

	// boss-error bubbles never present over the adopted plan
	m = bossReply(t, m, "boss-error-1", "session exploded")
	if m.plan.Value() != bossPlan {
		t.Fatal("an error bubble must not clobber the presented plan")
	}

	// build mode is quiet: the SAME completion arrives out of plan mode
	m = runMsg(t, m, ctrlP()) // → build (pane buffer keeps for restore)
	m = bossReply(t, m, "b2", "# Unrelated\n- build chatter")
	if m.plan.Value() != bossPlan {
		t.Fatal("presentation is plan-mode only — build-mode replies leave the pane alone")
	}
}

// TestPlanAntiClobberKeepsUserEdit pins (c): once the user edits the pane
// (userDirty), a NEW boss completion must NOT refresh the pane — the dim
// "boss replied — your edited plan kept" note rides the office channel.
// The latch resets on approve and on a clean adoption.
func TestPlanAntiClobberKeepsUserEdit(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	v1 := "# Plan v1\n- matte panels"
	m = bossReply(t, m, "b1", v1)

	// the user clicks into the pane and edits (the anti-clobber latch)
	m = runMsg(t, m, paneClick())
	if !m.plan.Focused() {
		t.Fatal("a click inside the pane must focus it (existing swallow-routing)")
	}
	if m.plan.Value() != v1 {
		t.Fatalf("manual-open must never clobber a presented plan, got %q", firstNonEmptyLine(m.plan.Value()))
	}
	m = runMsg(t, m, pressKey('!'))
	if !m.plan.UserDirty() {
		t.Fatal("a user edit must latch userDirty")
	}
	edited := v1 + "!"
	t.Logf("user edited the pane → buffer tail %q (dirty=%t)", m.plan.Value(), m.plan.UserDirty())

	// the boss replies again — the user's edit SURVIVES; the note lands
	v2 := "# Plan v2\n- glassmorphic lanes"
	m = bossReply(t, m, "b2", v2)
	t.Logf("boss v2 arrived (%q) → pane kept %q", firstNonEmptyLine(v2), m.plan.Value())
	if m.plan.Value() != edited {
		t.Fatalf("anti-clobber: the user's edit must survive the boss's reply:\n got %q\nwant %q", m.plan.Value(), edited)
	}
	if got := lastOfficeMsg(m); got != planKeptNotice {
		t.Fatalf("the anti-clobber notice must ride the office channel: got %q, want %q", got, planKeptNotice)
	}

	// esc back to chat (buffer keeps), then approve CONSUMES the latch:
	// the edit is what gets signed off (F1: the arm + fire double-press)
	m = runMsg(t, m, escKey())
	if m.plan.Focused() {
		t.Fatal("esc must blur the editor back to chat")
	}
	if m.plan.Value() != edited {
		t.Fatal("esc keeps the plan buffer")
	}
	m = approveDoublePress(t, m)
	if len(b.agentCalls) != 1 || b.agentCalls[0].text != approvePrefix+edited {
		t.Fatalf("the EDITED plan is what ships to build: %+v", b.agentCalls)
	}
	if m.plan.UserDirty() {
		t.Fatal("approve resets the dirty latch")
	}
	t.Logf("approve consumed the latch → pane buffer persisted for restore (%q), dirty=%t",
		firstNonEmptyLine(m.plan.Value()), m.plan.UserDirty())
}

// TestPlanApproveHappyPath pins (d): a presented plan + ctrl+x from the
// CHAT input (focus stays in the composer the whole time) sends the
// composed prompt through the agent seam with agent="build", flips the
// office back to build, RETAINS the buffer (restore on re-entry), resets
// the latch, and lands the notice.
func TestPlanApproveHappyPath(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	body := "Plan: plan/build modes for the office\n1. wire the agent seam\n2. swap the floor slot"
	m = bossReply(t, m, "b1", body)

	m = approveDoublePress(t, m) // chat-focused ctrl+x twice — F1 arm → fire
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
	if m.plan.Value() != body {
		t.Fatal("the plan buffer is RETAINED for restore (pane hides with the mode flip)")
	}
	if m.plan.UserDirty() {
		t.Fatal("approve resets the dirty latch")
	}
	notice := lastOfficeMsg(m)
	wantNotice := fmt.Sprintf("[office] plan approved — sent to build (%d chars)", len(body))
	if notice != wantNotice {
		t.Fatalf("approve notice = %q, want %q", notice, wantNotice)
	}
	// the pane hides (build never shows it) and the plan KEEPS persisting
	if m.planPaneVisible() {
		t.Fatal("build mode never shows the pane")
	}
	if m.planText() != body {
		t.Fatalf("an approved plan keeps persisting for restore, got %q", m.planText())
	}
}

// TestPlanApproveFromEditorFocus pins (e): ctrl+x works from the
// editor-focused surface too (the wave-34 claim), with the same compose
// and flip as the chat-focused path.
func TestPlanApproveFromEditorFocus(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	body := "# Plan\n- approve me from the editor"
	m = bossReply(t, m, "b1", body)
	m = runMsg(t, m, paneClick())
	if !m.plan.Focused() {
		t.Fatal("setup: the pane must be focused for the editor-focus claim")
	}
	m = approveDoublePress(t, m)
	if len(b.agentCalls) != 1 {
		t.Fatalf("editor-focused approve must send once, got %d calls", len(b.agentCalls))
	}
	if b.agentCalls[0] != (agentCall{text: approvePrefix + body, agent: agentModeBuild}) {
		t.Fatalf("editor-focused approve = %+v", b.agentCalls[0])
	}
	if m.agentMode != agentModeBuild || m.plan.Focused() {
		t.Fatalf("approve flips to build and blurs: mode=%q focused=%t", m.agentMode, m.plan.Focused())
	}
	t.Logf("editor-focused approve wire: SendAgent(%q…, agent=%q)", strings.SplitN(body, "\n", 2)[0], b.agentCalls[0].agent)
}

// TestPlanApproveRefusesUnedited pins (f): ctrl+x with NO plan (empty,
// hidden pane) earns the dim refusal no-op; the manually-OPENED starter
// template is the wave-34 refusal (untouched boilerplate must not spend a
// build turn); whitespace-only is nothing too. F1: refusals land BEFORE
// the double-press arm — a refusal press NEVER stamps approveArmAt.
func TestPlanApproveRefusesUnedited(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	// (a) no plan presented at all — pane empty+hidden: ctrl+x is a no-op
	m = runMsg(t, m, ctrlX())
	if len(b.agentCalls) != 0 {
		t.Fatal("an empty pane must NOT send")
	}
	if m.agentMode != agentModePlan {
		t.Fatalf("a refused approve stays in plan mode: %q", m.agentMode)
	}
	if got := lastOfficeMsg(m); !strings.Contains(got, "nothing to approve") {
		t.Fatalf("empty-pane refusal notice = %q", got)
	}
	if !m.approveArmAt.IsZero() {
		t.Fatal("a refusal lands BEFORE the arm — nothing stamps")
	}
	if m.planPaneVisible() {
		t.Fatal("the refusal leaves the floor alone (pane still hidden)")
	}

	// (b) the manually-opened starter template refuses (unchanged = nothing
	// to sign off) — the template arms on click into an empty pane
	m = runMsg(t, m, paneClick())
	if !m.plan.IsStarter() {
		t.Fatalf("a click into an empty pane must arm the starter scaffold, got %q", firstNonEmptyLine(m.plan.Value()))
	}
	if !m.planPaneVisible() {
		t.Fatal("manual-open shows the pane (the scaffold is content)")
	}
	m = runMsg(t, m, ctrlX())
	if len(b.agentCalls) != 0 {
		t.Fatal("an untouched template must NOT send")
	}
	if m.agentMode != agentModePlan || !m.plan.Focused() {
		t.Fatalf("a refused approve stays in plan mode with the editor focused: mode=%q focused=%t", m.agentMode, m.plan.Focused())
	}
	if got := lastOfficeMsg(m); !strings.Contains(got, "nothing to approve") {
		t.Fatalf("template refusal notice = %q", got)
	}
	if !m.approveArmAt.IsZero() {
		t.Fatal("a template refusal lands BEFORE the arm — nothing stamps")
	}

	// (c) a whitespace-only scratch buffer is nothing too
	m.plan.SetValue("   \n  ")
	m = runMsg(t, m, ctrlX())
	if len(b.agentCalls) != 0 {
		t.Fatal("a whitespace-only buffer must NOT send")
	}
}

// TestPlanModeComposerSendsRouteAgent pins (g): in plan mode the office's
// NORMAL boss sends (the idle chat Enter AND the queue flush) ride
// SendAgent(text, "plan"); build-mode sends never touch the agent seam.
// The chat input keeps focus the whole time (the pane never steals it).
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

	// plan mode — chat keeps focus straight off the toggle; the send
	// carries agent="plan"
	m = runMsg(t, m, ctrlP())
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

// TestPlanPersistenceRoundTrip pins (h): session.json's planText survives
// a quit/relaunch into the editor buffer (a boot lands in BUILD mode —
// re-entering plan mode shows it), an empty/starter buffer never writes,
// /new clears the canvas to EMPTY, and an approved plan keeps persisting
// (its buffer is retained for restore).
func TestPlanPersistenceRoundTrip(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m.st.Mode = state.ModeLive // persistOfficeSession is live-only
	m.sessDir = dir

	// pristine (EMPTY) editor → the file carries NO planText
	m.persistOfficeSession(true)
	sf, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no session after persist")
	}
	if sf.PlanText != "" {
		t.Fatalf("a pristine editor must not write planText, got %q", sf.PlanText)
	}

	// a presented plan persists and hydrates into the next boot's pane
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
	// the boot rests in build; a ctrl+p re-entry shows the restored plan
	if m2.planPaneVisible() {
		t.Fatal("a boot never lands with the pane open (build mode)")
	}
	m2.setAgentMode(agentModePlan)
	if !m2.planPaneVisible() {
		t.Fatal("a restored plan owns the floor slot on plan-mode re-entry")
	}

	// /new clears the canvas to EMPTY (and with it, persistence)
	m2.newOffice()
	if m2.plan.Value() != m2.planTemplate || m2.planText() != "" {
		t.Fatalf("/new resets the plan canvas: value=%q", m2.plan.Value())
	}
	if strings.TrimSpace(m2.plan.Value()) != "" {
		t.Fatalf("/new clears to EMPTY (the conversation-first rest state), got %q", m2.plan.Value())
	}

	// a successful approve RETAINS the buffer — persistence keeps it for
	// the restore-on-re-entry story
	m3 := New(b, nil)
	m3.plan.SetValue(plan)
	m3.setAgentMode(agentModePlan)
	m3.approvePlan()
	if m3.planText() != plan {
		t.Fatalf("an approved plan keeps persisting for restore, got %q", m3.planText())
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

// TestPlanModeLayoutDesktopAndMobile pins (i): ctrl+p ALONE swaps nothing
// (empty+hidden — the normal floor/panel stack renders, the [plan] badge +
// idle hint ride the statusbar); a presented plan owns the floor slot
// (desktop) / the panel slot under the band (mobile); zen wins over plan
// mode entirely.
func TestPlanModeLayoutDesktopAndMobile(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	// EMPTY plan mode: no pane header anywhere; the badge + idle hint ride
	frame := ansi.Strip(m.Frame())
	if strings.Contains(frame, "PLAN · markdown") {
		t.Fatal("empty plan mode must NOT render the pane — chat-first means floor-first")
	}
	if !strings.Contains(frame, "[plan]") {
		t.Fatal("plan mode must show the [plan] statusbar badge even with the pane hidden")
	}
	if got := m.hintLine(); got != planHintIdle {
		t.Fatalf("idle hint rides the statusbar: %q", got)
	}

	// boss presents → the pane owns the floor slot (desktop)
	m = bossReply(t, m, "b1", "# Lobby plan\n- matte panels azimuth-washed")
	// Frame BEFORE the marker read: it is Frame that Syncs the pane's
	// size (SetSize on the concrete field) — View() measures against it.
	frame = ansi.Strip(m.Frame())
	paneMarker := firstNonEmptyLine(ansi.Strip(m.plan.View()))
	if paneMarker == "" {
		t.Fatal("the pane renders an empty view — cannot pin the layout swap")
	}
	if !strings.Contains(frame, paneMarker) {
		t.Fatalf("desktop plan mode must render the presented pane in the floor slot; marker %q missing", paneMarker)
	}
	if !strings.Contains(frame, "[plan]") {
		t.Fatal("desktop plan mode must show the [plan] statusbar badge")
	}
	if got := m.hintLine(); got != planHintPane {
		t.Fatalf("presented-plan hint rides the statusbar: %q", got)
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

// officeRowByText finds the newest office transcript row containing sub.
func officeRowByText(m Model, sub string) (state.ChatMsg, bool) {
	for i := len(m.st.Chat) - 1; i >= 0; i-- {
		c := m.st.Chat[i]
		if c.From == "office" && strings.Contains(c.Text, sub) {
			return c, true
		}
	}
	return state.ChatMsg{}, false
}

// --- F1: the ctrl+x approve-arm double press ---------------------------------

// TestApproveArmFlow pins the arm→fire pair: the FIRST ctrl+x stamps the
// arm + swaps the hint bar to the warn toast + sends NOTHING; the second
// press inside approveArmWindow fires the approve (send + flip + toast
// retires). Ctrl+x claims from the chat focus here; the editor-focus twin
// rides TestPlanApproveFromEditorFocus.
func TestApproveArmFlow(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	body := "# Arm plan\n- first arms, second fires"
	m = bossReply(t, m, "b1", body)

	// (a) ARMING press — direct Update: runMsg would execute the arm's own
	// tea.Tick (1.5s sleep + expiry clearing the young arm).
	nm, cmd := m.Update(ctrlX())
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("the arming press must schedule its own expiry tick")
	}
	if m.approveArmAt.IsZero() {
		t.Fatal("the first ctrl+x must STAMP the approve arm")
	}
	if hint := m.hintLine(); !strings.Contains(hint, approveArmToast) {
		t.Fatalf("the armed hint bar must swap to the approve toast, got %q", hint)
	}
	if len(b.agentCalls) != 0 {
		t.Fatalf("an arm sends NOTHING, got %+v", b.agentCalls)
	}
	if m.agentMode != agentModePlan {
		t.Fatalf("an arm never flips the mode, got %q", m.agentMode)
	}

	// (b) the second press inside the window FIRES.
	m = runMsg(t, m, ctrlX())
	if !m.approveArmAt.IsZero() {
		t.Fatal("the firing press clears the arm")
	}
	if len(b.agentCalls) != 1 {
		t.Fatalf("the fire sends once through the agent seam, got %d calls", len(b.agentCalls))
	}
	if b.agentCalls[0] != (agentCall{text: approvePrefix + body, agent: agentModeBuild}) {
		t.Fatalf("fired approve = %+v", b.agentCalls[0])
	}
	if m.agentMode != agentModeBuild {
		t.Fatalf("the fire flips to build on acceptance, got %q", m.agentMode)
	}
	if hint := m.hintLine(); strings.Contains(hint, approveArmToast) {
		t.Fatalf("the approve toast retires with the arm, got %q", hint)
	}
	t.Logf("F1 trace: ctrl+x → arm(stamp=%v, toast=%q) → ctrl+x → SendAgent(agent=%q) → mode=%q",
		!m.approveArmAt.IsZero(), approveArmToast, b.agentCalls[0].agent, m.agentMode)
}

// TestApproveArmOtherKeyDisarms pins: ANY other key press clears a live
// approve arm (its toast retires) — an armed ctrl+x can never surprise-
// fire behind later typing.
func TestApproveArmOtherKeyDisarms(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())
	m = bossReply(t, m, "b1", "# Disarm plan\n- typing kills the arm")

	nm, cmd := m.Update(ctrlX())
	m = nm.(Model)
	if cmd == nil || m.approveArmAt.IsZero() {
		t.Fatal("precondition: the first press armed")
	}

	m = runMsg(t, m, pressKey('z'))
	if !m.approveArmAt.IsZero() {
		t.Fatal("any other key press must disarm the approve arm")
	}
	if hint := m.hintLine(); strings.Contains(hint, approveArmToast) {
		t.Fatalf("the toast retires with the disarm, got %q", hint)
	}
	if len(b.agentCalls) != 0 {
		t.Fatalf("a disarmed arm never fires, got %+v", b.agentCalls)
	}

	// and a fresh pair still works (no stale state left behind)
	m = approveDoublePress(t, m)
	if len(b.agentCalls) != 1 {
		t.Fatalf("a fresh arm+fire pair works after the disarm, got %d calls", len(b.agentCalls))
	}
}

// TestApproveArmStaleReArms pins: a STALE first press (older than
// approveArmWindow) can't pair — the next press re-opens a FRESH arm (the
// quitarm time-injection idiom), and only the press after that fires.
func TestApproveArmStaleReArms(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())
	m = bossReply(t, m, "b1", "# Stale plan\n- old arms never pair")

	nm, _ := m.Update(ctrlX())
	m = nm.(Model)
	if m.approveArmAt.IsZero() {
		t.Fatal("precondition: the first press armed")
	}
	m.approveArmAt = time.Now().Add(-approveArmWindow - time.Second) // stale the opener

	// press 2: outside the window — NO fire, and this press becomes the
	// fresh opener.
	nm, _ = m.Update(ctrlX())
	m = nm.(Model)
	if m.approveArmAt.IsZero() || time.Since(m.approveArmAt) > approveArmWindow {
		t.Fatalf("a stale pair must RE-ARM with a fresh stamp, got %v", m.approveArmAt)
	}
	if len(b.agentCalls) != 0 {
		t.Fatalf("a stale pair must NOT fire, got %+v", b.agentCalls)
	}
	if m.agentMode != agentModePlan {
		t.Fatalf("a stale pair stays in plan mode, got %q", m.agentMode)
	}

	// press 3 completes the FRESH pair.
	m = runMsg(t, m, ctrlX())
	if len(b.agentCalls) != 1 || m.agentMode != agentModeBuild {
		t.Fatalf("the press after the re-arm must fire: calls=%+v mode=%q", b.agentCalls, m.agentMode)
	}
}

// TestApproveArmTickExpiryClears pins: the arm's own expiry tick
// (approveArmClearMsg) clears an arm old enough and leaves a YOUNGER
// re-arm alone (its own tick owns its expiry).
func TestApproveArmTickExpiryClears(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())
	m = bossReply(t, m, "b1", "# Expiry plan\n- ticks own windows")

	// young arm: a stale tick landing early must NOT clear it.
	nm, _ := m.Update(ctrlX())
	m = nm.(Model)
	nm, _ = m.Update(approveArmClearMsg{})
	m = nm.(Model)
	if m.approveArmAt.IsZero() {
		t.Fatal("a tick landing early must not clear a YOUNG approve arm")
	}

	// old arm: the real expiry case.
	m.approveArmAt = time.Now().Add(-approveArmWindow) // exactly aged out
	nm, _ = m.Update(approveArmClearMsg{})
	m = nm.(Model)
	if !m.approveArmAt.IsZero() {
		t.Fatal("the expiry tick must clear an approve arm old enough")
	}
	if hint := m.hintLine(); strings.Contains(hint, approveArmToast) {
		t.Fatalf("the toast retires with the expired arm, got %q", hint)
	}
}

// TestApproveArmToastCopyFrozen pins the toast + window copy contract.
func TestApproveArmToastCopyFrozen(t *testing.T) {
	if approveArmToast != "ctrl+x again: approve plan + switch to build" {
		t.Fatalf("the approve-arm toast copy is frozen, got %q", approveArmToast)
	}
	if approveArmWindow != quitArmWindow {
		t.Fatalf("the two armed windows are one contract: approve=%v quit=%v", approveArmWindow, quitArmWindow)
	}
}

// --- F2: a restored plan refuses an untouched approve -------------------------

// TestApproveRestoredRefusesUntilEdited pins F2: a plan buffer hydrated
// from session.json (restoredPlan) refuses the approve ARM with the dim
// restored notice — no stamp, no send, mode kept — until the member edits
// it (userDirty lifts the gate); the accept path clears the latch.
func TestApproveRestoredRefusesUntilEdited(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	plan := "Restored: lobby wall\n- matte panels"
	m.hydrateSession(&SessionFile{PlanText: plan})
	if !m.restoredPlan {
		t.Fatal("hydrate must latch restoredPlan when planText seeds the buffer")
	}
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())
	if !m.planPaneVisible() {
		t.Fatal("sanity: restored content owns the floor slot in plan mode")
	}

	// untouched: even repeated ctrl+x presses REFUSE before the arm
	for i := 0; i < 2; i++ {
		m = runMsg(t, m, ctrlX())
		if got := lastOfficeMsg(m); got != planRestoredNotice {
			t.Fatalf("press %d restored refusal notice = %q, want %q", i+1, got, planRestoredNotice)
		}
		if !m.approveArmAt.IsZero() {
			t.Fatalf("press %d: the restored refusal lands BEFORE the arm — nothing stamps", i+1)
		}
		if len(b.agentCalls) != 0 {
			t.Fatalf("press %d: an untouched restored plan must NOT send, got %+v", i+1, b.agentCalls)
		}
		if m.agentMode != agentModePlan {
			t.Fatalf("press %d: the refusal keeps plan mode, got %q", i+1, m.agentMode)
		}
	}

	// open + edit: the gate lifts (userDirty) and the accept clears the latch
	m = runMsg(t, m, paneClick())
	m = runMsg(t, m, pressKey('!'))
	if !m.plan.UserDirty() {
		t.Fatal("an edit must latch userDirty (the gate's lift)")
	}
	m = approveDoublePress(t, m)
	if len(b.agentCalls) != 1 {
		t.Fatalf("an edited restored plan approves, got %d calls", len(b.agentCalls))
	}
	if b.agentCalls[0].text != approvePrefix+plan+"!" {
		t.Fatalf("the EDITED restore ships: %+v", b.agentCalls[0])
	}
	if m.restoredPlan {
		t.Fatal("a successful approve clears the restored latch")
	}
	if m.agentMode != agentModeBuild {
		t.Fatalf("the edited restore flips to build, got %q", m.agentMode)
	}
}

// --- F3: the flip rides SEND ACCEPTANCE ---------------------------------------

// TestApproveRollbackOnSendError pins F3: when the approve's wire send
// fails, the office STAYS in plan mode (no stale flip) with a red
// transcript row — and the failure is the tagged approveErrMsg twin (no
// cross-talk with an ordinary send failure). When the wire recovers the
// retry flips on acceptance exactly like a first-time approve.
func TestApproveRollbackOnSendError(t *testing.T) {
	b := &errBackend{err: errors.New("wire down")}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	body := "# Rollback plan\n- keep the wiring intact"
	m = bossReply(t, m, "b1", body)

	nm, cmd := m.Update(ctrlX())
	m = nm.(Model)
	if cmd == nil || m.approveArmAt.IsZero() {
		t.Fatal("precondition: approvable pane arms on the first press")
	}
	m = runMsg(t, m, ctrlX()) // fire → SendAgent errors → approveErrMsg

	if m.agentMode != agentModePlan {
		t.Fatalf("F3 rollback: a failed approve must KEEP plan mode, got %q", m.agentMode)
	}
	if m.plan.Mode() != agentModePlan {
		t.Fatalf("the pane badge mirrors the kept mode, got %q", m.plan.Mode())
	}
	if got := m.plan.Value(); got != body {
		t.Fatalf("the plan buffer survives the rollback untouched, got %q", got)
	}
	row, ok := officeRowByText(m, "approve failed — still in plan: wire down")
	if !ok {
		t.Fatalf("the red approve-failed row must land in the transcript, rows: %+v", m.st.Chat)
	}
	if row.Meta != "error" {
		t.Fatalf("the approve-failed row is the red error class, got Meta=%q", row.Meta)
	}
	// no cross-talk: an approve failure is NOT the ordinary send failure
	if n := countOfficeRows(m, "send failed:"); n != 0 {
		t.Fatalf("approveErrMsg must not trip sendErrMsg's row, got %d", n)
	}
	if n := countOfficeRows(m, "plan approved — sent to build"); n != 0 {
		t.Fatalf("no optimistic approval notice on a failed wire, got %d", n)
	}
	t.Logf("F3 rollback trace: SendAgent → %q → approveErrMsg → mode stays %q, red row %q",
		"wire down", m.agentMode, row.Text)

	// the wire recovers: the retry flips on acceptance (exact fire path)
	b.err = nil
	m = approveDoublePress(t, m)
	if len(b.agentCalls) != 1 {
		t.Fatalf("the retry sends once, got %+v", b.agentCalls)
	}
	if m.agentMode != agentModeBuild {
		t.Fatalf("the flip rides acceptance, got %q", m.agentMode)
	}
	if n := countOfficeRows(m, "plan approved — sent to build"); n != 1 {
		t.Fatalf("the approval notice posts on acceptance, got %d", n)
	}
	t.Logf("F3 acceptance trace: SendAgent → nil → approveSentMsg → mode=%q (notice posted)", m.agentMode)
}

// --- F4: mid-turn flip transparency -------------------------------------------

// TestPlanCompletionNoteOnMidTurnExit pins the planSendPending counter: a
// plan-mode send outstanding while the member reflex-flips back to build
// leaves the boss's completed reply in chat only (the pane never opens) —
// the exit notice gains the in-flight suffix and the completion lands the
// ONCE-per-turn dim note. A plan-mode completion (no flip) notes nothing.
func TestPlanCompletionNoteOnMidTurnExit(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})

	// control: a build-mode completion with NO plan send never notes
	m = bossReply(t, m, "b0", "# Unrelated\n- build chatter")
	if n := countOfficeRows(m, planLandedInChat); n != 0 {
		t.Fatalf("no counter → no note, got %d", n)
	}

	// a plan-tagged send goes out; the boss is typing
	m = runMsg(t, m, ctrlP())
	m = typeText(t, m, "plan the doorway")
	m = runMsg(t, m, enterKey())
	if m.planSendPending != 1 {
		t.Fatalf("an accepted plan send tallies the counter, got %d", m.planSendPending)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})

	// the reflex flip mid-turn: exit notice gains the in-flight suffix
	m = runMsg(t, m, ctrlP())
	if got := lastOfficeMsg(m); !strings.Contains(got, "; an in-flight reply lands in chat") {
		t.Fatalf("exit-with-in-flight notice = %q", got)
	}

	// the boss's reply completes AFTER the flip: pane stays shut, the dim
	// note posts ONCE, the counter closes.
	m = bossReply(t, m, "b1", "# Door plan\n- hinge")
	if n := countOfficeRows(m, planLandedInChat); n != 1 {
		t.Fatalf("the land-in-chat note must post exactly once, got %d", n)
	}
	if m.planSendPending != 0 {
		t.Fatalf("the completion consumes the tally, got %d", m.planSendPending)
	}

	// a second completion posts nothing extra (once per turn-edge)
	m = bossReply(t, m, "b2", "# Door plan v2\n- knob")
	if n := countOfficeRows(m, planLandedInChat); n != 1 {
		t.Fatalf("the note is once per turn-edge, got %d", n)
	}

	// an IN-PLAN completion presents and notes nothing
	m = runMsg(t, m, ctrlP())
	m = typeText(t, m, "plan the threshold")
	m = runMsg(t, m, enterKey())
	if m.planSendPending != 1 {
		t.Fatalf("the second plan send tallies again, got %d", m.planSendPending)
	}
	m = bossReply(t, m, "b3", "# Threshold plan\n- flush")
	if n := countOfficeRows(m, planLandedInChat); n != 1 {
		t.Fatalf("an in-plan completion never notes, got %d", n)
	}
	if m.planSendPending != 0 {
		t.Fatalf("the in-plan completion still consumes the tally, got %d", m.planSendPending)
	}
	if !strings.Contains(m.plan.Value(), "Threshold") {
		t.Fatal("the in-plan completion presents into the pane as before")
	}
}

// --- F5: degrade visibility ----------------------------------------------------

// TestAgentDegradeBadgeAndWarning pins F5b: a backend carrying
// agentDegradeSeam latched ON renders "[plan·degraded]" in plan mode, and
// entering plan mode posts the one-time-per-session red warning.
func TestAgentDegradeBadgeAndWarning(t *testing.T) {
	b := &degradedBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	if got := m.agentBadge(); got != "" {
		t.Fatalf("build mode carries no badge, got %q", got)
	}

	m = runMsg(t, m, ctrlP())
	if got := m.agentBadge(); got != "[plan·degraded]" {
		t.Fatalf("a degraded plan badge = %q, want %q", got, "[plan·degraded]")
	}
	row, ok := officeRowByText(m, "can't route them")
	if !ok {
		t.Fatal("entering plan on a degraded serve posts the entry warning")
	}
	if row.Meta != "error" {
		t.Fatalf("the degrade warning is the red class, got Meta=%q", row.Meta)
	}
	if row.Text != planDegradeWarn {
		t.Fatalf("the warning copy = %q, want %q", row.Text, planDegradeWarn)
	}
	t.Logf("F5 badge: %q; warning row: %q", m.agentBadge(), row.Text)

	// once per session: exit + re-enter posts no second warning
	m = runMsg(t, m, ctrlP())
	m = runMsg(t, m, ctrlP())
	if n := countOfficeRows(m, "can't route them"); n != 1 {
		t.Fatalf("the entry warning is one-time-per-session, got %d", n)
	}
	if got := m.agentBadge(); got != "[plan·degraded]" {
		t.Fatalf("the badge degraded flag stays honest every frame, got %q", got)
	}

	// a NON-degraded backend badges + warns nothing
	clean := &agentRecBackend{}
	m2 := New(clean, nil)
	m2 = runMsg(t, m2, tea.WindowSizeMsg{Width: 140, Height: 40})
	m2 = runMsg(t, m2, ctrlP())
	if got := m2.agentBadge(); got != "[plan]" {
		t.Fatalf("a healthy serve keeps the plain badge, got %q", got)
	}
	if n := countOfficeRows(m2, "can't route them"); n != 0 {
		t.Fatalf("a healthy serve warns nothing, got %d", n)
	}
}

// TestAgentFieldStatusEscalatesToTranscript pins F5a: a backend EvStatus
// carrying the agent-field marker ALSO lands as a red transcript office
// row; an ordinary EvStatus stays statusline-only.
func TestAgentFieldStatusEscalatesToTranscript(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})

	note := "[theboringoffice] agent-field: plan/build agent field unavailable on this serve (400 rejected the agent field) — retried this prompt without it; future prompts skip it"
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: note})
	row, ok := officeRowByText(m, "agent field unavailable")
	if !ok {
		t.Fatalf("the marked status must escalate into the transcript, rows: %+v", m.st.Chat)
	}
	if row.Meta != "error" {
		t.Fatalf("the escalated row is the red error class, got Meta=%q", row.Meta)
	}
	if row.Text != note {
		t.Fatalf("the escalated row carries the status verbatim:\n got %q\nwant %q", row.Text, note)
	}
	if m.st.StatusLine != note {
		t.Fatalf("the statusline twin is untouched, got %q", m.st.StatusLine)
	}

	// a plain (unmarked) status never double-posts
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[theboringoffice] live - http://127.0.0.1:9999 | board: git"})
	if n := countOfficeRows(m, "[theboringoffice]"); n != 1 {
		t.Fatalf("unmarked statuses stay statusline-only (office rows=%d)", n)
	}
}

// --- F6: an ordinary failed send is transcript-visible --------------------------

// TestSendFailurePostsTranscriptRow pins F6: a rejected plain send posts
// the red transcript row IN ADDITION to the transient statusline twin.
func TestSendFailurePostsTranscriptRow(t *testing.T) {
	b := &failSendBackend{err: errors.New("wire snapped")}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})

	m = typeText(t, m, "ship it")
	m = runMsg(t, m, enterKey())

	row, ok := officeRowByText(m, "send failed: wire snapped")
	if !ok {
		t.Fatalf("the failed send must land a red transcript row, rows: %+v", m.st.Chat)
	}
	if row.Meta != "error" {
		t.Fatalf("the send-failed row is the red error class, got Meta=%q", row.Meta)
	}
	if !strings.Contains(m.st.StatusLine, "send failed") {
		t.Fatalf("the statusline twin is untouched, got %q", m.st.StatusLine)
	}
	t.Logf("F6 transcript row: %q (Meta=%q)", row.Text, row.Meta)
}
