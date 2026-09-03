---
title: "Running more than one coding agent at a time"
description: "A second terminal feels like speed. Then you have three sessions, two waiting on you, and no idea who owns auth. What actually breaks — and the few habits that keep the afternoon intact."
date: "2026-08-24"
author: "theboringfloor team"
categories: ["AI Agents", "Office"]
featured: true
---

Most people get to a second agent the same way. The first one is stuck on a permission, or grinding through tests, and a ticket is still sitting there. You open another terminal. You tell yourself you will check back in five minutes.

An hour later you have three sessions. Two of them are waiting on you. You cannot remember which one owns the auth change. Pane 2 has a diff that looks important. Pane 3 has been sitting on `chmod +x scripts/deploy.sh` since you were in pane 1, reading a thinking block that auto-collapsed.

This is not a model problem. Claude Code, Codex, Cursor, and OpenCode will all write the code. The hard part is staying oriented once more than one of them is moving. Parallelism does not fail at generation. It fails at *you*.

We have lost more time to reconstruction than to slow models. The rest of this post is the reconstruction bill, itemized, and the cheapest ways we have found to stop paying it.

## Start with one, longer than you think

A second session is only cheaper than a first session if you can still answer, without scanning every pane: what is in flight, what is blocked, and what landed.

If you cannot answer that for *one* agent — because the transcript is a wall, or the last useful diff is 400 lines up — two agents will not save time. They will double the archaeology.

Give the first session a way to finish without you. A test it can run. A plan you already approved. An allowlist for the commands you already type `y` on without reading (`git status`, the linter, the test target). Then, and only then, open another one.

Anthropic's [guidance on effective agents](https://www.anthropic.com/engineering/building-effective-agents) is the same shape: simplest path first, complexity when the path forks. [We wrote a whole post](/blog/start-with-a-workflow) about not hiring a committee for a loop-with-a-check. Parallelism is a later move on that ladder, not a personality.

A practical test before the second pane: can you walk away for ten minutes and come back to a green test or a parked ask, rather than a frozen modal in the only window you have?

## Isolate the files, or they will fight

Two agents in the same working tree will overwrite each other. This is not theoretical. One will format a file the other is mid-edit. One will revert a hunk it did not understand because the file changed under it. You spend the evening unscrewing a merge you never asked for.

We have seen "I'll just tell them not to touch the same files" last about twenty minutes. Models do not have your mental map of ownership. They have a prompt, a git status, and optimism.

The boring fix is git worktrees (or copies, if you are not in git):

```bash
git worktree add ../office-auth -b agent/auth-refresh
git worktree add ../office-tests -b agent/flaky-test
```

Each agent gets a checkout. Each checkout gets a branch. When a session is done, you review the branch the way you would review a coworker: `git diff main`, the tests, the files that should not have changed. You do not review by scrolling a chat.

Name the sessions the way you name branches. `oauth-migration` is findable tomorrow. `claude-3` is a riddle. Pane titles should match. If tmux says `bash` and the worktree is `office-auth`, you have already lost the mapping.

Cursor's swarm notes go further — at a thousand commits a second you need a VCS that is not Git. You are not at a thousand commits a second. You are at three panes. Worktrees are the version of that lesson that fits a Tuesday.

## What tmux is for, and what it is not

tmux, iTerm splits, extra Cursor windows, extra VS Code windows: they answer one question. Can I attach to more than one process?

They do not tell you which session claimed the flaky test. They do not tell you whether the diff in pane 2 is the one you should read first. They do not tell you pane 3 has been sitting on a permission for twenty minutes. They do not tell you what you missed while you were in pane 1.

People compensate with named panes, status scripts, homemade orchestrators, a sticky note on the laptop. Some of that is fine. `Ctrl-b w` with real names is already better than untitled windows. A list of processes is still not a picture of the work.

If you stay in tmux, treat it as a process manager, not a project manager. The project manager is a board, a file, a paper square: who owns what, what is blocked, what is waiting on you. The note is the coordination. The panes are where the typing happens.

When the note and the panes disagree, the panes are lying. Agents will claim work in chat and never update the note. Believe the board you force them (and yourself) to write down.

## Permissions will serialize you if you let them

One agent asking "can I run this?" is a speed bump. Four agents asking, each in their own pane, is a queue you are servicing by alt-tab. You are not parallel. You are a single-threaded approval router with extra monitors.

Two habits help.

**Pre-approve the boring commands.** Test runners, formatters, `git diff`. If you already type `y` without reading, it should not have been a prompt. That tell is worth more than any policy doc.

**Keep the scary ones queued, not modal.** A write to production config, a network call, a `chmod` on a deploy script — those should wait. They should not freeze every other session while they wait. If every ask steals the whole screen, "parallel" means four serial interruptions with extra windows.

There is a third habit that belongs in [the taxonomy post](/blog/a-permission-is-not-a-question): a product decision is not a shell allow. If "should this be public API" arrives in the same box as `git status`, you will approve an API with your `y` finger.

## Review in a fresh context

The session that wrote the code is a bad reviewer of that code. It will defend the path it took. It will explain why the extra interface is "cleaner." It will not see the files it forgot.

Have a second session, or a subagent with a clean context, look at the diff and only the diff. Tell it what "done" means: the tests that must pass, the files that should not have changed, the edge case you care about. Ask it to report gaps, not style. Then you read the gaps.

Chasing every finding from a reviewer that was told to "find issues" is how you get a `UserCacheFactory` you did not ask for. Instruct it like a senior who is slightly annoyed: missing tests, behavioral bugs, accidental blast radius. Not naming.

The transcript is how the agent spent the afternoon. The diff is what you are shipping. If your tool cannot put the diff in front of you without a search, you do not have a parallel setup. You have chat rooms.

## A shape that works when two is actually right

Two tickets, no overlapping files, each with a test.

```bash
git worktree add ../tbo-auth -b agent/auth-cache
git worktree add ../tbo-flaky -b agent/ci-flaky
```

Session A: auth cache, allowlist for `go test ./internal/auth`. Session B: flaky CI, allowlist for the CI reproduction command. A two-line note: `auth-cache | in progress`, `ci-flaky | blocked on log from CI`. When A is green, you review `agent/auth-cache` in a third, empty context. You do not open B until A is either merged or parked.

That is parallelism. Three untitled Claudes in one repo is not.

## When this is the wrong move

Do not parallelize a task you could describe in one sentence. A typo, a log line, a rename — one agent, no plan file, no worktree.

Do not parallelize when you do not have a check. If "looks done" is the only signal, you become the test suite for every session at once. That job does not get cheaper with more sessions.

Do not add a fourth agent because the third one is blocked on you. Unblock the third one. The permission, the question, the missing log. Another body does not unstick a queue of which you are the only worker.

Do not parallelize because the demo looks like a company. [Start with a workflow](/blog/start-with-a-workflow).

## How we encode this

We kept hitting the same afternoon: real `opencode` sessions, real sub-agents, a human reconstructing the shift from scrollback.

theboringfloor is a native terminal app for that shift. Floor, roster, board, mail, and work threads are views of the same work. Permission asks stack instead of hijacking the screen. Isolation still belongs in git worktrees. The office replaces air-traffic control across untitled panes. It does not replace git.

```bash
theboringfloor --demo
```

Demo is a tour. Live attaches to a server you already have. The goal is not more agents. It is still being able to say, at 4pm, who is doing what.

## Further reading

- [You launched the subagents. Now you have to watch them.](/blog/watching-subagent-work)
- [Start with a workflow. Agents come later.](/blog/start-with-a-workflow)
- [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic
- [Agent swarms and the new model economics](https://cursor.com/blog/agent-swarm-model-economics) — Cursor, for the version of this problem at absurd scale
