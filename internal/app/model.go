// Package app — the root Bubble Tea model for theboringoffice v2: state reducer
// (exact port of node-legacy/src/app.tsx officeReducer + initialState),
// layout, key routing, the power governor, and the backend event seam.
//
// Layout: topbar (1) | middle (floor left flex | right sidebar) | statusbar (1).
// The sidebar holds seven tabs — chat | terminal | agents | board | mail |
// activity | git — and its width is configurable (brain.json ui.sidebarWidth,
// 26..100 clamp, 0 = default 80; /compact mode narrows it to 30). The LEFT
// pane is a two-tab slot of its own — floor (default) | browser — flipped
// with ctrl+b (app/browser.go owns the switcher). /zen is a transient
// fullscreen-floor mode (sidebar hidden, any key exits).
// Events arrive as state.Event tea.Msgs (backend goroutine → tea.Program.Send);
// the animation tick is a re-arming tea.Tick loop governed by the brain.json
// power posture (power.go): busy = smooth (180ms/150ms/400ms), idle = cheap
// (1s/2s), auto drifts to 3s after 60s of quiet.
package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/gitx"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/projinfo"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	mailCap = 30
	// chatCap / thinkCap / toolCap are FUSES, not windows: the transcript is
	// WhatsApp-grade — the FULL history stays (oldest top, newest bottom,
	// everything scrollable, nothing eaten as new messages arrive). The
	// 10k bound only trips on pathological unbounded growth inside one
	// process; it must never clip a human conversation.
	chatCap   = 10000
	thinkCap  = 10000 // thinking blocks kept in chat
	toolCap   = 10000 // tool one-liners kept in chat
	bubbleCap = 3     // never more than 3 concurrent balloons (drop oldest)

	// Sidebar sizing: the default is 80 cols; brain.json ui.sidebarWidth is
	// clamped to 26..100 (explicit config wins over /compact); the compact
	// layout mode (/compact, ui.compact) narrows the default to 30.
	defaultSidebarW = 80
	compactSidebarW = 30
	sidebarMin      = 26
	sidebarMax      = 100

	degradeCols = 100 // below this, the sidebar shrinks instead of the floor
	// mobileMaxCols — below this window width the middle flips to the
	// mobile stack (compact floor band on top, active panel full-width
	// below). The horizontal split needs ~100 cols: the sidebar eats
	// ~26..80 of them, so a sub-100 window leaves the floor <65 cols.
	mobileMaxCols = 100
	minCols       = 40
	minRows       = 12

	// Message queue — the INTELLIGENT BACKLOG: Enter while the turn is
	// ROADBLOCKED (an open permission modal, or a question hold outstanding)
	// enqueues a numbered item; the turn-complete flush sends the whole
	// backlog as ONE composed [BATCH DISPATCH] prompt (the boss runs manager
	// dispatch discipline over it — trivial inline, parallel sub-agents for
	// the rest). Exactly 1 item keeps the plain FIFO send. A plain busy turn
	// (boss typing, no roadblock) does NOT enqueue — free-queuing sends such
	// prompts straight to the backend, which queues them server-side.
	queueCap = 10

	// batchTitleClip — the QueueItemStart board title is the first 60 chars
	// of the typed text (machine clip, not NL).
	batchTitleClip = 60
	// batchSummaryClip — per-item clip inside the composite user bubble
	// ("you › 3 items: fix the badge; ship v2; …").
	batchSummaryClip = 32
	// batchRespawnWindow — a session.error on the primary inside this window
	// of the batch send counts as "the boss died on the batch": ONE respawn.
	batchRespawnWindow = 5 * time.Second

	// delegatingQuietTicks — boss-side quiet horizon for the delegation
	// state: a pending boss placeholder with no stream/thought/primary-
	// tool activity for MORE than this many ticks (and busy workers on
	// the floor) flips BossDelegating on instead of the lonely "typing…" spin.
	delegatingQuietTicks = 6

	// pagerFetchTimeout — one history hop's worst case: the scroll-to-top
	// walk must never hang a gesture (sessionListTimeout's twin).
	pagerFetchTimeout = 10 * time.Second
	// pagerDemoSession — the demo walk's fixed session label: the demo
	// backend reports no primary session id (it IS the fixture) and its
	// MessagesPage ignores the id — the label exists so the pager's
	// binding, stale-landing drops and /new resets stay uniform.
	pagerDemoSession = "demo"
	// streamReconnectedMarker — the backend's ONE recovery note
	// (opencode.go's SSE ladder): a fresh stream re-arms the walk's
	// failure backoff (pager.ResetFailures) AND re-opens seeding when the
	// first seed hop died with the old stream.
	streamReconnectedMarker = "[theboringfloor] event stream: reconnected"
)

// batchMarker prefixes the ONE composed batch prompt. Machine format (the
// app writes it, the backend echoes it verbatim) — the chat render rewrites
// ANY chat-user echo carrying it into the compact composite bubble.
const batchMarker = "[BATCH DISPATCH — "

// The concierge contract lives in the state package (asserted here, owned
// by the backend dev): state.EvChatOffice ("chat-office") carries
// Msg{From:"office", Kind:"office", ID:"office-<msgID>"} with the boss
// stream's replace-in-place mechanics; state.ConciergeCapable is the
// SendConcierge(text) error seam the app type-asserts off backend (the
// same additive-seam pattern as teamBackend/attachmentBackend — harness
// stubs that lack it simply degrade to the boss queue).

// teamBackend — the backlog/board seam live and demo backends expose beyond
// state.Backend (the backend dev's contract; the app type-asserts it).
// QueueItemStart mirrors one backlog item to the board and returns its id
// ("" when the backend has no board seam — QueueItemDone("") is a no-op);
// QueueItemDone closes the row when the batch's turn completes;
// ResetPrimary(true) respawns a fresh boss session for the one-shot retry.
type teamBackend interface {
	QueueItemStart(index int, title string) string
	QueueItemDone(boardID string)
	ResetPrimary(forceNew bool) error
}

// attachmentBackend — the chat-input attachment seam live and demo backends
// expose beyond state.Backend (same type-assert pattern as teamBackend).
// The interface method stays out of state.Backend on purpose: harness
// stubs (uishot/headless) keep their plain-text Send and simply never
// attach. SendWith sends one prompt carrying file parts for atts (the
// backend reads each Attachment.Path at send time).
type attachmentBackend interface {
	SendWith(text string, atts []state.Attachment) error
}

// bypassBackend — the bypass-permissions seam both live transports
// (opencode + claudecode) expose beyond state.Backend (the backend dev's
// contract; the app type-asserts it, same additive-seam pattern as
// teamBackend/attachmentBackend). Pre-Start it latches the flag (nil
// return); once Start froze the instance's spawn argv/boot config the
// call errors "respawn required" — so the office's toggle ALWAYS builds
// a fresh backend and lands this on it BEFORE Start (respawnForBypass).
// Harness stubs without the seam simply skip the transport hop.
type bypassBackend interface {
	SetBypassPermissions(on bool) error
}

// Bypass-permissions copies (frozen — pinned by bypass_test.go and the
// uishot --bypass leg):
//
//   - bypassConfirmPrompt: the enable path's explicit confirm, paged
//     through the office's EXISTING question popover (a mode that
//     silences every guardrail never arms on one keypress). The answers
//     are exactly "enable"/"cancel"; cancel (and esc, and any custom
//     text) is a no-op.
//   - bypassOnNotice/bypassOffNotice: the toggle's transcript rows.
//   - bypassAutoNotice: one dim row per stray ask auto-answered while
//     the respawn lands (fmt %s = the tool name).
//   - bypassBadgeText: the topbar's loud segment while armed.
const (
	bypassConfirmPrompt = "Enable bypass permissions? Agents will run tools and browser actions WITHOUT asking — this office session only"
	bypassOnNotice      = "bypass permissions: ON — nothing will ask"
	bypassOffNotice     = "bypass permissions: OFF"
	bypassAutoNotice    = "bypass: auto-approved %s"
	bypassBadgeText     = " ⚠ BYPASS "
)

// bypassConfirmID — the office-local question hold's sentinel id: the
// /bypass enable confirm rides the EXISTING question popover but is NOT
// a backend question — questionAnswerMsg/questionLaterMsg intercept it
// locally (no AnswerQuestion wire, no /question parking, no fold-in of
// real boss questions).
const bypassConfirmID = "office-bypass-confirm"

// sendChat pushes one prompt through the attachment seam when the backend
// implements it, else falls back to the plain-text Send. The fallback can
// only fire in harness stubs — live and demo both implement the seam.
func sendChat(b state.Backend, text string, atts []state.Attachment) error {
	if ab, ok := b.(attachmentBackend); ok {
		return ab.SendWith(text, atts)
	}
	return b.Send(text)
}

// currentBackend owns a generation of the transport. A send takes a short
// lease on the accepting generation, then performs transport I/O without the
// holder lock. Replacing it moves the old generation to draining: existing
// leases may still enter Send, but no new lease can be admitted. Cleanup marks
// it retired immediately before Stop, once every lease has returned.
type currentBackend struct {
	mu         sync.Mutex
	current    *backendGeneration
	beforeSend func(*backendGeneration) // test seam: runs after admission, before transport I/O
}

// errBackendUnavailable is returned rather than treating an unavailable
// generation as a successful send. A successful tea command produces the
// normal chatSentMsg, so silently returning nil here makes the composer look
// accepted even though no transport ever received the prompt.
var errBackendUnavailable = errors.New("active backend unavailable")

type backendGenerationState uint8

const (
	backendAccepting backendGenerationState = iota
	backendDraining
	backendRetired
)

type backendGeneration struct {
	backend  state.Backend
	state    backendGenerationState
	inFlight int
	drained  chan struct{}
}

func newCurrentBackend(b state.Backend) *currentBackend {
	return &currentBackend{current: &backendGeneration{backend: b, state: backendAccepting, drained: make(chan struct{})}}
}

func (c *currentBackend) send(text string, atts []state.Attachment, agent string) error {
	c.mu.Lock()
	g := c.current
	if g == nil || g.backend == nil || g.state != backendAccepting {
		c.mu.Unlock()
		return errBackendUnavailable
	}
	g.inFlight++
	beforeSend := c.beforeSend
	c.mu.Unlock()
	defer c.release(g)
	if beforeSend != nil {
		beforeSend(g)
	}
	return sendChatMode(g.backend, text, atts, agent)
}

// lease admits one in-flight op on the accepting generation so Stop cannot
// race a cmd that captured m.backend at build time. fn runs without the
// holder lock. MCP/question/permission hops use this; chat send has its
// own path because of the beforeSend test seam.
func (c *currentBackend) lease(fn func(state.Backend) error) error {
	if c == nil {
		return errBackendUnavailable
	}
	c.mu.Lock()
	g := c.current
	if g == nil || g.backend == nil || g.state != backendAccepting {
		c.mu.Unlock()
		return errBackendUnavailable
	}
	g.inFlight++
	c.mu.Unlock()
	defer c.release(g)
	return fn(g.backend)
}

// sendConcierge leases the same accepting generation as ordinary chat. Busy
// sends may route to the concierge, but that route must not retain an old
// backend past a bypass replacement and race its Stop.
func (c *currentBackend) sendConcierge(text string) (handled bool, err error) {
	c.mu.Lock()
	g := c.current
	if g == nil || g.backend == nil || g.state != backendAccepting {
		c.mu.Unlock()
		return true, errBackendUnavailable
	}
	cb, ok := g.backend.(state.ConciergeCapable)
	if !ok {
		c.mu.Unlock()
		return false, nil
	}
	g.inFlight++
	c.mu.Unlock()
	defer c.release(g)
	return true, cb.SendConcierge(text)
}

func (c *currentBackend) supportsConcierge() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current == nil || c.current.backend == nil || c.current.state != backendAccepting {
		return false
	}
	_, ok := c.current.backend.(state.ConciergeCapable)
	return ok
}

func (c *currentBackend) release(g *backendGeneration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	g.inFlight--
	if g.state == backendDraining && g.inFlight == 0 {
		close(g.drained)
	}
}

// replace makes b current before returning. It never waits for old sends or
// calls Stop itself: callers schedule the returned command so Bubble Tea's
// update loop stays free of transport teardown.
func (c *currentBackend) replace(b state.Backend) tea.Cmd {
	c.mu.Lock()
	old := c.current
	c.current = &backendGeneration{backend: b, state: backendAccepting, drained: make(chan struct{})}
	if old != nil {
		old.state = backendDraining
		if old.inFlight == 0 {
			close(old.drained)
		}
	}
	c.mu.Unlock()
	if old == nil || old.backend == nil {
		return nil
	}
	return func() tea.Msg {
		<-old.drained
		c.mu.Lock()
		// drained closes only after replace made this generation draining and
		// every admitted lease released. Retiring under the same lock makes
		// this transition the final state change before Stop.
		old.state = backendRetired
		c.mu.Unlock()
		return backendStopMsg{err: old.backend.Stop()}
	}
}

// currentBackendSend is the chat panel's ordinary-Enter callback. Its backend
// lookup deliberately happens inside the returned tea.Cmd, at SEND time.
func currentBackendSend(current *currentBackend, plan *panels.PlanEditor, text string, atts []state.Attachment) tea.Cmd {
	return func() tea.Msg {
		// Slash commands dispatch locally, never touch the backend, and
		// never echo as chat-user.
		if strings.HasPrefix(text, "/") {
			return slashMsg{text: text}
		}
		agent := paneAgent(plan)
		if err := sendViaCurrentBackend(current, text, atts, agent); err != nil {
			cleanupAttachments(atts) // nobody will retry this prompt
			return sendErrMsg{err: err}
		}
		cleanupAttachments(atts)
		return chatSentMsg{text: text, agent: agent}
	}
}

// sendViaCurrentBackend resolves the accepting transport when a command
// actually runs. Commands issued while a bypass replacement is starting may
// otherwise retain the old backend and reach it after the generation has been
// retired. Callers pass paneAgent (or approve's build tag) computed at send time.
func sendViaCurrentBackend(current *currentBackend, text string, atts []state.Attachment, agent string) error {
	if current == nil {
		return errBackendUnavailable
	}
	return current.send(text, atts, agent)
}

// queueEntry — one backlog item: the typed text, its chat-input
// attachments (they must survive the busy wait and ride the flush), and
// the board row id QueueItemStart handed back ("" when the backend has no
// team seam).
type queueEntry struct {
	text    string
	atts    []state.Attachment
	boardID string
}

// QueueDebugf, when set (uisshot --debug only), receives message-queue
// trace lines. Nil in production — the hot path checks before formatting.
var QueueDebugf func(format string, args ...any)

func qdebugf(format string, args ...any) {
	if QueueDebugf != nil {
		QueueDebugf(format, args...)
	}
}

// cleanupAttachments removes panel-created temp dirs (pasted images live in
// os.MkdirTemp "theboringoffice-paste-*", Attachment.Temp). Best-effort, and ONLY
// ever called after a send has resolved: enqueue must not clean (the flush
// still needs the file), and the batch respawn path keeps them for its one
// retry — the cleanup fires on success or on the terminal failure.
func cleanupAttachments(atts []state.Attachment) {
	seen := map[string]bool{}
	for _, a := range atts {
		if a.Temp != "" && !seen[a.Temp] {
			seen[a.Temp] = true
			_ = os.RemoveAll(a.Temp)
		}
	}
}

// cleanupEntries is cleanupAttachments over a batch of queue entries
// (their attachments concatenate the same way the flush send sees them).
func cleanupEntries(items []queueEntry) {
	for _, it := range items {
		cleanupAttachments(it.atts)
	}
}

// attachNames is the " · "-joined display-name projection of an attachment
// list (board titles; the backend has its own unexported twin — packages
// don't share internals).
func attachNames(atts []state.Attachment) string {
	names := make([]string, len(atts))
	for i, a := range atts {
		names[i] = a.Name
	}
	return strings.Join(names, " · ")
}

// SoundBus — the sound engine seam (the sound dev owns the engine; the app
// only CALLS Play). Nil by default — manager injects via SetSoundBus.
type SoundBus interface {
	Play(name string)
}

// NotifyBus — the OS desktop-notification engine seam (internal/notify owns
// the engine; the app only CALLS Notify). Nil by default — main injects via
// SetNotifyBus. Mode flips (/notify) ride a SetMode type-assert, the same
// style of seam as the send-attachment one: wiring stays additive for headless
// stubs.
type NotifyBus interface {
	Notify(kind, title, body string)
}

// btwSnapshot — saved state for a /btw subchat session.
type btwSnapshot struct {
	chat      []state.ChatMsg
	tasks     []state.BoardTask
	mails     []state.MailItem
	primaryID string
}

// Model is the tea.Model for the whole app.
type Model struct {
	backend        state.Backend
	currentBackend *currentBackend // shared across value-copy tea updates
	// recentToolOutputs keeps the completed body beside the compact transcript
	// row. ChatMsg deliberately stores only a one-line tool summary; the recent
	// context handoff needs the bounded output too.
	recentToolOutputs map[string]string
	st                state.OfficeState
	cfg               *config.Config // brain.json (nil-tolerant: Default() substituted)
	gov               *governor      // power/caching bookkeeping, shared across copies

	// social — the office's SocialClock (ambient.go). Pointer, so the plan
	// survives the value-copy update loop. lastDispatchTick feeds its
	// "active dispatch in-flight <30 ticks" busy gate.
	social           *SocialClock
	lastDispatchTick int // -1 = no dispatch seen yet this run

	// snd — the sound bus (nil by default; manager injects). Reducer hook
	// points call playSound() which no-ops on nil.
	snd SoundBus

	// notifyBus — the OS desktop-notification seam (nil = headless; main
	// wires it UNCOUPLED from the sounds gate — a muted speaker config must
	// never mute the look-away pings). Every hook gates on !focused: the
	// pings exist for when the terminal has NO focus; in front of your face
	// the popover/spinner already carries the signal.
	notifyBus NotifyBus
	// focused — terminal focus latch, fed by bubbletea's ReportFocus stream
	// (tea.FocusMsg/tea.BlurMsg). DEFAULT TRUE: terminals without
	// focus-event support never stream, and unknown == focused — an
	// unsupported terminal must never false-ping.
	focused bool
	// permNotifyIDs — the permission COHORT for notifications: wire ids of
	// every UNANSWERED ask (pending ∪ esc'd pile alike). The ask that flips
	// the set 0→1 owns the ONE cohort ping; every later ask inside the same
	// cohort coalesces silently; user answers + server-side "resolved"
	// events drop ids; the set emptying re-arms the next cohort. Resolved
	// events and re-presses never mint pings — but a BLUR while the cohort
	// is live fires its own one (the "looked away during a block" intent,
	// BlurMsg case in Update).
	permNotifyIDs map[string]bool
	// notifyDoneArmed — the done-ping debounce: a user send (chatSentMsg/
	// busySentMsg) arms it, the FIRST completed boss bubble fires exactly
	// once and disarms — boss-error bubbles disarm silently too (the error
	// already owns the transcript), and a completion landing while
	// questionParked counts as not-done (the member just engaged through
	// the modal; the arm is consumed without a ping). The concierge lane
	// (EvChatOffice) never touches it.
	notifyDoneArmed bool

	// bossName/bossShort — the human boss label from cfg.Boss.Name: the full
	// string for roster rows ("jorge (El Jefe)"), its first word for the
	// busy placeholder/spinner ("jorge is typing…").
	bossName  string
	bossShort string

	width, height int
	middleH       int
	sidebar       int
	floorW        int
	tabs          *panels.Tabs
	chat          *panels.Chat
	agents        *panels.Agents // roster tab — floor-click selection highlight
	activity      *panels.Activity
	termTab       *termTabWrap // tab 2: the real OS-shell (lazy PTY, terminal.go)
	// browser — the LEFT pane's second tab: the in-TUI page viewer
	// (app/browser.go owns the wiring; panels/browser.go owns the pane).
	// browserSlashNote is the /open notice latch: set by the slash case,
	// consumed by the FIRST BrowserPageMsg that lands (in-pane navs stay
	// silent). leftTab is the floor slot's switcher position:
	// leftTabFloor (the office floor, the default) | leftTabBrowser.
	browser          *panels.Browser
	browserSlashNote string
	leftTab          int
	// termCaptured — the terminal tab's OPT-IN keyboard state (wave-42):
	// false = RELEASED (the default — office keys behave normally on the
	// terminal tab), true = CAPTURED via ctrl+space (wave-41: every key
	// goes to the shell until ctrl+space toggles back or ctrl+o releases).
	// normalizeTermCapture keeps it from ever escaping its tab: leaving
	// while captured auto-releases, and every (re-)entry starts RELEASED —
	// explicit opt-in each visit.
	termCaptured bool
	keys         KeyMap

	// plan/build agent mode (plan_mode.go), MODEL-side on purpose —
	// state.Mode already means live/demo and stays untouched. agentMode
	// is "plan"|"build"; plan is the colleague's contract-frozen floor-slot
	// plan editor (panels.PlanEditor, asserted against planEditorPane);
	// planTemplate is the pane's REST buffer captured at build time ("" —
	// conversation-first: the starter scaffold arms on manual open, see
	// openPlanForEdit), so the /new reset in sessions.go clears the canvas
	// back to empty+hidden and a pristine editor never fakes a saved plan.
	plan         *panels.PlanEditor
	planTemplate string
	agentMode    string
	// approveArmAt — the ctrl+x double-press arm (approveArmWindow, F1):
	// set by the first press (the hint bar swaps to the warn-class
	// approveArmToast), cleared by its own expiry tick (approveArmClearMsg),
	// by ANY other key press, or by the firing second press. plan_mode.go
	// owns the whole approve flow.
	approveArmAt time.Time
	// restoredPlan — F2: set when session.json's planText seeds the pane
	// buffer on boot (sessions.go hydrate); cleared on any edit (it follows
	// userDirty), on a boss adoption (presentBossPlan), on an approve, and
	// on /new. While set AND untouched, the approve arm refuses.
	restoredPlan bool
	// planSendPending — F4: outstanding plan-tagged sends; incremented on
	// each plan-mode send's acceptance (chatSentMsg/busySentMsg), consumed
	// one-per-completed-boss-reply (notePlanCompletion). A completion that
	// lands after a reflex flip back to build posts the land-in-chat note;
	// toggling OUT of plan mode with any pending shows the exit suffix.
	planSendPending int
	// planDegradeNoted — F5: the one-time-per-session entry warning for a
	// degraded agent seam (the badge keeps showing it every frame after).
	planDegradeNoted bool

	// floor-click bookkeeping (bubbletea v2 mouse): lastClickAgent/At
	// detect a DOUBLE-click on the same sprite (double-click = toggle
	// that agent's chat thread).
	lastClickAgent string
	lastClickAt    time.Time

	// zen — transient fullscreen-floor mode: sidebar hidden entirely, topbar
	// stays, statusbar minimal; any key exits. Never persisted (the ruling:
	// /zen is a focus session, not a preference).
	zen bool

	// threadFocus — the thread FOCUS view (ctrl+f): a fullscreen nested
	// panel holding ONE worker thread's complete transcript (panels.
	// ThreadFocus), nil while closed. zen outranks it in BOTH Frame and
	// the key claims; a permission/question float DISMOUNTS it
	// (dismountThreadFocus, applyEvent); esc/ctrl+f close it. focusThread
	// is the focused agent's name (the statusbar's hint segment + a digest
	// term). focusDeferredRender arms the main chat's render saver while
	// open (chat-side deferRender): the focus renders from the same
	// office state, so rebuilding the hidden main transcript per tick
	// would be wasted work.
	threadFocus         *panels.ThreadFocus
	focusThread         string
	focusDeferredRender bool

	// quitArmAt — the ctrl+q double-press arm (quitArmWindow): set by the
	// first press, cleared by its own expiry tick (armClearMsg), by ANY
	// other key press, or by the quitting second press itself. While set,
	// the statusbar hint (and the zen bar's right segment) swap to the
	// warn-class quitArmToast — see hintLine.
	quitArmAt time.Time
	// compactLive — the /compact session override: 0 = inherit
	// brain.json ui.compact, 1 = compact on, 2 = normal on. /mode
	// normal|compact writes cfg.UI.Compact (persisted) and clears this.
	compactLive int

	// mouse transcript selection (selection.go): a left-press over chat
	// text ARMS a pending selection (selPress pins the original press for
	// the motionless-release click replay; selDragged flips on the first
	// drag-motion); copyNote/copyNoteAt ride the status bar for
	// copyNoteWindow after a VERDICTED copy (darwin gates on pbcopy's
	// result); copyNoteBad picks the warn class for a failed copy.
	sel         int
	selPress    tea.Mouse
	selDragged  bool
	copyNote    string
	copyNoteAt  time.Time
	copyNoteBad bool

	// mouse plan-pane drag-select (plan_editor_selection.go): a left-press
	// inside the plan pane's body ARMS the pane's own selection
	// (planSelArmed pins the flight; planSelDragged flips on the first
	// motion — a motionless release is caret placement ONLY, never a copy).
	planSelArmed   bool
	planSelDragged bool

	// Inbound boss-turn image previews (images.go): the probe-once
	// rasterize latch keyed msgID|hash — the lazy rasterize rides ONE
	// tea.Cmd per new payload, never re-running on repeated pins.
	// imgLaneDet/imgLaneDetOK memoize the terminal's image-lane detect
	// read per Model (per boot): probes cost zero env traffic, and a
	// harness stubbing the terminal env per drive (uishot lane legs)
	// gets a fresh read per Model.
	imgProbed    map[string]bool
	imgLaneDet   panels.ImageLane
	imgLaneDetOK bool

	// frameNonce — bumped on every message that can mutate panel ephemera
	// the state digest can't see (textarea draft, scroll, spinner, theme
	// toggles). Part of the frame cache key (digest.go).
	frameNonce   uint64
	activityAdds int // total activity-log appends (digest term)

	// Message backlog (model-level so it survives tab switches): texts typed
	// while a boss reply is pending, each with its board row id.
	queue []queueEntry

	// btw — the /btw subchat save slot: non-nil while in a btw side session.
	// /done restores and nils it. /btw while non-nil is rejected (no nesting).
	// /new while in btw discards the save (you abandoned it). btwHiddenSnap
	// retains a side session that Esc hid behind its pinned main-chat bubble.
	btwSaved       *btwSnapshot
	btwHiddenSnap  *btwSnapshot
	btwPinMsgID    string

	// Batch dispatch bookkeeping (set by dispatchQueued, consumed by the
	// pending→non-pending completion transition):
	//   batchInFlight  — a composed batch is awaiting its turn
	//   batchRespawned — the ONE respawn for the in-flight batch is spent
	//   batchItems     — retained for the ≤5s session.error respawn
	//   batchDoneIDs   — board rows closed by QueueItemDone on completion
	//   batchSummaries — item texts for the composite user-bubble rewrite
	//   batchSentAt    — send time, bounds the session.error window
	//   respawns       — global respawn count for this session run
	batchInFlight  bool
	batchRespawned bool
	batchItems     []queueEntry
	batchDoneIDs   map[string]bool
	batchSummaries []string
	batchSentAt    time.Time
	respawns       int

	// Free-queuing tally (the anti-stuck flow): a prompt typed while the
	// boss is busy goes STRAIGHT to the backend (the serve queues it
	// server-side, draining after the current turn) — no client queue. The
	// model only COUNTS those in-flight sends so the UI keeps talking:
	// serverQueued is the running tally for THIS busy turn (drives the
	// "busy · N queued (server)" status line and, from the second send on,
	// the chat placeholder's "boss: turn N · your message rides next");
	// busySaved/busyStatus own the status-line swap+restore.
	serverQueued int
	busySaved    string // StatusLine saved when the busy compose first painted
	busyStatus   bool   // the busy compose owns StatusLine right now

	// conciergeNoted — the concierge routing latch: the "office routed:
	// boss busy → concierge" line (or the unavailable fallback notice)
	// prints ONCE per busy turn, not per message; resetServerTurn re-arms
	// it when the turn ends.
	conciergeNoted bool

	// Permission prompts (boss AND child sessions — a child's ask rides
	// the same queue as the boss's): permQ.pending is the fifo of
	// unanswered asks, pending[0] the OPEN prompt replacing the textarea
	// (asks stack, displayed one at a time with a "1 of N" position);
	// permQ.escd holds esc'd-but-unanswered prompts, most recent last,
	// /perm re-opens the newest. The floor's blocked sprite + the
	// statusbar/activity [blocked] lines come from EvBlocked, untouched.
	permQ permQueue

	// HOOKUP (browser_open.go): browserActionHolds parks the
	// ⟦browser-action: URL | op⟧ marker's allowed requests behind the
	// member's permission modal, keyed by their synthetic permission id
	// ("browser-action-N", numbered by browserActionSeq). The member's
	// answer resolves them LOCALLY via the permAnswerMsg hookup below
	// (consumeBrowserActionPerm) — these office-minted ids NEVER ride
	// the backend's AnswerPermission wire.
	browserActionHolds map[string]browserActionHold
	browserActionSeq   int

	// Bypass-permissions mode (/bypass): persisted to project-local
	// .theboringfloor/settings.json; restored at boot via
	// WithBypassPermissions. While on: the topbar carries the loud ⚠ BYPASS segment
	// (Frame's spliceBypassBadge), stray EvPermission asks are answered
	// allow-once immediately with NO modal (handlePermissionEvent's
	// bypass arm), and the office's OWN browser-action gate executes
	// without the synthetic modal (browser_open.go). Every toggle
	// respawns the transport: the backend freezes the flag into its spawn
	// argv/boot config at Start, so SetBypassPermissions rides the FRESH
	// instance pre-Start (respawnForBypass). bypassPerms is deliberately the
	// ACTIVE transport's mode, never the requested mode: the badge and the
	// local auto-approve gates must keep describing the backend that can
	// actually receive work. bypassDesired is the coalesced next mode while a
	// replacement is building/starting. The lifecycle is idle -> building ->
	// starting -> active|failed; bypassRestarting covers both in-flight steps.
	bypassPerms      bool
	bypassDesired    bool
	bypassRestarting bool
	bypassQueued     bool // TODO: remove field after test update — no longer used for logic

	// backendTransitioning admits exactly one asynchronous construction at a
	// time. backendTransitionID makes a delayed result harmless if an older
	// command ever reaches Update after its transition has been superseded.
	backendTransitioning bool
	backendTransitionID  uint64

	// Question holds (boss/primary session only): question is the OPEN
	// hold whose WIZARD popover replaces the textarea (radio/checkbox/
	// free-text pages, unlike the y/a/n permission popover); questionEscd
	// is the latest esc'd-but-unanswered hold /question can re-open.
	// questionParked survives defer: the opencode turn is parked at the
	// question reply API (not "typing") until a completed chat-boss
	// arrives after resolution, so the message queue must NOT flush and
	// typed text keeps enqueuing.
	question       *questionHold
	questionEscd   *questionHold
	questionParked bool
	parkedStatus   string // StatusLine saved at park, restored at unpark

	// modelPick — the open /model picker card (bare /model when the
	// backend lists models via the modelListBackend seam; nil = closed).
	// APP-LEVEL float (model.go owns it outright, panels.ModelPickerFrame
	// splices it over the composed frame): unlike the /session picker's
	// chat-embedded card this one never touches the chat panel. While a
	// permission/question float is up it yields keys and hides (a parked
	// turn outranks browsing), resuming when the float clears. Pointer —
	// the Model value copies share the component, like chat/social/gov.
	modelPick *panels.ModelPicker

	// activeThink — CallIDs with an OPEN boss EvThought stream (Done not
	// yet seen). Model-owned (the reducer stays pure): the chat panel
	// consults this set to render streaming blocks expanded/livecoded
	// while completed ones collapse to "thinking · N lines". Any
	// EvChatBoss (pending placeholder OR answer) clears it — a newer boss
	// turn downgrades older unfinished think entries to collapsed.
	activeThink map[string]bool

	// lastBossActivity — st.Tick of the last boss-side activity (stream
	// delta / thought / primary tool / any boss bubble event). Feeds the
	// delegation reducer hook (applyDelegation, P3).
	lastBossActivity int

	// Boss-wedge watchdog (W1): lastBossActivityAt is the WALL-CLOCK twin
	// of the tick counter above, refreshed by the same isBossActivity set
	// in applyDelegation — MINUS the send-side typing placeholder
	// (isSendSidePlaceholder), which the UI stages itself and which
	// therefore proves nothing about the server. NEVER derived from
	// st.Tick — the governor re-arms ticks at 180ms–3s depending on
	// posture (power.go), so tick deltas cannot measure real silence.
	// wedgeNoted is the one-shot latch: the "boss turn wedged" activity
	// line + hint-swap fire ONCE per wedge and ride until REAL boss
	// traffic re-arms it (mirror of conciergeNoted) or
	// resetServerTurn//stop closes the turn. wedgeAfter is the threshold
	// override (0 = bossWedgeAfter); the uishot/test harness seams it via
	// SetWedgeAfterForShot.
	lastBossActivityAt time.Time
	wedgeNoted         bool
	wedgeAfter         time.Duration

	// Idle-wrap watchdog: after a busy shift goes quiet, recap once if
	// the last real chat is not already from the boss or office (and
	// nobody is drafting). Shares wedgeAfter with W1 (0 = 2m).
	shiftBusy  bool
	ghostArmAt time.Time
	ghostNoted bool

	// Office-session persistence (sessions.go; LIVE mode only):
	// sessDir is the working directory the office belongs to ("" = no
	// persist), sessLast throttles the 5s cheap-write loop off EvTick.
	sessDir  string
	sessLast time.Time

	// agentmemoryOK — the /memory header's probe-state latch: the live
	// backend's boot status line announces the agentmemory probe verdict
	// ("… | board: agentmemory (<winner>)" hot / "| board: in-memory …"
	// offline — the same string-marker contract pattern as
	// agentFieldStatusMarker), latched in applyEventCore's EvStatus hook.
	// The additive MemoryLane seam, when implemented, overrules the latch.
	// false default = "file-only" (degrade-open).
	agentmemoryOK bool

	// Older-history pagination (the state.SessionPager seam; the BOSS /
	// primary thread only — employee-thread walks are a follow-up). The
	// design rulings, made concrete here:
	//   - ONE walk controller (panels.ThreadPager) bound at the FIRST
	//     backend event: pagerSession comes from PrimarySessionID()
	//     (live), falling back to pagerDemoSession in demo mode (the demo
	//     backend reports no primary id and its MessagesPage ignores it).
	//     A /new or a batch-respawn flips the primary id under us → the
	//     next event rebinds a FRESH pager (Seed is idempotent, never
	//     re-armed on a walked pager).
	//   - SEEDING: hydrate (≤200 msgs from session.json) covers the tail
	//     but only the serve can mint cursors — so the attach hop fetches
	//     the NEWEST page ONCE, metadata-only: NextCursor + HasMore feed
	//     pager.Seed, the page's ROWS ARE DISCARDED (never spliced):
	//     the hydrated tail is already richer (the ruling: hydrate keeps
	//     exact behavior, nothing is replaced). Overlap between the walk
	//     and the hydrated region is eaten by the dedupe baseline below
	//     instead — a walk never duplicates on screen.
	//   - ID NORMALIZATION: fetched rows carry the serve's RAW message
	//     id, but the transcript's stream conventions mint "bossmsg-"+id
	//     (assistant) / "user-"+<local seq> (user). Rows splice under
	//     "bossmsg-"+id / "user-"+id (pagerRowID), so an ASSISTANT row
	//     already on screen dedupes by id verbatim. The USER echo id can
	//     never collide (local seq), so user rows dedupe by TEXT
	//     multiplicity against the baseline (live echoes round-trip the
	//     prompt verbatim through the serve).
	//   - BASELINE: pagerBaseIDs + pagerBaseUtext snapshot the transcript
	//     AT SEED LANDING (hydrate + any pre-seed live echoes) and NEVER
	//     re-capture — spliced pages stay out of them, and because pages
	//     arrive strictly descending (rows ascending within), text
	//     multiplicity consumption aligns with the hydrated region's own
	//     occurrences before any genuinely-older same-text row arrives.
	//     A same-text user message OLDER than the whole hydrated region
	//     survives (its multiplicity is already zeroed).
	//   - pagerSeedFailed: the seed hop died → the walk stays UNSEEDED /
	//     unarmed (scroll-top inert) until a backend reconnect re-opens
	//     it; pagerNoSeam latches a seam-less backend (harness stubs) so
	//     the probe never repeats. Failures on older hops never banner —
	//     the pager's 3-strike latch is the whole surface.
	pager           *panels.ThreadPager
	pagerSession    string
	pagerSeeding    bool
	pagerSeeded     bool
	pagerSeedFailed bool
	pagerNoSeam     bool
	pagerBaseIDs    map[string]bool
	pagerBaseUtext  map[string]int

	// resumePin — an explicit opencode session id to boot into (main's
	// -s/--session flag, threaded via WithResumeSession). Set pre-Start:
	// it beats session.json's stored PrimaryID and skips the 4-day
	// freshness gate (deliberate resume semantics); "" = the normal
	// restore path.
	resumePin string

	// serverURL — the attach target main booted on (--server / env);
	// re-used by the /backend swap's backendFor re-construction
	// ("" = each transport spawns/resolves its own).
	serverURL string
	// emitFn — main's tea-loop bridge, injected post-construction via
	// SetEventSink (the same function the boot backend's Start drains
	// into). The /backend swap goroutine-starts the new transport on it.
	// nil in harnesses: swap still flips m.backend, starts nothing.
	emitFn func(state.Event)

	// execSession — the /session picker accept's exec-replace intent:
	// accept = quit + relaunch as `theboringoffice -s <id>` (recorded by
	// acceptSessionPick in session_picker.go, read by cmd's post-Run path
	// via ExecRequest). "" = a normal quit, no relaunch.
	execSession string

	// proj — cached project/git-branch info feeding the top bar right
	// segment (internal/projinfo; TTL-bounded, exec at most once per TTL
	// and ALWAYS off-frame: past the TTL the stale value is served while
	// a background goroutine re-probes git).
	proj *projinfo.Cache

	// boot — the animated ASCII splash shown while the backend warms up.
	// bootDone flips on the splash's done/skip msg (or any key); once done
	// the gate below routes every msg into the normal Update switch.
	boot     Boot
	bootDone bool
}

// permPrompt is a pending permission request — boss or child, the Agent
// field names the requester in the popover header.
type permPrompt struct {
	ID       string
	ToolName string
	Summary  string
	Agent    string // display name of the requesting agent ("boss" or child)
}

// permQueue — the pending-permission stack. pending is the fifo of asks
// waiting on an answer (pending[0] is what the popover displays); escd
// holds esc'd-but-unanswered asks, most recent last, /perm re-opens the
// newest. An ask lives in exactly ONE of the two slices.
type permQueue struct {
	pending []*permPrompt
	escd    []*permPrompt
}

// front is the displayed ask (pending[0]), nil while nothing is pending.
func (q *permQueue) front() *permPrompt {
	if len(q.pending) == 0 {
		return nil
	}
	return q.pending[0]
}

// resolve drops an id from BOTH slices (a server-side resolution races the
// popover and the esc'd pile alike). True when the dropped entry was the
// displayed front, i.e. the popover has to close or advance.
func (q *permQueue) resolve(id string) bool {
	displayed := q.front() != nil && q.front().ID == id
	q.pending = dropPrompt(q.pending, id)
	q.escd = dropPrompt(q.escd, id)
	return displayed
}

// view renders the queue front as the panel view — Index is the 1-based
// position of the displayed ask (the front is always 1), Total the queue
// depth, so the header reads "1 of N" while asks stack. nil = close.
func (q *permQueue) view() *panels.PermissionView {
	p := q.front()
	if p == nil {
		return nil
	}
	return &panels.PermissionView{
		ID:       p.ID,
		ToolName: p.ToolName,
		Summary:  p.Summary,
		Agent:    p.Agent,
		Index:    1,
		Total:    len(q.pending),
	}
}

// dropPrompt removes the first prompt with the given id (ids are unique
// per wire request — one drop is enough).
func dropPrompt(list []*permPrompt, id string) []*permPrompt {
	for i, p := range list {
		if p.ID == id {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// questionHold is a pending boss question request being paged through the
// chat panel's question popover as a WIZARD: IDs batches every pending
// wire request id of one question call (the SAME accumulated answer set
// replies to each batched id), Items is the request's pages (one
// state.QuestionItem per asked question), Answers accumulates one entry
// per page (a checkbox page's picked labels, [label] for radio,
// [text] for free-text; nil until that page is answered) and Cursor is
// the page currently on display.
type questionHold struct {
	IDs     []string
	Items   []state.QuestionItem
	Answers [][]string // one slot per Item, nil until that page is answered
	Cursor  int        // current wizard page
}

// legacyQuestionPage degrades a FLATTENED EvQuestion (Text only, no
// structured Questions) to one wizard page: a flat options list in
// ToolSummary ("a | b | c", anything but the "free-form answer" sentinel)
// becomes radio options, everything else a free-text page. Best-effort
// migration path for backends that predate the structured wire.
func legacyQuestionPage(ev state.Event) state.QuestionItem {
	page := state.QuestionItem{Question: ev.Text}
	if ev.ToolSummary != "" && ev.ToolSummary != "free-form answer" {
		for _, opt := range strings.Split(ev.ToolSummary, " | ") {
			if opt = strings.TrimSpace(opt); opt != "" {
				page.Options = append(page.Options, state.QuestionOption{Label: opt})
			}
		}
	}
	return page
}

// views renders the hold's CURRENT page as the panel's QuestionView — the
// kind is classified from the item's shape (options? multiple? confirm?)
// by the panels package, which owns the popover; Index is 1-based for the
// "1 of N" header. nil when nothing is open (SetQuestion closes on nil).
func (m *Model) questionView(h *questionHold) *panels.QuestionView {
	if h == nil || h.Cursor < 0 || h.Cursor >= len(h.Items) {
		return nil
	}
	it := h.Items[h.Cursor]
	return &panels.QuestionView{
		ID:       h.IDs[0],
		Question: it.Question,
		Header:   it.Header,
		Kind:     panels.ClassifyQuestion(it),
		Options:  it.Options,
		Index:    h.Cursor + 1,
		Total:    len(h.Items),
	}
}

// summary renders the accumulated answer set as the user bubble's short
// trace: each answered page renders its picks joined ", " (free-text
// pages recorded [text] render the text), pages joined " · ".
func (h *questionHold) summary() string {
	pages := make([]string, 0, len(h.Answers))
	for _, a := range h.Answers {
		if len(a) > 0 {
			pages = append(pages, strings.Join(a, ", "))
		}
	}
	return strings.Join(pages, " · ")
}

// chatSentMsg fires after backend.Send succeeds — the local user bubble and
// the typing placeholder are appended through the normal reducer path.
// agent carries the wire tag the send rode ("plan" in plan mode — F4's
// planSendPending tally rides it), "" for any ordinary build send.
type chatSentMsg struct {
	text  string
	agent string
}

// busySentMsg fires after a FREE-SEND resolves — a prompt that went
// straight to the backend while the boss was mid-turn (the serve queues it
// server-side, draining after the current turn). The model tallies it for
// the busy-status compose. agent is chatSentMsg's twin F4 field.
type busySentMsg struct {
	text  string
	agent string
}

// busySendReqMsg is the panel's busy-turn hand-off: Enter landed while the
// boss was mid-turn, but the ROUTING decision (boss server-queue vs
// concierge) needs the model's LIVE state, so the send happens in Update —
// see routeBusySend.
type busySendReqMsg struct {
	text string
	atts []state.Attachment
}

// conciergeSentMsg fires after backend.SendConcierge resolves — the office
// placeholder/answer bubbles arrive via the EvChatOffice seam.
type conciergeSentMsg struct {
	text   string
	notice bool // first concierge hop this turn — print after Send succeeds
}

// idleWrapSentMsg — the recap prompt reached the boss. Echo + typing
// bubble ride the backend event stream (same as a snapshot follow-up).
type idleWrapSentMsg struct{}

// idleWrapFailMsg — recap send missed the wire; Update posts the
// fallback office recap so the member is not left ghosted.
type idleWrapFailMsg struct{ err error }

// chatNoticeMsg is the chat panel's office-notice seam (attachment events:
// cap eviction, backspace removal, image-paste platform gaps).
type chatNoticeMsg struct{ text string }

// linkOpenMsg — the `o` target card's ferried choice (sessionPickMsg's
// twin): the model closes the card and runs the open through the verdict
// seam.
type linkOpenMsg struct{ target panels.LinkTarget }

// linkOpenCancelMsg — esc on the `o` target card: zero side effects, only
// the card closes (the transcript mark underneath stays).
type linkOpenCancelMsg struct{}

// browserOpenMsg — the exec verdict's landing (clipboardResultMsg's
// twin): a nil error rides the activity log as "→ opened: <name>"; an
// error becomes a dim "could not open: <reason>" transcript row — a
// browser hiccup is never fatal.
type browserOpenMsg struct {
	target panels.LinkTarget
	err    error
}

// sendErrMsg fires when the backend rejects a prompt.
type sendErrMsg struct{ err error }

// queueSendErrMsg fires when a backlog flush send (batch or single) is
// rejected. batch + !retry gets ONE respawn (ResetPrimary + resend the same
// composed batch); retry=true (or single) just surfaces the error.
type queueSendErrMsg struct {
	err   error
	items []queueEntry
	batch bool
	retry bool
}

// slashMsg fires when the chat input starts with "/" — local command, never
// sent to the backend.
type slashMsg struct{ text string }

// enqueueMsg fires when Enter lands while a boss reply is pending — the
// text joins the model-level queue instead of reaching the backend. The
// staged attachments ride along so the flush still has their files.
type enqueueMsg struct {
	text string
	atts []state.Attachment
}

// queueFlushMsg (400ms tick chain) flushes the next queued message.
type queueFlushMsg struct{}

// permAnswerMsg fires when the user answers an open permission prompt
// (y/a/n → "once"/"always"/"reject").
type permAnswerMsg struct{ response string }

// permLaterMsg fires on esc — the prompt stays pending, re-openable with
// /perm.
type permLaterMsg struct{}

// mcpStatusMsg ferries the /mcp (and /mcp reconnect) backend round trip
// back into Update — the render itself happens model-side so the panel's
// pure renderer (panels.RenderMCPStatus) gets the sidebar width.
type mcpStatusMsg struct {
	servers     []state.MCPServer
	err         error
	reconnected string // echo of the server /mcp reconnect <name> targeted
}

// questionAnswerMsg fires when the user confirms the CURRENT page of an
// open question wizard — the popover hands over its QuestionAnswer (the
// selected option labels and/or the free-text buffer); the model records
// it as that page's answers entry and advances. On the LAST page the
// accumulated [][]string goes through AnswerQuestion (this is the fix: a
// plain Send parks the opencode loop at the question reply API forever).
type questionAnswerMsg struct{ ans panels.QuestionAnswer }

// questionLaterMsg fires on esc in an open question wizard — the WHOLE
// request defers (pages + recorded answers intact), re-openable with
// /question.
type questionLaterMsg struct{}

// pagerSeedMsg — the eager attach-time seed hop's landing (METADATA-ONLY:
// NextCursor + HasMore feed pager.Seed; the page's rows are DISCARDED —
// the hydrated tail is already richer, and splicing them would fight the
// hydrate/live content for the tail region). sid guards staleness: a
// landing from a superseded binding (post-/new, post-respawn) drops.
type pagerSeedMsg struct {
	sid  string
	page state.SessionMessagesPage
	err  error
}

// pagerOlderMsg — one gesture-armed older-page hop's landing (the walk's
// splice input; rows still carry the serve's RAW ids — normalization +
// baseline dedupe happen app-side in applyOlderPage). sid guards
// staleness exactly like pagerSeedMsg's.
type pagerOlderMsg struct {
	sid  string
	page state.SessionMessagesPage
	err  error
}

// stopWorkMsg fires on a DOUBLE-esc in the main chat input — the chat
// panel's stop seam; handled exactly like /stop (abort + clean unwind).
// The ferry keeps the model value copy in Update the single writer.
type stopWorkMsg struct{}

// stopAbortResultMsg ferries the async /stop abort round trip back to the
// UI goroutine (the mcpStatusMsg twin): AbortSessions is the NETWORK half
// of /stop, so it rides a tea.Cmd — the office unwound synchronously
// already; here only the remote verdict lands (one dim note on failure,
// silence on success — the G1 contract, off the UI goroutine).
type stopAbortResultMsg struct{ err error }

// btwOfficeMsg ferries the async /btw NewOffice result back to the UI
// goroutine: the backend spawn + teardown rides a tea.Cmd so the 30s
// drain never parks the input.
type btwOfficeMsg struct {
	err      error
	trailing string // message after "/btw " to send on success
}

// doneOfficeMsg ferries the async /done SwapPrimary result back to the UI
// goroutine: same pattern as btwOfficeMsg — the teardown+spawn never parks
// the input.
type doneOfficeMsg struct{ err error }

// backendBuildMsg lands after a potentially slow BackendFactory call. The
// UI loop installs the completed transport in one short holder operation.
type backendBuildMsg struct {
	name        string
	oldName     string
	backend     state.Backend
	resumeID    string
	bypass      bool
	bypassValue bool
	transition  uint64
}

type backendReadyMsg struct{ result backendBuildMsg }

// backendStartMsg lands only after a fresh bypass transport has completed
// Start. Until then the current holder continues admitting work to the old
// generation; a failed start therefore cannot black out the office.
type backendStartMsg struct {
	result backendBuildMsg
	err    error
}

// backendStopMsg is the asynchronous teardown verdict for a retired backend.
// A failed Stop is intentionally non-fatal: the replacement is already live.
type backendStopMsg struct{ err error }

type backendConfigSaveMsg struct{ err error }

// armClearMsg — the ctrl+q quit arm's own expiry tick landed (scheduled
// with quitArmWindow by the arming press): retires the arm + its toast.
type armClearMsg struct{}

// Option — a functional New option (additive: existing New(b, cfg)
// callers compile untouched).
type Option func(*Model)

// WithResumeSession pins an explicit opencode session id to boot into
// (main's -s/--session flag): the restore leg routes it into the backend's
// PrimaryOverride INSTEAD of session.json's stored id and skips the 4-day
// freshness gate — deliberate resume semantics for when the automatic
// last-chat restore is not enough. "" is a no-op.
func WithResumeSession(id string) Option {
	return func(m *Model) { m.resumePin = id }
}

// WithServerURL records the attach target main resolved for this boot
// (--server flag / THEBORINGOFFICE_SERVER): the /backend swap re-constructs
// transports through backendFor with the same baseURL. "" = spawn mode.
func WithServerURL(u string) Option {
	return func(m *Model) { m.serverURL = u }
}

// WithBypassPermissions restores the persisted bypass-permissions state at
// boot. When on, it lands SetBypassPermissions on the backend BEFORE Start
// (main owns Start) and arms the model flags so the topbar badge and
// auto-approve gates are active from the first frame.
func WithBypassPermissions(on bool) Option {
	return func(m *Model) {
		if !on {
			return
		}
		if bb, ok := m.backend.(bypassBackend); ok {
			bb.SetBypassPermissions(true)
		}
		m.bypassPerms = true
		m.bypassDesired = true
	}
}

// SetEventSink injects the tea-loop bridge main hands the boot backend's
// Start (goroutine → tea.Program.Send). The /backend swap starts the NEW
// transport on the same sink, so swapped-in events arrive through exactly
// the lane the loop already drains. nil = swap starts nothing (harnesses).
func (m *Model) SetEventSink(fn func(state.Event)) { m.emitFn = fn }

// New builds the app around a backend + the brain.json config. cfg is
// nil-tolerant (config.Default() substituted — headless stubs and harnesses).
// backend.Start is NOT called here — main owns that (goroutine →
// tea.Program.Send).
func New(b state.Backend, cfg *config.Config, opts ...Option) Model {
	if cfg == nil {
		cfg = config.Default()
	}
	ambientOn = cfg.UI.AmbientChatter

	bossName := cfg.Boss.Name
	if bossName == "" {
		bossName = "boss (oikonomos)"
	}
	bossShort := bossName
	if i := strings.IndexAny(bossShort, " \t"); i > 0 {
		bossShort = bossShort[:i]
	}

	termTab := newTermTabWrap()
	browserTab := panels.NewBrowser()
	// The plan editor is built BEFORE the chat callback: the closure is a
	// one-time capture, so the CURRENT plan/build mode at send time
	// reaches it only through the shared pane pointer (see paneAgent).
	plan := panels.NewPlanEditor()
	// The office owns the agent mode; the pane's badge MIRRORS it. Its
	// constructor rests at "plan" (it is a plan editor) — normalize to the
	// office's build-mode rest state so paneAgent never misroutes a
	// build-mode prompt onto the "plan" wire tag.
	plan.SetMode(agentModeBuild)
	current := newCurrentBackend(b)
	chat := panels.NewChat(func(text string, atts []state.Attachment) tea.Cmd {
		return currentBackendSend(current, plan, text, atts)
	})
	chat.SetBossShortName(bossShort)
	agents := panels.NewAgents()
	agents.SetBossName(bossName)
	activity := panels.NewActivity()
	m := Model{
		backend:           b,
		currentBackend:    current,
		recentToolOutputs: map[string]string{},
		cfg:               cfg,
		gov:               &governor{lastBusy: time.Now()},
		bossName:          bossName,
		bossShort:         bossShort,
		st:                initialState(b.Mode()),
		chat:              chat,
		agents:            agents,
		activity:          activity,
		termTab:           termTab,
		browser:           browserTab,
		plan:              plan,
		planTemplate:      plan.Value(),
		agentMode:         agentModeBuild,
		activeThink:       map[string]bool{},
		focused:           true, // default: unknown == focused — never false-ping
		permNotifyIDs:     map[string]bool{},
		social:            newSocialClock(),
		lastDispatchTick:  -1,
		tabs: panels.NewTabs(
			chat,
			termTab,
			agents,
			panels.NewBoard(),
			panels.NewMail(),
			activity,
			// git rides index 6: the floor click pins activity at 5, so git
			// could only ever APPEND past it. Empty root → the panel
			// renders its "git unavailable" line.
			panels.NewGit(func() gitx.Repo { r, _ := gitx.Root(""); return r }()),
			// the browser is NOT here: it lives on the LEFT pane's
			// floor|browser slot (app/browser.go) — this strip stays at
			// seven tabs, indexes AND digit jumps unchanged.
		),
		keys: NewKeyMap(),
		boot: NewBoot(0, 0),
	}
	// Queue + permission seams: the panel owns the keys, the model owns the
	// queue/prompt state; callbacks ferry over tea.Msgs so the model value
	// copy in Update stays the single writer.
	chat.SetEnqueue(func(text string, atts []state.Attachment) tea.Cmd {
		return func() tea.Msg { return enqueueMsg{text: text, atts: atts} }
	})
	// Busy-send seam: while a boss reply is pending (busy, NOT roadblocked)
	// the panel hands Enter here — but WHERE the prompt goes is a live-state
	// decision (concierge vs the boss's server-side queue), so the closure
	// only ferries the text into the model and routeBusySend decides in
	// Update (busySendReqMsg).
	chat.SetBusySend(func(text string, atts []state.Attachment) tea.Cmd {
		return func() tea.Msg { return busySendReqMsg{text: text, atts: atts} }
	})
	// Attachment notices (cap eviction, chip removal, image-paste platform
	// gaps) surface as office chat notices like every other local outcome.
	chat.SetNoticeHandler(func(text string) tea.Cmd {
		return func() tea.Msg { return chatNoticeMsg{text: text} }
	})
	chat.SetPermissionHandlers(
		func(response string) tea.Cmd {
			return func() tea.Msg { return permAnswerMsg{response: response} }
		},
		func() tea.Cmd {
			return func() tea.Msg { return permLaterMsg{} }
		},
	)
	chat.SetQuestionHandlers(
		func(ans panels.QuestionAnswer) tea.Cmd {
			return func() tea.Msg { return questionAnswerMsg{ans: ans} }
		},
		func() tea.Cmd {
			return func() tea.Msg { return questionLaterMsg{} }
		},
	)
	// /session picker seams: enter accepts a row (the model re-anchors
	// the office onto its id), esc cancels (zero side effects) — both
	// ferry over tea.Msgs so the model value copy stays the single writer.
	chat.SetSessionPickerHandlers(
		func(id string) tea.Cmd {
			return func() tea.Msg { return sessionPickMsg{id: id} }
		},
		func() tea.Cmd {
			return func() tea.Msg { return sessionPickCancelMsg{} }
		},
	)
	// `o` target-card seams: enter fires the pick (the model closes the
	// card and runs the open; the exec verdict lands as browserOpenMsg),
	// esc cancels with zero side effects (the transcript mark survives).
	chat.SetLinkPickHandlers(
		func(t panels.LinkTarget) tea.Cmd {
			return func() tea.Msg { return linkOpenMsg{target: t} }
		},
		func() tea.Cmd {
			return func() tea.Msg { return linkOpenCancelMsg{} }
		},
	)
	// Double-esc seam: esc-esc in the main chat input aborts the running
	// turn. The panel owns the key timing (500ms window, modal/picker esc's
	// never count); the model owns stopWork — the same /stop path.
	chat.SetStopHandler(func() tea.Cmd {
		return func() tea.Msg { return stopWorkMsg{} }
	})
	m.tabs.SetCompact(m.compact())
	m.chat.SetCompact(m.compact())
	m.tabs.SetState(m.st)

	// Top bar's project+branch feed: cached, exec-bounded, ""-safe in demo
	// (projinfo falls back to os.Getwd when sessDir is "").
	m.proj = projinfo.DefaultCache()

	// Options land BEFORE the restore leg reads them (WithResumeSession
	// feeds resumePin) — construction above stays option-free.
	for _, opt := range opts {
		opt(&m)
	}

	// Office-session restore (LIVE ONLY — demo is a scripted tour and a
	// restored real transcript would confuse it; per the ruling, demo
	// skips). The PrimaryOverride MUST land before backend.Start (main
	// owns Start) so the saved boss session wins the boot choice; the
	// transcript/roster hydration is local state and safe pre-boot.
	if b != nil && b.Mode() == state.ModeLive {
		if dir, err := os.Getwd(); err == nil {
			m.sessDir = dir
			if m.resumePin != "" {
				// EXPLICIT RESUME PIN (-s/--session): beats session.json's
				// stored id AND skips the 4-day freshness gate — the member
				// asked for THIS session deterministically. resolvePrimary
				// (backend) still verifies the id server-side and degrades
				// open to find-or-create on a 404, so the boot never
				// hard-fails.
				pinned := false
				if ps, ok := b.(primarySeamBackend); ok {
					ps.PrimaryOverride(m.resumePin)
					pinned = true
				}
				sf, sfOK := LoadSession(dir)
				switch {
				case !pinned:
					// A live harness stub without the seam: never fake the
					// resume (that would be a silent substitution).
					m.noticeErr("-s/--session: this backend cannot pin a session — starting normally")
				case sfOK && sf.primaryIDFor(cfg.Backend.ResolvedName()) == m.resumePin:
					// Pin == the stored id: hydrate as today (the skip
					// guard below fires only on a DIFFERENT stored
					// transcript).
					m.hydrateSession(sf)
					m.bootDone = true
				default:
					// HYDRATE-SKIP GUARD: the pinned server session is not
					// the one session.json describes (or no file exists) —
					// hydrating that stale, unrelated transcript would mask
					// the pinned one.
					m.appendNotice(fmt.Sprintf("resumed session %s (explicit pin) · /new for a fresh office", m.resumePin), "boot")
				}
			} else if sf, ok := LoadSession(dir); ok && sf.Fresh() {
				// Per-backend restore pin: opencode rides the legacy
				// PrimaryID entry, claudecode its own primaryIDs entry —
				// a claude session must NEVER pin an opencode serve id
				// (fallback-literacy lives in primaryIDFor).
				if pin := sf.primaryIDFor(cfg.Backend.ResolvedName()); pin != "" {
					if ps, ok := b.(primarySeamBackend); ok {
						ps.PrimaryOverride(pin)
					}
				}
				m.hydrateSession(sf)
				// A restored office is already warm — skip the boot
				// splash so the transcript + restore notice are the
				// first frame (the splash opens cold starts only).
				m.bootDone = true
			}
		}
	}
	return m
}

// SelectTab activates a sidebar tab by name ("chat", "terminal", "agents",
// …). Used by harnesses (uishot) before the run starts; selecting the
// terminal tab lazy-spawns its shell on this first visit.
func (m *Model) SelectTab(name string) bool {
	ok := m.tabs.SetActiveByTitle(name)
	if ok {
		m.normalizeTermCapture() // leaving captured → release before re-entry
		m.maybeSpawnTerminal()
	}
	return ok
}

// ActiveTabIndex reports the selected tab position (harness seam for the
// click proofs — double-clicked floor sprites jump to chat, 0).
func (m Model) ActiveTabIndex() int { return m.tabs.ActiveIndex() }

// LeftTabIndex reports the LEFT pane's switcher position (leftTabFloor |
// leftTabBrowser) — the harness seam for the browser layout proofs.
func (m Model) LeftTabIndex() int { return m.leftTab }

// SetSoundBus injects the sound engine's bus (nil disables sound). The app
// only calls Play — the engine is owned elsewhere.
func (m *Model) SetSoundBus(bus SoundBus) {
	m.snd = bus
}

// SetNotifyBus injects the notification engine's bus (nil disables desktop
// notifications). The app only calls Notify — the engine is owned elsewhere.
func (m *Model) SetNotifyBus(bus NotifyBus) {
	m.notifyBus = bus
}

// playSound — reducer-property sound hook; no-ops while no bus is injected.
func (m *Model) playSound(name string) {
	if m.snd != nil {
		m.snd.Play(name)
	}
}

// State returns the current office state (read-only harness seam — uishot
// --social asserts on bubbles/sprites through it).
func (m Model) State() state.OfficeState {
	return m.st
}

// Init arms the first power-governed tick plus the boot splash's own
// frame ticker; applyEvent re-arms the office tick every cycle.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), bootTick())
}

// tickCmd re-arms the animation tick at the delay the governor picks for
// THIS cycle (busy signals from the current state + modals + think stream,
// idle duration from the drift clock). Busy refreshes lastBusy; 60s of
// continuous quiet in auto mode slips into screensaver cadence.
func (m *Model) tickCmd() tea.Cmd {
	modalOpen := m.permQ.front() != nil || m.question != nil
	thinkActive := len(m.activeThink) > 0
	now := time.Now()
	if officeBusy(m.st, modalOpen, thinkActive) {
		m.gov.lastBusy = now
	}
	delay := TickDelay(m.st, m.cfg, thinkActive, modalOpen, now.Sub(m.gov.lastBusy))
	m.gov.tickCount++
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return state.Event{Kind: state.EvTick}
	})
}

// Ticks — tick commands armed this run (uisot power proof).
func (m Model) Ticks() int { return m.gov.tickCount }

// Config — the live brain.json (nil-tolerant accessors live elsewhere).
func (m Model) Config() *config.Config { return m.cfg }

// FrameCacheStats — app-frame cache counters (uisot proof).
func (m Model) FrameCacheStats() (hits, misses uint64) {
	return m.gov.frameHits, m.gov.frameMisses
}

// Update routes keys, backend events and component ticks.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Boot gate: until the splash is done (cascade + ready, 4s cap, or a
	// keypress skip) INPUT feeds the boot component only — the office
	// tick, backend events and resizes keep flowing to the normal switch
	// underneath so the office is already live when the splash lifts.
	// The first backend event marks the boot uplink ready (goroutines
	// emit after Start connects; demo emits instantly).
	if !m.bootDone {
		switch msg := msg.(type) {
		case bootDoneMsg, bootSkipMsg:
			m.bootDone = true // splash lifts; the msg itself is harmless below
		case bootTickMsg:
			nb, cmd := m.boot.Update(msg)
			m.boot = nb
			if m.boot.Done() {
				m.bootDone = true
			}
			return m, cmd
		case tea.KeyPressMsg:
			// keys dismiss the splash AND keep flowing — the user's first
			// typed character must not vanish into the boot skip (uisot
			// workloads type at t≈0; ctrl+c rides the normal persist-quit
			// arm below).
			m.bootDone = true
		case tea.MouseClickMsg:
			// a click dismisses the splash AND the same click lands — the
			// floor/tab under it is the thing the user aimed at (uisot
			// click proofs depend on one click acting, not two).
			m.bootDone = true
		case tea.WindowSizeMsg:
			nb, _ := m.boot.Update(msg) // size also flows to resize below
			m.boot = nb
		case state.Event:
			m.boot.SetReady() // then keeps flowing — the office warms behind the splash
		case tea.QuitMsg:
			return m, tea.Quit
		}
	}
	var cmds []tea.Cmd
	// Auto-release invariant: any tab change routed in above (click,
	// event-driven SetActive, spawn-failure fallback) drops a stale shell
	// capture before the next key routes — capture never escapes its tab.
	m.normalizeTermCapture()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.BackgroundColorMsg:
		// Device light/dark: auto-adapt the theme to the terminal
		// background unless the user pinned one. Office floor re-points
		// inside; auto picks never persist, and macOS dark↔light flips
		// re-theme live as spontaneous events.
		chrome.SetThemeAuto(msg.IsDark())
	case tea.FocusMsg:
		// terminal regained focus: the desktop-ping gate closes until the
		// next blur. No frameNonce — focus alone repaints nothing.
		m.focused = true
	case tea.BlurMsg:
		// terminal lost focus: unfocused pings are live again. A LIVE
		// permission cohort fires its ping immediately — the member looked
		// away DURING a block, exactly the moment the nudge exists for.
		m.focused = false
		if len(m.permNotifyIDs) > 0 {
			if p := m.permCohortFront(); p != nil {
				m.fireNotification("permission", permNotifyBody(p.Agent, p.ToolName))
			}
		}
	case tea.KeyPressMsg:
		// keys can mutate panel ephemera (textarea, scroll) the state
		// digest can't see — invalidate the frame cache conservatively.
		m.frameNonce++
		// esc clears a live transcript selection FIRST (webpage rule):
		// while a highlight is up on the chat tab the key belongs to it
		// (it never reaches the textarea/dbl-esc and never unfolds a thread).
		// The thread-focus view's esc claims BEFORE the selection's though —
		// it is the view's ONE leave key, and the main chat's selection
		// (a) is behind a fullscreen pane and (b) must not swallow it.
		// The `o` target card's esc ALSO claims before the selection's —
		// the card floats ABOVE the mark like the focus pane: its esc
		// closes the card first, the mark's esc comes after. The left-pane
		// browser's esc outranks the selection too — while the switcher
		// sits on browser the key belongs to the pane (its leave-to-floor
		// contract), never to a chat-region mark.
		if msg.String() == "esc" && m.threadFocus == nil && m.tabs.ActiveIndex() == 0 && !m.browserActive() && m.chat != nil && !m.chat.LinkPickerOpen() && m.chat.SelectionActive() {
			m.chat.ClearSelection()
			m.sel = mselIdle
			break
		}
		if cmd := m.handleKey(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.MouseClickMsg:
		// clicks mutate the same panel ephemera (thread toggles, tab
		// switches, roster highlight) — same cache invalidation as keys.
		m.frameNonce++
		// The plan pane owns presses inside its own body region: the
		// pane's text selection ARMS here (the release decides the fate —
		// caret click vs copy-drag; the transcript's press-fate seam, one
		// surface down). Out-of-region presses fall through to the legacy
		// routing untouched.
		if msg.Button == tea.MouseLeft && m.planPaneRegionHit(msg.X, msg.Y) {
			if m.chat != nil {
				m.chat.ClearSelection() // webpage rule: a press elsewhere clears the transcript mark
			}
			m.sel = mselIdle
			col, vrow := m.planBodyCoords(msg.X, msg.Y)
			m.openPlanForEdit()
			m.plan.SelectionBeginAt(col, vrow)
			m.planSelArmed = true
			m.planSelDragged = false
			break
		}
		if cmd := m.handlePress(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.MouseReleaseMsg:
		// an armed plan-pane drag owns releases first (never both marks
		// live — the chat's region and the pane's region are disjoint).
		// A dragged release finalizes + copies through the same
		// verdict/toast seam as ctrl+c; a MOTIONLESS release decided
		// caret placement at the press and claims nothing further.
		if m.planSelArmed {
			m.planSelArmed = false
			if m.planSelDragged {
				m.frameNonce++
				col, vrow := m.planBodyCoords(msg.X, msg.Y)
				text, n, err := m.plan.SelectionFinish(col, vrow)
				if n == 0 && err == nil {
					m.plan.ClearSelection() // the drag decided nothing (transcript rule)
				}
				if cmd := m.planCopyVerdictCmd(text, n, err); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			break
		}
		// only an armed selection drag cares about releases — anything
		// else drops release events silently (no repaint, no forward).
		if m.sel == mselArmed {
			m.frameNonce++
			if cmd := m.handleRelease(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
		// The terminal tab's drag-select owns releases on its turf (the
		// chat's armed drag claims them first above — never both live).
		// Unarmed landings are cheap panel no-ops; the copy on a dragged
		// release paints, hence the nonce.
		if cmd, ok := m.sendTermMouse(msg); ok {
			m.frameNonce++
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case tea.MouseMotionMsg:
		// CellMotion reports motion ONLY while a button is pressed — that
		// is the selection drag's lifeblood. Battery rule: without an
		// armed drag the event is dropped with zero repaint cost.
		if m.planSelArmed {
			col, vrow := m.planBodyCoords(msg.X, msg.Y)
			if m.plan.SelectionDragTo(col, vrow) { // cheap compare per event
				m.frameNonce++ // the mark's pixels moved — repaint
				m.planSelDragged = true
			}
			break
		}
		if m.sel == mselArmed {
			m.frameNonce++
			m.handleMotion(msg)
			break
		}
		// Same lifeblood for the terminal tab's own drag (panel-side arm;
		// motion without one is a cheap no-op there).
		if _, ok := m.sendTermMouse(msg); ok {
			m.frameNonce++
		}
	case tea.MouseWheelMsg:
		// wheel scrolls the active panel (the default-arm's twin) — except
		// an open thread-focus, which owns the wheel for its own viewport
		// (the office underneath never scrolls behind the pane), and the
		// LEFT-pane browser, which owns it while the switcher sits on
		// browser (the chat's older-hop gesture below stays chat-scoped).
		m.frameNonce++
		if m.threadFocus != nil {
			if cmd := m.threadFocus.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
		if m.browserActive() {
			if cmd := m.browser.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
		if cmd := m.tabs.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		// TOP-GESTURE: a wheel-up that leaves the transcript on row 0 arms
		// ONE older-history hop (maybeArmOlder owns every guard).
		if msg.Button == tea.MouseWheelUp {
			if cmd := m.maybeArmOlder(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case chatSentMsg:
		// nothing local: backend.Send owns the echo (chat-user + pending boss
		// bubble) via the event stream — applying them here duplicated the bubbles.
		m.playSound("send")
		m.notifyDoneArmed = true // arm the done ping's one-shot debounce
		if msg.agent == agentModePlan {
			m.planSendPending++ // F4 — an in-flight plan-tagged turn
		}
	case busySendReqMsg:
		// the panel fired while the boss looked busy; route with live state
		cmds = append(cmds, m.routeBusySend(msg.text, msg.atts))
	case conciergeSentMsg:
		// the concierge backend owns the office echo (pending placeholder +
		// completion bubbles) via the EvChatOffice seam.
		m.playSound("send")
		if msg.notice {
			m.notice("office routed: boss busy → concierge")
		}
	case idleWrapSentMsg:
		m.notifyDoneArmed = true
		m.activity.Add(fmt.Sprintf("[%s] asked the boss for an idle recap", chrome.OfficeClock(m.st.Tick)))
		m.activityAdds++
		m.fireNotification("done", "the office went quiet — asking the boss for a recap")
	case idleWrapFailMsg:
		m.notice(idleWrapNotice)
		m.playSound("done")
		m.fireNotification("done", "the office went quiet — here's a recap")
	case busySentMsg:
		// free-queuing: straight to the server mid-turn — tally it so the
		// status compose + placeholder turn count keep the UI alive (the
		// client queue stays untouched).
		m.playSound("send")
		m.notifyDoneArmed = true // free-sends start real turns too → arm the done ping
		if msg.agent == agentModePlan {
			m.planSendPending++ // F4 — an in-flight plan-tagged turn
		}
		m.serverQueued++
		if m.chat != nil {
			m.chat.SetServerTurn(m.serverQueued)
		}
		m.applyBusyStatus()
		qdebugf("free-send: prompt went straight to the server (server-queued #%d this turn)", m.serverQueued)
	case chatNoticeMsg:
		// the chat panel's attachment notices join the office notice feed
		m.notice(msg.text)
	case sendErrMsg:
		if m.backendTransitioning {
			m.notice("backend restarting — please resend in a moment")
			cmds = append(cmds, m.applyEvent(state.Event{
				Kind: state.EvStatus,
				Text: "[theboringfloor] backend restarting — message not sent",
			}))
		} else {
			m.playSound("error")
			// F6 — a failed send is not just a transient statusline blip: the
			// transcript keeps the red row too (the next EvStatus overwrites
			// the line, never the transcript).
			m.noticeErr(fmt.Sprintf("send failed: %v", msg.err))
			cmds = append(cmds, m.applyEvent(state.Event{
				Kind: state.EvStatus,
				Text: fmt.Sprintf("[theboringfloor] send failed: %v", msg.err),
			}))
		}
	case backendBuildMsg:
		cmds = append(cmds, m.finishBackendTransition(msg))
	case backendReadyMsg:
		cmds = append(cmds, m.completeBackendTransition(msg.result))
	case backendStartMsg:
		cmds = append(cmds, m.completeBypassStart(msg))
	case backendStopMsg:
		if msg.err != nil {
			m.noticeErr("backend stop: " + msg.err.Error())
		}
	case backendConfigSaveMsg:
		if msg.err != nil {
			m.noticeErr("backend swap: brain.json save failed: " + msg.err.Error() + " — the swap holds for this session only")
		}
	case approveSentMsg:
		// F3 — the plan→build flip rides SEND ACCEPTANCE: the approved
		// plan's wire POST landed; NOW the office flips back (the pane
		// hides with the mode flip), the buffer persists for restore, the
		// dirty/restore latches reset, and the approval notice posts.
		if m.plan != nil {
			// approvePlan caps this snapshot before it crosses the wire;
			// preserve that accepted value exactly rather than rereading the
			// editable draft or transforming the completion payload again.
			approvedPlanTexts.Store(m.plan, msg.plan)
		}
		m.setAgentMode(agentModeBuild)
		m.restoredPlan = false
		if m.plan != nil {
			m.plan.SetUserDirty(false)
			m.plan.Blur()
		}
		m.playSound("send")
		m.notice(fmt.Sprintf("[office] plan approved — sent to build (%d chars)", len(msg.plan)))
	case approveErrMsg:
		// F3 rollback — the tag-flip never happened: plan mode KEPT, the
		// plan buffer untouched, the red row explains it in the transcript
		// (the statusline twin rides the EvStatus below).
		m.playSound("error")
		cmds = append(cmds, m.applyEvent(state.Event{
			Kind: state.EvStatus,
			Text: fmt.Sprintf("[theboringfloor] approve failed: %v", msg.err),
		}))
		m.noticeErr(fmt.Sprintf("approve failed — still in plan: %v", msg.err))
	case approvedPlanResult:
		if msg.err != nil {
			m.playSound("error")
			m.noticeErr(fmt.Sprintf("approved plan follow-up failed: %v", msg.err))
		}
	case queueSendErrMsg:
		// FAILURE RESPAWN — one per flush call: the boss session died at
		// Send; reset the primary and resend the SAME composed batch on the
		// fresh session. A retry failure just surfaces the error.
		if msg.batch && !msg.retry && !m.batchRespawned {
			if _, ok := m.team(); ok {
				// no cleanup here — the respawn retry resends the SAME
				// files, so the temp dirs must still exist.
				m.batchRespawned = true
				m.batchSentAt = time.Now()
				m.respawns++
				m.notice("boss went down — respawned a fresh session, resending batch")
				qdebugf("batch send failed (%v) — respawning (respawn #%d)", msg.err, m.respawns)
				cmds = append(cmds, m.resendBatchCmd(msg.items))
				break
			}
		}
		// terminal failure: no retry is coming for these attachments
		cleanupEntries(msg.items)
		m.playSound("error")
		cmds = append(cmds, m.applyEvent(state.Event{
			Kind: state.EvStatus,
			Text: fmt.Sprintf("[theboringfloor] send failed: %v", msg.err),
		}))
	case slashMsg:
		// slash handlers mutate panel-only visual state (thinking/tools/
		// diffs toggles, theme) — cover the frame cache with the nonce.
		m.frameNonce++
		if cmd := m.applySlash(msg.text); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case sessionListMsg:
		// the /session picker's async listing landed — panel-only state
		// the digest can't see, so cover the frame cache like slashMsg.
		m.frameNonce++
		m.handleSessionList(msg)
	case sessionPickMsg:
		m.frameNonce++
		if cmd := m.acceptSessionPick(msg.id); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case sessionPickCancelMsg:
		// esc cancels with zero side effects: only the card closes.
		m.frameNonce++
		if m.chat != nil {
			m.chat.CloseSessionPicker()
		}
	case linkOpenMsg:
		// the `o` target card submitted a target: the card closes and the
		// open rides the verdict seam (async — the UI goroutine never
		// shell-outs mid-frame).
		m.frameNonce++
		if m.chat != nil {
			m.chat.CloseLinkPicker()
		}
		cmds = append(cmds, m.openTargetCmd(msg.target))
	case linkOpenCancelMsg:
		// esc cancels with zero side effects: only the card closes, the
		// transcript mark stays (the esc-laws layer underneath).
		m.frameNonce++
		if m.chat != nil {
			m.chat.CloseLinkPicker()
		}
	case browserOpenMsg:
		// the exec verdict landed: success rides the activity tab
		// ("→ opened: <name>"), failure posts a dim transcript row
		// ("could not open: <reason>") — never fatal.
		m.frameNonce++
		if msg.err != nil {
			m.notice("could not open: " + msg.err.Error())
		} else {
			m.activity.Add(fmt.Sprintf("[%s] → opened: %s", chrome.OfficeClock(m.st.Tick), msg.target.Name))
			m.activityAdds++
		}
	case panels.BrowserPageMsg:
		// the browser tab's fetch landed (panel-only state → nonce): the
		// page forwards STRAIGHT to the pane (never the active-tab hop — a
		// mid-flight tab switch can't misdeliver it), then the /open
		// notice latch settles.
		m.frameNonce++
		if cmd := m.handleBrowserPage(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case panels.BrowserOpenedMsg:
		// the pane's OS-open verdict (a focused LOCAL link) — its note row
		// carries the outcome; no activity-tab line from the browser (spec §8).
		m.frameNonce++
		if cmd := m.handleBrowserOpened(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case panels.BrowserLeaveMsg:
		// the pane's q/esc — back to the chat tab (thread-focus's contract).
		m.frameNonce++
		m.handleBrowserLeave()
	case pagerSeedMsg:
		// the eager attach hop landed: seed the walk (or stay unseeded and
		// unarmed on failure — scroll-top simply never fetches). Rows are
		// DISCARDED by design (the hydrated tail is richer); the dedupe
		// baseline freezes onto the transcript as it stands NOW.
		if m.pager != nil && msg.sid == m.pagerSession {
			m.pagerSeeding = false
			if msg.err != nil {
				m.pagerSeedFailed = true
			} else {
				m.pager.Seed(msg.page.NextCursor, msg.page.HasMore)
				m.pagerSeeded = true
				m.capturePagerBaseline()
			}
		}
	case pagerOlderMsg:
		// the gesture-armed hop landed: splice + advance (or silently
		// strike toward the backoff latch).
		m.applyOlderPage(msg)
	case modelsListMsg:
		// the /model picker's async listing landed — picker + state field,
		// both invisible to the digest, so cover the frame cache like slashMsg.
		m.frameNonce++
		m.handleModelsList(msg)
	case modelPickMsg:
		// enter accepted a row: close the card and drive the EXISTING
		// /model-set slash path (model_picker.go's acceptModelPick).
		m.frameNonce++
		m.acceptModelPick(msg.ref)
	case modelPickCancelMsg:
		// esc cancels with zero side effects: only the card closes.
		m.frameNonce++
		m.closeModelPicker()
	case enqueueMsg:
		if len(m.queue) >= queueCap {
			m.noticeErr(fmt.Sprintf("backlog full (%d) — wait for the boss to catch up, or /queue clear", queueCap))
			cleanupAttachments(msg.atts) // never queued — nobody else will
		} else {
			n := len(m.queue) + 1
			ent := queueEntry{text: msg.text, atts: msg.atts}
			if tb, ok := m.team(); ok {
				// board row for the backlog item: title is the machine
				// first-N-chars clip of the typed text (an attach-only
				// item derives its title from the attachment names instead).
				title := clipRunes(msg.text, batchTitleClip)
				if msg.text == "" && len(msg.atts) > 0 {
					title = clipRunes("attachments: "+attachNames(msg.atts), batchTitleClip)
				}
				ent.boardID = tb.QueueItemStart(n, title)
			}
			m.queue = append(m.queue, ent)
			if m.chat != nil {
				m.chat.SetQueueLen(len(m.queue))
			}
			m.playSound("queued")
			qdebugf("enqueued %q as item #%d (board=%q, n=%d)", msg.text, n, ent.boardID, len(m.queue))
			m.notice(fmt.Sprintf("queued as item #%d — flushes as a batch when the boss frees up", n))
		}
	case queueFlushMsg:
		if len(m.queue) > 0 {
			cmds = append(cmds, m.flushQueued())
		}
	case permAnswerMsg:
		// y/a/n answers ONLY the displayed front, then the popover advances
		// to the next pending ask (or closes). The wire reply rides the
		// generic AnswerPermission seam — child holds sit in the same
		// backend pendingPerms map as the boss's.
		if p := m.permQ.front(); p != nil {
			pid, response := p.ID, msg.response
			m.permQ.pending = m.permQ.pending[1:]
			m.permQ.escd = dropPrompt(m.permQ.escd, pid) // defensive: an ask lives in exactly one slice
			delete(m.permNotifyIDs, pid)                 // the cohort shrinks; empty re-arms the next ping
			m.chat.SetPermission(m.permQ.view())
			// HOOKUP (browser_open.go): a parked browser-action hold
			// resolves LOCALLY — the modal answer drives the executor or
			// the rejection follow-up; the office-minted pid never rides
			// the backend's AnswerPermission wire.
			if handled, cmd := m.consumeBrowserActionPerm(pid, response); handled {
				cmds = append(cmds, cmd)
				break
			}
			current := m.currentBackend
			cmds = append(cmds, func() tea.Msg {
				err := current.lease(func(b state.Backend) error {
					return b.AnswerPermission(pid, response)
				})
				if err != nil {
					return sendErrMsg{err: err}
				}
				return nil
			})
		}
	case permLaterMsg:
		// esc parks the displayed front at the tail of the esc'd pile and
		// shows the next pending ask; /perm re-opens the most recent esc'd.
		if p := m.permQ.front(); p != nil {
			m.permQ.pending = m.permQ.pending[1:]
			m.permQ.escd = append(m.permQ.escd, p)
			m.chat.SetPermission(m.permQ.view())
			m.notice("esc'd permission pending (/perm)")
		}
	case questionAnswerMsg:
		if m.question != nil {
			hold := m.question
			if hold.IDs[0] == bypassConfirmID {
				// the /bypass enable confirm is OFFICE-LOCAL: the answer
				// never rides AnswerQuestion, the hold never parks into
				// /question. Exactly "enable" arms the mode (respawn
				// included); cancel/esc/custom text is a no-op.
				m.question = nil
				m.chat.SetQuestion(nil)
				m.chat.SetPermission(m.permQ.view())
				if len(msg.ans.Picks) == 1 && msg.ans.Picks[0] == "enable" {
					m.bypassDesired = true
					cmds = append(cmds, m.respawnForBypass())
				}
				break
			}
			// record THIS page's answer: the popover's picked labels win
			// (radio = one, checkbox = several), free-text pages record
			// the trimmed text as the one entry. A fully empty submission
			// is a no-op — the wizard stays on the current page.
			ans := msg.ans.Picks
			if len(ans) == 0 {
				if t := strings.TrimSpace(msg.ans.Text); t != "" {
					ans = []string{t}
				}
			}
			if len(ans) == 0 {
				break
			}
			hold.Answers[hold.Cursor] = append([]string(nil), ans...)
			if hold.Cursor < len(hold.Items)-1 {
				// more pages in this request — advance, re-render
				hold.Cursor++
				m.chat.SetQuestion(m.questionView(hold))
				break
			}
			// last page: submit the FULL accumulated answer set — one
			// AnswerQuestion per batched wire id, the same [][]string
			// (opencode's QuestionAnswer = string[] per asked question).
			m.question = nil
			m.chat.SetQuestion(nil)
			// the question popover hides the permission popover — re-push
			// the queue front now the region is free again.
			m.chat.SetPermission(m.permQ.view())
			current := m.currentBackend
			ids := append([]string(nil), hold.IDs...)
			answers := hold.Answers
			cmds = append(cmds, func() tea.Msg {
				err := current.lease(func(b state.Backend) error {
					for _, qid := range ids {
						if err := b.AnswerQuestion(qid, answers); err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					return sendErrMsg{err: err}
				}
				return nil
			})
			// the member's own answer set joins the transcript as a user
			// bubble where the answered request surfaced (the resolved
			// marker lands via the backend's resolved event).
			if trace := hold.summary(); trace != "" {
				m.st.Chat = capChat(appendChat(m.st.Chat, state.ChatMsg{
					ID:   "qans-" + ids[0],
					From: "user",
					Text: trace,
					At:   time.Now().UnixMilli(),
				}))
				m.tabs.SetState(m.st)
			}
		}
	case questionLaterMsg:
		if m.question != nil {
			if m.question.IDs[0] == bypassConfirmID {
				// esc CANCELS the /bypass arming confirm — the mode stays
				// OFF and nothing parks into /question (it is not a boss
				// question).
				m.question = nil
				m.chat.SetQuestion(nil)
				m.chat.SetPermission(m.permQ.view())
				break
			}
			m.questionEscd = m.question
			m.question = nil
			m.chat.SetQuestion(nil)
			// same re-push: the permission popover survives question defer.
			m.chat.SetPermission(m.permQ.view())
			m.notice("(question deferred — /question to reopen)")
		}
	case mcpStatusMsg:
		if msg.err != nil {
			m.noticeErr("mcp: " + msg.err.Error())
		} else {
			lines := panels.RenderMCPStatus(msg.servers, max(20, m.sidebar-4))
			if msg.reconnected != "" {
				lines = append([]string{"mcp: reconnected " + msg.reconnected}, lines...)
			}
			m.notice(strings.Join(lines, "\n"))
		}
	case stopWorkMsg:
		// double-esc in the main chat input == /stop (abort + unwind);
		// idle-safe: nothing runs → the abort seam no-ops, the unwind is
		// empty and only the status line reports like /stop would.
		if cmd := m.stopWork(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case stopAbortResultMsg:
		// the /stop abort hop landed back on the UI goroutine (it rode a
		// tea.Cmd — the network never parked the input). The unwind +
		// statusline are long done; on failure the office surfaces ONE dim
		// note, exactly like the old synchronous path (G1). Success is
		// silent. A late landing (office already moved on — /clear, a new
		// send) just appends the note, never a second unwind.
		if msg.err != nil {
			qdebugf("/stop: AbortSessions failed (unwound anyway): %v", msg.err)
			m.notice("/stop: abort signal failed remotely — office unwound anyway (the turn may still finish server-side; its reply lands as a fresh bubble)")
		}
	case btwOfficeMsg:
		// the /btw backend spawn landed back on the UI goroutine.
		if msg.err != nil {
			qdebugf("/btw: NewOffice failed: %v", msg.err)
			m.noticeErr("/btw: " + msg.err.Error())
			// Restore on failure.
			if m.btwSaved != nil {
				m.st.Chat = m.btwSaved.chat
				m.st.Tasks = m.btwSaved.tasks
				m.st.Mails = m.btwSaved.mails
				m.btwSaved = nil
			}
		} else {
			m.tabs.SetState(m.st)
			m.notice("btw session — esc or /done to return")
			if msg.trailing != "" {
				current := m.currentBackend
				cmds = append(cmds, func() tea.Msg {
					_ = sendViaCurrentBackend(current, msg.trailing, nil, "")
					return nil
				})
			}
		}
	case doneOfficeMsg:
		if msg.err != nil {
			qdebugf("/done: SwapPrimary failed: %v", msg.err)
			m.noticeErr("/done: " + msg.err.Error())
		} else if m.btwSaved != nil {
			// resumed INTO a btw session — the entry notice is already up
			m.notice("btw session — esc or /done to return")
		} else {
			m.notice("back from btw")
		}
	case armClearMsg:
		// the quit arm's expiry tick landed: a still-live arm old enough
		// retires (a YOUNGER re-arm survives — its own tick owns its
		// expiry, this landing just no-ops).
		if !m.quitArmAt.IsZero() && time.Since(m.quitArmAt) >= quitArmWindow {
			m.quitArmAt = time.Time{}
			m.frameNonce++ // the toast retires — the hint bar repaints
		}
	case approveArmClearMsg:
		// the approve arm's expiry tick landed (armClearMsg's twin):
		// retires a stale ctrl+x arm; a YOUNGER re-arm survives — its own
		// tick owns its expiry, this landing just no-ops.
		if !m.approveArmAt.IsZero() && time.Since(m.approveArmAt) >= approveArmWindow {
			m.approveArmAt = time.Time{}
			m.frameNonce++ // the toast retires — the hint bar repaints
		}
	case clipboardResultMsg:
		// darwin's pbcopy round-trip landed (selection.go): the toast
		// gates on the real verdict — success arms the frozen "Copied N
		// chars", failure rides the same seam as a warn. The old
		// OSC52-only path toasted unconditionally and lied whenever the
		// escape was swallowed.
		if msg.err != nil {
			cmds = append(cmds, m.armCopyNoteErr(msg.err))
		} else {
			cmds = append(cmds, m.armCopyNote(msg.n))
		}
	case copyNoteClearMsg:
		// the copy toast's own expiry tick landed (armClearMsg's twin):
		// retire a note old enough — a FRESHER re-arm's own tick owns its
		// expiry (a stale landing here just no-ops).
		if m.copyNote != "" && time.Since(m.copyNoteAt) >= copyNoteWindow {
			m.copyNote = ""
			m.frameNonce++ // the toast retires — the hint bar repaints
		}
	case state.Event:
		cmds = append(cmds, m.applyEvent(msg))
	case recentMessagesResult:
		if msg.err != nil {
			m.noticeErr("context: could not send recent messages to the boss — " + msg.err.Error())
		} else {
			m.notice(fmt.Sprintf("context: sent %d recent messages to the boss", msg.sent))
		}
	case imageRasterMsg:
		// the lazy image rasterize landed back on the UI goroutine
		// (images.go): rows into the chat panel, one owning-block repaint.
		m.frameNonce++
		if cmd := m.applyImageRaster(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.PasteMsg:
		// Terminal.app's cmd+v (bracketed paste): ONE deliberate path —
		// routePaste lands it on exactly one surface (never two, never
		// silently nowhere — the ignore case toasts a dim notice).
		m.frameNonce++
		if cmd := m.routePaste(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case panels.PlanPasteMsg:
		// the plan pane's own async clipboard read landed back (ctrl+v /
		// super+v while the editor was focused): deliver it to the pane
		// ONLY — a plan paste must never fall into the chat draft. A tool
		// miss degrades to ONE dim note (never silently).
		m.frameNonce++
		if msg.Err != nil {
			m.notice("paste failed: " + msg.Err.Error())
			break
		}
		if m.agentMode == agentModePlan && m.plan != nil && m.plan.Focused() {
			if cmd := m.plan.Update(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	default:
		// spinner ticks, mouse wheel, etc. → active tab (panel ephemera →
		// frame-cache nonce, same reasoning as keypresses)
		m.frameNonce++
		if cmd := m.tabs.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

// View builds the final frame for bubbletea v2 (alt-screen + mouse).
func (m Model) View() tea.View {
	// Boot splash owns the whole frame until done/skipped. Gated on View
	// (not Frame) so snapshot harnesses (cmd/uishot) keep printing the
	// office frame byte-identically — the splash never leaks into shots.
	if !m.bootDone && m.width > 0 {
		v := tea.NewView(m.boot.View())
		v.AltScreen = true
		v.ReportFocus = true // focus latch feeds the notify pings from tick one
		return v
	}
	v := tea.NewView(m.Frame())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true // terminal focus events drive the !focused ping gate
	return v
}

// projInfo — nil-safe feed for the top bar's project+branch segment.
func (m Model) projInfo() projinfo.Info {
	if m.proj == nil {
		return projinfo.Info{}
	}
	return m.proj.Get(m.sessDir)
}

// Frame renders the whole UI as one string — also what snapshot harnesses
// (cmd/uishot) print after the scripted run. Render cost is skipped when
// nothing changed: the digest cache returns the previous frame string
// verbatim, and the floor itself is memoized (office.CachedStyled — the
// same tick+sprites never rebuilds the grid).
func (m Model) Frame() string {
	if m.width == 0 {
		return "theboringoffice — waiting for terminal size…"
	}
	digest := m.frameDigest()
	if m.gov.frameCached != "" && m.gov.frameKey == digest {
		m.gov.frameHits++
		return m.gov.frameCached
	}
	m.gov.frameMisses++
	info := m.projInfo()
	top := chrome.TopBar(m.st, m.width, info)
	if m.compact() {
		top = chrome.TopBarCompact(m.st, m.width, info)
	}
	if m.bypassPerms {
		// the loud ⚠ BYPASS segment — always visible, every tab (the
		// topbar is shared): the office owns the badge, chrome owns the
		// bar's segment grammar, so the composed bar donates the cells.
		top = spliceBypassBadge(top, m.width)
	}
	var mid, bot string
	if m.zen {
		// zen (/zen · /focus floor) — transient fullscreen floor: sidebar
		// hidden entirely, topbar stays, chat gone, statusline minimal.
		mid = lipgloss.NewStyle().Width(m.width).Height(m.middleH).
			Render(office.CachedStyled(m.st, m.width, m.middleH))
		bot = chrome.StatusBarZen(m.st, m.width)
		if !m.quitArmAt.IsZero() {
			// the ctrl+q arm's toast is the zen bar's right segment too —
			// a double-press quit needs its visible affordance under /zen.
			bot = chrome.StatusBarZenHint(m.st, m.width,
				chrome.OnBarBold(chrome.Warn, " "+quitArmToast+" "))
		}
	} else if m.threadFocus != nil {
		// thread focus (ctrl+f) — the fullscreen nested thread panel owns
		// the whole middle region (zen's layout twin: sidebar hidden,
		// topbar stays), the zen statusbar seam carrying the "how to
		// leave" segment. zen's branch above still wins over it.
		mid = lipgloss.NewStyle().Width(m.width).Height(m.middleH).
			Render(m.threadFocus.View())
		bot = chrome.StatusBarZenHint(m.st, m.width,
			chrome.OnBar(chrome.Dim, " thread "+m.focusThread+" — esc · ctrl+f back to office "))
		if !m.quitArmAt.IsZero() {
			// the ctrl+q arm's toast outranks the focus chrome exactly
			// like it outranks zen's default segment.
			bot = chrome.StatusBarZenHint(m.st, m.width,
				chrome.OnBarBold(chrome.Warn, " "+quitArmToast+" "))
		}
	} else if m.mobile() {
		// mobile (auto, width < mobileMaxCols): the middle stacks
		// VERTICALLY — the left slot (floor|browser switcher + content)
		// as a compact band on top, the active panel full-width below it.
		// No horizontal split, no sidebar frame eating columns; the chrome
		// rows (topbar/statusbar) stay as-is.
		bandH := m.floorBandH()
		left := m.leftPaneView(m.width, bandH)
		var side string
		if m.planPaneVisible() {
			// a presented/edited plan swaps the PANEL slot (the big lower
			// region) for the plan editor; the floor band stays on top.
			// Plan mode with an EMPTY pane keeps the normal panel stack.
			m.plan.SetSize(m.width, m.middleH-bandH)
			side = m.plan.View()
		} else {
			side = lipgloss.NewStyle().Width(m.width).Height(m.middleH - bandH).
				Background(chrome.PanelBgColor).
				Render(m.tabs.View())
		}
		mid = lipgloss.JoinVertical(lipgloss.Left, left, side)
		bot = chrome.StatusBarAgent(m.st, m.hintLine(), len(m.queue), m.agentBadge(), m.width)
	} else {
		side := lipgloss.NewStyle().Width(m.sidebar).Height(m.middleH).
			Background(chrome.PanelBgColor).
			Render(m.tabs.View())
		if m.planPaneVisible() {
			// a presented/edited plan: the plan editor owns the floor slot.
			// Plan mode with an EMPTY pane leaves the normal office floor.
			m.plan.SetSize(m.floorW, m.middleH)
			mid = lipgloss.JoinHorizontal(lipgloss.Top, m.plan.View(), side)
		} else {
			mid = lipgloss.JoinHorizontal(lipgloss.Top, m.leftPaneView(m.floorW, m.middleH), side)
		}
		bot = chrome.StatusBarAgent(m.st, m.hintLine(), len(m.queue), m.agentBadge(), m.width)
	}
	frame := lipgloss.JoinVertical(lipgloss.Left, top, mid, bot)
	// The /model picker splices over the COMPOSED frame (app-level float —
	// centered on the whole terminal, layout-neutral: floor/sidebar/zen/
	// mobile all underlay the same). While a permission/question float is
	// up the picker hides and waits (a parked turn's modal outranks
	// browsing — it yields its keys in handleKey identically).
	if m.modelPick != nil && m.permQ.front() == nil && m.question == nil {
		frame = panels.ModelPickerFrame(m.modelPick, frame, m.width, m.height)
	}
	// the premium browser lane's frame-splice registry: published AFTER the
	// frame composed (the entry always matches what the renderer just got —
	// a cache-hit Frame republishes nothing because nothing it covers
	// changed). browser.go owns the origin math + the empty-state clears.
	m.publishZenbuFrame()
	// the chat previews' registry region rides the SAME seam (images.go's
	// publish — kitty previews splice through the wrapper, never the View
	// string, the wave-86 routing). ADDITIVE: the browser publish above is
	// untouched.
	m.publishChatMediaFrame()
	m.gov.frameKey, m.gov.frameCached = digest, frame
	return frame
}

// spliceBypassBadge — the /bypass topbar badge's app-side injection: the
// office owns the badge (chrome.TopBar's segment grammar is the chrome
// package's), so the COMPOSED bar donates the cells here. The standard
// bar's left↔right gap is the bar's only wide run of plain spaces —
// its first badgeWidth cells swap for the segment, width-exact, so the
// row never reflows. A bar with no such run (the compact layout, an
// extreme narrow terminal) drops trailing cells instead (ansi-aware
// truncate, then the badge re-fills to width) — the segment is ALWAYS
// visible: a mode that silences every permission prompt never gets to
// be subtle.
func spliceBypassBadge(top string, width int) string {
	seg := chrome.OnBarBold(chrome.Warn, bypassBadgeText)
	w := lipgloss.Width(seg)
	if w <= 0 || width <= w+2 {
		return top // no room at all — the bar's own truncate keeps the row
	}
	if i := strings.Index(top, strings.Repeat(" ", w)); i >= 0 {
		return top[:i] + seg + top[i+w:]
	}
	return ansi.Truncate(top, width-w, "") + seg
}

// hintLine — the statusbar's hint segment for THIS frame: the ctrl+q
// arm's HIGH-VISIBILITY toast while an arm is live (chrome's warn class
// on the bar background), the "Copied N chars" copy note while fresh
// (chrome's OK class, or warn for a verdicted copy FAILURE), the terminal
// hint while the shell tab is focused,
// else the static keymap line. PRECEDENCE: the quit-arm toast OUTRANKS
// the copy toast — safety first (the armed-quit affordance never hides
// behind a clipboard note; the note simply resumes once the arm retires,
// its own 2s window untouched). keys.HintLine is a free function
// with a frozen signature — the armed/terminal/copied swaps happen HERE,
// so the hint and the key handling still can't drift apart.
func (m Model) hintLine() string {
	if !m.quitArmAt.IsZero() {
		return chrome.OnBarBold(chrome.Warn, " "+quitArmToast+" ")
	}
	if !m.approveArmAt.IsZero() {
		// F1 — the ctrl+x approve arm rides the same warn seam, just
		// under the quit arm (quit-out-everything outranks approve).
		return chrome.OnBarBold(chrome.Warn, " "+approveArmToast+" ")
	}
	if m.btwHidden() {
		return chrome.OnBar(chrome.Dim, " btw hidden — click bubble or /btw to resume ")
	}
	if m.btwSaved != nil {
		return chrome.OnBarBold(chrome.OK, " btw — esc or /done to return ")
	}
	if m.copyNote != "" {
		if m.copyNoteBad {
			return chrome.OnBarBold(chrome.Warn, " "+m.copyNote+" ")
		}
		return chrome.OnBarBold(chrome.OK, " "+m.copyNote+" ")
	}
	if hasPendingBoss(m.st) && !m.questionParked && m.bossWedgeOverdue() {
		// W1 — the wedge hint derives from the silence CLOCK, not the
		// one-shot latch: it shows whenever a boss turn is overdue and
		// vanishes on its own the moment real traffic refreshes the
		// clock — no stale banner after a recovered turn, no chorus of
		// repeated notes either (the activity line fires once per turn).
		return chrome.OnBarBold(chrome.Warn, " "+wedgeHint+" ")
	}
	if m.terminalActive() {
		if m.termCapturedNow() {
			return termHintCaptured
		}
		return termHintReleased
	}
	if m.agentMode == agentModePlan {
		// conversation-first hint swap: no plan presented yet → the pane
		// is hidden and the member just talks; a presented/edited plan →
		// the click-to-edit surface hint (plan_mode.go — frozen copy).
		if m.planPaneVisible() {
			return planHintPane
		}
		return planHintIdle
	}
	return m.keys.HintLine()
}

// quitArmWindow — the ctrl+q double-press window (the chat panel's
// dblEscWindow pattern applied to the quit path): the FIRST press only
// arms — the hint bar swaps to the warn-class quitArmToast — and the
// second press inside the window quits via the existing persist + reap +
// tea.Quit path. A stale first press can't pair: it re-opens a fresh arm.
const quitArmWindow = 1500 * time.Millisecond

// pasteIgnoreNotice — the dim office row toasted when a paste has no
// focused text surface (never a silent drop). Frozen copy.
const pasteIgnoreNotice = "paste: nothing focused accepts text"

// routePaste — the ONE deliberate tea.PasteMsg path (Terminal.app's
// cmd+v, every terminal's bracketed paste): the paste lands on EXACTLY
// ONE surface, mirroring the KEY path's ownership so paste never
// disagrees with typing:
//
//  1. a FOCUSED plan editor (paste-over-selection lives inside the
//     pane — the pre-router behavior, pinned by
//     TestPlanPasteMsgRoutesToFocusedPane);
//  2. a CAPTURED terminal tab: the shell owns the keyboard while
//     captured (handleKey claims everything there, floats included),
//     so the paste writes to the PTY too — bracket-wrapped for
//     readline/zle/fish on the main screen, raw inside alt-screen apps
//     (panels/terminal.go);
//  3. the open question float's answer field (a parked turn owns the
//     chat's input; the panel batches the insert, multi-line kept);
//  4. the /model picker's search input — the picker wave's seam: any
//     picker implementing Paste(string) tea.Cmd takes the text; one
//     that hasn't trips the ignore notice instead of sinking the paste
//     into the disabled textarea;
//  5. the fullscreen thread-focus view owns its keys and has no text
//     surface — ignore with the notice;
//  6. the chat textarea — the chat tab, and ALSO the terminal tab
//     RELEASED (the office owns the keyboard there, so the draft takes
//     the text; the PTY never sees it);
//  7. otherwise (agents/board/mail/activity/git — no text surface):
//     ONE dim notice, never a silent drop.
func (m *Model) routePaste(msg tea.PasteMsg) tea.Cmd {
	if m.agentMode == agentModePlan && m.plan != nil && m.plan.Focused() &&
		m.permQ.front() == nil && m.question == nil && m.modelPick == nil {
		return m.plan.Update(msg)
	}
	if m.termCapturedNow() {
		return m.tabs.Update(msg)
	}
	if m.question != nil {
		return m.chat.Update(msg)
	}
	// the browser pane's inline URL editor owns paste while it is open —
	// the same ownership it has over keys.
	if m.browserActive() && m.browser != nil && m.browser.Editing() {
		return m.browser.Update(msg)
	}
	if m.modelPick != nil {
		if p, ok := any(m.modelPick).(interface{ Paste(string) tea.Cmd }); ok {
			return p.Paste(msg.Content)
		}
		m.notice(pasteIgnoreNotice)
		return nil
	}
	if m.threadFocus != nil {
		m.notice(pasteIgnoreNotice)
		return nil
	}
	if m.tabs.ActiveIndex() == 0 || m.terminalActive() {
		return m.chat.Update(msg)
	}
	m.notice(pasteIgnoreNotice)
	return nil
}

// handleKey implements the global keymap; unclaimed keys go to the tabs.
//
// The terminal tab's keyboard is OPT-IN (wave-42). RELEASED (the default)
// the office behaves NORMALLY on the terminal tab: tab/shift+tab cycle,
// 1..7 jump, ctrl+p toggles, q/ctrl+c quit, and typed letters/enter are
// consumed WITHOUT reaching the PTY or leaking to the chat. ctrl+space
// TOGGLES capture — ONE key, both ways, only while the terminal tab is
// active — then the terminal tab has the tightest claim (wave-41): the
// ONLY keys the app keeps are ctrl+space (toggle back out), ctrl+o
// (release alias back to the office keys) and ctrl+q (double-press to
// quit — claimed above). Every other key, tab, shift+tab, the digit jumps,
// q and ctrl+c INCLUDED, forwards to the REAL shell (term maps ctrl+c to
// 0x03 → SIGINT of the shell's foreground process, not an app quit; tab to
// 0x09 → the shell's completion; shift+tab to "\x1b[Z" → reverse
// completion).
func (m *Model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	chatActive := m.tabs.ActiveIndex() == 0
	termActive := m.terminalActive()
	// The left-pane browser, while the switcher sits on it, owns the
	// office's unclaimed keys: the right strip's chat/terminal surfaces
	// (the textarea, the shell) all yield — exactly like the browser tab
	// did when it rode the strip.
	browserActive := m.browserActive()
	if browserActive {
		chatActive, termActive = false, false
	}

	// ANY other key press clears a pending ctrl+q arm (its toast retires
	// on the next render via the keypress frameNonce bump).
	if key != "ctrl+q" && !m.quitArmAt.IsZero() {
		m.quitArmAt = time.Time{}
	}
	// Same for the ctrl+x approve arm (F1): any other key disarms it —
	// including edits that would change WHAT the fire would approve.
	if key != "ctrl+x" && !m.approveArmAt.IsZero() {
		m.approveArmAt = time.Time{}
	}

	switch {
	case key == "ctrl+q":
		// Double-press to quit (quitArmWindow), works EVERYWHERE — terminal
		// focus included: the first press arms + toasts the hint bar and
		// schedules its own expiry tick; the second press inside the window
		// runs the existing quit path untouched.
		now := time.Now()
		if !m.quitArmAt.IsZero() && now.Sub(m.quitArmAt) <= quitArmWindow {
			m.quitArmAt = time.Time{}
			m.persistOfficeSession(true) // final SYNC snapshot (live only)
			m.closeTerminal()
			m.closeBrowser()
			return tea.Quit
		}
		m.quitArmAt = now
		return tea.Tick(quitArmWindow, func(time.Time) tea.Msg { return armClearMsg{} })
	case m.zen:
		// any key exits zen (transient fullscreen floor); the key does
		// nothing else this press
		m.zen = false
		return nil
	}

	// The open /model picker owns EVERY key (question-modal contract — the
	// textarea is disabled; ↑/↓ walk its cursor, enter switches, esc
	// cancels, the card swallows the rest). It claims BEFORE tab switches
	// and the chat below, but AFTER ctrl+q/zen above, and it YIELDS while a
	// permission/question float is up (a parked turn outranks browsing —
	// the picker hides and waits, model_picker.go's contract).
	if m.modelPick != nil && m.permQ.front() == nil && m.question == nil {
		return m.modelPick.Key(msg)
	}

	// The open thread-focus view owns EVERY key (esc/ctrl+f leave, the
	// rest scroll or toggle INSIDE the pane). Claimed AFTER ctrl+q (the
	// quit arm outranks focus), AFTER zen (zen exits first press), and
	// YIELDING to a permission/question float — the model.go:1657
	// modelPicker pattern: a parked turn outranks the view.
	if m.threadFocus != nil && m.permQ.front() == nil && m.question == nil {
		return m.focusKey(msg)
	}

	// ESC while in a /btw side session hides it behind a pinned bubble.
	// Claimed after thread-focus (its esc wins while open)
	// and after model-picker, but before tab switches.
	if key == "esc" && m.btwSaved != nil && m.permQ.front() == nil && m.question == nil {
		return m.hideBtw()
	}

	// Tab-switch keys work on the terminal tab like ANY OTHER tab while the
	// shell keyboard is RELEASED (the default). In CAPTURED mode (opt-in
	// via ctrl+space) tab/shift+tab are the shell's completion keys and the
	// digit keys its ordinary input: they break out of this switch and fall
	// through to the shell forward below. The ctrl+space toggle and the
	// ctrl+o release alias are app-kept (never forwarded): the toggle flips
	// capture BOTH ways, the alias releases OUT only (never a dive).
	switch key {
	case "tab":
		if m.termCapturedNow() {
			break // captured: the shell's completion key — forwarded below
		}
		m.tabs.Next()
		m.maybeSpawnTerminal()
		return nil
	case "shift+tab":
		if m.termCapturedNow() {
			break // captured: the shell's reverse completion — forwarded below
		}
		m.tabs.Prev()
		m.maybeSpawnTerminal()
		return nil
	case "ctrl+b":
		// the LEFT pane's floor|browser switcher (the tab-cycle claim
		// tier): flips BOTH ways from every surface except a CAPTURED
		// terminal — there the shell owns 0x02 like every other byte.
		if m.termCapturedNow() {
			break // captured: ctrl+b is the shell's byte — forwarded below
		}
		m.toggleLeftTab()
		return nil
	case "ctrl+space":
		// THE one capture toggle: released ⇄ captured, BOTH ways, and only
		// while the terminal tab is active. History: this used to be
		// "ctrl+i" (dive-in only) — but ctrl+i is byte-identical to tab
		// (both emit 0x09; no distinguishable encoding on non-kitty
		// terminals), so the dive key smashed tab-to-leave. ctrl+space
		// emits 0x00 — safe, distinct, reversible.
		if termActive {
			m.setTermCaptured(!m.termCapturedNow())
			return nil
		}
	case "ctrl+o":
		// release OUT of shell capture (documented alias — never a dive):
		// stays on the tab, office keys live; inert while released.
		if m.termCapturedNow() {
			m.setTermCaptured(false)
			return nil
		}
	case "esc":
		// Escape is the quick exit from an actively captured terminal. Keep
		// it above the generic captured-key forward below so the shell never
		// receives its 0x1b byte.
		if m.termCapturedNow() {
			m.setTermCaptured(false)
			return nil
		}
	}
	if !chatActive && !m.termCapturedNow() {
		if idx := m.keys.TabJump(key); idx >= 0 {
			m.tabs.SetActive(idx)
			m.maybeSpawnTerminal()
			return nil
		}
	}

	if m.termCapturedNow() {
		// captured: everything else belongs to the shell — ctrl+p INCLUDED
		// (shell readline's previous-history key); the plan/build toggle
		// below never fires while the shell owns the keyboard.
		return m.tabs.Update(msg)
	}

	// ctrl+p — the plan/build mode toggle (toggle ONLY: chat keeps focus,
	// the pane does not open), claimable from every surface EXCEPT a
	// captured terminal (the shell owns it there; released = normal office).
	// The open /model picker already owns every key above (and yields to
	// floats, mirrored here): the permission/question/model floats keep
	// their keys — a parked turn outranks a mode switch.
	if key == "ctrl+p" && m.permQ.front() == nil && m.question == nil && m.modelPick == nil {
		return m.togglePlanMode()
	}

	// ctrl+x — F1: approve-arm double press, claimable from BOTH the chat
	// input and the plan editor focus while plan mode is active. Same
	// exclusion list as ctrl+p (floats; terminal focus returns above),
	// and claimed BEFORE the pane's key routing below and BEFORE the chat
	// textarea. The FIRST press arms (warn toast on the hint bar seam,
	// self-expiring tick); the second press inside approveArmWindow fires
	// approvePlan. REFUSALS land BEFORE the arm (approveRefusal: empty /
	// starter / restored-untouched) so the toast never promises a fire
	// that can't happen.
	if key == "ctrl+x" && m.agentMode == agentModePlan &&
		m.permQ.front() == nil && m.question == nil && m.modelPick == nil {
		// A live editor selection turns ctrl+x into CUT (the textarea
		// classic) — the approve arm's keyguard pauses while text is
		// marked, and a cut retires even a stale arm outright (a mouse
		// drag can arm a mark without any keypress clearing the arm).
		if m.plan.Focused() && m.plan.SelectionActive() {
			m.approveArmAt = time.Time{}
			return m.planCopySelectionCmd(true)
		}
		if refused := m.approveRefusal(); refused != "" {
			m.approveArmAt = time.Time{}
			m.notice(refused)
			return nil
		}
		now := time.Now()
		if !m.approveArmAt.IsZero() && now.Sub(m.approveArmAt) <= approveArmWindow {
			m.approveArmAt = time.Time{}
			return m.approvePlan()
		}
		m.approveArmAt = now
		return tea.Tick(approveArmWindow, func(time.Time) tea.Msg { return approveArmClearMsg{} })
	}

	// The plan editor, while focused, owns every remaining key — the
	// terminal tab's tight claim, mirrored for the plan surface. The
	// excepted keys ride the claims AROUND this block: ctrl+q (the quit
	// arm) and the tab switches above; ctrl+x (approve→build) via its own
	// global claim right above; ctrl+c (quit) via its fall-through case
	// inside; esc (blur back to the chat input) handled here. While a
	// perm/question float is up the pane yields entirely — the float's
	// chat-modal keys keep working (the model-picker contract, one level
	// down).
	if m.agentMode == agentModePlan && m.plan != nil && m.plan.Focused() &&
		m.permQ.front() == nil && m.question == nil {
		switch key {
		case "esc":
			// the selection owns esc first (webpage rule): clear the mark,
			// keep the pane focused; a bare esc blurs back to the chat
			// input (the plan buffer keeps; the editor just blurs).
			if m.plan.SelectionActive() {
				m.plan.ClearSelection()
				return nil
			}
			m.plan.Blur()
			return nil
		case "ctrl+c":
			// a live selection turns ctrl+c into COPY (the textarea
			// classic — the quit path below keeps its claim only when no
			// text is marked); a copy NEVER arms the approve pair and
			// retires a stale one outright.
			if m.plan.SelectionActive() {
				m.approveArmAt = time.Time{}
				return m.planCopySelectionCmd(false)
			}
		default:
			return m.plan.Update(msg)
		}
	}

	switch key {
	case "ctrl+c":
		m.persistOfficeSession(true) // final SYNC snapshot (live only)
		m.closeTerminal()
		m.closeBrowser()
		return tea.Quit
	case "q":
		// The browser is the ONE non-chat surface where q does NOT quit:
		// there it means "leave the tab" (the pane's own q/esc →
		// BrowserLeaveMsg → the floor tab). Every other non-chat tab
		// keeps the global quit. (Grab/quit tests pin the non-browser
		// branches.)
		if !chatActive && !browserActive {
			m.persistOfficeSession(true) // final SYNC snapshot (live only)
			m.closeTerminal()
			m.closeBrowser()
			return tea.Quit
		}
	case "ctrl+t":
		if chatActive {
			m.chat.ToggleThink()
			return nil
		}
	case "ctrl+d":
		if chatActive {
			m.chat.ToggleDiffs()
			return nil
		}
	case "ctrl+g":
		if chatActive {
			m.chat.ToggleThreads()
			return nil
		}
	case "ctrl+f":
		if chatActive {
			return m.toggleThreadFocus()
		}
	case "o":
		// open-in-browser (links.go owns the panel half): `o` claims ONLY
		// while (a) the chat tab is focused, (b) a transcript mark is live
		// over a bubble carrying ≥1 VERIFIED target, and (c) no floating
		// modal is up (permission/question/model picker — a parked turn
		// outranks browsing; the /model picker already claimed every key
		// above). While the target card itself is open IT owns the keys
		// (its claim sits inside the chat panel): re-opening must not
		// reset a browsed cursor. ONE target fires straight through the
		// verdict seam; MULTIPLE float the card. With no mark or no
		// verified target the key falls through untouched and types "o"
		// into the draft — the claim rule keeps plain typing safe.
		if chatActive && m.chat != nil && !m.chat.LinkPickerOpen() &&
			m.permQ.front() == nil && m.question == nil && m.modelPick == nil {
			targets := m.chat.OpenTargets()
			switch {
			case len(targets) > 1:
				m.chat.OpenLinkPicker(targets)
				return nil
			case len(targets) == 1:
				return m.openTargetCmd(targets[0])
			}
		}
	}
	if termActive {
		// Terminal active but RELEASED: every unclaimed key (typed letters,
		// enter, esc, arrows) is consumed — never the PTY, never the chat
		// textarea. Only a dead shell still receives input: "r" respawns.
		if m.termTab != nil && !m.termTab.alive() {
			return m.tabs.Update(msg)
		}
		return nil
	}
	if browserActive {
		// the left-pane browser owns every unclaimed key (scroll, link
		// cursor, [ ] history, r reload, o open, q/esc leave) — the right
		// strip's active tab (the chat textarea INCLUDED) never sees them.
		return m.browser.Update(msg)
	}
	cmd := m.tabs.Update(msg)
	// TOP-GESTURE: an ↑/pgup that leaves the transcript on row 0 arms ONE
	// older-history hop (maybeArmOlder owns every guard: the arm fires
	// only when AtTranscriptTop still reads true after the panel settled).
	if chatActive && (key == "up" || key == "pgup") {
		if arm := m.maybeArmOlder(); arm != nil {
			return tea.Batch(cmd, arm)
		}
	}
	return cmd
}

// openTargetCmd — the open-in-browser exec rides a tea.Cmd (the UI
// goroutine never shell-outs mid-frame; the clipboardResultMsg pattern).
// The VERDICT lands back as browserOpenMsg: success logs the activity
// tab's "→ opened: <name>", failure posts a dim "could not open:" row.
func (m *Model) openTargetCmd(t panels.LinkTarget) tea.Cmd {
	return func() tea.Msg {
		return browserOpenMsg{target: t, err: panels.OpenInBrowser(t)}
	}
}

// planCopySelectionCmd — the focused plan pane's copy/cut verdict effect
// (the ctrl+c/ctrl+x selection claims above ride it): the panel's stubbed
// clipboardCopyText seam already ran inside Copy/CutSelection; this wraps
// the OSC52 fallback + the toast around the shared verdict.
func (m *Model) planCopySelectionCmd(cut bool) tea.Cmd {
	var (
		text string
		n    int
		err  error
	)
	if cut {
		text, n, err = m.plan.CutSelection()
	} else {
		text, n, err = m.plan.CopySelection()
	}
	return m.planCopyVerdictCmd(text, n, err)
}

// planCopyVerdictCmd — the copy verdict for EVERY plan-pane copy path
// (ctrl+c, ctrl+x-cut, the mouse-drag release): success toasts the frozen
// "Copied N chars" on the statusbar seam (its own tick retires it) with
// tea.SetClipboard riding as the ssh fallback; a missing platform tool
// degrades to ONE dim note with the OSC52 channel still warm; an empty
// mark decides nothing (no toast, like the transcript).
func (m *Model) planCopyVerdictCmd(text string, n int, err error) tea.Cmd {
	if err != nil {
		m.notice("no os clipboard tool — copy rode the terminal escape (" + err.Error() + ")")
		return tea.SetClipboard(text)
	}
	if n == 0 {
		return nil
	}
	return tea.Batch(tea.SetClipboard(text), m.armCopyNote(n))
}

// planBodyCoords translates a screen point into the plan pane's BODY-space
// (col 0 at the pane's left edge, vrow 0 at the body's first row): the
// topbar row and the pane's own header row are subtracted; desktop measures
// against the floor slot, mobile against the panel slot under the band.
// Values may leave the body (header/footer/overhang) — the panel clamps.
func (m *Model) planBodyCoords(x, y int) (col, vrow int) {
	if m.mobile() {
		return x, y - (1 + m.floorBandH() + 1)
	}
	return x, y - 2
}

// planPaneRegionHit reports whether a screen point is INSIDE the plan
// pane's body rectangle (a press gate — motions/releases clamp instead):
// plan mode + a visible pane, the pane's own column lanes, body rows only.
// Floats keep their click claims (a parked turn outranks an editor press).
func (m *Model) planPaneRegionHit(x, y int) bool {
	if m.agentMode != agentModePlan || m.plan == nil || !m.planPaneVisible() ||
		m.permQ.front() != nil || m.question != nil || m.modelPick != nil ||
		m.threadFocus != nil || m.zen || m.height == 0 {
		return false
	}
	col, vrow := m.planBodyCoords(x, y)
	bw, paneH := m.floorW, m.middleH
	if m.mobile() {
		bw, paneH = m.width, m.middleH-m.floorBandH()
	}
	if col < 0 || col >= bw {
		return false
	}
	bodyH := paneH - 2 // the pane reserves its header + footer rows
	if bodyH < 1 {
		bodyH = 1
	}
	return vrow >= 0 && vrow < bodyH
}

// focusKey — the open thread-focus view owns every key: esc/ctrl+f leave
// (the office state underneath was never touched — scroll offsets, thread
// expansion and the draft all survived behind the pane), everything else
// forwards into the pane (its scrolling, its ↳ diff toggles). ctrl+q and
// zen never reach here (claimed above, like the model picker's gate), and
// neither float lets the view keep keys while a turn is parked.
func (m *Model) focusKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+f":
		m.closeThreadFocus()
		return nil
	}
	return m.threadFocus.Update(msg)
}

// toggleThreadFocus — ctrl+f OPEN: resolve the thread (the panel's
// ResolveFocusThread chain: expand-ledger tail → any live thread → the
// timeline's most recent → a hired worker with no recorded lines), mount
// the fullscreen pane at the LIVE state, and arm the main chat's render
// saver. No candidate → a dim office notice, no claim.
func (m *Model) toggleThreadFocus() tea.Cmd {
	name, ok := panels.ResolveFocusThread(m.chat, m.st)
	if !ok {
		m.notice("no worker threads yet")
		return nil
	}
	m.mountThreadFocus(name)
	return nil
}

// mountThreadFocus — the OPEN half of any focus enter (ctrl+f's resolved
// winner, or a thread-row click naming its agent directly): the fullscreen
// nested pane mounts at the LIVE state and the main chat's render saver
// arms (ResumeFromFocus catches the office up in ONE pass at close).
func (m *Model) mountThreadFocus(name string) {
	m.threadFocus = panels.NewThreadFocus(name, m.width, m.middleH)
	m.threadFocus.SetState(m.st)
	m.focusThread = name
	m.focusDeferredRender = true
	if m.chat != nil {
		m.chat.SetDeferredRender(true)
	}
}

// closeThreadFocus — esc/ctrl+f out (and the float-dismount path): the
// pane unmounts, the render saver clears, and ONE ResumeFromFocus
// re-render lands the main chat exactly at the snapshot the last pulse
// recorded — the return is byte-identical to the state the focus covered.
func (m *Model) closeThreadFocus() {
	if m.threadFocus == nil {
		return
	}
	m.threadFocus = nil
	m.focusThread = ""
	m.focusDeferredRender = false
	if m.chat != nil {
		m.chat.ResumeFromFocus()
	}
}

// dismountThreadFocus — a permission/question event armed a float the
// focus must NOT cover: the pane unmounts mid-keystroke and a one-line
// dim notice records the swap (the float keeps the pixels).
func (m *Model) dismountThreadFocus(why string) {
	if m.threadFocus == nil {
		return
	}
	name := m.focusThread
	m.closeThreadFocus()
	m.notice("thread focus — " + name + " closed (" + why + ")")
}

// clickDblWindow — two floor clicks on the SAME sprite inside this window
// count as a double-click (thread toggle), not two selections.
const clickDblWindow = 400 * time.Millisecond

// handleClick routes bubbletea v2 mouse click PRESSES — reached DIRECTLY
// for off-transcript presses (handlePress falls through) or REPLAYED on a
// motionless release after a transcript press (the selection seam pins
// its fate until release, so single-click semantics survive verbatim;
// the view's MouseModeCellMotion emits press / release / drag-motion,
// which is what the drag needs). FLOOR: a click on an employee's 3-cell
// sprite (office.HitAgent) selects the agent — activity tab opens, agents
// tab pins a ▸ marker on its row, and an office notice names it; a second
// click on the SAME sprite inside clickDblWindow instead toggles that
// agent's work thread in chat (and jumps there). CHAT (sidebar, chat tab):
// a click inside the open permission popover's card answers it on the spot
// (PermClick owns the card's fixed hit-map — same response strings the
// y/a/n keys send); a click on a worker thread's frame rows (the header,
// a collapsed thread's ↳ sneak, an expanded thread's closing summary)
// CLOSES the transcript view behind the thread-focus pane and opens THAT
// agent's own transcript nested inside it (esc/ctrl+f back out returns
// byte-identical). Every other row keeps its legacy seam (↳ diff sub-rows,
// user fold rows — the inline-expansion toggle lives on ctrl+g and the
// floor double-click below). Clicks landing in the 2-cell frame chrome
// (topbar row / statusbar row) are ignored outright.
func (m *Model) handleClick(msg tea.MouseClickMsg) tea.Cmd {
	// an open thread-focus renders clicks inert app-side (mirroring the
	// zen gate + the /model picker swallow): handlePress already routed
	// the pane's own ↳ diff toggle; nothing else may leak through here.
	if m.height == 0 || m.zen || m.threadFocus != nil || msg.Button != tea.MouseLeft {
		return nil
	}
	// the 2-cell chrome (topbar + statusbar) never reacts
	if msg.Y <= 0 || msg.Y >= m.height-1 {
		return nil
	}
	// The /model picker is keys-only (the /session picker's click-swallow
	// contract one level up): while its card is up a click lands on NOTHING
	// underneath — no floor selection, no thread toggle, no popover answer.
	if m.modelPick != nil {
		return nil
	}
	// Plan mode's floor-slot pane owns its region: a click INSIDE it opens
	// the pane for editing — an EMPTY pane first arms its starter scaffold
	// (the manual-open scratch workflow), a presented plan is never
	// clobbered — and the click is swallowed (no sprite hit-testing;
	// textarea cursor placement is out of v1). A click anywhere else
	// (chat region, floor band on mobile) hands typing back to the chat
	// input — the editor blurs and the permanently-focused chat textarea
	// simply resumes the keys on the existing routing.
	if m.agentMode == agentModePlan && m.plan != nil {
		if m.mobile() {
			if msg.Y >= 1+m.floorBandH() {
				m.openPlanForEdit()
				return nil
			}
		} else if msg.X < m.floorW {
			m.openPlanForEdit()
			return nil
		}
		m.plan.Blur()
	}
	if m.mobile() {
		// mobile stack: the floor band owns rows 1..floorBandH (hit-tested
		// by the shared floor tail below); BELOW the band sits the panel,
		// whose chat tab claims clicks exactly like the sidebar does —
		// content coords just shift down past the band.
		if msg.Y >= 1+m.floorBandH() {
			// git tab claims clicks in its panel region (same adjusted
			// coords contract as the sidebar branch).
			if m.tabs.ActiveIndex() == gitIndex {
				adj := msg
				adj.Y -= 1 + m.floorBandH()
				return m.tabs.Update(adj)
			}
			if m.tabs.ActiveIndex() != 0 || m.chat == nil {
				return nil
			}
			dx, dy := m.tabs.ContentOffset()
			cx, cy := msg.X-dx, msg.Y-(1+m.floorBandH()+dy)
			if cmd := m.chat.PermClick(cx, cy); cmd != nil {
				return cmd
			}
			if m.btwHidden() && m.chat.BtwPinRowAt(cx, cy) {
				return m.resumeBtw()
			}
			// a thread's frame rows open the nested thread-focus pane for
			// THAT agent (the clicked thread, not ctrl+f's resolved winner)
			if name, ok := m.chat.ThreadRowAt(cx, cy); ok {
				m.mountThreadFocus(name)
				return nil
			}
			m.chat.ClickRow(cx, cy)
			return nil
		}
	} else if msg.X >= m.floorW {
		// sidebar. The git tab claims clicks: row hit → open the file's
		// diff. Coords land in sidebar-box space (topbar stripped only) —
		// the panel subtracts Tabs.ContentOffset itself.
		if m.tabs.ActiveIndex() == gitIndex {
			adj := msg
			adj.X -= m.floorW
			adj.Y--
			return m.tabs.Update(adj)
		}
		// otherwise only the chat tab claims clicks (the permission
		// popover's card, then worker thread header rows). Content
		// coords = screen minus the box chrome (floor border col +
		// sidebar top row + tab bar & border — Tabs.ContentOffset).
		if m.tabs.ActiveIndex() != 0 || m.chat == nil {
			return nil
		}
		dx, dy := m.tabs.ContentOffset()
		cx, cy := msg.X-(m.floorW+dx), msg.Y-(1+dy)
		// the popover claims clicks inside its card first (fires the
		// answer seam); outside it returns nil and thread rows take over
		if cmd := m.chat.PermClick(cx, cy); cmd != nil {
			return cmd
		}
		if m.btwHidden() && m.chat.BtwPinRowAt(cx, cy) {
			return m.resumeBtw()
		}
		// a thread's frame rows open the nested thread-focus pane for
		// THAT agent (the clicked thread, not ctrl+f's resolved winner)
		if name, ok := m.chat.ThreadRowAt(cx, cy); ok {
			m.mountThreadFocus(name)
			return nil
		}
		m.chat.ClickRow(cx, cy)
		return nil
	}
	// The left-pane browser owns the floor slot's clicks while the
	// switcher sits on it: no sprite hit-testing underneath (link-click
	// navigation is wave-out — the pane's keys carry it).
	if m.browserActive() {
		return nil
	}
	id, ok := office.HitAgent(m.st, msg.X, msg.Y-1 /* topbar row */)
	if !ok {
		return nil
	}
	name := id
	if e := findEmployee(m.st, id); e != nil {
		name = e.Name
	}
	now := time.Now()
	double := m.lastClickAgent == id && now.Sub(m.lastClickAt) <= clickDblWindow
	m.lastClickAgent, m.lastClickAt = id, now
	if double {
		m.lastClickAgent = "" // a third click starts a fresh pair
		if m.chat != nil {
			m.chat.ToggleThread(name)
			m.tabs.SetActive(0) // show the thread it just toggled
		}
		return nil
	}
	if m.agents != nil {
		m.agents.SetSelected(name)
	}
	m.tabs.SetActive(5) // the activity tab shows the agent's work log
	m.notice(name + " selected")
	return nil
}

// terminalActive reports whether the focused tab is the OS-shell tab.
func (m *Model) terminalActive() bool {
	return m.tabs.ActiveIndex() == terminalIndex
}

// termCapturedNow — the EFFECTIVE capture state for routing: capture only
// counts while the terminal tab is the active one (a stale flag off-tab is
// released by normalizeTermCapture on the next routed message anyway, but
// opinions never read it — the guard keeps every consumer honest by
// construction).
func (m *Model) termCapturedNow() bool {
	return m.termCaptured && m.tabs.ActiveIndex() == terminalIndex
}

// normalizeTermCapture — the auto-release invariant (wave-42): shell
// capture can never escape its tab. Runs at Update entry (any routed tab
// change — click, event-driven SetActive, spawn-failure fallback) and in
// SelectTab, so a stale capture is dropped before the next key routes.
// Every terminal visit starts RELEASED: the opt-in is explicit per visit,
// never a memory of a prior capture.
func (m *Model) normalizeTermCapture() {
	if m.termCaptured && m.tabs.ActiveIndex() != terminalIndex {
		m.setTermCaptured(false)
	}
}

// setTermCaptured flips the capture flag and mirrors it into the terminal
// wrap (which gates key forwarding and syncs the inner panel's badge).
func (m *Model) setTermCaptured(on bool) {
	m.termCaptured = on
	if m.termTab != nil {
		m.termTab.setCaptured(on)
	}
}

// maybeSpawnTerminal lazy-spawns the terminal tab's shell on the first
// visit (battery: no PTY until the member asks for one). A spawn failure is
// a soft landing: office notice "terminal spawn failed: <err>" + fallback
// to the chat tab — never a crash.
func (m *Model) maybeSpawnTerminal() {
	if !m.terminalActive() || m.termTab == nil {
		return
	}
	if err := m.termTab.ensure(); err != nil {
		m.playSound("error")
		m.noticeErr(fmt.Sprintf("terminal spawn failed: %v", err))
		m.tabs.SetActive(0)
	}
}

// closeTerminal kills the spawned shell on the app quit path (Close is
// idempotent; a never-visited tab has no PTY to kill).
func (m *Model) closeTerminal() {
	if m.termTab != nil {
		m.termTab.close()
	}
}

// CloseTerminal is the exported quit-path hook for cmd/theboringoffice (the runtime
// intercepts tea.QuitMsg before Update, so an external p.Quit skips
// handleKey — call CloseTerminal alongside to never leak a shell process).
func (m *Model) CloseTerminal() { m.closeTerminal() }

// ExecRequest — the /session picker accept's exec-replace intent: the
// session id cmd/theboringoffice's post-Run path relaunches the binary with
// (`theboringoffice -s <id>`). "" when the picker never accepted this run
// (every other quit way leaves it empty).
func (m Model) ExecRequest() string { return m.execSession }

// LayoutInfo reports the computed frame geometry (uisshot --layout asserts).
func (m Model) LayoutInfo() (width, height, sidebar, floor int) {
	return m.width, m.height, m.sidebar, m.floorW
}

// applyEvent reduces one backend event, feeds panels + activity log, and
// re-arms the animation tick. Returns the next cmd when needed.
// applyEvent routes ONE backend (or synthetic) event through the pager
// kick + the media probe + the core handler. The kick is the older-
// history walk's whole attach surface: it rebinds/seeds the primary-
// session pager on the first event, re-arms it on a stream reconnect,
// and no-ops O(1) on every other event (the walk itself is gesture-
// driven, see maybeArmOlder). The media probe (images.go) buffers
// inbound boss-turn image payloads and fires the lazy rasterize cmd —
// model-owned UI state, exactly like the permission/question holds.
func (m *Model) applyEvent(ev state.Event) tea.Cmd {
	return tea.Batch(m.pagerKick(ev), m.applyMedia(ev), m.applyEventCore(ev), m.applyBrowserOpen(ev), m.applyRecentMessages(ev), m.applyPlanTools(ev), m.bypassLatchKick(ev))
}

// bypassLatchKick remains in the event batch for compatibility with backends
// that emit status during Start. Lifecycle completion is no longer inferred
// from those lines: Start's explicit result is the only authority, so a
// missing or reordered boot marker cannot wedge the restart latch.
func (m *Model) bypassLatchKick(ev state.Event) tea.Cmd {
	return nil
}

func (m *Model) applyEventCore(ev state.Event) tea.Cmd {
	// an allowed agent browser-open flips the left slot to the browser so
	// the member sees the page land (refused opens surface as a notice row
	// via applyBrowserOpen and never touch the slot).
	if ev.Kind == state.EvBrowserOpen && ev.BrowserOpenAllowed {
		m.leftTab = leftTabBrowser
	}
	// permission prompts + question holds are model-owned UI state (not
	// chat history) — handle before the reducer (the reducer also uses
	// the parked state: a question popover drops the typing placeholder).
	// permCmd carries the /bypass arm's auto-answer (the only cmd the
	// permission path ever mints); EvPermission events reduce to NOTHING
	// chat-owned (the asks are model-owned UI state), so the only return
	// such an event can reach is the final one below — the merge lives
	// there.
	var permCmd tea.Cmd
	if ev.Kind == state.EvPermission {
		permCmd = m.handlePermissionEvent(ev)
	}
	if ev.Kind == state.EvQuestion {
		m.handleQuestionEvent(ev)
	}
	// a permission/question float just armed: the thread-focus view must
	// not cover it — dismount mid-keystroke (the float keeps the pixels),
	// leaving a one-line dim notice.
	if (ev.Kind == state.EvPermission || ev.Kind == state.EvQuestion) &&
		(m.permQ.front() != nil || m.question != nil) && m.threadFocus != nil {
		m.dismountThreadFocus("a permission/question needs your answer")
	}

	// Think-stream bookkeeping (model-owned; the reducer stays pure):
	// open a CallID's stream on EvThought Done=false, close it on
	// Done=true. Defensive: ANY EvChatBoss — a fresh pending placeholder
	// included — downgrades every still-open stream to collapsed.
	switch ev.Kind {
	case state.EvThought:
		if ev.EmployeeID == "boss" && ev.CallID != "" {
			if ev.Done {
				delete(m.activeThink, ev.CallID)
			} else {
				m.activeThink[ev.CallID] = true
			}
		}
	case state.EvChatBoss:
		for id := range m.activeThink {
			delete(m.activeThink, id)
		}
	}

	// The backend echoes the composed batch prompt verbatim as chat-user;
	// the member sees ONE compact composite bubble instead of the raw
	// dispatch text ("you › 3 items: fix the badge; ship v2; …").
	if ev.Kind == state.EvChatUser && strings.HasPrefix(ev.Msg.Text, batchMarker) &&
		len(m.batchSummaries) > 0 {
		titles := make([]string, len(m.batchSummaries))
		for i, t := range m.batchSummaries {
			titles[i] = clipRunes(t, batchSummaryClip)
		}
		ev.Msg.Text = fmt.Sprintf("%d items: %s", len(titles), strings.Join(titles, "; "))
	}

	// session.error on the primary within the window of a batch send = the
	// boss died mid-batch: arm the ONE respawn (consumed at the completion
	// transition below, where a naive close would otherwise fire).
	respawn := ev.Kind == state.EvChatBoss &&
		strings.HasPrefix(ev.Msg.ID, "boss-error-") &&
		m.batchInFlight && !m.batchRespawned &&
		!m.batchSentAt.IsZero() && time.Since(m.batchSentAt) <= batchRespawnWindow

	// social-clock busy gate: remember the tick of the latest dispatch —
	// a dispatch younger than 30 ticks silences the clock (busy !== social).
	if ev.Kind == state.EvDispatch {
		m.lastDispatchTick = m.st.Tick
	}

	// sound hooks (no-op until a bus is injected): reply/dispatch/done/
	// alert/error at their reducer points.
	switch ev.Kind {
	case state.EvChatBoss:
		if !ev.Msg.Pending {
			if strings.HasPrefix(ev.Msg.ID, "boss-error-") {
				m.playSound("error") // session-level failure
			} else {
				m.playSound("reply")
			}
		}
	case state.EvReturned:
		m.playSound("done")
	case state.EvDispatch:
		m.playSound("dispatch")
	case state.EvBlocked:
		m.playSound("alert")
	}

	// notify: the done ping — an armed send's FIRST completed boss turn.
	// boss-error bubbles disarm silently (the error already owns the
	// transcript); a completion that lands while questionParked counts as
	// NOT-done (the member just engaged through the question modal — no
	// ping). Either way the arm is consumed exactly once per send; a later
	// respawn re-arms through its own chatSentMsg.
	if ev.Kind == state.EvChatBoss && !ev.Msg.Pending && m.notifyDoneArmed {
		m.notifyDoneArmed = false
		if !strings.HasPrefix(ev.Msg.ID, "boss-error-") && !m.questionParked {
			m.fireNotification("done", doneNotifyBody(ev.Msg.Text))
		}
	}

	prevPending := hasPendingBoss(m.st)
	m.st = reducer(m.st, ev)
	m.pruneRecentToolOutputs()
	m.applyDelegation(ev) // P3 — before panels see the state
	if ev.Kind != state.EvTick {
		m.updateIdleWrap()
	}
	// tool-output capture (EvTool.ToolOutput → the transcript's
	// click-to-expand body): the reducer's ChatMsg has no output field,
	// so the panel's expansion map rides this side feed, keyed by the
	// SAME entry id the merge uses (toolEntryID). Empty outputs never
	// reach the panel (a running call / an output-less tool renders the
	// pinned "no output as such" empty state there).
	if ev.Kind == state.EvTool && ev.ToolOutput != "" && m.chat != nil {
		m.chat.SetToolOutput(toolEntryID(ev.EmployeeName, ev.CallID), ev.ToolOutput)
	}
	if ev.Kind == state.EvTool && ev.ToolOutput != "" {
		if m.recentToolOutputs == nil {
			m.recentToolOutputs = map[string]string{}
		}
		m.recentToolOutputs[toolEntryID(ev.EmployeeName, ev.CallID)] = ev.ToolOutput
	}
	if m.chat != nil {
		m.chat.SetStreamingThink(m.activeThink)
	}
	m.tabs.SetState(m.st)
	if m.threadFocus != nil {
		// the focus's live pulse: every event (ticks included) re-filters
		// the focused agent's slice into the pane — tens of lines,
		// wtool/wthink/wdiff only, the clone's own rev gate keeps it cheap.
		m.threadFocus.SetState(m.st)
	}

	// F5a — the backend's agent-degrade latch note is statusline-only by
	// nature, and the NEXT EvStatus overwrites it; statuses carrying the
	// agentFieldStatusMarker escalade into the transcript as a red office
	// row (the marker is the string contract with opencode.go's postPrompt).
	// W4 extends the same seam to the serveDiedStatusMarker (opencode.go's
	// watchServe): a dead serve must never be a blink-and-miss-it line.
	if ev.Kind == state.EvStatus && (strings.HasPrefix(ev.Text, agentFieldStatusMarker) ||
		strings.HasPrefix(ev.Text, serveDiedStatusMarker)) {
		m.noticeErr(ev.Text)
	}

	// memory probe latch: the live backend's boot line announces the board
	// lane ("[theboringfloor] live - … | board: agentmemory (<winner>)" when
	// the agentmemory probe is hot, "| board: in-memory …" when offline) —
	// the /memory header reads this unless the additive AgentmemoryOK seam
	// exists (degrade-open file-only otherwise).
	if ev.Kind == state.EvStatus {
		if strings.Contains(ev.Text, "| board: agentmemory (") {
			m.agentmemoryOK = true
		} else if strings.Contains(ev.Text, "| board: in-memory") {
			m.agentmemoryOK = false
		}
	}

	// Plan-mode presentation (plan_mode.go): a COMPLETED boss bubble
	// while plan mode is active mirrors its markdown into the plan pane —
	// passive, chat keeps focus; a user-edited buffer is never clobbered
	// (the anti-clobber notice rides the office channel instead). F4's
	// completion hook runs first: a plan-tagged send whose turn ended
	// after a mid-turn flip BACK to build leaves its reply chat-only —
	// the transcript note says so once (the pane never opens then).
	if ev.Kind == state.EvChatBoss && !ev.Msg.Pending {
		m.notePlanCompletion(ev.Msg)
		m.presentBossPlan(ev.Msg)
	}

	// activity: mid-stream deltas are visual growth of ONE bubble — logging
	// each would spam the log (the placeholder's "typing…" and the final's
	// "reply" already bracket the turn). Skip pending-with-text events.
	isStreamDelta := ev.Kind == state.EvChatBoss && ev.Msg.Pending && ev.Msg.Text != ""
	// concierge pending deltas are one bubble's growth — the activity tab
	// gets ONE line per completed office answer, not per stream beat.
	isOfficeDelta := ev.Kind == state.EvChatOffice && ev.Msg.Pending
	if ev.Kind != state.EvTick && !isStreamDelta && !isOfficeDelta {
		m.activity.Add(m.describeEvent(ev))
		m.activityAdds++
		// [memory] fabric: a completed dispatch ALSO stamps the record's
		// landing into the log (the ledger-core dev owns the write; this is
		// the member-visible proof that it went down — same stamp seam as
		// describeEvent).
		if line, ok := m.describeMemoryRecorded(ev); ok {
			m.activity.Add(line)
			m.activityAdds++
		}
	}

	if ev.Kind == state.EvTick {
		// office-session cheap-write loop: at most one ASYNC snapshot write
		// per 5s window (sessions.go — no-op in demo mode).
		m.persistOfficeSession(false)
		// social clock: plans + fires its beats off the tick (EvBubble/
		// EvIdleDrift events through the normal reducer path — ambient.go).
		m.runSocial()
		// W1 wedge watchdog: notice-only, wall clock, one-shot per wedge
		// (checkBossWedge) — rides this same cheap loop since the tick is
		// the only event guaranteed to keep arriving during dead silence.
		m.checkBossWedge()
		wrap := m.checkIdleWrap()
		tick := m.tickCmd()
		if wrap == nil {
			return tick
		}
		return tea.Batch(wrap, tick)
	}
	// A completed boss bubble unblocks a parked question turn: the hold
	// resolved, the server resumed — the chat goes back to "typing" and
	// queued messages flush again.
	if m.questionParked && ev.Kind == state.EvChatBoss && !ev.Msg.Pending {
		m.questionParked = false
		if m.st.StatusLine == "[question] boss is waiting for your answer…" {
			m.st.StatusLine = m.parkedStatus
		}
		if m.chat != nil {
			m.chat.SetQuestionWaiting(false)
		}
		qdebugf("question resolved: completed boss reply unblocks queue")
		if len(m.queue) > 0 {
			return m.flushQueued()
		}
	}
	if !prevPending && hasPendingBoss(m.st) && m.chat != nil {
		return m.chat.SpinnerKick()
	}
	// While a question hold is outstanding the turn is PARKED at the
	// question reply API — not completed — so the queue must NOT flush.
	if prevPending && !hasPendingBoss(m.st) && !m.questionParked {
		// the boss reply landed (or errored out) — turn completed; close
		// the free-queuing tally with it
		m.resetServerTurn()
		if respawn {
			// session.error inside the window: fresh session + the SAME
			// batch, once. The rows stay open for the retry's turn.
			m.batchRespawned = true
			m.batchSentAt = time.Now()
			m.respawns++
			items := append([]queueEntry(nil), m.batchItems...)
			m.notice("boss went down — respawned a fresh session, resending batch")
			qdebugf("session.error inside batch window — respawning (respawn #%d, items=%d)", m.respawns, len(items))
			return m.resendBatchCmd(items)
		}
		// v1: the FIRST completed turn after a batch send closes every
		// board row of the batch — per-item close-outs over multi-turn
		// batches (the boss answering items one turn at a time) are a
		// later wave; good enough now.
		if len(m.batchDoneIDs) > 0 {
			if tb, ok := m.team(); ok {
				for id := range m.batchDoneIDs {
					tb.QueueItemDone(id)
				}
				qdebugf("batch turn completed: %d board row(s) done", len(m.batchDoneIDs))
			}
			m.batchDoneIDs = nil
			m.batchInFlight = false
		}
		if len(m.queue) > 0 {
			// the boss is free: flush the backlog (ONE batch when >1)
			return m.flushQueued()
		}
	}
	return permCmd // nil unless this event was a /bypass-armed EvPermission (see above)
}

// ---------------------------------------------------------------- older-history pagination (app wiring)
//
// The panel+backend seams (state.SessionPager, panels.ThreadPager /
// PrependOlder / TranscriptRows / PreserveAnchor / AtTranscriptTop) are
// frozen and proven in isolation; THIS section is the whole app loop
// around them, four verbs:
//
//	pagerKick      — attach: bind the ONE primary-thread pager + fire the
//	                 metadata-only seed hop (rides every applyEvent; O(1)
//	                 early-exit once bound). Also the reconnect seam:
//	                 streamReconnectedMarker re-arms the failure backoff
//	                 (and re-opens a dead seed).
//	maybeArmOlder  — gesture: wheel-up / ↑ / pgup landing on
//	                 AtTranscriptTop fires ONE async hop through
//	                 pager.StartOlder (its guards own single-flight,
//	                 top-latch and the 3-strike backoff). Question /
//	                 permission floats suppress the arm (a parked turn
//	                 outranks a walk), the thread-focus view and the
//	                 non-chat tabs keep their own scroll realms.
//	applyOlderPage — landing: dedupe + normalize the page, splice it into
//	                 the panel (PrependOlder) AND the reducer ledger
//	                 (st.Chat — or the next SetState re-base would erase
//	                 the splice), advance the walk (FinishOlder /
//	                 FailOlder), and PreserveAnchor the reader's row.
//	                 NO SetState call: the panel already holds the splice,
//	                 and the next natural event re-bases onto identical
//                 content (rev-gated render, offset untouched).
//
// Follow latch & scroll rulings: the splice never snaps a parked reader
// to the tail (PrependOlder owns that purity), and a reader PARKED AT
// THE TOP never gets auto-jumped on a landing — PreserveAnchor keeps
// their row pinned, the fresh page materializes ABOVE their viewport,
// and the NEXT wheel-up through it re-arms the walk at row 0.
// Employee-thread pagination is a follow-up: the panel's thread view
// owns no history walk, and ThreadPager binds to the primary session
// only.

// pagerKick — the attach surface, riding EVERY backend/synthetic event:
// (a) a stream reconnect re-arms the walk's backoff + re-opens a dead
// seed; (b) a backend without the state.SessionPager seam latches
// pagerNoSeam (harness stubs degrade, never probe twice); (c) the first
// event with a resolvable session id binds the pager and fires the ONE
// seed hop (metadata-only — rows discarded, see pagerSeedMsg); (d) a
// primary-id flip (/new, batch respawn) rebinds fresh on the next event.
func (m *Model) pagerKick(ev state.Event) tea.Cmd {
	if ev.Kind == state.EvStatus && ev.Text == streamReconnectedMarker {
		if m.pager != nil {
			m.pager.ResetFailures()
		}
		m.pagerSeedFailed = false
	}
	if m.pagerNoSeam || m.chat == nil || m.backend == nil {
		return nil
	}
	sp, ok := m.backend.(state.SessionPager)
	if !ok {
		m.pagerNoSeam = true
		return nil
	}
	sid := m.PrimarySessionID()
	if sid == "" && m.st.Mode == state.ModeDemo {
		sid = pagerDemoSession
	}
	if sid == "" {
		return nil // live pre-attach (no primary resolved yet): try the next event
	}
	if m.pager != nil && m.pagerSession == sid {
		if m.pagerSeeding || m.pagerSeeded || m.pagerSeedFailed {
			return nil
		}
		// bound but never seeded (a reconnect re-opened the latch): re-seed.
	} else {
		m.pager = panels.NewThreadPager(sid)
		m.pagerSession = sid
		m.pagerBaseIDs, m.pagerBaseUtext = nil, nil
	}
	m.pagerSeeding, m.pagerSeeded, m.pagerSeedFailed = true, false, false
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), pagerFetchTimeout)
		defer cancel()
		page, err := sp.MessagesPage(ctx, sid, "", panels.ThreadOlderPageSize)
		return pagerSeedMsg{sid: sid, page: page, err: err}
	}
}

// resetPager drops the whole binding (/new): the NEXT event rebumps a
// fresh pager for the minted session (pagerKick), and any in-flight
// landing from the OLD session dies on the sid guard.
func (m *Model) resetPager() {
	m.pager = nil
	m.pagerSession = ""
	m.pagerSeeding, m.pagerSeeded, m.pagerSeedFailed = false, false, false
	m.pagerBaseIDs, m.pagerBaseUtext = nil, nil
}

// capturePagerBaseline freezes the dedupe reference at SEED LANDING: the
// transcript ids (assistant rows match normalized fetched ids verbatim)
// plus user texts WITH multiplicity (the live echo's "user-"+<seq> ids
// can never collide with a fetched "user-"+<serveID>, so user rows match
// by text; see the Model field block for the ordering argument).
func (m *Model) capturePagerBaseline() {
	m.pagerBaseIDs = make(map[string]bool, len(m.st.Chat))
	m.pagerBaseUtext = make(map[string]int, len(m.st.Chat)/2)
	for _, c := range m.st.Chat {
		if c.ID != "" {
			m.pagerBaseIDs[c.ID] = true
		}
		if c.From == "user" {
			m.pagerBaseUtext[c.Text]++
		}
	}
}

// maybeArmOlder — the scroll-to-top gesture's hook: called AFTER the
// chat panel settled a wheel-up / ↑ / pgup. Every guard lives here (or
// in the pager): chat tab active, no thread-focus pane, no
// question/permission float, transcript AT row 0, and the pager's own
// seeded / top / single-flight / backoff contract. True arms ONE async
// hop through the same tea.Cmd plumbing the /session listing rides
// (never synchronous in Update).
func (m *Model) maybeArmOlder() tea.Cmd {
	if m.chat == nil || m.pager == nil || !m.pagerSeeded {
		return nil
	}
	if m.tabs.ActiveIndex() != 0 || m.threadFocus != nil {
		return nil
	}
	if m.permQ.front() != nil || m.question != nil {
		return nil
	}
	if !m.chat.AtTranscriptTop() {
		return nil
	}
	cursor, ok := m.pager.StartOlder()
	if !ok {
		return nil
	}
	sp, ok := m.backend.(state.SessionPager)
	if !ok {
		return nil // impossible (the kick latched pagerNoSeam) — defensive
	}
	sid := m.pagerSession
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), pagerFetchTimeout)
		defer cancel()
		page, err := sp.MessagesPage(ctx, sid, cursor, panels.ThreadOlderPageSize)
		return pagerOlderMsg{sid: sid, page: page, err: err}
	}
}

// applyOlderPage — one hop's landing: fail silently into the pager's
// backoff; on success DEDUPE + NORMALIZE (filterOlderRows), splice the
// survivors panel-side AND ledger-side with byte-stable anchoring. The
// ledger prepend deliberately bypasses capChat: its fuse drops the
// OLDEST entries first, which would eat exactly the pages the walk
// works for — history growth is gesture-bounded (and chatCap still
// guards the live append stream).
func (m *Model) applyOlderPage(msg pagerOlderMsg) {
	if m.pager == nil || m.chat == nil || msg.sid != m.pagerSession {
		return // superseded binding: drop the stale landing
	}
	if msg.err != nil {
		m.pager.FailOlder() // silent — 3 strikes latch, never a banner
		return
	}
	rows := m.filterOlderRows(msg.page.Rows)
	if len(rows) > 0 {
		before := m.chat.TranscriptRows()
		m.chat.PrependOlder(rows)
		m.st.Chat = append(pagerRowsToChat(rows), m.st.Chat...)
		m.chat.PreserveAnchor(before)
	}
	m.pager.FinishOlder(msg.page.NextCursor, msg.page.HasMore)
	// an all-overlap page (0 fresh rows) still advances the cursor — the
	// walk skips the hydrated region without a pixel moving.
}

// filterOlderRows drops fetched rows already on screen and NORMALIZES
// the survivors' ids into the transcript's stream conventions:
// bossmsg-<id> for assistant, user-<id> for user, the raw id otherwise
// ("reasoning"/serve-side tool rows don't round-trip — the office-note
// namespace never collides). Dedupe: normalized id in the baseline, and
// for user rows (whose live echoes mint unmatchable local-seq ids) the
// TEXT multiplicity — consume one occurrence per dropped row.
func (m *Model) filterOlderRows(rows []state.SessionMessageRow) []state.SessionMessageRow {
	out := make([]state.SessionMessageRow, 0, len(rows))
	for _, r := range rows {
		id := pagerRowID(r)
		if m.pagerBaseIDs[id] {
			continue
		}
		if r.Role == "user" && m.pagerBaseUtext[pagerRowText(r)] > 0 {
			m.pagerBaseUtext[pagerRowText(r)]--
			continue
		}
		r.ID = id
		out = append(out, r)
	}
	return out
}

// pagerRowID — the transcript id ONE fetched row splices under: the SAME
// conventions the live stream mints (opencode.go's EvChatBoss
// "bossmsg-"+info.ID; the user echo's "user-" prefix — with the serve id
// where the stream used a local seq, the only colliding-safe choice).
func pagerRowID(r state.SessionMessageRow) string {
	switch r.Role {
	case "assistant":
		return "bossmsg-" + r.ID
	case "user":
		return "user-" + r.ID
	default:
		return r.ID
	}
}

// pagerRowText — ONE row's spliced body: the text-bearing parts joined
// exactly like the panel's sessionRowToChat (tool/reasoning-only rows
// render empty — history splices for READING, never replaying calls).
func pagerRowText(r state.SessionMessageRow) string {
	var texts []string
	for _, p := range r.Parts {
		if p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	return strings.Join(texts, "\n\n")
}

// pagerRowsToChat — one filtered+normalized page's LEDGER form: field-
// for-field identical to the panel's sessionRowToChat (both see the same
// normalized ids), so the next SetState re-base paints byte-identical
// rows over the splice. (wtool-*/wthink-* entries never occur: a session
// message page carries user/assistant turns only — worker-thread rows
// are derived from tool EVENTS, never from the message history.)
func pagerRowsToChat(rows []state.SessionMessageRow) []state.ChatMsg {
	out := make([]state.ChatMsg, 0, len(rows))
	for _, r := range rows {
		m := state.ChatMsg{ID: r.ID, Text: pagerRowText(r), At: r.Created}
		switch r.Role {
		case "user":
			m.From, m.Kind = "user", "user"
		case "assistant":
			m.From, m.Kind = "boss", "boss"
		default:
			m.From = "office"
		}
		out = append(out, m)
	}
	return out
}

// applyDelegation — P3: while the boss dispatched work to children the
// primary session can sit quiet at its typing placeholder for minutes
// (dead feedback: "typing…" while nobody is typing). Track the last
// boss-side activity and, on every event, recompute st.BossDelegating:
// a pending boss placeholder exists AND no boss stream/thought/primary-
// tool activity for > delegatingQuietTicks AND ≥1 hired employee is
// visibly busy (working/to-manager/meeting). It clears instantly — any
// boss stream/thought/tool/bubble event refreshes the clock the same
// reduce, so the >-horizon comparison can never trigger.
func (m *Model) applyDelegation(ev state.Event) {
	if isBossActivity(ev) {
		m.lastBossActivity = m.st.Tick
		// W1 — the wall-clock twin: real silence is measured against this.
		// Only REAL server-side boss traffic refreshes the wedge clock —
		// the send-side typing placeholder is the UI's own optimistic
		// staging (opencode.go emits one on EVERY send/queue-flush), not
		// proof of life. Letting it stamp the clock restarted the 2m
		// window per send, so a wedged turn printed one red row per turn
		// forever. The LATCH, though, is NOT re-armed here: mid-turn
		// traffic must not re-arm the one-shot (a long turn with quiet
		// stretches printed the same row over and over); it re-arms only
		// at the turn END (completion//stop unwind). The hint bar derives
		// from the clock itself, so a recovered turn clears its own
		// banner without any latch flip.
		if !isSendSidePlaceholder(ev) {
			m.lastBossActivityAt = time.Now()
		}
	}
	busy := 0
	for _, e := range m.st.Employees {
		if e.Role == state.RoleManager {
			continue
		}
		switch e.Sprite {
		case state.SpriteWorking, state.SpriteToManager, state.SpriteMeeting:
			busy++
		}
	}
	m.st.BossDelegating = hasPendingBoss(m.st) && busy > 0 &&
		m.st.Tick-m.lastBossActivity > delegatingQuietTicks
}

// isBossActivity — the boss-side event set that resets the delegation
// quiet clock: any boss chat event (stream delta, placeholder, pinned or
// error bubble), a boss EvThought, a primary-session EvTool. NB: the
// wedge watchdog's WALL clock takes a stricter subset of this — the
// send-side placeholder above is excluded there (isSendSidePlaceholder).
func isBossActivity(ev state.Event) bool {
	switch ev.Kind {
	case state.EvChatBoss:
		return true
	case state.EvThought:
		return ev.EmployeeID == "boss" || ev.EmployeeName == "" || ev.EmployeeName == "boss"
	case state.EvTool:
		return ev.EmployeeName == "" || ev.EmployeeName == "boss"
	}
	return false
}

// isSendSidePlaceholder — the optimistic typing bubble the backend stages
// synchronously on EVERY Send ("boss-"+chatSeq, Pending:true, Text:"" —
// opencode.go's pendingID). It proves only that a prompt left the client,
// never that the server is alive, so it must NOT stamp or clear the wedge
// watchdog's wall clock: each send/queue-flush used to restart the 2m
// silence window, noting one "boss turn wedged" line per turn forever.
// Stream deltas (Pending with TEXT) and every pinned/completed boss
// bubble remain real traffic.
func isSendSidePlaceholder(ev state.Event) bool {
	return ev.Kind == state.EvChatBoss && ev.Msg.Pending && ev.Msg.Text == ""
}

// --- boss-wedge watchdog (W1) ------------------------------------------------

// bossWedgeAfter — the silence threshold: a pending boss turn (typing
// placeholder) with ZERO server-side boss traffic for this long counts as
// wedged. A parked question hold NEVER does — the boss is waiting on the
// USER's answer then, so silence is user-owned, not a wedge. Wall clock
// only — the tick governor's cadence varies 180ms–3s, so the threshold
// measures m.lastBossActivityAt, never st.Tick.
const bossWedgeAfter = 120 * time.Second

// wedgeHint — the statusline swap while the wedge latch is set (rides the
// hint seam, warn class). It never auto-kills: /stop is offered, but the
// turn may still complete on its own and queued input keeps working.
const wedgeHint = "boss looks wedged — no reply for 2m · /stop to abort · enter queues anyway"

// wedgeNote — the ONE line the watchdog writes per wedge episode: an
// ACTIVITY-TAB entry (the "[stamp] …" seam, same as describeEvent /
// [memory] recorded), NEVER a chat row — the transcript stays clean and
// the member-visible warning rides the status bar (wedgeHint) instead.
const wedgeNote = "boss turn wedged: no traffic for 2m — /stop unwinds it (queue intact); the turn may still complete on its own"

// checkBossWedge runs off the EvTick cheap loop (applyEvent's tick branch).
// NOTICE-ONLY by design: NEVER auto-kill, NEVER auto-respawn the turn — a
// slow model round or a long delegation quiet spell is indistinguishable
// from a dead one, and firing either would destroy real work. Armed when a
// boss placeholder is outstanding; a parked question hold is NEVER armed
// (waiting on the user's answer is not a wedge). Fires ONCE per wedge
// (wedgeNoted, re-armed only by REAL server-side boss traffic in
// applyDelegation — the send-side placeholder is isSendSidePlaceholder and
// cannot re-arm — or by resetServerTurn//stop closing the turn): one
// activity-tab line plus the hint swap, then silence until recovery. The
// note NEVER lands in st.Chat — a transcript row would linger for the
// whole session (and Snapshot/hydrate round-trips) long after the turn
// recovered; the status bar hint is derived from the silence clock, so it
// retires itself.
func (m *Model) checkBossWedge() {
	if m.wedgeNoted || m.questionParked || !hasPendingBoss(m.st) {
		return // already said, waiting on the user's question answer, or nothing outstanding
	}
	if !m.bossWedgeOverdue() {
		return
	}
	m.wedgeNoted = true
	m.activity.Add(fmt.Sprintf("[%s] %s", chrome.OfficeClock(m.st.Tick), wedgeNote))
	m.activityAdds++ // digest term (same seam as describeEvent adds)
}

// bossWedgeOverdue — the wall-clock wedge predicate, shared by the
// one-shot fire (checkBossWedge) and the live hint swap: outstanding
// boss placeholder AND the last REAL traffic opencode-side is older than
// the threshold. Zero clock = never fired yet this run.
func (m *Model) bossWedgeOverdue() bool {
	if m.lastBossActivityAt.IsZero() {
		return false
	}
	threshold := m.wedgeAfter
	if threshold <= 0 {
		threshold = bossWedgeAfter
	}
	return time.Since(m.lastBossActivityAt) >= threshold
}

// SetWedgeAfterForShot is the uishot/test harness seam for the wedge
// threshold (same additive pattern as SelectTab/PersistSession): the
// synchronous proof drivers can't wait out the 2m production floor. A
// zero/negative value restores the default.
func (m *Model) SetWedgeAfterForShot(d time.Duration) { m.wedgeAfter = d }

// idleWrapNotice is the one transcript recap when a shift goes idle and
// the last real chat was not from the boss or office.
const idleWrapNotice = "office recap: quiet for 2m with the floor idle — last chat wasn't from the boss or office. Looks like the shift finished."

// idleWrapPromptHead is the boss-facing check-in. It rides Send like other
// [theboringfloor] follow-ups so the member sees the ask, then the recap.
const idleWrapPromptHead = "[theboringfloor] check-in: idle 2m, no workers running, last chat wasn't from the boss or office. Recap the shift for the member in a few lines (what finished, what's blocked, or that nothing landed). Do not start new work."

func lastUserChat(st state.OfficeState) string {
	for i := len(st.Chat) - 1; i >= 0; i-- {
		c := st.Chat[i]
		if c.From == "user" && strings.TrimSpace(c.Text) != "" {
			return c.Text
		}
	}
	return ""
}

func idleWrapPrompt(st state.OfficeState) string {
	if u := lastUserChat(st); u != "" {
		return idleWrapPromptHead + " Their last ask: " + clipRunes(strings.TrimSpace(u), 80)
	}
	return idleWrapPromptHead
}

func hasPendingOffice(st state.OfficeState) bool {
	for _, c := range st.Chat {
		if c.From == "office" && c.Pending {
			return true
		}
	}
	return false
}

func (m *Model) liveWorkerCount() int {
	n := 0
	for _, e := range m.st.Employees {
		if e.Role == state.RoleManager || e.Role == state.RoleHR || e.Role == state.RoleCTO {
			continue
		}
		switch e.Sprite {
		case state.SpriteWorking, state.SpriteToManager, state.SpriteMeeting:
			n++
		}
	}
	return n
}

func (m *Model) officeWorkInFlight() bool {
	if hasPendingBoss(m.st) || hasPendingOffice(m.st) {
		return true
	}
	if m.questionParked || m.question != nil || m.permQ.front() != nil {
		return true
	}
	if m.liveWorkerCount() > 0 {
		return true
	}
	for _, t := range m.st.Tasks {
		if t.Status == state.TaskInProgress {
			return true
		}
	}
	return false
}

// lastRealChatFrom walks the transcript newest-first. Skips empty
// placeholders, tool/think rows, and local office notices (Kind !=
// "office") so a "queued as item" row cannot fake a wrap. Boss or
// concierge (From office + Kind office) are real wraps.
func lastRealChatFrom(st state.OfficeState) string {
	for i := len(st.Chat) - 1; i >= 0; i-- {
		c := st.Chat[i]
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		switch c.Kind {
		case "tool", "wtool", "wthink", "think", "wdiff":
			continue
		}
		if c.From == "office" && c.Kind != "office" {
			continue
		}
		return c.From
	}
	return ""
}

func lastChatIsBossOrOffice(st state.OfficeState) bool {
	switch lastRealChatFrom(st) {
	case "boss", "office":
		return true
	}
	return false
}

// updateIdleWrap arms a 2m recap clock only when a busy shift goes idle
// AND the last real chat is not already from the boss or office. A late
// wrap (boss/office bubble while armed) disarms without a recap.
func (m *Model) updateIdleWrap() {
	if m.officeWorkInFlight() {
		if !m.shiftBusy {
			m.shiftBusy = true
			m.ghostNoted = false
		}
		m.ghostArmAt = time.Time{}
		return
	}
	if lastChatIsBossOrOffice(m.st) {
		m.ghostArmAt = time.Time{}
		m.shiftBusy = false
		return
	}
	if m.shiftBusy {
		m.shiftBusy = false
		if lastRealChatFrom(m.st) != "" && m.ghostArmAt.IsZero() {
			m.ghostArmAt = time.Now()
		}
	}
}

func (m *Model) checkIdleWrap() tea.Cmd {
	if m.ghostNoted || m.ghostArmAt.IsZero() || m.officeWorkInFlight() {
		return nil
	}
	if lastChatIsBossOrOffice(m.st) || lastRealChatFrom(m.st) == "" {
		m.ghostArmAt = time.Time{}
		return nil
	}
	threshold := m.wedgeAfter
	if threshold <= 0 {
		threshold = bossWedgeAfter
	}
	if time.Since(m.ghostArmAt) < threshold {
		return nil
	}
	m.ghostNoted = true
	prompt := idleWrapPrompt(m.st)
	current := m.currentBackend
	return func() tea.Msg {
		if err := sendViaCurrentBackend(current, prompt, nil, ""); err != nil {
			return idleWrapFailMsg{err: err}
		}
		return idleWrapSentMsg{}
	}
}

// ActivityLines is the uishot/test read seam for the activity tab's raw
// log lines (same additive harness pattern as SetWedgeAfterForShot):
// proofs count entries byte-deterministically without parsing the
// clipped, ANSI-styled viewport frame.
func (m Model) ActivityLines() []string { return m.activity.Lines() }

// team type-asserts the optional teamBackend seam (live/demo backends).
func (m *Model) team() (teamBackend, bool) {
	if m.backend == nil {
		return nil, false
	}
	tb, ok := m.backend.(teamBackend)
	return tb, ok
}

// --- free-queuing status compose -------------------------------------------

// applyBusyStatus paints the free-queuing compose onto the status line
// ("busy · N queued (server)"), saving the prior line once so the turn
// completion (or /stop) can restore it.
func (m *Model) applyBusyStatus() {
	if m.serverQueued <= 0 {
		return
	}
	if !m.busyStatus {
		m.busySaved = m.st.StatusLine
		m.busyStatus = true
	}
	m.st.StatusLine = fmt.Sprintf("busy · %d queued (server)", m.serverQueued)
	m.tabs.SetState(m.st)
}

// clearBusyStatus restores the pre-busy status line when the busy compose
// owns it.
func (m *Model) clearBusyStatus() {
	if !m.busyStatus {
		return
	}
	m.st.StatusLine = m.busySaved
	m.busySaved = ""
	m.busyStatus = false
}

// serveDiedStatusMarker — the W4 string contract with opencode.go's
// watchServe: the "serve died" note is statusline-only by nature (the next
// EvStatus overwrites it), so it rides the F5a escalation seam into the
// transcript as a red office row.
const serveDiedStatusMarker = "[theboringfloor] opencode serve died"

// resetServerTurn closes the free-queuing tally for the busy turn that just
// ended (completion / error / /stop): placeholder turn count back to 0, the
// status compose restored, the concierge routing notice re-armed for the
// next busy turn — and the W1 wedge watchdog re-armed with it (the turn is
// closed, whatever its end).
func (m *Model) resetServerTurn() {
	m.serverQueued = 0
	m.conciergeNoted = false
	m.wedgeNoted = false // W1: the turn ended (completion//stop) — re-arm the watchdog
	m.clearBusyStatus()
	if m.chat != nil {
		m.chat.SetServerTurn(0)
	}
}

// --- concierge routing ------------------------------------------------------

// routeBusySend decides where a prompt typed while the boss looked busy
// actually goes (busySendReqMsg). CONCIERGE ROUTING fires only when the
// boss is genuinely busy — an outstanding pending placeholder/stream, the
// delegation quiet state, or a question hold resolved-but-turn-incomplete —
// AND cfg.Boss.Concierge is on: the concierge must never answer in parallel
// with an idle boss (zero duplication). The seam is text-only, so an
// attachment-carrying prompt keeps riding the boss queue (files must never
// be silently dropped). A backend without the ConciergeCapable seam
// degrades to the old free-send path with a one-time dim notice. Either
// routing notice prints at most ONCE per busy turn (conciergeNoted,
// re-armed by resetServerTurn).
func (m *Model) routeBusySend(text string, atts []state.Attachment) tea.Cmd {
	busy := hasPendingBoss(m.st) || m.st.BossDelegating || m.questionParked
	if busy && m.cfg != nil && m.cfg.Boss.Concierge && len(atts) == 0 {
		if m.currentBackend.supportsConcierge() {
			notice := !m.conciergeNoted
			m.conciergeNoted = true
			qdebugf("concierge: routed %q (boss busy)", text)
			current, plan := m.currentBackend, m.plan
			return func() tea.Msg {
				if handled, err := current.sendConcierge(text); handled {
					if err != nil {
						return sendErrMsg{err: err}
					}
					return conciergeSentMsg{text: text, notice: notice}
				}
				agent := paneAgent(plan)
				if err := sendViaCurrentBackend(current, text, nil, agent); err != nil {
					return sendErrMsg{err: err}
				}
				return busySentMsg{text: text, agent: agent}
			}
		}
		if !m.conciergeNoted {
			m.conciergeNoted = true
			m.notice("(concierge unavailable — boss queued it)")
		}
	}
	current, plan := m.currentBackend, m.plan
	return func() tea.Msg {
		agent := paneAgent(plan)
		if err := sendViaCurrentBackend(current, text, atts, agent); err != nil {
			cleanupAttachments(atts) // nobody will retry this prompt
			return sendErrMsg{err: err}
		}
		cleanupAttachments(atts)
		return busySentMsg{text: text, agent: agent}
	}
}

// --- /stop (abort + clean unwind) -------------------------------------------

// stopWork is /stop: abort the primary AND every live child session, then
// unwind the in-flight UI cleanly. The client queue is NOT touched (queued
// items send on the next turn), permission/question roadblocks stay put,
// and the whole thing plays no sound — a stop is not an error.
//
// G1 — the abort RPC is best-effort: a dead serve or a dead transport
// makes AbortSessions fail, and the old early-return LEFT THE OFFICE
// STRANDED on the wedged turn (placeholder forever "typing…", the wedge
// watchdog's advice literally not working). The OFFICE always recovers:
// on failure the result landing (stopAbortResultMsg) logs + prints one
// dim note; the unwind below already ran. The turn may still finish
// server-side later — its late completion then appends as a fresh bubble:
// the placeholder is already collapsed, so the reducer's (c) branch takes
// the non-pending "bossmsg-"+id append path and nothing double-pops.
//
// G5 (the UI-freeze fix) — the abort RPC is ALSO off the UI goroutine: it
// rides the returned tea.Cmd (bubbletea runs cmds on their own goroutine)
// and lands as stopAbortResultMsg. A wedged serve/network can therefore
// never park /stop — the unwind + statusline below happen synchronously
// and the input stays live while the hop is out. The abort seam stays
// AbortSessions() error: per-call bounding is the BACKEND's job
// (internal/backend's abortCallTimeout ctx), so harness stubs compile
// untouched.
func (m *Model) stopWork() tea.Cmd {
	var abortCmd tea.Cmd
	if ab, ok := m.backend.(state.SessionAborter); ok {
		abortCmd = func() tea.Msg { return stopAbortResultMsg{err: ab.AbortSessions()} }
	}
	m.unwindStoppedWork()
	m.resetServerTurn()
	m.st.StatusLine = fmt.Sprintf("stopped current work — queue intact (%d items)", len(m.queue))
	m.tabs.SetState(m.st)
	return abortCmd
}

// unwindStoppedWork is the /stop clean unwind (this wave's CX):
//
//	(a) every pending boss typing placeholder collapses to a dim
//	    "stopped by user" line;
//	(b) a STREAMING boss bubble keeps its text with a " (stopped)"
//	    appendix;
//	(c) every still-running tool entry (boss inline or worker-thread)
//	    swings to "✗ aborted" (dim-red at render);
//	(d) active worker threads collapse with a "✗ stopped" summary;
//	(e) BossThinking / BossDelegating clear.
//
// A pending concierge (office) answer unwinds the same way: an empty
// placeholder collapses to the "stopped by user" office line; a streaming
// one keeps its text with the " (stopped)" appendix. AbortSessions covers
// the concierge session on backends that implement it (a missing seam
// degrades quietly — the unwind above still runs).
func (m *Model) unwindStoppedWork() {
	next := make([]state.ChatMsg, 0, len(m.st.Chat))
	for _, c := range m.st.Chat {
		switch {
		case c.From == "office" && c.Pending && c.Text == "":
			// concierge placeholder → dim office line, like the boss's
			next = append(next, state.ChatMsg{
				ID: c.ID, From: "office", Text: "stopped by user",
				At: time.Now().UnixMilli(),
			})
		case c.From == "office" && c.Pending:
			// concierge streaming bubble → text kept + stopped appendix
			c.Pending = false
			c.Text = strings.TrimSuffix(c.Text, " (stopped)") + " (stopped)"
			next = append(next, c)
		case c.From == "boss" && c.Pending && strings.HasPrefix(c.ID, "boss-"):
			// (a) typing placeholder → dim office line
			next = append(next, state.ChatMsg{
				ID: c.ID, From: "office", Text: "stopped by user",
				At: time.Now().UnixMilli(),
			})
		case c.From == "boss" && c.Pending:
			// (b) streaming bubble → text preserved + stopped appendix
			c.Pending = false
			c.Text = strings.TrimSuffix(c.Text, " (stopped)") + " (stopped)"
			next = append(next, c)
		case c.Kind == "tool" && c.Meta == "running":
			// (c) boss inline tool: running → aborted
			c.Meta = "aborted"
			next = append(next, c)
		case (c.Kind == "wtool" || c.Kind == "wthink") &&
			strings.HasPrefix(c.Meta, "running"):
			// (c) worker-thread tool/think: running → aborted
			// (Meta carrier: state ␟ tick — keep the tick half)
			c.Meta = "aborted" + strings.TrimPrefix(c.Meta, "running")
			next = append(next, c)
		default:
			next = append(next, c)
		}
	}
	m.st.Chat = next
	m.st.BossThinking = false
	m.st.BossDelegating = false
	if m.chat != nil {
		// (d) the roster's busy sprites are the active worker set
		for _, e := range m.st.Employees {
			if e.Role == state.RoleManager {
				continue
			}
			switch e.Sprite {
			case state.SpriteWorking, state.SpriteToManager, state.SpriteMeeting:
				m.chat.MarkThreadStopped(e.Name)
			}
		}
	}
}

// composeBatch builds the ONE batch-dispatch prompt the boss session
// decomposes per its manager discipline: numbered independent work items,
// as many parallel sub-agents as the items decompose into for the
// non-trivial ones, a closing status table.
// Attachment-carrying items get a machine " 📎N" suffix on their numbered
// line; the actual file parts ride the same send (dispatchQueued).
func composeBatch(items []queueEntry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[BATCH DISPATCH — %d requests arrived while you were busy. "+
		"Treat each as an independent numbered work item: do trivial ones inline; "+
		"for non-trivial independent items DISPATCH AS MANY PARALLEL SUB-AGENTS "+
		"AS THE ITEMS DECOMPOSE INTO (never 1 or 2 for real work) per your "+
		"manager discipline; then finalize with a one-line-per-item status table.]\n",
		len(items))
	for i, it := range items {
		fmt.Fprintf(&sb, "%d. %s", i+1, it.text)
		if n := len(it.atts); n > 0 {
			fmt.Fprintf(&sb, " 📎%d", n)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// flushQueued drains the backlog: the intelligent flush after a turn
// completes (or a parked question unblocks).
func (m *Model) flushQueued() tea.Cmd {
	return m.dispatchQueued(false)
}

// dispatchQueued sends the backlog as ONE composed [BATCH DISPATCH] prompt
// when >1 items are queued; exactly 1 item keeps the plain FIFO send path.
// manual=true is /route — force the send NOW, bypassing the busy gate.
// Slash inputs never reach the queue (the chat panel dispatches them
// immediately) — the single-item slash guard stays defensive.
func (m *Model) dispatchQueued(manual bool) tea.Cmd {
	if len(m.queue) == 0 {
		return nil
	}
	items := m.queue
	m.queue = nil
	if m.chat != nil {
		m.chat.SetQueueLen(0)
	}
	if manual {
		if len(items) > 1 {
			m.notice(fmt.Sprintf("routed manually — batch dispatching %d queued items now", len(items)))
		} else {
			m.notice("routed manually — sending now")
		}
	}
	texts := make([]string, len(items))
	var boardIDs []string
	var batchAtts []state.Attachment // every item's chips ride the flush
	for i, it := range items {
		texts[i] = it.text
		batchAtts = append(batchAtts, it.atts...)
		if it.boardID != "" {
			boardIDs = append(boardIDs, it.boardID)
		}
	}
	sendText := texts[0]
	batch := len(texts) > 1
	if batch {
		sendText = composeBatch(items)
		qdebugf("flush: batch dispatching %d items as ONE send", len(texts))
	} else {
		qdebugf("flush %q (plain send, single item)", texts[0])
	}
	m.batchItems = items
	m.batchSummaries = texts
	if batch {
		m.batchInFlight = true
		m.batchRespawned = false
		m.batchSentAt = time.Now()
		if len(boardIDs) > 0 {
			m.batchDoneIDs = map[string]bool{}
			for _, id := range boardIDs {
				m.batchDoneIDs[id] = true
			}
		}
	}
	current, plan := m.currentBackend, m.plan
	send := func() tea.Msg {
		if !batch && strings.HasPrefix(texts[0], "/") {
			return slashMsg{text: texts[0]}
		}
		agent := paneAgent(plan)
		if err := sendViaCurrentBackend(current, sendText, batchAtts, agent); err != nil {
			// no cleanup: a respawn retry (queueSendErrMsg) may still
			// need the files; IT owns the cleanup on terminal failure.
			return queueSendErrMsg{err: err, items: items, batch: batch, retry: false}
		}
		cleanupEntries(items)
		return chatSentMsg{text: sendText, agent: agent}
	}
	return send
}

// resendBatchCmd — the ONE failure respawn: ResetPrimary(true) then resend
// the SAME composed batch (attachments included) on the fresh session.
// Errors come back with retry=true so the loop can never respawn twice for
// one flush call.
func (m *Model) resendBatchCmd(items []queueEntry) tea.Cmd {
	text := composeBatch(items)
	var atts []state.Attachment
	for _, it := range items {
		atts = append(atts, it.atts...)
	}
	current, plan := m.currentBackend, m.plan
	tb, _ := m.team()
	return func() tea.Msg {
		if tb != nil {
			if err := tb.ResetPrimary(true); err != nil {
				return queueSendErrMsg{err: fmt.Errorf("respawn: %w", err), batch: true, retry: true}
			}
		}
		agent := paneAgent(plan)
		if err := sendViaCurrentBackend(current, text, atts, agent); err != nil {
			return queueSendErrMsg{err: err, items: items, batch: true, retry: true}
		}
		cleanupEntries(items)
		return chatSentMsg{text: text, agent: agent}
	}
}

// handlePermissionEvent enqueues/opens permission asks from ANY session —
// boss/primary AND children ride the same queue now. Pending asks append
// behind the displayed front and the popover re-renders ("1 of N"); the
// Agent field names the requester. "resolved" drops the matching id from
// pending AND esc'd; a dropped front closes (or advances) the popover.
// The child's floor blocked sprite + activity line still come from the
// backend's EvBlocked/describeEvent paths — the popover is additional UI,
// not a floor-state replacement.
//
// BYPASS ARM: while m.bypassPerms is armed, a stray pending ask (emitted
// before the toggle's respawn landed) is answered allow-once IMMEDIATELY
// on the backend's AnswerPermission wire — no modal parks, no notify
// ping, no sound — and ONE dim transcript row logs the auto-approval.
// The returned cmd carries the wire reply (nil on every other path).
func (m *Model) handlePermissionEvent(ev state.Event) tea.Cmd {
	if ev.ToolState == "resolved" {
		m.permQ.resolve(ev.PermissionID)
		// the notification cohort shrinks with the ask; emptying it re-arms
		// the next cohort's ONE ping. Resolved events themselves are silent.
		delete(m.permNotifyIDs, ev.PermissionID)
		m.chat.SetPermission(m.permQ.view())
		return nil
	}
	if m.bypassPerms {
		m.notice(fmt.Sprintf(bypassAutoNotice, ev.ToolName))
		pid := ev.PermissionID
		current := m.currentBackend
		return func() tea.Msg {
			err := current.lease(func(b state.Backend) error {
				return b.AnswerPermission(pid, "once")
			})
			if err != nil {
				return sendErrMsg{err: err}
			}
			return nil
		}
	}
	agent := ev.EmployeeName
	if agent == "" {
		agent = "boss"
	}
	m.permQ.pending = append(m.permQ.pending, &permPrompt{
		ID: ev.PermissionID, ToolName: ev.ToolName, Summary: ev.ToolSummary,
		Agent: agent,
	})
	// notify cohort: the ask that flips the set 0→1 owns the ONE ping —
	// every later ask inside the cohort coalesces silently.
	if len(m.permNotifyIDs) == 0 {
		m.fireNotification("permission", permNotifyBody(agent, ev.ToolName))
	}
	m.permNotifyIDs[ev.PermissionID] = true
	m.playSound("alert") // every NEW ask opening the popover (boss or child)
	m.chat.SetPermission(m.permQ.view())
	return nil
}

// notifyTitle — every desktop ping's banner title: the product badge. The
// OS shows it once; the body carries the signal.
const notifyTitle = "theboringfloor"

// fireNotification — THE single bus call-site shape. Gates: an engine is
// wired (nil = headless harness), the terminal is UNfocused (these are
// look-away pings, never noise in front of your face), and /notify didn't
// turn them off in this session's config. Rate/silence details live in the
// engine (internal/notify), not here.
func (m *Model) fireNotification(kind, body string) {
	if m.notifyBus == nil || m.focused {
		return
	}
	if m.cfg != nil && m.cfg.UI.Notifications == "off" {
		return
	}
	m.notifyBus.Notify(kind, notifyTitle, body)
}

// permNotifyBody — the privacy-safe ask copy: agent + tool NAME only, never
// the ToolSummary (a banner is visible over your shoulder — file paths and
// command lines stay inside the terminal).
func permNotifyBody(agent, tool string) string {
	return "permission needed — " + agent + " needs " + tool
}

// doneNotifyBody — the completion copy: the boss's reply clipped to one
// line (clipRunes already flattens newlines), so a wall-of-markdown answer
// still fits on a banner.
func doneNotifyBody(text string) string {
	return "the boss is done — " + clipRunes(strings.TrimSpace(text), 60)
}

// permCohortFront — the ask a blur-cohort ping quotes: the displayed front
// first (it's what's on screen), else the newest esc'd one.
func (m *Model) permCohortFront() *permPrompt {
	if p := m.permQ.front(); p != nil {
		return p
	}
	if n := len(m.permQ.escd); n > 0 {
		return m.permQ.escd[n-1]
	}
	return nil
}

// handleQuestionEvent opens/closes the boss question WIZARD. Boss/primary
// requests (pending ToolState) park the turn: the open hold pages the
// request's question items through the chat panel's question popover one
// at a time (opencode waits at the question reply API — typing a chat
// message would queue but never resume it, the reported deadlock). An
// EvQuestion with NO structured Questions degrades to a single free-text
// page (legacy/flattened emitters — see legacyQuestionPage). "resolved"
// closes the whole request silently (ANY batched id — open and esc'd
// holds alike). Employee questions never open a popover — activity line
// only, like employee thoughts/permissions.
func (m *Model) handleQuestionEvent(ev state.Event) {
	if ev.ToolState == "resolved" {
		if m.question != nil && hasQuestionID(m.question.IDs, ev.QuestionID) {
			m.question = nil
			m.chat.SetQuestion(nil)
			// a closed question popover frees the region — re-push the
			// permission queue front (asks can have stacked behind it).
			m.chat.SetPermission(m.permQ.view())
		}
		if m.questionEscd != nil && hasQuestionID(m.questionEscd.IDs, ev.QuestionID) {
			m.questionEscd = nil
		}
		return
	}
	if ev.EmployeeName != "" && ev.EmployeeName != "boss" {
		return // child question: activity line only, no modal
	}
	if ev.QuestionID == "" {
		return
	}
	// the /bypass arming confirm is office-local: a REAL boss question
	// outranks it — close the confirm (cancel semantics, no-op) so the
	// fold-in below never appends boss pages onto the sentinel hold.
	if m.question != nil && m.question.IDs[0] == bypassConfirmID {
		m.question = nil
		m.chat.SetQuestion(nil)
	}
	items := ev.Questions
	if len(items) == 0 {
		items = []state.QuestionItem{legacyQuestionPage(ev)}
	}
	// fold-in (chosen over rejecting second requests): while ANY hold is
	// outstanding, a fresh pending boss question joins it — the batched
	// wire ids keep today's fold-in (a question call batching several
	// QuestionIDs emits one event per id; one submitted answer set
	// unblocks every batched id), and the fresh request's PAGES append as
	// extra wizard pages so the member walks everything outstanding in
	// one pass. A repeat of an already-known id folds nothing twice.
	h := m.question
	if h == nil {
		h = m.questionEscd
	}
	if h != nil {
		if !hasQuestionID(h.IDs, ev.QuestionID) {
			h.IDs = append(h.IDs, ev.QuestionID)
			h.Items = append(h.Items, items...)
			h.Answers = append(h.Answers, make([][]string, len(items))...)
			if h == m.question {
				// the page count grew behind the member — re-render so
				// the "N of M" header keeps telling the truth
				m.chat.SetQuestion(m.questionView(h))
			}
		}
		return
	}
	m.parkForQuestion()
	m.question = &questionHold{
		IDs:     []string{ev.QuestionID},
		Items:   append([]state.QuestionItem(nil), items...),
		Answers: make([][]string, len(items)),
	}
	m.chat.SetQuestion(m.questionView(m.question))
}

// parkForQuestion marks the turn parked at the question reply API: any
// pending boss typing placeholder ("boss-N") is dropped (the turn is
// WAITING, not typing), the status line reads "waiting for your answer",
// and the chat placeholder/enqueue gate flips to question-waiting.
func (m *Model) parkForQuestion() {
	m.questionParked = true
	kept := make([]state.ChatMsg, 0, len(m.st.Chat))
	for _, c := range m.st.Chat {
		if c.From == "boss" && c.Pending && strings.HasPrefix(c.ID, "boss-") {
			continue
		}
		kept = append(kept, c)
	}
	m.st.Chat = kept
	m.parkedStatus = m.st.StatusLine
	m.st.StatusLine = "[question] boss is waiting for your answer…"
	if m.chat != nil {
		m.chat.SetQuestionWaiting(true)
	}
}

func hasQuestionID(ids []string, id string) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

// compact — the live layout mode: the /compact session override beats
// brain.json ui.compact; compactLive=0 inherits the config.
func (m Model) compact() bool {
	switch m.compactLive {
	case 1:
		return true
	case 2:
		return false
	}
	return m.cfg.UI.Compact
}

// mobile — the AUTOMATIC narrow-terminal layout: window width below
// mobileMaxCols drops the horizontal floor|sidebar split for a vertical
// stack (compact floor band over the active panel). No command, no
// persistence: resize() picks the layout per window size, and the width
// term in frameDigest re-renders on the same frame the threshold crosses.
// Interplay with the other modes:
//   - zen (/zen) still wins outright — the transient fullscreen floor is
//     zen even when the window is narrow; mobile reshapes only the NORMAL
//     layout (Frame checks m.zen first).
//   - compact (/compact) stays orthogonal — it swaps topbar/tab-label
//     density and applies in either layout (Frame picks TopBarCompact
//     before the layout branch).
//
// m.width is always >= minCols by the time Frame renders (resize clamps).
func (m Model) mobile() bool {
	return m.width < mobileMaxCols
}

// floorBandH — the mobile floor band's row count: ~20% of the middle
// area, clamped to 8..14 (8 keeps a legible boss corner + pod row; 14
// leaves the panel the room it needs). resize clamps middleH >= 10, so
// the panel below always keeps >= 2 rows.
func (m Model) floorBandH() int {
	h := m.middleH / 5
	if h < 8 {
		h = 8
	}
	if h > 14 {
		h = 14
	}
	return h
}

// applyLayout pushes the current mode/UI config into every affected
// surface: tab-bar label density, the chat input rows, and the sidebar
// width. Called at build time and after /compact, /mode and /wide.
func (m *Model) applyLayout() {
	compact := m.compact()
	m.tabs.SetCompact(compact)
	m.chat.SetCompact(compact)
	if m.width > 0 {
		m.resize(m.width, m.height)
	}
}

// sidebarBase — the configured sidebar width before the narrow-terminal
// degrade: ui.sidebarWidth clamped to 26..100 wins outright; else the
// compact layout takes 30; else the 80-col default.
func (m Model) sidebarBase() int {
	if n := m.cfg.UI.SidebarWidth; n != 0 {
		if n < sidebarMin {
			n = sidebarMin
		}
		if n > sidebarMax {
			n = sidebarMax
		}
		return n
	}
	if m.compact() {
		return compactSidebarW
	}
	return defaultSidebarW
}

func (m *Model) resize(w, h int) {
	if w < minCols {
		w = minCols
	}
	if h < minRows {
		h = minRows
	}
	m.width, m.height = w, h
	m.middleH = h - 2
	if m.middleH < 1 {
		m.middleH = 1
	}
	base := m.sidebarBase()
	sw := base
	if w < degradeCols {
		// degrade gracefully: narrow terminals get a narrow sidebar
		sw = w / 3
		if sw < 20 {
			sw = 20
		}
		if sw > base {
			sw = base
		}
	}
	if w-sw < 8 {
		sw = w - 8
	}
	m.sidebar = sw
	m.floorW = w - sw
	if m.mobile() {
		// mobile stack: no sidebar — the tab strip owns the full width in
		// the rows under the floor band (Frame renders the band itself at
		// bandH; both sides share floorBandH() so they never drift apart).
		m.tabs.SetSize(w, m.middleH-m.floorBandH())
		// the browser rides the band's left slot — the switcher strip
		// eats one row (Frame renders the strip; both share the math).
		m.browser.SetSize(w, m.floorBandH()-1)
	} else {
		m.tabs.SetSize(sw, m.middleH)
		// the browser rides the LEFT floor slot — the switcher strip
		// eats one row of the slot's height.
		m.browser.SetSize(m.floorW, m.middleH-1)
	}
	if m.threadFocus != nil {
		// the open focus spans the whole middle region at ANY width —
		// desktop and terminal-shrink alike (Frame clamps the rest).
		m.threadFocus.SetSize(w, m.middleH)
	}
}

// --- reducer (exact port of node-legacy officeReducer + initialState) ------

func initialState(mode state.Mode) state.OfficeState {
	return state.OfficeState{
		Employees: []state.Employee{
			{ID: "manager", Name: "boss", Role: state.RoleManager, Seat: "manager", Sprite: state.SpriteAtDesk},
			{ID: "hr", Name: "hr", Role: state.RoleHR, Seat: "hr", Sprite: state.SpriteAtDesk},
		},
		Mode:       mode,
		StatusLine: fmt.Sprintf("[theboringfloor] %s - booting...", string(mode)),
	}
}

func capList[T any](list []T, maxN int) []T {
	if len(list) > maxN {
		return list[len(list)-maxN:]
	}
	return list
}

// appendChat appends one message, reusing the backing array when capacity
// allows instead of cloning the whole transcript per message. This is safe
// for the model-owned chat slice: the write always lands at index len(chat)
// — PAST the len of every previously returned header — so no earlier state
// snapshot's contents can change through the shared capacity (growth itself
// falls out of Go's natural doubling; a full-capacity append allocates).
// Only an APPEND rides the shared backing: the replace/merge arms still
// clone before mutating an element in place, which is what keeps swap
// semantics disjoint from the previous state.
func appendChat(chat []state.ChatMsg, msg state.ChatMsg) []state.ChatMsg {
	return append(chat, msg)
}

// capChat enforces the global chat fuse AND the per-kind fuses (all 10k —
// see the const block: history is retained in full in practice; a stream of
// thinking/tool entries drowns nothing). Only past the fuse does the oldest
// of a kind drop first.
func capChat(chat []state.ChatMsg) []state.ChatMsg {
	chat = capList(chat, chatCap)
	chat = capKind(chat, "think", thinkCap)
	chat = capKind(chat, "tool", toolCap)
	chat = capKind(chat, "wthink", thinkCap) // employee thread thoughts, per-CallID merged
	return chat
}

// capKind keeps at most maxN entries of the given Kind, dropping the oldest.
func capKind(chat []state.ChatMsg, kind string, maxN int) []state.ChatMsg {
	n := 0
	for _, m := range chat {
		if m.Kind == kind {
			n++
		}
	}
	if n <= maxN {
		return chat
	}
	drop := n - maxN
	out := make([]state.ChatMsg, 0, len(chat)-drop)
	for _, m := range chat {
		if m.Kind == kind && drop > 0 {
			drop--
			continue
		}
		out = append(out, m)
	}
	return out
}

func upsertTask(tasks []state.BoardTask, task state.BoardTask) []state.BoardTask {
	for i, t := range tasks {
		if t.ID == task.ID {
			next := append([]state.BoardTask(nil), tasks...)
			next[i] = task
			return next
		}
	}
	return append(tasks, task)
}

func findEmployee(st state.OfficeState, id string) *state.Employee {
	for i := range st.Employees {
		if st.Employees[i].ID == id {
			return &st.Employees[i]
		}
	}
	return nil
}

func setEmployee(st state.OfficeState, id string, fn func(e *state.Employee)) state.OfficeState {
	for i := range st.Employees {
		if st.Employees[i].ID == id {
			fn(&st.Employees[i])
		}
	}
	return st
}

// toolEntryID — the chat-entry ID one EvTool merges under (the reducer's
// EvTool case): boss/primary calls are "tool-<callID>", employee calls
// "wtool-<agent>-<callID>". The ToolOutput capture feed (applyEventCore)
// keys the chat panel's click-to-expand map by the SAME id, so the
// running→done merge that REPLACES the entry keeps its expansion state.
func toolEntryID(employeeName, callID string) string {
	if employeeName == "" || employeeName == "boss" {
		return "tool-" + callID
	}
	return "wtool-" + employeeName + "-" + callID
}

func reducer(st state.OfficeState, ev state.Event) state.OfficeState {
	switch ev.Kind {
	case state.EvTick:
		{
			tick := st.Tick + 1
			// drop expired balloons
			var bubbles []state.SpeechBubble
			for _, b := range st.Bubbles {
				if b.UntilTick > tick {
					bubbles = append(bubbles, b)
				}
			}
			st.Tick = tick
			st.Bubbles = bubbles
			// The legacy "every 140 ticks" chatter generator is gone — the
			// SocialClock (ambient.go) owns ALL self-originated floor chatter
			// now, planning beats off each EvTick in the model (its (d)
			// water-cooler covers the old solo case; the old line bank moved
			// to socialSoloBank). Explicit EvBubble backend events unchanged.
			return office.AdvanceSprites(st)
		}

	case state.EvHire:
		for _, e := range st.Employees {
			if e.ID == ev.Employee.ID {
				return st // id dedup — already on roster
			}
		}
		taken := make(map[string]bool, len(st.Employees))
		for _, e := range st.Employees {
			taken[e.Seat] = true
		}
		emp := ev.Employee
		emp.Seat = office.AssignSeat(taken, emp.Role)
		return officeStateWithEmployees(st, append(append([]state.Employee(nil), st.Employees...), emp))

	case state.EvFire:
		var emps []state.Employee
		var bubbles []state.SpeechBubble
		for _, e := range st.Employees {
			if e.ID != ev.EmployeeID {
				emps = append(emps, e)
			}
		}
		for _, b := range st.Bubbles {
			if b.EmployeeID != ev.EmployeeID {
				bubbles = append(bubbles, b)
			}
		}
		st.Employees = emps
		st.Bubbles = bubbles
		return st

	case state.EvDispatch:
		{
			ownerName := ev.Task.Owner
			if owner := findEmployee(st, ev.EmployeeID); owner != nil {
				ownerName = owner.Name
			}
			task := ev.Task
			task.Status = state.TaskInProgress
			task.Owner = ownerName
			st.Tasks = upsertTask(st.Tasks, task)
			st = setEmployee(st, ev.EmployeeID, func(e *state.Employee) {
				e.Sprite = state.SpriteToManager
				e.Task = task.Title
			})
			return st
		}

	case state.EvWorking:
		ownerName := ""
		if owner := findEmployee(st, ev.EmployeeID); owner != nil {
			ownerName = owner.Name
		}
		st = setEmployee(st, ev.EmployeeID, func(e *state.Employee) {
			e.Sprite = state.SpriteWorking
		})
		if ev.TaskID != "" {
			for i := range st.Tasks {
				if st.Tasks[i].ID == ev.TaskID {
					st.Tasks[i].Status = state.TaskInProgress
					if st.Tasks[i].Owner == "" {
						st.Tasks[i].Owner = ownerName
					}
				}
			}
		}
		return st

	case state.EvReturned:
		st = setEmployee(st, ev.EmployeeID, func(e *state.Employee) {
			e.Sprite = state.SpriteToDesk
			e.Task = ""
		})
		for i := range st.Tasks {
			if st.Tasks[i].ID == ev.TaskID {
				st.Tasks[i].Status = state.TaskDone
			}
		}
		st.Mails = capList(append(append([]state.MailItem(nil), st.Mails...), ev.Mail), mailCap)
		return st

	case state.EvIdleDrift:
		return setEmployee(st, ev.EmployeeID, func(e *state.Employee) {
			e.Sprite = state.SpriteToCoffee
		})

	case state.EvBlocked:
		st = setEmployee(st, ev.EmployeeID, func(e *state.Employee) {
			e.Sprite = state.SpriteAtMailbox
		})
		st.StatusLine = fmt.Sprintf("[blocked] %s", ev.Text)
		return st

	case state.EvTask:
		st.Tasks = upsertTask(st.Tasks, ev.Task)
		return st

	case state.EvMail:
		st.Mails = capList(append(append([]state.MailItem(nil), st.Mails...), ev.Mail), mailCap)
		return st

	case state.EvChatUser:
		st.Chat = capChat(appendChat(st.Chat, ev.Msg))
		return st

	case state.EvChatBoss:
		st.BossThinking = false // a boss turn ends the thinking affordance

		// typingPlaceholder — the Send-sequenced "boss-N" pending bubble.
		// Real bubbles (stream deltas + the pinned final) carry their own
		// stable ID ("bossmsg-"+messageID) and are never placeholders.
		isPlaceholder := func(m state.ChatMsg) bool {
			return m.From == "boss" && m.Pending && strings.HasPrefix(m.ID, "boss-")
		}
		msg := ev.Msg

		// (a) replace-in-place by ID: streaming deltas land on their stable
		// ID and grow the same bubble; the final re-emits that ID with
		// Pending=false; a duplicated completed event is idempotent. The
		// swap is one atomic slice — the chat count never inflates mid-stream.
		if msg.ID != "" {
			// stream deltas always land on the TAIL of the transcript, so
			// sweep backwards first — the hot path is one probe instead of
			// a full-transcript walk. The head sweep below stays as the
			// fallback so the replace/no-replace edge can't change
			// silently; whichever direction finds the ID pins the swap
			// index (IDs are unique by construction — the merge arms keep
			// them so).
			for i := len(st.Chat) - 1; i >= 0; i-- {
				if st.Chat[i].ID == msg.ID {
					next := append([]state.ChatMsg(nil), st.Chat...)
					next[i] = msg
					st.Chat = capChat(next)
					return st
				}
			}
			for i, m := range st.Chat {
				if m.ID == msg.ID {
					next := append([]state.ChatMsg(nil), st.Chat...)
					next[i] = msg
					st.Chat = capChat(next)
					return st
				}
			}
		}

		// (b) a fresh typing placeholder appends as-is; it gets replaced by
		// the FIRST real bubble of its reply cycle (branch below).
		if isPlaceholder(msg) {
			st.Chat = capChat(appendChat(st.Chat, msg))
			return st
		}

		// (c) a new real boss bubble: strip every remaining "boss-N" typing
		// placeholder of the send cycle, then append. The rebuilt slice
		// keeps the transcript's growth curve as capacity (+1 for the new
		// bubble) — an exact-fit make here would force the NEXT appendChat
		// to re-copy the whole transcript on every single bubble.
		rest := make([]state.ChatMsg, 0, cap(st.Chat)+1)
		for _, mgr := range st.Chat {
			if !isPlaceholder(mgr) {
				rest = append(rest, mgr)
			}
		}
		// A clearing event (Pending:false, empty text) strips stale
		// placeholders without appending an empty bubble — the turn ended
		// with no boss text (session.idle / result backstop).
		if !msg.Pending && strings.TrimSpace(msg.Text) == "" {
			st.Chat = capChat(rest)
			return st
		}
		st.Chat = capChat(appendChat(rest, msg))
		return st

	case state.EvChatOffice:
		// concierge answers — "chat-office" events (the parallel backend
		// contract): the SAME replace-in-place mechanics as the boss
		// stream. The pending placeholder ("office-<msgID>") appends once,
		// streaming growth re-emits the same ID, and the completion pin
		// swaps Pending→false in place; a duplicated completion is
		// idempotent. Rides capChat (the 10k fuse) like every chat entry,
		// and never touches boss typing/delegation state.
		msg := ev.Msg
		if msg.ID != "" {
			// tail-first sweep (boss/arm-a mechanics — see EvChatBoss):
			// concierge stream growth also lands on the tail; head sweep
			// kept as the fallback so no match edge can change silently.
			for i := len(st.Chat) - 1; i >= 0; i-- {
				if st.Chat[i].ID == msg.ID {
					next := append([]state.ChatMsg(nil), st.Chat...)
					next[i] = msg
					st.Chat = capChat(next)
					return st
				}
			}
			for i, c := range st.Chat {
				if c.ID == msg.ID {
					next := append([]state.ChatMsg(nil), st.Chat...)
					next[i] = msg
					st.Chat = capChat(next)
					return st
				}
			}
		}
		st.Chat = capChat(appendChat(st.Chat, msg))
		return st

	case state.EvThought:
		{
			// boss thoughts: thinking flag + a chat entry (Kind "think", 10k
			// fuse) keyed by CallID — mid-stream updates REPLACE the entry in place
			// (accumulated text), Done=true is the final update; the model's
			// activeThink set decides streaming vs collapsed at render.
			// employee thoughts: the activity line still records them, and
			// they ALSO merge (one per agent+CallID, accumulated text,
			// Done=true the final update — same stream mechanics as boss)
			// into the agent's own work thread as Kind "wthink": the chat
			// panel renders them inside the thread (dim-italic "thinking ·
			// N lines" rows live, a "· N think" count once collapsed,
			// full body under ctrl+g). Meta carries the tick (␟ separator,
			// same carrier as wtool) for the thread's staleness collapse.
			if ev.EmployeeID != "boss" {
				name := ev.EmployeeName
				if name == "" {
					if e := findEmployee(st, ev.EmployeeID); e != nil {
						name = e.Name
					}
				}
				if name == "" || name == "boss" {
					return st // unknown actor: activity-line only, as before
				}
				toolState := "running"
				if ev.Done {
					toolState = "done"
				}
				id := "wthink-" + name + "-" + ev.CallID
				if ev.CallID == "" {
					id = "wthink-" + name + "-" + nextMsgID()
				}
				entry := state.ChatMsg{
					ID:   id,
					From: name,
					Kind: "wthink",
					Text: ev.Text,
					Meta: toolState + "\x1f" + strconv.Itoa(st.Tick),
					At:   time.Now().UnixMilli(),
				}
				next := append([]state.ChatMsg(nil), st.Chat...)
				merged := false
				// tail-first: the merge target is almost always the most recent
				// entry (the stream updates what it just appended). On a miss
				// the head sweep below runs unchanged, so the merged/no-merge
				// boolean edge is byte-for-byte preserved.
				for i := len(next) - 1; i >= 0; i-- {
					if next[i].Kind == entry.Kind && next[i].ID == entry.ID {
						// birth stamp wins (see below).
						if next[i].At != 0 {
							entry.At = next[i].At
						}
						next[i] = entry
						merged = true
						break
					}
				}
				if !merged {
					for i, msg := range next {
						if msg.Kind == entry.Kind && msg.ID == entry.ID {
							// birth stamp wins: a stream update replaces text/meta
							// in place but NEVER re-stamps At — the first-seen
							// stamp pins the entry's timeline slot (the merged
							// thread sorts by it; re-stamping would swim it).
							if msg.At != 0 {
								entry.At = msg.At
							}
							next[i] = entry
							merged = true
							break
						}
					}
				}
				if !merged {
					next = append(next, entry)
				}
				st.Chat = capChat(next)
				return st
			}
			st.BossThinking = !ev.Done
			id := "think-" + ev.CallID
			if ev.CallID == "" {
				// no id to key on — legacy emitters stay append-only
				id = "think-" + nextMsgID()
			}
			entry := state.ChatMsg{
				ID:   id,
				From: "boss",
				Kind: "think",
				Text: ev.Text,
				Meta: ev.CallID, // renderer reads the CallID back from Meta
				At:   time.Now().UnixMilli(),
			}
			next := append([]state.ChatMsg(nil), st.Chat...)
			merged := false
			// tail-first: boss think deltas merge onto the entry the stream
			// just appended; head sweep below kept as the fallback, so the
			// merged/no-merge edge can't change silently.
			for i := len(next) - 1; i >= 0; i-- {
				if next[i].Kind == "think" && next[i].ID == entry.ID {
					// birth stamp wins (see the wthink merge above).
					if next[i].At != 0 {
						entry.At = next[i].At
					}
					next[i] = entry
					merged = true
					break
				}
			}
			if !merged {
				for i, msg := range next {
					if msg.Kind == "think" && msg.ID == entry.ID {
						// birth stamp wins (see the wthink merge above).
						if msg.At != 0 {
							entry.At = msg.At
						}
						next[i] = entry
						merged = true
						break
					}
				}
			}
			if !merged {
				next = append(next, entry)
			}
			st.Chat = capChat(next)
			return st
		}

	case state.EvTool:
		{
			// tool one-liners merge by CallID: running → done replaces the line.
			// BOSS/primary tools keep the classic inline "tool" Kind;
			// EMPLOYEE tools get Kind "wtool" — the chat panel lifts them
			// out of the flow into the per-agent workers-thread region at
			// the end (P2), so a sub-agent storm can't drown the boss
			// conversation. Their Meta carries the tool state plus the
			// latest activity tick (␟ separator) for the staleness
			// auto-collapse; merging is scoped per agent+CallID.
			name := ev.EmployeeName
			if name == "" {
				name = "boss"
			}
			kind := "tool"
			id := toolEntryID(name, ev.CallID)
			meta := ev.ToolState
			if name != "boss" {
				kind = "wtool"
				meta = ev.ToolState + "\x1f" + strconv.Itoa(st.Tick)
			}
			text := ev.ToolName
			if ev.ToolSummary != "" {
				text += " · " + ev.ToolSummary
			}
			line := state.ChatMsg{
				ID:   id,
				From: name,
				Kind: kind,
				Text: strings.ReplaceAll(text, "\n", " "), // chat rows are one-liners
				Meta: meta,
				At:   time.Now().UnixMilli(),
			}
			merged := false
			next := append([]state.ChatMsg(nil), st.Chat...)
			// tail-first: running → done replaces the line the tool just
			// appended (the tail); head sweep below kept as the fallback so
			// the merged/no-merge edge can't change silently.
			for i := len(next) - 1; i >= 0; i-- {
				if next[i].Kind == line.Kind && next[i].ID == line.ID {
					// birth stamp wins (see the wthink merge above).
					if next[i].At != 0 {
						line.At = next[i].At
					}
					next[i] = line
					merged = true
					break
				}
			}
			if !merged {
				for i, msg := range next {
					if msg.Kind == line.Kind && msg.ID == line.ID {
						// birth stamp wins (see the wthink merge above).
						if msg.At != 0 {
							line.At = msg.At
						}
						next[i] = line
						merged = true
						break
					}
				}
			}
			if !merged {
				next = append(next, line)
			}
			st.Chat = capChat(next)
			return st
		}

	case state.EvQuestion:
		{
			// a resolved boss question MUTATES the original Kind "question"
			// chat entry in place: Meta gains a trailing "␟answered" unit-
			// separator token (the options stay ahead of it) so the panel
			// renders the dim "✓ answered" suffix instead of the hint.
			if ev.ToolState == "resolved" && ev.QuestionID != "" {
				id := "q-" + ev.QuestionID
				// tail-first: a question resolves moments after it surfaced
				// (near the tail); the head sweep below stays as the fallback
				// so the mutate/no-mutate edge can't change silently.
				for i := len(st.Chat) - 1; i >= 0; i-- {
					m := st.Chat[i]
					if m.Kind == "question" && m.ID == id &&
						!strings.HasSuffix(m.Meta, "\x1fanswered") {
						next := append([]state.ChatMsg(nil), st.Chat...)
						next[i].Meta = m.Meta + "\x1fanswered"
						st.Chat = next
						return st
					}
				}
				for i, m := range st.Chat {
					if m.Kind == "question" && m.ID == id &&
						!strings.HasSuffix(m.Meta, "\x1fanswered") {
						next := append([]state.ChatMsg(nil), st.Chat...)
						next[i].Meta = m.Meta + "\x1fanswered"
						st.Chat = next
						break
					}
				}
				return st
			}
			// boss questions: Kind "question" chat entry (yellow "boss asks ›").
			// Employee questions are activity-line only (describeEvent), like
			// employee thoughts — the deep-work stream belongs to the boss.
			if ev.EmployeeName != "" && ev.EmployeeName != "boss" {
				return st
			}
			id := "q-" + ev.QuestionID
			if ev.QuestionID == "" {
				id = "q-" + nextMsgID()
			}
			st.Chat = capChat(appendChat(st.Chat, state.ChatMsg{
				ID:   id,
				From: "boss",
				Kind: "question",
				Text: ev.Text,
				// options ride in Meta for the renderer ("a | b | c")
				Meta: ev.ToolSummary,
				At:   time.Now().UnixMilli(),
			}))
			return st
		}

	case state.EvFileDiff:
		{
			name := ev.EmployeeName
			if name == "" {
				name = "boss"
			}
			// A per-CALL worker diff (the backend lifted the patch off one
			// completed edit/write tool part) rides Kind "wdiff" INSIDE the
			// agent's thread, adjacent to its [tool] row — not the flat
			// flow. Merge-by-ID like the wtool entries (repeated completed
			// frames replace in place, birth stamp preserved); at BIRTH the
			// entry inserts right AFTER its tool call's own row when that
			// row already exists, so the thread reads
			// "[tool] Edit x ✓ · +A -D" then "↳ diff · x +A -D" in natural
			// order. Boss/file-level diffs (no CallID, or the boss's own
			// tools) keep the classic Kind "diff" flow below unchanged.
			if ev.CallID != "" && name != "boss" {
				line := state.ChatMsg{
					ID:   "wdiff-" + name + "-" + ev.CallID,
					From: name,
					Kind: "wdiff",
					Text: ev.DiffBody,
					// same Meta carrier as the flat diff: path ␟ +adds ␟ -dels
					Meta: fmt.Sprintf("%s\x1f+%d\x1f-%d", ev.DiffPath, ev.DiffAdd, ev.DiffDel),
					At:   time.Now().UnixMilli(),
				}
				toolID := "wtool-" + name + "-" + ev.CallID
				next := append([]state.ChatMsg(nil), st.Chat...)
				merged := false
				// tail-first: a wdiff is born right beside its tool row at the
				// tail of the transcript. Both find-by-ID loops sweep backwards
				// first, with the head sweeps kept verbatim as fallbacks so the
				// merged/inserted boolean edges can't change silently.
				for i := len(next) - 1; i >= 0; i-- {
					if next[i].Kind == line.Kind && next[i].ID == line.ID {
						// birth stamp wins (see the wtool merge above).
						if next[i].At != 0 {
							line.At = next[i].At
						}
						next[i] = line
						merged = true
						break
					}
				}
				if !merged {
					for i, msg := range next {
						if msg.Kind == line.Kind && msg.ID == line.ID {
							// birth stamp wins (see the wtool merge above).
							if msg.At != 0 {
								line.At = msg.At
							}
							next[i] = line
							merged = true
							break
						}
					}
				}
				if !merged {
					inserted := false
					for i := len(next) - 1; i >= 0; i-- {
						if next[i].Kind == "wtool" && next[i].ID == toolID {
							// fresh tail first — the insert never aliases
							tail := append([]state.ChatMsg{line}, next[i+1:]...)
							next = append(next[:i+1], tail...)
							inserted = true
							break
						}
					}
					if !inserted {
						for i, msg := range next {
							if msg.Kind == "wtool" && msg.ID == toolID {
								// fresh tail first — the insert never aliases
								tail := append([]state.ChatMsg{line}, next[i+1:]...)
								next = append(next[:i+1], tail...)
								inserted = true
								break
							}
						}
					}
					if !inserted {
						next = append(next, line)
					}
				}
				st.Chat = capChat(next)
				return st
			}
			st.Chat = capChat(appendChat(st.Chat, state.ChatMsg{
				ID:   "diff-" + nextMsgID(),
				From: name,
				Kind: "diff",
				Text: ev.DiffBody,
				// Meta carrier for the collapsed header:
				// path ␟ +adds ␟ -dels (unit separator; panels parses it back)
				Meta: fmt.Sprintf("%s\x1f+%d\x1f-%d", ev.DiffPath, ev.DiffAdd, ev.DiffDel),
				At:   time.Now().UnixMilli(),
			}))
			return st
		}

	case state.EvBubble:
		ttl := ev.TTL
		if ttl == 0 {
			ttl = 40
		}
		bubble := state.SpeechBubble{
			ID:         fmt.Sprintf("bbl-%d-%05d", st.Tick, rand.Intn(100000)),
			EmployeeID: ev.EmployeeID,
			Text:       ev.Text,
			UntilTick:  st.Tick + ttl,
		}
		st.Bubbles = capList(append(append([]state.SpeechBubble(nil), st.Bubbles...), bubble), bubbleCap)
		return st

	case state.EvStatus:
		st.StatusLine = ev.Text
		// Backend-name latch (string-marker contract): every transport's
		// boot emits "[theboringfloor] backend: <name>" first, and the
		// /backend swap line ("… <old> → <new> (turn #N archived)")
		// re-latches through the same grammar — the topbar renders
		// st.BackendName between mode and agents.
		if name, ok := backendNameFromStatus(ev.Text); ok {
			st.BackendName = name
		}
		return st

	case state.EvOffline:
		// Connectivity watcher says down — badge + status land together.
		// The backend pairs this with its own EvStatus right after ("[theboringfloor]
		// offline — office waiting for internet…"); this text is the fallback
		// for a kind fired without a paired status.
		st.Offline = true
		st.StatusLine = "[office] OFFLINE — waiting for internet…"
		return st

	case state.EvOnline:
		st.Offline = false
		st.StatusLine = "[office] back online"
		return st

	case state.EvUsage:
		// Real opencode assistant-message counters (EvUsage carries
		// per-message DELTAS — see state.go); plain accumulation into the
		// conversation totals the status bar renders. Repeated frames can
		// never double-count: the backend ships growth only.
		st.TokensIn += ev.TokensIn
		st.TokensOut += ev.TokensOut
		st.CostUSD += ev.CostUSD
		// Prompt-cache counters ride the same += path (informational only;
		// CostUSD above already prices them).
		st.TokensCacheRead += ev.TokensCacheRead
		st.TokensCacheWrite += ev.TokensCacheWrite
		return st
	}
	return st
}

func officeStateWithEmployees(st state.OfficeState, emps []state.Employee) state.OfficeState {
	st.Employees = emps
	return st
}

func hasPendingBoss(st state.OfficeState) bool {
	for _, m := range st.Chat {
		if m.From == "boss" && m.Pending {
			return true
		}
	}
	return false
}

// --- chat send path ---------------------------------------------------------

var msgSeq atomic.Int64

func nextMsgID() string {
	return fmt.Sprintf("c%d", msgSeq.Add(1))
}

// --- slash commands (local, never sent to the backend) ---------------------

// slashHelp is the /help notice body (office-rendered, dim).
const slashHelp = `commands:
  /help              this list
  /clear             empty the chat
  /theme <name>      switch theme (persists)
  /themes            list themes
  /power [mode]      show/set the power governor (auto|performance|saver)
  /notify [on|off]   OS desktop notifications while unfocused (persists)
  /model [ref]       show/set the boss model (provider/model)
  /thinking on|off   show/hide thinking blocks
  /tools on|off      show/hide tool one-liners
  /diffs on|off      expand/collapse file diffs (ctrl+d toggles)
  /compact on|off    compact layout this session (narrow sidebar, short tabs)
  /mode normal|compact  layout mode (persists)
  /wide <n>          sidebar width 26..100 (0 = default 80, persists)
  /zen               fullscreen floor, any key exits (transient)
  /focus floor       alias of /zen
  /queue             show the backlog (numbered items batched on flush)
  /queue clear       drop all queued backlog items
  /route             force-dispatch the backlog now (bypasses the busy gate)
  /stop              abort current work (boss + workers); queue sends next turn
  @<file>            attach file (popover picker) · cmd+v pastes images (ctrl+v too)
   /perm              re-open an esc'd permission prompt
   /question          re-open a deferred boss question
   /bypass            toggle bypass permissions: agents run tools + browser
                      actions WITHOUT asking (this session only · confirm
                      to arm · respawns the backend)
   /images [mode]     boss-turn image previews: auto|ascii|off (persists)
  /open <url>        open a page in the browser tab (file:// or localhost)
  /new               fresh office (previous transcript archived on disk)
  /session           pick a past session to resume live (fallback prints
                     the id + path; boot flag -s|--session <id> pins one)
  /backend [name]    show the LLM transport; /backend opencode|claudecode
                     swaps mid-flight (idle office only · persists)
  /status            office status
  /mcp [reconnect x] show MCP servers; reconnect one by name
  /memory [filter]   the office ledger — completed dispatches, newest first
  /quit              exit theboringoffice`

// applySlash dispatches one slash command. Slash input never echoes as
// chat-user; every outcome surfaces as a From "office" chat notice.
func (m *Model) applySlash(input string) tea.Cmd {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return nil
	}
	cmd := strings.ToLower(fields[0])
	switch cmd {
	case "/help":
		m.notice(slashHelp)
	case "/mcp":
		// MCP console: status list from the backend's /mcp surface, or a
		// reconnect of one server (the live route POSTs /mcp/{name}/connect,
		// older serves may reject with a wrapped 404 — that error surfaces
		// as the notice). Both hops are async so the input never stalls.
		current := m.currentBackend
		if len(fields) >= 2 && fields[1] == "reconnect" {
			if len(fields) < 3 {
				m.noticeErr("/mcp: usage /mcp reconnect <name>")
				return nil
			}
			name := fields[2]
			m.notice("mcp: reconnecting " + name + "…")
			return func() tea.Msg {
				var sv []state.MCPServer
				var reconnOK bool
				err := current.lease(func(b state.Backend) error {
					if err := b.ReconnectMCP(name); err != nil {
						return err
					}
					reconnOK = true
					var e error
					sv, e = b.MCPServers()
					return e
				})
				msg := mcpStatusMsg{servers: sv, err: err}
				if reconnOK {
					msg.reconnected = name
				}
				return msg
			}
		}
		return func() tea.Msg {
			var sv []state.MCPServer
			err := current.lease(func(b state.Backend) error {
				var e error
				sv, e = b.MCPServers()
				return e
			})
			return mcpStatusMsg{servers: sv, err: err}
		}
	case "/clear":
		m.st.Chat = nil
		if m.chat != nil {
			m.chat.ClearAttachments() // staged chips die with the visible chat
		}
		m.tabs.SetState(m.st)
	case "/theme":
		if len(fields) < 2 {
			m.noticeErr("/theme: usage /theme <name>  (" + strings.Join(chrome.ThemeNames(), ", ") + ")")
			return nil
		}
		name := fields[1]
		if !chrome.SetTheme(name) {
			m.noticeErr(fmt.Sprintf("/theme: unknown theme %q (/themes)", name))
			return nil
		}
		_ = chrome.PersistTheme() // best effort
		office.SetTheme(name)     // floor palette follows chrome
		m.chat.RefreshTheme()
		m.tabs.SetState(m.st)
		m.notice("theme → " + chrome.CurrentTheme().Name)
	case "/themes":
		m.notice("themes: " + strings.Join(chrome.ThemeNames(), "  ") +
			"  (current: " + chrome.CurrentTheme().Name + ")")
	case "/power":
		if len(fields) < 2 {
			m.notice(fmt.Sprintf("power: %s (%s) · current tick %s — /power auto|performance|saver",
				PowerMode(m.cfg), powerDescribe(PowerMode(m.cfg)), m.currentTick()))
			return nil
		}
		mode := config.PowerMode(strings.ToLower(fields[1]))
		switch mode {
		case config.PowerAuto, config.PowerPerformance, config.PowerSaver:
		default:
			m.noticeErr(fmt.Sprintf("/power: unknown mode %q (auto|performance|saver)", fields[1]))
			return nil
		}
		m.cfg.UI.Power = mode
		m.notice(fmt.Sprintf("power → %s (%s) · current tick %s · %s",
			mode, powerDescribe(mode), m.currentTick(), m.persistCfg()))
	case "/notify":
		if len(fields) < 2 {
			cur := m.cfg.UI.Notifications
			if cur == "" {
				cur = "on"
			}
			m.notice(fmt.Sprintf("notifications %s (OS desktop pings while the terminal is unfocused) — /notify on|off", cur))
			return nil
		}
		mode := strings.ToLower(fields[1])
		if mode != "on" && mode != "off" {
			m.noticeErr(fmt.Sprintf("/notify: unknown mode %q (on|off)", fields[1]))
			return nil
		}
		m.cfg.UI.Notifications = mode
		// live-set the injected engine too (type-assert seam — headless
		// record-stubs may simply not accept a mode and stay honored by the
		// config gate above).
		if sm, ok := m.notifyBus.(interface{ SetMode(string) }); ok {
			sm.SetMode(mode)
		}
		m.notice(fmt.Sprintf("notifications → %s · %s", mode, m.persistCfg()))
	case "/model":
		if len(fields) < 2 {
			// bare /model: open the interactive picker when the backend
			// lists models via the additive seam (model_picker.go); every
			// seam-absent/failed-listing path lands the classic hint note.
			return m.openModelPicker()
		}
		ref := fields[1]
		if !strings.Contains(ref, "/") {
			m.noticeErr("/model: usage /model provider/model (e.g. anthropic/claude-haiku-4-5)")
			return nil
		}
		m.cfg.Boss.Model = config.ModelRef(ref)
		m.notice(fmt.Sprintf("boss model → %s (the backend honors it on the next send) · %s", ref, m.persistCfg()))
	case "/thinking":
		m.applyToggle("/thinking", fields, func(on bool) {
			m.chat.SetShowThinking(on)
		})
	case "/tools":
		m.applyToggle("/tools", fields, func(on bool) {
			m.chat.SetShowTools(on)
		})
	case "/diffs":
		m.applyToggle("/diffs", fields, func(on bool) {
			m.chat.SetDiffsExpanded(on)
		})
	case "/images":
		// boss-turn image previews: bare form reports the posture (incl.
		// the detect layer's lane read), an argument cycles AND persists
		// (same write-through as /power — /tools-style toggles stay
		// session-only, this one is a member preference).
		if len(fields) < 2 {
			m.notice(fmt.Sprintf("images %s (detected lane: %s) — /images auto|ascii|off",
				m.cfg.UI.Images, panels.DetectImageSupport()))
			return nil
		}
		mode := strings.ToLower(fields[1])
		if !config.ValidImagesMode(mode) {
			m.noticeErr(fmt.Sprintf("/images: unknown mode %q (auto|ascii|off)", fields[1]))
			return nil
		}
		m.cfg.UI.Images = mode
		m.onImagesLaneChanged() // additive seam: future lanes re-probe there
		m.notice(fmt.Sprintf("images → %s · %s", mode, m.persistCfg()))
	case "/compact":
		// live toggle — session override only (/mode persists the choice)
		if len(fields) < 2 || (fields[1] != "on" && fields[1] != "off") {
			m.noticeErr("/compact: usage /compact on|off  (/mode normal|compact persists)")
			return nil
		}
		if fields[1] == "on" {
			m.compactLive = 1
		} else {
			m.compactLive = 2
		}
		m.applyLayout()
		m.notice(fmt.Sprintf("compact → %s (this session · /mode %s persists)",
			fields[1], fields[1]))
	case "/mode":
		if len(fields) < 2 || (fields[1] != "normal" && fields[1] != "compact") {
			m.noticeErr("/mode: usage /mode normal|compact")
			return nil
		}
		m.cfg.UI.Compact = fields[1] == "compact"
		m.compactLive = 0 // cfg is the source again
		m.applyLayout()
		m.notice(fmt.Sprintf("layout mode → %s · %s", fields[1], m.persistCfg()))
	case "/wide":
		if len(fields) < 2 {
			m.noticeErr(fmt.Sprintf("/wide: usage /wide <%d..%d> (0 = default %d)", sidebarMin, sidebarMax, defaultSidebarW))
			return nil
		}
		n, err := strconv.Atoi(fields[1])
		if err != nil {
			m.noticeErr(fmt.Sprintf("/wide: %q is not a number (26..100, 0 = default)", fields[1]))
			return nil
		}
		if n < 0 {
			n = 0
		}
		if n > sidebarMax {
			m.notice(fmt.Sprintf("/wide: %d over the %d-col cap — clamped", n, sidebarMax))
			n = sidebarMax
		} else if n != 0 && n < sidebarMin {
			m.notice(fmt.Sprintf("/wide: %d under the %d-col floor — clamped", n, sidebarMin))
			n = sidebarMin
		}
		m.cfg.UI.SidebarWidth = n
		m.applyLayout()
		shown := n
		if n == 0 {
			shown = defaultSidebarW
			if m.compact() {
				shown = compactSidebarW
			}
		}
		m.notice(fmt.Sprintf("sidebar → %d cols · %s", shown, m.persistCfg()))
	case "/zen":
		// transient focus session — intentionally NOT persisted
		m.zen = true
	case "/focus":
		if len(fields) < 2 || fields[1] != "floor" {
			m.noticeErr("/focus: usage /focus floor  (alias of /zen)")
			return nil
		}
		m.zen = true
	case "/queue":
		if len(fields) >= 2 {
			if fields[1] == "clear" {
				// dropping the items also drops their sends — the temp
				// dirs go now (no flush is coming for them).
				cleanupEntries(m.queue)
				m.queue = nil
				if m.chat != nil {
					m.chat.SetQueueLen(0)
				}
				m.notice("backlog cleared")
			} else {
				m.noticeErr("/queue: usage /queue | /queue clear")
			}
			return nil
		}
		if len(m.queue) == 0 {
			m.notice("backlog empty — type while the boss is typing to queue an item")
			return nil
		}
		var sb strings.Builder
		if len(m.queue) > 1 {
			fmt.Fprintf(&sb, "%d queued (will batch-dispatch on flush):", len(m.queue))
		} else {
			fmt.Fprintf(&sb, "1 queued (sends on flush):")
		}
		for i, e := range m.queue {
			fmt.Fprintf(&sb, "\n  %d. %s", i+1, e.text)
			if n := len(e.atts); n > 0 {
				// same machine suffix the batch prompt gets
				fmt.Fprintf(&sb, " 📎%d", n)
			}
		}
		m.notice(sb.String())
	case "/route":
		// force the backlog out NOW, bypassing the busy gate. A parked
		// question hold stays blocking — a chat Send is what deadlocks the
		// parked loop (the answer must go through AnswerQuestion first).
		if m.questionParked || m.question != nil || m.questionEscd != nil {
			m.notice("/route: boss is waiting on your answer — answer the question first (/question)")
			return nil
		}
		if len(m.queue) == 0 {
			m.notice("nothing queued — type while the boss is typing to enqueue")
			return nil
		}
		return m.dispatchQueued(true)
	case "/stop":
		// abort the primary + every live child session, then unwind the
		// in-flight UI (placeholders collapse, tools ✗ aborted, threads ✗
		// stopped). The queue is untouched — it sends on the next turn.
		// The abort RPC rides the returned cmd (never the UI goroutine);
		// the unwind + statusline inside are already done.
		return m.stopWork()
	case "/bypass":
		// session-scoped bypass-permissions mode: enable asks the confirm
		// popover, disable is instant; both respawn the transport.
		return m.applyBypassSlash()
	case "/perm":
		// re-open the most recent esc'd ask: it jumps the queue to the
		// front (the popover displays it next). Nothing esc'd means either
		// a prompt is already open or there's genuinely nothing pending.
		if len(m.permQ.pending) == 0 && len(m.permQ.escd) == 0 {
			m.notice("no pending permission (/perm re-opens an esc'd prompt)")
			return nil
		}
		if len(m.permQ.escd) == 0 {
			m.notice("permission prompt already open — answer it or esc to defer")
			return nil
		}
		p := m.permQ.escd[len(m.permQ.escd)-1]
		m.permQ.escd = m.permQ.escd[:len(m.permQ.escd)-1]
		m.permQ.pending = append([]*permPrompt{p}, m.permQ.pending...)
		m.chat.SetPermission(m.permQ.view())
	case "/question":
		if m.question != nil {
			m.notice("boss question is open — answer it or esc to defer")
			return nil
		}
		if m.questionEscd == nil || len(m.questionEscd.IDs) == 0 {
			m.notice("no deferred boss question (/question re-opens a deferred question)")
			return nil
		}
		m.question = m.questionEscd
		m.questionEscd = nil
		// resume the wizard at the FIRST page whose answer slot is still
		// empty — pages already answered keep their recorded answers
		// (re-opening never forces re-answering).
		for i, a := range m.question.Answers {
			if len(a) == 0 {
				m.question.Cursor = i
				break
			}
		}
		m.chat.SetQuestion(m.questionView(m.question))
	case "/new":
		m.btwSaved = nil // /new abandons any btw session
		m.btwHiddenSnap = nil
		m.btwPinMsgID = ""
		m.newOffice()    // sessions.go — clear surfaces + fresh "theboringoffice office"
	case "/backend":
		// install-seeded brain.json backend.name's in-app twin: show the
		// active transport or swap it mid-flight (idle-office gate inside).
		return m.applyBackendSlash(fields)
	case "/session":
		// Interactive picker of the server's ROOT sessions (accept
		// re-anchors the office LIVE, esc cancels); a backend without the
		// listing seam — or a failed listing — falls back to the static
		// summary (current id + session.json path + the boot flag) with a
		// dim picker-unavailable note. See session_picker.go.
		return m.openSessionPicker()
	case "/status":
		var pend, doing, done int
		for _, t := range m.st.Tasks {
			switch t.Status {
			case state.TaskPending:
				pend++
			case state.TaskInProgress:
				doing++
			case state.TaskDone:
				done++
			}
		}
		m.notice(fmt.Sprintf("mode %s · theme %s · power %s · agents %d · board %d/%d/%d\n%s",
			m.st.Mode, chrome.CurrentTheme().Name, PowerMode(m.cfg), len(m.st.Employees),
			pend, doing, done, m.st.StatusLine))
	case "/memory":
		// Office memory: the project ledger (.opencode/office-ledger.md),
		// newest first; an optional case-fold substring narrows rows over
		// title/files. One small bounded file read on the spot — no
		// backend hop, nothing queued — stream-safe like /queue (the
		// outcome rides the same office-notice seam as every slash).
		m.notice(m.memoryBody(strings.Join(fields[1:], " ")))
	case "/open":
		// browser tab: parse the arg, jump to the pane, load — the dim
		// completion notice lands with the fetch verdict (app/browser.go).
		return m.applyOpenSlash(fields)
	case "/btw":
		if m.btwSaved != nil {
			m.noticeErr("already in a btw session — esc or /done to return first")
			return nil
		}
		if m.btwHidden() {
			return m.resumeBtw()
		}
		if hasPendingBoss(m.st) {
			m.noticeErr("/btw: boss is mid-turn — wait for it to finish or /stop first")
			return nil
		}
		// Save current state.
		var savedPrimary string
		if pb, ok := m.backend.(primarySeamBackend); ok {
			savedPrimary = pb.PrimaryID()
		}
		m.btwSaved = &btwSnapshot{
			chat:      append([]state.ChatMsg(nil), m.st.Chat...),
			tasks:     append([]state.BoardTask(nil), m.st.Tasks...),
			mails:     append([]state.MailItem(nil), m.st.Mails...),
			primaryID: savedPrimary,
		}
		// Clear surfaces (same pattern as newOffice).
		m.st.Chat = nil
		m.st.Tasks = nil
		m.st.Mails = nil
		m.st.Bubbles = nil
		m.st.BossThinking = false
		m.st.BossDelegating = false
		m.resetPager()
		if m.chat != nil {
			m.chat.ClearAttachments()
		}
		// Mint fresh backend session — async so the 30s teardown never
		// parks the UI goroutine.
		if ob, ok := m.backend.(officeSpawnBackend); ok && m.st.Mode == state.ModeLive {
			trailing := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/btw"))
			tb, hasTB := m.team()
			m.tabs.SetState(m.st)
			m.notice("starting btw session…")
			return func() tea.Msg {
				if hasTB {
					_ = tb.ResetPrimary(true)
				}
				_, err := ob.NewOffice()
				return btwOfficeMsg{err: err, trailing: trailing}
			}
		}
		m.tabs.SetState(m.st)
		m.notice("btw session — esc or /done to return")
	case "/done":
		return m.exitBtw()
	case "/quit":
		m.persistOfficeSession(true) // final SYNC snapshot (live only)
		m.closeTerminal()
		m.closeBrowser()
		return tea.Quit
	default:
		m.noticeErr(fmt.Sprintf("/ %s: no such command (/help)", strings.TrimPrefix(cmd, "/")))
	}
	return nil
}

func (m *Model) exitBtw() tea.Cmd {
	if m.btwSaved == nil {
		m.noticeErr("not in a btw session (/btw starts one)")
		return nil
	}
	if hasPendingBoss(m.st) {
		m.noticeErr("/done: boss is mid-turn — wait for it to finish or /stop first")
		return nil
	}
	saved := m.btwSaved
	m.btwSaved = nil
	m.btwHiddenSnap = nil
	m.btwPinMsgID = ""
	m.st.Chat = saved.chat
	m.st.Tasks = saved.tasks
	m.st.Mails = saved.mails
	m.st.Bubbles = nil
	m.st.BossThinking = false
	m.st.BossDelegating = false
	m.resetPager()
	if m.chat != nil {
		m.chat.ClearAttachments()
	}
	m.tabs.SetState(m.st)
	if saved.primaryID != "" {
		if sb, ok := m.backend.(btwSwapBackend); ok {
			m.notice("returning from btw…")
			return func() tea.Msg {
				err := sb.SwapPrimary(saved.primaryID)
				return doneOfficeMsg{err: err}
			}
		}
	}
	m.notice("back from btw")
	return nil
}

// btwHidden reports whether Esc has hidden a side session behind its pinned
// main-chat bubble. It keeps the hidden-state check at the call sites readable.
func (m *Model) btwHidden() bool {
	return m.btwHiddenSnap != nil
}

func (m *Model) hideBtw() tea.Cmd {
	if m.btwSaved == nil {
		m.noticeErr("not in a btw session (/btw starts one)")
		return nil
	}
	if hasPendingBoss(m.st) {
		m.noticeErr("/btw: boss is mid-turn — wait for it to finish or /stop first")
		return nil
	}

	var hiddenPrimary string
	if pb, ok := m.backend.(primarySeamBackend); ok {
		hiddenPrimary = pb.PrimaryID()
	}
	m.btwHiddenSnap = &btwSnapshot{
		chat:      append([]state.ChatMsg(nil), m.st.Chat...),
		tasks:     append([]state.BoardTask(nil), m.st.Tasks...),
		mails:     append([]state.MailItem(nil), m.st.Mails...),
		primaryID: hiddenPrimary,
	}

	saved := m.btwSaved
	m.btwSaved = nil
	m.st.Chat = saved.chat
	m.st.Tasks = saved.tasks
	m.st.Mails = saved.mails
	m.st.Bubbles = nil
	m.st.BossThinking = false
	m.st.BossDelegating = false
	m.resetPager()
	if m.chat != nil {
		m.chat.ClearAttachments()
	}
	m.btwPinMsgID = nextMsgID()
	m.st.Chat = append(m.st.Chat, state.ChatMsg{
		ID:   m.btwPinMsgID,
		From: "office",
		Meta: "btw-pin",
		Text: "btw session hidden — click to reopen",
	})
	m.tabs.SetState(m.st)
	if saved.primaryID != "" {
		if sb, ok := m.backend.(btwSwapBackend); ok {
			return func() tea.Msg {
				err := sb.SwapPrimary(saved.primaryID)
				return doneOfficeMsg{err: err}
			}
		}
	}
	return nil
}

func (m *Model) resumeBtw() tea.Cmd {
	if m.btwHiddenSnap == nil {
		m.noticeErr("no hidden btw session (/btw starts one)")
		return nil
	}
	if hasPendingBoss(m.st) {
		m.noticeErr("/btw: boss is mid-turn — wait for it to finish or /stop first")
		return nil
	}

	var savedPrimary string
	if pb, ok := m.backend.(primarySeamBackend); ok {
		savedPrimary = pb.PrimaryID()
	}
	m.btwSaved = &btwSnapshot{
		chat:      append([]state.ChatMsg(nil), m.st.Chat...),
		tasks:     append([]state.BoardTask(nil), m.st.Tasks...),
		mails:     append([]state.MailItem(nil), m.st.Mails...),
		primaryID: savedPrimary,
	}
	if m.btwPinMsgID != "" {
		chat := m.btwSaved.chat[:0]
		for _, msg := range m.btwSaved.chat {
			if msg.ID != m.btwPinMsgID {
				chat = append(chat, msg)
			}
		}
		m.btwSaved.chat = chat
	}

	hidden := m.btwHiddenSnap
	m.btwHiddenSnap = nil
	m.btwPinMsgID = ""
	m.st.Chat = hidden.chat
	m.st.Tasks = hidden.tasks
	m.st.Mails = hidden.mails
	m.st.Bubbles = nil
	m.st.BossThinking = false
	m.st.BossDelegating = false
	m.resetPager()
	if m.chat != nil {
		m.chat.ClearAttachments()
	}
	m.tabs.SetState(m.st)
	if hidden.primaryID != "" {
		if sb, ok := m.backend.(btwSwapBackend); ok {
			m.notice("resuming btw…")
			return func() tea.Msg {
				err := sb.SwapPrimary(hidden.primaryID)
				return doneOfficeMsg{err: err}
			}
		}
	}
	m.notice("btw session — esc or /done to return")
	return nil
}

// applyToggle parses "on|off" for a two-state slash command.
func (m *Model) applyToggle(name string, fields []string, set func(bool)) {
	if len(fields) < 2 || (fields[1] != "on" && fields[1] != "off") {
		m.noticeErr(name + ": usage " + name + " on|off")
		return
	}
	on := fields[1] == "on"
	set(on)
	stateWord := "on"
	if !on {
		stateWord = "off (hidden)"
	}
	m.tabs.SetState(m.st)
	m.notice(name + " → " + stateWord)
}

// --- office memory (/memory + the [memory] activity breadcrumb) ------------
//
// The member-facing memory surfaces. The source of truth TODAY is the
// project ledger the ledger-core dev appends on every completed dispatch —
// <dir>/.opencode/office-ledger.md, newest-first records:
//
//	### 2026-08-25 · <title> — <worker> (<role>) · <verdict>
//	files: internal/app/model.go, internal/app/memory_test.go
//	verify: go test ./internal/app/ -run Memory ✓
//
// /memory reads the file FRESH on every invocation (a boot always sees the
// dispatches that finished before it, across restarts — SessionFile.Ledger
// stays an optional later seam; nothing about this surface needs it). A
// missing/partial/empty ledger degrades OPEN: the honest dim notice, never
// an error row and never invented rows.

// ledgerPath — the project ledger's location for one working directory.
func ledgerPath(dir string) string {
	return filepath.Join(dir, ".opencode", "office-ledger.md")
}

// memoryDir — the directory /memory reads: the office's working directory,
// falling back to the process wd exactly like projinfo does (harness/demo
// models run with sessDir "").
func (m *Model) memoryDir() string {
	if m.sessDir != "" {
		return m.sessDir
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

// memoryLaneBackend — THE additive probe seam the header reads (the
// ledger-core's liveBackend.MemoryLane: "OK" while the agentmemory board
// lane probed live, "file-only" otherwise; deliberately NOT on
// state.Backend — harness stubs never implement it → degrade-open).
type memoryLaneBackend interface {
	MemoryLane() string
}

// memoryProbeState — the header's state word: the backend's MemoryLane
// seam when implemented ("OK" surfaces as the brief's "agentmemory OK"),
// else the boot-line latch (applyEventCore's EvStatus hook), else
// "file-only". Never a guess beyond what the office observed.
func (m *Model) memoryProbeState() string {
	if pb, ok := m.backend.(memoryLaneBackend); ok {
		if pb.MemoryLane() == "OK" {
			return "agentmemory OK"
		}
		return "file-only"
	}
	if m.agentmemoryOK {
		return "agentmemory OK"
	}
	return "file-only"
}

// officeMemoryEntry — one parsed ledger record: the
// "### YYYY-MM-DD · title — worker (role) · verdict" header line plus the
// optional "files:" body line and the "verify:" proof digest line. File
// order is kept (the ledger is written newest-first).
type officeMemoryEntry struct {
	date    string
	title   string
	worker  string
	role    string
	verdict string // normalized: "✓ done" | "✗ <word>" | raw when unclassifiable
	files   []string
	verify  string
}

// parseLedgerHead parses one ledger header minus its "### " prefix.
// nil = malformed (the caller counts the row dropped — the ledger keeps
// showing whatever DID parse).
func parseLedgerHead(head string) *officeMemoryEntry {
	// " · " parts: the date leads, the verdict trails; the title itself
	// may contain " · " (titles are prose), so the middle rejoins.
	parts := strings.Split(head, " · ")
	if len(parts) < 3 {
		return nil
	}
	date := strings.TrimSpace(parts[0])
	verdict := normalizeMemoryVerdict(parts[len(parts)-1])
	mid := strings.TrimSpace(strings.Join(parts[1:len(parts)-1], " · "))
	// the LAST " — " separates title from "worker (role)": the record
	// contract, without it the row is not a record.
	i := strings.LastIndex(mid, " — ")
	if i < 0 {
		return nil
	}
	title := strings.TrimSpace(mid[:i])
	who := strings.TrimSpace(mid[i+len(" — "):])
	worker, role := who, ""
	if j := strings.Index(who, " ("); j >= 0 && strings.HasSuffix(who, ")") {
		worker = strings.TrimSpace(who[:j])
		role = strings.TrimSpace(who[j+len(" (") : len(who)-1])
	}
	if date == "" || title == "" || worker == "" || verdict == "" {
		return nil
	}
	return &officeMemoryEntry{date: date, title: title, worker: worker, role: role, verdict: verdict}
}

// ledgerHeadShaped mirrors the ledger writer's block-open contract
// ("### YYYY-MM-DD · …", internal/backend/ledger.go's ledgerBlockStartRe):
// only those headings open a record — any other "### " line is prose
// (the preamble, a member's hand note) and never counts as a dropped row.
func ledgerHeadShaped(line string) bool {
	if !strings.HasPrefix(line, "### ") {
		return false
	}
	head := line[len("### "):]
	if len(head) < len("2006-01-02")+3 || !strings.HasPrefix(head[10:], " · ") {
		return false
	}
	for i := 0; i < 10; i++ {
		c := head[i]
		switch i {
		case 4, 7:
			if c != '-' {
				return false
			}
		default:
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// parseOfficeLedger parses the ledger body. dropped counts entry-SHAPED
// headers that failed the record grammar (rendered as ONE dim note
// afterwards); prose lines (the file's own preamble, non-shaped headings)
// are tolerated, and body lines belong to the entry above them — the
// writer bullets them ("- files: a, b"), a bare "files:" from a
// hand-rolled record parses too.
func parseOfficeLedger(body string) (entries []officeMemoryEntry, dropped int) {
	var cur *officeMemoryEntry
	flush := func() {
		if cur != nil {
			entries = append(entries, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(body, "\n") {
		if ledgerHeadShaped(line) {
			flush()
			cur = parseLedgerHead(strings.TrimSpace(strings.TrimPrefix(line, "### ")))
			if cur == nil {
				dropped++
			}
			continue
		}
		if strings.HasPrefix(line, "### ") || cur == nil {
			continue // non-entry headings + pre-first-entry body: prose
		}
		t := strings.TrimPrefix(strings.TrimSpace(line), "- ")
		switch {
		case strings.HasPrefix(t, "files:"):
			if rest := strings.TrimSpace(t[len("files:"):]); rest != "" && rest != "(none)" {
				for _, f := range strings.Split(rest, ",") {
					if f = strings.TrimSpace(f); f != "" {
						cur.files = append(cur.files, f)
					}
				}
			}
		case strings.HasPrefix(t, "verify:"):
			if rest := strings.TrimSpace(t[len("verify:"):]); rest != "(none)" {
				cur.verify = rest
			}
		}
	}
	flush()
	return entries, dropped
}

// normalizeMemoryVerdict folds a ledger verdict into the office's glyph
// contract (the terminal's own ✓/✗ from the thread rendering): done-ish →
// "✓ done", issues/failed/partial → "✗ <word>"; anything else renders as
// written. The writer wraps the verdict in backticks (`done`) — stripped.
func normalizeMemoryVerdict(v string) string {
	t := strings.Trim(strings.TrimSpace(v), "` ")
	l := strings.ToLower(t)
	switch {
	case strings.Contains(l, "✗") || strings.Contains(l, "issue") || strings.Contains(l, "partial") || strings.HasPrefix(l, "fail") || strings.HasPrefix(l, "stopped"):
		w := strings.TrimSpace(strings.ReplaceAll(t, "✗", ""))
		if w == "" {
			w = "issues"
		}
		return "✗ " + w
	case strings.Contains(l, "✓") || strings.Contains(l, "done") || strings.Contains(l, "complete"):
		w := strings.TrimSpace(strings.ReplaceAll(t, "✓", ""))
		if w == "" {
			w = "done"
		}
		return "✓ " + w
	default:
		return t
	}
}

// renderMemoryRow — one entry's pinned row:
//
//	2026-08-25 · <title> — <worker> (<role>) · ✓ done  ▸ files: a.go, b.go
//
// The " ▸ files:" segment only when the ledger carried files.
func renderMemoryRow(e officeMemoryEntry) string {
	var sb strings.Builder
	sb.WriteString("  " + e.date + " · " + e.title + " — " + e.worker)
	if e.role != "" {
		sb.WriteString(" (" + e.role + ")")
	}
	sb.WriteString(" · " + e.verdict)
	if len(e.files) > 0 {
		sb.WriteString("  ▸ files: " + strings.Join(e.files, ", "))
	}
	return sb.String()
}

// memoryRowMatches — the /memory <substr> filter: case-fold substring over
// title/files (verified commands deliberately stay OUT of the filter so a
// filter never hides an entry's own proof line).
func memoryRowMatches(e officeMemoryEntry, lf string) bool {
	if strings.Contains(strings.ToLower(e.title), lf) {
		return true
	}
	for _, f := range e.files {
		if strings.Contains(strings.ToLower(f), lf) {
			return true
		}
	}
	return false
}

// memoryBody — the /memory notice text: the header counts every record in
// the file (not just the filtered rows) and names the probe state; rows
// render newest-first exactly as written; a verify digest rides a dim
// continuation line under its own row so the pinned row stays single.
func (m *Model) memoryBody(filter string) string {
	project := m.projInfo().Project
	if project == "" || project == "." || project == "/" || project == string(filepath.Separator) {
		project = filepath.Base(m.memoryDir())
	}
	if project == "" || project == "." {
		project = "theboringoffice"
	}
	header := func(n int) string {
		unit := "dispatches"
		if n == 1 {
			unit = "dispatch"
		}
		return fmt.Sprintf("office memory — %s · %d %s recorded (%s)", project, n, unit, m.memoryProbeState())
	}
	b, err := os.ReadFile(ledgerPath(m.memoryDir()))
	if err != nil {
		// Degrade-open every way (fresh office, concurrent first boot,
		// ledger core still boarding): the honest empty state as ONE dim
		// line — never an error row, never invented rows.
		return header(0) + "\n" +
			"no dispatches recorded yet — the office records every completed dispatch here once it finishes one."
	}
	entries, dropped := parseOfficeLedger(string(b))
	lines := []string{header(len(entries))}
	filter = strings.TrimSpace(filter)
	lf := strings.ToLower(filter)
	shown := 0
	for _, e := range entries {
		if filter != "" && !memoryRowMatches(e, lf) {
			continue
		}
		lines = append(lines, renderMemoryRow(e))
		if e.verify != "" {
			lines = append(lines, "      verify: "+clipRunes(e.verify, 96))
		}
		shown++
	}
	switch {
	case shown > 0:
	case filter != "":
		lines = append(lines, fmt.Sprintf("no dispatches match %q", filter))
	default:
		// The file exists but holds no records (or only malformed ones).
		lines = append(lines,
			"no dispatches recorded yet — the office records every completed dispatch here once it finishes one.")
	}
	if dropped > 0 {
		// ONE dim note for partial parses (not a row per failure) — the
		// ledger keeps serving.
		unit := "row"
		if dropped > 1 {
			unit = "rows"
		}
		lines = append(lines, fmt.Sprintf("  (%d ledger %s skipped — malformed)", dropped, unit))
	}
	return strings.Join(lines, "\n")
}

// describeMemoryRecorded — the completed-dispatch breadcrumb, companion to
// describeEvent (same "[stamp] …" seam): a worker RETURN with a return-kind
// mail (exactly one per completed dispatch, live AND demo twins) leaves
// "[stamp] [memory] recorded: <title> → ledger" in the activity tab — the
// member-visible proof the record went down. ok=false for every other
// event: board-sync task upserts (EvTask) deliberately do NOT stamp —
// a boot-time re-sync of pre-existing done rows is not a recording.
func (m *Model) describeMemoryRecorded(ev state.Event) (string, bool) {
	if ev.Kind != state.EvReturned || ev.Mail.Kind != state.MailReturn {
		return "", false
	}
	title := strings.TrimSpace(strings.TrimPrefix(ev.Mail.Subject, "return: "))
	if title == "" {
		return "", false
	}
	return fmt.Sprintf("[%s] [memory] recorded: %s → ledger", chrome.OfficeClock(m.st.Tick), title), true
}

// persistCfg — brain.json write-through after an in-app mutation (/power,
// /model), best effort: the return string is the trailing word in the
// notice ("saved to brain.json" / the failure).
func (m *Model) persistCfg() string {
	if m.cfg == nil {
		return "in-memory config — not persisted"
	}
	if err := config.Save(m.cfg); err != nil {
		return "brain.json save failed: " + err.Error()
	}
	return "saved to brain.json"
}

// --- backend selection (install-seeded brain.json backend.name + /backend) --

// backendStatusMarker — the EvStatus prefix every transport's boot emits
// FIRST ("[theboringfloor] backend: opencode" / "… claudecode"); the
// reducer latches OfficeState.BackendName off it (topbar segment), and the
// /backend swap line ("[theboringfloor] backend: <old> → <new> (turn #N
// archived)") re-latches through the very same grammar. String-marker
// contract, same pattern as agentFieldStatusMarker.
const backendStatusMarker = "[theboringfloor] backend: "

// backendFailedMarker — the EvStatus prefix a transport's Start failure
// mints from the swap/respawn goroutine. The /bypass respawn latch
// (bypassLatchKick) clears on it too: a failed boot must never wedge the
// one-respawn-in-flight gate.
const backendFailedMarker = "[theboringfloor] backend failed: "

// backendNameFromStatus parses the marker grammar: "… backend: <name>" at
// boot, "… backend: <old> → <new> (…)" on swap — the arrow's RIGHT side is
// always the latched name. Only the two real transport names latch
// (whitelist: a refusal copy deliberately avoids this marker anyway).
func backendNameFromStatus(text string) (string, bool) {
	rest, ok := strings.CutPrefix(text, backendStatusMarker)
	if !ok {
		return "", false
	}
	if i := strings.LastIndex(rest, " → "); i >= 0 {
		rest = rest[i+len(" → "):]
	}
	if i := strings.IndexAny(rest, " (|"); i >= 0 {
		rest = rest[:i]
	}
	switch rest {
	case config.BackendNameDefault, config.BackendNameClaude:
		return rest, true
	}
	return "", false
}

// backendFor — the SINGLE transport resolver: boot (main, via BackendFor),
// the headless name display, and the mid-flight /backend swap all construct
// through this one call. "" resolves to opencode (config's own backfill
// contract — the only honest silence-proof default). baseURL is the
// opencode serve attach target ONLY: the claude transport spawns its own
// CLI (its first param is a bin OVERRIDE, which the office leaves to
// THEBORINGOFFICE_CLAUDE_BIN / PATH resolution).
func backendFor(name, baseURL, dir string, cfg *config.Config) state.Backend {
	switch name {
	case config.BackendNameClaude:
		return backend.NewClaude("", dir, cfg)
	default:
		return backend.NewLive(baseURL, dir, cfg)
	}
}

// BackendFor — cmd/*'s exported view of the resolver (main constructs the
// boot transport, headless names the resolved one). backendFor stays the
// package-internal spelling used by the swap itself.
func BackendFor(name, baseURL, dir string, cfg *config.Config) state.Backend {
	return backendFor(name, baseURL, dir, cfg)
}

// BackendFactory — the swap-time constructor hook: swapBackend builds the
// new transport via this var (never backendFor directly) so tests can
// substitute a stub (parity with app.SpawnTerminal). Production leaves it.
var BackendFactory = backendFor

// backendName — the office's view of the ACTIVE transport: the reducer's
// boot-hint latch first, the brain.json resolution as the boot-early
// fallback (pre-hint frames read the same value either way).
func (m *Model) backendName() string {
	if m.st.BackendName != "" {
		return m.st.BackendName
	}
	if m.cfg != nil {
		return m.cfg.Backend.ResolvedName()
	}
	return config.BackendNameDefault
}

// backendSwapBlockers — the IDLE gate for /backend <name> (the ruling: a
// transport swap is ERROR-PRONE, so it runs only when nothing is in
// flight). One short stable reason per blocker; the refusal notice joins
// them verbatim. Idle surfaces checked: boss/office typing, live worker
// sessions, unanswered permission/question floats, the backlog queue.
func (m *Model) backendSwapBlockers() []string {
	var why []string
	if hasPendingBoss(m.st) {
		why = append(why, "boss turn in flight")
	}
	for _, c := range m.st.Chat {
		if c.Pending && c.From != "boss" {
			why = append(why, "office reply in flight")
			break
		}
	}
	workers := m.liveWorkerCount()
	if workers > 0 {
		why = append(why, fmt.Sprintf("%d worker(s) live", workers))
	}
	if len(m.permQ.pending)+len(m.permQ.escd) > 0 {
		why = append(why, "unanswered permission prompt")
	}
	if m.question != nil || m.questionEscd != nil || m.questionParked {
		why = append(why, "unanswered boss question")
	}
	if len(m.queue) > 0 {
		why = append(why, fmt.Sprintf("queue non-empty (%d items)", len(m.queue)))
	}
	if m.batchInFlight {
		why = append(why, "batch flush in flight")
	}
	return why
}

// applyBackendSlash — /backend [opencode|claudecode]: bare prints the
// active transport, a validated name swaps it mid-flight (IDLE ONLY — the
// busy path is a refusal with the blocker list, never a forced tear-down:
// esc-esc//stop the turn first, then retry).
func (m *Model) applyBackendSlash(fields []string) tea.Cmd {
	if len(fields) < 2 {
		m.notice(fmt.Sprintf("backend: %s — /backend opencode|claudecode swaps (idle office only · persists to brain.json)", m.backendName()))
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(fields[1]))
	if !config.ValidBackendName(name) {
		m.noticeErr(fmt.Sprintf("/backend: unknown backend %q (opencode|claudecode)", fields[1]))
		return nil
	}
	if m.st.Mode != state.ModeLive {
		m.noticeErr("/backend: swap is live-only — the scripted tour's demo backend is fixed")
		return nil
	}
	if name == m.backendName() {
		m.notice("backend already on " + name + " · /session resumes a past one instead")
		return nil
	}
	if m.backendTransitioning {
		m.noticeErr("/backend: a backend transition is already being constructed — wait for it to finish")
		return nil
	}
	if why := m.backendSwapBlockers(); len(why) > 0 {
		msg := fmt.Sprintf("backend swap %s → %s refused — office busy: %s · wait for the turn to finish (esc-esc / /stop first), then /backend %s again",
			m.backendName(), name, strings.Join(why, "; "), name)
		m.noticeErr(msg)
		// statusline twin: the same verdict on the transient surface (the
		// marker grammar is deliberately NOT used — nothing re-latches).
		return m.applyEvent(state.Event{Kind: state.EvStatus, Text: msg})
	}
	m.backendTransitioning = true
	m.backendTransitionID++
	return m.swapBackend(name)
}

// swapBackend runs BackendFactory in a tea.Cmd. The old backend remains
// current until that result lands, so construction cannot park the UI loop.
func (m *Model) swapBackend(name string) tea.Cmd {
	oldName := m.backendName()
	serverURL, sessDir, cfg := m.serverURL, m.sessDir, m.cfg
	transition := m.backendTransitionID
	return func() tea.Msg {
		m.PersistSession() // outgoing archive; disk I/O stays off the UI loop
		resumeID := ""
		if sessDir != "" {
			if sf, ok := LoadSession(sessDir); ok {
				resumeID = sf.primaryIDFor(name)
			}
		}
		return backendBuildMsg{name: name, oldName: oldName, resumeID: resumeID, transition: transition,
			backend: BackendFactory(name, serverURL, sessDir, cfg)}
	}
}

// finishBackendTransition installs a completed replacement. Holder locking is
// limited to replace's generation flip; old send draining and Stop run in the
// returned Cmd, never on Bubble Tea's update goroutine.
func (m *Model) finishBackendTransition(result backendBuildMsg) tea.Cmd {
	if result.transition != m.backendTransitionID {
		if result.bypass && m.bypassRestarting {
			// A superseded bypass build can never become active. Do not leave
			// its lifecycle latch waiting for a status event that belongs to a
			// discarded transport.
			m.bypassRestarting = false
			m.bypassQueued = false // TODO: remove after test update
			m.backendTransitioning = false
		}
		return stopDiscardedBackend(result.backend)
	}
	if result.backend == nil {
		if result.bypass {
			return m.failBypassTransition(errors.New("factory returned no transport"))
		}
		m.noticeErr("backend: factory returned no transport")
		m.backendTransitioning = false
		return nil
	}
	// A bypass replacement is prepared and started before it displaces the
	// accepting generation. This is intentionally unlike /backend: toggling a
	// permission flag must never turn a slow or failed spawn into a send
	// blackout.
	if result.bypass {
		if ps, ok := result.backend.(primarySeamBackend); ok && result.resumeID != "" {
			ps.PrimaryOverride(result.resumeID)
		}
		if bb, ok := result.backend.(bypassBackend); ok {
			if err := bb.SetBypassPermissions(result.bypassValue); err != nil {
				return tea.Batch(
					stopDiscardedBackend(result.backend),
					m.failBypassTransition(fmt.Errorf("set bypass permissions: %w", err)),
				)
			}
		}
		if m.emitFn == nil {
			return m.completeBypassStart(backendStartMsg{result: result})
		}
		emit, nb := m.emitFn, result.backend
		return func() tea.Msg { return backendStartMsg{result: result, err: nb.Start(emit)} }
	}
	m.backend = result.backend
	cleanup := m.currentBackend.replace(result.backend)
	return tea.Batch(cleanup, func() tea.Msg { return backendReadyMsg{result: result} })
}

// completeBackendTransition performs the new backend's setup after the
// generation flip. It is separately scheduled from cleanup: both run outside
// the input handler, and the replacement is already visible to new sends.
func (m *Model) completeBackendTransition(result backendBuildMsg) tea.Cmd {
	if result.transition != m.backendTransitionID {
		if result.bypass {
			return m.finishBypassLatch()
		}
		return nil
	}
	m.backendTransitioning = false
	// Per-backend resume pin is loaded with the build command, so session disk
	// I/O cannot park Update. "" deliberately means a fresh transport.
	if ps, ok := result.backend.(primarySeamBackend); ok && result.resumeID != "" {
		ps.PrimaryOverride(result.resumeID)
	}
	if !result.bypass && m.cfg != nil {
		m.cfg.Backend.Name = result.name
	}
	m.resetPager() // the older-history walk belonged to the old transport
	started := false
	if m.emitFn != nil {
		emit, nb := m.emitFn, result.backend
		go func() {
			if err := nb.Start(emit); err != nil {
				emit(state.Event{Kind: state.EvStatus, Text: backendFailedMarker + err.Error()})
			}
		}()
		started = true
	}
	_ = started
	turns := 0
	for _, c := range m.st.Chat {
		if c.From == "boss" && !c.Pending {
			turns++
		}
	}
	msg := fmt.Sprintf("%s%s → %s (turn #%d archived)", backendStatusMarker, result.oldName, result.name, turns)
	cmd := m.applyEvent(state.Event{Kind: state.EvStatus, Text: msg})
	var saveCfg tea.Cmd
	if m.cfg != nil {
		cfg := m.cfg
		saveCfg = func() tea.Msg { return backendConfigSaveMsg{err: config.Save(cfg)} }
	}
	return tea.Batch(cmd, saveCfg)
}

// --- /bypass — the session-scoped bypass-permissions mode --------------------

// applyBypassSlash — /bypass toggles the mode. ENABLE goes through the
// office's existing question popover as an explicit confirm (a mode that
// silences every permission guardrail never arms on ONE keypress);
// cancel/esc/custom text is a no-op. DISABLE is instant — no confirm.
// Both paths land the transcript notice and respawn the transport (the
// flag freezes into the backend's spawn argv/boot config at Start, so
// only a fresh instance can carry it — respawnForBypass).
func (m *Model) applyBypassSlash() tea.Cmd {
	// The active mode owns the badge, but an in-flight desired ON is also a
	// real toggle target: cancel it immediately without a second build.
	if m.bypassPerms || (m.bypassRestarting && m.bypassDesired) {
		m.bypassDesired = false
		return m.respawnForBypass()
	}
	if m.question != nil {
		// A float already owns the popover region (the confirm itself, or
		// a boss question that outranks the toggle) — never stack.
		return nil
	}
	m.question = &questionHold{
		IDs: []string{bypassConfirmID},
		Items: []state.QuestionItem{{
			Question: bypassConfirmPrompt,
			Options: []state.QuestionOption{
				{Label: "enable"},
				{Label: "cancel"},
			},
		}},
		Answers: make([][]string, 1),
	}
	m.chat.SetQuestion(m.questionView(m.question))
	return nil
}

// respawnForBypass builds a fresh backend while the old one keeps accepting
// sends. The member's context survives: the fresh transport re-pins the
// CURRENT primary id. SetBypassPermissions lands BEFORE Start (the backend
// contract). Demo/harness offices skip the hop entirely. The old backend is
// drained and stopped by replace() in completeBypassStart, never up front —
// toggling a permission flag must never turn a slow spawn into a send blackout.
func (m *Model) respawnForBypass() tea.Cmd {
	if m.st.Mode != state.ModeLive {
		// demo/harness: no transport hop, just flip the flag
		if m.bypassPerms != m.bypassDesired {
			m.bypassPerms = m.bypassDesired
			m.notice(map[bool]string{true: bypassOnNotice, false: bypassOffNotice}[m.bypassPerms])
		}
		return nil
	}
	if m.bypassRestarting {
		// Already restarting — ignore duplicate toggle
		return nil
	}

	m.bypassRestarting = true
	m.backendTransitioning = true
	m.backendTransitionID++
	m.st.StatusLine = "[theboringfloor] backend restarting — bypass permissions " + bypassStateWord(m.bypassDesired)

	resumeID := ""
	if ps, ok := m.backend.(primarySeamBackend); ok {
		resumeID = ps.PrimaryID()
	}
	name, serverURL, sessDir, cfg := m.backendName(), m.serverURL, m.sessDir, m.cfg
	bypassValue, transition := m.bypassDesired, m.backendTransitionID

	return func() tea.Msg {
		if sessDir != "" {
			config.SaveProjectSettings(sessDir, config.ProjectSettings{BypassPermissions: bypassValue})
		}
		return backendBuildMsg{name: name, oldName: name, resumeID: resumeID,
			bypass: true, bypassValue: bypassValue, transition: transition,
			backend: BackendFactory(name, serverURL, sessDir, cfg)}
	}
}

// finishBypassLatch clears the respawn lifecycle latch.
func (m *Model) finishBypassLatch() tea.Cmd {
	m.bypassRestarting = false
	m.bypassQueued = false // TODO: remove after test update
	return nil
}

func bypassStateWord(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// failBypassTransition leaves the accepting generation and active mode in
// place. It is intentionally terminal for this request: retry is a later user
// toggle, not a tight factory/Start loop.
func (m *Model) failBypassTransition(err error) tea.Cmd {
	m.backendTransitioning = false
	m.bypassRestarting = false
	m.bypassQueued = false // TODO: remove after test update
	m.noticeErr("bypass restart failed: " + err.Error() + " — active backend unchanged")
	return m.applyEvent(state.Event{Kind: state.EvStatus,
		Text: "[theboringfloor] bypass restart failed — active permissions " + bypassStateWord(m.bypassPerms)})
}

// stopDiscardedBackend tears down a replacement that never became current.
// It always runs outside Update, so a partially started child cannot orphan
// while the accepting generation remains available to the UI.
func stopDiscardedBackend(backend state.Backend) tea.Cmd {
	if backend == nil {
		return nil
	}
	return func() tea.Msg { return backendStopMsg{err: backend.Stop()} }
}

// completeBypassStart commits a fresh transport only after Start succeeds.
// A now-stale successful build is stopped without ever becoming current, then
// one latest-desired successor is constructed. This makes rapid toggles a
// coalesced state machine rather than a chain of overlapping respawns.
func (m *Model) completeBypassStart(msg backendStartMsg) tea.Cmd {
	result := msg.result
	if result.transition != m.backendTransitionID {
		if result.bypass && m.bypassRestarting {
			// A superseded Start cannot deliver the status event that would
			// otherwise release this latch. Retire its candidate and leave the
			// already-active generation/status untouched.
			m.bypassRestarting = false
			m.bypassQueued = false // TODO: remove after test update
			m.backendTransitioning = false
		}
		return stopDiscardedBackend(result.backend)
	}
	if msg.err != nil {
		return tea.Batch(stopDiscardedBackend(result.backend), m.failBypassTransition(msg.err))
	}
	if result.bypassValue != m.bypassDesired {
		m.bypassRestarting = false
		m.bypassQueued = false // TODO: remove after test update
		m.backendTransitioning = false
		return tea.Batch(stopDiscardedBackend(result.backend), m.respawnForBypass())
	}
	m.backend = result.backend
	cleanup := m.currentBackend.replace(result.backend)
	m.bypassPerms = result.bypassValue
	m.bypassRestarting = false
	m.bypassQueued = false // TODO: remove after test update
	m.backendTransitioning = false
	m.resetPager()
	m.notice(map[bool]string{true: bypassOnNotice, false: bypassOffNotice}[m.bypassPerms])
	return tea.Batch(cleanup, m.applyEvent(state.Event{Kind: state.EvStatus,
		Text: backendStatusMarker + "bypass permissions " + bypassStateWord(m.bypassPerms)}))
}

// notice appends a dim local notice (From "office") to the chat.
func (m *Model) notice(text string) {
	m.appendNotice(text, "")
}

// noticeErr appends a red local notice (From "office", Meta "error").
func (m *Model) noticeErr(text string) {
	m.appendNotice(text, "error")
}

func (m *Model) appendNotice(text, meta string) {
	m.st.Chat = capChat(appendChat(m.st.Chat, state.ChatMsg{
		ID:   nextMsgID(),
		From: "office",
		Text: text,
		Meta: meta,
		At:   time.Now().UnixMilli(),
	}))
	m.tabs.SetState(m.st)
}

// --- activity descriptions --------------------------------------------------

// describeEvent formats the one-line activity entry for a processed event,
// timestamped with the office clock. Ticks never reach this (filtered
// above); every other event kind leaves a trace.
func (m *Model) describeEvent(ev state.Event) string {
	stamp := chrome.OfficeClock(m.st.Tick)
	var what string
	switch ev.Kind {
	case state.EvHire:
		what = fmt.Sprintf("hire %s (%s)", ev.Employee.Name, ev.Employee.Role)
	case state.EvFire:
		what = fmt.Sprintf("fire %s", ev.EmployeeID)
	case state.EvDispatch:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("dispatch → %s «%s»", name, ev.Task.Title)
	case state.EvWorking:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("working — %s", name)
	case state.EvReturned:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("returned ← %s «%s»", name, ev.Mail.Subject)
	case state.EvIdleDrift:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("coffee — %s", name)
	case state.EvBlocked:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("BLOCKED %s — %s", name, ev.Text)
	case state.EvTask:
		what = fmt.Sprintf("task upsert «%s» (%s)", ev.Task.Title, ev.Task.Status)
	case state.EvMail:
		what = fmt.Sprintf("mail %s→%s «%s»", ev.Mail.From, ev.Mail.To, ev.Mail.Subject)
	case state.EvChatUser:
		what = "you › " + ev.Msg.Text
	case state.EvChatBoss:
		if ev.Msg.Pending {
			what = "boss › typing…"
		} else {
			what = "boss › reply"
		}
	case state.EvChatOffice:
		what = "office › reply"
	case state.EvBubble:
		name := ev.EmployeeID
		if e := findEmployee(m.st, ev.EmployeeID); e != nil {
			name = e.Name
		}
		what = fmt.Sprintf("%s says %q", name, ev.Text)
	case state.EvThought:
		name := ev.EmployeeName
		if name == "" {
			name = ev.EmployeeID
		}
		what = "think — " + name + ": " + clipRunes(ev.Text, 60)
	case state.EvTool:
		name := ev.EmployeeName
		if name == "" {
			name = "boss"
		}
		toolState := ev.ToolState
		if toolState == "" {
			toolState = "running"
		}
		text := ev.ToolName
		if ev.ToolSummary != "" {
			text += " · " + ev.ToolSummary
		}
		what = fmt.Sprintf("tool — %s: %s (%s)", name, text, toolState)
	case state.EvPermission:
		name := ev.EmployeeName
		if name == "" {
			name = "boss"
		}
		toolState := ev.ToolState
		if toolState == "" {
			toolState = "pending"
		}
		text := ev.ToolName
		if ev.ToolSummary != "" {
			text += " · " + ev.ToolSummary
		}
		what = fmt.Sprintf("permission — %s: %s (%s)", name, text, toolState)
	case state.EvQuestion:
		name := ev.EmployeeName
		if name == "" {
			name = "boss"
		}
		what = "question — " + name + ": " + clipRunes(ev.Text, 60)
	case state.EvFileDiff:
		name := ev.EmployeeName
		if name == "" {
			name = "boss"
		}
		what = fmt.Sprintf("diff — %s: %s +%d -%d", name, ev.DiffPath, ev.DiffAdd, ev.DiffDel)
	case state.EvStatus:
		what = "status — " + ev.Text
	case state.EvOffline:
		what = "OFFLINE — waiting for internet…"
	case state.EvOnline:
		what = "back online — resumed"
	case state.EvUsage:
		what = fmt.Sprintf("usage — +%d in / +%d out tok · +$%.4f", ev.TokensIn, ev.TokensOut, ev.CostUSD)
	default:
		what = string(ev.Kind)
	}
	// keep each row single-line for the log
	what = strings.ReplaceAll(what, "\n", " ")
	return fmt.Sprintf("[%s] %s", stamp, what)
}

// clipRunes truncates machine text (activity descriptions) to n runes with
// an ellipsis — display layout, not NL.
func clipRunes(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
