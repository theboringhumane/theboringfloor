// main_test.go — the headless probe's memory-lane report content: the
// one-line "did the office remember" readout pins its grammar
// ("memory: agentmemory <OK|file-only> | ledger N dispatches | newest
// <ledgerId|->").
package main

import (
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/backend"
)

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
