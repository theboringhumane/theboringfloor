package cellmetrics

import (
	"testing"
	"time"
)

// TestResolveNoQueryInstantMiss — never queried: the miss is immediate
// (no window was ever opened to wait out).
func TestResolveNoQueryInstantMiss(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	start := time.Now()
	if w, h, ok := Resolve(); ok || w != 0 || h != 0 {
		t.Fatalf("Resolve before any query = (%d,%d,%v), want a clean miss", w, h, ok)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("the no-query miss must be instant, took %s", elapsed)
	}
}

// TestResolveTimeoutFallback — an answering-less terminal: Resolve waits
// out the FIRST query's response window, then misses (the caller's 9x18
// fallback) — and misses INSTANTLY ever after (zero steady-state cost).
func TestResolveTimeoutFallback(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	old := responseTimeout
	responseTimeout = 30 * time.Millisecond
	t.Cleanup(func() { responseTimeout = old })

	emitted := 0
	SetQueryFunc(func() { emitted++ })
	Query()
	if emitted != 1 {
		t.Fatalf("Query emits the probe once, got %d", emitted)
	}
	start := time.Now()
	if _, _, ok := Resolve(); ok {
		t.Fatal("no answer inside the window → the miss (the 9x18 fallback)")
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("Resolve must wait out the response window (~30ms); elapsed %s", elapsed)
	}
	// steady state: the window is spent — the miss is instant now.
	start = time.Now()
	if _, _, ok := Resolve(); ok {
		t.Fatal("the settled miss")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("the post-window miss must be instant (zero behavioral change), took %s", elapsed)
	}
}

// TestResolveAnswerInsideWindow — an answer landing DURING the wait wakes
// Resolve immediately with the metric (no waiting out the window).
func TestResolveAnswerInsideWindow(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	old := responseTimeout
	responseTimeout = 500 * time.Millisecond
	t.Cleanup(func() { responseTimeout = old })
	SetQueryFunc(func() {})
	Query()
	go func() {
		time.Sleep(20 * time.Millisecond)
		report(32, 16) // the wrapper's wire-order landing: h=32, w=16
	}()
	start := time.Now()
	w, h, ok := Resolve()
	if !ok || w != 16 || h != 32 {
		t.Fatalf("Resolve = (%d,%d,%v), want (16,32,true)", w, h, ok)
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("the answer must wake Resolve, not wait out the window: %s", elapsed)
	}
}

// TestLateAnswerUpdatesCurrent — the window expired (the fallback was
// taken), but a LATE answer still lands in Current: the NEXT shot uses
// the real metric.
func TestLateAnswerUpdatesCurrent(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	old := responseTimeout
	responseTimeout = 20 * time.Millisecond
	t.Cleanup(func() { responseTimeout = old })
	SetQueryFunc(func() {})
	Query()
	if _, _, ok := Resolve(); ok {
		t.Fatal("the first Resolve must miss (no answer inside the window)")
	}
	report(32, 16) // the late answer
	if w, h, ok := Current(); !ok || w != 16 || h != 32 {
		t.Fatalf("the late answer updates Current: (%d,%d,%v), want (16,32,true)", w, h, ok)
	}
	start := time.Now()
	if w, h, ok := Resolve(); !ok || w != 16 || h != 32 {
		t.Fatalf("the next Resolve serves the late metric: (%d,%d,%v)", w, h, ok)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("a landed metric never waits: %s", elapsed)
	}
}

// TestRequerySkipsTheBootWindow — the app routes the renderer's boot size
// through the same WindowSizeMsg path as real resizes: the FIRST Requery
// is a no-op (the startup Query covers t=0); every later one re-emits.
func TestRequerySkipsTheBootWindow(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	emitted := 0
	SetQueryFunc(func() { emitted++ })
	Requery()
	if emitted != 0 {
		t.Fatalf("the boot WindowSizeMsg never re-queries (the startup probe covers it): %d emits", emitted)
	}
	Requery()
	Requery()
	if emitted != 2 {
		t.Fatalf("every WindowSizeMsg after the first re-arms the probe: %d emits, want 2", emitted)
	}
}

// TestQueryWithoutEmitterOpensNoWindow — with no emitter installed
// (suites, uishot) Query is a pure no-op: NO response window opens, so
// Resolve can never block on a wait nobody can answer.
func TestQueryWithoutEmitterOpensNoWindow(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	old := responseTimeout
	responseTimeout = time.Hour // a block would be fatal-slow: the probe must not open one
	t.Cleanup(func() { responseTimeout = old })
	Query() // no emitter installed
	done := make(chan struct{})
	go func() { Resolve(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Resolve blocked on a window no emitter could answer")
	}
	if _, _, ok := Resolve(); ok {
		t.Fatal("no emitter, no answer → the miss")
	}
}

// TestSetForShotPinAndRestore — the viewport-math suites' seam: the pin
// serves Current AND Resolve instantly (no window math), the restore
// reverts to the pre-pin state.
func TestSetForShotPinAndRestore(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	unpin := SetForShot(16, 32)
	if w, h, ok := Current(); !ok || w != 16 || h != 32 {
		t.Fatalf("the pin serves Current: (%d,%d,%v), want (16,32,true)", w, h, ok)
	}
	start := time.Now()
	if w, h, ok := Resolve(); !ok || w != 16 || h != 32 {
		t.Fatalf("the pin serves Resolve: (%d,%d,%v)", w, h, ok)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("a pinned metric never waits: %s", elapsed)
	}
	unpin()
	if _, _, ok := Current(); ok {
		t.Fatal("the restore reverts to the pre-pin (no metric) state")
	}
}

// TestResetForShotIsolation — the full swap: an emitter + metric + window
// installed inside the reset vanish on restore (cross-package suites
// driving Query must never leak state into the zero-state tests).
func TestResetForShotIsolation(t *testing.T) {
	outer := ResetForShot()
	t.Cleanup(outer)
	emitted := 0
	SetQueryFunc(func() { emitted++ })
	Query()
	report(8, 4)
	if emitted != 1 {
		t.Fatalf("setup: one emit, got %d", emitted)
	}
	inner := ResetForShot()
	if _, _, ok := Current(); ok {
		t.Fatal("the reset wipes the metric")
	}
	Query() // no emitter in the clean state — the swapped-out one must NOT fire
	if emitted != 1 {
		t.Fatalf("the swapped-out emitter fired inside the reset: %d emits, want 1", emitted)
	}
	inner()
	if w, h, ok := Current(); !ok || w != 4 || h != 8 {
		t.Fatalf("the restore puts the pre-reset state back: (%d,%d,%v), want (4,8,true)", w, h, ok)
	}
	Query() // the restored emitter is live again
	if emitted != 2 {
		t.Fatalf("the restore puts the emitter back: %d emits, want 2", emitted)
	}
}

// TestReportRejectsDegenerate — a zero component is snipped by the
// wrapper but NEVER stored as a metric (a terminal sending h=0/w=0 is
// lying; the fallback stands).
func TestReportRejectsDegenerate(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	report(0, 16)
	report(32, 0)
	if _, _, ok := Current(); ok {
		t.Fatal("zero-component answers never become the metric")
	}
	report(32, 16)
	if w, h, ok := Current(); !ok || w != 16 || h != 32 {
		t.Fatalf("the real answer still lands: (%d,%d,%v)", w, h, ok)
	}
}
