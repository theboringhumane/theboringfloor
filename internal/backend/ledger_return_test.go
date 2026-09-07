package backend

import (
	"strings"
	"testing"
	"time"
)

func returnRecordFixture(dir, text string) ReturnRecordInput {
	return ReturnRecordInput{
		ProjectDir:    dir,
		PrimaryID:     "ses-primary",
		WorkerSession: "ses-worker",
		DispatchTitle: "Extract the ledger seam",
		WorkerName:    "tekton-17",
		WorkerRole:    "developer",
		ReturnText:    text,
		CompletedAt:   time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC).UnixMilli(),
	}
}

func TestRecordReturnGolden(t *testing.T) {
	dir := t.TempDir()
	recordReturn(returnRecordFixture(dir, "DONE\n- extracted the backend-agnostic seam\n\nFILES\n- internal/backend/ledger.go — recording helper\n\nVERIFY\n- go test ./internal/backend/... — PASS (exit 0)\n\nPROOF\n- returns record in both memory lanes\n\nISSUES\n- none"))

	const want = "### 2026-09-07 · Extract the ledger seam — tekton-17 (developer) · `done`\n" +
		"- summary: extracted the backend-agnostic seam\n" +
		"- files: internal/backend/ledger.go\n" +
		"- verify: go test ./internal/backend/... — PASS (exit 0)\n" +
		"- proof: returns record in both memory lanes\n" +
		"- ledgerId: led-1788782400000-93d6e927\n\n"
	if got := strings.TrimPrefix(ledgerFile(t, dir), string(renderLedgerSeed())); got != want {
		t.Fatalf("recorded block differs:\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestReturnLedgerEntryVerdicts(t *testing.T) {
	dir := t.TempDir()
	if got := returnLedgerEntry(returnRecordFixture(dir, "DONE\n- shipped\n\nISSUES\n- none")).Verdict; got != "done" {
		t.Fatalf("none issues verdict = %q, want done", got)
	}
	if got := returnLedgerEntry(returnRecordFixture(dir, "DONE\n- shipped\n\nISSUES\n- release credentials are unavailable")).Verdict; got != "issues" {
		t.Fatalf("real issues verdict = %q, want issues", got)
	}
}

func TestRecordReturnCapsAtFiftyNewestFirst(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 51; i++ {
		in := returnRecordFixture(dir, "DONE\n- completed\n\nISSUES\n- none")
		in.DispatchTitle = "dispatch " + string(rune('A'+i))
		in.CompletedAt += int64(i)
		recordReturn(in)
	}
	entries := NewLedger(dir).Entries()
	if len(entries) != ledgerCap {
		t.Fatalf("entries = %d, want %d", len(entries), ledgerCap)
	}
	if entries[0].DispatchTitle != "dispatch s" || entries[len(entries)-1].DispatchTitle != "dispatch B" {
		t.Fatalf("newest-first cap = first %q last %q, want dispatch s through dispatch B", entries[0].DispatchTitle, entries[len(entries)-1].DispatchTitle)
	}
}
