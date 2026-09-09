package backend

// Codex uses the CLI's public JSONL protocol. Each turn is a bounded child
// process; later turns resume the exact thread id returned by thread.started.
// Protocol: https://developers.openai.com/codex/noninteractive
// Authentication and model defaults belong to the user's Codex installation.
import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/theboringhumane/theboringfloor/internal/charter"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/plantools"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

type codexBackend struct {
	cfg                      *config.Config
	mu                       sync.Mutex
	fl                       *flow
	bin, dir, primary        string
	cmd                      *exec.Cmd
	done                     chan struct{}
	started, stopped, bypass bool
}

func NewCodex(bin, dir string, cfg *config.Config) state.Backend {
	if bin == "" {
		bin = config.Env("CODEX_BIN")
	}
	if bin == "" {
		bin = "codex"
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return &codexBackend{bin: bin, dir: dir, fl: newFlow(), cfg: cfgOrDefault(cfg)}
}
func (b *codexBackend) Mode() state.Mode { return state.ModeLive }
func (b *codexBackend) Start(emit func(state.Event)) error {
	b.mu.Lock()
	if b.started || b.stopped {
		b.mu.Unlock()
		return errors.New("codex backend already started or stopped")
	}
	bin, err := exec.LookPath(b.bin)
	if err != nil {
		b.mu.Unlock()
		return fmt.Errorf("Codex CLI unavailable: install Codex and run codex login: %w", err)
	}
	b.bin, b.started = bin, true
	b.mu.Unlock()
	b.fl.setEmit(emit)
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] backend: codex"})
	b.fl.emit(state.Event{Kind: state.EvStatus, Text: "Codex ready · workspace sandbox · saved CLI login"})
	return nil
}
func (b *codexBackend) PrimaryID() string { b.mu.Lock(); defer b.mu.Unlock(); return b.primary }
func (b *codexBackend) PrimaryOverride(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil {
		b.primary = id
	}
}
func (b *codexBackend) SetBypassPermissions(on bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		return errors.New("respawn required")
	}
	b.bypass = on
	return nil
}
func (b *codexBackend) NewOffice() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil {
		return "", errors.New("Codex is busy; stop or finish the current turn first")
	}
	b.primary = ""
	return "", nil
}
func (b *codexBackend) ResumeOffice(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil {
		return errors.New("Codex is busy")
	}
	b.primary = id
	return nil
}
func (b *codexBackend) SwapPrimary(id string) error { return b.ResumeOffice(id) }
func (b *codexBackend) Send(text string) error      { return b.send(text, nil, "") }
func (b *codexBackend) SendWith(text string, atts []state.Attachment) error {
	return b.send(text, atts, "")
}
func (b *codexBackend) SendAgent(text, agent string) error { return b.send(text, nil, agent) }

func (b *codexBackend) SendAgentWith(text string, atts []state.Attachment, agent string) error {
	return b.send(text, atts, agent)
}

func codexArgs(id, agent string, bypass bool, images []string) []string {
	sandbox := "workspace-write"
	if agent == "plan" {
		sandbox = "read-only"
	}
	// Config overrides are accepted for both exec and exec resume (resume
	// deliberately has a smaller flag set than exec).
	args := []string{"exec", "-c", "sandbox_mode=" + fmt.Sprintf("%q", sandbox), "-c", "approval_policy=\"never\""}
	if bypass && agent != "plan" {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	if id != "" {
		args = append(args, "resume")
	}
	args = append(args, "--json", "--skip-git-repo-check")
	for _, p := range images {
		args = append(args, "--image", p)
	}
	if id != "" {
		args = append(args, id)
	}
	return append(args, "-")
}

func (b *codexBackend) send(text string, atts []state.Attachment, agent string) error {
	var images, names []string
	prompt := text
	for _, a := range atts {
		path, err := filepath.Abs(a.Path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("attachment is not a regular file: %s", a.Name)
		}
		names = append(names, a.Name)
		if strings.HasPrefix(a.Mime, "image/") {
			images = append(images, path)
			continue
		}
		if info.Size() > 1<<20 {
			return fmt.Errorf("text attachment %s exceeds 1 MiB", a.Name)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		file.Close()
		if err != nil {
			return err
		}
		if len(raw) > 1<<20 {
			return fmt.Errorf("text attachment %s exceeds 1 MiB", a.Name)
		}
		prompt += "\n\nAttached file (reference data): " + a.Name + "\n" + string(raw)
	}
	if agent == "plan" {
		prompt = plantools.PlanningPrompt + "\n\n" + prompt
	}
	prompt = plantools.PromptPreamble + "\n\n" + prompt
	b.mu.Lock()
	if !b.started || b.stopped {
		b.mu.Unlock()
		return errors.New("Codex is not running")
	}
	if b.cmd != nil {
		b.mu.Unlock()
		return errors.New("Codex is working; wait for the current turn or use /stop")
	}
	cmd := exec.Command(b.bin, codexArgs(b.primary, agent, b.bypass, images)...)
	cmd.Dir = b.dir
	if b.primary == "" {
		prompt = charter.Text + "\n\n" + prompt
	}
	cmd.Stdin = strings.NewReader(teamPrompt(b.cfg, prompt))
	isolateProcessGroup(cmd)
	// CommandContext is not enough: kill the group so tool grandchildren do
	// not survive a stopped turn. WaitDelay also bounds inherited pipe handles.
	cmd.WaitDelay = stopKillGrace
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		b.mu.Unlock()
		return err
	}
	stderr := &codexTail{}
	cmd.Stderr = stderr
	if err = cmd.Start(); err != nil {
		b.mu.Unlock()
		return err
	}
	done := make(chan struct{})
	b.cmd, b.done = cmd, done
	b.mu.Unlock()
	turn := fmt.Sprintf("codex-%d", nowMs())
	b.fl.emit(state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{ID: turn + "-user", From: "user", Text: text, At: nowMs(), Meta: state.AttachMeta(names)}})
	b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{ID: turn, From: "boss", Pending: true, At: nowMs()}})
	// Send waits only for this turn's I/O, on the caller's tea.Cmd. This also
	// keeps temporary image attachments alive until the CLI has read them.
	b.readTurn(stdout, cmd, turn, stderr)
	b.mu.Lock()
	b.cmd = nil
	close(done)
	b.mu.Unlock()
	return nil
}

type codexTail struct{ data []byte }

func (t *codexTail) Write(p []byte) (int, error) {
	n := len(p)
	t.data = append(t.data, p...)
	if len(t.data) > 8000 {
		t.data = t.data[len(t.data)-8000:]
	}
	return n, nil
}

type codexWire struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Error    struct {
		Message string `json:"message"`
	} `json:"error"`
	Item struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Text    string `json:"text"`
		Command string `json:"command"`
		Status  string `json:"status"`
		Output  string `json:"aggregated_output"`
		Tool    string `json:"tool"`
		Server  string `json:"server"`
		Query   string `json:"query"`
		Changes []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"changes"`
	} `json:"item"`
	Usage struct {
		Input  int64 `json:"input_tokens"`
		Output int64 `json:"output_tokens"`
		Cached int64 `json:"cached_input_tokens"`
	} `json:"usage"`
}

func (b *codexBackend) readTurn(r io.Reader, cmd *exec.Cmd, turn string, stderr *codexTail) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 65536), 4<<20)
	failed, completed := "", false
	for sc.Scan() {
		var e codexWire
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if e.Type == "thread.started" && e.ThreadID != "" {
			b.mu.Lock()
			b.primary = e.ThreadID
			b.mu.Unlock()
		}
		if e.Type == "turn.completed" {
			completed = true
			b.fl.emit(state.Event{Kind: state.EvUsage, CallID: turn, TokensIn: e.Usage.Input, TokensOut: e.Usage.Output, TokensCacheRead: e.Usage.Cached})
		}
		if e.Type == "turn.failed" {
			failed = e.Error.Message
			if failed == "" {
				failed = "Codex turn failed"
			}
		}
		if e.Type == "error" {
			failed = e.Message
		}
		if strings.HasPrefix(e.Type, "item.") {
			b.emitCodexItem(e, turn)
		}
	}
	if err := sc.Err(); err != nil {
		failed = err.Error()
		_ = signalProcessGroup(cmd.Process, syscall.SIGKILL)
	}
	err := cmd.Wait()
	if err != nil && failed == "" {
		failed = strings.TrimSpace(string(stderr.data))
		if failed == "" {
			failed = err.Error()
		}
	}
	if !completed && failed == "" {
		failed = "Codex ended before completing the turn"
	}
	if failed != "" {
		failed = "Codex: " + failed
	}
	b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{ID: turn, From: "boss", Text: failed, At: nowMs()}})
}
func (b *codexBackend) emitCodexItem(e codexWire, turn string) {
	it := e.Item
	done := e.Type == "item.completed"
	id := turn + "-" + it.ID
	switch it.Type {
	case "agent_message":
		if done {
			it.Text = plantools.Scrub(it.Text, func(d plantools.Directive) {
				kind := state.EvPlanGetApproved
				if d.Kind == plantools.Present {
					kind = state.EvPlanPresent
				}
				if d.Kind == plantools.Update {
					kind = state.EvPlanUpdate
				}
				b.fl.emit(state.Event{Kind: kind, PlanToolText: d.Text})
			})
		}
		b.fl.emit(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{ID: "bossmsg-" + id, From: "boss", Kind: "boss", Text: it.Text, Pending: !done, At: nowMs()}})
	case "reasoning":
		b.fl.emit(state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss", CallID: id, Text: it.Text, Done: done})
	case "command_execution", "mcp_tool_call", "web_search", "file_change":
		name, summary := it.Type, it.Command
		if it.Type == "command_execution" {
			name = "Bash"
		}
		if it.Type == "mcp_tool_call" {
			name = it.Tool
			summary = it.Server
		}
		if it.Type == "web_search" {
			name = "Search"
			summary = it.Query
		}
		if it.Type == "file_change" {
			name = "Edit"
			var paths []string
			for _, c := range it.Changes {
				paths = append(paths, c.Path)
			}
			summary = strings.Join(paths, ", ")
		}
		status := "running"
		if done {
			status = "done"
		}
		if it.Status == "failed" {
			status = "error"
		}
		output := it.Output
		if len(output) > 8000 {
			output = "…" + output[len(output)-8000:]
		}
		b.fl.emit(state.Event{Kind: state.EvTool, EmployeeID: "boss", EmployeeName: "boss", CallID: id, ToolName: name, ToolSummary: summary, ToolState: status, ToolOutput: output})
	}
}
func (b *codexBackend) AbortSessions() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil {
		return nil
	}
	return signalProcessGroup(b.cmd.Process, syscall.SIGKILL)
}
func (b *codexBackend) Stop() error {
	b.mu.Lock()
	b.stopped = true
	if b.cmd != nil {
		_ = signalProcessGroup(b.cmd.Process, syscall.SIGKILL)
	}
	b.mu.Unlock()
	b.fl.stop()
	return nil
}
func (*codexBackend) AnswerPermission(string, string) error {
	return errors.New("Codex exec uses the selected sandbox; interactive approvals are unavailable")
}
func (*codexBackend) AnswerQuestion(string, [][]string) error {
	return errors.New("reply to Codex in the conversation")
}
func (*codexBackend) RejectQuestion(string) error { return errors.New("no pending Codex question") }
func (*codexBackend) MCPServers() ([]state.MCPServer, error) {
	return nil, errors.New("manage Codex MCP servers with codex mcp")
}
func (*codexBackend) ReconnectMCP(string) error {
	return errors.New("manage Codex MCP servers with codex mcp")
}

func (b *codexBackend) FreshOnStart() { b.PrimaryOverride("") }
