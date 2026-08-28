// input.go — the wrapping input reader: tea.WithInput's seam (the
// symmetric twin of the wave-81 tea.WithOutput frame wrapper). Every byte
// of the terminal's input passes through UNTOUCHED — except the cell-size
// response (CSI 6;<h>;<w>t, the answer to QueryCellSize), which is snipped
// out of the stream BEFORE bubbletea's parser ever sees it and reported
// to the registry.
//
// The response can split across reads, so a proper-prefix tail is HELD
// back undecided — bounded TWICE: the hold never exceeds maxResponseLen
// bytes (a longer "prefix" can never complete inside the window, so its
// oldest byte is released), and a prefix whose continuation never arrives
// within holdTimeout is released as ordinary input (a lone ESC — the esc
// KEY — must never be held hostage; ultraviolet's own 50ms escape
// disambiguation then runs one layer up). A response split slower than
// the hold simply leaks through: ultraviolet parses it as a CellSizeEvent
// the app ignores — the metric misses this round, the next Query's answer
// lands it. Soft degradation, never a stall.
//
// KNOWN EDGE: the snip is paste-blind — a literal CSI 6;<h>;<w>t typed
// inside a bracketed paste is eaten like a real answer (the wrapper sits
// below ultraviolet's paste framing). Real answers arrive in response to
// our own probes, never mid-paste in practice.
package cellmetrics

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// QueryCellSize — the probe main.go writes to the tty: CSI 16t, "report
// cell size in pixels". The terminal answers CSI 6;<h>;<w>t on stdin.
const QueryCellSize = "\x1b[16t"

// maxResponseLen bounds the hold window: the longest response ever
// snipped is "\x1b[6;" + 3 digits + ";" + 3 digits + "t" = 12 bytes (real
// metrics are 1–2 digits; 3 covers any plausible DPR). A partial prefix
// reaching this length without completing can never complete inside the
// window — its oldest byte is released and the rescan continues, so a
// non-matching prefix is NEVER swallowed past 12 bytes.
const maxResponseLen = 12

// holdTimeout — how long a partial response prefix waits for its
// continuation before being released as ordinary input. The terminal's
// real answer arrives in one write, so this guards only pathological
// chunk splits; it sits under ultraviolet's 50ms escape disambiguation so
// a lone ESC keypress's added latency stays imperceptible. A var — the
// house deadline-test idiom (suites shrink it, never the reverse).
var holdTimeout = 20 * time.Millisecond

// Reader wraps the terminal's input source: Read passes every byte
// through byte-identically EXCEPT complete cell-size responses, which are
// snipped and reported. Reads are serial (bubbletea's cancelreader reads
// from exactly one goroutine).
type Reader struct {
	src    io.Reader
	chunks chan readChunk
	once   sync.Once

	// owned by the Read caller's goroutine:
	pending []byte // the undecided proper-prefix tail
	out     []byte // decided passthrough bytes awaiting delivery
	readErr error  // the src's terminal error, delivered after the drain
}

type readChunk struct {
	data []byte
	err  error
}

// WrapInput installs the snipping wrapper over src (main.go: os.Stdin).
func WrapInput(src io.Reader) *Reader {
	return &Reader{src: src, chunks: make(chan readChunk)}
}

// Fd — the x/term File contract, delegated to the underlying source when
// IT has one (os.Stdin in production): bubbletea's raw-mode entry AND its
// ttyInput detection type-assert the input to term.File (tty_unix.go) —
// without Fd the tty NEVER goes raw and keys arrive line-buffered (the
// zenbu frame writer's exact contract, mirrored on the input side). An
// invalid fd when the wrapped source has none (term.IsTerminal reads
// false — exactly like any non-file reader).
func (r *Reader) Fd() uintptr {
	if f, ok := r.src.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Write — the x/term File contract's shape only (bubbletea never writes
// the input): delegated when the underlying source is writable (os.Stdin
// is), EOF otherwise.
func (r *Reader) Write(p []byte) (int, error) {
	if w, ok := r.src.(io.Writer); ok {
		return w.Write(p)
	}
	return 0, io.EOF
}

// Close — the x/term File contract, DELIBERATELY a no-op: bubbletea never
// closes the input, and the underlying fd 0 must survive the program (the
// /session exec-replace rides the same fds).
func (r *Reader) Close() error { return nil }

// pump — the src's dedicated read goroutine: every read (data, error, or
// both) lands on chunks as ONE chunk. Started on the first Read; parks on
// the unbuffered send when no Read is outstanding (input rates are
// human-scale; one fresh buffer per read is noise).
func (r *Reader) pump() {
	for {
		buf := make([]byte, 8192)
		n, err := r.src.Read(buf)
		if n == 0 && err == nil {
			continue // a (0, nil) read carries nothing — never wake the consumer
		}
		c := readChunk{err: err}
		if n > 0 {
			c.data = buf[:n]
		}
		r.chunks <- c
		if err != nil {
			return
		}
	}
}

// Read implements io.Reader: decided passthrough bytes first, then — on
// the src's terminal error — the held prefix flushes BEFORE the error
// (byte-identical to the end), then the error itself. While a proper
// prefix is held undecided, the hold timer races the next chunk.
func (r *Reader) Read(p []byte) (int, error) {
	r.once.Do(func() { go r.pump() })
	for {
		if len(r.out) > 0 {
			n := copy(p, r.out)
			r.out = r.out[n:]
			return n, nil
		}
		if r.readErr != nil {
			if len(r.pending) > 0 {
				// the src ended mid-prefix: the held bytes were ordinary
				// input all along — flush them ahead of the error.
				r.out = append(r.out, r.pending...)
				r.pending = nil
				continue
			}
			return 0, r.readErr
		}
		var hold <-chan time.Time
		if len(r.pending) > 0 {
			hold = time.After(holdTimeout)
		}
		select {
		case c := <-r.chunks:
			if len(c.data) > 0 {
				r.feed(c.data)
			}
			if c.err != nil {
				r.readErr = c.err
			}
		case <-hold:
			// the continuation never came — the held prefix is ordinary
			// input (the esc key's lone ESC is the common case).
			r.out = append(r.out, r.pending...)
			r.pending = nil
		}
	}
}

// feed folds one chunk into the byte stream: snips every complete
// response (reporting each to the registry) and moves decided bytes to
// out. On return, pending holds ONLY a proper-prefix tail shorter than
// maxResponseLen — everything else has been passed through or snipped.
func (r *Reader) feed(data []byte) {
	r.pending = append(r.pending, data...)
	for len(r.pending) > 0 {
		if r.pending[0] != 0x1b {
			// a passthrough run up to the next candidate ESC (or the end)
			i := bytes.IndexByte(r.pending, 0x1b)
			if i < 0 {
				i = len(r.pending)
			}
			r.out = append(r.out, r.pending[:i]...)
			r.pending = r.pending[i:]
			continue
		}
		h, w, consumed, st := matchCellResponse(r.pending)
		switch st {
		case matchFull:
			report(h, w) // snipped: never reaches bubbletea's parser
			r.pending = r.pending[consumed:]
		case matchPrefix:
			if len(r.pending) >= maxResponseLen {
				// can never complete inside the window: release the
				// oldest byte and rescan (overlapping candidates — an
				// ESC right after — survive the rescan).
				r.out = append(r.out, r.pending[0])
				r.pending = r.pending[1:]
				continue
			}
			return // hold the prefix; the next chunk (or the hold timer) resolves it
		default: // matchNone
			r.out = append(r.out, r.pending[0])
			r.pending = r.pending[1:]
		}
	}
}

type matchState int

const (
	matchNone   matchState = iota // definitely not the response — release the ESC
	matchPrefix                   // a proper prefix so far — hold for the continuation
	matchFull                     // a complete response — snip + report
)

// matchCellResponse classifies p (p[0] is ESC) against the response's
// shape: ESC [ 6 ; <height digits> ; <width digits> t — WIRE order,
// height first.
func matchCellResponse(p []byte) (h, w, consumed int, st matchState) {
	const prefix = "\x1b[6;"
	for i := 0; i < len(prefix); i++ {
		if i >= len(p) {
			return 0, 0, 0, matchPrefix
		}
		if p[i] != prefix[i] {
			return 0, 0, 0, matchNone
		}
	}
	i := len(prefix)
	h, n, st := scanDec(p[i:])
	if st != matchFull {
		return 0, 0, 0, st
	}
	i += n
	if i >= len(p) {
		return 0, 0, 0, matchPrefix
	}
	if p[i] != ';' {
		return 0, 0, 0, matchNone
	}
	i++
	w, n, st = scanDec(p[i:])
	if st != matchFull {
		return 0, 0, 0, st
	}
	i += n
	if i >= len(p) {
		return 0, 0, 0, matchPrefix
	}
	if p[i] != 't' {
		return 0, 0, 0, matchNone
	}
	return h, w, i + 1, matchFull
}

// scanDec scans a decimal run at p's head: matchFull with the run's value
// + length when a non-digit terminates it, matchPrefix when the buffer
// ends in (or before) the run, matchNone on a leading non-digit or an
// absurd value (cell px never reach 7 digits — bail so the hold cap
// releases the bytes as ordinary input).
func scanDec(p []byte) (val, n int, st matchState) {
	for n < len(p) && p[n] >= '0' && p[n] <= '9' {
		val = val*10 + int(p[n]-'0')
		n++
		if val > 1<<20 {
			return 0, 0, matchNone
		}
	}
	switch {
	case n == len(p):
		return val, n, matchPrefix // the run could continue with the next chunk
	case n == 0:
		return 0, 0, matchNone
	default:
		return val, n, matchFull
	}
}
