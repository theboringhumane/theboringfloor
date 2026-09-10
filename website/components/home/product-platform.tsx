import Link from 'next/link'
import type { CSSProperties } from 'react'
import { GitBranch, Mail, MessageCircle, MessageSquare } from 'lucide-react'
import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Cue, Sheet, Stamp } from '@/components/paper'

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
    <Sheet className="relative border-t border-rule">
      <div className="mx-auto max-w-7xl px-6 pt-20 md:px-10">
        <Beat
          index="3.6"
          label="Take a desk"
          title="Start with one desk. Grow into a full floor."
        >
          <p className="max-w-2xl text-pretty leading-relaxed text-ink-soft">
            Built for the team you have today, which may well be you and one agent.
          </p>
        </Beat>
      </div>

      <div className="mt-14 border-t border-rule">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-10 px-6 py-16 md:px-10 lg:grid-cols-2">
          <ScrollReveal direction="left" className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <h3 className="display-md text-ink">theboringfloor</h3>
              <Stamp tone="blue">For you</Stamp>
            </div>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-ink-soft">
              Open one terminal and a real agent session — OpenCode, Claude Code, or Codex —
              greets you. When work fans out, sub-agents take desks on the floor with the
              context you need to follow them.
            </p>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-ink-soft">
              Eight tools tabs: chat, terminal, agents, board, mail, activity, git, and
              files. Project floors keep their own navigator on the far left. No separate
              dashboard to configure, no status meeting to schedule. You just look across
              the floor.
            </p>
            <Link
              href="/get-started"
              className="mono-label mt-2 inline-flex w-fit items-center bg-ink px-5 py-3 text-paper transition-opacity hover:opacity-90"
            >
              Get started
            </Link>
          </ScrollReveal>

          <ScrollReveal direction="right" stagger={0.08} className="flex flex-col gap-3">
            {forYouActions.map((a, i) => (
              <div
                key={a.label}
                style={{ '--stair-offset': `${Math.min(i * 8, 40)}%` } as CSSProperties}
                className="stair-offset-item doc-brackets hairline flex w-fit items-center gap-2 bg-paper-2 px-4 py-2.5"
              >
                <a.icon className="size-4 text-blue" />
                <span className="text-sm text-ink sm:whitespace-nowrap">{a.label}</span>
              </div>
            ))}
          </ScrollReveal>
        </div>
      </div>

      <div className="paper-panel border-t border-rule">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-10 px-6 py-16 md:px-10 lg:grid-cols-[1fr_1.3fr]">
          <ScrollReveal direction="left" className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <h3 className="display-md text-ink">theboringfloor</h3>
              <Stamp tone="stamp">Your workflow</Stamp>
            </div>
            <p className="max-w-md text-pretty text-sm leading-relaxed text-ink-soft">
              Keep the hours and habits that already work. Tour the office in demo mode,
              walk in live, or attach the floor to an opencode server you already run.
            </p>
            <div className="doc-brackets hairline mt-2 px-4 py-4 font-mono text-xs leading-relaxed text-ink">
              <p>theboringfloor --demo</p>
              <p className="mt-2">theboringfloor</p>
              <p className="mt-2">theboringfloor --backend claudecode</p>
              <p className="mt-2">theboringfloor --backend codex</p>
              <p className="mt-2">theboringfloor --server</p>
              <p className="pl-4 text-ink-faint">http://127.0.0.1:4096</p>
            </div>
            <Link
              href="/get-started"
              className="mono-label mt-2 inline-flex w-fit items-center bg-ink px-5 py-3 text-paper transition-opacity hover:opacity-90"
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
              <div
                key={a.name}
                className="doc-brackets hairline flex flex-col justify-between p-4"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-ink">{a.name}</span>
                  <span className="size-1.5 rounded-full bg-blue" />
                </div>
                <p className="mt-8 font-mono text-[10px] text-ink-faint">{a.status}</p>
              </div>
            ))}
          </ScrollReveal>
        </div>
      </div>

      <div className="mx-auto max-w-7xl px-6 pb-12 md:px-10">
        <Cue>one binary, one terminal, as many desks as the work needs</Cue>
      </div>
    </Sheet>
  )
}
