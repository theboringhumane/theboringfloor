// chat_window.go — TRANSCRIPT VIRTUALIZATION: the per-block render cache
// plus the windowed viewport projection that together bound the chat
// panel's steady-state render work by the VIEWPORT, not the transcript.
//
// THE PROBLEM (pre-window): every SetState that cleared the revision gate
// re-rendered the WHOLE timeline (glamour markdown per boss message, plus
// the think/tool/diff/question/fold/thread builders for the rest), joined
// it into one string, re-split+re-padded it into lines, and re-posted all
// N lines to the bubbles viewport — whose SoftWrap machinery then walked
// EVERY line on EVERY SetContent/SetYOffset/View to recompute wrap heights
// the pre-folded renderers had already made trivial (every transcript row
// is wrapped at contentW before the viewport ever sees it).
//
// THE DESIGN — two orthogonal mechanisms:
//
//  1. PER-BLOCK RENDER CACHE (chatBlock): buildBlocks renders each
//     timeline ITEM (one message, one worker group) into its own buffer
//     and caches the finished PADDED lines + LOCAL click hit-maps keyed
//     by an invalidation fingerprint: the width/theme generation + every
//     toggle that re-shapes that kind's pixels (+ the office tick for
//     LIVE, tick-animated blocks), with the CONTENT compared structurally
//     (old.src == m — the reducer's cloned slice shares string backings,
//     so an unchanged message compares as headers, not bytes). An
//     unchanged block is NEVER re-rendered — appending / streaming into
//     the tail re-renders exactly the touched block(s); a resize (width
//     generation), a message update, a fold/thread toggle, or a /theme
//     switch (theme generation) each invalidate precisely the blocks they
//     move. Blocks carry their line HEIGHT and max cell WIDTH (computed
//     once, at render) — the metadata the window's blank-height model
//     needs (cached formatted lines/height metadata, per block).
//
//  2. WINDOWED PROJECTION (vpWindow): the viewport model only ever holds
//     a MATERIALIZED WINDOW of the transcript — the rows around the scroll
//     offset plus an overscan margin — with every row OUTSIDE the window
//     replaced by a blank of identical height. Because the renderers
//     pre-fold every row to ≤ the viewport width (and rogue rows — the
//     empty-state placeholder at a hairline width — are hard-cut to
//     exactly SoftWrap's chunks by setConversationLines's rescue), every
//     posted row has wrap height exactly 1, so the blank-height model is
//     EXACT: total posted rows == len(selLines), the scroll offset space
//     is unchanged, ClickRow hit-maps stay in absolute transcript line
//     space, and the visible slice is byte-identical to the full-content
//     viewport's.
//
//     window = [scrollOffset-overscan, scrollOffset+height+overscan)
//     page-in = lazily, only when the visible range leaves it
//
//     The projection is a PERSISTENT backing array aliased with the
//     viewport's content slice (SetContentLines stores the caller's slice
//     untouched when no line contains "\n" — ours never do): a scroll-only
//     window move blanks the old range and copies the new one IN PLACE —
//     O(window), zero SetContentLines re-scans per wheel notch — while
//     renders/rebuilds post through SetContentLines (the panel also needs
//     its scroll clamp there). View() runs a cheap covered-check
//     (syncWindow) before vp.View() so seams the panel does not own —
//     PreserveAnchor's SetYOffset (threads_opencode.go) or an app-level
//     Goto* — still land on materialized rows.
//
// THE INVARIANT that makes blanks free: vp.SoftWrap = false. Rendered
// transcript rows never exceed the viewport width (foldStyledLines /
// foldStyledRows / clipPlain budgets, plus setConversationLines's
// expandLines rescue cutting any rogue row to SoftWrap-exact chunks), so
// SoftWrap was already a no-op for this content — turning it off only
// removes its O(total lines) calculateLine / maxYOffset walks on every
// scroll and paint; it changes no pixel.
//
// selLines STAY FULL: selLines (the selection overlay/extraction row space
// AND TranscriptRows()'s answer — threads_opencode.go's pagination seams)
// keeps caching EVERY padded transcript row; the window only blanks the
// COPY posted to the viewport. Selection extraction, anchor preservation,
// and the absolute-row hit-maps are untouched.
package panels

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// -------------------------------------------------------------------
// per-block render cache
// -------------------------------------------------------------------

// blockFlags — the toggle bits a block's rendered pixels depend on, packed
// into the invalidation key. Meaning is kind-scoped:
//
//	bfExpandA    — think: thinkExpanded · diff: diffExpanded · group: resolved EXPANDED
//	bfExpandB    — user: fold-open · think: stream-open · group: threadLive
//	bfExpandC    — group: resolved FULL
//	bfStopped    — group: /stop latch
//	bfThreadHint — group: the live "ctrl+g · view subagents" tail row
type blockFlags uint16

const (
	bfExpandA    blockFlags = 1 << 0
	bfExpandB    blockFlags = 1 << 1
	bfExpandC    blockFlags = 1 << 2
	bfStopped    blockFlags = 1 << 3
	bfThreadHint blockFlags = 1 << 4
)

// blockHits — ONE rendered block's click hit-maps in BLOCK-LOCAL row space
// (row 0 = the block's first rendered line). Assembly offsets them into
// absolute transcript rows as the blocks stack (mergeBlockHits); nil maps
// are the common case ("no rows of that kind"). ClickRow's absolute lookup
// is byte-identical to the shared-map era.
type blockHits struct {
	thread   map[int]string
	toolDiff map[int]string
	userFold map[int]string
	toolOut  map[int]string // tool-output one-liner rows → tool entry ID (chat_toolrow.go)
}

// chatBlock — one timeline ITEM's cached render: the exact text the old
// monolithic builder wrote for it at this generation, the PADDED row slice
// (the cache payload — padding happens ONCE per render, here, instead of a
// full-transcript pad pass per render), its LOCAL hit-maps, and the
// dimensions the window + the rogue-row guard need WITHOUT re-reading the
// text (rows + the widest padded line).
type chatBlock struct {
	id     string          // timeline identity: message ID / "g:"+segment first-line ID
	key    uint64          // invalidation fingerprint (gen + toggle flags + tick-if-live)
	text   string          // transcript fragment, WITHOUT the inter-item "\n\n" glue
	plines []string        // the fragment's PADDED lines — selLines' row material
	hits   blockHits       // LOCAL row space
	media  []chatMediaSlot // kitty previews' paint slots, LOCAL row space (MediaFrameState's merge input)
	rows   int             // == len(plines) — the HEIGHT metadata
	wide   int             // widest PADDED line, CELLS (ANSI-aware) — the rogue-row guard's input
	src    state.ChatMsg   // MSG blocks: the message this block rendered
	lines  []state.ChatMsg // GROUP blocks: the segment's lines
	// unstable blocks (LIVE groups, OPEN think streams) animate with the
	// office tick — borrowed from the cache NEVER, rewritten every render
	// (the pre-cache behavior for exactly those rows).
	unstable bool
}

// finish derives the cache payload + dimensions from the rendered text:
// pad each line ONCE, then ONE ANSI-aware width pass.
func (b *chatBlock) finish() {
	b.plines = padFragmentLines(b.text)
	b.rows = len(b.plines)
	b.wide = 0
	for _, ln := range b.plines {
		if w := ansi.StringWidth(ln); w > b.wide {
			b.wide = w
		}
	}
}

// expandLines hard-cuts ONE rendered line into SoftWrap's exact chunks —
// contiguous ansi.Cut pieces of w cells (viewport.softWrap's loop); a
// fitting line yields itself as the single chunk. Used ONLY by
// setConversationLines's rogue-row rescue (the ~never path — the renderers
// pre-fold everything else tighter).
func expandLines(line string, w int) []string {
	if w <= 0 {
		return []string{line}
	}
	lw := ansi.StringWidth(line)
	if lw <= w {
		return []string{line}
	}
	out := make([]string, 0, (lw+w-1)/w)
	for idx := 0; idx < lw; idx += w {
		out = append(out, ansi.Cut(line, idx, idx+w))
	}
	return out
}

// -------------------------------------------------------------------
// windowed viewport projection
// -------------------------------------------------------------------

// winOverscanMin — the floor of the materialization margin: with a tiny
// viewport the margin would otherwise collapse to the wheel delta and
// every notch would re-materialize. At/above this, a wheel step (3) stays
// inside the window and page-in amortizes at PageUp/PageDown granularity.
const winOverscanMin = 8

// vpWindow — the projection's persistent state. proj is the EXACT slice
// handed to vp.SetContentLines: the viewport aliases its backing array, so
// scroll-only window moves mutate it in place and the next vp.View() picks
// the fresh materialization up with NO viewport call at all.
type vpWindow struct {
	proj   []string // len == len(selLines) when built; "" rows = the blank model
	lo, hi int      // materialized absolute rows [lo, hi)
	built  bool     // proj has been handed to the viewport at least once
}

// overscanSize — the margin: one viewport height above AND below (a
// PageUp/PageDown lands inside the fresh window), floored so mouse-wheel
// granularity doesn't thrash at hairline heights.
func (c *Chat) overscanSize() int {
	o := c.vp.Height()
	if o < winOverscanMin {
		o = winOverscanMin
	}
	return o
}

// winTarget is the raw materialization-want for an offset:
// [off-overscan, off+height+overscan) (unclamped).
func (c *Chat) winTarget(off int) (lo, hi int) {
	o := c.overscanSize()
	return off - o, off + c.vp.Height() + o
}

// winClamped clamps a target window INTO [0, total], materializing
// everything when the transcript is smaller than one full window.
func winClamped(lo, hi, total int) (int, int) {
	if lo < 0 {
		lo = 0
	}
	if hi > total {
		hi = total
	}
	if lo >= hi {
		lo, hi = 0, total
	}
	return lo, hi
}

// rebuildWindow posts the whole transcript through a FRESH window after a
// render: the posted slice is len(selLines) long (the blank-height model),
// with real rows only in [lo,hi). The anchor mirrors the caller's next
// move: follow → the maxYOffset SetState/forceRender's GotoBottom lands
// on; else the current scroll offset.
func (c *Chat) rebuildWindow() {
	total := len(c.selLines)
	vpH := c.vp.Height()
	if vpH < 1 {
		vpH = 1
	}
	off := c.vp.YOffset()
	if c.follow {
		off = total - vpH // exactly maxYOffset: GotoBottom's landing row
	}
	if off < 0 {
		off = 0
	}
	if off > total-vpH {
		off = total - vpH
	}
	if off < 0 {
		off = 0
	}
	lo0, hi0 := c.winTarget(off)
	lo, hi := winClamped(lo0, hi0, total)
	proj := make([]string, total)
	copy(proj[lo:hi], c.selLines[lo:hi])
	c.win = vpWindow{proj: proj, lo: lo, hi: hi, built: true}
	c.vp.SetContentLines(proj)
}

// syncWindow is the page-in seam — O(1) when the visible range sits inside
// the materialized window (every notch within the overscan), an in-place
// window MOVE when it steps across: blank the old range, copy the new from
// selLines, no viewport call (the projection is the viewport's own backing
// array, so the paint on this same msg cycle sees the fix). Called from
// the scroll arms in Update AND at the top of View (the catch-all for the
// seams the panel does not own — threads_opencode.go's PreserveAnchor
// SetYOffset, app-level Goto* helpers).
func (c *Chat) syncWindow() {
	if !c.win.built {
		return
	}
	total := len(c.selLines)
	if total == 0 || len(c.win.proj) != total {
		return // mid-rebuild — the next setConversationLines owns the post
	}
	vpH := c.vp.Height()
	if vpH < 1 {
		vpH = 1
	}
	yoff := c.vp.YOffset()
	if yoff >= c.win.lo && yoff+vpH <= c.win.hi {
		return // the visible slice is fully materialized
	}
	lo0, hi0 := c.winTarget(yoff)
	lo, hi := winClamped(lo0, hi0, total)
	if lo == c.win.lo && hi == c.win.hi {
		return
	}
	for i := c.win.lo; i < c.win.hi; i++ {
		c.win.proj[i] = ""
	}
	copy(c.win.proj[lo:hi], c.selLines[lo:hi])
	c.win.lo, c.win.hi = lo, hi
}

// mergeSpanInto offsets a block-local hit-map into the absolute one.
func mergeSpanInto(dst, src map[int]string, off int) {
	for r, v := range src {
		dst[off+r] = v
	}
}

// -------------------------------------------------------------------
// keys (invalidation fingerprints)
// -------------------------------------------------------------------

// keyMixer — FNV-1a over the SMALL key material of a block: the width /
// theme generation, the toggle flags, the tick while live, and (groups
// only) the resolved title + roster rollup strings. Message CONTENT is
// compared structurally in the borrow checks (old.src == m /
// old.lines == g.lines) — the reducer's cloned slices share string
// backings, so unchanged entries compare as slice headers, not bytes: the
// steady-state per-item cost drops to a map lookup + a header compare.
type keyMixer struct{ h uint64 }

func newKeyMixer() *keyMixer { return &keyMixer{h: 14695981039346656037} }

func (k *keyMixer) str(s string) *keyMixer {
	for i := 0; i < len(s); i++ {
		k.h ^= uint64(s[i])
		k.h *= 1099511628211
	}
	return k
}

func (k *keyMixer) boo(b bool) *keyMixer {
	if b {
		k.h ^= 1
		k.h *= 1099511628211
	}
	return k
}

func (k *keyMixer) num(n uint64) *keyMixer {
	for i := 0; i < 8; i++ {
		k.h ^= n & 0xff
		k.h *= 1099511628211
		n >>= 8
	}
	return k
}

func (k *keyMixer) done() uint64 { return k.h }

// renderGen — the width+theme generation stamped into every key: the wrap
// budgets (mdWidth for boss markdown, contentW for everything else) and
// the theme counter (bumped by RefreshTheme — styles are baked into
// rendered text, so a /theme switch must miss every block).
func (c *Chat) renderGen() uint64 {
	return newKeyMixer().
		num(uint64(uint32(c.mdWidth))).
		num(uint64(uint32(c.contentW()))).
		num(c.themeGen).
		done()
}

// borrowStaticBlock reuses a CONTENT-LESS block (the placeholder): pure
// key match.
func (c *Chat) borrowStaticBlock(id string, key uint64) *chatBlock {
	if id == "" || c.blockCache == nil {
		return nil
	}
	if old, ok := c.blockCache[id]; ok && !old.unstable && old.key == key {
		return old
	}
	return nil
}

// borrowMsgBlock reuses a message's cached render when the fingerprint AND
// THE SOURCE MESSAGE still match — the virtualization's core win: an
// unchanged message NEVER re-renders (no glamour, no fold, no builders).
func (c *Chat) borrowMsgBlock(m state.ChatMsg, key uint64) *chatBlock {
	if m.ID == "" || c.blockCache == nil {
		return nil
	}
	if old, ok := c.blockCache[m.ID]; ok && !old.unstable && old.key == key && old.src == m {
		return old
	}
	return nil
}

// borrowGroupBlock reuses a segment's cached render on fingerprint +
// line-set match.
func (c *Chat) borrowGroupBlock(id string, lines []state.ChatMsg, key uint64) *chatBlock {
	if c.blockCache == nil {
		return nil
	}
	old, ok := c.blockCache[id]
	if !ok || old.unstable || old.key != key || len(old.lines) != len(lines) {
		return nil
	}
	for i := range lines {
		if old.lines[i] != lines[i] {
			return nil
		}
	}
	return old
}

// pruneBlockCache drops identities no timeline item claimed this render
// (capChat's fuse, /clear): the cache stays transitive with the
// transcript's size.
func (c *Chat) pruneBlockCache() {
	if len(c.blockCache) == 0 {
		return
	}
	alive := make(map[string]struct{}, len(c.blocks))
	for _, b := range c.blocks {
		alive[b.id] = struct{}{}
	}
	for id := range c.blockCache {
		if _, ok := alive[id]; !ok {
			delete(c.blockCache, id)
		}
	}
}

// chatPadStr — the transcript's left-gutter pad as ONE shared string
// (separator rows and the cache's padded lines ride it; no per-row concat).
var chatPadStr = strings.Repeat(" ", chatPadL)

// padFragmentLines splits a rendered fragment into lines and pads each
// with the transcript's left gutter — done ONCE per block render; the
// cache carries the result across every borrow.
func padFragmentLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = chatPadStr + lines[i]
	}
	return lines
}
