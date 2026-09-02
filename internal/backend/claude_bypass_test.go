package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeBypassOfficeReplacementSpawns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		bypass bool
		base   string
	}{
		{"on", true, "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions"},
		{"off", false, "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --permission-prompt-tool stdio"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub, argvlog, _, _ := claudeSwapStubFiles(t)
			b := newClaudeBackend(stub, t.TempDir(), nil)
			if err := b.SetBypassPermissions(tc.bypass); err != nil {
				t.Fatalf("pre-Start SetBypassPermissions(%v): %v", tc.bypass, err)
			}
			if err := b.Start((&claudeEventLog{}).emit); err != nil {
				t.Fatalf("Start: %v", err)
			}
			defer func() { _ = b.Stop() }()
			if _, err := b.NewOffice(); err != nil {
				t.Fatalf("NewOffice: %v", err)
			}
			if err := b.SwapPrimary("sess-saved-bypass"); err != nil {
				t.Fatalf("SwapPrimary: %v", err)
			}

			argv := claudeArgvLines(t, argvlog, 3)
			want := []string{tc.base, tc.base, tc.base + " --resume sess-saved-bypass"}
			if strings.Join(argv, "\n") != strings.Join(want, "\n") {
				t.Fatalf("replacement argv drifted:\n got %q\nwant %q", argv, want)
			}
		})
	}
}

func TestClaudeBypassReconnectMCPRespawn(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
while IFS= read -r line; do
  case "$line" in
    *'"subtype":"initialize"'*)
`+claudeStubPreambleSh()+`      ;;
  esac
done
`)
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start((&claudeEventLog{}).emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	claudeWait(t, "the init pin before reconnect", 3*time.Second, func() bool { return b.PrimaryID() == "sess-sh-1" })
	if err := b.ReconnectMCP("memo"); err != nil {
		t.Fatalf("ReconnectMCP: %v", err)
	}
	claudeWait(t, "the reconnect death latch", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})
	if err := b.Send("reconnect now"); err != nil {
		t.Fatalf("Send after ReconnectMCP: %v", err)
	}
	argv := claudeArgvLines(t, argvlog, 2)
	want := "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions --resume sess-sh-1"
	if argv[1] != want {
		t.Fatalf("ReconnectMCP respawn argv drifted:\n got %q\nwant %q", argv[1], want)
	}
}

func TestClaudeBypassDeathRespawnAndLifecycle(t *testing.T) {
	argvlog := filepath.Join(t.TempDir(), "argv.log")
	stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
while IFS= read -r line; do
  case "$line" in
    *'"subtype":"initialize"'*)
`+claudeStubPreambleSh()+`      ;;
    *please-die-now*) exit 0 ;;
  esac
done
`)
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start((&claudeEventLog{}).emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if err := b.SetBypassPermissions(false); err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Start SetBypassPermissions = %v, want respawn-required error", err)
	}
	claudeWait(t, "the init pin before death", 3*time.Second, func() bool { return b.PrimaryID() == "sess-sh-1" })
	if err := b.Send("please-die-now"); err != nil {
		t.Fatalf("Send death trigger: %v", err)
	}
	claudeWait(t, "the death latch", 3*time.Second, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.died
	})
	if err := b.Send("respawn"); err != nil {
		t.Fatalf("Send respawn trigger: %v", err)
	}
	argv := claudeArgvLines(t, argvlog, 2)
	want := "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions --resume sess-sh-1"
	if argv[1] != want {
		t.Fatalf("death respawn argv drifted:\n got %q\nwant %q", argv[1], want)
	}
	_ = b.Stop()
	if err := b.SetBypassPermissions(false); err == nil || !strings.Contains(err.Error(), "respawn required") {
		t.Fatalf("post-Stop SetBypassPermissions = %v, want respawn-required error", err)
	}
}

func TestClaudeBypassRejectedModeIsActionable(t *testing.T) {
	stub := claudeStubScript(t, `echo 'unknown option: --dangerously-skip-permissions' >&2
exit 2
`)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.SetBypassPermissions(true); err != nil {
		t.Fatalf("pre-Start SetBypassPermissions(true): %v", err)
	}
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()
	claudeWait(t, "the actionable bypass rejection", 3*time.Second, func() bool {
		return log.hasStatusContaining("CLI rejected --dangerously-skip-permissions")
	})
	if !log.hasStatusContaining("upgrade Claude Code or turn bypass off and respawn") {
		t.Fatalf("the rejection must include recovery instructions: %v", log.snapshot())
	}
}

func TestClaudeBypassSpawnArgvOnAndOff(t *testing.T) {
	for _, tc := range []struct {
		name   string
		bypass bool
		want   string
	}{
		{"on", true, "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --dangerously-skip-permissions"},
		{"off", false, "-p --input-format stream-json --output-format stream-json --verbose --include-partial-messages --permission-prompt-tool stdio"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			argvlog := filepath.Join(t.TempDir(), "argv.log")
			stub := claudeStubScript(t, `printf '%s\n' "$*" >> `+argvlog+`
while IFS= read -r line; do :; done
`)
			b := newClaudeBackend(stub, t.TempDir(), nil)
			if err := b.SetBypassPermissions(tc.bypass); err != nil {
				t.Fatal(err)
			}
			if err := b.Start((&claudeEventLog{}).emit); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = b.Stop() }()
			claudeWait(t, "the argv record", 3*time.Second, func() bool {
				bits, err := os.ReadFile(argvlog)
				return err == nil && strings.TrimSpace(string(bits)) == tc.want
			})
		})
	}
}
