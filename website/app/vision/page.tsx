import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

export const metadata: Metadata = {
  title: 'Vision | theboringfloor',
  description:
    'Why we believe agents need a floor, a roster, and memory — because a prompt box shows you a spinner, not the work.',
}

const coworkers = [
  {
    id: '01',
    name: 'the manager',
    role: 'The boss',
    body: 'The one you talk to. Dispatches work to the team, keeps the shift coherent, and takes your messages mid-turn — interruptible, batchable, always reachable.',
  },
  {
    id: '02',
    name: 'theboringcto',
    role: 'The CTO',
    body: 'Reviews every drained batch before it lands. The adult in the room for the calls that matter.',
  },
  {
    id: '03',
    name: 'hr',
    role: 'People & health',
    body: 'Watches team health and how the work feels on the floor — even a team of agents needs that.',
  },
  {
    id: '04',
    name: 'tekton',
    role: 'The developers',
    body: 'Claims tickets off the board, opens work threads you can expand right in chat, and walks to the tea machine between batches.',
  },
  {
    id: '05',
    name: 'scouts, reviewers, runners',
    role: 'The supporting cast',
    body: 'Scouts recon the codebase before work starts, reviewers weigh in on every return, and runners carry the work until it is done.',
  },
  {
    id: '06',
    name: 'the floor itself',
    role: 'Ambient life',
    body: 'Desks, blinking server LEDs, the tea machine, light and dark themes, and sound. An office that feels inhabited, not rendered.',
  },
]

const etiquette = [
  'Message the boss mid-turn — your words batch into one composed send, not a lost line in the scrollback.',
  'The concierge answers instantly while the boss is busy, so you never talk to a wall.',
  '/stop aborts on the spot; free-send keeps you typing while the office is mid-turn.',
  'Permission asks stack as 1 of N instead of stealing the screen — approve, always-allow, or reject.',
]

const ceilings = [
  'You want one-shot answers, not an ongoing system. A plain chat window is fine for that — use it.',
  'You live in an IDE GUI, not a terminal. The office is a terminal app, and it will not pretend otherwise.',
  'You need deterministic structured-output pipelines, not coworkers you supervise.',
  'You have one agent, one file, one question — a roster and a board are overhead there.',
  'You would rather not supervise anything. A floor you can read assumes someone reading it.',
]

export default function VisionPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Our Vision</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The prompt box is a lonely place to work.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Working with an agent today means typing into a chat window, watching a spinner,
              and reading a wall of text back. The work is happening — you just can&apos;t see
              it. No desk, no board, no sign of who is doing what, or what finished while you
              were away.
            </p>
            <p className="mt-4 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              theboringfloor is our answer: a terminal app where your agents clock in as
              coworkers on a living ASCII floor, and the work stays where you can see it.
            </p>
          </div>
        </section>

        <section className="section-light border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The problem</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A spinner is not the work.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Agent work today is mechanical and transactional. You type, a spinner turns, a
              wall of text comes back. Run two sessions and it gets worse: three panes, two of
              them waiting on you, and no idea which one owns the auth change. You can&apos;t
              see the work move, you can&apos;t ask the next thing while it thinks, and when
              the window closes, everything it learned closes with it.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              That changes how you use the tool. You stop collaborating and start queuing
              requests. The agent never becomes more than an API call — not because it
              can&apos;t, but because there is nowhere for the work to stand still and be
              seen.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>What we believe</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Familiarity builds trust. Trust builds collaboration.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The office is one of the oldest shared-work interfaces we have. Desks you can
              read at a glance. A board with the work on the wall. Someone to ask. Someone
              keeping an eye on the team. Nobody needs onboarding to a desk.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Put an agent at one of those desks and something quiet happens. It walks to the
              whiteboard when the problem is big. It refills the tea machine between batches.
              You stop parsing logs and start reading a room — and you hand over more of the
              work, because you can finally see the work.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              That is the bet: anthropomorphizing the floor is not decoration, it is interface
              design. A system you can see is a system you can supervise.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-7xl px-6 py-20">
            <SectionTag>Meet the office</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A floor of coworkers, not cursors.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The floor runs on a roster, not a loop. Every worker has a name, a desk, and a
              status line that says what it is doing right now.
            </p>
            <ScrollReveal
              stagger={0.06}
              className="mt-12 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 lg:grid-cols-3"
            >
              {coworkers.map((c) => (
                <div key={c.id} className="flex flex-col gap-4 bg-background p-8">
                  <span className="font-mono text-xs text-muted-foreground">{c.id}</span>
                  <h3 className="text-xl font-semibold tracking-tight">{c.name}</h3>
                  <p className="font-mono text-xs uppercase tracking-wider text-accent">{c.role}</p>
                  <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                    {c.body}
                  </p>
                </div>
              ))}
            </ScrollReveal>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Talking to the office</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A queue that never locks you out.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A good boss is interruptible. Message the office mid-task, mid-batch, mid-turn —
              nothing you send falls into a black hole.
            </p>
            <ul className="mt-10 flex max-w-2xl flex-col gap-3 border-l border-border pl-4">
              {etiquette.map((line) => (
                <li key={line} className="text-sm leading-relaxed text-foreground/90 md:text-base">
                  {line}
                </li>
              ))}
            </ul>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Memory</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The office remembers, so you don&apos;t have to re-explain.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A coworker who forgets everything overnight is not much of a coworker. The
              office restores your last session when you open it,{' '}
              <span className="font-mono text-foreground">-s</span> /{' '}
              <span className="font-mono text-foreground">--session</span> pins a thread, and{' '}
              <span className="font-mono text-foreground">/session</span> picks the room you
              want to walk back into.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Under the floor, agentmemory keeps the lessons and the board state across
              shifts. Close the terminal on Friday; walk back in on Tuesday and the office is
              where you left it.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>On the horizon</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Built for coworkers, not one harness.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The office runs on opencode or Claude Code, honestly and visibly — same floor,
              same roster, either harness. The org, the floor, and the memory were built for
              coworkers, whichever harness they arrive in.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              We are at v0.2.x — the &quot;wave&quot; releases. Early, open, and building in
              public. More seats, more office.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Honest ceilings</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              When the office is the wrong tool.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              We would rather lose you here than at hour ten of the wrong setup. Skip the
              whole thing if:
            </p>
            <ul className="mt-10 flex max-w-2xl flex-col gap-3 border-l border-border pl-4">
              {ceilings.map((line) => (
                <li key={line} className="text-sm leading-relaxed text-foreground/90 md:text-base">
                  {line}
                </li>
              ))}
            </ul>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <h2 className="max-w-xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A proper office, for agents.
            </h2>
            <p className="max-w-xl text-pretty leading-relaxed text-muted-foreground">
              Tuesday at a good company: a floor with a hum to it, coworkers with names, work
              you can see moving, a boss you can interrupt, and a team that remembers. That is
              the whole idea — come walk the floor.
            </p>
            <div className="mt-2 flex flex-wrap gap-3">
              <Link
                href="/get-started"
                className="inline-flex items-center bg-foreground px-6 py-3 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
              >
                Get started
              </Link>
              <Link
                href="/blog"
                className="inline-flex items-center border border-border px-6 py-3 font-mono text-xs uppercase tracking-wider transition-colors hover:bg-secondary"
              >
                Read the blog
              </Link>
            </div>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
