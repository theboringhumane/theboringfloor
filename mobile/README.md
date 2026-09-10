# theboringfloor for Android

A native companion for your project floors. Connect to your authenticated `floorgate` in Settings, then open a project from **Floors**.

The **Inbox** is your landing page: plans waiting for review, blocked tickets, and tickets in review across all floors. Switch to All floors to see work in progress. Foreground refreshes run every 15 seconds with at most four concurrent floor requests; failures keep the previous view with a visible warning. It does not send push notifications or run work in the background.

Ticket details link the conversation, acceptance criteria, result summary, and recorded verification notes. **Prepare in live chat** links the ticket to the active backend/session and stages a prompt for review before sending. It never marks a ticket done. Verification notes are authored records, not automatically attested checks. Versioned ticket edits reject stale writes from another view.

Office status distinguishes planning, working, ready, offline, plan review, permission, and questions. Permissions and questions still require the desktop office. These execution states and ticket handoffs require **office and gateway v0.7.0+**; older gateways retain ordinary workspace access. Connection settings take effect immediately, and a gateway outage no longer prevents opening Settings.

- **Conversations:** browse saved conversations, read their history, resume one, or start a new conversation with Claude Code, OpenCode, or Codex and a floor team.
- **Board:** filter by team and edit tickets, status, priority, owner, description, and checklists. Add teams directly from the floor.
- **Files:** expand the project tree, preview source with line numbers, and stage a question about a file in chat. Previews are read only and bounded to 64 KiB; symlinks cannot escape the project.
- **Plan:** review, edit, and approve the assistant's plan. New plans open for review in live chat. Approval is tied to the exact draft you reviewed; a changed server draft must be reviewed again.

Chat shows user prompts and completed primary assistant replies. Tool calls, reasoning, worker activity, notices, and progress prose are omitted from the reading view. Commands and code blocks that are part of a real answer remain readable. Raw history stays intact and supplies pagination cursors. Older servers without a phase field use the last completed primary reply in each user turn.

Floor workspaces require the matching **v0.6.0 or newer gateway and office**. The transcript filter is entirely in the Android app. Gateway bearer authentication applies to all workspace, file, archive, and mutation routes; live floor writes run through the owning office.

## Development

```sh
flutter pub get
flutter analyze
flutter test
flutter build apk --release
```

Release signing uses `android/key.properties`; see `android/key.properties.example`. CI derives the app version from the release tag and uses a monotonically increasing workflow run number for Android updates.

The phone-size interaction fixtures cover teams, ticket edits, backend selection, clean transcripts, file previews, and stale plan approval. To render these screens locally:

```sh
mkdir -p /tmp/floor-mobile-shots
flutter test test/floor_view_test.dart --update-goldens --dart-define=FLOOR_SHOTS=true
```
