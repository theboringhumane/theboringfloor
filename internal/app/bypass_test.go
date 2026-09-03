// bypass_test.go — the /bypass session-scoped bypass-permissions mode:
//
//   - STATE: m.bypassPerms is session-scoped ONLY — every boot (New) starts
//     OFF, nothing rides brain.json/session.json.
//   - SLASH: /bypass toggles. ENABLE routes through the office's question
//     popover as an explicit confirm (enable/cancel; cancel/esc = no-op);
//     DISABLE is instant. Both land the pinned transcript notice.
//   - INDICATOR: while armed the topbar carries the loud ⚠ BYPASS segment
//     (full bar's gap splice AND the compact bar's truncate fallback).
//   - STRAY ASKS: while armed a pending EvPermission is answered
//     allow-once on the backend's wire immediately — no modal parks —
//     with one dim transcript row.
//   - BROWSER GATE: while armed the office's OWN browser-action gate
//     executes immediately (no synthetic modal) + the pinned log row.
//   - RESPAWN: every toggle respawns the transport — Stop old, factory
//     build, PrimaryOverride re-pin (the resumed session),
//     SetBypassPermissions(value) BEFORE Start. A toggle mid-respawn
//     queues ONE follow-up behind the fresh boot line (no double-spawn).
package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/browsertools/action"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/headless"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// --- recording seams ---------------------------------------------------------

// (permRecBackend — the recording-AnswerPermission seam — already lives in
// perm_queue_test.go; this file reuses it.)

// bypassLog — one ordered cross-instance event line per call (the
// respawn's ORDER is the contract: stop → override → bypass → start).
type bypassLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *bypassLog) add(s string) {
	l.mu.Lock()
	l.lines = append(l.lines, s)
	l.mu.Unlock()
}

func (l *bypassLog) get() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...)
}

// bypassStubBackend — a LIVE-mode recording backend with the primary +
// bypass seams (the respawn contract: SetBypassPermissions pre-Start).
type bypassStubBackend struct {
	*swapStubBackend
	name string
	log  *bypassLog

	bypassCalls  []bool
	startEntered chan struct{}
	startRelease chan struct{}
	startErr     error
	spawned      bool
}

func newBypassStub(name, primary string, log *bypassLog) *bypassStubBackend {
	return &bypassStubBackend{swapStubBackend: newSwapStub(primary), name: name, log: log}
}

// newBypassModel — newSwapModel's twin for the bypass stub (New takes the
// state.Backend interface; only the helper's signature is concrete).
func newBypassModel(b *bypassStubBackend, cfg *config.Config, dir string) Model {
	m := New(b, cfg)
	m.bootDone = true
	m.sessDir = dir
	return m
}

// bypassSoundLog records the reducer-owned sound hook so the interactive
// Enter regression can distinguish a delivered prompt from the historical
// rejected prompt/error-chime path.
type bypassSoundLog struct{ names []string }

func (s *bypassSoundLog) Play(name string) { s.names = append(s.names, name) }

// enterBypassChat drives the same character and Enter messages Bubble Tea
// receives from a member, rather than calling currentBackendSend directly.
func enterBypassChat(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
	}
	return runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
}

func armBypassReplacement(t *testing.T) (m Model, old, fresh *bypassStubBackend) {
	t.Helper()
	scratchHome(t)
	log := &bypassLog{}
	old = newBypassStub("old", "ses-live-1", log)
	fresh = newBypassStub("fresh", "", log)
	bypassFactory(t, fresh)
	m = newBypassModel(old, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	m = answerConfirm(t, m, "enable")
	if m.backend != fresh || !m.bypassPerms || m.bypassRestarting {
		t.Fatalf("bypass activation did not finish on fresh backend: backend=%T on=%v restarting=%v", m.backend, m.bypassPerms, m.bypassRestarting)
	}
	return m, old, fresh
}

func (b *bypassStubBackend) Stop() error {
	b.log.add(b.name + ":stop")
	return b.swapStubBackend.Stop()
}

func (b *bypassStubBackend) Start(emit func(state.Event)) error {
	b.log.add(b.name + ":start")
	// Model the OpenCode failure window: a child may exist before resolving
	// its primary fails and returns the Start error.
	b.mu.Lock()
	b.spawned = true
	b.mu.Unlock()
	if b.startEntered != nil {
		close(b.startEntered)
	}
	if b.startRelease != nil {
		<-b.startRelease
	}
	if b.startErr != nil {
		return b.startErr
	}
	return b.swapStubBackend.Start(emit)
}

func (b *bypassStubBackend) didSpawn() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spawned
}

func (b *bypassStubBackend) PrimaryOverride(id string) {
	b.log.add(b.name + ":override:" + id)
	b.swapStubBackend.PrimaryOverride(id)
}

func (b *bypassStubBackend) SetBypassPermissions(on bool) error {
	b.log.add(fmt.Sprintf("%s:bypass:%v", b.name, on))
	b.mu.Lock()
	b.bypassCalls = append(b.bypassCalls, on)
	b.mu.Unlock()
	return nil
}

func (b *bypassStubBackend) bypasses() []bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]bool(nil), b.bypassCalls...)
}

// bypassFactory installs a BackendFactory handing out builds IN ORDER
// (any name). Returns the call log.
func bypassFactory(t *testing.T, builds ...*bypassStubBackend) *[]string {
	t.Helper()
	calls := &[]string{}
	seq := 0
	old := BackendFactory
	BackendFactory = func(name, baseURL, dir string, cfg *config.Config) state.Backend {
		*calls = append(*calls, name)
		if seq < len(builds) {
			b := builds[seq]
			seq++
			return b
		}
		seq++
		return newBypassStub(fmt.Sprintf("extra-%d", seq), "", builds[0].log)
	}
	t.Cleanup(func() { BackendFactory = old })
	return calls
}

// answerConfirm drives the open /bypass confirm popover's answer through
// the SAME seam the chat panel's popover fires (questionAnswerMsg).
func answerConfirm(t *testing.T, m Model, picks ...string) Model {
	t.Helper()
	return runMsg(t, m, questionAnswerMsg{ans: panels.QuestionAnswer{Picks: picks}})
}

// --- state -------------------------------------------------------------------

func TestBypassStateResetsPerBoot(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, nil)
	if m.bypassPerms || m.bypassRestarting || m.bypassQueued {
		t.Fatal("every boot must start with the bypass mode fully OFF")
	}
	// a model that HAD the mode armed never leaks it into the next boot.
	m.bypassPerms = true
	m.bypassRestarting = true
	m2 := New(&recBackend{}, nil)
	if m2.bypassPerms || m2.bypassRestarting || m2.bypassQueued {
		t.Fatal("a fresh boot must never inherit the previous session's bypass state")
	}
}

// --- the confirm modal --------------------------------------------------------

func TestBypassEnableConfirmCancelAndEsc(t *testing.T) {
	scratchHome(t)

	// cancel pick: no-op — mode stays OFF, zero notices, zero respawn.
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	if m.question == nil || m.question.IDs[0] != bypassConfirmID {
		t.Fatalf("/bypass must open the arming confirm, got %+v", m.question)
	}
	// the card wraps the prompt across rows — pin the head fragment + the
	// two answer rows (the pinned copy's presence in the popover).
	if plain := ansi.Strip(m.Frame()); !strings.Contains(plain, "Enable bypass permissions?") ||
		!strings.Contains(plain, "browser actions WITHOUT asking") || !strings.Contains(plain, "enable") ||
		!strings.Contains(plain, "cancel") {
		t.Fatalf("the confirm carries the pinned prompt + answers, frame:\n%s", plain)
	}
	before := len(m.st.Chat)
	m = answerConfirm(t, m, "cancel")
	if m.question != nil || m.bypassPerms {
		t.Fatalf("cancel must close the confirm and keep the mode OFF (question=%+v bypass=%v)", m.question, m.bypassPerms)
	}
	if len(m.st.Chat) != before {
		t.Fatalf("cancel is a no-op — no transcript rows, got %d new", len(m.st.Chat)-before)
	}

	// esc: same no-op, and the hold NEVER parks into /question.
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	if m.question == nil {
		t.Fatal("the confirm re-opens on the next /bypass")
	}
	m = runMsg(t, m, questionLaterMsg{})
	if m.question != nil || m.questionEscd != nil || m.bypassPerms {
		t.Fatalf("esc cancels outright (question=%+v escd=%+v bypass=%v)", m.question, m.questionEscd, m.bypassPerms)
	}
	if len(m.st.Chat) != before {
		t.Fatalf("esc-cancel is a no-op — no transcript rows, got %d new", len(m.st.Chat)-before)
	}
}

func TestBypassConfirmNeverFoldsBossQuestion(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	if m.question == nil || m.question.IDs[0] != bypassConfirmID {
		t.Fatal("setup: the confirm must be open")
	}
	// a REAL boss question lands behind the confirm: the confirm cancels
	// (no-op) and the boss question opens as its own hold — never folded
	// onto the office-local sentinel.
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, EmployeeName: "boss",
		QuestionID: "q-boss-1", Text: "ship it?", ToolSummary: "yes | no"})
	if m.question == nil || m.question.IDs[0] != "q-boss-1" {
		t.Fatalf("the boss question must open as its own hold, got %+v", m.question)
	}
	if m.bypassPerms {
		t.Fatal("the boss question preempting the confirm must leave the mode OFF")
	}
}

// --- enable accept + respawn ---------------------------------------------------

func TestBypassEnableAcceptRespawnsWithFlag(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	oldStub := newBypassStub("old", "ses-live-1", log)
	fresh := newBypassStub("fresh", "", log)
	calls := bypassFactory(t, fresh)
	m := newBypassModel(oldStub, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})

	m = runMsg(t, m, slashMsg{text: "/bypass"})
	if m.question == nil {
		t.Fatal("/bypass must open the confirm")
	}
	m = answerConfirm(t, m, "enable")
	fresh.waitStarted(t, "fresh transport")

	if !m.bypassPerms {
		t.Fatal("enable arms the mode")
	}
	if got := lastOfficeText(t, m); got != bypassOnNotice {
		t.Fatalf("the ON notice is the pinned row, got %q", got)
	}
	if len(*calls) != 1 {
		t.Fatalf("exactly one factory build, got %v", *calls)
	}
	if got := fresh.bypasses(); len(got) != 1 || got[0] != true {
		t.Fatalf("the FRESH instance got SetBypassPermissions(true), got %v", got)
	}
	// The old generation stays accepting until fresh Start succeeds, then drains.
	want := []string{"fresh:override:ses-live-1", "fresh:bypass:true", "fresh:start", "old:stop"}
	if got := log.get(); strings.Join(got, ";") != strings.Join(want, ";") {
		t.Fatalf("respawn ordering:\n got %v\nwant %v", got, want)
	}
	if m.bypassRestarting {
		t.Fatal("successful Start must complete the lifecycle without waiting for a boot status line")
	}
	if !strings.Contains(m.st.StatusLine, "backend: bypass permissions on") {
		t.Fatalf("the active status must replace the restart notice, got %q", m.st.StatusLine)
	}
}

func TestBypassDisableInstantNoConfirm(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	oldStub := newBypassStub("old", "ses-live-1", log)
	fresh := newBypassStub("fresh", "", log)
	bypassFactory(t, fresh)
	m := newBypassModel(oldStub, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m.bypassPerms = true // armed session

	m = runMsg(t, m, slashMsg{text: "/bypass"})
	fresh.waitStarted(t, "disable respawn")
	if m.bypassPerms {
		t.Fatal("disable must be instant")
	}
	if m.question != nil {
		t.Fatalf("disable asks NO confirm, got %+v", m.question)
	}
	if got := lastOfficeText(t, m); got != bypassOffNotice {
		t.Fatalf("the OFF notice is the pinned row, got %q", got)
	}
	if got := fresh.bypasses(); len(got) != 1 || got[0] != false {
		t.Fatalf("the fresh instance got SetBypassPermissions(false), got %v", got)
	}
	// The same holder used by the chat callback must now point at the OFF
	// replacement, not at the just-retired ON backend.
	if msg := currentBackendSend(m.currentBackend, m.plan, "after bypass off", nil)(); msg == nil {
		t.Fatal("ordinary Enter after bypass OFF returned nil")
	} else if _, ok := msg.(chatSentMsg); !ok {
		t.Fatalf("ordinary Enter after bypass OFF = %T, want chatSentMsg", msg)
	}
	if got := oldStub.sentTexts; len(got) != 0 {
		t.Fatalf("retired ON backend send ledger = %v, want no calls", got)
	}
	if got := fresh.sentTexts; !reflect.DeepEqual(got, []string{"after bypass off"}) {
		t.Fatalf("OFF replacement send ledger = %v", got)
	}
}

// --- indicator -----------------------------------------------------------------

func TestBypassIndicatorInFrame(t *testing.T) {
	scratchHome(t)
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	off := m.Frame()
	if strings.Contains(ansi.Strip(off), "⚠ BYPASS") {
		t.Fatal("OFF: the topbar carries no bypass segment")
	}

	m.bypassPerms = true // the digest pins the flag — the repaint is free
	on := m.Frame()
	plain := ansi.Strip(on)
	if !strings.Contains(plain, "⚠ BYPASS") {
		t.Fatalf("ON: the topbar must carry the loud segment, top row:\n%s", strings.Split(plain, "\n")[0])
	}
	// the badge lives on the TOPBAR row (row 0), not somewhere mid-frame.
	if top := strings.Split(plain, "\n")[0]; !strings.Contains(top, "⚠ BYPASS") {
		t.Fatalf("the segment rides the topbar row, got %q", top)
	}

	// the compact bar's fallback path: no gap run → trailing cells donate.
	m.compactLive = 1
	compact := ansi.Strip(m.Frame())
	if !strings.Contains(strings.Split(compact, "\n")[0], "⚠ BYPASS") {
		t.Fatalf("compact: the segment survives the narrow bar, top row:\n%s", strings.Split(compact, "\n")[0])
	}
}

// --- stray-ask auto-answer ------------------------------------------------------

func TestBypassStrayAskAutoAnswer(t *testing.T) {
	scratchHome(t)
	b := &permRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.bypassPerms = true

	m = runMsg(t, m, state.Event{Kind: state.EvPermission, EmployeeName: "boss",
		PermissionID: "perm-stray-1", ToolName: "bash", ToolSummary: "rm -rf /tmp/x", ToolState: "pending"})

	if len(b.answered) != 1 || b.answered[0] != [2]string{"perm-stray-1", "once"} {
		t.Fatalf("the stray ask must auto-answer allow-once on the wire, got %v", b.answered)
	}
	if m.permQ.front() != nil || len(m.permQ.pending) != 0 || len(m.permQ.escd) != 0 {
		t.Fatalf("NO modal parks while armed: %+v (escd %d)", m.permQ.view(), len(m.permQ.escd))
	}
	if got := lastOfficeText(t, m); got != "bypass: auto-approved bash" {
		t.Fatalf("the dim auto-approval row is pinned, got %q", got)
	}

	// a resolved event still routes the existing way (defensive: a wire
	// resolution racing the arm).
	m.permQ.pending = []*permPrompt{{ID: "perm-old", ToolName: "write"}}
	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-old", ToolState: "resolved"})
	if len(m.permQ.pending) != 0 {
		t.Fatalf("resolved still drains the parked ask: %+v", m.permQ.pending)
	}

	// mode OFF: the same ask parks the modal exactly like before.
	m2 := New(&permRecBackend{}, nil)
	m2 = runMsg(t, m2, tea.WindowSizeMsg{Width: 140, Height: 30})
	m2 = runMsg(t, m2, state.Event{Kind: state.EvPermission, EmployeeName: "boss",
		PermissionID: "perm-normal-1", ToolName: "bash", ToolSummary: "make x", ToolState: "pending"})
	if m2.permQ.front() == nil {
		t.Fatal("mode OFF keeps the modal flow: the ask parks")
	}
}

// --- browser-action gate ---------------------------------------------------------

func TestBypassBrowserActionSkipsModal(t *testing.T) {
	scratchHome(t)
	var ran []string
	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		ran = append(ran, a.Op+":"+a.Sel)
		return &action.Result{URL: rawurl, Text: "clicked #buy"}, nil
	})

	b := &permRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.bypassPerms = true

	m = runMsg(t, m, state.Event{Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy"})

	if len(ran) != 1 || ran[0] != "click:#buy" {
		t.Fatalf("the action executes IMMEDIATELY while armed, ran %v", ran)
	}
	if m.permQ.view() != nil || len(m.browserActionHolds) != 0 {
		t.Fatalf("no synthetic modal, no parked hold: %+v / %+v", m.permQ.view(), m.browserActionHolds)
	}
	var sawLog bool
	for _, c := range m.st.Chat {
		if c.From == "office" && c.Text == "bypass: browser action auto-approved — click '#buy'" {
			sawLog = true
		}
	}
	if !sawLog {
		t.Fatalf("the pinned log row must land, chat tail: %+v", m.st.Chat[len(m.st.Chat)-3:])
	}
	// the outcome still posts BACK to the agent on the same session.
	if len(b.sentTexts) != 1 || !strings.Contains(b.sentTexts[0], "browser-action ok on https://theboring.name: click '#buy'") {
		t.Fatalf("the agent follow-up rides the send path, got %v", b.sentTexts)
	}
}

// --- queue-while-restarting -------------------------------------------------------

func TestBypassToggleQueuesBehindRespawn(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	oldStub := newBypassStub("old", "ses-live-1", log)
	fresh1 := newBypassStub("fresh1", "", log)
	fresh2 := newBypassStub("fresh2", "", log)
	calls := bypassFactory(t, fresh1, fresh2)
	m := newBypassModel(oldStub, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})

	// enable → respawn #1 starts (latch armed, no boot line yet).
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	m = answerConfirm(t, m, "enable")
	fresh1.waitStarted(t, "respawn #1")
	if len(*calls) != 1 {
		t.Fatalf("one build so far, got %v", *calls)
	}

	// The normal synchronous fake completes Start before this next input, so
	// disable starts a second, independent transition.
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	if m.bypassPerms {
		t.Fatal("disable commits OFF after its replacement starts")
	}
	if len(*calls) != 2 {
		t.Fatalf("disable constructs its fresh OFF backend, got %v builds", *calls)
	}
	fresh2.waitStarted(t, "respawn #2")
	if got := fresh2.bypasses(); len(got) != 1 || got[0] != false {
		t.Fatalf("the follow-up's fresh instance got SetBypassPermissions(false), got %v", got)
	}
	if m.bypassRestarting || m.bypassQueued {
		t.Fatalf("completed transition latches = restarting:%v queued:%v", m.bypassRestarting, m.bypassQueued)
	}
}

func TestBypassRespawnLatchClearsOnBootFailure(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	oldStub := newBypassStub("old", "ses-live-1", log)
	fresh1 := newBypassStub("fresh1", "", log)
	fresh2 := newBypassStub("fresh2", "", log)
	fresh1.startErr = errors.New("boom")
	calls := bypassFactory(t, fresh1, fresh2)
	m := newBypassModel(oldStub, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})

	m = runMsg(t, m, slashMsg{text: "/bypass"})
	m = answerConfirm(t, m, "enable")
	if m.bypassRestarting {
		t.Fatal("a Start failure clears the lifecycle latch (never a wedged gate)")
	}
	// the next toggle respawns again — the gate works.
	m = runMsg(t, m, slashMsg{text: "/bypass"})
	m = answerConfirm(t, m, "enable")
	fresh2.waitStarted(t, "respawn #2")
	if len(*calls) != 2 {
		t.Fatalf("the post-failure toggle respawns normally, got %v", *calls)
	}
}

func TestBypassRapidToggleQueuesWhileFactoryIsBlocked(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	old := newBypassStub("old", "ses-live-1", log)
	freshOn := newBypassStub("fresh-on", "", log)
	freshOff := newBypassStub("fresh-off", "", log)
	entered := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	builds := 0
	oldFactory := BackendFactory
	BackendFactory = func(string, string, string, *config.Config) state.Backend {
		mu.Lock()
		builds++
		build := builds
		mu.Unlock()
		if build == 1 {
			close(entered)
			<-release
			return freshOn
		}
		return freshOff
	}
	t.Cleanup(func() { BackendFactory = oldFactory })

	m := newBypassModel(old, config.Default(), t.TempDir())
	m.bypassPerms = true
	m.bypassDesired = true
	first := m.respawnForBypass()
	if !m.bypassRestarting {
		t.Fatal("the bypass latch must arm before the factory command runs")
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- first() }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first factory did not block")
	}

	// This is the second rapid /bypass toggle: it changes the desired value,
	// but must not construct a competing backend while the first build runs.
	m.bypassDesired = false
	if cmd := m.respawnForBypass(); cmd != nil {
		t.Fatal("second toggle must queue, not schedule another factory command")
	}
	if !m.bypassQueued {
		t.Fatal("second toggle must record one queued desired state")
	}
	mu.Lock()
	gotBuilds := builds
	mu.Unlock()
	if gotBuilds != 1 {
		t.Fatalf("factory builds while first is blocked = %d, want 1", gotBuilds)
	}

	close(release)
	m = runMsg(t, m, <-result)
	mu.Lock()
	gotBuilds = builds
	mu.Unlock()
	if gotBuilds != 2 {
		t.Fatalf("queued final state must construct exactly once after the first build, got %d builds", gotBuilds)
	}
	if got := freshOn.bypasses(); len(got) != 1 || got[0] != true {
		t.Fatalf("first construction must retain its captured ON value, got %v", got)
	}
	if got := freshOff.bypasses(); len(got) != 1 || got[0] != false {
		t.Fatalf("queued construction must apply final OFF value once, got %v", got)
	}
	if got := freshOn.stopCount(); got != 1 {
		t.Fatalf("rapid stale candidate Stop calls = %d, want 1", got)
	}
	if m.backend != freshOff || m.bypassRestarting || m.bypassQueued {
		t.Fatalf("final backend/latches = backend:%T restarting:%v queued:%v, want fresh-off/false/false", m.backend, m.bypassRestarting, m.bypassQueued)
	}
	attachment := state.Attachment{Name: "handoff.pdf", Mime: "application/pdf", Path: "/tmp/handoff.pdf"}
	if msg := currentBackendSend(m.currentBackend, m.plan, "after rapid toggles", []state.Attachment{attachment})(); msg == nil {
		t.Fatal("ordinary Enter after rapid toggles returned nil")
	} else if _, ok := msg.(chatSentMsg); !ok {
		t.Fatalf("ordinary Enter after rapid toggles = %T, want chatSentMsg", msg)
	}
	if got := old.sentTexts; len(got) != 0 {
		t.Fatalf("original backend send ledger = %v, want no calls", got)
	}
	if got := freshOn.sentTexts; len(got) != 0 {
		t.Fatalf("discarded ON candidate send ledger = %v, want no calls", got)
	}
	if got := freshOff.sentTexts; !reflect.DeepEqual(got, []string{"after rapid toggles"}) || len(freshOff.sentAtts) != 1 || len(freshOff.sentAtts[0]) != 1 || freshOff.sentAtts[0][0] != attachment {
		t.Fatalf("final OFF backend send/attachment ledger = texts:%v atts:%+v", got, freshOff.sentAtts)
	}
}

func TestBypassSlowStartKeepsOldBackendAndActiveIndicator(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	old := newBypassStub("old", "ses-live-1", log)
	fresh := newBypassStub("fresh", "", log)
	fresh.startEntered = make(chan struct{})
	fresh.startRelease = make(chan struct{})
	bypassFactory(t, fresh)
	m := newBypassModel(old, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	m.bypassDesired = true

	build := m.respawnForBypass()
	buildMsg := build()
	next, start := m.Update(buildMsg)
	m = next.(Model)
	started := make(chan tea.Msg, 1)
	go func() { started <- start() }()
	select {
	case <-fresh.startEntered:
	case <-time.After(time.Second):
		t.Fatal("fresh Start did not block")
	}
	if !m.bypassRestarting || m.bypassPerms || m.backend != old {
		t.Fatalf("while starting: restarting=%v active=%v backend=%T; want true/false/old", m.bypassRestarting, m.bypassPerms, m.backend)
	}
	if !strings.Contains(m.st.StatusLine, "backend restarting — bypass permissions on") {
		t.Fatalf("transient restart status = %q", m.st.StatusLine)
	}
	if strings.Contains(ansi.Strip(m.Frame()), "⚠ BYPASS") {
		t.Fatal("badge must describe active OFF backend during an ON start")
	}
	sendThroughCurrentBackend(t, &m, "old remains usable", nil)

	close(fresh.startRelease)
	m = runMsg(t, m, <-started)
	if m.bypassRestarting || !m.bypassPerms || m.backend != fresh {
		t.Fatalf("after start: restarting=%v active=%v backend=%T; want false/true/fresh", m.bypassRestarting, m.bypassPerms, m.backend)
	}
	if !strings.Contains(ansi.Strip(m.Frame()), "⚠ BYPASS") {
		t.Fatal("badge must turn ON only after the ON backend starts")
	}
}

func TestBypassCommitRoutesOrdinarySendsBeforeAndAfterOldCleanup(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	old := newBypassStub("old", "ses-live-1", log)
	fresh := newBypassStub("fresh", "", log)
	fresh.startEntered = make(chan struct{})
	fresh.startRelease = make(chan struct{})
	bypassFactory(t, fresh)
	m := newBypassModel(old, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m.bypassDesired = true

	// Drive build and Start separately so fresh is committed while old cleanup
	// remains delayed; this mirrors a user pressing Enter immediately after ON.
	build := m.respawnForBypass()
	next, start := m.Update(build())
	m = next.(Model)
	started := make(chan tea.Msg, 1)
	go func() { started <- start() }()
	select {
	case <-fresh.startEntered:
	case <-time.After(time.Second):
		t.Fatal("fresh Start did not begin")
	}
	close(fresh.startRelease)
	next, committed := m.Update(<-started)
	m = next.(Model)
	if m.backend != fresh || !m.bypassPerms || m.bypassRestarting {
		t.Fatalf("successful bypass commit = backend:%T on:%v restarting:%v, want fresh/true/false", m.backend, m.bypassPerms, m.bypassRestarting)
	}

	// With no status side-effect command, tea.Batch elides to the cleanup
	// command itself. Keep it unexecuted until after the first fresh send.
	cleanup := committed
	if cleanup == nil {
		t.Fatal("successful commit must retain old-generation cleanup")
	}

	before := currentBackendSend(m.currentBackend, m.plan, "before old cleanup", nil)()
	if _, ok := before.(chatSentMsg); !ok {
		t.Fatalf("pre-cleanup ordinary Enter = %T, want chatSentMsg", before)
	}
	m = runMsg(t, m, before)
	attachment := state.Attachment{Name: "notes.md", Mime: "text/markdown", Path: "/tmp/notes.md"}
	after := currentBackendSend(m.currentBackend, m.plan, "after old cleanup", []state.Attachment{attachment})()
	if _, ok := after.(chatSentMsg); !ok {
		t.Fatalf("post-cleanup ordinary Enter = %T, want chatSentMsg", after)
	}
	m = runMsg(t, m, cleanup())
	m = runMsg(t, m, after)

	if got := old.stopCount(); got != 1 {
		t.Fatalf("old backend Stop calls = %d, want 1", got)
	}
	if got := fresh.stopCount(); got != 0 {
		t.Fatalf("fresh backend Stop calls = %d, want 0", got)
	}
	if got := old.sentTexts; len(got) != 0 {
		t.Fatalf("old backend send ledger = %v, want no calls", got)
	}
	if got := fresh.sentTexts; !reflect.DeepEqual(got, []string{"before old cleanup", "after old cleanup"}) {
		t.Fatalf("fresh backend send ledger = %v", got)
	}
	if got := fresh.sentAtts; len(got) != 2 || len(got[0]) != 0 || len(got[1]) != 1 || got[1][0] != attachment {
		t.Fatalf("fresh attachment ledger = %+v, want only the second SendWith to carry %+v", got, attachment)
	}

	// Send acceptance makes no duplicate local bubbles. The backend's normal
	// user/pending events still update the post-transition transcript.
	chatBefore := len(m.st.Chat)
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{From: "user", Text: "after old cleanup"}})
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{From: "boss", Pending: true}})
	if len(m.st.Chat) != chatBefore+2 || m.st.Chat[len(m.st.Chat)-2].From != "user" || !m.st.Chat[len(m.st.Chat)-1].Pending {
		t.Fatalf("post-commit user/pending lifecycle = %+v", m.st.Chat[chatBefore:])
	}
}

func TestBypassFactoryAndStartFailuresKeepOldBackendUsable(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	old := newBypassStub("old", "ses-live-1", log)
	oldFactory := BackendFactory
	BackendFactory = func(string, string, string, *config.Config) state.Backend { return nil }
	t.Cleanup(func() { BackendFactory = oldFactory })
	m := newBypassModel(old, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m.bypassDesired = true
	m = runMsg(t, m, m.respawnForBypass()())
	if m.bypassRestarting || m.bypassPerms || m.backend != old {
		t.Fatalf("factory failure changed active state: restarting=%v active=%v backend=%T", m.bypassRestarting, m.bypassPerms, m.backend)
	}
	sendThroughCurrentBackend(t, &m, "factory failure still sends", nil)

	failing := newBypassStub("failing", "", log)
	failing.startErr = errors.New("start boom")
	BackendFactory = func(string, string, string, *config.Config) state.Backend { return failing }
	m.bypassDesired = true
	m = runMsg(t, m, m.respawnForBypass()())
	if m.bypassRestarting || m.bypassPerms || m.backend != old || !strings.Contains(m.st.StatusLine, "active permissions off") {
		t.Fatalf("start failure changed active state/status: restarting=%v active=%v backend=%T status=%q", m.bypassRestarting, m.bypassPerms, m.backend, m.st.StatusLine)
	}
	if got := failing.stopCount(); got != 1 {
		t.Fatalf("partially started candidate Stop calls = %d, want 1", got)
	}
	if !failing.didSpawn() {
		t.Fatal("failing candidate must model a child spawned before Start returned its error")
	}
	if got := old.stopCount(); got != 0 {
		t.Fatalf("active backend Stop calls after candidate failure = %d, want 0", got)
	}
	sendThroughCurrentBackend(t, &m, "start failure still sends", nil)
}

func TestBypassStaleStartedCandidateStopsExactlyOnce(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	old := newBypassStub("old", "ses-live-1", log)
	stale := newBypassStub("stale", "", log)
	m := newBypassModel(old, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})
	m.bypassPerms = true
	m.bypassRestarting = true
	m.backendTransitioning = true
	m.backendTransitionID = 2

	// This result completed Start for the superseded transition. It must be
	// discarded without replacing or stopping the accepting generation.
	m = runMsg(t, m, backendStartMsg{result: backendBuildMsg{
		backend:     stale,
		bypass:      true,
		bypassValue: false,
		transition:  1,
	}, err: nil})
	if m.backend != old || !m.bypassPerms {
		t.Fatalf("stale candidate changed active backend/mode: backend=%T bypass=%v", m.backend, m.bypassPerms)
	}
	if got := stale.stopCount(); got != 1 {
		t.Fatalf("stale started candidate Stop calls = %d, want 1", got)
	}
	if got := old.stopCount(); got != 0 {
		t.Fatalf("active backend Stop calls = %d, want 0", got)
	}
	if m.bypassRestarting || m.backendTransitioning {
		t.Fatalf("stale result must clear latches: restarting=%v transitioning=%v", m.bypassRestarting, m.backendTransitioning)
	}
	sendThroughCurrentBackend(t, &m, "old remains usable after stale candidate", nil)
}

func TestBypassRespawnSkippedInDemo(t *testing.T) {
	scratchHome(t)
	b := &permRecBackend{}
	m := New(b, nil) // demo backend
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	m = runMsg(t, m, slashMsg{text: "/bypass"})
	m = answerConfirm(t, m, "enable")
	if !m.bypassPerms {
		t.Fatal("demo: the office-side mode still arms")
	}
	if m.bypassRestarting || m.bypassQueued {
		t.Fatal("demo: no respawn latch — the scripted tour's backend is fixed")
	}
	if got := lastOfficeText(t, m); got != bypassOnNotice {
		t.Fatalf("demo: the ON notice lands verbatim, got %q", got)
	}
}

// TestBypassInteractiveEnterReachesReplacement is the v0.3.22 regression:
// after the member confirms /bypass and the replacement has finished Start,
// a real composer Enter must use the fresh accepting generation. Previously
// every Enter took the unavailable branch, adding a red send-failed row and
// playing the error sound.
func TestBypassInteractiveEnterReachesReplacement(t *testing.T) {
	m, old, fresh := armBypassReplacement(t)
	sounds := &bypassSoundLog{}
	m.SetSoundBus(sounds)

	m = enterBypassChat(t, m, "deliver after bypass")
	if got := fresh.sentTexts; !reflect.DeepEqual(got, []string{"deliver after bypass"}) {
		t.Fatalf("fresh backend send ledger = %v, want delivered interactive Enter", got)
	}
	if got := old.sentTexts; len(got) != 0 {
		t.Fatalf("retired backend send ledger = %v, want no calls", got)
	}
	if got := sounds.names; !reflect.DeepEqual(got, []string{"send"}) {
		t.Fatalf("interactive bypass sound sequence = %v, want [send] (no error)", got)
	}
	for _, row := range m.st.Chat {
		if row.Meta == "error" && strings.Contains(row.Text, "send failed") {
			t.Fatalf("interactive bypass Enter must not add a send failure row: %+v", row)
		}
	}
}

// TestBypassBusyQueueAndResendReachReplacement is the sibling of interactive
// Enter: busy free-send, queue flush, and batch resend must also lease the
// replacement instead of the backend captured when the cmd was built.
func TestBypassBusyQueueAndResendReachReplacement(t *testing.T) {
	m, old, fresh := armBypassReplacement(t)
	m.st.BossDelegating = true
	m = runMsg(t, m, busySendReqMsg{text: "busy after bypass"})
	m = runMsg(t, m, enqueueMsg{text: "queued after bypass", atts: nil})
	m = runMsg(t, m, queueFlushMsg{})
	m = runMsg(t, m, m.resendBatchCmd([]queueEntry{{text: "batch after bypass"}})())
	if got := old.sentTexts; len(got) != 0 {
		t.Fatalf("retired backend send ledger = %v, want no calls", got)
	}
	if got := fresh.sentTexts; !reflect.DeepEqual(got, []string{
		"busy after bypass",
		"queued after bypass",
		composeBatch([]queueEntry{{text: "batch after bypass"}}),
	}) {
		t.Fatalf("fresh backend send ledger = %v, want busy + flush + resend", got)
	}
}

// TestInteractiveEnterWithoutActiveBackendStillRejects is the paired control:
// no active generation is a real failure, so Enter must keep its visible error
// row and error sound rather than being silently accepted.
func TestInteractiveEnterWithoutActiveBackendStillRejects(t *testing.T) {
	scratchHome(t)
	log := &bypassLog{}
	m := newBypassModel(newBypassStub("live", "ses-live-1", log), config.Default(), t.TempDir())
	sounds := &bypassSoundLog{}
	m.SetSoundBus(sounds)
	// The chat callback captured this holder at New time; making its current
	// generation unavailable models a genuine transport blackout.
	m.currentBackend.current = nil

	m = enterBypassChat(t, m, "must remain rejected")
	if got := sounds.names; !reflect.DeepEqual(got, []string{"error"}) {
		t.Fatalf("unavailable interactive Enter sound sequence = %v, want [error]", got)
	}
	if len(m.st.Chat) == 0 {
		t.Fatal("unavailable interactive Enter must append a send failure row")
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" || !strings.Contains(row.Text, "send failed: active backend unavailable") {
		t.Fatalf("unavailable interactive Enter row = %+v, want active-backend send failure", row)
	}
}

func TestBypassQuestionMCPBrowserReachReplacement(t *testing.T) {
	m, old, fresh := armBypassReplacement(t)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "q-after",
		EmployeeName: "boss", ToolState: "pending", Text: "ship?"})
	m = runMsg(t, m, questionAnswerMsg{ans: panels.QuestionAnswer{Text: "yes"}})
	if len(old.qAnswers) != 0 {
		t.Fatalf("retired AnswerQuestion = %+v, want none", old.qAnswers)
	}
	if len(fresh.qAnswers) != 1 || fresh.qAnswers[0].id != "q-after" {
		t.Fatalf("fresh AnswerQuestion = %+v, want q-after", fresh.qAnswers)
	}

	m = runMsg(t, m, slashMsg{text: "/mcp"})
	m = runMsg(t, m, slashMsg{text: "/mcp reconnect alpha"})
	if old.mcpCalls != 0 || len(old.mcpReconned) != 0 {
		t.Fatalf("retired MCP hops calls=%d reconn=%v", old.mcpCalls, old.mcpReconned)
	}
	if fresh.mcpCalls != 2 || !reflect.DeepEqual(fresh.mcpReconned, []string{"alpha"}) {
		t.Fatalf("fresh MCP hops calls=%d reconn=%v, want 2 + [alpha]", fresh.mcpCalls, fresh.mcpReconned)
	}

	m = runMsg(t, m, state.Event{Kind: state.EvPermission, EmployeeName: "boss",
		PermissionID: "perm-after", ToolName: "bash", ToolSummary: "ls", ToolState: "pending"})
	if len(old.permAnswers) != 0 {
		t.Fatalf("retired AnswerPermission = %v, want none", old.permAnswers)
	}
	if !reflect.DeepEqual(fresh.permAnswers, [][2]string{{"perm-after", "once"}}) {
		t.Fatalf("fresh AnswerPermission = %v, want allow-once on replacement", fresh.permAnswers)
	}

	pinFakeBrowserEngines(t, nil, func(ctx context.Context, rawurl string, maxText int) (*headless.SnapResult, error) {
		return &headless.SnapResult{URL: rawurl, Title: "t", Text: "body"}, nil
	})
	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserSnapshot, Text: "https://theboring.name/docs", BrowserOpenAllowed: true,
	})
	if cmd == nil {
		t.Fatal("allowed snapshot must return the engine cmd")
	}
	m = drainBrowserCmd(t, m, cmd)
	if len(old.sentTexts) != 0 {
		t.Fatalf("retired snapshot send = %v, want none", old.sentTexts)
	}
	if len(fresh.sentTexts) != 1 || !strings.Contains(fresh.sentTexts[0], "snapshot of https://theboring.name/docs") {
		t.Fatalf("fresh snapshot send = %v", fresh.sentTexts)
	}

	pinFakeBrowserAction(t, func(ctx context.Context, rawurl string, a action.Action) (*action.Result, error) {
		return &action.Result{URL: rawurl, Text: "ok"}, nil
	})
	m = runMsg(t, m, state.Event{Kind: state.EvBrowserAction, Text: "https://theboring.name",
		BrowserOpenAllowed: true, BrowserActionOp: "click", BrowserActionSel: "#buy"})
	if len(old.sentTexts) != 0 {
		t.Fatalf("retired action send = %v, want none", old.sentTexts)
	}
	if len(fresh.sentTexts) != 2 || !strings.Contains(fresh.sentTexts[1], "browser-action ok") {
		t.Fatalf("fresh action send = %v", fresh.sentTexts)
	}
}
