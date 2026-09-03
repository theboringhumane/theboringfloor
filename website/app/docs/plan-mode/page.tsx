import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Plan mode',
  description:
    'ctrl+p flips the office into a read-only planning pass: agent plan markers fill the pane as drafts, you review or edit them, and ctrl+x twice approves a plan for the build agent.',
  alternates: {
    canonical: '/docs/plan-mode',
  },
  openGraph: {
    title: 'Plan mode · theboringfloor',
    description:
      'ctrl+p flips the office into a read-only planning pass: agent plan markers fill the pane as drafts, you review or edit them, and ctrl+x twice approves a plan for the build agent.',
    url: `${SITE_URL}/docs/plan-mode`,
    type: 'website',
  },
}

const toggleKeys = [
  { combo: 'ctrl+p', action: 'toggle plan/build' },
  { combo: '[plan]', action: 'statusbar badge' },
]

const gateKeys = [{ combo: 'click pane', action: 'scratch from the starter template' }]

const presentKeys = [
  { combo: 'click', action: 'enter the plan editor' },
  { combo: 'esc', action: 'back to chat, edits kept' },
]

const approveKeys = [
  { combo: 'ctrl+x twice', action: 'confirm approval → build agent' },
  { combo: 'ctrl+p', action: 'exit to build, send nothing' },
]

const limits = [
  {
    label: 'when ctrl+p refuses',
    body: 'The toggle will not flip while you are shell-captured in the terminal tab, or while a question or permission float is open. Answer the ask first, then plan.',
  },
  {
    label: 'hollow plans',
    body: 'ctrl+x refuses an empty buffer and the untouched starter template alike — the office will not hand the crew a plan nobody wrote. Both refusals land as a notice, never a silent drop.',
  },
  {
    label: 'what persists',
    body: 'The latest approved plan survives across sessions, bounded to 20,000 runes. Drafts and updates remain drafts; they cannot overwrite that approved version until you approve them.',
  },
  {
    label: 'approval is not exemption',
    body: 'Approving a plan changes who gets the prompt. It does not approve permissions: every ask the build run raises still stacks as 1 of N with your call on it.',
  },
]

const keepReading = [
  {
    href: '/docs/permissions-and-questions',
    kicker: 'next',
    title: 'Permissions and questions',
    blurb: 'Approval flips the mode — the asks the build run raises still go through this queue.',
  },
  {
    href: '/docs/chat-and-threads',
    kicker: 'related',
    title: 'Chat and work threads',
    blurb: 'The chat that keeps focus while the plan pane presents in the floor slot.',
  },
  {
    href: '/docs/backends',
    kicker: 'related',
    title: 'Backends',
    blurb: 'Plan agent and build agent ride the same pinned transport — opencode or Claude Code.',
  },
  {
    href: '/docs/getting-started',
    kicker: 'start',
    title: 'Getting started',
    blurb: 'Tour the office in demo mode and try ctrl+p yourself.',
  },
]

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <div className="mt-10 overflow-hidden border border-border bg-(--shot-frame)">
      <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
        <span className="size-2.5 rounded-full bg-destructive/70" />
        <span className="size-2.5 rounded-full bg-chart-4/70" />
        <span className="size-2.5 rounded-full bg-chart-2/70" />
        <span className="ml-2 font-mono text-xs text-muted-foreground">{caption}</span>
      </div>
      <img
        src={src}
        alt={alt}
        width={5086}
        height={2896}
        loading="lazy"
        className="shot-img block h-auto w-full"
      />
    </div>
  )
}

function KeyChips({ keys }: { keys: { combo: string; action: string }[] }) {
  return (
    <div className="mt-6 flex flex-wrap items-center gap-3">
      {keys.map((k) => (
        <span
          key={k.combo}
          className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs"
        >
          <span className="text-accent">{k.combo}</span>
          <span className="text-muted-foreground">{k.action}</span>
        </span>
      ))}
    </div>
  )
}

export default function PlanModePage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Plan mode</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              Plan first. Then let the crew build.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              One key flips the office from building to planning. The boss plans read-only, explicit
              agent markers fill the floor-slot pane as drafts, your edits survive the next reply,
              and a two-step approval hands your plan to the build crew.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The toggle</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              ctrl+p flips the mode, not the conversation.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <span className="font-mono text-foreground">ctrl+p</span> moves the office between{' '}
              <span className="font-medium text-foreground">build</span> (the default) and{' '}
              <span className="font-medium text-foreground">plan</span> — a mode toggle only. The
              statusbar wears a <span className="font-mono text-foreground">[plan]</span> badge,
              prompts ride the read-only plan agent, and chat keeps focus: you keep talking the
              way you already do. The plan pane stays hidden until it has content — an empty mode
              opens nothing.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The status line keeps the mode honest:{' '}
              <span className="font-mono text-foreground">
              plan · boss plans read-only · ctrl+p exits · ctrl+x twice approves a draft
              </span>
              . Read-only means the planning pass cannot run tools against your tree — it thinks,
              it drafts, it waits.
            </p>
            <KeyChips keys={toggleKeys} />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Agent markers</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The boss names the plan it wants you to review.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The boss uses a multiline <span className="font-mono text-foreground">plan-present</span>{' '}
              block to fill the existing plan pane, and a multiline{' '}
              <span className="font-mono text-foreground">plan-update</span> block to refresh its
              draft. Ordinary chat stays in chat. These are agent-only protocol markers: they make a
              draft visible for your review; they do not execute work and they do not approve it.
            </p>
            <pre className="mt-6 max-w-2xl overflow-x-auto border border-border bg-card p-4 font-mono text-sm leading-relaxed text-foreground"><code>{`⟦plan-present⟧
# Goal
Add the requested capability.

# Steps
1. Inspect the current flow.
2. Make the focused change.
⟦/plan-present⟧`}</code></pre>
            <pre className="mt-4 max-w-2xl overflow-x-auto border border-border bg-card p-4 font-mono text-sm leading-relaxed text-foreground"><code>{`⟦plan-update⟧
# Goal
Add the requested capability with the clarified edge case.
⟦/plan-update⟧`}</code></pre>
            <KeyChips keys={gateKeys} />
            <Shot
              src="/shots/docs/plan-gated.png"
              alt="theboringfloor plan mode with a non-plan-shaped boss reply: the chatter stays in chat and the plan pane keeps its last plan with a dim note"
              caption="before — gated: chatter in chat, the pane holds its last plan"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Presented</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Once it presents, the draft is yours to edit.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When an agent plan block lands, the hint swaps to{' '}
              <span className="font-mono text-foreground">
                plan · click to edit · ctrl+x twice approves → build · ctrl+p exits
              </span>
              . Click in and edit until it reads like your plan, not the office&apos;s guess. The pane
              remains a draft until you approve it; edits latch the pane as yours, and a fresh boss
              reply leaves your version untouched (one dim note:{' '}
              <span className="font-mono text-foreground">
                boss replied — your edited plan kept
              </span>
              ). <span className="font-mono text-foreground">esc</span> steps back to chat with
              the draft intact, and mermaid diagrams render inline in the read-only view.
            </p>
            <KeyChips keys={presentKeys} />
            <Shot
              src="/shots/docs/plan-presented.png"
              alt="theboringfloor plan mode with a presented plan: markdown plan in the floor-slot pane, a rendered mermaid diagram, and the click-to-edit hint"
              caption="after — presented: markdown + mermaid, click to edit, ctrl+x twice approves"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Approve</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              ctrl+x twice hands the plan to the build agent.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The first <span className="font-mono text-foreground">ctrl+x</span> asks you to
              confirm. Press <span className="font-mono text-foreground">ctrl+x</span> again to
              approve: the office sends{' '}
              <span className="font-mono text-foreground">
                Approved plan — implement it exactly as specified:
              </span>{' '}
              plus your plan body to the build agent and flips the mode back to build, badge off.
              The approved plan is retained across sessions, up to 20,000 runes. An empty buffer —
              or the untouched starter template — is refused with a notice, not sent as hollow
              instructions.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Approval changes who gets the prompt — nothing else. Every permission ask the build
              run raises still stacks in the{' '}
              <Link
                href="/docs/permissions-and-questions"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                permission queue
              </Link>{' '}
              with your call on it, and the work itself shows up in the{' '}
              <Link
                href="/docs/chat-and-threads"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                chat as work threads
              </Link>
              .
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A later <span className="font-mono text-foreground">plan-present</span> or{' '}
              <span className="font-mono text-foreground">plan-update</span> is only a new draft:
              it never replaces your retained approved plan by itself. When the boss needs that
              decision again, it sends this own-line marker; the office returns the latest approved
              plan, not an unapproved draft.
            </p>
            <pre className="mt-6 max-w-2xl overflow-x-auto border border-border bg-card p-4 font-mono text-sm leading-relaxed text-foreground"><code>{`⟦plan-get-approved⟧`}</code></pre>
            <KeyChips keys={approveKeys} />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Honest edges</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              What this doesn&apos;t do yet.
            </h2>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {limits.map((l) => (
                <div key={l.label} className="bg-background p-6">
                  <p className="font-mono text-xs uppercase tracking-wider text-accent">
                    {l.label}
                  </p>
                  <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{l.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {keepReading.map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  className="flex flex-col gap-2 bg-background p-6 transition-colors hover:bg-secondary"
                >
                  <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                    {l.kicker}
                  </p>
                  <p className="font-medium text-foreground">{l.title}</p>
                  <p className="text-sm leading-relaxed text-muted-foreground">{l.blurb}</p>
                </Link>
              ))}
            </div>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
