# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
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

