---
title: "If the UI can lie, you will believe it"
description: "Agent workspaces fail when the picture is prettier than the log. A test: every animation should map to an event you could write to JSONL — and a labeled demo is the only honest fake."
date: "2026-08-23"
author: "theboringoffice team"
categories: ["AI Agents", "Updates"]
featured: false
---

You open a new "agent office." Someone is walking to a desk. A badge says *Working*. A feed says *Deploying the future*. A progress ring is 70% of the way to a sentence that does not exist in the repo.

None of that happened. The demo seeded a story. You nod because the story is the one you wanted: a room full of coworkers, you as the calm manager, work happening at a human tempo you can see.

That is how these tools get adopted and then abandoned. The first session felt like a workplace. The second session — live, attached to your actual model — was a chat with extra CSS. The walkers had been a timer. The feed had been copy. You cannot go back to believing it, and you cannot recommend it, and you waste the next quarter explaining why "we tried agents" meant "we tried a trailer."

We care about this more than is polite, because we shipped sprites on purpose. Sprites that lie would make us a trailer with a CLI. This post is the test we use on ourselves, and the test we think you should use on everyone else.

## Demand a mapping, not a metaphor

A useful agent UI is a projection of a log. Not a movie that happens to share a brand with a log.

An agent walking to a desk means a task was dispatched — a real child session, a real tool call, a real worktree, something you could find in a trace. Typing means tokens or tool events are arriving *now*. Mail means a result came back to a place you can open. A red badge means a human decision is queued, with a command or a question attached.

If you cannot name the event behind the motion, the motion is decoration. Decoration trains you to trust a picture that the next live run will not match. You will wait for the walker that means "tests passed" and it will mean "we played the idle animation."

The test we use internally: could you rebuild the same screen from a JSONL of events, with a `DEMO` flag as the only lie? If the floor cannot be a function of the log, the floor is a cartoon. Cartoons are fine in a trailer. They are poison in an ops tool.

This is the same standard as a metrics dashboard. If CPU is 12% on the graph and 90% on the box, you throw out the graph. Agent UIs get a pass they have not earned because the cartoon is charming. Charm is not a calibration.

## Demo is a tour. Label it.

Simulated events are fine. You should be able to learn the keyboard without burning API budget, without a live model, on a plane.

Unlabeled simulated events are fraud — including to yourself, three weeks later, when you cannot remember whether the floor was real. You will make architecture decisions on a memory of walkers.

A honest trial has three steps.

**1. Run the labeled tour.** Learn `tab`, the queue, where the diff lives. See the shapes. The banner should say demo in a font you cannot ignore. If you have to hunt for the word, they did not want you to find it.

**2. Attach to a real session you already have.** Claude Code, OpenCode, Codex — whatever you actually use. Not their hosted golden path only. If the product cannot attach, it wants to own the runtime. Maybe that is fine for you. Know it.

**3. Do one small ticket.** Watch whether the UI updates when the *tools* update, not when a timer fires. `git diff` appears in a thread. A permission is the command that was actually proposed. Done is the test that actually ran.

If step 3 needs a sales engineer, the workspace is not for a two-person team. If step 3 still looks like step 1, you never left the trailer.

## Failure modes we have seen (and done)

**The busy office.** Four avatars, always walking, even when the session is idle. Motion as proof of value. You will feel productive. The repo will not.

**The delayed truth.** Events batch and then animate in a pretty order. The pretty order is not the causal order. You will debug a race that did not happen, because the UI serialized it for the camera.

**The collapsed handoff.** "Agent finished." No diff. The diff is in a transcript two clicks away that did not update. Finished is a status. A handoff is files.

**The unlabeled mix.** Live tokens, seeded walkers. Half real. You cannot tell which half. This is worse than a full demo. At least a full demo you can discount.

## When a prettier picture is the wrong move

Skip the spatial metaphor if you have one agent and a single PR. A transcript and a diff viewer are enough. A floor is overhead and a joke you will hide in screenshots.

Skip it if the vendor will not show you the event names. "AI coworkers" with no log is a screensaver. Ask for the schema. If they say proprietary, believe them: the picture is the product, the work is elsewhere.

Skip it if the demo cannot be turned off. Live mode that still injects fake activity is worse than no demo. You will ship on fake green.

Skip it if you need to screenshare a non-technical stakeholder through a fantasy of management. That meeting should be a PR, not a sim.

## How we encode this

theboringoffice is in development on purpose. The floor is playful. The contract is not: a real `opencode` boss, real task sub-agents, real permission asks, real diffs. Demo mode uses the same event shape, marked `DEMO`.

```bash
theboringoffice --demo
```

When that feels like a workplace instead of a trailer, attach a live server you already run:

```bash
theboringoffice --server http://127.0.0.1:4096
```

If you ever catch our floor moving without an event, file it like a metrics bug. It is a metrics bug. We would rather look quiet than look employed.

## Further reading

- [Start with a workflow. Agents come later.](/blog/start-with-a-workflow)
- [Put the workspace where the agents already are](/blog/universal-cli)
- [A small team's operating system for coding agents](/blog/field-guide-great-tools)
- [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) — Anthropic, on traces and not guessing why an agent failed
