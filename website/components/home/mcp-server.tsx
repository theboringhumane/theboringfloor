import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'
import { SectionTag } from '@/components/section-tag'

const cards = [
  {
    tool: 'plan_present',
    title: 'Present a plan draft',
    body: 'Puts a plan draft in the member’s plan pane. Requires a live office; it only presents the draft and does not execute work or approve anything.',
  },
  {
    tool: 'plan_update',
    title: 'Update the draft',
    body: 'Updates the plan draft in the member’s pane. Requires a live office; the member still reviews, edits, and approves the plan.',
  },
  {
    tool: 'plan_get_approved',
    title: 'Read the approved plan',
    body: 'Does not require a live office. Reads the member-approved plan from the live office, or from its on-disk snapshot when the office is not running.',
  },
  {
    tool: 'transcript_read',
    title: 'Read recent messages',
    body: 'Does not require a live office. Reads recent transcript messages from the live office, or from its on-disk snapshot when the office is not running.',
  },
  {
    tool: 'transcript_search',
    title: 'Search this project’s tail',
    body: 'Does not require a live office. Searches this project only, using the on-disk transcript’s most recent 200 messages—not a complete or cross-project history.',
  },
  {
    tool: 'office_status',
    title: 'Check the office',
    body: 'Does not require a live office. Reports whether the office is live, which backend it uses, and its message counts.',
  },
]

export function McpServer() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <ScrollReveal>
          <SectionTag>The office MCP server</SectionTag>
          <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
            Your coding agent can drive the office.
          </h2>
          <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
            <span className="font-mono text-foreground">thefloor_mcp</span> is the local MCP
            server shipped with the office. A configured OpenCode or Claude Code agent can present
            plans, read the approved plan and recent transcript, search this project’s recent tail,
            and check office status. The two plan-writing tools require a running office: without
            one, they return an error. They only present or update drafts; the member reviews, edits,
            and approves with ctrl+x twice.
          </p>
        </ScrollReveal>

        <ScrollReveal className="mt-10 border border-border bg-card">
          <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
            <span className="font-mono text-xs text-muted-foreground">thefloor_mcp — agent tools</span>
          </div>
          <pre className="overflow-x-auto px-6 py-5 font-mono text-xs leading-relaxed text-muted-foreground">
            {`plan_present { text }
plan_update { text }
plan_get_approved {}
transcript_read { limit? }
transcript_search { query, limit? }
office_status {}`}
          </pre>
        </ScrollReveal>

        <ScrollReveal
          stagger={0.06}
          className="mt-6 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 lg:grid-cols-3"
        >
          {cards.map((card) => (
            <div key={card.tool} className="flex flex-col gap-4 bg-background p-8">
              <span className="font-mono text-xs text-muted-foreground">{card.tool}</span>
              <h3 className="text-lg font-semibold tracking-tight">{card.title}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">{card.body}</p>
            </div>
          ))}
        </ScrollReveal>

        <ScrollReveal className="mt-6">
          <Link
            href="/docs/mcp-server"
            className="inline-flex w-fit items-center border border-border px-4 py-2 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
          >
            Read the MCP server docs
          </Link>
        </ScrollReveal>
      </div>
    </section>
  )
}
