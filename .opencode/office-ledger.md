# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-08-28 · Dev Q4: chat previews + bandwidth (@developer... — theboringcto-1 (cto) · `issues`
- summary: Everything is verified. Composing the final return:
- files: `internal/panels/zenbu_frame.go`, `internal/panels/zenbu_frame_test.go`, `internal/panels/chat.go`, `internal/panels/chat_window.go`, `internal/panels/kitty_lanes.go`, `internal/panels/kitty_lanes_test.go`, `internal/app/images.go`, `internal/app/model.go`, `internal/app/images_lane_test.go`, `internal/app/model_image_splice_test.go`, `cmd/uishot/main.go`, NOT touched: `internal/panels/chat_raster.go` (in scope; needed no change
- verify: (the pre-fix capture DOES contain the chip text `paste-diagram` — the office ran and rendered; only the APC was eaten. Post-fix, `/tmp/tb-of
- proof: *1. The wrapper's chat-media splice bytes for the pinned layout** (130×32, sidebar 80 → chat content origin (51,3), media col 51+9=60, slot
- ledgerId: led-1787919903555-dfe6e47f

### 2026-08-28 · Dev Q1: real cell metrics (@developer subagent) — tekton-1 (developer) · `issues`
- summary: All verified. Assembling the return.
- files: `internal/cellmetrics/cellmetrics.go`, `internal/cellmetrics/input.go`, `internal/cellmetrics/cellmetrics_test.go`, `internal/cellmetrics/input_test.go`, `cmd/theboringoffice/main.go`, `internal/panels/browser.go`, `internal/panels/browser_test.go`, `/tmp/drive_cellmetrics.py`, `/tmp/tb-office`
- verify: ```
- proof: *The office's query bytes, live on the PTY** (boot probe lands before bubbletea's own init queries; the re-arm probe lands mid-stream at the
- ledgerId: led-1787919496400-8f1f0b35

### 2026-08-28 · Dev Q3: browser action tools (@developer suba... — tekton-3 (developer) · `issues`
- summary: The `uishot` binary is a foreign build artifact (wave-86's rebuilds — I never build into the repo). All my gates are green. Final return:
- files: `internal/state/state.go`, `internal/browsertools/browsertools.go`, `internal/browsertools/browsertools_test.go`, `internal/browsertools/action/action.go`, `internal/browsertools/action/euid_unix.go` / `euid_other.go`, `internal/browsertools/action/action_test.go`, `internal/browsertools/action/live_test.go`, `internal/browsertools/action/testdata/fixture.html`, `internal/app/browser_open.go`, `internal/app/browser_open_test.go`, `internal/app/model.go`, `internal/backend/opencode.go` / `claude.go`
- verify: ```
- proof: *The new preamble block, verbatim** (appended; everything above it byte-identical — strict-prefix test):
- ledgerId: led-1787919294859-dcdc32a8

### 2026-08-28 · Dev Q2: engine render cache (@developer subag... — tekton-2 (developer) · `issues`
- summary: My hunks are intact and every gate is green. Assembling the return.
- files: `internal/headless/cache.go`, `internal/headless/cache_test.go`, `internal/headless/headless.go`, `internal/app/browser_open.go`
- verify: (all 20 test packages `ok`, incl. `internal/headless 2.171s`, `internal/app 71.222s`, `internal/backend 38.987s`, `internal/panels 13.764s`)
- proof: *1. The singleflight test** (`TestScreenshotSingleflightShare`, through the PUBLIC `Screenshot` front, executor seam swapped): 3 goroutines
- ledgerId: led-1787918541838-9f28997e

### 2026-08-28 · Dev P2: tab screenshot display (@developer su... — tekton-2 (developer) · `issues`
- summary: Everything is green and stable. Assembling the developer return.
- files: `internal/panels/browser.go`, `internal/panels/zenbu_frame.go`, `internal/panels/browser_test.go`, `internal/panels/browser_panel_lane_test.go`, `internal/panels/browser_hint_test.go`, `internal/app/browser.go`, `internal/app/browser_test.go`, `internal/app/browser_frame_test.go`, `internal/app/browser_lane_test.go`, `cmd/uishot/main.go`
- verify: ```
- proof: *1. The shot-mode frame through the REAL app** (`go run ./cmd/uishot --browser --lane shot`, ANSI-stripped — the ` shot ` badge + strip in t
- ledgerId: led-1787914456203-48a66ca2

### 2026-08-28 · Dev P3: agent shot/snapshot tools (@developer... — tekton-3 (developer) · `issues`
- summary: **Markers**: added `⟦browser-screenshot: URL⟧` (`MarkerShot`) and `⟦browser-snapshot: URL⟧` (`MarkerSnap`) to the marker protocol — whole-line rule, own regex p
- files: `internal/browsertools/browsertools.go`, `internal/browsertools/browsertools_test.go`, `internal/state/state.go`, `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/browser_open_test.go`, `internal/app/browser_open.go`, `internal/app/browser_open_test.go`
- verify: ```
- proof: *The new PromptPreamble, verbatim (as it rides the first prompt on both backends):**
- ledgerId: led-1787913705452-62a86d3a

### 2026-08-28 · Dev P4: zenbu default-off + docs (@developer... — tekton-4 (developer) · `issues`
- summary: All checks complete. Assembling the return.
- files: `internal/panels/browser_lane.go`, `internal/panels/browser_lane_test.go`, `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane_kitty_test.go`, `install.sh`, `README.md`, `website/app/docs/browser-tab/page.tsx`, NOT touched
- verify: ```
- proof: *The new reason-matrix rows** (`TestBrowserLaneOptInGate`, verbatim env → `(lane, reason, var)`):
- ledgerId: led-1787912379395-954f78bc

### 2026-08-28 · Dev P1: headless engine pkg (@developer subag... — tekton-1 (developer) · `issues`
- summary: Created `internal/headless` — the pure-Go CDP (chromedp) engine: `Available` (memoized discovery: `THEBORINGOFFICE_CHROME` → macOS `/Applications` Chrome/Chromi
- files: `internal/headless/headless.go`, `internal/headless/euid_unix.go` / `euid_other.go`, `internal/headless/headless_test.go`, `internal/headless/live_test.go`, `internal/headless/testdata/fixture.html`, `go.mod`, `go.sum`
- verify: $ `go test ./... -count=1 -p 4 2>&1 | grep -cE '^FAIL'` → **6, not 0** — all foreign (`internal/panels`, `internal/app`, one transient mid-e
- proof: *Discovery on this machine** (from live test logs):
- ledgerId: led-1787911901187-88662352

### 2026-08-28 · Dev N: fix freeze chain-tail leak (@developer... — tekton-2 (developer) · `issues`
- summary: All gates green, 5× stability run clean, final tree state confirmed. The `.opencode/office-ledger.md` modification is the office app's own recorder, not mine.
- files: `internal/panels/browser_lane.go`, `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane_test.go`
- verify: Stability: `go test ./internal/panels -count=5 -run 'TestBrowserLaneFreezeMidChainReal|TestBrowserLaneFrozenDeathMidChainReset'` → `ok … 12.
- proof: *The mid-chain freeze → thaw cycle (real streaming fake, ~2fps, 684-char chains):**
- ledgerId: led-1787909844914-65cad1e4

### 2026-08-28 · Scout: ghostty kitty paint rules (@explore su... — skopos-2 (scout) · `issues`
- summary: I have complete primary-source coverage of both the latest stable (v1.3.1) and main. Assembling the return.
- files: No files touched
- verify: Read-only task; no repo commands run. Evidence is quoted source + links below. Version provenance: `https://api.github.com/repos/ghostty-org
- proof: ### 1. Support matrix: f=32 (RGBA raw) and o=z (zlib) — supported on both versions
- ledgerId: led-1787909757974-d43f07e5

### 2026-08-28 · Dev N: lane keep-alive suspend (@developer su... — tekton-2 (developer) · `issues`
- summary: All green, no foreign leakage, no commits. The work is complete — assembling the return.
- files: `internal/panels/browser_lane.go`, `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane_test.go`, `internal/panels/browser_panel_lane_test.go`, `internal/app/browser.go`, `internal/app/browser_lane_test.go`, `internal/app/browser_frame_test.go`, `cmd/uishot/main.go`, `/tmp/drive_office_pty_keepalive.py` + rebuilt `/tmp/tb-office`, NOT touched
- verify: ```
- proof: *The flip cycle's byte trace (REAL office + REAL Electron terminal-browser, 160×48 PTY, ghostty env — run 2, pid 49126):**
- ledgerId: led-1787906810427-c11441a2

### 2026-08-28 · Dev A: attribution default-on core (@develope... — tekton-3 (developer) · `issues`
- summary: All checks green. Assembling the final return.
- files: `internal/config/config.go`, `internal/config/config_attribution_test.go`, `internal/app/attribution.go`, `internal/app/attribution_test.go`, `cmd/theboringoffice/main.go`
- verify: ```
- proof: *(a) The new brain.json skeleton** — verbatim head of `go run ./cmd/theboringoffice --print-default-config` (what a first boot writes):
- ledgerId: led-1787906218308-ceabf988

### 2026-08-28 · Dev B: attribution docs + scripts (@developer... — tekton-4 (developer) · `issues`
- summary: All checks green. Assembling the final return.
- files: `README.md`, `scripts/install-majdoor-hook.sh`, `scripts/majdoor-commit-msg-hook.sh`, `install.sh`, NOT touched: `scripts/majdoor-env.sh`
- verify: ```
- proof: *The rewritten README attribution section, verbatim as a reader will see it (README.md L434–479):**
- ledgerId: led-1787905461465-2aa38b80

### 2026-08-28 · Scout: majdoor default-on seams (@explore sub... — skopos-1 (scout) · `issues`
- summary: All evidence gathered; the tree was never touched. Return per contract:
- files: No files written or modified
- verify: Grep `WithMajdoorAuthorEnv|MajdoorEnvMerge` repo-wide (`*.go`) → 13 matches across 6 files (callers: term.go:112, headless/main.go:1076, ope
- proof: The recon itself is the artifact — see **Findings** (Q1–Q9 with verbatim snippets), **Seam recommendation** (main.go after L95; function in
- ledgerId: led-1787905355035-7b614236

### 2026-08-28 · Dev M: lane flicker/size/leak fixes (@develop... — tekton-1 (developer) · `issues`
- summary: All 32 kitty/zenbu tests pass (25 pre-existing + 7 new). Every gate verified. Final return:
- files: `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane.go`, `internal/panels/browser_lane_kitty_test.go`, `internal/app/browser_frame_test.go`, `cmd/uishot/main.go`, (`internal/panels/zenbu_frame.go`, `zenbu_frame_test.go`
- verify: ```
- proof: *Before → after, the drive's APC key lines** (the manager's wave-81 analysis of the same child: every frame a *new* content-hash id, `a=d` b
- ledgerId: led-1787904770440-a1216f17

### 2026-08-27 · Dev L: move splice to WithOutput (@developer... — tekton-2 (developer) · `issues`
- summary: All gates green. Assembling the final return.
- files: `internal/panels/zenbu_frame.go`, `internal/panels/zenbu_frame_test.go`, `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane.go`, `internal/panels/browser_lane_kitty_test.go`, `internal/app/browser.go`, `internal/app/model.go`, `internal/app/browser_frame_test.go`, `cmd/theboringoffice/main.go`, `cmd/uishot/main.go`
- verify: Pre-fix it printed `ESC_G sequences: 0` / `a=T present: False` (18,668 bytes). Now: **34 ESC_G, a=T True** — the REAL child's `f=32,o=z,s=12
- proof: *The wrapper's emitted bytes for a known registry state** (TestZenbuFrameWriterSplice: origin (0,3), image pane-local (5,2), APC `\x1b_Ga=T,
- ledgerId: led-1787858669896-648f7454

### 2026-08-27 · Dev L: kitty passthrough lane (@developer sub... — tekton-1 (developer) · `issues`
- summary: All gates green. Assembling the final return.
- files: `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane.go`, `internal/panels/browser_lane_kitty_test.go`, `cmd/uishot/main.go`, `.opencode/office-ledger.md`
- verify: New tests: `TestKittyStream{SplitCleanGrid,SplitBoundaries,BELTerminator,ChainFinalChunkOmitsM,ProbeIgnored,Malformed,APCBodyCap}`, `TestZen
- proof: *1. The REAL stream (825KB captured from terminal-browser v0.6.0 on a PTY) — before → after:**
- ledgerId: led-1787856360063-526c5f07

### 2026-08-27 · Scout: kitty escape surface map (@explore sub... — skopos-1 (scout) · `issues`
- summary: Recon complete. All evidence gathered; the tree was never modified.
- files: `internal/panels/kitty_lanes.go`, `chat_raster.go`, `image_detect.go`, `internal/panels/chat.go`, `tabs.go`, `terminal.go`, `internal/panels/browser_lane.go`, `browser.go`, `internal/term/term.go`, `grid.go`, `notify.go`, `internal/app/model.go`, `browser.go`, `images.go`, `power.go`, `cmd/theboringoffice/main.go`, `cmd/uishot/main.go`, `go.mod`, `README.md`, module cache: `charm.land/bubbletea/v2@v2.0.9/{tea,options,cursed_renderer}.go`, `github.com/charmbr
- verify: (none)
- proof: (none)
- ledgerId: led-1787855712243-206df16b

### 2026-08-27 · Dev J: text-lane reason hint (@developer suba... — tekton-1 (developer) · `issues`
- summary: All gates green. Assembling the final return.
- files: `internal/panels/browser_lane.go`, `internal/panels/browser.go`, `internal/panels/browser_hint_test.go`, `cmd/uishot/main.go`, `README.md`, NOT touched
- verify: ```
- proof: *The ANSI-stripped starter card with the binary-missing hint** (`go run ./cmd/uishot --browser --lane hint`, leg E — hermetic empty-PATH fix
- ledgerId: led-1787853075573-5c5d2d1c

### 2026-08-27 · Dev K: installer terminal-browser step (@deve... — tekton-2 (developer) · `issues`
- summary: All gates green. Assembling the final return.
- files: `install.sh`, `internal/panels/browser_lane.go` shows modified in `git status`
- verify: *7. shellcheck:** NOT on PATH (`which shellcheck` → not found) — could not run; see ISSUES.
- proof: *Help text — before → after:**
- ledgerId: led-1787852194575-4788f24f

### 2026-08-27 · Dev I: abort-ladder test budgets (@developer... — tekton-6 (developer) · `issues`
- summary: **Fully green loaded run** — `grep -cE '^FAIL'` → **0**, my test `--- PASS` under full-suite load, and even the previously-flaked panels test and the concurrent
- files: `internal/backend/claude_abort_test.go`
- verify: ```
- proof: Before → after of each raised budget (verbatim diff hunks):
- ledgerId: led-1787849348047-ef91c581

### 2026-08-27 · Dev H: perm SessionID + wire fixtures (@devel... — tekton-5 (developer) · `issues`
- summary: **FIX A (SessionID fallback):** `mapClaudeControlRequest`'s `can_use_tool` arm now computes `sessionID := raw.SessionID; if sessionID == "" { sessionID = ctx.pr
- files: `internal/backend/claude_events.go`, `internal/backend/claude_wire_21247_test.go`, `internal/backend/claude_events_test.go`
- verify: ```
- proof: *FIX A — the mapper before→after** (verbatim `git diff internal/backend/claude_events.go`):
- ledgerId: led-1787849344177-bd8f7527

### 2026-08-27 · Dev G: real claude CLI integration (@develope... — tekton-4 (developer) · `issues`
- summary: Everything is green and evidence-complete. Assembling the final return.
- files: `internal/backend/claude_live_test.go`, `internal/backend/claude.go`
- verify: My diff cannot reach that path: the seam is one nil-guarded hook call per parsed stdout frame; the abort stubs never set it.
- proof: *Test 1 — one bubble per message.id, live:** the real wire produced ONE assistant message (`msg_bdrk_01VYNbLS7HFdMQdjnwLFGmNe`) across TWO a
- ledgerId: led-1787848497729-bcf8d3de

### 2026-08-27 · Dev E: wire zenbu lane live (@developer subag... — tekton-2 (developer) · `issues`
- summary: All gates green. Assembling the final return.
- files: `internal/panels/browser.go`, `internal/app/browser.go`, `internal/panels/browser_panel_lane_test.go`, `internal/app/browser_lane_test.go`, `internal/panels/browser_nav_test.go`, `internal/app/browser_test.go`, `internal/app/browser_open_test.go`, `cmd/uishot/main.go`, `README.md`, NOT touched (foreign in-flight work, ledger led-1787844677003 / led-1787846237568): `install.sh`, `i
- verify: $ `go run ./cmd/uishot --browsertab` → `exit=0` (byte-identical two drives, text lane unchanged)
- proof: *Leg C — `/open` through the REAL app: the zenbu lane live in the LEFT slot** (ANSI-stripped, `go run ./cmd/uishot --browser --lane live`):
- ledgerId: led-1787847438500-935cead5

### 2026-08-27 · Dev D: claude bubble/perm/think fixes (@devel... — tekton-1 (developer) · `issues`
- summary: All gates green — 18/18 packages ok. Assembling the final return.
- files: `internal/backend/claude_regression_test.go`, `internal/backend/claude.go`, `internal/backend/claude_events.go`
- verify: (all 18 packages `ok` in the clean sweep, including `internal/backend 31.974s` and `internal/panels 4.418s`. The two Requirement-6 flakes —
- proof: ### Bug 1 — the two IDs and the single-bubble assertion
- ledgerId: led-1787846237568-0edae27f

### 2026-08-27 · Dev F: fix installer npm 404 (@developer suba... — tekton-3 (developer) · `issues`
- summary: **Renamed the npm package to the real scoped name** `@agentmemory/agentmemory` in both mandated spots: the auto-install command inside `setup_agentmemory` and t
- files: `install.sh`
- verify: The run really executed end-to-end (downloaded v0.2.23, installed the binary into the scratch prefix, seeded scratch brain.json) — and still
- proof: *Before → after, auto-install (setup_agentmemory):**
- ledgerId: led-1787844677003-48d81f07

### 2026-08-27 · Commit, tag v0.2.23, push release (@developer... — tekton-14 (developer) · `issues`
- summary: Full suite green before any commit: `go test ./...` all packages `ok`, `go build ./...` exit 0, gofmt clean on every pending file.
- files: n/a (no content edits
- verify: ```
- proof: $ `git log --oneline -6`
- ledgerId: led-1787840610933-b0e063ec

### 2026-08-27 · Render all dialog kinds (@developer subagent) — tekton-13 (developer) · `issues`
- summary: **`supportedDialogKinds` declaration wired, evidence-backed**: the office now writes an `initialize` control_request as the FIRST stdin line of every claude pro
- files: `internal/backend/claude.go`, `internal/backend/claude_events.go`, `internal/backend/claude_dialog_test.go`, `internal/backend/claude_dialog_kinds_test.go`, `internal/backend/claude_send_test.go`, `README.md`
- verify: Post-restore: `diff` backup vs live for both files → `RESTORE-IDENTICAL`, full `-run Claude` suite `ok 12.572s`.
- proof: ### (a) The initialize/declaration evidence + exact bytes
- ledgerId: led-1787840395521-eb7c1e8a

### 2026-08-27 · Scout all dialog-kind schemas (@explore subag... — skopos-3 (scout) · `issues`
- summary: Cleanup done. Assembling the final return.
- files: Scratch only (repo untouched, | path | why |, |---|---|, | `/var/folders/d4/5hc5gjcd18x7np_cy6yzbwm80000gn/T/opencode/claude-strings-allkinds.txt` | fresh `s, | `/var/folders/d4/5hc5gjcd18x7np_cy6yzbwm80000gn/T/opencode/claude-strings-offsets.txt` | `strings , | `/var/folders/d4/5hc5gjcd18x7np_cy6yzbwm80000gn/T/opencode/extract_kinds2.py` | balanced-brace reg, | `/var/folders/d4/5hc5gjcd18x7np_cy6yzbwm80000gn/T/opencode/kind_registrations2.json` | all 35 raw , | `/var/folders/d4/5hc5gjcd18x7np_cy6yzbwm80000gn/T/opencode/print_kinds.py`, `resolve4.py`, `map_ti, `git -C … status --porcelain` shows only pre-existing parallel-agent modifications
- verify: ```
- proof: ### 0. De-minification legend (resolved via chunk export chains, verbatim evidence)
- ledgerId: led-1787838913882-ea93da52

### 2026-08-27 · Round-trip preview+metadata fidelity (@develo... — theboringcto-1 (cto) · `done`
- summary: Restore byte-identical, and the state.go diff shows exactly my additive block (the `EvBrowserOpen` hunk below it is the parallel browser-lane agent's pre-existi
- files: `internal/state/state.go`, `internal/backend/claude_events.go`, `internal/backend/claude.go`, `internal/backend/claude_dialog_test.go`
- verify: Post-restore: `diff` backup vs `internal/backend/claude.go` → `RESTORE_IDENTICAL`; both full suites above were run AFTER the restore.
- proof: *Decode side** (`internal/backend/claude_events.go`, verbatim):
- ledgerId: led-1787838662008-5c1659b1

### 2026-08-27 · Fix dialog result object shape (@developer su... — tekton-12 (developer) · `issues`
- summary: All gates green. Assembling the return.
- files: `internal/backend/claude.go`, `internal/backend/claude_dialog_test.go`, `internal/backend/claude_perm_test.go`, (scratch, sanctioned tmp dir, not repo: `claude-strings-result.txt`, `claude.go.bak`)
- verify: (claude.go then restored from backup; suites above re-ran AFTER restore; gofmt clean.)
- proof: ### (a) Binary evidence (CLI 2.1.247, `strings -a` + python windowed slices, verbatim)
- ledgerId: led-1787838096617-fe510ca9

### 2026-08-27 · Fix claude dialog request parsing (@developer... — tekton-11 (developer) · `issues`
- summary: **Re-keyed `claudeControlRequest` request-side dialog parsing to the real CLI 2.1.247 wire shape**: removed the nonexistent flat `Question string json:"question
- files: `internal/backend/claude_events.go`, `internal/backend/claude_perm_test.go`, `internal/backend/claude_dialog_test.go`, NOT touched: `internal/backend/claude.go` (response side preserved byte-for-byte), app/panels/state/
- verify: 2. Payload decode skipped (`if false && len(req.Payload) > 0`) → `--- FAIL: TestClaudeDialogRequestMapping … want 1 event, got 0: []` AND `-
- proof: ### (a) Binary evidence (CLI 2.1.247, extracted via `strings -a` + python slicing)
- ledgerId: led-1787837527276-896494e8

### 2026-08-27 · Harden boss bubble pin (@developer subagent) — tekton-9 (developer) · `issues`
- summary: All gates green. The fix is complete and verified. Final return:
- files: `internal/backend/claude_events.go`, `internal/backend/claude_events_test.go`
- verify: (`Text:Honey nev Pending:false` under the mutation IS the original freeze bug, caught red-handed.) After restore: `--- PASS` on all three, `
- proof: The new pin logic (verbatim from `internal/backend/claude_events.go`):
- ledgerId: led-1787836933824-b73526fd

### 2026-08-27 · Verify+fix claude dialog shape (@developer su... — tekton-10 (developer) · `issues`
- summary: All gates green. Assembling the return.
- files: `internal/backend/claude.go`, `internal/backend/claude_perm_test.go`, `internal/backend/claude_dialog_test.go`
- verify: ```
- proof: *(a) Binary evidence — the dialog response schema (CLI 2.1.247, verbatim zod literal from the strings extraction):**
- ledgerId: led-1787836807067-29bb7139

### 2026-08-27 · Fix claude permission protocol (@developer su... — tekton-8 (developer) · `issues`
- summary: **RC1 (primary, modal never appears):** `spawnClaude` argv now appends `--permission-prompt-tool stdio` (claude.go). Headless `claude -p` only wires `canUseTool
- files: `internal/backend/claude.go`, `internal/backend/claude_events.go`, `internal/backend/claude_perm_test.go`, `internal/backend/claude_spawn_test.go`, NOT touched
- verify: $ `claude --version` → `2.1.247 (Claude Code)`
- proof: *The new argv (claude.go, verbatim):**
- ledgerId: led-1787836154304-1e40d050

### 2026-08-27 · Scout claude permission flow (@explore subagent) — skopos-2 (scout) · `issues`
- summary: All evidence is in hand. Tree state is byte-identical to the start (the dirty paths belong to parallel agents' browser-lane work; my only writes were scratch fi
- files: | path | why read |, |---|---|, | `internal/backend/claude.go` | spawn argv, | `internal/backend/claude_events.go` | `claudeControlRequest` struct, | `internal/backend/claude_perm_test.go` | the assumed wire shapes in existing tests, | `internal/backend/opencode.go` | comparison `AnswerPermission`, | `internal/state/state.go` | `EvPermission` kind, | `internal/app/model.go` | event routing, | `internal/panels/chat.go` | `renderTool` ✗-on-error glyph, | `internal/panels/perm_modal.go` | modal rows "Allow once/Allow always/Reject", | `/Users/theboringhumane/.local/share/claude/versions/2.1.247`, Temp scratch
- verify: `claude --help`, `claude --version` (read-only invocations; `--permission-prompt-tool` is hidden from help — matches `.hideHelp()` in the bi
- proof: ### Flow as-built today (frame → parse → event → modal → answer)
- ledgerId: led-1787835847817-078bc776

### 2026-08-27 · Fix claude thinking rendering (@developer sub... — tekton-7 (developer) · `issues`
- summary: All gates green on the settled tree. Final return:
- files: `internal/backend/claude_events.go`, `internal/backend/claude_events_test.go`, NOT touched: `internal/panels/*` (bug was purely normalization
- verify: (panels suite not run — panels untouched, per brief.)
- proof: *The fixed blocks (verbatim from `git diff internal/backend/claude_events.go`):**
- ledgerId: led-1787835813091-80b4e238

### 2026-08-27 · Fix mapClaudeAssistant ID priority (@develope... — tekton-6 (developer) · `done`
- summary: Flipped the ID priority in `mapClaudeAssistant`: now `uuid := raw.Message.ID` with fallback to `raw.UUID` (was UUID-first), so assistant snapshots join the bubb
- files: `internal/backend/claude_events.go`, `internal/backend/claude_events_test.go`
- verify: ```
- proof: *The flipped block** (`internal/backend/claude_events.go:784–787`, verbatim):
- ledgerId: led-1787835389395-057f2460

### 2026-08-27 · Dev C: correct browser docs copy (@developer... — tekton-5 (developer) · `done`
- summary: **CORRECTION 1 (layout):** hero body now says the left pane is a two-tab slot (floor default, browser behind `ctrl+b`) instead of "The sidebar's last tab"; meta
- files: `website/app/docs/browser-tab/page.tsx`, `website/components/home/under-the-hood.tsx`
- verify: Sanity sweeps: `grep -o "sidebar's last tab|No digit key in v1|unlocks outbound…"` → only the new `unlocks outbound http` matches; `grep -c
- proof: ### Hero (framing line kept, body corrected)
- ledgerId: led-1787834994310-6f96bdbd

### 2026-08-27 · Dev A: browser to left pane (@developer subag... — tekton-1 (developer) · `issues`
- summary: All gates green. Assembling the return.
- files: `internal/app/browser.go`, `internal/app/model.go`, `internal/app/digest.go`, `internal/app/keys.go`, `internal/panels/tabs.go`, `internal/app/browser_test.go`, `internal/app/mobile_test.go`, `cmd/uishot/main.go`, `README.md`, NOT touched: `internal/backend/*`, `internal/state/*`, `internal/browsertools/`, `internal/app/brows
- verify: ```
- proof: *The new layout (ANSI-stripped `--browsertab` frame)** — switcher strip on the LEFT with ` browser ` active (accent bg in raw ANSI), fixture
- ledgerId: led-1787834270085-4aa2ae31

### 2026-08-27 · Dev B: agent browser-open tool (@developer su... — tekton-2 (developer) · `issues`
- summary: All verification complete. The full-suite gate is green on a settled tree, with two provably foreign load-flakes documented below. Assembling the return.
- files: `internal/browsertools/browsertools.go`, `internal/browsertools/browsertools_test.go`, `internal/state/state.go`, `internal/state/state_test.go`, `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/browser_open_test.go`, `internal/backend/claude_send_test.go`, `internal/backend/claude_spawn_test.go`, `internal/app/browser_open.go`, `internal/app/browser_open_test.go`
- verify: (Full suite re-run 25+ times during this session — green on a settled tree; the intermittent bursts are two foreign load-flakes, evidence in
- proof: *The agent-visible tool definition** — the exact `browsertools.PromptPreamble` text that rides the first prompt of every boss session on bot
- ledgerId: led-1787833798237-5057c76d

### 2026-08-27 · Dev C: website browser feature (@developer su... — tekton-3 (developer) · `issues`
- summary: All gates pass. Assembling the return.
- files: `website/app/docs/browser-tab/page.tsx`, `website/app/docs/page.tsx`, `website/app/sitemap.ts`, `website/components/home/under-the-hood.tsx`
- verify: ```
- proof: ### The new docs page (`/docs/browser-tab`) — the copy a visitor reads
- ledgerId: led-1787811364148-dcb78f2e

### 2026-08-27 · Scout: left-pane + tool seams (@explore subag... — skopos-1 (scout) · `done`
- summary: # Recon Map — theboringoffice (read-only, tree untouched)
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1787811295889-6d3b00cc

### 2026-08-27 · Dev: fix claude spawn init gating (@general s... — tekton-9 (developer) · `issues`
- summary: # Developer Return — claude backend: no-init-wait Start + real `~/.claude` default
- files: | path | why |, |---|---|, | `internal/backend/claude.go` | Start rewrite, | `internal/backend/claude_spawn_test.go` | rewritten: hook-first fixture + 6 new contract tests, | `internal/backend/claude_send_test.go` | stub bodies → `claudeStubPreambleSh()`, | `internal/backend/claude_abort_test.go` | same fixture swap, | `internal/backend/claude_perm_test.go` | same fixture swap, | `internal/backend/claude_resume_test.go` | hook prefix + post-Start `PrimaryID` assert wrapped in , NOT touched: `cmd/uishot/*`
- verify: ```
- proof: *opencode → `/backend claudecode` → Start returns nil immediately → send a prompt → system/init maps on arrival → primaryID pinned + turn pr
- ledgerId: led-1787807516525-e1344b4d

### 2026-08-27 · Dev B: email+account in scripts/docs (@develo... — tekton-8 (developer) · `done`
- summary: Swapped every `majdoor@office.theboring.name` → `themajdoor@theboring.name` in the three scripts and README, including the escaped grep/marker patterns (`majdoo
- files: `scripts/majdoor-commit-msg-hook.sh`, `scripts/install-majdoor-hook.sh`, `scripts/majdoor-env.sh`, `README.md`
- verify: (repo-wide `grep -rln "majdoor@office"` now returns NOTHING — the parallel Go-lane developer's files no longer carry the old email either; m
- proof: *Rendered README account paragraph (README.md L394–409, verbatim from `sed -n '394,410p'`):**
- ledgerId: led-1787806050847-faf32bb5

### 2026-08-27 · Dev A: email swap in Go lane (@developer suba... — tekton-7 (developer) · `done`
- summary: All gates green. Final return:
- files: `internal/gitx/attribution.go`, `internal/gitx/attribution_test.go`, `internal/gitx/attribution_env_test.go`, `internal/term/term_test.go`
- verify: (zero matches — brief-required grep returns NOTHING; case-insensitive sweep `grep -rni` also exit 1, zero matches)
- proof: New const lines (`internal/gitx/attribution.go` L12–20, pasted verbatim from `sed -n '12,20p'`):
- ledgerId: led-1787806022904-e40224a2

### 2026-08-27 · Scout: env/spawn seams map (@explore subagent) — skopos-2 (scout) · `issues`
- summary: All recon complete. Tree state confirmed: the dirty/untracked files belong to parallel agents (browser lane, attribution wave) — my tools were read/grep/glob pl
- files: | path | why read |, |---|---|, | `internal/term/term.go` | THE shell spawn seam (`Spawn` L92–134), | `internal/gitx/attribution.go` | existing Majdoor consts + trailer helper, | `internal/gitx/gitx.go` | `execGit` L73–89 + frozen read-only contract L1–15, | `internal/backend/opencode.go` | `spawnServe` L876–949, | `internal/backend/claude.go` | `claudeChildEnv` allowlist L190–217, `spawnClaude` L223–254, CLAUDE, | `internal/backend/charter.go` | `envOrLegacy` helper L215–223, NO_AUTOCHARTER read L74 |, | `internal/panels/terminal.go` | `spawnTermSession` seam L117 + `spawn()` L185–192, | `internal/panels/browser_lane.go` | zenbu PTY spawn L217–232, BrowserLaneOffEnv L90–131 |, | `internal/panels/links.go` | TerminalBrowserOffEnv L255–291, terminalBrowserOpen env passthrough L, | `internal/app/terminal.go` | `SpawnTerminal` factory seam L65–136
- verify: (none)
- proof: (none)
- ledgerId: led-1787805127958-5a6f8c5b

### 2026-08-27 · Dev B: env snippet + docs (@developer subagent) — tekton-6 (developer) · `done`
- summary: All gates green, including the zsh-sourcing edge case (silent both ways, exports only when the flag is `true`).
- files: `scripts/majdoor-env.sh`, `README.md`
- verify: ```
- proof: *Rendered README section (README.md L394–447):**
- ledgerId: led-1787805088486-f29a8929

### 2026-08-27 · Dev B: hook installer + docs (@developer suba... — tekton-4 (developer) · `issues`
- summary: **NEW `scripts/majdoor-commit-msg-hook.sh`** (755) — POSIX sh `commit-msg` hook, `set -eu`, grep/sed only: appends `Co-authored-by: TheBoringMajdoor <majdoor@of
- files: `scripts/majdoor-commit-msg-hook.sh`, `scripts/install-majdoor-hook.sh`, `README.md`, `install.sh`
- verify: Repo with `git config core.hooksPath .githooks`: installer resolved hooks dir to `.../.githooks`, hook landed there, `.git/hooks` kept only
- proof: *Rendered README section (as it appears in README.md, lines 394–425):**
- ledgerId: led-1787804936780-819340bb

### 2026-08-27 · Dev A: Majdoor trailer in Go commits (@develo... — tekton-3 (developer) · `issues`
- summary: My only additions are the two new `internal/gitx/attribution*` files; all other dirty paths belong to parallel agents (browser lane work, ledger appends) — unto
- files: `internal/gitx/attribution.go`, `internal/gitx/attribution_test.go`
- verify: ```
- proof: Real test case (passing subtest `existing other trailer → appended inside the block, no blank line`), BEFORE → AFTER:
- ledgerId: led-1787804671681-a5a7aa6a

