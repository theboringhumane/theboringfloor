---
title: "The Permission Queue: control without constant interruption"
description: "A development update on turning agent permissions into clear, queued decisions instead of a stream of blocking prompts."
date: "2026-08-22"
author: "theboringoffice team"
categories: ["Office", "Engineering", "Updates"]
featured: true
---

Trust is not a setting you turn on once. It is a relationship between what an agent can do, what a human needs to see, and how quickly both can keep moving.

Most agent workflows fail one of two ways. They ask for approval so often that a developer becomes a full-time button clicker, or they stop asking until an avoidable surprise becomes a costly one. Neither is autonomy. Both are operational debt.

The Permission Queue is our answer in the current development build of theboringoffice.

## A queue, not an interruption

When permission requests stack up, the office shows the front request as `1 of N`. The human has a short, direct set of choices:

- `y` — allow once;
- `a` — allow always;
- `n` — reject;
- `esc` — dismiss for now and return later with `/perm`.

The important idea is structural. An agent asking a question should not turn the whole interface into a dead end. The request becomes visible, ordered work. The rest of the office remains legible.

## Decisions need context

A permission is rarely meaningful in isolation. The useful context is the work that led to it: the current task, the work thread, the diff or tool action, and the other questions waiting behind it.

That is why the Permission Queue lives alongside chat, agents, board, mail, activity, and the embedded terminal. It is not a standalone compliance product. It is part of the same shift.

Questions follow the same principle. The boss can open focused text, radio, checkbox, or confirmation prompts. A deferred question can be reopened with `/question`. The system asks for attention in a form that makes answering possible—not merely unavoidable.

## Built for the person who still has a job to do

A solo developer might be fixing a customer issue while an agent investigates a flaky test. A startup engineer might have three parallel tasks moving while preparing a release. In both cases, the product should protect focus instead of treating every agent event as equally urgent.

The Permission Queue is not a claim that the office can make every decision for you. It is a commitment to make the decisions that remain clear, recoverable, and proportionate.

That is what control should feel like in an agent workspace: less surveillance, fewer dead ends, and a human loop that can keep up with the work.
