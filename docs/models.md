# Model selection

`/model` opens the active backend's native main-model catalog for OpenCode, Codex, or Claude Code. The installed native client and its current listing determine the available rows. If discovery is unavailable, use `/model <native-ref>` to enter a supported reference manually.

| Command | Action |
|---|---|
| `/model` | Pick the main (boss) model. |
| `/model <native-ref>` | Set the main model manually. |
| `/submodel` | Pick a native agent type, then its model, where supported. |
| `/submodel <agent>` | Open the model picker for that native agent type. |
| `/submodel <agent> <native-ref>` | Set a supported native agent/model choice manually. |

Native agent names come from the backend; they are distinct from the floor roster's roles. Manual commands retain native validation and do not bypass an unsupported capability.

Type to filter a catalog, use arrows to move, and press Enter to select. Escape cancels browsing, including a catalog that is still loading. Once a selection is applying, input waits for acknowledgment. The app saves the preference only after the backend adapter accepts it; it applies to future requests. A save failure is reported separately when the runtime change has already applied.

Discovery and selection errors include the native failure and guidance where available. A rejected choice is not saved or silently replaced with a default. A catalog's **default** marker describes native metadata; it does not establish the effective model of a resumed session. The selected marker reflects the saved office override.

## Native references and limits

These are syntax examples, not a guarantee of availability for your account or installed client:

| Backend | Main-model example | Native agent behavior |
|---|---|---|
| OpenCode (`opencode`) | `/model anthropic/claude-sonnet-4` | Uses `provider/model` references and selectable native subagents. Agent model changes preserve existing project agent prompts, tools, and unrelated configuration. |
| Codex (`codex`) | `/model gpt-5.4` | Per-agent switching is currently unavailable because native role model settings override explicit dispatch models; configure Codex roles directly or inherit models. |
| Claude Code (`claudecode`) | `/model sonnet` or `/model opus[1m]` | Agent choices use native aliases, such as `sonnet` or `opus`, from the available catalog. |

OpenCode agent model changes require every session in the project to be idle; busy or retrying sessions cause the change to be rejected. Once idle, the office preserves agent configuration, refreshes the native cache, and verifies activation without recreating the conversation.

Codex has no universal reset-to-default operation for resumed threads. Select an explicit native model when changing a resumed thread's model; `gpt-5.4` above is only an example token.

For Claude Code, an Opus agent alias derived from `opus[1m]` is labeled as an agent alias and does not carry the main session's `[1m]` context modifier. Saved agent choices fill in a missing dispatch model; an explicit dispatch model is retained. Fork agents always inherit their parent's model, and a dispatch without a native agent type cannot use a type-specific preference. Native agent model hooks must be acknowledged by Claude Code.

## Saved preferences

Accepted choices are saved in `~/.theboringfloor/configs/brain.json` under `modelPreferences`, keyed by `opencode`, `codex`, or `claudecode`. Each backend has a `boss` reference and an `agents` map keyed by native agent name. Choices remain isolated when switching backends. For example:

```json
{
  "modelPreferences": {
    "opencode": {"boss": "anthropic/claude-sonnet-4", "agents": {"explore": "anthropic/claude-sonnet-4"}},
    "codex": {"boss": "gpt-5.4"},
    "claudecode": {"boss": "opus[1m]", "agents": {"Explore": "sonnet"}}
  }
}
```

Use the exact native names and references listed by your backend. The schema can store a Codex `agents` map, but that does not enable the unsupported per-agent operation.

If an OpenCode scoped choice is absent, the main model falls back to `backend.bossModel` (`Backend.BossModel`), then `boss.model` (`Boss.Model`); native agent choices fall back to the legacy `agentModels` map. Claude Code and Codex do not inherit those legacy OpenCode values. An explicit empty scoped value masks the legacy fallback; clearing an office override does not promise to reset a resumed native session.
