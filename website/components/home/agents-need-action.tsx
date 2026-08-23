import { ScrollReveal } from "@/components/scroll-reveal";

const roster = ["tekton", "skopos", "dikastes", "hemerodromos"];

export function AgentsNeedAction() {
  return (
    <section
      id="solutions"
      className="relative overflow-hidden border-b border-border bg-background"
    >
      <div className="relative mx-auto max-w-7xl px-6 py-14">
        <p className="text-center font-mono text-xs uppercase tracking-wider text-muted-foreground">
          A team that stays visible
        </p>
        <h2 className="mx-auto mt-4 max-w-2xl text-center text-3xl font-semibold tracking-tight md:text-4xl">
          More hands on the work. One clear place to manage it.
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-center text-sm leading-relaxed text-muted-foreground">
          Give the busywork to agents without trading away context. You stay
          close to the decisions, not the terminal scrollback.
        </p>

        <div className="mx-auto my-10 aspect-video w-full max-w-5xl overflow-hidden border border-border bg-black">
          <iframe
            className="h-full w-full"
            src="https://www.youtube.com/embed/6SmqEydHsFQ?autoplay=1&mute=1&playsinline=1&rel=0&modestbranding=1"
            title="Agents need action demo"
            loading="lazy"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            allowFullScreen
          />
        </div>

        <ScrollReveal
          stagger={0.08}
          className="mx-auto mt-8 flex max-w-4xl flex-wrap items-center justify-center gap-x-14 gap-y-5 text-muted-foreground sm:justify-between"
        >
          {roster.map((l) => (
            <span
              key={l}
              className="font-mono text-2xl font-semibold tracking-tight opacity-70 md:text-3xl"
            >
              {l}
            </span>
          ))}
        </ScrollReveal>
      </div>

      <div className="relative border-t border-border">
        <div className="mx-auto grid max-w-7xl grid-cols-1 gap-6 px-6 py-14 lg:grid-cols-3">
          <ScrollReveal
            direction="left"
            className="border border-border bg-card p-5 font-mono text-xs"
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

          <ScrollReveal
            delay={0.1}
            className="flex flex-col justify-center border border-border bg-card p-6"
          >
            <p className="font-mono text-xs text-muted-foreground">
              boss (oikonomos)
            </p>
            <p className="mt-4 text-sm leading-relaxed">
              Ship the auth fix, get dikastes to review it, and post a digest to
              #eng when it&apos;s green.
            </p>
            <p className="mt-6 font-mono text-xs text-accent">Work Thread ›</p>
            <p className="mt-2 font-mono text-xs text-muted-foreground">
              Live work thread
            </p>
          </ScrollReveal>

          <ScrollReveal
            direction="right"
            className="border border-border bg-card p-5 font-mono text-xs"
          >
            <p className="text-muted-foreground">AGENTMEMORY_SIGNALS</p>
            <p className="mt-3 text-muted-foreground">
              BOSS_SESSION: oikonomos-04
            </p>
            <div className="mt-3 flex items-center justify-between border border-border px-3 py-2">
              <span>deploy.sh — reviewed</span>
              <span className="text-chart-2">● delivered</span>
            </div>
            <p className="mt-4 text-muted-foreground">BOARD_ACTION</p>
            <p className="mt-3 border border-border px-3 py-2">
              TICKET_OPENED — flaky-test-482
            </p>
          </ScrollReveal>
        </div>
      </div>
    </section>
  );
}
