import { cn } from '@/lib/utils'

export function SectionTag({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs uppercase tracking-wider text-muted-foreground',
        className,
      )}
    >
      <span className="size-2 bg-accent" aria-hidden="true" />
      {children}
    </div>
  )
}
