import Link from 'next/link'
import { ArrowUpRight, CheckCheck, Inbox, Link2 } from 'lucide-react'
import { SectionTag } from '@/components/section-tag'

export function MobileOffice() {
  return (
    <section className="border-b border-border px-6 py-16 md:px-10 lg:px-14">
      <div className="grid items-center gap-12 lg:grid-cols-2">
        <div>
          <SectionTag>Your office, within reach</SectionTag>
          <h2 className="mt-5 max-w-lg text-balance text-3xl font-medium tracking-tight md:text-5xl">Know what needs you.<br />Leave room to focus.</h2>
          <p className="mt-6 max-w-lg text-base leading-relaxed text-muted-foreground">The Android inbox brings plan reviews, blocked tickets, and work ready for review into one place. Open a floor, make the next decision, and keep the context with the work.</p>
          <div className="mt-8 space-y-6">
            {[
              { Icon: Inbox, title: 'An inbox for decisions', text: 'See which floors are working and which need attention. Review presented plans from your phone.' },
              { Icon: Link2, title: 'Tickets connected to conversations', text: 'Prepare a handoff with acceptance criteria, link its conversation, and return to the discussion from the ticket.' },
              { Icon: CheckCheck, title: 'A result you can review', text: 'Keep result summaries and verification notes beside the checklist. Review recorded evidence before accepting the work.' },
            ].map(({ Icon, title, text }) => <div key={title} className="flex gap-4"><Icon className="mt-1 size-5 shrink-0 text-accent" aria-hidden /><div><h3 className="font-medium">{title}</h3><p className="mt-1 max-w-md text-sm leading-relaxed text-muted-foreground">{text}</p></div></div>)}
          </div>
          <div className="mt-8 flex flex-wrap gap-5">
            <a href="https://github.com/theboringhumane/theboringfloor/releases/latest" className="inline-flex items-center gap-2 bg-foreground px-5 py-3 font-mono text-xs text-background">Get Android <ArrowUpRight className="size-4" aria-hidden /></a>
            <Link href="/docs/workspaces#android" className="inline-flex items-center gap-2 font-mono text-xs underline underline-offset-4">Explore the workflow <ArrowUpRight className="size-4" aria-hidden /></Link>
          </div>
          <p className="mt-4 text-xs leading-relaxed text-muted-foreground">Connect to your authenticated gateway. Inbox execution states and ticket handoffs require office and gateway v0.7.0+. Tool permissions and assistant questions are handled in the desktop office.</p>
        </div>
        <div>
          <div className="grid grid-cols-2 items-start gap-4 sm:gap-6">
            <img src="/shots/mobile/inbox.webp" alt="Android inbox showing floors that need plan and ticket review." width={412} height={892} loading="lazy" className="h-auto w-full rounded-3xl border border-border shadow-xl" />
            <img src="/shots/mobile/ticket-review.webp" alt="Ticket review with acceptance criteria, result summary, and recorded verification notes." width={412} height={892} loading="lazy" className="mt-12 h-auto w-full rounded-3xl border border-border shadow-xl" />
          </div>
          <p className="mt-5 text-center font-mono text-[10px] uppercase tracking-wider text-muted-foreground">Actual Flutter UI · illustrative project data</p>
        </div>
      </div>
    </section>
  )
}
