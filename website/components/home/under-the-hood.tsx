import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const cards = [
  {
    id: '01',
    title: 'One file on disk',
    body: 'Written in Go and shipped as a single static binary. No Electron, no Chromium, no runtime to install. The whole office is one file you can delete.',
  },
  {
    id: '02',
    title: 'The floor moves when work moves',
    body: "Sprites walk when the LLM transport says something happened — the opencode serve event stream, or a stream-json feed from your claude CLI — never on animation timers. Coffee steam and rack LEDs tick gently in the background, so an idle office stays cheap.",
  },
  {
    id: '03',
    title: 'A power governor, built in',
    body: '/power auto, saver, or performance. Saver slows the tick and coalesces renders; performance keeps the floor crisp. Your battery gets a say.',
  },
  {
    id: '04',
    title: 'Polling with manners',
    body: 'When nothing changes, quiet polls stretch out. The moment work starts, they snap back. A calm board costs next to nothing.',
  },
  {
    id: '05',
    title: 'Sound, synthesized',
    body: 'UI sounds are synthesized in pure Go at runtime — no audio files shipped. Platform player, terminal bell, or silence.',
  },
  {
    id: '06',
    title: 'Tested by its own hands',
    body: 'The app ships its own screenshot and regression drivers — uishot and floorshot — so every wave is checked by deterministic, scripted runs before it lands.',
  },
  {
    id: '07',
    title: 'A browser, no browser engine',
    body: 'The browser tab renders pages as text and link rows on any terminal — no headless Chromium, no runtime. On kitty-capable terminals an installed zenbu terminal-browser embeds the real page in the pane; https pages open by default; plain http beyond localhost stays off unless you allow it.',
  },
]

export function UnderTheHood() {
  return (
    <section id="under-the-hood" className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <SectionTag>Under the hood</SectionTag>
        <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
          Kind to your machine.
        </h2>
        <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
          A living office sounds expensive. It isn&apos;t. This is a terminal app that respects
          the box it runs on — we sweat the boring parts carefully so the agents can be the
          interesting ones.
        </p>

        <div className="mt-8 flex flex-wrap items-center gap-2 font-mono text-xs">
          <span className="border border-border bg-card px-3 py-1.5 text-accent">
            −25%
            <span className="ml-2 text-muted-foreground">chat render hot path (4.07s → 3.06s)</span>
          </span>
          <span className="border border-border bg-card px-3 py-1.5 text-accent">
            −61%
            <span className="ml-2 text-muted-foreground">sampled CPU in the tick profile</span>
          </span>
          <span className="border border-border bg-card px-3 py-1.5 text-accent">
            0
            <span className="ml-2 text-muted-foreground">forced frames per second at idle</span>
          </span>
          <span className="border border-border bg-card px-3 py-1.5 text-accent">
            −28%
            <span className="ml-2 text-muted-foreground">per-delta inbox ingestion (74.5s → 53.2s)</span>
          </span>
        </div>
        <p className="mt-3 font-mono text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
          measured on v0.2.11, this machine, full-suite runs
        </p>

        <ScrollReveal
          stagger={0.06}
          className="mt-14 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2 lg:grid-cols-3"
        >
          {cards.map((c) => (
            <div key={c.id} className="flex flex-col gap-4 bg-background p-8">
              <span className="font-mono text-xs text-muted-foreground">{c.id}</span>
              <h3 className="text-xl font-semibold tracking-tight">{c.title}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                {c.body}
              </p>
            </div>
          ))}
        </ScrollReveal>

        <div className="mt-12 border border-border bg-card">
          <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
            <span className="size-2.5 rounded-full bg-destructive/70" />
            <span className="size-2.5 rounded-full bg-chart-4/70" />
            <span className="size-2.5 rounded-full bg-chart-2/70" />
            <span className="ml-2 font-mono text-xs text-muted-foreground">install.sh</span>
          </div>
          <div className="flex flex-col gap-4 px-6 py-6 md:flex-row md:items-center md:justify-between">
            <p className="overflow-x-auto font-mono text-xs leading-relaxed text-foreground">
              <span className="text-muted-foreground">$ </span>
              curl -fsSL https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh
            </p>
            <p className="shrink-0 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              macOS · Linux · amd64 · arm64
            </p>
          </div>
          <p className="border-t border-border px-6 py-3 font-mono text-xs text-muted-foreground">
            Cross-compiled by GoReleaser, tagged publicly. House rule: zero new dependencies
            without need.
          </p>
        </div>
      </div>
    </section>
  )
}
