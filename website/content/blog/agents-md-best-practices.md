---
title: "AGENTS.md best practices: the agent reads it before your code"
description: "AGENTS.md, CLAUDE.md, and their cousins decide what your coding agent knows before it touches a file — the landscape, the numbers, and the practices that keep the repo's second README from lying."
date: "2026-08-30"
author: "theboringfloor team"
categories: ["AI Agents", "Engineering"]
featured: true
---

Watch a coding agent start work on your repo sometime. Before it reads a line of your code, it reads a file you probably wrote in ninety seconds: `AGENTS.md`, `CLAUDE.md`, `.cursor/rules` — whichever flavor your tool of choice looks for. That file is the brief. Every edit the agent makes this afternoon is downstream of what you put in it, and every session starts cold, so the file is the only onboarding the agent ever gets.

That makes the instruction file the repo's second README — the one written for the reader who actually follows instructions. And most of them are bad. Stale commands. Walls of prose. Conventions the linter already enforces. One we reviewed last month contained a database password, sitting two lines above "always run the tests."

This post is the version we wish more repos had: what these files are, who reads which one, and the practices that separate a file that steers the agent from a file that just costs tokens.

*Updated August 2026 — file formats, load rules, and every number below re-checked against current vendor docs and primary sources.*

## The second README got big while nobody was watching

Instruction files are not a niche habit anymore, and the numbers are not close.

The [2025 Stack Overflow Developer Survey](https://survey.stackoverflow.co/2025/ai) found 84% of respondents using or planning to use AI tools in their development process, up from 76% the year before, with 51% of professional developers using them daily. Every one of those tools has to be told how your repo works, every session, from scratch. The [AGENTS.md site](https://agents.md/) puts the convention's own adoption at over 60,000 open-source projects at time of writing, and its [GitHub code search](https://github.com/search?q=path%3AAGENTS.md+NOT+is%3Afork+NOT+is%3Aarchived&type=code) is the live count. OpenAI's main repo alone carried 88 nested `AGENTS.md` files when the site documented it.

Anthropic's [2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report) frames why this file matters more every quarter: "software development is shifting from writing code to orchestrating agents that write code." Orchestration starts with instructions. The instruction file is where instructions live.

But here is the catch, and it is the reason for everything below. The most-cited study on these files — [Gloaguen et al., ETH Zurich and LogicStar.ai](https://arxiv.org/abs/2602.11988), *Evaluating AGENTS.md* — benchmarked context files across SWE-bench Lite and AGENTbench with several coding agents. LLM-generated instruction files *reduced* task success by about 3% on average while raising inference cost by over 20%. Developer-written files helped — by about 4%. Read that again: the same filename, opposite signs. The difference was not the format. It was whether a human had decided what deserved to be in the file.

One study, specific harnesses, margins of a few points — treat it as a warning, not a law. The warning: an instruction file is leverage in both directions. Write it like it matters, because the agent assumes it does.

## What these files are

**`AGENTS.md` is the cross-vendor convention.** Plain markdown, repo root, no required fields. It [emerged from](https://agents.md/) collaborative work across OpenAI Codex, Amp, Jules from Google, Cursor, and Factory, and is now stewarded by the [Agentic AI Foundation](https://aaif.io/) under the Linux Foundation. The pitch is right there on the site: README files are for humans; `AGENTS.md` is the predictable place agents read first. Codex, Cursor, Jules, the Copilot coding agent, opencode, Zed, Aider, and a growing list of others all read it. In monorepos you can nest it — the file closest to the edited code wins.

**`CLAUDE.md` is Claude Code's project memory.** Anthropic's [memory docs](https://code.claude.com/docs/en/memory) describe a small hierarchy, loaded broadest to most specific: a managed-policy file deployed by IT, your personal `~/.claude/CLAUDE.md`, the project `./CLAUDE.md` (or `./.claude/CLAUDE.md`) shared through git, and a gitignored `CLAUDE.local.md` for personal sandbox notes. Files in your working directory and every directory above it load at launch, concatenated root-down so the closest file is read last; files in subdirectories load on demand when Claude touches code underneath them. Two details worth knowing before you write one. Anthropic recommends keeping it under roughly 200 lines — it rides the context window of every session, and longer files reduce adherence. And `CLAUDE.md` content arrives as a user message, not a system prompt: it steers behavior, it does not enforce it. Enforcement is what hooks and permissions are for.

**The `@` import is the feature people miss.** A `CLAUDE.md` can pull in other files with `@path/to/file` anywhere in the file. Relative paths resolve against the file containing the import, imports recurse up to four hops, and parsing skips fenced code blocks. Import a path outside the working directory — say `@~/.claude/my-instructions.md` — and Claude Code shows a one-time approval dialog, because that import is how a repo you cloned would smuggle instructions into your session.

**Everything else is a vendor dialect of the same idea.** Gemini CLI reads `GEMINI.md` hierarchically — global, project, and just-in-time subdirectory files, with its own `@` imports — and can be pointed at `AGENTS.md` instead via `context.fileName` in `.gemini/settings.json`. Copilot reads `.github/copilot-instructions.md`. Cursor reads `.cursor/rules/`. Windsurf, post-merger, is migrating its rules to `.devin/rules/`.

## CLAUDE.md vs AGENTS.md vs the rest of the neighborhood

The landscape, compressed:

| File | Who reads it | Format | Nesting & scoping | Notes |
| --- | --- | --- | --- | --- |
| `AGENTS.md` | Codex, Cursor, Jules, Copilot coding agent, opencode, Zed, Aider, more | Plain markdown, any headings | Nested files; the closest one to the edited file wins | Cross-vendor convention, stewarded by the [Agentic AI Foundation](https://aaif.io/) |
| `CLAUDE.md` | Claude Code | Markdown with `@path` imports; `.claude/rules/` for path-scoped extras | cwd plus ancestors at launch; subdirectory files load on demand; user and managed-policy scopes | Claude Code's project memory; target under ~200 lines; shim it with `@AGENTS.md` if the canonical file is AGENTS.md |
| `GEMINI.md` | Gemini CLI | Markdown with `@/path` imports | Global `~/.gemini`, project, and just-in-time subdirectory scans, concatenated | Default context filename; repointable to `AGENTS.md` in [settings](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/gemini-md.md) |
| `.cursor/rules/*.mdc` | Cursor | Markdown plus YAML frontmatter (`description`, `globs`, `alwaysApply`) | Four activation modes: always, intelligent, glob-matched, manual `@`-mention | Plain `.md` files there are ignored; the legacy `.cursorrules` dotfile is gone from the [current docs](https://cursor.com/docs/rules) — migrate |
| `.github/copilot-instructions.md` | GitHub Copilot chat, code review, coding agent | Plain markdown | Repo-wide; path-scoped siblings in `.github/instructions/*.instructions.md` with `applyTo` | The coding agent offers to [draft one](https://docs.github.com/en/copilot/customizing-copilot/adding-custom-instructions-for-github-copilot) on your first PR |
| `.windsurf/rules/*.md` | Windsurf Cascade (now Devin) | Markdown with activation frontmatter, 12,000 characters per file | Always, manual, and glob modes; legacy `.windsurfrules` still read | New setups prefer `.devin/rules/*.md` per the [Cascade docs](https://docs.devin.ai/desktop/cascade/memories) |

Two rows deserve a second look. On `.cursorrules` vs `.cursor/rules`: if your repo still has the single dotfile at the root, it is a legacy artifact — Cursor's rules page now lists project rules, user rules, team rules, and `AGENTS.md`, and nothing else. Split the dotfile into scoped `.mdc` rules or fold it into `AGENTS.md` and move on.

On `CLAUDE.md` vs `AGENTS.md`: this is not a fight you need to have. Claude Code reads `CLAUDE.md` and not `AGENTS.md` — but the [documented pattern](https://code.claude.com/docs/en/memory) is a two-line shim:

```markdown
@AGENTS.md

## Claude Code
Use plan mode for changes under `src/billing/`.
```

The import keeps one source of truth; the lines below it hold Claude-specific behavior. A symlink (`ln -s AGENTS.md CLAUDE.md`) works too when you need nothing Claude-specific — everywhere except Windows, where the import avoids the privilege dance. What you should not do is maintain both files as full copies. They will drift, and the drift is silent.

## What belongs in the file, and what doesn't

A good instruction file answers one question: what would you tell a strong contractor on day one that they could not learn by reading the code for an hour? Everything that fails that test is padding the agent pays for on every session.

**Belongs:**

- **Build, test, and verify commands.** The exact invocations, with the flags that matter. This is the file's center of gravity: agents execute what you write down, so write down things that can be executed and checked. Run, don't describe.
- **An architecture map in lines, not paragraphs.** One line per directory or package, saying what lives there and what it must not do. The agent can read code; what it cannot read is which boundaries are load-bearing.
- **Conventions with an example each.** "Handlers return structured errors with a `code` field — see `internal/api/errors.go`" beats three paragraphs on error philosophy. Point at canonical files instead of copying their contents; copies rot.
- **Trapdoors.** The test suite that needs Docker running. The directory that is generated and must never be edited. The migration that must be two PRs. Anything that has ever burned an agent — or a new hire — for an afternoon.
- **Review discipline.** What "done" means in this repo: which checks run before a change is offered, what a PR title looks like, whether tests for changed code are optional (they are not).
- **Explicit don'ts.** Files to never touch, commands to never run, dependencies to never add. Agents are optimistic; the don'ts are where optimism goes to be corrected.

**Doesn't belong:**

- **Anything a linter or CI already catches.** Formatting rules, import order, naming trivia. If a check can fail the build, it does not need prose. Cursor's own [rules docs](https://cursor.com/docs/rules) say it plainly: the agent already knows common style conventions — use a linter.
- **Duplicated documentation.** If `docs/architecture.md` says it, the instruction file should link to it, not restate it. Two copies means one is lying within a quarter.
- **Prose about the project.** History, motivation, marketing. The agent does not need to be inspired; it needs to run the right test.
- **Secrets.** This file goes into the context window of every session and over the wire to every provider you point at it. It is committed to git. Nothing with a blast radius belongs in it.
- **Vibes.** "Write clean code," "be careful with the database," "test thoroughly." Anthropic's guidance is specificity you can verify: "Run `npm test` before committing," not "test your changes." An instruction the agent cannot act on is a token tax with a morale problem.

## AGENTS.md best practices, numbered

The reusable core, in the order you will use it. Most of these apply verbatim to `CLAUDE.md`, `.mdc` rules, and the rest of the family — the filename changes, the physics don't.

**1. Start empty. Grow from observed mistakes.** Resist the urge to scaffold a comprehensive file on day one — and treat `/init`-style generators as a first draft you cut hard, not a finished artifact. The ETH study above found LLM-generated context files slightly *worse* than none. Both Cursor's docs ("add rules only when you notice Agent making the same mistake repeatedly") and Anthropic's ("Claude makes the same mistake a second time") prescribe the same loop: the file earns a line every time reality proves the line is needed. A file built from scars fits the repo. A file built from imagination fits a generic repo, which the agent already knows.

**2. Write commands, not descriptions.** "Test your changes" is a wish. "Run `go test ./internal/... -count=1`; a green run is part of done" is an instruction. Commands are checkable, the agent will actually attempt them — the AGENTS.md FAQ confirms agents execute the checks you list — and failures become legible instead of silent. Every claim in the file should survive the question: how would the agent verify this?

**3. Put the trapdoors in writing.** Every repo has them: the suite that flakes under parallel load, the generated file that looks hand-written, the local service that must be running. The agent will find them the expensive way, one failed run at a time, unless you spend three lines. Keep a running list; when a session dies to something surprising, the trapdoor section gets the commit.

**4. Budget the file like context, because it is context.** The whole file rides every session's window. Anthropic targets under ~200 lines for `CLAUDE.md`; Cursor suggests under 500 for a rule. Past a few hundred lines, adherence drops and the file starts competing with the actual task for attention. When it grows, split it: path-scoped rules (`.claude/rules/` with `paths:` frontmatter, `.mdc` with `globs:`) load only when the agent touches matching files, which is the correct unit of attention.

**5. In a monorepo, nest by directory.** One root file for repo-wide truth — how to build, how to run checks, the cross-cutting don'ts — plus a file per package for package-local truth. The precedence rule for `AGENTS.md` is simple: the file closest to the edited code wins, and an explicit prompt in chat overrides everything. Claude Code's variant concatenates ancestor files and reads the closest last; Cursor applies nested `AGENTS.md` the same way, with the more specific file taking precedence. Give each nested file an owner: the team that owns the package. OpenAI's 88-file repo is the extreme end of this, not the starting point.

**6. One canonical file, thin vendor shims.** Pick `AGENTS.md` as the source of truth — it is the only file the whole ecosystem reads — and make every vendor file a shim: `CLAUDE.md` opens with `@AGENTS.md` and adds only Claude-specific lines, Gemini CLI's `context.fileName` points at `AGENTS.md`, Aider gets `read: AGENTS.md` in its config. The alternative is three copies drifting apart, and you will not notice the drift. The agent will, and it will not tell you.

**7. Review changes to the file like code.** This file is the steering wheel for every agent session in the repo, which makes its pull requests higher-stakes than most code changes. Require review. Watch for edits that smuggle in scope ("while I'm here, agents should also…"), edits that weaken a don't, and edits written by an agent about how agents should behave — that last category needs the most scrutiny, not the least. We said it in the [2026 trends post](/blog/agentic-coding-2026): the file that steers the fleet is the last file to hand to the fleet.

**8. Test the file like code.** The file has one job: change what the agent does. So measure that. Watch a session fail at something, patch the file, re-run a similar task, and keep the line only if the failure does not repeat. The metric is boring and binary: the agent stops making the mistake. In Claude Code, run `/context` to confirm the file even loaded into the session before debugging anything fancier. A line of instruction that changes nothing is decoration, and decoration here costs tokens four times a day.

**9. Refresh generated sections mechanically.** Some sections legitimately describe moving parts — the package map, the command list. If you keep them, generate them: a small script or CI step that rewrites the section from source, so the file cannot silently diverge from the repo. If you cannot generate it, keep it out. Hand-maintained mirrors of reality rot on a schedule.

**10. Never let it lie.** A stale instruction is worse than no instruction, because the agent trusts the file over its own exploration. The wrong test command does not produce no testing — it produces a failing run the agent then tries to fix. When a command changes, the file changes in the same PR. When a section stops being true, delete the section. Absence is honest; the agent explores. A lie sends it somewhere with confidence.

## A worked example: one AGENTS.md for a Go TUI repo

This is a compressed but realistic root file for a terminal UI project in Go. The `<!-- -->` comments are annotations for you, the maintainer — a nice side effect of one Claude Code behavior: block-level HTML comments are [stripped](https://code.claude.com/docs/en/memory) before the file enters context, so maintainer notes cost the agent zero tokens.

```markdown
# AGENTS.md

<!-- Audience: coding agents. Humans start at README.md. -->

## Commands
- Build: `go build ./...`
- Test everything: `go test ./... -count=1`
- Test one package: `go test ./internal/panels -run TestFreeze -count=1`
- Lint: `gofmt -l .` must print nothing; `go vet ./...` must pass
- See the UI without a backend: `go run ./cmd/uitest --demo`

<!-- Every command above was run today. If one breaks, fix the file in the same PR. -->

## Map
- `internal/app` — event loop and routing; owns no rendering
- `internal/panels` — the visible panes; keep them pure: state in, frames out
- `internal/backend` — agent process adapters; one file per backend
- `cmd/` — entry points only, no logic

## Conventions
- Errors: wrap with `%w`, return structured errors from backends — see `internal/backend/errors.go`
- Tests: table-driven, no test network access, no sleeps under 500ms — see `internal/app/model_test.go`
- Commits: imperative, one concern, no "wip"

## Trapdoors
- `internal/panels/testdata/` is generated by `go generate ./...` — never edit by hand
- The full suite flakes if run with `-race` on macOS CI; run `-race` locally only
- PTY tests need a real terminal size; they skip themselves under `CI=true`

## Don'ts
- Don't add dependencies — ask first; the module file is reviewed line by line
- Don't run `go test ./...` with `-p` above 4; the lane tests fight over the display
- Don't edit files under `vendor/` or `testdata/`

## Review
- Done means: build green, targeted tests green, gofmt clean
- PR title: `[<package>] <what changed>`; the diff must not touch files outside the task
```

Seventy lines. Everything in it is executable, checkable, or a pointed finger at a file that shows the convention. There is no prose to skim, and nothing in it requires updating unless the repo itself changed.

## Nesting, imports, and the monorepo

The advanced moves are all about one problem: the file got big, and context is finite.

**Nesting** is the first move, and the tools mostly agree on the semantics. Root file for repo-wide truth; package files for package-local truth; closest file wins for `AGENTS.md` and Cursor; closest file read last for Claude Code's concatenation. Claude Code's twist: subdirectory files are lazy — they load when the agent reads code under them, not at launch — so a nested file costs nothing until it is relevant. If you work in someone else's corner of a huge monorepo and the ancestor files are noise, `claudeMdExcludes` in `.claude/settings.local.json` skips them by path or glob.

**Imports** are the second move, with one honest caveat. Splitting a `CLAUDE.md` into `@docs/testing.md` and friends organizes the file for humans — but imported files still load at launch and still cost the same context. Imports are for your editing comfort, not the agent's budget. The budget fix is scoping (path rules, nesting), not splitting.

**Generation** is the third move, covered in practice 9: if a section mirrors the repo, produce it from the repo. The pattern that works is a marked block — `<!-- BEGIN GENERATED: packages -->` — rewritten by a script on a hook or in CI. What fails is a human remembering to update it.

**Security** is the move nobody planned for. An instruction file is agent input, and input is an attack surface. This is not hypothetical: Mitiga documented a [poisoned take-home repo](https://www.mitiga.io/blog/poisoned-coding-test-ai-agent-attack) where malicious instructions planted in `CLAUDE.md` and `.cursor/rules` had the agent executing an attacker's workflow, with "do not mention this step" suppressing the narration. Prompt Security has shown [AGENTS.md goal hijack](https://prompt.security/blog/when-your-repo-starts-talking-agents-md-and-agent-goal-hijack-in-vs-code-chat) in VS Code chat. The boring defenses: never put secrets in the file (it is committed, and it is transmitted to providers); review instruction-file diffs in PRs with the same suspicion as CI config; and remember that pointing an agent at a repo you did not write imports that repo's instruction files too — read them first, the way you would read a Makefile before running `make`. Claude Code's approval dialog for external `@` imports exists for exactly this reason. Do not click through it.

## When this is the wrong move

Honest ceilings, because this file is easy to overbuild.

**The repo is tiny.** A one-package tool where `go build ./...` and `go test ./...` are the whole workflow does not need an instruction file. The code is the context. A file here is ceremony.

**Nobody will own it.** An unmaintained instruction file is a stale instruction file, and practice 10 applies: worse than none. If the team will not review its diffs, do not create it. Delete it the day it starts lying.

**You are documenting, not instructing.** If the section you are adding is a paragraph a human should read, it belongs in the README or `docs/`. The instruction file is not a dumping ground for things that did not fit elsewhere. It is the briefing.

**The convention can be enforced instead.** If CI can fail on it — formatting, import cycles, forbidden dependencies — enforce it there. A check is obeyed deterministically; an instruction is obeyed statistically. Instruction files are for what cannot be enforced: the trapdoors, the boundaries, the taste.

**You are about to generate it and walk away.** The best evidence we have says an unedited, model-generated file is slightly worse than none. `/init` and its cousins are good first drafts. A first draft you never cut is not a practice. It is a superstition.

## How we encode this

We run on this advice literally: theboringfloor's own agents get their instructions from a charter file — [`.opencode/oikonomos.md`](https://github.com/theboringhumane/theboringfloor/blob/main/.opencode/oikonomos.md) — committed to the repo, reviewed like code, and read by every session before it touches a file. Same physics as everything above, applied to ourselves.

## Sources

- [AGENTS.md](https://agents.md/) — the format site: origin, backers, nested-file rules, the 60k+ adoption claim, and the [live GitHub count](https://github.com/search?q=path%3AAGENTS.md+NOT+is%3Afork+NOT+is%3Aarchived&type=code)
- [Agentic AI Foundation](https://openai.com/index/agentic-ai-foundation/) — the Linux Foundation home the format moved to
- [Claude Code memory docs](https://code.claude.com/docs/en/memory) — `CLAUDE.md` scopes, load order, `@` imports, the `@AGENTS.md` shim, the 200-line guidance
- [Cursor rules docs](https://cursor.com/docs/rules) — `.mdc` frontmatter, activation modes, nested `AGENTS.md` support
- [Gemini CLI: GEMINI.md](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/gemini-md.md) — the hierarchical context file and its imports
- [GitHub Copilot custom instructions](https://docs.github.com/en/copilot/customizing-copilot/adding-custom-instructions-for-github-copilot) — `copilot-instructions.md` and path-scoped `.instructions.md`
- [Cascade memories and rules](https://docs.devin.ai/desktop/cascade/memories) — `.devin/rules`, `.windsurf/rules`, and the legacy `.windsurfrules`
- [2025 Stack Overflow Developer Survey: AI](https://survey.stackoverflow.co/2025/ai) — the 84% and 51%-daily adoption figures
- [2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report) — Anthropic, on the shift from writing to orchestrating
- [Evaluating AGENTS.md (Gloaguen et al., ETH Zurich)](https://arxiv.org/abs/2602.11988) — the study on generated vs. human-written context files
- [How a poisoned coding test turned an AI agent into an attacker](https://www.mitiga.io/blog/poisoned-coding-test-ai-agent-attack) — Mitiga, on instruction files as an injection surface
- [AGENTS.md and agent goal hijack in VS Code chat](https://prompt.security/blog/when-your-repo-starts-talking-agents-md-and-agent-goal-hijack-in-vs-code-chat) — Prompt Security
- [The code got cheap this year. The supervision didn't.](/blog/agentic-coding-2026) — our earlier post, where the fleet-steering line comes from
