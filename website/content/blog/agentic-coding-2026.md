---
title: "The code got cheap this year. The supervision didn't."
description: "Coding agents went from autocomplete to async workers you hand tickets to, and the bottleneck moved from model quality to supervision — what actually changed in 2026, and how to stay oriented when several run at once."
date: "2026-08-29"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering", "Office"]
featured: true
---

Most Tuesdays look like this now. You assign an issue to an agent from your phone while the kettle boils. A cloud agent is already chewing a backlog ticket in a sandbox somewhere. In your terminal, two more are mid-task, each in its own worktree. By lunch there will be four pull requests with your name on the review line.

None of that is remarkable anymore. That is the remarkable part.

Two years ago the argument was about whether a model could write code at all. That argument is over. The AI coding agents themselves — Claude Code, Codex, Copilot, Jules, OpenCode — all write plausible code, and the gap between them matters less every quarter. What did not get cheaper is knowing what happened. Four agents means four streams of work, and every stream ends at the same place: you, reading a diff, deciding whether it should exist.

This is what actually changed about agentic coding in 2026 — not the models, but the shape of the work around them — and the habits that keep a multi-agent afternoon from becoming a reconstruction bill.

## Autocomplete became delegation

The unit of work changed. You used to accept a completion. Now you hand over a ticket.

Simon Willison coined the cleanest label for the new shape: [async coding agents](https://simonwillison.net/tags/async-coding-agents/) — tools you set a task that "check out and iterate on code in their own hosted environment and submit the proposed change as a pull request." In 2026 that is no longer a category. It is the default. Every major vendor ships it:

- **GitHub's Copilot coding agent** went [generally available last September](https://github.blog/changelog/2025-09-25-copilot-coding-agent-is-now-generally-available/). Assign an issue, it works in a GitHub Actions environment, you get a draft PR.
- **OpenAI's Codex** runs each task in its own sandboxed container and comes back with terminal logs and test output you can audit.
- **Google's Jules** is async-first: it writes a plan you approve before any code exists, then works on a cloud VM and returns a PR.
- **Claude Code on the web** runs the same play from a browser — describe the task, walk away, review the PR.

The autocomplete that finishes your line still exists. It is just not where the growth is. Anthropic's [2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report) puts it from the buyer's side: "software development is shifting from writing code to orchestrating agents that write code" — with case studies from Rakuten, CRED, TELUS, and Zapier, not toy repos.

The loop inverted, and that is the part people miss. Pairing means the agent waits on you. Delegation means you wait on the agent. Once you are the one waiting, starting a second one stops feeling like multitasking and starts feeling obvious. Which is how the next thing happened.

## One agent became a roster

The second change is quieter than the first: coordination primitives went native.

Inside one session, subagents give you focused workers with their own context windows whose results come back summarized to the caller. Claude Code's experimental [agent teams](https://code.claude.com/docs/en/agent-teams) go further: teammates are independent instances that claim work off a shared task list, message each other directly instead of routing through the lead, and unblock dependent tasks when they finish. Anthropic's own docs carry the honest caveat, and it is worth quoting in spirit — each teammate is a separate Claude instance, so a team costs meaningfully more tokens than one session with subagents. Multi-agent is not free parallelism. It is a trade.

Addy Osmani's [Code Agent Orchestra](https://addyosmani.com/blog/code-agent-orchestra/) is the best map of the moment we have read. His name for the shift: conductor to orchestrator. He sorts the 2026 landscape into three tiers — in-process subagents and teams, local orchestrators that run several agents in isolated worktrees on your machine, and the cloud async agents from the last section — and notes that most working developers end up using all three, sometimes in the same day. Anthropic's report names the same trend from the enterprise side: organizations moving from a single coding assistant to coordinated groups of specialized agents working in parallel.

Two things we keep relearning, and both are in those sources:

**Fewer, scoped agents beat a crowd.** Three agents with real file ownership outperform six sharing a checkout. Osmani puts the working limit at three to five — roughly the number you can meaningfully review. We have never seen a fourth pane beat a second worktree.

**Rosters are for parallel work, not hard work.** A task that needs one long chain of reasoning gets nothing from being split across five context windows. Multi-agent workflows pay when the work has parts that genuinely run independently — or intermediate output that would bury the main thread. Splitting a sequential task just gives you sequential work with a coordination tax.

## The bottleneck moved to review

Generation got fast. Verification did not. The queue now ends at you, and it is longer than it used to be.

The quote of the year belongs to Morgan Stanley's Khalid Elsawaf, in a [Moderne writeup on the review bottleneck](https://moderne.ai/blog/ai-didnt-break-coding-it-broke-code-review): "In the time that Dov and I have been speaking, you probably could have produced a thousand-file, 10,000-line PR on the back of just a simple prompt. No human here is going to review that." An [arXiv position paper from June](https://arxiv.org/html/2606.13175v1) makes the structural version of the argument: AI generation plus mandatory human review is not a stable endpoint, because reviews of agent-written code degrade into rubber stamps — the tests pass, the code looks right, and the cognitive cost of real scrutiny is prohibitive.

Osmani says it in one line: the bottleneck is no longer generation, it is verification. David Poll [pushes the point further](https://www.davidpoll.com/2026/02/code-review-is-not-about-catching-bugs/): review is shifting from "does this work" to intent and taste — whether the change should exist at all. Nobody has automated that part, and the people who claim otherwise are selling something.

On our floor the failure mode has a name: the approved-without-reading PR. You cannot fix it with a faster model. The fixes that work are boring:

**Checks the agent runs before you look.** Tests, lint, a reproduction command. If "looks done" is the only signal, you are the test suite — and that job does not get cheaper with more agents. It gets more expensive, linearly.

**Review in a context that did not write the code.** A fresh session, or a second agent with a clean window, told exactly what done means: which tests must pass, which files should not have changed, which edge case you care about. Ask it for gaps, not style. The session that wrote the code will defend the code.

**Diffs you can find without archaeology.** The transcript is how the agent spent the afternoon. The diff is what you are shipping. Any setup that cannot put the diff in front of you in one keystroke is not a parallel setup. It is chat rooms.

## A vague prompt is now a fleet-wide bug

When one agent misreads a vague instruction, you lose twenty minutes. When four misread it, you lose the afternoon in four directions at once.

This is the compounding nobody demos. Osmani calls it directly: the spec is the leverage. A vague spec does not produce one vague result — it multiplies through every parallel run, each agent going slightly wrong in a slightly different direction, and then you get to do four reviews on four wrong things. Precision compounds the other way. Clear boundaries, integration points, edge cases, and invariants multiply into implementations that agree with each other.

Write the ticket like the reader cannot ask a follow-up — because with async agents, it either cannot, or it will burn an hour guessing. Every brief, whether it goes to a terminal agent or a cloud VM, needs the same three things: what done means, what must not change, and what to do when it gets stuck.

The same discipline applies to the file every agent reads first. Osmani's roundup cites research (Gloaguen et al., ETH Zurich) finding that LLM-generated `AGENTS.md` files offer no measurable benefit and can marginally reduce success rates, while human-written ones help. Keep yours short and human: style, gotchas, architecture decisions, test strategy. Never let an agent edit it unsupervised. The file that steers the fleet is the last file to hand to the fleet.

## Run it like ops, not like a demo

The multi-agent setups that survive a full week are run like operations, not like a launch video.

Osmani names the loop — plan, spawn, monitor, verify, integrate, retro — and calls it the factory model. Our version, learned the expensive way:

**Plan** the split before you spawn anything. If you cannot say which files each agent owns, you do not have a plan. You have a merge conflict with a waiting room.

**Spawn into isolation.** Worktrees, still. Nothing has replaced them, and [the habits post](/blog/running-multiple-coding-agents) has the full version.

```bash
git worktree add ../floor-auth -b agent/auth-cache
git worktree add ../floor-billing -b agent/invoice-retry
```

**Monitor on a cadence, not a hover.** Check in every ten minutes. Kill anything stuck three iterations on the same error and reassign it to a fresh context. Hovering is how you end up watching thinking blocks all afternoon.

**Verify** with the checklist from the review section, every time, before merge. No green-checkmark exceptions for agents you like.

**Integrate** branches the way you would review a coworker's — `git diff main`, the tests, the files that should not have moved.

**Retro** into `AGENTS.md` when a session teaches you something. Compound learning only compounds if you write it down.

The skills that matter in 2026 are decomposition, writing checks, and reading diffs. Code is abundant. Judgment is the scarce resource, and it does not scale with your pane count.

## When this is the wrong move

Do not delegate a task you can describe in one sentence. A typo, a log line, a rename — one agent, no ticket, no worktree.

Do not run a fleet without a check. If the only signal is "it said it finished," every agent you add makes you slower.

Do not add agents because the current ones are blocked on you. You are the queue. Unblock the queue.

Do not adopt orchestration before one agent finishes cleanly. If you cannot say what a single session did this morning, five sessions will not make it clearer.

And do not confuse motion with progress. Four green PRs nobody has read are not four features. They are four obligations with branch names.

## How we encode this

We kept living this exact afternoon — real sessions, real sub-agents, a human reconstructing the shift from scrollback — so we built the floor we wanted to stand on.

theboringfloor is a native terminal app for running several coding agents at once. The roster shows who is working and who is waiting on you. Permission asks stack in one queue instead of hijacking four windows. Work threads keep the diff attached to the task, so review starts at the diff instead of the transcript. Isolation still belongs to git worktrees — the office replaces air-traffic control across untitled panes, not git.

```bash
theboringfloor --demo
```

The goal was never more agents. It is still being able to say, at 4pm, who did what.

## Further reading

- [2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report) — Anthropic's eight trends, with case studies from Rakuten, CRED, TELUS, and Zapier
- [Orchestrate teams of Claude Code sessions](https://code.claude.com/docs/en/agent-teams) — the agent teams docs, including the honest token-cost trade-off
- [The Code Agent Orchestra](https://addyosmani.com/blog/code-agent-orchestra/) — Addy Osmani's patterns talk: the three tiers, the factory loop, the verification bottleneck
- [Copilot coding agent is now generally available](https://github.blog/changelog/2025-09-25-copilot-coding-agent-is-now-generally-available/) — GitHub's changelog for the assign-an-issue, get-a-PR flow
- [async-coding-agents](https://simonwillison.net/tags/async-coding-agents/) — Simon Willison's running taxonomy of the delegate-and-review category
- [AI didn't break coding. It broke code review.](https://moderne.ai/blog/ai-didnt-break-coding-it-broke-code-review) — Moderne, with the Morgan Stanley quote
- [The End of Code Review](https://arxiv.org/html/2606.13175v1) — the arXiv argument that generation plus rubber-stamp review is not a stable endpoint
- [Code review is not about catching bugs](https://www.davidpoll.com/2026/02/code-review-is-not-about-catching-bugs/) — David Poll on review shifting to intent and taste
- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents) — our earlier post on worktrees, permissions, and the reconstruction bill
