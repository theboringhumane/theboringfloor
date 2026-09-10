import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Sheet } from '@/components/paper'

const features = [
  {
    title: 'Permission Queue',
    description:
      'The office keeps moving while asks wait for your call. Requests stack as 1 of N with allow-once, always, or reject — your judgment requested, never hijacked.',
    visual: '/shots/permission-queue.png' as const,
  },
  {
    title: 'Work Threads',
    description:
      'Hear what changed from the worker who changed it. Diffs, tool calls, and thinking stay attached to the work, so review feels like a handoff — not a log you have to dig through.',
    visual: '/shots/work-threads-diffs.png' as const,
  },
]

export function FeatureGrid() {
  return (
    <Sheet id="products" tone="panel" className="overflow-hidden border-t border-rule">

      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10 lg:px-14">
        <Beat
          index="2.2"
          label="Read the index"
          title="Two surfaces carry the day."
        >
          <p className="max-w-xl text-pretty text-sm leading-relaxed text-ink-soft">
            Everything else on the floor hangs off these. One holds your
            decisions. One holds the work.
          </p>

          <ScrollReveal stagger={0.1} className="mt-12 border-t border-rule">
            {features.map((f, i) => (
              <div
                key={f.title}
                className="grid grid-cols-1 items-start gap-6 border-b border-rule py-8 md:grid-cols-[3rem_1fr_16rem]"
              >
                <span className="font-mono text-xs text-ink-faint">
                  {String(i + 1).padStart(2, '0')}
                </span>
                <div className="flex flex-col gap-3">
                  <h3 className="display-md text-ink">{f.title}</h3>
                  <p className="max-w-md text-pretty text-sm leading-relaxed text-ink-soft">
                    {f.description}
                  </p>
                  <Link
                    href="/docs"
                    className="mono-label w-fit border-b border-rule pb-0.5 text-ink transition-colors hover:text-blue"
                  >
                    Learn more
                  </Link>
                </div>
                <img
                  src={f.visual}
                  alt={f.title}
                  className="shot-img hairline w-full object-cover"
                />
              </div>
            ))}
          </ScrollReveal>
        </Beat>
      </div>
    </Sheet>
  )
}
