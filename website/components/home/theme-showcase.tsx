"use client";

import { useEffect, useId, useRef, useState, type CSSProperties } from "react";
import Link from "next/link";
import { ArrowUpRight, Check, Copy, Moon, Sun } from "lucide-react";
import palettes from "@/lib/theme-palettes.json";
import sizes from "@/lib/shot-sizes.json";

const descriptions: Record<string, string> = {
  cockpit: "Midnight blue. Phosphor instruments. Your command deck, ready for anything.",
  noir: "Soft charcoal and a little amber. A quiet place for your next big idea.",
  paper: "A fresh sheet for a fresh start. Clear ink and warm accents for daylight desks.",
  mono: "Every shade of focus. A grayscale workspace that lets the work speak.",
  dracula: "A familiar after-dark favorite. Purple, mint, and a bright spark of yellow.",
  solarized: "Measured contrast. Deep blue-green. A carefully balanced classic.",
  "tokyo-night": "City lights for late-night ideas. Electric blue on a deep indigo canvas.",
  "catppuccin-mocha": "Soft pastels, a dark canvas, and a little lavender. Settle in.",
  "catppuccin-latte": "The lighter side of Catppuccin. Creamy whites with playful pastel accents.",
  nord: "Cool blues and crisp highlights. A little Arctic calm for a busy floor.",
  gruvbox: "Warm earth tones and golden details. A retro palette with plenty of character.",
  "one-dark": "Balanced charcoal. Familiar colors. An easy place to find your flow.",
  "rose-pine": "Muted rose, soft gold, and deep purple. A quieter kind of color.",
  "github-light": "Clean white and confident blue. A bright, familiar home for your code.",
};
const themeName = (id: string) => id === "github-light" ? "GitHub Light" : id.split("-").map((word) => word[0].toUpperCase() + word.slice(1)).join(" ");

export function ThemeShowcase({ documentation = false }: { documentation?: boolean }) {
  const [selected, setSelected] = useState(0);
  const [feedback, setFeedback] = useState("");
  const [loaded, setLoaded] = useState<string | null>(null);
  const [failed, setFailed] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const preview = useRef<HTMLImageElement>(null);
  const panel = useRef<HTMLDivElement>(null);
  const id = useId();
  const palette = palettes[selected];
  const name = themeName(palette.id);
  const src = `/shots/themes/${palette.id}.webp`;
  const dimensions = (sizes as Record<string, { width: number; height: number }>)[src];
  const command = `/theme ${palette.id}`;
  const style = { "--palette-bg": palette.background, "--palette-ink": palette.foreground, "--palette-accent": palette.colors[0] } as CSSProperties;

  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);
  useEffect(() => {
    // Cached images can finish before React attaches the load listener.
    if (preview.current?.complete && preview.current.naturalWidth) setLoaded(palette.id);
  }, [palette.id]);

  function choose(index: number, reveal = false) {
    setSelected(index);
    setFeedback("");
    setFailed(null);
    if (timer.current) clearTimeout(timer.current);
    if (reveal && panel.current && panel.current.clientWidth < 740) {
      panel.current.scrollIntoView({ block: "start", behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "instant" : "smooth" });
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setFeedback("Copied. Paste it into the app’s command input.");
    } catch {
      setFeedback("Select the command above to copy it manually.");
    }
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setFeedback(""), 4000);
  }

  return (
    <div className="theme-showcase" style={style}>
      <div className="theme-gallery-label">
        <span><span className="status-square" /> THE PALETTE LIBRARY</span>
        <span>{palettes.length} THEMES / ALL INCLUDED</span>
      </div>
      <div ref={panel} className="theme-display" role="tabpanel" id={`${id}-preview`} aria-labelledby={`${id}-theme-${selected}`} tabIndex={0}>
        <div className="theme-detail">
          <div className="theme-edition">
            <span>{String(selected + 1).padStart(2, "0")} / {palettes.length}</span>
            <span>{palette.dark ? <Moon size={12} /> : <Sun size={12} />}{palette.dark ? "Dark" : "Light"}</span>
          </div>
          <div key={palette.id} className="theme-story">
            <h3>{name}</h3>
            <p>{descriptions[palette.id] || "A fresh palette for your office."}</p>
            <div className="theme-swatches" aria-label={`${name} accent colors`}>
              {palette.colors.map((color, i) => <span key={i} style={{ background: color }} title={color} />)}
            </div>
          </div>
          <div className="theme-command">
            <span className="eyebrow">MAKE IT YOURS</span>
            <div>
              <code>{command}</code>
              <button type="button" onClick={copy} aria-label={`Copy ${command}`}>
                {feedback.startsWith("Copied") ? <Check size={16} /> : <Copy size={16} />}
              </button>
            </div>
            <p role="status">{feedback || "Run this command inside theboringfloor."}</p>
          </div>
        </div>
        <div className="theme-stage">
          <div className="theme-screen-label">
            <span>theboringfloor / {palette.id}</span>
            <a href={src} target="_blank" rel="noreferrer" aria-label={`View ${name} screenshot full size`}>Full size <ArrowUpRight size={13} /></a>
          </div>
          <a className="theme-preview" href={src} target="_blank" rel="noreferrer" aria-label={`Inspect ${name} theme full size`}>
            <img ref={preview} key={palette.id} src={src} width={dimensions.width} height={dimensions.height}
              alt={`The actual theboringfloor cockpit in ${name}: project navigation, tactical floor, agent operations, and command console.`}
              loading="lazy" onLoad={() => setLoaded(palette.id)} onError={() => setFailed(palette.id)}
              className={loaded === palette.id ? "is-loaded" : ""} />
            {failed === palette.id && <span className="theme-image-error">Preview unavailable. Open the full-size image to retry. ↗</span>}
          </a>
          <div className="theme-screen-caption"><span>ACTUAL INTERFACE</span><span>SIMULATED DEMO MISSION</span></div>
        </div>
      </div>
      <div className="theme-picker" role="tablist" aria-label="Choose an app theme">
        {palettes.map((theme, i) => (
          <button key={theme.id} type="button" role="tab" id={`${id}-theme-${i}`} aria-selected={selected === i}
            aria-controls={`${id}-preview`} tabIndex={selected === i ? 0 : -1} onClick={() => choose(i, true)}
            onKeyDown={(event) => {
              let next = i;
              if (event.key === "ArrowRight") next = (i + 1) % palettes.length;
              else if (event.key === "ArrowLeft") next = (i - 1 + palettes.length) % palettes.length;
              else if (event.key === "Home") next = 0;
              else if (event.key === "End") next = palettes.length - 1;
              else return;
              event.preventDefault();
              choose(next);
              document.getElementById(`${id}-theme-${next}`)?.focus({ preventScroll: true });
            }}>
            <span className="theme-thumbnail" style={{ background: theme.background }}>
              <img src={`/shots/themes/${theme.id}-thumb.webp`} alt="" width={384} height={Math.round(dimensions.height * 384 / dimensions.width)} loading="lazy" />
              <span className="theme-selected-mark" aria-hidden="true"><Check size={12} /></span>
            </span>
            <span className="theme-choice-name">{themeName(theme.id)}</span>
            <span className="theme-choice-mode">{theme.dark ? "DARK" : "LIGHT"}{theme.id === "cockpit" ? " / DEFAULT" : ""}</span>
          </button>
        ))}
      </div>
      <div className="theme-gallery-footer">
        <p>Your favorite colors belong here, too. Import a VS Code theme or make your own.</p>
        <Link href={documentation ? "#import-theme" : "/docs/layout-themes-power/#import-theme"} className="text-link">Make it personal <ArrowUpRight size={16} /></Link>
      </div>
    </div>
  );
}
