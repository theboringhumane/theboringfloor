import Link from "next/link";
import type { Metadata } from "next";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { PageCover } from "@/components/page-cover";
import { Chapter, Cue, Stamp } from "@/components/paper";
import { SoundCard, type SoundSpec } from "@/components/sounds/sound-card";
import { SITE_URL } from "@/lib/site";

export const metadata: Metadata = {
  title: "Sounds",
  description:
    "Seven synthesized office chimes, one meaning each — what every sound says, when it plays, and the WAV files themselves.",
  alternates: {
    canonical: "/sounds",
  },
  openGraph: {
    title: "Sounds · theboringfloor",
    description:
      "Seven synthesized office chimes, one meaning each — what every sound says, when it plays, and the WAV files themselves.",
    url: `${SITE_URL}/sounds`,
    type: "website",
  },
};

const sounds: SoundSpec[] = [
  {
    name: "queued",
    duration: "40ms",
    waveform: "sine 660Hz, -12dB",
    meaning: "argo stands the queue down — something just joined the backlog",
  },
  {
    name: "send",
    duration: "60ms",
    waveform: "glide 520→640Hz, -14dB",
    meaning: "message on its way",
  },
  {
    name: "reply",
    duration: "90ms",
    waveform: "two-tone C5→G5",
    meaning: "boss finished a turn",
  },
  {
    name: "done",
    duration: "110ms",
    waveform: "rising triad C5-E5-G5",
    meaning: "task completed",
  },
  {
    name: "dispatch",
    duration: "80ms",
    waveform: "brown-noise LP",
    meaning: "the dispatch left the office",
  },
  {
    name: "alert",
    duration: "140ms",
    waveform: "2×55ms square 880Hz beeps / 30ms gap",
    meaning: "something needs you right now",
  },
  {
    name: "error",
    duration: "140ms",
    waveform: "descent 440→220Hz",
    meaning: "something didn't make it",
  },
];

export default function SoundsPage() {
  return (
    <div className="paper-ground min-h-svh">
      <div className="doc-frame">
        <SiteHeader framed />
        <main>
          <PageCover
            eyebrow="THE SOUND OF WORK / OFFICE CHIMES"
            title={
              <>
                A little sound.
                <br />
                <span className="blue-text">A little meaning.</span>
              </>
            }
            description="Seven original chimes for the moments that matter. A message sent, a job done, a team that needs your attention."
            art="network"
          />

          {/* I — the library of specimens */}
          <section className="relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="I" title="Hear the library">
              <h3 className="display-lg mt-8 max-w-2xl text-balance text-ink">
                One chime, one meaning.
              </h3>
              <p className="mt-6 max-w-2xl text-pretty leading-relaxed text-ink-soft">
                No notification soup. When the office makes a sound it is saying
                exactly one thing — this is the whole vocabulary, served back as
                plain WAVs.
              </p>
              <div className="mt-12 grid grid-cols-1 gap-6 md:grid-cols-2">
                {sounds.map((s) => (
                  <SoundCard key={s.name} sound={s} />
                ))}
              </div>
              <p className="mono-label mt-6 text-ink-faint">
                16-bit mono 22050 Hz PCM · rendered by internal/sound, never
                sampled
              </p>
            </Chapter>
          </section>

          {/* II — provenance */}
          <section className="paper-panel relative border-b border-rule px-6 py-16 md:px-10 lg:px-14">
            <Chapter numeral="II" title="Trace the source">
              <p className="mt-8 max-w-2xl text-pretty text-lg leading-relaxed text-ink-soft">
                These are synthesized at boot inside the office — no assets
                shipped in the binary — and they play through the
                terminal&apos;s own player. Today, the site serves them back as
                plain WAVs.
              </p>
              <div className="mt-8 flex flex-wrap items-center gap-3">
                <Link
                  href="/get-started"
                  className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
                >
                  <Stamp className="transition-colors hover:border-ink hover:text-ink">
                    Open the office
                  </Stamp>
                </Link>
                <Cue>headphones optional</Cue>
              </div>
            </Chapter>
          </section>
        </main>
        <SiteFooter />
      </div>
    </div>
  );
}
