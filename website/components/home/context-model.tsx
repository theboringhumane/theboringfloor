import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'

export function ContextModel() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto grid max-w-7xl md:grid-cols-2">
        <ScrollReveal className="min-w-0 border-b border-border px-6 py-14 md:border-b-0 md:border-r md:px-10">
          <p className="font-mono text-xs uppercase tracking-wider text-accent">Project memory</p>
          <h3 className="mt-5 text-3xl font-medium tracking-tight">Pick up where the team left off.</h3>
          <p className="mt-5 text-sm leading-relaxed text-muted-foreground">Teams and tickets belong to the project. Conversations keep their own backend and history. Start a new task without clearing the work that came before it.</p>
          <div className="mt-7 border border-border bg-card p-5 font-mono text-xs leading-loose"><p>floor.json <span className="text-muted-foreground">→ teams + tickets</span></p><p>conversations/ <span className="text-muted-foreground">→ saved sessions</span></p><p>Ctrl+R <span className="text-muted-foreground">→ search loaded transcript</span></p></div>
          <p className="mt-5 text-sm leading-relaxed text-muted-foreground">Local conversation archives retain up to 10,000 messages. A small 200-message snapshot keeps the next startup quick; agentmemory adds a separate decision and task ledger when connected.</p>
          <Link href="/docs/workspaces#history" className="mt-6 inline-block text-sm underline underline-offset-4">How history works</Link>
        </ScrollReveal>
        <ScrollReveal className="min-w-0 px-6 py-14 md:px-10">
          <p className="font-mono text-xs uppercase tracking-wider text-accent">Your choice of backend</p>
          <h3 className="mt-5 text-3xl font-medium tracking-tight">Different agents. The same workspace.</h3>
          <div className="mt-7 divide-y divide-border border border-border">{[['OpenCode', 'Start a server or attach to one you already run.'], ['Claude Code', 'Use your installed Claude CLI and login.'], ['Codex', 'Run with your Codex CLI login and model settings.']].map(([name, text]) => <div key={name} className="p-5"><p className="font-mono text-sm">{name}</p><p className="mt-2 text-sm text-muted-foreground">{text}</p></div>)}</div>
          <p className="mt-5 text-sm leading-relaxed text-muted-foreground">Press Ctrl+N to choose a backend, team, and title for a new conversation. Each resumes on the backend that created it.</p>
          <Link href="/docs/backends" className="mt-6 inline-block text-sm underline underline-offset-4">Compare backend support</Link>
        </ScrollReveal>
      </div>
    </section>
  )
}
