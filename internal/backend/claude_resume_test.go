// claude_resume_test.go — the death-watch / --resume pin: kill the stub
// mid-turn and the NEXT Send respawns the process with `--resume <uuid>`,
// where the uuid is ALWAYS the one system/init pinned (never anything
// derived from a user Send) — all without a single chat line before it.
package backend

import (
	"bufio"
	"github.com/theboringhumane/theboringoffice/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeResumePinAfterMidTurnKill(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	capture := filepath.Join(t.TempDir(), "capture.log")
	// the stub records every invocation's argv; per user line it answers
	// one Roger turn so a respawned send STILL completes (the placeholder
	// drain behavior under a resume).
	stubBody := `printf '%s\n' "$*" >> ` + argvlog + `
` + claudeStubSh(claudeStubHookLines) + `printf '%s\n' '{"type":"system","subtype":"init","cwd":"/tmp","session_id":"sess-uuid-42","model":"claude-test-1","mcp_servers":[],"claude_code_version":"2.1.246","uuid":"00000000-0000-4000-8000-000000000042"}'
n=0
while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
  case "$line" in
    *'"type":"user"'*)
      n=$((n+1))
      printf '%s\n' '{"type":"assistant","message":{"id":"msg-r'$n'","role":"assistant","content":[{"type":"text","text":"Roger that."}]},"session_id":"sess-uuid-42","uuid":"msg-r'$n'","parent_tool_use_id":null}'
      printf '%s\n' '{"type":"result","subtype":"success","is_error":false,"session_id":"sess-uuid-42","uuid":"res-r'$n'","total_cost_usd":0,"usage":{"input_tokens":1,"output_tokens":1}}'
      ;;
  esac
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	// init pins WHENEVER the reader maps it (Start never waits on init).
	claudeWait(t, "the init pin", 3*time.Second, func() bool {
		return b.PrimaryID() == "sess-uuid-42"
	})

	// a mid-turn kill: send, then murder the child before any reply
	// (the stub's scripted reply is fast — kill right after the write
	// landed, mid-flight by construction).
	if err := b.Send("kill me mid-turn"); err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	b.mu.Lock()
	proc := b.proc
	b.mu.Unlock()
	if proc == nil {
		t.Fatalf("no live child to kill")
	}
	if err := proc.Process.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	claudeWait(t, "the death-watch latch (died=true)", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})

	// the NEXT Send respawns with --resume <init uuid>
	if err := b.Send("and we're back"); err != nil {
		t.Fatalf("Send 2 (the respawning one): %v", err)
	}
	claudeWait(t, "the second argv recorded", 3*time.Second, func() bool {
		f, err := os.Open(argvlog)
		if err != nil {
			return false
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		n := 0
		for sc.Scan() {
			n++
		}
		return n >= 2
	})
	f, err := os.Open(argvlog)
	if err != nil {
		t.Fatalf("read argv log: %v", err)
	}
	defer f.Close()
	var argvLines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		argvLines = append(argvLines, sc.Text())
	}
	resumeFound := ""
	for _, l := range argvLines {
		for i, w := range strings.Fields(l) {
			if w == "--resume" && i+1 < len(strings.Fields(l)) {
				resumeFound = strings.Fields(l)[i+1]
			}
		}
	}
	if resumeFound == "" {
		t.Fatalf("no --resume flag in respawn argv: %v", argvLines)
	}
	if resumeFound != "sess-uuid-42" {
		t.Fatalf("--resume pin = %q, want the system/init uuid sess-uuid-42 (never a Send-derived value)", resumeFound)
	}

	// the respawned turn actually completes against the new process
	claudeWait(t, "the resumed turn's Roger reply", 3*time.Second, func() bool {
		for _, e := range log.snapshot() {
			if e.Kind == state.EvChatBoss && e.Msg.Text == "Roger that." {
				return true
			}
		}
		return false
	})
	if !log.hasStatusContaining("respawned with --resume") {
		t.Fatalf("the respawn note must name the pin once")
	}
}
