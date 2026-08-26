// ledger_test.go — the office ledger's full unit matrix:
//
//	(a) goldens: a fresh append materializes the chartered seed + the
//	    frozen per-entry block, byte-exact;
//	(b) ordering: entries land NEWEST-first under the entries marker;
//	(c) dedupe + byte-stability: the same ledgerId twice never rewrites
//	    a byte (SAFE to call twice), a re-render with nothing changed
//	    never rewrites either;
//	(d) the 50-entry cap drops the oldest off the tail;
//	(e) malformed ledgers (seed lost, marker lost) recover with the
//	    canonical seed and their salvageable entries re-homed;
//	(f) read-back: Latest/Len/SummaryText, tolerant of foreign blocks
//	    (verbatim on disk, invisible to readers);
//	(g) the return-contract parser: DONE/FILES/VERIFY/PROOF/ISSUES
//	    sections in their real-world spellings, verdict derivation;
//	(h) LedgerID determinism.
package backend

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ledgerEntryFixture is a fully-stamped entry (2026-08-25 12:00:00 UTC —
// the UTC-pinned date render keeps goldens machine-independent).
func ledgerEntryFixture() LedgerEntry {
	return LedgerEntry{
		LedgerID:      "led-1756987200000-0a1b2c3d",
		DispatchTitle: "Ship the memory lane",
		WorkerName:    "tekton-3",
		WorkerRole:    "developer",
		WorkerSession: "ses-9",
		Verdict:       "done",
		Summary:       "DONE\n- chose fnv-1a for the id digest\n- appends are byte-stable\n",
		Files:         []string{"internal/backend/ledger.go", "internal/backend/ledger_test.go"},
		VerifyDigest:  "go test ./internal/backend/ — ok (exit 0)",
		ProofOneLiner: "the ledger renders newest-first",
		CompletedAt:   time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).UnixMilli(),
		PrimaryID:     "ses-1",
		Project:       "proj",
	}
}

// ledgerFixtureBlock is ledgerEntryFixture's exact on-disk block.
const ledgerFixtureBlock = "### 2026-08-25 · Ship the memory lane — tekton-3 (developer) · `done`\n" +
	"- summary: chose fnv-1a for the id digest\n" +
	"- files: internal/backend/ledger.go, internal/backend/ledger_test.go\n" +
	"- verify: go test ./internal/backend/ — ok (exit 0)\n" +
	"- proof: the ledger renders newest-first\n" +
	"- ledgerId: led-1756987200000-0a1b2c3d\n\n"

func ledgerFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".opencode", "office-ledger.md"))
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	return string(raw)
}

// (a) the golden: append into an absent file => chartered seed + the
// frozen block, byte-exact (the seed comes from the charter pass's own
// renderLedgerSeed — the writer never forks it).
func TestLedgerAppendGolden(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	if err := led.Append(ledgerEntryFixture()); err != nil {
		t.Fatalf("append: %v", err)
	}
	want := string(renderLedgerSeed()) + ledgerFixtureBlock
	if got := ledgerFile(t, dir); got != want {
		t.Fatalf("ledger bytes diverged from the frozen shape:\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

// (b) newest-first: a second append lands DIRECTLY under the marker,
// ahead of the older block.
func TestLedgerNewestFirst(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	first := ledgerEntryFixture()
	second := ledgerEntryFixture()
	second.LedgerID = "led-1756987260000-f0e1d2c4"
	second.DispatchTitle = "Second dispatch"
	second.WorkerName = "skopos-1"
	second.WorkerRole = "scout"
	second.Summary = "done the recon"
	if err := led.Append(first); err != nil {
		t.Fatal(err)
	}
	if err := led.Append(second); err != nil {
		t.Fatal(err)
	}
	got := ledgerFile(t, dir)
	iNew := strings.Index(got, "### 2026-08-25 · Second dispatch — skopos-1 (scout) · `done`")
	iOld := strings.Index(got, "### 2026-08-25 · Ship the memory lane — tekton-3 (developer) · `done`")
	iMarker := strings.Index(got, ledgerEntriesMarker)
	if iMarker < 0 || iNew < 0 || iOld < 0 {
		t.Fatalf("ledger missing marker/blocks:\n%s", got)
	}
	if !(iMarker < iNew && iNew < iOld) {
		t.Fatalf("want marker < newest < oldest, got %d < %d < %d", iMarker, iNew, iOld)
	}
}

// (c) dedupe: the same ledgerId appended twice writes ONE entry, and the
// second call leaves every byte untouched (safe twice + the byte-stable
// self-diff gate).
func TestLedgerDedupeTwiceByteStable(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	e := ledgerEntryFixture()
	if err := led.Append(e); err != nil {
		t.Fatal(err)
	}
	before := ledgerFile(t, dir)
	if err := led.Append(e); err != nil {
		t.Fatal(err)
	}
	if got := ledgerFile(t, dir); got != before {
		t.Fatalf("a duplicate ledgerId must never rewrite a byte")
	}
	if n := strings.Count(before, "- ledgerId: "+e.LedgerID); n != 1 {
		t.Fatalf("want exactly one ledgerId line, got %d", n)
	}
	if n := strings.Count(before, "### 2026-08-25 ·"); n != 1 {
		t.Fatalf("want exactly one entry block, got %d", n)
	}
	// A re-append of a KNOWN id through a FRESH handle (process restart
	// shape): still a no-op — the file's own dedupe gate, not in-memory.
	if err := NewLedger(dir).Append(e); err != nil {
		t.Fatal(err)
	}
	if got := ledgerFile(t, dir); got != before {
		t.Fatalf("restart-shape duplicate must still be a no-op")
	}
}

// (d) the cap: 55 appends keep the newest 50 and the oldest five age out.
func TestLedgerCap50(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).UnixMilli()
	for i := 0; i < 55; i++ {
		at = at + 60_000 // a minute apart: strictly older -> strictly newer
		brief := "brief-" + itoa(i)
		if err := led.Append(LedgerEntry{
			DispatchTitle: brief,
			WorkerName:    "tekton-1",
			WorkerRole:    "developer",
			Verdict:       "done",
			Summary:       "did " + brief,
			CompletedAt:   at,
			Project:       filepath.Base(dir),
		}); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	got := ledgerFile(t, dir)
	if n := strings.Count(got, "### 2026-08-25 ·"); n != ledgerCap {
		t.Fatalf("cap: want %d blocks, got %d", ledgerCap, n)
	}
	entries := led.Entries()
	if len(entries) != ledgerCap {
		t.Fatalf("Len at the cap: want %d, got %d", ledgerCap, len(entries))
	}
	if entries[0].DispatchTitle != "brief-54" {
		t.Fatalf("newest-first at the cap: want brief-54 first, got %q", entries[0].DispatchTitle)
	}
	if entries[len(entries)-1].DispatchTitle != "brief-5" {
		t.Fatalf("the cap keeps the NEWEST 50: brief-5 is the oldest survivor, got %q", entries[len(entries)-1].DispatchTitle)
	}
}

// (e1) malformed: the seed/marker is gone — recovery writes the canonical
// seed and re-homes the salvageable entry blocks (garbage prose drops).
func TestLedgerMalformedRecovers(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mangled := "some FOREIGN preamble that was never a seed\n\n" +
		"### 2026-08-20 · old hand entry — dev#7 (developer) · `done`\n" +
		"- summary: shipped by hand\n" +
		"- files: (none)\n" +
		"- verify: (none)\n" +
		"- proof: (none)\n" +
		"- ledgerId: led-old-1\n\n" +
		"random tail prose\n"
	path := filepath.Join(ocDir, "office-ledger.md")
	if err := os.WriteFile(path, []byte(mangled), 0o644); err != nil {
		t.Fatal(err)
	}
	led := NewLedger(dir)
	e := ledgerEntryFixture()
	if err := led.Append(e); err != nil {
		t.Fatal(err)
	}
	got := ledgerFile(t, dir)
	if !strings.HasPrefix(got, string(renderLedgerSeed())) {
		t.Fatalf("recovery must restore the canonical seed, got:\n%.200s", got)
	}
	if strings.Contains(got, "FOREIGN preamble") || strings.Contains(got, "random tail") {
		t.Fatalf("recovery must drop garbage prose, got:\n%s", got)
	}
	iNew := strings.Index(got, "### 2026-08-25 · Ship the memory lane")
	iOld := strings.Index(got, "### 2026-08-20 · old hand entry — dev#7 (developer) · `done`")
	if iNew < 0 || iOld < 0 || iNew > iOld {
		t.Fatalf("recovery must keep salvageable entries, newest first (%d, %d):\n%s", iNew, iOld, got)
	}
	// And the recovered file parses back.
	latest, ok := led.Latest()
	if !ok || latest.LedgerID != e.LedgerID {
		t.Fatalf("the recovered ledger must read back its newest entry, got %+v ok=%v", latest, ok)
	}
}

// (e2) malformed variant: an EMPTY file (touch-only crash remnant) also
// recovers to seed + entry.
func TestLedgerEmptyFileRecovers(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "office-ledger.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	led := NewLedger(dir)
	if err := led.Append(ledgerEntryFixture()); err != nil {
		t.Fatal(err)
	}
	if got, want := ledgerFile(t, dir), string(renderLedgerSeed())+ledgerFixtureBlock; got != want {
		t.Fatalf("empty file must recover to seed+block:\n%q", got)
	}
}

// (f1) read-back: Latest / Len / the lossy parse contract (one-line
// summary, date-precision CompletedAt, unstored fields zero).
func TestLedgerReadBack(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	if _, ok := led.Latest(); ok {
		t.Fatal("missing file: Latest must be ok=false")
	}
	e := ledgerEntryFixture()
	if err := led.Append(e); err != nil {
		t.Fatal(err)
	}
	latest, ok := led.Latest()
	if !ok {
		t.Fatal("after append: Latest must find the entry")
	}
	if latest.LedgerID != e.LedgerID || latest.DispatchTitle != e.DispatchTitle ||
		latest.WorkerName != "tekton-3" || latest.WorkerRole != "developer" || latest.Verdict != "done" {
		t.Fatalf("Latest fields off: %+v", latest)
	}
	if latest.Summary != "chose fnv-1a for the id digest" {
		t.Fatalf("the file carries the one-line summary digest, got %q", latest.Summary)
	}
	if len(latest.Files) != 2 || latest.Files[0] != "internal/backend/ledger.go" {
		t.Fatalf("files read-back off: %v", latest.Files)
	}
	if latest.WorkerSession != "" || latest.PrimaryID != "" {
		t.Fatalf("unstored fields read back zero, got %+v", latest)
	}
	if led.Len() != 1 {
		t.Fatalf("Len: want 1, got %d", led.Len())
	}
}

// (f2) SummaryText: the boss-quotable digest grammar + the n-cap.
func TestLedgerSummaryText(t *testing.T) {
	dir := t.TempDir()
	led := NewLedger(dir)
	if got := led.SummaryText("proj", 5); got != "no dispatches recorded in proj yet" {
		t.Fatalf("empty summary: %q", got)
	}
	first := ledgerEntryFixture()
	second := ledgerEntryFixture()
	second.LedgerID = "led-1756987260000-f0e1d2c4"
	second.DispatchTitle = "Second dispatch"
	second.Verdict = "issues"
	if err := led.Append(first); err != nil {
		t.Fatal(err)
	}
	if err := led.Append(second); err != nil {
		t.Fatal(err)
	}
	got := led.SummaryText("proj", 1)
	if !strings.HasPrefix(got, "2 dispatches recorded in proj (newest 1 shown)") {
		t.Fatalf("summary header: %q", got)
	}
	if !strings.Contains(got, "Second dispatch — tekton-3 (developer) · issues") {
		t.Fatalf("summary must carry the newest verdict, got %q", got)
	}
	if strings.Contains(got, "Ship the memory lane") {
		t.Fatalf("n=1 must hide the older entry, got %q", got)
	}
	if full := led.SummaryText("proj", 0); !strings.Contains(full, "Ship the memory lane") {
		t.Fatalf("n<=0 shows all entries, got %q", full)
	}
}

// (f3) foreign blocks: a hand-shaped "### " section survives Append
// verbatim (dedupe/cap ride raw text) but never parses into Entries.
func TestLedgerForeignBlocksSurviveUnparsed(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".opencode")
	if err := os.MkdirAll(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := string(renderLedgerSeed()) +
		"### 2026-08-26 · shipped the thing — dev#7 · done\n"
	if err := os.WriteFile(filepath.Join(ocDir, "office-ledger.md"), []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}
	led := NewLedger(dir)
	if err := led.Append(ledgerEntryFixture()); err != nil {
		t.Fatal(err)
	}
	got := ledgerFile(t, dir)
	if !strings.Contains(got, "### 2026-08-26 · shipped the thing — dev#7 · done\n") {
		t.Fatalf("a foreign block must survive verbatim:\n%s", got)
	}
	if led.Len() != 1 {
		t.Fatalf("foreign blocks never parse: want Len 1, got %d", led.Len())
	}
	if latest, _ := led.Latest(); latest.LedgerID != "led-1756987200000-0a1b2c3d" {
		t.Fatalf("Latest skips the unparseable block, got %+v", latest)
	}
}

// (g) the return-contract parser: real-world header spellings, files
// glosses, verify/proof digests, and the issues->verdict rule.
func TestLedgerReturnContractParser(t *testing.T) {
	text := strings.Join([]string{
		"Preamble chatter the parser must drop.",
		"1. DONE",
		"- fixed the tokenizer edge case",
		"- kept the reducer pure",
		"2. FILES",
		"- internal/backend/ledger.go — the writer",
		"- internal/backend/opencode.go (the emission sites)",
		"3. VERIFY",
		"- go build ./... — ok",
		"- go test ./internal/backend/ — PASS (exit 0)",
		"4. PROOF",
		"- the ledger renders newest-first at 50",
		"- a second screenshot line",
		"5. ISSUES",
		"- none",
	}, "\n")
	sections := parseLedgerSections(text)
	if got := sections["DONE"]; len(got) == 0 {
		t.Fatalf("DONE section not mined: %v", sections)
	}
	files := ledgerFilePaths(sections["FILES"])
	if len(files) != 2 || files[0] != "internal/backend/ledger.go" || files[1] != "internal/backend/opencode.go" {
		t.Fatalf("files must drop their glosses, got %v", files)
	}
	if got := ledgerLastLine(sections["VERIFY"], 140); got != "go test ./internal/backend/ — PASS (exit 0)" {
		t.Fatalf("verify digest keeps the last-passed command + exit, got %q", got)
	}
	if got := ledgerFirstLine(sections["PROOF"], 140); got != "the ledger renders newest-first at 50" {
		t.Fatalf("proof one-liner, got %q", got)
	}
	if issues := ledgerIssues(sections["ISSUES"]); issues != nil {
		t.Fatalf(`"none" issues must collapse to nil (verdict done), got %v`, issues)
	}
	// A REAL issues body flips the verdict's input.
	issues := ledgerIssues([]string{"", "- the parser still rejects CRLF files", "- see wave note"})
	if len(issues) != 2 {
		t.Fatalf("real issues must survive as bullets, got %v", issues)
	}
	// Header spellings the contract actually produces.
	for _, line := range []string{"DONE", "1. DONE", "2) FILES", "### VERIFY", "**PROOF:**", "5. ISSUES — or \"none\""} {
		if matchLedgerSection(line) == "" {
			t.Fatalf("%q must open a section", line)
		}
	}
	for _, line := range []string{"the work is done here", "files touched: 3", "proofread the essay", ""} {
		if matchLedgerSection(line) != "" {
			t.Fatalf("%q must NOT open a section", line)
		}
	}
}

// (h) LedgerID: deterministic per (stamp, title, worker), chronological-
// readable, 8-hex digest; two different shapes differ.
func TestLedgerIDDeterministic(t *testing.T) {
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).UnixMilli()
	a := LedgerID(at, "Ship the memory lane", "tekton-3")
	if LedgerID(at, "Ship the memory lane", "tekton-3") != a {
		t.Fatal("same inputs must mint the same ledger id (tests pin it)")
	}
	if !strings.HasPrefix(a, "led-") {
		t.Fatalf("id shape led-<ms>-<hash>, got %q", a)
	}
	if !regexp8Hex.MatchString(a[strings.LastIndex(a, "-")+1:]) {
		t.Fatalf("digest must be 8 lowercase hex, got %q", a)
	}
	if b := LedgerID(at, "Ship the memory lane", "tekton-4"); b == a {
		t.Fatal("a different worker must mint a different id")
	}
}

var regexp8Hex = regexp.MustCompile(`^[0-9a-f]{8}$`)
