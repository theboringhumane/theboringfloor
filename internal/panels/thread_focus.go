// thread_focus.go — the THREAD FOCUS VIEW: ctrl+f opens ONE worker
// thread (Developer/Subagent Task) as a fullscreen nested panel showing
// its COMPLETE transcript — every tool call, every think body (FULL
// expand), every per-call wdiff with its ↳ sub-row and clickable parsed
// body — rendered by the IDENTICAL row builders the main chat uses,
// running at the frame width on a PRIVATE *Chat clone.
//
// The pane owns exactly two rows of layout: row 0 is its own header
// (done/live glyph + threadTitle + "· N tool calls[ · M thinks]"), the
// rest is the clone's transcript viewport. Scrolling (pgup/pgdn/up/down/
// wheel) forwards to the clone's Update; clicks claim ONLY the "↳ diff"
// sub-rows (the clone's own toolDiffRows hit-map + ToggleThreadDiff —
// thread/fold/floor rows stay inert in v1). esc/ctrl+f never reach here:
// the app consumes them as the leave pair (precedence: ctrl+q outranks
// focus, /zen outranks focus, a permission/question float dismounts it).
//
// While the focus is open the main chat's SetState runs in deferRender
// (chat.go): this pane renders from the SAME office state, so rebuilding
// the hidden main transcript per tick would be wasted work — one
// ResumeFromFocus re-render catches the main chat up at close.
package panels

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// focusEmptyText — the /tools-off empty state: the agent is hired but
// wrote NO wtool/wthink/wdiff lines (a tools-hidden adapter records
// nothing). Frozen copy — pinned by thread_focus_test.go.
const focusEmptyText = "no recorded tool calls for %s (tools hidden?)"

// ThreadFocus — the fullscreen thread panel. The clone is a REAL Chat
// with no send wiring (NewChat(nil)): identical row builders, the
// expansion latched per-agent (threadsExpanded + threadExpand[name]=true)
// so the transcript always opens FULLY expanded, and the same
// threadDiffFor/toolDiffRows wdiff plumbing — but fed ONLY the focused
// agent's lines (SetState's filter), so its whole timeline is the
// thread's own group(s).
type ThreadFocus struct {
	name   string
	clone  *Chat
	w, h   int
	empty  bool // no recorded lines for this agent — the /tools-off row
	live   bool // header glyph: roster-active + inside the staleness horizon
	tools  int  // header counters (wdiff counts as NEITHER — threadSummary's rule)
	thinks int
}

// NewThreadFocus builds the panel: the clone sized at the FULL frame
// width (identical wrap budgets), the expansion latched BEFORE the first
// SetState so the transcript renders FULLY expanded from row one.
func NewThreadFocus(name string, w, h int) *ThreadFocus {
	clone := NewChat(nil)
	tf := &ThreadFocus{name: name, clone: clone}
	tf.SetSize(w, h)
	clone.threadsExpanded = true
	clone.threadExpand = map[string]bool{name: true}
	return tf
}

// SetSize sizes the panel: the clone takes the frame width; its viewport
// gets every row but the header's.
func (tf *ThreadFocus) SetSize(w, h int) {
	if w < 4 {
		w = 4
	}
	if h < 2 {
		h = 2
	}
	tf.w, tf.h = w, h
	tf.clone.SetSize(w, h)
	tf.clone.vp.SetHeight(h - 1) // row 0 is the pane's own header
	tf.clone.forceRender()
}

// SetState — the live pulse: re-feeds the clone the focused agent's slice
// of the LATEST office state (the app fires one per event, ticks
// included, while the focus is open — the filter is tens of lines,
// wtool/wthink/wdiff only, and the clone's own revision gate still skips
// no-change renders). Also recomputes the header's counters and the
// done/live glyph (threadLive's exact rule, roster + staleness horizon).
func (tf *ThreadFocus) SetState(st state.OfficeState) {
	filtered := make([]state.ChatMsg, 0, len(st.Chat))
	lastTick := 0
	tools, thinks := 0, 0
	for _, m := range st.Chat {
		if m.From != tf.name {
			continue
		}
		switch m.Kind {
		case wtoolKind:
			tools++
		case wthinkKind:
			thinks++
		case wdiffKind:
			// a per-call diff rides its tool call — it counts as NEITHER
			// a call NOR a think (the threadSummary rollup rule, verbatim)
		default:
			continue
		}
		filtered = append(filtered, m)
		if _, tk := parseWtoolMeta(m.Meta); tk > lastTick {
			lastTick = tk
		}
	}
	tf.tools, tf.thinks = tools, thinks
	tf.empty = len(filtered) == 0
	tf.live = false
	for _, e := range st.Employees {
		if e.Name == tf.name && e.Role != state.RoleManager && workerSpriteActive(e.Sprite) &&
			len(filtered) > 0 && st.Tick-lastTick <= wtoolStaleTicks {
			tf.live = true
			break
		}
	}
	sst := st
	sst.Chat = filtered
	tf.clone.SetState(sst)
}

// Update — the app's focus-key forward: every key but the leave pair
// (esc/ctrl+f, already consumed app-side) lands here. Scroll keys ride
// the clone's own vp (pgup/pgdn/up/down/wheel). The clone's "back one"
// thread ledger is EMPTY (the expansion latch is SET, never walked via
// ExpandThread), so ↑ can never fold the focused thread — it scrolls.
func (tf *ThreadFocus) Update(msg tea.Msg) tea.Cmd {
	return tf.clone.Update(msg)
}

// Click — a LEFT press inside the pane's region, in PANE coords (row 0 is
// the header). v1 claim set: ONLY the "↳ diff · path" sub-rows toggle
// (their parsed bodies open/close through the clone's own toolDiffRows
// hit-map + ToggleThreadDiff — the main chat's exact machinery); the
// header, the thread's frame rows, user-fold rows and everything else
// stay inert. Returns true when the click was claimed.
func (tf *ThreadFocus) Click(x, y int) bool {
	c := tf.clone
	vy := y - 1 // the header owns row 0 outright
	if vy < 0 || vy >= c.vp.Height() {
		return false
	}
	line := vy + c.vp.YOffset()
	if id, ok := c.toolDiffRows[line]; ok {
		c.ToggleThreadDiff(id)
		return true
	}
	return false
}

// scrollOffset — the clone viewport's Y offset (the panel test proves
// scrolling moves the pane's own view, not the office's).
func (tf *ThreadFocus) scrollOffset() int { return tf.clone.vp.YOffset() }

// headerRow — row 0: the done/live glyph + the thread's title + the
// running counters, ONE clipped row (the thread header's single-row
// contract). Unlike the main chat's collapsed/LIVE heads, the focus
// header ALWAYS carries the counters — the focus IS the accounting view.
func (tf *ThreadFocus) headerRow() string {
	body := tf.clone.threadTitle(tf.name)
	unit := "tool calls"
	if tf.tools == 1 {
		unit = "tool call"
	}
	body += " · " + itoa(tf.tools) + " " + unit
	if tf.thinks > 0 {
		body += " · " + itoa(tf.thinks) + " think"
	}
	body = clipPlain(body, tf.w-2) // the 2-cell glyph field eats the head
	if tf.live {
		return threadLiveGlyph(tf.clone.tick) + " " + body
	}
	return chrome.PanelDim.Render("✓") + " " + chrome.PanelDim.Render(body)
}

// View renders: the header on row 0, then the clone's transcript viewport
// (or the one-line /tools-off empty state) below. The app's Frame wraps
// the render in the same lipgloss Width×Height clamp the zen mid gets, so
// the panel never overflows its middle region.
func (tf *ThreadFocus) View() string {
	rows := make([]string, 0, tf.h)
	rows = append(rows, tf.headerRow())
	if tf.empty {
		rows = append(rows,
			chrome.PanelDim.Render("  "+fmt.Sprintf(focusEmptyText, tf.name)))
	} else {
		rows = append(rows, tf.clone.vp.View())
	}
	return strings.Join(rows, "\n")
}

// ResolveFocusThread — the ctrl+f OPEN resolution (the app calls it; the
// resolver lives here because the data does: the expand ledger, the
// tick/liveness math and the roster are all panel-side). Order:
//
//  1. the most recently INTERACTED thread (the expand ledger's tail) —
//     the thread the member last opened by hand wins outright;
//  2. any LIVE thread (threadLive's exact rule — roster sprite busy +
//     the freshest tagged line inside the staleness horizon), newest
//     activity first;
//  3. the most recent thread in the timeline;
//  4. the newest HIRED non-manager with no transcript lines at all — the
//     /tools-off agent stays focusable so its empty state names itself;
//  5. ("", false) — nothing to focus; the app prints "no worker threads
//     yet" and never claims the key.
func ResolveFocusThread(c *Chat, st state.OfficeState) (string, bool) {
	if c == nil {
		return "", false
	}
	// 1) the ledger tail — a hand-expanded thread outranks everything
	if n := len(c.threadExpandOrder); n > 0 {
		return c.threadExpandOrder[n-1], true
	}
	// thread-bearing names in first-appearance order, each with its
	// freshest tagged tick
	var order []string
	fresh := map[string]int{}
	for _, m := range c.chat {
		if m.Kind != wtoolKind && m.Kind != wthinkKind && m.Kind != wdiffKind {
			continue
		}
		if _, seen := fresh[m.From]; !seen {
			order = append(order, m.From)
		}
		if _, tk := parseWtoolMeta(m.Meta); tk > fresh[m.From] {
			fresh[m.From] = tk
		}
	}
	// 2) any live thread — newest activity wins
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		if av, ok := c.agents[name]; ok && av.active && c.tick-fresh[name] <= wtoolStaleTicks {
			return name, true
		}
	}
	// 3) the most recent thread in the timeline
	if len(order) > 0 {
		return order[len(order)-1], true
	}
	// 4) a hired WORKER with NO recorded lines (the /tools-off empty
	//    state stays reachable). Workers only: the always-seated boss
	//    (RoleManager) and hr (RoleHR) are office furniture, never tool
	//    threads — skipping them keeps "no worker threads yet" answerable
	//    on a fresh office.
	for i := len(st.Employees) - 1; i >= 0; i-- {
		switch st.Employees[i].Role {
		case state.RoleDeveloper, state.RoleScout, state.RoleReviewer, state.RoleRunner:
			return st.Employees[i].Name, true
		}
	}
	return "", false
}
