// memory_test.go — the office-memory surfaces contract:
//
//	/memory renders the project ledger (.opencode/office-ledger.md —
//	newest-first "### date · title — worker (role) · verdict" records,
//	written by the ledger-core on each completed dispatch) with the exact
//	row format, the probe-state parenthetical, the missing/empty-file
//	degrade-open notice, the malformed-row partial parse note, and the
//	case-fold title/files filter — PLUS the two fabric proofs that the
//	memory is live: the boot-line probe latch and the per-completed-
//	dispatch "[memory] recorded: <title> → ledger" activity line.
package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// memoryFixtureLedger — three newest-first records in the ledger writer's
// canonical block layout (internal/backend/ledger.go renderLedgerBlock):
// backticked verdict, bulleted fields, "(none)" placeholders — exercising
// done (with files + verify), done with "(none)" placeholders, and the
// ✗ issues glyph.
const memoryFixtureLedger = `# theboringfloor ledger — completed dispatches (newest first)

<!-- ledger:entries -->

### 2026-08-25 · Plan-pane gating — tekton-4 (developer) · ` + "`done`" + `
- summary: gated plan replies by shape
- files: internal/app/model.go, internal/app/plan_shape_test.go
- verify: go test ./internal/app/ -count=1 -run Memory ✓
- proof: only plan-shaped replies present
- ledgerId: 2026-08-25T010203Z-plan-pane-gating-tekton-4

### 2026-08-25 · Slash popover audit — skopos-1 (scout) · ` + "`done`" + `
- summary: (none)
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: 2026-08-25T000102Z-slash-popover-audit-skopos-1

### 2026-08-24 · Power governor ticks — hemerodromos-2 (runner) · ` + "`issues`" + `
- summary: saver drift late under load
- files: internal/app/power.go
- verify: go test ./internal/app/ -count=1 -run Power ✗
- proof: tick table drifted one slot
- ledgerId: 2026-08-24T010203Z-power-governor-ticks-hemerodromos-2
`

// writeMemoryLedger plants the ledger fixture under dir/.opencode.
func writeMemoryLedger(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath(dir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newMemoryModel — a demo-mode model pointed at dir (sessDir pins the
// ledger's home, like plan_mode_test.go's pattern).
func newMemoryModel(b state.Backend, dir string) Model {
	m := New(b, nil)
	m.sessDir = dir
	return m
}

// (a) /memory with a fixture ledger renders the header + the EXACT rows,
// newest first, verify digests riding their own rows.
func TestMemorySlashRendersLedgerRows(t *testing.T) {
	dir := t.TempDir()
	writeMemoryLedger(t, dir, memoryFixtureLedger)
	m := newMemoryModel(&recBackend{}, dir)
	m = runMsg(t, m, slashMsg{text: "/memory"})

	last := lastChat(t, m)
	if last.From != "office" || last.Meta != "" {
		t.Fatalf("/memory must land ONE clean dim office notice, from=%q meta=%q", last.From, last.Meta)
	}
	plain := ansi.Strip(last.Text)
	project := filepath.Base(dir)
	for _, want := range []string{
		"office memory — " + project + " · 3 dispatches recorded (file-only)",
		"  2026-08-25 · Plan-pane gating — tekton-4 (developer) · ✓ done  ▸ files: internal/app/model.go, internal/app/plan_shape_test.go",
		"      verify: go test ./internal/app/ -count=1 -run Memory ✓",
		"  2026-08-25 · Slash popover audit — skopos-1 (scout) · ✓ done",
		"  2026-08-24 · Power governor ticks — hemerodromos-2 (runner) · ✗ issues  ▸ files: internal/app/power.go",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("/memory notice missing %q:\n%s", want, plain)
		}
	}
	// newest first: the 08-25 rows precede the 08-24 row.
	i25 := strings.Index(plain, "Plan-pane gating")
	i24 := strings.Index(plain, "Power governor ticks")
	if i25 < 0 || i24 < 0 || i25 > i24 {
		t.Fatalf("rows must render newest-first (i25=%d i24=%d):\n%s", i25, i24, plain)
	}
	// The scout row carries NO files segment, NO verify continuation.
	scoutTail := plain[strings.Index(plain, "Slash popover audit"):]
	if strings.Contains(strings.Split(scoutTail, "\n")[0], "▸ files") {
		t.Fatalf("a files-less record must not render the ▸ segment:\n%s", scoutTail)
	}
	t.Logf("/memory transcript:\n%s", plain)
}

// (b) the filter argument narrows rows case-fold over title/files —
// header keeps the TOTAL count.
func TestMemorySlashFilterNarrows(t *testing.T) {
	dir := t.TempDir()
	writeMemoryLedger(t, dir, memoryFixtureLedger)
	m := newMemoryModel(&recBackend{}, dir)

	// case-fold: "PLAN" matches "Plan-pane gating" (title) and
	// plan_shape_test.go (files) — the two WRONG rows must be gone but the
	// verify continuation of the matching row stays.
	m = runMsg(t, m, slashMsg{text: "/memory PLAN"})
	last := lastChat(t, m)
	plain := ansi.Strip(last.Text)
	if !strings.Contains(plain, "office memory — "+filepath.Base(dir)+" · 3 dispatches recorded") {
		t.Fatalf("a filtered view keeps the TOTAL in the header:\n%s", plain)
	}
	if !strings.Contains(plain, "Plan-pane gating") {
		t.Fatalf("filter PLAN must keep the plan row:\n%s", plain)
	}
	if !strings.Contains(plain, "verify: go test ./internal/app/ -count=1 -run Memory") {
		t.Fatalf("a kept row drags its verify digest along:\n%s", plain)
	}
	for _, gone := range []string{"Slash popover audit", "Power governor ticks"} {
		if strings.Contains(plain, gone) {
			t.Fatalf("filter PLAN must hide %q:\n%s", gone, plain)
		}
	}

	// a filter over FILES (not titles) narrows to the power row.
	m = runMsg(t, m, slashMsg{text: "/memory power.go"})
	plain = ansi.Strip(lastChat(t, m).Text)
	if !strings.Contains(plain, "Power governor ticks") || strings.Contains(plain, "Plan-pane") {
		t.Fatalf("filter over files must land the power row only:\n%s", plain)
	}

	// a filter matching nothing gets the honest empty-filter line.
	m = runMsg(t, m, slashMsg{text: "/memory zzz-not-a-dispatch"})
	plain = ansi.Strip(lastChat(t, m).Text)
	if !strings.Contains(plain, `no dispatches match "zzz-not-a-dispatch"`) {
		t.Fatalf("a dead filter must say so plainly:\n%s", plain)
	}
}

// (c) no ledger file (fresh office / concurrent first boot / ledger core
// not landed yet): the fixed degrade-open notice, never an error row.
func TestMemorySlashMissingFile(t *testing.T) {
	dir := t.TempDir() // nothing written
	m := newMemoryModel(&recBackend{}, dir)
	m = runMsg(t, m, slashMsg{text: "/memory"})

	last := lastChat(t, m)
	if last.Meta == "error" {
		t.Fatalf("a missing ledger is never an error row, got: %q", last.Text)
	}
	plain := ansi.Strip(last.Text)
	if !strings.Contains(plain, "office memory — "+filepath.Base(dir)+" · 0 dispatches recorded (file-only)") {
		t.Fatalf("the missing-file header must count 0 honestly:\n%s", plain)
	}
	if !strings.Contains(plain, "no dispatches recorded yet — the office records every completed dispatch here once it finishes one.") {
		t.Fatalf("the pinned missing-ledger line must render verbatim:\n%s", plain)
	}
}

// (d) partial parse: good rows render, entry-SHAPED-but-malformed headers
// collapse into ONE dim trailing note, and non-shaped "### " prose is
// ignored (never counted). An all-malformed ledger still reads as the
// missing state — never invented rows.
func TestMemorySlashPartialLedger(t *testing.T) {
	dir := t.TempDir()
	writeMemoryLedger(t, dir, `# ledger
### 2026-08-25 · Real record — tekton-1 (developer) · `+"`done`"+`
- files: a.go

### this heading is prose — the ledger's own noise, never a dropped row
### 2026-08-26 · missing the em-dash record · `+"`done`"+`
`)
	m := newMemoryModel(&recBackend{}, dir)
	m = runMsg(t, m, slashMsg{text: "/memory"})

	plain := ansi.Strip(lastChat(t, m).Text)
	if !strings.Contains(plain, "office memory — "+filepath.Base(dir)+" · 1 dispatch recorded (file-only)") {
		t.Fatalf("one parseable row counts 1 (singular unit):\n%s", plain)
	}
	if !strings.Contains(plain, "  2026-08-25 · Real record — tekton-1 (developer) · ✓ done  ▸ files: a.go") {
		t.Fatalf("the parseable row must render exactly:\n%s", plain)
	}
	if !strings.Contains(plain, "  (1 ledger row skipped — malformed)") {
		t.Fatalf("malformed rows collapse into ONE dim note:\n%s", plain)
	}

	// the file exists but holds ZERO parseable records — degrade to the
	// missing-records line (with the skip note), never invent rows.
	dir2 := t.TempDir()
	writeMemoryLedger(t, dir2, `### 2026-08-25 · garbage record without the em-dash worker separator · `+"`done`"+`
`)
	m2 := newMemoryModel(&recBackend{}, dir2)
	m2 = runMsg(t, m2, slashMsg{text: "/memory"})
	plain2 := ansi.Strip(lastChat(t, m2).Text)
	if !strings.Contains(plain2, "no dispatches recorded yet — the office records every completed dispatch here once it finishes one.") {
		t.Fatalf("an all-malformed ledger still reads the honest empty state:\n%s", plain2)
	}
	if !strings.Contains(plain2, "  (1 ledger row skipped — malformed)") {
		t.Fatalf("singular skip note:\n%s", plain2)
	}
}

// (e) probe-state surfaces: the boot-line marker latches "agentmemory OK",
// the offline marker latches "file-only", and the ADDITIVE seam (when a
// backend implements it) overrules the latch.
func TestMemoryProbeState(t *testing.T) {
	dir := t.TempDir()
	writeMemoryLedger(t, dir, memoryFixtureLedger)
	m := newMemoryModel(&recBackend{}, dir)

	// boot line with a hot probe — the string contract of opencode.go's
	// Start status (same marker pattern as agentFieldStatusMarker).
	m = runMsg(t, m, state.Event{Kind: state.EvStatus,
		Text: "[theboringfloor] live - http://127.0.0.1:9999 | board: agentmemory (GET /agentmemory/actions)"})
	if !strings.Contains(m.memoryBody(""), "(agentmemory OK)") {
		t.Fatalf("the hot-probe boot line must flip the header state:\n%s", m.memoryBody(""))
	}

	// the offline marker flips it back.
	m = runMsg(t, m, state.Event{Kind: state.EvStatus,
		Text: "[theboringfloor] live - http://127.0.0.1:9999 | board: in-memory | agentmemory: offline (in-memory board)"})
	if !strings.Contains(m.memoryBody(""), "(file-only)") {
		t.Fatalf("the offline boot line must report file-only:\n%s", m.memoryBody(""))
	}

	// the ADDITIVE MemoryLane seam overrules the latch, both ways (the
	// ledger-core's liveBackend contract: "OK" | "file-only").
	m = runMsg(t, m, state.Event{Kind: state.EvStatus,
		Text: "[theboringfloor] live - http://127.0.0.1:9999 | board: agentmemory (GET /agentmemory/actions)"})
	m.backend = &memorySeamBackend{lane: "file-only"}
	if !strings.Contains(m.memoryBody(""), "(file-only)") {
		t.Fatalf("an implemented seam answering file-only must overrule a hot latch:\n%s", m.memoryBody(""))
	}
	m.backend = &memorySeamBackend{lane: "OK"}
	if !strings.Contains(m.memoryBody(""), "(agentmemory OK)") {
		t.Fatalf("an implemented seam answering OK must say agentmemory OK:\n%s", m.memoryBody(""))
	}
}

// memorySeamBackend — the recBackend PLUS the additive MemoryLane seam
// (the live backend's real contract, mirrored in-test).
type memorySeamBackend struct {
	recBackend
	lane string
}

func (b *memorySeamBackend) MemoryLane() string { return b.lane }

// (g) registration byte-pins: the /help page lists the /memory row (the
// help page IS the grep surface) — the popover-list twin is pinned
// panels-side in popover_cmds_test.go.
func TestMemorySlashHelpRowPinned(t *testing.T) {
	for _, want := range []string{"/memory [filter]", "the office ledger"} {
		if !strings.Contains(slashHelp, want) {
			t.Fatalf("/help must place the /memory row (%q missing):\n%s", want, slashHelp)
		}
	}
}

// (f) the fabric proof: each completed dispatch (EvReturned with a
// return-kind mail — exactly one, live and demo twins alike) stamps the
// "[memory] recorded: <title> → ledger" line into the activity tab AFTER
// its returned line; a board-sync EvTask (even done) NEVER stamps one.
func TestMemoryRecordedActivityLine(t *testing.T) {
	m := newMemoryModel(&recBackend{}, t.TempDir())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = runMsg(t, m, state.Event{Kind: state.EvDispatch, EmployeeID: "tekton-4", Task: state.BoardTask{
		ID: "t1", Title: "Plan-pane gating", Status: state.TaskInProgress, Owner: "tekton-4"}})
	afterDispatch := m.activityAdds
	m = runMsg(t, m, state.Event{Kind: state.EvTask, Task: state.BoardTask{
		ID: "t1", Title: "Plan-pane gating", Status: state.TaskDone, Owner: "tekton-4"}})
	m = runMsg(t, m, state.Event{Kind: state.EvReturned, EmployeeID: "tekton-4", TaskID: "t1",
		Mail: state.MailItem{From: "tekton-4", To: "manager", Subject: "return: Plan-pane gating", Kind: state.MailReturn}})

	// returned + recorded: the return pair adds exactly 3 lines (task
	// upsert, returned, memory recorded).
	if got := m.activityAdds - afterDispatch; got != 3 {
		t.Fatalf("a completed dispatch adds upsert+returned+recorded (3 lines), got %d", got)
	}
	act := ansi.Strip(m.activity.View())
	ir := strings.Index(act, "returned ← tekton-4 «return: Plan-pane gating»")
	im := strings.Index(act, "[memory] recorded: Plan-pane gating → ledger")
	if ir < 0 || im < 0 || im < ir {
		t.Fatalf("the recorded line must follow the returned line (ir=%d im=%d):\n%s", ir, im, act)
	}

	// NEGATIVE: a board-sync task upsert (agentmemory mirror) alone —
	// even done — never claims a record went down.
	m2 := newMemoryModel(&recBackend{}, t.TempDir())
	m2 = runMsg(t, m2, tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 = runMsg(t, m2, state.Event{Kind: state.EvTask, Task: state.BoardTask{
		ID: "t9", Title: "stale synced row", Status: state.TaskDone, Owner: "x"}})
	if strings.Contains(ansi.Strip(m2.activity.View()), "[memory] recorded:") {
		t.Fatalf("an unpaired board-sync done must never stamp a recorded line:\n%s", ansi.Strip(m2.activity.View()))
	}

	// NEGATIVE: a return mail with the WRONG kind is not a record either.
	m3 := newMemoryModel(&recBackend{}, t.TempDir())
	m3 = runMsg(t, m3, tea.WindowSizeMsg{Width: 120, Height: 40})
	m3 = runMsg(t, m3, state.Event{Kind: state.EvReturned, EmployeeID: "hr", TaskID: "tX",
		Mail: state.MailItem{From: "hr", To: "manager", Subject: "policy ping", Kind: state.MailKind("note")}})
	if strings.Contains(ansi.Strip(m3.activity.View()), "[memory] recorded:") {
		t.Fatalf("a non-return-kind mail must never stamp a recorded line")
	}
}
