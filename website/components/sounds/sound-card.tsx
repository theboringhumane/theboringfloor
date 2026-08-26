'use client'

import { useRef, useState } from 'react'
import { Pause, Play } from 'lucide-react'

export type SoundSpec = {
  name: string
  duration: string
  waveform: string
  meaning: string
}

export function SoundCard({ sound }: { sound: SoundSpec }) {
  const audioRef = useRef<HTMLAudioElement>(null)
  const [playing, setPlaying] = useState(false)

  const toggle = () => {
    const el = audioRef.current
    if (!el) return
    if (playing) {
      el.pause()
    } else {
      el.currentTime = 0
      void el.play()
    }
  }

  return (
    <div className="flex flex-col gap-4 bg-background p-8">
      <div className="flex items-baseline justify-between gap-4">
        <h3 className="font-mono text-sm uppercase tracking-wider text-foreground">
          {sound.name}
        </h3>
        <span className="font-mono text-xs text-muted-foreground">{sound.duration}</span>
      </div>
      <p className="text-pretty text-sm leading-relaxed text-muted-foreground">
        {sound.meaning}
      </p>
      <div className="mt-auto flex flex-wrap items-center justify-between gap-4 border-t border-border pt-4">
        <button
          type="button"
          onClick={toggle}
          aria-label={`${playing ? 'Pause' : 'Play'} ${sound.name}`}
          className="inline-flex items-center gap-2 rounded-full border border-border px-4 py-1.5 font-mono text-xs uppercase tracking-wider text-foreground transition-colors hover:bg-secondary"
        >
          {playing ? (
            <Pause className="size-3" aria-hidden />
          ) : (
            <Play className="size-3" aria-hidden />
          )}
          [{playing ? 'pause' : 'play'}]
        </button>
        <p className="font-mono text-[11px] text-muted-foreground/70">
          waveform: {sound.waveform}
        </p>
      </div>
      <audio
        ref={audioRef}
        src={`/sounds/${sound.name}.wav`}
        preload="none"
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={() => setPlaying(false)}
      />
    </div>
  )
}
