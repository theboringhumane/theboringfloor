// claude_events.go — normalize the claude CLI's stream-json JSONL frames
// into state.Events for the office floor. Pure helpers only: no I/O, no
// timers, no UI framework — mirrors events.go (the opencode SSE mapper).
// The live claude backend (claude.go) owns the process; this module
// decides WHAT one stdout line means for the office, given a mutable
// context object.
//
// Wire reference (claude Code >= 2.x, `-p --input-format stream-json
// --output-format stream-json --verbose --include-partial-messages
// --permission-prompt-tool stdio`):
//
//	system/init                      -> status "[claude] init model=.. session=.." (+ mcp pin)
//	stream_event (content_block_*)   -> boss bubble growth / tool folding / thought growth
//	assistant (snapshot)             -> pinned chat-boss ("bossmsg-"+uuid), thoughts, tool_use
//	user (tool_result content)       -> tool done; Task result -> returned + mail + task done
//	control_request can_use_tool     -> EvPermission pending (ID = request_id)
//	control_request request_user_dialog -> EvQuestion pending (ID = request_id)
//	                                     for every dialog_kind in claudeRenderedDialogKinds
//	                                     (the set the office declares via initialize
//	                                     .supportedDialogKinds); every other kind stays
//	                                     parked (never answered, never settled)
//	result (usage/modelUsage)        -> EvUsage DELTAS (running-total, keyed by session id)
//	system/api_retry                 -> status "[claude] retry attempt n/m after Nms"
//	anything else                    -> ignored silently (open-ended parser rule)
package backend

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// ---------------- claude wire shapes ----------------
// Only the fields the mapping reads; unknown fields are ignored by
// encoding/json (the same tolerant rule events.go applies).

type claudeMCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// claudeBlock is one content block of an assistant/user message snapshot.
type claudeBlock struct {
	Type      string         `json:"type"`        // text|thinking|tool_use|tool_result
	Text      string         `json:"text"`        // text
	Thinking  string         `json:"thinking"`    // thinking
	ID        string         `json:"id"`          // tool_use id ("toolu_…")
	Name      string         `json:"name"`        // tool name ("Bash","Write","Task",…)
	Input     map[string]any `json:"input"`       // tool_use input
	ToolUseID string         `json:"tool_use_id"` // tool_result owner
	Content   any            `json:"content"`     // tool_result body (string|[]blocks)
	IsError   bool           `json:"is_error"`    // tool_result failure flag
}

type claudeMessage struct {
	ID      string        `json:"id"`
	Role    string        `json:"role"`
	Model   string        `json:"model"`
	Content []claudeBlock `json:"content"`
}

// claudeUsage is the token blob of a result frame (running totals across
// the conversation's turns).
type claudeUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
}

func (u claudeUsage) zero() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheReadInputTokens == 0 && u.CacheCreationInputTokens == 0
}

// claudeControlRequest is the request blob inside a control_request frame.
// The REAL can_use_tool payload carries tool_name, display_name, input,
// description, permission_suggestions, blocked_path, decision_reason and
// tool_use_id — there is NO tool_input / input_preview on the wire.
// permission_suggestions stays RAW (the CLI-native PermissionUpdate[]):
// an "Allow always" answer re-emits it verbatim as updatedPermissions.
// The REAL request_user_dialog payload carries dialog_kind + payload
// (opaque per kind) + an optional tool_use_id — the CLI 2.1.247 emitter,
// verbatim:
//
//	request:{subtype:"request_user_dialog",dialog_kind:r,payload:s,
//	         ...a&&{tool_use_id:a}}
//
// — there are NO flat question/options fields anywhere on the dialog wire.
type claudeControlRequest struct {
	Subtype               string          `json:"subtype"`                // can_use_tool | request_user_dialog
	ToolName              string          `json:"tool_name"`              // can_use_tool
	Input                 map[string]any  `json:"input"`                  // can_use_tool: the tool call's input
	Description           string          `json:"description"`            // can_use_tool: the CLI's own one-line summary
	ToolUseID             string          `json:"tool_use_id"`            // can_use_tool; also dialogs raised by a tool
	PermissionSuggestions json.RawMessage `json:"permission_suggestions"` // can_use_tool: standing-grant candidates
	DialogKind            string          `json:"dialog_kind"`            // request_user_dialog: the registered dialog kind
	Payload               json.RawMessage `json:"payload"`                // request_user_dialog: opaque per kind
}

// claudeEvent is one stdout line off the claude process. RawMessage fields
// keep the mapping open-ended: fields it never reads are tolerated.
type claudeEvent struct {
	Type            string `json:"type"`
	Subtype         string `json:"subtype"`
	SessionID       string `json:"session_id"`
	UUID            string `json:"uuid"`
	ParentToolUseID string `json:"parent_tool_use_id"`

	// system/init + system/*
	Cwd            string            `json:"cwd"`
	Model          string            `json:"model"` // init.model (assistant.model otherwise)
	PermissionMode string            `json:"permissionMode"`
	APIKeySource   string            `json:"apiKeySource"`
	Version        string            `json:"claude_code_version"`
	MCPServers     []claudeMCPServer `json:"mcp_servers"`

	// system/api_retry
	Attempt      int   `json:"attempt"`
	MaxAttempts  int   `json:"max_attempts"`
	RetryDelayMs int64 `json:"retry_delay_ms"`

	// control frames
	RequestID string               `json:"request_id"`
	Request   claudeControlRequest `json:"request"`

	// assistant / user frames
	Message claudeMessage `json:"message"`

	// stream_event envelope (the partial-message channel)
	Event json.RawMessage `json:"event"`

	// result frame
	IsError      bool                       `json:"is_error"`
	Result       string                     `json:"result"`
	DurationMs   int64                      `json:"duration_ms"`
	NumTurns     int                        `json:"num_turns"`
	TotalCostUSD float64                    `json:"total_cost_usd"`
	Usage        *claudeUsage               `json:"usage"`
	ModelUsage   map[string]json.RawMessage `json:"modelUsage"`
	Errors       []string                   `json:"errors"`
}

// claudeStreamInner is the `event` payload of a stream_event frame — the
// Anthropic message-stream protocol (message_start / content_block_start /
// content_block_delta / content_block_stop / message_delta).
type claudeStreamInner struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`         // text_delta | thinking_delta | input_json_delta
		Text        string `json:"text"`         // text_delta
		Thinking    string `json:"thinking"`     // thinking_delta
		PartialJSON string `json:"partial_json"` // input_json_delta
	} `json:"delta"`
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
	} `json:"message"`
	ContentBlock struct {
		Type string `json:"type"` // text | thinking | tool_use
		ID   string `json:"id"`   // tool_use
		Name string `json:"name"` // tool_use
	} `json:"content_block"`
}

// ---------------- claude norm context (state, not I/O) ----------------

// claudeToolHold remembers one in-flight tool_use: its floor owner, the
// name, the growing input JSON (folded off stream deltas) and whether its
// live summary ever emitted (completion via snapshot refreshes once).
type claudeToolHold struct {
	callID     string
	name       string // lowercased kind ("bash","write","read",…)
	ownerID    string
	ownerName  string
	inputJSON  string // accumulated partial_json / snapshot-frozen input
	summary    string // last summary LINE emitted (change-dedupe)
	snapshotIn bool   // the assistant snapshot refreshed the input
}

// claudeTask is a hired subagent run (one Task tool_use).
type claudeTask struct {
	toolUseID  string
	employeeID string
	employee   state.Employee
	returned   bool
	lastWorkAt int64
}

// claudeUsageLast mirrors ocUsageLast: counters already emitted for the
// session, so repeated result frames ship only their growth.
type claudeUsageLast = ocUsageLast

// claudePermMeta is the raw can_use_tool payload slice a reply needs
// after the modal answers: Suggestions is the request's
// permission_suggestions verbatim (re-emitted as updatedPermissions on an
// "Allow always"), Input the tool call's input (the summary fallback).
type claudePermMeta struct {
	Suggestions json.RawMessage
	Input       map[string]any
}

// claudeNormCtx is the claude twin of normCtx (mutable, not goroutine-safe —
// the backend holds a mutex). Keying: PRIMARY conversation frames carry
// parent_tool_use_id == ""; anything else belongs to the subagent run whose
// Task tool_use id it quotes.
type claudeNormCtx struct {
	cfg *config.Config

	primaryID string // the boss conversation's claude session_id (system/init)
	model     string // init.model (status lines)
	mcp       []state.MCPServer

	// boss text streaming: assistant message uuid -> accumulated delta
	// text / first-delta stamp. Both freed when the snapshot pins.
	textAccum map[string]string
	textStart map[string]int64
	streamed  map[string]bool // uuid -> a stream delta grew this bubble
	pinned    map[string]bool // uuid -> the assistant snapshot pinned final text

	// streaming identity: the message uuid the CURRENT stream's blocks
	// belong to (message_start), plus per-index block kinds and the
	// tool_use id an index opened (input_json_delta folding).
	curMsgUUID  string
	blockKind   map[int]string // content-block index -> "text"|"thinking"|"tool_use"
	blockTool   map[int]string // content-block index -> tool_use.id
	thoughtOpen string         // uuid of an open thinking stream ("" none)
	// thoughtAccum: uuid -> the RAW growing thinking transcript. EvThought
	// carries the accumulated transcript (the opencode contract, events.go):
	// the chat reducer replaces-on-merge, so a bare delta would shrink the
	// rendered thought to its last chunk. Freed on the stream's close.
	thoughtAccum map[string]string
	// msgStreamOpen: the uuid of the MAIN-conversation message whose stream
	// is OPEN (message_start seen, no message_stop yet) — the text-twin of
	// thoughtOpen. While it matches an assistant snapshot's message id the
	// snapshot is a PARTIAL (--include-partial-messages): it REFRESHES the
	// boss bubble without pinning — the pin is one-shot, so freezing on a
	// partial would truncate the bubble forever. Subagent streams
	// (parent_tool_use_id set) never touch it.
	msgStreamOpen string

	// tools / tasks
	tools map[string]*claudeToolHold // tool_use.id -> hold
	tasks map[string]*claudeTask     // Task tool_use.id -> run

	nameCounts map[state.EmployeeRole]int
	seatSeq    int

	pendingPerms     map[string]permHold // request_id -> hold
	pendingQuestions map[string]permHold // request_id -> hold
	// permMeta stashes the raw can_use_tool payload slices an answer
	// needs LATER (the backend-side alternative to threading them through
	// state.Event — the Event surface stays unchanged): the CLI-native
	// permission_suggestions (re-emitted verbatim as updatedPermissions
	// when the boss picks "Allow always") plus the tool input.
	permMeta map[string]claudePermMeta // request_id -> stash
	// dialogMeta stashes the per-kind decode a PENDING request_user_dialog
	// answer needs later (the permMeta analog): family, the F1 tool input,
	// the label->result bytes for label-select kinds, the flagged rules
	// for auto_mode_flagged_allow. Written by mapClaudeControlRequest,
	// read+deleted by AnswerQuestion/RejectQuestion.
	dialogMeta map[string]claudeDialogMeta // request_id -> stash

	usageLast map[string]claudeUsageLast // session_id -> last running totals
	usageDone map[string]bool            // result uuid -> already emitted
}

func newClaudeNormCtx(cfg *config.Config) *claudeNormCtx {
	if cfg == nil {
		cfg = config.Default()
	}
	return &claudeNormCtx{
		cfg:              cfg,
		textAccum:        make(map[string]string),
		textStart:        make(map[string]int64),
		streamed:         make(map[string]bool),
		pinned:           make(map[string]bool),
		blockKind:        make(map[int]string),
		blockTool:        make(map[int]string),
		thoughtAccum:     make(map[string]string),
		tools:            make(map[string]*claudeToolHold),
		tasks:            make(map[string]*claudeTask),
		nameCounts:       make(map[state.EmployeeRole]int),
		pendingPerms:     make(map[string]permHold),
		pendingQuestions: make(map[string]permHold),
		permMeta:         make(map[string]claudePermMeta),
		dialogMeta:       make(map[string]claudeDialogMeta),
		usageLast:        make(map[string]claudeUsageLast),
		usageDone:        make(map[string]bool),
	}
}

// ---------------- small helpers ----------------

// claudeToolName lowercases a wire tool name into the office's glyph kind
// (Bash -> bash, Write -> write, TodoWrite -> todowrite).
func claudeToolName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return "tool"
	}
	return n
}

// claudeToolSummary is the one-liner under a tool glyph: input.file_path
// first, then command, then pattern/url/query, then any short string.
// Reads the hold's accumulated (possibly partial) input JSON with a
// best-effort parse — a streaming prefix that doesn't unmarshal yet still
// yields its first complete string field.
func claudeToolSummary(input map[string]any) string {
	for _, key := range []string{"file_path", "filePath", "command", "pattern", "url", "query", "path"} {
		if v, _ := input[key].(string); strings.TrimSpace(v) != "" {
			return shortTitle(strings.TrimSpace(v), 60)
		}
	}
	for _, v := range input {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return shortTitle(strings.TrimSpace(s), 60)
		}
	}
	return "working"
}

// claudeInputFromJSON best-effort decodes a (possibly partial) JSON blob
// into a map; the prefix is padded with a closing brace for the tolerant
// second pass — a half-written {"file_path": "src/ma parses enough for the
// summary regexes. Never consults the filesystem.
func claudeInputFromJSON(blob string) map[string]any {
	var m map[string]any
	if json.Unmarshal([]byte(blob), &m) == nil && m != nil {
		return m
	}
	// Partial stream: pull the first complete string field with a
	// hand-rolled scan instead of guessing a repair — deterministic and
	// never panics on truncated JSON.
	m = map[string]any{}
	for _, key := range []string{"file_path", "filePath", "command", "pattern", "url", "query", "path"} {
		if s := claudeJSONStringField(blob, key); s != "" {
			m[key] = s
			break
		}
	}
	return m
}

// claudeJSONStringField reads "<key>":"<value>" out of a partial JSON text
// (first match wins; the value must be complete — a truncated string
// yields nothing).
func claudeJSONStringField(blob, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(blob, needle)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(blob[idx+len(needle):])
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimSpace(rest[1:])
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	var out strings.Builder
	for i := 1; i < len(rest); i++ {
		c := rest[i]
		if c == '\\' && i+1 < len(rest) {
			out.WriteByte(rest[i+1])
			i++
			continue
		}
		if c == '"' {
			return out.String()
		}
		out.WriteByte(c)
	}
	return "" // truncated
}

// claudeToolResultText flattens a tool_result content (string or list of
// text blocks) into one summary line.
func claudeToolResultText(content any) string {
	switch c := content.(type) {
	case string:
		return shortTitle(c, 240)
	case []any:
		var parts []string
		for _, cell := range c {
			if m, ok := cell.(map[string]any); ok {
				if t, _ := m["text"].(string); strings.TrimSpace(t) != "" {
					parts = append(parts, strings.TrimSpace(t))
				}
			}
		}
		return shortTitle(strings.Join(parts, " "), 240)
	}
	return ""
}

// ---------------- subagent runs ----------------

// claudeRoleForAgent maps a Task's subagent_type/description onto an office
// role — the SAME matcher the opencode backend leans on (events.go
// roleFromSession), so roster naming is one rule across backends.
func claudeRoleForAgent(subagentType, description string) state.EmployeeRole {
	return roleFromSession(description, subagentType)
}

// claudeIssueEmployee mints the floor row for a subagent run: names ride
// the office's Greek bases (never the member's title text), keyed off the
// subagent TYPE the office itself defines, exactly one row per Task call.
func (ctx *claudeNormCtx) claudeIssueEmployee(toolUseID, subagentType, description string) (state.Employee, state.BoardTask, []state.Event) {
	role := claudeRoleForAgent(subagentType, description)
	n := ctx.nameCounts[role] + 1
	ctx.nameCounts[role] = n
	emp := state.Employee{
		ID:     "task-" + toolUseID,
		Name:   ctx.nameBaseClaude(role) + "-" + itoa(n),
		Role:   role,
		Seat:   "desk-" + itoa(ctx.seatSeq+1),
		Sprite: state.SpriteToManager,
		Task:   shortTitle(orTitle(description), 0),
	}
	ctx.seatSeq++
	task := state.BoardTask{
		ID:     "task-" + toolUseID,
		Title:  shortTitle(orTitle(description), 0),
		Status: state.TaskInProgress,
		Owner:  emp.Name,
		At:     nowMs(),
	}
	ctx.tasks[toolUseID] = &claudeTask{toolUseID: toolUseID, employeeID: emp.ID, employee: emp}
	return emp, task, []state.Event{
		{Kind: state.EvHire, Employee: emp},
		{Kind: state.EvDispatch, Task: task, EmployeeID: emp.ID},
	}
}

// nameBaseClaude is the roster base with the cfg override applied (mirrors
// normCtx.nameBase: brain.json Roles[<rolekey>].NamePrefix wins).
func (ctx *claudeNormCtx) nameBaseClaude(role state.EmployeeRole) string {
	if rc, ok := ctx.cfg.Roles[string(role)]; ok && rc.NamePrefix != "" {
		return rc.NamePrefix
	}
	return nameBase(role)
}

// taskFor resolves the subagent run a parent-tool-use frame belongs to.
func (ctx *claudeNormCtx) taskFor(parentToolUseID string) (*claudeTask, bool) {
	t, ok := ctx.tasks[parentToolUseID]
	return t, ok
}

// throttledClaudeWorking caps "working" pulses at one per 500ms per
// subagent (mirrors normCtx.throttledWorking).
func (ctx *claudeNormCtx) throttledClaudeWorking(t *claudeTask, now int64, force bool) []state.Event {
	if !force && now-t.lastWorkAt < 500 {
		return nil
	}
	t.lastWorkAt = now
	return []state.Event{{Kind: state.EvWorking, EmployeeID: t.employeeID, TaskID: "task-" + t.toolUseID}}
}

// ---------------- tool holds ----------------

// claudeToolStart registers (or refreshes) a tool_use hold and returns the
// running EvTool when the summary line actually changed.
func (ctx *claudeNormCtx) claudeToolStart(callID, name, ownerID, ownerName string, input map[string]any) []state.Event {
	if callID == "" {
		return nil
	}
	kind := claudeToolName(name)
	summary := claudeToolSummary(input)
	h := ctx.tools[callID]
	if h == nil {
		h = &claudeToolHold{callID: callID, ownerID: ownerID, ownerName: ownerName}
		ctx.tools[callID] = h
	}
	h.name = kind
	h.ownerID, h.ownerName = ownerID, ownerName
	if h.summary == summary && summary != "working" {
		return nil
	}
	h.summary = summary
	return []state.Event{{
		Kind: state.EvTool, EmployeeID: ownerID, EmployeeName: ownerName,
		ToolName: kind, ToolSummary: summary, ToolState: "running", CallID: callID,
	}}
}

// claudeToolFold grows a hold's input JSON off a stream delta and emits a
// refreshed running EvTool when the derived summary changed.
func (ctx *claudeNormCtx) claudeToolFold(callID, partial string) []state.Event {
	h := ctx.tools[callID]
	if h == nil || partial == "" || h.snapshotIn {
		return nil // unknown call or already frozen by the snapshot
	}
	h.inputJSON += partial
	summary := claudeToolSummary(claudeInputFromJSON(h.inputJSON))
	if summary == h.summary || summary == "working" {
		return nil
	}
	h.summary = summary
	return []state.Event{{
		Kind: state.EvTool, EmployeeID: h.ownerID, EmployeeName: h.ownerName,
		ToolName: h.name, ToolSummary: summary, ToolState: "running", CallID: callID,
	}}
}

// claudeToolFinish closes a hold on tool_result: done (or error) on the
// same CallID. Unknown call ids close nothing (tolerant parse).
func (ctx *claudeNormCtx) claudeToolFinish(callID string, isErr bool, resultText string) []state.Event {
	h := ctx.tools[callID]
	if h == nil {
		return nil
	}
	delete(ctx.tools, callID)
	toolState := "done"
	summary := h.summary
	if isErr {
		toolState = "error"
		if s := shortTitle(resultText, 60); s != "untitled brief" {
			summary = s
		}
	}
	return []state.Event{{
		Kind: state.EvTool, EmployeeID: h.ownerID, EmployeeName: h.ownerName,
		ToolName: h.name, ToolSummary: summary, ToolState: toolState, CallID: callID,
	}}
}

// ---------------- boss text streaming ----------------

// claudeTextDelta folds one main-conversation text delta into the growing
// boss bubble ("bossmsg-"+uuid, Pending:true).
func (ctx *claudeNormCtx) claudeTextDelta(uuid, delta string, now int64) []state.Event {
	if uuid == "" || delta == "" {
		return nil
	}
	if ctx.textStart[uuid] == 0 {
		ctx.textStart[uuid] = now
	}
	accumulated := capBossText(ctx.textAccum[uuid] + delta)
	ctx.textAccum[uuid] = accumulated
	ctx.streamed[uuid] = true
	return []state.Event{{
		Kind: state.EvChatBoss,
		Msg: state.ChatMsg{
			ID: "bossmsg-" + uuid, From: "boss", Kind: "boss",
			Text: accumulated, At: ctx.textStart[uuid], Pending: true,
		},
	}}
}

// claudeAssistantPin closes an assistant message's bubble with the
// SNAPSHOT's pinned text (Pending:false on the same "bossmsg-"+uuid id).
// Idempotent per uuid and never double-grows a streamed bubble: the pin
// REPLACES the streaming text (and a snapshot that arrives with no stream
// mints the bubble outright). An empty snapshot (tool-only message) is
// mid-turn protocol and pins nothing.
//
// --include-partial-messages wrinkle: PARTIAL snapshots share the
// message's id and land MID-stream (before message_stop), possibly
// carrying INCOMPLETE text. While the message's stream is still open
// (msgStreamOpen == uuid) the snapshot REFRESHES the growing bubble
// (Pending:true, never shorter than the text already streamed) WITHOUT
// pinning — the pin is one-shot, so freezing on a partial would truncate
// the bubble and ignore the complete text. The bubble pins at the
// stream's message_stop (claudeStreamClosePin) or at the first snapshot
// that arrives once the stream has closed. (The text-twin of the thought
// rule: Done: ctx.thoughtOpen != uuid.)
func (ctx *claudeNormCtx) claudeAssistantPin(uuid, text string, now int64) []state.Event {
	if uuid == "" || ctx.pinned[uuid] {
		return nil
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if ctx.msgStreamOpen != "" && ctx.msgStreamOpen == uuid {
		// mid-stream PARTIAL snapshot: refresh the growing bubble with the
		// snapshot's text, still pending, NOT pinned. textAccum tracks the
		// LONGEST known text (a stale partial never shrinks the rendered
		// bubble) so the message_stop close-pin and the abort flush always
		// land the most complete text seen.
		if ctx.textStart[uuid] == 0 {
			ctx.textStart[uuid] = now
		}
		accumulated := capBossText(strings.TrimSpace(text))
		if len(ctx.textAccum[uuid]) > len(accumulated) {
			accumulated = ctx.textAccum[uuid]
		}
		ctx.textAccum[uuid] = accumulated
		return []state.Event{{
			Kind: state.EvChatBoss,
			Msg: state.ChatMsg{
				ID: "bossmsg-" + uuid, From: "boss", Kind: "boss",
				Text: accumulated, At: ctx.textStart[uuid], Pending: true,
			},
		}}
	}
	ctx.pinned[uuid] = true
	at := ctx.textStart[uuid]
	if at == 0 {
		at = now
	}
	delete(ctx.textAccum, uuid)
	delete(ctx.textStart, uuid)
	delete(ctx.streamed, uuid)
	return []state.Event{{
		Kind: state.EvChatBoss,
		Msg: state.ChatMsg{
			ID: "bossmsg-" + uuid, From: "boss", Kind: "boss",
			Text: strings.TrimSpace(text), At: at, Pending: false,
		},
	}}
}

// claudeStreamClosePin pins the boss bubble when the message's stream
// CLOSES (message_stop): the real wire sends NO final assistant snapshot
// after message_stop (proven by capture — the last assistant frame is the
// mid-stream PARTIAL), so the close itself must land the final
// Pending:false bubble or the refreshed text would hang pending forever.
// Pins with the best-known text: textAccum holds the accumulated deltas
// kept at max(deltas, partial snapshots) by the refresh path. Silent when
// the bubble already pinned or no text ever grew (tool-only message); the
// caller gates parent == "" so subagent streams never reach it.
func (ctx *claudeNormCtx) claudeStreamClosePin(uuid string, now int64) []state.Event {
	if uuid == "" || ctx.pinned[uuid] {
		return nil
	}
	if strings.TrimSpace(ctx.textAccum[uuid]) == "" {
		return nil
	}
	return ctx.claudeAssistantPin(uuid, ctx.textAccum[uuid], now)
}

// claudeInterruptedStreamEvents is the claude twin of
// interruptedStreamEvents: every open stream flushes as a final
// Pending=false bubble carrying the interrupted note (abort/stop).
func claudeInterruptedStreamEvents(ctx *claudeNormCtx, note string, now int64) []state.Event {
	var ids []string
	for uuid, accum := range ctx.textAccum {
		if strings.TrimSpace(accum) != "" {
			ids = append(ids, uuid)
		}
	}
	var evs []state.Event
	for _, uuid := range ids {
		if ctx.pinned[uuid] {
			continue
		}
		at := ctx.textStart[uuid]
		if at == 0 {
			at = now
		}
		evs = append(evs, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID:      "bossmsg-" + uuid,
			From:    "boss",
			Kind:    "boss",
			Text:    ctx.textAccum[uuid] + "\n" + note,
			At:      at,
			Pending: false,
		}})
		ctx.pinned[uuid] = true
	}
	for k := range ctx.textAccum {
		delete(ctx.textAccum, k)
	}
	for k := range ctx.textStart {
		delete(ctx.textStart, k)
	}
	for k := range ctx.streamed {
		delete(ctx.streamed, k)
	}
	return evs
}

// ---------------- usage ----------------

// claudeTotalUsage combines a result frame's `usage` with a per-model
// `modelUsage` fallback (when fields are missing): the running totals the
// delta arm reads. `usage` wins on every field it actually carries; the
// modelUsage roll-up fills whatever is zero.
func claudeTotalUsage(raw claudeEvent) claudeUsage {
	var total claudeUsage
	if raw.Usage != nil {
		total = *raw.Usage
	}
	if missing := total.InputTokens == 0; missing && len(raw.ModelUsage) > 0 {
		for _, cell := range raw.ModelUsage {
			var m struct {
				InputTokens              int64 `json:"inputTokens"`
				OutputTokens             int64 `json:"outputTokens"`
				CacheReadInputTokens     int64 `json:"cacheReadInputTokens"`
				CacheCreationInputTokens int64 `json:"cacheCreationInputTokens"`
			}
			if json.Unmarshal(cell, &m) == nil {
				total.InputTokens += m.InputTokens
				total.OutputTokens += m.OutputTokens
				total.CacheReadInputTokens += m.CacheReadInputTokens
				total.CacheCreationInputTokens += m.CacheCreationInputTokens
			}
		}
	}
	return total
}

// mapClaudeUsage turns one main-conversation result frame into an EvUsage
// DELTA (rolling-total semantics keyed by session id; result-uuid dedupe;
// an all-zero growth emits nothing).
func (ctx *claudeNormCtx) mapClaudeUsage(raw claudeEvent) []state.Event {
	if raw.ParentToolUseID != "" {
		return nil // a subagent's own usage never bills against the office tally twice
	}
	key := raw.SessionID
	if key == "" {
		key = ctx.primaryID
	}
	if key == "" || raw.UUID == "" {
		return nil
	}
	if ctx.usageDone[raw.UUID] {
		return nil
	}
	cur := claudeTotalUsage(raw)
	if cur.zero() && raw.TotalCostUSD == 0 {
		return nil
	}
	ctx.usageDone[raw.UUID] = true
	prev := ctx.usageLast[key]
	d := ocUsageLast{
		in:     cur.InputTokens - prev.in,
		out:    cur.OutputTokens - prev.out,
		cacheR: cur.CacheReadInputTokens - prev.cacheR,
		cacheW: cur.CacheCreationInputTokens - prev.cacheW,
		cost:   raw.TotalCostUSD - prev.cost,
	}
	ctx.usageLast[key] = ocUsageLast{
		in: cur.InputTokens, out: cur.OutputTokens,
		cacheR: cur.CacheReadInputTokens, cacheW: cur.CacheCreationInputTokens,
		cost: raw.TotalCostUSD,
	}
	if d.in == 0 && d.out == 0 && d.cacheR == 0 && d.cacheW == 0 && d.cost == 0 {
		return nil // zero-delta suppressed
	}
	return []state.Event{{
		Kind: state.EvUsage, CallID: raw.UUID,
		TokensIn: d.in, TokensOut: d.out,
		TokensCacheRead: d.cacheR, TokensCacheWrite: d.cacheW,
		CostUSD: d.cost,
	}}
}

// ---------------- the one mapping entry point ----------------

// mapClaudeEvent is the claude mapper: one stdout frame in, state.Events
// out. Unknown types/subtypes and partial inputs are ignored silently;
// malformed payloads never panic.
func mapClaudeEvent(raw claudeEvent, ctx *claudeNormCtx, now int64) []state.Event {
	switch raw.Type {
	case "system":
		switch raw.Subtype {
		case "init":
			ctx.primaryID = raw.SessionID
			ctx.model = raw.Model
			ctx.mcp = nil
			for _, s := range raw.MCPServers {
				ctx.mcp = append(ctx.mcp, state.MCPServer{Name: s.Name, Status: s.Status})
			}
			return []state.Event{{Kind: state.EvStatus,
				Text: fmt.Sprintf("[claude] init model=%s session=%s", raw.Model, raw.SessionID)}}
		case "api_retry":
			return []state.Event{{Kind: state.EvStatus,
				Text: fmt.Sprintf("[claude] retry attempt %d/%d after %dms", raw.Attempt, raw.MaxAttempts, raw.RetryDelayMs)}}
		}
		return nil

	case "stream_event":
		var inner claudeStreamInner
		if len(raw.Event) == 0 || json.Unmarshal(raw.Event, &inner) != nil {
			return nil
		}
		return ctx.mapClaudeStream(inner, raw.ParentToolUseID, now)

	case "assistant":
		return ctx.mapClaudeAssistant(raw, now)

	case "user":
		return ctx.mapClaudeUser(raw, now)

	case "control_request":
		return ctx.mapClaudeControlRequest(raw, now)

	case "control_response", "keep_alive", "control_cancel_request":
		return nil // our own responses / protocol keep-alives

	case "result":
		return ctx.mapClaudeUsage(raw)
	}
	return nil
}

// mapClaudeStream folds one stream_event's inner Anthropic event.
func (ctx *claudeNormCtx) mapClaudeStream(inner claudeStreamInner, parentToolUseID string, now int64) []state.Event {
	switch inner.Type {
	case "message_start":
		ctx.curMsgUUID = inner.Message.ID
		if parentToolUseID == "" {
			ctx.msgStreamOpen = inner.Message.ID
		}
		ctx.blockKind = map[int]string{}
		ctx.blockTool = map[int]string{}
		if ctx.thoughtOpen != "" {
			delete(ctx.thoughtAccum, ctx.thoughtOpen) // a thought abandoned mid-stream
		}
		ctx.thoughtOpen = ""
		return nil

	case "content_block_start":
		kind := inner.ContentBlock.Type
		ctx.blockKind[inner.Index] = kind
		if kind == "tool_use" {
			id := inner.ContentBlock.ID
			ctx.blockTool[inner.Index] = id
			ownerID, ownerName := ctx.ownerFor(parentToolUseID)
			return ctx.claudeToolStart(id, inner.ContentBlock.Name, ownerID, ownerName, nil)
		}
		if kind == "text" && parentToolUseID == "" {
			if ctx.curMsgUUID == "" {
				ctx.curMsgUUID = "stream"
			}
		}
		return nil

	case "content_block_delta":
		switch inner.Delta.Type {
		case "text_delta":
			if parentToolUseID != "" {
				if t, ok := ctx.taskFor(parentToolUseID); ok {
					return ctx.throttledClaudeWorking(t, now, false)
				}
				return nil
			}
			return ctx.claudeTextDelta(ctx.curMsgUUID, inner.Delta.Text, now)
		case "thinking_delta":
			// an empty delta is a keep-alive (the real wire trails one):
			// emitting it would replace-merge the rendered thought to "".
			if inner.Delta.Thinking == "" {
				return nil
			}
			ownerID, ownerName := ctx.ownerFor(parentToolUseID)
			uuid := ctx.curMsgUUID
			if uuid == "" {
				uuid = "stream"
			}
			ctx.thoughtOpen = uuid
			// EvThought carries the GROWING transcript (the events.go
			// contract): the chat reducer replaces-on-merge by CallID, so a
			// bare chunk would shrink the rendered thought to the last delta.
			// The accumulator stays RAW (capThought trims the ends, which
			// would eat paragraph breaks mid-transcript); cap only at emit.
			ctx.thoughtAccum[uuid] += inner.Delta.Thinking
			return []state.Event{{
				Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
				Text: capThought(ctx.thoughtAccum[uuid]), CallID: "think-" + uuid, Done: false,
			}}
		case "input_json_delta":
			id := ctx.blockTool[inner.Index]
			return ctx.claudeToolFold(id, inner.Delta.PartialJSON)
		}
		return nil

	case "content_block_stop", "message_delta", "message_stop":
		if inner.Type == "content_block_stop" || inner.Type == "message_stop" {
			if ctx.thoughtOpen != "" {
				uuid := ctx.thoughtOpen
				ctx.thoughtOpen = ""
				ownerID, ownerName := ctx.ownerFor(parentToolUseID)
				// the close carries the FULL transcript: a bare Done=true
				// would replace-merge the chat entry to empty — on the real
				// wire the partial assistant snapshot lands BEFORE this stop,
				// so the wipe erased the thought the member just saw.
				text := capThought(ctx.thoughtAccum[uuid])
				delete(ctx.thoughtAccum, uuid)
				return []state.Event{{
					Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
					Text: text, CallID: "think-" + uuid, Done: true,
				}}
			}
		}
		if inner.Type == "message_stop" && parentToolUseID == "" {
			// the main message's stream CLOSED — pin the boss bubble from
			// the best-known text. Close the open-marker FIRST so the pin
			// path can never mistake this for a mid-stream refresh.
			ctx.msgStreamOpen = ""
			return ctx.claudeStreamClosePin(ctx.curMsgUUID, now)
		}
		return nil
	}
	return nil
}

// ownerFor resolves the floor actor of a stream/assistant frame: the main
// conversation is the boss; a known subagent run is its employee.
func (ctx *claudeNormCtx) ownerFor(parentToolUseID string) (string, string) {
	if parentToolUseID == "" {
		return "boss", "boss"
	}
	if t, ok := ctx.tasks[parentToolUseID]; ok {
		return t.employeeID, t.employee.Name
	}
	return "task-" + parentToolUseID, "taskgast"
}

// mapClaudeAssistant handles an assistant SNAPSHOT frame (the complete
// message after streaming): text pins the boss bubble, thinking pins
// thoughts, tool_use blocks register running tools (Task tools hire the
// subagent run).
func (ctx *claudeNormCtx) mapClaudeAssistant(raw claudeEvent, now int64) []state.Event {
	uuid := raw.Message.ID
	if uuid == "" {
		uuid = raw.UUID
	}
	parent := raw.ParentToolUseID
	ownerID, ownerName := ctx.ownerFor(parent)
	var evs []state.Event
	var texts []string
	for _, block := range raw.Message.Content {
		switch block.Type {
		case "text":
			if parent == "" && strings.TrimSpace(block.Text) != "" {
				texts = append(texts, strings.TrimSpace(block.Text))
			}
		case "thinking":
			if strings.TrimSpace(block.Thinking) != "" {
				// With --include-partial-messages a PARTIAL snapshot shares the
				// message id and lands MID-stream (before content_block_stop).
				// While the stream's thought for this uuid is still open the
				// snapshot refreshes the transcript WITHOUT closing it — a
				// premature Done=true would collapse the live thought while
				// more thinking is still streaming. Once the stream closed
				// (or never streamed), the snapshot pins the thought done.
				evs = append(evs, state.Event{
					Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
					Text: capThought(block.Thinking), CallID: "think-" + uuid,
					Done: ctx.thoughtOpen != uuid,
				})
			}
		case "tool_use":
			name := block.Name
			if strings.EqualFold(name, "Task") {
				subagentType, _ := block.Input["subagent_type"].(string)
				description, _ := block.Input["description"].(string)
				// a Task call's hire is one-shot per tool_use id
				if _, hired := ctx.tasks[block.ID]; !hired {
					_, _, hires := ctx.claudeIssueEmployee(block.ID, subagentType, description)
					h := ctx.tools[block.ID]
					if h == nil {
						ctx.tools[block.ID] = &claudeToolHold{callID: block.ID, snapshotIn: true}
					}
					evs = append(evs, hires...)
					continue
				}
			}
			if h := ctx.tools[block.ID]; h != nil {
				h.snapshotIn = true
			}
			evs = append(evs, ctx.claudeToolStart(block.ID, name, ownerID, ownerName, block.Input)...)
		}
	}
	if parent == "" {
		evs = append(evs, ctx.claudeAssistantPin(uuid, strings.Join(texts, "\n\n"), now)...)
		return evs
	}
	// a subagent's own assistant beat: the floor sees work, never chat.
	if t, ok := ctx.taskFor(parent); ok {
		evs = append(evs, ctx.throttledClaudeWorking(t, now, false)...)
	}
	return evs
}

// mapClaudeUser handles a user frame: tool_result content closes tool
// holds (a Task result RETURNS the subagent: task done + mail + EvReturned).
// User text blocks are the stdin echo of our own Send — they never surface
// (Send owns the chat-user echo, the same ruling as the opencode backend).
func (ctx *claudeNormCtx) mapClaudeUser(raw claudeEvent, now int64) []state.Event {
	var evs []state.Event
	for _, block := range raw.Message.Content {
		if block.Type != "tool_result" {
			continue
		}
		callID := block.ToolUseID
		if callID == "" {
			continue
		}
		resultText := claudeToolResultText(block.Content)
		if t, ok := ctx.tasks[callID]; ok && !t.returned {
			t.returned = true
			emp := t.employee
			done := state.BoardTask{
				ID: "task-" + callID, Title: emp.Task,
				Status: state.TaskDone, Owner: emp.Name, At: now,
			}
			body := resultText
			if body == "" {
				body = emp.Task
			}
			mail := state.MailItem{
				ID: "mail-" + callID, From: emp.Name, To: "manager",
				At: now, Subject: "return: " + emp.Task,
				Body: sliceMax(body, 240), Kind: state.MailReturn,
			}
			evs = append(evs,
				state.Event{Kind: state.EvTask, Task: done},
				state.Event{Kind: state.EvReturned, EmployeeID: t.employeeID, TaskID: done.ID, Mail: mail},
			)
			delete(ctx.tools, callID)
			continue
		}
		evs = append(evs, ctx.claudeToolFinish(callID, block.IsError, resultText)...)
	}
	return evs
}

// claudeAskUserDialogKind is the CLI's AskUserQuestion tool dialog kind.
// Binary evidence (CLI 2.1.247): the permission-dialog dispatch table maps
// the AskUserQuestion tool to this kind —
//
//	sM({matches:(e)=>e===Z7,dialog:zor,build:Esr})   // Z7 = AskUserQuestion
//	zor=el({kind:"permission_ask_user_question",
//	        payload: …("requestId"in e)&&("toolName"in e)&&
//	                ("permissionResult"in e)&&("questions"in e)…,
//	        result: …("behavior"in e)…, default:{behavior:"cancelled"}})
//
// and Esr builds the wire payload as the base permission descriptor
// (requestId/toolName/input/description/permissionResult/userFacingName/…)
// PLUS questions (the tool input's questions verbatim) + metadataSource.
const claudeAskUserDialogKind = "permission_ask_user_question"

// ---------------- the rendered dialog-kind table ----------------
//
// claudeRenderedDialogKinds is the EXACT set of request_user_dialog
// dialog_kind values the office can render (CLI 2.1.247 binary evidence —
// every kind's registration literal was extracted from the installed
// binary; see claude_dialog_kinds_test.go for the verbatim literals). The
// protocol declare-gate (binary zod .describe, verbatim):
//
//	"A kind is only sent in sessions where some attached client declared
//	 it in initialize.supportedDialogKinds (declare exactly the kinds you
//	 can render); … A host that receives a kind it did not declare must
//	 not answer it … An unanswered dialog is cancelled by the CLI after
//	 its dialog deadline."
//
// — so this ONE table drives BOTH sides: claude.go's initialize writer
// declares exactly these kinds, and mapClaudeControlRequest renders
// exactly these kinds (anything else parks unanswered: NO EvQuestion, NO
// hold — the CLI's own deadline settles it as the kind's default).
//
// NOT declared/rendered (fail closed, parked):
//   - computer_use_approval: fully opaque payload (typeof e==="object"),
//     result {granted:[],denied:[],flags} — nothing renderable.
//   - local_jsx: payload is a process-local nodeId into the CLI's own
//     LocalJsxRegistryContext ("A local_jsx dialog cannot render outside
//     of a LocalJsxRegistryContext provider"), result z.null() — there is
//     no answer the office could meaningfully send.
//   - mcp_url_elicitation: requires FREE-TEXT URL input (mode "url") the
//     office's question modal cannot represent; answering blind would
//     fabricate a URL. (MCP elicitations also ride the dedicated
//     "elicitation" control subtype, not this table.)
var claudeRenderedDialogKinds = []string{
	// F0 — the AskUserQuestion tool dialog (result: permission decision
	// object carrying "behavior").
	claudeAskUserDialogKind,
	// F1 — permission gates (result: object with "behavior":
	// allow/deny/cancelled; default {behavior:"cancelled"}).
	"permission_prompt",
	"permission_bash",              // payload adds command, classifierState
	"permission_browser",           // payload adds verbPhrase, chrome{host,url}
	"permission_enter_plan_mode",   //
	"permission_exit_plan_mode_v2", // payload adds plan (markdown)
	"permission_file",              // payload adds filePath, operationType
	"permission_monitor",           // payload adds intervalMs
	"permission_powershell",        // payload adds command
	"permission_skill",             // payload adds skill
	"permission_webfetch",          // payload adds hostname
	"permission_workflow",          // payload adds script
	// F2 — enum consent kinds (result: a bare enum STRING from the kind's
	// own vocabulary; the default is the kind's dismissal value).
	"cloud_sync_consent",           // [sync device_tools not_now], default not_now
	"fable_overage_consent_prompt", // [consent switch_default cancelled]
	"refusal_fallback_prompt",      // [retry_fallback edit_prompt cancelled]
	"chrome_install_upsell",        // [install not_now dont_ask_again cancelled]
	"chrome_install_setup",         // STREAMING (phase updates re-emit): [continue keep_waiting skip cancelled]
	"auto_mode_setup_review",       // [accept decline cancelled]
	"resume_return",                // [compact continue dismiss never cancelled]
	"managed_settings_security",    // [approved rejected deferred_no_consent_surface]
	"auto_default_nudge",           // [accepted declined cancelled]
	"cost_threshold",               // [acknowledged cancelled]
	"ide_onboarding",               // [dismissed cancelled]
	"it2_setup",                    // [installed use-tmux cancelled]
	// F3 — structured results.
	"goal_proposal",           // result {approved:bool, explicit?:bool}
	"auto_mode_flagged_allow", // result {toRemove:[...]} | "cancelled"
	"sandbox_network_access",  // result {allow:bool, persistToSettings:bool} | "cancelled"
	"peer_inbound_approval",   // result {behavior:"approve"|"deny"|"cancelled"}
}

// claudeRenderedDialogKindSet is the park gate: a kind outside the
// declared set is never rendered (and therefore never answered).
var claudeRenderedDialogKindSet = func() map[string]bool {
	m := make(map[string]bool, len(claudeRenderedDialogKinds))
	for _, k := range claudeRenderedDialogKinds {
		m[k] = true
	}
	return m
}()

// claudeDialogFamily selects the result builder an answer takes.
type claudeDialogFamily int

const (
	// dialogFamilyAUQ — the AskUserQuestion dialog: the CLI-native
	// permission decision {behavior:"allow",updatedInput:{questions,
	// answers}} (claudeAskUserResultJSON).
	dialogFamilyAUQ claudeDialogFamily = iota
	// dialogFamilyPermission — an F1 permission gate: Allow once/Allow
	// always -> {behavior:"allow"(,updatedInput:<payload.input>)},
	// Reject -> {behavior:"deny",message:"Denied by the boss in
	// theboringoffice"}.
	dialogFamilyPermission
	// dialogFamilyLabelResult — a single-select page whose option labels
	// map 1:1 onto prebuilt result bytes (F2 enum kinds: result is a bare
	// JSON string; F3 goal_proposal/sandbox_network_access/
	// peer_inbound_approval: result is a fixed object per option).
	dialogFamilyLabelResult
	// dialogFamilyFlaggedAllow — auto_mode_flagged_allow's multi-select:
	// the picked rule labels become {toRemove:[...]} ("Remove them all"
	// expands to every flagged rule; an empty pick leaves them all).
	dialogFamilyFlaggedAllow
)

// claudeDialogMeta stashes what a PENDING dialog's answer needs later —
// the dialog twin of permMeta (the backend-side alternative to threading
// per-kind data through state.Event, whose surface stays unchanged).
type claudeDialogMeta struct {
	kind   string
	family claudeDialogFamily
	// input — F1 only: the payload's tool input, re-emitted as
	// updatedInput on an allow answer (the CLI's own "yes" builder:
	// {behavior:"allow", updatedInput:t.input}). nil when the payload
	// carried no input key.
	input json.RawMessage
	// resultByLabel — dialogFamilyLabelResult only: option label -> the
	// EXACT result bytes that label settles (a bare enum string for F2, a
	// fixed object for the F3 label kinds). A picked label outside the map
	// (the modal's free-text row) fails closed: no write, the dialog
	// stays parked.
	resultByLabel map[string]json.RawMessage
	// flagged — dialogFamilyFlaggedAllow only: every flagged rule string
	// (the "Remove them all" expansion set).
	flagged []string
}

// claudeDialogQuestion is ONE asked question of the AskUserQuestion dialog
// payload — the CLI 2.1.247 zod literal (the AskUserQuestion inputSchema's
// questions element):
//
//	{question, header, options: [{label, description, preview?}] (min 2,
//	 max 4), multiSelect (default false)}
type claudeDialogQuestion struct {
	Question    string `json:"question"`
	Header      string `json:"header"`
	MultiSelect bool   `json:"multiSelect"`
	Options     []struct {
		Label       string `json:"label"`
		Description string `json:"description"`
		Preview     string `json:"preview"` // optional markdown (gss: preview:I().optional())
	} `json:"options"`
}

// claudeAskUserDialogPayload is the decoded payload of dialog_kind
// permission_ask_user_question. Only questions + the analytics context
// are read; every other field of the base descriptor is tolerated
// (encoding/json's open-ended rule). Metadata/metadataSource are kept
// RAW (json.RawMessage — the AskUserQuestion tool input tolerates a
// metadata object, "not displayed to the user") so the dialog answer's
// updatedInput re-emits them byte-verbatim, exactly as the CLI's own
// submit builder spreads the original input.
type claudeAskUserDialogPayload struct {
	Questions      []claudeDialogQuestion `json:"questions"`
	MetadataSource json.RawMessage        `json:"metadataSource"`
	Metadata       json.RawMessage        `json:"metadata"`
}

// mapClaudeControlRequest: can_use_tool -> EvPermission pending;
// request_user_dialog -> EvQuestion pending. Both ride a permHold so the
// backend's AnswerPermission/AnswerQuestion/RejectQuestion writers can
// resolve the SAME request_id.
func (ctx *claudeNormCtx) mapClaudeControlRequest(raw claudeEvent, now int64) []state.Event {
	if raw.RequestID == "" {
		return nil
	}
	req := raw.Request
	switch req.Subtype {
	case "can_use_tool":
		ownerID, ownerName := ctx.ownerFor(raw.ParentToolUseID)
		if raw.ParentToolUseID == "" && req.ToolUseID != "" {
			// the control frame may lack parent_tool_use_id: the asking
			// tool_use id still names its subagent run when one owns it.
			if t, ok := ctx.tasks[req.ToolUseID]; ok {
				ownerID, ownerName = t.employeeID, t.employee.Name
			}
		}
		// The modal summary rides the CLI's own description line first
		// (the real wire carries no input_preview), falling back to a
		// summary of the tool input.
		summary := strings.TrimSpace(req.Description)
		if summary == "" {
			summary = claudeToolSummary(req.Input)
		}
		toolName := claudeToolName(req.ToolName)
		// The REAL 2.1.247 can_use_tool envelope carries NO session_id
		// (live capture, Bedrock) — fill from the pinned primary session
		// id (the same source the system/init arm pins), so the office's
		// EvPermission never carries SessionID="" in production. With no
		// primary pinned yet it stays empty: fill, never invent.
		sessionID := raw.SessionID
		if sessionID == "" {
			sessionID = ctx.primaryID
		}
		ctx.pendingPerms[raw.RequestID] = permHold{
			SessionID: sessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Title: toolName, Summary: summary,
		}
		ctx.permMeta[raw.RequestID] = claudePermMeta{
			Suggestions: req.PermissionSuggestions, Input: req.Input,
		}
		evs := []state.Event{{
			Kind: state.EvPermission, PermissionID: raw.RequestID,
			SessionID: sessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			ToolName: toolName, ToolSummary: summary, ToolState: "pending",
		}}
		if ownerID != "boss" {
			evs = append(evs, state.Event{
				Kind: state.EvBlocked, EmployeeID: ownerID,
				Text: shortTitle("permission: "+toolName+" "+summary, 60),
			})
		}
		return evs
	case "request_user_dialog":
		// The wire carries dialog_kind + payload — NEVER the flat
		// question/options fields the old parser read. The park rule
		// (binary schema-doc, verbatim on claudeRenderedDialogKinds): a
		// host receiving a kind it did not declare must NOT answer it (a
		// {behavior:"cancelled"} reply is a REAL settlement — the user
		// dismissing the dialog — and an error-subtype reply is
		// discarded), and the CLI cancels unanswered dialogs at their
		// deadline — so a kind the office cannot render surfaces NO
		// EvQuestion and NO hold. Only kinds in claudeRenderedDialogKinds
		// (the initialize.declared set) render.
		if !claudeRenderedDialogKindSet[req.DialogKind] {
			return nil
		}
		render, ok := claudeRenderDialog(req.DialogKind, req.Payload)
		if !ok || len(render.items) == 0 {
			// undecodable payload / a known kind with NO renderable page:
			// park, same as unknown kinds — an empty popover would be a
			// settle path for nothing rendered.
			return nil
		}
		ownerID, ownerName := ctx.ownerFor(raw.ParentToolUseID)
		ctx.pendingQuestions[raw.RequestID] = permHold{
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Title: "question", Summary: render.summary,
		}
		ctx.dialogMeta[raw.RequestID] = render.meta
		return []state.Event{{
			Kind: state.EvQuestion, QuestionID: raw.RequestID,
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Text: shortTitle(render.text, 240), ToolSummary: shortTitle(render.summary, 120),
			ToolState: "pending", Questions: render.items,
		}}
	}
	return nil
}

// claudeDialogRender is one decoded dialog: the modal pages, the flattened
// text/summary the EvQuestion carries, and the per-kind answer stash.
type claudeDialogRender struct {
	items   []state.QuestionItem
	text    string // flattened page texts (EvQuestion.Text)
	summary string // option labels joined " | " (EvQuestion.ToolSummary)
	meta    claudeDialogMeta
}

// claudeRenderDialog decodes one DECLARED dialog kind's payload into the
// render. ok=false parks the dialog (undecodable payload / no renderable
// page) — the office never answers what it did not render.
func claudeRenderDialog(kind string, payload json.RawMessage) (claudeDialogRender, bool) {
	switch kind {
	case claudeAskUserDialogKind:
		return claudeRenderAskUserDialog(payload)
	// F1 — the permission gates.
	case "permission_prompt", "permission_bash", "permission_browser",
		"permission_enter_plan_mode", "permission_exit_plan_mode_v2",
		"permission_file", "permission_monitor", "permission_powershell",
		"permission_skill", "permission_webfetch", "permission_workflow":
		return claudeRenderPermissionDialog(kind, payload)
	// F2/F3 label-select kinds (single page, labels map 1:1 to results).
	case "cloud_sync_consent":
		return claudeRenderCloudSyncConsent(payload)
	case "fable_overage_consent_prompt":
		return claudeRenderFableOverage(payload)
	case "refusal_fallback_prompt":
		return claudeRenderRefusalFallback(payload)
	case "chrome_install_upsell":
		return claudeRenderChromeInstallUpsell(payload)
	case "chrome_install_setup":
		return claudeRenderChromeInstallSetup(payload)
	case "auto_mode_setup_review":
		return claudeRenderAutoModeSetupReview(payload)
	case "resume_return":
		return claudeRenderResumeReturn(payload)
	case "managed_settings_security":
		return claudeRenderManagedSettingsSecurity(payload)
	case "auto_default_nudge":
		return claudeRenderAutoDefaultNudge(payload)
	case "cost_threshold":
		return claudeRenderCostThreshold(payload)
	case "ide_onboarding":
		return claudeRenderIdeOnboarding(payload)
	case "it2_setup":
		return claudeRenderIt2Setup(payload)
	// F3 — structured results.
	case "goal_proposal":
		return claudeRenderGoalProposal(payload)
	case "auto_mode_flagged_allow":
		return claudeRenderAutoModeFlaggedAllow(payload)
	case "sandbox_network_access":
		return claudeRenderSandboxNetworkAccess(payload)
	case "peer_inbound_approval":
		return claudeRenderPeerInboundApproval(payload)
	}
	return claudeDialogRender{}, false
}

// claudeDialogPage is the one-page render helper every non-AUQ kind uses:
// the modal's Question carries the dialog title (+ the body on a following
// paragraph), Header rides nothing (the AUQ chip stays AUQ-only), Options
// are the selectable answers. text/summary flatten the same way the AUQ
// arm's do.
func claudeDialogPage(title, body string, opts []state.QuestionOption, multiple bool) claudeDialogRender {
	question := strings.TrimSpace(title)
	if b := strings.TrimSpace(body); b != "" {
		question += "\n\n" + b
	}
	item := state.QuestionItem{Question: question, Options: opts, Multiple: multiple}
	var options []string
	for _, o := range opts {
		options = append(options, o.Label)
	}
	summary := strings.Join(options, " | ")
	if summary == "" {
		summary = "free-form answer"
	}
	return claudeDialogRender{
		items:   []state.QuestionItem{item},
		text:    question,
		summary: summary,
	}
}

// withLabelResults fills the render's meta for a dialogFamilyLabelResult
// kind: pairs of (option label, exact result bytes).
func (r claudeDialogRender) withLabelResults(kind string, pairs ...any) claudeDialogRender {
	r.meta = claudeDialogMeta{
		kind:          kind,
		family:        dialogFamilyLabelResult,
		resultByLabel: make(map[string]json.RawMessage, len(pairs)/2),
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		label, _ := pairs[i].(string)
		raw, _ := pairs[i+1].(json.RawMessage)
		r.meta.resultByLabel[label] = raw
	}
	return r
}

// enumResult marshals one F2 enum answer string (the result is a BARE JSON
// string on the wire — the kind's registration: result:ae(()=>un([...]))).
func enumResult(v string) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

// ---------------- F0: permission_ask_user_question ----------------

// claudeRenderAskUserDialog decodes the AskUserQuestion dialog payload
// into the office's question pages (the pre-existing path, unchanged):
// questions + the payload-level analytics context (metadataSource/
// metadata ride EVERY page, re-emitted in the answer's updatedInput).
func claudeRenderAskUserDialog(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload claudeAskUserDialogPayload
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	var texts, options []string
	var items []state.QuestionItem
	for _, q := range payload.Questions {
		s := strings.TrimSpace(q.Question)
		if s == "" {
			continue
		}
		texts = append(texts, s)
		item := state.QuestionItem{
			Question: s,
			Header:   strings.TrimSpace(q.Header),
			Multiple: q.MultiSelect,
			// The payload-level analytics context rides EVERY page
			// (the questionStash keys pages, not payloads; the
			// result builder reads the first carrier).
			Meta:       payload.Metadata,
			MetaSource: payload.MetadataSource,
		}
		for _, opt := range q.Options {
			if l := strings.TrimSpace(opt.Label); l != "" {
				item.Options = append(item.Options, state.QuestionOption{
					Label:       l,
					Description: strings.TrimSpace(opt.Description),
					Preview:     strings.TrimSpace(opt.Preview),
				})
				options = append(options, l)
			}
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return claudeDialogRender{}, false
	}
	summary := strings.Join(options, " | ")
	if summary == "" {
		summary = "free-form answer"
	}
	return claudeDialogRender{
		items:   items,
		text:    strings.Join(texts, " "),
		summary: summary,
		meta:    claudeDialogMeta{kind: claudeAskUserDialogKind, family: dialogFamilyAUQ},
	}, true
}

// ---------------- F1: the permission gates ----------------

// claudePermDialogPayload is the F1 dialog payload: the base permission
// descriptor (binary builder Ky: requestId/toolName/input/description/
// permissionResult/userFacingName/…/showAlwaysAllow) PLUS the kind's own
// subject fields. Only the fields the office renders are decoded.
type claudePermDialogPayload struct {
	RequestID   string          `json:"requestId"`
	ToolName    string          `json:"toolName"`
	Input       json.RawMessage `json:"input"`
	Description string          `json:"description"`
	// showAlwaysAllow is the CLI's own "the always-row exists" flag
	// (Ky: permissionResult suppressions, org caps and tool rules already
	// folded in). false => the office drops its "Allow always" row too.
	ShowAlwaysAllow bool `json:"showAlwaysAllow"`
	// kind subjects:
	Command         string `json:"command"`         // permission_bash, permission_powershell
	ClassifierState string `json:"classifierState"` // permission_bash
	VerbPhrase      string `json:"verbPhrase"`      // permission_browser
	Chrome          *struct {
		Host string `json:"host"`
		URL  string `json:"url"`
	} `json:"chrome"` // permission_browser
	Plan          string `json:"plan"`          // permission_exit_plan_mode_v2 (markdown)
	FilePath      string `json:"filePath"`      // permission_file
	OperationType string `json:"operationType"` // permission_file
	IntervalMs    int64  `json:"intervalMs"`    // permission_monitor
	Skill         string `json:"skill"`         // permission_skill
	Hostname      string `json:"hostname"`      // permission_webfetch
	Script        string `json:"script"`        // permission_workflow
}

// claudeDialogAllowLabels are the F1 option labels, mirroring the office's
// own permission modal rows (Allow once / Allow always / Reject).
const (
	claudeDialogAllowOnce   = "Allow once"
	claudeDialogAllowAlways = "Allow always"
	claudeDialogReject      = "Reject"
)

// claudeRenderPermissionDialog renders an F1 permission gate as ONE
// question-modal page: the CLI's notification title + the kind's subject
// as the body, and the office permission modal's own three rows. The
// result bytes are built at answer time (dialogFamilyPermission) — Allow
// once/always -> {behavior:"allow"} (+updatedInput re-emitting the
// payload's tool input, the CLI's own "yes" builder), Reject ->
// {behavior:"deny",message:"Denied by the boss in theboringoffice"}.
//
// "Allow always" can NOT attach permissionUpdates: the F1 payload carries
// NO permission_suggestions (the CLI computes those locally from the live
// tool at render time — e.g. tHe/IOe build addRules from tool+ruleContent)
// — so the always leg is a plain allow, and the office's standing-grant
// semantics stay on the can_use_tool path where suggestions ride the wire.
func claudeRenderPermissionDialog(kind string, rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload claudePermDialogPayload
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	// Modal titles: the CLI's notification titles verbatim (binary title
	// map: Kd="Claude needs your permission" x9 kinds; enter/exit plan
	// mode carry their own; permission_browser's dialog component titles
	// itself "Claude in Chrome wants to <verbPhrase>[ on <host>]").
	var title, subject string
	switch kind {
	case "permission_enter_plan_mode":
		title = "Claude Code wants to enter plan mode"
	case "permission_exit_plan_mode_v2":
		title = "Claude Code needs your approval for the plan"
		subject = payload.Plan
	case "permission_browser":
		title = "Claude in Chrome wants to " + strings.TrimSpace(payload.VerbPhrase)
		if payload.Chrome != nil && strings.TrimSpace(payload.Chrome.Host) != "" {
			title += " on " + strings.TrimSpace(payload.Chrome.Host)
		}
		subject = payload.Description
	case "permission_bash", "permission_powershell":
		title = "Claude needs your permission"
		subject = payload.Command
		if kind == "permission_bash" && strings.TrimSpace(payload.ClassifierState) != "" {
			subject += "\nclassifierState: " + strings.TrimSpace(payload.ClassifierState)
		}
	case "permission_file":
		title = "Claude needs your permission"
		subject = strings.TrimSpace(strings.TrimSpace(payload.OperationType) + " " + strings.TrimSpace(payload.FilePath))
	case "permission_monitor":
		title = "Claude needs your permission"
		subject = fmt.Sprintf("intervalMs: %d", payload.IntervalMs)
	case "permission_skill":
		title = "Claude needs your permission"
		subject = payload.Skill
	case "permission_webfetch":
		title = "Claude needs your permission"
		subject = payload.Hostname
	case "permission_workflow":
		title = "Claude needs your permission"
		subject = payload.Script
	default: // permission_prompt
		title = "Claude needs your permission"
		subject = payload.Description
	}
	opts := []state.QuestionOption{{Label: claudeDialogAllowOnce}}
	if payload.ShowAlwaysAllow {
		opts = append(opts, state.QuestionOption{Label: claudeDialogAllowAlways})
	}
	opts = append(opts, state.QuestionOption{Label: claudeDialogReject})
	render := claudeDialogPage(title, subject, opts, false)
	if tn := strings.TrimSpace(payload.ToolName); tn != "" {
		render.items[0].Header = tn // the asking tool's chip (the AUQ header seam)
	}
	render.meta = claudeDialogMeta{
		kind:   kind,
		family: dialogFamilyPermission,
		input:  payload.Input,
	}
	return render, true
}

// ---------------- F2: the enum consent kinds ----------------

// claudeRenderCloudSyncConsent — payload {folder,launchFolder?,title,
// body,detail?,fileCount?,totalBytes?} -> enum [sync device_tools
// not_now]. The CLI ships the dialog copy IN the payload (title/body/
// detail ride the wire); the option labels are NOT recoverable from the
// binary (the component builds them inline), so the office's labels are
// derived from the enum values.
func claudeRenderCloudSyncConsent(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Folder       string `json:"folder"`
		LaunchFolder string `json:"launchFolder"`
		Title        string `json:"title"`
		Body         string `json:"body"`
		Detail       string `json:"detail"`
		FileCount    *int64 `json:"fileCount"`
		TotalBytes   *int64 `json:"totalBytes"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = "Claude wants to sync a folder"
	}
	var body strings.Builder
	body.WriteString(strings.TrimSpace(payload.Body))
	if d := strings.TrimSpace(payload.Detail); d != "" {
		if body.Len() > 0 {
			body.WriteString("\n\n")
		}
		body.WriteString(d)
	}
	if payload.FileCount != nil {
		if body.Len() > 0 {
			body.WriteString("\n\n")
		}
		fmt.Fprintf(&body, "%d files", *payload.FileCount)
		if payload.TotalBytes != nil {
			fmt.Fprintf(&body, ", %d bytes", *payload.TotalBytes)
		}
		body.WriteString(" in " + payload.Folder) // the consent's subject folder
	}
	render := claudeDialogPage(title, body.String(), []state.QuestionOption{
		{Label: "Sync this folder"},
		{Label: "Use device tools"},
		{Label: "Not now"},
	}, false)
	return render.withLabelResults("cloud_sync_consent",
		"Sync this folder", enumResult("sync"),
		"Use device tools", enumResult("device_tools"),
		"Not now", enumResult("not_now"),
	), true
}

// claudeRenderFableOverage — payload {overagesEnabled,balanceCents?,
// currency?} -> enum [consent switch_default cancelled]. Title is the
// CLI's notification title "Session paused" (binary title map); the
// labels derive from the CLI's own notification summary ("choose:
// continue Fable 5 on usage credits or switch models").
func claudeRenderFableOverage(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		OveragesEnabled bool   `json:"overagesEnabled"`
		BalanceCents    *int64 `json:"balanceCents"`
		Currency        string `json:"currency"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	body := fmt.Sprintf("overagesEnabled: %t", payload.OveragesEnabled)
	if payload.BalanceCents != nil {
		currency := payload.Currency
		if currency == "" {
			currency = "USD"
		}
		body += fmt.Sprintf("\nbalance: %d cents (%s)", *payload.BalanceCents, currency)
	}
	render := claudeDialogPage("Session paused", body, []state.QuestionOption{
		{Label: "Continue Fable 5 on usage credits"},
		{Label: "Switch to the default model"},
	}, false)
	return render.withLabelResults("fable_overage_consent_prompt",
		"Continue Fable 5 on usage credits", enumResult("consent"),
		"Switch to the default model", enumResult("switch_default"),
	), true
}

// claudeRenderRefusalFallback — payload {originalModel,fallbackModel,
// apiRefusalCategory?,guidanceText?,retractedMessageUuids?} -> enum
// [retry_fallback edit_prompt cancelled]. Title "Session paused" (binary
// title map); the option labels follow the CLI's own bHe builder ("Switch
// to <fallback>" when the model is known, else the static fallback;
// "Edit prompt and retry with <original>"/"Edit prompt and retry").
func claudeRenderRefusalFallback(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		OriginalModel      string `json:"originalModel"`
		FallbackModel      string `json:"fallbackModel"`
		APIRefusalCategory string `json:"apiRefusalCategory"`
		GuidanceText       string `json:"guidanceText"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	var body strings.Builder
	if g := strings.TrimSpace(payload.GuidanceText); g != "" {
		body.WriteString(g)
	}
	if body.Len() > 0 {
		body.WriteString("\n\n")
	}
	body.WriteString("Refused model: " + payload.OriginalModel + "\nfallback: " + payload.FallbackModel)
	retryLabel := "Switch to the fallback model"
	if m := strings.TrimSpace(payload.FallbackModel); m != "" {
		retryLabel = "Switch to " + m
	}
	editLabel := "Edit prompt and retry"
	if m := strings.TrimSpace(payload.OriginalModel); m != "" {
		editLabel = "Edit prompt and retry with " + m
	}
	render := claudeDialogPage("Session paused", body.String(), []state.QuestionOption{
		{Label: retryLabel},
		{Label: editLabel},
	}, false)
	return render.withLabelResults("refusal_fallback_prompt",
		retryLabel, enumResult("retry_fallback"),
		editLabel, enumResult("edit_prompt"),
	), true
}

// claudeRenderChromeInstallUpsell — payload {} -> enum [install not_now
// dont_ask_again cancelled]. Title/body/options are the CLI's own
// component copy verbatim (binary: title "Claude wants to use your
// browser"; options $Oe with descriptions).
func claudeRenderChromeInstallUpsell(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	if len(rawPayload) > 0 {
		var probe map[string]any
		if err := json.Unmarshal(rawPayload, &probe); err != nil {
			return claudeDialogRender{}, false
		}
	}
	render := claudeDialogPage("Claude wants to use your browser",
		"This task could use your Chrome browser. The Claude in Chrome extension lets Claude navigate sites, click buttons, and fill forms in your existing session.",
		[]state.QuestionOption{
			{Label: "Install extension", Description: "Opens the install page in Chrome"},
			{Label: "Not now", Description: "Continue without browser tools"},
			{Label: "Don't ask again", Description: "Revisit anytime with /chrome"},
		}, false)
	return render.withLabelResults("chrome_install_upsell",
		"Install extension", enumResult("install"),
		"Not now", enumResult("not_now"),
		"Don't ask again", enumResult("dont_ask_again"),
	), true
}

// claudeRenderChromeInstallSetup — payload {phase, installPageOpened} ->
// enum [continue keep_waiting skip cancelled]. STREAMING kind: payload
// updates re-arrive on the SAME request_id as the phase transitions
// (waiting_install -> connecting -> stalled -> connected/failed); every
// update re-emits the EvQuestion (replace-merge by QuestionID — the app's
// fold-in dedupes repeats, so the modal keeps the FIRST phase's pages,
// but the stash overwrites, so an answer always maps against the LATEST
// phase). The option set per phase mirrors the CLI's own component:
// connected offers Continue, stalled offers Keep waiting, every phase
// offers the skip row; other phases park on the skip row only.
func claudeRenderChromeInstallSetup(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Phase             string `json:"phase"`
		InstallPageOpened bool   `json:"installPageOpened"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	body := "Finish setup in Chrome. This screen follows along and updates on its own as each step completes."
	if payload.Phase != "" {
		body = "phase: " + payload.Phase + "\n\n" + body
	}
	skip := state.QuestionOption{Label: "Continue without browser tools", Description: "Finish setup later with /chrome"}
	pairs := []any{"Continue without browser tools", enumResult("skip")}
	var opts []state.QuestionOption
	switch payload.Phase {
	case "connected":
		opts = append(opts, state.QuestionOption{Label: "Continue with browser tools", Description: "Claude picks the task back up in your browser"})
		pairs = append(pairs, "Continue with browser tools", enumResult("continue"))
	case "stalled":
		opts = append(opts, state.QuestionOption{Label: "Keep waiting", Description: "Setup keeps checking for the connection"})
		pairs = append(pairs, "Keep waiting", enumResult("keep_waiting"))
	}
	opts = append(opts, skip)
	render := claudeDialogPage("Setting up Claude in Chrome", body, opts, false)
	return render.withLabelResults("chrome_install_setup", pairs...), true
}

// claudeRenderAutoModeSetupReview — payload {environment[],allow[],
// soft_deny[],hard_deny[],remove_from_permissions_allow[],notes[],mode}
// -> enum [accept decline cancelled]. Title/options are the CLI's own
// copy verbatim (binary: "Auto-mode setup proposal is ready for review",
// [{accept,"Looks good — save it"},{decline,"Discard and exit"}]); the
// body summarizes the proposal lists.
func claudeRenderAutoModeSetupReview(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Environment                []string `json:"environment"`
		Allow                      []string `json:"allow"`
		SoftDeny                   []string `json:"soft_deny"`
		HardDeny                   []string `json:"hard_deny"`
		RemoveFromPermissionsAllow []string `json:"remove_from_permissions_allow"`
		Notes                      []string `json:"notes"`
		Mode                       string   `json:"mode"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	var body strings.Builder
	fmt.Fprintf(&body, "mode: %s", payload.Mode)
	fmt.Fprintf(&body, "\nallow: %d · soft_deny: %d · hard_deny: %d · remove_from_permissions_allow: %d",
		len(payload.Allow), len(payload.SoftDeny), len(payload.HardDeny), len(payload.RemoveFromPermissionsAllow))
	for _, n := range payload.Notes {
		if s := strings.TrimSpace(n); s != "" {
			body.WriteString("\n" + s)
		}
	}
	render := claudeDialogPage("Auto-mode setup proposal is ready for review", body.String(), []state.QuestionOption{
		{Label: "Looks good — save it"},
		{Label: "Discard and exit"},
	}, false)
	return render.withLabelResults("auto_mode_setup_review",
		"Looks good — save it", enumResult("accept"),
		"Discard and exit", enumResult("decline"),
	), true
}

// claudeRenderResumeReturn — payload {sessionAgeMinutes,estimatedTokens}
// -> enum [compact continue dismiss never cancelled]. The CLI composes
// the title from the payload ("This session is <age> old and <tokens>
// tokens.") and offers compact/continue/never (esc -> "dismiss" — the
// office's own dismiss path settles envelope-cancelled instead, which the
// CLI substitutes with the kind default "cancelled"; both are in-vocab).
func claudeRenderResumeReturn(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		SessionAgeMinutes int64 `json:"sessionAgeMinutes"`
		EstimatedTokens   int64 `json:"estimatedTokens"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	render := claudeDialogPage(
		fmt.Sprintf("This session is %d minutes old and %d tokens.", payload.SessionAgeMinutes, payload.EstimatedTokens),
		"Resuming the full session will consume a substantial portion of your usage limits. We recommend resuming from a summary.",
		[]state.QuestionOption{
			{Label: "Resume from summary (recommended)"},
			{Label: "Resume full session as-is"},
			{Label: "Don't ask me again"},
		}, false)
	return render.withLabelResults("resume_return",
		"Resume from summary (recommended)", enumResult("compact"),
		"Resume full session as-is", enumResult("continue"),
		"Don't ask me again", enumResult("never"),
	), true
}

// claudeRenderManagedSettingsSecurity — payload {settings} -> enum
// [approved rejected deferred_no_consent_surface]. The settings blob is
// an opaque object on the wire (ym(typeof e==="object")); the office
// shows it raw (capped) for the boss's review.
func claudeRenderManagedSettingsSecurity(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Settings json.RawMessage `json:"settings"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	body := "Review the managed settings change before it applies."
	if len(payload.Settings) > 0 {
		body += "\n\n" + shortTitle(string(payload.Settings), 800)
	}
	render := claudeDialogPage("A managed settings change needs your approval", body, []state.QuestionOption{
		{Label: "Approve"},
		{Label: "Reject"},
	}, false)
	return render.withLabelResults("managed_settings_security",
		"Approve", enumResult("approved"),
		"Reject", enumResult("rejected"),
	), true
}

// claudeRenderAutoDefaultNudge — payload {currentMode} -> enum [accepted
// declined cancelled]. Title/body/labels are the CLI's component copy
// verbatim (binary: title "Make auto mode your default permission
// mode?", options "Yes, set auto mode as my default permission mode" /
// "No, keep <currentMode>").
func claudeRenderAutoDefaultNudge(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		CurrentMode string `json:"currentMode"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	noLabel := "No, keep " + payload.CurrentMode
	render := claudeDialogPage("Make auto mode your default permission mode?",
		"Auto mode lets Claude handle permission prompts automatically. Claude checks each tool call for risky actions and prompt injection before executing, runs the ones it assesses as lower-risk, and blocks the rest.",
		[]state.QuestionOption{
			{Label: "Yes, set auto mode as my default permission mode"},
			{Label: noLabel},
		}, false)
	return render.withLabelResults("auto_default_nudge",
		"Yes, set auto mode as my default permission mode", enumResult("accepted"),
		noLabel, enumResult("declined"),
	), true
}

// claudeRenderCostThreshold — payload {} -> enum [acknowledged
// cancelled]. Title/body/label are the CLI's component copy verbatim
// (binary: title "You've spent $5 on the Anthropic API this session.",
// the costs-doc link, option {value:"ok", label:"Got it, thanks!"} ->
// mapped to "acknowledged" by the CLI's own wrapper).
func claudeRenderCostThreshold(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	if len(rawPayload) > 0 {
		var probe map[string]any
		if err := json.Unmarshal(rawPayload, &probe); err != nil {
			return claudeDialogRender{}, false
		}
	}
	render := claudeDialogPage("You've spent $5 on the Anthropic API this session.",
		"Learn more about how to monitor your spending: https://code.claude.com/docs/en/costs",
		[]state.QuestionOption{{Label: "Got it, thanks!"}}, false)
	return render.withLabelResults("cost_threshold",
		"Got it, thanks!", enumResult("acknowledged"),
	), true
}

// claudeRenderIdeOnboarding — payload {installationStatus?} -> enum
// [dismissed cancelled]. The CLI's own dialog settles EVERY confirmation
// as "dismissed" (binary wrapper: onDone:()=>t("dismissed")); the office
// offers the single ack row.
func claudeRenderIdeOnboarding(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		InstallationStatus json.RawMessage `json:"installationStatus"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	render := claudeDialogPage("Claude Code is available as an IDE extension",
		"Claude Code is available as a CLI in the terminal, desktop app (Mac/Windows), web app (claude.ai/code), and IDE extensions (VS Code, JetBrains).",
		[]state.QuestionOption{{Label: "Got it"}}, false)
	return render.withLabelResults("ide_onboarding",
		"Got it", enumResult("dismissed"),
	), true
}

// claudeRenderIt2Setup — payload {tmuxAvailable} -> enum [installed
// use-tmux cancelled]. Body/options are the CLI's component copy verbatim
// (binary: "To use native iTerm2 split panes for teammates, you need the
// it2 CLI tool."; options "Install it2 now"/"Use tmux instead"). The
// "Use tmux instead" row only exists when tmuxAvailable (the CLI's own
// condition). NOTE: the office cannot run the it2 installer — an
// "Install it2 now" answer settles the dialog as "installed" and the
// CLI's own post-answer verification re-prompts if the tool is still
// missing (fail-safe).
func claudeRenderIt2Setup(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		TmuxAvailable bool `json:"tmuxAvailable"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	opts := []state.QuestionOption{
		{Label: "Install it2 now", Description: "Requires Python (uvx, pipx, or pip)"},
	}
	pairs := []any{"Install it2 now", enumResult("installed")}
	if payload.TmuxAvailable {
		opts = append(opts, state.QuestionOption{Label: "Use tmux instead", Description: "Opens teammates in a separate tmux session"})
		pairs = append(pairs, "Use tmux instead", enumResult("use-tmux"))
	}
	render := claudeDialogPage("To use native iTerm2 split panes for teammates, you need the it2 CLI tool.",
		"This enables teammates to appear as split panes within your current window.",
		opts, false)
	return render.withLabelResults("it2_setup", pairs...), true
}

// ---------------- F3: structured results ----------------

// claudeRenderGoalProposal — payload {condition} -> result {approved:
// bool, explicit?:bool}. Title/body are the CLI's component copy verbatim
// (binary: "Claude proposed a session goal", confirmLabel "Set this goal"
// -> {approved:!0,explicit:!0}, cancelLabel "Not now" -> {approved:!1,
// explicit:!0}).
func claudeRenderGoalProposal(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Condition string `json:"condition"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	body := strings.TrimSpace(payload.Condition)
	if body != "" {
		body += "\n\n"
	}
	body += "Approving sets this as the session goal, like running /goal: after each turn a separate check decides whether the condition is met, and Claude keeps working until it is."
	render := claudeDialogPage("Claude proposed a session goal", body, []state.QuestionOption{
		{Label: "Set this goal"},
		{Label: "Not now"},
	}, false)
	return render.withLabelResults("goal_proposal",
		"Set this goal", json.RawMessage(`{"approved":true,"explicit":true}`),
		"Not now", json.RawMessage(`{"approved":false,"explicit":true}`),
	), true
}

// claudeFlaggedAllowRemoveAll is the pseudo-option of the
// auto_mode_flagged_allow page: picking it expands to EVERY flagged rule.
const claudeFlaggedAllowRemoveAll = "Remove them all"

// claudeRenderAutoModeFlaggedAllow — payload {flagged[]string, runId} ->
// result UNION {toRemove:[...]} | "cancelled". ONE multi-select page over
// the flagged rule strings plus the "Remove them all" row; confirming an
// EMPTY selection settles {toRemove:[]} ("leave them", the CLI's own "Your
// setup is saved either way" semantics).
func claudeRenderAutoModeFlaggedAllow(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Flagged []string `json:"flagged"`
		RunID   string   `json:"runId"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	var rules []string
	for _, r := range payload.Flagged {
		if s := strings.TrimSpace(r); s != "" {
			rules = append(rules, s)
		}
	}
	if len(rules) == 0 {
		return claudeDialogRender{}, false // nothing to review: park
	}
	opts := make([]state.QuestionOption, 0, len(rules)+1)
	for _, r := range rules {
		opts = append(opts, state.QuestionOption{Label: r})
	}
	opts = append(opts, state.QuestionOption{Label: claudeFlaggedAllowRemoveAll})
	render := claudeDialogPage("Auto-mode setup flagged some permission rules for review",
		"Select the rules to remove from the allow list — your setup is saved either way.",
		opts, true)
	render.meta = claudeDialogMeta{
		kind:    "auto_mode_flagged_allow",
		family:  dialogFamilyFlaggedAllow,
		flagged: rules,
	}
	return render, true
}

// claudeRenderSandboxNetworkAccess — payload {host, port?} -> result
// UNION {allow:bool, persistToSettings:bool} | "cancelled". The CLI's own
// rows: "Yes" -> {allow:!0,persistToSettings:!1}, the don't-ask-again row
// -> {allow:!0,persistToSettings:!0,persistRow:<rendered node>} (the
// office OMITS persistRow — process-local render data, and the field is
// optional in the result schema), "No" -> {allow:!1,persistToSettings:!1}.
func claudeRenderSandboxNetworkAccess(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		Host string `json:"host"`
		Port *int64 `json:"port"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	subject := strings.TrimSpace(payload.Host)
	if payload.Port != nil {
		subject += fmt.Sprintf(":%d", *payload.Port)
	}
	render := claudeDialogPage("Claude wants network access", subject, []state.QuestionOption{
		{Label: "Yes"},
		{Label: "Yes, don't ask again"},
		{Label: "No"},
	}, false)
	return render.withLabelResults("sandbox_network_access",
		"Yes", json.RawMessage(`{"allow":true,"persistToSettings":false}`),
		"Yes, don't ask again", json.RawMessage(`{"allow":true,"persistToSettings":true}`),
		"No", json.RawMessage(`{"allow":false,"persistToSettings":false}`),
	), true
}

// claudeRenderPeerInboundApproval — payload {holdCause, preview} ->
// result {behavior:"approve"|"deny"|"cancelled"}. Title/options are the
// CLI's component copy verbatim (binary: "A message from another session
// needs your approval", [{approve,"Deliver this message to Claude"},
// {deny,"Deny — drop it and tell the sender it was declined"}]).
func claudeRenderPeerInboundApproval(rawPayload json.RawMessage) (claudeDialogRender, bool) {
	var payload struct {
		HoldCause string `json:"holdCause"`
		Preview   string `json:"preview"`
	}
	if len(rawPayload) > 0 {
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return claudeDialogRender{}, false
		}
	}
	body := strings.TrimSpace(payload.Preview)
	if c := strings.TrimSpace(payload.HoldCause); c != "" {
		if body != "" {
			body += "\n\n"
		}
		body += "holdCause: " + c
	}
	render := claudeDialogPage("A message from another session needs your approval", body, []state.QuestionOption{
		{Label: "Deliver this message to Claude"},
		{Label: "Deny — drop it and tell the sender it was declined"},
	}, false)
	return render.withLabelResults("peer_inbound_approval",
		"Deliver this message to Claude", json.RawMessage(`{"behavior":"approve"}`),
		"Deny — drop it and tell the sender it was declined", json.RawMessage(`{"behavior":"deny"}`),
	), true
}
