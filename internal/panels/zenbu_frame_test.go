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

// TestZenbuFrameWriterIdempotentReemit — the SAME registry state across
// two flushes re-emits the byte-identical splice (kitty dedupes by id;
// the cached APC string is never re-encoded), and cursor save/restore
// pairs 1:1.
func TestZenbuFrameWriterIdempotentReemit(t *testing.T) {
	var out strings.Builder
	reg := &ZenbuFrameRegistry{}
	w := NewZenbuFrameWriter(&out, reg)
	apc := "\x1b_Ga=T,t=d,q=2,C=1,i=00000009,f=100;SURFTVA==\x1b\\"
	reg.Publish(true, 0, 3, []ZenbuFrameImage{frameTestImage(9, 1, 1, apc)}, nil)
	_, _ = w.Write([]byte("A"))
	_, _ = w.Write([]byte("B"))
	got := out.String()
	splice := "\x1b7\x1b[5;2H" + apc + "\x1b8"
	if got != "A"+splice+"B"+splice {
		t.Fatalf("every flush re-emits the byte-identical splice:\n%q", got)
	}
	if saves, restores := strings.Count(got, "\x1b7"), strings.Count(got, "\x1b8"); saves != restores || saves != 2 {
		t.Fatalf("cursor save/restore pairs 1:1: %d vs %d", saves, restores)
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
