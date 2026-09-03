import Link from 'next/link'
import type { Metadata } from 'next'
import { getAllPosts, formatDate, categoryColors } from '@/lib/blog'
import { BlogFilterList } from '@/components/blog/blog-filter-list'
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
      <section className="border-b border-border px-6 py-14 md:px-10 md:py-16 lg:px-14">
        <p className="flex items-center gap-3 font-mono text-[11px] uppercase tracking-[0.22em] text-muted-foreground">
          <span className="h-px w-6 bg-border" aria-hidden />
          Notes from the floor
        </p>
        <h1 className="mt-6 max-w-[12ch] font-sans text-[clamp(2.75rem,8vw,5.5rem)] font-medium leading-[0.92] tracking-[-0.04em]">
          Blog
        </h1>
        <p className="mt-6 max-w-xl text-pretty text-base leading-relaxed text-muted-foreground">
          Product updates and engineering notes on coding agents, opencode subagents, and running a
          virtual office in the terminal.
        </p>
      </section>

      <section className="border-b border-border">
        <div className="flex items-center justify-between gap-4 border-b border-border px-6 py-3 font-mono text-[10px] uppercase tracking-[0.16em] text-muted-foreground md:px-10 lg:px-14">
          <span>Featured</span>
          <span className="hidden sm:inline">{featured.length} posts</span>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3">
          {featured.map((post, i) => (
            <Link
              key={post.slug}
              href={`/blog/${post.slug}`}
              className={[
                'flex flex-col gap-3 p-6 transition-colors hover:bg-secondary/40 md:p-8',
                i < featured.length - 1 ? 'border-b border-border md:border-b-0 md:border-r' : '',
              ]
                .filter(Boolean)
                .join(' ')}
            >
              <span className="font-mono text-xs text-muted-foreground">{formatDate(post.date)}</span>
              <h2 className="text-lg font-semibold leading-snug tracking-tight text-balance">
                {post.title}
              </h2>
              <p className="text-sm leading-relaxed text-muted-foreground line-clamp-2">
                {post.description}
              </p>
              <div className="mt-auto flex flex-wrap items-center justify-between gap-2 pt-4">
                <span className="text-xs text-muted-foreground">{post.author}</span>
                <div className="flex flex-wrap gap-2">
                  {post.categories.map((c) => (
                    <span
                      key={c}
                      className={`border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${categoryColors[c]}`}
                    >
                      {c}
                    </span>
                  ))}
                </div>
              </div>
            </Link>
          ))}
        </div>
      </section>

      <section className="border-b border-border px-6 py-12 md:px-10 lg:px-14">
        <BlogFilterList posts={posts} categoryCounts={categoryCounts} />
      </section>
    </main>
  )
}
