// Package gitx provides read-only git status/diff data for the Git panel.
//
// CONTRACT IMPLEMENTED (backend landed on 2026-08-24): the exported API
// below is frozen — a parallel UI developer (internal/panels/gitpanel.go)
// compiles against these exact signatures. Signatures may only change via
// the manager; the bodies in this file implement the contract.
//
// Behavior notes for the panel:
//   - every git call runs under a 2s context timeout, with GIT_PAGER/PAGER
//     forced to cat, and is strictly read-only (rev-parse / status / diff);
//   - Summary line counts come from `git diff --numstat HEAD`, which cannot
//     see untracked files: untracked files add to Summary.Untracked but
//     always contribute 0 to LinesAdded/LinesRemoved;
//   - Diff returns plain, never colorized output (--no-color) — coloring is
//     the panel's job.
package gitx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// FileStatus describes one path from `git status --porcelain=v1`.
type FileStatus struct {
	Path   string // repo-relative path (rename target if renamed)
	Code   string // two-letter porcelain code, e.g. " M", "A ", "??" (index char, worktree char)
	Staged bool   // true when the change is staged in the index
}

// Summary aggregates repo-wide change stats shown in the panel header.
type Summary struct {
	Modified     int // tracked files with modifications (staged or unstaged)
	Added        int // newly added/staged files
	Untracked    int // untracked files ("??")
	Deleted      int // deleted files
	LinesAdded   int // + lines across the working-tree-vs-HEAD diff (numstat)
	LinesRemoved int // - lines across the working-tree-vs-HEAD diff (numstat)
}

// Repo identifies the repository the TUI is operating on.
type Repo struct {
	// Root is the absolute path of the working-tree root.
	Root string
}

// ErrNotImplemented was returned by the pre-implementation skeleton bodies.
// Retained so the exported surface stays frozen; no body returns it anymore.
var ErrNotImplemented = errors.New("gitx: not implemented yet")

// gitTimeout caps every git invocation — a frame must never wait on git.
const gitTimeout = 2 * time.Second

// maxDiffLines caps Diff output per file; anything longer ends early with a
// truncMarker line so the panel renders an explicit cutoff.
const maxDiffLines = 50_000

// truncMarker is appended as its own line when Diff output hits maxDiffLines.
const truncMarker = "… truncated …"

// execGit is the test seam: run `git args...` in dir bounded by ctx and
// return raw stdout. Errors wrap git's stderr wording. Tests (and the uishot
// shot harness) swap it wholesale to fake git.
var execGit = func(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// Env-invariant: no pager may ever intercept panel-bound output.
	cmd.Env = append(os.Environ(), "GIT_PAGER=cat", "PAGER=cat")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), msg, err)
	}
	return stdout.String(), nil
}

// git runs one bounded, read-only git call in dir.
func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	return execGit(ctx, dir, args...)
}

// git runs one bounded, read-only git call at the repo root.
func (r Repo) git(args ...string) (string, error) { return git(r.Root, args...) }

// resolveDir maps "" to the process working directory (projinfo pattern).
func resolveDir(dir string) string {
	if dir != "" {
		return dir
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return dir
}

// Root resolves dir to its git working-tree root (git rev-parse --show-toplevel).
func Root(dir string) (Repo, error) {
	d := resolveDir(dir)
	out, err := git(d, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, fmt.Errorf("gitx: root of %s: %w", d, err)
	}
	top := strings.TrimSpace(out)
	if top == "" {
		return Repo{}, fmt.Errorf("gitx: root of %s: git returned an empty toplevel", d)
	}
	return Repo{Root: top}, nil
}

// Status lists changed paths, sorted by Path. Renames report the target path.
func (r Repo) Status() ([]FileStatus, error) {
	out, err := r.git("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("gitx: status: %w", err)
	}
	return parseStatus(out), nil
}

// parseStatus decodes `git status --porcelain=v1 -z` output. In -z mode every
// record is "XY path\0"; a rename/copy record is "XY new\0old\0" (the arrow
// and the quoting of the plain format disappear). Whether the index or the
// worktree side holds an R, the record's path is the rename target and the
// following NUL field is the source, which we skip.
func parseStatus(out string) []FileStatus {
	fields := strings.Split(out, "\x00")
	files := make([]FileStatus, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f == "" || len(f) < 4 {
			continue // trailing NUL padding, or malformed (git guarantees "XY path")
		}
		code, path := f[:2], f[3:]
		if strings.ContainsAny(code, "RC") {
			i++ // skip the rename/copy source path; Path reports the target
		}
		files = append(files, FileStatus{
			Path:   path,
			Code:   code,
			Staged: code[0] != ' ' && code[0] != '?',
		})
	}
	slices.SortFunc(files, func(a, b FileStatus) int { return strings.Compare(a.Path, b.Path) })
	return files
}

// Stat returns repo-wide change counts for the header summary line.
func (r Repo) Stat() (Summary, error) {
	files, err := r.Status()
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for _, f := range files {
		if f.Code == "??" {
			sum.Untracked++
			continue
		}
		// One file counts at most once per bucket, from either code side.
		if strings.ContainsAny(f.Code, "MT") {
			sum.Modified++
		}
		if strings.Contains(f.Code, "A") {
			sum.Added++
		}
		if strings.Contains(f.Code, "D") {
			sum.Deleted++
		}
	}

	// Line totals vs HEAD in a single pass (staged + unstaged combined).
	// `git diff HEAD` cannot see untracked files, so they add nothing here.
	num, err := r.git("diff", "--no-color", "--numstat", "HEAD", "--", ".")
	if err != nil {
		return Summary{}, fmt.Errorf("gitx: numstat: %w", err)
	}
	sum.LinesAdded, sum.LinesRemoved = parseNumstat(num)
	return sum, nil
}

// parseNumstat tallies `git diff --numstat` lines, each "<add>\t<del>\t<path>".
// Binary entries carry "-\t-\t" and simply count as zero lines — never an
// error.
func parseNumstat(out string) (added, removed int) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		if a, err := strconv.Atoi(fields[0]); err == nil {
			added += a
		}
		if d, err := strconv.Atoi(fields[1]); err == nil {
			removed += d
		}
	}
	return added, removed
}

// Diff returns the colored-ready unified diff of path vs HEAD (staged +
// unstaged combined). For untracked files it returns a synthetic all-+
// diff of the file contents. Never mutates the repo.
func (r Repo) Diff(path string) (string, error) {
	out, err := r.git("diff", "--no-color", "HEAD", "--", path)
	if err != nil {
		return "", fmt.Errorf("gitx: diff %s: %w", path, err)
	}
	if strings.TrimSpace(out) != "" {
		return truncate(out), nil
	}
	// git prints nothing for untracked files — synthesize the all-+ view.
	untracked, uerr := r.isUntracked(path)
	if uerr != nil {
		return "", fmt.Errorf("gitx: diff %s: %w", path, uerr)
	}
	if !untracked {
		return "", nil // clean path: nothing to show
	}
	syn, serr := r.syntheticDiff(path)
	if serr != nil {
		return "", fmt.Errorf("gitx: diff %s: %w", path, serr)
	}
	return syn, nil // syntheticDiff already enforces maxDiffLines
}

// isUntracked reports whether path appears in Status() with code "??".
func (r Repo) isUntracked(path string) (bool, error) {
	files, err := r.Status()
	if err != nil {
		return false, err
	}
	for _, f := range files {
		if f.Path == path && f.Code == "??" {
			return true, nil
		}
	}
	return false, nil
}

// syntheticDiff builds the all-+ pseudo-diff the panel shows for an untracked
// file (git itself prints nothing for those). Kept deliberately minimal per
// the contract: header lines, then each content line prefixed "+" — no @@
// hunk math. An empty file gets a "(empty file)" note instead of + lines.
func (r Repo) syntheticDiff(path string) (string, error) {
	f, err := os.Open(filepath.Join(r.Root, path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n", path, path)
	fmt.Fprintf(&b, "--- /dev/null\n")
	fmt.Fprintf(&b, "+++ b/%s\n", path)

	lines, more, err := readLines(f, maxDiffLines+1)
	if err != nil {
		return "", err
	}
	switch {
	case len(lines) == 0:
		b.WriteString("(empty file)\n")
	case more:
		for _, ln := range lines[:maxDiffLines] {
			b.WriteByte('+')
			b.WriteString(ln)
			b.WriteByte('\n')
		}
		b.WriteString(truncMarker + "\n")
	default:
		for _, ln := range lines {
			b.WriteByte('+')
			b.WriteString(ln)
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

// readLines reads up to limit lines from rd; more reports whether it stopped
// early because the limit was reached. Lines come back without their trailing
// "\n". bufio.Reader (no scanner cap) handles arbitrarily long lines.
func readLines(rd io.Reader, limit int) (lines []string, more bool, err error) {
	br := bufio.NewReader(rd)
	for len(lines) < limit {
		line, rerr := br.ReadString('\n')
		if rerr != nil && rerr != io.EOF {
			return nil, false, rerr
		}
		if line == "" && rerr == io.EOF {
			return lines, false, nil
		}
		lines = append(lines, strings.TrimSuffix(line, "\n"))
		if rerr == io.EOF {
			return lines, false, nil
		}
	}
	return lines, true, nil
}

// truncate caps rendered (real) diff output at maxDiffLines lines, closing
// over-long output with a truncMarker line.
func truncate(diff string) string {
	seen := 0
	for i := 0; i < len(diff); i++ {
		if diff[i] == '\n' {
			seen++
			if seen == maxDiffLines {
				if i+1 >= len(diff) {
					return diff // exactly at the cap, nothing more to show
				}
				return diff[:i+1] + truncMarker + "\n"
			}
		}
	}
	return diff
}
