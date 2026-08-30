import { ScrollReveal } from '@/components/scroll-reveal'

const roster = ['tekton', 'skopos', 'dikastes', 'hemerodromos']

export function AgentsNeedAction() {
  return (
    <section
      id="solutions"
      className="relative overflow-hidden border-b border-border bg-background"
    >
      <div className="relative px-6 py-14 md:px-10 lg:px-14">
        <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
          A team that works in the open
        </p>
        <h2 className="mt-4 max-w-2xl text-3xl font-semibold tracking-tight md:text-4xl">
          More hands on the work. One floor where it all happens.
        </h2>
        <p className="mt-4 max-w-xl text-sm leading-relaxed text-muted-foreground">
          Hand the busywork to the team without handing over the context. You stay close to the
          decisions — and out of the scrollback.
        </p>

        <ScrollReveal
          stagger={0.08}
          className="mt-10 flex flex-wrap items-center gap-x-10 gap-y-4 text-muted-foreground"
        >
          {roster.map((l) => (
            <span
              key={l}
              className="font-mono text-xl font-semibold tracking-tight opacity-70 md:text-2xl"
            >
              {l}
            </span>
          ))}
        </ScrollReveal>
      </div>

      <div className="border-t border-border">
        <div className="flex items-center justify-between gap-4 px-6 py-3 font-mono text-[10px] uppercase tracking-[0.16em] text-muted-foreground md:px-10 lg:px-14">
          <span>The office, from a client that isn&apos;t there</span>
          <span className="hidden sm:inline">Watch the floor</span>
        </div>
        <div className="aspect-video w-full border-t border-border bg-(--shot-frame)">
          <iframe
            className="shot-img h-full w-full"
            src="https://www.youtube.com/embed/6SmqEydHsFQ?autoplay=1&mute=1&playsinline=1&rel=0&modestbranding=1"
            title="Agents need action demo"
            loading="lazy"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            allowFullScreen
          />
        </div>
      </div>

      <div className="relative border-t border-border">
        <div className="grid grid-cols-1 lg:grid-cols-3">
          <ScrollReveal
            direction="left"
            className="border-b border-border bg-card p-5 font-mono text-xs lg:border-b-0 lg:border-r"
          >
            <p className="text-muted-foreground">PERMISSION_QUEUE — 1 of 3</p>
            <div className="mt-3 flex items-center gap-2 border border-border px-3 py-2 text-foreground/80">
              chmod +x scripts/deploy.sh
              <span className="ml-auto text-muted-foreground">tekton-02</span>
            </div>
            <div className="mt-3 space-y-2">
              <p className="border border-border px-3 py-2">
                ALLOW ONCE <span className="text-muted-foreground">· y</span>
                <span className="ml-2 block text-muted-foreground">
                  Run this action, ask again next time
                </span>
              </p>
              <p className="border border-border px-3 py-2">
                ALLOW ALWAYS <span className="text-muted-foreground">· a</span>
                <span className="ml-2 block text-muted-foreground">
                  Never ask for this action again
                </span>
              </p>
            </div>
          </ScrollReveal>

          <ScrollReveal delay={0.1} className="flex flex-col justify-center border-b border-border bg-card p-6 lg:border-b-0 lg:border-r">
            <p className="font-mono text-xs text-muted-foreground">boss (oikonomos)</p>
            <p className="mt-4 text-sm leading-relaxed">
              Ship the auth fix, get dikastes to review it, and post a digest to #eng when
              it&apos;s green.
            </p>
            <p className="mt-6 font-mono text-xs text-accent">Work Thread ›</p>
            <p className="mt-2 font-mono text-xs text-muted-foreground">Live work thread</p>
          </ScrollReveal>

          <ScrollReveal
            direction="right"
            className="bg-card p-5 font-mono text-xs"
          >
            <p className="text-muted-foreground">AGENTMEMORY_SIGNALS</p>
            <p className="mt-3 text-muted-foreground">BOSS_SESSION: oikonomos-04</p>
            <div className="mt-3 flex items-center justify-between border border-border px-3 py-2">
              <span>deploy.sh — reviewed</span>
              <span className="text-chart-2">● delivered</span>
            </div>
            <p className="mt-4 text-muted-foreground">BOARD_ACTION</p>
            <p className="mt-3 border border-border px-3 py-2">TICKET_OPENED — flaky-test-482</p>
          </ScrollReveal>
        </div>
      </div>
    </section>
  )
}
