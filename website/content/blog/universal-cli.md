---
title: "A terminal office for the work agents actually do"
description: "A development update on the native CLI: real opencode sessions, visible sub-agent work, durable context, and a demo mode."
date: "2026-08-20"
author: "theboringoffice team"
categories: ["Release", "Updates"]
featured: true
---

We are building theboringoffice as a native Go CLI because the place where coding agents work should not be another browser tab asking to be managed.

It is a startup office in your terminal. The boss is a real `opencode` session managed by Oikonomos. Employees are real opencode task sub-agents. The task board is backed by agentmemory actions; the mail room is backed by agentmemory signals. The floor gives those moving parts a shape you can understand at a glance.

## Try the office before you trust it

Development tools should earn their place in a workflow. The fastest honest evaluation is a clearly labeled tour:

```bash
theboringoffice --demo
```

Demo mode simulates the same event flow the interface uses in live mode. It is designed to show the interaction without pretending there is a backend running behind it.

When you are ready to run a real shift, use the installer:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh
```

Then launch the office:

```bash
theboringoffice
```

Live mode starts `opencode serve` and opens the boss session. If you already have a server, attach directly instead:

```bash
theboringoffice --server http://127.0.0.1:4096
```

## One terminal, six useful views

The right-side cockpit moves between chat, terminal, agents, board, mail, and activity with `tab`. These are not separate products. They are different views of the same shift:

- **Chat** is where you speak to the boss.
- **Terminal** is an embedded, real PTY shell.
- **Agents** shows the roster and current state of work.
- **Board** and **mail** carry agentmemory actions and signals.
- **Activity** makes the office's recent events visible.

Work threads keep sub-agent diffs, tool calls, and thinking attached to the task. Permission and question popovers surface decisions without trapping the rest of the workflow.

## Context is part of the interface

The office normally restores the last chat on launch. To return to a particular past session, pass its ID:

```bash
theboringoffice --session <your-session-id>
```

The configuration lives at `~/.theboringoffice/configs/brain.json`; inspect the defaults with `theboringoffice --print-default-config`.

This is a development-stage tool, and we are keeping the promise deliberately small: make real agent work visible, make human decisions manageable, and make useful context survive the next restart. If that changes how a solo developer or small team can use agents, the office has done its job.
