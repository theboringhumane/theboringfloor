import type { MetadataRoute } from 'next'
import { SITE_URL } from '@/lib/site'

export const dynamic = 'force-static'

/** Explicit allow-list so AI crawlers know they are welcome (not blocked). */
const AI_CRAWLERS = [
  'GPTBot',
  'ChatGPT-User',
  'OAI-SearchBot',
  'ClaudeBot',
  'anthropic-ai',
  'Google-Extended',
  'Googlebot',
  'Bingbot',
  'PerplexityBot',
  'Applebot',
  'Applebot-Extended',
  'Bytespider',
  'CCBot',
  'meta-externalagent',
  'FacebookBot',
]

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: '/',
      },
      ...AI_CRAWLERS.map((userAgent) => ({
        userAgent,
        allow: '/' as const,
      })),
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
    host: SITE_URL,
  }
}
