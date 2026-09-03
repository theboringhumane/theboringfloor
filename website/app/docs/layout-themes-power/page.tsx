import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Layouts, themes & power | theboringfloor',
  description:
    '/compact, /mode, /wide and /zen for the layout; five themes with live preview; and the /power governor that decides how hard the office renders.',
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

const powerRows = [
  { mode: 'auto (default)', busy: '180ms ticks', idle: '1s', drift: '3s' },
  { mode: 'performance', busy: '150ms flat', idle: '—', drift: '—' },
  { mode: 'saver', busy: '400ms', idle: '2s', drift: '—' },
]

const themes = [
  { name: 'noir', note: 'the default — dim room, accent ember' },
  { name: 'paper', note: 'light, for daylight desks' },
  { name: 'mono', note: 'greys only, no accent' },
  { name: 'dracula', note: 'the classic purple' },
  { name: 'solarized', note: 'the measured palette' },
]

export default function LayoutThemesPowerPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Docs — layout, themes & power</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              Shape the office to the terminal it&apos;s in.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              The same floor runs in a widescreen and in an ssh session half that wide.
              Layouts set the geometry, themes set the paint, and the power governor
              decides how hard the office renders when nothing on it is moving.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Layouts</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Normal is the floor plan. Everything else is a slash command.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Out of the box the office boots into the normal layout: the full sidebar
              with spelled-out tab names and the full-height composer. This is the
              reference everything else narrows, widens or hides.
            </p>
            <Shot
              src="/shots/docs/layout-normal.png"
              alt="theboringfloor normal layout: full sidebar with spelled-out tab names next to the floor"
              caption="normal — the default layout, full sidebar, full composer"
            />
            <p className="mt-14 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/compact on</Code> narrows the sidebar to 30 columns, swaps the tab
              labels for short letters and drops the input to two rows — for narrow
              windows and side-by-side terminals. It holds for this session; if you want
              it to stick, <Code>/mode compact</Code> is the same choice persisted to
              brain.json (<Code>/mode normal</Code> walks back).
            </p>
            <Shot
              src="/shots/docs/layout-compact.png"
              alt="theboringfloor compact layout: 30-column sidebar, short tab letters, two-row input"
              caption="compact — 30-column sidebar, short tab letters, two-row input"
            />
            <p className="mt-14 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/wide &lt;n&gt;</Code> sets the sidebar width directly — 26 to 100
              columns, 0 restores the default of 80 — and persists. <Code>/zen</Code>{' '}
              goes the other way: fullscreen floor, minimal chrome, the sidebar entirely
              out of the way, any key exits. <Code>/focus floor</Code> is its alias.
            </p>
            <Shot
              src="/shots/docs/layout-wide.png"
              alt="theboringfloor wide layout: sidebar widened beyond the 80-column default"
              caption="/wide 100 — the sidebar stretched past its 80-column default"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Themes</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Five paints, previewed before you commit.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The office ships five themes. Pick one at launch with{' '}
              <Code>--theme noir|paper|mono|dracula|solarized</Code>, or in-app with{' '}
              <Code>/theme &lt;name&gt;</Code> — either way the choice persists to{' '}
              <Code>~/.config/theboringfloor/theme</Code>. <Code>/themes</Code> lists
              them without leaving the chat.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The pleasant part is the try-on: arrowing through the slash popover&apos;s{' '}
              <Code>/theme</Code> matches applies each one as a live preview while you
              pass it. <Code>enter</Code> commits what you land on. You never switch a
              theme blind.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              {themes.map((t) => (
                <Chip key={t.name} combo={t.name} action={t.note} />
              ))}
            </div>
            <Shot
              src="/shots/docs/theme-dracula.png"
              alt="theboringfloor dracula theme applied to the floor and sidebar"
              caption="theme dracula — the whole office, repainted live"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Power governor</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The office goes cheap when nothing moves.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/power auto|performance|saver</Code> sets the render cadence and
              writes back to brain.json. Auto is the default and the one most people
              should keep: it reads whether anything is live — streaming, pending
              replies, walkers, open modals, ambient bubbles — and slows the tick the
              moment the floor goes quiet.
            </p>
            <div className="mt-8 max-w-2xl overflow-x-auto border border-border">
              <table className="w-full border-collapse text-left font-mono text-xs">
                <thead>
                  <tr className="border-b border-border">
                    <th className="px-4 py-3 font-normal uppercase tracking-wider text-muted-foreground">
                      mode
                    </th>
                    <th className="px-4 py-3 font-normal uppercase tracking-wider text-muted-foreground">
                      busy
                    </th>
                    <th className="px-4 py-3 font-normal uppercase tracking-wider text-muted-foreground">
                      idle
                    </th>
                    <th className="px-4 py-3 font-normal uppercase tracking-wider text-muted-foreground">
                      drift (1 min quiet)
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {powerRows.map((r) => (
                    <tr key={r.mode} className="border-b border-border last:border-0">
                      <td className="px-4 py-3 text-accent">{r.mode}</td>
                      <td className="px-4 py-3 text-muted-foreground">{r.busy}</td>
                      <td className="px-4 py-3 text-muted-foreground">{r.idle}</td>
                      <td className="px-4 py-3 text-muted-foreground">{r.drift}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              What <Code>saver</Code> actually slows is the cadence the ambience rides:
              walk cycles, tea-machine steam, server LEDs, the typing pulse. It never
              drops events and never slows the boss — the backend keeps its own pace
              regardless. Underneath all three modes the renders are memoized (a frame
              digest on the app; <Code>(size, planGen, tick, renderRev)</Code> on the
              floor), and the agentmemory board poll backs off 2&times; after five quiet
              syncs, capped at 4&times; and reset on change. Battery on a laptop is a
              render problem, and <Code>performance</Code> is 150ms flat for when it
              isn&apos;t.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="/power auto" action="read the room, default" />
              <Chip combo="/power saver" action="400ms busy, 2s idle" />
              <Chip combo="/power performance" action="150ms flat" />
            </div>
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
                <strong className="text-foreground">
                  /compact narrows the sidebar, not the floor.
                </strong>{' '}
                The office floor&apos;s own plan is fixed — resizing the building is not
                a command yet.
              </li>
              <li>
                <strong className="text-foreground">/zen is a view, not a
                screensaver.</strong> Any key exits, so it will not stay parked on your
                second monitor.
              </li>
              <li>
                <strong className="text-foreground">
                  Saver changes cadence, not capability.
                </strong>{' '}
                The floor animates slower; the work behind it does not.
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
                  href: '/docs/keys-and-slash',
                  title: 'Keys & slash commands',
                  body: 'The popover that live-previews /theme as you arrow.',
                },
                {
                  href: '/docs/getting-started',
                  title: 'Getting started',
                  body: 'brain.json and the flags that survive a boot.',
                },
                {
                  href: '/docs/queue-board-memory',
                  title: 'Queue, board & memory',
                  body: 'What keeps running while the floor idles.',
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
