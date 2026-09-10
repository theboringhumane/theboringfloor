import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Panel, Sheet, Stamp } from '@/components/paper'

const cards = [
  {
    id: '01',
    title: 'Silent while you watch',
    body: 'The office tracks focus and blur on your terminal. While you can see the floor, it says nothing at all — a good coworker doesn’t tap your shoulder when you’re already looking at them.',
  },
  {
    id: '02',
    title: 'Blocked on your call',
    body: 'When an agent can’t move without you, one OS notification names the employee and the tool — “permission needed — tekton-1 needs write”. The floor waits on your decision, not the other way around.',
  },
  {
    id: '03',
    title: 'The boss is done',
    body: 'When a turn completes, one ping carries the gist of the finished work — “the boss is done — …”. Enough, while you’re pouring the tea, to decide whether it’s worth walking back.',
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
    <Sheet className="overflow-hidden border-t border-rule">

      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10 lg:px-14">
        <ScrollReveal>
          <Beat
            index="2.5"
            label="Tap the shoulder"
            title="The office is quiet on purpose."
          >
            <p className="max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
              Your attention is the most precious resource on the floor, so the
              office spends almost none of it. While you&apos;re watching, it
              stays fully silent. Away from the keyboard, it taps your shoulder
              exactly twice — when the crew is blocked on your decision, and when
              the work is done. Good coworkers don&apos;t hover.
            </p>
          </Beat>
        </ScrollReveal>

        <ScrollReveal stagger={0.06} className="mt-12 border-t border-rule">
          {cards.map((c) => (
            <div
              key={c.id}
              className="grid grid-cols-1 items-start gap-x-8 gap-y-2 border-b border-rule py-6 md:grid-cols-[3rem_14rem_1fr]"
            >
              <span className="font-mono text-xs text-ink-faint">{c.id}</span>
              <h3 className="text-base tracking-tight text-ink">{c.title}</h3>
              <p className="max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
                {c.body}
              </p>
            </div>
          ))}
        </ScrollReveal>

        <ScrollReveal className="mt-8 flex flex-wrap items-center gap-3">
          {hatches.map((h) => (
            <Stamp key={h.combo} tone="ink" className="gap-2">
              <span className="text-blue">{h.combo}</span>
              <span className="text-ink-faint">{h.action}</span>
            </Stamp>
          ))}
        </ScrollReveal>

        <ScrollReveal className="mt-12">
          <Panel label="what the tap isn’t">
            <p className="max-w-3xl text-pretty text-sm leading-relaxed text-ink-soft">
              No click-to-return today. The banner comes from your OS — osascript
              on macOS, notify-send on Linux — delivered with zero new
              dependencies, and an OS banner can&apos;t focus a terminal window
              without a signed helper app. So the deal is an honest one:{' '}
              <span className="text-ink">you hear it, you tab back.</span> The
              office rings the bell; your feet do the walking.
            </p>
            <p className="mono-label mt-6 border-t border-rule pt-4">
              Off means off — /notify off, the env var, or the config key all end
              the same way: silence.
            </p>
          </Panel>
        </ScrollReveal>
      </div>
    </Sheet>
  )
}
