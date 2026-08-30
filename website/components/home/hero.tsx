'use client'

import Link from 'next/link'
import { useState } from 'react'
import { ShaderBackground } from '@/components/ui/light-blue-plasma-shader-w-grain-interactive'

const INSTALL =
  'curl -fsSL https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh'

export function Hero() {
  const [copied, setCopied] = useState(false)

  async function copyInstall() {
    try {
      await navigator.clipboard.writeText(INSTALL)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1600)
    } catch {
      /* ignore */
    }
  }

  return (
    <section
      id="hero"
      className="relative overflow-hidden border-b border-border bg-(--band-bg) text-(--band-fg)"
    >
      <ShaderBackground className="shot-img pointer-events-none absolute inset-y-0 right-0 w-[55%] opacity-40 max-md:opacity-25" />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-(--band-bg) via-(--band-bg)/85 to-transparent" />

      <div className="relative px-6 pb-20 pt-16 md:px-10 md:pb-24 md:pt-20 lg:px-14">
        <p className="flex items-center gap-3 font-mono text-[11px] uppercase tracking-[0.22em] text-(--band-fg)/45">
          <span className="h-px w-6 bg-(--band-fg)/35" aria-hidden />
          The virtual office
        </p>

        <h1 className="mt-10 max-w-[14ch] font-sans text-[clamp(3.25rem,11vw,8.5rem)] font-medium leading-[0.9] tracking-[-0.05em]">
          Work with agents
          <br />
          that feel like{' '}
          <span className="text-(--band-accent)">coworkers.</span>
        </h1>

        <p className="mt-10 max-w-lg text-pretty font-sans text-base leading-relaxed text-(--band-fg)/50 md:text-lg">
          A terminal app where your opencode or Claude Code boss and sub-agents clock in as
          coworkers on a living floor. See the work move, talk to the boss like a person, and
          come back tomorrow to an office that remembers.
        </p>

        <div className="mt-12 flex max-w-xl items-stretch border border-(--band-fg)/15 bg-(--band-well) font-mono text-sm">
          <code className="min-w-0 flex-1 overflow-x-auto px-4 py-3.5 text-(--band-fg)/75">
            <span className="text-(--band-fg)/35">$ </span>
            curl -fsSL …/install.sh | sh
          </code>
          <button
            type="button"
            onClick={copyInstall}
            className="shrink-0 border-l border-(--band-fg)/15 px-4 text-[11px] uppercase tracking-wider text-(--band-fg)/55 transition-colors hover:bg-(--band-fg)/5 hover:text-(--band-fg)"
          >
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>

        <div className="mt-6 flex flex-wrap items-center gap-3">
          <Link
            href="/get-started"
            className="border border-(--band-fg)/80 bg-(--band-fg) px-6 py-3 font-mono text-xs uppercase tracking-wider text-(--band-on-fg) transition-opacity hover:opacity-90"
          >
            Open the office
          </Link>
          <Link
            href="/get-started"
            className="border border-(--band-fg)/25 px-6 py-3 font-mono text-xs uppercase tracking-wider text-(--band-fg)/80 transition-colors hover:bg-(--band-fg)/5"
          >
            Tour demo mode
          </Link>
        </div>
      </div>
    </section>
  )
}
