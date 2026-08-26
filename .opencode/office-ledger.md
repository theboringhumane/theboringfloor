# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-08-26 · Dev A: claude backend stdio core (@general su... — tekton-1 (developer) · `issues`
- summary: All verification is green. The final state of the work:
- files: | path | why |, |---|---|, | `cmd/uishot/main.go` | **Edited this session:** dispatch branch for the pre-declared `*claudeMode`, | `cmd/uishot/claude_proof.go` | **Edited this session:** `sendDeep`/`drainDeep` recursive cmd pump;, | `internal/backend/claude.go` | pre-existing this wave, | `internal/backend/claude_events.go` | pre-existing, | `internal/backend/claude_events_test.go` | pre-existing, | `internal/backend/claude_spawn_test.go` | pre-existing, | `internal/backend/claude_send_test.go` | pre-existing, | `internal/backend/claude_usage_test.go` | pre-existing, | `internal/backend/claude_abort_test.go` | pre-existing, | `internal/backend/claude_perm_test.go` | pre-existing
- verify: (`grep -c "^FAIL\|^--- FAIL"` over the full suite → `0`)
- proof: *A. The stub-captured stdin schedule — the byte-pinned Send→control round-trip contract (verbatim from the run's printed artifact):**
- ledgerId: led-1787755258436-7a82ffef

### 2026-08-26 · Dev B: backend select + switching (@general s... — tekton-2 (developer) · `issues`
- summary: All VERIFY gates pass. Assembling the return.
- files: | path | why |, |---|---|, | `install.sh` | **Edited this session:** `BACKEND_STATE` strings no longer repeat the backend name , | `cmd/theboringoffice/main.go` | Verified: `--backend` flag + THEBORINGOFFICE_BACKEND env + brain.j, | `internal/config/config.go` | Verified: `BackendConfig.Name`, `Default()`, `Load()` backfill, help, | `internal/app/model.go` | Verified: `/backend` routing, | `internal/app/sessions.go` | Verified: `Backend`+`PrimaryIDs` schema, `primaryIDFor`/`mergeBackend, | `internal/app/backend_switch_test.go` | Verified: swap-gate test matrix, | `internal/app/config_backend_test.go` | Verified: resolver/grammar/latch tests, | `cmd/headless/main.go` | Verified: `backendNameLine` boot row. |, | `README.md` | Verified: install selector + `/backend` slash + brain.json note. |
- verify: ```
- proof: *The `/backend` config after a swap** — `TestBackendSwapForceEscThenSwaps` drives `/backend claudecode` and then reads the persisted brain.j
- ledgerId: led-1787755083800-341ecbc9

### 2026-08-26 · Reformat return per contract (@general subagent) — tekton-11 (developer) · `issues`
- summary: Wrote the Claude Code support announcement post, `website/content/blog/claude-code-support.md` — title "The office doesn't care which brain the boss has", dated
- files: | path | why |, |---|---|, | `website/content/blog/claude-code-support.md` | The announcement post |
- verify: ```
- proof: *Blog index (`/blog`)** — new card appears in the filter list under Release / Updates, dated Aug 26, 2026; featured grid untouched:
- ledgerId: led-1787754277263-74b45559

### 2026-08-26 · Dev: claude code blog post (@general subagent) — tekton-10 (developer) · `issues`
- summary: All verified. Final return per the contract:
- files: | path | why |, |---|---|, | `website/content/blog/claude-code-support.md` | The announcement post
- verify: ```
- proof: ````markdown
- ledgerId: led-1787754202911-40e8c557

### 2026-08-26 · Dev: site copy for claudecode (@general subag... — tekton-9 (developer) · `issues`
- summary: All checks pass. The remaining "(Coming Soon)" tags (Codex, Cursor, Pi) are correctly still pending — only Claude Code graduated. Return per the contract:
- files: | path | why |, |---|---|, | `website/app/layout.tsx` | Meta description + OG/Twitter description: "opencode TUI" → "opencode o, | `website/app/get-started/page.tsx` | HOW-to: flag in run card, brain.json pin, `/backend` swap, pe, | `website/app/docs/page.tsx` | Quickstart item: "Connecting opencode" → "Choosing a transport, | `website/app/vision/page.tsx` | Killed stale "Claude Code support is coming", | `website/components/home/hero.tsx` | Hero subcopy: "your opencode boss" → "your opencode or Claude, | `website/components/home/under-the-hood.tsx` | Card 02: event stream now transport-accurate, | `website/components/home/context-model.tsx` | "Any Model, Any Backend" body: removed "coming soon", | `website/components/home/product-platform.tsx` | "real opencode session" → "real agent session, | `website/components/home/why-composio.tsx` | Floor tab: sprite events / walk-cycle points now tran, | `website/components/site-header.tsx` | Products menu: "any opencode agent" → "any opencode or Clau
- verify: Exported-HTML spot check: `out/get-started/index.html` → `backend claudecode: FOUND`, `brain.json: FOUND`, `/backend: FOUND`; `out/index.htm
- proof: *Hero (homepage `components/home/hero.tsx`):**
- ledgerId: led-1787754149948-1d276152

### 2026-08-26 · Scout: claude code interface map (@explore su... — skopos-7 (scout) · `issues`
- summary: Recon complete. Zero files modified — read-only survey (54 reads/greps across `website/`, plus README/install.sh/backend cross-checks for the accuracy claims). 
- files: None touched. Key files read, `website/app/{layout,page,sitemap,robots,get-started/page,docs/page,vision/page,blog/page,blog/layou
- verify: Commands run (all read-only): `rg -n -i 'opencode|claude|claudecode|anthropic' website` (excl. node_modules/out) → 60+ hits catalogued; `rg
- proof: ## (a) File → line → current copy → verdict
- ledgerId: led-1787754085119-b1a1ae34

### 2026-08-26 · Reformat return per contract (@general subagent) — tekton-8 (developer) · `issues`
- summary: Proof regenerated end-to-end on the live tree. Here is my return, re-emitted under the exact required headers.
- files: | path | why |, |---|---|, | `internal/app/model.go` | `stopAbortResultMsg` type + Update case; `stopWork()` returns the async , | `internal/backend/opencode.go` | `abortCallTimeout` + ctx-bounded `abortSession`; `stopDrainTimeou, | `cmd/theboringoffice/main.go` | use `p.Run()`'s final model for all teardown/ExecRequest/PrimarySe, | `internal/app/stop_async_test.go`, | `internal/backend/abort_timeout_test.go`, | `cmd/theboringoffice/main_test.go`, NOT touched
- verify: ```
- proof: Simulated frames from the PTY run (ANSI-stripped, marker lines) — the member-visible result with the abort POST **parked forever** server-si
- ledgerId: led-1787753663410-08175efd

### 2026-08-26 · Reformat return per contract (@general subagent) — tekton-7 (developer) · `issues`
- summary: Recovered my original return content — re-emitting under the exact required headers:
- files: | path | why |, |---|---|, | `internal/app/model.go` | `stopAbortResultMsg` type + Update case; `stopWork()` returns the async , | `internal/backend/opencode.go` | `abortCallTimeout` + ctx-bounded `abortSession`; `Stop()` → `stop, | `cmd/theboringoffice/main.go` | **use `p.Run()`'s final model for all teardown/ExecRequest/Primary, | `internal/app/stop_async_test.go`, | `internal/backend/abort_timeout_test.go`, | `cmd/theboringoffice/main_test.go`, NOT touched
- verify: ```
- proof: `go build -o …/tbofix ./cmd/theboringoffice`, driven through a real PTY (132×40, `TERM=xterm-256color`) by a python driver against a fake se
- ledgerId: led-1787753396307-f9174760

### 2026-08-26 · Dev: fix slash-command UI hang (@general suba... — tekton-3 (developer) · `issues`
- summary: All green on the final tree. Here is the return.
- files: | path | why |, |---|---|, | `internal/app/model.go` | `stopAbortResultMsg` type + Update case; `stopWork()` returns the async , | `internal/backend/opencode.go` | `abortCallTimeout` + ctx-bounded `abortSession`; `Stop()` → `stop, | `cmd/theboringoffice/main.go` | **use `p.Run()`'s final model for all teardown/ExecRequest/Primary, | `internal/app/stop_async_test.go`, | `internal/backend/abort_timeout_test.go`, | `cmd/theboringoffice/main_test.go`, NOT touched
- verify: ```
- proof: `go build -o …/tbofix ./cmd/theboringoffice`, driven through a real PTY (132×40, `TERM=xterm-256color`) by a python driver against a fake se
- ledgerId: led-1787753255200-8ba77a05

### 2026-08-26 · Dev: terminal tab select+copy (@general subag... — tekton-4 (developer) · `issues`
- summary: # Developer Return — terminal mouse selection + clipboard
- files: `internal/panels/terminal.go`, `internal/panels/clipboard.go`, `internal/panels/terminal_sel_test.go`, `README.md`, NOT touched: `internal/app/*`, `internal/backend/*`, `cmd/*`, `internal/term/*` (no accessor gaps fo
- verify: ```
- proof: *Chosen UX (plain):** copy happens **on release** (release-copies, like the chat transcript — no follow-up keypress). Scroll conflict rule:
- ledgerId: led-1787752838508-bdbccc34

### 2026-08-26 · Scout: token cost audit (@explore subagent) — skopos-6 (scout) · `done`
- summary: All recon complete — no files modified (read-only: repo reads, `wc`, and `sqlite3 -readonly` against the serve's own database). Everything below is measured fro
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1787752735460-8014f4cc

### 2026-08-26 · Scout: claude code interface map (@explore su... — skopos-5 (scout) · `issues`
- summary: Recon complete — every hop read, zero files modified. Return per the contract:
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/state/state.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/opencode.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/config/config.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/chrome/topbar.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/sessions.go`, Local: `~/.local/bin/claude`
- verify: `cmd/uishot` + `install.sh --dry-run --backend claudecode` shell test — narrated seed via the existing `[dry-run]` shim; topbar segment rend
- proof: (none)
- ledgerId: led-1787752496998-b3a1b15a

### 2026-08-26 · Scout: hang root-cause map (@explore subagent) — skopos-3 (scout) · `done`
- summary: Recon complete — every hop read. Here is the return per the contract.
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1787752231253-57caa826

### 2026-08-26 · Scout: terminal selection plumbing (@explore... — skopos-4 (scout) · `issues`
- summary: All plumbing mapped. Nothing was written — report follows per the return contract.
- files: No files modified. Read-only recon covered:, `internal/panels/terminal.go`, `internal/panels/chat_selection.go`, `internal/app/selection.go`, `internal/app/model.go`, `internal/app/terminal.go`, `internal/panels/tabs.go`, `internal/panels/popover.go`, `internal/panels/gitpanel.go`, `internal/term/{term,grid,scroll}.go`, `internal/panels/chat_attach.go`, `cmd/theboringoffice/main.go`, `go.mod`, bubbletea v2.0.9 module source
- verify: Read-only brief: no build/test commands run. Evidence is file reads + targeted greps quoted with paths/lines throughout the map below. `tea.
- proof: # Terminal mouse-select + copy: plumbing map
- ledgerId: led-1787752148530-07c6f66d

### 2026-08-26 · Scout: follow-ups form gap (@explore subagent) — skopos-2 (scout) · `done`
- summary: Ready to work. I don't see a specific task or request in this session yet — no search target, bug report, or feature brief has been given.
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1787751903952-f51ea9ad

### 2026-08-26 · Dev: wedge row only on bar and activity (@gen... — tekton-1 (developer) · `issues`
- summary: All verification complete. Final return per the contract:
- files: `internal/app/model.go`, `internal/app/stuck_test.go`, `internal/panels/activity.go`, `cmd/uishot/main.go`, `README.md`, NOT touched (deliberately): `internal/app/sessions.go` (`bootWarnNoticeMeta` Snapshot strip + hydrat
- verify: ```
- proof: *Transcript (chat tab) while wedged — CLEAN, no wedge row:**
- ledgerId: led-1787751850050-ec5940b0

### 2026-08-26 · Dev: projects/hash home path (@general subagent) — tekton-2 (developer) · `issues`
- summary: All verified. Reporting per the return contract.
- files: `internal/app/sessions.go`, `internal/app/sessions_test.go`, `cmd/headless/main.go`, `README.md`, `docs/architecture.md`
- verify: `go build ./... 2>&1 | tail -5`: **(silent — build exit: 0)**
- proof: On-disk layout BEFORE → AFTER (`<home>` = `$THEBORINGOFFICE_HOME` else `$HOME`):
- ledgerId: led-1787751565876-c97c3f5c

### 2026-08-26 · Scout: projects/hash home layout (@explore su... — skopos-1 (scout) · `done`
- summary: Search complete. All touchpoints enumerated below — no files were modified.
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1787751284230-0aeca140

### 2026-08-26 · Dev: sounds library page + assets (@general s... — tekton-2 (developer) · `done`
- summary: Everything verified. Final return per contract:
- files: `cmd/soundexport/main.go`, `website/public/sounds/{queued,send,reply,done,dispatch,alert,error}.wav`, `website/app/sounds/page.tsx`, `website/components/sounds/sound-card.tsx`, `website/components/site-header.tsx`, `website/components/site-footer.tsx`, `website/app/sitemap.ts`
- verify: ```
- proof: *Built HTML region (`out/sounds/index.html`, one card, comment markers stripped):**
- ledgerId: led-1787748809858-082ee7de

### 2026-08-26 · Dev: board sync hook on completion (@general... — tekton-1 (developer) · `issues`
- summary: All checks green and the tree is clean. Final return:
- files: `internal/backend/boardsync.go`, `internal/backend/opencode.go`, `internal/backend/boardsync_test.go`, `cmd/uishot/main.go`, `README.md`, (not mine: `.opencode/*` pre-existing boss-session edits; the tracked `uishot` binary was briefly re
- verify: ```
- proof: ```
- ledgerId: led-1787747811860-aab00add

