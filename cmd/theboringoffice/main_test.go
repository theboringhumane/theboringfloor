// main_test.go — the process-exit deadline: stopBounded must never let a
// hung backend Stop hold the process between the restored screen and the
// exit/exec (the quit-path half of the UI-freeze fix).
package main

import (
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// hungStopBackend — a state.Backend whose Stop NEVER returns (the wedged
// serve/vfs/lock the exit path must survive). release lets the test leash
// the leaked goroutine.
type hungStopBackend struct{ release chan struct{} }

func (h *hungStopBackend) Mode() state.Mode                        { return state.ModeLive }
func (h *hungStopBackend) Start(func(state.Event)) error           { return nil }
func (h *hungStopBackend) Send(string) error                       { return nil }
func (h *hungStopBackend) AnswerPermission(string, string) error   { return nil }
func (h *hungStopBackend) AnswerQuestion(string, [][]string) error { return nil }
func (h *hungStopBackend) RejectQuestion(string) error             { return nil }
func (h *hungStopBackend) MCPServers() ([]state.MCPServer, error)  { return nil, nil }
func (h *hungStopBackend) ReconnectMCP(string) error               { return nil }
func (h *hungStopBackend) Stop() error                             { <-h.release; return nil }

// A Stop that never returns must not hold stopBounded beyond the
// deadline — the exit path marches to os.Exit / syscall.Exec regardless.
func TestStopBoundedAgainstNeverReturningStop(t *testing.T) {
	old := stopDeadline
	stopDeadline = 150 * time.Millisecond
	t.Cleanup(func() { stopDeadline = old })

	b := &hungStopBackend{release: make(chan struct{})}
	t.Cleanup(func() { close(b.release) }) // leash the runaway goroutine

	t0 := time.Now()
	stopBounded(b)
	dt := time.Since(t0)

	if dt < stopDeadline {
		t.Fatalf("stopBounded must wait out its deadline for the wedged Stop, returned in %v", dt)
	}
	if dt > stopDeadline+2*time.Second {
		t.Fatalf("a never-returning Stop held the exit path %v beyond its %v deadline", dt, stopDeadline)
	}
}

// A prompt Stop passes straight through (no deadline burn on the happy
// path).
func TestStopBoundedFastOnHealthyBackend(t *testing.T) {
	old := stopDeadline
	stopDeadline = 150 * time.Millisecond
	t.Cleanup(func() { stopDeadline = old })

	b := &hungStopBackend{release: make(chan struct{})}
	close(b.release) // Stop returns immediately

	t0 := time.Now()
	stopBounded(b)
	if dt := time.Since(t0); dt > 500*time.Millisecond {
		t.Fatalf("a healthy Stop must not touch the deadline, took %v", dt)
	}
}
