---
title: "Put the workspace where the agents already are"
description: "Coding agents live in the terminal. A dashboard that 'manages' them is another tab to lose. Attach to the session you have — restore yesterday — don't kidnap the runtime."
date: "2026-08-20"
author: "theboringoffice team"
categories: ["Release", "Updates"]
featured: false
---

You already have a session. Claude Code in a pane. OpenCode on `:4096`. Codex in a worktree. The work is happening. The files are on disk. The tests are a command you could run without a vendor.

Then a product asks you to open a dashboard. New login. New tab. Maybe a new runtime that is "their" agent, not yours. The actual generation is still in the terminal, or it has been quietly moved, and you will find out which when the bill arrives.

You now have two places that can be wrong. The workspace should attach to the session, not kidnap it. This sounds obvious until you have spent a week in a browser control plane that cannot see the PTY you already paid for.

We built a CLI because that is where the agents we use already live. Not because the terminal is morally superior. Because a second home for the same work is how status diverges, and diverged status is how you merge the wrong branch.

## The second home problem

In the terminal: the model just wrote `auth.go`. In the dashboard: last event was "planning" from two minutes ago, or a websocket that stalled, or a pretty card that has not refreshed. You believe the dashboard because it has a logo. You ship the terminal's files. Or the reverse.

Browser workspaces also have a habit of becoming the runtime. To make the cards true, they run the agent. Now you have their sandbox, their keys, their idea of a worktree. Your editor is a spectator. Spectators do not catch the `chmod`.

Attach-first tools are boring to market. They look like "just a skin." A skin over a live session is a lot of what you need: permissions, diffs, a board, a roster — *of work that already exists*. Skins that secretly replace the session are not skins.

## One process, several views of the same shift

You do not need six apps. You need six questions, answered from one event stream:

| Question | View |
| --- | --- |
| What did I tell the boss? | Chat |
| What is the shell doing? | A real PTY, not a fake |
| Who is on a task? | Roster |
| What is claimed / blocked / done? | Board |
| What came back? | Mail |
| What just happened? | Activity |

If those views are different products, you will desync them. Slack for chat, GitHub for the board, a vendor UI for the agent, tmux for the shell. Each is fine. Together they are a scavenger hunt. `tab` between views of one stream is not exciting. It is how you still know, at 4pm, which question you are in.

A fake terminal — a web div that approximates a PTY — will betray you the first time you need a real TUI, a pager, or a program that asks about window size. If the workspace cannot host a real shell, it is a chat app with ambitions.

## Attach beats spawn

Spawning "our" agent runtime is how vendors lock the afternoon. Attaching is how you keep the one you already configured, with the MCP servers you already fought, the `CLAUDE.md` you already wrote, the allowlists you already regret.

If you already have a server:

```bash
theboringoffice --server http://127.0.0.1:4096
```

If you need yesterday's room, not a new one:

```bash
theboringoffice --session <your-session-id>
```

A workspace that cannot attach will eventually fight your toolchain. It will want its own keys. It will not see your worktree. It will offer a "migration." Migrations are how Tuesdays die.

Spawn is not evil. Demo mode spawns a fake stream so you can learn the keys. Live mode should prefer attach when attach is possible. The order is the product ethic.

## Context is part of the interface

Quitting the TUI should not erase the room. Restore the last chat by default. Keep a path to a named session. Put config in a file you can `cat`:

```bash
theboringoffice --print-default-config
```

If "memory" is a slogan and the next launch is a blank prompt, you paid for yesterday twice. [We wrote about that](/blog/self-improving-ai-system). Native CLI is not sufficient for memory — a Go binary can still greet you empty — but a CLI that hides config in a keychain you cannot dump is how "it forgot" becomes un-reproducible.

Electron is not a moral failing. It is another process, another updater, another thing that wants focus. If your agents are in a PTY, a Go binary that attaches is less of a circus. If your agents only live in a browser IDE, a TUI is extra weight. Stay where the files are. We are not trying to win that case.

## When a native CLI is the wrong move

Agents only in a browser IDE, never in a PTY: do not add a terminal office for the aesthetic. You will alt-tab back to the IDE and the office will rot.

Screenshare to someone who should not see a TUI: record a PR, not a terminal. We will not win a design-review aesthetic contest and we should not try.

You wanted a hiring marketplace for agents, a headcount fantasy, a control plane for a company that does not exist: different product. We do not have one. [The UI should not pretend we do](/blog/agents-can-sign-up).

You need Windows-first, click-first, no-shell: we are the wrong tool. That is a ceiling, not a sneer.

## How we encode this

theboringoffice is a Go binary, not Electron. Boss = real `opencode` session. Employees = opencode task sub-agents. Board and mail sit on agentmemory. Try the labeled tour first:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh

theboringoffice --demo
```

Then attach. If attach feels worse than the demo, the demo was the liar. Believe attach.

## Further reading

- [If the UI can lie, you will believe it](/blog/agents-can-sign-up)
- [Memory is whether tomorrow inherits today](/blog/self-improving-ai-system)
- [Don't let approvals steal the screen](/blog/tool-router-beta)
- [Securely indexing large codebases](https://cursor.com/blog/secure-codebase-indexing) — Cursor, on not making the user wait for a second copy of work they already have
