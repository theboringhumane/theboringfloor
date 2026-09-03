'use client'

import { createContext, useCallback, useContext, useLayoutEffect, useMemo, useState } from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'theboringfloor-theme'
const LEGACY_STORAGE_KEY = 'theboringoffice-theme'
const MEDIA = '(prefers-color-scheme: dark)'

function isTheme(value: string | null): value is Theme {
  return value === 'light' || value === 'dark' || value === 'system'
}

function readStoredTheme(): Theme {
  if (typeof window === 'undefined') return 'light'
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY) ?? window.localStorage.getItem(LEGACY_STORAGE_KEY)
    return isTheme(stored) ? stored : 'light'
  } catch {
    return 'light'
  }
}

function resolveTheme(theme: Theme): ResolvedTheme {
  if (theme !== 'system') return theme
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia(MEDIA).matches ? 'dark' : 'light'
}

// Mirror of the inline anti-FOUC script in app/layout.tsx — the two must agree.
function applyTheme(resolved: ResolvedTheme) {
  const root = document.documentElement
  root.classList.remove('light', 'dark')
  root.classList.add(resolved)
  root.style.colorScheme = resolved
}

type ThemeContextValue = {
  theme: Theme
  resolvedTheme: ResolvedTheme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'light',
  resolvedTheme: 'light',
  setTheme: () => {},
})

export function useTheme() {
  return useContext(ThemeContext)
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  // Lazy initializers read the same localStorage key the inline head script
  // applied before paint, so the first client render already matches the DOM.
  const [theme, setThemeState] = useState<Theme>(readStoredTheme)
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    resolveTheme(readStoredTheme()),
  )

  // Re-apply on mount. The inline script already did this pre-paint, so this
  // is a no-op in production; it matters in development, where React Strict
  // Mode's remount resets <html> attributes it doesn't own.
  useLayoutEffect(() => {
    applyTheme(resolveTheme(readStoredTheme()))
  }, [])

  // Follow OS preference changes only while the user picked "system".
  useLayoutEffect(() => {
    if (theme !== 'system') return
    const mq = window.matchMedia(MEDIA)
    const onChange = () => {
      const resolved = resolveTheme('system')
      applyTheme(resolved)
      setResolvedTheme(resolved)
    }
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [theme])

  const setTheme = useCallback((next: Theme) => {
    const resolved = resolveTheme(next)
    try {
      window.localStorage.setItem(STORAGE_KEY, next)
    } catch {
      /* storage unavailable — theme still applies for this session */
    }
    applyTheme(resolved)
    setThemeState(next)
    setResolvedTheme(resolved)
  }, [])

  const value = useMemo(
    () => ({ theme, resolvedTheme, setTheme }),
    [theme, resolvedTheme, setTheme],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}
