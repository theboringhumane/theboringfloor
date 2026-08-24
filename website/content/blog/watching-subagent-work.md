---
title: "You launched the subagents. Now you have to watch them."
description: "Parallel reviewers and specialist workers are easy to spawn. The job that remains is supervision: permissions, diffs, and a handoff you would actually accept from a coworker."
date: "2026-08-24"
author: "theboringoffice team"
categories: ["AI Agents", "Engineering"]
featured: true
---

Spawning a specialist is the fun part. A review agent, a test runner, a security pass — each with its own prompt, each in its own context so they do not pollute the session that is still writing code.

Then they all come back at once.

One wants `npm test`. One wants to write `auth.go`. One finished a review you have not opened. One is blocked on a question that looks like a permission prompt but is actually a product decision. You are now the merge queue.

This is the part of "multi-agent" setups that blog posts skip, because it is not a clever prompt. It is operations.

## Generation got cheap. Attention did not.

A year ago the question was whether an agent could finish a non-trivial change. That bar moved. People now run nine parallel reviewers, or a writer and a reviewer in two windows, or a fleet of Claude Code subagents that each own a category of nits.

Output scales. Interrupts scale with it.

The failures we keep seeing — in our own usage, and in the writeups people publish — have the same shape:

**You cannot find the change.** It is in a transcript, somewhere, mixed with tool noise and a thinking block that auto-collapsed. You wanted a handoff. You got archaeology.

**Every permission is an all-hands meeting.** Four agents, four panes, four "allow this?". You stop being a programmer and start being a clicker.

**Done is silent.** An agent stalled while you were reading another one. You notice when the clock does, or when someone asks where the PR is.

**The writer grades its own exam.** The same context that produced the patch then "reviews" it, and of course the patch looks reasonable.

None of that is fixed by a better model. It is fixed by how the work is shown to you.

## Watch the diff, not the autobiography

An agent session is a diary. You rarely need the diary.

You need:

- the files that changed
- the test command that was run, and whether it passed
- anything still waiting on you
- a short account of *why*, if the why is not obvious from the diff

If your tool cannot produce that without you grepping the log, you do not have a team. You have chat rooms.

A work thread — diffs, tool calls, and thinking attached to *one* task — is the difference. Click the worker, see the work. Collapse the thinking. Keep the patch.

When you review, use a fresh context. Anthropic's Claude Code guidance is explicit about this: a reviewer that did not write the code will actually look for gaps. Tell it the requirements and the tests, and tell it not to report style. Then you decide. Chasing every finding from a reviewer that was told to "find issues" is how you get extra abstraction layers you did not ask for.

## Permissions should wait, not freeze the floor

You will not pre-approve everything. You should not.

You *will* type `y` on the tenth `git status` without reading it. That is the tell. Put those commands on an allowlist.

Leave the rest in a queue: allow once, allow always, reject. The rest of the agents keep working. A `chmod` on a deploy script can sit at 1 of 3 while the test runner finishes. That is the whole trick. Modal prompts make you the bottleneck the specialists were supposed to remove.

A useful extra: you can tell a permission from a *question*. "May I run this?" is a permission. "Should this be a breaking API change?" is a decision. If both arrive as the same popup, you will treat product calls like shell noise, or the reverse.

## Status has to live somewhere that is not prose

"I'll take the flaky test" in a chat message is gone the next time you scroll.

A board — even a crude one — is enough: claimed, blocked, done. If the only status is buried in replies, you will rebuild a project tracker in your head every time you sit down. That cost is why people say agents "don't save time" after a week of using them.

The same goes for coming back tomorrow. If the session dies and the next one starts from a blank slate, you paid for yesterday twice. Memory is not a slogan here. It is whether the board, the last chat, and the last decision are still there after `ctrl+q`.

## Do not add a tenth until you can see the nine

A practical bar, before you write another subagent prompt:

- Can you name who is blocked without polling windows?
- Can you review the last change without searching scrollback?
- Does one permission ask stop everyone else?
- Will tomorrow's you inherit any of this?

If the answer is no, the next specialist makes your afternoon worse. Fix the watching first. One implementer you can supervise beats four you babysit.

## What we wanted to look at

We wanted a shift we could understand at a glance: who is walking, who is waiting, what changed.

theboringoffice is that workplace in a terminal. Boss chat is a real `opencode` session. Sub-agents show up as employees on a floor. Work threads keep the diff with the worker. The permission queue is 1 of N, with `y` / `a` / `n`, and it does not kidnap the rest of the office. agentmemory keeps the board, the mail, and the last session across restarts.

You can take the tour without a live backend:

```bash
theboringoffice --demo
```

Or install and clock in:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh

theboringoffice
```

Subagents are a good idea. They are also how you accidentally hire a team you cannot manage. Build the watching on purpose, or the extra hands just mean extra noise.
