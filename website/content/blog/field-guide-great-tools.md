---
title: "A small team's operating system for coding agents"
description: "Attention is the scarce resource, not tokens. Four conditions that let a solo developer get more attempts from agents without becoming their full-time supervisor — and when the OS is overhead."
date: "2026-08-21"
author: "theboringoffice team"
categories: ["AI Agents", "Office"]
featured: true
---

The promise is not that a two-person startup becomes a two-hundred-person company. We have sat through those decks. They are about headcount you will not have and dashboards you will not staff.

The honest promise is smaller: more attempts, more parallel exploration, less time on mechanical work — *if* you can still understand what is happening.

For a solo developer, the scarce resource is not tokens. Tokens got cheap in the same year attention did not. Every extra terminal, every hidden retry, every decision that arrives without context spends attention. A good setup gives some of it back. A bad setup is a second job titled "babysit the robots."

This is an operating system in the old sense: the thing that schedules work, interrupts, and memory so a person can still be a person. Not a kernel. A set of conditions. If your tools fail them, the tools are toys.

## 1. Make the work visible before you automate more of it

Do not start with "how many agents." Start with whether you can answer, in a few seconds, without a hunt:

1. What is being worked on?
2. What is blocked?
3. What changed?

If the answer lives in four scrollbacks, you do not have leverage. You have archaeology. We have all done the archaeology. It feels like work. It is not the work you hired the model for.

Peripheral vision exists so you do not reconstruct the shift later: a roster, a thread per task, an activity list, a floor if you like sprites. The metaphor is optional. The mapping is not. An avatar walking should mean a task was dispatched. If it means a timer fired, you are watching a screensaver. [We wrote about that lie](/blog/agents-can-sign-up).

Visibility before automation sounds conservative. It is the opposite. You cannot automate what you cannot see. You can only add noise.

## 2. Queue decisions; do not queue the whole shift

An agent will need a human. Permissions, product forks, a missing screenshot. The wrong response is to freeze everyone until that human returns from coffee.

Ordered asks: allow once, allow always for a pattern you can name, reject, come back later from a list. Human attention becomes an input you schedule, not a background polling job.

The clicker failure mode is real and it shows up fast. Three agents, three modals, you are a receptionist. The surprise failure mode is the hangover of "just allow all." [Queue the scary ones](/blog/tool-router-beta). [Do not put product questions in the same box](/blog/a-permission-is-not-a-question).

If your "human in the loop" cannot go to the bathroom without stalling the loop, you did not build a loop. You built a leash.

## 3. Treat context as infrastructure

A productive system does not go blank when the terminal quits. It remembers what happened, what was decided, and where work stopped.

Session restore. A board that is not a chat message. Mail that is not a collapsed thinking block. Lessons you can delete. If Monday cannot inherit Friday, you are renting contractors by the hour and paying onboarding every morning.

[Memory is whether tomorrow inherits today](/blog/self-improving-ai-system) — not whether the vendor said "self-improving." Inspectable state beats a vector store you cannot argue with.

Anthropic treats context as a finite budget. You should treat *your* context the same way. Notes, not novels. Boards, not vibes.

## 4. Lower the cost of the first honest experiment

Adoption should not start as an integration project. It should start as a look at the thing.

A labeled demo, then attach to a server you already run. Small teams do not need a mandate from a platform committee. They need a reversible way to learn whether a tool earns a place in the day.

If the trial requires a sales engineer, SSO, and a week of "onboarding the agents," you will never know if it worked. You will only know you spent the week. Honest experiment: one ticket, one live session, watch whether the UI updates when the *tools* update.

`--demo` that is unlabeled is not an experiment. It is an ad. Demand the label. Then demand live.

## When this OS is overhead

One agent, one ticket, tests in the loop: a checklist in the PR is enough. Do not install a workplace for a rename. [Start with a workflow](/blog/start-with-a-workflow).

If you cannot spare the attention to *review* parallel work, do not spawn parallel work. Serial and supervised beats parallel and ignored. Ignored agents still write files.

If your company already has a board, a review tool, and a chat, and they are actually used, do not add a fourth status system. Attach. Duplicate state is how Monday disagrees with itself.

If you wanted an autonomous company: we do not have one, and we do not think you want the one in the deck.

## How we encode this

theboringoffice is that OS in a terminal: live opencode events as a floor, a permission queue, agentmemory under board and mail, `--demo` before you trust live mode.

```bash
theboringoffice --demo
```

The best setup is not the one with the most moving parts. It is the one that gives you more attempts while preserving the ability to see, decide, and remember. If we add a moving part that does not serve those three verbs, it is decoration. Delete it.

## Further reading

- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents)
- [Start with a workflow. Agents come later.](/blog/start-with-a-workflow)
- [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic
- [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Anthropic
