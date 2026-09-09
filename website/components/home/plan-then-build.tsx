import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const keys = [
  { combo: 'ctrl+p', action: 'plan' },
  { combo: 'ctrl+x twice', action: 'approve → build' },
  { combo: '[plan]', action: 'statusbar badge' },
]

export function PlanThenBuild() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <ScrollReveal>
          <SectionTag>Plan, then build</SectionTag>
          <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
            The office drafts the plan. You approve it. Then the crew builds.
          </h2>
          <p className="mt-4 max-w-2xl text-pretty text-sm leading-relaxed text-muted-foreground">
            Substantial requests enter planning automatically. The boss investigates,
            presents a draft with the floor MCP or plan markers, and waits for review.
            The plan opens beside your conversation, even from an expanded or focused view.
            Edit it, then press Ctrl+X twice to approve. Ctrl+P lets you switch modes yourself.
          </p>
          <div className="mt-6 flex flex-wrap items-center gap-3">
            {keys.map((k) => (
              <span
                key={k.combo}
                className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs"
              >
                <span className="text-accent">{k.combo}</span>
                <span className="text-muted-foreground">{k.action}</span>
              </span>
            ))}
          </div>
        </ScrollReveal>

        <ScrollReveal
          delay={0.1}
          className="mx-auto mt-10 max-w-5xl overflow-hidden border border-border bg-(--shot-frame)"
        >
          <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
            <span className="size-2.5 rounded-full bg-destructive/70" />
            <span className="size-2.5 rounded-full bg-chart-4/70" />
            <span className="size-2.5 rounded-full bg-chart-2/70" />
            <span className="ml-2 font-mono text-xs text-muted-foreground">
              plan.md — plan mode
            </span>
          </div>
          <img
            src="/shots/workspaces/plan.webp"
            alt="theboringfloor plan mode: project floors, a markdown plan editor, and the conversation"
            width={1548}
            height={1014}
            loading="lazy"
            className="shot-img block h-auto w-full"
          />
        </ScrollReveal>
      </div>
    </section>
  )
}
