// plan_mode.go — the plan/build agent mode ("agentMode"): the member's
// switch between steering the boss's PLANNING pass (prompts ride
// SendAgent(text,"plan")) and the normal build pipeline (plain Send — a
// build-mode prompt never carries an "agent" key on the wire).
//
// The flow is CONVERSATION-FIRST: ctrl+p only flips the mode (chat keeps
// focus, sends route per-mode as before) — the plan pane stays hidden
// until it has content. A COMPLETED boss reply while plan mode is active
// is mirrored passively into the pane (the member keeps typing) ONLY when
// it LOOKS like a plan — looksLikePlan: a markdown structure signal +
// document heft. Boss chatter/status narration never presents: the pane
// keeps its current content and a dim, debounced transcript note explains
// (see notePlanShapeRejection). The pane owns the floor slot exactly when
// plan mode is on AND the buffer is non-empty (boss-presented, user-
// edited, or persisted-restored). A click inside the pane arms the
// starter scaffold and focuses editing — userDirty tracks those edits so
// a fresh boss reply never clobbers them. Persistence rides the same
// gate: chatter-shaped buffers never survive a boot.
//
// State lives MODEL-side (never OfficeState — state.Mode already means
// live/demo): m.agentMode is "plan"|"build", mirrored into the plan pane
// via setAgentMode so the pane renders its own mode badge AND the chat
// send closure (built once in New, before any Model copy exists) reads
// the CURRENT mode at send time through the pointer-shared pane.
//
// The pane is the colleague's contract-frozen panels.PlanEditor: the app
// guards the shape with the compile assert below and drives it through a
// typed field. SetSize stays OUT of the asserted interface — its fluent
// *panels.PlanEditor return cannot ride an interface method, so the model
// sizes the pane on the concrete field at Frame time instead.
package app

import (
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// Agent-mode ids. state.Mode (live/demo) is untouched — these ids ride
// MODEL state and the wire's "agent" field only.
const (
	agentModePlan  = "plan"
	agentModeBuild = "build"
)

// planEditorPane — the plan editor's contract as the app drives it
// (panels.PlanEditor implements EXACTLY this). SetSize is deliberately
// absent: the colleague's SetSize(w,h) returns *panels.PlanEditor
// (fluent), which an interface method cannot express — the model calls it
// on the concrete field at layout time instead.
type planEditorPane interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Focus() tea.Cmd
	Blur()
	Focused() bool
	Value() string
	SetValue(string)
	Mode() string
	SetMode(m string)
}

// The colleague's pane must satisfy the app-side shape — drift fails the
// build HERE, not at an assert deep in the layout code.
var _ planEditorPane = (*panels.PlanEditor)(nil)

// agentBackend — the plan/build routing seam live and demo backends expose
// beyond state.Backend (the same additive type-assert pattern as
// attachmentBackend in model.go; harness stubs without it degrade to the
// plain-text send). SendAgent sends one prompt with the agent tag riding
// the payload ("plan"|"build"); backends keep the key out of the payload
// entirely when a serve rejects it (live backend's degrade latch).
type agentBackend interface {
	SendAgent(text, agent string) error
}

// approvePrefix — the fixed compose head for an approved plan: a plan the
// member signs off leaves the editor and becomes a BUILD-agent prompt.
// Frozen copy — pinned by plan_mode_test.go.
const approvePrefix = "Approved plan — implement it exactly as specified:\n\n"

// Statusline hint swaps for plan mode (hintLine picks per pane
// visibility): boss-idle-empty means the pane is hidden and the member
// just talks; pane-visible means a presented/edited plan sits in the
// floor slot. Frozen copy — pinned by plan_mode_test.go.
const (
	planHintIdle = "plan · boss plans read-only · ctrl+p exits · ctrl+x twice approves a presented plan"
	planHintPane = "plan · click to edit · ctrl+x twice: approve → build · ctrl+p exits"
)

// planKeptNotice — the anti-clobber transcript note: a fresh boss reply
// arrived while the pane carries USER edits — the edits win, the pane is
// not touched. Frozen copy — pinned by plan_mode_test.go.
const planKeptNotice = "boss replied — your edited plan kept"

// Shape-gate transcript copies (frozen — pinned by plan_shape_test.go):
//
//   - planNotPlanNotice: a completed boss reply FAILED the looksLikePlan
//     gate while the pane carries real content — the pane keeps its last
//     plan and the dim note says why. Debounced: at most once per boss
//     turn (message ID) and never twice for the same chatter text.
//   - planChatValveNotice: the escape valve — a chatter reply arrived
//     while the pane shows NOTHING and no plan has presented yet this
//     session (first-time plan mode: the boss is talking but hasn't
//     planned). Fires ONCE per office session (the latch rides the
//     pane-keyed gate state).
const (
	planNotPlanNotice   = "boss's reply didn't look like a plan — the pane kept its last plan (click to edit ¦ ctrl+p exits ¦ ctrl+x twice approves a presented plan)"
	planChatValveNotice = "boss is chatting; when it writes a plan it lands on the left"
)

// approveArmToast — the high-visibility statusbar line swapped in while a
// ctrl+x approve arm is live (the FIRST press arms instead of firing; the
// second press inside approveArmWindow approves + flips to build). Mirrors
// the ctrl+q quitArmToast pattern exactly. Frozen copy — pinned by
// plan_mode_test.go.
const approveArmToast = "ctrl+x again: approve plan + switch to build"

// Refusal/notice copies (frozen — pinned by plan_mode_test.go):
//
//   - planNothingNotice: ctrl+x with no plan presented (empty+hidden pane).
//   - planStarterNotice: ctrl+x on the never-edited starter template.
//   - planRestoredNotice: F2 — a restored-from-session buffer REFUSES the
//     arm until the member has touched it.
//   - planLandedInChat: F4 — the boss's reply for a plan-tagged send
//     completed while the office had already flipped back to build (the
//     pane never opens then — the transcript carries the tell instead).
//   - planDegradeWarn: F5 — one-time-per-session entry warning when the
//     serve rejected the plan/build agent field (labels only, no routing).
const (
	planNothingNotice  = "[office] nothing to approve — the boss hasn't presented a plan yet · click the plan pane to scratch one · ctrl+p exits"
	planStarterNotice  = "[office] nothing to approve — edit the plan, then ctrl+x twice (ctrl+p exits plan mode)"
	planRestoredNotice = "plan restored from your last session — open it (click), edit, then ctrl+x twice"
	planLandedInChat   = "boss reply landed in chat (plan mode was off when it completed)"
	planDegradeWarn    = "this serve rejected the agent field — plan mode labels sends but can't route them (retry bare)"
)

// agentFieldStatusMarker — the string contract between the live backend's
// degrade latch note (opencode.go's postPrompt) and the app's transcript
// escalation (F5a): an EvStatus text carrying this prefix ALSO lands in
// the transcript as a red office row, because the statusline-only note is
// overwritten by the next status event.
const agentFieldStatusMarker = "[theboringoffice] agent-field:"

// approveArmWindow — the ctrl+x double-press window (identical to
// quitArmWindow by design: the two armed-destructive keys feel the same).
const approveArmWindow = 1500 * time.Millisecond

// approveSentMsg / approveErrMsg — the approved-plan send's async
// resolutions (the F3 tagged twins of chatSentMsg/sendErrMsg: a failed
// approve must NEVER be mistaken for an ordinary failed send — the
// rollback is exact). The flip to build rides approveSentMsg ONLY.
type approveSentMsg struct{ planLen int }
type approveErrMsg struct{ err error }

// approveArmClearMsg — the approve arm's own expiry tick landed
// (armClearMsg's twin): retires the arm + its toast when old enough; a
// YOUNGER re-arm's own tick owns its expiry.
type approveArmClearMsg struct{}

// setAgentMode is the ONE mutation point for m.agentMode — the pane's own
// mode badge (pane.SetMode) never drifts from the model's.
func (m *Model) setAgentMode(mode string) {
	m.agentMode = mode
	if m.plan != nil {
		m.plan.SetMode(mode)
	}
}

// planAgent is the agent tag the composer send paths attach in plan mode
// (build returns "" — plain Send, no agent field on the wire ever).
func (m *Model) planAgent() string {
	if m.agentMode == agentModePlan {
		return agentModePlan
	}
	return ""
}

// paneAgent is planAgent's twin for the ONE closure that cannot reach the
// model (the chat send callback built in New, captured before any Model
// copy exists): the pane pointer is shared across every copy and
// setAgentMode keeps its Mode() in lockstep, so a prompt typed at T
// routes by the mode AT T, not at app build time.
func paneAgent(plan *panels.PlanEditor) string {
	if plan != nil && plan.Mode() == agentModePlan {
		return agentModePlan
	}
	return ""
}

// sendChatMode is sendChat + the agent seam: in plan mode (agent ==
// "plan") a text-only prompt rides SendAgent(text, "plan"); everything
// else — build mode, a harness stub without the seam, a file-carrying
// prompt (attachments win over the tag: full fidelity beats metadata) —
// takes the existing attachment/plain path untouched.
func sendChatMode(b state.Backend, text string, atts []state.Attachment, agent string) error {
	if agent != "" && len(atts) == 0 {
		if ab, ok := b.(agentBackend); ok {
			return ab.SendAgent(text, agent)
		}
	}
	return sendChat(b, text, atts)
}

// togglePlanMode is ctrl+p: MODE TOGGLE ONLY. It flips build↔plan; the
// chat input KEEPS focus (sends route per-mode as before); the plan pane
// does NOT open on toggle — pane visibility is planPaneVisible's rule
// (content-presented only). The key's exclusions — focused terminal tab,
// open perm/question/model floats — live at the claim site (handleKey),
// not here.
func (m *Model) togglePlanMode() tea.Cmd {
	if m.plan == nil {
		return nil
	}
	if m.agentMode == agentModePlan {
		m.setAgentMode(agentModeBuild)
		m.plan.Blur()
		note := "[office] build mode — prompts go straight to the boss"
		if m.planSendPending > 0 {
			// F4: a plan-tagged send is still in flight — its completion
			// lands in the chat; say so at the EXACT exit the member made.
			note += "; an in-flight reply lands in chat"
		}
		m.notice(note)
		return nil
	}
	m.setAgentMode(agentModePlan)
	m.notice("[office] plan mode — the boss's reply lands in the plan pane · ctrl+x twice approves a presented plan · ctrl+p exits")
	if m.agentDegraded() && !m.planDegradeNoted {
		// F5: degraded-entering warning, once per office session — the
		// badge flips too, but a first-time member gets told in text.
		m.planDegradeNoted = true
		m.noticeErr(planDegradeWarn)
	}
	return nil
}

// planPaneVisible — the ONE floor-slot swap rule: the pane owns the slot
// (desktop: the floor; mobile: the panel region under the band) IFF plan
// mode is active AND the pane carries content (boss-presented, user-edited
// or persisted-restored). An empty plan-mode buffer leaves the normal
// office floor in place.
func (m *Model) planPaneVisible() bool {
	return m.agentMode == agentModePlan && m.plan != nil &&
		strings.TrimSpace(m.plan.Value()) != ""
}

// --- the presentation shape gate -----------------------------------------
// Real plans are STRUCTURED markdown documents; boss chatter/status
// narration isn't. A completed boss reply mirrors into the pane ONLY when
// it (a) carries at least one markdown structure signal and (b) has
// enough body to be a document (≥3 lines, ≥160 non-whitespace chars).
// The gate reads SHAPE ONLY — sender metadata decides nothing here beyond
// the existing From=="boss" guard (an "anonymized" gate: a chatter-shaped
// bubble from the boss and a plan-shaped bubble get judged on their text,
// never on a label the wire attached).
var (
	planShapeHeading  = regexp.MustCompile(`^#{1,4}\s`)
	planShapeBullet   = regexp.MustCompile(`^\s*[-*]\s`)
	planShapeNumbered = regexp.MustCompile(`^\s*\d+\.\s`)
	planShapeBoldLbl  = regexp.MustCompile(`^\*\*[A-Za-z][^*]{2,40}\*\*\s*:?`)
)

// planShapeMinLines / planShapeMinNonWS — the document-heft floor: a one-
// liner bullet or a bare "Goal" heading with nothing under it is not a
// plan, it's a fragment.
const (
	planShapeMinLines = 3
	planShapeMinNonWS = 160
)

// looksLikePlan reports whether the text is plan-SHAPED: at least one
// markdown structure signal (heading, bullet, numbered list, fenced code
// block, or a bold section label like "**Steps**:") riding a body of
// ≥planShapeMinLines lines and ≥planShapeMinNonWS non-whitespace chars.
// Small, pure, and deliberately heuristic — it errs on the side of NOT
// presenting (the pane keeping its last plan is always safe).
func looksLikePlan(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) < planShapeMinLines {
		return false
	}
	nonWS := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			nonWS++
		}
	}
	if nonWS < planShapeMinNonWS {
		return false
	}
	for _, ln := range lines {
		if planShapeHeading.MatchString(ln) || planShapeBullet.MatchString(ln) ||
			planShapeNumbered.MatchString(ln) || planShapeBoldLbl.MatchString(ln) ||
			strings.HasPrefix(strings.TrimSpace(ln), "```") {
			return true
		}
	}
	return false
}

// planGateLatch — the shape gate's per-instance policy memory. It lives
// OUT of Model's fields (the pane pointer is shared across every tea
// value-copy of the model, so keying on it keeps ONE latch per app.New)
// and carries exactly the debounce/once-per-session state the rejection
// notices need.
type planGateLatch struct {
	valvePosted bool   // the empty-pane escape valve fired (once per office session)
	noteMsgID   string // boss message id the "kept its last plan" note last fired for
	noteText    string // chatter text that note fired for — identical repeats stay silent
}

// planGateLatches is keyed by the app's *panels.PlanEditor (stable for the
// app's lifetime — /new reuses the pane). One entry per app instance;
// test models each get a fresh pane, so latches never leak across tests.
var planGateLatches sync.Map // *panels.PlanEditor → *planGateLatch

// planGate returns this model's gate latch (creating it on first use).
func (m *Model) planGate() *planGateLatch {
	v, _ := planGateLatches.LoadOrStore(m.plan, &planGateLatch{})
	return v.(*planGateLatch)
}

// notePlanShapeRejection — the chatter side of the gate: the reply did
// NOT look like a plan, so the pane keeps its current content. What the
// transcript hears about it is shaped by what the pane holds:
//
//   - pane shows content IDENTICAL to the reply → silence (dedupe: the
//     same text is already where it would land).
//   - pane is EMPTY → the escape valve, ONCE per office session: teach
//     the first-time member that boss chatter is fine and plans land on
//     the left. Never refires within the session (no note-spam while the
//     boss narrates).
//   - pane shows the untouched starter template → silence (the member
//     just opened the scratch pane; nothing needs explaining).
//   - pane shows a real last plan → the dim "kept its last plan" note,
//     debounced to at most once per boss turn (message ID) and never
//     twice for identical chatter text.
func (m *Model) notePlanShapeRejection(msg state.ChatMsg) {
	cur := strings.TrimSpace(m.plan.Value())
	if cur != "" && cur == strings.TrimSpace(msg.Text) {
		return
	}
	if cur == "" {
		g := m.planGate()
		if !g.valvePosted {
			g.valvePosted = true
			m.notice(planChatValveNotice)
		}
		return
	}
	if m.plan.IsStarter() {
		return
	}
	g := m.planGate()
	if g.noteMsgID == msg.ID || g.noteText == msg.Text {
		return
	}
	g.noteMsgID, g.noteText = msg.ID, msg.Text
	m.notice(planNotPlanNotice)
}

// presentBossPlan — the bossCompleted mirror for plan mode: a COMPLETED
// boss markdown bubble is adopted as the pane content (passive — chat
// keeps focus) IF it LOOKS like a plan (looksLikePlan). Boss chatter/
// status narration never presents: the pane keeps its current content
// and notePlanShapeRejection decides what (if anything) the transcript
// says about it. Anti-clobber: a pane carrying USER edits (userDirty) is
// untouched and the dim "kept" note rides the office notice channel; the
// latch resets on each adoption. Error bubbles and empty finals never
// present.
func (m *Model) presentBossPlan(msg state.ChatMsg) {
	if m.plan == nil || m.agentMode != agentModePlan {
		return
	}
	if msg.From != "boss" || strings.HasPrefix(msg.ID, "boss-error-") {
		return
	}
	if strings.TrimSpace(msg.Text) == "" {
		return
	}
	if !looksLikePlan(msg.Text) {
		m.notePlanShapeRejection(msg)
		return
	}
	if m.plan.UserDirty() {
		m.notice(planKeptNotice)
		return
	}
	m.plan.SetValue(msg.Text)
	m.plan.SetUserDirty(false)
	m.restoredPlan = false // F2: a boss-adopted buffer is presented, not restored
}

// notePlanCompletion — the F4 turn-edge hook, driven for EVERY completed
// boss reply (applyEvent, right before presentBossPlan): a completed
// reply consumes one outstanding plan-tagged send. When the office flipped
// back to build MID-TURN (the reflex ctrl+p), presentBossPlan skips and
// the pane never opens — the transcript must carry the tell instead, ONCE
// per completion, and only for normal completions (error/empty finals
// already speak for themselves).
func (m *Model) notePlanCompletion(msg state.ChatMsg) {
	if m.planSendPending <= 0 {
		return
	}
	m.planSendPending--
	if m.agentMode != agentModePlan &&
		msg.From == "boss" && !msg.Pending &&
		!strings.HasPrefix(msg.ID, "boss-error-") &&
		strings.TrimSpace(msg.Text) != "" {
		m.notice(planLandedInChat)
	}
}

// openPlanForEdit — a click inside the pane region in plan mode (the
// existing swallow-routing): arm the starter scaffold into an EMPTY pane
// (the scratch workflow; a presented plan is never clobbered), then hand
// the pane the keys. esc blurs back to chat (handleKey's pane branch).
func (m *Model) openPlanForEdit() {
	m.plan.ArmStarter()
	m.plan.Focus()
}

// approveRefusal — the claim-side refusal gate (F1/F2): returns the
// refusal notice text when the approve can NOT fire, "" when it can.
// Refusals land BEFORE the double-press arm — a toast promising "approve
// + switch to build" never shows when nothing can fire. Ideal order:
//
//   - F2: a restored-from-session buffer (SessionFile.PlanText hydrate)
//     refuses while UNTOUCHED — approving stale text the member never
//     re-read would spend a build turn on a session ago. ANY edit lifts
//     the gate: the latch follows userDirty (its own write side), so a
//     restored plan that was edited is just a plan.
//   - an empty/hidden pane (no plan yet);
//   - the never-touched starter template (only reachable after a manual
//     open) — approving boilerplate would spend a whole build turn on
//     nothing.
func (m *Model) approveRefusal() string {
	if m.plan == nil {
		return planNothingNotice
	}
	if m.restoredPlan {
		if m.plan.UserDirty() {
			m.restoredPlan = false // edited since restore — the gate lifts
		} else {
			return planRestoredNotice
		}
	}
	if strings.TrimSpace(m.plan.Value()) == "" {
		return planNothingNotice
	}
	if m.plan.IsStarter() {
		return planStarterNotice
	}
	return ""
}

// approvePlan is the ctrl+x FIRE (the double-press's second strike —
// model.go's claim arms first): the composed prompt (approvePrefix + the
// plan body) leaves through the agent seam with agent="build". F3: the
// mode flip rides SEND ACCEPTANCE — the closure returns the tagged
// approveSentMsg (Update's case flips to build, blurs the editor, resets
// the dirty/restore latches, posts the approval notice) or the tagged
// approveErrMsg (plan mode KEPT, red transcript row, pane untouched) —
// never a stale flip on a wire the serve rejected. The buffer PERSISTS
// either way for restore — a re-entered plan mode finds the plan again.
func (m *Model) approvePlan() tea.Cmd {
	if refused := m.approveRefusal(); refused != "" {
		m.notice(refused)
		return nil
	}
	v := m.plan.Value()
	text := approvePrefix + v
	b := m.backend
	return func() tea.Msg {
		if b != nil {
			var err error
			if ab, ok := b.(agentBackend); ok {
				err = ab.SendAgent(text, agentModeBuild)
			} else {
				// seam-less harness stub — degrade open to the plain send
				err = sendChat(b, text, nil)
			}
			if err != nil {
				// tagged twin — no cross-talk with an ordinary failed
				// send (sendErrMsg keeps its own generic transcript row).
				return approveErrMsg{err: err}
			}
		}
		return approveSentMsg{planLen: len(v)}
	}
}

// planText is the persistence projection (session.json's planText field):
// the buffer content worth keeping across boots. An empty or never-edited
// starter template is NOTHING ("", and the omitempty tag drops it from
// the file) — a pristine editor must not fake a saved plan. The same
// looksLikePlan gate as presentation applies: a buffer holding scratch
// chatter never persists — only plan-shaped documents survive a boot. An
// approved plan KEEPS persisting (its buffer is retained for restore).
func (m *Model) planText() string {
	if m.plan == nil {
		return ""
	}
	v := m.plan.Value()
	if strings.TrimSpace(v) == "" || m.plan.IsStarter() || !looksLikePlan(v) {
		return ""
	}
	return v
}

// agentDegradeSeam — the additive backend seam (F5b): the live backend's
// promptAgentRejected latch surfaced app-side. Type-asserted exactly like
// agentBackend — harness stubs without it are simply never degraded.
type agentDegradeSeam interface {
	AgentDegraded() bool
}

// agentDegraded — the serve rejected the plan/build agent field: sends
// still go out (bare), but plan-mode ROUTING is dead weight. The badge
// and the entry warning ride this so the member is never told plan mode
// works while it silently cannot.
func (m *Model) agentDegraded() bool {
	if m.backend == nil {
		return false
	}
	if d, ok := m.backend.(agentDegradeSeam); ok {
		return d.AgentDegraded()
	}
	return false
}

// agentBadge is the statusbar's plan/build marker segment: "[plan]" while
// plan mode is active, "" in build — the default office's statusbar stays
// byte-identical to before (the badge only ever appears during a plan
// session). F5b: a degraded serve shows "[plan·degraded]" — the label
// keeps showing but stops LYING about routing.
func (m *Model) agentBadge() string {
	if m.agentMode != agentModePlan {
		return ""
	}
	if m.agentDegraded() {
		return "[plan·degraded]"
	}
	return "[plan]"
}
