---
title: "Don't let approvals steal the screen"
description: "Four agents, four 'allow this?', you are a clicker. Here is why modal prompts serialize a parallel shift — and the queue habits that give attention back without going full yes-to-all."
date: "2026-08-22"
author: "theboringfloor team"
categories: ["Office", "Engineering"]
featured: true
---

You type `y` without reading. That is the tell.

The tenth `git status` trained you. Now a `chmod` on a deploy script rides the same muscle. Or the inverse: you are "being responsible," so every `ls` is a meeting, and four other sessions sit idle while you write a paragraph about listing files.

Trust is not a toggle in a settings page. It is which commands you have already decided, which still deserve a human, and whether that human is allowed to be a person with another window open. Most agent UIs get the first part (a prompt) and skip the second (the rest of the shift). The prompt becomes the product. Everything else waits.

We did not set out to build a "permission system." We set out to stop losing the afternoon to the frontmost modal.

## Two ways the day dies

**Clicker.** Everything is modal. Four agents, four "allow this?", you are no longer programming. You are a very expensive interrupt handler. Parallelism on the metal, serialism in your neck.

**Surprise.** Someone turned prompts off, or pressed allow-always on a pattern they could not state. A network call goes out. A force-push. A `rm` that matched `*` because the glob was "the repo." Friday. You find out from Slack.

Neither is autonomy. Both are ops debt. The industry oscillates between them because the honest middle — a queue with an allowlist you could explain in a code review — is less exciting to demo than either "full auto" or "human in the loop" as a slogan.

Slogans are not a UX. A UX is what happens when the third agent asks while you are reading the first agent's diff.

## Why modal is a trap at n>1

With one agent, a blocking prompt is almost honest. You were looking at that pane anyway. The cost is a keystroke.

With three agents, a blocking prompt in pane 2 is a halt in a system you cannot see. Pane 1's writer may still be generating. Pane 3's tests may have finished. You will not know, because the modal ate the keyboard, or because you context-switched to dismiss it and forgot pane 3 existed.

This is the same lesson Anthropic's research agents hit in production: long-running, stateful, errors compound, you cannot restart from zero. A permission that freezes the world is a tiny outage. Tiny outages, stacked, are the day.

The structural fix is dull. An ask is work. It has an owner (you), a payload (the command), a place in line. The rest of the agents keep moving. You handle the front of the queue when you have a slice of attention, not when the toolkit panics.

## A queue is a list, not a vibe

Useful answers are short:

- allow once
- allow always — for a *pattern you can say out loud*
- reject
- dismiss and come back from a list

If there is no list, dismiss is amnesia. `/perm`, `1 of N`, a sidebar of parked asks — pick one. We use `1 of N` on the front and `/perm` to reopen what we `esc`'d. The numbers matter. "Some permissions" is how the `chmod` sits for twenty minutes.

Order is FIFO unless you are doing something clever. You are not doing something clever. Do not let the loudest agent jump the queue by redrawing a modal on top.

## What belongs on the allowlist

Promote the commands you already do not read. That sentence is the whole policy.

Usually: `git status`, `git diff`, the formatter, the test binary for *this* repo, maybe `ls` and `rg` if you are not in a tree that can see secrets.

Usually not: network, `chmod`, `rm`, anything under `scripts/`, anything that writes outside the worktree, package publish, docker to a remote, cloud CLIs.

"Allow always" should be something you could paste into a PR description: `go test ./...`. If you cannot paste it, you do not understand it, and "always" is a gun.

Revisit the list when you catch yourself reading a prompt again. That means the list is too tight, or the agent is asking for a new class of thing. Either way, a human should update policy, not a tired `y`.

## The scary ones should be *more* visible, not more blocking

A `chmod` on `scripts/deploy.sh` should be large: full command, path, which agent, which task. It should still be in the queue. The test runner should not stop because you have not decided about deploy yet.

We used to think "important = modal." Important = hard to miss *when you look*. Modal is "you must look now," which is how unimportant `git status` trains you to dismiss important `chmod`.

If everything is urgent, nothing is. That is not a poster. That is a permission UI with one component.

## Built for the person who still has a job

A solo developer is fixing a customer issue while an agent investigates a flaky test. A startup engineer has three tasks moving and a release notes file open. In both cases the product should protect focus.

That does not mean hiding asks. It means asks that wait, a board that shows blocked-on-you, and a keyboard that still types in chat while something sits at 2 of 2.

The queue is not a claim that the office can make every decision. It is a claim that the decisions that remain can be clear, recoverable, and proportionate — and that they will not kidnap the shift.

## When a queue is the wrong move

One agent, one trusted directory, five-minute rename: approve in place. A queue UI is a second product.

Unattended overnight: a queue that waits on you is a halt. Tight allowlist, sandbox, or do not leave it running. A parked `chmod` at 3am is not safety. It is a stuck worker you will forget.

If you cannot name what "allow always" matches, do not press it.

If the tool will not show you the command in full, do not allow it. Truncated `rm -rf /usr/lo…` is a horror movie, not a UX.

## How we encode this

In theboringfloor, permission asks stack. The front of the queue shows `1 of N`. `y` / `a` / `n` / `esc`. Come back with `/perm`. Questions are a different popover — [because they are not permissions](/blog/a-permission-is-not-a-question). The office stays legible while something waits.

```bash
theboringfloor --demo
```

Watch whether a second agent's work continues while an ask sits. If it does not, the demo is showing you a screensaver. Demand the queue of your tools, including ours.

## Further reading

- [A permission is not a question](/blog/a-permission-is-not-a-question)
- [You launched the subagents. Now you have to watch them.](/blog/watching-subagent-work)
- [A postmortem of three recent issues](https://www.anthropic.com/engineering/a-postmortem-of-three-recent-issues) — Anthropic, on how overlapping failures confuse operators (your `y` finger is an operator)
