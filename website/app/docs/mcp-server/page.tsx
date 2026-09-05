import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'MCP server | theboringfloor',
  description:
    'thefloor_mcp connects your configured agent to the local office for plan drafts, approved plans, transcript reads, and office status.',
  alternates: {
    canonical: '/docs/mcp-server',
  },
  openGraph: {
    title: 'MCP server · theboringfloor',
    description:
      'thefloor_mcp connects your configured agent to the local office for plan drafts, approved plans, transcript reads, and office status.',
    url: `${SITE_URL}/docs/mcp-server`,
    type: 'website',
  },
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] border border-border bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

type Tool = {
  name: string
  args: string
  does: string
  availability: string
  example: string
}

const tools: Tool[] = [
  {
    name: 'plan_present',
    args: '{ text }',
    does: 'Present a plan draft in the plan pane.',
    availability: 'Live office required.',
    example: 'plan_present({ text: "# Goal\\nReview this plan." })',
  },
  {
    name: 'plan_update',
    args: '{ text }',
    does: 'Update the plan draft in the plan pane.',
    availability: 'Live office required.',
    example: 'plan_update({ text: "# Goal\\nReview the revised plan." })',
  },
  {
    name: 'plan_get_approved',
    args: '{}',
    does: 'Read the member-approved plan.',
    availability: 'Live or on-disk.',
    example: 'plan_get_approved({})',
  },
  {
    name: 'transcript_read',
    args: '{ limit? }',
    does: 'Read recent office transcript messages.',
    availability: 'Live or on-disk.',
    example: 'transcript_read({ limit: 20 })',
  },
  {
    name: 'transcript_search',
    args: '{ query, limit? }',
    does: 'Search this project’s recent transcript tail.',
    availability: 'On-disk, current project only.',
    example: 'transcript_search({ query: "approval", limit: 10 })',
  },
  {
    name: 'office_status',
    args: '{}',
    does: 'Report whether the office is live, its backend, and message counts.',
    availability: 'Available without a live office.',
    example: 'office_status({})',
  },
]

const keepReading = [
  {
    href: '/docs/plan-mode',
    kicker: 'related',
    title: 'Plan mode',
    blurb: 'Review or edit drafts, then press ctrl+x twice to approve one for the build agent.',
  },
  {
    href: '/docs/chat-and-threads',
    kicker: 'related',
    title: 'Chat & work threads',
    blurb: 'The transcript the MCP server can read from the live office or project snapshot.',
  },
  {
    href: '/docs/getting-started',
    kicker: 'start',
    title: 'Getting started',
    blurb: 'Install the office, choose a backend, and open your first live session.',
  },
]

export default function MCPServerPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>MCP server</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              Give your configured agent a local line to the office.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              <Code>thefloor_mcp</Code> is a local MCP server for the office. It lets a configured
              agent present plan drafts, read the approved plan, read or search the current project
              transcript, and check office status. It does not expose browser tools.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Installation</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              It ships with the office and registers itself.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>thefloor_mcp</Code> ships in the same release archive as <Code>theboringfloor</Code>.
              The installer registers it in your global OpenCode configuration under the{' '}
              <Code>mcp</Code> key. When the Claude CLI is present, it also registers the server with
              Claude Code at user scope.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
               Set <Code>THEFLOOR_NO_MCP_INSTALL=1</Code> to keep the installer from making
              either automatic registration. Set <Code>THEFLOOR_PROJECT_DIR</Code> when you need
              <Code>thefloor_mcp</Code> to bind to a project directory other than its current one.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Tools</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Six tools, all scoped to one office project.
            </h2>
            <div className="mt-8 overflow-x-auto border border-border">
              <table className="w-full min-w-180 text-left text-sm">
                <thead className="border-b border-border bg-card font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  <tr>
                    <th className="px-4 py-3 font-normal">Tool</th>
                    <th className="px-4 py-3 font-normal">Arguments</th>
                    <th className="px-4 py-3 font-normal">Does</th>
                    <th className="px-4 py-3 font-normal">Availability</th>
                  </tr>
                </thead>
                <tbody>
                  {tools.map((tool, index) => (
                    <tr key={tool.name} className={index > 0 ? 'border-t border-border' : ''}>
                      <td className="px-4 py-3 align-top font-mono text-xs text-accent">{tool.name}</td>
                      <td className="px-4 py-3 align-top font-mono text-xs text-foreground">{tool.args}</td>
                      <td className="px-4 py-3 align-top leading-relaxed text-muted-foreground">{tool.does}</td>
                      <td className="px-4 py-3 align-top leading-relaxed text-muted-foreground">{tool.availability}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="mt-8 max-w-3xl border border-border bg-card p-5 font-mono text-xs leading-relaxed text-foreground">
              {tools.map((tool) => (
                <p key={tool.name}>{tool.example}</p>
              ))}
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Plans</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Presentation is not execution.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>plan_present</Code> and <Code>plan_update</Code> only present plan drafts. They
              never execute work and they never approve a plan. Review or edit the draft in the plan
              pane, then press <Code>ctrl+x</Code> twice to approve it for the build agent.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              These write tools require a running office. When the office is not live, they return an
              error rather than silently succeeding; there is no offline fallback. The existing{' '}
              <Link
                href="/docs/plan-mode"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                plan markers
              </Link>{' '}
              remain available as an additional path.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Live and on-disk reads</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Reads can fall back; writes cannot.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>plan_get_approved</Code> and <Code>transcript_read</Code> read from the live
              office when it is running, or the on-disk project snapshot when it is not.
              <Code> transcript_search</Code> searches the on-disk snapshot for the current project
              only. It cannot read another project&apos;s transcript.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The on-disk transcript keeps only its most recent 200 messages per project. That means
              <Code>transcript_search</Code> searches the recent tail, not complete history.
              <Code> office_status</Code> reports whether the office is live, which backend it uses,
              and its message counts.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Office control API</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Local-only, token-protected, and discovered per project.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The office runs its control API on <Code>127.0.0.1</Code> at an ephemeral port and
              requires a bearer token. It writes discovery details to{' '}
              <Code>~/.theboringfloor/projects/&lt;dirhash&gt;/control.json</Code>. The file is mode{' '}
              <Code>0600</Code> and contains the office PID, port, token, project directory,
              start time, and version. <Code>thefloor_mcp</Code> reads it for the project it is bound
              to.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
               Set <Code>THEFLOOR_NO_CONTROL=1</Code> to disable the control API entirely.
              Without a running control API, the write tools return their live-office-required error;
              the documented on-disk read paths remain separate.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {keepReading.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className="flex flex-col gap-2 bg-background p-6 transition-colors hover:bg-secondary"
                >
                  <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                    {link.kicker}
                  </p>
                  <p className="font-medium text-foreground">{link.title}</p>
                  <p className="text-sm leading-relaxed text-muted-foreground">{link.blurb}</p>
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
