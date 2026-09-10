import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { Beat, Cue, Stamp } from '@/components/paper'

export const metadata: Metadata = {
  title: 'Get Started | theboringfloor',
  description: 'Install theboringfloor, tour the office in demo mode, and connect your coding agents in minutes.',
}

/**
 * A mono specimen block. Lines starting with `#` print as faint comments; every
 * other line is a command and is reproduced verbatim — do not reformat, wrap or
 * "tidy" these strings.
 */
function Specimen({ lines }: { lines: string[] }) {
  return (
    <div className="doc-brackets hairline overflow-x-auto bg-paper-2 p-5 font-mono text-xs leading-relaxed md:p-6">
      {lines.map((line, i) =>
        line.startsWith('#') ? (
          <p key={line} className={i === 0 ? 'text-ink-faint' : 'mt-3 text-ink-faint'}>
            {line}
          </p>
        ) : (
          <p key={line} className="text-ink">
            {line}
          </p>
        ),
      )}
    </div>
  )
}

const installLines = ['# install the latest binary', 'curl -fsSL https://boringfloor.com/install.sh | sh']

const runLines = [
  '# tour the office',
  'theboringfloor --demo',
  '# start a live office (opencode is the default transport)',
  'theboringfloor',
  '# same office, on your Claude Code CLI',
  'theboringfloor --backend claudecode',
  '# or use your Codex CLI login',
  'codex login',
  'theboringfloor --backend codex',
  '# or attach to an existing opencode server',
  'theboringfloor --server http://127.0.0.1:4096',
]

const resumeLines = ['# inspect the defaults', 'theboringfloor --print-default-config']

export default function GetStartedPage() {
  return (
    <div className="paper-ground min-h-svh">
      <div className="doc-frame">
        <SiteHeader framed />
        <main>
          {/* Masthead — the cover line of the install sheet. */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 md:py-20 lg:px-14">

            <div className="flex items-baseline gap-4 border-b border-rule pb-3">
              <span className="mono-label text-ink">00</span>
              <span className="mono-label">Get started</span>
              <span className="h-px flex-1 bg-rule" aria-hidden="true" />
              <span className="mono-label hidden sm:inline">macOS · Linux · Windows</span>
            </div>

            <h1 className="display-xl mt-8 max-w-[16ch] text-balance text-ink">
              Open the office in minutes
            </h1>

            <p className="mt-6 max-w-xl text-pretty text-base leading-relaxed text-ink-soft">
              Install the native Go CLI, take a tour in demo mode, then run a live office with a
              real boss and working sub-agents — on OpenCode, Claude Code, or Codex.
            </p>

            <div className="mt-8 flex flex-wrap items-center gap-3">
              <Link
                href="/docs"
                className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
              >
                <Stamp className="transition-colors hover:border-ink hover:text-ink">
                  Read the docs
                </Stamp>
              </Link>
              <Link
                href="/vision"
                className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
              >
                <Stamp className="transition-colors hover:border-ink hover:text-ink">
                  Read our vision
                </Stamp>
              </Link>
            </div>
          </section>

          {/* The three numbered steps, each with its specimen. */}
          <section className="relative border-b border-rule px-6 py-14 md:px-10 md:py-16 lg:px-14">

            <div className="flex flex-col gap-14">
              <Beat index="01" label="Install the binary">
                <div className="grid gap-6 md:grid-cols-2 md:items-start">
                  <p className="max-w-prose text-sm leading-relaxed text-ink-soft">
                    The installer places a native binary on your PATH and wires agentmemory as a
                    reboot-safe service.
                  </p>
                  <Specimen lines={installLines} />
                </div>
              </Beat>

              <Beat index="02" label="Run the office">
                <div className="grid gap-6 md:grid-cols-2 md:items-start">
                  <p className="max-w-prose text-sm leading-relaxed text-ink-soft">
                    Live mode starts <span className="font-mono text-ink">opencode serve</span> by
                    default — or one <span className="font-mono text-ink">claude</span> CLI process
                    per session with{' '}
                    <span className="font-mono text-ink">--backend claudecode</span> — and opens the
                    boss chat.
                  </p>
                  <Specimen lines={runLines} />
                </div>
              </Beat>

              <Beat index="03" label="Resume the shift">
                <div className="grid gap-6 md:grid-cols-2 md:items-start">
                  <p className="max-w-prose text-sm leading-relaxed text-ink-soft">
                    Ctrl+E opens project floors; Ctrl+N starts a conversation with a backend and
                    team. Your last chat returns automatically. Use{' '}
                    <span className="font-mono text-ink">--session</span> to choose another.
                  </p>
                  <Specimen lines={resumeLines} />
                </div>
              </Beat>
            </div>
          </section>

          {/* Marginalia — where the config lives. */}
          <section className="relative border-b border-rule px-6 py-12 md:px-10 lg:px-14">
            <div className="flex items-center gap-3">
              <span className="mono-label text-ink">Config</span>
              <span className="h-px flex-1 bg-rule" aria-hidden="true" />
            </div>
            <p className="mt-6 max-w-2xl font-mono text-xs leading-relaxed text-ink-soft">
              Config lives at <span className="text-ink">~/.theboringfloor/configs/brain.json</span>.
              Pin the default transport with{' '}
              <span className="text-ink">{`"backend": { "name": "claudecode" }`}</span> — opencode
              stays the default otherwise; <span className="text-ink">/backend</span> swaps mid-flight
              and persists. Inspect the defaults with{' '}
              <span className="text-ink">theboringfloor --print-default-config</span>.
            </p>
            <Cue className="mt-6 block">one file, no wizard</Cue>
          </section>
        </main>
        <SiteFooter />
      </div>
    </div>
  )
}
