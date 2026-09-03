// browser_lane_kitty_test.go — the kitty passthrough's byte-level
// contract suite (deterministic, hermetic — a scripted stream through
// the splitter, no real child): the APC extraction leaves the grid with
// ONLY the text chrome, chunk chains join across non-aligned boundaries
// and across EVERY possible split point of the reader's buffer, latest
// transmission wins with the stale frame's delete queued, a=d selectors
// map to the office ids, the store stays bounded, malformed sequences
// drop+log-once without ever panicking (or wedging the lane), the
// controller's FrameState surfaces the cached re-emission (office id +
// pane-local origin) for the frame-splice wrapper while the RegionView
// itself stays APC-free, resize retires placements (delete queued +
// SIGWINCH), and Close flushes the deletes through the direct-emit seam
// + clears the registry. (The wrapper's byte-level contracts live in
// zenbu_frame_test.go.)
package panels

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/term"
)

// kittyTestPayload — the deterministic fake frame (content irrelevant —
// the store never image-decodes; the bytes only need to base64/decode
// round-trip).
var kittyTestPayload = []byte("\x89PNG\r\n\x1a\nFAKEKITTYFRAME0123456789abcdefghijklmnopqrstuvwxyz")

func kittyTestB64() string { return base64.StdEncoding.EncodeToString(kittyTestPayload) }

// kittyTestID — the STABLE office-side id for the fake's canonical frame
// (child i=1, no placement key → placement 0): ZenbuOfficeID over (child
// id, placement), NOT the payload (the wave-81 content hash was the
// flicker bug).
func kittyTestID() uint32 { return ZenbuOfficeID(1, 0) }

func kittyTestIDHash8() string { return KittyIDHash8(kittyTestID()) }

// kittyAPC assembles one APC: ESC_G + ctrl + ";" + b64 + ESC\.
func kittyAPC(ctrl, b64 string) string {
	return "\x1b_G" + ctrl + ";" + b64 + "\x1b\\"
}

// kittyScriptedStream — the canonical scripted child: chrome text +
// a cursor move, ONE chunked frame (m=1 × 2, m=0 — the first chunk cut
// at a NON-4-aligned boundary, 7 base64 chars, proving join-then-decode),
// then more chrome. The APC lands with the cursor at (0,1): "\x1b[H"
// homes, "TOOLBAR" paints row 0, "\r\n" drops the cursor to row 1.
func kittyScriptedStream() (stream, payloadB64 string) {
	b64 := kittyTestB64()
	chunk := func(ctrl, payload string) string {
		return kittyAPC(ctrl, payload)
	}
	return "\x1b[2J\x1b[H" +
			"TOOLBAR-ROW" + "\r\n" +
			chunk("a=T,t=d,f=100,i=1,q=2,m=1", b64[:7]) +
			chunk("m=1", b64[7:41]) +
			chunk("m=0", b64[41:]) +
			"\x1b[3;1H" + "page text row",
		b64
}

// newKittyRig — the splitter over a real grid+scrollback pair (the
// production wiring's exact shape).
func newKittyRig(cols, rows int) (*kittyStream, *zenbuImageStore, *term.Grid, *term.Scrollback) {
	g := term.NewGrid(cols, rows)
	sb := term.NewScrollback(0)
	store := newZenbuImageStore()
	ks := newKittyStream(io.MultiWriter(sb, g), g, store)
	return ks, store, g, sb
}

// TestKittyStreamSplitCleanGrid — the scripted stream splits: the grid
// paints ONLY the text chrome (no base64 anywhere, no APC fragment), the
// scrollback retains clean text only, and the store holds ONE placed
// image at the captured (0,1) origin with the FULL joined payload.
func TestKittyStreamSplitCleanGrid(t *testing.T) {
	stream, b64 := kittyScriptedStream()
	ks, store, g, sb := newKittyRig(40, 8)
	if _, err := ks.Write([]byte(stream)); err != nil {
		t.Fatalf("splitter write: %v", err)
	}
	// the grid: text chrome only.
	if got := g.LineText(0); got != "TOOLBAR-ROW" {
		t.Fatalf("row 0 = %q, want the toolbar text", got)
	}
	if got := g.LineText(2); got != "page text row" {
		t.Fatalf("row 2 = %q, want the page text", got)
	}
	for y := 0; y < g.Rows(); y++ {
		lt := g.LineText(y)
		if strings.Contains(lt, b64[:16]) || strings.ContainsAny(lt, "\x1b") {
			t.Fatalf("grid row %d carries APC garbage: %q", y, lt)
		}
	}
	// the retained stream is clean too (the payload NEVER reaches sb).
	if raw := string(sb.Raw()); strings.Contains(raw, "a=T,t=d") || strings.Contains(raw, b64[:16]) {
		t.Fatalf("the scrollback retains APC bytes:\n%q", raw)
	}
	// the store: one image, placed at the cursor's (0,1) at commit time.
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.images) != 1 {
		t.Fatalf("one image expected, got %d", len(store.images))
	}
	im := store.images[1]
	if im == nil {
		t.Fatalf("the image is keyed by the child's id i=1: %v", store.images)
	}
	if im.officeID != kittyTestID() {
		t.Fatalf("office id = %08x, want the STABLE ZenbuOfficeID(1,0) %08x", im.officeID, kittyTestID())
	}
	if !im.placed || im.ox != 0 || im.oy != 1 {
		t.Fatalf("placement = (%d,%d) placed=%v, want (0,1) true", im.ox, im.oy, im.placed)
	}
	if im.b64 != b64 {
		t.Fatalf("the joined payload is the full base64 (%d chars), got %d", len(b64), len(im.b64))
	}
	if im.format != "100" {
		t.Fatalf("format = %q, want 100", im.format)
	}
}

// TestKittyStreamSplitBoundaries — the SAME scripted stream split at
// EVERY possible byte offset: the final grid text, the store image, and
// the drop stats are identical at every boundary (partial APCs across
// reads are the whole point of the incremental parser).
func TestKittyStreamSplitBoundaries(t *testing.T) {
	stream, b64 := kittyScriptedStream()
	wantID := kittyTestID()
	for i := 0; i <= len(stream); i++ {
		ks, store, g, _ := newKittyRig(40, 8)
		if _, err := ks.Write([]byte(stream[:i])); err != nil {
			t.Fatalf("split %d: first write: %v", i, err)
		}
		if _, err := ks.Write([]byte(stream[i:])); err != nil {
			t.Fatalf("split %d: second write: %v", i, err)
		}
		if got := g.LineText(0); got != "TOOLBAR-ROW" {
			t.Fatalf("split %d: row 0 = %q", i, got)
		}
		if got := g.LineText(2); got != "page text row" {
			t.Fatalf("split %d: row 2 = %q", i, got)
		}
		store.mu.Lock()
		im := store.images[1]
		drops := store.drops
		store.mu.Unlock()
		if drops != 0 {
			t.Fatalf("split %d: no drops expected, got %d", i, drops)
		}
		if im == nil || im.officeID != wantID || !im.placed || im.ox != 0 || im.oy != 1 || im.b64 != b64 {
			t.Fatalf("split %d: image = %+v", i, im)
		}
	}
}

// TestKittyStreamBELTerminator — the BEL-terminated APC form splits the
// same way (the grid's own APC rule accepts both terminators).
func TestKittyStreamBELTerminator(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, _, _ := newKittyRig(40, 8)
	ks.Write([]byte("\x1b[1;1H" + "\x1b_Ga=T,t=d,f=100,i=1,q=2;" + b64 + "\x07"))
	store.mu.Lock()
	defer store.mu.Unlock()
	if im := store.images[1]; im == nil || im.officeID != kittyTestID() || !im.placed {
		t.Fatalf("BEL-terminated transmit must land: %+v", im)
	}
}

// TestKittyStreamChainFinalChunkOmitsM — the spec's m-default: a chain
// closed by a chunk WITHOUT an m key still commits (absent m = final).
func TestKittyStreamChainFinalChunkOmitsM(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, _, _ := newKittyRig(40, 8)
	ks.Write([]byte("\x1b[H" + kittyAPC("a=T,t=d,f=100,i=1,m=1", b64[:8]) + kittyAPC("", b64[8:])))
	store.mu.Lock()
	defer store.mu.Unlock()
	im := store.images[1]
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the m-less final chunk closes the chain: %+v", im)
	}
}

// TestKittyStreamProbeIgnored — the child's startup queries (a=q, with
// the t=f/t=s fallback mediums) store nothing and paint nothing.
func TestKittyStreamProbeIgnored(t *testing.T) {
	ks, store, g, _ := newKittyRig(40, 8)
	ks.Write([]byte(
		kittyAPC("i=4207,a=q,t=d,f=24,s=1,v=1", "AAAA") +
			kittyAPC("i=300,a=q,t=f,f=32,s=1,v=1", "L3Zhci9mb2xkZXJz") +
			kittyAPC("i=299,a=q,t=s,f=32,s=1,v=1", "L3B4LXg=") +
			"chrome",
	))
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.images) != 0 {
		t.Fatalf("queries never store: %d images", len(store.images))
	}
	if got := g.LineText(0); got != "chrome" {
		t.Fatalf("the text chrome paints after the probes: %q", got)
	}
	if store.drops != 0 {
		t.Fatalf("queries are not malformed: drops %d", store.drops)
	}
}

// TestKittyStreamMalformed — every malformation drops + logs once,
// NEVER panics, and the surrounding text chrome keeps painting.
func TestKittyStreamMalformed(t *testing.T) {
	b64 := kittyTestB64()
	cases := []struct {
		name   string
		stream string
		drops  int // additional drops expected
	}{
		{"undecodable payload", kittyAPC("a=T,t=d,f=100,i=1", "!!!not-base64!!!") + "after1", 1},
		{"transmit without payload", kittyAPC("a=T,t=d,f=100,i=1", "") + "after2", 1},
		{"transmit without id", kittyAPC("a=T,t=d,f=100", b64) + "after3", 1},
		{"unsupported medium t=f", kittyAPC("a=T,t=f,f=100,i=1", b64) + "after4", 1},
		{"unicode-placeholder U=1", kittyAPC("a=T,t=d,f=32,i=1,U=1", b64) + "after5", 1},
		{"chunk without a chain", kittyAPC("m=0", b64) + "after6", 1},
		{"interrupted chain", kittyAPC("a=T,t=d,f=100,i=1,m=1", b64[:8]) + kittyAPC("a=d,d=a", "") + "after7", 1},
		// the stray-ESC abort: an interleaved CSI corrupts the APC — the
		// body + the base64 TAIL discard to the aborted APC's own
		// terminator (NEVER downstream — the wave-82 leak fix), and the
		// text AFTER the terminator still paints.
		{"stray ESC tail-discard", "\x1b_Ga=T,t=d,f=100,i=1;" + b64[:8] + "\x1b[2J" + b64[8:24] + "\x1b\\" + "after8", 1},
		{"bad key token", kittyAPC("a=T,zzz", b64) + "after9", 1},
		{"placement for unknown id", kittyAPC("a=p,i=77", "") + "after10", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ks, store, g, _ := newKittyRig(40, 8)
			ks.Write([]byte(tc.stream)) // must not panic
			drops, note := store.dropStats()
			if drops != tc.drops {
				t.Fatalf("drops = %d, want %d (note %q)", drops, tc.drops, note)
			}
			if tc.drops > 0 && note == "" {
				t.Fatal("the log-once note records the first reason")
			}
			// the trailing text painted (the lane never wedged).
			found := false
			for y := 0; y < g.Rows(); y++ {
				if strings.Contains(g.LineText(y), "after") {
					found = true
				}
			}
			if !found {
				t.Fatalf("the text after the malformation still paints:\n%s", g.ScreenText())
			}
			// and nothing stored (every case above is a reject).
			store.mu.Lock()
			defer store.mu.Unlock()
			if len(store.images) != 0 && tc.name != "interrupted chain" {
				t.Fatalf("malformed commands never store: %d images", len(store.images))
			}
		})
	}
	// the log-once discipline: first reason pins, later ones only count.
	ks, store, _, _ := newKittyRig(40, 8)
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1", "!!!") + kittyAPC("a=T,t=d,f=100,i=1", "???")))
	drops, note := store.dropStats()
	if drops != 2 || !strings.Contains(note, "undecodable payload") {
		t.Fatalf("first reason pins + the count grows: drops=%d note=%q", drops, note)
	}
}

// TestKittyStreamAPCBodyCap — an unterminated runaway APC drops at the
// cap (the lane never wedges holding a forever-pending payload).
func TestKittyStreamAPCBodyCap(t *testing.T) {
	old := maxKittyAPCBody
	maxKittyAPCBody = 64 // shrunk for the test
	t.Cleanup(func() { maxKittyAPCBody = old })
	ks, store, g, _ := newKittyRig(40, 8)
	ks.Write([]byte("\x1b_Ga=T,t=d,f=100,i=1;" + strings.Repeat("A", 256)))
	if _, note := store.dropStats(); !strings.Contains(note, "cap") {
		t.Fatalf("the cap drop logs once: %q", note)
	}
	// the lane still works afterwards (text + a fresh valid APC land).
	ks.Write([]byte("alive" + kittyAPC("a=T,t=d,f=100,i=1,q=2", kittyTestB64())))
	store.mu.Lock()
	defer store.mu.Unlock()
	if im := store.images[1]; im == nil || !im.placed {
		t.Fatalf("the lane survives the runaway drop: %+v", im)
	}
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("text after the runaway paints: %q", got)
	}
}

// TestZenbuImageStoreLatestWins — FIX A's core regression: the child
// reuses i=1 for EVERY repaint (the wave-82 capture: 34/34 frames) —
// two sequential frames with the same child id + placement emit the SAME
// office id with ZERO a=d between them (kitty's a=T replaces atomically
// terminal-side: no flicker, no empty gap). Only a PLACEMENT-id change
// re-ids (the old placement's delete queues — the terminal never
// composites two generations).
func TestZenbuImageStoreLatestWins(t *testing.T) {
	store := newZenbuImageStore()
	payloadA := []byte("frame generation A")
	payloadB := []byte("frame generation B")
	b64A := base64.StdEncoding.EncodeToString(payloadA)
	b64B := base64.StdEncoding.EncodeToString(payloadB)
	idA := ZenbuOfficeID(1, 0)

	ks, _, g, _ := newKittyRig(40, 8)
	ks.store = store
	_ = g
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,q=2;"+b64A), 0, 0)
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,q=2;"+b64B), 0, 0)

	store.mu.Lock()
	im := store.images[1]
	if im == nil || im.b64 != b64B || im.officeID != idA {
		t.Fatalf("latest transmission wins under the STABLE id: %+v (want id %08x)", im, idA)
	}
	if len(store.images) != 1 {
		t.Fatalf("id reuse never grows the store: %d", len(store.images))
	}
	store.mu.Unlock()
	if dels := store.drainPendingIDs(); len(dels) != 0 {
		t.Fatalf("same-(id,placement) replace queues ZERO deletes (atomic terminal-side): %v", dels)
	}

	// a PLACEMENT change re-ids: the old placement's delete queues.
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,p=2,q=2;"+b64B), 0, 0)
	store.mu.Lock()
	im = store.images[1]
	if im == nil || im.officeID != ZenbuOfficeID(1, 2) {
		t.Fatalf("a placement change re-ids: %+v (want %08x)", im, ZenbuOfficeID(1, 2))
	}
	store.mu.Unlock()
	dels := store.drainPendingIDs()
	if len(dels) != 1 || dels[0] != idA {
		t.Fatalf("the old placement's delete queued: %v (want [%08x])", dels, idA)
	}
	if dels[0] == ZenbuOfficeID(1, 2) {
		t.Fatalf("the LIVE generation is never deleted: %v", dels)
	}
}

// TestZenbuOfficeIDStableAndNamespaced — the id's contract: stable per
// (child id, placement), sensitive to BOTH halves, and namespaced away
// from the chat lane's content-hash ids (a KittyImageID over payload
// bytes never equals the lane id for the same bytes).
func TestZenbuOfficeIDStableAndNamespaced(t *testing.T) {
	if ZenbuOfficeID(1, 1) != ZenbuOfficeID(1, 1) {
		t.Fatal("deterministic per (child id, placement)")
	}
	if ZenbuOfficeID(1, 1) == ZenbuOfficeID(2, 1) || ZenbuOfficeID(1, 1) == ZenbuOfficeID(1, 2) {
		t.Fatal("sensitive to BOTH the child id and the placement id")
	}
	if ZenbuOfficeID(1, 0) == KittyImageID(kittyTestPayload) {
		t.Fatal("namespaced away from the chat lane's content-hash ids")
	}
}

// TestZenbuImageStoreDeletes — the a=d selectors map to the office ids.
func TestZenbuImageStoreDeletes(t *testing.T) {
	mk := func() (*zenbuImageStore, uint32) {
		store := newZenbuImageStore()
		store.apply(mustParse(t, "a=T,t=d,f=100,i=1,p=7,q=2;"+kittyTestB64()), 0, 0)
		return store, ZenbuOfficeID(1, 7)
	}

	// d=i (the explicit id delete).
	store, id := mk()
	store.apply(mustParse(t, "a=d,d=i,i=1"), 0, 0)
	store.mu.Lock()
	if len(store.images) != 0 {
		t.Fatalf("d=i removes the image: %d left", len(store.images))
	}
	store.mu.Unlock()
	if dels := store.drainPendingIDs(); len(dels) != 1 || dels[0] != id {
		t.Fatalf("d=i queues the office delete: %v", dels)
	}

	// placement delete d=p with the matching placement id.
	store, id = mk()
	store.apply(mustParse(t, "a=d,d=p,i=1,p=7"), 0, 0)
	if dels := store.drainPendingIDs(); len(dels) != 1 || dels[0] != id {
		t.Fatalf("d=p with the matching placement deletes: %v", dels)
	}
	// …but a NON-matching placement id keeps the image.
	store, _ = mk()
	store.apply(mustParse(t, "a=d,d=p,i=1,p=9"), 0, 0)
	if dels := store.drainPendingIDs(); len(dels) != 0 {
		t.Fatalf("a non-matching placement delete is a no-op: %v", dels)
	}

	// d=A (delete all).
	store, id = mk()
	store.apply(mustParse(t, "a=d,d=A,q=2"), 0, 0)
	store.mu.Lock()
	if len(store.images) != 0 {
		t.Fatalf("d=A clears the store: %d left", len(store.images))
	}
	store.mu.Unlock()
	if dels := store.drainPendingIDs(); len(dels) != 1 || dels[0] != id {
		t.Fatalf("d=A queues every placed image's delete: %v", dels)
	}
}

// TestZenbuImageStoreBound — the store stays bounded over a long
// session: the oldest evicts, and an evicted PLACED image's delete
// queues (no zombie frame on the terminal).
func TestZenbuImageStoreBound(t *testing.T) {
	store := newZenbuImageStore()
	var wantEvictedDelete uint32
	for i := 0; i < zenbuMaxLiveImages+4; i++ {
		payload := []byte(fmt.Sprintf("frame %03d", i))
		b64 := base64.StdEncoding.EncodeToString(payload)
		if i == 0 {
			wantEvictedDelete = ZenbuOfficeID(100, 0) // the fake's i=100, no placement key
		}
		store.apply(mustParse(t, fmt.Sprintf("a=T,t=d,f=100,i=%d,q=2;%s", 100+i, b64)), 0, 0)
	}
	store.mu.Lock()
	if len(store.images) != zenbuMaxLiveImages {
		t.Fatalf("the store is bounded at %d, got %d", zenbuMaxLiveImages, len(store.images))
	}
	if store.images[100] != nil {
		t.Fatal("the oldest image evicted")
	}
	store.mu.Unlock()
	dels := store.drainPendingIDs()
	if len(dels) == 0 || dels[0] != wantEvictedDelete {
		t.Fatalf("the evicted placed image's delete queued: %v", dels)
	}
}

// TestZenbuImageStoreTransmitOnlyThenPlace — a=t stores WITHOUT a
// placement (nothing splices); a=p later places it (the splice appears)
// and a geometry override rebuilds the cached emit.
func TestZenbuImageStoreTransmitOnlyThenPlace(t *testing.T) {
	store := newZenbuImageStore()
	store.apply(mustParse(t, "a=t,t=d,f=100,i=1,q=2;"+kittyTestB64()), 0, 0)
	if ps := store.placements(); len(ps) != 0 {
		t.Fatalf("transmit-only never splices: %v", ps)
	}
	store.apply(mustParse(t, "a=p,i=1,c=30,r=10"), 3, 2)
	ps := store.placements()
	if len(ps) != 1 || ps[0].ox != 3 || ps[0].oy != 2 {
		t.Fatalf("a=p places at the cursor: %+v", ps)
	}
	if !strings.Contains(ps[0].frame, ",c=30,r=10,") && !strings.HasSuffix(strings.Split(ps[0].frame, ";")[0], ",c=30,r=10") {
		t.Fatalf("the placement geometry rides the emit: %q", ps[0].frame[:80])
	}
}

// TestKittyEmitAndDeleteFrames — the office-side APC shapes byte-pinned:
// the emit is a=T,t=d with q=2 + C=1 forced, the STABLE office id, the
// child's geometry keys verbatim (no body box pinned → pass-through);
// the delete is d=I (delete+free).
func TestKittyEmitAndDeleteFrames(t *testing.T) {
	store := newZenbuImageStore()
	store.apply(mustParse(t, "a=T,t=d,f=32,o=z,s=1600,v=960,i=1,p=1,q=2;"+kittyTestB64()), 0, 0)
	ps := store.placements()
	if len(ps) != 1 {
		t.Fatalf("one placement: %v", ps)
	}
	hash8 := KittyIDHash8(ZenbuOfficeID(1, 1)) // the fake's i=1,p=1
	want := "\x1b_Ga=T,t=d,q=2,C=1,i=" + hash8 + ",f=32,o=z,s=1600,v=960,p=1;" + kittyTestB64() + "\x1b\\"
	if ps[0].frame != want {
		t.Fatalf("the emit frame is byte-pinned:\n got %q\nwant %q", ps[0].frame, want)
	}
	// the frame is cached: a second placements() returns the same string
	// (no re-encode per repaint).
	if ps2 := store.placements(); ps2[0].frame != want {
		t.Fatal("the emit frame caches across repaints")
	}
	// the emit measures ZERO display cells and strips to nothing (the
	// frame row can never mis-measure or leak glyphs into text rows).
	if w := ansi.StringWidth(want); w != 0 {
		t.Fatalf("the emit frame is zero-cell, got %d", w)
	}
	if s := ansi.Strip(want); s != "" {
		t.Fatalf("the emit frame strips clean, got %q", s)
	}
	if got, want := kittyDeleteFrame(ZenbuOfficeID(1, 1)), "\x1b_Ga=d,d=I,i="+hash8+",q=2;\x1b\\"; got != want {
		t.Fatalf("the delete frame is byte-pinned: %q vs %q", got, want)
	}
}

// TestZenbuImageStoreBodyBox — FIX B: with the pane's body box pinned,
// EVERY re-emission carries c=bodyCols,r=bodyRows — a child-supplied
// c=/r= is REPLACED (forward-compat), an absent one is ADDED, and a
// box change (the resize wiring) lands in the very next apply.
func TestZenbuImageStoreBodyBox(t *testing.T) {
	store := newZenbuImageStore()
	store.setBodyBox(64, 14)
	// child-supplied c=/r= is replaced by the office box:
	store.apply(mustParse(t, "a=T,t=d,f=32,o=z,s=992,v=960,c=999,r=999,i=1,p=1,q=2;"+kittyTestB64()), 0, 0)
	ps := store.placements()
	if len(ps) != 1 {
		t.Fatalf("one placement: %v", ps)
	}
	hash8 := KittyIDHash8(ZenbuOfficeID(1, 1))
	want := "\x1b_Ga=T,t=d,q=2,C=1,i=" + hash8 + ",f=32,o=z,s=992,v=960,c=64,r=14,p=1;" + kittyTestB64() + "\x1b\\"
	if ps[0].frame != want {
		t.Fatalf("the office box REPLACES child c=/r=:\n got %q\nwant %q", ps[0].frame, want)
	}
	// an absent child c=/r= is added from the box:
	store.apply(mustParse(t, "a=T,t=d,f=32,o=z,s=992,v=960,i=1,p=1,q=2;"+kittyTestB64()), 0, 0)
	if ps := store.placements(); ps[0].frame != want {
		t.Fatalf("the office box is ADDED when the child sends none:\n got %q\nwant %q", ps[0].frame, want)
	}
	// the box changes (the resize wiring): the NEXT apply carries the new
	// dims immediately — transmit AND placement paths alike.
	store.setBodyBox(40, 8)
	store.apply(mustParse(t, "a=T,t=d,f=32,o=z,s=992,v=960,i=1,p=1,q=2;"+kittyTestB64()), 0, 0)
	if ps := store.placements(); !strings.Contains(ps[0].frame, ",c=40,r=8,") {
		t.Fatalf("the new box lands on the next transmit: %q", ps[0].frame[:120])
	}
	store.apply(mustParse(t, "a=p,i=1,p=1"), 0, 0)
	if ps := store.placements(); !strings.Contains(ps[0].frame, ",c=40,r=8,") {
		t.Fatalf("the new box lands on the next placement: %q", ps[0].frame[:120])
	}
}

// mustParse — the parser's test helper (fail on error).
func mustParse(t *testing.T, apcBody string) kittyCmd {
	t.Helper()
	cmd, err := parseKittyAPC([]byte(apcBody))
	if err != nil {
		t.Fatalf("parse %q: %v", apcBody, err)
	}
	return cmd
}

// TestKittyParseMatrix — the parser's corner table.
func TestKittyParseMatrix(t *testing.T) {
	c := mustParse(t, "a=T,f=32,o=z,s=1600,v=960,t=d,i=1,p=1,C=1,q=2,m=1")
	if c.action != 'T' || c.format != "32" || c.okey != "z" || c.pw != "1600" || c.ph != "960" ||
		c.medium != 'd' || c.id != 1 || !c.hasID || c.place != 1 || !c.hasPlace || c.ckey != "1" || c.more != 1 {
		t.Fatalf("the full frame command parses: %+v", c)
	}
	c = mustParse(t, "m=0")
	if c.action != 0 || c.more != 0 {
		t.Fatalf("a chunk continuation is actionless with m: %+v", c)
	}
	c = mustParse(t, "a=T,t=d,f=100,i=1")
	if c.more != 0 {
		t.Fatalf("an absent m defaults to 0 (the spec's final chunk): %+v", c)
	}
	c = mustParse(t, "a=d,d=I,i=7,q=2")
	if c.action != 'd' || c.delWhat != "I" || c.id != 7 {
		t.Fatalf("the delete parses: %+v", c)
	}
	for _, bad := range []string{"a", "a=", "a=TT", "i=abc", "m=7", "o=q", "q=", "d=ii"} {
		if _, err := parseKittyAPC([]byte(bad)); err == nil {
			t.Fatalf("%q must be malformed", bad)
		}
	}
}

// -------------------------------------------------------------------
// the controller + REAL-session integration (fixture binary, real PTY)
// -------------------------------------------------------------------

// kittyLaneFake — the scripted fake binary: homes the cursor, paints a
// toolbar row, emits ONE chunked kitty frame (m=1/m=1/m=0, the first
// chunk cut at a non-4-aligned boundary), paints a page text row, then
// parks. Mirrors the real child's stream shape (browser_lane_kitty.go's
// header) at fixture scale.
func kittyLaneFake(root string) error {
	b64 := kittyTestB64()
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:7] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[7:41] + "\\033\\\\'\n" +
		"printf '\\033_Gm=0;" + b64[41:] + "\\033\\\\'\n" +
		"printf '\\033[3;1H'\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// plantKittyLaneFake — the fixture-PATH binary + the hermetic kitty env
// (TestBrowserLaneReapReal's precedent: real PTY, real exec).
func plantKittyLaneFake(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	if err := kittyLaneFake(root); err != nil {
		t.Fatalf("plant the fake binary: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// waitLaneGrid polls until the needle paints the live grid (the reader
// loop is async; the marker comes AFTER the frame APC in the script, so
// a visible marker guarantees the transmission committed).
func waitLaneGrid(t *testing.T, g *term.Grid, want string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		for y := 0; y < g.Rows(); y++ {
			if strings.Contains(g.LineText(y), want) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the fake's %q never painted the embedded grid", want)
}

// TestBrowserLaneKittyFrameState — the FULL pane path over a real PTY
// (the wave-81 emission redesign): the RegionView frame is PURE TEXT
// (chrome + grid rows — the View string carries ZERO APC bytes; the
// renderer would eat them), the controller's FrameState surfaces the
// office-side re-emission for the frame-splice wrapper (the OFFICE
// content id + the pane-local origin + the cached verbatim APC), and
// Close flushes the delete through the direct-emit seam AND clears the
// shared registry.
func TestBrowserLaneKittyFrameState(t *testing.T) {
	plantKittyLaneFake(t)
	var emitted []string
	restore := SetZenbuEmitForShot(func(s string) { emitted = append(emitted, s) })
	t.Cleanup(restore)

	c := NewBrowserLaneController(64, 16)
	const u = "https://x.dev/kitty"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess, ok := c.Session().(*ZenbuSession)
	if !ok {
		t.Fatalf("the real seam embeds a *ZenbuSession, got %T", c.Session())
	}
	waitLaneGrid(t, sess.Grid(), "zenbu-fake open "+u)

	frame := c.RegionView(nil)
	plain := ansi.Strip(frame)
	// (a) the text chrome: toolbar + marker + badge + strip, base64-free.
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · " + u, "TB-TOOLBAR", "zenbu-fake open " + u} {
		if !strings.Contains(plain, want) {
			t.Fatalf("the premium frame carries %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, kittyTestB64()[:16]) {
		t.Fatalf("NO base64 glyphs in any text row:\n%s", plain)
	}
	// (b) the View string carries ZERO APC bytes — the renderer eats
	// zero-width sequences, so the image rides the frame wrapper only.
	if strings.Contains(frame, "\x1b_G") {
		t.Fatalf("the RegionView must stay APC-free (the wrapper emits):\n%q", frame)
	}
	// (c) the registry contribution: ONE image — the OFFICE stable id
	// (ZenbuOfficeID over child i=1 + placement 0), the pane-local origin
	// (0,1) (the child homed, painted the toolbar, then transmitted),
	// and the cached verbatim APC — carrying the pane's body box c=64,
	// r=14 (FIX B: the controller is 64x16, bodyH = 16-2).
	imgs, deletes := c.FrameState()
	if len(deletes) != 0 {
		t.Fatalf("no deletes pending on a healthy frame: %v", deletes)
	}
	if len(imgs) != 1 {
		t.Fatalf("one live image for the registry, got %d", len(imgs))
	}
	emitHead := "\x1b_Ga=T,t=d,q=2,C=1,i=" + kittyTestIDHash8() + ",f=100,c=64,r=14;"
	im := imgs[0]
	if im.OfficeID != kittyTestID() || im.OX != 0 || im.OY != 1 {
		t.Fatalf("registry image = id %08x @(%d,%d), want %08x @(0,1)", im.OfficeID, im.OX, im.OY, kittyTestID())
	}
	if im.Frame != emitHead+kittyTestB64()+"\x1b\\" {
		t.Fatalf("the FULL payload rides the emit verbatim:\n got %q\nwant %q", im.Frame, emitHead+kittyTestB64()+"\x1b\\")
	}
	// (d) the cached frame is stable across repaints (no re-encode) —
	// and the deletes queue drained with the first read.
	imgs2, deletes2 := c.FrameState()
	if len(imgs2) != 1 || imgs2[0].Frame != im.Frame {
		t.Fatal("the emit frame caches across repaints")
	}
	if len(deletes2) != 0 {
		t.Fatalf("the drained queue stays drained: %v", deletes2)
	}
	// (e) Close: the delete flushes DIRECTLY (the pane is gone) and the
	// shared registry clears (the renderer's final flush finds nothing).
	ZenbuRegistry().Publish(true, 0, 3, imgs, nil) // a live entry to clear
	c.Close()
	if len(emitted) == 0 || !strings.Contains(strings.Join(emitted, ""), kittyDeleteFrame(kittyTestID())) {
		t.Fatalf("Close flushes the office-side delete for %08x: %q", kittyTestID(), emitted)
	}
	if active, _, _, regImgs, regDels := ZenbuRegistry().snapshot(); active || len(regImgs) != 0 || len(regDels) != 0 {
		t.Fatalf("Close clears the registry: active=%v imgs=%d dels=%d", active, len(regImgs), len(regDels))
	}
}

// TestBrowserLaneKittyResizeDeletes — a pane resize: the live
// placement's geometry goes stale — the store un-places it (the
// registry's next publish carries NO image, so the wrapper's emitted-set
// diff deletes it) and the queued retire delete drains through the next
// FrameState read — while the PTY takes the SIGWINCH.
func TestBrowserLaneKittyResizeDeletes(t *testing.T) {
	plantKittyLaneFake(t)
	c := NewBrowserLaneController(64, 16)
	const u = "https://x.dev/kitty"
	if err := c.OpenURL(u); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess := c.Session().(*ZenbuSession)
	waitLaneGrid(t, sess.Grid(), "zenbu-fake open "+u)
	if imgs, _ := c.FrameState(); len(imgs) != 1 {
		t.Fatalf("setup: one live placement before the resize, got %d", len(imgs))
	}
	c.SetSize(40, 10)
	// the PTY + the screen model took the new box…
	if cols, rows := sess.Size(); cols != 40 || rows != 8 {
		t.Fatalf("the resize SIGWINCHes the child: %dx%d, want 40x8", cols, rows)
	}
	// …and the stale placement retired: nothing live for the registry,
	// the retire delete queued exactly once.
	imgs, deletes := c.FrameState()
	if len(imgs) != 0 {
		t.Fatalf("the resize retires every placement: %v", imgs)
	}
	if len(deletes) != 1 || deletes[0] != kittyTestID() {
		t.Fatalf("the retire delete drains with the publish: %v", deletes)
	}
	if _, deletes2 := c.FrameState(); len(deletes2) != 0 {
		t.Fatalf("the retire delete drains ONCE, not per read: %v", deletes2)
	}
	c.Close()
}

// TestZenbuSessionCloseDeleteIdempotent — Close's delete flush fires
// exactly once (the session's idempotent-Close contract extends to the
// terminal's image store).
func TestZenbuSessionCloseDeleteIdempotent(t *testing.T) {
	store := newZenbuImageStore()
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,q=2;"+kittyTestB64()), 0, 0)
	if got := store.dropAll(); !strings.Contains(got, kittyDeleteFrame(kittyTestID())) {
		t.Fatalf("dropAll queues the placed image's delete: %q", got)
	}
	if got := store.dropAll(); got != "" {
		t.Fatalf("a second dropAll is empty (idempotent): %q", got)
	}
	if ps := store.placements(); len(ps) != 0 {
		t.Fatalf("dropAll un-places everything: %v", ps)
	}
}

// -------------------------------------------------------------------
// FIX C — the wave-82 base64-leak regressions
// -------------------------------------------------------------------

// b64Runs — count base64-ish runs (len ≥ 40) in a text blob (the leak
// signature: ~30 dense rows of payload glyphs).
func b64Runs(s string) int {
	n, run := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '=' {
			run++
			if run == 40 {
				n++
			}
		} else {
			run = 0
		}
	}
	return n
}

// gridB64Runs — the base64-run count across every grid row.
func gridB64Runs(g *term.Grid) int {
	n := 0
	for y := 0; y < g.Rows(); y++ {
		n += b64Runs(g.LineText(y))
	}
	return n
}

// TestKittyStreamInterleavedSequenceDropsTail — the EXACT wave-82 leak
// shape from the real capture (terminal-browser v0.6.0): a racy OSC-7
// cwd report lands INSIDE a chunk's base64 payload, the payload RESUMES
// after the OSC's BEL, and the chunk's own ESC\ ends it. The aborted
// frame's tail must NEVER paint the grid or the scrollback (the old
// resync-at-ESC let ~30 dense base64 rows through), the open chain dies
// with it, the surrounding chrome survives, and the NEXT well-formed
// frame lands cleanly.
func TestKittyStreamInterleavedSequenceDropsTail(t *testing.T) {
	b64 := kittyTestB64()
	osc7 := "\x1b]7;file://host/var/folders/x\x07"
	stream := "\x1b[2J\x1b[H" + "TB-TOOLBAR\r\n" +
		kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]) + // the chain opens
		"\x1b_Gm=1;" + b64[20:40] + osc7 + b64[40:64] + "\x1b\\" + // CORRUPT: OSC-7 interleaved mid-payload
		kittyAPC("m=1", b64[64:80]) + // chain-less continuation (the frame is lost)
		"\x1b[4;1H" + "STILL-ALIVE\r\n" +
		kittyAPC("a=T,t=d,f=100,i=1,q=2", b64) + // the NEXT frame lands clean
		"\x1b[5;1H" + "done"
	ks, store, g, sb := newKittyRig(40, 8)
	ks.Write([]byte(stream))
	// THE regression pin: zero base64 downstream — grid AND scrollback.
	if n := gridB64Runs(g); n != 0 {
		t.Fatalf("base64 leaked into the grid (%d runs):\n%s", n, g.ScreenText())
	}
	if n := b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("base64 leaked into the scrollback (%d runs)", n)
	}
	// the corrupt frame dropped (chain aborted + chain-less chunk)…
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "stray ESC") {
		t.Fatalf("the abort + chain drop log-once: drops=%d note=%q", drops, note)
	}
	// …the surrounding chrome survived…
	if got := g.LineText(0); got != "TB-TOOLBAR" {
		t.Fatalf("the toolbar painted: %q", got)
	}
	found := false
	for y := 0; y < g.Rows(); y++ {
		if strings.Contains(g.LineText(y), "STILL-ALIVE") {
			found = true
		}
	}
	if !found {
		t.Fatalf("text after the corrupt frame still paints:\n%s", g.ScreenText())
	}
	// …and the NEXT well-formed frame under the same child id landed.
	store.mu.Lock()
	defer store.mu.Unlock()
	im := store.images[1]
	if im == nil || !im.placed || im.b64 != b64 || im.officeID != kittyTestID() {
		t.Fatalf("the next frame lands cleanly: %+v", im)
	}
}

// TestKittyStreamDiscardResyncsFreshAPC — a stray ESC that OPENS a fresh
// graphics APC (the child aborted + restarted) resyncs straight into the
// new command — no tail follows an aborted-then-restarted transmission.
func TestKittyStreamDiscardResyncsFreshAPC(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, g, _ := newKittyRig(40, 8)
	ks.Write([]byte("\x1b[H" +
		"\x1b_Ga=T,t=d,f=100,i=9;" + b64[:16] + // aborted mid-body, no terminator
		kittyAPC("a=T,t=d,f=100,i=1,q=2", b64) + // the fresh frame
		"after",
	))
	store.mu.Lock()
	defer store.mu.Unlock()
	if im := store.images[1]; im == nil || !im.placed {
		t.Fatalf("the fresh APC lands: %+v", store.images)
	}
	if store.images[9] != nil {
		t.Fatal("the aborted command never stores")
	}
	if got := g.LineText(0); got != "after" {
		t.Fatalf("the text after resyncs: %q", got)
	}
}

// TestKittyStreamChainCap — the pending-chain bound (8MiB production;
// shrunk here): an over-cap chain DROPS log-once and its remaining
// chunks drop chain-less — never a flush downstream, and the lane keeps
// working.
func TestKittyStreamChainCap(t *testing.T) {
	old := maxKittyChainB64
	maxKittyChainB64 = 32 // shrunk for the test
	t.Cleanup(func() { maxKittyChainB64 = old })
	b64 := kittyTestB64() // 76 chars > the 32-char cap once joined
	ks, store, g, sb := newKittyRig(40, 8)
	ks.Write([]byte("\x1b[H" +
		kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]) + // chain opens (20)
		kittyAPC("m=1", b64[20:60]) + // 20+40 = 60 > 32 → the chain drops
		kittyAPC("m=0", b64[60:]) + // chain-less → dropped
		"alive",
	))
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "cap") {
		t.Fatalf("the cap drop + the chain-less remainder log-once: drops=%d note=%q", drops, note)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.images) != 0 {
		t.Fatalf("the over-cap chain never stores: %d", len(store.images))
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the over-cap chain never leaks downstream (%d runs)", n)
	}
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("the lane survives the cap drop: %q", got)
	}
}

// TestKittyStreamResetDropsChain — the splitter-level half of the
// session-churn contract: 3 chunks of an m=1 chain arrive, the session
// then dies (reset mid-chain) — ZERO bytes of the chain reach the grid
// or the scrollback, and a fresh splitter/session parses cleanly.
func TestKittyStreamResetDropsChain(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, g, sb := newKittyRig(40, 8)
	ks.Write([]byte("\x1b[2J\x1b[H" + "TB-TOOLBAR\r\n" +
		kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]) +
		kittyAPC("m=1", b64[20:48]) +
		"\x1b_Gm=1;" + b64[48:], // the death lands mid-chunk: no terminator
	))
	ks.reset()
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the reset dropped the chain with ZERO downstream bytes (%d runs):\n%s", n, g.ScreenText())
	}
	if got := g.LineText(0); got != "TB-TOOLBAR" {
		t.Fatalf("the chrome survives the reset: %q", got)
	}
	drops, note := store.dropStats()
	if drops != 1 || !strings.Contains(note, "reset") {
		t.Fatalf("the reset drop logs once: drops=%d note=%q", drops, note)
	}
	store.mu.Lock()
	if len(store.images) != 0 {
		t.Fatalf("no partial frame stored: %d", len(store.images))
	}
	store.mu.Unlock()
	// the next session (a fresh splitter+store) parses cleanly.
	ks2, store2, g2, _ := newKittyRig(40, 8)
	ks2.Write([]byte("\x1b[H" + kittyAPC("a=T,t=d,f=100,i=1,q=2", b64)))
	store2.mu.Lock()
	defer store2.mu.Unlock()
	if im := store2.images[1]; im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the next session's frames parse cleanly: %+v", store2.images[1])
	}
	_ = g2
}

// kittyLaneMidChainDeathFake — the session-churn fake: homes, paints the
// toolbar + marker, streams 3 chunks of an m=1 chain — chunk 2 with the
// REAL child's OSC-7 interleave mid-payload — and DIES with chunk 3
// UNTERMINATED (the capture's exact EOF tail).
func kittyLaneMidChainDeathFake(root string) error {
	b64 := kittyTestB64()
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		"printf 'mid-chain death next\\r\\n'\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:20] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[20:36] + "\\033]7;file://host/tmp/x\\a" + b64[36:52] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[52:] + "'\n" + // UNTERMINATED — the death lands mid-chunk
		"exit 1\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// TestBrowserLaneSessionMidChainDeath — FIX C's session-level regression
// (requirement 3): a REAL fake child streams 3 chunks of an m=1 chain
// (one carrying the OSC-7 interleave) then DIES mid-chunk — the grid +
// scrollback contain ZERO base64, the chrome painted before the chain
// survives, and Close (the churn path) resets the splitter cleanly.
func TestBrowserLaneSessionMidChainDeath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the PTY seam is darwin/linux (creack/pty)")
	}
	pinKittyEnv(t)
	root := t.TempDir()
	if err := kittyLaneMidChainDeathFake(root); err != nil {
		t.Fatalf("plant the dying fake: %v", err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	restore := SetZenbuEmitForShot(func(string) {})
	t.Cleanup(restore)

	c := NewBrowserLaneController(64, 16)
	if err := c.OpenURL("https://x.dev/dies"); err != nil {
		t.Fatalf("open: %v", err)
	}
	sess, ok := c.Session().(*ZenbuSession)
	if !ok {
		t.Fatalf("the premium lane embeds a *ZenbuSession, got %T", c.Session())
	}
	waitLaneGrid(t, sess.Grid(), "mid-chain death next")
	for deadline := time.Now().Add(3 * time.Second); !sess.Exited() && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if !sess.Exited() {
		t.Fatal("the dying fake never reaped")
	}
	// THE pin: zero base64 in the grid AND the retained scrollback.
	if n := gridB64Runs(sess.Grid()); n != 0 {
		t.Fatalf("base64 leaked into the grid (%d runs):\n%s", n, sess.Grid().ScreenText())
	}
	if n := b64Runs(string(sess.Scrollback().Raw())); n != 0 {
		t.Fatalf("base64 leaked into the scrollback (%d runs)", n)
	}
	if got := sess.Grid().LineText(0); got != "TB-TOOLBAR" {
		t.Fatalf("the chrome painted before the chain survives: %q", got)
	}
	c.Close() // the churn path: reset + deletes + kill, idempotent, no panic
}
