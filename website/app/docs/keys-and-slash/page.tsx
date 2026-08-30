import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Keys & slash commands | theboringoffice',
  description:
    'Every key and slash command in theboringoffice, grouped by where your hands are — plus the popover that means you never memorize the list.',
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] border border-border bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="mt-10 overflow-hidden border border-border bg-black">
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

type Key = { combo: string; action: string }

function KeyTable({ keys }: { keys: Key[] }) {
  return (
    <div className="mt-6 overflow-hidden border border-border">
      {keys.map((k, i) => (
        <div
          key={k.combo}
          className={`flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-baseline sm:gap-6 ${
            i > 0 ? 'border-t border-border' : ''
          }`}
        >
          <span className="w-40 shrink-0 font-mono text-xs text-accent">{k.combo}</span>
          <span className="text-sm leading-relaxed text-muted-foreground">{k.action}</span>
        </div>
      ))}
    </div>
  )
}

function SlashGroup({
  title,
  commands,
}: {
  title: string
  commands: { cmd: string; does: string }[]
}) {
  return (
    <div className="mt-8">
      <h3 className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
        {title}
      </h3>
      <div className="mt-3 overflow-hidden border border-border">
        {commands.map((c, i) => (
          <div
            key={c.cmd}
            className={`flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-baseline sm:gap-6 ${
              i > 0 ? 'border-t border-border' : ''
            }`}
          >
            <span className="w-48 shrink-0 font-mono text-xs text-accent">{c.cmd}</span>
            <span className="text-sm leading-relaxed text-muted-foreground">{c.does}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

const navigationKeys: Key[] = [
  { combo: 'q / ctrl+c', action: 'quit the office' },
  { combo: 'ctrl+q', action: 'arm quit — works everywhere, embedded terminal included' },
]

const paneKeys: Key[] = [
  {
    combo: 'tab / shift+tab / 1..7',
    action:
      'switch panel: chat · terminal · agents · board · mail · activity · git — only while shell-captured (ctrl+space) are these NOT intercepted',
  },
  { combo: 'up / down / pgup / pgdn / wheel', action: 'scroll the active panel' },
  {
    combo: 'enter / click a file',
    action: 'git tab: open its colored unified diff — b / esc back to the list, r refresh',
  },
]

const composerKeys: Key[] = [
  { combo: 'enter', action: 'send to the boss — free-sends into the backlog while the boss is busy' },
  {
    combo: 'shift+enter / ctrl+j',
    action: 'newline — ctrl+j covers terminals that deliver shift+enter as a bare enter',
  },
  {
    combo: '@',
    action: 'at a word start: attach-file picker — type to filter, arrows choose, enter / tab attach, esc close',
  },
  {
    combo: 'ctrl+v',
    action: 'paste text — attaches the image instead when the clipboard holds one',
  },
  {
    combo: 'paste >20 lines / >2000 chars',
    action: 'collapses to a one-line [pasted N lines · M chars] chip — deletes as one backspace unit, expands back to the full text on send',
  },
  { combo: 'backspace', action: 'on an empty input: drop the newest attachment chip' },
  { combo: 'y a n esc', action: 'answer a permission prompt — allow once / always / reject / defer' },
  { combo: 'ctrl+p', action: 'toggle plan/build mode — the plan pane opens once a plan has content' },
  { combo: 'ctrl+x', action: 'plan mode: approve the presented or edited plan → sent to the build agent' },
]

const threadKeys: Key[] = [
  { combo: 'ctrl+g', action: 'expand / collapse a worker thread inline in the chat' },
  {
    combo: 'ctrl+f',
    action:
      'open the most recent worker thread as a fullscreen focus pane — esc / ctrl+f closes back, office state byte-identical',
  },
  { combo: 'ctrl+t', action: 'expand / collapse completed thinking blocks' },
  { combo: 'ctrl+d', action: 'expand / collapse diff blocks' },
]

const terminalKeys: Key[] = [
  {
    combo: 'ctrl+space',
    action: 'toggle shell keyboard capture both ways — released is the default, leaving auto-releases',
  },
  { combo: 'ctrl+o', action: 'release shell capture (alias — release only, never a dive)' },
  { combo: 'r', action: 'respawn the shell when it dies' },
  { combo: 'drag + release', action: 'select terminal text; release copies, · Copied N chars rides the badge row' },
  { combo: 'esc / typing', action: 'cancel the selection highlight' },
  { combo: 'paste', action: 'into the shell, wrapped in bracketed-paste markers — readline / zle safe' },
]

const browserKeys: Key[] = [
  { combo: 'ctrl+b', action: 'flip the left pane between the floor and the browser — the only way in' },
  { combo: 'up / down / j / k', action: 'move the link cursor' },
  { combo: 'o', action: 'open the focused link — a file rides to the OS browser, http(s) navigates in place' },
  { combo: 'e', action: 'inline URL editor in the location bar — prefilled, enter opens, esc cancels' },
  { combo: 'O (shift+o)', action: 'open the current page in the OS browser' },
  { combo: '[ / ]', action: 'back / forward in the 100-page history ring, scroll offsets restored' },
  { combo: 'r', action: 'reload in place, no duplicate history' },
  { combo: 'pgup / pgdn', action: 'scroll the body' },
  { combo: 'q / esc', action: 'back to the floor' },
]

export default function KeysAndSlashPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Docs — keys & slash</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              You don&apos;t memorize this list. The popover does.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Every command in the office sits behind one discovery surface: type{' '}
              <Code>/</Code> and narrow. The tables below are the reference copy —
              grouped by where your hands already are.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The discovery surface</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The slash popover is how you find everything.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/</Code> at a word start opens the picker. Keep typing to narrow it:{' '}
              <Code>/th</Code> filters to <Code>/theme</Code>, <Code>/themes</Code>,{' '}
              <Code>/thinking</Code>. Arrows move through the matches,{' '}
              <Code>enter</Code> or <Code>tab</Code> accepts, <Code>esc</Code> walks away
              clean with no side effects. Arrowing through the <Code>/theme</Code>{' '}
              matches applies each theme as a live preview while you pass it — accept and
              it sticks; escape and the office settles back to what it was.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              That popover is the intended way to learn the office — the reference table
              further down this page is what it is filtering.
            </p>
            <Shot
              src="/shots/docs/slash-popover.png"
              alt="theboringoffice slash command popover: typing /th filters the command list to theme and thinking commands"
              caption="slash popover — /th narrows the list; arrows move, enter accepts"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Key table</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The keys, by where your hands are.
            </h2>

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Navigation
            </h3>
            <KeyTable keys={navigationKeys} />

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Panes
            </h3>
            <KeyTable keys={paneKeys} />

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Chat composer
            </h3>
            <KeyTable keys={composerKeys} />
            <p className="mt-4 max-w-2xl text-pretty text-sm leading-relaxed text-muted-foreground">
              Attachments stage as dim paperclip chips above the input — capped at five,
              the oldest drops past the cap — ride the message queue like text, and go
              out as prompt file parts; the echoed bubble shows an attachment-count
              suffix. <Code>/clear</Code> or a send clears the chips.
            </p>
            <p className="mt-4 max-w-2xl text-pretty text-sm leading-relaxed text-muted-foreground">
              Pastes route to the focused surface, one deliberate path. In the chat
              input a paste over twenty lines or two thousand chars collapses to a
              one-line <Code>[pasted N lines · M chars]</Code> chip — it deletes as a
              single backspace unit and expands back to the full text on send. A
              question dialog&apos;s free-answer field takes a multi-line paste
              verbatim. The terminal tab wraps the paste in bracketed-paste markers
              for the shell. And picker filter inputs — <Code>/model</Code>,{' '}
              <Code>/session</Code>, <Code>@</Code> — take paste as filter text.
            </p>

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Threads
            </h3>
            <KeyTable keys={threadKeys} />

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Terminal tab
            </h3>
            <KeyTable keys={terminalKeys} />

            <h3 className="mt-10 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Browser tab
            </h3>
            <KeyTable keys={browserKeys} />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The model picker</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              /model swaps the boss&apos;s brain, and it writes back.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Find <Code>/model</Code> in the popover, then hand it the full id as{' '}
              <Code>/model provider/model</Code> — for example{' '}
              <Code>/model anthropic/claude-sonnet-4-5</Code>. From then on the choice
              rides every boss prompt as a{' '}
              <Code>{'{"model":{"providerID","modelID"}}'}</Code> pair, and the command
              writes back to brain.json like <Code>/power</Code> and{' '}
              <Code>/theme</Code> do, so the next boot keeps it.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              One honest caveat from the engine room: role models ride as best-effort
              notes. Which model a sub-agent actually gets is opencode&apos;s call at
              dispatch time, not the office&apos;s.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The picker filters as you type: a <Code>filter: …</Code> row shows the
              live query, the title badge runs <Code>N/M</Code> (narrowed of total), a
              dead query says <Code>(no matches)</Code>, <Code>backspace</Code> edits,{' '}
              <Code>ctrl+u</Code> clears, and <Code>esc</Code> clears a live filter
              first — a second <Code>esc</Code> closes the picker. Pastes land in the
              filter too. The <Code>/session</Code> picker and the <Code>@</Code>{' '}
              attach picker filter the same way.
            </p>
            <Shot
              src="/shots/docs/model-picker.png"
              alt="theboringoffice /model command: picking the boss model as provider/model"
              caption="/model — the boss model rides every prompt and persists"
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Slash reference</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              The full command table, grouped.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              These are office commands — anything the picker doesn&apos;t list is sent
              to the boss as plain text.
            </p>

            <SlashGroup
              title="The queue"
              commands={[
                { cmd: '/queue [clear]', does: 'show the backlog — clear drops it' },
                { cmd: '/route', does: 'force-dispatch the backlog as one [BATCH DISPATCH] now' },
              ]}
            />
            <SlashGroup
              title="The work"
              commands={[
                { cmd: '/stop', does: 'abort current work (boss + workers)' },
                { cmd: '/new', does: 'fresh office — the transcript is archived' },
                { cmd: '/session', does: 'past-sessions picker; switch the office live' },
                { cmd: '/backend', does: 'print the active LLM transport; swap opencode/claudecode while idle' },
                { cmd: '/status', does: 'office status' },
                { cmd: '/mcp [reconnect <name>]', does: 'MCP server status; reconnect one server' },
              ]}
            />
            <SlashGroup
              title="Popovers"
              commands={[
                { cmd: '/perm', does: 're-open an esc\u2019d permission prompt' },
                { cmd: '/question', does: 're-open a deferred boss question' },
              ]}
            />
            <SlashGroup
              title="Permissions"
              commands={[
                { cmd: '/bypass', does: 'toggle session-scoped bypass-permissions — confirm to enable, instant off' },
              ]}
            />
            <p className="mt-4 max-w-2xl text-pretty text-sm leading-relaxed text-muted-foreground">
              <strong className="text-foreground">Bypass permissions.</strong>{' '}
              <Code>/bypass</Code> is the deliberate escape hatch: agents run tools
              and browser actions without asking, for this office session only.
              Enabling asks for an explicit confirm first; disabling is instant.
              While on, every tab&apos;s topbar carries a loud{' '}
              <Code> ⚠ BYPASS </Code> segment, and any ask that still slips through
              — including the office&apos;s own browser-action prompt — is
              auto-approved with a dim log row in the transcript. Toggling respawns
              the backend so the mode actually reaches the agent (claude resumes
              your session context), and every boot starts with bypass OFF —
              brain.json stays untouched. On opencode the allow-all is a real edit
              to the project&apos;s <Code>.opencode/opencode.json</Code> — disabling
              strips it automatically, and it only lingers when the office quits
              with bypass still on (a per-tool <Code>deny</Code> still beats the{' '}
              <Code>*</Code> wildcard); claude writes nothing at all. The full story
              lives with the{' '}
              <Link
                href="/docs/permissions-and-questions"
                className="text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-accent"
              >
                permissions docs
              </Link>
              .
            </p>
            <SlashGroup
              title="Display"
              commands={[
                { cmd: '/theme <name>', does: 'switch theme (persists)' },
                { cmd: '/themes', does: 'list themes' },
                { cmd: '/thinking on|off', does: 'show / hide thinking blocks' },
                { cmd: '/tools on|off', does: 'show / hide tool one-liners' },
                { cmd: '/diffs on|off', does: 'expand / collapse file diffs' },
                { cmd: '/clear', does: 'empty the chat' },
              ]}
            />
            <SlashGroup
              title="Floor & power"
              commands={[
                { cmd: '/compact on|off', does: 'compact layout, this session' },
                { cmd: '/mode normal|compact', does: 'layout mode (persists)' },
                { cmd: '/wide 26..100', does: 'sidebar width — 0 restores the default of 80' },
                { cmd: '/zen', does: 'fullscreen floor, any key exits' },
                { cmd: '/focus floor', does: 'alias of /zen' },
                { cmd: '/power auto|performance|saver', does: 'the power governor — how hard the office renders' },
              ]}
            />
            <SlashGroup
              title="Boss & memory"
              commands={[
                { cmd: '/model provider/model', does: 'boss model — rides every prompt, persists' },
                { cmd: '/memory [filter]', does: 'completed dispatches from the project ledger, newest first' },
              ]}
            />
            <SlashGroup
              title="Meta"
              commands={[
                { cmd: '/help', does: 'the command list, in-app' },
                { cmd: '/quit', does: 'exit theboringoffice' },
              ]}
            />
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Sounds & clipboard</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Two quiet channels worth knowing.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <strong className="text-foreground">Sounds.</strong> The office has a small
              vocabulary of synthesized chimes — queued, send, reply, done, dispatch,
              alert, error. <Code>ui.sounds</Code> in brain.json takes{' '}
              <Code>on</Code>, <Code>bell</Code> or <Code>off</Code>, and{' '}
              <Code>THEBORINGOFFICE_MUTE=1</Code> silences it from the environment. The{' '}
              <Link href="/sounds" className="text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-accent">
                sounds library
              </Link>{' '}
              plays every chime and serves the WAVs.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <strong className="text-foreground">The clipboard.</strong> Drag-select
              over the chat transcript and the span copies through the terminal&apos;s
              OSC52 clipboard on release, with a transient{' '}
              <Code>Copied N chars</Code> note on the status bar; <Code>esc</Code> or a
              plain click clears the highlight. The terminal tab adds pbcopy, wl-copy,
              xclip or xsel ahead of OSC52 — the details live with the{' '}
              <Link
                href="/docs/terminal-and-git-tabs"
                className="text-foreground underline decoration-border underline-offset-4 transition-colors hover:text-accent"
              >
                terminal tab docs
              </Link>
              .
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Ceilings</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              What this doesn&apos;t do yet.
            </h2>
            <ul className="mt-6 flex max-w-2xl flex-col gap-3 leading-relaxed text-muted-foreground">
              <li>
                <strong className="text-foreground">Keys are fixed.</strong> brain.json
                has no remap key — the table above is the table.
              </li>
              <li>
                <strong className="text-foreground">Attachments are files and
                images.</strong> The <Code>@</Code> picker and <Code>ctrl+v</Code> image
                paste are the whole staging surface.
              </li>
              <li>
                <strong className="text-foreground">OSC52 depends on the
                terminal.</strong> A terminal that strips OSC52 keeps the host-clipboard
                paths only — the copied note stays honest, the pasteboard stays empty.
              </li>
            </ul>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Keep reading</SectionTag>
            <div className="mt-8 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-3">
              {[
                {
                  href: '/docs/plan-mode',
                  title: 'Plan mode',
                  body: 'ctrl+p and ctrl+x in context — draft, approve, build.',
                },
                {
                  href: '/docs/permissions-and-questions',
                  title: 'Permissions & questions',
                  body: 'The prompt behind y / a / n / esc.',
                },
                {
                  href: '/docs/queue-board-memory',
                  title: 'Queue, board & memory',
                  body: '/queue, /route, /stop and /memory in depth.',
                },
              ].map((l) => (
                <Link
                  key={l.href}
                  href={l.href}
                  className="group flex flex-col gap-2 bg-background p-8 transition-colors hover:bg-card"
                >
                  <span className="font-mono text-xs uppercase tracking-wider text-accent">
                    {l.title}
                  </span>
                  <span className="text-sm leading-relaxed text-muted-foreground group-hover:text-foreground">
                    {l.body}
                  </span>
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
