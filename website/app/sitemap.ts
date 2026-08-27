import type { MetadataRoute } from 'next'
import { getAllPosts } from '@/lib/blog'
import { SITE_URL } from '@/lib/site'

export const dynamic = 'force-static'

export default function sitemap(): MetadataRoute.Sitemap {
  const staticRoutes: MetadataRoute.Sitemap = [
    { path: '', priority: 1, changeFrequency: 'weekly' as const },
    { path: '/get-started', priority: 0.9, changeFrequency: 'weekly' as const },
    { path: '/vision', priority: 0.8, changeFrequency: 'monthly' as const },
    { path: '/docs', priority: 0.8, changeFrequency: 'weekly' as const },
    { path: '/docs/getting-started', priority: 0.8, changeFrequency: 'weekly' as const },
    { path: '/docs/backends', priority: 0.8, changeFrequency: 'weekly' as const },
    { path: '/docs/chat-and-threads', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/permissions-and-questions', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/plan-mode', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/queue-board-memory', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/terminal-and-git-tabs', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/browser-tab', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/layout-themes-power', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/docs/keys-and-slash', priority: 0.7, changeFrequency: 'weekly' as const },
    { path: '/blog', priority: 0.9, changeFrequency: 'daily' as const },
    {
      path: '/sounds',
      priority: 0.7,
      changeFrequency: 'monthly' as const,
      lastModified: new Date('2026-08-26'),
    },
  ].map(({ path, priority, changeFrequency, lastModified }) => ({
    url: `${SITE_URL}${path}`,
    lastModified: lastModified ?? new Date('2026-08-24'),
    changeFrequency,
    priority,
  }))

  const blogRoutes: MetadataRoute.Sitemap = getAllPosts().map((post) => ({
    url: `${SITE_URL}/blog/${post.slug}`,
    lastModified: new Date(post.date),
    changeFrequency: 'monthly' as const,
    priority: post.featured ? 0.8 : 0.7,
  }))

  return [...staticRoutes, ...blogRoutes]
}
