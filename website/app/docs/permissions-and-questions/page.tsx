import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Permissions and questions',
  description:
    'The office asks before it acts: a stacked 1 of N permission queue with y/a/n/esc, a question wizard with four answer kinds, and a concierge that picks up when the boss is mid-turn.',
  alternates: {
    canonical: '/docs/permissions-and-questions',
  },
  openGraph: {
    title: 'Permissions and questions · theboringoffice',
    description:
      'The office asks before it acts: a stacked 1 of N permission queue with y/a/n/esc, a question wizard with four answer kinds, and a concierge that picks up when the boss is mid-turn.',
    url: `${SITE_URL}/docs/permissions-and-questions`,
    type: 'website',
  },
}

const permKeys = [
  { combo: 'y', action: 'allow once' },
  { combo: 'a', action: 'allow always' },
  { combo: 'n', action: 'reject' },
  { combo: 'esc', action: 'park the ask' },
  { combo: '/perm', action: 're-open an esc’d ask' },
]

const questionKeys = [
  { combo: 'esc', action: 'defer the question' },
  { combo: '/question', action: 're-open a deferred one' },
]

const conciergeKeys = [
  { combo: 'enter', action: 'free-send — never blocks' },
  { combo: '/queue', action: 'show the backlog' },
  { combo: '/route', action: 'force-dispatch now' },
  { combo: '/stop', action: 'abort boss + workers' },
]

const bypassKeys = [{ combo: '/bypass', action: 'toggle bypass-permissions mode' }]

const bypassProtections = [
  {
    label: 'session only',
    body: 'Every boot starts with bypass OFF. The toggle lives in memory and the badge dies with the office session — brain.json never gains a key.',
  },
  {
    label: 'confirm on enable',
    body: 'Turning it on is a two-step: the office’s question modal makes you confirm that agents will run tools and browser actions without asking. Turning it off is instant.',
  },
  {
    label: 'loud indicator',
    body: 'While it’s on, every tab’s topbar carries a ⚠ BYPASS segment. The mode never hides, and neither does the dim log row each auto-approved ask leaves in the transcript.',
  },
  {
    label: 'what lands on disk',
    body: 'Nothing. Claude receives its command-line flag; the office-owned opencode child receives an ephemeral OPENCODE_CONFIG_CONTENT allow-all override. brain.json, .opencode/opencode.json, and the parent process environment remain byte-identical, and every boot starts OFF.',
  },
]

const questionKinds = [
  {
    kind: 'text',
    body: 'A free-answer field. The boss gets your sentence back verbatim — names, flags, judgment calls. A multi-line paste lands in the field verbatim too, newlines and all.',
  },
  {
    kind: 'radio',
    body: 'Pick one. Mutually exclusive options, one choice.',
  },
  {
    kind: 'checkbox',
    body: 'Pick several. Zero or more options stick.',
  },
  {
    kind: 'confirm',
    body: 'Yes or no. The smallest possible fork.',
  },
]

const limits = [
  {
    label: 'parked is not answered',
    body: 'esc gets an ask out of your face, not out of the world. Nothing answers it for you later — /perm or /question re-opens it when you are ready.',
  },
  {
    label: 'the concierge lane',
    body: 'The concierge answers the message; it does not drain the backlog. Queued sends still flush on the office’s schedule (or /route) — and if the concierge itself is unavailable, a notice says so and your prompt rides the queue.',
  },
  {
    label: 'a wedged turn',
    body: 'Two silent minutes on a busy turn earns one wedge note — status bar and activity log, never the transcript. It is a note, not a rescue: /stop is the lever that actually aborts boss and workers.',
  },
  {
    label: 'the click path',
    body: 'The queue’s menu is clickable, but the office never answers on your behalf — allow-once, always, and reject all wait for an explicit key or click. There is no silent default.',
  },
]

const keepReading = [
  {
    href: '/docs/queue-board-memory',
    kicker: 'next',
    title: 'Queue, board, and memory',
    blurb: 'Where parked sends and flushed batches land — the backlog the concierge defers to.',
  },
  {
    href: '/docs/chat-and-threads',
    kicker: 'related',
    title: 'Chat and work threads',
    blurb: 'The transcript these floats sit on top of — streaming replies, threads, and diffs.',
  },
  {
    href: '/docs/keys-and-slash',
    kicker: 'reference',
    title: 'Keys and slash commands',
    blurb: 'y / a / n / esc, /perm, /question, /queue, /route, /stop — the whole table.',
  },
  {
    href: '/docs/backends',
    kicker: 'related',
    title: 'Backends',
    blurb: 'The same queue discipline on opencode or Claude Code — the answers map onto each transport’s own permission replies.',
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

export default function PermissionsAndQuestionsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Permissions &amp; questions</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The office asks before it acts.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Agents don&apos;t get silent yeses here. Permission asks stack into one queue you
              answer on your terms, boss questions open as small wizards instead of modal essays,
              and a concierge takes the message when the boss is mid-turn.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Permission queue</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Asks stack as 1 of N. None of them steals the screen.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When the boss or a worker wants to run a command, write a path, or touch a host, the
              ask joins a queue instead of hijacking the terminal. The front of the queue shows
              where you stand —{' '}
              <span className="font-mono text-foreground">1 of N</span> — with a clickable menu:
              allow once, allow always, reject.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The keys say the same thing:{' '}
              <span className="font-mono text-foreground">y</span> allows this ask once,{' '}
              <span className="font-mono text-foreground">a</span> allows the pattern always,{' '}
              <span className="font-mono text-foreground">n</span> rejects it, and{' '}
              <span className="font-mono text-foreground">esc</span> parks the float without
              answering — the rest of the office keeps working either way. Parked is not lost:{' '}
              <span className="font-mono text-foreground">/perm</span> re-opens an esc&apos;d
              prompt.
            </p>
            <p className="mt-4 max-w-2xl border border-border bg-card p-5 text-pretty text-sm leading-relaxed text-muted-foreground">
              <span className="font-medium text-foreground">
                What allow-always does not cover:
              </span>{' '}
              it answers one tool pattern, and that is all. It never pre-answers the boss&apos;s
              questions — those are decisions, not permissions — and asks for anything you have
              not already blessed still stack into the queue as 1 of N. Press{' '}
              <span className="font-mono text-foreground">a</span> only on a pattern you can say
              out loud.
            </p>
            <KeyChips keys={permKeys} />
            <Shot
              src="/shots/docs/permission-modal.png"
              alt="theboringoffice permission queue: the front ask with a 1 of N counter and the allow once / always / reject menu open"
              caption="permission queue — 1 of N; y once · a always · n reject · esc park"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Bypass mode</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              /bypass is the deliberate escape hatch.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Some rooms don&apos;t need a bouncer: a throwaway repo, a sandboxed
              environment, a demo you&apos;ll reset in an hour.{' '}
              <span className="font-mono text-foreground">/bypass</span> toggles a
              session-scoped bypass-permissions mode for exactly those rooms. Enabling
              asks for an explicit confirm first — agents will run tools and browser
              actions without asking, this office session only — so the loud path is
              never the accidental one. Disabling is instant.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              While it&apos;s on, two ask surfaces go quiet. Backend tool asks stop
              entirely — claude spawns with{' '}
              <span className="font-mono text-foreground">
                --dangerously-skip-permissions
              </span>
               , and the office-owned opencode child gets an ephemeral{' '}
              <span className="font-mono text-foreground">OPENCODE_CONFIG_CONTENT</span>{' '}
              permission allow-all — and the
              office&apos;s own browser-action prompt (click, fill, eval) is skipped
              too. Anything that still slips an ask through is auto-approved, with a
              dim log row in the transcript so the record stays honest. Toggling
              respawns the backend so the mode actually reaches the agent; on claude,
              your session context resumes across the respawn.
            </p>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {bypassProtections.map((p) => (
                <div key={p.label} className="bg-background p-6">
                  <p className="font-mono text-xs uppercase tracking-wider text-accent">
                    {p.label}
                  </p>
                  <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{p.body}</p>
                </div>
              ))}
            </div>
            <KeyChips keys={bypassKeys} />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Question wizard</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Boss questions open as small wizards, not modal essays.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When the boss needs a decision mid-dispatch, the ask opens as popover pages,
              classified automatically into four kinds:
            </p>
            <div className="mt-8 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {questionKinds.map((q) => (
                <div key={q.kind} className="bg-background p-5">
                  <p className="font-mono text-xs uppercase tracking-wider text-accent">{q.kind}</p>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{q.body}</p>
                </div>
              ))}
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <span className="font-mono text-foreground">esc</span> defers the whole wizard into a
              lane of its own — the chat stays usable underneath — and{' '}
              <span className="font-mono text-foreground">/question</span> re-opens the deferred
              page when you have an answer. There is no allow-always on a question: each one is a
              decision the office refuses to make for you.
            </p>
            <KeyChips keys={questionKeys} />
            <Shot
              src="/shots/docs/question-modal.png"
              alt="theboringoffice question wizard: a boss question open as a popover page with classified answer options"
              caption="question wizard — one page per ask, classified automatically"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Concierge</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Send mid-turn and the concierge picks up.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <span className="font-mono text-foreground">enter</span> while the boss works never
              blocks: the prompt free-sends into the backlog and the status line reads{' '}
              <span className="font-mono text-foreground">busy · N queued</span>. The queue is a
              backlog the office manages, not a tunnel — flush goes out as one{' '}
              <span className="font-mono text-foreground">[BATCH DISPATCH]</span> the boss
              decomposes into parallel sub-agents, and{' '}
              <span className="font-mono text-foreground">/route</span> forces it early. The board
              side of that story lives in{' '}
              <Link
                href="/docs/queue-board-memory"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                queue, board, and memory
              </Link>
              .
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              If the boss is mid-task, the office concierge answers instantly as a real
              conversation turn, noted in chat as{' '}
              <span className="font-mono text-foreground">
                office routed: boss busy → concierge
              </span>
              . If the concierge is unavailable, a notice says so and the prompt rides the backlog
              instead — no silent drop, no fake reply.
            </p>
            <KeyChips keys={conciergeKeys} />
            <Shot
              src="/shots/docs/concierge.png"
              alt="theboringoffice concierge: a message sent while the boss is mid-turn answered by the office concierge, noted in chat as boss busy, routed to concierge"
              caption="concierge — mid-turn send answered, routing noted in chat"
            />
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
