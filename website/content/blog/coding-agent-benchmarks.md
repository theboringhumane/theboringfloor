---
title: "How to read coding agent benchmarks without getting played"
description: "Every lab's chart says its own model is the best coding agent. What SWE-bench, Terminal-Bench, and the arenas actually measure, why the same model swings twenty points between harnesses, and how to pick with a test you run yourself."
date: "2026-08-25"
author: "theboringoffice team"
categories: ["AI Agents", "Engineering"]
featured: true
---

Open the AI news feed on any given week this year and you can watch three labs win the same argument. One week Claude tops the SWE-bench leaderboard. The next, Codex tops Terminal-Bench. A third headline has Kimi or Grok leading the arena. Your group chat picks a side. Your manager asks which one the team should standardize on.

Here is the part the headlines skip: all three claims were true at the moment they were screenshotted. They were true about different tests, run in different rooms, graded by different judges. Some of them stopped being true the following Tuesday.

A benchmark score is a photo of a model on one afternoon, in one harness, on one dataset. Marketing frames it as a portrait. This post is how we read the photo — what each of the big coding benchmarks actually measures, why the same model can look elite and mediocre in the same week, and the only evaluation we actually trust for picking a model: a small one we run ourselves.

## The same model, three different report cards

"Best coding model 2026" is not one question. It is at least four, and each benchmark answers exactly one of them. Before you compare any two numbers, figure out which question produced them.

| Benchmark | What it actually measures | What it misses |
| --- | --- | --- |
| [SWE-bench Verified](https://www.swebench.com/) | Can the agent produce a patch that passes the hidden tests for one of 500 human-filtered GitHub issues, from 12 Python repos | Long-horizon work, your stack, your private codebase; the problems and their fixes are public and old |
| [Terminal-Bench 2.x](https://www.tbench.ai/leaderboard/terminal-bench/2.0) | Can the agent operate a real terminal: install deps, fix builds, debug systems, finish ops tasks in a sandboxed shell | Interactive judgment, ambiguous tickets; the agent harness dominates the result |
| [LMArena WebDev](https://arena.ai/leaderboard/code/webdev) | Which of two anonymous outputs a human voter preferred, at scale, on web-building prompts | Correctness. Taste, formatting, and speed of the demo sway votes |
| Vendor model cards | Whatever benchmark the lab chose to run, under the lab's own harness, at the lab's chosen settings | Comparability. The lab picked the room where its model looks best |

None of these is the question you actually have, which is: *will this model close my tickets, in my repo, at a price I can defend?* Every number below should be read as evidence about that question, not an answer to it.

## SWE-bench Verified measures one good afternoon

SWE-bench Verified is the score every launch post leads with, so it is worth knowing precisely what it is. [The benchmark](https://www.swebench.com/) is 500 issues scraped from merged pull requests in twelve popular Python repositories, filtered by human annotators. The agent gets the repo and the issue text. It wins if its patch passes the tests.

That is a real skill. It is also a narrow one. [Epoch AI's anatomy of the benchmark](https://epoch.ai/publications/what-skills-does-swe-bench-verified-evaluate) is the best single read on what is inside the 500: about 90% of the tasks are fixes an experienced engineer could finish in under an hour, and 39% are under fifteen minutes. Django alone supplies nearly half the issues; five repos supply over 80%. Half the issues predate 2020. Epoch rates the contamination risk as high, because all of it — the code, the issues, the gold patches — has been public training data for years.

The scores tell the saturation story on their own. Top results went from roughly 20% in August 2024 to the high 70s by early 2026; as of this week [the Verified leaderboard](https://www.swebench.com/) tops out at 79.2%. When a test everyone trains on starts returning A-minuses for everyone, the test has stopped separating the class.

The labs know it. OpenAI has said it no longer evaluates frontier models on SWE-bench Verified, because improvements "increasingly reflect how much the model was exposed to the benchmark at training time" — a sentence worth reading twice, quoted in [this synthesis of the production gap](https://tianpan.co/blog/2026-04-09-agentic-coding-production-swebench-gap). The same piece notes that on Scale AI's SWE-bench Pro, a contamination-resistant variant built from held-out and proprietary codebases, top frontier models score around 23%. The gap between 80 and 23 is not a rounding error. It is the difference between a benchmark and a measurement.

How to read a SWE-bench claim in 2026: it tells you the model can fix a well-specified bug in a famous Python repo. That correlates with being a decent coding agent. It does not tell you how the model handles your hundred-file TypeScript monorepo with the bespoke auth layer.

## Terminal-Bench grades the shell, not just the patch

Terminal-Bench asks a different question, and for daily-driver purposes it is the more honest one: can the model survive a real shell? Tasks involve standing up environments, repairing broken builds, wrestling dependencies, debugging systems — the ops half of the job that SWE-bench never touches.

Two things make [the Terminal-Bench boards](https://www.tbench.ai/leaderboard) unusually readable, if you look past the accuracy column.

First, they publish the agent *and* the model as separate columns, which quietly admits the central truth of this whole post: the score belongs to the combination, not the model. On [the 2.0 board](https://www.tbench.ai/leaderboard/terminal-bench/2.0), Claude Opus 4.6 appears eleven times with scores ranging from 80.2% down to 58.0% — a 22-point spread for identical weights, decided entirely by the harness around them.

Second, [the 2.1 board](https://www.tbench.ai/leaderboard/terminal-bench/2.1) publishes cost. The number-one entry, Claude Code with Fable 5, scored 83.8% at $553 per run. The number-two entry, Codex with GPT-5.5, scored 83.1% at $2,059 — four times the money for 0.7 points less. Meanwhile Cursor CLI running Grok 4.5 hit 79.3% at $134, the cheapest run in the top five. Accuracy is the column that makes the keynote. Cost is the column that shows up on your invoice. The 2.1 board even carries a "Hacks" column — one top-five entry shows a 9-point deduction — because gaming these tests is now common enough to grade.

Watch the ceiling, too. The 2.1 board is already bunching up around 84%, and [Snorkel's benchmark analysis](https://snorkel.ai/blog/senior-swe-bench-evaluating-coding-agents-like-senior-engineers/) notes that on the harder, not-yet-saturated Terminal-Bench 3, the best model manages 43.5%. The pattern repeats: every benchmark is hard until it is not, and then it is marketing.

## The harness is half the score

This is the single most ignored line in every benchmark table, and the one that will save you from the most bad comparisons: **you cannot compare two scores that were earned in different scaffolds.**

The scaffold — the agent loop, the tools, the retry policy, the prompt — is not a neutral wrapper. [Epoch AI's measurements](https://epoch.ai/publications/what-skills-does-swe-bench-verified-evaluate) put the scaffold effect at up to 20 points: Claude 3.7 Sonnet went from 62.3% to 70.2% on Verified with a custom scaffold; GPT-4o jumped from 23% to 33.2% just by switching from SWE-agent to Agentless; DeepSeek R1 scored 33% in Epoch's tooling versus the 57.6% its own lab reported.

You can see it live on today's boards. On the [SWE-bench Verified leaderboard](https://www.swebench.com/), Claude 4.5 Opus at medium effort scores 79.2% inside the live-SWE-agent harness and 74.4% inside mini-SWE-agent — same model, same test, five points apart. Kimi K2 scores 71.2% with the Lingxi v1.5 harness and 43.8% with mini-SWE-agent: a 27-point swing on the same weights. The arena has even started printing the harness in the model's name — [LMArena's WebDev board](https://arena.ai/leaderboard/code/webdev) lists one entry as "gpt-5.6-sol-xhigh (codex-harness)," which is exactly the kind of honesty every leaderboard needs.

So when a launch post says "Model X beats Model Y by three points," the first question is not whether three points matters. It is whose harness each side was wearing. If the vendor ran its own agent against a generic-harness baseline, you are reading an advertisement, not a result.

The fair fights do exist. The bash-only section of [swebench.com](https://www.swebench.com/) runs every model through the same 100-line mini-SWE-agent, and there the current reading is Claude 4.5 Opus at 76.8%, with Gemini 3 Flash and MiniMax M2.5 close behind — the latter at $0.07 average cost per task versus Claude's $0.75. Same harness, same test, and suddenly the comparison means something, including the price tag.

## Arena scores measure applause

[LMArena's WebDev leaderboard](https://arena.ai/leaderboard/code/webdev) is the third number that gets quoted at you, and it is a different instrument entirely. There is no test suite. Humans get two anonymous outputs side by side and vote for the one they like. As of this week the board reads claude-opus-5-max at 1691, kimi-k3-max at 1674, qwen3.8-max at 1669, with grok-4.6-high fifth at 1629.

Three things to hold in your head before using an arena score to pick anything:

**Preference is not correctness.** Voters reward what looks good in a thirty-second comparison — layout, polish, the demo working on the first try. Nobody in that voting booth ran the test suite, checked the diff for the security hole, or maintained the output for a quarter.

**The error bars eat the ranking.** The gap between rank 1 and rank 4 on that board is 28 Elo points, and the individual uncertainties are ±8 to ±17. Ranks two through four are statistically overlapping. "Kimi passes Qwen for second" is a coin flip with a press release.

**It is still signal — just a different one.** The arena is the best public proxy we have for "will the first draft look competent," which matters for prototyping, front-end scaffolding, and demos. It tells you almost nothing about whether the agent will survive your CI.

If a model wins the arena and loses every harness-matched benchmark, believe the benchmarks. Applause is cheaper than correctness.

## Run the only benchmark that matters: yours

After all of this, the move that actually settles the model question for your team costs one afternoon and no faith.

Build a private eval out of your own history. Pull ten to twenty tasks your repo really produced last month: a bug with a reproduction, a dependency upgrade that fought back, a small feature with a spec, one miserable debugging session. Freeze the harness — the same agent CLI you actually use, the same prompt, the same tools allowed, the same timeout — and run every candidate model against the same set, two or three times each, because single runs are noisy. Score pass or fail with your own tests. Write down wall-clock time and dollars next to each run.

Two details make this work where leaderboards fail. The tasks are private, so no model has trained on them — your little benchmark is uncontaminated by construction. And the tasks are *yours*, so the distribution finally matches the job: your languages, your repo size, your kind of ambiguity.

Keep the task list and re-run it when a model bumps a version. A personal trend line — "the new checkpoint closed 3 of our 15 last month, now it closes 9" — is worth more than any public board, because it moves with the thing you are actually deciding about.

This is also the only honest way to answer the Claude-versus-Codex-versus-Grok-versus-Kimi question. This week's boards have Claude leading Terminal-Bench, GPT-5.5's harness topping Terminal-Bench 2.0, Kimi second on the web arena, and Grok posting the cheapest top-five terminal run. All four are right. Your repo breaks the tie.

## When benchmarks genuinely mislead

Do not trust a coding benchmark number when any of these hold:

**The margin is inside the error bars.** Terminal-Bench entries carry ±2–3% uncertainties; arena ranks two through four overlap outright. A two-point lead is not a lead. It is a shrug.

**The two rows used different harnesses.** You are comparing scaffolds, not models. Check the agent column or skip the comparison.

**The benchmark is public, old, and Python-shaped.** Half of SWE-bench Verified predates 2020, five repos supply over 80% of it, and its own co-creator has stopped reporting it. Treat any Verified score as "trained on the exam" until shown otherwise.

**The metric is preference and the pitch is correctness.** Arena votes measure what humans applaud. Launch posts cite them as if they measure what compiles.

**The vendor picked the room.** A model card is the lab's best afternoon: its harness, its settings, its chosen benchmark. Useful as a ceiling, useless as an expectation.

And the reverse: benchmarks mislead least when the harness is held constant (bash-only SWE-bench, the Terminus-2-only rows on Terminal-Bench), when the board publishes cost and error bars, and when you read the trend over months instead of the rank of the week. Used that way, they are a fine first filter. They are just never the last one.

## How we handle this on the floor

We run a lot of coding agents, so we stopped reading leaderboards as verdicts and started treating them as nominations.

theboringoffice is a terminal app for running several coding agents like an office floor — and the workflow above is what it makes cheap: hand the same ticket to two agents in isolated worktrees, let them run side by side, then compare diffs and test results instead of screenshots of someone else's benchmark. The model that wins your fifteen tasks is your model, whatever the arena says this week.

## Further reading

- [SWE-bench leaderboards](https://www.swebench.com/) — including the bash-only section, the fairest public fight
- [Terminal-Bench 2.0](https://www.tbench.ai/leaderboard/terminal-bench/2.0) and [2.1](https://www.tbench.ai/leaderboard/terminal-bench/2.1) leaderboards — note the agent column and the cost column
- [LMArena WebDev leaderboard](https://arena.ai/leaderboard/code/webdev) — the preference instrument
- [What skills does SWE-bench Verified evaluate?](https://epoch.ai/publications/what-skills-does-swe-bench-verified-evaluate) — Epoch AI's anatomy: task sizes, repo concentration, contamination, scaffold effects
- [Agentic coding in production: what SWE-bench scores don't tell you](https://tianpan.co/blog/2026-04-09-agentic-coding-production-swebench-gap) — the 80%-versus-23% gap, the METR slowdown study, and OpenAI's retirement of Verified
- [Senior SWE-bench: evaluating coding agents like senior engineers](https://snorkel.ai/blog/senior-swe-bench-evaluating-coding-agents-like-senior-engineers/) — saturation on Terminal-Bench 2.1 versus the 43.5% ceiling on Terminal-Bench 3
