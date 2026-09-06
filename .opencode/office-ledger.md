# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
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

### 2026-09-05 · Rename internal/panels (@developer subagent) — tekton-22 (developer) · `done`
- summary: Removed all case-insensitive `theboringoffice` references from `internal/panels`.
- files: `internal/panels/browser.go`, `internal/panels/browser_lane.go`, `internal/panels/links.go`, `internal/panels/browser_lane_kitty.go`, `internal/panels/browser_lane_test.go`, `internal/panels/browser_panel_lane_test.go`, `internal/panels/browser_test.go`, `internal/panels/chat_attach.go`, `internal/panels/chat_attach_test.go`, `internal/panels/chat_loading.go`, `internal/panels/popover.go`
- verify: ```
- proof: | Constant / read | Old value | New value | Suffix passed to `config` |
- ledgerId: led-1788624458336-fe1e0e8d

### 2026-09-05 · Fix headless and chrome (@developer subagent) — tekton-23 (developer) · `done`
- summary: Repaired `internal/headless` build by replacing removed `brand.Get("HOME")` with `config.Env("HOME")`.
- files: `internal/headless/headless.go`, `internal/headless/cache.go`, `internal/headless/cache_test.go`, `internal/headless/headless_test.go`, `internal/headless/live_test.go`, `internal/headless/testdata/fixture.html`, `internal/chrome/styles.go`, `internal/chrome/styles_test.go`, `internal/chrome/topbar.go`, `internal/chrome/topbar_test.go`, `internal/chrome/statusbar.go`
- verify: ```
- proof: | Constant | Old value | New value | Suffix passed to accessor |
- ledgerId: led-1788624317569-245b94bc

### 2026-09-05 · Rename scripts and docs (@developer subagent) — tekton-27 (developer) · `issues`
- summary: Renamed helper-script product references, GitHub raw URLs, and auto-commit environment variables to `theboringfloor` / `THEFLOOR_AUTO_COMMIT`.
- files: `scripts/README.md`, `scripts/install-majdoor-hook.sh`, `scripts/majdoor-commit-msg-hook.sh`, `scripts/majdoor-env.sh`, `docs/architecture.md`, `docs/shots/cabins-3000.svg`, `docs/shots/grafeio-1000.svg`, `docs/shots/grafeio-2500.svg`, `docs/shots/grafeio-4000.svg`
- verify: ```
- proof: | File | Old command/path/variable | Final command/path/variable |
- ledgerId: led-1788624281238-d74741d9

### 2026-09-05 · Fix sound notify term (@developer subagent) — tekton-24 (developer) · `issues`
- summary: Replaced all three deleted `brand.Get` call sites with `internal/config` accessors:
- files: `internal/sound/player.go`, `internal/sound/sound.go`, `internal/sound/sound_test.go`, `internal/notify/notify.go`, `internal/notify/notify_test.go`, `internal/term/term.go`, `internal/term/term_test.go`, `internal/netwatch/netwatch.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788624269210-b075fe8b

### 2026-09-05 · Rename internal/backend (@developer subagent) — tekton-16 (developer) · `issues`
- summary: Removed every case-insensitive `theboringoffice` trace from `internal/backend` and `internal/charter`.
- files: `internal/backend/agentmemory.go`, `internal/backend/backend.go`, `internal/backend/browser_open_test.go`, `internal/backend/bypass_permissions_test.go`, `internal/backend/cfg_test.go`, `internal/backend/charter.go`, `internal/backend/charter_claude.go`, `internal/backend/charter_claude_test.go`, `internal/backend/charter_test.go`, `internal/backend/claude.go`, `internal/backend/claude_dialog_kinds_test.go`, `internal/backend/claude_events.go`
- verify: ```
- proof: | Previous environment access | Final access |
- ledgerId: led-1788623912251-c3ab7f37

### 2026-09-05 · Version and installers (@developer subagent) — tekton-17 (developer) · `issues`
- summary: Renamed `internal/version.String()` output and documentation to `theboringfloor`.
- files: `internal/version/version.go`, `internal/version/version_test.go`, `install.sh`, `install.ps1`, `install_ps1_test.go`
- verify: ```
- proof: *New stamped version line**
- ledgerId: led-1788623722715-5f55b1ec

### 2026-09-05 · Rename cmd packages (@developer subagent) — tekton-18 (developer) · `issues`
- summary: Renamed the main command package directory with `git mv`: `cmd/theboringoffice` → `cmd/theboringfloor`.
- files: `cmd/README.md`, `cmd/claudestub/main.go`, `cmd/headless/main.go`, `cmd/soundtest/main.go`, `cmd/theboringoffice/main.go` → `cmd/theboringfloor/main.go`, `cmd/theboringoffice/main_test.go` → `cmd/theboringfloor/main_test.go`, `cmd/thefloor_mcp/main.go`, `cmd/thefloor_mcp/mcp_test.go`, `cmd/uishot/claude_proof.go`, `cmd/uishot/main.go`, `cmd/uishot/terminal_panel_stub.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788623708644-7ba5b142

### 2026-09-05 · Brand and env foundation (@developer subagent) — tekton-14 (developer) · `done`
- summary: Added canonical `config.Env`, `config.EnvBool`, and `config.LookupEnv` accessors with silent legacy fallback and canonical precedence semantics.
- files: `internal/brand/brand.go`, `internal/config/config.go`, `internal/config/migrate.go`, `internal/config/config_test.go`, `internal/config/env_test.go`
- verify: The two surviving lines are the intentional, silent legacy fallback reads in `Env` and `LookupEnv`; no old product, environment, or state-di
- proof: ```go
- ledgerId: led-1788623551002-f54ac3bf

### 2026-09-05 · Rename website (@developer subagent) — tekton-20 (developer) · `issues`
- summary: Replaced all legacy `THEBORINGOFFICE_*` documentation with canonical `THEFLOOR_*` variables.
- files: `website/app/docs/browser-tab/page.tsx`, `website/app/docs/getting-started/page.tsx`, `website/app/docs/mcp-server/page.tsx`, `website/app/layout.tsx`, `website/components/home/context-model.tsx`, `website/components/theme-provider.tsx`, `website/public/_redirects`, `website/public/favicon_io/site.webmanifest`
- verify: ```
- proof: | file:line | string | why it must stay |
- ledgerId: led-1788623541114-1f1163df

### 2026-09-05 · Rename internal/app (@developer subagent) — tekton-15 (developer) · `issues`
- summary: Removed every `theboringoffice` reference from `internal/app`, including comments, notices, status text, chat markers, fixtures, and test expectations.
- files: `internal/app/attribution.go`, `internal/app/backend_switch_test.go`, `internal/app/browser_frame_test.go`, `internal/app/browser_open.go`, `internal/app/browser_open_test.go`, `internal/app/browser_test.go`, `internal/app/control_test.go`, `internal/app/mcp_cmd_test.go`, `internal/app/memory_test.go`, `internal/app/model.go`, `internal/app/model_image_test.go`, `internal/app/notify_hook_test.go`
- verify: ```
- proof: ### Product-prefixed environment call sites changed
- ledgerId: led-1788623481541-0890e2a1

### 2026-09-05 · Rename docs and charter (@developer subagent) — tekton-21 (developer) · `issues`
- summary: Renamed remaining active root documentation references, badges, Go install path, environment variables, and release archive wording to `theboringfloor`.
- files: `README.md`, `.opencode/oikonomos.md`, `.opencode/opencode-video-analysis.md`, `.gitignore`
- verify: ```
- proof: ### README install section
- ledgerId: led-1788623457944-d05ef7d2

### 2026-09-05 · Release config rename (@developer subagent) — tekton-19 (developer) · `issues`
- summary: Renamed the GoReleaser project, primary build ID, and archive ID to `theboringfloor`.
- files: `.goreleaser.yaml`
- verify: ```
- proof: ```yaml
- ledgerId: led-1788623380395-c6821a37

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

