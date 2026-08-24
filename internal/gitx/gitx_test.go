package gitx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers: every test builds a REAL git repo in t.TempDir(). Commit
// identity is passed via `git -c` flags so nothing depends on the developer's
// global git config.
// ---------------------------------------------------------------------------

// gitIn runs a setup git (not the seam — the thing under test is gitx) and
// fails the test on error. Returns trimmed combined output.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// initRepo creates a fresh repo on branch main.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	gitIn(t, dir, "init", "-b", "main")
	gitIn(t, dir, "config", "commit.gpgsign", "false")
}

// commitAll stages everything and commits with a hermetic identity.
func commitAll(t *testing.T, dir string) {
	t.Helper()
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "x")
}

// writeFile creates (or overwrites) a repo file, making parent dirs.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// workRoot returns the test repo built on dir, resolved through Root itself
// so Repo.Root is git's physical toplevel (macOS /tmp symlinks, etc.).
func workRoot(t *testing.T, dir string) Repo {
	t.Helper()
	repo, err := Root(dir)
	if err != nil {
		t.Fatalf("Root(%s): %v", dir, err)
	}
	return repo
}

// ---------------------------------------------------------------------------

func TestRoot(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "a.txt", "a\n")
	commitAll(t, dir)

	sub := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	repo, err := Root(sub)
	if err != nil {
		t.Fatalf("Root(subdir): %v", err)
	}
	want := gitIn(t, dir, "rev-parse", "--show-toplevel")
	if repo.Root != want {
		t.Fatalf("Root(subdir).Root = %q, want %q", repo.Root, want)
	}

	// Not a repo: the error must wrap git's own stderr wording.
	_, err = Root(t.TempDir())
	if err == nil {
		t.Fatal("Root(non-repo): want error, got nil")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("Root(non-repo) error = %q, want git stderr wording wrapped", err)
	}
	t.Logf("non-repo error (proves stderr wrapping): %v", err)
}

func TestStatus(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	// Baseline commit: one file per scenario.
	writeFile(t, dir, "keep.txt", "k\n")
	writeFile(t, dir, "mod.txt", "one\ntwo\nthree\n")
	writeFile(t, dir, "del.txt", "bye\n")
	writeFile(t, dir, "gone.txt", "g\n")
	writeFile(t, dir, "old.txt", "o\n")
	commitAll(t, dir)

	// Worktree matrix:
	writeFile(t, dir, "mod.txt", "one\ntwo\nthree\nfour\n") // unstaged modify  → " M"
	writeFile(t, dir, "add.txt", "new\n")                   // staged new       → "A "
	gitIn(t, dir, "add", "add.txt")
	if err := os.Remove(filepath.Join(dir, "del.txt")); err != nil { // unstaged delete → " D"
		t.Fatal(err)
	}
	gitIn(t, dir, "rm", "-q", "gone.txt")      // staged delete   → "D "
	gitIn(t, dir, "mv", "old.txt", "new.txt")  // staged rename   → "R "
	writeFile(t, dir, "untracked.txt", "u\n")  // untracked        → "??"
	writeFile(t, dir, "space name.txt", "s\n") // untracked, space → "??"

	// The raw porcelain both formats, for the record (rename + untracked proof).
	t.Logf("porcelain v1 (plain):\n%s", gitIn(t, dir, "status", "--porcelain=v1", "--untracked-files=all"))
	t.Logf("porcelain v1 -z (%%q, NUL-terminated): %q",
		gitIn(t, dir, "status", "--porcelain=v1", "-z", "--untracked-files=all"))

	repo := workRoot(t, dir)
	got, err := repo.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	t.Logf("parsed Status(): %+v", got)

	want := []FileStatus{
		{Path: "add.txt", Code: "A ", Staged: true},
		{Path: "del.txt", Code: " D", Staged: false},
		{Path: "gone.txt", Code: "D ", Staged: true},
		{Path: "mod.txt", Code: " M", Staged: false},
		{Path: "new.txt", Code: "R ", Staged: true}, // rename reports the TARGET
		{Path: "space name.txt", Code: "??", Staged: false},
		{Path: "untracked.txt", Code: "??", Staged: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Status() =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestStat(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	writeFile(t, dir, "m.txt", "one\ntwo\nthree\n")
	writeFile(t, dir, "d.txt", "bye\n")
	writeFile(t, dir, "keep.txt", "k\n")
	commitAll(t, dir)

	writeFile(t, dir, "m.txt", "one\ntwo\nthree\nfour\nfive\n") // +2 unstaged
	writeFile(t, dir, "a.txt", "x\ny\nz\n")                     // staged new: +3
	gitIn(t, dir, "add", "a.txt")
	writeFile(t, dir, "u.txt", "1\n2\n3\n4\n5\n")                  // untracked: 0 line contribution
	if err := os.Remove(filepath.Join(dir, "d.txt")); err != nil { // -1
		t.Fatal(err)
	}

	t.Logf("numstat raw: %q", gitIn(t, dir, "diff", "--numstat", "HEAD", "--", "."))

	repo := workRoot(t, dir)
	got, err := repo.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	want := Summary{Modified: 1, Added: 1, Untracked: 1, Deleted: 1, LinesAdded: 5, LinesRemoved: 1}
	if got != want {
		t.Fatalf("Stat() = %+v, want %+v", got, want)
	}
}

// TestWorkedExample pins the canonical contract example — a temp repo with
// exactly 1 modified, 1 staged-added, 1 untracked file — to exact return
// values (this is the example the manager report renders).
func TestWorkedExample(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "mod.txt", "one\ntwo\n")
	commitAll(t, dir)

	writeFile(t, dir, "mod.txt", "one\ntwo\nthree\n") // +1 unstaged modify
	writeFile(t, dir, "add.txt", "x\ny\n")            // staged new: +2
	gitIn(t, dir, "add", "add.txt")
	writeFile(t, dir, "new.txt", "u1\nu2\nu3\n") // untracked: 0 line stats

	repo := workRoot(t, dir)

	st, err := repo.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	wantStatus := []FileStatus{
		{Path: "add.txt", Code: "A ", Staged: true},
		{Path: "mod.txt", Code: " M", Staged: false},
		{Path: "new.txt", Code: "??", Staged: false},
	}
	if !reflect.DeepEqual(st, wantStatus) {
		t.Fatalf("Status() = %+v, want %+v", st, wantStatus)
	}

	sum, err := repo.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	wantSum := Summary{Modified: 1, Added: 1, Untracked: 1, LinesAdded: 3}
	if sum != wantSum {
		t.Fatalf("Stat() = %+v, want %+v", sum, wantSum)
	}
	t.Logf("worked example: Status()=%+v Stat()=%+v", st, sum)
}

func TestStatBinary(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "keep.txt", "k\n")
	commitAll(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "bin.bin"), []byte{0x00, 0x01, 0x02, 0x00, 0x03, 0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "bin.bin")

	raw := gitIn(t, dir, "diff", "--numstat", "HEAD", "--", ".")
	t.Logf("numstat with binary entry: %q", raw)
	if !strings.HasPrefix(raw, "-\t-\t") {
		t.Fatalf("sanity: expected binary numstat row, got %q", raw)
	}

	repo := workRoot(t, dir)
	got, err := repo.Stat()
	if err != nil {
		t.Fatalf("Stat with binary: %v", err)
	}
	want := Summary{Added: 1, LinesAdded: 0, LinesRemoved: 0}
	if got != want {
		t.Fatalf("Stat() = %+v, want %+v (binary counts 0 lines, no error)", got, want)
	}
}

func TestDiffTracked(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "f.txt", "alpha\nbeta\n")
	commitAll(t, dir)

	writeFile(t, dir, "f.txt", "alpha\nBETA\n")
	gitIn(t, dir, "add", "f.txt")                      // staged half
	writeFile(t, dir, "f.txt", "alpha\nBETA\ngamma\n") // plus unstaged half

	repo := workRoot(t, dir)
	diff, err := repo.Diff("f.txt")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	t.Logf("Diff(f.txt):\n%s", diff)

	if !strings.HasPrefix(diff, "diff --git a/f.txt b/f.txt") {
		t.Fatalf("diff header missing/wrong:\n%s", diff)
	}
	for _, want := range []string{"-beta", "+BETA", "+gamma"} { // staged+unstaged vs HEAD
		if !strings.Contains(diff, want) {
			t.Fatalf("diff missing %q:\n%s", want, diff)
		}
	}
	if strings.Contains(diff, "\x1b[") {
		t.Fatalf("diff contains ANSI color codes (--no-color broken):\n%q", diff)
	}
}

func TestDiffUntracked(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "keep.txt", "k\n")
	commitAll(t, dir)

	writeFile(t, dir, "u.txt", "hello\nworld\n")
	writeFile(t, dir, "nonl.txt", "hello\nworld") // no trailing newline
	writeFile(t, dir, "empty.txt", "")
	writeFile(t, dir, "sub/inner.txt", "deep\n") // nested untracked path

	repo := workRoot(t, dir)

	cases := map[string]string{
		"u.txt":    "diff --git a/u.txt b/u.txt\n--- /dev/null\n+++ b/u.txt\n+hello\n+world\n",
		"nonl.txt": "diff --git a/nonl.txt b/nonl.txt\n--- /dev/null\n+++ b/nonl.txt\n+hello\n+world\n",
		"empty.txt": "diff --git a/empty.txt b/empty.txt\n--- /dev/null\n+++ b/empty.txt\n" +
			"(empty file)\n",
		"sub/inner.txt": "diff --git a/sub/inner.txt b/sub/inner.txt\n--- /dev/null\n" +
			"+++ b/sub/inner.txt\n+deep\n",
	}
	for path, want := range cases {
		got, err := repo.Diff(path)
		if err != nil {
			t.Fatalf("Diff(%s): %v", path, err)
		}
		if got != want {
			t.Fatalf("Diff(%s) =\n%q\nwant\n%q", path, got, want)
		}
	}
	t.Logf("Diff(u.txt) synthetic render:\n%s", cases["u.txt"])
}

func TestDiffUntrackedTruncated(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "keep.txt", "k\n")
	commitAll(t, dir)

	var sb strings.Builder
	for i := 1; i <= maxDiffLines+5; i++ {
		fmt.Fprintf(&sb, "l%d\n", i)
	}
	writeFile(t, dir, "big.txt", sb.String())

	repo := workRoot(t, dir)
	diff, err := repo.Diff("big.txt")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(diff, truncMarker) {
		t.Fatalf("no truncation marker in >50k line synthetic diff")
	}
	if got := strings.Count(diff, "\n"); got != maxDiffLines+4 { // 3 header + 50000 plus + 1 marker
		t.Fatalf("truncated diff has %d newlines, want %d", got, maxDiffLines+4)
	}
	if strings.Contains(diff, fmt.Sprintf("+l%d", maxDiffLines+5)) {
		t.Fatalf("content past the cap leaked into truncated diff")
	}
	tail := diff[strings.LastIndex(strings.TrimRight(diff, "\n"), "\n")+1:]
	if tail != truncMarker+"\n" {
		t.Fatalf("truncated diff does not end with marker, tail = %q", tail)
	}
}

func TestDiffNothingToShow(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "keep.txt", "k\n")
	commitAll(t, dir)

	repo := workRoot(t, dir)
	for _, path := range []string{"keep.txt", "does-not-exist.txt"} {
		got, err := repo.Diff(path)
		if err != nil {
			t.Fatalf("Diff(%s): %v", path, err)
		}
		if got != "" {
			t.Fatalf("Diff(%s) = %q, want empty (clean path)", path, got)
		}
	}
}

func TestExecGitSeam(t *testing.T) {
	orig := execGit
	t.Cleanup(func() { execGit = orig })

	var gotDir string
	var gotArgs []string
	execGit = func(ctx context.Context, dir string, args ...string) (string, error) {
		gotDir = dir
		gotArgs = append(gotArgs, args...)
		return " M a.txt\x00R  c.txt\x00b.txt\x00", nil
	}

	got, err := Repo{Root: "/fake"}.Status()
	if err != nil {
		t.Fatalf("Status via seam: %v", err)
	}
	if gotDir != "/fake" {
		t.Fatalf("seam saw dir %q, want /fake", gotDir)
	}
	if !reflect.DeepEqual(gotArgs, []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"}) {
		t.Fatalf("seam saw args %v", gotArgs)
	}
	want := []FileStatus{
		{Path: "a.txt", Code: " M", Staged: false},
		{Path: "c.txt", Code: "R ", Staged: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("seam-fed Status() = %+v, want %+v", got, want)
	}
}

func TestRepoFromSubdir(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "m.txt", "one\n")
	commitAll(t, dir)
	writeFile(t, dir, "m.txt", "one\ntwo\n")

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	repo, err := Root(sub) // resolves to toplevel, so git runs at Root
	if err != nil {
		t.Fatalf("Root(sub): %v", err)
	}
	st, err := repo.Status()
	if err != nil {
		t.Fatalf("Status from subdir-rooted repo: %v", err)
	}
	want := []FileStatus{{Path: "m.txt", Code: " M", Staged: false}}
	if !reflect.DeepEqual(st, want) {
		t.Fatalf("Status() = %+v, want %+v", st, want)
	}
	sum, err := repo.Stat()
	if err != nil {
		t.Fatalf("Stat from subdir-rooted repo: %v", err)
	}
	if sum.Modified != 1 || sum.LinesAdded != 1 || sum.LinesRemoved != 0 {
		t.Fatalf("Stat() = %+v, want Modified=1 +1/-0", sum)
	}
}
