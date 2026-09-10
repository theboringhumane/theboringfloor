import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Panel, Sheet } from '@/components/paper'

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
    <Sheet tone="panel" className="overflow-hidden border-t border-rule">

      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10 lg:px-14">
        <ScrollReveal>
          <Beat
            index="2.4"
            label="Wire up the agent"
            title="Your coding agent can drive the office."
          >
            <p className="max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
              <span className="font-mono text-ink">thefloor_mcp</span> is the
              local MCP server shipped with the office. A configured OpenCode,
              Claude Code, or Codex agent can present plans, read the approved
              plan and recent transcript, search this project&apos;s recent tail,
              and check office status. The two plan-writing tools require a
              running office: without one, they return an error. They only
              present or update drafts; the member reviews, edits, and approves
              with ctrl+x twice.
            </p>
          </Beat>
        </ScrollReveal>

        <ScrollReveal className="mt-12">
          <Panel label="thefloor_mcp — agent tools">
            <pre className="overflow-x-auto font-mono text-xs leading-relaxed text-ink">
              {`plan_present { text }
plan_update { text }
plan_get_approved {}
transcript_read { limit? }
transcript_search { query, limit? }
office_status {}`}
            </pre>
          </Panel>
        </ScrollReveal>

        <ScrollReveal stagger={0.06} className="mt-12 border-t border-rule">
          {cards.map((card) => (
            <div
              key={card.tool}
              className="grid grid-cols-1 items-start gap-x-8 gap-y-2 border-b border-rule py-6 md:grid-cols-[13rem_1fr]"
            >
              <span className="font-mono text-xs text-blue">{card.tool}</span>
              <div>
                <h3 className="text-base tracking-tight text-ink">{card.title}</h3>
                <p className="mt-2 max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
                  {card.body}
                </p>
              </div>
            </div>
          ))}
        </ScrollReveal>

        <ScrollReveal className="mt-8">
          <Link
            href="/docs/mcp-server"
            className="mono-label inline-flex w-fit items-center border-b border-rule pb-0.5 text-ink transition-colors hover:text-blue"
          >
            Read the MCP server docs
          </Link>
        </ScrollReveal>
      </div>
    </Sheet>
  )
}
