'use client'

import { useRef, useState } from 'react'
import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import { SectionTag } from '@/components/section-tag'

const views = [
  { id: 'transcript', label: 'The workspace', title: 'Your projects. Their people. The conversation.', description: 'Floors on the far left. A live office in the middle. Transcript and tools on the right. Move between projects without losing your place.', alt: 'Three-pane terminal workspace with project floors on the far left, the office in the middle, and the conversation on the right.' },
  { id: 'board', label: 'Tickets & teams', title: 'Give the work a home, too.', description: 'Track tickets through five stages with priority, team, owner, and checklists. Start with UI, Coding, Frontend, and Backend teams, or add your own.', alt: 'Project ticket board with backlog, in progress, blocked, review, and done columns beside the office.' },
  { id: 'files', label: 'Project files', title: 'Read the code without leaving the floor.', description: 'Expand the project tree, preview source with line numbers, and attach a file to the conversation. Folders load as you explore.', alt: 'Project file tree and a source preview in the right tools pane.' },
  { id: 'new-conversation', label: 'Pick a backend', title: 'A new conversation. Your choice of agent.', description: 'Choose OpenCode, Claude Code, or Codex for each conversation. Give it a title and team; return to its history from the same project floor.', alt: 'New conversation form with title, team, and OpenCode, Claude Code, or Codex backend selection.' },
  { id: 'plan', label: 'Plan & approve', title: 'Make the plan visible before work begins.', description: 'Substantial requests enter planning. When the boss presents a plan, it opens beside your conversation. Review, edit, then press Ctrl+X twice to approve.', alt: 'Plan tool presentation opens a markdown review pane in the middle of the three-pane workspace.' },
]

export function WorkspaceTour() {
  const [active, setActive] = useState(0)
  const buttons = useRef<(HTMLButtonElement | null)[]>([])
  return (
    <section id="workspaces" className="border-b border-border px-6 py-16 md:px-10 lg:px-14">
      <div className="flex flex-wrap items-end justify-between gap-6">
        <div>
          <SectionTag>A floor for every project</SectionTag>
          <h2 className="mt-5 max-w-2xl text-balance text-3xl font-medium tracking-tight md:text-5xl">All the context. One place to work.</h2>
        </div>
        <Link href="/docs/workspaces" className="inline-flex items-center gap-2 font-mono text-xs underline underline-offset-4">Explore workspaces <ArrowUpRight className="size-4" aria-hidden /></Link>
      </div>
      <div role="tablist" aria-label="Explore the workspace" className="mt-9 flex gap-2 overflow-x-auto pb-3">
        {views.map((item, index) => (
          <button key={item.id} ref={(el) => { buttons.current[index] = el }} type="button" role="tab" id={`tour-tab-${item.id}`} aria-selected={active === index} aria-controls={`tour-panel-${item.id}`} tabIndex={active === index ? 0 : -1}
            onClick={() => setActive(index)}
            onKeyDown={(event) => {
              let next = index
              if (event.key === 'ArrowRight') next = (index + 1) % views.length
              else if (event.key === 'ArrowLeft') next = (index + views.length - 1) % views.length
              else if (event.key === 'Home') next = 0
              else if (event.key === 'End') next = views.length - 1
              else return
              event.preventDefault(); setActive(next); buttons.current[next]?.focus()
            }}
            className={`shrink-0 border px-4 py-3 font-mono text-xs transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent ${active === index ? 'border-foreground bg-foreground text-background' : 'border-border text-muted-foreground hover:text-foreground'}`}>{item.label}</button>
        ))}
      </div>
      {views.map((view, index) => <div key={view.id} role="tabpanel" id={`tour-panel-${view.id}`} aria-labelledby={`tour-tab-${view.id}`} hidden={active !== index} tabIndex={0} className="mt-2 border border-border">
        <div className="grid gap-3 border-b border-border p-5 md:grid-cols-2 md:gap-10 md:p-7">
          <h3 className="text-xl font-medium tracking-tight">{view.title}</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">{view.description}</p>
        </div>
        <a href={`/shots/workspaces/${view.id}.webp`} target="_blank" rel="noreferrer" className="block bg-[#161619]" aria-label={`Open full-size image: ${view.label}`}>
          <img src={`/shots/workspaces/${view.id}.webp`} width={1548} height={1014} alt={view.alt} loading="lazy" className="block h-auto w-full" />
        </a>
        <div className="flex flex-wrap justify-between gap-2 border-t border-border px-5 py-3 font-mono text-[10px] uppercase tracking-wider text-muted-foreground"><span>Actual app UI · illustrative project data</span><span>Click image to inspect</span></div>
      </div>)}
      <div className="mt-6 grid gap-px border border-border bg-border sm:grid-cols-3">
        {[['01 / Floors', 'Projects, teams, and saved conversations.'], ['02 / Office', 'The crew, browser, or plan under review.'], ['03 / Tools', 'Transcript, tickets, files, terminal, and git.']].map(([label, text]) => <div key={label} className="bg-background p-5"><p className="font-mono text-xs text-accent">{label}</p><p className="mt-2 text-sm text-muted-foreground">{text}</p></div>)}
      </div>
    </section>
  )
}
