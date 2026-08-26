// claudestub — a deterministic stand-in for the Claude Code CLI in
// stream-json mode. It speaks the exact protocol subset theboringoffice's
// claude backend reads/writes (`claude -p --input-format stream-json
// --output-format stream-json --verbose --include-partial-messages`):
//
//	stdout  <- the protocol stream (system/init, stream_event, assistant,
//	          user tool_result, control_request, result)
//	stdin   -> user messages + control_response lines (parsed, scripted)
//	stderr  -> nothing (the backend's error buffer stays empty)
//
// Every frame/uuid/usage number is SCRIPTED CONSTANT — two runs over the
// same stdin schedule are byte-identical (no wall clock). Handshakes
// (permission / dialog) hang the scripted turn until the matching
// control_response arrives, exactly like the real lane.
//
// Env:
//
//	THEBORINGOFFICE_CLAUDE_STUB_SCENARIO  planshot (default) | roundtrip | silent | no-init
//	THEBORINGOFFICE_CLAUDE_STUB_CAPTURE   append every STDIN line here (proof)
//	THEBORINGOFFICE_CLAUDE_STUB_STDOUTLOG append every STDOUT frame here (proof)
//	THEBORINGOFFICE_CLAUDE_STUB_ARGV      record the exact argv (proof)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ---------------- constant frames (scripted schedule) ----------------

const stubSession = "00000000-0000-4000-8000-0000000000c1"

// versionLine covers the "version line" the office tolerates before init
// (unknown subtype — the mapper ignores it silently, like the real CLI's
// leading chatter).
const versionLine = `{"type":"system","subtype":"version","version":"2.1.246"}`

func initLine(session string) string {
	return fmt.Sprintf(`{"type":"system","subtype":"init","cwd":"%s","session_id":"%s","mcp_servers":[{"name":"office-memo","status":"connected"},{"name":"filesystem","status":"connected"}],"model":"claude-stub-4","permissionMode":"default","apiKeySource":"stub","claude_code_version":"2.1.246","uuid":"init-0000","tools":["Task","Bash","Read","Write","Edit","Glob","Grep"]}`, cwd(), session)
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return d
}

// streamEnvel wraps an Anthropic inner event in the claude stream_event
// envelope (constant session/parent per scripted beat).
func streamEnvel(inner, parent string) string {
	return fmt.Sprintf(`{"type":"stream_event","event":%s,"session_id":"%s","parent_tool_use_id":%s}`, inner, stubSession, parent)
}

const nullParent = "null"

// msgStart / blockStart / textDelta / blockStop script one message's
// partial stream.
func msgStart(id string) string {
	return streamEnvel(fmt.Sprintf(`{"type":"message_start","message":{"id":"%s","type":"message","role":"assistant","model":"claude-stub-4"}}`, id), nullParent)
}

func blockStartText() string {
	return streamEnvel(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`, nullParent)
}

func textDelta(text string) string {
	body, _ := json.Marshal(text)
	return streamEnvel(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":`+string(body)+`}}`, nullParent)
}

func blockStop() string {
	return streamEnvel(`{"type":"content_block_stop","index":0}`, nullParent)
}

// assistantMsg snapshots one assistant message (full content, after any
// partial stream).
func assistantMsg(id, parent string, blocks ...string) string {
	content := strings.Join(blocks, ",")
	return fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","model":"claude-stub-4","content":[%s]},"session_id":"%s","uuid":"%s","parent_tool_use_id":%s}`, content, stubSession, id, parent)
}

func textBlock(text string) string {
	body, _ := json.Marshal(text)
	return `{"type":"text","text":` + string(body) + `}`
}

func thinkingBlock(text string) string {
	body, _ := json.Marshal(text)
	return `{"type":"thinking","thinking":` + string(body) + `,"signature":"stub"}`
}

func toolUseBlock(id, name, input string) string {
	return fmt.Sprintf(`{"type":"tool_use","id":"%s","name":"%s","input":%s}`, id, name, input)
}

// toolResultUser returns a user frame closing a tool_use.
func toolResultUser(callID, result string) string {
	body, _ := json.Marshal(result)
	return fmt.Sprintf(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"%s","content":%s,"is_error":false}]},"session_id":"%s","parent_tool_use_id":null}`, callID, body, stubSession)
}

// controlPerm / controlAsk are the out-of-band asks.
func controlPerm(requestID, toolName, input, preview string) string {
	return fmt.Sprintf(`{"type":"control_request","request_id":"%s","request":{"subtype":"can_use_tool","tool_name":"%s","tool_input":%s,"input_preview":"%s"},"session_id":"%s"}`, requestID, toolName, input, preview, stubSession)
}

func controlAsk(requestID, question string, options []string) string {
	opts, _ := json.Marshal(options)
	return fmt.Sprintf(`{"type":"control_request","request_id":"%s","request":{"subtype":"request_user_dialog","question":"%s","options":%s},"session_id":"%s"}`, requestID, question, opts, stubSession)
}

// resultLine ends a turn (usage numbers are running totals — the office's
// delta arm reads the growth).
func resultLine(uuid string, in, out, cacheR, cacheW int, cost float64) string {
	return fmt.Sprintf(`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"ok","session_id":"%s","total_cost_usd":%v,"usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d},"modelUsage":{"claude-stub-4":{"inputTokens":%d,"outputTokens":%d,"cacheReadInputTokens":%d,"cacheCreationInputTokens":%d,"webSearchRequests":0,"contextWindow":200000}},"permission_denials":[],"uuid":"%s"}`, stubSession, cost, in, out, cacheR, cacheW, in, out, cacheR, cacheW, uuid)
}

// ---------------- planshot script ----------------
// Mirrors the opencode path's planshot behavior classes: TWO boss CHATTER
// replies (status narration — the plan pane's shape gate keeps them out),
// then a plan-SHAPED reply (# Goal / # Steps) that presents. Between them:
// ONE tool_use + tool_result, ONE permission control_request (the turn
// parks until answered), ONE question dialog (parks until answered), and
// ONE subagent Task run (task_started -> frames -> task_notification).

const (
	chatter1 = "quick sync — sent to ops; scanning the lobby wall options now, the structured plan lands in a beat."
	chatter2 = "still sketching — matte panels vs glass lanes, comparing sightlines; the plan proper is next."
	planText = "# Goal\n" +
		"A gallery lobby wall that feels calm, not corporate.\n" +
		"# Steps\n" +
		"- matte panels azimuth-washed along the long east wall\n" +
		"- glassmorphic kanban lanes for the return shelf by the tea machine\n" +
		"- zero clerical chrome anywhere near the entrance doors"
	askQuestion = "Which finish for the east wall — matte or gloss, or both in bands?"
	taskDesc    = "scan the lobby poster mocks"
	taskResult  = "mock scan done: 3 fixture scans summarized"
)

// turn1: stream chatter-1 (two deltas) + its snapshot + the turn result.
func (s *stub) turn1() {
	s.emit(msgStart("msg-00000001"))
	s.emit(blockStartText())
	s.emit(textDelta("quick sync — sent to ops; scanning the lobby "))
	s.emit(textDelta("wall options now, the structured plan lands in a beat."))
	s.emit(blockStop())
	s.emit(assistantMsg("msg-00000001", nullParent, textBlock(chatter1)))
	s.emit(resultLine("res-00000001", 1000, 200, 4000, 900, 0.0042))
}

// turn2: a Write tool_use (streamed input_json) + its tool_result + a
// permission ask — the turn PARKS until the office answers behavior allow.
func (s *stub) turn2Phase1() {
	s.emit(msgStart("msg-00000002"))
	s.emit(streamEnvel(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_0001","name":"Write","input":{}}}`, nullParent))
	s.emit(streamEnvel(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"file"}}`, nullParent))
	s.emit(streamEnvel(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"_path\": \"hello.html\", \"content\": \"<h1>lobby</h1>\"}"}}`, nullParent))
	s.emit(streamEnvel(`{"type":"content_block_stop","index":0}`, nullParent))
	s.emit(assistantMsg("msg-00000002", nullParent, toolUseBlock("toolu_0001", "Write", `{"file_path":"hello.html","content":"<h1>lobby</h1>"}`)))
	s.emit(toolResultUser("toolu_0001", "File created successfully at: /tmp/hello.html"))
	s.emit(controlPerm("req-owl-1", "Bash", `{"command":"open hello.html"}`, "open hello.html"))
	s.parkedPerm = "req-owl-1"
}

func (s *stub) turn2Phase2() {
	// answered: the turn resumes with chatter-2 and settles
	s.emit(msgStart("msg-00000002b"))
	s.emit(blockStartText())
	s.emit(textDelta("still sketching — matte panels vs glass lanes, "))
	s.emit(textDelta("comparing sightlines; the plan proper is next."))
	s.emit(blockStop())
	s.emit(assistantMsg("msg-00000002b", nullParent, thinkingBlock("chatter is narration, not a plan — the gate will keep it out of the pane"), textBlock(chatter2)))
	s.emit(resultLine("res-00000002", 2400, 460, 8500, 1300, 0.0110))
}

// turn3: a dialog ask — parks until answered — then the plan-SHAPED reply.
func (s *stub) turn3Phase1() {
	s.emit(controlAsk("req-q-1", askQuestion, []string{"matte", "gloss", "both"}))
	s.parkedAsk = "req-q-1"
}

func (s *stub) turn3Phase2() {
	s.emit(msgStart("msg-00000003"))
	s.emit(blockStartText())
	s.emit(textDelta("# Goal\nA gallery lobby wall that feels calm, "))
	s.emit(textDelta("not corporate.\n# Steps\n- matte panels azimuth-washed along the long east wall"))
	s.emit(blockStop())
	s.emit(assistantMsg("msg-00000003", nullParent, textBlock(planText)))
	s.emit(resultLine("res-00000003", 3100, 620, 9100, 1500, 0.0166))
}

// turn4: one subagent Task run — hire via the Task tool_use, its own
// beats ride parent_tool_use_id, the tool_result returns it.
func (s *stub) turn4() {
	taskID := "toolu_task_0001"
	s.emit(assistantMsg("msg-00000004", nullParent, toolUseBlock(taskID, "Task", `{"description":"scan the lobby poster mocks","subagent_type":"Explore","prompt":"scan mocks"}`)))
	// the subagent's own beat (task_started class traffic)
	s.emit(assistantMsg("msg-00000004a", `"`+taskID+`"`, textBlock("reading the mock index…")))
	// its own tool_use + result (task_notification class traffic)
	s.emit(assistantMsg("msg-00000004b", `"`+taskID+`"`, toolUseBlock("toolu_sub_0001", "Glob", `{"pattern":"mocks/*.md"}`)))
	s.emit(fmt.Sprintf(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"%s","content":"3 mocks","is_error":false}]},"session_id":"%s","parent_tool_use_id":"%s"}`, "toolu_sub_0001", stubSession, taskID))
	// the Task's own result returns the run
	s.emit(toolResultUser(taskID, taskResult))
	s.emit(resultLine("res-00000004", 3400, 700, 9100, 1500, 0.0191))
}

// roger turns keep anything beyond n=4 alive (the office queue always
// gets a completed answer).
func (s *stub) roger(text string, n int) {
	id := fmt.Sprintf("msg-r%05d", n)
	s.emit(assistantMsg(id, nullParent, textBlock("Roger: "+text)))
	s.emit(resultLine(fmt.Sprintf("res-r%05d", n), 3500, 710, 9100, 1500, 0.0195))
}

// ---------------- the stub engine ----------------

type stub struct {
	out      *bufio.Writer
	scenario string
	userN    int
	// the handshake lanes: request_id in-flight ("", else parked)
	parkedPerm string
	parkedAsk  string
}

func (s *stub) emit(line string) {
	s.out.WriteString(line)
	s.out.WriteByte('\n')
	s.out.Flush()
	s.logStdout(line)
}

func (s *stub) logStdout(line string) {
	if p := os.Getenv("THEBORINGOFFICE_CLAUDE_STUB_STDOUTLOG"); p != "" {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(line + "\n")
			f.Close()
		}
	}
}

func (s *stub) logCapture(line string) {
	if p := os.Getenv("THEBORINGOFFICE_CLAUDE_STUB_CAPTURE"); p != "" {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(line + "\n")
			f.Close()
		}
	}
}

// frameHeads classifies one stdin line for the scripted handlers.
type stubIn struct {
	Type       string `json:"type"`
	RequestID  string `json:"request_id"`
	Message    struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	Response struct {
		RequestID string `json:"request_id"`
		Response  struct {
			Behavior string `json:"behavior"`
			Answer   string `json:"answer"`
		} `json:"response"`
	} `json:"response"`
}

func (s *stub) onUser(text string) {
	s.userN++
	if s.scenario != "planshot" {
		s.roger(text, s.userN)
		return
	}
	switch s.userN {
	case 1:
		s.turn1()
	case 2:
		s.turn2Phase1()
	case 3:
		s.turn3Phase1()
	case 4:
		s.turn4()
	default:
		s.roger(text, s.userN)
	}
}

// onControlResponse unlocks a parked ask — strict behavior: the turn's
// continuation only plays for the matching request_id (byte-deterministic
// handshakes, never a race).
func (s *stub) onControlResponse(id, behavior, answer string) {
	if s.scenario != "planshot" {
		return
	}
	if id == s.parkedPerm {
		s.parkedPerm = ""
		if behavior == "allow" || behavior == "allow_always" {
			s.turn2Phase2()
		}
		return
	}
	if id == s.parkedAsk {
		s.parkedAsk = ""
		if behavior == "allow" {
			s.turn3Phase2()
		}
	}
}

func main() {
	session := stubSession
	for i, a := range os.Args {
		if a == "--resume" && i+1 < len(os.Args) {
			session = os.Args[i+1]
		}
	}
	_ = session // the --resume pin flows through the init line
	if p := os.Getenv("THEBORINGOFFICE_CLAUDE_STUB_ARGV"); p != "" {
		if f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			f.WriteString(strings.Join(os.Args[1:], " ") + "\n")
			f.Close()
		}
	}

	s := &stub{out: bufio.NewWriter(os.Stdout), scenario: os.Getenv("THEBORINGOFFICE_CLAUDE_STUB_SCENARIO")}
	if s.scenario == "" {
		s.scenario = "planshot"
	}
	switch s.scenario {
	case "no-init":
		fmt.Fprintln(os.Stderr, "claudestub: simulated boot failure")
		os.Exit(2)
	case "silent":
		// never emits init — the office's start timeout path
		select {}
	}

	s.emit(versionLine)
	s.emit(initLine(session))

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		s.logCapture(line)
		var in stubIn
		if json.Unmarshal([]byte(line), &in) != nil {
			continue
		}
		switch in.Type {
		case "user":
			var text string
			for _, c := range in.Message.Content {
				if c.Type == "text" {
					text = c.Text
					break
				}
			}
			s.onUser(text)
		case "control_response":
			s.onControlResponse(in.Response.RequestID, in.Response.Response.Behavior, in.Response.Response.Answer)
		case "control_request":
			// interrupt: settle immediately, claude-style (a result closes the turn)
			s.emit(resultLine("res-interrupted", 0, 0, 0, 0, 0))
		}
	}
}
