import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

export const metadata: Metadata = {
  title: 'Vision | theboringoffice',
  description:
    'Why we are building a living office for agents — working with agents should feel beautiful and more human.',
}

const coworkers = [
  {
    id: '01',
    name: 'the manager',
    role: 'The boss',
    body: 'The boss of the floor. Dispatches work to the team, keeps the shift coherent, and is the one you talk to — interruptible, batchable, always reachable.',
  },
  {
    id: '02',
    name: 'theboringcto',
    role: 'The CTO',
    body: 'Seated from boot. Reviews every drained batch before it lands and owns the architecture briefs — the adult in the room for the decisions that matter.',
  },
  {
    id: '03',
    name: 'hr',
    role: 'People & health',
    body: 'Overlooks team health and people issues on the floor — because even a team of agents does its best work when someone is watching how the work feels.',
  },
  {
    id: '04',
    name: 'tekton',
    role: 'The developers',
    body: 'Claims tickets off the board, opens work threads you can expand right in chat, drops mail to coworkers, and walks to the tea machine between batches.',
  },
  {
    id: '05',
    name: 'scouts, reviewers, runners',
    role: 'The supporting cast',
    body: 'Scouts recon the codebase before work starts, reviewers weigh in on every return, and runners carry the work through the floor until it is done.',
  },
  {
    id: '06',
    name: 'the floor itself',
    role: 'Ambient life',
    body: 'Coffee steam, blinking server LEDs, social chatter on the social clock, light and dark themes, and sound. An office that feels inhabited, not rendered.',
  },
]

const etiquette = [
  'Message the boss while it works — your words are intelligently batched into one composed send, not lost in the scrollback.',
  'The concierge answers instantly while the boss is busy, so you are never talking to a wall.',
  '/stop aborts on the spot, and free-send keeps you typing while the office is mid-turn.',
  'Permission asks stack as 1 of N instead of stealing your screen — approve, always-allow, or reject, and never get locked out.',
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
              Working with agents should feel beautiful and more human.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              theboringoffice is a terminal app where your agents clock in as coworkers on a
              living ASCII floor — they sit at desks, walk to the tea machine, gather at the
              whiteboard, and answer you in chat, on the board, and in the mail.
            </p>
            <p className="mt-4 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Not another coding agent. A place where agents work — and where you work with them.
            </p>
          </div>
        </section>

        <section className="section-light border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The problem</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The prompt box is a lonely place to work.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Working with an agent today is mechanical and transactional: you type into a chat
              window, watch a spinner, and get a wall of text back. You can&apos;t see the work
              move, you can&apos;t ask the next thing while it thinks, and when the window closes,
              everything it learned closes with it.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Prompts in a void. That loneliness isn&apos;t a cosmetic problem — it changes how you
              relate to the tool. You stop collaborating and start queuing requests, and the agent
              never gets the chance to be anything more than an API call.
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
              The office is humanity&apos;s oldest shared-work interface. Long before dashboards
              and terminals, complex work got coordinated by putting people in a room: desks you
              can read at a glance, a board with the work on the wall, someone to ask, someone
              keeping an eye on the team. Everyone already knows how an office works — nobody
              needs onboarding to a desk.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When your agent sits at that desk, walks to the whiteboard when the problem is big,
              and refills the tea machine between batches, something quiet happens: you stop
              parsing logs and start reading a room. Familiarity turns into trust; trust turns
              into better collaboration. You hand the system more of the work, because you can
              finally see the work.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              That is the bet behind theboringoffice: anthropomorphizing the floor is not
              decoration — it is interface design. A system that feels alive is a system you
              instinctively work with. People-shaped interfaces, for machine-shaped workers.
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
              This is a proper org, not a loop. theboringoffice hires to a roster, fires by task,
              and gives every worker a name, a desk, and a status line that says what it is doing
              right now.
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
              No queues. No blocking. Ask anything.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A good boss is interruptible. You can message the office at any moment — mid-task,
              mid-batch, mid-anything — and nothing you send falls into a black hole.
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
              A coworker who forgets everything overnight isn&apos;t much of a coworker.
              theboringoffice restores your last session when you open it, and{' '}
              <span className="font-mono text-foreground">-s</span> /{' '}
              <span className="font-mono text-foreground">--session</span> pins any thread so you
              can resume exactly where you left off.{' '}
              <span className="font-mono text-foreground">/session</span> lets you pick the room
              you want to walk back into.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Under the floor, agentmemory keeps the lessons, notes, and board state across shifts
              — the board syncs, the lessons compound, and yesterday&apos;s context is still there
              tomorrow. Close the terminal on Friday; walk back in on Tuesday and the office is
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
              Today, the office runs on opencode — honestly and visibly. Claude Code support is
              coming soon, and the floor was designed for exactly that: the org, the panels, and
              the memory are built for coworkers, whichever harness they arrive in.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              We are at v0.2.0 — the &quot;wave&quot; releases — early, open, and building in
              public. More agents, more seats, more office. Stay tuned.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <h2 className="max-w-xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A proper office, for agents.
            </h2>
            <p className="max-w-xl text-pretty leading-relaxed text-muted-foreground">
              It should feel like Tuesday at a good company: a floor with a hum to it, coworkers
              with names, work you can see moving, a boss you can interrupt, and a team that
              remembers. Not just a coding agent — an alive system, a proper office.
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
