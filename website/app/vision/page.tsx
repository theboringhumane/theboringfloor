import Link from "next/link";
import type { Metadata } from "next";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { Chapter, Cue, Stamp } from "@/components/paper";
import { PageCover } from "@/components/page-cover";

export const metadata: Metadata = {
  title: "Vision | theboringfloor",
  description:
    "Why we believe agents need a floor, a roster, and memory — because a prompt box shows you a spinner, not the work.",
};

const coworkers = [
  {
    id: "01",
    name: "the manager",
    role: "The boss",
    body: "The one you talk to. Dispatches work to the team, keeps the shift coherent, and takes your messages mid-turn — interruptible, batchable, always reachable.",
  },
  {
    id: "02",
    name: "theboringcto",
    role: "The CTO",
    body: "Reviews every drained batch before it lands. The adult in the room for the calls that matter.",
  },
  {
    id: "03",
    name: "hr",
    role: "People & health",
    body: "Watches team health and how the work feels on the floor — even a team of agents needs that.",
  },
  {
    id: "04",
    name: "tekton",
    role: "The developers",
    body: "Claims tickets off the board, opens work threads you can expand right in chat, and walks to the tea machine between batches.",
  },
  {
    id: "05",
    name: "scouts, reviewers, runners",
    role: "The supporting cast",
    body: "Scouts recon the codebase before work starts, reviewers weigh in on every return, and runners carry the work until it is done.",
  },
  {
    id: "06",
    name: "the floor itself",
    role: "Ambient life",
    body: "Desks, blinking server LEDs, the tea machine, light and dark themes, and sound. An office that feels inhabited, not rendered.",
  },
];

const etiquette = [
  "Message the boss mid-turn — your words batch into one composed send, not a lost line in the scrollback.",
  "The concierge answers instantly while the boss is busy, so you never talk to a wall.",
  "/stop aborts on the spot; free-send keeps you typing while the office is mid-turn.",
  "Permission asks stack as 1 of N instead of stealing the screen — approve, always-allow, or reject.",
];

const ceilings = [
  "You want one-shot answers, not an ongoing system. A plain chat window is fine for that — use it.",
  "You live in an IDE GUI, not a terminal. The office is a terminal app, and it will not pretend otherwise.",
  "You need deterministic structured-output pipelines, not coworkers you supervise.",
  "You have one agent, one file, one question — a roster and a board are overhead there.",
  "You would rather not supervise anything. A floor you can read assumes someone reading it.",
];

/** A numbered ledger row — the essay's lists print as rows, not cards. */
function Row({
  index,
  children,
}: {
  index: string;
  children: React.ReactNode;
}) {
  return (
    <li className="flex gap-4 py-4 md:gap-6">
      <span className="mono-label mt-1 shrink-0 text-ink-faint">{index}</span>
      <div className="min-w-0">{children}</div>
    </li>
  );
}

export default function VisionPage() {
  return (
    <div className="paper-ground min-h-svh">
      <div className="doc-frame">
        <SiteHeader framed />
        <main>
          <PageCover
            eyebrow="OUR VISION / BETTER TOGETHER"
            title={
              <>
                The prompt box
                <br />
                is a lonely
                <br />
                <span className="blue-text">place to work.</span>
              </>
            }
            description="We believe working with agents should feel like working with a team: shared context, visible progress, and a place for everyone."
            art="network"
          />

          {/* I — the problem */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="I" title="Diagnose the spinner">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                A spinner is not the work.
              </h3>
              <div className="mt-6 max-w-2xl space-y-4 text-pretty leading-relaxed text-ink-soft">
                <p>
                  Agent work today is mechanical and transactional. You type, a
                  spinner turns, a wall of text comes back. Run two sessions and
                  it gets worse: three panes, two of them waiting on you, and no
                  idea which one owns the auth change. You can&apos;t see the
                  work move, you can&apos;t ask the next thing while it thinks,
                  and when the window closes, everything it learned closes with
                  it.
                </p>
                <p>
                  That changes how you use the tool. You stop collaborating and
                  start queuing requests. The agent never becomes more than an
                  API call — not because it can&apos;t, but because there is
                  nowhere for the work to stand still and be seen.
                </p>
              </div>
            </Chapter>
          </section>

          {/* II — the bet */}
          <section className="paper-panel relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="II" title="State the bet">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                Familiarity builds trust. Trust builds collaboration.
              </h3>
              <div className="mt-6 max-w-2xl space-y-4 text-pretty leading-relaxed text-ink-soft">
                <p>
                  The office is one of the oldest shared-work interfaces we
                  have. Desks you can read at a glance. A board with the work on
                  the wall. Someone to ask. Someone keeping an eye on the team.
                  Nobody needs onboarding to a desk.
                </p>
                <p>
                  Put an agent at one of those desks and something quiet
                  happens. It walks to the whiteboard when the problem is big.
                  It refills the tea machine between batches. You stop parsing
                  logs and start reading a room — and you hand over more of the
                  work, because you can finally see the work.
                </p>
                <p>
                  That is the bet: anthropomorphizing the floor is not
                  decoration, it is interface design. A system you can see is a
                  system you can supervise.
                </p>
              </div>
              <Cue className="mt-8 block">no onboarding required</Cue>
            </Chapter>
          </section>

          {/* III — the roster */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="III" title="Read the roster">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                A floor of coworkers, not cursors.
              </h3>
              <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-ink-soft">
                The floor runs on a roster, not a loop. Every worker has a name,
                a desk, and a status line that says what it is doing right now.
              </p>
              <ol className="mt-10 divide-y divide-rule border-y border-rule">
                {coworkers.map((c) => (
                  <Row key={c.id} index={c.id}>
                    <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
                      <h3 className="font-mono text-sm uppercase tracking-[0.14em] text-ink">
                        {c.name}
                      </h3>
                      <span className="mono-label text-stamp">{c.role}</span>
                    </div>
                    <p className="mt-2 max-w-2xl text-pretty text-sm leading-relaxed text-ink-soft">
                      {c.body}
                    </p>
                  </Row>
                ))}
              </ol>
            </Chapter>
          </section>

          {/* IV — etiquette */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="IV" title="Interrupt the boss">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                A queue that never locks you out.
              </h3>
              <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-ink-soft">
                A good boss is interruptible. Message the office mid-task,
                mid-batch, mid-turn — nothing you send falls into a black hole.
              </p>
              <ol className="mt-10 max-w-3xl divide-y divide-rule border-y border-rule">
                {etiquette.map((line, i) => (
                  <Row key={line} index={`4.${i + 1}`}>
                    <p className="text-pretty text-sm leading-relaxed text-ink-soft md:text-base">
                      {line}
                    </p>
                  </Row>
                ))}
              </ol>
            </Chapter>
          </section>

          {/* V — memory */}
          <section className="paper-panel relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="V" title="Keep the shift">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                The office remembers, so you don&apos;t have to re-explain.
              </h3>
              <div className="mt-6 max-w-2xl space-y-4 text-pretty leading-relaxed text-ink-soft">
                <p>
                  A coworker who forgets everything overnight is not much of a
                  coworker. The office restores your last session when you open
                  it, <span className="font-mono text-ink">-s</span> /{" "}
                  <span className="font-mono text-ink">--session</span> pins a
                  thread, and{" "}
                  <span className="font-mono text-ink">/session</span> picks the
                  room you want to walk back into.
                </p>
                <p>
                  Under the floor, agentmemory keeps the lessons and the board
                  state across shifts. Close the terminal on Friday; walk back
                  in on Tuesday and the office is where you left it.
                </p>
              </div>
            </Chapter>
          </section>

          {/* VI — the horizon */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="VI" title="Pick no favourite harness">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                Built for coworkers, not one harness.
              </h3>
              <div className="mt-6 max-w-2xl space-y-4 text-pretty leading-relaxed text-ink-soft">
                <p>
                  The office runs on opencode or Claude Code, honestly and
                  visibly — same floor, same roster, either harness. The org,
                  the floor, and the memory were built for coworkers, whichever
                  harness they arrive in.
                </p>
                <p>
                  We are at v0.2.x — the &quot;wave&quot; releases. Early, open,
                  and building in public. More seats, more office.
                </p>
              </div>
            </Chapter>
          </section>

          {/* VII — ceilings */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="VII" title="Admit the ceilings">
              <h3 className="display-lg mt-8 max-w-3xl text-balance text-ink">
                When the office is the wrong tool.
              </h3>
              <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-ink-soft">
                We would rather lose you here than at hour ten of the wrong
                setup. Skip the whole thing if:
              </p>
              <ol className="mt-10 max-w-3xl divide-y divide-rule border-y border-rule">
                {ceilings.map((line, i) => (
                  <Row key={line} index={`7.${i + 1}`}>
                    <p className="text-pretty text-sm leading-relaxed text-ink-soft md:text-base">
                      {line}
                    </p>
                  </Row>
                ))}
              </ol>
            </Chapter>
          </section>

          {/* VIII — the close */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="VIII" title="Walk the floor">
              <h3 className="display-lg mt-8 max-w-2xl text-balance text-ink">
                A proper office, for agents.
              </h3>
              <p className="mt-6 max-w-xl text-pretty leading-relaxed text-ink-soft">
                Tuesday at a good company: a floor with a hum to it, coworkers
                with names, work you can see moving, a boss you can interrupt,
                and a team that remembers. That is the whole idea — come walk
                the floor.
              </p>
              <div className="mt-8 flex flex-wrap items-center gap-3">
                <Link
                  href="/get-started"
                  className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                >
                  <Stamp className="transition-colors hover:border-ink hover:text-ink">
                    Get started
                  </Stamp>
                </Link>
                <Link
                  href="/blog"
                  className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                >
                  <Stamp className="transition-colors hover:border-ink hover:text-ink">
                    Read the blog
                  </Stamp>
                </Link>
              </div>
            </Chapter>
          </section>
        </main>
        <SiteFooter />
      </div>
    </div>
  );
}
