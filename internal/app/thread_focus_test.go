// thread_focus_test.go — the thread focus view's APP-SIDE binding contract
// (fakes only, real Model + real Chat panel; the pane's own rendering proof
// lives in internal/panels/thread_focus_test.go):
//
//	(a) ctrl+f's resolution chain — "no worker threads yet" on an empty
//	    office, the live-winner on a busy one, the expand-ledger tail
//	    outranking both;
//	(b) esc closes and the main chat returns BYTE-IDENTICAL (scroll
//	    offset + threadExpand + draft text preserved);
//	(c) esc inside the focus NEVER reaches the chat's dbl-esc /stop
//	    tracker or its back-one ledger (both probes);
//	(d) an EvQuestion while open DISMOUNTS the focus (float keeps the
//	    pixels, one-line dim notice left behind);
//	(e) zen outranks focus both ways (Frame branch + the ctrl+f press);
//	(f) the frame digest flips on open AND on close;
//	(g) the deferred render: while open the main chat does NOT re-render
//	    (renderConversation call-count probe), the close's ResumeFromFocus
//	    re-renders EXACTLY once at the current snapshot.
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// tfKey / tfEsc / tfCtrlF — the scripted keys.
func tfCtrlF() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 'f', Mod: tea.ModCtrl}) }
func tfEsc() tea.KeyPressMsg   { return tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}) }
func tfPgUp() tea.KeyPressMsg  { return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}) }

// tfSetup — a sized office carrying TWO worker threads (tekton-1's edit
// rides a per-call wdiff) + a settled boss turn. tekton-1's lines are the
// newest: the live chain resolves to it.
func tfSetup(t *testing.T) (Model, *selBackend) {
	t.Helper()
	b := &selBackend{sessBackend: sessBackend{primary: "ses-threadfocus"}}
	m := selSetupModel(t, b)
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "Wire the SSE stream"}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteWorking, Task: "Scan the repo"}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-tf", From: "user", Kind: "user", Text: "wire the stream", At: 1}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "b-tf", From: "boss", Kind: "boss", Text: "tekton-1 is on it.", At: 2}})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeID: "sco-1", EmployeeName: "skopos-1",
		ToolName: "grep", ToolSummary: "SSE, 3 hits", ToolState: "done", CallID: "call-s1"})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		ToolName: "read", ToolSummary: "internal/room/manager.go", ToolState: "done", CallID: "call-t1"})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		ToolName: "edit", ToolSummary: "internal/room/handler.go", ToolState: "done", CallID: "call-t2"})
	m = runMsg(t, m, state.Event{Kind: state.EvFileDiff, EmployeeName: "tekton-1", SessionID: "ses-kid",
		CallID: "call-t2", DiffPath: "internal/room/handler.go", DiffAdd: 3, DiffDel: 1,
		DiffBody: "--- a/internal/room/handler.go\n+++ b/internal/room/handler.go\n@@ -1 +1,2 @@\n-a\n+wireMarker\n"})
	return m, b
}

// (a) the resolution chain.
func TestThreadFocusResolutionChain(t *testing.T) {
	// (a-1) an empty office: notice, NO pane, the key already handled.
	{
		m := tfSetupEmpty(t)
		m, _ = selUpdate(t, m, tfCtrlF())
		if m.threadFocus != nil {
			t.Fatalf("an empty office must not open a focus")
		}
		found := false
		for _, c := range m.st.Chat {
			if c.From == "office" && strings.Contains(c.Text, "no worker threads yet") {
				found = true
			}
		}
		if !found {
			t.Fatalf("the empty office answers ctrl+f with the dim notice:\n%s", ansi.Strip(m.Frame()))
		}
	}

	// (a-2) two live threads, no hand interaction: the LIVE chain picks
	// the newest-activity live thread (tekton-1).
	m, _ := tfSetup(t)
	m, _ = selUpdate(t, m, tfCtrlF())
	if m.threadFocus == nil || m.focusThread != "tekton-1" {
		t.Fatalf("the live chain must open tekton-1, got %q (pane=%v)", m.focusThread, m.threadFocus != nil)
	}
	fw := ansi.Strip(m.Frame())
	if !strings.Contains(fw, "Developer Task — Wire the SSE stream · 2 tool calls") ||
		!strings.Contains(fw, "esc · ctrl+f back to office") {
		t.Fatalf("the focus frame must carry the header + the leave hint:\n%s", fw)
	}

	// (a-3) the expand-ledger TAIL outranks the live winner: a hand-open
	// on skopos-1 re-parks the ledger, and the next ctrl+f opens IT.
	m, _ = selUpdate(t, m, tfEsc()) // back to the office first
	m.chat.ExpandThread("skopos-1", true)
	m, _ = selUpdate(t, m, tfCtrlF())
	if m.threadFocus == nil || m.focusThread != "skopos-1" {
		t.Fatalf("the ledger tail must outrank the live winner — want skopos-1, got %q", m.focusThread)
	}
}

// tfSetupEmpty — a sized, hired-boss-only office (no worker threads at all).
func tfSetupEmpty(t *testing.T) Model {
	t.Helper()
	m := selSetupModel(t, &selBackend{sessBackend: sessBackend{primary: "ses-threadfocus-none"}})
	return m
}

// (b) esc closes BYTE-IDENTICAL: scroll offset + threadExpand + draft
// text all preserved (the frame compare covers all three — every one of
// them is pixels).
func TestThreadFocusEscRestoresByteIdentical(t *testing.T) {
	m, _ := tfSetup(t)
	// dye the water: expand every thread (baseline), scroll up off the
	// bottom, and leave a live draft in the input.
	m, _ = selUpdate(t, m, tea.KeyPressMsg(tea.Key{Code: 'g', Mod: tea.ModCtrl}))
	for i := 0; i < 4; i++ {
		m, _ = selUpdate(t, m, tfPgUp())
	}
	for _, r := range "draft in flight" {
		m, _ = selUpdate(t, m, pressKey(r))
	}
	pre := m.Frame()
	if !strings.Contains(ansi.Strip(pre), "draft in flight") {
		t.Fatalf("precondition: the draft is visible in the frame:\n%s", ansi.Strip(pre))
	}

	m, _ = selUpdate(t, m, tfCtrlF())
	if m.threadFocus == nil {
		t.Fatalf("precondition: the focus opened")
	}
	focusFrame := m.Frame()
	if focusFrame == pre {
		t.Fatalf("the focus must REPLACE the frame (fullscreen nested panel)")
	}
	// interact INSIDE the pane: scroll its own viewport + "type" into its
	// hidden clone input — none of it may leak back to the office.
	m, _ = selUpdate(t, m, tfPgUp())
	m, _ = selUpdate(t, m, pressKey('x'))

	m, _ = selUpdate(t, m, tfEsc())
	if m.threadFocus != nil {
		t.Fatalf("esc must close the focus")
	}
	if post := m.Frame(); post != pre {
		t.Fatalf("esc must return the office BYTE-IDENTICAL (scroll offset / threadExpand / draft drifted)\n--- before ---\n%s\n--- after ---\n%s", pre, post)
	}
	if m.focusDeferredRender {
		t.Fatalf("the deferred render saver must clear on close")
	}
}

// (c) the focus's esc is CONSUMED — it never reaches the chat panel's
// dbl-esc stop tracker, nor its back-one thread ledger.
func TestThreadFocusEscNeverTouchesChatSeams(t *testing.T) {
	// (c-1) the dbl-esc tracker: one esc INSIDE the focus + one real esc
	// afterwards must NOT pair (the focus's press never stamped the
	// opener); the control press right after still fires the pair.
	m, b := tfSetup(t)
	m, _ = selUpdate(t, m, tfCtrlF())
	m, _ = selUpdate(t, m, tfEsc()) // the focus's press — consumed entirely
	m, cmd := selUpdate(t, m, tfEsc())
	if selHasStopLeaf(cmd) || b.aborts != 0 {
		t.Fatalf("the focus's esc stamped the dbl-esc opener — the lone main-chat esc paired with it and fired /stop")
	}
	m, cmd = selUpdate(t, m, tfEsc()) // the REAL pair: opener + completer
	if !selHasStopLeaf(cmd) {
		t.Fatalf("the dbl-esc tracker must stay live underneath (esc-esc fires /stop)")
	}

	// (c-2) the back-one ledger: a hand-expanded thread must still be
	// expanded after the focus's esc (a leaked esc would have folded).
	m2, _ := tfSetup(t)
	m2.chat.ExpandThread("tekton-1", true) // the ledger tail
	m2, _ = selUpdate(t, m2, tfCtrlF())    // resolves to the ledger tail
	m2, _ = selUpdate(t, m2, tfEsc())      // consumed by the focus
	if got := ansi.Strip(m2.Frame()); !strings.Contains(got, "↳ diff · internal/room/handler.go") {
		t.Fatalf("the focus's esc folded the main chat's expanded thread (the ledger moved):\n%s", got)
	}
}

// (d) EvQuestion while open: the float keeps the pixels — the pane
// dismounts and a one-line dim notice records the swap.
func TestThreadFocusQuestionDismounts(t *testing.T) {
	m, _ := tfSetup(t)
	m, _ = selUpdate(t, m, tfCtrlF())
	if m.threadFocus == nil {
		t.Fatalf("precondition: the focus opened")
	}
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, EmployeeName: "boss",
		QuestionID: "q-tf", Text: "which db?", ToolSummary: "postgres | sqlite"})
	if m.threadFocus != nil || m.focusThread != "" {
		t.Fatalf("a question float must dismount the focus (pane=%v name=%q)", m.threadFocus != nil, m.focusThread)
	}
	if m.question == nil || m.permQ.front() != nil {
		t.Fatalf("the question hold must own the float slot (question=%v)", m.question != nil)
	}
	found := ""
	for _, c := range m.st.Chat {
		if c.From == "office" && strings.Contains(c.Text, "thread focus") {
			found = c.Text
		}
	}
	if !strings.Contains(found, "tekton-1") || !strings.Contains(found, "permission/question") {
		t.Fatalf("a one-line dim notice must record the dismount, got %q", found)
	}
	// and the deferred saver is off: the main chat re-renders normally
	if m.focusDeferredRender {
		t.Fatalf("the dismount must clear the render saver")
	}

	// an EvPermission float dismounts identically.
	m2, _ := tfSetup(t)
	m2, _ = selUpdate(t, m2, tfCtrlF())
	if m2.threadFocus == nil {
		t.Fatalf("precondition: the focus opened")
	}
	m2 = runMsg(t, m2, state.Event{Kind: state.EvPermission, EmployeeName: "boss",
		PermissionID: "perm-tf", ToolName: "write", ToolSummary: "handler.go"})
	if m2.threadFocus != nil || m2.permQ.front() == nil {
		t.Fatalf("a permission float must dismount the focus and own the slot (pane=%v front=%v)",
			m2.threadFocus != nil, m2.permQ.front() != nil)
	}
}

// (e) zen outranks focus — both directions.
func TestThreadFocusZenWins(t *testing.T) {
	m, _ := tfSetup(t)
	m, _ = selUpdate(t, m, tfCtrlF())
	if m.threadFocus == nil {
		t.Fatalf("precondition: the focus opened")
	}
	// /zen while focused: the zen branch masks the pane (the focus parks
	// underneath, untouched).
	m = runMsg(t, m, slashMsg{text: "/zen"})
	if !m.zen || m.threadFocus == nil {
		t.Fatalf("zen must mask a parked focus, not kill it (zen=%v focus=%v)", m.zen, m.threadFocus != nil)
	}
	zf := ansi.Strip(m.Frame())
	if !strings.Contains(zf, "any key exits") || strings.Contains(zf, "esc · ctrl+f back to office") {
		t.Fatalf("the zen chrome must own the statusbar over the focus:\n%s", zf)
	}
	// ANY key exits zen — even an esc — and the focus simply resumes.
	m, _ = selUpdate(t, m, tfEsc())
	if m.zen {
		t.Fatalf("any key must exit zen")
	}
	if m.threadFocus == nil {
		t.Fatalf("the zen-exit esc must NOT fall through into the focus (it parks untouched underneath)")
	}
	if ff := ansi.Strip(m.Frame()); !strings.Contains(ff, "esc · ctrl+f back to office") {
		t.Fatalf("after zen exits, the parked focus resumes:\n%s", ff)
	}

	// the other direction: ctrl+f while zen'd (nothing open) is eaten by
	// the zen gate — it EXITS zen and opens nothing on that press.
	m, _ = tfSetup(t)
	m = runMsg(t, m, slashMsg{text: "/zen"})
	if !m.zen || m.threadFocus != nil {
		t.Fatalf("precondition: zen active, focus closed")
	}
	m, _ = selUpdate(t, m, tfCtrlF())
	if m.zen {
		t.Fatalf("any key must exit zen (ctrl+f included)")
	}
	if m.threadFocus != nil {
		t.Fatalf("the zen gate eats the press — ctrl+f must not open the focus on the same key")
	}
}

// (f) the digest flips on open AND on close (and the identical second
// frame while open is a pure cache hit).
func TestThreadFocusDigestFlips(t *testing.T) {
	m, _ := tfSetup(t)
	d0 := m.frameDigest()
	m, _ = selUpdate(t, m, tfCtrlF())
	d1 := m.frameDigest()
	if d0 == d1 {
		t.Fatalf("opening the focus must flip the frame digest")
	}
	hits := m.gov.frameHits
	_ = m.Frame()
	_ = m.Frame()
	if m.gov.frameHits == hits {
		t.Fatalf("an identical second focus frame must hit the digest cache")
	}
	m, _ = selUpdate(t, m, tfEsc())
	if d2 := m.frameDigest(); d2 == d1 {
		t.Fatalf("closing the focus must flip the digest back")
	}
}

// (g) the deferred render: SetState pulses while open record rev+snapshot
// on the main chat but run ZERO renderConversation rebuilds; close runs
// EXACTLY one at the current snapshot.
func TestThreadFocusDefersMainChatRender(t *testing.T) {
	m, _ := tfSetup(t)
	rc0 := m.chat.RenderCalls()
	if rc0 == 0 {
		t.Fatalf("precondition: the setup rendered the main chat")
	}
	m, _ = selUpdate(t, m, tfCtrlF())
	if !m.focusDeferredRender {
		t.Fatalf("opening the focus arms the saver")
	}
	// pulses while open: a tick + a fresh tool row for the focused agent.
	// The tick goes through selUpdate: runMsg would execute tickCmd's
	// sleeping re-arm (the stuck_test idiom — the cmd is observable-only).
	m, _ = selUpdate(t, m, state.Event{Kind: state.EvTick})
	m = runMsg(t, m, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		ToolName: "write", ToolSummary: "markerPulsed.go", ToolState: "done", CallID: "call-t3"})
	if got := m.chat.RenderCalls(); got != rc0 {
		t.Fatalf("the main chat must NOT re-render behind the open focus (rc0=%d got=%d)", rc0, got)
	}
	m, _ = selUpdate(t, m, tfEsc())
	if got := m.chat.RenderCalls(); got != rc0+1 {
		t.Fatalf("the close's ResumeFromFocus must re-render EXACTLY once (rc0=%d got=%d)", rc0, got)
	}
	t.Logf("deferred render: rc0=%d open→%+d · close→+1", rc0, m.chat.RenderCalls()-1-rc0)
	if m.focusDeferredRender {
		t.Fatalf("close must clear the saver")
	}
	if got := ansi.Strip(m.Frame()); !strings.Contains(got, "markerPulsed.go") {
		t.Fatalf("the single catch-up render must land at the CURRENT snapshot:\n%s", got)
	}
}
