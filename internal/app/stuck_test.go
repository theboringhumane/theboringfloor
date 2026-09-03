// stuck_test.go — the boss-stuck-busy edge cases, app leg (fakes only,
// never a real server):
//
//	W1 — the wedge watchdog: a pending boss turn with NO wall-clock
//	     SERVER-SIDE traffic for bossWedgeAfter fires exactly ONCE (one
//	     ACTIVITY-TAB line + the hint-seam swap — NEVER a transcript
//	     row). The send-side typing placeholder ("boss-N", Pending,
//	     empty text) is the UI's own staging, so it NEITHER arms the
//	     clock NOR re-arms the latch — only real server-side traffic
//	     (stream deltas, thoughts, tools, completions) does; a wedged
//	     turn can therefore never note one line per send again. A
//	     parked question hold NEVER fires (the boss is waiting on the
//	     USER's answer — user-owned silence is not a wedge). Silence
//	     is wall clock only, never st.Tick (the governor cadence varies
//	     180ms–3s);
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
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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

// countWedgeLines tallies ACTIVITY-TAB lines carrying the frozen wedge
// copy — the note's ONLY home since the transcript-off wave (a wedge
// never lands in st.Chat; assertNoWedgeInChat pins that invariant).
func countWedgeLines(m Model) int {
	n := 0
	for _, ln := range m.activity.Lines() {
		if strings.Contains(ln, "boss turn wedged") {
			n++
		}
	}
	return n
}

// assertNoWedgeInChat — the transcript-off invariant: NO chat row ever
// carries the wedge copy, in any meta class, boot-scoped or not.
func assertNoWedgeInChat(t *testing.T, m Model) {
	t.Helper()
	for _, c := range m.st.Chat {
		if strings.Contains(c.Text, "boss turn wedged") {
			t.Fatalf("the wedge note must never land in the transcript, found %q (meta %q)", c.Text, c.Meta)
		}
	}
}

// W1(a) — fires exactly ONCE at a fake-stale clock: the activity line,
// the latch, the hint-seam swap; a second tick says nothing new.
func TestWedgeWatchdogFiresOnce(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter+time.Second)

	m = pumpTicks(m, 1)
	if !m.wedgeNoted {
		t.Fatal("a placeholder idle past bossWedgeAfter must latch the wedge note")
	}
	if n := countWedgeLines(m); n != 1 {
		t.Fatalf("the wedge note must land exactly once, got %d activity lines", n)
	}
	assertNoWedgeInChat(t, m)
	if hint := m.hintLine(); !strings.Contains(hint, wedgeHint) {
		t.Fatalf("the latch must swap the hint seam: hint = %q, want %q", hint, wedgeHint)
	}
	var wedgeLine string
	for _, ln := range m.activity.Lines() {
		if strings.Contains(ln, "boss turn wedged") {
			wedgeLine = ln
		}
	}
	if !strings.Contains(wedgeLine, "/stop unwinds it (queue intact)") {
		t.Fatalf("the wedge line must offer /stop with the queue intact, got %q", wedgeLine)
	}

	m = pumpTicks(m, 3)
	if n := countWedgeLines(m); n != 1 {
		t.Fatalf("the watchdog is one-shot per wedge — %d lines after more ticks", n)
	}
	assertNoWedgeInChat(t, m)
}

// W1(h) — the notice lives OFF the transcript: a fired wedge renders as
// ONE dim-timestamped line in the activity VIEW (same "[stamp] …" seam as
// every other entry), ZERO rows in st.Chat, and the hint seam reads the
// warn swap while armed — re-ticking duplicates nothing.
func TestWedgeNoticeLivesOffTranscript(t *testing.T) {
	m, _ := wedgeFixture(t)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 160, Height: 40})
	stale(&m, bossWedgeAfter+time.Second)

	m = pumpTicks(m, 1)
	if !m.wedgeNoted {
		t.Fatal("setup: the wedge must latch past the threshold")
	}

	// st.Chat carries ZERO wedge rows.
	assertNoWedgeInChat(t, m)

	// the activity tab carries exactly ONE line, timestamped through the
	// same "[stamp] …" seam as describeEvent / [memory] recorded entries.
	if n := countWedgeLines(m); n != 1 {
		t.Fatalf("the activity tab must hold the wedge note exactly once, got %d", n)
	}
	var wedgeLine string
	for _, ln := range m.activity.Lines() {
		if strings.Contains(ln, "boss turn wedged") {
			wedgeLine = ln
		}
	}
	if !strings.HasPrefix(wedgeLine, "[") || !strings.Contains(wedgeLine, "] boss turn wedged: ") {
		t.Fatalf("the wedge line must ride the [stamp] activity format, got %q", wedgeLine)
	}
	if !strings.Contains(wedgeLine, "/stop unwinds it (queue intact); the turn may still complete on its own") {
		t.Fatalf("the /stop advice must survive verbatim in the activity line, got %q", wedgeLine)
	}

	// the activity VIEW renders it once, dim-timestamped like every entry
	// (the pane clips long lines to width, so the full copy is asserted
	// on the raw line above).
	raw := m.activity.View()
	plain := ansi.Strip(raw)
	if n := strings.Count(plain, "boss turn wedged"); n != 1 {
		t.Fatalf("the activity view must render the wedge line exactly once, got %d:\n%s", n, plain)
	}
	if !strings.Contains(raw, "\x1b[") {
		t.Fatalf("the wedge line must render through the dim-timestamp style, raw view has no styling:\n%s", plain)
	}
	if i := strings.Index(raw, "boss turn wedged"); i < 8 || !strings.Contains(raw[max(0, i-40):i], "\x1b[") {
		t.Fatalf("the wedge line's [stamp] prefix must render dim-styled, raw view:\n%s", raw)
	}

	// the status bar reads the red swap while the latch is armed.
	if hint := m.hintLine(); !strings.Contains(hint, wedgeHint) {
		t.Fatalf("the status bar must read the wedge hint while armed, got %q", hint)
	}

	// the next tick fires nothing — one activity line per episode.
	adds := m.activityAdds
	m = pumpTicks(m, 1)
	if m.activityAdds != adds {
		t.Fatalf("a further tick must never duplicate the note (activityAdds %d → %d)", adds, m.activityAdds)
	}
	assertNoWedgeInChat(t, m)
}

// W1(b) — quiet on activity: a fresh wall clock keeps the watchdog silent.
func TestWedgeWatchdogQuietOnActivity(t *testing.T) {
	m, _ := wedgeFixture(t) // the fixture's arming stamp is fresh
	m = pumpTicks(m, 3)
	if m.wedgeNoted {
		t.Fatal("fresh boss activity must keep the watchdog quiet")
	}
	if n := countWedgeLines(m); n != 0 {
		t.Fatalf("no wedge line for a live turn, got %d", n)
	}
	assertNoWedgeInChat(t, m)
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
	if m.wedgeNoted || countWedgeLines(m) != 0 {
		t.Fatalf("60 ticks under 2m of wall silence must not fire (ticks are not time)")
	}
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted || countWedgeLines(m) != 1 {
		t.Fatal("past 2m of wall silence, the first tick must fire exactly once")
	}
	assertNoWedgeInChat(t, m)
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
	if n := countWedgeLines(m); n != 2 {
		t.Fatalf("the re-armed watchdog must note the second wedge once, got %d lines", n)
	}
	assertNoWedgeInChat(t, m)
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
	if n := countWedgeLines(m); n != 0 {
		t.Fatalf("a boss waiting on the user's answer is not wedged — got %d lines", n)
	}
	assertNoWedgeInChat(t, m)
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
	if m.wedgeNoted || countWedgeLines(m) != 0 {
		t.Fatal("no real traffic ever → the watchdog stays silent (notice-only beats false-positive)")
	}
	assertNoWedgeInChat(t, m)

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
	if !m.wedgeNoted || countWedgeLines(m) != 1 {
		t.Fatal("once armed by real traffic, past-threshold silence must fire exactly once")
	}
	assertNoWedgeInChat(t, m)
}

// W1(g) — placeholders mid-wedge never RE-ARM: fire once, re-emit the
// placeholder family (the every-send/queue-flush staging) and the latch
// holds, the clock does not move, the activity line stays exactly one —
// identical repeated lines from placeholder emissions are impossible.
// REAL traffic then re-opens a fresh episode: one further line, no more.
func TestWedgePlaceholderNeverReArms(t *testing.T) {
	m, _ := wedgeFixture(t)
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if !m.wedgeNoted || countWedgeLines(m) != 1 {
		t.Fatal("setup: the first wedge must latch with exactly one activity line")
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
	if n := countWedgeLines(m); n != 1 {
		t.Fatalf("placeholders must never reprint the wedge note, got %d lines", n)
	}

	// real traffic (a stream delta) re-stamps the CLOCK mid-turn — the
	// derived hint clears itself — but does NOT re-arm the fired latch:
	// one activity line per TURN, never one per quiet stretch (the field
	// fix).
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "bossmsg-m2", From: "boss", Text: "half an answer…", Pending: true}})
	if !m.wedgeNoted {
		t.Fatal("mid-turn real traffic must NOT re-arm the fired latch (one line per turn until the turn ends)")
	}
	if m.lastBossActivityAt.Equal(armedAt) {
		t.Fatal("real boss traffic must re-stamp the watchdog clock")
	}
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 2)
	if n := countWedgeLines(m); n != 1 {
		t.Fatalf("a spent latch must NEVER reprint mid-turn, got %d lines", n)
	}
	assertNoWedgeInChat(t, m)

	// close the mid-stream delta first (its completion swap), so the
	// turn-ending bubble below actually closes the WHOLE turn — an open
	// stream delta counts as pending boss and would keep the latch armed.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "bossmsg-m2", From: "boss", Text: "half an answer…"}})
	if hasPendingBoss(m.st) {
		t.Fatal("the mid-stream delta's completion must clear its placeholder")
	}

	// the turn actually ends: the completion placeholder-swap clears the
	// latch for the NEXT episode (existing reset path).
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "b1", From: "boss", Text: "done — report shipped"}})
	if m.wedgeNoted {
		t.Fatal("the completion (turn end) must re-arm the watchdog for the next episode")
	}

	// a NEW turn stalls again: one more line, exactly once.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-2", From: "boss", Pending: true}})
	stale(&m, bossWedgeAfter+time.Second)
	m = pumpTicks(m, 1)
	if n := countWedgeLines(m); n != 2 {
		t.Fatalf("the turn-ended watchdog notes the next episode once, got %d lines", n)
	}
	assertNoWedgeInChat(t, m)
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
	assertNoWedgeInChat(t, m) // even mid-wedge, the note stays out of the transcript

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
	assertNoWedgeInChat(t, m)
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
	assertNoWedgeInChat(t, m)
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
	assertNoWedgeInChat(t, m)
}

func idleWrapFixture(t *testing.T) Model {
	t.Helper()
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-ask", From: "user", Text: "ship the recap path",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking, Task: "ship it",
	}})
	if !m.shiftBusy {
		t.Fatal("fixture: a live developer must open a busy shift")
	}
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: state.MailItem{Kind: state.MailReturn, From: "tekton-1", Subject: "done"}})
	if m.shiftBusy || m.ghostArmAt.IsZero() {
		t.Fatal("fixture: worker return with last chat from the user must arm the idle-wrap clock")
	}
	return m
}

func TestIdleWrapFiresWhenShiftGoesQuiet(t *testing.T) {
	m := idleWrapFixture(t)
	b := m.backend.(*recBackend)
	m.ghostArmAt = time.Now().Add(-bossWedgeAfter - time.Second)
	cmd := m.checkIdleWrap()
	if cmd == nil {
		t.Fatal("2m idle after work with no wrap must ask the boss for a recap")
	}
	if !m.ghostNoted {
		t.Fatal("the latch must arm before the send hops")
	}
	msg := cmd()
	if _, ok := msg.(idleWrapSentMsg); !ok {
		t.Fatalf("recap send = %T, want idleWrapSentMsg", msg)
	}
	m = runMsg(t, m, msg)
	want := idleWrapPromptHead + " Their last ask: ship the recap path"
	if !reflect.DeepEqual(b.sentTexts, []string{want}) {
		t.Fatalf("boss recap prompt = %v, want %q", b.sentTexts, want)
	}
	for _, c := range m.st.Chat {
		if c.Text == idleWrapNotice {
			t.Fatal("a successful recap send must not dump the fallback notice")
		}
	}
	n := 0
	for _, ln := range m.ActivityLines() {
		if strings.Contains(ln, "asked the boss for an idle recap") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("activity recap lines = %d, want 1", n)
	}
}

func TestIdleWrapSendFailureFallsBackToNotice(t *testing.T) {
	m := idleWrapFixture(t)
	m.ghostArmAt = time.Now().Add(-bossWedgeAfter - time.Second)
	m.currentBackend.current = nil
	cmd := m.checkIdleWrap()
	if cmd == nil {
		t.Fatal("unavailable transport must still try the recap hop")
	}
	msg := cmd()
	if _, ok := msg.(idleWrapFailMsg); !ok {
		t.Fatalf("failed recap send = %T, want idleWrapFailMsg", msg)
	}
	m = runMsg(t, m, msg)
	found := false
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Text == idleWrapNotice {
			found = true
		}
	}
	if !found {
		t.Fatalf("fallback recap must land in the transcript, chat=%+v", m.st.Chat)
	}
}

func TestIdleWrapSilentWhenBossWrapped(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-ask", From: "user", Text: "scan it",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteWorking, Task: "scan",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "b-wrap", From: "boss", Text: "scout is back — shift closed."}})
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "sco-1",
		Mail: state.MailItem{Kind: state.MailReturn, From: "skopos-1"}})
	if !m.ghostArmAt.IsZero() || m.ghostNoted {
		t.Fatal("a boss wrap during the shift must not arm idle-wrap")
	}
	m = pumpTicks(m, 3)
	if m.ghostNoted {
		t.Fatal("boss wrap → no recap")
	}
}

func TestIdleWrapSilentWhenOfficeConciergeSpoke(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-ask", From: "user", Text: "status?",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking,
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatOffice, Msg: state.ChatMsg{
		ID: "office-1", From: "office", Kind: "office", Text: "workers are on it.",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "dev-1",
		Mail: state.MailItem{Kind: state.MailReturn, From: "tekton-1"}})
	if !m.ghostArmAt.IsZero() || m.ghostNoted {
		t.Fatal("a concierge wrap must not arm idle-wrap")
	}
}

func TestIdleWrapSkipsLocalOfficeNoticeAsWrap(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-ask", From: "user", Text: "do the thing",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking,
	}})
	m.notice("queued as item #1 — flushes as a batch when the boss frees up")
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "dev-1",
		Mail: state.MailItem{Kind: state.MailReturn, From: "tekton-1"}})
	if m.ghostArmAt.IsZero() {
		t.Fatal("a dim office notice must not count as a wrap — last real chat is still the user")
	}
}

func TestIdleWrapSilentWithNoRealChat(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking,
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "dev-1",
		Mail: state.MailItem{Kind: state.MailReturn, From: "tekton-1"}})
	if !m.ghostArmAt.IsZero() {
		t.Fatal("no user/boss/office chat → do not recap an empty floor")
	}
}

func TestIdleWrapSilentWhileWorkersOrBossPending(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
		ID: "u-ask", From: "user", Text: "go",
	}})
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking,
	}})
	m.ghostArmAt = time.Now().Add(-bossWedgeAfter - time.Second)
	m = pumpTicks(m, 2)
	if m.ghostNoted {
		t.Fatal("a live developer must not fire idle-wrap")
	}

	m2, _ := wedgeFixture(t)
	m2.ghostArmAt = time.Now().Add(-bossWedgeAfter - time.Second)
	m2 = pumpTicks(m2, 2)
	if m2.ghostNoted {
		t.Fatal("a pending boss turn is W1, not idle-wrap")
	}
}

func TestIdleWrapSilentOnBoot(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m = pumpTicks(m, 2)
	if m.ghostNoted || !m.ghostArmAt.IsZero() {
		t.Fatal("boot idle with no shift must not arm or fire idle-wrap")
	}
}

func TestIdleWrapRearmsOnNextShift(t *testing.T) {
	m := idleWrapFixture(t)
	m.ghostArmAt = time.Now().Add(-bossWedgeAfter - time.Second)
	if cmd := m.checkIdleWrap(); cmd == nil {
		t.Fatal("first quiet shift must fire")
	}
	m = runMsg(t, m, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-2", Name: "tekton-2", Role: state.RoleDeveloper, Sprite: state.SpriteWorking,
	}})
	if m.ghostNoted || !m.shiftBusy {
		t.Fatal("a new live worker must clear the latch and open a fresh shift")
	}
}
