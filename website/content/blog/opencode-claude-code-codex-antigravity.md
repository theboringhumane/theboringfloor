---
title: "Claude Code vs Codex vs opencode vs Antigravity is the wrong fight"
description: "Everyone argues about the best AI coding CLI of 2026; the afternoons that actually ship give Claude Code, Codex, opencode, and Google Antigravity different jobs and never let them share a checkout."
date: "2026-08-28"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering"]
featured: false
---

Monday, 2pm. Claude Code is forty minutes into a refactor it planned before it touched a file. A Codex cloud task is grinding through a dependency bump in the background, and it will come back as a diff you review like a pull request. opencode is open in a second pane, pointed at a cheaper model, writing the throwaway migration script you will never commit. Antigravity has one more agent verifying a UI change in a real browser, screenshot by screenshot, while you read the diffs.

None of these tools is winning. That is the point.

Every "best AI coding CLI of 2026" list assumes the four of them are competing for one job. They are not. They are four different shapes of agentic coding tool, and the question that decides whether your afternoon works is not which one is smartest. It is which shape owns which job — and how you keep them from stepping on each other once more than one is moving.

## The winner question is a category error

Claude Code vs Codex, opencode vs Antigravity: the versus framing is how you end up running the wrong tool for the wrong job and blaming the model.

All four will write the code. What actually differs is everything around the code: where the agent runs, whose models it will touch, how the work comes back to you, and what evidence it leaves behind. A terminal agent you babysit is a different purchase from a cloud task you review at 5pm. An open harness that takes any provider's keys is a different commitment from an IDE that promotes the agent to its own surface and hands you screenshots as proof.

Once you see them as shapes, the ranking content reads differently. "Antigravity beats opencode" is a nonsense sentence the way "truck beats kayak" is: true on some water, irrelevant to the trip you are actually making.

So we stopped ranking them. We route to them. Here is the routing table we ended up with — what each one is genuinely good at, where it runs, and the one limit to respect.

## Claude Code is the senior you brief once

Claude Code is Anthropic's agent, and its center of gravity is your terminal. It works alongside your CLI tools, with IDE extensions if you want them, and it is built for the long, careful session: read the repo, propose a plan, execute it across many files, run the tests, explain itself. The [product page](https://claude.com/product/claude-code) says it "meets you where you code," and for once the marketing undersells it. It lives where you code.

Two details matter in practice. First, `CLAUDE.md`: the project memory file is where you pour the senior-dev briefing once — conventions, landmines, the deploy script nobody touches — and every later session inherits it. Second, the autonomy ladder Anthropic keeps shipping: plan mode, checkpoints you can rewind when a turn goes wrong, and a headless mode (`claude -p`) that lets the same agent run inside scripts and CI. The [checkpoints announcement](https://www.anthropic.com/news/enabling-claude-code-to-work-more-autonomously) is the one to read for the flavor: rewind the turn, don't re-prompt from scratch.

```bash
curl -fsSL https://claude.ai/install.sh | bash
```

Give Claude Code the jobs where being right beats being fast: the refactor that touches eleven files, the migration where the tests are the spec, the bug that needs someone to hold the whole call graph at once. The limit is the flip side of the same shape. It is a terminal companion, not a dispatcher — if the job is "go away for three hours and open a PR," the next tool was built for exactly that.

## Codex is the CLI that hands the long jobs to the cloud

Codex's trick is that the agent does not have to stay attached to your attention.

The [Codex CLI](https://github.com/openai/codex) is OpenAI's open-source, Rust-built terminal agent. It runs locally against your checkout, you sign in with a ChatGPT account (Codex rides on the Plus, Pro, Business, Edu, and Enterprise plans), and locally it behaves like the other terminal agents: reads, edits, runs commands, asks before it does something scary.

The personality shows up when you stop watching. The same agent runs as Codex cloud tasks on OpenAI's hosted surface. You hand off a well-scoped job — the dependency bump, the codemod, the test backfill — and it works in its own configured environment, then comes back as a reviewable diff. You can triage and launch those tasks without leaving the terminal, and the same agent follows you into the IDE extension, GitHub, Slack, and Linear. Of the four tools here, Codex is the one that most wants to be a queue.

```bash
npm install -g @openai/codex
# or: brew install --cask codex
```

Give Codex the jobs with a crisp definition of done and no need for your company while they run. Batch work. Background work. The limit: cloud tasks execute in OpenAI's environment, with setup scripts standing in for your laptop's quirks — and the review habit is not optional. A diff you never read is just unreviewed code with better marketing.

## opencode is the harness that doesn't care whose model you brought

opencode is the only tool on this list that refuses to have an opinion about models, and that refusal is the feature.

It is open source (MIT), from the Anomaly/SST crew, and it runs as a terminal TUI, a desktop app, or an IDE extension. You bring API keys for whichever providers you already trust; the `/connect` flow wires them up, and there is a curated model list (opencode Zen) if you would rather copy someone else's homework. The [docs](https://opencode.ai/docs/) read like the tool feels: plan mode on Tab, `/undo` when a turn goes sideways, `/init` to write an `AGENTS.md` so the agent learns your repo's shape.

The payoff is right-sizing. The frontier model gets the hairy refactor. The cheap, fast model gets the throwaway script, the rename, the "is this regex right." Same harness, same keybindings, same muscle memory — the engine is a dial, not a marriage. And because opencode can run as a server, not just an interactive TUI, it is the harness other tools (ours included) drive programmatically.

```bash
curl -fsSL https://opencode.ai/install | bash
# or: npm install -g opencode-ai
```

The limit is honest: freedom has a config file. Point it at a weak model and you have a weak agent in a very nice TUI. opencode makes the model your decision, which means it also makes it your problem.

## Google Antigravity is the IDE that shows its work

Antigravity is the newest shape here and the least like a CLI. Google launched it alongside Gemini 3 as an "agent-first" IDE, and the [intro post](https://antigravity.google/blog/introducing-google-antigravity) is unusually plain about the bet: agents should not be chatbots bolted onto a sidebar. They should get their own surface.

In practice that means two views. The Editor is the familiar AI-powered IDE — agent in the side panel, synchronous, you in the loop. The Manager surface flips the relationship: a mission-control view for spawning, orchestrating, and watching multiple agents across multiple workspaces in parallel. Each agent gets direct access to the editor, the terminal, and a browser (the browser control comes from Google's computer-use model), and here is the part we did not expect to like: the agent reports back with Artifacts. Task lists, plans, screenshots, browser recordings. Evidence, not vibes. You comment on an artifact and the agent adjusts without you breaking its flow.

There is a knowledge base so agents learn from past work, model optionality beyond Gemini 3 (Claude Sonnet 4.5 and GPT-OSS are in the menu), and — as of this writing — it is a public preview at no charge for individuals, on macOS, Linux, and Windows. It is a download from [antigravity.google](https://antigravity.google/), not a package install.

Give Antigravity the jobs whose proof is a working page: the UI change that has to be clicked through, the end-to-end feature that spans frontend, terminal, and browser, any task where a screenshot is worth a thousand-line transcript. The limit: it is an IDE, not a terminal citizen. If your home is tmux, Antigravity is a second house, not a renovation.

## Four shapes, one routing table

Compressed, the personalities look like this:

| Tool | Model(s) | Where it runs | What it's genuinely good at |
| --- | --- | --- | --- |
| Claude Code | Anthropic's Claude models | Terminal first, IDE extensions | Long, careful, multi-file work in a repo it has read; brief-it-once sessions |
| Codex | OpenAI's models | Local CLI, cloud tasks, IDE extension, GitHub/Slack/Linear | Unattended and batch jobs that come back as reviewable diffs |
| opencode | Yours — any provider's keys, or the curated Zen list | Terminal TUI, desktop app, IDE extension | Right-sizing the model to the job; owning the harness; being embedded |
| Google Antigravity | Gemini 3 (Pro and Flash), plus Claude Sonnet 4.5 and GPT-OSS options | Desktop IDE: Editor + Manager surfaces (macOS, Linux, Windows) | End-to-end tasks verified in a real browser; supervising parallel agents on artifact evidence |

The rows are personalities, not rankings. Every cell is "it depends" with the depends filled in.

## Running them together is the actual skill

The tools never fight each other. Your checkout does, and your attention does. Three habits keep the rotation from becoming a food fight.

**Isolate the files, or they will fight.** Two agents in one working tree will overwrite each other. One formats a file the other is mid-edit; one reverts a hunk it does not recognize because the file changed under it. The boring fix is git worktrees:

```bash
git worktree add ../proj-auth -b agent/auth-refresh
git worktree add ../proj-ui -b agent/ui-polish
```

Each agent gets a checkout, each checkout gets a branch, and you review `git diff main` the way you would review a coworker — not by scrolling a chat. We wrote the [long version of this](/blog/running-multiple-coding-agents) already. The short version: the branch is the only honest record of what an agent did.

**Match the job to the shape.** The sections above, as one breath: Claude Code for the careful refactor you would hand a senior. Codex cloud for the well-scoped job with a crisp done that does not need you in the room. opencode with a right-sized model for everything in between, especially the cheap stuff. Antigravity for anything whose proof is a page that actually renders. Wrong-fit routing is how you get a cloud task dispatched for a typo and a babysat terminal grinding through a codemod.

**Watch the queue, not the panes.** The bottleneck in a four-tool afternoon is never generation; it is you, alt-tabbing between permission asks. Pre-approve the boring commands everywhere the tool allows it, keep the scary ones queued instead of modal, and remember that [a permission is not a question](/blog/a-permission-is-not-a-question). When two sessions both claim a file, believe what you wrote down — the branch names, the two-line note — because both panes will sound confident.

## When this is the wrong move

One tool already does your whole week. Then adding three more is an expense report, not a workflow. Start with the single job that annoys you most, route it to the shape that fits, and stop there until something else hurts.

You have no checks. Four agents without a test suite means you are the CI pipeline for all four. That job does not get cheaper with more panes.

You are shopping benchmarks. The model under each harness moves monthly; the shapes move slowly. Route on shape, and let the vendors race each other underneath your habits.

You want the demo, not the shift. Four tools running at once looks like a company and behaves like a queue of which you are the only worker. [Start with a workflow](/blog/start-with-a-workflow); the rotation is a later move on that ladder, not a personality.

## How we encode this

We run this rotation ourselves, and theboringfloor exists because the coordination hurt more than the models did. It is a terminal app that treats coding agents like an office floor:

- opencode and Claude Code sessions share one floor and one event model — pick the boss's brain at boot (`--backend claudecode`), and the rest of the shift is identical.
- Permission asks stack in a queue instead of hijacking the screen, so four sessions do not turn you into a single-threaded approval router.
- The board — who owns what, what is blocked, what landed — is the source of truth, not the scrollback.

Isolation still belongs to git worktrees. The office replaces the alt-tab archaeology, not git.

```bash
theboringfloor --demo
```

The goal was never more agents. It is still being able to say, at 4pm, who is doing what — whichever brain they brought.

## Further reading

- [Introducing Google Antigravity](https://antigravity.google/blog/introducing-google-antigravity) and the [Google Developers announcement](https://developers.googleblog.com/build-with-google-antigravity-our-new-agentic-development-platform/) — the Manager surface and Artifacts, from the source
- [Claude Code](https://claude.com/product/claude-code) and its [setup docs](https://code.claude.com/docs/en/setup) — Anthropic
- [Enabling Claude Code to work more autonomously](https://www.anthropic.com/news/enabling-claude-code-to-work-more-autonomously) — Anthropic, on checkpoints and the autonomy ladder
- [openai/codex on GitHub](https://github.com/openai/codex) — OpenAI; install, sign-in, and how the CLI meets Codex cloud
- [opencode docs](https://opencode.ai/docs/) — install, providers, plan mode, `/undo`
- [Running more than one coding agent at a time](/blog/running-multiple-coding-agents) — our longer field guide to the multi-agent afternoon
