import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Cue, Sheet, Stamp } from '@/components/paper'

const keys = [
  { combo: 'ctrl+p', action: 'plan' },
  { combo: 'ctrl+x twice', action: 'approve → build' },
  { combo: '[plan]', action: 'statusbar badge' },
]

export function PlanThenBuild() {
  return (
    <Sheet className="overflow-hidden border-t border-rule">

      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10 lg:px-14">
        <ScrollReveal>
          <Beat
            index="2.3"
            label="Read the plan first"
            title="The office drafts the plan. You approve it. Then the crew builds."
          >
            <p className="max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
              Substantial requests enter planning automatically. The boss
              investigates, presents a draft with the floor MCP or plan markers,
              and waits for review. The plan opens beside your conversation,
              even from an expanded or focused view. Edit it, then press Ctrl+X
              twice to approve. Ctrl+P lets you switch modes yourself.
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              {keys.map((k) => (
                <Stamp key={k.combo} tone="ink" className="gap-2">
                  <span className="text-blue">{k.combo}</span>
                  <span className="text-ink-faint">{k.action}</span>
                </Stamp>
              ))}
            </div>
          </Beat>
        </ScrollReveal>

        <ScrollReveal delay={0.1} className="mx-auto mt-14 max-w-5xl">
          <div className="doc-brackets hairline overflow-hidden bg-(--shot-frame)">
            <div className="mono-label border-b border-rule px-4 py-2.5">
              plan.md — plan mode
            </div>
            <img
              src="/shots/workspaces/plan.webp"
              alt="theboringfloor plan mode: project floors, a markdown plan editor, and the conversation"
              width={1548}
              height={1014}
              loading="lazy"
              className="shot-img block h-auto w-full"
            />
          </div>
          <Cue className="mt-4 block">fig. 2.3 — the draft, before anyone builds</Cue>
        </ScrollReveal>
      </div>
    </Sheet>
  )
}
