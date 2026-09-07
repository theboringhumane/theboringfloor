# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-07 · Fix adaptive icon scale (@developer subagent) — tekton-15 (developer) · `issues`
- summary: Regenerated `assets/logo_foreground.png` with transparent padding reduced from a 50.7% artwork width to **88.3%**, preserving the flower’s aspect ratio and cent
- files: `mobile/assets/logo_foreground.png`, `mobile/android/app/src/main/res/drawable*/ic_launcher_{foreground,monochrome}.png`, `mobile/android/app/src/main/res/mipmap-*/ic_launcher.png`, `mobile/android/app/src/main/res/mipmap-anydpi-v26/ic_launcher.xml`, `mobile/android/app/src/main/res/values/colors.xml`
- verify: ```
- proof: ```text
- ledgerId: led-1788770899972-c60597c4

### 2026-09-07 · In-app loading screen (@developer subagent) — tekton-14 (developer) · `issues`
- summary: Added a theme-aware `SplashView` with the bundled logo and restrained progress indicator.
- files: `mobile/lib/views/splash_view.dart`, `mobile/lib/main.dart`, `mobile/lib/app.dart`, `mobile/test/splash_view_test.dart`, `mobile/assets/logo.png`
- verify: ```
- proof: ```text
- ledgerId: led-1788770416246-626d9f7e

### 2026-09-07 · App icon and native splash (@developer subagent) — tekton-13 (developer) · `issues`
- summary: Converted `website/public/imgs/logo.jpg` into `mobile/assets/logo.png`: a 1080×1071 PNG with alpha/transparency, declared for `Image.asset('assets/logo.png')`.
- files: `mobile/assets/logo.png`, `mobile/assets/logo_foreground.png`, `mobile/assets/logo_dark.png`, `mobile/pubspec.yaml`, `mobile/pubspec.lock`, `mobile/flutter_launcher_icons.yaml`, `mobile/flutter_native_splash.yaml`, `mobile/android/app/src/main/res/mipmap-{mdpi,hdpi,xhdpi,xxhdpi,xxxhdpi}/ic_launcher.png`, `mobile/android/app/src/main/res/mipmap-anydpi-v26/ic_launcher.xml`, `mobile/android/app/src/main/res/drawable-{mdpi,hdpi,xhdpi,xxhdpi,xxxhdpi}/`, `mobile/android/app/src/main/res/drawable/`, `mobile/android/app/src/main/res/drawable-v21/`
- verify: ```
- proof: ```xml
- ledgerId: led-1788770407317-b0e7a4e1

### 2026-09-07 · Run full Go test suite (@runner subagent) — hemerodromos-1 (runner) · `done`
- summary: ## RUNS
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788768872652-8de6e188

### 2026-09-07 · Build transcript features (@developer subagent) — tekton-12 (developer) · `done`
- summary: Rebuilt the transcript view around store-owned paginated message state.
- files: `mobile/lib/views/session_view.dart`, `mobile/lib/store/session_store.dart`, `mobile/lib/components/message_bubble.dart`, `mobile/test/session_view_test.dart`
- verify: ```
- proof: *(a) Transcript on open — newest content is immediately visible**
- ledgerId: led-1788768245581-5924e536

### 2026-09-07 · Finish shell, terminal, settings (@developer... — tekton-11 (developer) · `issues`
- summary: Implemented bounded, cancelable start-office readiness polling in `AppShell`, including:
- files: `mobile/lib/app.dart`, `mobile/lib/views/project_picker.dart`, `mobile/lib/views/terminal_view.dart`, `mobile/lib/views/settings_view.dart`, `mobile/test/app_test.dart`, `mobile/test/terminal_view_test.dart`, `mobile/test/settings_view_test.dart`
- verify: ```
- proof: ```text
- ledgerId: led-1788767903729-a95ab632

### 2026-09-07 · Polish home screen and tiles (@developer suba... — tekton-10 (developer) · `issues`
- summary: Polished `SpaceView` into a minimal inbox-style home screen: large title, pill search field, themed single-select filters, generous spacing, and pull-to-refresh
- files: `mobile/lib/views/space_view.dart`, `mobile/lib/components/project_tile.dart`, `mobile/test/space_view_test.dart`
- verify: ```
- proof: ```text
- ledgerId: led-1788767785783-8de7d3a8

### 2026-09-07 · Restore transcript features (@developer subag... — tekton-9 (developer) · `issues`
- summary: No files changed. The assigned scope cannot implement pagination correctly without an API-layer change that the brief explicitly forbids.
- files: none
- verify: Not run: no implementation was made because the requirements are internally contradictory.
- proof: ```text
- ledgerId: led-1788767489363-3ffad48c

### 2026-09-07 · Flutter architecture + navbar (@developer sub... — theboringcto-3 (cto) · `issues`
- summary: Restructured `mobile/lib/` into API, models, stores, hooks, views, components, and utilities.
- files: `mobile/lib/main.dart`, `mobile/lib/app.dart`, `mobile/lib/api/endpoints.dart`, `mobile/lib/api/gateway_client.dart`, `mobile/lib/api/models.dart`, `mobile/lib/models/models.dart`, `mobile/lib/models/project.dart`, `mobile/lib/models/session.dart`, `mobile/lib/models/transcript.dart`, `mobile/lib/models/exec_result.dart`, `mobile/lib/store/settings_store.dart`, `mobile/lib/store/projects_store.dart`
- verify: ```
- proof: ### Persistent navigation
- ledgerId: led-1788767306605-55b8cda0

### 2026-09-07 · Gateway remote exec/TTY endpoint (@developer... — tekton-8 (developer) · `issues`
- summary: Added authenticated `POST /api/v1/exec` command execution via `/bin/sh -c`.
- files: `cmd/floorgate/gateway.go`, `cmd/floorgate/main.go`, `cmd/floorgate/exec.go`, `cmd/floorgate/exec_unix.go`, `cmd/floorgate/exec_other.go`, `cmd/floorgate/exec_test.go`
- verify: ```
- proof: | METHOD PATH | auth | request JSON | response JSON | status codes |
- ledgerId: led-1788767187441-cb9aaafe

### 2026-09-07 · Scout mobile app + gateway seams (@explore su... — skopos-2 (scout) · `issues`
- summary: Confirms `internal/projects/projects.go` and its test now show as modified (they were not in the earlier `git status --short` at the start of this session — ano
- files: | Path | Why read |, |---|---|, | `mobile/lib/main.dart` | App root, routing decision point |, | `mobile/lib/screens/projects_screen.dart` | Home screen, project list/filter/start-office flow |, | `mobile/lib/screens/session_screen.dart` | Chat/transcript screen, stop/new actions |, | `mobile/lib/api/models.dart` | Wire-shape Dart models, | `mobile/lib/api/gateway_client.dart` | HTTP client, every gateway call site |, | `mobile/lib/settings_store.dart` | Token/base-URL persistence, | `mobile/lib/theme.dart` | Material 3 theme builder |, | `mobile/lib/time.dart` | Relative-time formatter |, | `mobile/test/*.dart`, | `mobile/pubspec.yaml`, `mobile/pubspec.lock` | Declared/resolved packages
- verify: No file was created, edited, or deleted by this task.
- proof: ## Req 1 — File tree
- ledgerId: led-1788767177960-3797ac48

### 2026-09-07 · Transcript pagination API (@developer subagent) — tekton-6 (developer) · `done`
- summary: Added backward transcript pagination using the opaque `before` message-ID cursor.
- files: `internal/app/control.go`, `internal/app/control_test.go`, `internal/controlsrv/server.go`, `internal/controlsrv/server_test.go`, `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: ```http
- ledgerId: led-1788767049533-3a1c4300

### 2026-09-07 · Filter /var project directories (@developer s... — tekton-7 (developer) · `done`
- summary: Added exported, documented `IsSystemDir(dir string) bool`, using cleaned, slash-normalized path segments to identify `/var`, `/private/var`, `/tmp`, `/private/t
- files: `internal/projects/projects.go`, `internal/projects/projects_test.go`
- verify: ```
- proof: | `IsSystemDir` input | Output |
- ledgerId: led-1788767036468-25b9d33c

### 2026-09-07 · Transcript screen redesign (@developer subagent) — theboringcto-2 (cto) · `issues`
- summary: Added `SessionScreen` as a calm, bottom-anchored, reverse transcript view with a 50-message initial page.
- files: `mobile/lib/screens/session_screen.dart`, `mobile/test/session_screen_test.dart`
- verify: ```
- proof: *(a) Transcript on open**
- ledgerId: led-1788766729657-d231e344

### 2026-09-07 · Material You theme and app shell (@developer... — tekton-5 (developer) · `issues`
- summary: Added a single-source Material 3 theme with seeded light/dark fallback schemes and low-elevation, surface-toned component styling.
- files: `mobile/lib/theme.dart`, `mobile/lib/main.dart`, `mobile/pubspec.yaml`, `mobile/pubspec.lock`, `mobile/test/theme_test.dart`, `mobile/test/gateway_client_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788766692741-4c3583e0

### 2026-09-07 · Home screen redesign (@developer subagent) — theboringcto-1 (cto) · `issues`
- summary: Added `ProjectsScreen` as a minimal Material 3 projects surface with live client-side name/directory search and composable All / Running / Stopped filters.
- files: `mobile/lib/screens/projects_screen.dart`, `mobile/test/projects_screen_test.dart`
- verify: ```
- proof: Default list with a running and stopped project:
- ledgerId: led-1788766608075-44946236

### 2026-09-07 · Single-instance office guard (@developer suba... — tekton-4 (developer) · `done`
- summary: Added exported sentinel `ErrLiveDiscovery`, matchable with `errors.Is`, for attempts to replace another live office’s discovery record.
- files: `internal/control/control.go`, `internal/control/control_test.go`
- verify: ```
- proof: | Existing `control.json` state | Owner alive? | Same process? | `WriteDiscovery` result |
- ledgerId: led-1788766228002-b3d016f0

### 2026-09-07 · Harden office launcher detachment (@developer... — tekton-3 (developer) · `done`
- summary: Resolved `theboringfloor` with `exec.LookPath`; missing binaries now return the distinct `ErrOfficeBinaryNotFound` error and launch failures are logged while re
- files: `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`, `cmd/floorgate/process_unix.go`, `cmd/floorgate/process_other.go`, `cmd/floorgate/process_unix_test.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788766224775-808631ce

### 2026-09-07 · Scout headless office boot (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Confirmed — no working-directory flag, no `os.Chdir`, no positional arg handling. Now I have all evidence for the final report.
- files: | Path | Why inspected |, |---|---|, | `cmd/theboringfloor/main.go` | Full boot order: control server start, discovery write, signal trap, | `internal/control/control.go` | `Discovery` shape, `ControlPath`, `WriteDiscovery`/`ReadDiscovery`, | `internal/controlsrv/server.go` | `New()`/`Start()` constructor deps, route table, admission contr, | `internal/app/control.go` | `applyControl`, | `internal/app/control_mutations.go` | What backs `/v1/message`, `/v1/stop`, `/v1/session/new` |, | `internal/app/model.go:3347` | Confirms `applyControl`/`applyControlMutations` are wired into the , | `internal/cellmetrics/input.go`, `cellmetrics.go` | The `tea.WithInput` wrapper's `Fd()`/passthrou, | `internal/panels/zenbu_frame.go:380-432` | The `tea.WithOutput` wrapper's `Fd()`/passthrough seman, | `charm.land/bubbletea/v2@v2.0.9/tea.go` (module cache) | `Run()`'s TTY-fallback logic (`OpenTTY()`, | `charm.land/bubbletea/v2@v2.0.9/tty.go`, `tty_unix.go` | `initInput()`
- verify: Both processes killed (`kill -TERM` then `kill -9` after the 3s teardown deadline), scratch dir removed. No stray processes from this sessio
- proof: | Requirement for headless boot | Exists today? | Where (file:line) | What is missing |
- ledgerId: led-1788766020220-1ac69403

### 2026-09-07 · Document start-office endpoint (@developer su... — tekton-2 (developer) · `done`
- summary: Added the `POST /api/v1/projects/{id}/start` endpoint to the existing gateway API table, including bearer authentication, its empty-body request rule, and async
- files: `website/app/docs/control-plane/page.tsx`
- verify: `bun run lint` was not run because `website/package.json` has no `lint` script.
- proof: | Method path | What it does | Response |
- ledgerId: led-1788765912454-25fb30ab

### 2026-09-07 · App start-office control (@developer subagent) — tekton-1 (developer) · `done`
- summary: Added typed `StartOfficeOutcome` / `StartOfficeResult` API results for accepted launch, already-running, missing-project, and launch-failed responses.
- files: `mobile/lib/api/models.dart`, `mobile/lib/api/gateway_client.dart`, `mobile/lib/main.dart`, `mobile/test/gateway_client_test.dart`, `mobile/test/models_test.dart`
- verify: ```
- proof: ```text
- ledgerId: led-1788765878873-0ea90218

### 2026-09-07 · Scout headless office boot (@explore subagent) — skopos-9 (scout) · `done`
- summary: I have sufficient evidence for all 8 requirements plus a valuable empirical finding. Let me do a final repo-cleanliness check and the last VERIFY commands befor
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788765551313-22c6a09b

### 2026-09-07 · Gateway start-office route (@developer subagent) — tekton-22 (developer) · `issues`
- summary: Added authenticated `POST /api/v1/projects/{id}/start`.
- files: `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: ```http
- ledgerId: led-1788762142720-cb5e840e

### 2026-09-07 · App tolerates missing office routes (@develop... — tekton-20 (developer) · `done`
- summary: Added a `GatewayClient.session` loader that treats `status` and `transcript` as required while safely degrading a failed `busy` request to `null`.
- files: `mobile/lib/api/models.dart`, `mobile/lib/api/gateway_client.dart`, `mobile/lib/main.dart`, `mobile/test/models_test.dart`, `mobile/test/gateway_client_test.dart`
- verify: ```
- proof: ```text
- ledgerId: led-1788761859818-fbd16a5d

### 2026-09-07 · Audit control server route coverage (@develop... — tekton-21 (developer) · `issues`
- summary: Added data-driven route coverage tests for every declared `control.Route*` constant.
- files: `internal/controlsrv/server_test.go`
- verify: ```
- proof: | Route | Method | Status before | Status after |
- ledgerId: led-1788761814730-c35cc144

### 2026-09-07 · Stop floorgate masking office errors (@develo... — tekton-19 (developer) · `done`
- summary: Changed non-2xx office proxy handling to preserve the office’s original HTTP status code.
- files: `cmd/floorgate/gateway.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: ```http
- ledgerId: led-1788761659116-670725e9

### 2026-09-07 · Install Claude Code ledger hook (@developer s... — tekton-18 (developer) · `done`
- summary: Added project-scoped Claude Code hook configuration at `.claude/settings.json`.
- files: `internal/backend/charter_claude.go`, `internal/backend/charter_claude_hooks.go`, `internal/backend/charter_claude_hooks_test.go`
- verify: ```
- proof: Fresh project output:
- ledgerId: led-1788761219290-162c3028

### 2026-09-07 · Extract backend-agnostic ledger recorder (@de... — tekton-17 (developer) · `issues`
- summary: Added backend-agnostic `ReturnRecordInput` and `recordReturn`, which derives the return verdict and ledger entry, mirrors it to agentmemory, and appends it to t
- files: `internal/backend/ledger.go`, `internal/backend/opencode.go`, `internal/backend/ledger_return_test.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788761034131-7512c598

### 2026-09-07 · Scout Claude return signals and hooks (@explo... — skopos-8 (scout) · `issues`
- summary: # Read-Only Reconnaissance Report
- files: | Path | Why inspected |, |---|---|, | `internal/backend/ledger.go` | `LedgerEntry` schema, `Ledger.Append`, dedupe/cap/atomic-write cont, | `internal/backend/opencode.go` (~1340-1505, 2755-2895) | `maybeChildReturned`, `ledgerEntryForRetu, | `internal/backend/boardsync.go` | `reconcileBoardDone`, | `internal/backend/claude.go`, | `internal/backend/claude_events.go` (full) | Found `EvReturned` emission in `mapClaudeUser` (Task/, | `internal/backend/charter_claude.go` | `<dir>/CLAUDE.md` + `<dir>/.opencode/oikonomos.md` writers , | `internal/backend/charter_claude_mcp.go` | `<dir>/.claude/mcp-servers.md` writer, | `~/.claude/settings.json` (redacted) | Live proof a `hooks.Stop` entry is configured on this machi, | `~/.claude.json`, `<repo>/.claude/` | Confirmed presence, listed keys/files only, no values printe, | `/Users/theboringhumane/.local/share/claude/versions/2.1.263` (the resolved `claude` binary) | `st
- verify: Nothing else under `.claude/` is written by theboringfloor — `settings.json`/`settings.local.json` in the repo's `.claude/` (seen in the dir
- proof: *Requirement-7 tradeoff table**
- ledgerId: led-1788760931201-237d495a

### 2026-09-07 · Boot-time stale work reconciler (@developer s... — tekton-14 (developer) · `done`
- summary: Added `state.TaskStalled` with the persisted value `"stalled"` for work owned by a process that died.
- files: `internal/state/state.go`, `internal/app/sessions.go`, `internal/app/sessions_stalled_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788760076364-782bb33a

### 2026-09-07 · Backend death-path recovery (@developer subag... — tekton-15 (developer) · `done`
- summary: Settled OpenCode and Claude in-flight boss placeholders as failed when their child process exits, including zero-text streams.
- files: `internal/backend/opencode.go`, `internal/backend/claude.go`, `internal/backend/events.go`, `internal/backend/claude_events.go`, `internal/backend/process_unix.go`, `internal/backend/process_other.go`, `internal/backend/process_death_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788760031013-c56cf1d1

### 2026-09-07 · Graceful shutdown signal trap (@developer sub... — tekton-16 (developer) · `done`
- summary: Added SIGINT/SIGTERM handling before Bubble Tea starts.
- files: `cmd/theboringfloor/main.go`, `cmd/theboringfloor/signal_shutdown_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788759883962-cb24dfcb

### 2026-09-07 · Scout ledger and dispatch tracking (@explore... — skopos-7 (scout) · `issues`
- summary: # Read-Only Reconnaissance Report
- files: | Path | Why |, |---|---|, | `internal/backend/ledger.go` | The ledger writer/reader itself, | `internal/backend/opencode.go`, | `internal/backend/claude.go` | Confirmed **no** `LedgerEntry`/`NewLedger`/office-ledger reference , | `internal/backend/agentmemory.go` | Confirms `Verdict` also rides an agentmemory observation tag, | `internal/app/model.go`, | `internal/state/state.go` (lines ~230–260, ~360–430) | `BoardTask`, `TaskStatus` (3 states only, | `internal/app/plan_tools.go` | Full file, | `internal/app/plan_mode.go`, | `internal/app/sessions.go` (lines 85–244 only, via targeted grep+read, | `internal/backend/boardsync.go` | `reconcileBoardDone`
- verify: (empty output — success, as expected)
- proof: | Artifact | Written by (file:line) | Trigger | Survives restart? | Encodes unfinished work? |
- ledgerId: led-1788759557706-f98082b7

### 2026-09-07 · Scout backend process lifecycle (@explore sub... — skopos-6 (scout) · `issues`
- summary: Confirmed: only two spawn sites in the whole package (`spawnServe` in opencode.go, `spawnClaude` in claude.go); `charter.go` never spawns a process — it only wr
- files: `internal/backend/backend.go`, `internal/backend/opencode.go` (full read across several windows), `internal/backend/claude.go` (full read across several windows), `internal/backend/claude_events.go` (lines 680-720), `internal/backend/events.go` (grep only, lines ~853-865, ~1743, ~1763), `internal/backend/abort_timeout_test.go`, `internal/backend/opencode_bypass_integration_test.go`, `i, `internal/state/state.go` (lines 683-762), `internal/app/model.go` (lines 3955-4024), `internal/app/stuck_test.go` (grep only), `cmd/theboringfloor/main.go`, `cmd/floorgate/main.go` (grep only), `internal/control/control.go` (grep only), Repo-wide grep for `SysProcAttr|Setpgid|Setsid` and `signal.Notify`
- verify: ```
- proof: | Failure mode | What happens today (file:line) | User-visible result | Recoverable identifier available? |
- ledgerId: led-1788759542465-26b22db6

### 2026-09-07 · Scout crash/resume state seams (@explore suba... — skopos-5 (scout) · `issues`
- summary: Read `internal/app/sessions.go` in full (SessionFile schema, Snapshot, SaveSession, LoadSession, hydrateSession, persistOfficeSession/persistOfficePin call site
- files: `internal/app/sessions.go`, `internal/state/state.go`, `internal/config/config.go`, `internal/app/model.go` (lines 1400–1482, 3940–4230, 5000–5099, plus grepped `permQ`/`question`/`wed, `cmd/theboringfloor/main.go`, `internal/backend/ledger.go`, `internal/backend/opencode.go` (`saveLedgerLanes`/`saveLedgerAsync`), `internal/app/stuck_test.go`, `internal/app/btw_busy_test.go`
- verify: ```
- proof: | State | Persisted? | Where (file:line) | Restored on restart? |
- ledgerId: led-1788759357075-7fc4d9cc

### 2026-09-07 · Document automated APK releases (@developer s... — tekton-13 (developer) · `issues`
- summary: Updated the Android install documentation to state that every version tag is built and signed by GitHub Actions and attached to its GitHub Release as `theboring
- files: `website/app/docs/control-plane/page.tsx`
- verify: ```
- proof: ### Install the Android app
- ledgerId: led-1788758130522-6561acea

### 2026-09-07 · Make NDK pin overridable (@developer subagent) — tekton-12 (developer) · `done`
- summary: Replaced the hardcoded Android NDK version with the optional Gradle property `theboringfloorNdkVersion`.
- files: `mobile/android/app/build.gradle.kts`
- verify: ```
- proof: *Before**
- ledgerId: led-1788758107104-b63e3dce

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

