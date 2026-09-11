"use client";

import { useEffect, useRef, useState } from "react";
import { Pause, Play, RotateCw } from "lucide-react";
import { BlueprintArt } from "./blueprint-art";

export function OfficeScene() {
  const host = useRef<HTMLDivElement>(null);
  const controls = useRef<{
    paused: boolean;
    turn: number;
    update?: () => void;
  }>({ paused: false, turn: 0 });
  const [paused, setPaused] = useState(false);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const element = host.current;
    if (!element) return;
    let disposed = false;
    let cleanup: (() => void) | undefined;
    // Load WebGL separately; the original SVG remains visible if it is unavailable.
    import("./office-renderer")
      .then(({ mountOffice }) => {
        if (disposed) return;
        cleanup = mountOffice(
          element,
          controls.current,
          () => setReady(true),
          () => setReady(false),
        );
      })
      .catch(() => setReady(false));
    return () => {
      disposed = true;
      cleanup?.();
    };
  }, []);
  return (
    <div className="office-scene">
      <div className="scene-topline">
        <span>
          <i /> THE FLOOR IS ALIVE
        </span>
        <span>EST. IN YOUR TERMINAL</span>
      </div>
      <div
        className={`scene-fallback ${ready ? "scene-fallback--hidden" : ""}`}
      >
        <BlueprintArt kind="floor" />
      </div>
      <div
        ref={host}
        className="scene-canvas"
        role="img"
        aria-label="An interactive blue isometric office with agent coworkers, desks, plants, and a server room"
      />
      <div className="scene-bottomline">
        <span>YOUR IDEAS. THEIR NEXT SHIFT.</span>
        <div className="scene-controls">
          {ready && (
            <>
              <button
                type="button"
                aria-label="Rotate the office"
                onClick={() => {
                  controls.current.turn += Math.PI / 2;
                  controls.current.update?.();
                }}
              >
                <RotateCw size={15} />
              </button>
              <button
                type="button"
                aria-label={
                  paused ? "Play office animation" : "Pause office animation"
                }
                onClick={() => {
                  controls.current.paused = !paused;
                  controls.current.update?.();
                  setPaused(!paused);
                }}
              >
                {paused ? <Play size={15} /> : <Pause size={15} />}
              </button>
            </>
          )}
          <span className="scene-coordinate">01 / THE OFFICE</span>
        </div>
      </div>
    </div>
  );
}
