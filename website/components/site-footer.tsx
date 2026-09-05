import Link from "next/link";
import { DISCORD_INVITE, GITHUB_REPO, LINKEDIN_URL, X_URL } from "@/lib/site";

const columns = [
  {
    title: "Features",
    links: [
      "The Office",
      "Work Threads",
      "Permission Queue",
      "Concierge",
      "CLI",
    ],
  },
  {
    title: "Solutions",
    links: [
      "Engineering Teams",
      // 'Ops & Support',
      "Solo Builders",
      "Agencies",
      // 'Late Nights',
      // 'On-Call',
      // 'Side Projects',
      // 'Founders',
    ],
  },
  {
    title: "For Agents",
    links: [
      "opencode",
      "Claude Code",
      "Codex (Coming Soon)",
      "Cursor (Coming Soon)",
      "Pi (Coming Soon)",
    ],
  },
  {
    title: "Resources",
    links: ["Docs", "Blog", "Floor Plan", "Vision", "Setup Guides", "Sounds"],
  },
  // {
  //   title: 'Company',
  //   links: ['Careers', 'Trust', 'Contact Us', 'Support', 'Terms', 'Privacy Policy'],
  // },
];

// Real routes for the footer entries that have one; the rest render as plain text.
const linkHrefs: Record<string, string> = {
  "The Office": "/#toolkits",
  "Work Threads": "/docs/chat-and-threads",
  "Permission Queue": "/docs/permissions-and-questions",
  Concierge: "/docs/queue-board-memory",
  CLI: "/docs/getting-started",
  opencode: "/docs/backends",
  "Claude Code": "/docs/backends",
  Docs: "/docs",
  Blog: "/blog",
  "Floor Plan": "/docs/layout-themes-power",
  Vision: "/vision",
  "Setup Guides": "/get-started",
  Sounds: "/sounds",
};

export function SiteFooter() {
  return (
    <>
      <div className="section-light">
        <div className="flex mx-auto max-w-7xl px-6 py-16 flex-col justify-between gap-10 border-b border-border pb-12 md:flex-row md:items-end text-black">
          <h2 className="max-w-lg text-black text-3xl font-semibold tracking-tight md:text-4xl">
            The desks are empty. Your agents are waiting.
          </h2>
          <div className="flex flex-shrink-0 gap-3">
            <Link
              href="/get-started"
              className="rounded-sm border border-border text-(--band-on-fg) px-5 py-2.5 bg-primary hover:text-black font-mono text-xs uppercase tracking-wider transition-colors hover:bg-secondary"
            >
              Take a Tour
            </Link>
            <Link
              href="/get-started"
              className="rounded-sm border border-border bg-foreground px-5 py-2.5 font-mono text-xs uppercase tracking-wider text-(--band-on-fg) transition-opacity hover:opacity-90"
            >
              Open the Office
            </Link>
            <a
              href={DISCORD_INVITE}
              target="_blank"
              rel="noreferrer"
              className="rounded-sm border border-border text-black px-5 py-2.5 font-mono text-xs uppercase tracking-wider transition-colors hover:bg-secondary"
            >
              Join Discord
            </a>
          </div>
        </div>
      </div>

      <footer className="border-t border-border bg-background">
        <div className="mx-auto max-w-7xl px-6 py-16">
          <div className="grid grid-cols-2 gap-8 py-12 md:grid-cols-5">
            {columns.map((col) => (
              <div key={col.title}>
                <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
                  {col.title}
                </p>
                <ul className="mt-4 flex flex-col gap-3">
                  {col.links.map((link) => {
                    const href = linkHrefs[link];

                    return (
                      <li key={link}>
                        {href ? (
                          <Link
                            href={href}
                            className="font-mono text-xs uppercase tracking-wider text-foreground/80 transition-colors hover:text-foreground"
                          >
                            {link}
                          </Link>
                        ) : (
                          <span className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
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

          <div className="flex flex-col items-center justify-between gap-4 border-t border-border pt-8 sm:flex-row">
            <div className="flex items-center gap-2">
              <img
                src="/imgs/logo.jpg"
                alt="theboringfloor"
                className="h-8 w-8"
              />
              <span className="font-mono text-xs text-muted-foreground">
                © theboringfloor 2026
              </span>
            </div>
            <div className="flex items-center gap-4 text-muted-foreground">
              <a
                href={DISCORD_INVITE}
                target="_blank"
                rel="noreferrer"
                aria-label="Discord"
                className="hover:text-foreground"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z" />
                </svg>
              </a>
              <a href={X_URL} target="_blank" rel="noreferrer" aria-label="X" className="hover:text-foreground">
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                </svg>
              </a>
              <a
                href={GITHUB_REPO}
                target="_blank"
                rel="noreferrer"
                aria-label="GitHub"
                className="hover:text-foreground"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M12 .5C5.73.5.75 5.48.75 11.75c0 4.94 3.2 9.13 7.65 10.6.56.1.76-.24.76-.54 0-.27-.01-1.17-.02-2.12-3.11.67-3.77-1.32-3.77-1.32-.51-1.3-1.24-1.64-1.24-1.64-1.02-.69.08-.68.08-.68 1.12.08 1.71 1.15 1.71 1.15 1 1.71 2.62 1.22 3.26.93.1-.72.39-1.22.71-1.5-2.48-.28-5.09-1.24-5.09-5.53 0-1.22.44-2.22 1.15-3-.12-.28-.5-1.42.11-2.96 0 0 .94-.3 3.08 1.15a10.7 10.7 0 0 1 5.6 0c2.14-1.45 3.08-1.15 3.08-1.15.61 1.54.23 2.68.11 2.96.72.78 1.15 1.78 1.15 3 0 4.3-2.62 5.25-5.11 5.53.4.35.76 1.03.76 2.08 0 1.5-.01 2.71-.01 3.08 0 .3.2.65.77.54 4.44-1.48 7.64-5.66 7.64-10.6C23.25 5.48 18.27.5 12 .5z" />
                </svg>
              </a>
              <a
                href={LINKEDIN_URL}
                target="_blank"
                rel="noreferrer"
                aria-label="LinkedIn"
                className="hover:text-foreground"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M20.45 20.45h-3.56v-5.57c0-1.33-.02-3.04-1.85-3.04-1.86 0-2.14 1.45-2.14 2.94v5.67H9.34V9h3.42v1.56h.05c.48-.9 1.64-1.85 3.38-1.85 3.61 0 4.28 2.38 4.28 5.47zM5.34 7.43a2.06 2.06 0 1 1 0-4.13 2.06 2.06 0 0 1 0 4.13zM3.56 20.45h3.56V9H3.56z" />
                </svg>
              </a>
            </div>
          </div>
        </div>
      </footer>
    </>
  );
}
