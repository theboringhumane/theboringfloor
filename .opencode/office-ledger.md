# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-03 · Pin approved plan value (@developer subagent) — tekton-8 (developer) · `done`
- summary: Captured a rune-capped approved-plan snapshot once at approval command creation.
- files: `internal/app/plan_mode.go`, `internal/app/model.go`, `internal/app/plan_mode_test.go`, `internal/app/plan_tools_test.go`
- verify: ```
- proof: *Worked oversized multibyte approval**
- ledgerId: led-1788435150629-b1136ea0

### 2026-09-03 · Final plan tools signoff (@reviewer subagent) — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788434756483-097a2700

### 2026-09-03 · Add install.ps1 checksum tests (@developer su... — tekton-7 (developer) · `issues`
- summary: Added a repository-root Go regression test for `install.ps1`’s `Get-Checksum` match expression.
- files: `install_ps1_test.go`
- verify: ```
- proof: | Input line | Expected match |
- ledgerId: led-1788434233704-a1ba9673

### 2026-09-03 · Fix install.ps1 checksum (@developer subagent) — tekton-6 (developer) · `issues`
- summary: Replaced `Get-Checksum`’s composite-format (`-f`) regex construction with string concatenation.
- files: `install.ps1`
- verify: ```
- proof: ```powershell
- ledgerId: led-1788433994550-eaa676c7

### 2026-09-03 · Scout install.ps1 checksum (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Confirmed the reported failure is caused by the format operator (`-f`) in `Get-Checksum`, not by parsing the downloaded checksum file.
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/install.ps1`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.goreleaser.yaml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.github/workflows/release.yml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/README.md`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/app/docs/getting-started/page.tsx`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/README.md`, No dedicated installer/checksum test file exists in the repository.
- verify: ```
- proof: ### Broken function — `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/install.ps1`
- ledgerId: led-1788433807440-4622233e

### 2026-09-03 · Cap approved plan persistence (@developer sub... — tekton-5 (developer) · `done`
- summary: Added a central rune-safe approved-plan cap of 20,000 runes with the exact visible suffix `… [approved plan truncated]`.
- files: `internal/app/plan_mode.go`, `internal/app/plan_tools.go`, `internal/app/sessions.go`, `internal/app/plan_mode_test.go`, `internal/app/plan_tools_test.go`
- verify: ```
- proof: *Worked oversized multibyte approval**
- ledgerId: led-1788433458614-21f2223a

### 2026-09-03 · Harden plan marker overlaps (@developer subag... — tekton-4 (developer) · `issues`
- summary: Made valid `plan-present` and `plan-update` blocks opaque during extraction: an own-line `⟦plan-get-approved⟧` inside either block is retained as plan body text
- files: `internal/plantools/plantools.go`, `internal/plantools/plantools_test.go`
- verify: ```
- proof: ### Valid present block containing an approval marker
- ledgerId: led-1788433367856-2182a00f

### 2026-09-03 · Review plan tools end-to-end (@reviewer subag... — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788432981454-b93fae9a

### 2026-09-03 · Build plan tool protocol (@developer subagent) — tekton-1 (developer) · `issues`
- summary: Added backend-neutral `internal/plantools` parsing and scrubbing for strict plan-present, plan-update, and own-line plan-get-approved directives.
- files: `internal/plantools/plantools.go`, `internal/plantools/plantools_test.go`, `internal/state/state.go`, `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/plan_tools_test.go`, `internal/backend/browser_open_test.go`, `internal/backend/claude_spawn_test.go`, `internal/backend/claude_office_swap_test.go`, `internal/backend/claude_attachment_test.go`
- verify: ```
- proof: ### Agent first-prompt harness preamble
- ledgerId: led-1788429772606-0c38e556

### 2026-09-03 · Implement plan tool app state (@developer sub... — tekton-2 (developer) · `issues`
- summary: Added a separate pane-keyed approved-plan store, independent from the existing draft `PlanText`.
- files: `internal/app/plan_mode.go`, `internal/app/plan_mode_test.go`, `internal/app/plan_tools.go`, `internal/app/plan_tools_test.go`, `internal/app/sessions.go`
- verify: ```
- proof: ### Agent-presented plan
- ledgerId: led-1788429541066-88774d32

### 2026-09-03 · Document plan tools (@developer subagent) — tekton-3 (developer) · `issues`
- summary: Documented the agent-only plan-tool protocol in the README with exact multiline `plan-present` / `plan-update` blocks and the own-line `plan-get-approved` marke
- files: `README.md`, `website/app/docs/plan-mode/page.tsx`, `website/app/docs/keys-and-slash/page.tsx`
- verify: ```
- proof: ### Agent plan tools
- ledgerId: led-1788429274819-4f3306c4

### 2026-09-02 · Review browser policy prompt (@reviewer subag... — theboringcto-3 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788359451842-23854787

### 2026-09-02 · Strengthen browser tool prompt (@developer su... — tekton-11 (developer) · `done`
- summary: Added an explicit first-turn browser policy before all directive syntax.
- files: `internal/browsertools/browsertools.go`, `internal/browsertools/browsertools_test.go`, `internal/backend/browser_open_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788359224714-1740b26e

### 2026-09-02 · Add browser policy to charter (@developer sub... — tekton-12 (developer) · `issues`
- summary: Added one canonical **Browser use (office default)** section to the embedded office charter.
- files: `internal/charter/charter.md`, `.opencode/oikonomos.md`, `internal/charter/charter_test.go`, `internal/backend/charter_test.go`, `internal/backend/charter_claude_test.go`
- verify: ```
- proof: ### Canonical charter section
- ledgerId: led-1788359196951-3c2bfb6e

### 2026-09-02 · Document built-in browser preference (@develo... — tekton-13 (developer) · `done`
- summary: Added a concise README reference directing agents to the office’s built-in browser directives first.
- files: `README.md`, `website/app/docs/browser-tab/page.tsx`
- verify: ```
- proof: ### Built-in first
- ledgerId: led-1788358967521-41f5ab2d

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

