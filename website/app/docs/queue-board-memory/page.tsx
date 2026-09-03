import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Queue, board & memory | theboringfloor',
  description:
    'Free-send while the boss works, the [BATCH DISPATCH] flush, board sync on completion, the office memory ledger, and what /stop unwinds.',
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] border border-border bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

function Chip({ combo, action }: { combo: string; action: string }) {
  return (
    <span className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs">
      <span className="text-accent">{combo}</span>
      <span className="text-muted-foreground">{action}</span>
    </span>
  )
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="mt-10 overflow-hidden border border-border bg-(--shot-frame)">
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
    </figure>
  )
}

export default function QueueBoardMemoryPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Docs — queue, board & memory</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The prompt never locks.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Send while the boss is mid-turn and the line lands in a backlog the office
              manages — not a tunnel, not a lost keystroke. Flushes go out as one batch,
              completions sweep the board behind them, and everything the office finishes
              lands in a ledger the next boss reads.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Free-send</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The backlog is a queue the office manages.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Hit <Code>enter</Code> while the boss works and the prompt free-sends into the
              backlog — the status line reads <Code>busy · N queued</Code> and you keep
              typing. Queued items are not invisible: each one gets a board row, so the
              office and its telemetry agree on what is waiting.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When the backlog drains it goes out as a single{' '}
              <Code>[BATCH DISPATCH]</Code> — one composed message the boss decomposes
              into parallel sub-agents, instead of N separate turns replayed in order.
              <Code>/route</Code> forces that flush early, and <Code>/queue</Code> shows
              the backlog (<Code>/queue clear</Code> drops it). A dead boss respawns a
              fresh session and resends the batch — the queue is not hostage to one
              process.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="enter" action="free-send into the backlog" />
              <Chip combo="/queue" action="show the backlog" />
              <Chip combo="/queue clear" action="drop the backlog" />
              <Chip combo="/route" action="force-dispatch now" />
            </div>
            <Shot
              src="/shots/docs/batch-dispatch.png"
              alt="theboringfloor chat tab: three queued prompts flushing out as one [BATCH DISPATCH] message"
              caption="chat — a three-item backlog flushing as one [BATCH DISPATCH]"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Board sync</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Every completion sweeps the board behind it.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A worker return — or a flushed queue item — flips stranded DOING rows to
              DONE so the office board stays honest without you policing it. Only
              office-owned DOING rows are in play: agentmemory-mirrored rows own their
              own lifecycle and are never touched.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The matching is conservative by design. The completion&apos;s own exact row
              is excluded — the caller already closed it, and the sweep never
              double-flips it. Then the same worker&apos;s oldest in-progress row flips,
              one per completion. Then a normalized title-prefix match (&ge;8 characters,
              case- and whitespace-insensitive) catches twins. A row flips only when
              exactly one rule claims it — ambiguity across workers flips nothing, and
              nothing about it is printed or fudged.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When anything did move you get exactly one dim note:{' '}
              <Code>[office] board sync: flipped N rows to done</Code>. No sweep, no
              banner — and that note stays on the status bar and the activity log, never
              in the transcript.
            </p>
            <Shot
              src="/shots/docs/board-sync.png"
              alt="theboringfloor board tab: DOING rows flipping to DONE behind a worker return, with the dim board sync note"
              caption="board — two stranded DOING rows flipping to DONE"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Memory</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The ledger is the office&apos;s completed-work memory.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Every completed dispatch lands in{' '}
              <Code>&lt;dir&gt;/.opencode/office-ledger.md</Code> — newest first, capped
              at 50 — as one{' '}
              <Code>### date · title — worker (role) · `verdict`</Code> block carrying the
              summary line, the touched files, the verify digest, a proof one-liner and a{' '}
              <Code>ledgerId</Code>. The charter wires that file into the boss&apos;s
              instructions, so the next session&apos;s boss reads what this project
              already finished before re-dispatching it. No more &quot;I don&apos;t have
              any memory of a prior task.&quot;
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The same record also rides to agentmemory as an{' '}
              <Code>office_dispatch_done</Code> observation when that server is up —
              file-only when it is not. The boot splash&apos;s{' '}
              <Code>memory: ledger armed</Code> versus{' '}
              <Code>memory: file-only (agentmemory :3111 refused)</Code> line, and the
              status line&apos;s <Code>memory:</Code> note, tell you which lane got it.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              In-app, <Code>/memory</Code> reads the ledger newest-first: the header
              counts the records and names the memory state (<Code>agentmemory OK</Code>{' '}
              or <Code>file-only</Code>), a missing ledger shows an honest empty state,
              and <Code>/memory fix</Code> narrows rows case-folded over title and files.
              Each completed dispatch also stamps a{' '}
              <Code>[memory] recorded: &lt;title&gt; &rarr; ledger</Code> line into the
              activity tab.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="/memory" action="completed dispatches, newest first" />
              <Chip combo="/memory fix" action="filter over title/files" />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Unwinding</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              /stop aborts. The watchdog names a wedge.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/stop</Code> aborts current work — boss and workers — on the spot,
              and free-send keeps taking prompts the whole time. If a busy turn goes
              silent for two minutes, the wedge watchdog notes it once: the office calls
              that a wedge, and it reads on the status bar and in the activity log —
              never in the transcript.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A wedge is a suspicion, not a lock. The office itself keeps answering:
              keys, panels, scrolling and <Code>/stop</Code> all still work while the
              note is up, so a silent turn never takes the floor down with it.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The inverse is an idle floor. After workers finish and the last real chat was
              yours — not the boss, not the office concierge — two quiet minutes asks the boss
              to write a recap in chat. The fallback office note only appears if that send
              misses the wire. A late boss or concierge wrap cancels the recap.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="/stop" action="abort boss + workers" />
              <Chip combo="2min" action="of silence = wedged, noted once" />
              <Chip combo="2min idle" action="boss writes a recap" />
            </div>
            <Shot
              src="/shots/docs/stop-unwind.png"
              alt="theboringfloor: /stop aborting in-flight work with the wedge watchdog note on the status bar"
              caption="chat — /stop unwinds the turn; the wedge note rides the status bar"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Ceilings</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              What this doesn&apos;t do yet.
            </h2>
            <ul className="mt-6 flex max-w-2xl flex-col gap-3 leading-relaxed text-muted-foreground">
              <li>
                <strong className="text-foreground">Stuck rows stay stuck.</strong> Board
                sync never flips a row it can&apos;t prove — a cross-worker doubt means
                no flip and no drama, so a stray DOING row is yours to close by hand.
              </li>
              <li>
                <strong className="text-foreground">The ledger is not recall.</strong>{' '}
                It records completed dispatches for the next boss, newest 50. The
                boss&apos;s scrollback is transcript history; agentmemory&apos;s semantic
                search is agentmemory&apos;s. <Code>/memory</Code> only reads the ledger.
              </li>
              <li>
                <strong className="text-foreground">/stop is not a rollback.</strong> It
                aborts the turn — whatever the turn already changed on disk stays
                changed.
              </li>
              <li>
                <strong className="text-foreground">/queue clear has no undo.</strong>{' '}
                Dropped backlog items are gone, not parked.
              </li>
            </ul>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-8 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-3">
              {[
                {
                  href: '/docs/chat-and-threads',
                  title: 'Chat & threads',
                  body: 'What a dispatched batch looks like while it runs.',
                },
                {
                  href: '/docs/plan-mode',
                  title: 'Plan mode',
                  body: 'Draft the plan first; the batch follows it.',
                },
                {
                  href: '/docs/keys-and-slash',
                  title: 'Keys & slash commands',
                  body: '/queue, /route, /stop and /memory in the full table.',
                },
              ].map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  className="group flex flex-col gap-2 bg-background p-8 transition-colors hover:bg-card"
                >
                  <span className="font-mono text-xs uppercase tracking-wider text-accent">
                    {l.title}
                  </span>
                  <span className="text-sm leading-relaxed text-muted-foreground group-hover:text-foreground">
                    {l.body}
                  </span>
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
