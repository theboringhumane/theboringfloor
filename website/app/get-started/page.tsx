import Link from "next/link";
import type { Metadata } from "next";
import { ArrowUpRight, Check, Terminal } from "lucide-react";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { PageCover } from "@/components/page-cover";
import { InstallCommand } from "@/components/install-command";
import { GITHUB_REPO } from "@/lib/site";
export const metadata: Metadata = {
  title: "Get started",
  description:
    "Install theboringfloor on macOS, Linux, or Windows. Tour your new terminal office and connect OpenCode, Claude Code, or Codex.",
  alternates: { canonical: "/get-started" },
};
const backends = [
  {
    name: "OpenCode",
    note: "The default. Your office starts OpenCode for you.",
    command: "theboringfloor",
    link: "https://opencode.ai",
  },
  {
    name: "Claude Code",
    note: "Use your installed Claude CLI and existing account.",
    command: "theboringfloor --backend claudecode",
    link: "https://code.claude.com/docs/en/setup",
  },
  {
    name: "Codex",
    note: "Sign in to the Codex CLI, then bring it to the floor.",
    command: "codex login\ntheboringfloor --backend codex",
    link: "https://developers.openai.com/codex/cli",
  },
];
export default function GetStartedPage() {
  return (
    <div className="blue-site">
      <SiteHeader framed />
      <main id="main-content">
        <PageCover
          eyebrow="WELCOME ABOARD / GET STARTED"
          title={
            <>
              Your next big thing
              <br />
              starts <span className="blue-text">right here.</span>
            </>
          }
          description="A few minutes, one terminal, and a whole new team. Let’s get your office up and running."
          art="floor"
        >
          <a href="#install" className="button-primary">
            Let’s clock in <ArrowUpRight size={18} />
          </a>
          <Link href="/docs/getting-started" className="text-link">
            The full setup guide <ArrowUpRight size={17} />
          </Link>
        </PageCover>
        <section id="install" className="setup-step section-pad">
          <div className="setup-step-copy">
            <span className="eyebrow">01 / MAKE YOURSELF AT HOME</span>
            <h2 className="section-title">
              Open a terminal.
              <br />
              Open an office.
            </h2>
            <p>
              The installer adds the native application and sets up agentmemory
              as a reboot-safe service.
            </p>
            <ul className="check-list">
              <li>
                <Check size={16} /> macOS, Linux, and Windows
              </li>
              <li>
                <Check size={16} /> Native Go binary
              </li>
              <li>
                <Check size={16} /> Free and MIT licensed
              </li>
            </ul>
          </div>
          <div className="setup-command">
            <InstallCommand />
            <a
              href={`${GITHUB_REPO}/releases/latest`}
              target="_blank"
              rel="noreferrer"
              className="text-link"
            >
              Prefer a manual download? <ArrowUpRight size={15} />
            </a>
            <p>
              On Windows, open a new PowerShell window after installation so the
              new PATH takes effect.
            </p>
          </div>
        </section>
        <section className="setup-step section-pad">
          <div className="setup-step-copy">
            <span className="eyebrow">02 / TAKE A LOOK AROUND</span>
            <h2 className="section-title">
              Meet the office.
              <br />
              No keys needed.
            </h2>
            <p>
              Try demo mode first. Walk the floor, explore the panels, and get a
              feel for the workspace before connecting a coding backend.
            </p>
          </div>
          <div className="demo-command">
            <Terminal size={26} />
            <span className="eyebrow">YOUR FIRST LOOK</span>
            <code>theboringfloor --demo</code>
            <span>Demo mode is a tour. Live agents come next.</span>
          </div>
        </section>
        <section className="setup-backends section-pad">
          <div className="section-kicker">
            <span className="eyebrow">
              03 / BRING YOUR FAVORITE INTELLIGENCE
            </span>
          </div>
          <div className="section-heading-row">
            <h2 className="section-title">
              Same office.
              <br />
              Your kind of agent.
            </h2>
            <p className="section-lede">
              Set up your preferred coding CLI and its authentication first.
              Then start your live office.
            </p>
          </div>
          <div className="backend-cards">
            {backends.map((backend) => (
              <article key={backend.name}>
                <h3>{backend.name}</h3>
                <p>{backend.note}</p>
                <pre>
                  <code>{backend.command}</code>
                </pre>
                <a
                  href={backend.link}
                  target="_blank"
                  rel="noreferrer"
                  className="text-link"
                >
                  Set up {backend.name}
                  <ArrowUpRight size={15} />
                </a>
              </article>
            ))}
          </div>
          <p className="setup-note">
            Your coding backend and model provider may have their own
            subscriptions or usage costs.{" "}
            <Link href="/docs/backends">Read about backends →</Link>
          </p>
        </section>
        <section className="setup-next section-pad">
          <span className="eyebrow">YOU’RE IN. MAKE YOURSELF AT HOME.</span>
          <h2 className="section-title">Give your idea a floor.</h2>
          <p>
            Press <kbd>Ctrl + E</kbd> to open project floors. Use{" "}
            <kbd>Ctrl + N</kbd> to start a conversation and choose your team.
            Your last conversation returns when you come back.
          </p>
          <div>
            <Link className="button-primary" href="/docs/workspaces">
              Explore your workspace <ArrowUpRight size={18} />
            </Link>
            <Link className="text-link" href="/docs/keys-and-slash">
              Learn the shortcuts <ArrowUpRight size={16} />
            </Link>
          </div>
        </section>
      </main>
      <SiteFooter />
    </div>
  );
}
