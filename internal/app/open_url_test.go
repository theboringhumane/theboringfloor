// open_url_test.go — the OPTIONAL terminal-browser lane suite
// (open_url.go's app resolver + panels' candidate cascade):
//
//	(a) the resolver MATRIX: LookPath found/missing × terminal
//	    match/mismatch (ghostty/wezterm/kitty-marker match; iTerm2 a
//	    dead-end; tmux a default miss) — driven PURE through the exported
//	    panels.ResolveOpenToolFrom (env + counting probe injected), the
//	    probe's call count pinned per row (a host-miss row must never
//	    even LOOK for the binary);
//	(b) the KILL SWITCH: THEFLOOR_NO_TERMINAL_BROWSER=1 consults
//	    the env FIRST — a present binary on a matched terminal still
//	    resolves system-open and the probe's call count is ZERO (the
//	    binary is never even probed, recorded on the lookup counter);
//	(c) the CASCADE: a non-zero-exit terminal-browser (fixture fake on
//	    PATH) hands the SAME target to the system opener immediately —
//	    ONE attempt logged per leg (never a retry), the verdict nil, and
//	    the resolution STILL prefers terminal-browser (the fallback is a
//	    runtime event, not a resolver flip);
//	(d) the happy lane: an exit-0 candidate is the whole open (the
//	    system opener's log stays ABSENT — no cascade ever fired);
//	(e) exec FRAMING: `terminal-browser open <target>` — a file target
//	    runs with cmd.Dir = its parent dir, a URL target leaves Dir
//	    empty (the child inherits this process's cwd), pinned through
//	    the fake's logged pwd.
//
// Real-fixture legs ((c)-(e)) run the REAL default chain over POSIX fake
// scripts logging to $FAKE_OPEN_LOG_DIR — PATH pins "<fixture>:<orig>"
// (fixture first SHADOWS any real terminal-browser/open on the host) and
// every terminal env var is pinned (serial-suite hygiene: never
// t.Parallel with env swaps; t.Setenv restores everything).
package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/panels"
)

// TestResolveBrowserToolMatrix — the selection matrix driven PURE
// through panels' exported core (the app's resolver is a delegation):
// LookPath found/missing × terminal match/mismatch — six base rows plus
// the kitty-marker and wezterm named-match rows — with the probe's call
// count pinned per row.
func TestResolveBrowserToolMatrix(t *testing.T) {
	const tb = panels.OpenToolTerminalBrowser
	const sys = panels.OpenToolSystemOpen
	rows := []struct {
		name       string
		env        map[string]string
		probeOK    bool // whether LookPath finds "terminal-browser"
		want       panels.OpenTool
		wantProbes int
	}{
		{"lookup-found + matched terminal (ghostty)", map[string]string{"TERM_PROGRAM": "ghostty"}, true, tb, 1},
		{"lookup-failed + matched terminal (ghostty)", map[string]string{"TERM_PROGRAM": "ghostty"}, false, sys, 1},
		{"lookup-found + matched terminal (wezterm)", map[string]string{"TERM_PROGRAM": "WezTerm"}, true, tb, 1},
		{"lookup-failed + matched terminal (wezterm)", map[string]string{"TERM_PROGRAM": "WezTerm"}, false, sys, 1},
		{"lookup-found + terminal mismatch (iTerm.app)", map[string]string{"TERM_PROGRAM": "iTerm.app"}, true, sys, 0},
		{"lookup-failed + terminal mismatch (iTerm.app)", map[string]string{"TERM_PROGRAM": "iTerm.app"}, false, sys, 0},
		{"lookup-found + tmux (default miss)", map[string]string{"TMUX": "/tmp/tmux-1000/default,1,0", "TERM_PROGRAM": "ghostty"}, true, sys, 0},
		{"lookup-failed + tmux (default miss)", map[string]string{"TMUX": "/tmp/tmux-1000/default,1,0", "TERM_PROGRAM": "ghostty"}, false, sys, 0},
		{"lookup-found + kitty's own marker", map[string]string{"KITTY_WINDOW_ID": "7", "TERM_PROGRAM": "whatever"}, true, tb, 1},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			probes := 0
			probe := func(name string) (string, error) {
				if name != "terminal-browser" {
					t.Fatalf("the resolver only ever probes for terminal-browser, got %q", name)
				}
				probes++
				if row.probeOK {
					return "/usr/local/bin/terminal-browser", nil
				}
				return "", errors.New("not on PATH")
			}
			got := panels.ResolveOpenToolFrom(func(k string) string { return row.env[k] }, probe)
			if got != row.want {
				t.Fatalf("resolve = %v, want %v", got, row.want)
			}
			if probes != row.wantProbes {
				t.Fatalf("a host-miss row must NEVER probe the binary: probes=%d, want %d", probes, row.wantProbes)
			}
		})
	}
	if panels.OpenToolTerminalBrowser.String() != "terminal-browser" || panels.OpenToolSystemOpen.String() != "system-open" {
		t.Fatalf("the lane words: tb=%q sys=%q", panels.OpenToolTerminalBrowser, panels.OpenToolSystemOpen)
	}
}

// TestResolveBrowserToolKillSwitch — THEFLOOR_NO_TERMINAL_BROWSER=1
// consults the env FIRST: a present binary on a matched terminal still
// resolves system-open, and the lookup registry records ZERO probes (the
// candidate is never even looked at). "0"/set-to-anything-else do NOT
// toggle it (the MUTE-style "exactly \"1\"" rule).
func TestResolveBrowserToolKillSwitch(t *testing.T) {
	env := map[string]string{
		panels.TerminalBrowserOffEnv: "1",
		"TERM_PROGRAM":               "ghostty",
	}
	probes := 0
	probe := func(string) (string, error) { probes++; return "/usr/local/bin/terminal-browser", nil }
	if got := panels.ResolveOpenToolFrom(func(k string) string { return env[k] }, probe); got != panels.OpenToolSystemOpen {
		t.Fatalf("the kill-switch forces the system opener, got %v", got)
	}
	if probes != 0 {
		t.Fatalf("a switched-off lane is never even probed, got %d lookups", probes)
	}
	for _, v := range []string{"0", "true", ""} {
		env[panels.TerminalBrowserOffEnv] = v
		if got := panels.ResolveOpenToolFrom(func(k string) string { return env[k] }, probe); got != panels.OpenToolTerminalBrowser {
			t.Fatalf("kill-switch=%q does NOT toggle: %v", v, got)
		}
	}
}

// fakeBrowserScripts plants the POSIX fakes: a `terminal-browser` that
// logs "terminal-browser pwd=… args=…" and exits $FAKE_TB_EXIT (default
// 0), plus `open` and `xdg-open` logging "system-open …" and exiting 0.
// PATH is pinned "<fixture>:<orig>" (fixture FIRST — any real binary on
// the host is shadowed) and every resolver env var pinned. Returns the
// two log paths (a log ABSENT means its leg never ran).
func fakeBrowserScripts(t *testing.T) (tbLog, sysLog string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv(panels.TerminalBrowserOffEnv, "")
	t.Setenv("FAKE_OPEN_LOG_DIR", dir)
	tbLog, sysLog = filepath.Join(dir, "tb.log"), filepath.Join(dir, "system.log")
	scripts := map[string]string{
		"terminal-browser": "#!/bin/sh\n" +
			"printf 'terminal-browser pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/tb.log\"\n" +
			"exit \"${FAKE_TB_EXIT:-0}\"\n",
		"open": "#!/bin/sh\n" +
			"printf 'system-open pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/system.log\"\n" +
			"exit 0\n",
		"xdg-open": "#!/bin/sh\n" +
			"printf 'system-open pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/system.log\"\n" +
			"exit 0\n",
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return tbLog, sysLog
}

// openLogLines reads a fake's capture file (absent == the leg never ran).
func openLogLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	return lines
}

// pwdMatches — a fake's logged "pwd=<dir>" names the WANTED dir: the raw
// spelling AND the symlink-resolved one both count (darwin's getcwd keeps
// the /var symlink opaque for a chdir'd child on some hosts, physical on
// others — both spellings name the same directory).
func pwdMatches(logLine, want string) bool {
	if strings.Contains(logLine, "pwd="+want) {
		return true
	}
	if ev, err := filepath.EvalSymlinks(want); err == nil && ev != want {
		return strings.Contains(logLine, "pwd="+ev)
	}
	return false
}

// TestResolveBrowserToolDelegationPin — app.ResolveBrowserTool delegates
// to panels.ResolveOpenTool on the LIVE env: over the fixture env the
// result is the candidate lane itself.
func TestResolveBrowserToolDelegationPin(t *testing.T) {
	fakeBrowserScripts(t)
	if got := ResolveBrowserTool(); got != BrowserToolTerminalBrowser {
		t.Fatalf("the app resolver delegates the live chain: got %v", got)
	}
	if ResolveBrowserTool().String() != "terminal-browser" {
		t.Fatalf("the lane word round-trips: %q", ResolveBrowserTool())
	}
	t.Setenv(panels.TerminalBrowserOffEnv, "1")
	if got := ResolveBrowserTool(); got != BrowserToolSystemOpen {
		t.Fatalf("the kill-switch flips the live resolver mid-session: got %v", got)
	}
}

// TestOpenURLCascadeFallsToSystemOpen — a NON-ZERO-exit terminal-browser
// hands the SAME target to the system opener IMMEDIATELY: ONE candidate
// attempt logged (never a retry), ONE system-open capture, the verdict
// nil (never fatal), and the resolver still PREFERS terminal-browser
// (the cascade is the runtime leg beneath the preference).
func TestOpenURLCascadeFallsToSystemOpen(t *testing.T) {
	tbLog, sysLog := fakeBrowserScripts(t)
	t.Setenv("FAKE_TB_EXIT", "1")
	target := panels.LinkTarget{Kind: panels.LinkURL, Value: "https://opencode.ai/docs", Name: "opencode.ai/docs"}
	if ResolveBrowserTool() != BrowserToolTerminalBrowser {
		t.Fatal("the resolver PREFERS the candidate (the fallback is runtime)")
	}
	if err := panels.OpenInBrowser(target); err != nil {
		t.Fatalf("the cascade absorbs the candidate failure — verdict stays clean: %v", err)
	}
	tb, sys := openLogLines(t, tbLog), openLogLines(t, sysLog)
	if len(tb) != 1 || !strings.Contains(tb[0], "terminal-browser pwd=") || !strings.HasSuffix(tb[0], "args=open https://opencode.ai/docs") {
		t.Fatalf("EXACTLY ONE terminal-browser attempt (no double-call retry) with the URL: %v", tb)
	}
	if len(sys) != 1 || !strings.HasSuffix(sys[0], "https://opencode.ai/docs") {
		t.Fatalf("the system opener received the SAME target exactly once: %v", sys)
	}
}

// TestOpenURLTerminalBrowserSuccessLane — an exit-0 candidate IS the
// whole open: the system opener's log stays absent (the cascade never
// fired) and the verdict is clean.
func TestOpenURLTerminalBrowserSuccessLane(t *testing.T) {
	tbLog, sysLog := fakeBrowserScripts(t)
	if err := panels.OpenInBrowser(panels.LinkTarget{Kind: panels.LinkURL, Value: "https://opencode.ai/docs", Name: "opencode.ai/docs"}); err != nil {
		t.Fatalf("a clean candidate open: %v", err)
	}
	if tb := openLogLines(t, tbLog); len(tb) != 1 || !strings.HasSuffix(tb[0], "args=open https://opencode.ai/docs") {
		t.Fatalf("the candidate captured the URL exactly once: %v", tb)
	}
	if sys := openLogLines(t, sysLog); len(sys) != 0 {
		t.Fatalf("a successful candidate never cascades to the system opener: %v", sys)
	}
}

// TestOpenURLExecFraming — `terminal-browser open <target>`: a FILE
// target runs with cmd.Dir = its parent dir (logged pwd pins it), a URL
// target leaves Dir empty (the child inherits this process's cwd).
func TestOpenURLExecFraming(t *testing.T) {
	tbLog, _ := fakeBrowserScripts(t)
	f := filepath.Join(t.TempDir(), "trace.log")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := panels.OpenInBrowser(panels.LinkTarget{Kind: panels.LinkFile, Value: f, Name: "trace.log"}); err != nil {
		t.Fatalf("the file target opens through the candidate: %v", err)
	}
	tb := openLogLines(t, tbLog)
	if len(tb) != 1 || !pwdMatches(tb[0], filepath.Dir(f)) || !strings.HasSuffix(tb[0], "args=open "+f) {
		t.Fatalf("a FILE target runs with cmd.Dir = the parent dir: %v", tb)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := panels.OpenInBrowser(panels.LinkTarget{Kind: panels.LinkURL, Value: "https://opencode.ai/docs"}); err != nil {
		t.Fatalf("the URL target opens through the candidate: %v", err)
	}
	tb = openLogLines(t, tbLog)
	if len(tb) != 2 || !pwdMatches(tb[1], cwd) {
		t.Fatalf("a URL target leaves Dir empty (inherits the process cwd %q): %v", cwd, tb)
	}
}
