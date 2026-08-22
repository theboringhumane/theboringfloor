import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Vision | theboringoffice',
  description: 'Why we are building a terminal office where real coding agents can work visibly and remember their shifts.',
}

const principles = [
  {
    id: '01',
    title: 'Real work should be visible',
    body:
      'The boss is a real opencode session and employees are real task sub-agent sessions. Their events drive the floor, work threads, activity feed, and status—not a decorative simulation.',
  },
  {
    id: '02',
    title: 'Humans should decide, not babysit',
    body:
      'Permission requests stack in a queue with explicit allow-once, always, and reject choices. Questions open as focused prompts, so attention goes to the decisions that need it.',
  },
  {
    id: '03',
    title: 'A shift should survive a restart',
    body:
      'agentmemory preserves sessions, decisions, lessons, board actions, and mail signals. The office restores the last chat by default, so context does not disappear when the terminal closes.',
  },
  {
    id: '04',
    title: 'The terminal is the workplace',
    body:
      'theboringoffice is a native Go and Bubble Tea app with an embedded PTY, keyboard-first controls, and a real-time view of what the backend is doing. It belongs where coding work already happens.',
  },
]

const shiftStages = [
  { step: '01', label: 'Start live mode or attach to an existing opencode server.' },
  { step: '02', label: 'Oikonomos manages the boss session while opencode sub-agents take on dispatched work.' },
  { step: '03', label: 'SSE events update the floor, work threads, activity, and agent roster as work progresses.' },
  { step: '04', label: 'agentmemory keeps the board, mail, sessions, and lessons available for the next shift.' },
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
              A real office for coding agents, inside your terminal.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Coding agents need more than a prompt box. They need a place where work is visible,
              questions and permissions reach the right person, and the context from yesterday&apos;s
              shift is still there tomorrow. theboringoffice makes that place terminal-native.
            </p>
          </div>
        </section>

        <section className="section-light border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The hard part is everything around the model.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A useful coding agent needs an operating environment: a real session, a way to
              dispatch work, a clear place for permissions and questions, and a record of what
              happened. A terminal log alone is not enough to run that reliably.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              theboringoffice turns those moving parts into one observable workspace. The Go TUI
              renders the office from live opencode SSE events, while agentmemory carries the
              actions, signals, sessions, and lessons that make work continuous.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-7xl px-6 py-20">
            <SectionTag>What we believe</SectionTag>
            <div className="mt-12 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2">
              {principles.map((p) => (
                <div key={p.id} className="flex flex-col gap-4 bg-background p-8">
                  <span className="font-mono text-xs text-muted-foreground">{p.id}</span>
                  <h3 className="text-xl font-semibold tracking-tight">{p.title}</h3>
                  <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                    {p.body}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>How the office works</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              One stateful shift, from prompt to durable context
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The office is built around the work already happening in opencode. A single state
              model feeds the floor and sidebar while the backend owns network calls and event
              normalization—making the work legible without pretending it is something else.
            </p>

            <div className="mt-14 flex flex-col">
              {shiftStages.map((stage, i) => (
                <div
                  key={stage.step}
                  className={
                    i === shiftStages.length - 1
                      ? 'flex gap-8 py-6'
                      : 'flex gap-8 border-b border-border py-6'
                  }
                >
                  <span className="w-16 shrink-0 font-mono text-sm text-accent">
                    {stage.step}
                  </span>
                  <p className="text-sm leading-relaxed text-foreground/90 md:text-base">
                    {stage.label}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <h2 className="max-w-xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Start a shift in the office.
            </h2>
            <p className="max-w-xl text-pretty leading-relaxed text-muted-foreground">
              Tour the terminal workspace in demo mode, then connect a real opencode server when
              you&apos;re ready. Your chats, board actions, mail, and lessons can keep their context
              between shifts.
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
