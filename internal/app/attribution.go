package app

// Majdoor commit-attribution hook ensure — the write side of brain.json's
// top-level "attribution" knob, wired once at boot.
//
// Why internal/app and NOT internal/gitx: gitx.go's package header
// declares that package read-only git status/diff data for the Git panel
// ("strictly read-only (rev-parse / status / diff)"), and installing or
// removing a hook file is a write — so the ensure lives here next to the
// boot wiring. The trailer constant and the message-side idempotent
// helper stay in gitx/attribution.go (untouched).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EnsureMajdoorHook result statuses — short, stable strings; the boot
// wiring logs the returned one verbatim on every launch.
const (
	hookStatusInstalled      = "installed"       // no commit-msg hook existed; ours is in (0755)
	hookStatusPresent        = "present"         // a byte-identical hook was already installed — no-op
	hookStatusSkippedForeign = "skipped-foreign" // a DIFFERENT hook exists — never overwritten/chained/removed
	hookStatusRemoved        = "removed"         // attribution off: our own byte-identical hook uninstalled
	hookStatusAbsent         = "absent"          // attribution off and no hook present — nothing to do
	hookStatusNoRepo         = "no-repo"         // dir is not inside a git work tree — clean no-op
)

// hookGitTimeout bounds the two discovery rev-parses — boot must never
// wait on a wedged git.
const hookGitTimeout = 2 * time.Second

// majdoorCommitMsgHook is the EXACT body installed as the repo's
// commit-msg hook: byte-identical to scripts/majdoor-commit-msg-hook.sh
// (TestMajdoorHookMatchesCanonicalScript pins the sync — update both or
// neither). The hook appends exactly one MajdoorTrailer to every commit
// message, skips an already-stamped message, and NEVER rejects a commit.
const majdoorCommitMsgHook = `#!/bin/sh
# theboringfloor — TheBoringMajdoor attribution hook (git commit-msg).
#
# Auto-installed by the office into the repo it boots in when attribution is
# on (the default); scripts/install-majdoor-hook.sh covers repos the office
# never boots in. Either way git invokes it as:
#   .git/hooks/commit-msg <message-file>
#
# Every commit gets exactly one
#   Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
# trailer: skipped when a trailer carrying that email is already present
# (matched case-insensitively), joined onto an existing trailer block with no
# blank line, otherwise paragraph-broken with one. Running it twice changes
# nothing. POSIX sh; grep/sed are the only text tools.
set -eu

MSG_FILE="${1:?usage: commit-msg <message-file>}"

TRAILER="Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>"

# Already stamped? ANY trailer line (Token: value) carrying our email, in any
# case, means this message is done — never stamp twice.
if grep -qiE '^[A-Za-z][A-Za-z0-9-]*[[:space:]]*:.*themajdoor@theboring\.name' "$MSG_FILE"; then
    exit 0
fi

TMP_MSG="${MSG_FILE}.majdoor.$$"
trap 'rm -f "$TMP_MSG"' EXIT

# Strip trailing blank lines first, so the trailer hugs the end of the
# message instead of drifting off it. (Classic portable sed: trailing blanks
# accumulate in the pattern space and are dropped only at EOF; interior
# blank lines survive untouched.)
sed -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$MSG_FILE" > "$TMP_MSG"

# Trailer-block etiquette: when the message already ENDS in a trailer block
# (its last line is "Token: value") ours joins that block directly; anything
# else gets one blank line first, opening a fresh trailer paragraph.
if [ -s "$TMP_MSG" ] \
    && sed -n '$p' "$TMP_MSG" | grep -qE '^[A-Za-z][A-Za-z0-9-]*[[:space:]]*:'; then
    printf '%s\n' "$TRAILER" >> "$TMP_MSG"
else
    printf '\n%s\n' "$TRAILER" >> "$TMP_MSG"
fi

mv -f "$TMP_MSG" "$MSG_FILE"
`

// EnsureMajdoorHook makes the repo containing dir carry the office's
// majdoor commit-msg hook (enabled) or removes it again (disabled), and
// reports what it did as one of the hookStatus* strings.
//
// Boot-safety contract (the office must boot fine anywhere):
//   - dir outside a git work tree (or git itself missing/wedged on the
//     discovery rev-parse) → hookStatusNoRepo, nil error;
//   - an existing hook that is NOT byte-identical to ours is NEVER
//     overwritten, chained, or removed → hookStatusSkippedForeign, nil;
//   - the hooks dir is resolved via `git rev-parse --git-path hooks` run
//     from the work-tree root, so core.hooksPath (relative to the root or
//     absolute) and linked worktrees land in the right place.
func EnsureMajdoorHook(dir string, enabled bool) (status string, err error) {
	root, err := hookGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		// Any discovery failure — not a repo, bare repo, git absent — is
		// a clean no-op: boot resilience beats diagnostics here, and the
		// boot wiring still logs the status.
		return hookStatusNoRepo, nil
	}
	hooksDir, err := hookGit(root, "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", fmt.Errorf("attribution: resolve hooks dir in %s: %w", root, err)
	}
	if !filepath.IsAbs(hooksDir) {
		hooksDir = filepath.Join(root, hooksDir)
	}
	hookPath := filepath.Join(hooksDir, "commit-msg")

	existing, rerr := os.ReadFile(hookPath)
	if rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		return "", fmt.Errorf("attribution: read %s: %w", hookPath, rerr)
	}
	have := rerr == nil
	ours := have && string(existing) == majdoorCommitMsgHook

	if enabled {
		switch {
		case !have:
			if err := os.MkdirAll(hooksDir, 0o755); err != nil {
				return "", fmt.Errorf("attribution: create %s: %w", hooksDir, err)
			}
			if err := os.WriteFile(hookPath, []byte(majdoorCommitMsgHook), 0o755); err != nil {
				return "", fmt.Errorf("attribution: install %s: %w", hookPath, err)
			}
			return hookStatusInstalled, nil
		case ours:
			return hookStatusPresent, nil
		default:
			return hookStatusSkippedForeign, nil
		}
	}

	switch {
	case !have:
		return hookStatusAbsent, nil
	case ours:
		if err := os.Remove(hookPath); err != nil {
			return "", fmt.Errorf("attribution: remove %s: %w", hookPath, err)
		}
		return hookStatusRemoved, nil
	default:
		return hookStatusSkippedForeign, nil
	}
}

// hookGit runs one bounded git call in dir and returns trimmed stdout;
// stderr wording is wrapped into the error (gitx's execGit pattern).
func hookGit(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), hookGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), msg, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
