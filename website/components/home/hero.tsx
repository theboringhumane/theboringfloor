"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Database } from "lucide-react";

import { Chapter, Panel, Sheet, Stamp } from "@/components/paper";
import { ChaosCanvas } from "@/components/home/chaos-canvas";
import { INSTALL_PS1, INSTALL_SH, SITE_NAME } from "@/lib/site";

const INSTALL_UNIX = `curl -fsSL ${INSTALL_SH} | sh`;
const INSTALL_WIN = `irm ${INSTALL_PS1} | iex`;

/** The two ends of the hero animation, printed on the sheet's frame. */
function FrameLabels({ className = "" }: { className?: string }) {
  return (
    <div className={`mono-label flex items-center gap-3 text-ink-faint ${className}`} aria-hidden="true">
      <span>⊢ Scrollback</span>
      <span className="h-px flex-1 bg-rule" />
      <span>The board ⊣</span>
    </div>
  );
}

type Platform = "mac" | "linux" | "windows";

function AppleMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden>
      <path
        fill="currentColor"
        d="M16.365 1.43c0 1.14-.37 2.15-1.06 3.19-.91 1.29-2.11 2.04-3.39 2.04-.13-.96.27-1.98.94-3.1C13.5 2.34 14.89 1.5 16.365 1.43zM20.48 17.21c-.58 1.33-.85 1.92-1.59 3.09-1.03 1.61-2.48 3.62-4.27 3.64-1.58.03-1.99-1.03-4.15-1.02-2.16.01-2.61 1.06-4.2 1.03-1.79-.03-3.16-1.83-4.19-3.44C.27 16.76-.96 11.97.94 8.7c.95-1.64 2.67-2.68 4.53-2.71 1.68-.03 3.27 1.13 4.15 1.13.86 0 2.73-1.4 4.6-1.19.78.03 2.98.32 4.39 2.39-3.74 2.05-3.14 7.39.87 8.89z"
      />
    </svg>
  );
}

function WindowsMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden>
      <path
        fill="currentColor"
        d="M1 3.54 10.74 2.2v9.37H1zm10.87-1.52L23 0v11.56H11.87zM1 12.73h9.74V22.1L1 20.78zm10.87.01H23V24l-11.13-1.56z"
      />
    </svg>
  );
}

function LinuxMark({ className }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={`inline-block bg-current ${className ?? ""}`}
      style={{
        maskImage: "url(/icons/linux.svg)",
        WebkitMaskImage: "url(/icons/linux.svg)",
        maskRepeat: "no-repeat",
        WebkitMaskRepeat: "no-repeat",
        maskPosition: "center",
        WebkitMaskPosition: "center",
        maskSize: "contain",
        WebkitMaskSize: "contain",
      }}
    />
  );
}

export function Hero() {
  const [copied, setCopied] = useState(false);
  const [os, setOs] = useState<Platform>("mac");

  useEffect(() => {
    const ua = navigator.userAgent;
    if (/windows|win32|win64/i.test(ua)) setOs("windows");
    else if (/linux/i.test(ua) && !/android/i.test(ua)) setOs("linux");
    else setOs("mac");
  }, []);

  const install = os === "windows" ? INSTALL_WIN : INSTALL_UNIX;

  async function copyInstall() {
    try {
      await navigator.clipboard.writeText(install);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      /* ignore */
    }
  }

  return (
    <Sheet id="hero" className="paper-ruled border-b border-rule">
      <ChaosCanvas />
      <div className="relative z-10 px-6 pb-16 pt-12 md:px-10 md:pb-20 md:pt-16 lg:px-14">
        <div className="flex flex-wrap items-baseline justify-between gap-3 border-b border-rule pb-3">
          <span className="mono-label text-ink">{SITE_NAME}</span>
          <span className="mono-label">Terminal office · open source</span>
        </div>
        <FrameLabels />

        <h1 className="display-xl mt-16 max-w-[13ch] text-balance text-ink md:mt-24">
          Give every project its own <span className="italic">floor</span>.
        </h1>
        <p className="mt-8 max-w-xl text-pretty text-base leading-relaxed text-ink-soft md:text-lg">
          Projects, agent teams, tickets and conversations in one terminal. Plan
          the work, watch the office build it, keep the code in view. OpenCode,
          Claude Code or Codex — picked per conversation.
        </p>

        <div className="mt-10 flex flex-wrap items-center gap-2">
          <Stamp>Terminal UI</Stamp>
          <Stamp>Android companion</Stamp>
          <Stamp tone="stamp">Plan before build</Stamp>
        </div>

        <Panel label="Install" className="mt-10 max-w-2xl">
          <div className="flex border border-rule border-b-0">
            {[
              { id: "mac" as const, label: "macOS", Icon: AppleMark },
              { id: "linux" as const, label: "Linux", Icon: LinuxMark },
              { id: "windows" as const, label: "Windows", Icon: WindowsMark },
            ].map(({ id, label, Icon }) => {
              const on = os === id;
              return (
                <button
                  key={id}
                  type="button"
                  aria-pressed={on}
                  aria-label={`${label} install`}
                  onClick={() => setOs(id)}
                  className={`flex flex-1 items-center justify-center gap-2 px-3 py-2.5 font-mono text-[11px] uppercase tracking-[0.14em] transition-colors ${
                    on
                      ? "bg-ink text-paper"
                      : "text-ink-faint hover:text-ink"
                  }`}
                >
                  <Icon className="size-3.5 shrink-0" />
                  {label}
                </button>
              );
            })}
          </div>
          <div className="flex items-stretch border border-rule bg-paper font-mono text-sm">
            <code className="min-w-0 flex-1 overflow-x-auto px-4 py-3.5 text-ink">
              <span className="text-ink-faint">
                {os === "windows" ? "> " : "$ "}
              </span>
              {install}
            </code>
            <button
              type="button"
              onClick={copyInstall}
              className="shrink-0 border-l border-rule px-4 font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft transition-colors hover:text-ink"
            >
              {copied ? "Copied" : "Copy"}
            </button>
          </div>

          <div className="mt-5 flex flex-wrap items-center gap-3">
            <Link
              href="/get-started"
              className="bg-ink px-6 py-3 font-mono text-[11px] uppercase tracking-[0.14em] text-paper transition-opacity hover:opacity-90"
            >
              Open the office
            </Link>
            <Link
              href="/get-started"
              className="border border-rule px-6 py-3 font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft transition-colors hover:text-ink"
            >
              Tour demo mode
            </Link>
          </div>
        </Panel>

        <a
          href="https://synehq.com/"
          target="_blank"
          rel="noreferrer"
          className="mt-10 flex max-w-md overflow-hidden border border-rule transition-colors hover:bg-paper-2"
        >
          <div className="flex min-w-0 flex-1 flex-col justify-between gap-2 p-3">
            <p className="text-base font-medium tracking-tight text-ink">
              Talk to your data like a human.
            </p>
            <div className="flex items-center gap-2 text-ink-faint">
              <Database className="size-3.5 shrink-0" aria-hidden />
              <span className="font-mono text-[10px] uppercase tracking-[0.18em]">
                Query data with AI agents
              </span>
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-2 border-l border-rule p-3">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-ink-faint">
              Sponsored
            </span>
            <img
              className="h-7 w-auto"
              src="https://framerusercontent.com/images/DpVtRdL2gGDwSRNF4GSIdB6Ajkg.svg?scale-down-to=512&width=840&height=299"
              alt="Syne HQ"
            />
          </div>
        </a>

        <Chapter
          numeral="I."
          title="The floor"
          id="chapter-i"
          className="mt-20 md:mt-28"
          lede={
            <>One office. Every project gets its own room in it.</>
          }
        >
          <p className="mt-6 max-w-xl text-sm leading-relaxed text-ink-soft">
            The floor is where the work is kept: a board, the threads, the
            files, and the crew that clocked in. What follows is the tour.
          </p>
        </Chapter>
        <FrameLabels className="mt-16" />
      </div>
    </Sheet>
  );
}
