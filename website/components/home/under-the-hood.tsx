import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Panel, Sheet, Stamp } from '@/components/paper'

const cards = [
  {
    id: '01',
    title: 'One file on disk',
    body: 'Written in Go, shipped as a single static binary. No Electron, no Chromium, no runtime to install. The whole office is one file, and you can delete it.',
  },
  {
    id: '02',
    title: 'The floor moves when work moves',
    body: "Sprites walk when the LLM transport says something happened — the opencode serve event stream, or a stream-json feed from your claude CLI. Never on animation timers. Coffee steam and rack LEDs tick gently in the background, so an idle office stays cheap.",
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
    title: 'Built-in browsing',
    body: 'The built-in browser renders pages as text and link rows. With Chrome installed, kitty-capable terminals can also show headless screenshots. External links open in your system browser; no terminal-browser package required.',
  },
]

const metrics = [
  { value: '−25%', note: 'chat render hot path (4.07s → 3.06s)' },
  { value: '−61%', note: 'sampled CPU in the tick profile' },
  { value: '0', note: 'forced frames per second at idle' },
  { value: '−28%', note: 'per-delta inbox ingestion (74.5s → 53.2s)' },
]

export function UnderTheHood() {
  return (
    <Sheet id="under-the-hood" className="relative border-t border-rule">
      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10">
        <Beat index="3.2" label="Open the hood" title="Kind to your machine.">
          <p className="max-w-2xl text-pretty leading-relaxed text-ink-soft">
            A living office sounds expensive. It isn&apos;t. This is a terminal app that
            respects the box it runs on — the boring parts are sweated carefully so the
            agents can be the interesting ones.
          </p>

          <div className="mt-9 flex flex-wrap items-center gap-2">
            {metrics.map((m) => (
              <Stamp key={m.note} tone="stamp" className="normal-case tracking-normal">
                <span className="tracking-[0.14em]">{m.value}</span>
                <span className="ml-1.5 text-ink-faint">{m.note}</span>
              </Stamp>
            ))}
          </div>
          <p className="mono-label mt-3 text-ink-faint">
            measured on v0.2.11, this machine, full-suite runs
          </p>

          <ScrollReveal
            stagger={0.06}
            className="mt-14 grid grid-cols-1 gap-px border border-rule bg-rule md:grid-cols-2 lg:grid-cols-3"
          >
            {cards.map((c) => (
              <div key={c.id} className="flex flex-col gap-4 bg-paper p-8">
                <span className="font-mono text-xs text-ink-faint">{c.id}</span>
                <h3 className="text-xl font-medium tracking-tight text-ink">{c.title}</h3>
                <p className="text-pretty text-sm leading-relaxed text-ink-soft">{c.body}</p>
              </div>
            ))}
          </ScrollReveal>

          <Panel label="install.sh" className="mt-12">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <p className="overflow-x-auto font-mono text-xs leading-relaxed text-ink">
                <span className="text-ink-faint">$ </span>
                curl -fsSL https://boringfloor.com/install.sh | sh
              </p>
              <p className="mono-label shrink-0">
                macOS · Linux · Windows · amd64 · arm64
              </p>
            </div>
            <p className="mt-5 border-t border-rule pt-4 font-mono text-xs text-ink-faint">
              Cross-compiled by GoReleaser, tagged publicly. House rule: zero new
              dependencies without need.
            </p>
          </Panel>
        </Beat>
      </div>
    </Sheet>
  )
}
