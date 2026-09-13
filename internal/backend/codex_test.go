package backend

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/plantools"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestCodexPlanToolsAndAttachments(t *testing.T) {
	dir := t.TempDir()
	bin, args, prompt := filepath.Join(dir, "codex"), filepath.Join(dir, "args"), filepath.Join(dir, "prompt")
	t.Setenv("CODEX_TEST_ARGS", args)
	t.Setenv("CODEX_TEST_PROMPT", prompt)
	script := `#!/bin/sh
printf '%s\n' "$@" > "$CODEX_TEST_ARGS"
cat > "$CODEX_TEST_PROMPT"
cat <<'JSON'
{"type":"thread.started","thread_id":"plan-thread"}
{"type":"item.started","item":{"id":"answer","type":"agent_message","text":"⟦plan-present⟧\n# Draft"}}
{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"⟦plan-present⟧\n# Draft\n\n- Inspect the project.\n⟦/plan-present⟧"}}
{"type":"turn.completed"}
JSON
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "requirements.txt")
	if err := os.WriteFile(path, []byte("Keep keyboard navigation"), 0600); err != nil {
		t.Fatal(err)
	}
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var events []state.Event
	if err := b.Start(func(e state.Event) { events = append(events, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.SendAgentWith("Plan the explorer", []state.Attachment{{Path: path, Name: "requirements.txt", Mime: "text/plain"}}, "plan"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(args)
	if !strings.Contains(string(raw), `sandbox_mode="read-only"`) {
		t.Fatalf("wrong sandbox: %s", raw)
	}
	raw, _ = os.ReadFile(prompt)
	for _, want := range []string{plantools.PromptPreamble, plantools.PlanningPrompt, "Keep keyboard navigation"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing prompt context: %q", want)
		}
	}
	count := 0
	for _, ev := range events {
		if ev.Kind == state.EvPlanPresent {
			count++
			if ev.PlanToolText != "# Draft\n\n- Inspect the project." {
				t.Fatalf("bad plan: %q", ev.PlanToolText)
			}
		}
		if ev.Kind == state.EvChatBoss && !ev.Msg.Pending && strings.Contains(ev.Msg.Text, "⟦plan-present⟧") {
			t.Fatal("marker leaked into settled transcript")
		}
	}
	if count != 1 {
		t.Fatalf("plan emitted %d times", count)
	}
}

func TestCodexStreamsResumesAndReportsToolFailure(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-fake")
	log := filepath.Join(dir, "args")
	script := `#!/bin/sh
printf '%s\n' "$@" >> "$CODEX_TEST_ARGS"
cat >/dev/null
cat <<'JSON'
{"type":"thread.started","thread_id":"thread-first"}
{"type":"item.started","item":{"id":"tool-1","type":"command_execution","command":"go test ./...","status":"in_progress"}}
{"type":"item.completed","item":{"id":"tool-1","type":"command_execution","command":"go test ./...","status":"failed","aggregated_output":"FAIL: useful error"}}
{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"I found a failing check."}}
{"type":"turn.completed","usage":{"input_tokens":20,"output_tokens":8,"cached_input_tokens":3}}
JSON
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_TEST_ARGS", log)
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var events []state.Event
	if err := b.Start(func(e state.Event) { events = append(events, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.Send("Check this project"); err != nil {
		t.Fatal(err)
	}
	if b.PrimaryID() != "thread-first" {
		t.Fatal(b.PrimaryID())
	}
	var answer, tool, usage bool
	for _, e := range events {
		if e.Kind == state.EvChatBoss && !e.Msg.Pending && e.Msg.Text == "I found a failing check." {
			answer = true
		}
		if e.Kind == state.EvTool && e.ToolState == "error" && e.ToolOutput == "FAIL: useful error" {
			tool = true
		}
		if e.Kind == state.EvUsage && e.TokensIn == 20 {
			usage = true
		}
	}
	if !answer || !tool || !usage {
		t.Fatalf("event mapping answer=%v tool=%v usage=%v", answer, tool, usage)
	}
	if err := b.SendAgent("Plan only", "plan"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(log)
	args := string(raw)
	if !strings.Contains(args, "resume\n--json\n--skip-git-repo-check\nthread-first\n-") || !strings.Contains(args, "sandbox_mode=\"read-only\"") {
		t.Fatalf("resume/plan args: %s", args)
	}
	if strings.Contains(args, "--dangerously") {
		t.Fatal("normal sends bypassed sandbox")
	}
}

// TestCodexPrimaryLearnedOnceOnThreadStarted covers the prompt-persist seam
// (state.EvPrimaryLearned): thread.started must emit it exactly once, even
// across a second send that resumes the SAME thread and repeats the exact
// same thread_id in its own thread.started line (a thread spans many
// turns — the second must not re-fire).
func TestCodexPrimaryLearnedOnceOnThreadStarted(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-fake")
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"type":"thread.started","thread_id":"thread-durable"}
{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"ok"}}
{"type":"turn.completed"}
JSON
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var events []state.Event
	if err := b.Start(func(e state.Event) { events = append(events, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.Send("first"); err != nil {
		t.Fatal(err)
	}
	if err := b.Send("second"); err != nil { // resumes the same thread
		t.Fatal(err)
	}
	count, got := 0, ""
	for _, e := range events {
		if e.Kind == state.EvPrimaryLearned {
			count++
			got = e.PrimaryID
		}
	}
	if count != 1 {
		t.Fatalf("EvPrimaryLearned fired %d times, want 1 (same thread id repeated across turns)", count)
	}
	if got != "thread-durable" {
		t.Fatalf("EvPrimaryLearned.PrimaryID = %q, want %q", got, "thread-durable")
	}
	if b.PrimaryID() != "thread-durable" {
		t.Fatalf("PrimaryID() = %q, want %q", b.PrimaryID(), "thread-durable")
	}
}

// TestCodexPrimaryLearnedFiresAgainOnGenuinelyNewThread covers the other
// half of the dedup: a DIFFERENT thread id (e.g. after /new mints a fresh
// thread) must still emit its own EvPrimaryLearned rather than latching
// silent after the first turn ever seen.
func TestCodexPrimaryLearnedFiresAgainOnGenuinelyNewThread(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-fake")
	args := filepath.Join(dir, "args")
	t.Setenv("CODEX_TEST_ARGS", args)
	script := `#!/bin/sh
printf '%s\n' "$@" >> "$CODEX_TEST_ARGS"
cat >/dev/null
n=$(grep -c '^exec$' "$CODEX_TEST_ARGS")
cat <<JSON
{"type":"thread.started","thread_id":"thread-$n"}
{"type":"turn.completed"}
JSON
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var events []state.Event
	if err := b.Start(func(e state.Event) { events = append(events, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.Send("first"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.NewOffice(); err != nil { // clears b.primary -> a fresh thread next turn
		t.Fatal(err)
	}
	if err := b.Send("second"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range events {
		if e.Kind == state.EvPrimaryLearned {
			ids = append(ids, e.PrimaryID)
		}
	}
	if len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("EvPrimaryLearned ids = %v, want two distinct ids (one per genuinely new thread)", ids)
	}
}

// TestCodexNoPrimaryLearnedWithoutThreadStarted covers a turn whose stdout
// never carries thread.started (e.g. it fails before Codex assigns one):
// PrimaryID() and the event stream must both stay exactly as they started.
func TestCodexNoPrimaryLearnedWithoutThreadStarted(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-fake")
	script := `#!/bin/sh
cat >/dev/null
cat <<'JSON'
{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"ok, no thread id here"}}
{"type":"turn.completed"}
JSON
`
	if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var events []state.Event
	if err := b.Start(func(e state.Event) { events = append(events, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	if err := b.Send("hello"); err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Kind == state.EvPrimaryLearned {
			t.Fatalf("unexpected EvPrimaryLearned without a thread.started line: %+v", e)
		}
	}
	if b.PrimaryID() != "" {
		t.Fatalf("PrimaryID() = %q, want empty (never learned)", b.PrimaryID())
	}
}

func TestCodexStopSettlesRunningTurn(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex-fake")
	os.WriteFile(bin, []byte("#!/bin/sh\ncat >/dev/null\nexec sleep 60\n"), 0700)
	b := NewCodex(bin, dir, nil).(*codexBackend)
	var mu sync.Mutex
	pending := false
	if err := b.Start(func(e state.Event) {
		if e.Msg.Pending {
			mu.Lock()
			pending = true
			mu.Unlock()
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	done := make(chan error, 1)
	go func() { done <- b.Send("wait") }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		ready := pending
		mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("turn did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := b.NewOffice(); err == nil {
		t.Fatal("replaced a busy Codex session")
	}
	if err := b.AbortSessions(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("abort did not settle")
	}
}
