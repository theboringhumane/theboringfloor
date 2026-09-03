// claude_office_swap_test.go — the /btw + /done + /new seams on the claude
// backend (officeSpawnBackend.NewOffice, btwSwapBackend.SwapPrimary — the
// wave-95 opencode seams ported to the process-per-session world).
// NewOffice respawns a FRESH process (no --resume), clears the id latches
// (the next system/init re-pins primaryID), and RESETS the briefed latch
// so the browser-tool preamble re-rides the first Send; SwapPrimary
// respawns with `--resume <saved id>`, pins primaryID/resumeID/
// primaryOverride to the saved id, and KEEPS briefed (the resumed
// session's context retains the preamble contract). Both ride the
// teardown ladder on the old process (no died latch — a swap, not a
// crash) and mirror the opencode backend's fire/hire/status events.
package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/browsertools"
	"github.com/theboringhumane/theboringfloor/internal/chatcontext"
	"github.com/theboringhumane/theboringfloor/internal/plantools"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// claudeSwapStubBody — one stub script for both seams: records every
// spawn's argv (one line per process), numbers the spawns via a counter
// file (session_id "sess-sh-<n>"), records each process's exit via an
// EXIT trap, captures every stdin line, and answers each process's first
// user line with the real stream shape (init -> assistant -> result).
func claudeSwapStubBody(argvlog, capture, exitlog, counter string) string {
	return `printf '%s\n' "$*" >> "` + argvlog + `"
n=$(cat "` + counter + `" 2>/dev/null || echo 0)
n=$((n+1))
printf '%s\n' "$n" > "` + counter + `"
trap 'echo exit-$n >> "` + exitlog + `"' EXIT
` + claudeStubSh(claudeStubHookLines) + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      printf '%s\n' '{"type":"system","subtype":"init","cwd":"/tmp","session_id":"sess-sh-'$n'","model":"claude-test-1","mcp_servers":[],"claude_code_version":"2.1.246","uuid":"00000000-0000-4000-8000-00000000000'$n'"}'
      printf '%s\n' '{"type":"assistant","message":{"id":"msg-sw-'$n'","role":"assistant","content":[{"type":"text","text":"Roger."}]},"session_id":"sess-sh-'$n'","uuid":"msg-sw-'$n'","parent_tool_use_id":null}'
      printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"sess-sh-'$n'","uuid":"res-sw-'$n'","total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":1}}'
      ;;
  esac
done
`
}

// claudeSwapStubFiles — the four recording paths + the stub script, rooted
// in one temp dir.
func claudeSwapStubFiles(t *testing.T) (stub, argvlog, capture, exitlog string) {
	t.Helper()
	tmp := t.TempDir()
	argvlog = filepath.Join(tmp, "argv.log")
	capture = filepath.Join(tmp, "capture.log")
	exitlog = filepath.Join(tmp, "exit.log")
	counter := filepath.Join(tmp, "n")
	stub = claudeStubScript(t, claudeSwapStubBody(argvlog, capture, exitlog, counter))
	return stub, argvlog, capture, exitlog
}

// claudeArgvLines waits for the argv log to hold want spawn records, then
// returns them (one "$*" line per spawned process).
func claudeArgvLines(t *testing.T, argvlog string, want int) []string {
	t.Helper()
	claudeWait(t, "the spawn argv records", 3*time.Second, func() bool {
		bits, err := os.ReadFile(argvlog)
		if err != nil || strings.TrimSpace(string(bits)) == "" {
			return false
		}
		return len(strings.Split(strings.TrimSpace(string(bits)), "\n")) == want
	})
	bits, err := os.ReadFile(argvlog)
	if err != nil {
		t.Fatalf("read argv log: %v", err)
	}
	return strings.Split(strings.TrimSpace(string(bits)), "\n")
}

// claudeWaitExit waits for process n's EXIT-trap record (the teardown
// ladder honored procWait, so the record lands deterministically — the
// poll only bounds fs visibility).
func claudeWaitExit(t *testing.T, exitlog string, n string) {
	t.Helper()
	claudeWait(t, "process "+n+"'s exit record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(exitlog)
		return err == nil && strings.Contains(string(bits), "exit-"+n)
	})
}

// TestClaudeNewOfficeFreshRespawn — /btw + /new: the replacement process
// spawns WITHOUT --resume, the old process exits on the teardown ladder
// (no died latch), the manager row fires/re-hires, the next system/init
// re-pins primaryID, and briefed RESETS (the preamble re-rides the fresh
// session's first Send — the trap a --resume respawn must never fall
// into, and the inverse here).
func TestClaudeNewOfficeFreshRespawn(t *testing.T) {
	stub, argvlog, capture, exitlog := claudeSwapStubFiles(t)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if err := b.Send("first"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "the first session's init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})
	b.mu.Lock()
	briefed := b.briefed
	b.mu.Unlock()
	if !briefed {
		t.Fatalf("briefed must latch after the first clean Send")
	}

	id, err := b.NewOffice()
	if err != nil {
		t.Fatalf("NewOffice: %v", err)
	}
	if id != "" {
		t.Fatalf("NewOffice = %q — claude's session id pins on the NEXT init; want \"\"", id)
	}
	if got := b.PrimaryID(); got != "" {
		t.Fatalf("PrimaryID = %q after NewOffice — the pin clears until the fresh init lands", got)
	}

	// the old process rode the teardown ladder (stdin close → exit →
	// trap) — a swap, not a crash: no died latch, no death status.
	claudeWaitExit(t, exitlog, "1")
	b.mu.Lock()
	died := b.died
	b.mu.Unlock()
	if died {
		t.Fatalf("a controlled swap must never latch died")
	}

	// the replacement spawn rides NO --resume (a FRESH session).
	argvLines := claudeArgvLines(t, argvlog, 2)
	t.Logf("spawn 1 argv: %s", argvLines[0])
	t.Logf("spawn 2 argv: %s", argvLines[1])
	if strings.Contains(argvLines[1], "--resume") {
		t.Fatalf("NewOffice's spawn must be FRESH (no --resume): %s", argvLines[1])
	}
	if !strings.Contains(argvLines[1], "--permission-prompt-tool stdio") {
		t.Fatalf("the fresh spawn keeps the boot's permission wiring: %s", argvLines[1])
	}

	// fire the old manager row, re-hire (id "" pre-init — the Start boot
	// convention), one status: the opencode NewOffice event shape.
	fireIdx, hireIdx, statusFresh := -1, -1, false
	for i, e := range log.snapshot() {
		if e.Kind == state.EvFire && e.EmployeeID == "sess-sh-1" {
			fireIdx = i
		}
		if fireIdx >= 0 && i > fireIdx && hireIdx < 0 && e.Kind == state.EvHire && e.Employee.Name == "manager" && e.Employee.ID == "" {
			hireIdx = i
		}
		if e.Kind == state.EvStatus && strings.Contains(e.Text, "new office session fresh") {
			statusFresh = true
		}
	}
	if fireIdx < 0 || hireIdx < 0 || !statusFresh {
		t.Fatalf("NewOffice must fire the old row, re-hire the manager, and status the fresh office (fire=%d hire=%d status=%v): %v", fireIdx, hireIdx, statusFresh, log.snapshot())
	}

	// briefed RESET: the preamble re-rides the fresh session's first Send
	// — and the fresh process got its OWN initialize declaration.
	if err := b.Send("second"); err != nil {
		t.Fatalf("Send after NewOffice: %v", err)
	}
	claudeWait(t, "the fresh session's init re-pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-2"
	})
	claudeWait(t, "both user lines captured", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	wantSecond := string(claudeUserLineFor(browsertools.PromptPreamble + "\n\n" + chatcontext.PromptPreamble + "\n\n" + plantools.PromptPreamble + "\n\nsecond"))
	if len(lines) != 2 || lines[1] != wantSecond || !strings.Contains(lines[1], "⟦open-browser:") {
		t.Fatalf("the preamble must re-ride the fresh session's first Send:\n got %v\nwant %q", lines, wantSecond)
	}
	initDecls := 0
	for _, ln := range claudeCaptureRaw(t, capture) {
		if strings.Contains(ln, `"subtype":"initialize"`) {
			initDecls++
		}
	}
	if initDecls != 2 {
		t.Fatalf("each process attach re-declares the dialog kinds — want 2 initialize lines, got %d: %v", initDecls, claudeCaptureRaw(t, capture))
	}
}

// TestClaudeSwapPrimaryResumesSaved — /done: the replacement process rides
// `--resume <saved id>`, primaryID pins to the saved id IMMEDIATELY (and
// survives the resumed process's own init — the override latch wins over
// the wire), and briefed KEEPS its latch (the resumed session's context
// retains the preamble — it must NOT ride again).
func TestClaudeSwapPrimaryResumesSaved(t *testing.T) {
	stub, argvlog, capture, exitlog := claudeSwapStubFiles(t)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if err := b.Send("first"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	claudeWait(t, "the first session's init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-1"
	})

	if err := b.SwapPrimary("sess-saved-9"); err != nil {
		t.Fatalf("SwapPrimary: %v", err)
	}
	if got := b.PrimaryID(); got != "sess-saved-9" {
		t.Fatalf("PrimaryID = %q after SwapPrimary — want the saved pin sess-saved-9", got)
	}

	// the old process rode the teardown ladder.
	claudeWaitExit(t, exitlog, "1")
	b.mu.Lock()
	died := b.died
	b.mu.Unlock()
	if died {
		t.Fatalf("a controlled swap must never latch died")
	}

	// the replacement spawn rides --resume sess-saved-9.
	argvLines := claudeArgvLines(t, argvlog, 2)
	t.Logf("spawn 1 argv: %s", argvLines[0])
	t.Logf("spawn 2 argv: %s", argvLines[1])
	if !strings.Contains(argvLines[1], "--resume sess-saved-9") {
		t.Fatalf("SwapPrimary's spawn must resume the saved id: %s", argvLines[1])
	}

	// fire/hire/status mirror opencode's SwapPrimary.
	fireIdx, hireIdx, statusSwap := -1, -1, false
	for i, e := range log.snapshot() {
		if e.Kind == state.EvFire && e.EmployeeID == "sess-sh-1" {
			fireIdx = i
		}
		if fireIdx >= 0 && i > fireIdx && hireIdx < 0 && e.Kind == state.EvHire && e.Employee.Name == "manager" && e.Employee.ID == "sess-saved-9" {
			hireIdx = i
		}
		if e.Kind == state.EvStatus && strings.Contains(e.Text, "primary session swapped to sess-saved-9") {
			statusSwap = true
		}
	}
	if fireIdx < 0 || hireIdx < 0 || !statusSwap {
		t.Fatalf("SwapPrimary must fire the old row, hire the pinned manager, and status the swap (fire=%d hire=%d status=%v): %v", fireIdx, hireIdx, statusSwap, log.snapshot())
	}

	// briefed KEPT: the next Send rides NO preamble; the resume's own
	// init (sess-sh-2 on the wire) cannot re-pin over the override.
	if err := b.Send("third"); err != nil {
		t.Fatalf("Send after SwapPrimary: %v", err)
	}
	claudeWait(t, "the resumed session's own init", 3*time.Second, func() bool {
		return log.hasStatusContaining("[claude] init model=claude-test-1 session=sess-sh-2")
	})
	if got := b.PrimaryID(); got != "sess-saved-9" {
		t.Fatalf("the override must win over the resume's wire id, got %q", got)
	}
	b.mu.Lock()
	resume := b.resumeID
	override := b.primaryOverride
	b.mu.Unlock()
	if resume != "sess-saved-9" || override != "sess-saved-9" {
		t.Fatalf("the resume + override latches must stay the saved id (resume=%q override=%q)", resume, override)
	}
	claudeWait(t, "both user lines captured", 2*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	wantThird := string(claudeUserLineFor("third"))
	if len(lines) != 2 || lines[1] != wantThird {
		t.Fatalf("a resumed session keeps the briefed latch — NO preamble on the next Send:\n got %v\nwant %q", lines, wantThird)
	}
}

// TestClaudeNewOfficeBeforeInit — /btw on a never-Sent backend: no id was
// ever pinned, so NO fire row rides, the respawn is still fresh, and the
// swap never invents an id.
func TestClaudeNewOfficeBeforeInit(t *testing.T) {
	stub, argvlog, _, exitlog := claudeSwapStubFiles(t)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	// Start never waits on the stub: let process 1 finish its spawn
	// record (the counter write) so the swap's spawn numbers honestly.
	counter := filepath.Join(filepath.Dir(argvlog), "n")
	claudeWait(t, "process 1's spawn record", 3*time.Second, func() bool {
		bits, err := os.ReadFile(counter)
		return err == nil && strings.TrimSpace(string(bits)) == "1"
	})

	id, err := b.NewOffice()
	if err != nil {
		t.Fatalf("NewOffice: %v", err)
	}
	if id != "" {
		t.Fatalf("NewOffice = %q pre-init — want \"\"", id)
	}
	claudeWaitExit(t, exitlog, "1")
	argvLines := claudeArgvLines(t, argvlog, 2)
	if strings.Contains(argvLines[1], "--resume") {
		t.Fatalf("NewOffice's spawn must be FRESH (no --resume): %s", argvLines[1])
	}

	managerHires := 0
	for _, e := range log.snapshot() {
		if e.Kind == state.EvFire {
			t.Fatalf("no id was ever pinned — no fire row may ride: %+v", e)
		}
		if e.Kind == state.EvHire && e.Employee.Name == "manager" {
			managerHires++
		}
	}
	if managerHires != 2 {
		t.Fatalf("the manager re-seats with the fresh office (boot hire + swap hire), got %d: %v", managerHires, log.snapshot())
	}

	// the fresh session still pins honestly when its init finally lands.
	if err := b.Send("hello"); err != nil {
		t.Fatalf("Send after NewOffice: %v", err)
	}
	claudeWait(t, "the fresh session's init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-sh-2"
	})
}
