# Blog writing style guide

For `content/blog/*.md`. Distilled from Anthropic Engineering, Cursor research posts, OpenAI Developer Blog, and Google DeepMind — then bent toward theboringoffice voice.

`WRITING.md` in this folder is the guide, not a post. `lib/blog.ts` skips it.

Sources reviewed (Firecrawl, 2026-08-24):

- [Building effective agents](https://www.anthropic.com/engineering/building-effective-agents)
- [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
- [A postmortem of three recent issues](https://www.anthropic.com/engineering/a-postmortem-of-three-recent-issues)
- [Agent swarms and the new model economics](https://cursor.com/blog/agent-swarm-model-economics)
- [Securely indexing large codebases](https://cursor.com/blog/secure-codebase-indexing)
- [Shell + Skills + Compaction](https://developers.openai.com/blog/skills-shell-tips)
- [Gemini Deep Think / scientific discovery](https://deepmind.google/blog/accelerating-mathematical-and-scientific-discovery-with-gemini-deep-think/)

---

## What the labs do (steal these)

| Lab | What they write | Structure habit | Voice |
| --- | --- | --- | --- |
| **Anthropic Engineering** | Lessons from shipping agents; mental models; postmortems | Thesis → definitions → when/when-not → numbered principles → summary | Calm “we,” plain words, diagrams as proof |
| **Cursor** | Empirical research: hypothesis → experiment → numbers → mechanism | Cold open with a result; sections named after the idea; charts carry argument | Dry, specific, no hype; footnotes for caveats |
| **OpenAI Dev Blog** | How-to for builders using their APIs | Mental model → numbered tips → named patterns → recap + docs links | Imperative, concrete, “use when / don’t use when” |
| **DeepMind** | Research narrative + taxonomy of claims | Claim → method → results with papers → careful non-overclaim | Formal, citation-heavy, credit-heavy |

Common patterns across all four:

1. **Lead with the finding or the problem**, not the company.
2. **Define terms early** (workflow vs agent, planner vs worker, skill vs prompt).
3. **When / when not** — explicit refusal to oversell.
4. **Numbered principles or tips** readers can reuse tomorrow.
5. **Concrete failure modes** before the fix.
6. **One product mention at the end**, or as a worked example — never the spine of the post.
7. **Acknowledgements / sources** when claiming hard facts.

---

## Our voice (theboringoffice)

We write like someone who ran the agents all afternoon and is warning a friend.

- **First person plural (“we”).** Floor talk, not press release.
- **Concrete over clever.** Commands, panes, diffs, queues — not “seamless orchestration.”
- **Problem first.** Scene the reader already lived (three terminals, one stuck on `chmod`).
- **Product last.** theboringoffice appears after the advice stands alone. If you delete our name, the post should still help.
- **No SEO slop.** No “In today’s rapidly evolving AI landscape.” No keyword stuffing. Titles are sentences a human would say.
- **Short sentences where the claim is sharp.** Longer ones only when unpacking a mechanism.
- **Honest ceilings.** “When this is the wrong move” sections are mandatory for how-tos.

Tone neighbors: Anthropic’s clarity + Cursor’s empiricism + our terminal grit. Not DeepMind’s conference voice. Not launch-blog hype.

---

## Post shapes (pick one)

### A. Field guide / how-to (default)

Like Anthropic “building effective…” + OpenAI tips posts.

1. Hook scene (2–4 short paras)
2. Name the real problem (not the feature)
3. 4–7 sections, each one job + one concrete fix
4. “When this is wrong”
5. Optional: how we encode it in the product (≤1 short section)
6. Links / further reading

Examples we already have: `running-multiple-coding-agents.md`, `watching-subagent-work.md`.

### B. Engineering deep dive

Like Cursor indexing / Anthropic multi-agent research.

1. Result or constraint in the first screen
2. Mental model (diagram or tight analogy)
3. Mechanism section(s)
4. Failure modes + mitigations
5. Numbers if you have them; otherwise skip charts — don’t invent
6. What you’d change next

### C. Postmortem / incident

Like Anthropic’s three-bugs post.

1. Plain statement of impact + what it was *not*
2. Timeline
3. Each failure, isolated
4. Why detection lagged
5. What you’re changing
6. How readers can help (feedback paths)

Use sparingly. Only when we broke something real.

### D. Research / experiment writeup

Like Cursor swarm economics.

1. Hypothesis + task
2. Setup (controls, what was withheld)
3. Results with one primary metric
4. Deep dive into *why* the numbers look that way
5. Caveats / footnotes
6. Artifact link if public

---

## Titles & frontmatter

**Title**

- Prefer a claim or a tension: “You launched the subagents. Now you have to watch them.”
- Avoid: “Announcing X”, “Introducing our revolutionary…”, “N Ways to…”
- Max ~70 chars when possible; okay to go longer if the sentence is the point.

**Description** (meta)

- One sentence: problem + what the reader leaves with.
- No brand name required.
- Matches the hook; doesn’t spoiler the whole post.

**Categories**

- Use existing tags (`AI Agents`, `Engineering`, `Office`, …). Don’t invent a tag per post.

**Featured**

- Only if the post teaches something durable (field guide / deep dive), not a minor changelog.

---

## Section craft

- **H2 = job of the section**, not a tease. Good: “Isolate the files, or they will fight.” Bad: “A better approach.”
- **Open each section with the claim**, then evidence / steps.
- **Lists for decisions; prose for scenes.** Don’t bullet a story.
- **Code blocks** only when the reader should paste or recognize a command. Prefer real shells over pseudocode.
- **Bold** for a short label or a failure-mode name — never whole paragraphs.
- **Diagrams** when a topology or loop is the point (orchestrator → workers). Skip decorative art.
- **Tables** for taxonomies or when/when-not. Skip for narrative.

---

## Sentence-level rules

**Do**

- Name tools the reader uses: Claude Code, Codex, OpenCode, tmux, git worktrees.
- Prefer “you” for the reader’s moves; “we” for what we observed or built.
- Put the surprising word at the end of the sentence when you can.
- Cut throat-clearing: “It’s important to note that…”, “In order to…”

**Don’t**

- Anthropomorphize models as teammates without the ops cost (“your AI coworkers” is fine on marketing; in blog, show the queue and the merge).
- Promise autonomy without supervision.
- Cite benchmarks you didn’t run.
- Write “simple yet powerful,” “game-changer,” “unlock,” “delight.”
- End with a sales CTA paragraph. One quiet product pointer is enough.

---

## Product placement

Allowed:

- Closing section: “This is the ops model we built into theboringoffice…” with 2–4 bullets of behavior, not feature list.
- Inline: one sentence if a concrete UI matches the advice (“a work thread — diffs attached to one task”).

Forbidden:

- Hero that is only the product.
- Every section ending in “and that’s why you need us.”
- Comparison tables dunking on named competitors.

---

## Length & pacing

| Shape | Target |
| --- | --- |
| Field guide | 1,800–2,800 words — a read, not a card |
| Deep dive | 2,000–3,500 |
| Postmortem | as long as the facts need; no padding |
| Changelog / thin update | don’t blog it — ship a short note elsewhere |

Outline-length posts read as slop even when the claims are right. Scene, mechanism, worked afternoon, failure names, when-not. If you can tweet the whole post, it is not a post.

Readers skim. Front-load. Mid-post should still make sense if they jump to an H2.

---

## Checklist before publish

- [ ] First paragraph works with brand name removed
- [ ] At least one “when not / don’t” beat
- [ ] Every H2 teaches; none are filler
- [ ] Commands / paths are copy-paste real
- [ ] Product section ≤ ~15% of length (or absent)
- [ ] Description ≠ title restated with synonyms
- [ ] No invented metrics
- [ ] Links to primary sources where we lean on others’ claims (Anthropic docs, papers, etc.)

---

## Quick templates

### Hook (field guide)

```text
[Scene the reader already did — 2–3 sentences.]

[Name the failure mode in one line.]

[This is not a model problem. It is an ops / attention / tooling problem.]
```

### Principle section

```text
## [Imperative or causal title]

[Claim.]

[Why it breaks / what people try.]

[The boring fix — steps or commands.]

[One edge case or limit.]
```

### Close

```text
## [Optional: how we encode this]

[2–4 behaviors, no pitch deck.]

## Further reading

- [Primary source]
- [Related post]
```

---

## What we take from each lab (one line)

- **Anthropic** → define terms; when/when-not; numbered lessons; show the diagram of the system.
- **Cursor** → lead with a measured result; name failure modes; let charts argue; footnotes for honesty.
- **OpenAI** → mental model up top; tips as routing rules; patterns you can name and reuse.
- **DeepMind** → don’t overclaim; taxonomize contribution level; cite; credit collaborators.

We keep the first three’s *craft*. We keep DeepMind’s *honesty about claims*. We keep our floor voice.
