import { FolderGit2 } from 'lucide-react'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'

const DISCORD_INVITE = 'https://discord.gg/YPDsHVHTVf'
const GITHUB_REPO = 'https://github.com/theboringhumane/theboringoffice'

const discordLog = [
  { who: 'theboringhumane', text: 'wave 34 is open — the floor needs a doorbell' },
  { who: 'skopos-02', text: 'issue #112 filed — doorbell spec, one page' },
  { who: 'tekton-01', text: 'claimed. PR up within the hour' },
  { who: 'dikastes-01', text: 'reviewed, merged. welcome to the commit log' },
]

const waysIn = [
  {
    id: '01',
    title: 'Lurk productively',
    body: 'The Discord is where waves get argued about before they ship. The channel history is the design doc — read as far back as you like.',
  },
  {
    id: '02',
    title: 'File the boring bugs',
    body: 'If something in the office is boring in the wrong way, open an issue. Reproduction steps are the love language around here.',
  },
  {
    id: '03',
    title: 'Ship a wave',
    body: "Pick an issue, open a pull request, and your commit lands in the same public log as everyone else's. Reviews are direct; merges are quick.",
  },
]

function DiscordIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
      className={className}
    >
      <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" />
    </svg>
  )
}

export function Community() {
  return (
    <section id="community" className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-20">
        <SectionTag>Join the community</SectionTag>
        <h2 className="mt-6 max-w-2xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-5xl">
          The office is open. Build this together.
        </h2>
        <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
          theboringoffice grows in public waves, and the whole crew — the maintainer, the
          regulars, and whoever walked in five minutes ago — argues about the next one in
          the Discord. Come watch the waves land, tell us which part is boring in the wrong
          way, or pick up an issue and ship a wave yourself. Contributors welcome; the
          commit log has room.
        </p>

        <div className="mt-14 grid grid-cols-1 gap-10 lg:grid-cols-2">
          <ScrollReveal direction="left" className="flex flex-col gap-5">
            <p className="max-w-lg text-pretty text-sm leading-relaxed text-muted-foreground">
              No invite required, no application form. The Discord is where designs get
              argued about before they ship, and the GitHub log is where the arguments
              settle. Lurk as long as you like — the channel is the office hours, and the
              office never closes.
            </p>
            <p className="max-w-lg text-pretty text-sm leading-relaxed text-muted-foreground">
              When you&apos;re ready to do more than watch, the issue tracker is the work
              board. Claim one, open a pull request, and your handle lands in the same
              public log as everyone else&apos;s.
            </p>
            <div className="mt-2 flex flex-wrap gap-3">
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 bg-foreground px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
              >
                <DiscordIcon className="size-3.5" />
                Join the Discord
              </a>
              <a
                href={GITHUB_REPO}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-2 border border-border px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:bg-secondary"
              >
                <FolderGit2 className="size-3.5" aria-hidden />
                Contribute on GitHub
              </a>
            </div>
          </ScrollReveal>

          <ScrollReveal direction="right" className="border border-border bg-card">
            <div className="flex items-center gap-2 border-b border-border px-4 py-2.5">
              <span className="size-2.5 rounded-full bg-destructive/70" />
              <span className="size-2.5 rounded-full bg-chart-4/70" />
              <span className="size-2.5 rounded-full bg-chart-2/70" />
              <span className="ml-2 font-mono text-xs text-muted-foreground">
                #the-floor — discord
              </span>
            </div>
            <div className="flex flex-col gap-2.5 px-5 py-5 font-mono text-xs leading-relaxed">
              {discordLog.map((l) => (
                <p key={l.who}>
                  <span className="text-accent">{l.who}</span>{' '}
                  <span className="text-foreground/80">{l.text}</span>
                </p>
              ))}
              <p className="mt-2 text-muted-foreground">
                … and the next wave is already being argued about.
              </p>
            </div>
          </ScrollReveal>
        </div>

        <ScrollReveal
          stagger={0.06}
          className="mt-12 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-3"
        >
          {waysIn.map((c) => (
            <div key={c.id} className="flex flex-col gap-4 bg-background p-8">
              <span className="font-mono text-xs text-muted-foreground">{c.id}</span>
              <h3 className="text-lg font-semibold tracking-tight">{c.title}</h3>
              <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
                {c.body}
              </p>
            </div>
          ))}
        </ScrollReveal>
      </div>
    </section>
  )
}
