// stuck_test.go — the boss-stuck-busy edge cases, app leg (fakes only,
// never a real server):
//
//	W1 — the wedge watchdog: a pending boss turn with NO wall-clock
//	     SERVER-SIDE traffic for bossWedgeAfter fires exactly ONCE (one
//	     red transcript row + the hint-seam swap). The send-side typing
//	     placeholder ("boss-N", Pending, empty text) is the UI's own
//	     staging, so it NEITHER arms the clock NOR re-arms the latch —
//	     only real server-side traffic (stream deltas, thoughts, tools,
//	     completions) does; a wedged turn can therefore never print one
//	     red row per send again. A parked question hold NEVER fires (the
//	     boss is waiting on the USER's answer — user-owned silence is
//	     not a wedge). Silence is wall clock only, never st.Tick (the
//	     governor cadence varies 180ms–3s);
//	W2 (G1) — /stop with a FAILING AbortSessions must NOT strand the
//	     office: one dim note + the exact same clean unwind as success
//	     (placeholder collapse, statusline, watchdog re-armed);
//	F4 sanity — a plan-tagged completion is untouched by the watchdog
//	     (planSendPending still consumes, the latch stays clear).
//
// The EvTick driver calls m.applyEvent DIRECTLY and drops the returned
// tickCmd: executing the governor's tea.Tick would sleep the cadence
// (the quitarm arm-test idiom) — nothing else in the tick branch matters
// to these assertions.
package app

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// abortStubBackend — the /stop seam with a fail knob (G1): recBackend
// covers the base contract, AbortSessions records the call and returns
// the scripted error.
type abortStubBackend struct {
	recBackend
	abortErr   error
	abortCalls int
}

func (a *abortStubBackend) AbortSessions() error {
	a.abortCalls++
	return a.abortErr
}

// wedgeFixture boots a model with ONE outstanding boss typing placeholder
// (the send-sequenced "boss-1") and the watchdog clock ARMED to now. Under
// the W1 contract the placeholder is client-side staging and must NOT
// touch the wall clock — the fixture asserts that, then stamps the clock
// directly (the same white-box seam stale() uses; real-traffic arming
// THROUGH the event path is proven by TestWedgePlaceholderNeverArms).
func wedgeFixture(t *testing.T) (Model, *abortStubBackend) {
	t.Helper()
	b := &abortStubBackend{}
	m := New(b, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})
	if !m.lastBossActivityAt.IsZero() {
		t.Fatal("fixture: the send-side placeholder must never arm the watchdog wall clock")
	}
	m.lastBossActivityAt = time.Now() // stands in for real server-side traffic
	return m, b
}

// pumpTicks feeds n EvTicks straight through applyEvent WITHOUT executing
// the returned tick cmd (its tea.Tick would sleep the governor cadence).
func pumpTicks(m Model, n int) Model {
	for i := 0; i < n; i++ {
		m.applyEvent(state.Event{Kind: state.EvTick})
	}
	return m
}

// stale ages the watchdog's wall clock past the threshold (the brief's
// injection: NEVER tick-derived — the governor's cadence varies, so the
// check is wall-clock truth only).
func stale(m *Model, d time.Duration) {
	m.lastBossActivityAt = time.Now().Add(-d)
}

// countWedgeRows tallies red (Meta "error") transcript rows carrying the
// frozen wedge copy.
func countWedgeRows(m Model) int {
	n := 0
	for _, c := range m.st.Chat {
		if c.Meta == "error" && strings.Contains(c.Text, "boss turn wedged") {
			n++
		}
	}
	return n
}

// W1(a) — fires exactly ONCE at a fake-stale clock: the red row, the
// latch, the hint-seam swap; a second tick says nothing new.
func TestWedgeWatchdogFiresOnce(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter+time.Second)

	m = pumpTicks(m, 1)
	if !m.wedgeNoted {
		t.Fatal("a placeholder idle past bossWedgeAfter must latch the wedge note")
	}
	if n := countWedgeRows(m); n != 1 {
		t.Fatalf("the wedge row must print exactly once, got %d", n)
	}
	if hint := m.hintLine(); !strings.Contains(hint, wedgeHint) {
		t.Fatalf("the latch must swap the hint seam: hint = %q, want %q", hint, wedgeHint)
	}
	if !strings.Contains(m.st.Chat[len(m.st.Chat)-1].Text, "/stop unwinds it (queue intact)") {
		t.Fatalf("the wedge row must offer /stop with the queue intact, got %q", m.st.Chat[len(m.st.Chat)-1].Text)
	}

	m = pumpTicks(m, 3)
	if n := countWedgeRows(m); n != 1 {
		t.Fatalf("the watchdog is one-shot per wedge — %d rows after more ticks", n)
	}
}

// W1(b) — quiet on activity: a fresh wall clock keeps the watchdog silent.
func TestWedgeWatchdogQuietOnActivity(t *testing.T) {
	m, _ := wedgeFixture(t) // the fixture's arming stamp is fresh
	m = pumpTicks(m, 3)
	if m.wedgeNoted {
		t.Fatal("fresh boss activity must keep the watchdog quiet")
	}
	if n := countWedgeRows(m); n != 0 {
		t.Fatalf("no wedge row for a live turn, got %d", n)
	}
	if hint := m.hintLine(); strings.Contains(hint, wedgeHint) {
		t.Fatalf("nothing outstanding to say on the hint seam, got %q", hint)
	}
	// boss stream delta (pending with text) is activity too — it must
	// re-stamp the clock even mid-silence.
	stale(&m, bossWedgeAfter-time.Second)
	m = pumpTicks(m, 1) // 120s-1000ms… under threshold
	if m.wedgeNoted {
		t.Fatal("under-threshold silence must not fire")
	}
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "bossmsg-m1", From: "boss", Text: "half an answer…", Pending: true}})
	m = pumpTicks(m, 1) // the same wall instant would have been over — delta re-stamped it
	if m.wedgeNoted {
		t.Fatal("a stream delta is boss activity: it must re-arm the clock")
	}
}

// W1(c) — the threshold is WALL CLOCK, never st.Tick: sixty ticks of
// silence under 2m (a busy-governor fast cadence would cross any naive
// tick counter) stay quiet; a wall age just over 2m fires on the FIRST
// tick after.
func TestWedgeWatchdogWallClock(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter-time.Second)
	m = pumpTicks(m, 60)
	if m.wedgeNoted || countWedgeRows(m) != 0 {
		t.Fatalf("60 ticks under 2m of wall silence must not fire (ticks are not time)")
	}
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted || countWedgeRows(m) != 1 {
		t.Fatal("past 2m of wall silence, the first tick must fire exactly once")
	}
}

// W1(d) — latch re-arms after a completion: fire once, the completed boss
// bubble clears the latch (its placeholder appends), then a fresh stall
// fires the note AGAIN (one per wedge, not one per session).
func TestWedgeLatchRearmsAfterCompletion(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted {
		t.Fatal("first stall must latch")
	}

	// the turn actually completes: placeholder replaced, latch cleared.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "b1", From: "boss", Text: "done — report shipped"}})
	if m.wedgeNoted {
		t.Fatal("boss activity (the completion) must re-arm the watchdog")
	}
	if hasPendingBoss(m.st) {
		t.Fatal("the completion must close the placeholder")
	}
	if hint := m.hintLine(); strings.Contains(hint, wedgeHint) {
		t.Fatalf("recovery must retire the hint swap, got %q", hint)
	}

	// a NEW turn stalls again: fresh placeholder, stale clock, one note.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-2", From: "boss", Pending: true}})
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if n := countWedgeRows(m); n != 2 {
		t.Fatalf("the re-armed watchdog must note the second wedge once, got %d rows", n)
	}
}

// W1(e) — a parked question hold NEVER fires the watchdog: the boss is
// WAITING on the user's answer, and that user-owned silence is not a
// wedge — the latch stays clear and no row prints, however stale the turn.
// (Answering resumes real traffic, which re-stamps the clock then.)
func TestWedgeQuestionParkedSilent(t *testing.T) {
	m, _ := wedgeFixture(t)
	m.questionParked = true
	// park semantics: the placeholder is dropped from the chat.
	m.st.Chat = nil
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 3)
	if m.wedgeNoted {
		t.Fatal("a parked question hold must never latch the wedge note")
	}
	if n := countWedgeRows(m); n != 0 {
		t.Fatalf("a boss waiting on the user's answer is not wedged — got %d rows", n)
	}
	if hint := m.hintLine(); strings.Contains(hint, wedgeHint) {
		t.Fatalf("the hint seam must stay quiet on a parked question, got %q", hint)
	}
}

// W1(f) — a send-side placeholder ALONE never arms the watchdog clock (it
// proves only that a prompt left the client): a turn with zero server-side
// life stays silent instead of crying wolf. Real traffic THROUGH the event
// path then arms the clock — the first stale stretch after it fires once.
func TestWedgePlaceholderNeverArms(t *testing.T) {
	b := &abortStubBackend{}
	m := New(b, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})
	if !m.lastBossActivityAt.IsZero() {
		t.Fatal("the send-side placeholder must not stamp the watchdog wall clock")
	}
	m = pumpTicks(m, 3)
	if m.wedgeNoted || countWedgeRows(m) != 0 {
		t.Fatal("no real traffic ever → the watchdog stays silent (notice-only beats false-positive)")
	}

	// the first REAL server-side beat (a stream delta — pending WITH text)
	// arms the clock through the normal reducer path; stale it and the
	// first tick fires exactly once.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "bossmsg-m1", From: "boss", Text: "working on it —", Pending: true}})
	if m.lastBossActivityAt.IsZero() {
		t.Fatal("a real stream delta must arm the watchdog wall clock")
	}
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted || countWedgeRows(m) != 1 {
		t.Fatal("once armed by real traffic, past-threshold silence must fire exactly once")
	}
}

// W1(g) — placeholders mid-wedge never RE-ARM: fire once, re-emit the
// placeholder family (the every-send/queue-flush staging) and the latch
// holds, the clock does not move, the row stays exactly one — identical
// repeated rows from placeholder emissions are impossible. REAL traffic
// then re-opens a fresh episode: one further row, no more.
func TestWedgePlaceholderNeverReArms(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted || countWedgeRows(m) != 1 {
		t.Fatal("setup: the first wedge must latch with exactly one row")
	}

	armedAt := m.lastBossActivityAt
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-2", From: "boss", Pending: true}})
	if !m.lastBossActivityAt.Equal(armedAt) {
		t.Fatal("a mid-wedge placeholder must NOT move the watchdog clock")
	}
	if !m.wedgeNoted {
		t.Fatal("a mid-wedge placeholder must NOT clear the wedge latch")
	}
	m = pumpTicks(m, 3)
	if n := countWedgeRows(m); n != 1 {
		t.Fatalf("placeholders must never reprint the wedge row, got %d rows", n)
	}

	// real traffic (a stream delta) re-arms the episode; a fresh silence
	// notes ONE more row — one per episode, never per placeholder.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "bossmsg-m2", From: "boss", Text: "half an answer…", Pending: true}})
	if m.wedgeNoted {
		t.Fatal("real boss traffic must re-arm the latch for the next episode")
	}
	if m.lastBossActivityAt.Equal(armedAt) {
		t.Fatal("real boss traffic must re-stamp the watchdog clock")
	}
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 2)
	if n := countWedgeRows(m); n != 2 {
		t.Fatalf("the re-armed watchdog notes the second episode exactly once, got %d rows", n)
	}
}

// W2 (G1) — /stop ALWAYS unwinds: abort fails remotely, yet the office
// recovers exactly like success — placeholder collapses to the
// "stopped by user" dim row, the statusline reads stopped, the watchdog
// re-arms, and exactly ONE dim note (never the old red early-return)
// records the remote failure.
func TestStopAbortErrorUnwinds(t *testing.T) {
	m, b := wedgeFixture(t)
	b.abortErr = errors.New("abort: dial 127.0.0.1: refused")
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1) // wedge latched (the watchdog's /stop advice is what we're proving)
	if !m.wedgeNoted {
		t.Fatal("setup: the wedge latch should be set before /stop")
	}

	m = runMsg(t, m, slashMsg{text: "/stop"})

	if b.abortCalls != 1 {
		t.Fatalf("AbortSessions must be attempted once, got %d", b.abortCalls)
	}
	// (a) the placeholder unwound EXACTLY like a success stop.
	found := false
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Text == "stopped by user" {
			found = true
		}
	}
	if !found {
		t.Fatal("a failed abort must still collapse the placeholder to \"stopped by user\"")
	}
	if hasPendingBoss(m.st) {
		t.Fatal("no placeholder may survive /stop, abort failure or not")
	}
	// (b) ONE dim note about the remote failure (Meta "" — the old path's
	// red noticeErr row is gone).
	n := 0
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Meta == "" && strings.Contains(c.Text, "abort signal failed remotely") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want exactly one dim abort-failure note, got %d", n)
	}
	for _, c := range m.st.Chat {
		if c.Meta == "error" && strings.Contains(c.Text, "abort failed") {
			t.Fatalf("the old early-return red row must be gone, found %q", c.Text)
		}
	}
	// (c) the statusline + watchdog close exactly like success.
	if want := "stopped current work — queue intact (0 items)"; m.st.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", m.st.StatusLine, want)
	}
	if m.wedgeNoted {
		t.Fatal("/stop closes the turn: the watchdog must re-arm")
	}
	if m.st.BossThinking || m.st.BossDelegating {
		t.Fatal("BossThinking/BossDelegating must clear on /stop even when abort fails")
	}
}

// W2 sanity — a WORKING abort keeps its exact behavior (no dim failure
// note), lockstep with the --stop uishot leg.
func TestStopAbortSuccessUnchanged(t *testing.T) {
	m, b := wedgeFixture(t)
	m = runMsg(t, m, slashMsg{text: "/stop"})
	if b.abortCalls != 1 {
		t.Fatalf("AbortSessions must be attempted once, got %d", b.abortCalls)
	}
	for _, c := range m.st.Chat {
		if strings.Contains(c.Text, "abort signal failed") {
			t.Fatalf("a successful abort prints no failure note, found %q", c.Text)
		}
	}
	if want := "stopped current work — queue intact (0 items)"; m.st.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", m.st.StatusLine, want)
	}
}

// F4 sanity — a plan-tagged completion rides the SAME event path as the
// watchdog's re-arm: planSendPending consumes normally, no wedge
// interference.
func TestWedgePlanCompletionUnaffected(t *testing.T) {
	m, _ := wedgeFixture(t)
	m.planSendPending = 1 // chatSentMsg stamped it for the plan-tagged send
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1) // wedged while the plan-turn is out
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "b1", From: "boss", Text: "plan landed"}})
	if m.planSendPending != 0 {
		t.Fatalf("F4: the completion must consume the plan-tagged send, got %d", m.planSendPending)
	}
	if m.wedgeNoted {
		t.Fatal("the plan completion is boss activity: the latch must clear")
	}
}
