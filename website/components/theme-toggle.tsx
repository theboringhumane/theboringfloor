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

const focusRing =
  'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue focus-visible:ring-offset-2 focus-visible:ring-offset-paper'

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
        className={cn(
          'inline-flex items-center justify-center border border-rule p-1.5 text-ink-soft transition-colors hover:border-ink-faint hover:text-ink',
          focusRing,
        )}
      >
        {mounted ? (
          <ResolvedIcon className="size-3.5" aria-hidden />
        ) : (
          <span className="size-3.5" aria-hidden />
        )}
      </button>
      {open && (
        <div
          role="menu"
          aria-label="Theme"
          className="doc-brackets absolute right-0 top-full z-50 mt-2 w-36 border border-rule bg-paper-2 p-1"
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
              className={cn(
                'flex w-full items-center gap-2 px-2.5 py-2 font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-soft transition-colors hover:text-ink',
                focusRing,
              )}
            >
              <Icon className="size-3.5" aria-hidden />
              <span className="flex-1 text-left">{label}</span>
              {theme === value && <Check className="size-3.5 text-stamp" aria-hidden />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
