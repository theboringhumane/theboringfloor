import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Get Started | theboringoffice',
  description: 'Install theboringoffice, tour the office in demo mode, and connect your coding agents in minutes.',
}

export default function GetStartedPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border pb-16">
          <div className="mx-auto max-w-3xl px-6 py-24 text-center">
            <SectionTag className="mx-auto">Get Started</SectionTag>
            <h1 className="mt-8 text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              Open the office in minutes
            </h1>
            <p className="mx-auto mt-6 max-w-xl text-pretty leading-relaxed text-muted-foreground">
              Install the native Go CLI, take a tour in demo mode, then run a live office with a real
              opencode boss and working sub-agents.
            </p>

            <div className="mx-auto mt-10 max-w-md border border-border bg-card p-6 text-left font-mono text-xs leading-relaxed">
              <p className="text-muted-foreground"># install the latest binary</p>
              <p>curl -fsSL \</p>
              <p className="pl-4">https://raw.githubusercontent.com/theboringhumane/</p>
              <p className="pl-4">theboringoffice/main/install.sh | sh</p>
              <p className="mt-3 text-muted-foreground"># tour the office</p>
              <p>theboringoffice --demo</p>
              <p className="mt-3 text-muted-foreground"># start a live office</p>
              <p>theboringoffice</p>
              <p className="mt-3 text-muted-foreground"># or attach to an existing server</p>
              <p>theboringoffice --server http://127.0.0.1:4096</p>
            </div>

            <div className="mt-10 flex flex-wrap items-center justify-center gap-3">
              <Link
                href="/docs"
                className="inline-flex items-center bg-foreground px-6 py-3 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
              >
                Read the docs
              </Link>
              <Link
                href="/vision"
                className="inline-flex items-center border border-border px-6 py-3 font-mono text-xs uppercase tracking-wider transition-colors hover:bg-secondary"
              >
                Read our vision
              </Link>
            </div>
          </div>

          <div className="mx-auto mt-12 grid max-w-3xl gap-px border border-border bg-border text-left sm:grid-cols-3">
            <div className="bg-background p-5">
              <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">01 / Install</p>
              <p className="mt-3 text-sm leading-relaxed">The installer places a native binary on your PATH and wires agentmemory as a reboot-safe service.</p>
            </div>
            <div className="bg-background p-5">
              <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">02 / Run</p>
              <p className="mt-3 text-sm leading-relaxed">Live mode starts <span className="font-mono text-foreground">opencode serve</span> and opens the boss chat.</p>
            </div>
            <div className="bg-background p-5">
              <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">03 / Resume</p>
              <p className="mt-3 text-sm leading-relaxed">Your last chat returns automatically. Use <span className="font-mono text-foreground">--session</span> to choose another.</p>
            </div>
          </div>

          <p className="mx-auto mt-8 max-w-2xl text-center font-mono text-xs leading-relaxed text-muted-foreground">
            Config lives at <span className="text-foreground">~/.theboringoffice/configs/brain.json</span>.
            Inspect the defaults with <span className="text-foreground">theboringoffice --print-default-config</span>.
          </p>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
