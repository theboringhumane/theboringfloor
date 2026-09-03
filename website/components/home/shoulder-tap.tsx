import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const cards = [
  {
    id: '01',
    title: 'Silent while you watch',
    body: 'The office tracks focus and blur on your terminal. While you can see the floor, it says nothing at all — a good coworker doesn&apos;t tap your shoulder when you&apos;re already looking at them.',
  },
  {
    id: '02',
    title: 'Blocked on your call',
    body: 'When an agent can&apos;t move without you, one OS notification names the employee and the tool — &quot;permission needed — tekton-1 needs write&quot;. The floor waits on your decision, not the other way around.',
  },
  {
    id: '03',
    title: 'The boss is done',
    body: 'When a turn completes, one ping carries the gist of the finished work — &quot;the boss is done — …&quot;. Enough, while you&apos;re pouring the tea, to decide whether it&apos;s worth walking back.',
  },
  {
    id: '04',
    title: 'One ping, never a wall',
    body: 'Ten asks in a burst coalesce into a single banner. The office would rather lose a little detail in the summary than spend your attention twice.',
  },
]

const hatches = [
  { combo: '/notify on|off', action: 'toggle it in chat' },
  { combo: 'THEBORINGFLOOR_NO_NOTIFY=1', action: 'env kill switch' },
  { combo: 'ui.notifications', action: 'brain.json setting' },
]

export function ShoulderTap() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <ScrollReveal>
          <SectionTag>The tap on your shoulder</SectionTag>
          <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
            The office is quiet on purpose.
          </h2>
          <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
            Your attention is the most precious resource on the floor, so the office spends
            almost none of it. While you&apos;re watching, it stays fully silent. Away from the
            keyboard, it taps your shoulder exactly twice — when the crew is blocked on your
            decision, and when the work is done. Good coworkers don&apos;t hover.
          </p>
        </ScrollReveal>

        <ScrollReveal
          stagger={0.06}
          className="mt-14 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 lg:grid-cols-4"
        >
          {cards.map((c) => (
            <div key={c.id} className="flex flex-col gap-4 bg-background p-8">
              <span className="font-mono text-xs text-muted-foreground">{c.id}</span>
              <h3 className="text-lg font-semibold tracking-tight">{c.title}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                {c.body}
              </p>
            </div>
          ))}
        </ScrollReveal>
        {'\n'}
        <ScrollReveal className="mt-6 flex flex-wrap items-center gap-3">
          {hatches.map((h) => (
            <span
              key={h.combo}
              className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs"
            >
              <span className="text-accent">{h.combo}</span>
              <span className="text-muted-foreground">{h.action}</span>
            </span>
          ))}
        </ScrollReveal>
        {'\n'}
        <ScrollReveal className="mt-6 border border-border bg-card">
          <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
            <span className="size-2.5 rounded-full bg-destructive/70" />
            <span className="size-2.5 rounded-full bg-chart-4/70" />
            <span className="size-2.5 rounded-full bg-chart-2/70" />
            <span className="ml-2 font-mono text-xs text-muted-foreground">
              what the tap isn&apos;t
            </span>
          </div>
          <p className="px-6 py-6 text-pretty text-sm leading-relaxed text-muted-foreground">
            No click-to-return today. The banner comes from your OS — osascript on macOS,
            notify-send on Linux — delivered with zero new dependencies, and an OS banner
            can&apos;t focus a terminal window without a signed helper app. So the deal is an
            honest one: <span className="text-foreground">you hear it, you tab back.</span>{' '}
            The office rings the bell; your feet do the walking.
          </p>
          <p className="border-t border-border px-6 py-3 font-mono text-xs text-muted-foreground">
            Off means off — /notify off, the env var, or the config key all end the same way:
            silence.{'\n'}
          </p>
        </ScrollReveal>
      </div>
    </section>
  )
}
