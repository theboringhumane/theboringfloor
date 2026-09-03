package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/gitx"
)

// ---------------------------------------------------------------------------
// Helpers: every test builds a REAL git repo in t.TempDir() (gitx_test.go's
// pattern). Commit identity rides `git -c` flags so nothing depends on the
// developer's global git config.
// ---------------------------------------------------------------------------

// hookGitIn runs a SETUP git command in dir (the ensure is the thing under
// test) and fails the test on error. Returns trimmed combined output.
func hookGitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// initHookRepo creates a fresh repo on branch main with signing off.
func initHookRepo(t *testing.T, dir string) {
	t.Helper()
	hookGitIn(t, dir, "init", "-b", "main")
	hookGitIn(t, dir, "config", "commit.gpgsign", "false")
}

// commitEmpty makes one --allow-empty commit; the installed hook fires here.
func commitEmpty(t *testing.T, dir, msg string) {
	t.Helper()
	hookGitIn(t, dir, "-c", "user.email=t@t", "-c", "user.name=t",
		"commit", "--allow-empty", "-m", msg)
}

// readHook returns the installed hook's bytes and mode.
func readHook(t *testing.T, hookPath string) (string, os.FileMode) {
	t.Helper()
	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook %s: %v", hookPath, err)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook %s: %v", hookPath, err)
	}
	return string(b), info.Mode().Perm()
}

// ---------------------------------------------------------------------------

// TestEnsureMajdoorHookInstallsAndStamps is the end-to-end core: on an
// empty repo the ensure installs the executable hook, git itself then
// appends exactly one majdoor trailer per commit through it, and a second
// commit proves the hook's idempotence. Discovery from a SUBDIR resolves
// to the same repo (boot runs from wherever the office was launched).
func TestEnsureMajdoorHookInstallsAndStamps(t *testing.T) {
	dir := t.TempDir()
	initHookRepo(t, dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")

	status, err := EnsureMajdoorHook(dir, true)
	if err != nil {
		t.Fatalf("EnsureMajdoorHook(enable): %v", err)
	}
	if status != hookStatusInstalled {
		t.Fatalf("status = %q, want %q", status, hookStatusInstalled)
	}

	body, mode := readHook(t, hookPath)
	if body != majdoorCommitMsgHook {
		t.Fatalf("installed hook body differs from the embedded canonical body")
	}
	if mode&0o111 == 0 {
		t.Fatalf("installed hook mode %o is not executable", mode)
	}

	// Discovery from a subdirectory must resolve to the same repo: the
	// hook is already there, byte-identical → present, no second install.
	sub := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	status, err = EnsureMajdoorHook(sub, true)
	if err != nil {
		t.Fatalf("EnsureMajdoorHook(subdir): %v", err)
	}
	if status != hookStatusPresent {
		t.Fatalf("subdir status = %q, want %q", status, hookStatusPresent)
	}

	// Real commits through git itself: the hook fires and appends exactly
	// one trailer; a second commit through the same hook stays single.
	commitEmpty(t, dir, "test")
	log1 := hookGitIn(t, dir, "log", "-1", "--format=%B")
	if got := strings.Count(log1, gitx.MajdoorTrailer); got != 1 {
		t.Fatalf("first commit body carries the trailer %d times, want 1:\n%s", got, log1)
	}
	t.Logf("git log -1 --format=%%B after install:\n%s", log1)

	commitEmpty(t, dir, "test two")
	log2 := hookGitIn(t, dir, "log", "-1", "--format=%B")
	if got := strings.Count(log2, gitx.MajdoorTrailer); got != 1 {
		t.Fatalf("second commit body carries the trailer %d times, want 1 (idempotent):\n%s", got, log2)
	}
}

// TestEnsureMajdoorHookPresentNoOp: a byte-identical hook is left
// untouched — no rewrite, no error.
func TestEnsureMajdoorHookPresentNoOp(t *testing.T) {
	dir := t.TempDir()
	initHookRepo(t, dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")

	if status, err := EnsureMajdoorHook(dir, true); err != nil || status != hookStatusInstalled {
		t.Fatalf("install: status=%q err=%v", status, err)
	}
	before, beforeMode := readHook(t, hookPath)

	status, err := EnsureMajdoorHook(dir, true)
	if err != nil {
		t.Fatalf("second EnsureMajdoorHook: %v", err)
	}
	if status != hookStatusPresent {
		t.Fatalf("status = %q, want %q", status, hookStatusPresent)
	}
	after, afterMode := readHook(t, hookPath)
	if after != before || afterMode != beforeMode {
		t.Fatalf("present hook was rewritten (content or mode changed)")
	}
}

// TestEnsureMajdoorHookSkippedForeignOnEnable: a pre-existing hook that
// is NOT ours is never overwritten or chained — the office leaves the
// member's hook exactly as it found it.
func TestEnsureMajdoorHookSkippedForeignOnEnable(t *testing.T) {
	dir := t.TempDir()
	initHookRepo(t, dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")
	foreign := "#!/bin/sh\necho member's own hook\nexit 0\n"
	if err := os.WriteFile(hookPath, []byte(foreign), 0o755); err != nil {
		t.Fatal(err)
	}

	status, err := EnsureMajdoorHook(dir, true)
	if err != nil {
		t.Fatalf("EnsureMajdoorHook(foreign present): %v", err)
	}
	if status != hookStatusSkippedForeign {
		t.Fatalf("status = %q, want %q", status, hookStatusSkippedForeign)
	}
	if body, _ := readHook(t, hookPath); body != foreign {
		t.Fatalf("foreign hook was clobbered:\n%s", body)
	}
}

// TestEnsureMajdoorHookRemovesOursOnDisable: attribution off uninstalls
// our own byte-identical hook; a second disabled run reports absent.
func TestEnsureMajdoorHookRemovesOursOnDisable(t *testing.T) {
	dir := t.TempDir()
	initHookRepo(t, dir)
	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")

	if status, err := EnsureMajdoorHook(dir, true); err != nil || status != hookStatusInstalled {
		t.Fatalf("install: status=%q err=%v", status, err)
	}
	status, err := EnsureMajdoorHook(dir, false)
	if err != nil {
		t.Fatalf("EnsureMajdoorHook(disable): %v", err)
	}
	if status != hookStatusRemoved {
		t.Fatalf("status = %q, want %q", status, hookStatusRemoved)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("hook still present after removal (stat err=%v)", err)
	}

	// The commit-msg hook is gone: a fresh commit carries NO trailer.
	commitEmpty(t, dir, "unsigned")
	log := hookGitIn(t, dir, "log", "-1", "--format=%B")
	if strings.Contains(log, gitx.MajdoorTrailer) {
		t.Fatalf("trailer present after opt-out removal:\n%s", log)
	}

	status, err = EnsureMajdoorHook(dir, false)
	if err != nil {
		t.Fatalf("second EnsureMajdoorHook(disable): %v", err)
	}
	if status != hookStatusAbsent {
		t.Fatalf("status = %q, want %q", status, hookStatusAbsent)
	}
}

// TestEnsureMajdoorHookDisabledLeavesForeignAndAbsent covers the two
// remaining disabled no-ops: a foreign hook is never removed, and an
// absent hook stays absent.
func TestEnsureMajdoorHookDisabledLeavesForeignAndAbsent(t *testing.T) {
	t.Run("foreign untouched", func(t *testing.T) {
		dir := t.TempDir()
		initHookRepo(t, dir)
		hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")
		foreign := "#!/bin/sh\necho member's own hook\n"
		if err := os.WriteFile(hookPath, []byte(foreign), 0o755); err != nil {
			t.Fatal(err)
		}
		status, err := EnsureMajdoorHook(dir, false)
		if err != nil {
			t.Fatalf("EnsureMajdoorHook(disable, foreign): %v", err)
		}
		if status != hookStatusSkippedForeign {
			t.Fatalf("status = %q, want %q", status, hookStatusSkippedForeign)
		}
		if body, _ := readHook(t, hookPath); body != foreign {
			t.Fatalf("foreign hook removed/altered under attribution off")
		}
	})

	t.Run("absent stays absent", func(t *testing.T) {
		dir := t.TempDir()
		initHookRepo(t, dir)
		status, err := EnsureMajdoorHook(dir, false)
		if err != nil {
			t.Fatalf("EnsureMajdoorHook(disable, absent): %v", err)
		}
		if status != hookStatusAbsent {
			t.Fatalf("status = %q, want %q", status, hookStatusAbsent)
		}
	})
}

// TestEnsureMajdoorHookNoRepo: outside a git work tree the ensure is a
// clean no-op in both postures — the office boots fine anywhere.
func TestEnsureMajdoorHookNoRepo(t *testing.T) {
	dir := t.TempDir() // deliberately NOT git-init'd
	for _, enabled := range []bool{true, false} {
		status, err := EnsureMajdoorHook(dir, enabled)
		if err != nil {
			t.Fatalf("EnsureMajdoorHook(non-repo, enabled=%v): %v", enabled, err)
		}
		if status != hookStatusNoRepo {
			t.Fatalf("status = %q, want %q (enabled=%v)", status, hookStatusNoRepo, enabled)
		}
	}
	// And nothing was scribbled into the plain directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("non-repo dir gained files: %v", entries)
	}
}

// TestEnsureMajdoorHookHonorsHooksPath: core.hooksPath (relative to the
// repo root, or absolute) redirects the install — the default .git/hooks
// must stay empty — and git really fires the hook from the redirected dir.
func TestEnsureMajdoorHookHonorsHooksPath(t *testing.T) {
	t.Run("relative", func(t *testing.T) {
		dir := t.TempDir()
		initHookRepo(t, dir)
		hookGitIn(t, dir, "config", "core.hooksPath", ".githooks")

		status, err := EnsureMajdoorHook(dir, true)
		if err != nil {
			t.Fatalf("EnsureMajdoorHook(hooksPath relative): %v", err)
		}
		if status != hookStatusInstalled {
			t.Fatalf("status = %q, want %q", status, hookStatusInstalled)
		}
		if _, err := os.Stat(filepath.Join(dir, ".githooks", "commit-msg")); err != nil {
			t.Fatalf("hook not installed under core.hooksPath: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "commit-msg")); !os.IsNotExist(err) {
			t.Fatalf("default hooks dir must stay empty (stat err=%v)", err)
		}

		commitEmpty(t, dir, "hookspath commit")
		log := hookGitIn(t, dir, "log", "-1", "--format=%B")
		if got := strings.Count(log, gitx.MajdoorTrailer); got != 1 {
			t.Fatalf("commit via core.hooksPath hook carries the trailer %d times, want 1:\n%s", got, log)
		}
	})

	t.Run("absolute", func(t *testing.T) {
		dir := t.TempDir()
		initHookRepo(t, dir)
		hooksAbs := filepath.Join(t.TempDir(), "shared-hooks")
		hookGitIn(t, dir, "config", "core.hooksPath", hooksAbs)

		status, err := EnsureMajdoorHook(dir, true)
		if err != nil {
			t.Fatalf("EnsureMajdoorHook(hooksPath absolute): %v", err)
		}
		if status != hookStatusInstalled {
			t.Fatalf("status = %q, want %q", status, hookStatusInstalled)
		}
		if _, err := os.Stat(filepath.Join(hooksAbs, "commit-msg")); err != nil {
			t.Fatalf("hook not installed under absolute core.hooksPath: %v", err)
		}
	})
}

// TestMajdoorHookMatchesCanonicalScript pins the embedded hook body
// byte-for-byte against the repo's canonical source,
// scripts/majdoor-commit-msg-hook.sh — the installed hook must be exactly
// the script the docs/installers ship (update both or neither).
func TestMajdoorHookMatchesCanonicalScript(t *testing.T) {
	canonical := filepath.Join("..", "..", "scripts", "majdoor-commit-msg-hook.sh")
	b, err := os.ReadFile(canonical)
	if err != nil {
		t.Skipf("canonical script not readable from package dir (%v) — sync unverifiable here", err)
	}
	if string(b) != majdoorCommitMsgHook {
		t.Fatalf("embedded majdoorCommitMsgHook drifted from %s", canonical)
	}
}

// TestMajdoorHookCarriesMajdoorTrailer cross-pins the hook body against
// gitx's trailer constant: the script must stamp exactly that line (and
// detect it case-insensitively via the email).
func TestMajdoorHookCarriesMajdoorTrailer(t *testing.T) {
	if !strings.Contains(majdoorCommitMsgHook, gitx.MajdoorTrailer) {
		t.Fatalf("hook body does not stamp gitx.MajdoorTrailer %q", gitx.MajdoorTrailer)
	}
	if !strings.Contains(majdoorCommitMsgHook, gitx.MajdoorEmail) {
		t.Fatalf("hook body does not key on gitx.MajdoorEmail %q", gitx.MajdoorEmail)
	}
}

// TestEnsureMajdoorHookManualProof is the manager-facing end-to-end
// driver: pointed at a real repo via HOOK_PROOF_DIR it runs the boot-time
// ensure (enabled) so a shell transcript afterwards can `git commit` and
// watch the trailer land outside the test process. Unset → skip (the
// branch matrix above covers CI).
func TestEnsureMajdoorHookManualProof(t *testing.T) {
	dir := os.Getenv("HOOK_PROOF_DIR")
	if dir == "" {
		t.Skip("HOOK_PROOF_DIR unset — manual-proof driver only")
	}
	status, err := EnsureMajdoorHook(dir, true)
	if err != nil {
		t.Fatalf("EnsureMajdoorHook(%s): %v", dir, err)
	}
	t.Logf("HOOK_PROOF_DIR=%s status=%s", dir, status)
	if status != hookStatusInstalled && status != hookStatusPresent {
		t.Fatalf("status = %q, want installed|present", status)
	}
}
