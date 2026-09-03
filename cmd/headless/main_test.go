// main_test.go — the headless probe's memory-lane report content: the
// one-line "did the office remember" readout pins its grammar
// ("memory: agentmemory <OK|file-only> | ledger N dispatches | newest
// <ledgerId|->").
package main

import (
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/config"
)

// the boot summary's transport row: demo names itself, live names the
// brain.json resolution ("" backfills to opencode), explicit names ride.
func TestBackendNameLine(t *testing.T) {
	cfg := config.Default()
	if got, want := backendNameLine(false, cfg), "[backend] demo"; got != want {
		t.Errorf("demo run: got %q want %q", got, want)
	}
	if got, want := backendNameLine(true, cfg), "[backend] opencode"; got != want {
		t.Errorf("live default: got %q want %q", got, want)
	}
	cfg2 := config.Default()
	cfg2.Backend.Name = "" // pre-schema brain.json: the backfill must name it
	if got, want := backendNameLine(true, cfg2), "[backend] opencode"; got != want {
		t.Errorf("live empty name: got %q want %q", got, want)
	}
	cfg3 := config.Default()
	cfg3.Backend.Name = config.BackendNameClaude
	if got, want := backendNameLine(true, cfg3), "[backend] claudecode"; got != want {
		t.Errorf("live claudecode: got %q want %q", got, want)
	}
}

func TestMemoryReportLine(t *testing.T) {
	dir := t.TempDir()
	led := backend.NewLedger(dir)
	if got, want := memoryReportLine("file-only", led),
		"[memory] memory: agentmemory file-only | ledger 0 dispatches | newest -"; got != want {
		t.Fatalf("empty ledger report:\n got %q\nwant %q", got, want)
	}
	e := backend.LedgerEntry{
		LedgerID:      "led-1756987200000-0a1b2c3d",
		DispatchTitle: "Probe the memory lane",
		WorkerName:    "tekton-1",
		WorkerRole:    "developer",
		Verdict:       "done",
		Summary:       "did the probe",
		CompletedAt:   time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).UnixMilli(),
	}
	if err := led.Append(e); err != nil {
		t.Fatalf("seed the ledger: %v", err)
	}
	if got, want := memoryReportLine("OK", led),
		"[memory] memory: agentmemory OK | ledger 1 dispatches | newest led-1756987200000-0a1b2c3d"; got != want {
		t.Fatalf("populated report:\n got %q\nwant %q", got, want)
	}
}
