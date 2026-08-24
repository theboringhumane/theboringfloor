import Link from 'next/link'
import { GitBranch, Mail, MessageCircle, MessageSquare } from 'lucide-react'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const forYouActions = [
  { icon: MessageSquare, label: 'See which task is moving across the floor' },
  { icon: MessageCircle, label: 'Open the work thread behind any change' },
  { icon: Mail, label: 'Catch the handoff in the mail tray' },
  { icon: MessageCircle, label: 'Approve the decision, not every step' },
  { icon: MessageSquare, label: 'Walk back in and catch up in seconds' },
  { icon: GitBranch, label: 'Send the backlog out when you’re ready' },
]

const agents = [
  { name: 'tekton-03', status: 'Fixed the flaky test' },
  { name: 'hemerodromos-01', status: 'Delivered the mail run' },
  { name: 'skopos-02', status: 'Watching the dashboards' },
  { name: 'dikastes-01', status: 'Reviewing the deploy diff' },
  { name: 'grammateus-04', status: 'Filed the meeting notes' },
  { name: 'kubernetes-07', status: 'Kept the rack humming' },
]

export function ProductPlatform() {
  return (
    <section className="section-light">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <SectionTag>Built for the team you have today</SectionTag>

        <h2 className="mt-6 max-w-xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
          Start with one desk. Grow into a full floor.
        </h2>
      </div>

      <div className="border-t border-border">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-10 px-6 py-16 lg:grid-cols-2">
          <ScrollReveal direction="left" className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <h3 className="text-2xl font-semibold tracking-tight">theboringoffice</h3>
              <span className="border border-accent px-2 py-0.5 font-mono text-xs uppercase tracking-wider text-accent">
                For You
              </span>
            </div>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-muted-foreground">
              Open one terminal and a real opencode session greets you. When work fans out,
              sub-agents take desks on the floor with the context you need to follow them.
            </p>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-muted-foreground">
              Six tabs, one office: chat, terminal, agents, board, mail, and activity. No separate
              dashboard to configure, no status meeting to schedule — you just look across the
              floor.
            </p>
            <Link
              href="/get-started"
              className="mt-2 inline-flex w-fit items-center bg-foreground px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
            >
              Get started
            </Link>
          </ScrollReveal>

          <ScrollReveal direction="right" stagger={0.08} className="flex flex-col gap-3">
            {forYouActions.map((a, i) => (
              <div
                key={a.label}
                style={{ '--stair-offset': `${Math.min(i * 8, 40)}%` }}
                className="stair-offset-item flex w-fit items-center gap-2 border border-border bg-card px-4 py-2.5 shadow-sm"
              >
                <a.icon className="size-4 text-accent" />
                <span className="text-sm sm:whitespace-nowrap">{a.label}</span>
              </div>
            ))}
          </ScrollReveal>
        </div>
      </div>

      <div className="border-t border-border bg-[#0a0a0a] text-white">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-10 px-6 py-16 lg:grid-cols-[1fr_1.3fr]">
          <ScrollReveal direction="left" className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <h3 className="text-2xl font-semibold tracking-tight">theboringoffice</h3>
              <span className="border border-accent px-2 py-0.5 font-mono text-xs uppercase tracking-wider text-accent">
                Your workflow
              </span>
            </div>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-white/60">
              Keep the hours and habits that already work. Tour the office in demo mode, walk in
              live, or attach the floor to an opencode server you already run.
            </p>
            <div className="mt-2 border border-white/10 bg-white/5 px-4 py-4 font-mono text-xs leading-relaxed">
              <p>theboringoffice --demo</p>
              <p className="mt-2">theboringoffice</p>
              <p className="mt-2">theboringoffice --server</p>
              <p className="pl-4">http://127.0.0.1:4096</p>
            </div>
            <Link
              href="/get-started"
              className="mt-2 inline-flex w-fit items-center bg-white px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-[#0a0a0a] transition-opacity hover:opacity-90"
            >
              See setup
            </Link>
          </ScrollReveal>

          <ScrollReveal
            direction="right"
            stagger={0.06}
            className="grid grid-cols-2 gap-4 sm:grid-cols-3"
          >
            {agents.map((a) => (
              <div key={a.name} className="flex flex-col justify-between border border-white/10 bg-white/3 p-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{a.name}</span>
                  <span className="size-1.5 rounded-full bg-chart-2" />
                </div>
                <p className="mt-8 font-mono text-[10px] text-white/40">{a.status}</p>
              </div>
            ))}
          </ScrollReveal>
        </div>
      </div>
    </section>
  )
}
