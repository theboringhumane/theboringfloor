'use client'

import Link from 'next/link'
import { ShaderBackground } from '@/components/ui/light-blue-plasma-shader-w-grain-interactive'

export function Hero() {

  return (
    <section
      id="hero"
      className="relative min-h-[70svh] overflow-hidden bg-[#0b0b0b] text-white"
    >
      <ShaderBackground className="absolute inset-0 opacity-95" />
      <div className="absolute inset-0 bg-[#0b0b0b]/35" />
      <div className="relative mx-auto flex min-h-[70svh] max-w-7xl flex-col items-center justify-center px-6 pb-16 pt-24 text-center">
        <h1 className="max-w-3xl font-sans text-3xl font-medium leading-[1.05] tracking-[-0.04em] sm:text-4xl md:text-5xl lg:text-5xl">
          Work with agents
          <br />
          that feel like coworkers.
        </h1>
        <p
          data-hero-fade
          className="mt-6 max-w-xl text-pretty font-sans text-base leading-relaxed text-white/55 md:text-lg"
        >
          A terminal app where your opencode boss and sub-agents clock in as coworkers on a
          living floor. See the work move, talk to the boss like a person, and come back tomorrow
          to an office that remembers.
        </p>
        <div data-hero-fade className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/get-started"
            className="bg-white px-7 py-4 font-mono text-sm uppercase tracking-wider text-black transition-transform hover:-translate-y-0.5"
          >
            Open the office
          </Link>
          <Link
            href="/get-started"
            className="border border-white/35 px-7 py-4 font-mono text-sm uppercase tracking-wider text-white transition-colors hover:bg-white/10"
          >
            Tour demo mode
          </Link>

        </div>
        <p data-hero-fade className="mt-5 font-mono text-xs uppercase tracking-wider text-white/45">
          Native CLI · demo first · live opencode when you&apos;re ready
        </p>
      </div>
    </section>
  )
}
