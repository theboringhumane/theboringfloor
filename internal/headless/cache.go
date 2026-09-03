// cache.go — the engine's render cache (wave 86): the public Screenshot /
// Snapshot fronts ride a per-key memo so the agent-tool path (the 990x540
// artifact in internal/app) and the pane display path (pane dims in
// internal/panels) never re-render the same (url, box) twice inside a short
// window, and concurrent requests for the same key share ONE Chrome run
// (singleflight: the leader renders, the result fans out — every waiter
// gets its own copy).
//
// Discipline:
//   - TTL 30s, capacity 16 entries per cache (LRU) — a flipping pane or a
//     polling agent stays off Chrome. Values are deep-copied ON RETURN so a
//     caller mutating its PNG never poisons the next reader;
//   - policy refusals (*PolicyError) and chrome-missing (ErrChromeNotFound)
//     are NEVER cached — they fail fast on every call (the policy check is
//     pure, and a browser appearing mid-session must be picked up);
//   - caller cancellations (context.Canceled) are never cached either: one
//     caller's abort must not poison the key for the next;
//   - timeouts (context.DeadlineExceeded) and navigation failures
//     (*NavError) ARE cached for 5s (the negative cache) — a hammering
//     agent re-gets the SAME error instead of fork-bombing Chrome against
//     a dead host;
//   - the cache holds VALUES ONLY — never a chromedp context; every render
//     still rides run()'s 15s budget and temp-profile cleanup.
//
// CacheStats() feeds a future status row: hits, misses, shares, negative
// hits, live entries.
package headless

import (
	"container/list"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/brand"
)

const (
	// cacheTTL — how long a fresh render serves from memory.
	cacheTTL = 30 * time.Second
	// cacheNegTTL — how long a timeout/navigation failure serves from
	// memory (the negative cache).
	cacheNegTTL = 5 * time.Second
	// cacheCap — LRU entries per cache (screenshot and snapshot each).
	cacheCap = 16
)

// Executor seams — the ONE swap point per engine op: prod wires the real
// chromedp bodies (renderScreenshot / renderSnapshot in headless.go); cache
// tests pin fakes (NO chrome). The cache wraps these — never the reverse —
// so a swapped fake exercises the exact prod layering.
var (
	execScreenshot = renderScreenshot
	execSnapshot   = renderSnapshot
)

// The package caches — one per op, package-level so BOTH engine call sites
// (the agent tool in internal/app, the pane display in internal/panels)
// share them. Tests swap in fresh instances (and a pinned clock) and
// restore on cleanup.
var (
	shotCache = newShotCache(time.Now)
	snapCache = newSnapCache(time.Now)
)

// Stats — the cache telemetry snapshot returned by CacheStats. The counters
// are process-lifetime monotonic; Entries is a point-in-time size (expired
// entries purge lazily on next access, so it can overstate freshness).
type Stats struct {
	Hits    int64 // served a fresh cached value (the executor did NOT run)
	Misses  int64 // the executor ran (this caller led the render)
	Shares  int64 // joined an in-flight render (the leader's result fanned out)
	NegHits int64 // served a cached timeout/navigation error
	Entries int   // live entries right now, across both caches
}

// CacheStats — hits, misses, shares, negative-hits and live entries across
// BOTH caches (screenshot + snapshot), for a future status row.
func CacheStats() Stats {
	var s Stats
	shotCache.statsInto(&s)
	snapCache.statsInto(&s)
	return s
}

// entry — one cached value (or one cached error) with its expiry. val is
// the CANONICAL copy: it is never handed out — every caller gets a clone.
type entry[V any] struct {
	key     string
	val     V
	err     error // non-nil marks a negative entry
	expires time.Time
}

// flight — one in-flight render: waiters block on done, then clone val
// (close(done) happens-before their reads).
type flight[V any] struct {
	done chan struct{}
	val  V
	err  error
}

// cache — one LRU + singleflight memo over string keys. The counters ride
// mu (no atomics); the clock and the copier are seams for tests.
type cache[V any] struct {
	mu       sync.Mutex
	ll       *list.List // front = most recently used; values are *entry[V]
	items    map[string]*list.Element
	inflight map[string]*flight[V]
	cap      int
	ttl      time.Duration
	negTTL   time.Duration
	now      func() time.Time // clock seam — tests pin it
	clone    func(V) V        // copy-on-return
	hits     int64
	misses   int64
	shares   int64
	negHits  int64
}

func newCache[V any](capacity int, ttl, negTTL time.Duration, now func() time.Time, clone func(V) V) *cache[V] {
	return &cache[V]{
		ll:       list.New(),
		items:    make(map[string]*list.Element),
		inflight: make(map[string]*flight[V]),
		cap:      capacity,
		ttl:      ttl,
		negTTL:   negTTL,
		now:      now,
		clone:    clone,
	}
}

func newShotCache(now func() time.Time) *cache[*Result] {
	return newCache[*Result](cacheCap, cacheTTL, cacheNegTTL, now, cloneResult)
}

func newSnapCache(now func() time.Time) *cache[*SnapResult] {
	return newCache[*SnapResult](cacheCap, cacheTTL, cacheNegTTL, now, cloneSnapResult)
}

// do — the get-or-render path: fresh hit → clone and go; in-flight → share
// the leader's single Chrome run; otherwise lead the render and cache per
// the error discipline (success 30s, timeout/navigation 5s, everything else
// never).
func (c *cache[V]) do(key string, exec func() (V, error)) (V, error) {
	c.mu.Lock()
	if el, ok := c.items[key]; ok {
		e := el.Value.(*entry[V])
		if c.now().Before(e.expires) {
			c.ll.MoveToFront(el)
			if e.err != nil {
				c.negHits++
				err := e.err
				c.mu.Unlock()
				var zero V
				return zero, err
			}
			c.hits++
			val := e.val
			c.mu.Unlock()
			return c.clone(val), nil
		}
		// expired — purge lazily (Entries shrinks on access, not on time)
		c.ll.Remove(el)
		delete(c.items, key)
	}
	if f, ok := c.inflight[key]; ok {
		c.shares++
		c.mu.Unlock()
		<-f.done
		if f.err != nil {
			var zero V
			return zero, f.err
		}
		return c.clone(f.val), nil
	}
	f := &flight[V]{done: make(chan struct{})}
	c.inflight[key] = f
	c.misses++
	c.mu.Unlock()

	val, err := exec()

	c.mu.Lock()
	delete(c.inflight, key)
	f.val, f.err = val, err
	close(f.done)
	switch {
	case err == nil:
		c.storeLocked(key, val, nil, c.ttl)
	case negativeCacheable(err):
		c.storeLocked(key, val, err, c.negTTL)
	}
	c.mu.Unlock()
	if err != nil {
		var zero V
		return zero, err
	}
	return c.clone(val), nil
}

// storeLocked — insert-or-replace key, then trim to capacity from the back
// (LRU). mu held.
func (c *cache[V]) storeLocked(key string, val V, err error, ttl time.Duration) {
	expires := c.now().Add(ttl)
	if el, ok := c.items[key]; ok {
		e := el.Value.(*entry[V])
		e.val, e.err, e.expires = val, err, expires
		c.ll.MoveToFront(el)
		return
	}
	el := c.ll.PushFront(&entry[V]{key: key, val: val, err: err, expires: expires})
	c.items[key] = el
	for c.ll.Len() > c.cap {
		back := c.ll.Back()
		c.ll.Remove(back)
		delete(c.items, back.Value.(*entry[V]).key)
	}
}

// statsInto — add this cache's counters into s. Locks.
func (c *cache[V]) statsInto(s *Stats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s.Hits += c.hits
	s.Misses += c.misses
	s.Shares += c.shares
	s.NegHits += c.negHits
	s.Entries += len(c.items)
}

// negativeCacheable — the 5s negative-cache discipline: timeouts and
// navigation failures ONLY. Policy refusals, chrome-missing and caller
// cancellations fail fast forever (never cached).
func negativeCacheable(err error) bool {
	var pol *PolicyError
	if errors.As(err, &pol) {
		return false
	}
	if errors.Is(err, ErrChromeNotFound) || errors.Is(err, context.Canceled) {
		return false
	}
	var nav *NavError
	return errors.Is(err, context.DeadlineExceeded) || errors.As(err, &nav)
}

// cloneResult — copy-on-return: the PNG slice is the caller's own.
func cloneResult(r *Result) *Result {
	if r == nil {
		return nil
	}
	out := *r
	out.PNG = append([]byte(nil), r.PNG...)
	return &out
}

// cloneSnapResult — copy-on-return: the Links slice is the caller's own.
func cloneSnapResult(s *SnapResult) *SnapResult {
	if s == nil {
		return nil
	}
	out := *s
	out.Links = append([]Link(nil), s.Links...)
	return &out
}

// shotKey / snapKey — the cache keys: (url, widthPx, heightPx) and
// (url, maxText), \x00-joined so no URL string can collide with the
// numeric fields.
func shotKey(rawurl string, widthPx, heightPx int) string {
	return rawurl + "\x00" + strconv.Itoa(widthPx) + "\x00" + strconv.Itoa(heightPx)
}

func snapKey(rawurl string, maxText int) string {
	return rawurl + "\x00" + strconv.Itoa(maxText)
}

// --- SaveShot — the engine-owned PNG landing convention ------------------

// shotsDir — the PNG landing zone: <THEBORINGOFFICE_HOME>/shots when the
// member/harness overrides home, else <os.TempDir>/shots.
func shotsDir() string {
	if home := brand.Get("HOME"); home != "" {
		return filepath.Join(home, "shots")
	}
	return filepath.Join(os.TempDir(), "shots")
}

// SaveShot writes one PNG as <shotsDir()>/<ts>-<hash8>.png (ts = unix
// millis, hash8 = sha1(png)[:8] hex — a re-shot of the same page lands a
// new file per ts while identical bytes share the hash tail). Dirs are
// 0o755, files 0o644 — the exact convention the app's agent-screenshot
// tool has always used (its saver now delegates here).
func SaveShot(png []byte) (string, error) {
	dir := shotsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	sum := sha1.Sum(png)
	name := fmt.Sprintf("%d-%s.png", time.Now().UnixMilli(), hex.EncodeToString(sum[:4]))
	path := filepath.Join(dir, name)
	return path, os.WriteFile(path, png, 0o644)
}
