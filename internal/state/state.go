// Package state — the ONE contract backend and UI both speak.
// Port of node-legacy/src/state.ts. UI never calls SDK/HTTP directly;
// backend never renders.
package state

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// Attachment — one chat-input attachment riding the outgoing boss prompt
// as an opencode prompt_async file part ({type:"file", mime, filename,
// url} — url carries the base64 data URL). The chat panel builds them
// (a clipboard-image paste lands in a temp PNG; the @ picker points at a
// repo file); the backend reads Path's bytes at SEND time. The backend
// seam is OPTIONAL: state.Backend.Send stays plain text, and backends
// that take files implement the AttachmentSender seam the app
// type-asserts (the same pattern model.go uses for teamBackend).
type Attachment struct {
	Name string `json:"name"`           // display name AND the wire "filename"
	Mime string `json:"mime,omitempty"` // resolved at attach time; senders re-sniff when empty
	Path string `json:"path"`           // the file the sender base64s into the data URL
	// Temp — non-empty when Path lives in a panel-created temp dir
	// (os.MkdirTemp "theboringoffice-paste-*"): the app removes the dir once the
	// send resolves (best effort — a queued send must find the file at
	// flush time, so removal never happens at enqueue).
	Temp string `json:"-"`
}

// AttachMetaSep splits the user-bubble Meta attachment carrier:
// "att" ␟ name ␟ name… (unit separator — names may contain spaces).
// Written by backends on the EvChatUser echo, read by the chat panel
// renderer for the dim " · 📎 N" suffix. Same separator trick as the
// diff Meta carrier (diffMetaSep in panels/chat.go).
const AttachMetaSep = "\x1f"

// AttachMetaPrefix tags a user bubble's Meta as the attachment carrier
// (every other Meta user — think/tool/question/diff/office-error — owns
// its own Kind or From, so the prefix cannot collide).
const AttachMetaPrefix = "att"

// AttachMeta builds the ChatMsg.Meta carrier from attachment names;
// "" when there are none (a plain user turn keeps an empty Meta).
func AttachMeta(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return AttachMetaPrefix + AttachMetaSep + strings.Join(names, AttachMetaSep)
}

// MediaItem — ONE inbound image riding a completed boss turn (the wire's
// file/image part, reduced to what a transcript preview needs). The
// Meta carrier below holds the SMALL fields (mime/filename/dims/hash);
// the heavy URL (usually a base64 data URL) rides ONLY state.Event.Media
// — ChatMsg.Meta stays snapshot-sized (an 8 MiB payload must never land
// in the persisted session transcript).
type MediaItem struct {
	Mime     string `json:"mime"`               // image/* (required — anything else is dropped at the gate)
	Filename string `json:"filename,omitempty"` // display name for the 🖼 chip
	W        int    `json:"w,omitempty"`        // decoded pixel width (0 = dims unknown)
	H        int    `json:"h,omitempty"`        // decoded pixel height (0 = dims unknown)
	Hash     string `json:"hash,omitempty"`     // DataURLHash(URL) — the payload lookup key; "" means chip-only (remote URL / undecodable)
	URL      string `json:"url,omitempty"`      // the data URL — Event.Media payload carrier only; mirrors must never fetch a remote URL
}

// MediaMetaPrefix tags a BOSS bubble's Meta as the image-attachment
// carrier. Reads distinguishable from AttachMetaPrefix ("att") at the
// separator: "att\x1f" vs "attach\x1f" never collide.
const MediaMetaPrefix = "attach"

// MediaMetaSep re-uses the attach carrier's unit separator.
const MediaMetaSep = AttachMetaSep

// MediaMaxPayloadBytes — the image payload cap EVERY layer enforces
// (8 MiB, the repo's atMaxSize-scale contract): the URL gate rejects
// bigger decoded payloads, and the rasterizer refuses bigger byte
// slices.
const MediaMaxPayloadBytes = 8 << 20

// ParseDataImageURL splits a "data:<mime>;base64,<payload>" URL into the
// declared mime and the decoded bytes. The gate is strict on purpose —
// the URL arrived off the wire, so never trust the filename: only
// "data:" URLs, only "image/" mimes, only ";base64" encodings, and only
// payloads that decode inside MediaMaxPayloadBytes. A remote http(s) URL
// never reaches here (callers skip external fetches — the security cap).
func ParseDataImageURL(u string) (mime string, raw []byte, err error) {
	if !strings.HasPrefix(u, "data:") {
		return "", nil, errors.New("not a data URL")
	}
	comma := strings.Index(u, ",")
	if comma < 0 {
		return "", nil, errors.New("data URL without a payload separator")
	}
	head := strings.ToLower(strings.TrimPrefix(u[:comma], "data:"))
	if !strings.HasSuffix(head, ";base64") {
		return "", nil, errors.New("data URL not base64-encoded")
	}
	mime = strings.TrimSuffix(head, ";base64")
	if !strings.HasPrefix(mime, "image/") {
		return "", nil, errors.New("data URL mime is not an image")
	}
	raw, err = base64.StdEncoding.DecodeString(u[comma+1:])
	if err != nil {
		return "", nil, errors.New("data URL bad base64")
	}
	if len(raw) > MediaMaxPayloadBytes {
		return "", nil, errors.New("image payload over the 8 MiB cap")
	}
	return mime, raw, nil
}

// DataURLHash — the deterministic payload key the chat carrier speaks:
// sha1(url)[:12] hex. It keys BOTH the backend's emit dedupe touches and
// the app's payload buffer + the panel's raster store (collisions across
// one transcript are inconsequential: same hash ⇒ same URL ⇒ same image).
func DataURLHash(u string) string {
	sum := sha1.Sum([]byte(u))
	return hex.EncodeToString(sum[:6])
}

// MediaMeta builds the ChatMsg.Meta carrier from image MediaItems:
// "attach" ␟ N ␟ (mime ␟ filename ␟ W ␟ H ␟ hash)… — N image groups, in
// wire order. "" when there are none. The URL rides the Event.Media
// sibling, keyed by Hash (deterministic: sha1(url)[:12]).
func MediaMeta(items []MediaItem) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(MediaMetaPrefix + MediaMetaSep + strconv.Itoa(len(items)))
	for _, it := range items {
		b.WriteString(MediaMetaSep + it.Mime + MediaMetaSep + it.Filename + MediaMetaSep +
			strconv.Itoa(it.W) + MediaMetaSep + strconv.Itoa(it.H) + MediaMetaSep + it.Hash)
	}
	return b.String()
}

// ParseMediaMeta decodes the image carrier back into MediaItems (URL/hash
// lookups only — the heavy payload never rides Meta). ok is false when
// Meta isn't an image carrier at all (plain boss turns parse false).
func ParseMediaMeta(meta string) (items []MediaItem, ok bool) {
	if !strings.HasPrefix(meta, MediaMetaPrefix+MediaMetaSep) {
		return nil, false
	}
	fields := strings.Split(meta, MediaMetaSep)
	// attach, N, then N×5 fields (a truncated carrier degrades-silently).
	if len(fields) < 2 {
		return nil, false
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil || n < 1 || len(fields) < 2+5*n {
		return nil, false
	}
	items = make([]MediaItem, 0, n)
	for i := 0; i < n; i++ {
		base := 2 + 5*i
		w, _ := strconv.Atoi(fields[base+2])
		h, _ := strconv.Atoi(fields[base+3])
		items = append(items, MediaItem{
			Mime: fields[base], Filename: fields[base+1], W: w, H: h, Hash: fields[base+4],
		})
	}
	return items, true
}

// ParseAttachMeta decodes the carrier back into names; ok is false when
// Meta isn't an attachment list at all.
func ParseAttachMeta(meta string) (names []string, ok bool) {
	if !strings.HasPrefix(meta, AttachMetaPrefix+AttachMetaSep) {
		return nil, false
	}
	rest := strings.TrimPrefix(meta, AttachMetaPrefix+AttachMetaSep)
	if rest == "" {
		return nil, false
	}
	return strings.Split(rest, AttachMetaSep), true
}

// SpriteState — where an employee is / what they're doing (drives glyphs+walkers).
type SpriteState string

const (
	SpriteAtDesk    SpriteState = "at-desk"
	SpriteWorking   SpriteState = "working"
	SpriteToManager SpriteState = "to-manager"
	SpriteMeeting   SpriteState = "meeting"
	SpriteToDesk    SpriteState = "to-desk"
	SpriteToCoffee  SpriteState = "to-coffee"
	SpriteCoffee    SpriteState = "coffee"
	SpriteAtMailbox SpriteState = "at-mailbox"
)

// EmployeeRole — the office seat.
type EmployeeRole string

const (
	RoleManager   EmployeeRole = "manager"
	RoleHR        EmployeeRole = "hr"
	RoleDeveloper EmployeeRole = "developer"
	RoleScout     EmployeeRole = "scout"
	RoleReviewer  EmployeeRole = "reviewer"
	RoleRunner    EmployeeRole = "runner"
	RoleCTO       EmployeeRole = "cto"
)

// IsArchitectureBrief — the ONE architecture-brief matcher for task
// titles. A brief is architecture-flavored when its title names an
// architecture activity: it contains "architect", "design" or "review"
// (case-insensitive substring match — "designer" trips "design" too; the
// contract is deliberately the dumb substring, not NL). Both backends
// consult this — never re-check locally: the demo's ad-hoc dispatch
// routing and the live roleFromSession mapping share it, and it is the
// unit-tested surface (no second copy may grow anywhere).
func IsArchitectureBrief(title string) bool {
	hay := strings.ToLower(title)
	return strings.Contains(hay, "architect") ||
		strings.Contains(hay, "design") ||
		strings.Contains(hay, "review")
}

type Employee struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Role   EmployeeRole `json:"role"`
	Seat   string       `json:"seat"`
	Sprite SpriteState  `json:"sprite"`
	Task   string       `json:"task,omitempty"`
}

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in-progress"
	TaskDone       TaskStatus = "done"
)

type BoardTask struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Status TaskStatus `json:"status"`
	Owner  string     `json:"owner,omitempty"`
	At     int64      `json:"at"`
}

type MailKind string

const (
	MailBrief  MailKind = "brief"
	MailReturn MailKind = "return"
	MailNotice MailKind = "notice"
	MailUser   MailKind = "user"
)

type MailItem struct {
	ID      string   `json:"id"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	At      int64    `json:"at"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Kind    MailKind `json:"kind"`
}

type ChatMsg struct {
	ID      string `json:"id"`
	From    string `json:"from"` // "user" | "boss" | "office"
	Text    string `json:"text"`
	At      int64  `json:"at"`
	Pending bool   `json:"pending,omitempty"`
	// Kind — "user" | "boss" | "think" | "tool" | "office". Empty "" keeps
	// existing literals valid (classic user/boss chat).
	//
	// "office" (the concierge) follows the EXACT same streaming contract as
	// "boss": EvChatOffice carries Msg{ID:"office-"+<messageID>, From:
	// "office", Kind:"office", Pending:true, Text:accumulated-so-far} —
	// repeated updates of the SAME Msg.ID grow the one bubble; the
	// completion pin re-emits that ID with Pending:false and the pinned
	// full text. UI: update-in-place by Msg.ID, never append a streaming
	// update as a new bubble.
	//
	// "boss" STREAMING contract (live text-part deltas): while the boss's
	// final answer streams in, EvChatBoss carries Msg{ID:"bossmsg-"+<messageID>,
	// Kind:"boss", Pending:true, Text:accumulated-so-far} — repeated updates
	// of the SAME Msg.ID grow the one bubble. The completion pin re-emits the
	// same ID with Pending:false and the pinned full text; the UI replaces
	// the streaming bubble with it. A stream that dies before completion
	// (abort/error/stop) ends Pending:false with a "[theboringfloor] stream
	// interrupted" note appended. UI: update-in-place by Msg.ID, never
	// append a streaming update as a new bubble.
	Kind string `json:"kind,omitempty"`
	// Meta — short decoration, e.g. "read · src/main.go". Empty for plain chat.
	Meta string `json:"meta,omitempty"`
}

// ModelInfo — one switchable boss model of the /model picker (bare
// /model). Provider + ID are the wire halves of the "provider/model"
// ModelRef the free-form /model command already sets; Name is the
// serve's optional display label ("Claude Sonnet 4.5") — empty means the
// ID renders (the picker never blanks a row). Fed on demand by the
// backend's ListModels seam (GET /provider on the live wire; fixed
// fixtures in demo/harness stubs) — never by an event: listings answer
// a click, they don't stream, so no EvModels kind exists (the same
// call-and-render shape the /session picker's ListSessions uses).
type ModelInfo struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
}

// QuestionOption is one selectable answer of a boss question popover.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// Preview — the CLI AskUserQuestion option's optional markdown preview
	// (wire: questions[].options[].preview, a plain string). The office
	// never renders it today: it rides the event so the claude backend's
	// dialog answer can re-emit it inside updatedInput (the CLI's own
	// submit builder spreads the original input), keeping the granted
	// tool call's full original context. ADDITIVE, omitempty.
	Preview string `json:"preview,omitempty"`
}

// QuestionItem is one page of a question.asked request: its text, an
// optional header chip, its selectable options, and whether several
// options may be picked (checkbox) or exactly one (radio). A question
// with NO options is a free-text (textarea) page.
type QuestionItem struct {
	Question string           `json:"question"`
	Header   string           `json:"header,omitempty"`
	Options  []QuestionOption `json:"options,omitempty"`
	Multiple bool             `json:"multiple,omitempty"`
	// Meta/MetaSource — the AskUserQuestion dialog payload's raw
	// metadata/metadataSource (CLI analytics context, "not displayed to
	// the user"), carried VERBATIM (raw wire bytes) so the claude
	// backend's dialog answer can re-emit them inside updatedInput.
	// Payload-level (one value per dialog): every page of one dialog
	// carries the same bytes, and result builders read the first
	// carrier. nil for opencode-sourced questions and for dialogs
	// without metadata. ADDITIVE, omitempty.
	Meta       json.RawMessage `json:"meta,omitempty"`
	MetaSource json.RawMessage `json:"metaSource,omitempty"`
}

// SpeechBubble — ambient office chatter balloon, expires after ttl ticks.
type SpeechBubble struct {
	ID         string `json:"id"`
	EmployeeID string `json:"employeeId"`
	Text       string `json:"text"`
	UntilTick  int    `json:"untilTick"`
}

type Mode string

const (
	ModeLive Mode = "live"
	ModeDemo Mode = "demo"
)

type OfficeState struct {
	Employees  []Employee     `json:"employees"`
	Tasks      []BoardTask    `json:"tasks"`
	Mails      []MailItem     `json:"mails"`
	Chat       []ChatMsg      `json:"chat"`
	Bubbles    []SpeechBubble `json:"bubbles"`
	Mode       Mode           `json:"mode"`
	StatusLine string         `json:"statusLine"`
	Tick       int            `json:"tick"`
	// Offline — set while the connectivity watcher reports the network
	// down (EvOffline/EvOnline). UI shows an "[offline]" badge and the
	// backend suspends its pump/poller until EvOnline clears it.
	Offline bool `json:"offline,omitempty"`
	// BossThinking — the primary session is between a prompt and its reply,
	// with a live EvThought open. UI dims the desk glyph / shows a spinner.
	BossThinking bool `json:"bossThinking,omitempty"`
	// BossDelegating — a boss typing placeholder is outstanding but the
	// primary session has been quiet (no stream/thought/primary-tool) for
	// >6 ticks while hired employees are visibly busy (working/to-manager/
	// meeting): the boss is MANAGING, not generating. The UI swaps the
	// noisy typing spinner for a settled "delegating" row and stamps the
	// floor nameplate "[delegat]". Clears the moment any boss-side
	// activity lands.
	BossDelegating bool `json:"bossDelegating,omitempty"`
	// TokensIn/TokensOut/CostUSD — the conversation's REAL usage totals,
	// accumulated reducer-side from EvUsage deltas the backend lifts
	// verbatim out of opencode assistant messages (AssistantMessage
	// .tokens/.cost — required on the wire since serve 1.18.19, verified
	// against GET /doc). Zero while nothing real has been reported; the
	// status bar hides its usage segment in that case rather than
	// estimate. TokensIn counts wire `input`; TokensOut counts wire
	// `output`+`reasoning` (cache read/write stay out of the headline —
	// opencode's own CostUSD is the authoritative bill). Additive state:
	// a new office starts at 0 and the session snapshot never threads
	// them (Snapshot keeps chat only).
	TokensIn  int64   `json:"tokensIn,omitempty"`
	TokensOut int64   `json:"tokensOut,omitempty"`
	CostUSD   float64 `json:"costUsd,omitempty"`
	// TokensCacheRead/TokensCacheWrite — cumulative provider prompt-cache
	// counters (Anthropic/OpenAI), accumulated the same += way. These are
	// INFORMATIONAL ONLY: CostUSD above already prices every cache token
	// (writes at 1.25x, reads at 0.1x) — the fields exist so the member
	// can SEE that prompt caching is actually happening, never to bill.
	TokensCacheRead  int64 `json:"tokensCacheRead,omitempty"`
	TokensCacheWrite int64 `json:"tokensCacheWrite,omitempty"`
	// Models — the most recent provider/model listing the /model picker
	// fetched on demand (bare /model → the backend's ListModels seam).
	// Fetch-on-demand ONLY: no event writes it and nothing polls — the
	// reducer never touches the field, so the last listing simply rides
	// the office state until the next picker open refetches. Additive:
	// a fresh office starts empty, and (like the usage counters) the
	// session snapshot never threads it.
	Models []ModelInfo `json:"models,omitempty"`
	// BackendName — the selected LLM transport ("opencode"|"claudecode"),
	// latched reducer-side off the backend's boot EvStatus name hint
	// ("[theboringfloor] backend: <name>") and the /backend swap line —
	// the same string-marker contract pattern as the agentmemory boot
	// latch. The topbar renders it between mode and agents (the compact
	// bar drops it with mode). "" pre-hint: nothing renders. Additive
	// state: the session snapshot never threads it (each boot's hint
	// re-latches it).
	BackendName string `json:"backendName,omitempty"`
}

// EventKind — Go has no tagged unions; one Event struct with a Kind + optional fields.
type EventKind string

const (
	EvHire      EventKind = "hire"
	EvFire      EventKind = "fire"
	EvDispatch  EventKind = "dispatch"
	EvWorking   EventKind = "working"
	EvReturned  EventKind = "returned"
	EvIdleDrift EventKind = "idle-drift"
	EvBlocked   EventKind = "blocked"
	EvTask      EventKind = "task"
	EvMail      EventKind = "mail"
	EvChatUser  EventKind = "chat-user"
	EvChatBoss  EventKind = "chat-boss"
	// EvChatOffice — the office concierge's chat lane (ADDITIVE). When the
	// boss's turn is occupied the app routes chat through the concierge,
	// whose replies ride this kind instead of EvChatBoss: exactly ONE of
	// the two fires per assistant message. Msg carries From "office",
	// Kind "office", ID "office-"+messageID; the streaming contract
	// (Pending growth + completion pin, 150ms-coalesced deltas) mirrors
	// the boss stream (see ChatMsg.Kind).
	EvChatOffice EventKind = "chat-office"
	EvThought    EventKind = "thought"
	EvTool       EventKind = "tool"
	EvBubble     EventKind = "bubble"
	EvStatus     EventKind = "status"
	EvTick       EventKind = "tick"
	EvPermission EventKind = "permission"
	EvQuestion   EventKind = "question"
	EvFileDiff   EventKind = "diff"
	// EvUsage — a usage/cost delta lifted from one opencode assistant
	// message.updated (AssistantMessage.tokens/.cost). CallID carries the
	// messageID; TokensIn/TokensOut/CostUSD are the GROWTH since the last
	// frame for that same message (never an absolute re-report), so the
	// reducer simply accumulates. Emitted only for sessions the office
	// owns (primary, concierge pseudo-desk, hired children) and only when
	// the delta is non-zero.
	EvUsage EventKind = "usage"
	// EvChatMedia — an inbound IMAGE file/image part sighted on the
	// primary ("boss") session's message stream (ADDITIVE). Never a chat
	// row: the reducer passes it through untouched (unknown-kind default)
	// and the APP buffers the payload model-side (same ownership shape as
	// EvPermission/EvQuestion's model-owned UI state). Msg.ID carries the
	// owning boss bubble's identity ("bossmsg-"+messageID) — the bubble's
	// own completion pin (EvChatBoss) re-announces the same media on its
	// Meta carrier + Event.Media, so a missed SSE frame still previews
	// (both paths dedupe by MediaItem.Hash). CallID carries the wire
	// partID (dedupe + the lazy-rasterize probe key).
	EvChatMedia EventKind = "chat-media"
	// EvOffline/EvOnline — network connectivity transitions from the
	// backend's watcher (pure-Go probe). Emit once per transition.
	EvOffline EventKind = "offline"
	EvOnline  EventKind = "online"
	// EvBrowserOpen — an AGENT-originated request to open a URL in the
	// office's browser tab (ADDITIVE; see internal/browsertools — the
	// marker protocol: the boss agent emits ⟦open-browser: URL⟧ on its
	// own line, the backend strips the marker from the pinned transcript
	// and runs the URL policy BEFORE this event exists). Text carries
	// the requested URL; BrowserOpenAllowed is the policy verdict (false
	// → nothing opens and BrowserOpenReason carries the member-facing
	// refusal). The reducer passes it through untouched (unknown-kind
	// default); the APP owns the reaction (internal/app/browser_open.go).
	EvBrowserOpen EventKind = "browser-open"
	// EvBrowserScreenshot — an AGENT-originated request to RENDER a page
	// for the member (ADDITIVE; the ⟦browser-screenshot: URL⟧ marker):
	// the app runs the headless engine (internal/headless), saves the
	// PNG, flips the left slot to the browser tab, and drives the pane's
	// normal open (the tab's own display path picks the shot up there).
	// Same field contract as EvBrowserOpen, plus the result-leg fields
	// below. The reducer passes it through untouched.
	EvBrowserScreenshot EventKind = "browser-screenshot"
	// EvBrowserSnapshot — an AGENT-originated request to READ a page
	// (ADDITIVE; the ⟦browser-snapshot: URL⟧ marker): the app runs the
	// headless engine and posts the page's text+links BACK to the agent
	// as a synthetic follow-up prompt on the same backend session; the
	// member sees a one-line dim note (never the full text). Same field
	// contract as EvBrowserOpen, plus the result-leg fields below. The
	// reducer passes it through untouched.
	EvBrowserSnapshot EventKind = "browser-snapshot"
	// EvBrowserAction — an AGENT-originated request to MUTATE a page
	// (ADDITIVE; the ⟦browser-action: URL | op⟧ marker — click/fill/eval):
	// ALWAYS gated by the member's permission modal (actions mutate, so
	// even a policy-allowed localhost URL asks first — approve-once only).
	// REQUEST leg: Text = the policy-decided URL, BrowserOpenAllowed/
	// BrowserOpenReason = the verdict (a refusal posts the red reason row
	// and NEVER opens a modal), BrowserActionOp/Sel/Arg = the parsed
	// action. The app parks an allowed request as a modal hold and the
	// member's answer drives execution/rejection (browser_open.go).
	// RESULT leg (BrowserToolDone=true): success re-uses
	// BrowserOpenAllowed=true with BrowserActionFinalURL +
	// BrowserActionResult; failure carries the member/agent-facing error
	// in BrowserOpenReason. The reducer passes it through untouched.
	EvBrowserAction EventKind = "browser-action"
	// EvRecentMessages — a read-only boss request for a synthetic follow-up
	// containing recent office transcript context. RecentMessagesCount is
	// already clamped by internal/chatcontext to 1..50; the reducer passes
	// this additive event to the app, which owns collecting and sending it.
	EvRecentMessages EventKind = "recent-messages"
)

// Event — the wire between backend and the tea.Model. Only fields relevant
// to Kind are populated.
type Event struct {
	Kind       EventKind `json:"kind"`
	Employee   Employee  `json:"employee,omitempty"`   // hire
	EmployeeID string    `json:"employeeId,omitempty"` // fire/dispatch/working/returned/idle/blocked/bubble
	Task       BoardTask `json:"task,omitempty"`       // dispatch/task + returned.TaskID via Task.ID
	TaskID     string    `json:"taskId,omitempty"`     // working/returned
	Mail       MailItem  `json:"mail,omitempty"`       // returned/mail
	Msg        ChatMsg   `json:"msg,omitempty"`        // chat-user/chat-boss/chat-office
	Text       string    `json:"text,omitempty"`       // status note / bubble text / thought text
	TTL        int       `json:"ttl,omitempty"`        // bubble
	// EmployeeName — human label for the actor. The backend fills it from
	// the employee registry ("boss" for the primary session, "tekton-1"
	// etc. for children) so the UI never has to resolve an ID back to a name.
	EmployeeName string `json:"employeeName,omitempty"` // thought/tool
	// Tool fields (EvTool).
	ToolName    string `json:"toolName,omitempty"`
	ToolSummary string `json:"toolSummary,omitempty"` // e.g. "src/main.go" or "THEBORINGOFFICE_*, 12 hits"
	ToolState   string `json:"toolState,omitempty"`   // "running" | "done" | "error"
	CallID      string `json:"callId,omitempty"`      // part/call id for dedupe
	// ToolOutput — the tool's RESULT text on the done/error event,
	// human-readable and plain (ADDITIVE). claude: the tool_result
	// block's text content, multi-block text joined with newlines,
	// structured/non-text content rendered compactly ("[image data]").
	// opencode: the completed tool state's output string verbatim (an
	// errored state carries its error text — the UI shows both the same
	// way; the error styling stays ToolState's business). Both backends
	// cap it at 8000 bytes, tail-kept with a leading "…" when trimmed
	// (errors and exits live at the tail), so multi-megabyte outputs
	// never grow memory. "" on running events and when the tool
	// returned nothing — the UI renders its own empty state.
	ToolOutput string `json:"toolOutput,omitempty"`
	Done       bool   `json:"done,omitempty"` // thought completion
	// Permission/question/diff fields (EvPermission/EvQuestion/EvFileDiff).
	// SessionID is the opencode session the event belongs to ("boss"-side
	// events carry the primary id); PermissionID/QuestionID are the wire
	// request ids (per…/que…) the UI hands back to AnswerPermission.
	PermissionID string `json:"permissionId,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
	QuestionID   string `json:"questionId,omitempty"`
	// Questions — the STRUCTURED pages of an EvQuestion popover (radio /
	// checkbox / textarea), one QuestionItem per asked question. ADDITIVE:
	// Text/ToolSummary still carry the legacy flattened "a | b | c" one-
	// liner for history/list views, and backends leave this nil until the
	// structured question wire (ocQuestionInfo) feeds it.
	Questions []QuestionItem `json:"questions,omitempty"`
	DiffPath  string         `json:"diffPath,omitempty"` // file path relative to the working dir
	DiffBody  string         `json:"diffBody,omitempty"` // compact unified diff, capped by the backend
	DiffAdd   int            `json:"diffAdd,omitempty"`
	DiffDel   int            `json:"diffDel,omitempty"`
	// Usage fields (EvUsage): per-message DELTAS, not absolutes — the
	// backend remembers what each messageID already reported and sends
	// only the growth, so the app accumulates with += and repeated
	// message.updated frames can never double-count.
	TokensIn  int64   `json:"tokensIn,omitempty"`  // wire tokens.input growth
	TokensOut int64   `json:"tokensOut,omitempty"` // wire tokens.output+reasoning growth
	CostUSD   float64 `json:"costUsd,omitempty"`   // wire cost growth (USD, opencode-computed)
	// Cache read/write growth, copied verbatim off tokens.cache.{read,write}.
	// Kept OUT of the headline token totals (provider-billing overlap) and
	// informational only: CostUSD above already prices cache. The app
	// surfaces these so the operator can verify prompt caching is live.
	TokensCacheRead  int64 `json:"tokensCacheRead,omitempty"`  // wire tokens.cache.read growth
	TokensCacheWrite int64 `json:"tokensCacheWrite,omitempty"` // wire tokens.cache.write growth
	// Media — inbound boss-turn images (EvChatBoss completion pin /
	// EvChatMedia): the ONE payload channel for image bytes (the data
	// URLs). Kept OFF ChatMsg so the session snapshot never threads
	// multi-KB blobs; the app buffers them on receipt, keyed by
	// MediaItem.Hash, and drops the event copy. nil on every other kind.
	Media []MediaItem `json:"media,omitempty"`
	// Browser-open fields (EvBrowserOpen): the backend's policy verdict
	// on the agent-requested URL riding Text. BrowserOpenAllowed=false
	// means nothing opens and BrowserOpenReason carries the member-facing
	// refusal (the app posts it as a red notice); true means the app
	// drives the browser pane's open path.
	BrowserOpenAllowed bool   `json:"browserOpenAllowed,omitempty"`
	BrowserOpenReason  string `json:"browserOpenReason,omitempty"`
	// Browser screenshot/snapshot RESULT-leg fields (EvBrowserScreenshot/
	// EvBrowserSnapshot). The REQUEST leg carries Text + the verdict
	// exactly like EvBrowserOpen; the app's engine cmd (tea.Cmd) lands
	// the RESULT leg back through Update's state.Event case with
	// BrowserToolDone=true: success re-uses BrowserOpenAllowed=true
	// (BrowserShotPath holds the saved PNG path; BrowserSnapTitle/
	// BrowserSnapLinks describe the snapshot delivered back to the
	// agent), failure keeps BrowserOpenAllowed=false and carries the
	// member-facing engine/save/send error in BrowserOpenReason.
	BrowserToolDone  bool   `json:"browserToolDone,omitempty"`
	BrowserShotPath  string `json:"browserShotPath,omitempty"`
	BrowserSnapTitle string `json:"browserSnapTitle,omitempty"`
	BrowserSnapLinks int    `json:"browserSnapLinks,omitempty"`
	// Browser action fields (EvBrowserAction) — the MUTATING sibling's
	// payload + result. REQUEST leg: BrowserActionOp is "click" | "fill" |
	// "eval", BrowserActionSel the CSS selector (click/fill),
	// BrowserActionArg the fill value or the eval JS expression. RESULT
	// leg: BrowserActionFinalURL is the post-action location,
	// BrowserActionResult the op's result text ("clicked <sel>" /
	// "filled <sel>" / the eval JSON, 4KB rune-safe capped by the
	// engine). No existing field can carry these (the permission modal
	// itself reuses the PermissionID/ToolName/ToolSummary fields).
	BrowserActionOp       string `json:"browserActionOp,omitempty"`
	BrowserActionSel      string `json:"browserActionSel,omitempty"`
	BrowserActionArg      string `json:"browserActionArg,omitempty"`
	BrowserActionFinalURL string `json:"browserActionFinalUrl,omitempty"`
	BrowserActionResult   string `json:"browserActionResult,omitempty"`
	// RecentMessagesCount is the requested transcript-message count for
	// EvRecentMessages. It is meaningful only for that event kind.
	RecentMessagesCount int `json:"recentMessagesCount,omitempty"`
}

// MCPServer is one configured MCP server with its live status as the
// backend last reported it (tools count/status fields are best-effort —
// older serves return less).
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`           // connected|disabled|needs_auth|failed|unknown
	Detail string `json:"detail,omitempty"` // error text / tool count / auth hint
}

// Backend — one per run. Demo scripted, live via opencode serve + agentmemory.
type Backend interface {
	Mode() Mode
	// Start wires events; f MUST be safe to call from backend goroutines
	// (the app hands it tea.Program.Send).
	Start(emit func(Event)) error
	// Send pushes user chat to the boss. Plain text on purpose: chat-input
	// attachments ride an OPTIONAL second seam — SendWith(text, atts
	// []Attachment) — that the app type-asserts (see attachmentBackend in
	// internal/app/model.go, the same pattern as the team board seam).
	// Keeping them out of this interface means harness stubs (uishot,
	// headless) unchanged: they simply never attach files.
	Send(text string) error
	// AnswerPermission replies to a pending permission prompt. response is
	// "once" | "always" | "reject" (opencode serve permission.reply enum).
	AnswerPermission(permissionID, response string) error
	// AnswerQuestion replies to a pending question request (the boss used
	// the question tool; the agent loop is PARKED at question.asked until
	// the question API gets this reply — a normal chat prompt does NOT
	// answer it, which is the question-loop deadlock). answers is one
	// string array per asked question, in order — QuestionItem.Multiple
	// (checkbox) pages put several labels in THEIR array, radio/free-text
	// pages put exactly one; the backend ships it verbatim (payload
	// answers = string[][], opencode's QuestionAnswer = string[]).
	AnswerQuestion(requestID string, answers [][]string) error
	// RejectQuestion declines a pending question request outright, freeing
	// the parked turn without an answer (opencode serve exposes a true
	// reject route: POST /question/{requestID}/reject, /doc 1.18.19).
	RejectQuestion(requestID string) error
	// MCPServers lists the configured MCP servers with live status.
	MCPServers() ([]MCPServer, error)
	// ReconnectMCP asks the server to re-establish one MCP connection.
	ReconnectMCP(name string) error
	Stop() error
}

// ConciergeCapable — the office-concierge seam (ADDITIVE; deliberately NOT
// folded into Backend, same convention as the attachment/team/office-session/
// abort seams — harness stubs stay untouched, the app type-asserts it).
//
// The concierge keeps the member answered while the boss's turn is occupied:
// SendConcierge delivers chat to a lightweight side session ("theboringoffice
// concierge", lazily created on FIRST use on the live backend) that answers
// instantly — directly when the message is trivial, by dispatching its own
// developer sub-agents (tracked like the boss's children) when it is real
// work. Replies arrive as EvChatOffice bubbles (From "office"), never
// EvChatBoss. When the feature is off (config boss.concierge=false) a backend
// degrades open by treating the message as a normal boss Send. The live
// backend additionally emits the chat-user echo (same ownership as Send, so
// the app never echoes twice).
type ConciergeCapable interface {
	SendConcierge(text string) error
}

// SessionRow — one opencode session as the /session picker lists it
// (the live backend's ListSessions seam — ADDITIVE, like
// ConciergeCapable/SessionAborter below: never folded into Backend, the
// app type-asserts it). ParentID carries the wire's parent linkage (a
// row with a non-empty ParentID is a CHILD session — the picker keeps
// roots only); Created/Updated are unix millis off the wire; Messages is
// the GET /session/{id}/message row count, -1 when that count's fetch
// failed (a count gap must never hide the session itself).
type SessionRow struct {
	ID       string `json:"id"`
	ParentID string `json:"parentID"`
	Title    string `json:"title"`
	Created  int64  `json:"created"` // unix millis
	Updated  int64  `json:"updated"` // unix millis
	Messages int    `json:"messages"`
}

// SessionAborter — the /stop seam (ADDITIVE; deliberately NOT folded into
// Backend, same convention as the attachment/team/office-session seams —
// see internal/app sessions.go "backend seams (additive; type-asserted,
// never added to state.Backend)": harness stubs stay untouched). Backends
// that can cancel in-flight LLM turns implement it; the app type-asserts
// it when wiring /stop.
//
// AbortSessions cancels the primary ("boss") turn AND every live child
// session's turn — an opencode abort only ends its own session's run, so
// sub-agent work the boss fanned out must be called out session by
// session. Per-session failures are non-fatal: collected into the returned
// error (errors.Join), the sessions that DID abort still stop. The boss's
// outstanding typing placeholder closes with a stopped marker so the UI
// never shows a frozen "typing…" bubble for a dead turn (live backend),
// and the demo backend emits "[demo] abort ok".
type SessionAborter interface {
	AbortSessions() error
}

// ---------------------------------------------------------------- older-history pagination (ADDITIVE)

// SessionMessagePart — one part of a fetched history row, reduced to what
// a transcript splice needs: the part TYPE ("text", "reasoning", "tool",
// …) and its text body. Tool parts keep their type with an empty body —
// pagination splices history for READING, it never replays calls.
type SessionMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// File-part retention ("file"/"image" types): URL + mime + filename
	// ride verbatim off the wire so a future history-side image lane can
	// preview without re-fetching; splicers that don't know them ignore
	// them (empty for text/tool/reasoning parts).
	URL      string `json:"url,omitempty"`
	Mime     string `json:"mime,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// SessionMessageRow — ONE message of a fetched history page: the
// transcript's splice unit. Created/Completed are unix millis off the
// wire's info.time; Parts ride oldest-first exactly as the message was
// authored.
type SessionMessageRow struct {
	ID        string               `json:"id"`
	Role      string               `json:"role"` // "user" | "assistant" | …
	Created   int64                `json:"created"`
	Completed int64                `json:"completed,omitempty"`
	Parts     []SessionMessagePart `json:"parts,omitempty"`
}

// SessionMessagesPage — ONE page of a session's message history. Rows
// run oldest→newest WITHIN the page (the serve's ascending order, the
// transcript splice's input order); NextCursor is the OPAQUE walk
// continuation — feed it back as `before` to fetch the NEXT OLDER page —
// and HasMore is the boolean read on it: the serve omits its
// X-Next-Cursor header on the OLDEST page, so NextCursor "" == the
// transcript's top (HasMore false).
type SessionMessagesPage struct {
	Rows       []SessionMessageRow `json:"rows"`
	NextCursor string              `json:"nextCursor,omitempty"`
	HasMore    bool                `json:"hasMore"`
}

// SessionPager — the older-history pagination seam (ADDITIVE;
// deliberately NOT folded into Backend, the same convention as
// SessionAborter/ConciergeCapable above — harness stubs stay untouched,
// the app type-asserts it).
//
// MessagesPage fetches ONE page of a session's message history: the
// NEWEST page when before == "" , the page immediately OLDER than the
// opaque cursor otherwise; limit < 1 clamps to 1. The live backend rides
// GET /session/{id}/message?limit=N[&before=cursor] against serve
// 1.18.19 and reads the continuation off the X-Next-Cursor RESPONSE
// header (absent at the top); the demo twin walks a fixed in-memory
// fixture with byte-identical walk semantics so the top-of-transcript
// gesture is dogfoodable without a server.
type SessionPager interface {
	MessagesPage(ctx context.Context, sessionID, before string, limit int) (SessionMessagesPage, error)
}
