import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import { Beat, Cue, Sheet } from '@/components/paper'

const notes = [
  ['a', 'An inbox for decisions', 'See which floors are working and which need you. Review a presented plan from your phone.'],
  ['b', 'Tickets tied to conversations', 'Prepare a handoff with acceptance criteria, link its conversation, and get back to the discussion from the ticket.'],
  ['c', 'A result you can review', 'Result summaries and verification notes sit beside the checklist. Read the recorded evidence before you accept the work.'],
]

export function MobileOffice() {
  return (
    <Sheet className="border-b border-rule">
      <div className="px-6 py-16 md:px-10 md:py-20 lg:px-14">
        <Beat index="1.2" label="Carry the floor">
          <div className="grid items-start gap-12 lg:grid-cols-2">
            <div>
              <h2 className="display-lg mt-6 max-w-lg text-balance text-ink">Know what needs you.<br />Leave room to focus.</h2>
              <p className="mt-6 max-w-lg text-base leading-relaxed text-ink-soft">The Android inbox collects plan reviews, blocked tickets and work waiting on review. Open a floor, make the next decision, leave the context with the work.</p>

              <div className="mt-10 border-t border-rule">
                {notes.map(([mark, title, text]) => (
                  <div key={title} className="border-b border-rule py-4">
                    <h3 className="flex items-baseline gap-3 font-medium tracking-tight text-ink">
                      <span className="hairline px-1.5 py-0.5 font-mono text-[0.6875rem] font-normal leading-none">{mark}</span>
                      {title}
                    </h3>
                    <p className="mt-2 max-w-md text-sm leading-relaxed text-ink-soft">{text}</p>
                  </div>
                ))}
              </div>

              <div className="mt-8 flex flex-wrap items-center gap-4">
                <a href="https://github.com/theboringhumane/theboringfloor/releases/latest" className="inline-flex items-center gap-2 bg-ink px-5 py-3 font-mono text-[11px] uppercase tracking-[0.14em] text-paper transition-opacity hover:opacity-90">Get Android <ArrowUpRight className="size-3.5" aria-hidden /></a>
                <Link href="/docs/workspaces#android" className="inline-flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft underline underline-offset-4 transition-colors hover:text-ink">Explore the workflow <ArrowUpRight className="size-3.5" aria-hidden /></Link>
              </div>
              <p className="mt-6 max-w-md text-xs leading-relaxed text-ink-faint">Connects to your authenticated gateway. Inbox execution states and ticket handoffs need office and gateway v0.7.0+. Tool permissions and assistant questions stay in the desktop office.</p>
            </div>

            <figure className="doc-brackets hairline bg-paper-2 p-5 md:p-6">
              <div className="grid grid-cols-2 items-start gap-4 sm:gap-6">
                <img src="/shots/mobile/inbox.webp" alt="Android inbox showing floors that need plan and ticket review." width={412} height={892} loading="lazy" className="h-auto w-full border border-rule" />
                <img src="/shots/mobile/ticket-review.webp" alt="Ticket review with acceptance criteria, result summary, and recorded verification notes." width={412} height={892} loading="lazy" className="mt-10 h-auto w-full border border-rule" />
              </div>
              <figcaption className="mt-5 flex flex-wrap items-center justify-between gap-2">
                <span className="mono-label">Fig. 1.2 — actual Flutter UI, illustrative data</span>
                <Cue>the phone is for deciding, not typing</Cue>
              </figcaption>
            </figure>
          </div>
        </Beat>
      </div>
    </Sheet>
  )
}
