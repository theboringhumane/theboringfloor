// backend_switch_test.go — the /backend mid-flight swap contract:
//
//   - the IDLE gate: ANY in-flight surface (boss/office typing, live
//     workers, unanswered permission/question, backlog queue, batch flush)
//     refuses with a reason-bearing copy on BOTH the transcript (office
//     row) and the statusline (EvStatus twin) — and tears down NOTHING.
//   - esc-esc//stop unwinds the turn → the retry swaps: old transport
//     stopped, new one factory-built + Started on the boot sink, ONE
//     "[theboringfloor] backend: <old> → <new> (turn #N archived)"
//     EvStatus lands (activity + statusline + the topbar latch), and the
//     choice persists to brain.json.
//   - session.json pins sessions PER BACKEND: sequential persists under
//     two transports keep both PrimaryIDs entries, the legacy PrimaryID
//     slot tracks the active one, and a pre-schema file (no map at all)
//     decodes with the opencode fallback intact (fallback-literacy).
package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// swapStubBackend — a LIVE-mode recording backend for the /backend tests:
// Start signals on a buffered channel (the app starts the swapped
// transport on a goroutine), Stop counts, and the primary seam scripts
// the per-transport session pins (sessions.go primarySeamBackend).
type swapStubBackend struct {
	recBackend
	mu        sync.Mutex
	started   chan struct{}
	emitFn    func(state.Event)
	stops     int
	primary   string
	overrides []string
}

func newSwapStub(primary string) *swapStubBackend {
	return &swapStubBackend{started: make(chan struct{}, 4), primary: primary}
}

func (b *swapStubBackend) Mode() state.Mode { return state.ModeLive }

func (b *swapStubBackend) Start(emit func(state.Event)) error {
	b.mu.Lock()
	b.emitFn = emit
	b.mu.Unlock()
	b.started <- struct{}{}
	return nil
}

func (b *swapStubBackend) Stop() error {
	b.mu.Lock()
	b.stops++
	b.mu.Unlock()
	return nil
}

func (b *swapStubBackend) stopCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stops
}

func (b *swapStubBackend) PrimaryOverride(id string) {
	b.mu.Lock()
	b.overrides = append(b.overrides, id)
	b.mu.Unlock()
}

func (b *swapStubBackend) pinnedOverrides() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.overrides...)
}

func (b *swapStubBackend) PrimaryID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.primary
}

func (b *swapStubBackend) setPrimary(id string) {
	b.mu.Lock()
	b.primary = id
	b.mu.Unlock()
}

func (b *swapStubBackend) waitStarted(t *testing.T, what string) {
	t.Helper()
	select {
	case <-b.started:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s: the swapped transport was never Started on the boot sink", what)
	}
}

// installFactory points BackendFactory at builds (by backend NAME) and
// records the call order. Returns the recording slice.
func installFactory(t *testing.T, builds map[string]*swapStubBackend) *[]string {
	t.Helper()
	calls := &[]string{}
	mu := sync.Mutex{}
	old := BackendFactory
	BackendFactory = func(name, baseURL, dir string, cfg *config.Config) state.Backend {
		mu.Lock()
		*calls = append(*calls, name)
		mu.Unlock()
		if b, ok := builds[name]; ok {
			return b
		}
		t.Errorf("factory asked for unscripted backend %q", name)
		return newSwapStub("")
	}
	t.Cleanup(func() { BackendFactory = old })
	return calls
}

// newSwapModel — a live-mode model on stub, with the session dir pinned at
// dir (sessions go under the test's THEBORINGOFFICE_HOME scratch) and the
// boot splash lifted (backend input flows; the splash is a no-show).
func newSwapModel(b *swapStubBackend, cfg *config.Config, dir string) Model {
	m := New(b, cfg)
	m.bootDone = true
	m.sessDir = dir
	return m
}

// seedBossTurn stages the "typing" placeholder that makes the office busy.
func seedBossTurn(m *Model) {
	m.st.Chat = append(m.st.Chat, state.ChatMsg{
		ID: "boss-typing-1", From: "boss", Kind: "boss", Pending: true,
		At: time.Now().UnixMilli(),
	})
	m.tabs.SetState(m.st)
}

// lastOfficeText reads the newest office chat row's plain text.
func lastOfficeText(t *testing.T, m Model) string {
	t.Helper()
	last := lastChat(t, m)
	if last.From != "office" {
		t.Fatalf("expected an office notice row, got from=%q text=%q", last.From, last.Text)
	}
	return ansi.Strip(last.Text)
}

// (a) BUSY REFUSES: a pending boss turn keeps the transport pinned; the
// refusal carries the blocker on both surfaces; NOTHING tears down.
func TestBackendSwapBusyRefuses(t *testing.T) {
	scratchHome(t)
	stubA := newSwapStub("ses-op-1")
	m := newSwapModel(stubA, config.Default(), t.TempDir())
	seedBossTurn(&m)

	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})

	if got := lastOfficeText(t, m); !strings.Contains(got, "refused") ||
		!strings.Contains(got, "boss turn in flight") ||
		!strings.Contains(got, "esc-esc / /stop first") ||
		!strings.Contains(got, "/backend claudecode again") {
		t.Fatalf("busy refusal notice must name the blocker + the wait-first recovery, got:\n%s", got)
	}
	if !strings.Contains(m.st.StatusLine, "refused") || !strings.Contains(m.st.StatusLine, "boss turn in flight") {
		t.Fatalf("the refusal must ride the statusline too, got %q", m.st.StatusLine)
	}
	if m.backend != stubA {
		t.Fatalf("a refused swap must keep the current transport (got %T)", m.backend)
	}
	if stubA.stopCount() != 0 {
		t.Fatalf("a refused swap must NOT stop the current transport (%d stops)", stubA.stopCount())
	}
	if m.st.BackendName != "" {
		t.Fatalf("a refusal must never touch the topbar latch, got %q", m.st.BackendName)
	}
}

// (b) the IDLE gate's blocker inventory: every busy surface refuses with
// ITS reason in the copy.
func TestBackendSwapBlockerInventory(t *testing.T) {
	cases := []struct {
		name  string
		stage func(*Model)
		want  string
	}{
		{"office typing", func(m *Model) {
			m.st.Chat = append(m.st.Chat, state.ChatMsg{
				ID: "office-1", From: "office", Kind: "office", Pending: true, Text: "…",
				At: time.Now().UnixMilli()})
		}, "office reply in flight"},
		{"live worker", func(m *Model) {
			m.st.Employees = append(m.st.Employees, state.Employee{
				ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteWorking})
		}, "1 worker(s) live"},
		{"permission pending", func(m *Model) {
			m.permQ.pending = []*permPrompt{{ID: "p1", ToolName: "bash"}}
		}, "unanswered permission prompt"},
		{"question parked", func(m *Model) { m.questionParked = true }, "unanswered boss question"},
		{"queue backlog", func(m *Model) {
			m.queue = append(m.queue, queueEntry{text: "later"}, queueEntry{text: "later 2"})
		}, "queue non-empty (2 items)"},
		{"batch flush", func(m *Model) { m.batchInFlight = true }, "batch flush in flight"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scratchHome(t)
			stubA := newSwapStub("ses-op-1")
			m := newSwapModel(stubA, config.Default(), t.TempDir())
			tc.stage(&m)
			m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
			got := lastOfficeText(t, m)
			if !strings.Contains(got, "refused") || !strings.Contains(got, tc.want) {
				t.Fatalf("refusal must carry %q, got:\n%s", tc.want, got)
			}
			if m.backend != stubA || stubA.stopCount() != 0 {
				t.Fatalf("refused swap must not touch the transport (backend=%T stops=%d)", m.backend, stubA.stopCount())
			}
		})
	}
}

// (c) FORCE-ESC CANCELS THE BLOCK, THEN THE SWAP RUNS: esc-esc unwinds the
// turn (the /stop path), the retry drains-first: old stopped, new built +
// Started on the sink, ONE swap EvStatus, latch + brain.json persist.
func TestBackendSwapForceEscThenSwaps(t *testing.T) {
	home := scratchHome(t)
	dir := t.TempDir()
	stubA := newSwapStub("ses-op-1")
	stubB := newSwapStub("")
	calls := installFactory(t, map[string]*swapStubBackend{"claudecode": stubB})
	cfg := config.Default()
	m := newSwapModel(stubA, cfg, dir)
	m.SetEventSink(func(state.Event) {})

	seedBossTurn(&m)
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	if !strings.Contains(m.st.StatusLine, "refused") {
		t.Fatalf("while busy the swap must refuse, got statusline %q", m.st.StatusLine)
	}

	// esc-esc — the panel's double-esc seam ferries stopWorkMsg into the
	// model (the /stop path): turn unwound, status line reports the stop.
	m = runMsg(t, m, stopWorkMsg{})
	if hasPendingBoss(m.st) {
		t.Fatalf("the esc-esc //stop path must unwind the boss turn, chat: %+v", m.st.Chat)
	}

	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	stubB.waitStarted(t, "claudecode")
	if stubA.stopCount() != 1 {
		t.Fatalf("the swap must stop the old transport exactly once, got %d", stubA.stopCount())
	}
	if *calls != nil && len(*calls) != 1 || len(*calls) == 1 && (*calls)[0] != "claudecode" {
		t.Fatalf("the swap must factory-build exactly claudecode, calls=%v", *calls)
	}
	wantLine := "[theboringfloor] backend: opencode → claudecode (turn #0 archived)"
	if m.st.StatusLine != wantLine {
		t.Fatalf("swap statusline:\n got %q\nwant %q", m.st.StatusLine, wantLine)
	}
	if m.st.BackendName != "claudecode" {
		t.Fatalf("the swap line must re-latch the topbar name, got %q", m.st.BackendName)
	}
	if m.cfg.Backend.Name != "claudecode" {
		t.Fatalf("the swap must persist the choice onto brain.json, cfg=%q", m.cfg.Backend.Name)
	}
	if stubB.pinnedOverrides() != nil {
		t.Fatalf("a transport with no archived pin must boot FRESH, overrides=%v", stubB.pinnedOverrides())
	}
	raw, err := os.ReadFile(filepath.Join(home, ".theboringfloor", "configs", "brain.json"))
	if err != nil {
		t.Fatalf("brain.json was not persisted: %v", err)
	}
	if !strings.Contains(string(raw), `"claudecode"`) {
		t.Fatalf("brain.json must carry the swapped name, got:\n%s", raw)
	}
	// The swap lands on the SAME sink the boot backend used (goroutine →
	// tea.Program.Send parity): the new transport's Start must receive it.
	stubB.mu.Lock()
	emitWasSink := stubB.emitFn != nil
	stubB.mu.Unlock()
	if !emitWasSink {
		t.Fatalf("the swapped transport must be Started with the boot sink")
	}
}

// (d) DRAIN-FLUSH: the turn completing (boss reply lands) re-opens the
// gate; the swap runs immediately after — and the archived chat SURVIVES.
func TestBackendSwapDrainFlushThenSwaps(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	stubA := newSwapStub("ses-op-1")
	stubB := newSwapStub("")
	installFactory(t, map[string]*swapStubBackend{"claudecode": stubB})
	m := newSwapModel(stubA, config.Default(), dir)
	m.SetEventSink(func(state.Event) {})

	seedBossTurn(&m)
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	if !strings.Contains(m.st.StatusLine, "refused") {
		t.Fatalf("mid-turn the swap must refuse, got %q", m.st.StatusLine)
	}
	// The reply pins: the turn completes, the queue drains, the gate opens.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "boss-typing-1", From: "boss", Kind: "boss", Text: "Pineapple.",
		At: time.Now().UnixMilli(),
	}})
	if hasPendingBoss(m.st) {
		t.Fatalf("a completed boss bubble must end the turn")
	}
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	stubB.waitStarted(t, "claudecode")
	wantLine := "[theboringfloor] backend: opencode → claudecode (turn #1 archived)"
	if m.st.StatusLine != wantLine {
		t.Fatalf("swap after drain must count the archived turn:\n got %q\nwant %q", m.st.StatusLine, wantLine)
	}
	// chat archive preserved: the user row-free transcript still shows the
	// completed boss reply after the swap.
	found := false
	for _, c := range m.st.Chat {
		if c.From == "boss" && c.Text == "Pineapple." {
			found = true
		}
	}
	if !found {
		t.Fatalf("the swap must preserve the chat archive, chat=%+v", m.st.Chat)
	}
}

// (e) THE SWAP LINE PRINTS EXACTLY ONCE: one activity row, zero transcript
// rows (an EvStatus, not a chat echo), statusline carries it verbatim.
func TestBackendSwapStatusPrintedOnce(t *testing.T) {
	scratchHome(t)
	stubA := newSwapStub("ses-op-1")
	stubB := newSwapStub("")
	installFactory(t, map[string]*swapStubBackend{"claudecode": stubB})
	m := newSwapModel(stubA, config.Default(), t.TempDir())
	m.SetEventSink(func(state.Event) {})

	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	stubB.waitStarted(t, "claudecode")

	const needle = "backend: opencode → claudecode"
	activity := 0
	for _, ln := range m.activity.Lines() {
		if strings.Contains(ln, needle) {
			activity++
		}
	}
	if activity != 1 {
		t.Fatalf("the swap line must land exactly once in activity, got %d of:\n%s", activity, strings.Join(m.activity.Lines(), "\n"))
	}
	chat := 0
	for _, c := range m.st.Chat {
		if strings.Contains(c.Text, needle) {
			chat++
		}
	}
	if chat != 0 {
		t.Fatalf("the swap line must NOT echo into the transcript (%d rows)", chat)
	}
}

// (f) SESSION PINS ARE PER BACKEND: persists under both transports keep
// both PrimaryIDs entries, the legacy slot tracks the ACTIVE transport,
// and swapping back re-pins the ORIGINAL opencode session.
func TestBackendSwapSessionPinsPerBackend(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	stubA := newSwapStub("ses-op-1")
	stubB := newSwapStub("ses-claude-9")
	stubC := newSwapStub("")
	calls := installFactory(t, map[string]*swapStubBackend{"claudecode": stubB, "opencode": stubC})
	m := newSwapModel(stubA, config.Default(), dir)
	m.SetEventSink(func(state.Event) {})

	m.PersistSession()
	sf, ok := LoadSession(dir)
	if !ok {
		t.Fatal("persist must write session.json")
	}
	if sf.Backend != "opencode" || sf.PrimaryID != "ses-op-1" || sf.PrimaryIDs["opencode"] != "ses-op-1" {
		t.Fatalf("opencode persist:\nbackend=%q primary=%q pins=%v", sf.Backend, sf.PrimaryID, sf.PrimaryIDs)
	}

	// over to claudecode: no archived pin → fresh boot there.
	m = runMsg(t, m, slashMsg{text: "/backend claudecode"})
	stubB.waitStarted(t, "claudecode")
	if got := stubB.pinnedOverrides(); got != nil {
		t.Fatalf("first claudecode boot must NOT pin anything, got %v", got)
	}
	// the CLI minted its session id after Start (runTurn's system.init twin)
	stubB.setPrimary("ses-claude-9")
	m.PersistSession()
	sf, ok = LoadSession(dir)
	if !ok {
		t.Fatal("re-persist must keep session.json")
	}
	if sf.PrimaryIDs["opencode"] != "ses-op-1" || sf.PrimaryIDs["claudecode"] != "ses-claude-9" {
		t.Fatalf("both transports' pins must coexist, got %v", sf.PrimaryIDs)
	}
	if sf.Backend != "claudecode" || sf.PrimaryID != "ses-claude-9" {
		t.Fatalf("the ACTIVE transport owns Backend+PrimaryID, got backend=%q primary=%q", sf.Backend, sf.PrimaryID)
	}
	// raw codec: the JSON really carries the map (not just the struct).
	raw, _ := os.ReadFile(SessionPath(dir))
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("session.json must stay valid JSON: %v", err)
	}
	pins, _ := doc["primaryIDs"].(map[string]any)
	if pins["opencode"] != "ses-op-1" || pins["claudecode"] != "ses-claude-9" {
		t.Fatalf("raw primaryIDs key: %v", doc["primaryIDs"])
	}

	// and BACK to opencode: the ORIGINAL session re-pins (no cross-pinning).
	m = runMsg(t, m, slashMsg{text: "/backend opencode"})
	stubC.waitStarted(t, "opencode")
	got := stubC.pinnedOverrides()
	if len(got) != 1 || got[0] != "ses-op-1" {
		t.Fatalf("swapping back must re-pin the archived opencode session, got %v", got)
	}
	if len(*calls) != 2 || (*calls)[0] != "claudecode" || (*calls)[1] != "opencode" {
		t.Fatalf("factory call order: %v", *calls)
	}
}

// (g) PRE-SCHEMA FILES DECODE CLEAN: a session.json written before this
// bump (no backend/primaryIDs keys at all) keeps PrimaryID as the opencode
// fallback and yields "" for claudecode (never a cross-pinned id).
func TestBackendSwapLegacySessionCodec(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	legacy := `{
  "dir": "` + dir + `",
  "primaryID": "ses-old-7",
  "agents": [],
  "tasks": [],
  "mails": [],
  "chat": [{"id": "c1", "from": "user", "text": "hi", "at": 1}],
  "savedAt": ` + jsonNumber(time.Now().UnixMilli()) + `
}`
	p := SessionPath(dir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	sf, ok := LoadSession(dir)
	if !ok {
		t.Fatal("a pre-schema session.json must decode")
	}
	if sf.PrimaryIDs != nil {
		t.Fatalf("pre-schema files must keep a NIL pin map, got %v", sf.PrimaryIDs)
	}
	if got := sf.primaryIDFor("opencode"); got != "ses-old-7" {
		t.Fatalf("legacy PrimaryID is the opencode entry, got %q", got)
	}
	if got := sf.primaryIDFor("claudecode"); got != "" {
		t.Fatalf("a pre-schema file must never cross-pin claudecode, got %q", got)
	}
	if got := sf.primaryIDFor(""); got != "ses-old-7" {
		t.Fatalf(`"" normalizes to opencode, got %q`, got)
	}
}

// (h) bare show / unknown name / same-name / demo refusal.
func TestBackendSlashValidation(t *testing.T) {
	scratchHome(t)

	// bare: prints the active transport without touching anything.
	stubA := newSwapStub("ses-op-1")
	m := newSwapModel(stubA, config.Default(), t.TempDir())
	m = runMsg(t, m, slashMsg{text: "/backend"})
	if got := lastOfficeText(t, m); !strings.Contains(got, "backend: opencode") ||
		!strings.Contains(got, "/backend opencode|claudecode") {
		t.Fatalf("bare /backend must show the active transport + usage, got %q", got)
	}

	// unknown name: refused before ANY gate logic.
	m = runMsg(t, m, slashMsg{text: "/backend zephyr"})
	if got := lastOfficeText(t, m); !strings.Contains(got, `unknown backend "zephyr"`) {
		t.Fatalf("unknown names must refuse, got %q", got)
	}
	if m.backend != stubA {
		t.Fatalf("validation refusal must keep the transport")
	}

	// same name: cheap note, no teardown.
	m = runMsg(t, m, slashMsg{text: "/backend opencode"})
	if got := lastOfficeText(t, m); !strings.Contains(got, "backend already on opencode") {
		t.Fatalf("same-name must be a no-op note, got %q", got)
	}
	if stubA.stopCount() != 0 {
		t.Fatalf("same-name must not stop the transport")
	}

	// demo mode: the scripted tour's backend is fixed.
	demo := New(&recBackend{}, config.Default())
	demo.bootDone = true
	demo = runMsg(t, demo, slashMsg{text: "/backend claudecode"})
	if got := lastOfficeText(t, demo); !strings.Contains(got, "live-only") {
		t.Fatalf("demo must refuse the swap as live-only, got %q", got)
	}
}

// jsonNumber keeps the legacy fixture literal readable.
func jsonNumber(ms int64) string {
	b, _ := json.Marshal(ms)
	return string(b)
}
