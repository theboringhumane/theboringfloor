import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Docs | theboringoffice',
  description:
    'The office manual — install, backends, chat, plan mode, the queue and board, panels, themes, and every key and slash command.',
  alternates: {
    canonical: '/docs',
  },
  openGraph: {
    title: 'Docs · theboringoffice',
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
      {
        href: '/docs/backends',
        name: 'Backends',
        promise: 'opencode server-attach or the claudecode stream-json child — pick and swap the brain.',
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

export default function DocsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Documentation</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              Read the office manual.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Everything shipped, documented the way it runs: install the office, pick a backend,
              work the chat, and learn every panel. No dead links — each card below opens a real
              page.
            </p>
            <div className="mt-12">
              <Shot
                src="/shots/docs/office-overview.png"
                alt="theboringoffice overview: the office floor, chat work thread, and panel sidebar"
                caption="theboringoffice — floor, chat, and panels"
              />
            </div>
          </div>
        </section>

        {groups.map((g) => (
          <section key={g.title} className="border-b border-border">
            <div className="mx-auto max-w-7xl px-6 py-16">
              <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                {g.title}
              </h2>
              <div
                className={`mt-6 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 ${
                  g.items.length > 2 ? 'lg:grid-cols-3' : ''
                }`}
              >
                {g.items.map((item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    className="group flex flex-col gap-3 bg-background p-8 transition-colors hover:bg-secondary"
                  >
                    <span className="text-sm font-medium text-foreground transition-colors group-hover:text-accent">
                      {item.name}
                    </span>
                    <span className="text-sm leading-relaxed text-muted-foreground">
                      {item.promise}
                    </span>
                    <span className="mt-auto pt-3 font-mono text-xs text-muted-foreground">
                      {item.href}
                    </span>
                  </Link>
                ))}
              </div>
            </div>
          </section>
        ))}

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>The honest bit</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              These pages document what ships today. Codex, Cursor, and Pi transports are still
              marked &quot;(Coming Soon)&quot; — the{' '}
              <Link href="/docs/backends" className="text-foreground underline underline-offset-4 hover:text-accent">
                backends page
              </Link>{' '}
              says which brains are real right now. Never been here before? Start with{' '}
              <Link
                href="/docs/getting-started"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
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
