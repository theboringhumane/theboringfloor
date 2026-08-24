---
title: "Subagents are easy. Watching their work is the hard part"
description: "Parallel review agents and specialist subagents are everywhere in 2026. The bottleneck moved from generation to supervision — permissions, diffs, and handoffs."
date: "2026-08-24"
author: "theboringoffice team"
categories: ["AI Agents", "Engineering"]
featured: true
---

A year ago, the hard question was whether an agent could finish a non-trivial change. In 2026 the harder question shows up in every serious workflow post: once you have **many** agents — reviewers, implementers, security checkers, test runners — who watches them?

People are publishing elaborate Claude Code subagent fleets for parallel code review. Others are wiring specialist workers for tests, style, and deployment safety. The generation side is getting cheaper. The supervision side is getting noisier.

## Where the bottleneck moved

Search Reddit and engineering blogs for coding agents and you will see the same complaints under different names:

- **Permission fatigue** — allow this shell command, allow that write, allow the next network call. One agent is manageable. Four agents fighting for your attention is not.
- **Diff soup** — the change exists somewhere in a transcript. Finding *which* agent produced *which* hunk is archaeology.
- **Handoff failure** — the implementer finished, the reviewer has notes, and you still do not have a single place that feels like a coworker briefing.
- **Silent success / silent failure** — an agent stalled while you were reading another pane. You only notice when the clock does.

These are not model problems. They are **operations** problems.

## Specialists scale output. They also scale interrupts.

Parallel subagents are a good idea. Security review should not wait on style nits. Tests should not wait on prose. The mistake is assuming that "launch nine agents" includes a human-usable merge of their work.

Without structure, each specialist becomes another interrupt channel:

1. Agent A wants `npm test`
2. Agent B wants a write to `auth.go`
3. Agent C finished a review nobody asked you to open yet
4. Agent D is blocked on a question that looks like a permission prompt but is really a product decision

If every interrupt is modal, you become the serial bottleneck the agents were supposed to remove.

## What "watching the work" should feel like

A usable multi-agent office has a few non-negotiables:

### 1. Permissions wait their turn

Asks should stack as a queue — allow once, allow always, reject — without stealing the whole screen every time. The rest of the floor keeps moving.

### 2. Diffs stay attached to the worker

A work thread is not a log dump. It is the story of one piece of work: tool calls, thinking you can collapse, and the patch you will actually review.

### 3. The board is the source of truth

Tickets claimed, tickets blocked, tickets done. If status only lives in prose replies, you will rebuild a project tracker in your head every morning.

### 4. Shoulder taps are rare and specific

Notify when you are away and something needs a human call. Stay quiet while you are already looking. Attention is the scarce resource.

## Why we built an office instead of another agent launcher

theboringoffice assumes the agents already exist — starting with `opencode` as the boss and sub-agents as employees. The product job is to make their shift legible:

- walk the floor and see motion tied to real events
- open a work thread the way you would ask a coworker what changed
- answer the permission queue without pausing every other desk
- come back tomorrow to a team that still remembers yesterday

You can tour that shape without wiring production:

```bash
theboringoffice --demo
```

Or install and run live:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh

theboringoffice
```

## A short checklist before you add another subagent

- Do you know who is blocked without polling panes?
- Can you review the last change without searching scrollback?
- Will a permission ask serialize the whole fleet?
- Will tomorrow's you inherit today's context?

If any answer is no, do not add a tenth specialist yet. Fix how you watch the nine you already have.

Subagents made coding agents productive. Supervision is what makes them trustworthy. Build for the second problem on purpose.
