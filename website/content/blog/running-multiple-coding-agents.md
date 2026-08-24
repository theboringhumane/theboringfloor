---
title: "Running more than one coding agent at a time"
description: "What actually breaks when you put Claude Code, Codex, or OpenCode in parallel — and the few things that keep the afternoon from dissolving into panes."
date: "2026-08-24"
author: "theboringoffice team"
categories: ["AI Agents", "Office"]
featured: true
---

Most people get to a second agent the same way. The first one is stuck on a permission, or grinding through tests, and you still have a ticket sitting there. You open another terminal. You tell yourself you will check back in five minutes.

An hour later you have three sessions, two of them waiting on you, and you cannot remember which one owns the auth change.

This is not a model problem. Claude Code, Codex, Cursor, and OpenCode will all write the code. The hard part is staying oriented once more than one of them is moving at once.

## Start with one, longer than you think

A second session is only cheaper than a first session if you can still answer, without scanning every pane: what is in flight, what is blocked, and what landed.

If you cannot answer that for *one* agent — because the transcript is a wall, or because the last useful diff is 400 lines up — two agents will not save time. They will double the reconstruction work.

Give the first session a way to finish without you. Tests it can run. A plan you already approved. An allowlist for the commands you trust (`git status`, the linter, the test target). Then, and only then, open another one.

Anthropic's own guidance on Claude Code is the same shape: explore, plan, implement, and give the agent a check it can run. Parallelism is a later move.

## Isolate the files, or they will fight

Two agents in the same working tree will overwrite each other. This is not theoretical. One will format a file the other is mid-edit. One will revert a hunk it did not understand. You spend the evening unscrewing a merge you never asked for.

The boring fix is git worktrees (or copies, if you are not in git):

```bash
git worktree add ../office-auth -b agent/auth-refresh
git worktree add ../office-tests -b agent/flaky-test
```

Each agent gets a checkout. Each checkout gets a branch. When a session is done, you review the branch the way you would review a coworker, not by scrolling a chat.

Name the sessions the way you name branches. `oauth-migration` is findable tomorrow. `claude-3` is not.

## What tmux is for, and what it is not

tmux (or iTerm splits, or extra Cursor windows) answers one question: can I attach to more than one process?

It does not tell you:

- which session claimed the flaky test
- whether the diff in pane 2 is the one you should read first
- that pane 3 has been sitting on `chmod +x scripts/deploy.sh` for twenty minutes
- what you missed while you were in pane 1

People compensate with named panes, status scripts, and homemade orchestrators. Some of that is fine. A `Ctrl-b w` list of session names is already better than untitled windows. But a list of processes is still not a picture of the work.

If you stay in tmux, at least give every pane a title that matches the branch, and keep a single note — a file, a board, a paper square — of who owns what. The note is the actual coordination. The panes are just where the typing happens.

## Permissions will serialize you if you let them

One agent asking "can I run this?" is a speed bump. Four agents asking, each in their own pane, is a queue you are servicing by alt-tab.

Two habits help:

1. **Pre-approve the boring commands.** Test runners, formatters, `git diff`. If you already type `y` without reading, it should not have been a prompt.
2. **Keep the scary ones queued, not modal.** A write to production config, a network call, a `chmod` on a deploy script — those should wait. They should not freeze every other session while they wait.

If every ask steals the whole screen, "parallel" just means four serial interruptions with extra windows.

## Review in a fresh context

The session that wrote the code is a bad reviewer of that code. It will defend the path it took.

Have a second session (or a subagent with a clean context) look at the diff and only the diff. Tell it what "done" means: the tests that must pass, the files that should not have changed, the edge case you care about. Ask it to report gaps, not style.

Then you read the gaps. Not the full transcript. The transcript is how the agent spent the afternoon. The diff is what you are shipping.

## When this is the wrong move

Do not parallelize a task you could describe in one sentence. A typo, a log line, a rename — one agent, no plan file, no worktree.

Do not parallelize when you do not have a check. If "looks done" is the only signal, you become the test suite for every session at once.

Do not add a fourth agent because the third one is blocked on you. Unblock the third one.

## Why we built a floor instead of another pane manager

We kept hitting the same afternoon: real `opencode` sessions, real sub-agents, and a human (us) reconstructing the shift from scrollback.

theboringoffice is a native terminal app for that shift. The boss is an `opencode` session. Employees are task sub-agents. The floor, roster, board, mail, and work threads are different views of the same work, not extra products. Permission asks stack instead of hijacking the screen. Yesterday's session is still there in the morning.

It will not replace git worktrees, and it should not. Isolation still belongs in the repo. What it replaces is the part where you play air-traffic control across untitled panes.

If you want to see the interaction without wiring a live server:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh

theboringoffice --demo
```

Demo mode is labeled as a tour. When that feels like a workplace instead of a dashboard, run it live — or attach to an `opencode` server you already have:

```bash
theboringoffice --server http://127.0.0.1:4096
```

The goal is not more agents. It is still being able to say, at 4pm, who is doing what.
