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
        'mono-label inline-flex items-center gap-2 border border-rule px-2.5 py-1.5',
        className,
      )}
    >
      <span className="size-1.5 bg-stamp" aria-hidden="true" />
      {children}
    </div>
  )
}
