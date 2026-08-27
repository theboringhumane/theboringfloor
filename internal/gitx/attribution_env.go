package gitx

import (
	"os"
	"strings"
)

// AutoCommitFlag is the office's own env flag opting every spawned child
// process (embedded terminal shells, backend agent processes) into
// majdoor-authored commits: when it is exactly "true" (case-insensitive),
// the four GIT_AUTHOR_*/GIT_COMMITTER_* vars are injected into the child's
// environment so any `git commit` run inside is AUTHORED by the majdoor.
//
// Why the spawn env and not a git hook: git runs hooks as CHILD processes,
// so a hook's `export` never reaches the git parent — the only seam that
// covers every downstream commit is the office's own process-launch env.
const AutoCommitFlag = "THEBORINGOFFICE_AUTO_COMMIT"

// The git author/committer env keys the majdoor injection owns.
const (
	gitAuthorNameKey     = "GIT_AUTHOR_NAME"
	gitAuthorEmailKey    = "GIT_AUTHOR_EMAIL"
	gitCommitterNameKey  = "GIT_COMMITTER_NAME"
	gitCommitterEmailKey = "GIT_COMMITTER_EMAIL"
)

// MajdoorAuthorEnv returns the four KEY=VALUE pairs that make git's author
// AND committer the majdoor, built from the MajdoorName/MajdoorEmail
// consts (never duplicated strings). Order is fixed: author name, author
// email, committer name, committer email.
func MajdoorAuthorEnv() []string {
	return []string{
		gitAuthorNameKey + "=" + MajdoorName,
		gitAuthorEmailKey + "=" + MajdoorEmail,
		gitCommitterNameKey + "=" + MajdoorName,
		gitCommitterEmailKey + "=" + MajdoorEmail,
	}
}

// MajdoorAuthorEnvActive reports whether AutoCommitFlag is set to exactly
// "true" (case-insensitive) per getenv; empty/unset/anything else is
// false. getenv is injected so the package never reads the process env at
// init and tests never touch it either.
func MajdoorAuthorEnvActive(getenv func(string) string) bool {
	return strings.EqualFold(getenv(AutoCommitFlag), "true")
}

// WithMajdoorAuthorEnv is MajdoorEnvMerge bound to the real process env —
// the one-liner every spawn seam calls on the child env it assembled.
func WithMajdoorAuthorEnv(env []string) []string {
	return MajdoorEnvMerge(env, os.Getenv)
}

// MajdoorEnvMerge merges MajdoorAuthorEnv() into env when the office
// auto-commit flag is active per getenv. The four majdoor vars WIN over
// any pre-existing GIT_AUTHOR_*/GIT_COMMITTER_* entries in env: those are
// dropped first, so the child sees exactly one value per key. All other
// vars keep their relative order. When the flag is off env is returned
// unchanged (the same slice).
func MajdoorEnvMerge(env []string, getenv func(string) string) []string {
	if !MajdoorAuthorEnvActive(getenv) {
		return env
	}
	out := make([]string, 0, len(env)+4)
	for _, kv := range env {
		k := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			k = kv[:i]
		}
		switch k {
		case gitAuthorNameKey, gitAuthorEmailKey, gitCommitterNameKey, gitCommitterEmailKey:
			continue // stripped — the majdoor value appended below wins
		}
		out = append(out, kv)
	}
	return append(out, MajdoorAuthorEnv()...)
}
