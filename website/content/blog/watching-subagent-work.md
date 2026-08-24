---
title: "You launched the subagents. Now you have to watch them."
description: "Spawning specialists is easy. The job that remains is supervision: diffs you can find, asks that wait instead of freeze, and a handoff you would accept from a coworker."
date: "2026-08-24"
author: "theboringoffice team"
categories: ["AI Agents", "Engineering"]
featured: true
---

Spawning a specialist is the fun part. A review agent, a test runner, a security pass — each with its own prompt, each in its own context so they do not pollute the session that is still writing code.

Then they all come back at once.

One wants `npm test`. One wants to write `auth.go`. One finished a review you have not opened. One is blocked on a box that looks like a permission and is actually a product decision. You are the merge queue. You are also the receptionist, the PM, and the person who was supposed to be writing the other ticket.

This is the part of "multi-agent" setups that launch posts skip, because it is not a clever prompt. It is operations. Generation got cheap. Attention did not. If you do not build the watching on purpose, the extra hands are extra noise with a cloud bill.

## Generation got cheap. Attention did not.

A year ago the question was whether an agent could finish a non-trivial change. That bar moved. People now run nine parallel reviewers, or a writer and a reviewer in two windows, or a fleet of Claude Code subagents that each own a category of nits.

Output scales. Interrupts scale with it. Anthropic's multi-agent research system used on the order of 15× the tokens of a chat, and they said out loud that the architecture only makes sense when the task is valuable enough to pay for that. A lint pass is not that task. A literature review might be. Your Tuesday ticket, maybe.

The failures we keep seeing — in our own usage, and in the writeups other people publish — have the same shape. They are worth naming, because unnamed they feel like "agents are overhyped" instead of "we cannot see the work."

**You cannot find the change.** It is in a transcript, mixed with tool noise and a thinking block that auto-collapsed. You wanted a handoff. You got archaeology.

**Every permission is an all-hands meeting.** Four agents, four panes, four "allow this?". You stop being a programmer and start being a clicker.

**Done is silent.** An agent stalled while you were reading another one. You notice when the clock does, or when someone asks where the PR is.

**The writer grades its own exam.** The same context that produced the patch then "reviews" it, and of course the patch looks reasonable.

None of that is fixed by a better model. It is fixed by how the work is shown to you — and by refusing to spawn the next specialist until the last one is inspectable.

## Watch the diff, not the autobiography

An agent session is a diary. You rarely need the diary. You need the files that changed, the test command that ran and whether it passed, anything still waiting on you, and a short *why* if the why is not obvious from the diff.

If your tool cannot produce that without you grepping the log, you do not have a team. You have chat rooms.

A work thread — diffs, tool calls, and thinking attached to *one* task — is the difference. Click the worker, see the work. Collapse the thinking. Keep the patch. Thinking is useful while the agent is stuck. It is residue once the patch exists. Tools that make residue the default view are tools for the model, not for you.

When you review, use a fresh context. Anthropic's Claude Code guidance is explicit: a reviewer that did not write the code will actually look for gaps. Tell it the requirements and the tests. Tell it not to report style. Then you decide.

A reviewer told only to "find issues" will find issues. That is its job in the same way a linter told to be angry will be angry. You will grow an extra abstraction layer because someone had to file a finding. Instruct the reviewer like a person you trust: behavioral bugs, missing tests, blast radius. You can bikeshed names yourself in the PR, in five minutes, without a specialist.

## Permissions should wait, not freeze the floor

You will not pre-approve everything. You should not. A sandbox is not an excuse to stop looking at `rm` and network.

You *will* type `y` on the tenth `git status` without reading it. That is the tell. Put those commands on an allowlist. Leave the rest in a queue: allow once, allow always, reject, park. The rest of the agents keep working. A `chmod` on a deploy script can sit at 1 of 3 while the test runner finishes. That is the whole trick.

Modal prompts make you the bottleneck the specialists were supposed to remove. If allowing a test in pane 2 blanks the writer in pane 1, you did not build a team. You built a mutex around your attention.

A useful extra, expanded in [A permission is not a question](/blog/a-permission-is-not-a-question): "May I run this?" is a permission. "Should this be a breaking API change?" is a decision. If both arrive as the same popup, you will treat product calls like shell noise, or the reverse. The mix-up is trained by the chrome, not by the model.

## Status has to live somewhere that is not prose

"I'll take the flaky test" in a chat message is gone the next time you scroll. A board — even a crude one — is enough: claimed, blocked, done. Names that match branches. If the only status is buried in replies, you will rebuild a project tracker in your head every time you sit down. That cost is why people say agents do not save time after a week of using them.

The same goes for coming back tomorrow. If the session dies and the next one starts from a blank slate, you paid for yesterday twice. Memory is not a slogan. It is whether the board, the last chat, and the last decision are still there after `ctrl+q`. Anthropic's [context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) piece is the same idea at the token layer: notes on disk, compaction, subagents that return summaries. At the human layer it is even simpler. Write the state down where a tired person can see it.

We have tried "the boss agent will keep the status." The boss agent writes a paragraph. The paragraph is slightly wrong. We still need the board. Use the model to *update* the board if you want. Do not use the model *as* the board.

## Do not add a tenth until you can see the nine

A practical bar, before you write another subagent prompt:

- Can you name who is blocked without polling windows?
- Can you review the last change without searching scrollback?
- Does one permission ask stop everyone else?
- Will tomorrow's you inherit any of this?

If the answer is no, the next specialist makes your afternoon worse. Fix the watching first. One implementer you can supervise beats four you babysit.

Anthropic's research orchestrator had to be *taught* not to spawn 50 workers for a simple query. Your prompts need the same budget: one worker for a fact, a few for a comparison, a fleet only when the question is actually wide. Coding is usually not wide. Coding is deep in two files. Depth wants a single context and a test, not a crowd.

## A shift that is actually supervisable

Writer in a worktree, tests on an allowlist, reviewer spawned *after* the diff exists, with the diff as the only input. Test runner is not a person. It is a command the writer runs. Security pass is a second fresh context on the same diff, told to look for authz and secret leaks, not style.

You look at: board (writer | review | done), permission queue if anything scary appeared, the reviewer's gap list. You do not look at four autobiographies.

That is a small team. It is also enough for most days.

## How we encode this

We wanted a shift we could understand at a glance: who is walking, who is waiting, what changed.

In theboringoffice, boss chat is a real `opencode` session. Sub-agents show up as employees. Work threads keep the diff with the worker. The permission queue is 1 of N and does not kidnap the rest of the office. agentmemory keeps the board, the mail, and the last session across restarts.

```bash
theboringoffice --demo
```

Subagents are a good idea. They are also how you accidentally hire a team you cannot manage. Build the watching on purpose.

## Further reading

- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents)
- [A permission is not a question](/blog/a-permission-is-not-a-question)
- [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) — Anthropic
- [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Anthropic
