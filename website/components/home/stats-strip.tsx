import { FolderGit2, LayoutGrid, ShieldCheck, Terminal } from 'lucide-react'
import { cn } from '@/lib/utils'

const stats = [
  { value: 'CLI', label: 'Native binary', icon: Terminal },
  { value: 'Demo', label: 'Tour first', icon: LayoutGrid },
  { value: 'Threads', label: 'Work + diffs', icon: FolderGit2 },
  { value: 'Queue', label: 'Your permissions', icon: ShieldCheck },
]

export function StatsStrip() {
  return (
    <section className="border-b border-border" aria-label="Product pillars">
      <div className="grid grid-cols-2 md:grid-cols-4">
        {stats.map((s, i) => (
          <div
            key={s.label}
            className={cn(
              'flex min-h-[9.5rem] flex-col justify-between p-6 md:p-8',
              i < 2 && 'border-b border-border md:border-b-0',
              i !== 3 && 'md:border-r md:border-border',
              i % 2 === 0 && 'border-r border-border',
            )}
          >
            <p className="font-sans text-3xl font-medium tracking-tight md:text-4xl">{s.value}</p>
            <div className="mt-8 flex items-center gap-2 text-muted-foreground">
              <s.icon className="size-3.5 shrink-0" aria-hidden />
              <span className="font-mono text-[10px] uppercase tracking-[0.18em]">{s.label}</span>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
