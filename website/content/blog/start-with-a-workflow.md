---
title: "Start with a workflow. Agents come later."
description: "Most 'multi-agent' coding setups are a scripted path with extra avatars. Here is the ladder we use — and the afternoon that taught us to stop at step one."
date: "2026-08-24"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering"]
featured: true
---

Tuesday. Four subagents. A ticket that, in hindsight, was one function and a test.

We had a researcher to "survey the auth package," a writer to implement, a reviewer to nit, and a test runner because we had read that specialists should not share context. By 11:40 the researcher was still summarizing `middleware.go` into a document nobody would open. The writer had already patched the same file. The reviewer found a naming issue in a helper the writer had not finished. The test runner was blocked on `npm test` in a pane we had not looked at since 10:15.

We spent the lunch break merging four stories of the same hour. The PR that landed was 40 lines. The transcripts were 80 kilobytes.

That is the usual failure, and it is not a model failure. Claude, Codex, Gemini, and OpenCode will all write the function. What we keep doing is hiring a committee for a job that is a loop with a check.

Anthropic wrote this down in [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents) and then had to keep repeating it as the tooling got more exciting: the teams that ship use simple, composable patterns. You add a swarm when a single path actually forks. You do not add a swarm because the demo looks busy.

## Two words people collapse on purpose

**Workflow.** You decide the steps, or a small script does. The model fills a step. Outline, then patch, then tests. A gate can stop a bad intermediate: "the plan does not mention the public API; rewrite the plan." The path is known even if the tokens are not.

**Agent.** The model picks the next tool in a loop. You cannot honestly list the number of steps in advance. Useful when the work discovers itself — a flaky test that might be time, cache, or a race, and you will not know which probes matter until you have run two of them.

Most coding work is a workflow wearing an agent costume. "Write the function, run the tests, fix until green" is a loop with a check. The agent SDK you used to spawn four specialists did not change that. It added coordination tax: who owns the file, who is allowed to `git add`, whose review is binding, what happens when they disagree.

People collapse the words because "agent" is the budget line. "Workflow" sounds like last year. Last year is often the correct year.

## An afternoon that should have been a workflow

The ticket: cache the user object on the request so the handler does not hit the DB twice.

A workflow looks like this:

1. You (or the model, once) write a failing test that calls the handler twice and asserts one query.
2. The same session implements until the test is green.
3. A *fresh* session — new context, only the diff — reports gaps: did we cache a stale role? did we forget logout? You read the gaps. You do not read the diary.

That is prompt chaining plus a gate. It is boring. It lands before lunch.

The committee version looks like this:

- Researcher: "auth is implemented in several files…" (true, useless)
- Writer: edits `handler.go` and, for completeness, `middleware.go`
- Reviewer: wants a `UserCache` interface and a factory
- Test runner: still on the permission prompt for `go test`

You now have a design argument, two dirty files, and no green test. The extra "intelligence" went into the parts of the problem that were not the problem.

## The ladder (stop at the first rung that holds)

**Rung 1 — One session, one check.** Paste the ticket. Point at the test file. Allow the test command. Review the diff as if a coworker mailed a branch. If this completes the work, stop. Most tickets die here and should.

**Rung 2 — Chain, with a gate.** If the change is bigger than a screen, make the model write a plan you can reject. Only then implement. Then a clean-context review of the diff, instructed to ignore style and report missing tests. You are still in a workflow. The model is not electing a CEO.

**Rung 3 — Route by hardness.** A formatting pass or a docstring can be a small, cheap model. The gnarly refactor stays on a frontier model. *You* classify. If the classifier is another agent with a vague prompt, you have reinvented a manager who does not know the repo.

**Rung 4 — Specialists, in parallel, with isolation.** Only when you can already supervise one session — you can name what is in flight, what is blocked, what landed — and the files do not overlap. Git worktrees. Named sessions. A board that is not a chat message. This is [the parallelism post](/blog/running-multiple-coding-agents). It is not where you start a Tuesday.

If you cannot name the check, you are not ready for rung 4. You became the test suite. Four agents whose only signal is "looks done" is four ways to be confidently wrong at once.

## Teach the job like a ticket, not like a vibe

Anthropic's research-system writeup is blunt about this: early orchestrators spawned dozens of workers for simple queries, duplicated searches, and never stopped. The fix was not a smarter manager model. It was *task descriptions* — objective, output format, tools, boundaries — and an effort budget scaled to the query.

The same bug shows up in coding. "Research the auth system" is how two workers summarize the same `grep`. Compare:

> Look at `internal/auth` only. List the functions that read `User` from the DB. Do not propose a design. 30 lines max.

versus

> Help with auth.

The second one feels collaborative. It produces a staff meeting.

Scale effort in the prompt, in numbers, because models are bad at "how hard is this." Fact lookup: one worker, a handful of tool calls. A comparison of two approaches: two workers, then *you* pick. A fleet for a comma is how you burn the day and call it "agentic."

## What we tried that did not help

**A standing "manager" agent.** It summarized the other agents into a status paragraph that was always slightly wrong. We still had to open the diffs. We had added a translator.

**Shared working tree, "they'll coordinate."** They formatted each other's files. Coordination was a merge conflict at 4pm.

**Reviewer in the same context as the writer.** It praised the approach. Of course it did. It had just invented the approach.

**More specialists when someone was blocked on us.** The blocked one still needed `y` on a permission. The new one needed a worktree we had not made. We now had five blocked things.

The pattern: we kept adding roles because roles are easy to name. We avoided the check because the check is a test you have to write.

## When the loop (an actual agent) is the right move

Open-ended debugging. "Fails in CI, passes locally." You will not list the probes in advance. A model with a shell, in a loop, with you watching the commands, is the right shape. A committee is not — they will each pick a different theory and write a different patch.

Unknown blast radius. A refactor that might touch twelve files or two. An orchestrator that *discovers* the files can be worth it. Anthropic called this out as the case for orchestrator-workers. The key word is discover. If you already know it is `handler.go` and `handler_test.go`, you do not need discovery. You need a patch.

Long-running migration where the context window will fill with tool junk. Then you want compaction, notes on disk, maybe a subagent that returns a summary instead of a novel. That is [context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents), not a headcount problem.

## When a swarm is theatre

A rename, a log line, a one-file bug. One session. No plan file. No worktree.

A task you cannot grade. If the only rubric is vibes, extra agents multiply vibes.

A demo that only looks good with four walking avatars. If the UI needs a crowd to feel like work, the UI is selling you a crowd. [If the picture can lie, you will believe it](/blog/agents-can-sign-up).

A manager you cannot fire. If the product spawns workers you did not ask for and will not show you the diffs without a tour, you do not have a workflow. You have a screensaver with an API bill.

## How we encode this

theboringfloor assumes a boss session — the workflow you are already in — and employees as *task* sub-agents you can see. The floor is for the moment you have actually forked work, not for making a one-line fix look like a company. It will not stop you from spawning too many. The ladder will.

```bash
theboringfloor --demo
```

Demo is labeled. When you are done touring, attach the server you already run. The product is a place to supervise a shift. It is not a reason to hire a shift.

## Further reading

- [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic
- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents)
- [You launched the subagents. Now you have to watch them.](/blog/watching-subagent-work)
- [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Anthropic
