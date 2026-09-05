// notify_hook_test.go — the app's desktop-notification hooks end-to-end with a
// recording bus at the seam: startup focus defaults true (an unsupported
// terminal never false-pings), a permission COHORT pings once per 0→1 flip of
// the unanswered set (generic agent+tool copy — never the ToolSummary), the
// done debounce fires exactly once per send, focused and /notify-off stay
// silent, and a blur during a live cohort pings immediately (the "looked away
// during a block" intent).
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// tapNotifyBus — recording NotifyBus for the hook asserts (taps the full
// kind|title|body triple; modes records live SetMode flips from /notify).
type tapNotifyBus struct {
	taps  []string
	modes []string
}

func (b *tapNotifyBus) Notify(kind, title, body string) {
	b.taps = append(b.taps, kind+"|"+title+"|"+body)
}
func (b *tapNotifyBus) SetMode(mode string) { b.modes = append(b.modes, mode) }

func newNotifyRig(t *testing.T) (*Model, *tapNotifyBus) {
	t.Helper()
	m := New(&recBackend{}, nil)
	bus := &tapNotifyBus{}
	m.SetNotifyBus(bus)
	return &m, bus
}

func permEvent(id, agent, tool, toolState string) state.Event {
	return state.Event{Kind: state.EvPermission, PermissionID: id,
		EmployeeName: agent, ToolName: tool, ToolSummary: "/do/not/leak/me",
		ToolState: toolState}
}

func bossDoneEvent(id, text string) state.Event {
	return state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: id, From: "boss", Text: text, Pending: false}}
}

func TestNotifyStartupFocusDefaultsTrue(t *testing.T) {
	m, bus := newNotifyRig(t)
	if !m.focused {
		t.Fatal("startup focus latch must default true — unsupported terminals never false-ping")
	}
	// a full ask+turn cycle while focused stays silent
	*m = runMsg(t, *m, permEvent("perm-1", "boss", "write", "pending"))
	*m = runMsg(t, *m, chatSentMsg{text: "do it"})
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-1", "done"))
	if len(bus.taps) != 0 {
		t.Fatalf("focused pings must never fire, got %v", bus.taps)
	}
}

func TestNotifyPermissionCohortExactlyOnce(t *testing.T) {
	m, bus := newNotifyRig(t)

	// blur first — asking while blurred pings the cohort opener ONCE with
	// generic body (agent + tool NAME; the ToolSummary path must NOT leak)
	*m = runMsg(t, *m, tea.BlurMsg{})
	*m = runMsg(t, *m, permEvent("perm-1", "boss", "write", "pending"))
	want := "permission|theboringfloor|permission needed — boss needs write"
	if len(bus.taps) != 1 || bus.taps[0] != want {
		t.Fatalf("cohort-opening ask must ping once with generic copy:\ngot  %v\nwant [%q]", bus.taps, want)
	}

	// a child ask inside the same cohort coalesces silently
	*m = runMsg(t, *m, permEvent("perm-2", "tekton-1", "read", "pending"))
	if len(bus.taps) != 1 {
		t.Fatalf("child ask inside a live cohort must coalesce, got %v", bus.taps)
	}

	// resolving the boss ask shrinks but keeps the cohort live; a THIRD ask
	// still rides the same cohort — silent
	*m = runMsg(t, *m, permEvent("perm-1", "boss", "", "resolved"))
	*m = runMsg(t, *m, permEvent("perm-3", "skopos-1", "grep", "pending"))
	if len(bus.taps) != 1 {
		t.Fatalf("resolves + later cohort asks stay silent, got %v", bus.taps)
	}

	// empty the cohort → the NEXT 0→1 asks re-arms and pings again
	*m = runMsg(t, *m, permEvent("perm-2", "tekton-1", "", "resolved"))
	*m = runMsg(t, *m, permEvent("perm-3", "skopos-1", "", "resolved"))
	*m = runMsg(t, *m, permEvent("perm-4", "boss", "bash", "pending"))
	if len(bus.taps) != 2 || bus.taps[1] != "permission|theboringfloor|permission needed — boss needs bash" {
		t.Fatalf("a fresh cohort after emptying must re-arm exactly once, got %v", bus.taps)
	}
}

func TestNotifyDoneExactlyOnce(t *testing.T) {
	m, bus := newNotifyRig(t)
	*m = runMsg(t, *m, tea.BlurMsg{})

	// send → completion fires ONE done ping, body clipped to one line
	*m = runMsg(t, *m, chatSentMsg{text: "wire the notifier"})
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-1", "done — the fleet is wired.\nTests are green too."))
	want := "done|theboringfloor|the boss is done — done — the fleet is wired. Tests are green too."
	if len(bus.taps) != 1 || bus.taps[0] != want {
		t.Fatalf("armed completion must ping once with the clipped reply:\ngot  %v\nwant [%q]", bus.taps, want)
	}

	// the arm is consumed: a second completion with NO new send is silent
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-2", "nothing new"))
	if len(bus.taps) != 1 {
		t.Fatalf("the done arm is one-shot — a later completion must not re-ping, got %v", bus.taps)
	}

	// a new send re-arms; boss-error bubbles disarm silently (the error
	// already owns the transcript)
	*m = runMsg(t, *m, chatSentMsg{text: "again"})
	*m = runMsg(t, *m, bossDoneEvent("boss-error-9", "boom"))
	if len(bus.taps) != 1 {
		t.Fatalf("boss-error completions must not ping, got %v", bus.taps)
	}
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-3", "respawn reply"))
	if len(bus.taps) != 1 {
		t.Fatalf("the error disarmed the send — no deferred ping, got %v", bus.taps)
	}
}

func TestNotifyQuestionParkedCountsAsNotDone(t *testing.T) {
	m, bus := newNotifyRig(t)
	*m = runMsg(t, *m, tea.BlurMsg{})
	*m = runMsg(t, *m, chatSentMsg{text: "plan it"})

	// the turn parks at a question; the unblocking completion lands while
	// questionParked — the member just engaged through the modal: no ping,
	// and the arm is consumed.
	m.questionParked = true
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-1", "resumed after the answer"))
	if len(bus.taps) != 0 {
		t.Fatalf("parked-question completions count as not-done, got %v", bus.taps)
	}
	if m.notifyDoneArmed {
		t.Fatal("the done arm must be consumed even when the ping is skipped")
	}
}

func TestNotifyBlurFiresLiveCohortImmediately(t *testing.T) {
	m, bus := newNotifyRig(t)

	// asks arrive while FOCUSED (silent), then the member looks away mid-block
	*m = runMsg(t, *m, permEvent("perm-1", "boss", "write", "pending"))
	*m = runMsg(t, *m, permEvent("perm-2", "tekton-1", "read", "pending"))
	if len(bus.taps) != 0 {
		t.Fatalf("focused asks mint no pings, got %v", bus.taps)
	}
	*m = runMsg(t, *m, tea.BlurMsg{})
	want := "permission|theboringfloor|permission needed — boss needs write"
	if len(bus.taps) != 1 || bus.taps[0] != want {
		t.Fatalf("blur during a live cohort must fire the front's ping:\ngot  %v\nwant [%q]", bus.taps, want)
	}

	// every blur while the cohort is live is its own reminder nudge
	*m = runMsg(t, *m, tea.FocusMsg{})
	*m = runMsg(t, *m, tea.BlurMsg{})
	if len(bus.taps) != 2 || bus.taps[1] != want {
		t.Fatalf("a second blur on the live cohort must re-nudge, got %v", bus.taps)
	}

	// cohort emptied (both answered user-side) → later blurs fall silent
	*m = runMsg(t, *m, permAnswerMsg{response: "once"})   // front: boss ask
	*m = runMsg(t, *m, permAnswerMsg{response: "always"}) // front now: child ask
	*m = runMsg(t, *m, tea.FocusMsg{})
	*m = runMsg(t, *m, tea.BlurMsg{})
	if len(bus.taps) != 2 {
		t.Fatalf("an emptied cohort must stop blur nudges, got %v", bus.taps)
	}
}

func TestNotifySlashToggle(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir()) // persistCfg lands in scratch, never the real brain.json
	m, bus := newNotifyRig(t)

	// bare /notify: status line reports the current mode
	*m = runMsg(t, *m, slashMsg{text: "/notify"})
	if last := m.st.Chat[len(m.st.Chat)-1]; last.Text == "" || !strings.Contains(last.Text, "notifications on") {
		t.Fatalf("bare /notify must report the mode, last notice: %q", last.Text)
	}

	// /notify off: config flips, bus mode flips live, later pings die at the
	// config gate
	*m = runMsg(t, *m, slashMsg{text: "/notify off"})
	if m.cfg.UI.Notifications != "off" {
		t.Fatalf("/notify off must flip cfg.UI.Notifications, got %q", m.cfg.UI.Notifications)
	}
	if len(bus.modes) != 1 || bus.modes[0] != "off" {
		t.Fatalf("/notify off must live-set the bus SetMode, got %v", bus.modes)
	}
	*m = runMsg(t, *m, tea.BlurMsg{})
	*m = runMsg(t, *m, permEvent("perm-1", "boss", "write", "pending"))
	*m = runMsg(t, *m, chatSentMsg{text: "x"})
	*m = runMsg(t, *m, bossDoneEvent("bossmsg-1", "done"))
	if len(bus.taps) != 0 {
		t.Fatalf("/notify off must gate every hook, got %v", bus.taps)
	}
}
