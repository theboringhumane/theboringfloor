'use client'

import { useRef, useState } from 'react'
import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import { Beat, Cue, Sheet } from '@/components/paper'

const views = [
  { id: 'transcript', label: 'The workspace', title: 'Your projects. Their people. The conversation.', description: 'Floors on the far left. A live office in the middle. Transcript and tools on the right. Move between projects without losing your place.', alt: 'Three-pane terminal workspace with project floors on the far left, the office in the middle, and the conversation on the right.' },
  { id: 'board', label: 'Tickets & teams', title: 'Give the work a home, too.', description: 'Five stages, with priority, team, owner and checklists on every ticket. Start with the UI, Coding, Frontend and Backend teams, or write your own.', alt: 'Project ticket board with backlog, in progress, blocked, review, and done columns beside the office.' },
  { id: 'files', label: 'Project files', title: 'Read the code without leaving the floor.', description: 'Expand the project tree, preview source with line numbers, attach a file to the conversation. Folders load as you open them.', alt: 'Project file tree and a source preview in the right tools pane.' },
  { id: 'new-conversation', label: 'Pick a backend', title: 'A new conversation. Your choice of agent.', description: 'OpenCode, Claude Code or Codex, chosen per conversation. Give it a title and a team; its history stays on the same project floor.', alt: 'New conversation form with title, team, and OpenCode, Claude Code, or Codex backend selection.' },
  { id: 'plan', label: 'Plan & approve', title: 'The plan is visible before the work starts.', description: 'Substantial requests enter planning. When the boss presents a plan it opens beside your conversation. Read it, edit it, then press Ctrl+X twice to approve.', alt: 'Plan tool presentation opens a markdown review pane in the middle of the three-pane workspace.' },
]

const panes = [
  ['01', 'Floors', 'Projects, teams and saved conversations.'],
  ['02', 'Office', 'The crew, the browser, or the plan under review.'],
  ['03', 'Tools', 'Transcript, tickets, files, terminal and git.'],
]

export function WorkspaceTour() {
  const [active, setActive] = useState(0)
  const buttons = useRef<(HTMLButtonElement | null)[]>([])
  return (
    <Sheet id="workspaces" className="border-b border-rule">
      <div className="px-6 py-16 md:px-10 md:py-20 lg:px-14">
        <Beat index="1.1" label="Walk the floor">
          <h2 className="display-lg mt-6 max-w-2xl text-balance text-ink">All the context. One place to work.</h2>
          <div className="mt-6 flex flex-wrap items-end justify-between gap-6">
            <p className="max-w-xl text-base leading-relaxed text-ink-soft">
              Three panes, no tab hunting. The project you are in decides what
              every pane shows.
            </p>
            <Link href="/docs/workspaces" className="inline-flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft underline underline-offset-4 transition-colors hover:text-ink">
              Explore workspaces <ArrowUpRight className="size-3.5" aria-hidden />
            </Link>
          </div>

          <div role="tablist" aria-label="Explore the workspace" className="mt-10 flex gap-2 overflow-x-auto pb-3">
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
                className={`shrink-0 border px-4 py-2.5 font-mono text-[11px] uppercase tracking-[0.14em] transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue ${active === index ? 'border-ink bg-ink text-paper' : 'border-rule text-ink-faint hover:text-ink'}`}>{item.label}</button>
            ))}
          </div>

          {views.map((view, index) => <div key={view.id} role="tabpanel" id={`tour-panel-${view.id}`} aria-labelledby={`tour-tab-${view.id}`} hidden={active !== index} tabIndex={0} className="doc-brackets hairline mt-2 bg-paper-2">
            <div className="grid gap-3 border-b border-rule p-5 md:grid-cols-2 md:gap-10 md:p-7">
              <h3 className="display-md text-ink">{view.title}</h3>
              <p className="text-sm leading-relaxed text-ink-soft">{view.description}</p>
            </div>
            <a href={`/shots/workspaces/${view.id}.webp`} target="_blank" rel="noreferrer" className="block bg-panel" aria-label={`Open full-size image: ${view.label}`}>
              <img src={`/shots/workspaces/${view.id}.webp`} width={1548} height={1014} alt={view.alt} loading="lazy" className="shot-img block h-auto w-full" />
            </a>
            <div className="flex flex-wrap justify-between gap-2 border-t border-rule px-5 py-3"><span className="mono-label">Fig. 1.1 — actual app UI, illustrative data</span><Cue>click the image to inspect</Cue></div>
          </div>)}

          <dl className="mt-10 border-t border-rule">
            {panes.map(([index, name, text]) => (
              <div key={name} className="flex flex-col gap-1 border-b border-rule py-4 sm:flex-row sm:items-baseline sm:gap-8">
                <dt className="mono-label w-40 shrink-0 text-ink">{index} / {name}</dt>
                <dd className="text-sm leading-relaxed text-ink-soft">{text}</dd>
              </div>
            ))}
          </dl>
        </Beat>
      </div>
    </Sheet>
  )
}
