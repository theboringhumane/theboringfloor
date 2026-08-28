// Package cellmetrics learns the REAL terminal's cell pixel size at
// runtime so the browser tab's headless screenshots size their viewport in
// true pixels instead of the 9x18 guess (soft/mis-sized on terminals
// whose cells differ — ghostty's default is ~16x32 device px at 2x DPR).
//
// THE DANCE: main.go emits CSI 16t ("report cell size in pixels") BEFORE
// p.Run and re-arms on every WindowSizeMsg after the boot's first (a
// SIGWINCH batch — font zoom, ctrl+= / ctrl+- in ghostty/kitty, resizes
// the cell's px); the terminal answers CSI 6;<heightPx>;<widthPx>t on
// stdin, and the wrapping input reader (WrapInput — tea.WithInput's seam,
// the symmetric twin of the wave-81 tea.WithOutput frame wrapper) snips
// the response OUT of the byte stream BEFORE bubbletea's parser ever sees
// it and reports (h, w) here. No new message type reaches the app.
//
// NON-ANSWERING TERMINALS (tmux, iTerm, most): Resolve waits out ONE
// responseTimeout window from the FIRST emitted query, then reports the
// miss forever after — the caller's 9x18 fallback with zero blocking in
// steady state. A LATE answer still updates Current (the next shot uses
// it).
package cellmetrics

import (
	"sync"
	"time"
)

// responseTimeout — the ONE wait window: from the first emitted query the
// registry waits this long for the terminal's answer before Resolve
// reports the miss (the caller falls back to 9x18). A var, not a const —
// the house deadline-test idiom (suites shrink it, never the reverse).
var responseTimeout = 150 * time.Millisecond

// regState — the whole learnable state (ResetForShot snapshots/swaps it as
// one value so a pinned metric or a recording emitter never leaks across
// tests).
type regState struct {
	w, h         int           // the latest answer's cell size in pixels
	have         bool          // an answer has landed at least once
	queried      bool          // the first query went out (the window's start)
	firstQueryAt time.Time     // when the first query went out
	requerySeen  bool          // Requery's skip-first latch (the boot WindowSizeMsg)
	queryFn      func()        // the tty emitter (main.go wires DirectEmit)
	answered     chan struct{} // closed when the FIRST answer lands
}

var reg = struct {
	mu sync.Mutex
	st regState
}{st: regState{answered: make(chan struct{})}}

// Current — the latest learned cell pixel size, non-blocking. ok is false
// until the terminal's first answer lands.
func Current() (w, h int, ok bool) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	return reg.st.w, reg.st.h, reg.st.have
}

// Resolve — Current, except within the ONE response window after the
// first query it waits out the window's remainder for the answer (the
// boot probe lands well before the member's first /open, so the wait
// virtually never materializes; a non-answering terminal pays it at most
// once, then misses instantly forever after).
func Resolve() (w, h int, ok bool) {
	reg.mu.Lock()
	if reg.st.have {
		w, h = reg.st.w, reg.st.h
		reg.mu.Unlock()
		return w, h, true
	}
	if !reg.st.queried {
		reg.mu.Unlock()
		return 0, 0, false
	}
	remain := time.Until(reg.st.firstQueryAt.Add(responseTimeout))
	ch := reg.st.answered
	reg.mu.Unlock()
	if remain <= 0 {
		return 0, 0, false
	}
	select {
	case <-ch:
	case <-time.After(remain):
	}
	return Current()
}

// SetQueryFunc installs the tty emitter Query/Requery ride (main.go wires
// the frame writer's DirectEmit so the probe serializes with frame
// flushes — never interleaved mid-frame).
func SetQueryFunc(fn func()) {
	reg.mu.Lock()
	reg.st.queryFn = fn
	reg.mu.Unlock()
}

// Query emits CSI 16t through the installed emitter (a nil emitter is a
// no-op — suites/uishot never wait on a window they cannot answer: the
// response window's start stamps ONLY when an emitter is installed). The
// FIRST emission starts the response window; later emissions just re-arm
// the probe (the answer re-parses into Current whenever it lands).
func Query() {
	reg.mu.Lock()
	fn := reg.st.queryFn
	if fn != nil && !reg.st.queried {
		reg.st.queried = true
		reg.st.firstQueryAt = time.Now()
	}
	reg.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Requery — the resize re-arm: the FIRST call is a no-op (the app routes
// the renderer's boot size through the same WindowSizeMsg case, and the
// startup Query already covers t=0); every later call re-emits the probe.
func Requery() {
	reg.mu.Lock()
	seen := reg.st.requerySeen
	reg.st.requerySeen = true
	reg.mu.Unlock()
	if !seen {
		return
	}
	Query()
}

// report — the input wrapper's answer landing (WIRE order: the response
// is CSI 6;<height>;<width>t). A degenerate zero component is snipped but
// never stored; every real answer updates Current (a late answer included
// — the next shot rides it) and closes the response window's wait.
func report(h, w int) {
	if h <= 0 || w <= 0 {
		return
	}
	reg.mu.Lock()
	reg.st.h, reg.st.w = h, w
	if !reg.st.have {
		reg.st.have = true
		close(reg.st.answered)
	}
	reg.mu.Unlock()
}

// SetForShot pins the learned metric (the viewport-math suites' seam) and
// returns the restore for the PREVIOUS metric triple.
func SetForShot(w, h int) (restore func()) {
	reg.mu.Lock()
	oldW, oldH, oldHave := reg.st.w, reg.st.h, reg.st.have
	reg.st.w, reg.st.h = w, h
	if !reg.st.have {
		reg.st.have = true
		close(reg.st.answered)
	}
	reg.mu.Unlock()
	return func() {
		reg.mu.Lock()
		reg.st.w, reg.st.h, reg.st.have = oldW, oldH, oldHave
		reg.mu.Unlock()
	}
}

// ResetForShot installs a clean registry (no metric, no query window, no
// emitter) and returns the restore for the swapped-out state — the full
// isolation seam for cross-package suites driving Query/Requery.
func ResetForShot() (restore func()) {
	reg.mu.Lock()
	old := reg.st
	reg.st = regState{answered: make(chan struct{})}
	reg.mu.Unlock()
	return func() {
		reg.mu.Lock()
		reg.st = old
		reg.mu.Unlock()
	}
}
