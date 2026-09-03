// chat_window_test.go — the virtualization's own proofs (chat_window.go):
//
//	MATH        — window target/overscan/clamp arithmetic, the
//	              blank-model projection shape (posted rows ==
//	              len(selLines), materialized ⊆ viewport+overscan),
//	              and LAZY page-in across boundaries.
//	PIXELS      — at offsets across the whole transcript the windowed
//	              viewport paints BYTE-IDENTICALLY to the reference
//	              model (the pre-window viewport over the same padded
//	              lines), before AND after appends/updates/resizes.
//	INVALIDATE  — an append/update re-renders ONLY the touched blocks
//	              (pointer-survival probe); a width change, a theme
//	              switch, or a kind's toggle misses exactly its blocks.
//	SCROLL LOCK — appending while scrolled UP moves neither the scroll
//	              offset nor a single painted cell; the follow latch
//	              keeps the tail pinned.
//	PAGINATION  — the prepend-compensation contract (offset bumped by
//	              the exact rendered growth, read-row pinned) lands on
//	              materialized rows; selection extraction reads FULL
//	              lines across the blank model.
//
// Plus the benchmarks the change exists for: steady-state append /
// streaming-delta / scroll cost at 200 vs 2000 messages.
package panels

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// -------------------------------------------------------------------
// fixtures
// -------------------------------------------------------------------

// windowTranscript builds a MIXED transcript — boss markdown turns, user
// notes, think/tool/diff/office entries and a settled worker thread every
// 25 messages — the shape that exercises every block kind through the
// cache. Deterministic: IDs, timestamps, and texts derive from i only.
func windowTranscript(n int) []state.ChatMsg {
	chat := make([]state.ChatMsg, 0, n)
	for i := 0; i < n; i++ {
		at := int64(1000 + i*10)
		switch {
		case i%25 == 24:
			// a settled worker wave: two tool calls + a think for one agent
			chat = append(chat,
				state.ChatMsg{ID: fmt.Sprintf("w-%d-a", i), From: "tekton-1", Kind: wtoolKind,
					Text: "read · internal/panels/chat.go", Meta: "done\x1f5", At: at},
				state.ChatMsg{ID: fmt.Sprintf("w-%d-b", i), From: "tekton-1", Kind: wthinkKind,
					Text: "planning the fold", Meta: "call-x\x1f5", At: at + 1},
				state.ChatMsg{ID: fmt.Sprintf("w-%d-c", i), From: "tekton-1", Kind: wtoolKind,
					Text: "edit · internal/panels/chat.go", Meta: "done\x1f5", At: at + 2},
			)
		case i%7 == 3:
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("u-%d", i), From: "user", Kind: "user",
				Text: fmt.Sprintf("user note %d — keep it brief", i), At: at})
		case i%9 == 5:
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("th-%d", i), From: "boss", Kind: thinkKind,
				Text: fmt.Sprintf("reasoning line %d about the request", i), Meta: "", At: at})
		case i%11 == 7:
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("t-%d", i), From: "boss", Kind: toolKind,
				Text: fmt.Sprintf("bash · go build ./... (shard %d)", i), Meta: "done", At: at})
		case i%13 == 9:
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("d-%d", i), From: "boss", Kind: diffKind,
				Text: "@@ -1,2 +1,2 @@\n-old\n+new", Meta: "internal/panels/chat.go\x1f+1\x1f-1", At: at})
		default:
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("b-%d", i), From: "boss", Kind: "",
				Text: fmt.Sprintf("Boss turn **%d** with `code` and a list:\n\n- one\n- two\n- three", i), At: at})
		}
	}
	return chat
}

// refViewportView computes the PRE-WINDOW reference paint for rows
// [off, off+h) over the padded full lines: the bubbles viewport with
// wrap-free rows renders exactly the posted slice, framed by the same
// lipgloss width/height fill its View() applies.
func refViewportView(c *Chat, off int) string {
	vpW := c.vp.Width()
	vpH := c.vp.Height()
	h := vpH
	if off+h > len(c.selLines) {
		h = len(c.selLines) - off
	}
	if h < 0 {
		h = 0
	}
	return lipgloss.NewStyle().Width(vpW).Height(vpH).Render(
		strings.Join(c.selLines[off:off+h], "\n"))
}

// -------------------------------------------------------------------
// 1. window + overscan MATH
// -------------------------------------------------------------------

func TestWindowMathTargets(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 24)
	vpH := c.vp.Height()
	if vpH != 20 {
		t.Fatalf("SetSize(60,24) must leave 20 transcript rows, got %d", vpH)
	}
	if o := c.overscanSize(); o != vpH {
		t.Fatalf("overscan must be one viewport above AND below: %d != %d", o, vpH)
	}
	lo, hi := c.winTarget(100)
	if lo != 100-vpH || hi != 100+vpH+vpH {
		t.Fatalf("window = [offset-overscan, offset+height+overscan): got [%d,%d)", lo, hi)
	}
	// clamps: top underflow, bottom overflow, and the degenerate
	// smaller-than-one-window transcript (materialize everything).
	if lo, hi := winClamped(-5, 40, 200); lo != 0 || hi != 40 {
		t.Fatalf("top clamp: got [%d,%d)", lo, hi)
	}
	if lo, hi := winClamped(190, 260, 200); lo != 190 || hi != 200 {
		t.Fatalf("bottom clamp: got [%d,%d)", lo, hi)
	}
	if lo, hi := winClamped(0, 30, 12); lo != 0 || hi != 12 {
		t.Fatalf("degenerate: the whole %d rows must materialize, got [%d,%d)", 12, lo, hi)
	}
}

// TestWindowBlankModelProjection is the core shape proof: after a render
// the posted projection is FULL-LENGTH with real rows only inside the
// window, and materialization stays bounded by the viewport.
func TestWindowBlankModelProjection(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 24)
	// plain user rows: 1 rendered row each + 1 separator after the first
	const n = 400
	chat := make([]state.ChatMsg, 0, n)
	for i := 0; i < n; i++ {
		chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("m-%03d", i), From: "user", Kind: "user",
			Text: fmt.Sprintf("note %03d", i), At: int64(1000 + i*10)})
	}
	c.SetState(state.OfficeState{Tick: 1, Chat: chat})
	total := len(c.selLines)
	if want := 1 + (n-1)*2; total != want {
		t.Fatalf("%d one-row messages + separators must produce %d rows, got %d", n, want, total)
	}
	// the projection is full-length: the blank-height model keeps the
	// scroll space untouched. (This is the property the pagination seam
	// TranscriptRows()=len(selLines) — threads_opencode.go — reads: it
	// stays true verbatim under the window.)
	if len(c.win.proj) != total {
		t.Fatalf("projection must span the whole transcript: %d != %d", len(c.win.proj), total)
	}
	if got := c.vp.TotalLineCount(); got != total {
		t.Fatalf("the viewport's own line tally must equal the transcript: %d != %d", got, total)
	}
	// follow-latched: the window hugs the bottom — exactly maxYOffset
	// + overscan above, transcript end below.
	vpH := c.vp.Height()
	o := c.overscanSize()
	bottom := total - vpH
	wantLo := bottom - o
	if wantLo < 0 {
		wantLo = 0
	}
	if c.win.lo != wantLo || c.win.hi != total {
		t.Fatalf("follow window must be [%d,%d), got [%d,%d)", wantLo, total, c.win.lo, c.win.hi)
	}
	// inside the window: real rows; outside: blanks. And NO more than
	// height + 2×overscan rows are materialized.
	materialized := 0
	for i := 0; i < total; i++ {
		inWindow := i >= c.win.lo && i < c.win.hi
		if inWindow {
			materialized++
			if c.win.proj[i] != c.selLines[i] {
				t.Fatalf("row %d: materialized row must equal selLines", i)
			}
		} else if c.win.proj[i] != "" {
			t.Fatalf("row %d outside the window must be BLANK, got %q", i, c.win.proj[i])
		}
	}
	if materialized > vpH+2*o {
		t.Fatalf("materialization must stay bounded by viewport+2·overscan (%d), got %d", vpH+2*o, materialized)
	}
	// and the paint at the latch is the reference paint.
	if got, want := c.vp.View(), refViewportView(c, bottom); got != want {
		t.Fatalf("bottom paint must be byte-identical to the reference:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// -------------------------------------------------------------------
// 2. PIXEL IDENTITY across offsets / scrolling
// -------------------------------------------------------------------

// TestWindowPixelIdentityAcrossScroll sweeps offsets through a long mixed
// transcript and asserts byte-identical paints vs the full-content
// reference at EVERY stop — before content churn, after an appended tail,
// and after a mid-transcript update.
func TestWindowPixelIdentityAcrossScroll(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 30)
	chat := windowTranscript(300)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	total := len(c.selLines)
	if total <= c.vp.Height()+2*c.overscanSize() {
		t.Fatalf("the fixture must exceed one materialization window: %d rows", total)
	}
	assertPaintAt := func(off int) {
		t.Helper()
		c.vp.SetYOffset(off)
		c.syncWindow() // the View() seam, called directly for determinism
		if got, want := c.vp.View(), refViewportView(c, c.vp.YOffset()); got != want {
			t.Fatalf("offset %d: windowed paint diverges from the reference:\n--- got ---\n%s\n--- want ---\n%s",
				off, got, want)
		}
	}
	maxOff := total - c.vp.Height()
	for _, off := range []int{0, 1, 3, 17, maxOff / 3, maxOff / 2, 2 * maxOff / 3, maxOff - 1, maxOff} {
		assertPaintAt(off)
	}
	// churn: append to the tail while scrolled mid-transcript, and update
	// one mid-transcript message — repaint everywhere, still identical.
	c.follow = false
	c.vp.SetYOffset(maxOff / 2)
	chat = append(chat, state.ChatMsg{ID: "tail-new", From: "boss", Kind: "",
		Text: "fresh **tail** turn", At: 100000})
	mid := len(chat) / 2
	chat[mid] = state.ChatMsg{ID: chat[mid].ID, From: chat[mid].From, Kind: chat[mid].Kind,
		Meta: chat[mid].Meta, Text: "REPLACED mid-transcript body with a longer markdown paragraph that folds", At: chat[mid].At}
	c.SetState(state.OfficeState{Tick: 41, Chat: chat})
	maxOff = len(c.selLines) - c.vp.Height()
	for _, off := range []int{0, maxOff / 4, maxOff / 2, maxOff} {
		assertPaintAt(off)
	}
}

// TestWindowLazyPageIn: scrolling INSIDE the overscan moves no window
// boundary (the O(1)-notch seam); crossing it pages the window over
// (blanks the old range, copies the new) — the paint never differs.
func TestWindowLazyPageIn(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 24)
	chat := windowTranscript(200)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	c.follow = false
	vpH := c.vp.Height()
	total := len(c.selLines)
	// park mid-transcript (sync materializes around the park), then notch
	// once INSIDE the overscan: the window must NOT move (page-in is lazy
	// — it triggers only when the visible range reaches an edge).
	park := (total - vpH) / 2
	c.vp.SetYOffset(park)
	c.syncWindow()
	lo0, hi0 := c.win.lo, c.win.hi
	if y := c.vp.YOffset(); y < lo0 || y+vpH > hi0 {
		t.Fatalf("sanity: the park must sit in its window: off %d, window [%d,%d)", y, lo0, hi0)
	}
	c.vp.SetYOffset(park + 1)
	c.syncWindow()
	if c.win.lo != lo0 || c.win.hi != hi0 {
		t.Fatalf("a notch INSIDE the overscan must not move the window: [%d,%d) → [%d,%d)", lo0, hi0, c.win.lo, c.win.hi)
	}
	// jump past the window's far edge: page-in must move the window and
	// the paint must stay the reference's.
	far := hi0 + vpH
	if far+vpH > total {
		far = total - vpH
	}
	c.vp.SetYOffset(far)
	c.syncWindow()
	if c.win.lo <= lo0 && c.win.hi == hi0 {
		t.Fatalf("crossing the window edge must page-in (was [%d,%d), still [%d,%d))", lo0, hi0, c.win.lo, c.win.hi)
	}
	if yoff := c.vp.YOffset(); yoff < c.win.lo || yoff+vpH > c.win.hi {
		t.Fatalf("after page-in the visible range must be materialized: off %d, window [%d,%d)", yoff, c.win.lo, c.win.hi)
	}
	if got, want := c.vp.View(), refViewportView(c, c.vp.YOffset()); got != want {
		t.Fatalf("post-page-in paint must equal the reference:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	// the re-materialized window must carry EXACTLY its selLines slice…
	for i := c.win.lo; i < c.win.hi; i++ {
		if c.win.proj[i] != c.selLines[i] {
			t.Fatalf("row %d: the paged window must materialize selLines verbatim", i)
		}
	}
	// …and the abandoned range must be blanked (spot-check the old top).
	if lo0 < c.win.lo && c.win.proj[lo0] != "" {
		t.Fatalf("row %d left the window and must be blank, got %q", lo0, c.win.proj[lo0])
	}
}

// -------------------------------------------------------------------
// 3. append while SCROLLED UP — zero scroll-behavior change
// -------------------------------------------------------------------

// TestWindowAppendWhileScrolledUp: a reader parked mid-transcript gets a
// stream of tail appends — the offset doesn't move, the painted cells
// don't move, and the visible range never shows a blank row.
func TestWindowAppendWhileScrolledUp(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(70, 26)
	chat := windowTranscript(120)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	// park: scroll well into history, follow released (a wheel-up user).
	c.follow = false
	off := 30
	c.vp.SetYOffset(off)
	c.syncWindow()
	before := c.vp.View()
	beforeRows := len(c.selLines)
	// stream ten tail appends — the reader's cell content must be FROZEN
	// across every single one.
	for i := 0; i < 10; i++ {
		chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("stream-%d", i), From: "boss", Kind: "",
			Text: fmt.Sprintf("streamed tail %d", i), At: int64(20000 + i*10)})
		c.SetState(state.OfficeState{Tick: 41 + i, Chat: chat})
		if c.vp.YOffset() != off {
			t.Fatalf("append %d: YOffset moved (%d → %d) for a scrolled-up reader", i, off, c.vp.YOffset())
		}
		if got := c.vp.View(); got != before {
			t.Fatalf("append %d: the parked reader's paint changed:\n--- was ---\n%s\n--- now ---\n%s", i, before, got)
		}
		// the visible range is always fully materialized (no blank peek).
		if y := c.vp.YOffset(); y < c.win.lo || y+c.vp.Height() > c.win.hi {
			t.Fatalf("append %d: visible [%d,%d) not covered by window [%d,%d)", i, y, y+c.vp.Height(), c.win.lo, c.win.hi)
		}
	}
	if len(c.selLines) <= beforeRows {
		t.Fatalf("the appended tail must grow the transcript: %d → %d", beforeRows, len(c.selLines))
	}
	// and scrolling DOWN from there lands on the fresh rows (the tail is
	// reachable — the window pages across the growth).
	c.vp.SetYOffset(len(c.selLines) - c.vp.Height())
	c.syncWindow()
	got := ansi.Strip(c.vp.View())
	if !strings.Contains(got, "streamed tail 9") {
		t.Fatalf("the grown tail must be reachable by scrolling:\n%s", got)
	}
}

// TestWindowFollowLatch: with follow ON, an append re-aims the window at
// the new bottom and GotoBottom lands inside materialized rows (the
// bottom-row paint is the tail itself, not blanks).
func TestWindowFollowLatch(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(70, 26)
	chat := windowTranscript(80)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	if !c.follow {
		t.Fatal("a fresh chat follows the bottom")
	}
	chat = append(chat, state.ChatMsg{ID: "pinned-tail", From: "boss", Kind: "",
		Text: "the pinned tail row", At: 99999})
	c.SetState(state.OfficeState{Tick: 41, Chat: chat})
	yoff := c.vp.YOffset()
	if want := len(c.selLines) - c.vp.Height(); yoff != want {
		t.Fatalf("follow must stay pinned at the grown bottom: %d != %d", yoff, want)
	}
	if yoff < c.win.lo || yoff+c.vp.Height() > c.win.hi {
		t.Fatalf("the follow window must cover the bottom: off %d, window [%d,%d)", yoff, c.win.lo, c.win.hi)
	}
	if got := ansi.Strip(c.vp.View()); !strings.Contains(got, "the pinned tail row") {
		t.Fatalf("the bottom paint must carry the fresh tail:\n%s", got)
	}
}

// -------------------------------------------------------------------
// 4. INVALIDATION — block-granular
// -------------------------------------------------------------------

// blockPointers snapshots the per-identity cache pointers (the probe for
// "rendered" vs "borrowed": a stable pointer == NO re-render happened).
func blockPointers(c *Chat) map[string]*chatBlock {
	out := make(map[string]*chatBlock, len(c.blocks))
	for _, b := range c.blocks {
		out[b.id] = b
	}
	return out
}

// TestWindowInvalidationAppendUpdate — requirement: message append/update
// re-renders the AFFECTED blocks only.
func TestWindowInvalidationAppendUpdate(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 30)
	chat := windowTranscript(60)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	before := blockPointers(c)
	// (a) append: every old block survives; exactly one new block exists.
	chat = append(chat, state.ChatMsg{ID: "app-1", From: "user", Kind: "user", Text: "one more", At: 90001})
	c.SetState(state.OfficeState{Tick: 41, Chat: chat})
	afterApp := blockPointers(c)
	for id, p := range before {
		if afterApp[id] != p {
			t.Fatalf("append must not re-render block %q (pointer moved)", id)
		}
	}
	if _, ok := afterApp["app-1"]; !ok {
		t.Fatal("the appended message must materialize its own block")
	}
	if len(c.blocks) != len(before)+1 {
		t.Fatalf("block count must grow by one: %d → %d", len(before), len(c.blocks))
	}
	// (b) update ONE message: exactly its block re-renders.
	chat[10] = state.ChatMsg{ID: chat[10].ID, From: chat[10].From, Kind: chat[10].Kind,
		Text: "RE-WORDED body", Meta: chat[10].Meta, At: chat[10].At}
	c.SetState(state.OfficeState{Tick: 42, Chat: chat})
	afterUpd := blockPointers(c)
	changed := map[string]bool{}
	for id, p := range afterApp {
		if afterUpd[id] != p {
			changed[id] = true
		}
	}
	if len(changed) != 1 || !changed[chat[10].ID] {
		t.Fatalf("updating one message must re-render ONLY its block, moved: %v (want only %q)", changed, chat[10].ID)
	}
	// (c) the updated text actually painted (a borrow can't resurrect).
	if convo := ansi.Strip(c.renderConversation()); !strings.Contains(convo, "RE-WORDED body") {
		t.Fatalf("the updated block must render the new text:\n%s", convo)
	}
}

// TestWindowInvalidationResizeTheme — width and theme generations miss
// EVERY block; a same-width SetSize misses none.
func TestWindowInvalidationResizeTheme(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 30)
	chat := windowTranscript(40)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	base := blockPointers(c)
	// same-size SetSize: no re-render at all (the re-join path runs only
	// on a width flip — never here).
	c.SetSize(80, 30)
	for id, p := range blockPointers(c) {
		if base[id] != p {
			t.Fatalf("same-size SetSize must not re-render block %q", id)
		}
	}
	// width change + the next SetState: every block misses on the new
	// generation (the mdWidth/contentW budgets baked into the key).
	c.SetSize(64, 30)
	c.SetState(state.OfficeState{Tick: 41, Chat: append(chat, state.ChatMsg{
		ID: "after-resize", From: "user", Kind: "user", Text: "post", At: 90001})})
	resized := blockPointers(c)
	for id, p := range base {
		if resized[id] == p {
			t.Fatalf("a width change must invalidate block %q (the fold budgets moved)", id)
		}
	}
	// theme switch: RefreshTheme bumps the generation → everything misses.
	c.RefreshTheme()
	themed := blockPointers(c)
	for id, p := range resized {
		if themed[id] == p {
			t.Fatalf("a theme change must invalidate block %q (styles bake into text)", id)
		}
	}
}

// TestWindowInvalidationToggles — fold/thread/think/diff toggles re-render
// ONLY the kind of block they move (the pixels they own).
func TestWindowInvalidationToggles(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 30)
	chat := windowTranscript(60)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	before := blockPointers(c)
	// ctrl+d: diff blocks move, everything else borrows.
	c.ToggleDiffs()
	c.forceRender()
	afterDiff := blockPointers(c)
	for id, p := range before {
		isDiff := strings.HasPrefix(id, "d-")
		if (afterDiff[id] != p) != isDiff {
			t.Fatalf("ToggleDiffs must re-render diff blocks ONLY; block %q moved=%v", id, afterDiff[id] != p)
		}
	}
	// ctrl+t: think blocks move; diffs stay borrowed now.
	before2 := blockPointers(c)
	c.ToggleThink()
	c.forceRender()
	afterThink := blockPointers(c)
	for id, p := range before2 {
		isThink := strings.HasPrefix(id, "th-")
		if (afterThink[id] != p) != isThink {
			t.Fatalf("ToggleThink must re-render think blocks ONLY; block %q moved=%v", id, afterThink[id] != p)
		}
	}
	// a thread toggle: EXACTLY that segment's block moves.
	c.SetState(state.OfficeState{Tick: 41, Chat: chat}) // rebuild baselines post-toggles
	before3 := blockPointers(c)
	var aGroup string
	for _, b := range c.blocks {
		if strings.HasPrefix(b.id, "g:") {
			aGroup = strings.TrimPrefix(b.id, "g:")
			break
		}
	}
	if aGroup == "" {
		t.Fatal("fixture must contain a worker thread")
	}
	// find the agent NAME behind the segment id (first wtool line's From)
	agent := ""
	for _, m := range chat {
		if m.ID == aGroup {
			agent = m.From
			break
		}
	}
	c.ToggleThread(agent)
	c.forceRender()
	afterThread := blockPointers(c)
	for id, p := range before3 {
		mine := id == "g:"+aGroup
		if (afterThread[id] != p) != mine {
			t.Fatalf("ToggleThread(%q) must re-render ONLY its block; %q moved=%v", agent, id, afterThread[id] != p)
		}
	}
}

// -------------------------------------------------------------------
// 5. pagination seams + selection across the blank model
// -------------------------------------------------------------------

// TestWindowPaginationSeams — the PREPEND-COMPENSATION contract the
// pagination flow (threads_opencode.go's PreserveAnchor) drives through
// the viewport: bump the offset by EXACTLY the rendered growth after a
// spliced-in older page, and the previously-top row keeps its screen
// cell. The window must materialize the compensated rows BEFORE the
// paint (the View seam), and the row-space invariants the seam reads
// (len(selLines) == the banner's full-transcript count) must hold with
// the blank model in the middle.
func TestWindowPaginationSeams(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 10)
	var chat []state.ChatMsg
	for i := 11; i <= 40; i++ {
		chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("m-%02d", i), From: "user", Kind: "user",
			Text: fmt.Sprintf("m-%02d", i), At: int64(i * 10)})
	}
	c.SetState(state.OfficeState{Tick: 1, Chat: chat})
	c.follow = false
	c.vp.SetYOffset(4)
	before := len(c.selLines) // == TranscriptRows() when the seam exists
	topRow := c.selLines[4]
	var older []state.ChatMsg
	for i := 1; i <= 10; i++ {
		older = append(older, state.ChatMsg{ID: fmt.Sprintf("m-%02d", i), From: "user", Kind: "user",
			Text: fmt.Sprintf("m-%02d", i), At: int64(i * 10)})
	}
	merged := make([]state.ChatMsg, 0, len(older)+len(chat))
	merged = append(merged, older...)
	merged = append(merged, chat...)
	c.SetState(state.OfficeState{Tick: 2, Chat: merged})
	// PreserveAnchor's mechanism verbatim: bump by the rendered growth.
	c.vp.SetYOffset(c.vp.YOffset() + len(c.selLines) - before)
	growth := len(c.selLines) - before
	if growth != 20 { // 10 message rows + their 10 separator rows
		t.Fatalf("10 one-row messages + separators must grow 20 rows, got %d", growth)
	}
	wantOff := 4 + growth
	if c.vp.YOffset() != wantOff {
		t.Fatalf("anchor compensation must land at %d, got %d", wantOff, c.vp.YOffset())
	}
	// the View seam materializes the compensated rows BEFORE paint —
	// the reading row must be REAL content, not a window blank.
	_ = c.View()
	if got := c.selLines[c.vp.YOffset()]; got != topRow {
		t.Fatalf("the compensated top row must be the same content: %q != %q", got, topRow)
	}
	if y := c.vp.YOffset(); y < c.win.lo || y+c.vp.Height() > c.win.hi {
		t.Fatalf("the compensated viewport must be materialized: off %d, window [%d,%d)", y, c.win.lo, c.win.hi)
	}
	if len(c.win.proj) != len(c.selLines) {
		t.Fatalf("the posted projection must still span the transcript: %d != %d", len(c.win.proj), len(c.selLines))
	}
}

// TestWindowSelectionExtractionAcrossBlanks — selection endpoints resolve
// in FULL transcript space (selLines never blanks): a selection spanning
// far past the materialized window extracts the REAL text, not blanks.
func TestWindowSelectionExtractionAcrossBlanks(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 12)
	chat := windowTranscript(160)
	c.SetState(state.OfficeState{Tick: 40, Chat: chat})
	total := len(c.selLines)
	// selection from deep history to the tail — rows far outside the
	// bottom-anchored window on BOTH ends.
	winLo, winHi := c.win.lo, c.win.hi
	if winLo <= total/2 && winHi >= total/2 {
		t.Fatalf("fixture sanity: the window [%d,%d) must NOT already cover row %d", winLo, winHi, total/2)
	}
	c.sel = selState{active: false, finalized: true, aRow: 0, aCol: 0, hRow: total - 1, hCol: c.w - 1}
	got := c.selText()
	if strings.TrimSpace(got) == "" {
		t.Fatal("a transcript-spanning selection must extract text")
	}
	// markers from DEEP history (the window blanks there) and the tail
	// must BOTH come through — the extraction source never windows.
	// (user notes are verbatim plain rows: "user note <i> — keep it brief".)
	if !strings.Contains(got, "user note 3 — keep it brief") {
		t.Fatalf("deep-history marker must survive extraction (off-window rows!):\n%.400s", got)
	}
	if !strings.Contains(got, "user note 157 — keep it brief") {
		t.Fatalf("tail marker must survive extraction:\n%.400s", got)
	}
	// the window projection was untouched by the endpoints.
	if len(c.win.proj) != total {
		t.Fatalf("selection must not disturb the projection length: %d != %d", len(c.win.proj), total)
	}
}

// -------------------------------------------------------------------
// 6. BENCHMARKS — the point of the change
// -------------------------------------------------------------------

// benchChat returns a panel preloaded with an n-message mixed transcript,
// follow-latched at the bottom (the steady state while streaming).
func benchChat(b *testing.B, n int) (*Chat, state.OfficeState) {
	c := NewChat(nil)
	c.SetSize(100, 40)
	st := state.OfficeState{Tick: 40, Chat: windowTranscript(n)}
	c.SetState(st)
	return c, st
}

// appendMsg returns a fresh tail message for iteration i (unique ID so the
// SetState revision gate keeps passing).
func appendMsg(st state.OfficeState, i int) state.OfficeState {
	next := make([]state.ChatMsg, len(st.Chat), len(st.Chat)+1)
	copy(next, st.Chat)
	next = append(next, state.ChatMsg{
		ID: fmt.Sprintf("bench-append-%d", i), From: "boss", Kind: "",
		Text: fmt.Sprintf("steady-state streamed **turn %d** with some markdown body text", i),
		At:   int64(900000 + i*10)})
	return state.OfficeState{Tick: st.Tick + 1, Chat: next}
}

// BenchmarkChatAppendSteadyState200 — one tail append per iteration over a
// 200-message transcript.
func BenchmarkChatAppendSteadyState200(b *testing.B) {
	c, st := benchChat(b, 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st = appendMsg(st, i)
		c.SetState(st)
	}
}

// BenchmarkChatAppendSteadyState2000 — the same append over a 2000-message
// transcript: the windowed panel's per-append cost must stay near the
// 200-time (bounded by the viewport + cheap splices), NOT the
// transcript's glamour cost.
func BenchmarkChatAppendSteadyState2000(b *testing.B) {
	c, st := benchChat(b, 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st = appendMsg(st, i)
		c.SetState(st)
	}
}

// BenchmarkChatStreamDelta2000 — a streaming boss bubble at the tail of a
// 2000-message transcript: the LAST block re-renders each tick, every
// other block borrows.
func BenchmarkChatStreamDelta2000(b *testing.B) {
	c := NewChat(nil)
	c.SetSize(100, 40)
	base := windowTranscript(2000)
	b.ResetTimer()
	var body strings.Builder
	for i := 0; i < b.N; i++ {
		body.WriteString("delta ")
		next := make([]state.ChatMsg, len(base)+1)
		copy(next, base)
		next[len(base)] = state.ChatMsg{ID: "stream-tail", From: "boss", Kind: "",
			Text: body.String(), Pending: true, At: 900000}
		c.SetState(state.OfficeState{Tick: 41 + i, Chat: next})
		if body.Len() > 4000 {
			body.Reset()
		}
	}
}

// BenchmarkChatScrollNotch2000 — one-row scrolls through a parked
// mid-transcript window (the interactive jank path): syncWindow + paint.
func BenchmarkChatScrollNotch2000(b *testing.B) {
	c := NewChat(nil)
	c.SetSize(100, 40)
	c.SetState(state.OfficeState{Tick: 40, Chat: windowTranscript(2000)})
	c.follow = false
	mid := (len(c.selLines) - c.vp.Height()) / 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.vp.SetYOffset(mid + i%200)
		c.syncWindow()
		_ = c.vp.View()
	}
}

// BenchmarkChatColdRender2000 — reference point: a FULL cold render of the
// 2000-message transcript (what every SetState paid before the cache).
func BenchmarkChatColdRender2000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := NewChat(nil)
		c.SetSize(100, 40)
		c.SetState(state.OfficeState{Tick: 40, Chat: windowTranscript(2000)})
	}
}
