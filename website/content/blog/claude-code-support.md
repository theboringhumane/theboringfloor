---
title: "The office doesn't care which brain the boss has"
description: "Claude Code is now a supported LLM transport next to opencode. Same floor, same roster, same board, same ledger — you pick which CLI the boss thinks with at boot."
date: "2026-08-26"
author: "theboringfloor team"
categories: ["Release", "Updates"]
featured: false
---

Claude Code is now a transport the boss can run on. Boot with `--backend claudecode` and you get the identical office — floor, roster, board, mail, ledger — with the Claude Code CLI doing the thinking instead of `opencode serve`. opencode stays the default. Both transports stay.

This is not a pivot. The office was never the model. The office is the shift: who is on it, what is claimed, what came back, what the ledger says happened while you were making coffee. The boss's brain is a pipe with opinions, and a pipe is swappable if the event model holds. It holds. So now there are two.

The honest reason: some of you live in Claude Code. The subscription is paid, the muscle memory is paid, sometimes the organization decided for you. Asking those members to adopt a second runtime to get a floor and a board was a tax, not a feature. We removed the tax.

This is the same argument we make about [attaching to the session you already have](/blog/universal-cli), one level down. Kidnapping the runtime is kidnapping the runtime, whether the kidnapper is a dashboard or our own installer. The office should meet your CLI where it lives. For a lot of terminals, where it lives is `claude`.

## Pick at boot, or in the config

For one run:

```bash
theboringfloor --backend claudecode
```

To make it stick, edit `~/.theboringfloor/configs/brain.json`:

```json
{
  "backend": { "name": "claudecode" }
}
```

No flag and no key means opencode, exactly like before. The default did not move under you.

## A process per session, not a daemon

For each office session we spawn exactly one Claude Code process:

```bash
claude -p --input-format stream-json --output-format stream-json --verbose --include-partial-messages
```

Not a daemon. Not a shared server holding everyone's state hostage. One session, one child process, pinned to that session's uuid. The stream-json stdout is JSONL, and we normalize it into the same event model the opencode transport feeds: partial messages, tool calls, results, control requests.

Downstream of that seam, the office cannot tell which runtime the boss is on. That is the whole design. The floor does not render "claude events" or "opencode events." It renders events. The opencode side of the seam — `opencode serve` plus SSE — keeps feeding the identical model from the other direction.

We asked for `--include-partial-messages` because a boss that only speaks in finished paragraphs makes the chat feel like a fax machine. Partial stream in, live transcript out. You watch the boss think the way you watch a person type: messily, in order, interruptible.

The per-session shape also buys the failure story. If the claude process dies mid-turn, the next send respawns it with `--resume <pinned session uuid>`. The shift continues from where it stopped, not from a blank prompt. Memory that survives a dead child is the difference between an office and a chat window. And when one session's boss wedges, it is one session's problem — there is no shared process to take the rest of the floor down with it.

## Parity is the feature

A second runtime that only does chat is a demo. Parity means the surfaces you already learned keep working, because the control surfaces map one-to-one:

| Claude Code control surface | What the office shows |
| --- | --- |
| `can_use_tool` control request | A permission card. once / always / reject map to `allow` / `allow_always` / `deny` |
| The boss asks a question | The question modal — [questions are not permissions](/blog/a-permission-is-not-a-question) |
| You hit `/stop` | Interrupt control request, then SIGINT, then SIGTERM. A signal-killed child is a clean kill |
| Partial messages while the boss thinks | Boss chat streams live, not in one lump at the end |

Everything around those surfaces is the same build: the floor, the roster, the board, mail, the ledger, session restore, themes, sounds. Session restore deserves its own clause — quitting the TUI on claudecode and coming back inherits the room, the same as it does on opencode. If you could run the office on opencode yesterday, nothing about running it on claudecode today asks you to relearn a keystroke.

The permission mapping is the one worth reading twice. Claude Code's `can_use_tool` control request is a question with a payload, and the office already has a card for questions with payloads. once becomes `allow`. always becomes `allow_always` — still your responsibility to only press it on a [pattern you can say out loud](/blog/tool-router-beta). reject becomes `deny`. The queue, the `1 of N`, the whole discipline carries over unchanged, because it was never opencode's discipline. It was the office's.

## When claudecode is the wrong move

You do not have Claude Code — no subscription, no seat, no habit. Stay on opencode. It is the default, it is first-class, and it is not being wound down.

You already like opencode and expect a switch to make the office *better*. It will not. Parity means parity: same floor, same queue, same ledger. Switch transports because your runtime situation demands it, not because the word "new" itched.

You want one global claude daemon multiplexing every session. We did not build that, on purpose. Per-session processes are how a wedged boss stays one session's problem and how `/stop` can mean it.

## How we encode this

In theboringfloor, the transport is a detail behind the event model:

- Two transports — `opencode serve` over SSE, and the Claude Code CLI over stream-json — normalize into one stream the whole office renders.
- One child process per session, pinned to a uuid; a respawn resumes the shift instead of restarting it.
- Control requests become the same cards, modals, and `/stop` you already use.

```bash
theboringfloor --backend claudecode
```

Then watch the floor, not the transport. If you can tell which brain the boss has from anywhere but the boot line, that is a bug — the office is supposed to be the boring part.

## Further reading

- [Put the workspace where the agents already are](/blog/universal-cli)
- [A permission is not a question](/blog/a-permission-is-not-a-question)
- [You launched the subagents. Now you have to watch them.](/blog/watching-subagent-work)
