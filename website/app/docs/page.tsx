import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'
import { Figure } from '@/components/paper'

export const metadata: Metadata = {
  title: 'Docs | theboringfloor',
  description:
    'The office manual — install, backends, chat, plan mode, the queue and board, panels, themes, and every key and slash command.',
  alternates: {
    canonical: '/docs',
  },
  openGraph: {
    title: 'Docs · theboringfloor',
    description:
      'The office manual — install, backends, chat, plan mode, the queue and board, panels, themes, and every key and slash command.',
    url: `${SITE_URL}/docs`,
    type: 'website',
  },
}

type DocLink = { href: string; name: string; promise: string }

const groups: { title: string; items: DocLink[] }[] = [
  {
    title: 'Install & Setup',
    items: [
      {
        href: '/docs/getting-started',
        name: 'Getting started',
        promise: 'One curl, one demo tour, one live office — running in about five minutes.',
      },
    ],
  },
  {
    title: 'Core',
    items: [
      { href: '/docs/workspaces', name: 'Floors & workspaces', promise: 'Project floors, teams, tickets, files, and a backend for each conversation.' },
      {
        href: '/docs/backends',
        name: 'Backends',
        promise: 'OpenCode, Claude Code, and Codex — choose a backend for each conversation.',
      },
      {
        href: '/docs/chat-and-threads',
        name: 'Chat & work threads',
        promise: 'One chat tab, replies that stream character-by-character, sub-agent threads in the open.',
      },
      {
        href: '/docs/plan-mode',
        name: 'Plan mode',
        promise: 'Talk the plan out, edit the draft yourself, approve it — then the crew builds.',
      },
      {
        href: '/docs/mcp-server',
        name: 'MCP server',
        promise: 'The local MCP bridge for plan drafts, approved plans, and the current project transcript.',
      },
      {
        href: '/docs/control-plane',
        name: 'Remote control plane',
        promise: 'floorgate, Tailscale, the Android app, and one authenticated API for live offices.',
      },
    ],
  },
  {
    title: 'Workflow',
    items: [
      {
        href: '/docs/permissions-and-questions',
        name: 'Permissions & questions',
        promise: 'The gate the office never vaults — allow once, always, or reject, in place.',
      },
      {
        href: '/docs/queue-board-memory',
        name: 'Queue, board & memory',
        promise: 'Type while the boss types; board rows flip themselves; memory survives reboots.',
      },
    ],
  },
  {
    title: 'Panels & Reference',
    items: [
      {
        href: '/docs/terminal-and-git-tabs',
        name: 'Terminal & git tabs',
        promise: 'A real shell and a live git panel one tab away from the chat.',
      },
      {
        href: '/docs/browser-tab',
        name: 'Browser tab',
        promise: 'Built-in text navigation, headless screenshots, and your system browser.',
      },
      {
        href: '/docs/layout-themes-power',
        name: 'Layout, themes & power',
        promise: 'Compact mode, themes, and the battery dial that keeps an idle office cheap.',
      },
      {
        href: '/docs/keys-and-slash',
        name: 'Keys & slash commands',
        promise: 'Every key binding and every slash command, one short table each.',
      },
    ],
  },
]

export default function DocsPage() {
  let n = 0
  return (
    <>
      <SiteHeader />
      <main className="relative paper-ground">
        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 lg:px-20 pb-20 pt-16 md:pt-24">
            <SectionTag>Documentation</SectionTag>
            <h1 className="display-lg mt-8 max-w-3xl text-balance text-ink">
              Read the office manual.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-ink-soft">
              Everything shipped, documented the way it runs: install the office, pick a backend,
              work the chat, and learn every panel. No dead links — every entry in the table of
              contents below opens a real page.
            </p>
            <div className="mt-12">
              <Figure caption="theboringfloor — floor, chat, and panels">
                <img
                  src="/shots/workspaces/transcript.webp"
                  alt="theboringfloor overview: the office floor, chat work thread, and panel sidebar"
                  width={1548}
                  height={1014}
                  loading="lazy"
                  className="shot-img block h-auto w-full"
                />
              </Figure>
            </div>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 lg:px-20 py-16 md:py-24">
            <div className="flex items-baseline gap-4 border-b border-rule pb-3">
              <span className="mono-label text-ink">00</span>
              <span className="mono-label">Table of contents</span>
            </div>
            {groups.map((g) => (
              <div key={g.title} className="mt-10">
                <div className="flex items-center gap-3">
                  <span className="mono-label text-ink-faint">{g.title}</span>
                  <span className="h-px flex-1 bg-rule" aria-hidden="true" />
                </div>
                <ol className="mt-2 border-t border-rule">
                  {g.items.map((item) => {
                    n += 1
                    const no = String(n).padStart(2, '0')
                    return (
                      <li key={item.href} className="border-b border-rule">
                        <Link
                          href={item.href}
                          className="group grid grid-cols-[2.5rem_1fr] items-baseline gap-x-4 gap-y-1 py-5 hover:bg-paper-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue md:grid-cols-[2.5rem_18rem_1fr]"
                        >
                          <span className="font-mono text-[0.6875rem] tracking-[0.14em] text-ink-faint">
                            {no}
                          </span>
                          <span className="font-mono text-sm uppercase tracking-[0.08em] text-ink group-hover:underline group-hover:underline-offset-4">
                            {item.name}
                          </span>
                          <span className="col-start-2 text-sm leading-relaxed text-ink-soft md:col-start-3">
                            {item.promise}
                          </span>
                        </Link>
                      </li>
                    )
                  })}
                </ol>
              </div>
            ))}
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>The honest bit</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-ink-soft">
              OpenCode, Claude Code, and Codex are supported. Capabilities vary by transport — the{' '}
              <Link href="/docs/backends" className="text-ink underline underline-offset-4">
                backends page
              </Link>{' '}
              says which brains are real right now. Never been here before? Start with{' '}
              <Link href="/docs/getting-started" className="text-ink underline underline-offset-4">
                getting started
              </Link>
              .
            </p>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
