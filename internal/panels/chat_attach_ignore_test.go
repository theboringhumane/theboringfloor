// chat_attach_ignore_test.go — the @ picker's gitignore-awareness: rule
// parsing for every pattern class, last-match-wins, deeper-dir priority,
// safe defaults, gitignore negation overrides, .git's unconditional
// exclusion, a full fixture-tree walk (before/after listing printed for
// the eyeball proof), and the mtime cache's refresh discipline.
package panels

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// writes fixture files (creating parents) under root.
func writeFixture(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("// fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestIgnoreRuleMatch pins every pattern class of the gitignore dialect
// the picker honors: negation, dir-only, anchored `/`, `*`, `**`,
// basename-at-any-depth, escapes, comments/blanks.
func TestIgnoreRuleMatch(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		path    string
		isDir   bool
		want    bool
	}{
		{"basename file", "*.log", "a.log", false, true},
		{"basename at any depth", "*.log", "logs/a.log", false, true},
		{"star never crosses slash", "a*c", "a/b/c", false, false},
		{"star within one segment", "a*c", "axyzc", false, true},
		{"dir-only matches dir", "build/", "build", true, true},
		{"dir-only at any depth", "build/", "sub/build", true, true},
		{"dir-only skips files", "build/", "build", false, false},
		{"dir-only skips nested file", "build/", "sub/build", false, false},
		{"anchored hits root", "/dist", "dist", true, true},
		{"anchored misses depth", "/dist", "sub/dist", true, false},
		{"inner slash anchors", "docs/aa.txt", "docs/aa.txt", false, true},
		{"inner slash misses deeper", "docs/aa.txt", "x/docs/aa.txt", false, false},
		{"doublestar crosses slash", "**/gen/**", "a/gen/b/c", false, true},
		{"doublestar zero segments", "**/gen", "gen", true, true},
		{"prefix doublestar", "a/**/b", "a/x/y/b", false, true},
		{"prefix doublestar direct", "a/**/b", "a/b", false, true},
		{"prefix doublestar misses without tail", "a/**/b", "a/x/y", false, false},
		{"escaped space", `my\ file.txt`, "my file.txt", false, true},
		{"escaped bang is not negation", `\!bang.txt`, "!bang.txt", false, true},
		{"escaped star is literal", `literal\*.txt`, "literal*.txt", false, true},
		{"escaped star is not glob", `literal\*.txt`, "literalX.txt", false, false},
		{"trailing spaces ignored", "knob.txt   ", "knob.txt", false, true},
	}
	for _, tc := range cases {
		rules := parseIgnoreLines(tc.pattern, "")
		if got := isIgnoredPath(rules, tc.path, tc.isDir); got != tc.want {
			t.Errorf("%s: %q vs %q (dir=%v): got %v, want %v", tc.name, tc.pattern, tc.path, tc.isDir, got, tc.want)
		}
	}
	// comments and blank lines parse to NOTHING
	if rules := parseIgnoreLines("# comment\n\n   \n*.log\n", ""); len(rules) != 1 {
		t.Fatalf("comments/blanks must not parse: got %d rules", len(rules))
	}
}

// TestIgnoreRuleLastMatchWins: within and across rule sources the LAST
// matching rule decides — negation mid-group re-includes, a later
// positive re-excludes.
func TestIgnoreRuleLastMatchWins(t *testing.T) {
	rules := parseIgnoreLines("*.log\n!keep.log\n", "")
	if isIgnoredPath(rules, "a.log", false) != true {
		t.Fatal("a.log must be excluded by *.log")
	}
	if isIgnoredPath(rules, "keep.log", false) != false {
		t.Fatal("keep.log must be re-included by the later negation")
	}
	// flip the order: the positive pattern wins now (last match)
	rules2 := parseIgnoreLines("!keep.log\n*.log\n", "")
	if isIgnoredPath(rules2, "keep.log", false) != true {
		t.Fatal("with reversed order keep.log must be excluded (last match wins)")
	}
}

// TestIgnoreDeeperDirWins: a nested .gitignore's rules apply inside its
// own subtree only, and there they outrank shallower rules.
func TestIgnoreDeeperDirWins(t *testing.T) {
	rules := parseIgnoreLines("*.tmp\n", "")
	rules = append(rules, parseIgnoreLines("!keep.tmp\n", "sub")...)
	if isIgnoredPath(rules, "sub/keep.tmp", false) {
		t.Fatal("deeper negation must re-include inside its subtree")
	}
	if isIgnoredPath(rules, "sub/deep/keep.tmp", false) {
		t.Fatal("deeper negation must cover the whole subtree")
	}
	if !isIgnoredPath(rules, "other/keep.tmp", false) {
		t.Fatal("a nested rule must not leak outside its subtree")
	}
	if !isIgnoredPath(rules, "keep.tmp", false) {
		t.Fatal("the root rule still applies at root")
	}
}

// TestWalkAttachSafeDefaultsOnly: with NO .gitignore anywhere, the
// built-in rules alone must keep the list clean.
func TestWalkAttachSafeDefaultsOnly(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		".git/config",               // object store
		".venv/lib/py.py",           // virtualenv
		".venv-3.12/bin/tool",       // versioned virtualenv dir
		"node_modules/left-pad/i.js", // package tree
		"__pycache__/m.cpython.pyc", // bytecode cache
		"cache.pyc",                 // bare bytecode leaf
		".DS_Store",                 // finder metadata
		".pytest_cache/v/cache/n",   // test cache
		".mypy_cache/3.12/x",        // typechecker cache
		".ruff_cache/c",             // linter cache
		".tox/py312/bin/x",          // tox env
		"dist/bundle.js",            // packaged output
		"build/app.js",              // compiled output
		"target/debug/t",            // rust output
		".next/static/x.js",         // next.js output
		".turbo/cache/sum",          // turbo cache
		"out/export/index.html",     // goreleaser/static-export output
		"internal/vendor/dep.go",    // NESTED vendor: artifact
		// ...and what must SURVIVE:
		"src/keep.go",
		"main.go",
		"vendor/rootdep.go", // ROOT vendor is intentional — stays
	} {
		writeFixture(t, root, rel)
	}
	got := walkAttachFiles(root)
	saw := map[string]bool{}
	for _, f := range got {
		saw[f] = true
	}
	for _, want := range []string{"src/keep.go", "main.go", "vendor/rootdep.go"} {
		if !saw[want] {
			t.Errorf("defaults-only: %q must be listed, got %v", want, got)
		}
	}
	for _, unwanted := range []string{
		".git/config", ".venv/lib/py.py", ".venv-3.12/bin/tool",
		"node_modules/left-pad/i.js", "__pycache__/m.cpython.pyc", "cache.pyc",
		".DS_Store", ".pytest_cache/v/cache/n", ".mypy_cache/3.12/x",
		".ruff_cache/c", ".tox/py312/bin/x", "dist/bundle.js", "build/app.js",
		"target/debug/t", ".next/static/x.js", ".turbo/cache/sum",
		"out/export/index.html", "internal/vendor/dep.go",
	} {
		if saw[unwanted] {
			t.Errorf("defaults-only: %q must be filtered out, got %v", unwanted, got)
		}
	}
}

// TestWalkAttachGitIgnoreNegationOverridesDefaults: an explicit `!` in
// the repo's real .gitignore supersedes the built-in defaults — the
// member owns the final word.
func TestWalkAttachGitIgnoreNegationOverridesDefaults(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".gitignore")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("!.venv/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, ".venv/pkg/tool.py")
	writeFixture(t, root, "src/keep.go")
	got := walkAttachFiles(root)
	saw := map[string]bool{}
	for _, f := range got {
		saw[f] = true
	}
	if !saw[".venv/pkg/tool.py"] {
		t.Errorf("!.venv/ must re-include the venv, got %v", got)
	}
	if !saw["src/keep.go"] {
		t.Errorf("src/keep.go must be listed, got %v", got)
	}
}

// TestWalkAttachGitAlwaysExcluded: .git is pruned unconditionally — no
// rule, not even `!.git/`, ever re-includes the object store.
func TestWalkAttachGitAlwaysExcluded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("!.git/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, ".git/config")
	writeFixture(t, root, "main.go")
	for _, f := range walkAttachFiles(root) {
		if strings.HasPrefix(f, ".git/") {
			t.Fatalf(".git must never list, even with !.git/ in .gitignore: got %q", f)
		}
	}
}

// TestWalkAttachFixtureTree — the full before/after: one seeded tree,
// listed once with the OLD prune-only logic (.git + node_modules) and
// once with the new gitignore-aware walk, so the cleaned list is visible
// side by side. Nested .gitignore scoping and anchoring are pinned too.
func TestWalkAttachFixtureTree(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		".git/config",
		".venv/pkg/tool.py",
		"node_modules/foo.js",
		"dist/out.js",
		"src/keep.go",
		"main.go",
		"sub/.gitignore",
		"sub/x.snap",
		"sub/keep.go",
		"sub/.venv-restore/note.py",
		"sub/only-root-here.txt",
		"sub/nested/only-root-here.txt",
		"other/x.snap",
	} {
		writeFixture(t, root, rel)
	}
	// nested .gitignore: a subtree rule (*.snap), a dir rule, an anchored rule
	if err := os.WriteFile(filepath.Join(root, "sub", ".gitignore"),
		[]byte("*.snap\n.venv-restore/\n/only-root-here.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// BEFORE: the pre-change walker (pruned .git + node_modules only)
	before := walkAttachFilesLegacyForTest(root)
	after := walkAttachFiles(root)
	fmt.Println("---- @ PICKER LISTING (fixture tree) ----")
	fmt.Printf("before (%d entries, prune .git/node_modules only):\n", len(before))
	for _, f := range sortedForTest(before) {
		fmt.Println("  " + f)
	}
	fmt.Printf("after (%d entries, gitignore-aware):\n", len(after))
	for _, f := range sortedForTest(after) {
		fmt.Println("  " + f)
	}
	fmt.Println("---- END LISTING ----")

	saw := map[string]bool{}
	for _, f := range after {
		saw[f] = true
	}
	for _, want := range []string{
		"src/keep.go", "main.go", "sub/keep.go",
		"sub/nested/only-root-here.txt", // the anchored rule doesn't reach here
		"other/x.snap", // sub's *.snap must not escape its subtree
		"sub/.gitignore", // gitignore files themselves are editable sources
	} {
		if !saw[want] {
			t.Errorf("fixture tree: %q must be listed, got %v", want, after)
		}
	}
	for _, unwanted := range []string{
		".git/config", ".venv/pkg/tool.py", "node_modules/foo.js", "dist/out.js",
		"sub/x.snap",                 // nested .gitignore's basename rule
		"sub/.venv-restore/note.py",  // nested dir rule (and the .venv* default)
		"sub/only-root-here.txt",     // "/only-root-here.txt" anchored at sub
	} {
		if saw[unwanted] {
			t.Errorf("fixture tree: %q must be filtered out, got %v", unwanted, after)
		}
	}
	// the whole point: the after list is the before list MINUS the noise
	if len(after) >= len(before) {
		t.Fatalf("gitignore-aware walk must shrink the list: before %d, after %d", len(before), len(after))
	}
}

// walkAttachFilesLegacyForTest mirrors the pre-change walker (pruned
// .git + node_modules only) so the test output shows the exact delta.
func walkAttachFilesLegacyForTest(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(root)+"/")
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			if strings.Count(rel, "/") >= atMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if binaryishExt[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > atMaxSize {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out
}

func sortedForTest(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// TestIgnoreCacheRefresh: a .gitignore mtime bump re-reads the rules
// exactly once; an untouched file is served from cache with no re-read
// (stat only). Changing CONTENT (with a bumped mtime) changes behavior.
func TestIgnoreCacheRefresh(t *testing.T) {
	root := t.TempDir()
	gi := filepath.Join(root, ".gitignore")
	now := time.Now()
	if err := os.WriteFile(gi, []byte("a.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(gi, now, now); err != nil {
		t.Fatal(err)
	}
	ignoreFileCache.mu.Lock()
	ignoreFileCache.m = map[string]ignoreFileCacheEntry{}
	ignoreFileCache.parses = 0
	ignoreFileCache.mu.Unlock()

	rules, _, _ := loadIgnoreRules(root)
	if ignoreFileCache.parses != 1 {
		t.Fatalf("first load must parse once, got %d", ignoreFileCache.parses)
	}
	if !isIgnoredPath(rules, "a.txt", false) {
		t.Fatal("a.txt must be excluded by the .gitignore")
	}
	// second load, nothing touched: cached — no disk churn beyond the stat
	if rules, _, _ = loadIgnoreRules(root); ignoreFileCache.parses != 1 {
		t.Fatalf("unchanged mtime must hit the cache (no re-parse), got %d parses", ignoreFileCache.parses)
	}
	// mtime bump only: re-parse once
	if err := os.Chtimes(gi, now.Add(2*time.Second), now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if rules, _, _ = loadIgnoreRules(root); ignoreFileCache.parses != 2 {
		t.Fatalf("mtime bump must invalidate (one re-parse), got %d parses", ignoreFileCache.parses)
	}
	// content change + bump: the NEW rules apply
	if err := os.WriteFile(gi, []byte("b.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(gi, now.Add(4*time.Second), now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	rules, _, _ = loadIgnoreRules(root)
	if ignoreFileCache.parses != 3 {
		t.Fatalf("content change must re-parse once more, got %d parses", ignoreFileCache.parses)
	}
	if isIgnoredPath(rules, "a.txt", false) || !isIgnoredPath(rules, "b.txt", false) {
		t.Fatal("refreshed rules must exclude b.txt and stop excluding a.txt")
	}
}
