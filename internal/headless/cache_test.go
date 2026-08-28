// cache_test.go — the render-cache contract, fully hermetic (fake executor
// seams, pinned clocks — NO chrome): hit/miss/share counters, singleflight
// fan-out (N callers → 1 executor run), LRU eviction order, TTL expiry, the
// 5s negative cache (and what must NEVER be cached), copy-on-return
// mutation isolation, the CacheStats aggregation, and the SaveShot path
// convention.
package headless

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Compile-time pins — the executor seam signatures, verbatim.
var (
	_ func(context.Context, string, int, int) (*Result, error) = renderScreenshot
	_ func(context.Context, string, int) (*SnapResult, error)  = renderSnapshot
)

// fakeClock — the pinned time seam for TTL tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{t: time.Unix(1_700_000_000, 0)} }

func (f *fakeClock) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

// swapEngine pins fresh caches (on the given clock) + fake executors for
// one test; the returned func restores the prod wiring. A nil executor
// keeps the prod one.
func swapEngine(t *testing.T, clock *fakeClock, shot func(context.Context, string, int, int) (*Result, error), snap func(context.Context, string, int) (*SnapResult, error)) (restore func()) {
	t.Helper()
	oldShotCache, oldSnapCache := shotCache, snapCache
	oldShotExec, oldSnapExec := execScreenshot, execSnapshot
	shotCache, snapCache = newShotCache(clock.now), newSnapCache(clock.now)
	if shot != nil {
		execScreenshot = shot
	}
	if snap != nil {
		execSnapshot = snap
	}
	return func() {
		shotCache, snapCache = oldShotCache, oldSnapCache
		execScreenshot, execSnapshot = oldShotExec, oldSnapExec
	}
}

// --- hits, misses, key isolation -----------------------------------------

// TestCacheHitMiss — the "~0ms hit" guarantee, structurally: a repeat call
// for the same key never runs the executor, and the counters say so.
func TestCacheHitMiss(t *testing.T) {
	c := newShotCache(time.Now)
	var runs atomic.Int64
	exec := func() (*Result, error) {
		runs.Add(1)
		return &Result{URL: "https://x.example/", Title: "T", PNG: []byte{1, 2, 3}}, nil
	}
	key := shotKey("https://x.example/", 990, 540)
	r1, err := c.do(key, exec)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := c.do(key, exec)
	if err != nil {
		t.Fatal(err)
	}
	if runs.Load() != 1 {
		t.Fatalf("the second call must be a cache hit (executor ran %d times, want 1)", runs.Load())
	}
	if !bytes.Equal(r1.PNG, r2.PNG) || r1.Title != r2.Title {
		t.Fatalf("hit returned a different result: %+v vs %+v", r1, r2)
	}
	var s Stats
	c.statsInto(&s)
	if s.Misses != 1 || s.Hits != 1 || s.Entries != 1 || s.Shares != 0 || s.NegHits != 0 {
		t.Fatalf("stats = %+v, want Misses=1 Hits=1 Entries=1 Shares=0 NegHits=0", s)
	}
}

// TestCacheKeyIsolation — (url, widthPx, heightPx) is the screenshot key,
// (url, maxText) the snapshot key: changing any field renders afresh.
func TestCacheKeyIsolation(t *testing.T) {
	c := newShotCache(time.Now)
	var runs atomic.Int64
	exec := func() (*Result, error) {
		runs.Add(1)
		return &Result{PNG: []byte{1}}, nil
	}
	c.do(shotKey("https://x.example/", 990, 540), exec)     // miss
	c.do(shotKey("https://x.example/", 990, 540), exec)     // hit
	c.do(shotKey("https://x.example/", 320, 200), exec)     // other box → miss
	c.do(shotKey("https://other.example/", 990, 540), exec) // other url → miss
	if runs.Load() != 3 {
		t.Fatalf("runs = %d, want 3 (box and url are key fields)", runs.Load())
	}
	var s Stats
	c.statsInto(&s)
	if s.Entries != 3 {
		t.Fatalf("Entries = %d, want 3 distinct keys", s.Entries)
	}
}

// --- singleflight ----------------------------------------------------------

// TestScreenshotSingleflightShare — THE wave-86 guarantee, through the
// public front: N concurrent Screenshot calls for the same (url, box)
// share ONE executor (Chrome) run; every caller gets the same content as
// its own copy.
func TestScreenshotSingleflightShare(t *testing.T) {
	fc := newFakeClock()
	var runs atomic.Int64
	entered := make(chan struct{})
	release := make(chan struct{})
	shot := func(ctx context.Context, rawurl string, w, h int) (*Result, error) {
		if runs.Add(1) == 1 {
			close(entered)
		}
		<-release // hold the render until every caller has joined the flight
		return &Result{URL: rawurl, Title: "Shared", PNG: []byte{9, 8, 7, 6}}, nil
	}
	defer swapEngine(t, fc, shot, nil)()

	const n = 3
	results := make([]*Result, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[0], errs[0] = Screenshot(context.Background(), "https://shared.example/", 990, 540)
	}()
	<-entered // the leader is inside the executor
	for i := 1; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = Screenshot(context.Background(), "https://shared.example/", 990, 540)
		}(i)
	}
	// every waiter must be parked on the flight before the leader returns
	deadline := time.After(3 * time.Second)
	for {
		shotCache.mu.Lock()
		s := shotCache.shares
		shotCache.mu.Unlock()
		if s == int64(n-1) {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("waiters never joined the flight: shares=%d, want %d", s, n-1)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(release)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("executor ran %d times for %d concurrent callers, want 1", got, n)
	}
	for i := 1; i < n; i++ {
		if !bytes.Equal(results[0].PNG, results[i].PNG) || results[0].Title != results[i].Title {
			t.Fatalf("caller %d got a different result than the leader", i)
		}
	}
	// each caller's copy is its own: mutating one poisons none
	results[0].PNG[0] = 0xFF
	for i := 1; i < n; i++ {
		if results[i].PNG[0] == 0xFF {
			t.Fatalf("caller %d aliases caller 0's PNG slice", i)
		}
	}
	st := CacheStats()
	if st.Misses != 1 || st.Shares != int64(n-1) || st.Hits != 0 {
		t.Fatalf("CacheStats = %+v, want Misses=1 Shares=%d Hits=0", st, n-1)
	}
	t.Logf("singleflight: %d concurrent callers → %d executor run (shares=%d), identical content, isolated slices",
		n, runs.Load(), st.Shares)
}

// TestCacheSingleflightError — a failing leader fans the SAME error out to
// every waiter (one run total), and the failure negative-caches for 5s.
func TestCacheSingleflightError(t *testing.T) {
	c := newShotCache(time.Now)
	var runs atomic.Int64
	entered := make(chan struct{})
	release := make(chan struct{})
	navErr := &NavError{URL: "https://dead.example/", Err: errors.New("net::ERR_CONNECTION_REFUSED")}
	exec := func() (*Result, error) {
		if runs.Add(1) == 1 {
			close(entered)
		}
		<-release
		return nil, navErr
	}
	key := shotKey("https://dead.example/", 990, 540)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(1)
	go func() { defer wg.Done(); _, errs[0] = c.do(key, exec) }()
	<-entered
	wg.Add(1)
	go func() { defer wg.Done(); _, errs[1] = c.do(key, exec) }()
	deadline := time.After(3 * time.Second)
	for {
		c.mu.Lock()
		s := c.shares
		c.mu.Unlock()
		if s == 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("the waiter never joined the flight")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(release)
	wg.Wait()
	if errs[0] != navErr || errs[1] != navErr {
		t.Fatalf("the leader's error must fan out verbatim: %v / %v", errs[0], errs[1])
	}
	if runs.Load() != 1 {
		t.Fatalf("executor ran %d times, want 1 (the failure was shared)", runs.Load())
	}
	// the shared failure negative-caches: a third call is a negHit
	if _, err := c.do(key, exec); err != navErr {
		t.Fatalf("the shared failure must negative-cache: %v", err)
	}
	if runs.Load() != 1 {
		t.Fatalf("a negative-cached failure re-ran the executor: runs = %d", runs.Load())
	}
	var s Stats
	c.statsInto(&s)
	if s.Misses != 1 || s.Shares != 1 || s.NegHits != 1 {
		t.Fatalf("stats = %+v, want Misses=1 Shares=1 NegHits=1", s)
	}
}

// --- LRU / TTL ---------------------------------------------------------------

// TestCacheLRUEviction — at capacity the LEAST-recently-used entry goes;
// a hit bumps its key back to the front.
func TestCacheLRUEviction(t *testing.T) {
	fc := newFakeClock()
	c := newCache[*Result](3, cacheTTL, cacheNegTTL, fc.now, cloneResult)
	ran := map[string]int{}
	exec := func(url string) func() (*Result, error) {
		return func() (*Result, error) {
			ran[url]++
			return &Result{URL: url, PNG: []byte(url)}, nil
		}
	}
	c.do("A", exec("A"))
	c.do("B", exec("B"))
	c.do("C", exec("C")) // LRU order now: C B A (A oldest)
	c.do("A", exec("A")) // hit — A bumps to the front: A C B (B oldest)
	c.do("D", exec("D")) // over capacity → evicts B, NOT A
	c.do("B", exec("B")) // B was evicted → the executor runs again
	if ran["B"] != 2 {
		t.Errorf("B (the LRU entry) ran %d times, want 2 (evicted by D after A's bump)", ran["B"])
	}
	for _, k := range []string{"A", "C", "D"} {
		if ran[k] != 1 {
			t.Errorf("%s ran %d times, want 1 (retained)", k, ran[k])
		}
	}
	var s Stats
	c.statsInto(&s)
	if s.Entries != 3 {
		t.Errorf("Entries = %d, want 3 (at capacity)", s.Entries)
	}
	if s.Misses != 5 || s.Hits != 1 {
		t.Errorf("stats = %+v, want Misses=5 Hits=1", s)
	}
	t.Logf("LRU: insert A B C, bump A, insert D → B evicted (A C D retained)")
}

// TestCacheTTLExpiry — a fresh entry serves for 30s (pinned clock), then
// the executor runs again.
func TestCacheTTLExpiry(t *testing.T) {
	fc := newFakeClock()
	c := newShotCache(fc.now)
	var runs atomic.Int64
	exec := func() (*Result, error) {
		runs.Add(1)
		return &Result{URL: "https://x.example/", PNG: []byte{1}}, nil
	}
	key := shotKey("https://x.example/", 990, 540)
	c.do(key, exec)                    // miss → runs 1
	fc.advance(cacheTTL - time.Second) // t+29s
	c.do(key, exec)                    // hit
	if runs.Load() != 1 {
		t.Fatalf("inside the 30s TTL: runs = %d, want 1", runs.Load())
	}
	fc.advance(2 * time.Second) // t+31s
	c.do(key, exec)             // expired → the executor re-runs
	if runs.Load() != 2 {
		t.Fatalf("past the 30s TTL: runs = %d, want 2", runs.Load())
	}
	var s Stats
	c.statsInto(&s)
	if s.Hits != 1 || s.Misses != 2 {
		t.Fatalf("stats = %+v, want Hits=1 Misses=2", s)
	}
}

// --- the negative cache ------------------------------------------------------

// TestCacheNegativeTTL — a navigation failure is served from memory for 5s
// (negHit, no re-run), then the executor tries again.
func TestCacheNegativeTTL(t *testing.T) {
	fc := newFakeClock()
	c := newShotCache(fc.now)
	var runs atomic.Int64
	navErr := &NavError{URL: "https://dead.example/", Err: errors.New("net::ERR_CONNECTION_REFUSED")}
	exec := func() (*Result, error) {
		runs.Add(1)
		return nil, navErr
	}
	key := shotKey("https://dead.example/", 990, 540)
	if _, err := c.do(key, exec); err != navErr {
		t.Fatalf("first call: err = %v, want the nav error", err)
	}
	if runs.Load() != 1 {
		t.Fatalf("runs = %d, want 1", runs.Load())
	}
	fc.advance(cacheNegTTL - time.Second) // t+4s: inside the negative window
	if _, err := c.do(key, exec); err != navErr {
		t.Fatalf("negative hit must return the SAME error: %v", err)
	}
	if runs.Load() != 1 {
		t.Fatalf("the cached error must not re-run the executor: runs = %d", runs.Load())
	}
	var s Stats
	c.statsInto(&s)
	if s.NegHits != 1 || s.Misses != 1 {
		t.Fatalf("stats = %+v, want NegHits=1 Misses=1", s)
	}
	fc.advance(cacheNegTTL + time.Second) // t+10s: the window has closed
	if _, err := c.do(key, exec); err != navErr {
		t.Fatalf("after the negative TTL: err = %v", err)
	}
	if runs.Load() != 2 {
		t.Fatalf("after the 5s negative TTL the executor re-runs: runs = %d, want 2", runs.Load())
	}
	s = Stats{} // statsInto ACCUMULATES — start from zero
	c.statsInto(&s)
	if s.Misses != 2 || s.NegHits != 1 {
		t.Fatalf("stats = %+v, want Misses=2 NegHits=1", s)
	}
}

// TestNegativeCacheable — the discipline table, pure: ONLY timeouts and
// navigation failures cache negatively; policy refusals, chrome-missing,
// cancellations and unknown errors never do.
func TestNegativeCacheable(t *testing.T) {
	navErr := &NavError{URL: "https://dead.example/", Err: errors.New("boom")}
	timeoutErr := fmt.Errorf("headless: https://slow.example/: timed out after %s: %w", navTimeout, context.DeadlineExceeded)
	canceledErr := fmt.Errorf("headless: https://x.example/: canceled: %w", context.Canceled)
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"navigation failure", navErr, true},
		{"engine timeout", timeoutErr, true},
		{"policy refusal", &PolicyError{URL: "http://x.example/", Reason: "plain http is off"}, false},
		{"chrome missing", ErrChromeNotFound, false},
		{"caller canceled", canceledErr, false},
		{"plain canceled", context.Canceled, false},
		{"unknown error", errors.New("weird"), false},
	}
	for _, c := range cases {
		if got := negativeCacheable(c.err); got != c.want {
			t.Errorf("%s: negativeCacheable = %v, want %v", c.name, got, c.want)
		}
	}
	// the classified nav error stays detectable through the public chain
	var nav *NavError
	if !errors.As(classify("https://x.example/", errors.New("net::ERR")), &nav) {
		t.Error("classify must return a *NavError for navigation failures")
	}
}

// TestCacheNeverCachesFastFailures — policy refusals, chrome-missing and
// caller cancellations fail fast on EVERY call (through the public front).
func TestCacheNeverCachesFastFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"policy refusal", &PolicyError{URL: "http://x.example/", Reason: "plain http is off"}},
		{"chrome missing", ErrChromeNotFound},
		{"caller canceled", fmt.Errorf("headless: https://x.example/: canceled: %w", context.Canceled)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fc := newFakeClock()
			var runs atomic.Int64
			shot := func(ctx context.Context, rawurl string, w, h int) (*Result, error) {
				runs.Add(1)
				return nil, c.err
			}
			defer swapEngine(t, fc, shot, nil)()
			for i := 0; i < 2; i++ {
				if _, err := Screenshot(context.Background(), "https://x.example/", 990, 540); err != c.err {
					t.Fatalf("call %d: err = %v, want the exact failure %v", i, err, c.err)
				}
			}
			if runs.Load() != 2 {
				t.Fatalf("%s: executor ran %d times over 2 calls, want 2 (fail fast, never cached)", c.name, runs.Load())
			}
			if st := CacheStats(); st.NegHits != 0 || st.Entries != 0 {
				t.Fatalf("%s must never be cached: %+v", c.name, st)
			}
		})
	}
}

// --- copy-on-return ----------------------------------------------------------

// TestCacheCopyOnReturn — a caller mutating its result (PNG bytes, titles,
// link slices) never poisons the cached canonical or the next reader.
func TestCacheCopyOnReturn(t *testing.T) {
	c := newShotCache(time.Now)
	exec := func() (*Result, error) {
		return &Result{URL: "https://x.example/", Title: "T", PNG: []byte{1, 2, 3, 4}}, nil
	}
	key := shotKey("https://x.example/", 990, 540)
	r1, err := c.do(key, exec)
	if err != nil {
		t.Fatal(err)
	}
	r1.PNG[0] = 0xFF
	r1.Title = "MUTATED"
	r2, err := c.do(key, exec) // hit
	if err != nil {
		t.Fatal(err)
	}
	if r2.PNG[0] != 1 || r2.Title != "T" {
		t.Fatalf("a caller mutation poisoned the cache: PNG[0]=%d Title=%q", r2.PNG[0], r2.Title)
	}

	sc := newSnapCache(time.Now)
	sexec := func() (*SnapResult, error) {
		return &SnapResult{URL: "https://x.example/", Text: "hi", Links: []Link{{Text: "a", URL: "https://a/"}}}, nil
	}
	skey := snapKey("https://x.example/", 6000)
	s1, err := sc.do(skey, sexec)
	if err != nil {
		t.Fatal(err)
	}
	s1.Links[0].URL = "https://evil.example/"
	s1.Text = "MUTATED"
	s2, err := sc.do(skey, sexec) // hit
	if err != nil {
		t.Fatal(err)
	}
	if s2.Links[0].URL != "https://a/" || s2.Text != "hi" {
		t.Fatalf("a caller mutation poisoned the snapshot cache: %+v", s2)
	}
}

// --- through the public fronts + CacheStats aggregation ----------------------

// TestScreenshotThroughCache — the scripted sequence over BOTH public
// fronts, asserting the executor-run counts AND the CacheStats aggregate.
func TestScreenshotThroughCache(t *testing.T) {
	fc := newFakeClock()
	var shotRuns, snapRuns atomic.Int64
	shot := func(ctx context.Context, rawurl string, w, h int) (*Result, error) {
		shotRuns.Add(1)
		return &Result{URL: rawurl, Title: "T", PNG: []byte{byte(w), byte(h)}}, nil
	}
	snap := func(ctx context.Context, rawurl string, maxText int) (*SnapResult, error) {
		snapRuns.Add(1)
		return &SnapResult{URL: rawurl, Title: "T", Text: "hello", Links: []Link{{Text: "a", URL: "https://a/"}}}, nil
	}
	defer swapEngine(t, fc, shot, snap)()

	if _, err := Screenshot(context.Background(), "https://x.example/", 990, 540); err != nil {
		t.Fatal(err)
	}
	if _, err := Screenshot(context.Background(), "https://x.example/", 990, 540); err != nil {
		t.Fatal(err)
	} // hit
	if _, err := Screenshot(context.Background(), "https://x.example/", 320, 200); err != nil {
		t.Fatal(err)
	} // other box → miss
	if _, err := Snapshot(context.Background(), "https://x.example/", 6000); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), "https://x.example/", 6000); err != nil {
		t.Fatal(err)
	} // hit
	if shotRuns.Load() != 2 || snapRuns.Load() != 1 {
		t.Fatalf("executor runs: shot=%d snap=%d, want 2 and 1", shotRuns.Load(), snapRuns.Load())
	}
	st := CacheStats()
	want := Stats{Hits: 2, Misses: 3, Shares: 0, NegHits: 0, Entries: 3} // 2 shot boxes + 1 snap key
	if st != want {
		t.Fatalf("CacheStats = %+v, want %+v", st, want)
	}
	t.Logf("scripted sequence → CacheStats: %+v", st)
}

// TestSnapshotThroughCache — the snapshot cache keys on (url, maxText) and
// clones on return, through the public front.
func TestSnapshotThroughCache(t *testing.T) {
	fc := newFakeClock()
	var runs atomic.Int64
	snap := func(ctx context.Context, rawurl string, maxText int) (*SnapResult, error) {
		runs.Add(1)
		return &SnapResult{URL: rawurl, Text: "body", Links: []Link{{Text: "a", URL: "https://a/"}}}, nil
	}
	defer swapEngine(t, fc, nil, snap)()
	if _, err := Snapshot(context.Background(), "https://x.example/", 6000); err != nil {
		t.Fatal(err)
	}
	r1, err := Snapshot(context.Background(), "https://x.example/", 6000) // hit
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(context.Background(), "https://x.example/", 100); err != nil {
		t.Fatal(err)
	} // other maxText → miss
	if runs.Load() != 2 {
		t.Fatalf("runs = %d, want 2 (maxText is a key field)", runs.Load())
	}
	if st := CacheStats(); st.Hits != 1 || st.Misses != 2 || st.Entries != 2 {
		t.Fatalf("CacheStats = %+v, want Hits=1 Misses=2 Entries=2", st)
	}
	// copy-on-return through the public front
	r1.Links[0].URL = "https://evil.example/"
	r2, err := Snapshot(context.Background(), "https://x.example/", 6000)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Links[0].URL != "https://a/" {
		t.Fatalf("a caller mutation poisoned the cache: %+v", r2.Links[0])
	}
}

// TestScreenshotInvalidViewport — the caller-bug error returns BEFORE the
// cache is touched (no miss counted, no entry stored), message unchanged.
func TestScreenshotInvalidViewport(t *testing.T) {
	fc := newFakeClock()
	var runs atomic.Int64
	shot := func(ctx context.Context, rawurl string, w, h int) (*Result, error) {
		runs.Add(1)
		return &Result{}, nil
	}
	defer swapEngine(t, fc, shot, nil)()
	_, err := Screenshot(context.Background(), "https://x.example/", 0, 540)
	if err == nil {
		t.Fatal("an invalid viewport must error")
	}
	want := "headless: invalid viewport 0x540 (both dimensions must be > 0)"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	if runs.Load() != 0 {
		t.Fatalf("the executor ran %d times for an invalid viewport, want 0", runs.Load())
	}
	if st := CacheStats(); st.Misses != 0 || st.Entries != 0 {
		t.Fatalf("an invalid viewport must never touch the cache: %+v", st)
	}
}

// --- SaveShot -----------------------------------------------------------------

// TestSaveShot — the PNG landing convention, verbatim:
// <THEBORINGOFFICE_HOME>/shots/<unix-millis>-<sha1(png)[:8]>.png, dirs
// 0o755, files 0o644, bytes round-trip.
func TestSaveShot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", home)
	png := []byte{0x89, 'P', 'N', 'G', 1, 2, 3, 4}
	p, err := SaveShot(png)
	if err != nil {
		t.Fatalf("SaveShot: %v", err)
	}
	if dir := filepath.Dir(p); dir != filepath.Join(home, "shots") {
		t.Fatalf("dir = %q, want %q", dir, filepath.Join(home, "shots"))
	}
	sum := sha1.Sum(png)
	wantHash := hex.EncodeToString(sum[:4])
	base := filepath.Base(p)
	if !strings.HasSuffix(base, "-"+wantHash+".png") {
		t.Fatalf("name %q must end with -<hash8>.png (hash8=%s)", base, wantHash)
	}
	tsPart := strings.TrimSuffix(strings.TrimSuffix(base, ".png"), "-"+wantHash)
	ts, perr := strconv.ParseInt(tsPart, 10, 64)
	if perr != nil || ts <= 0 {
		t.Fatalf("the ts prefix %q must be unix millis: %v", tsPart, perr)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, png) {
		t.Fatal("the written PNG does not round-trip")
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("file perms = %o, want 644", fi.Mode().Perm())
	}
	di, err := os.Stat(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o755 {
		t.Errorf("dir perms = %o, want 755", di.Mode().Perm())
	}
	// identical bytes share the hash tail (a re-shot differs only by ts)
	p2, err := SaveShot(png)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p2, "-"+wantHash+".png") {
		t.Fatalf("the second save %q must carry the same hash tail", p2)
	}
	t.Logf("SaveShot: %s", p)
}

// TestShotsDir — the landing-zone selection: the home override wins, else
// <os.TempDir>/shots.
func TestShotsDir(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", "")
	if got := shotsDir(); got != filepath.Join(os.TempDir(), "shots") {
		t.Fatalf("no override → os.TempDir()/shots, got %q", got)
	}
	t.Setenv("THEBORINGOFFICE_HOME", "/tmp/member-home")
	if got := shotsDir(); got != "/tmp/member-home/shots" {
		t.Fatalf("the home override wins, got %q", got)
	}
}
