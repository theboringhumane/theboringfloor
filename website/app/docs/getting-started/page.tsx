import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Getting Started | theboringoffice',
  description:
    'Install theboringoffice with one curl, tour it in demo mode or boot a live office, and resume any past session.',
  alternates: {
    canonical: '/docs/getting-started',
  },
  openGraph: {
    title: 'Getting Started · theboringoffice',
    description:
      'Install theboringoffice with one curl, tour it in demo mode or boot a live office, and resume any past session.',
    url: `${SITE_URL}/docs/getting-started`,
    type: 'website',
  },
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="overflow-hidden border border-border bg-black">
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
        className="block h-auto w-full"
      />
    </figure>
  )
}

function CmdBlock({ lines }: { lines: { t: string; dim?: boolean }[] }) {
  return (
    <div className="border border-border bg-card p-6 font-mono text-xs leading-relaxed">
      {lines.map((l, i) => (
        <p key={i} className={l.dim ? 'text-muted-foreground' : undefined}>
          {l.t}
        </p>
      ))}
    </div>
  )
}

function Key({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center border border-border px-2 py-0.5 font-mono text-xs text-accent">
      {children}
    </span>
  )
}

export default function GettingStartedPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Getting started</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              From curl to a working office in five minutes.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              One install line, one demo tour, then a live office with a real boss and working
              sub-agents. This page is the whole on-ramp — no prerequisites beyond a terminal.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              1 · Install
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              One line puts the binary and its memory service on your machine.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The installer drops the{' '}
              <code className="font-mono text-xs text-foreground">theboringoffice</code> binary on
              your PATH and wires{' '}
              <code className="font-mono text-xs text-foreground">agentmemory</code> as a
              reboot-safe service. That memory layer is what makes the task board and mail room
              real — it is not optional wallpaper.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '# binary + agentmemory service, one line', dim: true },
                  { t: 'curl -fsSL \\' },
                  { t: '  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh' },
                ]}
              />
            </div>
            <p className="mt-8 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Want Claude Code as the brain instead of the default opencode? Pass{' '}
              <code className="font-mono text-xs text-foreground">--backend</code> at install time
              — it seeds <code className="font-mono text-xs text-foreground">brain.json</code>
              &apos;s <code className="font-mono text-xs text-foreground">backend.name</code> so the
              choice survives every boot. claudecode needs the{' '}
              <code className="font-mono text-xs text-foreground">claude</code> CLI on PATH; a
              missing CLI is a warning, not a failed install.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '# same one-liner, transport pinned at install time', dim: true },
                  { t: 'curl -fsSL \\' },
                  { t: '  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh -s -- --backend claudecode' },
                  { t: '', dim: true },
                  { t: '# or pin one boot from the shell (flag > env > brain.json > opencode)', dim: true },
                  { t: 'THEBORINGOFFICE_BACKEND=claudecode theboringoffice' },
                ]}
              />
            </div>
            <p className="mt-8 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The full transport story — including the mid-flight{' '}
              <code className="font-mono text-xs text-foreground">/backend</code> swap — lives on
              the{' '}
              <Link href="/docs/backends" className="text-foreground underline underline-offset-4 hover:text-accent">
                backends page
              </Link>
              .
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              2 · First run
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              Tour first, go live second.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Three ways to boot. <strong className="font-medium text-foreground">Demo mode</strong>{' '}
              simulates the whole floor — walkers, mail, the works — and labels itself DEMO so you
              never mistake the tour for real work. A bare boot is the{' '}
              <strong className="font-medium text-foreground">live office</strong>: it spawns{' '}
              <code className="font-mono text-xs text-foreground">opencode serve</code> and the real
              boss. And <code className="font-mono text-xs text-foreground">--server</code> attaches
              to an opencode server you already have running, instead of spawning a new one.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '# tour: simulated events, labeled DEMO', dim: true },
                  { t: 'theboringoffice --demo' },
                  { t: '', dim: true },
                  { t: '# live: spawns `opencode serve`, real boss (oikonomos)', dim: true },
                  { t: 'theboringoffice' },
                  { t: '', dim: true },
                  { t: '# attach to an existing server instead of spawning one', dim: true },
                  { t: 'theboringoffice --server http://127.0.0.1:4096' },
                ]}
              />
            </div>
            <div className="mt-12">
              <Shot
                src="/shots/docs/first-run-chat.png"
                alt="First live run of theboringoffice: chat tab in focus, boss reply streaming"
                caption="first live boot — chat tab in focus"
              />
            </div>
            <p className="mt-8 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Once you are in, <Key>tab</Key> cycles the right panel — chat, agents, board, mail,
              activity, git — and the boss answers in the same window. The full tour of the chat
              surface is in{' '}
              <Link
                href="/docs/chat-and-threads"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                chat &amp; work threads
              </Link>
              .
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              3 · Resume a session
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              The office re-opens your last chat. You can overrule it.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              By default the office restores wherever you left off. To step back into a specific
              past session, hand it the ID — a bad or forgotten ID never blocks the door, the app
              warns and boots normally with a fresh one.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '# resume a specific past session (short form: -s)', dim: true },
                  { t: 'theboringoffice --session <your-session-id>' },
                ]}
              />
            </div>
            <p className="mt-8 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              In-app, <Key>/session</Key> opens a picker of the past sessions the server keeps for
              this directory: type to narrow, <Key>↑</Key>/<Key>↓</Key> move, <Key>enter</Key>{' '}
              accepts, <Key>esc</Key> cancels with no side effects. Accepting switches the office
              live and the choice sticks to the next boot. If the boss is mid-work the switch is
              refused — <Key>/stop</Key> it or wait. With no server list to show (demo mode, server
              down), <Key>/session</Key> falls back to printing the current ID and where it lives on
              disk so you can note it down for the flag later.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Session IDs are pinned per transport — swapping backends never cross-pinches sessions.
              The mechanics are on the{' '}
              <Link href="/docs/backends" className="text-foreground underline underline-offset-4 hover:text-accent">
                backends page
              </Link>
              .
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              4 · Where your brain lives
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              One config file, one session file per working directory.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The whole office is run by{' '}
              <code className="font-mono text-xs text-foreground">
                ~/.theboringoffice/configs/brain.json
              </code>{' '}
              — created with defaults on first run, inspectable anytime with{' '}
              <code className="font-mono text-xs text-foreground">
                theboringoffice --print-default-config
              </code>
              . Session state lives per working directory at{' '}
              <code className="font-mono text-xs text-foreground">
                ~/.theboringoffice/projects/&lt;dirhash&gt;/session.json
              </code>
              , so two projects never share a floor.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Upgrading from an older build? Nothing to migrate by hand: old{' '}
              <code className="font-mono text-xs text-foreground">
                ~/.theboringoffice/sessions/&lt;dirhash&gt;
              </code>{' '}
              session files and the pre-rename{' '}
              <code className="font-mono text-xs text-foreground">~/.grafeio</code> config, theme
              and sessions are still READ — writes land on the new paths only — and{' '}
              <code className="font-mono text-xs text-foreground">GRAFEIO_*</code> env vars keep
              working as fallbacks for the{' '}
              <code className="font-mono text-xs text-foreground">THEBORINGOFFICE_*</code> ones.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>What this doesn&apos;t do yet</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              The installer wires opencode and claudecode only — Codex, Cursor, and Pi are still
              marked &quot;(Coming Soon)&quot; elsewhere on this site. And{' '}
              <code className="font-mono text-xs text-foreground">--session</code> restores sessions
              for the working directory you boot from, not another one&apos;s history. Next stops:{' '}
              <Link href="/docs/backends" className="text-foreground underline underline-offset-4 hover:text-accent">
                backends
              </Link>
              , then the{' '}
              <Link
                href="/docs/keys-and-slash"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                keys &amp; slash commands
              </Link>{' '}
              reference.
            </p>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
