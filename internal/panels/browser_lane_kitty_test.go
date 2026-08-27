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

	"github.com/theboringhumane/theboringoffice/internal/term"
)

// kittyTestPayload — the deterministic fake frame (content irrelevant —
// the store never image-decodes; the bytes only need to base64/decode
// round-trip and hash deterministically).
var kittyTestPayload = []byte("\x89PNG\r\n\x1a\nFAKEKITTYFRAME0123456789abcdefghijklmnopqrstuvwxyz")

func kittyTestB64() string { return base64.StdEncoding.EncodeToString(kittyTestPayload) }

func kittyTestID() uint32 { return KittyImageID(kittyTestPayload) }

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
		t.Fatalf("office id = %08x, want KittyImageID %08x", im.officeID, kittyTestID())
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
		{"stray ESC resync", "\x1b_Ga=T,t=d,f=100,i=1;" + b64[:8] + "\x1b[2J" + "after8", 1},
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

// TestZenbuImageStoreLatestWins — id reuse (the child's per-frame
// pattern): the new transmission replaces, and the STALE frame's office
// delete queues (the terminal never composites two generations).
func TestZenbuImageStoreLatestWins(t *testing.T) {
	store := newZenbuImageStore()
	payloadA := []byte("frame generation A")
	payloadB := []byte("frame generation B")
	b64A := base64.StdEncoding.EncodeToString(payloadA)
	b64B := base64.StdEncoding.EncodeToString(payloadB)
	idA, idB := KittyImageID(payloadA), KittyImageID(payloadB)

	ks, _, g, _ := newKittyRig(40, 8)
	ks.store = store
	_ = g
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,q=2;"+b64A), 0, 0)
	store.apply(mustParse(t, "a=T,t=d,f=100,i=1,q=2;"+b64B), 0, 0)

	store.mu.Lock()
	im := store.images[1]
	if im == nil || im.b64 != b64B || im.officeID != idB {
		t.Fatalf("latest transmission wins: %+v", im)
	}
	if len(store.images) != 1 {
		t.Fatalf("id reuse never grows the store: %d", len(store.images))
	}
	store.mu.Unlock()
	dels := store.drainPendingIDs()
	if len(dels) != 1 || dels[0] != idA {
		t.Fatalf("the stale generation's delete queued: %v", dels)
	}
	if dels[0] == idB {
		t.Fatalf("the LIVE generation is never deleted: %v", dels)
	}
}

// TestZenbuImageStoreDeletes — the a=d selectors map to the office ids.
func TestZenbuImageStoreDeletes(t *testing.T) {
	mk := func() (*zenbuImageStore, uint32) {
		store := newZenbuImageStore()
		store.apply(mustParse(t, "a=T,t=d,f=100,i=1,p=7,q=2;"+kittyTestB64()), 0, 0)
		return store, kittyTestID()
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
			wantEvictedDelete = KittyImageID(payload)
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
// the emit is a=T,t=d with q=2 + C=1 forced, the office content id, the
// child's geometry keys verbatim; the delete is d=I (delete+free).
func TestKittyEmitAndDeleteFrames(t *testing.T) {
	store := newZenbuImageStore()
	store.apply(mustParse(t, "a=T,t=d,f=32,o=z,s=1600,v=960,i=1,p=1,q=2;"+kittyTestB64()), 0, 0)
	ps := store.placements()
	if len(ps) != 1 {
		t.Fatalf("one placement: %v", ps)
	}
	want := "\x1b_Ga=T,t=d,q=2,C=1,i=" + kittyTestIDHash8() + ",f=32,o=z,s=1600,v=960,p=1;" + kittyTestB64() + "\x1b\\"
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
	if got, want := kittyDeleteFrame(kittyTestID()), "\x1b_Ga=d,d=I,i="+kittyTestIDHash8()+",q=2;\x1b\\"; got != want {
		t.Fatalf("the delete frame is byte-pinned: %q vs %q", got, want)
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
	// (c) the registry contribution: ONE image — the OFFICE content id,
	// the pane-local origin (0,1) (the child homed, painted the toolbar,
	// then transmitted), and the cached verbatim APC.
	imgs, deletes := c.FrameState()
	if len(deletes) != 0 {
		t.Fatalf("no deletes pending on a healthy frame: %v", deletes)
	}
	if len(imgs) != 1 {
		t.Fatalf("one live image for the registry, got %d", len(imgs))
	}
	emitHead := "\x1b_Ga=T,t=d,q=2,C=1,i=" + kittyTestIDHash8() + ",f=100;"
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
