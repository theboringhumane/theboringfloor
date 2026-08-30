import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Chat and work threads',
  description:
    'How the boss chat reads: streaming markdown replies, thinking blocks that unfold and collapse, tool-call rows, sub-agent work threads with inline diffs, and the ctrl+f fullscreen thread focus.',
  alternates: {
    canonical: '/docs/chat-and-threads',
  },
  openGraph: {
    title: 'Chat and work threads · theboringoffice',
    description:
      'How the boss chat reads: streaming markdown replies, thinking blocks that unfold and collapse, tool-call rows, sub-agent work threads with inline diffs, and the ctrl+f fullscreen thread focus.',
    url: `${SITE_URL}/docs/chat-and-threads`,
    type: 'website',
  },
}

const replyKeys = [
  { combo: 'ctrl+t', action: 'expand completed thinking' },
  { combo: '/thinking on|off', action: 'show/hide thinking' },
  { combo: '/tools on|off', action: 'show/hide tool rows' },
]

const threadKeys = [
  { combo: 'ctrl+g', action: 'expand thread inline' },
  { combo: 'ctrl+d', action: 'toggle diff blocks' },
  { combo: '/diffs on|off', action: 'diffs for the session' },
]

const focusKeys = [
  { combo: 'ctrl+f', action: 'open / close thread focus' },
  { combo: 'esc', action: 'back to the office' },
]

const limits = [
  {
    label: 'scrollback',
    body: 'Walking back past the top of the transcript pages older history in one screen at a time — live sessions only. The demo walks a canned 500-row history; real older history is live-only.',
  },
  {
    label: 'while you focus',
    body: 'Thread focus defers repainting the main chat underneath — deliberate, not broken. Leaving catches the parent transcript up in exactly one pass, scroll position, expansions, and draft byte-identical.',
  },
  {
    label: 'during floats',
    body: 'A question or permission float suppresses the top-of-transcript scrollback gesture. Answer or park the ask first, then walk back.',
  },
  {
    label: 'thinking',
    body: 'Thinking transcripts unfold while the turn runs and auto-collapse when the reply lands. Collapsed is the resting state — ctrl+t re-opens completed blocks, /thinking off hides the lane outright.',
  },
]

const keepReading = [
  {
    href: '/docs/permissions-and-questions',
    kicker: 'next',
    title: 'Permissions and questions',
    blurb: 'The asks that float over this chat — the 1 of N queue, the question wizard, and the concierge.',
  },
  {
    href: '/docs/plan-mode',
    kicker: 'next',
    title: 'Plan mode',
    blurb: 'ctrl+p turns this same chat into a read-only planning pass with an approve gate.',
  },
  {
    href: '/docs/keys-and-slash',
    kicker: 'reference',
    title: 'Keys and slash commands',
    blurb: 'Every binding named here — ctrl+t, ctrl+d, ctrl+g, ctrl+f — plus /thinking, /tools, and /diffs.',
  },
  {
    href: '/docs/getting-started',
    kicker: 'start',
    title: 'Getting started',
    blurb: 'Install, open the demo, and watch a chat like this one run itself.',
  },
]

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <div className="mt-10 overflow-hidden border border-border bg-(--shot-frame)">
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
        className="shot-img block h-auto w-full"
      />
    </div>
  )
}

function KeyChips({ keys }: { keys: { combo: string; action: string }[] }) {
  return (
    <div className="mt-6 flex flex-wrap items-center gap-3">
      {keys.map((k) => (
        <span
          key={k.combo}
          className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs"
        >
          <span className="text-accent">{k.combo}</span>
          <span className="text-muted-foreground">{k.action}</span>
        </span>
      ))}
    </div>
  )
}

export default function ChatAndThreadsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Chat &amp; threads</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The chat is the office, in writing.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              One panel carries the whole conversation. The boss&apos;s replies stream in as
              markdown, thinking unfolds while the turn runs and tucks itself away after, every
              tool call leaves a row, and sub-agent labor hangs off the transcript as threads you
              can open fullscreen.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The reply</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Replies stream in as markdown.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The boss&apos;s answer types out character-by-character on a single bubble — no
              spinner in a corner, no batched dump at the end. The transcript is glamour-rendered,
              so bold, lists, and code fences format and wrap inside the panel instead of bleeding
              through the chrome.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              While the turn runs, the thinking transcript unfolds in real time above the reply —
              then folds itself away when the answer lands, so a finished chat reads clean.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Between the two, every tool the boss touches leaves a one-line row in the transcript.
              The working detail stays visible, not hidden behind a spinner.
            </p>
            <KeyChips keys={replyKeys} />
            <Shot
              src="/shots/docs/chat-thinking.png"
              alt="theboringoffice boss chat: a reply streaming in as markdown, an expanded thinking block above it, and tool-call rows between turns"
              caption="boss chat — markdown reply streaming, thinking expanded (ctrl+t)"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Work threads</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Sub-agent work hangs off the transcript as threads.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              When the office dispatches sub-agents, their labor renders as opencode-style threads
              right in the chat. A live thread is one collapsed row — a braille spinner, the
              thread title, a running tool-call count — with a{' '}
              <span className="font-mono text-foreground">↳</span> sneak row previewing the latest
              action underneath. When the work returns, the row rolls up to a{' '}
              <span className="font-mono text-foreground">✓</span> summary.
            </p>
            <div className="mt-8 max-w-3xl border border-border bg-card p-5 font-mono text-xs leading-relaxed">
              <p className="text-foreground">
                ⠿ Explore Task — Scout question kinds recon (· 2 tool calls ✓ done)
              </p>
              <p className="pl-2 text-muted-foreground">
                ↳ Read internal/panels/chat.go … running
              </p>
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <span className="font-mono text-foreground">ctrl+g</span> expands the full thread
              inline in the chat, and every tool call inside carries its own per-call{' '}
              <span className="font-mono text-foreground">↳ diff</span> — line numbers, full-row
              red/green tints, inline syntax; the same diff the git tab shows.{' '}
              <span className="font-mono text-foreground">ctrl+d</span> expands or collapses diff
              blocks on the spot. The full toggle table lives in{' '}
              <Link
                href="/docs/keys-and-slash"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                keys and slash commands
              </Link>
              .
            </p>
            <KeyChips keys={threadKeys} />
            <Shot
              src="/shots/docs/work-threads.png"
              alt="theboringoffice chat with collapsed sub-agent work threads: braille spinner, tool-call count, a ↳ sneak row, and a per-call inline diff"
              caption="work threads — live rows, ↳ sneaks, per-call diffs"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Thread focus</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              ctrl+f takes one thread fullscreen.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Clicking a thread&apos;s frame — the header, its collapsed sneak, or an expanded
              thread&apos;s closing summary — swaps the transcript for a nested focus pane: that
              agent&apos;s complete transcript (every tool call, think body, and per-call diff you
              can click open) scrolling at the frame width, live-pulsed while it works.{' '}
              <span className="font-mono text-foreground">ctrl+f</span> opens the pane without the
              mouse — the most recently expanded thread wins, else any live one.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The statusbar reads{' '}
              <span className="font-mono text-foreground">esc · ctrl+f back to office</span>, and
              leaving returns the parent transcript underneath byte-identical — scroll,
              expansions, and your half-typed draft untouched.
            </p>
            <KeyChips keys={focusKeys} />
            <Shot
              src="/shots/docs/thread-focus.png"
              alt="theboringoffice thread focus: one worker's full transcript fullscreen — tool calls, thinking bodies, and clickable per-call diffs"
              caption="thread focus — one worker, fullscreen, live; esc back"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Honest edges</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              What this doesn&apos;t do yet.
            </h2>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {limits.map((l) => (
                <div key={l.label} className="bg-background p-6">
                  <p className="font-mono text-xs uppercase tracking-wider text-accent">
                    {l.label}
                  </p>
                  <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{l.body}</p>
                </div>
              ))}
            </div>
            <p className="mt-6 max-w-2xl text-pretty text-sm leading-relaxed text-muted-foreground">
              Floats are a lane of their own — permission asks and boss questions sit on top of
              this transcript.{' '}
              <Link
                href="/docs/permissions-and-questions"
                className="text-foreground/90 underline underline-offset-4 transition-colors hover:text-accent"
              >
                Permissions and questions
              </Link>{' '}
              covers them.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border sm:grid-cols-2">
              {keepReading.map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  className="flex flex-col gap-2 bg-background p-6 transition-colors hover:bg-secondary"
                >
                  <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                    {l.kicker}
                  </p>
                  <p className="font-medium text-foreground">{l.title}</p>
                  <p className="text-sm leading-relaxed text-muted-foreground">{l.blurb}</p>
                </Link>
              ))}
            </div>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
