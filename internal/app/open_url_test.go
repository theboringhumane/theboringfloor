// System browser tests, including migration from retired environment flags.
package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/panels"
)

func TestResolveBrowserToolIgnoresRetiredEnvironment(t *testing.T) {
	for _, host := range []string{"ghostty", "kitty", "WezTerm", "iTerm.app", ""} {
		for _, flag := range []string{"", "0", "1", "true"} {
			env := func(key string) string {
				if key == "TERM_PROGRAM" {
					return host
				}
				return flag
			}
			probe := func(string) (string, error) { t.Fatal("probed removed executable"); return "", nil }
			if got := panels.ResolveOpenToolFrom(env, probe); got != panels.OpenToolSystemOpen {
				t.Fatal(got)
			}
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
