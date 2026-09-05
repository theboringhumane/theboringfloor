import { ImageResponse } from 'next/og'
import { getAllPosts, getPostBySlug } from '@/lib/blog'
import { SITE_NAME, SITE_TAGLINE } from '@/lib/site'

export const alt = SITE_NAME
export const size = {
  width: 1200,
  height: 630,
}
export const contentType = 'image/png'
export const dynamic = 'force-static'

export function generateStaticParams() {
  return getAllPosts().map((post) => ({ slug: post.slug }))
}

export default async function OpenGraphImage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params
  const post = getPostBySlug(slug)
  const title = post?.title ?? SITE_NAME

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
        <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
          <div style={{ display: 'flex', fontSize: 66, fontWeight: 700, letterSpacing: '-0.055em', lineHeight: 1.05 }}>
            {title}
          </div>
          <div style={{ display: 'flex', fontSize: 30, letterSpacing: '-0.025em' }}>{SITE_TAGLINE}</div>
        </div>
      </div>
    ),
    size,
  )
}
