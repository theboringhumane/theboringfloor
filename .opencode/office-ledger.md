# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-07 · Fail-closed release signing (@developer subag... — tekton-9 (developer) · `done`
- summary: Replaced the unsafe debug-signing fallback with release-task-graph validation that fails closed when `key.properties` is missing, required properties are blank,
- files: `mobile/android/app/build.gradle.kts`, `mobile/android/.gitignore`
- verify: ```
- proof: ```kotlin
- ledgerId: led-1788757244310-bbf30ac6

### 2026-09-07 · Reject empty bearer tokens (@developer subagent) — tekton-10 (developer) · `done`
- summary: Rejected empty and whitespace-only office control tokens during `controlsrv.New`, with the same panic-at-construction pattern used for a nil sink.
- files: `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`, `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: ### Office control server authentication
- ledgerId: led-1788757173123-20cbbbb8

### 2026-09-07 · Sign Android release APK (@developer subagent) — tekton-7 (developer) · `issues`
- summary: Created a self-signed RSA-2048 JKS release keystore outside the repository at `~/.theboringfloor/android/`, with the required alias, DN, validity, and `0600` pe
- files: `mobile/android/app/build.gradle.kts`, `mobile/android/key.properties`, `mobile/android/key.properties.example`, `/Users/theboringhumane/.theboringfloor/android/theboringfloor-release.jks`
- verify: ```
- proof: *Release keystore password — save this securely:**
- ledgerId: led-1788756970695-39f78cc0

### 2026-09-07 · Review control plane diff (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788756828415-bfb1c8c7

### 2026-09-07 · Document APK install and signing (@developer... — tekton-8 (developer) · `issues`
- summary: Added additive **Install the Android app** and **Signing key** sections immediately before the existing v1 limitations section.
- files: `website/app/docs/control-plane/page.tsx`
- verify: ```
- proof: ### Install the Android app
- ledgerId: led-1788756786680-908f537e

### 2026-09-07 · Flutter control plane app (@developer subagent) — tekton-5 (developer) · `issues`
- summary: Installed Flutter **3.47.2** with `brew install --cask flutter`.
- files: `mobile/.gitignore`, `mobile/pubspec.yaml`, `mobile/pubspec.lock`, `mobile/analysis_options.yaml`, `mobile/.metadata`, `mobile/README.md`, `mobile/lib/main.dart`, `mobile/lib/api/models.dart`, `mobile/lib/api/gateway_client.dart`, `mobile/lib/settings_store.dart`, `mobile/lib/time.dart`, `mobile/test/models_test.dart`
- verify: ```
- proof: ### Projects screen
- ledgerId: led-1788756143465-8e85f4f5

### 2026-09-07 · Build floorgate API gateway (@developer subag... — tekton-4 (developer) · `done`
- summary: Added `floorgate`, a stateless authenticated HTTP gateway binary with a stable bind address and persistent gateway bearer token.
- files: `cmd/floorgate/main.go`, `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: | METHOD PATH | auth | request | response | error codes |
- ledgerId: led-1788755508575-bd32973f

### 2026-09-07 · Project enumeration package (@developer subag... — tekton-3 (developer) · `done`
- summary: Added `internal/projects`, a typed project-discovery package for persisted theboringfloor projects.
- files: `internal/projects/projects.go`, `internal/projects/projects_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788755335593-10c3bf8f

### 2026-09-07 · Control plane docs and runbook (@developer su... — tekton-6 (developer) · `done`
- summary: Added `/docs/control-plane`, documenting the `floorgate` gateway, bearer-token setup, Tailscale path, full gateway API, Android client, and v1 limitations.
- files: `website/app/docs/control-plane/page.tsx`, `website/app/sitemap.ts`, `website/app/llms.txt/route.ts`, `website/app/docs/page.tsx`
- verify: `bun run build` was used because `website/package.json` declares `bun@1.3.5` as its package manager and `bun.lock` is present.
- proof: ### Remote control plane
- ledgerId: led-1788755269201-ded00dad

### 2026-09-07 · Office control app reducers (@developer subag... — tekton-2 (developer) · `done`
- summary: Added UI-goroutine reducers for remote send, stop, and new-session events.
- files: `internal/app/control_mutations.go`, `internal/app/control_mutations_test.go`, `internal/app/control.go`, `internal/app/model.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788755216423-13432871

### 2026-09-07 · Office control wire and routes (@developer su... — tekton-1 (developer) · `issues`
- summary: Added control-wire event kinds `EvControlSend`, `EvControlStop`, and `EvControlNew`, plus `Event.ControlText`.
- files: `internal/state/state.go`, `internal/control/control.go`, `internal/control/control_test.go`, `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`
- verify: ```
- proof: | METHOD PATH | request JSON | response JSON | emitted `state.Event` |
- ledgerId: led-1788755072077-2fee6364

### 2026-09-07 · Scout session control seams (@explore subagent) — skopos-3 (scout) · `issues`
- summary: I now have complete evidence for all 10 requirements. Producing the final report.
- files: `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`, `internal/control/control.go`, `internal/state/state.go` (lines 420-822), `internal/app/control.go`, `internal/app/model.go` (targeted greps + reads: 128-372, 890-990, 1020-1180, 1760-1830, 2030-2170, , `internal/app/sessions.go` (lines 340-573), `internal/app/plan_tools.go`, `internal/app/plan_tools_test.go`, `internal/panels/chat.go` (lines 300-320, 1010-1110, 1850-1910), `internal/backend/backend.go`, `internal/backend/opencode.go` (lines 440-620, 1575-1605, 1925-1975), `internal/backend/claude.go` (lines 200-330, 1040-1090, 1630-1675), `cmd/theboringfloor/main.go` (lines 150-299)
- verify: `TestControlPlanProjectionAndNilPlan` (line 12) — calls `m.applyControl(state.Event{Kind: state.EvControlQuery, ...})` directly and asserts
- proof: | Action | Narrowest safe seam (func signature + file:line) | Thread-safety mechanism to use | Exists today? |
- ledgerId: led-1788754825222-823459d0

### 2026-09-07 · Scout on-disk session storage (@explore subag... — skopos-2 (scout) · `issues`
- summary: # Read-Only Reconnaissance Report: Session/Project/Transcript Persistence
- files: | Path | Why inspected |, |---|---|, | `internal/config/config.go` | `Path()`, `Env`/`HomeOverride`, `THEFLOOR_HOME` override, `save()` a, | `internal/config/migrate.go` | Confirms legacy-home migration is a documented no-op today |, | `internal/brand/brand.go` | `DotDir = ".theboringfloor"`, `CLI = "theboringfloor"` |, | `internal/state/state.go` | `ChatMsg`, `Employee`, `BoardTask`, `MailItem`, `Event`/`EventKind` wi, | `internal/app/sessions.go` | `SessionFile`, `SessionPath`, `SessionDirHash`, `LoadSession`, `SaveS, | `internal/sessionsearch/sessionsearch.go` | Read-only session reader; root-resolution; confirms si, | `internal/control/control.go` | `Discovery` struct, `ControlPath`, `WriteDiscovery`/`ReadDiscovery, | `internal/controlsrv/server.go` | Loopback HTTP control server: `/v1/health`, `/v1/transcript`, `/, | `cmd/theboringfloor/main.go`, | `internal/backend/claude.go`
- verify: ```
- proof: ```
- ledgerId: led-1788754790329-7bf0655b

### 2026-09-07 · Scout control server surface (@explore subagent) — skopos-1 (scout) · `issues`
- summary: All evidence gathered. Producing the final report.
- files: `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`, `internal/control/control.go`, `internal/control/control_test.go`, `internal/app/control.go`, `internal/app/control_test.go`, `internal/app/plan_tools.go`, `internal/app/model.go` (lines 3335-3349), `internal/state/state.go` (lines 520-663), `cmd/thefloor_mcp/office.go`, `cmd/thefloor_mcp/mcp.go`, `cmd/theboringfloor/main.go` (lines 190-269)
- verify: `TestControlHelperProcess` — not a real assertion-bearing test; it is the subprocess entry point `startOfficeNamedProcess` `exec.Command`s i
- proof: ## Current control API contract
- ledgerId: led-1788754665496-8c68e4fb

### 2026-09-07 · Scout mobile toolchain and net (@explore suba... — skopos-4 (scout) · `issues`
- summary: All probes complete. Here is the recon report.
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.gitignore`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.goreleaser.yaml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/.github/workflows/release.yml`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/.gitignore`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/website/` (dir listing), `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/go.mod` (grep only, line 1 `go` directive), Grep across repo for `website` references
- verify: *30Gi free** on a 460Gi volume (93% used). The Flutter SDK itself is small (~1-2 GB including Dart), and the Android SDK is **already presen
- proof: | Tool | Installed? | Version | Ready to build an APK? | Blocking gap |
- ledgerId: led-1788754652283-cf0652e1

### 2026-09-06 · Fix btw swap failure paths (@developer subagent) — tekton-12 (developer) · `done`
- summary: Made `/btw` startup transactional by removing the pre-spawn `ResetPrimary(true)` call. `NewOffice()` now runs while the original primary remains seated; if spaw
- files: `internal/app/model.go`, `internal/app/btw_guard_test.go`, `internal/app/btw_swap_failure_test.go`
- verify: ```
- proof: *`/btw` startup failure path — before:**
- ledgerId: led-1788723543051-05b14dcc

### 2026-09-06 · Fix gofmt in cmd packages (@developer subagent) — tekton-14 (developer) · `done`
- summary: Applied `gofmt -w` to the three assigned pre-existing formatting-drift files only.
- files: `cmd/kittyprobe/main.go`, `cmd/thefloor_mcp/mcp.go`, `cmd/thefloor_mcp/office.go`
- verify: ```
- proof: ```diff
- ledgerId: led-1788723411448-5eb631af

### 2026-09-06 · Make config save atomic (@developer subagent) — tekton-13 (developer) · `done`
- summary: Replaced `brain.json`’s direct `os.WriteFile` with same-directory temp-file writing followed by `os.Rename`.
- files: `internal/config/config.go`, `internal/config/config_test.go`
- verify: ```
- proof: *Before**
- ledgerId: led-1788723407771-3017fa06

### 2026-09-06 · Verify backend changes pre-release (@develope... — tekton-10 (developer) · `done`
- summary: Fixed `/submodel` backend plumbing: `liveBackend.ApplyAgentModels` now writes the OpenCode agent configuration, and `Start` applies persisted `Config.AgentModel
- files: `internal/backend/opencode.go`, `internal/backend/charter_agentmodel.go`, `internal/backend/charter_agentmodel_test.go`
- verify: ```
- proof: `charter_agentmodel.go` now does plumb the override into the OpenCode agent harness: `ApplyAgentModels` writes `.opencode/opencode.json`, an
- ledgerId: led-1788723224478-a559a2e4

### 2026-09-06 · Review release diff blast radius (@reviewer s... — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788723048616-a041d146

### 2026-09-06 · Pre-release hygiene and secret scan (@develop... — tekton-11 (developer) · `issues`
- summary: Added a targeted ignore rule for the machine-local `.claude/settings.local.json`.
- files: `.gitignore`
- verify: Literal `sk-` false-positive locations were limited to normal code/docs/dependency content: `.opencode/opencode-video-analysis.md:46`; `cmd/
- proof: ```gitignore
- ledgerId: led-1788723001966-d818b2d3

### 2026-09-06 · Scout release mechanics (@explore subagent) — skopos-5 (scout) · `issues`
- summary: # Read-Only Reconnaissance: Release Cut Process for v0.3.31
- files: | Path | Why inspected |, |---|---|, | `.goreleaser.yaml` | Build IDs, binaries, archive naming, ldflags, | `internal/version/version.go` | Confirm version is stamped, not hardcoded, | `.github/workflows/release.yml` | Only CI workflow; confirms tag-push → goreleaser, | `.gitignore` | Check for `.claude`/`.opencode` ignore rules, | `website/app/changelog/page.tsx` | Changelog surface, | `website/lib/changelog.ts` | Confirms GitHub API fetch, no hardcoded version, | `install.sh` | Grep for hardcoded version strings, | `install.ps1` | Grep for hardcoded version strings, | (git only, no files) | `git config`, `git status -sb`, `git log`, `git tag`, `git show`, `git rev-
- verify: Convention: **lowercase conventional-commit-style prefixes** — `feat:`, `feat(scope):`, `fix:`, `refactor:`, `site:` (website-only changes)
- proof: ```bash
- ledgerId: led-1788722801372-6cbcab36

### 2026-09-06 · Close out submodel + btw work (@developer sub... — tekton-9 (developer) · `issues`
- summary: Formatted the pre-existing `internal/panels/chat_attach_ignore_test.go` source so the required repository-wide `gofmt -l` check is clean.
- files: `internal/panels/chat_attach_ignore_test.go`, `internal/app/btw_orphan_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788722521120-dbaa8991

### 2026-09-06 · Add submodel slash command (@developer subagent) — tekton-8 (developer) · `issues`
- summary: Added `applySubmodel(fields []string) tea.Cmd`, validating agent names and model references, persisting valid overrides to `brain.json`, and emitting the requir
- files: `internal/app/submodel.go`, `internal/app/submodel_test.go`, `internal/panels/popover.go`, `internal/panels/popover_cmds_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788719627750-61aa4350

### 2026-09-06 · Fix btw session persistence (@developer subag... — tekton-4 (developer) · `issues`
- summary: `hydrateSession` now removes persisted `Meta == "btw-pin"` rows, with a comment explaining that their hidden side-session snapshots are memory-only and cannot b
- files: `internal/app/sessions.go`, `internal/app/sessions_test.go`, `internal/app/btw_persist_test.go`
- verify: ```
- proof: *`session.json` persisted while inside an active `/btw` side session:**
- ledgerId: led-1788717123453-3ff0f73f

### 2026-09-06 · Add agent model config storage (@developer su... — tekton-6 (developer) · `done`
- summary: Added `Config.AgentModels map[string]ModelRef` with the `json:"agentModels"` tag and documentation distinguishing OPENCODE agent names from theboringfloor roste
- files: `internal/config/config.go`, `internal/config/config_test.go`
- verify: ```
- proof: ```json
- ledgerId: led-1788716975353-15ba95ae

### 2026-09-06 · Scout swap routing and agent model (@explore... — skopos-4 (scout) · `issues`
- summary: # READ-ONLY RECONNAISSANCE — /btw safety + per-sub-agent model config
- files: | Path | Why |, |---|---|, | `internal/app/sessions.go:337-361` | `officeSpawnBackend`/`btwSwapBackend`/`primarySeamBackend` in, | `internal/app/model.go:5585-5592,6000-6099` | `hasPendingBoss` guard + the actual `/btw` start, | `internal/app/btw_busy_test.go` | Existing characterization tests documenting the guard as "CURREN, | `internal/backend/opencode.go:1474-1574,2251-2430,2627-2679,2881-2947` | `SwapPrimary`/`NewOffice`, | `internal/backend/events.go:205-234,1463-1612` | `mapOCEvent`'s primaryID-gated classification, th, | `internal/backend/claude.go:200-330,920-989` | Claude `NewOffice`/`SwapPrimary`, `teardownProc`, | `internal/config/config.go:67-91,165-204` | `RoleConfig.Model` field + `Roles` default map, | `internal/backend/charter.go:165-212` | `mergeInstruction`'s "never clobber a hand-shaped config" , | `~/.config/opencode/opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `~/.config/openco, | `https://opencode.ai/config.json` | Live first-party JSON Schema fetch |
- verify: ```
- proof: (none)
- ledgerId: led-1788716642557-db88cd51

### 2026-09-06 · Always-visible btw return bubble (@developer... — tekton-1 (developer) · `done`
- summary: Added a fixed hidden-BTW footer row below the scrollable transcript whenever a `btw-pin` message exists and the panel has enough room.
- files: `internal/panels/chat.go`, `internal/panels/chat_window.go`, `internal/panels/btw_pin_sticky_test.go`
- verify: ```
- proof: *Hidden BTW session, even after scrolling transcript history to the top:**
- ledgerId: led-1788716618819-849f315a

### 2026-09-06 · Characterize btw busy and orphan bugs (@devel... — tekton-2 (developer) · `issues`
- summary: Added two hermetic characterization tests for `/btw` refusal when any boss chat message is pending, including a stale turn after the wedge watchdog fires.
- files: `internal/app/btw_busy_test.go`, `internal/app/btw_orphan_test.go`
- verify: ```
- proof: *`TestBtwStartRefusedWhileBossPending`** — Locks in that a pending boss bubble prevents `/btw` and appends the mid-turn error; once concurre
- ledgerId: led-1788716616609-cbb09535

### 2026-09-06 · Scout BTW session lifecycle (@explore subagent) — skopos-1 (scout) · `issues`
- summary: I now have complete evidence for all 7 requirements. Let me finalize the report.
- files: | Path | Why inspected |, |---|---|, | `internal/app/model.go` | Owns `btwSaved`/`btwHiddenSnap`/`btwPinMsgID` fields, `/btw`/`/done` sla, | `internal/app/sessions.go` | `btwSwapBackend`/`officeSpawnBackend`/`primarySeamBackend` interfaces, | `internal/app/btw_hide_test.go` | The only app-level btw lifecycle tests, | `internal/panels/chat.go` | `btwPinRows` hit-map, `BtwPinRowAt`, `ClickRow`, rendering of the `btw, | `internal/panels/chat_window.go` | `hitSpans.btwPin` field backing the hit-map |, | `internal/panels/popover.go` | `/btw`/`/done` help-popover copy only |, | `internal/panels/popover_cmds_test.go` | Confirms `/btw`/`/done` are registered as real slash comm, | `internal/backend/opencode.go` | `liveBackend.NewOffice`/`SwapPrimary`/`PrimaryID`/`ResetPrimary`,, | `internal/backend/claude.go` | `liveClaudeBackend.NewOffice`/`SwapPrimary`, `EvChatBoss` Pending e, | `internal/backend/claude_office_swap_test.go` | Backend-level `/btw`/`/done`/`/new` coverage on th
- verify: ```
- proof: (none)
- ledgerId: led-1788716230839-b493c711

### 2026-09-06 · Scout model picker slash command (@explore su... — skopos-3 (scout) · `issues`
- summary: Read-only recon complete: traced the slash-command registry/dispatch, the full `/model` (boss model) flow end-to-end, the model listing source, how sub-agent mo
- files: `internal/panels/popover.go`, `internal/app/model.go`, `internal/app/model_picker.go`, `internal/panels/model_picker.go`, `internal/backend/models_live.go`, `internal/backend/demo.go`, `internal/backend/opencode.go` (~1660–1810), `internal/backend/claude.go`, `internal/backend/events.go` (~214–226), `internal/backend/claude_events.go` (~435–500), `internal/config/config.go`, `internal/backend/charter.go`
- verify: Clean build, no errors, no output (after dependency download on first run).
- proof: (none)
- ledgerId: led-1788716149469-dd31b95f

### 2026-09-06 · Scout transcript clickable bubbles (@explore... — skopos-2 (scout) · `issues`
- summary: Read-only reconnaissance of the transcript rendering + mouse-click pipeline in the Go TUI. **No files edited.** Found that a directly-analogous feature — a clic
- files: `internal/panels/chat.go`, `internal/panels/chat_window.go`, `internal/panels/chat_selection.go`, `internal/panels/perm_modal.go`, `internal/panels/question_modal.go`, `internal/panels/links.go`, `internal/app/model.go`, `internal/app/selection.go`, `internal/app/btw_hide_test.go`, --, # FINDINGS, ## 1. Where the transcript is rendered, The transcript is built by a **per-block render cache + windowed viewport**, not a single monolithic
- verify: ```
- proof: (none)
- ledgerId: led-1788716148776-d9bbc016

### 2026-09-05 · Fix attachment path collision (@developer sub... — tekton-39 (developer) · `done`
- summary: Moved the Claude MCP prompt attachment to `<dir>/.claude/mcp-servers.md`.
- files: `internal/backend/charter_claude_mcp.go`, `internal/backend/charter_claude_mcp_test.go`, `internal/backend/charter_claude.go`, `internal/backend/charter_claude_test.go`
- verify: The final grep has **no `.opencode/mcp-servers.md` path** in either Claude-owned implementation file. The sole match is the deliberately rec
- proof: *Claude attachment path**
- ledgerId: led-1788631960891-61a0572d

### 2026-09-05 · Fix stale MCP listing race (@developer subagent) — tekton-37 (developer) · `issues`
- summary: **Root cause diagnosed:** a same-boot ordering race, not an MCP config parser/enumerator drop. `cmd/theboringfloor/main.go:267` launched `mcpinstall.Ensure` asy
- files: `cmd/theboringfloor/main.go`, `internal/backend/charter_mcp_test.go`
- verify: ```
- proof: *Root-cause evidence**
- ledgerId: led-1788631740781-e18c3ecf

### 2026-09-05 · Claude Code MCP awareness (@developer subagent) — tekton-38 (developer) · `issues`
- summary: Added Claude Code MCP discovery from `$CLAUDE_CONFIG_DIR/.claude.json` (or `~/.claude.json`), matching `projects` entries, and `<project>/.mcp.json`.
- files: `internal/backend/charter_claude.go`, `internal/backend/charter_claude_test.go`, `internal/backend/charter_claude_mcp.go`, `internal/backend/charter_claude_mcp_test.go`
- verify: ```
- proof: ```md
- ledgerId: led-1788631729521-e5c660ed

### 2026-09-05 · Add Open Graph images (@developer subagent) — tekton-36 (developer) · `issues`
- summary: Added build-time `ImageResponse` Open Graph images for the homepage and every statically generated blog post.
- files: `website/app/opengraph-image.tsx`, `website/app/blog/[slug]/opengraph-image.tsx`, `website/app/layout.tsx`, `website/app/blog/[slug]/page.tsx`
- verify: ```
- proof: *Approach: dynamic `ImageResponse`, statically emitted at build time.** Next.js 16.3.0 successfully exported the root and all per-blog-post
- ledgerId: led-1788630881006-50b19428

### 2026-09-05 · Fix footer dead links (@developer subagent) — tekton-35 (developer) · `issues`
- summary: Added the exact approved footer destination map for all navigable labels.
- files: `website/components/site-footer.tsx`
- verify: ```
- proof: ### Features
- ledgerId: led-1788630768192-46d94fd1

### 2026-09-05 · Rename WhyComposio component (@developer suba... — tekton-34 (developer) · `done`
- summary: Renamed the homepage floor-tour component from `WhyComposio` to `FloorTour`.
- files: `website/components/home/why-composio.tsx` → `website/components/home/floor-tour.tsx`, `website/app/page.tsx`
- verify: ```
- proof: ```text
- ledgerId: led-1788630737993-76bdf84c

### 2026-09-05 · Review MCP claim accuracy (@reviewer subagent) — theboringcto-4 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: 4. Move, revert, or explicitly separate the four out-of-scope working-tree changes from this marketing-copy delivery.
- proof: (none)
- ledgerId: led-1788630372983-f0fe5d9f

### 2026-09-05 · Scout prefill reply errors (@explore subagent) — skopos-8 (scout) · `issues`
- summary: Performed read-only repo reconnaissance; made no edits.
- files: `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/model.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/plan_mode.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/app/recent_messages.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/opencode.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/parts.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/events.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/claude.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/internal/backend/models_live.go`, `/Users/theboringhumane/Projects/lynxlabs/theboringoffice/cmd/theboringfloor/main.go`
- verify: ```
- proof: ## Error origins
- ledgerId: led-1788630347611-4c6d5614

### 2026-09-05 · MCP discoverability surfaces (@developer suba... — tekton-33 (developer) · `done`
- summary: Added `/docs/mcp-server` and the pre-existing omitted `/changelog` route to the static sitemap.
- files: `website/app/sitemap.ts`, `website/app/llms.txt/route.ts`, `website/README.md`
- verify: ```
- proof: ```md
- ledgerId: led-1788630139200-be1e34f2

### 2026-09-05 · Homepage MCP section (@developer subagent) — tekton-31 (developer) · `issues`
- summary: Added the `McpServer` homepage section with a six-tool responsive card grid, an agent-tools panel, live-office constraints, and MCP docs link.
- files: `website/components/home/mcp-server.tsx`, `website/app/page.tsx`
- verify: ```
- proof: ### The office MCP server
- ledgerId: led-1788630118543-fc4bef2d

### 2026-09-05 · MCP launch blog post (@developer subagent) — tekton-32 (developer) · `issues`
- summary: Added a 1,817-word engineering blog post announcing `thefloor_mcp` through the honest tension between prompt markers and a typed tool protocol.
- files: `website/content/blog/prompt-markers-are-not-a-protocol.md`
- verify: ```
- proof: ```md
- ledgerId: led-1788630109481-407aa1ba

### 2026-09-05 · Scout blog and voice (@general subagent) — skopos-7 (scout) · `issues`
- summary: Determined that blog posts are Markdown files in `website/content/blog/`, loaded directly from the filesystem. A new post requires one new `<slug>.md` file; its
- files: `website/lib/blog.ts`, `website/lib/blog-types.ts`, `website/app/blog/page.tsx`, `website/components/blog/blog-filter-list.tsx`, `website/app/blog/[slug]/page.tsx`, `website/app/globals.css`, `website/content/blog/WRITING.md`, `website/content/blog/universal-cli.md`, `website/content/blog/watching-subagent-work.md`, `website/content/blog/a-permission-is-not-a-question.md`, `website/content/blog/claude-code-support.md`, `website/content/blog/running-multiple-coding-agents.md`
- verify: No build was run; this was read-only reconnaissance.
- proof: ### Content system and new-post requirements
- ledgerId: led-1788629884252-a06f4d20

### 2026-09-05 · Scout homepage structure (@general subagent) — skopos-5 (scout) · `issues`
- summary: Determined the homepage is assembled in a single ordered list in `website/app/page.tsx:16-39`; it renders 12 marketing sections between the shared header and fo
- files: `website/app/page.tsx`, `website/app/globals.css`, `website/app/layout.tsx`, `website/package.json`, `website/lib/gsap.ts`, `website/components/theme-provider.tsx`, `website/components/section-tag.tsx`, `website/components/scroll-reveal.tsx`, `website/components/site-header.tsx`, `website/components/ui/button.tsx`, `website/components/ui/light-blue-plasma-shader-w-grain-interactive.tsx`, `website/components/home/hero.tsx`
- verify: ```
- proof: ### Homepage Sections In Order
- ledgerId: led-1788629850851-851d9e58

### 2026-09-05 · Scout nav SEO and llms (@general subagent) — skopos-6 (scout) · `issues`
- summary: **Top navigation is hardcoded, not filesystem-derived.** Desktop links are individual `<Link>` elements in `website/components/site-header.tsx:104-121`; mobile 
- files: `website/components/site-header.tsx`, `website/components/site-footer.tsx`, `website/app/docs/page.tsx`, `website/app/docs/mcp-server/page.tsx`, `website/app/layout.tsx`, `website/app/get-started/page.tsx`, `website/app/blog/page.tsx`, `website/app/blog/[slug]/page.tsx`, `website/lib/site.ts`, `website/lib/blog.ts`, `website/app/llms.txt/route.ts`, `website/app/sitemap.ts`
- verify: ```
- proof: | Surface | File | Manual or automatic | Edit needed to register a new page |
- ledgerId: led-1788629846384-1c3fb967

### 2026-09-05 · Restore sessions fallback (@developer subagent) — tekton-30 (developer) · `done`
- summary: Restored the same-product `sessions/<dirhash>/session.json` read fallback in `app.LoadSession`, after the canonical `projects/` path.
- files: `internal/app/sessions.go`, `internal/app/sessions_test.go`, `internal/sessionsearch/sessionsearch.go`, `internal/sessionsearch/sessionsearch_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788627818212-9bc6b55c

### 2026-09-05 · Fix website migration docs (@developer subagent) — tekton-29 (developer) · `done`
- summary: Corrected the getting-started migration guidance: only legacy `THEBORINGOFFICE_*` environment variables remain as silent fallbacks; prior session-layout reads a
- files: `website/app/docs/getting-started/page.tsx`, `website/app/docs/backends/page.tsx`
- verify: ```
- proof: ### Getting Started — rendered migration passage
- ledgerId: led-1788624951494-898c896a

### 2026-09-05 · Rename agent preamble pkgs (@developer subagent) — tekton-25 (developer) · `done`
- summary: Renamed all three agent-facing harness headers to `theboringfloor` without changing their surrounding prompt contracts or ordering.
- files: `internal/browsertools/browsertools.go`, `internal/browsertools/browsertools_test.go`, `internal/browsertools/action/action.go`, `internal/browsertools/action/action_test.go`, `internal/browsertools/action/live_test.go`, `internal/browsertools/action/testdata/fixture.html`, `internal/chatcontext/chatcontext.go`, `internal/plantools/plantools.go`, `internal/plantools/plantools_test.go`
- verify: ```
- proof: ### Harness preamble headers
- ledgerId: led-1788624545907-a75d6ddb

### 2026-09-05 · Rename newer internal pkgs (@developer subagent) — tekton-26 (developer) · `done`
- summary: Routed `internal/control` home override resolution through `config.Env("HOME")`; `internal/config` introduces no import cycle.
- files: `internal/control/control.go`, `internal/control/control_test.go`, `internal/sessionsearch/sessionsearch.go`, `internal/sessionsearch/sessionsearch_test.go`, `internal/mcpinstall/mcpinstall.go`, `internal/mcpinstall/mcpinstall_test.go`, `internal/gitx/attribution.go`, `internal/gitx/attribution_env.go`, `internal/gitx/attribution_env_test.go`, `internal/state/state.go`, `internal/state/state_test.go`
- verify: ```
- proof: ### Session-search root resolution
- ledgerId: led-1788624478363-7707bc21

