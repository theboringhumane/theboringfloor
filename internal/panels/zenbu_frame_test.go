// zenbu_frame_test.go — the frame-splice emission seam's byte-level
// contracts (zenbu_frame.go): the wrapper passes renderer bytes through
// unchanged and appends, after each flush, cursor-save + CUP (1-based,
// the registry origin + the pane-local offset) + the cached verbatim APC
// + cursor-restore per live image; an empty registry passes through
// only; an active→empty transition flushes exactly one a=d per
// previously-emitted id; re-emission is idempotent (the cached string,
// never a re-encode); deletes + cursor discipline pair up byte-for-byte;
// DirectEmit serializes with flushes; Finish sweeps + seals.
package panels

import (
	"strings"
	"testing"
)

// frameTestImage — one deterministic registry image.
func frameTestImage(id uint32, ox, oy int, frame string) ZenbuFrameImage {
	return ZenbuFrameImage{OfficeID: id, OX: ox, OY: oy, Frame: frame}
}

// TestZenbuFrameWriterPassthroughEmptyRegistry — flush in → the exact
// same bytes out, NOTHING appended (floor/text-lane/no-lane posture).
func TestZenbuFrameWriterPassthroughEmptyRegistry(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	flush := "\x1b[2J\x1b[1;1Hsome renderer bytes\x1b[0m"
	if n, err := w.Write([]byte(flush)); err != nil || n != len(flush) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(flush))
	}
	if out.String() != flush {
		t.Fatalf("an empty registry passes the flush through byte-identically:\n got %q\nwant %q", out.String(), flush)
	}
}

// TestZenbuFrameWriterSplice — one live image at pane-local (5,2) with
// the registry origin (0,3): the flush is followed by DECSC + CUP(6;6)
// (1-based: row 3+2+1, col 0+5+1) + the verbatim APC + DECRC.
func TestZenbuFrameWriterSplice(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apc := "\x1b_Ga=T,t=d,q=2,C=1,i=deadbeef,f=100;UEFZTE9BRA==\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(0xdeadbeef, 5, 2, apc)}, nil)
	flush := "FRAME-BYTES"
	if _, err := w.Write([]byte(flush)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := flush + "\x1b7\x1b[6;6H" + apc + "\x1b8"
	if out.String() != want {
		t.Fatalf("the splice lands after the flush at the absolute cell:\n got %q\nwant %q", out.String(), want)
	}
}

// TestZenbuFrameWriterDeleteOnClear — the active→empty transition: the
// next flush carries ONE a=d for the previously-emitted id and NO APC;
// the flush after that passes through clean (the delete fires ONCE).
func TestZenbuFrameWriterDeleteOnClear(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apc := "\x1b_Ga=T,t=d,q=2,C=1,i=0000002a,f=100;QUJD\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(42, 0, 0, apc)}, nil)
	_, _ = w.Write([]byte("F1"))
	out.Reset()
	reg.Clear()
	_, _ = w.Write([]byte("F2"))
	want := "F2" + kittyDeleteFrame(42)
	if out.String() != want {
		t.Fatalf("the registry clear flushes exactly one a=d:\n got %q\nwant %q", out.String(), want)
	}
	out.Reset()
	_, _ = w.Write([]byte("F3"))
	if out.String() != "F3" {
		t.Fatalf("the delete fired once (no stale a=d on the next flush): %q", out.String())
	}
}

// TestZenbuFrameWriterDrainedDeletes — the store-queue deletes riding
// the registry publish land AHEAD of the placements (delete-then-add
// convergence inside one flush), and the emitted diff never double-
// deletes a re-published id.
func TestZenbuFrameWriterDrainedDeletes(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apcOld := "\x1b_Ga=T,t=d,q=2,C=1,i=00000007,f=100;T0xE\x1b\\"
	apcNew := "\x1b_Ga=T,t=d,q=2,C=1,i=00000008,f=100;TkVX\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(7, 0, 0, apcOld)}, nil)
	_, _ = w.Write([]byte("F1"))
	out.Reset()
	// the child replaced the frame: the store drained the stale id's
	// delete, the registry carries the new image.
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(8, 0, 0, apcNew)}, []uint32{7})
	_, _ = w.Write([]byte("F2"))
	want := "F2" + kittyDeleteFrame(7) + "\x1b7\x1b[4;1H" + apcNew + "\x1b8"
	if out.String() != want {
		t.Fatalf("the drained delete rides ahead of the placement:\n got %q\nwant %q", out.String(), want)
	}
}

// TestZenbuFrameWriterBandwidthSkip — the SAME registry state across
// two erase-free flushes re-emits NOTHING (the terminal already holds
// the image — the wave-86 bandwidth ruling); a flush carrying an
// erase-display/erase-line sequence re-emits EVERYTHING (ED/EL could
// have clobbered terminal-side image state); deletes ALWAYS flush, even
// under the skip. The emitted bytes stay the cached APC string, never a
// re-encode, and cursor save/restore pairs 1:1.
func TestZenbuFrameWriterBandwidthSkip(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apc := "\x1b_Ga=T,t=d,q=2,C=1,i=00000009,f=100;SURFTVA==\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(9, 1, 1, apc)}, nil)
	splice := "\x1b7\x1b[5;2H" + apc + "\x1b8"

	// flush 1: the first emission splices; flush 2 (identical registry,
	// no erase) splices NOTHING.
	_, _ = w.Write([]byte("A"))
	_, _ = w.Write([]byte("B"))
	if got := out.String(); got != "A"+splice+"B" {
		t.Fatalf("an unchanged image skips the re-emit on an erase-free flush:\n got %q\nwant %q", got, "A"+splice+"B")
	}

	// an ED flush (the resize shape) re-emits EVERYTHING live; the next
	// erase-free flush skips again.
	out.Reset()
	_, _ = w.Write([]byte("\x1b[2J\x1b[1;1H"))
	if got, want := out.String(), "\x1b[2J\x1b[1;1H"+splice; got != want {
		t.Fatalf("an ED flush re-emits every live image:\n got %q\nwant %q", got, want)
	}
	out.Reset()
	_, _ = w.Write([]byte("C"))
	if got := out.String(); got != "C" {
		t.Fatalf("the erase-free flush after the repair skips again: %q", got)
	}

	// an EL flush (erase-line) re-emits too (a cleared row tail could
	// have clobbered the image's cells).
	out.Reset()
	_, _ = w.Write([]byte("\x1b[10;1H\x1b[K"))
	if got, want := out.String(), "\x1b[10;1H\x1b[K"+splice; got != want {
		t.Fatalf("an EL flush re-emits every live image:\n got %q\nwant %q", got, want)
	}

	// deletes ALWAYS flush, even when nothing else emits: the
	// active→empty transition under an erase-free flush.
	out.Reset()
	reg.Clear()
	_, _ = w.Write([]byte("D"))
	if got, want := out.String(), "D"+kittyDeleteFrame(9); got != want {
		t.Fatalf("deletes always flush (skip or not):\n got %q\nwant %q", got, want)
	}

	// cursor save/restore paired 1:1 across the whole drive.
	if saves, restores := strings.Count(splice, "\x1b7"), strings.Count(splice, "\x1b8"); saves != 1 || restores != 1 {
		t.Fatalf("the splice's cursor discipline pairs 1:1: %d vs %d", saves, restores)
	}
}

// TestZenbuFrameWriterSkipTable — the skip's decision table, byte-asserted
// per row: (registry change, flush erase) → emitted suffix.
func TestZenbuFrameWriterSkipTable(t *testing.T) {
	apcA := "\x1b_Ga=T,t=d,q=2,C=1,i=0000000a,f=100;QQ==\x1b\\"
	apcA2 := "\x1b_Ga=T,t=d,q=2,C=1,i=0000000a,f=100;Qg==\x1b\\" // same id, NEW payload bytes
	spliceAt := func(row, col int, apc string) string {
		return "\x1b7\x1b[" + itoa(row) + ";" + itoa(col) + "H" + apc + "\x1b8"
	}
	cases := []struct {
		name   string
		flush  string // the renderer bytes riding the flush
		mutate func(*ZenbuFrameRegistry)
		want   string // the splice suffix after the flush
	}{
		{"identical + clean flush", "F", nil, ""},
		{"identical + ED", "\x1b[2J", nil, spliceAt(4, 6, apcA)},
		{"identical + EL", "\x1b[0K", nil, spliceAt(4, 6, apcA)},
		{"identical + ED param", "\x1b[3J", nil, spliceAt(4, 6, apcA)},
		{"moved origin + clean flush", "F", func(r *ZenbuFrameRegistry) {
			r.Publish(true, 0, 4, []ZenbuFrameImage{frameTestImage(10, 5, 0, apcA)}, nil)
		}, spliceAt(5, 6, apcA)},
		{"changed APC bytes + clean flush", "F", func(r *ZenbuFrameRegistry) {
			r.Publish(true, 0, 4, []ZenbuFrameImage{frameTestImage(10, 5, 0, apcA2)}, nil)
		}, spliceAt(5, 6, apcA2)},
		{"new image joins + clean flush", "F", func(r *ZenbuFrameRegistry) {
			r.Publish(true, 0, 4, []ZenbuFrameImage{
				frameTestImage(10, 5, 0, apcA2),
				frameTestImage(11, 1, 1, apcA),
			}, nil)
		}, spliceAt(6, 2, apcA)}, // ONLY the newcomer emits (id 11 at row 4+1+1, col 0+1+1)
		{"queued delete of a LIVE id re-places (converges)", "F", func(r *ZenbuFrameRegistry) {
			r.Publish(true, 0, 4, []ZenbuFrameImage{
				frameTestImage(10, 5, 0, apcA2),
				frameTestImage(11, 1, 1, apcA),
			}, []uint32{10})
		}, kittyDeleteFrame(10) + spliceAt(5, 6, apcA2)}, // the delete never swallows the re-place
		{"drop one image + clean flush", "F", func(r *ZenbuFrameRegistry) {
			r.Publish(true, 0, 4, []ZenbuFrameImage{frameTestImage(11, 1, 1, apcA)}, nil)
		}, kittyDeleteFrame(10)}, // the diff delete; the survivor skips
	}
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(10, 5, 0, apcA)}, nil)
	_, _ = w.Write([]byte("seed")) // the first emission (always splices)
	seed := "seed" + spliceAt(4, 6, apcA)
	if got := out.String(); got != seed {
		t.Fatalf("the seed flush splices the first emission:\n got %q\nwant %q", got, seed)
	}
	for _, tc := range cases {
		out.Reset()
		if tc.mutate != nil {
			tc.mutate(reg)
		}
		if _, err := w.Write([]byte(tc.flush)); err != nil {
			t.Fatalf("%s: Write: %v", tc.name, err)
		}
		if got, want := out.String(), tc.flush+tc.want; got != want {
			t.Fatalf("%s:\n got %q\nwant %q", tc.name, got, want)
		}
	}
}

// TestZenbuFrameWriterTwoImagesOrdering — two live images splice in the
// registry's paint order, each with its own CUP (the absolute math per
// image), and a clear deletes BOTH (sorted by id — byte-deterministic).
func TestZenbuFrameWriterTwoImagesOrdering(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apcA := "\x1b_Ga=T,t=d,q=2,C=1,i=0000000a,f=100;QQ==\x1b\\"
	apcB := "\x1b_Ga=T,t=d,q=2,C=1,i=00000014,f=32,o=z,s=8,v=8;Qg==\x1b\\"
	reg.Publish(true, 2, 10, []ZenbuFrameImage{
		frameTestImage(10, 4, 0, apcA),
		frameTestImage(20, 0, 6, apcB),
	}, nil)
	_, _ = w.Write([]byte("F"))
	want := "F" +
		"\x1b7\x1b[11;7H" + apcA + "\x1b8" + // row 10+0+1, col 2+4+1
		"\x1b7\x1b[17;3H" + apcB + "\x1b8" // row 10+6+1, col 2+0+1
	if out.String() != want {
		t.Fatalf("two images splice in paint order at their own cells:\n got %q\nwant %q", out.String(), want)
	}
	out.Reset()
	reg.Clear()
	_, _ = w.Write([]byte("G"))
	want = "G" + kittyDeleteFrame(10) + kittyDeleteFrame(20)
	if out.String() != want {
		t.Fatalf("the clear deletes both ids (sorted, deterministic):\n got %q\nwant %q", out.String(), want)
	}
}

// TestZenbuFrameWriterDirectEmitAndFinish — the lane-lifecycle delete
// path: DirectEmit's bytes pass straight through (serialized with
// flushes), and Finish sweeps one a=d per still-emitted id then seals
// the wrapper (passthrough-only, idempotent).
func TestZenbuFrameWriterDirectEmitAndFinish(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apc := "\x1b_Ga=T,t=d,q=2,C=1,i=00000063,f=100;Wg==\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(99, 0, 0, apc)}, nil)
	_, _ = w.Write([]byte("F1"))
	w.DirectEmit("\x1b_Ga=d,d=A,q=2;\x1b\\")
	if got := out.String(); !strings.HasSuffix(got, "\x1b_Ga=d,d=A,q=2;\x1b\\") {
		t.Fatalf("DirectEmit lands serialized after the flush+splice: %q", got)
	}
	out.Reset()
	reg.Clear() // the lane's Close already cleared; Finish still sweeps the emitted set
	w.Finish()
	if got := out.String(); got != kittyDeleteFrame(99) {
		t.Fatalf("Finish sweeps the still-emitted id once: %q", got)
	}
	// sealed: no more splices, no second sweep.
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(100, 0, 0, apc)}, nil)
	out.Reset()
	_, _ = w.Write([]byte("F2"))
	w.Finish()
	if got := out.String(); got != "F2" {
		t.Fatalf("post-Finish the wrapper is passthrough-only + idempotent: %q", got)
	}
}

// TestZenbuFrameWriterChatMediaRegion — the TWO regions are independent:
// the chat-media publish's ABSOLUTE cells splice after the browser
// region's origin-resolved images (publish order), a chat-only clear
// a=d's ONLY the chat ids (the lane keeps painting), and a browser-only
// Clear leaves the chat images untouched.
func TestZenbuFrameWriterChatMediaRegion(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	laneAPC := "\x1b_Ga=T,t=d,q=2,C=1,i=00000001,f=32,o=z,s=8,v=8;TEFORQ==\x1b\\"
	chatAPC := "\x1b_Ga=T,t=d,f=100,i=abcdef01,q=2;Q0hBVA==\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(1, 2, 1, laneAPC)}, nil)
	reg.PublishChatMedia([]ZenbuFrameImage{frameTestImage(0xabcdef01, 60, 5, chatAPC)}) // ABSOLUTE cell
	_, _ = w.Write([]byte("F"))
	want := "F" +
		"\x1b7\x1b[5;3H" + laneAPC + "\x1b8" + // the lane: origin (0,3) + local (2,1) → CUP(5;3)
		"\x1b7\x1b[6;61H" + chatAPC + "\x1b8" // the chat preview: the absolute cell → CUP(6;61)
	if got := out.String(); got != want {
		t.Fatalf("both regions splice at their own cells in publish order:\n got %q\nwant %q", got, want)
	}

	// the chat region clears (tab flip / scroll-off): the wrapper's diff
	// a=d's ONLY the chat id — the lane's image skips (unchanged).
	out.Reset()
	reg.PublishChatMedia(nil)
	_, _ = w.Write([]byte("G"))
	if got, want := out.String(), "G"+kittyDeleteFrame(0xabcdef01); got != want {
		t.Fatalf("the chat-region clear a=d's only the chat id:\n got %q\nwant %q", got, want)
	}

	// the lane's Clear leaves the chat region untouched: re-publish the
	// chat preview, Clear (browser), flush — the chat image skips (the
	// terminal holds it), nothing emits.
	reg.PublishChatMedia([]ZenbuFrameImage{frameTestImage(0xabcdef01, 60, 5, chatAPC)})
	_, _ = w.Write([]byte("H")) // the chat preview re-places (it was deleted above)
	out.Reset()
	reg.Clear() // the BROWSER region only
	_, _ = w.Write([]byte("I"))
	if got, want := out.String(), "I"+kittyDeleteFrame(1); got != want {
		t.Fatalf("the browser Clear never touches the chat region:\n got %q\nwant %q", got, want)
	}
	out.Reset()
	_, _ = w.Write([]byte("J")) // the chat image is unchanged — the skip holds
	if got := out.String(); got != "J" {
		t.Fatalf("the surviving chat image skips the re-emit: %q", got)
	}
}

// TestZenbuFrameWriterChatScrollCUP — the scroll contract, byte-for-byte:
// a chat preview re-published at a NEW absolute row (the transcript
// scrolled by one) re-emits at the new CUP (kitty's same-id atomic
// replace), and a preview that scrolls OUT of the visible window (absent
// from the publish) is a=d'd by the emitted-set diff.
func TestZenbuFrameWriterChatScrollCUP(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	chatAPC := "\x1b_Ga=T,t=d,f=100,i=00c0ffee,q=2;U0NST0xM\x1b\\"
	reg.PublishChatMedia([]ZenbuFrameImage{frameTestImage(0x00c0ffee, 60, 10, chatAPC)})
	_, _ = w.Write([]byte("S0"))
	if got, want := out.String(), "S0"+"\x1b7\x1b[11;61H"+chatAPC+"\x1b8"; got != want {
		t.Fatalf("the initial publish splices at CUP(11;61):\n got %q\nwant %q", got, want)
	}
	// scroll by one row: the SAME image at absolute row 9 now.
	out.Reset()
	reg.PublishChatMedia([]ZenbuFrameImage{frameTestImage(0x00c0ffee, 60, 9, chatAPC)})
	_, _ = w.Write([]byte("S1"))
	if got, want := out.String(), "S1"+"\x1b7\x1b[10;61H"+chatAPC+"\x1b8"; got != want {
		t.Fatalf("the scroll re-emits at the moved CUP byte-for-byte:\n got %q\nwant %q", got, want)
	}
	// scroll past: the publish drops the preview — the diff a=d's it.
	out.Reset()
	reg.PublishChatMedia(nil)
	_, _ = w.Write([]byte("S2"))
	if got, want := out.String(), "S2"+kittyDeleteFrame(0x00c0ffee); got != want {
		t.Fatalf("the scrolled-off preview is a=d'd:\n got %q\nwant %q", got, want)
	}
	out.Reset()
	_, _ = w.Write([]byte("S3")) // the delete fired ONCE
	if got := out.String(); got != "S3" {
		t.Fatalf("no stale a=d after the scroll-off: %q", got)
	}
}

// TestZenbuFrameWriterFd — the x/term File contract: Fd delegates to the
// underlying writer's (the colorprofile/ttyOutput assertions survive the
// wrap); a writer without one yields the invalid fd.
func TestZenbuFrameWriterFd(t *testing.T) {
	reg := &ZenbuFrameRegistry{}
	var out strings.Builder
	w := NewZenbuFrameWriter(&out, reg)
	if w.Fd() != ^uintptr(0) {
		t.Fatalf("a non-file writer's Fd = %#x, want the invalid fd", w.Fd())
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close is a deliberate no-op: %v", err)
	}
	if _, err := w.Read(make([]byte, 1)); err == nil {
		t.Fatal("a non-readable writer reads EOF")
	}
}
