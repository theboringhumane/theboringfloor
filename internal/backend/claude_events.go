// claude_events.go — normalize the claude CLI's stream-json JSONL frames
// into state.Events for the office floor. Pure helpers only: no I/O, no
// timers, no UI framework — mirrors events.go (the opencode SSE mapper).
// The live claude backend (claude.go) owns the process; this module
// decides WHAT one stdout line means for the office, given a mutable
// context object.
//
// Wire reference (claude Code >= 2.x, `-p --input-format stream-json
// --output-format stream-json --verbose --include-partial-messages`):
//
//	system/init                      -> status "[claude] init model=.. session=.." (+ mcp pin)
//	stream_event (content_block_*)   -> boss bubble growth / tool folding / thought growth
//	assistant (snapshot)             -> pinned chat-boss ("bossmsg-"+uuid), thoughts, tool_use
//	user (tool_result content)       -> tool done; Task result -> returned + mail + task done
//	control_request can_use_tool     -> EvPermission pending (ID = request_id)
//	control_request request_user_dialog -> EvQuestion pending (ID = request_id)
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
type claudeControlRequest struct {
	Subtype      string         `json:"subtype"`       // can_use_tool | request_user_dialog
	ToolName     string         `json:"tool_name"`     // can_use_tool
	ToolInput    map[string]any `json:"tool_input"`    // can_use_tool
	InputPreview string         `json:"input_preview"` // can_use_tool
	Question     string         `json:"question"`      // request_user_dialog
	Options      []string       `json:"options"`       // request_user_dialog
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

	// tools / tasks
	tools map[string]*claudeToolHold // tool_use.id -> hold
	tasks map[string]*claudeTask     // Task tool_use.id -> run

	nameCounts map[state.EmployeeRole]int
	seatSeq    int

	pendingPerms     map[string]permHold // request_id -> hold
	pendingQuestions map[string]permHold // request_id -> hold

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
		tools:            make(map[string]*claudeToolHold),
		tasks:            make(map[string]*claudeTask),
		nameCounts:       make(map[state.EmployeeRole]int),
		pendingPerms:     make(map[string]permHold),
		pendingQuestions: make(map[string]permHold),
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
func (ctx *claudeNormCtx) claudeAssistantPin(uuid, text string, now int64) []state.Event {
	if uuid == "" || ctx.pinned[uuid] {
		return nil
	}
	if strings.TrimSpace(text) == "" {
		return nil
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
		ctx.blockKind = map[int]string{}
		ctx.blockTool = map[int]string{}
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
			ownerID, ownerName := ctx.ownerFor(parentToolUseID)
			uuid := ctx.curMsgUUID
			if uuid == "" {
				uuid = "stream"
			}
			ctx.thoughtOpen = uuid
			return []state.Event{{
				Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
				Text: capThought(inner.Delta.Thinking), CallID: "think-" + uuid, Done: false,
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
				return []state.Event{{
					Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
					CallID: "think-" + uuid, Done: true,
				}}
			}
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
	uuid := raw.UUID
	if uuid == "" {
		uuid = raw.Message.ID
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
				evs = append(evs, state.Event{
					Kind: state.EvThought, EmployeeID: ownerID, EmployeeName: ownerName,
					Text: capThought(block.Thinking), CallID: "think-" + uuid, Done: true,
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
		summary := req.InputPreview
		if strings.TrimSpace(summary) == "" {
			summary = claudeToolSummary(req.ToolInput)
		}
		toolName := claudeToolName(req.ToolName)
		ctx.pendingPerms[raw.RequestID] = permHold{
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Title: toolName, Summary: summary,
		}
		evs := []state.Event{{
			Kind: state.EvPermission, PermissionID: raw.RequestID,
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
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
		ownerID, ownerName := ctx.ownerFor(raw.ParentToolUseID)
		question := strings.TrimSpace(req.Question)
		if question == "" {
			question = "question from the floor"
		}
		var options []string
		var items []state.QuestionItem
		item := state.QuestionItem{Question: shortTitle(question, 240)}
		for _, opt := range req.Options {
			if s := strings.TrimSpace(opt); s != "" {
				options = append(options, s)
				item.Options = append(item.Options, state.QuestionOption{Label: s})
			}
		}
		items = append(items, item)
		summary := strings.Join(options, " | ")
		if summary == "" {
			summary = "free-form answer"
		}
		ctx.pendingQuestions[raw.RequestID] = permHold{
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Title: "question", Summary: summary,
		}
		return []state.Event{{
			Kind: state.EvQuestion, QuestionID: raw.RequestID,
			SessionID: raw.SessionID, EmployeeID: ownerID, EmployeeName: ownerName,
			Text: shortTitle(question, 240), ToolSummary: shortTitle(summary, 120),
			ToolState: "pending", Questions: items,
		}}
	}
	return nil
}
