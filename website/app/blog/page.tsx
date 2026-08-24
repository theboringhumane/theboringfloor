import Link from 'next/link'
import type { Metadata } from 'next'
import { SiteHeader } from '@/components/site-header'
import { SiteFooter } from '@/components/site-footer'
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
    title: 'Blog · theboringoffice',
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
    <>
      <SiteHeader />
      <main>
        <div className="mx-auto max-w-7xl px-6 py-16">
          <h1 className="text-5xl font-bold tracking-tight md:text-6xl">Blog</h1>
        </div>

        <div className="border-t border-border">
          <div className="mx-auto max-w-7xl px-6 py-12">
            <h2 className="mb-6 font-mono text-xs uppercase tracking-wider text-muted-foreground">
              Featured
            </h2>
            <div className="grid grid-cols-1 gap-px overflow-hidden border border-border bg-border md:grid-cols-3">
              {featured.map((post) => (
                <Link
                  key={post.slug}
                  href={`/blog/${post.slug}`}
                  className="flex flex-col gap-3 bg-background p-6 transition-colors hover:bg-secondary/40"
                >
                  <span className="font-mono text-xs text-muted-foreground">
                    {formatDate(post.date)}
                  </span>
                  <h3 className="text-lg font-semibold leading-snug tracking-tight text-balance">
                    {post.title}
                  </h3>
                  <p className="text-sm leading-relaxed text-muted-foreground line-clamp-2">
                    {post.description}
                  </p>
                  <div className="mt-auto flex items-center justify-between pt-3">
                    <span className="text-xs text-muted-foreground">{post.author}</span>
                    <div className="flex gap-2">
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
          </div>
        </div>

        <div className="border-t border-border">
          <div className="mx-auto max-w-7xl px-6 py-12">
            <BlogFilterList posts={posts} categoryCounts={categoryCounts} />
          </div>
        </div>
      </main>
      <SiteFooter />
    </>
  )
}
