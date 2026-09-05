// claude_proof.go — the --claude uishot leg: the REAL live claude backend
// (internal/backend/claude.go) against the compiled claudestub binary
// (cmd/claudestub), driven through the REAL app model the synchronous way
// (no tea.Program, no wall clock — every tick-free frame is replayable).
// The opencode path's own proofs read behavior through the same reducer
// and renderer — so --claude --planshot asserts the planshot behaviors
// byte-for-byte identically.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// sendDeep / drainDeep — the synchronous proof driver's FULL message pump.
// drainCmd pumps exactly one cmd level through focusDriver.send, which
// DISCARDS the cmd that level's Update returns; interactive answers (the
// y-press's permAnswerMsg, the question popover's questionAnswerMsg)
// carry the actual backend wire call (AnswerPermission / AnswerQuestion)
// on that SECOND cmd level, so a one-level pump closes the popover in
// state but never writes the control_response. sendDeep recurses until
// the chain empties (timer arms time out harmlessly at 250ms).
func sendDeep(d *focusDriver, msg tea.Msg, depth int) {
	if msg == nil || depth > 8 {
		return
	}
	tm, c := d.m.Update(msg)
	if fm, ok := tm.(app.Model); ok {
		d.m = fm
	}
	drainDeep(d, c, depth+1)
}

func drainDeep(d *focusDriver, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 8 {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(250 * time.Millisecond):
		return // a timer arm (tick/cursor blink): nothing the proof needs
	}
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			drainDeep(d, c, depth+1)
		}
		return
	}
	sendDeep(d, msg, depth+1)
}

// claudeDrive — the artifacts ONE drive yields (frames + the stub-captured
// wire logs). Two drives must be byte-identical.
type claudeDrive struct {
	frame1, frame2, frame3 string
	chatText               string   // every chat transcript row, joined (state-side; the 32-row viewport scrolls)
	roster                 string   // every hired employee name, joined (state-side; agents-tab/floor surfaces are not on screen in the chat shot)
	capture                []string // every stdin line the office wrote
	stdoutLog              []string // every frame the stub emitted
}

// buildClaudeStub compiles cmd/claudestub once per run (a fixed binary
// shared by both drives — the process re-spawns per drive, the bytes
// don't change).
func buildClaudeStub() (string, func(), error) {
	dir, err := os.MkdirTemp("", "theboringfloor-claudestub-bin")
	if err != nil {
		return "", func() {}, err
	}
	bin := filepath.Join(dir, "claude-test-stub")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/claudestub")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", func() {}, fmt.Errorf("go build ./cmd/claudestub: %v\n%s", err, out)
	}
	return bin, func() { os.RemoveAll(dir) }, nil
}

// readLinesSafe reads a file into non-empty lines ("" when absent).
func readLinesSafe(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		lines = append(lines, sc.Text())
	}
	return lines
}

// driveClaudePlan runs the full planshot interaction against the live
// claude backend: ctrl+p plan mode, three scripted boss beats (chatter,
// permission+chatter, dialog+plan), one subagent Task run. Every wait is
// condition-driven (the stub answers in ms) — no wall clock influences
// the frames.
func driveClaudePlan(bin string, drive int) (*claudeDrive, error) {
	scratch, err := os.MkdirTemp("", fmt.Sprintf("theboringfloor-claude-drive-%d-", drive))
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scratch)
	projDir := filepath.Join(os.TempDir(), "theboringfloor-claude-proj")
	_ = os.RemoveAll(projDir)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(projDir)

	capturePath := filepath.Join(scratch, "capture.log")
	stdoutLogPath := filepath.Join(scratch, "stdout.log")
	for _, kv := range [][2]string{
		{"THEFLOOR_HOME", scratch},
		{"THEFLOOR_CLAUDE_CONFIG", filepath.Join(scratch, "claude-config")},
		{"THEFLOOR_CLAUDE_BIN", bin},
		{"THEFLOOR_CLAUDE_STUB_SCENARIO", "planshot"},
		{"THEFLOOR_CLAUDE_STUB_CAPTURE", capturePath},
		{"THEFLOOR_CLAUDE_STUB_STDOUTLOG", stdoutLogPath},
	} {
		old, had := os.LookupEnv(kv[0])
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			return nil, err
		}
		defer func() {
			if had {
				os.Setenv(kv[0], old)
			} else {
				os.Unsetenv(kv[0])
			}
		}()
	}

	b := backend.NewClaude("", projDir, config.Default())
	m := app.New(b, config.Default())
	if !m.SelectTab("chat") {
		return nil, fmt.Errorf("chat tab not selectable")
	}
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})

	ch := make(chan state.Event, 4096)
	if err := b.Start(func(e state.Event) { ch <- e }); err != nil {
		return nil, fmt.Errorf("backend Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	pump := func() {
		for {
			select {
			case e := <-ch:
				d.send(e)
			default:
				return
			}
		}
	}
	waitFor := func(what string, cond func() bool) error {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			pump()
			if cond() {
				return nil
			}
			time.Sleep(2 * time.Millisecond)
		}
		return fmt.Errorf("drive %d: timed out waiting for %s\nframe so far:\n%s\n---DEBUG capture---\n%s\n---DEBUG stdout---\n%s", drive, what, ansi.Strip(d.m.Frame()), strings.Join(readLinesSafe(capturePath), "\n"), strings.Join(readLinesSafe(stdoutLogPath), "\n"))
	}
	key := func(k tea.KeyPressMsg) tea.Cmd {
		tm, c := d.m.Update(k)
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}
	typeIn := func(s string) {
		for _, r := range s {
			_ = key(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	stripped := func() string { return ansi.Strip(d.m.Frame()) }
	chatHas := func(sub string) bool {
		for _, c := range d.m.State().Chat {
			if strings.Contains(c.Text, sub) {
				return true
			}
		}
		return false
	}

	// boot settled: the office's hires + statuses landed
	if err := waitFor("the claude boot (hires + live status)", func() bool {
		return strings.Contains(d.m.State().StatusLine, "live (claude)") || chatHasFilter(d, "live (claude)")
	}); err != nil {
		return nil, err
	}

	// plan mode on (the REAL ctrl+p claim site)
	pump()
	_ = key(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	pump()

	// SEND #1 — chatter beat (stream + pinned): the pane's shape gate must
	// refuse it (empty pane → office floor keeps the slot).
	typeIn("plan the lobby gallery wall")
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 0)
	if err := waitFor("chatter-1 completion", func() bool { return chatHas("quick sync — sent to ops") }); err != nil {
		return nil, err
	}
	frame1 := d.m.Frame()

	// SEND #2 — the permission control round-trip: the popover opens on
	// the control_request, y answers Allow once, the parked turn resumes.
	pump()
	typeIn("ok")
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 0)
	if err := waitFor("the PERMISSION REQUIRED popover", func() bool {
		return strings.Contains(stripped(), "PERMISSION REQUIRED")
	}); err != nil {
		return nil, err
	}
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}), 0) // Allow once → control_response written
	if err := waitFor("chatter-2 completion (answered perm resumed the turn)", func() bool {
		return chatHas("still sketching") || strings.Contains(stripped(), "still sketching")
	}); err != nil {
		return nil, err
	}
	if err := waitFor("the control_response req-owl-1 written once", func() bool {
		for _, ln := range readLinesSafe(capturePath) {
			if ln == `{"type":"control_response","response":{"request_id":"req-owl-1","response":{"behavior":"allow"}}}` {
				return true
			}
		}
		return false
	}); err != nil {
		return nil, err
	}

	// SEND #3 — the dialog control round-trip: the question popover opens,
	// a typed answer returns the dialog, the plan-SHAPED reply presents.
	pump()
	typeIn("now the plan proper")
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 0)
	if err := waitFor("the question popover", func() bool {
		return strings.Contains(stripped(), "Which finish for the east wall")
	}); err != nil {
		return nil, err
	}
	typeIn("matte")
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 0)
	if err := waitFor("the plan-shaped boss reply (azimuth)", func() bool {
		return chatHas("azimuth") || strings.Contains(stripped(), "azimuth")
	}); err != nil {
		return nil, err
	}
	if err := waitFor("the control_response req-q-1 written once", func() bool {
		for _, ln := range readLinesSafe(capturePath) {
			if ln == `{"type":"control_response","response":{"request_id":"req-q-1","response":{"behavior":"allow","answer":"matte"}}}` {
				return true
			}
		}
		return false
	}); err != nil {
		return nil, err
	}
	frame2 := d.m.Frame()

	// SEND #4 — the subagent Task run: hire (skopos-1) + its beats + the
	// task notification RETURNS it (mail + board flip).
	pump()
	typeIn("scan the options too")
	sendDeep(d, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), 0)
	if err := waitFor("the subagent return (skopos-1 mail)", func() bool {
		for _, c := range d.m.State().Chat {
			_ = c
		}
		for _, ml := range d.m.State().Mails {
			if strings.Contains(ml.Subject, "return:") && strings.Contains(ml.Body, "mock scan done") {
				return true
			}
		}
		return false
	}); err != nil {
		return nil, err
	}
	frame3 := d.m.Frame()

	var chatRows []string
	for _, c := range d.m.State().Chat {
		chatRows = append(chatRows, c.Text)
	}
	var rosterNames []string
	for _, e := range d.m.State().Employees {
		rosterNames = append(rosterNames, e.Name)
	}

	return &claudeDrive{
		frame1:    frame1,
		frame2:    frame2,
		frame3:    frame3,
		chatText:  strings.Join(chatRows, "\n"),
		roster:    strings.Join(rosterNames, "\n"),
		capture:   readLinesSafe(capturePath),
		stdoutLog: readLinesSafe(stdoutLogPath),
	}, nil
}

func chatHasFilter(d *focusDriver, sub string) bool {
	for _, c := range d.m.State().Chat {
		if strings.Contains(c.Text, sub) {
			return true
		}
	}
	return false
}

// — frame contract (the same behavior classes the opencode --planshot
// proof asserts; the reducer + renderer don't know which transport fed
// them) —

func assertFrameContains(tag, frame string, wants ...string) error {
	stripped := ansi.Strip(frame)
	for _, want := range wants {
		if !strings.Contains(stripped, want) {
			return fmt.Errorf("%s: frame missing %q", tag, want)
		}
	}
	return nil
}

func assertFrameLacks(tag, frame string, rejects ...string) error {
	stripped := ansi.Strip(frame)
	for _, reject := range rejects {
		if strings.Contains(stripped, reject) {
			return fmt.Errorf("%s: frame must NOT contain %q", tag, reject)
		}
	}
	return nil
}

// runClaudePlanProof — the --claude [--planshot] entry point: TWO
// condition-driven drives, byte-compared; frames, the stub-captured
// stdin/stdout round trip, and the asserts print.
func runClaudePlanProof() error {
	bin, cleanup, err := buildClaudeStub()
	if err != nil {
		return err
	}
	defer cleanup()

	d1, err := driveClaudePlan(bin, 1)
	if err != nil {
		return err
	}
	d2, err := driveClaudePlan(bin, 2)
	if err != nil {
		return err
	}

	fmt.Println("===== UI SHOT · CLAUDE PLAN frame 1/3 — plan mode ACTIVE, boss CHATTER pinned: pane refuses (empty → floor keeps the slot, [plan] badge + idle hint, escape-valve note exactly once) =====")
	fmt.Println(d1.frame1)
	fmt.Println("===== UI SHOT =====")
	if err := assertFrameContains("claude A", d1.frame1,
		"[=BOSS=]",                    // the office floor still owns the slot
		"[plan]",                      // statusbar agent badge
		"[office] plan mode",          // the toggle's own notice
		"plan the lobby gallery wall", // the user's ask, chat lane
		"quick sync — sent to ops",    // the boss's CHATTER reply, chat-only
		"boss is chatting",            // the escape-valve note (once per session)
	); err != nil {
		return err
	}
	if n := strings.Count(d1.chatText, "boss is chatting; when it writes a plan it lands on the left"); n != 1 {
		return fmt.Errorf("claude A: the escape-valve note must fire EXACTLY once in the transcript, got %d", n)
	}
	if err := assertFrameLacks("claude A", d1.frame1,
		"PLAN · markdown",         // no pane header while the buffer is empty
		"click to edit",           // no pane footer either
		"didn't look like a plan", // the kept-last-plan note never fires over an empty pane
	); err != nil {
		return err
	}

	fmt.Println("===== UI SHOT · CLAUDE PLAN frame 2/3 — permission control round-trip answered (req-owl-1 → allow), dialog answered (req-q-1 → \"matte\"), plan-SHAPED reply PRESENTED into the markdown pane (azimuth) =====")
	fmt.Println(d1.frame2)
	fmt.Println("===== UI SHOT =====")
	if err := assertFrameContains("claude B", d1.frame2,
		"PLAN · markdown", // pane header in the floor slot
		"azimuth",         // the boss's plan text, mirrored into the pane (unique marker)
		"[plan]",          // statusbar agent badge
		"click to edit",   // pane footer hint — the pane is UNFOCUSED
		"Goal",            // the plan body's adopted heading
		"A gallery lobby wall that feels calm, not", // the plan body itself
	); err != nil {
		return err
	}
	if n := strings.Count(d1.chatText, "boss is chatting"); n != 1 {
		return fmt.Errorf("claude B: the escape-valve note must remain exactly one transcript row, got %d", n)
	}
	if err := assertFrameLacks("claude B", d1.frame2,
		"[=BOSS=]",                // the pane REPLACES the office floor in the slot
		"didn't look like a plan", // no rejection note ever fired
	); err != nil {
		return err
	}

	fmt.Println("===== UI SHOT · CLAUDE PLAN frame 3/3 — the subagent Task run: skopos-1 hired (roster), beats attributed, task_notification RETURNED (mail + board flip) =====")
	fmt.Println(d1.frame3)
	fmt.Println("===== UI SHOT =====")
	if err := assertFrameContains("claude C", d1.frame3,
		"Explore Task — scan the lobby poster mocks", // the subagent's thread header
		"↳ Glob mocks/*.md",                          // its own tool beat, attributed to the thread
	); err != nil {
		return err
	}
	// the hire's NAME lands on the roster/floor surfaces (agents tab), not
	// the chat tab — assert membership on the state roster (the event-level
	// skopos-* pin lives in claude_events_test.go's SubagentLifecycle).
	if !strings.Contains(d1.roster, "skopos-1") {
		return fmt.Errorf("claude C: roster missing skopos-1 (the Task run's scout hire), roster:\n%s", d1.roster)
	}

	// the stub-captured wire: stdin (what the office wrote) is the
	// byte-pin of the Send → user-line + control round-trip contract.
	fmt.Println("--- stub-captured stdin (what the office wrote, verbatim) ---")
	for _, ln := range d1.capture {
		fmt.Println(ln)
	}
	fmt.Println("--- stub-emitted stdout frame sample (Send → init → assistant → result round trip, first turn) ---")
	limit := 12
	if len(d1.stdoutLog) < limit {
		limit = len(d1.stdoutLog)
	}
	for _, ln := range d1.stdoutLog[:limit] {
		fmt.Println(ln)
	}

	// capture contract: EXACT wire schedule — user line ×4, one
	// control_response per answered ask, in order.
	wantCapture := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"plan the lobby gallery wall"}]},"parent_tool_use_id":null}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"ok"}]},"parent_tool_use_id":null}`,
		`{"type":"control_response","response":{"request_id":"req-owl-1","response":{"behavior":"allow"}}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"now the plan proper"}]},"parent_tool_use_id":null}`,
		`{"type":"control_response","response":{"request_id":"req-q-1","response":{"behavior":"allow","answer":"matte"}}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"scan the options too"}]},"parent_tool_use_id":null}`,
	}
	if strings.Join(d1.capture, "\n") != strings.Join(wantCapture, "\n") {
		return fmt.Errorf("claude wire schedule drifted:\n got:\n%s\nwant:\n%s", strings.Join(d1.capture, "\n"), strings.Join(wantCapture, "\n"))
	}

	// byte-determinism: drive 1 == drive 2, every artifact.
	if d1.frame1 != d2.frame1 || d1.frame2 != d2.frame2 || d1.frame3 != d2.frame3 {
		return fmt.Errorf("claude: frames differ between the two drives (byte-determinism violated)")
	}
	if strings.Join(d1.capture, "\n") != strings.Join(d2.capture, "\n") ||
		strings.Join(d1.stdoutLog, "\n") != strings.Join(d2.stdoutLog, "\n") {
		return fmt.Errorf("claude: the stub wire logs differ between the two drives")
	}

	fmt.Println("asserts: OK — the live claude backend (claude -p stream-json) rides the same planshot behaviors as the opencode path: ctrl+p flips ONLY the mode; TWO boss chatter replies (status narration) NEVER present (frame 1: floor kept, [plan] badge, escape-valve note exactly once); the permission control round-trip (req-owl-1 can_use_tool → control_response behavior allow) and the dialog round-trip (req-q-1 → \"matte\") resume their parked turns; the plan-SHAPED reply presents into the pane (frame 2: PLAN · markdown + azimuth, floor swapped out); the subagent Task run hires skopos-1 and RETURNS via task_notification (frame 3); the stdin wire schedule is byte-pinned; two drives byte-identical")
	fmt.Println("CLAUDE-PLANSHOT: OK")
	return nil
}
