import { ProductScreenshot } from "@/components/product-screenshot";
import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Model selection | theboringfloor',
  description:
    "/model and /submodel pick each backend's own native model catalog for the boss and for sub-agents — per backend, with the limits stated plainly.",
  alternates: {
    canonical: '/docs/models',
  },
  openGraph: {
    title: 'Model selection · theboringfloor',
    description:
      "/model and /submodel pick each backend's own native model catalog for the boss and for sub-agents — per backend, with the limits stated plainly.",
    url: `${SITE_URL}/docs/models`,
    type: 'website',
  },
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="m-0">
      <div className="doc-brackets hairline bg-paper-2 p-2 md:p-3">
        <ProductScreenshot
          src={src}
          alt={alt}
          width={5086}
          height={2896}
          loading="lazy"
          className="shot-img block h-auto w-full"
        />
      </div>
      <figcaption className="mono-label mt-3 text-ink-faint">{caption}</figcaption>
    </figure>
  )
}

function CmdBlock({ lines }: { lines: { t: string; dim?: boolean }[] }) {
  return (
    <div className="doc-brackets hairline bg-paper-2 p-6 font-mono text-xs leading-relaxed">
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
    <span className="inline-flex items-center hairline px-2 py-0.5 font-mono text-xs text-accent">
      {children}
    </span>
  )
}

export default function ModelsPage() {
  return (
    <>
      <SiteHeader />
      <main className="relative paper-ground">
        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>03 · Model selection</SectionTag>
            <h1 className="mt-8 max-w-3xl display-lg text-balance text-ink">
              Every backend has its own catalog. The office just asks it.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              <span className="font-mono">/model</span> picks the boss&apos;s model.{' '}
              <span className="font-mono">/submodel</span> picks a model for one native
              agent type. Both now work natively on OpenCode, Claude Code, and Codex — not
              just the one backend that used to have it — and each backend&apos;s own
              catalog, defaults, and limits drive what you see.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="mono-label">Two commands</h2>
            <h3 className="mt-4 max-w-2xl display-md text-balance text-ink">
              The boss model, and a model per agent type.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <span className="font-mono">/model</span> opens the active backend&apos;s
              native main-model catalog. If discovery is unavailable for some reason, hand
              it a native reference directly:{' '}
              <code className="font-mono text-xs text-foreground">/model &lt;native-ref&gt;</code>.{' '}
              <span className="font-mono">/submodel</span> is the same idea one level
              down — pick a native agent type, then pick (or set) that type&apos;s model,
              where the backend supports it. Native agent names come from the backend
              itself; they are not the same list as the floor roster&apos;s roles.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '/model', dim: false },
                  { t: '  → the boss (main) model picker', dim: true },
                  { t: '/model <native-ref>', dim: false },
                  { t: '  → set the boss model manually', dim: true },
                  { t: '', dim: true },
                  { t: '/submodel', dim: false },
                  { t: '  → pick an agent type, then its model', dim: true },
                  { t: '/submodel <agent>', dim: false },
                  { t: '  → open the picker for that agent type', dim: true },
                  { t: '/submodel <agent> <native-ref>', dim: false },
                  { t: '  → set that agent/model pair manually', dim: true },
                ]}
              />
            </div>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Manual commands still go through native validation — typing a reference by
              hand is not a way around an unsupported capability. A rejected choice is
              never saved, and it is never silently swapped for a default.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <Shot
              src="/shots/any-model-backend.png"
              alt="theboringfloor model picker showing a native catalog with current, default, and unavailable badges"
              caption="/model — native catalog, current backend"
            />
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="mono-label">How the picker behaves</h2>
            <h3 className="mt-4 max-w-2xl display-md text-balance text-ink">
              Type to filter, arrows to move, Enter to apply — then it saves.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Every row in the catalog carries a description straight from the native
              client, plus badges where they apply:{' '}
              <code className="font-mono text-xs text-foreground">unavailable</code> for an
              entry the backend won&apos;t currently honor,{' '}
              <code className="font-mono text-xs text-foreground">current</code> for the
              office&apos;s saved override, and{' '}
              <code className="font-mono text-xs text-foreground">backend default</code>{' '}
              for the native client&apos;s own default — which describes the catalog, not
              the effective model of a session you&apos;ve already resumed.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Typing filters the list live. <Key>Esc</Key> cancels — including a catalog
              that&apos;s still loading — and once you&apos;ve picked a row, input waits
              for the backend to acknowledge before doing anything else. The order is
              always the same: the choice is <strong className="font-medium text-foreground">applied</strong>{' '}
              to the running backend first; only once that adapter accepts it does the
              office <strong className="font-medium text-foreground">save</strong> the
              preference for next time. If the save itself fails after a successful apply,
              that&apos;s reported as its own, separate message — the runtime change has
              already happened either way.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="mono-label">Saved per backend</h2>
            <h3 className="mt-4 max-w-2xl display-md text-balance text-ink">
              Switching backends never clobbers another backend&apos;s choice.
            </h3>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Accepted choices land in{' '}
              <code className="font-mono text-xs text-foreground">
                ~/.theboringfloor/configs/brain.json
              </code>{' '}
              under <code className="font-mono text-xs text-foreground">modelPreferences</code>,
              keyed by backend name — <code className="font-mono text-xs text-foreground">opencode</code>,{' '}
              <code className="font-mono text-xs text-foreground">codex</code>, or{' '}
              <code className="font-mono text-xs text-foreground">claudecode</code>. Each
              backend gets its own <code className="font-mono text-xs text-foreground">boss</code>{' '}
              reference and its own <code className="font-mono text-xs text-foreground">agents</code>{' '}
              map keyed by native agent name. Pick a model on Claude Code, swap to Codex
              with <Link href="/docs/backends" className="text-foreground underline underline-offset-4 hover:text-accent">/backend</Link>,
              swap back — your Claude Code choice is still exactly where you left it.
            </p>
            <div className="mt-8 max-w-2xl">
              <CmdBlock
                lines={[
                  { t: '{', dim: false },
                  { t: '  "modelPreferences": {', dim: false },
                  { t: '    "opencode": {"boss": "anthropic/claude-sonnet-4",', dim: true },
                  { t: '                "agents": {"explore": "anthropic/claude-sonnet-4"}},', dim: true },
                  { t: '    "codex": {"boss": "gpt-5.4"},', dim: true },
                  { t: '    "claudecode": {"boss": "opus[1m]",', dim: true },
                  { t: '                  "agents": {"Explore": "sonnet"}}', dim: true },
                  { t: '  }', dim: false },
                  { t: '}', dim: false },
                ]}
              />
            </div>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              If OpenCode has no scoped choice saved, the boss model falls back to{' '}
              <code className="font-mono text-xs text-foreground">backend.bossModel</code>,
              then <code className="font-mono text-xs text-foreground">boss.model</code>,
              and a missing native agent choice falls back to the legacy{' '}
              <code className="font-mono text-xs text-foreground">agentModels</code> map —
              that fallback chain is OpenCode-only; Claude Code and Codex don&apos;t
              inherit it. An explicit empty scoped value masks the legacy fallback rather
              than falling through it.
            </p>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <h2 className="mono-label">Three backends, three different deals</h2>
            <h3 className="mt-4 max-w-2xl display-md text-balance text-ink">
              The limits are the useful part. Read them before you rely on per-agent models.
            </h3>
            <div className="mt-6 grid grid-cols-1 gap-px overflow-hidden hairline bg-border md:grid-cols-3">
              <div className="flex flex-col gap-3 bg-background p-8">
                <h4 className="text-sm font-medium text-foreground">OpenCode</h4>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  Main-model references are <code className="font-mono text-xs text-foreground">provider/model</code>,
                  e.g. <code className="font-mono text-xs text-foreground">anthropic/claude-sonnet-4</code>.
                  Native subagents are selectable too, and an agent model change
                  preserves the project&apos;s existing agent prompts, tools, and every
                  other unrelated config field. It requires every session in the project
                  to be idle first — a busy or retrying session gets the change rejected.
                </p>
                <p className="mt-auto pt-3 text-sm font-medium text-foreground">
                  The honest limit: a saved model OpenCode rejects fails the send
                  outright — the office does not silently retry the turn without it.
                </p>
              </div>
              <div className="flex flex-col gap-3 bg-background p-8">
                <h4 className="text-sm font-medium text-foreground">Codex</h4>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  <span className="font-mono">/model</span> works for the boss —{' '}
                  <code className="font-mono text-xs text-foreground">/model gpt-5.4</code>{' '}
                  is a valid example token, not a guarantee for your account. There is no
                  universal reset-to-default for a resumed thread; pick an explicit model
                  if you want to change one.
                </p>
                <p className="mt-auto pt-3 text-sm font-medium text-foreground">
                  The honest limit: per-agent switching is unavailable. Codex&apos;s
                  native role model settings override an explicit dispatch model, so{' '}
                  <span className="font-mono">/submodel</span> refuses a per-agent choice
                  with an explicit error rather than accepting a setting it can&apos;t
                  honor — configure Codex roles directly, or inherit models, instead.
                </p>
              </div>
              <div className="flex flex-col gap-3 bg-background p-8">
                <h4 className="text-sm font-medium text-foreground">Claude Code</h4>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  Native aliases only — <code className="font-mono text-xs text-foreground">sonnet</code>{' '}
                  or <code className="font-mono text-xs text-foreground">opus[1m]</code>{' '}
                  for the main session. An Opus agent alias derived from{' '}
                  <code className="font-mono text-xs text-foreground">opus[1m]</code> is
                  labeled as an agent alias and drops the{' '}
                  <code className="font-mono text-xs text-foreground">[1m]</code> context
                  modifier. A saved agent choice only fills in a dispatch that&apos;s
                  missing a model — an explicit dispatch model is left alone, and a fork
                  agent always inherits its parent&apos;s model.
                </p>
                <p className="mt-auto pt-3 text-sm font-medium text-foreground">
                  How it actually lands: a <code className="font-mono text-xs text-foreground">PreToolUse</code>{' '}
                  hook fires when a sub-agent is dispatched, and that hook is what
                  injects the saved per-agent model — a dispatch with no native agent
                  type has nothing to key a type-specific preference off of, so it uses
                  the boss model. The hook must be acknowledged by Claude Code for the
                  turn to proceed.
                </p>
              </div>
            </div>
          </div>
        </section>

        <section className="border-b border-rule">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>Elsewhere in the manual</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              The full key and slash-command tables, including{' '}
              <span className="font-mono">/model</span>&apos;s filter keys, live in{' '}
              <Link
                href="/docs/keys-and-slash"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                keys &amp; slash commands
              </Link>
              . Backend setup, authentication, and how <span className="font-mono">/backend</span>{' '}
              swaps live are covered in{' '}
              <Link
                href="/docs/backends"
                className="text-foreground underline underline-offset-4 hover:text-accent"
              >
                backends
              </Link>
              .
            </p>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
