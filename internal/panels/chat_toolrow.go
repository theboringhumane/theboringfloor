// chat_toolrow.go — CLICK-TO-EXPAND TOOL OUTPUTS: any tool row in the
// transcript (the boss's inline "[tool] …" one-liner AND a worker
// thread's merged "[tool] …" row — no per-tool-name filtering, every
// tool event with a CallID) toggles an expanded body under the row: the
// captured ToolOutput (state.Event.ToolOutput — the app feeds it
// panel-side via SetToolOutput, keyed by the SAME chat-entry id the
// reducer's merge uses) rendered dim + wrapped to the chat width,
// height-capped at toolOutputMaxRows rows with a "… N more lines" tail
// note. The row carries a chevron — ▸ collapsed, ▾ expanded. A tool
// with NO captured output (still running, output-less, or an event from
// before the capture existed) expands to the honest one-row dim empty
// state: "no output as such".
//
// The mechanics are the user-fold pair's exact twins (chat.go's
// userExpanded/userFoldRows): toolExpanded holds the explicit per-entry
// state (default collapsed — rows expand/collapse INDEPENDENTLY, keyed
// by the entry's message ID, so a running→done merge that REPLACES the
// entry keeps the expansion: the done event updates the expanded body
// in place, never a collapse flicker); toolRows is the mouse hit-map
// (rendered row → entry ID, rebuilt every render, the one-liner's own
// rows ONLY — body rows pass through unclaimed, the same contract the
// ↳ diff bodies keep); ClickRow consults it right after toolDiffRows.
// Toggle mutations ride forceRender exactly like ToggleUserFold (the
// SetState revision gate compares state, not expansion).
//
// The block cache folds the expansion + the output text into each tool
// block's key (renderMsgBlock: bfExpandA + the output string while
// expanded; renderGroupBlock: "tool-open:<id>" + the output), so a
// SetToolOutput landing while the row is expanded re-renders exactly
// that block even when the done event carried no message change.
//
// The body rows are ordinary transcript rows: they ride selLines, so
// the office's existing mouse-selection machinery (press-drag-release →
// OSC52 copy) selects them like any other text.
package panels

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
)

// toolOutputMaxRows — the expanded body's height cap in rendered rows:
// past it the body shows its FIRST rows plus a dim-italic
// "… N more lines" tail note (the full text stays selectable nowhere
// else — the cap is the transcript's height hygiene, the same shape the
// think stream's "… N more above" tail keeps).
const toolOutputMaxRows = 12

// toolOutputEmpty — the MEMBER-PINNED empty-state copy (exact, never
// paraphrased): a tool with no captured output — running, output-less,
// or pre-capture — expands to this ONE dim row.
const toolOutputEmpty = "no output as such"

// toolChevron — the row's expand/collapse signal: ▸ collapsed (click
// opens), ▾ expanded (click folds).
func toolChevron(open bool) string {
	if open {
		return "▾ "
	}
	return "▸ "
}

// ToggleToolOutput flips ONE tool entry's output body between collapsed
// (the bare one-liner) and expanded (the wrapped dim body under the
// row) — the ToggleUserFold twin for tool rows: RENDER state only, so
// it rides forceRender (the SetState revision gate compares state, not
// expansion — without the force the click would look dead).
func (c *Chat) ToggleToolOutput(id string) {
	if c.toolExpanded == nil {
		c.toolExpanded = map[string]bool{}
	}
	c.toolExpanded[id] = !c.toolExpanded[id]
	c.forceRender()
}

// SetToolOutput records ONE tool entry's captured result text (the
// app's EvTool.ToolOutput feed, keyed by the reducer's merge id —
// "tool-<callID>" / "wtool-<agent>-<callID>"). Empty outputs never
// overwrite (a done event without output keeps a partial running
// capture); the app calls this only for non-empty payloads. When the
// entry is expanded RIGHT NOW the body updates in place (forceRender +
// the block key's output fold) — no collapse flicker; collapsed rows
// pick the text up at their next expand (no pixels depend on it until
// then).
func (c *Chat) SetToolOutput(id, output string) {
	if id == "" || output == "" {
		return
	}
	if c.toolOutputs == nil {
		c.toolOutputs = map[string]string{}
	}
	if c.toolOutputs[id] == output {
		return
	}
	c.toolOutputs[id] = output
	if c.toolExpanded[id] {
		c.forceRender()
	}
}

// toolOutputRows renders ONE entry's expanded body: every row dim (the
// terminal IS the monospace — "mono" here means the raw output text,
// never markdown), wrapped to budget cells, indented indentCells under
// the row's text start. Incoming ANSI is stripped first — a bash
// escape must never bleed color into the transcript's rows. The empty
// state is the pinned "no output as such" (one dim row). Over
// toolOutputMaxRows the body keeps its FIRST rows + the
// "… N more lines" tail note.
func (c *Chat) toolOutputRows(id string, indentCells, budget int) []string {
	indent := strings.Repeat(" ", indentCells)
	if budget < 1 {
		budget = 1
	}
	out := ansi.Strip(c.toolOutputs[id])
	if strings.TrimSpace(out) == "" {
		return []string{indent + chrome.DimText.Render(toolOutputEmpty)}
	}
	var rows []string
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		rows = append(rows, foldStyledRows(ln, budget, budget)...)
	}
	more := 0
	if len(rows) > toolOutputMaxRows {
		more = len(rows) - toolOutputMaxRows
		rows = rows[:toolOutputMaxRows]
	}
	body := make([]string, 0, len(rows)+1)
	for _, ln := range rows {
		body = append(body, indent+chrome.DimText.Render(ln))
	}
	if more > 0 {
		body = append(body, indent+chrome.DimText.Italic(true).Render("… "+itoa(more)+" more lines"))
	}
	return body
}

// pruneToolOutputs drops output captures whose chat entry left the
// transcript (capChat's fuse, /clear): keyed on the entries SetState
// just adopted, so a long session never accretes outputs for rows that
// no longer render. Size-gated — the scan runs only once the map can
// actually hold stale ids.
func (c *Chat) pruneToolOutputs() {
	if len(c.toolOutputs) <= len(c.chat)+64 {
		return
	}
	alive := make(map[string]bool, len(c.chat))
	for _, m := range c.chat {
		alive[m.ID] = true
	}
	for id := range c.toolOutputs {
		if !alive[id] {
			delete(c.toolOutputs, id)
		}
	}
}
