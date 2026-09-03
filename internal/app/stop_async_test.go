// stop_async_test.go — the UI-freeze regression pins (app leg, fakes only,
// never a real server):
//
//	G5(a) — /stop NEVER parks the UI goroutine on AbortSessions: with a
//	     backend whose abort wedges forever (the black-holed serve), the
//	     slash's Update must return at once, the synchronous unwind must
//	     land, and the office must keep answering the VERY NEXT command
//	     while the abort hop is still out. The async landing then arrives
//	     harmlessly: exactly one G1 dim note on failure, silence on
//	     success, and never a second unwind;
//	G5(b) — /clear is I/O-free and returns instantly, even with a fat
//	     transcript and even while that wedged abort is still parked
//	     mid-flight (the reported freeze was /stop's collateral: every
//	     later message queued behind the wedged UI goroutine);
//	G5(c) — the double-esc stop seam (stopWorkMsg) rides the same async
//	     shape.
//
// The watchdogs below fire ONLY on failure — each one runs the Update
// (or cmd) on a goroutine and fails the test when the goroutine never
// returns. House style stands: nothing here ever sleeps a tick.
package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// hungAbortBackend — the /stop seam behind a wedged serve: AbortSessions
// announces itself on called, then parks until the test releases it
// (read: the network hop that never answers). recBackend covers the rest
// of the Backend contract.
type hungAbortBackend struct {
	recBackend
	err     error
	calls   int
	called  chan struct{}
	release chan struct{}
}

func (h *hungAbortBackend) AbortSessions() error {
	h.calls++
	select {
	case h.called <- struct{}{}:
	default:
	}
	<-h.release
	return h.err
}

// slashUpdateOut carries Update's (model, cmd) back across the watchdog
// goroutine.
type slashUpdateOut struct {
	m   Model
	cmd tea.Cmd
}

// updateQuick runs ONE msg through m.Update on a goroutine and fails the
// test when the goroutine has not returned inside the budget — the
// structural proof that a slash handler can never park the UI (the
// failure-only watchdog; 2s is an eternity for a goroutine that just
// returns).
func updateQuick(t *testing.T, what string, m Model, msg tea.Msg) slashUpdateOut {
	t.Helper()
	done := make(chan slashUpdateOut, 1)
	go func() {
		nm, cmd := m.Update(msg)
		done <- slashUpdateOut{m: nm.(Model), cmd: cmd}
	}()
	select {
	case out := <-done:
		return out
	case <-time.After(2 * time.Second):
		t.Fatalf("%s parked the UI goroutine — the freeze is back", what)
		return slashUpdateOut{}
	}
}

// hungStopFixture boots a model over a hungAbortBackend with ONE
// outstanding boss typing placeholder (wedgeFixture's hung-abort twin).
// err is the abort's scripted return; the wedging is released only by
// closing b.release.
func hungStopFixture(t *testing.T, err error) (Model, *hungAbortBackend) {
	t.Helper()
	b := &hungAbortBackend{
		err:     err,
		called:  make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	m := New(b, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})
	return m, b
}

// execQuick runs ONE tea.Cmd on a goroutine and fails the test when it
// has not delivered its message inside the budget.
func execQuick(t *testing.T, what string, cmd tea.Cmd) tea.Msg {
	t.Helper()
	res := make(chan tea.Msg, 1)
	go func() { res <- cmd() }()
	select {
	case msg := <-res:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatalf("%s never delivered — the async hop is wedged", what)
		return nil
	}
}

// countNotes tallies chat rows carrying text in one Meta class (the G1
// note counter's local twin, meta-agnostic to keep this file standalone).
func countNotes(m Model, meta, needle string) int {
	n := 0
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Meta == meta && strings.Contains(c.Text, needle) {
			n++
		}
	}
	return n
}

// G5(a) — the wedge-proof: /stop with a NEVER-answering AbortSessions
// returns instantly, unwinds synchronously, and keeps the office alive;
// the async verdict then lands exactly once, with the G1 dim note.
func TestStopNeverParksUIOnWedgedAbort(t *testing.T) {
	m, b := hungStopFixture(t, errors.New("abort: dial 127.0.0.1: wedged"))

	out := updateQuick(t, "/stop (against a wedged abort)", m, slashMsg{text: "/stop"})
	m, abortCmd := out.m, out.cmd

	// the SYNCHRONOUS half landed: placeholder collapsed, statusline set,
	// and the abort was NOT attempted on the UI goroutine.
	found := false
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Text == "stopped by user" {
			found = true
		}
	}
	if !found {
		t.Fatal("the placeholder must collapse to \"stopped by user\" the instant /stop returns")
	}
	if hasPendingBoss(m.st) {
		t.Fatal("no placeholder may survive /stop — wedged abort or not")
	}
	if want := "stopped current work — queue intact (0 items)"; m.st.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", m.st.StatusLine, want)
	}
	if b.calls != 0 {
		t.Fatalf("AbortSessions must NOT run on the UI goroutine, already called %d time(s)", b.calls)
	}
	if abortCmd == nil {
		t.Fatal("/stop must ship the async abort cmd (the backend has the abort seam)")
	}

	// the office is ALIVE while the abort is still parked: the very next
	// slash answers at once.
	m = runMsg(t, m, slashMsg{text: "/queue"})
	if countNotes(m, "", "backlog empty") != 1 {
		t.Fatal("the slash after /stop must answer immediately — the UI loop was frozen")
	}

	// the async half: the cmd runs OFF the UI goroutine, announces the
	// attempt, parks on the wedge, and — once released with the failure —
	// lands its verdict; G1 then prints exactly ONE dim note, no reds.
	stopRes := make(chan tea.Msg, 1)
	go func() { stopRes <- abortCmd() }()
	select {
	case <-b.called:
	case <-time.After(2 * time.Second):
		t.Fatal("the async abort cmd never reached the backend")
	}
	if b.calls != 1 {
		t.Fatalf("AbortSessions must be attempted exactly once, got %d", b.calls)
	}
	close(b.release)
	var msg tea.Msg
	select {
	case msg = <-stopRes:
	case <-time.After(2 * time.Second):
		t.Fatal("the released abort cmd never delivered its result")
	}
	if _, ok := msg.(stopAbortResultMsg); !ok {
		t.Fatalf("the abort cmd must land a stopAbortResultMsg, got %T", msg)
	}

	m = runMsg(t, m, msg)
	if n := countNotes(m, "", "abort signal failed remotely"); n != 1 {
		t.Fatalf("want exactly one dim abort-failure note on the landing, got %d", n)
	}
	for _, c := range m.st.Chat {
		if c.Meta == "error" && strings.Contains(c.Text, "abort failed") {
			t.Fatalf("no red early-return rows may come back, found %q", c.Text)
		}
	}
	// never a second unwind: the stopped row stayed unique.
	stopped := 0
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Text == "stopped by user" {
			stopped++
		}
	}
	if stopped != 1 {
		t.Fatalf("the async landing must never re-unwind, got %d stopped rows", stopped)
	}
}

// G5(a2) — the success landing is silent: a released nil verdict adds
// nothing to the transcript.
func TestStopAbortSuccessLandingSilent(t *testing.T) {
	m, b := hungStopFixture(t, nil)

	out := updateQuick(t, "/stop", m, slashMsg{text: "/stop"})
	m, abortCmd := out.m, out.cmd

	before := len(m.st.Chat)
	go func() { <-b.called; close(b.release) }()
	msg := execQuick(t, "the successful abort cmd", abortCmd)
	m = runMsg(t, m, msg)

	if got := len(m.st.Chat); got != before {
		t.Fatalf("a successful abort landing must add ZERO rows, went %d -> %d", before, got)
	}
	if n := countNotes(m, "", "abort signal failed"); n != 0 {
		t.Fatalf("a successful abort prints no failure note, got %d", n)
	}
}

// G5(b) — /clear is instant: a fat transcript (the 200-row cap) clears
// in one bounded Update, and a second /clear no-ops just as fast.
func TestClearReturnsFastAlways(t *testing.T) {
	b := &hungAbortBackend{
		called:  make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	m := New(b, config.Default())
	for i := 0; i < 200; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{
			ID:   "row-" + strings.Repeat("x", 3) + string(rune('a'+i%26)),
			From: "boss", Text: strings.Repeat("a longish boss answer, wave after wave. ", 8),
			At: int64(1700000000000 + i),
		})
	}
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 160, Height: 40}) // arm the frame machinery too

	out := updateQuick(t, "/clear over a 200-row transcript", m, slashMsg{text: "/clear"})
	m = out.m
	if len(m.st.Chat) != 0 {
		t.Fatalf("/clear must empty the transcript, %d rows left", len(m.st.Chat))
	}
	// repeat safety: the second /clear is the same instant no-op.
	out = updateQuick(t, "/clear (repeat)", m, slashMsg{text: "/clear"})
	m = out.m
	if len(m.st.Chat) != 0 {
		t.Fatalf("/clear (repeat) must stay empty, %d rows left", len(m.st.Chat))
	}
}

// G5(b2) — the reported three-command freeze was /stop's collateral:
// /clear returns instantly even while the wedged abort hop is STILL
// parked mid-flight.
func TestClearInstantWhileAbortWedged(t *testing.T) {
	m, b := hungStopFixture(t, errors.New("abort: wedged"))

	out := updateQuick(t, "/stop (against a wedged abort)", m, slashMsg{text: "/stop"})
	m = out.m
	// run the async hop but never release it: the abort is parked for real.
	go func() { _ = out.cmd() }()
	select {
	case <-b.called:
	case <-time.After(2 * time.Second):
		t.Fatal("setup: the abort hop never started")
	}

	out = updateQuick(t, "/clear while the abort is wedged mid-flight", m, slashMsg{text: "/clear"})
	m = out.m
	if len(m.st.Chat) != 0 {
		t.Fatalf("/clear must empty the transcript even mid-abort, %d rows left", len(m.st.Chat))
	}
	close(b.release) // let the parked hop die; its landing goes nowhere after this
}

// G5(c) — the double-esc stop seam ships the same async cmd shape (one
// hop per press, verdict by message — never on the UI goroutine).
func TestDoubleEscStopShipsAsyncCmd(t *testing.T) {
	m, b := hungStopFixture(t, nil)

	out := updateQuick(t, "double-esc stop (against a wedged abort)", m, stopWorkMsg{})
	m, abortCmd := out.m, out.cmd
	if abortCmd == nil {
		t.Fatal("the double-esc stop must ship the async abort cmd")
	}
	if b.calls != 0 {
		t.Fatalf("the abort must not run on the UI goroutine, called %d time(s)", b.calls)
	}
	if want := "stopped current work — queue intact (0 items)"; m.st.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", m.st.StatusLine, want)
	}
}
