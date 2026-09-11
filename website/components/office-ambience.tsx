"use client";

import { useEffect, useRef, useState, type RefObject } from "react";
import { Volume2, VolumeX } from "lucide-react";
import { createOfficeAudio, type OfficeAudio } from "@/lib/office-audio";

export function OfficeAmbience({ scene, standalone = false }: { scene?: RefObject<HTMLDivElement | null>; standalone?: boolean }) {
  const host = useRef<HTMLDivElement>(null);
  const engine = useRef<OfficeAudio | null>(null);
  const [enabled, setEnabled] = useState(false);
  const [visible, setVisible] = useState(true);
  const [volume, setVolume] = useState(18);
  const [error, setError] = useState("");

  useEffect(() => {
    const element = scene?.current || host.current;
    if (!element) return;
    let inView = true;
    const update = () => setVisible(inView && !document.hidden);
    const observer = new IntersectionObserver(([entry]) => { inView = entry.isIntersecting; update(); }, { threshold: 0.05 });
    observer.observe(element);
    document.addEventListener("visibilitychange", update);
    return () => { observer.disconnect(); document.removeEventListener("visibilitychange", update); };
  }, [scene]);

  useEffect(() => {
    if (engine.current) void engine.current.setActive(enabled && visible).catch(() => {
      setEnabled(false);
      setError("Audio couldn’t start. Tap to try again.");
    });
  }, [enabled, visible]);

  useEffect(() => () => { engine.current?.dispose(); engine.current = null; }, []);

  function toggle() {
    setError("");
    try {
      // Created only after a click; browser autoplay rules stay in charge.
      if (!engine.current) engine.current = createOfficeAudio();
      engine.current.setVolume(volume / 100);
      void engine.current.setActive(!enabled && visible).catch(() => {
        setEnabled(false);
        setError("Audio couldn’t start. Tap to try again.");
      });
      setEnabled(!enabled);
    } catch {
      setError("Office ambience isn’t available in this browser.");
    }
  }

  return (
    <div ref={host} data-audio-state={enabled ? (visible ? "playing" : "paused") : "off"} className={`office-ambience ${standalone ? "office-ambience--standalone" : ""}`}>
      <button type="button" className="ambience-toggle" aria-pressed={enabled} aria-label={`${enabled ? "Turn off" : "Turn on"} office ambience`} onClick={toggle}
        title="Soft room tone, typing, a coffee machine, and a printer. Pauses when this scene leaves view.">
        {enabled ? <Volume2 size={15} /> : <VolumeX size={15} />}
        <span>{enabled ? "Sound on" : "Sound off"}</span>
      </button>
      {enabled && <label className="ambience-volume"><span className="sr-only">Office ambience volume</span>
        <input type="range" min="0" max="50" step="1" value={volume} aria-valuetext={`${volume} percent`} onChange={(event) => {
          const value = Number(event.target.value);
          setVolume(value);
          engine.current?.setVolume(value / 100);
        }} />
      </label>}
      <span role="status" className={error ? "ambience-error" : "sr-only"}>{error || (enabled ? "Office ambience enabled. Pauses when the scene is out of view." : "Office ambience off.")}</span>
    </div>
  );
}
