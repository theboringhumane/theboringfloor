// chat_toolrow_test.go — behavior proofs for CLICK-TO-EXPAND TOOL
// OUTPUTS (chat_toolrow.go + the chat.go/threads_opencode.go seams):
//
//	(a) EXPAND: a click on a boss inline tool row opens the captured
//	    ToolOutput under it — dim, wrapped to the chat width at the
//	    row's hanging indent, ▸ → ▾;
//	(b) COLLAPSE: a second click on the one-liner folds the body back
//	    to the exact pre-click shape (and the body rows themselves are
//	    click-dead);
//	(c) EMPTY STATE: no captured output (running tool, output-less
//	    tool, pre-capture event) expands to the pinned ONE-row dim
//	    "no output as such";
//	(d) RUNNING → DONE IN PLACE: expanded while running (empty state),
//	    the done event + SetToolOutput update the SAME expanded body —
//	    the expansion never flickers closed;
//	(e) INDEPENDENCE: two tool rows expand/collapse independently
//	    (a set of expanded entry IDs);
//	(f) THE CAP: output past toolOutputMaxRows keeps its FIRST rows +
//	    the "… N more lines" tail note;
//	(g) ALL KINDS: a worker thread's wtool row expands the same way
//	    (no per-tool-name filtering — the row's chevron lives inside
//	    the expanded thread);
//	(h) CACHE HONESTY: SetToolOutput while expanded repaints WITHOUT a
//	    SetState (the block key folds the output — forceRender alone
//	    carries the pixel change).
//
// No clocks: every timestamp is a literal; toggles ride ClickRow /
// ToggleToolOutput directly (the same seams the app's mouse handler
// runs).
package panels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	state "github.com/theboringhumane/theboringoffice/internal/state"
)

// toolRowChat builds the bare chat holding ONE completed boss tool call
// ("tool-c1", read · x.go ✓). SetSize is the Chat's CONTENT width (the
// fold tests' convention), so 44 here means transcript TEXT runs
// contentW() = 44 − chatPadL − chatPadR = 40 cells wide.
func toolRowChat(t *testing.T) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "tool-c1", From: "boss", Kind: "tool", Text: "read · x.go", Meta: "done", At: 10},
	}})
	return c
}

// toolRowOf reports the content row a tool entry's one-liner registered
// in the toolRows hit-map (the click target the app's mouse seam lands
// on), failing the test when the row never registered.
func toolRowOf(t *testing.T, c *Chat, id string) int {
	t.Helper()
	for row, rid := range c.toolRows {
		if rid == id {
			return row
		}
	}
	t.Fatalf("tool entry %q registered NO click row (map: %v)", id, c.toolRows)
	return -1
}

// TestToolRowExpandShowsOutput is proof (a): the click opens the body —
// the captured output's rows render dim-wrapped under the one-liner at
// the [tool] ▸ hanging indent, and the chevron flips to ▾.
func TestToolRowExpandShowsOutput(t *testing.T) {
	c := toolRowChat(t)
	c.SetToolOutput("tool-c1", "alpha one\nbeta two")
	row := toolRowOf(t, c, "tool-c1")
	if !c.ClickRow(3, row) {
		t.Fatal("a click on the tool row was not claimed")
	}
	convo := ansi.Strip(c.renderConversation())
	t.Logf("---- EXPANDED tool row (44 cols, ansi-stripped) ----\n%s\n----", convo)
	want := "[tool] ▾ read · x.go ✓\n" +
		"         alpha one\n" +
		"         beta two"
	if convo != want {
		t.Fatalf("expanded tool row shape:\n got %q\nwant %q", convo, want)
	}
	// the chevron signaled the state flip; the one-liner still registers
	if c.toolRows[row] != "tool-c1" {
		t.Fatalf("the expanded one-liner must keep its click row (%d → tool-c1), got %v", row, c.toolRows)
	}
	// the body rows are click-DEAD (they never register)
	if _, ok := c.toolRows[row+1]; ok {
		t.Fatalf("a body row must never register in toolRows, got %v", c.toolRows)
	}
	if c.ClickRow(3, row+1) {
		t.Fatal("a click on the output BODY row must fall through unclaimed")
	}
}

// TestToolRowExpandWrapsLongOutput is proof (a, second half): a long
// single-line output wraps to the chat width — every wrapped row hangs
// at the one-liner's 9-cell text indent and none overflows the content
// budget.
func TestToolRowExpandWrapsLongOutput(t *testing.T) {
	c := toolRowChat(t)
	long := strings.Repeat("word ", 30) // 150 cells of wrappable text
	c.SetToolOutput("tool-c1", strings.TrimSpace(long))
	c.ToggleToolOutput("tool-c1")
	convo := ansi.Strip(c.renderConversation())
	t.Logf("---- WRAPPED output body (44 cols, ansi-stripped) ----\n%s\n----", convo)
	lines := strings.Split(convo, "\n")
	if len(lines) < 4 { // one-liner + ≥3 wrapped body rows
		t.Fatalf("a 150-cell output must wrap to several body rows, got %d:\n%s", len(lines), convo)
	}
	for i, ln := range lines[1:] {
		if !strings.HasPrefix(ln, "         ") {
			t.Fatalf("body row %d must hang at the 9-cell indent: %q", i+1, ln)
		}
		if w := cellWidth(ln); w > 39 { // toolW = contentW − 1
			t.Fatalf("body row %d overflows the fold budget (%d cells): %q", i+1, w, ln)
		}
	}
}

// TestToolRowCollapseHidesOutput is proof (b): the second click folds
// back to the byte-exact collapsed shape.
func TestToolRowCollapseHidesOutput(t *testing.T) {
	c := toolRowChat(t)
	c.SetToolOutput("tool-c1", "alpha one")
	collapsed := ansi.Strip(c.renderConversation())
	if !strings.Contains(collapsed, "[tool] ▸ read · x.go ✓") {
		t.Fatalf("precondition: the collapsed row carries the ▸ chevron:\n%s", collapsed)
	}
	row := toolRowOf(t, c, "tool-c1")
	if !c.ClickRow(3, row) || !c.ClickRow(3, row) {
		t.Fatal("the expand/collapse click round-trip was not claimed")
	}
	if convo := ansi.Strip(c.renderConversation()); convo != collapsed {
		t.Fatalf("the second click must restore the byte-exact collapsed shape:\n got %q\nwant %q", convo, collapsed)
	}
}

// TestToolRowEmptyState is proof (c): a tool with NO captured output
// expands to exactly the pinned one-row dim "no output as such".
func TestToolRowEmptyState(t *testing.T) {
	c := toolRowChat(t) // no SetToolOutput anywhere — the pre-capture shape
	c.ToggleToolOutput("tool-c1")
	convo := ansi.Strip(c.renderConversation())
	want := "[tool] ▾ read · x.go ✓\n" +
		"         " + toolOutputEmpty
	if convo != want {
		t.Fatalf("empty-state expanded shape:\n got %q\nwant %q", convo, want)
	}
	// exactly ONE body row, and it is the dim-rendered pinned copy
	raw := c.renderConversation()
	body := strings.Split(raw, "\n")[1]
	if body != "         "+chrome.DimText.Render(toolOutputEmpty) {
		t.Fatalf("the empty state must be the ONE dim pinned row, got %q", body)
	}
}

// TestToolRowRunningThenDoneInPlace is proof (d): expanded while the
// tool runs (empty state), the done merge + SetToolOutput update the
// SAME expanded body — c.toolExpanded never flips, no collapse flicker.
func TestToolRowRunningThenDoneInPlace(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "tool-c1", From: "boss", Kind: "tool", Text: "bash · go build ./...", Meta: "running", At: 10},
	}})
	c.ToggleToolOutput("tool-c1")
	if convo := ansi.Strip(c.renderConversation()); !strings.Contains(convo, toolOutputEmpty) {
		t.Fatalf("while running the expanded body must read %q:\n%s", toolOutputEmpty, convo)
	}
	// the done event lands: the reducer's merge REPLACES the entry
	// (same ID, Meta running→done) and the app feeds the output FIRST
	// (applyEventCore's order: SetToolOutput before SetState)
	c.SetToolOutput("tool-c1", "build ok")
	c.SetState(state.OfficeState{Tick: 2, Chat: []state.ChatMsg{
		{ID: "tool-c1", From: "boss", Kind: "tool", Text: "bash · go build ./...", Meta: "done", At: 10},
	}})
	if !c.toolExpanded["tool-c1"] {
		t.Fatal("the running→done merge must NEVER collapse an expanded row")
	}
	convo := ansi.Strip(c.renderConversation())
	want := "[tool] ▾ bash · go build ./... ✓\n" +
		"         build ok"
	if convo != want {
		t.Fatalf("the done event must update the expanded body in place:\n got %q\nwant %q", convo, want)
	}
	if strings.Contains(convo, toolOutputEmpty) {
		t.Fatalf("the empty state must be gone once the output landed:\n%s", convo)
	}
}

// TestToolRowsIndependent is proof (e): two rows expand and collapse
// independently of each other.
func TestToolRowsIndependent(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(44, 30)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "tool-a", From: "boss", Kind: "tool", Text: "read · a.go", Meta: "done", At: 10},
		{ID: "tool-b", From: "boss", Kind: "tool", Text: "grep · needle", Meta: "done", At: 11},
	}})
	c.SetToolOutput("tool-a", "alpha body")
	c.SetToolOutput("tool-b", "beta body")
	c.ToggleToolOutput("tool-a")
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "[tool] ▾ read · a.go ✓") || !strings.Contains(convo, "alpha body") {
		t.Fatalf("row A must be expanded with its body:\n%s", convo)
	}
	if !strings.Contains(convo, "[tool] ▸ grep · needle ✓") || strings.Contains(convo, "beta body") {
		t.Fatalf("row B must stay collapsed while A is open:\n%s", convo)
	}
	c.ToggleToolOutput("tool-b")
	convo = ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "alpha body") || !strings.Contains(convo, "beta body") {
		t.Fatalf("both bodies must render while both rows are expanded:\n%s", convo)
	}
	c.ToggleToolOutput("tool-a")
	convo = ansi.Strip(c.renderConversation())
	if strings.Contains(convo, "alpha body") || !strings.Contains(convo, "[tool] ▸ read · a.go ✓") {
		t.Fatalf("row A must fold independently:\n%s", convo)
	}
	if !strings.Contains(convo, "[tool] ▾ grep · needle ✓") || !strings.Contains(convo, "beta body") {
		t.Fatalf("row B must survive A's collapse:\n%s", convo)
	}
}

// TestToolRowOutputCap is proof (f): output past the row cap keeps its
// FIRST toolOutputMaxRows rows + the "… N more lines" tail note.
func TestToolRowOutputCap(t *testing.T) {
	c := toolRowChat(t)
	var lines []string
	for i := 1; i <= toolOutputMaxRows+6; i++ {
		lines = append(lines, fmt.Sprintf("row-%02d", i))
	}
	c.SetToolOutput("tool-c1", strings.Join(lines, "\n"))
	c.ToggleToolOutput("tool-c1")
	convo := ansi.Strip(c.renderConversation())
	t.Logf("---- CAPPED output body (44 cols, ansi-stripped) ----\n%s\n----", convo)
	if !strings.Contains(convo, "row-01") || !strings.Contains(convo, fmt.Sprintf("row-%02d", toolOutputMaxRows)) {
		t.Fatalf("the capped body keeps its FIRST %d rows:\n%s", toolOutputMaxRows, convo)
	}
	if strings.Contains(convo, fmt.Sprintf("row-%02d", toolOutputMaxRows+1)) {
		t.Fatalf("row %d must hide under the cap:\n%s", toolOutputMaxRows+1, convo)
	}
	if !strings.Contains(convo, "… 6 more lines") {
		t.Fatalf("the tail note must read %q:\n%s", "… 6 more lines", convo)
	}
	// one-liner + cap + note — nothing more
	if n := len(strings.Split(convo, "\n")); n != 1+toolOutputMaxRows+1 {
		t.Fatalf("the capped expansion must be 1+%d+1 rows, got %d:\n%s", toolOutputMaxRows, n, convo)
	}
}

// TestToolRowWorkerThreadExpands is proof (g): a worker thread's wtool
// row carries the same chevron + click-to-expand body — no per-kind
// filtering — while the thread's own toggle stays on its frame rows.
func TestToolRowWorkerThreadExpands(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(60, 30)
	c.SetState(state.OfficeState{Tick: 1, Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "ship it", At: 1},
		{ID: "wtool-tekton-1-c9", From: "tekton-1", Kind: "wtool", Text: "read · internal/panels/chat.go", Meta: "done\x1f1", At: 10},
	}})
	c.ToggleThread("tekton-1") // the output toggle needs the row on screen
	convo := ansi.Strip(c.renderConversation())
	if !strings.Contains(convo, "  [tool] ▸ Read internal/panels/chat.go ✓") {
		t.Fatalf("the thread's tool row must carry the ▸ chevron:\n%s", convo)
	}
	row := toolRowOf(t, c, "wtool-tekton-1-c9")
	c.SetToolOutput("wtool-tekton-1-c9", "package panels")
	if !c.ClickRow(3, row) {
		t.Fatal("a click on the thread's tool row was not claimed")
	}
	convo = ansi.Strip(c.renderConversation())
	t.Logf("---- EXPANDED wtool row inside the thread (60 cols, ansi-stripped) ----\n%s\n----", convo)
	if !strings.Contains(convo, "  [tool] ▾ Read internal/panels/chat.go ✓") {
		t.Fatalf("the wtool chevron must flip to ▾:\n%s", convo)
	}
	if !strings.Contains(convo, "\n    package panels") {
		t.Fatalf("the wtool body must hang at the thread's 4-cell content indent:\n%s", convo)
	}
	tbAssertExpanded(t, c, "tekton-1", true, "after the tool-row click (thread untouched)")
	// and the second click folds just the body
	if !c.ClickRow(3, row) {
		t.Fatal("the second click on the thread's tool row was not claimed")
	}
	if convo = ansi.Strip(c.renderConversation()); strings.Contains(convo, "package panels") {
		t.Fatalf("the wtool body must fold back:\n%s", convo)
	}
	tbAssertExpanded(t, c, "tekton-1", true, "after the tool-row re-click (thread still untouched)")
}

// TestToolRowSetOutputRepaintsExpanded is proof (h): SetToolOutput
// while the row is expanded changes the RENDERED pixels with NO
// SetState call — the block-cache key folds the output, so
// forceRender alone carries the change (the borrow must miss).
func TestToolRowSetOutputRepaintsExpanded(t *testing.T) {
	c := toolRowChat(t)
	c.ToggleToolOutput("tool-c1")
	before := ansi.Strip(c.View())
	if !strings.Contains(before, toolOutputEmpty) {
		t.Fatalf("precondition: the expanded row shows the empty state:\n%s", before)
	}
	c.SetToolOutput("tool-c1", "late capture") // no SetState anywhere
	after := ansi.Strip(c.View())
	if after == before {
		t.Fatal("SetToolOutput on an expanded row must repaint without a SetState")
	}
	if !strings.Contains(after, "late capture") || strings.Contains(after, toolOutputEmpty) {
		t.Fatalf("the expanded body must swap to the landed output:\n%s", after)
	}
	// collapsed rows take NO repaint (no pixels depend on the map)
	c.ToggleToolOutput("tool-c1") // fold
	before = ansi.Strip(c.View())
	c.SetToolOutput("tool-c1", "late capture v2")
	if after := ansi.Strip(c.View()); after != before {
		t.Fatal("SetToolOutput on a COLLAPSED row must not repaint")
	}
}

// TestToolRowANSISanitized — captured output's own escape sequences
// never bleed into the transcript (a bash color run would otherwise
// tint every following row): the body renders them stripped.
func TestToolRowANSISanitized(t *testing.T) {
	c := toolRowChat(t)
	c.SetToolOutput("tool-c1", "\x1b[31mred text\x1b[0m")
	c.ToggleToolOutput("tool-c1")
	raw := c.renderConversation()
	if strings.Contains(raw, "\x1b[31m") {
		t.Fatalf("the output's own SGR must be stripped from the body:\n%q", raw)
	}
	if !strings.Contains(ansi.Strip(raw), "red text") {
		t.Fatalf("the stripped text still renders:\n%q", raw)
	}
}
