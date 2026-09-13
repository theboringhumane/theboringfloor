// codex_primary_persist_test.go — state.EvPrimaryLearned's prompt-persist
// contract (sessions.go/model.go): a backend that just resolved (or
// changed) its primary/thread id must see it durable on disk immediately,
// without waiting for the periodic 5s cheap-write loop (EvTick) or a clean
// shutdown — and a repeat of the SAME id must never trigger a redundant
// save (a thread spans many turns and repeats its id on every one).
package app

import (
	"os"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// TestPrimaryLearnedPersistsImmediately proves the write happens off the
// EvPrimaryLearned event itself — no EvTick and no shutdown call ever
// fires in this test, yet session.json appears with the learned id.
func TestPrimaryLearnedPersistsImmediately(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	backend := &pinBackend{primary: "thread-A"}
	m := New(backend, nil)
	m.sessDir = dir

	m = runMsg(t, m, state.Event{Kind: state.EvPrimaryLearned, PrimaryID: "thread-A"})

	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: expected an immediate write on EvPrimaryLearned")
	}
	if got.PrimaryID != "thread-A" {
		t.Fatalf("persisted primary = %q, want %q", got.PrimaryID, "thread-A")
	}
}

// TestPrimaryLearnedSkipsRedundantSaveForSameID proves a repeat of the
// already-saved id is a no-op: the file is deleted after the first save,
// then the identical event is replayed — a redundant save would recreate
// it, so its continued absence is the proof (timestamp comparisons would
// be flaky at millisecond resolution; presence/absence is not).
func TestPrimaryLearnedSkipsRedundantSaveForSameID(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	backend := &pinBackend{primary: "thread-A"}
	m := New(backend, nil)
	m.sessDir = dir

	m = runMsg(t, m, state.Event{Kind: state.EvPrimaryLearned, PrimaryID: "thread-A"})
	if _, ok := LoadSession(dir); !ok {
		t.Fatal("expected the first EvPrimaryLearned to persist immediately")
	}
	if err := os.Remove(SessionPath(dir)); err != nil {
		t.Fatalf("removing session.json before the redundant replay: %v", err)
	}

	m = runMsg(t, m, state.Event{Kind: state.EvPrimaryLearned, PrimaryID: "thread-A"})
	if _, ok := LoadSession(dir); ok {
		t.Fatal("a repeat EvPrimaryLearned with the same id redundantly re-saved session.json")
	}
}

// TestPrimaryLearnedPersistsOnActualChange proves the dedup is keyed on
// VALUE, not on "has fired before": a genuinely new id (e.g. /new minting a
// fresh Codex thread) must still force its own prompt save.
func TestPrimaryLearnedPersistsOnActualChange(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	backend := &pinBackend{primary: "thread-A"}
	m := New(backend, nil)
	m.sessDir = dir

	m = runMsg(t, m, state.Event{Kind: state.EvPrimaryLearned, PrimaryID: "thread-A"})
	backend.primary = "thread-B"
	m = runMsg(t, m, state.Event{Kind: state.EvPrimaryLearned, PrimaryID: "thread-B"})

	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: expected a persisted session after the second, distinct id")
	}
	if got.PrimaryID != "thread-B" {
		t.Fatalf("persisted primary = %q, want %q (second, distinct id)", got.PrimaryID, "thread-B")
	}
}
