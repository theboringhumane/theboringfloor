import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
import { SectionTag } from '@/components/section-tag'
import { ScrollReveal } from '@/components/scroll-reveal'
import { SoundCard, type SoundSpec } from '@/components/sounds/sound-card'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Sounds',
  description:
    'Seven synthesized office chimes, one meaning each — what every sound says, when it plays, and the WAV files themselves.',
  alternates: {
    canonical: '/sounds',
  },
  openGraph: {
    title: 'Sounds · theboringoffice',
    description:
      'Seven synthesized office chimes, one meaning each — what every sound says, when it plays, and the WAV files themselves.',
    url: `${SITE_URL}/sounds`,
    type: 'website',
  },
}

const sounds: SoundSpec[] = [
  {
    name: 'queued',
    duration: '40ms',
    waveform: 'sine 660Hz, -12dB',
    meaning: 'argo stands the queue down — something just joined the backlog',
  },
  {
    name: 'send',
    duration: '60ms',
    waveform: 'glide 520→640Hz, -14dB',
    meaning: 'message on its way',
  },
  {
    name: 'reply',
    duration: '90ms',
    waveform: 'two-tone C5→G5',
    meaning: 'boss finished a turn',
  },
  {
    name: 'done',
    duration: '110ms',
    waveform: 'rising triad C5-E5-G5',
    meaning: 'task completed',
  },
  {
    name: 'dispatch',
    duration: '80ms',
    waveform: 'brown-noise LP',
    meaning: 'the dispatch left the office',
  },
  {
    name: 'alert',
    duration: '140ms',
    waveform: '2×55ms square 880Hz beeps / 30ms gap',
    meaning: 'something needs you right now',
  },
  {
    name: 'error',
    duration: '140ms',
    waveform: 'descent 440→220Hz',
    meaning: "something didn't make it",
  },
]

export default function SoundsPage() {
  return (
    <>
      <SiteHeader />
      <main>
        <section className="border-b border-border">
          <div className="mx-auto max-w-5xl px-6 pb-20 pt-16 md:pt-24">
            <SectionTag>Sounds</SectionTag>
            <h1 className="mt-8 max-w-3xl text-balance text-4xl font-semibold leading-tight tracking-tight md:text-6xl">
              The office&apos;s sonic weather.
            </h1>
            <p className="mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              Seven synthesized chimes, deterministic and dim-lit every boot. Each has a single
              meaning — played and heard by the people actually running this office.
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto max-w-7xl px-6 py-20">
            <SectionTag>The library</SectionTag>
            <h2 className="mt-6 max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight md:text-4xl">
              One chime, one meaning.
            </h2>
            <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-muted-foreground">
              No notification soup. When the office makes a sound it is saying exactly one
              thing — this is the whole vocabulary, served back as plain WAVs.
            </p>
            <ScrollReveal
              stagger={0.06}
              className="mt-14 grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-2"
            >
              {sounds.map((s) => (
                <SoundCard key={s.name} sound={s} />
              ))}
            </ScrollReveal>
            <p className="mt-4 font-mono text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
              16-bit mono 22050 Hz PCM · rendered by internal/sound, never sampled
            </p>
          </div>
        </section>

        <section className="border-b border-border">
          <div className="mx-auto flex max-w-5xl flex-col items-start gap-6 px-6 py-20">
            <SectionTag>From the terminal</SectionTag>
            <p className="max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
              These are synthesized at boot inside the office — no assets shipped in the binary —
              and they play through the terminal&apos;s own player. Today, the site serves them
              back as plain WAVs.
            </p>
            <div className="mt-2 flex flex-wrap gap-3">
              <Link
                href="/get-started"
                className="inline-flex items-center bg-foreground px-6 py-3 font-mono text-xs uppercase tracking-wider text-background transition-opacity hover:opacity-90"
              >
                Open the office
              </Link>
            </div>
          </div>
        </section>
      </main>
      <SiteFooter />
    </>
  )
}
