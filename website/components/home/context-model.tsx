import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'

export function ContextModel() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto grid max-w-7xl grid-cols-1 md:grid-cols-2">
        <ScrollReveal
          direction="left"
          className="flex flex-col gap-6 border-b border-border px-6 py-14 md:border-b-0 md:border-r md:px-10"
        >
          <h3 className="text-2xl font-semibold tracking-tight">A Team That Remembers</h3>

          <div className="border border-border bg-card">
            <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
              <span className="size-2.5 rounded-full bg-destructive/70" />
              <span className="size-2.5 rounded-full bg-chart-4/70" />
              <span className="size-2.5 rounded-full bg-chart-2/70" />
              <span className="ml-2 font-mono text-xs text-muted-foreground">office.py</span>
            </div>
            <pre className="overflow-x-auto px-4 py-4 font-mono text-xs leading-relaxed">
              <code>
                <span className="text-muted-foreground">1</span>{'  '}office
                <span className="text-accent">=</span>theboringoffice.
                <span className="text-chart-2">attach</span>(server
                <span className="text-accent">=</span>
                <span className="text-chart-3">&quot;127.0.0.1:4096&quot;</span>)
                {'\n'}
                <span className="text-muted-foreground">2</span>{'  '}roster
                <span className="text-accent">=</span>office.
                <span className="text-chart-2">employees</span>()
                {'\n'}
                <span className="text-muted-foreground">3</span>{'  '}memory
                <span className="text-accent">=</span>office.
                <span className="text-chart-2">recall</span>(
                <span className="text-chart-3">&quot;yesterday&apos;s batch&quot;</span>)
              </code>
            </pre>
          </div>

          <p className="max-w-sm text-pretty text-sm leading-relaxed text-muted-foreground">
            The office runs on agentmemory. Decisions, lessons, and yesterday&apos;s batch survive
            every ctrl+q — nobody on your team ever starts from a blank slate.
          </p>
          <Link
            href="/docs"
            className="inline-flex w-fit items-center border border-border px-4 py-2 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
          >
            Learn more
          </Link>
        </ScrollReveal>

        <ScrollReveal direction="right" className="flex flex-col gap-6 px-6 py-14 md:px-10">
          <h3 className="text-2xl font-semibold tracking-tight">Any Model, Any Backend</h3>

          <div className="relative h-56 overflow-hidden border border-border bg-card" aria-label="Supported model and coding agent logos">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,var(--accent),transparent_62%)] opacity-10" />
            <div className="absolute left-1/2 top-1/2 size-24 -translate-x-1/2 -translate-y-1/2 rounded-full border border-white/30 bg-background/80 shadow-[0_0_45px_var(--white)] shadow-white/20" />
            <div className="absolute left-1/2 top-1/2 flex size-12 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border border-white/50 bg-white/10 font-mono text-lg font-semibold text-accent">
              <img src="/imgs/logo.jpg" alt="Theboringoffice" className="size-full bg-blend-color " />
            </div>

            <div className="absolute left-[8%] top-[18%] flex items-center gap-2 border border-border bg-background/90 px-3 py-2 shadow-lg transition-transform hover:-translate-y-1">
              <img src="https://raw.githubusercontent.com/lobehub/lobe-icons/refs/heads/master/packages/static-png/dark/cursor.png" alt="Cursor" className="size-7 rounded-md border border-foreground/40 bg-background p-1" />
              <span className="font-mono text-xs text-foreground">Cursor</span>
            </div>
            <div className="absolute right-[8%] top-[14%] flex items-center gap-2 border border-border bg-background/90 px-3 py-2 shadow-lg transition-transform hover:-translate-y-1">
              <img src="https://raw.githubusercontent.com/lobehub/lobe-icons/refs/heads/master/packages/static-avatar/avatars/claudecode.webp" alt="Claude" className="size-7 rounded-md border border-foreground/40 bg-background p-1" />
              <span className="font-mono text-xs text-foreground">Claude</span>
            </div>
            <div className="absolute bottom-[16%] left-[14%] flex items-center gap-2 border border-border bg-background/90 px-3 py-2 shadow-lg transition-transform hover:translate-y-1">
              <img src="https://raw.githubusercontent.com/lobehub/lobe-icons/refs/heads/master/packages/static-png/dark/opencode.png" alt="OpenCode" className="size-7 rounded-md border border-foreground/40 bg-background p-1" />
              <span className="font-mono text-xs text-foreground">OpenCode</span>
            </div>
            <div className="absolute bottom-[13%] right-[12%] flex items-center gap-2 border border-border bg-background/90 px-3 py-2 shadow-lg transition-transform hover:translate-y-1">
            <img src="https://raw.githubusercontent.com/lobehub/lobe-icons/refs/heads/master/packages/static-png/dark/codex.png" alt="AgentMemory" className="size-7 rounded-md border border-foreground/40 bg-background p-1" />
              <span className="font-mono text-xs text-foreground">Codex</span>
            </div>

          </div>

          <p className="max-w-sm text-pretty text-sm leading-relaxed text-muted-foreground">
            Swap the boss&apos;s model with <span className="font-mono text-foreground">/model</span>,
            or attach to any running opencode server. No lock-in — run the office locally or
            point it at your own backend. Claude Code support is coming soon: same floor, more
            coworkers.
          </p>
          <Link
            href="/docs"
            className="inline-flex w-fit items-center border border-border px-4 py-2 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
          >
            Learn more
          </Link>
        </ScrollReveal>
      </div>
    </section>
  )
}
