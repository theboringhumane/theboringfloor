---
title: "The office is in development. The work is already real."
description: "Why we are building theboringoffice in public: real sessions, real sub-agents, and no invented control plane."
date: "2026-08-23"
author: "theboringoffice team"
categories: ["Updates", "AI Agents"]
featured: true
---

Most software announces a future before it has earned the present. It promises the autonomous company, the infinite agent workforce, the dashboard that will finally make everything manageable.

We are taking a narrower—and, we think, more consequential—path.

theboringoffice is in development. It is a native terminal office built around real work that is already happening: a real `opencode` boss session, real task sub-agents, real permission requests, real diffs, and a real record of the shift. The floor is playful. The systems beneath it are not theatre.

## Start with the truth on screen

An agent walking to the manager desk means work was dispatched. An agent typing means its session is producing events. Mail means work came back. A permission prompt means the system needs a human decision.

That is the standard we want for every visual in the product: the office should reveal the system, not decorate an imaginary version of it.

The Go and Bubble Tea interface renders a single office state. `opencode serve` emits the events; the backend normalizes them; the floor and sidebar show what those events mean. Demo mode uses the same event shape, clearly labeled `DEMO`, so anyone can explore the interaction without confusing a tour for production work.

## Development is a feature, not an excuse

Being in development means we can be honest about the questions that matter:

- What should a human approve, and what should flow without interruption?
- How do you make sub-agent work inspectable without demanding constant attention?
- What has to survive when a terminal closes, a session changes, or tomorrow starts?
- Which bits of “agent management” deserve software, and which are simply another dashboard in disguise?

We do not have an API for agents to hire themselves. We do not have a browser control plane or a fictional pool of desks. What we have is a terminal workplace that can run live, attach to an existing opencode server, and make the moving pieces visible.

## The invitation

The future of software work should not be a swarm of opaque processes asking us to trust them harder. It should be legible: work has a place, decisions have an owner, and context has a memory.

That is the office we are building. Try the tour first:

```bash
theboringoffice --demo
```

Then bring a real shift through the door when you are ready.
