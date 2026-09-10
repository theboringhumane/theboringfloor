'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { FolderGit2 as Github, Menu, X } from 'lucide-react'

import { cn } from '@/lib/utils'
import { DISCORD_INVITE, GITHUB_REPO } from '@/lib/site'
import { ThemeToggle } from '@/components/theme-toggle'

/** The masthead nav. Order and hrefs are the document's table of contents. */
const NAV = [
  { href: '/#workspaces', label: 'Floor Plan' },
  { href: '/blog', label: 'Blog' },
  { href: '/vision', label: 'Vision' },
  { href: '/sounds', label: 'Sounds' },
  { href: '/changelog', label: 'Changelog' },
  { href: '/docs', label: 'Docs' },
]

const focusRing =
  'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue focus-visible:ring-offset-2 focus-visible:ring-offset-paper'

function DiscordMark({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" />
    </svg>
  )
}

export function SiteHeader({
  seamless = false,
  framed = false,
}: {
  seamless?: boolean
  framed?: boolean
}) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [pastHero, setPastHero] = useState(false)

  useEffect(() => {
    if (!seamless || framed) return
    const hero = document.getElementById('hero')
    if (!hero) return
    const io = new IntersectionObserver(
      ([e]) => setPastHero(!e.isIntersecting),
      { threshold: 0 },
    )
    io.observe(hero)
    return () => io.disconnect()
  }, [seamless, framed])

  const solid = framed || !seamless || pastHero || mobileOpen

  return (
    <>
      <header
        className={cn(
          'z-50 w-full border-t border-rule transition-[background-color,border-color,backdrop-filter] duration-300',
          framed ? 'sticky top-0' : 'fixed top-0',
          solid
            ? 'border-b border-rule bg-paper/90 backdrop-blur supports-[backdrop-filter]:bg-paper/75'
            : 'border-b border-transparent bg-transparent',
        )}
      >
        <div
          className={cn(
            'flex h-14 items-center justify-between gap-6 px-6 md:h-16 md:px-10 lg:px-14',
            !framed && 'mx-auto max-w-7xl',
          )}
        >
          <Link
            href="/"
            className={cn(
              'group inline-flex items-center gap-2.5 py-2 text-ink',
              focusRing,
            )}
          >
            <span
              className="size-1.5 shrink-0 rounded-full bg-stamp"
              aria-hidden
            />
            <span className="font-mono text-[0.8125rem] font-medium uppercase leading-none tracking-[0.16em]">
              theboringfloor
            </span>
          </Link>

          <nav aria-label="Main" className="hidden lg:block">
            <ul className="flex items-center gap-7 font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-soft">
              {NAV.map((item) => (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className={cn(
                      'block py-2 transition-colors hover:text-ink',
                      focusRing,
                    )}
                  >
                    {item.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>

          <div className="hidden items-center gap-5 lg:flex">
            <ThemeToggle />
            <a
              href={GITHUB_REPO}
              target="_blank"
              rel="noreferrer"
              className={cn(
                'inline-flex items-center gap-1.5 py-2 font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-soft transition-colors hover:text-ink',
                focusRing,
              )}
            >
              <Github className="size-3.5" aria-hidden />
              GitHub
            </a>
            <a
              href={DISCORD_INVITE}
              target="_blank"
              rel="noreferrer"
              className={cn(
                'inline-flex items-center gap-1.5 py-2 font-mono text-[0.6875rem] lowercase tracking-[0.12em] text-ink transition-colors hover:text-stamp',
                focusRing,
              )}
            >
              <span aria-hidden className="text-ink-faint">
                [
              </span>
              <DiscordMark className="size-3.5" />
              join discord
              <span aria-hidden className="text-ink-faint">
                ]
              </span>
            </a>
          </div>

          <button
            type="button"
            className={cn('text-ink lg:hidden', focusRing)}
            onClick={() => setMobileOpen((v) => !v)}
            aria-label="Toggle menu"
            aria-expanded={mobileOpen}
          >
            {mobileOpen ? <X className="size-6" /> : <Menu className="size-6" />}
          </button>
        </div>

        <div
          className={cn(
            'overflow-hidden border-t border-rule lg:hidden',
            mobileOpen ? 'max-h-[36rem] bg-paper' : 'max-h-0 border-t-0',
          )}
        >
          <nav
            aria-label="Mobile"
            className="flex flex-col gap-1 px-6 py-4 font-mono text-xs uppercase tracking-[0.14em] text-ink-soft"
          >
            <Link
              href="/#solutions"
              className={cn('py-2.5 transition-colors hover:text-ink', focusRing)}
              onClick={() => setMobileOpen(false)}
            >
              Solutions
            </Link>
            {NAV.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={cn('py-2.5 transition-colors hover:text-ink', focusRing)}
                onClick={() => setMobileOpen(false)}
              >
                {item.label}
              </Link>
            ))}
            <a
              href={GITHUB_REPO}
              target="_blank"
              rel="noreferrer"
              className={cn(
                'inline-flex items-center gap-2 py-2.5 transition-colors hover:text-ink',
                focusRing,
              )}
              onClick={() => setMobileOpen(false)}
            >
              <Github className="size-4" aria-hidden />
              GitHub
            </a>
            <a
              href={DISCORD_INVITE}
              target="_blank"
              rel="noreferrer"
              className={cn(
                'mt-2 inline-flex items-center justify-center gap-2 border border-rule px-4 py-2.5 lowercase tracking-[0.12em] text-ink transition-colors hover:border-stamp hover:text-stamp',
                focusRing,
              )}
              onClick={() => setMobileOpen(false)}
            >
              <span aria-hidden className="text-ink-faint">
                [
              </span>
              <DiscordMark className="size-4" />
              join discord
              <span aria-hidden className="text-ink-faint">
                ]
              </span>
            </a>
            <div className="mt-2 flex items-center justify-between border-t border-rule pt-4">
              <span className="text-ink-faint">Theme</span>
              <ThemeToggle />
            </div>
          </nav>
        </div>
      </header>
      {!seamless && !framed && <div className="h-16" aria-hidden />}
    </>
  )
}
