import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { BrandMark } from "@/components/brand-mark";
import { ThemeToggle } from "@/components/theme-toggle";
import { DISCORD_INVITE, GITHUB_REPO, X_URL } from "@/lib/site";

const columns = [
  {
    title: "The product",
    links: [
      ["The office", "/#workspaces"],
      ["Plan mode", "/docs/plan-mode"],
      ["Mobile companion", "/#mobile"],
      ["Get started", "/get-started"],
    ],
  },
  {
    title: "For builders",
    links: [
      ["Documentation", "/docs"],
      ["Choose a backend", "/docs/backends"],
      ["MCP server", "/docs/mcp-server"],
      ["Source code", GITHUB_REPO],
    ],
  },
  {
    title: "Around the office",
    links: [
      ["Our vision", "/vision"],
      ["Journal", "/blog"],
      ["Changelog", "/changelog"],
      ["Office sounds", "/sounds"],
    ],
  },
  {
    title: "Say hello",
    links: [
      ["Discord", DISCORD_INVITE],
      ["GitHub", GITHUB_REPO],
      ["X / Twitter", X_URL],
      ["RSS feed", "/rss.xml"],
    ],
  },
];
export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="footer-top">
        <Link href="/" className="wordmark">
          <BrandMark />
          <span>theboringfloor.</span>
        </Link>
        <p>
          A little office.
          <br />A lot of possibility.
        </p>
        <a
          href={GITHUB_REPO}
          target="_blank"
          rel="noreferrer"
          className="footer-open"
        >
          Open source. Open doors.
          <ArrowUpRight size={18} />
        </a>
      </div>
      <div className="footer-columns">
        {columns.map((column) => (
          <div key={column.title}>
            <h2>{column.title}</h2>
            {column.links.map(([label, href]) =>
              href.startsWith("http") ? (
                <a key={label} href={href} target="_blank" rel="noreferrer">
                  {label}
                  <ArrowUpRight size={12} />
                </a>
              ) : (
                <Link key={label} href={href}>
                  {label}
                </Link>
              ),
            )}
          </div>
        ))}
      </div>
      <div className="footer-word" aria-hidden="true">
        make room.
      </div>
      <div className="footer-bottom">
        <span>© {new Date().getUTCFullYear()} theboringfloor</span>
        <span>Built in the open. Released under MIT.</span>
        <ThemeToggle />
      </div>
    </footer>
  );
}
