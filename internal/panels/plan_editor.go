// plan_editor.go — the full-pane markdown PLAN workspace hosted in the app's
// floor region: one header row, a body, one footer hint row.
//
// WHY this exists: plan mode is CONVERSATION-FIRST — the member chats with
// the boss; a completed boss reply is mirrored INTO this pane by the app
// (passive presentation, chat keeps focus). The pane is also the optional
// scratch surface: a click inside it arms the starter template and focuses
// raw editing. The pane deliberately has ZERO app/backend knowledge: the
// app drives Focus/Blur/SetSize/SetMode/ArmStarter, routes keys, and owns
// the presentation policy; everything else (editing, rendering, mermaid
// awareness, glamour memoization, the userDirty anti-clobber latch) lives
// behind the PlanEditor contract.
//
// Body composition rule (spec): FOCUSED + plan-mode shows the live textarea
// (raw editing); UNFOCUSED or build-mode shows the READ-ONLY glamour render
// of Value() with the mermaid caption pass — build mode additionally hides
// all textarea chrome and ignores keys entirely (the plan is approved).
//
// PERF: glamour is by far the most expensive render in this package, so the
// read-only path memoizes on (valHash, width, mode, themeName) — editing
// keystrokes NEVER touch glamour (the focused path doesn't call it), and a
// blur/refocus cycle with an unchanged buffer re-renders nothing.
package panels

import (
	"fmt"
	"hash/fnv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
)

// PlanEditor modes (the string contract the app flips between).
const (
	planModePlan  = "plan"
	planModeBuild = "build"
)

// mermaidCaption is the SINGLE marker line injected above every top-level
// ```mermaid fence in the read-only render. WHY only a caption: there is no
// mermaid CLI (mmdc) on the render path in v1, so diagram slots get a visible
// textual marker instead of a raster — and the STORED buffer never carries
// it (the caption pass works on a copy; Value() round-trips byte-identical).
const mermaidCaption = "╭─ mermaid diagram ─╮"

// starterPlan — the new-pane buffer (~10 lines): goal/context/steps
// scaffolding plus ONE example ```mermaid block, so the diagram lane (and a
// non-zero header count) is visible the moment the pane opens.
const starterPlan = `# Goal
<what this plan achieves>

# Context
<files, constraints, decisions so far>

# Steps
1. ` + "`<first step>`" + `
2. ` + "`<second step>`" + `

` + "```mermaid\nflowchart LR\n    plan --> build\n```"

// PlanEditor is the floor-region plan pane. Field discipline: `ta` is the
// ONLY source of truth for the buffer (Value() is a byte-identical round
// trip); `rv` is the read-only viewport that shows the glamour render; the
// memo trio (mdKey/mdRender/renderCount) guarantees glamour runs at most
// once per (buffer, width, mode, theme) change. userDirty is the
// anti-clobber latch: any buffer mutation delivered through Update (a real
// keystroke edit, never an app-side SetValue) latches it; the app reads it
// before adopting a fresh boss reply and clears it on each boss-set
// adoption and on approve.
type PlanEditor struct {
	ta        textarea.Model
	rv        viewport.Model
	w, h      int
	focused   bool
	mode      string
	userDirty bool

	// taTheme tracks the chrome theme the textarea styles were last applied
	// against, so a /theme switch re-points them on the next frame without
	// the app having to call anything (styles read live chrome vars).
	taTheme string

	// mdKey is "(valHash|width|mode|themeName)" of the last glamour render;
	// mdRender is its cached output; renderCount counts real glamour runs
	// (the memo proof probe — tests assert it does NOT move per frame).
	mdKey       string
	mdRender    string
	renderCount int
}

// NewPlanEditor builds the pane: UNFOCUSED, plan mode, EMPTY buffer — the
// conversation-first default (the app keeps the pane hidden until the boss
// presents a reply, the user edits, or a persisted plan restores; the
// starter scaffold arms on demand via ArmStarter). The host is expected to
// SetSize before the first frame; a sane default keeps pre-SetSize Views
// renderable.
func NewPlanEditor() *PlanEditor {
	ta := textarea.New()
	ta.Prompt = "› " // the house editing gutter (mirrors chat)
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	applyTextareaStyles(&ta) // reads the live chrome palette

	rv := viewport.New(viewport.WithWidth(1), viewport.WithHeight(1))
	// glamour already word-wraps at the pane width; SoftWrap is the guard
	// rail for any row that still overflows (tables, long code lines).
	rv.SoftWrap = true

	e := &PlanEditor{ta: ta, rv: rv, mode: planModePlan, taTheme: chrome.CurrentTheme().Name}
	e.SetSize(30, 10)
	return e
}

// ArmStarter loads the starter template into an EMPTY buffer — the app's
// manual-open path (a click inside the floor slot gives a blank pane the
// scratch scaffold). A non-empty buffer is NEVER clobbered (a presented
// plan wins over boilerplate). Returns true when it armed.
func (e *PlanEditor) ArmStarter() bool {
	if strings.TrimSpace(e.ta.Value()) != "" {
		return false
	}
	e.ta.SetValue(starterPlan)
	return true
}

// IsStarter reports the buffer IS the untouched starter template (the
// approve refusal: approving boilerplate would spend a whole build turn on
// nothing).
func (e *PlanEditor) IsStarter() bool { return e.ta.Value() == starterPlan }

// UserDirty reports whether the user has edited anything since the last
// boss-set refresh adoption / approve reset (the anti-clobber latch).
func (e *PlanEditor) UserDirty() bool { return e.userDirty }

// SetUserDirty is the latch's write side — the app clears it when a boss
// reply is adopted into the pane and on approve.
func (e *PlanEditor) SetUserDirty(d bool) { e.userDirty = d }

// Update forwards input to the inner textarea ONLY when focused AND editable
// (plan mode). In build mode the plan is approved → keys are fully ignored;
// unfocused the buffer must not move at all. A keystroke that actually
// changes the buffer latches userDirty (app-side SetValue never does).
func (e *PlanEditor) Update(msg tea.Msg) tea.Cmd {
	if !e.focused || e.mode == planModeBuild {
		return nil
	}
	before := e.ta.Value()
	var cmd tea.Cmd
	e.ta, cmd = e.ta.Update(msg)
	if e.ta.Value() != before {
		e.userDirty = true
	}
	return cmd
}

// View renders the full pane at the last SetSize dims: header + body +
// footer. Degenerate dims never panic — the header still renders (unclipped
// below width 1) so there's always ≥1 non-empty line.
func (e *PlanEditor) View() string {
	// theme switched under us? re-point the textarea chrome before drawing
	// (reads the live chrome vars; zero cost when the theme is unchanged).
	if theme := chrome.CurrentTheme().Name; theme != e.taTheme {
		applyTextareaStyles(&e.ta)
		e.taTheme = theme
	}

	_, bh := e.bodyDims()
	var body string
	if e.readOnly() {
		e.renderReadOnly() // memoized; refreshes rv content on key change
		body = e.rv.View()
	} else {
		body = e.ta.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, e.header(), fit(body, bh), e.footer())
}

// SetSize stores the pane dims and sizes the textarea + read-only viewport.
// Idempotent: raw dims in, same derived dims out. Returns e for chaining.
func (e *PlanEditor) SetSize(w, h int) *PlanEditor {
	e.w, e.h = w, h
	bw, bh := e.bodyDims()
	e.ta.SetWidth(bw)
	e.ta.SetHeight(bh)
	e.rv.SetWidth(bw)
	e.rv.SetHeight(bh)
	return e
}

// Focus arms raw editing (delegates the blink cmd to the textarea).
func (e *PlanEditor) Focus() tea.Cmd {
	e.focused = true
	return e.ta.Focus()
}

// Blur disarms editing; the next frame shows the read-only markdown render.
func (e *PlanEditor) Blur() {
	e.focused = false
	e.ta.Blur()
}

// Focused reports whether keystrokes reach the textarea.
func (e *PlanEditor) Focused() bool { return e.focused }

// Value is the RAW buffer — byte-identical round-trip, NEVER the captioned
// or rendered copy.
func (e *PlanEditor) Value() string { return e.ta.Value() }

// SetValue replaces the whole buffer (app-side plan load/import).
func (e *PlanEditor) SetValue(s string) { e.ta.SetValue(s) }

// Mode is "plan" or "build"; it drives the header label + color and the
// read-only-locked build body.
func (e *PlanEditor) Mode() string { return e.mode }

// SetMode flips plan/build. Unknown values coalesce to plan (the editable,
// default lane) so a stray app value can never lock the editor.
func (e *PlanEditor) SetMode(m string) {
	if m != planModeBuild {
		m = planModePlan
	}
	e.mode = m
}

// readOnly — the body renders markdown instead of the textarea: build mode
// always, plan mode only while unfocused.
func (e *PlanEditor) readOnly() bool { return e.mode == planModeBuild || !e.focused }

// bodyDims clamps the pane dims into a usable body rectangle: the pane
// reserves 1 header row + 1 footer row and never hands a widget <1 cell.
func (e *PlanEditor) bodyDims() (w, h int) {
	w, h = e.w, e.h-2
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

// header renders the one-line house-topbar-style row: mode label (colored —
// accent for PLAN, ok green for BUILD) + "markdown (N mermaid diagram[s])"
// on the left, the key hints dim on the right, clipped to the pane width.
func (e *PlanEditor) header() string {
	label, labelStyle := "PLAN", chrome.AccentText
	if e.mode == planModeBuild {
		label, labelStyle = "BUILD", lipgloss.NewStyle().Foreground(chrome.OK)
	}
	n := mermaidCount(e.Value())
	diagrams := fmt.Sprintf("%d mermaid diagram", n)
	if n != 1 {
		diagrams += "s"
	}
	left := labelStyle.Render(label) + chrome.DimText.Render(" · markdown ") +
		chrome.DimText.Render("("+diagrams+")")
	right := chrome.DimText.Render("ctrl+x approve → build · ctrl+p exits")
	gap := e.w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	if e.w >= 1 && lipgloss.Width(line) > e.w {
		line = ansi.Truncate(line, e.w, "")
	}
	return line
}

// footer renders the one-line dim hint row for the current mode: the
// edit-focused hint while the textarea owns keys, the click-to-edit surface
// hint while the pane sits passive (chat keeps typing), the build lane's
// exit pointer.
func (e *PlanEditor) footer() string {
	hint := "enter: newline · esc: done editing"
	switch {
	case e.mode == planModeBuild:
		hint = "ctrl+p back to plan"
	case !e.focused:
		hint = "click to edit · ctrl+x approve → build · ctrl+p exits"
	}
	s := chrome.DimText.Render(hint)
	if e.w >= 1 && lipgloss.Width(s) > e.w {
		s = ansi.Truncate(s, e.w, "")
	}
	return s
}

// renderReadOnly fills the read-only viewport with the memoized glamour
// render of the MERMAID-CAPTIONED buffer, scrolled to top. The glamour call
// — the expensive step — runs only when (buffer, width, mode, theme) moves.
func (e *PlanEditor) renderReadOnly() {
	key := e.renderKey()
	if key == e.mdKey {
		return
	}
	e.renderCount++
	src := mermaidCaptioned(e.Value())
	e.mdRender = strings.Join(cleanMarkdown(e.renderGlamour(src)), "\n")
	e.mdKey = key
	e.rv.SetContent(e.mdRender)
	e.rv.GotoTop()
}

// renderKey is the memo key: (value hash, pane width, mode, theme name).
func (e *PlanEditor) renderKey() string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(e.Value()))
	return fmt.Sprintf("%016x|%d|%s|%s", h.Sum64(), e.w, e.mode, chrome.CurrentTheme().Name)
}

// renderGlamour runs src through glamour at the body width with the active
// chrome markdown style. Building the renderer per call is deliberate: this
// only runs on a memo MISS, so there is no stale-renderer state to keep.
func (e *PlanEditor) renderGlamour(src string) string {
	w, _ := e.bodyDims()
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(chrome.MarkdownStyle()),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return src
	}
	out, err := r.Render(src)
	if err != nil {
		return src
	}
	return out
}

// --- mermaid awareness (pure; NEVER touches the stored buffer) -------------

// fenceMark is a CommonMark fenced-code marker: a left-trimmed run of ≥3
// backticks or tildes plus its (possibly empty) info string. Closers are the
// same mark with EMPTY info — so a ```mermaid line INSIDE an open fence can
// never close it and is never mistaken for a top-level opener.
type fenceMark struct {
	char   byte // '`' or '~'
	length int  // run length
	info   string
}

// scanFence classifies one line; returns ok=false for non-fence lines.
func scanFence(line string) (fenceMark, bool) {
	l := strings.TrimLeft(line, " \t")
	if len(l) < 3 {
		return fenceMark{}, false
	}
	c := l[0]
	if c != '`' && c != '~' {
		return fenceMark{}, false
	}
	n := 0
	for n < len(l) && l[n] == c {
		n++
	}
	if n < 3 {
		return fenceMark{}, false
	}
	return fenceMark{char: c, length: n, info: strings.TrimSpace(l[n:])}, true
}

// mermaidOpeners returns the line indexes of every TOP-LEVEL ```mermaid
// fence opener — i.e. openers not inside another fence (nested mermaid
// fences inside a plain fenced block are invisible here, which is exactly
// what both the caption pass and the header count want).
func mermaidOpeners(src string) map[int]bool {
	opens := map[int]bool{}
	var open fenceMark // valid only while inFence
	inFence := false
	for i, ln := range strings.Split(src, "\n") {
		m, ok := scanFence(ln)
		if !ok {
			continue
		}
		if !inFence {
			if m.info == "mermaid" {
				opens[i] = true
			}
			inFence = true
			open = m
			continue
		}
		// inside a fence: only a same-char, ≥length, no-info run closes it
		if m.char == open.char && m.length >= open.length && m.info == "" {
			inFence = false
		}
	}
	return opens
}

// mermaidCaptioned returns a COPY of src with mermaidCaption injected
// directly above every top-level ```mermaid fence. Pure: src is never
// mutated, and with zero mermaid fences src returns unchanged.
func mermaidCaptioned(src string) string {
	opens := mermaidOpeners(src)
	if len(opens) == 0 {
		return src
	}
	lines := strings.Split(src, "\n")
	var b strings.Builder
	b.Grow(len(src) + len(opens)*(len(mermaidCaption)+1))
	for i, ln := range lines {
		if opens[i] {
			b.WriteString(mermaidCaption)
			b.WriteByte('\n')
		}
		b.WriteString(ln)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// mermaidCount counts top-level ```mermaid fence openers (the header's
// "(N mermaid diagram[s])").
func mermaidCount(src string) int { return len(mermaidOpeners(src)) }
