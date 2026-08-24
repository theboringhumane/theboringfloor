// threads_opencode.go — the opencode-style subagent thread renderer.
//
// A subagent's merged wtool/wthink entries (workerGroup) draw as flat
// lines — no bordered card. COLLAPSED (the default, live threads
// included):
//
//	⠿ Explore Task — Scout question kinds recon
//	  ↳ Read internal/panels/chat.go
//
// The header owns a 2-cell glyph field: an ANIMATED braille run
// (threadLiveFrames ⠿⠷⠯⠟⠻⠽⠾ in the house accent — cycled by the
// OFFICE TICK, c.tick%len, the same timer-free drive thinkFrames and
// pendingBlockBar already use) while the thread is LIVE — roster sprite
// busy AND the freshest wtool/wthink meta-tick inside
// c.tick-wtoolStaleTicks, the exact rule the "team is working" loading
// row mirrors — a dim "✓" once DONE (no animation), a dim-red "✗" after
// /stop. The title reads "<Kind> Task — <task>" — Kind from the roster
// role the SetState rollup already captures (a scout runs recon:
// opencode's "Explore"), the task from the sticky workerTasks/agentView
// title — with "Subagent Task — <agent name>" as the no-task fallback.
// A LIVE header carries the glyph + title and NOTHING else — the rollup
// shows only once the thread SETTLES: the collapsed done/stopped header
// KEEPS the old summary card's trailing rollup, dim: "(· N tool calls
// [· M think] ✓ done)" — or "✗ stopped" (dim-red) for a /stop unwind —
// the exact counts the bordered card's "▾ …" line carried. The header
// is always exactly ONE row: a long title elides at the panel width
// with clipPlain, it never wraps to a hanging continuation.
//
// The row below is a dim "  ↳ <Verb> <rest>" sneak peek at the thread's
// NEWEST TOOL entry — BARE: no " ✓" / " ✗" / " … running" state mark
// (the state lives in the header's glyph and the settled thread's
// rollup). The reducer writes tool text as "<lowercase verb> · <rest>"
// ("read · internal/panels/chat.go"); a display-side shaping
// (shapeToolText) renders it as opencode's "<Verb> <rest>" form
// ("Read internal/panels/chat.go") — already-shaped text passes through
// unchanged, so the shaping is IDEMPOTENT. A trailing wthink never
// steals the peek (thoughts stay rolled up in the summary counts), but
// a think-ONLY thread falls back to "thinking · N lines". The sneak is
// always exactly ONE row, elided with clipPlain like the header, and
// shows for EVERY thread, live or done, collapsed or EXPANDED (there it
// trails the tool list as the "current task" line).
//
// A per-agent click on the header row toggles the thread (the threadRows
// hit-map is STATE-CONDITIONAL: collapsed registers header + sneak,
// expanded registers header + closing summary — the whole bubble's frame
// rows — while expanded tool rows and the mid-list sneak pass through);
// the ctrl+g baseline expands the thread to list its merged rows —
// "[tool] <Verb> <rest> <state mark>" in ToolStyle (the same
// display-side shaping, the ✓/✗/✗ aborted/… running mark kept), think
// rows dim-italic with bodies only on a FULL expand — all 2-cell
// indented under the header and STILL wrapping with a 4-cell hanging
// continuation — followed by the ↳ sneak and a dim closing summary line
// ("  · N tool calls[ · M think] ✓ done", the same rollup wording as the
// settled collapsed header).
// While at least one RENDERED thread is live a dim-italic "ctrl+g ·
// view subagents" hint row trails the last thread block of the
// timeline.
//
// Per-call diffs (Kind wdiff — a completed employee edit/write whose
// patch rode the wire's tool-part metadata, app-tagged
// "wdiff-<agent>-<callid>") pin INSIDE the thread: the matching tool row
// and the collapsed sneak gain a dim "· +A -D" count suffix, and the
// EXPANDED thread renders a one-row "  ↳ diff · path +A -D" sub-row
// DIRECTLY beneath that tool row (counts in the flat-diff inks). The
// sub-row owns the NEW toolDiffRows click hit-map — a click there
// opens/closes the parsed body (diffRows/renderDiffRow/diffClip
// verbatim) without touching the thread's own toggle; body rows never
// register. Rollups and the sneak's last-tool scan count wdiff entries
// as NEITHER tools nor thinks, and the flat-diff world keeps ctrl+d to
// itself (thread diff bodies are click-only).
//
// This file also owns the pending row's breathing block bar
// (pendingBlockBar) — the opencode "Build · …" loading line's vibe for
// "<boss> is typing…": height blocks ▁…█ sweeping left-to-right off the
// office tick, deterministic per tick, one row exactly.
package panels

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// threadHintText — the dim-italic hint row trailing the last thread block
// while ≥1 rendered thread is live (ctrl+g is the app keymap's thread
// toggle).
const threadHintText = "ctrl+g · view subagents"

// threadSpinnerStyle — the thread glyph's ink: the house accent
// (chrome.Accent — opencode's spinner rides the accent hue). Re-pointed
// on /theme switches via RefreshTheme — read through a constructor so
// the SGR pair lands again.
func threadSpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(chrome.Accent)
}

// threadLiveFrames — the braille run the LIVE header glyph walks:
// opencode's chunky full-dot cycle, OPENED by ⠿ (the locked gallery
// frame's glyph). Cycled by the office tick (c.tick%len — the same
// timer-free drive thinkFrames and pendingBlockBar use), so SetState's
// existing tick pump is the only animator and a test pins the exact
// frame arithmetically (tick 21 → frame 0 → "⠿").
var threadLiveFrames = []string{"⠿", "⠷", "⠯", "⠟", "⠻", "⠽", "⠾"}

// threadLiveGlyph — the accent-styled braille frame the LIVE header
// shows at office tick `tick`. Pure function of state: no goroutines,
// no timers, deterministic per tick.
func threadLiveGlyph(tick int) string {
	return threadSpinnerStyle().Render(threadLiveFrames[tick%len(threadLiveFrames)])
}

// threadKindLabel — the title's role-ish Kind label, derived from the
// roster seat the SetState rollup already captures (no new state, no
// invented API): a scout runs recon — opencode's read-only "Explore"
// agent — a developer develops; the rest read as their seat. "" marks an
// unknown seat → the caller falls back to "Subagent Task".
func threadKindLabel(role state.EmployeeRole) string {
	switch role {
	case state.RoleScout:
		return "Explore"
	case state.RoleDeveloper:
		return "Developer"
	case state.RoleReviewer:
		return "Reviewer"
	case state.RoleRunner:
		return "Runner"
	case state.RoleHR:
		return "HR"
	}
	return ""
}

// threadLive — the render half of the staleness rule the "team is
// working" row mirrors (chat_loading.go's anyThreadActive): roster sprite
// busy + the group's freshest wtool/wthink meta-tick inside the
// wtoolStaleTicks horizon. Pure read over c.agents + c.tick.
func (c *Chat) threadLive(g workerGroup) bool {
	av, ok := c.agents[g.name]
	return ok && av.active && c.tick-g.lastTick <= wtoolStaleTicks
}

// threadTitle — the header's text: "<Kind> Task — <task>". The task is
// the STICKY last-known dispatch title (workerTasks survives EvReturned's
// task clear, agentView overrides while the roster still carries it);
// with no task at all the agent's NAME is the title
// ("Subagent Task — <agent name>"), and with no known role the Kind half
// falls back to "Subagent".
func (c *Chat) threadTitle(name string) string {
	task := c.workerTasks[name]
	role := state.EmployeeRole("")
	if av, ok := c.agents[name]; ok {
		role = av.role
		if av.task != "" {
			task = av.task
		}
	}
	if task == "" {
		return "Subagent Task — " + name
	}
	if kind := threadKindLabel(role); kind != "" {
		return kind + " Task — " + task
	}
	return "Subagent Task — " + task
}

// threadSummary — the rollup wording the bordered card's collapsed
// summary row carried (no caret, no parens): "· N tool calls ✓ done",
// joined by "· M think" when thoughts ride the thread, and swung to
// "✗ stopped" for the /stop unwind. A SETTLED (done/stopped) collapsed
// header wraps it in parens ("(· 2 tool calls ✓ done)") — a LIVE header
// never carries it — and the expanded thread's closing line renders it
// bare. The counts rule matches the old summary's exactly: tools always
// counted (singular "tool call" at 1), thinks only when present.
func (c *Chat) threadSummary(g workerGroup, stopped bool) string {
	tools, thinks := 0, 0
	for _, m := range g.lines {
		switch m.Kind {
		case wthinkKind:
			thinks++
		case wdiffKind:
			// a per-call diff rides its tool call — it is NOT a second call
		default:
			tools++
		}
	}
	unit := "tool calls"
	if tools == 1 {
		unit = "tool call"
	}
	summary := "· " + itoa(tools) + " " + unit
	if thinks > 0 {
		summary += " · " + itoa(thinks) + " think"
	}
	if stopped {
		return summary + " ✗ stopped"
	}
	return summary + " ✓ done"
}

// shapeToolText — the display-side shaping that turns the reducer's
// tool text ("<lowercase verb> · <rest>" — internal/app/model.go emits
// "read · internal/panels/chat.go") into opencode's "<Verb> <rest>"
// form ("Read internal/panels/chat.go"): the head TitleCased, the " · "
// join swallowed. Only text matching ^([a-z][a-z ._-]*) · (.*)$ is
// shaped; anything else — the already-shaped target form included
// ("Read internal/panels/chat.go"), "[x] y", empty heads — passes
// through UNCHANGED, making the shaping IDEMPOTENT.
func shapeToolText(s string) string {
	i := strings.Index(s, " · ")
	if i <= 0 {
		return s
	}
	head := s[:i]
	for j, r := range head {
		switch {
		case r >= 'a' && r <= 'z':
		case j > 0 && (r == ' ' || r == '.' || r == '_' || r == '-'):
		default:
			return s // head is not the reducer's lowercase verb — pass through
		}
	}
	return strings.ToUpper(head[:1]) + head[1:] + " " + s[i+len(" · "):]
}

// wtoolStateSuffix — the state mark the EXPANDED tool rows carry:
// " ✓" done, " ✗" error, " ✗ aborted" the /stop unwind, " … running"
// anything live (or unrecognized). The ↳ sneak dropped it: a peek is
// BARE — the state lives in the header's glyph and the settled thread's
// trailing rollup.
func wtoolStateSuffix(meta string) string {
	toolState, _ := parseWtoolMeta(meta)
	switch toolState {
	case "done":
		return " ✓"
	case "error":
		return " ✗"
	case "aborted": // /stop unwind swung a running call here
		return " ✗ aborted"
	default: // running (or anything unexpected)
		return " … running"
	}
}

// workerToolLine renders one merged employee tool entry — the boss's
// inline one-liner shape built from the reducer's "<verb> · <rest>"
// text SHAPED to opencode's "<Verb> <rest>" form ("[tool] Read x ✓",
// "[tool] Read y … running", "[tool] Edit z ✗ aborted").
func workerToolLine(m state.ChatMsg) string {
	return "[tool] " + shapeToolText(m.Text) + wtoolStateSuffix(m.Meta)
}

// threadHeaderRows — the ONE title row (single-row contract: the header
// NEVER wraps — a long title elides at the panel width with clipPlain,
// no hanging continuation): a 2-cell glyph field, then the title. LIVE
// heads carry the office-tick braille glyph (threadLiveGlyph — every
// live thread pulses the same frame per tick) with the title in DEFAULT
// ink and NOTHING ELSE (no rollup while the thread is running); done
// heads dim the "✓" and the whole row; /stop-stopped heads swap to a
// dim-red "✗" with the "✗ stopped" summary. Only a SETTLED collapsed
// head trails the dim rollup "(· N tool calls[ · M think] ✓ done)" —
// clipped along with the title when the panel is narrow.
func (c *Chat) threadHeaderRows(g workerGroup, live, stopped, collapsed bool) []string {
	// the segment's OWN birth-captured title wins (chat.go's workforceGap
	// epoch split): two epochs of a recycled desk name keep their own
	// task titles. Synthetic groups without a capture (tests, legacy)
	// resolve the live sticky map as before.
	body := g.title
	if body == "" {
		body = c.threadTitle(g.name)
	}
	var glyph string
	switch {
	case stopped:
		glyph = chrome.ErrText.Faint(true).Render("✗")
	case live:
		glyph = threadLiveGlyph(c.tick) // office-tick braille run
	default:
		glyph = chrome.DimText.Render("✓")
	}
	if collapsed && (stopped || !live) {
		body += " (" + c.threadSummary(g, stopped) + ")"
	}
	body = clipPlain(body, c.contentW()-2) // 2-cell glyph field eats the head
	switch {
	case stopped:
		body = chrome.ErrText.Faint(true).Render(body)
	case !live:
		body = chrome.DimText.Render(body)
	}
	return []string{glyph + " " + body}
}

// threadSneakRows — the thread's peek at its NEWEST TOOL entry:
// "  ↳ <Verb> <rest>" — BARE (no " ✓/✗/… running" state mark; the
// header's glyph and the settled rollup carry the state), the reducer's
// "<verb> · <rest>" text run through shapeToolText. A trailing wthink
// never steals the peek — thoughts roll up in the summary counts, per
// the old summary's rule — but a think-ONLY thread peeks "thinking · N
// lines" so every thread keeps its second line. Everything dim
// (thoughts dim-italic), always exactly ONE row elided at the panel
// width with clipPlain. Rendered for EVERY thread: collapsed threads as
// the second line, expanded threads as the "current task" line under
// the tool list.
func (c *Chat) threadSneakRows(g workerGroup) []string {
	if len(g.lines) == 0 {
		return nil
	}
	lastTool := -1
	for i := len(g.lines) - 1; i >= 0; i-- {
		if g.lines[i].Kind != wthinkKind && g.lines[i].Kind != wdiffKind {
			lastTool = i
			break
		}
	}
	textStyle := chrome.DimText
	var text string
	budget := c.contentW() - 4
	if lastTool >= 0 {
		text = shapeToolText(g.lines[lastTool].Text)
		// a per-call diff under this tool peeks its counts along: the
		// dim "· +A -D" suffix eats tail budget BEFORE the clip, so the
		// sneak never overruns the single-row contract
		if dm, ok := threadDiffFor(g, g.lines[lastTool].ID); ok {
			if suf := diffCountSuffix(dm.Meta); suf != "" {
				budget -= cellWidth(suf)
				if budget < 1 {
					budget = 1
				}
				text += suf
			}
		}
	} else {
		think := g.lines[len(g.lines)-1]
		textStyle = chrome.DimText.Italic(true)
		text = "thinking · " + countLines(foldStyledRows(think.Text, c.contentW()-4, c.contentW()-4)) + " lines"
	}
	return []string{chrome.DimText.Render("  ↳ ") + textStyle.Render(clipPlain(text, budget))}
}

// threadExpandedRows — the thread's merged tool/think rows, 2-cell
// indented under the header (continuations hang 4 cells in, under the
// text): "[tool] <Verb> <rest> <state mark>" in ToolStyle (workerToolLine
// shapes the reducer's "<verb> · <rest>" text), thoughts via wthinkRows
// (bodies only on a FULL expand). A tool call whose completed edit/write
// rode a per-call patch (Kind wdiff, "wdiff-<agent>-<callid>" — the SAME
// agent+call tail as its "wtool-" id) gains a dim "· +A -D" count suffix,
// and its wdiff line renders DIRECTLY beneath as a clickable
// "↳ diff · path +A -D" sub-row (c.threadDiffOpen[id] opens the parsed
// body there — the flat-diff diffRows machinery verbatim). The second
// return maps each ↳ row's index WITHIN the returned slice to its wdiff
// msg ID for the toolDiffRows click hit-map. These rows still WRAP,
// never truncate — the single-row contract binds ONLY the header, the
// sneak and the ↳ diff sub-row.
func (c *Chat) threadExpandedRows(g workerGroup, full bool) ([]string, map[int]string) {
	var rows []string
	diffAt := map[int]string{}
	for _, m := range g.lines {
		switch {
		case m.Kind == wthinkKind:
			rows = append(rows, c.wthinkRows(m, full)...)
			continue
		case m.Kind == wdiffKind:
			diffAt[len(rows)] = m.ID
			rows = append(rows, c.wdiffRows(m)...)
			continue
		}
		suf := ""
		if dm, ok := threadDiffFor(g, m.ID); ok {
			suf = diffCountSuffix(dm.Meta)
		}
		headB := c.contentW() - 2 - cellWidth(suf)
		contB := c.contentW() - 4 - cellWidth(suf)
		if headB < 1 {
			headB = 1
		}
		if contB < 1 {
			contB = 1
		}
		lines := foldStyledRows(workerToolLine(m), headB, contB)
		for j, ln := range lines {
			prefix := "  "
			if j > 0 {
				prefix = "    "
			}
			row := chrome.ToolStyle.Render(prefix + ln)
			if j == len(lines)-1 && suf != "" {
				row += chrome.DimText.Render(suf)
			}
			rows = append(rows, row)
		}
	}
	return rows, diffAt
}

// threadDiffFor pairs a merged tool entry ("wtool-<agent>-<callid>") with
// its per-call diff ("wdiff-<agent>-<callid>") inside one thread.
func threadDiffFor(g workerGroup, toolID string) (state.ChatMsg, bool) {
	want := wdiffKind + strings.TrimPrefix(toolID, wtoolKind)
	for _, m := range g.lines {
		if m.Kind == wdiffKind && m.ID == want {
			return m, true
		}
	}
	return state.ChatMsg{}, false
}

// diffCountSuffix — the plain-text " · +A -D" counter suffix a tool row /
// sneak gains when its paired per-call diff carries counts (either side
// > 0); "" otherwise. Callers style it dim and subtract its cell width
// from the row's clip budget FIRST.
func diffCountSuffix(meta string) string {
	_, adds, dels := parseDiffMeta(meta)
	parts := ""
	if adds != "" && adds != "+0" {
		parts += " " + adds
	}
	if dels != "" && dels != "-0" {
		parts += " " + dels
	}
	if parts == "" {
		return ""
	}
	return " ·" + parts
}

// wdiffRows renders one per-call worker diff as the thread's "↳ diff ·
// path +A -D" sub-row (ONE clipped row, dim like the sneak — but the
// counts stay green/red, the flat collapsed-diff header's inks). A click
// (toolDiffRows → ToggleThreadDiff) opens the parsed body DIRECTLY
// beneath: c.diffRows' line-numbered tinted rows + the diffClip trailer,
// the flat-diff body EXACTLY as-is (thread diffs ignore ctrl+d — click
// only).
func (c *Chat) wdiffRows(m state.ChatMsg) []string {
	path, adds, dels := parseDiffMeta(m.Meta)
	counts := ""
	if adds != "" && adds != "+0" {
		counts += adds
	}
	if dels != "" && dels != "-0" {
		if counts != "" {
			counts += " "
		}
		counts += dels
	}
	labelW := c.contentW() - 4 - cellWidth(counts)
	if counts != "" {
		labelW-- // the space before the counts
	}
	if labelW < 1 {
		labelW = 1
	}
	head := chrome.DimText.Render("  ↳ ") + chrome.DimText.Render(clipPlain("diff · "+path, labelW))
	if adds != "" && adds != "+0" {
		head += " " + chrome.OKText.Render(adds)
	}
	if dels != "" && dels != "-0" {
		head += " " + chrome.ErrText.Render(dels)
	}
	rows := []string{head}
	if c.threadDiffOpen[m.ID] {
		var b strings.Builder
		drows, _ := c.diffRows(m, path)
		c.renderDiffRows(&b, drows)
		if b.Len() > 0 {
			rows = append(rows, strings.Split(strings.TrimPrefix(b.String(), "\n"), "\n")...)
		}
	}
	return rows
}

// threadClosingRows — the expanded thread's last line: the same rollup
// wording as the collapsed trailing info, bare (no parens), 2-cell
// indented under the header ("  · N tool calls[ · M think] ✓ done"),
// dim — dim-red faint once /stop stopped the thread. Never clickable
// (it is the tail of the tool list, not the thread's toggle).
func (c *Chat) threadClosingRows(g workerGroup, stopped bool) []string {
	style := chrome.DimText
	if stopped {
		style = chrome.ErrText.Faint(true)
	}
	rows := foldStyledRows(c.threadSummary(g, stopped), c.contentW()-2, c.contentW()-4)
	out := make([]string, 0, len(rows))
	for i, r := range rows {
		prefix := "  "
		if i > 0 {
			prefix = "    "
		}
		out = append(out, style.Render(prefix+r))
	}
	return out
}

// wthinkRows renders one merged employee thinking entry as thread content
// rows (2-cell indent under the header, bodies 4-cell). While the thread
// shows tool rows it is a single dim-italic "thinking · N lines" row; on
// a FULL expand (ctrl+g / a per-agent mouse override) the body renders
// too — capped to the stream tail (the freshest thinkStreamLines lines),
// matching the boss's collapsed-vs-expanded thinking shape.
func (c *Chat) wthinkRows(m state.ChatMsg, full bool) []string {
	think := chrome.DimText.Italic(true)
	// fold at the FULL body budget ("    " is 4 cells) so no row clips
	lines := foldStyledRows(m.Text, c.contentW()-4, c.contentW()-4)
	if !full {
		return []string{chrome.DimText.Render("  ") + think.Render("thinking · "+countLines(lines)+" lines")}
	}
	rows := []string{chrome.DimText.Render("  ") + think.Render("thinking")}
	shown := lines
	if more := len(lines) - thinkStreamLines; more > 0 {
		rows = append(rows, think.Render("    … "+itoa(more)+" more above"))
		shown = lines[more:]
	}
	for _, ln := range shown {
		rows = append(rows, chrome.DimText.Render("    "+ln))
	}
	return rows
}

// renderWorkerGroup draws ONE agent's thread INLINE at its timeline slot
// (mergeChatTimeline interleaves threads with the conversation — there is
// no docked region) as an opencode-style thread: header row + ↳ sneak
// row while COLLAPSED (the default — live threads included), or header +
// the merged tool/think rows + the ↳ "current task" sneak + the dim
// closing summary while EXPANDED. A per-agent override wins the ctrl+g
// baseline outright, and a /stop stopped thread force-collapses until an
// explicit expand re-opens it. The threadRows hit-map is STATE-
// CONDITIONAL (whole-bubble clicking): COLLAPSED registers the ONE
// header row plus the ONE sneak row (clicking either expands); EXPANDED
// registers the header row plus EVERY closing-summary row (the bubble's
// head and tail bracket the content — clicking either collapses), while
// the ↳ sneak (now the mid-list "current task" line, not a frame edge)
// and the internal tool/think rows pass through unclaimed. The block
// carries its OWN leading "\n\n" — the same blank-row glue the render
// loop writes between message items.
func (c *Chat) renderWorkerGroup(b *strings.Builder, g workerGroup) {
	stopped := c.threadStop[g.name]
	expanded := c.threadsExpanded
	full := c.threadsExpanded
	if v, ok := c.threadExpand[g.name]; ok {
		expanded = v
		full = v
	}
	if stopped {
		if _, ok := c.threadExpand[g.name]; !ok {
			expanded = false
		}
	}
	live := c.threadLive(g)
	header := c.threadHeaderRows(g, live, stopped, !expanded)
	sneak := c.threadSneakRows(g)
	var rows []string
	rows = append(rows, header...)
	sneakAt := len(rows)
	closingAt := -1
	var closing []string
	var diffHits map[int]string // ↳ diff sub-rows, indexed WITHIN expanded rows
	expandedAt := len(rows)
	if expanded {
		var expandedRows []string
		expandedRows, diffHits = c.threadExpandedRows(g, full)
		rows = append(rows, expandedRows...)
		sneakAt = len(rows)
		rows = append(rows, sneak...)
		closingAt = len(rows) // captured BEFORE the closing slice lands
		closing = c.threadClosingRows(g, stopped)
		rows = append(rows, closing...)
	} else {
		rows = append(rows, sneak...)
	}
	if c.threadRows != nil {
		base := strings.Count(b.String(), "\n")
		// +2, not +1: the block's own "\n\n" lead ends the previous
		// item's last row (+1) and leaves the ONE blank separator row
		// (+2) before the header lands — EXCEPT at the top edge, where
		// there is no previous item and the lead never survives (see
		// the b.Len()==0 case below), so every row lands at base+0
		lead := 2
		if b.Len() == 0 {
			lead = 0 // the group tops the transcript: renderConversation's
			// TrimLeft eats the block's own "\n\n" lead, shifting
			// every displayed row up by exactly 2
		}
		for i := range header {
			c.threadRows[base+lead+i] = g.name
		}
		if expanded {
			// the EXPANDED bubble's frame rows: head (above) + the
			// closing summary's row(s) (tail) — a click anywhere on the
			// frame folds the bubble back
			for i := range closing {
				c.threadRows[base+lead+closingAt+i] = g.name
			}
			// …while each ↳ diff sub-row registers into the SEPARATE
			// toolDiffRows map (checked after threadRows): a click there
			// toggles the diff body, never the thread itself
			for i, id := range diffHits {
				if c.toolDiffRows != nil {
					c.toolDiffRows[base+lead+expandedAt+i] = id
				}
			}
		} else {
			for i := range sneak {
				c.threadRows[base+lead+sneakAt+i] = g.name
			}
		}
	}
	b.WriteString("\n\n" + strings.Join(rows, "\n"))
}

// pendingRamp — the height-block ramp the pending bar walks: the head
// cell is solid, the wave falls to the baseline behind it. HEIGHT blocks
// only, never width blocks: "▌" is the retired stream caret — banned from
// every chat state (TestNoCaretTypingRowAboveInput pins the ban).
var pendingRamp = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// pendingBarCells — the bar's width in cells (one full ramp — the head
// wraps seamlessly past the tail).
const pendingBarCells = 8

// pendingBlockBar renders the typing row's block-glyph column for office
// tick `tick`: the solid head sits at tick%pendingBarCells with the ramp
// trailing behind it — a left-to-right breathing wave, one cell of
// progress per tick. Deterministic per tick (tick 0 gives
// "█▇▆▅▄▃▂▁"), so tests pin the frame exactly and SetState's office-tick
// pump is the ONLY driver it needs.
func pendingBlockBar(tick int) string {
	var b strings.Builder
	for i := 0; i < pendingBarCells; i++ {
		behind := (i - (tick % pendingBarCells) + 2*pendingBarCells) % pendingBarCells
		b.WriteString(pendingRamp[pendingBarCells-1-behind])
	}
	return b.String()
}

// ----------------------------------------------------------------------------- older-history pagination
//
// The transcript's TOP-OF-SCROLL walk into a session's stored history
// (the state.SessionPager fetch seam lives on the backend; the app's
// wiring — internal/app — drives these seams; threads_opencode_test.go
// pins them proof-by-proof). Four pieces, each one pure of the others:
//
//   - ThreadPager — the WALK CONTROLLER: a per-session state machine
//     (seeded / single-flight / failure-backoff / top-latched) that
//     remembers where the walk rides so the same page never fetches
//     twice, a dead serve backs off instead of hammering, and "no more"
//     is sticky. It never fetches and never renders.
//   - PrependOlder — the SPLICE: one fetched page (oldest→newest, the
//     serve's ascending order) lands BEFORE the current transcript head,
//     in order, skipping any row id already present (a resumed boot page
//     overlapping the walk can never duplicate). It re-renders the
//     posted rows but NEVER moves the scroll offset and NEVER consults
//     the pager — content-only.
//   - TranscriptRows / PreserveAnchor — the RULER + the ANCHOR:
//     TranscriptRows() reads len(selLines) verbatim (the selection row
//     cache stays FULL under the windowed viewport's blank-height model
//     — chat_window.go — so it is the transcript's true rendered
//     height); the caller snapshots it BEFORE the splice, and
//     PreserveAnchor(before) bumps the viewport offset by EXACTLY the
//     rendered growth, so the reader's top row keeps its screen cell
//     across a head splice (mechanism pinned by
//     TestWindowPaginationSeams).
//   - AtTranscriptTop — the GESTURE PROBE: true while the viewport's
//     scroll offset sits at the transcript's first row. It answers
//     position only; the WALK decision (is there more? single-flight?
//     backoff?) is the pager's.
//
// The walk is PAGE-IN at the gesture only — nothing pre-fetches, nothing
// threads events through the office state; a landing page rides these
// seams. The thread-focus view owns NO history walk (its closure's ONE
// re-render absorbs whatever the main transcript spliced meanwhile).
//
// ThreadOlderPageSize — the page size one top gesture fetches (the
// state.SessionPager limit): one screenful of history per hop — big
// enough that a long scroll-up drains in a handful of fetches, small
// enough that a hop never janks the render.
const ThreadOlderPageSize = 50

// threadPagerMaxFailures — the consecutive failure tally that backs the
// walk off (StartOlder refuses) until ResetFailures re-arms: three
// straight misses reads as a dead serve, not a flaky hop, and the walk
// keeps its cursor so the retry continues where it stopped — history
// doesn't move, so it never rescans.
const threadPagerMaxFailures = 3

// ThreadPager — the older-history walk controller ONE transcript owns.
// PURE STATE: no fetch (the backend answers those), no pixels (the chat
// renders those). Its contract is the guard contract: Seed latches the
// walk anchor once, StartOlder single-flights one hop and hands out its
// cursor, FinishOlder lands it (advancing the cursor, latching the top),
// FailOlder strikes a failure, ResetFailures re-arms the backoff.
type ThreadPager struct {
	sessionID string // the session this walk reads (the boot's pin)
	cursor    string // the NEXT hop's `before` (the last page's NextCursor)
	top       bool   // FinishOlder(hasMore=false) latched: the oldest page is in
	inFlight  bool   // a hop is outstanding (StartOlder's single-flight mark)
	failures  int    // consecutive FailOlders since the last FinishOlder
	seeded    bool   // Seed ran — the walk has an anchor
}

// NewThreadPager binds a walk controller to a session id (the office's
// primary at boot; a session swap builds a fresh pager).
func NewThreadPager(sessionID string) *ThreadPager {
	return &ThreadPager{sessionID: sessionID}
}

// Seed latches the walk anchor from the BOOT page's continuation: the
// cursor the first top-gesture hop rides and whether any older page
// exists at all (hasMore=false latches the top at once — a boot page
// that finishes the history arms nothing). IDEMPOTENT: a re-seed is a
// no-op — the walk never moves backwards; a session swap starts a new
// pager instead.
func (p *ThreadPager) Seed(cursor string, hasMore bool) {
	if p.seeded {
		return
	}
	p.seeded = true
	p.cursor = cursor
	p.top = !hasMore
}

// StartOlder guards ONE older hop: false when the walk never seeded
// (nothing fetched yet), sits at the transcript's top (FinishOlder
// latched it), has a hop outstanding (single-flight), or has struck
// threadPagerMaxFailures consecutive failures (ResetFailures re-arms).
// True latches the in-flight mark and hands out the cursor the hop
// rides; the caller MUST land it with exactly one FinishOlder/FailOlder
// — the mark holds until then.
func (p *ThreadPager) StartOlder() (cursor string, ok bool) {
	if !p.seeded || p.top || p.inFlight || p.failures >= threadPagerMaxFailures {
		return "", false
	}
	p.inFlight = true
	return p.cursor, true
}

// FinishOlder lands a successful hop: the single-flight mark clears, the
// failure tally resets, the NEXT hop rides the page's NextCursor, and a
// hasMore=false page latches the top PERMANENTLY (history only grows at
// the tail; once the oldest page is in, there is never more).
func (p *ThreadPager) FinishOlder(nextCursor string, hasMore bool) {
	p.inFlight = false
	p.failures = 0
	p.cursor = nextCursor
	if !hasMore {
		p.top = true
	}
}

// FailOlder abandons a failed hop: the single-flight mark clears (the
// gesture can retry — the cursor NEVER moved, history doesn't drift) and
// the consecutive-failure tally bumps; at threadPagerMaxFailures
// StartOlder backs off until ResetFailures.
func (p *ThreadPager) FailOlder() {
	p.inFlight = false
	p.failures++
}

// ResetFailures re-arms a backed-off walk (a fresh explicit top gesture
// after a quiet spell is a retry, not spam): the tally zeroes, the
// cursor stays — the next hop asks exactly where the walk stopped.
func (p *ThreadPager) ResetFailures() {
	p.failures = 0
}

// TranscriptRows — the transcript's FULL rendered row count: len(selLines)
// verbatim. The blank-height window (chat_window.go) leaves selLines
// whole — it is the selection row-space cache AND the scroll offset
// space — so an older-page splice grows this by exactly the page's
// rendered rows, and PreserveAnchor's compensation is the delta of two
// reads around the splice.
func (c *Chat) TranscriptRows() int { return len(c.selLines) }

// PreserveAnchor — the prepend-compensation seam: the caller reads
// TranscriptRows() BEFORE the splice and hands that snapshot over after
// PrependOlder landed; the viewport offset bumps by exactly the rendered
// growth so the reader's current top row keeps its screen cell. A stale
// snapshot (growth <= 0 — a double-bump guard, a no-op re-render) moves
// nothing. The compensated rows materialize lazily at the next paint:
// the windowed viewport's View seam (syncWindow) catches the bump — the
// exact mechanism TestWindowPaginationSeams pins verbatim.
func (c *Chat) PreserveAnchor(beforeRows int) {
	growth := c.TranscriptRows() - beforeRows
	if growth <= 0 {
		return
	}
	c.vp.SetYOffset(c.vp.YOffset() + growth)
}

// AtTranscriptTop — the top-gesture probe: true while the transcript
// viewport's scroll offset sits at the FIRST row (the blank-height model
// keeps the offset space byte-identical — YOffset 0 IS the transcript's
// head, blanks or not). Position-pure: a transcript shorter than the
// viewport reads at-top too — arming an actual hop needs the pager's
// guards (hasMore, single-flight, backoff), never this alone.
func (c *Chat) AtTranscriptTop() bool { return c.vp.YOffset() == 0 }

// PrependOlder splices ONE fetched older page into the transcript HEAD:
// the page's rows (oldest→newest — the serve's ascending order, exactly
// what state.SessionPager walks) map to chat entries and land BEFORE the
// current first entry, in page order. Idempotent per message id: a row
// whose id is ALREADY in the transcript is skipped (a resumed boot page
// overlapping the walk, a double-arriving landing — neither duplicates;
// the result count returns FRESH rows only). Re-renders the posted rows
// (a splice grows TranscriptRows immediately) but NEVER touches the
// scroll offset, the follow latch, or any pager — the anchor is
// PreserveAnchor's, the walk is ThreadPager's: content-only. Returns the
// number of rows ADDED after dedupe.
//
// The splice is panel-local: the app pairs it with its own ledger (a
// fetched page must also ride INTO the reducer's slice or the next
// office SetState re-bases the transcript — model.go owns that wiring);
// this seam keeps the splice mechanics provable in isolation.
func (c *Chat) PrependOlder(rows []state.SessionMessageRow) int {
	if len(rows) == 0 {
		return 0
	}
	seen := make(map[string]bool, len(c.chat)+len(rows))
	for _, m := range c.chat {
		seen[m.ID] = true
	}
	fresh := make([]state.ChatMsg, 0, len(rows))
	for _, r := range rows {
		if r.ID != "" && seen[r.ID] {
			continue // already in the transcript — a duplicate never lands
		}
		seen[r.ID] = true // and a page never duplicates ITSELF
		fresh = append(fresh, sessionRowToChat(r))
	}
	if len(fresh) == 0 {
		return 0
	}
	merged := make([]state.ChatMsg, 0, len(fresh)+len(c.chat))
	merged = append(merged, fresh...)
	merged = append(merged, c.chat...)
	c.chat = merged
	// Re-render the posted rows NOW (the walk's ruler reads the growth)
	// but respect the reader: no offset move (PreserveAnchor's lane), no
	// bottom snap (a head splice while parked must never yank).
	c.setConversationLines(c.renderConversationLines())
	return len(fresh)
}

// sessionRowToChat maps ONE fetched history row to its transcript entry:
// a user wire role renders as a user turn, an assistant role as a boss
// turn, anything else as an office note. Text-bearing parts join with
// "\n\n"; tool/reasoning-typed parts without bodies leave no row (the
// splice renders history for reading — it never replays calls). At rides
// Created, so the merged timeline keeps the page's ascending order.
func sessionRowToChat(r state.SessionMessageRow) state.ChatMsg {
	var texts []string
	for _, p := range r.Parts {
		if p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	m := state.ChatMsg{ID: r.ID, Text: strings.Join(texts, "\n\n"), At: r.Created}
	switch r.Role {
	case "user":
		m.From, m.Kind = "user", "user"
	case "assistant":
		m.From, m.Kind = "boss", "boss"
	default:
		m.From = officeFrom
	}
	return m
}
