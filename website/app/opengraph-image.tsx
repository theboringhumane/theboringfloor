import { ImageResponse } from 'next/og'
import { SITE_NAME, SITE_TAGLINE } from '@/lib/site'

export const alt = `${SITE_NAME} — ${SITE_TAGLINE}`
export const size = {
  width: 1200,
  height: 630,
}
export const contentType = 'image/png'
export const dynamic = 'force-static'

export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          alignItems: 'flex-start',
          backgroundColor: '#f8f8f4',
          color: '#0a0a0a',
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          justifyContent: 'center',
          padding: '72px',
          width: '100%',
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 22 }}>
          <div style={{ display: 'flex', fontSize: 88, fontWeight: 700, letterSpacing: '-0.06em' }}>
            {SITE_NAME}
          </div>
          <div style={{ display: 'flex', fontSize: 42, letterSpacing: '-0.025em' }}>{SITE_TAGLINE}</div>
        </div>
      </div>
    ),
    size,
  )
}
