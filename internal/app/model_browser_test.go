// model_browser_test.go — the open-in-OS-browser hotkey's APP seam:
//
//	(a) the `o` claim rule: claims ONLY with a live transcript mark over a
//	    verified-target bubble + the chat tab focused + no floating modal;
//	    otherwise the key falls through — WITHOUT a mark, over a plain
//	    bubble, or while a permission float is up, "o" types into the
//	    draft untouched (typing is safe);
//	(b) a SINGLE target fires straight through the verdict seam (no card)
//	    and logs the activity tab's "→ opened: <name>";
//	(c) MULTIPLE targets float the target card; while the card is open `o`
//	    itself belongs to the card (the re-open guard never resets a
//	    browsed cursor), enter picks the cursor's target, esc cancels with
//	    the runner untouched — and the card's esc outranks the mark-clear
//	    (the float stack's layering: card first, then the mark);
//	(d) the exec VERDICT is the visible artifact: a runner error posts the
//	    dim transcript row "could not open: <reason>" and logs NO "→
//	    opened:" line — a browser hiccup is never fatal;
//	(e) the exec seam is STUBBED through SetOpenRunnerForShot (+Cleanup)
//	    for the whole suite: the real `open -g` would push a browser at
//	    the USER's screen mid-suite — capture is the contract.
package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// browserRig — the shared harness: a model seeded with events, the exec
// seam captured. The mark rides SelectionBegin directly (the real mouse
// plumbing is proven app-side in selection/term tests; this suite owns
// the CLAIM + verdict, one level down the stack).
type browserRig struct {
	m      Model
	opened []panels.LinkTarget
}

func newBrowserRig(t *testing.T, evs ...state.Event) *browserRig {
	t.Helper()
	h := &browserRig{}
	restore := panels.SetOpenRunnerForShot(func(tgt panels.LinkTarget) error {
		h.opened = append(h.opened, tgt)
		return nil
	})
	t.Cleanup(restore)
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 130, Height: 32})
	for _, ev := range evs {
		m = runMsg(t, m, ev)
	}
	h.m = m
	return h
}

// bossLinkEv — a COMPLETED boss bubble (the pinned transcript shape).
func bossLinkEv(id, text string) state.Event {
	return state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: id, From: "boss", Kind: "boss", Text: text, At: 10, Pending: false}}
}

// key feeds ONE key through the real Update; runMsg drains the returned
// cmd tree, so the open's verdict lands synchronously inside the press.
func (h *browserRig) key(t *testing.T, code rune, text string) {
	t.Helper()
	h.m = runMsg(t, h.m, tea.KeyPressMsg(tea.Key{Code: code, Text: text}))
}

// mark arms a one-cell transcript selection on the seeded bubble (a press
// with no release — the armed mark resolves the bubble all the same and
// the clipboard seam never runs in a test).
func (h *browserRig) mark(t *testing.T) {
	t.Helper()
	if h.m.chat == nil {
		t.Fatal("chat panel exists")
	}
	if !h.m.chat.SelectionBegin(4, 0) {
		t.Fatal("a press on the transcript's first row arms the mark")
	}
	if !h.m.chat.SelectionActive() {
		t.Fatal("the armed mark reports active")
	}
}

func (h *browserRig) openedValues() []string {
	var vs []string
	for _, tgt := range h.opened {
		vs = append(vs, tgt.Value)
	}
	return vs
}

// TestBrowserClaimRule — the gate's full matrix: no mark ⇒ types "o";
// a mark over a PLAIN bubble ⇒ types "o"; a permission float up ⇒ the
// modal outranks (no open, no card).
func TestBrowserClaimRule(t *testing.T) {
	// no mark at all: the key types into the draft.
	h := newBrowserRig(t, bossLinkEv("bossmsg-b1", "see https://opencode.ai/docs"))
	h.key(t, 'o', "o")
	if len(h.opened) != 0 || h.m.chat.LinkPickerOpen() {
		t.Fatalf("without a mark `o` is a draft key — nothing opens: %v", h.opened)
	}
	if f := ansi.Strip(h.m.Frame()); !strings.Contains(f, "› o") {
		t.Fatalf("the unclaimed `o` typed into the draft:\n%s", f)
	}

	// a mark over a PLAIN (target-free) bubble: same fall-through.
	h2 := newBrowserRig(t, bossLinkEv("bossmsg-b2", "plain words only"))
	h2.mark(t)
	h2.key(t, 'o', "o")
	if len(h2.opened) != 0 || h2.m.chat.LinkPickerOpen() {
		t.Fatalf("a plain bubble's mark holds no targets — `o` types: %v", h2.opened)
	}

	// a permission float parks the turn: browsing yields to the modal.
	h3 := newBrowserRig(t, bossLinkEv("bossmsg-b3", "see https://opencode.ai/docs"))
	h3.mark(t)
	h3.m = runMsg(t, h3.m, state.Event{Kind: state.EvPermission, PermissionID: "perm-1",
		EmployeeID: "boss", EmployeeName: "boss", ToolName: "Write",
		ToolSummary: "/tmp/x", ToolState: "pending"})
	h3.key(t, 'o', "o")
	if len(h3.opened) != 0 || h3.m.chat.LinkPickerOpen() {
		t.Fatalf("the permission modal outranks `o`: %v", h3.opened)
	}
}

// TestBrowserSingleTargetOpens — the happy path: mark the bubble, press
// `o`, the runner fires ONCE with the URL, NO card floats, and the
// activity tab logs "→ opened: opencode.ai/docs".
func TestBrowserSingleTargetOpens(t *testing.T) {
	h := newBrowserRig(t, bossLinkEv("bossmsg-b1", "spec: https://opencode.ai/docs — read it"))
	h.mark(t)
	h.key(t, 'o', "o")
	if h.m.chat.LinkPickerOpen() {
		t.Fatal("a single target never floats the card")
	}
	if len(h.opened) != 1 || h.opened[0].Value != "https://opencode.ai/docs" || h.opened[0].Kind != panels.LinkURL {
		t.Fatalf("the URL fired exactly once: %+v", h.opened)
	}
	lines := h.m.ActivityLines()
	var openedLine string
	for _, ln := range lines {
		if strings.Contains(ln, "→ opened:") {
			openedLine = ln
		}
	}
	if !strings.Contains(openedLine, "→ opened: opencode.ai/docs") {
		t.Fatalf("the activity tab logs the opened line, got %q in %v", openedLine, lines)
	}
	// after the open the mark STAYS (esc-lawful layering: esc owns it next).
	if !h.m.chat.SelectionActive() {
		t.Fatal("the mark survives the open — esc clears it, not `o`")
	}
}

// TestBrowserFileTargetOpens — a prose FILE path that verifies opens the
// ABSOLUTE resolved path (never the raw token).
func TestBrowserFileTargetOpens(t *testing.T) {
	f := filepath.Join(t.TempDir(), "trace.log")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newBrowserRig(t, bossLinkEv("bossmsg-b1", "log saved to "+f+" now"))
	h.mark(t)
	h.key(t, 'o', "o")
	if len(h.opened) != 1 || h.opened[0].Kind != panels.LinkFile || h.opened[0].Value != f {
		t.Fatalf("the verified file opened absolute: %+v", h.opened)
	}
}

// TestBrowserMultiTargetCard — MULTIPLE targets float the card: `o` while
// the card is open belongs to the card (no cursor reset — the walked row
// survives to enter), enter picks the cursor's target, the card closes,
// and the opened line logs the PICK's name.
func TestBrowserMultiTargetCard(t *testing.T) {
	img := filepath.Join(t.TempDir(), "shot.png")
	if err := os.WriteFile(img, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	it := state.MediaItem{Mime: "image/png", Filename: img}
	h := newBrowserRig(t, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-b1", From: "boss", Kind: "boss",
		Text: "spec https://opencode.ai/docs done", At: 10, Pending: false,
		Meta: state.MediaMeta([]state.MediaItem{it})}, Media: []state.MediaItem{it}})
	h.mark(t)
	h.key(t, 'o', "o")
	if !h.m.chat.LinkPickerOpen() {
		t.Fatal("multiple targets float the target card")
	}
	if len(h.opened) != 0 {
		t.Fatalf("opening the card fires NOTHING yet: %v", h.opened)
	}
	// walk to the media row (row 2) and press `o` — the CARD owns the key:
	// the cursor must NOT reset (the re-open guard).
	h.key(t, tea.KeyDown, "")
	h.key(t, 'o', "o")
	h.key(t, tea.KeyEnter, "")
	if len(h.opened) != 1 || h.opened[0].Value != img || !h.opened[0].Media {
		t.Fatalf("enter picked the WALKED row (the media target), not a reset row 1: %+v", h.opened)
	}
	if h.m.chat.LinkPickerOpen() {
		t.Fatal("the card closed on the pick landing")
	}
	var openedLine string
	for _, ln := range h.m.ActivityLines() {
		if strings.Contains(ln, "→ opened:") {
			openedLine = ln
		}
	}
	if !strings.Contains(openedLine, "→ opened: shot.png") {
		t.Fatalf("the activity line names the pick: %q", openedLine)
	}
}

// TestBrowserCardEscLayers — esc while the card is open closes the CARD
// first; the mark underneath clears on the NEXT esc (the float stack's
// ordering: card over mark).
func TestBrowserCardEscLayers(t *testing.T) {
	h := newBrowserRig(t, bossLinkEv("bossmsg-b1", "see https://opencode.ai/docs and https://example.com/x"))
	h.mark(t)
	h.key(t, 'o', "o")
	if !h.m.chat.LinkPickerOpen() {
		t.Fatal("two URL targets float the card")
	}
	h.key(t, tea.KeyEscape, "")
	if h.m.chat.LinkPickerOpen() {
		t.Fatal("first esc closed the card")
	}
	if !h.m.chat.SelectionActive() {
		t.Fatal("the mark SURVIVED the card's esc (the layered claim)")
	}
	if len(h.opened) != 0 {
		t.Fatalf("esc opened nothing: %v", h.opened)
	}
	h.key(t, tea.KeyEscape, "")
	if h.m.chat.SelectionActive() {
		t.Fatal("the second esc cleared the mark underneath")
	}
}

// TestBrowserOpenVerdictError — a runner error posts the dim transcript
// row "could not open: <reason>" and logs NO "→ opened:" (never fatal).
func TestBrowserOpenVerdictError(t *testing.T) {
	h := newBrowserRig(t, bossLinkEv("bossmsg-b1", "see https://opencode.ai/docs"))
	restore2 := panels.SetOpenRunnerForShot(func(panels.LinkTarget) error {
		return errors.New("xdg-open: boom")
	})
	t.Cleanup(restore2)
	h.mark(t)
	h.key(t, 'o', "o")
	if strings.Contains(strings.Join(h.m.ActivityLines(), "\n"), "→ opened:") {
		t.Fatal("a failed open logs NO opened line")
	}
	if f := ansi.Strip(h.m.Frame()); !strings.Contains(f, "could not open: xdg-open: boom") {
		t.Fatalf("the dim transcript row carries the verdict:\n%s", f)
	}
}
