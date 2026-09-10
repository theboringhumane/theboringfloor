import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Chapter, Cue, Panel, Sheet, Stamp } from '@/components/paper'

const roster = ['tekton', 'skopos', 'dikastes', 'hemerodromos']

export function AgentsNeedAction() {
  return (
    <Sheet id="solutions" className="overflow-hidden border-t border-rule">

      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10 lg:px-14">
        <Chapter
          numeral="II."
          title="THE WORK"
          id="chapter-ii"
          lede={
            <>
              More hands on the work. One floor where all of it happens.
            </>
          }
        >
          <p className="mt-6 max-w-xl text-pretty text-sm leading-relaxed text-ink-soft">
            Hand the busywork to the team without handing over the context. You
            stay close to the decisions and out of the scrollback.
          </p>
        </Chapter>

        <Beat
          index="2.1"
          label="Put the crew on it"
          className="mt-16"
          title="A team that works in the open."
        >
          <ScrollReveal
            stagger={0.08}
            className="mt-2 flex flex-wrap items-center gap-x-10 gap-y-4"
          >
            {roster.map((l) => (
              <span
                key={l}
                className="font-mono text-xl tracking-tight text-ink-faint md:text-2xl"
              >
                {l}
              </span>
            ))}
          </ScrollReveal>
          <Cue className="mt-6 block">fig. 2.1 — the roster, clocked in</Cue>
        </Beat>
      </div>

      <div className="border-t border-rule">
        <div className="mono-label flex items-center justify-between gap-4 px-6 py-3 md:px-10 lg:px-14">
          <span>The office, from a client that isn&apos;t there</span>
          <span className="hidden sm:inline">Watch the floor</span>
        </div>
        <div className="aspect-video w-full border-t border-rule bg-(--shot-frame)">
          <iframe
            className="shot-img h-full w-full"
            src="https://www.youtube.com/embed/YNrL5NMUvsA?autoplay=1&mute=1&playsinline=1&rel=0&modestbranding=1"
            title="Give every project its own floor."
            loading="lazy"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            allowFullScreen
          />
        </div>
      </div>

      <div className="border-t border-rule">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-6 px-6 py-14 md:px-10 lg:grid-cols-3 lg:px-14">
          <ScrollReveal direction="left">
            <Panel label="Permission queue — 1 of 3" className="h-full">
              <div className="hairline flex items-center gap-2 px-3 py-2 font-mono text-xs text-ink">
                chmod +x scripts/deploy.sh
                <span className="ml-auto text-ink-faint">tekton-02</span>
              </div>
              <div className="mt-3 space-y-2 font-mono text-xs">
                <p className="hairline px-3 py-2 text-ink">
                  ALLOW ONCE <span className="text-ink-faint">· y</span>
                  <span className="mt-1 block text-ink-soft">
                    Run this action, ask again next time
                  </span>
                </p>
                <p className="hairline px-3 py-2 text-ink">
                  ALLOW ALWAYS <span className="text-ink-faint">· a</span>
                  <span className="mt-1 block text-ink-soft">
                    Never ask for this action again
                  </span>
                </p>
              </div>
            </Panel>
          </ScrollReveal>

          <ScrollReveal delay={0.1}>
            <Panel label="boss (oikonomos)" className="flex h-full flex-col justify-center">
              <p className="text-pretty text-sm leading-relaxed text-ink">
                Ship the auth fix, get dikastes to review it, and post a digest
                to #eng when it&apos;s green.
              </p>
              <div className="mt-6 flex items-center gap-3">
                <Stamp tone="blue">Work thread ›</Stamp>
                <span className="mono-label">Live work thread</span>
              </div>
            </Panel>
          </ScrollReveal>

          <ScrollReveal direction="right">
            <Panel label="agentmemory_signals" className="h-full font-mono text-xs">
              <p className="text-ink-soft">BOSS_SESSION: oikonomos-04</p>
              <div className="hairline mt-3 flex items-center justify-between px-3 py-2 text-ink">
                <span>deploy.sh — reviewed</span>
                <span className="text-blue">● delivered</span>
              </div>
              <p className="mono-label mt-5">Board action</p>
              <p className="hairline mt-3 px-3 py-2 text-ink">
                TICKET_OPENED — flaky-test-482
              </p>
            </Panel>
          </ScrollReveal>
        </div>
      </div>
    </Sheet>
  )
}
