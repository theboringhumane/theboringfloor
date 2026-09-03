// abort_timeout_test.go — the /stop network bounds + the teardown budget
// (backend leg, httptest doubles only, never a real serve — stuck_test.go's
// idiom):
//
//	G5(d) — every abort POST is ctx-bounded per call (abortCallTimeout):
//	     a black-holed serve (socket answered, reply never sent) fails the
//	     hop as an observed error instead of parking it forever;
//	G5(e) — Stop is step-bounded: the ledger drain never outwaits
//	     stopDrainTimeout, and the spawned-child SIGKILL + reap never
//	     outwaits stopKillGrace — the whole Stop lands inside its budget
//	     against a LIVE child and a never-ending saver.
package backend

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

// G5(d) — the per-call abort ctx: a serve that parks /session/{id}/abort
// errors the hop at abortCallTimeout, not at leisure.
func TestAbortCallBoundedOnWedgedServe(t *testing.T) {
	old := abortCallTimeout
	abortCallTimeout = 150 * time.Millisecond
	t.Cleanup(func() { abortCallTimeout = old })

	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/abort") {
			<-unblock // the black hole: answered socket, never a reply
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(func() { close(unblock); srv.Close() })
	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")

	b := newLiveBackend(srv.URL, t.TempDir(), config.Default())
	b.mu.Lock()
	b.baseURL = srv.URL
	b.primaryID = "ses-x"
	b.mu.Unlock()

	t0 := time.Now()
	err := b.AbortSessions()
	dt := time.Since(t0)

	if err == nil {
		t.Fatal("a wedged abort serve must surface as an error, not silence (the G1 note rides it)")
	}
	if dt < abortCallTimeout {
		t.Fatalf("the abort die must ride the ctx deadline, returned in %v (< %v) — the post failed EARLY?", dt, abortCallTimeout)
	}
	if dt > abortCallTimeout+2*time.Second {
		t.Fatalf("the abort hop must die at its per-call deadline, took %v (budget %v)", dt, abortCallTimeout)
	}
}

// G5(e) — Stop's budget with both knives drawn: an in-flight ledger saver
// that NEVER finishes (the drain cap must not wait it out) and a live
// spawned child that has to be killed + reaped.
func TestStopBoundedKillsSpawnedChild(t *testing.T) {
	oldDrain, oldGrace := stopDrainTimeout, stopKillGrace
	stopDrainTimeout = 80 * time.Millisecond
	stopKillGrace = 200 * time.Millisecond
	t.Cleanup(func() { stopDrainTimeout, stopKillGrace = oldDrain, oldGrace })

	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")
	b := newLiveBackend("", t.TempDir(), config.Default())

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn the fake serve child: %v", err)
	}
	b.proc = cmd

	// one never-ending ledger saver: the drain must cap exactly at
	// stopDrainTimeout, not wait the saver out.
	b.ledgerWG.Add(1)
	t.Cleanup(func() { b.ledgerWG.Done() })

	t0 := time.Now()
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop errored: %v", err)
	}
	dt := time.Since(t0)

	if dt < stopDrainTimeout {
		t.Fatalf("Stop should ride the drain cap out for the in-flight saver, returned in %v (< %v)", dt, stopDrainTimeout)
	}
	if dt > stopDrainTimeout+stopKillGrace+2*time.Second {
		t.Fatalf("Stop overran its teardown budget: %v (drain %v + grace %v)", dt, stopDrainTimeout, stopKillGrace)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("the spawned serve child must be DEAD after Stop")
	}
}

// G5(e2) — the same Stop with NO child and NO savers is a pure scissor
// pass: it must return immediately (no phantom wait in the budget).
func TestStopImmediateWhenIdle(t *testing.T) {
	oldDrain, oldGrace := stopDrainTimeout, stopKillGrace
	stopDrainTimeout = 80 * time.Millisecond
	stopKillGrace = 200 * time.Millisecond
	t.Cleanup(func() { stopDrainTimeout, stopKillGrace = oldDrain, oldGrace })

	t.Setenv("AGENTMEMORY_URL", "http://127.0.0.1:1")
	b := newLiveBackend("", t.TempDir(), config.Default())

	t0 := time.Now()
	if err := b.Stop(); err != nil {
		t.Fatalf("Stop errored: %v", err)
	}
	if dt := time.Since(t0); dt > 500*time.Millisecond {
		t.Fatalf("an idle Stop must never touch the budget, took %v", dt)
	}
}
