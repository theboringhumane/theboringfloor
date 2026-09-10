'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { cn } from '@/lib/utils'
import { formatDate, type BlogCategory, type BlogPostMeta } from '@/lib/blog-types'

/* Selected state is carried by TWO signals, never colour alone: the pill
   inverts (ink stock, paper type) and the leading register mark flips from an
   open box to a filled one. Both survive a monochrome print and both ramps. */
function FilterPill({
  label,
  count,
  selected,
  onSelect,
}: {
  label: string
  count: number
  selected: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onSelect}
      className={cn(
        'flex w-full items-center justify-between gap-3 border px-2.5 py-1.5 text-left font-mono text-[0.6875rem] uppercase leading-none tracking-[0.14em] transition-colors',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue',
        selected
          ? 'border-ink bg-ink text-paper'
          : 'border-rule text-ink-soft hover:border-ink hover:text-ink',
      )}
    >
      <span className="flex items-center gap-2">
        <span aria-hidden="true">{selected ? '■' : '□'}</span>
        {label}
      </span>
      <span className={selected ? 'text-paper' : 'text-ink-faint'}>{count}</span>
    </button>
  )
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
    <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_15rem] lg:gap-14">
      <div className="flex flex-col divide-y divide-rule border-t border-rule">
        {filtered.map((post) => (
          <article key={post.slug}>
            <Link
              href={`/blog/${post.slug}`}
              className="group grid grid-cols-1 gap-3 py-6 transition-colors hover:bg-paper-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue sm:grid-cols-[9rem_1fr] sm:gap-6"
            >
              <div className="flex flex-row items-baseline gap-3 sm:flex-col sm:gap-1.5">
                <time dateTime={post.date} className="mono-label whitespace-nowrap text-ink">
                  {formatDate(post.date)}
                </time>
                <span className="mono-label text-ink-faint">{post.author}</span>
              </div>

              <div>
                <h2 className="display-md text-ink group-hover:underline group-hover:decoration-rule group-hover:underline-offset-4">
                  {post.title}
                </h2>
                {post.description && (
                  <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-soft">
                    {post.description}
                  </p>
                )}
                {post.categories.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {post.categories.map((c) => (
                      <span
                        key={c}
                        className="border border-rule px-2 py-1 font-mono text-[0.6875rem] uppercase leading-none tracking-[0.14em] text-ink-faint"
                      >
                        {c}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </Link>
          </article>
        ))}
      </div>

      <div>
        <div className="flex items-center gap-3 border-b border-rule pb-2">
          <span className="mono-label text-ink">Filed under</span>
          <span className="h-px flex-1 bg-rule" aria-hidden="true" />
        </div>
        <div className="mt-4 flex flex-col gap-1.5">
          <FilterPill
            label="All"
            count={posts.length}
            selected={filter === 'All'}
            onSelect={() => setFilter('All')}
          />
          {categories.map((c) => (
            <FilterPill
              key={c}
              label={c}
              count={categoryCounts[c]}
              selected={filter === c}
              onSelect={() => setFilter(c)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
