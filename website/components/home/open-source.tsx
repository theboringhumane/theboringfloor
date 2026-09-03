import Link from 'next/link'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const waveLog = [
  { hash: 'f09d0ba', text: 'website: homepage copy in the vision voice' },
  { hash: '830eca6', text: 'wave 33: topbar reads the ldflags-stamped version' },
  { hash: 'f26ad24', text: 'wave 32: transcript mouse selection, OSC52 copy' },
  { hash: 'aa1e645', text: 'wave 31: transcript UX pack' },
  { hash: '8ec5d0b', text: 'wave 30: session lifecycle UX' },
  { hash: '38a6cf6', text: 'wave 29: website on Cloudflare Pages' },
]

const inspectable = [
  {
    id: '01',
    title: 'The log is the changelog',
    body: 'Every feature ships as a numbered wave — a small, reviewable commit you can read in one sitting. Wave 22 through wave 33 and counting.',
  },
  {
    id: '02',
    title: 'Releases in the open',
    body: 'Releases are tagged and cross-compiled publicly — currently the v0.2.x wave releases, cut from the same commits you just read.',
  },
  {
    id: '03',
    title: 'This site, same repo',
    body: "The marketing site lives in website/ in the same repository and deploys to Cloudflare Pages. What you're reading is checked in next to the code it describes.",
  },
  {
    id: '04',
    title: 'Company welcome',
    body: 'MIT licensed, and stars, watchers, and issues are all welcome. Tell us which part is boring in the wrong way.',
  },
]

export function OpenSource() {
  return (
    <section id="open-source" className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <SectionTag>Built in public</SectionTag>
        <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
          Read the waves. Watch us work.
        </h2>
        <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
          theboringfloor is built in the open because an office should be inspectable
          end-to-end. The code, the commits, the releases — even this website — all live in one
          public repository.
        </p>

        <div className="mt-14 grid grid-cols-1 gap-10 lg:grid-cols-2">
          <ScrollReveal direction="left" className="flex flex-col gap-5">
            <p className="max-w-lg text-pretty text-sm leading-relaxed text-muted-foreground">
              There is no private version of the office. The floor that ships to your terminal
              is the floor in the repo — the same sprites, the same waves, the same commit
              messages. If you want to know how something works, you read it; if you want to
              change how it works, you know exactly where it lives.
            </p>
            <p className="max-w-lg text-pretty text-sm leading-relaxed text-muted-foreground">
              And if you&apos;d rather just look around first, demo mode walks you through a
              full shift with a scripted team — no setup, nothing to install beyond the one
              binary.
            </p>
            <div className="mt-2 border border-border bg-card px-4 py-3 font-mono text-xs leading-relaxed">
              <p>
                <span className="text-muted-foreground">$ </span>theboringfloor --demo
              </p>
            </div>
            <div className="mt-2 flex flex-wrap gap-3">
              <a
                href="https://github.com/theboringhumane/theboringfloor"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center bg-foreground px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
              >
                github.com/theboringhumane/theboringfloor
              </a>
              <Link
                href="/get-started"
                className="inline-flex items-center border border-border px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
              >
                Tour demo mode
              </Link>
            </div>
          </ScrollReveal>

          <ScrollReveal direction="right" className="border border-border bg-card">
            <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
              <span className="size-2.5 rounded-full bg-destructive/70" />
              <span className="size-2.5 rounded-full bg-chart-4/70" />
              <span className="size-2.5 rounded-full bg-chart-2/70" />
              <span className="ml-2 font-mono text-xs text-muted-foreground">
                git log --oneline
              </span>
            </div>
            <div className="flex flex-col gap-2.5 px-5 py-5 font-mono text-xs leading-relaxed">
              {waveLog.map((l) => (
                <p key={l.hash}>
                  <span className="text-accent">{l.hash}</span>{' '}
                  <span className="text-foreground/80">{l.text}</span>
                </p>
              ))}
              <p className="mt-2 text-muted-foreground">… and every wave before it, in the open.</p>
            </div>
          </ScrollReveal>
        </div>

        <ScrollReveal
          stagger={0.06}
          className="mt-12 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 lg:grid-cols-4"
        >
          {inspectable.map((c) => (
            <div key={c.id} className="flex flex-col gap-4 bg-background p-8">
              <span className="font-mono text-xs text-muted-foreground">{c.id}</span>
              <h3 className="text-lg font-semibold tracking-tight">{c.title}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                {c.body}
              </p>
            </div>
          ))}
        </ScrollReveal>
      </div>
    </section>
  )
}
