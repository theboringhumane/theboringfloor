import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Cue, Panel, Sheet } from '@/components/paper'

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
    body: 'Every feature ships as a numbered wave — a small, reviewable commit you can read in one sitting. Wave 22 through wave 33, and counting.',
  },
  {
    id: '02',
    title: 'Releases in the open',
    body: 'Releases are tagged and cross-compiled in public — currently the v0.2.x wave releases, cut from the same commits you just read.',
  },
  {
    id: '03',
    title: 'This site, same repo',
    body: "The marketing site lives in website/ in the same repository and deploys to Cloudflare Pages. What you're reading is checked in next to the code it describes.",
  },
  {
    id: '04',
    title: 'Company welcome',
    body: 'MIT licensed. Stars, watchers, and issues are all welcome. Tell us which part is boring in the wrong way.',
  },
]

export function OpenSource() {
  return (
    <Sheet id="open-source" className="relative border-t border-rule">
      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10">
        <Beat index="3.4" label="Read the source" title="Read the waves. Watch us work.">
          <p className="max-w-2xl text-pretty leading-relaxed text-ink-soft">
            theboringfloor is built in the open because an office should be inspectable
            end to end. The code, the commits, the releases — this website included — all
            live in one public repository.
          </p>

          <div className="mt-14 grid grid-cols-1 gap-10 lg:grid-cols-2">
            <ScrollReveal direction="left" className="flex flex-col gap-5">
              <p className="max-w-lg text-pretty text-sm leading-relaxed text-ink-soft">
                There is no private version of the office. The floor that ships to your
                terminal is the floor in the repo: the same sprites, the same waves, the
                same commit messages. Want to know how something works? Read it. Want to
                change it? You already know where it lives.
              </p>
              <p className="max-w-lg text-pretty text-sm leading-relaxed text-ink-soft">
                And if you&apos;d rather look around first, demo mode walks you through a
                full shift with a scripted team — no setup, nothing to install beyond the
                one binary.
              </p>
              <Panel label="Try it first" className="mt-2">
                <p className="font-mono text-xs leading-relaxed text-ink">
                  <span className="text-ink-faint">$ </span>theboringfloor --demo
                </p>
              </Panel>
              <div className="mt-2 flex flex-wrap gap-3">
                <a
                  href="https://github.com/theboringhumane/theboringfloor"
                  target="_blank"
                  rel="noreferrer"
                  className="mono-label inline-flex items-center bg-ink px-5 py-3 text-paper transition-opacity hover:opacity-90"
                >
                  github.com/theboringhumane/theboringfloor
                </a>
                <Link
                  href="/get-started"
                  className="mono-label hairline inline-flex items-center px-5 py-3 text-ink transition-colors hover:bg-panel"
                >
                  Tour demo mode
                </Link>
              </div>
            </ScrollReveal>

            <ScrollReveal direction="right">
              <Panel label="git log --oneline">
                <div className="flex flex-col gap-2.5 font-mono text-xs leading-relaxed">
                  {waveLog.map((l) => (
                    <p key={l.hash}>
                      <span className="text-blue">{l.hash}</span>{' '}
                      <span className="text-ink">{l.text}</span>
                    </p>
                  ))}
                  <p className="mt-2 text-ink-faint">
                    … and every wave before it, in the open.
                  </p>
                </div>
              </Panel>
              <Cue className="mt-4 block">no squashed history, no private branch</Cue>
            </ScrollReveal>
          </div>

          <ScrollReveal
            stagger={0.06}
            className="mt-12 grid grid-cols-1 gap-px border border-rule bg-rule md:grid-cols-2 lg:grid-cols-4"
          >
            {inspectable.map((c) => (
              <div key={c.id} className="flex flex-col gap-4 bg-paper p-8">
                <span className="font-mono text-xs text-ink-faint">{c.id}</span>
                <h3 className="text-lg font-medium tracking-tight text-ink">{c.title}</h3>
                <p className="text-pretty text-sm leading-relaxed text-ink-soft">{c.body}</p>
              </div>
            ))}
          </ScrollReveal>
        </Beat>
      </div>
    </Sheet>
  )
}
