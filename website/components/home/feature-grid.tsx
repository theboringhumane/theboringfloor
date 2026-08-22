import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'

const features = [
  {
    title: 'Permission Queue',
    description:
      'Let agents keep moving while you stay in control. Requests queue up with clear allow-once, always, or reject choices when your attention is needed.',
    visual: '/shots/permission-queue.png' as const,
  },
  {
    title: 'Work Threads',
    description:
      'Know what changed before you review it. Diffs, tool calls, and thinking stay attached to the work instead of disappearing into a terminal log.',
    visual: '/shots/work-threads-diffs.png' as const,
  },
]

export function FeatureGrid() {
  return (
    <section id="products" className="border-b border-border">
      <div className="mx-auto grid max-w-7xl grid-cols-1 md:grid-cols-2">
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
            <div className="flex flex-1 flex-col justify-center gap-4 px-6 py-10 md:px-10">
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
              <img src={f.visual} alt={f.title} className="h-full w-full object-cover" />
            </div>
          </ScrollReveal>
        ))}
      </div>
    </section>
  )
}
