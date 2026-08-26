import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Backends | theboringoffice',
  description:
    'opencode server-attach vs the claudecode stream-json child — picking, pinning, and swapping the LLM transport mid-flight.',
  alternates: {
    canonical: '/docs/backends',
  },
  openGraph: {
    title: 'Backends · theboringoffice',
    description:
      'opencode server-attach vs the claudecode stream-json child — picking, pinning, and swapping the LLM transport mid-flight.',
    url: `${SITE_URL}/docs/backends`,
    type: 'website',
  },
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="overflow-hidden border border-border bg-black">
      <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
        <span className="size-2.5 rounded-full bg-destructive/70" />
        <span className="size-2.5 rounded-full bg-chart-4/70" />
        <span className="size-2.5 rounded-full bg-chart-2/70" />
        <span className="ml-2 font-mono text-xs text-muted-foreground">{caption}</span>
      </div>
      <img
        src={src}
        alt={alt}
        width={5086}
        height={2896}
        loading="lazy"
        className="block h-auto w-full"
      />
    </figure>
  )
}

function CmdBlock({ lines }: { lines: { t: string; dim?: boolean }[] }) {
  return (
    <div className="border border-border bg-card p-6 font-mono text-xs leading-relaxed">
      {lines.map((l, i) => (
        <p key={i} className={l.dim ? 'text-muted-foreground' : undefined}>
          {l.t}
        </p>
      ))}
    </div>
  )
}

function Key({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center border border-border px-2 py-0.5 font-mono text-xs text-accent">
      {children}
    </span>
  )
}

export default function BackendsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Backends</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The office doesn&apos;t care which brain the boss has.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Two real transports today — opencode over a live server, or Claude Code as a
              per-turn child process. Same floor, same board, same queue; only the wiring under
              the chat changes.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-7xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              The two transports, side by side
            </h2>
            <div className="mt-6 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2">
              <div className="flex flex-col gap-3 bg-background p-8">
                <h3 className="text-sm font-medium text-foreground">opencode (default)</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  The office spawns — or attaches to — an{' '}
                  <code className="font-mono text-xs text-foreground">opencode serve</code> process
                  and streams its events over SSE. One shared server fronts every session for the
                  working directory; this is what pre-existing brain.json files silently mean.
                </p>
                <span className="mt-auto pt-3 font-mono text-xs text-muted-foreground">
                  backend.name: &quot;opencode&quot;
                </span>
              </div>
              <div className="flex flex-col gap-3 bg-background p-8">
                <h3 className="text-sm font-medium text-foreground">claudecode</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  The office runs the{' '}
                  <code className="font-mono text-xs text-foreground">claude</code> CLI in headless
                  stream-json mode as a child process — one process per turn, streamed line by
                  line into the same chat surface. Needs the claude CLI on PATH; absent, the office
                  warns instead of failing.
                </p>
                <span className="mt-auto pt-3 font-mono text-xs text-muted-foreground">
                  backend.name: &quot;claudecode&quot;
                </span>
              </div>
            </div>
            <p className="mt-4 max-w-2xl font-mono text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
              Also on the menu: Codex (Coming Soon) · Cursor (Coming Soon) · Pi (Coming Soon)
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <div className="grid gap-10 md:grid-cols-2">
              <Shot
                src="/shots/docs/office-overview.png"
                alt="theboringoffice on the opencode backend: floor, chat thread, and panels"
                caption="opencode (default)"
              />
              <Shot
                src="/shots/docs/backend-claude.png"
                alt="theboringoffice on the claudecode backend: claude CLI child streaming a boss reply"
                caption="claudecode — stream-json child"
              />
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Same office either way — the topbar badge tells you which brain is on duty.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Picking it, three ways
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              Flag beats env, env beats brain, brain beats default.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The precedence chain is short and total:{' '}
              <code className="font-mono text-xs text-foreground">--backend</code> flag on the
              binary overrides{' '}
              <code className="font-mono text-xs text-foreground">THEBORINGOFFICE_BACKEND</code>{' '}
              (with <code className="font-mono text-xs text-foreground">GRAFEIO_BACKEND</code> as
              the upgrade-era fallback), which overrides{' '}
              <code className="font-mono text-xs text-foreground">brain.json</code>&apos;s{' '}
              <code className="font-mono text-xs text-foreground">backend.name</code> — which{' '}
              <code className="font-mono text-xs text-foreground">install.sh --backend</code> seeds
              for you. Nothing set anywhere means opencode. An invalid name is rejected with a
              stderr warning and the office falls back to opencode rather than refusing to boot.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '# 1 · seed it at install time', dim: true },
                  { t: '... | sh -s -- --backend claudecode' },
                  { t: '', dim: true },
                  { t: '# 2 · pin it in ~/.theboringoffice/configs/brain.json', dim: true },
                  { t: '"backend": { "name": "claudecode" }' },
                  { t: '', dim: true },
                  { t: '# 3 · pin one boot from the shell', dim: true },
                  { t: 'THEBORINGOFFICE_BACKEND=claudecode theboringoffice' },
                ]}
              />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Swapping mid-flight
            </h2>
            <h3 className="mt-4 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight">
              <span className="font-mono">/backend</span> swaps live — but only from an idle office.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Bare <Key>/backend</Key> prints the active transport. With a name —{' '}
              <Key>/backend claudecode</Key> — it swaps mid-flight, archives the current turn,
              persists the name to brain.json, and lands one status line:{' '}
              <code className="font-mono text-xs text-foreground">
                [theboringoffice] backend: opencode → claudecode
              </code>
              . The topbar shows the active name between mode and agents at all times.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The gate is strict by design: a boss turn in flight, a queued backlog, live workers,
              or an unanswered question or permission each get a refusal that names the blockers —
              the swap never preempts work. Settle the floor first (
              <Key>/stop</Key> helps), then swap.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Sessions don&apos;t cross the aisle: session.json pins session IDs{' '}
              <strong className="font-medium text-foreground">per transport</strong> (
              <code className="font-mono text-xs text-foreground">primaryIDs</code>), so swapping
              back later resumes that transport&apos;s own session instead of cross-pinning IDs
              between backends. Resuming a specific session by ID is covered in{' '}
              <Link
                href="/docs/getting-started"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                getting started
              </Link>
              , and the key chip for every other slash command lives in{' '}
              <Link
                href="/docs/keys-and-slash"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                keys &amp; slash commands
              </Link>
              .
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>What this doesn&apos;t do yet</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Role models are best-effort — sub-agent model dispatch is opencode&apos;s call, not
              the office&apos;s. And Codex, Cursor, and Pi remain &quot;(Coming Soon)&quot; tags;
              the two transports above are the whole menu today.
            </p>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
