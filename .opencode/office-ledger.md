# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-09-13 · Fix Codex model lockout (@developer subagent) — tekton-4 (developer) · `issues`
- summary: Confirms only `codex.go` was pre-tracked (my diff), and my four other scoped files are new/untracked from the larger uncommitted feature — consistent with the b
- files: `internal/backend/codex.go`, `internal/backend/charter_codex_agentmodel_test.go`, `internal/backend/models_codex_test.go`, `internal/backend/models_codex.go`, `internal/backend/charter_codex_agentmodel.go`
- verify: Note on flakes observed (not caused by my changes, none in my scope): two separate runs of the same `-run 'Codex|Agent|Model'`/full-package
- proof: *Send-time check — before (the defect, from `codex.go:197-202` as read at brief time):**
- ledgerId: led-1789273450863-76391bcc

### 2026-09-13 · Harden Claude hook fallback (@developer subag... — tekton-3 (developer) · `issues`
- summary: Read `claude.go`'s hook routing (line 649 originally), `claude_events.go`'s `mapClaudeControlRequest` (line ~1359 switch with no `hook_callback` case), and `cla
- files: `internal/backend/claude.go`, `internal/backend/claude_events.go`, `internal/backend/claude_models_test.go`, `internal/backend/claude_models.go`, `cmd/claudestub/main.go`
- verify: Fix restored immediately after; `gofmt -l` confirmed clean afterward.
- proof: *Matching condition — before:**
- ledgerId: led-1789273407714-e31270be

### 2026-09-13 · Review model selection diff (@reviewer subagent) — theboringcto-1 (cto) · `issues`
- summary: ## VERDICT
- files: | Path | Why inspected |, |---|---|, | `internal/backend/opencode.go` | Req 1, | `internal/backend/models_opencode_selection.go` | Req 1, | `internal/backend/models_test.go` | Req 1, | `internal/backend/models_opencode_selection_test.go` | Req 1, | `internal/backend/claude_models.go` | Req 2, | `internal/backend/claude.go` | Req 2, | `internal/backend/claude_events.go` | Req 2, | `cmd/claudestub/main.go` | Req 2, | `internal/backend/claude_models_test.go` | Req 2, | `internal/app/model_selection.go` | Req 3
- verify: Requirement 8 (secrets/leakage): **Satisfied.** Grepped all new/changed model-selection files for credential-shaped identifiers (`API_KEY`,
- proof: (none)
- ledgerId: led-1789272772746-e56a5907

### 2026-09-13 · Draft v0.9.0 commit message (@general subagent) — tekton-2 (developer) · `issues`
- summary: 1. **Convention**: read `git log -6 --format=%B` in full. Confirmed: subject is `type(scope): lowercase imperative summary`, no trailing period. Scope (`website
- files: | Path | Why it mattered |, |---|---|, | `internal/state/state.go`, `internal/state/model_catalog.go` | Core `ModelInfo` shape + optional c, | `internal/config/config.go`, `internal/config/model_preferences.go` | Backend-scoped preference st, | `internal/backend/opencode.go`, `models_test.go` | Confirms the fail-outright-on-invalid-model beh, | `internal/backend/codex.go`, `models_codex.go` | Confirms `--model` at spawn and the explicit per-, | `internal/backend/claude.go`, `claude_models.go`, `charter_agentmodel.go` | Confirms control_reque, | `internal/app/model_picker.go`, `submodel.go`, `internal/panels/model_picker.go` | Confirms native, | `cmd/uishot/main.go`, `model_proof.go` | Confirms `--modelshot` was replaced wholesale with a dete, | `docs/models.md`, `README.md`, `docs/README.md` | New doc + index links |, | `.opencode/oikonomos.md`, `mobile/pubspec.yaml`, `website/package.json`, `mobile/.gitignore` | The, | `cmd/claudestub/main.go` | Confirms the fixture stub's `list_models`/`set_model` control_request h
- verify: `git status --short` before and after my session is unchanged — I only ran `git log`, `git diff`, `git status`, `git tag`, `cat`, and read t
- proof: ```
- ledgerId: led-1789272472044-89e0f662

### 2026-09-13 · Version and ignore hygiene (@developer subagent) — tekton-1 (developer) · `issues`
- summary: Bumped `mobile/pubspec.yaml` version from `0.7.0+3` to `0.9.0+4` (only the `version:` line changed).
- files: `mobile/pubspec.yaml`, `website/package.json`, `mobile/.gitignore`
- verify: Diff between the two `git status --short` runs: `?? mobile/devtools_options.yaml` disappears; `M mobile/.gitignore`, `M mobile/pubspec.yaml`
- proof: *`mobile/pubspec.yaml` version line**
- ledgerId: led-1789272342717-4396cb3b

### 2026-09-13 · Full release verification (@runner subagent) — hemerodromos-1 (runner) · `issues`
- summary: Everything ran clean. Full gate is GREEN. Here's the complete report.
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1789272188092-a7de4c6d

### 2026-09-13 · Map the release process (@explore subagent) — skopos-1 (scout) · `issues`
- summary: Recon complete. Full report below.
- files: | Path | Why read |, |---|---|, | `internal/state/state.go` | `ModelInfo` gains `Ref`/`Description`/`Disabled`/`IsDefault`, | `internal/config/config.go`, `internal/config/model_preferences.go`, | `internal/app/model.go`, `internal/app/model_selection.go`, | `internal/app/submodel.go`, `internal/app/submodel_test.go` | `/submodel` rewritten from OpenCode-, | `internal/app/model_picker.go`, `internal/app/model_picker_test.go` | boss `/model` slash path rew, | `internal/panels/model_picker.go`, `internal/panels/model_picker_native_test.go`, | `internal/panels/popover.go`, `internal/panels/popover_cmds_test.go` | Slash-help copy for `/model, | `internal/backend/claude.go` | Live Claude backend: `--model` at spawn, control_request/control_re, | `internal/backend/claude_models.go`, `internal/backend/claude_models_test.go`, | `internal/backend/claude_events.go`, `internal/backend/claude_dialog_kinds_test.go` | Wire types f
- verify: ```
- proof: ### 1. The change surface
- ledgerId: led-1789271737135-7c9d6fbd

### 2026-09-07 · Fix activity review findings (@developer suba... — theboringcto-4 (cto) · `issues`
- summary: Added optional `officeWorking` plumbing from `SessionView` through `ActivityGroupRow` into `ActivityBubble`, defaulting to `false`.
- files: `mobile/lib/components/activity_bubble.dart`, `mobile/lib/components/message_bubble.dart`, `mobile/lib/views/session_view.dart`, `mobile/test/activity_bubble_test.dart`
- verify: ```
- proof: *Chip gate — before**
- ledgerId: led-1788802042880-3df3f716

### 2026-09-07 · Draft wave 116 commit message (@general subag... — tekton-27 (developer) · `issues`
- summary: Confirms the v0.4.1→v0.4.2→v0.4.3→v0.4.4 one-patch-per-wave convention; next is v0.4.5.
- files: `internal/control/control.go`, `internal/control/control_test.go`, `internal/app/control.go`, `internal/app/control_activity_test.go` (untracked, new), `cmd/floorgate/gateway_test.go`, `mobile/lib/models/transcript.dart`, `mobile/lib/models/activity_group.dart`, `mobile/lib/components/activity_bubble.dart`, `mobile/lib/views/session_view.dart`, `mobile/test/activity_bubble_test.dart`, `activity_group_test.dart`, `session_view_test.dart`, `tran, `.opencode/office-ledger.md`, `.claude/settings.json`
- verify: ```
- proof: ```
- ledgerId: led-1788801874490-573ca7bf

### 2026-09-07 · Review structured activity diff (@reviewer su... — theboringcto-3 (cto) · `done`
- summary: ## VERDICT — FAIL — the new projection can expose a filesystem path through `Employee.Task`, and the client displays unknown roles as chips despite the specifie
- files: `.claude/settings.json`, `.opencode/office-ledger.md`, `cmd/floorgate/gateway_test.go`, `internal/app/control.go`, `internal/app/control_activity_test.go`, `internal/control/control.go`, `internal/control/control_test.go`, `internal/state/state.go`, `mobile/lib/components/activity_bubble.dart`, `mobile/lib/models/activity_group.dart`, `mobile/lib/models/transcript.dart`, `mobile/lib/views/session_view.dart`
- verify: Satisfied: aside from the scope violations above, changed runtime code is attributable to structured transcript activity; no changed paginat
- proof: (none)
- ledgerId: led-1788801838246-c465fac0

### 2026-09-07 · Full release verification (@runner subagent) — hemerodromos-3 (runner) · `issues`
- summary: ## VERDICT — RELEASE-READY
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788801778873-a5de4947

### 2026-09-07 · Project activity onto transcript (@developer... — tekton-23 (developer) · `done`
- summary: Added O(1) employee-roster indexing per transcript projection.
- files: `internal/app/control.go`, `internal/app/control_activity_test.go`
- verify: ```
- proof: `state.Employee` read before implementation:
- ledgerId: led-1788800188989-09496523

### 2026-09-07 · Wire running state into transcript (@develope... — tekton-26 (developer) · `issues`
- summary: Passed the positional running signal to each activity row: only index `0` (the newest entry in the reversed transcript) receives it, and only while `store.isWor
- files: `mobile/lib/views/session_view.dart`, `mobile/test/session_view_test.dart`, `mobile/lib/components/activity_bubble.dart`, `mobile/lib/models/activity_group.dart`, `mobile/test/activity_group_test.dart`, `mobile/test/transcript_attachment_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788800027217-78228487

### 2026-09-07 · Structured activity in mobile models (@develo... — tekton-25 (developer) · `done`
- summary: Added optional, defensively decoded `TranscriptActivity` data (`role`, `task`, `state`) to transcript messages.
- files: `mobile/lib/models/transcript.dart`, `mobile/lib/models/activity_group.dart`, `mobile/lib/components/activity_bubble.dart`, `mobile/test/transcript_attachment_test.dart`, `mobile/test/activity_group_test.dart`, `mobile/test/activity_bubble_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788799955375-1bfe6fd7

### 2026-09-07 · Transcript activity wire fields (@developer s... — tekton-22 (developer) · `done`
- summary: Added the additive `TranscriptActivity` control-plane wire type with optional `role`, `task`, and `state` JSON fields.
- files: `internal/control/control.go`, `internal/control/control_test.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788799865182-dc67011e

### 2026-09-07 · Gateway activity passthrough test (@developer... — tekton-24 (developer) · `done`
- summary: Added `TestGatewayTranscriptPassesActivityAndFutureFieldsThroughUnchanged`, a hermetic gateway-to-fake-office test that compares the raw transcript response byt
- files: `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: Successful office responses travel as a byte stream, with no transcript JSON struct, decode, validation, or re-encode in the gateway:
- ledgerId: led-1788799841402-129cf909

### 2026-09-07 · Fix two failing Flutter tests (@developer sub... — tekton-21 (developer) · `issues`
- summary: **WorkingChip failure — stale test expectation corrected.** `WorkingChip` rotates only when its four-second `AnimationController` completes; the initial frame m
- files: `mobile/test/working_chip_test.dart`
- verify: ```
- proof: | Failing test | Root cause | Product bug or stale test | What changed |
- ledgerId: led-1788799444418-02e6980b

### 2026-09-07 · Draft release commit message (@general subagent) — tekton-19 (developer) · `issues`
- summary: Enumerated every changed/untracked path with `git status --short` and `git diff --stat HEAD` (44 tracked files, +2267/‑1487, plus ~26 untracked new files/assets
- files: | Path | Why read |, |---|---|, | `internal/app/control.go` | Confirms pagination now counts user turns only, | `internal/control/control.go` | Confirms additive `TranscriptResponse.Working` and `TranscriptMess, | `internal/control/attachments.go` | Confirms `SaveAttachments` now returns `SavedAttachment{Name,M, | `internal/state/state.go` | New `Event.ControlAttachments []Attachment` field |, | `internal/backend/claude.go` | New `claudeAttachmentPrompt` with an explicit "image data was not i, | `mobile/lib/hooks/use_polling.dart` | Confirms lifecycle-aware, backoff-on-failure, overlap-guarde, | `mobile/lib/store/session_store.dart` | Confirms `isWorking`, `pollGeneration`, `newestMessageGene, | `mobile/lib/views/session_view.dart` | Confirms `Polling(enabled: store.isWorking && visible, rest, | `mobile/lib/components/message_bubble.dart` | Confirms long-press-to-copy timer replaces a broken , | `mobile/lib/theme.dart`, `mobile/lib/utils/typography.dart` | Confirms a brand-new `AppFonts`/`bui
- verify: Confirmed: only read-only git commands were run (`status`, `diff`, `diff --stat`, `log`, `tag`, `check-ignore`) plus non-git inspection (`gr
- proof: *(a) Proposed commit message**
- ledgerId: led-1788799182760-34ff2a3a

### 2026-09-07 · Fix two failing Flutter tests (@developer sub... — tekton-20 (developer) · `done`
- summary: Diagnosed **failure 1** as **(a) a stale finder**: the composer’s send affordance changed from `Icons.send_outlined` to an arrow-circle icon, while its `Send fo
- files: `mobile/test/attachment_test.dart`, `mobile/test/working_chip_test.dart`
- verify: ```
- proof: | Failing test | Root cause | Classification | What changed |
- ledgerId: led-1788799118961-01e69678

### 2026-09-07 · Full release verification (@runner subagent) — hemerodromos-2 (runner) · `issues`
- summary: ## RUNS
- files: none
- verify: (none)
- proof: | Check | Command | Result | Notes |
- ledgerId: led-1788798867544-a4de47b4

### 2026-09-07 · Map the release process (@explore subagent) — skopos-5 (scout) · `issues`
- summary: All findings gathered. No modifications were made by me — the diffs shown (`mobile/pubspec.yaml`, `.opencode/office-ledger.md`, etc.) are pre-existing/concurren
- files: | Path | Why read |, |---|---|, | `.github/workflows/release.yml` | The only CI workflow; defines the tag-push trigger and both rele, | `.goreleaser.yaml` | Build matrix, ldflags version stamping, archive naming, changelog grouping |, | `CLAUDE.md` | Confirms only the oikonomos charter is imported; no release policy here |, | `README.md` | Install instructions, badges referencing `release.yml`, doc index |, | `website/AGENTS.md` | Ruled out, | `scripts/README.md`, `scripts/majdoor-commit-msg-hook.sh`, `scripts/install-majdoor-hook.sh` | Com, | `internal/version/version.go`, `internal/version/version_test.go` | The single Go version source o, | `mobile/pubspec.yaml:19` | Mobile app version field |, | `mobile/android/app/build.gradle.kts:50-62` | Confirms `versionCode`/`versionName` are derived fro, | `website/package.json:3` | Website version field |
- verify: --
- proof: ```bash
- ledgerId: led-1788798825341-789d6971

### 2026-09-07 · Smooth auto-scroll on polling (@developer sub... — tekton-18 (developer) · `issues`
- summary: Added a newest-message generation signal to `SessionStore`, allowing the transcript UI to distinguish live refresh arrivals from older-page pagination.
- files: `mobile/lib/views/session_view.dart`, `mobile/lib/store/session_store.dart`, `mobile/test/session_view_test.dart`, `mobile/test/session_store_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788798580174-901fe2e2

### 2026-09-07 · Scout sub-agent activity UI data (@explore su... — skopos-4 (scout) · `issues`
- summary: All confirmed read-only. Here is the complete report.
- files: | Path | Why read |, |---|---|, | `internal/panels/chat_loading.go` | The "team is working, | `internal/panels/activity.go` | Ruled out as unrelated, | `internal/panels/threads_opencode.go` | The sub-agent task-row renderer, | `internal/panels/chat.go`, | `internal/state/state.go`, | `internal/app/control.go` | `controlTranscript`/`controlWorking`, | `internal/control/control.go` | `TranscriptMessage`, `TranscriptResponse`, `BusyResponse` wire sha, | `internal/controlsrv/server.go`, | `cmd/floorgate/gateway.go` (150-239) | Confirms the public gateway is a thin proxy, | `internal/app/model.go` (5180-5383, plus grep hits) | The reducer that actually builds `wtool`/`wt
- verify: ```
- proof: ### 1. The word list
- ledgerId: led-1788798462886-ef7e654e

### 2026-09-07 · Nested sub-agent activity bubbles (@developer... — tekton-17 (developer) · `issues`
- summary: Extended `ActivityGroup` with optional `taskTitle`, optional `agentType`, and ordered `ActivityTool` child entries.
- files: `mobile/lib/models/activity_group.dart`, `mobile/lib/components/activity_bubble.dart`, `mobile/lib/components/message_bubble.dart`, `mobile/test/activity_group_test.dart`, `mobile/test/activity_bubble_test.dart`
- verify: ```
- proof: *Transcript data evidence:** `TranscriptMessage` currently contains only:
- ledgerId: led-1788798244013-c64e3031

### 2026-09-07 · Loading snake + working chip (@developer suba... — tekton-16 (developer) · `issues`
- summary: Added a standalone `LoadingSnake` row-progress widget with:
- files: `mobile/lib/components/loading_snake.dart`, `mobile/lib/components/working_chip.dart`, `mobile/lib/utils/working_words.dart`, `mobile/test/loading_snake_test.dart`, `mobile/test/working_chip_test.dart`, `mobile/lib/models/activity_group.dart`, `mobile/lib/store/session_store.dart`, `mobile/lib/components/activity_bubble.dart`
- verify: ```
- proof: ### Public API
- ledgerId: led-1788798241111-3bc0ca6e

### 2026-09-07 · Fix failing panels test (@developer subagent) — tekton-14 (developer) · `done`
- summary: Characterized the reported PTY integration failure: **0/5** reproductions failed; the race-enabled run also passed.
- files: `internal/panels/browser_lane_kitty_test.go`
- verify: ```
- proof: *Failure characterization:** `TestBrowserLaneKittyFrameState` failed **0 of 5** reproduction runs. Its race-enabled run passed.
- ledgerId: led-1788798222028-8f0bd9c7

### 2026-09-07 · Format and lint mobile app (@developer subagent) — tekton-15 (developer) · `issues`
- summary: Ran `dart format lib test`; it formatted three concurrently modified files: `lib/views/session_view.dart`, `lib/views/space_view.dart`, and `test/session_store_
- files: `mobile/lib/views/session_view.dart`, `mobile/lib/views/space_view.dart`, `mobile/test/session_store_test.dart`, `mobile/test/session_view_test.dart`
- verify: ```
- proof: *Polling enablement — before:**
- ledgerId: led-1788798067290-fcce520c

### 2026-09-07 · Expose agent-busy status on API (@developer s... — tekton-11 (developer) · `done`
- summary: Investigated requirement 1: the transcript response did **not** already expose a busy/working/running signal. The separate `/v1/busy` response has `busy`, but `
- files: `internal/control/control.go`, `internal/app/control.go`, `internal/app/control_test.go`, `internal/controlsrv/server_test.go`, `cmd/floorgate/gateway_test.go`
- verify: ```
- proof: *Working session — before**
- ledgerId: led-1788797899087-99a78e99

### 2026-09-07 · Draft release commit message (@general subagent) — tekton-12 (developer) · `issues`
- summary: 1. **House style internalized** by reading full bodies of waves 114, 113, and 112 (`git log -1 --format=%B <sha>` for `53258b3`, `17c73c6`, `af63eef`). Observed
- files: | Path | Why it mattered |, |---|---|, | `internal/state/state.go` | New `Event.ControlAttachments []Attachment` field, | `internal/control/attachments.go`, | `internal/control/control.go`, | `internal/controlsrv/server.go`, | `internal/app/control.go`, | `internal/app/control_mutations.go` | `currentBackendSend` now forwards `ev.ControlAttachments` in, | `internal/backend/claude.go`, | `internal/backend/parts_test.go` | New proof-test for existing OpenCode real-image-part delivery (, | `cmd/floorgate/gateway_test.go` | Fixture update only, for the new `"working"` JSON field |, | `mobile/lib/hooks/use_polling.dart` | `Polling` gains `enabled`, `restartToken`, failure backoff, 
- verify: (No file under my control was ever touched by `edit`/`write`/`bash -c "... > file"` — every command above is a read: `git log`, `git diff`,
- proof: ```
- ledgerId: led-1788797753438-2bff1c0f

### 2026-09-07 · Poll transcript while working (@developer sub... — tekton-13 (developer) · `issues`
- summary: Wired transcript polling to the existing gateway **`/busy`** signal (`Busy.busy`), not a new transcript `working` field.
- files: `mobile/lib/hooks/use_polling.dart`, `mobile/lib/store/session_store.dart`, `mobile/lib/views/session_view.dart`, `mobile/test/session_store_test.dart`, `mobile/test/session_view_test.dart`
- verify: ```
- proof: ```text
- ledgerId: led-1788797555206-62f6c915

### 2026-09-07 · Full release verification (@runner subagent) — hemerodromos-1 (runner) · `issues`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788797180402-a7de4c6d

### 2026-09-07 · Scout transcript polling state (@explore suba... — skopos-3 (scout) · `issues`
- summary: `mobile/lib/views/session_view.dart` — transcript screen; now wraps the Scaffold in `Polling`
- files: `mobile/lib/views/session_view.dart`, `mobile/lib/store/session_store.dart`, `mobile/lib/hooks/use_polling.dart`, `mobile/lib/api/gateway_client.dart`, `mobile/lib/api/endpoints.dart`, `mobile/lib/models/session.dart`, `mobile/lib/models/transcript.dart`, `mobile/lib/models/project.da, `mobile/lib/app.dart`, `internal/control/control.go`, `internal/app/control.go`, `internal/controlsrv/server.go`, `cmd/floo, `mobile/test/session_store_test.dart`, `mobile/test/session_view_test.dart`
- verify: `Polling.maxConsecutiveFailures` (default 3) with exponential backoff means after 3 consecutive `refresh()` failures polling stops entirely
- proof: | concern | file:line | current behavior | what a developer must change |
- ledgerId: led-1788797139658-6cb909cd

### 2026-09-07 · Untrack Gradle build artifact (@developer sub... — tekton-10 (developer) · `done`
- summary: Added the narrow Android Gradle output ignore rule to `mobile/.gitignore`: `/android/build/`.
- files: `mobile/.gitignore`, `mobile/android/build/reports/problems/problems-report.html`
- verify: ```
- proof: `mobile/.gitignore` now contains:
- ledgerId: led-1788797046603-34ca6a12

### 2026-09-07 · Remove NDK dependency from app (@developer su... — tekton-7 (developer) · `issues`
- summary: Removed `google_fonts` from the Flutter dependency graph; regenerated `pubspec.lock` removes `jni`, `jni_flutter`, `objective_c`, `path_provider*`, `code_assets
- files: `mobile/pubspec.yaml`, `mobile/pubspec.lock`, `mobile/lib/utils/typography.dart`, `mobile/lib/main.dart`, `mobile/test/font_assets_test.dart`, `mobile/test/typography_test.dart`, `mobile/test/theme_test.dart`, `mobile/test/settings_view_test.dart`, `mobile/test/space_view_test.dart`, `mobile/test/terminal_view_test.dart`, `mobile/test/message_bubble_test.dart`, `mobile/test/markdown_body_test.dart`
- verify: There are no `jni`, `jni_flutter`, or `path_provider` entries.
- proof: ### `_withFamily`
- ledgerId: led-1788796101218-0447b0be

### 2026-09-07 · Repair local Android NDK (@developer subagent) — tekton-6 (developer) · `issues`
- summary: Renamed the malformed NDK directory without deleting it:
- files: `/Users/theboringhumane/Library/Android/sdk/ndk/28.2.13676358.broken-20260907`, `/Users/theboringhumane/Library/Android/sdk/ndk/28.2.13676358.reinstall-incomplete-20260907`, No repo file was manually modified. `git status --short` currently contains existing/concurrent work, ```text, M .opencode/office-ledger.md, M internal/app/control.go, M internal/app/control_mutations.go, M internal/app/control_test.go, M internal/backend/claude.go, M internal/backend/claude_attachment_test.go, M internal/backend/parts_test.go, M internal/control/attachments.go
- verify: ```
- proof: | NDK version/path | `source.properties` present? | Healthy? |
- ledgerId: led-1788796093139-4296301e

### 2026-09-07 · Count user messages for paging (@developer su... — tekton-8 (developer) · `done`
- summary: Changed transcript page assembly to count only messages where `From == "user"` against `limit`.
- files: `internal/app/control.go`, `internal/app/control_test.go`
- verify: ```
- proof: ```text
- ledgerId: led-1788796023957-05092dbf

### 2026-09-07 · Paginate by user messages only (@explore suba... — skopos-2 (scout) · `issues`
- summary: All modifications shown are pre-existing/concurrent-agent changes (present before I started and continuing to churn during my read-only recon); I made zero edit
- files: | Path | Why read |, |---|---|, | `internal/app/control.go` | Page assembly function, limit clamp, `before`-cursor resolution |, | `internal/app/control_mutations.go` | Confirmed no pagination logic lives here, | `internal/app/model.go`, | `internal/state/state.go` | `ChatMsg`/`Event` wire shapes, `Kind` doc comment, `EvControlQuery`/`C, | `internal/control/control.go` | `TranscriptMessage`/`TranscriptResponse` wire types, route/query c, | `internal/controlsrv/server.go` | HTTP route dispatch + `transcriptParams` query parsing |, | `internal/controlsrv/server_test.go` | Existing limit/before validation tests |, | `cmd/floorgate/gateway.go` | Public gateway's `/transcript` param forwarding |, | `cmd/floorgate/gateway_test.go` | Existing gateway-level transcript test |, | `internal/panels/chat.go`
- verify: (none)
- proof: | Layer | file:line | Current limit semantics | What must change |
- ledgerId: led-1788795998033-0c55ff4d

### 2026-09-07 · Diagnose Android NDK build break (@explore su... — skopos-1 (scout) · `issues`
- summary: Read-only investigation of the `:jni` Gradle-configuration NDK failure on `mobile/`'s Android debug build: traced the exact pub dependency chain, inspected the 
- files: | Path | Why read |, |---|---|, | `mobile/pubspec.yaml` | Confirm `google_fonts: ^6.3.3` is the only new direct dep, and that all 4 , | `mobile/pubspec.lock` | Full resolved dependency graph, | `mobile/android/app/build.gradle.kts` | App-level `ndkVersion` pin, | `mobile/android/build.gradle.kts` | Root project config, `evaluationDependsOn` ordering |, | `mobile/android/settings.gradle.kts` | Confirms `dev.flutter.flutter-plugin-loader` is what auto-i, | `mobile/android/gradle.properties`, `local.properties` | SDK path, no other NDK overrides |, | `mobile/android/build/reports/problems/problems-report.html` | Corroborates a real Gradle 9 config, | `~/.pub-cache/hosted/pub.dev/jni-1.0.3/android/build.gradle` | **The actual Gradle file that requi, | `~/.pub-cache/hosted/pub.dev/jni-1.0.3/src/CMakeLists.txt` | Confirms native C sources exist and a, | `~/.pub-cache/hosted/pub.dev/jni_flutter-1.0.3/android/build.gradle` | Confirms `jni_flutter` itse
- verify: ```
- proof: ## 1. Dependency chain to `jni`/`jni_flutter`
- ledgerId: led-1788795951627-97278451

### 2026-09-07 · Client paging by user messages (@developer su... — tekton-9 (developer) · `done`
- summary: Merged older transcript pages by message ID and timestamp instead of raw-list concatenation, preventing duplicate IDs while preserving chronological order even 
- files: `mobile/lib/store/session_store.dart`, `mobile/test/session_store_test.dart`, `mobile/test/session_view_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788795884462-fb85f148

### 2026-09-07 · Fix typography review blockers (@developer su... — theboringcto-2 (cto) · `issues`
- summary: Removed the runtime-fetching bypass from `AppFonts`, so all typography helpers and theme slots now invoke the real `google_fonts` loaders even with network fetc
- files: `mobile/lib/utils/typography.dart`, `mobile/lib/views/settings_view.dart`, `mobile/test/typography_test.dart`, `mobile/test/theme_test.dart`, `mobile/test/settings_view_test.dart`, `mobile/test/font_assets_test.dart`
- verify: ```
- proof: ### `AppFonts` bypass removal
- ledgerId: led-1788795315691-9fda982a

### 2026-09-07 · Bundle font assets offline (@developer subagent) — tekton-5 (developer) · `issues`
- summary: Bundled static TTF assets for Inter, Space Grotesk, Playfair Display, and JetBrains Mono under `assets/fonts/`.
- files: `mobile/pubspec.yaml`, `mobile/lib/main.dart`, `mobile/test/font_assets_test.dart`, `mobile/assets/fonts/*.ttf`, `mobile/assets/fonts/LICENSES/Inter-OFL.txt`, `mobile/assets/fonts/LICENSES/SpaceGrotesk-OFL.txt`, `mobile/assets/fonts/LICENSES/PlayfairDisplay-OFL.txt`, `mobile/assets/fonts/LICENSES/JetBrainsMono-OFL.txt`
- verify: ```
- proof: ```yaml
- ledgerId: led-1788794950084-086353e2

### 2026-09-07 · Review typography diff (@reviewer subagent) — theboringcto-1 (cto) · `done`
- summary: ## VERDICT
- files: (none)
- verify: (none)
- proof: (none)
- ledgerId: led-1788794578355-d74bd2bf

### 2026-09-07 · Apply typography to components (@developer su... — tekton-4 (developer) · `done`
- summary: Applied `AppFonts` to shared component typography without changing widget constructors, layout, colors, sizes, weights, scrolling, or interaction behavior.
- files: `mobile/lib/components/message_bubble.dart`, `mobile/lib/components/markdown_body.dart`, `mobile/lib/components/code_block.dart`, `mobile/lib/components/project_tile.dart`, `mobile/lib/components/empty_state.dart`, `mobile/lib/components/attachment_chip.dart`, `mobile/lib/components/transcript_attachment_chip.dart`, `mobile/lib/utils/markdown_theme.dart`, `mobile/test/markdown_body_test.dart`, `mobile/test/message_bubble_test.dart`
- verify: ```
- proof: | Component | Element | Font family applied |
- ledgerId: led-1788794318660-62e2d9a4

### 2026-09-07 · Typography foundation (@developer subagent) — tekton-2 (developer) · `issues`
- summary: Added `google_fonts ^6.3.3` and resolved it in `pubspec.lock` (`google_fonts` declares Dart SDK `^3.9.0`, compatible with this app’s `^3.13.2`).
- files: `mobile/pubspec.yaml`, `mobile/pubspec.lock`, `mobile/lib/theme.dart`, `mobile/lib/utils/typography.dart`, `mobile/test/typography_test.dart`, `mobile/test/theme_test.dart`
- verify: ```
- proof: ```dart
- ledgerId: led-1788794211284-1c8b0e74

### 2026-09-07 · Apply typography to views (@developer subagent) — tekton-3 (developer) · `issues`
- summary: Applied `AppFonts.serif` exactly once to the **All Repos** home hero title, preserving the existing `displaySmall` style.
- files: `mobile/lib/views/space_view.dart`, `mobile/lib/views/terminal_view.dart`, `mobile/lib/views/settings_view.dart`, `mobile/lib/views/project_picker.dart`, `mobile/test/space_view_test.dart`, `mobile/test/terminal_view_test.dart`, `mobile/test/settings_view_test.dart`
- verify: ```
- proof: | Screen | Element | Font family applied |
- ledgerId: led-1788794084396-d63ad0d5

### 2026-09-07 · Fix stale Claude prompt test (@developer suba... — tekton-1 (developer) · `done`
- summary: Updated `TestClaudeSendWithWritesQuotedPathReferences` to assert the complete Claude attachment prompt, including the explicit non-inlined-image disclosure.
- files: `internal/backend/claude_attachment_test.go`
- verify: The live-Claude tests self-skipped because `THEFLOOR_LIVE_CLAUDE=1` was intentionally not set.
- proof: | Old expected attachment prompt suffix | New expected attachment prompt suffix |
- ledgerId: led-1788793933492-1d928ad2

### 2026-09-07 · Backend image delivery audit (@developer suba... — tekton-29 (developer) · `issues`
- summary: Added table-driven tests that use real temporary PNG files to prove OpenCode payloads contain:
- files: `internal/backend/claude.go`, `internal/backend/parts_test.go`
- verify: ```
- proof: ### OpenCode `prompt_async` body captured by the real-PNG unit test
- ledgerId: led-1788792517112-32c9eef0

### 2026-09-07 · Office app attachment delivery (@developer su... — tekton-28 (developer) · `issues`
- summary: Passed `ev.ControlAttachments` directly to `currentBackendSend` for `EvControlSend`; text-only sends still pass a nil attachment slice.
- files: `internal/app/control_mutations.go`, `internal/app/control.go`, `internal/app/control_attachments_test.go`
- verify: ```
- proof: *`EvControlSend` branch — before**
- ledgerId: led-1788792510828-466acb70

### 2026-09-07 · Mobile transcript attachment chips (@develope... — tekton-30 (developer) · `done`
- summary: Added `TranscriptAttachment` metadata parsing to transcript messages; absent/null/malformed attachment data safely resolves to an empty or filtered list.
- files: `mobile/lib/models/transcript.dart`, `mobile/lib/components/message_bubble.dart`, `mobile/lib/components/transcript_attachment_chip.dart`, `mobile/test/message_bubble_test.dart`, `mobile/test/transcript_attachment_test.dart`, `mobile/test/session_view_test.dart`
- verify: ```
- proof: Transcript JSON decoded by the mobile client:
- ledgerId: led-1788792443841-b28315c1

### 2026-09-07 · Event attachment field (@developer subagent) — tekton-26 (developer) · `issues`
- summary: Added additive `Event.ControlAttachments []Attachment` immediately after `ControlText`.
- files: `internal/state/state.go`, `internal/state/event_attachments_test.go`
- verify: ```
- proof: ```go
- ledgerId: led-1788792242958-f832492b

