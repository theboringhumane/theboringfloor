---
title: "What Kimi K3 actually changes when you run coding agents all day"
description: "Moonshot's Kimi K3 puts a 2.8T open-weights coding model with a 1M-token window behind an Anthropic-compatible endpoint, so swapping it into your terminal is an env var — the real questions are the bill, the thinking tax, and which jobs you stop giving your most expensive model."
date: "2026-08-26"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering"]
featured: false
---

Here is a scene from last week. Pane 1: your usual Claude session, mid-refactor, waiting on a permission. Pane 2: you exported two environment variables, relaunched the same harness, and now the agent in that pane is a 2.8-trillion-parameter open-weights model from Moonshot AI in Beijing. It just navigated a repo you have not finished reading and came back with a plausible plan.

Nobody installed anything. Nobody migrated a plugin, rewrote a hook, or moved an MCP server. The model swap was a config edit, and that is the part worth sitting with: when a frontier-class coding model becomes an endpoint instead of an install, the model stops being the decision. The decisions are the ones the launch posts skip. What does it cost per afternoon. Which jobs does it actually do better. Where does it sit on the roster next to the models you already run.

We spent the week reading Moonshot's model card, their pricing pages, and their own harness docs, and running the math against how we actually use coding agents. This is the field guide we wish had existed instead of the benchmark screenshots.

## What Kimi K3 actually is, in the parts that touch your afternoon

Kimi K3 is Moonshot AI's flagship, released as open weights on July 27, 2026 under their own [Kimi K3 License](https://github.com/MoonshotAI/Kimi-K3/blob/main/LICENSE) — read it before you build a product on it, because it is not plain MIT. The [model card](https://github.com/MoonshotAI/Kimi-K3) claims a first: the world's first open model in the three-trillion-parameter class.

The specs that matter when the model is driving a terminal, not a chat box:

- **2.8T total parameters, 104B active.** A mixture-of-experts with 896 experts, 16 selected per token. You are not paying to run 2.8T of compute per token, but you are talking to something that large on the far side.
- **A 1,048,576-token context window.** A full medium-sized repository fits in one prompt. The RAG pipeline you built to chunk your codebase becomes optional for a lot of tasks.
- **Native vision.** A screenshot of a broken layout can go straight into the coding loop — no describe-the-image-in-words step.
- **MXFP4 weights with quantization-aware training.** The weights are release-ready at 4-bit, not retrofitted. Arithmetic says the weights alone are around 1.4TB, so "open weights" means a cluster, not your gaming rig. Moonshot publishes [vLLM and SGLang recipes](https://github.com/MoonshotAI/Kimi-K3#5-deployment); neither is a laptop recipe.

Two behaviors matter more than any spec. First, K3 always thinks. There is no non-thinking mode; you set `reasoning_effort` to `low`, `high`, or `max` (default `max`) and it returns `reasoning_content` on every turn. Second, it was trained in what Moonshot calls preserved thinking history: on multi-turn calls you are expected to pass the assistant's `reasoning_content` and `tool_calls` back verbatim, not just the final text. If your harness truncates thinking blocks to save tokens, K3 is the model that will notice.

And one honest caveat on the 1M window: Moonshot's own footnotes show K3 scoring 91.2 on BrowseComp with context compaction at 300K versus 90.4 with the full 1M window and no management. A bigger window raises the ceiling. It does not repeal context hygiene.

## The benchmark table is basically a tie — and that is the story

Moonshot's [evaluation table](https://github.com/MoonshotAI/Kimi-K3#3-evaluation-results) puts K3 next to Claude Fable 5, GPT-5.6 Sol, Claude Opus 4.8, GPT-5.5, and GLM-5.2. Read it the way you read any vendor table: as their best day, with their footnotes. Even read that way, the coding rows are instructive:

| Benchmark | Kimi K3 | Best in table | Gap |
| --- | --- | --- | --- |
| Terminal-Bench 2.1 | 88.3 | 88.8 (GPT-5.6 Sol) | half a point |
| SWE-Marathon | 42.0 | 42.0 (K3 leads) | none |
| DeepSWE | 67.5 | 73.0 (GPT-5.6 Sol) | 5.5 behind |
| FrontierSWE | 81.2 | 86.6 (Claude Fable 5) | 5.4 behind |

K3 wins some long-horizon rows, loses some others, and lands within a few points of the closed frontier everywhere else. For an open-weights model you can download and serve yourself, "basically tied with the best closed models on terminal work" is the headline. It is not the best coding model you can rent. It is arguably the best coding model you can own.

The footnotes hide the most useful lesson: the same model scores differently under different harnesses. K3 hits 67.5 on DeepSWE under Kimi Code and 67.3 under mini-SWE-agent; 72.9 on their in-house bench under Kimi Code and 73.7 under Claude Code. The delta between harnesses is the same size as the delta between models. If you take one thing from the table, take that: your scaffold is a first-class variable, and switching models inside a harness you have already tuned is cheaper than chasing a leaderboard across tools.

Do not trust our reading either. Point it at your repo, on your task, with your tests. That is the only benchmark that survives contact with Tuesday.

## The bill: $3 in, $15 out, and a thinking tax

The [official pricing](https://platform.kimi.ai/docs/pricing/chat-k3) is flat across the whole 1M window: $3.00 per million input tokens on a cache miss, $0.30 on a cache hit, $15.00 per million output. The Kimi K2 vs K3 math is where it gets real, because Moonshot's own [K2.7 Code pricing](https://platform.kimi.ai/docs/pricing/chat-k27-code) is $0.95 in and $4.00 out with a 256K window — K3 costs roughly three to four times more per token than the coding model Moonshot was selling you in June.

Output price is the one that bites, because agent sessions are output-heavy. A model that writes a 600-line diff across eight files is generating output tokens all afternoon. At $15 a million, K3 is not the model you point at a rename loop.

Three more line items the pricing page will not tell you:

**The thinking tax.** Always-on reasoning means you pay for thinking tokens on every turn, including the turns where the task was "fix this typo." Use `reasoning_effort: low` for the small loops, or drop to `k3-256k` — Moonshot's own docs recommend the 256K variant for everyday single-file work and note it consumes about half the quota of the 1M model on their subscription tiers.

**Speed is a tier, not a spec.** Artificial Analysis [measured K3 at max effort around 39 output tokens per second](https://artificialanalysis.ai/models/comparisons/kimi-k3-vs-kimi-k2-5), against roughly 64 for K2.5. If latency is the bottleneck, Moonshot's K2.7 Code HighSpeed pushes ~180 tokens per second — at $8.00 per million output, still half of K3. Slow-and-smart and fast-and-cheap are both on the menu. They are different menu items.

**Subscription quota is not API pricing.** On the Kimi Code subscription, your tier caps the window: Moderato gets K3 at 256K, and only Allegretto and above open the full 1M. Moonshot's own pages have disagreed about exactly which tier includes what, so verify what your plan grants at checkout rather than trusting any summary — this one included.

## Kimi Code is a real CLI, but you do not have to move

Alongside the model, Moonshot ships [Kimi Code CLI](https://github.com/MoonshotAI/kimi-code) — an MIT-licensed, TypeScript terminal agent with subagents, MCP support, lifecycle hooks, video input, and an Agent Client Protocol mode for Zed and JetBrains. It is a serious harness, not a demo:

```bash
curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash
cd your-project
kimi
```

First run, `/login`, then `/model` to pick K3. Read-only operations run without confirmation; writes and shell commands ask first, which is the correct default.

But if your afternoon already lives in Claude Code or OpenCode — your hooks, your MCP servers, your muscle memory — moving harnesses for a model is the wrong trade, and Moonshot says so themselves by maintaining first-party guides for both. Their [Claude Code guide](https://www.kimi.com/code/docs/en/third-party-tools/claude-code) is environment variables:

```bash
export ANTHROPIC_BASE_URL=https://api.kimi.com/coding/
export ANTHROPIC_API_KEY=your_kimi_code_key
export ANTHROPIC_MODEL="k3-256k"
export CLAUDE_CODE_EFFORT_LEVEL=high
export CLAUDE_CODE_AUTO_COMPACT_WINDOW=262144
export CLAUDE_CODE_MAX_CONTEXT_TOKENS=262144
claude
```

Verify with `/status` once it starts — the base URL should read `api.kimi.com/coding/`. One gotcha worth knowing: the guide warns that disabling thinking in Claude Code silently routes you back to K2.6, so if the session feels suspiciously cheap, check.

Their [OpenCode guide](https://www.kimi.com/code/docs/en/third-party-tools/opencode) is even shorter — Kimi For Coding is a built-in provider:

```bash
opencode auth login   # pick "Kimi For Coding", paste a key from the Kimi Code Console
opencode
# then in-session:
/models    # k3, k3-256k, or kimi-for-coding (K2.7 Code)
/variants  # thinking effort: low / high / max
```

One account trap that cost us twenty minutes: `api.kimi.com` and `api.moonshot.cn` are separate account systems with non-interchangeable keys. The Kimi Code subscription bills through the former; pay-as-you-go platform keys live on the latter. If you get a 401 with a key you are sure is right, you are probably holding a key from the other platform.

## Where K3 earns a seat on the roster

The mental model that survived our week: K3 is not a replacement for anything. It is a new hire with a specific resume. Assign it the way you would assign any agent — by job shape.

**Give K3 the long, wide, or visual jobs.** The 1M window earns its price when the task is "understand this repository, then change it" — migrations that span dozens of files, archaeology in a codebase nobody owns anymore, refactors where the relevant context genuinely exceeds 256K. Native vision makes it the natural pick for screenshot-in-the-loop frontend work: paste the broken layout, get the CSS fix.

**Keep the bounded loops on the cheap models.** A ticket with a clear diff surface and a test does not need a 2.8T reasoner at $15 per million output. K2.7 Code at $4, or `k3-256k` at half the quota, covers the rename-test-ship grind. This is the same discipline as [not hiring a committee for a loop-with-a-check](/blog/start-with-a-workflow) — model size is not a personality, it is a budget line.

**Keep your incumbent where its harness is the point.** If a workflow depends on Claude Code's hooks, your MCP servers, or a Codex setup the team already tuned, the harness is the asset. Run K3 underneath it via the env vars above instead of migrating the workflow to chase a checkpoint.

A worked afternoon: two worktrees, as always — the model does not change the [isolation rules](/blog/running-multiple-coding-agents).

```bash
git worktree add ../tbo-search-migration -b agent/search-migration
git worktree add ../tbo-button-fix -b agent/button-fix
```

Session A gets the search migration: OpenCode pointed at `k3` with the 1M window, because the task is reading forty files before writing one. Session B gets the button fix on `kimi-for-coding`, because the diff surface is two files and a snapshot test. When A parks or finishes, you review `agent/search-migration` in a third, fresh context — [the session that wrote the code is a bad reviewer of it](/blog/running-multiple-coding-agents), and a 1M-token context does not change that either.

## When this is the wrong move

Do not put K3 on tasks you can describe in one sentence. A typo, a log line, a version bump — that is a `k3-256k` or K2.7 job at most, at a fraction of the output price, and it does not need max thinking effort to find a semicolon.

Do not buy the 1M window for tasks that fit in 30K. Most coding tasks fit in 30K. The window is for repository-scale reads, and it roughly doubles subscription quota burn against the 256K variant — pay it when the task is wide, not because the number is nice.

Do not self-host to save money. The weights are genuinely open, and ~1.4TB of MXFP4 plus serving overhead means a cluster, plus an ops burden you will pay in evenings. If you do not already serve your own models, the API is the product you want.

Do not switch harnesses to chase the model. If your hooks, allowlists, and MCP servers live in your current CLI, bring K3 to them. The harness delta in Moonshot's own footnotes cuts both ways — a tuned scaffold with a slightly weaker model beats a fresh scaffold you misconfigured at 4pm.

And the standing rule, unchanged by any checkpoint: do not add any model to the roster without a check it can run. If "looks done" is the signal, a smarter model just produces more convincing unfinished work, faster, at $15 a million tokens.

## How we encode this

We built theboringfloor on a bet that the model layer would keep churning and the ops layer would not: the permissions queue, the worktrees, the review-in-a-fresh-context habit survive every release week. K3 is the fourth time this year a "this changes everything" launch changed the roster and nothing else.

So the office treats it as a roster edit:

- An OpenCode session on the floor can point at Kimi For Coding with the `opencode auth login` flow above — the work thread, the diffs, and the parked permission asks do not care which lab signed the checkpoint.
- A Claude Code session can silently be K3 underneath via the env vars, which is exactly as confusing at standup as it sounds — name the session after the ticket, not the model.
- The roster shows which model is on which ticket, because at $15 versus $4 per million output, "who is running what" is now a cost question, not trivia.

```bash
theboringfloor --demo
```

The goal was never the newest model. It is still being able to say, at 4pm, who is doing what — and what it costs.

## Further reading

- [MoonshotAI/Kimi-K3 — model card, architecture, and benchmark table](https://github.com/MoonshotAI/Kimi-K3) and the [K3 technical report](https://github.com/MoonshotAI/Kimi-K3/blob/main/k3_tech_report.pdf)
- [Kimi K3 pricing](https://platform.kimi.ai/docs/pricing/chat-k3) and [Kimi K2.7 Code pricing](https://platform.kimi.ai/docs/pricing/chat-k27-code) — Kimi API Platform
- [Kimi Code CLI](https://github.com/MoonshotAI/kimi-code) and the [getting started guide](https://www.kimi.com/en/help/kimi-code/cli-getting-started)
- [Using Kimi in OpenCode](https://www.kimi.com/code/docs/en/third-party-tools/opencode) and [Kimi K3 in Claude Code](https://www.kimi.com/code/docs/en/third-party-tools/claude-code) — Moonshot's own harness guides
- [Kimi K3 vs K2.5 comparison](https://artificialanalysis.ai/models/comparisons/kimi-k3-vs-kimi-k2-5) — Artificial Analysis, for independent speed and price measurements
- [China's Moonshot AI claims Kimi K3 can rival OpenAI and Anthropic](https://www.bbc.com/news/articles/cy9w4q8pgp0o) — BBC, on the open-weights release
- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents) — our roster-and-worktree field guide, which this post assumes
