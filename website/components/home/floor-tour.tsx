'use client'

import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'
import { SectionTag } from '@/components/section-tag'
import { gsap, ScrollTrigger } from '@/lib/gsap'

const tabs = [
  {
    id: '01',
    label: 'The Floor',
    heading: 'A floor that lives',
    body: "tekton walks a ticket to the board; hemerodromos drops mail at the tray. Every sprite is a real sub-agent event from your transport — opencode or Claude Code — so a glance at the floor replaces a dashboard.",
    points: [
      'Real sub-agents, opencode or Claude Code, behind every walk cycle',
      'Coffee steam, blinking rack LEDs, an uplink ripple — all tick-driven',
      'Six tabs, one office: chat, terminal, agents, board, mail, activity',
    ],
    log: [
      { text: '> tekton-03 walking to board…', tone: 'muted' as const },
      { text: 'task claimed — flaky-test-482', tone: 'success' as const },
      { text: '> hemerodromos-01 dropping mail', tone: 'muted' as const },
      { text: 'idle: skopos-02 refilling tea', tone: 'warning' as const },
    ],
  },
  {
    id: '02',
    label: 'Live Work Threads',
    heading: 'Every diff, out loud',
    body: 'Work comes back the way a coworker would report it: diffs, tool calls, and thinking rendered as opencode-style threads, right in the chat — nothing buried in a log.',
    points: [
      'Line-numbered diffs with full-row red/green tints and inline syntax',
      'Thinking transcripts stream in real time, then auto-collapse',
      'Click the header — or ctrl+g — to expand the full thread',
    ],
    log: [
      { text: '> tekton-03 opened work thread', tone: 'muted' as const },
      { text: 'diff applied — auth.go +18 -4', tone: 'success' as const },
      { text: '> thinking… collapsed after 6 lines', tone: 'muted' as const },
      { text: 'thread pending review: dikastes-01', tone: 'warning' as const },
    ],
  },
  {
    id: '03',
    label: 'Permission Queue',
    heading: 'Never locked out',
    body: 'Permission asks wait their turn instead of stealing your screen. They stack 1 of N, and boss questions open as classified popovers you can answer in one keystroke — the office keeps working, and so do you.',
    points: [
      '"1 of N" queue with allow-once, always, or reject — y, a, n, esc',
      'Questions auto-classified: text, radio, checkbox, confirm',
      '/perm and /question re-open anything you dismissed',
    ],
    log: [
      { text: '> chmod +x scripts/deploy.sh', tone: 'muted' as const },
      { text: 'permission queued — 1 of 3', tone: 'success' as const },
      { text: '> awaiting boss decision…', tone: 'muted' as const },
      { text: 'unanswered: 2 questions in queue', tone: 'warning' as const },
    ],
  },
  {
    id: '04',
    label: 'Concierge & Backlog',
    heading: 'Send anytime, blocked never',
    body: 'Message the boss mid-task the way you’d tap a coworker’s shoulder. The concierge answers instantly, and anything that has to wait piles into a backlog the office manages — not a black hole.',
    points: [
      'Concierge replies as a real turn — noted as "office routed: boss busy"',
      'Backlog flushes as one batch, decomposed into parallel sub-agents',
      'A dead boss respawns a fresh session and resends the batch',
    ],
    log: [
      { text: '> boss busy — concierge routing reply', tone: 'muted' as const },
      { text: 'backlog: 4 messages queued', tone: 'success' as const },
      { text: '> BATCH DISPATCH — 4 parallel sub-agents', tone: 'muted' as const },
      { text: 'boss session died — respawning…', tone: 'warning' as const },
    ],
  },
]

export function FloorTour() {
  const [active, setActive] = useState(0)
  const wrapperRef = useRef<HTMLDivElement>(null)
  const pinRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const activeRef = useRef(0)
  const tab = tabs[active]

  // Scroll-hijack: pin the tab panel while scrolling through the wrapper's
  // extra height, and advance the active tab based on scroll progress.
  useEffect(() => {
    const wrapper = wrapperRef.current
    const pin = pinRef.current
    if (!wrapper || !pin) return

    const mm = gsap.matchMedia()

    mm.add('(min-width: 1024px)', () => {
      const st = ScrollTrigger.create({
        trigger: wrapper,
        start: 'top top+=64',
        end: 'bottom bottom',
        pin,
        pinSpacing: false,
        onUpdate: (self) => {
          const i = Math.min(tabs.length - 1, Math.floor(self.progress * tabs.length))
          if (i !== activeRef.current) {
            activeRef.current = i
            setActive(i)
          }
        },
      })

      return () => st.kill()
    })

    return () => mm.revert()
  }, [])

  // Fade the terminal + copy panel whenever the active tab changes.
  useEffect(() => {
    if (!panelRef.current) return
    gsap.fromTo(
      panelRef.current.querySelectorAll('[data-panel-item]'),
      { opacity: 0, y: 10 },
      { opacity: 1, y: 0, duration: 0.5, stagger: 0.06, ease: 'power2.out' },
    )
  }, [active])

  return (
    // Stable public anchor; do not rename.
    <section id="toolkits" className="border-b border-border">
      <div ref={wrapperRef} className="relative lg:h-[280vh]">
        <div ref={pinRef} className="mx-auto max-w-7xl px-6 py-20 lg:py-0 lg:min-h-svh lg:flex lg:flex-col lg:justify-center">
          <SectionTag>Why it feels human</SectionTag>

          <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
            A boss you can talk to.
            <br />
            A team you can watch work.
          </h2>

          <div className="mt-14 grid grid-cols-1 gap-10 lg:grid-cols-[240px_1fr_1fr]">
            <div className="flex flex-col gap-1 border border-border">
              {tabs.map((t, i) => (
                <button
                  key={t.id}
                  onClick={() => setActive(i)}
                  className={cn(
                    'flex items-center gap-3 border-b border-border px-4 py-4 text-left font-mono text-xs uppercase tracking-wider last:border-b-0 transition-colors duration-300',
                    active === i
                      ? 'bg-accent text-accent-foreground'
                      : 'text-muted-foreground hover:bg-secondary',
                  )}
                >
                  <span className="opacity-70">{t.id}</span>
                  {t.label}
                </button>
              ))}
            </div>

            <div
              ref={panelRef}
              className="flex min-h-72 flex-col justify-between gap-6 border border-border bg-card p-6"
            >
              <div data-panel-item className="flex items-center gap-2 border-b border-border pb-3">
                <span className="size-2.5 rounded-full bg-destructive/70" />
                <span className="size-2.5 rounded-full bg-chart-4/70" />
                <span className="size-2.5 rounded-full bg-chart-2/70" />
              </div>
              <div className="space-y-3 font-mono text-xs leading-relaxed text-muted-foreground">
                <p data-panel-item className="text-foreground">
                  {tab.heading.toUpperCase()}
                </p>
                {tab.log.map((line, i) => (
                  <p
                    key={i}
                    data-panel-item
                    className={cn(
                      line.tone === 'success' && 'text-chart-2',
                      line.tone === 'warning' && 'text-chart-4',
                    )}
                  >
                    {line.text}
                  </p>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-6 py-2">
              <span className="font-mono text-xs text-muted-foreground">{tab.id}</span>
              <h3 className="text-2xl font-semibold tracking-tight">{tab.heading}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">{tab.body}</p>
              <ul className="flex flex-col gap-3 border-l border-border pl-4">
                {tab.points.map((p) => (
                  <li key={p} className="text-sm leading-relaxed text-foreground/90">
                    {p}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
