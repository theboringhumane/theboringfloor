import { ImageResponse } from 'next/og'
import { SITE_NAME, SITE_TAGLINE, SITE_URL } from '@/lib/site'

export const alt = `${SITE_NAME} — ${SITE_TAGLINE}`
export const size = {
  width: 1200,
  height: 630,
}
export const contentType = 'image/png'
export const dynamic = 'force-static'

/**
 * Literal sRGB equivalents of the paper tokens in app/globals.css.
 * Satori (next/og) does not resolve CSS custom properties, Tailwind classes,
 * or `oklch()` — it silently paints oklch() as black — so these are hex.
 * This exemption applies to the opengraph-image files only.
 */
const PAPER = '#e8e3db' // oklch(0.918 0.012 84)
const PAPER_2 = '#f1ede7' // oklch(0.948 0.009 84)
const PANEL = '#e3d1b3' // oklch(0.868 0.045 82)
const INK = '#34271c' // oklch(0.285 0.028 62)
const INK_SOFT = '#60544a' // oklch(0.455 0.022 62)
const RULE = 'rgba(52, 39, 28, 0.18)' // oklch(0.285 0.028 62 / 18%)
const STAMP = '#a12d1a' // oklch(0.475 0.155 32)

const BRACKET = 34
const FRAME = 40

/** Hairline registration bracket, drawn at one corner of the frame. */
function Bracket({ top, left }: { top: boolean; left: boolean }) {
  return (
    <div
      style={{
        display: 'flex',
        position: 'absolute',
        width: BRACKET,
        height: BRACKET,
        ...(top
          ? { top: -1, borderTop: `2px solid ${INK}` }
          : { bottom: -1, borderBottom: `2px solid ${INK}` }),
        ...(left
          ? { left: -1, borderLeft: `2px solid ${INK}` }
          : { right: -1, borderRight: `2px solid ${INK}` }),
      }}
    />
  )
}

export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          backgroundColor: PAPER,
          color: INK,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          position: 'relative',
          width: '100%',
        }}
      >
        {/* faint ruled grid */}
        <div
          style={{
            display: 'flex',
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundImage: `linear-gradient(to right, ${RULE} 1px, transparent 1px), linear-gradient(to bottom, ${RULE} 1px, transparent 1px)`,
            backgroundSize: '60px 60px',
            opacity: 0.5,
          }}
        />

        {/* document frame + corner registration brackets */}
        <div
          style={{
            display: 'flex',
            position: 'absolute',
            top: FRAME,
            left: FRAME,
            right: FRAME,
            bottom: FRAME,
            border: `1px solid ${RULE}`,
          }}
        >
          <Bracket top left />
          <Bracket top left={false} />
          <Bracket top={false} left />
          <Bracket top={false} left={false} />
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            justifyContent: 'space-between',
            padding: '92px 96px',
            width: '100%',
          }}
        >
          {/* eyebrow */}
          <div
            style={{
              alignItems: 'center',
              color: INK_SOFT,
              display: 'flex',
              fontSize: 21,
              justifyContent: 'space-between',
              letterSpacing: '0.22em',
              textTransform: 'uppercase',
              width: '100%',
            }}
          >
            <div style={{ alignItems: 'center', display: 'flex' }}>
              <div style={{ display: 'flex', width: 12, height: 12, backgroundColor: STAMP, marginRight: 16 }} />
              <div style={{ display: 'flex' }}>Terminal UI for coding agents</div>
            </div>
            <div style={{ display: 'flex' }}>Open source</div>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
            <div
              style={{
                display: 'flex',
                fontSize: 92,
                fontWeight: 700,
                letterSpacing: '-0.055em',
                lineHeight: 1,
              }}
            >
              {SITE_NAME}
            </div>
            <div style={{ display: 'flex', height: 1, backgroundColor: RULE, margin: '34px 0 30px', width: '100%' }} />
            <div
              style={{
                display: 'flex',
                fontSize: 46,
                letterSpacing: '-0.03em',
                lineHeight: 1.15,
                maxWidth: 840,
              }}
            >
              {SITE_TAGLINE}
            </div>
          </div>

          {/* footer line */}
          <div
            style={{
              alignItems: 'center',
              color: INK_SOFT,
              display: 'flex',
              fontSize: 19,
              justifyContent: 'space-between',
              letterSpacing: '0.16em',
              textTransform: 'uppercase',
              width: '100%',
            }}
          >
            <div style={{ display: 'flex' }}>{SITE_URL.replace('https://', '')}</div>
            <div
              style={{
                display: 'flex',
                backgroundColor: PANEL,
                border: `1px solid ${RULE}`,
                color: INK,
                padding: '8px 18px',
              }}
            >
              OpenCode · Claude Code · Codex
            </div>
          </div>
        </div>

        {/* bottom edge tint, like a trimmed sheet */}
        <div
          style={{
            display: 'flex',
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            height: 10,
            backgroundColor: PAPER_2,
          }}
        />
      </div>
    ),
    size,
  )
}
