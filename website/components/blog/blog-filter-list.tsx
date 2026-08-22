'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { cn } from '@/lib/utils'
import { formatDate, type BlogCategory, type BlogPostMeta } from '@/lib/blog-types'

const dotColors: Record<BlogCategory, string> = {
  Release: 'bg-chart-2',
  Office: 'bg-chart-3',
  Engineering: 'bg-chart-5',
  'AI Agents': 'bg-chart-4',
  Updates: 'bg-chart-1',
}

export function BlogFilterList({
  posts,
  categoryCounts,
}: {
  posts: BlogPostMeta[]
  categoryCounts: Record<string, number>
}) {
  const [filter, setFilter] = useState<string>('All')

  const filtered = useMemo(() => {
    if (filter === 'All') return posts
    return posts.filter((p) => p.categories.includes(filter as BlogCategory))
  }, [filter, posts])

  const categories = Object.keys(categoryCounts).sort()

  return (
    <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_240px]">
      <div className="flex flex-col divide-y divide-border border-t border-border">
        {filtered.map((post) => (
          <Link
            key={post.slug}
            href={`/blog/${post.slug}`}
            className="flex flex-col gap-1 py-5 transition-colors hover:bg-secondary/30 sm:flex-row sm:items-baseline sm:justify-between sm:gap-6"
          >
            <div className="flex items-start gap-2">
              <span
                className={cn(
                  'mt-2 size-1.5 flex-shrink-0 rounded-full',
                  dotColors[post.categories[0]] ?? 'bg-muted-foreground',
                )}
              />
              <div>
                <h3 className="text-base font-medium leading-snug">{post.title}</h3>
                {post.description && (
                  <p className="mt-1 max-w-xl text-sm leading-relaxed text-muted-foreground">
                    {post.description}
                  </p>
                )}
                <div className="mt-2 flex gap-2">
                  {post.categories.map((c) => (
                    <span
                      key={c}
                      className="border border-border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider text-muted-foreground"
                    >
                      {c}
                    </span>
                  ))}
                </div>
              </div>
            </div>
            <div className="flex flex-shrink-0 flex-col items-end gap-1 pl-5 sm:pl-0">
              <span className="font-mono text-xs text-muted-foreground">
                {formatDate(post.date)}
              </span>
              <span className="text-xs text-muted-foreground">{post.author}</span>
            </div>
          </Link>
        ))}
      </div>

      <div className="flex flex-col gap-1 font-mono text-xs uppercase tracking-wider">
        <button
          onClick={() => setFilter('All')}
          className={cn(
            'flex items-center justify-between px-3 py-2 text-left transition-colors',
            filter === 'All'
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground hover:bg-secondary',
          )}
        >
          <span className="flex items-center gap-2">
            <span className="size-1.5 rounded-full bg-current" />
            All
          </span>
          <span>{posts.length}</span>
        </button>
        {categories.map((c) => (
          <button
            key={c}
            onClick={() => setFilter(c)}
            className={cn(
              'flex items-center justify-between px-3 py-2 text-left transition-colors',
              filter === c
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:bg-secondary',
            )}
          >
            <span className="flex items-center gap-2">
              <span className={cn('size-1.5 rounded-full', dotColors[c as BlogCategory])} />
              {c}
            </span>
            <span>{categoryCounts[c]}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
