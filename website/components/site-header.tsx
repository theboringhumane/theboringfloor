'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { FolderGit2 as Github, Menu, X } from 'lucide-react'

import { cn } from '@/lib/utils'
import { DISCORD_INVITE, GITHUB_REPO } from '@/lib/site'
import { ThemeToggle } from '@/components/theme-toggle'

const solutions = [
  { name: 'Engineering Teams', description: 'Sub-agents that fix, review, and ship' },
  { name: 'Ops & Support', description: 'A floor that never leaves tickets untouched' },
  { name: 'Solo Builders', description: 'A whole office, run by one person' },
  { name: 'Agencies', description: 'Staff every client with tireless desks' },
  { name: 'Late Nights', description: 'The lights stay on so you do not have to' },
]

export function SiteHeader({
  seamless = false,
  framed = false,
}: {
  seamless?: boolean
  framed?: boolean
}) {
  const [solutionsOpen, setSolutionsOpen] = useState(false)
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
        'z-50 w-full transition-[background-color,border-color,backdrop-filter] duration-300',
        framed
          ? 'sticky top-0 border-b border-border bg-canvas/90 backdrop-blur supports-[backdrop-filter]:bg-canvas/80'
          : 'fixed top-0',
        !framed &&
          (solid
            ? 'border-b border-border bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/70'
            : 'border-b border-transparent bg-transparent'),
      )}
    >
      <div
        className={cn(
          'flex h-14 items-center justify-between px-6 md:h-16 md:px-10 lg:px-14',
          !framed && 'mx-auto max-w-7xl',
        )}
      >
        <div className="flex items-center gap-2 lg:gap-4">
          <img src="/imgs/logo.jpg" alt="theboringoffice" className="h-8 w-8" />
          <Link href="/" className="flex items-center gap-2">
            <span className="font-mono text-sm font-bold uppercase tracking-wider text-foreground">
              The Boring Office
            </span>
          </Link>
        </div>

        <nav className="hidden items-center gap-8 font-mono text-xs uppercase tracking-wider text-muted-foreground lg:flex">
          {/*<div
            className="relative"
            onMouseEnter={() => setSolutionsOpen(true)}
            onMouseLeave={() => setSolutionsOpen(false)}
          >
            <button className="flex items-center gap-1 py-2 transition-colors hover:text-foreground">
              Solutions
              <ChevronDown className="size-3" />
            </button>
            {solutionsOpen && (
              <div className="absolute left-1/2 top-full w-80 -translate-x-1/2 pt-2">
                <div className="rounded-md border border-border bg-popover p-2 shadow-xl">
                  {solutions.map((item) => (
                    <div
                      key={item.name}
                      className="rounded-sm px-3 py-2.5 hover:bg-accent/10"
                    >
                      <p className="font-sans text-sm normal-case tracking-normal text-foreground">
                        {item.name}
                      </p>
                      <p className="mt-0.5 font-sans text-xs normal-case tracking-normal text-muted-foreground">
                        {item.description}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>*/}

          <Link href="/#toolkits" className="py-2 transition-colors hover:text-foreground">
            Floor Plan
          </Link>
          <Link href="/blog" className="py-2 transition-colors hover:text-foreground">
            Blog
          </Link>
          <Link href="/vision" className="py-2 transition-colors hover:text-foreground">
            Vision
          </Link>
          <Link href="/sounds" className="py-2 transition-colors hover:text-foreground">
            Sounds
          </Link>
          <Link href="/changelog" className="py-2 transition-colors hover:text-foreground">
            Changelog
          </Link>
          <Link href="/docs" className="py-2 transition-colors hover:text-foreground">
            Docs
          </Link>
        </nav>

        <div className="hidden items-center gap-4 lg:flex">
          <ThemeToggle />
          <a
            href={GITHUB_REPO}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 py-2 font-mono text-xs uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
          >
            <Github className="size-3.5" aria-hidden />
            GitHub
          </a>
          <a
            href={DISCORD_INVITE}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 rounded-sm bg-foreground px-4 py-2 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
          >
            <svg className="size-3.5" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" />
            </svg>
            Join Discord
          </a>
        </div>

        <button
          className="text-foreground lg:hidden"
          onClick={() => setMobileOpen((v) => !v)}
          aria-label="Toggle menu"
        >
          {mobileOpen ? <X className="size-6" /> : <Menu className="size-6" />}
        </button>
      </div>

      <div
        className={cn(
          'overflow-hidden border-t border-border lg:hidden',
          mobileOpen ? 'max-h-[28rem]' : 'max-h-0 border-t-0',
          !solid && 'border-transparent',
        )}
      >
        <nav className="flex flex-col gap-1 px-6 py-4 font-mono text-sm uppercase tracking-wider text-muted-foreground">
          <Link href="/#solutions" className="py-2" onClick={() => setMobileOpen(false)}>
            Solutions
          </Link>
          <Link href="/#toolkits" className="py-2" onClick={() => setMobileOpen(false)}>
            Floor Plan
          </Link>
          <Link href="/blog" className="py-2" onClick={() => setMobileOpen(false)}>
            Blog
          </Link>
          <Link href="/vision" className="py-2" onClick={() => setMobileOpen(false)}>
            Vision
          </Link>
          <Link href="/sounds" className="py-2" onClick={() => setMobileOpen(false)}>
            Sounds
          </Link>
          <Link href="/changelog" className="py-2" onClick={() => setMobileOpen(false)}>
            Changelog
          </Link>
          <Link href="/docs" className="py-2" onClick={() => setMobileOpen(false)}>
            Docs
          </Link>
          <a
            href={GITHUB_REPO}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-2 py-2"
            onClick={() => setMobileOpen(false)}
          >
            <Github className="size-4" aria-hidden />
            GitHub
          </a>
          <a
            href={DISCORD_INVITE}
            target="_blank"
            rel="noreferrer"
            className="mt-2 inline-flex items-center justify-center gap-2 rounded-sm bg-foreground px-4 py-2.5 text-background transition-opacity hover:opacity-90"
            onClick={() => setMobileOpen(false)}
          >
            <svg className="size-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" />
            </svg>
            Join Discord
          </a>
          <div className="mt-2 flex items-center justify-between border-t border-border pt-4">
            <span className="font-mono text-sm uppercase tracking-wider text-muted-foreground">
              Theme
            </span>
            <ThemeToggle />
          </div>
        </nav>
      </div>
    </header>
{!seamless && !framed && <div className="h-16" aria-hidden />}
    </>
  )
}
