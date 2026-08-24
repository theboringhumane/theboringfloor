---
title: "A permission is not a question"
description: "Shell allow, product decision, and status are three interrupts. One popup trains you to rubber-stamp a breaking change like it was git status. Here is how the mix-up happens, and how to unmix it."
date: "2026-08-24"
author: "theboringoffice team"
categories: ["Office", "Engineering"]
featured: true
---

The tenth prompt of the afternoon is a box. You hit `y` because you have hit `y` nine times.

Prompt one was `git status`. Two was `git diff`. Three was the linter. Four through eight were the test runner discovering it needed network for a fixture you forgot was remote. Nine was `chmod +x` on a script in `scripts/`. Ten was: *Should we remove the v1 `/users` route, or keep it behind a flag for a quarter?*

Same chrome. Same shortcut. Same muscle.

We have done this. We have also done the inverse: treated `ls` like a product meeting because the last box had been a product meeting, and now we were "being careful," which meant four agents sat idle while we composed a paragraph about listing a directory.

The model is not confused about the difference. The *interrupt* is. If every halt looks like a shell allow, two out of three kinds of halt will be handled wrong.

## The mix-up is trained, not accidental

Agent tools borrowed the oldest UI in Unix: a yes/no before a dangerous command. That is correct for `rm`. It is a terrible schema for everything else an agent might need you for.

So the product grows. "Allow always" appears, because you are tired. "Allow all this session" appears, because "always" felt like a gun. Then a designer adds a text field for "the agent has a question," and to save a component it goes in the same modal. Then "done" is a toast you must dismiss because otherwise you might miss it.

You now have one box that means policy, authorship, and notification. Your hands learn a single response. The cheapest response wins. That is `y`.

Cursor's swarm writeups and Anthropic's production notes keep circling the same ops fact: at human tempo, code review and standups work. At agent tempo, the interrupt budget is the product. If you spend it on `git status`, you will not have it for the API break.

## Three interrupts, three jobs

**Permission.** May this process run this command, write this path, talk to this host? You are the policy. The payload is a command line, a path, maybe a diff hunk. The answers are allow once, allow always (for a *pattern* you can state), reject, or park it and return. There is no essay. If you are writing an essay, it was a question.

**Question.** A fork in the product or the design. Default timeouts, flag names, "is this breaking," "do we support the old client." The payload is a decision. The answers are a sentence, a radio, a set of checkboxes. "Allow always" on a question is how you skip thinking. There is no always for "drop v1."

**Status.** Something finished, stalled, or claimed. The payload is a fact. The answer is you seeing it — or not, until you look. It should not take the keyboard. A badge, a board column, a line in an activity list. If status is a modal, you will hate the tool by Wednesday, and you will be right.

You should be able to tell which one you are in before you read the body. Command plus path plus agent name: permission. A sentence with a tradeoff: question. "Worker B finished `auth.go`": status. If you have to *decide* which kind it is, the UI already failed.

## A worked hour

11:02 — Writer agent wants `go test ./internal/auth`. Permission. You have a test allowlist. It should not have appeared. You add `go test` to the allowlist after the second identical ask, not the twelfth.

11:07 — Same agent: `curl https://api.stripe.com`. Permission, different class. Network. Once, maybe, if you are looking at the command. Never "always" unless you mean it.

11:11 — Reviewer: "The handler returns 500 on unknown role. 401 or 403?" Question. Radio. You answer 403. If this had been `y`/`n`, you would have approved a status code like a shell command.

11:14 — Test runner: tests green. Status. You do not press anything. You see a check on the task, or you don't, when you next look at the board.

11:18 — Writer: `rm internal/auth/legacy.go`. Permission. You read it. You reject it because the question at 11:11 did not authorize deleting the old router. The agent was collapsing a product decision into a file delete. That collapse is common. The permission UI should make the path large. The question UI should have happened *before* the delete, not as a side effect of `rm`.

If those four events share a component, 11:11 and 11:14 will be handled as if they were 11:02. That is how v1 dies on a Thursday because `y` was in your fingers.

## Allowlists are for permissions only

Write the pattern in language you would put in a code review.

Good: `git status`, `git diff`, `gofmt`, `go test ./...`, the project linter.

Bad: "anything in this repo." That is how `scripts/deploy.sh` rides with `gofmt`.

Worse: "allow always" on a sentence. There is no glob for product judgment.

The tell is the one we keep repeating because it keeps being true: if you already type `y` without reading, it should not have been a prompt. Promote that command. Do not promote the *mood* of approval.

Parked permissions need a list. `/perm` or a queue index (`1 of N`) is enough. "I'll get to it" with no list is how `chmod` sits in pane 3 for twenty minutes while you argue with a reviewer about names.

## Questions need a shape that is not a keystroke of habit

Text, when the answer is a string (the flag name). Radio, when the answer is one of a known set. Checkboxes, when several constraints apply ("keep v1," "add deprecation header," "log hits"). Confirm, when you are about to do something irreversible *and* you already know what it is.

A question should carry the task, the relevant diff, and the constraint. "Breaking or not?" with no snippet is how you get a confident wrong answer. "Breaking or not?" next to the router diff is a decision you can own.

Reopen parked questions on purpose. If the only copy of the decision is in a collapsed thinking block, you will make it twice, differently.

## Status is not a conversation

"I'll take the flaky test" in chat is gone when you scroll. A board cell is not gone.

Done should be loud on the board and quiet on the keyboard. Stalled should be a blocked column, not a modal that says "still running." Claimed should have a name that matches the branch, not `claude-3`.

This is why people say agents "don't save time" after a week. They saved generation. They spent it again reconstructing state from prose. Prose is a bad database.

## When one box is enough

One agent, one trusted directory, you watching the pane, a five-minute rename. Taxonomy is overhead. Approve in place. Stop.

Unattended overnight: a queue that waits on you is a halt. Use a tight allowlist and a sandbox, or do not leave it running. A question left unanswered at 2am is a stuck worker, not a thoughtful pause.

If you cannot write down what "allow always" matches, do not press it. That rule is dull and it beats a lot of incident writeups.

## How we encode this

theboringoffice splits them on purpose. Permission queue: `1 of N`, `y` / `a` / `n` / `esc`, `/perm` to return. Questions: their own popover, `/question` to reopen. Activity and the board carry status so "done" is not a dialog.

The split is the feature. The floor is how we draw who is waiting. If we ever put a product question in the permission chrome, we will have failed the taxonomy we are telling you to demand of every tool, including ours.

## Further reading

- [Don't let approvals steal the screen](/blog/tool-router-beta)
- [You launched the subagents. Now you have to watch them.](/blog/watching-subagent-work)
- [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents) — Anthropic, on not adding complexity that does not pay
