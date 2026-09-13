// floor_handoff_test.go — launchFloor's busy-floor handoff gate (every
// backend hands off when busy; only opencode also carries the in-flight
// turn via --server, and only claude/codex post the tradeoff notice) and
// the handoffCurrentFloor helper it drives.
//
// Every test injects floorHandoffSpawn/floorHandoffReady (package vars —
// same seam convention as SpawnTerminal/BackendFactory) so nothing here
// ever execs a real theboringfloor binary or makes a real HTTP call.
package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

// stubHandoffSeams installs fake floorHandoffSpawn/floorHandoffReady for the
// duration of the test and returns a recorder of what launchFloor asked for.
type handoffSpawnCall struct {
	dir, backendName, session, serverURL string
}

func stubHandoffSeams(t *testing.T, ready bool, spawnErr error) (calls *[]handoffSpawnCall, killed *bool) {
	t.Helper()
	calls = &[]handoffSpawnCall{}
	killed = new(bool)
	origSpawn, origReady := floorHandoffSpawn, floorHandoffReady
	t.Cleanup(func() { floorHandoffSpawn, floorHandoffReady = origSpawn, origReady })
	floorHandoffSpawn = func(dir, backendName, session, serverURL string) (func(), error) {
		*calls = append(*calls, handoffSpawnCall{dir: dir, backendName: backendName, session: session, serverURL: serverURL})
		if spawnErr != nil {
			return nil, spawnErr
		}
		return func() { *killed = true }, nil
	}
	floorHandoffReady = func(string, time.Duration) bool { return ready }
	return calls, killed
}

// attachableBackend is a pinBackend that also implements
// state.ServerAttachable, for the --server-wiring tests below. url is
// what ServerURL() returns until released flips (a real liveBackend's
// one-way latch — see opencode.go's ServerURL/ReleaseServe).
type attachableBackend struct {
	pinBackend
	url      string
	released bool
}

func (b *attachableBackend) ServerURL() string {
	if b.released {
		return ""
	}
	return b.url
}
func (b *attachableBackend) ReleaseServe() { b.released = true }

// busyModel builds a live pinBackend-backed Model rooted at dir with a
// pending boss turn (the cheapest backendSwapBlockers trip: "boss turn in
// flight") and the given active backend name.
func busyModel(t *testing.T, dir, backendName, primary string) Model {
	t.Helper()
	m := New(&pinBackend{primary: primary}, nil)
	m.sessDir = dir
	m.st.BackendName = backendName
	m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true, At: 1}}
	return m
}

// idleModel is busyModel without the pending boss turn, so
// backendSwapBlockers() comes back empty and the floor counts as idle.
func idleModel(t *testing.T, dir, backendName, primary string) Model {
	t.Helper()
	m := New(&pinBackend{primary: primary}, nil)
	m.sessDir = dir
	m.st.BackendName = backendName
	return m
}

func lastNotice(m Model) state.ChatMsg {
	return m.st.Chat[len(m.st.Chat)-1]
}

// TestHandoffRespawnsUnderTheDepartingBackend — the detached child must be
// pinned to the backend the departing floor was actually running. The
// session id handed to it is that backend's OWN primary (claude's uuid,
// codex's thread id), so respawning under a different backend would try to
// resume a claude or codex conversation as an opencode one and silently
// lose it. This regressed once: the spawn hardcoded "--backend opencode"
// back when only opencode was ever handed off.
func TestHandoffRespawnsUnderTheDepartingBackend(t *testing.T) {
	for _, backendName := range []string{"opencode", "claudecode", "codex"} {
		t.Run(backendName, func(t *testing.T) {
			t.Setenv("THEFLOOR_HOME", t.TempDir())
			departing := t.TempDir()
			target := t.TempDir()
			calls, _ := stubHandoffSeams(t, true, nil)

			m := busyModel(t, departing, backendName, "sess-"+backendName)
			if cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"}); cmd == nil {
				t.Fatal("a busy floor must hand off and proceed, got nil cmd")
			}
			if len(*calls) != 1 {
				t.Fatalf("floorHandoffSpawn calls = %d, want 1", len(*calls))
			}
			if got := (*calls)[0].backendName; got != backendName {
				t.Errorf("detached child spawned with --backend %q, want the DEPARTING floor's backend %q", got, backendName)
			}
			if got := (*calls)[0].session; got != "sess-"+backendName {
				t.Errorf("detached child pinned to session %q, want %q", got, "sess-"+backendName)
			}
		})
	}
}

// TestLaunchFloorIdleOpencodeSkipsHandoff — an idle floor has no in-flight
// turn to rescue and its conversation is already resumable from the
// persisted session, so switching away from it must take the cheap
// exec-replace path rather than spawning a detached office nobody needs.
// Paying for a background process on every idle switch would leak an
// office for each floor a member merely browses past.
func TestLaunchFloorIdleOpencodeSkipsHandoff(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, killed := stubHandoffSeams(t, true, nil)

	m := idleModel(t, departing, "opencode", "sess-a")
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("idle opencode switch must proceed (quit), got nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("proceeding cmd() = %#v, want tea.QuitMsg", cmd())
	}
	// Dir is compared through workspace.Canonical: on darwin the temp dir
	// arrives as /var/... and is stored as its /private/var/... realpath.
	wantDir, err := workspace.Canonical(target)
	if err != nil {
		t.Fatalf("canonicalizing target: %v", err)
	}
	if m.execFloor == nil || m.execFloor.Dir != wantDir {
		t.Fatalf("execFloor after idle switch = %+v, want the target floor %q", m.execFloor, wantDir)
	}
	if len(*calls) != 0 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 0 — an idle floor must not be handed off", len(*calls))
	}
	if *killed {
		t.Error("an idle switch spawns nothing, so it must never kill anything")
	}
}

// TestLaunchFloorBusyOpencodeHandsOffAndProceeds — requirement 7 leg 1: a
// busy opencode floor switch spawns a handoff (pinned to the departing
// floor's own dir + session), proceeds with the switch, and posts NO
// tradeoff notice — for opencode nothing was lost, so there is nothing to
// disclose (requirement 5).
func TestLaunchFloorBusyOpencodeHandsOffAndProceeds(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, killed := stubHandoffSeams(t, true, nil)

	m := busyModel(t, departing, "opencode", "sess-a")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("busy opencode switch must proceed (quit), got nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("proceeding cmd() = %#v, want tea.QuitMsg", cmd())
	}
	if m.execFloor == nil || m.execFloor.Backend != "opencode" {
		t.Fatalf("execFloor after handoff = %+v, want the target floor request", m.execFloor)
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1", len(*calls))
	}
	if (*calls)[0].dir != departing {
		t.Errorf("handoff spawned for dir %q, want the DEPARTING floor %q", (*calls)[0].dir, departing)
	}
	if (*calls)[0].session != "sess-a" {
		t.Errorf("handoff spawned with session %q, want the departing floor's own primary %q", (*calls)[0].session, "sess-a")
	}
	if *killed {
		t.Error("a successful handoff must not kill the detached office it just verified healthy")
	}
	if len(m.st.Chat) != before {
		t.Errorf("opencode handoff posted a notice %q, want none — nothing was lost for opencode", lastNotice(m).Text)
	}
}

// TestLaunchFloorBusyClaudeHandsOffWithTradeoffNotice — requirement 5 leg
// 2: a busy claude floor is now handed off (not refused), spawns with its
// own primary session id, passes no --server (claude has no
// state.ServerAttachable seam), and posts the tradeoff notice — the
// conversation continues in the background, but the in-flight turn did
// not survive and needs re-running.
func TestLaunchFloorBusyClaudeHandsOffWithTradeoffNotice(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, killed := stubHandoffSeams(t, true, nil)

	m := busyModel(t, departing, "claudecode", "sess-a")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("a busy claudecode floor must now hand off and proceed (quit), got nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("proceeding cmd() = %#v, want tea.QuitMsg", cmd())
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1 — a busy claudecode floor must be handed off", len(*calls))
	}
	if (*calls)[0].dir != departing {
		t.Errorf("handoff spawned for dir %q, want the DEPARTING floor %q", (*calls)[0].dir, departing)
	}
	if (*calls)[0].session != "sess-a" {
		t.Errorf("handoff spawned with session %q, want claude's own primary %q", (*calls)[0].session, "sess-a")
	}
	if (*calls)[0].serverURL != "" {
		t.Errorf("spawn serverURL = %q, want empty — claude has no ServerAttachable seam", (*calls)[0].serverURL)
	}
	if *killed {
		t.Error("a successful handoff must not kill the detached office it just verified healthy")
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("chat length after handoff = %d, want %d (exactly one tradeoff notice appended)", len(m.st.Chat), before+1)
	}
	notice := lastNotice(m)
	want := "claudecode floor moved to the background — its conversation keeps running there, but the turn in progress (boss turn in flight) did not survive the switch and will need to be re-run."
	if notice.Text != want {
		t.Errorf("tradeoff notice = %q, want %q", notice.Text, want)
	}
	if notice.Meta == "error" {
		t.Errorf("tradeoff notice Meta = %q, want a non-error notice (this is a disclosure, not a failure)", notice.Meta)
	}
}

// TestLaunchFloorIdleClaudeSkipsHandoff — an idle claude floor has no
// in-flight turn to disclose or rescue, so it takes the cheap exec-replace
// path exactly like an idle opencode floor, and posts no notice.
func TestLaunchFloorIdleClaudeSkipsHandoff(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, _ := stubHandoffSeams(t, true, nil)

	m := idleModel(t, departing, "claudecode", "sess-a")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("idle claudecode switch must proceed (quit), got nil cmd")
	}
	if len(*calls) != 0 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 0 — an idle floor must not be handed off", len(*calls))
	}
	if len(m.st.Chat) != before {
		t.Errorf("idle switch posted a notice %q, want none", lastNotice(m).Text)
	}
}

// TestLaunchFloorBusySameFloorReselectIsNotAnError — re-selecting the floor
// you are already on while it is busy is a no-op, never a refusal: the
// busy check sits after the same-floor short-circuit precisely so a member
// clicking their current floor is not told to open a second terminal.
func TestLaunchFloorBusySameFloorReselectIsNotAnError(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	canonical, err := workspace.Canonical(departing)
	if err != nil {
		t.Fatalf("canonicalizing departing: %v", err)
	}
	calls, _ := stubHandoffSeams(t, true, nil)

	m := busyModel(t, canonical, "claudecode", "sess-a")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: canonical, Backend: "claudecode", Session: "sess-a"})

	if cmd != nil {
		t.Fatal("re-selecting the current floor must be a no-op, got a proceeding cmd")
	}
	if m.execFloor != nil {
		t.Fatalf("execFloor must stay nil on a same-floor no-op, got %+v", m.execFloor)
	}
	if len(*calls) != 0 {
		t.Fatalf("a same-floor no-op must spawn nothing, got %d spawn call(s)", len(*calls))
	}
	if len(m.st.Chat) != before {
		t.Errorf("same-floor no-op posted a notice %q, want none", lastNotice(m).Text)
	}
}

// TestLaunchFloorBusyCodexHandsOffWithTradeoffNotice — same leg for codex,
// confirming the gate reads the CURRENT backend name rather than a
// hardcoded assumption (requirement 5), and that codex's own primary
// session id (not claude's or opencode's) is the one pinned.
func TestLaunchFloorBusyCodexHandsOffWithTradeoffNotice(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, _ := stubHandoffSeams(t, true, nil)

	m := busyModel(t, departing, "codex", "sess-b")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("a busy codex floor must now hand off and proceed (quit), got nil cmd")
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1 — a busy codex floor must be handed off", len(*calls))
	}
	if (*calls)[0].session != "sess-b" {
		t.Errorf("handoff spawned with session %q, want codex's own primary %q", (*calls)[0].session, "sess-b")
	}
	if (*calls)[0].serverURL != "" {
		t.Errorf("spawn serverURL = %q, want empty — codex has no ServerAttachable seam", (*calls)[0].serverURL)
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("chat length after handoff = %d, want %d (exactly one tradeoff notice appended)", len(m.st.Chat), before+1)
	}
	notice := lastNotice(m)
	want := "codex floor moved to the background — its conversation keeps running there, but the turn in progress (boss turn in flight) did not survive the switch and will need to be re-run."
	if notice.Text != want {
		t.Errorf("tradeoff notice = %q, want %q", notice.Text, want)
	}
	if notice.Meta == "error" {
		t.Errorf("tradeoff notice Meta = %q, want a non-error notice (this is a disclosure, not a failure)", notice.Meta)
	}
}

// TestLaunchFloorIdleCodexSkipsHandoff — the codex counterpart of
// TestLaunchFloorIdleClaudeSkipsHandoff.
func TestLaunchFloorIdleCodexSkipsHandoff(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, _ := stubHandoffSeams(t, true, nil)

	m := idleModel(t, departing, "codex", "sess-b")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("idle codex switch must proceed (quit), got nil cmd")
	}
	if len(*calls) != 0 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 0 — an idle floor must not be handed off", len(*calls))
	}
	if len(m.st.Chat) != before {
		t.Errorf("idle switch posted a notice %q, want none", lastNotice(m).Text)
	}
}

// TestLaunchFloorHandoffAbortsForClaudeExactlyAsOpencode — requirement 5:
// the abort-on-failed-handoff path is backend-agnostic. A claude floor
// whose detached office never becomes healthy aborts the switch, reaps
// the half-started child, restores the departing floor's own discovery
// record, and posts NO tradeoff notice (the handoff never succeeded, so
// there is nothing to disclose — only the abort's own error notice).
func TestLaunchFloorHandoffAbortsForClaudeExactlyAsOpencode(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	origTimeout, origPoll := handoffReadyTimeout, handoffPollInterval
	handoffReadyTimeout, handoffPollInterval = 30*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { handoffReadyTimeout, handoffPollInterval = origTimeout, origPoll })
	calls, killed := stubHandoffSeams(t, false, nil)

	m := busyModel(t, departing, "claudecode", "sess-a")
	before := len(m.st.Chat)
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd != nil {
		t.Fatal("an aborted claude handoff must never quit — the switch must never proceed")
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1 (the one attempt)", len(*calls))
	}
	if !*killed {
		t.Error("an aborted handoff must reap the half-started detached office")
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("chat length after aborted handoff = %d, want %d (exactly the abort's own error notice, no tradeoff notice)", len(m.st.Chat), before+1)
	}
	notice := lastNotice(m)
	if notice.Meta != "error" {
		t.Errorf("abort must surface an error notice, got Meta=%q text=%q", notice.Meta, notice.Text)
	}
}

// TestLaunchFloorHandoffAbortsWhenReadinessNeverSucceeds — requirement 4 +
// 7 leg 3: a handoff whose readiness probe never succeeds aborts the
// switch, reaps the half-started child, restores the departing floor's
// own discovery record verbatim, and leaves state unchanged (never quits).
func TestLaunchFloorHandoffAbortsWhenReadinessNeverSucceeds(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	origTimeout, origPoll := handoffReadyTimeout, handoffPollInterval
	handoffReadyTimeout, handoffPollInterval = 30*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { handoffReadyTimeout, handoffPollInterval = origTimeout, origPoll })
	calls, killed := stubHandoffSeams(t, false, nil)

	original := control.Discovery{PID: 987654321, Port: 4242, Token: "orig-token", Dir: departing, StartedAt: 111, Version: "test"}
	if err := control.WriteDiscovery(departing, original); err != nil {
		t.Fatalf("seed discovery: %v", err)
	}

	m := busyModel(t, departing, "opencode", "sess-a")
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd != nil {
		t.Fatal("an aborted handoff must never quit — the switch must never proceed")
	}
	if m.execFloor != nil {
		t.Fatalf("execFloor must stay nil on an aborted handoff, got %+v", m.execFloor)
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1 (the one attempt)", len(*calls))
	}
	if !*killed {
		t.Error("an aborted handoff must reap the half-started detached office")
	}
	restored, ok := control.ReadDiscovery(departing)
	if !ok {
		t.Fatal("the departing floor's own discovery record must be restored on abort")
	}
	if restored.Port != original.Port || restored.Token != original.Token || restored.Dir != original.Dir {
		t.Fatalf("restored discovery = %+v, want the original %+v verbatim", restored, original)
	}
	notice := lastNotice(m)
	if notice.Meta != "error" {
		t.Errorf("abort must surface an error notice, got Meta=%q text=%q", notice.Meta, notice.Text)
	}
}

// TestLaunchFloorPersistsSessionEvenOnAbortedHandoff — requirement 2 + 7:
// PersistSession is durability, not teardown — it must run even on the
// aborted handoff path.
func TestLaunchFloorPersistsSessionEvenOnAbortedHandoff(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	origTimeout, origPoll := handoffReadyTimeout, handoffPollInterval
	handoffReadyTimeout, handoffPollInterval = 20*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { handoffReadyTimeout, handoffPollInterval = origTimeout, origPoll })
	stubHandoffSeams(t, false, nil)

	if _, ok := LoadSession(departing); ok {
		t.Fatal("precondition: no session.json should exist yet")
	}

	m := busyModel(t, departing, "opencode", "sess-a")
	if cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"}); cmd != nil {
		t.Fatal("this handoff is stubbed to fail readiness and must abort (nil cmd)")
	}

	sf, ok := LoadSession(departing)
	if !ok {
		t.Fatal("PersistSession must have run even though the handoff aborted — session.json is missing")
	}
	if sf.PrimaryID != "sess-a" {
		t.Errorf("persisted PrimaryID = %q, want %q", sf.PrimaryID, "sess-a")
	}
}

// TestLaunchFloorHandoffSpawnErrorAborts — a spawn failure (binary
// resolution, /dev/null open, exec.Start all bubble up as an error) is
// treated exactly like a readiness failure: abort, restore, never quit.
func TestLaunchFloorHandoffSpawnErrorAborts(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	spawnErr := errString("spawn refused")
	calls, killed := stubHandoffSeams(t, true, spawnErr)

	m := busyModel(t, departing, "opencode", "sess-a")
	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd != nil {
		t.Fatal("a spawn error must abort the switch")
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1", len(*calls))
	}
	if *killed {
		t.Error("a spawn error means nothing started — kill must not be invoked on a nil handle")
	}
	notice := lastNotice(m)
	if notice.Text == "" || notice.Meta != "error" {
		t.Fatalf("spawn-error abort must surface an error notice, got %+v", notice)
	}
}

// TestLaunchFloorHandoffAttachesServerAndReleasesOnSuccess — requirement
// 5 + 7: when the departing backend implements state.ServerAttachable
// and offers a non-empty ServerURL(), a successful handoff passes
// --server <url> to the detached child AND releases the serve (only
// after the readiness probe already succeeded).
func TestLaunchFloorHandoffAttachesServerAndReleasesOnSuccess(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, _ := stubHandoffSeams(t, true, nil)

	backend := &attachableBackend{url: "http://127.0.0.1:54321"}
	m := New(backend, nil)
	m.sessDir = departing
	m.st.BackendName = "opencode"
	m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true, At: 1}}

	cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"})

	if cmd == nil {
		t.Fatal("a healthy handoff must proceed (quit), got nil cmd")
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1", len(*calls))
	}
	if (*calls)[0].serverURL != "http://127.0.0.1:54321" {
		t.Errorf("spawn serverURL = %q, want the backend's ServerURL()", (*calls)[0].serverURL)
	}
	if !backend.released {
		t.Error("a healthy handoff must call ReleaseServe() so this process stops owning the serve")
	}
}

// TestLaunchFloorHandoffOmitsServerWhenNotAttachable — the counterpart:
// a backend without the ServerAttachable capability (pinBackend, the
// existing test double) must never pass --server — the child falls back
// to spawning its own serve exactly as before this feature existed.
func TestLaunchFloorHandoffOmitsServerWhenNotAttachable(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	calls, _ := stubHandoffSeams(t, true, nil)

	m := busyModel(t, departing, "opencode", "sess-a")
	if cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"}); cmd == nil {
		t.Fatal("busy opencode switch must proceed")
	}
	if len(*calls) != 1 {
		t.Fatalf("floorHandoffSpawn calls = %d, want 1", len(*calls))
	}
	if (*calls)[0].serverURL != "" {
		t.Errorf("spawn serverURL = %q, want empty (backend has no ServerAttachable capability)", (*calls)[0].serverURL)
	}
}

// TestLaunchFloorHandoffFailedReadinessNeverReleases — requirement 5 +
// 7: a readiness probe that never succeeds must NOT call ReleaseServe —
// the serve stays owned by this process so the normal shutdown path
// still kills it. An orphan with no owner must never happen.
func TestLaunchFloorHandoffFailedReadinessNeverReleases(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	departing := t.TempDir()
	target := t.TempDir()
	origTimeout, origPoll := handoffReadyTimeout, handoffPollInterval
	handoffReadyTimeout, handoffPollInterval = 30*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { handoffReadyTimeout, handoffPollInterval = origTimeout, origPoll })
	stubHandoffSeams(t, false, nil)

	backend := &attachableBackend{url: "http://127.0.0.1:54321"}
	m := New(backend, nil)
	m.sessDir = departing
	m.st.BackendName = "opencode"
	m.st.Chat = []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true, At: 1}}

	if cmd := m.launchFloor(FloorLaunch{Dir: target, Backend: "opencode"}); cmd != nil {
		t.Fatal("a failed readiness probe must abort the switch (nil cmd)")
	}
	if backend.released {
		t.Error("a failed readiness probe must NOT release the serve — it stays owned by this process")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
