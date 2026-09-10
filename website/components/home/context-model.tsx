import Link from 'next/link'
import { ScrollReveal } from '@/components/scroll-reveal'
import { Beat, Chapter, Cue, Panel, Sheet, Stamp } from '@/components/paper'

const backends: [string, string][] = [
  ['OpenCode', 'Start a server, or attach to one you already run.'],
  ['Claude Code', 'Uses your installed Claude CLI and its login.'],
  ['Codex', 'Runs on your Codex CLI login and model settings.'],
]

export function ContextModel() {
  return (
    <Sheet className="relative border-t border-rule">
      <div className="mx-auto max-w-7xl px-6 py-20 md:px-10">
        <Chapter
          numeral="III."
          title="Under the floor"
          id="chapter-iii"
          lede={
            <>
              An office you can open, read, and put back together.
            </>
          }
        >
          <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-ink-soft">
            The half of the document that deals in mechanics: what the floor keeps,
            what it runs on, and who has their name on the commits.
          </p>
        </Chapter>

        <Beat
          index="3.1"
          label="Keep the context"
          title="Pick up where the team left off."
          className="mt-20"
        >
          <div className="grid gap-12 md:grid-cols-2 md:gap-14">
            <ScrollReveal className="min-w-0">
              <Stamp>Project memory</Stamp>
              <p className="mt-5 max-w-lg text-pretty text-sm leading-relaxed text-ink-soft">
                Teams and tickets belong to the project, not to the session you opened
                them in. Every conversation keeps its own backend and its own history.
                Start something new; nothing before it gets swept off the desk.
              </p>
              <Panel label="On disk" className="mt-7 font-mono text-xs leading-loose">
                <p>
                  floor.json <span className="text-ink-faint">→ teams + tickets</span>
                </p>
                <p>
                  conversations/ <span className="text-ink-faint">→ saved sessions</span>
                </p>
                <p>
                  Ctrl+R <span className="text-ink-faint">→ search loaded transcript</span>
                </p>
              </Panel>
              <p className="mt-5 max-w-lg text-pretty text-sm leading-relaxed text-ink-soft">
                Local conversation archives hold up to 10,000 messages. A 200-message
                snapshot keeps the next startup quick. Connect agentmemory and you get a
                separate ledger of decisions and tasks alongside it.
              </p>
              <Link
                href="/docs/workspaces#history"
                className="mono-label mt-7 inline-block text-ink underline underline-offset-4"
              >
                How history works
              </Link>
            </ScrollReveal>

            <ScrollReveal className="min-w-0">
              <Stamp tone="blue">Your choice of backend</Stamp>
              <h3 className="display-md mt-5 text-ink">
                Different agents. The same workspace.
              </h3>
              <div className="doc-brackets hairline mt-7 divide-y divide-rule bg-paper-2">
                {backends.map(([name, text]) => (
                  <div key={name} className="p-5">
                    <p className="font-mono text-sm text-ink">{name}</p>
                    <p className="mt-2 text-sm text-ink-soft">{text}</p>
                  </div>
                ))}
              </div>
              <p className="mt-5 max-w-lg text-pretty text-sm leading-relaxed text-ink-soft">
                Ctrl+N picks the backend, the team, and the title for a new conversation.
                Each one resumes on the backend that started it.
              </p>
              <Cue className="mt-4 block">no re-plumbing, no second workspace</Cue>
              <Link
                href="/docs/backends"
                className="mono-label mt-7 inline-block text-ink underline underline-offset-4"
              >
                Compare backend support
              </Link>
            </ScrollReveal>
          </div>
        </Beat>
      </div>
    </Sheet>
  )
}
