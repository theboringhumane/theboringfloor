// chat_attach_ignore.go — .gitignore-aware filtering for the @ attach
// picker. The picker's job is listing USEFUL, editable files; build
// output, dependency trees, venvs and caches are noise. The rules come
// from two sources, evaluated LAST-MATCH-WINS in this order:
//
//  1. attachSafeDefaults (below) — built-in exclusions every project
//     gets, .gitignore or not.
//  2. every .gitignore file found from the root down (shallow first) —
//     a rule's scope is the subtree of the directory that holds it, so
//     deeper files win inside their own subtree, exactly like git.
//
// Because the defaults sit FIRST, a member's explicit `!` negation in a
// real .gitignore can re-include a defaulted path (`!.venv/` → the venv
// lists again); nothing can re-include `.git/` — the walker prunes it
// unconditionally before rules are consulted.
//
// Semantics follow git: `#` comments, blank lines, `!` negation, a
// trailing `/` marks directory-only matches, a leading `/` anchors at
// the .gitignore's directory, patterns with an inner `/` are anchored
// full-path matches, patterns without `/` match the BASENAME at any
// depth, `*` never crosses `/`, `**` (as a whole segment) crosses it,
// and the escapes `\ `, `\!`, `\*` survive as literals.
//
// Performance: rule files are parsed once and cached by (root, path,
// mtime, size); a later open re-stats but never re-reads unchanged
// .gitignore files (no disk churn). Match-time cost is a HasPrefix
// short-circuit per base dir plus bounded segment globs — no regex.
package panels

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// attachSafeDefaults — the always-on exclusion list, written as real
// gitignore lines (comments included) and parsed through the same
// pipeline as on-disk rules (base ""). Each rule carries its reason:
// these are module artifacts and swap carriers, not editable sources.
const attachSafeDefaults = `
# the repo's own object store — never attachable (the walker also prunes
# it unconditionally, rules or no rules)
.git/

# python virtualenvs (.venv, .venv-3.12, .venv-restore, ...) — interpreter artifacts
.venv*/

# node dependency tree — installed artifacts
node_modules/

# python bytecode caches + their compiled leaves
__pycache__/
*.pyc

# macOS Finder metadata
.DS_Store

# test-runner / typechecker / linter state caches
.pytest_cache/
.mypy_cache/
.ruff_cache/

# python tox build envs
.tox/

# packaged / compiled build output: node dist, generic build, rust
# target, next.js .next, turborepo cache, goreleaser/static-export out
dist/
build/
target/
.next/
.turbo/
out/

# vendored deps INSIDE packages are artifacts — but a ROOT vendor/ is
# often intentional source-of-truth, so re-include only the top level
**/vendor/
!/vendor/
`

// ignoreRule — one parsed .gitignore line, already split for matching.
// base is the slash path (relative to the walk root, "" = root) of the
// directory holding the .gitignore; the rule only applies under it.
type ignoreRule struct {
	base     string   // owning .gitignore's dir, "" for root / defaults
	neg      bool     // `!` negation — a match RE-INCLUDES the path
	dirOnly  bool     // trailing `/`: matches directories only
	anchored bool     // leading `/` or any inner `/`: full-path match at base
	segs     []string // pattern segments (split on "/"), leading `/` stripped
}

// parseIgnoreLines parses a .gitignore body (already split on "\n")
// into rules scoped at base. Invalid lines are skipped silently — a
// picker is best-effort, never fatal on a weird pattern.
func parseIgnoreLines(body, base string) []ignoreRule {
	var rules []ignoreRule
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSuffix(raw, "\r") // tolerate CRLF-authored files
		line = trimIgnoreTrailingSpaces(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		neg := false
		if strings.HasPrefix(line, "!") {
			neg = true
			line = line[1:]
		}
		dirOnly := false
		if strings.HasSuffix(line, "/") {
			dirOnly = true
			line = line[:len(line)-1]
		}
		anchored := false
		if strings.HasPrefix(line, "/") {
			anchored = true
			line = line[1:]
		}
		if line == "" {
			continue // "!", "/" or "!" + "/" alone — no pattern left
		}
		line = unescapeIgnorePattern(line)
		if strings.Contains(line, "/") {
			anchored = true
		}
		rules = append(rules, ignoreRule{
			base: base, neg: neg, dirOnly: dirOnly,
			anchored: anchored, segs: strings.Split(line, "/"),
		})
	}
	return rules
}

// trimIgnoreTrailingSpaces drops trailing spaces unless the last one is
// backslash-escaped (`\ ` keeps a literal trailing space). A doubled
// backslash before the space cancels the escape (parity count, like git).
func trimIgnoreTrailingSpaces(s string) string {
	for strings.HasSuffix(s, " ") {
		bs := 0
		for i := len(s) - 2; i >= 0 && s[i] == '\\'; i-- {
			bs++
		}
		if bs%2 == 1 {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

// unescapeIgnorePattern folds the minimal escape set: `\ ` → a literal
// space, `\!` → a literal bang, `\*` → a literal star (emitted as the
// `[*]` character class so the glob matcher treats it literally).
// A backslash before anything else is left alone.
func unescapeIgnorePattern(p string) string {
	if !strings.ContainsRune(p, '\\') {
		return p
	}
	r := []rune(p)
	var b strings.Builder
	for i := 0; i < len(r); i++ {
		if r[i] == '\\' && i+1 < len(r) {
			switch r[i+1] {
			case ' ', '!':
				b.WriteRune(r[i+1])
				i++
				continue
			case '*':
				b.WriteString("[*]")
				i++
				continue
			}
		}
		b.WriteRune(r[i])
	}
	return b.String()
}

// matchIgnoreSeg matches one path segment against one pattern segment:
// `*`, `?` and character classes, never crossing `/` (segments hold no
// slashes by construction). A malformed class ([ without ]) falls back
// to literal equality — a bad pattern must never sink the picker.
func matchIgnoreSeg(pat, seg string) bool {
	if ok, err := path.Match(pat, seg); err == nil {
		return ok
	}
	return pat == seg
}

// matchIgnoreSegs matches a full path (as segments) against an anchored
// pattern (as segments) — `**` as a whole segment crosses `/`,
// consuming any number of segments including zero.
func matchIgnoreSegs(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for i := 0; i <= len(segs); i++ {
				if matchIgnoreSegs(pat[1:], segs[i:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 || !matchIgnoreSeg(pat[0], segs[0]) {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}

// ruleMatches reports whether ONE rule matches rel (a slash path
// relative to the walk root). The base prefix short-circuits via
// HasPrefix before any glob work; basename patterns test only the last
// segment (parent dirs are matched/pruned as their own path during the
// walk, which is how git's "pattern matching a directory ignores
// everything under it" falls out naturally).
func ruleMatches(r ignoreRule, rel string, isDir bool) bool {
	if r.dirOnly && !isDir {
		return false
	}
	sub := rel
	if r.base != "" {
		if !strings.HasPrefix(rel, r.base+"/") {
			return false
		}
		sub = rel[len(r.base)+1:]
	}
	if r.anchored {
		return matchIgnoreSegs(r.segs, strings.Split(sub, "/"))
	}
	segs := strings.Split(sub, "/")
	return matchIgnoreSeg(r.segs[0], segs[len(segs)-1])
}

// isIgnoredPath applies the assembled rule list to rel, LAST-MATCH-WINS
// in list order (defaults first, then .gitignore files shallow → deep):
// a matching negation flips the verdict back to included.
func isIgnoredPath(rules []ignoreRule, rel string, isDir bool) bool {
	ignored := false
	for _, r := range rules {
		if ruleMatches(r, rel, isDir) {
			ignored = !r.neg
		}
	}
	return ignored
}

// ---------------------------------------------------------------- rule loading + cache

// ignoreFileCacheEntry — one parsed .gitignore file, keyed by
// (root, path) and validated by mtime+size: an unchanged file is never
// re-read, a touched one is re-parsed exactly once.
type ignoreFileCacheEntry struct {
	modTime time.Time
	size    int64
	rules   []ignoreRule
}

var ignoreFileCache = struct {
	mu     sync.Mutex
	m      map[string]ignoreFileCacheEntry
	parses int // test-visible: counts real re-reads+parses (cache misses)
}{m: map[string]ignoreFileCacheEntry{}}

// cachedParseIgnoreFile returns the compiled rules for one .gitignore
// file, re-reading it only when its mtime/size changed.
func cachedParseIgnoreFile(root, drel, name string, info os.FileInfo) []ignoreRule {
	abs := filepath.Join(root, filepath.FromSlash(joinSlash(drel, name)))
	key := root + "\x00" + filepath.ToSlash(abs)
	ignoreFileCache.mu.Lock()
	if e, ok := ignoreFileCache.m[key]; ok && e.modTime.Equal(info.ModTime()) && e.size == info.Size() {
		ignoreFileCache.mu.Unlock()
		return e.rules
	}
	ignoreFileCache.mu.Unlock()
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil // vanished between stat and read: skip, never fatal
	}
	rules := parseIgnoreLines(string(data), drel)
	ignoreFileCache.mu.Lock()
	ignoreFileCache.m[key] = ignoreFileCacheEntry{modTime: info.ModTime(), size: info.Size(), rules: rules}
	ignoreFileCache.parses++
	ignoreFileCache.mu.Unlock()
	return rules
}

// joinSlash joins a relative slash dir ("" at root) with a name.
func joinSlash(drel, name string) string {
	if drel == "" {
		return name
	}
	return drel + "/" + name
}

// loadIgnoreRules assembles the picker's full rule list for root:
// attachSafeDefaults first, then every .gitignore found under root
// (shallow before deep, each scoped to its own directory). Returns the
// rules, the newest .gitignore mtime seen (zero when none exists), and
// the first hard error (best-effort: unreadable dirs/files are skipped,
// not fatal — the picker degrades to defaults-only). Discovery prunes
// exactly like the listing walk (.git always, rule-ignored dirs, depth
// cap): git never consults a .gitignore under an excluded directory,
// and neither does the picker.
func loadIgnoreRules(root string) ([]ignoreRule, time.Time, error) {
	rules := parseIgnoreLines(attachSafeDefaults, "")
	var lastWriteAt time.Time
	var firstErr error
	var gather func(drel string, active []ignoreRule)
	gather = func(drel string, active []ignoreRule) {
		dirPath := filepath.Join(root, filepath.FromSlash(drel))
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			if firstErr == nil && os.IsNotExist(err) == false {
				firstErr = err
			}
			return
		}
		// clamp cap so the append below ALWAYS copies — siblings must not
		// share a backing array across recursion branches
		active = active[:len(active):len(active)]
		// this dir's own .gitignore applies BEFORE descending its children
		for _, e := range entries {
			if e.Name() != ".gitignore" || e.IsDir() {
				continue
			}
			info, err := os.Stat(filepath.Join(dirPath, e.Name()))
			if err != nil {
				continue
			}
			if info.ModTime().After(lastWriteAt) {
				lastWriteAt = info.ModTime()
			}
			fileRules := cachedParseIgnoreFile(root, drel, e.Name(), info)
			active = append(active, fileRules...)
			rules = append(rules, fileRules...)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if e.Name() == ".git" {
				continue // the object store is never walked, never ruled on
			}
			rel := joinSlash(drel, e.Name())
			if isIgnoredPath(active, rel, true) {
				continue // excluded dir: git never reads .gitignore under it either
			}
			if strings.Count(rel, "/") >= atMaxDepth {
				continue // the listing walk prunes here too — deeper rules can't apply
			}
			gather(rel, active)
		}
	}
	gather("", rules) // discovery also prunes by the defaults (no node_modules descent)
	return rules, lastWriteAt, firstErr
}
