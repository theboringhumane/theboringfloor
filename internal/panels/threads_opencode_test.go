// threads_opencode_test.go — behavior proofs for the opencode-style
// thread renderer (threads_opencode.go), LOCKED frame:
//
//	⠿ Explore Task — Scout question kinds recon
//	  ↳ Read internal/panels/chat.go
//
//	(a) a LIVE collapsed thread renders the office-tick braille glyph
//	    (c.tick%len(threadLiveFrames)) + its "<Kind> Task — <task>" title
//	    and NOTHING else — no rollup while running — and its second line
//	    is the dim BARE "  ↳ <Verb> <rest>" sneak at the NEWEST tool
//	    (the state mark is gone from the peek);
//	(b) a DONE thread dims a "✓", KEEPS the collapsed rollup
//	    ("(· N tool calls ✓ done)"), and no live-glyph braille frame ever
//	    shows while nothing is live;
//	(c) an expanded thread lists its "[tool] <shaped> <state mark>" rows
//	    under the SAME header, then the ↳ sneak AGAIN (still bare) as the
//	    "current task" line, then the dim closing summary;
//	(d) the "ctrl+g · view subagents" hint row appears ONLY while ≥1
//	    rendered thread is live — gone when the threads go stale and when
//	    /tools is off;
//	(e) ClickRow on the ONE header row toggles, on the ONE sneak row
//	    toggles in BOTH states, on an EXPANDED internal tool row does NOT
//	    (false);
//	(f) the pending row renders the breathing block-glyph column + the
//	    existing typing text in EXACTLY one row (SetSize budget intact);
//	(g) the live glyph is a pure function of the office tick: tick 20 →
//	    threadLiveFrames[6] "⠾", tick 21 → frame 0 "⠿" — no timers;
//	(h) a trailing wthink never steals the ↳ sneak — the peek pins the
//	    thread's NEWEST TOOL line, display-SHAPED ("edit · lex.go" →
//	    "Edit lex.go"), the thought rolls up in the "· M think" count
//	    (and a think-ONLY thread falls back to "thinking · N lines");
//	(i) a /stop-stopped thread reads "✗ <title> (· … ✗ stopped)",
//	    force-collapses under the ctrl+g baseline, and re-opens only on
//	    an explicit per-agent expand (closing line "· … ✗ stopped");
//	(j) shapeToolText is IDEMPOTENT: the reducer's "<verb> · <rest>"
//	    shape maps to "<Verb> <rest>", target-shaped text rides through.
//	(k) a thread group TOPPING the timeline (born before every chat
//	    entry) registers its hit-map at the TRIMMED top — the header on
//	    content row 0, not base+2 (renderConversation's TrimLeft eats
//	    the block's "\n\n" lead there) — so a click on the visual header
//	    row toggles, and no registration hijacks the next item's row.
//	(l) per-call diffs (Kind wdiff, "wdiff-<agent>-<callid>") pin INSIDE
//	    the thread: collapsed shows only the sneak's dim "· +A -D" suffix,
//	    expanded gains the tool row's suffix AND a one-row "↳ diff · path
//	    +A -D" sub-row DIRECTLY beneath that tool row — rollups count the
//	    diff as NEITHER tool nor think;
//	(m) the ↳ diff sub-row owns the toolDiffRows hit-map: ClickRow there
//	    opens/closes the parsed body (diffClip machinery verbatim) without
//	    touching the thread's own toggle; body rows never register.
//	(n)–(s) the older-history pagination seams (the section at the bottom
//	    of threads_opencode.go): the ThreadPager walk controller
//	    (Seed/StartOlder/FinishOlder/FailOlder/ResetFailures),
//	    PrependOlder, AtTranscriptTop, TranscriptRows and PreserveAnchor —
//	(n) a prepend splices a fetched page ahead of the transcript head in
//	    exact page order, skips ids already present (no dupes, a page
//	    never duplicates itself), returns fresh-count only, and stays
//	    PURE of offset / follow latch / pager state;
//	(o) the hasMore walk: 500 canned rows at ThreadOlderPageSize pages —
//	    the boot page + exactly 9 older hops, page heads his-451…his-001,
//	    then the top latch refuses further hops;
//	(p) the in-flight + failure guards: unseeded refuses, one hop at a
//	    time, three straight failures back off (the cursor NEVER moves),
//	    ResetFailures re-arms, the top latch outranks a reset;
//	(q) AtTranscriptTop probes the first viewport row and nothing else;
//	(r) PreserveAnchor keeps the reading row BYTE-IDENTICAL across a
//	    head splice (growth exactly 2 rows per spliced message);
//	(s) the 500-message worked example end to end: 50 boot rows + 9
//	    walked pages, 999 rendered rows, the reading row surviving all
//	    nine compensations byte-for-byte.
//
// No clocks, no sleeps: every office tick and wtool meta-tick is a
// literal (Meta carries "state␟tick" like the reducer writes —
// parseWtoolMeta reads it back), and the live glyph is c.tick-indexed.
package panels

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	state "github.com/theboringhumane/theboringoffice/internal/state"
)

// newOpencodeChat builds the deterministic two-thread fixture every
// opencode proof walks at office tick 20: skopos-1 (SCOUT role →
// "Explore") LIVE with its newest tool call still running, tekton-1
// (DEVELOPER) DONE, and a pending EMPTY boss reply so the typing row
// carries the block bar. Thread birth slots interleave the user turn by
// At; the done thread logs activity 10+ ticks back but inside
// wtoolStaleTicks — its IDLE sprite is what settles it. The wtool texts
// ride the ASPIRATIONAL target shape ("Read internal/panels/chat.go")
// to prove the shaping is idempotent; proof (h) feeds the reducer's own
// "<verb> · <rest>" shape for the mapping half.
func newOpencodeChat(t *testing.T, w, h int) *Chat {
	t.Helper()
	return newOpencodeChatAtTick(t, w, h, 20)
}

// newOpencodeChatAtTick — newOpencodeChat at an arbitrary office tick.
// The live glyph is c.tick%len(threadLiveFrames): tick 20 → frame 6
// ("⠾"), tick 21 → frame 0 ("⠿" — the locked frame's opener). The
// fixture's meta-ticks stay 1-11 ticks back, so every tick up to ~130
// keeps skopos-1 LIVE (busy sprite + activity inside wtoolStaleTicks).
func newOpencodeChatAtTick(t *testing.T, w, h, tick int) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(w, h)
	c.SetState(state.OfficeState{
		Tick: tick,
		Employees: []state.Employee{
			{ID: "sco-1", Name: "skopos-1", Role: state.RoleScout,
				Sprite: state.SpriteWorking, Task: "Scout question kinds recon"}, // LIVE
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Extend state+backend question kinds"}, // returned
		},
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "ship the question kinds", At: 10},
			// skopos-1 — LIVE: busy sprite, freshest activity 1 tick old
			{ID: "s1", From: "skopos-1", Kind: wtoolKind, Text: "List internal/panels", Meta: "done\x1f18", At: 20},
			{ID: "s2", From: "skopos-1", Kind: wtoolKind, Text: "Read internal/panels/chat.go", Meta: "running\x1f19", At: 30},
			// tekton-1 — DONE: idle sprite settles the thread
			{ID: "x1", From: "tekton-1", Kind: wtoolKind, Text: "Grep questionKind", Meta: "done\x1f9", At: 40},
			{ID: "x2", From: "tekton-1", Kind: wtoolKind, Text: "Read internal/backend/backend.go", Meta: "done\x1f10", At: 50},
			{ID: "b1", From: "boss", Kind: "boss", Pending: true, At: 60}, // the typing row (block bar)
		},
	})
	return c
}

// staleChat re-feeds the fixture far past the staleness horizon with idle
// sprites at 84 cols (every settled header's rollup fits ONE row inside
// the chatPadL/chatPadR-inset budget — exact pins, no clips): both
// threads go done, the hint row and the loading row hide.
func staleChat(t *testing.T) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(84, 24)
	c.SetState(state.OfficeState{
		Tick: 1000,
		Employees: []state.Employee{
			{ID: "sco-1", Name: "skopos-1", Role: state.RoleScout,
				Sprite: state.SpriteAtDesk, Task: "Scout question kinds recon"},
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Extend state+backend question kinds"},
		},
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "ship the question kinds", At: 10},
			{ID: "s1", From: "skopos-1", Kind: wtoolKind, Text: "List internal/panels", Meta: "done\x1f18", At: 20},
			{ID: "s2", From: "skopos-1", Kind: wtoolKind, Text: "Read internal/panels/chat.go", Meta: "done\x1f19", At: 30},
			{ID: "x1", From: "tekton-1", Kind: wtoolKind, Text: "Grep questionKind", Meta: "done\x1f9", At: 40},
			{ID: "x2", From: "tekton-1", Kind: wtoolKind, Text: "Read internal/backend/backend.go", Meta: "done\x1f10", At: 50},
		},
	})
	return c
}

// TestThreadLiveSpinnerTitleAndSneak is proof (a): the live thread's
// header is the office-tick braille glyph (tick 20 → frame 6 "⠾") +
// "Explore Task — Scout question kinds recon" (scout maps to opencode's
// Explore kind) and NOTHING ELSE — no rollup while running — and its
// collapsed second line is the dim BARE "  ↳ Read
// internal/panels/chat.go" sneak at the RUNNING latest call, not the
// older one, with no state mark trailing.
func TestThreadLiveSpinnerTitleAndSneak(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	convo := ansi.Strip(c.renderConversation())
	for _, want := range []string{
		"⠾ Explore Task — Scout question kinds recon",
		"  ↳ Read internal/panels/chat.go",
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("live thread missing shape %q:\n%s", want, convo)
		}
	}
	// row-seam rules the neighbor DONE thread can't shadow: the LIVE
	// header carries NO rollup, and its sneak is BARE (no ✓/✗/… running)
	for _, ln := range strings.Split(convo, "\n") {
		if strings.Contains(ln, "Explore Task — Scout question kinds recon") && strings.Contains(ln, "(·") {
			t.Fatalf("a LIVE collapsed header must not carry the rollup, got %q", ln)
		}
		if strings.Contains(ln, "↳ Read internal/panels/chat.go") {
			if strings.Contains(ln, "… running") || strings.Contains(ln, "✓") || strings.Contains(ln, "✗") {
				t.Fatalf("the sneak is BARE — no state mark trails it, got %q", ln)
			}
		}
	}
	// the OLDER tool call must not be the sneak (it's the expanded-only
	// history) — and nothing renders the expanded list while collapsed
	if strings.Contains(convo, "[tool] ") || strings.Contains(convo, "↳ List") {
		t.Fatalf("collapsed live thread must sneak ONLY the newest entry:\n%s", convo)
	}
}

// TestThreadDoneCheckNoSpinner is proof (b): settled threads dim a "✓"
// glyph, the collapsed header KEEPS the old summary card's rollup
// ("(· N tool calls ✓ done)" — exact at 80 cols), the sneak is the bare
// shaped peek, the DEVELOPER role reads as "Developer", and NO
// live-glyph braille frame (the whole threadLiveFrames set is scanned)
// appears anywhere once nothing is live.
func TestThreadDoneCheckNoSpinner(t *testing.T) {
	c := staleChat(t)
	convo := ansi.Strip(c.renderConversation())
	for _, want := range []string{
		"✓ Explore Task — Scout question kinds recon (· 2 tool calls ✓ done)",
		"✓ Developer Task — Extend state+backend question kinds (· 2 tool calls ✓ done)",
		"  ↳ Read internal/panels/chat.go",
		"  ↳ Read internal/backend/backend.go",
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("done thread missing shape %q:\n%s", want, convo)
		}
	}
	for _, frame := range threadLiveFrames {
		if strings.Contains(convo, frame) {
			t.Fatalf("a DONE thread must never show live-glyph frame %q:\n%s", frame, convo)
		}
	}
}

// TestThreadExpandedListsToolRows is proof (c): a per-agent expand keeps
// the SAME (rollup-free) header, lists the merged "[tool] <shaped>
// <state mark>" rows 2-cell indented beneath it, then re-shows the ↳
// sneak — still BARE — as the "current task" line AFTER the last tool
// row, and closes with the dim summary line — in that exact order.
func TestThreadExpandedListsToolRows(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	c.ToggleThread("skopos-1")
	convo := ansi.Strip(c.renderConversation())
	fmt.Println("---- OPENCODE EXPANDED THREAD (60 cols: skopos-1 zoomed live, ansi-stripped) ----")
	for _, ln := range strings.Split(convo, "\n") {
		if strings.Contains(ln, "Explore Task") || strings.Contains(ln, "[tool] ") ||
			strings.Contains(ln, "  ↳ ") || strings.Contains(ln, "  · ") {
			fmt.Printf("%2d|%s|\n", len([]rune(ln)), ln)
		}
	}
	fmt.Println("---- END THREAD ----")
	for _, want := range []string{
		"⠾ Explore Task — Scout question kinds recon",
		"  [tool] List internal/panels ✓",
		"  [tool] Read internal/panels/chat.go … running",
		"  ↳ Read internal/panels/chat.go",
		"  · 2 tool calls ✓ done",
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("expanded thread missing row %q:\n%s", want, convo)
		}
	}
	// the expanded sneak is BARE too — only the [tool] rows carry marks
	for _, ln := range strings.Split(convo, "\n") {
		if strings.Contains(ln, "↳ Read internal/panels/chat.go") && strings.Contains(ln, "… running") {
			t.Fatalf("the ↳ sneak never carries the state mark, got %q", ln)
		}
	}
	// ORDER: header < tool rows < ↳ sneak (the current-task line) <
	// closing summary
	iHead := strings.Index(convo, "⠾ Explore Task")
	iTool := strings.Index(convo, "  [tool] List internal/panels ✓")
	iSneak := strings.Index(convo, "  ↳ Read internal/panels/chat.go")
	iClose := strings.Index(convo, "  · 2 tool calls ✓ done")
	if !(iHead < iTool && iTool < iSneak && iSneak < iClose) {
		t.Fatalf("expanded thread must run header → tools → sneak → closing (h=%d t=%d s=%d c=%d):\n%s",
			iHead, iTool, iSneak, iClose, convo)
	}
	// the quiet neighbor stays collapsed while its sibling zooms
	if !strings.Contains(convo, "✓ Developer Task — Extend state+backend question kinds") ||
		!strings.Contains(convo, "  ↳ Read internal/backend/backend.go") {
		t.Fatalf("the done thread must survive its neighbor's expand:\n%s", convo)
	}
}

// TestThreadHintRowOnlyWhileLive is proof (d): the dim
// "ctrl+g · view subagents" hint trails the LAST thread block only while
// ≥1 rendered thread is live; stale threads drop it, and /tools off (no
// threads rendered) drops it too.
func TestThreadHintRowOnlyWhileLive(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	convo := ansi.Strip(c.renderConversation())
	iTekton := strings.Index(convo, "Developer Task —")
	iHint := strings.Index(convo, threadHintText)
	if iHint < 0 || iHint < iTekton {
		t.Fatalf("the hint row must trail the LAST thread block (hint at %d, last thread at %d):\n%s", iHint, iTekton, convo)
	}

	stale := staleChat(t)
	if convo := ansi.Strip(stale.renderConversation()); strings.Contains(convo, threadHintText) {
		t.Fatalf("the hint row must die with the last LIVE thread:\n%s", convo)
	}

	off := newOpencodeChat(t, 60, 24)
	off.SetShowTools(false)
	convo = ansi.Strip(off.renderConversation())
	if strings.Contains(convo, threadHintText) || strings.Contains(convo, "Task —") {
		t.Fatalf("/tools off renders no threads, so no hint row may show:\n%s", convo)
	}
}

// TestThreadClickToggleSemantics is proof (e): the thread toggles from
// its FRAME rows — whole-bubble clicking, state-conditional: while
// COLLAPSED the ONE header row and the ONE ↳ sneak row toggle (both
// single-row under the clip contract); while EXPANDED the header and the
// CLOSING summary rows (the bubble's head and tail) toggle, and the
// internal tool rows AND the mid-list ↳ sneak (the "current task" line
// is content, not a frame edge) fall through UNCLAIMED. A closing-row
// click collapses back to the collapsed 2-row registration, and a
// multi-row closing summary registers EVERY one of its rows.
func TestThreadClickToggleSemantics(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	rows := func(agent string) []int {
		var lines []int
		for i := 0; i < 50; i++ {
			if c.threadRows[i] == agent {
				lines = append(lines, i)
			}
		}
		return lines
	}
	rowOf := func(needle string) int {
		for i, ln := range strings.Split(ansi.Strip(c.renderConversation()), "\n") {
			if strings.Contains(ln, needle) {
				return i
			}
		}
		return -1
	}
	// collapsed: ONE header row + ONE sneak row (single-row contract)
	scout := rows("skopos-1")
	if len(scout) != 2 {
		t.Fatalf("a collapsed thread must register its header + sneak (2 rows), got %v", scout)
	}
	// header toggles
	if !c.ClickRow(3, scout[0]) {
		t.Fatal("click on the header row was not claimed")
	}
	tbAssertExpanded(t, c, "skopos-1", true, "after header click")
	// expanded: the FRAME rows toggle — the header (top) + the CLOSING
	// summary row (bottom); the second registered row must BE the
	// closing row's visual line
	scout = rows("skopos-1")
	if len(scout) != 2 {
		t.Fatalf("an expanded thread must register header + closing (2 rows), got %v", scout)
	}
	closingRow := rowOf("  · 2 tool calls ✓ done")
	if closingRow < 0 || scout[1] != closingRow {
		t.Fatalf("the expanded thread's second registered row must BE the closing summary's visual row %d, got %v", closingRow, scout)
	}
	// the internal tool rows must NOT toggle
	if c.ClickRow(3, scout[0]+1) {
		t.Fatalf("click on an expanded internal tool row (line %d) must not be claimed", scout[0]+1)
	}
	tbAssertExpanded(t, c, "skopos-1", true, "after internal-row click (no-op)")
	// the mid-list ↳ sneak (the "current task" line) is content while
	// expanded: its click must pass through — the CLOSING rows own the
	// bottom of the bubble now
	sneakRow := rowOf("  ↳ Read internal/panels/chat.go")
	if sneakRow <= scout[0] || sneakRow >= scout[1] {
		t.Fatalf("the expanded sneak must sit between the frame rows %v, got %d", scout, sneakRow)
	}
	if c.ClickRow(3, sneakRow) {
		t.Fatalf("click on the expanded sneak row (line %d) must pass through unclaimed", sneakRow)
	}
	tbAssertExpanded(t, c, "skopos-1", true, "after expanded-sneak click (no-op)")
	// the CLOSING summary row collapses the thread
	if !c.ClickRow(3, scout[1]) {
		t.Fatal("click on the closing summary row was not claimed")
	}
	tbAssertExpanded(t, c, "skopos-1", false, "after closing-row click")
	// …restoring the collapsed set of exactly 2: header + sneak — and the
	// sneak row toggles from there as well
	scout = rows("skopos-1")
	if len(scout) != 2 {
		t.Fatalf("re-collapsed thread must register header + sneak again, got %v", scout)
	}
	if !c.ClickRow(3, scout[1]) {
		t.Fatal("click on the collapsed sneak row was not claimed")
	}
	tbAssertExpanded(t, c, "skopos-1", true, "after collapsed-sneak click")
}

// TestThreadClickExpandedMultiRowClosing — the closing summary folded
// over MULTIPLE rows (narrow panel) registers EVERY one of its rows: a
// folded summary row is part of the bubble's bottom frame edge, so each
// of them toggles. The tool rows and the mid-list sneak between header
// and closing stay unclaimed.
func TestThreadClickExpandedMultiRowClosing(t *testing.T) {
	c := newOpencodeChat(t, 26, 24) // narrow: the closing rollup folds to 2 rows
	c.ToggleThread("skopos-1")
	var scout []int
	for i := 0; i < 50; i++ {
		if c.threadRows[i] == "skopos-1" {
			scout = append(scout, i)
		}
	}
	if len(scout) != 3 {
		t.Fatalf("expanded thread with a 2-row closing summary must register header + BOTH closing rows, got %v", scout)
	}
	// the registered rows are CONTIGUOUS around nothing: header, then
	// the two closing rows at the very bottom of the block
	if scout[2] != scout[1]+1 {
		t.Fatalf("the multi-row closing must register contiguous rows, got %v", scout)
	}
	// the LAST closing row toggles too
	if !c.ClickRow(3, scout[2]) {
		t.Fatal("click on the final closing row was not claimed")
	}
	tbAssertExpanded(t, c, "skopos-1", false, "after final-closing-row click")
}

// TestPendingRowBlockBarOneRow is proof (f): the typing row is the
// breathing block-glyph column + the existing "<boss> is typing…" text in
// EXACTLY one row — the SetSize budget (whole View == h rows) does not
// move, and the retired caret glyph never comes back.
func TestPendingRowBlockBarOneRow(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	view := ansi.Strip(c.View())
	rows := strings.Split(view, "\n")
	if len(rows) != 24 {
		t.Fatalf("the restyle must not move the row budget: View drew %d rows, want 24:\n%s", len(rows), view)
	}
	ri := typingRowIdx(rows, "is typing…")
	if ri < 0 {
		t.Fatalf("the typing row is missing:\n%s", view)
	}
	di := chatDividerRow(rows)
	if ri != di+1 {
		t.Fatalf("the typing row must be the FIRST row below the divider (divider %d, typing %d):\n%s", di, ri, view)
	}
	// the block column: the deterministic tick-20 frame of the breathing
	// bar, bold-magenta in front of the unchanged busy text
	want := pendingBlockBar(20) + " boss is typing…"
	if !strings.Contains(rows[ri], want) {
		t.Fatalf("the pending row must be the block bar + existing text %q, got %q:\n%s", want, rows[ri], view)
	}
	if strings.Contains(view, "▌") {
		t.Fatalf("the retired caret glyph must never appear:\n%s", view)
	}
	if r := pendingBlockBar(0); r != "█▇▆▅▄▃▂▁" {
		t.Fatalf("the deterministic first bar frame must be %q, got %q", "█▇▆▅▄▃▂▁", r)
	}
	if pendingBlockBar(1) == pendingBlockBar(0) {
		t.Fatal("the bar must BREATHE across ticks")
	}
}

// TestLiveGlyphCyclesOffTheOfficeTick is proof (g): the live header's
// braille glyph is a PURE FUNCTION of the office tick — no spinner
// model, no timer: tick 20 renders threadLiveFrames[6] ("⠾"), tick 21
// wraps to frame 0 ("⠿" — the locked frame's opener), and the done
// thread's "✓" never moves. Every frame is exactly ONE cell wide (the
// 2-cell glyph field's budget).
func TestLiveGlyphCyclesOffTheOfficeTick(t *testing.T) {
	c := newOpencodeChat(t, 60, 24)
	if convo := ansi.Strip(c.renderConversation()); !strings.Contains(convo, "⠾ Explore Task") {
		t.Fatalf("tick 20 must render threadLiveFrames[6] (⠾):\n%s", convo)
	}
	c21 := newOpencodeChatAtTick(t, 60, 24, 21)
	convo := ansi.Strip(c21.renderConversation())
	if !strings.Contains(convo, "⠿ Explore Task") {
		t.Fatalf("tick 21 must wrap around to threadLiveFrames[0] (⠿):\n%s", convo)
	}
	if strings.Contains(convo, "⠾") {
		t.Fatalf("the old frame must be gone at the new tick:\n%s", convo)
	}
	if !strings.Contains(convo, "✓ Developer Task") {
		t.Fatalf("the done thread's ✓ must never animate:\n%s", convo)
	}
	for _, f := range threadLiveFrames {
		if w := ansi.StringWidth(f); w != 1 {
			t.Fatalf("live frame %q must be 1 cell wide, got %d", f, w)
		}
	}
}

// TestThreadSneakPinsLatestToolOverThink is proof (h): when a thread's
// NEWEST entry is a thought, the collapsed ↳ sneak still peeks the last
// TOOL line (thoughts roll up in the "· M think" summary counts — they
// never lead the peek); the peek is the reducer's text SHAPED
// ("edit · lex.go" → "Edit lex.go") and BARE; a thread with NO tool
// line at all falls back to the "thinking · N lines" peek so it keeps a
// second row.
func TestThreadSneakPinsLatestToolOverThink(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 24) // wide: every settled header + rollup fits one row — exact pins
	c.SetState(state.OfficeState{
		Tick: 50,
		Employees: []state.Employee{
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Fix the lexer"},
			{ID: "dev-2", Name: "tekton-2", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Muse only"},
		},
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "one", At: 10},
			// tekton-1 — tool THEN thought: the thought is newest, the
			// sneak still pins the (shaped) tool line
			{ID: "x1", From: "tekton-1", Kind: wtoolKind, Text: "edit · lex.go", Meta: "done\x1f5", At: 20},
			{ID: "x2", From: "tekton-1", Kind: wthinkKind, Text: "a thought\nover two lines", Meta: "c1\x1f6", At: 30},
			// tekton-2 — think-ONLY thread: the fallback peek
			{ID: "y1", From: "tekton-2", Kind: wthinkKind, Text: "musing", Meta: "c2\x1f7", At: 40},
		},
	})
	convo := ansi.Strip(c.renderConversation())
	for _, want := range []string{
		// the LAST TOOL line leads the sneak, shaped and bare…
		"  ↳ Edit lex.go",
		// …and the thought rolls up in the settled header's rollup count
		"✓ Developer Task — Fix the lexer (· 1 tool call · 1 think ✓ done)",
		// think-ONLY thread: the fallback peek + its zero-tool rollup
		"  ↳ thinking · 1 lines",
		"✓ Developer Task — Muse only (· 0 tool calls · 1 think ✓ done)",
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("sneak must pin the last tool line (shape %q):\n%s", want, convo)
		}
	}
	// …and a thought NEVER leads the peek of a tool-bearing thread
	if strings.Contains(convo, "↳ a thought") || strings.Contains(convo, "↳ thinking · 2 lines") {
		t.Fatalf("a thought must never lead the sneak of a tool-bearing thread:\n%s", convo)
	}
}

// TestShapeToolTextIdempotent is proof (j): the display-side shaping
// turns the reducer's "<lowercase verb> · <rest>" tool texts into
// opencode's "<Verb> <rest>" — and ONLY those: anything already in the
// target shape (the aspirational texts this package's fixtures carry),
// a tagged line, or a head outside the verb regex rides through
// UNCHANGED. Every output re-shaped is itself: the shaping is
// IDEMPOTENT.
func TestShapeToolTextIdempotent(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"read · internal/panels/chat.go", "Read internal/panels/chat.go"}, // the reducer's own emit
		{"bash · go test", "Bash go test"},
		{"grep -rn foo · internal/", "Grep -rn foo internal/"},
		{"read_file · x", "Read_file x"},
		{"Read internal/panels/chat.go", "Read internal/panels/chat.go"}, // the target form — untouched
		{"List internal/panels", "List internal/panels"},                 // no " · " join
		{"read", "read"},
		{"[tool] read · x", "[tool] read · x"}, // tagged — head busts the verb regex
		{"PR #7 · fix", "PR #7 · fix"},         // capital head — not the reducer's verb
		{"", ""},
	} {
		if got := shapeToolText(tc.in); got != tc.want {
			t.Errorf("shapeToolText(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if got := shapeToolText(tc.want); got != tc.want {
			t.Errorf("shapeToolText(%q) = %q — the shaping must be IDEMPOTENT", tc.want, got)
		}
	}
}

// TestThreadStoppedCheckAndRollup is proof (i): a /stop-stopped thread
// dims-red a "✗" glyph with the "✗ stopped" rollup on its collapsed
// header, FORCE-collapses under the ctrl+g baseline, and re-opens only
// on an explicit per-agent expand — the same stopped wording the old
// summary card carried.
func TestThreadStoppedCheckAndRollup(t *testing.T) {
	c := newOpencodeChat(t, 80, 24)
	c.MarkThreadStopped("skopos-1")
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "✗ Explore Task — Scout question kinds recon (· 2 tool calls ✗ stopped)") {
		t.Fatalf("stopped thread must read ✗ + the ✗ stopped rollup:\n%s", convo)
	}
	tbAssertExpanded(t, c, "skopos-1", false, "stopped, no gesture")
	// the ctrl+g baseline does NOT re-open a stopped thread
	c.ToggleThreads()
	tbAssertExpanded(t, c, "skopos-1", false, "stopped under the ctrl+g baseline")
	if convo := ansi.Strip(c.renderConversation()); strings.Contains(convo, "[tool] List internal/panels") {
		t.Fatalf("a stopped thread must stay folded under ctrl+g:\n%s", convo)
	}
	// an explicit per-agent expand re-opens it — closing line carries
	// the stopped wording too
	c.ToggleThreads() // baseline back off (isolation from the next line)
	c.ToggleThread("skopos-1")
	convo = ansi.Strip(c.renderConversation())
	for _, want := range []string{
		"✗ Explore Task — Scout question kinds recon",
		"  [tool] List internal/panels ✓",
		"  · 2 tool calls ✗ stopped",
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("expanded stopped thread missing shape %q:\n%s", want, convo)
		}
	}
}

// TestThreadFirstGroupTopEdgeHitMap is proof (k): when the thread group
// TOPS the merged timeline (its birth At earlier than every chat entry),
// the loop writes nothing before it and renderConversation's TrimLeft
// eats the block's own "\n\n" lead — the header is content row 0, no
// blank row above it. The hit-map must register base+0 there, not
// base+2 (registration runs BEFORE the block write, so b.Len()==0
// uniquely marks the top edge): a base+2 registration puts every row 2
// LOW — the visual header row answers no click, and the sneak's
// registration lands on the NEXT timeline item's row (click-hijack).
func TestThreadFirstGroupTopEdgeHitMap(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 24)
	c.SetState(state.OfficeState{
		Tick: 50,
		Employees: []state.Employee{
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Fix the lexer's top edge"},
		},
		Chat: []state.ChatMsg{
			// the thread is born BEFORE the user's first word — it
			// tops the merged timeline (mid-timeline slots keep the
			// base+2 registration: their "\n\n" lead survives)
			{ID: "x1", From: "tekton-1", Kind: wtoolKind, Text: "Read internal/panels/chat.go", Meta: "done\x1f5", At: 10},
			{ID: "x2", From: "tekton-1", Kind: wtoolKind, Text: "edit · lex.go", Meta: "done\x1f6", At: 20},
			{ID: "u1", From: "user", Kind: "user", Text: "now the follow-up", At: 30},
		},
	})
	convo := ansi.Strip(c.renderConversation())
	rows := strings.Split(convo, "\n")
	wantHeader := "✓ Developer Task — Fix the lexer's top edge (· 2 tool calls ✓ done)"

	// (a) NO leading blank row: the transcript TOP is the thread header
	// itself — TrimLeft already ate the block's "\n\n" lead
	if rows[0] != wantHeader {
		t.Fatalf("the top-edge group must open the transcript with its header (row 0 = %q):\n%s", rows[0], convo)
	}

	// (b) the hit-map must land on the SAME rows the trimmed content
	// shows: the LOWEST registered key IS the header's visual row
	headerRow := -1
	for i, r := range rows {
		if r == wantHeader {
			headerRow = i
			break
		}
	}
	if headerRow < 0 {
		t.Fatalf("the header row is missing:\n%s", convo)
	}
	var keys []int
	for k := range c.threadRows {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	if len(keys) != 2 {
		t.Fatalf("the collapsed top-edge thread must register header + sneak (2 rows), got %v", keys)
	}
	t.Logf("top-edge hit-map: header visual row %d, registered rows %v (min %d)", headerRow, keys, keys[0])
	if keys[0] != headerRow {
		t.Fatalf("the lowest registered row must BE the header's visual row %d, got %v — the top-edge registration is riding low through the TrimLeft", headerRow, keys)
	}

	// (c) clicking the header's VISUAL row toggles the thread — both
	// ways — and the following user message's row is NOT clickable
	// (the pre-fix sneak registration hijacked exactly that row)
	if !c.ClickRow(1, headerRow) {
		t.Fatalf("click on the top-edge header row %d was not claimed", headerRow)
	}
	tbAssertExpanded(t, c, "tekton-1", true, "after top-edge header click")
	if !c.ClickRow(1, headerRow) { // the header row stays 0 while expanded
		t.Fatalf("second click on the top-edge header row %d was not claimed", headerRow)
	}
	tbAssertExpanded(t, c, "tekton-1", false, "after re-click (collapsed again)")
	userRow := -1
	for i, r := range strings.Split(ansi.Strip(c.renderConversation()), "\n") {
		if strings.Contains(r, "now the follow-up") {
			userRow = i
			break
		}
	}
	if userRow < 0 {
		t.Fatalf("the user message row is missing:\n%s", convo)
	}
	t.Logf("user message visual row %d is unregistered (pre-fix the sneak registration hijacked it)", userRow)
	if c.ClickRow(1, userRow) {
		t.Fatalf("click on the following user message's row %d must fall through", userRow)
	}
	tbAssertExpanded(t, c, "tekton-1", false, "after user-row click (no-op)")
}

// newWdiffChat — the per-call-diff fixture: tekton-1's DONE thread
// (idle sprite, ticks far past the meta marks) carries a read, then an
// edit whose completed patch rode in as "wdiff-tekton-1-tc-2" — the REAL
// reducer's id pairing (tool "wtool-tekton-1-tc-2" ↔ diff tail), so
// threadDiffFor resolves exactly like production.
func newWdiffChat(t *testing.T, w, h int) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(w, h)
	c.SetState(state.OfficeState{
		Tick: 1000,
		Employees: []state.Employee{
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper,
				Sprite: state.SpriteAtDesk, Task: "Patch the lexer"},
		},
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "patch the lexer", At: 10},
			{ID: "wtool-tekton-1-tc-1", From: "tekton-1", Kind: wtoolKind,
				Text: "Read internal/panels/chat.go", Meta: "done\x1f5", At: 20},
			{ID: "wtool-tekton-1-tc-2", From: "tekton-1", Kind: wtoolKind,
				Text: "edit · lex.go", Meta: "done\x1f8", At: 30},
			{ID: "wdiff-tekton-1-tc-2", From: "tekton-1", Kind: wdiffKind,
				Text: "--- a/internal/panels/lex.go\n" +
					"+++ b/internal/panels/lex.go\n" +
					"@@ -10,1 +10,3 @@\n" +
					" return rows\n" +
					"-old\n" +
					"+new one\n" +
					"+new two\n",
				Meta: "internal/panels/lex.go\x1f+2\x1f-1", At: 40},
		},
	})
	return c
}

// TestThreadWdiffCollapsedSneakSuffix is proof (l)-collapsed: the thread
// stays the two-row collapsed shape, the sneak (the NEWEST tool — the
// edit, not its diff) gains the dim "· +2 -1" count suffix, and NO
// "↳ diff" row or body line leaks into the collapsed view. The rollup
// counts the diff as NEITHER tool nor think ("· 2 tool calls").
func TestThreadWdiffCollapsedSneakSuffix(t *testing.T) {
	c := newWdiffChat(t, 84, 24)
	convo := ansi.Strip(c.renderConversation())
	for _, want := range []string{
		"✓ Developer Task — Patch the lexer (· 2 tool calls ✓ done)", // diff NOT a 3rd call
		"  ↳ Edit lex.go · +2 -1",                                    // sneak: newest tool + counts
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("collapsed wdiff thread missing shape %q:\n%s", want, convo)
		}
	}
	if strings.Contains(convo, "↳ diff ·") || strings.Contains(convo, "new one") {
		t.Fatalf("the collapsed thread must not show the ↳ diff row or body:\n%s", convo)
	}
	if strings.Contains(convo, "[tool] ") {
		t.Fatalf("collapsed threads show no tool rows:\n%s", convo)
	}
}

// TestThreadWdiffExpandedSubRow is proof (l)-expanded: the edit's tool
// row carries the "· +2 -1" suffix, a one-row "↳ diff · path +2 -1"
// sub-row sits DIRECTLY beneath it (before the read row? no — in natural
// chat order right after ITS tool), the body stays closed until clicked,
// and the closing rollup still reads "· 2 tool calls".
func TestThreadWdiffExpandedSubRow(t *testing.T) {
	c := newWdiffChat(t, 84, 24)
	c.ToggleThread("tekton-1")
	convo := ansi.Strip(c.renderConversation())
	fmt.Println("---- THREAD + PER-CALL DIFF (84 cols, ansi-stripped) ----")
	for _, ln := range strings.Split(convo, "\n") {
		if strings.Contains(ln, "Developer Task") || strings.Contains(ln, "[tool] ") ||
			strings.Contains(ln, "↳ diff") || strings.Contains(ln, "  · ") {
			fmt.Printf("%2d|%s|\n", len([]rune(ln)), ln)
		}
	}
	fmt.Println("---- END THREAD ----")
	for _, want := range []string{
		"  [tool] Read internal/panels/chat.go ✓", // the OTHER tool stays suffix-free
		"  [tool] Edit lex.go ✓ · +2 -1",          // the edited tool's count suffix
		"  ↳ diff · internal/panels/lex.go +2 -1", // the diff sub-row
		"  ↳ Edit lex.go · +2 -1",                 // sneak: the current-task line, suffixed too
		"  · 2 tool calls ✓ done",                 // diff counted as NEITHER tool nor think
	} {
		if !strings.Contains(convo, want) {
			t.Fatalf("expanded wdiff thread missing row %q:\n%s", want, convo)
		}
	}
	if strings.Contains(convo, "↳ diff ·\n") || strings.Contains(convo, "new one") {
		t.Fatalf("the diff body stays closed until the ↳ row is clicked:\n%s", convo)
	}
	// ORDER: edit tool row < its ↳ diff sub-row < sneak < closing
	iTool := strings.Index(convo, "[tool] Edit lex.go")
	iDiff := strings.Index(convo, "↳ diff · internal/panels/lex.go")
	iSneak := strings.LastIndex(convo, "↳ Edit lex.go")
	iClose := strings.Index(convo, "  · 2 tool calls ✓ done")
	if !(iTool >= 0 && iTool < iDiff && iDiff < iSneak && iSneak < iClose) {
		t.Fatalf("the ↳ diff sub-row must land right beneath its tool row (t=%d d=%d s=%d c=%d):\n%s",
			iTool, iDiff, iSneak, iClose, convo)
	}
}

// TestThreadWdiffClickTogglesBody is proof (m): the ↳ diff sub-row
// registers into toolDiffRows (EXACTLY one entry — body rows never
// register, the thread's own frame rows are untouched); ClickRow on the
// sub-row's visual line opens the parsed body (the flat-diff gutter/+
// rows verbatim), a second click closes it, and the thread itself stays
// expanded throughout.
func TestThreadWdiffClickTogglesBody(t *testing.T) {
	c := newWdiffChat(t, 84, 24)
	c.ToggleThread("tekton-1")
	rowOf := func(needle string) int {
		for i, ln := range strings.Split(ansi.Strip(c.renderConversation()), "\n") {
			if strings.Contains(ln, needle) {
				return i
			}
		}
		return -1
	}
	diffRow := rowOf("↳ diff · internal/panels/lex.go")
	if diffRow < 0 {
		t.Fatalf("the ↳ diff sub-row is missing from the expanded thread")
	}
	// the hit-map carries EXACTLY that row → the wdiff id
	if len(c.toolDiffRows) != 1 || c.toolDiffRows[diffRow] != "wdiff-tekton-1-tc-2" {
		t.Fatalf("toolDiffRows must register the ↳ row %d alone, got %v", diffRow, c.toolDiffRows)
	}
	// click → the parsed body opens (gutter + tinted rows are ANSI-styled;
	// the plain text of the addition rows survives ansi.Strip)
	if !c.ClickRow(chatPadL, diffRow) {
		t.Fatalf("click on the ↳ diff row %d was not claimed", diffRow)
	}
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "new one") || !strings.Contains(convo, "new two") {
		t.Fatalf("after the click the parsed body must render:\n%s", convo)
	}
	if !strings.Contains(convo, "old") || !strings.Contains(convo, "return rows") {
		t.Fatalf("the body keeps its context/deletion rows too:\n%s", convo)
	}
	tbAssertExpanded(t, c, "tekton-1", true, "after ↳-click (thread toggle untouched)")
	// the body rows themselves are NOT clickable — only the ↳ sub-row is
	bodyRow := rowOf("new one")
	if bodyRow >= 0 && c.ClickRow(chatPadL, bodyRow) {
		t.Fatalf("click on a diff BODY row %d must fall through", bodyRow)
	}
	// click again → the body closes, the ↳ sub-row survives
	if !c.ClickRow(chatPadL, diffRow) {
		t.Fatalf("second click on the ↳ diff row %d was not claimed", diffRow)
	}
	convo = ansi.Strip(c.renderConversation())
	if strings.Contains(convo, "new one") || !strings.Contains(convo, "↳ diff · internal/panels/lex.go +2 -1") {
		t.Fatalf("the second click must close the body:\n%s", convo)
	}
	tbAssertExpanded(t, c, "tekton-1", true, "after ↳-close-click")
}

// TestThreadWdiffTrailingNeverStealsSneak mirrors proof (h): a wdiff as
// the thread's LAST line never becomes (or scrambles) the sneak — the
// peek still pins the newest TOOL line, suffixed.
func TestThreadWdiffTrailingNeverStealsSneak(t *testing.T) {
	c := newWdiffChat(t, 84, 24)
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "  ↳ Edit lex.go · +2 -1") {
		t.Fatalf("a trailing wdiff must leave the sneak on the newest tool:\n%s", convo)
	}
	if strings.Contains(convo, "↳ diff internal") || strings.Contains(convo, "  ↳ diff · internal/panels/lex.go\n") {
		t.Fatalf("the ↳ diff row belongs to the EXPANDED thread only:\n%s", convo)
	}
}

// TestThreadOpencodeFrame prints the canonical gallery frame — one LIVE
// thread (the ⠿-opened braille glyph at tick 21 + bare sneak, the
// LOCKED two-line shot), one DONE thread (✓ + rollup + sneak), the hint
// row, and the block-bar typing row — for eyeball review.
func TestThreadOpencodeFrame(t *testing.T) {
	c := newOpencodeChatAtTick(t, 60, 24, 21)
	view := ansi.Strip(c.View())
	fmt.Println("---- OPENCODE THREADS (60 cols: live + done + hint + pending bar, ansi-stripped) ----")
	for _, r := range strings.Split(view, "\n") {
		fmt.Printf("%2d|%s|\n", len([]rune(r)), r)
	}
	fmt.Println("---- END PANEL ----")
	for i, r := range strings.Split(view, "\n") {
		if w := len([]rune(r)); w > 60 {
			t.Fatalf("row %d overflows the 60-col budget (%d cells): %q", i, w, r)
		}
	}
	// the LOCKED collapsed-live frame: the two lines CONTIGUOUS, header
	// rollup-free, sneak bare (the unpadded transcript — View pads rows)
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "⠿ Explore Task — Scout question kinds recon\n  ↳ Read internal/panels/chat.go") {
		t.Fatalf("the locked collapsed-live frame is not on screen:\n%s", convo)
	}
}

// ----------------------------------------------------------------------------- older-history pagination proofs

// walkHistory — the (n)/(o)/(q)/(r)/(s) proofs' canned server: N rows
// oldest→newest ("his-001"…) answering state.SessionPager with the
// serve's exact walk semantics (the demoHistoryRows twin in demo.go):
// before == "" answers the NEWEST page; the cursor is the previous
// answer's OWN oldest row id (the X-Next-Cursor twin); NextCursor +
// HasMore ride while older rows remain, the oldest slice drops both.
// Rows are user-role with 1-line texts so the RENDERED growth is exactly
// 2 rows per spliced message (body + separator) — cell-exact anchors.
type walkHistory struct{ rows []state.SessionMessageRow }

var _ state.SessionPager = (*walkHistory)(nil) // the proofs drive the REAL seam shape

func newWalkHistory(n int) *walkHistory {
	rows := make([]state.SessionMessageRow, 0, n)
	for i := 1; i <= n; i++ {
		rows = append(rows, state.SessionMessageRow{
			ID:      fmt.Sprintf("his-%03d", i),
			Role:    "user",
			Created: int64(1000 + i*10),
			Parts:   []state.SessionMessagePart{{Type: "text", Text: fmt.Sprintf("note %03d", i)}},
		})
	}
	return &walkHistory{rows: rows}
}

func (w *walkHistory) MessagesPage(_ context.Context, _ string, before string, limit int) (state.SessionMessagesPage, error) {
	if limit < 1 {
		limit = 1
	}
	end := len(w.rows)
	if before != "" {
		end = -1
		for i, r := range w.rows {
			if r.ID == before {
				end = i
				break
			}
		}
		if end < 0 {
			return state.SessionMessagesPage{}, fmt.Errorf("unknown before cursor %q", before)
		}
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	page := state.SessionMessagesPage{
		Rows:    append([]state.SessionMessageRow(nil), w.rows[start:end]...),
		HasMore: start > 0,
	}
	if page.HasMore {
		page.NextCursor = w.rows[start].ID
	}
	return page, nil
}

// seedTranscript drops a fetched page's rows into the chat as USER
// entries (the walk's boot page shape: SetState is the reducer's own
// ingress, so the splice tests start from a real rendered transcript).
// The reader is parked up in history: follow stays OFF for the whole
// walk so the bottom snap never fights it.
func seedTranscript(t *testing.T, c *Chat, rows []state.SessionMessageRow) {
	t.Helper()
	chat := make([]state.ChatMsg, 0, len(rows))
	for _, r := range rows {
		chat = append(chat, state.ChatMsg{ID: r.ID, From: "user", Kind: "user", Text: r.Parts[0].Text, At: r.Created})
	}
	c.follow = false // BEFORE SetState: the parked reader never snaps to the tail
	c.SetState(state.OfficeState{Tick: 1, Chat: chat})
}

// TestPrependOlderOrderDedupePurity is proof (n): the splice lands the
// fetched page ahead of the head IN PAGE ORDER (oldest→newest), reports
// fresh rows only, ignores ids already present AND duplicate ids inside
// one page, and is PURE — no offset move, no follow flip, no pager
// touch (those are PreserveAnchor's/ThreadPager's lanes).
func TestPrependOlderOrderDedupePurity(t *testing.T) {
	ctx := context.Background()
	src := newWalkHistory(500)
	boot, err := src.MessagesPage(ctx, "ses-1", "", ThreadOlderPageSize)
	if err != nil || len(boot.Rows) != ThreadOlderPageSize {
		t.Fatalf("boot page: %v (%d rows)", err, len(boot.Rows))
	}
	c := NewChat(nil)
	c.SetSize(60, 24)
	seedTranscript(t, c, boot.Rows) // his-451..his-500 on screen
	pager := NewThreadPager("ses-1")
	pager.Seed(boot.NextCursor, boot.HasMore) // untouched below: purity of the splice
	c.vp.SetYOffset(4)

	page, err := src.MessagesPage(ctx, "ses-1", boot.NextCursor, ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("older page: %v", err)
	}
	if added := c.PrependOlder(page.Rows); added != ThreadOlderPageSize {
		t.Fatalf("a full page must splice %d fresh rows, got %d", ThreadOlderPageSize, added)
	}
	// ORDER: the page lands ahead of the previous head, oldest→newest
	for idx, want := range map[int]string{0: "his-401", 49: "his-450", 50: "his-451"} {
		if c.chat[idx].ID != want {
			t.Fatalf("head splice order: chat[%d] must be %q, got %q", idx, want, c.chat[idx].ID)
		}
	}
	// PURITY: offset, follow latch and pager state are someone else's lane
	if off := c.vp.YOffset(); off != 4 {
		t.Fatalf("PrependOlder must never move the scroll offset, got %d", off)
	}
	if c.follow {
		t.Fatal("PrependOlder must never flip the follow latch")
	}
	if pager.inFlight || pager.top || pager.cursor != boot.NextCursor || !pager.seeded {
		t.Fatalf("PrependOlder must never consult the pager, got %+v", pager)
	}
	// NO-DUPES, same page twice: 0 fresh, transcript untouched
	if again := c.PrependOlder(page.Rows); again != 0 || len(c.chat) != 100 {
		t.Fatalf("a re-spliced page must add 0 rows and keep 100 entries, got %d added / %d entries", again, len(c.chat))
	}
	// the resumed-boot OVERLAP: his-396..his-405 — 5 fresh older rows
	// ahead of the head, 5 already in the transcript
	overlap := src.rows[395:405]
	if added := c.PrependOlder(overlap); added != 5 {
		t.Fatalf("the overlap page must add ONLY its 5 fresh rows, got %d", added)
	}
	one := 0
	for _, m := range c.chat {
		if m.ID == "his-401" {
			one++
		}
	}
	if one != 1 {
		t.Fatalf("his-401 must appear EXACTLY once after the overlap splice, got %d", one)
	}
	for i, want := range []string{"his-396", "his-397", "his-398", "his-399", "his-400", "his-401"} {
		if c.chat[i].ID != want {
			t.Fatalf("head row %d must be %q after the overlap splice, got %q", i, want, c.chat[i].ID)
		}
	}
	// a page never duplicates ITSELF: one row id repeated in a fresh page
	selfDupe := append(append([]state.SessionMessageRow{}, src.rows[189:194]...), src.rows[189])
	if added := c.PrependOlder(selfDupe); added != 5 {
		t.Fatalf("a self-duplicated page must splice each row ONCE (5), got %d", added)
	}
	if len(c.chat) != 110 {
		t.Fatalf("the four splices must total 110 transcript entries, got %d", len(c.chat))
	}
}

// TestThreadPagerOlderHistoryHasMoreWalk is proof (o): the walk drains a
// 500-row history in EXACTLY 10 pages of ThreadOlderPageSize (the boot
// page + 9 older hops), each page opening 50 rows below the previous's,
// and the oldest page closing the walk (hasMore=false / cursor "") with
// the top latch refusing anything further.
func TestThreadPagerOlderHistoryHasMoreWalk(t *testing.T) {
	ctx := context.Background()
	src := newWalkHistory(500)
	pager := NewThreadPager("ses-1")

	boot, err := src.MessagesPage(ctx, "ses-1", "", ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("boot page: %v", err)
	}
	// the boot page is the NEWEST slice, and 450 older rows keep the
	// walk open on the page's OWN oldest id
	if got := boot.Rows[0].ID; got != "his-451" {
		t.Fatalf("the newest page must open at his-451, got %q", got)
	}
	if !boot.HasMore || boot.NextCursor != "his-451" {
		t.Fatalf("older rows must keep the walk open, got hasMore=%v cursor=%q", boot.HasMore, boot.NextCursor)
	}
	pager.Seed(boot.NextCursor, boot.HasMore)

	var heads []string // each fetched page's oldest row id, in fetch order
	var all []string   // every row id ever fetched
	pages := 1         // the boot page itself
	addPage := func(p state.SessionMessagesPage) {
		if len(p.Rows) != ThreadOlderPageSize {
			t.Fatalf("every page must carry %d rows, got %d", ThreadOlderPageSize, len(p.Rows))
		}
		heads = append(heads, p.Rows[0].ID)
		for _, r := range p.Rows {
			all = append(all, r.ID)
		}
	}
	addPage(boot)
	var last state.SessionMessagesPage
	for {
		cursor, ok := pager.StartOlder()
		if !ok {
			break
		}
		page, err := src.MessagesPage(ctx, "ses-1", cursor, ThreadOlderPageSize)
		if err != nil {
			pager.FailOlder()
			continue
		}
		pager.FinishOlder(page.NextCursor, page.HasMore)
		last = page
		addPage(page)
		pages++
	}
	if pages != 10 {
		t.Fatalf("500 rows at %d/page must drain in 10 pages, got %d", ThreadOlderPageSize, pages)
	}
	wantHeads := []string{"his-451", "his-401", "his-351", "his-301", "his-251", "his-201", "his-151", "his-101", "his-051", "his-001"}
	for i, want := range wantHeads {
		if heads[i] != want {
			t.Fatalf("page %d must open at %q, got %q", i, want, heads[i])
		}
	}
	if last.HasMore || last.NextCursor != "" {
		t.Fatalf("the oldest page must close the walk (hasMore=false, cursor \"\"), got %v/%q", last.HasMore, last.NextCursor)
	}
	if !pager.top {
		t.Fatal("FinishOlder(hasMore=false) must latch the top PERMANENTLY")
	}
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("a topped walk must refuse further hops")
	}
	uniq := map[string]bool{}
	for i, id := range all {
		if uniq[id] {
			t.Fatalf("row %d: id %q fetched twice — a walk never duplicates", i, id)
		}
		uniq[id] = true
	}
	if len(uniq) != 500 {
		t.Fatalf("the walk must fetch all 500 rows exactly once, got %d", len(uniq))
	}
}

// TestThreadPagerInFlightAndFailureGuards is proof (p): the controller's
// guard contract — unseeded refuses, Seed is idempotent (a re-seed never
// moves the anchor), one hop in flight at a time, three straight
// failures back the walk off WITHOUT moving the cursor, ResetFailures
// re-arms, a success clears the tally AND advances, and the top latch
// outranks even a reset.
func TestThreadPagerInFlightAndFailureGuards(t *testing.T) {
	// UNSEEDED: nothing fetched yet → nothing to walk
	pager := NewThreadPager("ses-1")
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("an unseeded pager must refuse older hops")
	}
	// Seed is idempotent — a re-seed NEVER moves the walk backwards
	pager.Seed("cur-450", true)
	pager.Seed("cur-999", false)
	if pager.cursor != "cur-450" || pager.top {
		t.Fatalf("a re-seed must be a no-op, got cursor=%q top=%v", pager.cursor, pager.top)
	}
	// a boot page that ends the history (hasMore=false) arms nothing
	dry := NewThreadPager("ses-2")
	dry.Seed("", false)
	if _, ok := dry.StartOlder(); ok {
		t.Fatal("a boot-closed walk must never arm a hop")
	}
	// SINGLE-FLIGHT: one hop at a time…
	cursor, ok := pager.StartOlder()
	if !ok || cursor != "cur-450" {
		t.Fatalf("the first seeded hop must ride the boot cursor, got %q/%v", cursor, ok)
	}
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("a second hop must refuse while the first is in flight")
	}
	// …and a failure re-opens the guard on the SAME cursor (no rescan)
	pager.FailOlder()
	if cursor, ok = pager.StartOlder(); !ok || cursor != "cur-450" {
		t.Fatalf("a failed hop retries the SAME cursor, got %q/%v", cursor, ok)
	}
	// BACKOFF: strikes 2 and 3 still re-open the guard once each…
	pager.FailOlder()
	if _, ok := pager.StartOlder(); !ok {
		t.Fatal("the 2nd failure must not back the walk off yet")
	}
	pager.FailOlder()
	// …but strike 3 latches the backoff
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("3 straight failures must back the walk off")
	}
	// ResetFailures re-arms — the cursor STAYS (history doesn't move)
	pager.ResetFailures()
	if cursor, ok = pager.StartOlder(); !ok || cursor != "cur-450" {
		t.Fatalf("a re-armed walk retries where it stopped, got %q/%v", cursor, ok)
	}
	// a success clears the tally AND advances the cursor
	pager.FinishOlder("cur-400", true)
	if pager.failures != 0 || pager.cursor != "cur-400" || pager.top {
		t.Fatalf("FinishOlder must reset strikes and advance, got failures=%d cursor=%q top=%v",
			pager.failures, pager.cursor, pager.top)
	}
	if _, ok := pager.StartOlder(); !ok {
		t.Fatal("a fresh success must re-arm the next hop immediately")
	}
	// the top latch is permanent — failures, a reset, nothing beats it
	pager.FailOlder()
	pager.ResetFailures()
	if _, ok := pager.StartOlder(); !ok {
		t.Fatal("the reset walk must hop once more")
	}
	pager.FinishOlder("", false)
	if !pager.top {
		t.Fatal("hasMore=false must latch the top")
	}
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("the top latch outranks everything")
	}
	pager.ResetFailures()
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("even a re-armed walk never moves past the top")
	}
}

// TestAtTranscriptTopProbe is proof (q): the gesture probe is EXACTLY
// "the viewport's offset is the transcript's first row" — true at the
// head, false one row down, and false again after a landed page +
// anchor bump parks the reader mid-transcript. Arming an actual hop is
// the pager's contract, never this probe's.
func TestAtTranscriptTopProbe(t *testing.T) {
	ctx := context.Background()
	src := newWalkHistory(500)
	boot, err := src.MessagesPage(ctx, "ses-1", "", ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("boot page: %v", err)
	}
	c := NewChat(nil)
	c.SetSize(60, 24)
	seedTranscript(t, c, boot.Rows)
	if !c.AtTranscriptTop() {
		t.Fatal("a fresh top-parked transcript must probe at-top (offset 0)")
	}
	c.vp.SetYOffset(1)
	if c.AtTranscriptTop() {
		t.Fatal("one row below the head is not the top")
	}
	c.vp.SetYOffset(4)
	page, err := src.MessagesPage(ctx, "ses-1", boot.NextCursor, ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("older page: %v", err)
	}
	before := c.TranscriptRows()
	c.PrependOlder(page.Rows)
	c.PreserveAnchor(before)
	if c.AtTranscriptTop() {
		t.Fatal("an anchor-compensated reader is mid-transcript, not at the top")
	}
}

// TestPreserveAnchorCompensatesPrepend is proof (r): across a 50-message
// head splice the reader's top row keeps its screen cell BYTE-IDENTICAL
// — TranscriptRows grows by exactly 2 rows per spliced message (body +
// separator), PreserveAnchor(before) bumps the offset by that growth,
// the window materializes the compensated rows at paint time, and a
// stale zero-growth snapshot moves nothing.
func TestPreserveAnchorCompensatesPrepend(t *testing.T) {
	ctx := context.Background()
	src := newWalkHistory(500)
	boot, err := src.MessagesPage(ctx, "ses-1", "", ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("boot page: %v", err)
	}
	c := NewChat(nil)
	c.SetSize(60, 24)
	seedTranscript(t, c, boot.Rows) // 50 one-row user bubbles → 99 rendered rows
	if rows := c.TranscriptRows(); rows != 99 {
		t.Fatalf("50 messages + separators must render 2·50-1=99 rows, got %d", rows)
	}
	c.vp.SetYOffset(4)
	topRow := c.selLines[4]
	before := c.TranscriptRows()

	page, err := src.MessagesPage(ctx, "ses-1", boot.NextCursor, ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("older page: %v", err)
	}
	added := c.PrependOlder(page.Rows)
	if added != ThreadOlderPageSize {
		t.Fatalf("the page must add %d messages, got %d", ThreadOlderPageSize, added)
	}
	if rows := c.TranscriptRows(); rows != before+2*added {
		t.Fatalf("the splice must grow exactly %d rendered rows, got %d→%d", 2*added, before, rows)
	}
	c.PreserveAnchor(before)
	if off := c.vp.YOffset(); off != 4+2*added {
		t.Fatalf("the anchor bump must land at %d, got %d", 4+2*added, off)
	}
	// the reading row keeps its cell — and the window materializes the
	// compensated rows at paint time (the View seam catches the bump)
	_ = c.View()
	if got := c.selLines[c.vp.YOffset()]; got != topRow {
		t.Fatalf("the compensated top row must be byte-identical:\n got  %q\nwant %q", got, topRow)
	}
	if y := c.vp.YOffset(); y < c.win.lo || y+c.vp.Height() > c.win.hi {
		t.Fatalf("the compensated viewport must be materialized: off %d, window [%d,%d)", y, c.win.lo, c.win.hi)
	}
	// the stale-snapshot guard: zero growth moves nothing (no double-bump)
	c.PreserveAnchor(c.TranscriptRows())
	if off := c.vp.YOffset(); off != 4+2*added {
		t.Fatalf("a zero-growth snapshot must move nothing, got %d", off)
	}
}

// TestOlderHistory500MessageWalk is proof (s), the 500-message worked
// example END TO END: the boot page seeds 50 rows, nine pager-guarded
// hops walk the rest, every hop prepends exactly 50 messages / +100
// rendered rows, PreserveAnchor re-parks the same reading row after
// every compensation, and the walk closes at the top with all 500
// entries in order and 999 rendered rows on screen.
func TestOlderHistory500MessageWalk(t *testing.T) {
	ctx := context.Background()
	src := newWalkHistory(500)
	pager := NewThreadPager("ses-1")
	c := NewChat(nil)
	c.SetSize(60, 24)

	boot, err := src.MessagesPage(ctx, "ses-1", "", ThreadOlderPageSize)
	if err != nil {
		t.Fatalf("boot page: %v", err)
	}
	pager.Seed(boot.NextCursor, boot.HasMore)
	if added := c.PrependOlder(boot.Rows); added != ThreadOlderPageSize {
		t.Fatalf("the boot page must seed %d transcript entries, got %d", ThreadOlderPageSize, added)
	}
	c.vp.SetYOffset(10)
	anchorRow := c.selLines[10] // the reading row watched across every splice
	if !strings.Contains(anchorRow, "note 456") {
		t.Fatalf("sanity: selLines[10] must be the his-456 bubble row, got %q", anchorRow)
	}

	t.Log("hop | messages | rendered rows | Δrows | yOffset after PreserveAnchor")
	hops := 0
	for {
		before := c.TranscriptRows()
		cursor, ok := pager.StartOlder()
		if !ok {
			break
		}
		page, err := src.MessagesPage(ctx, "ses-1", cursor, ThreadOlderPageSize)
		if err != nil {
			pager.FailOlder()
			continue
		}
		added := c.PrependOlder(page.Rows)
		pager.FinishOlder(page.NextCursor, page.HasMore)
		c.PreserveAnchor(before)
		hops++
		growth := c.TranscriptRows() - before
		t.Logf("  %d | %3d (+ %2d) | %5d | +%3d | %d", hops, len(c.chat), added, c.TranscriptRows(), growth, c.vp.YOffset())
		if added != ThreadOlderPageSize {
			t.Fatalf("hop %d: a mid-walk page must add %d messages, got %d", hops, ThreadOlderPageSize, added)
		}
		if growth != 100 {
			t.Fatalf("hop %d: the rendered growth must be exactly 100 rows, got %d", hops, growth)
		}
		if page.HasMore && page.NextCursor == "" {
			t.Fatalf("hop %d: an open walk must carry its cursor", hops)
		}
	}
	if hops != 9 {
		t.Fatalf("450 older rows at %d/hop must drain in 9 hops, got %d", ThreadOlderPageSize, hops)
	}
	if !pager.top {
		t.Fatal("the 10th page (his-001..his-050) closes the walk — the top must latch")
	}
	// the full history: 500 entries, in order, 999 rendered rows
	if len(c.chat) != 500 {
		t.Fatalf("the full history must be 500 transcript entries, got %d", len(c.chat))
	}
	for i, m := range c.chat {
		if want := fmt.Sprintf("his-%03d", i+1); m.ID != want {
			t.Fatalf("transcript entry %d must be %q, got %q", i, want, m.ID)
		}
	}
	if rows := c.TranscriptRows(); rows != 999 {
		t.Fatalf("500 one-row bubbles + separators must render 2·500-1=999 rows, got %d", rows)
	}
	if off := c.vp.YOffset(); off != 10+9*100 {
		t.Fatalf("9 × 100-row compensation must land the reader at %d, got %d", 910, off)
	}
	_ = c.View()
	if got := c.selLines[c.vp.YOffset()]; got != anchorRow {
		t.Fatalf("after 9 splices the anchor row must still be byte-identical:\n got  %q\nwant %q", got, anchorRow)
	}
	if _, ok := pager.StartOlder(); ok {
		t.Fatal("a topped walk refuses further hops")
	}
}
