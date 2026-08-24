'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import { ChevronDown, FolderGit2 as Github, Menu, X } from 'lucide-react'

const GITHUB_REPO = 'https://github.com/theboringhumane/theboringoffice'
import { cn } from '@/lib/utils'

const products = [
  { name: 'The Office', description: 'Give any opencode agent a desk and a name' },
  { name: 'Work Threads', description: 'Real diffs and thinking, per task, per agent' },
  { name: 'Permission Queue', description: 'One inbox for every yes, no, and always' },
  { name: 'Concierge', description: 'Someone always answers, even when the boss is out' },
  { name: 'CLI', description: 'Open the office from your terminal' },
]

const solutions = [
  { name: 'Engineering Teams', description: 'Sub-agents that fix, review, and ship' },
  { name: 'Ops & Support', description: 'A floor that never leaves tickets untouched' },
  { name: 'Solo Builders', description: 'A whole office, run by one person' },
  { name: 'Agencies', description: 'Staff every client with tireless desks' },
  { name: 'Late Nights', description: 'The lights stay on so you do not have to' },
]

export function SiteHeader({ seamless = false }: { seamless?: boolean }) {
  const [productsOpen, setProductsOpen] = useState(false)
  const [solutionsOpen, setSolutionsOpen] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [pastHero, setPastHero] = useState(false)

  useEffect(() => {
    if (!seamless) return
    const hero = document.getElementById('hero')
    if (!hero) return
    const io = new IntersectionObserver(
      ([e]) => setPastHero(!e.isIntersecting),
      { threshold: 0 },
    )
    io.observe(hero)
    return () => io.disconnect()
  }, [seamless])

  const solid = !seamless || pastHero || mobileOpen

  return (
    <>
    <header
      className={cn(
        'fixed top-0 z-50 w-full transition-[background-color,border-color,backdrop-filter] duration-300',
        solid
          ? 'border-b border-border bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/70'
          : 'border-b border-transparent bg-transparent',
      )}
    >
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
        <div className="flex items-center gap-2 lg:gap-4">
          <img src="/imgs/logo.jpg" alt="theboringoffice" className="h-8 w-8" />
          <Link href="/" className="flex items-center gap-2">
            <span className="font-mono text-sm font-bold uppercase tracking-wider text-foreground">
              The Boring Office
            </span>
          </Link>
        </div>

        <nav className="hidden items-center gap-8 font-mono text-xs uppercase tracking-wider text-muted-foreground lg:flex">
          <div
            className="relative"
            onMouseEnter={() => setProductsOpen(true)}
            onMouseLeave={() => setProductsOpen(false)}
          >
            <button className="flex items-center gap-1 py-2 transition-colors hover:text-foreground">
              Products
              <ChevronDown className="size-3" />
            </button>
            {productsOpen && (
              <div className="absolute left-1/2 top-full w-80 -translate-x-1/2 pt-2">
                <div className="rounded-md border border-border bg-popover p-2 shadow-xl">
                  {products.map((item) => (
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
          </div>

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
          <Link href="/docs" className="py-2 transition-colors hover:text-foreground">
            Docs
          </Link>
        </nav>

        <div className="hidden items-center gap-3 lg:flex">
          <a
            href={GITHUB_REPO}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 py-2 font-mono text-xs uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground"
          >
            <Github className="size-3.5" aria-hidden />
            GitHub
          </a>
          <Link
            href="/get-started"
            className="rounded-sm bg-foreground px-4 py-2 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
          >
            Open the Office
          </Link>
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
          mobileOpen ? 'max-h-96' : 'max-h-0 border-t-0',
          !solid && 'border-transparent',
        )}
      >
        <nav className="flex flex-col gap-1 px-6 py-4 font-mono text-sm uppercase tracking-wider text-muted-foreground">
          <Link href="/#products" className="py-2" onClick={() => setMobileOpen(false)}>
            Products
          </Link>
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
          <Link
            href="/get-started"
            className="mt-2 rounded-sm bg-foreground px-4 py-2.5 text-center text-background"
            onClick={() => setMobileOpen(false)}
          >
            Open the Office
          </Link>
        </nav>
      </div>
    </header>
    {!seamless && <div className="h-16" aria-hidden />}
    </>
  )
}
