---
title: "Grok coding is real now. Here's where it fits and where it doesn't."
description: "xAI ships a new Grok every few weeks and the names stopped helping months ago — here's the current coding lineup, what it costs, what it's genuinely good at, and the jobs to give to someone else."
date: "2026-08-27"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering"]
featured: false
---

Someone on your team dropped a benchmark screenshot in chat this week. Half the model names on it did not exist in the spring. One of them was a Grok, sitting close enough to the top that the question arrived on schedule: is Grok good for coding now?

The honest answer is: which week did you ask? In the last twelve months xAI shipped Grok Code Fast 1, Grok 4 Fast, Grok 4.1, Grok 4.1 Fast, Grok 4.5, Grok 4.6, a fast variant of 4.6, a terminal agent called Grok Build, and an always-on agent product called Grok Bot. Some of those are models. Some are products. One is a subscription tier wearing a model's name.

We spent the last couple of weeks routing real tickets through the current lineup. This post is the map we wish someone had handed us first: what exists, what it costs, what it is actually good at, and where you should still send the hard work.

## First, separate the models from the products

The confusing part is that "Grok" now names at least four different things you can spend money on. Before any routing decision, get the taxonomy straight:

| Name | What it is | Where you meet it | Cost signal |
| --- | --- | --- | --- |
| `grok-4.6` | Flagship model, 500k context, Feb 2026 cutoff | API, Cursor, Copilot, Grok Build, Bedrock, Foundry | $2 / 1M input, $6 / 1M output |
| `grok-4.6-fast` | Same class, faster | API | Twice the base price |
| Grok Code Fast 1 | 2025's small purpose-built xAI coding model | Mostly historical now | Was $0.20 / 1M input |
| Grok Build | The terminal coding agent (a product, not a model) | Your terminal, web, mobile | SuperGrok / X Premium Plus |
| Grok Bot | Always-on computer-use teammates (also a product) | Their own cloud computers | Subscription-gated beta |

A few things fall out of that table. Grok Code Fast 1 was the interesting experiment of late 2025 — a small reasoning model trained from scratch on a programming-heavy corpus, deliberately tuned to be good with grep, terminals, and file edits, and priced at $0.20 per million input tokens. xAI shipped it in stealth under the codename `sonic`, then gave it away free for weeks through GitHub Copilot, Cursor, Cline, Roo Code, Kilo Code, opencode, and Windsurf. It earned xAI a seat at the coding table.

It is also no longer the story. The Grok 4.1 Fast line was retired on May 15, 2026, and xAI's own docs now answer "which model should I choose for code?" with one entry: Grok 4.6. The fast-cheap lineage did not die — it was absorbed into the flagship family as configurable reasoning effort and a fast variant — but if you have `grok-code-fast-1` pinned anywhere, treat it as a migration item, not a foundation.

Grok Build is not a model at all. It is the terminal coding agent xAI launched in May: plan mode, clean diffs, AGENTS.md and MCP support, parallel subagents in their own worktrees, a headless `-p` flag, ACP for building your own orchestration. And Grok Bot is something else again — it gets its own section below, because conflating it with coding is the most common mistake in the current discourse.

## Grok 4.6's real strengths: long runs and first passes

Grok 4.6, released August 12, is explicitly tuned for long-running agentic work: staying with a task across many steps, researching before writing, and — the part we kept noticing — checking its own work more often before moving on. xAI says it trained the model on a wide spread of agentic RL tasks, including kernel optimization, web development, and CAD, and it shows in the shape of the output.

On greenfield tasks it produces unusually strong first passes. Give it a concrete product idea and it comes back with structure and a visual language in one shot, which makes it a natural fit for the iterate-in-the-loop pattern: start from something substantial, then refine with feedback rounds. On refinement work inside an existing codebase — the bread and butter of our afternoons — it follows AGENTS.md conventions well and only occasionally invents abstractions nobody asked for. Nothing a review pass does not catch.

Now the part the benchmark screenshots leave out. On [xAI's own published table](https://x.ai/news/grok-4-6), Grok 4.6 leads GDPVal-AA (1753) and the Harvey LAB legal eval, and sits a point off the top of the Artificial Analysis index. It *loses* DeepSWE (65.9% vs 73% for GPT-5.6 Sol Max) and loses Terminal-Bench badly (26% vs 34.6%) — the two boards that most resemble a long, grubby agentic session in a real shell. These are vendor-reported numbers with competitor figures pulled from public leaderboards, so season to taste. But you do not need to run your own eval to draw the conclusion: Grok 4.6 is a frontier model that wins some boards and loses others. It is not the best coding model in the world. It can still be the best model for a specific slot in your rotation, which is a different question entirely.

## Route by price and latency, not by loyalty

The economics are the actual story. Grok 4.6 costs $2 per million input tokens and $6 per million output, with a 500k context window. The fast variant is twice the price. The old Code Fast 1 was a tenth of the base input price. xAI's pricing history says the company wants to be the value option at every tier it plays in.

That suggests a routing policy, and it is the same policy we apply to every provider: spend frontier tokens on judgment, spend cheap tokens on volume.

- **Greenfield first passes, scaffolding, "make me a v1 of this dashboard":** Grok 4.6. A strong first pass saves more review time than a perfect fifth pass, and this is its best-demonstrated skill.
- **Long reasoning tails — subtle concurrency bugs, refactors across module boundaries, the ticket nobody wants:** give these to whichever model tops *your* eval this month. On xAI's own numbers, that is not automatically Grok.
- **High-volume loops — test-fix cycling, renames, mechanical migrations:** the cheapest fast tier you can reach. A model that returns before you context-switch is worth more than a model that returns smarter.
- **Fresh-context research:** Grok's web and X search tools are server-side and opt-in. Without them, the model knows nothing after its February 2026 cutoff. If the task needs to know what shipped last week, wire the tools or pick a different lane.

The one-sentence version: Grok 4.6 is a strong default for generating and a defensible choice for iterating, and the models that beat it on DeepSWE and Terminal-Bench are already in your rotation. Use both. That is what a rotation is for.

## Wire it up: API, agent CLI, or the official harness

Three paste-able paths, cheapest first.

**Raw API.** The xAI API is OpenAI-compatible:

```bash
curl -s https://api.x.ai/v1/chat/completions \
  -H "Authorization: Bearer $XAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4.6",
    "messages": [{"role": "user", "content": "Explain why this test is flaky."}]
  }'
```

**Your existing agent CLI.** If you run opencode, Grok is a first-class provider — `opencode auth login`, pick xAI, paste the key, then set the model:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "xai/grok-4.6"
}
```

Cursor and GitHub Copilot both carry Grok 4.6 in the picker as of mid-August; Cline, Roo Code, Kilo Code, and Windsurf have carried Grok models since the Code Fast days. If you pay for SuperGrok or X Premium, you can also sign in with the subscription inside opencode, Kilo Code, or Warp instead of metering API tokens — worth checking before you put a card on the API console.

**The official harness.** Grok Build installs with one line:

```bash
curl -fsSL https://x.ai/cli/install.sh | bash
```

Worth an afternoon even if you keep your current driver. The plan-approve-diff loop is disciplined, worktree-scoped subagents are the correct isolation primitive, and xAI open-sourced the harness in July, so you can read how a frontier lab thinks an agent loop should be assembled. We stole at least one idea.

One config habit regardless of path: pin models when reproducibility matters. Per xAI's alias rules, plain `grok-4.6` floats forward to the latest stable release, while a dated `<modelname>-<date>` variant stays put. For anything running unattended in CI, stay put.

## Grok Bot is an ops product, not a coding model

Because the names collide: Grok Bot, launched August 11 in early beta, is not the next Grok Code. It is a team of always-on agents that each get their own cloud computer, sign into your apps the way a person would — including tools with no clean API or MCP — and keep working after you close the laptop. You message a Bot like a colleague. It files the ticket, updates the CRM, reproduces the bug in the product UI, and comes back when something needs your approval.

Three design ideas are worth stealing if you build agent tooling:

- **Bots coordinate without you.** They message each other, share threads, and can sit in a group chat with a chief-of-staff Bot on top assigning lanes. The human stops being the message bus.
- **Routines are learned by watching.** You do a job once with the Bot following along; it saves the workflow and runs it alone next time. No DSL, no flowchart builder.
- **Separate usage quotas.** Work handed to a Bot does not burn your Grok or Cursor plan allowance, which removes the strange guilt math of "is this task worth the tokens."

It is gated behind SuperGrok tiers and Cursor's paid plans, and it is aimed at ops-shaped work: sales outbound, inboxes, invoices, onboarding. A Bot can reproduce a bug and hand the fix to a debugging Bot, but the debugging is still done by the models above. If your problem is "who writes the diff," Grok Bot is not the answer. If your problem is "who keeps the plates spinning between diffs," it is the most interesting thing xAI shipped this month.

## Treat xAI as a fast-moving dependency

The boring ops advice for a provider that ships this fast:

1. **Expect retirements.** The 4.1 Fast models got about six months between launch and shutdown. Pin dated model versions anywhere a surprise would hurt, and skim the docs changelog the way you skim a dependency's release notes.
2. **Watch the naming.** The company itself is now SpaceXAI after the February acquisition, and the docs, console, and branding are mid-migration. Assume more renames, not fewer.
3. **Watch the distribution list.** Grok 4.6 reached Cursor, Copilot, OpenRouter, Vercel, Cloudflare, Amazon Bedrock, and Microsoft Foundry inside two weeks. If procurement or data-residency rules once decided where you could run it, those constraints may have quietly dissolved. Recheck before you write policy.
4. **Watch the training loop.** The 4.6 notes say xAI used Grok 4.5 to regenerate SFT trajectories across agent harnesses, then filtered traces with model-based checks — while the free tiers and subscription CLIs collect agentic usage at scale. The loop is closing. The gap between "the model in the benchmark screenshot" and "the model in your terminal" will keep shrinking.

## When Grok is the wrong pick

Do not switch because of a screenshot. If your harness, prompts, and permissions are tuned around another model's tool-calling habits, a swap costs a week of weird failures before it pays anything back. Migration is a project, not a config line.

Do not route your hardest terminal work to it yet. On xAI's own table, Terminal-Bench and DeepSWE belong to someone else this month. If your day is long-horizon shell sessions, the leaderboard that matters is not the one Grok wins.

Do not treat the chatbot as fresh. Grok knows nothing past February 1, 2026 unless you enable the server-side search tools. An agent with a stale map of your dependencies will confidently import packages that no longer exist.

Do not build on `grok-code-fast-1`. It was a good model and an important one, but the line it opened has moved into the flagship family. Check the docs before anything you pin still resolves.

And do not hand Grok Bot the keys to anything you would not hand a new hire on day one. It signs into real tools with real credentials. Routines, approvals, and least-privilege accounts are not optional accessories; they are the product.

## How we slot it in

Multi-model is the default state of our floor, so Grok 4.6 arrived as one more worker with a particular shape: fast generator, strong first pass, cheap enough to keep warm.

In theboringfloor it shows up as another pane next to the Claude Code and Codex sessions. The habits from [our multi-agent post](/blog/running-multiple-coding-agents) all still apply: worktrees isolate the writes, the board tracks who owns what, and permission asks queue instead of hijacking the screen. The only thing a new Grok model changes is the routing table — greenfield tickets and scaffold work drift toward the Grok panes now — and the reconstruction bill for "which session made this mess" does not move at all.

That is the real lesson of xAI's release cadence. Models will keep arriving faster than you can re-tune for them. The floor, the board, and the worktrees are what make a new model a Tuesday instead of a quarter.

## Further reading

- [Introducing Grok 4.6](https://x.ai/news/grok-4-6) — xAI, including the eval table we quote
- [Introducing Grok Bot](https://x.ai/news/introducing-grok-bot) — the always-on teammate product
- [Grok Code Fast 1](https://x.ai/news/grok-code-fast-1) — the August 2025 model that started the coding story
- [Introducing Grok Build](https://x.ai/news/grok-build-cli) — the terminal agent, its install, and its subagent model
- [Grok Models & Pricing](https://docs.x.ai/developers/models) — context windows, aliases, cutoff dates
- [May 15, 2026 Model Retirement](https://docs.x.ai/developers/migration/may-15-retirement) — what the fast cadence costs
- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents) — our ops model for exactly this situation
