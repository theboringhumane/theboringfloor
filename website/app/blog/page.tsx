import Link from 'next/link'
import type { Metadata } from 'next'
import { getAllPosts, formatDate } from '@/lib/blog'
import { BlogFilterList } from '@/components/blog/blog-filter-list'
import { Stamp } from '@/components/paper'
import { SITE_URL } from '@/lib/site'

export const metadata: Metadata = {
  title: 'Blog',
  description:
    'Product updates and engineering notes on AI coding agents, opencode subagents, and running a virtual office in the terminal.',
  alternates: {
    canonical: '/blog',
    types: {
      'application/rss+xml': `${SITE_URL}/rss.xml`,
    },
  },
  openGraph: {
    title: 'Blog · theboringfloor',
    description:
      'Product updates and engineering notes on AI coding agents, opencode subagents, and running a virtual office in the terminal.',
    url: `${SITE_URL}/blog`,
    type: 'website',
  },
}

export default function BlogPage() {
  const posts = getAllPosts()
  const featured = posts.filter((p) => p.featured).slice(0, 6)

  const categoryCounts = posts.reduce<Record<string, number>>((acc, p) => {
    p.categories.forEach((c) => {
      acc[c] = (acc[c] ?? 0) + 1
    })
    return acc
  }, {})

  return (
    <main>
      {/* Masthead */}
      <section className="relative border-b border-rule px-6 py-16 md:px-10 md:py-20 lg:px-14">

        <div className="flex items-baseline gap-4 border-b border-rule pb-3">
          <span className="mono-label text-ink">00</span>
          <span className="mono-label">Issue index</span>
          <span className="h-px flex-1 bg-rule" aria-hidden="true" />
          <span className="mono-label hidden sm:inline">
            {posts.length} {posts.length === 1 ? 'entry' : 'entries'}
          </span>
        </div>

        <h1 className="display-xl mt-8 max-w-[12ch] text-balance text-ink">Blog</h1>

        <p className="mt-6 max-w-xl text-pretty text-base leading-relaxed text-ink-soft">
          Product updates and engineering notes on coding agents, opencode subagents, and running a
          virtual office in the terminal.
        </p>

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <Stamp tone="stamp">Open source</Stamp>
          <Link
            href="/rss.xml"
            className="focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue"
          >
            <Stamp className="transition-colors hover:border-ink hover:text-ink">RSS feed</Stamp>
          </Link>
        </div>
      </section>

      {/* Featured entries — dateline + title only; the full index prints below. */}
      {featured.length > 0 && (
        <section className="relative border-b border-rule px-6 py-10 md:px-10 md:py-12 lg:px-14">
          <div className="flex items-center gap-3">
            <span className="mono-label text-ink">Selected</span>
            <span className="h-px flex-1 bg-rule" aria-hidden="true" />
          </div>

          <ol className="mt-6 divide-y divide-rule border-t border-rule">
            {featured.map((post, i) => (
              <li key={post.slug}>
                <Link
                  href={`/blog/${post.slug}`}
                  className="group grid grid-cols-1 gap-1 py-4 transition-colors hover:bg-paper-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue sm:grid-cols-[auto_10rem_1fr] sm:items-baseline sm:gap-6"
                >
                  <span className="mono-label text-ink-faint">
                    {String(i + 1).padStart(2, '0')}
                  </span>
                  <time
                    dateTime={post.date}
                    className="mono-label whitespace-nowrap text-ink-faint"
                  >
                    {formatDate(post.date)}
                  </time>
                  <span className="display-md text-ink group-hover:underline group-hover:decoration-rule group-hover:underline-offset-4">
                    {post.title}
                  </span>
                </Link>
              </li>
            ))}
          </ol>
        </section>
      )}

      <section className="relative border-b border-rule px-6 py-12 md:px-10 md:py-14 lg:px-14">
        <BlogFilterList posts={posts} categoryCounts={categoryCounts} />
      </section>
    </main>
  )
}
