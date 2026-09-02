# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-02 · Final bypass signoff (@reviewer subagent) — theboringcto-3 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788351738668-04574287

### 2026-09-02 · Clean failed bypass candidate (@developer sub... — tekton-5 (developer) · `done`
- summary: Added asynchronous, single-path teardown for discarded backend candidates through `stopDiscardedBackend`.
- files: `internal/app/model.go`, `internal/app/bypass_test.go`, `internal/app/current_backend_test.go`
- verify: ```
- proof: ### Partially spawned bypass candidate fails
- ledgerId: led-1788351629817-106d98c0

### 2026-09-02 · Final bypass signoff (@reviewer subagent) — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788351092796-035740f4

### 2026-09-02 · Harden OpenCode bypass process (@developer su... — tekton-4 (developer) · `issues`
- summary: Made `spawnServe`’s reaper the sole `cmd.Wait` owner via a shared `serveExit` completion object.
- files: `internal/backend/opencode.go`, `internal/backend/bypass_permissions_test.go`
- verify: ```
- proof: ### Sole Wait-owner lifecycle
- ledgerId: led-1788350621832-afd8c3c2

### 2026-09-02 · Review bypass end-to-end (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788350055458-01a3ea02

### 2026-09-02 · Fix bypass transition state machine (@develop... — tekton-1 (developer) · `issues`
- summary: Reworked `/bypass` lifecycle so the active-mode flag is separate from the requested mode:
- files: `internal/app/model.go`, `internal/app/bypass_test.go`
- verify: ```
- proof: ### Bypass lifecycle timeline
- ledgerId: led-1788349714135-f9cdb7a2

### 2026-09-02 · Fix OpenCode bypass isolation (@developer sub... — tekton-2 (developer) · `issues`
- summary: Replaced OpenCode’s persistent bypass config mutation with a process-scoped `OPENCODE_CONFIG_CONTENT={"permission":{"*":"allow"}}` child environment override.
- files: `internal/backend/opencode.go`, `internal/backend/charter.go`, `internal/backend/bypass_permissions_test.go`
- verify: ```
- proof: ### Bypass ON — spawned OpenCode child
- ledgerId: led-1788349304507-ebbb42f2

### 2026-09-02 · Fix Claude bypass execution (@developer subag... — tekton-3 (developer) · `issues`
- summary: Retained the bypass mode across **every Claude process spawn path**: initial `Start`, death-respawn, `NewOffice`, `SwapPrimary`, and `ReconnectMCP`’s deferred r
- files: `internal/backend/claude.go`, `internal/backend/claude_bypass_test.go`
- verify: ```
- proof: ### Spawn argv inventory
- ledgerId: led-1788349152823-5095aa6e

### 2026-09-02 · Final runtime routing signoff (@reviewer suba... — theboringcto-4 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788325711855-c6f0404f

### 2026-09-02 · Finalize backend generation lifecycle (@devel... — tekton-8 (developer) · `issues`
- summary: Implemented explicit backend-generation lifecycle: **accepting → draining → retired**.
- files: `internal/app/model.go`, `internal/app/current_backend_test.go`, `internal/app/bypass_test.go`
- verify: ```
- proof: ### Backend admission/swap timeline
- ledgerId: led-1788325543015-a2213d3e

### 2026-09-02 · Recheck runtime attachment routing (@reviewer... — theboringcto-3 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788324150414-94ab5827

### 2026-09-02 · Make backend replacement nonblocking (@develo... — tekton-7 (developer) · `issues`
- summary: Reworked the current-backend holder to lease a backend generation under a short mutex, then perform `Send`/`SendWith` without holding a lock.
- files: `internal/app/model.go`, `internal/app/current_backend_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788324040808-146edbc8

### 2026-09-02 · Review runtime attachment routing (@reviewer... — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788323285559-6197cebd

### 2026-09-02 · Independent cost regression review (@general... — theboringcto-1 (cto) · `issues`
- summary: Performed a read-only audit of OpenCode and Claude Code usage parsing, reducer accumulation, and status-bar rendering.
- files: `internal/backend/events.go`, `internal/backend/usage_test.go`, `internal/backend/claude_events.go`, `internal/backend/claude_usage_test.go`, `internal/app/model.go`, `internal/app/usage_test.go`, `internal/chrome/statusbar.go`, `internal/chrome/statusbar_test.go`, `internal/state/state.go`, `internal/state/state_test.go`
- verify: ```
- proof: | Scenario | Provider pricing semantics | Expected session result | Actual code result | Verdict |
- ledgerId: led-1788322779833-c2a5f22a

### 2026-09-02 · Trace session cost math (@explore subagent) — skopos-1 (scout) · `issues`
- summary: **Verdict: cache-token pricing adjustments are fully included in the repository’s displayed session cost, but only indirectly.** The repository does **not** cal
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/state/state.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/model.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/usage_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/usage_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/claude_events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/claude_usage_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/chrome/statusbar.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/chrome/statusbar_test.go`, No files were modified.
- verify: ```
- proof: ### Worked OpenCode token-count-to-dollar example
- ledgerId: led-1788322730566-cb94eae6

### 2026-09-02 · Audit cache token provenance (@explore subagent) — skopos-2 (scout) · `issues`
- summary: Traced OpenCode `message.updated` and Claude `result` usage payloads into `state.EvUsage`.
- files: None modified, Inspected:, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/claude_events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/usage_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/claude_usage_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/state/state.go`
- verify: ```
- proof: | Provider payload / field | Parse and normalization evidence | `EvUsage` accounting field | Cache-cost treatment / verdict |
- ledgerId: led-1788322718999-e6820584

### 2026-09-02 · Fix current backend routing (@developer subag... — tekton-4 (developer) · `issues`
- summary: Replaced the constructor-captured backend chat callback with a shared, synchronized current-backend holder resolved when the `tea.Cmd` executes.
- files: `internal/app/model.go`, `internal/app/current_backend_test.go`
- verify: ```
- proof: ### Ordinary Enter backend call ledger
- ledgerId: led-1788322652683-7b1c830c

### 2026-09-02 · Add app attachment wire tests (@developer sub... — tekton-6 (developer) · `issues`
- summary: Added end-to-end app-to-wire regression tests for Markdown and Go attachments with paths containing spaces.
- files: `internal/app/attachment_backend_integration_test.go`
- verify: ```
- proof: ### OpenCode immediate dispatch after `/backend opencode`
- ledgerId: led-1788322504772-eec9c8fd

### 2026-09-02 · Fix Claude send failure handling (@developer... — tekton-5 (developer) · `issues`
- summary: Changed Claude’s prompt-write failure path to return a wrapped, actionable error after emitting the existing failed-prompt bubble, so callers can retain and ret
- files: `internal/backend/claude.go`, `internal/backend/claude_send_test.go`, `internal/backend/claude_attachment_test.go`
- verify: ```
- proof: ### Failed attachment write → error and retryable queue item
- ledgerId: led-1788322037838-8452ebf5

### 2026-09-02 · Reproduce OpenCode attachment error (@general... — tekton-1 (developer) · `issues`
- summary: Verified the repository `HEAD` is `8a0f963c`, exactly tag `v0.3.16`.
- files: None modified., Read-only evidence reviewed:, `internal/backend/parts.go`, `internal/backend/opencode.go`, `internal/backend/parts_test.go`, `internal/app/model.go`, `internal/app/attach_queue_test.go`, `internal/app/plan_mode.go`, `internal/state/state.go`, `internal/panels/chat_attach.go`, `cmd/headless/main.go`
- verify: ```
- proof: ### Current OpenCode attachment wire
- ledgerId: led-1788321743872-43360ec8

### 2026-09-02 · Reproduce Claude attachment error (@general s... — tekton-2 (developer) · `issues`
- summary: Traced attachment routing from `panels.Chat` through `app.New`, `sendChatMode`, `sendChat`, and the Claude stream-json writer.
- files: None. Read-only investigation.
- verify: ```
- proof: ### Current Claude wire
- ledgerId: led-1788321712259-a6a41335

### 2026-09-02 · Audit installed release parity (@general suba... — tekton-3 (developer) · `issues`
- summary: Confirmed `HEAD` is `8a0f963c320ad8649d1c4865c96a5eb9d4c038e3`, exactly `v0.3.16`; `origin/main` and the tag point to the same commit.
- files: None. This was read-only; the release asset was downloaded only to a temporary audit directory outsi
- verify: ```
- proof: For an `.md`, source file, archive, or any non-image/PDF attachment, the current installed binary produces a text-only prompt part, not a me
- ledgerId: led-1788321688312-2e9569eb

### 2026-09-01 · Harden attachment validation (@developer suba... — tekton-6 (developer) · `issues`
- summary: Enforced content-validated upload eligibility: only sniffed PNG, JPEG, GIF, WebP, and `%PDF-` PDFs can produce OpenCode file parts.
- files: `internal/backend/parts.go`, `internal/backend/claude.go`, `internal/backend/parts_test.go`, `internal/backend/claude_attachment_test.go`
- verify: ```
- proof: ### Spoofed or mismatched content — one text/path-reference part, zero file parts
- ledgerId: led-1788278696126-92c03363

### 2026-09-01 · Add Windows installer docs (@developer subagent) — tekton-7 (developer) · `issues`
- summary: Added `install.ps1`, a Windows PowerShell installer that:
- files: `install.ps1`, `README.md`, `website/app/docs/getting-started/page.tsx`
- verify: ```
- proof: ### Windows installation instructions
- ledgerId: led-1788278470142-008c64bc

### 2026-09-01 · Audit Windows support (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Mapped the current release mechanism: a tag-triggered GitHub Actions workflow invokes GoReleaser, which currently builds only `darwin` and `linux` archives.
- files: ### Repository conventions inspected, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/CLAUDE.md`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.opencode/oikonomos.md`, ### Release pipeline and artifacts, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.github/workflows/release.yml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.goreleaser.yaml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/go.mod`, ### Installer and installation documentation, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/install.sh`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/README.md`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/app/docs/getting-started/page.tsx`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/app/get-started/page.tsx`
- verify: ```
- proof: ### Intended Windows install and release flow
- ledgerId: led-1788278367901-b193f8c1

### 2026-09-01 · Review attachment + theme fixes (@reviewer su... — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788277949403-fd4aa6ca

### 2026-09-01 · Fix theme cache and frame proof (@developer s... — tekton-5 (developer) · `done`
- summary: Added active theme identity (`chrome.CurrentTheme().Name`) to `frameDigest`, so chrome’s package-global palette changes invalidate cached frame bytes without co
- files: `internal/app/digest.go`, `internal/app/theme_frame_test.go`, `cmd/uishot/main.go`
- verify: ```
- proof: ### Cache identity and reuse
- ledgerId: led-1788273016116-0f6a6c2a

### 2026-09-01 · Unify panel background styling (@developer su... — tekton-4 (developer) · `issues`
- summary: Made `Panel*` and `OnPanel*` semantic text styles foreground-only; sidebar wrappers and `PanelBox` remain the sole continuous panel-background owners.
- files: `internal/chrome/styles.go`, `internal/chrome/styles_test.go`, `internal/panels/chat.go`, `internal/panels/threads_opencode.go`, `internal/panels/agents.go`, `internal/panels/board.go`, `internal/panels/question_modal.go`, `internal/panels/links.go`, `internal/panels/model_picker.go`, `internal/panels/session_picker.go`, `internal/panels/session_picker_search_test.go`, `internal/panels/theme_surface_test.go`
- verify: ```
- proof: *Paper — ANSI-stripped populated chat and expanded tool output**
- ledgerId: led-1788272544629-537b4e8d

### 2026-09-01 · Fix attachment transport policy (@developer s... — tekton-3 (developer) · `issues`
- summary: Added a shared attachment preparation path that accepts only readable regular files, resolves missing MIME types, converts paths to absolute paths, and safely q
- files: `internal/backend/parts.go`, `internal/backend/parts_test.go`, `internal/backend/claude.go`, `internal/backend/claude_attachment_test.go`
- verify: ```
- proof: *OpenCode mixed attachment payload** — text/source/archive become path references; PNG, PDF, and JPEG remain real file parts:
- ledgerId: led-1788272376527-e1ff52b3

### 2026-09-01 · Audit theme background clash (@general subagent) — tekton-2 (developer) · `issues`
- summary: Audited the in-flight `PanelBg` path and all requested sidebar surfaces without changing files.
- files: None. Read-only audit.
- verify: None. Read-only task; no test or mutation commands run.
- proof: ### Theme initialization and propagation
- ledgerId: led-1788271986249-b7c64a1b

### 2026-09-01 · Review current dirty regression (@general sub... — theboringcto-1 (cto) · `issues`
- summary: Performed a read-only regression review of attachment transport and `PanelBg` rendering seams.
- files: None modified., Relevant reviewed files:, `internal/app/model.go:139-196`, `internal/app/model.go:1363-1370`, `internal/app/digest.go:31-77`, `internal/backend/parts.go:23-52`, `internal/backend/opencode.go:437-450`, `internal/backend/claude.go:996-1096`, `internal/panels/chat_attach.go:249-335, 418-432`, `internal/chrome/styles.go:477-514`, `internal/panels/chat.go:609-612, 3045-3104`, `internal/app/model.go:4323-4370`
- verify: ```
- proof: ### Attachment ownership and regression proof
- ledgerId: led-1788271942905-0329b4d8

### 2026-09-01 · Map attachment send pipeline (@general subagent) — tekton-1 (developer) · `issues`
- summary: Traced the attachment path from picker/paste staging through `state.Attachment`, app dispatch, and both live backends.
- files: None. Read-only recon.
- verify: None. Read-only task; no test or mutation commands run.
- proof: ### Attachment model and MIME detection
- ledgerId: led-1788271926463-19c48cee

### 2026-08-31 · Dev W2: tool output expand UI (@developer sub... — tekton-4 (developer) · `issues`
- summary: All green. Assembling the return contract:
- files: `cmd/uishot/main.go`, `internal/panels/threads_opencode_test.go`, `internal/app/model.go`
- verify: ```
- proof: *1. Expanded row WITH output** (unit test log, verbatim — `TestToolRowExpandShowsOutput`, 44 cols, ANSI-stripped):
- ledgerId: led-1788153137253-e58a3047

### 2026-08-31 · Dev W1: tool output capture (@developer subag... — tekton-1 (developer) · `issues`
- summary: All verification complete. Composing the return.
- files: `internal/state/state.go`, `internal/backend/events.go`, `internal/backend/claude_events.go`, `internal/backend/tool_output_test.go` *(new)*, `internal/backend/claude_live_test.go`, NOT touched
- verify: ```
- proof: *claude done-event JSON** (unit test log, verbatim — text result `"total 42"`):
- ledgerId: led-1788152725848-9c68eb87

### 2026-08-31 · Dev: claude backend NewOffice+SwapPrimary (@d... — tekton-3 (developer) · `issues`
- summary: All gates green — 92 TestClaude tests pass (including the 3 new ones), zero FAILs. Composing the return.
- files: `internal/backend/claude.go`, `internal/backend/claude_office_swap_test.go`, Foreign dirty paths in `git status` (`internal/app/*`, `internal/panels/*`, `internal/backend/claude
- verify: (`git diff … | grep -c '^-[^-]'` → `0`: purely additive)
- proof: Stub-recorded claude argv (verbatim from `-v` run) — **NewOffice respawns FRESH, SwapPrimary respawns with `--resume <saved>`**:
- ledgerId: led-1788152562733-a7c99f48

### 2026-08-31 · Scout: opencode→claude backend parity gap (@e... — skopos-1 (scout) · `issues`
- summary: Recon complete. All evidence gathered; the tree was never touched.
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788151932135-a37dc962

### 2026-08-30 · Dev V: claude charter bridge (@developer suba... — tekton-1 (developer) · `issues`
- summary: All gates green. Composing the return.
- files: `internal/backend/charter_claude.go`, `internal/backend/claude.go`, `internal/backend/charter_claude_test.go`, `internal/backend/claude_live_test.go`
- verify: ```
- proof: *Generated CLAUDE.md, create case (exact bytes):**
- ledgerId: led-1788107692913-a438188d

### 2026-08-30 · Dev V2: backend priming docs (@developer suba... — tekton-2 (developer) · `issues`
- summary: Scope clean — exactly my two files. All gates green. Composing the return.
- files: `README.md`, `website/app/docs/backends/page.tsx`
- verify: $ `git status --porcelain` → ` M README.md` / ` M website/app/docs/backends/page.tsx` (scope-exact, no commits)
- proof: *README — the new paragraph as a GitHub reader sees it (rendered):**
- ledgerId: led-1788107061420-80830a89

### 2026-08-30 · Dev U: match sections + invert shots (@develo... — tekton-11 (developer) · `issues`
- summary: All gates green. Composing the return.
- files: `website/app/globals.css`, `website/components/home/hero.tsx`, `website/components/home/product-platform.tsx`, `website/components/home/agents-need-action.tsx`, `website/components/home/plan-then-build.tsx`, `website/components/home/feature-grid.tsx`, `website/app/docs/{page,backends,chat-and-threads,getting-started,keys-and-slash,layout-themes-power, `website/components/site-footer.tsx`, scratch only
- verify: $ CDP-driven real Chrome (`Google Chrome.app --headless=new`, served `out/`): html class confirmed `light`/`dark` per scenario; home 13411px
- proof: *(a) Light home, full page (segments 0–3):** every section light. Hero: off-white band, near-black headline, `coworkers.` in the light-theme
- ledgerId: led-1788101277659-a2b5ec62

### 2026-08-30 · Dev U: site light theme + toggle (@developer... — tekton-10 (developer) · `issues`
- summary: All gates green. Scope is exactly my files (the ledger modification is the office app's own recorder; the untracked blog post untouched). Composing the return.
- files: `website/app/globals.css`, `website/app/layout.tsx`, `website/components/theme-provider.tsx`, `website/components/theme-toggle.tsx`, `website/components/site-header.tsx`, `website/components/home/hero.tsx`, `website/components/home/context-model.tsx`
- verify: $ Contrast spot-check (computed oklch→sRGB): accent `rgb(31,93,194)` on light bg = **5.98:1** (AA); foreground **19.72:1**; muted-foreground
- proof: *Screenshot 1 — LIGHT default (fresh profile, no localStorage → script resolves light):** white/off-white page with faint gutter grid; light
- ledgerId: led-1788099926687-04ed2fd8

### 2026-08-30 · Blog: AGENTS.md best practices (@developer su... — tekton-9 (developer) · `done`
- summary: All gates green. Composing the return.
- files: `website/content/blog/agents-md-best-practices.md`
- verify: Bonus checks: `grep -o 'agents-md-best-practices' out/sitemap.xml` → `agents-md-best-practices`; `grep -c 'agents-md-best-practices' out/rss
- proof: *Frontmatter (verbatim):**
- ledgerId: led-1788099390242-ea217786

### 2026-08-30 · Dev T1: finish bypass core + UI (@developer s... — tekton-8 (developer) · `issues`
- summary: All 11 bypass tests pass, and the diff confirms the in-tree work spans exactly the scoped files. The earlier 5-FAIL full-suite run was parallel-load flake (this
- files: `internal/app/model.go`, `internal/app/browser_open.go`, `internal/app/digest.go`, `internal/app/bypass_test.go`, `cmd/uishot/main.go`
- verify: ```
- proof: *1. The confirm modal's exact copy** (leg A — `/bypass` typed, mode still OFF, amber QUESTION card centered over chat):
- ledgerId: led-1788085240989-7f3798ae

### 2026-08-30 · Dev T1: bypass core + UI (@developer subagent) — tekton-1 (developer) · `done`
- summary: Final verification runs, captured clean:
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788081642205-4ca0f989

### 2026-08-30 · Docs: disable strips bypass block (@developer... — tekton-7 (developer) · `done`
- summary: **README.md** — rewrote the tail of the `/bypass` paragraph: the stale "it stays on disk until you remove it" claim is now "disabling `/bypass` strips it automa
- files: `README.md`, `website/app/docs/keys-and-slash/page.tsx`, `website/app/docs/permissions-and-questions/page.tsx`
- verify: ```
- proof: *README.md — the `/bypass` paragraph a reader sees (verbatim, line 122):**
- ledgerId: led-1788074746754-c53f5f94

### 2026-08-30 · Recon in-flight bypass diff (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Scout recon complete. The tree is fully green on the final snapshot. Return per contract:
- files: | Path | Role |, |---|---|, | `internal/app/model.go`, | `internal/app/browser_open.go`, | `internal/app/digest.go`, | `internal/app/bypass_test.go` (new, 499 lines) | 11 app-layer tests, | `internal/backend/claude.go`, | `internal/backend/opencode.go`, | `internal/backend/charter.go`, | `internal/backend/bypass_permissions_test.go`, | `internal/backend/claude_live_test.go`, | `cmd/uishot/main.go` | `--bypass` demo-proof leg (appeared as modified DURING this scout
- verify: `go test ./internal/backend/ -count=1` (full) → one FAIL `TestClaudeSendMidTurnQueuedNoBlocking` under parallel-suite load; **isolated re-ru
- proof: | Layer | Status | Evidence |
- ledgerId: led-1788074510988-72aa04ea

### 2026-08-30 · Verify bypass docs+website (@developer subagent) — tekton-5 (developer) · `issues`
- summary: Everything verified. Composing the return.
- files: `README.md`, `website/app/docs/keys-and-slash/page.tsx`, `website/app/docs/permissions-and-questions/page.tsx`
- verify: (36/36 static pages generated — both edited docs pages compiled and prerendered, so the JSX/TSX in my edits is syntactically valid. README i
- proof: *README.md — the corrected `/bypass` paragraph a reader sees:**
- ledgerId: led-1788074473205-f40e45e1

### 2026-08-30 · Verify+fix backend/app bypass (@developer sub... — tekton-4 (developer) · `issues`
- summary: All gates green. Composing the return.
- files: `internal/app/bypass_test.go`, Read-only for verification: `internal/app/backend_switch_test.go`
- verify: (none of these are in my scope — gofmt is clean on every file I touched: `gofmt -l` on all 9 scoped files returns nothing)
- proof: The compile error before → after (vet, verbatim):
- ledgerId: led-1788074472169-89e3ef6b

### 2026-08-30 · Dev T2: backend bypass surfaces (@developer s... — tekton-2 (developer) · `issues`
- summary: Everything is green on the settled tree. Composing the return.
- files: `internal/backend/claude.go`, `internal/backend/opencode.go`, `internal/backend/charter.go`, `internal/backend/bypass_permissions_test.go`, `internal/backend/claude_live_test.go`, NOT touched: `internal/state/state.go` (interface lives here
- verify: ```
- proof: *The two claude argv shapes** (stub-recorded, byte-exact pinned in `TestClaudeSpawnArgvBypassPermissions` / `TestClaudeSpawnArgvWithoutBypas
- ledgerId: led-1788070903262-1fe015ae

### 2026-08-30 · Dev T3: bypass docs (@developer subagent) — tekton-3 (developer) · `done`
- summary: My three files are exactly the in-scope ones; the other dirty paths are pre-existing foreign work (I never touched them, no commits made).
- files: `README.md`, `website/app/docs/keys-and-slash/page.tsx`, `website/app/docs/permissions-and-questions/page.tsx`
- verify: ```
- proof: *README — the new row and paragraph, as rendered:**
- ledgerId: led-1788069094327-95b6629e

### 2026-08-28 · Dev S2: picker harmonization (@developer suba... — tekton-9 (developer) · `issues`
- summary: All gates green on the settled tree. Composing the return.
- files: `internal/panels/session_picker.go`, `internal/panels/chat_attach.go`, `internal/panels/chat_attach_test.go`, `internal/panels/session_picker_search_test.go`, `internal/panels/chat.go`, NOT touched: `chat_attach_ignore.go`
- verify: (21 packages `ok`, incl. `internal/backend 33.105s`, `internal/panels 13.710s`, `internal/app 79.979s`. A first full-suite run flaked `TestC
- proof: *1. /session card mid-filter (typed `a`) — ANSI-stripped render** (cursor row reversed; rows 2–3 re-inked):
- ledgerId: led-1787925672531-0028a687

