# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-02 · Exclude recovery control noise (@developer su... — tekton-10 (developer) · `done`
- summary: Added exported recent-message control protocol constants and `chatcontext.IsControlText`.
- files: `internal/chatcontext/chatcontext.go`, `internal/chatcontext/chatcontext_test.go`, `internal/app/recent_messages.go`, `internal/app/recent_messages_test.go`
- verify: ```
- proof: ### Recovery transcript after a marker-only request
- ledgerId: led-1788358198859-eacde36d

### 2026-09-02 · Review recent context tool (@reviewer subagent) — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788358006305-3a6a8118

### 2026-09-02 · Implement recent-message context reply (@deve... — tekton-8 (developer) · `issues`
- summary: Added `EvRecentMessages` app handling that builds and asynchronously sends a bounded synthetic transcript follow-up through `currentBackend`, so a backend repla
- files: `internal/app/recent_messages.go`, `internal/app/recent_messages_test.go`, `internal/app/model.go`
- verify: ```
- proof: ### Synthetic follow-up delivered to the current backend
- ledgerId: led-1788357742342-f5e928ad

### 2026-09-02 · Build recent-message tool protocol (@develope... — tekton-7 (developer) · `done`
- summary: Added backend-neutral `internal/chatcontext` marker parsing, preamble, scrubbing, count clamping, and marker-only fallback behavior.
- files: `internal/chatcontext/chatcontext.go`, `internal/chatcontext/chatcontext_test.go`, `internal/state/state.go`, `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/recent_messages_test.go`, `internal/backend/browser_open_test.go`, `internal/backend/claude_attachment_test.go`, `internal/backend/claude_office_swap_test.go`, `internal/backend/claude_spawn_test.go`
- verify: ```
- proof: ### First-boss-prompt preamble appended after the browser preamble
- ledgerId: led-1788357466180-7e0bdb38

### 2026-09-02 · Make tabs clickable (@developer subagent) — tekton-5 (developer) · `done`
- summary: Added rendered-tab hit testing that shares the exact density, compact-label, separator, and narrow-width geometry used by the right-panel tab bar.
- files: `internal/panels/tabs.go`, `internal/panels/tabs_test.go`, `internal/app/selection.go`, `internal/app/tab_mouse_test.go`
- verify: ```
- proof: | Click target in the rendered right-panel tab row | Resulting active tab |
- ledgerId: led-1788357422937-adffc8d5

### 2026-09-02 · Document recent-message tool (@developer suba... — tekton-9 (developer) · `done`
- summary: Added a concise README reference and recovery behavior paragraph for the agent-only recent-message marker.
- files: `README.md`, `website/app/docs/chat-and-threads/page.tsx`, `website/app/docs/keys-and-slash/page.tsx`
- verify: ```
- proof: ### README — Keys reference
- ledgerId: led-1788357314604-9760bcd4

### 2026-09-02 · Improve terminal focus (@developer subagent) — tekton-6 (developer) · `issues`
- summary: Made a left click inside the terminal viewport capture terminal keyboard focus, including blank body cells.
- files: `internal/panels/terminal.go`, `internal/panels/terminal_sel_test.go`, `internal/app/terminal.go`, `internal/app/model.go`, `internal/app/term_mouse_test.go`, `internal/app/grab_test.go`
- verify: ```
- proof: ### Terminal focus interaction
- ledgerId: led-1788357185580-50cd0e3c

### 2026-09-02 · Review post-bypass sends (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788357123804-6121aabd

### 2026-09-02 · Map click focus UX (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Mapped the existing right-panel tab rendering and selection seams.
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/panels/tabs.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/panels/tabs_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/model.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/selection.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/terminal.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/term_mouse_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/grab_test.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/panels/terminal.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/panels/terminal_sel_test.go`
- verify: Targeted read/search only; no tests or mutations were run.
- proof: ### Current interaction flow
- ledgerId: led-1788356875568-12c362d5

### 2026-09-02 · Trace bypass message black hole (@developer s... — tekton-1 (developer) · `done`
- summary: Changed unavailable current-backend sends from a silent success to `active backend unavailable`, which routes through the existing visible `sendErrMsg` path ins
- files: `internal/app/model.go`, `internal/app/bypass_test.go`, `internal/app/current_backend_test.go`
- verify: ```
- proof: ### Successful bypass ON — ordinary Enter routing
- ledgerId: led-1788356808316-879f94db

### 2026-09-02 · Live OpenCode post-bypass send (@developer su... — tekton-2 (developer) · `issues`
- summary: Added a hermetic, spawned-serve OpenCode integration test for the bypass replacement path.
- files: `internal/backend/opencode_bypass_integration_test.go`
- verify: ```
- proof: ### Hermetic bypass replacement round trip
- ledgerId: led-1788356498572-80e0515c

### 2026-09-02 · Live Claude post-bypass send (@developer suba... — tekton-3 (developer) · `issues`
- summary: Added deterministic Claude bypass-candidate integration coverage for the `/bypass` handoff shape:
- files: `internal/backend/claude_bypass_candidate_integration_test.go`
- verify: ```
- proof: ### Deterministic candidate handoff
- ledgerId: led-1788356473861-e8b025c9

### 2026-09-02 · Refresh changelog v0.3.20 (@developer subagent) — tekton-4 (developer) · `done`
- summary: Updated changelog release fetching so each static-export build gets a distinct GitHub API fetch URL, preventing a prior build cache from reusing a stale release
- files: `website/lib/changelog.ts`
- verify: ```
- proof: *Live:** https://office.theboring.name/changelog/
- ledgerId: led-1788356383888-74b5c70a

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

