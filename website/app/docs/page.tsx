import Link from 'next/link'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata = {
  title: 'Docs | theboringoffice',
  description: 'Reference documentation for hiring, wiring, and running theboringoffice.',
}

const sections = [
  {
    title: 'Quickstart',
    items: ['Installation', 'Choosing a transport (opencode, Claude Code)', 'Hiring your first agent'],
  },
  {
    title: 'Core Concepts',
    items: ['The Office', 'Work Threads', 'Permission Queue', 'Sandboxes'],
  },
  {
    title: 'Floor Plan',
    items: ['Browsing the desks', 'Custom rooms', 'Seating agents by team'],
  },
  {
    title: 'Reference',
    items: ['CLI commands', 'Config file', 'Slack notifications'],
  },
]

export default function DocsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-7xl px-6 py-16">
            <SectionTag>Documentation</SectionTag>
            <h1 className="mt-8 text-balance text-5xl font-bold tracking-tight md:text-6xl">
              Docs
            </h1>
            <p className="mt-4 max-w-xl text-pretty leading-relaxed text-muted-foreground">
              Everything you need to hire your first agent, hand out desks, and keep the office
              running without you in it.
            </p>
          </div>
        </section>

        <section>
          <div className="mx-auto grid max-w-7xl grid-cols-1 gap-px overflow-hidden border-b border-border bg-border md:grid-cols-2 lg:grid-cols-4">
            {sections.map((s) => (
              <div key={s.title} className="flex flex-col gap-3 bg-background p-8">
                <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  {s.title}
                </h2>
                <ul className="flex flex-col gap-2">
                  {s.items.map((item) => (
                    <li key={item}>
                      <Link
                        href="#"
                        className="text-sm text-foreground/90 transition-colors hover:text-accent"
                      >
                        {item}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
