"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Database } from "lucide-react";
import { ShaderBackground } from "@/components/ui/light-blue-plasma-shader-w-grain-interactive";

import { INSTALL_PS1, INSTALL_SH } from "@/lib/site";

const INSTALL_UNIX = `curl -fsSL ${INSTALL_SH} | sh`;
const INSTALL_WIN = `irm ${INSTALL_PS1} | iex`;

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
    <section
      id="hero"
      className="relative overflow-hidden border-b border-border bg-(--band-bg) text-(--band-fg)"
    >
      <ShaderBackground className="shot-img pointer-events-none absolute inset-y-0 right-0 w-[55%] opacity-40 max-md:opacity-25" />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-(--band-bg) via-(--band-bg)/85 to-transparent" />

      <div className="relative px-6 pb-20 pt-16 md:px-10 md:pb-24 md:pt-20 lg:px-14">
        <a
          href="https://synehq.com/"
          target="_blank"
          rel="noreferrer"
          className="mt-8 flex max-w-xl overflow-hidden border border-(--band-fg)/15 transition-colors hover:bg-(--band-fg)/4"
        >
          <div className="flex min-w-0 flex-1 flex-col justify-between p-2">
            <p className="font-sans text-lg font-medium tracking-tight md:text-xl">
              Talk to your data like a human.
            </p>
            <div className="flex items-center gap-2 text-(--band-fg)/45">
              <Database className="size-3.5 shrink-0" aria-hidden />
              <span className="font-mono text-[10px] uppercase tracking-[0.18em]">
                Query data with AI agents
              </span>
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end border-l border-(--band-fg)/15 p-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-(--band-fg)/40">
              Sponsored
            </span>
            <img
              className="h-7 w-auto"
              src="https://framerusercontent.com/images/DpVtRdL2gGDwSRNF4GSIdB6Ajkg.svg?scale-down-to=512&width=840&height=299"
              alt="Syne HQ"
            />
          </div>
        </a>

        <h1 className="mt-10 max-w-[14ch] font-sans text-[clamp(3.25rem,11vw,8rem)] font-medium leading-[0.9] tracking-[-0.05em]">
          <span className="">Work with agents</span>
          <br />
          that feel like{" "}
          <span className="text-(--band-accent) italic">coworkers.</span>
        </h1>
        <p className="mt-8 max-w-lg text-pretty font-sans text-base leading-relaxed text-(--band-fg)/50 md:text-lg">
          A terminal app where your opencode or Claude Code boss and sub-agents
          clock in as coworkers on a living floor. See the work move, talk to
          the boss like a person, and come back tomorrow to an office that
          remembers.
        </p>

        <div className="mt-12 flex max-w-xl flex-col">
          <div className="flex border border-b-0 border-(--band-fg)/15">
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
                  className={`flex flex-1 items-center justify-center gap-2 px-3 py-2.5 font-mono text-[11px] uppercase tracking-wider transition-colors ${
                    on
                      ? "bg-(--band-fg)/10 text-(--band-fg)"
                      : "text-(--band-fg)/40 hover:bg-(--band-fg)/5 hover:text-(--band-fg)/70"
                  }`}
                >
                  <Icon className="size-3.5 shrink-0" />
                  {label}
                </button>
              );
            })}
          </div>
          <div className="flex items-stretch border border-(--band-fg)/15 bg-(--band-well) font-mono text-sm">
            <code className="min-w-0 flex-1 overflow-x-auto px-4 py-3.5 text-(--band-fg)/75">
              <span className="text-(--band-fg)/35">
                {os === "windows" ? "> " : "$ "}
              </span>
              {install}
            </code>
            <button
              type="button"
              onClick={copyInstall}
              className="shrink-0 border-l border-(--band-fg)/15 px-4 text-[11px] uppercase tracking-wider text-(--band-fg)/55 transition-colors hover:bg-(--band-fg)/5 hover:text-(--band-fg)"
            >
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
        </div>

        <div className="mt-6 flex flex-wrap items-center gap-3">
          <Link
            href="/get-started"
            className="border border-(--band-fg)/80 bg-(--band-fg) px-6 py-3 font-mono text-xs uppercase tracking-wider text-(--band-on-fg) transition-opacity hover:opacity-90"
          >
            Open the office
          </Link>
          <Link
            href="/get-started"
            className="border border-(--band-fg)/25 px-6 py-3 font-mono text-xs uppercase tracking-wider text-(--band-fg)/80 transition-colors hover:bg-(--band-fg)/5"
          >
            Tour demo mode
          </Link>
        </div>
      </div>
    </section>
  );
}
