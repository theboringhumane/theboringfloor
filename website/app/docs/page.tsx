import Link from "next/link";
import type { Metadata } from "next";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { SITE_URL } from "@/lib/site";
import { PageCover } from "@/components/page-cover";
import { ArrowUpRight } from "lucide-react";

export const metadata: Metadata = {
  title: "Docs | theboringfloor",
  description:
    "The office manual — install, backends, chat, plan mode, the queue and board, panels, themes, and every key and slash command.",
  alternates: {
    canonical: "/docs",
  },
  openGraph: {
    title: "Docs · theboringfloor",
    description:
      "The office manual — install, backends, chat, plan mode, the queue and board, panels, themes, and every key and slash command.",
    url: `${SITE_URL}/docs`,
    type: "website",
  },
};

type DocLink = { href: string; name: string; promise: string };

const groups: { title: string; items: DocLink[] }[] = [
  {
    title: "Install & Setup",
    items: [
      {
        href: "/docs/getting-started",
        name: "Getting started",
        promise:
          "One curl, one demo tour, one live office — running in about five minutes.",
      },
    ],
  },
  {
    title: "Core",
    items: [
      {
        href: "/docs/workspaces",
        name: "Floors & workspaces",
        promise:
          "Project floors, teams, tickets, files, and a backend for each conversation.",
      },
      {
        href: "/docs/backends",
        name: "Backends",
        promise:
          "OpenCode, Claude Code, and Codex — choose a backend for each conversation.",
      },
      {
        href: "/docs/chat-and-threads",
        name: "Chat & work threads",
        promise:
          "One chat tab, replies that stream character-by-character, sub-agent threads in the open.",
      },
      {
        href: "/docs/plan-mode",
        name: "Plan mode",
        promise:
          "Talk the plan out, edit the draft yourself, approve it — then the crew builds.",
      },
      {
        href: "/docs/mcp-server",
        name: "MCP server",
        promise:
          "The local MCP bridge for plan drafts, approved plans, and the current project transcript.",
      },
      {
        href: "/docs/control-plane",
        name: "Remote control plane",
        promise:
          "floorgate, Tailscale, the Android app, and one authenticated API for live offices.",
      },
    ],
  },
  {
    title: "Workflow",
    items: [
      {
        href: "/docs/permissions-and-questions",
        name: "Permissions & questions",
        promise:
          "The gate the office never vaults — allow once, always, or reject, in place.",
      },
      {
        href: "/docs/queue-board-memory",
        name: "Queue, board & memory",
        promise:
          "Type while the boss types; board rows flip themselves; memory survives reboots.",
      },
    ],
  },
  {
    title: "Panels & Reference",
    items: [
      {
        href: "/docs/terminal-and-git-tabs",
        name: "Terminal & git tabs",
        promise:
          "A real shell and a live git panel one tab away from the chat.",
      },
      {
        href: "/docs/browser-tab",
        name: "Browser tab",
        promise:
          "Built-in text navigation, headless screenshots, and your system browser.",
      },
      {
        href: "/docs/layout-themes-power",
        name: "Layout, themes & power",
        promise:
          "Compact mode, themes, and the battery dial that keeps an idle office cheap.",
      },
      {
        href: "/docs/keys-and-slash",
        name: "Keys & slash commands",
        promise:
          "Every key binding and every slash command, one short table each.",
      },
    ],
  },
];

export default function DocsPage() {
  return (
    <div className="blue-site">
      <SiteHeader framed />
      <main id="main-content">
        <PageCover
          eyebrow="THE OFFICE MANUAL / DOCUMENTATION"
          title={
            <>
              Find your feet.
              <br />
              <span className="blue-text">Build your thing.</span>
            </>
          }
          description="From your first install to your next big project. Everything you need to make the office your own."
          art="memory"
        >
          <Link className="button-primary" href="/docs/getting-started">
            Start with the basics <ArrowUpRight size={18} />
          </Link>
          <Link className="text-link" href="/docs/keys-and-slash">
            Find a shortcut <ArrowUpRight size={17} />
          </Link>
        </PageCover>
        <section className="docs-directory section-pad">
          {groups.map((group, index) => (
            <div key={group.title} className="docs-group">
              <div className="docs-group-heading">
                <span className="eyebrow">0{index + 1} / THE MANUAL</span>
                <h2>{group.title}</h2>
              </div>
              <div className="docs-links">
                {group.items.map((item) => (
                  <Link key={item.href} href={item.href}>
                    <h3>
                      {item.name}
                      <ArrowUpRight size={18} />
                    </h3>
                    <p>{item.promise}</p>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </section>
        <section className="docs-help section-pad">
          <div>
            <span className="eyebrow">THREE BACKENDS. ONE FAMILIAR FLOOR.</span>
            <h2 className="section-title">
              Bring the brain
              <br />
              you work best with.
            </h2>
          </div>
          <div>
            <p className="section-lede">
              OpenCode, Claude Code, and Codex are supported. Our backend guide
              explains setup, authentication, and the capabilities available
              with each.
            </p>
            <Link className="text-link" href="/docs/backends">
              Choose your backend <ArrowUpRight size={17} />
            </Link>
          </div>
        </section>
      </main>
      <SiteFooter />
    </div>
  );
}
