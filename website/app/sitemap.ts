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
    { path: '/blog', priority: 0.9, changeFrequency: 'daily' as const },
  ].map(({ path, priority, changeFrequency }) => ({
    url: `${SITE_URL}${path}`,
    lastModified: new Date('2026-08-24'),
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
