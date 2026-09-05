# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-05 · Fix uninstall regression (@developer subagent) — tekton-13 (developer) · `done`
- summary: Added a bounded (5-second), stderr-silent `--version` ownership probe for legacy installations without a manifest entry.
- files: `install.sh`
- verify: ```
- proof: ### 1. Manifest present and matching
- ledgerId: led-1788622496263-5095481c

### 2026-09-05 · Bound control admission (@developer subagent) — tekton-11 (developer) · `issues`
- summary: Added immediate buffered-channel admission control for read projection requests, defaulting to 16 in-flight requests and configurable through `Options.MaxInFlig
- files: `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`
- verify: ```
- proof: ```http
- ledgerId: led-1788622365860-df6c5536

### 2026-09-05 · Harden stale discovery (@developer subagent) — tekton-10 (developer) · `issues`
- summary: Added `bootId` to control discovery records, automatically generated with cryptographic randomness when absent before atomic persistence.
- files: `internal/control/control.go`, `internal/control/control_test.go`, `cmd/thefloor_mcp/office.go`, `cmd/thefloor_mcp/mcp_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788622341438-38ea2e3e

### 2026-09-05 · Fix uninstall ownership (@developer subagent) — tekton-12 (developer) · `issues`
- summary: Added an installer-owned SHA-256 manifest at `${PREFIX}/.theboringfloor-manifest` after binary installation.
- files: `install.sh`
- verify: ```
- proof: ```text
- ledgerId: led-1788622331259-c74341f3

### 2026-09-05 · Fix config merge safety (@developer subagent) — tekton-9 (developer) · `done`
- summary: Switched OpenCode config decoding to `json.Decoder.UseNumber()` so existing JSON number literals—including integers above `2^53`—are retained losslessly when th
- files: `internal/mcpinstall/mcpinstall.go`, `internal/mcpinstall/mcpinstall_test.go`
- verify: ```
- proof: *Before registration**
- ledgerId: led-1788622266449-d3ab97aa

### 2026-09-05 · Blast radius review (@reviewer subagent) — theboringcto-3 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: Return to developer: (1) preserve JSON numbers with `json.Decoder.UseNumber()` or use a lossless JSON edit strategy, and add a regression ca
- proof: (none)
- ledgerId: led-1788622033491-0011c5f3

### 2026-09-05 · Security review control API (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: Return to developer: redesign discovery/auth so an arbitrary same-UID process cannot recover a reusable bearer capability from a predictable
- proof: (none)
- ledgerId: led-1788621988061-6b87074a

### 2026-09-05 · Concurrency review (@reviewer subagent) — theboringcto-2 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788621980727-e8109dc3

### 2026-09-05 · thefloor_mcp binary (@developer subagent) — tekton-5 (developer) · `issues`
- summary: Added the dependency-free `thefloor_mcp` stdio MCP server under `cmd/thefloor_mcp`.
- files: `cmd/thefloor_mcp/main.go`, `cmd/thefloor_mcp/mcp.go`, `cmd/thefloor_mcp/office.go`, `cmd/thefloor_mcp/mcp_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788621571738-3cad5ffc

### 2026-09-05 · Document thefloor_mcp (@developer subagent) — tekton-8 (developer) · `done`
- summary: Added a README section for `thefloor_mcp`: release packaging, automatic registration, all six tools, live/offline behavior, transcript limits, environment varia
- files: `README.md`, `website/app/docs/mcp-server/page.tsx`, `website/app/docs/page.tsx`, `website/app/docs/plan-mode/page.tsx`
- verify: ```
- proof: ### README — MCP server and office control
- ledgerId: led-1788621533250-c2ca5aa9

### 2026-09-05 · MCP config auto-install (@developer subagent) — tekton-6 (developer) · `done`
- summary: Added `internal/mcpinstall` with a non-fatal, idempotent `Ensure(binPath)` registration pass for OpenCode and Claude Code.
- files: `internal/mcpinstall/mcpinstall.go`, `internal/mcpinstall/mcpinstall_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788621516211-0a3d7a1f

### 2026-09-05 · App control wiring (@developer subagent) — tekton-3 (developer) · `issues`
- summary: Added the read-only `Model.applyControl` projection handler for plan, transcript, status, and unknown control queries.
- files: `internal/app/control.go`, `internal/app/control_test.go`, `cmd/theboringoffice/main.go`
- verify: ```
- proof: Sample Model state for plan projection:
- ledgerId: led-1788621495460-434ae9cb

### 2026-09-05 · Release plumbing second binary (@developer su... — tekton-7 (developer) · `issues`
- summary: Added the `thefloor_mcp` GoReleaser build for the same macOS/Linux/Windows amd64+arm64 matrix and matching version metadata ldflags.
- files: `.goreleaser.yaml`, `install.sh`, `install.ps1`
- verify: ```
- proof: ```yaml
- ledgerId: led-1788621491302-edce56a9

### 2026-09-05 · Control HTTP server (@developer subagent) — tekton-2 (developer) · `issues`
- summary: Added `internal/controlsrv`, a loopback-only, bearer-token-protected HTTP control server that bridges requests exclusively through `state.Event` sink emissions.
- files: `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`
- verify: ```
- proof: ```http
- ledgerId: led-1788621478620-ed046d20

### 2026-09-05 · Session search package (@developer subagent) — tekton-4 (developer) · `issues`
- summary: Added the read-only, current-project-scoped `internal/sessionsearch` package.
- files: `internal/sessionsearch/sessionsearch.go`, `internal/sessionsearch/sessionsearch_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788621473137-0574530a

### 2026-09-05 · Control protocol package (@developer subagent) — tekton-1 (developer) · `done`
- summary: Added dependency-light `internal/control` shared contract package for the office control server and `thefloor_mcp` client.
- files: `internal/control/control.go`, `internal/control/control_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788621402385-f67cad24

### 2026-09-05 · Scout MCP config injection (@general subagent) — skopos-3 (scout) · `issues`
- summary: **Existing injection precedent**
- files: `internal/backend/charter.go`, `internal/backend/charter_claude.go`, `internal/backend/charter_mcp.go`, `internal/backend/opencode.go`, `.opencode/mcp-servers.md`, `~/.config/opencode/opencode.json`, `~/.config/opencode/opencode.jsonc`, `~/.claude.json`, `<repo>/.mcp.json`
- verify: ```
- proof: ```json
- ledgerId: led-1788619865338-5f58033c

### 2026-09-05 · Scout session search corpus (@general subagent) — skopos-2 (scout) · `issues`
- summary: `session.json` is the per-working-directory office snapshot defined by `internal/app/sessions.go:82-112`. Its canonical write location is `~/.theboringfloor/pro
- files: `internal/app/sessions.go`, `internal/state/state.go`, `internal/app/model.go`, `internal/app/session_picker.go`, `internal/panels/session_picker.go`, `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/claude_events.go`, `internal/backend/backend.go`, Corpus measured: `~/.theboringfloor/projects/**/session.json`., Temporary-file corpus measured: `~/.theboringfloor/projects/**/.session-*.tmp`., Read fallback roots confirmed in `LoadSession`: `~/.theboringoffice/projects`, `~/.theboringfloor/se
- verify: ```
- proof: `searchable?` below means that the field contains text or stable metadata which exists in the snapshot; it does **not** indicate an existing
- ledgerId: led-1788619861391-19e1dcd4

### 2026-09-05 · Scout build and MCP overlap (@general subagent) — skopos-4 (scout) · `issues`
- summary: `cmd/` contains **nine** Go command packages. Only `cmd/theboringoffice` is shipped by the current GoReleaser configuration, as the executable `theboringfloor`;
- files: `.goreleaser.yaml`, `.github/workflows/release.yml`, `go.mod`, `install.sh`, `install.ps1`, `cmd/README.md`, `cmd/theboringoffice/main.go`, `cmd/headless/main.go`, `cmd/uishot/main.go`, `cmd/floorshot/main.go`, `cmd/termshot/main.go`, `cmd/soundtest/main.go`
- verify: ```
- proof: | capability | already covered by | verdict: duplicate/partial/new |
- ledgerId: led-1788619841031-c5bd4f28

### 2026-09-05 · Scout office IPC seam (@general subagent) — skopos-1 (scout) · `issues`
- summary: **1. Ambient local HTTP server:** No. The running `theboringoffice` binary does not start an HTTP listener, bind a port, or register HTTP handlers. Repository-w
- files: `cmd/theboringoffice/main.go`, `internal/app/ambient.go`, `internal/app/browser.go`, `internal/app/browser_open.go`, `internal/app/model.go`, `internal/app/open_url.go`, `internal/app/plan_mode.go`, `internal/app/plan_tools.go`, `internal/app/sessions.go`, `internal/backend/agentmemory.go`, `internal/backend/claude.go`, `internal/backend/opencode.go`
- verify: read-only inspection; no commands run
- proof: No ambient HTTP server exists.
- ledgerId: led-1788619817699-0983ee85

### 2026-09-03 · Write BTW hide/resume tests (@developer subag... — tekton-3 (developer) · `done`
- summary: Added regression coverage for BTW hide, resume, permanent exit, `/new` cleanup, invalid-state notices, and the complete `/btw → Esc → /btw → /done` lifecycle.
- files: `internal/app/btw_hide_test.go`
- verify: ```
- proof: `TestBtwHidePreservesSessionAndPinsMainChat`
- ledgerId: led-1788454375559-c1810436

### 2026-09-03 · Review BTW hide/resume diff (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788454280607-280b2bb9

### 2026-09-03 · Panel: BTW pin bubble render (@developer suba... — tekton-2 (developer) · `issues`
- summary: Added block-local `btwPin` hit maps and transcript-level `btwPinRows` aggregation.
- files: `internal/panels/chat.go`, `internal/panels/chat_window.go`
- verify: ```
- proof: Rendered transcript bubble:
- ledgerId: led-1788453886770-e624a327

### 2026-09-03 · App state: BTW hide/resume (@developer subagent) — tekton-1 (developer) · `issues`
- summary: Added hidden BTW-session state (`btwHiddenSnap`) and pinned-bubble ID tracking (`btwPinMsgID`) to `Model`.
- files: `internal/app/model.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788453828341-4cc750ba

### 2026-09-03 · Scout /btw UI code (@explore subagent) — skopos-1 (scout) · `issues`
- summary: ## `/btw` recon report
- files: (none)
- verify: Read-only reconnaissance only. No commands or files were modified.
- proof: ```text
- ledgerId: led-1788453452610-e3c67fb7

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

