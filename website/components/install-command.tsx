"use client";
import { useEffect, useRef, useState } from "react";
import { Check, Copy, Terminal } from "lucide-react";
import { INSTALL_PS1, INSTALL_SH } from "@/lib/site";
export function InstallCommand({ compact = false }: { compact?: boolean }) {
  const [platform, setPlatform] = useState("macOS");
  const [status, setStatus] = useState("");
  const timeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (/Windows/i.test(navigator.userAgent)) setPlatform("Windows");
    else if (
      /Linux/i.test(navigator.userAgent) &&
      !/Android/i.test(navigator.userAgent)
    )
      setPlatform("Linux");
    return () => {
      if (timeout.current) clearTimeout(timeout.current);
    };
  }, []);
  const command =
    platform === "Windows"
      ? `irm ${INSTALL_PS1} | iex`
      : `curl -fsSL ${INSTALL_SH} | sh`;
  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setStatus("Copied to clipboard");
    } catch {
      setStatus("Select the command to copy it manually.");
    }
    if (timeout.current) clearTimeout(timeout.current);
    timeout.current = setTimeout(() => setStatus(""), 3000);
  }
  return (
    <div
      className={`install-command ${compact ? "install-command--compact" : ""}`}
    >
      <div className="install-platforms" aria-label="Operating system">
        {["macOS", "Linux", "Windows"].map((os) => (
          <button
            key={os}
            type="button"
            aria-pressed={platform === os}
            onClick={() => {
              setPlatform(os);
              setStatus("");
            }}
          >
            {os}
          </button>
        ))}
        <Terminal size={15} aria-hidden="true" />
      </div>
      <div className="install-code">
        <span aria-hidden="true">{platform === "Windows" ? ">" : "$"}</span>
        <code>{command}</code>
        <button type="button" onClick={copy} aria-label="Copy install command">
          {status.startsWith("Copied") ? (
            <Check size={17} />
          ) : (
            <Copy size={17} />
          )}
        </button>
      </div>
      <span className="install-feedback" role="status">
        {status || "One command. A whole new way to work."}
      </span>
    </div>
  );
}
