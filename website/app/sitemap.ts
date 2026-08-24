import type { MetadataRoute } from 'next'
import { getAllPosts } from '@/lib/blog'

const BASE_URL = 'https://theboringoffice.pages.dev'

export const dynamic = 'force-static'

export default function sitemap(): MetadataRoute.Sitemap {
  const staticRoutes: MetadataRoute.Sitemap = [
    '',
    '/vision',
    '/blog',
    '/docs',
    '/get-started',
  ].map((route) => ({
    url: `${BASE_URL}${route}`,
    lastModified: new Date('2026-08-24'),
  }))

  const blogRoutes: MetadataRoute.Sitemap = getAllPosts().map((post) => ({
    url: `${BASE_URL}/blog/${post.slug}`,
    lastModified: new Date(post.date),
  }))

  return [...staticRoutes, ...blogRoutes]
}
