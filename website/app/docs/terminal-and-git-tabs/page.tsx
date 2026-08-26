import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Terminal & git tabs | theboringoffice',
  description:
    'A real PTY in the sidebar with opt-in key capture and mouse copy, and a git tab with live status counts and a colored diff viewer.',
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
    <figure className="mt-10 overflow-hidden border border-border bg-black">
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

export default function TerminalAndGitTabsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Docs — terminal & git</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              A real shell, mounted in the office wall.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Two of the sidebar&apos;s seven tabs are not telemetry. The terminal tab is
              a PTY running your <Code>$SHELL</Code>; the git tab is a live repo cockpit
              with a colored diff viewer. Both behave the way the tools you already had
              do — the office just stopped making you leave for them.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Terminal tab</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              A real PTY, spawned when you first look at it.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The terminal tab runs an OS shell (<Code>$SHELL</Code>) on a real PTY
              (creack/pty). It is lazily spawned on first visit — the office never pays
              for a shell you never open — it resizes with the panel, the mouse scrolls
              the scrollback, and when the shell dies <Code>r</Code> respawns it in
              place.
            </p>
            <Shot
              src="/shots/docs/terminal-tab.png"
              alt="theboringoffice terminal tab: a live PTY running a shell in the sidebar"
              caption="terminal — a live PTY in the sidebar, office keys still working"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keyboard capture</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              One key decides who owns the keyboard.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              By default the terminal does <em>not</em> grab the keyboard. Released is
              the default state: on the terminal tab the office keys keep working —{' '}
              <Code>tab</Code>, <Code>shift+tab</Code>, <Code>1..7</Code> and{' '}
              <Code>q</Code> still do office things. <Code>ctrl+space</Code> is the one
              capture toggle and it flips both ways: dive in and every key goes to the
              shell (<Code>tab</Code> completes, <Code>shift+tab</Code> sends the
              backtab sequence, digits type, <Code>q</Code> types, <Code>ctrl+c</Code>{' '}
              is SIGINT); the same key releases you back to the office keys in place.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Two guards keep it honest. <Code>ctrl+o</Code> also releases — an alias
              that can release but never dive — and leaving the tab auto-releases, so
              every visit starts released. The toggle itself cannot collide with your
              shell bindings: <Code>ctrl+space</Code> emits byte <Code>0x00</Code>, a
              key of its own that nothing else shares. The status bar says which mode
              you are in — <Code>office keys · ctrl+space &rarr; shell</Code> released,{' '}
              <Code>typing &rarr; shell · ctrl+space release</Code> captured.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="ctrl+space" action="toggle shell capture, both ways" />
              <Chip combo="ctrl+o" action="release (never dives)" />
              <Chip combo="leave the tab" action="auto-releases" />
              <Chip combo="r" action="respawn a dead shell" />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Mouse select & copy</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Drag, release, copied.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Drag over terminal text — the live screen or scrolled-up scrollback,
              captured or released — and the span highlights in reverse video. On
              release it copies to the system clipboard (pbcopy, wl-copy, xclip or xsel
              on PATH, plus OSC52 best-effort) and a dim{' '}
              <Code>· Copied N chars</Code> note rides the badge row. No follow-up
              keypress: the release is the copy. <Code>esc</Code> or typing cancels the
              highlight instead.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Shell-side mouse modes never fight this: the grid ignores the PTY&apos;s
              mouse-reporting requests and no mouse bytes reach the shell — selection
              stays yours, full-screen TUI apps keep their keyboard focus.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Git tab</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Live repo state, one glance wide.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The git tab reads the working tree the office sits on. A header summary
              carries the counts — modified, added, untracked, deleted, plus a +/-
              lines readout — above a scrollable file list with status glyphs:{' '}
              <Code>M</Code> modified, <Code>A</Code> added, <Code>??</Code> untracked,{' '}
              <Code>D</Code> deleted, <Code>R</Code> renamed; a <Code>*</Code> suffix
              means staged. A clean tree just says{' '}
              <Code>working tree clean</Code>.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>enter</Code> or a click on a file opens its colored unified diff
              inline — <Code>+</Code> rows green, <Code>-</Code> rows red, hunk headers
              marked with <Code>@@</Code>. <Code>b</Code> or <Code>esc</Code> returns
              to the file list and <Code>r</Code> refreshes. Enough to review what a
              worker just wrote without spawning another pane.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="enter / click" action="open the colored diff" />
              <Chip combo="b / esc" action="back to the file list" />
              <Chip combo="r" action="refresh" />
            </div>
            <Shot
              src="/shots/docs/git-tab.png"
              alt="theboringoffice git tab: status counts, file list with status glyphs, and a colored unified diff"
              caption="git — status counts, the file list, and a colored diff on enter"
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
                <strong className="text-foreground">One shell, one PTY.</strong> The
                terminal tab is not a multiplexer — splits and panes remain tmux&apos;s
                job, and the office is fine inside tmux.
              </li>
              <li>
                <strong className="text-foreground">OSC52 is best-effort.</strong>{' '}
                Terminals that strip OSC52 quietly get nothing back; the badge still
                reports what the office sent.
              </li>
              <li>
                <strong className="text-foreground">The git tab reads; it never
                writes.</strong> No staging, committing or conflict resolution — that
                is what the terminal tab two keys away is for.
              </li>
              <li>
                <strong className="text-foreground">One file&apos;s diff at a
                time.</strong> The viewer is a review surface, not a three-way merge
                tool.
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
                  body: 'ctrl+space and the rest of the key table, in full.',
                },
                {
                  href: '/docs/chat-and-threads',
                  title: 'Chat & threads',
                  body: 'The tab to the terminal\u2019s left: the boss conversation.',
                },
                {
                  href: '/docs/getting-started',
                  title: 'Getting started',
                  body: 'Install, first boot, first dispatch.',
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
