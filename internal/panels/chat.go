// chat.go — THE STAR of the sidebar: the chat tab.
//
// Layout inside the tab panel (top → bottom):
//
//	viewport  — the whole conversation, glamour-rendered markdown.
//	            user turns are plain-wrapped and prefixed cyan "you › ";
//	            boss turns are rendered THROUGH GLAMOUR as markdown
//	            (**bold**, lists, fences format + wrap) with a yellow
//	            "boss › " hanging indent, then HARD-FOLDED to the panel
//	            width (glamour's wrap misses code fences and unbreakable
//	            URLs — foldStyledLines refolds those, ANSI-aware).
//	            Kind "think" entries render as dim-italic thinking blocks —
//	            LIVE while their CallID streams (spinner header, growing
//	            tail, always expanded), then COLLAPSED to
//	            "thinking · N lines" until ctrl+t expands all;
//	            Kind "tool" entries (BOSS/primary) render as dim one-liners
//	            merged by CallID (wrapped with a hanging indent, never
//	            clipped); Kind "wtool" entries (EMPLOYEE tools)
//	            group into one thread per agent, and the thread joins
//	            the SAME timeline as every other entry —
//	            mergeChatTimeline places it at its birth timestamp
//	            (creation time), so threads scroll with the
//	            conversation instead of docking after it. Threads
//	            render opencode-style (threads_opencode.go) — flat
//	            frame lines, no boxing: the HEADER row is the 2-cell
//	            glyph field plus the "<Kind> Task — <task>" title
//	            (Kind from the roster role — scout = "Explore" —
//	            fallback "Subagent Task — <agent>"), ONE row elided
//	            to the panel width with clipPlain (tabs.go), never
//	            wrapped. The glyph is the animated braille spinner
//	            (dot frames incl. ⠿, house accent — one shared
//	            model, every live thread pulses the same frame)
//	            while the thread is LIVE, and a live header carries
//	            NO rollup text; a DONE head swaps to a dim "✓" with
//	            the dim trailing rollup "(· N tool calls[ · M
//	            think] ✓ done)", a /stop-stopped head to a dim-red
//	            "✗" with "✗ stopped". The row under the header is
//	            the sneak peek — dim "  ↳ <Verb> <rest>", the
//	            thread's NEWEST TOOL entry BARE: one row,
//	            clipPlain-elided like the header, NO state suffix,
//	            and a trailing thought never steals it (a
//	            think-ONLY thread sneaks "thinking · N lines"
//	            instead); the raw "read · x" meta text is shaped
//	            into "Read x" display-side inside
//	            threads_opencode.go. A per-agent click toggles the
//	            thread from its FRAME rows (the threadRows hit-map
//	            is state-conditional — collapsed: header + sneak;
//	            expanded: header + the closing rollup rows) or the
//	            ctrl+g baseline expands the thread to its merged
//	            "[tool] …"/think rows 2-cell indented (long CONTENT
//	            rows wrap with hanging continuations), then the ↳
//	            sneak again as the "current task" line and a dim
//	            closing rollup ("  · N tool calls ✓ done"); ctrl+g
//	            expands/collapses all threads at once, /stop
//	            force-collapses to the "✗ … ✗ stopped" header, and
//	            while ≥1 thread is live a dim-italic "ctrl+g · view
//	            subagents" hint row trails the last thread block.
//	            USER turns longer than userFoldVisible rendered rows
//	            FOLD to their head rows + a dim-italic "… +N more
//	            lines · click to expand" hint (the 📎 count moves
//	            onto the hint); a click on the hint — y-only hit-map
//	            userFoldRows — expands the full body + a "… collapse"
//	            trailer. Every viewport content line rides a
//	            chatPadL-cell left inset with the wrap budget shrunk
//	            chatPadL+chatPadR (contentW) — the transcript keeps
//	            a 2-cell margin off both panel edges while the
//	            divider/spinner/textarea chrome stays full width;
//	            Kind "office" entries (the concierge's EvChatOffice seam)
//	            render as INFO-cyan "office ›" markdown bubbles — a real
//	            turn, streamed replace-by-ID like the boss, with a dim
//	            "office is answering…" placeholder while pending-empty;
//	            From "office" entries with no Kind render as dim local
//	            notices (red when Meta == "error").
//	divider
//	spinner   — the typing row, shown for the WHOLE pending period:
//	            while ANY boss reply is outstanding (a breathing
//	            block-glyph bar — threads_opencode.go's
//	            pendingBlockBar — + " <boss> is typing…", named from
//	            brain.json boss.name) — with or
//	            without streamed text. A pending boss bubble WITH text
//	            keeps rendering in the viewport as a streaming "boss ›"
//	            turn (glamour markdown re-rendered per delta); the row
//	            below is the liveness signal now — there is NO caret.
//	            While BossDelegating holds (boss quiet, workers busy)
//	            the row swaps to a settled dim " <boss>: delegating ·
//	            N busy" — no spinner.
//	textarea  — multiline input; Enter sends, Shift+Enter (or Ctrl+J) is a
//	            newline, placeholder "talk to the boss…". NEVER locked: while
//	            the boss is busy, Enter FREE-SENDS (the app routes the prompt
//	            straight to the backend, which queues it server-side and
//	            drains it after the current turn — no client queue, nothing
//	            hides); the placeholder reads "<boss> is typing…", and from
//	            the second direct send on "<boss>: turn N · your message
//	            rides next" while the status line carries "busy · N queued
//	            (server)". A client-side backlog item is created ONLY for
//	            roadblocks: a permission popover or a question hold
//	            outstanding. The permission popover FLOATS centered over
//	            the whole chat view as an amber bordered card (textarea
//	            keeps rendering + typing under it — see perm_modal.go):
//	            up/down/tab walk the menu cursor, enter confirms the
//	            highlighted option, y/a/n quick-answer, esc defers; every
//	            other key still types into the textarea below.
//	question  — while a boss question page is open, a QUESTION POPOVER
//	            FLOATS over this view (yellow card, same splice as the
//	            permission popover — question_modal.go): four kinds —
//	            radio/checkbox/confirm option pages and a free-text
//	            textarea box — plus a "Type your own answer…" row on
//	            option pages. The popover OWNS every key while open
//	            (enter/ctrl+enter submits → AnswerQuestion, esc defers
//	            → /question re-opens); the textarea underneath is
//	            DISABLED. While the hold is outstanding (open or
//	            deferred) the placeholder reads "boss is waiting
//	            for your answer… · N queued" and Enter still enqueues.
//
// Scroll: mouse wheel + PgUp/PgDn always scroll the conversation; ↑/↓ move
// inside a multi-line draft and scroll the conversation otherwise. ctrl+t
// expands/collapses ALL thinking blocks, ctrl+d expands/collapses ALL diff
// entries (both handled by the app keymap). While any worker thread renders
// expanded, esc and ↑ are "back one" FIRST: each press folds the MOST
// RECENTLY expanded thread back to its collapsed summary row (one thread
// per press); with nothing expanded, ↑ walks its old scroll path untouched
// and esc-esc in the main input (two UNCONSUMED presses inside 500ms)
// aborts the running turn — the same path as /stop.
//
// Selection: a left-drag over the transcript selects its VISIBLE text
// (reverse-video overlay, extraction to plain text, OSC52 copy rises
// app-side) — chat_selection.go owns the five-method seam the app calls;
// esc clears a finished highlight before anything else (app routing).
package panels

import (
	"fmt"
	"image/color"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	chlexers "github.com/alecthomas/chroma/v2/lexers"
	chstyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	userPrefix   = "you › "
	bossPrefix   = "boss › "
	officePrefix = "office › "

	placeholderIdle = "talk to the boss…"
	placeholderWait = "boss is waiting for your answer…"

	// defaultBossShort — the busy placeholder's boss word when no config
	// short name is set ("boss is typing…"; SetBossShortName personalizes).
	defaultBossShort = "boss"

	textareaH = 3 // rows of multiline input at the bottom of the tab

	// userFoldVisible — a user bubble whose WRAPPED body exceeds this many
	// rendered rows collapses to its first userFoldVisible rows plus a
	// dim-italic "… +N more lines · click to expand" hint row (the hint
	// toggles on click; the expanded bubble trails a "… collapse" row
	// instead). The count is taken POST-fold (wrap + refold), i.e. over
	// the rows the screen would actually draw.
	userFoldVisible = 3

	// chatPadL / chatPadR — the chat transcript's inside gutters: the
	// viewport indents every content line chatPadL cells (applied
	// viewport-locally by setConversation), and every transcript wrap/clip
	// budget runs contentW() = w − chatPadL − chatPadR so bubbles keep a
	// chatPadR-cell margin off the right edge too. Rows OUTSIDE the
	// transcript viewport — the divider, the typing/loading rows, the
	// chips, the @ picker, the slash popover and the textarea — keep the
	// full panel width.
	chatPadL = 2
	chatPadR = 2
)

// thinkKind / toolKind / wtoolKind / questionKind / diffKind / officeFrom /
// errMeta — chat entry markers the app reducer tags onto state.ChatMsg so
// this panel can style them without touching the user/boss turn paths.
// wtoolKind is the EMPLOYEE tool line: grouped into a per-agent thread that
// renders INLINE in the conversation timeline (see mergeChatTimeline), as an
// opencode-style spinner+sneak thread, collapsed by default.
// wthinkKind is
// the EMPLOYEE EvThought line (the app reducer merges one per agent+CallID,
// same stream mechanics as the boss's thinkKind): it joins the SAME worker
// thread — a dim-italic "thinking · N lines" row inside the expanded entry
// list, a "· M think" count in the rollup summaries, the ↳ sneak's
// fallback only when a thread has NO tool line yet, a full body under
// ctrl+g.
const (
	thinkKind  = "think"
	toolKind   = "tool"
	wtoolKind  = "wtool"
	wthinkKind = "wthink"
	// wdiffKind is the per-CALL worker diff ("wdiff-<agent>-<callid>"): a
	// completed employee edit/write lifted its patch off the wire's
	// tool-part metadata, so the diff pins INSIDE the agent's thread —
	// directly beneath its [tool] row as a clickable "↳ diff · path +A -D"
	// sub-row (click opens/closes the parsed body; the body reuses the
	// flat-diff parsing/rendering machinery). It merges into the SAME
	// workerGroup as wtool/wthink but counts as NEITHER in the rollups.
	wdiffKind    = "wdiff"
	questionKind = "question"
	diffKind     = "diff"
	officeFrom   = "office"
	errMeta      = "error"
	// bootWarnMeta — the wedge watchdog user's boot-scoped error class:
	// renders EXACTLY like errMeta red, butSnapshot-/boot-scoped cleansing
	// routes it (its real home is internal/app/sessions.go's
	// bootWarnNoticeMeta, kept in lockstep).
	bootWarnMeta = "boot-warn"
	// officeKind — concierge chat bubbles (the EvChatOffice seam: From
	// "office" + Kind "office"). A concierge answer is a REAL conversation
	// turn — INFO-colored "office ›" label over glamour markdown, same
	// replace-by-ID streaming mechanics as the boss — NOT a dim notice
	// (notices keep Kind "" and land in renderNotice).
	officeKind = "office"
)

// wtoolStaleTicks — the staleness horizon that decides whether a workers
// thread counts as LIVE (busy sprite + tool activity inside the window).
// It drives the "team is working" loading row (chat_loading.go) and the
// per-tick re-render gate in SetState — thread EXPANSION no longer rides
// it: every thread is collapsed by default, live ones included, on the
// ctrl+g / per-agent baseline alone.
const wtoolStaleTicks = 120

// dblEscWindow — two esc presses in the MAIN chat input inside this window
// count as a double-esc and fire the stop seam (the app's /stop abort
// path). A lone press is only recorded; window-close pairs re-arm so a
// third esc starts a fresh pair instead of stacking interrupts.
const dblEscWindow = 500 * time.Millisecond

// diffMetaSep splits the diff ChatMsg.Meta carrier: path ␟ +adds ␟ -dels
// (unit separator — paths may contain spaces). Written by the app reducer,
// read only here.
const diffMetaSep = "\x1f"

// diffClip is the max body lines shown in an expanded diff before
// "+N more" truncation.
const diffClip = 30

// PermissionView is the open permission popover the chat panel floats
// over the chat region (set/cleared by the app via SetPermission).
type PermissionView struct {
	ID       string // pending wire request id
	ToolName string // permission name, e.g. "Write", "external_directory"
	Summary  string // one-liner, e.g. "/tmp/x"
	Agent    string // display name of the requesting agent ("boss" or child)
	Index    int    // 1-based position in the pending queue
	Total    int    // total pending permissions
}

// QuestionView / QuestionAnswer / QuestionKind along with the whole
// question popover live in question_modal.go — the boss question is a
// FLOATING yellow card over the assembled chat view now (the old
// region-replacing modal is gone; the row budget never budges).

// Chat is the chat tab panel.
type Chat struct {
	vp viewport.Model
	ta textarea.Model
	// sp — the braille spinner (spinner.MiniDot, magenta) whose frame
	// renders as the glyph of every LIVE worker-thread header. Advanced
	// by the spinner.TickMsg arm below; pre-tick it renders the first
	// frame ("⠋"), which the tests pin. The pending typing row does NOT
	// read it — that row's block bar breathes off the office tick
	// (pendingBlockBar).
	sp     spinner.Model
	onSend func(text string, atts []state.Attachment) tea.Cmd

	// Queue + permission + question + free-send seams (set by the app at
	// build time). onEnqueue is for ROADBLOCKS only: Enter while a question
	// hold is outstanding / a permission modal is open enqueues (the turn
	// is parked at the reply API — a chat prompt would not resume it).
	// onBusySend is FREE-QUEUING: Enter while the boss is merely busy goes
	// straight to the backend, which queues the prompt server-side.
	onEnqueue    func(text string, atts []state.Attachment) tea.Cmd
	onBusySend   func(text string, atts []state.Attachment) tea.Cmd
	onPermAnswer func(response string) tea.Cmd
	onPermLater  func() tea.Cmd // esc defers the prompt
	// Permission popover state: perm is the open ask (set/cleared by the
	// app; overlapping asks queue app-side and swap here one at a time),
	// permSel the menu cursor (0=Allow once, 1=Allow always, 2=Reject).
	// The popover is a floating OVERLAY, not a region — the textarea
	// keeps rendering + typing under it (see perm_modal.go).
	perm     *PermissionView
	permSel  int
	queueLen int

	// Question popover — the open boss question page FLOATS over the
	// view (question_modal.go — same card mechanics as the permission
	// popover) and OWNS every key while open: the turn is parked at the
	// question reply API, so the main textarea is disabled until the
	// answer goes through onQuestionAnswer (questionWaiting survives
	// esc-defer: the turn stays parked and typed text ENQUEUES until
	// the resolved + completed boss reply). qSel is the menu cursor on
	// option pages, qPicked the checkbox toggles (sized to
	// len(question.Options)), qText the multi-line buffer of the TEXT
	// page AND the 1-line custom-answer input of the option pages.
	question         *QuestionView
	onQuestionAnswer func(a QuestionAnswer) tea.Cmd
	onQuestionLater  func() tea.Cmd
	qSel             int
	qPicked          []bool
	qText            string
	questionWaiting  bool // question hold outstanding (open or deferred)

	// pasteChips — the chat textarea's collapsed large pastes, in
	// insertion order (chat_paste.go): the draft holds each chip's
	// one-line token ("[pasted N lines · M chars]"), send expands the
	// tokens back to the full original text, backspace pops a chip as
	// ONE unit. Cleared with the draft on every send.
	pasteChips []pasteChip

	// Session picker (/session) — session_picker.go: an ADDITIVE floating
	// card listing the server's root sessions (type to narrow, ↑/↓, enter
	// re-anchors the office LIVE, esc cancels). Same float mechanics as
	// the permission/question popovers; it OWNS every key while open
	// (question-modal style — the textarea under it is disabled), but
	// yields to a permission/question float popping over it (a parked
	// turn outranks browsing). Open → loading state; the app's async
	// ListSessions hop lands via SetSessionPickerRows.
	sessPick        *sessionPickState
	onSessionPick   func(id string) tea.Cmd
	onSessionCancel func() tea.Cmd

	// Double-esc interrupt (the panic-key): in the MAIN chat input — no
	// modal or picker consumed the key — two esc presses inside
	// dblEscWindow fire onStopEsc, the app's /stop abort path. A lone esc
	// is just recorded (single-esc behavior in the input stays a no-op);
	// a completed double-esc re-arms so a third press opens a fresh pair.
	lastEscAt time.Time
	onStopEsc func() tea.Cmd

	chat    []state.ChatMsg // rendered snapshot
	pending bool
	// bossShort — the boss's display short name (first word of brain.json
	// boss.name), used by the busy placeholder + typing spinner line.
	bossShort string
	// pendingSpin — the typing row (" boss is typing…" spinner) shows for
	// the WHOLE pending period: any outstanding boss reply, text or not.
	// The streaming bubble in the viewport carries the partial text; this
	// row (below the divider, above the input) carries the liveness.
	pendingSpin bool
	follow      bool // stick to the bottom unless the user scrolled up

	showThinking    bool // /thinking on|off — collected blocks visible (default true)
	showTools       bool // /tools on|off    — tool one-liners visible (default true)
	thinkExpanded   bool // ctrl+t — thinking expanded; DEFAULT false (collapsed)
	diffExpanded    bool // ctrl+d or /diffs on|off — diffs expanded; DEFAULT false
	threadsExpanded bool // ctrl+g — completed worker threads expanded; DEFAULT false
	compactRows     bool // /compact layout — the textarea trims to 2 visible rows

	// agents — per-name worker rollup captured from st.Employees each
	// SetState (task title for the thread header, sprite-active for the
	// live/collapsed decision). The reducer's wtool entries carry only the
	// name; the roster is the source of their decoration.
	agents map[string]agentView
	// workerTasks — sticky last-known task title per worker name. The
	// EvReturned reducer CLEARS Employee.Task, but a collapsed thread's
	// summary must still read "tekton-1 · Wire… (· N tool calls ✓ done)".
	workerTasks map[string]string
	// segTitles — per-SEGMENT dispatch titles, memoized by the segment's
	// first chat line id. renderConversation rebuilds the worker groups
	// from scratch every render; without the memo, an earlier epoch of a
	// recycled desk name would re-capture the CURRENT sticky task on every
	// rebuild and both epochs would wear today's title.
	segTitles map[string]string

	// delegating/delegatingN — P3: BossDelegating flips the pending-spin
	// row from the typing spinner to a settled " <boss>: delegating ·
	// N busy" (dim, no spinner). N = hired (non-manager) employees in
	// working/to-manager/meeting sprites.
	delegating  bool
	delegatingN int

	// serverTurn — the app's free-queuing tally: prompts sent DIRECT to the
	// backend during the current busy turn (the serve queues them
	// natively). 0 = none this turn. From the second send on the busy
	// placeholder reads "<boss>: turn N · your message rides next".
	serverTurn int

	// streamingThink — CallIDs with an OPEN boss EvThought stream (set by
	// the app from its model-owned active set). Streaming blocks always
	// render expanded (live transcript) regardless of ctrl+t; when the
	// stream closes (Done / new boss turn) they collapse to "thinking ·
	// N lines". tick drives the header spinner — no extra timer.
	streamingThink map[string]bool
	tick           int

	w, h      int
	md        *glamour.TermRenderer
	mdWidth   int
	renderRev uint64 // cheap changed-detection for SetState

	// deferRender — the thread-focus view's render saver (set by the app
	// via SetDeferredRender while its focusDeferredRender flag is up):
	// the focus pane renders from the SAME office state every pulse, so
	// this main transcript's SetState keeps recording rev+snapshot but
	// SKIPS renderConversation+vp.SetContent — rebuilding hidden rows
	// per tick would be wasted work. ResumeFromFocus forces exactly one
	// re-render at close. Zero behavior change while false (the default).
	deferRender bool
	renderCalls int // rebuild tally — the focus deferral's probe

	// Attachment state (chat_attach.go): staged chips above the textarea
	// plus the @ file-picker popover. The chips drain into onSend/onEnqueue
	// on Enter; temp paste dirs are removed by the app's send closures (or
	// here when a chip is dropped before ever sending).
	atts             []chatAttachment
	atOpen           bool                      // @ picker visible
	atFrag           string                    // live "@fragment" filter (tail-derived)
	atFiles          []string                  // walked repo files; nil until the walk answers
	atFiltered       []string                  // refilterAttach's visible slice
	atSel            int                       // selected row inside atFiltered
	pasteSeq         int                       // paste-N.png naming for image attaches
	pasteUnsupported bool                      // non-darwin notice latch (fire once)
	onNotice         func(text string) tea.Cmd // office-notice seam (cap/drop/platform)

	// Open-in-browser float (links.go): `o` over a selected bubble with
	// MULTIPLE verified targets floats this centered card (the /session
	// picker's shape — no filter, the count is tiny); enter/esc fire the
	// app's ferried handlers, the mark underneath stays esc-lawful.
	openPick     *linkPickState
	onLinkPick   func(t LinkTarget) tea.Cmd
	onLinkCancel func() tea.Cmd

	// Slash popover (popover.go): "/" at a word start opens the command
	// menu; "/theme " flips it into the live-preview theme picker.
	slashOpen      bool
	slashMode      int // slashModeCmd | slashModeTheme
	slashFrag      string
	slashSel       int
	slashCmds      []slashCommand // refilterSlash's visible slice (cmd mode)
	slashThemes    []string       // refilterSlash's visible slice (theme mode)
	slashPrevTheme string         // theme captured when a preview session opened

	// click/expansion bookkeeping for the workers-thread region:
	// threadExpand holds PER-AGENT explicit expansion (double-click an
	// agent on the floor / click a thread's header row — a set entry
	// wins the ctrl+g default outright: every thread is COLLAPSED BY
	// DEFAULT, live ones included); threadRows maps rendered content
	// line → agent name, STATE-CONDITIONALLY: collapsed registers the
	// header row + the ↳ sneak row, expanded registers the header row +
	// the closing summary row(s) — the whole bubble's frame rows toggle,
	// the internal tool/think rows and the expanded sneak pass through.
	// threadStop holds the /stop markers: a stopped thread force-collapses
	// and its header reads "✗ … · stopped" until an explicit expand
	// re-opens the rows. threadExpandOrder is the expansion-ORDER ledger
	// (oldest first) the esc/↑ "back one" collapse walks: ExpandThread
	// appends, collapseLastThread pops.
	threadExpand      map[string]bool
	threadRows        map[int]string
	threadStop        map[string]bool
	threadExpandOrder []string

	// threadDiffOpen / toolDiffRows — the per-call thread-diff pair,
	// twins of threadExpand/threadRows: toolDiffRows is the click hit-map
	// (rendered content row → wdiff msg ID) registering ONLY the expanded
	// thread's "↳ diff · path +A -D" sub-rows (rebuilt every render, body
	// rows themselves never register); threadDiffOpen holds each toggled-
	// open wdiff body (default closed). ClickRow consults toolDiffRows
	// right AFTER threadRows. Thread diffs stay click-only — the flat
	// diff world keeps ctrl+d.
	threadDiffOpen map[string]bool
	toolDiffRows   map[int]string

	// userExpanded / userFoldRows — the user-bubble fold pair, twins of
	// threadExpand/threadRows: a user bubble whose wrapped body exceeds
	// userFoldVisible rows renders collapsed (head rows + a clickable
	// "… +N more lines · click to expand" hint at the hanging indent);
	// userExpanded holds the explicit per-bubble state (default folded),
	// keyed by userFoldKey (message ID, timestamp fallback); userFoldRows
	// is the mouse hit-map, rebuilt every render — the folded bubble's
	// hint row and the expanded bubble's "… collapse" row ONLY (body
	// rows never register), each mapping its content row → the fold key.
	userExpanded map[string]bool
	userFoldRows map[int]string
	btwPinRows   map[int]string

	// toolExpanded / toolRows / toolOutputs — the per-call tool-output
	// triple, twins of userExpanded/userFoldRows (chat_toolrow.go):
	// toolExpanded holds the explicit per-entry expansion (default
	// collapsed — every tool row expands/collapses INDEPENDENTLY), keyed
	// by the entry's message ID ("tool-<callID>" /
	// "wtool-<agent>-<callID>"), so a running→done merge that REPLACES
	// the entry keeps its expansion and the done event updates the
	// expanded body in place; toolRows is the mouse hit-map (rendered
	// row → entry ID, rebuilt every render — the one-liner's own rows
	// ONLY, body rows never register); toolOutputs carries each entry's
	// captured result text (the app's SetToolOutput feed of
	// state.Event.ToolOutput) — absent/empty renders the pinned
	// "no output as such" empty state.
	toolExpanded map[string]bool
	toolRows     map[int]string
	toolOutputs  map[string]string

	diffCache map[string]diffCacheEntry // parsed+hilighted diff rows by msg ID

	// Inbound boss-turn image previews (chat_raster.go owns the paint;
	// the app's lazy rasterize pushes through SetImageRaster /
	// SetImageFailed): media[msgID][hash] is that bubble's view row —
	// the raster rows, or the failed latch (the chip drops to the dim
	// "unsupported image" copy). mediaRev counts arrivals per bubble:
	// folded into that bubble's block-cache key (renderMsgBlock), exactly
	// one re-render per landing — the cached block for a bubble whose
	// media set NEVER changed stays borrowed.
	media    map[string]map[string]chatMediaView
	mediaRev map[string]uint64

	// Transcript mouse selection (chat_selection.go): a left-press over
	// the transcript viewport arms a pending selection, drag-motion moves
	// its head, a dragged release extracts the visible plain text (the
	// app rides it into OSC52 + the "Copied N chars" toast). sel holds
	// the endpoints in CONTENT-LINE space (scroll-independent); selLines
	// caches the padded lines as posted to the viewport — the highlight
	// overlay's row space and the (clean, pre-highlight) extraction
	// source — refreshed on EVERY setConversation.
	sel      selState
	selLines []string

	// Transcript virtualization (chat_window.go):
	//   — blockCache/blocks: buildBlocks renders each timeline item ONCE
	//     per (identity × width/theme generation × its toggles × content)
	//     and caches the finished PADDED lines + LOCAL click hit-map;
	//     assembly splices cached rows (an append / stream update
	//     re-renders only the touched tail blocks).
	//   — win: the viewport's posted content is the same LINE COUNT as
	//     selLines but materialized only around the scroll offset
	//     (+overscan) — the blank-height model. Scroll/page-in, follow,
	//     ClickRow and TranscriptRows() are unchanged: the posted row
	//     space is one-to-one with selLines.
	//   — themeGen: RefreshTheme's bump — styles bake into cached
	//     fragments, so a /theme switch must miss every block.
	//   — convWide: widest PADDED line of the last assembly — the
	//     rogue-row guard's skip-scan signal.
	blockCache map[string]*chatBlock
	blocks     []*chatBlock
	win        vpWindow
	themeGen   uint64
	convWide   int
}

// NewChat builds the panel; onSend is invoked on Enter with a non-empty
// draft (or any staged attachments) — the app turns it into backend.Send +
// chat-user/pending events. Attachments ride the second argument (nil for
// a plain text send).
func NewChat(onSend func(text string, atts []state.Attachment) tea.Cmd) *Chat {
	vp := viewport.New(viewport.WithWidth(10), viewport.WithHeight(5))
	// SoftWrap OFF, on purpose (chat_window.go): the transcript renderers
	// pre-fold every row to the viewport width (foldStyledLines /
	// foldStyledRows / clipPlain budgets + the rogue-row rescue), so
	// wrapping was already a no-op for this content — with it off the
	// viewport's per-scroll/per-paint walks drop from O(total rows) to
	// O(1) and the window's blank-height model stays exact (every posted
	// row is exactly 1 cell tall).
	vp.SoftWrap = false
	vp.MouseWheelEnabled = true

	ta := textarea.New()
	ta.Prompt = "› "
	ta.Placeholder = placeholderIdle
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(textareaH)
	ta.SetWidth(30)
	// Enter SENDs; Shift+Enter or Ctrl+J inserts a newline. Delivery:
	// shift+enter only REACHES us on kitty-keyboard-protocol terminals
	// (ghostty, kitty — the office negotiates that protocol already);
	// legacy terminals encode shift+enter as a bare enter, where ctrl+j
	// (0x0a, universally distinct) is the fallback that always lands.
	// Both paths are pinned in chat_paste_test.go.
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("shift+enter", "ctrl+j"),
		key.WithHelp("shift+enter", "newline"),
	)
	applyTextareaStyles(&ta)
	ta.Focus()

	// the braille thread spinner — ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏ in the
	// theme's magenta accent — the LIVE worker-thread glyph field
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(threadSpinnerStyle()),
	)

	c := &Chat{vp: vp, ta: ta, sp: sp, onSend: onSend, follow: true,
		showThinking: true, showTools: true, diffCache: map[string]diffCacheEntry{},
		workerTasks: map[string]string{}, blockCache: map[string]*chatBlock{}}
	c.SetSize(30, 10)
	return c
}

// applyTextareaStyles points the textarea at the live chrome palette —
// called at build time AND on every /theme switch (RefreshTheme).
func applyTextareaStyles(ta *textarea.Model) {
	styles := textarea.DefaultDarkStyles()
	// Bubbles renders the editing surface independently of the sidebar
	// wrapper. Paint every cell-bearing state so the textarea stays one
	// continuous panel surface in both focus states (including its blank tail).
	styles.Focused.Base = styles.Focused.Base.Background(chrome.PanelBgColor)
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(chrome.Accent).Background(chrome.PanelBgColor)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(chrome.Dim).Background(chrome.PanelBgColor)
	styles.Focused.CursorLine = lipgloss.NewStyle().Background(chrome.PanelBgColor)
	styles.Focused.Text = lipgloss.NewStyle().Background(chrome.PanelBgColor)
	styles.Focused.EndOfBuffer = lipgloss.NewStyle().Background(chrome.PanelBgColor)
	styles.Blurred.Base = styles.Blurred.Base.Background(chrome.PanelBgColor)
	styles.Blurred.Prompt = styles.Blurred.Prompt.Background(chrome.PanelBgColor)
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(chrome.Accent).Faint(true).Background(chrome.PanelBgColor)
	styles.Blurred.CursorLine = styles.Blurred.CursorLine.Background(chrome.PanelBgColor)
	styles.Blurred.Text = styles.Blurred.Text.Background(chrome.PanelBgColor)
	styles.Blurred.EndOfBuffer = styles.Blurred.EndOfBuffer.Background(chrome.PanelBgColor)
	ta.SetStyles(styles)
}

// RefreshTheme re-points every cached style-derived surface at the active
// theme: textarea styles, spinner color, and the glamour renderer (rebuilt
// lazily at the next boss turn). Called by the app on /theme switches.
func (c *Chat) RefreshTheme() {
	applyTextareaStyles(&c.ta)
	c.sp.Style = threadSpinnerStyle() // the magenta braille thread glyph
	c.md = nil
	c.diffCache = map[string]diffCacheEntry{} // syntax colours are theme-bound
	// styles bake into every cached transcript fragment — bump the
	// generation so the next render re-renders every block at the new
	// palette (the same miss a width flip forces).
	c.themeGen++
	c.forceRender()
}

// ToggleThink expands/collapses ALL thinking blocks in the conversation
// (ctrl+t, routed by the app keymap).
func (c *Chat) ToggleThink() {
	c.thinkExpanded = !c.thinkExpanded
	c.forceRender()
}

// ToggleDiffs expands/collapses ALL diff entries in the conversation
// (ctrl+d, routed by the app keymap).
func (c *Chat) ToggleDiffs() {
	c.diffExpanded = !c.diffExpanded
	c.forceRender()
}

// ToggleThreads expands/collapses ALL worker threads in the conversation
// (ctrl+g, routed by the app keymap) — the only GLOBAL expand baseline:
// threads are collapsed by default, live ones included, so the opens are
// this baseline and the per-agent click override (a /stop stopped thread
// stays folded to its "✗ stopped" header frame under both until an
// explicit per-agent expand). Independent of the thinking toggles.
func (c *Chat) ToggleThreads() {
	c.threadsExpanded = !c.threadsExpanded
	c.forceRender()
}

// ToggleThreadDiff opens/closes ONE worker diff's parsed body inside its
// thread (mouse: click the expanded thread's "↳ diff · path +A -D"
// sub-row — toolDiffRows keys the lookup). Thread diffs are click-only:
// the flat-diff ctrl+d baseline (diffExpanded) never opens them.
func (c *Chat) ToggleThreadDiff(id string) {
	if c.threadDiffOpen == nil {
		c.threadDiffOpen = map[string]bool{}
	}
	c.threadDiffOpen[id] = !c.threadDiffOpen[id]
	c.forceRender()
}

// ExpandThread sets ONE agent's work-thread expansion explicitly (mouse:
// double-click the agent's floor sprite / click the thread's header or
// sneak row). A set per-agent override wins the ctrl+g default outright
// until the next call (a mouse-collapsed live thread stays collapsed
// until re-clicked) — ctrl+g still moves the global baseline underneath
// it.
// Every call also moves the agent inside threadExpandOrder (expand →
// newest tail, collapse → out) — the ledger esc/↑ "back one" pops from.
func (c *Chat) ExpandThread(agent string, expanded bool) {
	if c.threadExpand == nil {
		c.threadExpand = map[string]bool{}
	}
	c.threadExpand[agent] = expanded
	c.threadExpandOrder = removeString(c.threadExpandOrder, agent)
	if expanded {
		c.threadExpandOrder = append(c.threadExpandOrder, agent)
	}
	c.forceRender()
}

// ToggleThread flips ONE agent's thread from its CURRENT effective state
// (what renderWorkerGroup shows for it right now).
func (c *Chat) ToggleThread(agent string) {
	c.ExpandThread(agent, !c.threadExpandedNow(agent))
}

// userFoldKey — one user bubble's fold-state key: the message's stable ID
// ("user-N" server-side), falling back to its send timestamp so an
// empty-ID message still folds without sharing a key with every other
// empty-ID message.
func userFoldKey(m state.ChatMsg) string {
	if m.ID != "" {
		return m.ID
	}
	return fmt.Sprintf("at-%d", m.At)
}

// ToggleUserFold flips ONE user bubble between its folded shape (head
// rows + the "… +N more lines · click to expand" hint) and its expanded
// shape (the full body + a "… collapse" trailer) — the ToggleThread twin
// for user turns: it mutates RENDER state only, so it rides forceRender
// exactly like the thread toggle (the SetState revision gate compares
// state, not expansion — without the force the click would look dead).
func (c *Chat) ToggleUserFold(id string) {
	if c.userExpanded == nil {
		c.userExpanded = map[string]bool{}
	}
	c.userExpanded[id] = !c.userExpanded[id]
	c.forceRender()
}

// threadExpandedNow — the thread's effective expansion under the same
// rule renderWorkerGroup applies: COLLAPSED BY DEFAULT (a live, busy
// thread folds to its collapsed header + sneak frame like any other —
// the live half is gone); a set per-agent override wins outright, a /stop
// stopped thread force-collapses until an explicit expand re-opens it,
// else the ctrl+g baseline decides.
func (c *Chat) threadExpandedNow(name string) bool {
	if v, ok := c.threadExpand[name]; ok {
		return v
	}
	if c.threadStop[name] {
		return false
	}
	return c.threadsExpanded
}

// threadEffectivelyExpanded — the SAME expanded/collapsed rule
// renderWorkerGroup renders by, evaluated for one thread: a set per-agent
// override wins outright, a /stop stopped thread force-collapses until an
// explicit expand re-opens it, else the ctrl+g baseline decides. The
// active/lastTick liveness pair is UNREAD here now (collapsed by default,
// live threads too) — it still feeds the loading row via the caller.
func (c *Chat) threadEffectivelyExpanded(name string, active bool, lastTick int) bool {
	if v, ok := c.threadExpand[name]; ok {
		return v
	}
	if c.threadStop[name] {
		return false
	}
	return c.threadsExpanded
}

// expandedThreadNames — the agent names whose worker threads render
// EXPANDED right now, in first-appearance chat order. The /tools +
// /thinking visibility switches mirror renderConversation's: a thread
// whose lines are all hidden has no row on screen to fold.
func (c *Chat) expandedThreadNames() []string {
	var names []string
	seen := map[string]bool{}
	lastTick := map[string]int{}
	for _, m := range c.chat {
		switch m.Kind {
		case wtoolKind:
			if !c.showTools {
				continue
			}
		case wthinkKind:
			if !c.showThinking {
				continue
			}
		default:
			continue
		}
		if !seen[m.From] {
			seen[m.From] = true
			names = append(names, m.From)
		}
		if _, tk := parseWtoolMeta(m.Meta); tk > lastTick[m.From] {
			lastTick[m.From] = tk
		}
	}
	var expanded []string
	for _, name := range names {
		active := false
		if av, ok := c.agents[name]; ok {
			active = av.active
		}
		if c.threadEffectivelyExpanded(name, active, lastTick[name]) {
			expanded = append(expanded, name)
		}
	}
	return expanded
}

// removeString filters every occurrence of s out of slice, order kept.
func removeString(slice []string, s string) []string {
	out := slice[:0]
	for _, v := range slice {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

// collapseLastThread is the esc/↑ "back one" gesture: fold the MOST
// RECENTLY expanded worker thread back to its collapsed header frame and
// report true (the key is spent). The fold is the same mutation as
// clicking the thread's header row — a per-agent ExpandThread(false)
// override — so a ctrl+g-expanded thread folds exactly like an explicitly
// opened one, every other thread keeps its rendering, and a later click
// re-opens it as always. The ledger knows the explicit expands; a thread
// expanded by the ctrl+g baseline has no entry, so it joins oldest-first
// (chat order) and its freshest expansion ends up on top of the stack.
// With NO thread expanded nothing is mutated and false is returned — the
// key falls through to exactly its old path (no sticky state).
func (c *Chat) collapseLastThread() bool {
	expanded := c.expandedThreadNames()
	if len(expanded) == 0 {
		return false
	}
	open := make(map[string]bool, len(expanded))
	for _, name := range expanded {
		open[name] = true
	}
	// prune the ledger to threads still expanded (one may have folded on
	// its own — the live agent went quiet, a ctrl+g blanket collapse…)
	tracked := make(map[string]bool, len(c.threadExpandOrder))
	order := c.threadExpandOrder[:0]
	for _, name := range c.threadExpandOrder {
		if open[name] {
			order = append(order, name)
			tracked[name] = true
		}
	}
	for _, name := range expanded {
		if !tracked[name] {
			order = append(order, name)
		}
	}
	c.threadExpandOrder = order
	c.ExpandThread(order[len(order)-1], false)
	return true
}

// ThreadRowAt is ClickRow's READ-ONLY twin for the worker-thread frame
// rows: it reports WHICH agent's thread claims content coords (x, y) —
// the threadRows hit-map (collapsed: header + sneak rows; expanded:
// header + closing-summary rows) — WITHOUT toggling anything. The
// floating cards keep ClickRow's precedence (a point a question /
// permission / picker card swallows never resolves to the thread
// underneath), and out-of-viewport or unclaimed rows answer ("", false).
// The app uses it to OPEN the clicked thread's transcript in the
// thread-focus pane; the panel's own inline toggle (ClickRow, ctrl+g,
// the floor double-click) is untouched.
func (c *Chat) ThreadRowAt(x, y int) (string, bool) {
	if c.cardClaims(x, y) {
		return "", false
	}
	if y < 0 || y >= c.vp.Height() {
		return "", false
	}
	name, ok := c.threadRows[y+c.vp.YOffset()]
	return name, ok
}

// BtwPinRowAt reports whether the chat-content point's row is a hidden-BTW
// pin bubble. The app uses this read-only lookup to resume the hidden session
// before ClickRow claims the pin row from transcript selection.
func (c *Chat) BtwPinRowAt(x, y int) bool {
	if y < 0 || y >= c.vp.Height() {
		return false
	}
	line := y + c.vp.YOffset()
	_, ok := c.btwPinRows[line]
	return ok
}

// ClickRow handles a mouse click at (x, y) IN CHAT CONTENT COORDS
// (viewport row 0 at the top of the chat panel; the app translated the
// screen coords over the tab/border chrome). While a QUESTION popover is
// open its card claims every click inside its frame FIRST (option rows
// answer/toggle via PermClick → questionClick — the question owns the
// float slot, so no permission can be visible under it); when the
// permission popover is visible its hit-map wins next: the card is a
// fixed (scroll-independent) overlay, so its rows are plain content-row
// absolutes off the same geometry the View splices with — every click
// inside the card frame is claimed here (swallowed: option rows answer
// via PermClick) so a click can never leak through to a thread
// underneath. A hit on a worker thread's HEADER row (the spinner/✓/✗
// title line — a single clipPlain-elided row, no wrapped continuations)
// toggles that agent's thread — collapsed, the ↳ sneak row toggles too;
// expanded, the CLOSING summary rows toggle instead (whole-bubble means
// clicking any frame row of the bubble — its head or its tail — folds
// it back). A hit on a user bubble's fold row ("… +N more lines · click
// to expand" / "… collapse") toggles that bubble's fold. All lookups are
// y-only: the row maps carry full rows, no x math.
// Returns true when the click was claimed.
func (c *Chat) ClickRow(x, y int) bool {
	if c.cardClaims(x, y) {
		return true // the floating card swallows every click in its frame
	}
	if y < 0 || y >= c.vp.Height() {
		return false
	}
	line := y + c.vp.YOffset()
	if name, ok := c.threadRows[line]; ok {
		c.ToggleThread(name)
		return true
	}
	// an expanded thread's "↳ diff · path +A -D" sub-row toggles THAT
	// wdiff's parsed body open/closed — checked right after the thread's
	// frame rows; the body rows themselves fall through unclaimed.
	if id, ok := c.toolDiffRows[line]; ok {
		c.ToggleThreadDiff(id)
		return true
	}
	// a tool one-liner's own rows toggle ITS output body (the boss's
	// inline rows and an expanded thread's wtool rows alike — a thread's
	// frame rows won above, its content rows never registered there);
	// the expanded body rows never register, so they fall through.
	if id, ok := c.toolRows[line]; ok {
		c.ToggleToolOutput(id)
		return true
	}
	// the user-bubble fold rows: a collapsed bubble's "… +N more lines"
	// hint expands it, an expanded bubble's "… collapse" trailer folds it
	// back — y-only lookup, same row-coordinate seam as the threads
	if id, ok := c.userFoldRows[line]; ok {
		c.ToggleUserFold(id)
		return true
	}
	if _, ok := c.btwPinRows[line]; ok {
		return true // claimed — the app routes btw-pin clicks
	}
	return false
}

// cardClaims reports whether the chat-content point (x, y) lands inside a
// floating card's frame (question popover, permission popover, /session
// picker) — those clicks belong to the card path, NOT to thread/fold
// toggles and NOT to text selection.
func (c *Chat) cardClaims(x, y int) bool {
	if c.question != nil {
		top, left, cardW, rows, _ := c.questCardGeom()
		if y >= top && y < top+len(rows) && x >= left && x < left+cardW {
			return true // the question card swallows every click in its frame
		}
	}
	if c.permVisible() {
		top, left, cardW, rows, _ := c.permCardGeom()
		if y >= top && y < top+len(rows) && x >= left && x < left+cardW {
			return true // the card swallows every click inside its frame
		}
	}
	if c.sessPick != nil {
		top, left, cardW, rows := c.sessCardGeom()
		if y >= top && y < top+len(rows) && x >= left && x < left+cardW {
			return true // the picker card swallows clicks (keys-only picker)
		}
	}
	if c.openPick != nil {
		top, left, cardW, rows := c.linkCardGeom()
		if y >= top && y < top+len(rows) && x >= left && x < left+cardW {
			return true // the target card swallows clicks (keys-only picker)
		}
	}
	return false
}

// SetDiffsExpanded shows diffs expanded (on) or collapsed (off) —
// /diffs on|off.
func (c *Chat) SetDiffsExpanded(on bool) {
	if c.diffExpanded == on {
		return
	}
	c.diffExpanded = on
	c.forceRender()
}

// DiffsExpanded reports whether diff entries render expanded.
func (c *Chat) DiffsExpanded() bool { return c.diffExpanded }

// SetEnqueue wires the app's enqueue callback — for ROADBLOCKS only (Enter
// while a question hold is outstanding / a permission modal is open). A
// plain busy turn does NOT enqueue: it free-sends via onBusySend. Staged
// attachments ride along so a queued prompt keeps its files.
func (c *Chat) SetEnqueue(fn func(text string, atts []state.Attachment) tea.Cmd) { c.onEnqueue = fn }

// SetBusySend wires the app's FREE-SEND callback (free-queuing / the
// anti-stuck flow): Enter while a boss reply is pending sends DIRECTLY to
// the backend — the serve queues the prompt server-side and drains it
// after the current turn, so the prompt never hides in a client queue.
func (c *Chat) SetBusySend(fn func(text string, atts []state.Attachment) tea.Cmd) { c.onBusySend = fn }

// SetServerTurn hands the panel the app's free-queuing tally for the
// current busy turn (0 = none / idle again). The busy placeholder reflects
// it (from the second send on: "<boss>: turn N · your message rides next").
func (c *Chat) SetServerTurn(n int) {
	if c.serverTurn == n {
		return
	}
	c.serverTurn = n
	c.refreshPlaceholder()
}

// MarkThreadStopped force-collapses one agent's worker thread (the /stop
// unwind): its summary reads "✗ stopped" until an explicit per-agent
// expand (mouse click / double-click) re-opens the rows.
func (c *Chat) MarkThreadStopped(name string) {
	if c.threadStop == nil {
		c.threadStop = map[string]bool{}
	}
	if c.threadStop[name] {
		return
	}
	c.threadStop[name] = true
	c.forceRender()
}

// SetPermissionHandlers wires the app's permission answer/defer
// callbacks: onPermAnswer fires for the popover's option rows (click),
// enter-confirmed selection, and the y/a/n quick keys — with the strings
// "once"|"always"|"reject"; onPermLater fires for the esc defer.
func (c *Chat) SetPermissionHandlers(answer func(response string) tea.Cmd, later func() tea.Cmd) {
	c.onPermAnswer, c.onPermLater = answer, later
}

// SetPermission opens (non-nil) or closes (nil) the permission popover —
// a floating, centered card spliced over the assembled chat view (the
// textarea keeps rendering + typing underneath; see perm_modal.go).
// Opening closes the @ picker + slash popover (their nav keys would
// otherwise fight the popover's arrows/enter) and resets the menu
// cursor to the first option; staged chips stay — they belong to the
// draft. While a question modal is open the popover stays QUEUED here
// (it renders once the question closes) — the question owns the keys.
func (c *Chat) SetPermission(p *PermissionView) {
	if p != nil {
		c.closeAttachPicker()
		c.closeSlashPicker(true) // the popover owns the choice keys now
		c.permSel = 0            // a fresh ask opens on "Allow once"
	}
	c.perm = p
}

// SetQuestionHandlers wires the app's question answer/defer callbacks:
// the popover submits a QuestionAnswer per page (Picks for option rows
// — one label for radio/confirm, every toggle for checkbox — Text for
// the TEXT page and the custom-answer row; → AnswerQuestion), esc
// defers (/question re-opens).
func (c *Chat) SetQuestionHandlers(answer func(a QuestionAnswer) tea.Cmd, later func() tea.Cmd) {
	c.onQuestionAnswer, c.onQuestionLater = answer, later
}

// SetStopHandler wires the app's double-esc interrupt seam: two esc
// presses inside dblEscWindow in the main chat input fire it (the app's
// /stop abort path). Unset (or with nothing running), esc-esc is a
// harmless no-op.
func (c *Chat) SetStopHandler(fn func() tea.Cmd) { c.onStopEsc = fn }

// SetQuestion opens (non-nil) or closes (nil) the boss question popover —
// a floating, centered yellow card spliced over the assembled chat view
// (question_modal.go; the textarea keeps rendering UNDER it but is
// disabled — the popover owns every key while open). Opening closes the
// @ picker + slash popover (their nav/typing keys would otherwise fight
// the popover's) and resets the page state: the menu cursor lands on the
// first option, checkbox toggles size to the option count, the text
// buffer empties. NO row-budget change: the card splices at render time,
// SetSize needs no re-run (unlike the old region-replacing modal).
func (c *Chat) SetQuestion(q *QuestionView) {
	c.question = q
	if q != nil {
		c.qSel = 0
		c.qText = ""
		c.qPicked = make([]bool, len(q.Options))
		c.closeAttachPicker()
		c.closeSlashPicker(true)
	}
}

// SetQuestionWaiting marks whether a boss question hold is outstanding
// (open or esc-deferred). While waiting the placeholder reads "boss is
// waiting for your answer…" and Enter ENQUEUES — the turn is parked at
// the question reply API, not typing.
func (c *Chat) SetQuestionWaiting(on bool) {
	c.questionWaiting = on
	c.refreshPlaceholder()
}

// SetQueueLen updates the queue count shown in the busy placeholder
// ("<boss> is typing… · N queued"). The queue itself lives in the app model.
func (c *Chat) SetQueueLen(n int) {
	c.queueLen = n
	c.refreshPlaceholder()
}

// SetBossShortName personalizes the busy placeholder and the typing spinner
// line from the config boss name's first word ("jorge is typing…"). Empty
// restores "boss".
func (c *Chat) SetBossShortName(name string) {
	if c.bossShort == name {
		return
	}
	c.bossShort = name
	c.refreshPlaceholder()
}

// typingText — the busy-state wording: "<bossShort> is typing…".
func (c *Chat) typingText() string {
	name := c.bossShort
	if name == "" {
		name = defaultBossShort
	}
	return name + " is typing…"
}

// delegatingText — the P3 delegation wording in place of the typing
// spinner: "<bossShort>: delegating · <n> busy".
func (c *Chat) delegatingText() string {
	name := c.bossShort
	if name == "" {
		name = defaultBossShort
	}
	return name + ": delegating · " + itoa(c.delegatingN) + " busy"
}

// turnText — the busy placeholder's free-queuing wording once a second
// prompt is already queued server-side this turn ("boss: turn 2 · your
// message rides next").
func (c *Chat) turnText() string {
	name := c.bossShort
	if name == "" {
		name = defaultBossShort
	}
	return name + ": turn " + itoa(c.serverTurn) + " · your message rides next"
}

// refreshPlaceholder recomputes the textarea placeholder from pending +
// queue + question-hold + server-turn state. A parked question turn
// (WAITING, not typing) wins over the typing text in both wording and
// queue badge; the free-queuing turn text wins over the plain typing text
// from the second direct send on.
func (c *Chat) refreshPlaceholder() {
	base := ""
	switch {
	case c.questionWaiting:
		base = placeholderWait
	case c.pending:
		base = c.typingText()
		if c.serverTurn >= 2 {
			base = c.turnText()
		}
	default:
		c.ta.Placeholder = placeholderIdle
		return
	}
	if c.queueLen > 0 {
		c.ta.Placeholder = base + " · " + itoa(c.queueLen) + " queued"
		return
	}
	c.ta.Placeholder = base
}

// chatMediaView — ONE inbound boss-turn image as the bubble renders it:
// the painted raster rows (nil until the lazy probe lands — the chip
// alone meanwhile), a native-lane escape frame (kitty strip / OSC 1337),
// or failed=true (the dim "unsupported image" copy).
//
// NATIVE-FRAME ROUTING (the wave-86 splice): a KITTY strip (kitty=true,
// id = the parsed i= word) never rides the View string — bubbletea's
// renderer decodes the View into cells and DROPS zero-width APCs (the
// wave-81 forensics), so the strip only bloated the differ and never
// painted. Instead renderMediaRows emits PURE reservation rows and the
// frame SPLICES through the ZenbuFrameWriter (zenbu_frame.go's chat
// region — the app publishes the absolute cell per Frame). An OSC 1337
// frame carries no id and iTerm2 has no image-delete escape, so it
// keeps the OLD embedded-row behavior (the splice's emitted-set diff
// could never target it — a scrolled-off image would ghost forever).
type chatMediaView struct {
	rows     []string // ASCII lane: the half-block truecolor rows
	frame    string   // native lane: ONE atomic escape frame
	id       uint32   // kitty lane: the strip's parsed i= id (0 = unparseable → embed)
	kitty    bool     // the frame is a kitty strip (splice-routed); OSC 1337 stays embedded
	cellRows int      // native lane: the reserved cell-box height (frame row included)
	failed   bool
}

// SetImageRaster pushes ONE image's finished raster rows into the panel
// (the app's rasterize landing). Keyed by msgID+hash — the carrier's own
// identity (state.ParseMediaMeta reads the same hash at render). Push is
// idempotent, and the media-revision bump re-renders exactly the owning
// bubble's block.
func (c *Chat) SetImageRaster(msgID, hash string, rows []string) {
	if msgID == "" || hash == "" || len(rows) == 0 {
		return
	}
	if c.media == nil {
		c.media = map[string]map[string]chatMediaView{}
	}
	if c.mediaRev == nil {
		c.mediaRev = map[string]uint64{}
	}
	if c.media[msgID] == nil {
		c.media[msgID] = map[string]chatMediaView{}
	}
	c.media[msgID][hash] = chatMediaView{rows: append([]string(nil), rows...)}
	c.mediaRev[msgID]++
	c.forceRender()
}

// SetImageFrame pushes ONE image's native-lane escape frame into the
// panel (the kitty placeholder strip or the OSC 1337 sequence — the
// probe's lane pick landed). cellRows is the reserved cell-box height the
// frame visually spends (the SAME box the ASCII lane would occupy), so
// renderMediaRows pads the bubble identically on every lane. The frame
// is stored VERBATIM and never folded; push is idempotent and bumps the
// owning bubble's media revision, exactly like SetImageRaster. The
// kitty-ness + the i= id are parsed ONCE here (kittyFrameID) — the
// splice routing + the registry publish read them off the stored view.
func (c *Chat) SetImageFrame(msgID, hash, frame string, cellRows int) {
	if msgID == "" || hash == "" || frame == "" || cellRows < 1 {
		return
	}
	if c.media == nil {
		c.media = map[string]map[string]chatMediaView{}
	}
	if c.mediaRev == nil {
		c.mediaRev = map[string]uint64{}
	}
	if c.media[msgID] == nil {
		c.media[msgID] = map[string]chatMediaView{}
	}
	id, kitty := kittyFrameID(frame)
	c.media[msgID][hash] = chatMediaView{frame: frame, id: id, kitty: kitty, cellRows: cellRows}
	c.mediaRev[msgID]++
	c.forceRender()
}

// SetImageFailed latches ONE image as undecodable (the rasterize probe
// came back empty): the chip swaps to the dim "unsupported image" copy.
func (c *Chat) SetImageFailed(msgID, hash string) {
	if msgID == "" || hash == "" {
		return
	}
	if c.media == nil {
		c.media = map[string]map[string]chatMediaView{}
	}
	if c.mediaRev == nil {
		c.mediaRev = map[string]uint64{}
	}
	if c.media[msgID] == nil {
		c.media[msgID] = map[string]chatMediaView{}
	}
	c.media[msgID][hash] = chatMediaView{failed: true}
	c.mediaRev[msgID]++
	c.forceRender()
}

// SetShowThinking shows/hides collected thinking blocks (/thinking on|off).
func (c *Chat) SetShowThinking(on bool) {
	if c.showThinking == on {
		return
	}
	c.showThinking = on
	c.forceRender()
}

// ShowThinking reports whether thinking blocks render.
func (c *Chat) ShowThinking() bool { return c.showThinking }

// SetStreamingThink hands the panel the live think-stream set — CallIDs
// whose boss EvThought hasn't seen Done yet. Called by the app right
// before SetState (render consults the set there). Defensive copy: the
// caller keeps mutating its own map.
func (c *Chat) SetStreamingThink(ids map[string]bool) {
	if len(ids) == 0 {
		c.streamingThink = nil
		return
	}
	next := make(map[string]bool, len(ids))
	for id := range ids {
		next[id] = true
	}
	c.streamingThink = next
}

// SetShowTools shows/hides tool one-liners (/tools on|off).
func (c *Chat) SetShowTools(on bool) {
	if c.showTools == on {
		return
	}
	c.showTools = on
	c.forceRender()
}

// ShowTools reports whether tool one-liners render.
func (c *Chat) ShowTools() bool { return c.showTools }

// setConversationLines posts the rendered transcript into the viewport:
// every content line rides the chatPadL left gutter (the right gutter
// falls out of the contentW wrap budget, not padding — the block cache
// carries the lines pre-padded), while rows outside the viewport —
// divider, typing/loading rows, chips, pickers, textarea — keep the full
// panel width. The padded lines are ALSO the mouse-selection's coordinate
// space (selLines cache) and hijack-free extraction source: while a
// selection is live the reverse-video overlay splices over them HERE (the
// one seam every render path flows through), so the highlight survives
// SetState rebuilds, fold/thread toggles and scroll alike (see
// chat_selection.go).
func (c *Chat) setConversationLines(lines []string) {
	// ROGUE-ROW RESCUE: vp.SoftWrap is off (chat_window.go), so a padded
	// line wider than the viewport would clip at the terminal edge —
	// where the old SoftWrap viewport hard-cut it into wrapped rows.
	// Renderers never produce one (they fold at contentW ≤ w-chatPadR);
	// the empty-state placeholder at a hairline width can. When the
	// assembly's convWide even REACHES the viewport width, cut the
	// offenders into SoftWrap's exact chunks (expandLines) so the posted
	// rows stay ≤ width — the blank-height model then holds (every posted
	// row is 1 cell tall) AND the pixels match the old behavior. The
	// guard skips the scan entirely when nothing can overflow (free).
	if vpW := c.vp.Width(); vpW > 0 && c.convWide > vpW {
		out := make([]string, 0, len(lines))
		for _, ln := range lines {
			out = append(out, expandLines(ln, vpW)...)
		}
		lines = out
	}
	c.selLines = lines
	if c.sel.active || c.sel.finalized {
		c.selOverlay(lines)
	}
	// the windowed projection: post the SAME row count as selLines (the
	// blank-height model — scroll space, ClickRow rows, TranscriptRows()
	// all unchanged) with real content only around the scroll anchor
	// (chat_window.go). The caller's follow-latch GotoBottom lands inside
	// the freshly materialized window by construction (rebuildWindow aims
	// at maxYOffset when following).
	c.rebuildWindow()
}

// forceRender re-renders the conversation outside the SetState revision gate
// (toggles change the pixels, not the state).
func (c *Chat) forceRender() {
	c.setConversationLines(c.renderConversationLines())
	if c.follow {
		c.vp.GotoBottom()
	}
}

// Title implements Tab.
func (c *Chat) Title() string { return "chat" }

// Pending reports whether a boss reply is outstanding.
func (c *Chat) Pending() bool { return c.pending }

// SpinnerKick returns the cmd that RE-ARMS the braille-spin animation. The
// app fires it on the BOSS-PENDING flip: applyEvent (internal/app/
// model.go) calls it exactly when the office state goes from NO pending
// boss bubble to one (!prevPending && hasPendingBoss) — the moment a
// delegation burst's first LIVE worker-thread header appears too. The
// returned c.sp.Tick schedules the first spinner.TickMsg; the panel's own
// Update arm answers each one with sp.Update, but re-arms the chain ONLY
// while spinnerLive holds — a settled session emits ZERO spinner ticks
// (the power governor's idle duty: each tick used to bump frameNonce → a
// full Frame() at 10k-line transcripts every ~83ms forever).
func (c *Chat) SpinnerKick() tea.Cmd { return c.sp.Tick }

// spinnerLive — the liveness set the spinner chain gates on: every glyph
// that visibly animates while the chain runs. The set is the same one the
// loading/typing rows read:
//   - c.anyThreadActive() — the "team is working…" loading row + every LIVE
//     worker-thread header (chat_loading.go: roster sprite busy + freshest
//     wtool/wthink meta-tick inside wtoolStaleTicks). The header braille
//     itself is a pure function of the office tick now (threadLiveGlyph),
//     so gating the chain changes no pixels while live.
//   - c.pendingSpin — the "<boss> is typing…" row below the divider,
//     shown for the whole boss-pending period (SetState).
//
// When BOTH are quiet nothing animates: the next Tick is withheld and the
// chain stops. A later boss-pending flip re-arms via SpinnerKick.
func (c *Chat) spinnerLive() bool {
	return c.pendingSpin || c.anyThreadActive()
}

// inputRows is the textarea's visible height: textareaH normally, trimmed
// to 2 rows in the /compact layout. The permission popover is a floating
// overlay now (not a region), so it no longer bumps the input rows when
// it opens.
func (c *Chat) inputRows() int {
	if c.compactRows {
		return 2
	}
	return textareaH
}

// SetCompact switches the input region between the full 3-row textarea and
// the compact 2-row one (the app calls it on /compact + /mode changes).
func (c *Chat) SetCompact(on bool) {
	if c.compactRows == on {
		return
	}
	c.compactRows = on
	c.ta.SetHeight(c.inputRows())
	c.SetSize(c.w, c.h) // viewport takes/relinquishes the extra row
}

// SetSize implements Tab: splits content height across viewport / divider /
// spinner / bottom region. The typing (spinner) row sits BELOW the divider,
// one row while pendingSpin. The bottom region is ALWAYS the textarea now
// — BOTH the permission popover and the question popover are overlays:
// they budget no rows, they splice over the assembled view at render time
// instead — PLUS the attachment surfaces above it: the chips row(s) when
// files are staged, and the @ picker box while open. Chips and picker
// consume rows, so the viewport shrinks for them instead of overlapping.
func (c *Chat) SetSize(w, h int) {
	if w < 4 {
		w = 4
	}
	wChanged := w != c.w
	c.w, c.h = w, h
	spH := 0
	if c.pendingSpin {
		spH = 1
	}
	regionH := c.inputRows()
	regionH += c.chipsH() + c.popoverH() + c.slashH()
	// the "team is working" row takes one line while any worker thread is
	// live — zero otherwise (the row self-hides, so vpH must follow suit)
	ldH := 0
	if c.anyThreadActive() {
		ldH = 1
	}
	vpH := h - regionH - 1 /* divider */ - spH - ldH
	if vpH < 1 {
		vpH = 1
	}
	c.vp.SetWidth(w)
	c.vp.SetHeight(vpH)
	c.ta.SetWidth(w)
	// cellWidth, not len: bossPrefix's "›" is 3 bytes but 1 cell — the
	// byte count would rob the wrap of 2 columns. The budget runs off
	// contentW(): the transcript pads chatPadL/chatPadR cells inside the
	// viewport (setConversation), so bubbles wrap tighter by both.
	c.mdWidth = c.contentW() - cellWidth(bossPrefix) - 1
	if c.mdWidth < 10 {
		c.mdWidth = 10
	}
	c.md = nil // rebuilt lazily at the new wrap width
	// A width CHANGE re-plans the posted transcript NOW (not only at the
	// next SetState): the window re-posts at the current scroll offset
	// and rogue-wide rows re-cut (the same rows the old SoftWrap viewport
	// served pre-render). Rendering stays lazy — the cached blocks are
	// re-JOINED, never re-rendered, here (the next SetState's generation
	// flip misses them all anyway).
	if wChanged && len(c.blocks) > 0 {
		c.setConversationLines(c.assembleConversationLines())
		if c.follow {
			c.vp.GotoBottom()
		}
	}
}

// cellWidth is the display-cell count of s (runes here — prefixes are all
// single-cell glyphs; lipgloss.Width on a styled/cursed string belongs to
// the ansi-aware foldStyledLines instead).
func cellWidth(s string) int { return len([]rune(s)) }

// contentW — the transcript's text budget: the panel width minus the
// chatPadL/chatPadR insets. Every transcript width source (markdown wrap,
// tool one-liners, question bubbles, diff rows, thread header/sneak/
// expanded/closing/wthink clips) wraps or clips to this so the padded
// viewport keeps its right gutter.
func (c *Chat) contentW() int { return c.w - chatPadL - chatPadR }

// SetState implements Tab: keeps the latest chat slice, re-renders the
// conversation when it changed, and keeps scroll pinned to the bottom.
// Also captures the roster rollup (workers-thread decoration) and the
// delegation state (P3 spinner-row swap) from the incoming state.
func (c *Chat) SetState(st state.OfficeState) {
	c.tick = st.Tick
	agents := make(map[string]agentView, len(st.Employees))
	busy := 0
	for _, e := range st.Employees {
		if e.Role == state.RoleManager {
			continue
		}
		av := agentView{task: e.Task, active: workerSpriteActive(e.Sprite), role: e.Role}
		agents[e.Name] = av
		if av.active {
			busy++
		}
	}
	c.agents = agents
	c.delegating = st.BossDelegating
	c.delegatingN = busy
	if c.workerTasks == nil {
		c.workerTasks = map[string]string{}
	}
	for _, e := range st.Employees {
		if e.Role != state.RoleManager && e.Task != "" {
			c.workerTasks[e.Name] = e.Task // sticky: survives EvReturned's task clear
		}
	}
	rev := revision(st.Chat)
	// fold the stream set into the revision (order-independent) so closing a
	// stream re-renders even when Done's final text equals the last update.
	srev := uint64(14695981039346656037)
	for id := range c.streamingThink {
		h := uint64(14695981039346656037)
		for i := 0; i < len(id); i++ {
			h ^= uint64(id[i])
			h *= 1099511628211
		}
		srev ^= h
	}
	rev ^= srev
	// fold the roster rollup + delegation flag in (order-independent) — a
	// dispatch filling a thread header's task or a delegating flip changes
	// pixels without touching the chat slice.
	arev := uint64(14695981039346656037)
	for name, a := range c.agents {
		h := uint64(14695981039346656037)
		for _, s := range []string{name, a.task} {
			for i := 0; i < len(s); i++ {
				h ^= uint64(s[i])
				h *= 1099511628211
			}
		}
		if a.active {
			h ^= 1
			h *= 1099511628211
		}
		arev ^= h
	}
	rev ^= arev
	if c.delegating {
		rev ^= 0x9e3779b97f4a7c15
	}
	rev ^= uint64(uint32(c.delegatingN))
	// a streaming boss bubble does NOT need per-tick re-renders: the old
	// blinking caret is gone — every pixel a pending bubble draws comes
	// straight from its text, and text changes always land in rev (when a
	// partial survives into a later tick its own diff re-triggers). Think
	// streams DO still animate per tick — see len(c.streamingThink) below.
	// a worker thread within its staleness horizon re-renders every tick —
	// the liveness horizon the loading row reads is tick-relative and
	// invisible to rev (the thread frames themselves are stable per rev
	// now: collapsed-by-default killed the tick-relative expand boundary).
	wtoolRecent := false
	for _, m := range st.Chat {
		if m.Kind != wtoolKind && m.Kind != wthinkKind {
			continue
		}
		_, tk := parseWtoolMeta(m.Meta)
		if c.tick-tk <= wtoolStaleTicks+2 {
			wtoolRecent = true
			break
		}
	}
	if rev == c.renderRev && len(st.Chat) == len(c.chat) && len(c.streamingThink) == 0 && !wtoolRecent {
		return
	}
	c.renderRev = rev
	c.chat = cloneChat(st.Chat)
	c.pruneToolOutputs() // output captures die with their transcript rows (chat_toolrow.go)

	wasPending := c.pending
	wasSpin := c.pendingSpin
	c.pending, c.pendingSpin = false, false
	for _, m := range c.chat {
		if m.From == "boss" && m.Pending {
			// the typing row runs for the WHOLE pending period — the
			// streaming bubble shows the text, the row shows the life
			c.pending, c.pendingSpin = true, true
		}
	}
	if c.pending != wasPending {
		// the textarea NEVER locks — typing while the boss works is the
		// whole point of the queue; only the placeholder text reacts
		c.refreshPlaceholder()
	}
	if c.pendingSpin != wasSpin {
		c.SetSize(c.w, c.h) // typing row appears/disappears
	}

	if c.deferRender {
		// the open thread-focus view renders this same state itself —
		// the rev+snapshot above are recorded, so ResumeFromFocus's ONE
		// re-render lands at the latest state when the focus closes.
		return
	}
	c.setConversationLines(c.renderConversationLines())
	if c.follow {
		c.vp.GotoBottom()
	}
}

// SetDeferredRender arms/clears the focus deferral (the app drives it
// from its Model-owned focusDeferredRender flag; see deferRender above).
func (c *Chat) SetDeferredRender(on bool) { c.deferRender = on }

// RenderCalls — the renderConversation rebuild tally (the focus
// deferral's test probe: with the saver armed, SetState pulses must NOT
// move it; close must move it exactly once).
func (c *Chat) RenderCalls() int { return c.renderCalls }

// ResumeFromFocus — the thread-focus CLOSE: disarm the deferral and force
// EXACTLY ONE re-render at the latest snapshot. SetState's revision gate
// recorded every pulse's revision, so a fresh forceRender's twin (rebuild
// + respect the follow pin) is the whole catch-up — the member returns to
// the conversation byte-identical to where they left it.
func (c *Chat) ResumeFromFocus() {
	c.deferRender = false
	c.setConversationLines(c.renderConversationLines())
	if c.follow {
		c.vp.GotoBottom()
	}
}

// Update implements Interactive: Enter sends, Shift+Enter newline, wheel +
// pgup/pgdn + (single-line) arrows scroll the conversation. Paste follows
// the macOS OS defaults: cmd+v is the user's gesture, and it arrives three
// ways — kitty-protocol terminals report it as a "super+v" key, every
// other terminal converts it to bracketed paste (tea.PasteMsg), and an
// image-only clipboard delivers NOTHING in Terminal.app/iTerm2 (the
// app-side image trigger is the only probe path there). ctrl+v and
// super+v both probe the clipboard for an image (attach as a chip on a
// hit, replay text paste on a miss). tea.PasteMsg gets its own smart arm:
// a Finder file copy pastes as (escaped) path text and stages chips, an
// empty/whitespace-only paste on darwin probes for an image (the miss
// re-feeds the original bytes), everything else inserts as plain text.
// "@" at a word start opens the attach-file popover, which owns
// ↑/↓/enter/tab/esc while open (the question-modal precedence pattern) —
// every other key still reaches the textarea. Non-key messages the panel
// doesn't claim (the textarea's own clipboard pasteMsgs) fall through the
// default arm INTO the textarea.
func (c *Chat) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// While a boss question popover is open it OWNS EVERY KEY: the
		// turn is parked at the question reply API, so the main textarea
		// is DISABLED entirely — the popover's card edits its own inputs
		// (arrow/tab cursor, option toggles, free-text and custom-answer
		// buffers), enter/ctrl+enter submits the page's QuestionAnswer,
		// esc defers. PgUp/PgDn still scroll the transcript. (The old
		// region-replacing modal's key gate is question_modal.go's
		// questKey now — the arm is one line because the popover
		// carries its own per-kind switch.)
		if c.question != nil {
			return c.questKey(msg)
		}
		// While a permission popover is open its choice keys are RESERVED
		// (they never reach the textarea): up/down/tab walk the menu
		// cursor, enter confirms the highlighted option, y/a/n are the
		// quick answers regardless of the cursor, esc defers. Every OTHER
		// key keeps typing into the visible textarea below — the popover
		// floats, it does not lock. (The question modal above claims its
		// own keys first, so a queued permission can't steal here.)
		if c.perm != nil {
			switch msg.String() {
			case "up":
				c.permMove(-1)
				return nil
			case "down", "tab": // both walk the cursor forward
				c.permMove(1)
				return nil
			case "enter":
				if c.onPermAnswer != nil {
					return c.onPermAnswer(permOptions[c.permSel].response)
				}
				return nil
			case "y":
				if c.onPermAnswer != nil {
					return c.onPermAnswer("once")
				}
				return nil
			case "a":
				if c.onPermAnswer != nil {
					return c.onPermAnswer("always")
				}
				return nil
			case "n":
				if c.onPermAnswer != nil {
					return c.onPermAnswer("reject")
				}
				return nil
			case "esc":
				if c.onPermLater != nil {
					return c.onPermLater()
				}
				return nil
			}
		}
		// The /session picker owns EVERY key while open (question-modal
		// style — the textarea is disabled): typing narrows the session
		// list, enter accepts the highlighted row, esc cancels. It claims
		// AFTER the question/permission floats — a parked turn's modal
		// outranks browsing; the picker waits underneath and resumes when
		// the float clears.
		if c.sessPick != nil && c.question == nil && c.perm == nil {
			return c.sessKey(msg)
		}
		// The `o` target card owns EVERY key while open (the /session
		// picker's contract) and claims at the same rank: a parked turn's
		// modal outranks browsing, so it yields to the floats above. Its
		// own `o` presses are swallowed (the card, not the mark, owns
		// keys now); enter fires the app's open ferry, esc cancels.
		if c.openPick != nil && c.question == nil && c.perm == nil {
			return c.linkPickKey(msg)
		}
		// The @ popover owns its nav/attach keys while open (the question-
		// modal precedence pattern: claimed FIRST, nothing reaches the
		// textarea); every other key still goes to the textarea below.
		// The slash popover owns its nav/apply keys while open (claimed
		// first, like the modal/picker precedence above); every other key
		// still goes to the textarea below.
		if c.slashOpen {
			switch msg.String() {
			case "up":
				c.slashMove(-1) // theme mode: arrows PREVIEW live
				return nil
			case "down":
				c.slashMove(1)
				return nil
			case "enter", "tab":
				// a zero-match cmd-mode filter means the typed tail is a
				// command the popover does not KNOW (its menu is a curated
				// subset of applySlash's grammar): close the box and let
				// Enter fall through to the normal send path — sinking the
				// first Enter here would force a re-press.
				if msg.String() == "enter" && c.slashMode == slashModeCmd && c.slashCount() == 0 {
					c.closeSlashPicker(false)
					break
				}
				return c.slashPicked()
			case "esc":
				c.closeSlashPicker(true) // keeps the fragment; unwinds a preview
				return nil
			}
		}
		if c.atOpen {
			switch msg.String() {
			case "up":
				c.atMove(-1)
				return nil
			case "down":
				c.atMove(1)
				return nil
			case "enter", "tab":
				return c.attachPicked()
			case "esc":
				c.closeAttachPicker() // keeps the typed fragment
				return nil
			}
		}
		// esc with an open worker thread is "back one": fold the most
		// recently expanded thread to its summary row. The precedence
		// above is untouched — modals/pickers already claimed their own
		// esc — and with NO thread expanded the key falls through into the
		// switch below, whose "esc" case runs the double-esc interrupt
		// tracker (only an UNCONSUMED esc ever counts toward the pair).
		if msg.String() == "esc" && c.collapseLastThread() {
			return nil
		}
		switch msg.String() {
		case "ctrl+v", "super+v":
			// Image paste probe (async tea.Cmd): a clipboard holding an
			// image ATTACHES it as a chip; a text/empty clipboard replays
			// the textarea's own text paste (see onClipPaste). Two names,
			// one gesture: "super+v" is cmd+v as reported by kitty-
			// keyboard-protocol terminals (Terminal.app/iTerm2 swallow
			// CMD entirely — cmd+v there arrives as tea.PasteMsg below,
			// or as nothing at all when the clipboard is image-only,
			// making ctrl+v the only trigger that still reaches us).
			c.closeSlashPicker(true) // pasting is not fragment typing
			return c.startImagePaste()
		case "esc":
			// Double-esc = interrupt. The modal/picker arms above already
			// consumed their esc (close/defer), so only a MAIN-input esc
			// reaches here: a lone press is merely recorded — single-esc
			// in the input was (and stays) a no-op — and the second one
			// inside dblEscWindow fires the app's /stop abort path. The
			// key never reaches the textarea (it ignores esc anyway).
			now := time.Now()
			if c.onStopEsc != nil && !c.lastEscAt.IsZero() && now.Sub(c.lastEscAt) <= dblEscWindow {
				c.lastEscAt = time.Time{} // re-arm: a third esc opens a fresh pair
				return c.onStopEsc()
			}
			c.lastEscAt = now
			return nil
		case "backspace":
			// a collapsed paste chip left of the cursor deletes as ONE
			// unit (chat_paste.go) — never per-rune into the token's
			// middle, which would strand an unexpandable stub.
			if c.popPasteChipBeforeCursor() {
				return nil
			}
			if c.atOpen {
				// fragment editing: the picker lives/dies by the tail
				// recheck after the textarea eats the key
				var cmd tea.Cmd
				c.ta, cmd = c.ta.Update(msg)
				c.afterDraftEdit()
				return cmd
			}
			if c.slashOpen {
				// same tail-recheck contract — backspace over the lone "/"
				// breaks the match and closes the popover
				var cmd tea.Cmd
				c.ta, cmd = c.ta.Update(msg)
				c.afterSlashEdit()
				return cmd
			}
			// an EMPTY draft + staged chips: backspace pops the newest
			// attachment (the quiet undo for a mis-paste)
			if strings.TrimSpace(c.ta.Value()) == "" && len(c.atts) > 0 {
				return c.popAttachment()
			}
			var cmd tea.Cmd
			c.ta, cmd = c.ta.Update(msg)
			return cmd
		case "enter":
			// expand-on-send: the member sees the one-line chips, the
			// agent receives the FULL original paste (chat_paste.go).
			text := strings.TrimSpace(c.expandPasteChips(c.ta.Value()))
			if text == "" && len(c.atts) == 0 {
				return nil
			}
			c.ta.Reset()
			c.pasteChips = nil // the draft's chips die with the draft
			c.follow = true
			c.vp.GotoBottom()
			if strings.HasPrefix(text, "/") {
				// slash commands are local and always immediate — /queue
				// and /perm must work while the boss is typing. Chips are
				// NOT eaten: a stray /help must not sink a staged paste —
				// they belong to the next real prompt (/clear drops them).
				if c.onSend != nil {
					return c.onSend(text, nil)
				}
				return nil
			}
			atts := c.drainAttachments()
			// The client queue is ONLY for roadblocks: an outstanding
			// question hold (a plain Send would park the opencode loop at
			// the question reply API — the reported deadlock) or an open
			// permission modal. Everything else sends DIRECTLY — free-
			// queuing: while the boss is busy the prompt goes straight to
			// the backend, which queues it server-side and drains it after
			// the current turn (no hide, no client-side flush cadence).
			if (c.questionWaiting || c.perm != nil) && c.onEnqueue != nil {
				return c.onEnqueue(text, atts)
			}
			if c.pending && c.onBusySend != nil {
				return c.onBusySend(text, atts)
			}
			if c.onSend != nil {
				return c.onSend(text, atts)
			}
			return nil
		case "up", "down":
			// ↑ shares esc's "back one" gesture: an expanded worker
			// thread folds (most recent first) INSTEAD of scrolling;
			// with none expanded the key walks its untouched path —
			// multi-line draft move here, conversation scroll below.
			if msg.String() == "up" && c.collapseLastThread() {
				return nil
			}
			if c.ta.LineCount() > 1 {
				var cmd tea.Cmd
				c.ta, cmd = c.ta.Update(msg)
				return cmd
			}
			fallthrough
		case "pgup", "pgdown":
			var cmd tea.Cmd
			c.vp, cmd = c.vp.Update(msg)
			sawBottom := c.vp.AtBottom()
			if msg.String() == "down" || msg.String() == "pgdown" {
				if sawBottom {
					c.follow = true
				}
			} else {
				c.follow = false
			}
			c.syncWindow() // page the projection across the scrolled boundary
			return cmd
		default:
			// "@" at a word boundary opens the attach picker AFTER the
			// textarea takes the rune (the fragment starts empty); "/" at a
			// word boundary opens the slash popover the same way. Other
			// typed keys re-derive the fragment from the draft tail; nav
			// keys (arrows, ctrl-word ops, home/end) close the popovers —
			// they move the cursor off the tail the popovers track.
			opening := !c.atOpen && msg.Text == "@" && c.atWordBoundary()
			slashOpening := !c.slashOpen && msg.Text == "/" && c.atWordBoundary()
			var cmd tea.Cmd
			if c.atOpen && msg.Text == "" {
				c.closeAttachPicker()
			}
			if c.slashOpen && msg.Text == "" {
				c.closeSlashPicker(true)
			}
			c.ta, cmd = c.ta.Update(msg)
			if opening {
				return tea.Batch(cmd, c.openAttachPicker())
			}
			if slashOpening {
				return tea.Batch(cmd, c.openSlashPicker())
			}
			if c.atOpen && msg.Text != "" {
				c.afterDraftEdit()
			}
			if c.slashOpen && msg.Text != "" {
				c.afterSlashEdit()
			}
			return cmd
		}
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		c.vp, cmd = c.vp.Update(msg)
		c.follow = c.vp.AtBottom()
		c.syncWindow() // page the projection across the scrolled boundary
		return cmd
	case spinner.TickMsg:
		var cmd tea.Cmd
		c.sp, cmd = c.sp.Update(msg)
		if !c.spinnerLive() {
			// power governor: quiet session → the chain STOPS here (no
			// re-arm, zero 83ms ticks → zero frameNonce bumps → zero
			// full Frame() renders). SpinnerKick is the re-arm on the
			// next boss-pending flip.
			return nil
		}
		return cmd
	case clipPasteMsg:
		// the image probe answered (chat_attach.go) — fired by the
		// ctrl+v/super+v key trigger or the tea.PasteMsg reprobe below
		return c.onClipPaste(msg)
	case attachWalkMsg:
		// the @ picker's file walk answered
		c.onAttachWalk(msg)
		return nil
	case tea.PasteMsg:
		// Bracketed paste — on macOS this IS cmd+v (Terminal.app/iTerm2
		// swallow the CMD key and convert the paste themselves). The OWNED
		// surfaces claim it first, in the key path's exact precedence: the
		// boss question popover's answer field (the main textarea is
		// DISABLED while the turn is parked — before this arm a paste
		// silently landed in that dead textarea), then the /session
		// picker's filter (its Paste seam); the open-target card swallows
		// (its keys swallow too — no text surface).
		if c.question != nil {
			return c.questPaste(msg.Content)
		}
		if c.sessPick != nil && c.perm == nil {
			return c.sessPick.Paste(msg.Content)
		}
		if c.openPick != nil && c.perm == nil {
			return nil
		}
		// Smart classification, in order:
		if paths, ok := pasteFilePaths(msg.Content); ok {
			// 1) Finder file copy: the terminal delivered the copied
			//    file(s) as (escaped) path text — stage one chip per
			//    path, type nothing into the draft.
			return c.attachPastedFiles(paths)
		}
		if runtime.GOOS == "darwin" && strings.TrimSpace(msg.Content) == "" {
			// 2) Empty/whitespace-only paste on darwin: cmd+v with an
			//    image-only clipboard has no bytes to send — probe for
			//    image bytes; a miss re-feeds the ORIGINAL content into
			//    the textarea (see onClipPaste's reprobe arm).
			return c.startImagePasteReprobe(msg.Content)
		}
		// 3) plain text — LARGE pastes collapse to a one-line chip
		//    (chat_paste.go: the full text rides along and expands back
		//    on send); small pastes insert literally through the
		//    textarea's own PasteMsg arm — ONE batched insert, never the
		//    per-rune drain typing pays.
		if pasteChipThreshold(msg.Content) {
			c.insertPasteChip(msg.Content)
			return nil
		}
		var cmd tea.Cmd
		c.ta, cmd = c.ta.Update(msg)
		return cmd
	default:
		// The textarea's own internal pasteMsg/pasteErrMsg (its ctrl+v
		// clipboard path) are NOT key presses — everything unclaimed
		// forwards to the textarea so plain-text paste reaches the
		// input. (This arm is the paste regression fix: before it, all
		// of the above were dropped here. tea.PasteMsg itself moved UP
		// into its own smart arm above — Finder copies and image
		// clipboards now beat plain insertion.)
		var cmd tea.Cmd
		c.ta, cmd = c.ta.Update(msg)
		return cmd
	}
}

// View implements Tab. Below the divider (top → bottom): the typing row
// (while a boss reply is pending — spinner + "… is typing…", or the dim
// delegating line), the dim attachment chips row (when files are staged),
// the @ picker box (while open), then the input region — ALWAYS the
// textarea. The permission popover and the question popover are NOT
// regions: after everything assembles, each SPLICES its card rows over
// the result, cell by cell (perm_modal.go / question_modal.go), so the
// row budget SetSize computed never changes. A question popover hides
// the permission popover (permVisible) — it owns the float slot and
// every key; its textarea underneath keeps rendering but is disabled.
func (c *Chat) View() string {
	// the catch-all window sync: scroll seams the panel does not own
	// (threads_opencode.go's PreserveAnchor bump, app-level Goto*)
	// must paint on materialized rows even though no key came through
	// Update. O(1) when the viewport sits inside the overscan window.
	c.syncWindow()
	var b strings.Builder
	b.WriteString(c.vp.View())
	b.WriteString("\n")
	// live worker threads → the colorful "team is working" row, glued to
	// the input right above the divider; "" (self-hidden) when idle and
	// SetSize has already released its row
	if row := c.loadingRow(c.w); row != "" {
		b.WriteString(row)
		b.WriteString("\n")
	}
	b.WriteString(chrome.PanelDim.Render(fitPlain(strings.Repeat("─", c.w), c.w)))
	if c.pendingSpin {
		// the typing row — glued to the input, not the transcript: the
		// viewport holds the words, this row holds the pulse. ONE row
		// exactly: the breathing block-glyph bar (opencode's "Build · …"
		// vibe, pendingBlockBar — height blocks only, "▌" would be the
		// retired caret) + the same busy text as ever.
		b.WriteString("\n")
		if c.delegating {
			// P3 — the boss dispatched out and went quiet: a settled dim
			// delegation row, NO spinner blinking for minutes
			b.WriteString(chrome.PanelDim.Render(" " + c.delegatingText()))
		} else {
			b.WriteString(threadSpinnerStyle().Render(pendingBlockBar(c.tick)))
			b.WriteString(chrome.PanelAccent.Render(" " + c.typingText()))
		}
	}
	b.WriteString("\n")
	if chips := c.renderAttachChips(); chips != "" {
		// staged attachments sit right above the input they will attach to
		b.WriteString(chips)
		b.WriteString("\n")
	}
	if c.atOpen {
		b.WriteString(c.renderAttachPopover())
		b.WriteString("\n")
	}
	if c.slashOpen {
		b.WriteString(c.renderSlashPopover())
		b.WriteString("\n")
	}
	b.WriteString(c.ta.View())
	out := b.String()
	if c.permVisible() || c.question != nil || c.sessPick != nil || c.openPick != nil {
		// each open FLOAT splices its card rows over the assembled lines
		// (textarea rows included): cells are replaced, never lines —
		// the layout never jumps. Only one alert float renders at a time:
		// permVisible is false while a question owns the slot. The session
		// picker splices FIRST (bottom) — a permission/question card
		// popping over it wins the top while it waits underneath. The `o`
		// target card sits at the very bottom of the stack (a browse can
		// never bury a parked turn's modal, nor a peer picker).
		bg := strings.Split(out, "\n")
		if c.openPick != nil {
			bg = c.linkOverlay(bg)
		}
		if c.sessPick != nil {
			bg = c.sessOverlay(bg)
		}
		if c.permVisible() {
			bg = c.permOverlay(bg)
		}
		if c.question != nil {
			bg = c.questOverlay(bg)
		}
		out = strings.Join(bg, "\n")
	}
	return out
}

// buildBlocks collects the timeline (messages + per-agent worker threads)
// and renders every item through the per-block CACHE (chat_window.go): an
// unchanged block is borrowed, so a steady-state append/stream re-renders
// only the touched tail blocks. Populates c.blocks (+ prunes the cache).
// Employee tool entries (wtoolKind) collect into per-agent worker threads,
// but the threads are NOT a docked bottom region any more —
// mergeChatTimeline interleaves them with the visible entries by
// timestamp (a thread anchors at its creation time), so every entry —
// chat message or subagent thread — scrolls in chronological order.
func (c *Chat) buildBlocks() {
	c.renderCalls++ // the focus deferral's probe (RenderCalls)
	visible := make([]state.ChatMsg, 0, len(c.chat))
	var workers []workerGroup
	workerIdx := map[string]int{}
	for _, m := range c.chat {
		if m.Kind == wtoolKind || m.Kind == wthinkKind || m.Kind == wdiffKind {
			// /tools off hides the whole workers region; /thinking off
			// takes only the think rows (the tools keep rendering)
			if (m.Kind == wtoolKind || m.Kind == wdiffKind) && !c.showTools {
				continue
			}
			if m.Kind == wthinkKind && !c.showThinking {
				continue
			}
			idx, ok := workerIdx[m.From]
			if !ok {
				idx = len(workers)
				workerIdx[m.From] = idx
				workers = append(workers, workerGroup{name: m.From, title: c.segmentTitle(m)})
			} else if pl := workers[idx].lines[len(workers[idx].lines)-1]; m.At > 0 && pl.At > 0 &&
				m.At-pl.At > workforceGap.Milliseconds() {
				// epoch boundary: the desk name was recycled (restart) or
				// re-tasked after a long silence — split a NEW segment
				// that captures its OWN dispatch title and birth slot
				// instead of welding into the old wave's thread.
				idx = len(workers)
				workerIdx[m.From] = idx
				workers = append(workers, workerGroup{name: m.From, title: c.segmentTitle(m)})
			}
			workers[idx].lines = append(workers[idx].lines, m)
			if _, tk := parseWtoolMeta(m.Meta); tk > workers[idx].lastTick {
				workers[idx].lastTick = tk
			}
			continue
		}
		if m.From == "boss" && m.Pending && m.Text == "" {
			continue // the spinner line speaks for the EMPTY typing placeholder;
			// a pending bubble WITH text renders below (streaming reply)
		}
		if m.Kind == thinkKind && !c.showThinking {
			continue
		}
		if m.Kind == toolKind && !c.showTools {
			continue
		}
		visible = append(visible, m)
	}
	// prune stale segment-title memos: keep only the titles of segments
	// alive in THIS rebuild (a long session's capChat fuse drops old
	// lines; their first-line ids must not accumulate in segTitles).
	if len(c.segTitles) > 0 {
		alive := make(map[string]string, len(workers))
		for _, g := range workers {
			if len(g.lines) > 0 {
				alive[g.lines[0].ID] = g.title
			}
		}
		c.segTitles = alive
	}
	if len(visible) == 0 && len(workers) == 0 {
		content := chrome.PanelDim.Render("  no messages yet — ask the boss for something.")
		c.blocks = []*chatBlock{c.noteBlock(content, "chat-empty")}
		c.pruneBlockCache()
		return
	}
	items := mergeChatTimeline(visible, workers)
	// the opencode hint row trails the LAST thread block of the timeline
	// while ≥1 RENDERED thread is live (visibility-aware: /tools off
	// builds no tool threads, so none can be live on screen) — it rides
	// INSIDE that group's block (the same adjacency as ever: group rows,
	// one blank row, hint row).
	lastGroup := -1
	for i, item := range items {
		if item.Group >= 0 {
			lastGroup = i
		}
	}
	anyLive := false
	for _, g := range workers {
		if c.threadLive(g) {
			anyLive = true
			break
		}
	}
	// assemble from the per-block cache (chat_window.go): each item
	// renders ONCE per (identity × width/theme generation × its toggles ×
	// its content); unchanged spans are borrowed, so a steady-state
	// stream/append only re-renders its own tail blocks.
	gen := c.renderGen()
	blocks := make([]*chatBlock, 0, len(items))
	for i, item := range items {
		if item.Group >= 0 {
			blocks = append(blocks, c.renderGroupBlock(workers[item.Group], i == lastGroup && anyLive, gen))
		} else {
			blocks = append(blocks, c.renderMsgBlock(item.Msg, gen))
		}
	}
	c.blocks = blocks
	c.pruneBlockCache()
}

// renderConversation rebuilds the transcript as TEXT — the test-facing
// seam (the fold/thread/epoch suites split + strip it): byte-identical to
// what the pre-block monolithic builder produced. The panel itself renders
// through renderConversationLines (no join/split/pad round-trip per
// frame).
func (c *Chat) renderConversation() string {
	c.buildBlocks()
	c.mergeBlockHits()
	var b strings.Builder
	for i, blk := range c.blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(blk.text)
	}
	// a fragment trailing "\n" would otherwise end the transcript on a
	// blank row (the old TrimRight contract — kept verbatim)
	return strings.TrimRight(b.String(), "\n")
}

// renderConversationLines is the PANEL's render pipeline: the same block
// assembly, delivered as the PADDED row slice (selLines' exact content).
func (c *Chat) renderConversationLines() []string {
	c.buildBlocks()
	return c.assembleConversationLines()
}

// noteBlock renders a STATIC one-off fragment (the empty-transcript
// placeholder) through the block pipeline so the whole panel always has a
// coherent c.blocks.
func (c *Chat) noteBlock(content, id string) *chatBlock {
	if old := c.borrowStaticBlock(id, c.renderGen()); old != nil {
		return old
	}
	blk := &chatBlock{id: id, key: c.renderGen(), text: content}
	blk.finish()
	c.blockCache[id] = blk
	return blk
}

// mergeBlockHits stacks every block's LOCAL hit-maps into the absolute
// ClickRow maps (row arithmetic = blocks + one blank separator row each —
// the same plan the padded lines follow).
func (c *Chat) mergeBlockHits() {
	c.threadRows = map[int]string{}   // mouse hit-map, rebuilt every render
	c.userFoldRows = map[int]string{} // user-bubble fold hit-map, same rebuild
	c.toolDiffRows = map[int]string{} // ↳ diff sub-row hit-map, same rebuild
	c.toolRows = map[int]string{}     // tool-output one-liner hit-map, same rebuild
	c.btwPinRows = map[int]string{}   // hidden-BTW pin bubble hit-map, same rebuild
	row := 0                          // absolute start row of the CURRENT block
	for i, blk := range c.blocks {
		if i > 0 {
			row++ // the ONE separator row between timeline items
		}
		mergeSpanInto(c.threadRows, blk.hits.thread, row)
		mergeSpanInto(c.toolDiffRows, blk.hits.toolDiff, row)
		mergeSpanInto(c.userFoldRows, blk.hits.userFold, row)
		mergeSpanInto(c.toolRows, blk.hits.toolOut, row)
		mergeSpanInto(c.btwPinRows, blk.hits.btwPin, row)
		row += blk.rows
	}
}

// assembleConversationLines stacks c.blocks into the PADDED transcript row
// slice — the exact lines the old (join → split → per-line pad) pipeline
// produced: block rows in order with ONE padded-blank separator between
// blocks, trailing blank rows trimmed (the old TrimRight contract in line
// form). Also refreshes convWide (the widest PADDED line over all blocks —
// setConversationLines's rogue-row guard) and merges the click hit-maps.
func (c *Chat) assembleConversationLines() []string {
	c.mergeBlockHits()
	total := 0
	for i := range c.blocks {
		total += c.blocks[i].rows + 1 // +1 swallows the separator at i>0
	}
	lines := make([]string, 0, total)
	wide := 0
	for i, blk := range c.blocks {
		if i > 0 {
			// the ONE separator between timeline items: exactly one blank
			// row, padded like the old pad loop's empty lines
			lines = append(lines, chatPadStr)
		}
		lines = append(lines, blk.plines...)
		if blk.wide > wide {
			wide = blk.wide
		}
	}
	// trailing pad-only rows die at the transcript's tail — TrimRight's
	// line-form ("…\n\n" never leaves a dangling blank row).
	for len(lines) > 0 && lines[len(lines)-1] == chatPadStr {
		lines = lines[:len(lines)-1]
	}
	c.convWide = wide
	return lines
}

// renderMsgBlock renders ONE non-group timeline item (any chat message) as
// a cached block: the fragment text byte-identical to what the old
// item-loop body wrote into the shared builder, hit registrations in
// BLOCK-LOCAL row space.
func (c *Chat) renderMsgBlock(m state.ChatMsg, gen uint64) *chatBlock {
	// the invalidation key: the generation + every toggle that re-shapes
	// this kind's pixels (the CONTENT rides the borrow's src==m compare).
	// showThinking/showTools do NOT reach here — they filter the timeline
	// upstream.
	streaming := m.Kind == thinkKind && m.Meta != "" && c.streamingThink[m.Meta]
	var fl blockFlags
	switch {
	case m.Kind == thinkKind:
		if c.thinkExpanded {
			fl |= bfExpandA
		}
		if streaming {
			fl |= bfExpandB
		}
	case m.Kind == diffKind:
		if c.diffExpanded {
			fl |= bfExpandA
		}
	case m.Kind == toolKind:
		if c.toolExpanded[m.ID] {
			fl |= bfExpandA
		}
	case m.From == "user":
		if c.userExpanded[userFoldKey(m)] {
			fl |= bfExpandB
		}
	}
	k := newKeyMixer().num(gen).num(uint64(fl))
	// an expanded tool row's body IS the captured output — fold the text
	// in so a SetToolOutput landing while expanded (no message change)
	// still misses the borrow and re-renders the one block.
	if m.Kind == toolKind && c.toolExpanded[m.ID] {
		k.str(c.toolOutputs[m.ID])
	}
	if streaming {
		// the streaming header's spinner frame rides the office tick —
		// fold it in so each tick's frame re-renders (the old per-tick
		// rebuild for exactly this block kind)
		k.num(uint64(uint32(c.tick)))
	}
	// image-media arrivals re-shape ONLY carrier bubbles: fold that
	// bubble's media revision in — a landing re-renders the one owning
	// block, every other block's cache stays borrowed.
	if _, carrier := state.ParseMediaMeta(m.Meta); carrier {
		k.num(c.mediaRev[m.ID])
	}
	key := k.done()
	if old := c.borrowMsgBlock(m, key); old != nil {
		return old
	}
	var b strings.Builder
	hits := blockHits{}
	var mediaSlots []chatMediaSlot // boss bubbles: the kitty previews' paint slots (block-local rows)
	switch {
	case m.Kind == thinkKind:
		c.renderThink(&b, m)
	case m.Kind == toolKind:
		// boss inline tool one-liner — WRAPPED, never clipped and
		// never burst mid-glyph: the first row flows tight against the
		// bubble above (no leading indent), continuations hang under
		// the tool text start. The ▸/▾ chevron rides the prefix (the
		// whole one-liner's rows register in the toolRows click
		// hit-map); expanded, the captured output body renders dim
		// under the row (chat_toolrow.go) at the SAME hanging indent —
		// body rows never register.
		toolW := c.contentW() - 1
		open := c.toolExpanded[m.ID]
		prefix := toolWrapPrefix + toolChevron(open)
		indent := strings.Repeat(" ", cellWidth(prefix))
		lines := foldStyledRows(renderTool(m, open), toolW, toolW-cellWidth(prefix))
		b.WriteString(lines[0])
		for _, ln := range lines[1:] {
			b.WriteString("\n" + indent + ln)
		}
		hits.toolOut = map[int]string{}
		for i := range lines {
			hits.toolOut[i] = m.ID
		}
		if open {
			for _, ln := range c.toolOutputRows(m.ID, cellWidth(prefix), toolW-cellWidth(prefix)) {
				b.WriteString("\n" + ln)
			}
		}
	case m.Kind == questionKind:
		c.renderQuestion(&b, m)
	case m.Kind == diffKind:
		c.renderDiff(&b, m)
	case m.Kind == officeKind:
		// concierge (EvChatOffice) — a real turn, not a notice: the
		// INFO "office ›" case above renderNotice's dim-office line
		c.renderOffice(&b, m)
	case m.From == officeFrom && m.Meta == "btw-pin":
		prefix := chrome.OnPanel(chrome.Accent, "↩ ")
		textW := c.contentW() - cellWidth("↩ ")
		b.WriteString(prefix + chrome.OnPanelBold(chrome.White, clipPlain(m.Text, textW)))
		hits.btwPin = map[int]string{0: m.ID}
	case m.From == officeFrom:
		c.renderNotice(&b, m)
	case m.From == "user":
		prefix := chrome.OnPanel(chrome.Info, userPrefix)
		lines := strings.Split(strings.TrimRight(wrapPlain(m.Text, c.mdWidth+1), "\n"), "\n")
		// the open-in-browser beacon: a verified-target bubble wears a
		// dim " · o (open)" on its FIRST row (folded bubbles keep their
		// head rows, so the beacon survives the fold). Extraction is THE
		// gate (os.Stat-verified paths only) — the beacon never advertises
		// a target `o` could not fire.
		if len(ExtractChatTargets(m)) > 0 {
			lines[0] += chrome.PanelDim.Render(" · o (open)")
		}
		attachSuffix := ""
		if names, ok := state.ParseAttachMeta(m.Meta); ok && len(names) > 0 {
			// the backend's chat-user echo carries the attachment
			// names in Meta — history shows the dim " · 📎 N" suffix
			attachSuffix = chrome.PanelDim.Render(" · 📎 " + itoa(len(names)))
			lines[len(lines)-1] += attachSuffix
		}
		lines = foldStyledLines(lines, c.mdWidth+1)
		if len(lines) > userFoldVisible {
			// a LONG user turn folds to its head rows + a one-row
			// dim-italic hint (a click there expands it — the
			// userFoldRows hit-map carries exactly that row, body
			// rows never); expanded, the bubble keeps every row and
			// trails a clickable "… collapse" instead. The 📎 suffix
			// leaves the body for the hint row while folded (the
			// expanded shape keeps the pre-fold rendering: suffix on
			// the last body row). Hint/trailer are SINGLE clipped
			// rows (the thread header's contract): an overflowing
			// row would burst mid-word and break every content-row
			// click map. The LOCAL hit row is the block's OWN 0-based
			// line index (nothing precedes writePrefixed inside this
			// block) — mergeBlockHits offsets it into absolute rows.
			key := userFoldKey(m)
			hintW := c.contentW() - cellWidth(userPrefix) // the hanging indent eats the head
			if c.userExpanded[key] {
				hits.userFold = map[int]string{len(lines): key}
				lines = append(lines, chrome.PanelDim.Italic(true).Render(clipPlain("… collapse", hintW)))
			} else {
				hint := chrome.PanelDim.Italic(true).Render(clipPlain(
					"… +"+itoa(len(lines)-userFoldVisible)+" more lines · click to expand",
					hintW-cellWidth(ansi.Strip(attachSuffix)))) + attachSuffix
				hits.userFold = map[int]string{userFoldVisible: key}
				lines = append(lines[:userFoldVisible], hint)
			}
		}
		writePrefixed(&b, prefix, strings.Repeat(" ", cellWidth(userPrefix)), lines)
	default:
		prefix := chrome.OnPanel(chrome.Accent, bossPrefix)
		lines := cleanMarkdown(c.renderMarkdown(m.Text))
		// a streaming reply is just the bubble itself growing — no
		// caret, no extra row: the typing row below the divider is the
		// liveness signal for the whole pending period.
		// the open-in-browser beacon rides the FIRST body row THROUGH the
		// fold (the 📎 suffix's contract: a full row wraps it to the
		// continuation row, ANSI-aware, never clipped — and the media
		// chip prepends clean ABOVE it with the raster rows untouched).
		// The extract gate guarantees only verified targets advertise it.
		if len(lines) > 0 && len(ExtractChatTargets(m)) > 0 {
			lines[0] += chrome.PanelDim.Render(" · o (open)")
		}
		lines = foldStyledLines(lines, c.mdWidth)
		// Inbound image previews slot the chip + raster ABOVE the body
		// (completed bubbles only — a pending stream never previews).
		// Kitty previews ride the frame splice: the media rows are PURE
		// reservation rows and the paint slots ride the block (MediaFrameState
		// offsets them into absolute transcript rows at publish time).
		var mediaLines []string
		mediaLines, mediaSlots = c.renderMediaRows(m)
		lines = append(mediaLines, lines...)
		writePrefixed(&b, prefix, strings.Repeat(" ", cellWidth(bossPrefix)), lines)
	}
	blk := &chatBlock{id: m.ID, key: key, text: b.String(), hits: hits, src: m, unstable: streaming, media: mediaSlots}
	blk.finish()
	if m.ID != "" {
		c.blockCache[m.ID] = blk
	}
	return blk
}

// renderGroupBlock renders ONE worker-thread segment as a cached block.
// renderWorkerGroup (threads_opencode.go) registers its frame rows by
// reading the buffer's CURRENT row count + writing directly into
// c.threadRows/c.toolDiffRows/c.toolRows — so the block capture swaps in
// PRIVATE maps and a PRIVATE (empty) builder: the registrations come out
// in block-local space (its b.Len()==0 top-edge path IS the block-local
// convention), then mergeBlockHits offsets them like every other block's.
// The trailing live-hint row stays glued to the last live group.
func (c *Chat) renderGroupBlock(g workerGroup, hint bool, gen uint64) *chatBlock {
	// resolve the toggle state EXACTLY as renderWorkerGroup does (the key
	// must miss on every input that re-paints the thread)…
	live := c.threadLive(g)
	stopped := c.threadStop[g.name]
	expanded := c.threadsExpanded
	full := c.threadsExpanded
	if v, ok := c.threadExpand[g.name]; ok {
		expanded = v
		full = v
	}
	if stopped {
		if _, ok := c.threadExpand[g.name]; !ok {
			expanded = false
		}
	}
	// …then fingerprint every NON-LINE input (the lines themselves ride
	// the borrow's header compare): the sticky title resolution + roster
	// rollup the header reads, the resolved toggles, the per-call diff
	// opens, the hint tail, and the tick while LIVE (the glyph animates).
	id := "g:" + g.lines[0].ID
	k := newKeyMixer().num(gen).str(g.name).str(g.title)
	k.str(c.threadTitle(g.name)) // the LIVE sticky resolution (≠ g.title while running)
	if av, ok := c.agents[g.name]; ok {
		k.str(av.task).str(string(av.role)).boo(av.active)
	}
	for _, m := range g.lines {
		if m.Kind == wdiffKind && c.threadDiffOpen[m.ID] {
			k.str("diff-open:" + m.ID)
		}
		// an expanded wtool row's output body re-paints the thread (the
		// same fold as the ↳ diff opens, output text included so a
		// SetToolOutput landing mid-expand still misses the borrow)
		if m.Kind == wtoolKind && c.toolExpanded[m.ID] {
			k.str("tool-open:" + m.ID).str(c.toolOutputs[m.ID])
		}
	}
	var fl blockFlags
	if expanded {
		fl |= bfExpandA
	}
	if live {
		fl |= bfExpandB
	}
	if full {
		fl |= bfExpandC
	}
	if stopped {
		fl |= bfStopped
	}
	if hint {
		fl |= bfThreadHint
	}
	k.num(uint64(fl))
	if live {
		k.num(uint64(uint32(c.tick)))
	}
	key := k.done()
	if old := c.borrowGroupBlock(id, g.lines, key); old != nil {
		return old
	}
	savedThread, savedDiff, savedTool := c.threadRows, c.toolDiffRows, c.toolRows
	c.threadRows, c.toolDiffRows, c.toolRows = map[int]string{}, map[int]string{}, map[int]string{}
	var b strings.Builder
	c.renderWorkerGroup(&b, g)
	hits := blockHits{thread: c.threadRows, toolDiff: c.toolDiffRows, toolOut: c.toolRows}
	c.threadRows, c.toolDiffRows, c.toolRows = savedThread, savedDiff, savedTool
	text := strings.TrimPrefix(b.String(), "\n\n") // the block's own lead becomes the assembly's separator
	if hint {
		text += "\n\n" + chrome.PanelDim.Italic(true).Render(threadHintText)
	}
	blk := &chatBlock{id: id, key: key, text: text, hits: hits, lines: g.lines, unstable: live}
	blk.finish()
	c.blockCache[id] = blk
	return blk
}

// agentView — a roster rollup entry for the workers-thread decoration:
// the agent's dispatch task (thread title), sprite-liveness (the thread's
// live/done glyph + the loading row's live/hide rule), and roster role
// (the title's "<Kind>" label — see threads_opencode.go).
type agentView struct {
	task   string
	active bool
	role   state.EmployeeRole
}

// workforceGap — the epoch boundary for same-name worker threads. Desk
// names are recycled across restarts (a re-hired "tekton-18" is a NEW
// wave of work, not the old one continuing), and a long-running
// continuous session can still re-task the same agent after a quiet
// stretch: a same-name line arriving more than workforceGap after the
// name's PREVIOUS line opens a NEW thread segment (its own timeline
// birth slot, its own captured dispatch title) instead of welding into
// the old group. Inside the gap lines merge as today — a quickly
// re-tasked agent reads as one work session (the guard for continuous
// sessions). Stamp-less (At==0) lines never force a split.
const workforceGap = 10 * time.Minute

// workerGroup — one agent SEGMENT's merged wtool/wthink entries (chat
// order) plus the latest activity tick, rendered as one work thread. A
// recycled desk name yields one group per epoch (see workforceGap);
// title is the segment's OWN dispatch title, captured at the segment's
// birth (segmentTitle) — never re-read from the sticky per-name map on
// later renders, so two epochs of one name keep their own titles.
type workerGroup struct {
	name     string
	title    string
	lines    []state.ChatMsg
	lastTick int
}

// segmentTitle resolves a worker-thread segment's header title AT THE
// SEGMENT'S BIRTH (m = its first chat line) and memoizes it by that
// line's id: the group rebuild every render then re-uses the birth
// capture for settled epochs while the latest, still-running epoch's
// title follows the live sticky resolution. A memo miss (first sight of
// the segment — live birth, or the first render after a boot restore)
// falls back to threadTitle's live sticky-map resolution.
func (c *Chat) segmentTitle(m state.ChatMsg) string {
	if c.segTitles == nil {
		c.segTitles = map[string]string{}
	}
	if t, ok := c.segTitles[m.ID]; ok {
		return t
	}
	t := c.threadTitle(m.From)
	c.segTitles[m.ID] = t
	return t
}

// workerSpriteActive — the sprite states that count as "the agent is still
// at work" for the thread's live/expanded decision.
func workerSpriteActive(s state.SpriteState) bool {
	switch s {
	case state.SpriteWorking, state.SpriteToManager, state.SpriteMeeting:
		return true
	}
	return false
}

// parseWtoolMeta decodes the employee-tool Meta carrier the reducer
// writes: toolState ␟ tick. A Meta without the separator yields tick 0
// (immediately stale → collapsed).
func parseWtoolMeta(meta string) (toolState string, tick int) {
	if i := strings.IndexByte(meta, diffMetaSep[0]); i >= 0 {
		return meta[:i], atoiHead(meta[i+1:])
	}
	return meta, 0
}

// The worker-thread renderer itself (renderWorkerGroup, the header/sneak
// /expanded row builders, wthinkRows, workerToolLine, the role→Kind
// labels, the pending block bar) lives in threads_opencode.go — the
// opencode-style spinner+sneak design replaced the old bordered cards.

// thinkStreamLines caps the visible body of a STREAMING think block —
// the tail of the accumulated text (the freshest lines) stays visible.
const thinkStreamLines = 10

// thinkFrames cycles the streaming header's spinner char, one step per
// office tick (no extra timer — render consults c.tick).
var thinkFrames = []string{"|", "/", "-", "\\"}

// renderThink renders one Kind="think" entry in one of three shapes:
//
//	STREAMING  — dim-italic "⠿ thinking…" spinner header + the accumulated
//	             text dim-italic, LAST thinkStreamLines lines max with a
//	             "… N more above" first line; always expanded (ctrl+t
//	             cannot collapse a live transcript).
//	collapsed  — one dim "thinking · N lines" line (the default once the
//	             stream's Done update lands / a new boss turn starts).
//	expanded   — dim-italic "thinking" header + greyed body (ctrl+t).
func (c *Chat) renderThink(b *strings.Builder, m state.ChatMsg) {
	think := chrome.PanelDim.Italic(true)
	body := chrome.PanelDim
	lines := strings.Split(strings.TrimRight(wrapPlain(m.Text, c.mdWidth+1), "\n"), "\n")
	if m.Meta != "" && c.streamingThink[m.Meta] {
		frame := thinkFrames[c.tick%len(thinkFrames)]
		if frame < "" {
			frame = "|"
		}
		b.WriteString(think.Render(frame + " thinking…"))
		shown := lines
		if more := len(lines) - thinkStreamLines; more > 0 {
			b.WriteString("\n  ")
			b.WriteString(think.Render("… " + itoa(more) + " more above"))
			shown = lines[more:]
		}
		for _, ln := range shown {
			b.WriteString("\n  ")
			b.WriteString(think.Render(ln))
		}
		return
	}
	if !c.thinkExpanded {
		b.WriteString(think.Render("thinking · ") + body.Render(countLines(lines)+" lines"))
		return
	}
	b.WriteString(think.Render("thinking"))
	for _, ln := range lines {
		b.WriteString("\n  ")
		b.WriteString(body.Render(ln))
	}
}

// countLines is the display count for a collapsed thinking block.
func countLines(lines []string) string {
	n := len(lines)
	if n > 1 {
		return itoa(n)
	}
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return "0"
	}
	return "1"
}

// itoa avoids fmt for a digit-only render.
func itoa(n int) string {
	if n >= 10 {
		return itoa(n/10) + string(rune('0'+n%10))
	}
	return string(rune('0' + n%10))
}

// toolWrapPrefix starts every boss inline tool one-liner — continuation
// rows of a wrapped line hang under the text start: this many cells PLUS
// the ▸/▾ chevron field (toolChevron, chat_toolrow.go) the render path
// appends to it.
const toolWrapPrefix = "[tool] "

// renderTool renders one Kind="tool" one-liner, merged by CallID upstream:
// "[tool] ▸ read · src/main.go ✓" (done) / "… running" / red "✗" (error) /
// dim-red "✗ aborted" (/stop). The ▸/▾ chevron (toolChevron — the
// click-to-expand signal, chat_toolrow.go) rides the prefix. The whole
// line comes out as ONE styled blob; the call site folds it with
// foldStyledRows — escape sequences are consumed atomically there, so a
// fold boundary never shreds one.
func renderTool(m state.ChatMsg, open bool) string {
	line := toolWrapPrefix + toolChevron(open) + m.Text
	switch m.Meta {
	case "done":
		return chrome.PanelTool.Render(line + " ✓")
	case "error":
		return chrome.PanelErr.Faint(true).Render(line + " ✗")
	case "aborted": // /stop unwind swung a running call here
		return chrome.PanelErr.Faint(true).Render(line + " ✗ aborted")
	default: // running (or anything unexpected)
		return chrome.PanelTool.Render(line + " … running")
	}
}

// renderQuestion renders one Kind="question" entry (boss question tool):
// yellow "boss asks › <text>", dim options inline when present, then the
// "(answer by typing below)" hint while pending — or a dim "✓ answered"
// suffix once the resolved event landed (the app reducer marks Meta with
// a trailing ␟answered unit-separator token, keeping the options intact).
func (c *Chat) renderQuestion(b *strings.Builder, m state.ChatMsg) {
	options := m.Meta
	answered := false
	if parts := strings.Split(m.Meta, diffMetaSep); len(parts) == 2 && parts[1] == "answered" {
		options, answered = parts[0], true
	}
	qPrefix := "boss asks › "
	indent := strings.Repeat(" ", cellWidth(qPrefix))
	// cellWidth, not len — "›" is 3 bytes but 1 cell; the byte count would
	// shave 2 columns off the wrap budget and misalign the hanging indent
	wrapW := c.contentW() - cellWidth(qPrefix) - 1 // prefix + transcript insets
	lines := strings.Split(strings.TrimRight(wrapPlain(m.Text, wrapW), "\n"), "\n")
	for i := range lines {
		lines[i] = chrome.PanelWarn.Render(lines[i])
	}
	lines = foldStyledLines(lines, wrapW)
	writePrefixed(b, chrome.PanelWarn.Bold(true).Render(qPrefix), indent, lines)
	if options != "" {
		optLines := strings.Split(strings.TrimRight(wrapPlain("("+options+")", wrapW), "\n"), "\n")
		for i := range optLines {
			optLines[i] = chrome.PanelDim.Render(optLines[i])
		}
		for _, ln := range foldStyledLines(optLines, wrapW) {
			b.WriteString("\n" + indent + ln)
		}
	}
	if answered {
		b.WriteString("\n" + indent + chrome.PanelDim.Render("✓ answered"))
	} else {
		b.WriteString("\n" + indent + chrome.PanelDim.Italic(true).Render("(answer by typing below)"))
	}
}

// renderDiff renders one Kind="diff" entry opencode-style. Collapsed
// (default): a single "diff · path +A -D" line with green/red counts.
// Expanded (ctrl+d / /diffs on): a dim-bold "← Edit|New file|Delete <path>"
// header over LINE-NUMBERED unified rows — deletion rows tinted DiffDelBg
// (dark red), addition rows tinted DiffAddBg (dark green) to the FULL panel
// width, context rows dim with no tint, @@ hunk headers dim italic with no
// gutter number. The +/- marker sits inside the tinted row. Text inside the
// rows is syntax-coloured through chroma (theme-mapped style) on top of the
// tint when a lexer matches the file. Body is clipped to diffClip rows with
// a "+N more" trailer.
func (c *Chat) renderDiff(b *strings.Builder, m state.ChatMsg) {
	path, adds, dels := parseDiffMeta(m.Meta)
	if !c.diffExpanded {
		header := chrome.PanelDim.Render("diff · " + path)
		if adds != "" {
			header += " " + chrome.PanelOK.Render(adds)
		}
		if dels != "" {
			header += " " + chrome.PanelErr.Render(dels)
		}
		b.WriteString(header)
		return
	}
	rows, op := c.diffRows(m, path)
	opWord := "Edit"
	switch op {
	case diffOpNew:
		opWord = "New file"
	case diffOpDel:
		opWord = "Delete"
	}
	b.WriteString(clipStyled(chrome.PanelDim.Bold(true), "← "+opWord+" "+path, c.contentW()))
	c.renderDiffRows(b, rows)
}

// renderDiffRows writes the LINE-NUMBERED body of a parsed diff: one
// renderDiffRow per parsed row (gutter + tinted add/del/context bars),
// clipped to diffClip rows with a "+N more" trailer. The rows come from
// c.diffRows (parseDiffBody + chroma paint, cached per msg ID) so BOTH
// the flat diff (ctrl+d, above) and a worker thread's clicked-open ↳
// diff body (threads_opencode.go wdiffRows) draw the byte-same body.
func (c *Chat) renderDiffRows(b *strings.Builder, rows []diffRow) {
	maxNum := 0
	for _, r := range rows {
		if r.num > maxNum {
			maxNum = r.num
		}
	}
	gutterW := 5
	if n := len(itoa(maxNum)); n > gutterW {
		gutterW = n
	}
	shown := rows
	more := 0
	if len(rows) > diffClip {
		shown = rows[:diffClip]
		more = len(rows) - diffClip
	}
	for i := range shown {
		b.WriteString("\n")
		b.WriteString(renderDiffRow(shown[i], gutterW, c.contentW()))
	}
	if more > 0 {
		b.WriteString("\n" + strings.Repeat(" ", gutterW+3) +
			chrome.PanelDim.Italic(true).Render("+"+itoa(more)+" more"))
	}
}

// parseDiffMeta decodes the diff Meta carrier (path ␟ +adds ␟ -dels)
// written by the app reducer.
func parseDiffMeta(meta string) (path, adds, dels string) {
	parts := strings.Split(meta, diffMetaSep)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}
	return meta, "", ""
}

// --- opencode-style diff rows -------------------------------------------------

type diffOp int

const (
	diffOpEdit diffOp = iota // --- a/path +++ b/path
	diffOpNew                // --- /dev/null (file created)
	diffOpDel                // +++ /dev/null (file removed)
)

type diffRowKind int

const (
	dkContext diffRowKind = iota
	dkAdd
	dkDel
	dkHunk // @@ header or "\ No newline…" note — no gutter number
)

// diffSpan is one styled text segment inside a diff row. fg is "" to
// inherit the row ink, else "#rrggbb" from the chroma style.
type diffSpan struct {
	text                    string
	fg                      string
	bold, italic, underline bool
}

// diffRow — one display row of a parsed unified diff. num is the gutter
// line number (old-side for deletions+context, new-side for additions; 0
// for hunk rows). oldLine/newLine index the row's text inside the old-side
// and new-side source streams (-1 when absent). spans covers the text
// portion AFTER the +/- marker.
type diffRow struct {
	kind             diffRowKind
	num              int
	oldLine, newLine int
	spans            []diffSpan
}

// diffCacheEntry — parsed+highlighted diff rows keyed by chat msg ID;
// syntax colours are theme-bound so RefreshTheme clears the map.
type diffCacheEntry struct {
	theme string
	rows  []diffRow
	op    diffOp
}

// diffRows parses m.Text (unified diff body) into rows and paints chroma
// spans from the matching lexer; results are cached per msg ID + theme.
func (c *Chat) diffRows(m state.ChatMsg, path string) ([]diffRow, diffOp) {
	if ent, ok := c.diffCache[m.ID]; ok && ent.theme == chrome.CurrentTheme().Name {
		return ent.rows, ent.op
	}
	rows, op, oldBody, newBody := parseDiffBody(m.Text)
	lx := chlexers.Match(path)
	if lx != nil && lx.Config() != nil && lx.Config().Name == "fallback" {
		lx = nil
	}
	st := chstyles.Get(chrome.DiffChromaStyle)
	oldSpans := tokenizeSide(lx, st, oldBody)
	newSpans := tokenizeSide(lx, st, newBody)
	for i := range rows {
		if rows[i].kind == dkHunk {
			continue
		}
		line := rows[i].oldLine
		spans := oldSpans
		if rows[i].kind == dkAdd || (line < 0 && rows[i].newLine >= 0) {
			line, spans = rows[i].newLine, newSpans
		}
		// additions read the NEW-side stream, deletions the OLD-side stream;
		// context prefers new (falls back to old for pure-deletion files)
		if rows[i].kind == dkContext && rows[i].newLine >= 0 {
			line, spans = rows[i].newLine, newSpans
		}
		if line >= 0 && line < len(spans) {
			rows[i].spans = spans[line]
		}
	}
	if c.diffCache == nil {
		c.diffCache = map[string]diffCacheEntry{}
	}
	c.diffCache[m.ID] = diffCacheEntry{theme: chrome.CurrentTheme().Name, rows: rows, op: op}
	return rows, op
}

// parseDiffBody decodes a unified diff body into display rows and returns
// the old-side (context+deletions) and new-side (context+additions) source
// texts for chroma; each row records its line index in both streams.
func parseDiffBody(body string) (rows []diffRow, op diffOp, oldBody, newBody string) {
	op = diffOpEdit
	var oldLines, newLines []string
	body = strings.ReplaceAll(body, "\t", "    ") // vp soft-wrap expands tabs
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	oldN, newN := 1, 1
	seenHunk := false
	for _, ln := range lines {
		switch {
		case !seenHunk && strings.HasPrefix(ln, "--- "):
			if strings.TrimSpace(strings.TrimPrefix(ln, "--- ")) == "/dev/null" {
				op = diffOpNew
			}
		case !seenHunk && strings.HasPrefix(ln, "+++ "):
			if strings.TrimSpace(strings.TrimPrefix(ln, "+++ ")) == "/dev/null" {
				op = diffOpDel
			}
		case strings.HasPrefix(ln, "@@"):
			seenHunk = true
			oldN, newN = parseHunkHeader(ln)
			rows = append(rows, diffRow{kind: dkHunk, oldLine: -1, newLine: -1,
				spans: []diffSpan{{text: ln}}})
		case strings.HasPrefix(ln, "\\"):
			// "\ No newline at end of file" — a note, not source text
			rows = append(rows, diffRow{kind: dkHunk, oldLine: -1, newLine: -1,
				spans: []diffSpan{{text: ln}}})
		case strings.HasPrefix(ln, "-"):
			rows = append(rows, diffRow{kind: dkDel, num: oldN,
				oldLine: len(oldLines), newLine: -1})
			oldLines = append(oldLines, ln[1:])
			oldN++
		case strings.HasPrefix(ln, "+"):
			rows = append(rows, diffRow{kind: dkAdd, num: newN,
				oldLine: -1, newLine: len(newLines)})
			newLines = append(newLines, ln[1:])
			newN++
		default: // context: " text" or a bare empty line
			text := ln
			if strings.HasPrefix(ln, " ") {
				text = ln[1:]
			}
			// context rows carry the OLD gutter number and exist in BOTH
			// side streams.
			rows = append(rows, diffRow{kind: dkContext, num: oldN,
				oldLine: len(oldLines), newLine: len(newLines)})
			oldLines = append(oldLines, text)
			newLines = append(newLines, text)
			oldN++
			newN++
		}
	}
	return rows, op, strings.Join(oldLines, "\n"), strings.Join(newLines, "\n")
}

// parseHunkHeader extracts the -o[,l] +n[,m] counters from an @@ header.
func parseHunkHeader(ln string) (oldN, newN int) {
	oldN, newN = 1, 1
	for _, f := range strings.Fields(strings.TrimPrefix(ln, "@@")) {
		if strings.HasPrefix(f, "-") {
			oldN = atoiHead(f[1:])
		} else if strings.HasPrefix(f, "+") {
			newN = atoiHead(f[1:])
		}
	}
	return
}

// atoiHead parses the leading digits of s ("52,11" → 52). 0 on garbage.
func atoiHead(s string) int {
	n := 0
	for i := 0; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// tokenizeSide splits `text` into per-line chroma spans using the theme's
// chroma style. fg of every span maps to a lipgloss hex colour or "" (the
// row ink wins); backgrounds from chroma are DROPPED so the row tint stays
// uniform. On any failure (nil lexer, tokenise error) each line is one
// plain span.
func tokenizeSide(lx chroma.Lexer, st *chroma.Style, text string) [][]diffSpan {
	plain := func() [][]diffSpan {
		ls := strings.Split(text, "\n")
		out := make([][]diffSpan, len(ls))
		for i := range ls {
			out[i] = []diffSpan{{text: ls[i]}}
		}
		return out
	}
	if text == "" || lx == nil || st == nil {
		return plain()
	}
	it, err := lx.Tokenise(nil, text)
	if err != nil {
		return plain()
	}
	lines := [][]diffSpan{{}}
	for _, tok := range it.Tokens() {
		entry := st.Get(tok.Type)
		sp := diffSpan{fg: ""}
		if entry.Colour.IsSet() {
			sp.fg = entry.Colour.String()
		}
		sp.bold = entry.Bold == chroma.Yes
		sp.italic = entry.Italic == chroma.Yes
		sp.underline = entry.Underline == chroma.Yes
		parts := strings.Split(tok.Value, "\n")
		for i, p := range parts {
			if i > 0 {
				lines = append(lines, []diffSpan{})
			}
			if p == "" {
				continue
			}
			s := sp
			s.text = p
			lines[len(lines)-1] = append(lines[len(lines)-1], s)
		}
	}
	return lines
}

// renderDiffRow renders one parsed row opencode-style: dim right-aligned
// gutter number, +/- marker INSIDE the row, text clipped to the panel
// width and the whole row padded out with the tint background so it reads
// as a full-width bar. When the theme's bg slot is nil (mono), the tint is
// suppressed and the ink emphasises instead (bold adds / underline dels).
func renderDiffRow(row diffRow, gutterW, width int) string {
	if width < 1 {
		width = 1
	}
	if row.kind == dkHunk {
		textW := width - gutterW - 3
		if textW < 1 {
			textW = 1
		}
		return strings.Repeat(" ", gutterW+3) +
			chrome.PanelDim.Italic(true).Render(clipPlain(row.spans[0].text, textW))
	}
	var fg color.Color = chrome.DiffCtxFg
	var bg color.Color
	switch row.kind {
	case dkAdd:
		fg, bg = chrome.DiffAddFg, chrome.DiffAddBg
	case dkDel:
		fg, bg = chrome.DiffDelFg, chrome.DiffDelBg
	}
	base := lipgloss.NewStyle().Foreground(fg)
	if bg != nil {
		base = base.Background(bg)
	} else {
		// tint suppressed (mono): bold additions / underlined deletions
		switch row.kind {
		case dkAdd:
			base = base.Bold(true)
		case dkDel:
			base = base.Underline(true)
		}
	}
	gstyle := lipgloss.NewStyle().Foreground(chrome.DiffGutterFg)
	if bg != nil {
		gstyle = gstyle.Background(bg)
	}
	gut := itoa(row.num)
	for len(gut) < gutterW {
		gut = " " + gut
	}
	marker := " "
	switch row.kind {
	case dkAdd:
		marker = "+"
	case dkDel:
		marker = "-"
	}
	var sb strings.Builder
	sb.WriteString(gstyle.Render(gut))
	sb.WriteString(base.Render(" " + marker + " "))
	textW := width - gutterW - 3 // gutter + " " + marker + " "
	if textW < 1 {
		textW = 1
	}
	used := 0
	for _, sp := range clipSpans(row.spans, textW) {
		st := base
		if sp.fg != "" {
			st = st.Foreground(lipgloss.Color(sp.fg))
		}
		if sp.bold {
			st = st.Bold(true)
		}
		if sp.italic {
			st = st.Italic(true)
		}
		if sp.underline {
			st = st.Underline(true)
		}
		sb.WriteString(st.Render(sp.text))
		used += lipgloss.Width(sp.text)
	}
	if bg != nil && used < textW {
		sb.WriteString(base.Render(strings.Repeat(" ", textW-used)))
	}
	return sb.String()
}

// clipSpans truncates a span run to w display cells, splitting spans at the
// boundary. (spans are plain text — no ANSI.)
func clipSpans(spans []diffSpan, w int) []diffSpan {
	if w < 0 {
		w = 0
	}
	var out []diffSpan
	used := 0
	for _, sp := range spans {
		remain := w - used
		if remain <= 0 {
			break
		}
		sw := lipgloss.Width(sp.text)
		if sw <= remain {
			out = append(out, sp)
			used += sw
			continue
		}
		s := sp
		s.text = clipPlain(sp.text, remain)
		if s.text != "" {
			out = append(out, s)
		}
		break
	}
	return out
}

// clipStyled clips plain text to w cells then renders it with style.
func clipStyled(style lipgloss.Style, s string, w int) string {
	return style.Render(clipPlain(s, w))
}

// renderOffice renders one Kind="office" concierge bubble: an INFO-colored
// (cyan) "office ›" label over glamour-rendered markdown — visually a
// peer of the boss's yellow turn, but its own voice. A pending entry with
// no text yet draws the dim "office is answering…" placeholder; a pending
// entry WITH text streams in place (glamour re-rendered per accumulated
// delta, the completion pin settling it — the same replace-by-ID contract
// as the boss stream, and like the boss stream there is NO caret: the
// agents roster's "answering" word carries the liveness).
func (c *Chat) renderOffice(b *strings.Builder, m state.ChatMsg) {
	prefix := chrome.OnPanel(chrome.Info, officePrefix)
	indent := strings.Repeat(" ", cellWidth(officePrefix))
	if m.Pending && m.Text == "" {
		b.WriteString(prefix)
		b.WriteString(chrome.PanelDim.Render("office is answering…"))
		return
	}
	lines := cleanMarkdown(c.renderMarkdown(m.Text))
	// office bubbles hang at 9 cells — fold at the office's own budget, not
	// the boss's (contentW−8): the shared budget let office continuations
	// reach contentW+1, one cell over the right gutter.
	lines = foldStyledLines(lines, c.contentW()-cellWidth(officePrefix)-1)
	writePrefixed(b, prefix, indent, lines)
}

// renderNotice renders a local From="office" notice (slash-command output):
// dim by default, red when Meta == "error".
func (c *Chat) renderNotice(b *strings.Builder, m state.ChatMsg) {
	style := chrome.PanelDim
	if m.Meta == errMeta || m.Meta == bootWarnMeta {
		style = chrome.PanelErr
	}
	lines := strings.Split(strings.TrimRight(wrapPlain(m.Text, c.mdWidth+1), "\n"), "\n")
	for i := range lines {
		lines[i] = style.Render(lines[i])
	}
	lines = foldStyledLines(lines, c.mdWidth+1)
	writePrefixed(b, style.Render(officePrefix), strings.Repeat(" ", cellWidth(officePrefix)), lines)
}

// renderMarkdown runs a boss turn through glamour with the sidebar wrap and
// the active theme's glamour style (chrome.MarkdownStyle).
func (c *Chat) renderMarkdown(text string) string {
	if c.md == nil {
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(chrome.MarkdownStyle()),
			glamour.WithWordWrap(c.mdWidth),
		)
		if err != nil {
			return text
		}
		c.md = r
	}
	out, err := c.md.Render(text)
	if err != nil {
		return text
	}
	return out
}

// cleanMarkdown trims glamour's frame noise: right-trailing styled spaces on
// every line and lines with no printable text at the EDGES. The leading strip
// matters: glamour's Document.BlockPrefix emits one blank frame line ABOVE
// every markdown block; if it survives, writePrefixed spends the "boss › "
// prefix on an empty row and the entire body falls one level to continuation
// indent — the bubble reads ~2 cells off (the "misaligned boss bubble" wart).
// Interior blank rows (paragraph/fence separation) are untouched.
func cleanMarkdown(out string) []string {
	lines := strings.Split(out, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " ")
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// writePrefixed writes lines, prefixing the first with `prefix` and hanging
// the rest under it with `indent`. Newlines go BETWEEN lines only, never
// after the last: callers are timeline item bodies, which end on content
// (the render loop owns all separation).
func writePrefixed(b *strings.Builder, prefix, indent string, lines []string) {
	for i, ln := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if i == 0 {
			b.WriteString(prefix + ln)
		} else {
			b.WriteString(indent + ln)
		}
	}
}

// chatMediaSlot — ONE splice-routed kitty preview's paint slot, in
// BLOCK-LOCAL row space (the same convention blockHits carries): row =
// the FIRST reservation row's local index (the image's paint cell —
// where the pre-splice APC row's cursor stood), id = the strip's i=
// office id (the wrapper's a=d target), frame = the cached verbatim APC
// (PlaceholderStrip — q=2, f=100, NO c=/r=: the wave-81 emission
// ruling). Stored on the chatBlock like the local hit-maps: the merge
// offsets it into absolute transcript rows exactly like them.
type chatMediaSlot struct {
	row   int
	id    uint32
	frame string
}

// chatMediaCol — the media image's cell column inside the chat content
// box (0-based): the transcript's left gutter + the boss bubble's
// hanging indent (cellWidth("boss › ") = 7 — the reservation rows sit
// at the bubble's body indent, exactly where the pre-splice APC row's
// cursor stood).
const chatMediaCol = chatPadL + 7

// renderMediaRows — the boss bubble's inbound image rows (the v1 preview):
// ONE dim chip per image ("🖼 name · WxH · mime", or the dim
// "unsupported image · click txt link" copy when the payload is remote,
// undecodable, or failed the rasterize) followed by its half-block
// truecolor raster rows when the lazy probe landed them. Cards stack in
// carrier order, all folded to the bubble's mdWidth budget (the rows are
// pre-folded paint — foldStyledRows refolds the ANSI atoms, never bursts
// mid-glyph). Empty for plain turns: no carrier Meta, or the bubble is
// still pending (the pin alone previews).
//
// THE KITTY LANE RIDES THE FRAME SPLICE (the wave-86 routing): the
// strip is NEVER embedded in the View string (bubbletea's renderer drops
// zero-width APCs — the wave-81 forensics — so an embedded strip only
// bloated the differ): the media rows are PURE reservation rows (blank,
// the SAME cell-box height) and the slot registers the paint cell for
// the registry publish. The OSC 1337 (iterm) lane keeps the OLD
// embedded frame row — no id, no delete escape: the splice could never
// target it.
func (c *Chat) renderMediaRows(m state.ChatMsg) ([]string, []chatMediaSlot) {
	if m.Pending {
		return nil, nil
	}
	items, ok := state.ParseMediaMeta(m.Meta)
	if !ok {
		return nil, nil
	}
	var lines []string
	var slots []chatMediaSlot
	for _, it := range items {
		v := c.media[m.ID][it.Hash]
		lines = append(lines, mediaChipLine(it, v))
		if v.failed {
			continue
		}
		if v.frame != "" {
			if v.kitty && v.id != 0 {
				// the splice routing: cellRows PURE reservation rows (the
				// wrapper paints the pixels at the slot's absolute cell) —
				// ZERO APC bytes in the View string.
				slots = append(slots, chatMediaSlot{row: len(lines), id: v.id, frame: v.frame})
				for i := 0; i < v.cellRows; i++ {
					lines = append(lines, "")
				}
				continue
			}
			// OSC 1337 (and an unparseable-id kitty frame — never splice
			// what the diff cannot target): the OLD embedded row — ONE
			// atomic frame written VERBATIM (folding could burst the
			// escape mid-sequence) plus cellRows-1 reservation rows.
			lines = append(lines, v.frame)
			for i := 1; i < v.cellRows; i++ {
				lines = append(lines, "")
			}
			continue
		}
		for _, row := range v.rows {
			lines = append(lines, foldStyledRows(row, c.mdWidth, c.mdWidth)...)
		}
	}
	return lines, slots
}

// MediaFrameState — the chat pane's registry contribution for the frame
// splice (zenbu_frame.go's CHAT-MEDIA region): every VISIBLE kitty
// preview as {office id, chat-content-local cell (0-based), the cached
// verbatim APC}. Block-local slots stack into absolute transcript rows
// by the SAME +1-separator plan mergeBlockHits owns, then the viewport's
// scroll offset maps them into the window — rows scrolled above/below
// never publish (the wrapper's emitted-set diff flushes their a=d:
// scrolled-off media vanishes cleanly). OSC 1337 previews are never
// here (no id — see chatMediaView). The app adds the sidebar/band
// origin and publishes per Frame; a stale read between renders is
// impossible (blocks and the scroll offset are the same goroutine's
// state the Frame just painted).
func (c *Chat) MediaFrameState() []ZenbuFrameImage {
	vpH := c.vp.Height()
	if vpH < 1 || len(c.blocks) == 0 {
		return nil
	}
	yoff := c.vp.YOffset()
	var out []ZenbuFrameImage
	row := 0
	for i, blk := range c.blocks {
		if i > 0 {
			row++ // the ONE separator row between timeline items
		}
		for _, s := range blk.media {
			vr := row + s.row - yoff
			if vr < 0 || vr >= vpH {
				continue
			}
			out = append(out, ZenbuFrameImage{OfficeID: s.id, OX: chatMediaCol, OY: vr, Frame: s.frame})
		}
		row += blk.rows
	}
	return out
}

// mediaChipLine — the ONE dim chip row per inbound image. The unsupported
// copy covers remote URLs (never fetched), header-rejected payloads, and
// rasterize failures — anything the office could not paint itself.
func mediaChipLine(it state.MediaItem, v chatMediaView) string {
	name := it.Filename
	if name == "" {
		name = "image"
	}
	if v.failed || it.W < 1 || it.H < 1 {
		return chrome.PanelDim.Render("🖼 " + clipPlain(name, 40) + " · unsupported image · click txt link")
	}
	return chrome.PanelDim.Render(fmt.Sprintf("🖼 %s · %d×%d · %s", clipPlain(name, 40), it.W, it.H, it.Mime))
}

// wrapPlain greedy word-wraps text to w cells. No semantics — just a
// visual fold (user turns, question/options rows, think bodies, modals).
// ANSI-aware (foldWrap inside): styled text is safe — escapes are
// consumed atomically and never counted against the width, and a token
// that alone busts the row hard-splits at the cell edge (never overflow,
// never clip: every glyph survives on some row).
func wrapPlain(s string, w int) string {
	return foldWrap(s, w, w)
}

// foldStyledLines hard-folds every line of an ALREADY-SHAPED block (one
// glamour-rendered boss turn, post cleanMarkdown) to w display cells,
// ANSI-aware. Glamour's own word wrap misses code fences and unbreakable
// URLs — without this pass those rows overflow the bubble, and the
// viewport's soft wrap would burst them mid-word/mid-glyph with no
// hanging indent. Word boundaries are kept where they exist; unbreakable
// tokens hard-split at the cell edge. Lines that fit pass through
// VERBATIM (fence indentation survives).
func foldStyledLines(lines []string, w int) []string {
	var out []string
	for _, ln := range lines {
		out = append(out, foldStyledRows(ln, w, w)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// foldStyledRows folds ONE visual line to firstW cells for the first row
// and contW for every continuation row — write the first row after the
// bubble/hanging prefix and continuations under a matching indent so
// prefix + row cells stay inside the panel. ANSI-aware; each returned row
// is a clean single row (no newlines).
func foldStyledRows(line string, firstW, contW int) []string {
	rows := strings.Split(foldWrap(line, firstW, contW), "\n")
	if len(rows) == 0 {
		return []string{""}
	}
	return rows
}

// foldWrap greedy word-wraps one visual line (or '\n'-joined paragraphs)
// to firstW cells, continuation rows to contW. ANSI escape sequences cost
// 0 cells and are copied through atomically (see seqLen); wide glyphs are
// measured and split at grapheme-cluster boundaries. A paragraph that
// fits is kept VERBATIM (code-fence indentation survives the pass).
func foldWrap(s string, firstW, contW int) string {
	if firstW < 1 {
		firstW = 1
	}
	if contW < 1 {
		contW = 1
	}
	width := firstW
	var out []string
	var cur strings.Builder
	curW := 0
	flush := func() {
		out = append(out, cur.String())
		cur.Reset()
		curW = 0
		width = contW
	}
	for pi, para := range strings.Split(s, "\n") {
		if pi > 0 {
			flush() // an explicit newline always owns a row boundary
		}
		if pw := ansi.StringWidth(para); pw <= width {
			cur.WriteString(para)
			curW = pw
			continue
		}
		for _, word := range splitStyledWords(para) {
			ww := ansi.StringWidth(word)
			if curW > 0 {
				if curW+1+ww <= width {
					cur.WriteString(" " + word)
					curW += 1 + ww
					continue
				}
				flush()
			}
			// the word heads a row; an unbreakable token that busts the
			// budget on its own hard-splits at the cell edge
			for ww > width {
				head, rest := cutStyled(word, width)
				if head == "" {
					break // unsplittable glyph wider than the row: overflow it whole, never clip
				}
				out = append(out, head)
				width = contW
				word = rest
				ww = ansi.StringWidth(word)
			}
			cur.WriteString(word)
			curW = ww
		}
	}
	flush()
	return strings.Join(out, "\n")
}

// splitStyledWords cuts a styled line into whitespace-separated words;
// escape sequences ride with the word they style (they split atomically
// via seqLen, so a fold never shreds one). Whitespace itself is always a
// separator — fold boundaries own the spacing.
func splitStyledWords(s string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); {
		if n := seqLen(s, i); n > 0 {
			cur.WriteString(s[i : i+n])
			i += n
			continue
		}
		if s[i] == ' ' || s[i] == '\t' {
			flush()
			i++
			continue
		}
		cur.WriteByte(s[i])
		i++
	}
	flush()
	return words
}

// cutStyled takes the longest HEAD of word that fits budget display cells
// (escape sequences always ride along, 0 cells; graphemes stay whole —
// ansi.FirstGraphemeCluster measures them). Returns ("", word) when the
// FIRST printable glyph alone busts the budget: there is nothing to cut,
// and the caller must overflow the glyph rather than destroy it.
func cutStyled(word string, budget int) (head, rest string) {
	i, cells := 0, 0
	for i < len(word) {
		if n := seqLen(word, i); n > 0 {
			i += n
			continue
		}
		cl, w := ansi.FirstGraphemeCluster(word[i:], ansi.GraphemeWidth)
		if cells+w > budget {
			if cells == 0 {
				return "", word
			}
			return word[:i], word[i:]
		}
		cells += w
		i += len(cl)
	}
	return word, ""
}

// seqLen returns the byte length of the escape sequence starting at s[i]
// when s[i] is ESC/CSI, 0 otherwise — CSI (ESC [ / 0x9b), OSC (ESC ],
// BEL/ST-terminated), and plain two-byte ESC·final forms. Our rendered
// content is lipgloss SGR, but the splitter must not trip over a
// pasted-content sequence either.
func seqLen(s string, i int) int {
	if s[i] != 0x1b && s[i] != 0x9b {
		return 0
	}
	j := i + 1
	if j >= len(s) {
		return 1
	}
	csi := s[i] == 0x9b
	if !csi && s[j] == '[' {
		csi, j = true, j+1
	}
	if csi {
		for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3f {
			j++ // parameter bytes
		}
		for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2f {
			j++ // intermediate bytes
		}
		if j < len(s) {
			j++ // final byte
		}
		return j - i
	}
	if s[j] == ']' { // OSC — runs to BEL or ST
		j++
		for j < len(s) {
			if s[j] == 0x07 {
				return j + 1 - i
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2 - i
			}
			j++
		}
		return len(s) - i
	}
	return 2 // ESC + one final byte
}

// revision is a cheap FNV-1a over every rendered field of the chat slice —
// tool merges replace entries IN PLACE (same ID, changed Meta), and think
// entries append with their own Kind, so a last-message shortcut would miss
// real changes.
func revision(chat []state.ChatMsg) uint64 {
	if len(chat) == 0 {
		return 0
	}
	h := uint64(14695981039346656037)
	mix := func(s string) {
		for i := 0; i < len(s); i++ {
			h ^= uint64(s[i])
			h *= 1099511628211
		}
	}
	for _, m := range chat {
		mix(m.ID)
		mix(m.From)
		mix(m.Kind)
		mix(m.Meta)
		mix(m.Text)
		if m.Pending {
			h ^= 1
			h *= 1099511628211
		}
	}
	return h
}

func cloneChat(in []state.ChatMsg) []state.ChatMsg {
	out := make([]state.ChatMsg, len(in))
	copy(out, in)
	return out
}
