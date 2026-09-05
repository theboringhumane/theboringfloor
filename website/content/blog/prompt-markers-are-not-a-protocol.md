---
title: "Prompt markers are not a protocol"
description: "Fenced directives can carry an agent instruction until parsing, ordering, scrubbing, and return values turn the prompt into an unreliable API; here is the boundary to use instead."
date: "2026-09-05"
author: "theboringfloor team"
categories: ["Release", "Engineering"]
featured: true
---

You ask an agent to put a plan in the pane beside the chat. It writes a fenced directive at the end of its reply. Something downstream has to find that fence, decide whether it arrived before or after the prose it describes, remove it before you read it, and turn the remaining text into an action.

That works right up until the directive is malformed, repeated, quoted in an explanation, or arrives after the reply that was meant to depend on it. Then the model has emitted a tiny API call through a channel built for human prose.

We have shipped that pattern. Prompt markers still work in the office. They are useful glue when the only thing an agent can emit is text. But they are a fragile contract, and pretending otherwise makes the failure look like a model problem when it is a protocol problem.

## A marker that reaches the visible reply already failed

A marker is a fenced directive the model emits into its answer. The office parses it, performs the requested UI action, and scrubs the directive from the reply you see. That last step matters. You came for a plan or a status update, not an implementation detail leaking into the transcript.

The whole arrangement puts several jobs on one piece of text:

- **Parsing:** recognize the marker without mistaking an example, a quote, or ordinary Markdown for an instruction.
- **Ordering:** decide whether the marker runs before the prose, after it, or between two visible messages.
- **Scrubbing:** remove only the instruction, while preserving the answer around it.
- **Result delivery:** tell the model what happened in a form it can use next.

The first three are brittle because natural-language output is not a stable wire format. A fence can be incomplete. A response can contain two candidates. Streaming can expose a partial block before the parser knows what the whole answer says. A sanitizer can eat text the reader needed, or leave text that was supposed to be invisible.

The fourth is the deeper problem. After a marker fires, the model does not receive a typed result from the operation. It has more text. A sentence saying that a plan was presented is not the same thing as a result object that another tool call can inspect. The model has to infer its state from its own transcript, which is a bad place to keep state that affects what happens next.

## Directives made the prompt carry two languages

Prompts are for instructions and explanations. Protocols are for named operations, argument shapes, result shapes, and failures that a caller can distinguish from prose. Mixing both languages is tempting because it gets you moving without an integration boundary.

We noticed the cost when an agent needed to do more than decorate an answer. A plan belongs in the plan pane. A member-approved plan is a fact with a lifecycle. A transcript search needs a query and a bounded result, not a paragraph that another model has to interpret. These are operations.

Markers can express operations, but only by making the parser guess where the operation begins and ends. They also make ordering implicit. If an agent says it updated a plan and includes a marker, is that sentence evidence that the update succeeded, or merely its intention? A tool call has a less exciting but much better answer: wait for the result.

This is a useful dividing line beyond our own software. Use text for the thing a person should read. Use a tool when software must reliably act on the thing. The distinction gets more important as one action supplies the input to the next.

## Give the agent a named operation and a typed result

MCP gives an agent a tools surface instead of asking it to hide instructions in Markdown. A caller names a tool, sends arguments, and receives success or failure through the protocol. For example, an MCP client can call the office status tool with a normal `tools/call` request:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"office_status","arguments":{}}}
```

That request is not prose that a second parser has to discover and delete. Its boundary is explicit. If a plan-writing tool fails because the office is not live, the caller receives a failure rather than a confident sentence written before the action completed.

Typed does not mean that every value is elaborate. `office_status` takes no arguments. `transcript_search` takes a query. The important part is that the caller knows which operation it asked for, what it supplied, and whether the operation returned. That makes retries, branching, and error handling possible without teaching an answer parser every new directive the product grows.

## The six tools are small on purpose

We shipped `thefloor_mcp`, a local MCP server, with six operations. They are the office state an agent can use without trying to reconstruct it from visible chat.

| Tool | Arguments | What it does | Availability |
| --- | --- | --- | --- |
| `plan_present` | `{text}` | Presents a plan draft in the plan pane. | Running office required. |
| `plan_update` | `{text}` | Updates that plan draft. | Running office required. |
| `plan_get_approved` | `{}` | Reads the member-approved plan. | Live office, otherwise on-disk snapshot. |
| `transcript_read` | `{limit?}` | Reads recent transcript messages. | Live office, otherwise on-disk snapshot. |
| `transcript_search` | `{query, limit?}` | Searches this project's transcript. | On disk; current project only. |
| `office_status` | `{}` | Reports whether the office is live, its backend, and message counts. | Live office, otherwise on-disk snapshot. |

The narrowness is deliberate. We did not expose a general remote-control surface, and browser tools are not exposed over MCP. A tool should exist because a caller has a reliable job for it, not because a local process could theoretically click every part of the UI.

The boundaries matter on reads too. `transcript_search` is current-project only. The on-disk transcript holds a recent tail capped at 200 messages, not full history and not a cross-project memory system. An agent that needs older context should be told that it did not find it, rather than being allowed to imply a search was comprehensive.

## Keep the UI loop on its own thread

An MCP server is a separate process boundary from the office UI. The running office listens only on loopback, at `127.0.0.1` on an ephemeral port. The server discovers that endpoint through a `0600` file at:

```text
~/.theboringfloor/projects/<dirhash>/control.json
```

The connection is bearer-token protected. That makes this a local, member-owned integration point rather than an unauthenticated port that any nearby process can casually drive.

There is another boundary that is easier to miss. The office UI has a single goroutine. The HTTP layer never touches it directly. A request becomes an event on the UI loop; the answer comes back through a correlated reply. The networking code can wait for the response, but it cannot race the renderer or mutate office state from the side.

This is not the kind of detail that makes a launch diagram pretty. It is the detail that prevents a convenience API from becoming a second, unsynchronized way to operate the application.

## A draft is not an approval

The two write tools are intentionally limited. `plan_present` and `plan_update` put a draft into the plan pane. They do not execute work. They do not approve the plan. With no running office, both return an error; there is no offline write fallback waiting to surprise you later.

That limitation is product policy, not missing plumbing. A member reviews the draft and presses `ctrl+x` twice to approve it. An agent can prepare a proposal, but it cannot turn its own proposal into authorization by emitting a tool call.

We wrote before that a permission is not a question. This is the same separation applied to plans. Presenting a draft is an action with a clear owner. Approval is a deliberate human decision about what should happen next. If a tool surface turns those into the same event, it makes a decision look like transport.

## Offline reads are a recovery path, not a memory promise

Some reads can still be useful when the office is closed. `plan_get_approved` and `transcript_read` can use the on-disk snapshot when there is no live office. That lets a new agent orient itself before the UI is running, and it avoids making all context disappear at process exit.

It also has a ceiling. The snapshot is not the live application, and a transcript tail is not an archive. The write tools do not inherit this behavior because a stale snapshot is not a safe place to stage UI mutations. The search tool stays scoped to the current project for the same reason: a result set should not quietly blend unrelated worktrees just because their files happen to live on the same machine.

When you design local agent integrations, name these fallback rules before you add the convenience path. "Works offline" is not a feature description until you say which data, how recent, and what cannot happen.

## When this is the wrong move

If you never use plan mode, the plan tools buy you nothing. Keep the workflow you already supervise instead of adding an MCP server because a release note mentioned one.

If your agent client has no MCP support, this path is unavailable. Prompt markers still work; they are not being removed to force a migration. Use the interface your agent can actually call, and do not build a bridge just to admire the bridge.

If what you need is browser automation, this is also the wrong surface. Browser tools are not exposed here. If you need full transcript history or context across projects, do not treat the recent 200-message, current-project tail as a substitute. It will give you a bounded local view, not an archive.

And if the work needs a product decision, no tool call should make that decision disappear. A plan draft can arrive through MCP. The member still reviews it and presses `ctrl+x` twice. That friction is where authorship lives.

## A quiet path into the office

`thefloor_mcp` ships in the same release archive as the `theboringfloor` binary. It registers itself in your global OpenCode configuration and, when the `claude` CLI is present, in Claude Code too. The server is an additional first-class path alongside prompt markers, not a replacement for them.

For the tool details and setup notes, see the [MCP server documentation](/docs/mcp-server). The larger rule is portable: once an agent needs to change or read application state reliably, stop asking prose to impersonate a protocol.
