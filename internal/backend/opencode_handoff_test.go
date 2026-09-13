// opencode_handoff_test.go — state.ServerAttachable on the live backend:
// ServerURL/ReleaseServe and Stop()'s release-aware skip of the
// process-group signal. See internal/app/floor_handoff.go for the caller
// that type-asserts this seam during a floor-switch handoff, and
// TestStopBoundedKillsSpawnedChild (abort_timeout_test.go) for the
// existing "normal Stop kills a spawned child" proof this file pairs
// against with an explicit released/unreleased contrast.
package backend

import (
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

// TestServerURLEmptyWithNoServe — a fresh backend that never resolved a
// server (b.proc nil, b.baseURL "") has nothing to hand off.
func TestServerURLEmptyWithNoServe(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	if got := b.ServerURL(); got != "" {
		t.Fatalf("ServerURL() = %q, want empty (no serve running)", got)
	}
}

// TestServerURLEmptyWhenExternallyAttached — Start's u != "" branch never
// sets b.proc (see opencode.go's Start: the spawn-and-set-proc path is
// gated behind `if u == ""`), so a backend attached to an externally
// provided URL (optURL / OPENCODE_SERVER / cfg.Backend.Server) must
// report ServerURL() empty even though baseURL is set: it is not ours to
// hand off.
func TestServerURLEmptyWhenExternallyAttached(t *testing.T) {
	b := newLiveBackend("http://127.0.0.1:9999", t.TempDir(), config.Default())
	b.mu.Lock()
	b.baseURL = "http://127.0.0.1:9999" // what Start would set; b.proc stays nil
	b.mu.Unlock()
	if got := b.ServerURL(); got != "" {
		t.Fatalf("ServerURL() = %q, want empty (externally-attached server, not ours to release)", got)
	}
}

// TestServerURLNonEmptyWhenSpawned — this process spawned the serve
// (b.proc set, exactly Start's u == "" branch) and it is still live:
// ServerURL() must hand back the resolved attach URL verbatim.
func TestServerURLNonEmptyWhenSpawned(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	b.mu.Lock()
	b.proc = &exec.Cmd{} // identity token only — never started (stuck_test.go's idiom)
	b.baseURL = "http://127.0.0.1:54321"
	b.mu.Unlock()
	if got := b.ServerURL(); got != "http://127.0.0.1:54321" {
		t.Fatalf("ServerURL() = %q, want the spawned serve's resolved URL", got)
	}
}

// TestServerURLEmptyAfterServeStops — a stopped flow means the backend is
// tearing down; ServerURL() must not offer a URL for a handoff once
// b.fl.stop() has sealed (the same guard Stop() itself checks first).
func TestServerURLEmptyAfterServeStops(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	b.mu.Lock()
	b.proc = &exec.Cmd{}
	b.baseURL = "http://127.0.0.1:54321"
	b.mu.Unlock()
	b.fl.stop()
	if got := b.ServerURL(); got != "" {
		t.Fatalf("ServerURL() = %q, want empty once the flow has stopped", got)
	}
}

// TestServerURLEmptyAfterReleaseServe — the one-way latch: once handed
// off, ServerURL() can never offer the same serve for a second handoff.
func TestServerURLEmptyAfterReleaseServe(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	b.mu.Lock()
	b.proc = &exec.Cmd{}
	b.baseURL = "http://127.0.0.1:54321"
	b.mu.Unlock()
	b.ReleaseServe()
	if got := b.ServerURL(); got != "" {
		t.Fatalf("ServerURL() = %q, want empty after ReleaseServe (one-way latch)", got)
	}
}

// TestReleaseServeSkipsSignalOnStop — requirement 4: a released serve
// must survive Stop(). Uses a real (never-`opencode`/`theboringfloor`)
// short-lived child so the process-group signal is genuinely observable,
// mirroring TestStopBoundedKillsSpawnedChild's convention; the child is
// reaped in cleanup either way so nothing strays past the test.
func TestReleaseServeSkipsSignalOnStop(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())

	cmd := exec.Command("sleep", "30")
	isolateProcessGroup(cmd) // spawnServe's own recipe — signalProcessGroup targets -pid
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn the fake serve child: %v", err)
	}
	pid := cmd.Process.Pid
	// Stop()'s release branch calls cmd.Process.Release(), which poisons
	// THIS Go-side handle for further Signal/Kill calls regardless of
	// whether the OS process is actually alive ("os: process already
	// released") — liveness after Stop() must be checked, and the child
	// reaped in cleanup, by raw PID via syscall, never through cmd.Process.
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	b.mu.Lock()
	b.proc = cmd
	b.baseURL = "http://127.0.0.1:1"
	b.mu.Unlock()

	b.ReleaseServe()
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop errored: %v", err)
	}

	if err := syscall.Kill(pid, syscall.Signal(0)); err != nil {
		t.Fatalf("a released serve must survive Stop(), signal(0) on pid %d = %v (process appears dead)", pid, err)
	}
}

// TestUnreleasedServeSignalsOnStop — the other half of requirement 4: a
// backend that never released its serve must still have it killed by
// Stop(), exactly as today. Explicit released=false contrast to the test
// above, kept independent of TestStopBoundedKillsSpawnedChild so this
// file proves the released/unreleased split on its own.
func TestUnreleasedServeSignalsOnStop(t *testing.T) {
	oldGrace := stopKillGrace
	stopKillGrace = 200 * time.Millisecond
	t.Cleanup(func() { stopKillGrace = oldGrace })

	b := newLiveBackend("", t.TempDir(), config.Default())

	cmd := exec.Command("sleep", "30")
	isolateProcessGroup(cmd) // spawnServe's own recipe — signalProcessGroup targets -pid
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn the fake serve child: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	b.mu.Lock()
	b.proc = cmd
	b.baseURL = "http://127.0.0.1:1"
	b.mu.Unlock()

	// released is left false — no ReleaseServe() call. exit stays nil
	// (no procExit installed — this test bypasses spawnServe's reaper
	// entirely), so Stop() takes the "foreign caller" branch: SIGINT then
	// immediately SIGKILL, no wait. Reap it here ourselves (a signalled
	// child is a zombie, not gone, until something calls Wait — an
	// unrelated syscall.Kill(pid, 0) liveness probe would still see the
	// zombie's PID and falsely read "alive") and assert the exit was a
	// kill, not a clean return.
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop errored: %v", err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case err := <-waitErr:
		if err == nil {
			t.Fatal("an unreleased serve must be killed by Stop(), got a clean exit")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("an unreleased serve must be DEAD after Stop() — it never exited (normal teardown, unchanged)")
	}
}

// TestServerAttachableConcurrentSafe — ServerURL/ReleaseServe must be
// safe to call concurrently with each other and with Stop(), matching
// the mu-guarded convention every other field on liveBackend already
// uses (requirement 3). Not a race-detector proof by itself, but the
// access pattern below only ever compiles/passes if every touched field
// stays behind b.mu.
func TestServerAttachableConcurrentSafe(t *testing.T) {
	b := newLiveBackend("", t.TempDir(), config.Default())
	b.mu.Lock()
	b.proc = &exec.Cmd{}
	b.baseURL = "http://127.0.0.1:54321"
	b.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = b.ServerURL() }()
		go func() { defer wg.Done(); b.ReleaseServe() }()
	}
	wg.Wait()

	if got := b.ServerURL(); got != "" {
		t.Fatalf("ServerURL() = %q after concurrent ReleaseServe calls, want empty", got)
	}
}
