import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'

export const metadata: Metadata = {
  title: 'Browser tab | theboringfloor',
  description:
    'A text-mode HTML page viewer on the left pane — headings, indexed links, history, /open — with rendered headless screenshots on kitty-capable terminals and an opt-in zenbu terminal-browser embedded lane.',
}

function Code({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-[0.2rem] border border-border bg-card px-1 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

function Chip({ combo, action }: { combo: string; action: string }) {
  return (
    <span className="inline-flex items-center gap-2 border border-border px-3 py-1.5 font-mono text-xs">
      <span className="text-accent">{combo}</span>
      <span className="text-muted-foreground">{action}</span>
    </span>
  )
}

export default function BrowserTabPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Docs — browser tab</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The web, as text, one tab over.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              The left pane is a two-tab slot — the office floor by default, the
              browser behind <Code>ctrl+b</Code> — and that second tab is a real
              in-TUI page viewer. Web pages render
              as navigable text and link rows — no external binary, no headless
              Chromium, no runtime to install — so it works on every terminal the
              office runs on. On kitty-capable terminals with Chrome installed the
              tab upgrades to rendered headless screenshots; an older embedded
              zenbu <Code>terminal-browser</Code> lane survives behind an explicit
              opt-in flag; everywhere else, text is the feature, not the fallback.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The text lane</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              HTML in, readable rows out.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              A loaded page paints a <Code>▸ &lt;url&gt; · &lt;title&gt;</Code> bar over
              the rendered body: a title line, bold headings, wrapped paragraphs,
              bullet rows, <Code>a │ b</Code> table rows, code rows, and{' '}
              <Code>🖼 &lt;alt&gt;</Code> image chips — image bytes are never fetched,
              so a heavy page stays cheap. Links arrive as <Code>text [n]</Code> with
              their URLs indexed in a side map, stable-ordered and deduped by exact
              URL.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Fetches are disciplined: 10 seconds bounded, 4 MiB capped, and the
              payload is content-sniffed — HTML only. A PDF or a PNG lands a dim{' '}
              <Code>unsupported content type</Code> row instead of a parse, and a
              non-2xx lands a dim error row. The viewer never pretends a failure is
              a page.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Getting around</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Links are a cursor, not a mouse.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>↑</Code>/<Code>↓</Code> (or <Code>j</Code>/<Code>k</Code>) move the
              link cursor — link rows go dim to bright as it lands, auto-scrolled
              into view. <Code>o</Code> opens the focused link: a local file rides to
              the OS browser, an http(s) link navigates in place. <Code>e</Code> opens
              an inline URL editor right in the location bar — prefilled with the
              current URL, <Code>enter</Code> opens the edit, <Code>esc</Code> cancels
              — and <Code>O</Code> (shift+o) sends the current page to the OS browser.{' '}
              <Code>[</Code> and <Code>]</Code> walk a 100-page history ring with scroll
              offsets restored, <Code>r</Code> reloads in place without duplicating
              history, and <Code>q</Code> / <Code>esc</Code> leaves back to the floor.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The browser lives on the left pane, in the slot that usually holds the
              office floor — <Code>ctrl+b</Code> flips between floor and browser, and{' '}
              <Code>q</Code> / <Code>esc</Code> from the browser returns to the floor.
              The sidebar strip keeps its seven tabs — chat · terminal · agents ·
              board · mail · activity · git — with no browser entry and no digit key
              for it. Idle, the tab shows its starter card:{' '}
              <Code>▸ enter a url · /open &lt;url&gt; · e to edit · o for file</Code>.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="↑ / ↓" action="move the link cursor" />
              <Chip combo="o" action="open the focused link" />
              <Chip combo="e" action="edit the URL inline, in the location bar" />
              <Chip combo="O" action="open the page in the OS browser" />
              <Chip combo="[ / ]" action="back / forward, 100-page ring" />
              <Chip combo="r" action="reload in place" />
              <Chip combo="pgup / pgdn" action="scroll the body" />
              <Chip combo="q / esc" action="back to the floor" />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Opening pages</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              One slash command, the chat&apos;s open key — and the boss.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              <Code>/open &lt;url&gt;</Code> jumps the browser tab to a page: a{' '}
              <Code>file://</Code> URL or a bare path reads straight off disk, an
              http(s) URL fetches — subject to the policy below. From the chat side,
              the <Code>o</Code> hotkey on a bubble carrying a verified URL or
              on-disk path opens that target too, so a link the boss hands you is one
              keypress from rendered.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The boss can open pages for you outright. On either backend — opencode
              or Claude Code — an agent can ask the office to open a URL in the
              browser tab; when it does, the left slot flips to the browser
              automatically and a dim transcript note{' '}
              <Code>browser: opening &lt;url&gt; (asked by the boss)</Code> marks who
              sent you there. A refused open posts the reason instead of a page.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The boss&apos;s reach goes past opening. It can take a screenshot of a
              page for you — rendered in the tab on kitty-capable terminals, the PNG
              saved under <Code>~/.theboringfloor/shots/</Code> with the path posted
              to the transcript — or snapshot a page for itself, the text and links
              riding back to it as a follow-up message so it can read what it sent you
              to. And when reading is not enough it can ask to <strong className="text-foreground">act</strong>{' '}
              on a page — click an element, fill a field, evaluate JavaScript — which
              always routes through your permission prompt first: approve-once only,
              no standing grant, not even for localhost. Each action drives a fresh
              page load, and the outcome — the result, the error, or your rejection —
              goes back to the boss as a follow-up.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>Built-in first</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Members open; agents direct the office.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Members use <Code>/open &lt;url&gt;</Code>. Agents use the office&apos;s
              own-line directives instead of launching a browser process themselves:
            </p>
            <div className="mt-6 max-w-3xl border border-border bg-card p-5 font-mono text-xs leading-relaxed text-foreground">
              <p>/open &lt;url&gt; <span className="text-muted-foreground">member opens a page</span></p>
              <p>⟦open-browser: URL⟧ <span className="text-muted-foreground">agent opens it in the member&apos;s tab</span></p>
              <p>⟦browser-screenshot: URL⟧ <span className="text-muted-foreground">agent renders a PNG for the member</span></p>
              <p>⟦browser-snapshot: URL⟧ <span className="text-muted-foreground">agent reads text and links back</span></p>
              <p>⟦browser-action: URL | click: CSS-SELECTOR⟧ <span className="text-muted-foreground">agent acts only after permission</span></p>
              <p>⟦browser-action: URL | fill: CSS-SELECTOR = VALUE⟧ <span className="text-muted-foreground">agent fills only after permission</span></p>
              <p>⟦browser-action: URL | eval: JS-EXPRESSION⟧ <span className="text-muted-foreground">agent evaluates only after permission</span></p>
            </div>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Open, screenshot, and snapshot are read-only. Browser action can click,
              fill, or evaluate JavaScript, so it is mutating and always
              permission-gated. The built-in directives work for localhost and
              external <Code>https://</Code> pages. Agents fall back to Chrome,
              Chromium, Playwright, Puppeteer, or a terminal browser only when the
              member explicitly asks, or when the built-in path fails and the agent
              explains why.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>The premium lane</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Rendered screenshots by default; the embedded browser is opt-in.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              On a kitty-capable terminal — kitty or ghostty; tmux and the iTerm2
              family stay text — with Chrome installed, the tab shows rendered
              screenshots of the page: the headless browser runs out-of-process
              and the pane paints the result under a <Code> shot </Code> badge and a{' '}
              <Code>▸ headless chromium · &lt;url&gt;</Code> strip. The text viewer
              remains the universal default everywhere else; the premium path is an
              upgrade the tab resolves live, not a dependency.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              Every shot also lands on disk —{' '}
              <Code>~/.theboringfloor/shots/&lt;ts&gt;-&lt;hash&gt;.png</Code>, or the
              temp dir when the office home is overridden — with the path posted to
              the transcript, so your <Code>o</Code>-to-open habit works on the file
              too. On a terminal without kitty graphics the text lane carries a dim{' '}
              <Code>screenshot: &lt;path&gt;</Code> row in place of the paint, and a
              failed shot — Chrome absent, navigation refused, timeout — stays text
              with one dim classified reason row, never a blank pane.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The older EMBEDDED lane — zenbu&apos;s <Code>terminal-browser</Code>,
              a real Chromium app living inside the pane — is retained but{' '}
              <strong className="text-foreground">off by default</strong>. Opt in
              explicitly: install the binary (re-run the office installer with{' '}
              <Code>--with-terminal-browser</Code>) and export{' '}
              <Code>THEFLOOR_ZENBU_LANE=1</Code>. Then the tab embeds the
              live page at the pane&apos;s exact pixel size under a top strip{' '}
              <Code>▸ zenbu terminal-browser · &lt;url&gt;</Code> and a{' '}
              <Code>zenbu</Code> badge.
            </p>
            <p className="mt-4 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The fallback is honest both ways: a non-zero or instant (&lt;300ms) exit
              drops back to the text viewer with the URL state kept and a dim{' '}
              <Code>zenbu exited (&lt;code&gt;) — falling back to text mode</Code>{' '}
              note; a clean exit just returns to text quietly. Two kill-switches
              force the text lane — <Code>THEFLOOR_TERMINAL_BROWSER_OFF=1</Code>{' '}
              or <Code>THEFLOOR_NO_TERMINAL_BROWSER=1</Code> — and they win
              over the opt-in flag.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="kitty / ghostty + Chrome" action="rendered screenshots" />
              <Chip combo="THEFLOOR_ZENBU_LANE=1" action="opt in to the embedded lane" />
              <Chip combo="terminal-browser on PATH" action="the embedded lane's binary" />
              <Chip combo="&lt;300ms exit" action="falls back to text, URL kept" />
            </div>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 py-20">
            <SectionTag>URL policy</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              Localhost always. https anywhere. Plain http by flag.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              The fetch posture is narrow about plain http and nothing else.{' '}
              <Code>file://</Code> URLs and bare paths read off disk. Localhost —{' '}
              <Code>localhost</Code>, <Code>127.0.0.1</Code>, <Code>::1</Code> — is
              always allowed on either scheme, and <Code>https://</Code> opens any
              host by default. Only plain <Code>http://</Code> beyond localhost asks
              for an explicit unlock: export{' '}
              <Code>THEFLOOR_BROWSER_ALLOW_HTTP=1</Code> — read at use time,
              with no config schema and no brain.json key. A blocked fetch says so in
              a dim row and tells you the exact flag.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Chip combo="file:// · bare paths" action="read straight off disk" />
              <Chip combo="localhost / 127.0.0.1 / ::1" action="always allowed, either scheme" />
              <Chip combo="https://" action="any host by default" />
              <Chip combo="THEFLOOR_BROWSER_ALLOW_HTTP=1" action="unlocks outbound http" />
            </div>
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
                <strong className="text-foreground">The text lane is text.</strong>{' '}
                No CSS layout, no JavaScript — pages arrive as readable rows. Sites
                that exist only as a script bundle have nothing to render.
              </li>
              <li>
                <strong className="text-foreground">Image bytes are never
                fetched.</strong> Images render as <Code>🖼 &lt;alt&gt;</Code> chips;
                the viewer reads markup, not media.
              </li>
              <li>
                <strong className="text-foreground">HTML only.</strong> PDFs, images
                and other payloads land the dim <Code>unsupported content type</Code>{' '}
                row instead of a parse.
              </li>
              <li>
                <strong className="text-foreground">One key route in.</strong> The
                browser isn&apos;t part of the sidebar&apos;s tab cycle and has no
                digit key — <Code>ctrl+b</Code> on the left pane is the only way in,{' '}
                <Code>q</Code> / <Code>esc</Code> the way back to the floor.
              </li>
              <li>
                <strong className="text-foreground">The premium lane is kitty/ghostty
                only.</strong> tmux and the iTerm2 family stay on the text lane — a
                protocol boundary, not a preference.
              </li>
              <li>
                <strong className="text-foreground">The embedded lane is off by
                default.</strong> The zenbu <Code>terminal-browser</Code> embed is
                opt-in (<Code>THEFLOOR_ZENBU_LANE=1</Code> + the binary on{' '}
                <Code>PATH</Code>); the default premium path is headless
                screenshots.
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
                  href: '/docs/keys-and-slash',
                  title: 'Keys & slash commands',
                  body: '/open, o, and the rest of the key table, in full.',
                },
                {
                  href: '/docs/terminal-and-git-tabs',
                  title: 'Terminal & git tabs',
                  body: 'The sidebar tabs to the browser\u2019s right: shell and repo.',
                },
                {
                  href: '/docs/getting-started',
                  title: 'Getting started',
                  body: 'Install, first boot, first dispatch.',
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
