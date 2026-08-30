'use client'

import { useEffect, useRef, useState } from 'react'
import { Check, Monitor, Moon, Sun } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useTheme, type Theme } from '@/components/theme-provider'

const OPTIONS: { value: Theme; label: string; Icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', Icon: Sun },
  { value: 'dark', label: 'Dark', Icon: Moon },
  { value: 'system', label: 'System', Icon: Monitor },
]

export function ThemeToggle({ className }: { className?: string }) {
  const { theme, resolvedTheme, setTheme } = useTheme()
  const [open, setOpen] = useState(false)
  // The icon depends on client-only state (localStorage / matchMedia); render a
  // same-size placeholder until mounted so hydration output matches the server.
  const [mounted, setMounted] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => setMounted(true), [])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const ResolvedIcon = resolvedTheme === 'dark' ? Moon : Sun

  return (
    <div ref={rootRef} className={cn('relative', className)}>
      <button
        type="button"
        aria-label="Toggle theme"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center justify-center p-2 text-muted-foreground transition-colors hover:text-foreground"
      >
        {mounted ? (
          <ResolvedIcon className="size-4" aria-hidden />
        ) : (
          <span className="size-4" aria-hidden />
        )}
      </button>
      {open && (
        <div
          role="menu"
          aria-label="Theme"
          className="absolute right-0 top-full z-50 mt-2 w-36 rounded-md border border-border bg-popover p-1 shadow-xl"
        >
          {OPTIONS.map(({ value, label, Icon }) => (
            <button
              key={value}
              type="button"
              role="menuitemradio"
              aria-checked={theme === value}
              onClick={() => {
                setTheme(value)
                setOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-sm px-2.5 py-2 font-mono text-xs uppercase tracking-wider text-muted-foreground transition-colors hover:bg-accent/10 hover:text-foreground"
            >
              <Icon className="size-3.5" aria-hidden />
              <span className="flex-1 text-left">{label}</span>
              {theme === value && <Check className="size-3.5 text-foreground" aria-hidden />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
