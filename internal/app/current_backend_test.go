// current_backend_test.go pins ordinary chat dispatch to the backend that is
// current when Enter's tea.Cmd runs, rather than the backend New received.
package app

import (
	"errors"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

type currentBackendCall struct {
	text string
	atts []state.Attachment
}

type currentBackendStub struct {
	*swapStubBackend
	mu    sync.Mutex
	calls []currentBackendCall
}

type blockingCurrentBackend struct {
	*currentBackendStub
	entered chan struct{}
	release chan struct{}
	stopped chan struct{}
}

func newBlockingCurrentBackend(primary string) *blockingCurrentBackend {
	return &blockingCurrentBackend{
		currentBackendStub: newCurrentBackendStub(primary),
		entered:            make(chan struct{}),
		release:            make(chan struct{}),
		stopped:            make(chan struct{}),
	}
}

func (b *blockingCurrentBackend) Send(text string) error {
	return b.block(currentBackendCall{text: text})
}

func (b *blockingCurrentBackend) SendWith(text string, atts []state.Attachment) error {
	return b.block(currentBackendCall{text: text, atts: append([]state.Attachment(nil), atts...)})
}

func (b *blockingCurrentBackend) block(call currentBackendCall) error {
	b.mu.Lock()
	b.calls = append(b.calls, call)
	b.mu.Unlock()
	close(b.entered)
	<-b.release
	return nil
}

func (b *blockingCurrentBackend) Stop() error {
	if err := b.swapStubBackend.Stop(); err != nil {
		return err
	}
	close(b.stopped)
	return nil
}

func newCurrentBackendStub(primary string) *currentBackendStub {
	return &currentBackendStub{swapStubBackend: newSwapStub(primary)}
}

func (b *currentBackendStub) Send(text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, currentBackendCall{text: text})
	return nil
}

func (b *currentBackendStub) SendWith(text string, atts []state.Attachment) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, currentBackendCall{text: text, atts: append([]state.Attachment(nil), atts...)})
	return nil
}

func (b *currentBackendStub) sent() []currentBackendCall {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]currentBackendCall(nil), b.calls...)
}

// sendThroughCurrentBackend executes the exact tea.Cmd ordinary Enter gets
// from New's chat callback. It intentionally does not read m.backend.
func sendThroughCurrentBackend(t *testing.T, m *Model, text string, atts []state.Attachment) {
	t.Helper()
	if msg := currentBackendSend(m.currentBackend, m.plan, text, atts)(); msg == nil {
		t.Fatal("ordinary Enter must return a chat result")
	}
}

func runCurrentCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	return runMsg(t, m, cmd())
}

func requireCurrentCalls(t *testing.T, b *currentBackendStub, want ...string) {
	t.Helper()
	got := b.sent()
	if len(got) != len(want) {
		t.Fatalf("send ledger length: got %+v, want %v", got, want)
	}
	for i, text := range want {
		if got[i].text != text {
			t.Fatalf("send ledger[%d] text: got %q, want %q (ledger=%+v)", i, got[i].text, text, got)
		}
	}
}

func TestCurrentBackendInitialAndBackendReplacement(t *testing.T) {
	scratchHome(t)
	initial := newCurrentBackendStub("initial")
	replacement := newCurrentBackendStub("replacement")
	oldFactory := BackendFactory
	BackendFactory = func(string, string, string, *config.Config) state.Backend { return replacement }
	t.Cleanup(func() { BackendFactory = oldFactory })
	m := New(initial, config.Default())
	m.bootDone = true
	m.sessDir = t.TempDir()

	sendThroughCurrentBackend(t, &m, "before swap", nil)
	m = runCurrentCmd(t, m, m.swapBackend("claudecode"))
	sendThroughCurrentBackend(t, &m, "after swap", nil)

	requireCurrentCalls(t, initial, "before swap")
	requireCurrentCalls(t, replacement, "after swap")
}

func TestCurrentBackendBypassAndRapidReplacementKeepAttachments(t *testing.T) {
	scratchHome(t)
	first := newCurrentBackendStub("first")
	bypass := newCurrentBackendStub("bypass")
	last := newCurrentBackendStub("last")
	builds := []*currentBackendStub{bypass, last}
	next := 0
	oldFactory := BackendFactory
	BackendFactory = func(string, string, string, *config.Config) state.Backend {
		b := builds[next]
		next++
		return b
	}
	t.Cleanup(func() { BackendFactory = oldFactory })

	m := New(first, config.Default())
	m.bootDone = true
	m.sessDir = t.TempDir()

	// /bypass respawns a same-name live transport; ordinary Enter must land on
	// that replacement, including source-file attachments.
	m.bypassPerms = true
	m = runCurrentCmd(t, m, m.respawnForBypass())
	md := state.Attachment{Name: "notes.md", Mime: "text/markdown", Path: "/tmp/notes.md"}
	goFile := state.Attachment{Name: "model.go", Mime: "text/x-go", Path: "/tmp/model.go"}
	sendThroughCurrentBackend(t, &m, "after bypass", []state.Attachment{md, goFile})

	// A rapid second transition replaces the just-respawned backend; no
	// ordinary send may return to either retired transport.
	m = runCurrentCmd(t, m, m.swapBackend("claudecode"))
	sendThroughCurrentBackend(t, &m, "after rapid replacement", nil)

	requireCurrentCalls(t, first)
	requireCurrentCalls(t, bypass, "after bypass")
	requireCurrentCalls(t, last, "after rapid replacement")
	got := bypass.sent()[0].atts
	if len(got) != 2 || got[0].Name != "notes.md" || got[1].Name != "model.go" {
		t.Fatalf("current backend must receive both source attachments through SendWith, got %+v", got)
	}
}

func TestCurrentBackendReplacementDoesNotBlockOrReuseRetiredGeneration(t *testing.T) {
	old := newBlockingCurrentBackend("old")
	new := newCurrentBackendStub("new")
	current := newCurrentBackend(old)

	sent := make(chan error, 1)
	go func() { sent <- current.send("old send", nil, "") }()
	select {
	case <-old.entered:
	case <-time.After(time.Second):
		t.Fatal("old send did not begin")
	}

	started := time.Now()
	cleanup := current.replace(new)
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("replacement blocked behind old send for %s", elapsed)
	}
	if err := current.send("new send", nil, ""); err != nil {
		t.Fatalf("new generation send: %v", err)
	}
	requireCurrentCalls(t, new, "new send")
	requireCurrentCalls(t, old.currentBackendStub, "old send")

	stopped := make(chan tea.Msg, 1)
	go func() { stopped <- cleanup() }()
	select {
	case <-old.stopped:
		t.Fatal("old Stop began before its leased send released")
	default:
	}
	close(old.release)
	if err := <-sent; err != nil {
		t.Fatalf("old generation send: %v", err)
	}
	select {
	case <-old.stopped:
	case <-time.After(time.Second):
		t.Fatal("old Stop did not begin after send release")
	}
	if _, ok := (<-stopped).(backendStopMsg); !ok {
		t.Fatal("cleanup must return backendStopMsg")
	}
}

func TestCurrentBackendRapidReplacementStopsEachRetiredGenerationOnce(t *testing.T) {
	first := newCurrentBackendStub("first")
	second := newCurrentBackendStub("second")
	third := newCurrentBackendStub("third")
	current := newCurrentBackend(first)
	firstCleanup := current.replace(second)
	secondCleanup := current.replace(third)
	if firstCleanup == nil || secondCleanup == nil {
		t.Fatal("each retired generation needs a cleanup command")
	}
	firstCleanup()
	secondCleanup()
	if got := first.stopCount(); got != 1 {
		t.Fatalf("first retired generation Stop calls = %d, want 1", got)
	}
	if got := second.stopCount(); got != 1 {
		t.Fatalf("second retired generation Stop calls = %d, want 1", got)
	}
	requireCurrentCalls(t, first)
	requireCurrentCalls(t, second)
	if err := current.send("third send", nil, ""); err != nil {
		t.Fatalf("third generation send: %v", err)
	}
	requireCurrentCalls(t, third, "third send")
}

func TestCurrentBackendUnavailableSendSurfacesError(t *testing.T) {
	current := &currentBackend{}
	if err := current.send("must not disappear", nil, ""); !errors.Is(err, errBackendUnavailable) {
		t.Fatalf("unavailable backend send error = %v, want %v", err, errBackendUnavailable)
	}

	cmd := currentBackendSend(current, nil, "must not disappear", nil)
	if msg := cmd(); msg == nil {
		t.Fatal("unavailable ordinary Enter must produce sendErrMsg, got nil")
	} else if _, ok := msg.(sendErrMsg); !ok {
		t.Fatalf("unavailable ordinary Enter must produce sendErrMsg, got %T", msg)
	}
}

func TestCurrentBackendLeaseDrainsBeforeRetiring(t *testing.T) {
	old := newCurrentBackendStub("old")
	new := newCurrentBackendStub("new")
	current := newCurrentBackend(old)
	admitted := make(chan struct{})
	releaseAdmission := make(chan struct{})
	var leased *backendGeneration
	current.beforeSend = func(g *backendGeneration) {
		if g.backend != old {
			return
		}
		leased = g
		close(admitted)
		<-releaseAdmission
	}

	sent := make(chan error, 1)
	go func() { sent <- current.send("old preaccepted send", nil, "") }()
	select {
	case <-admitted:
	case <-time.After(time.Second):
		t.Fatal("old generation did not acquire its lease")
	}

	cleanup := current.replace(new)
	current.mu.Lock()
	stateAfterSwap := leased.state
	current.mu.Unlock()
	if stateAfterSwap != backendDraining {
		t.Fatalf("old generation state after swap = %v, want draining", stateAfterSwap)
	}
	if err := current.send("new send", nil, ""); err != nil {
		t.Fatalf("new generation send: %v", err)
	}
	requireCurrentCalls(t, new, "new send")
	requireCurrentCalls(t, old)

	stopped := make(chan tea.Msg, 1)
	go func() { stopped <- cleanup() }()
	select {
	case msg := <-stopped:
		t.Fatalf("old generation retired/stopped before its admitted send ran: %T", msg)
	default:
	}
	close(releaseAdmission)
	if err := <-sent; err != nil {
		t.Fatalf("old preaccepted send: %v", err)
	}
	requireCurrentCalls(t, old, "old preaccepted send")
	if _, ok := (<-stopped).(backendStopMsg); !ok {
		t.Fatal("drained generation cleanup must return backendStopMsg")
	}
	current.mu.Lock()
	stateAfterDrain := leased.state
	current.mu.Unlock()
	if stateAfterDrain != backendRetired {
		t.Fatalf("old generation state after its send = %v, want retired", stateAfterDrain)
	}
	if got := old.stopCount(); got != 1 {
		t.Fatalf("old generation Stop calls = %d, want 1", got)
	}
}

func TestStaleBackendBuildResultCannotReplaceLatestGeneration(t *testing.T) {
	old := newCurrentBackendStub("old")
	stale := newCurrentBackendStub("stale")
	m := New(old, config.Default())
	m.backendTransitioning = true
	m.backendTransitionID = 2

	cmd := m.finishBackendTransition(backendBuildMsg{
		name:       "claudecode",
		oldName:    "opencode",
		backend:    stale,
		transition: 1,
	})
	if cmd == nil {
		t.Fatal("a stale build candidate must schedule asynchronous teardown")
	}
	if _, ok := cmd().(backendStopMsg); !ok {
		t.Fatal("stale build teardown must report backendStopMsg")
	}
	if m.backend != old {
		t.Fatalf("stale build replaced backend with %T, want original", m.backend)
	}
	if err := m.currentBackend.send("still old", nil, ""); err != nil {
		t.Fatalf("current backend send: %v", err)
	}
	requireCurrentCalls(t, old, "still old")
	requireCurrentCalls(t, stale)
	if got := stale.stopCount(); got != 1 {
		t.Fatalf("stale build Stop calls = %d, want 1", got)
	}
}

func TestStaleBypassBuildClearsLatch(t *testing.T) {
	old := newCurrentBackendStub("old")
	stale := newCurrentBackendStub("stale")
	m := New(old, config.Default())
	m.bypassRestarting = true
	m.backendTransitioning = true
	m.backendTransitionID = 2

	// A bypass build result with a stale transition ID must still clear
	// the bypass latch so a future /bypass toggle is not permanently wedged.
	cmd := m.finishBackendTransition(backendBuildMsg{
		name:       "claudecode",
		oldName:    "claudecode",
		backend:    stale,
		bypass:     true,
		transition: 1,
	})
	if m.backend != old {
		t.Fatal("stale bypass build must not replace the backend")
	}
	if m.bypassRestarting {
		t.Fatal("stale bypass build must clear bypassRestarting")
	}
	// The latch cleanup may return a cmd (queued follow-up); drain it.
	if cmd != nil {
		_ = cmd()
	}
}
