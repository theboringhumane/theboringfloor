// gitpanel_test.go — pure/no-exec coverage of the GIT tab: glyph + diff-line
// classification, middle-clip row truncation, ContentOffset-adjusted click
// hit-mapping, and the async msg flow (status/diff) against injected fake
// fetch seams — git is never exec'd, and the pinned tick loop keeps
// wall-clock tea.Tick cmds out of every executed cmd (nothing sleeps).
package panels

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/gitx"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// --- fixtures -------------------------------------------------------------

func gitFixtureFiles() []gitx.FileStatus {
	return []gitx.FileStatus{
		{Path: "internal/app/model.go", Code: " M", Staged: false},
		{Path: "internal/panels/gitpanel.go", Code: "A ", Staged: true},
		{Path: "README.md", Code: "??", Staged: false},
		{Path: "internal/gitx/gitx.go", Code: " D", Staged: false},
		{Path: "docs/OLD.md", Code: "R ", Staged: true},
	}
}

func gitFixtureSummary() gitx.Summary {
	return gitx.Summary{Modified: 3, Added: 1, Untracked: 2, Deleted: 1, LinesAdded: 120, LinesRemoved: 45}
}

const gitFixtureDiff = `diff --git a/internal/app/model.go b/internal/app/model.go
index 1111111..2222222 100644
--- a/internal/app/model.go
+++ b/internal/app/model.go
@@ -10,6 +10,7 @@ func (m Model) Update
 context line
-old line
+new line
 tail context
`

// fakeGitBackend records seam calls; no git binary is ever executed.
type fakeGitBackend struct {
	files     []gitx.FileStatus
	summary   gitx.Summary
	statusErr error
	diffText  string
	diffErr   error

	statusCalls  int
	diffCalls    int
	lastDiffPath string
}

// newFakeGit builds the panel with fake fetch seams and a FROZEN clock.
// The tick loop is pinned (tickSeq preset + fresh lastTickAt) so the
// trailing ensureLoop inside Update never arms a wall-clock tea.Tick —
// every cmd these tests execute is a fake fetch that returns immediately.
func newFakeGit(fb *fakeGitBackend) *Git {
	g := NewGit(gitx.Repo{Root: "/fake"})
	now := time.UnixMilli(1_700_000_000_000)
	g.nowFn = func() time.Time { return now }
	g.lastFetch = now
	g.lastTickAt = now
	g.tickSeq = 1
	g.dirty = false
	g.statusFn = func() ([]gitx.FileStatus, gitx.Summary, error) {
		fb.statusCalls++
		return fb.files, fb.summary, fb.statusErr
	}
	g.diffFn = func(path string) (string, error) {
		fb.diffCalls++
		fb.lastDiffPath = path
		return fb.diffText, fb.diffErr
	}
	return g
}

// deliverGitCmds executes cmd (recursively unwrapping tea.BatchMsg) and
// feeds every produced message back into g.Update. Only ever runs fake
// fetches — the pinned loop above keeps tea.Tick out of the mix.
func deliverGitCmds(g *Git, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			deliverGitCmds(g, c)
		}
		return
	}
	if msg != nil {
		g.Update(msg)
	}
}

// loadFiles drives one full status+stat round-trip through the msg surface.
func loadFiles(g *Git) {
	g.dirty = true
	g.lastFetch = time.Time{}
	deliverGitCmds(g, g.maybeRefresh())
}

func gitKey(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	case "esc":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})
	case "up":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	case "down":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	case "pgup":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp})
	case "pgdown":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown})
	default:
		r := []rune(s)[0]
		return tea.KeyPressMsg(tea.Key{Code: r, Text: s})
	}
}

// --- classifiers ------------------------------------------------------------

func TestGitClassifyGlyph(t *testing.T) {
	cases := []struct {
		code  string
		glyph string
		kind  gitGlyphKind
	}{
		{"??", "??", gitGlyphUntracked},
		{"A ", "A", gitGlyphAdded},
		{"AM", "A", gitGlyphAdded}, // A wins per spec precedence
		{"AD", "A", gitGlyphAdded},
		{" D", "D", gitGlyphDeleted},
		{"D ", "D", gitGlyphDeleted},
		{"R ", "R", gitGlyphRenamed},
		{"RM", "R", gitGlyphRenamed},
		{" M", "M", gitGlyphModified},
		{"M ", "M", gitGlyphModified},
		{"MM", "M", gitGlyphModified},
		{"  ", "M", gitGlyphModified},
	}
	for _, tc := range cases {
		glyph, kind := classifyGlyph(tc.code)
		if glyph != tc.glyph || kind != tc.kind {
			t.Errorf("classifyGlyph(%q) = (%q, %d), want (%q, %d)", tc.code, glyph, kind, tc.glyph, tc.kind)
		}
	}
}

func TestGitGlyphCellStagedMarker(t *testing.T) {
	cell, _ := glyphCell(gitx.FileStatus{Code: " M", Staged: true})
	if cell != "M*" {
		t.Fatalf("staged modified cell = %q, want %q", cell, "M*")
	}
	cell, _ = glyphCell(gitx.FileStatus{Code: " M"})
	if cell != "M " {
		t.Fatalf("unstaged modified cell = %q, want %q", cell, "M ")
	}
	cell, _ = glyphCell(gitx.FileStatus{Code: "??", Staged: true})
	if cell != "??" { // 2-col glyph: never grows a third column
		t.Fatalf("untracked cell = %q, want %q", cell, "??")
	}
}

func TestGitClassifyDiffLine(t *testing.T) {
	cases := []struct {
		line  string
		class gitLineClass
	}{
		{"+new line", gitLineAdd},
		{"-old line", gitLineDel},
		{"@@ -10,6 +10,7 @@ func (m Model) Update", gitLineHunk},
		{"diff --git a/x b/x", gitLineMeta},
		{"index 1111111..2222222 100644", gitLineMeta},
		{"--- a/x", gitLineMeta}, // meta wins over the "-" rule
		{"+++ b/x", gitLineMeta}, // meta wins over the "+" rule
		{" context", gitLinePlain},
		{"", gitLinePlain},
		{"\\ No newline at end of file", gitLinePlain},
	}
	for _, tc := range cases {
		if got := classifyDiffLine(tc.line); got != tc.class {
			t.Errorf("classifyDiffLine(%q) = %d, want %d", tc.line, got, tc.class)
		}
	}
}

func TestGitStyleDiffLineColors(t *testing.T) {
	if got, want := styleDiffLine("+x"), chrome.OKText.Render("+x"); got != want {
		t.Errorf("add line style mismatch: got %q want %q", got, want)
	}
	if got, want := styleDiffLine("-x"), chrome.ErrText.Render("-x"); got != want {
		t.Errorf("del line style mismatch: got %q want %q", got, want)
	}
	if got, want := styleDiffLine("@@ -1 +1 @@"), chrome.InfoText.Render("@@ -1 +1 @@"); got != want {
		t.Errorf("hunk style mismatch: got %q want %q", got, want)
	}
	if got, want := styleDiffLine("index abc"), chrome.DimText.Render("index abc"); got != want {
		t.Errorf("meta style mismatch: got %q want %q", got, want)
	}
	if got := styleDiffLine(" plain"); got != " plain" {
		t.Errorf("plain line must pass through unstyled, got %q", got)
	}
}

// --- truncation -------------------------------------------------------------

func TestGitMiddleClip(t *testing.T) {
	if got := middleClip("short.go", 80); got != "short.go" {
		t.Errorf("short path must pass through, got %q", got)
	}
	got := middleClip("internal/panels/gitpanel.go", 12)
	if len([]rune(got)) != 12 || !strings.Contains(got, "…") {
		t.Errorf("middleClip to 12 = %q (runes %d), want 12 runes with an ellipsis", got, len([]rune(got)))
	}
	if want := "intern" + "…" + "el.go"; got != want {
		t.Errorf("middleClip = %q, want %q", got, want)
	}
	if got := middleClip("abcdefghij", 1); got != "…" {
		t.Errorf("width 1 = %q, want ellipsis", got)
	}
	if got := middleClip("anything", 0); got != "" {
		t.Errorf("width 0 = %q, want empty", got)
	}
}

func TestGitRowRenderTruncatesMiddle(t *testing.T) {
	fs := gitx.FileStatus{Path: "internal/panels/gitpanel.go", Code: " M"}
	row := renderGitRow(fs, false, 12)
	if w := lipgloss.Width(row); w > 12 {
		t.Fatalf("row wider than budget: %d cells (%q)", w, row)
	}
	if !strings.Contains(row, "…") {
		t.Fatalf("long path must be middle-truncated with an ellipsis, got %q", row)
	}
	// both ends of the path survive the elision
	if !strings.Contains(row, "in") || !strings.Contains(row, ".go") {
		t.Fatalf("middle truncation must keep both path ends, got %q", row)
	}
}

func TestGitRowRenderCursorAndStaged(t *testing.T) {
	plain := renderGitRow(gitx.FileStatus{Path: "a.go", Code: " M"}, false, 30)
	if strings.Contains(plain, "›") {
		t.Fatalf("unselected row must not carry the cursor marker, got %q", plain)
	}
	sel := renderGitRow(gitx.FileStatus{Path: "a.go", Code: " M"}, true, 30)
	if !strings.Contains(sel, chrome.AccentText.Render("› ")) {
		t.Fatalf("selected row must carry the accent cursor marker, got %q", sel)
	}
	staged := renderGitRow(gitx.FileStatus{Path: "a.go", Code: "A ", Staged: true}, false, 30)
	if !strings.Contains(staged, chrome.OKText.Render("A*")) {
		t.Fatalf("staged added row must carry the starred green glyph, got %q", staged)
	}
}

// --- click hit-mapping (ContentOffset-adjusted) -------------------------------

func TestGitListRowAt(t *testing.T) {
	fb := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary()}
	g := newFakeGit(fb)
	g.SetSize(30, 10)
	loadFiles(g)

	dx, dy := (&Tabs{}).ContentOffset() // the ONE exported geometry fact
	y0 := dy + gitHeaderRows            // first list row in sidebar-box space

	if row := g.listRowAt(dx+2, y0); row != 0 {
		t.Errorf("click on row 0 = %d, want 0", row)
	}
	if row := g.listRowAt(dx+2, y0+4); row != 4 {
		t.Errorf("click on row 4 = %d, want 4", row)
	}
	if row := g.listRowAt(dx+2, y0+5); row != -1 {
		t.Errorf("click past the list = %d, want -1", row)
	}
	if row := g.listRowAt(dx+2, dy+1); row != -1 {
		t.Errorf("click on the summary header = %d, want -1", row)
	}
	if row := g.listRowAt(0, y0); row != -1 {
		t.Errorf("click on the border column = %d, want -1", row)
	}

	g.SetSize(30, 5)   // 3 body rows for 5 files: real scroll room (max YOffset 2)
	g.vp.SetYOffset(2) // scrolled: the same screen row maps deeper
	if row := g.listRowAt(dx+2, y0); row != 2 {
		t.Errorf("scrolled click on screen row 0 = %d, want 2", row)
	}
	g.vp.SetYOffset(0)

	g.mode = gitModeDiff
	if row := g.listRowAt(dx+2, y0); row != -1 {
		t.Errorf("diff mode must not map list rows, got %d", row)
	}
	g.mode = gitModeList
	g.err = errors.New("boom")
	if row := g.listRowAt(dx+2, y0); row != -1 {
		t.Errorf("error body must not map list rows, got %d", row)
	}
}

// --- header render ------------------------------------------------------------

func TestGitHeaderRender(t *testing.T) {
	sum := gitFixtureSummary()
	want := "3" + chrome.DimText.Render(" mod") + chrome.DimText.Render(" · ") +
		"1" + chrome.DimText.Render(" add") + chrome.DimText.Render(" · ") +
		"2" + chrome.DimText.Render(" untr") + chrome.DimText.Render(" · ") +
		"1" + chrome.DimText.Render(" del")
	if got := renderGitSummary(sum, 80); got != fitLabel(want, 80) {
		t.Errorf("summary line mismatch:\n got %q\nwant %q", got, fitLabel(want, 80))
	}
	want2 := chrome.OKText.Render("+120") + " " + chrome.ErrText.Render("−45")
	if got := renderGitNumstat(sum); got != want2 {
		t.Errorf("numstat line mismatch:\n got %q\nwant %q", got, want2)
	}
}

// --- async msg flow ---------------------------------------------------------

func TestGitStatusFlowDebounced(t *testing.T) {
	fb := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary()}
	g := newFakeGit(fb)
	g.SetSize(40, 20)
	loadFiles(g)

	if fb.statusCalls != 1 {
		t.Fatalf("first refresh must fetch once, got %d calls", fb.statusCalls)
	}
	if g.err != nil || len(g.files) != len(gitFixtureFiles()) {
		t.Fatalf("status msg must land files cleanly, err=%v files=%d", g.err, len(g.files))
	}
	if g.fetching {
		t.Fatalf("fetching latch must clear when the msg lands")
	}
	if out := g.View(); !strings.Contains(out, "internal/app/model.go") {
		t.Fatalf("list must render the fetched files, got:\n%s", out)
	}

	// debounce window: a dirty mark inside it is swallowed
	g.SetState(state.OfficeState{}) // the refresh clock marks dirty
	if cmd := g.maybeRefresh(); cmd != nil {
		t.Fatalf("debouncer must swallow a refresh inside the window")
	}
	if fb.statusCalls != 1 {
		t.Fatalf("debounced refresh must not re-exec, got %d calls", fb.statusCalls)
	}

	// window passed → the same dirty mark fires one more fetch
	g.lastFetch = g.nowFn().Add(-2 * gitRefreshEvery)
	cmd := g.maybeRefresh()
	if cmd == nil {
		t.Fatalf("a dirty panel past the debounce window must refresh")
	}
	if cmd2 := g.maybeRefresh(); cmd2 != nil {
		t.Fatalf("never double-exec while a fetch is in flight")
	}
	deliverGitCmds(g, cmd)
	if fb.statusCalls != 2 {
		t.Fatalf("post-window refresh must fetch exactly once more, got %d", fb.statusCalls)
	}

	// tick loop: a stale generation drops silently, the live one re-arms
	if cmd := g.Update(gitTickMsg{seq: 999}); cmd != nil {
		t.Fatalf("stale tick must drop, got a cmd back")
	}
	if cmd := g.Update(gitTickMsg{seq: g.tickSeq}); cmd == nil {
		t.Fatalf("the live tick must re-arm (do not execute — it sleeps)")
	}
}

func TestGitDiffFlow(t *testing.T) {
	fb := &fakeGitBackend{
		files:    gitFixtureFiles(),
		summary:  gitFixtureSummary(),
		diffText: gitFixtureDiff,
	}
	g := newFakeGit(fb)
	g.SetSize(40, 8) // 5 diff body rows: the 9-line fixture scrolls
	loadFiles(g)

	// cursor to row 1, enter opens its diff (async)
	g.Update(gitKey("down"))
	if g.cursor != 1 {
		t.Fatalf("down must move the cursor to 1, got %d", g.cursor)
	}
	cmd := g.Update(gitKey("enter"))
	if g.mode != gitModeDiff {
		t.Fatalf("enter must switch to diff mode")
	}
	if want := gitFixtureFiles()[1].Path; g.diffPath != want {
		t.Fatalf("diff must open the cursor file %q, got %q", want, g.diffPath)
	}
	if cmd == nil {
		t.Fatalf("enter must return the async diff fetch cmd")
	}
	deliverGitCmds(g, cmd)
	if fb.diffCalls != 1 || fb.lastDiffPath != gitFixtureFiles()[1].Path {
		t.Fatalf("diff fetch must run once for the cursor path, got %d calls for %q", fb.diffCalls, fb.lastDiffPath)
	}

	view := g.View()
	if !strings.Contains(view, chrome.Header.Render("─ "+gitFixtureFiles()[1].Path+" ─")) {
		t.Errorf("diff header must pin the path, got:\n%s", view)
	}
	// body-level content checks: the fixture's bottom lines sit below the
	// 5-row viewport fold, so assert the staged vp content (which View
	// draws), not the visible window. Long lines are clipped to g.w first.
	content := g.renderDiff()
	if !strings.Contains(content, "+new line") || !strings.Contains(content, "-old line") {
		t.Errorf("diff body must render fetched lines, got:\n%s", content)
	}
	if !strings.Contains(content, chrome.OKText.Render("+new line")) {
		t.Errorf("additions must render green, got:\n%s", content)
	}
	hunk := clipPlain("@@ -10,6 +10,7 @@ func (m Model) Update", 40)
	if !strings.Contains(content, chrome.InfoText.Render(hunk)) {
		t.Errorf("hunk headers must render cyan, got:\n%s", content)
	}

	// in diff mode up/down scroll the diff, they do NOT move the cursor
	before := g.cursor
	for i := 0; i < 3; i++ {
		g.Update(gitKey("down"))
	}
	if g.cursor != before {
		t.Fatalf("diff-mode down must not move the list cursor (%d → %d)", before, g.cursor)
	}
	if g.vp.YOffset() == 0 {
		t.Fatalf("diff-mode down must scroll the diff body")
	}

	// r reloads status+stat AND the open diff
	statusBefore, diffBefore := fb.statusCalls, fb.diffCalls
	deliverGitCmds(g, g.Update(gitKey("r")))
	if fb.statusCalls != statusBefore+1 {
		t.Fatalf("r must reload status (+1), got %d calls", fb.statusCalls)
	}
	if fb.diffCalls != diffBefore+1 {
		t.Fatalf("r must reload the open diff (+1), got %d calls", fb.diffCalls)
	}
	if g.mode != gitModeDiff {
		t.Fatalf("r must keep the diff open")
	}

	// b returns to the list with cursor + scroll state preserved
	g.Update(gitKey("b"))
	if g.mode != gitModeList {
		t.Fatalf("b must return to list mode")
	}
	if g.cursor != before {
		t.Fatalf("back must preserve the list cursor %d, got %d", before, g.cursor)
	}
	if out := g.View(); !strings.Contains(out, "internal/app/model.go") {
		t.Fatalf("list state must survive the diff round-trip, got:\n%s", out)
	}

	// esc returns too
	g.Update(gitKey("enter"))
	if g.mode != gitModeDiff {
		t.Fatalf("enter on the preserved cursor must reopen its diff")
	}
	g.Update(gitKey("esc"))
	if g.mode != gitModeList {
		t.Fatalf("esc must return to list mode")
	}
}

func TestGitClickOpensDiff(t *testing.T) {
	fb := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary(), diffText: gitFixtureDiff}
	g := newFakeGit(fb)
	g.SetSize(40, 20)
	loadFiles(g)

	dx, dy := (&Tabs{}).ContentOffset()
	y0 := dy + gitHeaderRows
	click := tea.MouseClickMsg(tea.Mouse{X: dx + 3, Y: y0 + 2, Button: tea.MouseLeft})
	cmd := g.Update(click)
	if g.mode != gitModeDiff {
		t.Fatalf("a row click must open the diff")
	}
	if want := gitFixtureFiles()[2].Path; g.diffPath != want {
		t.Fatalf("row click must open row 2's %q, got %q", want, g.diffPath)
	}
	if g.cursor != 2 {
		t.Fatalf("the click selects its row too, cursor = %d", g.cursor)
	}
	deliverGitCmds(g, cmd)
	if fb.diffCalls != 1 {
		t.Fatalf("click must fire exactly one async diff fetch, got %d", fb.diffCalls)
	}

	// back to list; a click on the summary header opens nothing
	g.Update(gitKey("b"))
	if cmd := g.Update(tea.MouseClickMsg(tea.Mouse{X: dx + 3, Y: dy, Button: tea.MouseLeft})); cmd != nil {
		t.Fatalf("header clicks must not fetch")
	}
	if g.mode != gitModeList {
		t.Fatalf("header clicks must leave the panel in list mode")
	}
}

// --- render states ------------------------------------------------------------

func TestGitRenderGlyphsAndColors(t *testing.T) {
	fb := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary()}
	g := newFakeGit(fb)
	g.SetSize(46, 20)
	loadFiles(g)

	out := g.View()
	for want := range map[string]struct{}{
		chrome.WarnText.Render("M "):   {}, // unstaged modified
		chrome.OKText.Render("A*"):     {}, // staged added
		chrome.OKText.Render("??"):     {}, // untracked
		chrome.ErrText.Render("D "):    {}, // deleted
		chrome.InfoText.Render("R*"):   {}, // staged renamed
		chrome.OKText.Render("+120"):   {},
		chrome.ErrText.Render("−45"):   {},
		chrome.DimText.Render(" untr"): {},
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view must contain %q, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "internal/panels/gitpanel.go") {
		t.Errorf("view must render file paths, got:\n%s", out)
	}
}

func TestGitRenderCleanAndUnavailable(t *testing.T) {
	// clean tree
	fb := &fakeGitBackend{files: nil, summary: gitx.Summary{}}
	g := newFakeGit(fb)
	g.SetSize(40, 10)
	loadFiles(g)
	if out := g.View(); !strings.Contains(out, chrome.OKText.Render("working tree clean")) {
		t.Errorf("clean repo must show the green clean state, got:\n%s", out)
	}

	// backend error (the ErrNotImplemented skeleton phase): dim line, no panic
	fb2 := &fakeGitBackend{statusErr: gitx.ErrNotImplemented}
	g2 := newFakeGit(fb2)
	g2.SetSize(60, 10)
	loadFiles(g2)
	out := g2.View()
	if !strings.Contains(out, chrome.DimText.Render("git unavailable: "+gitx.ErrNotImplemented.Error())) {
		t.Errorf("backend error must render the dim unavailable line, got:\n%s", out)
	}
	if g2.err == nil {
		t.Errorf("the error must be latched for the body render")
	}

	// diff fetch failure: same contract inside the diff view
	fb3 := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary(), diffErr: errors.New("diff boom")}
	g3 := newFakeGit(fb3)
	g3.SetSize(40, 10)
	loadFiles(g3)
	deliverGitCmds(g3, g3.Update(gitKey("enter")))
	if out := g3.View(); !strings.Contains(out, chrome.DimText.Render("git unavailable: diff boom")) {
		t.Errorf("diff error must render the dim unavailable line, got:\n%s", out)
	}
}

func TestGitBootOrderSetStateBeforeSetSize(t *testing.T) {
	// hydrateSession pushes state before the first SetSize (mail_test.go's
	// boot-order regression): no panic, no exec, renders at zero size.
	fb := &fakeGitBackend{files: gitFixtureFiles(), summary: gitFixtureSummary()}
	g := newFakeGit(fb)
	g.SetState(state.OfficeState{})
	_ = g.View()
	g.SetSize(60, 20)
	loadFiles(g)
	if out := g.View(); !strings.Contains(out, "README.md") {
		t.Fatalf("deferred render at real size must show files, got:\n%s", out)
	}
}

func TestGitCursorKeepsVisible(t *testing.T) {
	files := gitFixtureFiles()
	files = append(files,
		gitx.FileStatus{Path: "extra/one.go", Code: " M"},
		gitx.FileStatus{Path: "extra/two.go", Code: " M"},
		gitx.FileStatus{Path: "extra/three.go", Code: " M"},
	)
	fb := &fakeGitBackend{files: files, summary: gitFixtureSummary()}
	g := newFakeGit(fb)
	g.SetSize(30, 7) // 5 body rows
	loadFiles(g)

	for i := 0; i < 6; i++ {
		g.Update(gitKey("j"))
	}
	if g.cursor != 6 {
		t.Fatalf("j moves the cursor, got %d", g.cursor)
	}
	if got, want := g.vp.YOffset(), 6-5+1; got != want {
		t.Fatalf("cursor must stay visible: YOffset %d, want %d", got, want)
	}
	g.Update(gitKey("k"))
	if g.cursor != 5 {
		t.Fatalf("k moves the cursor back, got %d", g.cursor)
	}
	// clamp at both ends
	for i := 0; i < 20; i++ {
		g.Update(gitKey("k"))
	}
	if g.cursor != 0 {
		t.Fatalf("cursor must clamp at 0, got %d", g.cursor)
	}
	if g.vp.YOffset() != 0 {
		t.Fatalf("viewport must follow the cursor to the top, YOffset %d", g.vp.YOffset())
	}
}
