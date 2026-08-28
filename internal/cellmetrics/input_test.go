package cellmetrics

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"
)

// chunkReader yields the payload as the caller's EXACT chunks, then EOF
// (the split sweep's two-write source).
type chunkReader struct {
	chunks [][]byte
	i      int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.i >= len(c.chunks) {
		return 0, io.EOF
	}
	n := copy(p, c.chunks[c.i])
	c.i++
	return n, nil
}

// trickleReader yields the payload n bytes per Read (the every-offset
// split's worst-case chunker: n=1 gives every byte its own Read).
type trickleReader struct {
	data []byte
	n    int
}

func (t *trickleReader) Read(p []byte) (int, error) {
	if len(t.data) == 0 {
		return 0, io.EOF
	}
	k := t.n
	if k > len(t.data) {
		k = len(t.data)
	}
	copy(p, t.data[:k])
	t.data = t.data[k:]
	return k, nil
}

// readAll drains the wrapper to EOF with the given read-buffer size
// (small buffers force the Read-side chunking contract), returning the
// passthrough bytes.
func readAll(t *testing.T, r *Reader, bufSize int) []byte {
	t.Helper()
	var out []byte
	buf := make([]byte, bufSize)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
	}
}

// pinHold — the sweep tests drive chunks back-to-back through the pump;
// pinning the hold window absurdly high keeps the release timer out of
// the determinism story (the timer path has its own test).
func pinHold(t *testing.T, d time.Duration) {
	t.Helper()
	old := holdTimeout
	holdTimeout = d
	t.Cleanup(func() { holdTimeout = old })
}

// TestResponseSplitEveryOffset — the response rides a stream of ordinary
// bytes, split across reads at EVERY possible byte offset (plus the
// 1-byte trickle that exercises all offsets at once): the stream passes
// through minus the response, byte-identically, and the metric lands.
func TestResponseSplitEveryOffset(t *testing.T) {
	pinHold(t, 30*time.Second)
	const resp = "\x1b[6;32;16t"
	for split := 0; split <= len(resp); split++ {
		t.Run(fmt.Sprintf("split@%d", split), func(t *testing.T) {
			restore := ResetForShot()
			t.Cleanup(restore)
			src := &chunkReader{chunks: [][]byte{
				[]byte("pre" + resp[:split]),
				[]byte(resp[split:] + "post"),
			}}
			out := readAll(t, WrapInput(src), 4096)
			if got, want := string(out), "prepost"; got != want {
				t.Fatalf("passthrough = %q, want %q (the response snipped out)", got, want)
			}
			if w, h, ok := Current(); !ok || w != 16 || h != 32 {
				t.Fatalf("the metric landed: (%d,%d,%v), want (16,32,true)", w, h, ok)
			}
		})
	}
	t.Run("trickle-1-byte", func(t *testing.T) {
		restore := ResetForShot()
		t.Cleanup(restore)
		src := &trickleReader{data: []byte("pre" + resp + "post"), n: 1}
		out := readAll(t, WrapInput(src), 4096)
		if got, want := string(out), "prepost"; got != want {
			t.Fatalf("passthrough = %q, want %q", got, want)
		}
		if w, h, ok := Current(); !ok || w != 16 || h != 32 {
			t.Fatalf("the metric landed: (%d,%d,%v), want (16,32,true)", w, h, ok)
		}
	})
	// the same sweep with a TINY read buffer — the Read-side chunking
	// contract (out drains in len(p)-sized pieces, order preserved).
	t.Run("tiny-read-buffer", func(t *testing.T) {
		restore := ResetForShot()
		t.Cleanup(restore)
		src := &trickleReader{data: []byte("pre" + resp + "post"), n: 1}
		out := readAll(t, WrapInput(src), 3)
		if got, want := string(out), "prepost"; got != want {
			t.Fatalf("passthrough = %q, want %q", got, want)
		}
	})
}

// TestPassThroughByteIdentity — the fuzz-ish table: nasty inputs that
// must come through BYTE-IDENTICALLY (no snip, no reorder, no swallow),
// the registry untouched.
func TestPassThroughByteIdentity(t *testing.T) {
	pinHold(t, 30*time.Second)
	everyByte := make([]byte, 256)
	for i := range everyByte {
		everyByte[i] = byte(i) // contains a bare ESC (0x1b) followed by 0x1c — a non-match
	}
	cases := []struct {
		name string
		in   []byte
	}{
		{"plain ascii", []byte("the quick brown fox /open https://a.dev/\r")},
		{"every byte", everyByte},
		{"esc alone mid-stream", []byte("q\x1b")},
		{"ss3 up arrow", []byte("\x1bOw")},
		{"csi up arrow", []byte("\x1b[Aw")},
		{"dsr query echo", []byte("\x1b[6n")},                             // ESC [ 6 n — the closest collider's head
		{"cpr reply row 6", []byte("\x1b[6;20R")},                         // ESC [ 6 ; 2 0 R — digits, then NOT ';'
		{"cpr reply row 6 col 7", []byte("\x1b[6;7Rj")},                   // digits, R, trailing key
		{"osc bg reply", []byte("\x1b]11;rgb:0000/0000/0000\x07")},        // the theme probe's answer
		{"kitty apc frame", []byte("\x1b_Ga=T,t=d,f=100;QUJDRA==\x1b\\")}, // the shot lane's own bytes
		{"partial response at EOF", []byte("abc\x1b[6;32;16")},            // never completed — flushed on EOF
		{"bare prefix tail at EOF", []byte("\x1b[6;")},                    // the prefix itself, flushed on EOF
		{"empty digits", []byte("\x1b[6;;16t")},                           // ';' where a digit must lead — non-match
		{"over-cap response", []byte("\x1b[6;9999;9999t")},                // 14 bytes > the 12-byte window: never snipped
		{"over-cap digits", []byte("\x1b[6;12345678;9t")},                 // absurd metric bails the digit run
		{"utf8 multibyte", []byte("héllo ✦ wörld — em · dashes")},
		{"esc then keys", []byte("\x1bqq/open\r")},
		{"two streams of esc", []byte("\x1b\x1b\x1b[6n\x1b[6;1R")},
	}
	for _, bufSize := range []int{1, 3, 4096} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/buf%d", tc.name, bufSize), func(t *testing.T) {
				restore := ResetForShot()
				t.Cleanup(restore)
				out := readAll(t, WrapInput(&trickleReader{data: tc.in, n: 2}), bufSize)
				if !bytes.Equal(out, tc.in) {
					t.Fatalf("byte-identity broke:\n in  %q\n out %q", tc.in, out)
				}
				if _, _, ok := Current(); ok {
					t.Fatalf("no metric may land from %q", tc.in)
				}
			})
		}
	}
	// the seeded fuzz leg: random ESC-free noise spliced with random
	// non-matching escape heads — 200 rounds, byte-identity throughout.
	t.Run("seeded-fuzz", func(t *testing.T) {
		restore := ResetForShot()
		t.Cleanup(restore)
		rng := rand.New(rand.NewSource(81))
		for round := 0; round < 200; round++ {
			n := rng.Intn(64)
			payload := make([]byte, 0, n+16)
			for i := 0; i < n; i++ {
				payload = append(payload, byte(rng.Intn(255)+1)) // 0x01..0xFF
				if payload[len(payload)-1] == 0x1b && rng.Intn(2) == 0 {
					payload = append(payload, byte(rng.Intn(255)+1)) // a random byte after ESC
				}
			}
			// splice a random NON-matching escape head (never the
			// response's exact prefix tail).
			head := []byte{'\x1b', '[', byte('0' + rng.Intn(10)), ';', byte('0' + rng.Intn(10)), 'R'}
			if rng.Intn(2) == 0 {
				head = []byte{'\x1b', ']', '1', '1', ';', 'x', '\a'}
			}
			at := rng.Intn(len(payload) + 1)
			payload = append(payload[:at], append(head, payload[at:]...)...)
			out := readAll(t, WrapInput(&trickleReader{data: payload, n: 3}), 7)
			if !bytes.Equal(out, payload) {
				t.Fatalf("round %d: byte-identity broke:\n in  %q\n out %q", round, payload, out)
			}
		}
		if _, _, ok := Current(); ok {
			t.Fatal("no metric may land from the fuzz stream")
		}
	})
}

// TestSnipMatrix — the response IS snipped (and reported) across the
// placement matrix: alone, embedded, back-to-back, at EOF, at the exact
// 12-byte window boundary; the degenerate zero response is snipped but
// never stored.
func TestSnipMatrix(t *testing.T) {
	pinHold(t, 30*time.Second)
	const resp = "\x1b[6;32;16t"
	cases := []struct {
		name     string
		in       string
		wantOut  string
		wantW    int
		wantH    int
		wantHave bool
	}{
		{"alone", resp, "", 16, 32, true},
		{"embedded", "ab" + resp + "cd", "abcd", 16, 32, true},
		{"at stream head", resp + "tail", "tail", 16, 32, true},
		{"at EOF", "lead" + resp, "lead", 16, 32, true},
		{"back to back", resp + resp, "", 16, 32, true},
		{"max window exactly", "\x1b[6;999;999t", "", 999, 999, true},
		{"esc then response", "\x1b" + resp, "\x1b", 16, 32, true},
		{"key bytes around", "\x1b[A" + resp + "q", "\x1b[Aq", 16, 32, true},
		{"zero height snipped, never stored", "\x1b[6;0;16t", "", 0, 0, false},
		{"zero width snipped, never stored", "\x1b[6;32;0t", "", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore := ResetForShot()
			t.Cleanup(restore)
			out := readAll(t, WrapInput(&trickleReader{data: []byte(tc.in), n: 2}), 4096)
			if got := string(out); got != tc.wantOut {
				t.Fatalf("passthrough = %q, want %q", got, tc.wantOut)
			}
			w, h, ok := Current()
			if ok != tc.wantHave || (ok && (w != tc.wantW || h != tc.wantH)) {
				t.Fatalf("Current = (%d,%d,%v), want (%d,%d,%v)", w, h, ok, tc.wantW, tc.wantH, tc.wantHave)
			}
		})
	}
	// the LAST answer wins (the re-arm's whole point: the metric tracks
	// the terminal's CURRENT cell size).
	t.Run("last answer wins", func(t *testing.T) {
		restore := ResetForShot()
		t.Cleanup(restore)
		out := readAll(t, WrapInput(&trickleReader{data: []byte(resp + "\x1b[6;40;20t"), n: 5}), 4096)
		if got := string(out); got != "" {
			t.Fatalf("both responses snipped: %q", got)
		}
		if w, h, ok := Current(); !ok || w != 20 || h != 40 {
			t.Fatalf("the last answer wins: (%d,%d,%v), want (20,40,true)", w, h, ok)
		}
	})
}

// TestHoldTimerReleasesLoneEsc — a lone ESC (the esc KEY) is never held
// hostage: held ONLY while its continuation could still arrive, it
// flushes as ordinary input once the hold window passes.
func TestHoldTimerReleasesLoneEsc(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	pinHold(t, 50*time.Millisecond)
	pr, pw := io.Pipe()
	defer pw.Close()
	r := WrapInput(pr)
	done := make(chan byte, 1)
	go func() { // the reader FIRST (io.Pipe writes block on a reader)
		buf := make([]byte, 1)
		if n, err := r.Read(buf); n == 1 && err == nil {
			done <- buf[0]
		}
	}()
	if _, err := pw.Write([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}
	// the hold window is 50ms: at 10ms the ESC must STILL be held (its
	// continuation could legitimately be in flight). A flush-happy
	// implementation fails here; a slow machine can only pass spuriously.
	select {
	case b := <-done:
		t.Fatalf("the ESC flushed BEFORE the hold window passed (the response would never survive a split): %q", b)
	case <-time.After(10 * time.Millisecond):
	}
	select {
	case b := <-done:
		if b != 0x1b {
			t.Fatalf("delivered %q, want the lone ESC", b)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the lone ESC was held hostage past the hold window")
	}
	if _, _, ok := Current(); ok {
		t.Fatal("a lone ESC is not a metric")
	}
}

// TestHoldTimerContinuationWins — the split response's continuation
// arriving INSIDE the hold window completes the snip (the timer never
// gets to release the prefix).
func TestHoldTimerContinuationWins(t *testing.T) {
	restore := ResetForShot()
	t.Cleanup(restore)
	pinHold(t, 500*time.Millisecond) // the 20ms sleep must beat the hold even loaded
	pr, pw := io.Pipe()
	defer pw.Close()
	r := WrapInput(pr)
	outCh := make(chan []byte, 4)
	go func() { // the reader FIRST (io.Pipe writes block on a reader)
		for {
			buf := make([]byte, 8)
			n, err := r.Read(buf)
			if n > 0 {
				outCh <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()
	if _, err := pw.Write([]byte("\x1b[6;3")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond) // inside the hold window
	if _, err := pw.Write([]byte("2;16t")); err != nil {
		t.Fatal(err)
	}
	// the response completed: snipped + reported; the trailing key passes.
	if _, err := pw.Write([]byte("q")); err != nil {
		t.Fatal(err)
	}
	select {
	case b := <-outCh:
		if string(b) != "q" {
			t.Fatalf("delivered %q, want just the trailing key (the response snipped)", b)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the trailing key never arrived — the response wasn't snipped")
	}
	if w, h, ok := Current(); !ok || w != 16 || h != 32 {
		t.Fatalf("the metric landed: (%d,%d,%v), want (16,32,true)", w, h, ok)
	}
	// and not one byte of the response leaks through.
	select {
	case b := <-outCh:
		t.Fatalf("response bytes leaked through the snip: %q", b)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestMatchCellResponseUnit — the classifier's truth table, driven
// directly (no IO): every proper prefix holds, every deviation releases,
// completions parse in WIRE order (height first).
func TestMatchCellResponseUnit(t *testing.T) {
	// every proper prefix of the canonical response holds.
	const resp = "\x1b[6;32;16t"
	for i := 1; i < len(resp); i++ {
		if _, _, _, st := matchCellResponse([]byte(resp[:i])); st != matchPrefix {
			t.Fatalf("prefix %q: st = %v, want matchPrefix", resp[:i], st)
		}
	}
	// the completion.
	if h, w, n, st := matchCellResponse([]byte(resp)); st != matchFull || h != 32 || w != 16 || n != len(resp) {
		t.Fatalf("full: (%d,%d,%d,%v), want (32,16,%d,matchFull)", h, w, n, st, len(resp))
	}
	// deviations release at the first wrong byte.
	for _, s := range []string{
		"\x1b]6;32;16t",  // OSC head
		"\x1b[5;32;16t",  // the window-pixel reply's cousin (wrong Ps)
		"\x1b[6n",        // DSR
		"\x1b[6;32R",     // CPR row 6
		"\x1b[6;32,16t",  // the wrong separator
		"\x1b[6;32;16s",  // the wrong final byte
		"\x1b[6;x;16t",   // a non-digit leading the run
		"\x1b[6;32;16t+", // (the + is beyond the match — see below)
	} {
		st := func() matchState {
			_, _, _, st := matchCellResponse([]byte(s))
			return st
		}()
		if s == "\x1b[6;32;16t+" {
			if st != matchFull {
				t.Fatalf("%q: the trailing byte is the NEXT event's — the match stands", s)
			}
			continue
		}
		if st != matchNone {
			t.Fatalf("%q: st = %v, want matchNone", s, st)
		}
	}
	// buffer exhaustion INSIDE the digit runs still holds.
	for _, s := range []string{"\x1b[6;3", "\x1b[6;32;", "\x1b[6;32;1"} {
		if _, _, _, st := matchCellResponse([]byte(s)); st != matchPrefix {
			t.Fatalf("%q: st = %v, want matchPrefix (digits could continue)", s, st)
		}
	}
}
