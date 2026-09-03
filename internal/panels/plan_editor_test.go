// plan_editor_test.go — the PlanEditor contract: the empty conversation-first
// rest state and the on-demand starter scaffold (ArmStarter / IsStarter),
// the userDirty anti-clobber latch, the pure mermaid caption/count pass
// (nested fences stay invisible), focus gating of keystrokes, SetSize clamps,
// the focused-vs-read-only body split, glamour memoization, and
// theme-switch safety.
package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
)

// typeRun sends each rune of s through the pane as a real key press.
func typeRun(e *PlanEditor, s string) {
	for _, r := range s {
		e.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
}

// frame renders a t.Logf dump box so -v output shows the real pane layout
// (ANSI-stripped: composition is what's being proven, not the palette).
func frame(t *testing.T, title, view string) {
	t.Helper()
	t.Logf("--- %s ---\n%s\n--- /%s ---", title, ansi.Strip(view), title)
}

func TestPlanEditorRestStateAndStarter(t *testing.T) {
	e := NewPlanEditor()
	// the conversation-first rest state: EMPTY buffer, unfocused, plan mode
	if v := e.Value(); strings.TrimSpace(v) != "" {
		t.Fatalf("NewPlanEditor must start EMPTY (the app hides it until content), got %q", v)
	}
	if e.IsStarter() {
		t.Fatalf("an empty buffer is NOT the untouched template (templates report starter-only)")
	}
	if e.Focused() {
		t.Fatalf("NewPlanEditor must be unfocused initially")
	}
	if e.Mode() != "plan" {
		t.Fatalf("NewPlanEditor must start in plan mode, got %q", e.Mode())
	}
	if e.UserDirty() {
		t.Fatalf("a fresh pane carries no user edits")
	}

	// ArmStarter — the app's manual-open path: the scaffold loads, the pane
	// reports starter (the approve refusal), and the arm is NOT a user edit
	if !e.ArmStarter() {
		t.Fatalf("ArmStarter must arm into an empty buffer")
	}
	if v := e.Value(); strings.TrimSpace(v) == "" {
		t.Fatalf("starter template must be non-empty")
	}
	if got := mermaidCount(e.Value()); got < 1 {
		t.Fatalf("starter template must contain ≥1 mermaid fence, got %d", got)
	}
	if !e.IsStarter() {
		t.Fatalf("freshly-armed buffer must report IsStarter (untouched template)")
	}
	if e.UserDirty() {
		t.Fatalf("arming the template is an app-side load, not a user edit")
	}

	// ArmStarter NEVER clobbers a non-empty buffer (a presented plan wins)
	if e.ArmStarter() {
		t.Fatalf("ArmStarter must refuse a non-empty buffer")
	}

	// any actual edit drops the starter report
	e.SetValue("# custom")
	if e.IsStarter() {
		t.Fatalf("an edited buffer must NOT report IsStarter")
	}
}

func TestMermaidCaptionPass(t *testing.T) {
	// (a) one top-level fence → caption sits directly ABOVE the fence line
	src := "# plan\n\n```mermaid\nflowchart LR\n    a --> b\n```\n"
	out := mermaidCaptioned(src)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	cIdx, fenceIdx := -1, -1
	for i, l := range lines {
		if l == mermaidCaption {
			cIdx = i
		}
		if strings.HasPrefix(l, "```mermaid") && fenceIdx == -1 {
			fenceIdx = i
		}
	}
	if cIdx == -1 || fenceIdx == -1 || cIdx != fenceIdx-1 {
		t.Fatalf("caption must sit directly above the fence, got lines %q (cIdx=%d fenceIdx=%d)", lines, cIdx, fenceIdx)
	}

	// (b) a ```mermaid line NESTED inside a plain fenced block is content,
	// not an opener — no caption, no count
	nested := "```\nsome code\n```mermaid\nflowchart LR\n a-->b\n```\n"
	if got := mermaidCaptioned(nested); strings.Contains(got, mermaidCaption) {
		t.Fatalf("nested mermaid inside a plain fence must NOT be captioned, got:\n%s", got)
	}
	if got := mermaidCount(nested); got != 0 {
		t.Fatalf("nested mermaid must not count, got %d", got)
	}

	// (c) the counter for 0 / 1 / 3 (the 3-fence case also interleaves a
	// plain fence to prove it doesn't swallow the following mermaid opener)
	if got := mermaidCount("no fences here\n"); got != 0 {
		t.Fatalf("0-fence count, got %d", got)
	}
	if got := mermaidCount(src); got != 1 {
		t.Fatalf("1-fence count, got %d", got)
	}
	three := "```mermaid\na-->b\n```\nmid\n```go\nfmt.Println(1)\n```\nmore\n```mermaid\nc-->d\n```\n\n```mermaid\ne-->f\n```\n"
	if got := mermaidCount(three); got != 3 {
		t.Fatalf("3-fence count, got %d", got)
	}
	// every top-level opener in the 3-fence doc gets its own caption
	if n := strings.Count(mermaidCaptioned(three), mermaidCaption); n != 3 {
		t.Fatalf("3 top-level fences → 3 captions, got %d", n)
	}

	// (d) Value() is byte-identical after the caption pass (the captioned
	// string is a display copy, never stored)
	e := NewPlanEditor()
	e.ArmStarter()
	before := e.Value()
	_ = mermaidCaptioned(before)
	e.Blur()
	_ = e.View() // runs the full read-only render path (captioned internally)
	if got := e.Value(); got != before {
		t.Fatalf("Value() must round-trip byte-identical:\n want %q\n got  %q", before, got)
	}
}

// TestPlanEditorUserDirtyLatch pins the anti-clobber latch: app-side
// SetValue (boss presentation, hydrate, approve reset) never dirties; a
// real keystroke edit does; the app's SetUserDirty clears it.
func TestPlanEditorUserDirtyLatch(t *testing.T) {
	e := NewPlanEditor()
	if e.UserDirty() {
		t.Fatalf("a fresh pane is clean")
	}
	e.SetValue("# boss's presented plan") // app-side load — never a user edit
	if e.UserDirty() {
		t.Fatalf("app-side SetValue must NOT latch userDirty")
	}

	// keystroke edits while unfocused/build keep the latch clean
	typeRun(e, "zzz")
	if e.UserDirty() {
		t.Fatalf("unfocused keys never reach the buffer — no dirty")
	}

	// a REAL edit latches; a buffer-preserving cursor key does not
	e.Focus()
	typeRun(e, "x")
	if !e.UserDirty() {
		t.Fatalf("a keystroke buffer change must latch userDirty")
	}
	t.Logf("user edit latched userDirty: buffer=%q dirty=%t", e.Value(), e.UserDirty())

	e.SetUserDirty(false) // the boss-set adoption / approve reset seam
	if e.UserDirty() {
		t.Fatalf("SetUserDirty(false) must clear the latch")
	}
	e.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft})) // cursor move only
	if e.UserDirty() {
		t.Fatalf("a buffer-preserving keystroke must not re-dirty")
	}
}

func TestPlanEditorFocusGating(t *testing.T) {
	e := NewPlanEditor()
	before := e.Value()
	typeRun(e, "zzz")
	if got := e.Value(); got != before {
		t.Fatalf("unfocused Update must be a no-op on the buffer:\n was %q\n now %q", before, got)
	}
	if cmd := e.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})); cmd != nil {
		t.Fatalf("unfocused Update must return nil cmd (nothing forwarded)")
	}

	e.Focus()
	if !e.Focused() {
		t.Fatalf("Focus() must arm the pane")
	}
	typeRun(e, "zzz")
	if got := e.Value(); !strings.Contains(got, "zzz") {
		t.Fatalf("focused Update must reach the textarea, buffer %q lacks zzz", got)
	}

	e.Blur()
	armed := e.Value()
	typeRun(e, "qqq")
	if got := e.Value(); got != armed {
		t.Fatalf("Blurred pane must stop forwarding keys, buffer moved %q → %q", armed, got)
	}
}

func TestPlanEditorSetSizeClamps(t *testing.T) {
	e := NewPlanEditor()
	for _, d := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {5, 3}, {60, 20}, {200, 80}, {0, 0}} {
		e.SetSize(d[0], d[1]) // must not panic
		e.Focus()
		v := e.View() // must not panic
		first := ""
		if ls := strings.Split(v, "\n"); len(ls) > 0 {
			first = ansi.Strip(ls[0])
		}
		if strings.TrimSpace(first) == "" {
			t.Fatalf("SetSize(%d,%d): header must be ≥1 line non-empty, got first line %q of\n%s", d[0], d[1], first, ansi.Strip(v))
		}
		e.Blur()
		v = e.View()
		if ls := strings.Split(v, "\n"); len(ls) == 0 || strings.TrimSpace(ansi.Strip(ls[0])) == "" {
			t.Fatalf("SetSize(%d,%d): read-only header must be ≥1 line non-empty", d[0], d[1])
		}
	}
	// idempotence: same dims in → identical frame out
	e.SetSize(60, 20)
	a := e.View()
	e.SetSize(60, 20)
	if b := e.View(); a != b {
		t.Fatalf("SetSize must be idempotent, frames differ:\n%s\n---\n%s", a, b)
	}
}

func TestPlanEditorFocusedVsReadOnly(t *testing.T) {
	e := NewPlanEditor().SetSize(80, 20)
	e.ArmStarter()

	// focused plan-mode: the live textarea — prompt glyph + raw buffer chars
	e.Focus()
	v := ansi.Strip(e.View())
	if !strings.Contains(v, "›") {
		t.Fatalf("focused View must show the textarea prompt, got:\n%s", v)
	}
	if !strings.Contains(v, "# Goal") {
		t.Fatalf("focused View must show raw buffer chars, got:\n%s", v)
	}
	last := func(s string) string {
		ls := strings.Split(s, "\n")
		return ls[len(ls)-1]
	}
	if !strings.Contains(last(v), "enter: newline · esc: done editing") {
		t.Fatalf("FOCUSED plan-mode footer hint missing, got last line %q", last(v))
	}
	frame(t, "plan FOCUSED (live textarea) 80x20", e.View())

	// unfocused plan-mode: read-only glamour render WITH the caption — the
	// passive presentation surface (the footer advertises the way back in)
	e.Blur()
	v = ansi.Strip(e.View())
	if !strings.Contains(v, mermaidCaption) {
		t.Fatalf("read-only View must contain the mermaid caption, got:\n%s", v)
	}
	if !strings.Contains(v, "PLAN · markdown (1 mermaid diagram)") {
		t.Fatalf("header must label PLAN + the diagram count, got first line %q", strings.Split(v, "\n")[0])
	}
	if !strings.Contains(v, "ctrl+x approve → build · ctrl+p exits") {
		t.Fatalf("header must carry the key hints, got first line %q", strings.Split(v, "\n")[0])
	}
	if !strings.Contains(v, "click to edit · ctrl+x approve → build · ctrl+p exits") {
		t.Fatalf("UNFOCUSED plan-mode footer hint missing, got last line %q", last(v))
	}
	// glamour actually formatted the markdown (heading restyled, so the raw
	// "# Goal" markup is gone from the render even though the words stay)
	if strings.Contains(v, "# Goal") {
		t.Fatalf("read-only body must be RENDERED markdown, raw '# Goal' leaked:\n%s", v)
	}

	// build mode: header flips to BUILD, caption still rendered, keys ignored
	e.SetMode("build")
	if e.Mode() != "build" {
		t.Fatalf("SetMode(build) must stick")
	}
	armed := e.Value()
	typeRun(e, "BOGUS")
	if got := e.Value(); got != armed {
		t.Fatalf("build mode is read-only — typing must be ignored, buffer moved")
	}
	v = ansi.Strip(e.View())
	if !strings.Contains(v, "BUILD · markdown (1 mermaid diagram)") {
		t.Fatalf("build header must label BUILD, got first line %q", strings.Split(v, "\n")[0])
	}
	if !strings.Contains(v, mermaidCaption) {
		t.Fatalf("build read-only render must contain the caption, got:\n%s", v)
	}
	if !strings.Contains(v, "ctrl+p back to plan") {
		t.Fatalf("build footer hint missing, got last line %q", strings.Split(v, "\n")[len(strings.Split(v, "\n"))-1])
	}
	if strings.Contains(v, "# Goal") {
		t.Fatalf("build body must be RENDERED markdown, raw '# Goal' leaked:\n%s", v)
	}

	// narrow width: the right hint is the clip target (spec: dimmer, clipped
	// to width) — header & footer must stay clean one-liners, never overflow
	e.SetSize(45, 20)
	ls := strings.Split(ansi.Strip(e.View()), "\n")
	for _, l := range []string{ls[0], ls[len(ls)-1]} {
		if w := ansi.StringWidth(l); w > 45 {
			t.Fatalf("chrome row overflows w=45 (%d cells): %q", w, l)
		}
	}
}

func TestPlanEditorFrames60x20(t *testing.T) {
	// the contract's proof frames: the pane at the floor-region's 60x20
	e := NewPlanEditor().SetSize(60, 20)
	e.ArmStarter()
	e.Blur()
	frame(t, "plan read-only 60x20", e.View())
	e.SetMode("build")
	frame(t, "build read-only 60x20", e.View())
}

func TestPlanEditorRenderMemoized(t *testing.T) {
	e := NewPlanEditor().SetSize(60, 20) // unfocused → read-only path
	first := e.View()
	second := e.View()
	if first != second {
		t.Fatalf("equal View() calls must return the same string")
	}
	if e.renderCount != 1 {
		t.Fatalf("two equal read-only Views must render glamour ONCE, renderCount=%d", e.renderCount)
	}

	e.SetValue(e.Value() + "\nextra step\n")
	_ = e.View()
	if e.renderCount != 2 {
		t.Fatalf("a buffer change must invalidate the memo, renderCount=%d", e.renderCount)
	}

	// focus flips to the textarea path — which must NEVER call glamour
	e.Focus()
	for i := 0; i < 3; i++ {
		_ = e.View()
	}
	if e.renderCount != 2 {
		t.Fatalf("the focused textarea path must never touch glamour, renderCount=%d", e.renderCount)
	}
	// ... and editing keystrokes in there don't either
	typeRun(e, "ek")
	_ = e.View()
	if e.renderCount != 2 {
		t.Fatalf("editing keystrokes must not re-render glamour, renderCount=%d", e.renderCount)
	}
}

func TestPlanEditorThemeSwitch(t *testing.T) {
	before := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(before) })

	e := NewPlanEditor().SetSize(60, 20)
	e.ArmStarter()
	_ = e.View() // warm the memo under the current theme
	renders := e.renderCount

	other := ""
	for _, n := range chrome.ThemeNames() {
		if n != before {
			other = n
			break
		}
	}
	if other == "" {
		t.Skip("need ≥2 themes to exercise the switch")
	}
	if !chrome.SetTheme(other) {
		t.Fatalf("SetTheme(%q) must succeed", other)
	}
	// read chrome.* at render time: both paths must survive the switch
	e.Focus()
	_ = e.View()
	e.Blur()
	v := ansi.Strip(e.View())
	if !strings.Contains(v, "PLAN · markdown (1 mermaid diagram)") {
		t.Fatalf("post-switch header must still render, got first line %q", strings.Split(v, "\n")[0])
	}
	if e.renderCount != renders+1 {
		t.Fatalf("theme switch must invalidate the glamour memo exactly once, renders %d → %d", renders, e.renderCount)
	}
	frame(t, "theme-switched plan read-only 60x20", e.View())
}
