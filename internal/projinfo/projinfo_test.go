// projinfo_test.go — the Current contract: resolve the repo toplevel project
// and branch (short SHA when detached), degrade silently everywhere git
// can't answer; and the Cache contract: exec at most once per TTL per dir,
// stale-on-error, never hammering a broken repo — and past the TTL, Get
// serves the stale value INSTANTLY while one async refresh runs per dir
// (git execs never ride the frame's goroutine).
package projinfo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// refreshIdle reports whether no async refresh is in flight for dir. Because
// the refresh goroutine stores the entry and clears the in-flight flag
// inside the same locked section, an idle cache also proves the store has
// landed — tests can use this as a deterministic completion seam.
func refreshIdle(c *Cache, dir string) bool {
	key := resolveDir(dir)
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.refreshing[key]
}

// eventually polls cond until it holds, failing after a generous deadline
// (async refresh timing must never turn a test run flaky).
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH: %v", err)
	}
}

// initRepo turns dir into a real repo on branch main with one commit.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "seed.txt")
	run("-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "seed")
}

func TestCurrent(t *testing.T) {
	needGit(t)

	t.Run("plain dir is not a repo", func(t *testing.T) {
		dir := t.TempDir()
		info := Current(dir)
		if want := filepath.Base(dir); info.Project != want {
			t.Errorf("Project = %q, want dir basename %q", info.Project, want)
		}
		if info.Branch != "" {
			t.Errorf("Branch = %q, want empty outside a repo", info.Branch)
		}
	})

	t.Run("real repo resolves toplevel project and branch", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir)
		info := Current(dir)
		if want := filepath.Base(dir); info.Project != want {
			t.Errorf("Project = %q, want toplevel basename %q", info.Project, want)
		}
		if info.Branch != "main" {
			t.Errorf("Branch = %q, want main", info.Branch)
		}
	})

	t.Run("nested dir still reports the repo root project", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir)
		sub := filepath.Join(dir, "a", "b")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		info := Current(sub)
		if want := filepath.Base(dir); info.Project != want {
			t.Errorf("Project = %q, want repo root %q", info.Project, want)
		}
		if info.Branch != "main" {
			t.Errorf("Branch = %q, want main", info.Branch)
		}
	})

	t.Run("detached HEAD stands in the short SHA", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir)
		run := func(args ...string) string {
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
			}
			return strings.TrimSpace(string(out))
		}
		run("checkout", "--detach", "HEAD")
		sha := run("rev-parse", "--short", "HEAD")
		if info := Current(dir); info.Branch != sha {
			t.Errorf("Branch = %q, want short SHA %q when detached", info.Branch, sha)
		}
	})

	t.Run("missing dir degrades to its own basename", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope")
		info := Current(missing)
		if info.Project != "nope" {
			t.Errorf("Project = %q, want %q", info.Project, "nope")
		}
		if info.Branch != "" {
			t.Errorf("Branch = %q, want empty for a missing dir", info.Branch)
		}
	})

	t.Run("empty dir means the working directory", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Skip(err)
		}
		out, err := exec.Command("git", "-C", wd, "rev-parse", "--show-toplevel").Output()
		if err != nil {
			t.Skipf("package dir not inside a repo: %v", err)
		}
		if want := filepath.Base(strings.TrimSpace(string(out))); Current("").Project != want {
			t.Errorf("Project = %q, want %q (working-dir fallback)", Current("").Project, want)
		}
	})
}

func TestCacheMaxOneExecPerTTLPerDir(t *testing.T) {
	dir := t.TempDir()

	orig := execGit
	t.Cleanup(func() { execGit = orig })

	var fail atomic.Bool
	var calls atomic.Int32
	execGit = func(_ context.Context, _ string, args ...string) (string, error) {
		calls.Add(1)
		if fail.Load() {
			return "", errors.New("git exploded")
		}
		switch {
		case len(args) == 2 && args[1] == "--show-toplevel":
			return dir, nil
		case len(args) == 3 && args[1] == "--abbrev-ref":
			return "main", nil
		default:
			return "", errors.New("unexpected git args: " + strings.Join(args, " "))
		}
	}

	c := NewCache(60 * time.Millisecond)
	want := Info{Project: filepath.Base(dir), Branch: "main"}

	if got := c.Get(dir); got != want {
		t.Fatalf("first Get = %+v, want %+v", got, want)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("first Get ran %d git execs, want 2 (toplevel + branch)", n)
	}

	if got := c.Get(dir); got != want {
		t.Fatalf("cached Get = %+v, want %+v", got, want)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("second Get within TTL ran git again (calls=%d, want 2)", n)
	}

	time.Sleep(80 * time.Millisecond)
	if got := c.Get(dir); got != want {
		t.Fatalf("expired Get = %+v, want stale %+v (refresh is async)", got, want)
	}
	// the expiry fired ONE async refresh: wait for it to actually land.
	eventually(t, "the post-TTL async refresh (4 git execs)", func() bool {
		return calls.Load() == 4 && refreshIdle(c, dir)
	})

	if got := c.Get(dir); got != want {
		t.Fatalf("refreshed Get = %+v, want %+v", got, want)
	}
	if n := calls.Load(); n != 4 {
		t.Fatalf("post-TTL Get must refresh exactly once (calls=%d, want 4)", n)
	}

	// Git breaks: the refresh keeps the last good Info, and the failure is
	// memoized too — a broken repo must not be re-probed every frame.
	fail.Store(true)
	time.Sleep(80 * time.Millisecond)
	if got := c.Get(dir); got != want {
		t.Fatalf("expired Get under broken git = %+v, want stale %+v", got, want)
	}
	eventually(t, "the failed async refresh (one more exec)", func() bool {
		return calls.Load() == 5 && refreshIdle(c, dir) // toplevel probe fails -> single exec
	})
	if got := c.Get(dir); got != want {
		t.Fatalf("post-failure Get = %+v, want stale %+v", got, want)
	}
	if n := calls.Load(); n != 5 {
		t.Fatalf("TTL must apply to failures too (calls=%d, want 5)", n)
	}
}

func TestDefaultCacheTTL(t *testing.T) {
	if DefaultCache().ttl != 5*time.Second {
		t.Fatalf("DefaultCache TTL = %v, want 5s", DefaultCache().ttl)
	}
}

// TestCacheExpiredGetReturnsStaleAndRefreshesAsync pins the async-refresh
// contract that keeps git execs OFF the Frame path:
//
//	(a) an expired Get returns the STALE value IMMEDIATELY — proven by
//	    letting the refresh's toplevel probe hang on a channel: a blocking
//	    Get would never come back — and schedules ONE refresh;
//	(b) a later Get after the refresh completed returns the FRESH value;
//	(c) repeated Gets while a refresh is in flight fire NO extra refresh.
func TestCacheExpiredGetReturnsStaleAndRefreshesAsync(t *testing.T) {
	dir := t.TempDir()

	orig := execGit
	t.Cleanup(func() { execGit = orig })

	release := make(chan struct{}) // closing lets in-flight git probes answer
	var calls atomic.Int32
	var seenFirst atomic.Bool
	var branch atomic.Value
	branch.Store("old-branch")
	execGit = func(_ context.Context, _ string, args ...string) (string, error) {
		calls.Add(1)
		switch {
		case len(args) == 2 && args[1] == "--show-toplevel":
			if seenFirst.Swap(true) {
				<-release // every toplevel probe but the cold one waits
			}
			return dir, nil
		case len(args) == 3 && args[1] == "--abbrev-ref":
			return branch.Load().(string), nil
		default:
			return "", errors.New("unexpected git args: " + strings.Join(args, " "))
		}
	}

	c := NewCache(60 * time.Millisecond)
	wantStale := Info{Project: filepath.Base(dir), Branch: "old-branch"}
	wantFresh := Info{Project: filepath.Base(dir), Branch: "new-branch"}

	// getFetching calls Get and FAILS if it has not returned within a hard
	// wall — the whole point: the UI goroutine must never wait on git.
	getFetching := func(what string) Info {
		type res struct{ info Info }
		done := make(chan res, 1)
		go func() { done <- res{c.Get(dir)} }()
		select {
		case r := <-done:
			return r.info
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("%s: Get blocked on git — expired reads must return stale immediately", what)
			return Info{}
		}
	}

	// Cold Get: synchronous as ever (there is no stale to serve yet) and it
	// yields the REAL answer, not an empty placeholder.
	if got := getFetching("cold Get"); got != wantStale {
		t.Fatalf("cold Get = %+v, want %+v (the cold path still computes synchronously)", got, wantStale)
	}
	t.Logf("trace: cold Get -> %+v (sync; git calls=%d)", wantStale, calls.Load())

	// Expire the entry, flip the branch, and read again: the answer must be
	// the OLD value, returned without waiting for the probe it scheduled.
	time.Sleep(80 * time.Millisecond)
	branch.Store("new-branch")

	if got := getFetching("expired Get"); got != wantStale {
		t.Fatalf("(a) expired Get = %+v, want the STALE %+v returned immediately", got, wantStale)
	}
	t.Logf("trace: expired Get -> stale %+v instantly while refresh in flight", wantStale)
	// The scheduled refresh has STARTED (it hangs inside the toplevel probe
	// right now): one cold toplevel+branch pair + one in-flight toplevel.
	eventually(t, "the scheduled async refresh to begin (3 git calls)", func() bool {
		return calls.Load() == 3
	})

	// (c) no refresh storm: hammering expired Gets while the first refresh
	// is still in flight schedules nothing extra.
	for i := 0; i < 3; i++ {
		if got := getFetching("storm Get"); got != wantStale {
			t.Fatalf("(c) storm Get = %+v, want stale %+v", got, wantStale)
		}
	}
	if n := calls.Load(); n != 3 {
		t.Fatalf("(c) expired Gets during an in-flight refresh fired git again (calls=%d, want 3)", n)
	}
	t.Logf("trace: 3 hammer Gets during in-flight refresh -> stale %+v, git calls still %d (no storm)", wantStale, calls.Load())

	// (b) release git; once the refresh lands, Gets return the FRESH value
	// without any further execs.
	close(release)
	eventually(t, "the async refresh to complete", func() bool { return refreshIdle(c, dir) })
	if got := getFetching("post-refresh Get"); got != wantFresh {
		t.Fatalf("(b) Get after refresh completion = %+v, want fresh %+v", got, wantFresh)
	}
	if n := calls.Load(); n != 4 {
		t.Fatalf("(b) the completed refresh ran %d git calls total, want 4 (2 cold + 2 refresh)", n)
	}
	t.Logf("trace: post-completion Get -> fresh %+v (git calls=%d) — stale->fresh hand-off complete", wantFresh, calls.Load())
}
