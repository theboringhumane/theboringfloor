import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'

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
    <section id="products" className="border-b border-border">
      <div className="grid grid-cols-1 md:grid-cols-2">
        {features.map((f, i) => (
          <ScrollReveal
            key={f.title}
            direction={i === 0 ? 'left' : 'right'}
            delay={i * 0.1}
            className={
              i === 0
                ? 'flex flex-col-reverse border-b border-border md:border-b-0 md:border-r md:flex-row'
                : 'flex flex-col-reverse border-b border-border md:border-b-0 md:flex-row'
            }
          >
            <div className="flex flex-1 flex-col justify-center gap-4 px-6 py-10 md:px-10 lg:px-14">
              <h3 className="text-2xl font-semibold tracking-tight">{f.title}</h3>
              <p className="max-w-sm text-pretty text-sm leading-relaxed text-muted-foreground">
                {f.description}
              </p>
              <Link
                href="/docs"
                className="inline-flex w-fit items-center border border-border px-4 py-2 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
              >
                Learn more
              </Link>
            </div>
            <div className="shrink-0 md:w-68">
              <img src={f.visual} alt={f.title} className="shot-img h-full w-full object-cover" />
            </div>
          </ScrollReveal>
        ))}
      </div>
    </section>
  )
}
