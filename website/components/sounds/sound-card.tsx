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
    <div className="doc-brackets hairline flex flex-col gap-4 bg-paper-2 p-6 md:p-7">
      <div className="flex items-baseline justify-between gap-4">
        <h3 className="font-mono text-sm uppercase tracking-[0.14em] text-ink">{sound.name}</h3>
        <span className="mono-label text-ink-faint">{sound.duration}</span>
      </div>
      <p className="text-pretty text-sm leading-relaxed text-ink-soft">{sound.meaning}</p>
      <div className="mt-auto flex flex-wrap items-center justify-between gap-4 border-t border-rule pt-4">
        <button
          type="button"
          onClick={toggle}
          aria-label={`${playing ? 'Pause' : 'Play'} ${sound.name}`}
          className="inline-flex items-center gap-2 border border-rule px-3 py-1.5 font-mono text-[0.6875rem] uppercase leading-none tracking-[0.14em] text-ink-soft transition-colors hover:border-ink hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
        >
          {playing ? (
            <Pause className="size-3" aria-hidden />
          ) : (
            <Play className="size-3" aria-hidden />
          )}
          [{playing ? 'pause' : 'play'}]
        </button>
        <p className="font-mono text-[0.6875rem] text-ink-faint">waveform: {sound.waveform}</p>
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
