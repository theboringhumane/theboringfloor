// gitpanel.go — GIT tab: a read-only view of the working tree backed by
// internal/gitx (status porcelain + numstat header + per-file unified
// diffs). The panel compiles against the FROZEN gitx contract; while the
// backend skeleton returns gitx.ErrNotImplemented the body shows a dim
// "git unavailable: <err>" line instead of panicking or going blank.
//
// Layout (list mode):
//
//	3 mod · 1 add · 2 untr · 1 del      <- summary line (dim labels)
//	+120 −45                            <- numstat (+ green, − red)
//	› M* internal/panels/gitpanel.go    <- scrollable file rows
//	  ?? internal/panels/gitpanel_test.go
//
// Diff mode pins a "─ <path> ─" header under the summary and scrolls the
// per-line colored diff instead. enter / click opens the cursor file's
// diff, b / esc returns (list cursor + scroll survive), r reloads
// status+stat (and the open diff).
//
// Refresh clock: every git fetch runs inside a tea.Cmd (the TUI thread
// never execs git — same per-frame exec ban as projinfo's TTL cache).
// SetState (the panels' refresh clock, fired on every office tick) only
// marks dirty; a re-arming tea.Tick loop (gitTickMsg, seq-guarded so a
// single tick is ever in flight) converts dirty into a fetch at most once
// per gitRefreshEvery. NewGit can't hand its first fetch to the runtime —
// the app batches Init() at startup; the first Update self-arms as a
// fallback so the panel still works when wiring forgets.
package panels

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/gitx"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// gitRefreshEvery bounds both the SetState→fetch debounce and the
// re-arming refresh tick: while the office is ticking (180ms–3s) git
// status is re-fetched at most this often, never per frame.
const gitRefreshEvery = 2 * time.Second

// gitHeaderRows — the two summary lines pinned above the body viewport in
// every mode (summary + numstat).
const gitHeaderRows = 2

// Panel view modes.
const (
	gitModeList = iota // scrollable file rows
	gitModeDiff        // pinned path header + colored diff body
)

// gitGlyphKind is the semantic category a porcelain code maps to (drives
// the glyph's chrome style).
type gitGlyphKind int

const (
	gitGlyphModified gitGlyphKind = iota
	gitGlyphAdded
	gitGlyphUntracked
	gitGlyphDeleted
	gitGlyphRenamed
)

// gitLineClass is the style class of one raw unified-diff line.
type gitLineClass int

const (
	gitLinePlain gitLineClass = iota
	gitLineAdd                // "+" prefix — green
	gitLineDel                // "-" prefix — red
	gitLineHunk               // "@@" hunk header — cyan
	gitLineMeta               // diff --git / index / --- / +++ — dim
)

// --- async messages (every fetch lands as one of these) -----------------

// gitStatusMsg delivers one statusFn (status + stat) run.
type gitStatusMsg struct {
	files   []gitx.FileStatus
	summary gitx.Summary
	err     error
}

// gitDiffMsg delivers one diffFn run for path.
type gitDiffMsg struct {
	path string
	text string
	err  error
}

// gitTickMsg is the seq-guarded refresh tick (a stale seq = a newer loop
// owns the cadence; the dead message silently drops).
type gitTickMsg struct{ seq int }

// Git is the git sidebar tab panel.
type Git struct {
	repo gitx.Repo
	vp   viewport.Model
	w, h int

	files   []gitx.FileStatus
	summary gitx.Summary
	err     error // last status/stat error → "git unavailable" body
	cursor  int   // selected file row (list mode)

	mode     int // gitModeList | gitModeDiff
	diffPath string
	diffText string
	diffErr  error

	dirty      bool      // SetState's refresh-clock mark
	fetching   bool      // a status fetch is in flight (never double-exec)
	lastFetch  time.Time // debounce anchor of the last fired fetch
	tickSeq    int       // generation counter — one live tick at a time
	lastTickAt time.Time // loop-liveness probe for ensureLoop

	// test seams: default to the repo methods; tests inject fakes so the
	// msg flow never execs git.
	statusFn func() ([]gitx.FileStatus, gitx.Summary, error)
	diffFn   func(string) (string, error)
	nowFn    func() time.Time
}

// NewGit builds the git panel for repo and pre-arms the first status+stat
// fetch (dirty + a zero debounce anchor — the first cmd pass fires it).
func NewGit(repo gitx.Repo) *Git {
	vp := viewport.New(viewport.WithWidth(10), viewport.WithHeight(5))
	vp.MouseWheelEnabled = true
	g := &Git{
		repo:  repo,
		vp:    vp,
		dirty: true,
		nowFn: time.Now,
	}
	g.statusFn = g.fetchStatusStat
	g.diffFn = g.repo.Diff
	return g
}

// Title implements Tab.
func (g *Git) Title() string { return "git" }

// SetSize implements Tab; re-renders the body at the new width.
func (g *Git) SetSize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	g.w, g.h = w, h
	g.vp.SetWidth(w)
	g.resizeVP()
	g.refreshBody()
}

// SetState implements Tab: the office tick is the panels' refresh clock —
// every push marks the panel dirty; the debounced tick loop (or the next
// key/mouse pass) turns it into ONE in-flight status+stat fetch. The open
// diff is deliberately NOT refreshed here (r reloads it).
func (g *Git) SetState(_ state.OfficeState) { g.dirty = true }

// Init kicks the first async status+stat fetch and arms the refresh tick.
// The app batches this once at startup — a constructor cannot hand a cmd
// to the runtime.
func (g *Git) Init() tea.Cmd {
	return tea.Batch(g.maybeRefresh(), g.armTick())
}

// Update implements Interactive: list cursor / diff scroll keys, enter and
// clicks open diffs, wheel scrolls the body, the tick loop and the typed
// fetch messages drive all data in async.
func (g *Git) Update(msg tea.Msg) tea.Cmd {
	cmd := g.handle(msg)
	return tea.Batch(cmd, g.ensureLoop(), g.maybeRefresh())
}

// View implements Tab: 2-line summary header + body, fitted to g.h. Pure
// string assembly of cached state — never a fetch.
func (g *Git) View() string {
	var b strings.Builder
	b.WriteString(g.renderHeader())
	if g.mode == gitModeDiff {
		b.WriteString("\n")
		b.WriteString(g.renderDiffHeader())
	}
	b.WriteString("\n")
	b.WriteString(g.vp.View())
	return fit(b.String(), g.h)
}

// ---------------------------------------------------------------------------
// message handling
// ---------------------------------------------------------------------------

// handle dispatches one message, returning only the message's own cmd
// (Update bolts the loop/refresh ensure onto every pass).
func (g *Git) handle(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case gitTickMsg:
		if msg.seq != g.tickSeq {
			return nil // a newer loop owns the cadence
		}
		return tea.Batch(g.armTick(), g.maybeRefresh())
	case gitStatusMsg:
		g.applyStatus(msg)
		return nil
	case gitDiffMsg:
		if g.mode == gitModeDiff && g.diffPath == msg.path {
			g.diffErr = msg.err
			g.diffText = msg.text
			g.refreshBody()
			g.vp.SetYOffset(0)
		}
		return nil
	case tea.KeyPressMsg:
		return g.handleKey(msg)
	case tea.MouseWheelMsg:
		// raw to the active tab: wheel scrolls the list or the open diff
		var cmd tea.Cmd
		g.vp, cmd = g.vp.Update(msg)
		return cmd
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return nil
		}
		if row := g.listRowAt(msg.X, msg.Y); row >= 0 {
			g.cursor = row
			return g.openDiff()
		}
		return nil
	}
	return nil
}

// handleKey — mode-dependent key surface. List: j/k + arrows move the
// cursor, enter opens the diff, pgup/pgdn scroll the list. Diff: the same
// keys scroll the diff instead, b/esc returns, r reloads everything.
func (g *Git) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	if g.mode == gitModeDiff {
		switch key {
		case "b", "esc":
			g.mode = gitModeList
			g.resizeVP()
			g.refreshBody() // list state (cursor + scroll) preserved
			return nil
		case "r":
			return g.refreshAll()
		case "up", "down", "k", "j", "pgup", "pgdn", "pgdown", "home", "end":
			var cmd tea.Cmd
			g.vp, cmd = g.vp.Update(msg)
			return cmd
		}
		return nil
	}
	switch key {
	case "up", "k":
		g.moveCursor(-1)
	case "down", "j":
		g.moveCursor(1)
	case "enter":
		return g.openDiff()
	case "r":
		return g.refreshAll()
	case "pgup", "pgdn", "pgdown":
		var cmd tea.Cmd
		g.vp, cmd = g.vp.Update(msg)
		return cmd
	}
	return nil
}

// applyStatus lands one gitStatusMsg: error state takes over the list
// body (never a panic/blank); success replaces files + summary and clamps
// the cursor into the (possibly shrunk) list.
func (g *Git) applyStatus(msg gitStatusMsg) {
	g.fetching = false
	g.err = msg.err
	if msg.err != nil {
		g.files = nil
		g.summary = gitx.Summary{}
		g.cursor = 0
	} else {
		g.files = msg.files
		g.summary = msg.summary
		g.clampCursor()
	}
	if g.mode == gitModeList {
		g.refreshBody()
	}
}

// ---------------------------------------------------------------------------
// commands (the ONLY place git data is fetched — all async)
// ---------------------------------------------------------------------------

// fetchStatusStat is the default statusFn: the repo's Status + Stat fused
// into one result (either error fails the pair).
func (g *Git) fetchStatusStat() ([]gitx.FileStatus, gitx.Summary, error) {
	files, ferr := g.repo.Status()
	sum, serr := g.repo.Stat()
	if ferr != nil {
		return files, sum, ferr
	}
	return files, sum, serr
}

// fetchStatusCmd runs statusFn off the UI goroutine, delivering gitStatusMsg.
func (g *Git) fetchStatusCmd() tea.Cmd {
	fn := g.statusFn
	return func() tea.Msg {
		files, sum, err := fn()
		return gitStatusMsg{files: files, summary: sum, err: err}
	}
}

// fetchDiffCmd runs diffFn(path) off the UI goroutine, delivering gitDiffMsg.
func (g *Git) fetchDiffCmd(path string) tea.Cmd {
	fn := g.diffFn
	return func() tea.Msg {
		text, err := fn(path)
		return gitDiffMsg{path: path, text: text, err: err}
	}
}

// maybeRefresh fires the debounced status+stat fetch when the refresh
// clock marked us dirty, no fetch is in flight, and the debounce window
// passed. The constructor's zero lastFetch makes the first call fire.
func (g *Git) maybeRefresh() tea.Cmd {
	if !g.dirty || g.fetching {
		return nil
	}
	if g.nowFn().Sub(g.lastFetch) < gitRefreshEvery {
		return nil
	}
	g.dirty = false
	g.fetching = true
	g.lastFetch = g.nowFn()
	return g.fetchStatusCmd()
}

// refreshAll is the r key: a full manual refresh, debounce-bypassing —
// status+stat plus the open diff's reload.
func (g *Git) refreshAll() tea.Cmd {
	g.dirty = false
	g.fetching = true
	g.lastFetch = g.nowFn()
	cmd := g.fetchStatusCmd()
	if g.mode == gitModeDiff && g.diffPath != "" {
		cmd = tea.Batch(cmd, g.fetchDiffCmd(g.diffPath))
	}
	return cmd
}

// armTick re-arms the seq-guarded refresh tick (one live tick at a time).
func (g *Git) armTick() tea.Cmd {
	g.tickSeq++
	seq := g.tickSeq
	g.lastTickAt = g.nowFn()
	return tea.Tick(gitRefreshEvery, func(time.Time) tea.Msg { return gitTickMsg{seq: seq} })
}

// ensureLoop re-arms the tick when the loop was never started (the app
// forgot Init) or provably died (no tick landed for two debounce windows —
// e.g. it fired while another tab was active). Bounded chatter: at most
// one extra arm per window.
func (g *Git) ensureLoop() tea.Cmd {
	if g.tickSeq == 0 || g.nowFn().Sub(g.lastTickAt) >= 2*gitRefreshEvery {
		return g.armTick()
	}
	return nil
}

// ---------------------------------------------------------------------------
// list state
// ---------------------------------------------------------------------------

// moveCursor steps the file cursor (clamped) and keeps the row visible.
func (g *Git) moveCursor(d int) {
	if len(g.files) == 0 {
		return
	}
	g.cursor += d
	g.clampCursor()
	g.refreshBody()
	g.keepCursorVisible()
}

// clampCursor pins the cursor inside the current file list.
func (g *Git) clampCursor() {
	if g.cursor >= len(g.files) {
		g.cursor = len(g.files) - 1
	}
	if g.cursor < 0 {
		g.cursor = 0
	}
}

// keepCursorVisible scrolls the body viewport so the cursor row is on screen.
func (g *Git) keepCursorVisible() {
	if g.cursor < g.vp.YOffset() {
		g.vp.SetYOffset(g.cursor)
	}
	if h := g.vp.Height(); h > 0 && g.cursor >= g.vp.YOffset()+h {
		g.vp.SetYOffset(g.cursor - h + 1)
	}
}

// openDiff switches to diff mode for the cursor file and kicks its async
// load. No-op on an error/empty list.
func (g *Git) openDiff() tea.Cmd {
	if g.err != nil || g.cursor < 0 || g.cursor >= len(g.files) {
		return nil
	}
	path := g.files[g.cursor].Path
	g.mode = gitModeDiff
	g.diffPath = path
	g.diffText = ""
	g.diffErr = nil
	g.resizeVP()
	g.refreshBody() // loading body until gitDiffMsg lands
	g.vp.SetYOffset(0)
	return g.fetchDiffCmd(path)
}

// listRowAt maps a sidebar mouse point to a files index: subtract the
// Tabs box chrome (Tabs.ContentOffset — the ONE exported geometry fact;
// the app forwards clicks in sidebar-box space), then the panel's own
// 2-line summary header, then add the viewport scroll offset. -1 = miss
// (header rows, out of bounds, diff mode, error body).
func (g *Git) listRowAt(x, y int) int {
	if g.mode != gitModeList || g.err != nil {
		return -1
	}
	dx, dy := (&Tabs{}).ContentOffset()
	cx, cy := x-dx, y-dy
	if cx < 0 || cx >= g.w {
		return -1
	}
	row := cy - gitHeaderRows + g.vp.YOffset()
	if row < 0 || row >= len(g.files) {
		return -1
	}
	return row
}

// ---------------------------------------------------------------------------
// layout + render (pure string assembly of cached state)
// ---------------------------------------------------------------------------

// bodyRows — viewport height for the current mode (the diff mode's pinned
// path header eats one extra row).
func (g *Git) bodyRows() int {
	rows := g.h - gitHeaderRows
	if g.mode == gitModeDiff {
		rows--
	}
	return rows
}

// resizeVP syncs the viewport height to the current mode's body budget.
func (g *Git) resizeVP() {
	h := g.bodyRows()
	if h < 1 {
		h = 1
	}
	g.vp.SetHeight(h)
}

// refreshBody rebuilds the viewport content for the active mode.
func (g *Git) refreshBody() {
	if g.mode == gitModeDiff {
		g.vp.SetContent(g.renderDiff())
	} else {
		g.vp.SetContent(g.renderList())
	}
}

// renderHeader — the pinned two summary lines.
func (g *Git) renderHeader() string {
	return renderGitSummary(g.summary, g.w) + "\n" + renderGitNumstat(g.summary)
}

// renderGitSummary — line 1: "3 mod · 1 add · 2 untr · 1 del", numbers in
// default ink, labels + separators dim. fitLabel clips ANSI-safely.
func renderGitSummary(s gitx.Summary, w int) string {
	segs := []struct {
		n     int
		label string
	}{
		{s.Modified, " mod"},
		{s.Added, " add"},
		{s.Untracked, " untr"},
		{s.Deleted, " del"},
	}
	var b strings.Builder
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(chrome.PanelDim.Render(" · "))
		}
		b.WriteString(strconv.Itoa(seg.n))
		b.WriteString(chrome.PanelDim.Render(seg.label))
	}
	return fitLabel(b.String(), w)
}

// renderGitNumstat — line 2: "+120 −45", additions green, removals red
// (U+2212 minus, matching the design string).
func renderGitNumstat(s gitx.Summary) string {
	return chrome.PanelOK.Render("+"+strconv.Itoa(s.LinesAdded)) + " " +
		chrome.PanelErr.Render("−"+strconv.Itoa(s.LinesRemoved))
}

// renderList — the scrollable file rows, the clean-state string, or the
// dim unavailable line.
func (g *Git) renderList() string {
	if g.err != nil {
		return chrome.PanelDim.Render(clipPlain("git unavailable: "+g.err.Error(), g.w))
	}
	if len(g.files) == 0 {
		return chrome.PanelOK.Render("working tree clean")
	}
	rows := make([]string, len(g.files))
	for i, fs := range g.files {
		rows[i] = renderGitRow(fs, i == g.cursor, g.w)
	}
	return strings.Join(rows, "\n")
}

// renderGitRow — one list row: cursor marker (2 cols) + 2-col status
// glyph + path, middle-truncated with "…" when long. Pure; width-driven.
func renderGitRow(fs gitx.FileStatus, selected bool, w int) string {
	if w < 1 {
		w = 1
	}
	marker := "  "
	if selected {
		marker = chrome.PanelAccent.Render("› ")
	}
	cell, kind := glyphCell(fs)
	pathW := w - 2 - 2 - 1 // marker + glyph cell + separator space
	if pathW < 1 {
		pathW = 1
	}
	row := marker + glyphStyle(kind).Render(cell) + " " + middleClip(fs.Path, pathW)
	if selected {
		row = lipgloss.NewStyle().Bold(true).Render(row)
	}
	return row
}

// renderDiffHeader — the pinned "─ <path> ─" line of the diff mode.
func (g *Git) renderDiffHeader() string {
	w := g.w - 4 // "─ " + " ─"
	if w < 1 {
		w = 1
	}
	return chrome.PanelHeader.Render("─ " + middleClip(g.diffPath, w) + " ─")
}

// renderDiff — the colored unified-diff body (or the dim error/loading line).
func (g *Git) renderDiff() string {
	if g.diffErr != nil {
		return chrome.PanelDim.Render(clipPlain("git unavailable: "+g.diffErr.Error(), g.w))
	}
	if g.diffText == "" {
		return chrome.PanelDim.Render("loading diff …")
	}
	lines := strings.Split(strings.TrimRight(g.diffText, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = styleDiffLine(clipPlain(ln, g.w))
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// glyph + diff-line classifiers (pure, unit-tested)
// ---------------------------------------------------------------------------

// classifyGlyph maps a porcelain status code to its glyph + category:
// "??" untracked, else first hit of A / D / R, else modified.
func classifyGlyph(code string) (glyph string, kind gitGlyphKind) {
	switch {
	case code == "??":
		return "??", gitGlyphUntracked
	case strings.Contains(code, "A"):
		return "A", gitGlyphAdded
	case strings.Contains(code, "D"):
		return "D", gitGlyphDeleted
	case strings.Contains(code, "R"):
		return "R", gitGlyphRenamed
	default:
		return "M", gitGlyphModified
	}
}

// glyphCell — the 2-col glyph cell: single-letter glyphs gain a "*"
// staged marker, else a padding space ("M*" / "M "; "??" is already 2).
func glyphCell(fs gitx.FileStatus) (string, gitGlyphKind) {
	glyph, kind := classifyGlyph(fs.Code)
	if len(glyph) == 1 {
		if fs.Staged {
			return glyph + "*", kind
		}
		return glyph + " ", kind
	}
	return glyph, kind
}

// glyphStyle — semantic chrome style per glyph category (never hardcoded
// colors): untracked/added green, deleted red, renamed cyan, modified amber.
func glyphStyle(kind gitGlyphKind) lipgloss.Style {
	switch kind {
	case gitGlyphAdded, gitGlyphUntracked:
		return chrome.PanelOK
	case gitGlyphDeleted:
		return chrome.PanelErr
	case gitGlyphRenamed:
		return chrome.PanelInfo
	default:
		return chrome.PanelWarn
	}
}

// classifyDiffLine — the style class of one raw diff line. Meta prefixes
// win over the +/-/@@ content prefixes ("---"/"+++" are file headers, not
// deletions/additions).
func classifyDiffLine(line string) gitLineClass {
	switch {
	case strings.HasPrefix(line, "diff --git "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "---"),
		strings.HasPrefix(line, "+++"):
		return gitLineMeta
	case strings.HasPrefix(line, "@@"):
		return gitLineHunk
	case strings.HasPrefix(line, "+"):
		return gitLineAdd
	case strings.HasPrefix(line, "-"):
		return gitLineDel
	default:
		return gitLinePlain
	}
}

// styleDiffLine colors one (already width-clipped) diff line per its class.
func styleDiffLine(line string) string {
	switch classifyDiffLine(line) {
	case gitLineAdd:
		return chrome.PanelOK.Render(line)
	case gitLineDel:
		return chrome.PanelErr.Render(line)
	case gitLineHunk:
		return chrome.PanelInfo.Render(line)
	case gitLineMeta:
		return chrome.PanelDim.Render(line)
	default:
		return line
	}
}

// middleClip truncates s to w columns by keeping both ends and eliding the
// middle with "…" (long repo paths stay recognizable at both ends).
func middleClip(s string, w int) string {
	r := []rune(s)
	if w <= 0 {
		return ""
	}
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	head := (w - 1 + 1) / 2 // ceil so the odd cell favors the head
	tail := w - 1 - head
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}
