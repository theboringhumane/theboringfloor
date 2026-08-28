// zenbu_frame.go — the zenbu premium lane's EMISSION architecture (the
// wave-81 redesign; the parser/store/splitter half stays in
// browser_lane_kitty.go).
//
// WHY NOT THE VIEW STRING: bubbletea v2.0.9's production renderer
// (cursed_renderer over ultraviolet) DECODES the View string into cells —
// ultraviolet's styled.go accumulates a non-SGR zero-width sequence (the
// APC) into cell.Content then OVERWRITES it on the next printable glyph
// (styled.go:150/236/252), and terminal_renderer.go:516-522 DROPS
// zero-width cells outright ("should not be written to the screen"). An
// APC riding the View string never reaches the TTY (production proof:
// 0 ESC_G in 18,668 captured bytes on a real ghostty PTY) — it only
// bloated every frame through the differ. So the pane's RegionView now
// paints PURE TEXT (grid rows only) and the images ride a different seam:
//
// THE SEAM (module-cache verified): tea.WithOutput(io.Writer)
// (options.go:28-34) — the cursed renderer writes every flush through
// p.output (tea.go:1069-1074; one io.Copy of a bytes.Buffer per frame in
// cursed_renderer.go:616 — bytes.Buffer's WriteTo lands it as ONE Write
// call per flush). ZenbuFrameWriter WRAPS that output: every renderer
// byte passes through unchanged, and after each frame flush it appends,
// for each live premium-lane image, cursor-save + CUP (1-based) to the
// image's ABSOLUTE screen cell + the cached verbatim APC (q=2, C=1 —
// emitLocked's contract) + cursor-restore. Kitty z-order (images paint
// below text glyphs, above the background) keeps the child's text chrome
// readable over the image.
//
// THE REGISTRY: the Model publishes once per Frame() (app/browser.go's
// publishZenbuFrame — the bottom-of-Frame seam, so the entry always
// matches the frame the renderer just got): the pane's ABSOLUTE cell
// origin of the body grid (desktop AND mobile: x=0, y=3 — topbar 1 +
// switcher strip 1 + the RegionView badge row 1; model.go's Frame owns
// the branch structure the helper mirrors) + the live image list (office
// content ids + pane-local cell offsets from the grid-cursor capture +
// the cached verbatim APC bytes) + the store's drained pending deletes
// (bounded — one render interval of replacements). When the lane paints
// nothing (floor showing / text lane / zen / thread-focus / plan slot /
// closed) the registry is EMPTY and the wrapper emits nothing — and the
// active→empty transition makes the wrapper flush one a=d per
// previously-emitted id (the diff against its own emitted set).
//
// DELETES have three riders, all converging on the same a=d frames
// (kitty no-ops an unknown id, q=2 hushes the answer): (1) the wrapper's
// emitted-set diff after every flush; (2) the store's pending queue
// drained into the registry publish; (3) the DIRECT path — suspend /
// fallback-flip / Close / quit flush through the zenbuEmit seam, which
// production wires to the wrapper's DirectEmit (main.go) so the bytes
// serialize with frame flushes under the wrapper's mutex; ZenbuSession.
// Close ALSO clears the registry, so the renderer's final flush (the
// alt-screen exit at tea.Quit) finds nothing to re-emit — no ghosts over
// the restored main screen, and the wrapper's Finish (main.go teardown,
// right after p.Run returns) sweeps one last a=d per still-emitted id
// for the FATAL path (an external kill skips the Update quit paths).
//
// IDEMPOTENCE: the wrapper re-emits the current live images after EVERY
// flush (kitty dedupes by image id; q=2 suppresses the responses), so
// any screen-clearing the renderer performs (resize ED and friends) is
// repaired on the very next flush. Only the cached assembled APC strings
// ride — never a per-frame re-encode (emitLocked's cache).
//
// CURSOR DISCIPLINE: DECSC (ESC 7) + DECRC (ESC 8) around every splice;
// the emitted APC itself is C=1 (cursor-stay), so the renderer's cursor
// is byte-identically where it left it.
package panels

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// ZenbuFrameImage — ONE live premium-lane image for the frame splice:
// the office content id (the a=d target), the pane-local cell origin
// (the grid cursor at commit), and the cached verbatim a=T APC
// (emitLocked — q=2 + C=1 enforced, the child's payload+geometry keys
// verbatim, ZERO re-encode per flush).
type ZenbuFrameImage struct {
	OfficeID uint32
	OX, OY   int    // pane-local cell origin (0-based, grid coordinates)
	Frame    string // the cached verbatim APC ("" = skip)
}

// ZenbuFrameRegistry — the Model→wrapper channel: Frame() publishes the
// premium lane's absolute origin + live images + drained deletes; the
// output wrapper snapshots it after each renderer flush. Mutex-guarded
// (Frame runs on the Update/render path, the renderer flushes on its own
// goroutine, the lane's Close clears from the Update goroutine). The
// entry is replaced WHOLESALE per publish — a snapshot is always one
// coherent frame's truth.
type ZenbuFrameRegistry struct {
	mu      sync.Mutex
	active  bool
	originX int
	originY int
	images  []ZenbuFrameImage
	deletes []uint32
}

// zenbuRegistry — the process singleton (one office, one browser pane):
// the app publishes, main.go's wrapper reads, the lane's Close clears.
var zenbuRegistry = &ZenbuFrameRegistry{}

// ZenbuRegistry — the shared registry (main.go installs the wrapper on
// it; the app's Frame publishes; tests drive it directly).
func ZenbuRegistry() *ZenbuFrameRegistry { return zenbuRegistry }

// Publish replaces the entry wholesale (images/deletes copied — the
// caller's slices are never retained). active=false is Clear.
func (r *ZenbuFrameRegistry) Publish(active bool, originX, originY int, images []ZenbuFrameImage, deletes []uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active, r.originX, r.originY = active, originX, originY
	r.images = append([]ZenbuFrameImage(nil), images...)
	r.deletes = append([]uint32(nil), deletes...)
}

// Clear — the lane paints nothing (floor / text lane / zen / focus /
// plan slot / closed): the wrapper's next flush emits no images and
// diff-deletes whatever it emitted before.
func (r *ZenbuFrameRegistry) Clear() { r.Publish(false, 0, 0, nil, nil) }

// snapshot — the wrapper's read: one coherent copy of the entry.
func (r *ZenbuFrameRegistry) snapshot() (active bool, originX, originY int, images []ZenbuFrameImage, deletes []uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active, r.originX, r.originY,
		append([]ZenbuFrameImage(nil), r.images...),
		append([]uint32(nil), r.deletes...)
}

// SnapshotForTest — the harness's read of the published entry (the
// wrapper's snapshot, exported for the app-level splice proofs).
func (r *ZenbuFrameRegistry) SnapshotForTest() (active bool, originX, originY int, images []ZenbuFrameImage, deletes []uint32) {
	return r.snapshot()
}

// -------------------------------------------------------------------
// the frame-splice output wrapper
// -------------------------------------------------------------------

// ZenbuFrameWriter — an io.Writer installed over the program's output
// (tea.WithOutput): EVERY byte of the renderer passes through unchanged;
// after each Write (one renderer flush — cursed_renderer's one-io.Copy-
// per-frame, bytes.Buffer's single-Write WriteTo) the current registry's
// images are re-emitted at their absolute cells (cursor-saved/restored),
// preceded by that flush's deletes (the store's drained queue + the
// emitted-set diff). Concurrency: every write to the underlying stream —
// renderer flush, DirectEmit lane-lifecycle delete, Finish sweep — is
// serialized under the wrapper's own mutex, so a splice can never
// interleave mid-frame.
//
// The wrapper ALSO satisfies x/term's File interface (Fd + Read + Close)
// by delegation: bubbletea's ttyOutput detection (tty_unix.go:30) and
// colorprofile.Detect (env.go:33) type-assert the output to term.File —
// without Fd() the office's color profile and size probing would degrade.
type ZenbuFrameWriter struct {
	dst io.Writer
	reg *ZenbuFrameRegistry

	mu      sync.Mutex
	emitted map[uint32]bool // office ids the terminal currently holds (the diff base)
	closed  bool            // Finish ran: passthrough-only from then on
}

// NewZenbuFrameWriter wraps dst (os.Stdout in production) with the
// frame-splice seam reading reg.
func NewZenbuFrameWriter(dst io.Writer, reg *ZenbuFrameRegistry) *ZenbuFrameWriter {
	return &ZenbuFrameWriter{dst: dst, reg: reg, emitted: map[uint32]bool{}}
}

// Write implements io.Writer: the renderer's bytes pass through FIRST
// (the returned n/err are exactly the underlying write's — the splice is
// best-effort and never fails the frame), then the registry's current
// images splice after the flush.
func (w *ZenbuFrameWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.dst.Write(p)
	if err == nil && !w.closed {
		w.spliceLocked()
	}
	return n, err
}

// spliceLocked — the post-flush emission: (1) the store's drained
// deletes, (2) the emitted-set diff (ids the terminal holds that the
// registry no longer lists — the active→empty transition included),
// (3) every live image re-emitted at its absolute cell (idempotent:
// kitty dedupes by id, q=2 hushes the answer). Deletes ride ahead of
// placements so a same-id delete-then-add converges inside one flush.
func (w *ZenbuFrameWriter) spliceLocked() {
	active, ox, oy, images, deletes := w.reg.snapshot()
	live := map[uint32]bool{}
	if active {
		for _, im := range images {
			live[im.OfficeID] = true
		}
	}
	var b strings.Builder
	deleted := map[uint32]bool{} // one a=d per id per flush (queue ∪ diff)
	for _, id := range deletes {
		if deleted[id] {
			continue
		}
		deleted[id] = true
		b.WriteString(kittyDeleteFrame(id))
	}
	var stale []uint32
	for id := range w.emitted {
		if !live[id] {
			stale = append(stale, id)
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i] < stale[j] }) // byte-deterministic
	for _, id := range stale {
		if !deleted[id] {
			b.WriteString(kittyDeleteFrame(id))
		}
		delete(w.emitted, id)
	}
	if active {
		for _, im := range images {
			if im.Frame == "" {
				continue
			}
			row, col := oy+im.OY+1, ox+im.OX+1 // CUP is 1-based
			if row < 1 {
				row = 1
			}
			if col < 1 {
				col = 1
			}
			fmt.Fprintf(&b, "\x1b7\x1b[%d;%dH", row, col)
			b.WriteString(im.Frame)
			b.WriteString("\x1b8")
			w.emitted[im.OfficeID] = true
		}
	}
	if b.Len() > 0 {
		_, _ = io.WriteString(w.dst, b.String())
	}
}

// DirectEmit — the production zenbuEmit target (main.go wires it through
// SetZenbuEmit): the lane-lifecycle deletes (suspend / fallback-flip /
// Close / quit) write straight to the underlying stream, serialized with
// frame flushes under the wrapper's mutex — a direct delete can never
// interleave mid-frame or mid-splice.
func (w *ZenbuFrameWriter) DirectEmit(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = io.WriteString(w.dst, s)
}

// Finish — the main.go teardown sweep (right after p.Run returns): one
// a=d per still-emitted id, then the wrapper seals (passthrough-only).
// The CLEAN quit path already deleted through the lane Close (these are
// q=2-hushed no-op dupes); the FATAL path (an external kill skips the
// Update quit paths) is the real rider — images must not ghost over the
// restored main screen. Idempotent.
func (w *ZenbuFrameWriter) Finish() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	if len(w.emitted) == 0 {
		return
	}
	ids := make([]uint32, 0, len(w.emitted))
	for id := range w.emitted {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var b strings.Builder
	for _, id := range ids {
		b.WriteString(kittyDeleteFrame(id))
	}
	_, _ = io.WriteString(w.dst, b.String())
	w.emitted = map[uint32]bool{}
}

// Fd — the x/term File contract, delegated to the underlying writer when
// IT has one (os.Stdout in production): bubbletea's ttyOutput detection
// and colorprofile.Detect type-assert the output to term.File, and the
// office's color profile + terminal-size probing ride that assertion. An
// invalid fd when the wrapped stream has none (term.IsTerminal reads
// false — exactly like any non-file writer).
func (w *ZenbuFrameWriter) Fd() uintptr {
	if f, ok := w.dst.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Read — the x/term File contract, delegated when the underlying writer
// is readable (os.Stdout is); EOF otherwise (bubbletea never reads the
// output — the interface's shape only).
func (w *ZenbuFrameWriter) Read(p []byte) (int, error) {
	if r, ok := w.dst.(io.Reader); ok {
		return r.Read(p)
	}
	return 0, io.EOF
}

// Close — the x/term File contract, DELIBERATELY a no-op: bubbletea
// never closes the output (verified v2.0.9), and the underlying fd 1
// must survive the program (main.go prints the resume line after Run).
func (w *ZenbuFrameWriter) Close() error { return nil }

// -------------------------------------------------------------------
// the lane's registry contribution (the Frame() publish's read side)
// -------------------------------------------------------------------

// FrameState — the controller's registry contribution: the live images
// (office id + pane-local origin + cached verbatim APC, the store's
// paint order) + the pending delete IDS drained from the store (bounded
// by one render interval of replacements — the queue's only other drain
// is dropAll at Close). (nil, nil) while the text lane paints or the
// session's fake lacks the kitty surface.
func (c *BrowserLaneController) FrameState() ([]ZenbuFrameImage, []uint32) {
	if c.sess == nil {
		return nil, nil
	}
	im, ok := c.sess.(zenbuImageSurface)
	if !ok {
		return nil, nil
	}
	body := c.bodyH()
	var imgs []ZenbuFrameImage
	for _, p := range im.imagePlacements() {
		if p.oy < 0 || p.oy >= body {
			continue // off-pane placements never emit
		}
		imgs = append(imgs, ZenbuFrameImage{OfficeID: p.officeID, OX: p.ox, OY: p.oy, Frame: p.frame})
	}
	return imgs, im.drainImageDeleteIDs()
}

// LaneFrameState — the pane's registry contribution: (nil, nil) while
// the lane resolves/paints text or no controller exists (the app's
// publish clears the registry then).
func (b *Browser) LaneFrameState() ([]ZenbuFrameImage, []uint32) {
	if b.lane == nil || !b.lane.PremiumActive() {
		return nil, nil
	}
	return b.lane.FrameState()
}
