---
title: "Memory is whether tomorrow inherits today"
description: "A capable agent without a record is a contractor who forgets the meeting. Continuity is inspectable state — notes, boards, session IDs — not a model that 'improves itself' in secret."
date: "2026-08-19"
author: "theboringfloor team"
categories: ["Engineering", "AI Agents"]
featured: true
---

Friday, 6:40pm. The agent finally has the shape of the billing package. It knows why the webhook retries, which test is a liar, which comment in `invoice.go` is load-bearing. You quit the terminal. You have a train.

Monday, 9:10am. New session. Blank prompt. You paste a shorter version of the ticket. The agent rediscovers the webhook, proposes a retry policy you rejected on Friday, and formats a file you already formatted. You have paid for Friday again, in tokens and in the hour it takes to get angry.

A capable agent without memory is impressive in the moment and expensive by the week. It is not a team. It is a series of amnesiac contractors who interview well.

The change that matters is not a system that rewrites its own rules overnight and calls that "self-improving." It is a system that lets work compound *in public* — where a tired human can see what was decided, what is still open, and what must not be tried again.

## A shift leaves more than a diff

The PR is the output. The shift also leaves:

- the conversation that framed the task (why we are not using the vendor SDK)
- decisions and constraints (timeouts, the flag name, "don't touch billing export")
- work still blocked on a person (legal, a screenshot, a prod log)
- results another agent can use (the failing test, the repro command)
- lessons worth carrying (the liar test, the comment in `invoice.go`)

If the only artifact is a PR, you lost the *why*. If the only artifact is a chat log, you lost the *state*. Chat logs are novels. State is a table. You want both, and you want the table in a place you can open without grepping a transcript for the word "decided."

Anthropic's [context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) note is the token-side of this: the window is a budget, compaction and notes on disk beat stuffing every tool dump into the next prompt. The human-side is even less mystical. Write it down where Monday can find it.

## Inspectable or it did not happen

"Self-improving" is a dangerous phrase when it means the policy changed and nobody can say how.

We have seen tools that "learn your codebase" and cannot show the snippet they learned. We have seen lesson stores that cannot delete. We have seen memory that is a vector blob and a shrug. When it goes wrong — and it will, because retrieved notes go stale — you cannot argue with a shrug.

Prefer a smaller contract: preserve the evidence, show the current state, let people decide what becomes policy.

A board you can read. Mail you can open. A session ID you can pass at launch. Lessons you can list and delete. If the memory cannot be audited by the same person who owns the repo, it is not memory. It is mood.

Small teams do not have an ops desk to untangle a mysterious agent history. Continuity has to be cheap. Inspection has to be normal. "Trust the model" is not a backup.

## Restarting should not erase the room

Default: last chat comes back. This is unglamorous and it is the difference between a workplace and a demo.

When you need a particular room, not the latest one:

```bash
theboringfloor --session <your-session-id>
```

If `ctrl+q` means a blank prompt, you paid for yesterday twice. If the product sells "memory" and still greets you with an empty composer, the marketing ate the feature.

Config should be a file you can `cat`, not only a GUI. `--print-default-config` exists because we have lost hours to hidden JSON. Hidden state is how "the office forgot" becomes un-debuggable.

## What to persist, what to throw away

Persist: decisions, constraints, the board, the mail that is a handoff, the session you might reopen, lessons that are *short and named*.

Throw away: raw tool dumps, the fifth identical `git status`, thinking blocks, embeddings of secrets, the novel the researcher wrote about the package you already know.

Compaction is a feature. Cursor's swarm "Field Guide" experiment — a line-budgeted folder the agents curate for their successors — is the same idea with a budget. Unlimited memory is a junk drawer. Junk drawers do not make Monday smarter. They make retrieval worse. Context rot is not a metaphor; even the lab posts treat the window as a finite attention budget.

## When not to persist

Do not keep secrets in the same store as lessons. API keys in "memory" are a breach with extra steps. If the agent saw a `.env`, that is an incident, not a note.

Do not persist every retrieval as truth. Show notes as notes. Stale "we don't support SSO" will ship a wrong answer with confidence.

Do not persist on behalf of a customer without a story for export and delete. Memory that cannot leave is lock-in dressed as intelligence.

## How we encode this

theboringfloor uses agentmemory for the layer under the floor: sessions restore, the board is actions, mail is signals, recall and lessons are optional inheritance for the next shift. The office shows the work. Memory keeps what a future session should be able to find. We would rather be boring and inspectable than magical and wrong.

If a lesson is bad, delete it. If a session is done, you should still be able to open it. That is the whole product bet, under the sprites.

## Further reading

- [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Anthropic
- [Don't let approvals steal the screen](/blog/tool-router-beta)
- [A small team's operating system for coding agents](/blog/field-guide-great-tools)
- [Shell + Skills + Compaction](https://developers.openai.com/blog/skills-shell-tips) — OpenAI, on not stuffing procedures into the system prompt
