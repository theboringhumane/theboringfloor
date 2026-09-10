import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Layouts, themes & power | theboringfloor',
  description:
    'A cockpit layout, 14 themes with live preview, VS Code theme imports and customization, and the /power rendering governor.',
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] hairline bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

function Chip({ combo, action }: { combo: string; action: string }) {
  return (
    <span className="inline-flex items-center gap-2 hairline px-3 py-1.5 font-mono text-xs">
      <span className="text-accent">{combo}</span>
      <span className="text-muted-foreground">{action}</span>
    </span>
  )
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="mt-10 m-0">
      <div className="doc-brackets hairline bg-paper-2 p-2 md:p-3">
        <img
          src={src}
          alt={alt}
          width={5086}
          height={2896}
          loading="lazy"
          className="shot-img block h-auto w-full"
        />
      </div>
      <figcaption className="mono-label mt-3 text-ink-faint">{caption}</figcaption>
    </figure>
  )
}

const powerRows = [
  { mode: 'auto (default)', busy: '180ms ticks', idle: '1s', drift: '3s' },
  { mode: 'performance', busy: '150ms flat', idle: '—', drift: '—' },
  { mode: 'saver', busy: '400ms', idle: '2s', drift: '—' },
]

const themes = [
  { name: 'cockpit', note: 'dark default, cyan instruments' },
  { name: 'noir', note: 'dim room, accent ember' },
  { name: 'paper', note: 'light, for daylight desks' },
  { name: 'mono', note: 'greys only, no accent' },
  { name: 'dracula', note: 'the classic purple' },
  { name: 'solarized', note: 'the measured palette' },
  { name: 'tokyo-night', note: 'blue neon' },
  { name: 'catppuccin-mocha', note: 'dark pastel' },
  { name: 'catppuccin-latte', note: 'light pastel' },
  { name: 'nord', note: 'arctic blue' },
  { name: 'gruvbox', note: 'warm retro' },
  { name: 'one-dark', note: 'balanced charcoal' },
  { name: 'rose-pine', note: 'muted rose and purple' },
  { name: 'github-light', note: 'crisp daylight' },
]

export default function LayoutThemesPowerPage() {
  return (
    <>
      <SiteHeader />
      <main className="relative paper-ground">
        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>12 · Docs — layout, themes & power</SectionTag>
            <h1 className="mt-8 max-w-3xl display-lg text-balance text-ink">
              Shape the office to the terminal it&apos;s in.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              The same floor runs in a widescreen and in an ssh session half that wide.
              Layouts set the geometry, themes set the paint, and the power governor
              decides how hard the office renders when nothing on it is moving.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Layouts</SectionTag>
            <h2 className="mt-6 max-w-2xl display-md text-balance text-ink md:text-4xl">
              Normal is the floor plan. Everything else is a slash command.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The desktop layout has three panes: project floors on the far left, the office
              in the middle, and transcript and eight tools tabs on the right. Ctrl+E focuses
              floors; Ctrl+W expands tools while retaining the navigator. Below 100 columns,
              floors use a drawer. See <Link href="/docs/workspaces" className="underline">workspaces</Link> for the complete workflow.
            </p>
            <Shot
              src="/shots/workspaces/transcript.webp"
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

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Themes</SectionTag>
            <h2 className="mt-6 max-w-2xl display-md text-balance text-ink md:text-4xl">
              Fourteen palettes. Bring your own, too.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Every palette uses the same cockpit: tactical floor, live task meter,
              agent network, dispatches and numbered tool consoles. Instruments
              collapse when the terminal is small. Dark terminals default to Cockpit;
              light terminals use Paper. Pick a palette at launch with{' '}
              <Code>--theme tokyo-night</Code>, or save a choice in-app with{' '}
              <Code>/theme &lt;name&gt;</Code>. <Code>/themes</Code> lists them all.
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
              src="/shots/docs/cockpit-themes.png"
              alt="The cockpit layout in all fourteen built-in palettes"
              caption="one cockpit, fourteen palettes — simulated demo mission"
            />
            <h3 className="mt-12 text-2xl font-semibold">Import a VS Code theme</h3>
            <p className="mt-4 max-w-2xl leading-relaxed text-muted-foreground">
              Use <Code>/theme import &quot;/path/to/My Theme.json&quot;</Code> or launch with{' '}
              <Code>--import-theme ./my-theme.jsonc</Code>. The imported palette appears
              in the picker as <Code>custom-my-theme</Code> and your choice is saved.
              Installed VS Code extensions usually keep their files under{' '}
              <Code>~/.vscode/extensions/&lt;extension&gt;/themes/</Code>; the extension&apos;s{' '}
              <Code>package.json</Code> lists them in <Code>contributes.themes</Code>.
            </p>
            <h3 className="mt-10 text-2xl font-semibold">Edit any palette</h3>
            <p className="mt-4 max-w-2xl leading-relaxed text-muted-foreground">
              Run <Code>/theme export ~/my-theme.json</Code>, edit its{' '}
              <Code>name</Code>, <Code>colors</Code> or <Code>tokenColors</Code>, then{' '}
              <Code>/theme import ~/my-theme.json</Code>. Reimport the same name to
              update it live. Exports require a new destination file. Saved imports
              live in <Code>~/.config/theboringfloor/themes/</Code>, or under{' '}
              <Code>$XDG_CONFIG_HOME</Code> when set. Edit those files directly and use{' '}
              <Code>/theme reload</Code> to refresh the library.
            </p>
            <p className="mt-4 max-w-2xl leading-relaxed text-muted-foreground">
              JSON/JSONC comments, trailing commas, relative local includes, transparent
              hex colors and common TextMate scopes for diff syntax are supported.
              Editor, status bar, border, button, terminal ANSI and diff colors map to
              the cockpit; missing slots use light or dark defaults. VS Code-specific
              components, semantic-token rules, complex language selectors and{' '}
              <Code>.tmTheme</Code> references do not transfer. Import the JSON theme
              file itself, rather than an extension package.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Power governor</SectionTag>
            <h2 className="mt-6 max-w-2xl display-md text-balance text-ink md:text-4xl">
              The office goes cheap when nothing moves.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/power auto|performance|saver</Code> sets the render cadence and
              writes back to brain.json. Auto is the default and the one most people
              should keep: it reads whether anything is live — streaming, pending
              replies, walkers, open modals, ambient bubbles — and slows the tick the
              moment the floor goes quiet.
            </p>
            <div className="mt-8 max-w-2xl overflow-x-auto hairline">
              <table className="w-full border-collapse text-left font-mono text-xs">
                <thead>
                  <tr className="border-b border-rule">
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
                    <tr key={r.mode} className="border-b border-rule last:border-0">
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

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Ceilings</SectionTag>
            <h2 className="mt-6 max-w-2xl display-md text-balance text-ink md:text-4xl">
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

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-8 grid grid-cols-1 gap-px overflow-hidden hairline bg-border md:grid-cols-3">
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
