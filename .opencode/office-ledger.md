# Office Ledger — completed work

The ledger in ./.opencode/office-ledger.md is the office's own completed-work memory: before dispatching sub-agents, check it (and agentmemory if equipped) so already-completed work is not repeated. Newly completed dispatches are recorded there by the app automatically; quote its ledger IDs when the member asks whether something was done.

Entries land newest-first below as `### YYYY-MM-DD · <title> — <worker> · <verdict>`,
recorded by the office app itself after each verified return. This file is
append-only state: the charter pass seeds it when absent and never rewrites it.

<!-- ledger:entries -->
### 2026-08-26 · Dev: board sync hook on completion (@general... — tekton-1 (developer) · `issues`
- summary: All checks green and the tree is clean. Final return:
- files: `internal/backend/boardsync.go`, `internal/backend/opencode.go`, `internal/backend/boardsync_test.go`, `cmd/uishot/main.go`, `README.md`, (not mine: `.opencode/*` pre-existing boss-session edits; the tracked `uishot` binary was briefly re
- verify: ```
- proof: ```
- ledgerId: led-1787747811860-aab00add

