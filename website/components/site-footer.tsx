import Link from "next/link";
import { Beat, Chapter } from "@/components/paper";
import {
  DISCORD_INVITE,
  GITHUB_REPO,
  LINKEDIN_URL,
  SITE_NAME,
  SITE_TAGLINE,
  X_URL,
} from "@/lib/site";

const columns = [
  {
    title: "Features",
    links: [
      "The Office",
      "Workspaces",
      "Plan Mode",
      "Work Threads",
      "Permission Queue",
      "Concierge",
      "CLI",
    ],
  },
  {
    title: "Solutions",
    links: ["Engineering Teams", "Solo Builders", "Agencies"],
  },
  {
    title: "For Agents",
    links: [
      "opencode",
      "Claude Code",
      "Codex",
      "Cursor (Coming Soon)",
      "Pi (Coming Soon)",
    ],
  },
  {
    title: "Resources",
    links: ["Docs", "Blog", "Floor Plan", "Vision", "Setup Guides", "Sounds"],
  },
];

// Real routes for the footer entries that have one; the rest render as plain text.
const linkHrefs: Record<string, string> = {
  "The Office": "/#workspaces",
  "Work Threads": "/docs/chat-and-threads",
  "Permission Queue": "/docs/permissions-and-questions",
  Concierge: "/docs/queue-board-memory",
  CLI: "/docs/getting-started",
  opencode: "/docs/backends",
  "Claude Code": "/docs/backends",
  Codex: "/docs/backends",
  Workspaces: "/docs/workspaces",
  "Plan Mode": "/docs/plan-mode",
  Docs: "/docs",
  Blog: "/blog",
  "Floor Plan": "/docs/layout-themes-power",
  Vision: "/vision",
  "Setup Guides": "/get-started",
  Sounds: "/sounds",
};

// The site is a static export (next.config.mjs: output: 'export'), so this
// module is evaluated once at build time — the stamp is the build date, not a
// per-request clock. Pinned to UTC/en-US so the string is identical on every
// machine that builds it.
const BUILD_STAMP = new Intl.DateTimeFormat("en-US", {
  year: "numeric",
  month: "short",
  day: "2-digit",
  timeZone: "UTC",
}).format(new Date());
const BUILD_YEAR = new Intl.DateTimeFormat("en-US", {
  year: "numeric",
  timeZone: "UTC",
}).format(new Date());

const colophon = [
  ["Framework", "Next.js 16"],
  ["Runtime", "React 19"],
  ["Styles", "Tailwind CSS 4"],
  ["Language", "TypeScript 5.7"],
  ["Last updated", BUILD_STAMP],
];

const socials = [
  {
    href: DISCORD_INVITE,
    label: "Discord",
    path: "M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z",
  },
  {
    href: X_URL,
    label: "X",
    path: "M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z",
  },
  {
    href: GITHUB_REPO,
    label: "GitHub",
    path: "M12 .5C5.73.5.75 5.48.75 11.75c0 4.94 3.2 9.13 7.65 10.6.56.1.76-.24.76-.54 0-.27-.01-1.17-.02-2.12-3.11.67-3.77-1.32-3.77-1.32-.51-1.3-1.24-1.64-1.24-1.64-1.02-.69.08-.68.08-.68 1.12.08 1.71 1.15 1.71 1.15 1 1.71 2.62 1.22 3.26.93.1-.72.39-1.22.71-1.5-2.48-.28-5.09-1.24-5.09-5.53 0-1.22.44-2.22 1.15-3-.12-.28-.5-1.42.11-2.96 0 0 .94-.3 3.08 1.15a10.7 10.7 0 0 1 5.6 0c2.14-1.45 3.08-1.15 3.08-1.15.61 1.54.23 2.68.11 2.96.72.78 1.15 1.78 1.15 3 0 4.3-2.62 5.25-5.11 5.53.4.35.76 1.03.76 2.08 0 1.5-.01 2.71-.01 3.08 0 .3.2.65.77.54 4.44-1.48 7.64-5.66 7.64-10.6C23.25 5.48 18.27.5 12 .5z",
  },
  {
    href: LINKEDIN_URL,
    label: "LinkedIn",
    path: "M20.45 20.45h-3.56v-5.57c0-1.33-.02-3.04-1.85-3.04-1.86 0-2.14 1.45-2.14 2.94v5.67H9.34V9h3.42v1.56h.05c.48-.9 1.64-1.85 3.38-1.85 3.61 0 4.28 2.38 4.28 5.47zM5.34 7.43a2.06 2.06 0 1 1 0-4.13 2.06 2.06 0 0 1 0 4.13zM3.56 20.45h3.56V9H3.56z",
  },
];

const focusRing =
  "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-blue focus-visible:ring-offset-2 focus-visible:ring-offset-paper";

export function SiteFooter() {
  return (
    <footer className="paper-ground relative border-t border-rule">

      <div className="relative mx-auto max-w-7xl px-6 py-16 md:px-10 lg:px-14">
        {/* Closing call — the last line of the document before the imprint. */}
        <div className="doc-brackets hairline flex flex-col justify-between gap-8 bg-paper-2 p-6 md:flex-row md:items-end md:p-8">
          <h2 className="display-md max-w-lg text-balance text-ink">
            The desks are empty. Your agents are waiting.
          </h2>
          <div className="flex flex-wrap gap-3 font-mono text-[0.6875rem] uppercase tracking-[0.14em]">
            <Link
              href="/get-started"
              className={`border border-rule px-5 py-2.5 text-ink-soft transition-colors hover:border-ink-faint hover:text-ink ${focusRing}`}
            >
              Take a Tour
            </Link>
            <Link
              href="/get-started"
              className={`border border-ink bg-ink px-5 py-2.5 text-paper transition-opacity hover:opacity-90 ${focusRing}`}
            >
              Open the Office
            </Link>
            <a
              href={DISCORD_INVITE}
              target="_blank"
              rel="noreferrer"
              className={`border border-rule px-5 py-2.5 text-ink-soft transition-colors hover:border-stamp hover:text-stamp ${focusRing}`}
            >
              Join Discord
            </a>
          </div>
        </div>

        <Chapter numeral="IV." title="Colophon" className="mt-20" />

        <div className="mt-12 grid gap-14 lg:grid-cols-[minmax(0,20rem)_minmax(0,1fr)]">
          <Beat index="4.1" label="About">
            <p className="max-w-sm text-sm leading-relaxed text-ink-soft">
              {SITE_NAME} — {SITE_TAGLINE}. Open source, and boring on purpose.
            </p>
            <dl className="mt-6 flex flex-col gap-2 font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-faint">
              {colophon.map(([term, value]) => (
                <div key={term} className="flex items-baseline gap-3">
                  <dt className="w-28 shrink-0">{term}</dt>
                  <span className="h-px flex-1 bg-rule" aria-hidden="true" />
                  <dd className="text-ink-soft">{value}</dd>
                </div>
              ))}
            </dl>
          </Beat>

          <Beat index="4.2" label="Links, etc.">
            <div className="grid grid-cols-2 gap-x-8 gap-y-10 md:grid-cols-4">
              {columns.map((col) => (
                <div key={col.title}>
                  <p className="mono-label text-ink-faint">{col.title}</p>
                  <ul className="mt-4 flex flex-col gap-3">
                    {col.links.map((link) => {
                      const href = linkHrefs[link];

                      return (
                        <li key={link}>
                          {href ? (
                            <Link
                              href={href}
                              className={`font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-soft transition-colors hover:text-ink ${focusRing}`}
                            >
                              {link}
                            </Link>
                          ) : (
                            <span className="font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-faint">
                              {link}
                            </span>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </div>
              ))}
            </div>

            <div className="mt-12 flex flex-col items-start justify-between gap-4 border-t border-rule pt-6 sm:flex-row sm:items-center">
              <span className="font-mono text-[0.6875rem] uppercase tracking-[0.14em] text-ink-faint">
                © {SITE_NAME} {BUILD_YEAR} — printed on demand
              </span>
              <div className="flex items-center gap-4 text-ink-faint">
                {socials.map((social) => (
                  <a
                    key={social.label}
                    href={social.href}
                    target="_blank"
                    rel="noreferrer"
                    aria-label={social.label}
                    className={`transition-colors hover:text-ink ${focusRing}`}
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                      aria-hidden="true"
                    >
                      <path d={social.path} />
                    </svg>
                  </a>
                ))}
              </div>
            </div>
          </Beat>
        </div>
      </div>
    </footer>
  );
}
